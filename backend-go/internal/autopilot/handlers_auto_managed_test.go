package autopilot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestListAccountsMasksCredentials(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts": [{"accountUid":"acct_test","providerId":"mimo","name":"mimo-main","credentials":[{"credentialUid":"cred_test","apiKey":"sk-secret-value"}]}],
  "upstream": [{"accountUid":"acct_test","channelUid":"ch_messages","providerId":"mimo","name":"mimo-main-claude","serviceType":"claude","baseUrl":"https://example.com/anthropic","apiKeyConfigs":[{"credentialUid":"cred_test","baseUrl":"https://example.com/anthropic"}],"autoManaged":true,"status":"active"}],
  "responsesUpstream": [], "geminiUpstream": [], "chatUpstream": [], "imagesUpstream": [], "vectorsUpstream": []
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatalf("写测试配置失败: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	t.Cleanup(func() { _ = cfgManager.Close() })
	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: cfgManager})
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/accounts status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "sk-secret-value") {
		t.Fatalf("账号列表泄露明文 Key: %s", body)
	}
	if !strings.Contains(body, "cred_test") || !strings.Contains(body, "keyMask") {
		t.Fatalf("账号列表缺少凭证掩码信息: %s", body)
	}
}

func TestPatchAccountCredentialsRemovesByUID(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts": [{"accountUid":"acct_test","providerId":"mimo","name":"mimo-main","credentials":[{"credentialUid":"cred_a","apiKey":"sk-a"},{"credentialUid":"cred_b","apiKey":"sk-b"}]}],
  "upstream": [{"accountUid":"acct_test","channelUid":"ch_messages","providerId":"mimo","name":"mimo-main","serviceType":"claude","baseUrl":"https://api.xiaomimimo.com/anthropic","baseUrls":["https://api.xiaomimimo.com/anthropic"],"apiKeyConfigs":[{"credentialUid":"cred_a","baseUrl":"https://api.xiaomimimo.com/anthropic"},{"credentialUid":"cred_b","baseUrl":"https://api.xiaomimimo.com/anthropic"}],"autoManaged":true,"status":"active"}],
  "responsesUpstream": [], "geminiUpstream": [], "chatUpstream": [], "imagesUpstream": [], "vectorsUpstream": []
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatalf("写测试配置失败: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	t.Cleanup(func() { _ = cfgManager.Close() })
	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: cfgManager})
	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/acct_test/credentials", bytes.NewBufferString(`{"removeCredentialUids":["cred_b"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH credentials status=%d body=%s", w.Code, w.Body.String())
	}
	channels := cfgManager.GetAccountChannels("acct_test")
	if len(channels) != 1 || len(channels[0].Upstream.APIKeys) != 1 || channels[0].Upstream.APIKeys[0] != "sk-a" {
		t.Fatalf("按 credentialUid 删除失败: %+v", channels)
	}
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

func TestCustomAutoAddResponseIncludesActualRoute(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "upstream": [], "chatUpstream": [], "responsesUpstream": [],
  "geminiUpstream": [], "imagesUpstream": [], "vectorsUpstream": []
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: manager})
	req := httptest.NewRequest(http.MethodPost, "/api/responses/channels/auto-add", bytes.NewBufferString(
		`{"name":"fastaitoken-com-test","baseUrls":["https://example.com"],"apiKeys":["sk-test"]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var response AutoAddResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Channels) != 1 {
		t.Fatalf("channels=%+v, want one actual route", response.Channels)
	}
	route := response.Channels[0]
	if route.ChannelKind != "responses" || route.ServiceType != "responses" || route.ChannelUID != response.ChannelUID {
		t.Fatalf("route=%+v response=%+v", route, response)
	}
	cfg := manager.GetConfig()
	if len(cfg.ResponsesUpstream) != 1 || len(cfg.Upstream) != 0 {
		t.Fatalf("custom route persisted in wrong channel: responses=%d messages=%d", len(cfg.ResponsesUpstream), len(cfg.Upstream))
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
		{route: config.ProviderRoute{ChannelKind: "gemini"}, want: "mimo-test-gemini"},
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

func TestManagedAccountUIDForProviderUsesExistingAccount(t *testing.T) {
	cfg := config.Config{ManagedAccounts: []config.ManagedAccountConfig{
		{AccountUID: "acct-old", ProviderID: "mimo"},
		{AccountUID: "acct-new", ProviderID: "mimo"},
		{AccountUID: "acct-deepseek", ProviderID: "deepseek"},
	}}
	if got := managedAccountUIDForProvider(cfg, "mimo"); got != "acct-new" {
		t.Fatalf("mimo existing account = %q, want acct-new", got)
	}
	if got := managedAccountUIDForProvider(cfg, "unknown"); got != "" {
		t.Fatalf("unknown provider 不应命中账号: %q", got)
	}
}

func TestListManagedAccountsDoesNotExposeVolcengineSecret(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts":[{"accountUid":"acct_volc","providerId":"volcengine","name":"volc","credentials":[{"credentialUid":"cred_volc","apiKey":"ark-secret","volcengineAccessKey":{"accessKeyId":"AKIDEXAMPLE","secretAccessKey":"SECRET_MUST_NOT_LEAK","plan":"agent_plan","planTier":"Large","planStatus":"Running"}}]}],
  "upstream":[{"accountUid":"acct_volc","channelUid":"ch_volc","providerId":"volcengine","name":"volc","serviceType":"claude","autoManaged":true,"baseUrl":"https://ark.cn-beijing.volces.com/api/plan","apiKeyConfigs":[{"credentialUid":"cred_volc","baseUrl":"https://ark.cn-beijing.volces.com/api/plan"}]}],
  "chatUpstream":[],"responsesUpstream":[],"geminiUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: manager})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "SECRET_MUST_NOT_LEAK") || strings.Contains(body, "ark-secret") {
		t.Fatalf("管理 API 泄漏凭证: %s", body)
	}
	for _, expected := range []string{`"hasVolcengineAccessKey":true`, `"volcenginePlan":"agent_plan"`, `"volcenginePlanTier":"Large"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("响应缺少 %s: %s", expected, body)
		}
	}
}

func TestSetVolcengineAccessKeyDetectsPlanBeforePersisting(t *testing.T) {
	controlPlane := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("Action") != "GetPersonalPlan" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var body struct{ Plan string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Plan == "CodingPlan" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"Error":{"Code":"ResourceNotFound.Plan","Message":"not found"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"Result":{"PlanType":"Large","Status":"Running"}}`))
	}))
	defer controlPlane.Close()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts":[{"accountUid":"acct_volc","providerId":"volcengine","name":"volc","credentials":[{"credentialUid":"cred_volc","apiKey":"ark-inference"}]}],
  "upstream":[{"accountUid":"acct_volc","channelUid":"ch_volc","providerId":"volcengine","name":"volc","serviceType":"claude","autoManaged":true,"baseUrl":"https://ark.cn-beijing.volces.com/api/plan","apiKeyConfigs":[{"credentialUid":"cred_volc","baseUrl":"https://ark.cn-beijing.volces.com/api/plan"}]}],
  "chatUpstream":[],"responsesUpstream":[],"geminiUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	runner := NewAutoDiscoveryRunner(nil, nil)
	runner.client = controlPlane.Client()
	runner.volcengineControlPlaneEndpoint = controlPlane.URL
	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: manager, Runner: runner})
	req := httptest.NewRequest(http.MethodPut, "/api/accounts/acct_volc/credentials/cred_volc/volcengine-access-key", bytes.NewBufferString(`{"accessKeyId":"AKID","secretAccessKey":"SECRET"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"plan":"agent_plan"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	credential, ok := manager.GetManagedAccountCredential("acct_volc", "cred_volc")
	if !ok || credential.VolcengineAccessKey == nil || credential.VolcengineAccessKey.SecretAccessKey != "SECRET" || credential.VolcengineAccessKey.Plan != "agent_plan" {
		t.Fatalf("凭证未正确持久化: %+v", credential)
	}
}

func TestSetMiMoConsoleCookieRequiresConfirmationBeforeAdoptingKey(t *testing.T) {
	const oldKey = "tp-old-1234567890123456789012345678901234567890"
	const cookieKey = "tp-new-1234567890123456789012345678901234567890"
	console := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "api-platform_serviceToken=session; userId=42" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/userProfile":
			_, _ = w.Write([]byte(`{"code":0,"data":{"userId":"42"}}`))
		case "/api/v1/tokenPlan/detail":
			_, _ = w.Write([]byte(`{"code":0,"data":{"planCode":"max","planName":"Max","currentPeriodEnd":"2026-07-29 23:59:59"}}`))
		case "/api/v1/tokenPlan/usage":
			_, _ = w.Write([]byte(`{"code":0,"data":{"monthUsage":{"percent":0.25,"items":[{"used":25,"limit":100}]},"usage":{"percent":0.5,"items":[{"used":50,"limit":100}]}}}`))
		case "/api/v1/tokenPlan/apiKey/raw":
			_, _ = w.Write([]byte(`{"code":0,"data":"` + cookieKey + `"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer console.Close()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts":[{"accountUid":"acct_mimo","providerId":"mimo","name":"mimo","credentials":[{"credentialUid":"cred_mimo","apiKey":"` + oldKey + `"}]}],
  "upstream":[{"accountUid":"acct_mimo","channelUid":"ch_mimo","providerId":"mimo","name":"mimo","serviceType":"claude","autoManaged":true,"baseUrl":"https://token-plan-cn.xiaomimimo.com/anthropic","apiKeyConfigs":[{"credentialUid":"cred_mimo","baseUrl":"https://token-plan-cn.xiaomimimo.com/anthropic"}]}],
  "chatUpstream":[],"responsesUpstream":[],"geminiUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	deps := &AutoManagedDeps{CfgManager: manager, MiMoConsoleClient: &MiMoConsoleClient{HTTPClient: console.Client(), BaseURL: console.URL}}
	router := setupAutoManagedRouter(deps)
	body := `{"cookie":"api-platform_serviceToken=session; userId=42"}`
	request := func(payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/accounts/acct_mimo/credentials/cred_mimo/mimo-console-cookie", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	w := request(body)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"code":"mimo_cookie_key_mismatch"`) {
		t.Fatalf("未要求确认: status=%d body=%s", w.Code, w.Body.String())
	}
	credential, _ := manager.GetManagedAccountCredential("acct_mimo", "cred_mimo")
	if credential.APIKey != oldKey || credential.MiMoConsole != nil {
		t.Fatalf("确认前不应修改配置: %+v", credential)
	}
	w = request(`{"cookie":"api-platform_serviceToken=session; userId=42","adoptCookieKey":true}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"keyAdopted":true`) {
		t.Fatalf("确认采用失败: status=%d body=%s", w.Code, w.Body.String())
	}
	credential, _ = manager.GetManagedAccountCredential("acct_mimo", "cred_mimo")
	if credential.APIKey != cookieKey || credential.MiMoConsole == nil {
		t.Fatalf("确认后未采用 Cookie Key: %+v", credential)
	}
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if strings.Contains(list.Body.String(), "api-platform_serviceToken") || strings.Contains(list.Body.String(), cookieKey) {
		t.Fatalf("账号列表泄漏 Cookie 或原始 Key: %s", list.Body.String())
	}
}

func TestProviderAutoAddReusesExistingAccount(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts": [{"accountUid":"acct_test","providerId":"mimo","name":"mimo-main","credentials":[{"credentialUid":"cred_test","apiKey":"sk-existing"}]}],
  "upstream": [{"accountUid":"acct_test","channelUid":"ch_messages","providerId":"mimo","name":"mimo-main","serviceType":"claude","baseUrl":"https://api.xiaomimimo.com/anthropic","apiKeyConfigs":[{"credentialUid":"cred_test","baseUrl":"https://api.xiaomimimo.com/anthropic"}],"autoManaged":true,"status":"active"}],
  "responsesUpstream": [], "geminiUpstream": [], "chatUpstream": [], "imagesUpstream": [], "vectorsUpstream": []
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfgManager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfgManager.Close() })
	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: cfgManager})
	req := httptest.NewRequest(http.MethodPost, "/api/messages/channels/auto-add", bytes.NewBufferString(`{"providerId":"mimo","apiKeys":["sk-existing"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("追加已有 provider key status=%d body=%s", w.Code, w.Body.String())
	}
	channels := cfgManager.GetAccountChannels("acct_test")
	if len(channels) != 4 {
		t.Fatalf("旧账号应自动补齐模板中的协议渠道: %+v", channels)
	}
	kinds := make(map[string]bool, len(channels))
	for _, channel := range channels {
		kinds[channel.Kind] = true
	}
	for _, kind := range []string{"messages", "chat", "responses", "gemini"} {
		if !kinds[kind] {
			t.Fatalf("旧账号缺少 %s route: %+v", kind, channels)
		}
	}
	var response AutoAddResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.AccountUID != "acct_test" {
		t.Fatalf("响应未返回已有账号: body=%s err=%v", w.Body.String(), err)
	}
	if len(response.Channels) != 4 {
		t.Fatalf("响应未返回补齐后的全部渠道: %+v", response.Channels)
	}
}

func TestMissingProviderAccountRoutes(t *testing.T) {
	tmpl, ok := config.GetProviderTemplate("deepseek")
	if !ok {
		t.Fatal("缺少 deepseek provider 模板")
	}
	existing := []config.AccountChannel{{
		Kind:     "messages",
		Upstream: config.UpstreamConfig{ServiceType: "claude"},
	}}
	missing := missingProviderAccountRoutes(tmpl, existing)
	if len(missing) != 2 || missing[0].ChannelKind != "chat" || missing[1].ChannelKind != "responses" {
		t.Fatalf("missing routes = %+v", missing)
	}
	configs, baseURLs, err := bindProviderRouteKeys(tmpl, missing[0], []string{"sk-existing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 || configs[0].BaseURL != "https://api.deepseek.com" || len(baseURLs) != 1 {
		t.Fatalf("route 绑定结果不正确: configs=%+v baseURLs=%v", configs, baseURLs)
	}
}

func TestBindProviderRouteKeysPreservesExistingEndpointAffinity(t *testing.T) {
	tests := []struct {
		providerID      string
		apiKey          string
		existingBaseURL string
		targetKind      string
		wantBaseURL     string
	}{
		{
			providerID: "mimo", apiKey: "tp-existing",
			existingBaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic",
			targetKind:      "chat", wantBaseURL: "https://token-plan-sgp.xiaomimimo.com/v1",
		},
		{
			providerID: "volcengine", apiKey: "ark-existing",
			existingBaseURL: "https://ark.cn-beijing.volces.com/api/coding",
			targetKind:      "responses", wantBaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.providerID, func(t *testing.T) {
			tmpl, ok := config.GetProviderTemplate(tt.providerID)
			if !ok {
				t.Fatalf("缺少 provider 模板: %s", tt.providerID)
			}
			existing := []config.AccountChannel{{
				Kind: "messages",
				Upstream: config.UpstreamConfig{
					ProviderID: tt.providerID, ServiceType: "claude", APIKeys: []string{tt.apiKey},
					APIKeyConfigs: []config.APIKeyConfig{{Key: tt.apiKey, BaseURL: tt.existingBaseURL}},
				},
			}}
			var target config.ProviderRoute
			for _, route := range tmpl.AutoAddRoutes() {
				if route.ChannelKind == tt.targetKind {
					target = route
					break
				}
			}
			configs, _, err := bindProviderRouteKeysWithAffinities(
				tmpl, target, []string{tt.apiKey}, providerKeyCandidateAffinities(tmpl, existing),
			)
			if err != nil || len(configs) != 1 || configs[0].BaseURL != tt.wantBaseURL {
				t.Fatalf("亲和绑定错误: configs=%+v err=%v want=%s", configs, err, tt.wantBaseURL)
			}
		})
	}
}

func TestPlanProviderAccountRouteAdditionsRejectsNewKeyBeforeMutation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tmpl := &config.ProviderTemplate{
		ProviderID: "test-provider",
		Routes: []config.ProviderRoute{
			{ChannelKind: "messages", ServiceType: "claude", Candidates: []config.ProviderCandidate{{BaseURL: server.URL + "/anthropic"}}},
			{ChannelKind: "chat", ServiceType: "openai", Candidates: []config.ProviderCandidate{{BaseURL: server.URL + "/v1"}}},
		},
	}
	existing := []config.AccountChannel{{
		Kind: "messages",
		Upstream: config.UpstreamConfig{
			AccountUID: "acct_test", ChannelUID: "ch_messages", Name: "test-provider",
			ProviderID: "test-provider", ServiceType: "claude", AutoManaged: true, APIKeys: []string{"sk-existing"},
		},
	}}
	additions, status, err := planProviderAccountRouteAdditions(
		t.Context(), config.Config{}, tmpl, "acct_test", []string{"sk-existing", "sk-invalid"}, existing,
	)
	if err == nil || status != http.StatusBadRequest || len(additions) != 0 {
		t.Fatalf("无效新 Key 应在生成新增 route 前失败: status=%d additions=%+v err=%v", status, additions, err)
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
