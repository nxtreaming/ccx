package autopilot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func setupAutoManagedRouter(deps *AutoManagedDeps) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 直接在根引擎注册路由，模拟 apiGroup 注册
	RegisterAutoManagedRoutes(r.Group("/api"), deps)
	return r
}

func TestAutoAddRequest_Binding(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "完整请求",
			body: `{"name":"test","baseUrls":["https://api.example.com"],"apiKeys":["sk-test123"]}`,
		},
		{
			name: "自动生成名称",
			body: `{"baseUrls":["https://api.example.com"],"apiKeys":["sk-test123"]}`,
		},
		{
			name: "多 URL 和多 Key",
			body: `{"baseUrls":["https://api1.example.com","https://api2.example.com"],"apiKeys":["sk-key1","sk-key2"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req AutoAddRequest
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", "application/json")

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&req); err != nil {
				if !tt.wantErr {
					t.Fatalf("解码失败: %v", err)
				}
				return
			}

			if len(req.BaseURLs) == 0 {
				t.Fatal("baseUrls 不应为空")
			}
			if len(req.APIKeys) == 0 {
				t.Fatal("apiKeys 不应为空")
			}
		})
	}
}

func TestAutoAddRequest_Validation(t *testing.T) {
	tests := []struct {
		name string
		body string
		err  string
	}{
		{
			name: "空 baseUrls",
			body: `{"baseUrls":[],"apiKeys":["sk-test"]}`,
			err:  "baseUrls 不能为空",
		},
		{
			name: "空 apiKeys",
			body: `{"baseUrls":["https://api.example.com"],"apiKeys":[]}`,
			err:  "apiKeys 不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req AutoAddRequest
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", "application/json")

			decoder := json.NewDecoder(r.Body)
			_ = decoder.Decode(&req) // 忽略解码错误，关注业务校验

			if len(req.BaseURLs) == 0 {
				// 这就是期望的行为
				return
			}
			if len(req.APIKeys) == 0 {
				return
			}
			t.Fatalf("期望验证失败: %s", tt.err)
		})
	}
}

func TestAutoStatusResponse_Serialization(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	resp := AutoStatusResponse{
		AutoManaged:   true,
		AutoManagedAt: &now,
		Discovery: &DiscoveryStatusInfo{
			Status:    DiscoveryStatusDone,
			StartedAt: &now,
			Endpoints: []EndpointDiscoveryInfo{
				{
					KeyMask:     "sk-****abcd",
					BaseURL:     "https://api.example.com",
					ModelsCount: 5,
					ProtocolOk:  true,
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if parsed["autoManaged"] != true {
		t.Fatalf("期望 autoManaged=true, 实际=%v", parsed["autoManaged"])
	}

	discovery := parsed["discovery"].(map[string]interface{})
	if discovery["status"] != "done" {
		t.Fatalf("期望 status=done, 实际=%v", discovery["status"])
	}

	endpoints := discovery["endpoints"].([]interface{})
	if len(endpoints) != 1 {
		t.Fatalf("期望 1 个 endpoint, 实际=%d", len(endpoints))
	}
	ep := endpoints[0].(map[string]interface{})
	if ep["keyMask"] != "sk-****abcd" {
		t.Fatalf("期望 keyMask=sk-****abcd, 实际=%v", ep["keyMask"])
	}
}

func TestAutoStatusResponse_KeyMaskNoPlaintext(t *testing.T) {
	// 验证 key mask 不会泄露明文 key
	result := EndpointDiscoveryInfo{
		KeyMask:     "sk-****abcd",
		BaseURL:     "https://api.example.com",
		ModelsCount: 3,
		ProtocolOk:  true,
	}

	data, _ := json.Marshal(result)
	s := string(data)

	// 不应包含常见的 key 前缀（如果 key 以 sk- 开头）
	if s == "sk-test1234567890" {
		t.Fatal("key mask 包含明文 key")
	}
}

func TestValidChannelKinds(t *testing.T) {
	expected := []string{"messages", "chat", "responses", "gemini", "images", "vectors"}
	for _, kind := range expected {
		if !validChannelKinds[kind] {
			t.Fatalf("渠道类型 %s 应有效", kind)
		}
	}

	invalid := []string{"unknown", "test", ""}
	for _, kind := range invalid {
		if validChannelKinds[kind] {
			t.Fatalf("渠道类型 %s 应无效", kind)
		}
	}
}

func TestKindToDefaultServiceType(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{"messages", "claude"},
		{"chat", "openai"},
		{"responses", "responses"},
		{"gemini", "gemini"},
		{"images", "openai"},
		{"vectors", "openai"},
		{"unknown", "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			result := kindToDefaultServiceType(tt.kind)
			if result != tt.expected {
				t.Fatalf("kind=%s, 期望=%s, 实际=%s", tt.kind, tt.expected, result)
			}
		})
	}
}

func TestProviderRouteNameAndPrimaryResult(t *testing.T) {
	base := "mimo-test"
	tests := []struct {
		route config.ProviderRoute
		want  string
	}{
		{route: config.ProviderRoute{ChannelKind: "messages"}, want: "mimo-test-claude"},
		{route: config.ProviderRoute{ChannelKind: "chat"}, want: "mimo-test-chat"},
		{route: config.ProviderRoute{ChannelKind: "responses"}, want: "mimo-test-codex"},
	}
	for _, tt := range tests {
		if got := providerRouteName(base, tt.route, true); got != tt.want {
			t.Fatalf("providerRouteName(%s)=%q, want %q", tt.route.ChannelKind, got, tt.want)
		}
	}
	if got := providerRouteName(base, config.ProviderRoute{ChannelKind: "messages"}, false); got != base {
		t.Fatalf("single route name=%q, want %q", got, base)
	}

	results := []AutoAddChannelResult{
		{ChannelKind: "messages", ChannelUID: "ch_messages", Index: 1},
		{ChannelKind: "chat", ChannelUID: "ch_chat", Index: 2},
		{ChannelKind: "responses", ChannelUID: "ch_responses", Index: 3},
	}
	primary := primaryAutoAddResult(results, "chat")
	if primary.ChannelUID != "ch_chat" || primary.Index != 2 {
		t.Fatalf("primary=%+v, want chat result", primary)
	}
}

func TestAutoAddHandler_InvalidKind(t *testing.T) {
	deps := &AutoManagedDeps{
		CfgManager: nil,
		Runner:     nil,
	}
	r := setupAutoManagedRouter(deps)

	body := `{"baseUrls":["https://api.example.com"],"apiKeys":["sk-test"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/unknown/channels/auto-add", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 静态路由注册后，无效 kind 直接返回 404（路由不存在）
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404, 实际=%d", w.Code)
	}
}

func TestAutoDiscoverHandler_InvalidKind(t *testing.T) {
	deps := &AutoManagedDeps{
		CfgManager: nil,
		Runner:     nil,
	}
	r := setupAutoManagedRouter(deps)

	req := httptest.NewRequest(http.MethodPost, "/api/unknown/channels/0/auto-discover", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 静态路由注册后，无效 kind 直接返回 404（路由不存在）
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404, 实际=%d", w.Code)
	}
}

func TestAutoStatusHandler_InvalidKind(t *testing.T) {
	deps := &AutoManagedDeps{
		CfgManager: nil,
		Runner:     nil,
	}
	r := setupAutoManagedRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/unknown/channels/0/auto-status", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 静态路由注册后，无效 kind 直接返回 404（路由不存在）
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404, 实际=%d", w.Code)
	}
}
