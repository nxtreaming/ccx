package config

import "strings"

// ProviderTemplate 描述一个官方 provider 的模板化添加配置。
//
// 设计目标：用户只需选择 provider + 输入 API Key，系统按 key 前缀选候选 baseURL，
// 探测验证可用性后自动创建渠道，无需手填 baseURL / 选协议 / 配兼容开关。
//
// baseURL 按上游原生协议区分端点，调用时走该 baseURL 的原生协议路径，零协议转换。
// ProviderTemplate 只描述来源和端点，不再承载 channel-presets 兼容开关；
// autoManaged 渠道的模型与能力差异由后端智能调度/ModelResolver 处理。
// 首批 provider 均使用 Anthropic 兼容入口（ServiceType=claude / ChannelKind=messages）。
type ProviderTemplate struct {
	ProviderID     string              `json:"providerId"`     // "mimo" / "deepseek" / ...
	DisplayName    string              `json:"displayName"`    // "小米 MiMo"
	Description    string              `json:"description"`    // key 前缀说明等
	ChannelKind    string              `json:"channelKind"`    // "messages" / "chat"
	ServiceType    string              `json:"serviceType"`    // "claude" / "openai"
	OriginType     string              `json:"originType"`     // "official_api"
	OriginTier     string              `json:"originTier"`     // "first"
	KeyPrefixRules []KeyPrefixRule     `json:"keyPrefixRules"` // key 前缀 → plan 判别
	Candidates     []ProviderCandidate `json:"candidates"`     // 候选 baseURL（含 plan 标签）
}

// KeyPrefixRule 按 API Key 前缀判别 plan 类型。
type KeyPrefixRule struct {
	Prefix  string `json:"prefix"`  // "sk-" / "tp-"
	PlanTag string `json:"planTag"` // "payg" / "token_plan"
}

// ProviderCandidate 一个候选 baseURL 及其归属信息。
type ProviderCandidate struct {
	BaseURL  string `json:"baseUrl"`  // 完整 baseURL，如 https://api.xiaomimimo.com/anthropic
	PlanTag  string `json:"planTag"`  // "payg" / "token_plan" / ""
	Region   string `json:"region"`   // "cn" / "sgp" / "ams" / "global" / ""
	Priority int    `json:"priority"` // 同 plan 内探测优先级（数字越小越先探测）
}

// builtinProviderTemplates 编译期内置的官方 provider 模板。
//
// URL 来源（2026-07 联网核实官方文档）：
//   - MiMo:     https://mimo.mi.com/docs（按量 sk- / Token Plan tp-，TP 分 cn/sgp/ams 三区域集群）
//   - DeepSeek: https://api-docs.deepseek.com（Anthropic 兼容 /anthropic）
//   - Kimi:     https://api.moonshot.ai/anthropic（全球）/ https://api.moonshot.cn/anthropic（中国）
//   - GLM:      https://open.bigmodel.cn/api/anthropic
//
// baseURL 一律填 Anthropic 兼容入口且不带 /v1（claude provider 会自动补 /v1/messages）。
var builtinProviderTemplates = []ProviderTemplate{
	{
		ProviderID:  "mimo",
		DisplayName: "小米 MiMo",
		Description: "sk- 按量付费 / tp- Token Plan 订阅（自动判别区域集群）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		KeyPrefixRules: []KeyPrefixRule{
			{Prefix: "sk-", PlanTag: "payg"},
			{Prefix: "tp-", PlanTag: "token_plan"},
		},
		Candidates: []ProviderCandidate{
			{BaseURL: "https://api.xiaomimimo.com/anthropic", PlanTag: "payg", Region: "global", Priority: 0},
			{BaseURL: "https://token-plan-cn.xiaomimimo.com/anthropic", PlanTag: "token_plan", Region: "cn", Priority: 0},
			{BaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic", PlanTag: "token_plan", Region: "sgp", Priority: 1},
			{BaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic", PlanTag: "token_plan", Region: "ams", Priority: 2},
		},
	},
	{
		ProviderID:  "deepseek",
		DisplayName: "DeepSeek",
		Description: "DeepSeek 官方 API（Anthropic 兼容协议）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		Candidates: []ProviderCandidate{
			{BaseURL: "https://api.deepseek.com/anthropic", PlanTag: "", Region: "", Priority: 0},
		},
	},
	{
		ProviderID:  "kimi",
		DisplayName: "Kimi (Moonshot)",
		Description: "Moonshot Kimi 官方 API（Anthropic 兼容，自动判别全球/中国节点）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		Candidates: []ProviderCandidate{
			{BaseURL: "https://api.moonshot.cn/anthropic", PlanTag: "", Region: "cn", Priority: 0},
			{BaseURL: "https://api.moonshot.ai/anthropic", PlanTag: "", Region: "global", Priority: 1},
		},
	},
	{
		ProviderID:  "glm",
		DisplayName: "智谱 GLM",
		Description: "智谱 GLM 官方 API（Anthropic 兼容协议）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		Candidates: []ProviderCandidate{
			{BaseURL: "https://open.bigmodel.cn/api/anthropic", PlanTag: "", Region: "", Priority: 0},
		},
	},
}

// ListProviderTemplates 返回所有内置 provider 模板。
func ListProviderTemplates() []ProviderTemplate {
	out := make([]ProviderTemplate, len(builtinProviderTemplates))
	copy(out, builtinProviderTemplates)
	return out
}

// GetProviderTemplate 按 providerId 查找模板，未找到返回 (nil, false)。
func GetProviderTemplate(providerID string) (*ProviderTemplate, bool) {
	for i := range builtinProviderTemplates {
		if builtinProviderTemplates[i].ProviderID == providerID {
			return &builtinProviderTemplates[i], true
		}
	}
	return nil, false
}

// CandidatesForKey 按 API Key 前缀返回该 key 应优先探测的候选 baseURL 顺序。
//
// 规则（对应用户决策）：
//  1. key 前缀命中某 PlanTag → 该 plan 的候选优先（按 Priority 升序），其余候选追加在后作为回退
//  2. 前缀不匹配任何规则（或模板无前缀规则）→ 返回全部候选（按 Priority 升序）
//
// 返回的候选已按探测顺序排列，调用方依次探测，命中即用。
func (t *ProviderTemplate) CandidatesForKey(apiKey string) []ProviderCandidate {
	matchedPlan := ""
	for _, rule := range t.KeyPrefixRules {
		if rule.Prefix != "" && strings.HasPrefix(apiKey, rule.Prefix) {
			matchedPlan = rule.PlanTag
			break
		}
	}

	sortByPriority := func(list []ProviderCandidate) []ProviderCandidate {
		sorted := make([]ProviderCandidate, len(list))
		copy(sorted, list)
		// 稳定插入排序（候选数量极少，无需 sort 包）
		for i := 1; i < len(sorted); i++ {
			for j := i; j > 0 && sorted[j].Priority < sorted[j-1].Priority; j-- {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			}
		}
		return sorted
	}

	if matchedPlan == "" {
		return sortByPriority(t.Candidates)
	}

	preferred := make([]ProviderCandidate, 0, len(t.Candidates))
	fallback := make([]ProviderCandidate, 0, len(t.Candidates))
	for _, cand := range t.Candidates {
		if cand.PlanTag == matchedPlan {
			preferred = append(preferred, cand)
		} else {
			fallback = append(fallback, cand)
		}
	}
	return append(sortByPriority(preferred), sortByPriority(fallback)...)
}
