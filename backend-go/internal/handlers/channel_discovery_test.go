package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestBuildTransientDiscoveryChannelRequiresBaseURLAndKey(t *testing.T) {
	req := ChannelDiscoveryRequest{ServiceType: "openai"}
	_, err := buildTransientDiscoveryChannel(req)
	if err == nil {
		t.Fatal("expected error for missing base URL and api key")
	}
}

func TestBuildTransientDiscoveryChannelAllowsServiceTypeDetection(t *testing.T) {
	req := ChannelDiscoveryRequest{
		BaseURLs: []string{"https://api.example.com"},
		APIKey:   "sk-test",
	}

	channel, err := buildTransientDiscoveryChannel(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel.ServiceType != "" {
		t.Fatalf("serviceType=%q, want empty before discovery", channel.ServiceType)
	}
}

func TestBuildTransientDiscoveryChannelDoesNotNeedConfigManager(t *testing.T) {
	req := ChannelDiscoveryRequest{
		ChannelKind:        "responses",
		ServiceType:        "openai",
		BaseURLs:           []string{"https://api.example.com/v1"},
		APIKey:             "sk-test",
		AuthHeader:         "bearer",
		CustomHeaders:      map[string]string{"X-Test": "yes"},
		ProxyURL:           "http://127.0.0.1:8080",
		InsecureSkipVerify: true,
		ModelMapping:       map[string]string{"gpt": "actual-main"},
	}

	channel, err := buildTransientDiscoveryChannel(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel.Name != "临时发现渠道" || channel.ServiceType != "openai" {
		t.Fatalf("unexpected channel identity: %#v", channel)
	}
	if channel.BaseURL != "https://api.example.com/v1" {
		t.Fatalf("baseUrl = %q", channel.BaseURL)
	}
	if got := channel.GetAllBaseURLs(); len(got) != 1 || got[0] != "https://api.example.com" {
		t.Fatalf("canonical base urls = %#v", got)
	}
	if len(channel.APIKeys) != 1 || channel.APIKeys[0] != "sk-test" {
		t.Fatalf("api keys = %#v", channel.APIKeys)
	}
	if !channel.InsecureSkipVerify || channel.ProxyURL == "" || channel.AuthHeader != "bearer" {
		t.Fatalf("transport fields not copied: %#v", channel)
	}
	if channel.ModelMapping["gpt"] != "actual-main" {
		t.Fatalf("model mapping not copied: %#v", channel.ModelMapping)
	}
}

func TestBuildDiscoveryMappingRecommendationUsesOnlySuccessfulModels(t *testing.T) {
	selected := DiscoverySelectedModels{Strong: "actual-pro", Primary: "actual-main", Fast: "actual-mini"}
	successByProtocol := map[string][]string{"responses": {"actual-main", "actual-mini"}}

	rec := buildDiscoveryMappingRecommendation("responses", "", selected, successByProtocol, []string{"codex"})
	if rec.ChannelKind != "responses" {
		t.Fatalf("channelKind=%q", rec.ChannelKind)
	}
	if rec.ModelMapping["gpt"] != "actual-main" || rec.ModelMapping["mini"] != "actual-mini" {
		t.Fatalf("unexpected mapping: %#v", rec.ModelMapping)
	}
	if rec.ModelMapping["codex"] == "actual-pro" {
		t.Fatalf("codex should not map to failed actual-pro: %#v", rec.ModelMapping)
	}
	if _, ok := rec.ModelMapping["gpt-5"]; ok {
		t.Fatalf("codex recommendation should keep stable source aliases only: %#v", rec.ModelMapping)
	}
	if rec.ReasoningMapping["gpt"] != "max" || rec.ReasoningMapping["mini"] != "high" || rec.ReasoningMapping["codex"] != "high" {
		t.Fatalf("unexpected reasoning mapping: %#v", rec.ReasoningMapping)
	}
	if len(rec.SupportedModels) != 0 {
		t.Fatalf("discovery should not set supportedModels: %#v", rec.SupportedModels)
	}
	if len(rec.Evidence) != 1 || rec.Evidence[0].Type != "reasoning" || !strings.Contains(rec.Evidence[0].Message, "验证工具调用与思考回传") {
		t.Fatalf("reasoning evidence should explain follow-up capability probes: %#v", rec.Evidence)
	}
}

func TestBuildDiscoveryMappingRecommendationUsesStableClaudeSourceAliases(t *testing.T) {
	selected := DiscoverySelectedModels{Strong: "claude-opus-4-7", Primary: "claude-opus-4-7", Fast: "claude-opus-4-7"}
	successByProtocol := map[string][]string{"messages": {"claude-opus-4-7"}}

	rec := buildDiscoveryMappingRecommendation("messages", "", selected, successByProtocol, []string{"claude-code"})
	wantSources := map[string]string{
		"fable":  "claude-opus-4-7",
		"haiku":  "claude-opus-4-7",
		"opus":   "claude-opus-4-7",
		"sonnet": "claude-opus-4-7",
	}
	if len(rec.ModelMapping) != len(wantSources) {
		t.Fatalf("unexpected source alias count: %#v", rec.ModelMapping)
	}
	for source, target := range wantSources {
		if rec.ModelMapping[source] != target {
			t.Fatalf("modelMapping[%q]=%q, want %q; full mapping=%#v", source, rec.ModelMapping[source], target, rec.ModelMapping)
		}
	}
	for _, internalRole := range []string{"strong", "primary", "fast"} {
		if _, ok := rec.ModelMapping[internalRole]; ok {
			t.Fatalf("internal role %q should not be exposed as source alias: %#v", internalRole, rec.ModelMapping)
		}
	}
	if rec.ReasoningMapping["opus"] != "max" || rec.ReasoningMapping["sonnet"] != "max" || rec.ReasoningMapping["haiku"] != "high" || rec.ReasoningMapping["fable"] != "max" {
		t.Fatalf("unexpected Claude reasoning mapping: %#v", rec.ReasoningMapping)
	}
	if len(rec.SupportedModels) != 0 {
		t.Fatalf("discovery should not set supportedModels: %#v", rec.SupportedModels)
	}
}

// 当 messages 协议探测失败但 responses 协议成功时，
// buildDiscoveryMappingRecommendation 应降级使用 responses 的模型，
// 确保 Claude 别名映射仍能生成（避免发现结果空映射）。
func TestBuildDiscoveryMappingRecommendationFallsBackWhenChannelProtocolFails(t *testing.T) {
	selected := DiscoverySelectedModels{Strong: "gpt-5.5", Primary: "gpt-5.4", Fast: "gpt-5.4-mini"}
	// messages 协议失败，responses 协议成功
	successByProtocol := map[string][]string{
		"responses": {"gpt-5.4", "gpt-5.4-mini", "gpt-5.5"},
	}

	rec := buildDiscoveryMappingRecommendation("messages", "", selected, successByProtocol, []string{"claude-code"})

	if len(rec.ModelMapping) == 0 {
		t.Fatal("expected non-empty modelMapping when falling back to successful protocol models; got empty")
	}
	for _, alias := range []string{"opus", "sonnet", "haiku", "fable"} {
		if _, ok := rec.ModelMapping[alias]; !ok {
			t.Fatalf("expected Claude alias %q in modelMapping; got %#v", alias, rec.ModelMapping)
		}
	}
	if len(rec.SupportedModels) != 0 {
		t.Fatalf("discovery should not set supportedModels: %#v", rec.SupportedModels)
	}
}

func TestRecommendDiscoveryChannelKindKeepsExplicitRequestedProtocol(t *testing.T) {
	protocols := []DiscoveryProtocolResult{
		{Protocol: "messages", Success: false},
		{Protocol: "responses", Success: true},
	}

	got := recommendDiscoveryChannelKind("messages", []string{"codex"}, protocols)
	if got != "messages" {
		t.Fatalf("recommended channelKind=%q, want messages", got)
	}
}

func TestRecommendDiscoveryChannelKindFallsBackWithoutExplicitProtocol(t *testing.T) {
	protocols := []DiscoveryProtocolResult{
		{Protocol: "messages", Success: true},
		{Protocol: "responses", Success: true},
	}

	got := recommendDiscoveryChannelKind("", []string{"codex"}, protocols)
	if got != "responses" {
		t.Fatalf("recommended channelKind=%q, want responses", got)
	}
}

func TestResolveDiscoveryServiceType(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		protocol  string
		want      string
	}{
		{name: "preserve explicit", requested: "openai", protocol: "responses", want: "openai"},
		{name: "messages", protocol: "messages", want: "claude"},
		{name: "responses", protocol: "responses", want: "responses"},
		{name: "chat", protocol: "chat", want: "openai"},
		{name: "gemini", protocol: "gemini", want: "gemini"},
		{name: "unknown", protocol: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveDiscoveryServiceType(tt.requested, tt.protocol); got != tt.want {
				t.Fatalf("resolveDiscoveryServiceType(%q, %q)=%q, want %q", tt.requested, tt.protocol, got, tt.want)
			}
		})
	}
}

func TestDiscoveryProtocolProbeModelsCanCoverEveryDiscoveredModel(t *testing.T) {
	models := DiscoveryModelsResult{
		Items:    []string{"model-a", "model-b", "model-c", "model-d", "model-e", "model-f", "model-g"},
		Selected: DiscoverySelectedModels{Strong: "model-g", Primary: "model-a", Fast: "model-b"},
	}
	got := discoveryProtocolProbeModels(models, true)
	if len(got) != len(models.Items) {
		t.Fatalf("全量探测模型数=%d, want %d: %v", len(got), len(models.Items), got)
	}
	for i, model := range models.Items {
		if got[i] != model {
			t.Fatalf("全量探测顺序[%d]=%q, want %q", i, got[i], model)
		}
	}

	limited := discoveryProtocolProbeModels(models, false)
	if len(limited) >= len(models.Items) {
		t.Fatalf("普通发现仍应使用代表模型，got %v", limited)
	}
}

func TestRunDiscoveryProtocolProbeMatchesClaudeCodeOnlyRelay(t *testing.T) {
	models := []string{"claude-opus-4-7", "claude-opus-4-8"}
	seen := make(map[string]bool, len(models))
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "claude-cli/"+claudeCodeProbeVersion) ||
			r.Header.Get("X-App") == "" ||
			r.Header.Get("anthropic-beta") == "" ||
			r.Header.Get("anthropic-version") == "" {
			http.Error(w, "not a Claude Code client", http.StatusServiceUnavailable)
			return
		}

		var body struct {
			Model  string `json:"model"`
			System []struct {
				Text string `json:"text"`
			} `json:"system"`
			Metadata struct {
				UserID string `json:"user_id"`
			} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if len(body.System) == 0 || !strings.HasPrefix(body.System[0].Text, "x-anthropic-billing-header") ||
			!strings.Contains(body.System[0].Text, "cc_entrypoint=") {
			http.Error(w, "missing Claude Code billing block", http.StatusServiceUnavailable)
			return
		}
		var userID claudeCodeProbeUserID
		if err := json.Unmarshal([]byte(body.Metadata.UserID), &userID); err != nil ||
			userID.DeviceID == "" || userID.SessionID == "" ||
			r.Header.Get("X-Claude-Code-Session-Id") != userID.SessionID {
			http.Error(w, "invalid Claude Code metadata", http.StatusServiceUnavailable)
			return
		}

		seen[body.Model] = true
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
		_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer upstream.Close()

	channel := &config.UpstreamConfig{
		Name:        "Claude Code only",
		BaseURL:     upstream.URL,
		BaseURLs:    []string{upstream.URL},
		APIKeys:     []string{"sk-test"},
		ServiceType: "claude",
	}
	result := runDiscoveryProtocolProbe(context.Background(), channel, "messages", models, 5*time.Second, nil)
	if !result.Success || !sameStringSet(result.SuccessModels, models) {
		t.Fatalf("probe result=%#v, want both Opus models successful", result)
	}
	for _, model := range models {
		if !seen[model] {
			t.Fatalf("model %q was not probed", model)
		}
	}
}

func TestBuildClaudeCompatRequestUsesClaudeCodeClientFingerprint(t *testing.T) {
	channel := &config.UpstreamConfig{AuthHeader: "bearer"}
	req, err := buildClaudeCompatRequest(
		"https://api.example.com",
		buildClaudeToolCallProbeBody("claude-opus-4-7"),
		channel,
		"sk-test",
	)
	if err != nil {
		t.Fatalf("buildClaudeCompatRequest() error = %v", err)
	}
	defer req.Body.Close()

	var body struct {
		System []struct {
			Text string `json:"text"`
		} `json:"system"`
		Metadata struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if len(body.System) < 2 || body.System[0].Text != claudeCodeProbeBillingHeader {
		t.Fatalf("system=%#v, want billing block before probe prompt", body.System)
	}
	var userID claudeCodeProbeUserID
	if err := json.Unmarshal([]byte(body.Metadata.UserID), &userID); err != nil {
		t.Fatalf("decode metadata.user_id: %v", err)
	}
	if userID.SessionID == "" || req.Header.Get("X-Claude-Code-Session-Id") != userID.SessionID {
		t.Fatalf("session header=%q metadata=%#v", req.Header.Get("X-Claude-Code-Session-Id"), userID)
	}
	if req.Header.Get("User-Agent") != claudeCodeProbeUserAgent ||
		req.Header.Get("anthropic-dangerous-direct-browser-access") != "true" {
		t.Fatalf("unexpected Claude Code headers: %#v", req.Header)
	}
}

func TestSelectDiscoveryModelsPrefersToolCapableModels(t *testing.T) {
	global := map[string]config.UpstreamModelCapability{
		"plain-main": {
			ContextWindowTokens: 1000000,
			Capabilities:        map[string]bool{"toolCalls": false},
		},
		"tool-main": {
			ContextWindowTokens: 200000,
			Capabilities:        map[string]bool{"toolCalls": true},
		},
	}

	selected := selectDiscoveryModels([]string{"plain-main", "tool-main"}, global)
	if selected.Primary != "tool-main" {
		t.Fatalf("primary=%q, want tool-main", selected.Primary)
	}
}

func TestDiscoveryRecommendationUsesMiMoVisionCapabilities(t *testing.T) {
	models := []string{"mimo-v2.5", "mimo-v2.5-pro"}
	selected := selectDiscoveryModels(models, nil)
	successByProtocol := map[string][]string{"messages": models}

	rec := buildDiscoveryMappingRecommendation("messages", "", selected, successByProtocol, []string{"claude-code"})
	applyDiscoveryModelCapabilityRecommendations(&rec, models, successByProtocol["messages"], nil)

	for source, target := range rec.ModelMapping {
		if target != "mimo-v2.5-pro" {
			t.Fatalf("modelMapping[%q]=%q, want mimo-v2.5-pro; full mapping=%#v", source, target, rec.ModelMapping)
		}
	}
	if !sameStringSet(rec.NoVisionModels, []string{"mimo-v2.5-pro"}) {
		t.Fatalf("noVisionModels=%#v", rec.NoVisionModels)
	}
	if rec.VisionFallbackModel != "mimo-v2.5" {
		t.Fatalf("visionFallbackModel=%q, want mimo-v2.5", rec.VisionFallbackModel)
	}
}

func sameStringSet(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(got))
	for _, item := range got {
		seen[item]++
	}
	for _, item := range want {
		if seen[item] == 0 {
			return false
		}
		seen[item]--
	}
	return true
}

func TestChannelDiscoveryHandlerDiscoversTransientResponsesChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-main"},{"id":"actual-mini"}]}`))
		case "/v1/responses":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
				t.Fatalf("Authorization header = %q", got)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"responses",
		"serviceType":"openai",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["codex"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Models.Source != "models_endpoint" {
		t.Fatalf("models source=%q", resp.Models.Source)
	}
	if resp.Recommendation.ChannelKind != "responses" {
		t.Fatalf("recommended channelKind=%q", resp.Recommendation.ChannelKind)
	}
	if resp.Recommendation.ModelMapping["gpt"] != "actual-main" {
		t.Fatalf("modelMapping=%#v", resp.Recommendation.ModelMapping)
	}
	var responsesOK bool
	for _, protocol := range resp.Protocols {
		if protocol.Protocol == "responses" && protocol.Success {
			responsesOK = true
			break
		}
	}
	if !responsesOK {
		t.Fatalf("protocols=%#v", resp.Protocols)
	}
}

func TestChannelDiscoveryAutoDetectsServiceTypeAndProbesEveryModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	models := []string{"actual-a", "actual-b", "actual-c", "actual-d", "actual-e", "actual-f", "actual-g"}
	seenResponsesModels := make(map[string]bool, len(models))
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-a"},{"id":"actual-b"},{"id":"actual-c"},{"id":"actual-d"},{"id":"actual-e"},{"id":"actual-f"},{"id":"actual-g"}]}`))
		case "/v1/responses":
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Model != "" {
				seenResponsesModels[body.Model] = true
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))
	body := []byte(`{
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Recommendation.ChannelKind != "responses" || resp.Recommendation.ServiceType != "responses" {
		t.Fatalf("recommendation=%#v", resp.Recommendation)
	}
	var responsesModels []string
	for _, protocol := range resp.Protocols {
		if protocol.Protocol == "responses" {
			responsesModels = protocol.SuccessModels
			break
		}
	}
	if !sameStringSet(responsesModels, models) {
		t.Fatalf("responses successModels=%#v, want %#v", responsesModels, models)
	}
	for _, model := range models {
		if !seenResponsesModels[model] {
			t.Fatalf("model %q was not probed through responses", model)
		}
	}
}

func TestChannelDiscoveryUsesInjectedModelsFetcher(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/anthropic/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()

	fetchers := ChannelDiscoveryModelFetchers{
		"messages": func(_ context.Context, req DiscoveryModelsFetchRequest) (DiscoveryModelsFetchResponse, error) {
			if req.BaseURL != upstream.URL+"/anthropic" {
				t.Fatalf("baseURL=%q", req.BaseURL)
			}
			if req.APIKey != "sk-test" || req.ServiceType != "claude" {
				t.Fatalf("unexpected fetch request: %#v", req)
			}
			return DiscoveryModelsFetchResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"object":"list","data":[{"id":"actual-main"},{"id":"actual-mini"}]}`),
			}, nil
		},
	}

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscoveryWithModelFetchers(nil, fetchers))

	body := []byte(`{
		"channelKind":"messages",
		"serviceType":"claude",
		"baseUrls":["` + upstream.URL + `/anthropic"],
		"apiKey":"sk-test"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Models.Source != "messages_models_handler" {
		t.Fatalf("models source=%q", resp.Models.Source)
	}
	if resp.Models.StatusCode != http.StatusOK {
		t.Fatalf("models status=%d", resp.Models.StatusCode)
	}
	if got := strings.Join(resp.Models.Items, ","); got != "actual-main,actual-mini" {
		t.Fatalf("models=%q", got)
	}
	for _, warning := range resp.Models.Warnings {
		if strings.Contains(warning, "404") || strings.Contains(warning, "built-in probe") {
			t.Fatalf("unexpected fallback warning: %#v", resp.Models.Warnings)
		}
	}
}

func TestDiscoverAutoTransientModelsMergesGenericAndGeminiLists(t *testing.T) {
	channel := &config.UpstreamConfig{
		BaseURL:  "https://api.example.com",
		BaseURLs: []string{"https://api.example.com"},
		APIKeys:  []string{"sk-test"},
	}
	seenServiceTypes := make(map[string]string)
	fetchers := ChannelDiscoveryModelFetchers{
		"messages": func(_ context.Context, req DiscoveryModelsFetchRequest) (DiscoveryModelsFetchResponse, error) {
			seenServiceTypes["messages"] = req.ServiceType
			return DiscoveryModelsFetchResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"data":[{"id":"shared"},{"id":"generic-only"}]}`),
			}, nil
		},
		"gemini": func(_ context.Context, req DiscoveryModelsFetchRequest) (DiscoveryModelsFetchResponse, error) {
			seenServiceTypes["gemini"] = req.ServiceType
			return DiscoveryModelsFetchResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"models":[{"name":"models/shared"},{"name":"models/gemini-only"}]}`),
			}, nil
		},
	}

	result := discoverAutoTransientModelsWithFetchers(context.Background(), channel, "sk-test", fetchers)
	if !sameStringSet(result.Items, []string{"shared", "generic-only", "gemini-only"}) {
		t.Fatalf("models=%#v", result.Items)
	}
	if result.Source != "messages_models_handler+gemini_models_handler" {
		t.Fatalf("source=%q", result.Source)
	}
	if seenServiceTypes["messages"] != "claude" || seenServiceTypes["gemini"] != "gemini" {
		t.Fatalf("service types=%#v", seenServiceTypes)
	}
}

func TestChannelDiscoveryCompatUsesDiscoveredActualModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var sawDefaultCompatModel bool
	var sawActualCompatToolProbe bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-main"}]}`))
		case "/v1/responses":
			body, _ := io.ReadAll(r.Body)
			if bytes.Contains(body, []byte("gpt-5.4-mini")) {
				sawDefaultCompatModel = true
			}
			if bytes.Contains(body, []byte("image_generation")) && bytes.Contains(body, []byte("actual-main")) {
				sawActualCompatToolProbe = true
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"responses",
		"serviceType":"responses",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["codex"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if sawDefaultCompatModel {
		t.Fatal("compat probe used default gpt-5.4-mini instead of discovered actual model")
	}
	if !sawActualCompatToolProbe {
		t.Fatal("expected image_generation compat probe to use discovered actual model")
	}
}

func TestChannelDiscoveryReportsResponsesToolCallCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var sawToolProbe bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-main"}]}`))
		case "/v1/responses":
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "text/event-stream")
			if bytes.Contains(body, []byte(`"name":"ccx_probe"`)) {
				sawToolProbe = true
				_, _ = w.Write([]byte("event: response.output_item.added\ndata: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"ccx_probe\"}}\n\n"))
				_, _ = w.Write([]byte("event: response.function_call_arguments.delta\ndata: {\"type\":\"response.function_call_arguments.delta\",\"delta\":\"{\\\"value\\\":\\\"ok\\\"}\"}\n\n"))
				return
			}
			_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"responses",
		"serviceType":"responses",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["codex"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !sawToolProbe {
		t.Fatal("expected tool-call probe request")
	}

	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Capabilities.ToolCalls.Tested || !resp.Capabilities.ToolCalls.Supported {
		t.Fatalf("tool capability=%#v", resp.Capabilities.ToolCalls)
	}
	if !resp.Recommendation.Compat["codexNativeToolPassthrough"] {
		t.Fatalf("expected codexNativeToolPassthrough recommendation, compat=%#v", resp.Recommendation.Compat)
	}
}

func TestChannelDiscoveryReportsVisionCapabilityAndFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var sawVisionProbeFallback bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"mimo-v2.5"},{"id":"mimo-v2.5-pro"}]}`))
		case "/v1/messages":
			body, _ := io.ReadAll(r.Body)
			if bytes.Contains(body, []byte(`"type":"image"`)) {
				if bytes.Contains(body, []byte(`"model":"mimo-v2.5"`)) {
					sawVisionProbeFallback = true
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
			_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"messages",
		"serviceType":"claude",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["claude-code"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !sawVisionProbeFallback {
		t.Fatal("expected vision probe to use mimo-v2.5 fallback model")
	}

	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Capabilities.Vision.Tested || !resp.Capabilities.Vision.Supported {
		t.Fatalf("vision capability=%#v", resp.Capabilities.Vision)
	}
	for source, target := range resp.Recommendation.ModelMapping {
		if target != "mimo-v2.5-pro" {
			t.Fatalf("modelMapping[%q]=%q, want mimo-v2.5-pro; full mapping=%#v", source, target, resp.Recommendation.ModelMapping)
		}
	}
	if !sameStringSet(resp.Recommendation.NoVisionModels, []string{"mimo-v2.5-pro"}) {
		t.Fatalf("noVisionModels=%#v", resp.Recommendation.NoVisionModels)
	}
	if resp.Recommendation.VisionFallbackModel != "mimo-v2.5" {
		t.Fatalf("visionFallbackModel=%q", resp.Recommendation.VisionFallbackModel)
	}
}

func TestChannelDiscoveryReportsClaudeThinkingPassbackCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var sawToolProbe bool
	var sawHistoricalThinkingProbe bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-claude"}]}`))
		case "/v1/messages":
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "text/event-stream")
			switch {
			case bytes.Contains(body, []byte(`"name":"ccx_probe"`)):
				sawToolProbe = true
				_, _ = w.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"ccx_probe\",\"input\":{}}}\n\n"))
				_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"value\\\":\\\"ok\\\"}\"}}\n\n"))
				_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
			case bytes.Contains(body, []byte("previous reasoning")):
				sawHistoricalThinkingProbe = true
				_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
				_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
			case bytes.Contains(body, []byte(`"thinking"`)):
				_, _ = w.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\"}}\n\n"))
				_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"analysis\"}}\n\n"))
				_, _ = w.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"))
				_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
				_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
			default:
				_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n"))
				_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"messages",
		"serviceType":"claude",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["claude-code"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !sawToolProbe {
		t.Fatal("expected tool-call probe request")
	}
	if !sawHistoricalThinkingProbe {
		t.Fatal("expected historical thinking passback probe request")
	}

	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Capabilities.ToolCalls.Tested || !resp.Capabilities.ToolCalls.Supported {
		t.Fatalf("tool capability=%#v", resp.Capabilities.ToolCalls)
	}
	if !resp.Capabilities.ThinkingPassback.Tested || !resp.Capabilities.ThinkingPassback.Required {
		t.Fatalf("thinking capability=%#v", resp.Capabilities.ThinkingPassback)
	}
	if !resp.Recommendation.Compat["passbackReasoningContent"] || !resp.Recommendation.Compat["passbackThinkingBlocks"] {
		t.Fatalf("thinking passback recommendations missing: %#v", resp.Recommendation.Compat)
	}
}
