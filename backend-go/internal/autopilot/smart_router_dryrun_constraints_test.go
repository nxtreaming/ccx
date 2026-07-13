package autopilot

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/scheduler"
)

func TestBuildPlanMatchesAutoHardConstraintSemantics(t *testing.T) {
	const (
		model    = "dryrun-context-model"
		shortUID = "ch_dryrun_short"
		longUID  = "ch_dryrun_long"
	)
	tests := []struct {
		name             string
		windows          []int
		wantFallback     bool
		wantSelectedUID  string
		wantPlanUIDs     []string
		wantPlanSelected []bool
		wantAutoUIDs     []string
	}{
		{
			name: "过滤后推荐可承载渠道", windows: []int{4096, 16384},
			wantSelectedUID: longUID,
			wantPlanUIDs:    []string{longUID, shortUID}, wantPlanSelected: []bool{true, false},
			wantAutoUIDs: []string{longUID},
		},
		{
			name: "全部不满足时与 auto 一致 fail-open", windows: []int{4096, 4096},
			wantFallback: true, wantSelectedUID: shortUID,
			wantPlanUIDs: []string{shortUID, longUID}, wantPlanSelected: []bool{false, false},
			wantAutoUIDs: []string{shortUID, longUID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseTestConfig()
			cfg.Upstream = cfg.Upstream[:2]
			uids := []string{shortUID, longUID}
			for i := range cfg.Upstream {
				cfg.Upstream[i].ChannelUID = uids[i]
				cfg.Upstream[i].SupportedModels = []string{model}
				cfg.Upstream[i].ModelCapabilities = map[string]config.UpstreamModelCapability{
					model: {ContextWindowTokens: tt.windows[i]},
				}
			}
			cfg.AutopilotRouting = config.AutopilotRoutingConfig{RoutingMode: "auto"}

			cfgManager, cleanup := createTestConfigManager(t, cfg)
			defer cleanup()
			router := NewSmartRouter(nil, nil, nil, cfgManager)
			newProfile := func() *RequestProfile {
				return &RequestProfile{
					Model: model, ChannelKind: "messages", Operation: "completion",
					EstTokens: 1000, ContextNeed: 8192,
				}
			}

			plan := router.BuildPlan(newProfile())
			if plan.FallbackUsed != tt.wantFallback || plan.SelectedChannelUID != tt.wantSelectedUID {
				t.Fatalf("plan fallback=%v selected=%q, want %v/%q",
					plan.FallbackUsed, plan.SelectedChannelUID, tt.wantFallback, tt.wantSelectedUID)
			}
			gotPlanUIDs := make([]string, len(plan.Candidates))
			gotSelected := make([]bool, len(plan.Candidates))
			for i, candidate := range plan.Candidates {
				gotPlanUIDs[i] = candidate.ChannelUID
				gotSelected[i] = candidate.Selected
				if !candidate.Selected && (len(candidate.FilterReasons) != 1 || candidate.FilterReasons[0] != "上下文窗口不满足") {
					t.Fatalf("filtered candidate %s reasons=%v", candidate.ChannelUID, candidate.FilterReasons)
				}
			}
			if !reflect.DeepEqual(gotPlanUIDs, tt.wantPlanUIDs) || !reflect.DeepEqual(gotSelected, tt.wantPlanSelected) {
				t.Fatalf("plan candidates=%v selected=%v, want %v/%v",
					gotPlanUIDs, gotSelected, tt.wantPlanUIDs, tt.wantPlanSelected)
			}

			filter := router.CandidateFilterFor(newProfile())
			if filter == nil {
				t.Fatal("auto mode should return candidate filter")
			}
			processed := cfgManager.GetConfig()
			channels := []scheduler.ChannelInfo{
				{Index: 0, Name: processed.Upstream[0].Name, Status: "active"},
				{Index: 1, Name: processed.Upstream[1].Name, Status: "active"},
			}
			result, err := filter(
				channels,
				func(ch scheduler.ChannelInfo) *config.UpstreamConfig { return &processed.Upstream[ch.Index] },
				func(_ scheduler.ChannelInfo, upstream *config.UpstreamConfig) bool { return upstream != nil },
			)
			if err != nil {
				t.Fatalf("auto filter error = %v", err)
			}
			gotAutoUIDs := make([]string, len(result))
			for i, channel := range result {
				gotAutoUIDs[i] = processed.Upstream[channel.Index].ChannelUID
			}
			if !reflect.DeepEqual(gotAutoUIDs, tt.wantAutoUIDs) {
				t.Fatalf("auto candidates=%v, want %v", gotAutoUIDs, tt.wantAutoUIDs)
			}
		})
	}
}

func TestRoutingPlanCandidateJSONKeepsScoreFields(t *testing.T) {
	plan := RoutingPlan{
		Candidates: []RoutingPlanCandidate{{
			ScoredCandidate: ScoredCandidate{ChannelUID: "ch_json", Score: 1.5},
			Selected:        false,
			FilterReasons:   []string{"上下文窗口不满足"},
		}},
		SelectedChannelUID: "ch_json",
		FallbackUsed:       true,
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	candidates, ok := payload["candidates"].([]interface{})
	if !ok || len(candidates) != 1 {
		t.Fatalf("candidates payload = %#v", payload["candidates"])
	}
	candidate := candidates[0].(map[string]interface{})
	if candidate["channelUid"] != "ch_json" || candidate["score"] != 1.5 || candidate["selected"] != false {
		t.Fatalf("candidate JSON fields = %#v", candidate)
	}
}

func TestBuildPlanExcludesInactiveOrCredentiallessChannels(t *testing.T) {
	cfg := baseTestConfig()
	cfg.Upstream[0].ChannelUID = "ch_active"
	cfg.Upstream[1].ChannelUID = "ch_paused"
	cfg.Upstream[1].Status = "paused"
	cfg.Upstream[2].ChannelUID = "ch_no_key"
	cfg.Upstream[2].APIKeys = nil
	cfg.AutopilotRouting = config.AutopilotRoutingConfig{RoutingMode: "shadow"}

	cfgManager, cleanup := createTestConfigManager(t, cfg)
	defer cleanup()
	plan := NewSmartRouter(nil, nil, nil, cfgManager).BuildPlan(&RequestProfile{
		Model: "unknown-model", ChannelKind: "messages", Operation: "completion", EstTokens: 1000,
	})
	if len(plan.Candidates) != 1 || plan.Candidates[0].ChannelUID != "ch_active" {
		t.Fatalf("configured candidates = %+v, want only ch_active", plan.Candidates)
	}
}
