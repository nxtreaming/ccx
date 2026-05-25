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
		wantNativeTool bool
		wantModels     []string
		wantModelMap   map[string]string
		wantReasoning  map[string]string
		wantFallback   string
		wantNormalize  bool
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
			name:          "deepseek chat (openai endpoint)",
			req:           CreateChannelRequest{Provider: ProviderDeepSeek, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://api.deepseek.com/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantVision:    true,
		},
		{
			name:           "deepseek responses (openai endpoint)",
			req:            CreateChannelRequest{Provider: ProviderDeepSeek, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://api.deepseek.com/v1",
			wantService:    "openai",
			wantVision:     true,
			wantNativeTool: true,
			wantNormalize:  true,
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
			wantModelMap: map[string]string{
				"haiku":  "mimo-v2.5-pro",
				"opus":   "mimo-v2.5-pro",
				"sonnet": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:          "mimo chat",
			req:           CreateChannelRequest{Provider: ProviderMiMo, Target: TargetChat, APIKey: "tp-test"},
			wantBaseURL:   "https://api.xiaomimimo.com/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantModelMap: map[string]string{
				"gpt-5": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:        "mimo responses",
			req:         CreateChannelRequest{Provider: ProviderMiMo, Target: TargetResponses, APIKey: "tp-test"},
			wantBaseURL: "https://api.xiaomimimo.com/v1",
			wantService: "openai",
			wantModelMap: map[string]string{
				"gpt-5": "mimo-v2.5-pro",
			},
			wantFallback: "mimo-v2.5",
		},
		{
			name:         "compshare messages",
			req:          CreateChannelRequest{Provider: ProviderCompshare, Target: TargetMessages, APIKey: "cs-test"},
			wantBaseURL:  "https://cp.compshare.cn",
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
			name:          "compshare chat",
			req:           CreateChannelRequest{Provider: ProviderCompshare, Target: TargetChat, APIKey: "cs-test"},
			wantBaseURL:   "https://cp.compshare.cn/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantVision:    true,
			wantModelMap: map[string]string{
				"gpt-5.4":      "deepseek-v4-pro",
				"gpt-5.4-mini": "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt-5.4": "max",
			},
		},
		{
			name:           "compshare responses",
			req:            CreateChannelRequest{Provider: ProviderCompshare, Target: TargetResponses, APIKey: "cs-test"},
			wantBaseURL:    "https://cp.compshare.cn/v1",
			wantService:    "openai",
			wantVision:     true,
			wantNativeTool: true,
			wantNormalize:  true,
			wantModelMap: map[string]string{
				"gpt":  "deepseek-v4-pro",
				"mini": "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt": "max",
			},
		},
		{
			name:          "kimi chat",
			req:           CreateChannelRequest{Provider: ProviderKimi, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://api.moonshot.cn/v1",
			wantService:   "openai",
			wantNormalize: true,
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
			name:          "glm chat",
			req:           CreateChannelRequest{Provider: ProviderGLM, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://open.bigmodel.cn/api/coding/paas/v4#",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:           "glm responses",
			req:            CreateChannelRequest{Provider: ProviderGLM, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://open.bigmodel.cn/api/coding/paas/v4#",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
		},
		{
			name:          "minimax chat",
			req:           CreateChannelRequest{Provider: ProviderMiniMax, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://api.minimax.chat/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantModelMap:  map[string]string{"gpt-5": "MiniMax-M2.7"},
		},
		{
			name:           "minimax responses",
			req:            CreateChannelRequest{Provider: ProviderMiniMax, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://api.minimax.chat/v1",
			wantService:    "openai",
			wantCodex:      false,
			wantStripCodex: false,
			wantNativeTool: true,
			wantModelMap:   map[string]string{"gpt-5": "MiniMax-M2.7"},
			wantNormalize:  true,
		},
		{
			name:          "dashscope chat",
			req:           CreateChannelRequest{Provider: ProviderDashScope, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantModelMap: map[string]string{
				"gpt-5.5":       "glm-5.1",
				"gpt-5.4":       "deepseek-v4-pro",
				"gpt-5.4-mini":  "deepseek-v4-flash",
				"gpt-5.3-codex": "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt-5.5":       "high",
				"gpt-5.4":       "max",
				"gpt-5.4-mini":  "high",
				"gpt-5.3-codex": "high",
			},
		},
		{
			name:           "dashscope responses",
			req:            CreateChannelRequest{Provider: ProviderDashScope, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap: map[string]string{
				"gpt-5.5":       "glm-5.1",
				"gpt-5.4":       "deepseek-v4-pro",
				"gpt-5.4-mini":  "deepseek-v4-flash",
				"gpt-5.3-codex": "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt-5.5":       "high",
				"gpt-5.4":       "max",
				"gpt-5.4-mini":  "high",
				"gpt-5.3-codex": "high",
			},
		},
		{
			name:          "opencode zen chat",
			req:           CreateChannelRequest{Provider: ProviderOpenCodeZen, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://opencode.ai/zen/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantModelMap:  map[string]string{"gpt-5": "glm-5.1"},
		},
		{
			name:          "opencode go chat",
			req:           CreateChannelRequest{Provider: ProviderOpenCodeGo, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://opencode.ai/zen/go/v1",
			wantService:   "openai",
			wantNormalize: true,
			wantModelMap:  map[string]string{"gpt-5": "glm-5.1"},
		},
		{
			name:           "opencode zen responses",
			req:            CreateChannelRequest{Provider: ProviderOpenCodeZen, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://opencode.ai/zen/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"gpt-5": "glm-5.1"},
		},
		{
			name:           "opencode go responses",
			req:            CreateChannelRequest{Provider: ProviderOpenCodeGo, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://opencode.ai/zen/go/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"gpt-5": "glm-5.1"},
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
			if got.CodexNativeToolPassthrough != tt.wantNativeTool {
				t.Fatalf("CodexNativeToolPassthrough = %v, want %v", got.CodexNativeToolPassthrough, tt.wantNativeTool)
			}
			if got.NormalizeNonstandardChatRoles != tt.wantNormalize {
				t.Fatalf("NormalizeNonstandardChatRoles = %v, want %v", got.NormalizeNonstandardChatRoles, tt.wantNormalize)
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
			for source, target := range tt.wantReasoning {
				if got.ReasoningMapping[source] != target {
					t.Fatalf("ReasoningMapping[%q] = %q, want %q; all mappings: %#v", source, got.ReasoningMapping[source], target, got.ReasoningMapping)
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
	// kimi 现已支持 Messages，此用例验证不支持的 target（如自定义拼接的错误值）仍能正确拒绝。
	_, err := BuildPayload(CreateChannelRequest{Provider: ProviderKimi, Target: "invalid-target", APIKey: "sk-test"})
	if err == nil {
		t.Fatal("BuildPayload() expected error for kimi+invalid-target")
	}
}

func TestBestPlanForTarget(t *testing.T) {
	tests := []struct {
		provider string
		target   string
		want     string
	}{
		{ProviderDeepSeek, TargetMessages, "anthropic"},
		{ProviderDeepSeek, TargetChat, "openai-chat"},
		{ProviderDeepSeek, TargetResponses, "openai-chat"},
		{ProviderCompshare, TargetMessages, "anthropic"},
		{ProviderCompshare, TargetChat, "openai-chat"},
		{ProviderCompshare, TargetResponses, "openai-chat"},
	}
	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.target, func(t *testing.T) {
			preset, ok := FindPreset(tt.provider)
			if !ok {
				t.Fatalf("FindPreset(%s) failed", tt.provider)
			}
			got := bestPlanForTarget(preset, tt.target)
			if got != tt.want {
				t.Fatalf("bestPlanForTarget(%s, %s) = %q, want %q", tt.provider, tt.target, got, tt.want)
			}
		})
	}
}

func TestPresetsIncludesCompshareAtThirdPosition(t *testing.T) {
	presets := Presets()
	if len(presets) < 3 {
		t.Fatalf("Presets() length = %d, want at least 3", len(presets))
	}
	if presets[2].ID != ProviderCompshare {
		t.Fatalf("Presets()[2].ID = %q, want %q", presets[2].ID, ProviderCompshare)
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
