package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/conversation"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/ratelimit"
	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/warmup"
)

// createTestConfigManager 创建测试用配置管理器
func createTestConfigManager(t *testing.T, cfg config.Config) (*config.ConfigManager, func()) {
	t.Helper()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "scheduler-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建临时配置文件
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

	// 创建配置管理器
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

// createTestScheduler 创建测试用调度器
func createTestScheduler(t *testing.T, cfg config.Config) (*ChannelScheduler, func()) {
	t.Helper()

	cfgManager, cleanup := createTestConfigManager(t, cfg)
	messagesMetrics := metrics.NewMetricsManager()
	responsesMetrics := metrics.NewMetricsManager()
	geminiMetrics := metrics.NewMetricsManager()
	chatMetrics := metrics.NewMetricsManager()
	imagesMetrics := metrics.NewMetricsManager()
	traceAffinity := session.NewTraceAffinityManager()
	urlManager := warmup.NewURLManager(30*time.Second, 3)

	scheduler := NewChannelScheduler(cfgManager, messagesMetrics, responsesMetrics, geminiMetrics, chatMetrics, imagesMetrics, traceAffinity, urlManager)
	scheduler.SetRateLimitManager(ratelimit.NewManager())

	return scheduler, func() {
		messagesMetrics.Stop()
		responsesMetrics.Stop()
		chatMetrics.Stop()
		geminiMetrics.Stop()
		imagesMetrics.Stop()
		cleanup()
	}
}

// TestPromotedChannelBypassesHealthCheck 测试促销渠道绕过健康检查
func TestPromotedChannelBypassesHealthCheck(t *testing.T) {
	// 设置促销截止时间为 5 分钟后
	promotionUntil := time.Now().Add(5 * time.Minute)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "normal-channel",
				BaseURL:  "https://normal.example.com",
				APIKeys:  []string{"sk-normal-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "promoted-channel",
				BaseURL:        "https://promoted.example.com",
				APIKeys:        []string{"sk-promoted-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟促销渠道之前有高失败率（使其不健康）
	metricsManager := scheduler.messagesMetricsManager
	for i := 0; i < 10; i++ {
		metricsManager.RecordFailure("https://promoted.example.com", "sk-promoted-key", "claude")
	}

	// 验证促销渠道确实不健康
	isHealthy := metricsManager.IsChannelHealthyWithKeys("https://promoted.example.com", []string{"sk-promoted-key"}, "claude")
	if isHealthy {
		t.Fatal("促销渠道应该被标记为不健康")
	}

	// 选择渠道 - 促销渠道应该被选中，即使它不健康
	result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 1 {
		t.Errorf("期望选择促销渠道 (index=1)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Reason != "promotion_priority" {
		t.Errorf("期望选择原因为 promotion_priority，实际为 %s", result.Reason)
	}

	if result.Upstream.Name != "promoted-channel" {
		t.Errorf("期望选择 promoted-channel，实际选择了 %s", result.Upstream.Name)
	}

	if got := scheduler.GetCurrentChannelIndex(ChannelKindMessages); got != 1 {
		t.Errorf("运行态当前渠道 = %d, want 1", got)
	}
}

// TestPromotedChannelSkippedAfterFailure 测试促销渠道在本次请求失败后被跳过
func TestPromotedChannelSkippedAfterFailure(t *testing.T) {
	promotionUntil := time.Now().Add(5 * time.Minute)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "normal-channel",
				BaseURL:  "https://normal.example.com",
				APIKeys:  []string{"sk-normal-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "promoted-channel",
				BaseURL:        "https://promoted.example.com",
				APIKeys:        []string{"sk-promoted-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟促销渠道在本次请求中已经失败
	failedChannels := map[int]bool{
		1: true, // 促销渠道已失败
	}

	// 选择渠道 - 应该跳过促销渠道，选择正常渠道
	result, err := scheduler.SelectChannel(context.Background(), "test-user", failedChannels, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 0 {
		t.Errorf("期望选择正常渠道 (index=0)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Upstream.Name != "normal-channel" {
		t.Errorf("期望选择 normal-channel，实际选择了 %s", result.Upstream.Name)
	}

	if got := scheduler.GetCurrentChannelIndex(ChannelKindMessages); got != 0 {
		t.Errorf("运行态当前渠道 = %d, want 0", got)
	}
}

func TestPromotedChannelSkippedInRuntimeCooldown(t *testing.T) {
	promotionUntil := time.Now().Add(5 * time.Minute)
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "normal-channel",
				BaseURL:  "https://normal.example.com",
				APIKeys:  []string{"sk-normal-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "promoted-channel",
				BaseURL:        "https://promoted.example.com",
				APIKeys:        []string{"sk-promoted-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()
	scheduler.MarkChannelCooldown(ChannelKindMessages, 1, time.Minute)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}
	if result.ChannelIndex != 0 {
		t.Fatalf("期望跳过 cooldown 促销渠道并选择 index=0，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "priority_order" {
		t.Fatalf("期望选择原因为 priority_order，实际为 %s", result.Reason)
	}
}

// TestNonPromotedChannelStillChecksHealth 测试非促销渠道仍然检查健康状态
func TestNonPromotedChannelStillChecksHealth(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "unhealthy-channel",
				BaseURL:  "https://unhealthy.example.com",
				APIKeys:  []string{"sk-unhealthy-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "healthy-channel",
				BaseURL:  "https://healthy.example.com",
				APIKeys:  []string{"sk-healthy-key"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟第一个渠道不健康
	metricsManager := scheduler.messagesMetricsManager
	for i := 0; i < 10; i++ {
		metricsManager.RecordFailure("https://unhealthy.example.com", "sk-unhealthy-key", "claude")
	}

	// 选择渠道 - 应该跳过不健康的渠道，选择健康的渠道
	result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 1 {
		t.Errorf("期望选择健康渠道 (index=1)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Upstream.Name != "healthy-channel" {
		t.Errorf("期望选择 healthy-channel，实际选择了 %s", result.Upstream.Name)
	}
}

// TestExpiredPromotionNotBypassHealthCheck 测试过期的促销不绕过健康检查
func TestExpiredPromotionNotBypassHealthCheck(t *testing.T) {
	// 设置促销截止时间为过去
	promotionUntil := time.Now().Add(-5 * time.Minute)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "healthy-channel",
				BaseURL:  "https://healthy.example.com",
				APIKeys:  []string{"sk-healthy-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "expired-promoted-channel",
				BaseURL:        "https://expired.example.com",
				APIKeys:        []string{"sk-expired-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil, // 已过期
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟过期促销渠道不健康
	metricsManager := scheduler.messagesMetricsManager
	for i := 0; i < 10; i++ {
		metricsManager.RecordFailure("https://expired.example.com", "sk-expired-key", "claude")
	}

	// 选择渠道 - 过期促销渠道不应该被优先选择，应该选择健康的渠道
	result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 0 {
		t.Errorf("期望选择健康渠道 (index=0)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Upstream.Name != "healthy-channel" {
		t.Errorf("期望选择 healthy-channel，实际选择了 %s", result.Upstream.Name)
	}
}

func TestSelectChannel_DefaultRouteRejectsPrefixedOnlyChannels(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:        "kimi-only",
				BaseURL:     "https://kimi.example.com",
				APIKeys:     []string{"sk-kimi"},
				Status:      "active",
				Priority:    1,
				RoutePrefix: "kimi",
			},
			{
				Name:        "deepseek-only",
				BaseURL:     "https://deepseek.example.com",
				APIKeys:     []string{"sk-deepseek"},
				Status:      "active",
				Priority:    2,
				RoutePrefix: "deepseek",
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	_, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err == nil {
		t.Fatal("SelectChannel() error = nil, want default route rejection")
	}
}

// TestDeleteChannelMetrics_SharedMetricsKeyPreserved 测试删除渠道时共享的 metricsKey 被保留
func TestDeleteChannelMetrics_SharedMetricsKeyPreserved(t *testing.T) {
	// 场景：两个渠道共享同一个 (BaseURL, APIKey) 组合
	// 删除其中一个渠道时，共享的 metricsKey 应该被保留

	testCases := []struct {
		name string
		kind ChannelKind
	}{
		{"Messages", ChannelKindMessages},
		{"Responses", ChannelKindResponses},
		{"Gemini", ChannelKindGemini},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sharedBaseURL := "https://shared.example.com"
			sharedAPIKey := "sk-shared-key"

			// 根据渠道类型构建配置
			var cfg config.Config
			switch tc.kind {
			case ChannelKindMessages:
				cfg = config.Config{
					Upstream: []config.UpstreamConfig{
						{
							Name:     "channel-A",
							BaseURL:  sharedBaseURL,
							APIKeys:  []string{sharedAPIKey, "sk-exclusive-A"},
							Status:   "active",
							Priority: 1,
						},
						{
							Name:     "channel-B",
							BaseURL:  sharedBaseURL,
							APIKeys:  []string{sharedAPIKey},
							Status:   "active",
							Priority: 2,
						},
					},
				}
			case ChannelKindResponses:
				cfg = config.Config{
					ResponsesUpstream: []config.UpstreamConfig{
						{
							Name:     "channel-A",
							BaseURL:  sharedBaseURL,
							APIKeys:  []string{sharedAPIKey, "sk-exclusive-A"},
							Status:   "active",
							Priority: 1,
						},
						{
							Name:     "channel-B",
							BaseURL:  sharedBaseURL,
							APIKeys:  []string{sharedAPIKey},
							Status:   "active",
							Priority: 2,
						},
					},
				}
			case ChannelKindGemini:
				cfg = config.Config{
					GeminiUpstream: []config.UpstreamConfig{
						{
							Name:     "channel-A",
							BaseURL:  sharedBaseURL,
							APIKeys:  []string{sharedAPIKey, "sk-exclusive-A"},
							Status:   "active",
							Priority: 1,
						},
						{
							Name:     "channel-B",
							BaseURL:  sharedBaseURL,
							APIKeys:  []string{sharedAPIKey},
							Status:   "active",
							Priority: 2,
						},
					},
				}
			}

			scheduler, cleanup := createTestScheduler(t, cfg)
			defer cleanup()

			// 根据渠道类型获取对应的 metricsManager
			var metricsManager *metrics.MetricsManager
			switch tc.kind {
			case ChannelKindMessages:
				metricsManager = scheduler.messagesMetricsManager
			case ChannelKindResponses:
				metricsManager = scheduler.responsesMetricsManager
			case ChannelKindGemini:
				metricsManager = scheduler.geminiMetricsManager
			}

			// 为所有 key 记录一些指标
			metricsManager.RecordSuccess(sharedBaseURL, sharedAPIKey, tcServiceType(tc.kind))
			metricsManager.RecordSuccess(sharedBaseURL, "sk-exclusive-A", tcServiceType(tc.kind))

			// 验证指标存在
			sharedMetricsKey := metrics.GenerateMetricsIdentityKey(sharedBaseURL, sharedAPIKey, tcServiceType(tc.kind))
			exclusiveMetricsKey := metrics.GenerateMetricsIdentityKey(sharedBaseURL, "sk-exclusive-A", tcServiceType(tc.kind))

			if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), sharedMetricsKey) {
				t.Fatal("共享 metricsKey 应该存在")
			}
			if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), exclusiveMetricsKey) {
				t.Fatal("独占 metricsKey 应该存在")
			}

			// 从配置中移除 channel-A
			var channelAConfig config.UpstreamConfig
			var err error
			switch tc.kind {
			case ChannelKindMessages:
				channelAConfig = cfg.Upstream[0]
				_, err = scheduler.configManager.RemoveUpstream(0)
			case ChannelKindResponses:
				channelAConfig = cfg.ResponsesUpstream[0]
				_, err = scheduler.configManager.RemoveResponsesUpstream(0)
			case ChannelKindGemini:
				channelAConfig = cfg.GeminiUpstream[0]
				_, err = scheduler.configManager.RemoveGeminiUpstream(0)
			}
			if err != nil {
				t.Fatalf("移除渠道失败: %v", err)
			}

			// 调用 DeleteChannelMetrics
			scheduler.DeleteChannelMetrics(&channelAConfig, tc.kind)

			// 验证结果
			// 共享的 metricsKey 应该被保留（因为 channel-B 还在使用）
			if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), sharedMetricsKey) {
				t.Error("共享 metricsKey 应该被保留，但被删除了")
			}

			// 独占的 metricsKey 应该被删除
			if hasMetricsKey(metricsManager.GetAllKeyMetrics(), exclusiveMetricsKey) {
				t.Error("独占 metricsKey 应该被删除，但仍然存在")
			}
		})
	}
}

// TestDeleteChannelMetrics_AllExclusiveKeysDeleted 测试删除渠道时所有独占的 metricsKey 都被删除
func TestDeleteChannelMetrics_AllExclusiveKeysDeleted(t *testing.T) {
	// 场景：渠道有多个独占的 (BaseURL, APIKey) 组合
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "channel-to-delete",
				BaseURL:  "https://exclusive.example.com",
				APIKeys:  []string{"sk-key-1", "sk-key-2"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "other-channel",
				BaseURL:  "https://other.example.com",
				APIKeys:  []string{"sk-other-key"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	metricsManager := scheduler.messagesMetricsManager

	// 为所有 key 记录指标
	metricsManager.RecordSuccess("https://exclusive.example.com", "sk-key-1", "claude")
	metricsManager.RecordSuccess("https://exclusive.example.com", "sk-key-2", "claude")
	metricsManager.RecordSuccess("https://other.example.com", "sk-other-key", "claude")

	// 从配置中移除要删除的渠道
	channelToDelete := cfg.Upstream[0]
	_, err := scheduler.configManager.RemoveUpstream(0)
	if err != nil {
		t.Fatalf("移除渠道失败: %v", err)
	}

	// 调用 DeleteChannelMetrics
	scheduler.DeleteChannelMetrics(&channelToDelete, ChannelKindMessages)

	// 验证结果
	key1 := metrics.GenerateMetricsIdentityKey("https://exclusive.example.com", "sk-key-1", "claude")
	key2 := metrics.GenerateMetricsIdentityKey("https://exclusive.example.com", "sk-key-2", "claude")
	otherKey := metrics.GenerateMetricsIdentityKey("https://other.example.com", "sk-other-key", "claude")

	// 被删除渠道的所有 metricsKey 都应该被删除
	if hasMetricsKey(metricsManager.GetAllKeyMetrics(), key1) {
		t.Error("sk-key-1 的 metricsKey 应该被删除")
	}
	if hasMetricsKey(metricsManager.GetAllKeyMetrics(), key2) {
		t.Error("sk-key-2 的 metricsKey 应该被删除")
	}
	// 其他渠道的 metricsKey 应该保留
	if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), otherKey) {
		t.Error("其他渠道的 metricsKey 应该被保留")
	}
}

// TestDeleteChannelMetrics_SkipsWhenUpstreamStillInConfig 测试前置条件守卫：渠道仍在配置中时跳过删除
func TestDeleteChannelMetrics_SkipsWhenUpstreamStillInConfig(t *testing.T) {
	// 场景：在渠道仍在配置中时调用 DeleteChannelMetrics
	// 应该记录警告但仍然执行（可能结果不正确）
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "channel-still-in-config",
				BaseURL:  "https://example.com",
				APIKeys:  []string{"sk-key"},
				Status:   "active",
				Priority: 1,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	metricsManager := scheduler.messagesMetricsManager
	metricsManager.RecordSuccess("https://example.com", "sk-key", "claude")

	// 不从配置中移除渠道，直接调用 DeleteChannelMetrics
	// 这违反了前置条件，但方法应该仍然执行（只是结果可能不正确）
	channelConfig := cfg.Upstream[0]
	scheduler.DeleteChannelMetrics(&channelConfig, ChannelKindMessages)

	// 由于渠道仍在配置中，collectUsedCombinations 会返回该组合
	// 因此 metricsKey 不会被删除
	metricsKey := metrics.GenerateMetricsIdentityKey("https://example.com", "sk-key", "claude")

	if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), metricsKey) {
		t.Error("由于渠道仍在配置中，metricsKey 应该被保留（前置条件违反时的预期行为）")
	}
}

func TestDeleteChannelMetrics_DeletesOnlyRealServiceTypeIdentity(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:        "channel-to-delete",
				BaseURL:     "https://shared.example.com",
				APIKeys:     []string{"sk-key"},
				ServiceType: "gemini",
				Status:      "active",
				Priority:    1,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	metricsManager := scheduler.messagesMetricsManager
	metricsManager.RecordSuccess("https://shared.example.com", "sk-key", "openai")
	metricsManager.RecordSuccess("https://shared.example.com", "sk-key", "gemini")
	legacyKey := metrics.GenerateMetricsIdentityKey("https://shared.example.com", "sk-key", "openai")
	currentKey := metrics.GenerateMetricsIdentityKey("https://shared.example.com", "sk-key", "gemini")

	if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), legacyKey) {
		t.Fatal("非真实 serviceType 的 metricsKey 应该存在")
	}
	if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), currentKey) {
		t.Fatal("真实 serviceType 的 metricsKey 应该存在")
	}

	channelToDelete := cfg.Upstream[0]
	_, err := scheduler.configManager.RemoveUpstream(0)
	if err != nil {
		t.Fatalf("移除渠道失败: %v", err)
	}

	scheduler.DeleteChannelMetrics(&channelToDelete, ChannelKindMessages)

	if hasMetricsKey(metricsManager.GetAllKeyMetrics(), currentKey) {
		t.Error("真实 serviceType 的 metricsKey 应该被删除")
	}
	if !hasMetricsKey(metricsManager.GetAllKeyMetrics(), legacyKey) {
		t.Error("非真实 serviceType 的 metricsKey 不应被本次删除清理")
	}
}

func TestDeleteChannelMetrics_RemovesEquivalentLegacyMetricsKeys(t *testing.T) {
	serviceType := "claude"
	baseURLs := []string{"https://shared.example.com"}
	apiKeys := []string{"sk-key"}

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "channel-to-delete",
			BaseURL:     baseURLs[0],
			APIKeys:     apiKeys,
			ServiceType: serviceType,
			Status:      "active",
			Priority:    1,
		}},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	metricsManager := scheduler.messagesMetricsManager
	deletedKeys := metricsManager.DeleteKeysForChannel(baseURLs, apiKeys, serviceType)
	identityKey := metrics.GenerateMetricsIdentityKey(baseURLs[0], apiKeys[0], serviceType)
	legacyKey := metrics.GenerateMetricsKey(baseURLs[0], apiKeys[0])

	if !containsString(deletedKeys, identityKey) {
		t.Fatalf("deletedKeys should contain identity key %s", identityKey)
	}
	if !containsString(deletedKeys, legacyKey) {
		t.Fatalf("deletedKeys should contain equivalent legacy key %s", legacyKey)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// hasMetricsKey 辅助函数：检查 metricsKey 是否存在于指标列表中
func hasMetricsKey(allMetrics []*metrics.KeyMetrics, metricsKey string) bool {
	for _, m := range allMetrics {
		if m.MetricsKey == metricsKey {
			return true
		}
	}
	return false
}

func TestFallbackSkipsRuntimeCooldownChannel(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "unhealthy-channel",
				BaseURL:  "https://unhealthy.example.com",
				APIKeys:  []string{"sk-unhealthy"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "cooldown-channel",
				BaseURL:  "https://cooldown.example.com",
				APIKeys:  []string{"sk-cooldown"},
				Status:   "active",
				Priority: 2,
			},
			{
				Name:     "fallback-channel",
				BaseURL:  "https://fallback.example.com",
				APIKeys:  []string{"sk-fallback"},
				Status:   "active",
				Priority: 3,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	scheduler.MarkChannelCooldown(ChannelKindMessages, 1, time.Minute)

	activeChannels := []ChannelInfo{
		{Index: 1, Name: "cooldown-channel", Priority: 1, Status: "active"},
		{Index: 2, Name: "fallback-channel", Priority: 2, Status: "active"},
	}
	result, err := scheduler.selectFallbackChannel(activeChannels, map[int]bool{}, ChannelKindMessages)
	if err != nil {
		t.Fatalf("fallback 选择失败: %v", err)
	}
	if result.ChannelIndex != 2 {
		t.Fatalf("期望 fallback 跳过 cooldown 渠道并选择 index=2，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "fallback" {
		t.Fatalf("期望选择原因为 fallback，实际为 %s", result.Reason)
	}
}

func TestAffinityYieldToHigherPriorityHealthyChannel(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "high-priority-channel",
				BaseURL:  "https://high.example.com",
				APIKeys:  []string{"sk-high"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "affinity-channel",
				BaseURL:  "https://affinity.example.com",
				APIKeys:  []string{"sk-affinity"},
				Status:   "active",
				Priority: 9,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	scheduler.traceAffinity.SetPreferredChannel(string(ChannelKindMessages)+":test-user", 1)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 0 {
		t.Fatalf("期望选择更高优先级渠道 index=0，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "priority_order" {
		t.Fatalf("期望选择原因为 priority_order，实际为 %s", result.Reason)
	}
}

func TestAffinityStillWorksWithoutHigherPriorityAlternative(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "affinity-channel",
				BaseURL:  "https://affinity.example.com",
				APIKeys:  []string{"sk-affinity"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "lower-priority-channel",
				BaseURL:  "https://low.example.com",
				APIKeys:  []string{"sk-low"},
				Status:   "active",
				Priority: 9,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	scheduler.traceAffinity.SetPreferredChannel(string(ChannelKindMessages)+":test-user", 0)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 0 {
		t.Fatalf("期望继续选择亲和渠道 index=0，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "trace_affinity" {
		t.Fatalf("期望选择原因为 trace_affinity，实际为 %s", result.Reason)
	}
}

func TestAffinitySkipsRuntimeCooldownChannel(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "affinity-channel",
				BaseURL:  "https://affinity.example.com",
				APIKeys:  []string{"sk-affinity"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "fallback-channel",
				BaseURL:  "https://fallback.example.com",
				APIKeys:  []string{"sk-fallback"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	scheduler.traceAffinity.SetPreferredChannel(string(ChannelKindMessages)+":test-user", 0)
	scheduler.MarkChannelCooldown(ChannelKindMessages, 0, time.Minute)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}
	if result.ChannelIndex != 1 {
		t.Fatalf("期望跳过 cooldown 亲和渠道并选择 index=1，实际为 index=%d", result.ChannelIndex)
	}
}

func TestSelectChannelFiltersSupportedModels(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:            "image-excluded",
				BaseURL:         "https://excluded.example.com",
				APIKeys:         []string{"sk-excluded"},
				Status:          "active",
				Priority:        1,
				SupportedModels: []string{"gpt-4*", "!*image*"},
			},
			{
				Name:            "image-allowed",
				BaseURL:         "https://allowed.example.com",
				APIKeys:         []string{"sk-allowed"},
				Status:          "active",
				Priority:        2,
				SupportedModels: []string{"gpt-4*"},
			},
			{
				Name:            "invalid-pattern-fallback",
				BaseURL:         "https://invalid.example.com",
				APIKeys:         []string{"sk-invalid"},
				Status:          "active",
				Priority:        3,
				SupportedModels: []string{"foo*bar", "claude-*"},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	t.Run("命中排除规则时跳过高优先级渠道", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "gpt-4-image-preview", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("期望选择 index=1，实际为 %d", result.ChannelIndex)
		}
	})

	t.Run("模型过滤跳过渠道时记录命中排除规则", func(t *testing.T) {
		var buf bytes.Buffer
		oldOutput := log.Writer()
		log.SetOutput(&buf)
		defer log.SetOutput(oldOutput)

		result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "gpt-4-image-preview", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("期望选择 index=1，实际为 %d", result.ChannelIndex)
		}

		logOutput := buf.String()
		if !strings.Contains(logOutput, "[Scheduler-ModelFilter] 跳过渠道 [0] image-excluded") {
			t.Fatalf("期望记录模型过滤跳过日志，实际日志: %s", logOutput)
		}
		if !strings.Contains(logOutput, "模型 \"gpt-4-image-preview\" 不被 supportedModels 支持 (命中排除规则 !*image*)") {
			t.Fatalf("期望记录命中的排除规则，实际日志: %s", logOutput)
		}
	})

	t.Run("非法规则被跳过且不影响合法规则", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "claude-3-7-sonnet", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 2 {
			t.Fatalf("期望选择 index=2，实际为 %d", result.ChannelIndex)
		}
	})

	t.Run("所有活跃渠道都不支持模型时返回明确错误", func(t *testing.T) {
		_, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "gemini-2.5-pro", "", "")
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if err.Error() != "没有 Messages 渠道支持模型 \"gemini-2.5-pro\"，请检查渠道的 supportedModels 配置" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSelectChannelTraceAffinityStillRespectsSupportedModels(t *testing.T) {
	cfg := config.Config{
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:            "LocalHostClaude",
				BaseURL:         "http://127.0.0.1:3699",
				APIKeys:         []string{"sk-local"},
				Status:          "active",
				Priority:        1,
				SupportedModels: []string{"claude-*"},
				ServiceType:     "responses",
			},
			{
				Name:            "LocalHostOpenAIChat",
				BaseURL:         "http://127.0.0.1:3699",
				APIKeys:         []string{"sk-local"},
				Status:          "active",
				Priority:        2,
				SupportedModels: []string{"gpt-5.5", "gpt-5.4"},
				ServiceType:     "responses",
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	scheduler.traceAffinity.SetPreferredChannel(string(ChannelKindResponses)+":test-user", 0)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindResponses, "gpt-5.5", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 1 {
		t.Fatalf("期望跳过不支持模型的亲和渠道并选择 index=1，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "priority_order" {
		t.Fatalf("期望回退到 priority_order，实际为 %s", result.Reason)
	}
}

func tcServiceType(kind ChannelKind) string {
	switch kind {
	case ChannelKindGemini:
		return "gemini"
	case ChannelKindResponses:
		return "responses"
	case ChannelKindChat:
		return "openai"
	default:
		return "claude"
	}
}

func TestNormalizedMetricsServiceType(t *testing.T) {
	tests := []struct {
		name       string
		kind       ChannelKind
		configured string
		want       string
	}{
		{name: "messages default", kind: ChannelKindMessages, want: "claude"},
		{name: "responses default", kind: ChannelKindResponses, want: "responses"},
		{name: "gemini default", kind: ChannelKindGemini, want: "gemini"},
		{name: "chat default", kind: ChannelKindChat, want: "openai"},
		{name: "configured wins", kind: ChannelKindChat, configured: "responses", want: "responses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizedMetricsServiceType(tt.kind, tt.configured); got != tt.want {
				t.Fatalf("NormalizedMetricsServiceType(%q, %q)=%q, want=%q", tt.kind, tt.configured, got, tt.want)
			}
		})
	}
}

func TestSelectChannelByName(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "channel-a",
				BaseURL:  "https://a.example.com",
				APIKeys:  []string{"sk-a"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "channel-b",
				BaseURL:  "https://b.example.com",
				APIKeys:  []string{"sk-b"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	t.Run("指定渠道名直接定位", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "channel-b")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("期望选择 index=1 (channel-b)，实际为 %d", result.ChannelIndex)
		}
		if result.Reason != "channel_pin" {
			t.Fatalf("期望原因为 channel_pin，实际为 %s", result.Reason)
		}
		if result.Upstream.Name != "channel-b" {
			t.Fatalf("期望选择 channel-b，实际为 %s", result.Upstream.Name)
		}
	})

	t.Run("指定渠道名跳过更高优先级渠道", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "channel-b")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("应跳过高优先级 channel-a，实际选择 index=%d", result.ChannelIndex)
		}
	})

	t.Run("指定不存在的渠道名返回错误", func(t *testing.T) {
		_, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "nonexistent")
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if !strings.Contains(err.Error(), "nonexistent") {
			t.Fatalf("错误信息应包含渠道名，实际: %v", err)
		}
	})

	t.Run("指定已失败的渠道名返回错误", func(t *testing.T) {
		failed := map[int]bool{1: true}
		_, err := scheduler.SelectChannel(context.Background(), "test-user", failed, ChannelKindMessages, "", "", "channel-b")
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if !strings.Contains(err.Error(), "已失败") {
			t.Fatalf("错误信息应提示已失败，实际: %v", err)
		}
	})

	t.Run("空渠道名走正常选择逻辑", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 0 {
			t.Fatalf("正常逻辑应选高优先级 index=0，实际为 %d", result.ChannelIndex)
		}
		if result.Reason == "channel_pin" {
			t.Fatal("空渠道名不应触发 channel_pin")
		}
	})
}

func TestSelectChannelFiltersByContextWindowStableOrder(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "cheap-200k",
				BaseURL:  "https://cheap.example.com",
				APIKeys:  []string{"sk-cheap"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-5",
				},
			},
			{
				Name:     "mid-272k",
				BaseURL:  "https://mid.example.com",
				APIKeys:  []string{"sk-mid"},
				Status:   "active",
				Priority: 2,
				ModelCapabilities: map[string]config.UpstreamModelCapability{
					"mid-model": {ContextWindowTokens: 272000},
				},
				ModelMapping: map[string]string{
					"agent": "mid-model",
				},
			},
			{
				Name:     "premium-1m",
				BaseURL:  "https://premium.example.com",
				APIKeys:  []string{"sk-premium"},
				Status:   "active",
				Priority: 3,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-6",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	tests := []struct {
		name     string
		required int
		wantName string
	}{
		{name: "50k keeps first 200k channel", required: 50000, wantName: "cheap-200k"},
		{name: "230k skips 200k and keeps 272k first", required: 230000, wantName: "mid-272k"},
		{name: "500k skips 200k and 272k", required: 500000, wantName: "premium-1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
				UserID:         "user-context",
				FailedChannels: map[int]bool{},
				Kind:           ChannelKindMessages,
				Model:          "agent",
				ContextRequirement: &ContextRequirement{
					InputTokens:    tt.required - 8192,
					OutputTokens:   8192,
					RequiredTokens: tt.required,
				},
			})
			if err != nil {
				t.Fatalf("SelectChannelWithOptions() error = %v", err)
			}
			if result.Upstream.Name != tt.wantName {
				t.Fatalf("selected channel = %q, want %q", result.Upstream.Name, tt.wantName)
			}
		})
	}
}

func TestSelectChannelUsesAgentProfileMinimumWindow(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "legacy-200k",
				BaseURL:  "https://legacy.example.com",
				APIKeys:  []string{"sk-legacy"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"sonnet": "claude-sonnet-4-5",
					"haiku":  "claude-haiku-4-5",
				},
			},
			{
				Name:     "modern-1m",
				BaseURL:  "https://modern.example.com",
				APIKeys:  []string{"sk-modern"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"sonnet": "claude-sonnet-4-6",
					"haiku":  "claude-haiku-4-5",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	sonnetProfile := config.ResolveAgentModelProfile("sonnet", nil)
	if !sonnetProfile.Known {
		t.Fatal("expected sonnet agent profile")
	}
	sonnet, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-sonnet-profile",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "sonnet",
		ContextRequirement: &ContextRequirement{
			InputTokens:                41808,
			OutputTokens:               8192,
			RequiredTokens:             50000,
			MinimumContextWindowTokens: sonnetProfile.Profile.ContextWindowTokens,
		},
	})
	if err != nil {
		t.Fatalf("sonnet SelectChannelWithOptions() error = %v", err)
	}
	if sonnet.Upstream.Name != "modern-1m" {
		t.Fatalf("sonnet selected channel = %q, want modern-1m", sonnet.Upstream.Name)
	}

	haikuProfile := config.ResolveAgentModelProfile("haiku", nil)
	if !haikuProfile.Known {
		t.Fatal("expected haiku agent profile")
	}
	haiku, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-haiku-profile",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "haiku",
		ContextRequirement: &ContextRequirement{
			InputTokens:                41808,
			OutputTokens:               8192,
			RequiredTokens:             50000,
			MinimumContextWindowTokens: haikuProfile.Profile.ContextWindowTokens,
		},
	})
	if err != nil {
		t.Fatalf("haiku SelectChannelWithOptions() error = %v", err)
	}
	if haiku.Upstream.Name != "legacy-200k" {
		t.Fatalf("haiku selected channel = %q, want legacy-200k", haiku.Upstream.Name)
	}
}

func TestSelectChannelFiltersExplicitOutputLimit(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "sonnet-64k-output",
				BaseURL:  "https://sonnet.example.com",
				APIKeys:  []string{"sk-sonnet"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-6",
				},
			},
			{
				Name:     "opus-128k-output",
				BaseURL:  "https://opus.example.com",
				APIKeys:  []string{"sk-opus"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"agent": "claude-opus-4-8",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	result, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-output",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "agent",
		ContextRequirement: &ContextRequirement{
			InputTokens:       1000,
			OutputTokens:      128000,
			RequiredTokens:    129000,
			ExplicitOutputMax: true,
		},
	})
	if err != nil {
		t.Fatalf("SelectChannelWithOptions() error = %v", err)
	}
	if result.Upstream.Name != "opus-128k-output" {
		t.Fatalf("selected channel = %q, want opus-128k-output", result.Upstream.Name)
	}
}

func TestSelectChannelCompactionSkipsWindowButKeepsOutputLimit(t *testing.T) {
	cfg := config.Config{
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:     "sonnet-200k",
				BaseURL:  "https://sonnet.example.com",
				APIKeys:  []string{"sk-sonnet"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-5",
				},
			},
			{
				Name:     "opus-128k-output",
				BaseURL:  "https://opus.example.com",
				APIKeys:  []string{"sk-opus"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"agent": "claude-opus-4-8",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	result, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-compact",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindResponses,
		Model:          "agent",
		ContextRequirement: &ContextRequirement{
			InputTokens:          491808,
			OutputTokens:         8192,
			RequiredTokens:       500000,
			SkipWindowValidation: true,
		},
	})
	if err != nil {
		t.Fatalf("SelectChannelWithOptions() error = %v", err)
	}
	if result.Upstream.Name != "sonnet-200k" {
		t.Fatalf("selected channel = %q, want sonnet-200k", result.Upstream.Name)
	}

	result, err = scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-compact-output",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindResponses,
		Model:          "agent",
		ContextRequirement: &ContextRequirement{
			InputTokens:          1000,
			OutputTokens:         128000,
			RequiredTokens:       129000,
			ExplicitOutputMax:    true,
			SkipWindowValidation: true,
		},
	})
	if err != nil {
		t.Fatalf("SelectChannelWithOptions() output error = %v", err)
	}
	if result.Upstream.Name != "opus-128k-output" {
		t.Fatalf("selected output channel = %q, want opus-128k-output", result.Upstream.Name)
	}
}

func TestSelectChannelPinnedChannelMustSatisfyContext(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "cheap-200k",
				BaseURL:  "https://cheap.example.com",
				APIKeys:  []string{"sk-cheap"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-5",
				},
			},
			{
				Name:     "premium-1m",
				BaseURL:  "https://premium.example.com",
				APIKeys:  []string{"sk-premium"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-6",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	_, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-pin",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "agent",
		ChannelName:    "cheap-200k",
		ContextRequirement: &ContextRequirement{
			InputTokens:    491808,
			OutputTokens:   8192,
			RequiredTokens: 500000,
		},
	})
	if err == nil {
		t.Fatal("SelectChannelWithOptions() error = nil, want pinned context rejection")
	}
	if !strings.Contains(err.Error(), "指定渠道") {
		t.Fatalf("error = %q, want pinned channel rejection", err.Error())
	}
}

func TestSelectChannelUnknownContextPolicy(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "unknown",
				BaseURL:  "https://unknown.example.com",
				APIKeys:  []string{"sk-unknown"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "vendor-unknown",
				},
			},
			{
				Name:     "premium-1m",
				BaseURL:  "https://premium.example.com",
				APIKeys:  []string{"sk-premium"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-6",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	small, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-small",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "agent",
		ContextRequirement: &ContextRequirement{
			InputTokens:    41808,
			OutputTokens:   8192,
			RequiredTokens: 50000,
		},
	})
	if err != nil {
		t.Fatalf("small SelectChannelWithOptions() error = %v", err)
	}
	if small.Upstream.Name != "unknown" {
		t.Fatalf("small selected channel = %q, want unknown", small.Upstream.Name)
	}

	large, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-large",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "agent",
		ContextRequirement: &ContextRequirement{
			InputTokens:    491808,
			OutputTokens:   8192,
			RequiredTokens: 500000,
		},
	})
	if err != nil {
		t.Fatalf("large SelectChannelWithOptions() error = %v", err)
	}
	if large.Upstream.Name != "premium-1m" {
		t.Fatalf("large selected channel = %q, want premium-1m", large.Upstream.Name)
	}
}

func TestSelectChannelManualOverridePreservedWhenContextFiltersChannel(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "manual-cheap-200k",
				BaseURL:  "https://cheap.example.com",
				APIKeys:  []string{"sk-cheap"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-5",
				},
			},
			{
				Name:     "fallback-1m",
				BaseURL:  "https://premium.example.com",
				APIKeys:  []string{"sk-premium"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-6",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	overrideMgr := conversation.NewOverrideManager(time.Hour)
	defer overrideMgr.Stop()
	if err := overrideMgr.SetOverride("conv-1", string(ChannelKindMessages), "user-override", []conversation.ChannelEntry{
		{ChannelIndex: 0, ChannelName: "manual-cheap-200k"},
	}, time.Hour); err != nil {
		t.Fatalf("SetOverride() error = %v", err)
	}
	scheduler.SetConversationComponents(nil, overrideMgr)

	result, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:         "user-override",
		FailedChannels: map[int]bool{},
		Kind:           ChannelKindMessages,
		Model:          "agent",
		ContextRequirement: &ContextRequirement{
			InputTokens:    491808,
			OutputTokens:   8192,
			RequiredTokens: 500000,
		},
	})
	if err != nil {
		t.Fatalf("SelectChannelWithOptions() error = %v", err)
	}
	if result.Upstream.Name != "fallback-1m" {
		t.Fatalf("selected channel = %q, want fallback-1m", result.Upstream.Name)
	}
	if _, ok := overrideMgr.GetOverrideForUser(string(ChannelKindMessages), "user-override"); !ok {
		t.Fatal("manual override was cleared; want preserved")
	}
}

func TestTraceAffinityUsesContextBuckets(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "cheap-200k",
				BaseURL:  "https://cheap.example.com",
				APIKeys:  []string{"sk-cheap"},
				Status:   "active",
				Priority: 1,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-5",
				},
			},
			{
				Name:     "premium-1m",
				BaseURL:  "https://premium.example.com",
				APIKeys:  []string{"sk-premium"},
				Status:   "active",
				Priority: 2,
				ModelMapping: map[string]string{
					"agent": "claude-sonnet-4-6",
				},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	largeReq := &ContextRequirement{InputTokens: 491808, OutputTokens: 8192, RequiredTokens: 500000}
	smallReq := &ContextRequirement{InputTokens: 41808, OutputTokens: 8192, RequiredTokens: 50000}
	scheduler.SetTraceAffinityForRequirement("user-bucket", 1, ChannelKindMessages, largeReq)

	result, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		UserID:             "user-bucket",
		FailedChannels:     map[int]bool{},
		Kind:               ChannelKindMessages,
		Model:              "agent",
		ContextRequirement: smallReq,
	})
	if err != nil {
		t.Fatalf("SelectChannelWithOptions() error = %v", err)
	}
	if result.Upstream.Name != "cheap-200k" {
		t.Fatalf("small context selected channel = %q, want cheap-200k", result.Upstream.Name)
	}
}
