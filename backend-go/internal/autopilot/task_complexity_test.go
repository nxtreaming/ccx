package autopilot

import "testing"

func TestInferTaskComplexity(t *testing.T) {
	tests := []struct {
		name    string
		signals ComplexitySignals
		want    TaskComplexity
	}{
		{name: "空正文未知", signals: ComplexitySignals{}, want: TaskComplexityUnknown},
		{name: "问候轻量", signals: ComplexitySignals{PromptText: "你好", MessageCount: 1}, want: TaskComplexityTrivial},
		{name: "普通实现任务", signals: ComplexitySignals{PromptText: "请实现用户列表分页并补充测试", MessageCount: 1}, want: TaskComplexityRoutine},
		{name: "短但困难的证明", signals: ComplexitySignals{PromptText: "prove that this algorithm is lock-free", MessageCount: 1}, want: TaskComplexityComplex},
		{name: "架构调试任务", signals: ComplexitySignals{PromptText: "定位分布式调度的根因并重构架构", MessageCount: 2}, want: TaskComplexityComplex},
		{name: "长对话复杂", signals: ComplexitySignals{PromptText: "继续修复", MessageCount: 12}, want: TaskComplexityComplex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InferTaskComplexity(tt.signals); got != tt.want {
				t.Fatalf("InferTaskComplexity() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyUsesSemanticComplexityBeforeTokenCount(t *testing.T) {
	complex := Classify(ClassifierInput{
		ChannelKind: "messages",
		Operation:   "completion",
		AgentRole:   "main",
		EstTokens:   200,
		Complexity:  TaskComplexityComplex,
	})
	if complex != TaskClassSupervisor {
		t.Fatalf("complex short request = %q, want supervisor", complex)
	}

	routine := Classify(ClassifierInput{
		ChannelKind: "messages",
		Operation:   "completion",
		AgentRole:   "main",
		EstTokens:   200,
		Complexity:  TaskComplexityRoutine,
	})
	if routine != TaskClassWorker {
		t.Fatalf("routine short request = %q, want worker", routine)
	}
}
