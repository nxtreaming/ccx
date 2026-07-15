package autopilot

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/utils"
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
	AccountUID       string                 `json:"accountUid"`
	ChannelUID       string                 `json:"channelUid"`
	Index            int                    `json:"index"`
	DiscoveryStarted bool                   `json:"discoveryStarted"`
	Channels         []AutoAddChannelResult `json:"channels,omitempty"`
}

// AutoAddChannelResult 描述 provider 快速添加一次创建出的单条渠道。
type AutoAddChannelResult struct {
	AccountUID       string `json:"accountUid"`
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
	CfgManager        *config.ConfigManager
	Runner            *AutoDiscoveryRunner
	MiMoConsoleClient *MiMoConsoleClient
	DeepSeekClient    *DeepSeekClient
}

// RegisterAutoManagedRoutes 注册自动托管 API 路由。
// 路由直接挂载到 apiGroup（不创建子组），与现有渠道管理路由共存。
//
// 注意：必须为每个 kind 显式注册静态路径，不能用 `:kind` 参数，
// 否则会与现有的 `/messages/channels/...` 等静态路由在 Gin radix tree 中冲突。
func RegisterAutoManagedRoutes(apiGroup *gin.RouterGroup, deps *AutoManagedDeps) {
	apiGroup.GET("/accounts", handleListAccounts(deps))
	apiGroup.PATCH("/accounts/:accountUid", handleRenameAccount(deps))
	apiGroup.PUT("/accounts/:accountUid", handleUpdateAccount(deps))
	apiGroup.DELETE("/accounts/:accountUid", handleDeleteAccount(deps))
	apiGroup.POST("/accounts/:accountUid/credentials", handleAddAccountCredentials(deps))
	apiGroup.PATCH("/accounts/:accountUid/credentials", handlePatchAccountCredentials(deps))
	apiGroup.DELETE("/accounts/:accountUid/credentials/:credentialUid", handleDeleteAccountCredential(deps))
	apiGroup.PUT("/accounts/:accountUid/credentials/:credentialUid/volcengine-access-key", handleSetVolcengineAccessKey(deps))
	apiGroup.DELETE("/accounts/:accountUid/credentials/:credentialUid/volcengine-access-key", handleClearVolcengineAccessKey(deps))
	apiGroup.POST("/accounts/:accountUid/credentials/:credentialUid/volcengine-plan-usage/refresh", handleRefreshVolcenginePlanUsage(deps))
	apiGroup.PUT("/accounts/:accountUid/credentials/:credentialUid/mimo-console-cookie", handleSetMiMoConsoleCookie(deps))
	apiGroup.POST("/accounts/:accountUid/credentials/:credentialUid/mimo-console-cookie/refresh", handleRefreshMiMoConsoleCookie(deps))
	apiGroup.DELETE("/accounts/:accountUid/credentials/:credentialUid/mimo-console-cookie", handleClearMiMoConsoleCookie(deps))
	apiGroup.GET("/accounts/:accountUid/deepseek-balance", handleDeepSeekBalance(deps))
	kinds := []string{"messages", "chat", "responses", "gemini", "images", "vectors"}
	for _, kind := range kinds {
		apiGroup.POST("/"+kind+"/channels/auto-add", handleAutoAdd(deps))
		apiGroup.POST("/"+kind+"/channels/:id/auto-discover", handleAutoDiscover(deps))
		apiGroup.GET("/"+kind+"/channels/:id/auto-status", handleAutoStatus(deps))
	}
}

type managedDeepSeekCredentialBalance struct {
	CredentialUID string                `json:"credentialUid"`
	KeyMask       string                `json:"keyMask"`
	IsAvailable   bool                  `json:"isAvailable"`
	BalanceInfos  []DeepSeekBalanceInfo `json:"balanceInfos,omitempty"`
	FetchedAt     time.Time             `json:"fetchedAt"`
	Error         string                `json:"error,omitempty"`
}

func handleDeepSeekBalance(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		cfg := deps.CfgManager.GetConfig()
		var account *config.ManagedAccountConfig
		for i := range cfg.ManagedAccounts {
			if cfg.ManagedAccounts[i].AccountUID == accountUID {
				account = &cfg.ManagedAccounts[i]
				break
			}
		}
		if account == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
			return
		}
		if account.ProviderID != "deepseek" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅 DeepSeek 自动托管账号支持余额查询"})
			return
		}

		client := deps.DeepSeekClient
		if client == nil {
			client = NewDeepSeekClient(nil)
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), deepSeekRequestTimeout)
		defer cancel()
		balances := make([]managedDeepSeekCredentialBalance, len(account.Credentials))
		workers := make(chan struct{}, deepSeekBalanceWorkers)
		var wg sync.WaitGroup
		for i, credential := range account.Credentials {
			wg.Add(1)
			go func(index int, credential config.ManagedAccountCredential) {
				defer wg.Done()
				select {
				case workers <- struct{}{}:
					defer func() { <-workers }()
				case <-ctx.Done():
					balances[index] = managedDeepSeekCredentialBalance{
						CredentialUID: credential.CredentialUID,
						KeyMask:       utils.MaskAPIKey(credential.APIKey),
						FetchedAt:     time.Now().UTC(),
						Error:         "DeepSeek 余额查询超时",
					}
					return
				}
				balance, err := client.FetchBalance(ctx, credential.APIKey)
				balances[index] = managedDeepSeekCredentialBalance{
					CredentialUID: credential.CredentialUID,
					KeyMask:       utils.MaskAPIKey(credential.APIKey),
					FetchedAt:     time.Now().UTC(),
				}
				if err != nil {
					balances[index].Error = err.Error()
					return
				}
				balances[index].IsAvailable = balance.IsAvailable
				balances[index].BalanceInfos = balance.BalanceInfos
			}(i, credential)
		}
		wg.Wait()
		c.JSON(http.StatusOK, gin.H{"accountUid": accountUID, "balances": balances})
	}
}

func handleRenameAccount(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		if err := deps.CfgManager.RenameManagedAccount(accountUID, req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"accountUid": accountUID, "name": strings.TrimSpace(req.Name)})
	}
}

type managedAccountCredentialView struct {
	CredentialUID             string                      `json:"credentialUid"`
	KeyMask                   string                      `json:"keyMask"`
	HasVolcengineAccessKey    bool                        `json:"hasVolcengineAccessKey,omitempty"`
	VolcengineAccessKeyIDMask string                      `json:"volcengineAccessKeyIdMask,omitempty"`
	VolcenginePlan            string                      `json:"volcenginePlan,omitempty"`
	VolcenginePlanTier        string                      `json:"volcenginePlanTier,omitempty"`
	VolcenginePlanStatus      string                      `json:"volcenginePlanStatus,omitempty"`
	VolcenginePlanUsage       *config.VolcenginePlanUsage `json:"volcenginePlanUsage,omitempty"`
	HasMiMoConsoleCookie      bool                        `json:"hasMiMoConsoleCookie,omitempty"`
	MiMoTokenPlan             *managedMiMoTokenPlanView   `json:"mimoTokenPlan,omitempty"`
}

type managedMiMoTokenPlanView struct {
	PlanCode         string                         `json:"planCode"`
	PlanName         string                         `json:"planName"`
	CurrentPeriodEnd string                         `json:"currentPeriodEnd"`
	Expired          bool                           `json:"expired"`
	MonthUsage       config.MiMoTokenPlanUsageQuota `json:"monthUsage"`
	CurrentUsage     config.MiMoTokenPlanUsageQuota `json:"currentUsage"`
	ValidatedAt      time.Time                      `json:"validatedAt"`
}

type managedAccountChannelView struct {
	Kind        string `json:"kind"`
	ChannelUID  string `json:"channelUid"`
	Name        string `json:"name"`
	ServiceType string `json:"serviceType"`
	Status      string `json:"status"`
}

type managedAccountView struct {
	AccountUID    string                         `json:"accountUid"`
	ProviderID    string                         `json:"providerId"`
	Name          string                         `json:"name"`
	Credentials   []managedAccountCredentialView `json:"credentials"`
	Channels      []managedAccountChannelView    `json:"channels"`
	EndpointCount int                            `json:"endpointCount"`
}

func handleListAccounts(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		cfg := deps.CfgManager.GetConfig()
		accounts := make([]managedAccountView, 0, len(cfg.ManagedAccounts))
		for _, account := range cfg.ManagedAccounts {
			view := managedAccountView{AccountUID: account.AccountUID, ProviderID: account.ProviderID, Name: account.Name}
			for _, credential := range account.Credentials {
				credentialView := managedAccountCredentialView{
					CredentialUID: credential.CredentialUID,
					KeyMask:       utils.MaskAPIKey(credential.APIKey),
				}
				if credential.VolcengineAccessKey != nil {
					credentialView.HasVolcengineAccessKey = true
					credentialView.VolcengineAccessKeyIDMask = utils.MaskAPIKey(credential.VolcengineAccessKey.AccessKeyID)
					credentialView.VolcenginePlan = credential.VolcengineAccessKey.Plan
					credentialView.VolcenginePlanTier = credential.VolcengineAccessKey.PlanTier
					credentialView.VolcenginePlanStatus = credential.VolcengineAccessKey.PlanStatus
					credentialView.VolcenginePlanUsage = credential.VolcengineAccessKey.Usage
				}
				if credential.MiMoConsole != nil {
					credentialView.HasMiMoConsoleCookie = true
					credentialView.MiMoTokenPlan = mimoTokenPlanView(credential.MiMoConsole)
				}
				view.Credentials = append(view.Credentials, credentialView)
			}
			for _, channel := range deps.CfgManager.GetAccountChannels(account.AccountUID) {
				view.Channels = append(view.Channels, managedAccountChannelView{
					Kind: channel.Kind, ChannelUID: channel.Upstream.ChannelUID, Name: channel.Upstream.Name,
					ServiceType: channel.Upstream.ServiceType, Status: channel.Upstream.Status,
				})
			}
			if deps.Runner != nil && deps.Runner.store != nil {
				view.EndpointCount = len(deps.Runner.store.ListByAccount(account.AccountUID))
			}
			accounts = append(accounts, view)
		}
		c.JSON(http.StatusOK, gin.H{"accounts": accounts})
	}
}

func mimoTokenPlanView(source *config.MiMoConsoleCredential) *managedMiMoTokenPlanView {
	if source == nil {
		return nil
	}
	return &managedMiMoTokenPlanView{
		PlanCode: source.PlanCode, PlanName: source.PlanName, CurrentPeriodEnd: source.CurrentPeriodEnd,
		Expired: source.Expired, MonthUsage: source.MonthUsage, CurrentUsage: source.CurrentUsage, ValidatedAt: source.ValidatedAt,
	}
}

func handleSetMiMoConsoleCookie(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Cookie         string `json:"cookie"`
			AdoptCookieKey bool   `json:"adoptCookieKey"`
		}
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		accountUID, credentialUID := strings.TrimSpace(c.Param("accountUid")), strings.TrimSpace(c.Param("credentialUid"))
		credential, ok := deps.CfgManager.GetManagedAccountCredential(accountUID, credentialUID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "推理 Key 凭证不存在"})
			return
		}
		channels := deps.CfgManager.GetAccountChannels(accountUID)
		if len(channels) == 0 || channels[0].Upstream.ProviderID != "mimo" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅 MiMo 自动托管账号支持绑定控制台 Cookie"})
			return
		}
		verification, err := mimoConsoleClient(deps).Verify(c.Request.Context(), req.Cookie)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		matches := mimoKeysEqual(credential.APIKey, verification.APIKey)
		if !matches && !req.AdoptCookieKey {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Cookie 所属 Token Plan Key 与当前渠道 Key 不一致",
				"code":  "mimo_cookie_key_mismatch", "currentKeyMask": utils.MaskAPIKey(credential.APIKey),
				"cookieKeyMask": utils.MaskAPIKey(verification.APIKey),
			})
			return
		}
		replacementKey := ""
		if !matches {
			replacementKey = verification.APIKey
		}
		if err := deps.CfgManager.BindManagedAccountMiMoConsole(accountUID, credentialUID, replacementKey, verification.Snapshot); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		started := 0
		if replacementKey != "" {
			for _, channel := range deps.CfgManager.GetAccountChannels(accountUID) {
				if triggerDiscoveryForChannel(deps, channel.Kind, channel.Upstream.ChannelUID) {
					started++
				}
			}
		}
		response := gin.H{
			"accountUid": accountUID, "credentialUid": credentialUID, "keyAdopted": replacementKey != "",
			"keyMask": utils.MaskAPIKey(verification.APIKey), "tokenPlan": mimoTokenPlanView(&verification.Snapshot),
			"discoveryStarted": started,
		}
		if replacementKey != "" {
			response["adoptedApiKey"] = replacementKey
		}
		c.JSON(http.StatusOK, response)
	}
}

func handleRefreshMiMoConsoleCookie(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		accountUID, credentialUID := strings.TrimSpace(c.Param("accountUid")), strings.TrimSpace(c.Param("credentialUid"))
		credential, ok := deps.CfgManager.GetManagedAccountCredential(accountUID, credentialUID)
		if !ok || credential.MiMoConsole == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "未绑定 MiMo 控制台 Cookie"})
			return
		}
		verification, err := mimoConsoleClient(deps).Verify(c.Request.Context(), credential.MiMoConsole.Cookie)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !mimoKeysEqual(credential.APIKey, verification.APIKey) {
			c.JSON(http.StatusConflict, gin.H{"error": "Cookie 所属 Key 已变化，请重新绑定并确认是否采用新 Key", "code": "mimo_cookie_key_mismatch"})
			return
		}
		if err := deps.CfgManager.BindManagedAccountMiMoConsole(accountUID, credentialUID, "", verification.Snapshot); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tokenPlan": mimoTokenPlanView(&verification.Snapshot)})
	}
}

func handleClearMiMoConsoleCookie(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		if err := deps.CfgManager.ClearManagedAccountMiMoConsole(strings.TrimSpace(c.Param("accountUid")), strings.TrimSpace(c.Param("credentialUid"))); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func mimoConsoleClient(deps *AutoManagedDeps) *MiMoConsoleClient {
	if deps != nil && deps.MiMoConsoleClient != nil {
		return deps.MiMoConsoleClient
	}
	return &MiMoConsoleClient{HTTPClient: &http.Client{Timeout: 10 * time.Second}}
}

func mimoKeysEqual(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func handleSetVolcengineAccessKey(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		var req struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		credentialUID := strings.TrimSpace(c.Param("credentialUid"))
		channels := deps.CfgManager.GetAccountChannels(accountUID)
		if len(channels) == 0 || channels[0].Upstream.ProviderID != "volcengine" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅火山方舟自动托管账号支持绑定 Access Key"})
			return
		}
		if _, ok := deps.CfgManager.GetManagedAccountCredential(accountUID, credentialUID); !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "推理 Key 凭证不存在"})
			return
		}
		pair := &config.VolcengineAccessKeyPair{AccessKeyID: strings.TrimSpace(req.AccessKeyID), SecretAccessKey: strings.TrimSpace(req.SecretAccessKey)}
		planClient := &volcenginePlanClient{}
		if deps.Runner != nil {
			planClient.Endpoint = deps.Runner.volcengineControlPlaneEndpoint
			planClient.HTTPClient = deps.Runner.client
		}
		hint := ""
		for _, channel := range channels {
			for _, keyConfig := range channel.Upstream.APIKeyConfigs {
				if keyConfig.CredentialUID == credentialUID {
					hint = volcenginePlanFromBaseURL(keyConfig.BaseURL)
					break
				}
			}
			if hint != "" {
				break
			}
		}
		plan, err := planClient.DetectPlan(c.Request.Context(), pair, hint)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := deps.CfgManager.SetManagedAccountVolcengineAccessKey(accountUID, credentialUID, req.AccessKeyID, req.SecretAccessKey); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := deps.CfgManager.SetManagedAccountVolcenginePlan(accountUID, credentialUID, plan.Plan, plan.Tier, plan.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 用量查询失败不阻断保存流程，只记录错误快照。
		usage, usageErr := planClient.FetchUsage(c.Request.Context(), pair, plan.Plan)
		if usageErr != nil {
			usage = &config.VolcenginePlanUsage{FetchedAt: time.Now(), Error: usageErr.Error()}
			log.Printf("[Volcengine-Usage] 查询套餐用量失败: %v", usageErr)
		}
		if err := deps.CfgManager.SetManagedAccountVolcenginePlanUsage(accountUID, credentialUID, usage); err != nil {
			log.Printf("[Volcengine-Usage] 保存套餐用量失败: %v", err)
		}
		started := 0
		for _, channel := range deps.CfgManager.GetAccountChannels(accountUID) {
			if triggerDiscoveryForChannel(deps, channel.Kind, channel.Upstream.ChannelUID) {
				started++
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"accountUid": accountUID, "credentialUid": credentialUID,
			"accessKeyIdMask": utils.MaskAPIKey(strings.TrimSpace(req.AccessKeyID)),
			"plan":            plan.Plan, "planTier": plan.Tier, "planStatus": plan.Status,
			"usage":            usage,
			"discoveryStarted": started,
		})
	}
}

func handleClearVolcengineAccessKey(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		credentialUID := strings.TrimSpace(c.Param("credentialUid"))
		if err := deps.CfgManager.ClearManagedAccountVolcengineAccessKey(accountUID, credentialUID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

const volcenginePlanUsageTTL = 5 * time.Minute

func handleRefreshVolcenginePlanUsage(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		credentialUID := strings.TrimSpace(c.Param("credentialUid"))
		credential, ok := deps.CfgManager.GetManagedAccountCredential(accountUID, credentialUID)
		if !ok || credential.VolcengineAccessKey == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "未绑定火山 Access Key"})
			return
		}
		accessKey := credential.VolcengineAccessKey
		// TTL 缓存：距上次成功查询未超过 TTL 时直接返回缓存。
		force := strings.EqualFold(strings.TrimSpace(c.Query("force")), "true")
		if !force && accessKey.Usage != nil && accessKey.Usage.Error == "" &&
			time.Since(accessKey.Usage.FetchedAt) < volcenginePlanUsageTTL {
			c.JSON(http.StatusOK, gin.H{"usage": accessKey.Usage, "cached": true})
			return
		}
		pair := &config.VolcengineAccessKeyPair{AccessKeyID: accessKey.AccessKeyID, SecretAccessKey: accessKey.SecretAccessKey}
		planClient := &volcenginePlanClient{}
		if deps.Runner != nil {
			planClient.Endpoint = deps.Runner.volcengineControlPlaneEndpoint
			planClient.HTTPClient = deps.Runner.client
		}
		usage, err := planClient.FetchUsage(c.Request.Context(), pair, accessKey.Plan)
		if err != nil {
			usage = &config.VolcenginePlanUsage{FetchedAt: time.Now(), Error: err.Error()}
			log.Printf("[Volcengine-Usage] 查询套餐用量失败: account=%s credential=%s err=%v", accountUID, credentialUID, err)
		}
		if saveErr := deps.CfgManager.SetManagedAccountVolcenginePlanUsage(accountUID, credentialUID, usage); saveErr != nil {
			log.Printf("[Volcengine-Usage] 保存套餐用量失败: %v", saveErr)
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "usage": usage})
			return
		}
		c.JSON(http.StatusOK, gin.H{"usage": usage, "cached": false})
	}
}

func handleDeleteAccount(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		channels := deps.CfgManager.GetAccountChannels(accountUID)
		if len(channels) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
			return
		}
		for _, channel := range channels {
			if !channel.Upstream.AutoManaged {
				c.JSON(http.StatusConflict, gin.H{"error": "账号包含非自动托管渠道，拒绝级联删除"})
				return
			}
		}
		removed, err := deps.CfgManager.DeleteAccountChannels(accountUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if deps.Runner != nil && deps.Runner.store != nil {
			if err := deps.Runner.store.DeleteByAccount(accountUID); err != nil {
				log.Printf("[AutoManaged-Delete] 清理账号画像失败 account=%s: %v", accountUID, err)
			}
			for _, channelUID := range removed {
				for _, profile := range deps.Runner.store.ListByChannel(channelUID) {
					if err := deps.Runner.store.Delete(profile.EndpointUID); err != nil {
						log.Printf("[AutoManaged-Delete] 清理旧渠道画像失败 channel=%s endpoint=%s: %v", channelUID, profile.EndpointUID, err)
					}
				}
			}
		}
		log.Printf("[AutoManaged-Delete] 已删除自动托管账号 account=%s channels=%d", accountUID, len(removed))
		c.JSON(http.StatusOK, gin.H{"accountUid": accountUID, "deletedChannels": len(removed)})
	}
}

type updateAccountRequest struct {
	Name    string   `json:"name"`
	APIKeys []string `json:"apiKeys"`
}

type updateAccountResponse struct {
	AccountUID       string `json:"accountUid"`
	KeyCount         int    `json:"keyCount"`
	ChannelCount     int    `json:"channelCount"`
	DiscoveryStarted int    `json:"discoveryStarted"`
}

// handleUpdateAccount 在账号范围原子替换凭证集合，并只为新增 Key 探测各协议 route。
func handleUpdateAccount(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		response, status, err := applyManagedAccountUpdate(c.Request.Context(), deps, strings.TrimSpace(c.Param("accountUid")), req)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
	}
}

func applyManagedAccountUpdate(ctx context.Context, deps *AutoManagedDeps, accountUID string, req updateAccountRequest) (updateAccountResponse, int, error) {
	if deps == nil || deps.CfgManager == nil {
		return updateAccountResponse{}, http.StatusServiceUnavailable, fmt.Errorf("配置管理器不可用")
	}
	req.APIKeys = uniqueNonEmptyStrings(req.APIKeys)
	if len(req.APIKeys) == 0 {
		return updateAccountResponse{}, http.StatusBadRequest, fmt.Errorf("apiKeys 不能为空")
	}
	channels := deps.CfgManager.GetAccountChannels(accountUID)
	if len(channels) == 0 {
		return updateAccountResponse{}, http.StatusNotFound, fmt.Errorf("账号不存在")
	}
	providerID := channels[0].Upstream.ProviderID
	tmpl, ok := config.GetProviderTemplate(providerID)
	if !ok || providerID == "" {
		return updateAccountResponse{}, http.StatusBadRequest, fmt.Errorf("仅 provider 自动托管账号支持账号级更新")
	}
	updates, status, err := planManagedAccountUpdates(ctx, accountUID, req, channels, tmpl, len(channels))
	if err != nil {
		return updateAccountResponse{}, status, err
	}
	if err := deps.CfgManager.UpdateAccountChannels(accountUID, updates); err != nil {
		return updateAccountResponse{}, http.StatusInternalServerError, err
	}
	started := 0
	for _, accountChannel := range channels {
		if triggerDiscoveryForChannel(deps, accountChannel.Kind, accountChannel.Upstream.ChannelUID) {
			started++
		}
	}
	return updateAccountResponse{AccountUID: accountUID, KeyCount: len(req.APIKeys), ChannelCount: len(channels), DiscoveryStarted: started}, http.StatusOK, nil
}

func planManagedAccountUpdates(
	ctx context.Context,
	accountUID string,
	req updateAccountRequest,
	channels []config.AccountChannel,
	tmpl *config.ProviderTemplate,
	totalRouteCount int,
) ([]config.AccountChannelUpdate, int, error) {
	if len(channels) == 0 || tmpl == nil {
		return nil, http.StatusNotFound, fmt.Errorf("账号不存在")
	}
	req.APIKeys = uniqueNonEmptyStrings(req.APIKeys)
	if len(req.APIKeys) == 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("apiKeys 不能为空")
	}
	providerID := channels[0].Upstream.ProviderID
	baseName := strings.TrimSpace(req.Name)
	if baseName == "" {
		baseName = strings.TrimSuffix(channels[0].Upstream.Name, accountRouteSuffix(channels[0].Kind))
	}
	updates := make([]config.AccountChannelUpdate, 0, len(channels))
	for _, accountChannel := range channels {
		channel := accountChannel.Upstream
		if !channel.AutoManaged || channel.ProviderID != providerID {
			return nil, http.StatusConflict, fmt.Errorf("账号包含非托管渠道或 provider 不一致")
		}
		route, found := providerRouteForChannel(tmpl, accountChannel.Kind, channel.ServiceType)
		if !found {
			return nil, http.StatusConflict, fmt.Errorf("provider %s 缺少 %s route", providerID, accountChannel.Kind)
		}
		existing := make(map[string]config.APIKeyConfig, len(channel.APIKeyConfigs))
		for _, keyConfig := range channel.APIKeyConfigs {
			existing[keyConfig.Key] = keyConfig
		}
		var added []string
		for _, key := range req.APIKeys {
			if _, exists := existing[key]; !exists {
				added = append(added, key)
			}
		}
		if len(added) > 0 {
			verified, _, err := verifyProviderRouteKeys(ctx, tmpl, route, added)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}
			for _, keyConfig := range verified {
				existing[keyConfig.Key] = keyConfig
			}
		}
		configs := make([]config.APIKeyConfig, 0, len(req.APIKeys))
		baseURLs := make([]string, 0, len(req.APIKeys))
		for _, key := range req.APIKeys {
			keyConfig, exists := existing[key]
			if !exists {
				keyConfig = config.APIKeyConfig{Key: key, BaseURL: channel.BoundBaseURLForKey(key)}
			}
			if keyConfig.CredentialUID == "" {
				keyConfig.CredentialUID = config.GenerateCredentialUID(accountUID, key)
			}
			configs = append(configs, keyConfig)
			if keyConfig.BaseURL != "" {
				baseURLs = append(baseURLs, keyConfig.BaseURL)
			}
		}
		updates = append(updates, config.AccountChannelUpdate{
			ChannelUID: channel.ChannelUID, Name: providerRouteName(baseName, route, totalRouteCount > 1),
			APIKeys: append([]string(nil), req.APIKeys...), APIKeyConfig: configs, BaseURLs: uniqueNonEmptyStrings(baseURLs),
		})
	}
	return updates, http.StatusOK, nil
}

func handleAddAccountCredentials(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		var req struct {
			APIKeys []string `json:"apiKeys"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		channels := deps.CfgManager.GetAccountChannels(accountUID)
		if len(channels) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
			return
		}
		desired := append([]string(nil), channels[0].Upstream.APIKeys...)
		desired = append(desired, req.APIKeys...)
		response, status, err := applyManagedAccountUpdate(c.Request.Context(), deps, accountUID, updateAccountRequest{APIKeys: desired})
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, response)
	}
}

func handlePatchAccountCredentials(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		var req struct {
			AddAPIKeys           []string `json:"addApiKeys"`
			RemoveCredentialUIDs []string `json:"removeCredentialUids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		channels := deps.CfgManager.GetAccountChannels(accountUID)
		if len(channels) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
			return
		}
		removeSet := make(map[string]bool, len(req.RemoveCredentialUIDs))
		for _, uid := range req.RemoveCredentialUIDs {
			removeSet[strings.TrimSpace(uid)] = true
		}
		removedKeys := make(map[string]string)
		var desired []string
		for _, keyConfig := range channels[0].Upstream.APIKeyConfigs {
			if removeSet[keyConfig.CredentialUID] {
				removedKeys[keyConfig.CredentialUID] = keyConfig.Key
				continue
			}
			desired = append(desired, keyConfig.Key)
		}
		if len(removedKeys) != len(removeSet) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "包含不存在的 credentialUid"})
			return
		}
		desired = append(desired, req.AddAPIKeys...)
		if len(uniqueNonEmptyStrings(desired)) == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "账号至少需要保留一个 Key"})
			return
		}
		response, status, err := applyManagedAccountUpdate(c.Request.Context(), deps, accountUID, updateAccountRequest{APIKeys: desired})
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		cleanupRemovedCredentialProfiles(deps, accountUID, channels, removedKeys)
		c.JSON(http.StatusOK, response)
	}
}

func cleanupRemovedCredentialProfiles(deps *AutoManagedDeps, accountUID string, channels []config.AccountChannel, removed map[string]string) {
	if deps == nil || deps.Runner == nil || deps.Runner.store == nil {
		return
	}
	for credentialUID, apiKey := range removed {
		if err := deps.Runner.store.DeleteByCredential(accountUID, credentialUID); err != nil {
			log.Printf("[AutoManaged-CredentialDelete] 清理凭证画像失败 account=%s credential=%s: %v", accountUID, credentialUID, err)
		}
		keyHash := KeyHashFromAPIKey(apiKey)
		for _, channel := range channels {
			for _, profile := range deps.Runner.store.ListByChannel(channel.Upstream.ChannelUID) {
				if profile.CredentialUID == credentialUID || profile.KeyHash == keyHash {
					if err := deps.Runner.store.Delete(profile.EndpointUID); err != nil {
						log.Printf("[AutoManaged-CredentialDelete] 清理旧凭证画像失败 endpoint=%s: %v", profile.EndpointUID, err)
					}
				}
			}
		}
	}
}

func handleDeleteAccountCredential(deps *AutoManagedDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.CfgManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "配置管理器不可用"})
			return
		}
		accountUID := strings.TrimSpace(c.Param("accountUid"))
		credentialUID := strings.TrimSpace(c.Param("credentialUid"))
		channels := deps.CfgManager.GetAccountChannels(accountUID)
		if len(channels) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
			return
		}
		var removeKey string
		for _, keyConfig := range channels[0].Upstream.APIKeyConfigs {
			if keyConfig.CredentialUID == credentialUID {
				removeKey = keyConfig.Key
				break
			}
		}
		if removeKey == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
			return
		}
		var desired []string
		for _, key := range channels[0].Upstream.APIKeys {
			if key != removeKey {
				desired = append(desired, key)
			}
		}
		if len(desired) == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "不能删除账号的最后一个 Key，请删除整个账号"})
			return
		}
		response, status, err := applyManagedAccountUpdate(c.Request.Context(), deps, accountUID, updateAccountRequest{APIKeys: desired})
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		cleanupRemovedCredentialProfiles(deps, accountUID, channels, map[string]string{credentialUID: removeKey})
		c.JSON(http.StatusOK, response)
	}
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func providerRouteForChannel(tmpl *config.ProviderTemplate, kind, serviceType string) (config.ProviderRoute, bool) {
	for _, route := range tmpl.AutoAddRoutes() {
		if route.ChannelKind == kind && route.ServiceType == serviceType {
			return route, true
		}
	}
	return config.ProviderRoute{}, false
}

func accountRouteSuffix(kind string) string {
	switch kind {
	case "messages":
		return "-claude"
	case "chat":
		return "-chat"
	case "responses":
		return "-codex"
	case "gemini":
		return "-gemini"
	default:
		return "-" + kind
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
		req.APIKeys = uniqueNonEmptyStrings(req.APIKeys)
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
			AccountUID:    config.GenerateAccountUID(),
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
			AccountUID:       upstream.AccountUID,
			ChannelUID:       channelUID,
			Index:            index,
			DiscoveryStarted: discoveryStarted,
			Channels: []AutoAddChannelResult{{
				AccountUID:       upstream.AccountUID,
				ChannelKind:      kind,
				ChannelUID:       channelUID,
				Index:            index,
				Name:             upstream.Name,
				ServiceType:      upstream.ServiceType,
				DiscoveryStarted: discoveryStarted,
			}},
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
	if accountUID := managedAccountUIDForProvider(deps.CfgManager.GetConfig(), tmpl.ProviderID); accountUID != "" {
		appendCredentialsToProviderAccount(c, deps, requestKind, accountUID, req.APIKeys)
		return
	}

	// provider 账号是按 providerId 唯一的官方托管资源，不接受客户端自定义名称。
	// 使用 providerId 作为保留名（如 mimo），前端展示时再映射为品牌友好名。
	baseName := tmpl.ProviderID

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
	accountUID := config.GenerateAccountUID()
	additions := make([]config.AccountChannelAddition, 0, len(planned))
	for _, item := range planned {
		for i := range item.keyConfigs {
			item.keyConfigs[i].CredentialUID = config.GenerateCredentialUID(accountUID, item.keyConfigs[i].Key)
		}
		upstream := config.UpstreamConfig{
			Name:          item.name,
			AccountUID:    accountUID,
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
		additions = append(additions, config.AccountChannelAddition{Kind: item.route.ChannelKind, Upstream: upstream})
	}
	if err := deps.CfgManager.ApplyAccountChannelChanges(accountUID, nil, additions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建渠道失败: %v", err)})
		return
	}

	cfg = deps.CfgManager.GetConfig()
	results := make([]AutoAddChannelResult, 0, len(additions))
	for _, addition := range additions {
		upstream := addition.Upstream
		index := findChannelIndexByUID(getChannelSlice(cfg, addition.Kind), upstream.ChannelUID)
		discoveryStarted := triggerDiscoveryForChannel(deps, addition.Kind, upstream.ChannelUID)
		results = append(results, AutoAddChannelResult{
			AccountUID:       accountUID,
			ChannelKind:      addition.Kind,
			ChannelUID:       upstream.ChannelUID,
			Index:            index,
			Name:             upstream.Name,
			ServiceType:      upstream.ServiceType,
			DiscoveryStarted: discoveryStarted,
		})
		log.Printf("[AutoManaged-Add] 创建自动托管 provider 渠道: provider=%s kind=%s serviceType=%s name=%s uid=%s",
			tmpl.ProviderID, addition.Kind, upstream.ServiceType, upstream.Name, upstream.ChannelUID)
	}

	primary := primaryAutoAddResult(results, requestKind)
	c.JSON(http.StatusCreated, AutoAddResponse{
		AccountUID:       accountUID,
		ChannelUID:       primary.ChannelUID,
		Index:            primary.Index,
		DiscoveryStarted: primary.DiscoveryStarted,
		Channels:         results,
	})
}

func managedAccountUIDForProvider(cfg config.Config, providerID string) string {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ""
	}
	uid := ""
	for _, account := range cfg.ManagedAccounts {
		if account.ProviderID == providerID {
			uid = account.AccountUID
		}
	}
	return uid
}

func appendCredentialsToProviderAccount(c *gin.Context, deps *AutoManagedDeps, requestKind, accountUID string, apiKeys []string) {
	channels := deps.CfgManager.GetAccountChannels(accountUID)
	if len(channels) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider 账号不存在"})
		return
	}
	var desired []string
	for _, channel := range channels {
		desired = append(desired, channel.Upstream.APIKeys...)
	}
	desired = uniqueNonEmptyStrings(append(desired, apiKeys...))
	tmpl, ok := config.GetProviderTemplate(channels[0].Upstream.ProviderID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider 模板不存在"})
		return
	}
	additions, status, err := planProviderAccountRouteAdditions(c.Request.Context(), deps.CfgManager.GetConfig(), tmpl, accountUID, desired, channels)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	accountName := strings.TrimSuffix(channels[0].Upstream.Name, accountRouteSuffix(channels[0].Kind))
	updates, status, err := planManagedAccountUpdates(c.Request.Context(), accountUID, updateAccountRequest{
		Name: accountName, APIKeys: desired,
	}, channels, tmpl, len(channels)+len(additions))
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if err := deps.CfgManager.ApplyAccountChannelChanges(accountUID, updates, additions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, addition := range additions {
		log.Printf("[AutoManaged-Add] 已补齐 provider 账号渠道: provider=%s account=%s kind=%s serviceType=%s uid=%s",
			tmpl.ProviderID, accountUID, addition.Kind, addition.Upstream.ServiceType, addition.Upstream.ChannelUID)
	}

	channels = deps.CfgManager.GetAccountChannels(accountUID)
	discoveryStarted := make(map[string]bool, len(channels))
	for _, channel := range channels {
		discoveryStarted[channel.Upstream.ChannelUID] = triggerDiscoveryForChannel(deps, channel.Kind, channel.Upstream.ChannelUID)
	}
	results := make([]AutoAddChannelResult, 0, len(channels))
	cfg := deps.CfgManager.GetConfig()
	for _, channel := range channels {
		index := findChannelIndexByUID(getChannelSlice(cfg, channel.Kind), channel.Upstream.ChannelUID)
		results = append(results, AutoAddChannelResult{
			AccountUID: accountUID, ChannelKind: channel.Kind, ChannelUID: channel.Upstream.ChannelUID,
			Index: index, Name: channel.Upstream.Name, ServiceType: channel.Upstream.ServiceType,
			DiscoveryStarted: discoveryStarted[channel.Upstream.ChannelUID],
		})
	}
	primary := primaryAutoAddResult(results, requestKind)
	log.Printf("[AutoManaged-Add] 已向 provider 账号追加凭证: provider=%s account=%s keys=%d", channels[0].Upstream.ProviderID, accountUID, len(apiKeys))
	c.JSON(http.StatusOK, AutoAddResponse{
		AccountUID: accountUID, ChannelUID: primary.ChannelUID, Index: primary.Index,
		DiscoveryStarted: primary.DiscoveryStarted, Channels: results,
	})
}

func planProviderAccountRouteAdditions(
	ctx context.Context,
	cfg config.Config,
	tmpl *config.ProviderTemplate,
	accountUID string,
	apiKeys []string,
	existing []config.AccountChannel,
) ([]config.AccountChannelAddition, int, error) {
	if tmpl == nil || len(existing) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("provider 账号不存在")
	}
	missing := missingProviderAccountRoutes(tmpl, existing)
	if len(missing) == 0 {
		return nil, http.StatusOK, nil
	}

	baseName := strings.TrimSuffix(existing[0].Upstream.Name, accountRouteSuffix(existing[0].Kind))
	allRoutes := tmpl.AutoAddRoutes()
	for _, route := range missing {
		name := providerRouteName(baseName, route, len(allRoutes) > 1)
		if channelNameExists(getChannelSlice(cfg, route.ChannelKind), name) {
			return nil, http.StatusConflict, fmt.Errorf("渠道名称 '%s' 已存在", name)
		}
	}

	existingKeys := make(map[string]bool)
	for _, channel := range existing {
		for _, apiKey := range channel.Upstream.APIKeys {
			existingKeys[apiKey] = true
		}
	}
	var addedKeys []string
	for _, apiKey := range uniqueNonEmptyStrings(apiKeys) {
		if !existingKeys[apiKey] {
			addedKeys = append(addedKeys, apiKey)
		}
	}

	now := time.Now()
	additions := make([]config.AccountChannelAddition, 0, len(missing))
	affinities := providerKeyCandidateAffinities(tmpl, existing)
	for _, route := range missing {
		keyConfigs, _, err := bindProviderRouteKeysWithAffinities(tmpl, route, apiKeys, affinities)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		if len(addedKeys) > 0 {
			verified, _, err := verifyProviderRouteKeys(ctx, tmpl, route, addedKeys)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}
			verifiedByKey := make(map[string]config.APIKeyConfig, len(verified))
			for _, keyConfig := range verified {
				verifiedByKey[keyConfig.Key] = keyConfig
			}
			for i := range keyConfigs {
				if verifiedConfig, ok := verifiedByKey[keyConfigs[i].Key]; ok {
					keyConfigs[i] = verifiedConfig
				}
			}
		}
		baseURLs := make([]string, 0, len(keyConfigs))
		for i := range keyConfigs {
			keyConfigs[i].CredentialUID = config.GenerateCredentialUID(accountUID, keyConfigs[i].Key)
			baseURLs = append(baseURLs, keyConfigs[i].BaseURL)
		}
		upstream := config.UpstreamConfig{
			Name:          providerRouteName(baseName, route, len(allRoutes) > 1),
			AccountUID:    accountUID,
			ChannelUID:    config.GenerateChannelUID(),
			ServiceType:   route.ServiceType,
			Status:        "active",
			AutoManaged:   true,
			AutoManagedAt: &now,
			ProviderID:    tmpl.ProviderID,
			OriginType:    tmpl.OriginType,
			OriginTier:    tmpl.OriginTier,
			BaseURL:       baseURLs[0],
			BaseURLs:      uniqueNonEmptyStrings(baseURLs),
			APIKeys:       append([]string(nil), apiKeys...),
			APIKeyConfigs: keyConfigs,
		}
		additions = append(additions, config.AccountChannelAddition{Kind: route.ChannelKind, Upstream: upstream})
	}
	return additions, http.StatusOK, nil
}

// bindProviderRouteKeys 为已验证过的托管账号凭证选择 route 的首选官方端点。
// 旧账号补齐新协议时不应重新发送模型请求；端点可达性由随后触发的 discovery 异步确认。
func bindProviderRouteKeys(tmpl *config.ProviderTemplate, route config.ProviderRoute, apiKeys []string) ([]config.APIKeyConfig, []string, error) {
	return bindProviderRouteKeysWithAffinities(tmpl, route, apiKeys, nil)
}

type providerCandidateAffinity struct {
	PlanTag  string
	Region   string
	Priority int
}

func bindProviderRouteKeysWithAffinities(
	tmpl *config.ProviderTemplate,
	route config.ProviderRoute,
	apiKeys []string,
	affinities map[string]providerCandidateAffinity,
) ([]config.APIKeyConfig, []string, error) {
	if tmpl == nil {
		return nil, nil, fmt.Errorf("provider 模板为空")
	}
	apiKeys = uniqueNonEmptyStrings(apiKeys)
	if len(apiKeys) == 0 {
		return nil, nil, fmt.Errorf("apiKeys 不能为空")
	}

	keyConfigs := make([]config.APIKeyConfig, 0, len(apiKeys))
	baseURLs := make([]string, 0, len(route.Candidates))
	seenBaseURL := make(map[string]bool)
	for _, apiKey := range apiKeys {
		candidates := tmpl.CandidatesForRouteKey(route, apiKey)
		if len(candidates) == 0 || strings.TrimSpace(candidates[0].BaseURL) == "" {
			return nil, nil, fmt.Errorf("provider %s 无可用候选端点（kind=%s serviceType=%s）", tmpl.ProviderID, route.ChannelKind, route.ServiceType)
		}
		selected := candidates[0]
		if affinity, ok := affinities[apiKey]; ok {
			for _, candidate := range candidates {
				if candidate.PlanTag == affinity.PlanTag && candidate.Region == affinity.Region && candidate.Priority == affinity.Priority {
					selected = candidate
					break
				}
			}
		}
		baseURL := strings.TrimSpace(selected.BaseURL)
		keyConfigs = append(keyConfigs, config.APIKeyConfig{Key: apiKey, BaseURL: baseURL})
		if !seenBaseURL[baseURL] {
			seenBaseURL[baseURL] = true
			baseURLs = append(baseURLs, baseURL)
		}
	}
	return keyConfigs, baseURLs, nil
}

func providerKeyCandidateAffinities(tmpl *config.ProviderTemplate, existing []config.AccountChannel) map[string]providerCandidateAffinity {
	affinities := make(map[string]providerCandidateAffinity)
	if tmpl == nil {
		return affinities
	}
	for _, accountChannel := range existing {
		route, ok := providerRouteForChannel(tmpl, accountChannel.Kind, accountChannel.Upstream.ServiceType)
		if !ok {
			continue
		}
		for _, apiKey := range accountChannel.Upstream.APIKeys {
			if _, exists := affinities[apiKey]; exists {
				continue
			}
			boundURL := utils.CanonicalBaseURL(accountChannel.Upstream.BoundBaseURLForKey(apiKey), route.ServiceType)
			if boundURL == "" {
				continue
			}
			for _, candidate := range tmpl.CandidatesForRouteKey(route, apiKey) {
				if utils.CanonicalBaseURL(candidate.BaseURL, route.ServiceType) == boundURL {
					affinities[apiKey] = providerCandidateAffinity{
						PlanTag: candidate.PlanTag, Region: candidate.Region, Priority: candidate.Priority,
					}
					break
				}
			}
		}
	}
	return affinities
}

func missingProviderAccountRoutes(tmpl *config.ProviderTemplate, existing []config.AccountChannel) []config.ProviderRoute {
	present := make(map[string]bool, len(existing))
	for _, channel := range existing {
		present[channel.Kind+"\x00"+channel.Upstream.ServiceType] = true
	}
	var missing []config.ProviderRoute
	for _, route := range tmpl.AutoAddRoutes() {
		if !present[route.ChannelKind+"\x00"+route.ServiceType] {
			missing = append(missing, route)
		}
	}
	return missing
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
	case "gemini":
		return baseName + "-gemini"
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
