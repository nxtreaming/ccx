package autopilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	deepSeekAPIBaseURL       = "https://api.deepseek.com"
	deepSeekRequestTimeout   = 10 * time.Second
	deepSeekMaxResponseBytes = 256 << 10
	deepSeekBalanceWorkers   = 4
)

// DeepSeekBalanceInfo 是 DeepSeek 官方余额接口返回的单币种余额。
type DeepSeekBalanceInfo struct {
	Currency        string `json:"currency"`
	TotalBalance    string `json:"totalBalance"`
	GrantedBalance  string `json:"grantedBalance"`
	ToppedUpBalance string `json:"toppedUpBalance"`
}

// DeepSeekBalance 是 DeepSeek GET /user/balance 的规范化响应。
type DeepSeekBalance struct {
	IsAvailable  bool                  `json:"isAvailable"`
	BalanceInfos []DeepSeekBalanceInfo `json:"balanceInfos"`
}

// DeepSeekClient 访问 DeepSeek 官方账号 API。BaseURL 仅用于测试注入。
type DeepSeekClient struct {
	HTTPClient *http.Client
	BaseURL    string
}

func NewDeepSeekClient(client *http.Client) *DeepSeekClient {
	if client == nil {
		client = &http.Client{Timeout: deepSeekRequestTimeout}
	}
	return &DeepSeekClient{HTTPClient: client, BaseURL: deepSeekAPIBaseURL}
}

func (c *DeepSeekClient) FetchBalance(ctx context.Context, apiKey string) (DeepSeekBalance, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" || strings.ContainsAny(apiKey, "\r\n") {
		return DeepSeekBalance{}, fmt.Errorf("DeepSeek API Key 无效")
	}
	baseURL := deepSeekAPIBaseURL
	client := http.DefaultClient
	if c != nil {
		if strings.TrimSpace(c.BaseURL) != "" {
			baseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
		}
		if c.HTTPClient != nil {
			client = c.HTTPClient
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/user/balance", nil)
	if err != nil {
		return DeepSeekBalance{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return DeepSeekBalance{}, fmt.Errorf("请求 DeepSeek 余额失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, deepSeekMaxResponseBytes))
	if err != nil {
		return DeepSeekBalance{}, fmt.Errorf("读取 DeepSeek 余额失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DeepSeekBalance{}, fmt.Errorf("DeepSeek 余额接口返回 HTTP %d%s", resp.StatusCode, deepSeekErrorSuffix(body))
	}

	var wire struct {
		IsAvailable  bool `json:"is_available"`
		BalanceInfos []struct {
			Currency        string `json:"currency"`
			TotalBalance    string `json:"total_balance"`
			GrantedBalance  string `json:"granted_balance"`
			ToppedUpBalance string `json:"topped_up_balance"`
		} `json:"balance_infos"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return DeepSeekBalance{}, fmt.Errorf("DeepSeek 余额响应不是有效 JSON")
	}
	result := DeepSeekBalance{IsAvailable: wire.IsAvailable, BalanceInfos: make([]DeepSeekBalanceInfo, 0, len(wire.BalanceInfos))}
	for _, item := range wire.BalanceInfos {
		result.BalanceInfos = append(result.BalanceInfos, DeepSeekBalanceInfo{
			Currency: strings.ToUpper(strings.TrimSpace(item.Currency)), TotalBalance: item.TotalBalance,
			GrantedBalance: item.GrantedBalance, ToppedUpBalance: item.ToppedUpBalance,
		})
	}
	return result, nil
}

func deepSeekErrorSuffix(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && strings.TrimSpace(envelope.Error.Message) != "" {
		return ": " + strings.TrimSpace(envelope.Error.Message)
	}
	return ""
}
