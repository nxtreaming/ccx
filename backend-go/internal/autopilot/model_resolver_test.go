package autopilot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

// ── 测试辅助 ──

// makeModelProfile 创建测试用 ModelProfile，仅填充 ResolveModel 需要的字段。
func makeModelProfile(modelID string, family ModelFamily, tier QualityTier, ctxTokens int,
	reasoning, vision, toolCalls bool, probeOK bool, latencyMs int64) ModelProfile {
	return ModelProfile{
		ChannelUID:        "ch_test",
		ChannelKind:       "messages",
		MetricsKey:        "metrics_test",
		ModelID:           modelID,
		ModelFamily:       family,
		QualityTier:       tier,
		ContextTokens:     ctxTokens,
		SupportsReasoning: reasoning,
		SupportsVision:    vision,
		SupportsToolCalls: toolCalls,
		ProbeSuccess:      probeOK,
		ProbeLatencyMs:    latencyMs,
	}
}

// newTestResolver 创建带预填充画像的 ModelResolver（无 cfgManager，跳过手动映射检查）。
func newTestResolver(t *testing.T, profiles []ModelProfile) *ModelResolver {
	t.Helper()
	// 直接构造 ModelProfileStore，仅使用内存缓存（测试不需要 SQLite）。
	store := &ModelProfileStore{
		cache:     make(map[string]*ModelProfile),
		dirtyKeys: make(map[string]struct{}),
	}
	for i := range profiles {
		p := profiles[i]
		_ = store.Upsert(&p)
	}
	return NewModelResolver(store, nil)
}

// createTestConfigManagerForResolver 创建测试用 ConfigManager。
func createTestConfigManagerForResolver(t *testing.T, cfg config.Config) (*config.ConfigManager, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "model-resolver-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	configFile := filepath.Join(tmpDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("写入配置文件失败: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configFile, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("创建配置管理器失败: %v", err)
	}
	cleanup := func() {
		cfgManager.Close()
		os.RemoveAll(tmpDir)
	}
	return cfgManager, cleanup
}

// ── CapabilityFloor 测试 ──

func TestBuildCapabilityFloorFromRequestProfile(t *testing.T) {
	profile := &RequestProfile{
		ContextNeed:   128000,
		ReasoningNeed: true,
		VisionNeed:    true,
		ToolUseNeed:   true,
		QualityNeed:   QualityTierHigh,
	}
	floor := BuildCapabilityFloorFromRequestProfile(profile)

	if floor.MinContextTokens != 128000 {
		t.Errorf("MinContextTokens = %d, want 128000", floor.MinContextTokens)
	}
	if !floor.NeedsReasoning {
		t.Error("NeedsReasoning should be true")
	}
	if !floor.NeedsVision {
		t.Error("NeedsVision should be true")
	}
	if !floor.NeedsToolCalls {
		t.Error("NeedsToolCalls should be true")
	}
	if floor.MinQualityTier != QualityTierHigh {
		t.Errorf("MinQualityTier = %s, want high", floor.MinQualityTier)
	}

	// 空 profile 应生成零值 floor
	empty := BuildCapabilityFloorFromRequestProfile(&RequestProfile{})
	if empty.MinContextTokens != 0 || empty.NeedsReasoning || empty.NeedsVision ||
		empty.NeedsToolCalls || empty.MinQualityTier != "" {
		t.Errorf("empty profile should produce zero-value floor, got %+v", empty)
	}
}

func TestQualityTargetFromRequestProfileUsesTaskClass(t *testing.T) {
	tests := []struct {
		name       string
		taskClass  TaskClass
		quality    QualityTier
		context    int
		tool       bool
		reasoning  bool
		complexity TaskComplexity
		wantTarget QualityTier
	}{
		{name: "lightweight opus 降到 low", taskClass: TaskClassLightweight, quality: QualityTierPremium, wantTarget: QualityTierLow},
		{name: "worker opus 使用 normal", taskClass: TaskClassWorker, quality: QualityTierPremium, wantTarget: QualityTierNormal},
		{name: "worker 工具请求至少 normal", taskClass: TaskClassWorker, quality: QualityTierPremium, tool: true, wantTarget: QualityTierNormal},
		{name: "supervisor 保持 high", taskClass: TaskClassSupervisor, quality: QualityTierPremium, wantTarget: QualityTierHigh},
		{name: "复杂 supervisor 保持 premium", taskClass: TaskClassSupervisor, quality: QualityTierPremium, complexity: TaskComplexityComplex, wantTarget: QualityTierPremium},
		{name: "复杂 worker 提升到 high", taskClass: TaskClassWorker, quality: QualityTierPremium, complexity: TaskComplexityComplex, wantTarget: QualityTierHigh},
		{name: "常规 supervisor 使用 normal", taskClass: TaskClassSupervisor, quality: QualityTierPremium, complexity: TaskComplexityRoutine, wantTarget: QualityTierNormal},
		{name: "长上下文至少 high", taskClass: TaskClassWorker, quality: QualityTierPremium, context: 50_000, wantTarget: QualityTierHigh},
		{name: "低档请求不被升级", taskClass: TaskClassSupervisor, quality: QualityTierNormal, wantTarget: QualityTierNormal},
		{name: "未知分类保持原档位", taskClass: TaskClass("unknown"), quality: QualityTierPremium, wantTarget: QualityTierPremium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &RequestProfile{
				TaskClass:     tt.taskClass,
				QualityNeed:   tt.quality,
				ContextNeed:   tt.context,
				ToolUseNeed:   tt.tool,
				ReasoningNeed: tt.reasoning,
				Complexity:    tt.complexity,
			}
			if got := ResolveQualityTarget(profile); got != tt.wantTarget {
				t.Fatalf("ResolveQualityTarget() = %q, want %q", got, tt.wantTarget)
			}
			floor := BuildCapabilityFloorFromRequestProfile(profile)
			if floor.MinQualityTier != tt.wantTarget {
				t.Fatalf("floor.MinQualityTier = %q, want %q", floor.MinQualityTier, tt.wantTarget)
			}
		})
	}
}

// ── filterByCapabilityFloor 测试 ──

func TestFilterByCapabilityFloor_DropsUnderQualified(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("model-a", ModelFamilyClaude, QualityTierPremium, 200000,
			true, true, true, true, 100), // 全满足
		makeModelProfile("model-b", ModelFamilyClaude, QualityTierNormal, 200000,
			true, true, true, true, 100), // quality 不满足 premium
		makeModelProfile("model-c", ModelFamilyClaude, QualityTierPremium, 50000,
			true, true, true, true, 100), // context 不足
		makeModelProfile("model-d", ModelFamilyClaude, QualityTierPremium, 200000,
			false, true, true, true, 100), // 无 reasoning
		makeModelProfile("model-e", ModelFamilyClaude, QualityTierPremium, 200000,
			true, false, true, true, 100), // 无 vision
		makeModelProfile("model-f", ModelFamilyClaude, QualityTierPremium, 200000,
			true, true, false, true, 100), // 无 tool calls
		makeModelProfile("model-g", ModelFamilyClaude, QualityTierPremium, 200000,
			true, true, true, false, 100), // ProbeSuccess=false
	}

	floor := CapabilityFloor{
		MinContextTokens: 100000,
		NeedsReasoning:   true,
		NeedsVision:      true,
		NeedsToolCalls:   true,
		MinQualityTier:   QualityTierPremium,
	}

	eligible := filterByCapabilityFloor(profiles, floor)

	if len(eligible) != 1 {
		t.Fatalf("expected 1 eligible, got %d", len(eligible))
	}
	if eligible[0].ModelID != "model-a" {
		t.Errorf("expected model-a, got %s", eligible[0].ModelID)
	}
}

func TestFilterByCapabilityFloor_ZeroFloorPassesAllProbed(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("m1", ModelFamilyUnknown, QualityTierLow, 0,
			false, false, false, true, 0),
		makeModelProfile("m2", ModelFamilyUnknown, QualityTierLow, 0,
			false, false, false, false, 0), // 未探测
	}
	eligible := filterByCapabilityFloor(profiles, CapabilityFloor{})
	if len(eligible) != 1 {
		t.Fatalf("expected 1 (only probed), got %d", len(eligible))
	}
}

// ── rankBySimilarity 测试 ──

func TestRankBySimilarity_PrefersSameFamily(t *testing.T) {
	floor := CapabilityFloor{MinQualityTier: QualityTierHigh}
	eligible := []ModelProfile{
		makeModelProfile("gpt-5.3", ModelFamilyOpenAI, QualityTierHigh, 200000,
			true, false, true, true, 50),
		makeModelProfile("claude-sonnet-4-6", ModelFamilyClaude, QualityTierHigh, 200000,
			true, false, true, true, 50),
	}

	best := rankBySimilarity(eligible, "claude-sonnet-5", floor)
	if best.ModelID != "claude-sonnet-4-6" {
		t.Errorf("expected claude-sonnet-4-6 (same family), got %s", best.ModelID)
	}
}

func TestRankBySimilarity_PrefersSameQualityTier(t *testing.T) {
	floor := CapabilityFloor{MinQualityTier: QualityTierPremium}
	eligible := []ModelProfile{
		makeModelProfile("gpt-5.3", ModelFamilyOpenAI, QualityTierHigh, 200000,
			true, false, true, true, 50),
		makeModelProfile("gpt-5.4", ModelFamilyOpenAI, QualityTierPremium, 200000,
			true, false, true, true, 50),
	}

	best := rankBySimilarity(eligible, "gpt-5.5", floor)
	if best.ModelID != "gpt-5.4" {
		t.Errorf("expected gpt-5.4 (same quality tier), got %s", best.ModelID)
	}
}

func TestRankBySimilarity_PrefersClosestQualityTarget(t *testing.T) {
	floor := CapabilityFloor{MinQualityTier: QualityTierHigh}
	eligible := []ModelProfile{
		makeModelProfile("k3", ModelFamilyKimi, QualityTierPremium, 200000,
			true, false, true, true, 1),
		makeModelProfile("kimi-for-coding", ModelFamilyKimi, QualityTierHigh, 200000,
			true, false, true, true, 100),
	}

	best := rankBySimilarity(eligible, "claude-opus-4-8", floor)
	if best.ModelID != "kimi-for-coding" {
		t.Errorf("expected closest high-tier model kimi-for-coding, got %s", best.ModelID)
	}
}

func TestResolveModel_UsesTaskQualityTarget(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("k3", ModelFamilyKimi, QualityTierPremium, 1_048_576,
			true, true, true, true, 10),
		makeModelProfile("kimi-for-coding", ModelFamilyKimi, QualityTierHigh, 262_144,
			true, true, true, true, 100),
		makeModelProfile("kimi-v2", ModelFamilyKimi, QualityTierNormal, 128_000,
			false, false, true, true, 50),
	}
	resolver := newTestResolver(t, profiles)

	tests := []struct {
		name      string
		profile   RequestProfile
		wantModel string
	}{
		{
			name: "lightweight 选择最接近的标准模型",
			profile: RequestProfile{
				Model: "claude-opus-4-8", ChannelKind: "messages", QualityNeed: QualityTierPremium,
				TaskClass: TaskClassLightweight, ContextNeed: 1000,
			},
			wantModel: "kimi-v2",
		},
		{
			name: "supervisor 选择 high 而不是 premium K3",
			profile: RequestProfile{
				Model: "claude-opus-4-8", ChannelKind: "messages", QualityNeed: QualityTierPremium,
				TaskClass: TaskClassSupervisor, ContextNeed: 1000,
			},
			wantModel: "kimi-for-coding",
		},
		{
			name: "大上下文硬约束保留 K3",
			profile: RequestProfile{
				Model: "claude-opus-4-8", ChannelKind: "messages", QualityNeed: QualityTierPremium,
				TaskClass: TaskClassSupervisor, ContextNeed: 500_000,
			},
			wantModel: "k3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			floor := BuildCapabilityFloorFromRequestProfile(&tt.profile)
			mapped, resolved, reason := resolver.ResolveModel(
				tt.profile.Model, "ch_test", "messages", "metrics_test", floor)
			if !resolved || mapped != tt.wantModel {
				t.Fatalf("ResolveModel() = (%q, %v, %q), want %q", mapped, resolved, reason, tt.wantModel)
			}
		})
	}
}

func TestRankBySimilarity_PrefersCloserContext(t *testing.T) {
	floor := CapabilityFloor{MinContextTokens: 100000}
	eligible := []ModelProfile{
		makeModelProfile("model-far", ModelFamilyClaude, QualityTierNormal, 1000000,
			false, false, false, true, 100),
		makeModelProfile("model-close", ModelFamilyClaude, QualityTierNormal, 110000,
			false, false, false, true, 100),
	}

	best := rankBySimilarity(eligible, "claude-haiku-4-5", floor)
	if best.ModelID != "model-close" {
		t.Errorf("expected model-close (closer context), got %s", best.ModelID)
	}
}

func TestRankBySimilarity_PrefersLowerLatency(t *testing.T) {
	floor := CapabilityFloor{}
	eligible := []ModelProfile{
		makeModelProfile("fast", ModelFamilyClaude, QualityTierNormal, 100000,
			false, false, false, true, 50),
		makeModelProfile("slow", ModelFamilyClaude, QualityTierNormal, 100000,
			false, false, false, true, 500),
	}

	best := rankBySimilarity(eligible, "claude-haiku-4-5", floor)
	if best.ModelID != "fast" {
		t.Errorf("expected fast (lower latency tie-break), got %s", best.ModelID)
	}
}

// ── ResolveModel 端到端测试 ──

func TestResolveModel_NoProfiles_ReturnsFalse(t *testing.T) {
	resolver := newTestResolver(t, nil)
	mapped, resolved, reason := resolver.ResolveModel(
		"claude-opus-4-8", "ch_empty", "messages", "mkey", CapabilityFloor{})
	if resolved {
		t.Error("expected resolved=false when no profiles")
	}
	if mapped != "claude-opus-4-8" {
		t.Errorf("expected passthrough model, got %s", mapped)
	}
	if reason != "no_model_profiles" {
		t.Errorf("expected reason 'no_model_profiles', got %s", reason)
	}
}

func TestResolveModel_AllFilteredOut_ReturnsFalse(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("tiny-model", ModelFamilyUnknown, QualityTierLow, 1000,
			false, false, false, true, 100),
	}
	resolver := newTestResolver(t, profiles)

	floor := CapabilityFloor{
		MinContextTokens: 100000,
		MinQualityTier:   QualityTierHigh,
	}
	mapped, resolved, reason := resolver.ResolveModel(
		"claude-opus-4-8", "ch_test", "messages", "metrics_test", floor)
	if resolved {
		t.Error("expected resolved=false when all filtered")
	}
	if reason != "no_capable_model" {
		t.Errorf("expected reason 'no_capable_model', got %s", reason)
	}
	if mapped != "claude-opus-4-8" {
		t.Errorf("expected passthrough model, got %s", mapped)
	}
}

func TestResolveModel_FindsBestMatch(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("claude-sonnet-4-6", ModelFamilyClaude, QualityTierHigh, 200000,
			true, false, true, true, 80),
		makeModelProfile("gpt-5.3", ModelFamilyOpenAI, QualityTierHigh, 200000,
			true, false, true, true, 60),
	}
	resolver := newTestResolver(t, profiles)

	floor := CapabilityFloor{MinQualityTier: QualityTierHigh}
	mapped, resolved, reason := resolver.ResolveModel(
		"claude-sonnet-5", "ch_test", "messages", "metrics_test", floor)
	if !resolved {
		t.Error("expected resolved=true")
	}
	if mapped != "claude-sonnet-4-6" {
		t.Errorf("expected claude-sonnet-4-6, got %s", mapped)
	}
	if reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestResolveModel_GPT56RequiresPremiumReplacement(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("glm-4.5", ModelFamilyGLM, QualityTierNormal, 128000,
			false, false, true, true, 20),
		makeModelProfile("glm-5.1", ModelFamilyGLM, QualityTierHigh, 202800,
			true, false, true, true, 30),
		makeModelProfile("glm-5.2", ModelFamilyGLM, QualityTierPremium, 1048576,
			true, false, true, true, 40),
	}
	for i := range profiles {
		profiles[i].ChannelKind = "responses"
	}
	resolver := newTestResolver(t, profiles)
	floor := CapabilityFloor{MinQualityTier: ModelProfileQualityTierFromFamily(ModelFamilyOpenAI, "gpt-5.6-sol")}

	mapped, resolved, reason := resolver.ResolveModel(
		"gpt-5.6-sol", "ch_test", "responses", "metrics_test", floor)
	if !resolved || mapped != "glm-5.2" {
		t.Fatalf("ResolveModel() = (%q, %v, %q), want premium glm-5.2", mapped, resolved, reason)
	}
}

func TestResolveModel_RefreshesLegacyAutoDiscoveryCapabilities(t *testing.T) {
	profile := makeModelProfile("glm-5.2", ModelFamilyGLM, QualityTierHigh, 272000,
		false, false, false, true, 0)
	profile.ChannelKind = "responses"
	profile.Source = "auto_discovery"
	store := &ModelProfileStore{
		cache:     make(map[string]*ModelProfile),
		dirtyKeys: make(map[string]struct{}),
	}
	if err := store.Upsert(&profile); err != nil {
		t.Fatal(err)
	}
	cfgManager, cleanup := createTestConfigManagerForResolver(t, config.Config{
		ResponsesUpstream: []config.UpstreamConfig{{
			ChannelUID: "ch_test", AutoManaged: true, ServiceType: "openai",
		}},
	})
	defer cleanup()
	resolver := NewModelResolver(store, cfgManager)

	mapped, resolved, reason := resolver.ResolveModel(
		"gpt-5.6-sol", "ch_test", "responses", "metrics_test",
		CapabilityFloor{MinQualityTier: QualityTierPremium, NeedsReasoning: true, NeedsToolCalls: true})
	if !resolved || mapped != "glm-5.2" {
		t.Fatalf("ResolveModel() = (%q, %v, %q), want refreshed glm-5.2 capabilities", mapped, resolved, reason)
	}
	refreshed := store.Get("ch_test", "responses", "metrics_test", "glm-5.2")
	if refreshed == nil || refreshed.QualityTier != QualityTierPremium ||
		refreshed.ContextTokens != 1048576 || !refreshed.SupportsReasoning || !refreshed.SupportsToolCalls {
		t.Fatalf("旧自动发现画像未在内存中完成升级: %+v", refreshed)
	}
}

func TestResolveModel_RefreshesKimiK3VisionCapabilities(t *testing.T) {
	profile := makeModelProfile("k3", ModelFamilyKimi, QualityTierPremium, 262144,
		true, false, true, true, 0)
	profile.Source = "auto_discovery"
	store := &ModelProfileStore{
		cache:     make(map[string]*ModelProfile),
		dirtyKeys: make(map[string]struct{}),
	}
	if err := store.Upsert(&profile); err != nil {
		t.Fatal(err)
	}
	cfgManager, cleanup := createTestConfigManagerForResolver(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			ChannelUID: "ch_test", AutoManaged: true, ProviderID: "kimi", ServiceType: "claude",
		}},
	})
	defer cleanup()
	resolver := NewModelResolver(store, cfgManager)

	mapped, resolved, reason := resolver.ResolveModel(
		"claude-opus-4-8", "ch_test", "messages", "metrics_test",
		CapabilityFloor{
			MinContextTokens: 200000,
			MinQualityTier:   QualityTierPremium,
			NeedsReasoning:   true,
			NeedsVision:      true,
			NeedsToolCalls:   true,
		})
	if !resolved || mapped != "k3" {
		t.Fatalf("ResolveModel() = (%q, %v, %q), want vision-capable k3", mapped, resolved, reason)
	}
	refreshed := store.Get("ch_test", "messages", "metrics_test", "k3")
	if refreshed == nil || !refreshed.SupportsVision || !refreshed.SupportsToolCalls ||
		!refreshed.SupportsReasoning || refreshed.QualityTier != QualityTierPremium {
		t.Fatalf("K3 自动发现画像未按当前注册表刷新: %+v", refreshed)
	}
}

func TestResolveModelAnyEndpoint_MapsWithoutExactModelMatch(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("mimo-v2.5-pro", ModelFamilyMiMo, QualityTierHigh, 1000000,
			true, false, true, true, 80),
		makeModelProfile("mimo-v2.5", ModelFamilyMiMo, QualityTierNormal, 1000000,
			true, true, true, true, 90),
	}
	resolver := newTestResolver(t, profiles)

	mapped, found, reason := resolver.ResolveModelAnyEndpoint("claude-sonnet-5", "ch_test", "messages")
	if !found {
		t.Fatalf("expected found=true, reason=%s", reason)
	}
	if mapped == "" || mapped == "claude-sonnet-5" {
		t.Fatalf("expected request model to be mapped to discovered model, got %q", mapped)
	}
}

func TestResolveModel_IgnoresLegacyManualRedirectForAutoManagedProvider(t *testing.T) {
	upstream := config.UpstreamConfig{
		ChannelUID:   "ch_test",
		AutoManaged:  true,
		ProviderID:   "mimo",
		ModelMapping: map[string]string{"claude-sonnet-5": "legacy-target"},
	}
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{upstream},
	}
	cfgManager, cleanup := createTestConfigManagerForResolver(t, cfg)
	defer cleanup()

	store := &ModelProfileStore{
		cache:     make(map[string]*ModelProfile),
		dirtyKeys: make(map[string]struct{}),
	}
	profile := makeModelProfile("mimo-v2.5-pro", ModelFamilyMiMo, QualityTierHigh, 1000000,
		true, false, true, true, 80)
	if err := store.Upsert(&profile); err != nil {
		t.Fatalf("写入模型画像失败: %v", err)
	}
	resolver := NewModelResolver(store, cfgManager)

	mapped, resolved, reason := resolver.ResolveModel(
		"claude-sonnet-5", "ch_test", "messages", "metrics_test", CapabilityFloor{})
	if !resolved {
		t.Fatalf("expected resolved=true, reason=%s", reason)
	}
	if mapped == "legacy-target" {
		t.Fatalf("autoManaged provider should ignore legacy modelMapping, got %q", mapped)
	}
	if mapped != "mimo-v2.5-pro" {
		t.Fatalf("mapped = %q, want mimo-v2.5-pro", mapped)
	}
}

func TestResolveModel_ManualRedirect_ShortCircuits(t *testing.T) {
	upstream := config.UpstreamConfig{
		ChannelUID:   "ch_manual",
		ModelMapping: map[string]string{"claude-opus-4-8": "claude-opus-4-7"},
	}
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{upstream},
	}
	cfgManager, cleanup := createTestConfigManagerForResolver(t, cfg)
	defer cleanup()

	resolver := &ModelResolver{
		profileStore: nil, // 无 ModelProfileStore
		cfgManager:   cfgManager,
	}

	mapped, resolved, reason := resolver.ResolveModel(
		"claude-opus-4-8", "ch_manual", "messages", "any", CapabilityFloor{})

	if !resolved {
		t.Error("expected resolved=true for manual redirect")
	}
	if mapped != "claude-opus-4-7" {
		t.Errorf("expected claude-opus-4-7, got %s", mapped)
	}
	if reason != "manual_redirect" {
		t.Errorf("expected reason 'manual_redirect', got %s", reason)
	}
}

func TestResolveModel_ManualRedirect_NotApplied_WhenNoMapping(t *testing.T) {
	upstream := config.UpstreamConfig{
		ChannelUID:   "ch_manual",
		ModelMapping: nil,
	}
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{upstream},
	}
	cfgManager, cleanup := createTestConfigManagerForResolver(t, cfg)
	defer cleanup()

	resolver := &ModelResolver{
		profileStore: nil,
		cfgManager:   cfgManager,
	}

	mapped, resolved, _ := resolver.ResolveModel(
		"claude-opus-4-8", "ch_manual", "messages", "any", CapabilityFloor{})

	if resolved {
		t.Error("expected resolved=false when no mapping and no store")
	}
	if mapped != "claude-opus-4-8" {
		t.Errorf("expected passthrough, got %s", mapped)
	}
}

func TestResolveModel_NilStore_FailOpen(t *testing.T) {
	resolver := NewModelResolver(nil, nil)
	mapped, resolved, reason := resolver.ResolveModel(
		"claude-sonnet-5", "ch_x", "messages", "mkey", CapabilityFloor{})
	if resolved {
		t.Error("expected resolved=false with nil store")
	}
	if reason != "model_profile_store_unavailable" {
		t.Errorf("expected 'model_profile_store_unavailable', got %s", reason)
	}
	if mapped != "claude-sonnet-5" {
		t.Errorf("expected passthrough, got %s", mapped)
	}
}

func TestResolveModel_ProbeSuccessFalse_Filtered(t *testing.T) {
	profiles := []ModelProfile{
		makeModelProfile("model-x", ModelFamilyClaude, QualityTierPremium, 200000,
			true, true, true, false, 100), // probeOK=false
	}
	resolver := newTestResolver(t, profiles)

	_, resolved, reason := resolver.ResolveModel(
		"claude-opus-4-8", "ch_test", "messages", "metrics_test", CapabilityFloor{})
	if resolved {
		t.Error("expected resolved=false when all candidates have ProbeSuccess=false")
	}
	if reason != "no_capable_model" {
		t.Errorf("expected 'no_capable_model', got %s", reason)
	}
}
