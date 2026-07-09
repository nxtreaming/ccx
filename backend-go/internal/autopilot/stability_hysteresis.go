package autopilot

// applyStabilityHysteresis 在无状态的 DeriveStabilityTier 之上增加连续窗口滞后层，
// 防止单轮噪声导致 StabilityTier 晋降级抖动。
//
// 调用时机：collectAll() 中，carryForwardProbeFields 之后、Upsert 之前。
// old 为 store.Get 返回的已存储画像（首次画像为 nil），
// current 为本轮 DeriveEndpointProfile 新构造的画像。
// rawTier 为本轮无状态推导出的 StabilityTier（即 profile.StabilityTier）。
// threshold 为触发晋降级所需的连续窗口数（<=0 时按 1 处理）。
func applyStabilityHysteresis(old, current *KeyEndpointProfile, rawTier StabilityTier, threshold int) {
	if threshold <= 0 {
		threshold = 1
	}

	// 首次画像或从未经过滞后：直接采纳 rawTier，不等 threshold。
	if old == nil || old.EffectiveStabilityTier == "" {
		current.EffectiveStabilityTier = rawTier
		current.StabilityPendingTier = rawTier
		current.StabilityPendingStreak = 1
		return
	}

	// 搬运旧的 pending 状态到 current
	current.StabilityPendingTier = old.StabilityPendingTier
	current.StabilityPendingStreak = old.StabilityPendingStreak

	if rawTier == current.StabilityPendingTier {
		// 与当前候选档一致，累加连续计数
		current.StabilityPendingStreak++
	} else {
		// 候选档变更，重置计数
		current.StabilityPendingTier = rawTier
		current.StabilityPendingStreak = 1
	}

	// 默认保持旧的 EffectiveStabilityTier
	current.EffectiveStabilityTier = old.EffectiveStabilityTier

	// 达到阈值则采纳候选档
	if current.StabilityPendingStreak >= threshold {
		current.EffectiveStabilityTier = rawTier
	}
}
