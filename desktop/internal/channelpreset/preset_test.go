package channelpreset

import (
	"slices"
	"testing"
)

func TestBuildPayload(t *testing.T) {
	tests := []struct {
		name               string
		req                CreateChannelRequest
		wantBaseURL        string
		wantService        string
		wantVision         bool
		wantPassback       bool
		wantCodex          bool
		wantStripCodex     bool
		wantNativeTool     bool
		wantModels         []string
		wantModelMap       map[string]string
		wantNoModelMap     bool
		wantReasoning      map[string]string
		wantNoReasoningMap bool
		wantFallback       string
		wantNormalize      bool
		wantNoVisionModels []string
	}{
		{
			name:         "deepseek messages (anthropic endpoint)",
			req:          CreateChannelRequest{Provider: ProviderDeepSeek, Target: TargetMessages, APIKey: "sk-test"},
			wantBaseURL:  "https://api.deepseek.com/anthropic",
			wantService:  "claude",
			wantVision:   true,
			wantPassback: true,
			wantModelMap: map[string]string{
				"fable":  "deepseek-v4-pro",
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
				"codex": "deepseek-v4-flash",
				"gpt":   "deepseek-v4-pro",
				"mini":  "deepseek-v4-flash",
			},
		},
		{
			name:         "mimo messages (token plan)",
			req:          CreateChannelRequest{Provider: ProviderMiMo, Target: TargetMessages, PlanID: "token-sgp-anthropic", APIKey: "tp-test"},
			wantBaseURL:  "https://token-plan-sgp.xiaomimimo.com/anthropic",
			wantService:  "claude",
			wantPassback: true,
			wantModelMap: map[string]string{
				"fable":  "mimo-v2.5-pro",
				"haiku":  "mimo-v2.5-pro",
				"opus":   "mimo-v2.5-pro",
				"sonnet": "mimo-v2.5-pro",
			},
			wantNoVisionModels: []string{"mimo-v2.5-pro"},
			wantFallback:       "mimo-v2.5",
		},
		{
			name:         "mimo messages (auto plan)",
			req:          CreateChannelRequest{Provider: ProviderMiMo, Target: TargetMessages, APIKey: "tp-test"},
			wantBaseURL:  "https://api.xiaomimimo.com/anthropic",
			wantService:  "claude",
			wantPassback: true,
			wantModelMap: map[string]string{
				"fable":  "mimo-v2.5-pro",
				"haiku":  "mimo-v2.5-pro",
				"opus":   "mimo-v2.5-pro",
				"sonnet": "mimo-v2.5-pro",
			},
			wantNoVisionModels: []string{"mimo-v2.5-pro"},
			wantFallback:       "mimo-v2.5",
		},
		{
			name:               "mimo chat",
			req:                CreateChannelRequest{Provider: ProviderMiMo, Target: TargetChat, APIKey: "tp-test"},
			wantBaseURL:        "https://api.xiaomimimo.com/v1",
			wantService:        "openai",
			wantNormalize:      true,
			wantNoVisionModels: []string{"mimo-v2.5-pro"},
			wantFallback:       "mimo-v2.5",
		},
		{
			name:        "mimo responses",
			req:         CreateChannelRequest{Provider: ProviderMiMo, Target: TargetResponses, APIKey: "tp-test"},
			wantBaseURL: "https://api.xiaomimimo.com/v1",
			wantService: "openai",
			wantModelMap: map[string]string{
				"codex": "mimo-v2.5",
				"gpt":   "mimo-v2.5-pro",
			},
			wantNoVisionModels: []string{"mimo-v2.5-pro"},
			wantFallback:       "mimo-v2.5",
		},
		{
			name:         "compshare messages",
			req:          CreateChannelRequest{Provider: ProviderCompshare, Target: TargetMessages, APIKey: "cs-test"},
			wantBaseURL:  "https://cp.compshare.cn",
			wantService:  "claude",
			wantVision:   false,
			wantPassback: true,
			wantModelMap: map[string]string{
				"fable":  "glm-5.1",
				"haiku":  "deepseek-v4-flash",
				"opus":   "glm-5.1",
				"sonnet": "glm-5.1",
			},
			wantNoVisionModels: []string{"deepseek-v4-flash"},
			wantFallback:       "MiniMax-M2.7",
		},
		{
			name:               "compshare chat",
			req:                CreateChannelRequest{Provider: ProviderCompshare, Target: TargetChat, APIKey: "cs-test"},
			wantBaseURL:        "https://cp.compshare.cn/v1",
			wantService:        "openai",
			wantNormalize:      true,
			wantVision:         false,
			wantNoVisionModels: []string{"deepseek-v4-flash"},
			wantFallback:       "MiniMax-M2.7",
		},
		{
			name:           "compshare responses",
			req:            CreateChannelRequest{Provider: ProviderCompshare, Target: TargetResponses, APIKey: "cs-test"},
			wantBaseURL:    "https://cp.compshare.cn/v1",
			wantService:    "openai",
			wantVision:     false,
			wantNativeTool: true,
			wantNormalize:  true,
			wantModelMap: map[string]string{
				"codex": "deepseek-v4-flash",
				"gpt":   "glm-5.1",
				"mini":  "deepseek-v4-flash",
			},
			wantNoVisionModels: []string{"deepseek-v4-flash"},
			wantFallback:       "MiniMax-M2.7",
		},
		{
			name:           "runapi messages",
			req:            CreateChannelRequest{Provider: ProviderRunAPI, Target: TargetMessages, APIKey: "runapi-test"},
			wantBaseURL:    "https://runapi.co/v1",
			wantService:    "claude",
			wantNoModelMap: true,
			wantNormalize:  false,
		},
		{
			name:           "runapi chat",
			req:            CreateChannelRequest{Provider: ProviderRunAPI, Target: TargetChat, APIKey: "runapi-test"},
			wantBaseURL:    "https://runapi.co/v1",
			wantService:    "openai",
			wantNormalize:  true,
			wantNoModelMap: true,
		},
		{
			name:               "runapi responses",
			req:                CreateChannelRequest{Provider: ProviderRunAPI, Target: TargetResponses, APIKey: "runapi-test"},
			wantBaseURL:        "https://runapi.co/v1",
			wantService:        "responses",
			wantCodex:          false,
			wantStripCodex:     false,
			wantNoModelMap:     true,
			wantNoReasoningMap: true,
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
			name:           "kimi coding plan responses",
			req:            CreateChannelRequest{Provider: ProviderKimi, Target: TargetResponses, PlanID: "coding-openai-chat", APIKey: "sk-test"},
			wantBaseURL:    "https://api.kimi.com/coding/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"codex": "kimi-for-coding", "gpt": "kimi-for-coding"},
		},
		{
			name:        "kimi coding plan messages",
			req:         CreateChannelRequest{Provider: ProviderKimi, Target: TargetMessages, PlanID: "coding-anthropic", APIKey: "sk-test"},
			wantBaseURL: "https://api.kimi.com/coding",
			wantService: "claude",
			wantModelMap: map[string]string{
				"fable":  "kimi-for-coding",
				"haiku":  "kimi-for-coding",
				"opus":   "kimi-for-coding",
				"sonnet": "kimi-for-coding",
			},
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
		},
		{
			name:           "minimax responses",
			req:            CreateChannelRequest{Provider: ProviderMiniMax, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://api.minimax.chat/v1",
			wantService:    "openai",
			wantCodex:      false,
			wantStripCodex: false,
			wantNativeTool: true,
			wantModelMap:   map[string]string{"codex": "MiniMax-M2.7", "gpt": "MiniMax-M3", "mini": "MiniMax-M2.7"},
			wantNormalize:  true,
		},
		{
			name:          "dashscope chat",
			req:           CreateChannelRequest{Provider: ProviderDashScope, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:           "dashscope responses",
			req:            CreateChannelRequest{Provider: ProviderDashScope, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap: map[string]string{
				"codex": "deepseek-v4-flash",
				"gpt":   "deepseek-v4-pro",
				"mini":  "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt": "max",
			},
		},
		{
			name:           "dashscope coding plan responses",
			req:            CreateChannelRequest{Provider: ProviderDashScope, Target: TargetResponses, PlanID: "coding-openai-chat", APIKey: "sk-sp-test"},
			wantBaseURL:    "https://coding.dashscope.aliyuncs.com/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap: map[string]string{
				"codex": "deepseek-v4-flash",
				"gpt":   "deepseek-v4-pro",
				"mini":  "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt": "max",
			},
		},
		{
			name:        "dashscope coding plan messages",
			req:         CreateChannelRequest{Provider: ProviderDashScope, Target: TargetMessages, PlanID: "coding-anthropic", APIKey: "sk-sp-test"},
			wantBaseURL: "https://coding.dashscope.aliyuncs.com/apps/anthropic",
			wantService: "claude",
		},
		{
			name:          "dashscope coding plan chat",
			req:           CreateChannelRequest{Provider: ProviderDashScope, Target: TargetChat, PlanID: "coding-openai-chat", APIKey: "sk-sp-test"},
			wantBaseURL:   "https://coding.dashscope.aliyuncs.com/v1",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:        "dashscope token plan messages",
			req:         CreateChannelRequest{Provider: ProviderDashScope, Target: TargetMessages, PlanID: "token-plan-anthropic", APIKey: "sk-tp-test"},
			wantBaseURL: "https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic",
			wantService: "claude",
		},
		{
			name:          "dashscope token plan chat",
			req:           CreateChannelRequest{Provider: ProviderDashScope, Target: TargetChat, PlanID: "token-plan-openai-chat", APIKey: "sk-tp-test"},
			wantBaseURL:   "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:           "dashscope token plan responses",
			req:            CreateChannelRequest{Provider: ProviderDashScope, Target: TargetResponses, PlanID: "token-plan-openai-chat", APIKey: "sk-tp-test"},
			wantBaseURL:    "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap: map[string]string{
				"codex": "deepseek-v4-flash",
				"gpt":   "deepseek-v4-pro",
				"mini":  "deepseek-v4-flash",
			},
			wantReasoning: map[string]string{
				"gpt": "max",
			},
		},
		{
			name:          "opencode zen chat",
			req:           CreateChannelRequest{Provider: ProviderOpenCodeZen, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://opencode.ai/zen/v1",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:          "opencode go chat",
			req:           CreateChannelRequest{Provider: ProviderOpenCodeGo, Target: TargetChat, APIKey: "sk-test"},
			wantBaseURL:   "https://opencode.ai/zen/go/v1",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:           "opencode zen responses",
			req:            CreateChannelRequest{Provider: ProviderOpenCodeZen, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://opencode.ai/zen/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"codex": "glm-5.1", "gpt": "glm-5.1"},
		},
		{
			name:           "opencode go responses",
			req:            CreateChannelRequest{Provider: ProviderOpenCodeGo, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://opencode.ai/zen/go/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"codex": "glm-5.1", "gpt": "glm-5.1"},
		},
		{
			name:           "kimi responses auto-review redirect",
			req:            CreateChannelRequest{Provider: ProviderKimi, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://api.moonshot.cn/v1",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"codex": "kimi-k2.7", "gpt": "kimi-k2.7"},
		},
		{
			name:           "glm responses auto-review redirect",
			req:            CreateChannelRequest{Provider: ProviderGLM, Target: TargetResponses, APIKey: "sk-test"},
			wantBaseURL:    "https://open.bigmodel.cn/api/coding/paas/v4#",
			wantService:    "openai",
			wantCodex:      true,
			wantStripCodex: true,
			wantModelMap:   map[string]string{"codex": "glm-5.1", "gpt": "glm-5.1"},
		},
		{
			name:         "volc-ark messages (anthropic endpoint)",
			req:          CreateChannelRequest{Provider: ProviderVolcArk, Target: TargetMessages, APIKey: "ark-test"},
			wantBaseURL:  "https://ark.cn-beijing.volces.com/api/coding#",
			wantService:  "claude",
			wantPassback: true,
			wantModelMap: map[string]string{
				"fable":  "minimax-m3",
				"haiku":  "deepseek-v4-flash",
				"opus":   "minimax-m3",
				"sonnet": "minimax-m3",
			},
			wantNoVisionModels: []string{"deepseek-v4-flash"},
			wantFallback:       "minimax-m2.7",
		},
		{
			name:          "volc-ark chat",
			req:           CreateChannelRequest{Provider: ProviderVolcArk, Target: TargetChat, APIKey: "ark-test"},
			wantBaseURL:   "https://ark.cn-beijing.volces.com/api/coding/v3",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:        "volc-ark responses",
			req:         CreateChannelRequest{Provider: ProviderVolcArk, Target: TargetResponses, APIKey: "ark-test"},
			wantBaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3",
			wantService: "openai",
			wantModelMap: map[string]string{
				"codex": "deepseek-v4-flash",
				"gpt":   "minimax-m3",
				"mini":  "deepseek-v4-flash",
			},
			wantNormalize:      true,
			wantNativeTool:     true,
			wantNoVisionModels: []string{"deepseek-v4-flash"},
			wantFallback:       "minimax-m2.7",
		},
		{
			name:        "qianfan messages (anthropic endpoint)",
			req:         CreateChannelRequest{Provider: ProviderQianfan, Target: TargetMessages, APIKey: "qf-test"},
			wantBaseURL: "https://qianfan.baidubce.com/anthropic/coding",
			wantService: "claude",
			wantModelMap: map[string]string{
				"fable":  "qianfan-code-latest",
				"haiku":  "qianfan-code-latest",
				"opus":   "qianfan-code-latest",
				"sonnet": "qianfan-code-latest",
			},
		},
		{
			name:          "qianfan chat",
			req:           CreateChannelRequest{Provider: ProviderQianfan, Target: TargetChat, APIKey: "qf-test"},
			wantBaseURL:   "https://qianfan.baidubce.com/v2/coding#",
			wantService:   "openai",
			wantNormalize: true,
		},
		{
			name:        "qianfan responses",
			req:         CreateChannelRequest{Provider: ProviderQianfan, Target: TargetResponses, APIKey: "qf-test"},
			wantBaseURL: "https://qianfan.baidubce.com/v2/coding#",
			wantService: "openai",
			wantModelMap: map[string]string{
				"codex": "qianfan-code-latest",
				"gpt":   "qianfan-code-latest",
				"mini":  "qianfan-code-latest",
			},
			wantNormalize:  true,
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
			if tt.wantNoModelMap && len(got.ModelMapping) != 0 {
				t.Fatalf("ModelMapping = %#v, want empty", got.ModelMapping)
			}
			for source, target := range tt.wantReasoning {
				if got.ReasoningMapping[source] != target {
					t.Fatalf("ReasoningMapping[%q] = %q, want %q; all mappings: %#v", source, got.ReasoningMapping[source], target, got.ReasoningMapping)
				}
			}
			if tt.wantNoReasoningMap && len(got.ReasoningMapping) != 0 {
				t.Fatalf("ReasoningMapping = %#v, want empty", got.ReasoningMapping)
			}
			if tt.wantNoVisionModels != nil {
				if !slices.Equal(got.NoVisionModels, tt.wantNoVisionModels) {
					t.Fatalf("NoVisionModels = %v, want %v", got.NoVisionModels, tt.wantNoVisionModels)
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

func TestChannelTargetConfigsExcludeRetiredGPTModels(t *testing.T) {
	retiredModels := map[string]struct{}{
		"gpt-5.2":       {},
		"gpt-5.2-codex": {},
		"gpt-5.3-codex": {},
	}

	assertActiveModel := func(field string, provider string, target string, model string) {
		if _, ok := retiredModels[model]; ok {
			t.Fatalf("%s for %s/%s contains retired model %q", field, provider, target, model)
		}
	}

	for target, providerConfigs := range channelTargetConfigs {
		for provider, config := range providerConfigs {
			for source, mapped := range config.ModelMapping {
				assertActiveModel("ModelMapping source", provider, target, source)
				assertActiveModel("ModelMapping target", provider, target, mapped)
			}
			for source := range config.ReasoningMapping {
				assertActiveModel("ReasoningMapping source", provider, target, source)
			}
		}
	}
}

func TestResponsesTargetMustIncludeCodexMapping(t *testing.T) {
	responsesConfigs, ok := channelTargetConfigs[TargetResponses]
	if !ok {
		t.Fatal("channelTargetConfigs[TargetResponses] not found")
	}
	for provider, config := range responsesConfigs {
		if len(config.ModelMapping) == 0 {
			continue
		}
		if _, found := config.ModelMapping["codex"]; !found {
			t.Fatalf("provider %q responses config missing codex mapping", provider)
		}
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
		{ProviderRunAPI, TargetMessages, "anthropic"},
		{ProviderRunAPI, TargetChat, "openai-chat"},
		{ProviderRunAPI, TargetResponses, "openai-chat"},
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

func TestPresetsIncludesRunAPIAfterCompshare(t *testing.T) {
	presets := Presets()
	if len(presets) < 4 {
		t.Fatalf("Presets() length = %d, want at least 4", len(presets))
	}
	if presets[3].ID != ProviderRunAPI {
		t.Fatalf("Presets()[3].ID = %q, want %q", presets[3].ID, ProviderRunAPI)
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

func TestBuildPayloadSetsProviderConsoleWebsite(t *testing.T) {
	got, err := BuildPayload(CreateChannelRequest{
		Provider: ProviderCompshare,
		Target:   TargetResponses,
		APIKey:   "cs-test",
	})
	if err != nil {
		t.Fatalf("BuildPayload() error = %v", err)
	}
	want := "https://console.compshare.cn/light-gpu/model-manage"
	if got.Website != want {
		t.Fatalf("Website = %q, want %q", got.Website, want)
	}
}

func TestBuildPayloadSetsRunAPIWebsite(t *testing.T) {
	got, err := BuildPayload(CreateChannelRequest{
		Provider: ProviderRunAPI,
		Target:   TargetResponses,
		APIKey:   "runapi-test",
	})
	if err != nil {
		t.Fatalf("BuildPayload() error = %v", err)
	}
	want := "https://runapi.co/console"
	if got.Website != want {
		t.Fatalf("Website = %q, want %q", got.Website, want)
	}
}
