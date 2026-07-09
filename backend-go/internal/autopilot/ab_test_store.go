package autopilot

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// ── ABTestStore: A/B 测试结果持久化 ──

// ABTestRecord 单次 A/B 测试影子请求记录。
type ABTestRecord struct {
	RecordUID string `json:"recordUid"`

	// 请求上下文
	Model       string `json:"model"`
	ChannelKind string `json:"channelKind"`

	// 主渠道结果
	PrimaryChannelUID string `json:"primaryChannelUid"`
	PrimarySuccess    bool   `json:"primarySuccess"`
	PrimaryStatusCode int    `json:"primaryStatusCode"`
	PrimaryLatencyMs  int64  `json:"primaryLatencyMs"`

	// 影子渠道结果
	ShadowChannelUID string `json:"shadowChannelUid"`
	ShadowSuccess    bool   `json:"shadowSuccess"`
	ShadowStatusCode int    `json:"shadowStatusCode"`
	ShadowLatencyMs  int64  `json:"shadowLatencyMs"`
	ShadowError      string `json:"shadowError,omitempty"`

	// 成本核算
	ShadowCostUSD float64 `json:"shadowCostUsd"`

	// 元信息
	TraceUID  string    `json:"traceUid,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// ABTestStats A/B 测试聚合统计。
type ABTestStats struct {
	TotalRecords     int     `json:"totalRecords"`
	ShadowSuccessCnt int     `json:"shadowSuccessCount"`
	ShadowFailCnt    int     `json:"shadowFailCount"`
	ShadowSuccessRate float64 `json:"shadowSuccessRate"`
	AvgShadowLatencyMs float64 `json:"avgShadowLatencyMs"`
	TotalShadowCostUSD float64 `json:"totalShadowCostUsd"`
	// 按渠道分组的影子成功率
	ByChannel map[string]*ABTestChannelStats `json:"byChannel,omitempty"`
}

// ABTestChannelStats 按渠道分组的 A/B 测试统计。
type ABTestChannelStats struct {
	ChannelUID     string  `json:"channelUid"`
	Count          int     `json:"count"`
	SuccessCount   int     `json:"successCount"`
	SuccessRate    float64 `json:"successRate"`
	AvgLatencyMs   float64 `json:"avgLatencyMs"`
	TotalCostUSD   float64 `json:"totalCostUsd"`
}

// ABTestStore 管理 A/B 测试结果的内存缓存与 SQLite 持久化。
type ABTestStore struct {
	db *sql.DB

	records []*ABTestRecord
	mu      sync.RWMutex

	maxRecords int
}

// NewABTestStoreWithDB 使用外部提供的 *sql.DB 创建 ABTestStore。
func NewABTestStoreWithDB(db *sql.DB) (*ABTestStore, error) {
	if err := initABTestStoreSchema(db); err != nil {
		return nil, fmt.Errorf("[ABTestStore-Init] 建表失败: %w", err)
	}

	store := &ABTestStore{
		db:         db,
		maxRecords: 2000,
	}

	if err := store.loadAll(); err != nil {
		return nil, fmt.Errorf("[ABTestStore-Init] 加载记录失败: %w", err)
	}

	log.Printf("[ABTestStore-Init] 初始化完成，已加载 %d 条 A/B 测试记录", len(store.records))
	return store, nil
}

func initABTestStoreSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS ab_test_records (
    record_uid        TEXT PRIMARY KEY,
    model             TEXT NOT NULL,
    channel_kind      TEXT NOT NULL,
    primary_channel   TEXT NOT NULL,
    primary_success   INTEGER NOT NULL DEFAULT 0,
    primary_status    INTEGER NOT NULL DEFAULT 0,
    primary_latency   INTEGER NOT NULL DEFAULT 0,
    shadow_channel    TEXT NOT NULL,
    shadow_success    INTEGER NOT NULL DEFAULT 0,
    shadow_status     INTEGER NOT NULL DEFAULT 0,
    shadow_latency    INTEGER NOT NULL DEFAULT 0,
    shadow_error      TEXT NOT NULL DEFAULT '',
    shadow_cost_usd   REAL NOT NULL DEFAULT 0.0,
    trace_uid         TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ab_test_model ON ab_test_records(model);
CREATE INDEX IF NOT EXISTS idx_ab_test_created ON ab_test_records(created_at);
`
	_, err := db.Exec(schema)
	return err
}

func (s *ABTestStore) loadAll() error {
	rows, err := s.db.Query(`
SELECT record_uid, model, channel_kind,
       primary_channel, primary_success, primary_status, primary_latency,
       shadow_channel, shadow_success, shadow_status, shadow_latency,
       shadow_error, shadow_cost_usd, trace_uid, created_at
FROM ab_test_records ORDER BY created_at DESC LIMIT ?`, s.maxRecords)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		r := &ABTestRecord{}
		var createdAt string
		if err := rows.Scan(
			&r.RecordUID, &r.Model, &r.ChannelKind,
			&r.PrimaryChannelUID, &r.PrimarySuccess, &r.PrimaryStatusCode, &r.PrimaryLatencyMs,
			&r.ShadowChannelUID, &r.ShadowSuccess, &r.ShadowStatusCode, &r.ShadowLatencyMs,
			&r.ShadowError, &r.ShadowCostUSD, &r.TraceUID, &createdAt,
		); err != nil {
			log.Printf("[ABTestStore-Load] 警告: 跳过记录: %v", err)
			continue
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			r.CreatedAt = t
		}
		s.records = append(s.records, r)
	}
	return rows.Err()
}

// Record 保存一条 A/B 测试结果（内存 + SQLite 双写）。
func (s *ABTestStore) Record(r *ABTestRecord) {
	if r == nil {
		return
	}
	r.CreatedAt = time.Now().UTC()

	s.mu.Lock()
	s.records = append(s.records, r)
	// 内存环形缓冲：超过上限时淘汰最旧
	if len(s.records) > s.maxRecords {
		s.records = s.records[len(s.records)-s.maxRecords:]
	}
	s.mu.Unlock()

	// 异步落盘
	go s.persistRecord(r)
}

func (s *ABTestStore) persistRecord(r *ABTestRecord) {
	_, err := s.db.Exec(`
INSERT OR REPLACE INTO ab_test_records
(record_uid, model, channel_kind,
 primary_channel, primary_success, primary_status, primary_latency,
 shadow_channel, shadow_success, shadow_status, shadow_latency,
 shadow_error, shadow_cost_usd, trace_uid, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.RecordUID, r.Model, r.ChannelKind,
		r.PrimaryChannelUID, boolToInt(r.PrimarySuccess), r.PrimaryStatusCode, r.PrimaryLatencyMs,
		r.ShadowChannelUID, boolToInt(r.ShadowSuccess), r.ShadowStatusCode, r.ShadowLatencyMs,
		r.ShadowError, r.ShadowCostUSD, r.TraceUID, r.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		log.Printf("[ABTestStore-Persist] 写入失败: uid=%s err=%v", r.RecordUID, err)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetStats 返回 A/B 测试聚合统计。
func (s *ABTestStore) GetStats() *ABTestStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &ABTestStats{
		TotalRecords: len(s.records),
		ByChannel:    make(map[string]*ABTestChannelStats),
	}

	for _, r := range s.records {
		if r.ShadowSuccess {
			stats.ShadowSuccessCnt++
		} else {
			stats.ShadowFailCnt++
		}
		stats.AvgShadowLatencyMs += float64(r.ShadowLatencyMs)
		stats.TotalShadowCostUSD += r.ShadowCostUSD

		cs, ok := stats.ByChannel[r.ShadowChannelUID]
		if !ok {
			cs = &ABTestChannelStats{ChannelUID: r.ShadowChannelUID}
			stats.ByChannel[r.ShadowChannelUID] = cs
		}
		cs.Count++
		if r.ShadowSuccess {
			cs.SuccessCount++
		}
		cs.AvgLatencyMs += float64(r.ShadowLatencyMs)
		cs.TotalCostUSD += r.ShadowCostUSD
	}

	if stats.TotalRecords > 0 {
		stats.ShadowSuccessRate = float64(stats.ShadowSuccessCnt) / float64(stats.TotalRecords)
		stats.AvgShadowLatencyMs /= float64(stats.TotalRecords)
	}
	for _, cs := range stats.ByChannel {
		if cs.Count > 0 {
			cs.SuccessRate = float64(cs.SuccessCount) / float64(cs.Count)
			cs.AvgLatencyMs /= float64(cs.Count)
		}
	}

	return stats
}

// ListRecent 返回最近 n 条记录。
func (s *ABTestStore) ListRecent(n int) []*ABTestRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.records)
	if n <= 0 || n > total {
		n = total
	}

	result := make([]*ABTestRecord, n)
	copy(result, s.records[total-n:])
	// 返回最新的在前
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// Close 关闭存储（不关闭外部传入的 db）。
func (s *ABTestStore) Close() {}
