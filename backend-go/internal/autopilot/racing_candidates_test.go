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
	if got[0].ChannelUID != "ch_a" || got[0].KeyIdentity != "key2" {
		t.Errorf("第一个影子应保留排名顺序（ch_a/key2），got %+v", got[0])
	}
	if got[1].ChannelUID != "ch_b" {
		t.Errorf("第二个影子应为 ch_b，got %+v", got[1])
	}
	for _, c := range got {
		if c.Selected == false {
			t.Errorf("不可行候选泄漏进影子集: %+v", c)
		}
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
