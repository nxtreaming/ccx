package autopilot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── 本地模型运行时画像 ──

// RuntimeType 本地运行时类型枚举。
type RuntimeType string

const (
	RuntimeTypeOllama           RuntimeType = "ollama"
	RuntimeTypeLMStudio         RuntimeType = "lmstudio"
	RuntimeTypeLlamaServer      RuntimeType = "llama_server"
	RuntimeTypeOpenAICompatible RuntimeType = "openai_compatible"
)

// LocalRuntimeStatus 运行时健康状态枚举。
type LocalRuntimeStatus string

const (
	LocalRuntimeHealthy     LocalRuntimeStatus = "healthy"
	LocalRuntimeSlow        LocalRuntimeStatus = "slow"
	LocalRuntimeUnavailable LocalRuntimeStatus = "unavailable"
	LocalRuntimeUnknown     LocalRuntimeStatus = "unknown"
)

// LocalModelRuntimeProfile 描述一个本地运行时实例。
type LocalModelRuntimeProfile struct {
	RuntimeUID  string      `json:"runtimeUid"`
	Name        string      `json:"name,omitempty"`
	RuntimeType RuntimeType `json:"runtimeType"`
	BaseURL     string      `json:"baseUrl"`

	// 探测发现的模型列表
	DiscoveredModels []string `json:"discoveredModels,omitempty"`

	// 健康状态
	Status    LocalRuntimeStatus `json:"status"`
	LatencyMs int64              `json:"latencyMs,omitempty"`

	// 本地资源提示
	ContextTokens     int     `json:"contextTokens,omitempty"`
	SupportsTools     bool    `json:"supportsTools,omitempty"`
	SupportsVision    bool    `json:"supportsVision,omitempty"`
	SupportsReasoning bool    `json:"supportsReasoning,omitempty"`
	TokensPerSecond   float64 `json:"tokensPerSecond,omitempty"`
	TimeoutMs         int     `json:"timeoutMs,omitempty"`

	// 观测
	LastProbeAt *time.Time `json:"lastProbeAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ── ollama API 响应结构 ──

// ollamaTagsResponse GET /api/tags 响应。
type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name string `json:"name"`
}

// openAIModelsResponse GET /v1/models 响应（ollama-openai / lmstudio / llama-server / openai-compatible 共用）。
type openAIModelsResponse struct {
	Data []openAIModel `json:"data"`
}

type openAIModel struct {
	ID string `json:"id"`
}

// ── LocalRuntimeStore: 内存缓存 + SQLite 持久化 ──

// LocalRuntimeStore 管理 LocalModelRuntimeProfile 的内存缓存与 SQLite 持久化。
// 与 ProfileStore 模式一致：内存为主读取源，写入双写内存+异步落盘。
type LocalRuntimeStore struct {
	db     *sql.DB
	dbPath string // 仅由 NewLocalRuntimeStore 设置，NewLocalRuntimeStoreWithDB 为空

	cache map[string]*LocalModelRuntimeProfile // key = runtimeUID
	mu    sync.RWMutex

	flushMu   sync.Mutex
	closed    bool
	dirtyKeys map[string]struct{}
}

// NewLocalRuntimeStore 创建 LocalRuntimeStore，自行管理 SQLite 连接。
func NewLocalRuntimeStore(dbPath string) (*LocalRuntimeStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("[LocalRuntimeStore-Init] 创建数据库目录失败: %w", err)
	}

	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("[LocalRuntimeStore-Init] 打开数据库失败: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store, err := newLocalRuntimeStoreFromDB(db, dbPath)
	if err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

// NewLocalRuntimeStoreWithDB 使用外部提供的 *sql.DB 创建（便于测试）。
func NewLocalRuntimeStoreWithDB(db *sql.DB) (*LocalRuntimeStore, error) {
	return newLocalRuntimeStoreFromDB(db, "")
}

func newLocalRuntimeStoreFromDB(db *sql.DB, dbPath string) (*LocalRuntimeStore, error) {
	if err := initLocalRuntimeStoreSchema(db); err != nil {
		return nil, fmt.Errorf("[LocalRuntimeStore-Init] 建表失败: %w", err)
	}

	store := &LocalRuntimeStore{
		db:        db,
		dbPath:    dbPath,
		cache:     make(map[string]*LocalModelRuntimeProfile),
		dirtyKeys: make(map[string]struct{}),
	}

	if err := store.loadAll(); err != nil {
		return nil, fmt.Errorf("[LocalRuntimeStore-Init] 加载数据失败: %w", err)
	}

	log.Printf("[LocalRuntimeStore-Init] 初始化完成，已加载 %d 条本地运行时", len(store.cache))
	return store, nil
}

// initLocalRuntimeStoreSchema 建表迁移。
func initLocalRuntimeStoreSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS autopilot_local_runtimes (
    runtime_uid   TEXT PRIMARY KEY,
    runtime_type  TEXT NOT NULL,
    base_url      TEXT NOT NULL,
    profile_json  TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_local_runtimes_type ON autopilot_local_runtimes(runtime_type);
`
	_, err := db.Exec(schema)
	return err
}

// loadAll 从 SQLite 加载全部运行时画像到内存缓存。
func (s *LocalRuntimeStore) loadAll() error {
	rows, err := s.db.Query("SELECT runtime_uid, profile_json FROM autopilot_local_runtimes")
	if err != nil {
		return err
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for rows.Next() {
		var uid string
		var profileJSON string
		if err := rows.Scan(&uid, &profileJSON); err != nil {
			log.Printf("[LocalRuntimeStore-LoadAll] 跳过损坏行: %v", err)
			continue
		}
		var profile LocalModelRuntimeProfile
		if err := json.Unmarshal([]byte(profileJSON), &profile); err != nil {
			log.Printf("[LocalRuntimeStore-LoadAll] 反序列化失败 uid=%s: %v", uid, err)
			continue
		}
		s.cache[uid] = &profile
	}
	return rows.Err()
}

// Upsert 插入或更新一条本地运行时画像。
func (s *LocalRuntimeStore) Upsert(profile *LocalModelRuntimeProfile) error {
	if profile.RuntimeUID == "" {
		return fmt.Errorf("[LocalRuntimeStore-Upsert] runtime_uid 不能为空")
	}

	s.mu.Lock()
	s.cache[profile.RuntimeUID] = profile
	s.mu.Unlock()

	s.flushMu.Lock()
	s.dirtyKeys[profile.RuntimeUID] = struct{}{}
	s.flushMu.Unlock()

	return nil
}

// Get 按 runtimeUID 从内存缓存获取画像。不存在返回 nil。
func (s *LocalRuntimeStore) Get(runtimeUID string) *LocalModelRuntimeProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.cache[runtimeUID]
	if p == nil {
		return nil
	}
	cp := *p
	if p.DiscoveredModels != nil {
		cp.DiscoveredModels = make([]string, len(p.DiscoveredModels))
		copy(cp.DiscoveredModels, p.DiscoveredModels)
	}
	if p.LastProbeAt != nil {
		t := *p.LastProbeAt
		cp.LastProbeAt = &t
	}
	return &cp
}

// ListAll 返回全部本地运行时画像副本。
func (s *LocalRuntimeStore) ListAll() []*LocalModelRuntimeProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*LocalModelRuntimeProfile, 0, len(s.cache))
	for _, p := range s.cache {
		cp := *p
		if p.DiscoveredModels != nil {
			cp.DiscoveredModels = make([]string, len(p.DiscoveredModels))
			copy(cp.DiscoveredModels, p.DiscoveredModels)
		}
		if p.LastProbeAt != nil {
			t := *p.LastProbeAt
			cp.LastProbeAt = &t
		}
		result = append(result, &cp)
	}
	return result
}

// Delete 从内存和 SQLite 删除指定本地运行时。
func (s *LocalRuntimeStore) Delete(runtimeUID string) error {
	s.mu.Lock()
	delete(s.cache, runtimeUID)
	s.mu.Unlock()

	s.flushMu.Lock()
	delete(s.dirtyKeys, runtimeUID)
	s.flushMu.Unlock()

	if _, err := s.db.Exec("DELETE FROM autopilot_local_runtimes WHERE runtime_uid = ?", runtimeUID); err != nil {
		return fmt.Errorf("[LocalRuntimeStore-Delete] 删除失败 uid=%s: %w", runtimeUID, err)
	}
	return nil
}

// Flush 将内存中标记为 dirty 的画像批量写入 SQLite。
func (s *LocalRuntimeStore) Flush() error {
	s.flushMu.Lock()
	if len(s.dirtyKeys) == 0 {
		s.flushMu.Unlock()
		return nil
	}
	keys := make([]string, 0, len(s.dirtyKeys))
	for k := range s.dirtyKeys {
		keys = append(keys, k)
	}
	s.dirtyKeys = make(map[string]struct{})
	s.flushMu.Unlock()

	s.mu.RLock()
	profiles := make([]*LocalModelRuntimeProfile, 0, len(keys))
	for _, k := range keys {
		if p, ok := s.cache[k]; ok {
			profiles = append(profiles, p)
		}
	}
	s.mu.RUnlock()

	if len(profiles) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("[LocalRuntimeStore-Flush] 开启事务失败: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
INSERT INTO autopilot_local_runtimes (runtime_uid, runtime_type, base_url, profile_json, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(runtime_uid) DO UPDATE SET
    runtime_type = excluded.runtime_type,
    base_url = excluded.base_url,
    profile_json = excluded.profile_json,
    updated_at = excluded.updated_at
`)
	if err != nil {
		return fmt.Errorf("[LocalRuntimeStore-Flush] 准备语句失败: %w", err)
	}
	defer stmt.Close()

	for _, p := range profiles {
		profileJSON, err := json.Marshal(p)
		if err != nil {
			log.Printf("[LocalRuntimeStore-Flush] 序列化失败 uid=%s: %v", p.RuntimeUID, err)
			continue
		}
		_, err = stmt.Exec(
			p.RuntimeUID,
			string(p.RuntimeType),
			p.BaseURL,
			string(profileJSON),
			time.Now().UTC().Format(time.RFC3339),
		)
		if err != nil {
			log.Printf("[LocalRuntimeStore-Flush] 写入失败 uid=%s: %v", p.RuntimeUID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("[LocalRuntimeStore-Flush] 提交事务失败: %w", err)
	}

	log.Printf("[LocalRuntimeStore-Flush] 已落盘 %d 条本地运行时", len(profiles))
	return nil
}

// Close 关闭 LocalRuntimeStore。先 Flush 剩余 dirty 数据。
func (s *LocalRuntimeStore) Close() error {
	if err := s.Flush(); err != nil {
		log.Printf("[LocalRuntimeStore-Close] Flush 失败: %v", err)
	}
	if s.dbPath != "" {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("[LocalRuntimeStore-Close] 关闭数据库失败: %w", err)
		}
	}
	return nil
}

// ── GenerateRuntimeUID 生成运行时唯一 ID ──

// GenerateRuntimeUID 生成 lr_ 前缀的运行时唯一标识。
func GenerateRuntimeUID() string {
	return "lr_" + uuid.New().String()[:8]
}

// ── ProbeRuntime 探测本地运行时 ──

const probeTimeout = 5 * time.Second

// ProbeRuntime 探测本地运行时的连通性和模型列表。
// 按 runtimeType 选择探测接口：
//   - ollama: GET /api/tags
//   - lmstudio / llama_server / openai_compatible: GET /v1/models
//
// 更新 profile 的 Status、DiscoveredModels、LatencyMs、LastProbeAt。
func ProbeRuntime(ctx context.Context, profile *LocalModelRuntimeProfile) error {
	if profile.BaseURL == "" {
		return fmt.Errorf("[LocalRuntime-Probe] baseUrl 不能为空")
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	var endpoint string
	if profile.RuntimeType == RuntimeTypeOllama {
		endpoint = strings.TrimRight(profile.BaseURL, "/") + "/api/tags"
	} else {
		endpoint = strings.TrimRight(profile.BaseURL, "/") + "/v1/models"
	}

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("[LocalRuntime-Probe] 构造请求失败: %w", err)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start)

	now := time.Now()
	profile.LastProbeAt = &now
	profile.LatencyMs = latency.Milliseconds()

	if err != nil {
		profile.Status = LocalRuntimeUnavailable
		profile.DiscoveredModels = nil
		return fmt.Errorf("[LocalRuntime-Probe] 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		profile.Status = LocalRuntimeUnavailable
		profile.DiscoveredModels = nil
		return fmt.Errorf("[LocalRuntime-Probe] 非预期状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		profile.Status = LocalRuntimeUnavailable
		profile.DiscoveredModels = nil
		return fmt.Errorf("[LocalRuntime-Probe] 读取响应失败: %w", err)
	}

	// 解析模型列表
	var models []string
	if profile.RuntimeType == RuntimeTypeOllama {
		var tagsResp ollamaTagsResponse
		if err := json.Unmarshal(body, &tagsResp); err != nil {
			profile.Status = LocalRuntimeUnavailable
			profile.DiscoveredModels = nil
			return fmt.Errorf("[LocalRuntime-Probe] 解析 ollama /api/tags 失败: %w", err)
		}
		for _, m := range tagsResp.Models {
			models = append(models, m.Name)
		}
	} else {
		var modelsResp openAIModelsResponse
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			profile.Status = LocalRuntimeUnavailable
			profile.DiscoveredModels = nil
			return fmt.Errorf("[LocalRuntime-Probe] 解析 /v1/models 失败: %w", err)
		}
		for _, m := range modelsResp.Data {
			models = append(models, m.ID)
		}
	}

	profile.DiscoveredModels = models

	// 判断健康状态：>2s 视为 slow
	if latency > 2*time.Second {
		profile.Status = LocalRuntimeSlow
	} else {
		profile.Status = LocalRuntimeHealthy
	}

	log.Printf("[LocalRuntime-Probe] 探测成功 runtime=%s type=%s models=%d latency=%dms",
		profile.RuntimeUID, profile.RuntimeType, len(models), profile.LatencyMs)

	return nil
}

// ValidRuntimeTypes 返回所有合法的 RuntimeType 值。
func ValidRuntimeTypes() []RuntimeType {
	return []RuntimeType{
		RuntimeTypeOllama,
		RuntimeTypeLMStudio,
		RuntimeTypeLlamaServer,
		RuntimeTypeOpenAICompatible,
	}
}

// IsValidRuntimeType 检查给定类型是否合法。
func IsValidRuntimeType(t RuntimeType) bool {
	switch t {
	case RuntimeTypeOllama, RuntimeTypeLMStudio, RuntimeTypeLlamaServer, RuntimeTypeOpenAICompatible:
		return true
	default:
		return false
	}
}
