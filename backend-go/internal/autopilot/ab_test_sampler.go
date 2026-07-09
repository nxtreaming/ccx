package autopilot

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/providers"

	"github.com/gin-gonic/gin"
)

// ── ABTestSampler: 低比例统计抽样双发 ──

// shadowRequestBudget 每小时影子请求硬预算。
// 仿 L2 探测 ProbeBudget 模式，但按小时而非按天重置。
type shadowRequestBudget struct {
	hourlyLimit int32
	used        atomic.Int32
	resetHour   atomic.Int64 // 当前预算所属的 unix hour（UTC）
	mu          sync.Mutex
	timeFunc    func() time.Time
}

func newShadowRequestBudget(hourlyLimit int) *shadowRequestBudget {
	if hourlyLimit <= 0 {
		hourlyLimit = DefaultABTestMaxShadowPerHour
	}
	b := &shadowRequestBudget{
		hourlyLimit: int32(hourlyLimit),
		timeFunc:    time.Now,
	}
	now := b.timeFunc()
	b.resetHour.Store(now.UTC().Unix() / 3600)
	return b
}

func (b *shadowRequestBudget) maybeReset() {
	now := b.timeFunc().UTC().Unix() / 3600
	if b.resetHour.Load() != now {
		b.mu.Lock()
		if b.resetHour.Load() != now {
			b.used.Store(0)
			b.resetHour.Store(now)
		}
		b.mu.Unlock()
	}
}

// TryConsume 尝试消耗一次影子请求额度。
func (b *shadowRequestBudget) TryConsume() bool {
	b.maybeReset()
	for {
		cur := b.used.Load()
		if cur >= b.hourlyLimit {
			return false
		}
		if b.used.CompareAndSwap(cur, cur+1) {
			return true
		}
	}
}

func (b *shadowRequestBudget) Remaining() int {
	b.maybeReset()
	return int(b.hourlyLimit) - int(b.used.Load())
}

func (b *shadowRequestBudget) Used() int {
	b.maybeReset()
	return int(b.used.Load())
}

// ── 候选缓存（SmartRouter → ABTestSampler 桥接）──

// ShadowCandidateCache 缓存 SmartRouter 最近排名的候选列表。
// key = model:channelKind，value = 排名后的候选。
type ShadowCandidateCache struct {
	mu       sync.RWMutex
	latest   map[string][]RoutingCandidate
}

func NewShadowCandidateCache() *ShadowCandidateCache {
	return &ShadowCandidateCache{
		latest: make(map[string][]RoutingCandidate),
	}
}

// Store 存储给定 model+kind 的排名候选。
func (c *ShadowCandidateCache) Store(model, channelKind string, candidates []RoutingCandidate) {
	key := model + ":" + channelKind
	c.mu.Lock()
	c.latest[key] = candidates
	c.mu.Unlock()
}

// Get 获取给定 model+kind 的排名候选（返回副本）。
func (c *ShadowCandidateCache) Get(model, channelKind string) []RoutingCandidate {
	key := model + ":" + channelKind
	c.mu.RLock()
	candidates := c.latest[key]
	c.mu.RUnlock()
	if len(candidates) == 0 {
		return nil
	}
	result := make([]RoutingCandidate, len(candidates))
	copy(result, candidates)
	return result
}

// ── ABTestSampler ──

// ABTestSamplerConfig A/B 测试采样器配置。
type ABTestSamplerConfig struct {
	Enabled                  bool
	SampleRatio              float64
	MaxShadowRequestsPerHour int
	ShadowCandidateCount     int
}

// DefaultABTestMaxShadowPerHour 默认每小时影子请求上限。
const DefaultABTestMaxShadowPerHour = 60

// ABTestSampler 低比例统计抽样双发采样器。
// 主请求路径完全不变：影子请求在主响应返回后异步发起，不影响主请求延迟或结果。
type ABTestSampler struct {
	store   *ABTestStore
	cache   *ShadowCandidateCache
	budget  *shadowRequestBudget
	config  func() ABTestSamplerConfig // 动态配置读取
	client  *http.Client
	timeFn  func() time.Time
}

// NewABTestSampler 创建 ABTestSampler。
func NewABTestSampler(store *ABTestStore, configFn func() ABTestSamplerConfig) *ABTestSampler {
	cfg := configFn()
	return &ABTestSampler{
		store:  store,
		cache:  NewShadowCandidateCache(),
		budget: newShadowRequestBudget(cfg.MaxShadowRequestsPerHour),
		config: configFn,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		timeFn: time.Now,
	}
}

// CandidateCache 返回候选缓存，供 SmartRouter 回调写入。
func (s *ABTestSampler) CandidateCache() *ShadowCandidateCache {
	return s.cache
}

// OnCandidatesRanked 返回 SmartRouter 的回调函数，写入候选缓存。
func (s *ABTestSampler) OnCandidatesRanked() func(model, channelKind string, candidates []RoutingCandidate) {
	return s.cache.Store
}

// ShouldSample 判断本次请求是否应触发影子请求。
// 安全检查链：Enabled → KillSwitch → SampleRatio → Budget。
func (s *ABTestSampler) ShouldSample(killSwitch bool) bool {
	cfg := s.config()
	if !cfg.Enabled {
		return false
	}
	// 全局 KillSwitch 触发时 A/B 采样立即停止
	if killSwitch {
		return false
	}
	if cfg.SampleRatio <= 0 {
		return false
	}
	if !s.budget.TryConsume() {
		return false
	}
	// 概率采样
	return randomFloat() < cfg.SampleRatio
}

// ExecuteShadowRequest 异步执行影子请求。
// 必须在主响应已返回给客户端之后调用。
// cfgManager 用于获取配置和 API key。
// primaryChannelUID 是主请求实际使用的渠道。
// primaryStatusCode 和 primaryLatencyMs 是主请求的结果。
// gin.Context 不需要：影子请求使用独立 context 和临时 gin.Context。
func (s *ABTestSampler) ExecuteShadowRequest(
	parentCtx context.Context,
	cfgManager *config.ConfigManager,
	bodyBytes []byte,
	model string,
	channelKind string,
	primaryChannelUID string,
	primaryStatusCode int,
	primaryLatencyMs int64,
) {
	cfg := s.config()
	if !cfg.Enabled {
		return
	}

	// 从候选缓存获取排名候选
	candidates := s.cache.Get(model, channelKind)
	if len(candidates) < 2 {
		return
	}

	// 选取 shadow 候选：排除主渠道，取排名最高的 N 个
	shadowCount := cfg.ShadowCandidateCount
	if shadowCount <= 0 {
		shadowCount = 1
	}
	shadowCandidates := make([]RoutingCandidate, 0, shadowCount)
	for _, cand := range candidates {
		if cand.ChannelUID == primaryChannelUID {
			continue
		}
		shadowCandidates = append(shadowCandidates, cand)
		if len(shadowCandidates) >= shadowCount {
			break
		}
	}
	if len(shadowCandidates) == 0 {
		return
	}

	shadowCand := shadowCandidates[0]

	// 获取 shadow 渠道的上游配置
	cfg_snapshot := cfgManager.GetConfig()
	shadowUpstream := findUpstreamByUID(&cfg_snapshot, shadowCand.ChannelUID, shadowCand.ChannelKind)
	if shadowUpstream == nil {
		return
	}

	// 复制 bodyBytes 防止异步使用时被回收
	bodyCopy := make([]byte, len(bodyBytes))
	copy(bodyCopy, bodyBytes)

	// 异步执行影子请求（不等待，不阻塞主请求）
	go func() {
		// 独立 context：主请求的 gin.Context 生命周期已结束
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		record := s.executeShadowHTTPRequest(ctx, cfgManager, shadowUpstream, shadowCand, bodyCopy, model, channelKind,
			primaryChannelUID, primaryStatusCode, primaryLatencyMs)

		if record != nil {
			s.store.Record(record)
			if s.config().Enabled {
				log.Printf("[ABTest-Sampler] 影子请求完成: shadow=%s success=%v latency=%dms cost=$%.6f",
					record.ShadowChannelUID, record.ShadowSuccess, record.ShadowLatencyMs, record.ShadowCostUSD)
			}
		}
	}()
}

// executeShadowHTTPRequest 执行单个影子 HTTP 请求并返回记录。
func (s *ABTestSampler) executeShadowHTTPRequest(
	ctx context.Context,
	cfgManager *config.ConfigManager,
	shadowUpstream *config.UpstreamConfig,
	shadowCand RoutingCandidate,
	bodyBytes []byte,
	model string,
	channelKind string,
	primaryChannelUID string,
	primaryStatusCode int,
	primaryLatencyMs int64,
) *ABTestRecord {
	// 获取 API key
	apiKey, err := cfgManager.GetNextAPIKey(shadowUpstream, nil, shadowCand.ChannelKind)
	if err != nil || apiKey == "" {
		return nil
	}

	// 构建临时 gin.Context 用于 provider 转换
	tempCtx := createTempGinContext(bodyBytes)

	// 获取 provider 并构建请求
	provider := providers.GetProvider(shadowUpstream.ServiceType)
	if provider == nil {
		return nil
	}

	httpReq, _, err := provider.ConvertToProviderRequest(tempCtx, shadowUpstream, apiKey)
	if err != nil {
		return nil
	}

	// 设置 context
	httpReq = httpReq.WithContext(ctx)

	// 执行影子请求
	startTime := s.timeFn()
	resp, err := s.client.Do(httpReq)
	latencyMs := s.timeFn().Sub(startTime).Milliseconds()

	record := &ABTestRecord{
		RecordUID:         generateABTestUID(),
		Model:             model,
		ChannelKind:       channelKind,
		PrimaryChannelUID: primaryChannelUID,
		PrimaryStatusCode: primaryStatusCode,
		PrimaryLatencyMs:  primaryLatencyMs,
		PrimarySuccess:    primaryStatusCode >= 200 && primaryStatusCode < 400,
		ShadowChannelUID:  shadowCand.ChannelUID,
		ShadowLatencyMs:   latencyMs,
		TraceUID:          "",
	}

	if err != nil {
		record.ShadowSuccess = false
		record.ShadowError = err.Error()
		return record
	}
	defer resp.Body.Close()
	// 消费并丢弃 body（避免连接泄漏）
	io.Copy(io.Discard, resp.Body)

	record.ShadowStatusCode = resp.StatusCode
	record.ShadowSuccess = resp.StatusCode >= 200 && resp.StatusCode < 400
	return record
}

// BudgetRemaining 返回当前小时剩余影子请求额度。
func (s *ABTestSampler) BudgetRemaining() int {
	return s.budget.Remaining()
}

// BudgetUsed 返回当前小时已用影子请求额度。
func (s *ABTestSampler) BudgetUsed() int {
	return s.budget.Used()
}

// ── 辅助函数 ──

// findUpstreamByUID 从配置中查找指定 channelUID 和 kind 的上游配置。
func findUpstreamByUID(cfg *config.Config, channelUID string, channelKind string) *config.UpstreamConfig {
	for i := range cfg.Upstream {
		up := &cfg.Upstream[i]
		if up.ChannelUID == channelUID {
			return up
		}
	}
	// 未匹配到 ChannelUID 时，尝试按名称匹配（兼容旧配置无 ChannelUID 的场景）
	for i := range cfg.Upstream {
		up := &cfg.Upstream[i]
		if up.Name == channelUID {
			return up
		}
	}
	return nil
}

// createTempGinContext 创建一个临时 gin.Context 用于 provider 请求转换。
// 只设置最小必要字段：request body 和 content-type。
func createTempGinContext(bodyBytes []byte) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

func randomFloat() float64 {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return 0
	}
	return float64(uint64(b[0])<<56|uint64(b[1])<<48|uint64(b[2])<<40|uint64(b[3])<<32|
		uint64(b[4])<<24|uint64(b[5])<<16|uint64(b[6])<<8|uint64(b[7])) / math.MaxUint64
}

func generateABTestUID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("ab_%x_%d", b, time.Now().UnixNano())
}
