# Claude Code 接入 CCX

如果你正在使用 **CCX Desktop**，可先在 [Agent Config](/guide/desktop/#Agent配置) 中写入本地 CCX 配置，再回到本页确认 Claude Code 的协议细节。

## 工作方式

```text
Claude Code  ->  CCX /v1/messages  ->  Messages 渠道  ->  上游 Anthropic 兼容端点
```

## 一、配置 CCX 渠道

1. 打开 CCX 管理界面，进入 **Messages** 入口
2. 点击「添加渠道」
3. 按上游提供商填写渠道配置

常见配置：

| 上游 | 服务类型 | Base URL 示例 |
|------|----------|---------------|
| Claude 官方 | `Claude` | `https://api.anthropic.com` |
| DeepSeek Anthropic 兼容 | `Claude` | `https://api.deepseek.com/anthropic` |
| Kimi 编码端点 | `Claude` | `https://api.kimi.com/coding/` |
| GLM Anthropic 兼容 | `Claude` | `https://open.bigmodel.cn/api/anthropic` |

::: tip
具体上游的 API Key、模型名和特殊开关，请参考对应的[提供商配置教程](/providers/)。
:::

## 二、配置 Claude Code

在终端中设置：

```bash
export ANTHROPIC_API_KEY="your-ccx-proxy-key"
export ANTHROPIC_BASE_URL="http://localhost:3000"
```

然后运行：

```bash
claude "你好"
```

::: warning
`ANTHROPIC_BASE_URL` 填 CCX 网关根地址，不要加 `/v1` 或 `/v1/messages`。
:::

## 三、模型映射建议

Claude Code 可能会使用 Claude 模型名发起请求，例如：

- `claude-opus-4-7`
- `claude-sonnet-4-6`
- `claude-haiku-4-5-20251001`

如果上游不是 Claude 官方，建议在 Messages 渠道中配置模型映射：

| 请求模型匹配 | 映射到上游模型示例 |
|--------------|--------------------|
| `opus` | 上游高能力模型 |
| `sonnet` | 上游主力模型 |
| `haiku` | 上游轻量模型 |

CCX 会按更长的匹配键优先匹配。模型名和映射值以你实际配置的上游为准。

## 常见问题

### Claude Code 返回 401 Unauthorized

检查：

1. `ANTHROPIC_API_KEY` 是否等于 CCX 的 `PROXY_ACCESS_KEY`。
2. CCX 启动时是否设置了正确的 `PROXY_ACCESS_KEY`。
3. Shell 中是否存在旧的环境变量覆盖当前值。
4. 若使用 CCX Desktop，可在托盘菜单或 **Gateway Monitor** 复制当前 `PROXY_ACCESS_KEY`。

### 配置后仍然请求官方 Claude

检查 Claude Code 当前认证方式。使用 `ANTHROPIC_API_KEY` 和 `ANTHROPIC_BASE_URL` 时，应确保当前终端会话能读取这两个变量。

```bash
printenv ANTHROPIC_BASE_URL
printenv ANTHROPIC_API_KEY
```

### 提示 404 或接口不存在

通常是 `ANTHROPIC_BASE_URL` 写错。

正确：

```bash
export ANTHROPIC_BASE_URL="http://localhost:3000"
```

错误示例：

```bash
export ANTHROPIC_BASE_URL="http://localhost:3000/v1"
export ANTHROPIC_BASE_URL="http://localhost:3000/v1/messages"
```

### 返回 model_not_found

检查 Messages 渠道：

1. 模型白名单是否包含请求模型，或模型映射后的上游模型
2. 模型映射是否覆盖了 Claude Code 实际请求的模型名
3. 上游实际模型名是否填写正确

### thinking 或 signature 相关错误

如果上游不完整兼容 Claude thinking/signature 行为，建议：

1. 优先使用真正支持 Claude Messages 协议的上游
2. 检查当前渠道是否开启或关闭了与 reasoning/thinking 相关的兼容选项
3. 避免把不兼容上游放在同一优先级链路中处理带 thinking 的长上下文会话

### Claude Code 的 `/model` 看不到预期模型

确认 CCX 至少有一个可用的 Messages 渠道，并且模型白名单或模型列表已包含对应模型。也可以用下面命令验证 CCX 模型接口：

```bash
curl http://localhost:3000/v1/models \
  -H "Authorization: Bearer your-ccx-proxy-key"
```
