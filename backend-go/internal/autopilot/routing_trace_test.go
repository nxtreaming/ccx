package autopilot

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ── MaskKey 脱敏测试（表驱动）──

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"空字符串", "", ""},
		{"短 key 全掩码", "abc", "****"},
		{"8 字符全掩码", "12345678", "****"},
		{"9 字符保留首尾", "123456789", "1234****6789"},
		{"长 key 保留前4后4", "sk-ant-api1234567890abcdef", "sk-a****cdef"},
		{"超短 key", "ab", "****"},
		{"单字符", "x", "****"},
		{"20 字符", "abcdefghijklmnopqrst", "abcd****qrst"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskKey(tt.key)
			if got != tt.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// ── SanitizeMetricsKey 脱敏测试（表驱动）──

func TestSanitizeMetricsKey(t *testing.T) {
	tests := []struct {
		name       string
		metricsKey string
		wantMasked bool // 是否期望 key 被掩码
	}{
		{"空字符串", "", false},
		{"纯 URL 无分隔符", "https://api.openai.com", false},
		{"管道分隔含长 key", "https://api.openai.com|sk-ant-api1234567890abcdef", true},
		{"斜杠分隔含 key", "https://api.anthropic.com/v1/sk-ant-api1234567890abcdef", true},
		{"短 key 掩码", "https://api.test.com|short", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeMetricsKey(tt.metricsKey)
			if tt.wantMasked {
				if got == tt.metricsKey {
					t.Errorf("SanitizeMetricsKey(%q) 未被掩码", tt.metricsKey)
				}
				// 确认不含原始 key
				if strings.Contains(got, "sk-ant") {
					t.Errorf("SanitizeMetricsKey(%q) 仍含原始 key: %q", tt.metricsKey, got)
				}
			}
		})
	}
}

// ── SanitizeTrace 脱敏校验（构造含 key 的输入确认掩码）──

func TestSanitizeTrace_KeyMasking(t *testing.T) {
	trace := &RoutingDecisionTrace{
		TraceUID:    "rt_test001",
		RequestKind: "messages",
		TaskClass:   TaskClassSupervisor,
		Mode:        RoutingModeShadow,
		PromptHash:  "abcdef1234567890",
		Candidates: []RoutingCandidate{
			{
				ChannelUID: "ch_001",
				MetricsKey: "https://api.openai.com|sk-ant-api1234567890abcdef",
				TotalScore: 0.85,
			},
			{
				ChannelUID: "ch_002",
				MetricsKey: "https://deepseek.com|sk-ds-12345678",
				TotalScore: 0.72,
			},
		},
		SelectedMetricsKey: "https://api.openai.com|sk-ant-api1234567890abcdef",
	}

	SanitizeTrace(trace)

	// 验证 promptHash 已清除
	if trace.PromptHash != "" {
		t.Errorf("SanitizeTrace 未清除 promptHash: %q", trace.PromptHash)
	}

	// 验证候选 metricsKey 已掩码
	for _, c := range trace.Candidates {
		if strings.Contains(c.MetricsKey, "sk-ant-api1234567890abcdef") {
			t.Errorf("候选 MetricsKey 未被掩码: %q", c.MetricsKey)
		}
		if strings.Contains(c.MetricsKey, "sk-ds-12345678") {
			t.Errorf("候选 MetricsKey 未被掩码: %q", c.MetricsKey)
		}
	}

	// 验证最终选择 metricsKey 已掩码
	if strings.Contains(trace.SelectedMetricsKey, "sk-ant-api1234567890abcdef") {
		t.Errorf("SelectedMetricsKey 未被掩码: %q", trace.SelectedMetricsKey)
	}

	// 验证掩码格式：保留前4后4
	if trace.SelectedMetricsKey != "https://api.openai.com|sk-a****cdef" {
		t.Errorf("SelectedMetricsKey 掩码格式不正确: %q", trace.SelectedMetricsKey)
	}
}

// ── SanitizeTraces 批量脱敏 ──

func TestSanitizeTraces_NoPlaintextLeakage(t *testing.T) {
	traces := []*RoutingDecisionTrace{
		{
			TraceUID:           "rt_001",
			RequestKind:        "messages",
			TaskClass:          TaskClassSupervisor,
			Mode:               RoutingModeShadow,
			PromptHash:         "abcdef1234567890",
			SelectedMetricsKey: "https://api.openai.com|sk-ant-api1234567890abcdef",
		},
		{
			TraceUID:           "rt_002",
			RequestKind:        "chat",
			TaskClass:          TaskClassWorker,
			Mode:               RoutingModeShadow,
			PromptHash:         "deadbeef12345678",
			SelectedMetricsKey: "https://deepseek.com|sk-ds-secretkeyvalue123",
		},
	}

	result := SanitizeTraces(traces)

	// 原始数据不应被修改
	if traces[0].PromptHash == "" {
		t.Error("SanitizeTraces 修改了原始数据")
	}

	// 结果应被脱敏
	for _, t_ := range result {
		if t_.PromptHash != "" {
			t.Errorf("SanitizeTraces 结果含 promptHash: %q", t_.PromptHash)
		}
		if strings.Contains(t_.SelectedMetricsKey, "secret") {
			t.Errorf("SanitizeTraces 结果含原始 key: %q", t_.SelectedMetricsKey)
		}
	}
}

// ── TraceStore 环形覆盖测试（纯内存，无 SQLite）──

func TestTraceStore_RingBuffer(t *testing.T) {
	store, err := NewTraceStoreWithDB(nil)
	if err != nil {
		t.Fatalf("创建 TraceStore 失败: %v", err)
	}
	defer store.Close()

	// 写入超过 traceMaxRecords 条记录
	total := traceMaxRecords + 100
	for i := 0; i < total; i++ {
		store.Record(&RoutingDecisionTrace{
			RequestKind: "messages",
			TaskClass:   TaskClassSupervisor,
			Mode:        RoutingModeShadow,
		})
	}

	stats := store.GetStats()
	if stats.TotalCount != traceMaxRecords {
		t.Errorf("环形缓冲区大小: got=%d, want=%d", stats.TotalCount, traceMaxRecords)
	}

	// 获取最近 N 条，验证数量正确
	recent := store.ListRecent(10)
	if len(recent) != 10 {
		t.Errorf("ListRecent(10) 返回 %d 条", len(recent))
	}

	// 获取全部，验证不超过上限
	all := store.ListRecent(traceMaxRecords + 100)
	if len(all) != traceMaxRecords {
		t.Errorf("ListRecent(超限) 返回 %d 条, want %d", len(all), traceMaxRecords)
	}
}

// ── TraceStore Mismatch 过滤测试 ──

func TestTraceStore_MismatchFilter(t *testing.T) {
	store, err := NewTraceStoreWithDB(nil)
	if err != nil {
		t.Fatalf("创建 TraceStore 失败: %v", err)
	}
	defer store.Close()

	// 写入 10 条：5 条 match，5 条 mismatch
	for i := 0; i < 5; i++ {
		store.Record(&RoutingDecisionTrace{
			RequestKind:      "messages",
			TaskClass:        TaskClassSupervisor,
			Mode:             RoutingModeShadow,
			ShadowChannelUID: "ch_shadow",
			ActualChannelUID: "ch_shadow",
			Match:            true,
		})
	}
	for i := 0; i < 5; i++ {
		store.Record(&RoutingDecisionTrace{
			RequestKind:      "messages",
			TaskClass:        TaskClassWorker,
			Mode:             RoutingModeShadow,
			ShadowChannelUID: "ch_shadow",
			ActualChannelUID: "ch_actual",
			Match:            false,
		})
	}

	// 全部记录
	all := store.ListRecentWithFilter(100, false)
	if len(all) != 10 {
		t.Errorf("ListRecentWithFilter(mismatch=false) 返回 %d 条, want 10", len(all))
	}

	// 只看 mismatch
	mismatches := store.ListRecentWithFilter(100, true)
	if len(mismatches) != 5 {
		t.Errorf("ListRecentWithFilter(mismatch=true) 返回 %d 条, want 5", len(mismatches))
	}

	// 确认 mismatch 记录确实不匹配
	for _, m := range mismatches {
		if m.Match {
			t.Errorf("mismatch 过滤返回了 match=true 的记录: uid=%s", m.TraceUID)
		}
	}
}

// ── TraceStats 统计测试 ──

func TestTraceStore_Stats(t *testing.T) {
	store, err := NewTraceStoreWithDB(nil)
	if err != nil {
		t.Fatalf("创建 TraceStore 失败: %v", err)
	}
	defer store.Close()

	// supervisor: 3 match + 1 mismatch = 4
	// worker: 2 match + 2 mismatch = 4
	for i := 0; i < 3; i++ {
		store.Record(&RoutingDecisionTrace{
			RequestKind:      "messages",
			TaskClass:        TaskClassSupervisor,
			Mode:             RoutingModeShadow,
			ShadowChannelUID: "ch_a",
			ActualChannelUID: "ch_a",
			Match:            true,
		})
	}
	store.Record(&RoutingDecisionTrace{
		RequestKind:      "messages",
		TaskClass:        TaskClassSupervisor,
		Mode:             RoutingModeShadow,
		ShadowChannelUID: "ch_a",
		ActualChannelUID: "ch_b",
		Match:            false,
	})
	for i := 0; i < 2; i++ {
		store.Record(&RoutingDecisionTrace{
			RequestKind:      "chat",
			TaskClass:        TaskClassWorker,
			Mode:             RoutingModeShadow,
			ShadowChannelUID: "ch_c",
			ActualChannelUID: "ch_c",
			Match:            true,
		})
	}
	for i := 0; i < 2; i++ {
		store.Record(&RoutingDecisionTrace{
			RequestKind:      "chat",
			TaskClass:        TaskClassWorker,
			Mode:             RoutingModeShadow,
			ShadowChannelUID: "ch_c",
			ActualChannelUID: "ch_d",
			Match:            false,
		})
	}

	stats := store.GetStats()

	if stats.TotalCount != 8 {
		t.Errorf("TotalCount: got=%d, want=8", stats.TotalCount)
	}
	if stats.MismatchCount != 3 {
		t.Errorf("MismatchCount: got=%d, want=3", stats.MismatchCount)
	}
	// mismatch rate = 3/8 = 0.375
	expectedRate := float64(3) / float64(8)
	if stats.MismatchRate != expectedRate {
		t.Errorf("MismatchRate: got=%f, want=%f", stats.MismatchRate, expectedRate)
	}
	if stats.TaskClassDist[string(TaskClassSupervisor)] != 4 {
		t.Errorf("TaskClassDist[supervisor]: got=%d, want=4", stats.TaskClassDist[string(TaskClassSupervisor)])
	}
	if stats.TaskClassDist[string(TaskClassWorker)] != 4 {
		t.Errorf("TaskClassDist[worker]: got=%d, want=4", stats.TaskClassDist[string(TaskClassWorker)])
	}
}

// ── GenerateTraceUID 稳定性测试 ──

func TestGenerateTraceUID_Stability(t *testing.T) {
	// 同样参数应生成稳定 UID
	uid1 := GenerateTraceUID("messages", TaskClassSupervisor, mustParseTime("2025-01-01T00:00:00Z"))
	uid2 := GenerateTraceUID("messages", TaskClassSupervisor, mustParseTime("2025-01-01T00:00:00Z"))
	if uid1 != uid2 {
		t.Errorf("同参数生成不同 UID: %s vs %s", uid1, uid2)
	}

	// 不同参数应生成不同 UID
	uid3 := GenerateTraceUID("chat", TaskClassWorker, mustParseTime("2025-01-01T00:00:00Z"))
	if uid1 == uid3 {
		t.Errorf("不同参数生成相同 UID: %s", uid1)
	}

	// 前缀校验
	if !strings.HasPrefix(uid1, "rt_") {
		t.Errorf("UID 缺少 rt_ 前缀: %s", uid1)
	}
}

// ── RoutingDecisionTrace 无敏感字段结构验证 ──

// TestRoutingDecisionTrace_NoSensitiveFieldNames 确认 RoutingDecisionTrace
// 的 JSON 序列化结果不含明文 key 和 promptHash（脱敏后）。
func TestRoutingDecisionTrace_NoSensitiveFieldNames(t *testing.T) {
	trace := RoutingDecisionTrace{
		TraceUID:           "rt_test",
		RequestKind:        "messages",
		TaskClass:          TaskClassSupervisor,
		TaskDomain:         TaskDomainCoding,
		RequestedModel:     "claude-sonnet-5",
		AgentRole:          "main",
		PromptHash:         "abcdef1234567890",
		Mode:               RoutingModeShadow,
		ShadowChannelUID:   "ch_001",
		ActualChannelUID:   "ch_002",
		Match:              false,
		SelectedMetricsKey: "https://api.openai.com|sk-masked",
		Candidates: []RoutingCandidate{
			{
				ChannelUID: "ch_001",
				MetricsKey: "https://api.openai.com|sk-masked",
				TotalScore: 0.85,
				Scores: []CandidateScore{
					{Dimension: "quality", Score: 0.9, Weight: 0.4},
				},
			},
		},
		GlobalFilterReasons: map[string][]string{
			"health": {"dead 渠道已过滤"},
		},
		SortReasons: []string{"quality_score 降序"},
	}

	// SanitizeTrace 后序列化不应含明文 key
	SanitizeTrace(&trace)
	data, _ := json.Marshal(trace)
	jsonStr := string(data)

	// 确认不含明文 key
	if strings.Contains(jsonStr, "sk-masked") {
		t.Error("SanitizeTrace 后仍含原始 key 明文")
	}
	// 确认不含 promptHash
	if strings.Contains(jsonStr, "abcdef1234567890") {
		t.Error("SanitizeTrace 后仍含 promptHash")
	}
}

// ── 辅助函数 ──

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// ── TraceUID 生成测试 ──

func TestGeneratePromptHash(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantLen int // 预期 hex 长度（16 位 = 8 字节）
	}{
		{"空字符串", "", 0},
		{"普通文本", "hello world", 16},
		{"中文文本", "你好世界", 16},
		{"长文本", strings.Repeat("a", 10000), 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePromptHash(tt.prompt)
			if len(got) != tt.wantLen {
				t.Errorf("GeneratePromptHash(%q) 长度: got=%d, want=%d", tt.prompt[:min(len(tt.prompt), 20)], len(got), tt.wantLen)
			}
		})
	}

	// 相同输入产生相同 hash
	h1 := GeneratePromptHash("test prompt")
	h2 := GeneratePromptHash("test prompt")
	if h1 != h2 {
		t.Errorf("同输入不同 hash: %s vs %s", h1, h2)
	}

	// 不同输入产生不同 hash
	h3 := GeneratePromptHash("other prompt")
	if h1 == h3 {
		t.Errorf("不同输入相同 hash: %s", h1)
	}
}

// ── TraceStore ListRecent 脱敏验证 ──

func TestTraceStore_ListRecent_Sanitized(t *testing.T) {
	store, err := NewTraceStoreWithDB(nil)
	if err != nil {
		t.Fatalf("创建 TraceStore 失败: %v", err)
	}
	defer store.Close()

	store.Record(&RoutingDecisionTrace{
		RequestKind:        "messages",
		TaskClass:          TaskClassSupervisor,
		Mode:               RoutingModeShadow,
		PromptHash:         "abcdef1234567890",
		SelectedMetricsKey: "https://api.openai.com|sk-ant-api1234567890abcdef",
	})

	recent := store.ListRecent(10)
	if len(recent) == 0 {
		t.Fatal("ListRecent 返回空")
	}

	trace := recent[0]
	// 脱敏后 promptHash 应为空
	if trace.PromptHash != "" {
		t.Errorf("ListRecent 结果含 promptHash: %q", trace.PromptHash)
	}
	// 脱敏后 key 应被掩码
	if strings.Contains(trace.SelectedMetricsKey, "sk-ant-api1234567890abcdef") {
		t.Errorf("ListRecent 结果含原始 key: %q", trace.SelectedMetricsKey)
	}
}
