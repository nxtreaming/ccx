package autopilot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── OutputMode 枚举 ──

// OutputMode 描述模板的输出语义（元数据，供前端展示和用户区分模板用途）。
type OutputMode string

const (
	OutputModeSummarize OutputMode = "summarize" // 摘要/压缩
	OutputModeRewrite   OutputMode = "rewrite"   // 改写/重述
	OutputModeExtract   OutputMode = "extract"   // 提取关键信息
)

// ValidOutputModes 返回所有合法 OutputMode。
func ValidOutputModes() []OutputMode {
	return []OutputMode{OutputModeSummarize, OutputModeRewrite, OutputModeExtract}
}

// IsValidOutputMode 检查给定值是否合法。
func IsValidOutputMode(m OutputMode) bool {
	switch m {
	case OutputModeSummarize, OutputModeRewrite, OutputModeExtract:
		return true
	default:
		return false
	}
}

// ── LocalTaskTemplate 数据类型 ──

// LocalTaskTemplate 描述一个本地任务模板。
// 模板匹配条件为 TaskClass×TaskDomain；空数组表示通配。
// PromptTemplate 支持基础变量替换：{{original_content}} 替换为会话 transcript。
type LocalTaskTemplate struct {
	TemplateID string `json:"templateId"`

	// 展示
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	OutputMode  OutputMode `json:"outputMode"`

	// 匹配条件：空数组=通配；多值=任一命中
	MatchTaskClasses []string `json:"matchTaskClasses,omitempty"`
	MatchDomains     []string `json:"matchDomains,omitempty"`

	// 模板内容：{{original_content}} 被替换为会话 transcript
	PromptTemplate string `json:"promptTemplate"`

	// 排序：优先级越高的模板优先匹配（同优先级按创建时间降序）
	Priority int `json:"priority"`

	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GenerateTemplateID 生成 lt_ 前缀的模板唯一标识。
func GenerateTemplateID() string {
	return "lt_" + uuid.New().String()[:8]
}

// ── LocalTaskTemplateStore ──

// LocalTaskTemplateStore 管理 LocalTaskTemplate 的内存缓存与 SQLite 持久化。
// 复用 ProfileStore 的 SQLite 连接模式。
type LocalTaskTemplateStore struct {
	db     *sql.DB
	dbPath string

	cache map[string]*LocalTaskTemplate // key = templateID
	mu    sync.RWMutex
}

// NewLocalTaskTemplateStoreWithDB 使用外部提供的 *sql.DB 创建（便于测试和共享连接）。
func NewLocalTaskTemplateStoreWithDB(db *sql.DB) (*LocalTaskTemplateStore, error) {
	if err := initLocalTaskTemplateSchema(db); err != nil {
		return nil, fmt.Errorf("[TaskTemplateStore-Init] 建表失败: %w", err)
	}

	store := &LocalTaskTemplateStore{
		db:    db,
		cache: make(map[string]*LocalTaskTemplate),
	}

	if err := store.loadAll(); err != nil {
		return nil, fmt.Errorf("[TaskTemplateStore-Init] 加载数据失败: %w", err)
	}

	log.Printf("[TaskTemplateStore-Init] 初始化完成，已加载 %d 条本地任务模板", len(store.cache))
	return store, nil
}

// initLocalTaskTemplateSchema 建表迁移。
func initLocalTaskTemplateSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS autopilot_local_task_templates (
    template_id    TEXT PRIMARY KEY,
    profile_json   TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);
`
	_, err := db.Exec(schema)
	return err
}

// PLACEHOLDER_LOADALL

// loadAll 从 SQLite 加载全部模板到内存缓存。
func (s *LocalTaskTemplateStore) loadAll() error {
	rows, err := s.db.Query("SELECT template_id, profile_json FROM autopilot_local_task_templates")
	if err != nil {
		return err
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for rows.Next() {
		var id string
		var profileJSON string
		if err := rows.Scan(&id, &profileJSON); err != nil {
			log.Printf("[TaskTemplateStore-LoadAll] 跳过损坏行: %v", err)
			continue
		}
		var t LocalTaskTemplate
		if err := json.Unmarshal([]byte(profileJSON), &t); err != nil {
			log.Printf("[TaskTemplateStore-LoadAll] 反序列化失败 id=%s: %v", id, err)
			continue
		}
		s.cache[id] = &t
	}
	return rows.Err()
}

// ── CRUD 操作 ──

// ListAll 返回全部模板副本。
func (s *LocalTaskTemplateStore) ListAll() []*LocalTaskTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*LocalTaskTemplate, 0, len(s.cache))
	for _, t := range s.cache {
		cp := *t
		cp.MatchTaskClasses = copyStrings(t.MatchTaskClasses)
		cp.MatchDomains = copyStrings(t.MatchDomains)
		result = append(result, &cp)
	}
	return result
}

// Get 按 templateID 获取模板副本。不存在返回 nil。
func (s *LocalTaskTemplateStore) Get(templateID string) *LocalTaskTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t := s.cache[templateID]
	if t == nil {
		return nil
	}
	cp := *t
	cp.MatchTaskClasses = copyStrings(t.MatchTaskClasses)
	cp.MatchDomains = copyStrings(t.MatchDomains)
	return &cp
}

// PLACEHOLDER_CRUD

// Upsert 插入或更新模板，并同步写入 SQLite。
func (s *LocalTaskTemplateStore) Upsert(t *LocalTaskTemplate) error {
	if t.TemplateID == "" {
		return fmt.Errorf("[TaskTemplateStore-Upsert] template_id 不能为空")
	}

	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	s.mu.Lock()
	s.cache[t.TemplateID] = t
	s.mu.Unlock()

	return s.persist(t)
}

// Delete 删除模板。
func (s *LocalTaskTemplateStore) Delete(templateID string) error {
	s.mu.Lock()
	delete(s.cache, templateID)
	s.mu.Unlock()

	if _, err := s.db.Exec("DELETE FROM autopilot_local_task_templates WHERE template_id = ?", templateID); err != nil {
		return fmt.Errorf("[TaskTemplateStore-Delete] 删除失败 id=%s: %w", templateID, err)
	}
	return nil
}

// persist 单条模板写入 SQLite（UPSERT）。
func (s *LocalTaskTemplateStore) persist(t *LocalTaskTemplate) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("[TaskTemplateStore-Persist] 序列化失败 id=%s: %w", t.TemplateID, err)
	}

	_, err = s.db.Exec(`
INSERT INTO autopilot_local_task_templates (template_id, profile_json, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(template_id) DO UPDATE SET
    profile_json = excluded.profile_json,
    updated_at = excluded.updated_at
`, t.TemplateID, string(data), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("[TaskTemplateStore-Persist] 写入失败 id=%s: %w", t.TemplateID, err)
	}
	return nil
}

// ── 模板匹配 ──

// FindBestPrompt 根据 taskClass 和 domain 查找最匹配的模板，返回解析后的提示词。
// 匹配规则：
//   - Enabled=false 的模板跳过
//   - MatchTaskClasses 为空=通配所有 TaskClass；否则需包含 taskClass
//   - MatchDomains 为空=通配所有 domain；否则需包含 domain
//   - 同优先级按创建时间降序取第一个
//   - 无匹配返回 ""
func (s *LocalTaskTemplateStore) FindBestPrompt(taskClass, domain, transcript string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type candidate struct {
		t        *LocalTaskTemplate
		createdAt time.Time
	}

	var best *candidate
	for _, t := range s.cache {
		if !t.Enabled {
			continue
		}
		if !matchesFilter(t.MatchTaskClasses, taskClass) {
			continue
		}
		if !matchesFilter(t.MatchDomains, domain) {
			continue
		}
		if best == nil || t.Priority > best.t.Priority ||
			(t.Priority == best.t.Priority && t.CreatedAt.After(best.createdAt)) {
			best = &candidate{t: t, createdAt: t.CreatedAt}
		}
	}

	if best == nil {
		return ""
	}

	// 变量替换
	prompt := best.t.PromptTemplate
	prompt = strings.ReplaceAll(prompt, "{{original_content}}", transcript)
	return prompt
}

// matchesFilter 检查 filter 是否包含 value。
// 空 filter 视为通配（匹配任意 value）。
// 空 value 视为未指定（只有空 filter 才匹配）。
func matchesFilter(filter []string, value string) bool {
	if len(filter) == 0 {
		return true // 通配
	}
	if value == "" {
		return false // 非通配 filter 无法匹配空值
	}
	lower := strings.ToLower(value)
	for _, f := range filter {
		if strings.ToLower(f) == lower {
			return true
		}
	}
	return false
}

// copyStrings 深拷贝字符串 slice。
func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	return cp
}

// PLACEHOLDER_VALIDATE

// Validate 检查模板字段合法性。
func (t *LocalTaskTemplate) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("name 不能为空")
	}
	if t.PromptTemplate == "" {
		return fmt.Errorf("promptTemplate 不能为空")
	}
	if !IsValidOutputMode(t.OutputMode) {
		return fmt.Errorf("outputMode 无效，合法值: %v", ValidOutputModes())
	}
	if t.Priority < 0 {
		return fmt.Errorf("priority 不能为负数")
	}
	// 校验 TaskClass 枚举
	for _, tc := range t.MatchTaskClasses {
		if !isValidTaskClassString(tc) {
			return fmt.Errorf("matchTaskClasses 包含无效值: %q，合法值: %v", tc, allTaskClassStrings())
		}
	}
	// 校验 TaskDomain 枚举
	for _, d := range t.MatchDomains {
		if !isValidTaskDomainString(d) {
			return fmt.Errorf("matchDomains 包含无效值: %q，合法值: %v", d, allTaskDomainStrings())
		}
	}
	return nil
}

// isValidTaskClassString 检查字符串是否为合法 TaskClass。
func isValidTaskClassString(s string) bool {
	switch TaskClass(strings.ToLower(strings.TrimSpace(s))) {
	case TaskClassSupervisor, TaskClassWorker, TaskClassLightweight,
		TaskClassVision, TaskClassLongContext, TaskClassImageGen, TaskClassEmbedding:
		return true
	default:
		return false
	}
}

// isValidTaskDomainString 检查字符串是否为合法 TaskDomain。
func isValidTaskDomainString(s string) bool {
	switch TaskDomain(strings.ToLower(strings.TrimSpace(s))) {
	case TaskDomainAestheticsUI, TaskDomainCodeReview, TaskDomainCoding,
		TaskDomainReasoning, TaskDomainWriting, TaskDomainTranslation,
		TaskDomainAgentic, TaskDomainGeneral:
		return true
	default:
		return false
	}
}

func allTaskClassStrings() []string {
	out := make([]string, 0, len(AllTaskClasses()))
	for _, tc := range AllTaskClasses() {
		out = append(out, string(tc))
	}
	return out
}

func allTaskDomainStrings() []string {
	out := make([]string, 0, len(AllTaskDomains()))
	for _, d := range AllTaskDomains() {
		out = append(out, string(d))
	}
	return out
}
