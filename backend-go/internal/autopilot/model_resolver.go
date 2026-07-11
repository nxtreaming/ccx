package autopilot

import (
	"fmt"
	"strings"

	"github.com/BenedictKing/ccx/internal/config"
)

// ── CapabilityFloor 能力下界 ──

// CapabilityFloor 描述请求对候选模型的最低能力要求。
// 由 RequestProfile 推导，用于 ModelResolver 的硬过滤阶段。
// 低于任一维度的候选模型将被剔除。
type CapabilityFloor struct {
	MinContextTokens int         // 最小上下文窗口（0=不限）
	NeedsReasoning   bool        // 必须支持推理
	NeedsVision      bool        // 必须支持视觉
	NeedsToolCalls   bool        // 必须支持工具调用
	MinQualityTier   QualityTier // 最低质量档
}

// BuildCapabilityFloorFromRequestProfile 从 RequestProfile 推导能力下界。
// 复用 RequestProfile 已有的 QualityNeed/ContextNeed/VisionNeed/ToolUseNeed/ReasoningNeed，
// 零额外计算。
func BuildCapabilityFloorFromRequestProfile(profile *RequestProfile) CapabilityFloor {
	return CapabilityFloor{
		MinContextTokens: profile.ContextNeed,
		NeedsReasoning:   profile.ReasoningNeed,
		NeedsVision:      profile.VisionNeed,
		NeedsToolCalls:   profile.ToolUseNeed,
		MinQualityTier:   profile.QualityNeed,
	}
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

	// Step 4: 硬过滤——只保留满足能力下界的模型。
	// CapabilityFloorEnabled=false 时跳过硬过滤（紧急逃生口，所有候选均可参与排序）。
	if r.cfgManager != nil {
		routingCfg := r.cfgManager.GetAutopilotRouting()
		if !routingCfg.ModelMapping.CapabilityFloorEnabled {
			// 仅过滤掉未验证的模型，不做能力下界检查
			var probeEligible []ModelProfile
			for _, p := range candidates {
				if p.ProbeSuccess {
					probeEligible = append(probeEligible, p)
				}
			}
			if len(probeEligible) == 0 {
				return requestModel, false, "no_probed_model"
			}
			candidates = probeEligible
		} else {
			candidates = filterByCapabilityFloor(candidates, floor)
		}
	} else {
		candidates = filterByCapabilityFloor(candidates, floor)
	}
	if len(candidates) == 0 {
		return requestModel, false, "no_capable_model"
	}

	// Step 5: 在满足下界的候选中选最佳匹配。
	best := rankBySimilarity(candidates, requestModel, floor)
	return best.ModelID, true, fmt.Sprintf("mapped %s->%s (family:%s, quality:%s)",
		requestModel, best.ModelID, best.ModelFamily, best.QualityTier)
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
	if r.profileStore == nil {
		return requestModel, false, "model_profile_store_unavailable"
	}

	candidates := make([]ModelProfile, 0)
	all := r.profileStore.ListByChannel(channelUID)
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

	for _, p := range candidates {
		if strings.EqualFold(p.ModelID, requestModel) {
			return p.ModelID, true, fmt.Sprintf("found_model_in_profile (endpoint=%s)", p.MetricsKey)
		}
	}

	floor := CapabilityFloor{}
	if r.cfgManager != nil {
		routingCfg := r.cfgManager.GetAutopilotRouting()
		if routingCfg.ModelMapping.CapabilityFloorEnabled {
			candidates = filterByCapabilityFloor(candidates, floor)
		}
	} else {
		candidates = filterByCapabilityFloor(candidates, floor)
	}
	if len(candidates) == 0 {
		return requestModel, false, "no_capable_model"
	}

	best := rankBySimilarity(candidates, requestModel, floor)
	return best.ModelID, true, fmt.Sprintf("mapped_any_endpoint %s->%s", requestModel, best.ModelID)
}

// ── 过滤与排序 ──

// filterByCapabilityFloor 只保留满足所有能力下界约束的模型。
// 与 capability_floor.go 的 CapabilityFloorReasons 逻辑一致，
// 但作用于 ModelProfile（而非 CandidateCapabilities），并额外检查 QualityTier。
func filterByCapabilityFloor(profiles []ModelProfile, floor CapabilityFloor) []ModelProfile {
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
		if qualityTierRank(p.QualityTier) < qualityTierRank(floor.MinQualityTier) {
			continue
		}
		eligible = append(eligible, p)
	}
	return eligible
}

// rankBySimilarity 在满足下界的候选中选择最佳匹配。
//
// 匹配优先级（高→低）：
//  1. 同模型族（claude→claude, openai→openai）
//  2. 同质量档（premium→premium, high→high）
//  3. 上下文窗口最接近请求下界（不浪费也不至于不够）
//  4. 探测延迟最低
func rankBySimilarity(eligible []ModelProfile, requestModel string, floor CapabilityFloor) ModelProfile {
	reqFamily := InferModelFamily(requestModel, "")
	reqTierRank := qualityTierRank(floor.MinQualityTier)

	type scored struct {
		profile ModelProfile
		score   int // 越大越好
		latency int64
		ctxDist int
		candID  string
	}

	scoredList := make([]scored, 0, len(eligible))
	for _, p := range eligible {
		s := scored{profile: p, latency: p.ProbeLatencyMs, candID: strings.ToLower(p.ModelID)}

		// 派系匹配：+1000
		if p.ModelFamily == reqFamily {
			s.score += 1000
		}

		// 质量档匹配：+100
		if qualityTierRank(p.QualityTier) == reqTierRank {
			s.score += 100
		}

		// 上下文窗口距离：越接近下界越好（归一化到 0-100 分）
		ctxDist := p.ContextTokens - floor.MinContextTokens
		if ctxDist < 0 {
			ctxDist = -ctxDist
		}
		s.ctxDist = ctxDist
		if floor.MinContextTokens > 0 && ctxDist <= floor.MinContextTokens/10 {
			s.score += 50 // 差距在 10% 以内加分
		}

		scoredList = append(scoredList, s)
	}

	// 排序：score 降序 → latency 升序 → ctxDist 升序 → modelID 字典序
	bestIdx := 0
	for i := 1; i < len(scoredList); i++ {
		a, b := scoredList[bestIdx], scoredList[i]
		if b.score > a.score {
			bestIdx = i
		} else if b.score == a.score {
			if b.latency < a.latency {
				bestIdx = i
			} else if b.latency == a.latency {
				if b.ctxDist < a.ctxDist {
					bestIdx = i
				} else if b.ctxDist == a.ctxDist && b.candID < a.candID {
					bestIdx = i
				}
			}
		}
	}

	return scoredList[bestIdx].profile
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
