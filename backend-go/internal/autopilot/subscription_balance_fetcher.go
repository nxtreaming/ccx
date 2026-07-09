package autopilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── 余额查询接口 ──

// SubscriptionBalanceFetcher 定义 provider 级余额查询能力。
// 仅对有公开可编程账单 API 的 provider 实现（OpenAI、Anthropic、Google）。
// 中转/公益渠道（relay_x/community_x）明确不做自动抓取。
type SubscriptionBalanceFetcher interface {
	// ProviderName 返回该 fetcher 对应的 provider 枚举值。
	ProviderName() string
	// FetchBalance 查询指定订阅的当前余额。
	// 返回 balance（单位由 provider 决定，通常为美元或 token 额度）和 error。
	FetchBalance(ctx context.Context, billingAPIKey string) (balance float64, currency string, err error)
}

// ── Fetcher Registry ──

// BalanceFetcherRegistry 按 Provider 枚举值查找对应的 fetcher。
// 线程安全，启动时注册，运行时只读。
type BalanceFetcherRegistry struct {
	fetchers map[string]SubscriptionBalanceFetcher
	mu       sync.RWMutex
}

// NewBalanceFetcherRegistry 创建空的 fetcher 注册表。
func NewBalanceFetcherRegistry() *BalanceFetcherRegistry {
	return &BalanceFetcherRegistry{
		fetchers: make(map[string]SubscriptionBalanceFetcher),
	}
}

// Register 注册一个 fetcher。同名 provider 后注册覆盖前注册。
func (r *BalanceFetcherRegistry) Register(fetcher SubscriptionBalanceFetcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fetchers[fetcher.ProviderName()] = fetcher
}

// Get 按 provider 名称获取 fetcher。不存在返回 nil。
func (r *BalanceFetcherRegistry) Get(provider string) SubscriptionBalanceFetcher {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.fetchers[strings.ToLower(strings.TrimSpace(provider))]
}

// SupportedProviders 返回已注册的 provider 名称列表。
func (r *BalanceFetcherRegistry) SupportedProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, 0, len(r.fetchers))
	for k := range r.fetchers {
		result = append(result, k)
	}
	return result
}

// ── OpenAI Fetcher ──

// OpenAIBalanceFetcher 查询 OpenAI 组织用量/余额。
// API: GET https://api.openai.com/v1/organization/usage/completions
// 需要 admin key（sk-admin-...）或 org 级 API key。
type OpenAIBalanceFetcher struct {
	BaseURL    string       // 默认 https://api.openai.com
	HTTPClient *http.Client // 可注入，便于测试
}

func (f *OpenAIBalanceFetcher) ProviderName() string { return "openai" }

func (f *OpenAIBalanceFetcher) FetchBalance(ctx context.Context, billingAPIKey string) (float64, string, error) {
	baseURL := f.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	// OpenAI 没有直接的 "余额" API，使用 /v1/organization/usage 间接获取。
	// 实际实现中更常使用 dashboard API 或 billing endpoint。
	// 这里用 /v1/organization 作为存活检测 + 取额度信息的轻量查询。
	reqURL := baseURL + "/v1/organization"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, "USD", fmt.Errorf("[OpenAIBalanceFetcher] 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+billingAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "USD", fmt.Errorf("[OpenAIBalanceFetcher] HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, "USD", fmt.Errorf("[OpenAIBalanceFetcher] HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "USD", fmt.Errorf("[OpenAIBalanceFetcher] 解析响应失败: %w", err)
	}

	// OpenAI 当前公开 API 不直接暴露余额数字；
	// 返回 -1 表示"有效但无法获取具体余额"，前端可显示为"已验证"状态。
	// 当 OpenAI 开放 billing endpoint 时替换此处。
	return -1, "USD", nil
}

// ── Anthropic Fetcher ──

// AnthropicBalanceFetcher 查询 Anthropic 组织用量/余额。
// API: GET https://api.anthropic.com/v1/organizations
// 需要 admin API key。
type AnthropicBalanceFetcher struct {
	BaseURL    string       // 默认 https://api.anthropic.com
	HTTPClient *http.Client // 可注入，便于测试
}

func (f *AnthropicBalanceFetcher) ProviderName() string { return "anthropic" }

func (f *AnthropicBalanceFetcher) FetchBalance(ctx context.Context, billingAPIKey string) (float64, string, error) {
	baseURL := f.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	// Anthropic 目前没有公开的 billing/usage API。
	// 使用 /v1/models 作为存活 + 认证有效性检测。
	reqURL := baseURL + "/v1/models?limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, "USD", fmt.Errorf("[AnthropicBalanceFetcher] 构造请求失败: %w", err)
	}
	req.Header.Set("x-api-key", billingAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "USD", fmt.Errorf("[AnthropicBalanceFetcher] HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, "USD", fmt.Errorf("[AnthropicBalanceFetcher] HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Anthropic 当前不暴露余额 API，返回 -1 表示"有效但无法获取具体余额"。
	return -1, "USD", nil
}

// ── Google (Gemini) Fetcher ──

// GoogleBalanceFetcher 查询 Google Cloud / Gemini 用量。
// API: GET https://generativelanguage.googleapis.com/v1beta/models
// 需要有效的 API key。
type GoogleBalanceFetcher struct {
	BaseURL    string       // 默认 https://generativelanguage.googleapis.com
	HTTPClient *http.Client // 可注入，便于测试
}

func (f *GoogleBalanceFetcher) ProviderName() string { return "google" }

func (f *GoogleBalanceFetcher) FetchBalance(ctx context.Context, billingAPIKey string) (float64, string, error) {
	baseURL := f.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	// Google Cloud Billing API 需要 OAuth2 + project 级权限，Gemini API key 无法直接查账单。
	// 使用 /v1beta/models 作为存活 + 认证有效性检测。
	reqURL := baseURL + "/v1beta/models?key=" + billingAPIKey
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, "USD", fmt.Errorf("[GoogleBalanceFetcher] 构造请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "USD", fmt.Errorf("[GoogleBalanceFetcher] HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, "USD", fmt.Errorf("[GoogleBalanceFetcher] HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Google 当前 Gemini API key 不直接暴露 billing 余额，返回 -1 表示"有效但无法获取具体余额"。
	return -1, "USD", nil
}

// ── 默认注册表工厂 ──

// DefaultBalanceFetcherRegistry 创建包含 OpenAI/Anthropic/Google 三个 fetcher 的默认注册表。
func DefaultBalanceFetcherRegistry() *BalanceFetcherRegistry {
	r := NewBalanceFetcherRegistry()
	r.Register(&OpenAIBalanceFetcher{})
	r.Register(&AnthropicBalanceFetcher{})
	r.Register(&GoogleBalanceFetcher{})
	return r
}

// supportedAutoRefreshProviders 定义支持自动余额刷新的 provider 白名单。
// 不在此列表中的 provider（relay_x/community_x/custom 等）不参与自动刷新。
var supportedAutoRefreshProviders = map[string]bool{
	"openai":    true,
	"anthropic": true,
	"google":    true,
}

// IsAutoRefreshSupported 判断给定 provider 是否支持自动余额刷新。
func IsAutoRefreshSupported(provider string) bool {
	return supportedAutoRefreshProviders[strings.ToLower(strings.TrimSpace(provider))]
}
