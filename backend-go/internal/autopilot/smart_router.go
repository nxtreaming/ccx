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
	profileStore      *ProfileStore
	intentStore       *ManualIntentStore
	traceStore        *TraceStore
	configManager     *config.ConfigManager
	advisor           *TrustedRoutingAdvisor      // Phase 2: 可信路由顾问（nil = 不启用）
	decisionStore     *AdvisorDecisionStore        // Phase 2: advisor 决策记录存储
	localRuntimeStore *LocalRuntimeStore           // Phase 2: 本地运行时存储（nil = 不纳入本地候选）

	// onCandidatesRanked Phase 4 Item 8: 候选排名回调（A/B 测试用）。
	// executeFilter 完成评分排序后调用，传入 ranked candidates。
	onCandidatesRanked func(model, channelKind string, candidates []RoutingCandidate)

	mu sync.RWMutex
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

// SetAdvisor 设置 TrustedRoutingAdvisor 和 AdvisorDecisionStore（由 main.go 在构造后调用）。
// nil 参数表示不启用对应功能（fail-safe：不影响调度）。
func (r *SmartRouter) SetAdvisor(advisor *TrustedRoutingAdvisor, decisionStore *AdvisorDecisionStore) {
	r.advisor = advisor
	r.decisionStore = decisionStore
}

// SetLocalRuntimeStore 设置 LocalRuntimeStore（由 main.go 在构造后调用）。
// nil 表示不纳入本地候选（fail-safe：不影响调度）。
func (r *SmartRouter) SetLocalRuntimeStore(store *LocalRuntimeStore) {
	r.localRuntimeStore = store
}

// SetOnCandidatesRanked 设置候选排名回调（Phase 4 Item 8: A/B 测试用）。
// executeFilter 完成评分排序后调用，将排名结果传递给调用方。
func (r *SmartRouter) SetOnCandidatesRanked(fn func(model, channelKind string, candidates []RoutingCandidate)) {
	r.onCandidatesRanked = fn
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

	// P1.5：按 channel 禁用——与 executeFilter（真实路由路径）保持一致。
	// 注意：这里只过滤"禁用渠道"这个硬约束，不像 kill switch/mode==off 那样
	// 让 BuildPlan 提前返回——BuildPlan 是诊断预览接口，即使 SmartRouter
	// 处于 off/kill switch 也要能算出"如果启用会怎样"（见 P0.5 不变量测试），
	// DisabledTaskClasses 属于同一类"是否运行"的开关，因此不在此处短路；
	// DisabledChannelUIDs 则是候选集合本身的硬约束，必须和真实路径一致。
	if disabledChannelUIDs := toStringSet(autopilotCfg.DisabledChannelUIDs); len(disabledChannelUIDs) > 0 {
		filtered := entries[:0:0]
		for _, e := range entries {
			if disabledChannelUIDs[e.ChannelUID] {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered
	}

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

	// OriginTier tie-breaker：与 executeFilter（真实路由路径）保持一致，
	// 否则 dry-run 预览结果会和实际调度在同分情况下产生不同的候选顺序。
	if len(candidates) > 1 {
		originTiers := make(map[string]ChannelOriginTier, len(entries))
		for _, e := range entries {
			originTiers[e.ChannelUID] = e.OriginTier
		}
		candidates = BreakTieByOriginTier(candidates, originTiers)
	}

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

	// P1.5：按 task class 禁用——命中时 SmartRouter 对本次请求完全不介入，
	// 与 kill switch 的 "return nil" 语义完全一致，只是作用范围缩小到单个 TaskClass。
	if isTaskClassDisabled(autopilotCfg.DisabledTaskClasses, profile.TaskClass) {
		return nil
	}

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
	disabledChannelUIDs := toStringSet(autopilotCfg.DisabledChannelUIDs)

	return func(
		channels []scheduler.ChannelInfo,
		upstreamFor func(scheduler.ChannelInfo) *config.UpstreamConfig,
		candidateAvailable func(scheduler.ChannelInfo, *config.UpstreamConfig) bool,
	) ([]scheduler.ChannelInfo, error) {
		return r.executeFilter(
			channels, upstreamFor, candidateAvailable,
			profile, weights, familyPrefs, routerMode, traceStore, disabledChannelUIDs,
		)
	}
}

// isTaskClassDisabled 判断 taskClass 是否命中禁用名单。
func isTaskClassDisabled(disabled []string, taskClass TaskClass) bool {
	for _, d := range disabled {
		if TaskClass(d) == taskClass {
			return true
		}
	}
	return false
}

// toStringSet 把字符串 slice 转成 set，便于 O(1) 查找。nil/空输入返回 nil。
func toStringSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
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
	disabledChannelUIDs map[string]bool,
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
		// P1.5：按 channel 禁用——命中的渠道对 autopilot 不存在，走和
		// "候选不可用" 完全相同的跳过路径，不影响其他非 autopilot 选路径。
		if disabledChannelUIDs[upstream.ChannelUID] {
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

	// ── OriginTier tie-breaker（同分时按 OriginTier rank 降序）──
	// 单轮 sort.SliceStable：Score 降序主序 + 同分时 OriginTier 降序次序
	// 稳定排序保证同分同 rank 的候选保持输入相对顺序不变
	if len(scoredEntries) > 1 {
		sort.SliceStable(scoredEntries, func(i, j int) bool {
			ci, cj := scoredEntries[i], scoredEntries[j]
			// 主序：Score 降序
			if ci.scored.Score != cj.scored.Score {
				return ci.scored.Score > cj.scored.Score
			}
			// 同分 tie-breaker：OriginTier rank 降序
			return originTierRank(ci.entry.OriginTier) > originTierRank(cj.entry.OriginTier)
		})
	}

	// ── Phase 2: Advisor hint + 本地候选 ──
	// 1) advisor hint 评估（shadow 模式下 Applied=false，不影响调度）
	var advisorDecisionUID string
	// advisorMinQualityTier 由下方闭包写入，供硬约束过滤阶段直接读取（类型化传值，
	// 不经过 trace.GlobalFilterReasons 的字符串往返 —— 避免格式变更导致静默失效）。
	var advisorMinQualityTier QualityTier
	if r.advisor != nil && r.decisionStore != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("[SmartRouter-Advisor] panic recovered (fail-open): %v", rec)
				}
			}()

			// 从 RequestProfile 构建 AdvisorInput
			advisorInput := AdvisorInput{
				RequestKind:      profile.ChannelKind,
				Operation:        profile.Operation,
				RequestedModel:   profile.Model,
				AgentRole:        profile.AgentRole,
				InputTokenBucket: classifyTokenBucket(profile.EstTokens),
				HasImage:         profile.HasImage,
				NeedsToolUse:     profile.ToolUseNeed,
				NeedsReasoning:   profile.ReasoningNeed,
				NeedsLongContext: profile.ContextNeed > 50000,
			}
			hint, _ := r.advisor.EvaluateShadow(advisorInput)

			// 获取 TrustedRoutingAdvisorConfig
			autopilotCfg := r.configManager.GetConfig().AutopilotRouting
			effect := ResolveAdvisorHintEffect(hint, autopilotCfg.TrustedRoutingAdvisor, profile.TaskClass)

			// 无论 Applied 与否，都记录决策（用于人工审查 promotion 依据）
			rec := &AdvisorDecisionRecord{
				AdvisorUID:        "heuristic", // 固定值：Phase 1 只使用启发式后端
				AdvisorOriginTier: "local",
				Mode:              r.advisor.State(),
				TaskClass:         profile.TaskClass,
				PromptHash:        profile.PromptHash,
				InputTokenBucket:  advisorInput.InputTokenBucket,
				Applied:           effect.Applied,
				Outcome:           "shadow", // 后续在调度结果回调中更新
				CreatedAt:         time.Now().UTC(),
			}
			if hint != nil {
				rec.Hint = *hint
			}
			if recordErr := r.decisionStore.Record(rec); recordErr != nil {
				log.Printf("[SmartRouter-Advisor] 决策记录失败: %v", recordErr)
			}
			advisorDecisionUID = rec.DecisionUID
			trace.AdvisorDecisionUID = advisorDecisionUID

			if effect.Applied {
				// auto 模式下：MinQualityTier 转化为硬约束过滤条件
				if mode == RoutingModeAuto && effect.MinQualityTier != "" {
					advisorMinQualityTier = effect.MinQualityTier
					// trace 侧仍记录可读原因，供 UI/人工审查展示（非控制流依赖）。
					trace.GlobalFilterReasons["advisor_min_quality_tier"] = []string{
						fmt.Sprintf("MinQualityTier=%s (Applied=true)", effect.MinQualityTier),
					}
				}

				// 本地候选允许标记：需结合 LocalModelRoutingConfig 再判一次
				if effect.AllowLocalCandidate && r.localRuntimeStore != nil {
					localCfg := autopilotCfg.LocalModelRouting
					localEntries := CollectLocalCandidates(r.localRuntimeStore, localCfg, profile.TaskClass)
					for _, le := range localEntries {
						localEntry := channelScoreEntry{
							ChannelUID:          le.RuntimeUID,
							ChannelKind:         profile.ChannelKind,
							OriginTier:          OriginTierLocal,
							HealthState:         HealthStateHealthy,
							EstimatedCost:       le.EstimatedCost,
							SupportsVision:      le.SupportsVision,
							SupportsToolCalls:   le.SupportsToolCalls,
							SupportsReasoning:   le.SupportsReasoning,
							ContextWindowTokens: le.ContextWindowTokens,
							ScoringCandidate: ScoringCandidate{
								ChannelUID:                le.RuntimeUID,
								QualityTier:               QualityTierNormal, // 中性默认值
								StabilityTier:             StabilityTierNormal,
								SpeedTier:                 SpeedTierNormal,
								CostTier:                  CostTierFree, // 本地运行时免费
								HealthState:               HealthStateHealthy,
								ProviderQualityScore:      0.5,
								ProviderQualityConfidence: 0.3,
								SavingsScore:              0.5,
								DomainStrengthScore:       0.5,
							},
						}
						// 本地候选纳入评分流程
						localEntry.ScoringCandidate.SavingsScore = savingsMap[le.RuntimeUID]
						localScored := ScoreCandidate(localEntry.ScoringCandidate, scoringCtx)
						scoredEntries = append(scoredEntries, scoredEntry{entry: localEntry, scored: localScored})
						// 本地候选成本为0，可能影响 savingsMap 归一化；
						// 但不影响排序结果（savings 只是其中一个维度）
					}
				}

				log.Printf("[SmartRouter-Advisor] hint生效 taskClass=%s MinQualityTier=%s AllowLocal=%v",
					string(profile.TaskClass), effect.MinQualityTier, effect.AllowLocalCandidate)
			}
		}()
	}

	// ── 人工意图匹配（设计 §4.6.4）──
	// 在评分排序后、构建结果前执行；shadow 模式只标注不影响输出。
	var matchedIntent *IntentMatchResult
	var intentTargetUID string
	if r.intentStore != nil && len(scoredEntries) > 1 {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("[SmartRouter-IntentMatch] panic recovered (fail-open): %v", rec)
				}
			}()

			activeIntents := r.intentStore.ListActive()
			if len(activeIntents) == 0 {
				return
			}

			matchCtx := &IntentMatchContext{
				ChannelKind: profile.ChannelKind,
				Model:       profile.Model,
				TaskClass:   profile.TaskClass,
				AgentRole:   profile.AgentRole,
				SessionID:   profile.SessionID,
				PromptHash:  profile.PromptHash,
			}
			matchedIntent = MatchIntent(matchCtx, activeIntents)
			if matchedIntent == nil || matchedIntent.ChannelUID == "" {
				matchedIntent = nil
				return
			}
			intentTargetUID = matchedIntent.ChannelUID

			// supervisor 保护：third-party 渠道的 model_trial 不覆盖 supervisor，
			// 除非意图 TaskClasses 显式包含 supervisor。
			if profile.TaskClass == TaskClassSupervisor &&
				matchedIntent.Intent.IntentType == IntentTypeModelTrial &&
				!intentExplicitlyTargetsSupervisor(matchedIntent.Intent) {
				for _, se := range scoredEntries {
					if se.entry.ChannelUID == intentTargetUID &&
						se.entry.OriginTier == OriginTierThird {
						log.Printf("[SmartRouter-SupervisorProtect] third-party 渠道 %s 的 model_trial 不覆盖 supervisor (intent=%s)",
							intentTargetUID, matchedIntent.Intent.IntentUID)
						trace.GlobalFilterReasons["supervisor_protect"] = []string{
							fmt.Sprintf("intent=%s: third-party model_trial blocked for supervisor", matchedIntent.Intent.IntentUID),
						}
						matchedIntent = nil
						intentTargetUID = ""
						return
					}
				}
			}

			// 将目标渠道提升到 scoredEntries 首位（protected candidate）
			targetIdx := -1
			for i, se := range scoredEntries {
				if se.entry.ChannelUID == intentTargetUID {
					targetIdx = i
					break
				}
			}
			if targetIdx < 0 {
				// 目标渠道不在候选中
				trace.GlobalFilterReasons["intent_target_missing"] = []string{
					fmt.Sprintf("intent=%s: target channel %s not in candidates", matchedIntent.Intent.IntentUID, intentTargetUID),
				}
				matchedIntent = nil
				intentTargetUID = ""
				return
			}
			if targetIdx > 0 {
				promoted := scoredEntries[targetIdx]
				copy(scoredEntries[1:targetIdx+1], scoredEntries[0:targetIdx])
				scoredEntries[0] = promoted
			}

			trace.ManualIntentUID = matchedIntent.Intent.IntentUID
			trace.GlobalFilterReasons["intent_match"] = matchedIntent.Reasons

			log.Printf("[SmartRouter-IntentMatch] uid=%s type=%s target=%s specificity=%d",
				matchedIntent.Intent.IntentUID, string(matchedIntent.Intent.IntentType),
				intentTargetUID, matchedIntent.Specificity)
		}()
	}

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

			// advisor hint 的 MinQualityTier 约束（只在 auto 模式生效，且 hint 真正 Applied 时才非零值）
			if advisorMinQualityTier != "" {
				if advisorMinQualityReasons := MinQualityTierReasons(se.entry.ScoringCandidate.QualityTier, advisorMinQualityTier); len(advisorMinQualityReasons) > 0 {
					reasons = append(reasons, advisorMinQualityReasons...)
				}
			}

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

		// 人工意图效果检查：目标渠道是否通过了硬约束过滤
		if matchedIntent != nil && intentTargetUID != "" {
			targetSurvived := false
			for _, ch := range result {
				upstream := upstreamFor(ch)
				matchUID := fmt.Sprintf("ch_%d", ch.Index)
				if upstream != nil && upstream.ChannelUID != "" {
					matchUID = upstream.ChannelUID
				}
				if matchUID == intentTargetUID {
					targetSurvived = true
					break
				}
			}
			if !targetSurvived {
				// 意图目标被硬约束过滤：回退到过滤后的默认排序
				result = filteredResult
				trace.FallbackUsed = true
				trace.GlobalFilterReasons["intent_fallback"] = []string{
					fmt.Sprintf("intent=%s: target %s filtered by hard constraints, fallback to score order",
						matchedIntent.Intent.IntentUID, intentTargetUID),
				}
				matchedIntent.FallbackUsed = true
				if r.intentStore != nil {
					r.intentStore.RecordFallback(matchedIntent.Intent.IntentUID)
				}
				log.Printf("[SmartRouter-IntentFallback] uid=%s target=%s filtered by hard constraints",
					matchedIntent.Intent.IntentUID, intentTargetUID)
			} else if r.intentStore != nil {
				r.intentStore.RecordHit(matchedIntent.Intent.IntentUID, true, 0)
			}
		}
	}

	// 记录 trace 信息
	trace.Candidates = candidates
	trace.CandidatesAfter = len(candidates)
	trace.SortReasons = []string{"smart_routing_score"}
	if matchedIntent != nil {
		trace.SortReasons = append(trace.SortReasons, "intent_promote")
		if matchedIntent.FallbackUsed {
			trace.SortReasons = append(trace.SortReasons, "intent_fallback")
		}
	}
	if mode == RoutingModeAssist {
		trace.SortReasons = append(trace.SortReasons, "assist_reorder")
		// assist 模式下意图命中即生效（无硬约束过滤），记录 RecordHit
		if matchedIntent != nil && !matchedIntent.FallbackUsed && r.intentStore != nil {
			r.intentStore.RecordHit(matchedIntent.Intent.IntentUID, true, 0)
		}
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

	// Phase 4 Item 8: 候选排名回调（A/B 测试用）
	// 在 trace 持久化之后、返回之前调用，确保候选数据已稳定。
	if r.onCandidatesRanked != nil && len(candidates) > 0 {
		r.onCandidatesRanked(profile.Model, profile.ChannelKind, candidates)
	}

	intentUID := ""
	if matchedIntent != nil {
		intentUID = matchedIntent.Intent.IntentUID
	}
	log.Printf("[SmartRouter-Filter] taskClass=%s mode=%s candidates=%d fallback=%v intent=%s shadow=%s duration=%dms",
		string(profile.TaskClass), string(mode), len(candidates), fallbackUsed,
		intentUID, trace.ShadowChannelUID, trace.DurationMs)

	// shadow/dryrun 模式：返回原始候选列表（不影响真实调度）
	if mode == RoutingModeShadow || mode == RoutingModeDryRun {
		return channels, nil
	}

	// assist/auto 模式：返回评分重排后的候选列表
	return result, nil
}

// channelScoreEntry 渠道评分输入条目。
type channelScoreEntry struct {
	ChannelUID          string
	ChannelKind         string
	MetricsKey          string
	OriginTier          ChannelOriginTier
	HealthState         HealthState
	EstimatedCost       float64
	ChannelIndex        int
	SupportsVision      bool  // 渠道是否支持识图（来自画像聚合 + 手动配置覆盖）
	SupportsToolCalls   bool  // 渠道是否支持工具调用（来自画像聚合）
	SupportsReasoning   bool  // 渠道是否支持推理（来自画像聚合）
	ContextWindowTokens int   // 渠道上下文窗口大小（0 = 未知，来自画像聚合）
	ScoringCandidate    ScoringCandidate
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

			// 工具调用、推理、上下文窗口能力（画像聚合）
			entry.SupportsToolCalls = agg.SupportsToolCalls
			entry.SupportsReasoning = agg.SupportsReasoning
			// ContextWindowTokens：ChannelProfile 当前无此聚合字段，
			// 画像聚合层尚未从模型注册表获取。
			// 暂填 0（未知），后续模型注册表 Phase 3 接入后填充。

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
//   - CapabilityFloor：请求需要工具调用但渠道不支持
//   - CapabilityFloor：上下文窗口需求大于渠道容量
func autoHardConstraintReasons(profile *RequestProfile, entry *channelScoreEntry) []string {
	var reasons []string

	// 识图硬约束
	if profile.VisionNeed && !entry.SupportsVision {
		reasons = append(reasons, "vision_unsupported")
	}

	// CapabilityFloor 三项硬约束（工具调用、推理、上下文窗口）
	reasons = append(reasons, CapabilityFloorReasons(CandidateCapabilities{
		SupportsToolCalls:   entry.SupportsToolCalls,
		SupportsReasoning:   entry.SupportsReasoning,
		ContextWindowTokens: entry.ContextWindowTokens,
	}, profile)...)

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

// classifyTokenBucket 将估算 token 数映射到 AdvisorInput 所需的分桶字符串。
// 遵循 AdvisorInput 白名单：<1k | 1-10k | 10-50k | 50k+
func classifyTokenBucket(estTokens int) string {
	if estTokens >= 50000 {
		return "50k+"
	}
	if estTokens >= 10000 {
		return "10-50k"
	}
	if estTokens >= 1000 {
		return "1-10k"
	}
	return "<1k"
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
