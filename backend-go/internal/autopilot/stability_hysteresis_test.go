package autopilot

import "testing"

func TestApplyStabilityHysteresis_StreakBelowThresholdDoesNotChangeEffective(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.StabilityTier = StabilityTierUnstable
	old.EffectiveStabilityTier = StabilityTierUnstable
	old.StabilityPendingTier = StabilityTierStable
	old.StabilityPendingStreak = 1 // streak = 1, threshold = 3 -> 不应晋升

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierStable // rawTier = stable

	applyStabilityHysteresis(old, current, StabilityTierStable, 3)

	if current.EffectiveStabilityTier != StabilityTierUnstable {
		t.Errorf("streak 低于阈值时 EffectiveStabilityTier 不应变更: got %s, want unstable", current.EffectiveStabilityTier)
	}
	if current.StabilityPendingStreak != 2 {
		t.Errorf("streak 应从 1 递增到 2: got %d", current.StabilityPendingStreak)
	}
	if current.StabilityPendingTier != StabilityTierStable {
		t.Errorf("StabilityPendingTier 应保持 stable: got %s", current.StabilityPendingTier)
	}
}

func TestApplyStabilityHysteresis_StreakReachingThresholdPromotes(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.StabilityTier = StabilityTierNormal
	old.EffectiveStabilityTier = StabilityTierNormal
	old.StabilityPendingTier = StabilityTierStable
	old.StabilityPendingStreak = 2 // streak = 2, threshold = 3 -> 第 3 轮达到阈值

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierStable

	applyStabilityHysteresis(old, current, StabilityTierStable, 3)

	if current.EffectiveStabilityTier != StabilityTierStable {
		t.Errorf("streak 达到阈值时应晋升: got %s, want stable", current.EffectiveStabilityTier)
	}
	if current.StabilityPendingStreak != 3 {
		t.Errorf("streak 应递增到 3: got %d", current.StabilityPendingStreak)
	}
}

func TestApplyStabilityHysteresis_StreakReachingThresholdDemotes(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.StabilityTier = StabilityTierStable
	old.EffectiveStabilityTier = StabilityTierStable
	old.StabilityPendingTier = StabilityTierUnstable
	old.StabilityPendingStreak = 1 // streak = 1, threshold = 2 -> 第 2 轮达到阈值

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierUnstable

	applyStabilityHysteresis(old, current, StabilityTierUnstable, 2)

	if current.EffectiveStabilityTier != StabilityTierUnstable {
		t.Errorf("streak 达到阈值时应降级: got %s, want unstable", current.EffectiveStabilityTier)
	}
	if current.StabilityPendingStreak != 2 {
		t.Errorf("streak 应递增到 2: got %d", current.StabilityPendingStreak)
	}
}

func TestApplyStabilityHysteresis_TierChangeResetsStreak(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.StabilityTier = StabilityTierNormal
	old.EffectiveStabilityTier = StabilityTierNormal
	old.StabilityPendingTier = StabilityTierStable
	old.StabilityPendingStreak = 3 // 已累积 3 轮 stable

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierUnstable // 突然变为 unstable

	applyStabilityHysteresis(old, current, StabilityTierUnstable, 3)

	// tier 变更应重置 streak 为 1
	if current.StabilityPendingStreak != 1 {
		t.Errorf("tier 变更后 streak 应重置为 1: got %d", current.StabilityPendingStreak)
	}
	if current.StabilityPendingTier != StabilityTierUnstable {
		t.Errorf("StabilityPendingTier 应更新为 unstable: got %s", current.StabilityPendingTier)
	}
	// 不应立即降级（streak=1 < threshold=3）
	if current.EffectiveStabilityTier != StabilityTierNormal {
		t.Errorf("tier 变更后不应立即降级: got %s, want normal", current.EffectiveStabilityTier)
	}
}

func TestApplyStabilityHysteresis_FirstEverProfileAdoptsImmediately(t *testing.T) {
	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierStable

	applyStabilityHysteresis(nil, current, StabilityTierStable, 5)

	if current.EffectiveStabilityTier != StabilityTierStable {
		t.Errorf("首次画像（old==nil）应直接采纳 rawTier: got %s, want stable", current.EffectiveStabilityTier)
	}
	if current.StabilityPendingStreak != 1 {
		t.Errorf("首次画像 streak 应为 1: got %d", current.StabilityPendingStreak)
	}
	if current.StabilityPendingTier != StabilityTierStable {
		t.Errorf("首次画像 pendingTier 应为 stable: got %s", current.StabilityPendingTier)
	}
}

func TestApplyStabilityHysteresis_FirstEverWithEmptyEffectiveAdoptsImmediately(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.EffectiveStabilityTier = "" // 从未经过滞后

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierNormal

	applyStabilityHysteresis(old, current, StabilityTierNormal, 3)

	if current.EffectiveStabilityTier != StabilityTierNormal {
		t.Errorf("EffectiveStabilityTier 为空时应直接采纳 rawTier: got %s, want normal", current.EffectiveStabilityTier)
	}
	if current.StabilityPendingStreak != 1 {
		t.Errorf("streak 应为 1: got %d", current.StabilityPendingStreak)
	}
}

func TestApplyStabilityHysteresis_ZeroThresholdTreatedAsOne(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.StabilityTier = StabilityTierNormal
	old.EffectiveStabilityTier = StabilityTierNormal
	old.StabilityPendingTier = StabilityTierStable
	old.StabilityPendingStreak = 0

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierStable

	applyStabilityHysteresis(old, current, StabilityTierStable, 0) // threshold <= 0 -> 1

	// threshold=0 treated as 1; streak=0+1=1 >= 1 -> should adopt
	if current.EffectiveStabilityTier != StabilityTierStable {
		t.Errorf("threshold=0 应视为 1，streak=1 >= 1 应晋升: got %s, want stable", current.EffectiveStabilityTier)
	}
	if current.StabilityPendingStreak != 1 {
		t.Errorf("streak 应为 1: got %d", current.StabilityPendingStreak)
	}
}

func TestApplyStabilityHysteresis_NegativeThresholdTreatedAsOne(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.StabilityTier = StabilityTierUnstable
	old.EffectiveStabilityTier = StabilityTierUnstable
	old.StabilityPendingTier = StabilityTierStable
	old.StabilityPendingStreak = 0

	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.StabilityTier = StabilityTierStable

	applyStabilityHysteresis(old, current, StabilityTierStable, -1) // negative -> 1

	if current.EffectiveStabilityTier != StabilityTierStable {
		t.Errorf("threshold=-1 应视为 1，streak=1 >= 1 应晋升: got %s, want stable", current.EffectiveStabilityTier)
	}
}
