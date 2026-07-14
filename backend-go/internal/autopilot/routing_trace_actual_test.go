package autopilot

import "testing"

func TestTraceStoreUpdateActualChannelTargetsTraceUID(t *testing.T) {
	store, err := NewTraceStoreWithDB(nil)
	if err != nil {
		t.Fatalf("NewTraceStoreWithDB() error = %v", err)
	}

	first := &RoutingDecisionTrace{
		TraceUID: "rt_first", Mode: RoutingModeShadow,
		SelectedChannelUID: "ch_shadow_first", ShadowChannelUID: "ch_shadow_first", Match: true,
	}
	second := &RoutingDecisionTrace{
		TraceUID: "rt_second", Mode: RoutingModeShadow,
		SelectedChannelUID: "ch_shadow_second", ShadowChannelUID: "ch_shadow_second", Match: true,
	}
	assist := &RoutingDecisionTrace{
		TraceUID: "rt_assist", Mode: RoutingModeAssist,
		SelectedChannelUID: "ch_assist",
	}
	endpoint := &RoutingDecisionTrace{
		TraceUID: "rt_endpoint", Mode: RoutingModeShadow,
		SortReasons: []string{"endpoint_policy_url_order"},
	}
	store.Record(first)
	store.Record(second)
	store.Record(assist)
	store.Record(endpoint)

	if err := store.UpdateActualChannel(first.TraceUID, "ch_actual_first"); err != nil {
		t.Fatalf("UpdateActualChannel(first) error = %v", err)
	}
	if err := store.UpdateActualChannel(endpoint.TraceUID, "ch_endpoint"); err != nil {
		t.Fatalf("UpdateActualChannel(endpoint) error = %v", err)
	}
	if err := store.UpdateActualChannel(assist.TraceUID, "ch_assist"); err != nil {
		t.Fatalf("UpdateActualChannel(assist) error = %v", err)
	}

	if first.ActualChannelUID != "ch_actual_first" || first.Match {
		t.Fatalf("first trace not updated correctly: %+v", first)
	}
	if second.ActualChannelUID != "" {
		t.Fatalf("second trace was cross-updated: %+v", second)
	}
	if endpoint.ActualChannelUID != "" {
		t.Fatalf("endpoint trace should not be treated as channel comparison: %+v", endpoint)
	}
	if assist.ActualChannelUID != "ch_assist" {
		t.Fatalf("assist trace should record actual channel: %+v", assist)
	}
	if assist.Match {
		t.Fatalf("assist trace should not enter shadow match statistics: %+v", assist)
	}
}

func TestTraceStatsMismatchRateUsesComparableTraces(t *testing.T) {
	store, err := NewTraceStoreWithDB(nil)
	if err != nil {
		t.Fatalf("NewTraceStoreWithDB() error = %v", err)
	}

	store.Record(&RoutingDecisionTrace{
		Mode: RoutingModeShadow, ShadowChannelUID: "ch_a", ActualChannelUID: "ch_a", Match: true,
	})
	store.Record(&RoutingDecisionTrace{
		Mode: RoutingModeShadow, ShadowChannelUID: "ch_a", ActualChannelUID: "ch_b", Match: false,
	})
	store.Record(&RoutingDecisionTrace{Mode: RoutingModeShadow}) // endpoint trace
	store.Record(&RoutingDecisionTrace{Mode: RoutingModeShadow, ShadowChannelUID: "ch_pending", Match: true})

	stats := store.GetStats()
	if stats.TotalCount != 4 || stats.ComparedCount != 2 || stats.MismatchCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.MismatchRate != 0.5 {
		t.Fatalf("MismatchRate = %v, want 0.5", stats.MismatchRate)
	}
}
