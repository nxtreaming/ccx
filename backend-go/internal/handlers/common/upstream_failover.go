// Package common 提供 handlers 模块的公共功能
package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/autopilot"
	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/keypool"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/middleware"
	"github.com/BenedictKing/ccx/internal/providers"
	"github.com/BenedictKing/ccx/internal/ratelimit"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/BenedictKing/ccx/internal/types"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/BenedictKing/ccx/internal/warmup"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

const (
	upstreamAccountPoolCooldown   = time.Minute
	upstreamOverloadedCooldown    = 15 * time.Second
	halfOpenProbeWaitTimeout      = 5 * time.Second
	halfOpenProbePollInterval     = 100 * time.Millisecond
	shortEmptyResponseRetryWindow = 10 * time.Second
)

// isClientSideError 判断错误是否由客户端明确取消（不应计入渠道失败）
// 仅识别 context.Canceled，broken pipe/connection reset 视为连接故障需要 failover
func isClientSideError(err error) bool {
	if err == nil {
		return false
	}
	// 只有 context.Canceled 才是明确的客户端取消意图
	return errors.Is(err, context.Canceled)
}

// NextAPIKeyFunc 返回下一个可用 API key（按 failover 策略）
type NextAPIKeyFunc func(upstream *config.UpstreamConfig, failedKeys map[string]bool) (string, error)

// BuildRequestFunc 构建上游请求（upstreamCopy.BaseURL 已写入当前尝试的 BaseURL）
type BuildRequestFunc func(c *gin.Context, upstreamCopy *config.UpstreamConfig, apiKey string) (*http.Request, error)

// DeprioritizeKeyFunc 对 quota 相关失败的 key 做降级（实现可选择是否记录日志）
type DeprioritizeKeyFunc func(apiKey string)

// HandleSuccessFunc 处理成功响应（负责写回客户端），并返回 usage（可为 nil）
// 注意：实现方需要自行关闭 resp.Body（与现有 handlers 保持一致）。
// actualRequestBody 为本次实际转发给上游的请求体，可用于 usage 估算等后处理。
type HandleSuccessFunc func(c *gin.Context, resp *http.Response, upstreamCopy *config.UpstreamConfig, apiKey string, actualRequestBody []byte) (*types.Usage, error)

// TryUpstreamOption 为渠道内 BaseURL/Key 轮转补充可选行为。
type TryUpstreamOption func(*tryUpstreamOptions)

type tryUpstreamOptions struct {
	channelLogOptions []ChannelLogOption
	endpointPolicy    *autopilot.EndpointAttemptPolicy // endpoint 级策略（nil 时不注入）
}

// WithSelectionTrace 将调度器的选择摘要写入后续渠道请求日志。
func WithSelectionTrace(selection *scheduler.SelectionResult) TryUpstreamOption {
	return func(opts *tryUpstreamOptions) {
		if opts == nil || selection == nil {
			return
		}
		opts.channelLogOptions = append(opts.channelLogOptions, WithChannelSelectionTrace(
			selection.Reason,
			scheduler.FormatSelectionTraceSummary(selection.Trace, 4),
		))
	}
}

// WithEndpointAttemptPolicy 将 autopilot EndpointAttemptPolicy 注入 TryUpstreamWithAllKeys。
// nil policy 时等同于不注入（fail-open）。
// panic 防护：policy 函数 panic 时回退原列表，记日志。
func WithEndpointAttemptPolicy(policy *autopilot.EndpointAttemptPolicy) TryUpstreamOption {
	return func(opts *tryUpstreamOptions) {
		if opts == nil || policy == nil {
			return
		}
		opts.endpointPolicy = policy
	}
}

// ── Autopilot 包级钩子（可选注入，nil 时默认行为不变）──

// endpointPolicyProviderHook 可选的 endpoint policy 提供者。
// 由 main.go 在 autopilot 初始化后注入；handlers 通过 TryUpstreamOption 传入 policy。
// 签名：(c, model, upstream) → *EndpointAttemptPolicy（nil 表示不注入）。
var endpointPolicyProviderHook func(c *gin.Context, model string, upstream *config.UpstreamConfig) *autopilot.EndpointAttemptPolicy

// SetEndpointPolicyProviderHook 设置 endpoint policy 提供者钩子。
// 由 main.go 在 autopilot 初始化后调用；nil 表示不注入。
func SetEndpointPolicyProviderHook(hook func(c *gin.Context, model string, upstream *config.UpstreamConfig) *autopilot.EndpointAttemptPolicy) {
	endpointPolicyProviderHook = hook
}

// notifyEndpointResultHook 可选的 endpoint 请求结果通知器。
// 由 main.go 在 autopilot 初始化后注入；用于实时更新 FastDecayScorer。
// 签名：(endpointUID, success)。
var notifyEndpointResultHook func(endpointUID string, success bool)

// SetNotifyEndpointResultHook 设置 endpoint 结果通知钩子。
// 由 main.go 在 autopilot 初始化后调用；nil 表示不通知。
func SetNotifyEndpointResultHook(hook func(endpointUID string, success bool)) {
	notifyEndpointResultHook = hook
}

// PostSuccessfulProxyHook 可选的代理成功后回调。
// 由 main.go 在 autopilot A/B 测试初始化后注入。
// 在主响应已写回客户端之后、函数返回之前触发。
// 签名：(channelKind, model, channelUID string, statusCode int, latencyMs int64, bodyBytes []byte)。
// 回调函数不应阻塞（如果需要异步操作，由回调内部自行管理）。
var postSuccessfulProxyHook func(channelKind, model, channelUID string, statusCode int, latencyMs int64, bodyBytes []byte)

// SetPostSuccessfulProxyHook 设置代理成功后回调钩子。
// 由 main.go 在 autopilot ABTestSampler 初始化后调用；nil 表示不触发。
func SetPostSuccessfulProxyHook(hook func(channelKind, model, channelUID string, statusCode int, latencyMs int64, bodyBytes []byte)) {
	postSuccessfulProxyHook = hook
}

func shouldNormalizeMetadataUserID(kind scheduler.ChannelKind, upstream *config.UpstreamConfig) bool {
	if upstream == nil {
		return false
	}
	if kind != scheduler.ChannelKindMessages {
		return false
	}
	return upstream.IsNormalizeMetadataUserIDEnabled()
}

func shouldStripBillingHeader(kind scheduler.ChannelKind, upstream *config.UpstreamConfig) bool {
	if upstream == nil {
		return false
	}
	if kind != scheduler.ChannelKindMessages {
		return false
	}
	return upstream.IsStripBillingHeaderEnabled()
}

// TryUpstreamWithAllKeys 尝试一个 upstream 的所有 BaseURL + Key（纯 failover）
// 返回:
//   - handled: 是否已向客户端写回响应（成功或非 failover 错误）
//   - successKey: 成功的 key（仅 handled=true 且成功时有值）
//   - successBaseURLIdx: 成功 BaseURL 的原始索引（用于指标记录）
//   - failoverErr: 最后一次可故障转移的上游错误（用于多渠道聚合错误）
//   - usage: usage 统计（可能为 nil）
func TryUpstreamWithAllKeys(
	c *gin.Context,
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	channelScheduler *scheduler.ChannelScheduler,
	kind scheduler.ChannelKind,
	apiType string,
	metricsManager *metrics.MetricsManager,
	upstream *config.UpstreamConfig,
	urlResults []warmup.URLLatencyResult,
	requestBody []byte,
	contextRequirement *scheduler.ContextRequirement,
	isStream bool,
	nextAPIKey NextAPIKeyFunc,
	buildRequest BuildRequestFunc,
	deprioritizeKey DeprioritizeKeyFunc,
	markURLFailure func(url string),
	markURLSuccess func(url string),
	handleSuccess HandleSuccessFunc,
	model string,
	operation string,
	channelIndex int,
	channelLogStore *metrics.ChannelLogStore,
	opts ...TryUpstreamOption,
) (handled bool, successKey string, successBaseURLIdx int, failoverErr *FailoverError, usage *types.Usage, lastError error) {
	if upstream == nil || len(upstream.APIKeys) == 0 {
		return false, "", 0, nil, nil, nil
	}
	if metricsManager == nil {
		return false, "", 0, nil, nil, nil
	}
	if nextAPIKey == nil || buildRequest == nil || handleSuccess == nil {
		return false, "", 0, nil, nil, nil
	}
	if len(urlResults) == 0 {
		return false, "", 0, nil, nil, nil
	}

	tryOpts := tryUpstreamOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&tryOpts)
		}
	}

	metricsServiceType := scheduler.NormalizedMetricsServiceType(kind, upstream.ServiceType)

	var lastFailoverError *FailoverError
	deprioritizeCandidates := make(map[string]bool)
	failedQuotaGroups := make(map[string]bool)
	probeAcquired := make(map[string]bool)
	// 当前持有的限速并发信号量释放函数（兜底：函数任意路径返回时释放，避免泄漏）
	var activeRateLimitRelease func()
	defer func() {
		if activeRateLimitRelease != nil {
			activeRateLimitRelease()
		}
		for key := range probeAcquired {
			parts := strings.SplitN(key, "|", 2)
			if len(parts) == 2 {
				metricsManager.ReleaseProbe(parts[0], parts[1], metricsServiceType)
			}
		}
	}()

	// 计算重定向后的模型（用于日志记录）
	redirectedModel := config.RedirectModel(model, upstream)
	capabilityRequestModel := model
	var originalModel string
	if redirectedModel != model {
		originalModel = model // 仅当发生重定向时记录原始模型
	}

	// 历史图片轮次限制：替换历史图片为占位符，避免不必要的 vision 回退
	if kind != scheduler.ChannelKindImages {
		effectiveLimit := resolveHistoricalImageTurnLimit(upstream)
		if effectiveLimit > 0 {
			if replaced, modified := StripHistoricalImagesWithContext(c, requestBody, effectiveLimit, envCfg.EnableRequestLogs, apiType); modified {
				requestBody = replaced
			}
		}
	}

	// Vision 能力检查：含图请求跳过不支持 vision 的渠道/模型
	if kind != scheduler.ChannelKindImages && HasImageContent(c, requestBody) {
		if upstream.NoVision {
			RequestLogf(c, "[%s-Vision] 跳过不支持视觉的渠道 [%d] %s", apiType, channelIndex, upstream.Name)
			return false, "", 0, nil, nil, fmt.Errorf("channel %s does not support vision", upstream.Name)
		}
		if isNoVisionModel(upstream, redirectedModel) {
			if upstream.VisionFallbackModel != "" {
				fallback := upstream.VisionFallbackModel
				RequestLogf(c, "[%s-Vision] 模型 %s 不支持视觉，使用 fallback: %s (渠道 [%d] %s)", apiType, redirectedModel, fallback, channelIndex, upstream.Name)
				if replaced, err := sjson.SetBytes(requestBody, "model", fallback); err == nil {
					requestBody = replaced
				}
				originalModel = model
				redirectedModel = fallback
				capabilityRequestModel = fallback
				if err := channelScheduler.ValidateUpstreamContext(kind, redirectedModel, upstream, contextRequirement); err != nil {
					RequestLogf(c, "[%s-Vision] fallback 模型 %s 不满足上下文需求，跳过渠道 [%d] %s: %v", apiType, redirectedModel, channelIndex, upstream.Name, err)
					return false, "", 0, nil, nil, err
				}
			} else {
				RequestLogf(c, "[%s-Vision] 模型 %s 不支持视觉且无 fallback，跳过渠道 [%d] %s", apiType, redirectedModel, channelIndex, upstream.Name)
				return false, "", 0, nil, nil, fmt.Errorf("model %s does not support vision", redirectedModel)
			}
		}
	}

	// ── EndpointAttemptPolicy: 步骤 1+2（FilterURLs + SortURLs）──
	// 对 urlResults 应用 policy 过滤/排序（设计 §4.6.2a）。
	// nil policy / hook 未设置 / panic 时均 fail-open，不影响现有逻辑。
	endpointPolicy := tryOpts.endpointPolicy
	if endpointPolicy == nil && endpointPolicyProviderHook != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[Autopilot-EndpointPolicy] endpointPolicyProviderHook panic: %v", r)
				}
			}()
			endpointPolicy = endpointPolicyProviderHook(c, model, upstream)
		}()
	}
	urlResults = applyPolicyToURLs(endpointPolicy, urlResults, apiType, c)

	for urlIdx, urlResult := range urlResults {
		currentBaseURL := urlResult.URL
		originalIdx := urlResult.OriginalIdx // 原始索引用于指标记录
		failedKeys := make(map[string]bool)  // 每个 BaseURL 重置失败 Key 列表
		maxRetries := len(upstream.APIKeys)
		shortEmptyRetried := make(map[string]bool)
		var retrySelection keypool.Selection
		var retryAPIKey string

		for keyAttempts, attemptOrdinal := 0, 0; keyAttempts < maxRetries || retryAPIKey != ""; attemptOrdinal++ {
			isRetryAttempt := attemptOrdinal > 0 || urlIdx > 0
			// 释放上一轮 attempt 的并发信号量（首次为空操作）
			if activeRateLimitRelease != nil {
				activeRateLimitRelease()
				activeRateLimitRelease = nil
			}
			attemptBody := requestBody
			if shouldStripBillingHeader(kind, upstream) {
				attemptBody, _ = RemoveBillingHeadersWithContext(c, attemptBody, envCfg.EnableRequestLogs, apiType)
			}
			if shouldNormalizeMetadataUserID(kind, upstream) {
				attemptBody = NormalizeMetadataUserID(attemptBody)
			}
			// Claude Messages 入口：将 messages 中的 system 角色抽回顶层 system 字段。
			// 在 provider 分发前统一处理，使所有上游类型（claude/openai/gemini/responses）均生效，
			// 兼容 Opus 4.8 / Fable 5 等将 system 作为消息 role 发送、而旧上游仅支持 user/assistant 的情况。
			if kind == scheduler.ChannelKindMessages && upstream.NormalizeSystemRoleToTopLevel {
				attemptBody = providers.NormalizeSystemRoleToTopLevel(attemptBody)
			}
			// 发往上游前按实际模型能力 clamp 最大输出 token：
			// 客户端/上游 subagent 可能发送超过模型上限的 max_tokens（如 Claude Code 默认 64000），
			// 而部分平台（火山方舟 kimi 系列硬限 32768）会直接 400。此处静默下调到模型上限，
			// 使请求成功而非被调度过滤为"无可用渠道"。
			if cap := config.ResolveUpstreamCapability(capabilityRequestModel, upstream, cfgManager.GetConfig().UpstreamModelCapabilities); cap.Capability.MaxOutputTokens > 0 {
				if clamped, changed := clampMaxTokensInBody(attemptBody, kind, cap.Capability.MaxOutputTokens); changed {
					attemptBody = clamped
					RequestLogf(c, "[%s-Clamp] max_tokens 超过模型 %q 上限 %d，已下调", apiType, cap.ActualModel, cap.Capability.MaxOutputTokens)
				}
			}
			RestoreRequestBody(c, attemptBody)
			c.Set("requestBodyBytes", attemptBody)

			var selection keypool.Selection
			var apiKey string
			internalRetry := retryAPIKey != ""
			if retryAPIKey != "" {
				selection = retrySelection
				apiKey = retryAPIKey
				retrySelection = keypool.Selection{}
				retryAPIKey = ""
			} else {
				keyAttempts++
				var err error
				// 步骤 5+6: EndpointAttemptPolicy FilterKeys + SortKeys
				// selectAttemptAPIKeyFiltered 在 keypool.CandidatesForModel 之后应用 policy 过滤/排序。
				// nil policy 时回退到 selectAttemptAPIKey（行为不变）。
				selection, apiKey, err = selectAttemptAPIKeyFiltered(channelScheduler, kind, channelIndex, upstream, failedKeys, failedQuotaGroups, redirectedModel, nextAPIKey, endpointPolicy, apiType, c)
				if err != nil {
					lastError = err
					break // 当前 BaseURL 没有可用 Key，尝试下一个 BaseURL
				}
			}

			// 检查熔断状态
			circuitState := metricsManager.GetKeyCircuitState(currentBaseURL, apiKey, metricsServiceType)
			if circuitState == metrics.CircuitStateOpen {
				failedKeys[apiKey] = true
				RequestLogf(c, "[%s-Circuit] 跳过 open 状态中的 Key: %s", apiType, utils.MaskAPIKey(apiKey))
				continue
			}
			if circuitState == metrics.CircuitStateHalfOpen {
				probeKey := currentBaseURL + "|" + apiKey
				if !metricsManager.TryAcquireProbe(currentBaseURL, apiKey, metricsServiceType) {
					RequestLogf(c, "[%s-Circuit] half-open 探针已占用，等待 Key 恢复: %s", apiType, utils.MaskAPIKey(apiKey))
					acquired, state := waitForHalfOpenProbe(c.Request.Context(), metricsManager, currentBaseURL, apiKey, metricsServiceType)
					if acquired {
						probeAcquired[probeKey] = true
						RequestLogf(c, "[%s-Circuit] 等待后取得 half-open 探针 Key: %s", apiType, utils.MaskAPIKey(apiKey))
					} else if state == metrics.CircuitStateClosed {
						RequestLogf(c, "[%s-Circuit] half-open 探针已由其他请求恢复，继续使用 Key: %s", apiType, utils.MaskAPIKey(apiKey))
					} else {
						failedKeys[apiKey] = true
						RequestLogf(c, "[%s-Circuit] half-open 探针等待超时或已熔断，暂时跳过 Key: %s", apiType, utils.MaskAPIKey(apiKey))
						continue
					}
				} else {
					probeAcquired[probeKey] = true
					RequestLogf(c, "[%s-Circuit] 使用 half-open 探针 Key: %s", apiType, utils.MaskAPIKey(apiKey))
				}
			}

			if envCfg.ShouldLog("info") {
				displayMax := maxRetries
				if internalRetry || len(shortEmptyRetried) > 0 {
					displayMax++
				}
				RequestLogf(c, "[%s-Key] 使用API密钥: %s (BaseURL %d/%d, 尝试 %d/%d)",
					apiType, utils.MaskAPIKey(apiKey), urlIdx+1, len(urlResults), attemptOrdinal+1, displayMax)
			}

			// 使用深拷贝避免并发修改问题
			upstreamCopy := upstream.Clone()
			upstreamCopy.BaseURL = currentBaseURL

			// Phase 3B-2: 应用 EndpointAttemptPolicy 的自动模型映射。
			// MappedModel 来自 ModelResolver（AutoManaged 渠道，三条件门控通过），
			// 优先级低于 RedirectModel（手动配置短路后 MappedModel 恒为空，不会双重映射）。
			var appliedMappedModel string
			if endpointPolicy != nil && endpointPolicy.ResolvedModelByEndpointUID != nil {
				keyHash := autopilot.KeyHashFromAPIKey(apiKey)
				euid := autopilot.GenerateEndpointUID(upstream.ChannelUID, currentBaseURL, keyHash)
				if mm := endpointPolicy.ResolvedModelByEndpointUID(euid); mm != "" {
					if replaced, err := sjson.SetBytes(attemptBody, "model", mm); err == nil {
						attemptBody = replaced
						appliedMappedModel = mm
						RestoreRequestBody(c, attemptBody)
						c.Set("requestBodyBytes", attemptBody)
						RequestLogf(c, "[%s-AutoModel] endpoint=%s model override: %s -> %s", apiType, euid, model, mm)
					}
				}
			}

			// 主动限速：在构建/发送请求前获取许可（渠道级 + Key/Quota scope）
			if rateLimitMgr := channelScheduler.GetRateLimitManager(); rateLimitMgr != nil {
				const maxRateLimitWait = 10 * time.Second
				releases := make([]func(), 0, 2)
				if limiter := rateLimitMgr.Get(apiType, channelIndex); limiter != nil {
					release, rlErr := limiter.Acquire(c.Request.Context(), maxRateLimitWait, time.Now())
					if rlErr != nil {
						lastError = rlErr
						RequestLogf(c, "[%s-RateLimit] 渠道限速器拦截: %v，尝试下一个 Key/渠道", apiType, rlErr)
						failedKeys[apiKey] = true
						continue
					}
					releases = append(releases, release)
				}
				if selection.LimiterScope != "" {
					keyLimiter := rateLimitMgr.GetOrCreateScoped(apiType, channelIndex, selection.LimiterScope, keypool.ConfigForCandidate(*upstream, selection.Config))
					release, rlErr := keyLimiter.Acquire(c.Request.Context(), maxRateLimitWait, time.Now())
					if rlErr != nil {
						lastError = rlErr
						RequestLogf(c, "[%s-RateLimit] Key/Quota 限速器拦截: scope=%s, err=%v，尝试下一个 Key/渠道", apiType, selection.LimiterScope, rlErr)
						failedKeys[apiKey] = true
						if selection.QuotaGroup != "" {
							failedQuotaGroups[selection.QuotaGroup] = true
						}
						for i := len(releases) - 1; i >= 0; i-- {
							releases[i]()
						}
						continue
					}
					releases = append(releases, release)
				}
				activeRateLimitRelease = func() {
					for i := len(releases) - 1; i >= 0; i-- {
						releases[i]()
					}
				}
			}

			req, err := buildRequest(c, upstreamCopy, apiKey)
			if err != nil {
				// buildRequest 失败通常是客户端参数问题或本地构建错误
				// 不应污染熔断统计，直接返回错误
				RequestLogf(c, "[%s-BuildRequest] 请求构建失败: %v", apiType, err)
				return false, "", 0, nil, nil, fmt.Errorf("request build failed: %w", err)
			}
			req = WithRequestLogContext(req, c)
			originalReasoningEffort := extractReasoningEffortForLog(requestBody)
			actualReasoningEffort := extractActualReasoningEffortForLog(req)

			// 记录请求开始
			channelScheduler.RecordRequestStart(currentBaseURL, apiKey, metricsServiceType, kind)

			// 计算本次尝试的 metricsKey（与统计同源的身份指纹）
			metricsKey := metrics.GenerateMetricsIdentityKey(currentBaseURL, apiKey, metricsServiceType)

			// 提取代理 Key 掩码（ProxyAuthMiddleware 写入 gin context），用于成本报表按用户维度分组
			proxyKeyMask := middleware.GetProxyKeyMask(c)
			logOpts := tryOpts.channelLogOptions
			if proxyKeyMask != "" {
				logOpts = append(logOpts, WithProxyKeyMask(proxyKeyMask))
			}

			// 创建 pending 状态日志（附带代理上下文与会话标识，用于 subagent 观测）
			logRequestID := CreatePendingLog(channelLogStore, metricsKey, channelIndex, upstream.Name, redirectedModel, originalModel, originalReasoningEffort, actualReasoningEffort, apiKey, currentBaseURL, apiType, operation, metrics.RequestSourceProxy, AgentContextFromGin(c), SessionIDFromGin(c), logOpts...)

			// TCP 建连开始即计数：将活跃度统计提前到发起上游请求之前；同时关联 proxyKeyMask 用于成本报表持久化
			requestID := metricsManager.RecordRequestConnectedWithProxyKeyMask(currentBaseURL, apiKey, metricsServiceType, redirectedModel, proxyKeyMask)

			lifecycleTrace := &RequestLifecycleTrace{
				OnConnected: func() {
					UpdateLogStatus(channelLogStore, metricsKey, logRequestID, metrics.StatusConnecting)
				},
				OnFirstResponseByte: func() {
					UpdateLogStatus(channelLogStore, metricsKey, logRequestID, metrics.StatusFirstByte)
				},
			}
			attemptStartedAt := time.Now()
			resp, err := SendRequestWithLifecycleTrace(req, upstream, envCfg, isStream, apiType, lifecycleTrace)
			if err != nil {
				lastError = err
				// 区分客户端取消和真实渠道故障（统一口径）
				if isClientSideError(err) {
					// 客户端取消：不计入失败，不触发 failover
					metricsManager.RecordRequestFinalizeClientCancel(currentBaseURL, apiKey, metricsServiceType, requestID)
					channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
					// 完成日志记录（客户端取消）
					CompleteLog(channelLogStore, metricsKey, logRequestID, 0, false, "client canceled", isRetryAttempt)
					RequestLogf(c, "[%s-Cancel] 请求已取消（SendRequest 阶段）", apiType)
					return true, "", 0, nil, nil, err
				}
				// 真实渠道故障：计入失败，继续 failover
				failedKeys[apiKey] = true
				cfgManager.MarkKeyAsFailed(apiKey, apiType)
				metricsManager.RecordRequestFinalizeFailureWithClass(currentBaseURL, apiKey, metricsServiceType, requestID, metrics.FailureClassRetryable)
				channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
				if markURLFailure != nil {
					markURLFailure(currentBaseURL)
				}
				// 步骤 10: 通知 autopilot FastDecayScorer 请求失败（连接错误）
				if notifyEndpointResultHook != nil {
					keyHash := autopilot.KeyHashFromAPIKey(apiKey)
					endpointUID := autopilot.GenerateEndpointUID(upstream.ChannelUID, currentBaseURL, keyHash)
					notifyEndpointResultHook(endpointUID, false)
				}
				// 记录渠道日志
				// 完成日志记录
				CompleteLog(channelLogStore, metricsKey, logRequestID, 0, false, err.Error(), isRetryAttempt)
				RequestLogf(c, "[%s-Key] 警告: API密钥失败: %v", apiType, err)
				continue
			}

			// 学习上游限流头：动态调整限速器状态（cooldown 等）
			if rateLimitMgr := channelScheduler.GetRateLimitManager(); rateLimitMgr != nil {
				now := time.Now()
				if limiter := rateLimitMgr.Get(apiType, channelIndex); limiter != nil {
					limiter.ApplyUpstreamHints(resp.Header, resp.StatusCode, now)
				}
				if selection.LimiterScope != "" {
					if limiter := rateLimitMgr.GetScoped(apiType, channelIndex, selection.LimiterScope); limiter != nil {
						limiter.ApplyUpstreamHints(resp.Header, resp.StatusCode, now)
					}
				}
			}

			// 通知 Autopilot 限速发现器（Phase 1 shadow，不修改调度链路）
			// endpointUID 与 profiler 同源：GenerateEndpointUID(channelUID, baseURL, keyHashFromAPIKey)
			// 复用已有的 metricsKey（与统计同源的身份指纹）
			keyHash := autopilot.KeyHashFromAPIKey(apiKey)
			signalEndpointUID := autopilot.GenerateEndpointUID(upstream.ChannelUID, currentBaseURL, keyHash)
			ratelimit.NotifySignal(
				signalEndpointUID, metricsKey, apiType, isStream,
				time.Since(attemptStartedAt).Milliseconds(),
				resp.Header, resp.StatusCode,
			)

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				respBodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				respBodyBytes = utils.DecompressGzipIfNeeded(resp, respBodyBytes)

				// 记录错误响应头（用于诊断限流 header）
				LogUpstreamResponseHeaders(c, resp, envCfg, apiType)

				shouldFailover, isQuotaRelated := ShouldRetryWithNextKeyWithLogTag(resp.StatusCode, respBodyBytes, cfgManager.GetFuzzyModeEnabled(), apiType, RequestLogTag(c))
				isTemporarilyOverloaded := IsUpstreamTemporarilyOverloaded(respBodyBytes)
				isAccountPoolUnavailable := IsUpstreamAccountPoolUnavailable(respBodyBytes)

				// 检查是否应永久拉黑该 Key（认证/权限/余额错误）
				blResult := ShouldBlacklistKey(resp.StatusCode, respBodyBytes)
				if blResult.ShouldBlacklist {
					isBalanceError := blResult.Reason == "insufficient_balance"
					if !isBalanceError || upstream.IsAutoBlacklistBalanceEnabled() {
						blacklistMessage := blResult.Message
						if strings.EqualFold(apiType, "Vectors") {
							blacklistMessage = errorBodySummaryForLog(apiType, resp.StatusCode, respBodyBytes)
						}
						if err := cfgManager.BlacklistKey(apiType, channelIndex, apiKey, blResult.Reason, blacklistMessage); err != nil {
							RequestLogf(c, "[%s-Blacklist] 拉黑 Key 失败: %v", apiType, err)
						}
					}
				}

				if shouldFailover {
					lastError = fmt.Errorf("上游错误: %d", resp.StatusCode)
					failedKeys[apiKey] = true
					cfgManager.MarkKeyAsFailed(apiKey, apiType)
					failureClass := metrics.FailureClassRetryable
					if isQuotaRelated {
						failureClass = metrics.FailureClassQuota
					}
					if isTemporarilyOverloaded || isAccountPoolUnavailable {
						failureClass = metrics.FailureClassOverloaded
					}
					metricsManager.RecordRequestFinalizeFailureWithClass(currentBaseURL, apiKey, metricsServiceType, requestID, failureClass)
					channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
					if markURLFailure != nil {
						markURLFailure(currentBaseURL)
					}
					// 步骤 10: 通知 autopilot FastDecayScorer 请求失败（HTTP 错误）
					if notifyEndpointResultHook != nil {
						keyHash := autopilot.KeyHashFromAPIKey(apiKey)
						endpointUID := autopilot.GenerateEndpointUID(upstream.ChannelUID, currentBaseURL, keyHash)
						notifyEndpointResultHook(endpointUID, false)
					}
					errorSummary := errorBodySummaryForLog(apiType, resp.StatusCode, respBodyBytes)
					if errorSummary != "" {
						RequestLogf(c, "[%s-Key] 上游错误详情摘要: channel=[%d] %s, key=%s, summary=%s", apiType, channelIndex, upstream.Name, utils.MaskAPIKey(apiKey), errorSummary)
					}
					RequestLogf(c, "[%s-Key] 警告: API密钥失败 (状态: %d)，尝试下一个密钥", apiType, resp.StatusCode)

					lastFailoverError = &FailoverError{
						Status: resp.StatusCode,
						Body:   respBodyBytes,
					}

					// 记录渠道日志
					channelErrorInfo := string(respBodyBytes)
					if strings.EqualFold(apiType, "Vectors") {
						channelErrorInfo = errorBodySummaryForLog(apiType, resp.StatusCode, respBodyBytes)
					}
					CompleteLog(channelLogStore, metricsKey, logRequestID, resp.StatusCode, false, channelErrorInfo, isRetryAttempt)

					if isQuotaRelated {
						deprioritizeCandidates[apiKey] = true
						if selection.QuotaGroup != "" {
							failedQuotaGroups[selection.QuotaGroup] = true
						}
					}
					if isAccountPoolUnavailable {
						channelScheduler.MarkChannelCooldown(kind, channelIndex, upstreamAccountPoolCooldown)
						RequestLogf(c, "[%s-Channel] 渠道 [%d] %s 上游账号池不可用，冷却 %s 并尝试下一个渠道", apiType, channelIndex, upstream.Name, upstreamAccountPoolCooldown)
						return false, "", 0, lastFailoverError, nil, lastError
					}
					if isTemporarilyOverloaded {
						channelScheduler.MarkChannelCooldown(kind, channelIndex, upstreamOverloadedCooldown)
						RequestLogf(c, "[%s-Channel] 渠道 [%d] %s 上游临时过载，冷却 %s 并尝试下一个渠道", apiType, channelIndex, upstream.Name, upstreamOverloadedCooldown)
						return false, "", 0, lastFailoverError, nil, lastError
					}
					continue
				}

				// 非 failover 错误，记录失败指标后返回（请求已处理）
				clientStatusCode := normalizeUpstreamErrorStatus(resp.StatusCode, respBodyBytes)
				channelErrorInfo := string(respBodyBytes)
				if strings.EqualFold(apiType, "Vectors") {
					errorSummary := errorBodySummaryForLog(apiType, resp.StatusCode, respBodyBytes)
					channelErrorInfo = errorSummary
					if errorSummary != "" {
						RequestLogf(c, "[Vectors-UpstreamError] channel=[%d] %s status=%d original_model=%q mapped_model=%q summary=%s",
							channelIndex, upstream.Name, resp.StatusCode, model, redirectedModel, errorSummary)
					}
				}
				metricsManager.RecordRequestFinalizeFailureWithClass(currentBaseURL, apiKey, metricsServiceType, requestID, metrics.FailureClassNonRetryable)
				channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
				// 记录渠道日志
				CompleteLog(channelLogStore, metricsKey, logRequestID, clientStatusCode, false, channelErrorInfo, isRetryAttempt)
				c.Data(clientStatusCode, "application/json", respBodyBytes)
				return true, "", 0, nil, nil, nil
			}

			// 成功响应：处理 quota key 降级
			if deprioritizeKey != nil && len(deprioritizeCandidates) > 0 {
				for key := range deprioritizeCandidates {
					deprioritizeKey(key)
				}
			}

			streamingUserID := ""
			if isStream {
				streamingUserID = trackStreamingConversation(c, channelScheduler, kind, model, channelIndex, upstream.Name)
				StartStreamTimeoutObservation(c, channelLogStore, metricsKey, logRequestID, time.Now())
			}
			usage, err = handleSuccess(c, resp, upstreamCopy, apiKey, attemptBody)
			if isStream {
				FinishStreamTimeoutObservation(c)
			}
			if err != nil {
				if isStream && streamingUserID != "" {
					channelScheduler.UpdateConversationStatus(kind, streamingUserID, "active")
				}
				lastError = err
				// 区分客户端错误和渠道故障
				if isClientSideError(err) {
					// 客户端取消/断开：计入总请求数但不计入失败
					metricsManager.RecordRequestFinalizeClientCancel(currentBaseURL, apiKey, metricsServiceType, requestID)
					channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
					RequestLogf(c, "[%s-Cancel] 请求已取消，停止渠道 failover", apiType)
					// 完成日志记录（客户端取消）
					CompleteLog(channelLogStore, metricsKey, logRequestID, http.StatusOK, false, "client canceled", isRetryAttempt)
				} else if errors.Is(err, ErrEmptyStreamResponse) || errors.Is(err, ErrInvalidResponseBody) || errors.Is(err, ErrEmptyNonStreamResponse) || errors.Is(err, ErrStreamFirstContentTimeout) || errors.Is(err, ErrStreamStalled) {
					// 空响应（流式 / 非流式）或无效响应体（如 HTML）或流式首字超时/断流：Header 未发送，可安全 failover
					retryKey := currentBaseURL + "|" + apiKey
					elapsed := time.Since(attemptStartedAt)
					if shouldRetryShortEmptyResponse(kind, err) && !shortEmptyRetried[retryKey] && elapsed <= shortEmptyResponseRetryWindow && !c.Writer.Written() {
						shortEmptyRetried[retryKey] = true
						retrySelection = selection
						retryAPIKey = apiKey
						metricsManager.RecordRequestFinalizeIgnored(currentBaseURL, apiKey, metricsServiceType, requestID)
						channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
						if probeAcquired[retryKey] {
							metricsManager.ReleaseProbe(currentBaseURL, apiKey, metricsServiceType)
							delete(probeAcquired, retryKey)
						}
						CompleteLog(channelLogStore, metricsKey, logRequestID, http.StatusOK, false, err.Error(), isRetryAttempt)
						RequestLogf(c, "[%s-EmptyResponse-Retry] 上游短空响应 (Key: %s, 耗时: %dms)，同渠道同 Key 重试一次",
							apiType, utils.MaskAPIKey(apiKey), elapsed.Milliseconds())
						continue
					}
					failedKeys[apiKey] = true
					cfgManager.MarkKeyAsFailed(apiKey, apiType)
					metricsManager.RecordRequestFinalizeFailureWithClass(currentBaseURL, apiKey, metricsServiceType, requestID, metrics.FailureClassRetryable)
					channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
					if markURLFailure != nil {
						markURLFailure(currentBaseURL)
					}
					// 记录渠道日志
					CompleteLog(channelLogStore, metricsKey, logRequestID, http.StatusOK, false, err.Error(), isRetryAttempt)
					RequestLogf(c, "[%s-InvalidResponse] 上游返回无效响应 (Key: %s): %v，尝试下一个密钥", apiType, utils.MaskAPIKey(apiKey), err)
					continue
				} else if blErr, ok := err.(*ErrBlacklistKey); ok {
					// SSE 流内检测到拉黑条件：Header 未发送，可安全 failover + 拉黑 Key
					failedKeys[apiKey] = true
					isBalanceError := blErr.Reason == "insufficient_balance"
					if !isBalanceError || upstream.IsAutoBlacklistBalanceEnabled() {
						if blacklistErr := cfgManager.BlacklistKey(apiType, channelIndex, apiKey, blErr.Reason, blErr.Message); blacklistErr != nil {
							RequestLogf(c, "[%s-Blacklist] 拉黑 Key 失败: %v", apiType, blacklistErr)
						}
					}
					cfgManager.MarkKeyAsFailed(apiKey, apiType)
					metricsManager.RecordRequestFinalizeFailureWithClass(currentBaseURL, apiKey, metricsServiceType, requestID, metrics.FailureClassRetryable)
					channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
					if markURLFailure != nil {
						markURLFailure(currentBaseURL)
					}
					CompleteLog(channelLogStore, metricsKey, logRequestID, http.StatusOK, false, fmt.Sprintf("key blacklisted: %s - %s", blErr.Reason, blErr.Message), isRetryAttempt)
					RequestLogf(c, "[%s-Blacklist] SSE 流内错误触发拉黑 (Key: %s, 原因: %s)，尝试下一个密钥", apiType, utils.MaskAPIKey(apiKey), blErr.Reason)
					continue
				} else {
					// 真实渠道故障：计入失败指标
					cfgManager.MarkKeyAsFailed(apiKey, apiType)
					metricsManager.RecordRequestFinalizeFailureWithClass(currentBaseURL, apiKey, metricsServiceType, requestID, metrics.FailureClassRetryable)
					channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
					// 记录渠道日志
					CompleteLog(channelLogStore, metricsKey, logRequestID, http.StatusOK, false, err.Error(), isRetryAttempt)
					RequestLogf(c, "[%s-Key] 警告: 响应处理失败: %v", apiType, err)
				}
				return true, "", 0, nil, usage, err
			}

			if markURLSuccess != nil {
				markURLSuccess(currentBaseURL)
			}
			// 步骤 9: 通知 autopilot FastDecayScorer 请求成功（§4.6.2a）
			if notifyEndpointResultHook != nil {
				keyHash := autopilot.KeyHashFromAPIKey(apiKey)
				endpointUID := autopilot.GenerateEndpointUID(upstream.ChannelUID, currentBaseURL, keyHash)
				notifyEndpointResultHook(endpointUID, true)
			}
			metricsManager.RecordRequestFinalizeSuccess(currentBaseURL, apiKey, metricsServiceType, requestID, usage)
			channelScheduler.RecordRequestEnd(currentBaseURL, apiKey, metricsServiceType, kind)
			if probeKey := currentBaseURL + "|" + apiKey; probeAcquired[probeKey] {
				metricsManager.ReleaseProbe(currentBaseURL, apiKey, metricsServiceType)
				delete(probeAcquired, probeKey)
			}
			// 记录渠道日志
			CompleteLog(channelLogStore, metricsKey, logRequestID, http.StatusOK, true, "", isRetryAttempt)

			// Phase 3B-2: 回显自动模型映射信息（受 EchoMappedModel 配置门控）。
			if appliedMappedModel != "" {
				routingCfg := cfgManager.GetAutopilotRouting()
				if routingCfg.ModelMapping.EchoMappedModel {
					c.Header("X-CCX-Mapped-Model", appliedMappedModel)
					c.Header("X-CCX-Original-Model", model)
					c.Header("X-CCX-Mapping-Source", "auto_resolve")
				}
			}

			// Phase 4 Item 8: 代理成功后回调（A/B 测试用）。
			// 在主响应已写回客户端之后触发，不影响主请求路径。
			if postSuccessfulProxyHook != nil {
				latencyMs := time.Since(attemptStartedAt).Milliseconds()
				// 复制 bodyBytes 防止异步使用时被原始切片回收
				bodyCopy := make([]byte, len(requestBody))
				copy(bodyCopy, requestBody)
				postSuccessfulProxyHook(string(kind), model, upstream.ChannelUID, http.StatusOK, latencyMs, bodyCopy)
			}

			return true, apiKey, originalIdx, nil, usage, nil
		}

		// 当前 BaseURL 的所有 Key 都失败，记录并尝试下一个 BaseURL
		if envCfg.ShouldLog("info") && urlIdx < len(urlResults)-1 {
			RequestLogf(c, "[%s-BaseURL] BaseURL %d/%d 所有 Key 失败，切换到下一个 BaseURL", apiType, urlIdx+1, len(urlResults))
		}
	}

	return false, "", 0, lastFailoverError, nil, lastError
}

func shouldRetryShortEmptyResponse(kind scheduler.ChannelKind, err error) bool {
	if kind != scheduler.ChannelKindMessages {
		return false
	}
	return errors.Is(err, ErrEmptyStreamResponse) || errors.Is(err, ErrEmptyNonStreamResponse)
}

// BuildDefaultURLResults 将 URLs 转为按原始顺序的结果列表（无动态排序）
func BuildDefaultURLResults(urls []string) []warmup.URLLatencyResult {
	results := make([]warmup.URLLatencyResult, len(urls))
	for i, url := range urls {
		results[i] = warmup.URLLatencyResult{
			URL:         url,
			OriginalIdx: i,
			Success:     true,
		}
	}
	return results
}

func waitForHalfOpenProbe(ctx context.Context, metricsManager *metrics.MetricsManager, baseURL, apiKey, serviceType string) (bool, metrics.CircuitState) {
	if metricsManager == nil {
		return false, metrics.CircuitStateOpen
	}

	timer := time.NewTimer(halfOpenProbeWaitTimeout)
	defer timer.Stop()
	ticker := time.NewTicker(halfOpenProbePollInterval)
	defer ticker.Stop()

	lastState := metrics.CircuitStateHalfOpen
	for {
		select {
		case <-ctx.Done():
			return false, lastState
		case <-timer.C:
			return false, lastState
		case <-ticker.C:
			lastState = metricsManager.GetKeyCircuitState(baseURL, apiKey, serviceType)
			if lastState != metrics.CircuitStateHalfOpen {
				return false, lastState
			}
			if metricsManager.TryAcquireProbe(baseURL, apiKey, serviceType) {
				return true, metrics.CircuitStateHalfOpen
			}
		}
	}
}

// ── EndpointAttemptPolicy 辅助函数 ──
//
// 设计 §4.6.2a 十步执行顺序的 policy 注入点：
//  步骤 1: applyPolicyToURLs — FilterURLs + SortURLs（URL 循环前）
//  步骤 5-6: selectAttemptAPIKeyFiltered — FilterKeys + SortKeys（每个 baseURL 内 key 候选阶段）
//  步骤 9-10: notifyEndpointResult — 请求成功/失败后更新 FastDecay（通过 hook）

// applyPolicyToURLs 对 urlResults 应用 EndpointAttemptPolicy 的 FilterURLs 和 SortURLs。
// 步骤 1 + 步骤 2：过滤 + 排序 baseURL 列表。
// policy 为 nil 时原样返回。
// 任一 policy 函数 panic 时回退原列表（fail-open）。
func applyPolicyToURLs(policy *autopilot.EndpointAttemptPolicy, urlResults []warmup.URLLatencyResult, apiType string, c *gin.Context) (ret []warmup.URLLatencyResult) {
	if policy == nil || len(urlResults) == 0 {
		return urlResults
	}
	ret = urlResults // 默认回退：panic recover 时返回原列表

	// 整体 recovery：任一 policy 函数 panic 时回退原列表（fail-open）
	defer func() {
		if r := recover(); r != nil {
			RequestLogf(c, "[%s-Autopilot-EndpointPolicy] applyPolicyToURLs panic: %v，回退原列表", apiType, r)
			ret = urlResults
		}
	}()

	// 步骤 1: FilterURLs
	urls := make([]string, len(urlResults))
	for i, r := range urlResults {
		urls[i] = r.URL
	}
	filtered := callPolicyFilterURLs(policy, urls, apiType, c)

	// 步骤 2: SortURLs
	sorted, _ := callPolicySortURLs(policy, filtered, apiType, c)

	// 按排序后的 URL 重建 urlResults（保留 OriginalIdx）
	urlToResult := make(map[string]warmup.URLLatencyResult, len(urlResults))
	for _, r := range urlResults {
		urlToResult[r.URL] = r
	}
	result := make([]warmup.URLLatencyResult, 0, len(sorted))
	for _, url := range sorted {
		if r, ok := urlToResult[url]; ok {
			result = append(result, r)
		}
	}
	return result
}

// callPolicyFilterURLs 安全调用 policy.FilterURLs，panic 时回退原列表。
func callPolicyFilterURLs(policy *autopilot.EndpointAttemptPolicy, urls []string, apiType string, c *gin.Context) []string {
	if policy == nil || policy.FilterURLs == nil {
		return urls
	}
	result := policy.FilterURLs(urls)
	return result
}

// callPolicySortURLs 安全调用 policy.SortURLs，panic 时回退原列表。
// 使用 result-capture 模式确保 panic 时返回原始输入。
func callPolicySortURLs(policy *autopilot.EndpointAttemptPolicy, urls []string, apiType string, c *gin.Context) ([]string, []autopilot.EndpointCandidate) {
	if policy == nil || policy.SortURLs == nil {
		return urls, nil
	}
	var result []string
	var candidates []autopilot.EndpointCandidate
	func() {
		defer func() {
			if r := recover(); r != nil {
				RequestLogf(c, "[%s-Autopilot-EndpointPolicy] SortURLs panic: %v，回退原列表", apiType, r)
			}
		}()
		result, candidates = policy.SortURLs(urls)
	}()
	if len(result) == 0 {
		return urls, nil
	}
	return result, candidates
}

// selectAttemptAPIKeyFiltered 对 policy 过滤/排序后的 key 列表选择下一个可用 API key。
// 步骤 5 (FilterKeys) + 步骤 6 (SortKeys)：在 keypool.CandidatesForModel 之后应用 endpoint 级策略。
// 与 selectAttemptAPIKey 逻辑一致，但使用 policy 过滤/排序后的候选列表。
// policy 过滤/排序失败时回退到 selectAttemptAPIKey（fail-open）。
func selectAttemptAPIKeyFiltered(
	channelScheduler *scheduler.ChannelScheduler,
	kind scheduler.ChannelKind,
	channelIndex int,
	upstream *config.UpstreamConfig,
	failedKeys map[string]bool,
	failedQuotaGroups map[string]bool,
	model string,
	fallback NextAPIKeyFunc,
	policy *autopilot.EndpointAttemptPolicy,
	apiType string,
	c *gin.Context,
) (keypool.Selection, string, error) {
	if policy == nil {
		return selectAttemptAPIKey(channelScheduler, kind, channelIndex, upstream, failedKeys, failedQuotaGroups, model, fallback)
	}

	if !keypool.HasEffectiveConfig(upstream) {
		// 无 keypool 配置时：对 raw APIKeys 应用 policy filter/sort
		apiKeys := upstream.APIKeys
		if len(apiKeys) == 0 {
			return keypool.Selection{}, "", fmt.Errorf("上游 %s 没有可用的API密钥", upstream.Name)
		}

		filtered := callPolicyFilterKeys(policy, upstream.BaseURL, apiKeys, apiType, c)
		if fallback != nil {
			key, err := fallback(upstream, failedKeys)
			if err != nil {
				return keypool.Selection{}, "", err
			}
			// 验证 key 在过滤后的列表中
			filteredSet := make(map[string]bool, len(filtered))
			for _, k := range filtered {
				filteredSet[k] = true
			}
			if filteredSet[key] {
				return keypool.Selection{APIKey: key}, key, nil
			}
			// policy 过滤掉了所有 key → fail-open：使用第一个未失败的原始 key
		}
		// 无 fallback 时回退到未过滤
		return selectAttemptAPIKey(channelScheduler, kind, channelIndex, upstream, failedKeys, failedQuotaGroups, model, fallback)
	}

	// keypool 路径：获取候选 → FilterKeys → SortKeys → 选择
	candidates := keypool.CandidatesForModel(upstream, failedKeys, model)
	if len(candidates) == 0 {
		return keypool.Selection{}, "", fmt.Errorf("上游 %s 没有可用的API密钥", upstream.Name)
	}

	// 步骤 5: FilterKeys
	candidateKeys := make([]string, len(candidates))
	for i, cand := range candidates {
		candidateKeys[i] = cand.APIKey
	}
	filteredKeys := callPolicyFilterKeys(policy, upstream.BaseURL, candidateKeys, apiType, c)

	// 步骤 6: SortKeys
	sortedKeys, _ := callPolicySortKeys(policy, upstream.BaseURL, filteredKeys, apiType, c)

	// 构建 candidate 查找表
	candidateMap := make(map[string]keypool.Candidate, len(candidates))
	for _, cand := range candidates {
		candidateMap[cand.APIKey] = cand
	}

	// 按 policy 排序后的顺序选择第一个可用 key
	var deferred []keypool.Selection
	for _, apiKey := range sortedKeys {
		if failedKeys[apiKey] {
			continue
		}
		cand, ok := candidateMap[apiKey]
		if !ok {
			continue
		}
		if cand.QuotaGroup != "" && failedQuotaGroups[cand.QuotaGroup] {
			continue
		}
		selection := keypool.Selection{
			APIKey:         cand.APIKey,
			CredentialID:   cand.Scope,
			CredentialName: cand.Config.Name,
			QuotaGroup:     cand.QuotaGroup,
			LimiterScope:   cand.Scope,
			Config:         cand.Config,
		}
		if channelScheduler != nil && selection.LimiterScope != "" {
			cfg := keypool.ConfigForCandidate(*upstream, selection.Config)
			deferForLoad, _, _ := channelScheduler.ShouldDeferForRateLimit(kind, channelIndex, selection.LimiterScope, cfg, time.Now())
			if deferForLoad {
				deferred = append(deferred, selection)
				continue
			}
		}
		return selection, cand.APIKey, nil
	}

	if len(deferred) > 0 {
		selection := deferred[0]
		return selection, selection.APIKey, nil
	}

	return keypool.Selection{}, "", fmt.Errorf("上游 %s 没有可用的API密钥", upstream.Name)
}

// callPolicyFilterKeys 安全调用 policy.FilterKeys，panic 时回退原列表。
func callPolicyFilterKeys(policy *autopilot.EndpointAttemptPolicy, baseURL string, keys []string, apiType string, c *gin.Context) []string {
	if policy == nil || policy.FilterKeys == nil {
		return keys
	}
	var result []string
	func() {
		defer func() {
			if r := recover(); r != nil {
				RequestLogf(c, "[%s-Autopilot-EndpointPolicy] FilterKeys panic: %v，回退原列表", apiType, r)
			}
		}()
		result = policy.FilterKeys(baseURL, keys)
	}()
	if len(result) == 0 {
		return keys
	}
	return result
}

// callPolicySortKeys 安全调用 policy.SortKeys，panic 时回退原列表。
// 使用 result-capture 模式确保 panic 时返回原始输入。
func callPolicySortKeys(policy *autopilot.EndpointAttemptPolicy, baseURL string, keys []string, apiType string, c *gin.Context) ([]string, []autopilot.EndpointCandidate) {
	if policy == nil || policy.SortKeys == nil {
		return keys, nil
	}
	var result []string
	var candidates []autopilot.EndpointCandidate
	func() {
		defer func() {
			if r := recover(); r != nil {
				RequestLogf(c, "[%s-Autopilot-EndpointPolicy] SortKeys panic: %v，回退原列表", apiType, r)
			}
		}()
		result, candidates = policy.SortKeys(baseURL, keys)
	}()
	if len(result) == 0 {
		return keys, nil
	}
	return result, candidates
}

func selectAttemptAPIKey(channelScheduler *scheduler.ChannelScheduler, kind scheduler.ChannelKind, channelIndex int, upstream *config.UpstreamConfig, failedKeys map[string]bool, failedQuotaGroups map[string]bool, model string, fallback NextAPIKeyFunc) (keypool.Selection, string, error) {
	if !keypool.HasEffectiveConfig(upstream) {
		if fallback == nil {
			return keypool.Selection{}, "", fmt.Errorf("上游 %s 没有可用的API密钥", upstream.Name)
		}
		apiKey, err := fallback(upstream, failedKeys)
		if err != nil {
			return keypool.Selection{}, "", err
		}
		return keypool.Selection{APIKey: apiKey}, apiKey, nil
	}

	var deferred []keypool.Selection
	for _, candidate := range keypool.CandidatesForModel(upstream, failedKeys, model) {
		if candidate.QuotaGroup != "" && failedQuotaGroups[candidate.QuotaGroup] {
			continue
		}
		selection := keypool.Selection{
			APIKey:         candidate.APIKey,
			CredentialID:   candidate.Scope,
			CredentialName: candidate.Config.Name,
			QuotaGroup:     candidate.QuotaGroup,
			LimiterScope:   candidate.Scope,
			Config:         candidate.Config,
		}
		if channelScheduler != nil && selection.LimiterScope != "" {
			cfg := keypool.ConfigForCandidate(*upstream, selection.Config)
			deferForLoad, _, _ := channelScheduler.ShouldDeferForRateLimit(kind, channelIndex, selection.LimiterScope, cfg, time.Now())
			if deferForLoad {
				deferred = append(deferred, selection)
				continue
			}
		}
		return selection, candidate.APIKey, nil
	}

	if len(deferred) > 0 {
		selection := deferred[0]
		return selection, selection.APIKey, nil
	}

	return keypool.Selection{}, "", fmt.Errorf("上游 %s 没有可用的API密钥", upstream.Name)
}

func trackStreamingConversation(c *gin.Context, channelScheduler *scheduler.ChannelScheduler, kind scheduler.ChannelKind, model string, channelIndex int, channelName string) string {
	if c == nil || channelScheduler == nil {
		return ""
	}

	lastUserMsg, _ := c.Get("lastUserMessage")
	lastUserMsgStr, _ := lastUserMsg.(string)
	lastUserMsgs, _ := c.Get("lastUserMessages")
	lastUserMessages, _ := lastUserMsgs.([]string)
	userMsgCount, _ := c.Get("userMessageCount")
	userMsgCountInt, _ := userMsgCount.(int)
	if lastUserMsgStr == "" && userMsgCountInt == 0 {
		return ""
	}

	userID, _, agentRole, ok := RequestConversationContextFromGin(c)
	if !ok || userID == "" {
		return ""
	}
	if agentCtx := AgentContextFromGin(c); agentCtx != nil && agentRole == "" {
		agentRole = agentCtx.AgentRole
	}

	channelScheduler.TrackConversationWithStatusAndMessages(kind, userID, model, channelIndex, channelName, "", lastUserMsgStr, lastUserMessages, userMsgCountInt, agentRole, "streaming", AgentContextFromGin(c))
	return userID
}
