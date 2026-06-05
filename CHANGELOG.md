## [v2.8.24] - 2026-06-04

### 修复

- **修复深层嵌套图片检测遗漏导致视觉回退失效** - 修复嵌套结构中图片 URL 未被识别，导致视觉回退策略无法生效的问题

## [v2.8.23] - 2026-06-04

## [v2.8.22] - 2026-06-04

### 修复

- **修复桌面端管理面板 CORS 认证失败** - 修复 `OPTIONS` 预检请求被认证中间件拦截返回 401，导致桌面端 WebView 无法访问管理接口的问题；`ENABLE_CORS=false` 时仍为带 `Origin` 的请求注入 CORS 响应头
- **version-bump 新增 CHANGELOG 覆盖率校验** - 版本升级流程新增 CHANGELOG 覆盖率校验，防止遗漏提交记录

### 文档

- **补全 v2.8.21 CHANGELOG** - 补全 v2.8.21 版本遗漏的 CHANGELOG 条目
- **新增 Windows 桌面图标透明度待办项** - 新增 Windows 桌面图标透明度相关的待办任务

## [v2.8.21] - 2026-06-04

### 新增

- **桌面端原生管理控制台** - 桌面端原生管理控制台替代 iframe WebUI，完整覆盖 i18n，管理入口统一命名为 Dashboard
- **桌面端渠道管理重构** - 添加渠道简化为快速粘贴模式（支持 Cmd/Ctrl+Enter 创建），编辑渠道弹窗对齐 WebUI 核心交互，新增预设模板、模型拉取、高级选项分组、Base URL 预览、服务类型动态排序、模型映射协议预置列表与 filter chips、API Key 必填校验
- **桌面端渠道操作增强** - 渠道卡片支持置顶置底并增加删除守卫，渠道日志弹窗重构为可展开列表并支持自动刷新，删除渠道时显示 loading 遮罩，模型重定向 target 输入框始终提供候选
- **桌面端会话驾驶舱重构** - 会话驾驶舱重构为瀑布流卡片并对齐 WebUI cockpit，能力测试对话框对齐 WebUI，能力测试多 job 并发轮询修复
- **桌面端 Codex 会话迁移** - 添加 Codex 会话迁移支持，快捷/插件模式切换对齐 WebUI
- **CLI --backupdir 选项** - 新增 `--backupdir` 选项指定配置备份目录
- **CLI 自定义运行时路径** - 支持自定义运行时路径
- **LOG_DIR=none 禁用日志文件** - 支持 `LOG_DIR=none` 环境变量禁用日志文件写入
- **渠道级请求超时配置** - 后端渠道配置新增 `requestTimeoutMs`，非流式上游请求优先使用渠道超时，未设置或为 0 时继承全局 `REQUEST_TIMEOUT`

### 修复

- **messages**: system 角色归一化对所有上游生效
- **日志 store 改用 metricsKey 分桶** - 根治删除渠道后日志串桶问题
- **模型过滤拒绝非法字符** - 模型过滤拒绝顿号等非法字符并按分隔符拆分
- **余额不足/套餐过期自动拉黑** - 补全 403 关键词自动拉黑，改为双词组合匹配覆盖更多上游变体
- **下架已停用 GPT 模型** - 从能力测试候选、模型重定向候选和预设中移除 `gpt-5.2` / `gpt-5.2-codex` / `gpt-5.3-codex`，haiku 映射改为 `gpt-5.4-mini`
- **桌面端编辑渠道大量 UI 修复** - 快捷键与底部操作栏、Esc/Enter 快捷键、窄窗口布局、高级选项平铺 Runtime/Transport 分组、Website 移入连接分组、模型卡片布局、noVisionModels 合并行级 toggle、思考强度/Text verbosity 下拉修复、popover 主题变量补全、移除冗余 Latency 列与重复快捷键提示、保存快捷键统一为 Cmd/Ctrl+Enter
- **桌面端快速粘贴优化** - Textarea 高度修正、自动应用识别结果、自动补全 name/serviceType、按钮提示显示平台对应修饰符、移除 Service Type 识别卡片
- **桌面端 Dashboard 修复** - 移除轮询可见性检查修复渠道列表不自动刷新、管理面板 Tabs 配色对齐深色主题、备用池文案对齐 WebUI、驾驶舱命名对齐并移除手动刷新、网关监控按钮改为「打开管理面板」
- **桌面端模型列表修复** - 拉取补传 key 参数修复 'No API key provided' 错误、源模型列表对齐 WebUI allSourceModelOptions、预设数据与按钮分组完全对齐 WebUI
- **桌面端移除一键配置已下线模板** - 移除 gpt-5.3/gpt-5.2 模板，haiku 改映射到 gpt-5.4-mini
- **桌面端渠道中心恢复原有 ChannelTab 界面** - 移除渠道测速按钮，补齐编辑弹窗快捷键
- **桌面端 i18n 补全** - 编辑渠道界面、编辑弹窗新增 i18n keys 等缺失翻译补全

### 重构

- **桌面端管理入口命名统一** - 管理入口统一命名为 Dashboard，隐藏批量测速按钮，移除 Codex 分组冗余开关
- **桌面端编辑渠道布局对齐** - 请求超时从基础信息移入 Runtime 分组，抽取渠道草稿保存逻辑统一提交路径
- **渠道管理清理** - 移除未使用代码（开启 noUnusedLocals）、删除渠道编辑弹窗冗余元素

### 文档

- **Docker 部署示例修正** - 修正 Docker 部署示例与实际 compose 保持一致
- **LOG_DIR=none 补充文档** - 补充 `LOG_DIR=none` 禁用日志文件文档及测试用例
- **新增 TODO 待办任务文档**

## [v2.8.20] - 2026-06-01

### 新增

- **渠道级请求超时配置** - 后端渠道配置新增 `requestTimeoutMs`，非流式上游请求优先使用渠道超时，未设置或为 0 时继承全局 `REQUEST_TIMEOUT`；五类渠道新增/更新路径均校验负数并支持清空回退。
- **Web UI 渠道请求超时表单** - 渠道编辑表单新增“请求超时 (ms)”输入项，支持编辑态清空恢复继承全局，并在复制渠道、payload 序列化、差异检测和多语言文案中同步该字段。

### 新增

- **桌面端 Codex Agent 配置双模式** - Codex 支持两种 CCX 模式：快捷模式（OpenAI provider + CCX 代理，不迁移会话、不支持插件）和插件模式（CCX 原生 provider + `requires_openai_auth = true`，支持插件但切换需会话迁移提示）。UI 下拉框拆分为两个选项，互切时显示会话迁移警告。
- **Codex 配置改用快捷/插件模式开关** - 桌面端 Codex 配置入口统一改为快捷模式/插件模式开关，明确不同接入路径。
- **Codex provider 块统一使用 `env_key`** - 将 `[model_providers.xxx]` 块中的 `temp_env_key` 统一替换为官方文档字段 `env_key`。
- **桌面端 AgentCard 编辑器打开支持多编辑器选择** - 配置文件和认证文件的"用编辑器打开"按钮现在支持多编辑器场景，通过透明 select 覆盖层提供编辑器选择下拉。
- **渠道中心已保存 key 的 preset 优先排列** - 桌面端渠道中心将已保存 API Key 的 preset 优先展示，便于快速选择常用渠道。
- **已配置 key 的渠道在渠道中心优先展示** - Web 前端渠道中心根据已配置 Key 状态优先排列渠道。
- **Codex 渠道模板新增 codex-auto-review 重定向** - 渠道模板新增 codex-auto-review 重定向能力。
- **新增 WebUI 新用户指引** - WebUI 增加新用户引导说明，帮助首次使用者快速理解渠道配置流程。
- **OpenAI 直连增加 API Key 勾选框** - 桌面端 OpenAI 直连新增「我有自己的 API Key」勾选框。
- **新用户指引添加渠道展示模拟表单** - 新用户指引增加渠道展示模拟表单，更直观展示快速模式字段。
- **Codex 第三方 provider 支持快捷/插件模式切换** - 第三方 Codex provider 支持在快捷模式与插件模式之间切换。

### 修复

- **下架已停用 GPT 模型** - 从能力测试候选、模型重定向候选和预设中移除 `gpt-5.2` / `gpt-5.2-codex` / `gpt-5.3-codex`，haiku 层模型统一替换为 `gpt-5.4-mini`，并为桌面端渠道模板补充旧模型回归检查。

- **浏览器直达移除 `ccx_desktop` 参数** - 修复浏览器直达 WebUI 时残留桌面端参数的问题。
- **默认语言改为中文并增加浏览器语言检测** - 修复默认语言选择逻辑，新增浏览器语言检测。
- **新用户指引协议切换说明拆分为三段** - 调整新用户指引中协议切换说明的展示结构。
- **新用户指引简化标签列表** - 将新用户指引标签列表简化为 Claude / Codex / Gemini 等。
- **新用户指引示例渠道改为 gpt-5.5 中转** - 更新新用户指引中的示例渠道文案。
- **新用户指引删除 quickHint** - 删除 quickHint，改为说明 CCX 自动识别。
- **删除新用户指引模拟表单及关联资源** - 删除模拟表单及其关联 i18n/CSS 资源，简化引导内容。
- **新用户指引添加渠道说明拆分为两段** - 调整添加渠道说明，使内容层次更清晰。
- **渠道置顶后保持渠道顺序** - 修复渠道置顶后顺序被打乱的问题。
- **修正 Codex 认证配置模式** - 修复 Codex 认证配置模式写入与展示不一致的问题。
- **渠道排序以 priority 优先，hasKey 仅作兜底** - 修复渠道排序中 hasKey 权重过高导致 priority 失效的问题。
- **消除 Codex 非 OpenAI provider 时悬空分隔符** - 修复 Codex 非 OpenAI provider 表单中多余竖线分隔符。
- **调整 Linux 数据目录到 XDG state** - 修复 Linux 桌面端数据目录位置，改用 XDG state 路径。
- **掩码 Codex 配置预览密钥** - 修复 Codex 配置预览中密钥未正确脱敏的问题。
- **修正 Codex `auth_mode` 为 `apikey`** - 修正 Codex 配置中的认证模式字段值。
- **OpenAI 直连取消勾选 API Key 时清空残留值** - 修复 OpenAI 直连关闭自带 Key 后仍保留旧值的问题。
- **修正 Codex 配置预览 `auth_mode` 被误掩码** - 修复配置预览中普通字段被误判为密钥的问题。
- **对齐 Codex 插件模式认证配置** - 修复 Codex 插件模式下认证配置与预期不一致的问题。
- **JSON 配置 diff 改为字段级掩码** - 修复 JSON 配置 diff 脱敏粒度过粗导致预览不准确的问题。
- **修正短密钥掩码后 JSON diff 漏报变更** - 修复短密钥脱敏后 JSON diff 无法识别真实变更的问题。
- **按字段掩码 Codex 配置 diff** - Codex 配置 diff 改为按字段掩码，避免误掩码普通配置项。
- **OpenAI 直连自带 Key 必须输入** - 修复 OpenAI 直连勾选自带 Key 时允许空值的问题。

### 重构

- **移除左上角 Logo 点击跳转 GitHub 链接** - 移除 UI 左上角 Logo 的 GitHub 跳转行为。
- **配置预览文件标签改用增删行数统计** - 桌面端配置预览文件标签改为展示增删行数统计。

### 文档

- **更新桌面安装文档** - 更新桌面安装文档，补充 Windows Store 已上架和 Homebrew 安装说明。
- **排障指南添加各入口 curl 快速验证命令** - 为排障指南补充各协议入口的 curl 快速验证命令。
- **排障指南首条添加关闭全局代理和 TUN** - 排障指南首条新增关闭全局代理和 TUN 的排查建议。

### 其他

- **修复 go fmt 格式化缩进** - 修正 Go 格式化后的缩进样式。
- **移除 Claude-Reasoning-Debug 调试日志** - 移除 Claude-Reasoning-Debug 调试日志及 summarizeAssistantReasoningState 函数。
- **精简新用户指引内容** - 移除新用户指引中的状态图例、开关和接入客户端三节。
- **补充 desktop type-check 脚本** - 为桌面前端补充 type-check 脚本。
- **固化 Wails3 CLI 安装版本** - CI 中固定 Wails3 CLI 安装版本，提升构建稳定性。

## [v2.8.19] - 2026-05-30

### 修复

- **补全 MiMo target 描述删除遗漏** - 补全桌面端 MiMo target 描述删除时的遗漏处理。

## [v2.8.18] - 2026-05-30

### 新增

- **按 ReasoningParamStyle 转换客户端原始 reasoning 参数** - 新增按 ReasoningParamStyle 配置项转换客户端原始 reasoning 参数的能力。
- **Passthrough 路径 ReasoningMapping 分支补充 reasoning_effort 样式处理** - 在 passthrough 路径的 ReasoningMapping 分支中补充 reasoning_effort 样式的处理逻辑。
- **桌面端 add Homebrew tap 自动化** - 新增 Homebrew tap 自动化发布流程。
- **桌面端点击版本号触发手动更新检查并计入冷却时间** - 点击版本号可手动触发更新检查，并计入冷却时间避免频繁检查。
- **桌面端 Codex CCX 代理切换为 openai_base_url 官方模式** - Codex CCX 代理从自定义模式切换为 openai_base_url 官方模式。

### 修复

- **渠道中心布局断点降低至 md 并修复窄屏 URL 溢出** - 修复渠道中心在窄屏下的布局问题，将断点降低至 md 并修复 URL 文本溢出。
- **Responses API 返回 426 让 Codex WebSocket 回退到 HTTP POST** - 修复 Responses API 返回 426 状态码时 Codex WebSocket 正确回退到 HTTP POST。
- **修复断裂文档链接，统一 PORT=3688，补充源码构建前置依赖** - 修复文档中断裂的链接，统一 PORT 配置为 3688，补充源码构建的前置依赖说明。
- **MiMo messages 目标描述统一为通用表述** - 统一桌面端 MiMo messages 渠道目标描述为通用表述。

### 重构

- **移除熔断器弹窗标题图标** - 移除熔断器弹窗中的标题图标，简化界面。
- **渠道中心添加目标标签改为产品名称** - 将渠道中心的添加目标标签改为显示产品名称。

### 其他

- **UI 熔断器配置改为预设按钮+滑块交互** - 将熔断器配置改为预设按钮+滑块的交互方式，删除重复的 .env 分组。
- **UI 熔断器参数三列并排布局+间隔线** - 调整熔断器参数为三列并排布局并增加间隔线。
- **UI 预设按钮移到滑块下方，参数列增加内边距** - 将预设按钮移到滑块下方，参数列增加内边距优化间距。

## [v2.8.17] - 2026-05-29

### 新增

- **新增 gpt-5.4-mini 到 codex responses 协议源模型列表** - 前端 codex responses 协议源模型列表新增 gpt-5.4-mini 模型。
- **更新 Gemini 源模型列表** - 更新前端 Gemini 源模型列表配置。
- **更新模型列表并同步前后端** - 更新模型列表并保持前后端配置同步。

### 修复

- **WebUI iframe 认证使用 ADMIN_ACCESS_KEY 而非 PROXY_ACCESS_KEY** - 修复桌面端 WebUI iframe 认证使用错误密钥的问题。

### 其他

- **简化 favicon，替换为极简 CX 描边图标** - 简化 favicon 设计，替换为极简 CX 描边图标。

## [v2.8.16] - 2026-05-29

### 新增

- **添加 DashScope Token Plan 预设** - 桌面端添加 DashScope Token Plan 预设。

### 修复

- **修正渠道一键预设显示** - 修正 UI 渠道一键预设显示问题。
- **复用 Windows 安装器旧安装目录** - 桌面端 Windows 安装器复用旧安装目录。
- **修正 Web Logo 主题配色** - 修正 Web Logo 在不同主题下的配色显示。
- **修正暗色模式 Logo 配色** - 修正桌面端暗色模式下 Logo 配色。
- **修正图标亮暗主题显示** - 修正 UI 图标在亮暗主题下的显示。
- **隐藏桌面 WebUI 版本检测** - 隐藏桌面端内嵌 WebUI 版本检测。

### 其他

- **隔离 backend 端口选择测试避免读取真实 .env** - 隔离桌面端 backend 端口选择测试，避免读取真实 `.env`。

## [v2.8.15] - 2026-05-29

### 新增

- **重设计应用图标与品牌 Logo** - 桌面端应用图标与品牌 Logo 全新设计。

### 修复

- **统一 BuildChannelView 高级开关字段输出** - 将 `normalizeSystemRoleToTopLevel`、`stripEmptyTextBlocks`、`passbackReasoningContent`、`passbackThinkingBlocks`、`injectDummyThoughtSignature`、`stripThoughtSignature` 六个高级开关字段统一收归 `BuildChannelView()` 公共函数输出，消除 messages/chat/responses/gemini 四个 handler 中的重复补丁，确保所有渠道类型的 WebUI 编辑回显与保存一致。
- **移除 PreviewAgentConfigDiff 未使用的 status 变量** - 清理桌面端 PreviewAgentConfigDiff 组件中未使用的 status 变量。
- **Agent 配置应用使用最新 .env 端口** - 桌面端 Agent 配置应用时使用最新的 .env 端口值。
- **统一 Web UI Logo 与应用图标** - 统一 Web UI 中的 Logo 和应用图标，保持视觉一致性。
- **补齐渠道一键模板配置** - 前端补齐渠道一键模板配置功能。
- **调整 Opus 4.8 能力探测请求** - 调整 Claude Opus 4.8 模型的能力探测请求参数。
- **修复预设渠道控制台网址和套餐显示** - 修复预设渠道中控制台网址和套餐信息显示不正确的问题。

## [v2.8.14] - 2026-05-29

### 新增

- **新增 normalizeSystemRoleToTopLevel 开关兼容旧 Claude 上游** - 新增系统角色规范化到顶层字段的兼容开关，便于旧版 Claude 上游适配。
- **统一桌面端与 Web UI 的 MiMo/DeepSeek 渠道预设配置** - 统一桌面端与 Web UI 的 MiMo、DeepSeek 渠道预设，保持跨入口配置一致。
- **补全渠道中心全部预设的中英文 i18n** - 为渠道中心全部预设补齐中英文国际化文案。
- **主题切换改为自动/亮色/暗色三按钮，默认跟随系统** - 桌面端主题切换调整为自动、亮色、暗色三按钮，并默认跟随系统主题。

### 修复

- **修复 Messages 嵌套工具结果图片未触发视觉回退** - 识别 `tool_result.content` 等嵌套 content 数组中的图片块，确保 MiMo 等配置 `visionFallbackModel` 的渠道在历史截图请求中正确切换到识图回退模型，并跳过 `noVision` 渠道。
- **调整主题选项排序** - 调整桌面端主题选项展示顺序，使默认与常用选项更符合使用习惯。
- **修复中文模式下渠道说明显示英文** - 修复中文界面中渠道说明仍显示英文的问题。
- **修复渠道 index 为 0 时能力测试无法启动** - 修复前端能力测试在渠道 index 为 0 时被误判而无法启动的问题。
- **修复桌面端内嵌 Web UI 能力测试无响应的问题** - 修复桌面端内嵌 Web UI 发起能力测试后无响应的问题。
- **修复亮色模式下所有状态色文字对比度不足** - 调整亮色模式状态色文本样式，提升可读性。

### 重构

- **移除浏览器模式的 Wails mock 传输层** - 移除桌面端浏览器模式下的 Wails mock 传输层，简化开发模式通信链路。

## [v2.8.13] - 2026-05-29

### 新增

- **添加 claude-opus-4-8 模型到能力探测与优先级列表** - 前端能力探测与模型优先级排序规则新增 claude-opus-4-8 模型支持。
- **桌面端亮暗色主题切换** - 桌面端新增亮色/暗色主题切换功能，支持用户自定义界面外观。

### 其他

- **桌面端 golang.org/x/sys 提升为直接依赖** - 将 `golang.org/x/sys` 从间接依赖提升为直接依赖，确保依赖关系清晰。

## [v2.8.12] - 2026-05-28

### 修复

- **Linux 桌面构建依赖与运行环境更新** - 更新 CI 中 Linux 桌面构建的系统依赖为 gtk4 与 webkitgtk-6.0，构建运行环境切换至 ubuntu-24.04，确保桌面端在最新 Linux 发行版上正常编译与运行。

## [v2.8.11] - 2026-05-28

### 新增

- **桌面端浏览器直连 Vite dev 端口可走通初始化** - 开发模式下支持浏览器直接连接 Vite dev 端口完成初始化流程（mock Wails IPC），无需通过 Wails 桌面窗口即可调试前端。

### 修复

- **WebUI 快速添加渠道不再误解析配置键** - 修正输入解析，避免 `api_key:`、`url:` 等字段名被识别为 API Key；解析逻辑改为保护 URL 中的冒号，并增加常见配置键名白名单过滤。
- **WebUI 快速添加渠道后新渠道始终置顶** - 修正快速添加流程中渠道定位方式，不再依赖 "index 最大" 的启发式；改为按渠道名称精确匹配新渠道，确保新渠道保持在第一位并触发促销期。
- **Windows 首次启动语言检测按平台分支** - 桌面端分平台检测系统语言，修复 Windows 首次启动时语言显示为英文的问题。
- **Linux AppImage 使用 Ubuntu 22.04 构建以兼容 GLIBC 2.35** - CI 构建环境从新版 Ubuntu 降级到 22.04，确保 AppImage 在较旧 Linux 发行版上正常运行。

### 文档

- **推荐下载卡片增加 gh-proxy 加速下载链接** - 为下载卡片添加 gh-proxy 加速镜像链接，提升国内用户下载体验。
- **站点 hero 区域单行标题与 tagline 扩展** - hero 区域改为单行标题、tagline 扩展上游列表与 env 代码块高亮。

## [v2.8.10] - 2026-05-28

### 新增

- **桌面端多语言切换** - 新增轻量 i18n 框架（zh-CN / en），默认英文并自动跟随系统语言，首次启动按系统语言选择、手动切换后持久化到 `ui-preferences.json`；DesktopService 暴露 `GetLanguagePreference` / `SaveLanguagePreference`；侧栏底部守护面板新增语言切换入口；Sidebar、Setup、Status（含 LogViewer、DiagnosticCard）、Agent（含 ProviderForm、ConfigDiffDialog）、Channel、Env、WebUI 等用户可见文案全部迁移到 `t()`，便于 Microsoft Store Windows 截图审核。
- **桌面端 Agent 配置支持 OpenCode** - configservice 新增 `PlatformOpenCode` 平台、`OpenCodeProxyState`、JSONC provider 块解析与 `auth.json` 读写；默认 ccx provider 走 `http://127.0.0.1:<port>/v1`，可选直连 DeepSeek/MiMo/Compshare/Kimi/GLM/MiniMax/DashScope/OpenCode Zen/Go；写入 `~/.config/opencode/opencode.jsonc` 与 `~/.local/share/opencode/auth.json`；前端 AgentTab/AgentCard/ConfigDiffDialog 渲染 OpenCode 卡片；同步补充中英文 OpenCode 客户端 Desktop 自动配置文档。
- **静默版本检查替代 wails updater 自动弹窗** - 关闭 wails updater 30 分钟自动 `CheckAndInstall`（无差别弹窗，失败场景尤为干扰）；新增 `desktopservice.CheckLatestRelease(force)` 直查 GitHub Releases，过滤 prerelease，服务端缓存 4 小时（错误 30 分钟）；侧栏版本号旁追加「新版 vX.X.X」胶囊，仅有更新时显示，点击打开 release 页（Store 分发不显示）；抽出 `useReleaseCheck` composable（启动 8s 首次检查、之后 4 小时一次）；托盘菜单「检查更新…」入口保留，主动点击仍走完整 wails updater 安装流程。
- **OpenCode 直连仅保留官方原生支持的国产渠道** - OpenCode 卡片下拉移除 MiMo/Compshare/DashScope（OpenCode 官方未原生支持），保留 ccx + DeepSeek/Kimi/GLM/MiniMax + Zen/Go；`useAgentConfig` 同步补齐对应 baseURL 分支。
- **渠道中心 preset 文案 i18n 化并区分 plan 渠道名** - 新增 `translateOrFallback` / `tf` helper，preset/plan/target 文案按 key 翻译、缺失 key 回退到 Go preset 原文；新增 `preset-messages.ts` 仅维护 EN，zh-CN 复用后端中文；channelName 默认值在选中非默认 plan 时追加 plan suffix，避免 MiMo/DashScope 多套餐同名覆盖。

### 修复

- **/v1 等裸 API 前缀不再被 SPA 兜底成 WebUI** - `isAPIPath` 仅匹配带尾部斜杠的前缀（"/v1/" 等），导致 `GET /v1`、`GET /api`、`GET /admin` 这类裸前缀落到 NoRoute 的 SPA 兜底分支返回 `index.html`；改为同时匹配 `path == prefix` 与 `prefix+"/"`，并补上之前漏掉的 `/v1beta`（Gemini 原生格式）；新增表驱动测试覆盖裸前缀、子路径以及 `/v1custom` / `/apifoo` 等非 API 路径。
- **状态页 ServiceDetails 去除与 Sidebar 重叠的 PID/Health 信息** - 移除卡片内 PID 与健康状态行（Sidebar 已展示），同时去掉冗余小标题，避免与左侧栏视觉重叠。
- **统一 Agent 卡片 Provider 下拉文案至 i18n** - Codex 卡片 ccx 选项原使用 `agent.localGateway`（"当前 CCX 网关"），与 Claude/OpenCode 卡片的 `agent.provider.localGateway`（"CCX 本地网关"）不一致；OpenCode 卡片其余选项及"访问官方控制台"按钮、API Key placeholder 也是硬编码中文；统一改用 `agent.provider.*` 与 `agent.openConsole` i18n key。
- **注册 mdi-head-snowflake 图标** - AddChannelModal 的"回传 Thinking Blocks"开关使用 `mdi-head-snowflake`，但 `vuetify.ts` 未导入，运行时图标回退到占位文本；补 `@mdi/js` 导入与 `iconMap` 映射。

### 优化

- **收紧 ServiceDetails 卡片上下内边距** - 桌面端状态页 ServiceDetails 卡片上下内边距收紧，提升信息密度。
- **语言切换合入侧栏底部守护面板** - 桌面端语言切换入口整合到侧栏底部守护面板，与现有偏好控件统一布局。

### 文档

- **新增桌面端教程截图** - 为桌面端使用指南补充压缩后的 CCX Desktop 截图，并将端口引用对齐到默认 3688。

## [v2.8.9] - 2026-05-27

### 修复

- **MiMo 计费目标 URL 不更新** - 修复桌面端 MiMo 计费配置切换后目标 URL 未正确更新的问题。
- **Claude 渠道思考块透传缺失** - 修复 Claude 渠道返回 reasoning/thinking blocks 时未按配置回传的问题。
- **Responses 透传渠道预设误展示 gpt-5.x 一键设置** - Responses 协议透传渠道隐藏不适用的 gpt-5.x 一键预设，避免误配置。
- **导引完成后默认页签错误** - 桌面端首次导引完成后默认跳转 Agent 配置，而非渠道中心。
- **Sidebar 导入路径错误** - 修复桌面端 `Sidebar.vue` 绑定导入路径使用 `@/bindings` 导致的问题，改为正确的 `@bindings`。
- **Responses 到 Claude 缺少 max_tokens 默认值** - 修复 Responses → Claude 链路缺少 `max_tokens` 默认值的问题。(#133)

### 其他

- **补充 max_tokens 边界测试** - 为 Responses → Claude 转换链路补充 `max_tokens` 边界测试，覆盖默认值与异常边界。

## [v2.8.8] - 2026-05-27

### 新增

- **DashScope Coding Plan 选项** - 桌面端 DashScope provider 新增 Coding Plan 计费选项，Claude 直连 Agent 配置支持 Coding Plan 选择器。

### 变更

- **桌面端自更新从手写 updater 改为 Wails v3 内置 Updater** - 删除 `desktop/internal/updater/` 包（含 macOS DMG 挂载、Windows NSIS 安装器、Linux AppImage nohup 替换等平台安装器，行为模式易被杀软误杀）；移除 `CheckUpdate`/`DownloadAndInstall`/`CancelUpdate` Go 导出方法和 `UpdateInfo` 结构体；改用 Wails v3 `pkg/updater`（GitHub Releases Provider + SHA256SUMS 校验 + 内置更新 UI 窗口 + 30 分钟定时检查）；删除前端 `useUpdater.ts` composable 和 `UpdateDialog.vue` 组件，Sidebar 版本按钮改为纯展示（更新检查通过托盘菜单触发）；Wails v3 依赖从 alpha.95 升级到 alpha.96
- **DashScope 按量付费计划标签对齐 MiMo 规范** - 统一 DashScope 渠道预设的按量付费计划标签显示，与 MiMo 规范保持一致。

### 修复

- **Chat 渠道选择 Responses 上游时请求格式与端点错误** - 修复 Chat 渠道 `serviceType: "responses"` 时仍以 Chat 格式发送到 `/v1/chat/completions` 的问题；新增 Chat↔Responses 双向协议转换（请求：messages→input, max_tokens→max_output_tokens；响应：output→choices），流式 SSE 事件转换（response.output_text.delta→Chat chunk），前端预期请求 URL 同步修正为 `/v1/responses`。(#130)
- **规范化 role 开关提示不够明确** - 明确 i18n 中规范化 role 开关的提示文案，注明 developer 角色转换与国内模型建议。

### 文档

- **添加英文 CCX Desktop 指南和入口链接** - 新增英文版 CCX 桌面端使用指南与文档入口链接。

## [v2.8.7] - 2026-05-27

### 文档

- **完善桌面端用户教程与客户端接入入口** - 补充桌面端使用教程与客户端接入文档入口说明。

### 重构

- **MiMo 渠道预设移除自定义 base URL，优化 token plan 切换与下拉标签** - MiMo 渠道预设不再设置自定义 base URL，同时优化 token plan 切换逻辑与下拉标签显示。
- **go fmt 格式化 channelpreset 缩进对齐** - 对桌面端 channelpreset 模块执行 go fmt 统一格式化。

### 修复

- **含图请求不再覆盖普通文本 Trace 亲和** - 含图请求 failover 到视觉渠道成功后，不再改写同一用户的普通文本 Trace 亲和，避免 DeepSeek 等文本渠道被长时间绕过；`HasImageContentCached` 只读缓存，不影响已有 Images 渠道行为。
- **恢复 BuildPayload Description 写入，移除 DMG 打包错误静默** - 恢复桌面端 BuildPayload 中 Description 字段的写入，移除 DMG 打包流程中对错误的静默处理以便排查问题，同时格式化相关测试文件。

### 其他

- **补充 NoVisionModels 断言覆盖** - 为桌面端 NoVisionModels 相关逻辑补充断言覆盖，增强测试完整性。

## [v2.8.6] - 2026-05-26

### 新增

- **Images 渠道内置模型列表新增 gpt-image-2** - 前端 Images 渠道的内置模型选项新增 `gpt-image-2`。
- **渠道中心 provider 官方控制台链接** - 桌面端渠道中心为各 provider 添加官方控制台访问链接，方便快速跳转管理；badges 与链接布局调整至标题同行，避免错位。
- **macOS DMG 安装包自定义拖拽背景** - 新增自定义 DMG 背景图与安装布局，Applications 快捷方式与 .app 拖拽区对齐 FlyShadow 风格。

### 变更

- **Compshare 渠道预设调整** - 高级模型重定向（opus/sonnet/gpt-5.4）从 deepseek-v4-pro 改为 glm-5.1；Chat/Responses 渠道移除内置模型重定向（用户自行在客户端指定模型）；视觉策略从全渠道 NoVision 改为仅 deepseek-v4-flash 关闭视觉，并设置 MiniMax-M2.7 为视觉回退模型。

### 修复

- **添加渠道不写入描述，新渠道插入列表首位** - 修复添加渠道时描述字段未写入的问题，新增渠道现在默认插入列表首位而非末尾。
- **渠道预设视觉与重定向策略重构** - 统一桌面端渠道预设的视觉（NoVision）与模型重定向配置，修正多个预设中不一致的策略。

## [v2.8.5] - 2026-05-26

### 新增

- **桌面端 Store MSIX 分发支持** - Windows 发布流程新增 Store MSIX 分发配置与打包脚本，为 Microsoft Store 分发补齐 manifest、package 脚本和 release 产物链路。
- **桌面端 Compshare 推广入口** - 桌面端新增 Compshare 推广入口与外链打开能力，便于用户从侧边栏访问相关服务。
- **Compshare Agent 直连支持** - 桌面端 Agent 配置新增 Compshare provider 选项，并同步支持直连配置状态识别与应用。
- **Compshare 渠道预设** - 渠道中心新增 Compshare 预设，补充对应模型映射、服务类型和预设测试覆盖。

### 文档

- **隐私政策说明** - 新增中英文隐私政策页面，并将其接入文档导航。
- **首页下载推荐展示优化** - 调整 README 首页下载推荐展示，并补充 VitePress 自适应下载推荐组件。
- **首页自适应下载推荐** - 文档首页新增根据平台自适应展示推荐下载项的能力。

### 其他

- **合并 Compshare 桌面渠道支持** - 合并 Compshare 桌面渠道相关功能分支，纳入本次发布范围。

## [v2.8.4] - 2026-05-25

### 修复

- **桌面端自动更新前停止后端进程** - 自动更新下载并校验安装包后，会先停止由桌面壳托管的 Go 后端，再启动平台安装器；如果当前后端由外部进程托管或停止失败，则中止更新并提示用户手动停止后重试，避免 Windows 安装时后端二进制被占用导致只更新 UI。

## [v2.8.3] - 2026-05-25

### 修复

- **桌面端重启 CCX 服务后误报启动失败** - 网关监控重启服务时，停止与启动阶段改用独立超时上下文，避免停止耗时挤占启动健康检查时间；同时健康检查成功后清理过期错误状态，避免日志显示已重启成功但状态卡片仍展示启动失败。
- **静默 health 轮询访问日志** - health 检查轮询现在遵循静默轮询日志配置，避免高频健康检查污染访问日志。
- **桌面端窗口拖动时背景闪烁** - 桌面端窗口背板改用不透明背景，降低拖动窗口时的透明闪烁感。
- **Logo 默认动画导致界面闪烁** - 关闭 Logo 默认动画，避免页面初始渲染和切换时出现不必要的视觉闪烁。
- **Responses 单渠道模式未追踪对话数据** - 修复 Responses 单渠道成功请求未写入对话追踪器的问题，使驾驶舱能够正确展示单渠道对话数据。

### 其他

- **补充 Responses 单渠道对话追踪回归测试** - 为 Responses 单渠道成功请求新增 handler 级测试，验证请求会写入 `ConversationTracker` 并生成驾驶舱对话记录，防止 `/conversations` 再次为空。

## [v2.8.2] - 2026-05-25

### 新增

- **桌面端最小化后隐藏 macOS Dock 图标** - macOS 下窗口最小化后同步隐藏 Dock 图标，减少后台运行时的 Dock 干扰。
- **桌面端路径项支持打开目录/编辑器操作** - Agent 相关路径展示新增纯图标操作按钮，可直接打开目录或用编辑器打开配置文件。

### 修复

- **桌面端渠道预设模型重定向键统一为 `gpt-5`** - MiMo、MiniMax、OpenCode Zen、OpenCode Go 的 Chat / Codex Responses 单模型重定向现在统一使用 `gpt-5` 作为源模型键，匹配 Codex / Chat 客户端实际请求模型；同时集中整理渠道预设的模型映射、reasoning 映射和 Codex 兼容开关配置。
- **桌面端中文系统默认语言仍为英文** - 环境配置表单在 `.env` 缺少 `APP_UI_LANGUAGE` 时会读取 WebView 暴露的系统/浏览器语言，中文系统默认填入 `zh-CN`，非中文系统继续默认 `en`，且不会覆盖用户已有 `.env` 设置。
- **桌面端 MiniMax 渠道预设与 Codex 兼容配置不完整** - MiniMax Responses 预设现在使用 `gpt-5 -> MiniMax-M2.7` 模型映射，并启用非标准 Chat 角色归一化；同步补充渠道预设测试对归一化字段的断言。
- **桌面端 Codex 应用 CCX 渠道时 auth 密钥不一致** - Codex Agent 配置现在优先读取桌面 dataDir、项目根目录和 `backend-go` 下 `.env` 中的 `PROXY_ACCESS_KEY`，再回退到进程环境变量或自动生成值，避免 `~/.codex/auth.json` 写入与实际后端配置不一致的密钥；同时增强 `.env` 解析以兼容 `export PROXY_ACCESS_KEY=...` 和等号周围空格。
- **能力测试复制到其他 Tab 时原生协议类型错误** - 当能力测试中某个原生协议测试成功后复制渠道到其他协议 Tab 时，新增渠道的 `serviceType` 现在按成功协议映射为对应原生上游类型（Claude Messages / OpenAI Chat / Gemini / Responses），不再沿用源渠道的协议类型；转换按钮同时保留目标 Tab 与成功协议两个语义，避免复制到目标 Tab 时写入错误上游类型。
- **桌面端初始配置完成状态误读环境变量** - `IsSetupComplete` 现在只读取桌面 dataDir 下的 `.env`，避免进程环境变量或项目根目录 `.env` 让首次启动引导被错误跳过，并补充对应单元测试。
- **桌面端 Agent 状态路径展示溢出** - Agent 卡片中的当前/目标 URL、配置文件和认证文件路径改为固定标签列 + 可换行内容列，保留编辑器打开按钮且避免长路径挤压布局。
- **桌面端网关监控详情字段未对齐** - 网关监控详情展示字段与后端服务状态字段保持一致，避免状态信息展示不完整或错位。
- **桌面端托盘隐藏 Dock 图标时崩溃** - 修复托盘状态下触发隐藏 Dock 图标可能导致桌面端崩溃的问题。
- **桌面端渠道预设刷新按钮点击报错** - 移除无效的刷新预设按钮，避免点击后触发前端异常。
- **桌面端页面内容区宽度受限** - 移除页面内容区 `max-w-5xl` 限制，使桌面端页面能够自适应占满可用空间。

### 文档

- **补充渠道预设模型重定向说明** - 在变更记录中补充渠道预设模型重定向的说明，明确 Codex / Chat 客户端实际请求模型与预设映射关系。
- **调整赞助商展示样式** - 优化 README 中赞助商展示区域的排版与视觉呈现。
- **添加赞助商注册链接** - 在 README 赞助商信息中补充可直接访问的注册链接。
- **添加优云智算赞助商信息** - README 新增优云智算赞助商介绍与展示图片。

### 其他

- **version-bump 增强 CHANGELOG 完整性校验** - 版本升级技能现在会校验从上一版本 tag 到 HEAD 的提交记录，补全或移动遗漏的 CHANGELOG 条目。
- **桌面端 Wails 绑定产物切换为 TypeScript** - 当前工作区包含 `desktop/frontend/bindings/` 下生成绑定从 `.js` 文件删除到 `.ts` 文件新增的变更，需随本轮未提交改动一起审核。

## [v2.8.1] - 2026-05-25

### 修复

- **桌面 App 内嵌 Web UI 删除渠道不生效** (#115) - Wails 3 的 WKWebView 在 iframe 上下文中对原生 `window.confirm()` 不弹对话框、静默返回 `false`，导致删除请求从未发出；替换为 Vuetify `v-dialog` 实现的通用确认框，兼容桌面 iframe 与浏览器双入口
- **LogViewer autoScroll 开关失效，新日志仍强制滚动** - 移除对 `filteredLogs` 的冗余 watch，改为监听 `searchQuery` 仅在用户改变搜索关键字时强制滚动；新日志增加时改走受 `autoScroll` 控制的 `props.logs.length` watch
- **缩短引导界面生成的 PROXY_ACCESS_KEY 长度** - 随机熵从 24 字节降为 8 字节，密钥从 `ccx-` + 48 字符 hex（52 字符）改为 `ccx-` + 16 字符 hex（20 字符）；64 位熵对本地代理网关已足够安全，显著改善初始引导界面的可读性与可复制性

### 文档

- **修正渠道名称说明，反映同名覆盖的实际行为** - 后端 `createChannel` 在检测到同名渠道时会执行 PUT 覆盖更新，原文案 "如重复，CCX 会拒绝创建" 与实际行为不符，容易造成误导

## [v2.8.0] - 2026-05-25

### 修复

- **TestDetectRootDirFallsBackToCwd 在 macOS `/private/var` 符号链接下的误报** - macOS 的 `/var` 是 `/private/var` 的符号链接，`t.TempDir()` 返回前者但 `os.Getwd()` 解析到后者，导致字符串直接比较失败；改用 `filepath.EvalSymlinks` 对齐期望值

### 其他

- **同步 channelpreset 测试期望与当前 preset 实现** - 修复 9 个因测试期望过时导致的失败用例，实现本身正确
  - 移除对 `SupportedModels` 字段的断言：`preset.go` 从未设置该字段（deepseek_chat 与 mimo 共 4 个用例）
  - `glm_chat` / `glm_responses` BaseURL 改为 coding plan 的实际值（`https://open.bigmodel.cn/api/coding/paas/v4#`），匹配 `bestPlanForTarget` 按 plan 数组顺序选第一个非 anthropic plan 的行为
  - `minimax_responses` 的 `CodexToolCompat` / `StripCodexClientTools` 改为 false：`applyTargetDefaults` 显式 override 为 native passthrough 策略，与 DeepSeek 一致
  - `TestBuildPayloadRejectsUnsupportedTarget` 改用 `invalid-target`：kimi preset 已支持 `NativeMessages`，原 kimi+messages 用例失效

## [v2.7.31] - 2026-05-25

### 破坏性变更

- **桌面端 dataDir 路径稳定化，移除 sha1 hash 子目录** - 历史版本将 dataDir 计算为 `{UserConfigDir}/ccx-desktop/{sha1(rootDir)[:10]}`，rootDir 由 `os.Getwd()` 决定，导致从 Dock、Spotlight、终端 `wails3 dev` 等不同方式启动时产生不同 hash 目录（实测一台机器下出现 3 个互不相通的目录），用户的 PROXY_ACCESS_KEY、渠道配置、Agent 配置快照在不同启动方式之间相互不可见
  - 新版本统一使用 `{UserConfigDir}/ccx-desktop/`，与 `bootstrap.log` 同级，dev 与 prod 共用
  - **升级影响**：从旧版本升级后，原 hash 子目录（如 `ccx-desktop/42099b4af0/`）不会自动迁移，用户首次启动会进入空配置状态，需要重新通过引导页设置或手动从旧 hash 目录复制 `.env`、`agent-config-state/`、`.config/` 到新位置
  - 不做自动迁移的原因：无法可靠判断哪个 hash 目录代表"主用户数据"

## [v2.7.30] - 2026-05-25

### 新增

- **桌面端首次启动引导流程** - 强制要求用户在启动 CCX 服务前完成最小可用配置，避免直接进入主界面后调用失败
  - 新增 SetupView 全屏引导页：检测 `.env` 中无 `PROXY_ACCESS_KEY` 时显示，预填随机生成的密钥（`ccx-<48hex>`），支持显示/隐藏、复制、重新生成
  - 完成后自动合并写入 `.env`（保留已有键值与注释），自动启动 CCX 服务并跳转至【渠道中心】标签页
  - 引导页展示 `.env` 文件路径方便后续调试，提示后续可在【环境参数】页继续修改其他字段
  - 新增 `IsSetupComplete` / `GenerateProxyAccessKey` 两个 Wails 绑定方法，前者只读判断密钥存在性，后者仅生成预览不写入文件
  - App 启动时三态渲染（Loading 占位 / Setup 引导页 / 主界面），避免界面切换闪烁

- **桌面端 bootstrap 启动日志** - 在 backend manager 初始化前写入固定位置的 `bootstrap.log`，覆盖 dataDir 计算阶段的启动诊断盲区
  - 新增 `mustGetwd` / `mustExecutable` 辅助函数，记录关键启动阶段
  - 便于 Windows 双击无反应、dataDir 解析异常等场景的故障排查

### 修复

- **桌面端版本号显示重复 `vv` 前缀** - 修复桌面端左下角及更新弹窗版本号显示为 `vv2.7.29` 的问题
  - VERSION 文件已自带 `v` 前缀，前端模板不应再硬编码 `v`
  - 修正 `Sidebar.vue` 左下角版本徽标，以及 `UpdateDialog.vue` 标题处的当前/最新版本对比

- **detectRootDir 向上遍历边界条件死循环** - 修复 `filepath.Dir` 到达根目录后返回自身导致的潜在死循环
  - 改为显式检测 `parent == dir` 终止
  - 新增测试验证 fallback 到 cwd 的行为

### 其他

- **NSIS 构建添加 `/INPUTCHARSET UTF8`** - 修复 NSIS 在非 ASCII 路径下编译失败的问题，并将 `ccx-go.exe` 构建产物加入 `.gitignore`
- **新增 `make install` 命令** - 一键安装根目录、`backend-go/`、`frontend/`、`desktop/`、`desktop/frontend/` 全部依赖

## [v2.7.29] - 2026-05-25

### 修复

- **批量延迟测试在 Chat 标签页报错 `e.forEach is not a function`** - 修复 Chat 渠道批量延迟测试时，前端 `pingAllChatChannels()` 未解包后端 `{"channels": [...]}` 响应导致 `forEach` 调用失败的问题
  - `api.ts` 中 `pingAllChatChannels()` 现在正确提取 `resp.channels` 并映射字段，与 Images/Gemini 处理方式一致

- **第三方 provider 直连使用 ANTHROPIC_AUTH_TOKEN** - 修复桌面端第三方 provider（DeepSeek、MiMo、Kimi、GLM、MiniMax、DashScope、OpenCode）直连 Claude Code 时，API Key 错误写入 `ANTHROPIC_API_KEY` 的问题，改为正确写入 `ANTHROPIC_AUTH_TOKEN`
  - `resolveClaudeProvider` 统一将第三方 provider 的 key 归入 `authToken` 返回值，与 CCX provider 行为一致
  - apply、restore、preview 三条路径均通过同一函数解析，无需额外修改

- **配置预览 Diff 掩码后丢失敏感字段变更** - 修复 Agent 配置预览弹窗中，敏感字段（如 `ANTHROPIC_AUTH_TOKEN`）变更时因掩码为 `***` 导致 diff 无法识别变更的问题
  - 新增 `computeJSONDiffWithMask` / `computeTextDiffWithMask`：先在原始内容上做 LCS diff 正确识别变更类型，再用掩码后的内容填充展示行
  - 更新全部 6 个 preview 函数（Claude/Codex apply & restore）使用先 diff 再掩码的方式
  - 新增 `extractNestedStringValues` 支持从嵌套 map（如 `env` 子 map）提取敏感字段值
  - 新增 4 个单元测试覆盖掩码 diff 核心场景

## [v2.7.28] - 2026-05-24

### 修复

- **Codex 第三方 provider 状态识别与恢复逻辑修复** - 修复桌面端 Codex 直连模式下第三方 provider 的运行状态识别和恢复逻辑
- **Codex Responses 转 Chat 时空 tool_choice 导致上游 400** - 修复当 Codex 请求仅携带非 function 类型 tools（全被过滤）时，`tool_choice` 和 `parallel_tool_calls` 仍被透传给上游 Chat API 导致拒绝的问题
  - `OpenAIChatConverter` 和 `ConvertResponsesToOpenAIChatRequest` 两条路径均增加 tools 存在性检查
  - 无 tools 时自动剥离 `tool_choice` 和 `parallel_tool_calls`，与透传路径 `stripCodexClientOnlyTools` 行为一致

- **WebUI iframe 切换标签时自动刷新** - 修复桌面端切换标签页后 WebUI iframe 白屏的问题，切回 WebUI 标签时自动刷新内容

- **Windows 桌面端双击无反应** - 修复 Windows 生产构建（`-H windowsgui`）下启动失败时错误静默吞掉的问题
  - 添加 Windows MessageBox 弹窗：WebView2 缺失、ccx-go 找不到等致命错误现在会弹窗提示用户
  - 添加顶层 `recover()` 兜底：未处理的 panic 不再导致进程静默退出
  - 添加单实例互斥锁：重复双击时弹窗提示"已在运行中"，检查系统托盘
  - 添加持久化文件日志：`%APPDATA%\ccx-desktop\<hash>\desktop.log` 记录启动日志便于排查
  - NSIS 安装器 `ccx-go.exe` 从可选改为必需，缺失则安装失败

## [v2.7.27] - 2026-05-24

### 新增

- **Codex 直连新增 DashScope / OpenCode Zen / Go 选项** - 桌面端 Codex 直连配置新增 DashScope、OpenCode Zen 和 OpenCode Go 三个新的 provider 选项
- **DashScope chat/responses 预设扩展** - DashScope 渠道预设增加 GPT 模型重定向和 reasoning mapping 支持

### 修复

- **Codex 直连目标 URL 显示** - 修复 Codex 直连时目标 URL 未正确显示各 provider 的 Responses 端点的问题

## [v2.7.26] - 2026-05-24

### 新增

- **多渠道 Messages 原生透传支持** - Kimi、GLM、MiniMax 新增 Messages 原生透传，DeepSeek codex responses 预设默认开启原生工具透传和 role 规范化
- **渠道预设扩展** - 新增 OpenCode Zen/Go 和阿里云 DashScope 渠道预设与 Agent 直连支持
- **Agent 直连选项** - Agent 配置新增 Kimi / GLM / MiniMax 直连选项
- **渠道中心自动覆盖** - 添加到 CCX 时自动覆盖同名渠道配置
- **Web UI 刷新按钮** - 内嵌 Web UI 页面添加刷新按钮
- **Codex OpenAI 切换** - 支持 Codex 无缝切换回 OpenAI provider

### 修复

- **渠道中心文案与 URL** - 修正描述文案错别字，移除 /v1 结尾 base URL 的 # 后缀
- **渠道中心 target 切换** - 修复切换 target 后被级联重置回 messages 的问题
- **桌面端二进制路径** - 修复 ccx-go 后端二进制未打包到 App Bundle 的路径错误
- **vite 插件路径** - 修复 vite 插件路径为绝对路径并更新前端依赖
- **OpenAI provider 配置** - 修复 OpenAI 模式下目标 URL 显示为空，移除无效配置块写入
- **监控面板初始化** - 初始化时预探测二进制路径，避免监控面板显示"未发现"

### 重构

- **Web UI 刷新按钮位置** - 刷新按钮移至标题栏，WebUITab 通过 defineExpose 暴露刷新方法
- **渠道预设精简** - 移除渠道预设中所有 SupportedModels 显式赋值，MiniMax 预设精简默认值
- **ccx-go 查找路径** - 改为向上遍历最多 6 层目录

### 其他

- **Wails 升级** - Wails v3.0.0-alpha.92 升级到 alpha.95

## [v2.7.25] - 2026-05-23

### 修复

- **渠道预设模型映射更新** - 更新桌面端渠道预设的模型映射配置
- **WebUI 默认语言改为中文** - 前端 WebUI 默认语言从英文改为中文
- **Agent 代理配置应用按钮文案统一** - 统一桌面端 Agent 代理配置的应用按钮文案
- **Checksum 文件路径前缀修正** - 去除 CI Release 流程中 checksum 文件的路径前缀

### 其他

- **桌面应用图标资源更新** - 更新桌面应用图标资源文件

## [v2.7.24] - 2026-05-23

### 修复

- **Release workflow Bad credentials 错误** - 修复 CI Release workflow 中 GitHub token 认证失败的问题

## [v2.7.23] - 2026-05-23

### 修复

- **MiMo 不同计费模式支持独立 API Key** - ProviderKeyAsset 的 Assets map 改为 `provider:planID` 复合 key，同一 provider 不同 plan 不再互相覆盖；前端查找保存的 key 时优先匹配 plan-specific key，回退到 provider-only key 兼容旧数据

## [v2.7.22] - 2026-05-23

### 修复

- **桌面端 CCX Provider 应用按钮状态修复** - CCX provider 的应用按钮不再依赖网关在线状态，避免网关未启动时无法操作
- **Release checksum 路径修复** - 修复 Release checksum 生成路径错误导致三个主平台构建失败的问题

## [v2.7.21] - 2026-05-23

### 修复

- **Release checksum 路径前缀修正** - 修复 CI release 流程中 checksum 文件路径前缀错误并精简 finalize job
- **桌面构建 ccx-go 查找路径** - 修正桌面端构建时 ccx-go 后端二进制的查找路径
- **渠道中心 Base URL 过滤** - 桌面端渠道中心按 target 协议过滤 Base URL 选项，避免显示不匹配的选项

### 其他

- **恢复 Wails 3 TypeScript 绑定文件** - 恢复桌面端 Wails 3 自动生成的 TypeScript 绑定文件
- **version-bump 技能文档更新** - 更新 version-bump 技能的 GitHub Actions 集成说明，新增 `gh run list` 查看构建进度命令

## [v2.7.20] - 2026-05-23

### 修复

- **MiMo 渠道模板 Base URL 修正** - 修正 MiMo 渠道模板 Base URL 为 Anthropic 协议入口，同时调整计费模式表单顺序
- **网关监控默认显示配置端口** - 网关监控打开时默认显示 .env 中配置的端口，启动后端时优先使用用户配置的端口
- **Windows Desktop amd64 安装包缺失** - 修复 CI 中 Windows Desktop amd64 安装包在 Release 中缺失的问题
- **macOS 应用图标重新设计** - 重新设计应用图标以适配 macOS Human Interface Guidelines
- **macOS 托盘图标交互修复** - 修复 macOS 托盘图标点击后窗口置顶和位置偏移的问题

## [v2.7.19] - 2026-05-23

### 修复

- **日志文件混入精简格式重复内容** - 修复 logger 将 stdout 精简格式与原始完整 JSON 双重写入日志文件的问题，确保日志文件仅保留原始完整记录

## [v2.7.18] - 2026-05-23

### 修复

- **DMG 安装体验与图标显示** - 改善 macOS DMG 安装包的安装体验与应用图标显示效果

## [v2.7.17] - 2026-05-23

### 修复

- **Task 依赖链 ARCH 传递遗漏** - 修复 CI 中 Task 依赖链未正确传递 ARCH 环境变量的问题

## [v2.7.16] - 2026-05-23

### 修复

- **桌面构建 ARCH 变量未通过 Task 依赖链传递** - 修复 CI 中桌面端构建时 ARCH 环境变量未正确传递到下游 Task 步骤的问题

### 其他

- **版本升级技能新增本地编译验证** - version-bump 技能在创建 tag 前强制执行本地测试和构建验证，编译失败时中止发布流程

## [v2.7.15] - 2026-05-23

### 变更

- **移除后端 OTA 更新机制** - 删除后端 updater 包及相关 API 接口，桌面端更新改为跳转到 GitHub Releases 下载页面

## [v2.7.14] - 2026-05-23

### 新增

- **桌面端 .env 密码字段复制按钮** - .env 配置中密码类型字段（如 `PROXY_ACCESS_KEY`）添加复制按钮，方便用户快速复制密钥值

### 修复

- **密码字段留空时隐藏复制按钮** - 当密码类型字段值为空时自动隐藏复制按钮，避免无效操作

### 变更

- **移除后端 OTA 自动更新** - 删除 `updater` 包、`/api/update/check` 接口及相关配置，桌面端更新改为跳转到 GitHub Releases 下载页面

### 其他

- **Release workflow 桌面构建前置修复** - CI 桌面端构建步骤前确保 `bin` 目录存在，修复构建流程中目录缺失问题

## [v2.7.13] - 2026-05-23

### 新增
- **Linux ARM64 桌面端支持**：CI 新增 `ubuntu-24.04-arm` runner 构建 arm64 AppImage，updater 移除 arm64 限制。
- **前端测试框架**：引入 vitest + jsdom，新增 `env-file.ts` 20 个单元测试。
- **Bindings 同步 CI 守护**：新增 `check-bindings.yml` workflow，PR 修改 Go 接口时自动验证 TypeScript bindings 同步。
- **桌面端 .env 文件外部编辑器打开**：Env 配置页新增"用编辑器打开"按钮，自动检测系统已安装的编辑器（VS Code、Cursor、Sublime Text、Vim 等），支持 `$EDITOR` / `$VISUAL` 环境变量优先。多个编辑器时提供下拉选择。

- **Desktop 内置 ccx-go 后端**：macOS .app、Windows NSIS、Linux AppImage/deb/rpm 安装包现在内置 ccx-go 后端二进制，用户无需单独下载。Desktop 与 ccx-go 共用版本号。
- **Release Sigstore 签名**：
  - 在 CI 发布流程中集成 Sigstore / cosign keyless signing，使用 GitHub Actions OIDC 颁发短期证书，无需管理密钥。
  - 三个构建平台（macOS / Windows / Linux）各自生成平台级 checksums 并签名，`finalize` job 合并为全平台 `checksums.txt` 后再次签名。
  - 新增 `docs/guide/verification.md` 与 `docs/en/guide/verification.md` 验证文档，说明 cosign 安装与签名验证步骤。
  - 现有 `.sha256` sidecar 文件与 updater 行为不变。

### 变更

- **桌面端 Env 配置访问控制置顶**：`PROXY_ACCESS_KEY` 和 `ADMIN_ACCESS_KEY` 配置组在 Env 配置表单中移到首位，便于首次配置时优先设置。
- **补齐桌面端测试覆盖**：新增 `configservice`（19 个纯函数 + 集成测试）和 `backend/manager`（16 个测试）测试文件。

### 修复
- **MSIX 打包缺少 wails.json**：新建 `desktop/wails.json`，修复 `wails3 tool msix --config` 引用不存在的文件。
- **parseEnvContent 兼容性**：Go 端 `.env` 解析支持 `export` 前缀和 `\r\n` 换行符。
- **桌面端 Agent 配置服务初始化错误日志**：`configservice.New()` 失败时不再静默丢弃错误，改为输出 `[Desktop-Init]` 日志便于排查。
- **渠道对话框下拉菜单位置偏移**：修复 Vuetify v-select/v-combobox 在添加/编辑渠道对话框内首次打开时菜单位置计算错误的问题，通过 eager 预渲染与 resize 触发确保下拉菜单正确定位。
- **DeepSeek `user_id` 限速与隔离字段透传**：
  - 让内部会话标识提取支持 Chat 请求体顶层 `user_id`，用于 Trace 亲和性与调度追踪。
  - 补齐 Chat → Claude、Messages → OpenAI/Responses、Responses → Claude 跨协议转换中的 `user_id` / `metadata.user_id` 映射，避免用户隔离标识在协议转换时丢失。
  - 增加相关单元测试覆盖关键转换路径。
- **Responses 转 Claude 上游 reasoning 回传补齐**：
  - 在 Responses 转 Claude/Messages 风格请求且启用 `PassbackReasoningContent` 时，同步补齐 `reasoning_content`，避免 MiMo 等上游跨渠道历史请求缺少 reasoning 回传字段。

## [v2.7.12] - 2026-05-22

### 修复

- **Windows Release 构建 NSIS makensis 不在 PATH** - 修复 CI 中 Windows Release 构建时 NSIS `makensis` 工具不在 PATH 导致构建失败的问题

## [v2.7.11] - 2026-05-22

### 新增

- **渠道预设 Token Plan 下拉显示实际 Base URL** - 桌面端渠道预设 Token Plan 下拉选项现在会显示实际的 Base URL，便于用户识别和选择

## [v2.7.10] - 2026-05-21

### 修复

- **MiMo Messages 渠道流式响应只输出 thinking 的问题**：
  - 停止为缺少真实思考内容的历史 assistant 消息注入 `"(no prior reasoning recorded)"` 占位 `thinking` 或 `reasoning_content`，避免 MiMo 将正式回答续写进假 thinking block，或继续因假回传返回 `reasoning_content ... must be passed back` 400。
  - 为历史 assistant 消息补齐 MiMo 要求的顶层 `reasoning_content` 字段：保留真实 `thinking` 块原文并同步回传到顶层字段，仅移除历史请求中旧版本注入的占位/空 `thinking` 块；若 assistant 历史没有真实思考内容，则补空字符串 `reasoning_content: ""`（保持顶层字段存在）；若历史 assistant 仅剩占位 thinking，则转为中性非空 text，避免空 content 或轮次丢失触发上游校验问题。
  - `thinking -> reasoning_content` 转换改为“保真搬运”：不再对真实思考文本做 trim/拼接改写；当消息已含顶层 `reasoning_content` 时保持原值，避免内容被改写后触发上游回传一致性校验失败。
  - 修复请求预处理误删真实 thinking 的问题：`thinking` 块里 `signature` 为空/null 时仅移除 `signature` 字段，不再删除整块 `thinking`，避免后续 `reasoning_content` 回传被掏空。
  - 增强 failover 非重试判定：当上游 400 的 `error.param`/`error.message` 命中 `reasoning_content in the thinking mode must be passed back` 时，判定为 schema 参数错误，不再继续 Key/渠道级 failover，避免请求漂移到其他渠道与错误熔断。
  - 流式预检测改为只有正式 text 或工具语义内容才视为有效响应；thinking-only 流会被判定为空响应并触发重试/失败，避免静默返回 200。

## [v2.7.9] - 2026-05-20

### 修复

- **Gemini 渠道流式响应下 tool call ID 生成策略优化**：
  - 在 `backend-go/internal/providers/gemini.go` 中，将 Gemini 流式响应下 functionCall 的 tool ID 生成策略从不稳定的索引拼接形式（如 `toolu_%d`）优化为稳定的、带有 `call_` 前缀的 16 字节随机 hex 字符串（通过 `crypto/rand` 生成），使 ID 在跨多轮对话时更具唯一性且对齐主流（如 OpenAI `call_xxx` 风格），避免在复杂 agent 多轮调用中发生冲突。
  - 在 `backend-go/internal/providers/gemini_stream_test.go` 中添加对应的测试断言，验证 tool call ID 的 `call_` 前缀生成逻辑。
- **修复 Gemini 流式响应中 functionCall 场景的 `stop_reason` 映射**：
  - 在 `backend-go/internal/providers/gemini.go` 中，引入 `hasToolUse` 状态标志。当流式响应中检测到 functionCall 时，将 `message_delta` 的 `stop_reason` 正确映射为 `tool_use`（而不是默认的 `end_turn`），确保下游消费端能够正确感知并启动工具调用执行，完全符合 Claude 规范。
  - 在 `backend-go/internal/providers/gemini_stream_test.go` 中新增 `TestGeminiHandleStreamResponse_FunctionCallMapsStopReasonToToolUse` 单元测试，模拟 functionCall 流式响应并验证 `stop_reason` 映射。

## [v2.7.8] - 2026-05-20

### 重构

- **清理 `providers` 包历史 lint 告警**：
  - `backend-go/internal/providers/openai.go` 中 `ConvertToClaudeResponse` 转换 tool_call 入参时，给 `json.Unmarshal` 补上错误检查；解析失败时降级保留 `toolCall.Function.Arguments` 原始字符串，避免静默丢失参数（errcheck）。
  - `backend-go/internal/providers/openai_stream_usage_test.go` 中 `TestOpenAIProvider_ConvertToClaudeResponse_CacheFieldInJSON` 给 `json.Unmarshal` 补上 `t.Fatalf` 错误检查，防止 unmarshal 失败时后续断言基于空 map 假阳性通过（errcheck）。
  - `backend-go/internal/providers/responses.go` 删除未被任何地方调用的 `formatFunctionCallHistory` 函数（13 行）：与配套 `formatFunctionCallOutputHistory` 不对称——`function_call` 分支始终直接保留原始 item，不需要降级为文本消息，该函数是早期设计残留（unused）。
  - `backend-go/internal/providers/claude.go` 中 `ConvertToProviderRequest` 模型重定向条件 `upstream.ModelMapping != nil && len(upstream.ModelMapping) > 0` 简化为 `len(upstream.ModelMapping) > 0`（Go 对 nil map 的 `len()` 定义为 0，gosimple/S1009）。

## [v2.7.7] - 2026-05-20

### 修复

- **`messages` 走 Gemini 上游时 tool_result 函数名错位导致工具调用沉默丢失**：
  - 修复 `backend-go/internal/providers/gemini.go` 的 `convertMessage` 在转换 Claude `tool_result` 为 Gemini `functionResponse` 时，把 `name` 字段直接填成 `tool_use_id` 的问题。Gemini 协议要求 `functionResponse.name` 必须等于前面对应 `functionCall.name`（函数名），否则上游无法匹配到对应的工具调用，会沉默返回空内容（典型表现：MCP / 多轮工具历史在 Gemini 上游被静默丢弃，下游模型回合显示空白）。
  - 新增 `buildToolUseIDNameMap`，在 `convertMessages` 入口先扫一遍 Claude 历史，构建 `tool_use_id → name` 映射；`convertMessage` 接收该映射并在 tool_result 转换时回查正确函数名。查不到映射的孤立 tool_result 回退使用 `tool_use_id` 兜底，避免完全丢字段。
  - 在 `backend-go/internal/providers/gemini_tool_result_test.go` 中将历史 fixture 升级为含 tool_use 的完整对话，并新增针对 id→name 映射回查的断言用例。

### 新增

- **Chat 渠道透传思考回传支持**：
  - 在 `backend-go/internal/handlers/chat/handler.go` 中引入了与 Messages 渠道一致的预处理逻辑，自动清理空 signature 字段和历史畸形 thinking 内容块，预防上游参数校验 400 错误。
  - 在 `buildProviderRequest` 中，当 `PassbackReasoningContent` 关闭时，在发送给 Claude 协议上游前自动剥离历史 thinking 块，避免跨上游复用签名导致 signature 错误。
  - 在 `backend-go/internal/config/config_chat.go`、`channels.go` 和 `channel_metrics_handler.go` 中同步支持了 `PassbackReasoningContent` 字段的更新与返回。
  - 修改了 `frontend/src/components/AddChannelModal.vue`，使得“回传 Reasoning Content”开关在 Chat 渠道且服务类型为 claude 时也能正常显示和配置。
  - 在 `deepseek_thinking_matrix_test.go` 中新增了 `TestChatHandler_PassbackReasoningContent` 测试用例。

### 修复

- **`messages` 走 Gemini 上游时 MCP / 自定义工具历史丢失导致重复调用**：
  - 修复 `backend-go/internal/converters/responses_to_gemini.go` 中 `responsesItemToGeminiContents` 漏掉 `custom_tool_call` 与 `custom_tool_call_output` 两类 item 的问题。归一化阶段会把 MCP 工具（如 `mcp__serena__*`）落到这两种类型，原实现走 switch 的隐式 default 直接丢弃，导致历史里只剩模型空回合，Gemini 视为"工具调用未返回"并不停重复发起同一工具调用。
  - 现在两类 item 分别转为 `model` 角色的 `FunctionCall`（携带 `DummyThoughtSignature`，与 `function_call` 行为一致）和 `user` 角色的 `FunctionResponse`，并对缺 `call_id`/`name` 的孤立项做兜底丢弃，避免构造出 Name 为空的 part 触发上游 400。
  - 在 `backend-go/internal/converters/gemini_responses_roundtrip_test.go` 新增 `TestResponsesToGeminiRequest_PreservesCustomToolCallHistory` 和 `TestResponsesToGeminiRequest_CustomToolCallOutputWithoutCallIDDropped` 两条回归用例。
- **SUBSCRIPTION_NOT_FOUND 余额不足故障转移**：
  - 将 `SUBSCRIPTION_NOT_FOUND` 错误码和 `"no active subscription found"` 错误信息识别为余额不足（`insufficient_balance`），从而正确触发渠道黑名单和故障转移。
  - 在 `failover_test.go` 和 `stream_test.go` 中补充了对应的普通请求与流式请求测试用例。
- **Gemini 渠道 tool_result 数组解析报错**：
  - 修复了 `messages` 接口转上游 `gemini` 类型渠道时，`tool_result` 包含数组（Content Blocks）导致的反序列化报错问题。
  - 将 `GeminiFunctionResponse.Response` 字段类型从 `map[string]interface{}` 变更为 `interface{}`，提高结构体容错性。
  - 在 `backend-go/internal/providers/gemini.go` 的 `convertMessage` 方法中，对 `tool_result` 的 `content` 进行了智能解析和规范化，确保转换后的 `response` 始终是一个符合 Gemini 官方协议要求的 JSON 对象。
  - 新增了 `gemini_tool_result_test.go` 单元测试，覆盖了 `tool_result` 为数组、字符串、JSON 对象等各种场景，验证其能正确转换为 Gemini 期望的格式。
- **Gemini 渠道空 text part 触发上游 400**：
  - 修复了 `messages` 接口转上游 `gemini` 类型渠道时，Claude assistant 消息中常出现的空 text 块（如带 tool_use 时的前置 padding）被无脑翻译为 `{"text": ""}`，被严格按 Gemini protobuf 校验的上游（如 vip.undyingapi.com）判定为 `contents[X].parts[Y].data: required oneof field 'data' must have one initialized field` 返回 400 的问题。
  - 在 `backend-go/internal/providers/gemini.go` 的 `convertMessage` 中处理 `text` 类型 content 时跳过空字符串，确保不向 Gemini 上游下发无意义的空 Part。
  - 在 `gemini_tool_result_test.go` 中新增 `TestGeminiProvider_ConvertMessage_SkipsEmptyTextBlock` 与 `TestGeminiProvider_ConvertMessage_KeepsNonEmptyTextBlock` 两条用例，覆盖空 text 块被剔除与非空 text 块仍保留两种场景。
- **Claude→Gemini provider 转换遵循 thought_signature 渠道开关**：
  - 修复了 `messages` 接口转上游 `gemini` 类型渠道时，Claude 协议本身不携带 thought_signature 字段，导致严格校验的上游（如 vip.undyingapi.com）返回 `Function call is missing a thought_signature in functionCall parts` 400 的问题。
  - 在 `backend-go/internal/providers/gemini.go` 的 `convertToGeminiRequest` 中按上游 `injectDummyThoughtSignature` / `stripThoughtSignature` 开关注入 part 层级的 `thoughtSignature`，与原生 Gemini 入口（`handlers/gemini/handler.go`）的策略对齐：默认不修改、`injectDummyThoughtSignature` 开启时注入 `DummyThoughtSignature`、`stripThoughtSignature` 优先级更高且在该场景为 no-op（Claude 协议本就无签名）。
  - 在 `gemini_tool_result_test.go` 中新增 `TestGeminiProvider_ConvertToGeminiRequest_InjectDummyThoughtSignature`、`TestGeminiProvider_ConvertToGeminiRequest_DefaultNoSignature`、`TestGeminiProvider_ConvertToGeminiRequest_StripThoughtSignatureNoOp` 三条用例。
- **补齐多渠道 thought_signature 字段链路**：
  - `backend-go/internal/config/config_messages.go` 的 `UpdateUpstream` 补充接收 `StripThoughtSignature`（之前漏了，开启了等于无效）。
  - `backend-go/internal/config/config_chat.go`、`config_responses.go` 的 update 函数补充接收 `InjectDummyThoughtSignature` 和 `StripThoughtSignature`，对齐 [[feedback_channel_config_field]] 的"五类渠道更新函数必须同步应用新字段"规则。
  - `backend-go/internal/handlers/messages/channels.go`、`chat/channels.go`、`responses/channels.go` 的 `GetUpstreams` 把这两个字段透出给前端；`handlers/channel_metrics_handler.go` 的 dashboard 端点同步补充。
  - `frontend/src/components/AddChannelModal.vue` 把 `injectDummyThoughtSignature` 开关的显示条件由仅 `props.channelType === 'gemini'` 改为 `(gemini || messages) && form.serviceType === 'gemini'`；`stripThoughtSignature` 开关的显示条件由仅 gemini 改为 `form.serviceType === 'gemini'` 且 channelType 属于 gemini/messages/chat/responses。chat/responses 渠道当前在 handler/converter 层默认无条件注入 dummy 签名，因此前端不暴露 `injectDummy` 开关以避免误导。
- **`maskKey` 短密钥脱敏过度**：
  - 修复了 `backend-go/main.go` 中启动日志 `maskKey` 对长度 ≤ 4 的密钥直接输出 `****`、完全遮蔽用户能识别的字符的问题。
  - 现在对长度 ≤ 3 的密钥保留首字符（如 `abc` → `a****`），长度 4~8 保留首尾字符，长度 > 8 保留前后各 2 字符。

### 新增

- **CCX 桌面外壳 MVP** - 新增 Wails3 桌面外壳，用于将现有 CCX 后端作为核心服务构件进行启动、停止、重启、托盘驻留和状态监控；外壳通过现有 `/health` 探活并在内置标签页中加载 CCX Web UI，避免改动核心代理、调度和现有 Web 管理界面逻辑。
  - 新增 `desktop/` Wails3 项目，包含后端子进程 supervisor、托盘菜单、状态页、内嵌 Web UI 标签页和前端绑定。
  - 根 `Makefile` 新增 `desktop-dev` / `desktop-build`，复用现有前端 embed 与 Go 后端构建流程。
  - 调整 `.gitignore`，保留 Wails `desktop/build/` 源配置，同时继续忽略桌面构建产物。

### 修复

- **桌面外壳 Wails Runtime 422** - 升级 `wails3` CLI 与 `@wailsio/runtime` 客户端到 alpha.92 / alpha.79，匹配 alpha.79 引入的 binding transport refactor（请求改用 JSON body），消除旧 CLI（alpha.40）读 URL query 时的 `missing object value` 422 错误，桌面事件订阅（`desktop:show-tab` / `desktop:tray-error`）和 typed binding 调用恢复正常；同步升级前端工具链 vite 8 / vue-tsc 3 / typescript 6 / vue 3.5。
- **空/畸形 Tool Call 自动重试** - 在 Fuzzy 模式下将空参数或非法 JSON 的 tool/function call 视为空响应并复用现有 failover，降低上游偶发 `Read({})` 等畸形工具调用对下游客户端的影响
  - 影响模块：Messages/Chat/Responses/Gemini 的非流式空响应判定、Messages/Responses 流式预检、Chat 流式写头前缓冲预检

## [v2.7.4] - 2026-05-18

### 修复

- **Mimo thinking mode 兼容性** - 修复 mimo（小米 MiMo）等 reasoning model 上游在多轮对话切换场景下返回 400 "The reasoning_content in the thinking mode must be passed back to the API" 的问题
  - 之前实现将 thinking 块转为顶层 `reasoning_content` 字段（OpenAI 风格），实测 mimo 的 Anthropic 协议下不认顶层 `reasoning_content`，仍返回 400
  - 新实现保留所有原有 thinking 块，对缺少 thinking 块的 assistant 消息注入占位 thinking 块 `{"type":"thinking","thinking":"(no prior reasoning recorded)"}`，让 mimo 通过 thinking mode 校验
  - 仅在渠道开启 `passbackReasoningContent: true` 时生效，对其他渠道无影响
  - 注意：旧对话切到 mimo 时模型上下文缺少真实推理内容，可能出现幻觉/指令遵循下降（mimo 官方公告明示的代价）
- **OpenAI Provider 缓存 Token 丢失** - 修复 OpenAI Provider 在协议转换（OpenAI/DeepSeek → Claude Messages）时丢弃上游 cache usage 字段的问题（#76）
  - 流式：`HandleStreamResponse` 现在从 terminal usage chunk（`choices: []`）中提取 `prompt_cache_hit_tokens`、`prompt_tokens_details.cached_tokens` 等字段，并注入到 final `message_delta.usage` 中
  - 非流式：`ConvertToClaudeResponse` 新增二次 raw parse，将 DeepSeek/OpenAI 格式的 cache 字段映射到 `CacheReadInputTokens`
  - Metrics：`annotatePromptTokensTotalForProvider` 扩展到 OpenAI Provider，确保缓存命中归一化口径与 Responses Provider 一致

## [v2.7.3] - 2026-05-18

### 优化

- **Docker 构建优化** - 使用 Go 交叉编译替代 QEMU 模拟，优化多架构镜像构建流程并增加层缓存，显著缩短 CI 构建时间

### 修复

- **Mimo reasoning_content 回传修复** - 修复 `convertThinkingToReasoningContent` 在将 thinking 块提取为 `reasoning_content` 字段时未从 content 数组中移除 thinking 块的问题；当从 Claude 原生渠道切换到 mimo 渠道时，残留的 thinking 块导致 mimo 上游返回 400 "The reasoning_content in the thinking mode must be passed back to the API" 错误

## [v2.7.2] - 2026-05-17

### 新增

- **缓存读写总统计** - 全局统计图表和模型统计图表新增缓存 Token 读写统计展示
  - 后端：`ModelHistoryDataPoint` 新增 `cacheCreationTokens`/`cacheReadTokens` 字段；`GetModelStatsHistory` 和全局统计的模型分桶聚合逻辑补充缓存 Token 累加
  - 前端 `GlobalStatsChart`：Summary cards 和 compact summary 新增缓存 R/W 统计（有数据时显示）；Tokens 图表视图动态添加 Cache Read/Write 系列线
  - 前端 `ModelStatsChart`：新增 Cache 视图切换，展示按模型分组的缓存 Token 趋势
  - 前端 `api.ts`：`ModelHistoryDataPoint` 类型补充缓存字段

## [v2.7.1] - 2026-05-17

### 新增

- **渠道统计对齐接入点统计** - 渠道 Key 趋势图新增 7d/30d 时间维度，并展示总请求次数、成功率、输入/输出 Token 汇总卡片，与接入点总览统计对齐 (#72)
  - 后端：`HistoryDataPoint` 补充 Token 字段；`MetricsHistoryResponse` / `ChannelKeyMetricsHistoryResponse` 新增 `summary` 汇总；Key 趋势接口移除 24h 上限，支持 30 天范围 SQLite 聚合
  - 前端：`KeyTrendChart` 新增 7d/30d 按钮、summary cards、长时间范围 x 轴日期格式
  - SQLite：新增 `idx_records_api_type_metrics_key_timestamp` 复合索引优化渠道级长范围查询

### 改进

- **Override 熔断自动清除** - 当驾驶舱设置的 override（next channel）序列中所有渠道均不可用（熔断）时，调度器自动清除该 override 而非仅跳过，避免前端 NEXT 标签长期残留
- **渠道熔断状态下发前端** - `GetConversationChannelsByKind` API 返回的渠道信息新增 `circuitOpen` 字段，前端驾驶舱可实时显示渠道熔断状态（FUSED 标记）

### 修复

- **Claude 协议空 Text Block 兼容开关** - 修复严格校验的第三方 Claude 协议上游因 Claude Code 在 `tool_use` 前插入裸空 `{"type":"text","text":""}` 占位块而返回 400 的兼容性问题；新增 Messages 渠道 `stripEmptyTextBlocks` 开关，在转发前按需移除空 text block，并同步接通前端配置、渠道视图与回归测试。
- **Responses SSE keep-alive** - 为 Responses 流式代理增加 SSE keep-alive 机制，每 15 秒向下游发送 `: keepalive` 注释行，防止 DeepSeek 等慢上游思考期间触发 Codex 客户端 idle timeout 断连 (#67)

## [v2.7.0] - 2026-05-17

### 新增

- **Responses Compact 本地压缩** - 当 responses 渠道上游为非原生 Responses 类型（openai/claude/gemini）时，`/v1/responses/compact` 端点自动切换为本地 compact 模式：将对话历史格式化为 transcript，通过现有 converter 管线发送普通请求让模型生成摘要，再包装为 Responses 格式返回。支持流式/非流式跟随客户端、session 历史读取与 compact 结果写回、大输入截断保护。原生 responses 上游若返回 404/405/501 也会自动回退本地 compact
- **SessionManager 新增只读查询与压缩会话创建** - 新增 `GetSessionByResponseID` 通过 responseID 只读查找 session；新增 `CreateCompactedSession` 创建压缩后的轻量会话并记录映射
- **ResponsesProvider 提取公共请求构建方法** - 新增 `ConvertBodyToProviderRequest` 公共入口，接受 bodyBytes 参数复用现有 URL/转换/认证逻辑，供 compact 等场景调用

### 改进

- **Messages 渠道模型列表兼容国内 Claude 协议入口** - 当 `base_url` 以 `/anthropic`、`/claude`、`/messages` 结尾时（如 `https://api.deepseek.com/anthropic`），模型列表获取自动尝试三段候选 URL：当前路径 → 剔除协议尾段 → 纯域名根路径，解决国内服务商模型接口不在兼容协议子路径下的问题。管理端"获取模型"同步适配。

---

## 历史版本（v2.0.0-go ~ v2.6.99）

> 以下为早期版本的里程碑摘要，`v2.7.0` 及之后保留完整条目。更细粒度的历史变更可通过 `git log` 查看。

### v2.6.x（2026-02 ~ 2026-05）

- **渠道能力测试** - 完成 4 协议互转、Codex 自定义工具兼容、RPM/超时/取消/重定向验证、结果缓存与多协议并发轮询。
- **独立 Images 渠道** - 新增 `/v1/images/generations`、`/v1/images/edits`、`/v1/images/variations`，接入独立 failover、metrics、日志和前端管理。
- **熔断器三态状态机** - 引入 `closed` / `open` / `half_open`、指数退避、单探针恢复、持久化和 Dashboard 状态展示。
- **渠道日志生命周期** - 支持 `pending`、`connecting`、`first_byte`、`streaming`、`completed`、`failed`、`cancelled` 全链路追踪与前端实时刷新。
- **会话调度看板** - 新增 Conversation Dashboard、ConversationTracker、OverrideManager 与按对话覆盖渠道序列能力。
- **Responses Compact 本地压缩** - 非原生 Responses 上游自动通过 converter 管线执行本地 compact，并写回轻量会话。
- **Vision 能力路由** - 新增 `noVision`、`noVisionModels`、`visionFallbackModel`，按图片输入动态跳过或降级模型。
- **Thinking / Reasoning 兼容** - 支持 DeepSeek、MiMo、Gemini 等上游的 `thinking`、`reasoning_content`、`thought_signature` 双向转换和回传。
- **Codex 工具兼容层** - 支持 Codex `apply_patch` 代理工具、字符串工具、native tool passthrough、工具调用回写和 `codexToolCompat` 统一开关。
- **模型与路由能力增强** - 支持模型过滤包含/排除规则、`X-Channel` 指定目标渠道、渠道级代理、自定义请求头、多 BaseURL 等价去重。
- **缓存与指标口径统一** - 归一 Responses / Messages 缓存命中率、Token usage、Key / Model 维度统计和历史分桶。
- **自动恢复机制** - 支持 UTC 0/8/16 时段自动恢复黑名单 key、half-open 探测和 suspended 渠道最小激活。
- **前端体验与性能优化** - 完成多语言管理界面、移动端导航、能力测试弹窗重构、SVG/Tooltip 常驻资源收敛和内存优化。
- **关键修复** - 覆盖 Fuzzy 模式 5xx 误判、空响应 failover、metadata.user_id JSON 对象兼容、401 字符串认证错误拉黑、客户端取消误计失败等问题。

### v2.5.x（2026-01）

- **Failover 逻辑模块化** - 提取通用 failover 分类、重试和黑名单判定逻辑。
- **Gemini 兼容增强** - `thought_signature` 可配置，补齐 Gemini CLI tools schema 与 Dashboard 字段。
- **指标记录架构优化** - 修复二次计数、客户端取消计数、换 key 后历史累计、渠道删除后串桶等问题。
- **渠道管理增强** - 支持渠道置顶/置底、前端模型映射智能选择、隐式缓存读取推断。
- **关键修复** - 包括 Gemini functionDeclaration parameters 类型、Models API 日志标签、响应 model 字段改写可配置化。

### v2.4.x（2025-12 ~ 2026-01）

- **Gemini API 完整实现** - 新增 Gemini API 模块、路由、历史指标和前端管理界面。
- **统计图表增强** - 新增全局流量 / Token 图、Key 趋势图、缓存命中率统计和多模型趋势展示。
- **低质量渠道处理机制** - 支持 `lowQuality` 渠道标记、暂停清理促销期和输出验证日志。
- **配置模块化拆分** - 按渠道类型拆分 config 包，减少单文件配置膨胀。
- **多端点调度优化** - 支持多 BaseURL failover、预热排序、非阻塞动态排序和并发竞争修复。
- **关键修复** - 覆盖流式工具调用分裂、空 signature 导致 Claude API 400、内容审核无限重试、403 预扣费降级、SSE 事件完整性等问题。

### v2.3.x（2025-12）

- **渠道级 API Key 策略与多 BaseURL** - 支持多 key / 多 URL 管理、促销期状态展示和延迟测试优化。
- **SQLite 指标持久化** - 引入本地指标存储，补齐 Responses API Token usage 与 Messages 缓存 TTL 细分统计。
- **快速添加渠道** - 支持协议自动检测、引号 / 等号 / 空格分割、`settings.json` 解析和多 Base URL 输入。
- **Models API 端点** - 新增模型聚合查询能力并完善 API Key / Base URL 去重。
- **关键修复** - 包括滑动窗口重建、Responses usage 缺失、缓存 token 检测、total_tokens 零值补全。

### v2.2.x ~ v2.1.x（2025-12）

- **Handlers 模块重构** - 将 handlers 重构为同级子包结构，完善 Stream 错误处理与 CountTokens 安全加固。
- **指标系统重构** - 指标绑定到 Key 级别，修复熔断器、单渠道路径指标和非 failover 错误计数。
- **错误处理与历史数据** - 新增 Fuzzy Mode 错误处理开关、渠道指标历史数据 API、Key 级使用趋势图和失败率可视化。
- **协议兼容增强** - 支持 TransformerMetadata、CacheControl、BaseURL `#` 结尾语义和 FinishReason 统一映射。
- **稳定性修复** - 覆盖请求体大小限制、goroutine 泄漏、数据竞争、优雅关闭、流式日志合成器类型等问题。

### v2.0.x（2025-10 ~ 2025-12）

- **Go 版本初始发布** - 完成核心功能移植、单文件部署和性能提升。
- **核心组件落地** - 引入 ChannelScheduler、MetricsManager、TraceAffinityManager 等基础模块。
