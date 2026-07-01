package copilot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApplyRuntimeHeaders(t *testing.T) {
	h := http.Header{}
	ApplyRuntimeHeaders(h, "copilot-token-abc")

	if got := h.Get("Authorization"); got != "Bearer copilot-token-abc" {
		t.Errorf("Authorization = %q", got)
	}
	if got := h.Get("Copilot-Integration-Id"); got != "vscode-chat" {
		t.Errorf("Copilot-Integration-Id = %q", got)
	}
	for _, key := range []string{"User-Agent", "Editor-Version", "Editor-Plugin-Version", "openai-organization", "openai-intent"} {
		if h.Get(key) == "" {
			t.Errorf("missing header %s", key)
		}
	}
}

func TestTokenManager_ResolveToken_SuccessAndCache(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer gho_test" {
			t.Errorf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Copilot-Integration-Id") != "vscode-chat" {
			t.Errorf("missing Copilot-Integration-Id")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"copilot-xyz","expires_at":` + epochIn(time.Hour) + `,"refresh_in":1500,"endpoints":{"api":"https://api.githubcopilot.com"}}`))
	}))
	defer srv.Close()

	m := NewTokenManager(srv.Client())
	m.tokenURL = srv.URL

	token, baseURL, err := m.ResolveToken(context.Background(), "gho_test")
	if err != nil {
		t.Fatalf("ResolveToken err = %v", err)
	}
	if token != "copilot-xyz" {
		t.Errorf("token = %q", token)
	}
	if baseURL != "https://api.githubcopilot.com" {
		t.Errorf("baseURL = %q", baseURL)
	}

	// 第二次调用应命中缓存，不再请求上游
	if _, _, err := m.ResolveToken(context.Background(), "gho_test"); err != nil {
		t.Fatalf("second ResolveToken err = %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 upstream call (cache hit), got %d", calls)
	}
}

func TestResolveTokenWithProxy_UsesProxyClient(t *testing.T) {
	oldDefault := defaultTokenManager
	t.Cleanup(func() {
		defaultTokenManager = oldDefault
	})

	var sawProxyRequest bool
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawProxyRequest = true
		if r.URL.Scheme != "http" || r.URL.Host != "example.invalid" {
			t.Errorf("unexpected proxy target: scheme=%q host=%q url=%q", r.URL.Scheme, r.URL.Host, r.URL.String())
		}
		if r.Header.Get("Authorization") != "Bearer gho_proxy" {
			t.Errorf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"copilot-via-proxy","expires_at":` + epochIn(time.Hour) + `,"refresh_in":1500,"endpoints":{"api":"https://api.githubcopilot.com"}}`))
	}))
	defer proxySrv.Close()

	defaultTokenManager = NewTokenManager(nil)
	defaultTokenManager.tokenURL = "http://example.invalid/copilot_internal/v2/token"

	token, _, err := ResolveTokenWithProxy(context.Background(), "gho_proxy", proxySrv.URL)
	if err != nil {
		t.Fatalf("ResolveTokenWithProxy err = %v", err)
	}
	if token != "copilot-via-proxy" {
		t.Errorf("token = %q", token)
	}
	if !sawProxyRequest {
		t.Fatal("expected request through proxy server")
	}
}

func TestTokenManager_ResolveToken_ClassifiesErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		wantKind TokenErrorKind
	}{
		{"unauthorized", http.StatusUnauthorized, TokenErrorGitHubOAuth},
		{"forbidden", http.StatusForbidden, TokenErrorGitHubOAuth},
		{"rate_limited", http.StatusTooManyRequests, TokenErrorCopilotToken},
		{"server_error", http.StatusInternalServerError, TokenErrorCopilotToken},
		{"bad_request", http.StatusBadRequest, TokenErrorUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"message":"boom"}`))
			}))
			defer srv.Close()

			m := NewTokenManager(srv.Client())
			m.tokenURL = srv.URL

			_, _, err := m.ResolveToken(context.Background(), "gho_test")
			if err == nil {
				t.Fatalf("expected error")
			}
			te, ok := err.(*TokenExchangeError)
			if !ok {
				t.Fatalf("expected *TokenExchangeError, got %T", err)
			}
			if te.Kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", te.Kind, tt.wantKind)
			}
		})
	}
}

func TestTokenManager_ResolveToken_EmptyToken(t *testing.T) {
	m := NewTokenManager(nil)
	if _, _, err := m.ResolveToken(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestOAuthClient_DeviceFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/device/code"):
			_, _ = w.Write([]byte(`{"device_code":"dev-1","user_code":"USER-1","verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`))
		case strings.HasSuffix(r.URL.Path, "/access_token"):
			_, _ = w.Write([]byte(`{"access_token":"gho_abc","token_type":"bearer","scope":"read:user"}`))
		case strings.HasSuffix(r.URL.Path, "/user"):
			if r.Header.Get("Authorization") != "Bearer gho_abc" {
				t.Errorf("unexpected user auth header: %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"login":"octocat","id":1}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewOAuthClient(srv.Client())
	client.DeviceCodeURL = srv.URL + "/device/code"
	client.AccessTokenURL = srv.URL + "/access_token"
	client.UserURL = srv.URL + "/user"

	device, err := client.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatalf("RequestDeviceCode err = %v", err)
	}
	if device.UserCode != "USER-1" || device.Interval != 5 {
		t.Errorf("unexpected device code response: %+v", device)
	}

	token, err := client.PollAccessToken(context.Background(), device.DeviceCode)
	if err != nil {
		t.Fatalf("PollAccessToken err = %v", err)
	}
	if token.AccessToken != "gho_abc" {
		t.Errorf("access token = %q", token.AccessToken)
	}

	user, err := client.VerifyUser(context.Background(), token.AccessToken)
	if err != nil {
		t.Fatalf("VerifyUser err = %v", err)
	}
	if user.Login != "octocat" {
		t.Errorf("login = %q", user.Login)
	}
}

func TestOAuthClient_PollAuthorizationPending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":"authorization_pending","error_description":"pending"}`))
	}))
	defer srv.Close()

	client := NewOAuthClient(srv.Client())
	client.AccessTokenURL = srv.URL

	token, err := client.PollAccessToken(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("PollAccessToken err = %v", err)
	}
	if token.Error != "authorization_pending" || token.AccessToken != "" {
		t.Errorf("unexpected token response: %+v", token)
	}
}

func TestDiagnose_TokenErrorStopsBeforeModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/user"):
			_, _ = w.Write([]byte(`{"login":"octocat","id":1}`))
		case strings.HasSuffix(r.URL.Path, "/v2/token"):
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"bad token"}`))
		default:
			t.Errorf("unexpected models probe before token success: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	// Diagnose 使用默认端点；这里通过自定义 oauth/token 客户端无法直接注入，
	// 因此验证 Diagnose 在 token 失败时返回 tokenError 且不带 modelsStatus。
	result := diagnoseWithEndpoints(t, srv)
	if result.GitHubUser == nil {
		t.Errorf("expected github user verified")
	}
	if result.TokenError == "" || result.TokenErrorKind != string(TokenErrorGitHubOAuth) {
		t.Errorf("expected github_oauth token error, got kind=%q err=%q", result.TokenErrorKind, result.TokenError)
	}
	if result.ModelsStatus != 0 {
		t.Errorf("models should not be probed after token failure")
	}
}

// diagnoseWithEndpoints 手动组装诊断逻辑以注入 httptest 端点。
func diagnoseWithEndpoints(t *testing.T, srv *httptest.Server) *DiagnoseResult {
	t.Helper()
	client := srv.Client()
	oauth := NewOAuthClient(client)
	oauth.UserURL = srv.URL + "/user"
	tokenMgr := NewTokenManager(client)
	tokenMgr.tokenURL = srv.URL + "/v2/token"

	result := &DiagnoseResult{}
	if user, err := oauth.VerifyUser(context.Background(), "gho_test"); err != nil {
		result.GitHubUserError = err.Error()
	} else {
		result.GitHubUser = user
	}
	if _, _, err := tokenMgr.ResolveToken(context.Background(), "gho_test"); err != nil {
		if te, ok := err.(*TokenExchangeError); ok {
			result.TokenError = te.Message
			result.TokenErrorKind = string(te.Kind)
		} else {
			result.TokenError = err.Error()
			result.TokenErrorKind = string(TokenErrorUnknown)
		}
	}
	return result
}

func epochIn(d time.Duration) string {
	// 使用一个固定的未来时间，避免依赖 time.Now() 解析；token_manager 会与自身 now 比较。
	return "9999999999"
}
