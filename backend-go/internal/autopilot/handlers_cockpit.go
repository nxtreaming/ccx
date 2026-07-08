package autopilot

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ── 驾驶舱 API 响应类型 ────────────────────────────────────────────

// CockpitOverviewResponse 是 GET /api/cockpit/overview 的响应结构。
type CockpitOverviewResponse struct {
	// 健康概览：渠道与 endpoint 级指标
	Health CockpitHealthSummary `json:"health"`
	// 订阅概览
	Subscriptions CockpitSubscriptionSummary `json:"subscriptions"`
	// 本地 runtime 概览
	LocalRuntimes CockpitLocalRuntimeSummary `json:"localRuntimes"`
	// 手动意图概览
	ManualIntents CockpitManualIntentSummary `json:"manualIntents"`
	// 待办列表（dead / misconfigured endpoint，最多 20 条）
	TodoItems []CockpitTodoItem `json:"todoItems"`
}

// CockpitHealthSummary 聚合渠道与 endpoint 健康信息。
type CockpitHealthSummary struct {
	TotalChannels  int            `json:"totalChannels"`
	TotalEndpoints int            `json:"totalEndpoints"`
	StateCounts    map[string]int `json:"stateCounts"` // key = HealthState 常量
}

// CockpitSubscriptionSummary 订阅中心汇总。
type CockpitSubscriptionSummary struct {
	Total         int                `json:"total"`
	BalanceByCode map[string]float64 `json:"balanceByCode"` // key = 货币代码 (CNY/USD/…)，value = 余额合计
	CountByMode   map[string]int     `json:"countByMode"`   // key = BillingMode, value = 数量
	CountByTier   map[string]int     `json:"countByTier"`   // key = OriginTier, value = 数量
}

// CockpitLocalRuntimeSummary 本地运行时汇总。
type CockpitLocalRuntimeSummary struct {
	Total        int            `json:"total"`
	StatusCounts map[string]int `json:"statusCounts"` // key = LocalRuntimeStatus 常量
	TotalModels  int            `json:"totalModels"`  // 所有 runtime 发现模型数之和
}

// CockpitManualIntentSummary 手动意图汇总。
type CockpitManualIntentSummary struct {
	ActiveCount int `json:"activeCount"` // 活跃意图数
	TotalCount  int `json:"totalCount"`  // 全部意图数（含过期/耗尽/禁用）
}

// CockpitTodoItem 单条待办事项。
type CockpitTodoItem struct {
	EndpointUID     string `json:"endpointUid"`
	ChannelUID      string `json:"channelUid"`
	ChannelKind     string `json:"channelKind"`
	BaseURL         string `json:"baseUrl"`
	HealthState     string `json:"healthState"`
	SuggestedAction string `json:"suggestedAction"`
}

// ── 路由注册 ───────────────────────────────────────────────────────

// RegisterCockpitRoutes 注册驾驶舱只读聚合 API。
func RegisterCockpitRoutes(router gin.IRouter, mgr *Manager) {
	group := router.Group("/cockpit")
	{
		group.GET("/overview", handleCockpitOverview(mgr))
	}
}

// ── Handler ─────────────────────────────────────────────────────────

func handleCockpitOverview(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := CockpitOverviewResponse{}

		// 1. 健康概览（复用 ProfileStore）
		populateHealthSummary(mgr, &resp.Health)

		// 2. 订阅概览
		populateSubscriptionSummary(mgr, &resp.Subscriptions)

		// 3. 本地 runtime 概览
		populateLocalRuntimeSummary(mgr, &resp.LocalRuntimes)

		// 4. 手动意图概览
		populateManualIntentSummary(mgr, &resp.ManualIntents)

		// 5. 待办列表（dead / misconfigured，最多 20 条）
		resp.TodoItems = collectTodoItems(mgr, 20)

		c.JSON(http.StatusOK, resp)
	}
}

// ── 内部聚合函数 ────────────────────────────────────────────────────

func populateHealthSummary(mgr *Manager, out *CockpitHealthSummary) {
	profiles := mgr.ProfileStore().ListAll()

	stateCounts := map[string]int{
		string(HealthStateUnknown):       0,
		string(HealthStateHealthy):       0,
		string(HealthStateDegraded):      0,
		string(HealthStateLimited):       0,
		string(HealthStateMisconfigured): 0,
		string(HealthStateDead):          0,
	}

	channelSet := make(map[string]struct{})
	for _, p := range profiles {
		channelSet[p.ChannelUID] = struct{}{}
		state := string(p.HealthState)
		if _, ok := stateCounts[state]; ok {
			stateCounts[state]++
		} else {
			stateCounts[state] = 1
		}
	}

	out.TotalChannels = len(channelSet)
	out.TotalEndpoints = len(profiles)
	out.StateCounts = stateCounts
}

func populateSubscriptionSummary(mgr *Manager, out *CockpitSubscriptionSummary) {
	store := mgr.SubscriptionStore()
	if store == nil {
		return
	}
	subs := store.ListAll()
	out.Total = len(subs)
	out.BalanceByCode = make(map[string]float64)
	out.CountByMode = make(map[string]int)
	out.CountByTier = make(map[string]int)

	for _, s := range subs {
		code := s.Currency
		if code == "" {
			code = "unknown"
		}
		out.BalanceByCode[code] += s.Balance

		mode := s.BillingMode
		if mode == "" {
			mode = "unknown"
		}
		out.CountByMode[mode]++

		tier := s.OriginTier
		if tier == "" {
			tier = "unknown"
		}
		out.CountByTier[tier]++
	}
}

func populateLocalRuntimeSummary(mgr *Manager, out *CockpitLocalRuntimeSummary) {
	store := mgr.LocalRuntimeStore()
	if store == nil {
		return
	}
	runtimes := store.ListAll()
	out.Total = len(runtimes)
	out.StatusCounts = make(map[string]int)

	for _, r := range runtimes {
		status := string(r.Status)
		if status == "" {
			status = "unknown"
		}
		out.StatusCounts[status]++
		out.TotalModels += len(r.DiscoveredModels)
	}
}

func populateManualIntentSummary(mgr *Manager, out *CockpitManualIntentSummary) {
	store := mgr.ManualIntentStore()
	if store == nil {
		return
	}
	out.ActiveCount = len(store.ListActive())
	out.TotalCount = len(store.ListAll())
}

func collectTodoItems(mgr *Manager, limit int) []CockpitTodoItem {
	profiles := mgr.ProfileStore().ListAll()

	var items []CockpitTodoItem
	for _, p := range profiles {
		if p.HealthState != HealthStateDead && p.HealthState != HealthStateMisconfigured {
			continue
		}
		if p.SuggestedAction == ActionNone {
			continue
		}
		items = append(items, CockpitTodoItem{
			EndpointUID:     p.EndpointUID,
			ChannelUID:      p.ChannelUID,
			ChannelKind:     p.ChannelKind,
			BaseURL:         p.BaseURL,
			HealthState:     string(p.HealthState),
			SuggestedAction: string(p.SuggestedAction),
		})
		if len(items) >= limit {
			break
		}
	}
	if items == nil {
		items = []CockpitTodoItem{}
	}
	return items
}
