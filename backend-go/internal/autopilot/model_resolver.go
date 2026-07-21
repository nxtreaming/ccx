package autopilot

import (
	"fmt"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
)

// ── CapabilityFloor 能力下界 ──

// CapabilityFloor 描述请求对候选模型的能力要求。
// 上下文、推理、视觉和工具调用是硬约束；质量档是优先目标，
// 仅在没有满足目标质量的模型时允许降档兜底。
type CapabilityFloor struct {
	MinContextTokens int         // 最小上下文窗口（0=不限）
	NeedsReasoning   bool        // 必须支持推理
	NeedsVision      bool        // 必须支持视觉
	NeedsToolCalls   bool        // 必须支持工具调用
	MinQualityTier   QualityTier // 目标质量档（无同档候选时允许降档）
}

// BuildCapabilityFloorFromRequestProfile 从 RequestProfile 推导能力下界。
// 复用 RequestProfile 已有的 QualityNeed/ContextNeed/VisionNeed/ToolUseNeed/ReasoningNeed，
// 零额外计算。
func BuildCapabilityFloorFromRequestProfile(profile *RequestProfile) CapabilityFloor {
	if profile == nil {
		return CapabilityFloor{}
	}
	return CapabilityFloor{
		MinContextTokens: profile.ContextNeed,
		NeedsReasoning:   profile.ReasoningNeed,
		NeedsVision:      profile.VisionNeed,
		NeedsToolCalls:   profile.ToolUseNeed,
		MinQualityTier:   requestQualityTarget(profile),
	}
}

func requestQualityTarget(profile *RequestProfile) QualityTier {
	if profile == nil {
		return ""
	}
	if profile.QualityTarget != "" {
		return profile.QualityTarget
	}
	return ResolveQualityTarget(profile)
}

// ── ModelResolver 模型自动映射器 ──

// ModelResolver 实现设计 doc §5.4 的模型自动映射逻辑。
// 当请求模型在渠道 supportedModels 中不存在时，从 ModelProfileStore 中
// 找到满足 CapabilityFloor 的最佳匹配模型。
//
// 仅对 AutoManaged==true 的渠道生效；手动配置渠道通过 config.RedirectModel
// 直接短路，不经过自动映射。
type ModelResolver struct {
	profileStore *ModelProfileStore
	cfgManager   *config.ConfigManager
}

// NewModelResolver 创建 ModelResolver。
// profileStore 为 nil 时所有自动映射退化为 no-op（fail-open）。
func NewModelResolver(profileStore *ModelProfileStore, cfgManager *config.ConfigManager) *ModelResolver {
	return &ModelResolver{
		profileStore: profileStore,
		cfgManager:   cfgManager,
	}
}

// ResolveModel 将请求模型映射到渠道实际支持的最佳模型。
//
// 返回:
//   - mappedModel: 映射后的模型名（可能与 requestModel 相同）
//   - resolved: true 表示成功映射，false 表示该渠道无满足下界的模型
//   - reason: 决策原因（用于 trace / 日志）
//
// 安全不变量:
//   - 显式 modelMapping（用户手动配置）始终优先，不经过能力下界检查
//   - 禁止链式映射：candidate 源始终是原始 GetModelProfiles 结果
//   - 仅 autoManaged 渠道走自动映射；手动渠道由 config.RedirectModel 短路
//   - 只有 ModelRoutingPolicy 白名单入口允许跨模型替代；其余请求必须精确命中模型 ID
func (r *ModelResolver) ResolveModel(
	requestModel string,
	channelUID string,
	channelKind string,
	metricsKey string,
	floor CapabilityFloor,
) (mappedModel string, resolved bool, reason string) {

	// Step 1: 显式 modelMapping（精确 → 模糊）始终优先。
	// 手动配置视为已知正确，不经过能力下界检查（设计 doc 安全边界）。
	if r.cfgManager != nil {
		upstream := r.findUpstream(channelUID, channelKind)
		if upstream != nil && !upstream.AutoManaged {
			redirected, matched := config.RedirectModelWithMatch(requestModel, upstream)
			if matched && redirected != requestModel {
				return redirected, true, "manual_redirect"
			}
		}
	}

	// Step 2: 无 ModelProfileStore 时自动映射不可用，fail-open。
	if r.profileStore == nil {
		return requestModel, false, "model_profile_store_unavailable"
	}

	// Step 3: 查询候选模型画像。
	candidates := r.profileStore.GetModelProfiles(channelUID, channelKind, metricsKey)
	if len(candidates) == 0 {
		return requestModel, false, "no_model_profiles"
	}
	candidates = r.refreshAutoDiscoveryCapabilities(candidates, channelUID, channelKind)

	// Step 4: 能力过滤——上下文、推理、视觉、工具调用仍是硬约束；
	// 质量档作为首选条件，只有更高质量候选完全不存在时才允许降档，
	// 避免“没有 Opus 等价模型就整条请求不可用”。
	qualityFallback := false
	// CapabilityFloorEnabled=false 时跳过硬过滤（紧急逃生口，所有候选均可参与排序）。
	if r.cfgManager != nil {
		routingCfg := r.cfgManager.GetAutopilotRouting()
		if !routingCfg.ModelMapping.CapabilityFloorEnabled {
			// 仅过滤掉未验证的模型，不做能力下界检查。
			probeEligible := filterProbedModelProfiles(candidates)
			if len(probeEligible) == 0 {
				return requestModel, false, "no_probed_model"
			}
			candidates = probeEligible
		} else {
			candidates, qualityFallback = filterByCapabilityFloorWithQualityFallback(candidates, floor)
		}
	} else {
		candidates, qualityFallback = filterByCapabilityFloorWithQualityFallback(candidates, floor)
	}
	if len(candidates) == 0 {
		return requestModel, false, "no_capable_model"
	}

	// Step 5: 精确模型始终优先；非自适应入口不得跨模型替代。
	if exact, found := findExactModelProfile(candidates, requestModel); found {
		return exact.ModelID, true, modelResolutionReason("found_exact_model_in_profile", qualityFallback)
	}
	if equivalent, found := findEquivalentModelProfile(candidates, requestModel); found {
		return equivalent.ModelID, true, modelResolutionReason("found_equivalent_model_in_profile", qualityFallback)
	}
	intent := ClassifyModelRoutingIntent(channelKind, requestModel)
	if !intent.AllowsSubstitution() {
		return requestModel, false, "exact_model_required"
	}

	// Step 6: 自适应入口在满足下界的候选中选最佳匹配。
	best := rankBySimilarity(candidates, requestModel, floor)
	baseReason := fmt.Sprintf("mapped %s->%s (intent:%s, family:%s, quality:%s)",
		requestModel, best.ModelID, intent, best.ModelFamily, best.QualityTier)
	return best.ModelID, true, modelResolutionReason(baseReason, qualityFallback)
}

// ResolveModelAnyEndpoint 在渠道的所有 endpoint 中判断 requestModel 是否可由自动映射支持。
// 不限定 metricsKey，适用于调度器候选筛选阶段（此时无具体 API Key）。
// 精确命中已发现模型时直接返回该模型；未命中时从该渠道所有已探测成功模型中选一个
// request-scoped 候选，避免 autoManaged 渠道在进入 EndpointAttemptPolicy 前被 active_model_filter 误剔除。
// 真正发送请求前仍会用带 metricsKey 和完整 CapabilityFloor 的 ResolveModel 再做一次 endpoint 级决策。
func (r *ModelResolver) ResolveModelAnyEndpoint(
	requestModel string,
	channelUID string,
	channelKind string,
) (mappedModel string, found bool, reason string) {
	return r.resolveModelAnyEndpoint(requestModel, channelUID, channelKind, CapabilityFloor{})
}

// ResolveModelAnyEndpointWithFloor 在渠道所有 endpoint 中查找满足完整能力下界的映射。
// 该方法只读且不修改配置，可供 dry-run 诊断和 scheduler 首次候选过滤复用。
func (r *ModelResolver) ResolveModelAnyEndpointWithFloor(
	requestModel string,
	channelUID string,
	channelKind string,
	floor CapabilityFloor,
) (mappedModel string, found bool, reason string) {
	return r.resolveModelAnyEndpoint(requestModel, channelUID, channelKind, floor)
}

func (r *ModelResolver) resolveModelAnyEndpoint(
	requestModel string,
	channelUID string,
	channelKind string,
	floor CapabilityFloor,
) (mappedModel string, found bool, reason string) {
	if r.profileStore == nil {
		return requestModel, false, "model_profile_store_unavailable"
	}

	candidates := make([]ModelProfile, 0)
	all := r.profileStore.ListActiveByChannel(channelUID)
	for _, p := range all {
		if p.ChannelKind != channelKind {
			continue
		}
		if !p.ProbeSuccess {
			continue
		}
		candidates = append(candidates, p)
	}
	if len(candidates) == 0 {
		return requestModel, false, "no_probed_model_profiles"
	}
	candidates = r.refreshAutoDiscoveryCapabilities(candidates, channelUID, channelKind)

	qualityFallback := false
	if r.cfgManager != nil {
		routingCfg := r.cfgManager.GetAutopilotRouting()
		if routingCfg.ModelMapping.CapabilityFloorEnabled {
			candidates, qualityFallback = filterByCapabilityFloorWithQualityFallback(candidates, floor)
		}
	} else {
		candidates, qualityFallback = filterByCapabilityFloorWithQualityFallback(candidates, floor)
	}
	if len(candidates) == 0 {
		return requestModel, false, "no_capable_model"
	}
	if exact, found := findExactModelProfile(candidates, requestModel); found {
		return exact.ModelID, true, modelResolutionReason("found_exact_model_in_profile", qualityFallback)
	}
	if equivalent, found := findEquivalentModelProfile(candidates, requestModel); found {
		return equivalent.ModelID, true, modelResolutionReason("found_equivalent_model_in_profile", qualityFallback)
	}
	intent := ClassifyModelRoutingIntent(channelKind, requestModel)
	if !intent.AllowsSubstitution() {
		return requestModel, false, "exact_model_required"
	}

	best := rankBySimilarity(candidates, requestModel, floor)
	baseReason := fmt.Sprintf("mapped_any_endpoint %s->%s (intent:%s)",
		requestModel, best.ModelID, intent)
	return best.ModelID, true, modelResolutionReason(baseReason, qualityFallback)
}

// ── 过滤与排序 ──

// filterByCapabilityFloor 只保留满足所有能力下界约束的模型。
// 与 capability_floor.go 的 CapabilityFloorReasons 逻辑一致，
// 但作用于 ModelProfile（而非 CandidateCapabilities），并额外检查 QualityTier。
func filterByCapabilityFloor(profiles []ModelProfile, floor CapabilityFloor) []ModelProfile {
	return filterByCapabilityFloorInternal(profiles, floor, true)
}

// filterByCapabilityFloorWithoutQuality 保留所有真实能力约束，仅跳过质量档约束。
// 用于“高档候选不存在时”的用户体验兜底；不会放行上下文或工具能力不足的模型。
func filterByCapabilityFloorWithoutQuality(profiles []ModelProfile, floor CapabilityFloor) []ModelProfile {
	return filterByCapabilityFloorInternal(profiles, floor, false)
}

// filterByCapabilityFloorWithQualityFallback 先按完整能力目标筛选；若仅质量档
// 导致无候选，则保留所有真实能力硬约束并允许质量降档。
func filterByCapabilityFloorWithQualityFallback(profiles []ModelProfile, floor CapabilityFloor) ([]ModelProfile, bool) {
	eligible := filterByCapabilityFloor(profiles, floor)
	if len(eligible) > 0 || floor.MinQualityTier == "" {
		return eligible, false
	}
	fallback := filterByCapabilityFloorWithoutQuality(profiles, floor)
	return fallback, len(fallback) > 0
}

func filterProbedModelProfiles(profiles []ModelProfile) []ModelProfile {
	eligible := make([]ModelProfile, 0, len(profiles))
	for _, profile := range profiles {
		if profile.ProbeSuccess {
			eligible = append(eligible, profile)
		}
	}
	return eligible
}

func filterByCapabilityFloorInternal(profiles []ModelProfile, floor CapabilityFloor, enforceQuality bool) []ModelProfile {
	var eligible []ModelProfile
	for _, p := range profiles {
		// 未验证通过的模型不参与自动映射
		if !p.ProbeSuccess {
			continue
		}
		if p.ContextTokens < floor.MinContextTokens {
			continue
		}
		if floor.NeedsReasoning && !p.SupportsReasoning {
			continue
		}
		if floor.NeedsVision && !p.SupportsVision {
			continue
		}
		if floor.NeedsToolCalls && !p.SupportsToolCalls {
			continue
		}
		if enforceQuality && qualityTierRank(p.QualityTier) < qualityTierRank(floor.MinQualityTier) {
			continue
		}
		eligible = append(eligible, p)
	}
	return eligible
}

// modelResolutionReason 标记发生了质量降档，但不改变现有调用方的映射结果。
func modelResolutionReason(reason string, qualityFallback bool) string {
	if !qualityFallback {
		return reason
	}
	return "quality_fallback: " + reason
}

// rankBySimilarity 在满足下界的候选中选择最佳匹配。
//
// 匹配优先级（高→低）：
//  1. 与当前任务质量目标的档位距离最近
//  2. 同模型族（claude→claude, openai→openai）
//  3. 上下文窗口最接近请求下界（不浪费也不至于不够）
//  4. 探测延迟最低
func rankBySimilarity(eligible []ModelProfile, requestModel string, floor CapabilityFloor) ModelProfile {
	reqFamily := InferModelFamily(requestModel, "")
	hasQualityTarget := floor.MinQualityTier != ""
	reqTierRank := qualityTierRank(floor.MinQualityTier)

	type scored struct {
		profile         ModelProfile
		qualityDistance int
		sameFamily      bool
		latency         int64
		ctxDist         int
		candID          string
	}

	scoredList := make([]scored, 0, len(eligible))
	for _, p := range eligible {
		s := scored{
			profile:    p,
			sameFamily: p.ModelFamily == reqFamily,
			latency:    p.ProbeLatencyMs,
			candID:     strings.ToLower(p.ModelID),
		}
		if hasQualityTarget {
			s.qualityDistance = absInt(qualityTierRank(p.QualityTier) - reqTierRank)
		}

		// 上下文窗口距离：越接近下界越好（不浪费也不至于不够）。
		ctxDist := p.ContextTokens - floor.MinContextTokens
		if ctxDist < 0 {
			ctxDist = -ctxDist
		}
		s.ctxDist = ctxDist
		scoredList = append(scoredList, s)
	}

	// 排序：质量距离升序 → 同派系 → 上下文距离升序 → 延迟升序 → modelID 字典序。
	bestIdx := 0
	for i := 1; i < len(scoredList); i++ {
		a, b := scoredList[bestIdx], scoredList[i]
		if hasQualityTarget && b.qualityDistance < a.qualityDistance {
			bestIdx = i
		} else if (!hasQualityTarget || b.qualityDistance == a.qualityDistance) && b.sameFamily != a.sameFamily {
			if b.sameFamily {
				bestIdx = i
			}
		} else if (!hasQualityTarget || b.qualityDistance == a.qualityDistance) && b.sameFamily == a.sameFamily {
			if b.ctxDist < a.ctxDist {
				bestIdx = i
			} else if b.ctxDist == a.ctxDist {
				if b.latency < a.latency {
					bestIdx = i
				} else if b.latency == a.latency && b.candID < a.candID {
					bestIdx = i
				}
			}
		}
	}

	return scoredList[bestIdx].profile
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// refreshAutoDiscoveryCapabilities 兼容由旧版本写入的自动发现画像。
// 旧实现误用了下游 AgentModelProfile，可能把 GLM-5.2 等上游模型写成错误窗口和能力；
// 运行时以当前上游能力注册表重新派生，后续自动发现会把同样结果持久化。
func (r *ModelResolver) refreshAutoDiscoveryCapabilities(
	candidates []ModelProfile,
	channelUID string,
	channelKind string,
) []ModelProfile {
	if len(candidates) == 0 {
		return candidates
	}

	var upstream *config.UpstreamConfig
	var global map[string]config.UpstreamModelCapability
	if r.cfgManager != nil {
		cfg := r.cfgManager.GetConfig()
		global = cfg.UpstreamModelCapabilities
		upstream = r.findUpstream(channelUID, channelKind)
	}

	refreshed := append([]ModelProfile(nil), candidates...)
	for i := range refreshed {
		profile := &refreshed[i]
		if profile.Source != "auto_discovery" {
			continue
		}
		oldFamily := profile.ModelFamily
		oldQuality := profile.QualityTier
		oldContext := profile.ContextTokens
		oldVision := profile.SupportsVision
		oldTools := profile.SupportsToolCalls
		oldReasoning := profile.SupportsReasoning
		profile.ModelFamily = InferModelFamily(profile.ModelID, "")
		profile.QualityTier = ModelProfileQualityTierFromFamily(profile.ModelFamily, profile.ModelID)
		if resolved := config.ResolveUpstreamCapability(profile.ModelID, upstream, global); resolved.Known {
			applyUpstreamModelCapability(profile, resolved.Capability)
		}
		if oldFamily != profile.ModelFamily || oldQuality != profile.QualityTier ||
			oldContext != profile.ContextTokens || oldVision != profile.SupportsVision ||
			oldTools != profile.SupportsToolCalls || oldReasoning != profile.SupportsReasoning {
			profile.UpdatedAt = time.Now()
			_ = r.profileStore.Upsert(profile)
		}
	}
	return refreshed
}

// ── 辅助 ──

// findUpstream 根据 channelUID 和 channelKind 从 ConfigManager 查找对应的 UpstreamConfig。
// 遍历所有渠道类型列表，匹配 ChannelUID。
// 返回 nil 表示未找到（渠道已删除或 UID 不匹配）。
func (r *ModelResolver) findUpstream(channelUID, channelKind string) *config.UpstreamConfig {
	if r.cfgManager == nil || channelUID == "" {
		return nil
	}
	cfg := r.cfgManager.GetConfig()

	type upstreamList struct {
		channels []config.UpstreamConfig
		kind     string
	}
	lists := []upstreamList{
		{cfg.Upstream, "messages"},
		{cfg.ResponsesUpstream, "responses"},
		{cfg.GeminiUpstream, "gemini"},
		{cfg.ChatUpstream, "chat"},
		{cfg.ImagesUpstream, "images"},
		{cfg.VectorsUpstream, "vectors"},
	}

	for _, ul := range lists {
		if ul.kind != channelKind {
			continue
		}
		for i := range ul.channels {
			if ul.channels[i].ChannelUID == channelUID {
				return &ul.channels[i]
			}
		}
	}
	return nil
}
