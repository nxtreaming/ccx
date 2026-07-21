package autopilot

import "strings"

// ── 确定性任务分类器（设计 §5.1 + P0.3）──
//
// Classify 根据脱敏的 ClassifierInput 确定性地推导 TaskClass。
// 同一输入永远产生同一输出；不引入 LLM 或概率判断。
//
// 规则顺序（优先级从高到低，§5.1）：
//
//  1. 原生生图端点：ChannelKind=="images" || ImageGenNeed → image_generation
//  2. 原生 embedding 端点：ChannelKind=="vectors" || EmbeddingNeed → embedding
//  3. 识图理解任务：HasImage && VisionNeed → vision
//  4. 长上下文任务：ContextNeed > 200_000 → long_context
//  5. 明确复杂的主任务 → supervisor
//  6. 轻任务：isLightweightRequest 命中 → lightweight
//  7. 子代理或常规任务 → worker
//  8. 主代理或未知 → supervisor
//     兜底：→ worker
//
// P0.3 约束：
//   - images → image_generation，不做 chat→images 协议转换
//   - vectors → embedding
//   - 不确定时升级到更保守分类，不降级到 lightweight
//   - advisor 只能在既定枚举内提供 hint，不能发明新分类
func Classify(input ClassifierInput) TaskClass {
	// 规则 1：原生生图端点优先判定；不做 chat → images 协议转换
	if input.ChannelKind == "images" || input.ImageGenNeed {
		return TaskClassImageGen
	}

	// 规则 2：原生 embedding / vectors 端点优先判定
	if input.ChannelKind == "vectors" || input.EmbeddingNeed {
		return TaskClassEmbedding
	}

	// 规则 3：识图理解任务
	if input.HasImage && input.VisionNeed {
		return TaskClassVision
	}

	// 规则 4：长上下文任务
	if input.ContextNeed > 200_000 {
		return TaskClassLongContext
	}

	// 规则 5：复杂主任务优先使用 supervisor；复杂子代理仍保持 worker 语义，
	// 其质量目标由 Complexity 单独提升。
	if input.Complexity == TaskComplexityComplex && input.AgentRole != "subagent" {
		return TaskClassSupervisor
	}

	// 规则 6：明确的低风险轻任务
	if isLightweightRequest(input) {
		return TaskClassLightweight
	}

	// 规则 7：子代理或正文明确为常规任务
	if input.AgentRole == "subagent" {
		return TaskClassWorker
	}
	if input.Complexity == TaskComplexityRoutine {
		return TaskClassWorker
	}

	// 规则 8：主代理/监工（正文未知时保守走 Supervisor）
	if input.AgentRole == "main" || input.AgentRole == "" {
		return TaskClassSupervisor
	}

	// 兜底：不确定时升级到更保守分类（worker），不降级到 lightweight
	return TaskClassWorker
}

// lightweightOperationWhitelist 白名单中的 Operation 视为低风险轻任务。
var lightweightOperationWhitelist = map[string]bool{
	"count_tokens":      true,
	"title_generation":  true,
	"classification":    true,
	"format_conversion": true,
	"summarize":         true,
	"translation":       true,
}

// lightweightModelSignals 弱信号：模型名包含这些子串暗示轻任务，但不能单独决定。
var lightweightModelSignals = []string{
	"haiku", "mini", "flash", "nano", "lite",
}

// isLightweightRequest 判断是否为明确的低风险轻任务（§5.1 + P0.3）。
//
// 规则（全部满足才返回 true）：
//  1. 无图片、工具、reasoning、长上下文或原生端点能力需求。
//  2. Operation 命中白名单（count_tokens、标题生成、分类、格式转换、摘要、翻译），
//     或正文复杂度明确为 trivial。
//  3. 模型名包含 haiku/mini/flash 等子串作为弱信号加分，但不能单独决定 lightweight。
//
// P0.3：模型名弱信号必须与其他条件组合，不能单独决定 lightweight。
func isLightweightRequest(input ClassifierInput) bool {
	// 需要原生端点能力的不是轻任务
	if input.ImageGenNeed || input.EmbeddingNeed || input.VisionNeed {
		return false
	}

	// 有图片、工具调用、推理需求的不是轻任务
	if input.HasImage || input.ToolUseNeed || input.ReasoningNeed {
		return false
	}

	// 需要长上下文的不是轻任务
	if input.ContextNeed > 200_000 {
		return false
	}

	// 有 AgentType 的通常是子代理/特定 agent 框架，不视为轻任务
	if input.AgentType != "" {
		return false
	}

	// 白名单操作在通过共同硬约束后可直接判定为轻任务。
	if lightweightOperationWhitelist[input.Operation] {
		return true
	}

	return input.Complexity == TaskComplexityTrivial && (input.EstTokens <= 0 || input.EstTokens < 10_000)
}

// classifyModelSignal 检查模型名是否包含轻量级弱信号关键词。
// 仅供外部评估参考，不参与 Classify 的确定性判定。
func classifyModelSignal(model string) bool {
	lower := strings.ToLower(model)
	for _, signal := range lightweightModelSignals {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

// BuildClassifierInput 从 RequestProfile 提取脱敏 ClassifierInput。
// 用于从已有画像构建分类输入，避免重复计算。
func BuildClassifierInput(profile *RequestProfile) ClassifierInput {
	return ClassifierInput{
		Model:         profile.Model,
		ChannelKind:   profile.ChannelKind,
		Operation:     profile.Operation,
		AgentRole:     profile.AgentRole,
		AgentType:     profile.AgentType,
		HasImage:      profile.HasImage,
		EstTokens:     profile.EstTokens,
		Complexity:    profile.Complexity,
		ContextNeed:   profile.ContextNeed,
		VisionNeed:    profile.VisionNeed,
		ImageGenNeed:  profile.ImageGenNeed,
		EmbeddingNeed: profile.EmbeddingNeed,
		ToolUseNeed:   profile.ToolUseNeed,
		ReasoningNeed: profile.ReasoningNeed,
	}
}

// ClassifyAndFill 对 ClassifierInput 执行确定性分类并填充 RequestProfile 的分类字段。
// 同时用 InferTaskDomain 填充 TaskDomain。
func ClassifyAndFill(profile *RequestProfile, input ClassifierInput) {
	profile.TaskClass = Classify(input)
	profile.TaskDomain = InferTaskDomain(input.DomainHints)
}
