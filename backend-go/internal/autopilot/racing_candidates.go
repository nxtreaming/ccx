package autopilot

import "strings"

// RacingShadowCandidates 从 SmartRouter 排名缓存中选取竞速影子候选。
//
// 与 ABTest 的 selectShadowCandidates（整渠道排除）不同，竞速按五元组粒度排除：
// 仅跳过主尝试行自身（ChannelUID+KeyIdentity+ActualModel 相同）与 Selected 行，
// 同渠道的其他 key / 其他执行模型绑定仍可作为影子目标——渠道整体慢时换绑定
// 不一定有效，但"某模型偶尔很慢"场景下同渠道其他模型常常正常。
// 返回按排名顺序至多 limit 条。
func RacingShadowCandidates(candidates []RoutingCandidate, primaryChannelUID, primaryKeyIdentity, primaryActualModel string, limit int) []RoutingCandidate {
	if limit <= 0 || len(candidates) == 0 {
		return nil
	}
	sameIdentity := func(c RoutingCandidate) bool {
		if c.ChannelUID != primaryChannelUID {
			return false
		}
		// 旧缓存行缺 key 维时空串比较即可（同渠道同模型视为同一行，保守排除）。
		return strings.TrimSpace(c.KeyIdentity) == strings.TrimSpace(primaryKeyIdentity) &&
			normalizeRoutingModelID(c.ActualModel) == normalizeRoutingModelID(primaryActualModel)
	}
	selected := make([]RoutingCandidate, 0, limit)
	for _, candidate := range candidates {
		if candidate.Selected || sameIdentity(candidate) {
			continue
		}
		selected = append(selected, candidate)
		if len(selected) >= limit {
			break
		}
	}
	return selected
}
