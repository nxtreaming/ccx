package autopilot

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ── RoutingDecisionTrace（P1.4）──

// RoutingMode 追踪记录的运行模式。
type RoutingMode string

const (
	RoutingModeShadow RoutingMode = "shadow"  // shadow：只计算+记录，不影响真实调度
	RoutingModeDryRun RoutingMode = "dry_run" // dry-run：计算但不生效，与 shadow 等价
	RoutingModeAssist RoutingMode = "assist"  // assist：重排候选列表（不删除），影响调度顺序
	RoutingModeAuto   RoutingMode = "auto"    // auto：硬约束过滤+重排，fail-open 兜底
	RoutingModeActive RoutingMode = "active"  // active：计算并影响真实调度（Phase 2+）
)

// CandidateScore 候选渠道单项得分明细。
type CandidateScore struct {
	Dimension string  `json:"dimension"` // quality | cost | health | domain | effort | stability
	Score     float64 `json:"score"`     // 该维度得分
	Weight    float64 `json:"weight"`    // 权重系数
}

// RoutingCandidate 路由候选渠道信息（已脱敏）。
type RoutingCandidate struct {
	ChannelUID  string `json:"channelUid"`
	MetricsKey  string `json:"metricsKey,omitempty"` // 已脱敏：不含 key 明文
	OriginTier  string `json:"originTier,omitempty"`
	ChannelKind string `json:"channelKind,omitempty"`
	HealthState string `json:"healthState,omitempty"`

	// 分数明细
	TotalScore float64          `json:"totalScore"`
	Scores     []CandidateScore `json:"scores,omitempty"`

	// 是否被选中
	Selected bool `json:"selected"`

	// 被过滤的原因列表（非空表示该候选被过滤掉）
	FilterReasons []string `json:"filterReasons,omitempty"`
}

// RoutingDecisionTrace 结构化路由决策追踪。
// 设计文档 P1.4：所有真实或 dry-run 路由都输出结构化 trace。
// 只记录解释性字段，不记录明文 prompt、密钥、敏感 header 或 multipart。
type RoutingDecisionTrace struct {
	TraceUID string `json:"traceUid"`

	// 请求特征摘要（脱敏）
	RequestKind    string     `json:"requestKind"` // messages | chat | responses | gemini
	TaskClass      TaskClass  `json:"taskClass"`
	TaskDomain     TaskDomain `json:"taskDomain,omitempty"`
	RequestedModel string     `json:"requestedModel,omitempty"`
	AgentRole      string     `json:"agentRole,omitempty"` // main | subagent

	// 关联标识
	ManualIntentUID    string `json:"manualIntentUid,omitempty"`
	AdvisorDecisionUID string `json:"advisorDecisionUid,omitempty"`

	// 候选列表（每项含分数明细与过滤原因）
	Candidates       []RoutingCandidate `json:"candidates"`
	CandidatesBefore int                `json:"candidatesBefore"` // 粗筛前候选数
	CandidatesAfter  int                `json:"candidatesAfter"`  // 粗筛后候选数

	// 过滤与排序说明
	GlobalFilterReasons map[string][]string `json:"globalFilterReasons,omitempty"` // key=过滤阶段, value=原因列表
	SortReasons         []string            `json:"sortReasons,omitempty"`

	// 最终选择
	SelectedChannelUID string  `json:"selectedChannelUid,omitempty"`
	SelectedMetricsKey string  `json:"selectedMetricsKey,omitempty"` // 已脱敏
	SelectedOriginTier string  `json:"selectedOriginTier,omitempty"`
	EstimatedCost      float64 `json:"estimatedCost,omitempty"`
	CostConfidence     float64 `json:"costConfidence,omitempty"`
	FallbackUsed       bool    `json:"fallbackUsed"`

	// Shadow 与实际调度结果对比
	ShadowChannelUID string `json:"shadowChannelUid,omitempty"` // shadow 建议的渠道
	ActualChannelUID string `json:"actualChannelUid,omitempty"` // 实际调度使用的渠道
	Match            bool   `json:"match"`                      // shadow 与实际是否一致

	// 脱敏标识
	PromptHash string `json:"promptHash,omitempty"` // prompt SHA256 前 16 位

	// 元信息
	Mode       RoutingMode `json:"mode"`
	DurationMs int64       `json:"durationMs"` // 决策耗时（毫秒）
	CreatedAt  time.Time   `json:"createdAt"`
}

// ── 脱敏输出 ──

// SanitizeMetricsKey 对 metricsKey 中可能包含的 API Key 进行掩码处理。
// metricsKey 格式通常为 "baseURL|apiKey" 或 "baseURL|keyMask"。
// 如果 apiKey 长度 > 8，保留前 4 后 4，中间用 **** 替代；否则全掩码。
func SanitizeMetricsKey(metricsKey string) string {
	if metricsKey == "" {
		return ""
	}

	// 查找分隔符
	for _, sep := range []string{"|", "/"} {
		idx := -1
		for i := len(metricsKey) - 1; i >= 0; i-- {
			if string(metricsKey[i]) == sep {
				idx = i
				break
			}
		}
		if idx > 0 && idx < len(metricsKey)-1 {
			prefix := metricsKey[:idx+1]
			key := metricsKey[idx+1:]
			return prefix + MaskKey(key)
		}
	}

	// 无分隔符，原样返回（可能本身就是 base URL）
	return metricsKey
}

// MaskKey 对 API Key 进行掩码。
// 长度 > 8 时保留前 4 后 4，中间用 **** 替代。
// 长度 <= 8 时全部掩码为 ****。
func MaskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) > 8 {
		return key[:4] + "****" + key[len(key)-4:]
	}
	return "****"
}

// SanitizeTrace 对 RoutingDecisionTrace 做脱敏输出。
// 副作用修改传入指针：
//   - 对所有候选和最终选择的 metricsKey 做掩码
//   - 清除 promptHash（API 响应不返回 hash）
//   - 不含消息正文（结构本身不存储）
func SanitizeTrace(t *RoutingDecisionTrace) {
	if t == nil {
		return
	}

	// 脱敏候选
	for i := range t.Candidates {
		t.Candidates[i].MetricsKey = SanitizeMetricsKey(t.Candidates[i].MetricsKey)
	}

	// 脱敏最终选择
	t.SelectedMetricsKey = SanitizeMetricsKey(t.SelectedMetricsKey)

	// 清除 promptHash（API 响应层不返回，防碰撞攻击）
	t.PromptHash = ""

	// 清除 shadow/actual channel UID 中可能的额外信息（只保留标识符）
	// ChannelUID 本身不含敏感信息（是 UUID），不做额外处理
}

// SanitizeTraces 批量脱敏，返回新切片（不修改原始数据）。
func SanitizeTraces(traces []*RoutingDecisionTrace) []*RoutingDecisionTrace {
	result := make([]*RoutingDecisionTrace, len(traces))
	for i, t := range traces {
		cp := *t
		SanitizeTrace(&cp)
		result[i] = &cp
	}
	return result
}

// ── PromptHash 生成 ──

// GeneratePromptHash 生成 prompt 的短哈希（前 16 位 hex）。
// 只用于关联，不用于安全校验。
func GeneratePromptHash(promptSnippet string) string {
	if promptSnippet == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(promptSnippet))
	return hex.EncodeToString(sum[:8])
}

// ── TraceUID 生成 ──

// GenerateTraceUID 生成路由追踪唯一标识。
func GenerateTraceUID(requestKind string, taskClass TaskClass, createdAt time.Time) string {
	h := fmt.Sprintf("%s|%s|%s", requestKind, string(taskClass), createdAt.Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(h))
	return "rt_" + hex.EncodeToString(sum[:8])
}

// ── TraceStore（内存环形 + 可选 SQLite 落盘）──

const (
	// traceMaxRecords 内存环形保留最大记录数。
	traceMaxRecords = 500

	// traceSampleRate SQLite 抽样率：每 N 条落盘 1 条。
	// mismatch 样本（shadow != actual）全部落盘，不受抽样率限制。
	traceSampleRate = 10
)

// TraceStore 管理 RoutingDecisionTrace 的内存环形缓存与 SQLite 可选落盘。
// 内存环形保留最近 500 条，超出时淘汰最旧记录。
// SQLite 落盘策略：1/10 抽样 + 所有 shadow 与实际不一致的样本。
type TraceStore struct {
	db *sql.DB

	records []*RoutingDecisionTrace // 按时间排序，最新在末尾
	mu      sync.RWMutex

	// 计数器，用于抽样判断
	counter atomic.Int64
}

// NewTraceStoreWithDB 使用外部 *sql.DB 创建 TraceStore。
// db 为 nil 时只使用内存环形缓存，不落盘。
func NewTraceStoreWithDB(db *sql.DB) (*TraceStore, error) {
	if db != nil {
		if err := initTraceSchema(db); err != nil {
			return nil, fmt.Errorf("[TraceStore-Init] 建表失败: %w", err)
		}
	}

	store := &TraceStore{
		db:      db,
		records: make([]*RoutingDecisionTrace, 0, traceMaxRecords),
	}

	// 从 SQLite 加载历史（如有）
	if db != nil {
		if err := store.loadRecent(); err != nil {
			log.Printf("[TraceStore-Init] 警告: 加载历史记录失败: %v", err)
		}
	}

	log.Printf("[TraceStore-Init] 初始化完成，已加载 %d 条追踪记录", len(store.records))
	return store, nil
}

// initTraceSchema 建表迁移。
func initTraceSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS autopilot_routing_traces (
    trace_uid       TEXT PRIMARY KEY,
    request_kind    TEXT    NOT NULL,
    task_class      TEXT    NOT NULL,
    task_domain     TEXT    NOT NULL DEFAULT '',
    requested_model TEXT    NOT NULL DEFAULT '',
    agent_role      TEXT    NOT NULL DEFAULT '',
    mode            TEXT    NOT NULL DEFAULT 'shadow',
    shadow_uid      TEXT    NOT NULL DEFAULT '',
    actual_uid      TEXT    NOT NULL DEFAULT '',
    match           INTEGER NOT NULL DEFAULT 0,
    selected_uid    TEXT    NOT NULL DEFAULT '',
    fallback_used   INTEGER NOT NULL DEFAULT 0,
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    prompt_hash     TEXT    NOT NULL DEFAULT '',
    candidates_json TEXT    NOT NULL DEFAULT '[]',
    created_at      TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_routing_traces_created
    ON autopilot_routing_traces(created_at);
CREATE INDEX IF NOT EXISTS idx_routing_traces_match
    ON autopilot_routing_traces(match);
CREATE INDEX IF NOT EXISTS idx_routing_traces_task_class
    ON autopilot_routing_traces(task_class);
`
	_, err := db.Exec(schema)
	return err
}

// loadRecent 从 SQLite 加载最近 traceMaxRecords 条到内存。
func (s *TraceStore) loadRecent() error {
	rows, err := s.db.Query(`
		SELECT trace_uid, request_kind, task_class, task_domain,
		       requested_model, agent_role, mode,
		       shadow_uid, actual_uid, match,
		       selected_uid, fallback_used, duration_ms,
		       prompt_hash, candidates_json, created_at
		FROM autopilot_routing_traces
		ORDER BY created_at DESC
		LIMIT ?`, traceMaxRecords)
	if err != nil {
		return err
	}
	defer rows.Close()

	var loaded []*RoutingDecisionTrace
	for rows.Next() {
		trace, err := scanTraceRow(rows)
		if err != nil {
			log.Printf("[TraceStore-LoadRecent] 跳过损坏行: %v", err)
			continue
		}
		loaded = append(loaded, trace)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// 反转为时间升序（SQL 用 DESC 取最新，内存按 ASC 存储）
	s.mu.Lock()
	for i := len(loaded) - 1; i >= 0; i-- {
		s.records = append(s.records, loaded[i])
	}
	s.mu.Unlock()

	return nil
}

// scanTraceRow 扫描一行并返回 RoutingDecisionTrace。
func scanTraceRow(rows *sql.Rows) (*RoutingDecisionTrace, error) {
	var t RoutingDecisionTrace
	var matchInt, fallbackInt int
	var createdAt string
	var candidatesJSON string

	err := rows.Scan(
		&t.TraceUID, &t.RequestKind, &t.TaskClass, &t.TaskDomain,
		&t.RequestedModel, &t.AgentRole, &t.Mode,
		&t.ShadowChannelUID, &t.ActualChannelUID, &matchInt,
		&t.SelectedChannelUID, &fallbackInt, &t.DurationMs,
		&t.PromptHash, &candidatesJSON, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	t.Match = matchInt != 0
	t.FallbackUsed = fallbackInt != 0
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	if candidatesJSON != "" && candidatesJSON != "[]" {
		_ = json.Unmarshal([]byte(candidatesJSON), &t.Candidates)
	}

	return &t, nil
}

// ── CRUD ──

// Record 记录一条路由追踪。自动填充 TraceUID / CreatedAt。
// 超出上限时淘汰最旧记录。SQLite 落盘遵循 1/10 抽样 + 全部 mismatch。
func (s *TraceStore) Record(t *RoutingDecisionTrace) {
	if t.TraceUID == "" {
		t.TraceUID = GenerateTraceUID(t.RequestKind, t.TaskClass, time.Now())
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}

	// 写入内存
	s.mu.Lock()
	s.records = append(s.records, t)

	// 淘汰最旧
	if len(s.records) > traceMaxRecords {
		excess := len(s.records) - traceMaxRecords
		s.records = s.records[excess:]
	}
	s.mu.Unlock()

	// SQLite 落盘：1/10 抽样 + 全部 mismatch
	if s.db != nil {
		count := s.counter.Add(1)
		isMismatch := !t.Match && t.ActualChannelUID != "" && t.ShadowChannelUID != ""
		if isMismatch || count%traceSampleRate == 0 {
			if err := s.persistTrace(t); err != nil {
				log.Printf("[TraceStore-Record] 警告: 落盘失败 uid=%s: %v", t.TraceUID, err)
			}
		}
	}
}

// ListRecent 返回最近 N 条记录（脱敏副本，时间降序）。
func (s *TraceStore) ListRecent(n int) []*RoutingDecisionTrace {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.records)
	if n <= 0 || n > total {
		n = total
	}

	// 从末尾取最近 N 条，反转为时间降序
	start := total - n
	result := make([]*RoutingDecisionTrace, n)
	for i := 0; i < n; i++ {
		cp := *s.records[start+i]
		result[n-1-i] = &cp
	}

	// 脱敏
	SanitizeTracesInPlace(result)
	return result
}

// ListRecentWithFilter 返回最近 N 条记录，支持 mismatch 过滤。
// mismatch=true 时只返回 shadow 与实际不一致的记录。
func (s *TraceStore) ListRecentWithFilter(n int, mismatchOnly bool) []*RoutingDecisionTrace {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.records)

	// 从末尾向前遍历，收集符合条件的记录
	var result []*RoutingDecisionTrace
	for i := total - 1; i >= 0 && len(result) < n; i-- {
		t := s.records[i]
		if mismatchOnly && t.Match {
			continue
		}
		if mismatchOnly && (t.ActualChannelUID == "" || t.ShadowChannelUID == "") {
			continue
		}
		cp := *t
		result = append(result, &cp)
	}

	// 脱敏
	SanitizeTracesInPlace(result)
	return result
}

// SanitizeTracesInPlace 就地脱敏（修改指针指向的副本）。
func SanitizeTracesInPlace(traces []*RoutingDecisionTrace) {
	for _, t := range traces {
		SanitizeTrace(t)
	}
}

// ── 统计 ──

// TraceStats 路由追踪统计汇总。
type TraceStats struct {
	TotalCount    int            `json:"totalCount"`
	MismatchCount int            `json:"mismatchCount"`
	MismatchRate  float64        `json:"mismatchRate"`
	TaskClassDist map[string]int `json:"taskClassDist"` // key=TaskClass, value=数量
	ModeDist      map[string]int `json:"modeDist"`      // key=RoutingMode, value=数量
}

// GetStats 计算统计汇总。
func (s *TraceStore) GetStats() TraceStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := TraceStats{
		TotalCount:    len(s.records),
		TaskClassDist: make(map[string]int),
		ModeDist:      make(map[string]int),
	}

	for _, t := range s.records {
		if !t.Match && t.ActualChannelUID != "" && t.ShadowChannelUID != "" {
			stats.MismatchCount++
		}
		stats.TaskClassDist[string(t.TaskClass)]++
		stats.ModeDist[string(t.Mode)]++
	}

	if stats.TotalCount > 0 {
		stats.MismatchRate = float64(stats.MismatchCount) / float64(stats.TotalCount)
	}

	return stats
}

// ── 持久化辅助 ──

// persistTrace 将追踪记录写入 SQLite。
func (s *TraceStore) persistTrace(t *RoutingDecisionTrace) error {
	candidatesJSON, err := json.Marshal(t.Candidates)
	if err != nil {
		candidatesJSON = []byte("[]")
	}

	matchInt := 0
	if t.Match {
		matchInt = 1
	}
	fallbackInt := 0
	if t.FallbackUsed {
		fallbackInt = 1
	}

	_, err = s.db.Exec(`
INSERT INTO autopilot_routing_traces
    (trace_uid, request_kind, task_class, task_domain,
     requested_model, agent_role, mode,
     shadow_uid, actual_uid, match,
     selected_uid, fallback_used, duration_ms,
     prompt_hash, candidates_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(trace_uid) DO UPDATE SET
    shadow_uid  = excluded.shadow_uid,
    actual_uid  = excluded.actual_uid,
    match       = excluded.match,
    duration_ms = excluded.duration_ms
`,
		t.TraceUID,
		t.RequestKind,
		string(t.TaskClass),
		string(t.TaskDomain),
		t.RequestedModel,
		t.AgentRole,
		string(t.Mode),
		t.ShadowChannelUID,
		t.ActualChannelUID,
		matchInt,
		t.SelectedChannelUID,
		fallbackInt,
		t.DurationMs,
		t.PromptHash,
		string(candidatesJSON),
		t.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("[TraceStore-Persist] 写入失败 uid=%s: %w", t.TraceUID, err)
	}
	return nil
}

// Close 关闭 TraceStore。当前无需特殊清理。
func (s *TraceStore) Close() error {
	return nil
}
