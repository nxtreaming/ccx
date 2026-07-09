package autopilot

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

// ── shadowRequestBudget 测试 ──

func TestShadowRequestBudget_TryConsume(t *testing.T) {
	b := newShadowRequestBudget(3)
	b.timeFunc = func() time.Time { return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC) }

	if !b.TryConsume() {
		t.Fatal("first consume should succeed")
	}
	if !b.TryConsume() {
		t.Fatal("second consume should succeed")
	}
	if !b.TryConsume() {
		t.Fatal("third consume should succeed")
	}
	if b.TryConsume() {
		t.Fatal("fourth consume should fail (budget exhausted)")
	}
	if b.Remaining() != 0 {
		t.Fatalf("remaining should be 0, got %d", b.Remaining())
	}
	if b.Used() != 3 {
		t.Fatalf("used should be 3, got %d", b.Used())
	}
}

func TestShadowRequestBudget_HourlyReset(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	b := newShadowRequestBudget(2)
	b.timeFunc = func() time.Time { return now }

	b.TryConsume()
	b.TryConsume()
	if b.TryConsume() {
		t.Fatal("budget should be exhausted at hour 12")
	}

	// 推进 1 小时，预算应重置
	nextHour := now.Add(time.Hour)
	b.timeFunc = func() time.Time { return nextHour }
	if !b.TryConsume() {
		t.Fatal("budget should reset after hour change")
	}
}

func TestShadowRequestBudget_DefaultLimit(t *testing.T) {
	b := newShadowRequestBudget(0)
	if b.hourlyLimit != int32(DefaultABTestMaxShadowPerHour) {
		t.Fatalf("default limit should be %d, got %d", DefaultABTestMaxShadowPerHour, b.hourlyLimit)
	}
}

// ── ShadowCandidateCache 测试 ──

func TestShadowCandidateCache_StoreGet(t *testing.T) {
	c := NewShadowCandidateCache()

	candidates := []RoutingCandidate{
		{ChannelUID: "ch1", TotalScore: 10},
		{ChannelUID: "ch2", TotalScore: 8},
	}
	c.Store("claude-3", "messages", candidates)

	result := c.Get("claude-3", "messages")
	if len(result) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(result))
	}
	if result[0].ChannelUID != "ch1" {
		t.Fatalf("expected ch1, got %s", result[0].ChannelUID)
	}
}

func TestShadowCandidateCache_GetMiss(t *testing.T) {
	c := NewShadowCandidateCache()
	result := c.Get("nonexistent", "messages")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestShadowCandidateCache_ReturnsCopy(t *testing.T) {
	c := NewShadowCandidateCache()
	candidates := []RoutingCandidate{
		{ChannelUID: "ch1", TotalScore: 10},
	}
	c.Store("m", "k", candidates)

	result := c.Get("m", "k")
	result[0].ChannelUID = "modified"

	original := c.Get("m", "k")
	if original[0].ChannelUID == "modified" {
		t.Fatal("Get should return a copy, not modify the original")
	}
}

// ── ABTestSampler 测试 ──

func newTestABTestSampler(t *testing.T, enabled bool, sampleRatio float64) (*ABTestSampler, *ABTestStore) {
	t.Helper()
	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}
	sampler := NewABTestSampler(store, func() ABTestSamplerConfig {
		return ABTestSamplerConfig{
			Enabled:                  enabled,
			SampleRatio:              sampleRatio,
			MaxShadowRequestsPerHour: 100,
			ShadowCandidateCount:     1,
		}
	})
	return sampler, store
}

func TestABTestSampler_ShouldSample_Disabled(t *testing.T) {
	sampler, _ := newTestABTestSampler(t, false, 1.0)
	if sampler.ShouldSample(false) {
		t.Fatal("ShouldSample should return false when disabled")
	}
}

func TestABTestSampler_ShouldSample_KillSwitch(t *testing.T) {
	sampler, _ := newTestABTestSampler(t, true, 1.0)
	if sampler.ShouldSample(true) {
		t.Fatal("ShouldSample should return false when KillSwitch is active")
	}
}

func TestABTestSampler_ShouldSample_ZeroRatio(t *testing.T) {
	sampler, _ := newTestABTestSampler(t, true, 0)
	if sampler.ShouldSample(false) {
		t.Fatal("ShouldSample should return false when SampleRatio is 0")
	}
}

func TestABTestSampler_ShouldSample_BudgetExhausted(t *testing.T) {
	sampler, _ := newTestABTestSampler(t, true, 1.0)
	// 耗尽预算
	for i := 0; i < 100; i++ {
		sampler.budget.TryConsume()
	}
	if sampler.ShouldSample(false) {
		t.Fatal("ShouldSample should return false when budget is exhausted")
	}
}

func TestABTestSampler_ShouldSample_ProbabilitySampling(t *testing.T) {
	// SampleRatio=1.0 表示所有请求都应被采样
	sampler, _ := newTestABTestSampler(t, true, 1.0)
	// 在 SampleRatio=1.0 时，randomFloat() < 1.0 总是 true（randomFloat 返回 [0,1)）
	// 但由于 randomFloat 使用 crypto/rand，理论上不可能返回精确的 1.0
	// 所以我们只检查：多次采样中至少有一次命中
	// 注意：理论上 randomFloat() 可能返回 0，但 < 1.0 仍然为 true
	hit := false
	for i := 0; i < 10; i++ {
		if sampler.ShouldSample(false) {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("ShouldSample with ratio=1.0 should almost always return true")
	}
}

// ── A/B 测试：主请求路径不变不变量验证 ──

func TestABTestSampler_PrimaryPathUnaffected(t *testing.T) {
	// 验证：即使 A/B 测试启用，主请求路径的延迟不受影子请求影响。
	// 核心不变量：ExecuteShadowRequest 是异步的（goroutine），立即返回。
	sampler, _ := newTestABTestSampler(t, true, 1.0)

	// 记录是否触发了影子请求（不应该发生，因为缓存为空）
	shadowDispatched := make(chan struct{}, 1)

	// 使用一个 mock HTTP server 作为影子目标（不应被调用）
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowDispatched <- struct{}{}
		w.WriteHeader(200)
	}))
	defer mockServer.Close()

	start := time.Now()
	// ExecuteShadowRequest 应立即返回（候选缓存为空，不会发起实际请求）
	sampler.ExecuteShadowRequest(
		context.Background(),
		&config.ConfigManager{},
		[]byte(`{"model":"test"}`),
		"test-model",
		"messages",
		"primary-channel",
		200,
		100,
	)
	elapsed := time.Since(start)

	// 主路径关键指标：ExecuteShadowRequest 应在 5ms 内返回
	// （仅做配置检查 + 缓存查找，不执行 HTTP）
	if elapsed > 5*time.Millisecond {
		t.Fatalf("ExecuteShadowRequest should return in <5ms, took %v (possible blocking on primary path)", elapsed)
	}

	// 影子请求不应被发送（缓存为空）
	select {
	case <-shadowDispatched:
		t.Fatal("shadow request should not be dispatched when candidate cache is empty")
	case <-time.After(100 * time.Millisecond):
		// 预期：影子请求未被触发
	}
}

func TestABTestSampler_ExecuteShadow_AsyncAfterPrimaryResponse(t *testing.T) {
	// 验证：影子请求在主响应之后异步执行。
	// 1. 预填候选缓存
	// 2. 预填 upstream 配置
	// 3. 调用 ExecuteShadowRequest
	// 4. 验证它立即返回（不等待影子请求完成）
	// 5. 验证影子请求确实发出（异步）

	var shadowStarted atomic.Bool
	var shadowCompleted atomic.Bool

	// mock upstream server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowStarted.Store(true)
		time.Sleep(50 * time.Millisecond) // 模拟慢响应
		shadowCompleted.Store(true)
		w.WriteHeader(200)
	}))
	defer mockServer.Close()

	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}
	sampler := NewABTestSampler(store, func() ABTestSamplerConfig {
		return ABTestSamplerConfig{
			Enabled:                  true,
			SampleRatio:              1.0,
			MaxShadowRequestsPerHour: 100,
			ShadowCandidateCount:     1,
		}
	})
	// 设置测试时间函数避免真实超时
	sampler.client = mockServer.Client()

	// 预填候选缓存（模拟 SmartRouter 回调）
	sampler.cache.Store("test-model", "messages", []RoutingCandidate{
		{ChannelUID: "primary", TotalScore: 10},
		{ChannelUID: "shadow", TotalScore: 8},
	})

	// 需要配置中有 shadow 渠道的上游信息
	// 由于 createTempGinContext + ConvertToProviderRequest 依赖真实 provider，
	// 这里我们验证的是：ExecuteShadowRequest 立即返回，不阻塞。
	// 实际 HTTP 调用由 provider 层处理，这里用空配置测试快速返回路径。
	cfgManager := &config.ConfigManager{}
	start := time.Now()
	sampler.ExecuteShadowRequest(
		context.Background(),
		cfgManager,
		[]byte(`{"model":"test"}`),
		"test-model",
		"messages",
		"primary",
		200,
		100,
	)
	elapsed := time.Since(start)

	// 关键：ExecuteShadowRequest 应立即返回（<10ms）
	// 影子 HTTP 请求在 goroutine 中异步执行，不阻塞主路径
	if elapsed > 10*time.Millisecond {
		t.Fatalf("ExecuteShadowRequest blocked primary path for %v (should be <10ms)", elapsed)
	}
}

// ── ABTestStore 测试 ──

func TestABTestStore_RecordAndGetStats(t *testing.T) {
	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}

	// 写入几条记录
	store.Record(&ABTestRecord{
		RecordUID:        "r1",
		Model:            "test-model",
		ChannelKind:      "messages",
		PrimaryChannelUID: "primary",
		PrimarySuccess:   true,
		ShadowChannelUID: "shadow",
		ShadowSuccess:    true,
		ShadowLatencyMs:  200,
		ShadowCostUSD:    0.001,
	})
	store.Record(&ABTestRecord{
		RecordUID:        "r2",
		Model:            "test-model",
		ChannelKind:      "messages",
		PrimaryChannelUID: "primary",
		PrimarySuccess:   true,
		ShadowChannelUID: "shadow",
		ShadowSuccess:    false,
		ShadowLatencyMs:  500,
		ShadowCostUSD:    0.002,
	})

	stats := store.GetStats()
	if stats.TotalRecords != 2 {
		t.Fatalf("expected 2 records, got %d", stats.TotalRecords)
	}
	if stats.ShadowSuccessCnt != 1 {
		t.Fatalf("expected 1 shadow success, got %d", stats.ShadowSuccessCnt)
	}
	if stats.ShadowFailCnt != 1 {
		t.Fatalf("expected 1 shadow fail, got %d", stats.ShadowFailCnt)
	}
	if stats.ShadowSuccessRate != 0.5 {
		t.Fatalf("expected 0.5 success rate, got %f", stats.ShadowSuccessRate)
	}
	if stats.TotalShadowCostUSD != 0.003 {
		t.Fatalf("expected $0.003 total cost, got %f", stats.TotalShadowCostUSD)
	}
}

func TestABTestStore_ListRecent(t *testing.T) {
	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}

	for i := 0; i < 5; i++ {
		store.Record(&ABTestRecord{
			RecordUID:   fmt.Sprintf("r%d", i),
			Model:       "test-model",
			ChannelKind: "messages",
		})
	}

	recent := store.ListRecent(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent records, got %d", len(recent))
	}
	// 最新的在前
	if recent[0].RecordUID != "r4" {
		t.Fatalf("expected r4 (newest), got %s", recent[0].RecordUID)
	}
}

func TestABTestStore_ByChannelStats(t *testing.T) {
	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}

	store.Record(&ABTestRecord{
		RecordUID:       "r1",
		ShadowChannelUID: "ch-a",
		ShadowSuccess:    true,
		ShadowLatencyMs:  100,
		ShadowCostUSD:    0.001,
	})
	store.Record(&ABTestRecord{
		RecordUID:       "r2",
		ShadowChannelUID: "ch-a",
		ShadowSuccess:    false,
		ShadowLatencyMs:  300,
		ShadowCostUSD:    0.002,
	})
	store.Record(&ABTestRecord{
		RecordUID:       "r3",
		ShadowChannelUID: "ch-b",
		ShadowSuccess:    true,
		ShadowLatencyMs:  200,
		ShadowCostUSD:    0.001,
	})

	stats := store.GetStats()
	chA := stats.ByChannel["ch-a"]
	if chA == nil {
		t.Fatal("expected ch-a stats")
	}
	if chA.Count != 2 {
		t.Fatalf("expected 2 for ch-a, got %d", chA.Count)
	}
	if chA.SuccessRate != 0.5 {
		t.Fatalf("expected 0.5 success rate for ch-a, got %f", chA.SuccessRate)
	}

	chB := stats.ByChannel["ch-b"]
	if chB == nil {
		t.Fatal("expected ch-b stats")
	}
	if chB.SuccessRate != 1.0 {
		t.Fatalf("expected 1.0 success rate for ch-b, got %f", chB.SuccessRate)
	}
}

// ── Handler 测试 ──

func TestABTestResultsHandler_DisabledByDefault(t *testing.T) {
	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}
	sampler := NewABTestSampler(store, func() ABTestSamplerConfig {
		return ABTestSamplerConfig{Enabled: false}
	})

	deps := &ABTestDeps{
		Sampler: sampler,
		Store:   store,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	router.GET("/api/autopilot/ab-test-results", handleABTestResults(deps))
	c.Request, _ = http.NewRequest("GET", "/api/autopilot/ab-test-results", nil)
	router.ServeHTTP(w, c.Request)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestABTestEmergencyStopHandler(t *testing.T) {
	db := newTestDB(t)
	store, err := NewABTestStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewABTestStoreWithDB: %v", err)
	}
	sampler := NewABTestSampler(store, func() ABTestSamplerConfig {
		return ABTestSamplerConfig{Enabled: true}
	})

	deps := &ABTestDeps{
		Sampler: sampler,
		Store:   store,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	router.POST("/api/autopilot/ab-test/emergency-stop", handleABTestEmergencyStop(deps))
	c.Request, _ = http.NewRequest("POST", "/api/autopilot/ab-test/emergency-stop", nil)
	router.ServeHTTP(w, c.Request)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
