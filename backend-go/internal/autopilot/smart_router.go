package autopilot

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/scheduler"
)

// ── SmartRouter（设计 §4.6 + §4.6.3 + §4.6.5 + P0.4 + P0.5）──

// RoutingPlan 一次请求的路由计划（§4.6.1）。
type RoutingPlan struct {
	RequestProfile *RequestProfile   `json:"requestProfile"`
	Candidates     []ScoredCandidate `json:"candidates"`
	SortReasons    []string          `json:"sortReasons,omitempty"`
	Mode           RoutingMode       `json:"mode"`
	Weights        ScoringWeights    `json:"weights"`
}

// SmartRouter 根据请求画像 + 渠道画像生成路由计划。
// shadow 模式：只计算 + 记录 RoutingDecisionTrace，不影响真实调度。
// active 模式：返回评分排序后的候选列表，改变调度顺序。
// off / kill switch：不注入 CandidateFilter，调度链路不变。
type SmartRouter struct {
	profileStore  *ProfileStore
	intentStore   *ManualIntentStore
	traceStore    *TraceStore
	configManager *config.ConfigManager
	mu            sync.RWMutex
}

// NewSmartRouter 创建 SmartRouter 实例。
func NewSmartRouter(
	profileStore *ProfileStore,
	intentStore *ManualIntentStore,
	traceStore *TraceStore,
	configManager *config.ConfigManager,
) *SmartRouter {
	return &SmartRouter{
		profileStore:  profileStore,
		intentStore:   intentStore,
		traceStore:    traceStore,
		configManager: configManager,
	}
}

// ConfigManager 返回内部 ConfigManager 引用。
func (r *SmartRouter) ConfigManager() *config.ConfigManager {
	return r.configManager
}

// TraceStore 返回内部 TraceStore 引用。
func (r *SmartRouter) TraceStore() *TraceStore {
	return r.traceStore
}

// ProfileStore 返回内部 ProfileStore 引用。
func (r *SmartRouter) ProfileStore() *ProfileStore {
	return r.profileStore
}

// IntentStore 返回内部 ManualIntentStore 引用。
func (r *SmartRouter) IntentStore() *ManualIntentStore {
	return r.intentStore
}

// BuildPlan 为请求构建路由计划（§4.6.1）。
// 用于 dry-run API 和诊断，不影响真实调度。
func (r *SmartRouter) BuildPlan(profile *RequestProfile) *RoutingPlan {
	if profile == nil {
		return &RoutingPlan{Mode: RoutingModeDryRun}
	}

	// 确定分类
	input := BuildClassifierInput(profile)
	ClassifyAndFill(profile, input)

	cfg := r.configManager.GetConfig()
	autopilotCfg := cfg.AutopilotRouting

	// 获取权重
	weights := DefaultTaskWeights()[profile.TaskClass]
	weights = ApplyCostPreference(weights, CostPreferenceMode(autopilotCfg.CostPreference.Mode))

	// 覆盖权重
	for k, v := range autopilotCfg.WeightOverrides {
		switch k {
		case "wQuality":
			weights.WQuality = v
		case "wStability":
			weights.WStability = v
		case "wSpeed":
			weights.WSpeed = v
		case "wCost":
			weights.WCost = v
		case "wSavings":
			weights.WSavings = v
		case "wTierMatch":
			weights.WTierMatch = v
		case "wFamily":
			weights.WFamily = v
		case "wProviderQuality":
			weights.WProviderQuality = v
		case "wDomain":
			weights.WDomain = v
		}
	}

	familyPrefs := r.loadFamilyPrefs(autopilotCfg.ModelFamilyPreference)

	// 收集并评分候选
	entries := r.collectChannelEntries(profile.ChannelKind, profile.Model)
	if len(entries) == 0 {
		return &RoutingPlan{
			RequestProfile: profile,
			Candidates:     nil,
			Mode:           RoutingModeDryRun,
			Weights:        weights,
		}
	}

	costs := make(map[string]float64, len(entries))
	for _, e := range entries {
		costs[e.ChannelUID] = e.EstimatedCost
	}
	savingsMap := NormalizeSavingsScore(costs)

	ctx := ScoringContext{
		TaskClass:   profile.TaskClass,
		TaskDomain:  profile.TaskDomain,
		FamilyPrefs: familyPrefs,
		Weights:     weights,
	}

	candidates := make([]ScoredCandidate, 0, len(entries))
	for _, e := range entries {
		e.ScoringCandidate.SavingsScore = savingsMap[e.ChannelUID]
		scored := ScoreCandidate(e.ScoringCandidate, ctx)
		candidates = append(candidates, scored)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return &RoutingPlan{
		RequestProfile: profile,
		Candidates:     candidates,
		SortReasons:    []string{"smart_routing_dryrun"},
		Mode:           RoutingModeDryRun,
		Weights:        weights,
	}
}

// CandidateFilterFor 为给定请求构建 scheduler.CandidateFilterFunc。
// 返回 nil 表示不注入（off / kill switch）。
// shadow 模式：计算评分 + 记录 RoutingDecisionTrace，返回原始候选列表。
// assist 模式：按评分重排渠道列表，不删除任何渠道。
// auto 模式：硬约束过滤 + 重排；过滤后为空则 fail-open 回退到只重排。
// active 模式：返回评分排序后的候选列表。
func (r *SmartRouter) CandidateFilterFor(profile *RequestProfile) scheduler.CandidateFilterFunc {
	if profile == nil {
		return nil
	}

	cfg := r.configManager.GetConfig()
	autopilotCfg := cfg.AutopilotRouting
	mode := autopilotCfg.EffectiveRoutingMode()

	// off / kill switch 不注入
	if mode == config.AutopilotModeOff {
		return nil
	}

	// 确定分类
	input := BuildClassifierInput(profile)
	ClassifyAndFill(profile, input)

	// 获取权重
	weights := DefaultTaskWeights()[profile.TaskClass]
	weights = ApplyCostPreference(weights, CostPreferenceMode(autopilotCfg.CostPreference.Mode))
	for k, v := range autopilotCfg.WeightOverrides {
		switch k {
		case "wQuality":
			weights.WQuality = v
		case "wStability":
			weights.WStability = v
		case "wSpeed":
			weights.WSpeed = v
		case "wCost":
			weights.WCost = v
		case "wSavings":
			weights.WSavings = v
		case "wTierMatch":
			weights.WTierMatch = v
		case "wFamily":
			weights.WFamily = v
		case "wProviderQuality":
			weights.WProviderQuality = v
		case "wDomain":
			weights.WDomain = v
		}
	}

	familyPrefs := r.loadFamilyPrefs(autopilotCfg.ModelFamilyPreference)

	traceStore := r.traceStore
	routerMode := RoutingMode(mode)

	return func(
		channels []scheduler.ChannelInfo,
		upstreamFor func(scheduler.ChannelInfo) *config.UpstreamConfig,
		candidateAvailable func(scheduler.ChannelInfo, *config.UpstreamConfig) bool,
	) ([]scheduler.ChannelInfo, error) {
		return r.executeFilter(
			channels, upstreamFor, candidateAvailable,
			profile, weights, familyPrefs, routerMode, traceStore,
		)
	}
}

// executeFilter 执行 SmartRouter 过滤逻辑。
func (r *SmartRouter) executeFilter(
	channels []scheduler.ChannelInfo,
	upstreamFor func(scheduler.ChannelInfo) *config.UpstreamConfig,
	candidateAvailable func(scheduler.ChannelInfo, *config.UpstreamConfig) bool,
	profile *RequestProfile,
	weights ScoringWeights,
	familyPrefs []ModelFamily,
	mode RoutingMode,
	traceStore *TraceStore,
) ([]scheduler.ChannelInfo, error) {
	startTime := time.Now()

	// 构建 RoutingDecisionTrace
	trace := &RoutingDecisionTrace{
		RequestKind:         profile.ChannelKind,
		TaskClass:           profile.TaskClass,
		TaskDomain:          profile.TaskDomain,
		RequestedModel:      profile.Model,
		AgentRole:           profile.AgentRole,
		Mode:                mode,
		CandidatesBefore:    len(channels),
		GlobalFilterReasons: make(map[string][]string),
	}

	// 构建评分上下文
	scoringCtx := ScoringContext{
		TaskClass:   profile.TaskClass,
		TaskDomain:  profile.TaskDomain,
		FamilyPrefs: familyPrefs,
		Weights:     weights,
	}

	// 收集所有候选的估算成本用于归一化
	costMap := make(map[string]float64, len(channels))
	entries := make([]channelScoreEntry, 0, len(channels))
	for _, ch := range channels {
		upstream := upstreamFor(ch)
		if upstream == nil || !candidateAvailable(ch, upstream) {
			continue
		}
		entry := r.buildChannelEntry(ch, upstream)
		entries = append(entries, entry)
		costMap[entry.ChannelUID] = entry.EstimatedCost
	}
	savingsMap := NormalizeSavingsScore(costMap)

	// 评分 + 按总分降序排序
	type scoredEntry struct {
		entry  channelScoreEntry
		scored ScoredCandidate
	}
	scoredEntries := make([]scoredEntry, 0, len(entries))
	for _, e := range entries {
		e.ScoringCandidate.SavingsScore = savingsMap[e.ChannelUID]
		scored := ScoreCandidate(e.ScoringCandidate, scoringCtx)
		scoredEntries = append(scoredEntries, scoredEntry{entry: e, scored: scored})
	}
	sort.Slice(scoredEntries, func(i, j int) bool {
		return scoredEntries[i].scored.Score > scoredEntries[j].scored.Score
	})

	// 从已排序的 scoredEntries 构建 trace 候选和结果列表
	result := make([]scheduler.ChannelInfo, 0, len(scoredEntries))
	candidates := make([]RoutingCandidate, 0, len(scoredEntries))
	for _, se := range scoredEntries {
		e := se.entry
		sc := se.scored
		candidates = append(candidates, RoutingCandidate{
			ChannelUID:  e.ChannelUID,
			MetricsKey:  SanitizeMetricsKey(e.MetricsKey),
			OriginTier:  string(e.OriginTier),
			ChannelKind: e.ChannelKind,
			HealthState: string(e.HealthState),
			TotalScore:  sc.Score,
			Scores: []CandidateScore{
				{Dimension: "quality", Score: sc.QualityScore, Weight: weights.WQuality},
				{Dimension: "stability", Score: sc.StabilityScore, Weight: weights.WStability},
				{Dimension: "speed", Score: sc.SpeedScore, Weight: weights.WSpeed},
				{Dimension: "cost", Score: sc.CostScore, Weight: weights.WCost},
				{Dimension: "savings", Score: sc.SavingsScore, Weight: weights.WSavings},
				{Dimension: "family", Score: sc.FamilyPrefScore, Weight: weights.WFamily},
				{Dimension: "provider_quality", Score: sc.ProviderQualityScore, Weight: weights.WProviderQuality},
				{Dimension: "domain", Score: sc.DomainStrengthScore, Weight: weights.WDomain},
			},
			Selected: true,
		})

		// 匹配回 ChannelInfo：优先用上游配置的 ChannelUID，回退到 ch_%d 格式
		for _, ch := range channels {
			upstream := upstreamFor(ch)
			matchUID := e.ChannelUID
			if upstream != nil && upstream.ChannelUID != "" {
				matchUID = upstream.ChannelUID
			} else {
				matchUID = fmt.Sprintf("ch_%d", ch.Index)
			}
			if matchUID == e.ChannelUID {
				result = append(result, ch)
				break
			}
		}
	}

	// ── auto 模式：硬约束过滤 + fail-open ──
	fallbackUsed := false
	if mode == RoutingModeAuto {
		filteredResult := make([]scheduler.ChannelInfo, 0, len(scoredEntries))
		for i, se := range scoredEntries {
			reasons := autoHardConstraintReasons(profile, &se.entry)
			if len(reasons) > 0 {
				candidates[i].Selected = false
				candidates[i].FilterReasons = reasons
				trace.GlobalFilterReasons["auto_hard_constraints"] = append(
					trace.GlobalFilterReasons["auto_hard_constraints"],
					se.entry.ChannelUID+": "+joinReasons(reasons),
				)
			} else {
				// 保留未被过滤的渠道：匹配回 ChannelInfo
				for _, ch := range channels {
					upstream := upstreamFor(ch)
					matchUID := se.entry.ChannelUID
					if upstream != nil && upstream.ChannelUID != "" {
						matchUID = upstream.ChannelUID
					} else {
						matchUID = fmt.Sprintf("ch_%d", ch.Index)
					}
					if matchUID == se.entry.ChannelUID {
						filteredResult = append(filteredResult, ch)
						break
					}
				}
			}
		}

		if len(filteredResult) > 0 {
			result = filteredResult
		} else if len(scoredEntries) > 0 {
			// fail-open：全部被过滤时回退到重排（不删除）
			fallbackUsed = true
			trace.FallbackUsed = true
			trace.GlobalFilterReasons["auto_failopen"] = []string{
				fmt.Sprintf("所有 %d 个候选均被硬约束过滤，回退到重排模式", len(scoredEntries)),
			}
			log.Printf("[SmartRouter-AutoFailOpen] taskClass=%s 全部候选被过滤，回退到重排", string(profile.TaskClass))
			// result 保持重排后的完整列表
		}
	}

	// 记录 trace 信息
	trace.Candidates = candidates
	trace.CandidatesAfter = len(candidates)
	trace.SortReasons = []string{"smart_routing_score"}
	if mode == RoutingModeAssist {
		trace.SortReasons = append(trace.SortReasons, "assist_reorder")
	} else if mode == RoutingModeAuto {
		if fallbackUsed {
			trace.SortReasons = append(trace.SortReasons, "auto_failopen_reorder")
		} else {
			trace.SortReasons = append(trace.SortReasons, "auto_filter_and_reorder")
		}
	}

	if len(candidates) > 0 {
		trace.SelectedChannelUID = candidates[0].ChannelUID
		trace.SelectedMetricsKey = candidates[0].MetricsKey
		trace.SelectedOriginTier = candidates[0].OriginTier
	}

	// 计算耗时
	trace.DurationMs = time.Since(startTime).Milliseconds()
	trace.CreatedAt = time.Now().UTC()

	// shadow/dryrun 模式：记录 shadow 建议的渠道
	if mode == RoutingModeShadow && len(candidates) > 0 {
		trace.ShadowChannelUID = candidates[0].ChannelUID
		trace.Match = true // 先假设匹配，实际填充时更新
	}

	// 持久化 trace
	if traceStore != nil {
		traceStore.Record(trace)
	}

	log.Printf("[SmartRouter-Filter] taskClass=%s mode=%s candidates=%d fallback=%v shadow=%s duration=%dms",
		string(profile.TaskClass), string(mode), len(candidates), fallbackUsed,
		trace.ShadowChannelUID, trace.DurationMs)

	// shadow/dryrun 模式：返回原始候选列表（不影响真实调度）
	if mode == RoutingModeShadow || mode == RoutingModeDryRun {
		return channels, nil
	}

	// assist/auto 模式：返回评分重排后的候选列表
	return result, nil
}

// channelScoreEntry 渠道评分输入条目。
type channelScoreEntry struct {
	ChannelUID       string
	ChannelKind      string
	MetricsKey       string
	OriginTier       ChannelOriginTier
	HealthState      HealthState
	EstimatedCost    float64
	ChannelIndex     int
	SupportsVision   bool // 渠道是否支持识图（来自画像聚合 + 手动配置覆盖）
	ScoringCandidate ScoringCandidate
}

// buildChannelEntry 从 ChannelInfo + UpstreamConfig 构建评分输入。
// 无画像时使用中性默认值（不惩罚）。
func (r *SmartRouter) buildChannelEntry(
	ch scheduler.ChannelInfo,
	upstream *config.UpstreamConfig,
) channelScoreEntry {
	channelUID := upstream.ChannelUID
	if channelUID == "" {
		channelUID = fmt.Sprintf("ch_%d", ch.Index)
	}
	entry := channelScoreEntry{
		ChannelUID:   channelUID,
		ChannelKind:  string(scheduler.ChannelKindMessages),
		ChannelIndex: ch.Index,
		HealthState:  HealthStateUnknown,
		OriginTier:   OriginTierUnknown, // 无画像时默认 unknown
	}

	// 从 ProfileStore 读取画像
	if r.profileStore != nil {
		profiles := r.profileStore.ListByChannel(channelUID)
		if len(profiles) > 0 {
			// 解引用指针切片
			profileValues := make([]KeyEndpointProfile, len(profiles))
			for i, p := range profiles {
				profileValues[i] = *p
			}
			agg := AggregateChannelProfile(channelUID, ch.Index, entry.ChannelKind, profileValues)
			entry.HealthState = agg.HealthState
			entry.OriginTier = ChannelOriginTier(agg.OriginTier)
			entry.MetricsKey = profiles[0].MetricsKey

			// 识图能力：画像聚合（Layer 1）+ 手动配置覆盖（Layer 0 优先级最高）
			entry.SupportsVision = agg.SupportsVision
			if upstream.NoVision {
				entry.SupportsVision = false
			}

			entry.ScoringCandidate = ScoringCandidate{
				ChannelUID:                channelUID,
				QualityTier:               agg.QualityTier,
				StabilityTier:             agg.StabilityTier,
				SpeedTier:                 agg.SpeedTier,
				CostTier:                  agg.CostTier,
				HealthState:               agg.HealthState,
				ProviderQualityScore:      0.5,
				ProviderQualityConfidence: 0.3,
				SavingsScore:              0.5,
				DomainStrengthScore:       0.5,
			}
			return entry
		}
	}

	// 无画像：中性默认值，不惩罚
	entry.ScoringCandidate = ScoringCandidate{
		ChannelUID:                channelUID,
		QualityTier:               QualityTierNormal,
		StabilityTier:             StabilityTierNormal,
		SpeedTier:                 SpeedTierNormal,
		CostTier:                  CostTierNormal,
		HealthState:               HealthStateUnknown,
		ProviderQualityScore:      0.5,
		ProviderQualityConfidence: 0.3,
		SavingsScore:              0.5,
		DomainStrengthScore:       0.5,
	}
	return entry
}

// autoHardConstraintReasons 检查 auto 模式的硬约束，返回不满足的原因列表。
// 空列表表示该渠道满足所有硬约束。
// 当前硬约束（逐批扩展）：
//   - vision 请求但渠道不支持识图
//   - CapabilityFloor：请求需要推理但渠道不支持（画像数据可用时）
func autoHardConstraintReasons(profile *RequestProfile, entry *channelScoreEntry) []string {
	var reasons []string

	// 识图硬约束
	if profile.VisionNeed && !entry.SupportsVision {
		reasons = append(reasons, "vision_unsupported")
	}

	// 模型级 NoVisionModels 精确匹配
	if profile.VisionNeed && entry.SupportsVision && profile.Model != "" {
		// 如果上游配置指定了此模型不支持视觉，通过 ProfileStore 的画像已覆盖
		// 此处不重复检查（画像层已处理）
	}

	// TODO(P2-后续): CapabilityFloor 扩展 —— 当模型注册表可用时检查
	//   - ContextNeed vs 上游 ContextWindowLimit
	//   - ToolUseNeed vs 上游 SupportsToolCalls
	//   - ReasoningNeed vs 上游 SupportsReasoning

	return reasons
}

// joinReasons 将原因列表拼接为逗号分隔字符串。
func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	result := reasons[0]
	for _, r := range reasons[1:] {
		result += "," + r
	}
	return result
}

// collectChannelEntries 收集指定 kind 的所有渠道条目。
// 仅用于 dry-run API（需要知道候选列表，但不依赖 scheduler）。
func (r *SmartRouter) collectChannelEntries(channelKind, model string) []channelScoreEntry {
	cfg := r.configManager.GetConfig()
	var upstreams []config.UpstreamConfig
	switch channelKind {
	case "responses":
		upstreams = cfg.ResponsesUpstream
	case "gemini":
		upstreams = cfg.GeminiUpstream
	case "chat":
		upstreams = cfg.ChatUpstream
	case "images":
		upstreams = cfg.ImagesUpstream
	case "vectors":
		upstreams = cfg.VectorsUpstream
	default:
		upstreams = cfg.Upstream
	}

	entries := make([]channelScoreEntry, 0, len(upstreams))
	for i, upstream := range upstreams {
		status := upstream.Status
		if status == "" {
			status = "active"
		}
		if status == "disabled" {
			continue
		}
		// 模型过滤
		if model != "" {
			if supported, _ := upstream.ExplainModelSupport(model); !supported {
				continue
			}
		}
		ch := scheduler.ChannelInfo{
			Index:    i,
			Name:     upstream.Name,
			Priority: upstream.Priority,
			Status:   status,
		}
		entry := r.buildChannelEntry(ch, &upstream)
		entry.ChannelKind = channelKind
		entries = append(entries, entry)
	}
	return entries
}

// loadFamilyPrefs 从配置加载派系偏好。
func (r *SmartRouter) loadFamilyPrefs(cfg config.ModelFamilyPreferenceConfig) []ModelFamily {
	if !cfg.Enabled || len(cfg.GlobalOrder) == 0 {
		return nil
	}
	prefs := make([]ModelFamily, 0, len(cfg.GlobalOrder))
	for _, f := range cfg.GlobalOrder {
		prefs = append(prefs, ModelFamily(f))
	}
	return prefs
}

// UpdateActualChannel 供调度完成后回调：用实际调度结果更新最近一条 shadow trace。
// shadow trace 与实际一致时 Match=true，否则 Match=false。
func (r *SmartRouter) UpdateActualChannel(actualChannelUID string) {
	if r.traceStore == nil || actualChannelUID == "" {
		return
	}

	r.traceStore.mu.RLock()
	defer r.traceStore.mu.RUnlock()

	total := len(r.traceStore.records)
	if total == 0 {
		return
	}

	// 取最近一条 shadow trace
	latest := r.traceStore.records[total-1]
	if latest.Mode != RoutingModeShadow {
		return
	}

	latest.ActualChannelUID = actualChannelUID
	latest.Match = latest.ShadowChannelUID == actualChannelUID

	// mismatch 样本强制落盘
	if !latest.Match && r.traceStore.db != nil {
		if err := r.traceStore.persistTrace(latest); err != nil {
			log.Printf("[SmartRouter-Update] 警告: mismatch 样本落盘失败: %v", err)
		}
	}
}
