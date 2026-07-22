package autopilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/utils"
)

// DiscoveryStatus 发现任务状态枚举。
type DiscoveryStatus string

const (
	DiscoveryStatusIdle    DiscoveryStatus = "idle"
	DiscoveryStatusRunning DiscoveryStatus = "running"
	DiscoveryStatusDone    DiscoveryStatus = "done"
	DiscoveryStatusFailed  DiscoveryStatus = "failed"
)

// ModelDiscoverySource 描述模型清单的事实来源。
// 它独立于 KeyEndpointProfile.Source；后者会在后续 L1 画像刷新时变成
// l1_passive，不能用来判断模型清单是否来自管控面或静态兜底。
const (
	ModelDiscoverySourceControlPlane    = "control_plane"
	ModelDiscoverySourceModelsAPI       = "models_api"
	ModelDiscoverySourceBuiltinManifest = "builtin_manifest"
	ModelDiscoverySourceBuiltinFallback = "builtin_fallback"
)

// EndpointDiscoveryResult 单个 (baseURL, key) 端点的发现结果。
type EndpointDiscoveryResult struct {
	KeyMask               string     `json:"keyMask"`
	BaseURL               string     `json:"baseUrl"`
	ModelsCount           int        `json:"modelsCount"`
	Models                []string   `json:"models,omitempty"`
	ProtocolOk            bool       `json:"protocolOk"`
	ErrorMessage          string     `json:"errorMessage,omitempty"`
	ModelDiscoverySource  string     `json:"modelDiscoverySource,omitempty"`
	ModelDiscoveryMessage string     `json:"modelDiscoveryMessage,omitempty"`
	ModelsDiscoveredAt    *time.Time `json:"modelsDiscoveredAt,omitempty"`
}

// DiscoveryTask 单渠道发现任务的运行时状态。
type DiscoveryTask struct {
	ChannelUID string                    `json:"channelUid"`
	Status     DiscoveryStatus           `json:"status"`
	StartedAt  *time.Time                `json:"startedAt,omitempty"`
	FinishedAt *time.Time                `json:"finishedAt,omitempty"`
	Error      string                    `json:"error,omitempty"`
	Endpoints  []EndpointDiscoveryResult `json:"endpoints"`
	cancel     context.CancelFunc        `json:"-"`
}

// AutoDiscoveryRunner 自动发现执行器。
// 内存状态机：每个渠道同时只运行一个发现任务，重复触发会被拒绝。
// 所有配置为空时零值即可用，不触发任何实际操作。
type AutoDiscoveryRunner struct {
	mu                             sync.Mutex
	tasks                          map[string]*DiscoveryTask // channelUID -> task
	store                          *ProfileStore             // nil 时不写画像，只记录结果
	hub                            *EventHub                 // nil 时不发布 discovery_completed/auto_mapping_applied 事件
	client                         *http.Client              // nil 时使用默认 client
	timeout                        time.Duration             // 单次请求超时，默认 10s
	volcengineControlPlaneEndpoint string

	// Phase 3B-2：自动发现时同步写入模型画像（nil 时不写 model_profiles，不影响现有功能）
	ModelProfileStore *ModelProfileStore
}

// NewAutoDiscoveryRunner 创建发现执行器。
// store 可为 nil（仅记录内存结果，不写持久化画像）。
// hub 可为 nil（不发布 Phase 3A 画像变更事件，向后兼容旧调用点）。
func NewAutoDiscoveryRunner(store *ProfileStore, hub *EventHub) *AutoDiscoveryRunner {
	return &AutoDiscoveryRunner{
		tasks:   make(map[string]*DiscoveryTask),
		store:   store,
		hub:     hub,
		timeout: 10 * time.Second,
	}
}

// GetTask 返回指定渠道的发现任务快照（nil 表示从未触发）。
func (r *AutoDiscoveryRunner) GetTask(channelUID string) *DiscoveryTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	task := r.tasks[channelUID]
	if task == nil {
		return nil
	}
	// 返回快照，不暴露 cancel
	snap := *task
	snap.cancel = nil
	// 深拷贝 Endpoints
	if len(task.Endpoints) > 0 {
		snap.Endpoints = make([]EndpointDiscoveryResult, len(task.Endpoints))
		copy(snap.Endpoints, task.Endpoints)
	}
	return &snap
}

// TriggerDiscovery 触发发现任务。
// 如果同渠道已有 running 任务则返回 false（拒绝重复触发）。
// 返回 true 表示已成功触发。
func (r *AutoDiscoveryRunner) TriggerDiscovery(channelUID string, channel *config.UpstreamConfig, cfgManager *config.ConfigManager) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.tasks[channelUID]; ok && existing.Status == DiscoveryStatusRunning {
		log.Printf("[AutoDiscovery-Trigger] 渠道 %s 发现任务已在运行中，拒绝重复触发", channelUID)
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	task := &DiscoveryTask{
		ChannelUID: channelUID,
		Status:     DiscoveryStatusRunning,
		StartedAt:  &now,
		cancel:     cancel,
	}
	r.tasks[channelUID] = task

	go r.runDiscovery(ctx, task, channel, cfgManager)
	return true
}

// runDiscovery 执行发现逻辑（在后台 goroutine 中运行）。
func (r *AutoDiscoveryRunner) runDiscovery(ctx context.Context, task *DiscoveryTask, channel *config.UpstreamConfig, cfgManager *config.ConfigManager) {
	defer func() {
		if rec := recover(); rec != nil {
			r.mu.Lock()
			task.Status = DiscoveryStatusFailed
			now := time.Now()
			task.FinishedAt = &now
			task.Error = fmt.Sprintf("panic: %v", rec)
			r.mu.Unlock()
			log.Printf("[AutoDiscovery-Run] 渠道 %s 发现任务 panic: %v", task.ChannelUID, rec)
		}
	}()

	endpoints := r.discoverEndpoints(ctx, channel, cfgManager)

	r.mu.Lock()
	task.Endpoints = endpoints
	now := time.Now()
	task.FinishedAt = &now

	// 检查是否有失败的端点
	failedCount := 0
	for _, ep := range endpoints {
		if !ep.ProtocolOk {
			failedCount++
		}
	}
	if failedCount == len(endpoints) && len(endpoints) > 0 {
		task.Status = DiscoveryStatusFailed
		task.Error = "所有端点均不可达"
	} else {
		task.Status = DiscoveryStatusDone
	}
	r.mu.Unlock()

	// 写画像到 ProfileStore + ModelProfileStore（在锁外执行，避免阻塞其他操作）
	r.writeProfiles(task.ChannelUID, channel, endpoints, cfgManager)

	// 探测完成后尝试自动写入 SupportedModels（安全守则：仅一致结果且用户未手动配置时写入）
	if cfgManager != nil {
		r.maybeAutoWriteChannelConfig(task.ChannelUID, channel, endpoints, cfgManager)
	}

	// Phase 3A：发布 discovery_completed 事件（只读展示，不影响调度）
	if r.hub != nil {
		channelKind := ""
		if cfgManager != nil {
			_, channelKind = findChannelIndexAndKind(cfgManager.GetConfig(), task.ChannelUID)
		}
		summary := fmt.Sprintf("%d/%d 端点可达", len(endpoints)-failedCount, len(endpoints))
		if task.Status == DiscoveryStatusFailed {
			summary = "发现失败: " + task.Error
		}
		now := time.Now()
		ev := ProfileChangeEvent{
			ChannelUID:  task.ChannelUID,
			ChannelKind: channelKind,
			EventType:   EventTypeDiscoveryComplete,
			Summary:     summary,
			CreatedAt:   now,
		}
		ev.EventUID = GenerateChangeEventUID(task.ChannelUID, ev.EventType, now)
		r.hub.Publish(ev)
	}

	log.Printf("[AutoDiscovery-Run] 渠道 %s 发现完成: %d/%d 端点可达",
		task.ChannelUID, len(endpoints)-failedCount, len(endpoints))
}

// discoverEndpoints 遍历所有 (baseURL, key) 组合，调用 GET /v1/models。
func (r *AutoDiscoveryRunner) discoverEndpoints(ctx context.Context, channel *config.UpstreamConfig, managers ...*config.ConfigManager) []EndpointDiscoveryResult {
	baseURLs := channel.GetAllBaseURLs()
	keys := channel.APIKeys

	if len(baseURLs) == 0 || len(keys) == 0 {
		return nil
	}

	client := r.client
	if client == nil {
		client = &http.Client{Timeout: r.timeout}
	}

	var results []EndpointDiscoveryResult
	var cfgManager *config.ConfigManager
	if len(managers) > 0 {
		cfgManager = managers[0]
	}
	for _, key := range keys {
		keyBaseURLs := baseURLs
		if bound := channel.BoundBaseURLForKey(key); bound != "" {
			keyBaseURLs = []string{bound}
		}
		for _, baseURL := range keyBaseURLs {
			select {
			case <-ctx.Done():
				return results
			default:
			}

			var result EndpointDiscoveryResult
			if channel.ProviderID == "volcengine" {
				result = r.discoverVolcenginePlanEndpoint(ctx, client, channel, baseURL, key, cfgManager)
			} else {
				result = r.probeEndpoint(ctx, client, channel, baseURL, key)
			}
			logEndpointDiscovery(channel.ChannelUID, result)
			results = append(results, result)
		}
	}
	return results
}

func (r *AutoDiscoveryRunner) discoverVolcenginePlanEndpoint(
	ctx context.Context,
	client *http.Client,
	channel *config.UpstreamConfig,
	baseURL string,
	apiKey string,
	cfgManager *config.ConfigManager,
) EndpointDiscoveryResult {
	result := EndpointDiscoveryResult{KeyMask: utils.MaskAPIKey(apiKey), BaseURL: baseURL}

	// 缺少账号上下文或未绑定 Access Key 时，回退到内置兜底模型清单，
	// 让渠道立即可用（绑定 Access Key 后会由管控面 FetchModels 覆盖为真实清单）。
	if cfgManager == nil || channel == nil || channel.AccountUID == "" {
		if r.applyVolcengineFallback(&result, channel, baseURL, "缺少自动托管账号上下文，已回退内置模型清单") {
			return result
		}
		result.ErrorMessage = "火山套餐模型发现需要自动托管账号上下文"
		return result
	}
	credentialUID := channel.CredentialUIDForKey(apiKey)
	credential, ok := cfgManager.GetManagedAccountCredential(channel.AccountUID, credentialUID)
	if !ok || credential.VolcengineAccessKey == nil {
		if r.applyVolcengineFallback(&result, channel, baseURL, "未绑定火山云 Access Key，已回退内置模型清单") {
			return result
		}
		result.ErrorMessage = "请先为该推理 Key 绑定火山云 Access Key ID 与 Secret Access Key"
		return result
	}
	planClient := &volcenginePlanClient{
		Endpoint:   r.volcengineControlPlaneEndpoint,
		HTTPClient: client,
	}
	plan, err := planClient.DetectPlan(ctx, credential.VolcengineAccessKey, volcenginePlanFromBaseURL(baseURL))
	if err != nil {
		if r.applyVolcengineFallback(&result, channel, baseURL, fmt.Sprintf("套餐识别失败(%v)，已回退内置模型清单", err)) {
			return result
		}
		result.ErrorMessage = err.Error()
		return result
	}
	models, err := planClient.FetchModels(ctx, credential.VolcengineAccessKey, plan.Plan)
	if err != nil {
		if r.applyVolcengineFallback(&result, channel, baseURL, fmt.Sprintf("模型发现失败(%v)，已回退内置模型清单", err)) {
			return result
		}
		result.ErrorMessage = err.Error()
		return result
	}
	if err := cfgManager.SetManagedAccountVolcenginePlan(channel.AccountUID, credentialUID, plan.Plan, plan.Tier, plan.Status); err != nil {
		log.Printf("[AutoDiscovery-Volcengine] 保存套餐识别结果失败 credential=%s: %v", credentialUID, err)
	}
	result.Models = models
	result.ModelsCount = len(models)
	result.ProtocolOk = true
	setModelDiscoveryMetadata(&result, ModelDiscoverySourceControlPlane, fmt.Sprintf("火山管控面 %s 模型清单", displayVolcenginePlan(plan.Plan)))
	return result
}

// applyVolcengineFallback 在无法通过火山管控面发现模型时，尝试命中内置兜底清单。
// 命中则填充 result（ProtocolOk=true）并返回 true；未命中返回 false。
func (r *AutoDiscoveryRunner) applyVolcengineFallback(result *EndpointDiscoveryResult, channel *config.UpstreamConfig, baseURL, message string) bool {
	manifest, ok := lookupDiscoveryBuiltinManifest(channel, baseURL)
	if !ok || len(manifest.ModelIDs) == 0 {
		return false
	}
	applyBuiltinFallbackModels(result, manifest, message)
	return result.ProtocolOk
}

// probeEndpoint 探测单个 (baseURL, key) 组合。
// 优先遵循内置模型清单；否则调 GET /v1/models 检查协议可达性和模型列表。
func (r *AutoDiscoveryRunner) probeEndpoint(ctx context.Context, client *http.Client, channel *config.UpstreamConfig, baseURL, apiKey string) EndpointDiscoveryResult {
	result := EndpointDiscoveryResult{
		KeyMask: utils.MaskAPIKey(apiKey),
		BaseURL: baseURL,
	}
	if channel == nil {
		result.ErrorMessage = "渠道配置为空"
		return result
	}

	manifest, hasManifest := lookupDiscoveryBuiltinManifest(channel, baseURL)
	if hasManifest && manifest.DisableProbe {
		if discoveryManifestServiceType(channel.ServiceType) != "messages" {
			applyBuiltinModels(&result, manifest, "内置模型清单")
			return result
		}
		verify := VerifyClaudeEndpoint(ctx, baseURL, apiKey, channel.AuthHeader)
		if verify.OK {
			applyBuiltinModels(&result, manifest, "内置模型清单")
			return result
		}
		result.ErrorMessage = verify.Message
		if result.ErrorMessage == "" && verify.Err != nil {
			result.ErrorMessage = verify.Err.Error()
		}
		if result.ErrorMessage == "" {
			result.ErrorMessage = fmt.Sprintf("HTTP %d", verify.StatusCode)
		}
		return result
	}

	// 构建 models URL。个别 provider 的协议入口与模型列表入口不同，优先使用清单覆盖值。
	modelsURL := buildModelsProbeURL(baseURL)
	if manifestURL, ok := config.ResolveBuiltinModelsURL(baseURL, discoveryManifestServiceType(channel.ServiceType)); ok {
		modelsURL = manifestURL
	}
	if channel.ServiceType == "gemini" {
		// Gemini 不支持 /v1/models，跳过
		result.ProtocolOk = false
		result.ErrorMessage = "Gemini 暂不支持 models 探测"
		return result
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("构建请求失败: %v", err)
		return result
	}

	// 设置认证头
	utils.SetAuthenticationHeaderWithOverride(req.Header, apiKey, channel.AuthHeader)

	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("请求失败: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if hasManifest && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
			applyBuiltinFallbackModels(&result, manifest, fmt.Sprintf("models 端点返回 HTTP %d，已回退内置模型清单", resp.StatusCode))
			return result
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		result.ErrorMessage = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return result
	}

	// 解析 models 响应
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("读取响应失败: %v", err)
		return result
	}

	models := parseModelsResponse(body)
	if hasManifest {
		models = filterExcludedDiscoveryModels(models, manifest.ExcludeModelPatterns)
	}
	if len(models) == 0 && hasManifest {
		applyBuiltinFallbackModels(&result, manifest, "models 端点返回空列表，已回退内置模型清单")
		return result
	}
	result.ModelsCount = len(models)
	result.Models = models
	result.ProtocolOk = true
	setModelDiscoveryMetadata(&result, ModelDiscoverySourceModelsAPI, "models API 返回实时模型清单")

	return result
}

func buildModelsProbeURL(baseURL string) string {
	skipVersionPrefix := strings.HasSuffix(baseURL, "#")
	if skipVersionPrefix {
		baseURL = strings.TrimSuffix(baseURL, "#")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if verifyVersionPattern.MatchString(baseURL) || skipVersionPrefix {
		return baseURL + "/models"
	}
	return baseURL + "/v1/models"
}

func lookupDiscoveryBuiltinManifest(channel *config.UpstreamConfig, baseURL string) (config.BuiltinModelsManifest, bool) {
	if channel == nil {
		return config.BuiltinModelsManifest{}, false
	}
	serviceType := discoveryManifestServiceType(channel.ServiceType)
	if serviceType == "" {
		return config.BuiltinModelsManifest{}, false
	}
	if manifest, ok := config.LookupBuiltinManifest(baseURL, serviceType); ok {
		return manifest, true
	}
	// OpenAI 兼容渠道在运行时会把末尾 /v1 规范化掉，但清单按实际 models
	// 端点记录为 host/v1。仅在直接匹配失败时补 /v1 重试。
	if serviceType == "openai" {
		return config.LookupBuiltinManifest(strings.TrimRight(baseURL, "/")+"/v1", serviceType)
	}
	return config.BuiltinModelsManifest{}, false
}

func discoveryManifestServiceType(serviceType string) string {
	switch strings.ToLower(strings.TrimSpace(serviceType)) {
	case "claude":
		return "messages"
	default:
		return strings.ToLower(strings.TrimSpace(serviceType))
	}
}

func applyBuiltinModels(result *EndpointDiscoveryResult, manifest config.BuiltinModelsManifest, message string) {
	result.Models = filterExcludedDiscoveryModels(append([]string(nil), manifest.ModelIDs...), manifest.ExcludeModelPatterns)
	result.ModelsCount = len(result.Models)
	result.ProtocolOk = len(result.Models) > 0
	result.ErrorMessage = message
	setModelDiscoveryMetadata(result, ModelDiscoverySourceBuiltinManifest, message)
}

// applyBuiltinFallbackModels 与明确配置的内置清单区分开来，记录这是一次
// 管控面/models API 失败后的回退，而不是正常的静态清单来源。
func applyBuiltinFallbackModels(result *EndpointDiscoveryResult, manifest config.BuiltinModelsManifest, message string) {
	applyBuiltinModels(result, manifest, message)
	result.ModelDiscoverySource = ModelDiscoverySourceBuiltinFallback
}

func setModelDiscoveryMetadata(result *EndpointDiscoveryResult, source, message string) {
	if result == nil {
		return
	}
	now := time.Now().UTC()
	result.ModelDiscoverySource = source
	result.ModelDiscoveryMessage = strings.TrimSpace(message)
	result.ModelsDiscoveredAt = &now
}

func logEndpointDiscovery(channelUID string, result EndpointDiscoveryResult) {
	source := result.ModelDiscoverySource
	if source == "" {
		source = "unknown"
	}
	discoveredAt := "-"
	if result.ModelsDiscoveredAt != nil {
		discoveredAt = result.ModelsDiscoveredAt.UTC().Format(time.RFC3339)
	}
	message := strings.TrimSpace(result.ModelDiscoveryMessage)
	if message == "" {
		message = strings.TrimSpace(result.ErrorMessage)
	}
	log.Printf("[AutoDiscovery-Endpoint] channel=%s key=%s baseURL=%s source=%s models=%d discoveredAt=%s protocolOk=%t message=%s",
		channelUID, result.KeyMask, result.BaseURL, source, result.ModelsCount, discoveredAt, result.ProtocolOk, message)
}

func filterExcludedDiscoveryModels(models []string, patterns []string) []string {
	if len(models) == 0 || len(patterns) == 0 {
		return models
	}

	excludeRules := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		rule, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("[AutoDiscovery-ModelsFilter] 忽略非法排除正则 %q: %v", pattern, err)
			continue
		}
		excludeRules = append(excludeRules, rule)
	}
	if len(excludeRules) == 0 {
		return models
	}

	filtered := make([]string, 0, len(models))
	for _, modelID := range models {
		excluded := false
		for _, rule := range excludeRules {
			if rule.MatchString(modelID) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, modelID)
		}
	}
	return filtered
}

// parseModelsResponse 解析 OpenAI /v1/models 响应体。
func parseModelsResponse(body []byte) []string {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	models := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models
}

// writeProfiles 将发现结果写入 KeyEndpointProfile。
// MVP：只更新 ModelListHash / AvailableModels / Source / UpdatedAt，不修改 modelMapping。
// Phase 3B-2：同时为 autoManaged 渠道写入每发现模型的 ModelProfile 行（仅当 modelProfileStore != nil）。
func (r *AutoDiscoveryRunner) writeProfiles(channelUID string, channel *config.UpstreamConfig, endpoints []EndpointDiscoveryResult, cfgManager *config.ConfigManager) {
	if r.store == nil {
		return
	}

	// Phase 3B-2：准备全局上游模型能力和渠道类型（用于 model_profiles 填充）
	var globalModelCapabilities map[string]config.UpstreamModelCapability
	var channelKind string
	var channelID int
	if cfgManager != nil {
		cfg := cfgManager.GetConfig()
		globalModelCapabilities = cfg.UpstreamModelCapabilities
		var idx int
		idx, channelKind = findChannelIndexAndKind(cfg, channelUID)
		if idx >= 0 {
			channelID = idx
		}
	}

	for _, ep := range endpoints {
		if !ep.ProtocolOk {
			continue
		}

		// 从 channel 的 APIKeys 中找到对应 key
		apiKey := ""
		for _, key := range channel.APIKeys {
			if utils.MaskAPIKey(key) == ep.KeyMask {
				apiKey = key
				break
			}
		}
		if apiKey == "" {
			continue
		}

		keyHash := KeyHashFromAPIKey(apiKey)
		metricsKey := computeMetricsIdentityKey(ep.BaseURL, apiKey, channel.ServiceType)
		endpointUID := GenerateEndpointUID(channelUID, ep.BaseURL, keyHash)

		// 尝试获取已有画像
		existing := r.store.Get(endpointUID)

		var profile KeyEndpointProfile
		if existing != nil {
			profile = *existing
		}

		// 更新发现相关字段
		profile.EndpointUID = endpointUID
		profile.AccountUID = channel.AccountUID
		profile.ChannelUID = channelUID
		profile.ChannelID = channelID
		if channelKind != "" {
			profile.ChannelKind = channelKind
		}
		if channel.ServiceType != "" {
			profile.ServiceType = channel.ServiceType
		}
		// Discovery 只证明端点和模型列表可达，不应把尚未经过 L1 profiler 的空档位
		// 解释为 low/unstable。使用与 SmartRouter 无画像路径相同的中性默认值。
		if profile.HealthState == "" {
			profile.HealthState = HealthStateUnknown
		}
		if profile.QualityTier == "" {
			profile.QualityTier = QualityTierNormal
		}
		if profile.StabilityTier == "" {
			profile.StabilityTier = StabilityTierNormal
		}
		if profile.SpeedTier == "" {
			profile.SpeedTier = SpeedTierNormal
		}
		if profile.CostTier == "" {
			profile.CostTier = CostTierNormal
		}
		profile.BaseURL = ep.BaseURL
		profile.IdentityBaseURL = utils.MetricsIdentityBaseURL(ep.BaseURL, channel.ServiceType)
		profile.KeyMask = ep.KeyMask
		profile.KeyHash = keyHash
		profile.MetricsKey = metricsKey
		profile.CredentialUID = channel.CredentialUIDForKey(apiKey)
		profile.AvailableModels = ep.Models
		if len(ep.Models) > 0 {
			hash := sha256.Sum256([]byte(strings.Join(ep.Models, ",")))
			profile.ModelListHash = hex.EncodeToString(hash[:8])
		}
		if ep.ModelDiscoverySource != "" {
			profile.ModelDiscoverySource = ep.ModelDiscoverySource
		}
		if ep.ModelDiscoveryMessage != "" {
			profile.ModelDiscoveryMessage = ep.ModelDiscoveryMessage
		}
		if ep.ModelsDiscoveredAt != nil {
			discoveredAt := ep.ModelsDiscoveredAt.UTC()
			profile.ModelsDiscoveredAt = &discoveredAt
		}
		profile.Source = "auto_discovery"
		profile.UpdatedAt = time.Now()

		if err := r.store.Upsert(&profile); err != nil {
			log.Printf("[AutoDiscovery-Profile] 写入画像失败 endpoint=%s: %v", endpointUID, err)
		}

		// Phase 3B-2：写入每个发现模型的 ModelProfile 行
		// 条件：modelProfileStore 非 nil + channel.AutoManaged == true + 有发现模型
		if r.ModelProfileStore != nil && channel.AutoManaged && len(ep.Models) > 0 {
			now := time.Now()
			for _, modelID := range ep.Models {
				family := InferModelFamily(modelID, "")
				qualityTier := ModelProfileQualityTierFromFamily(family, modelID)

				modelProfile := &ModelProfile{
					ChannelUID:   channelUID,
					ChannelID:    channelID,
					ChannelKind:  channelKind,
					ServiceType:  channel.ServiceType,
					MetricsKey:   metricsKey,
					ModelID:      modelID,
					UpdatedAt:    now,
					ModelFamily:  family,
					QualityTier:  qualityTier,
					ProbeSuccess: true, // 出现在 GET /v1/models 响应视为存在
					Source:       "auto_discovery",
				}
				if resolved := config.ResolveUpstreamCapability(modelID, channel, globalModelCapabilities); resolved.Known {
					applyUpstreamModelCapability(modelProfile, resolved.Capability)
				}
				// 自动发现会周期性重建能力画像，但不应清除 L3 探测或用户反馈写入的
				// endpoint×model 质量证据。证据只在同一复合主键下继承，避免跨 Key 污染。
				if existing := r.ModelProfileStore.Get(channelUID, channelKind, metricsKey, modelID); existing != nil {
					modelProfile.ProviderQualityScore = existing.ProviderQualityScore
					modelProfile.ProviderQualitySource = existing.ProviderQualitySource
					modelProfile.ProviderQualityConfidence = existing.ProviderQualityConfidence
					modelProfile.ProviderQualityProbeVersion = existing.ProviderQualityProbeVersion
					modelProfile.LastProbeAt = existing.LastProbeAt
					modelProfile.ProbeLatencyMs = existing.ProbeLatencyMs
					modelProfile.ProbeConfidence = existing.ProbeConfidence
				}
				if err := r.ModelProfileStore.Upsert(modelProfile); err != nil {
					log.Printf("[AutoDiscovery-ModelProfile] 写入模型画像失败 channel=%s model=%s: %v",
						channelUID, modelID, err)
				}
			}
		}
	}

	// 自动发现完成即持久化，避免等待下一轮 L1 画像刷新期间进程退出而丢失结果。
	if err := r.store.Flush(); err != nil {
		log.Printf("[AutoDiscovery-Profile] 画像落盘失败 channel=%s: %v", channelUID, err)
	}
	if r.ModelProfileStore != nil {
		if err := r.ModelProfileStore.Flush(); err != nil {
			log.Printf("[AutoDiscovery-ModelProfile] 模型画像落盘失败 channel=%s: %v", channelUID, err)
		}
	}
}

// maybeAutoWriteChannelConfig 在发现完成后，检查是否可以将一致模型列表写入渠道配置。
// 安全守则：
//  1. 仅当所有成功探测的 endpoint 返回完全相同的模型列表（集合相等，顺序无关）时才写入
//  2. 不覆盖用户已有的手动配置（SupportedModels 或 ModelMapping 非空时不写入）
//  3. ModelMapping 不自动写入（比 SupportedModels 更容易出错，留给用户手动确认）
func (r *AutoDiscoveryRunner) maybeAutoWriteChannelConfig(channelUID string, channel *config.UpstreamConfig, endpoints []EndpointDiscoveryResult, cfgManager *config.ConfigManager) {
	// cfgManager 为 nil 时直接返回（runDiscovery 入口已有 guard，此处防御直接调用）
	if cfgManager == nil {
		return
	}
	// 自动托管账号的模型可用性属于具体 binding，已写入 KeyEndpointProfile。
	// 不再回写渠道级 SupportedModels，避免多 Key 权限不一致时丢失可用候选。
	if channel != nil && channel.AutoManaged {
		log.Printf("[AutoDiscovery-ConfigSkip] 渠道 %s: 自动托管模型由 endpoint profile 持久化，不写渠道级 SupportedModels", channelUID)
		return
	}

	// 收集所有成功探测的 endpoint 的模型列表
	var okEndpoints []EndpointDiscoveryResult
	for _, ep := range endpoints {
		if ep.ProtocolOk && len(ep.Models) > 0 {
			okEndpoints = append(okEndpoints, ep)
		}
	}

	// 无成功探测结果，不写
	if len(okEndpoints) == 0 {
		log.Printf("[AutoDiscovery-ConfigSkip] 渠道 %s: 无可达端点或模型列表为空，跳过自动写入", channelUID)
		return
	}

	// 检查一致性：所有成功探测的 endpoint 返回的模型列表集合必须完全相同
	consistentModels := modelsSetConsistent(okEndpoints)
	if consistentModels == nil {
		log.Printf("[AutoDiscovery-ConfigSkip] 渠道 %s: 端点模型列表不一致（%d 个可达端点），跳过自动写入",
			channelUID, len(okEndpoints))
		return
	}

	// 检查用户已有配置：SupportedModels 或 ModelMapping 非空时不覆盖
	if len(channel.SupportedModels) > 0 {
		log.Printf("[AutoDiscovery-ConfigSkip] 渠道 %s: 用户已配置 SupportedModels（%d 项），不覆盖",
			channelUID, len(channel.SupportedModels))
		return
	}
	if len(channel.ModelMapping) > 0 {
		log.Printf("[AutoDiscovery-ConfigSkip] 渠道 %s: 用户已配置 ModelMapping（%d 项），不覆盖 SupportedModels",
			channelUID, len(channel.ModelMapping))
		return
	}

	// 通过 ConfigManager 更新 SupportedModels
	// 先从当前配置中找到该渠道的 index 和 kind
	cfg := cfgManager.GetConfig()
	index, kind := findChannelIndexAndKind(cfg, channelUID)
	if index < 0 || kind == "" {
		log.Printf("[AutoDiscovery-ConfigSkip] 渠道 %s: 在当前配置中未找到对应渠道，跳过写入", channelUID)
		return
	}

	// 排序后写入，确保结果稳定可读
	sorted := sortModels(consistentModels)

	update := config.UpstreamUpdate{
		SupportedModels: sorted,
	}

	_, err := updateChannelByKind(cfgManager, kind, index, update)
	if err != nil {
		log.Printf("[AutoDiscovery-ConfigWrite] 渠道 %s: 写入 SupportedModels 失败: %v", channelUID, err)
		return
	}

	log.Printf("[AutoDiscovery-ConfigWrite] 渠道 %s: 已自动写入 SupportedModels（%d 项模型）",
		channelUID, len(sorted))

	// Phase 3A：发布 auto_mapping_applied 事件（只读展示，不影响调度）
	if r.hub != nil {
		now := time.Now()
		ev := ProfileChangeEvent{
			ChannelUID:  channelUID,
			ChannelKind: kind,
			EventType:   EventTypeAutoMappingApply,
			Summary:     fmt.Sprintf("自动写入 SupportedModels（%d 项模型）", len(sorted)),
			CreatedAt:   now,
		}
		ev.EventUID = GenerateChangeEventUID(channelUID, ev.EventType, now)
		r.hub.Publish(ev)
	}
}

// modelsSetConsistent 检查所有 endpoint 的模型列表是否集合相等。
// 如果一致，返回任意一个端点的模型列表作为代表；如果不一致，返回 nil。
func modelsSetConsistent(endpoints []EndpointDiscoveryResult) []string {
	if len(endpoints) == 0 {
		return nil
	}

	// 将第一个端点的模型列表转为 set 作为基准
	baseSet := makeStringSet(endpoints[0].Models)

	for _, ep := range endpoints[1:] {
		candidateSet := makeStringSet(ep.Models)
		if !stringSetsEqual(baseSet, candidateSet) {
			return nil
		}
	}

	return endpoints[0].Models
}

// makeStringSet 将字符串列表转为 set（map[string]bool）。
func makeStringSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

// stringSetsEqual 判断两个 string set 是否完全相同。
func stringSetsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// sortModels 对模型列表排序，确保写入结果稳定可读。
func sortModels(models []string) []string {
	sorted := make([]string, len(models))
	copy(sorted, models)
	sort.Strings(sorted)
	return sorted
}

// findChannelIndexAndKind 在当前配置中根据 channelUID 找到该渠道的 index 和 kind（渠道类型）。
// 返回 (-1, "") 表示未找到。
func findChannelIndexAndKind(cfg config.Config, channelUID string) (int, string) {
	type sliceKind struct {
		channels []config.UpstreamConfig
		kind     string
	}
	slices := []sliceKind{
		{cfg.Upstream, "messages"},
		{cfg.ChatUpstream, "chat"},
		{cfg.ResponsesUpstream, "responses"},
		{cfg.GeminiUpstream, "gemini"},
		{cfg.ImagesUpstream, "images"},
		{cfg.VectorsUpstream, "vectors"},
	}
	for _, sk := range slices {
		for i, ch := range sk.channels {
			if ch.ChannelUID == channelUID {
				return i, sk.kind
			}
		}
	}
	return -1, ""
}

// updateChannelByKind 根据渠道类型调用对应的 ConfigManager 更新方法。
func updateChannelByKind(cfgManager *config.ConfigManager, kind string, index int, update config.UpstreamUpdate) (bool, error) {
	switch kind {
	case "messages":
		return cfgManager.UpdateUpstream(index, update)
	case "chat":
		return cfgManager.UpdateChatUpstream(index, update)
	case "responses":
		return cfgManager.UpdateResponsesUpstream(index, update)
	case "gemini":
		return cfgManager.UpdateGeminiUpstream(index, update)
	case "images":
		return cfgManager.UpdateImagesUpstream(index, update)
	case "vectors":
		return cfgManager.UpdateVectorsUpstream(index, update)
	default:
		return false, fmt.Errorf("不支持的渠道类型: %s", kind)
	}
}

// computeMetricsIdentityKey 内联计算 MetricsIdentityKey，
// 与 metrics.GenerateMetricsIdentityKey 逻辑完全一致，避免 autopilot → metrics 循环导入。
func computeMetricsIdentityKey(baseURL, apiKey, serviceType string) string {
	normalized := utils.MetricsIdentityBaseURL(baseURL, serviceType)
	h := sha256.New()
	h.Write([]byte(normalized + "|" + apiKey))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
