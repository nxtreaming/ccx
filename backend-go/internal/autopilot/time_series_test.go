package autopilot

import (
	"fmt"
	"testing"
	"time"
)

// TestAlignToBucket 测试 15 分钟桶对齐。
func TestAlignToBucket(t *testing.T) {
	parse := func(s string) time.Time {
		t.Helper()
		tt, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("时间解析失败: %v", err)
		}
		return tt
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "整点对齐",
			input:    "2025-07-08T09:00:00Z",
			expected: "2025-07-08T09:00:00Z",
		},
		{
			name:     "向下取整到整15分",
			input:    "2025-07-08T09:22:30Z",
			expected: "2025-07-08T09:15:00Z",
		},
		{
			name:     "正好在桶边界上",
			input:    "2025-07-08T09:15:00Z",
			expected: "2025-07-08T09:15:00Z",
		},
		{
			name:     "桶边界前1秒",
			input:    "2025-07-08T09:14:59Z",
			expected: "2025-07-08T09:00:00Z",
		},
		{
			name:     "第二个15分钟桶",
			input:    "2025-07-08T09:29:59Z",
			expected: "2025-07-08T09:15:00Z",
		},
		{
			name:     "第四个15分钟桶",
			input:    "2025-07-08T09:59:59Z",
			expected: "2025-07-08T09:45:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := parse(tt.input)
			expected := parse(tt.expected)
			got := alignToBucket(input)
			if !got.Equal(expected) {
				t.Errorf("alignToBucket(%v) = %v, want %v", input, got, expected)
			}
		})
	}
}

// TestRecordAndMerge 测试同一桶内多次 Record 正确累积。
func TestRecordAndMerge(t *testing.T) {
	store := NewTimeBucketStore()
	endpointUID := "abc123def456"

	// 记录 3 次请求（在同一个桶内）
	store.Record(endpointUID, 1, "metrics-key", true, 100)
	store.Record(endpointUID, 1, "metrics-key", true, 200)
	store.Record(endpointUID, 1, "metrics-key", false, 300)

	buckets := store.GetBuckets(endpointUID, 10)
	if len(buckets) != 1 {
		t.Fatalf("期望 1 个桶, 得到 %d", len(buckets))
	}

	b := buckets[0]
	if b.RequestCount != 3 {
		t.Errorf("RequestCount = %d, 期望 3", b.RequestCount)
	}
	if b.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, 期望 2", b.SuccessCount)
	}
	if b.FailureCount != 1 {
		t.Errorf("FailureCount = %d, 期望 1", b.FailureCount)
	}
	// 成功率 = 2/3 ≈ 0.6667
	if b.SuccessRate < 0.666 || b.SuccessRate > 0.667 {
		t.Errorf("SuccessRate = %f, 期望 ≈ 0.667", b.SuccessRate)
	}
}

// TestMultiBucketRecording 测试不同桶的自动创建。
func TestMultiBucketRecording(t *testing.T) {
	store := NewTimeBucketStore()
	endpointUID := "ep001"

	// 手动构造不同桶的记录
	// 使用 alignToBucket 确保精确对齐
	now := time.Now().UTC()
	currentBucket := alignToBucket(now)
	prevBucket := currentBucket.Add(-bucketSize)

	// 直接操作内部 ring 来模拟不同时间的桶（Record 使用 time.Now()）
	ring := newRingBuffer(maxBuckets)
	ring.Append(&TimeBucketMetrics{
		BucketStart:  prevBucket,
		BucketSize:   bucketSize,
		RequestCount: 5,
		SuccessCount: 4,
		SuccessRate:  0.8,
	})
	ring.Append(&TimeBucketMetrics{
		BucketStart:  currentBucket,
		BucketSize:   bucketSize,
		RequestCount: 3,
		SuccessCount: 3,
		SuccessRate:  1.0,
	})

	store.rings[endpointUID] = ring

	// 取最近 2 个桶
	buckets := store.GetBuckets(endpointUID, 2)
	if len(buckets) != 2 {
		t.Fatalf("期望 2 个桶, 得到 %d", len(buckets))
	}

	// 时间正序
	if !buckets[0].BucketStart.Equal(prevBucket) {
		t.Errorf("第一个桶 BucketStart = %v, 期望 %v", buckets[0].BucketStart, prevBucket)
	}
	if !buckets[1].BucketStart.Equal(currentBucket) {
		t.Errorf("第二个桶 BucketStart = %v, 期望 %v", buckets[1].BucketStart, currentBucket)
	}
	if buckets[0].RequestCount != 5 {
		t.Errorf("第一个桶 RequestCount = %d, 期望 5", buckets[0].RequestCount)
	}
}

// TestRingBufferOverflow 测试环形缓冲区溢出时覆盖最旧桶。
func TestRingBufferOverflow(t *testing.T) {
	// 小容量便于测试
	ring := newRingBuffer(3)

	// 填充 3 个桶（正好满）
	for i := 0; i < 3; i++ {
		bucketStart := time.Date(2025, 7, 1, 0, i*15, 0, 0, time.UTC)
		ring.Append(&TimeBucketMetrics{
			BucketStart:  bucketStart,
			RequestCount: i + 1,
		})
	}
	if ring.Len() != 3 {
		t.Fatalf("期望长度 3, 得到 %d", ring.Len())
	}

	// 追加第 4 个桶，覆盖最旧的（0:00 的桶）
	newBucketStart := time.Date(2025, 7, 1, 0, 45, 0, 0, time.UTC)
	ring.Append(&TimeBucketMetrics{
		BucketStart:  newBucketStart,
		RequestCount: 99,
	})
	if ring.Len() != 3 {
		t.Fatalf("溢出后期望长度仍为 3, 得到 %d", ring.Len())
	}

	// GetLastN(3) 应返回最新的 3 个
	last3 := ring.GetLastN(3)
	if len(last3) != 3 {
		t.Fatalf("GetLastN(3) 期望 3 个, 得到 %d", len(last3))
	}
	// 最旧的应是 0:15 的桶（0:00 的已被覆盖）
	if last3[0].RequestCount != 2 {
		t.Errorf("最旧桶 RequestCount = %d, 期望 2 (0:15)", last3[0].RequestCount)
	}
	// 最新的应是 0:45 的桶
	if last3[2].RequestCount != 99 {
		t.Errorf("最新桶 RequestCount = %d, 期望 99 (0:45)", last3[2].RequestCount)
	}
}

// TestGetBucketsPartial 测试取数不足 n 时返回实际数量。
func TestGetBucketsPartial(t *testing.T) {
	store := NewTimeBucketStore()
	endpointUID := "ep-partial"

	store.Record(endpointUID, 1, "mk", true, 50)

	buckets := store.GetBuckets(endpointUID, 100)
	if len(buckets) != 1 {
		t.Fatalf("期望 1 个桶（不足 n=100）, 得到 %d", len(buckets))
	}
}

// TestGetBucketsEmpty 测试无数据时返回 nil。
func TestGetBucketsEmpty(t *testing.T) {
	store := NewTimeBucketStore()
	buckets := store.GetBuckets("nonexistent", 10)
	if buckets != nil {
		t.Errorf("无数据时期望 nil, 得到 %v", buckets)
	}
}

// TestMultipleEndpoints 测试不同 endpointUID 互不干扰。
func TestMultipleEndpoints(t *testing.T) {
	store := NewTimeBucketStore()

	store.Record("ep-A", 1, "mk-a", true, 100)
	store.Record("ep-B", 2, "mk-b", false, 500)

	bucketsA := store.GetBuckets("ep-A", 10)
	bucketsB := store.GetBuckets("ep-B", 10)

	if len(bucketsA) != 1 || len(bucketsB) != 1 {
		t.Fatalf("每个 endpoint 应有 1 个桶, A=%d, B=%d", len(bucketsA), len(bucketsB))
	}
	if bucketsA[0].SuccessCount != 1 {
		t.Errorf("ep-A SuccessCount = %d, 期望 1", bucketsA[0].SuccessCount)
	}
	if bucketsB[0].FailureCount != 1 {
		t.Errorf("ep-B FailureCount = %d, 期望 1", bucketsB[0].FailureCount)
	}
}

// TestPercentileInt64 测试百分位计算。
func TestPercentileInt64(t *testing.T) {
	tests := []struct {
		name     string
		data     []int64
		p        int
		expected int64
	}{
		{
			name:     "空切片",
			data:     nil,
			p:        95,
			expected: 0,
		},
		{
			name:     "单元素",
			data:     []int64{100},
			p:        50,
			expected: 100,
		},
		{
			name:     "两元素取中位数",
			data:     []int64{100, 200},
			p:        50,
			expected: 150,
		},
		{
			name:     "901个元素P95",
			data:     makeRange(100, 1000), // 100, 101, ..., 1000 → 901 个元素
			p:        95,
			expected: 955, // rank=95/100*(901-1)=855.0 → data[855]=955, frac=0.0 → 955
		},
		{
			name:     "20个元素P95",
			data:     makeRange(1, 20),
			p:        95,
			expected: 19, // rank=18.05, data[18]=19, data[19]=20, frac=0.05 → 19*0.95+20*0.05=19.05→19
		},
		{
			name:     "P0取最小值",
			data:     []int64{10, 20, 30, 40, 50},
			p:        0,
			expected: 10,
		},
		{
			name:     "P100取最大值",
			data:     []int64{10, 20, 30, 40, 50},
			p:        100,
			expected: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 传入副本，避免排序影响
			data := make([]int64, len(tt.data))
			copy(data, tt.data)
			got := PercentileInt64(data, tt.p)
			if got != tt.expected {
				t.Errorf("PercentileInt64(%v, %d) = %d, 期望 %d", tt.data, tt.p, got, tt.expected)
			}
		})
	}
}

// makeRange 生成 [start, end] 的 int64 序列。
func makeRange(start, end int) []int64 {
	result := make([]int64, end-start+1)
	for i := range result {
		result[i] = int64(start + i)
	}
	return result
}

// TestRingBufferFindByStart 测试环形缓冲区二分查找。
func TestRingBufferFindByStart(t *testing.T) {
	ring := newRingBuffer(10)
	base := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	// 添加 5 个桶：0:00, 0:15, 0:30, 0:45, 1:00
	for i := 0; i < 5; i++ {
		ring.Append(&TimeBucketMetrics{
			BucketStart:  base.Add(time.Duration(i) * bucketSize),
			RequestCount: i + 1,
		})
	}

	// 查找存在的桶
	target := base.Add(30 * time.Minute) // 0:30
	found := ring.findByStart(target)
	if found == nil {
		t.Fatal("findByStart(0:30) 返回 nil")
	}
	if found.RequestCount != 3 {
		t.Errorf("findByStart(0:30).RequestCount = %d, 期望 3", found.RequestCount)
	}

	// 查找不存在的桶
	notFound := ring.findByStart(base.Add(7 * time.Minute))
	if notFound != nil {
		t.Errorf("findByStart(0:07) 时期望 nil, 得到 %v", notFound)
	}
}

// TestRingBufferWrappedFind 测试环绕后二分查找仍然正确。
func TestRingBufferWrappedFind(t *testing.T) {
	ring := newRingBuffer(3)
	base := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	// 填满 3 个桶再追加 1 个（覆盖第 0 个）
	for i := 0; i < 4; i++ {
		ring.Append(&TimeBucketMetrics{
			BucketStart:  base.Add(time.Duration(i) * bucketSize),
			RequestCount: i + 1,
		})
	}

	// 当前逻辑顺序：0:15(req=2), 0:30(req=3), 0:45(req=4)
	// 0:00 的桶已被覆盖

	found := ring.findByStart(base.Add(15 * time.Minute))
	if found == nil {
		t.Fatal("环绕后 findByStart(0:15) 返回 nil")
	}
	if found.RequestCount != 2 {
		t.Errorf("RequestCount = %d, 期望 2", found.RequestCount)
	}

	// 被覆盖的桶不应能查到
	overwritten := ring.findByStart(base)
	if overwritten != nil {
		t.Errorf("被覆盖的桶 0:00 不应查到, 得到 %v", overwritten)
	}
}

// TestRecordLatency 迋试延迟记录（取最大值作为 P95 近似）。
func TestRecordLatency(t *testing.T) {
	store := NewTimeBucketStore()
	uid := "ep-latency"

	store.Record(uid, 1, "mk", true, 50)
	store.Record(uid, 1, "mk", true, 200)
	store.Record(uid, 1, "mk", true, 150)

	buckets := store.GetBuckets(uid, 1)
	if len(buckets) != 1 {
		t.Fatalf("期望 1 个桶, 得到 %d", len(buckets))
	}
	// P95 近似取最大值
	if buckets[0].P95LatencyMs != 200 {
		t.Errorf("P95LatencyMs = %d, 期望 200", buckets[0].P95LatencyMs)
	}
}

// TestStoreDays 测试存储容量常量正确。
func TestStoreDays(t *testing.T) {
	if maxBuckets != 672 {
		t.Errorf("maxBuckets = %d, 期望 672 (7天×24小时×4桶)", maxBuckets)
	}
}

// BenchmarkRecord 性能基准。
func BenchmarkRecord(b *testing.B) {
	store := NewTimeBucketStore()
	uid := "bench-ep"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Record(uid, 1, fmt.Sprintf("mk-%d", i%10), i%3 != 0, int64(i%1000))
	}
}
