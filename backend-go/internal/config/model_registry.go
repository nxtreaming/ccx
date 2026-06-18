package config

import (
	"sort"
	"strings"
)

const (
	DefaultOutputReserveTokens     = 8192
	DefaultUnknownSafeWindowTokens = 200000
)

// ResolvedAgentModelProfile 描述下游 agent 模型解析结果。
type ResolvedAgentModelProfile struct {
	Profile        AgentModelProfile
	MatchedPattern string
	Source         string
	Known          bool
}

// ResolvedUpstreamCapability 描述实际模型能力解析结果。
type ResolvedUpstreamCapability struct {
	Capability     UpstreamModelCapability
	RequestModel   string
	ActualModel    string
	MatchedPattern string
	Source         string
	Known          bool
}

// IsContextRoutingEnabled 返回上下文路由是否启用，默认启用。
func (c ContextRoutingConfig) IsContextRoutingEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// EffectiveOutputReserveTokens 返回未显式请求输出上限时的预留 token。
func (c ContextRoutingConfig) EffectiveOutputReserveTokens() int {
	if c.DefaultOutputReserveTokens > 0 {
		return c.DefaultOutputReserveTokens
	}
	return DefaultOutputReserveTokens
}

// EffectiveUnknownSafeWindowTokens 返回未知能力渠道可接受的安全窗口。
func (c ContextRoutingConfig) EffectiveUnknownSafeWindowTokens() int {
	if c.UnknownSafeWindowTokens > 0 {
		return c.UnknownSafeWindowTokens
	}
	return DefaultUnknownSafeWindowTokens
}

// BuiltinAgentModelProfiles 返回 CCX 内置的下游 agent 模型知识库。
func BuiltinAgentModelProfiles() map[string]AgentModelProfile {
	return map[string]AgentModelProfile{
		"gpt-5.2": {
			DisplayName:            "GPT-5.5 / gpt-5.2",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 272000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "bytes",
			TruncationLimit:        10000,
			ReasoningEfforts:       []string{"low", "medium", "high", "xhigh"},
		},
		"gpt-5.4": {
			DisplayName:            "gpt-5.4",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 1000000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "tokens",
			TruncationLimit:        10000,
			ReasoningEfforts:       []string{"low", "medium", "high", "xhigh"},
			SupportsPriorityTier:   true,
		},
		"gpt-5.4-mini": {
			DisplayName:            "gpt-5.4-mini",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 272000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "tokens",
			TruncationLimit:        10000,
			ReasoningEfforts:       []string{"low", "medium", "high", "xhigh"},
		},
		"gpt-5.3-codex": {
			DisplayName:            "gpt-5.3-codex",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 272000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "tokens",
			TruncationLimit:        10000,
			ReasoningEfforts:       []string{"low", "medium", "high", "xhigh"},
		},
		"codex-auto-review": {
			DisplayName:            "Codex Auto Review",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 1000000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "tokens",
			TruncationLimit:        10000,
			ReasoningEfforts:       []string{"low", "medium", "high", "xhigh"},
		},
		"gpt-5.5": {
			DisplayName:            "GPT-5.5",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 272000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "tokens",
			TruncationLimit:        10000,
			ReasoningEfforts:       []string{"low", "medium", "high", "xhigh"},
			SupportsPriorityTier:   true,
		},
		"claude-haiku-4-5*": {
			DisplayName:         "Claude Haiku 4.5",
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
			ReasoningEfforts:    []string{"extended"},
		},
		"claude-sonnet-4-5*": {
			DisplayName:         "Claude Sonnet 4.5",
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
			ReasoningEfforts:    []string{"extended"},
		},
		"claude-opus-4-5*": {
			DisplayName:         "Claude Opus 4.5",
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
			ReasoningEfforts:    []string{"low", "medium", "high"},
		},
		"claude-sonnet-4-6*": {
			DisplayName:         "Claude Sonnet 4.6",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     64000,
			ReasoningEfforts:    []string{"low", "medium", "high", "max"},
		},
		"claude-opus-4-6*": {
			DisplayName:         "Claude Opus 4.6",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "max"},
		},
		"claude-opus-4-7*": {
			DisplayName:         "Claude Opus 4.7",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-opus-4-8*": {
			DisplayName:         "Claude Opus 4.8",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-fable-5*": {
			DisplayName:         "Claude Fable 5",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-mythos-5*": {
			DisplayName:         "Claude Mythos 5",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-mythos-preview*": {
			DisplayName:         "Claude Mythos Preview",
			ContextWindowTokens: 1000000,
			ReasoningEfforts:    []string{"max"},
		},
		"fable": {
			DisplayName:         "Claude Fable alias",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
		},
		"mythos": {
			DisplayName:         "Claude Mythos alias",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
		},
		"opus": {
			DisplayName:         "Claude Opus alias",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
		},
		"sonnet": {
			DisplayName:         "Claude Sonnet alias",
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     64000,
		},
		"haiku": {
			DisplayName:         "Claude Haiku alias",
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
		},
		"*": {
			DisplayName:            "Codex fallback",
			ContextWindowTokens:    272000,
			MaxContextWindowTokens: 272000,
			EffectiveContextRatio:  0.95,
			AutoCompactRatio:       0.90,
			TruncationMode:         "bytes",
			TruncationLimit:        10000,
		},
	}
}

// BuiltinUpstreamModelCapabilities 返回 CCX 内置的实际上游模型能力知识库。
func BuiltinUpstreamModelCapabilities() map[string]UpstreamModelCapability {
	return map[string]UpstreamModelCapability{
		"claude-haiku-4-5*": {
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
			ThinkingMode:        "extended",
		},
		"claude-sonnet-4-5*": {
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
			ThinkingMode:        "extended",
		},
		"claude-opus-4-5*": {
			ContextWindowTokens: 200000,
			MaxOutputTokens:     64000,
			ThinkingMode:        "extended",
			ReasoningEfforts:    []string{"low", "medium", "high"},
		},
		"claude-sonnet-4-6*": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     64000,
			ThinkingMode:        "adaptive",
			ReasoningEfforts:    []string{"low", "medium", "high", "max"},
		},
		"claude-opus-4-6*": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ThinkingMode:        "adaptive",
			ReasoningEfforts:    []string{"low", "medium", "high", "max"},
		},
		"claude-opus-4-7*": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ThinkingMode:        "adaptive_only",
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-opus-4-8*": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ThinkingMode:        "adaptive_only",
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-fable-5*": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ThinkingMode:        "adaptive_always_on",
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-mythos-5*": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ThinkingMode:        "adaptive_always_on",
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh", "max"},
		},
		"claude-mythos-preview*": {
			ContextWindowTokens: 1000000,
			ThinkingMode:        "adaptive",
			ReasoningEfforts:    []string{"max"},
		},
		"gpt-5.2": {
			ContextWindowTokens: 272000,
		},
		"gpt-5.5": {
			ContextWindowTokens: 272000,
		},
		"gpt-5.3-codex": {
			ContextWindowTokens: 272000,
		},
		"gpt-5.4": {
			ContextWindowTokens: 1000000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh"},
		},
		"gpt-5.4-mini": {
			ContextWindowTokens: 272000,
			MaxOutputTokens:     128000,
			ReasoningEfforts:    []string{"low", "medium", "high", "xhigh"},
		},
	}
}

// ResolveAgentModelProfile 解析下游 agent 模型语义。
func ResolveAgentModelProfile(requestModel string, global map[string]AgentModelProfile) ResolvedAgentModelProfile {
	if profile, pattern, ok := resolvePatternValue(requestModel, global); ok {
		return ResolvedAgentModelProfile{Profile: profile, MatchedPattern: pattern, Source: "global", Known: true}
	}
	if profile, pattern, ok := resolvePatternValue(requestModel, BuiltinAgentModelProfiles()); ok {
		return ResolvedAgentModelProfile{Profile: profile, MatchedPattern: pattern, Source: "builtin", Known: true}
	}
	return ResolvedAgentModelProfile{}
}

// ResolveUpstreamCapability 解析渠道中实际模型的能力。
func ResolveUpstreamCapability(requestModel string, upstream *UpstreamConfig, global map[string]UpstreamModelCapability) ResolvedUpstreamCapability {
	actualModel := requestModel
	if upstream != nil {
		actualModel = RedirectModel(requestModel, upstream)
		if capability, pattern, ok := resolveCapabilityForModels(actualModel, requestModel, upstream.ModelCapabilities); ok {
			return ResolvedUpstreamCapability{Capability: capability, RequestModel: requestModel, ActualModel: actualModel, MatchedPattern: pattern, Source: "channel", Known: true}
		}
	}
	if capability, pattern, ok := resolveCapabilityForModels(actualModel, requestModel, global); ok {
		return ResolvedUpstreamCapability{Capability: capability, RequestModel: requestModel, ActualModel: actualModel, MatchedPattern: pattern, Source: "global", Known: true}
	}
	if capability, pattern, ok := resolveCapabilityForModels(actualModel, requestModel, BuiltinUpstreamModelCapabilities()); ok {
		return ResolvedUpstreamCapability{Capability: capability, RequestModel: requestModel, ActualModel: actualModel, MatchedPattern: pattern, Source: "builtin", Known: true}
	}
	if upstream != nil && (upstream.DefaultCapability.ContextWindowTokens > 0 || upstream.DefaultCapability.MaxOutputTokens > 0) {
		return ResolvedUpstreamCapability{Capability: upstream.DefaultCapability, RequestModel: requestModel, ActualModel: actualModel, Source: "channel_default", Known: true}
	}
	return ResolvedUpstreamCapability{RequestModel: requestModel, ActualModel: actualModel}
}

func resolveCapabilityForModels(actualModel, requestModel string, capabilities map[string]UpstreamModelCapability) (UpstreamModelCapability, string, bool) {
	if capability, pattern, ok := resolvePatternValue(actualModel, capabilities); ok {
		return capability, pattern, true
	}
	if requestModel != actualModel {
		if capability, pattern, ok := resolvePatternValue(requestModel, capabilities); ok {
			return capability, pattern, true
		}
	}
	return UpstreamModelCapability{}, "", false
}

func resolvePatternValue[T any](model string, values map[string]T) (T, string, bool) {
	var zero T
	model = strings.TrimSpace(model)
	if model == "" || len(values) == 0 {
		return zero, "", false
	}
	if value, ok := values[model]; ok {
		return value, model, true
	}

	patterns := make([]string, 0, len(values))
	for pattern := range values {
		if pattern == model {
			continue
		}
		if isValidSupportedModelPattern(pattern) {
			patterns = append(patterns, pattern)
		}
	}
	sort.Slice(patterns, func(i, j int) bool {
		if len(patterns[i]) == len(patterns[j]) {
			return patterns[i] < patterns[j]
		}
		return len(patterns[i]) > len(patterns[j])
	})

	for _, pattern := range patterns {
		if matchSupportedModelPattern(pattern, model) {
			return values[pattern], pattern, true
		}
	}
	return zero, "", false
}
