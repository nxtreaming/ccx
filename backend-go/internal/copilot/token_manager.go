package copilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/httpclient"
)

const copilotTokenURL = "https://api.github.com/copilot_internal/v2/token"

type cachedToken struct {
	Token      string
	ExpiresAt  time.Time
	RefreshAt  time.Time
	APIBaseURL string
}

// TokenResponse 是 GitHub Copilot token exchange 的响应。
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	RefreshIn int64  `json:"refresh_in"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

// TokenManager 将 GitHub OAuth token 换成短期 Copilot API token 并缓存。
type TokenManager struct {
	client   *http.Client
	mu       sync.Mutex
	cache    map[string]cachedToken
	now      func() time.Time
	tokenURL string
}

var defaultTokenManager = NewTokenManager(nil)

// NewTokenManager 创建 TokenManager。
func NewTokenManager(client *http.Client) *TokenManager {
	if client == nil {
		client = &http.Client{Timeout: defaultRequestTimout}
	}
	return &TokenManager{
		client:   client,
		cache:    make(map[string]cachedToken),
		now:      time.Now,
		tokenURL: copilotTokenURL,
	}
}

type TokenErrorKind string

const (
	TokenErrorUnknown      TokenErrorKind = "unknown"
	TokenErrorGitHubOAuth  TokenErrorKind = "github_oauth"
	TokenErrorCopilotToken TokenErrorKind = "copilot_token"
)

type TokenExchangeError struct {
	Kind    TokenErrorKind
	Status  int
	Message string
}

func (e *TokenExchangeError) Error() string {
	return e.Message
}

func classifyTokenExchangeError(status int, body string) *TokenExchangeError {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return &TokenExchangeError{
			Kind:    TokenErrorGitHubOAuth,
			Status:  status,
			Message: fmt.Sprintf("GitHub OAuth token 无效或已过期: status=%d body=%s", status, body),
		}
	}
	if status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
		return &TokenExchangeError{
			Kind:    TokenErrorCopilotToken,
			Status:  status,
			Message: fmt.Sprintf("GitHub Copilot token exchange 暂时不可用: status=%d body=%s", status, body),
		}
	}
	return &TokenExchangeError{
		Kind:    TokenErrorUnknown,
		Status:  status,
		Message: fmt.Sprintf("GitHub Copilot token exchange 失败: status=%d body=%s", status, body),
	}
}

// ResolveToken 使用默认 TokenManager 解析 Copilot API token。
func ResolveToken(ctx context.Context, githubToken string) (string, string, error) {
	return defaultTokenManager.ResolveToken(ctx, githubToken)
}

// ResolveTokenWithProxy 使用可选代理解析 Copilot API token。
func ResolveTokenWithProxy(ctx context.Context, githubToken string, proxyURL string) (string, string, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return defaultTokenManager.ResolveToken(ctx, githubToken)
	}
	client := httpclient.GetManager().NewStandardClient(defaultRequestTimout, false, proxyURL)
	return defaultTokenManager.resolveToken(ctx, githubToken, client)
}

// ResolveToken 返回可用 Copilot token 与 API base URL。
func (m *TokenManager) ResolveToken(ctx context.Context, githubToken string) (string, string, error) {
	return m.resolveToken(ctx, githubToken, nil)
}

func (m *TokenManager) resolveToken(ctx context.Context, githubToken string, client *http.Client) (string, string, error) {
	githubToken = strings.TrimSpace(githubToken)
	if githubToken == "" {
		return "", "", fmt.Errorf("GitHub OAuth token 不能为空")
	}

	key := tokenCacheKey(githubToken)
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	if cached, ok := m.cache[key]; ok && cached.Token != "" && now.Before(cached.RefreshAt) && now.Before(cached.ExpiresAt) {
		return cached.Token, cached.APIBaseURL, nil
	}

	tokenResp, err := m.exchange(ctx, githubToken, client)
	if err != nil {
		return "", "", err
	}
	if tokenResp.Token == "" {
		return "", "", fmt.Errorf("GitHub Copilot token exchange 未返回 token")
	}

	expiresAt := time.Unix(tokenResp.ExpiresAt, 0)
	if tokenResp.ExpiresAt <= 0 || expiresAt.Before(now) {
		expiresAt = now.Add(30 * time.Minute)
	}
	refreshAt := expiresAt.Add(-5 * time.Minute)
	if tokenResp.RefreshIn > 0 {
		refreshAt = now.Add(time.Duration(tokenResp.RefreshIn) * time.Second)
	}
	if !refreshAt.Before(expiresAt) {
		refreshAt = expiresAt.Add(-1 * time.Minute)
	}
	if !refreshAt.After(now) {
		refreshAt = now.Add(1 * time.Minute)
	}

	apiBaseURL := strings.TrimRight(tokenResp.Endpoints.API, "/")
	if apiBaseURL == "" {
		apiBaseURL = DefaultAPIBaseURL
	}

	m.cache[key] = cachedToken{
		Token:      tokenResp.Token,
		ExpiresAt:  expiresAt,
		RefreshAt:  refreshAt,
		APIBaseURL: apiBaseURL,
	}
	return tokenResp.Token, apiBaseURL, nil
}

func (m *TokenManager) exchange(ctx context.Context, githubToken string, client *http.Client) (*TokenResponse, error) {
	tokenURL := m.tokenURL
	if tokenURL == "" {
		tokenURL = copilotTokenURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return nil, err
	}
	ApplyGitHubHeaders(req.Header)
	req.Header.Set("Authorization", "Bearer "+githubToken)

	if client == nil {
		client = m.client
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, classifyTokenExchangeError(resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func tokenCacheKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
