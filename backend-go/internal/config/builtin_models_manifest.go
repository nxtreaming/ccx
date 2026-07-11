package config

import (
	"strings"

	"github.com/BenedictKing/ccx/internal/presetstore"
)

// BuiltinModelsManifest 内置模型清单条目。
// 对部分官方订阅入口（如 Claude OAuth 订阅、Codex plan 入口），
// 上游 models 接口可能不存在（404）或返回错误清单。
// 命中时直接使用 ModelIDs 作为该入口的可用模型列表，跳过上游探测。
type BuiltinModelsManifest struct {
	// BaseURLPattern 精确 host 或 host+path 前缀匹配，
	// 如 "api.anthropic.com" 或 "api.anthropic.com/v1"。
	BaseURLPattern string `json:"baseUrlPattern"`

	// ServiceType 运行时上游服务类型；claude 会归一为 messages，OpenAI Chat 兼容入口用 openai。
	ServiceType string `json:"serviceType"`

	// PlanHint 订阅类型提示，仅用于文档和日志，不影响匹配逻辑。
	PlanHint string `json:"planHint,omitempty"`

	// ModelIDs 该入口实际可用的模型 ID 清单。
	ModelIDs []string `json:"modelIds"`

	// ExcludeModelPatterns 从上游 /v1/models 返回中剔除的模型 ID 正则。
	ExcludeModelPatterns []string `json:"excludeModelPatterns,omitempty"`

	// DisableProbe 为 true 时 Discovery 流程跳过 GET /v1/models，
	// 直接用 ModelIDs 生成 ModelProfile[]。
	DisableProbe bool `json:"disableProbe"`
}

// builtinModelsManifests 内置模型清单。
// 当前仅收录从仓库代码/模板/测试中能确认的官方入口。
// 新增条目须附注证据来源，不臆造 baseURL。
var builtinModelsManifests = []BuiltinModelsManifest{
	// Anthropic 官方 API 入口（来源：buildClaudeCompatibleModelsURLs 测试、
	// expectedRequestUrls 测试、前端 locale placeholder 确认 api.anthropic.com）。
	// 该入口的 /v1/models 正常可用，DisableProbe=false；
	// 当上游探测失败时回退使用此清单。
	{
		BaseURLPattern: "api.anthropic.com",
		ServiceType:    "messages",
		PlanHint:       "anthropic_api",
		ModelIDs: []string{
			"claude-fable-5",
			"claude-mythos-5",
			"claude-opus-4-8",
			"claude-opus-4-7",
			"claude-opus-4-6",
			"claude-sonnet-5",
			"claude-sonnet-4-6",
			"claude-sonnet-4-5",
			"claude-opus-4-5",
			"claude-haiku-4-5",
		},
		DisableProbe: false,
	},
	// 小米 MiMo Anthropic 兼容入口（来源：provider_templates.go 与 docs/providers/mimo.md）。
	// Anthropic 协议入口可用性通过 /v1/messages 验证；/v1/models 不作为能力判定依据。
	{
		BaseURLPattern:       "api.xiaomimimo.com/anthropic",
		ServiceType:          "messages",
		PlanHint:             "mimo_payg_anthropic",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         true,
	},
	{
		BaseURLPattern:       "token-plan-cn.xiaomimimo.com/anthropic",
		ServiceType:          "messages",
		PlanHint:             "mimo_token_plan_cn_anthropic",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         true,
	},
	{
		BaseURLPattern:       "token-plan-sgp.xiaomimimo.com/anthropic",
		ServiceType:          "messages",
		PlanHint:             "mimo_token_plan_sgp_anthropic",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         true,
	},
	{
		BaseURLPattern:       "token-plan-ams.xiaomimimo.com/anthropic",
		ServiceType:          "messages",
		PlanHint:             "mimo_token_plan_ams_anthropic",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         true,
	},
	// 小米 MiMo OpenAI Chat 兼容入口（来源：provider_templates.go 与 docs/providers/mimo.md）。
	// /v1/models 可用于发现；返回 ASR/TTS 等非文本模型时通过 ExcludeModelPatterns 过滤。
	{
		BaseURLPattern:       "api.xiaomimimo.com/v1",
		ServiceType:          "openai",
		PlanHint:             "mimo_payg_openai",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         false,
	},
	{
		BaseURLPattern:       "token-plan-cn.xiaomimimo.com/v1",
		ServiceType:          "openai",
		PlanHint:             "mimo_token_plan_cn_openai",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         false,
	},
	{
		BaseURLPattern:       "token-plan-sgp.xiaomimimo.com/v1",
		ServiceType:          "openai",
		PlanHint:             "mimo_token_plan_sgp_openai",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         false,
	},
	{
		BaseURLPattern:       "token-plan-ams.xiaomimimo.com/v1",
		ServiceType:          "openai",
		PlanHint:             "mimo_token_plan_ams_openai",
		ModelIDs:             mimoModelIDs(),
		ExcludeModelPatterns: mimoExcludeModelPatterns(),
		DisableProbe:         false,
	},
}

func mimoModelIDs() []string {
	return []string{
		"mimo-v2.5-pro",
		"mimo-v2.5",
	}
}

func mimoExcludeModelPatterns() []string {
	return []string{
		`^mimo-v2\.5-(?:asr|tts(?:-.+)?)$`,
	}
}

// LookupBuiltinManifest 根据上游 baseURL 和 serviceType 查找匹配的内置清单。
// 匹配规则：BaseURLPattern 作为 host 或 host+path 前缀匹配。
// 优先精确 host 匹配，其次 host+path 前缀匹配（最长前缀优先）。
// 返回匹配的清单和 true；未命中返回零值和 false。
func LookupBuiltinManifest(baseURL string, serviceType string) (BuiltinModelsManifest, bool) {
	normalized := normalizeBaseURLForManifest(baseURL)
	if normalized == "" {
		return BuiltinModelsManifest{}, false
	}

	if manifest, found := lookupBuiltinManifestIn(runtimeBuiltinModelsManifests(), normalized, serviceType); found {
		return manifest, true
	}
	return lookupBuiltinManifestIn(builtinModelsManifests, normalized, serviceType)
}

func runtimeBuiltinModelsManifests() []BuiltinModelsManifest {
	bundle := presetstore.Default().Get()
	if bundle == nil || bundle.BuiltinModelsManifests == nil || len(bundle.BuiltinModelsManifests.Manifests) == 0 {
		return nil
	}
	manifests := make([]BuiltinModelsManifest, 0, len(bundle.BuiltinModelsManifests.Manifests))
	for _, entry := range bundle.BuiltinModelsManifests.Manifests {
		manifests = append(manifests, BuiltinModelsManifest{
			BaseURLPattern:       entry.BaseURLPattern,
			ServiceType:          entry.ServiceType,
			PlanHint:             entry.PlanHint,
			ModelIDs:             append([]string(nil), entry.ModelIDs...),
			ExcludeModelPatterns: append([]string(nil), entry.ExcludeModelPatterns...),
			DisableProbe:         entry.DisableProbe,
		})
	}
	return manifests
}

func lookupBuiltinManifestIn(manifests []BuiltinModelsManifest, normalized string, serviceType string) (BuiltinModelsManifest, bool) {
	if len(manifests) == 0 {
		return BuiltinModelsManifest{}, false
	}
	var bestMatch BuiltinModelsManifest
	var bestMatchLen int
	found := false
	for _, manifest := range manifests {
		if manifest.ServiceType != serviceType {
			continue
		}
		pattern := strings.ToLower(manifest.BaseURLPattern)
		if !matchManifestPattern(normalized, pattern) {
			continue
		}
		if len(pattern) > bestMatchLen {
			bestMatch = manifest
			bestMatchLen = len(pattern)
			found = true
		}
	}
	return bestMatch, found
}

// matchManifestPattern 检查 normalized baseURL 是否匹配 manifest pattern。
// pattern 可以是纯 host（精确匹配）或 host+path 前缀匹配。
func matchManifestPattern(normalized string, pattern string) bool {
	// 精确匹配
	if normalized == pattern {
		return true
	}
	// host 精确匹配（pattern 不含路径，normalized 可能含路径）
	if !strings.Contains(pattern, "/") && strings.HasPrefix(normalized, pattern) {
		rest := normalized[len(pattern):]
		return rest == "" || rest[0] == '/'
	}
	// host+path 前缀匹配
	return strings.HasPrefix(normalized, pattern)
}

// normalizeBaseURLForManifest 将 baseURL 规范化为 host 或 host+path 形式用于匹配。
// 去掉 scheme（https://）、尾部 /、# 标记、版本段（/v1 等）。
func normalizeBaseURLForManifest(baseURL string) string {
	s := strings.ToLower(strings.TrimSpace(baseURL))
	if s == "" {
		return ""
	}
	// 去掉 scheme
	if idx := strings.Index(s, "://"); idx >= 0 {
		s = s[idx+3:]
	}
	// 去掉 #
	s = strings.TrimSuffix(s, "#")
	// 去掉尾部 /
	s = strings.TrimRight(s, "/")
	return s
}
