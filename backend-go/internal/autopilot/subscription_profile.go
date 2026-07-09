package autopilot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// SubscriptionProfile 描述渠道背后的套餐/余额/价格来源。
// Phase 1 仅手动维护，不做余额自动抓取。
type SubscriptionProfile struct {
	SubscriptionUID string `json:"subscriptionUid"`
	DisplayName     string `json:"displayName"`
	Provider        string `json:"provider"` // openai | anthropic | google | relay_x | community_x | custom
	OriginType      string `json:"originType"`
	OriginTier      string `json:"originTier"`

	BillingMode string  `json:"billingMode"` // official_api | token_plan | prepaid_credit | shared_free | unknown
	Currency    string  `json:"currency,omitempty"`
	Balance     float64 `json:"balance,omitempty"`

	// 套餐默认成本倍率；channel/key 可继续覆盖。
	GroupMultipliers   map[string]float64 `json:"groupMultipliers,omitempty"`
	RechargeMultiplier float64            `json:"rechargeMultiplier,omitempty"`

	LinkedChannelUIDs []string `json:"linkedChannelUids,omitempty"`
	Source            string   `json:"source"` // manual | imported | inferred
	Confidence        float64  `json:"confidence"`

	// ── Phase 4 Item 6：余额自动刷新 ──
	// BillingAPIKey 用于查询 provider 账单/余额的专用密钥（与推理 APIKeys 分离）。
	// 很多 provider 的账单 API 需要 admin/org 级密钥而非普通 API key，不能假设两者通用。
	// 未填写则该订阅不参与自动刷新，静默跳过。
	BillingAPIKey string `json:"billingApiKey,omitempty"`

	// AutoRefreshEnabled 单订阅级开关。即使全局 SubscriptionAutoRefresh.Enabled=true，
	// 该订阅也必须 AutoRefreshEnabled=true 且 BillingAPIKey 非空才纳入刷新队列。
	AutoRefreshEnabled bool `json:"autoRefreshEnabled,omitempty"`

	// LastBalanceRefreshAt 最近一次成功刷新余额的时间。
	LastBalanceRefreshAt *time.Time `json:"lastBalanceRefreshAt,omitempty"`

	// LastBalanceRefreshError 最近一次刷新失败的错误信息（成功后清空）。
	LastBalanceRefreshError string `json:"lastBalanceRefreshError,omitempty"`

	// ── 订阅级共享能力（§3.2.3，shadow 展示）──
	SharedCapability *SharedCapability `json:"sharedCapability,omitempty"` // 从同订阅 endpoint 聚合的共享能力

	// ── 订阅级用量窗口（§3.2.4）──
	UsageWindows []UsageWindow `json:"usageWindows,omitempty"` // 订阅级汇总用量窗口

	Notes      string     `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
}

// SharedCapability 描述同订阅下所有 endpoint 共享的能力画像。
// 由 BuildSharedCapability 从 endpoint 画像聚合（多数派投票），存储在 SubscriptionProfile 中。
// endpoint 画像通过引用继承，无需重复探测。
type SharedCapability struct {
	// ── 模型列表 ──
	ModelListHash string   `json:"modelListHash"` // 模型列表哈希（排序后 SHA-256）
	ModelList     []string `json:"modelList"`     // 多数派模型列表快照

	// ── 能力标签（多数派投票结果）──
	SupportsVision    bool `json:"supportsVision"`
	SupportsToolCalls bool `json:"supportsToolCalls"`
	SupportsReasoning bool `json:"supportsReasoning"`

	// ── 协议兼容开关快照 ──
	SupportsStreaming  bool `json:"supportsStreaming"`  // 流式支持
	SupportsLongCtx    bool `json:"supportsLongCtx"`    // 长上下文支持
	SupportsMultiModal bool `json:"supportsMultiModal"` // 多模态支持

	// ── 统计 ──
	TotalEndpoints   int      `json:"totalEndpoints"`             // 参与聚合的 endpoint 总数
	ConsistentCount  int      `json:"consistentCount"`            // 与共享能力一致的 endpoint 数
	InconsistentKeys []string `json:"inconsistentKeys,omitempty"` // 与共享能力不一致的 endpointUID 列表

	ProbedAt time.Time `json:"probedAt"` // 最近一次聚合计算时间
}

// SubscriptionStore 管理 SubscriptionProfile 的内存缓存与 SQLite 持久化。
// 复用 ProfileStore 的 SQLite 连接模式：接收 *sql.DB，自建表，JSON 列存本体。
type SubscriptionStore struct {
	db     *sql.DB
	dbPath string // 自管连接时非空；外部传入时为空

	cache map[string]*SubscriptionProfile // key = subscriptionUID
	mu    sync.RWMutex
}

// NewSubscriptionStore 创建 SubscriptionStore，自行管理 SQLite 连接。
// dbPath 为数据库文件路径，启动时自动建表并 loadAll 回内存。
func NewSubscriptionStore(dbPath string) (*SubscriptionStore, error) {
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("[SubscriptionStore-Init] 打开数据库失败: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store, err := newSubscriptionStoreFromDB(db, dbPath)
	if err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

// NewSubscriptionStoreWithDB 使用外部 *sql.DB 创建 SubscriptionStore（便于测试/复用连接）。
// 调用方负责 db 的生命周期管理；Close() 不会关闭该 db。
func NewSubscriptionStoreWithDB(db *sql.DB) (*SubscriptionStore, error) {
	return newSubscriptionStoreFromDB(db, "")
}

func newSubscriptionStoreFromDB(db *sql.DB, dbPath string) (*SubscriptionStore, error) {
	if err := initSubscriptionStoreSchema(db); err != nil {
		return nil, fmt.Errorf("[SubscriptionStore-Init] 建表失败: %w", err)
	}

	store := &SubscriptionStore{
		db:     db,
		dbPath: dbPath,
		cache:  make(map[string]*SubscriptionProfile),
	}

	if err := store.loadAll(); err != nil {
		return nil, fmt.Errorf("[SubscriptionStore-Init] 加载订阅画像失败: %w", err)
	}

	log.Printf("[SubscriptionStore-Init] 初始化完成，已加载 %d 条订阅画像", len(store.cache))
	return store, nil
}

// initSubscriptionStoreSchema 建表迁移。
func initSubscriptionStoreSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS autopilot_subscriptions (
    subscription_uid  TEXT PRIMARY KEY,
    profile_json      TEXT NOT NULL,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);
`
	_, err := db.Exec(schema)
	return err
}

// loadAll 从 SQLite 加载全部订阅画像到内存缓存。
func (s *SubscriptionStore) loadAll() error {
	rows, err := s.db.Query("SELECT subscription_uid, profile_json FROM autopilot_subscriptions")
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
			log.Printf("[SubscriptionStore-LoadAll] 跳过损坏行: %v", err)
			continue
		}
		var profile SubscriptionProfile
		if err := json.Unmarshal([]byte(profileJSON), &profile); err != nil {
			log.Printf("[SubscriptionStore-LoadAll] 反序列化失败 uid=%s: %v", uid, err)
			continue
		}
		s.cache[uid] = &profile
	}
	return rows.Err()
}

// ── CRUD ──

// Create 创建一条订阅画像。SubscriptionUID 不能为空且不能已存在。
func (s *SubscriptionStore) Create(profile *SubscriptionProfile) error {
	if profile.SubscriptionUID == "" {
		return fmt.Errorf("[SubscriptionStore-Create] subscription_uid 不能为空")
	}

	s.mu.RLock()
	_, exists := s.cache[profile.SubscriptionUID]
	s.mu.RUnlock()
	if exists {
		return fmt.Errorf("[SubscriptionStore-Create] subscription_uid=%s 已存在", profile.SubscriptionUID)
	}

	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	s.mu.Lock()
	s.cache[profile.SubscriptionUID] = profile
	s.mu.Unlock()

	return s.persist(profile)
}

// Get 按 subscriptionUID 从内存缓存获取画像。不存在返回 nil。
func (s *SubscriptionStore) Get(subscriptionUID string) *SubscriptionProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.cache[subscriptionUID]
	if p == nil {
		return nil
	}
	cp := *p
	// map 需要深拷贝
	if p.GroupMultipliers != nil {
		cp.GroupMultipliers = make(map[string]float64, len(p.GroupMultipliers))
		for k, v := range p.GroupMultipliers {
			cp.GroupMultipliers[k] = v
		}
	}
	return &cp
}

// Update 更新一条已存在的订阅画像（按 SubscriptionUID 匹配）。
func (s *SubscriptionStore) Update(profile *SubscriptionProfile) error {
	if profile.SubscriptionUID == "" {
		return fmt.Errorf("[SubscriptionStore-Update] subscription_uid 不能为空")
	}

	s.mu.Lock()
	existing, ok := s.cache[profile.SubscriptionUID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("[SubscriptionStore-Update] subscription_uid=%s 不存在", profile.SubscriptionUID)
	}
	// 保留创建时间
	profile.CreatedAt = existing.CreatedAt
	profile.UpdatedAt = time.Now()
	s.cache[profile.SubscriptionUID] = profile
	s.mu.Unlock()

	return s.persist(profile)
}

// Delete 从内存和 SQLite 删除指定订阅画像。
func (s *SubscriptionStore) Delete(subscriptionUID string) error {
	s.mu.Lock()
	delete(s.cache, subscriptionUID)
	s.mu.Unlock()

	if _, err := s.db.Exec("DELETE FROM autopilot_subscriptions WHERE subscription_uid = ?", subscriptionUID); err != nil {
		return fmt.Errorf("[SubscriptionStore-Delete] 删除失败 uid=%s: %w", subscriptionUID, err)
	}
	return nil
}

// ListAll 返回全部订阅画像副本。
func (s *SubscriptionStore) ListAll() []*SubscriptionProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*SubscriptionProfile, 0, len(s.cache))
	for _, p := range s.cache {
		cp := *p
		if p.GroupMultipliers != nil {
			cp.GroupMultipliers = make(map[string]float64, len(p.GroupMultipliers))
			for k, v := range p.GroupMultipliers {
				cp.GroupMultipliers[k] = v
			}
		}
		result = append(result, &cp)
	}
	return result
}

// ── 渠道链接 ──

// LinkChannel 将 channelUID 关联到指定订阅。幂等操作，已链接则跳过。
func (s *SubscriptionStore) LinkChannel(subscriptionUID, channelUID string) error {
	s.mu.Lock()
	p, ok := s.cache[subscriptionUID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("[SubscriptionStore-LinkChannel] subscription_uid=%s 不存在", subscriptionUID)
	}

	// 检查是否已链接
	for _, uid := range p.LinkedChannelUIDs {
		if uid == channelUID {
			s.mu.Unlock()
			return nil // 幂等
		}
	}
	p.LinkedChannelUIDs = append(p.LinkedChannelUIDs, channelUID)
	p.UpdatedAt = time.Now()
	s.mu.Unlock()

	return s.persist(p)
}

// UnlinkChannel 从指定订阅解绑 channelUID。幂等操作，未链接则跳过。
func (s *SubscriptionStore) UnlinkChannel(subscriptionUID, channelUID string) error {
	s.mu.Lock()
	p, ok := s.cache[subscriptionUID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("[SubscriptionStore-UnlinkChannel] subscription_uid=%s 不存在", subscriptionUID)
	}

	filtered := make([]string, 0, len(p.LinkedChannelUIDs))
	found := false
	for _, uid := range p.LinkedChannelUIDs {
		if uid == channelUID {
			found = true
			continue
		}
		filtered = append(filtered, uid)
	}
	if !found {
		s.mu.Unlock()
		return nil // 幂等
	}
	p.LinkedChannelUIDs = filtered
	p.UpdatedAt = time.Now()
	s.mu.Unlock()

	return s.persist(p)
}

// ── 内部辅助 ──

// persist 将单条订阅画像写入 SQLite（upsert）。
func (s *SubscriptionStore) persist(profile *SubscriptionProfile) error {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("[SubscriptionStore-Persist] 序列化失败 uid=%s: %w", profile.SubscriptionUID, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
INSERT INTO autopilot_subscriptions (subscription_uid, profile_json, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(subscription_uid) DO UPDATE SET
    profile_json = excluded.profile_json,
    updated_at = excluded.updated_at
`, profile.SubscriptionUID, string(profileJSON), now, now)

	if err != nil {
		return fmt.Errorf("[SubscriptionStore-Persist] 写入失败 uid=%s: %w", profile.SubscriptionUID, err)
	}
	return nil
}

// Close 关闭 SubscriptionStore。
// 仅自管连接（NewSubscriptionStore）会关闭 db；NewSubscriptionStoreWithDB 不关闭。
func (s *SubscriptionStore) Close() error {
	if s.dbPath != "" {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("[SubscriptionStore-Close] 关闭数据库失败: %w", err)
		}
	}
	return nil
}
