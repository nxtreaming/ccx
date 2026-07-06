package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/warmup"
	"github.com/gin-gonic/gin"
)

func TestDiagnoseSchedulerSelectionReturnsSelectionTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sch, cleanup := newSchedulerDiagnoseTestScheduler(t, config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "skipped",
				BaseURL:  "https://skipped.example.com",
				APIKeys:  []string{"sk-skip"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "selected",
				BaseURL:  "https://selected.example.com",
				APIKeys:  []string{"sk-select"},
				Status:   "active",
				Priority: 2,
			},
		},
	})
	defer cleanup()

	body := []byte(`{"model":"gpt-test","failedChannels":[0]}`)
	resp := postSchedulerDiagnose(t, sch, scheduler.ChannelKindMessages, body)

	if !resp.OK {
		t.Fatalf("ok = false, error = %q", resp.Error)
	}
	if resp.Kind != string(scheduler.ChannelKindMessages) {
		t.Fatalf("kind = %q, want messages", resp.Kind)
	}
	if resp.Selected.ChannelIndex != 1 || resp.Selected.ChannelName != "selected" {
		t.Fatalf("selected = %#v, want channel 1 selected", resp.Selected)
	}
	if resp.Reason != "priority_order" {
		t.Fatalf("reason = %q, want priority_order", resp.Reason)
	}
	if !strings.Contains(resp.Summary, "0:skipped@priority_order/failed_in_request") {
		t.Fatalf("summary = %q, want failed channel skip", resp.Summary)
	}
	if resp.Trace.Selected == nil || resp.Trace.Selected.ChannelIndex != 1 {
		t.Fatalf("trace.selected = %#v, want channel 1", resp.Trace.Selected)
	}
}

func TestDiagnoseSchedulerSelectionReturnsContextError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sch, cleanup := newSchedulerDiagnoseTestScheduler(t, config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "small-window",
				BaseURL: "https://small.example.com",
				APIKeys: []string{"sk-small"},
				Status:  "active",
				ModelCapabilities: map[string]config.UpstreamModelCapability{
					"gpt-large": {ContextWindowTokens: 1000},
				},
			},
		},
	})
	defer cleanup()

	body := []byte(`{"model":"gpt-large","contextRequirement":{"inputTokens":5000,"requiredTokens":5000}}`)
	resp := postSchedulerDiagnose(t, sch, scheduler.ChannelKindMessages, body)

	if resp.OK {
		t.Fatalf("ok = true, want false")
	}
	if !strings.Contains(resp.Error, "上下文") {
		t.Fatalf("error = %q, want context routing error", resp.Error)
	}
	if !strings.Contains(resp.Summary, "small-window@context_filter/context_window_exceeded") {
		t.Fatalf("summary = %q, want context filter skip", resp.Summary)
	}
	if len(resp.Trace.Candidates) != 1 || resp.Trace.Candidates[0].Reason != "context_window_exceeded" {
		t.Fatalf("trace.candidates = %#v, want context_window_exceeded", resp.Trace.Candidates)
	}
}

func TestDiagnoseSchedulerSelectionRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sch, cleanup := newSchedulerDiagnoseTestScheduler(t, config.Config{})
	defer cleanup()

	r := gin.New()
	r.POST("/messages/channels/scheduler/diagnose", DiagnoseSchedulerSelection(sch, scheduler.ChannelKindMessages))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/scheduler/diagnose", bytes.NewReader([]byte(`{`)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

type schedulerDiagnoseTestResponse struct {
	OK       bool                      `json:"ok"`
	Kind     string                    `json:"kind"`
	Reason   string                    `json:"reason"`
	Summary  string                    `json:"summary"`
	Error    string                    `json:"error"`
	Selected schedulerDiagnoseSelected `json:"selected"`
	Trace    scheduler.SelectionTrace  `json:"trace"`
}

type schedulerDiagnoseSelected struct {
	ChannelIndex int    `json:"channelIndex"`
	ChannelName  string `json:"channelName"`
	ServiceType  string `json:"serviceType"`
}

func postSchedulerDiagnose(t *testing.T, sch *scheduler.ChannelScheduler, kind scheduler.ChannelKind, body []byte) schedulerDiagnoseTestResponse {
	t.Helper()

	r := gin.New()
	r.POST("/messages/channels/scheduler/diagnose", DiagnoseSchedulerSelection(sch, kind))

	req := httptest.NewRequest(http.MethodPost, "/messages/channels/scheduler/diagnose", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}

	var resp schedulerDiagnoseTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func newSchedulerDiagnoseTestScheduler(t *testing.T, cfg config.Config) (*scheduler.ChannelScheduler, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	cfgManager, err := config.NewConfigManager(configFile, "")
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	messagesMetrics := metrics.NewMetricsManager()
	responsesMetrics := metrics.NewMetricsManager()
	geminiMetrics := metrics.NewMetricsManager()
	chatMetrics := metrics.NewMetricsManager()
	imagesMetrics := metrics.NewMetricsManager()
	vectorsMetrics := metrics.NewMetricsManager()
	traceAffinity := session.NewTraceAffinityManager()
	urlManager := warmup.NewURLManager(30*time.Second, 3)

	sch := scheduler.NewChannelScheduler(
		cfgManager,
		messagesMetrics,
		responsesMetrics,
		geminiMetrics,
		chatMetrics,
		imagesMetrics,
		traceAffinity,
		urlManager,
		vectorsMetrics,
	)

	cleanup := func() {
		cfgManager.Close()
		messagesMetrics.Stop()
		responsesMetrics.Stop()
		geminiMetrics.Stop()
		chatMetrics.Stop()
		imagesMetrics.Stop()
		vectorsMetrics.Stop()
		traceAffinity.Stop()
	}
	return sch, cleanup
}
