package autopilot

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ─── 请求/响应类型 ──────────────────────────────────────────────────────────────

// SubscriptionCreateRequest POST /api/subscriptions 请求体。
type SubscriptionCreateRequest struct {
	SubscriptionUID    string             `json:"subscriptionUid" binding:"required"`
	DisplayName        string             `json:"displayName" binding:"required"`
	Provider           string             `json:"provider"`
	OriginType         string             `json:"originType"`
	OriginTier         string             `json:"originTier"`
	BillingMode        string             `json:"billingMode"`
	Currency           string             `json:"currency"`
	Balance            float64            `json:"balance"`
	GroupMultipliers   map[string]float64 `json:"groupMultipliers,omitempty"`
	RechargeMultiplier float64            `json:"rechargeMultiplier"`
	Notes              string             `json:"notes"`
	Source             string             `json:"source"`
}

// SubscriptionUpdateRequest PUT /api/subscriptions/:uid 请求体。
// 所有字段可选，仅更新非零值字段。
type SubscriptionUpdateRequest struct {
	DisplayName        *string            `json:"displayName,omitempty"`
	Provider           *string            `json:"provider,omitempty"`
	OriginType         *string            `json:"originType,omitempty"`
	OriginTier         *string            `json:"originTier,omitempty"`
	BillingMode        *string            `json:"billingMode,omitempty"`
	Currency           *string            `json:"currency,omitempty"`
	Balance            *float64           `json:"balance,omitempty"`
	GroupMultipliers   map[string]float64 `json:"groupMultipliers,omitempty"`
	RechargeMultiplier *float64           `json:"rechargeMultiplier,omitempty"`
	Notes              *string            `json:"notes,omitempty"`
	Source             *string            `json:"source,omitempty"`
	Confidence         *float64           `json:"confidence,omitempty"`
}

// LinkRequest POST /api/subscriptions/:uid/link 请求体。
type LinkRequest struct {
	ChannelUID string `json:"channelUid" binding:"required"`
}

// UnlinkRequest POST /api/subscriptions/:uid/unlink 请求体。
type UnlinkRequest struct {
	ChannelUID string `json:"channelUid" binding:"required"`
}

// SubscriptionItem 订阅列表/详情响应单条。
type SubscriptionItem struct {
	SubscriptionUID    string             `json:"subscriptionUid"`
	DisplayName        string             `json:"displayName"`
	Provider           string             `json:"provider,omitempty"`
	OriginType         string             `json:"originType,omitempty"`
	OriginTier         string             `json:"originTier,omitempty"`
	BillingMode        string             `json:"billingMode,omitempty"`
	Currency           string             `json:"currency,omitempty"`
	Balance            float64            `json:"balance,omitempty"`
	GroupMultipliers   map[string]float64 `json:"groupMultipliers,omitempty"`
	RechargeMultiplier float64            `json:"rechargeMultiplier,omitempty"`
	LinkedChannelUIDs  []string           `json:"linkedChannelUids,omitempty"`
	Source             string             `json:"source,omitempty"`
	Confidence         float64            `json:"confidence,omitempty"`
	Notes              string             `json:"notes,omitempty"`
	CreatedAt          string             `json:"createdAt"`
	UpdatedAt          string             `json:"updatedAt"`
	ArchivedAt         string             `json:"archivedAt,omitempty"`
}

// SubscriptionsListResponse GET /api/subscriptions 返回结构。
type SubscriptionsListResponse struct {
	Subscriptions []SubscriptionItem `json:"subscriptions"`
	Total         int                `json:"total"`
}

// ─── 路由注册 ──────────────────────────────────────────────────────────────────

// RegisterSubscriptionRoutes 注册订阅中心 CRUD + 渠道链接 API 到给定路由组。
func RegisterSubscriptionRoutes(router gin.IRouter, store *SubscriptionStore) {
	group := router.Group("/subscriptions")
	{
		group.GET("", handleListSubscriptions(store))
		group.POST("", handleCreateSubscription(store))
		group.GET("/:uid", handleGetSubscription(store))
		group.PUT("/:uid", handleUpdateSubscription(store))
		group.DELETE("/:uid", handleDeleteSubscription(store))
		group.POST("/:uid/link", handleLinkChannel(store))
		group.POST("/:uid/unlink", handleUnlinkChannel(store))
	}
}

// ─── Handler 实现 ──────────────────────────────────────────────────────────────

// handleListSubscriptions GET /api/subscriptions
func handleListSubscriptions(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		all := store.ListAll()
		items := make([]SubscriptionItem, 0, len(all))
		for _, p := range all {
			items = append(items, toSubscriptionItem(p))
		}

		c.JSON(http.StatusOK, SubscriptionsListResponse{
			Subscriptions: items,
			Total:         len(items),
		})
	}
}

// handleCreateSubscription POST /api/subscriptions
func handleCreateSubscription(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SubscriptionCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}

		profile := &SubscriptionProfile{
			SubscriptionUID:    req.SubscriptionUID,
			DisplayName:        req.DisplayName,
			Provider:           req.Provider,
			OriginType:         req.OriginType,
			OriginTier:         req.OriginTier,
			BillingMode:        req.BillingMode,
			Currency:           req.Currency,
			Balance:            req.Balance,
			GroupMultipliers:   req.GroupMultipliers,
			RechargeMultiplier: req.RechargeMultiplier,
			Notes:              req.Notes,
			Source:             req.Source,
		}
		if profile.Source == "" {
			profile.Source = "manual"
		}

		if err := store.Create(profile); err != nil {
			if strings.Contains(err.Error(), "已存在") {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusCreated, toSubscriptionItem(profile))
	}
}

// handleGetSubscription GET /api/subscriptions/:uid
func handleGetSubscription(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		p := store.Get(uid)
		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "订阅不存在: " + uid})
			return
		}

		c.JSON(http.StatusOK, toSubscriptionItem(p))
	}
}

// handleUpdateSubscription PUT /api/subscriptions/:uid
func handleUpdateSubscription(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		existing := store.Get(uid)
		if existing == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "订阅不存在: " + uid})
			return
		}

		var req SubscriptionUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}

		// 合并更新
		if req.DisplayName != nil {
			existing.DisplayName = *req.DisplayName
		}
		if req.Provider != nil {
			existing.Provider = *req.Provider
		}
		if req.OriginType != nil {
			existing.OriginType = *req.OriginType
		}
		if req.OriginTier != nil {
			existing.OriginTier = *req.OriginTier
		}
		if req.BillingMode != nil {
			existing.BillingMode = *req.BillingMode
		}
		if req.Currency != nil {
			existing.Currency = *req.Currency
		}
		if req.Balance != nil {
			existing.Balance = *req.Balance
		}
		if req.GroupMultipliers != nil {
			existing.GroupMultipliers = req.GroupMultipliers
		}
		if req.RechargeMultiplier != nil {
			existing.RechargeMultiplier = *req.RechargeMultiplier
		}
		if req.Notes != nil {
			existing.Notes = *req.Notes
		}
		if req.Source != nil {
			existing.Source = *req.Source
		}
		if req.Confidence != nil {
			existing.Confidence = *req.Confidence
		}

		if err := store.Update(existing); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, toSubscriptionItem(existing))
	}
}

// handleDeleteSubscription DELETE /api/subscriptions/:uid
func handleDeleteSubscription(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		if err := store.Delete(uid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// handleLinkChannel POST /api/subscriptions/:uid/link
func handleLinkChannel(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		var req LinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}

		if err := store.LinkChannel(uid, req.ChannelUID); err != nil {
			if strings.Contains(err.Error(), "不存在") {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		p := store.Get(uid)
		c.JSON(http.StatusOK, toSubscriptionItem(p))
	}
}

// handleUnlinkChannel POST /api/subscriptions/:uid/unlink
func handleUnlinkChannel(store *SubscriptionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		var req UnlinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}

		if err := store.UnlinkChannel(uid, req.ChannelUID); err != nil {
			if strings.Contains(err.Error(), "不存在") {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		p := store.Get(uid)
		c.JSON(http.StatusOK, toSubscriptionItem(p))
	}
}

// ─── 内部辅助 ──────────────────────────────────────────────────────────────────

// toSubscriptionItem 将 SubscriptionProfile 转为 API 响应结构。
func toSubscriptionItem(p *SubscriptionProfile) SubscriptionItem {
	item := SubscriptionItem{
		SubscriptionUID:    p.SubscriptionUID,
		DisplayName:        p.DisplayName,
		Provider:           p.Provider,
		OriginType:         p.OriginType,
		OriginTier:         p.OriginTier,
		BillingMode:        p.BillingMode,
		Currency:           p.Currency,
		Balance:            p.Balance,
		GroupMultipliers:   p.GroupMultipliers,
		RechargeMultiplier: p.RechargeMultiplier,
		LinkedChannelUIDs:  p.LinkedChannelUIDs,
		Source:             p.Source,
		Confidence:         p.Confidence,
		Notes:              p.Notes,
		CreatedAt:          p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if p.ArchivedAt != nil {
		item.ArchivedAt = p.ArchivedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return item
}
