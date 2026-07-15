package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCandidatesForKeyPrefix(t *testing.T) {
	tmpl, ok := GetProviderTemplate("mimo")
	if !ok {
		t.Fatal("未找到 mimo 模板")
	}

	// sk- 前缀 → payg 优先
	skCands := tmpl.CandidatesForKey("sk-abc")
	if len(skCands) == 0 || skCands[0].PlanTag != "payg" {
		t.Errorf("sk- key 首选应为 payg，实际 %+v", skCands)
	}

	// tp- 前缀 → token_plan 优先，且按 Priority（cn=0 先）
	tpCands := tmpl.CandidatesForKey("tp-xyz")
	if len(tpCands) == 0 || tpCands[0].PlanTag != "token_plan" || tpCands[0].Region != "cn" {
		t.Errorf("tp- key 首选应为 token_plan/cn，实际 %+v", tpCands)
	}
	// tp- 回退候选应包含 payg（全部候选都在）
	if len(tpCands) != len(tmpl.Candidates) {
		t.Errorf("tp- key 应返回全部候选（含回退）%d，实际 %d", len(tmpl.Candidates), len(tpCands))
	}

	// 无匹配前缀 → 返回全部候选
	unknownCands := tmpl.CandidatesForKey("xx-none")
	if len(unknownCands) != len(tmpl.Candidates) {
		t.Errorf("未匹配前缀应返回全部候选 %d，实际 %d", len(tmpl.Candidates), len(unknownCands))
	}
}

func TestProviderTemplateMiMoRoutes(t *testing.T) {
	tmpl, ok := GetProviderTemplate("mimo")
	if !ok {
		t.Fatal("未找到 mimo 模板")
	}
	routes := tmpl.AutoAddRoutes()
	if len(routes) != 4 {
		t.Fatalf("mimo 应创建 4 条 route，实际 %d: %+v", len(routes), routes)
	}

	want := map[string]struct {
		serviceType string
		baseSuffix  string
	}{
		"messages":  {serviceType: "claude", baseSuffix: "/anthropic"},
		"chat":      {serviceType: "openai", baseSuffix: "/v1"},
		"responses": {serviceType: "openai", baseSuffix: "/v1"},
		"gemini":    {serviceType: "openai", baseSuffix: "/v1"},
	}
	for _, route := range routes {
		expect, ok := want[route.ChannelKind]
		if !ok {
			t.Fatalf("未知 route kind: %+v", route)
		}
		if route.ServiceType != expect.serviceType {
			t.Fatalf("route %s serviceType=%s, want %s", route.ChannelKind, route.ServiceType, expect.serviceType)
		}
		candidates := tmpl.CandidatesForRouteKey(route, "tp-test")
		if len(candidates) == 0 {
			t.Fatalf("route %s 没有候选", route.ChannelKind)
		}
		if !strings.HasSuffix(candidates[0].BaseURL, expect.baseSuffix) {
			t.Fatalf("route %s 首选 baseURL=%s, want suffix %s", route.ChannelKind, candidates[0].BaseURL, expect.baseSuffix)
		}
	}
	if !tmpl.SupportsChannelKind("messages") || !tmpl.SupportsChannelKind("chat") || !tmpl.SupportsChannelKind("responses") || !tmpl.SupportsChannelKind("gemini") {
		t.Fatalf("mimo 应支持 messages/chat/responses/gemini routes: %+v", routes)
	}
}

func TestCandidatesForKeyNoPrefixRules(t *testing.T) {
	// deepseek 无前缀规则，任意 key 返回全部候选
	tmpl, ok := GetProviderTemplate("deepseek")
	if !ok {
		t.Fatal("未找到 deepseek 模板")
	}
	cands := tmpl.CandidatesForKey("anything")
	if len(cands) != len(tmpl.Candidates) {
		t.Errorf("无前缀规则应返回全部候选 %d，实际 %d", len(tmpl.Candidates), len(cands))
	}
}

func TestProviderTemplateDeepSeekRoutes(t *testing.T) {
	tmpl, ok := GetProviderTemplate("deepseek")
	if !ok {
		t.Fatal("未找到 deepseek 模板")
	}
	routes := tmpl.AutoAddRoutes()
	if len(routes) != 3 {
		t.Fatalf("deepseek 应创建 messages/chat/responses 三条 route，实际 %d: %+v", len(routes), routes)
	}
	want := map[string]struct {
		serviceType string
		baseURL     string
	}{
		"messages":  {serviceType: "claude", baseURL: "https://api.deepseek.com/anthropic"},
		"chat":      {serviceType: "openai", baseURL: "https://api.deepseek.com"},
		"responses": {serviceType: "openai", baseURL: "https://api.deepseek.com"},
	}
	for _, route := range routes {
		expect, found := want[route.ChannelKind]
		if !found || route.ServiceType != expect.serviceType || len(route.Candidates) != 1 || route.Candidates[0].BaseURL != expect.baseURL {
			t.Fatalf("DeepSeek route 不符合预期: %+v", route)
		}
	}
}

func TestInferProviderIDFromBaseURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
		ok      bool
	}{
		{baseURL: "https://api.deepseek.com", want: "deepseek", ok: true},
		{baseURL: "https://api.deepseek.com/anthropic/v1", want: "deepseek", ok: true},
		{baseURL: "https://ark.cn-beijing.volces.com/api/plan/v3", want: "volcengine", ok: true},
		{baseURL: "https://relay.example/v1", ok: false},
		{baseURL: "https://api.deepseek.com.evil.example", ok: false},
	}
	for _, tt := range tests {
		got, ok := InferProviderIDFromBaseURL(tt.baseURL)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("InferProviderIDFromBaseURL(%q) = %q, %v; want %q, %v", tt.baseURL, got, ok, tt.want, tt.ok)
		}
	}
}

func TestProviderTemplateVolcenginePlanRoutes(t *testing.T) {
	tmpl, ok := GetProviderTemplate("volcengine")
	if !ok {
		t.Fatal("未找到火山方舟模板")
	}
	if tmpl.DisplayName != "火山方舟 Agent/Coding Plan" || len(tmpl.AutoAddRoutes()) != 4 {
		t.Fatalf("火山模板不完整: %+v", tmpl)
	}
	for _, route := range tmpl.AutoAddRoutes() {
		candidates := tmpl.CandidatesForRouteKey(route, "ark-test")
		if len(candidates) != 2 {
			t.Fatalf("route %s 应包含 Agent/Coding 两个候选: %+v", route.ChannelKind, candidates)
		}
		if !strings.Contains(candidates[0].BaseURL, "/api/plan") || !strings.Contains(candidates[1].BaseURL, "/api/coding") {
			t.Fatalf("route %s 候选顺序错误: %+v", route.ChannelKind, candidates)
		}
	}
}

func TestListAndGetProviderTemplate(t *testing.T) {
	all := ListProviderTemplates()
	if len(all) < 4 {
		t.Errorf("内置 provider 模板应 >= 4，实际 %d", len(all))
	}
	for _, id := range []string{"mimo", "deepseek", "kimi", "glm", "volcengine"} {
		if _, ok := GetProviderTemplate(id); !ok {
			t.Errorf("缺少 provider 模板: %s", id)
		}
	}
	if _, ok := GetProviderTemplate("nonexistent"); ok {
		t.Error("不存在的 providerId 应返回 false")
	}
}

func TestProviderTemplatesDoNotExposeChannelPresetRefs(t *testing.T) {
	data, err := json.Marshal(ListProviderTemplates())
	if err != nil {
		t.Fatalf("序列化 provider templates 失败: %v", err)
	}
	for _, field := range [][]byte{[]byte("presetRef"), []byte("presetCollection")} {
		if bytes.Contains(data, field) {
			t.Fatalf("provider template 不应暴露 channel preset 字段 %q: %s", field, data)
		}
	}
}
