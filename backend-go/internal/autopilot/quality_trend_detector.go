package autopilot

import (
	"fmt"
	"math"
	"time"
)

// ── 趋势方向 ──

// TrendDirection 描述 endpoint 质量的变化方向。
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving" // 质量改善
	TrendStable    TrendDirection = "stable"    // 质量稳定
	TrendDegrading TrendDirection = "degrading" // 质量下降
	TrendVolatile  TrendDirection = "volatile"  // 忙闲交替剧烈
)

// ── 时段质量 ──

// HourlyQuality 某个小时的平均质量（UTC）。
type HourlyQuality struct {
	Hour           int     `json:"hour"` // 0-23 UTC
	AvgSuccessRate float64 `json:"avgSuccessRate"`
	AvgP95Latency  int64   `json:"avgP95Latency"`
	SampleCount    int     `json:"sampleCount"` // 该小时总请求数（跨多天）
}

// ── 质量趋势 ──

// QualityTrend 描述 endpoint 质量的变化方向和幅度。
type QualityTrend struct {
	MetricsKey string    `json:"metricsKey"`
	DetectedAt time.Time `json:"detectedAt"`

	// ── 趋势方向 ──
	Direction TrendDirection `json:"direction"` // improving | stable | degrading | volatile

	// ── 对比基准 ──
	BaselineWindow      string  `json:"baselineWindow"` // "7d" | "24h" | "1h"
	BaselineSuccessRate float64 `json:"baselineSuccessRate"`
	CurrentSuccessRate  float64 `json:"currentSuccessRate"`
	DeltaPercent        float64 `json:"deltaPercent"` // 当前 vs 基准的变化百分比

	// ── 时段模式 ──
	HourlyPattern  []HourlyQuality `json:"hourlyPattern,omitempty"` // 24 小时质量热力图
	PeakHours      []int           `json:"peakHours,omitempty"`     // 质量低谷的小时列表
	OffPeakQuality QualityTier     `json:"offPeakQuality,omitempty"`

	// ── 触发动作 ──
	ShouldReevaluate bool   `json:"shouldReevaluate"` // 是否需要重新评估画像
	ReevalReason     string `json:"reevalReason,omitempty"`
}

// ── 内部聚合窗口 ──

// aggregatedWindow 内部聚合数据，用于趋势判定。
type aggregatedWindow struct {
	SuccessRate  float64
	P95LatencyMs int64
	RequestCount int
}

// ── QualityTrendDetector ──

// QualityTrendDetector 质量趋势检测器。
// Phase 1：从 TimeBucketStore 读取桶序列进行趋势分析。
// Future：与 ProfileStore / SQLite 持久化集成。
type QualityTrendDetector struct {
	store *TimeBucketStore
}

// NewQualityTrendDetector 创建趋势检测器。
func NewQualityTrendDetector(store *TimeBucketStore) *QualityTrendDetector {
	return &QualityTrendDetector{store: store}
}

// DetectTrend 分析 endpoint 的质量趋势。
// 从 TimeBucketStore 取最近 7 天桶，按设计 §3.10 趋势检测逻辑分析。
func (d *QualityTrendDetector) DetectTrend(
	endpointUID string,
	metricsKey string,
	currentTime time.Time,
) QualityTrend {
	currentTime = currentTime.UTC()

	// 1. 取最近 7 天的时间桶数据（Phase 1: 全量取，由 ringBuffer 自然截断至 7 天）
	buckets := d.store.GetBuckets(endpointUID, maxBuckets)
	if len(buckets) == 0 {
		return QualityTrend{
			MetricsKey:     metricsKey,
			DetectedAt:     currentTime,
			Direction:      TrendStable,
			BaselineWindow: "7d",
		}
	}

	// 2. 构建三个窗口的聚合
	current1h := aggregateBuckets(buckets,
		currentTime.Add(-1*time.Hour), currentTime)
	baseline24h := aggregateBuckets(buckets,
		currentTime.Add(-24*time.Hour), currentTime.Add(-1*time.Hour))
	baseline7d := aggregateBuckets(buckets,
		currentTime.Add(-7*24*time.Hour), currentTime.Add(-24*time.Hour))

	// 3. 判断趋势方向
	var trendDir TrendDirection
	if baseline24h.RequestCount > 0 {
		hourlyStdDev := computeHourlyStdDev(buckets, currentTime)
		switch {
		case current1h.SuccessRate < baseline24h.SuccessRate-15 &&
			baseline7d.RequestCount > 0 && current1h.SuccessRate < baseline7d.SuccessRate-10:
			trendDir = TrendDegrading
		case current1h.SuccessRate > baseline24h.SuccessRate+10:
			trendDir = TrendImproving
		case hourlyStdDev > 20:
			trendDir = TrendVolatile
		default:
			trendDir = TrendStable
		}
	} else {
		trendDir = TrendStable
	}

	// 4. 构建 24 小时质量热力图（UTC 小时 × 平均成功率）
	hourlyPattern := buildHourlyPattern(buckets)
	peakHours := findQualityTroughs(hourlyPattern, 70)

	// 5. 判断是否需要重新评估
	shouldReevaluate := false
	var reason string
	switch {
	case baseline7d.RequestCount > 0 && current1h.SuccessRate < baseline7d.SuccessRate*0.75:
		shouldReevaluate = true
		reason = fmt.Sprintf("成功率从 %.0f%% 降至 %.0f%%", baseline7d.SuccessRate*100, current1h.SuccessRate*100)
	case baseline7d.RequestCount > 0 && current1h.P95LatencyMs > baseline7d.P95LatencyMs*2:
		shouldReevaluate = true
		reason = fmt.Sprintf("p95 延迟从 %dms 升至 %dms", baseline7d.P95LatencyMs, current1h.P95LatencyMs)
	case len(peakHours) > 0 && currentHourInList(currentTime, peakHours):
		reason = fmt.Sprintf("当前处于已知低谷时段 %v", peakHours)
		// 不触发重评估，但标记为时段性降级
	}

	// 计算 DeltaPercent：当前 vs 7d 基准
	var deltaPercent float64
	if baseline7d.RequestCount > 0 && baseline7d.SuccessRate > 0 {
		deltaPercent = (current1h.SuccessRate - baseline7d.SuccessRate) / baseline7d.SuccessRate * 100
	}

	return QualityTrend{
		MetricsKey:          metricsKey,
		DetectedAt:          currentTime,
		Direction:           trendDir,
		BaselineWindow:      "7d",
		BaselineSuccessRate: baseline7d.SuccessRate,
		CurrentSuccessRate:  current1h.SuccessRate,
		DeltaPercent:        deltaPercent,
		HourlyPattern:       hourlyPattern,
		PeakHours:           peakHours,
		ShouldReevaluate:    shouldReevaluate,
		ReevalReason:        reason,
	}
}

// ── 聚合辅助函数 ──

// aggregateBuckets 聚合 [start, end) 时间范围内的桶。
// 范围左闭右开，避免桶被重复计入。
func aggregateBuckets(buckets []*TimeBucketMetrics, start, end time.Time) aggregatedWindow {
	var totalReq, totalSuccess int
	var allLatencies []int64

	for _, b := range buckets {
		if b.BucketStart.Before(start) || !b.BucketStart.Before(end) {
			continue
		}
		totalReq += b.RequestCount
		totalSuccess += b.SuccessCount
		if b.RequestCount > 0 && b.P95LatencyMs > 0 {
			allLatencies = append(allLatencies, b.P95LatencyMs)
		}
	}

	agg := aggregatedWindow{RequestCount: totalReq}
	if totalReq > 0 {
		agg.SuccessRate = float64(totalSuccess) / float64(totalReq) * 100
	}
	if len(allLatencies) > 0 {
		agg.P95LatencyMs = PercentileInt64(allLatencies, 95)
	}
	return agg
}

// buildHourlyPattern 构建 24 小时质量热力图（UTC 小时 → 平均质量）。
func buildHourlyPattern(buckets []*TimeBucketMetrics) []HourlyQuality {
	// 按 UTC 小时聚合
	type hourAgg struct {
		totalReq     int
		totalSuccess int
		totalLatency int64
		latencyCount int
	}
	hours := make([]hourAgg, 24)

	for _, b := range buckets {
		h := b.BucketStart.UTC().Hour()
		hours[h].totalReq += b.RequestCount
		hours[h].totalSuccess += b.SuccessCount
		if b.RequestCount > 0 && b.P95LatencyMs > 0 {
			hours[h].totalLatency += b.P95LatencyMs
			hours[h].latencyCount++
		}
	}

	pattern := make([]HourlyQuality, 24)
	for h := 0; h < 24; h++ {
		pattern[h].Hour = h
		pattern[h].SampleCount = hours[h].totalReq
		if hours[h].totalReq > 0 {
			pattern[h].AvgSuccessRate = float64(hours[h].totalSuccess) / float64(hours[h].totalReq) * 100
		}
		if hours[h].latencyCount > 0 {
			pattern[h].AvgP95Latency = hours[h].totalLatency / int64(hours[h].latencyCount)
		}
	}
	return pattern
}

// findQualityTroughs 找出质量低于阈值的小时列表。
// 阈值为成功率百分比（如 70 表示 70%）。
func findQualityTroughs(pattern []HourlyQuality, thresholdPct float64) []int {
	var troughs []int
	for _, h := range pattern {
		// 无样本的小时不计入低谷
		if h.SampleCount > 0 && h.AvgSuccessRate < thresholdPct {
			troughs = append(troughs, h.Hour)
		}
	}
	return troughs
}

// currentHourInList 判断当前 UTC 小时是否在列表中。
func currentHourInList(t time.Time, hours []int) bool {
	cur := t.UTC().Hour()
	for _, h := range hours {
		if h == cur {
			return true
		}
	}
	return false
}

// computeHourlyStdDev 计算最近 24 小时内，每小时成功率的标准差。
// 用于检测 volatile 趋势（stddev > 20 视为忙闲交替剧烈）。
func computeHourlyStdDev(buckets []*TimeBucketMetrics, currentTime time.Time) float64 {
	start := currentTime.Add(-24 * time.Hour)

	// 按小时聚合
	type hourData struct {
		totalReq     int
		totalSuccess int
	}
	hours := make(map[int]*hourData) // key: 小时偏移 (0-23)

	for _, b := range buckets {
		if b.BucketStart.Before(start) || !b.BucketStart.Before(currentTime) {
			continue
		}
		h := int(currentTime.Sub(b.BucketStart).Hours())
		if h < 0 || h >= 24 {
			continue
		}
		if hours[h] == nil {
			hours[h] = &hourData{}
		}
		hours[h].totalReq += b.RequestCount
		hours[h].totalSuccess += b.SuccessCount
	}

	// 收集有样本的小时成功率
	var rates []float64
	for _, hd := range hours {
		if hd.totalReq > 0 {
			rates = append(rates, float64(hd.totalSuccess)/float64(hd.totalReq)*100)
		}
	}

	return stdDev(rates)
}

// stdDev 计算浮点切片的总体标准差。数据不足返回 0。
func stdDev(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	var variance float64
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(data))
	return math.Sqrt(variance)
}
