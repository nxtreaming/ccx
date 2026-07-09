package autopilot

import (
	"testing"
	"time"
)

func TestCarryForwardProbeFields_NilOldIsNoop(t *testing.T) {
	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.ProbeSuccess = false

	carryForwardProbeFields(nil, current)

	if current.ProbeSuccess {
		t.Error("old 为 nil 时不应修改 current")
	}
}

func TestCarryForwardProbeFields_NilCurrentIsNoop(t *testing.T) {
	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	// 不应 panic
	carryForwardProbeFields(old, nil)
}

func TestCarryForwardProbeFields_CopiesProbeFields(t *testing.T) {
	lastProbe := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	old := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	old.LastProbeAt = &lastProbe
	old.ProbeSuccess = true
	old.ProbeLatencyMs = 123
	old.ProbeConfidence = 0.8
	old.ProbeStatusCode = 200
	old.ConsecutiveProbeSuccess = 3

	// current 模拟 DeriveEndpointProfile 每轮新构造的零值 struct，
	// 但 L1 诊断字段（HealthState/HealthConfidence）已经是本轮真实计算结果。
	current := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	current.HealthState = HealthStateDegraded
	current.HealthConfidence = 0.5

	carryForwardProbeFields(old, current)

	if current.LastProbeAt == nil || !current.LastProbeAt.Equal(lastProbe) {
		t.Errorf("LastProbeAt 未搬运: got %v, want %v", current.LastProbeAt, lastProbe)
	}
	if !current.ProbeSuccess {
		t.Error("ProbeSuccess 未搬运")
	}
	if current.ProbeLatencyMs != 123 {
		t.Errorf("ProbeLatencyMs: got %d, want 123", current.ProbeLatencyMs)
	}
	if current.ProbeConfidence != 0.8 {
		t.Errorf("ProbeConfidence: got %f, want 0.8", current.ProbeConfidence)
	}
	if current.ProbeStatusCode != 200 {
		t.Errorf("ProbeStatusCode: got %d, want 200", current.ProbeStatusCode)
	}
	if current.ConsecutiveProbeSuccess != 3 {
		t.Errorf("ConsecutiveProbeSuccess: got %d, want 3", current.ConsecutiveProbeSuccess)
	}

	// L1 诊断字段不应被 carry-forward 覆盖，保持本轮真实计算结果
	if current.HealthState != HealthStateDegraded {
		t.Errorf("HealthState 不应被覆盖: got %s, want degraded", current.HealthState)
	}
	if current.HealthConfidence != 0.5 {
		t.Errorf("HealthConfidence 不应被覆盖: got %f, want 0.5", current.HealthConfidence)
	}
}

// TestCarryForwardProbeFields_SurvivesL1OverwriteBug 回归测试：
// 模拟 collectAll 每轮调用 DeriveEndpointProfile 构造全新 struct 后 Upsert 的场景。
// 若不 carry-forward，Probe* 字段会在 L1 循环中被无声清零，破坏 scanAndEnqueue 的冷却期判定。
func TestCarryForwardProbeFields_SurvivesL1OverwriteBug(t *testing.T) {
	store := newTestProfileStore(t)

	lastProbe := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	probed := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	probed.LastProbeAt = &lastProbe
	probed.ConsecutiveProbeSuccess = 1
	if err := store.Upsert(probed); err != nil {
		t.Fatalf("写入探测后画像失败: %v", err)
	}

	// 模拟下一轮 L1：DeriveEndpointProfile 构造的全新 struct，Probe* 字段零值
	freshFromL1 := newTestProfile("ep-1", "ch-1", "messages", "https://example.com")
	freshFromL1.HealthState = HealthStateHealthy // 本轮真实流量信号

	carryForwardProbeFields(store.Get("ep-1"), freshFromL1)
	if err := store.Upsert(freshFromL1); err != nil {
		t.Fatalf("写入本轮画像失败: %v", err)
	}

	result := store.Get("ep-1")
	if result.LastProbeAt == nil || !result.LastProbeAt.Equal(lastProbe) {
		t.Errorf("carry-forward 后 LastProbeAt 应保留: got %v", result.LastProbeAt)
	}
	if result.ConsecutiveProbeSuccess != 1 {
		t.Errorf("carry-forward 后 ConsecutiveProbeSuccess 应保留: got %d, want 1", result.ConsecutiveProbeSuccess)
	}
	if result.HealthState != HealthStateHealthy {
		t.Errorf("HealthState 应使用本轮 L1 计算结果: got %s, want healthy", result.HealthState)
	}
}
