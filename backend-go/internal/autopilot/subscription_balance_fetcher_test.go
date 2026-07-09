package autopilot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// ── BalanceFetcherRegistry 测试 ──

func TestBalanceFetcherRegistry_RegisterAndGet(t *testing.T) {
	r := NewBalanceFetcherRegistry()

	// 未注册时返回 nil
	if got := r.Get("openai"); got != nil {
		t.Fatalf("期望 nil, got %v", got)
	}

	// 注册后可获取
	fetcher := &OpenAIBalanceFetcher{}
	r.Register(fetcher)

	got := r.Get("openai")
	if got == nil {
		t.Fatal("注册后未找到 fetcher")
	}
	if got.ProviderName() != "openai" {
		t.Fatalf("ProviderName 不匹配: got=%s, want=openai", got.ProviderName())
	}
}

func TestBalanceFetcherRegistry_SupportedProviders(t *testing.T) {
	r := DefaultBalanceFetcherRegistry()
	providers := r.SupportedProviders()
	if len(providers) != 3 {
		t.Fatalf("期望 3 个 provider, got %d", len(providers))
	}

	// 验证三个 provider 都在
	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[p] = true
	}
	for _, want := range []string{"openai", "anthropic", "google"} {
		if !providerSet[want] {
			t.Errorf("缺少 provider: %s", want)
		}
	}
}

func TestBalanceFetcherRegistry_CaseInsensitive(t *testing.T) {
	r := NewBalanceFetcherRegistry()
	r.Register(&OpenAIBalanceFetcher{})

	// 大小写不敏感
	if got := r.Get("OpenAI"); got == nil {
		t.Fatal("大小写不敏感查找失败: OpenAI")
	}
	if got := r.Get("OPENAI"); got == nil {
		t.Fatal("大小写不敏感查找失败: OPENAI")
	}
}

// ── IsAutoRefreshSupported 测试 ──

func TestIsAutoRefreshSupported(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"google", true},
		{"OpenAI", true},
		{"ANTHROPIC", true},
		{"relay_x", false},
		{"community_x", false},
		{"custom", false},
		{"", false},
		{"  openai  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := IsAutoRefreshSupported(tt.provider)
			if got != tt.want {
				t.Errorf("IsAutoRefreshSupported(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

// ── OpenAIBalanceFetcher 测试 ──

func TestOpenAIBalanceFetcher_FetchBalance_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organization" {
			t.Errorf("期望路径 /v1/organization, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("期望 Authorization=Bearer test-key, got %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{{"id": "org-123"}},
		})
	}))
	defer server.Close()

	fetcher := &OpenAIBalanceFetcher{BaseURL: server.URL, HTTPClient: server.Client()}
	balance, currency, err := fetcher.FetchBalance(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("FetchBalance 返回错误: %v", err)
	}
	if balance != -1 {
		t.Errorf("期望 balance=-1, got %f", balance)
	}
	if currency != "USD" {
		t.Errorf("期望 currency=USD, got %s", currency)
	}
}

func TestOpenAIBalanceFetcher_FetchBalance_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid key"}`)
	}))
	defer server.Close()

	fetcher := &OpenAIBalanceFetcher{BaseURL: server.URL, HTTPClient: server.Client()}
	_, _, err := fetcher.FetchBalance(context.Background(), "bad-key")
	if err == nil {
		t.Fatal("期望返回错误, got nil")
	}
}

// ── AnthropicBalanceFetcher 测试 ──

func TestAnthropicBalanceFetcher_FetchBalance_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("期望路径 /v1/models, got %s", r.URL.Path)
		}
		xKey := r.Header.Get("x-api-key")
		if xKey != "sk-ant-test" {
			t.Errorf("期望 x-api-key=sk-ant-test, got %s", xKey)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer server.Close()

	fetcher := &AnthropicBalanceFetcher{BaseURL: server.URL, HTTPClient: server.Client()}
	balance, _, err := fetcher.FetchBalance(context.Background(), "sk-ant-test")
	if err != nil {
		t.Fatalf("FetchBalance 返回错误: %v", err)
	}
	if balance != -1 {
		t.Errorf("期望 balance=-1, got %f", balance)
	}
}

func TestAnthropicBalanceFetcher_FetchBalance_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error": "forbidden"}`)
	}))
	defer server.Close()

	fetcher := &AnthropicBalanceFetcher{BaseURL: server.URL, HTTPClient: server.Client()}
	_, _, err := fetcher.FetchBalance(context.Background(), "bad-key")
	if err == nil {
		t.Fatal("期望返回错误, got nil")
	}
}

// ── GoogleBalanceFetcher 测试 ──

func TestGoogleBalanceFetcher_FetchBalance_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
		})
	}))
	defer server.Close()

	fetcher := &GoogleBalanceFetcher{BaseURL: server.URL, HTTPClient: server.Client()}
	balance, _, err := fetcher.FetchBalance(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("FetchBalance 返回错误: %v", err)
	}
	if balance != -1 {
		t.Errorf("期望 balance=-1, got %f", balance)
	}
}

func TestGoogleBalanceFetcher_FetchBalance_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error": "API key not valid"}`)
	}))
	defer server.Close()

	fetcher := &GoogleBalanceFetcher{BaseURL: server.URL, HTTPClient: server.Client()}
	_, _, err := fetcher.FetchBalance(context.Background(), "bad-key")
	if err == nil {
		t.Fatal("期望返回错误, got nil")
	}
}

// ── RefreshBudget 测试 ──

func TestRefreshBudget_TryConsume(t *testing.T) {
	budget := NewRefreshBudget(3, time.Now)

	// 初始状态
	if budget.Used() != 0 {
		t.Fatalf("初始 Used=%d, want 0", budget.Used())
	}
	if budget.Remaining() != 3 {
		t.Fatalf("初始 Remaining=%d, want 3", budget.Remaining())
	}

	// 消耗 3 次
	for i := 0; i < 3; i++ {
		if !budget.TryConsume() {
			t.Fatalf("第 %d 次 TryConsume 应该成功", i+1)
		}
	}

	// 第 4 次应该失败
	if budget.TryConsume() {
		t.Fatal("第 4 次 TryConsume 应该失败")
	}
	if budget.Used() != 3 {
		t.Fatalf("Used=%d, want 3", budget.Used())
	}
	if budget.Remaining() != 0 {
		t.Fatalf("Remaining=%d, want 0", budget.Remaining())
	}
}

func TestRefreshBudget_DailyReset(t *testing.T) {
	var now atomic.Int64
	now.Store(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC).Unix())
	timeFunc := func() time.Time {
		return time.Unix(now.Load(), 0).UTC()
	}

	budget := NewRefreshBudget(2, timeFunc)

	// 消耗完
	budget.TryConsume()
	budget.TryConsume()
	if budget.TryConsume() {
		t.Fatal("当日预算应该耗尽")
	}

	// 跨天
	now.Store(time.Date(2025, 1, 2, 1, 0, 0, 0, time.UTC).Unix())
	if !budget.TryConsume() {
		t.Fatal("跨天后预算应该重置")
	}
}

// ── SubscriptionRefreshWorker 集成测试 ──

// mockBalanceFetcher 用于测试的 mock fetcher。
type mockBalanceFetcher struct {
	provider string
	balance  float64
	currency string
	err      error
	callCount atomic.Int32
}

func (m *mockBalanceFetcher) ProviderName() string { return m.provider }

func (m *mockBalanceFetcher) FetchBalance(ctx context.Context, billingAPIKey string) (float64, string, error) {
	m.callCount.Add(1)
	return m.balance, m.currency, m.err
}

func TestSubscriptionRefreshWorker_RefreshAll_DoubleGateControl(t *testing.T) {
	// 创建内存中的 SubscriptionStore
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  42.5,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	// 全局开关关闭时，不应调用 fetcher
	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
		},
		func() bool { return false }, // 全局开关关闭
	)

	// 创建订阅（BillingAPIKey 非空 + AutoRefreshEnabled=true）
	store.Create(&SubscriptionProfile{
		SubscriptionUID:    "sub-1",
		DisplayName:        "Test",
		Provider:           "openai",
		BillingAPIKey:      "sk-admin-test",
		AutoRefreshEnabled: true,
	})

	worker.refreshAll()

	if mockFetcher.callCount.Load() != 0 {
		t.Fatalf("全局开关关闭时不应调用 fetcher, got %d calls", mockFetcher.callCount.Load())
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_BillingAPIKeyRequired(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  42.5,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
		},
		func() bool { return true }, // 全局开关开启
	)

	// BillingAPIKey 为空 → 跳过
	store.Create(&SubscriptionProfile{
		SubscriptionUID:    "sub-no-key",
		DisplayName:        "No Key",
		Provider:           "openai",
		BillingAPIKey:      "",
		AutoRefreshEnabled: true,
	})

	worker.refreshAll()

	if mockFetcher.callCount.Load() != 0 {
		t.Fatalf("BillingAPIKey 为空时不应调用 fetcher, got %d calls", mockFetcher.callCount.Load())
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_AutoRefreshEnabledRequired(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  42.5,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
		},
		func() bool { return true },
	)

	// AutoRefreshEnabled=false → 跳过
	store.Create(&SubscriptionProfile{
		SubscriptionUID:    "sub-disabled",
		DisplayName:        "Disabled",
		Provider:           "openai",
		BillingAPIKey:      "sk-test",
		AutoRefreshEnabled: false,
	})

	worker.refreshAll()

	if mockFetcher.callCount.Load() != 0 {
		t.Fatalf("AutoRefreshEnabled=false 时不应调用 fetcher, got %d calls", mockFetcher.callCount.Load())
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_UnsupportedProvider(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  42.5,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
		},
		func() bool { return true },
	)

	// relay_x 不在白名单 → 跳过
	store.Create(&SubscriptionProfile{
		SubscriptionUID:    "sub-relay",
		DisplayName:        "Relay",
		Provider:           "relay_x",
		BillingAPIKey:      "test",
		AutoRefreshEnabled: true,
	})

	worker.refreshAll()

	if mockFetcher.callCount.Load() != 0 {
		t.Fatalf("relay_x provider 不应被刷新, got %d calls", mockFetcher.callCount.Load())
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_SuccessfulRefresh(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  100.5,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
		},
		func() bool { return true },
	)

	// 满足所有条件的订阅
	store.Create(&SubscriptionProfile{
		SubscriptionUID:    "sub-valid",
		DisplayName:        "Valid",
		Provider:           "openai",
		BillingAPIKey:      "sk-admin-test",
		AutoRefreshEnabled: true,
		Balance:            0,
	})

	worker.refreshAll()

	if mockFetcher.callCount.Load() != 1 {
		t.Fatalf("期望调用 1 次 fetcher, got %d", mockFetcher.callCount.Load())
	}

	// 验证余额已更新
	profile := store.Get("sub-valid")
	if profile == nil {
		t.Fatal("订阅不存在")
	}
	if profile.Balance != 100.5 {
		t.Errorf("Balance=%f, want 100.5", profile.Balance)
	}
	if profile.Currency != "USD" {
		t.Errorf("Currency=%s, want USD", profile.Currency)
	}
	if profile.LastBalanceRefreshAt == nil {
		t.Error("LastBalanceRefreshAt 应该被设置")
	}
	if profile.LastBalanceRefreshError != "" {
		t.Errorf("LastBalanceRefreshError=%q, want empty", profile.LastBalanceRefreshError)
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_FetchError(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "anthropic",
		err:      fmt.Errorf("HTTP 403: forbidden"),
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
		},
		func() bool { return true },
	)

	store.Create(&SubscriptionProfile{
		SubscriptionUID:    "sub-fail",
		DisplayName:        "Fail",
		Provider:           "anthropic",
		BillingAPIKey:      "sk-ant-bad",
		AutoRefreshEnabled: true,
	})

	worker.refreshAll()

	if mockFetcher.callCount.Load() != 1 {
		t.Fatalf("期望调用 1 次 fetcher, got %d", mockFetcher.callCount.Load())
	}

	profile := store.Get("sub-fail")
	if profile == nil {
		t.Fatal("订阅不存在")
	}
	if profile.LastBalanceRefreshError == "" {
		t.Error("失败时 LastBalanceRefreshError 应该非空")
	}
	if profile.LastBalanceRefreshAt == nil {
		t.Error("失败时 LastBalanceRefreshAt 也应该被设置")
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_CooldownRespected(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	var now atomic.Int64
	now.Store(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC).Unix())
	timeFunc := func() time.Time {
		return time.Unix(now.Load(), 0).UTC()
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  50.0,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 24 * time.Hour,
			DailyBudget:     100,
			QuietLogs:       true,
			TimeFunc:        timeFunc,
		},
		func() bool { return true },
	)

	lastRefresh := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC) // 2小时前
	store.Create(&SubscriptionProfile{
		SubscriptionUID:        "sub-cooldown",
		DisplayName:            "Cooldown",
		Provider:               "openai",
		BillingAPIKey:          "sk-test",
		AutoRefreshEnabled:     true,
		LastBalanceRefreshAt:   &lastRefresh,
	})

	// 2小时前刷新过，24h 冷却期内应跳过
	worker.refreshAll()

	if mockFetcher.callCount.Load() != 0 {
		t.Fatalf("冷却期内不应调用 fetcher, got %d calls", mockFetcher.callCount.Load())
	}

	// 推进时间到 24h 后
	now.Store(time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC).Unix())
	worker.refreshAll()

	if mockFetcher.callCount.Load() != 1 {
		t.Fatalf("冷却期过后应调用 fetcher, got %d calls", mockFetcher.callCount.Load())
	}
}

func TestSubscriptionRefreshWorker_RefreshAll_DailyBudget(t *testing.T) {
	store, err := NewSubscriptionStoreWithDB(newTestDB(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}

	mockFetcher := &mockBalanceFetcher{
		provider: "openai",
		balance:  10.0,
		currency: "USD",
	}

	registry := NewBalanceFetcherRegistry()
	registry.Register(mockFetcher)

	// 每日预算仅为 2
	worker := NewSubscriptionRefreshWorker(
		store,
		registry,
		SubscriptionRefreshWorkerConfig{
			RefreshInterval: 1 * time.Hour,
			DailyBudget:     2,
			QuietLogs:       true,
		},
		func() bool { return true },
	)

	// 创建 4 个符合条件的订阅
	for i := 0; i < 4; i++ {
		store.Create(&SubscriptionProfile{
			SubscriptionUID:    fmt.Sprintf("sub-%d", i),
			DisplayName:        fmt.Sprintf("Sub %d", i),
			Provider:           "openai",
			BillingAPIKey:      "sk-test",
			AutoRefreshEnabled: true,
		})
	}

	worker.refreshAll()

	// 预算为 2，应该只调用 2 次
	if mockFetcher.callCount.Load() != 2 {
		t.Fatalf("期望调用 2 次（预算限制），got %d", mockFetcher.callCount.Load())
	}
}

// ── 辅助函数 ──

// newTestDB 定义在 profile_store_test.go 中，此处复用。
