package channelpreset

import (
	"slices"
	"testing"
)

func TestBuildPayload(t *testing.T) {
	tests := []struct {
		name           string
		req            CreateChannelRequest
		wantBaseURL    string
		wantService    string
		wantVision     bool
		wantPassback   bool
		wantCodex      bool
		wantStripCodex bool
		wantModels     []string
		wantModelMap   map[string]string
		wantFallback   string
	}{
		{
			name:         "deepseek messages (anthropic endpoint)",
			req:          CreateChannelRequest{Provider: ProviderDeepSeek, Target: TargetMessages, APIKey: "sk-test"},
			wantBaseURL:  "https://api.deepseek.com/anthropic",
			wantService:  "claude",
			wantVision:   true,
			wantPassback: true,
			wantModelMap: map[string]string{
				"haiku":  "deepseek-v4-flash",
				"opus":   "deepseek-v4-pro",
				"sonnet": "deepseek-v4-pro",
			},
		},
		{
			name:        "deepseek chat (openai endpoint)",
			req:         CreateChannelRequest{Provider: ProviderDeepSeek, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL: "https://api.deepseek.com/v1",
			wantService: "openai",
			wantVision:  true,
			wantModels:  []string{"deepseek-v4-*"},
		},
		{
			name:        "deepseek responses (openai endpoint)",
			req:         CreateChannelRequest{Provider: ProviderDeepSeek, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL: "https://api.deepseek.com/v1",
			wantService: "openai",
			wantVision:  true,
			wantModelMap: map[string]string{
				"gpt":  "deepseek-v4-pro",
				"mini": "deepseek-v4-flash",
			},
		},
		{
			name:         "mimo messages (token plan)",
			req:          CreateChannelRequest{Provider: ProviderMiMo, Target: TargetMessages, PlanID: "token-sgp-anthropic", APIKey: "tp-test"},
			wantBaseURL:  "https://token-plan-sgp.xiaomimimo.com/anthropic",
			wantService:  "claude",
			wantPassback: true,
			wantModels:   []string{"mimo-v2.5-pro", "mimo-v2.5"},
			wantModelMap: map[string]string{
				"haiku":  "mimo-v2.5-pro",
				"opus":   "mimo-v2.5-pro",
				"sonnet": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:         "mimo messages (auto plan)",
			req:          CreateChannelRequest{Provider: ProviderMiMo, Target: TargetMessages, APIKey: "tp-test"},
			wantBaseURL:  "https://api.xiaomimimo.com/anthropic",
			wantService:  "claude",
			wantPassback: true,
			wantModels:   []string{"mimo-v2.5-pro", "mimo-v2.5"},
			wantModelMap: map[string]string{
				"haiku":  "mimo-v2.5-pro",
				"opus":   "mimo-v2.5-pro",
				"sonnet": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:        "mimo chat",
			req:         CreateChannelRequest{Provider: ProviderMiMo, Target: TargetChat, APIKey: "tp-test"},
			wantBaseURL: "https://api.xiaomimimo.com/v1",
			wantService: "openai",
			wantModels:  []string{"mimo-v2.5-pro", "mimo-v2.5"},
			wantModelMap: map[string]string{
				"gpt": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:        "mimo responses",
			req:         CreateChannelRequest{Provider: ProviderMiMo, Target: TargetResponses, APIKey: "tp-test"},
			wantBaseURL: "https://api.xiaomimimo.com/v1",
			wantService: "openai",
			wantModels:  []string{"mimo-v2.5-pro", "mimo-v2.5"},
			wantModelMap: map[string]string{
				"gpt": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:        "kimi chat",
			req:         CreateChannelRequest{Provider: ProviderKimi, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL: "https://api.moonshot.cn/v1",
			wantService: "openai",
		},
		{
			name:           "kimi responses",
			req:            CreateChannelRequest{Provider: ProviderKimi, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://api.moonshot.cn/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
		},
		{
			name:        "glm chat",
			req:         CreateChannelRequest{Provider: ProviderGLM, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL: "https://open.bigmodel.cn/api/paas/v4",
			wantService: "openai",
		},
		{
			name:           "glm responses",
			req:            CreateChannelRequest{Provider: ProviderGLM, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://open.bigmodel.cn/api/paas/v4",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
		},
		{
			name:        "minimax chat",
			req:         CreateChannelRequest{Provider: ProviderMiniMax, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL: "https://api.minimax.chat/v1",
			wantService: "openai",
		},
		{
			name:           "minimax responses",
			req:            CreateChannelRequest{Provider: ProviderMiniMax, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://api.minimax.chat/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPayload(tt.req)
			if err != nil {
				t.Fatalf("BuildPayload() error = %v", err)
			}
			if got.BaseURL != tt.wantBaseURL {
				t.Fatalf("BaseURL = %q, want %q", got.BaseURL, tt.wantBaseURL)
			}
			if got.ServiceType != tt.wantService {
				t.Fatalf("ServiceType = %q, want %q", got.ServiceType, tt.wantService)
			}
			if got.NoVision != tt.wantVision {
				t.Fatalf("NoVision = %v, want %v", got.NoVision, tt.wantVision)
			}
			if got.PassbackReasoningContent != tt.wantPassback {
				t.Fatalf("PassbackReasoningContent = %v, want %v", got.PassbackReasoningContent, tt.wantPassback)
			}
			if got.CodexToolCompat != tt.wantCodex {
				t.Fatalf("CodexToolCompat = %v, want %v", got.CodexToolCompat, tt.wantCodex)
			}
			if got.StripCodexClientTools != tt.wantStripCodex {
				t.Fatalf("StripCodexClientTools = %v, want %v", got.StripCodexClientTools, tt.wantStripCodex)
			}
			if tt.wantModels != nil {
				if !slices.Equal(got.SupportedModels, tt.wantModels) {
					t.Fatalf("SupportedModels = %v, want %v", got.SupportedModels, tt.wantModels)
				}
			}
			for source, target := range tt.wantModelMap {
				if got.ModelMapping[source] != target {
					t.Fatalf("ModelMapping[%q] = %q, want %q; all mappings: %#v", source, got.ModelMapping[source], target, got.ModelMapping)
				}
			}
			if tt.wantFallback != "" {
				if got.VisionFallbackModel != tt.wantFallback {
					t.Fatalf("VisionFallbackModel = %q, want %q", got.VisionFallbackModel, tt.wantFallback)
				}
			}
		})
	}
}

func TestBuildPayloadRejectsUnsupportedTarget(t *testing.T) {
	_, err := BuildPayload(CreateChannelRequest{Provider: ProviderKimi, Target: TargetMessages, APIKey: "sk-test"})
	if err == nil {
		t.Fatal("BuildPayload() expected error for kimi+messages")
	}
}

func TestBestPlanForTarget(t *testing.T) {
	preset, _ := FindPreset(ProviderDeepSeek)
	tests := []struct {
		target string
		want   string
	}{
		{TargetMessages, "anthropic"},
		{TargetChat, "openai-chat"},
		{TargetResponses, "openai-chat"},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := bestPlanForTarget(preset, tt.target)
			if got != tt.want {
				t.Fatalf("bestPlanForTarget(deepseek, %s) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestBuildPayloadAutoCorrectsPlan(t *testing.T) {
	// 前端应在 target 变化时自动切换 plan，后端尊重显式 planID
	// 此测试验证：未指定 planID 时，chat target 自动选择 openai-chat plan
	got, err := BuildPayload(CreateChannelRequest{
		Provider: ProviderDeepSeek,
		Target:   TargetChat,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("BuildPayload() error = %v", err)
	}
	if got.BaseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("BaseURL = %q, want https://api.deepseek.com/v1", got.BaseURL)
	}
	if got.ServiceType != "openai" {
		t.Fatalf("ServiceType = %q, want openai", got.ServiceType)
	}
}
