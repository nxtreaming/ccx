package presetstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultUpdaterInterval = 30 * time.Minute
	defaultShardSizeLimit  = 2 << 20
	defaultHTTPTimeout     = 10 * time.Second
)

var requiredRemoteShardKinds = []string{"subscriptionPreset", "modelRegistry", "channelPresets", "builtinManifest"}

// UpdaterConfig 控制远程预置更新器行为。
type UpdaterConfig struct {
	Enabled bool

	IndexURL string
	Interval time.Duration
	CacheDir string

	HTTPClient *http.Client

	// AllowInsecureForTesting 仅供测试允许 HTTP，生产必须保持 false。
	AllowInsecureForTesting bool
}

type PresetIndex struct {
	SchemaVersion int                `json:"schemaVersion"`
	DataVersion   string             `json:"dataVersion"`
	PublishedAt   time.Time          `json:"publishedAt"`
	Shards        []PresetIndexShard `json:"shards"`
}

type PresetIndexShard struct {
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type UpdaterStatus struct {
	Enabled       bool       `json:"enabled"`
	Running       bool       `json:"running"`
	Checking      bool       `json:"checking"`
	Source        string     `json:"source,omitempty"`
	DataVersion   string     `json:"dataVersion,omitempty"`
	LastCheckAt   *time.Time `json:"lastCheckAt,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	NextCheckAt   *time.Time `json:"nextCheckAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
	CacheValid    bool       `json:"cacheValid"`
}

type redirectPolicy struct {
	allowInsecure bool
	baseURL       *url.URL
}

type PresetUpdater struct {
	store  *PresetStore
	config UpdaterConfig

	client *http.Client

	mu            sync.RWMutex
	lifecycleMu   sync.Mutex
	checkMu       sync.Mutex
	running       bool
	checking      bool
	source        string
	dataVersion   string
	lastCheckAt   *time.Time
	lastSuccessAt *time.Time
	nextCheckAt   *time.Time
	lastError     string
	cacheValid    bool
	cancel        context.CancelFunc
	done          chan struct{}
}

func NewPresetUpdater(store *PresetStore, cfg UpdaterConfig) *PresetUpdater {
	if store == nil {
		store = NewPresetStore(nil)
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultUpdaterInterval
	}
	indexURL, err := url.Parse(cfg.IndexURL)
	if err != nil {
		indexURL = nil
	}
	client := buildUpdaterHTTPClient(cfg.HTTPClient, redirectPolicy{
		allowInsecure: cfg.AllowInsecureForTesting,
		baseURL:       indexURL,
	})

	u := &PresetUpdater{
		store:  store,
		config: cfg,
		client: client,
		source: "embedded",
	}
	u.dataVersion = store.DataVersion()
	return u
}

// LoadCacheAtStartup 尝试从磁盘缓存恢复 bundle；仅较新缓存会切换为 cache 源。
func (u *PresetUpdater) LoadCacheAtStartup() error {
	if strings.TrimSpace(u.config.CacheDir) == "" {
		return fmt.Errorf("[PresetUpdater-Cache] cacheDir 未配置")
	}
	bundle, err := LoadCache(u.config.CacheDir)
	if err != nil {
		u.setCacheValid(false)
		return err
	}
	currentVersion := u.store.DataVersion()
	if compareDataVersion(bundle.DataVersion, currentVersion) <= 0 {
		log.Printf("[PresetUpdater-Cache] 跳过非新版本缓存: current=%q cached=%q", currentVersion, bundle.DataVersion)
		u.mu.Lock()
		u.source = "embedded"
		u.dataVersion = currentVersion
		u.cacheValid = true
		u.lastError = ""
		u.mu.Unlock()
		return nil
	}
	u.store.Swap(bundle)
	u.mu.Lock()
	u.source = "cache"
	u.dataVersion = bundle.DataVersion
	u.cacheValid = true
	u.lastError = ""
	u.mu.Unlock()
	return nil
}

func (u *PresetUpdater) Start(ctx context.Context) {
	u.lifecycleMu.Lock()
	defer u.lifecycleMu.Unlock()
	if !u.config.Enabled || u.running {
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	next := time.Now().Add(u.config.Interval)
	ready := make(chan struct{})
	done := make(chan struct{})

	u.mu.Lock()
	u.running = true
	u.lastError = ""
	u.nextCheckAt = &next
	u.cancel = cancel
	u.done = done
	u.mu.Unlock()

	go u.loop(runCtx, ready, done)
	<-ready
}

func (u *PresetUpdater) Stop() {
	u.lifecycleMu.Lock()
	cancel, done, running := u.snapshotLifecycleLocked()
	if !running {
		u.lifecycleMu.Unlock()
		return
	}
	u.mu.Lock()
	u.running = false
	u.checking = false
	u.nextCheckAt = nil
	u.cancel = nil
	u.done = nil
	u.mu.Unlock()
	u.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (u *PresetUpdater) Status() UpdaterStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return UpdaterStatus{
		Enabled:       u.config.Enabled,
		Running:       u.running,
		Checking:      u.checking,
		Source:        u.source,
		DataVersion:   u.dataVersion,
		LastCheckAt:   cloneTimePtr(u.lastCheckAt),
		LastSuccessAt: cloneTimePtr(u.lastSuccessAt),
		NextCheckAt:   cloneTimePtr(u.nextCheckAt),
		LastError:     u.lastError,
		CacheValid:    u.cacheValid,
	}
}

func (u *PresetUpdater) CheckOnce(ctx context.Context) error {
	u.checkMu.Lock()
	defer u.checkMu.Unlock()

	startedAt := time.Now()
	if err := u.validateIndexURL(); err != nil {
		u.finishCheck(startedAt, err, false)
		return err
	}

	u.beginCheck()
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("[PresetUpdater-Check] panic: %v", r)
			u.finishCheck(startedAt, err, false)
			panic(r)
		}
	}()

	indexURL, err := url.Parse(u.config.IndexURL)
	if err != nil {
		err = fmt.Errorf("[PresetUpdater-Check] 解析 index URL 失败: %w", err)
		u.finishCheck(startedAt, err, false)
		return err
	}

	indexBytes, err := u.fetchBytes(ctx, indexURL.String(), 0)
	if err != nil {
		err = fmt.Errorf("[PresetUpdater-Check] 拉取 index 失败: %w", err)
		u.finishCheck(startedAt, err, false)
		return err
	}

	var index PresetIndex
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		err = fmt.Errorf("[PresetUpdater-Check] 解析 index 失败: %w", err)
		u.finishCheck(startedAt, err, false)
		return err
	}
	if index.SchemaVersion > CurrentSchemaVersion {
		err = fmt.Errorf("[PresetUpdater-Check] schemaVersion %d 高于本二进制支持的 %d", index.SchemaVersion, CurrentSchemaVersion)
		u.finishCheck(startedAt, err, false)
		return err
	}

	currentVersion := u.store.DataVersion()
	if compareDataVersion(index.DataVersion, currentVersion) <= 0 {
		log.Printf("[PresetUpdater-Check] 跳过非新版本 index: current=%q incoming=%q", currentVersion, index.DataVersion)
		u.finishCheck(startedAt, nil, false)
		return nil
	}

	bundle, err := u.buildBundleFromIndex(ctx, indexURL, &index)
	if err != nil {
		u.finishCheck(startedAt, err, false)
		return err
	}
	if err := Validate(bundle); err != nil {
		err = fmt.Errorf("[PresetUpdater-Check] 校验 bundle 失败: %w", err)
		u.finishCheck(startedAt, err, false)
		return err
	}

	currentVersion = u.store.DataVersion()
	if compareDataVersion(bundle.DataVersion, currentVersion) <= 0 {
		log.Printf("[PresetUpdater-Check] 构建完成后跳过旧版本覆盖: current=%q incoming=%q", currentVersion, bundle.DataVersion)
		u.finishCheck(startedAt, nil, false)
		return nil
	}

	if strings.TrimSpace(u.config.CacheDir) != "" {
		if err := SaveCache(u.config.CacheDir, bundle); err != nil {
			err = fmt.Errorf("[PresetUpdater-Cache] 写缓存失败: %w", err)
			u.finishCheck(startedAt, err, false)
			return err
		}
	}

	u.store.Swap(bundle)
	u.finishCheck(startedAt, nil, true)
	log.Printf("[PresetUpdater-Check] 预置更新成功: version=%q", bundle.DataVersion)
	return nil
}

func (u *PresetUpdater) loop(ctx context.Context, ready chan<- struct{}, done chan struct{}) {
	defer close(done)
	defer func() {
		u.lifecycleMu.Lock()
		defer u.lifecycleMu.Unlock()
		u.mu.Lock()
		defer u.mu.Unlock()
		if u.done == done {
			u.running = false
			u.checking = false
			u.nextCheckAt = nil
			u.cancel = nil
			u.done = nil
		}
	}()

	close(ready)
	if err := u.CheckOnce(ctx); err != nil {
		log.Printf("[PresetUpdater-Start] 首次检查失败: %v", err)
	}

	ticker := time.NewTicker(u.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			next := time.Now().Add(u.config.Interval)
			u.mu.Lock()
			u.nextCheckAt = &next
			u.mu.Unlock()
			if err := u.CheckOnce(ctx); err != nil {
				log.Printf("[PresetUpdater-Loop] 周期检查失败: %v", err)
			}
		}
	}
}

func (u *PresetUpdater) buildBundleFromIndex(ctx context.Context, indexURL *url.URL, index *PresetIndex) (*PresetBundle, error) {
	legacyOnly := true
	for _, shard := range index.Shards {
		if canonicalShardKind(shard.Kind) != "subscriptionPreset" {
			legacyOnly = false
			break
		}
	}
	if legacyOnly {
		return nil, fmt.Errorf("[PresetUpdater-Check] legacy 单 subscription shard index 已不再支持正式更新")
	}

	base := u.store.Get()
	bundle := cloneBundle(base)
	bundle.SchemaVersion = index.SchemaVersion
	bundle.DataVersion = index.DataVersion

	seenKinds := map[string]bool{}
	for _, shard := range index.Shards {
		kind := canonicalShardKind(shard.Kind)
		if !isExpectedShardKind(kind) {
			continue
		}
		if seenKinds[kind] {
			return nil, fmt.Errorf("[PresetUpdater-Check] shard kind %s 重复", kind)
		}
		seenKinds[kind] = true

		body, err := u.fetchShard(ctx, indexURL, shard, kind)
		if err != nil {
			return nil, err
		}
		if err := applyShard(body, kind, bundle); err != nil {
			return nil, err
		}
	}

	for _, kind := range requiredRemoteShardKinds {
		if !seenKinds[kind] {
			return nil, fmt.Errorf("[PresetUpdater-Check] index 缺少 %s shard", kind)
		}
	}
	if err := Validate(bundle); err != nil {
		return nil, err
	}
	return bundle, nil
}

func isExpectedShardKind(kind string) bool {
	switch kind {
	case "subscriptionPreset", "modelRegistry", "channelPresets", "builtinManifest":
		return true
	default:
		return false
	}
}

func canonicalShardKind(kind string) string {
	if kind == "subscription" {
		return "subscriptionPreset"
	}
	return kind
}

func (u *PresetUpdater) fetchShard(ctx context.Context, indexURL *url.URL, shard PresetIndexShard, kind string) ([]byte, error) {
	if strings.TrimSpace(shard.SHA256) == "" {
		return nil, fmt.Errorf("[PresetUpdater-Check] %s shard 缺少 sha256", kind)
	}
	resolved, err := indexURL.Parse(shard.URL)
	if err != nil {
		return nil, fmt.Errorf("[PresetUpdater-Check] 解析 %s shard URL 失败: %w", kind, err)
	}
	body, err := u.fetchBytes(ctx, resolved.String(), defaultShardSizeLimit)
	if err != nil {
		return nil, fmt.Errorf("[PresetUpdater-Check] 拉取 %s shard 失败: %w", kind, err)
	}
	if !strings.EqualFold(strings.TrimSpace(shard.SHA256), sha256Hex(body)) {
		return nil, fmt.Errorf("[PresetUpdater-Check] %s shard SHA256 校验失败", kind)
	}
	return body, nil
}

func applyShard(body []byte, kind string, bundle *PresetBundle) error {
	switch kind {
	case "subscriptionPreset":
		var subscription SubscriptionPreset
		if err := json.Unmarshal(body, &subscription); err != nil {
			return fmt.Errorf("[PresetUpdater-Check] 解析 subscriptionPreset shard 失败: %w", err)
		}
		bundle.Subscription = subscription
	case "modelRegistry":
		var registry ModelRegistryPreset
		if err := json.Unmarshal(body, &registry); err != nil {
			return fmt.Errorf("[PresetUpdater-Check] 解析 modelRegistry shard 失败: %w", err)
		}
		bundle.ModelRegistry = &registry
	case "channelPresets":
		var channelPresets ChannelPresetsPreset
		if err := json.Unmarshal(body, &channelPresets); err != nil {
			return fmt.Errorf("[PresetUpdater-Check] 解析 channelPresets shard 失败: %w", err)
		}
		bundle.ChannelPresets = &channelPresets
	case "builtinManifest":
		var manifest BuiltinModelsManifestPreset
		if err := json.Unmarshal(body, &manifest); err != nil {
			return fmt.Errorf("[PresetUpdater-Check] 解析 builtinManifest shard 失败: %w", err)
		}
		bundle.BuiltinModelsManifests = &manifest
	default:
		return fmt.Errorf("[PresetUpdater-Check] 未知 shard kind %s", kind)
	}
	return nil
}

func fetchBytesNoRedirect(client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}

func (u *PresetUpdater) fetchBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	if err := u.validateRemoteURL(rawURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := fetchBytesNoRedirect(u.client, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	reader := io.Reader(resp.Body)
	if limit > 0 {
		reader = io.LimitReader(resp.Body, limit+1)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if limit > 0 && int64(len(body)) > limit {
		return nil, fmt.Errorf("响应体超过上限 %d bytes", limit)
	}
	return body, nil
}

func (u *PresetUpdater) validateIndexURL() error {
	if strings.TrimSpace(u.config.IndexURL) == "" {
		return fmt.Errorf("[PresetUpdater-Config] indexURL 不能为空")
	}
	return u.validateRemoteURL(u.config.IndexURL)
}

func (u *PresetUpdater) validateRemoteURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("[PresetUpdater-Config] 非法 URL: %w", err)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && u.config.AllowInsecureForTesting {
		return nil
	}
	return fmt.Errorf("[PresetUpdater-Config] 仅允许 HTTPS URL")
}

func (u *PresetUpdater) beginCheck() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.checking = true
}

func (u *PresetUpdater) finishCheck(at time.Time, err error, updated bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.checking = false
	u.lastCheckAt = cloneTimePtr(&at)
	next := at.Add(u.config.Interval)
	u.nextCheckAt = &next
	if err != nil {
		u.lastError = err.Error()
		return
	}
	u.lastError = ""
	if updated {
		u.source = "remote"
		u.dataVersion = u.store.DataVersion()
		u.cacheValid = strings.TrimSpace(u.config.CacheDir) != ""
		u.lastSuccessAt = cloneTimePtr(&at)
	}
}

func (u *PresetUpdater) setCacheValid(valid bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.cacheValid = valid
}

func (u *PresetUpdater) snapshotLifecycleLocked() (context.CancelFunc, chan struct{}, bool) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.cancel, u.done, u.running
}

func cloneTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	v := *src
	return &v
}

func compareDataVersion(a, b string) int {
	av := parseDataVersion(strings.TrimSpace(a))
	bv := parseDataVersion(strings.TrimSpace(b))
	return av.compare(bv)
}

type dataVersion struct {
	raw     string
	parts   []int
	hasText bool
}

func parseDataVersion(raw string) dataVersion {
	if raw == "" {
		return dataVersion{}
	}
	numericVersion := raw
	if numericVersion[0] == 'v' || numericVersion[0] == 'V' {
		numericVersion = numericVersion[1:]
	}
	fields := splitVersionTokens(numericVersion)
	if len(fields) == 0 {
		return dataVersion{raw: raw, hasText: true}
	}
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil {
			return dataVersion{raw: raw, hasText: true}
		}
		parts = append(parts, value)
	}
	return dataVersion{raw: raw, parts: parts}
}

func splitVersionTokens(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '.', '-', '_', '+':
			return true
		default:
			return false
		}
	})
}

func (v dataVersion) compare(other dataVersion) int {
	switch {
	case v.raw == other.raw:
		return 0
	case v.raw == "":
		return -1
	case other.raw == "":
		return 1
	}
	if !v.hasText && !other.hasText {
		maxLen := len(v.parts)
		if len(other.parts) > maxLen {
			maxLen = len(other.parts)
		}
		for i := 0; i < maxLen; i++ {
			left := 0
			if i < len(v.parts) {
				left = v.parts[i]
			}
			right := 0
			if i < len(other.parts) {
				right = other.parts[i]
			}
			if left < right {
				return -1
			}
			if left > right {
				return 1
			}
		}
		return 0
	}
	if v.raw < other.raw {
		return -1
	}
	return 1
}

func buildUpdaterHTTPClient(base *http.Client, policy redirectPolicy) *http.Client {
	var client http.Client
	if base == nil {
		client = http.Client{Timeout: defaultHTTPTimeout}
	} else {
		client = *base
		if client.Timeout <= 0 {
			client.Timeout = defaultHTTPTimeout
		}
	}
	client.CheckRedirect = buildRedirectChecker(policy)
	return &client
}

func buildRedirectChecker(policy redirectPolicy) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		if err := validateRedirectTarget(req.URL, policy.allowInsecure); err != nil {
			return err
		}
		if policy.baseURL != nil && !sameOrigin(policy.baseURL, req.URL) {
			return fmt.Errorf("[PresetUpdater-Config] redirect 仅允许同源")
		}
		if len(via) >= 10 {
			return fmt.Errorf("[PresetUpdater-Config] redirect 次数过多")
		}
		return nil
	}
}

func validateRedirectTarget(target *url.URL, allowInsecure bool) error {
	if target == nil {
		return fmt.Errorf("[PresetUpdater-Config] redirect 目标为空")
	}
	if target.Scheme == "https" {
		return nil
	}
	if target.Scheme == "http" && allowInsecure {
		return nil
	}
	return fmt.Errorf("[PresetUpdater-Config] redirect 仅允许 HTTPS URL")
}

func sameOrigin(base *url.URL, target *url.URL) bool {
	if base == nil || target == nil {
		return false
	}
	return strings.EqualFold(base.Scheme, target.Scheme) && strings.EqualFold(normalizeHost(base), normalizeHost(target))
}

func normalizeHost(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		if strings.EqualFold(u.Scheme, "https") {
			port = "443"
		} else if strings.EqualFold(u.Scheme, "http") {
			port = "80"
		}
	}
	return net.JoinHostPort(host, port)
}

// StatusHandler 返回 GET /api/presets/status 的只读处理器。
// updater 为 nil 时返回 disabled 状态而非错误，避免预置更新功能影响服务健康。
func StatusHandler(updater *PresetUpdater) gin.HandlerFunc {
	return func(c *gin.Context) {
		if updater == nil {
			c.JSON(http.StatusOK, UpdaterStatus{Enabled: false, Source: "embedded"})
			return
		}
		c.JSON(http.StatusOK, updater.Status())
	}
}
