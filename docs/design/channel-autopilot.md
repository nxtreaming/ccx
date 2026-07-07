# CCX 渠道自动托管 (Channel Autopilot) 设计方案

## 1. 设计目标

> 用户添加渠道只需 baseURL + apiKey，系统自动完成协议发现、模型映射、能力画像、健康诊断、智能调度。
> 高级用户可覆盖任何自动决策，但默认不需要碰。

### 1.1 核心用户故事

| # | 作为… | 我希望… | 这样我就能… |
|---|-------|---------|------------|
| U1 | 有 30-40 个中转站的用户 | 添加渠道时只填 URL + Key | 不用手动配 modelMapping / supportedModels / 兼容开关 |
| U2 | 想用 Opus 监工 + 白嫖子代理的用户 | 主代理自动走高智商稳定渠道，子代理自动走便宜/快速渠道 | 不用手动给每个渠道调优先级 |
| U3 | 渠道会死/会限流的用户 | 一眼看到哪些渠道死了、限流了、配置错了 | 快速清理或修复，不用逐个 ping |
| U4 | 有临时薅羊毛渠道的用户 | 临时池优先消耗，用完自动切常规池 | 不浪费免费额度 |

### 1.2 设计原则

- **SOLID**：Analyzer、Profiler、ModelResolver、SmartRouter 职责单一，接口隔离
- **KISS**：先用可解释规则，不做复杂 AI 打分
- **DRY**：复用现有 MetricsManager、能力测试、模型注册表、CandidateFilter
- **YAGNI**：Phase 1 只做自动画像 + 健康诊断，Phase 2 做智能调度，Phase 3 做自愈

---

## 2. 现有基础（可直接复用）

### 2.1 调度器筛选链

`backend-go/internal/scheduler/select.go` 中 `SelectChannelWithOptions` 已有完整筛选链：

```text
Active+Model过滤 → RoutePrefix过滤 → 上下文过滤 → CandidateFilter回调
→ Channel Pinning → Manual Override → Promotion优先 → Trace亲和
→ Priority遍历(含健康检查/熔断/限速/视觉保护) → Soft-skip回退 → Degraded兜底
```

**关键扩展点**：`SelectionOptions.CandidateFilter` 是注入式回调，autopilot 可在此注入标签/画像过滤逻辑。

### 2.2 指标系统

`backend-go/internal/metrics/` 提供：

| 能力 | 接口 | 说明 |
|------|------|------|
| 健康判断 | `IsChannelHealthyMultiURL` | 多 URL 聚合 |
| 失败率 | `CalculateChannelFailureRateMultiURL` | 滑动窗口 |
| 聚合指标 | `GetChannelAggregatedMetrics` | 15m 成功率/请求数/缓存率 |
| 熔断状态 | `GetChannelCircuitStateMultiURL` | Closed/Open/HalfOpen |
| 失败分类 | `FailureClass` | retryable/overloaded/non_retryable/quota |
| 请求日志 | `ChannelLog` | 含 AgentRole、模型、延迟、流健康 |

### 2.3 能力测试与模型发现

`backend-go/internal/handlers/channel_discovery.go` 已实现：

- 协议自动探测：对 messages/responses/chat/gemini 四协议并发探测
- 模型自动发现：拉 `/v1/models`，失败时用内置候选模型回退
- 模型映射推荐：根据探测结果自动生成 modelMapping
- 能力探测：工具调用、视觉、thinking passback 测试

`capability_test_runner.go` 提供完整的多模型轮询测试框架。

### 2.4 模型注册表

`backend-go/internal/config/generated_model_registry.go` + `model_registry.go`：

- `ResolveUpstreamCapability`：四层解析（channel → global → builtin → default）
- `ResolveAgentModelProfile`：下游代理模型上下文窗口
- 覆盖 Claude/GPT/Gemini/DeepSeek/Kimi/GLM 等主流模型
- 每个模型有 ContextWindowTokens、MaxOutputTokens、Capabilities、Pricing

### 2.5 角色识别

`backend-go/internal/utils/headers.go` 的 `ExtractAgentContext`：

- Codex 子代理：`client_metadata.x-openai-subagent` 精确识别
- Claude Code 子代理：`X-Claude-Code-Agent-Id` header 精确识别
- 启发式识别：消息数 + 工具调用模式
- 已用于 trace 亲和隔离（`:subagent` 后缀）

## 3. 数据模型

### 3.1 粒度模型：三层架构

**⚠️ 关键设计决策：画像粒度是 `(baseURL, apiKey)` 对，不是 channel**

原因：
1. 同一 channel 的不同 baseURL 可能走不同 CDN，延迟/超时/截断行为不同
2. 同一 channel 的不同 API Key 可能属于不同分组/租户，提供不同的模型列表、协议、价格倍率
3. Key 可能随时换分组，之前探测的模型列表会失效，需要重新检测
4. 现有 `MetricsManager` 的 identity 已经是 endpoint 粒度：`GenerateMetricsIdentityKey(baseURL, apiKey, serviceType)`

```text
Channel（渠道，稳定身份 ChannelUID）
  └── KeyEndpoint（baseURL + apiKey 对）  ← 画像的最小单元
        └── ModelProfile（该 endpoint 上的每个实际模型）
```

ChannelProfile 是 KeyEndpoint 画像的聚合视图，用于 UI 展示和调度粗筛。

### 3.1.1 稳定身份与 Metrics Identity

**channel index 不能作为持久身份。** 当前管理 API 和调度器大量使用数组 index 作为 `channelId`，但用户重排、删除、插入渠道后 index 会变化。如果画像表以 index 为主键，历史画像、健康证据、模型快照会串到另一个渠道。

落地时必须先引入稳定身份：

```go
type UpstreamConfig struct {
    // ... 现有字段 ...
    ChannelUID string `json:"channelUid,omitempty"` // 新增：渠道稳定 ID，创建后不因重排/改名/API Key 变更而改变
}
```

规则：
- `ChannelUID` 在新增渠道时生成；读取旧配置时由 ConfigManager 补齐并持久化一次。
- `ChannelID` 继续表示当前数组 index，仅用于兼容旧 API 和 UI 展示，不参与新画像表主键。
- `ChannelUID` 不从 name/baseURL/apiKey 推导，避免用户改名或换 key 后画像断裂。
- 删除渠道后，以 `channel_uid + channel_kind` 清理画像；重排渠道不触发画像迁移。

Metrics identity 也必须与现有实现保持一致：

```text
identityBaseURL = utils.MetricsIdentityBaseURL(baseURL, serviceType)
metricsKey      = metrics.GenerateMetricsIdentityKey(baseURL, apiKey, serviceType)
lookupKeys      = metrics.GenerateMetricsLookupKeys(baseURL, apiKey, serviceType)
```

注意：
- `serviceType` 影响 `identityBaseURL` 的默认版本前缀，必须写入画像记录。
- `lookupKeys` 只用于兼容旧历史数据，ProfileStore 主键使用当前规范 `metricsKey`。
- 画像表不保存明文 API Key，只保存 `keyMask` 和 `metricsKey`；如需按 key 展示名称，读取 `APIKeyConfig.Name`。

### 3.2 KeyEndpoint 画像 (KeyEndpointProfile)

**画像的最小单元**，对应一个具体的 `baseURL + apiKey` 组合：

```go
// backend-go/internal/autopilot/key_endpoint_profile.go

type KeyEndpointProfile struct {
    // ── 身份 ──
    ChannelUID     string `json:"channelUid"`     // 稳定渠道 ID，用于持久化主键
    ChannelID      int    `json:"channelId"`      // 当前配置数组 index，仅用于展示/兼容
    ChannelKind    string `json:"channelKind"`
    ServiceType    string `json:"serviceType"`    // metrics identity 依赖 serviceType
    BaseURL        string `json:"baseUrl"`        // 原始配置 URL，用于展示
    IdentityBaseURL string `json:"identityBaseUrl"` // MetricsIdentityBaseURL(baseURL, serviceType)
    KeyMask        string `json:"keyMask"`        // 掩码后的 key，如 sk-***abc
    MetricsKey     string `json:"metricsKey"`     // GenerateMetricsIdentityKey(baseURL, apiKey, serviceType)
    UpdatedAt      time.Time `json:"updatedAt"`

    // ── 自动推导维度 ──
    HealthState      HealthState    `json:"healthState"`
    HealthConfidence float64        `json:"healthConfidence"`
    QualityTier      QualityTier    `json:"qualityTier"`
    StabilityTier    StabilityTier  `json:"stabilityTier"`
    SpeedTier        SpeedTier      `json:"speedTier"`
    CostTier         CostTier       `json:"costTier"`
    CostProfile      CostProfile    `json:"costProfile,omitempty"`

    // ── 能力标签（该 endpoint 特有）──
    SupportsVision     bool `json:"supportsVision"`
    SupportsToolCalls  bool `json:"supportsToolCalls"`
    SupportsReasoning  bool `json:"supportsReasoning"`
    SupportsLongCtx    bool `json:"supportsLongCtx"`

    // ── 该 endpoint 的可用模型列表 ──
    AvailableModels    []string `json:"availableModels"`    // 探测到的实际模型列表
    ModelMapping       map[string]string `json:"modelMapping"` // 该 endpoint 的模型映射

    // ── 运行时指标（来自 MetricsManager）──
    SuccessRate15m   float64    `json:"successRate15m"`
    P95LatencyMs     int64      `json:"p95LatencyMs"`
    ConsecutiveFail  int        `json:"consecutiveFail"`
    LastSuccessAt    *time.Time `json:"lastSuccessAt,omitempty"`
    LastFailureAt    *time.Time `json:"lastFailureAt,omitempty"`

    // ── 自动限速画像（仅在未显式配置时用于运行态 limiter）──
    DiscoveredRPM          int     `json:"discoveredRpm,omitempty"`
    DiscoveredMaxConcurrent int    `json:"discoveredMaxConcurrent,omitempty"`
    RateLimitSource        string  `json:"rateLimitSource,omitempty"` // manual | header | passive_aimd | unknown
    RateLimitConfidence    float64 `json:"rateLimitConfidence,omitempty"`

    // ── 分组感知 ──
    DetectedGroup    string    `json:"detectedGroup,omitempty"`   // 检测到的 key 分组标识
    GroupChangedAt   *time.Time `json:"groupChangedAt,omitempty"` // 分组变更时间
    ModelListHash    string    `json:"modelListHash,omitempty"`   // 模型列表哈希，用于检测变更

    // ── 诊断 ──
    HealthEvidence   []string        `json:"healthEvidence"`
    SuggestedAction  SuggestedAction `json:"suggestedAction"`

    // ── 元数据 ──
    Source           string  `json:"source"`
    Confidence       float64 `json:"confidence"`
}
```

```go
// CostProfile 描述该 endpoint 的实际成本倍率。
// 目标是把模型注册表里的公开 USD 价格换算成用户真实付费成本。
type CostProfile struct {
    // 分组倍率：不同上游账号组/模型组常有不同计费倍率。
    // key 为模型组或通配符，例如 "*"、"claude-opus"、"gpt-5"、"gemini"。
    GroupMultipliers map[string]float64 `json:"groupMultipliers,omitempty"`

    // 充值倍率：充值赠送或折扣换算。1.0=无折扣；2.0=付 1 得 2，真实成本减半。
    RechargeMultiplier float64 `json:"rechargeMultiplier,omitempty"`

    // 最终成本倍率，按请求模型解析后计算：
    // effective = groupMultiplier / rechargeMultiplier
    EffectiveCostMultiplier float64 `json:"effectiveCostMultiplier,omitempty"`

    // 基于模型注册表 Pricing × EffectiveCostMultiplier 得到的估算价格。
    EffectiveInputCostPerMTok  float64 `json:"effectiveInputCostPerMTok,omitempty"`
    EffectiveOutputCostPerMTok float64 `json:"effectiveOutputCostPerMTok,omitempty"`

    Source     string  `json:"source"`     // manual | default | inferred
    Confidence float64 `json:"confidence"` // 手动配置为 1.0
}
```

### 3.3 Channel 画像 (ChannelProfile) — 聚合视图

ChannelProfile 不再存储原始数据，而是从 KeyEndpoint 画像聚合而来：

```go
// backend-go/internal/autopilot/channel_profile.go

type ChannelProfile struct {
    ChannelUID  string `json:"channelUid"`
    ChannelID   int    `json:"channelId"` // 当前配置数组 index，仅用于展示/兼容
    ChannelKind string `json:"channelKind"`
    UpdatedAt   time.Time `json:"updatedAt"`

    // ── 聚合维度（取所有 endpoint 的"最差可用"或"最佳代表"）──
    HealthState      HealthState    `json:"healthState"`     // 取最差：任一 endpoint dead → degraded
    QualityTier      QualityTier    `json:"qualityTier"`     // 取最佳 endpoint 的质量
    StabilityTier    StabilityTier  `json:"stabilityTier"`   // 取中位数
    SpeedTier        SpeedTier      `json:"speedTier"`       // 取中位数
    CostTier         CostTier       `json:"costTier"`        // 取最佳 endpoint 的成本

    // ── 能力标签（取并集，但标注不一致）──
    SupportsVision     bool `json:"supportsVision"`
    SupportsToolCalls  bool `json:"supportsToolCalls"`
    SupportsReasoning  bool `json:"supportsReasoning"`
    SupportsLongCtx    bool `json:"supportsLongCtx"`

    // ── 聚合指标 ──
    TotalEndpoints     int     `json:"totalEndpoints"`
    HealthyEndpoints   int     `json:"healthyEndpoints"`
    TotalModels        int     `json:"totalModels"`       // 去重后的模型总数
    SuccessRate15m     float64 `json:"successRate15m"`
    P95LatencyMs       int64   `json:"p95LatencyMs"`

    // ── 能力不一致警告 ──
    EndpointInconsistencies []EndpointInconsistency `json:"endpointInconsistencies,omitempty"`

    Source     string  `json:"source"`
    Confidence float64 `json:"confidence"`
}

// EndpointInconsistency 记录同一 channel 内不同 endpoint 的能力差异
type EndpointInconsistency struct {
    Dimension  string `json:"dimension"`  // "quality" | "vision" | "models" | "latency"
    Detail     string `json:"detail"`     // 例如 "endpoint-1: premium, endpoint-2: normal"
    Severity   string `json:"severity"`   // "info" | "warning"
}
```

**聚合策略**：

```text
维度        │ 策略              │ 原因
────────────┼───────────────────┼──────────────────────────────
HealthState │ 取最差            │ 任一 endpoint 死了，整个渠道至少 degraded
QualityTier │ 取最佳            │ 调度时选最佳 endpoint 的质量档
Stability   │ 取中位数          │ 避免单个坏 endpoint 拉低整个渠道
Speed       │ 取中位数          │ 同上
Cost        │ 取最佳            │ 便宜的 endpoint 存在就有价值
Vision等    │ 取并集            │ 只要有一个 endpoint 支持就算支持
```

### 3.4 模型画像 (ModelProfile)

每个 `(KeyEndpoint + 模型)` 组合的画像：

```go
// backend-go/internal/autopilot/model_profile.go

type ModelProfile struct {
    // ── 锚定到 KeyEndpoint ──
    ChannelUID  string `json:"channelUid"`
    ChannelID   int    `json:"channelId"`   // 当前配置数组 index，仅用于展示/兼容
    ChannelKind string `json:"channelKind"`
    ServiceType string `json:"serviceType"`
    MetricsKey  string `json:"metricsKey"`  // 精确到 identityBaseURL+key+serviceType
    ModelID     string `json:"modelId"`     // 该 endpoint 内的实际模型名
    UpdatedAt   time.Time `json:"updatedAt"`

    // ── 能力 ──
    QualityTier   QualityTier `json:"qualityTier"`
    SpeedTier     SpeedTier   `json:"speedTier"`
    ContextTokens int         `json:"contextTokens"`
    SupportsVision    bool    `json:"supportsVision"`
    SupportsToolCalls bool    `json:"supportsToolCalls"`
    SupportsReasoning bool    `json:"supportsReasoning"`

    // ── 探测结果 ──
    ProbeSuccess    bool      `json:"probeSuccess"`
    LastProbeAt     time.Time `json:"lastProbeAt"`
    ProbeLatencyMs  int64     `json:"probeLatencyMs"`
    ProbeConfidence float64   `json:"probeConfidence"`

    // ── 来源 ──
    Source string `json:"source"` // builtin_registry | auto_probe | capability_test | manual
}
```

### 3.5 请求画像 (RequestProfile)

每次请求在进入调度器前生成，不持久化：

```go
// backend-go/internal/autopilot/request_profile.go

type RequestProfile struct {
    // ── 来自请求 ──
    Model       string // 请求的目标模型
    AgentRole   string // "main" | "subagent"
    AgentType   string // "codex_subagent" | "claude_code_subagent"
    HasImage    bool   // 是否包含图片
    EstTokens   int    // 估算输入 token 数

    // ── 来自模型注册表 ──
    QualityNeed   QualityTier   // 该模型对应的质量需求
    ContextNeed   int           // 最小上下文窗口
    VisionNeed    bool          // 是否需要识图
    ToolUseNeed   bool          // 是否需要工具调用
    ReasoningNeed bool          // 是否需要推理

    // ── 任务分类 ──
    TaskClass TaskClass // supervisor | worker | lightweight | vision | long_context
}

type TaskClass string
const (
    TaskClassSupervisor   TaskClass = "supervisor"    // 主代理/监工
    TaskClassWorker       TaskClass = "worker"         // 子代理/干活
    TaskClassLightweight  TaskClass = "lightweight"    // 轻任务（摘要/标题）
    TaskClassVision       TaskClass = "vision"         // 识图任务
    TaskClassLongContext  TaskClass = "long_context"   // 长上下文任务
)
```

### 3.6 存储方案

| 数据 | 存储 | TTL |
|------|------|-----|
| KeyEndpointProfile | SQLite `key_endpoint_profiles` 表 + 内存缓存 | 持久化，运行时 5min 刷新 |
| ChannelProfile | 不持久化，运行时从 KeyEndpoint 聚合 | 内存计算 |
| ModelProfile | SQLite `model_profiles` 表 + 内存缓存 | 持久化，运行时 10min 刷新 |
| RequestProfile | 内存 | 请求级，不持久化 |
| TimeBucketMetrics | SQLite `time_bucket_metrics` 表 + 内存聚合桶 | 7 天滚动清理 |
| 健康证据 | SQLite `health_evidence` 表 | 7 天滚动清理 |
| 模型列表快照 | SQLite `model_list_snapshots` 表 | 30 天滚动清理，用于检测分组变更 |
| 画像变更日志 | SQLite `profile_changelog` 表 | 30 天滚动清理 |

```sql
-- 画像最小单元：baseURL + apiKey 对
CREATE TABLE key_endpoint_profiles (
    channel_uid       TEXT    NOT NULL,
    channel_id        INTEGER NOT NULL,      -- 当前配置数组 index，仅作展示快照
    channel_kind      TEXT    NOT NULL,
    service_type      TEXT    NOT NULL,
    metrics_key       TEXT    NOT NULL,      -- GenerateMetricsIdentityKey(baseURL, apiKey, serviceType)
    identity_base_url TEXT    NOT NULL,      -- MetricsIdentityBaseURL(baseURL, serviceType)
    base_url          TEXT    NOT NULL,      -- 原始配置 URL
    key_mask          TEXT    NOT NULL,      -- 掩码后的 key
    profile_json      TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL,
    PRIMARY KEY (channel_uid, channel_kind, metrics_key)
);
CREATE INDEX idx_key_endpoint_profiles_kind_index ON key_endpoint_profiles(channel_kind, channel_id);

-- 模型画像锚定到 endpoint
CREATE TABLE model_profiles (
    channel_uid  TEXT    NOT NULL,
    channel_id   INTEGER NOT NULL,          -- 当前配置数组 index，仅作展示快照
    channel_kind TEXT    NOT NULL,
    service_type TEXT    NOT NULL,
    metrics_key  TEXT    NOT NULL,          -- 精确到 endpoint
    model_id     TEXT    NOT NULL,
    profile_json TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL,
    PRIMARY KEY (channel_uid, channel_kind, metrics_key, model_id)
);
CREATE INDEX idx_model_profiles_kind_index ON model_profiles(channel_kind, channel_id);

-- 模型列表快照，用于检测 key 换分组
CREATE TABLE model_list_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_uid  TEXT    NOT NULL,
    channel_kind TEXT    NOT NULL,
    metrics_key  TEXT    NOT NULL,
    model_list   TEXT    NOT NULL,          -- JSON array of model IDs
    list_hash    TEXT    NOT NULL,          -- SHA256 of sorted model list
    detected_at  TEXT    NOT NULL
);
CREATE INDEX idx_snapshots_channel ON model_list_snapshots(channel_uid, channel_kind, detected_at);

CREATE TABLE health_evidence (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_uid TEXT    NOT NULL,
    channel_id  INTEGER NOT NULL,           -- 当前配置数组 index，仅作展示快照
    channel_kind TEXT   NOT NULL,
    metrics_key TEXT    NOT NULL DEFAULT '', -- 精确到 endpoint
    kind        TEXT    NOT NULL,
    evidence    TEXT    NOT NULL,
    severity    TEXT    NOT NULL,
    created_at  TEXT    NOT NULL
);
CREATE INDEX idx_health_evidence_channel ON health_evidence(channel_uid, channel_kind, created_at);
CREATE INDEX idx_health_evidence_endpoint ON health_evidence(channel_uid, channel_kind, metrics_key, created_at);
```

配置同步规则：
- 服务启动和配置热重载时，按 `channel_uid + channel_kind` 对齐当前 index，刷新画像中的 `channel_id` 展示快照。
- 配置中不存在的 `channel_uid` 标记为 `orphaned`，延迟清理；立即删除可能影响正在展示的历史诊断。
- API 返回时同时带 `channelUid` 和 `channelId`；所有写操作使用 `channelUid`，只读兼容接口可继续接收当前 index。

### 3.7 Key 分组变更检测

当 key 换了分组，之前的模型发现结果会失效。检测机制：

```go
// backend-go/internal/autopilot/group_change_detector.go

type GroupChangeDetector struct {
    profileStore *ProfileStore
}

// CheckGroupChange 对比本次探测的模型列表与上次快照
// 返回 (changed bool, oldHash string, newHash string, addedModels []string, removedModels []string)
func (d *GroupChangeDetector) CheckGroupChange(
    channelUID string,
    channelKind string,
    metricsKey string,
    currentModels []string,
) (bool, GroupChangeResult) {
    // 1. 取上次快照
    lastSnapshot := d.profileStore.GetLatestModelListSnapshot(channelUID, channelKind, metricsKey)
    if lastSnapshot == nil {
        // 首次，记录快照
        d.profileStore.SaveModelListSnapshot(channelUID, channelKind, metricsKey, currentModels)
        return false, GroupChangeResult{}
    }

    // 2. 计算当前模型列表哈希
    currentHash := hashModelList(currentModels)

    // 3. 对比
    if currentHash == lastSnapshot.ListHash {
        return false, GroupChangeResult{} // 无变化
    }

    // 4. 分组变更！计算差异
    added, removed := diffModelLists(lastSnapshot.ModelList, currentModels)

    // 5. 记录变更
    now := time.Now()
    d.profileStore.SaveModelListSnapshot(channelUID, channelKind, metricsKey, currentModels)
    d.profileStore.UpdateGroupChangedAt(channelUID, channelKind, metricsKey, now)

    return true, GroupChangeResult{
        OldHash:      lastSnapshot.ListHash,
        NewHash:      currentHash,
        AddedModels:  added,
        RemovedModels: removed,
        ChangedAt:    now,
    }
}
```

**触发时机**（与 L1/L2/L3 信号结合）：

```text
场景                          │ 触发方式              │ 动作
──────────────────────────────┼───────────────────────┼──────────────────────────────
L1 被动：请求返回 model_not_found │ 真实请求失败         │ 触发该 endpoint 的 L2 重探测
L2 探测：定期轻量探测           │ 每 2h 一次            │ 拉 /models 对比快照
新 key 添加                   │ 配置变更事件          │ 立即触发 L2 探测
用户手动「重新探测」            │ UI 按钮              │ 立即触发 L3 深度探测
```

**分组变更后的动作**：

```text
1. 标记该 endpoint 的 ModelProfile 为 stale
2. 重新触发 L2 模型发现（拉 /models）
3. 重新生成 ModelProfile
4. 如果模型列表变化导致能力标签变化 → 更新 KeyEndpointProfile → 重新聚合 ChannelProfile
5. 通知前端：endpoint 模型列表已变更
```

### 3.8 时序感知 — 快照到持续评估

**⚠️ 画像不是"探测一次定终身"。上游同一分组在忙时和闲时的质量差异可能很大。**

一个 endpoint 在凌晨 3 点测试通过，不代表早高峰 9 点仍然可用。画像系统必须从"静态快照"升级为"持续滚动评估"。

#### 时间桶指标 (TimeBucketMetrics)

```go
// backend-go/internal/autopilot/time_series.go

// TimeBucketMetrics 按固定时间桶记录 endpoint 的质量快照
type TimeBucketMetrics struct {
    ChannelID   int    `json:"channelId"`
    MetricsKey  string `json:"metricsKey"`
    BucketStart time.Time `json:"bucketStart"` // 桶起始时间（UTC，15 分钟对齐）
    BucketSize  time.Duration `json:"bucketSize"` // 桶大小，默认 15 分钟

    // ── 该桶内的聚合指标 ──
    RequestCount    int     `json:"requestCount"`
    SuccessCount    int     `json:"successCount"`
    FailureCount    int     `json:"failureCount"`
    OverloadedCount int     `json:"overloadedCount"` // 429
    StreamBreakCount int    `json:"streamBreakCount"`
    EmptyResponseCount int  `json:"emptyResponseCount"`

    P50LatencyMs  int64   `json:"p50LatencyMs"`
    P95LatencyMs  int64   `json:"p95LatencyMs"`
    P99LatencyMs  int64   `json:"p99LatencyMs"`

    SuccessRate   float64 `json:"successRate"`
    AvgInputTokens  int   `json:"avgInputTokens"`
    AvgOutputTokens int   `json:"avgOutputTokens"`
}
```

#### 质量趋势 (QualityTrend)

```go
// QualityTrend 描述 endpoint 质量的变化方向和幅度
type QualityTrend struct {
    MetricsKey  string    `json:"metricsKey"`
    DetectedAt  time.Time `json:"detectedAt"`

    // ── 趋势方向 ──
    Direction   TrendDirection `json:"direction"` // improving | stable | degrading | volatile

    // ── 对比基准 ──
    BaselineWindow   string  `json:"baselineWindow"`   // "7d" | "24h" | "1h"
    BaselineSuccessRate float64 `json:"baselineSuccessRate"`
    CurrentSuccessRate  float64 `json:"currentSuccessRate"`
    DeltaPercent        float64 `json:"deltaPercent"`   // 当前 vs 基准的变化百分比

    // ── 时段模式 ──
    HourlyPattern []HourlyQuality `json:"hourlyPattern,omitempty"` // 24 小时质量热力图
    PeakHours     []int           `json:"peakHours,omitempty"`     // 质量低谷的小时列表
    OffPeakQuality QualityTier    `json:"offPeakQuality,omitempty"`

    // ── 触发动作 ──
    ShouldReevaluate bool   `json:"shouldReevaluate"` // 是否需要重新评估画像
    ReevalReason     string `json:"reevalReason,omitempty"`
}

type TrendDirection string
const (
    TrendImproving TrendDirection = "improving"
    TrendStable    TrendDirection = "stable"
    TrendDegrading TrendDirection = "degrading"
    TrendVolatile  TrendDirection = "volatile" // 忙闲交替剧烈
)

// HourlyQuality 某个小时的平均质量（UTC）
type HourlyQuality struct {
    Hour         int     `json:"hour"`         // 0-23 UTC
    AvgSuccessRate float64 `json:"avgSuccessRate"`
    AvgP95Latency  int64   `json:"avgP95Latency"`
    SampleCount    int     `json:"sampleCount"` // 该小时总请求数（跨多天）
}
```

#### 趋势检测逻辑

```go
// backend-go/internal/autopilot/quality_trend_detector.go

type QualityTrendDetector struct {
    profileStore *ProfileStore
}

// DetectTrend 分析 endpoint 的质量趋势
func (d *QualityTrendDetector) DetectTrend(
    channelID int,
    metricsKey string,
    currentTime time.Time,
) QualityTrend {
    // 1. 取最近 7 天的时间桶数据
    buckets := d.profileStore.GetTimeBuckets(channelID, metricsKey, currentTime.Add(-7*24*time.Hour))

    // 2. 构建三个窗口的基准
    current1h  := aggregateBuckets(buckets, currentTime.Add(-1*time.Hour), currentTime)  // 最近 1 小时
    baseline24h := aggregateBuckets(buckets, currentTime.Add(-24*time.Hour), currentTime.Add(-1*time.Hour)) // 前 24 小时
    baseline7d  := aggregateBuckets(buckets, currentTime.Add(-7*24*time.Hour), currentTime.Add(-24*time.Hour)) // 前 7 天

    // 3. 判断趋势方向
    //    degrading：当前 1h 成功率比 24h 基准下降 > 15%，且比 7d 基准下降 > 10%
    //    improving：当前 1h 成功率比 24h 基准上升 > 10%
    //    volatile：最近 24h 内，小时成功率的标准差 > 20%
    //    stable：其他情况

    // 4. 构建 24 小时质量热力图（UTC 小时 × 平均成功率）
    hourlyPattern := buildHourlyPattern(buckets)
    peakHours := findQualityTroughs(hourlyPattern, threshold: 0.70)

    // 5. 判断是否需要重新评估
    shouldReevaluate := false
    var reason string
    switch {
    case current1h.SuccessRate < baseline7d.SuccessRate * 0.75:
        shouldReevaluate = true
        reason = fmt.Sprintf("成功率从 %.0f%% 降至 %.0f%%", baseline7d.SuccessRate*100, current1h.SuccessRate*100)
    case current1h.P95Latency > baseline7d.P95Latency * 2:
        shouldReevaluate = true
        reason = fmt.Sprintf("p95 延迟从 %dms 升至 %dms", baseline7d.P95Latency, current1h.P95Latency)
    case len(peakHours) > 0 && currentHourInList(currentTime, peakHours):
        // 当前正处于已知质量低谷
        reason = fmt.Sprintf("当前处于已知低谷时段 %v", peakHours)
        // 不触发重评估，但标记为时段性降级
    }

    return QualityTrend{
        MetricsKey:          metricsKey,
        DetectedAt:          currentTime,
        Direction:           trendDir,
        BaselineWindow:      "7d",
        BaselineSuccessRate: baseline7d.SuccessRate,
        CurrentSuccessRate:  current1h.SuccessRate,
        DeltaPercent:        (current1h.SuccessRate - baseline7d.SuccessRate) / baseline7d.SuccessRate * 100,
        HourlyPattern:       hourlyPattern,
        PeakHours:           peakHours,
        ShouldReevaluate:    shouldReevaluate,
        ReevalReason:        reason,
    }
}
```

#### 存储

```sql
-- 时间桶指标，7 天滚动
CREATE TABLE time_bucket_metrics (
    channel_uid  TEXT    NOT NULL,
    channel_kind TEXT    NOT NULL,
    channel_id   INTEGER NOT NULL,     -- 当前配置数组 index，仅作展示快照
    metrics_key  TEXT    NOT NULL,
    bucket_start TEXT    NOT NULL,     -- ISO 8601，15 分钟对齐
    bucket_json  TEXT    NOT NULL,
    PRIMARY KEY (channel_uid, channel_kind, metrics_key, bucket_start)
);
CREATE INDEX idx_time_bucket_metrics_kind_index ON time_bucket_metrics(channel_kind, channel_id, bucket_start);
-- 自动清理：DELETE WHERE bucket_start < datetime('now', '-7 days')
```

#### 对调度的影响

```text
场景                              │ 调度行为
──────────────────────────────────┼──────────────────────────────────────────
trend=stable                      │ 正常，使用当前画像
trend=improving                   │ 正常，画像质量档可能即将升级
trend=degrading + shouldReevaluate│ 降权，同时触发画像重评估
trend=volatile                    │ FastDecay 系数降低（衰减更快，回升更慢）
当前处于 peakHours 低谷           │ 该 endpoint 的 SpeedTier 临时降一档，不标记 dead
忙时质量低 + 闲时质量高           │ SmartRouter 在忙时自动倾向非低谷 endpoint
```

#### 与 HealthAnalyzer 的集成

趋势检测是 HealthAnalyzer 的输入信号之一，不是独立判定：

```text
HealthState 判定 = L1 被动指标 + 趋势信号 + L2/L3 探测（如需要）

具体：
  - L1 成功率 < 50% → 直接 dead（不看趋势）
  - L1 成功率 50-80% + trend=degrading → degraded，标记 shouldReevaluate
  - L1 成功率 80-95% + trend=degrading + 连续 3 个桶恶化 → degraded
  - L1 成功率 > 95% + trend=degrading → 保持 healthy，但在 UI 显示"质量下降趋势"
  - L1 成功率正常 + 当前处于 peakHours → 保持 healthy，但 SpeedTier 临时降档
```

#### 更新频率

```text
时间桶写入：每次请求后增量更新 15 分钟桶（内存聚合，桶结束时刷 SQLite）
趋势计算：每 15 分钟（桶结束时触发），不额外查询
小时热力图：每小时更新一次（24 个 UTC 小时 × 过去 7 天的数据）
```

## 4. 核心组件设计

### 4.1 组件总览

```text
┌──────────────────────────────────────────────────────────────┐
│                       Channel Autopilot                       │
│                                                              │
│  ┌───────────┐  ┌───────────┐  ┌──────────────────────────┐ │
│  │ Discovery  │  │ Profiler  │  │ HealthAnalyzer           │ │
│  │ (协议发现  │→│ (画像生成  │→│ (健康诊断/分层)           │ │
│  │  模型发现) │  │  能力推导) │  │                          │ │
│  └───────────┘  └───────────┘  └──────────┬───────────────┘ │
│        │              │                    │                 │
│        │              │         ┌──────────▼───────────┐     │
│        │              │         │ QualityTrendMonitor  │     │
│        │              │         │ (时序趋势/忙闲检测)  │     │
│        │              │         └──────────┬───────────┘     │
│        │              │                    │                 │
│        └──────────────┴────────────────────┘                 │
│                       ▼                                      │
│              ┌─────────────────┐                             │
│              │  ProfileStore   │ ← SQLite + 内存缓存         │
│              │  (endpoint 级)  │   + TimeBucket 指标         │
│              └─────────────────┘                             │
│                       ▼                                      │
│              ┌─────────────────┐                             │
│              │  SmartRouter    │ ← 注入 CandidateFilter      │
│              │ (任务分类→标签   │                              │
│              │  匹配→排序)     │                              │
│              └─────────────────┘                             │
│                       ▼                                      │
│              ┌─────────────────┐                             │
│              │ Scheduler       │ (现有，不修改核心链路)       │
│              │ SelectChannel   │                              │
│              └─────────────────┘                             │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 Discovery — 协议与模型发现

**职责**：对每个 KeyEndpoint 独立探测协议、发现模型、推荐映射。

**⚠️ 关键：探测粒度是 endpoint 级，不是 channel 级**

同一 channel 的不同 key 可能属于不同分组，返回不同模型列表。必须逐 endpoint 探测。

**触发时机**：
1. 渠道首次添加（`autoManaged: true` 时）
2. 手动触发「重新发现」
3. 定时刷新（每天一次，但逐 endpoint 轮转，不并发）
4. **分组变更检测触发**（model_not_found 或模型列表哈希变化）

**流程**：

```text
添加渠道(baseURL + key[])
  │
  ├─ 遍历每个 (baseURL, apiKey) 对：
  │   │
  │   ├─ 1. 协议探测：对单个 endpoint 探测 messages/responses/chat/gemini
  │   │     └─ 复用 capability_test_runner.executeModelTest
  │   │
  │   ├─ 2. 模型发现：GET /v1/models（用该 key）
  │   │     └─ 复用 channel_discovery.discoverTransientModels
  │   │     └─ 失败时用内置候选模型列表回退
  │   │
  │   ├─ 3. 分组变更检测：对比模型列表哈希
  │   │     └─ GroupChangeDetector.CheckGroupChange
  │   │     └─ 如果变更 → 标记 stale，重新生成 ModelProfile
  │   │
  │   ├─ 4. 模型选择：从发现的模型中选 Strong/Primary/Fast 三档
  │   │     └─ 复用 channel_discovery.selectDiscoveryModels
  │   │
  │   ├─ 5. 能力探测：对选中模型做硬失败检测（HTTP 错误/解析失败）
  │   │     └─ 复用 channel_discovery.runDiscoveryToolCallProbe
  │   │     └─ 复用 channel_discovery.runDiscoveryVisionProbe
  │   │
  │   ├─ 6. 映射推荐：根据协议类型生成该 endpoint 的 modelMapping
  │   │
  │   └─ 7. 生成 KeyEndpointProfile + ModelProfile[]
  │
  └─ 8. 聚合：从所有 endpoint 画像生成 ChannelProfile
```

**endpoint 间差异处理**：

```text
场景：同一 channel 的 key-A 返回 [opus, sonnet, haiku]，key-B 返回 [gpt-5.5, gpt-5.4-mini]
  │
  ├─ KeyEndpointProfile-A: qualityTier=premium, models=[opus, sonnet, haiku]
  ├─ KeyEndpointProfile-B: qualityTier=premium, models=[gpt-5.5, gpt-5.4-mini]
  │
  └─ ChannelProfile:
       - qualityTier = premium（取最佳 endpoint）
       - EndpointInconsistencies = [{"models", "key-A: Claude系列, key-B: GPT系列"}]
       - UI 显示：⚠️ 不同 Key 提供的模型系列不同
```

**与现有 Channel Discovery 的关系**：

现有 `POST /channel-discovery` 是一个"预览"接口，返回推荐但不自动应用。Autopilot 复用其核心逻辑，但：
- **逐 endpoint 探测**，而非整个 channel 一次
- 自动写入每个 endpoint 的 `modelMapping`、`supportedModels`、兼容开关
- 自动生成 `KeyEndpointProfile` + `ModelProfile` 记录
- 对 `autoManaged` 渠道静默执行，对非 auto 渠道提供「建议应用」按钮

### 4.3 Profiler — 画像生成器

**职责**：综合模型注册表、探测结果、运行时指标，生成 ChannelProfile 和 ModelProfile。

**推导规则**：

#### QualityTier 推导

```text
优先级 1：模型注册表 BuiltinUpstreamModelCapabilities 中的模型族
  claude-opus-* / gpt-5.5 / gpt-5.4     → premium
  claude-sonnet-* / gpt-5.3-codex        → high
  claude-haiku-* / gpt-5.4-mini          → normal
  其他                                    → low

优先级 2：渠道级 LowQuality 标记
  lowQuality: true → 最高 normal

优先级 3：capability-test 探测质量
  探测响应长度 > 100 tokens 且无截断 → 保持注册表推导
  探测失败或空响应                   → 降一档
```

#### StabilityTier 推导

```text
基于最近 1 小时指标：
  成功率 >= 95% 且 429 率 < 5%    → stable
  成功率 >= 80% 且 429 率 < 20%   → normal
  其他                            → unstable

额外信号（任一命中则降级）：
  连续失败 >= 5 次                → 最高 normal
  熔断器 open                     → unstable
  最近成功 > 6 小时前             → unstable
```

#### SpeedTier 推导

```text
基于最近 100 次请求的 p95 首 token 延迟：
  < 500ms   → fast
  < 2000ms  → normal
  >= 2000ms → slow

冷启动：无足够数据时用 capability-test 的 ProbeLatencyMs
  < 800ms   → fast
  < 3000ms  → normal
  >= 3000ms → slow
```

#### CostTier 推导

```text
优先级 1：用户手动成本画像
  APIKeyConfig.GroupMultipliers + RechargeMultiplier → 计算真实有效成本
  costHint                                           → 仅作为无价格数据时的粗粒度 fallback

优先级 2：模型注册表中的 Pricing 字段
  Input/Output 都是 0             → free
  EffectiveInput < $1/M 且 EffectiveOutput < $5/M   → cheap
  EffectiveInput < $10/M 且 EffectiveOutput < $30/M → normal
  其他                            → expensive

优先级 3：运行时行为启发（低置信度）
  频繁 429 且无 Retry-After      → 可能是免费/低配额，标记 cheap
  频繁 402/余额不足              → 有成本，标记 normal
```

#### EffectiveCost 推导

只用 `CostTier` 不足以做省钱调度。Autopilot 需要在满足能力和质量下界后，按 endpoint 的真实有效成本排序：

```text
baseInputCost  = ModelPricing.InputCacheMissPrice
baseOutputCost = ModelPricing.OutputPrice

groupMultiplier = matchGroupMultiplier(requestModel, APIKeyConfig.GroupMultipliers, default=1.0)
rechargeMultiplier = APIKeyConfig.RechargeMultiplier
if rechargeMultiplier <= 0:
    rechargeMultiplier = 1.0

effectiveMultiplier = groupMultiplier / rechargeMultiplier
effectiveInputCost  = baseInputCost  * effectiveMultiplier
effectiveOutputCost = baseOutputCost * effectiveMultiplier
```

例子：

```text
claude-opus 官价 input=$15/M, output=$75/M
key-A: groupMultiplier=0.8, rechargeMultiplier=1.0 → effective=0.8x
key-B: groupMultiplier=1.0, rechargeMultiplier=2.0 → effective=0.5x

在二者健康、上下文、能力和质量都满足时，优先 key-B。
```

约束：
- 成本优化只能在 `CapabilityFloor`、`QualityTier`、上下文窗口、vision/tool/reasoning 需求全部满足之后执行。
- 不允许为了省钱把 supervisor / premium 请求降级到不满足 `MinQualityTier` 的模型或 endpoint。
- 对低置信度成本画像，只能作为 tie-breaker，不能覆盖显式质量策略。

#### 能力标签推导

**⚠️ 原则：只做硬失败检测，不判定软质量问题**

识图/工具/reasoning 的"虚标"（渠道声称支持但实际返回垃圾）可靠判定很难。策略是：

- **硬失败**（可自动检测）：调用报错、格式错误、HTTP 错误码、解析失败
- **软质量问题**（留给人工）：答非所问、内容质量低、thinking 输出无意义

```text
SupportsVision：
  ── 硬条件（自动判定）──
  1. 注册表 Capabilities["vision"] == true
  2. 且 NoVision != true
  3. 且 (NoVisionModels 不含该模型 || VisionFallbackModel 已设置)
  ── 可选验证（L3 探测）──
  4. vision probe 返回 HTTP 200 且响应可解析（不要求内容质量）
  5. 如果 probe 返回 400/415/unsupported → 明确标记 SupportsVision=false

SupportsToolCalls：
  ── 硬条件（自动判定）──
  1. 注册表 Capabilities["toolCalls"] == true
  ── 可选验证（L3 探测）──
  2. tool_call probe 返回 HTTP 200 且响应含合法 tool_use block
  3. 如果 probe 返回 400/tool_not_found → 明确标记 SupportsToolCalls=false

SupportsReasoning：
  ── 硬条件（自动判定）──
  1. 注册表 ThinkingMode 非空
  ── 可选验证（L3 探测）──
  2. reasoning probe 返回 HTTP 200 且响应含 thinking/reasoning block
  3. 如果 probe 返回 400/thinking_not_supported → 明确标记 SupportsReasoning=false

SupportsLongCtx：
  1. ContextWindowTokens >= 200_000（来自注册表，无需探测）
  2. 或注册表 Supports1M == true
```

**虚标处理**：如果 L1 被动信号显示某渠道的 vision/tool/reasoning 请求**成功但用户标记为"结果差"**，系统不自动关闭标签，而是在 UI 上显示「⚠️ 用户反馈能力可能不准确」，允许人工 override。这避免了系统在"质量差"和"不支持"之间误判。

### 4.4 HealthAnalyzer — 健康诊断器

**职责**：持续分析渠道健康，生成 HealthState 和证据。

**⚠️ 核心原则：被动优先 (Passive-First)**

30-40 渠道 × 多模型的主动探测有 quota 成本，且白嫖渠道本身就抖。诊断信号分三层，成本递增：

| 层级 | 信号来源 | 成本 | 频率 | 适用场景 |
|------|---------|------|------|---------|
| L1 被动信号 | 真实请求的 MetricsManager | 零 | 实时/每次请求 | **默认层**，所有健康判定的主要依据 |
| L2 轻量探测 | 单模型 ping（最小 prompt） | 极低 | cooldown 复测 | L1 无数据（新渠道/长时间无请求） |
| L3 深度探测 | capability-test（多模型多协议） | 中 | 手动/每天 | 新渠道首次画像、用户主动触发、misconfigured 修复后 |

**分析周期**：
- L1 被动：每次请求后增量更新（复用 MetricsManager 已有的 RecordSuccess/RecordFailure）
- L1 聚合：每 5 分钟做一次滑动窗口聚合，更新 ChannelProfile
- L2 探测：仅在以下条件触发：
  - 渠道状态为 `unknown` 且添加超过 10 分钟
  - `limited`/`dead` 的 cooldown 到期
  - L1 数据不足（最近 1 小时请求数 < 3）
- L3 深度：仅在以下条件触发：
  - 用户手动点击「重新探测」
  - 新渠道 `autoManaged` 首次添加
  - `misconfigured` 状态修复后

**被动信号指标**（全部来自 MetricsManager 现有数据，无需额外请求）：

```text
成功/失败率     → MetricsManager.CalculateChannelFailureRateMultiURL
429 率         → FailureClass=overloaded 计数 / 总请求数
断流率         → ChannelLog.Status="streaming" 但无 "completed" 的比率
空响应率       → ChannelLog 中 DurationMs > 0 但 usage.InputTokens=0 的比率
p95 延迟       → ChannelLog.DurationMs 的 p95 分位
连续失败       → MetricsManager 滑动窗口 consecutiveFail
最后成功时间   → MetricsManager.GetChannelAggregatedMetrics.LastSuccessAt
熔断器状态     → MetricsManager.GetChannelCircuitStateMultiURL
Key 健康       → DisabledAPIKeys 数量 vs 总 Key 数量
```

**诊断逻辑**（见第 6 章详细设计）。

### 4.5 RateLimitDiscoverer — 上游 RPM 自动发现

**职责**：在用户未显式配置 `rateLimitRpm` / key 级 `rateLimitRpm` 时，为 endpoint 推导一个保守的运行态 RPM 限额，减少 429 和上游账号池冷却。

核心原则：
- **显式配置永远优先**：只要 channel 或 `APIKeyConfig` 设置了 `RateLimitRPM`，自动发现只做展示，不覆盖 limiter。
- **被动优先**：优先从真实响应头学习，不主动压测上游。
- **保守收敛**：无明确 header 时使用 AIMD（additive increase, multiplicative decrease）估算，只下调快、上调慢。
- **endpoint 粒度**：学习结果绑定 `metricsKey`，必要时映射到 key/quota scope limiter；不能只写 channel 级限速。

#### 4.5.1 信号来源

```text
优先级 │ 来源                         │ 动作
───────┼──────────────────────────────┼────────────────────────────────────
1      │ 用户显式 RateLimitRPM        │ 直接使用，自动发现不覆盖
2      │ x-ratelimit-limit-requests   │ 计算窗口内请求上限，换算 RPM
3      │ x-ratelimit-remaining/reset  │ 估算当前消耗速度与重置窗口
4      │ anthropic-ratelimit-*        │ 同上，按 reset 时间换算
5      │ Retry-After + 429/5xx        │ 进入 cooldown，并降低估算 RPM
6      │ 无 header 的 429 比率        │ AIMD 降低估算 RPM
7      │ 长时间成功且低延迟           │ AIMD 缓慢提高估算 RPM
```

现有 `ratelimit.ChannelLimiter.ApplyUpstreamHints` 已能解析 `Retry-After`、Anthropic remaining/reset、OpenAI remaining/reset 并施加 cooldown。Autopilot 在此基础上新增“可解释的 RPM 推导”，不要复制现有 cooldown 逻辑。

#### 4.5.2 学习状态

```go
// backend-go/internal/autopilot/rate_limit_discovery.go

type RateLimitProfile struct {
    ChannelUID   string    `json:"channelUid"`
    ChannelKind  string    `json:"channelKind"`
    MetricsKey   string    `json:"metricsKey"`
    Scope        string    `json:"scope"` // channel | key:<id> | quota:<id>
    EstimatedRPM int       `json:"estimatedRpm"`
    WindowSeconds int      `json:"windowSeconds"`
    MaxConcurrent int      `json:"maxConcurrent,omitempty"`
    Source       string    `json:"source"` // manual | header | passive_aimd
    Confidence   float64   `json:"confidence"`
    Last429At    *time.Time `json:"last429At,omitempty"`
    UpdatedAt    time.Time `json:"updatedAt"`
}
```

存储：
- 可放入 `KeyEndpointProfile.profile_json`；如果后续需要历史趋势，再独立建 `rate_limit_profiles` 表。
- 不写明文 key，只通过 `metricsKey` 和 limiter scope 关联。

#### 4.5.3 推导规则

```text
header 明确给 limit:
  estimatedRPM = normalize(limit, reset/window)
  confidence = 0.9

只有 remaining/reset:
  observedRate = requests_since_last_reset / elapsed
  estimatedRPM = min(current_estimate, inferred_window_capacity)
  confidence 逐次成功解析后提升，最高 0.75

429 + Retry-After:
  cooldownUntil = now + Retry-After
  estimatedRPM = max(minRPM, floor(current_estimate * 0.5))
  confidence = max(confidence, 0.7)

429 无 Retry-After:
  estimatedRPM = max(minRPM, floor(current_estimate * 0.7))
  confidence = max(confidence, 0.5)

连续稳定成功:
  每 10 分钟最多 +10%，且需要最近 15 分钟无 429
```

默认边界：
- `minRPM=1`，防止估算为 0 后永久不可用。
- `maxAutoRPM` 默认不超过 120，除非 header 明确给出更高 limit；避免自动学习把免费站打爆。
- 对流式请求同时学习 `MaxConcurrent`：若出现频繁排队、断流或 429，先降并发，再降 RPM。

#### 4.5.4 应用到运行态 limiter

```text
if manual channel/key RateLimitRPM > 0:
    使用手动配置；profile 只展示 header/passive 发现值
else if discovered EstimatedRPM > 0 and confidence >= threshold:
    将 discovered RPM 作为 runtime limiter 配置
else:
    保持现有默认；只依赖 Retry-After cooldown 和负载软跳过
```

落地时不要自动写入 `config.json`。UI 可以显示“建议设置 RPM=xx”，由用户一键采纳；运行态 limiter 可以使用 profile 中的估算值，服务重启后从 ProfileStore 恢复。

### 4.6 SmartRouter — 智能路由注入

**职责**：根据请求画像 + 渠道画像生成一次请求的路由计划。路由计划分两层：

1. **Channel 层**：通过 `CandidateFilter` 过滤/重排 channel。
2. **Endpoint 层**：通过 `EndpointAttemptPolicy` 过滤/重排同一 channel 内的 baseURL + key，并提供 endpoint 级模型覆盖。

只靠 `CandidateFilter` 不足以实现 endpoint 级画像，因为现有 `CandidateFilterFunc` 只能返回 `[]scheduler.ChannelInfo`，而实际 baseURL/key 选择发生在 `common.TryUpstreamWithAllKeys` 内部。

#### 4.6.1 请求路由计划

```go
// backend-go/internal/autopilot/smart_router.go

type RequestRoutingPlan struct {
    RequestProfile *RequestProfile
    CandidateFilter scheduler.CandidateFilterFunc
    EndpointPolicy  *EndpointAttemptPolicy
}

type SmartRouter struct {
    profileStore  *ProfileStore
    modelResolver *ModelResolver
    config        *SmartRoutingConfig
}

// BuildPlan 为每次请求构建完整路由计划。
func (r *SmartRouter) BuildPlan(profile *RequestProfile) *RequestRoutingPlan {
    return &RequestRoutingPlan{
        RequestProfile:  profile,
        CandidateFilter: r.buildCandidateFilter(profile),
        EndpointPolicy:  r.buildEndpointPolicy(profile),
    }
}
```

#### 4.6.2 EndpointAttemptPolicy

```go
// backend-go/internal/autopilot/endpoint_policy.go

type EndpointCandidate struct {
    ChannelUID  string
    ChannelKind string
    MetricsKey  string
    BaseURL     string
    KeyMask     string
    MappedModel string
    Score       float64
    Reason      string
}

type EndpointAttemptPolicy struct {
    RequestModel string
    ByChannelUID map[string][]EndpointCandidate // 已按优先级排序
    FailOpen     bool                           // true: 无画像时回退现有 key/baseURL 轮转
}
```

执行契约：
- `CandidateFilter` 只负责 channel 级顺序，不直接选择 key/baseURL。
- `EndpointAttemptPolicy` 通过新的 `common.TryUpstreamOption` 注入 `TryUpstreamWithAllKeys`。
- `TryUpstreamWithAllKeys` 在进入 baseURL/key 双层循环前，先按 policy 过滤和重排 `urlResults`。
- `selectAttemptAPIKey` 需要新增 policy-aware 分支：在 `keypool.CandidatesForModel` 结果上应用 endpoint 候选顺序、健康状态、FastDecay 分数和模型下界。
- 若 policy 对当前 channel 没有候选且 `FailOpen=true`，保持现有 failover 行为；若 `FailOpen=false`，跳过该 channel。

建议的最小接口：

```go
// backend-go/internal/handlers/common/upstream_failover.go

func WithEndpointAttemptPolicy(policy *autopilot.EndpointAttemptPolicy) TryUpstreamOption
```

这样可以复用现有 `TryUpstreamWithAllKeys` 的熔断、限流、拉黑、日志、URL warmup 行为，只在候选排序和模型覆盖点插入 autopilot。

#### 4.6.3 与现有手动控制的优先级

SmartRouter 不能破坏用户显式控制：

```text
最高优先级：X-Channel 指定渠道、手动序列 override、promotion
中间层级：SmartRouter channel 重排/过滤
底层约束：现有健康检查、熔断、限速、vision/context 保护、key/baseURL failover
```

当前 `CandidateFilter` 在 `SelectChannelWithOptions` 中执行得早于 X-Channel、manual override、promotion。Phase 2 落地时必须二选一：

1. 调整 `SelectChannelWithOptions` 的阶段顺序，让显式控制先定位候选，再让 SmartRouter 只处理剩余默认调度。
2. 保持现有顺序，但给 `CandidateFilter` 传入 `ProtectedChannels`，SmartRouter 对受保护 channel 只能降权，不能过滤。

推荐方案 1，行为更符合“显式用户意图优先”。

#### 4.6.4 与现有 CandidateFilter 的兼容

现有 handler 自有 filter（例如 vectors 的 embedding 维度过滤）必须与 SmartRouter 组合，而不是互相覆盖：

```text
active/model/context 过滤
→ protected manual controls
→ SmartRouter CandidateFilter（channel 粗筛/重排）
→ handler CandidateFilter（协议/业务硬约束）
→ priority/affinity/promotion fallback
→ TryUpstreamWithAllKeys + EndpointAttemptPolicy（endpoint 细选）
```

如果某个 handler 已经传入 `CandidateFilter`，公共 failover 外壳需要提供 `ComposeCandidateFilters`，按顺序合并多个 filter。

## 5. 智能调度策略

### 5.1 任务分类 (TaskClassifier)

请求进入时自动分类，决定调度策略：

```go
// backend-go/internal/autopilot/task_classifier.go

func ClassifyRequest(profile *RequestProfile) TaskClass {
    // 1. 识图任务优先判定
    if profile.HasImage && profile.VisionNeed {
        return TaskClassVision
    }

    // 2. 长上下文任务
    if profile.ContextNeed > 200_000 {
        return TaskClassLongContext
    }

    // 3. 主代理/监工
    if profile.AgentRole == "main" || profile.AgentRole == "" {
        // 主代理默认走 Supervisor 策略
        return TaskClassSupervisor
    }

    // 4. 子代理
    if profile.AgentRole == "subagent" {
        // 子代理默认走 Worker 策略
        return TaskClassWorker
    }

    return TaskClassWorker // 兜底
}
```

**轻任务识别**（可选，Phase 2+）：
- 模型名包含 `haiku` / `mini` / `flash`
- 上下文 < 10K 且无图片
- 请求类型为 `count_tokens` / 简单分类

### 5.2 调度策略矩阵

每个 TaskClass 对应一组优先级规则：

#### Supervisor（主代理/监工）

```text
优先级 1：qualityTier=high|premium + stabilityTier=stable + 长上下文
优先级 2：qualityTier=high + stabilityTier=normal
优先级 3：qualityTier=normal + stabilityTier=stable
同档排序：优先 stability，再按 estimatedRequestCost 从低到高
降级    ：qualityTier=high + stabilityTier=degraded（仅当无稳定高智商渠道时）
禁止    ：stabilityTier=unstable, costTier=free, qualityTier=low
```

#### Worker（子代理）

```text
硬约束  ：满足 CapabilityFloor + MinQualityTier，不允许质量降级省钱
优先级 1：estimatedRequestCost 最低 + stabilityTier>=normal
优先级 2：costTier=free|cheap + qualityTier=normal|high（临时池/白嫖池）
优先级 3：costTier=cheap + speedTier=fast
优先级 4：speedTier=fast + qualityTier=low|normal
优先级 5：qualityTier=normal + stabilityTier=stable（常规池）
默认跳过：costTier=expensive, qualityTier=premium
```

#### Lightweight（轻任务）

```text
优先级 1：estimatedRequestCost 最低 + speedTier=fast
优先级 2：speedTier=fast + costTier=free|cheap
优先级 3：costTier=free
优先级 4：speedTier=fast
禁止    ：qualityTier=premium, 视觉池, 长上下文池
```

#### Vision（识图任务）

```text
硬过滤  ：SupportsVision=true 且 SupportsToolCalls=true（如需要）
优先级 1：qualityTier=high|premium + vision
优先级 2：qualityTier=normal + vision
同档排序：estimatedRequestCost 从低到高，但不牺牲 vision/tool 硬约束
降级    ：当所有 vision 渠道不可用时，尝试 visionFallbackModel
禁止    ：SupportsVision=false 的渠道
```

#### LongContext（长上下文）

```text
硬过滤  ：ContextWindowTokens >= 请求需要的最小窗口
优先级 1：qualityTier=high|premium + longContext + stable
优先级 2：qualityTier=normal + longContext
同档排序：estimatedRequestCost 从低到高，但不牺牲上下文窗口
禁止    ：ContextWindowTokens < 需求 或 SupportsLongCtx=false
```

### 5.3 CandidateFilter 实现

```go
func (r *SmartRouter) filterByTaskStrategy(
    channels []scheduler.ChannelInfo,
    profiles map[int]*ChannelProfile,
    strategy taskStrategy,
) []scheduler.ChannelInfo {

    // 1. 硬过滤：排除不满足硬约束的渠道
    filtered := hardFilter(channels, profiles, strategy)

    // 2. 标签评分：每个渠道按策略规则打分
    scored := scoreChannels(filtered, profiles, strategy)

    // 3. 按分数降序排列
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Score > scored[j].Score
    })

    // 4. 返回重排后的 ChannelInfo 列表
    return scored.ToChannelInfoList()
}
```

**评分公式**：

```text
Score = w_quality * qualityScore
      + w_stability * stabilityScore
      + w_speed * speedScore
      + w_cost * costScore
      + w_savings * savingsScore
      + w_tier_match * tierMatchBonus
      - penalty

其中：
  qualityScore:   low=1, normal=2, high=3, premium=4
  stabilityScore: unstable=0, normal=1, stable=2
  speedScore:     slow=0, normal=1, fast=2
  costScore:      expensive=0, normal=1, cheap=2, free=3
  savingsScore:   normalizeCheapest(estimatedRequestCost)，越便宜越高，仅在满足硬约束后参与

  tierMatchBonus: 渠道画像标签匹配策略优先标签时 +10
  penalty:        healthState=degraded 时 -5, limited 时 -20

  权重根据 TaskClass 不同：
  Supervisor: w_quality=3, w_stability=2, w_speed=1, w_cost=0, w_savings=0.5
  Worker:     w_quality=1, w_stability=1, w_speed=2, w_cost=2, w_savings=3
  Lightweight:w_quality=0, w_stability=1, w_speed=3, w_cost=2, w_savings=3
  Vision:     w_quality=2, w_stability=2, w_speed=1, w_cost=1, w_savings=1
  LongContext: w_quality=2, w_stability=2, w_speed=1, w_cost=0, w_savings=1
```

`estimatedRequestCost` 使用请求级 token 估算：

```text
estimatedRequestCost =
  estimatedInputTokens  * effectiveInputCostPerMTok  / 1_000_000 +
  estimatedOutputTokens * effectiveOutputCostPerMTok / 1_000_000
```

当输出 token 不可预估时：
- supervisor/long_context 使用模型推荐输出上限的保守比例；
- worker/lightweight 使用最近同类请求的 p50 输出 token；
- 如果仍无数据，只使用输入成本做 tie-breaker，不做强排序。

### 5.4 模型自动映射 (ModelResolver)

当请求的模型在某个渠道的 `supportedModels` 中不存在时，自动寻找最佳映射。

**⚠️ 核心约束：能力下界 (Capability Floor)**

模型映射最大的风险是**语义降级**：用户以为在用 opus 级能力，实际被路由到白嫖模型，输出质量下降但无信号。因此映射必须满足能力下界约束：

```go
// backend-go/internal/autopilot/model_resolver.go

type CapabilityFloor struct {
    MinContextTokens   int    // 请求模型的 AgentModelProfile.ContextWindowTokens
    NeedsReasoning     bool   // 请求模型的 ThinkingMode 非空
    NeedsVision        bool   // 请求包含图片
    NeedsToolCalls     bool   // 请求包含工具定义
    MinQualityTier     QualityTier // 请求模型对应的质量档
}

type ModelResolver struct {
    profileStore *ProfileStore
}

// ResolveModel 将请求模型映射到渠道实际支持的模型
// 返回 (mappedModel, resolved, reason)
// resolved=false 表示该渠道无满足下界约束的模型，应跳过此渠道
func (r *ModelResolver) ResolveModel(
    requestModel string,
    channelUID string,
    channelKind string,
    metricsKey string,
    floor CapabilityFloor,
) (string, bool, string) {
    // 1. 查现有 modelMapping（精确匹配 → 模糊匹配）
    //    复用 config.RedirectModel
    //    如果有显式映射，信任用户配置，不做下界检查
    //    （用户手动设的映射视为已知正确）

    // 2. autoManaged 渠道：查 ModelProfile 表
    candidates := r.profileStore.GetModelProfiles(channelUID, channelKind, metricsKey)

    // 3. 硬过滤：只保留满足能力下界的模型
    eligible := filterByCapabilityFloor(candidates, floor)
    if len(eligible) == 0 {
        return "", false, "no model meets capability floor"
    }

    // 4. 在满足下界的模型中选最佳匹配
    //    匹配优先级：
    //    a. 同模型族（opus→opus, sonnet→sonnet）—— 最高优先
    //    b. 同质量档（premium→premium）
    //    c. 上下文窗口最接近（不超也不差太多）
    //    d. 探测延迟最低
    best := rankBySimilarity(eligible, requestModel, floor)

    return best.ModelID, true, fmt.Sprintf("mapped %s→%s (family:%s, quality:%s)",
        requestModel, best.ModelID, best.Family, best.QualityTier)
}

// filterByCapabilityFloor 只保留满足所有下界约束的模型
func filterByCapabilityFloor(profiles []ModelProfile, floor CapabilityFloor) []ModelProfile {
    var eligible []ModelProfile
    for _, p := range profiles {
        if !p.ProbeSuccess {
            continue // 未验证通过的模型不参与自动映射
        }
        if p.ContextTokens < floor.MinContextTokens {
            continue
        }
        if floor.NeedsReasoning && !p.SupportsReasoning {
            continue
        }
        if floor.NeedsVision && !p.SupportsVision {
            continue
        }
        if floor.NeedsToolCalls && !p.SupportsToolCalls {
            continue
        }
        if qualityTierRank(p.QualityTier) < qualityTierRank(floor.MinQualityTier) {
            continue
        }
        eligible = append(eligible, p)
    }
    return eligible
}
```

**映射示例**：

| 请求模型 | 能力下界 | 渠道实际模型 | 映射依据 | 是否通过 |
|----------|---------|-------------|---------|---------|
| `claude-opus-4-8` | context:1M, reasoning, quality:premium | `claude-opus-4-7` | 同 opus 族，满足全部下界 | ✓ |
| `claude-opus-4-8` | context:1M, reasoning, quality:premium | `claude-haiku-4-5` | haiku 不满足 quality:premium | ✗ 跳过渠道 |
| `gpt-5.5` | quality:premium, reasoning | `gpt-5.4` | 同 premium 档，满足下界 | ✓ |
| `claude-sonnet-5` | quality:high | `claude-sonnet-4-6` | 同 sonnet 族，满足下界 | ✓ |
| 请求含图片 | vision:true | 某渠道无 vision 模型 | 不满足 vision 下界 | ✗ 跳过渠道 |

#### 5.4.1 与现有模型过滤链路的关系

当前调度器的 active model filter 会先调用 `upstream.ExplainModelSupport(model)`，之后才进入 `CandidateFilter`。因此自动映射不能只放在 SmartRouter 后半段，否则 channel 会在映射前被剔除。

Phase 3 落地 ModelResolver 时必须调整为以下顺序：

```text
1. 构建 RequestProfile + CapabilityFloor
2. 收集候选 channel：
   - 显式 supportedModels 支持请求模型 → 保留
   - 显式 modelMapping 可把请求模型映射到上游模型 → 保留
   - autoManaged 且存在满足 CapabilityFloor 的 ModelProfile → 保留
   - 其他 → 过滤
3. 对每个候选 channel/endpoint 计算 request-scoped mappedModel
4. context filter 使用 mappedModel 的实际能力做窗口校验
5. TryUpstreamWithAllKeys 使用 EndpointAttemptPolicy 写入请求体中的实际 model
6. 响应 header/ChannelLog 回显 originalModel 与 mappedModel
```

实现上建议新增 `ModelSupportResolver`，替代 active model filter 中对 `ExplainModelSupport` 的直接调用：

```go
type ModelSupportResolution struct {
    Supported    bool
    ActualModel  string
    Source       string // supported_models | explicit_mapping | auto_profile
    Reason       string
}
```

这能避免“自动映射尚未运行，channel 已被 supportedModels 过滤掉”的死路。

**映射结果回显**：

这是调试的关键。映射发生时，必须在响应中标注真实使用的模型：

```text
方案 A（推荐）：在 response header 中回显
  X-CCX-Mapped-Model: claude-opus-4-7
  X-CCX-Original-Model: claude-opus-4-8
  X-CCX-Mapping-Source: auto_resolved

方案 B：在 Claude Messages 响应 body 的 model 字段用真实模型
  {"model": "claude-opus-4-7", ...}  // 而非请求的 claude-opus-4-8

方案 C：两者都做（最利于调试）
```

**安全边界**：
- 仅 `autoManaged: true` 的渠道触发自动映射
- 显式 `modelMapping` 始终优先，不经过下界检查（信任用户配置）
- 请求路径中的自动映射默认是 request-scoped，不直接写回 `modelMapping`
- Discovery 产生的映射建议可持久化；用户也可在 UI 将 request-scoped 映射保存为显式 `modelMapping`
- 映射日志记录每次决策（requestModel → mappedModel → floor → reason），写入 ChannelLog
- **禁止链式映射**：A→B 后不再 B→C，避免不可预测的降级链

## 6. 健康诊断系统

### 6.1 HealthState 状态机

```text
                    ┌──────────┐
          添加渠道 →│ unknown  │
                    └────┬─────┘
                         │ L2 探测成功 或 首次真实请求成功
                         ▼
                    ┌──────────┐
         ┌────────→│ healthy  │←────────┐
         │         └──┬───┬───┘         │
         │ L1被动信号  │   │ L1被动信号  │ L2 探测成功
         │ 成功率↓    │   │ 连续失败≥3  │ 或 真实请求成功
         │ 429增多    │   │             │
         │         ┌──▼┐  │        ┌────┴─────┐
         │         │deg│  │        │ limited  │
         │         │rad│  │        │(429/quota)│
         │         └─┬─┘  │        └────┬─────┘
         │           │    │             │
         │  L1连续≥10│    │ L1连续≥5   │ cooldown 到期
         │  或成功率  │    │ 且全部key   │ L2 探测失败
         │  <50%     │    │ 认证失败    │
         │         ┌─▼──┐ │        ┌────▼─────┐
         │         │dead│◄┘        │ dead     │
         │         └─┬──┘          └──────────┘
         │           │
         │  L2恢复   │  L1/L3 检测到配置错误
         │  探测成功  │
         │           │         ┌──────────────┐
         └───────────┘         │ misconfigured│← 用户修复后 L3 重测
                               └──────────────┘
```

**关键：所有正常状态转换基于 L1 被动信号，不需要额外请求。**

### 6.2 诊断规则

#### Dead（高置信度死亡）

```text
── 硬死（confidence >= 0.95，全部来自 L1 被动信号）──
  - 全部 Key 返回 401/403（最近 1 小时内的真实请求，FailureClass=non_retryable）
  - DNS/TLS 连接失败（ChannelLog 中 error 含 "dial tcp"/"tls"/"certificate"）
  - 连续失败 >= 15 次（MetricsManager 滑动窗口）

── 软死（confidence >= 0.80，L1 被动信号）──
  - 最近 24 小时无成功请求，且有失败记录
  - 熔断器 open 且 lastSuccessAt > 6 小时前
  - 成功率 < 10%（最近 1 小时，且请求样本 >= 5）

── 确认（仅在 L1 不足时触发 L2）──
  - L1 数据不足（请求数 < 5）但 L2 探测连续失败 >= 3 次
```

#### Degraded（可用但质量差）

```text
── 全部来自 L1 被动信号 ──
  - 成功率 50%-80%（最近 1 小时，请求样本 >= 10）
  - p95 延迟 > 5000ms（最近 1 小时）
  - 断流率 > 20%（ChannelLog 中 streaming→非completed 的比率，最近 30 分钟）
  - 空响应率 > 10%（usage 全零但无报错，最近 30 分钟）
```

#### Limited（限流中）

```text
── 全部来自 L1 被动信号 ──
  - FailureClass=overloaded 占比 > 30%（最近 15 分钟）
  - Retry-After header 出现在最近 5 分钟内的 ChannelLog
  - FailureClass=quota（402/insufficient_balance/insufficient_quota）
  - 熔断器 open 但 lastSuccessAt < 6 小时前（区别于 dead）
```

#### Misconfigured（配置疑似错误）

```text
── L1 被动信号 ──
  - 全部请求返回 404（modelMapping 指向不存在的模型）
  - 501/505（协议不支持）
  - capability-test 中仅部分协议成功，但 serviceType 配的是失败协议

── L3 深度探测确认（可选，用户手动触发）──
  - chat 协议成功但 serviceType 配为 claude
  - authHeader 类型与响应不匹配
```

#### Unknown（证据不足）

```text
  - 新添加的渠道，无历史数据
  - 最近 24 小时内请求数 < 5 且未运行 L2 探测
  - L3 capability-test 未运行或已过期（> 7 天）
```

### 6.3 死亡类型细分

```go
type DeathType string
const (
    DeathTypeHard       DeathType = "hard"        // DNS/TLS/认证（L1 被动即可判定）
    DeathTypeSoft       DeathType = "soft"        // 429/quota/临时错误（L1 被动即可判定）
    DeathTypeModel      DeathType = "model"       // 模型不可用（L1 或 L3）
    DeathTypeQuality    DeathType = "quality"     // 空响应/断流（L1 被动即可判定）
    DeathTypeUnknown    DeathType = "unknown"     // 无法分类
)
```

**注意：Quality 死亡只检测"硬失败"**（空响应、断流、格式错误），不检测"答非所问"等软质量问题。软质量问题留给人工标签 override。

### 6.4 健康诊断对调度的影响

```text
HealthState      │ 调度行为                    │ UI 表现
─────────────────┼─────────────────────────────┼──────────────
healthy          │ 正常参与调度                │ 绿色
degraded         │ 降权，只在同池不足时使用     │ 黄色
limited          │ cooldown 内跳过，到期 L2 复测│ 橙色
misconfigured    │ 不参与自动调度，提示修复     │ 紫色
dead             │ 默认移出调度，建议清理       │ 红色
unknown          │ 低风险请求小流量试探         │ 灰色
```

**自动恢复机制**：
- `limited` 渠道：cooldown 到期后触发 L2 轻量探测（单模型、最小 prompt），成功则回到 `healthy`
- `dead` 软死渠道：每 30 分钟检查一次 L1 被动信号，如果有真实请求成功则回到 `healthy`；无真实请求时每 2 小时触发 L2 探测，连续 3 次成功则恢复
- `dead` 硬死渠道：每 6 小时触发 L2 探测，连续 3 次成功则回到 `unknown`（不是直接 `healthy`，需真实请求验证）
- `misconfigured` 渠道：用户修复配置后手动触发 L3 深度探测

**L2 探测成本控制**：
- 每次 L2 探测只用 1 个模型、1 个 prompt、max_tokens=50
- 所有渠道的 L2 探测串行执行，间隔 >= 5 秒
- 每天 L2 探测总次数上限：`渠道数 × 12`（每 2 小时最多一次）
- 白嫖渠道的 L2 探测频率自动降低：如果连续 3 次 L2 探测失败，间隔翻倍（2h→4h→8h→24h）

### 6.5 白嫖池快速衰减机制

白嫖/临时池渠道的可用性是移动靶，不能靠滑动窗口慢慢反应。需要独立的衰减机制：

```go
// backend-go/internal/autopilot/fast_decay.go

// FastDecayScore 实时衰减评分，用于白嫖/临时池渠道
type FastDecayScore struct {
    ChannelID       int
    BaseScore       float64   // 基于 ChannelProfile 的基础分
    DecayFactor     float64   // 衰减系数 0.0-1.0
    LastUpdate      time.Time
    ConsecutiveFail int
}

// OnSuccess 请求成功时
func (s *FastDecayScore) OnSuccess() {
    s.DecayFactor = math.Min(1.0, s.DecayFactor+0.15) // 快速回升 +15%
    s.ConsecutiveFail = 0
}

// OnFailure 请求失败时
func (s *FastDecayScore) OnFailure() {
    s.ConsecutiveFail++
    // 指数衰减：连续失败越多，衰减越快
    // 1次失败: ×0.85, 2次: ×0.72, 3次: ×0.61, 5次: ×0.44, 10次: ×0.20
    s.DecayFactor *= math.Pow(0.85, float64(s.ConsecutiveFail))
}

// OnStreamBreak 断流时（比普通失败更严重）
func (s *FastDecayScore) OnStreamBreak() {
    s.ConsecutiveFail++
    s.DecayFactor *= math.Pow(0.70, float64(s.ConsecutiveFail)) // 更激进衰减
}

// EffectiveScore = BaseScore × DecayFactor
func (s *FastDecayScore) EffectiveScore() float64 {
    return s.BaseScore * s.DecayFactor
}
```

**触发条件**：`costTier=free|cheap` 或 `poolTag=temp` 的渠道自动启用 FastDecay。

**调度效果**：
- 一个白嫖渠道连续断流 3 次，EffectiveScore 从 1.0 降到 0.61，自动让位给下一个渠道
- 连续失败 10 次，降到 0.20，几乎不会被选中
- 成功一次立即回升 15%，避免"一朝被蛇咬"的永久惩罚
- 这比滑动窗口快得多：滑动窗口需要窗口滚动才反映变化，FastDecay 是请求级即时反应

## 7. API 设计

### 7.1 新增 API 端点

#### 渠道画像

```text
GET  /api/{kind}/channels/profiles          → 获取所有渠道画像
GET  /api/{kind}/channels/{id}/profile      → 获取单个渠道画像
POST /api/{kind}/channels/{id}/profile/refresh → 手动刷新画像
```

#### 模型画像

```text
GET  /api/{kind}/channels/{id}/model-profiles → 获取渠道下所有模型画像
```

#### 健康中心

```text
GET  /api/health-center/overview            → 全局健康概览（跨所有 kind）
GET  /api/health-center/channels            → 渠道健康列表（支持过滤/排序）
POST /api/health-center/batch-action        → 批量操作（复测/暂停/删除）
POST /api/health-center/channels/{id}/probe → 手动深度探测
```

#### 自动托管

```text
POST /api/{kind}/channels/auto-add          → 自动添加渠道（仅需 URL+Key）
POST /api/{kind}/channels/{id}/auto-discover → 重新触发自动发现
GET  /api/{kind}/channels/{id}/auto-status   → 自动托管状态
```

#### 智能路由（诊断用）

```text
POST /api/smart-routing/diagnose            → 智能路由诊断（dry-run）
GET  /api/smart-routing/config              → 获取自动路由配置
PUT  /api/smart-routing/config              → 更新自动路由配置
```

### 7.2 与现有 API 的关系

| 现有端点 | 变更 |
|---------|------|
| `POST /channel-discovery` | 不变，autopilot 内部复用其逻辑 |
| `POST /{kind}/channels` | 新增 `autoManaged` 字段，为 true 时自动触发 Discovery |
| `POST /{kind}/channels/{id}/capability-test` | 不变，autopilot 结果写入 ModelProfile |
| `POST /{kind}/channels/scheduler/diagnose` | 增加智能路由 trace 输出 |
| `GET /{kind}/channels/dashboard` | 增加 `healthState`、`qualityTier` 字段 |

### 7.3 WebSocket 推送

新增 `ws://api/autopilot/events` 通道，推送：

```json
{
  "type": "profile_updated",
  "channelId": 5,
  "channelKind": "messages",
  "healthState": "dead",
  "suggestedAction": "delete",
  "evidence": ["5/5 keys returned 401"]
}
```

事件类型：`profile_updated` / `health_changed` / `discovery_completed` / `auto_mapping_applied`

### 7.4 安全与权限边界

新增端点必须按管理接口处理：

- 所有 `/api/health-center/*`、`/api/smart-routing/*`、`/api/{kind}/channels/*/auto-*` 端点必须要求 `ADMIN_ACCESS_KEY` 或等价管理权限。
- `batch-action` 默认只允许 `refresh`、`probe`、`pause`，不默认开放删除；删除必须走现有单渠道删除流程并二次确认。
- WebSocket 必须鉴权，事件中禁止携带明文 API Key、Authorization header、自定义敏感 header、multipart 内容。
- 手动深度探测会请求真实上游，UI 必须显示本次操作会消耗上游额度；后台自动 L2 探测遵守全局预算。
- `auto-add` 写入配置前只保存用户提交的必要字段；探测日志只记录 key mask 和 metricsKey。

---

## 8. 前端设计

### 8.1 健康中心视图

新增 `HealthCenter.vue` 页面，作为渠道管理的高级视图。

#### 布局

```text
┌──────────────────────────────────────────────────────────┐
│ 渠道健康中心                    [批量复测] [批量暂停] [筛选] │
├──────────────────────────────────────────────────────────┤
│ ┌─ 统计卡片 ────────────────────────────────────────────┐│
│ │ 🟢 12 健康  🟡 3 降级  🟠 5 限流  🔴 4 死亡  ⚪ 2 新  ││
│ └───────────────────────────────────────────────────────┘│
│                                                          │
│ ┌─ 分组标签 ────────────────────────────────────────────┐│
│ │ [建议清理(4)] [需要修复(3)] [限流恢复中(5)]           ││
│ │ [质量较差(2)] [观察池(2)] [可用渠道(12)]              ││
│ └───────────────────────────────────────────────────────┘│
│                                                          │
│ ┌─ 渠道表格 ────────────────────────────────────────────┐│
│ │ 状态 │ 渠道 │ 协议 │ 模型数 │ 最后成功 │ 成功率 │p95 │ │
│ │      │      │      │        │          │        │延迟│ │
│ │ 🔴   │ xxx  │ chat │ 3/5    │ 72h前    │ 2%     │ -  │ │
│ │ 🟡   │ yyy  │ msgs │ 7/7    │ 5m前     │ 78%    │ 3s │ │
│ └───────────────────────────────────────────────────────┘│
│                                                          │
│ ┌─ 渠道详情侧栏（点击展开）─────────────────────────────┐│
│ │ 健康状态：🔴 Dead (confidence: 96%)                    ││
│ │ 死亡类型：硬死 - 全部 Key 认证失败                     ││
│ │ 证据：                                                 ││
│ │   • 5/5 keys returned 401                              ││
│ │   • capability-test failed 3 consecutive times         ││
│ │   • no successful request in 72h                       ││
│ │ 建议操作：[替换 Key] [删除渠道] [标记观察]             ││
│ │ 画像：quality=high, stability=unstable, speed=-        ││
│ │ 可用模型：gpt-5.4(✓), gpt-5.5(✗), ...                ││
│ └───────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────┘
```

#### 标签系统

每个渠道显示标签 chip：

| 标签 | 颜色 | 条件 |
|------|------|------|
| 高智商稳定 | 蓝 | qualityTier=high\|premium + stabilityTier=stable |
| 白嫖池 | 绿 | costTier=free |
| 临时池 | 橙 | 画像来源=auto_probe 且 confidence < 0.7 |
| 仅子代理 | 灰 | qualityTier=low + costTier=free\|cheap |
| 可识图 | 紫 | supportsVision=true |
| 长上下文 | 青 | supportsLongCtx=true |
| 全部 Key 失效 | 红 | evidence 含 "all keys failed" |
| 限流中 | 黄 | healthState=limited |
| 疑似配置错 | 紫 | healthState=misconfigured |

### 8.2 渠道卡片增强

在现有 `ChannelOrchestration.vue` 的每行中增加：

1. **健康状态 badge**：替换现有简单状态，使用 HealthState 六态 badge
2. **质量/稳定性/速度/成本标签**：在渠道名下方显示小 chip
3. **自动托管图标**：`autoManaged` 渠道显示机器人图标，hover 提示「自动托管中」
4. **一键操作**：死渠道显示「清理」快捷按钮
5. **有效成本预览**：在 key 详情中显示 `groupMultiplier / rechargeMultiplier = effectiveMultiplier` 和按模型估算的每百万 token 成本

### 8.3 添加渠道流程简化

当用户选择「快速添加」模式：

```text
┌─ 快速添加渠道 ──────────────────────────┐
│                                          │
│ 名称：[________________] (可选，自动生成) │
│ 地址：[https://xxx/v1________________]   │
│      [+ 添加 BaseURL]                    │
│ Key ：[sk-__________________________]    │
│      [+ 添加 Key]                        │
│                                          │
│ Key 成本（可选，展开编辑）                │
│ ┌ key mask │ 分组倍率 │ 充值倍率 │ 预估成本 ││
│ │ sk-***a  │ *:1.0   │ 1.0     │ 1.0x    ││
│ │ sk-***b  │ opus:1 │ 2.0     │ 0.5x    ││
│ └──────────────────────────────────────┘│
│                                          │
│ [x] 自动托管（推荐）                     │
│     系统将自动探测协议、发现模型、        │
│     生成映射、持续监控健康               │
│                                          │
│          [添加并探测]                    │
└──────────────────────────────────────────┘
```

点击「添加并探测」后：
1. 创建渠道（status=unknown）
2. 自动触发 Discovery + 能力测试
3. 显示进度条和实时探测日志
4. 完成后生成映射建议；仅当所有 endpoint 结果一致时自动写入 channel 级 modelMapping / supportedModels / 兼容开关
5. 生成初始 ChannelProfile

---

## 9. 配置设计

### 9.1 全局智能路由配置

在 `config.json` 新增顶层字段：

```json
{
  "smartRouting": {
    "enabled": true,
    "mode": "auto",

    "defaultAutoManaged": true,
    "autoDiscoveryOnAdd": true,

    "subagentUseCheapPool": true,
    "unknownChannelPolicy": "observe",
    "premiumFallbackForSubagent": false,
    "protectVisionChannels": true,
    "protectLongContextChannels": true,

    "healthCheck": {
      "enabled": true,
      "passiveSignalsOnly": false,
      "l2ProbeEnabled": true,
      "l2ProbeIntervalMinutes": 120,
      "l2ProbeMaxPerDay": 12,
      "deadProbeIntervalHours": 6,
      "deadConfidenceThreshold": 0.80,
      "autoExcludeDead": true
    },

    "fastDecay": {
      "enabled": true,
      "applyToCostTiers": ["free", "cheap"],
      "applyToPoolTags": ["temp"],
      "recoveryRate": 0.15,
      "decayBase": 0.85,
      "streamBreakDecayBase": 0.70
    },

    "rateLimitDiscovery": {
      "enabled": true,
      "applyOnlyWhenUnset": true,
      "preferHeaders": true,
      "passiveAimdEnabled": true,
      "minRpm": 1,
      "maxAutoRpm": 120,
      "confidenceThreshold": 0.6,
      "increaseIntervalMinutes": 10,
      "increaseStepPercent": 10,
      "decreaseOn429Percent": 50
    },

    "modelMapping": {
      "autoResolve": true,
      "capabilityFloorEnabled": true,
      "echoMappedModel": true,
      "forbidChainMapping": true
    },

    "costOptimization": {
      "enabled": true,
      "applyAfterQualityFloor": true,
      "requireCostConfidence": 0.6,
      "preferLowerEffectiveCost": true,
      "supervisorSavingsWeight": 0.5,
      "workerSavingsWeight": 3
    },

    "taskStrategies": {
      "supervisor": {
        "preferQuality": ["high", "premium"],
        "requireStability": ["stable", "normal"],
        "excludeTags": ["unstable", "free"]
      },
      "worker": {
        "preferCost": ["free", "cheap"],
        "preferSpeed": ["fast"],
        "excludeQuality": ["premium"]
      }
    }
  }
}
```

### 9.2 渠道级配置

现有 `UpstreamConfig` 新增字段：

```go
type UpstreamConfig struct {
    // ... 现有字段 ...

    // ── 稳定身份 ──
    ChannelUID string `json:"channelUid,omitempty"` // 渠道稳定 ID，ConfigManager 负责补齐旧配置

    // ── 自动托管 ──
    AutoManaged       bool   `json:"autoManaged,omitempty"`       // 启用自动托管
    AutoManagedAt     *time.Time `json:"autoManagedAt,omitempty"` // 开始托管时间
    CostHint          string `json:"costHint,omitempty"`          // 用户成本提示：free/cheap/normal/expensive
    GroupMultipliers  map[string]float64 `json:"groupMultipliers,omitempty"` // channel 默认分组倍率，key 可覆盖
    RechargeMultiplier float64 `json:"rechargeMultiplier,omitempty"` // channel 默认充值倍率，key 可覆盖
    QualityHint       string `json:"qualityHint,omitempty"`       // 用户质量提示（override 自动推导）
    PoolTag           string `json:"poolTag,omitempty"`           // 池标签：temp/regular/premium
    RoutingPriority   string `json:"routingPriority,omitempty"`   // 路由优先级 hint
}
```

**用户覆盖规则**：
- 用户手动设置的字段（QualityHint/CostHint/PoolTag）优先级高于自动推导
- 但自动推导的运行时指标（健康状态/熔断）始终生效，不受 override 影响
- 用户可通过 `autoManaged: false` 退出自动托管，回到手动模式

### 9.3 Key 级成本倍率配置

用户添加 `baseUrls[]` 和 `apiKeys[]` 后，必须能给每个 key 标注成本倍率。成本倍率属于 key 级语义，因为同一 baseURL 下不同 key 可能属于不同分组、租户或充值批次。

扩展 `APIKeyConfig`：

```go
type APIKeyConfig struct {
    // ... 现有字段 ...

    // 成本分组名称，仅用于 UI 展示和批量编辑；不要与 QuotaGroup 混用。
    CostGroup string `json:"costGroup,omitempty"`

    // 分组倍率：key 为模型组或通配符，例如 "*"、"claude-opus"、"gpt-5"、"gemini"。
    // 未命中时回退 channel.GroupMultipliers，再回退 1.0。
    GroupMultipliers map[string]float64 `json:"groupMultipliers,omitempty"`

    // 充值倍率：1.0=无折扣；2.0=付 1 得 2，真实成本减半。
    // 未设置时回退 channel.RechargeMultiplier，再回退 1.0。
    RechargeMultiplier float64 `json:"rechargeMultiplier,omitempty"`

    // 可选：用户直接给出的成本备注，例如 "8折企业组/双倍充值"。
    CostNote string `json:"costNote,omitempty"`
}
```

配置示例：

```json
{
  "baseUrls": ["https://proxy-a.example/v1", "https://proxy-b.example/v1"],
  "apiKeys": ["sk-key-a", "sk-key-b"],
  "apiKeyConfigs": [
    {
      "key": "sk-key-a",
      "name": "A 组",
      "costGroup": "vip-a",
      "groupMultipliers": { "*": 1.0, "claude-opus": 1.2 },
      "rechargeMultiplier": 1.0
    },
    {
      "key": "sk-key-b",
      "name": "充值活动组",
      "costGroup": "promo-b",
      "groupMultipliers": { "*": 1.0, "claude-opus": 1.0 },
      "rechargeMultiplier": 2.0
    }
  ]
}
```

解释：
- `sk-key-a` 的 Opus 成本是官价 `1.2x`。
- `sk-key-b` 的 Opus 成本是官价 `1.0 / 2.0 = 0.5x`。
- 如果两者质量、健康、上下文和能力都满足，worker/lightweight 请求优先 `sk-key-b`。

`baseUrls[] × apiKeys[]` 会生成多个 endpoint，但同一个 key 的成本配置默认应用到所有 baseURL。如果同一个 key 在不同 baseURL 上有不同价格，优先建议拆成两个 channel；不要在 MVP 中加入 baseURL 级成本 override，避免配置复杂度失控。

### 9.4 endpoint 级自动结果的落点

现有 `UpstreamConfig.ModelMapping`、`SupportedModels`、兼容开关都是 channel 级字段，而本方案的发现结果是 endpoint 级。落地规则：

```text
所有 endpoint 一致：
  可安全写入 channel 级 ModelMapping / SupportedModels / 兼容开关

endpoint 间不一致：
  写入 ProfileStore 的 KeyEndpointProfile / ModelProfile
  UI 显示不一致警告
  请求路径通过 EndpointAttemptPolicy 做 request-scoped 决策
  不自动覆盖 channel 级 ModelMapping / SupportedModels
```

这避免一个 channel 内 key-A 支持 Claude 系列、key-B 支持 GPT 系列时，自动配置把 channel 级模型白名单写成错误并集。

### 9.5 后台任务生命周期

Autopilot 需要一个后台 worker，但必须是保守、可停止、可观测的：

```text
启动：
  - 加载 ProfileStore
  - 为旧 UpstreamConfig 补齐 ChannelUID
  - 同步当前配置 index 快照
  - 启动 L1 聚合、L2 探测队列、过期数据清理

配置热重载：
  - 新 ChannelUID：创建 unknown profile
  - baseURL/apiKey/serviceType 变化：重算 endpoint 集合，新增 unknown，旧 endpoint 标记 orphaned
  - channel 重排：只更新 channel_id 展示快照，不迁移画像主键
  - channel 删除：标记 orphaned，延迟清理画像/证据/快照

关闭：
  - 停止探测队列
  - flush 内存时间桶
  - 不中断正在处理的真实用户请求
```

多实例部署先按“best effort”处理：Phase 1 不做跨进程锁；Phase 2 若要自动 L2/L3 探测，需要在 SQLite 增加 `autopilot_jobs(locked_until, owner)`，避免多个进程重复烧 quota。

---

## 10. 分阶段落地计划

### Phase 1：自动画像 + 健康诊断（MVP）

**目标**：用户能看到渠道健康状态（精确到 endpoint 级），系统自动推导画像，但不改变真实调度结果。Phase 1 是 shadow/read-only 阶段。

**范围**：
- [ ] `ChannelUID` 稳定身份补齐 + 配置热重载同步
- [ ] KeyEndpointProfile / ChannelProfile(聚合) / ModelProfile 数据模型 + SQLite 存储
- [ ] Profiler 画像推导（基于现有 MetricsManager + 模型注册表，L1 被动信号为主，endpoint 级粒度）
- [ ] CostProfile shadow 计算（key 级分组倍率/充值倍率 → endpoint 有效成本，仅展示）
- [ ] HealthAnalyzer 健康诊断（被动优先：L1 为主，endpoint 级判定 + channel 级聚合）
- [ ] FastDecay shadow score（只展示/诊断，不参与调度）
- [ ] RateLimitDiscoverer shadow profile（解析响应头和 429 模式，只展示建议 RPM，不改 limiter）
- [ ] GroupChangeDetector 快照对比（仅手动刷新或已有 discovery/capability-test 后更新，不自动重探测）
- [ ] 健康中心 API（`/api/health-center/*`，支持 endpoint 级展开）
- [ ] 前端健康中心视图
- [ ] 渠道卡片健康 badge 增强（显示 endpoint 级不一致警告）
- [ ] 标签系统（白嫖池/临时池/高智商稳定等）

**不做的事**：
- 不修改调度器核心链路
- 不让 FastDecay / HealthState 影响真实调度
- 不让 CostProfile 影响真实调度
- 不让自动发现的 RPM 影响真实 limiter
- 不自动触发 L2/L3 探测
- 不自动写入 modelMapping
- 不做模型自动映射

**预估工期**：2-3 周

### Phase 2：自动发现 + 智能调度

**目标**：添加渠道时自动发现，调度时能按 channel + endpoint 画像选择，但仍不做动态模型自动映射。

**范围**：
- [ ] `autoManaged` 字段 + 快速添加流程
- [ ] Discovery 自动触发（复用现有 channel_discovery 逻辑）
- [ ] 自动写入一致的 channel 级 modelMapping / supportedModels / 兼容开关；不一致结果保留在 ProfileStore
- [ ] SmartRouter + CandidateFilter 注入（保留 X-Channel/manual override/promotion 优先级）
- [ ] EndpointAttemptPolicy 注入 `TryUpstreamWithAllKeys`，支持 endpoint 级 baseURL/key 排序
- [ ] Cost-aware routing：满足质量/能力下界后，按 `estimatedRequestCost` 降低总费用
- [ ] FastDecay 从 shadow score 切换为 endpoint 调度降权
- [ ] RateLimitDiscoverer 对未显式设置 RPM 的 endpoint 应用 runtime limiter
- [ ] L2 轻量探测 worker + 每日预算 + 探测队列
- [ ] TaskClassifier + 五种任务策略
- [ ] 智能路由诊断 API（dry-run）
- [ ] 前端自动托管指示器 + 快速添加 UI

**预估工期**：3-4 周

### Phase 3：动态画像 + 自愈

**目标**：画像实时更新，渠道自动恢复。

**范围**：
- [ ] 运行时指标驱动画像实时更新
- [ ] 模型自动映射（ModelResolver + ModelSupportResolver + request-scoped mappedModel）
- [ ] active model filter / context filter 支持自动映射前置判定
- [ ] 自动恢复探测（limited/dead → healthy）
- [ ] 晋升/降级机制（连续成功→升级，连续失败→降级）
- [ ] WebSocket 推送画像变更事件
- [ ] 前端画像变更历史/时间线

**预估工期**：2-3 周

### Phase 4：高级特性

**范围**：
- [ ] 多维度标签系统扩展（用户自定义标签）
- [ ] A/B 测试：同一请求走不同渠道对比
- [ ] 成本报表：按用户/模型/渠道/key 统计真实有效成本
- [ ] 批量渠道管理（导入/导出/模板）
- [ ] 渠道推荐：根据使用模式推荐新渠道

---

## 11. 关键代码锚点

### 11.1 需要扩展的文件

| 文件 | 行号 | 扩展内容 |
|------|------|---------|
| `config/config.go` | `UpstreamConfig` / `APIKeyConfig` | 新增 ChannelUID/AutoManaged/CostHint/QualityHint/PoolTag，以及 key 级 GroupMultipliers/RechargeMultiplier |
| `config/config_loader.go` | 配置加载/保存 | 为旧配置补齐 ChannelUID，并保证重排不改变 UID |
| `scheduler/select.go` | `SelectChannelWithOptions` | 调整显式控制与 CandidateFilter 顺序；支持 ModelSupportResolver 前置判断 |
| `scheduler/recovery.go` | `SelectionOptions` | 必要时携带 protected channels / routing hints |
| `handlers/common/multi_channel_failover.go` | `HandleMultiChannelFailover*` | 构建 RequestProfile，传入 SmartRouter plan |
| `handlers/common/upstream_failover.go` | `TryUpstreamWithAllKeys` | 新增 WithEndpointAttemptPolicy，按 endpoint 画像排序 baseURL/key |
| `keypool/keyconfig.go` | `CandidatesForModel` | 支持 endpoint policy 过滤/排序后的 key 候选 |
| `ratelimit/hints.go` | `ApplyUpstreamHints` | 复用现有限流响应头解析；补充 limit/window 信号输出给 RateLimitDiscoverer |
| `ratelimit/limiter.go` | `ChannelLimiter` | 支持从 discovered runtime config 更新 RPM，但不覆盖手动配置 |
| `metrics/channel_metrics.go` | `MetricsManager` | 新增画像相关查询方法 |
| `metrics/cost.go` | 成本估算 | 支持 CostProfile 的 effective multiplier，不只使用模型官价 |
| `handlers/channel_discovery.go` | 全文 | Profiler 复用其探测逻辑 |
| `handlers/capability_test_runner.go` | `runCapabilityTestJob` | 测试结果写入 ModelProfile |

### 11.2 需要新增的文件

```text
backend-go/internal/autopilot/
├── key_endpoint_profile.go    # KeyEndpointProfile 类型（画像最小单元）
├── channel_profile.go         # ChannelProfile 聚合视图
├── model_profile.go           # ModelProfile 类型
├── request_profile.go         # RequestProfile + TaskClassifier
├── profile_store.go           # SQLite 持久化 + 内存缓存（endpoint 级）
├── group_change_detector.go   # Key 分组变更检测
├── time_series.go             # TimeBucketMetrics + 时序存储
├── quality_trend_detector.go  # QualityTrend 检测（忙闲/趋势/时段模式）
├── profiler.go                # 画像推导逻辑（L1 被动信号为主）
├── health_analyzer.go         # 健康诊断逻辑（被动优先 + 趋势信号 + L2 探测）
├── cost_resolver.go           # key 级分组倍率/充值倍率 → endpoint 有效成本
├── fast_decay.go              # 白嫖/临时池快速衰减评分
├── rate_limit_discovery.go    # 上游 RPM/并发限制自动发现（header + passive AIMD）
├── smart_router.go            # SmartRouter + CandidateFilter 构建
├── endpoint_policy.go         # EndpointAttemptPolicy + endpoint 候选排序
├── model_support_resolver.go  # active model filter 前置的自动支持判断
├── model_resolver.go          # 模型自动映射 + CapabilityFloor 约束
├── worker.go                  # L1 聚合 / L2 探测 / 清理后台任务
└── handlers.go                # API handlers

backend-go/internal/autopilot/
├── autopilot_test.go          # 画像推导测试
├── health_analyzer_test.go    # 健康诊断测试
├── cost_resolver_test.go      # 倍率继承、模型组匹配、有效成本计算测试
├── fast_decay_test.go         # 快速衰减测试
├── rate_limit_discovery_test.go # RPM header 解析与 AIMD 收敛测试
├── group_change_test.go       # 分组变更检测测试
├── quality_trend_test.go      # 趋势检测测试（忙闲模式识别）
├── endpoint_policy_test.go    # endpoint 排序与 fail-open/fail-closed 测试
├── model_support_resolver_test.go # supportedModels 前置兼容测试
└── smart_router_test.go       # 路由策略测试

frontend/src/components/
├── HealthCenter.vue        # 健康中心主视图
├── HealthCenterStats.vue   # 统计卡片
├── HealthChannelTable.vue  # 渠道健康表格
├── HealthChannelDetail.vue # 渠道详情侧栏
├── QuickAddChannel.vue     # 快速添加渠道
└── ChannelHealthBadge.vue  # 健康状态 badge（增强现有）
```

### 11.3 与现有代码的接口契约

| 接口 | 方向 | 说明 |
|------|------|------|
| `CandidateFilterFunc` | autopilot → scheduler | SmartRouter 用于 channel 级粗筛/重排 |
| `EndpointAttemptPolicy` | autopilot → common failover | SmartRouter 用于 endpoint 级 baseURL/key 细选 |
| `ModelSupportResolver` | autopilot → scheduler | 在 active model filter 阶段判断 autoManaged 渠道是否可 request-scoped 映射 |
| `RateLimitDiscoverer` | autopilot → ratelimit | 未显式配置 RPM 时提供 endpoint/key/quota scope 的 runtime limiter 建议 |
| `CostResolver` | autopilot → scheduler/common failover | 在满足质量和能力下界后，为 endpoint 排序提供 `estimatedRequestCost` |
| `MetricsManager.GetChannelAggregatedMetrics` | autopilot ← metrics | 画像推导读取运行时指标（入参是 baseURL + activeKeys + serviceType） |
| `MetricsManager.GetTimeWindowStatsForKey` | autopilot ← metrics | **endpoint 级**指标查询 |
| `config.ResolveUpstreamCapability` | autopilot ← config | 画像推导读取模型能力 |
| `config.ResolveAgentModelProfile` | autopilot ← config | RequestProfile 推导质量需求 |
| `channelDiscovery.*` | autopilot ← handlers | 复用探测逻辑（需抽取为可复用函数，支持单 endpoint 调用） |
| `PersistenceStore` | autopilot → metrics | 新表复用同一 SQLite 连接 |

---

## 12. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| ~~自动模型映射语义降级~~ | ~~用户以为用 opus 实际用 haiku~~ | **已缓解**：CapabilityFloor 硬约束 + 不满足则跳过渠道而非降级映射 + response header 回显真实模型 |
| ~~健康诊断烧 quota~~ | ~~30-40 渠道主动探测成本高~~ | **已缓解**：L1 被动优先（零成本），L2 仅在数据不足时触发，每天总次数上限 `渠道数×12` |
| ~~白嫖池状态抖动~~ | ~~渠道反复断流导致调度震荡~~ | **已缓解**：FastDecay 请求级即时衰减 + 成功快速回升 + 断流比普通失败衰减更快 |
| ~~能力虚标误判~~ | ~~系统误关 vision/tool 标签~~ | **已缓解**：只做硬失败检测（HTTP 错误/解析失败），软质量问题留给人工 override |
| channel index 漂移 | 重排/删除后画像串到其他渠道 | 新增 `ChannelUID` 作为画像主键；`channelId` 仅作当前 index 展示快照 |
| endpoint 决策无法落到真实请求 | SmartRouter 只重排 channel，无法选择具体 key/baseURL | 新增 `EndpointAttemptPolicy` 注入 `TryUpstreamWithAllKeys`，在 failover 层过滤/排序 endpoint |
| 自动映射被 supportedModels 前置过滤挡掉 | ModelResolver 没机会运行，autoManaged 渠道被提前剔除 | 新增 `ModelSupportResolver` 接管 active model filter 的支持判断 |
| 成本倍率配置错误 | 用户看到的具体费用不准，调度可能选错 key | 倍率必须显式展示来源和公式；低置信度成本只做 tie-breaker；UI 提供按 key 的有效价格预览 |
| 为省钱牺牲质量 | supervisor/vision/long-context 被路由到低质量或能力不足 endpoint | 成本排序只在 `CapabilityFloor`、`MinQualityTier`、上下文和能力硬约束通过后执行 |
| 自动 RPM 发现过于激进 | 误把免费/低配额上游打到 429 或封禁 | 只在未显式设置时启用；优先 header；无 header 用保守 AIMD；不主动压测；`maxAutoRPM` 封顶 |
| 自动 RPM 覆盖用户意图 | 用户手动设置被 runtime 学习值覆盖 | 手动 channel/key `RateLimitRPM` 永远优先；自动值只展示或用于未设置场景的 runtime limiter |
| endpoint 级探测成本倍增 | 5 key × 3 baseURL = 15 endpoint，探测量是 channel 级的 15 倍 | endpoint 级探测轮转执行，不并发；L2 探测每日总量仍有上限；channel 级聚合结果缓存避免重复计算 |
| Key 换分组后模型列表突变 | 调度时才发现模型不可用 | GroupChangeDetector 在 L2 探测和 model_not_found 时自动检测；分组变更立即标记 stale 并重探测 |
| endpoint 间能力不一致导致配置污染 | 同一 channel 有的 endpoint 支持 vision 有的不支持，自动写 channel 级配置会误导调度 | 只有所有 endpoint 一致时才写 channel 级配置；不一致结果保留 ProfileStore 并通过 EndpointAttemptPolicy request-scoped 生效 |
| SmartRouter 增加调度延迟 | 请求耗时增加 | 画像缓存在内存，CandidateFilter 只做内存操作（< 1ms）；聚合视图预计算，非实时聚合 |
| 与现有 X-Channel/manual override/promotion 冲突 | 用户显式选择被自动调度过滤 | 调整调度阶段或传入 ProtectedChannels；显式控制优先，SmartRouter 不过滤受保护渠道 |
| Phase 1 无智能调度时画像价值不明显 | 用户感知弱 | 明确 Phase 1 为 shadow/read-only；健康中心 + dry-run 诊断提前展示启用后效果 |
| 自动 modelMapping 覆盖用户手动配置 | 用户设置被意外覆盖 | 显式 modelMapping 始终优先；请求路径自动映射默认 request-scoped，不直接写配置 |
| 被动信号在低流量渠道不足 | 新渠道/冷渠道无法诊断 | 低流量时自动降级为 L2 探测，探测频率随请求量动态调整 |
| 上游忙闲时段质量差异大 | 闲时探测通过但忙时不可用 | TimeBucketMetrics 15 分钟桶 + QualityTrend 时段模式识别；忙时自动降档而非标记 dead；SmartRouter 在忙时倾向非低谷 endpoint |
| 趋势检测误判（短期波动 vs 真实恶化） | 频繁触发重评估浪费资源 | degrading 需要同时满足 24h 和 7d 双基准下降才触发重评估；volatile 状态只调 FastDecay 系数，不触发重评估 |
| 多实例重复探测 | 多进程同时 L2/L3 探测导致 quota 消耗翻倍 | Phase 1 不自动探测；Phase 2 引入 `autopilot_jobs(locked_until, owner)` 后再启用自动 worker |
| 管理 API 误操作 | 批量删除/暂停影响生产渠道 | 新端点要求管理权限；批量默认仅支持 refresh/probe/pause，删除走现有确认流程 |
