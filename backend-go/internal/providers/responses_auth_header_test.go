package providers

import (
	"context"
	"net/http"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestResponsesProvider_ConvertToProviderRequest_AuthHeaderByServiceType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		serviceType    string
		apiKey         string
		authHeader     string
		wantXAPIKey    string
		wantAuthBearer string
		wantGoogKey    string
		wantAnthropicV string
	}{
		{
			name:           "claude uses x-api-key",
			serviceType:    "claude",
			apiKey:         "sk-test-key-123",
			wantXAPIKey:    "sk-test-key-123",
			wantAuthBearer: "",
			wantAnthropicV: "2023-06-01",
		},
		{
			name:           "claude with non sk-ant key still uses x-api-key",
			serviceType:    "claude",
			apiKey:         "generic-token-abc",
			wantXAPIKey:    "generic-token-abc",
			wantAuthBearer: "",
			wantAnthropicV: "2023-06-01",
		},
		{
			name:           "openai uses Authorization Bearer",
			serviceType:    "openai",
			apiKey:         "sk-openai-key",
			wantXAPIKey:    "",
			wantAuthBearer: "Bearer sk-openai-key",
			wantAnthropicV: "",
		},
		{
			name:           "claude bearer override",
			serviceType:    "claude",
			apiKey:         "sk-ant-api03-test",
			authHeader:     "bearer",
			wantXAPIKey:    "",
			wantAuthBearer: "Bearer sk-ant-api03-test",
			wantAnthropicV: "2023-06-01",
		},
		{
			name:           "openai x-api-key override",
			serviceType:    "openai",
			apiKey:         "sk-openai-key",
			authHeader:     "x-api-key",
			wantXAPIKey:    "sk-openai-key",
			wantAuthBearer: "",
			wantAnthropicV: "",
		},
		{
			name:        "gemini uses x-goog-api-key",
			serviceType: "gemini",
			apiKey:      "gemini-key-xyz",
			wantXAPIKey: "",
			wantGoogKey: "gemini-key-xyz",
		},
		{
			name:           "default (empty serviceType) uses Authorization Bearer",
			serviceType:    "",
			apiKey:         "fallback-key",
			wantXAPIKey:    "",
			wantAuthBearer: "Bearer fallback-key",
			wantAnthropicV: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newGinContext(http.MethodPost, "/v1/responses", []byte(`{"model":"test","input":"hi"}`), context.Background())
			upstream := &config.UpstreamConfig{
				BaseURL:     "https://api.example.com",
				ServiceType: tt.serviceType,
				AuthHeader:  tt.authHeader,
			}

			provider := &ResponsesProvider{}
			req, _, err := provider.ConvertToProviderRequest(c, upstream, tt.apiKey)
			if err != nil {
				t.Fatalf("ConvertToProviderRequest() err = %v", err)
			}

			if got := req.Header.Get("x-api-key"); got != tt.wantXAPIKey {
				t.Errorf("x-api-key = %q, want %q", got, tt.wantXAPIKey)
			}

			if got := req.Header.Get("authorization"); got != tt.wantAuthBearer {
				t.Errorf("authorization = %q, want %q", got, tt.wantAuthBearer)
			}

			if got := req.Header.Get("x-goog-api-key"); got != tt.wantGoogKey {
				t.Errorf("x-goog-api-key = %q, want %q", got, tt.wantGoogKey)
			}

			if got := req.Header.Get("anthropic-version"); got != tt.wantAnthropicV {
				t.Errorf("anthropic-version = %q, want %q", got, tt.wantAnthropicV)
			}
		})
	}
}
