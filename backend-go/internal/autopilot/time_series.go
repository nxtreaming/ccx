package autopilot

import (
	"sort"
	"sync"
	"time"
)

// 时间桶粒度：Phase 1 只做 15 分钟主粒度。
// 设计文档提到多粒度（15 分钟 / 1 小时），Phase 1 仅实现 15 分钟桶。
const bucketSize = 15 * time.Minute

// 存储天数：7 天滚动。
const storeDays = 7

// 最大桶数 = 7 天 × 24 小时 × 4 桶/小时。
const maxBuckets = storeDays * 24 * 4 // 672

// TimeBucketMetrics 按固定时间桶记录 endpoint 的质量快照。
type TimeBucketMetrics struct {
	ChannelID   int           `json:"channelId"`
	MetricsKey  string        `json:"metricsKey"`
	BucketStart time.Time     `json:"bucketStart"` // 桶起始时间（UTC，15 分钟对齐）
	BucketSize  time.Duration `json:"bucketSize"`  // 桶大小，默认 15 分钟

	// ── 该桶内的聚合指标 ──
	RequestCount       int `json:"requestCount"`
	SuccessCount       int `json:"successCount"`
	FailureCount       int `json:"failureCount"`
	OverloadedCount    int `json:"overloadedCount"` // 429
	StreamBreakCount   int `json:"streamBreakCount"`
	EmptyResponseCount int `json:"emptyResponseCount"`

	P50LatencyMs int64 `json:"p50LatencyMs"`
	P95LatencyMs int64 `json:"p95LatencyMs"`
	P99LatencyMs int64 `json:"p99LatencyMs"`

	SuccessRate     float64 `json:"successRate"`
	AvgInputTokens  int     `json:"avgInputTokens"`
	AvgOutputTokens int     `json:"avgOutputTokens"`
}

// ringBuffer 固定大小环形桶缓冲区，按 BucketStart 排序。
type ringBuffer struct {
	buckets  []*TimeBucketMetrics
	writePos int  // 下一次写入位置
	capacity int  // 最大容量
	wrapped  bool // 是否已环绕至少一次
}

// newRingBuffer 创建固定容量的环形缓冲区。
func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		buckets:  make([]*TimeBucketMetrics, capacity),
		capacity: capacity,
	}
}

// Len 返回当前有效桶数量。
func (r *ringBuffer) Len() int {
	if r.wrapped {
		return r.capacity
	}
	return r.writePos
}

// Append 追加桶到环形缓冲区。超过容量时覆盖最旧桶。
func (r *ringBuffer) Append(b *TimeBucketMetrics) {
	r.buckets[r.writePos] = b
	r.writePos++
	if r.writePos >= r.capacity {
		r.writePos = 0
		r.wrapped = true
	}
}

// GetLastN 按时间正序返回最近 n 个桶（n <= Len）。
func (r *ringBuffer) GetLastN(n int) []*TimeBucketMetrics {
	total := r.Len()
	if n > total {
		n = total
	}
	if n <= 0 {
		return nil
	}

	result := make([]*TimeBucketMetrics, n)
	if !r.wrapped {
		// 未环绕：直接取 [writePos-n, writePos)
		copy(result, r.buckets[r.writePos-n:r.writePos])
	} else {
		// 已环绕：从 writePos 往回取 n 个
		start := r.writePos - n
		if start >= 0 {
			copy(result, r.buckets[start:r.writePos])
		} else {
			// 跨越环形边界
			tail := r.buckets[r.capacity+start:]
			copy(result, tail)
			copy(result[len(tail):], r.buckets[:r.writePos])
		}
	}
	return result
}

// findByStart 在环形缓冲区中二分查找 BucketStart 匹配的桶。
// 需要缓冲区内桶按 BucketStart 有序。
func (r *ringBuffer) findByStart(target time.Time) *TimeBucketMetrics {
	n := r.Len()
	if n == 0 {
		return nil
	}

	// 二分查找
	lo, hi := 0, n-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		b := r.at(mid)
		if b.BucketStart.Equal(target) {
			return b
		}
		if b.BucketStart.Before(target) {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return nil
}

// at 返回逻辑索引 i 处的桶（0 = 最旧）。
func (r *ringBuffer) at(i int) *TimeBucketMetrics {
	if !r.wrapped {
		return r.buckets[i]
	}
	idx := (r.writePos + i) % r.capacity
	return r.buckets[idx]
}

// TimeBucketStore 内存中按 endpointUID 管理环形桶。
// Phase 1：纯内存，不持久化；Future：可对齐 SQLite（设计 §3.10 存储）。
type TimeBucketStore struct {
	rings map[string]*ringBuffer
	mu    sync.RWMutex
}

// NewTimeBucketStore 创建 TimeBucketStore。
func NewTimeBucketStore() *TimeBucketStore {
	return &TimeBucketStore{
		rings: make(map[string]*ringBuffer),
	}
}

// Record 记录一次请求结果到当前 15 分钟桶。
// endpointUID 是稳定标识（GenerateEndpointUID 生成）。
// latencyMs 为请求延迟毫秒数。
func (s *TimeBucketStore) Record(
	endpointUID string,
	channelID int,
	metricsKey string,
	success bool,
	latencyMs int64,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	bucketStart := alignToBucket(now)

	ring := s.rings[endpointUID]
	if ring == nil {
		ring = newRingBuffer(maxBuckets)
		s.rings[endpointUID] = ring
	}

	// 查找当前桶
	b := ring.findByStart(bucketStart)
	if b == nil {
		b = &TimeBucketMetrics{
			ChannelID:   channelID,
			MetricsKey:  metricsKey,
			BucketStart: bucketStart,
			BucketSize:  bucketSize,
		}
		ring.Append(b)
	}

	// 累积写入
	b.RequestCount++
	if success {
		b.SuccessCount++
	} else {
		b.FailureCount++
	}
	if b.RequestCount > 0 {
		b.SuccessRate = float64(b.SuccessCount) / float64(b.RequestCount)
	}

	// 延迟：Phase 1 简化为 P95 近似（取最大值作为上界快照）
	if latencyMs > b.P95LatencyMs {
		b.P95LatencyMs = latencyMs
	}
	// P50/P99 Phase 1 暂用同值近似，Future 可引入 t-digest
	if latencyMs > b.P50LatencyMs {
		b.P50LatencyMs = latencyMs
	}
	if latencyMs > b.P99LatencyMs {
		b.P99LatencyMs = latencyMs
	}
}

// GetBuckets 返回 endpointUID 最近 n 个桶（时间正序）。
// 桶数量不足 n 时返回实际数量。
func (s *TimeBucketStore) GetBuckets(endpointUID string, n int) []*TimeBucketMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ring := s.rings[endpointUID]
	if ring == nil {
		return nil
	}
	return ring.GetLastN(n)
}

// alignToBucket 将时间对齐到 15 分钟桶的起始时间（向下取整）。
func alignToBucket(t time.Time) time.Time {
	t = t.UTC()
	minutes := t.Minute() / 15 * 15
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minutes, 0, 0, time.UTC)
}

// PercentileInt64 计算 int64 切片的百分位数（线性插值）。
// p 取值 0-100。切片会被排序。
func PercentileInt64(data []int64, p int) int64 {
	if len(data) == 0 || p < 0 || p > 100 {
		return 0
	}
	sort.Slice(data, func(i, j int) bool { return data[i] < data[j] })

	if len(data) == 1 {
		return data[0]
	}
	rank := float64(p) / 100.0 * float64(len(data)-1)
	lower := int(rank)
	upper := lower + 1
	if upper >= len(data) {
		return data[len(data)-1]
	}
	frac := rank - float64(lower)
	return int64(float64(data[lower])*(1-frac) + float64(data[upper])*frac)
}
