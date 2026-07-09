package autopilot

import "testing"

func TestAggregateChannelProfile_UsesEffectiveStabilityTierWhenSet(t *testing.T) {
	ep1 := KeyEndpointProfile{
		EndpointUID:            "ep-1",
		HealthState:            HealthStateHealthy,
		QualityTier:            QualityTierNormal,
		StabilityTier:          StabilityTierUnstable, // raw tier = unstable
		EffectiveStabilityTier: StabilityTierStable,    // 但滞后后为 stable
		SpeedTier:              SpeedTierNormal,
		CostTier:               CostTierNormal,
		Confidence:             0.5,
	}
	ep2 := KeyEndpointProfile{
		EndpointUID:            "ep-2",
		HealthState:            HealthStateHealthy,
		QualityTier:            QualityTierNormal,
		StabilityTier:          StabilityTierNormal,  // raw tier = normal
		EffectiveStabilityTier: StabilityTierNormal,   // 滞后后也为 normal
		SpeedTier:              SpeedTierNormal,
		CostTier:               CostTierNormal,
		Confidence:             0.5,
	}

	cp := AggregateChannelProfile("ch-1", 0, "messages", []KeyEndpointProfile{ep1, ep2})

	// 聚合应取中位数：[stable(2), normal(1)] -> 排序后 [1,2] -> median(index=1)=2 -> stable
	// 验证聚合读取的是 EffectiveStabilityTier 而非 StabilityTier
	// （如果读取 StabilityTier，ep1 为 unstable(0)，[0,1] median=1 -> normal，不会是 stable）
	if cp.StabilityTier != StabilityTierStable {
		t.Errorf("聚合 StabilityTier 应使用 EffectiveStabilityTier: got %s, want stable", cp.StabilityTier)
	}
}

func TestAggregateChannelProfile_FallsBackToStabilityTierWhenEffectiveEmpty(t *testing.T) {
	ep1 := KeyEndpointProfile{
		EndpointUID:            "ep-1",
		HealthState:            HealthStateHealthy,
		QualityTier:            QualityTierNormal,
		StabilityTier:          StabilityTierStable, // raw tier = stable
		EffectiveStabilityTier: "",                   // 未经过滞后
		SpeedTier:              SpeedTierNormal,
		CostTier:               CostTierNormal,
		Confidence:             0.5,
	}
	ep2 := KeyEndpointProfile{
		EndpointUID:            "ep-2",
		HealthState:            HealthStateHealthy,
		QualityTier:            QualityTierNormal,
		StabilityTier:          StabilityTierNormal,  // raw tier = normal
		EffectiveStabilityTier: StabilityTierNormal,   // 有滞后值
		SpeedTier:              SpeedTierNormal,
		CostTier:               CostTierNormal,
		Confidence:             0.5,
	}

	cp := AggregateChannelProfile("ch-1", 0, "messages", []KeyEndpointProfile{ep1, ep2})

	// ep1 EffectiveStabilityTier="" -> fallback to StabilityTier=stable(2)
	// ep2 EffectiveStabilityTier=normal(1)
	// [2, 1] -> sorted [1,2] -> median(index=1)=2 -> stable
	if cp.StabilityTier != StabilityTierStable {
		t.Errorf("EffectiveStabilityTier 为空时应回退到 StabilityTier: got %s, want stable", cp.StabilityTier)
	}
}

func TestAggregateChannelProfile_MixedEffectiveAndFallback(t *testing.T) {
	ep1 := KeyEndpointProfile{
		EndpointUID:            "ep-1",
		HealthState:            HealthStateHealthy,
		QualityTier:            QualityTierNormal,
		StabilityTier:          StabilityTierStable,
		EffectiveStabilityTier: StabilityTierStable,
		SpeedTier:              SpeedTierNormal,
		CostTier:               CostTierNormal,
		Confidence:             0.5,
	}
	ep2 := KeyEndpointProfile{
		EndpointUID:   "ep-2",
		HealthState:   HealthStateHealthy,
		QualityTier:   QualityTierNormal,
		StabilityTier: StabilityTierUnstable, // raw = unstable
		// EffectiveStabilityTier 零值（未经过滞后），fallback 到 StabilityTier=unstable(0)
		SpeedTier:  SpeedTierNormal,
		CostTier:   CostTierNormal,
		Confidence: 0.5,
	}

	cp := AggregateChannelProfile("ch-1", 0, "messages", []KeyEndpointProfile{ep1, ep2})

	// ep1: stable(2), ep2: fallback unstable(0)
	// [2, 0] -> sorted [0,2] -> median(index=1)=2 -> stable
	if cp.StabilityTier != StabilityTierStable {
		t.Errorf("混合场景聚合结果错误: got %s, want stable", cp.StabilityTier)
	}
}
