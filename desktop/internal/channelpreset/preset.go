package channelpreset

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	ProviderDeepSeek    = "deepseek"
	ProviderMiMo        = "mimo"
	ProviderCompshare   = "compshare"
	ProviderKimi        = "kimi"
	ProviderGLM         = "glm"
	ProviderMiniMax     = "minimax"
	ProviderDashScope   = "dashscope"
	ProviderOpenCodeZen = "opencode-zen"
	ProviderOpenCodeGo  = "opencode-go"

	TargetMessages  = "messages"
	TargetChat      = "chat"
	TargetResponses = "responses"
)

type ProviderPreset struct {
	ID                  string          `json:"id"`
	Label               string          `json:"label"`
	Description         string          `json:"description"`
	DirectAgent         bool            `json:"directAgent"`
	NativeMessages      bool            `json:"nativeMessages"`
	ChatCompatible      bool            `json:"chatCompatible"`
	ResponsesCompatible bool            `json:"responsesCompatible"`
	Plans               []ProviderPlan  `json:"plans"`
	Targets             []ChannelTarget `json:"targets"`
	DefaultTarget       string          `json:"defaultTarget"`
}

type ProviderPlan struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	BaseURL     string `json:"baseUrl"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended"`
	Custom      bool   `json:"custom"`
}

// Protocol 返回 plan 所属的协议类型。
func (p ProviderPlan) Protocol() string {
	id := strings.ToLower(p.ID)
	label := strings.ToLower(p.Label)
	baseURL := strings.TrimRight(strings.ToLower(p.BaseURL), "/")
	if strings.Contains(id, "anthropic") || strings.Contains(label, "anthropic") || strings.Contains(baseURL, "anthropic") || strings.HasSuffix(baseURL, "/messages") {
		return "anthropic"
	}
	return "openai"
}

type ChannelTarget struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended"`
}

type CreateChannelRequest struct {
	Provider    string `json:"provider"`
	Target      string `json:"target"`
	PlanID      string `json:"planId"`
	BaseURL     string `json:"baseUrl"`
	APIKey      string `json:"apiKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateChannelResult struct {
	Provider string `json:"provider"`
	Target   string `json:"target"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	Message  string `json:"message"`
}

type ChannelPayload struct {
	Name                          string            `json:"name"`
	Description                   string            `json:"description,omitempty"`
	Website                       string            `json:"website,omitempty"`
	ServiceType                   string            `json:"serviceType"`
	BaseURL                       string            `json:"baseUrl"`
	APIKeys                       []string          `json:"apiKeys"`
	ModelMapping                  map[string]string `json:"modelMapping,omitempty"`
	ReasoningMapping              map[string]string `json:"reasoningMapping,omitempty"`
	ReasoningParamStyle           string            `json:"reasoningParamStyle,omitempty"`
	PassbackReasoningContent      bool              `json:"passbackReasoningContent,omitempty"`
	PassbackThinkingBlocks        bool              `json:"passbackThinkingBlocks,omitempty"`
	NoVision                      bool              `json:"noVision,omitempty"`
	NoVisionModels                []string          `json:"noVisionModels,omitempty"`
	VisionFallbackModel           string            `json:"visionFallbackModel,omitempty"`
	SupportedModels               []string          `json:"supportedModels,omitempty"`
	StripEmptyTextBlocks          bool              `json:"stripEmptyTextBlocks,omitempty"`
	InjectDummyThoughtSignature   bool              `json:"injectDummyThoughtSignature,omitempty"`
	StripThoughtSignature         bool              `json:"stripThoughtSignature,omitempty"`
	CodexNativeToolPassthrough    bool              `json:"codexNativeToolPassthrough,omitempty"`
	CodexToolCompat               bool              `json:"codexToolCompat,omitempty"`
	StripCodexClientTools         bool              `json:"stripCodexClientTools,omitempty"`
	NormalizeNonstandardChatRoles bool              `json:"normalizeNonstandardChatRoles,omitempty"`
	Priority                      int               `json:"priority,omitempty"`
	Status                        string            `json:"status,omitempty"`
}

func Presets() []ProviderPreset {
	return []ProviderPreset{
		{
			ID:                  ProviderDeepSeek,
			Label:               "DeepSeek",
			Description:         "Messages 原生透传、Codex Responses、Chat 渠道透传三种用法。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://api.deepseek.com/anthropic", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "OpenAI-compatible", BaseURL: "https://api.deepseek.com/v1", Description: "Chat / Responses 通用入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderMiMo,
			Label:               "MiMo",
			Description:         "Messages 原生透传、Codex Responses、Chat 渠道透传；内置按量与 token plan 入口。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "按量 - Anthropic 入口", BaseURL: "https://api.xiaomimimo.com/anthropic", Description: "Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "按量 - OpenAI 入口", BaseURL: "https://api.xiaomimimo.com/v1", Description: "Chat / Responses 通用入口"},
				{ID: "token-cn", Label: "Token Plan - 中国", BaseURL: "https://token-plan-cn.xiaomimimo.com/v1", Description: "中国区订阅套餐"},
				{ID: "token-sgp", Label: "Token Plan - 新加坡", BaseURL: "https://token-plan-sgp.xiaomimimo.com/v1", Description: "新加坡区订阅套餐"},
				{ID: "token-ams", Label: "Token Plan - 欧洲", BaseURL: "https://token-plan-ams.xiaomimimo.com/v1", Description: "欧洲区订阅套餐"},
				{ID: "token-cn-anthropic", Label: "Token Plan - 中国 (Anthropic)", BaseURL: "https://token-plan-cn.xiaomimimo.com/anthropic", Description: "中国区订阅套餐 Anthropic 入口"},
				{ID: "token-sgp-anthropic", Label: "Token Plan - 新加坡 (Anthropic)", BaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic", Description: "新加坡区订阅套餐 Anthropic 入口"},
				{ID: "token-ams-anthropic", Label: "Token Plan - 欧洲 (Anthropic)", BaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic", Description: "欧洲区订阅套餐 Anthropic 入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "自动开启 reasoning passback 兼容", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderCompshare,
			Label:               "Compshare 套餐",
			Description:         "独立套餐 BaseURL 与 API Key，兼容 Anthropic Messages、OpenAI Chat 与 Codex Responses。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://cp.compshare.cn", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "OpenAI-compatible", BaseURL: "https://cp.compshare.cn/v1", Description: "OpenAI Chat / Responses 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderKimi,
			Label:               "Kimi / Moonshot",
			Description:         "Messages 原生透传、Codex Responses、Chat 渠道透传三种用法。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://api.moonshot.cn/anthropic", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "OpenAI-compatible", BaseURL: "https://api.moonshot.cn/v1", Description: "Moonshot OpenAI 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderGLM,
			Label:               "GLM / BigModel",
			Description:         "Messages 原生透传、Codex Responses、Chat 渠道透传三种用法。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://open.bigmodel.cn/api/anthropic", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "coding", Label: "OpenAI-compatible (Coding)", BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4#", Description: "智谱 Coding 套餐入口"},
				{ID: "openai-chat", Label: "OpenAI-compatible (通用)", BaseURL: "https://open.bigmodel.cn/api/paas/v4#", Description: "智谱通用 OpenAI 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderMiniMax,
			Label:               "MiniMax",
			Description:         "Messages 原生透传、Codex Responses、Chat 渠道透传三种用法。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://api.minimaxi.com/anthropic", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "OpenAI-compatible", BaseURL: "https://api.minimax.chat/v1", Description: "MiniMax OpenAI 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderDashScope,
			Label:               "阿里云 DashScope",
			Description:         "Messages 原生透传、Codex Responses、Chat 渠道透传三种用法。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "按量 - Anthropic 入口", BaseURL: "https://dashscope.aliyuncs.com/apps/anthropic", Description: "Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "按量 - OpenAI 入口", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Description: "Chat / Responses 通用入口"},
				{ID: "coding-anthropic", Label: "Coding Plan (Anthropic)", BaseURL: "https://coding.dashscope.aliyuncs.com/apps/anthropic", Description: "订阅套餐 Messages 入口"},
				{ID: "coding-openai-chat", Label: "Coding Plan (OpenAI)", BaseURL: "https://coding.dashscope.aliyuncs.com/v1", Description: "订阅套餐 OpenAI 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderOpenCodeZen,
			Label:               "OpenCode Zen",
			Description:         "按量付费精选模型网关，支持 Messages、Chat、Responses 三种协议。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://opencode.ai/zen/v1/messages", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "OpenAI-compatible", BaseURL: "https://opencode.ai/zen/v1", Description: "OpenCode Zen OpenAI 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
		{
			ID:                  ProviderOpenCodeGo,
			Label:               "OpenCode Go",
			Description:         "低成本开源编程模型订阅服务（$5/月起），支持 Messages、Chat、Responses 三种协议。",
			DirectAgent:         true,
			NativeMessages:      true,
			ChatCompatible:      true,
			ResponsesCompatible: true,
			Plans: []ProviderPlan{
				{ID: "anthropic", Label: "Anthropic-compatible", BaseURL: "https://opencode.ai/zen/go/v1/messages", Description: "Claude Messages 原生入口", Recommended: true},
				{ID: "openai-chat", Label: "OpenAI-compatible", BaseURL: "https://opencode.ai/zen/go/v1", Description: "OpenCode Go OpenAI 兼容入口"},
			},
			Targets: []ChannelTarget{
				{Type: TargetMessages, Label: "Messages 原生透传", Description: "Claude Code 直连或 CCX messages 渠道", Recommended: true},
				{Type: TargetResponses, Label: "Codex Responses", Description: "OpenAI Responses 协议，供 Codex 使用"},
				{Type: TargetChat, Label: "Chat 渠道透传", Description: "OpenAI Chat 协议，供 Chat 客户端使用"},
			},
			DefaultTarget: TargetMessages,
		},
	}
}

func BuildPayload(req CreateChannelRequest) (ChannelPayload, error) {
	preset, ok := FindPreset(req.Provider)
	if !ok {
		return ChannelPayload{}, fmt.Errorf("不支持的 provider: %s", req.Provider)
	}
	target := strings.TrimSpace(req.Target)
	if target == "" {
		target = preset.DefaultTarget
	}
	if !supportsTarget(preset, target) {
		return ChannelPayload{}, fmt.Errorf("%s 不支持添加到 %s 渠道", preset.Label, target)
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return ChannelPayload{}, fmt.Errorf("API Key 不能为空")
	}
	planID := strings.TrimSpace(req.PlanID)
	if planID == "" {
		planID = bestPlanForTarget(preset, target)
	}
	baseURL, err := ResolveBaseURL(preset, planID, req.BaseURL)
	if err != nil {
		return ChannelPayload{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = defaultChannelName(preset.ID, target)
	}
	payload := ChannelPayload{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		BaseURL:     baseURL,
		APIKeys:     []string{apiKey},
		Priority:    1,
		Status:      "active",
	}
	applyTargetDefaults(&payload, preset.ID, target)
	return payload, nil
}

// targetMatchesURL 判断 URL 是否与 target 协议兼容。
// messages target 使用 Anthropic 协议，需要 anthropic 路径；
// chat/responses target 使用 OpenAI 协议，不应使用 anthropic 路径。

// bestPlanForTarget 根据 target 自动选择最合适的 plan。
// 当 provider 有多个 plan（如 DeepSeek 的 /anthropic 和 /v1）时，
// 确保 chat/responses 选择 OpenAI-compatible 入口，messages 选择 Anthropic 入口。
func bestPlanForTarget(preset ProviderPreset, target string) string {
	if len(preset.Plans) == 0 {
		return ""
	}
	if len(preset.Plans) == 1 {
		return preset.Plans[0].ID
	}
	for _, plan := range preset.Plans {
		if !plan.Custom {
			isAnthropic := plan.Protocol() == "anthropic"
			if target == TargetMessages && isAnthropic {
				return plan.ID
			}
			if (target == TargetChat || target == TargetResponses) && !isAnthropic {
				return plan.ID
			}
		}
	}
	for _, plan := range preset.Plans {
		if plan.Recommended {
			return plan.ID
		}
	}
	return preset.Plans[0].ID
}

func FindPreset(provider string) (ProviderPreset, bool) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	for _, preset := range Presets() {
		if preset.ID == provider {
			return preset, true
		}
	}
	return ProviderPreset{}, false
}

// FilterPlansForTarget 按 target 协议过滤 plans，只返回兼容的入口。
// messages 使用 Anthropic 协议，chat/responses 使用 OpenAI 协议。
// 自定义 plan 始终保留。
func FilterPlansForTarget(plans []ProviderPlan, target string) []ProviderPlan {
	if len(plans) <= 1 {
		return plans
	}
	wantAnthropic := target == TargetMessages
	var filtered []ProviderPlan
	for _, plan := range plans {
		isAnthropic := plan.Protocol() == "anthropic"
		if wantAnthropic == isAnthropic {
			filtered = append(filtered, plan)
		}
	}
	if len(filtered) == 0 {
		return plans
	}
	return filtered
}

func ResolveBaseURL(preset ProviderPreset, planID string, customBaseURL string) (string, error) {
	customBaseURL = strings.TrimSpace(customBaseURL)
	planID = strings.TrimSpace(planID)
	if planID == "" {
		for _, plan := range preset.Plans {
			if plan.Recommended {
				planID = plan.ID
				break
			}
		}
	}
	if planID == "" && len(preset.Plans) > 0 {
		planID = preset.Plans[0].ID
	}
	for _, plan := range preset.Plans {
		if plan.ID != planID {
			continue
		}
		if plan.Custom {
			if customBaseURL == "" {
				return "", fmt.Errorf("自定义 plan 需要填写 Base URL")
			}
			return customBaseURL, nil
		}
		if customBaseURL != "" {
			return customBaseURL, nil
		}
		if plan.BaseURL == "" {
			return "", fmt.Errorf("plan %s 缺少 Base URL", plan.ID)
		}
		return plan.BaseURL, nil
	}
	if customBaseURL != "" {
		return customBaseURL, nil
	}
	return "", fmt.Errorf("未找到 provider %s 的 plan: %s", preset.ID, planID)
}

func supportsTarget(preset ProviderPreset, target string) bool {
	for _, item := range preset.Targets {
		if item.Type == target {
			return true
		}
	}
	return false
}

func defaultChannelName(provider string, target string) string {
	return fmt.Sprintf("desktop-%s-%s", provider, target)
}

type channelTargetConfig struct {
	ModelMapping                  map[string]string
	ReasoningMapping              map[string]string
	ReasoningParamStyle           string
	PassbackReasoningContent      bool
	PassbackThinkingBlocks        bool
	NoVision                      bool
	NoVisionModels                []string
	VisionFallbackModel           string
	StripEmptyTextBlocks          bool
	CodexNativeToolPassthrough    bool
	CodexToolCompat               *bool
	StripCodexClientTools         *bool
	NormalizeNonstandardChatRoles bool
}

var channelTargetConfigs = map[string]map[string]channelTargetConfig{
	TargetMessages: {
		ProviderDeepSeek: {
			ModelMapping: map[string]string{
				"haiku":  "deepseek-v4-flash",
				"opus":   "deepseek-v4-pro",
				"sonnet": "deepseek-v4-pro",
			},
			ReasoningParamStyle:      "reasoning",
			PassbackReasoningContent: true,
			PassbackThinkingBlocks:   true,
			NoVision:                 true,
		},
		ProviderMiMo: {
			ModelMapping: map[string]string{
				"haiku":  "mimo-v2.5-pro",
				"opus":   "mimo-v2.5-pro",
				"sonnet": "mimo-v2.5-pro",
			},
			ReasoningParamStyle:      "reasoning",
			PassbackReasoningContent: true,
			NoVisionModels:           []string{"mimo-v2.5-pro"},
			VisionFallbackModel:      "mimo-v2.5",
		},
		ProviderCompshare: {
			ModelMapping: map[string]string{
				"haiku":  "deepseek-v4-flash",
				"opus":   "glm-5.1",
				"sonnet": "glm-5.1",
			},
			ReasoningParamStyle:      "reasoning",
			PassbackReasoningContent: true,
			NoVisionModels:           []string{"deepseek-v4-flash"},
			VisionFallbackModel:      "MiniMax-M2.7",
		},
		ProviderKimi: {
			ModelMapping: map[string]string{
				"haiku":  "kimi-k2.6",
				"opus":   "kimi-k2.6",
				"sonnet": "kimi-k2.6",
			},
		},
		ProviderGLM: {
			ModelMapping: map[string]string{
				"haiku":  "glm-5.1",
				"opus":   "glm-5.1",
				"sonnet": "glm-5.1",
			},
		},
		ProviderMiniMax: {
			ModelMapping: map[string]string{
				"haiku":  "MiniMax-M2.7",
				"opus":   "MiniMax-M2.7",
				"sonnet": "MiniMax-M2.7",
			},
			PassbackReasoningContent: true,
		},
		ProviderDashScope: {
			ModelMapping: map[string]string{
				"haiku":  "glm-5.1",
				"opus":   "glm-5.1",
				"sonnet": "glm-5.1",
			},
		},
		ProviderOpenCodeZen: {
			ModelMapping: map[string]string{
				"haiku":  "glm-5.1",
				"opus":   "glm-5.1",
				"sonnet": "glm-5.1",
			},
		},
		ProviderOpenCodeGo: {
			ModelMapping: map[string]string{
				"haiku":  "glm-5.1",
				"opus":   "glm-5.1",
				"sonnet": "glm-5.1",
			},
		},
	},
	TargetChat: {
		ProviderDeepSeek: {
			ReasoningParamStyle: "reasoning",
			NoVision:            true,
		},
		ProviderMiMo: {
			ReasoningParamStyle: "reasoning",
			NoVisionModels:      []string{"mimo-v2.5-pro"},
			VisionFallbackModel: "mimo-v2.5",
		},
		ProviderCompshare: {
			ReasoningParamStyle: "reasoning",
			NoVisionModels:      []string{"deepseek-v4-flash"},
			VisionFallbackModel: "MiniMax-M2.7",
		},
		ProviderMiniMax:     {},
		ProviderDashScope:   {},
		ProviderOpenCodeZen: {},
		ProviderOpenCodeGo:  {},
	},
	TargetResponses: {
		ProviderDeepSeek: {
			ModelMapping: map[string]string{
				"gpt":  "deepseek-v4-pro",
				"mini": "deepseek-v4-flash",
			},
			ReasoningMapping:              map[string]string{"gpt": "max"},
			ReasoningParamStyle:           "reasoning",
			CodexToolCompat:               boolRef(false),
			StripCodexClientTools:         boolRef(false),
			CodexNativeToolPassthrough:    true,
			NormalizeNonstandardChatRoles: true,
			NoVision:                      true,
		},
		ProviderMiMo: {
			ModelMapping:          map[string]string{"gpt-5": "mimo-v2.5-pro"},
			ReasoningParamStyle:   "reasoning",
			CodexToolCompat:       boolRef(false),
			StripCodexClientTools: boolRef(false),
			NoVisionModels:        []string{"mimo-v2.5-pro"},
			VisionFallbackModel:   "mimo-v2.5",
		},
		ProviderCompshare: {
			ModelMapping: map[string]string{
				"gpt":  "glm-5.1",
				"mini": "deepseek-v4-flash",
			},
			ReasoningParamStyle:           "reasoning",
			CodexToolCompat:               boolRef(false),
			StripCodexClientTools:         boolRef(false),
			CodexNativeToolPassthrough:    true,
			NormalizeNonstandardChatRoles: true,
			NoVisionModels:                []string{"deepseek-v4-flash"},
			VisionFallbackModel:           "MiniMax-M2.7",
		},
		ProviderMiniMax: {
			ModelMapping:                  map[string]string{"gpt-5": "MiniMax-M2.7"},
			CodexToolCompat:               boolRef(false),
			StripCodexClientTools:         boolRef(false),
			CodexNativeToolPassthrough:    true,
			NormalizeNonstandardChatRoles: true,
		},
		ProviderDashScope: {
			ModelMapping: map[string]string{
				"gpt-5.5":       "glm-5.1",
				"gpt-5.4":       "deepseek-v4-pro",
				"gpt-5.4-mini":  "deepseek-v4-flash",
				"gpt-5.3-codex": "deepseek-v4-flash",
			},
			ReasoningMapping: map[string]string{
				"gpt-5.5":       "high",
				"gpt-5.4":       "max",
				"gpt-5.4-mini":  "high",
				"gpt-5.3-codex": "high",
			},
		},
		ProviderOpenCodeZen: {
			ModelMapping: map[string]string{"gpt-5": "glm-5.1"},
		},
		ProviderOpenCodeGo: {
			ModelMapping: map[string]string{"gpt-5": "glm-5.1"},
		},
	},
}

func boolRef(value bool) *bool {
	return &value
}

func applyChannelTargetConfig(payload *ChannelPayload, config channelTargetConfig) {
	payload.ModelMapping = maps.Clone(config.ModelMapping)
	payload.ReasoningMapping = maps.Clone(config.ReasoningMapping)
	payload.NoVisionModels = slices.Clone(config.NoVisionModels)
	payload.ReasoningParamStyle = config.ReasoningParamStyle
	payload.PassbackReasoningContent = payload.PassbackReasoningContent || config.PassbackReasoningContent
	payload.PassbackThinkingBlocks = payload.PassbackThinkingBlocks || config.PassbackThinkingBlocks
	payload.NoVision = payload.NoVision || config.NoVision
	payload.VisionFallbackModel = config.VisionFallbackModel
	payload.StripEmptyTextBlocks = payload.StripEmptyTextBlocks || config.StripEmptyTextBlocks
	payload.CodexNativeToolPassthrough = payload.CodexNativeToolPassthrough || config.CodexNativeToolPassthrough
	payload.NormalizeNonstandardChatRoles = payload.NormalizeNonstandardChatRoles || config.NormalizeNonstandardChatRoles
	if config.CodexToolCompat != nil {
		payload.CodexToolCompat = *config.CodexToolCompat
	}
	if config.StripCodexClientTools != nil {
		payload.StripCodexClientTools = *config.StripCodexClientTools
	}
}

func applyTargetDefaults(payload *ChannelPayload, provider string, target string) {
	switch target {
	case TargetMessages:
		payload.ServiceType = "claude"
		payload.StripEmptyTextBlocks = true
		payload.StripThoughtSignature = true
	case TargetChat:
		payload.ServiceType = "openai"
		payload.NormalizeNonstandardChatRoles = true
	case TargetResponses:
		payload.ServiceType = "openai"
		payload.CodexToolCompat = true
		payload.StripCodexClientTools = true
	}

	configs, ok := channelTargetConfigs[target]
	if !ok {
		return
	}
	config, ok := configs[provider]
	if !ok {
		return
	}
	applyChannelTargetConfig(payload, config)
}
