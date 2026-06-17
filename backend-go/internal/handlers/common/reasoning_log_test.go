package common

import "testing"

func TestExtractReasoningEffortForLog(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "chat reasoning_effort",
			body: `{"model":"gpt-5","reasoning_effort":"high"}`,
			want: "high",
		},
		{
			name: "responses reasoning object",
			body: `{"model":"gpt-5","reasoning":{"effort":"medium"}}`,
			want: "medium",
		},
		{
			name: "claude thinking budget",
			body: `{"model":"claude","thinking":{"type":"enabled","budget_tokens":8192}}`,
			want: "budget=8192",
		},
		{
			name: "claude thinking effort",
			body: `{"model":"claude","thinking":{"type":"enabled","effort":"max"}}`,
			want: "max",
		},
		{
			name: "claude output config effort",
			body: `{"model":"glm-5.2","output_config":{"effort":"max"}}`,
			want: "max",
		},
		{
			name: "claude disabled thinking wins over stale effort",
			body: `{"model":"claude","thinking":{"type":"disabled","effort":"max"}}`,
			want: "none",
		},
		{
			name: "gemini thinking level",
			body: `{"generationConfig":{"thinkingConfig":{"thinkingLevel":"HIGH"}}}`,
			want: "HIGH",
		},
		{
			name: "gemini include thoughts",
			body: `{"generationConfig":{"thinkingConfig":{"includeThoughts":true}}}`,
			want: "enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractReasoningEffortForLog([]byte(tt.body)); got != tt.want {
				t.Fatalf("extractReasoningEffortForLog() = %q, want %q", got, tt.want)
			}
		})
	}
}
