package autopilot

import (
	"strings"
	"unicode/utf8"
)

// TaskComplexity 是从当前请求文本提取后立即丢弃正文得到的难度信号。
type TaskComplexity string

const (
	TaskComplexityUnknown TaskComplexity = "unknown"
	TaskComplexityTrivial TaskComplexity = "trivial"
	TaskComplexityRoutine TaskComplexity = "routine"
	TaskComplexityComplex TaskComplexity = "complex"
)

// ComplexitySignals 只在请求入口短暂存在，不进入画像或持久化存储。
type ComplexitySignals struct {
	PromptText     string
	MessageCount   int
	ToolCount      int
	EstTokens      int
	HasDiffContext bool
}

var complexityKeywordWeights = map[string]int{
	"root cause": 3, "根因": 3, "formal proof": 3, "prove that": 3, "证明": 3,
	"architecture": 2, "architect": 2, "架构": 2, "race condition": 2, "竞态": 2,
	"deadlock": 2, "死锁": 2, "distributed": 2, "分布式": 2, "security audit": 2,
	"安全审计": 2, "threat model": 2, "威胁模型": 2, "performance regression": 2,
	"性能回归": 2, "refactor": 2, "重构": 2, "code review": 2, "代码审查": 2,
	"diagnose": 1, "诊断": 1, "debug": 1, "调试": 1, "algorithm": 1, "算法": 1,
	"multi-step": 1, "多步": 1, "end-to-end": 1, "端到端": 1,
}

var trivialPrompts = map[string]bool{
	"hi": true, "hello": true, "hey": true, "ping": true,
	"thanks": true, "thank you": true, "ok": true, "okay": true,
	"你好": true, "您好": true, "谢谢": true, "好的": true,
}

// InferTaskComplexity 使用稳定、可解释的结构和关键词信号估算当前任务难度。
func InferTaskComplexity(signals ComplexitySignals) TaskComplexity {
	text := strings.ToLower(strings.TrimSpace(signals.PromptText))
	if text == "" {
		return TaskComplexityUnknown
	}

	normalized := strings.Trim(text, " \t\r\n.!?;:,，。！？；：")
	if trivialPrompts[normalized] && signals.MessageCount <= 1 && signals.ToolCount == 0 {
		return TaskComplexityTrivial
	}

	score := 0
	switch {
	case signals.EstTokens >= 50_000:
		score += 4
	case signals.EstTokens >= 20_000:
		score += 2
	case signals.EstTokens >= 8_000:
		score++
	}
	if signals.MessageCount >= 12 {
		score += 3
	} else if signals.MessageCount >= 6 {
		score += 2
	} else if signals.MessageCount >= 3 {
		score++
	}
	if signals.ToolCount >= 5 {
		score += 2
	} else if signals.ToolCount > 0 {
		score++
	}
	if signals.HasDiffContext {
		score++
	}
	promptLength := utf8.RuneCountInString(text)
	if promptLength >= 4_000 {
		score += 2
	} else if promptLength >= 1_000 {
		score++
	}
	if strings.Count(text, "```") >= 2 {
		score++
	}
	for keyword, weight := range complexityKeywordWeights {
		if strings.Contains(text, keyword) {
			score += weight
		}
	}

	if score >= 3 {
		return TaskComplexityComplex
	}
	return TaskComplexityRoutine
}
