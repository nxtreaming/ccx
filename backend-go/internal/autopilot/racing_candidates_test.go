package autopilot

import "testing"

// 回归：2026-09-08 事故——竞速影子候选过滤把 Selected（=通过硬约束的可行候选）
// 误当「主调度选中行」跳过，导致缓存全是可行候选时 picked 恒空、竞速静默不触发。
func TestRacingShadowCandidatesPicksFeasibleNonPrimary(t *testing.T) {
	candidates := []RoutingCandidate{
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m1", Selected: true},  // 主行（身份命中）
		{ChannelUID: "ch_a", KeyIdentity: "key2", ActualModel: "m1", Selected: true},  // 同渠道换 key：可作影子
		{ChannelUID: "ch_b", KeyIdentity: "key3", ActualModel: "m2", Selected: true},  // 跨渠道：可作影子
		{ChannelUID: "ch_c", KeyIdentity: "key4", ActualModel: "m3", Selected: false}, // 不可行：不得作影子
	}
	got := RacingShadowCandidates(candidates, "ch_a", "key1", "m1", 3)
	if len(got) != 2 {
		t.Fatalf("应选出 2 个可行影子候选（同渠道换 key + 跨渠道），got %d: %+v", len(got), got)
	}
	// 多样性纪律：异渠道排在同渠道之前（渠道排队慢时同渠道影子大概率同样慢）
	if got[0].ChannelUID != "ch_b" {
		t.Errorf("第一个影子应为异渠道 ch_b，got %+v", got[0])
	}
	if got[1].ChannelUID != "ch_a" || got[1].KeyIdentity != "key2" {
		t.Errorf("第二个影子应为同渠道换 key（ch_a/key2），got %+v", got[1])
	}
	for _, c := range got {
		if c.Selected == false {
			t.Errorf("不可行候选泄漏进影子集: %+v", c)
		}
	}
}

// 回归（2026-09-08 20:39 生产事故）：主 key 不得作影子（同账号并发放大器）；
// 多影子之间按 key 分散（同 key 只取一行）。
func TestRacingShadowCandidatesKeyDiversity(t *testing.T) {
	candidates := []RoutingCandidate{
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m1", Selected: true}, // 主行
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m2", Selected: true}, // 主 key 他模型：排除
		{ChannelUID: "ch_a", KeyIdentity: "key2", ActualModel: "m1", Selected: true},
		{ChannelUID: "ch_b", KeyIdentity: "key3", ActualModel: "m1", Selected: true},
		{ChannelUID: "ch_b", KeyIdentity: "key3", ActualModel: "m2", Selected: true}, // 与上同 key：去重
	}
	got := RacingShadowCandidates(candidates, "ch_a", "key1", "m1", 3)
	if len(got) != 2 {
		t.Fatalf("主 key 行应排除+同 key 去重，应选 2 个, got %d: %+v", len(got), got)
	}
	if got[0].ChannelUID != "ch_b" || got[0].KeyIdentity != "key3" {
		t.Errorf("第一个影子应为异渠道 ch_b/key3, got %+v", got[0])
	}
	if got[1].ChannelUID != "ch_a" || got[1].KeyIdentity != "key2" {
		t.Errorf("第二个影子应为 ch_a/key2, got %+v", got[1])
	}
}

func TestRacingShadowCandidatesLimitsAndEmpty(t *testing.T) {
	candidates := []RoutingCandidate{
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m1", Selected: true},
		{ChannelUID: "ch_b", KeyIdentity: "key2", ActualModel: "m2", Selected: true},
		{ChannelUID: "ch_c", KeyIdentity: "key3", ActualModel: "m3", Selected: true},
	}
	if got := RacingShadowCandidates(candidates, "ch_a", "key1", "m1", 1); len(got) != 1 {
		t.Fatalf("limit=1 应只选 1 个, got %d", len(got))
	}
	// 全部候选与主行身份相同：无可行影子
	same := []RoutingCandidate{
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m1", Selected: true},
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m1", Selected: true},
	}
	if got := RacingShadowCandidates(same, "ch_a", "key1", "m1", 2); len(got) != 0 {
		t.Fatalf("同身份候选应全部排除, got %d", len(got))
	}
	// 全部不可行：空
	infeasible := []RoutingCandidate{
		{ChannelUID: "ch_b", KeyIdentity: "k", ActualModel: "m2", Selected: false},
	}
	if got := RacingShadowCandidates(infeasible, "ch_a", "key1", "m1", 2); len(got) != 0 {
		t.Fatalf("不可行候选不得作影子, got %d", len(got))
	}
}

// 回归：延迟负反馈学习结论（渠道×模型慢证据达阈值）的组合不得作影子目标。
func TestRacingShadowCandidatesExcludesLatencyDegraded(t *testing.T) {
	orig := racingShadowExcludesLookup
	t.Cleanup(func() { racingShadowExcludesLookup = orig })
	racingShadowExcludesLookup = func(channelUID, model string) bool {
		return channelUID == "ch_b" && model == "m2"
	}

	candidates := []RoutingCandidate{
		{ChannelUID: "ch_a", KeyIdentity: "key1", ActualModel: "m1", Selected: true}, // 主行
		{ChannelUID: "ch_b", KeyIdentity: "key2", ActualModel: "m2", Selected: true}, // 慢组合：排除
		{ChannelUID: "ch_c", KeyIdentity: "key3", ActualModel: "m3", Selected: true}, // 正常
	}
	got := RacingShadowCandidates(candidates, "ch_a", "key1", "m1", 3)
	if len(got) != 1 || got[0].ChannelUID != "ch_c" {
		t.Fatalf("慢组合应被排除, got %+v", got)
	}
}
