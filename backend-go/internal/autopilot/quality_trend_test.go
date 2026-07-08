package autopilot

import (
	"testing"
	"time"
)

// buildTestStore 构造测试用的 TimeBucketStore。
// 用 givenBuckets 直接注入（绕过 Record 的 time.Now() 依赖）。
func buildTestStore(endpointUID string, buckets []*TimeBucketMetrics) *TimeBucketStore {
	store := NewTimeBucketStore()
	ring := newRingBuffer(maxBuckets)
	for _, b := range buckets {
		ring.Append(b)
	}
	store.rings[endpointUID] = ring
	return store
}

// makeBuckets 生成时间连续的桶序列。
// start 为起始时间，count 为桶数量，每个桶间隔 bucketSize。
// successRate 为 0-100 范围的百分比。
func makeBuckets(start time.Time, count int, successRate float64, p95Latency int64) []*TimeBucketMetrics {
	buckets := make([]*TimeBucketMetrics, count)
	reqCount := 20
	successCount := int(float64(reqCount) * successRate / 100)
	for i := 0; i < count; i++ {
		buckets[i] = &TimeBucketMetrics{
			BucketStart:  start.Add(time.Duration(i) * bucketSize),
			BucketSize:   bucketSize,
			RequestCount: reqCount,
			SuccessCount: successCount,
			FailureCount: reqCount - successCount,
			SuccessRate:  successRate / 100,
			P95LatencyMs: p95Latency,
		}
	}
	return buckets
}

// TestDegradingSuccessRate 测试成功率突降场景。
func TestDegradingSuccessRate(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 88, 200)
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 88, 200)
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 60, 200)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-degrade", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-degrade", "mk-degrade", now)

	if trend.Direction != TrendDegrading {
		t.Errorf("Direction = %v, 期望 degrading", trend.Direction)
	}
	if trend.DeltaPercent >= 0 {
		t.Errorf("DeltaPercent = %f, 期望 < 0", trend.DeltaPercent)
	}
	// 成功率从 88% 降至 60%，满足 < baseline7d * 0.75 (66%)，触发重评估
	if !trend.ShouldReevaluate {
		t.Error("ShouldReevaluate = false, 期望 true (成功率突降超过 25%)")
	}
}

// TestImproving 测试成功率改善场景。
func TestImproving(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 80, 200)
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 80, 200)
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 100, 200)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-improve", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-improve", "mk-improve", now)

	if trend.Direction != TrendImproving {
		t.Errorf("Direction = %v, 期望 improving", trend.Direction)
	}
}

// TestVolatile 测试忙闲交替剧烈场景。
// 统一20请求/桶，偶数源小时100%，奇数源小时10%。
// computeHourlyStdDev 的 h-value 混合后交替77.5%/32.5%，stddev≈22.7>20。
// baseline24h≈57%，current1h(src23)=60%：60在[42,67]窗口内，不触发improving/degrading→volatile。
func TestVolatile(t *testing.T) {
	T := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)

	// 7d 基线：576 桶，80%
	buckets7d := makeBuckets(T.Add(-7*24*time.Hour), 576, 80, 200)

	var buckets24h []*TimeBucketMetrics
	for srcHour := 0; srcHour < 24; srcHour++ {
		hourStart := T.Add(-24 * time.Hour).Add(time.Duration(srcHour) * time.Hour)
		var rate float64
		switch {
		case srcHour == 23:
			rate = 60 // 落入 current1h 窗口，在 baseline24h 的 [-15,+10] 范围内
		case srcHour%2 == 0:
			rate = 100
		default:
			rate = 10
		}
		buckets24h = append(buckets24h, makeBuckets(hourStart, 4, rate, 200)...)
	}

	all := append(buckets7d, buckets24h...)

	store := buildTestStore("ep-vol", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-vol", "mk-vol", T)

	if trend.Direction != TrendVolatile {
		t.Errorf("Direction = %v, 期望 volatile", trend.Direction)
	}
}

// TestStable 测试稳定场景。
func TestStable(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 95, 200)
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 95, 200)
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 95, 200)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-stable", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-stable", "mk-stable", now)

	if trend.Direction != TrendStable {
		t.Errorf("Direction = %v, 期望 stable", trend.Direction)
	}
	if trend.ShouldReevaluate {
		t.Error("ShouldReevaluate = true, 期望 false（稳定场景）")
	}
}

// TestInsufficientData 测试数据不足时的默认行为。
func TestInsufficientData(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)

	// 仅 2 个桶（不足计算趋势）
	buckets := makeBuckets(now.Add(-30*time.Minute), 2, 80, 200)

	store := buildTestStore("ep-sparse", buckets)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-sparse", "mk-sparse", now)

	if trend.Direction != TrendStable {
		t.Errorf("Direction = %v, 期望 stable（数据不足）", trend.Direction)
	}
}

// TestNoData 测试完全无数据的场景。
func TestNoData(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	store := NewTimeBucketStore()
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("nonexistent", "mk-empty", now)

	if trend.Direction != TrendStable {
		t.Errorf("Direction = %v, 期望 stable（无数据）", trend.Direction)
	}
	if trend.MetricsKey != "mk-empty" {
		t.Errorf("MetricsKey = %v, 期望 mk-empty", trend.MetricsKey)
	}
}

// TestReevaluateLatencySpike 测试延迟飙升触发重评估。
func TestReevaluateLatencySpike(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 95, 200)
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 95, 200)
	// 当前 1h 延迟 500ms（> 200ms * 2 = 400ms，触发重评估）
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 95, 500)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-latency", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-latency", "mk-latency", now)

	if !trend.ShouldReevaluate {
		t.Error("ShouldReevaluate = false, 期望 true（延迟飙升超过 2 倍）")
	}
	if trend.ReevalReason == "" {
		t.Error("ReevalReason 为空, 期望包含延迟信息")
	}
}

// TestPeakHours 测试质量低谷时段检测。
// 仅使用24h数据（无7d基线），避免 buildHourlyPattern 被7d数据稀释。
// 统一20请求/桶，偶数源小时100%，奇数源小时10%，当前小时(src23)60%。
// buildHourlyPattern：偶数UTC小时≈100%，奇数UTC小时≈10%（全部<70%阈值）。
// baseline24h≈57%，current1h=60%：60在[42,67]窗口内→不触发improving/degrading。
// hourly stddev≈32.8>20→volatile。
func TestPeakHours(t *testing.T) {
	T := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)

	// 不含7d基线，避免 buildHourlyPattern 被基线数据稀释

	var buckets24h []*TimeBucketMetrics
	for srcHour := 0; srcHour < 24; srcHour++ {
		hourStart := T.Add(-24 * time.Hour).Add(time.Duration(srcHour) * time.Hour)
		var rate float64
		switch {
		case srcHour == 23:
			// UTC 11：当前小时，60% 避免触发 improving
			rate = 60
		case srcHour%2 == 0:
			// 偶数源小时（非低谷）：100%
			rate = 100
		default:
			// 奇数源小时（低谷时段）：10%（远低于70%阈值）
			rate = 10
		}
		buckets24h = append(buckets24h, makeBuckets(hourStart, 4, rate, 200)...)
	}

	store := buildTestStore("ep-peak", buckets24h)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-peak", "mk-peak", T)

	// baseline24h ≈ 57%，current1h = 60%：60 >= 42✓, 60 <= 67✓
	// 不触发 improving/degrading；hourly stddev ≈ 32.8 > 20 → volatile
	if trend.Direction != TrendVolatile {
		t.Errorf("Direction = %v, 期望 volatile", trend.Direction)
	}

	// 应检测到奇数 UTC 小时为低谷（10% < 70%）
	if len(trend.PeakHours) == 0 {
		t.Error("PeakHours 为空, 期望检测到质量低谷小时")
	}
	// 验证偶数 UTC 小时不在低谷列表中
	for _, h := range trend.PeakHours {
		if h%2 == 0 {
			t.Errorf("偶数 UTC 小时 %d 不应在低谷列表中", h)
		}
	}
}

// TestHourlyPattern 测试小时热力图生成。
func TestHourlyPattern(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)

	// 所有桶在小时 14 UTC
	start14 := time.Date(2025, 7, 7, 14, 0, 0, 0, time.UTC)
	buckets14 := makeBuckets(start14, 4, 90, 300)

	// 所有桶在小时 3 UTC
	start03 := time.Date(2025, 7, 7, 3, 0, 0, 0, time.UTC)
	buckets03 := makeBuckets(start03, 4, 50, 100)

	all := append(buckets14, buckets03...)

	store := buildTestStore("ep-heatmap", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-heatmap", "mk-heatmap", now)

	// 验证热力图结构
	if len(trend.HourlyPattern) != 24 {
		t.Fatalf("HourlyPattern 长度 = %d, 期望 24", len(trend.HourlyPattern))
	}

	// 小时 14 应有样本，成功率 90%
	h14 := trend.HourlyPattern[14]
	if h14.SampleCount != 80 { // 4 桶 × 20 请求/桶
		t.Errorf("小时 14 SampleCount = %d, 期望 80", h14.SampleCount)
	}
	if h14.AvgSuccessRate < 89.9 || h14.AvgSuccessRate > 90.1 {
		t.Errorf("小时 14 AvgSuccessRate = %f, 期望 90", h14.AvgSuccessRate)
	}

	// 小时 3 应有样本，成功率 50%
	h03 := trend.HourlyPattern[3]
	if h03.SampleCount != 80 {
		t.Errorf("小时 3 SampleCount = %d, 期望 80", h03.SampleCount)
	}

	// 小时 0 应无样本
	h00 := trend.HourlyPattern[0]
	if h00.SampleCount != 0 {
		t.Errorf("小时 0 SampleCount = %d, 期望 0（无样本）", h00.SampleCount)
	}
}

// TestStdDev 测试标准差计算。
func TestStdDev(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		expected float64 // 精确到 0.01
	}{
		{
			name:     "空切片",
			data:     nil,
			expected: 0,
		},
		{
			name:     "单元素",
			data:     []float64{50},
			expected: 0,
		},
		{
			name:     "两个相同元素",
			data:     []float64{50, 50},
			expected: 0,
		},
		{
			name:     "50和100",
			data:     []float64{50, 100},
			expected: 25, // stddev = |100-50|/2 = 25
		},
		{
			name:     "三个元素",
			data:     []float64{0, 50, 100},
			expected: 40.82, // sqrt((0-50)^2 + (50-50)^2 + (100-50)^2)/3) = sqrt(5000/3) ≈ 40.82
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stdDev(tt.data)
			if tt.expected == 0 {
				if got != 0 {
					t.Errorf("stdDev(%v) = %f, 期望 0", tt.data, got)
				}
			} else {
				diff := got - tt.expected
				if diff < -0.1 || diff > 0.1 {
					t.Errorf("stdDev(%v) = %f, 期望 ≈ %f", tt.data, got, tt.expected)
				}
			}
		})
	}
}

// TestBuildHourlyPattern 测试小时热力图构建。
func TestBuildHourlyPattern(t *testing.T) {
	// 构造两个桶：小时 5（100% 成功率）和小时 10（50% 成功率）
	buckets := []*TimeBucketMetrics{
		{
			BucketStart:  time.Date(2025, 7, 1, 5, 0, 0, 0, time.UTC),
			RequestCount: 10,
			SuccessCount: 10,
			P95LatencyMs: 100,
		},
		{
			BucketStart:  time.Date(2025, 7, 1, 10, 30, 0, 0, time.UTC),
			RequestCount: 10,
			SuccessCount: 5,
			P95LatencyMs: 300,
		},
	}

	pattern := buildHourlyPattern(buckets)

	if len(pattern) != 24 {
		t.Fatalf("pattern 长度 = %d, 期望 24", len(pattern))
	}

	// 小时 5
	if pattern[5].AvgSuccessRate != 100 {
		t.Errorf("小时 5 AvgSuccessRate = %f, 期望 100", pattern[5].AvgSuccessRate)
	}
	if pattern[5].SampleCount != 10 {
		t.Errorf("小时 5 SampleCount = %d, 期望 10", pattern[5].SampleCount)
	}
	if pattern[5].AvgP95Latency != 100 {
		t.Errorf("小时 5 AvgP95Latency = %d, 期望 100", pattern[5].AvgP95Latency)
	}

	// 小时 10
	if pattern[10].AvgSuccessRate != 50 {
		t.Errorf("小时 10 AvgSuccessRate = %f, 期望 50", pattern[10].AvgSuccessRate)
	}

	// 小时 0（无数据）
	if pattern[0].SampleCount != 0 {
		t.Errorf("小时 0 SampleCount = %d, 期望 0", pattern[0].SampleCount)
	}
}

// TestFindQualityTroughs 测试质量低谷查找。
func TestFindQualityTroughs(t *testing.T) {
	pattern := make([]HourlyQuality, 24)
	// 小时 3: 60%（低于 70% 阈值）
	pattern[3] = HourlyQuality{Hour: 3, AvgSuccessRate: 60, SampleCount: 10}
	// 小时 15: 65%（低于 70% 阈值）
	pattern[15] = HourlyQuality{Hour: 15, AvgSuccessRate: 65, SampleCount: 10}
	// 小时 8: 90%（高于阈值）
	pattern[8] = HourlyQuality{Hour: 8, AvgSuccessRate: 90, SampleCount: 10}
	// 小时 20: 0% 但无样本（不计入低谷）
	pattern[20] = HourlyQuality{Hour: 20, AvgSuccessRate: 0, SampleCount: 0}

	troughs := findQualityTroughs(pattern, 70)

	if len(troughs) != 2 {
		t.Fatalf("troughs 长度 = %d, 期望 2", len(troughs))
	}
	// 验证包含小时 3 和 15
	found := make(map[int]bool)
	for _, h := range troughs {
		found[h] = true
	}
	if !found[3] {
		t.Error("期望小时 3 在低谷列表中")
	}
	if !found[15] {
		t.Error("期望小时 15 在低谷列表中")
	}
}

// TestCurrentHourInList 测试当前小时匹配。
func TestCurrentHourInList(t *testing.T) {
	now := time.Date(2025, 7, 8, 14, 30, 0, 0, time.UTC)
	hours := []int{10, 14, 18}

	if !currentHourInList(now, hours) {
		t.Error("当前小时 14 应在列表中")
	}

	now2 := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	if currentHourInList(now2, hours) {
		t.Error("当前小时 12 不应在列表中")
	}

	// 空列表
	if currentHourInList(now, nil) {
		t.Error("空列表应返回 false")
	}
}

// TestSlowDegradation 测试缓降场景（成功率逐桶下降）。
func TestSlowDegradation(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)

	// 7 天基线：95%
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 95, 200)
	// 最近 24h：85%（低于基线 10% 但不满足 degrading 条件）
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 85, 200)
	// 最近 1h：75%（低于 24h 基准 85% 仅 10%，不满足 >15% 阈值）
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 75, 200)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-slow", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-slow", "mk-slow", now)

	// 缓降：75 vs 85 = 10%，不满足 degrading 的 15% 阈值
	if trend.Direction != TrendStable {
		t.Errorf("Direction = %v, 期望 stable（缓降不满足 degrading 阈值）", trend.Direction)
	}
	// 95*0.75 = 71.25, 当前 75 > 71.25，不触发
	if trend.ShouldReevaluate {
		t.Error("ShouldReevaluate = true, 期望 false（缓降未达重评估阈值）")
	}
}

// TestSevereDegradation 测试严重降级（成功率极低）。
func TestSevereDegradation(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 95, 200)
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 95, 200)
	// 当前 1h：30%（极低成功率）
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 30, 200)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-severe", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-severe", "mk-severe", now)

	if trend.Direction != TrendDegrading {
		t.Errorf("Direction = %v, 期望 degrading", trend.Direction)
	}
	if !trend.ShouldReevaluate {
		t.Error("ShouldReevaluate = false, 期望 true（严重降级）")
	}
	if trend.DeltaPercent >= 0 {
		t.Errorf("DeltaPercent = %f, 期望 < 0", trend.DeltaPercent)
	}
}

// TestRecovery 测试从降级中恢复。
func TestRecovery(t *testing.T) {
	now := time.Date(2025, 7, 8, 12, 0, 0, 0, time.UTC)
	// 7d 基线低（因为之前一直差）
	buckets7d := makeBuckets(now.Add(-7*24*time.Hour), 576, 60, 500)
	// 24h 基线开始好转
	buckets24h := makeBuckets(now.Add(-24*time.Hour), 84, 70, 300)
	// 当前 1h 恢复到 95%
	buckets1h := makeBuckets(now.Add(-1*time.Hour), 4, 95, 200)

	all := append(buckets7d, buckets24h...)
	all = append(all, buckets1h...)

	store := buildTestStore("ep-recovery", all)
	detector := NewQualityTrendDetector(store)

	trend := detector.DetectTrend("ep-recovery", "mk-recovery", now)

	// 当前 95 vs 24h 基准 70: (95-70)/70 = 35.7% > 10%，应为 improving
	if trend.Direction != TrendImproving {
		t.Errorf("Direction = %v, 期望 improving（恢复场景）", trend.Direction)
	}
	if trend.DeltaPercent <= 0 {
		t.Errorf("DeltaPercent = %f, 期望 > 0（恢复场景应为正值）", trend.DeltaPercent)
	}
}
