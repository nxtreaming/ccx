# Codex CLI / Codex App 接入 CCX

如果你正在使用 **CCX Desktop**，可先在 [Agent Config](/guide/desktop/#Agent配置) 中写入 Codex 配置，再回到本页确认 Responses 入口和 Base URL 规则。

## 工作方式

```text
Codex CLI / App  ->  CCX /v1/responses  ->  Responses 渠道  ->  上游 Responses 或 Chat 端点
```

## 一、配置 CCX 渠道

1. 打开 CCX 管理界面，进入 **Responses** 入口
2. 点击「添加渠道」
3. 根据上游能力选择服务类型

| 上游能力 | 服务类型 | 说明 |
|----------|----------|------|
| 原生 OpenAI Responses | `Responses` | 直接转发 Responses 协议 |
| OpenAI Chat 兼容 | `OpenAI Chat` | CCX 将 Responses 转换为 Chat Completions |
| Claude | `Claude` | CCX 将 Responses 转换为 Claude Messages |

如果上游是 Chat 兼容端点，且不支持 `developer` 等非标准 Chat role，建议编辑渠道并启用 **规范化非标准 Chat role**。

## 二、通用配置值

Codex CLI 和 Codex App 都使用下面这组值：

| 配置项 | 值 |
|--------|----|
| API Key | `your-ccx-proxy-key` |
| Base URL | `http://localhost:3000/v1` |
| Model | 客户端请求模型名，例如 `gpt-5` |

::: tip
这里的 API Key 是 CCX 的 `PROXY_ACCESS_KEY`，不是上游提供商的 API Key。上游 API Key 只配置在 CCX 渠道里。
:::

## 三、Codex CLI 配置

在终端中设置：

```bash
export OPENAI_API_KEY="your-ccx-proxy-key"
export OPENAI_BASE_URL="http://localhost:3000/v1"
```

然后运行：

```bash
codex "你好"
```

如果你的 Codex CLI 版本使用配置文件而不是环境变量，仍然填写同一组值：

```text
API Key: your-ccx-proxy-key
Base URL: http://localhost:3000/v1
Model: gpt-5
```

## 四、Codex App 配置

在 Codex App 的模型或 Provider 设置中，选择 OpenAI 兼容 / 自定义 API 配置，并填写：

| 设置项 | 值 |
|--------|----|
| API Key | `your-ccx-proxy-key` |
| Base URL | `http://localhost:3000/v1` |
| Model | `gpt-5` 或你的映射模型名 |

## 五、模型映射建议

Codex 默认可能请求 GPT 模型名。如果上游使用不同模型名，建议在 Responses 渠道里配置模型映射：

| 请求模型匹配 | 映射到上游模型示例 |
|--------------|--------------------|
| `gpt` | 上游主力模型 |
| `mini` | 上游轻量模型 |

::: tip
不要只给 `gpt-5` 配高能力模型而忽略 `gpt-5-mini`。CCX 按更长匹配键优先匹配，通常可以用 `gpt` 和 `mini` 覆盖常见 Codex 模型名。
:::

## 常见问题

### Codex CLI 和 Codex App 是否要分开配置？

不需要。两者使用同一套 CCX OpenAI 兼容配置：

```text
API Key  = PROXY_ACCESS_KEY
Base URL = http://localhost:3000/v1
Model    = 客户端请求模型名
```

区别只是 CLI 在终端或配置文件里填，App 在图形界面里填。

### 返回 401 Unauthorized

检查：

1. 优先查看当前环境变量是否设置了 `OPENAI_API_KEY`，它可能覆盖你在 Codex 配置里填写的 Key

   ```bash
   printenv OPENAI_API_KEY
   ```

2. `OPENAI_API_KEY` 或 App 中的 API Key 是否等于 CCX 的 `PROXY_ACCESS_KEY`
3. 是否误填了上游厂商 API Key
4. CCX 服务是否使用了同一个 `PROXY_ACCESS_KEY` 启动

### 返回 404 或接口不存在

检查 Base URL。

正确：

```bash
export OPENAI_BASE_URL="http://localhost:3000/v1"
```

不要填写：

```bash
export OPENAI_BASE_URL="http://localhost:3000"
export OPENAI_BASE_URL="http://localhost:3000/v1/responses"
```

### 返回 model_not_found

检查 Responses 渠道：

1. 模型白名单是否包含请求模型或映射后的上游模型
2. 模型映射是否覆盖 Codex 实际请求的模型名
3. 上游模型名是否填写为该厂商真实支持的名称

### 上游报 role 不支持

如果错误中出现 `developer`、`tool`、`system` 等 role 相关提示，编辑 Responses 渠道并启用 **规范化非标准 Chat role**。这对 DeepSeek 等 Chat 兼容上游尤其常见。

### 流式输出中断

检查：

1. 上游是否稳定支持流式响应
2. 代理或反向代理是否设置了过短超时
3. 当前渠道是否已被熔断或频繁 failover
4. 客户端是否请求了上游不支持的工具调用或响应格式

### 想确认请求是否走 Responses 入口

查看 CCX 后端日志或渠道日志，请求路径应为：

```text
/v1/responses
```

如果你看到 `/v1/chat/completions`，说明客户端当前没有走 Codex Responses 配置，而是走了 OpenAI Chat 配置。
