package autopilot

import (
	"sort"
	"strings"
)

// racingShadowExcludesLookup 供测试替换的慢组合排除查询（默认查延迟负反馈学习结论）。
var racingShadowExcludesLookup = func(channelUID, model string) bool {
	return learnedLatencyDegradedLookup(channelUID, model, "")
}

// RacingShadowCandidates 从 SmartRouter 排名缓存中选取竞速影子候选。
//
// 与 ABTest 的 selectShadowCandidates（整渠道排除）不同，竞速按五元组粒度排除：
// 仅跳过主尝试行自身（ChannelUID+KeyIdentity+ActualModel 相同），同渠道的其他
// key / 其他执行模型绑定仍可作为影子目标。
//
// 多样性纪律（kiro.rs 原版对齐，2026-09-08 20:39 生产事故补齐——三影子全打
// 主渠道且两个复用主 key，同账号并发 4×194KB 触发上游空流）：
//  1. 主 key 一律不选（与主分支并发同账号请求是放大器：限速/空流）；
//  2. 异渠道候选排在主渠道候选之前（渠道排队慢时同渠道影子大概率同样慢）；
//  3. 多影子之间按 keyIdentity 分散（同渠道多影子复用同一 key 等于自我限速）。
//
// 返回按上述纪律排序后至多 limit 条。
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
	// Selected 语义 = 通过当前模式硬约束的可行候选（可为多个），主调度选中行只是其中之一；
	// 影子必须从可行集选取，不可行候选（FilterReasons 非空）不得作影子。
	pool := make([]RoutingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Selected || sameIdentity(candidate) {
			continue
		}
		// 主 key 一律不作影子：与主分支并发同账号请求是放大器（限速/空流）。
		if strings.TrimSpace(candidate.KeyIdentity) != "" &&
			strings.TrimSpace(candidate.KeyIdentity) == strings.TrimSpace(primaryKeyIdentity) {
			continue
		}
		// 延迟负反馈学习：已学习为慢的组合不作影子目标——已知慢的候选救不了另一个慢的。
		if racingShadowExcludesLookup(candidate.ChannelUID, candidate.ActualModel) {
			continue
		}
		pool = append(pool, candidate)
	}
	if len(pool) == 0 {
		return nil
	}
	// 多样性排序：异渠道在前（稳定排序保持排名原序）；同渠道内按 keyIdentity
	// 去重（多影子分散 key，同 key 只取一行）。
	sort.SliceStable(pool, func(i, j int) bool {
		return pool[i].ChannelUID != primaryChannelUID && pool[j].ChannelUID == primaryChannelUID
	})
	selected := make([]RoutingCandidate, 0, limit)
	seenKeys := make(map[string]bool)
	for _, candidate := range pool {
		keyID := strings.TrimSpace(candidate.KeyIdentity)
		if keyID != "" {
			if seenKeys[keyID] {
				continue
			}
			seenKeys[keyID] = true
		}
		selected = append(selected, candidate)
		if len(selected) >= limit {
			break
		}
	}
	return selected
}
