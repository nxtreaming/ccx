package autopilot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/presetstore"
	"github.com/gin-gonic/gin"
)

// ─── new-api 订阅集成路由（§8.5.1）──────────────────────────────────────────

// newApiDefaults 返回 new-api 接入的建议预填值：优先取 presetstore，
// 缺失字段回退到编译期兜底（relay/second/token_plan）。
func newApiDefaults() presetstore.NewApiDefaults {
	d := presetstore.Default().Subscription().NewApiDefaults
	if d.OriginType == "" {
		d.OriginType = "relay"
	}
	if d.OriginTier == "" {
		d.OriginTier = "second"
	}
	if d.BillingMode == "" {
		d.BillingMode = "token_plan"
	}
	return d
}

// NewApiRouteDeps new-api 端点所需的依赖注入。
// Verify 端点只需要 SubscriptionStore；Provision 端点额外需要 CfgManager + Runner 来建渠道并触发 Discovery。
type NewApiRouteDeps struct {
	Store      *SubscriptionStore
	CfgManager *config.ConfigManager
	Runner     *AutoDiscoveryRunner
}

// RegisterNewApiSubscriptionRoutes 注册 new-api 集成的两个核心端点：
//
//	POST /subscriptions/newapi/verify    —— 校验令牌 + 预览账户/分组/模型（不落库）
//	POST /subscriptions/newapi/provision —— 完整流程：建 profile + 建 key + 建渠道 + 触发 Discovery
func RegisterNewApiSubscriptionRoutes(router gin.IRouter, deps *NewApiRouteDeps) {
	if deps == nil || deps.Store == nil {
		log.Printf("[NewApi-Routes] 依赖缺失，跳过注册")
		return
	}
	group := router.Group("/subscriptions")
	group.POST("/newapi/verify", handleNewApiVerify(deps))
	group.POST("/newapi/provision", handleNewApiProvision(deps))
}

// ─── 请求/响应类型 ───

// NewApiVerifyRequest POST /subscriptions/newapi/verify 请求体。
type NewApiVerifyRequest struct {
	BaseURL         string `json:"baseUrl" binding:"required"`
	AccessToken     string `json:"accessToken" binding:"required"`
	UserID          string `json:"userId"`
	AuthTokenMode   string `json:"authTokenMode,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	SubscriptionUID string `json:"subscriptionUid,omitempty"`
}

// NewApiVerifyResponse POST /subscriptions/newapi/verify 响应体（不落库）。
type NewApiVerifyResponse struct {
	Username        string             `json:"username"`
	UserID          int                `json:"userId"`
	Quota           int64              `json:"quota"`
	UsedQuota       int64              `json:"usedQuota"`
	Groups          map[string]float64 `json:"groups"`
	AvailableModels []string           `json:"availableModels"`
	// 派生建议：前端可直接展示
	SuggestedOriginType string `json:"suggestedOriginType"`
	SuggestedOriginTier string `json:"suggestedOriginTier"`
	AccessTokenMasked   string `json:"accessTokenMasked"`
}

// NewApiProvisionRequest POST /subscriptions/newapi/provision 请求体。
type NewApiProvisionRequest struct {
	SubscriptionUID  string   `json:"subscriptionUid" binding:"required"`
	DisplayName      string   `json:"displayName" binding:"required"`
	BaseURL          string   `json:"baseUrl" binding:"required"`
	AccessToken      string   `json:"accessToken" binding:"required"`
	UserID           string   `json:"userId"`
	AuthTokenMode    string   `json:"authTokenMode,omitempty"`
	ChannelKind      string   `json:"channelKind" binding:"required"` // messages/chat/responses/gemini
	ChannelName      string   `json:"channelName,omitempty"`
	ProvisionKeyName string   `json:"provisionKeyName,omitempty"`
	ProvisionGroup   string   `json:"provisionGroup,omitempty"`
	ProvisionModels  []string `json:"provisionModels,omitempty"`
	Notes            string   `json:"notes,omitempty"`
}

// NewApiProvisionResponse POST /subscriptions/newapi/provision 响应体。
type NewApiProvisionResponse struct {
	Subscription       SubscriptionItem `json:"subscription"`
	ChannelUID         string           `json:"channelUid"`
	ChannelIndex       int              `json:"channelIndex"`
	ProvisionedKey     string           `json:"provisionedKey"` // 明文 key，仅此次返回，前端必须立即转给渠道；后续只展示脱敏/不回显
	ProvisionedTokenID int              `json:"provisionedTokenId"`
	Reused             bool             `json:"reused"` // true 表示复用了已存在的同名 key
	DiscoveryStarted   bool             `json:"discoveryStarted"`
}

// ─── Handler ───

// handleNewApiVerify 校验 new-api 凭据 + 预览账户/分组/模型信息（不写入数据库）。
func handleNewApiVerify(deps *NewApiRouteDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NewApiVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}
		if req.AccessToken == "" || req.BaseURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "baseUrl 和 accessToken 必填"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()

		adapter := &NewApiAdapter{}

		// 1) 校验 + 取用户信息
		self, err := adapter.Verify(ctx, req.BaseURL, req.AccessToken, req.UserID, req.AuthTokenMode)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("校验失败: %v", err)})
			return
		}
		// 若前端没传 userId，用站点回填的 id 自动回填
		derivedUserID := req.UserID
		if derivedUserID == "" {
			derivedUserID = fmt.Sprintf("%d", self.ID)
		}

		// 2) 拉分组倍率（失败不阻断——只是少了分组预览）
		groups, _ := adapter.FetchGroups(ctx, req.BaseURL, req.AccessToken, derivedUserID, req.AuthTokenMode)

		// 3) 拉可用模型（失败不阻断）
		models, _ := adapter.FetchModels(ctx, req.BaseURL, req.AccessToken, derivedUserID, req.AuthTokenMode)

		defaults := newApiDefaults()
		resp := NewApiVerifyResponse{
			Username:            self.Username,
			UserID:              self.ID,
			Quota:               self.Quota,
			UsedQuota:           self.UsedQuota,
			Groups:              groups,
			AvailableModels:     models,
			SuggestedOriginType: defaults.OriginType,
			SuggestedOriginTier: defaults.OriginTier,
			AccessTokenMasked:   maskAccessToken(req.AccessToken),
		}
		c.JSON(http.StatusOK, resp)
	}
}

// handleNewApiProvision 完整流程：建 profile + 建 key + 建渠道 + 触发 Discovery。
func handleNewApiProvision(deps *NewApiRouteDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.CfgManager == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "配置管理器未就绪，无法建渠道"})
			return
		}

		var req NewApiProvisionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}
		if req.AccessToken == "" || req.BaseURL == "" || req.SubscriptionUID == "" || req.DisplayName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "subscriptionUid、displayName、baseUrl、accessToken、channelKind 必填"})
			return
		}
		if !validChannelKinds[req.ChannelKind] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的渠道类型: %s", req.ChannelKind)})
			return
		}
		// 提前校验 subscriptionUid 唯一性，避免白白在 new-api 侧建 key 后才发现 profile 冲突。
		if deps.Store.Get(req.SubscriptionUID) != nil {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("subscriptionUid=%s 已存在", req.SubscriptionUID)})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		adapter := &NewApiAdapter{}

		// 1) 校验 + 拉用户信息（同时获取 userId 兜底）
		self, err := adapter.Verify(ctx, req.BaseURL, req.AccessToken, req.UserID, req.AuthTokenMode)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("校验失败: %v", err)})
			return
		}
		derivedUserID := req.UserID
		if derivedUserID == "" {
			derivedUserID = fmt.Sprintf("%d", self.ID)
		}

		// 2) 拉分组倍率 + 可用模型（best-effort）
		groups, _ := adapter.FetchGroups(ctx, req.BaseURL, req.AccessToken, derivedUserID, req.AuthTokenMode)
		models, _ := adapter.FetchModels(ctx, req.BaseURL, req.AccessToken, derivedUserID, req.AuthTokenMode)

		// 3) 建/复用代理 key
		provName := req.ProvisionKeyName
		if provName == "" {
			provName = DefaultNewApiProvisionKeyName
		}
		tokenID, keyPlain, reused, err := adapter.ProvisionKey(ctx, req.BaseURL, req.AccessToken, derivedUserID, req.AuthTokenMode, NewApiProvisionOptions{
			Name:   provName,
			Group:  req.ProvisionGroup,
			Models: req.ProvisionModels,
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("建 key 失败: %v", err)})
			return
		}
		// 复用旧 key 时 keyPlain 为空——前端必须已经持有关键信息；此情形降级为报错让用户重试或手动填
		if keyPlain == "" {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("站点已存在同名 key=%s 但未返回明文，无法直接绑定，请删除后重试或手动填 key", provName)})
			return
		}

		// 4) 建 profile
		now := time.Now()
		defaults := newApiDefaults()
		profile := &SubscriptionProfile{
			SubscriptionUID:    req.SubscriptionUID,
			DisplayName:        req.DisplayName,
			Provider:           "new_api",
			OriginType:         defaults.OriginType,
			OriginTier:         defaults.OriginTier,
			BillingMode:        defaults.BillingMode,
			Currency:           "quota",
			Balance:            float64(self.Quota),
			GroupMultipliers:   groups,
			RechargeMultiplier: 1.0,
			LinkedChannelUIDs:  []string{},
			Source:             "newapi_provision",
			Confidence:         0.95,
			Notes:              req.Notes,
			// §8.5.1
			BaseURL:            req.BaseURL,
			AccessToken:        req.AccessToken, // 持久化但不出 API 响应
			UserID:             derivedUserID,
			AuthTokenMode:      req.AuthTokenMode,
			ProvisionKeyName:   provName,
			ProvisionGroup:     req.ProvisionGroup,
			ProvisionModels:    req.ProvisionModels,
			ProvisionedTokenID: tokenID,
			AvailableModels:    models,
			AutoRefreshEnabled: false, // new-api 走 Verify，不直接接 SubscriptionBalanceFetcher
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := deps.Store.Create(profile); err != nil {
			if strings.Contains(err.Error(), "已存在") {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		// 5) 建上游渠道
		channelName := req.ChannelName
		if channelName == "" {
			channelName = fmt.Sprintf("newapi-%s-%d", req.ChannelKind, time.Now().UnixMilli()%100000)
		}
		serviceType := kindToDefaultServiceType(req.ChannelKind)
		channelUID := config.GenerateChannelUID()
		upstream := config.UpstreamConfig{
			Name:          channelName,
			ChannelUID:    channelUID,
			BaseURL:       strings.TrimRight(req.BaseURL, "/"),
			BaseURLs:      []string{strings.TrimRight(req.BaseURL, "/")},
			APIKeys:       []string{keyPlain},
			ServiceType:   serviceType,
			Status:        "active",
			AutoManaged:   true,
			AutoManagedAt: &now,
			OriginType:    "relay",
			OriginTier:    "second",
		}

		switch req.ChannelKind {
		case "messages":
			err = deps.CfgManager.AddUpstream(upstream)
		case "chat":
			err = deps.CfgManager.AddChatUpstream(upstream)
		case "responses":
			err = deps.CfgManager.AddResponsesUpstream(upstream)
		case "gemini":
			err = deps.CfgManager.AddGeminiUpstream(upstream)
		case "images":
			err = deps.CfgManager.AddImagesUpstream(upstream)
		case "vectors":
			err = deps.CfgManager.AddVectorsUpstream(upstream)
		}
		if err != nil {
			// 渠道建失败：回滚 profile（最佳努力删除）
			_ = deps.Store.Delete(req.SubscriptionUID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("建渠道失败: %v", err)})
			return
		}

		// 6) 找到新建渠道的 index + 关联订阅
		cfg := deps.CfgManager.GetConfig()
		channels := getChannelSlice(cfg, req.ChannelKind)
		channelIndex := -1
		for i, ch := range channels {
			if ch.Name == channelName {
				channelIndex = i
				break
			}
		}
		if channelIndex < 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "渠道已建但无法定位"})
			return
		}
		if err := deps.Store.LinkChannel(req.SubscriptionUID, channelUID); err != nil {
			log.Printf("[NewApi-Provision] 关联渠道失败 subscription=%s channel=%s: %v", req.SubscriptionUID, channelUID, err)
		}

		// 7) 触发 Discovery（best-effort）
		discoveryStarted := false
		if deps.Runner != nil {
			cfg = deps.CfgManager.GetConfig()
			channels = getChannelSlice(cfg, req.ChannelKind)
			if channelIndex < len(channels) {
				ch := channels[channelIndex]
				discoveryStarted = deps.Runner.TriggerDiscovery(channelUID, &ch, deps.CfgManager)
			}
		}

		log.Printf("[NewApi-Provision] 完成 subscription=%s channelUID=%s tokenID=%d reused=%v discovery=%v",
			req.SubscriptionUID, channelUID, tokenID, reused, discoveryStarted)

		fresh := deps.Store.Get(req.SubscriptionUID)
		if fresh == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "订阅已建但无法回读"})
			return
		}
		c.JSON(http.StatusCreated, NewApiProvisionResponse{
			Subscription:       toSubscriptionItem(fresh),
			ChannelUID:         channelUID,
			ChannelIndex:       channelIndex,
			ProvisionedKey:     keyPlain,
			ProvisionedTokenID: tokenID,
			Reused:             reused,
			DiscoveryStarted:   discoveryStarted,
		})
	}
}
