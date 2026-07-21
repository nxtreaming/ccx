package autopilot

import (
	"fmt"
	"strings"
	"testing"
)

// ── Classify 确定性测试（§5.1 + P0.3）──
//
// 每个 TaskClass 至少 3 个用例 + 边界条件 + 确定性验证。

func TestClassify_ImageGeneration(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "images channel 直接判定",
			input: ClassifierInput{ChannelKind: "images", Operation: "image_generation"},
		},
		{
			name:  "ImageGenNeed=true 非 images 渠道",
			input: ClassifierInput{ChannelKind: "messages", Operation: "completion", ImageGenNeed: true},
		},
		{
			name:  "images channel + 有图片（图片优先于 vision）",
			input: ClassifierInput{ChannelKind: "images", HasImage: true, VisionNeed: true},
		},
		{
			name:  "images channel + image_edit 操作",
			input: ClassifierInput{ChannelKind: "images", Operation: "image_edit"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassImageGen {
				t.Errorf("Classify() = %v, want %v", got, TaskClassImageGen)
			}
		})
	}
}

func TestClassify_Embedding(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "vectors channel 直接判定",
			input: ClassifierInput{ChannelKind: "vectors", Operation: "embedding"},
		},
		{
			name:  "EmbeddingNeed=true 非 vectors 渠道",
			input: ClassifierInput{ChannelKind: "messages", Operation: "completion", EmbeddingNeed: true},
		},
		{
			name:  "vectors channel + 有图片（embedding 优先于 vision）",
			input: ClassifierInput{ChannelKind: "vectors", HasImage: true, VisionNeed: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassEmbedding {
				t.Errorf("Classify() = %v, want %v", got, TaskClassEmbedding)
			}
		})
	}
}

func TestClassify_Vision(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "有图片 + 需要识图",
			input: ClassifierInput{ChannelKind: "messages", HasImage: true, VisionNeed: true},
		},
		{
			name:  "有图片 + 需要识图 + 有工具",
			input: ClassifierInput{ChannelKind: "chat", HasImage: true, VisionNeed: true, ToolUseNeed: true},
		},
		{
			name:  "有图片 + 需要识图 + 子代理",
			input: ClassifierInput{ChannelKind: "messages", HasImage: true, VisionNeed: true, AgentRole: "subagent"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassVision {
				t.Errorf("Classify() = %v, want %v", got, TaskClassVision)
			}
		})
	}
}

func TestClassify_LongContext(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "ContextNeed 超过 200K",
			input: ClassifierInput{ChannelKind: "messages", ContextNeed: 200_001},
		},
		{
			name:  "ContextNeed 远超 200K",
			input: ClassifierInput{ChannelKind: "chat", ContextNeed: 1_000_000},
		},
		{
			name:  "长上下文 + 子代理（长上下文优先）",
			input: ClassifierInput{ChannelKind: "responses", ContextNeed: 250_000, AgentRole: "subagent"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassLongContext {
				t.Errorf("Classify() = %v, want %v", got, TaskClassLongContext)
			}
		})
	}
}

func TestClassify_Lightweight(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "count_tokens 操作（白名单）",
			input: ClassifierInput{ChannelKind: "messages", Operation: "count_tokens"},
		},
		{
			name:  "标题生成（白名单）",
			input: ClassifierInput{ChannelKind: "chat", Operation: "title_generation"},
		},
		{
			name:  "格式转换（白名单）",
			input: ClassifierInput{ChannelKind: "chat", Operation: "format_conversion"},
		},
		{
			name:  "摘要（白名单）",
			input: ClassifierInput{ChannelKind: "messages", Operation: "summarize"},
		},
		{
			name:  "翻译（白名单）",
			input: ClassifierInput{ChannelKind: "messages", Operation: "translation"},
		},
		{
			name: "低 token + 明确 trivial",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 5000, HasImage: false, ToolUseNeed: false,
				ReasoningNeed: false, ContextNeed: 0, VisionNeed: false,
				ImageGenNeed: false, EmbeddingNeed: false, AgentType: "",
				Complexity: TaskComplexityTrivial,
			},
		},
		{
			name: "正好 9999 token + 明确 trivial",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 9999, HasImage: false, ToolUseNeed: false,
				ReasoningNeed: false, ContextNeed: 0, VisionNeed: false,
				Complexity: TaskComplexityTrivial,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassLightweight {
				t.Errorf("Classify() = %v, want %v", got, TaskClassLightweight)
			}
		})
	}
}

func TestClassify_Supervisor(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "AgentRole=main",
			input: ClassifierInput{ChannelKind: "messages", AgentRole: "main", EstTokens: 50_000},
		},
		{
			name:  "AgentRole 未知（空串）默认 supervisor",
			input: ClassifierInput{ChannelKind: "messages", AgentRole: "", EstTokens: 30_000},
		},
		{
			name: "main + 有推理需求",
			input: ClassifierInput{
				ChannelKind: "responses", AgentRole: "main",
				EstTokens: 80_000, ReasoningNeed: true, ToolUseNeed: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassSupervisor {
				t.Errorf("Classify() = %v, want %v", got, TaskClassSupervisor)
			}
		})
	}
}

func TestClassify_Worker(t *testing.T) {
	tests := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "AgentRole=subagent",
			input: ClassifierInput{ChannelKind: "messages", AgentRole: "subagent", EstTokens: 20_000},
		},
		{
			name: "subagent + 有工具",
			input: ClassifierInput{
				ChannelKind: "messages", AgentRole: "subagent",
				EstTokens: 40_000, ToolUseNeed: true,
			},
		},
		{
			name: "subagent + 有推理",
			input: ClassifierInput{
				ChannelKind: "responses", AgentRole: "subagent",
				EstTokens: 60_000, ReasoningNeed: true,
			},
		},
		{
			name:  "兜底：未知 AgentRole 值 + 高 token",
			input: ClassifierInput{ChannelKind: "messages", AgentRole: "unknown_future_role", EstTokens: 20_000},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != TaskClassWorker {
				t.Errorf("Classify() = %v, want %v", got, TaskClassWorker)
			}
		})
	}
}

func TestClassify_UnknownTokenEstimateDoesNotDemote(t *testing.T) {
	got := Classify(ClassifierInput{
		ChannelKind: "messages",
		Operation:   "completion",
		AgentRole:   "",
		EstTokens:   0,
	})
	if got != TaskClassSupervisor {
		t.Fatalf("Classify() = %v, want %v", got, TaskClassSupervisor)
	}
}

func TestClassify_WhitelistedOperationStillRespectsHardNeeds(t *testing.T) {
	tests := []ClassifierInput{
		{ChannelKind: "messages", Operation: "summarize", EstTokens: 500, ToolUseNeed: true},
		{ChannelKind: "messages", Operation: "translation", EstTokens: 500, ReasoningNeed: true},
		{ChannelKind: "messages", Operation: "title_generation", EstTokens: 500, HasImage: true, VisionNeed: true},
	}
	for _, input := range tests {
		if got := Classify(input); got == TaskClassLightweight {
			t.Fatalf("Classify(%+v) unexpectedly returned lightweight", input)
		}
	}
}

// ── 边界条件测试 ──

func TestClassify_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    ClassifierInput
		expected TaskClass
	}{
		{
			name: "ContextNeed 恰好 200000（不触发 long_context）",
			input: ClassifierInput{
				ChannelKind: "messages", ContextNeed: 200_000,
				AgentRole: "main", EstTokens: 50_000,
			},
			expected: TaskClassSupervisor,
		},
		{
			name: "ContextNeed 恰好 200001（触发 long_context）",
			input: ClassifierInput{
				ChannelKind: "messages", ContextNeed: 200_001,
			},
			expected: TaskClassLongContext,
		},
		{
			name: "EstTokens 恰好 10000（不触发 lightweight 硬条件）",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 10_000, AgentRole: "main",
			},
			expected: TaskClassSupervisor,
		},
		{
			name: "EstTokens 9999 但正文难度未知（不降级）",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 9999, AgentRole: "main",
			},
			expected: TaskClassSupervisor,
		},
		{
			name: "有图片但无 VisionNeed（不触发 vision，回退到其他规则）",
			input: ClassifierInput{
				ChannelKind: "messages", HasImage: true, VisionNeed: false,
				AgentRole: "main", EstTokens: 30_000,
			},
			expected: TaskClassSupervisor,
		},
		{
			name: "有 VisionNeed 但无图片（不触发 vision）",
			input: ClassifierInput{
				ChannelKind: "messages", HasImage: false, VisionNeed: true,
				AgentRole: "main", EstTokens: 30_000,
			},
			expected: TaskClassSupervisor,
		},
		{
			name: "count_tokens + subagent（白名单优先）",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "count_tokens",
				AgentRole: "subagent",
			},
			expected: TaskClassLightweight,
		},
		{
			name: "images + EmbeddingNeed（images 规则优先级高于 vectors）",
			input: ClassifierInput{
				ChannelKind: "images", EmbeddingNeed: true,
			},
			expected: TaskClassImageGen,
		},
		{
			name: "vectors + ImageGenNeed（vectors 规则低于 images）",
			input: ClassifierInput{
				ChannelKind: "vectors", ImageGenNeed: true,
			},
			// ChannelKind=="images" 未命中，但 ImageGenNeed=true 命中规则 1
			expected: TaskClassImageGen,
		},
		{
			name:     "全部零值 + 未知角色（EstTokens=0 保守兜底）",
			input:    ClassifierInput{AgentRole: "unknown"},
			expected: TaskClassWorker,
		},
		{
			name: "lightweight 硬条件全满足但有 AgentType（不触发 lightweight）",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 5000, AgentType: "codex_subagent", AgentRole: "subagent",
			},
			expected: TaskClassWorker,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != tt.expected {
				t.Errorf("Classify() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ── 确定性验证（P0.3）──
// 同一输入跑 100 次，结果必须一致。

func TestClassify_Determinism(t *testing.T) {
	inputs := []struct {
		name  string
		input ClassifierInput
	}{
		{
			name:  "image_generation",
			input: ClassifierInput{ChannelKind: "images", Operation: "image_generation"},
		},
		{
			name:  "embedding",
			input: ClassifierInput{ChannelKind: "vectors", Operation: "embedding"},
		},
		{
			name:  "vision",
			input: ClassifierInput{ChannelKind: "messages", HasImage: true, VisionNeed: true},
		},
		{
			name:  "long_context",
			input: ClassifierInput{ChannelKind: "messages", ContextNeed: 250_000},
		},
		{
			name:  "lightweight",
			input: ClassifierInput{ChannelKind: "messages", Operation: "count_tokens"},
		},
		{
			name:  "supervisor",
			input: ClassifierInput{ChannelKind: "messages", AgentRole: "main", EstTokens: 50_000},
		},
		{
			name:  "worker",
			input: ClassifierInput{ChannelKind: "messages", AgentRole: "subagent", EstTokens: 20_000},
		},
		{
			name: "复杂边界",
			input: ClassifierInput{
				ChannelKind: "chat", Operation: "completion",
				AgentRole: "", EstTokens: 9999, HasImage: false,
				ToolUseNeed: false, ReasoningNeed: false, ContextNeed: 100_000,
			},
		},
	}

	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			first := Classify(tt.input)
			for i := 0; i < 100; i++ {
				got := Classify(tt.input)
				if got != first {
					t.Fatalf("第 %d 次 Classify() = %v, 首次 = %v（确定性违反）", i+1, got, first)
				}
			}
		})
	}
}

// ── 优先级顺序验证 ──

func TestClassify_PriorityOrder(t *testing.T) {
	tests := []struct {
		name     string
		input    ClassifierInput
		expected TaskClass
	}{
		{
			name: "images + 长上下文 + subagent（images 最高优先）",
			input: ClassifierInput{
				ChannelKind: "images", ContextNeed: 300_000,
				AgentRole: "subagent",
			},
			expected: TaskClassImageGen,
		},
		{
			name: "vectors + 长上下文（vectors 高于 long_context）",
			input: ClassifierInput{
				ChannelKind: "vectors", ContextNeed: 300_000,
			},
			expected: TaskClassEmbedding,
		},
		{
			name: "vision + 长上下文（vision 高于 long_context）",
			input: ClassifierInput{
				ChannelKind: "messages", HasImage: true, VisionNeed: true,
				ContextNeed: 300_000,
			},
			expected: TaskClassVision,
		},
		{
			name: "长上下文 + count_tokens（long_context 高于 lightweight）",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "count_tokens",
				ContextNeed: 300_000,
			},
			expected: TaskClassLongContext,
		},
		{
			name: "lightweight + main（lightweight 高于 supervisor）",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "count_tokens",
				AgentRole: "main",
			},
			expected: TaskClassLightweight,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != tt.expected {
				t.Errorf("Classify() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ── isLightweightRequest 测试 ──

func TestIsLightweightRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    ClassifierInput
		expected bool
	}{
		// 白名单 Operation
		{"count_tokens 白名单", ClassifierInput{Operation: "count_tokens"}, true},
		{"title_generation 白名单", ClassifierInput{Operation: "title_generation"}, true},
		{"classification 白名单", ClassifierInput{Operation: "classification"}, true},
		{"format_conversion 白名单", ClassifierInput{Operation: "format_conversion"}, true},
		{"summarize 白名单", ClassifierInput{Operation: "summarize"}, true},
		{"translation 白名单", ClassifierInput{Operation: "translation"}, true},

		// 硬条件全满足
		{"低 token + trivial", ClassifierInput{
			Operation: "completion", EstTokens: 5000, Complexity: TaskComplexityTrivial,
		}, true},
		{"正好 9999 + trivial", ClassifierInput{
			Operation: "completion", EstTokens: 9999, Complexity: TaskComplexityTrivial,
		}, true},

		// 硬条件不满足
		{"10000 token 不满足", ClassifierInput{
			Operation: "completion", EstTokens: 10_000, Complexity: TaskComplexityTrivial,
		}, false},
		{"有图片", ClassifierInput{
			Operation: "completion", EstTokens: 5000, HasImage: true, Complexity: TaskComplexityTrivial,
		}, false},
		{"有工具", ClassifierInput{
			Operation: "completion", EstTokens: 5000, ToolUseNeed: true, Complexity: TaskComplexityTrivial,
		}, false},
		{"有推理", ClassifierInput{
			Operation: "completion", EstTokens: 5000, ReasoningNeed: true, Complexity: TaskComplexityTrivial,
		}, false},
		{"需要识图", ClassifierInput{
			Operation: "completion", EstTokens: 5000, VisionNeed: true, Complexity: TaskComplexityTrivial,
		}, false},
		{"需要生图", ClassifierInput{
			Operation: "completion", EstTokens: 5000, ImageGenNeed: true, Complexity: TaskComplexityTrivial,
		}, false},
		{"需要 embedding", ClassifierInput{
			Operation: "completion", EstTokens: 5000, EmbeddingNeed: true, Complexity: TaskComplexityTrivial,
		}, false},
		{"长上下文", ClassifierInput{
			Operation: "completion", EstTokens: 5000, ContextNeed: 250_000, Complexity: TaskComplexityTrivial,
		}, false},
		{"有 AgentType", ClassifierInput{
			Operation: "completion", EstTokens: 5000, AgentType: "codex_subagent", Complexity: TaskComplexityTrivial,
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLightweightRequest(tt.input)
			if got != tt.expected {
				t.Errorf("isLightweightRequest() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ── classifyModelSignal 弱信号测试 ──

func TestClassifyModelSignal(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"claude-3-5-haiku", true},
		{"gpt-4o-mini", true},
		{"gemini-2.0-flash", true},
		{"gemini-nano", true},
		{"custom-lite-model", true},
		{"claude-sonnet-4", false},
		{"gpt-5.4", false},
		{"deepseek-v4", false},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := classifyModelSignal(tt.model)
			if got != tt.expected {
				t.Errorf("classifyModelSignal(%q) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

// ── AllTaskClasses 测试 ──

func TestAllTaskClasses(t *testing.T) {
	classes := AllTaskClasses()
	if len(classes) != 7 {
		t.Errorf("AllTaskClasses() 返回 %d 个, 想要 7 个", len(classes))
	}
	seen := map[TaskClass]bool{}
	for _, tc := range classes {
		if seen[tc] {
			t.Errorf("AllTaskClasses() 有重复: %v", tc)
		}
		seen[tc] = true
	}
	for _, want := range []TaskClass{
		TaskClassSupervisor, TaskClassWorker, TaskClassLightweight,
		TaskClassVision, TaskClassLongContext, TaskClassImageGen, TaskClassEmbedding,
	} {
		if !seen[want] {
			t.Errorf("AllTaskClasses() 缺少: %v", want)
		}
	}
}

// ── BuildClassifierInput 测试 ──

func TestBuildClassifierInput(t *testing.T) {
	profile := &RequestProfile{
		Model:         "claude-sonnet-4-20250514",
		ChannelKind:   "messages",
		Operation:     "completion",
		AgentRole:     "main",
		AgentType:     "claude_code_subagent",
		HasImage:      true,
		EstTokens:     50000,
		ContextNeed:   200000,
		VisionNeed:    true,
		ImageGenNeed:  false,
		EmbeddingNeed: false,
		ToolUseNeed:   true,
		ReasoningNeed: true,
	}
	input := BuildClassifierInput(profile)

	if input.Model != profile.Model {
		t.Errorf("Model = %q, want %q", input.Model, profile.Model)
	}
	if input.ChannelKind != profile.ChannelKind {
		t.Errorf("ChannelKind = %q, want %q", input.ChannelKind, profile.ChannelKind)
	}
	if input.HasImage != profile.HasImage {
		t.Errorf("HasImage = %v, want %v", input.HasImage, profile.HasImage)
	}
	if input.EstTokens != profile.EstTokens {
		t.Errorf("EstTokens = %d, want %d", input.EstTokens, profile.EstTokens)
	}
	if input.ToolUseNeed != profile.ToolUseNeed {
		t.Errorf("ToolUseNeed = %v, want %v", input.ToolUseNeed, profile.ToolUseNeed)
	}
}

// ── ClassifyAndFill 测试 ──

func TestClassifyAndFill(t *testing.T) {
	profile := &RequestProfile{
		ChannelKind: "messages",
		AgentRole:   "main",
		EstTokens:   50000,
	}
	input := ClassifierInput{
		ChannelKind: "messages",
		AgentRole:   "main",
		EstTokens:   50000,
		DomainHints: DomainHints{
			SystemPrompt: "你是一个代码审核助手",
		},
	}
	ClassifyAndFill(profile, input)

	if profile.TaskClass != TaskClassSupervisor {
		t.Errorf("TaskClass = %v, want %v", profile.TaskClass, TaskClassSupervisor)
	}
	if profile.TaskDomain != TaskDomainCodeReview {
		t.Errorf("TaskDomain = %v, want %v", profile.TaskDomain, TaskDomainCodeReview)
	}
}

// ── 基准测试 ──

func BenchmarkClassify(b *testing.B) {
	input := ClassifierInput{
		ChannelKind:   "messages",
		Operation:     "completion",
		AgentRole:     "subagent",
		EstTokens:     20000,
		ToolUseNeed:   true,
		ReasoningNeed: true,
		ContextNeed:   100000,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Classify(input)
	}
}

// ── 补充：低 token + 有特殊需求不触发 lightweight 的回归测试 ──

func TestClassify_LowTokenButNotLightweight(t *testing.T) {
	tests := []struct {
		name     string
		input    ClassifierInput
		expected TaskClass
	}{
		{
			name: "低 token + 有图片 + 无 VisionNeed",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 3000, HasImage: true, VisionNeed: false,
				AgentRole: "main",
			},
			// 有图片 → isLightweight=false → 主代理 → supervisor
			expected: TaskClassSupervisor,
		},
		{
			name: "低 token + 有工具",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 2000, ToolUseNeed: true,
				AgentRole: "subagent",
			},
			expected: TaskClassWorker,
		},
		{
			name: "低 token + 有推理",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 1000, ReasoningNeed: true,
				AgentRole: "main",
			},
			expected: TaskClassSupervisor,
		},
		{
			name: "低 token + codex_subagent",
			input: ClassifierInput{
				ChannelKind: "messages", Operation: "completion",
				EstTokens: 3000, AgentType: "codex_subagent",
				AgentRole: "subagent",
			},
			expected: TaskClassWorker,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input)
			if got != tt.expected {
				t.Errorf("Classify() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ── 所有七类全覆盖断言 ──
// 确保测试用例覆盖了全部 7 种 TaskClass。

func TestClassify_AllSevenClasses(t *testing.T) {
	coverage := map[TaskClass]bool{}
	cases := []ClassifierInput{
		{ChannelKind: "images"},                    // image_generation
		{ChannelKind: "vectors"},                   // embedding
		{HasImage: true, VisionNeed: true},         // vision
		{ContextNeed: 300_000},                     // long_context
		{Operation: "count_tokens"},                // lightweight
		{AgentRole: "main", EstTokens: 50_000},     // supervisor
		{AgentRole: "subagent", EstTokens: 20_000}, // worker
	}
	for _, input := range cases {
		coverage[Classify(input)] = true
	}
	for _, tc := range AllTaskClasses() {
		if !coverage[tc] {
			t.Errorf("TestClassify 缺少 %v 的覆盖", tc)
		}
	}
}

// ── 泛化 fuzz-like：随机组合输入不会 panic ──

func TestClassify_NoPanic(t *testing.T) {
	combos := []ClassifierInput{
		{},
		{ChannelKind: "unknown"},
		{Operation: "unknown_op"},
		{AgentRole: "weird_role", AgentType: "weird_type"},
		{HasImage: true, VisionNeed: true, ImageGenNeed: true, EmbeddingNeed: true},
		{EstTokens: -1, ContextNeed: -100},
		{Model: strings.Repeat("x", 10000)},
	}
	for i, input := range combos {
		t.Run(fmt.Sprintf("combo_%d", i), func(t *testing.T) {
			// 只要不 panic 就算通过
			_ = Classify(input)
		})
	}
}

// ── lightweightOperationWhitelist 完整性测试 ──

func TestLightweightOperationWhitelist(t *testing.T) {
	expected := []string{
		"count_tokens", "title_generation", "classification",
		"format_conversion", "summarize", "translation",
	}
	for _, op := range expected {
		if !lightweightOperationWhitelist[op] {
			t.Errorf("lightweightOperationWhitelist 缺少 %q", op)
		}
	}
}

// ── 确定性并发安全测试 ──

func TestClassify_ConcurrentDeterminism(t *testing.T) {
	input := ClassifierInput{
		ChannelKind: "messages", Operation: "completion",
		AgentRole: "main", EstTokens: 15000,
		ToolUseNeed: true, ReasoningNeed: false,
	}
	expected := Classify(input)

	const goroutines = 50
	const iterations = 100
	errCh := make(chan error, goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < iterations; i++ {
				got := Classify(input)
				if got != expected {
					errCh <- fmt.Errorf("goroutine iteration %d: got %v, want %v", i, got, expected)
					return
				}
			}
			errCh <- nil
		}()
	}

	for g := 0; g < goroutines; g++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
}
