package providers

import (
	"context"
	"net/http"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestClaudeProvider_ConvertToProviderRequest_AuthHeaderOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		apiKey            string
		authHeader        string
		wantXAPIKey       string
		wantAuthorization string
	}{
		{
			name:              "默认保持非sk-ant密钥使用Bearer",
			apiKey:            "sk-opencode-key",
			wantAuthorization: "Bearer sk-opencode-key",
		},
		{
			name:        "x-api-key覆盖非sk-ant密钥",
			apiKey:      "sk-opencode-key",
			authHeader:  "x-api-key",
			wantXAPIKey: "sk-opencode-key",
		},
		{
			name:              "bearer覆盖sk-ant密钥",
			apiKey:            "sk-ant-api03-test",
			authHeader:        "bearer",
			wantAuthorization: "Bearer sk-ant-api03-test",
		},
		{
			name:        "auto保持sk-ant密钥使用x-api-key",
			apiKey:      "sk-ant-api03-test",
			authHeader:  "auto",
			wantXAPIKey: "sk-ant-api03-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newGinContext(http.MethodPost, "/v1/messages", []byte(`{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":10}`), context.Background())
			upstream := &config.UpstreamConfig{
				BaseURL:     "https://api.example.com",
				ServiceType: "claude",
				AuthHeader:  tt.authHeader,
			}

			provider := &ClaudeProvider{}
			req, _, err := provider.ConvertToProviderRequest(c, upstream, tt.apiKey)
			if err != nil {
				t.Fatalf("ConvertToProviderRequest() err = %v", err)
			}

			if got := req.Header.Get("x-api-key"); got != tt.wantXAPIKey {
				t.Errorf("x-api-key = %q, want %q", got, tt.wantXAPIKey)
			}
			if got := req.Header.Get("authorization"); got != tt.wantAuthorization {
				t.Errorf("authorization = %q, want %q", got, tt.wantAuthorization)
			}
		})
	}
}
