package config

import (
	"net/url"
	"strings"
)

// ProviderTemplate 描述一个官方 provider 的模板化添加配置。
//
// 设计目标：用户只需选择 provider + 输入 API Key，系统按 key 前缀选候选 baseURL，
// 探测验证可用性后自动创建渠道，无需手填 baseURL / 选协议 / 配兼容开关。
//
// baseURL 按上游原生协议区分端点：Claude 请求优先走原生 Claude 入口；
// Chat / Codex Responses 请求走 OpenAI Chat 兼容入口，由后端做协议转换。
// ProviderTemplate 只描述来源和端点，不再承载 channel-presets 兼容开关；
// autoManaged 渠道的模型与能力差异由后端智能调度/ModelResolver 处理。
type ProviderTemplate struct {
	ProviderID     string              `json:"providerId"`     // "mimo" / "deepseek" / ...
	DisplayName    string              `json:"displayName"`    // "小米 MiMo"
	Description    string              `json:"description"`    // key 前缀说明等
	ChannelKind    string              `json:"channelKind"`    // 默认 route 的渠道类型，兼容旧前端
	ServiceType    string              `json:"serviceType"`    // 默认 route 的服务类型，兼容旧前端
	OriginType     string              `json:"originType"`     // "official_api"
	OriginTier     string              `json:"originTier"`     // "first"
	KeyPrefixRules []KeyPrefixRule     `json:"keyPrefixRules"` // key 前缀 → plan 判别
	Candidates     []ProviderCandidate `json:"candidates"`     // 默认 route 的候选 baseURL
	Routes         []ProviderRoute     `json:"routes,omitempty"`
}

// ProviderRoute 描述同一 provider 在某个 CCX 渠道协议下使用的原生上游入口。
type ProviderRoute struct {
	ChannelKind string              `json:"channelKind"`           // "messages" / "chat" / "responses"
	ServiceType string              `json:"serviceType"`           // "claude" / "openai" / "responses"
	Description string              `json:"description,omitempty"` // route 说明，仅展示/诊断
	Candidates  []ProviderCandidate `json:"candidates"`            // 该 route 的候选 baseURL
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
//   - 火山方舟: https://ark.cn-beijing.volces.com/api/plan（Agent Plan）与 /api/coding（Coding Plan）
//
// Claude route 的 baseURL 使用 Anthropic 兼容入口且不带 /v1（claude provider 会自动补 /v1/messages）。
// Chat/Responses/Gemini route 使用 OpenAI Chat 兼容入口 /v1（provider 自动拼 /chat/completions）。
var builtinProviderTemplates = []ProviderTemplate{
	{
		ProviderID:  "mimo",
		DisplayName: "小米 MiMo",
		Description: "sk- 按量付费 / tp- Token Plan 订阅（Claude 走 /anthropic，Chat/Codex/Gemini 走 /v1）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		KeyPrefixRules: []KeyPrefixRule{
			{Prefix: "sk-", PlanTag: "payg"},
			{Prefix: "tp-", PlanTag: "token_plan"},
		},
		Candidates: mimoClaudeCandidates(),
		Routes: []ProviderRoute{
			{
				ChannelKind: "messages",
				ServiceType: "claude",
				Description: "Claude Messages 原生 Anthropic 兼容入口",
				Candidates:  mimoClaudeCandidates(),
			},
			{
				ChannelKind: "chat",
				ServiceType: "openai",
				Description: "OpenAI Chat Completions 兼容入口",
				Candidates:  mimoChatCandidates(),
			},
			{
				ChannelKind: "responses",
				ServiceType: "openai",
				Description: "Codex/Responses 请求转换到 OpenAI Chat Completions",
				Candidates:  mimoChatCandidates(),
			},
			{
				ChannelKind: "gemini",
				ServiceType: "openai",
				Description: "Gemini 请求转换到 OpenAI Chat Completions",
				Candidates:  mimoChatCandidates(),
			},
		},
	},
	{
		ProviderID:  "deepseek",
		DisplayName: "DeepSeek",
		Description: "DeepSeek 官方 API（Claude Messages 与 OpenAI Chat 兼容；Responses 由 CCX 转换）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		Candidates:  deepseekClaudeCandidates(),
		Routes: []ProviderRoute{
			{
				ChannelKind: "messages",
				ServiceType: "claude",
				Description: "Claude Messages Anthropic 兼容入口",
				Candidates:  deepseekClaudeCandidates(),
			},
			{
				ChannelKind: "chat",
				ServiceType: "openai",
				Description: "OpenAI Chat Completions 兼容入口",
				Candidates:  deepseekOpenAICandidates(),
			},
			{
				ChannelKind: "responses",
				ServiceType: "openai",
				Description: "Responses 请求转换到 OpenAI Chat Completions",
				Candidates:  deepseekOpenAICandidates(),
			},
		},
	},
	{
		ProviderID:  "volcengine",
		DisplayName: "火山方舟 Agent/Coding Plan",
		Description: "ark- 套餐推理 Key（系统自动识别套餐；模型发现需为每个 Key 绑定火山云 Access Key）",
		ChannelKind: "messages",
		ServiceType: "claude",
		OriginType:  "official_api",
		OriginTier:  "first",
		KeyPrefixRules: []KeyPrefixRule{
			{Prefix: "ark-", PlanTag: "personal_plan"},
		},
		Candidates: volcenginePlanClaudeCandidates(),
		Routes: []ProviderRoute{
			{
				ChannelKind: "messages",
				ServiceType: "claude",
				Description: "Agent/Coding Plan Claude Messages 入口",
				Candidates:  volcenginePlanClaudeCandidates(),
			},
			{
				ChannelKind: "chat",
				ServiceType: "openai",
				Description: "Agent/Coding Plan OpenAI Chat Completions 入口",
				Candidates:  volcenginePlanChatCandidates(),
			},
			{
				ChannelKind: "responses",
				ServiceType: "openai",
				Description: "Codex/Responses 请求转换到套餐 Chat Completions",
				Candidates:  volcenginePlanChatCandidates(),
			},
			{
				ChannelKind: "gemini",
				ServiceType: "openai",
				Description: "Gemini 请求转换到套餐 Chat Completions",
				Candidates:  volcenginePlanChatCandidates(),
			},
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

func deepseekClaudeCandidates() []ProviderCandidate {
	return []ProviderCandidate{
		{BaseURL: "https://api.deepseek.com/anthropic", PlanTag: "", Region: "", Priority: 0},
	}
}

func deepseekOpenAICandidates() []ProviderCandidate {
	return []ProviderCandidate{
		{BaseURL: "https://api.deepseek.com", PlanTag: "", Region: "", Priority: 0},
	}
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

// InferProviderIDFromBaseURL 仅按官方模板候选端点识别 provider。
// 使用最长路径匹配，避免把仅承载同名模型的第三方 relay 误判为官方渠道。
func InferProviderIDFromBaseURL(baseURL string) (string, bool) {
	target, err := url.Parse(strings.TrimSuffix(strings.TrimSpace(baseURL), "#"))
	if err != nil || target.Hostname() == "" {
		return "", false
	}
	targetPath := strings.TrimRight(target.EscapedPath(), "/")
	bestProvider := ""
	bestPathLen := -1
	for _, tmpl := range builtinProviderTemplates {
		for _, route := range tmpl.AutoAddRoutes() {
			for _, candidate := range route.Candidates {
				candidateURL, parseErr := url.Parse(strings.TrimSpace(candidate.BaseURL))
				if parseErr != nil || !sameURLServer(target, candidateURL) {
					continue
				}
				candidatePath := strings.TrimRight(candidateURL.EscapedPath(), "/")
				if candidatePath != "" && targetPath != candidatePath && !strings.HasPrefix(targetPath, candidatePath+"/") {
					continue
				}
				if len(candidatePath) > bestPathLen {
					bestProvider = tmpl.ProviderID
					bestPathLen = len(candidatePath)
				}
			}
		}
	}
	return bestProvider, bestProvider != ""
}

func sameURLServer(left, right *url.URL) bool {
	if left == nil || right == nil || !strings.EqualFold(left.Hostname(), right.Hostname()) {
		return false
	}
	return effectiveURLPort(left) == effectiveURLPort(right)
}

func effectiveURLPort(value *url.URL) string {
	if value == nil {
		return ""
	}
	if port := value.Port(); port != "" {
		return port
	}
	switch strings.ToLower(value.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// AutoAddRoutes 返回 provider 快速添加时需要创建的渠道 route。
func (t *ProviderTemplate) AutoAddRoutes() []ProviderRoute {
	if t == nil {
		return nil
	}
	if len(t.Routes) > 0 {
		out := make([]ProviderRoute, len(t.Routes))
		copy(out, t.Routes)
		return out
	}
	if t.ChannelKind == "" || t.ServiceType == "" {
		return nil
	}
	return []ProviderRoute{{
		ChannelKind: t.ChannelKind,
		ServiceType: t.ServiceType,
		Candidates:  append([]ProviderCandidate(nil), t.Candidates...),
	}}
}

// SupportsChannelKind 判断 provider 是否可从指定渠道页发起快速添加。
func (t *ProviderTemplate) SupportsChannelKind(kind string) bool {
	for _, route := range t.AutoAddRoutes() {
		if route.ChannelKind == kind {
			return true
		}
	}
	return false
}

// CandidatesForKey 按 API Key 前缀返回默认 route 的候选 baseURL 顺序。
//
// 规则（对应用户决策）：
//  1. key 前缀命中某 PlanTag → 该 plan 的候选优先（按 Priority 升序），其余候选追加在后作为回退
//  2. 前缀不匹配任何规则（或模板无前缀规则）→ 返回全部候选（按 Priority 升序）
//
// 返回的候选已按探测顺序排列，调用方依次探测，命中即用。
func (t *ProviderTemplate) CandidatesForKey(apiKey string) []ProviderCandidate {
	if t == nil {
		return nil
	}
	return t.CandidatesForRouteKey(ProviderRoute{Candidates: t.Candidates}, apiKey)
}

// CandidatesForRouteKey 按 API Key 前缀返回指定 route 的候选 baseURL 顺序。
func (t *ProviderTemplate) CandidatesForRouteKey(route ProviderRoute, apiKey string) []ProviderCandidate {
	if t == nil {
		return nil
	}
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
		return sortByPriority(route.Candidates)
	}

	preferred := make([]ProviderCandidate, 0, len(route.Candidates))
	fallback := make([]ProviderCandidate, 0, len(route.Candidates))
	for _, cand := range route.Candidates {
		if cand.PlanTag == matchedPlan {
			preferred = append(preferred, cand)
		} else {
			fallback = append(fallback, cand)
		}
	}
	return append(sortByPriority(preferred), sortByPriority(fallback)...)
}

func mimoClaudeCandidates() []ProviderCandidate {
	return []ProviderCandidate{
		{BaseURL: "https://api.xiaomimimo.com/anthropic", PlanTag: "payg", Region: "global", Priority: 0},
		{BaseURL: "https://token-plan-cn.xiaomimimo.com/anthropic", PlanTag: "token_plan", Region: "cn", Priority: 0},
		{BaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic", PlanTag: "token_plan", Region: "sgp", Priority: 1},
		{BaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic", PlanTag: "token_plan", Region: "ams", Priority: 2},
	}
}

func mimoChatCandidates() []ProviderCandidate {
	return []ProviderCandidate{
		{BaseURL: "https://api.xiaomimimo.com/v1", PlanTag: "payg", Region: "global", Priority: 0},
		{BaseURL: "https://token-plan-cn.xiaomimimo.com/v1", PlanTag: "token_plan", Region: "cn", Priority: 0},
		{BaseURL: "https://token-plan-sgp.xiaomimimo.com/v1", PlanTag: "token_plan", Region: "sgp", Priority: 1},
		{BaseURL: "https://token-plan-ams.xiaomimimo.com/v1", PlanTag: "token_plan", Region: "ams", Priority: 2},
	}
}

func volcenginePlanClaudeCandidates() []ProviderCandidate {
	return []ProviderCandidate{
		{BaseURL: "https://ark.cn-beijing.volces.com/api/plan", PlanTag: "personal_plan", Region: "cn", Priority: 0},
		{BaseURL: "https://ark.cn-beijing.volces.com/api/coding", PlanTag: "personal_plan", Region: "cn", Priority: 1},
	}
}

func volcenginePlanChatCandidates() []ProviderCandidate {
	return []ProviderCandidate{
		{BaseURL: "https://ark.cn-beijing.volces.com/api/plan/v3", PlanTag: "personal_plan", Region: "cn", Priority: 0},
		{BaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3", PlanTag: "personal_plan", Region: "cn", Priority: 1},
	}
}
