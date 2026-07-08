package autopilot

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── 请求/响应类型 ──────────────────────────────────────────────────────────────

// CreateLocalRuntimeRequest POST /api/local-runtimes 请求体。
type CreateLocalRuntimeRequest struct {
	Name        string      `json:"name" binding:"required"`
	RuntimeType RuntimeType `json:"runtimeType" binding:"required"`
	BaseURL     string      `json:"baseUrl" binding:"required"`

	// 可选：本地资源提示
	ContextTokens     int     `json:"contextTokens,omitempty"`
	SupportsTools     bool    `json:"supportsTools,omitempty"`
	SupportsVision    bool    `json:"supportsVision,omitempty"`
	SupportsReasoning bool    `json:"supportsReasoning,omitempty"`
	TokensPerSecond   float64 `json:"tokensPerSecond,omitempty"`
	TimeoutMs         int     `json:"timeoutMs,omitempty"`
}

// LocalRuntimeResponse 单条运行时响应（供 GET / GET list / POST 使用）。
type LocalRuntimeResponse struct {
	RuntimeUID       string             `json:"runtimeUid"`
	Name             string             `json:"name,omitempty"`
	RuntimeType      RuntimeType        `json:"runtimeType"`
	BaseURL          string             `json:"baseUrl"`
	DiscoveredModels []string           `json:"discoveredModels,omitempty"`
	Status           LocalRuntimeStatus `json:"status"`
	LatencyMs        int64              `json:"latencyMs,omitempty"`

	ContextTokens     int     `json:"contextTokens,omitempty"`
	SupportsTools     bool    `json:"supportsTools,omitempty"`
	SupportsVision    bool    `json:"supportsVision,omitempty"`
	SupportsReasoning bool    `json:"supportsReasoning,omitempty"`
	TokensPerSecond   float64 `json:"tokensPerSecond,omitempty"`
	TimeoutMs         int     `json:"timeoutMs,omitempty"`

	LastProbeAt string `json:"lastProbeAt,omitempty"`
	UpdatedAt   string `json:"updatedAt"`
	CreatedAt   string `json:"createdAt"`
}

// LocalRuntimesResponse GET /api/local-runtimes 列表响应。
type LocalRuntimesResponse struct {
	Runtimes []LocalRuntimeResponse `json:"runtimes"`
	Total    int                    `json:"total"`
}

// ProbeResultResponse POST /api/local-runtimes/:uid/probe 响应。
type ProbeResultResponse struct {
	RuntimeUID       string             `json:"runtimeUid"`
	Status           LocalRuntimeStatus `json:"status"`
	DiscoveredModels []string           `json:"discoveredModels,omitempty"`
	LatencyMs        int64              `json:"latencyMs,omitempty"`
	LastProbeAt      string             `json:"lastProbeAt"`
	Error            string             `json:"error,omitempty"`
}

// ─── 路由注册 ────────────────────────────────────────────────────────────────────

// RegisterLocalRuntimeRoutes 注册本地运行时管理 API 到给定路由组。
//
// 路由：
//
//	GET    /api/local-runtimes           — 列表
//	POST   /api/local-runtimes           — 创建
//	GET    /api/local-runtimes/:uid      — 详情
//	DELETE /api/local-runtimes/:uid      — 删除
//	POST   /api/local-runtimes/:uid/probe — 探测
func RegisterLocalRuntimeRoutes(router gin.IRouter, store *LocalRuntimeStore) {
	group := router.Group("/local-runtimes")
	{
		group.GET("", handleListLocalRuntimes(store))
		group.POST("", handleCreateLocalRuntime(store))
		group.GET("/:uid", handleGetLocalRuntime(store))
		group.DELETE("/:uid", handleDeleteLocalRuntime(store))
		group.POST("/:uid/probe", handleProbeLocalRuntime(store))
	}
}

// ─── Handler 实现 ────────────────────────────────────────────────────────────────

// handleListLocalRuntimes GET /api/local-runtimes
func handleListLocalRuntimes(store *LocalRuntimeStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		runtimes := store.ListAll()
		items := make([]LocalRuntimeResponse, 0, len(runtimes))
		for _, p := range runtimes {
			items = append(items, toLocalRuntimeResponse(p))
		}
		c.JSON(http.StatusOK, LocalRuntimesResponse{
			Runtimes: items,
			Total:    len(items),
		})
	}
}

// handleCreateLocalRuntime POST /api/local-runtimes
func handleCreateLocalRuntime(store *LocalRuntimeStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLocalRuntimeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效: " + err.Error()})
			return
		}

		if !IsValidRuntimeType(req.RuntimeType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 runtimeType，合法值: ollama, lmstudio, llama_server, openai_compatible"})
			return
		}

		now := time.Now()
		profile := &LocalModelRuntimeProfile{
			RuntimeUID:        GenerateRuntimeUID(),
			Name:              req.Name,
			RuntimeType:       req.RuntimeType,
			BaseURL:           req.BaseURL,
			Status:            LocalRuntimeUnknown,
			ContextTokens:     req.ContextTokens,
			SupportsTools:     req.SupportsTools,
			SupportsVision:    req.SupportsVision,
			SupportsReasoning: req.SupportsReasoning,
			TokensPerSecond:   req.TokensPerSecond,
			TimeoutMs:         req.TimeoutMs,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if err := store.Upsert(profile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, toLocalRuntimeResponse(profile))
	}
}

// handleGetLocalRuntime GET /api/local-runtimes/:uid
func handleGetLocalRuntime(store *LocalRuntimeStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		profile := store.Get(uid)
		if profile == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "运行时不存在"})
			return
		}

		c.JSON(http.StatusOK, toLocalRuntimeResponse(profile))
	}
}

// handleDeleteLocalRuntime DELETE /api/local-runtimes/:uid
func handleDeleteLocalRuntime(store *LocalRuntimeStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		if store.Get(uid) == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "运行时不存在"})
			return
		}

		if err := store.Delete(uid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "已删除", "runtimeUid": uid})
	}
}

// handleProbeLocalRuntime POST /api/local-runtimes/:uid/probe
func handleProbeLocalRuntime(store *LocalRuntimeStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uid 不能为空"})
			return
		}

		profile := store.Get(uid)
		if profile == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "运行时不存在"})
			return
		}

		probeErr := ProbeRuntime(c.Request.Context(), profile)
		profile.UpdatedAt = time.Now()

		if err := store.Upsert(profile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新画像失败: " + err.Error()})
			return
		}

		resp := ProbeResultResponse{
			RuntimeUID:       profile.RuntimeUID,
			Status:           profile.Status,
			DiscoveredModels: profile.DiscoveredModels,
			LatencyMs:        profile.LatencyMs,
			LastProbeAt:      profile.LastProbeAt.Format(time.RFC3339),
		}
		if probeErr != nil {
			resp.Error = probeErr.Error()
		}

		c.JSON(http.StatusOK, resp)
	}
}

// ─── 内部辅助 ────────────────────────────────────────────────────────────────────

// toLocalRuntimeResponse 将内部 profile 转为 API 响应。
func toLocalRuntimeResponse(p *LocalModelRuntimeProfile) LocalRuntimeResponse {
	r := LocalRuntimeResponse{
		RuntimeUID:        p.RuntimeUID,
		Name:              p.Name,
		RuntimeType:       p.RuntimeType,
		BaseURL:           p.BaseURL,
		DiscoveredModels:  p.DiscoveredModels,
		Status:            p.Status,
		LatencyMs:         p.LatencyMs,
		ContextTokens:     p.ContextTokens,
		SupportsTools:     p.SupportsTools,
		SupportsVision:    p.SupportsVision,
		SupportsReasoning: p.SupportsReasoning,
		TokensPerSecond:   p.TokensPerSecond,
		TimeoutMs:         p.TimeoutMs,
		UpdatedAt:         p.UpdatedAt.Format(time.RFC3339),
		CreatedAt:         p.CreatedAt.Format(time.RFC3339),
	}
	if p.LastProbeAt != nil {
		r.LastProbeAt = p.LastProbeAt.Format(time.RFC3339)
	}
	return r
}
