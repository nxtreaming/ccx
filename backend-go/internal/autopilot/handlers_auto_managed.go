package autopilot

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

// ─── 请求/响应类型 ────────────────────────────────────────────────────────────────────

// AutoAddRequest POST /:kind/channels/auto-add 请求体。
//
// 两种模式：
//   - provider 模板模式：带 ProviderID + APIKeys，系统自动判别 baseURL（按 key 前缀探测验证），无需填 BaseURLs
//   - 自定义模式：带 BaseURLs + APIKeys，保持原有行为
type AutoAddRequest struct {
	Name            string   `json:"name,omitempty"`
	ProviderID      string   `json:"providerId,omitempty"`
	BaseURLs        []string `json:"baseUrls"`
	APIKeys         []string `json:"apiKeys"`
	SubscriptionUID string   `json:"subscriptionUid,omitempty"`
}

// AutoAddResponse POST /:kind/channels/auto-add 响应体。
type AutoAddResponse struct {
	ChannelUID       string                 `json:"channelUid"`
	Index            int                    `json:"index"`
	DiscoveryStarted bool                   `json:"discoveryStarted"`
	Channels         []AutoAddChannelResult `json:"channels,omitempty"`
}

// AutoAddChannelResult 描述 provider 快速添加一次创建出的单条渠道。
type AutoAddChannelResult struct {
	ChannelKind      string `json:"channelKind"`
	ChannelUID       string `json:"channelUid"`
	Index            int    `json:"index"`
	Name             string `json:"name"`
	ServiceType      string `json:"serviceType"`
	DiscoveryStarted bool   `json:"discoveryStarted"`
}

// AutoStatusResponse GET /:kind/channels/:id/auto-status 响应体。
type AutoStatusResponse struct {
	AutoManaged   bool                 `json:"autoManaged"`
	AutoManagedAt *time.Time           `json:"autoManagedAt,omitempty"`
	Discovery     *DiscoveryStatusInfo `json:"discovery,omitempty"`
}

// DiscoveryStatusInfo 发现状态信息。
type DiscoveryStatusInfo struct {
	Status     DiscoveryStatus         `json:"status"`
	StartedAt  *time.Time              `json:"startedAt,omitempty"`
	FinishedAt *time.Time              `json:"finishedAt,omitempty"`
	Error      string                  `json:"error,omitempty"`
	Endpoints  []EndpointDiscoveryInfo `json:"endpoints"`
}

// EndpointDiscoveryInfo 端点发现结果（key 已掩码）。
type EndpointDiscoveryInfo struct {
	KeyMask     string `json:"keyMask"`
	BaseURL     string `json:"baseUrl"`
	ModelsCount int    `json:"modelsCount"`
	ProtocolOk  bool   `json:"protocolOk"`
}

// ─── 路由注册 ─────────────────────────────────────────────────────────────────────────

// AutoManagedDeps 自动托管路由的依赖注入。
type AutoManagedDeps struct {
	CfgManager *config.ConfigManager
	Runner     *AutoDiscoveryRunner
}

// RegisterAutoManagedRoutes 注册自动托管 API 路由。
// 路由直接挂载到 apiGroup（不创建子组），与现有渠道管理路由共存。
//
// 注意：必须为每个 kind 显式注册静态路径，不能用 `:kind` 参数，
// 否则会与现有的 `/messages/channels/...` 等静态路由在 Gin radix tree 中冲突。
func RegisterAutoManagedRoutes(apiGroup *gin.RouterGroup, deps *AutoManagedDeps) {
	kinds := []string{"messages", "chat", "responses", "gemini", "images", "vectors"}
	for _, kind := range kinds {
		apiGroup.POST("/"+kind+"/channels/auto-add", handleAutoAdd(deps))
		apiGroup.POST("/"+kind+"/channels/:id/auto-discover", handleAutoDiscover(deps))
		apiGroup.GET("/"+kind+"/channels/:id/auto-status", handleAutoStatus(deps))
	}
}

// validChannelKinds 合法的渠道类型集合。
var validChannelKinds = map[string]bool{
	"messages":  true,
	"chat":      true,
	"responses": true,
	"gemini":    true,
	"images":    true,
	"vectors":   true,
}

// ─── 辅助函数 ─────────────────────────────────────────────────────────────────────────

// extractKindFromPath 从请求路径中提取 kind。
// 路径格式：/api/{kind}/channels/...
func extractKindFromPath(c *gin.Context) string {
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// 路径格式：api/{kind}/channels/...
	// parts[0]="api", parts[1]=kind
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ─── 处理函数 ─────────────────────────────────────────────────────────────────────────

// handleAutoAdd POST /{kind}/channels/auto-add
// 创建自动托管渠道并异步触发发现。
func handleAutoAdd(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := extractKindFromPath(c)
		if !validChannelKinds[kind] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的渠道类型: %s", kind)})
			return
		}

		var req AutoAddRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		if len(req.APIKeys) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "apiKeys 不能为空"})
			return
		}
		// provider 模板模式无需 baseUrls（由模板判定）；自定义模式必须带 baseUrls
		if req.ProviderID == "" && len(req.BaseURLs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "baseUrls 不能为空"})
			return
		}

		if req.ProviderID != "" {
			handleProviderAutoAdd(c, deps, kind, req)
			return
		}

		// 推导 serviceType
		serviceType := kindToDefaultServiceType(kind)
		name := req.Name
		if name == "" {
			name = fmt.Sprintf("auto-%s-%d", kind, time.Now().UnixMilli()%100000)
		}

		// 构建 UpstreamConfig
		now := time.Now()
		upstream := config.UpstreamConfig{
			Name:          name,
			ChannelUID:    config.GenerateChannelUID(), // 预分配 channelUID，避免竞态
			ServiceType:   serviceType,
			Status:        "active",
			AutoManaged:   true,
			AutoManagedAt: &now,
		}

		// 自定义模式：保持原有行为
		upstream.BaseURL = req.BaseURLs[0]
		upstream.BaseURLs = req.BaseURLs
		upstream.APIKeys = req.APIKeys

		// 调用对应类型的 Add 方法
		if err := addUpstreamByKind(deps.CfgManager, kind, upstream); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建渠道失败: %v", err)})
			return
		}

		// 获取创建后的 channelUid 和 index
		cfg := deps.CfgManager.GetConfig()
		channels := getChannelSlice(cfg, kind)
		index := findChannelIndexByUID(channels, upstream.ChannelUID)
		channelUID := upstream.ChannelUID

		// 异步触发发现（best-effort，不影响返回）
		discoveryStarted := triggerDiscoveryForChannel(deps, kind, channelUID)

		log.Printf("[AutoManaged-Add] 创建自动托管渠道: kind=%s name=%s uid=%s", kind, name, channelUID)

		c.JSON(http.StatusCreated, AutoAddResponse{
			ChannelUID:       channelUID,
			Index:            index,
			DiscoveryStarted: discoveryStarted,
		})
	}
}

func handleProviderAutoAdd(c *gin.Context, deps *AutoManagedDeps, requestKind string, req AutoAddRequest) {
	tmpl, ok := config.GetProviderTemplate(req.ProviderID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("未知的 provider: %s", req.ProviderID)})
		return
	}
	routes := tmpl.AutoAddRoutes()
	if len(routes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("provider %s 未配置可添加的渠道 route", req.ProviderID)})
		return
	}
	if !tmpl.SupportsChannelKind(requestKind) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("provider %s 不支持添加到 %s 渠道", req.ProviderID, requestKind)})
		return
	}

	baseName := req.Name
	if baseName == "" {
		baseName = fmt.Sprintf("%s-%d", tmpl.ProviderID, time.Now().UnixMilli()%100000)
	}

	cfg := deps.CfgManager.GetConfig()
	for _, route := range routes {
		name := providerRouteName(baseName, route, len(routes) > 1)
		if channelNameExists(getChannelSlice(cfg, route.ChannelKind), name) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("渠道名称 '%s' 已存在", name)})
			return
		}
	}

	type plannedRoute struct {
		route      config.ProviderRoute
		name       string
		keyConfigs []config.APIKeyConfig
		baseURLs   []string
	}
	planned := make([]plannedRoute, 0, len(routes))
	for _, route := range routes {
		keyConfigs, baseURLs, verr := verifyProviderRouteKeys(c.Request.Context(), tmpl, route, req.APIKeys)
		if verr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": verr.Error()})
			return
		}
		planned = append(planned, plannedRoute{
			route:      route,
			name:       providerRouteName(baseName, route, len(routes) > 1),
			keyConfigs: keyConfigs,
			baseURLs:   baseURLs,
		})
	}

	now := time.Now()
	results := make([]AutoAddChannelResult, 0, len(planned))
	for _, item := range planned {
		upstream := config.UpstreamConfig{
			Name:          item.name,
			ChannelUID:    config.GenerateChannelUID(),
			ServiceType:   item.route.ServiceType,
			Status:        "active",
			AutoManaged:   true,
			AutoManagedAt: &now,
			ProviderID:    tmpl.ProviderID,
			OriginType:    tmpl.OriginType,
			OriginTier:    tmpl.OriginTier,
			BaseURL:       item.baseURLs[0],
			BaseURLs:      item.baseURLs,
			APIKeys:       append([]string(nil), req.APIKeys...),
			APIKeyConfigs: item.keyConfigs,
		}
		if err := addUpstreamByKind(deps.CfgManager, item.route.ChannelKind, upstream); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建渠道失败: %v", err)})
			return
		}
		index := findChannelIndexByUID(getChannelSlice(deps.CfgManager.GetConfig(), item.route.ChannelKind), upstream.ChannelUID)
		discoveryStarted := triggerDiscoveryForChannel(deps, item.route.ChannelKind, upstream.ChannelUID)
		results = append(results, AutoAddChannelResult{
			ChannelKind:      item.route.ChannelKind,
			ChannelUID:       upstream.ChannelUID,
			Index:            index,
			Name:             upstream.Name,
			ServiceType:      upstream.ServiceType,
			DiscoveryStarted: discoveryStarted,
		})
		log.Printf("[AutoManaged-Add] 创建自动托管 provider 渠道: provider=%s kind=%s serviceType=%s name=%s uid=%s",
			tmpl.ProviderID, item.route.ChannelKind, item.route.ServiceType, upstream.Name, upstream.ChannelUID)
	}

	primary := primaryAutoAddResult(results, requestKind)
	c.JSON(http.StatusCreated, AutoAddResponse{
		ChannelUID:       primary.ChannelUID,
		Index:            primary.Index,
		DiscoveryStarted: primary.DiscoveryStarted,
		Channels:         results,
	})
}

func addUpstreamByKind(cfgManager *config.ConfigManager, kind string, upstream config.UpstreamConfig) error {
	switch kind {
	case "messages":
		return cfgManager.AddUpstream(upstream)
	case "chat":
		return cfgManager.AddChatUpstream(upstream)
	case "responses":
		return cfgManager.AddResponsesUpstream(upstream)
	case "gemini":
		return cfgManager.AddGeminiUpstream(upstream)
	case "images":
		return cfgManager.AddImagesUpstream(upstream)
	case "vectors":
		return cfgManager.AddVectorsUpstream(upstream)
	default:
		return fmt.Errorf("不支持的渠道类型: %s", kind)
	}
}

func providerRouteName(baseName string, route config.ProviderRoute, multiRoute bool) string {
	if !multiRoute {
		return baseName
	}
	switch route.ChannelKind {
	case "messages":
		return baseName + "-claude"
	case "chat":
		return baseName + "-chat"
	case "responses":
		return baseName + "-codex"
	default:
		return baseName + "-" + route.ChannelKind
	}
}

func channelNameExists(channels []config.UpstreamConfig, name string) bool {
	for _, ch := range channels {
		if ch.Name == name {
			return true
		}
	}
	return false
}

func findChannelIndexByUID(channels []config.UpstreamConfig, channelUID string) int {
	for i, ch := range channels {
		if ch.ChannelUID == channelUID {
			return i
		}
	}
	return -1
}

func triggerDiscoveryForChannel(deps *AutoManagedDeps, kind string, channelUID string) bool {
	if deps == nil || deps.Runner == nil || deps.CfgManager == nil || channelUID == "" {
		return false
	}
	cfg := deps.CfgManager.GetConfig()
	channels := getChannelSlice(cfg, kind)
	index := findChannelIndexByUID(channels, channelUID)
	if index < 0 || index >= len(channels) {
		return false
	}
	ch := channels[index]
	return deps.Runner.TriggerDiscovery(channelUID, &ch, deps.CfgManager)
}

func primaryAutoAddResult(results []AutoAddChannelResult, requestKind string) AutoAddChannelResult {
	for _, result := range results {
		if result.ChannelKind == requestKind {
			return result
		}
	}
	if len(results) > 0 {
		return results[0]
	}
	return AutoAddChannelResult{}
}

// handleAutoDiscover POST /{kind}/channels/:id/auto-discover
// 重新触发发现。
func handleAutoDiscover(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := extractKindFromPath(c)
		if !validChannelKinds[kind] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的渠道类型: %s", kind)})
			return
		}

		cfg := deps.CfgManager.GetConfig()
		channels := getChannelSlice(cfg, kind)
		id, found := findChannelIndex(channels, c.Param("id"))
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "渠道不存在"})
			return
		}

		channel := channels[id]
		if deps.Runner == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "发现服务未就绪"})
			return
		}

		started := deps.Runner.TriggerDiscovery(channel.ChannelUID, &channel, deps.CfgManager)
		if !started {
			c.JSON(http.StatusConflict, gin.H{"error": "发现任务已在运行中"})
			return
		}

		log.Printf("[AutoManaged-Discover] 重新触发发现: kind=%s id=%d uid=%s", kind, id, channel.ChannelUID)

		c.JSON(http.StatusOK, gin.H{
			"channelUid":       channel.ChannelUID,
			"discoveryStarted": true,
		})
	}
}

// handleAutoStatus GET /{kind}/channels/:id/auto-status
// 返回自动托管状态和发现结果。
func handleAutoStatus(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := extractKindFromPath(c)
		if !validChannelKinds[kind] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的渠道类型: %s", kind)})
			return
		}

		cfg := deps.CfgManager.GetConfig()
		channels := getChannelSlice(cfg, kind)
		id, found := findChannelIndex(channels, c.Param("id"))
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "渠道不存在"})
			return
		}

		channel := channels[id]
		resp := AutoStatusResponse{
			AutoManaged:   channel.AutoManaged,
			AutoManagedAt: channel.AutoManagedAt,
		}

		// 附加发现状态
		if deps.Runner != nil && channel.ChannelUID != "" {
			task := deps.Runner.GetTask(channel.ChannelUID)
			if task != nil {
				info := &DiscoveryStatusInfo{
					Status:     task.Status,
					StartedAt:  task.StartedAt,
					FinishedAt: task.FinishedAt,
					Error:      task.Error,
				}
				for _, ep := range task.Endpoints {
					info.Endpoints = append(info.Endpoints, EndpointDiscoveryInfo{
						KeyMask:     ep.KeyMask,
						BaseURL:     ep.BaseURL,
						ModelsCount: ep.ModelsCount,
						ProtocolOk:  ep.ProtocolOk,
					})
				}
				resp.Discovery = info
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

// ─── 辅助函数 ─────────────────────────────────────────────────────────────────────────

// kindToDefaultServiceType 根据渠道类型推导默认 serviceType。
func kindToDefaultServiceType(kind string) string {
	switch kind {
	case "messages":
		return "claude"
	case "gemini":
		return "gemini"
	case "chat", "images":
		return "openai"
	case "responses":
		return "responses"
	case "vectors":
		return "openai"
	default:
		return "openai"
	}
}

// getChannelSlice 根据 kind 从 Config 获取对应的渠道切片。
func getChannelSlice(cfg config.Config, kind string) []config.UpstreamConfig {
	switch kind {
	case "messages":
		return cfg.Upstream
	case "chat":
		return cfg.ChatUpstream
	case "responses":
		return cfg.ResponsesUpstream
	case "gemini":
		return cfg.GeminiUpstream
	case "images":
		return cfg.ImagesUpstream
	case "vectors":
		return cfg.VectorsUpstream
	default:
		return nil
	}
}

// findChannelIndex 按 channelUID 或整数索引在渠道列表中查找。
// `:id` 参数可以是 channelUID（ch_xxx）或整数下标，优先匹配 channelUID。
func findChannelIndex(channels []config.UpstreamConfig, idStr string) (int, bool) {
	// 先尝试按 channelUID 匹配
	for i, ch := range channels {
		if ch.ChannelUID == idStr {
			return i, true
		}
	}
	// 再尝试整数下标
	id := 0
	for _, ch := range idStr {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		id = id*10 + int(ch-'0')
	}
	if id >= 0 && id < len(channels) {
		return id, true
	}
	return 0, false
}
