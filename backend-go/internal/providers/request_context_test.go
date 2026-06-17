package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

type testContextKey string

func newGinContext(method, url string, body []byte, ctx context.Context) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	c.Request = req
	return c
}

func TestClaudeProvider_ConvertToProviderRequest_PassbackConvertsRealThinking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{
		"model": "mimo-v2.5-pro",
		"messages": [
			{"role": "assistant", "content": [
				{"type": "thinking", "thinking": "real reasoning"},
				{"type": "text", "text": "answer"}
			]}
		]
	}`)
	c := newGinContext(http.MethodPost, "/v1/messages", body, context.Background())
	upstream := &config.UpstreamConfig{
		BaseURL:                  "https://api.example.com",
		ServiceType:              "claude",
		PassbackReasoningContent: true,
	}

	p := &ClaudeProvider{}
	_, reqBody, err := p.ConvertToProviderRequest(c, upstream, "sk-ant-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest() err = %v", err)
	}
	if !bytes.Contains(reqBody, []byte(`"reasoning_content":"real reasoning"`)) {
		t.Fatalf("request body missing reasoning_content: %s", string(reqBody))
	}
	if !bytes.Contains(reqBody, []byte(`"type":"thinking"`)) {
		t.Fatalf("request body should keep real thinking block for compatibility: %s", string(reqBody))
	}
}

func TestClaudeProvider_InjectsReasoningEffortForRedirectedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{
		"model": "claude-opus-4-xhigh",
		"messages": [{"role": "user", "content": "hi"}],
		"thinking": {"type": "enabled", "effort": "xhigh", "budget_tokens": 32000},
		"output_config": {"effort": "xhigh"},
		"reasoning": {"effort": "xhigh"},
		"reasoning_effort": "xhigh"
	}`)
	c := newGinContext(http.MethodPost, "/v1/messages", body, context.Background())
	upstream := &config.UpstreamConfig{
		BaseURL:     "https://api.example.com",
		ServiceType: "claude",
		ModelMapping: map[string]string{
			"claude-opus-4": "claude-sonnet-4",
		},
		ReasoningMapping: map[string]string{
			"claude-opus-4": "max",
		},
	}

	p := &ClaudeProvider{}
	_, reqBody, err := p.ConvertToProviderRequest(c, upstream, "sk-ant-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest() err = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(reqBody, &got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if got["model"] != "claude-sonnet-4" {
		t.Fatalf("model = %v, want claude-sonnet-4; body=%s", got["model"], string(reqBody))
	}
	thinking, ok := got["thinking"].(map[string]interface{})
	if !ok {
		t.Fatalf("thinking = %#v, want object; body=%s", got["thinking"], string(reqBody))
	}
	if thinking["type"] != "enabled" || thinking["effort"] != "max" {
		t.Fatalf("thinking = %#v, want type=enabled effort=max", thinking)
	}
	if _, exists := thinking["budget_tokens"]; exists {
		t.Fatalf("thinking should not keep budget_tokens when effort is mapped: %#v", thinking)
	}
	if _, exists := got["reasoning"]; exists {
		t.Fatalf("reasoning should be removed from Claude passthrough body: %#v", got["reasoning"])
	}
	if _, exists := got["reasoning_effort"]; exists {
		t.Fatalf("reasoning_effort should be removed from Claude passthrough body: %#v", got["reasoning_effort"])
	}
	outputConfig, ok := got["output_config"].(map[string]interface{})
	if !ok {
		t.Fatalf("output_config = %#v, want object; body=%s", got["output_config"], string(reqBody))
	}
	if outputConfig["effort"] != "max" {
		t.Fatalf("output_config.effort = %v, want max", outputConfig["effort"])
	}
}

func TestClaudeProvider_DisablesThinkingForNoneReasoningEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{
		"model": "claude-sonnet-4",
		"messages": [{"role": "user", "content": "hi"}],
		"thinking": {"type": "enabled", "effort": "xhigh", "budget_tokens": 32000},
		"output_config": {"effort": "xhigh", "foo": "bar"}
	}`)
	c := newGinContext(http.MethodPost, "/v1/messages", body, context.Background())
	upstream := &config.UpstreamConfig{
		BaseURL:     "https://api.example.com",
		ServiceType: "claude",
		ReasoningMapping: map[string]string{
			"claude-sonnet-4": "none",
		},
	}

	p := &ClaudeProvider{}
	_, reqBody, err := p.ConvertToProviderRequest(c, upstream, "sk-ant-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest() err = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(reqBody, &got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	thinking, ok := got["thinking"].(map[string]interface{})
	if !ok {
		t.Fatalf("thinking = %#v, want object; body=%s", got["thinking"], string(reqBody))
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want type=disabled", thinking)
	}
	if _, exists := thinking["effort"]; exists {
		t.Fatalf("disabled thinking should not keep effort: %#v", thinking)
	}
	if _, exists := thinking["budget_tokens"]; exists {
		t.Fatalf("disabled thinking should not keep budget_tokens: %#v", thinking)
	}
	outputConfig, ok := got["output_config"].(map[string]interface{})
	if !ok {
		t.Fatalf("output_config = %#v, want object; body=%s", got["output_config"], string(reqBody))
	}
	if _, exists := outputConfig["effort"]; exists {
		t.Fatalf("disabled output_config should not keep effort: %#v", outputConfig)
	}
	if outputConfig["foo"] != "bar" {
		t.Fatalf("output_config should keep unrelated fields: %#v", outputConfig)
	}
}

func TestConvertToProviderRequest_PropagatesContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	key := testContextKey("test-key")
	ctx := context.WithValue(context.Background(), key, "ok")

	t.Run("claude", func(t *testing.T) {
		c := newGinContext(http.MethodPost, "/v1/messages", []byte(`{"model":"claude-3","messages":[]}`), ctx)
		upstream := &config.UpstreamConfig{BaseURL: "https://api.example.com", ServiceType: "claude"}

		p := &ClaudeProvider{}
		req, _, err := p.ConvertToProviderRequest(c, upstream, "sk-ant-test")
		if err != nil {
			t.Fatalf("ConvertToProviderRequest() err = %v", err)
		}
		if got := req.Context().Value(key); got != "ok" {
			t.Fatalf("req.Context().Value(key) = %v, want %v", got, "ok")
		}
	})

	t.Run("openai", func(t *testing.T) {
		c := newGinContext(http.MethodPost, "/v1/messages", []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`), ctx)
		upstream := &config.UpstreamConfig{BaseURL: "https://api.example.com", ServiceType: "openai"}

		p := &OpenAIProvider{}
		req, _, err := p.ConvertToProviderRequest(c, upstream, "sk-test")
		if err != nil {
			t.Fatalf("ConvertToProviderRequest() err = %v", err)
		}
		if got := req.Context().Value(key); got != "ok" {
			t.Fatalf("req.Context().Value(key) = %v, want %v", got, "ok")
		}
	})

	t.Run("gemini", func(t *testing.T) {
		c := newGinContext(http.MethodPost, "/v1/messages", []byte(`{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"hi"}]}`), ctx)
		upstream := &config.UpstreamConfig{BaseURL: "https://api.example.com", ServiceType: "gemini"}

		p := &GeminiProvider{}
		req, _, err := p.ConvertToProviderRequest(c, upstream, "AIza-test")
		if err != nil {
			t.Fatalf("ConvertToProviderRequest() err = %v", err)
		}
		if got := req.Context().Value(key); got != "ok" {
			t.Fatalf("req.Context().Value(key) = %v, want %v", got, "ok")
		}
	})

	t.Run("responses", func(t *testing.T) {
		c := newGinContext(http.MethodPost, "/v1/responses", []byte(`{"model":"gpt-4o","input":"hi"}`), ctx)
		upstream := &config.UpstreamConfig{BaseURL: "https://api.example.com", ServiceType: "responses"}

		p := &ResponsesProvider{}
		req, _, err := p.ConvertToProviderRequest(c, upstream, "sk-test")
		if err != nil {
			t.Fatalf("ConvertToProviderRequest() err = %v", err)
		}
		if got := req.Context().Value(key); got != "ok" {
			t.Fatalf("req.Context().Value(key) = %v, want %v", got, "ok")
		}
	})
}

func TestConvertToProviderRequest_UsesUpdatedRequestBodyBytesContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-3","messages":[],"metadata":{"user_id":"{\"device_id\":\"abc\"}"}}`)
	normalized := []byte(`{"model":"claude-3","messages":[],"metadata":{"user_id":"user_abc"}}`)
	c := newGinContext(http.MethodPost, "/v1/messages", body, context.Background())
	c.Set("requestBodyBytes", normalized)
	upstream := &config.UpstreamConfig{BaseURL: "https://api.example.com", ServiceType: "claude"}

	p := &ClaudeProvider{}
	_, reqBody, err := p.ConvertToProviderRequest(c, upstream, "sk-ant-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest() err = %v", err)
	}
	if string(reqBody) != string(normalized) {
		t.Fatalf("request body = %s, want %s", string(reqBody), string(normalized))
	}
}

func TestOpenAIProvider_ConvertToProviderRequest_MapsMetadataUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-4o","metadata":{"user_id":"deepseek_user_123"},"messages":[{"role":"user","content":"hi"}]}`)
	c := newGinContext(http.MethodPost, "/v1/messages", body, context.Background())
	upstream := &config.UpstreamConfig{BaseURL: "https://api.example.com", ServiceType: "openai"}

	p := &OpenAIProvider{}
	req, _, err := p.ConvertToProviderRequest(c, upstream, "sk-test")
	if err != nil {
		t.Fatalf("ConvertToProviderRequest() err = %v", err)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if got["user_id"] != "deepseek_user_123" {
		t.Fatalf("user_id = %v, want deepseek_user_123", got["user_id"])
	}
}
