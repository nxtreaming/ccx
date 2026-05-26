# Claude / OpenAI Chat / OpenAI Images / Codex Responses / Gemini API Proxy - CCX

[English](README.md) | 简体中文

[![GitHub release](https://img.shields.io/github/v/release/BenedictKing/ccx)](https://github.com/BenedictKing/ccx/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

CCX 是一个高性能的 AI API 代理与协议转换网关，支持 Claude、OpenAI Chat、OpenAI Images、Codex Responses 与 Gemini。它提供统一入口、内置 Web 管理界面、渠道编排、故障转移、多密钥管理和模型路由能力。

## 功能特性

- 后端与前端一体化架构，单端口部署
- 双密钥认证：`PROXY_ACCESS_KEY` 与可选 `ADMIN_ACCESS_KEY`
- 内置 Web 管理面板，支持渠道管理、测试、日志和监控
- 同时支持 Claude Messages、OpenAI Chat Completions、OpenAI Images、Codex Responses、Gemini
- 智能调度：优先级、促销期、健康检查、故障转移与熔断恢复
- 每个渠道支持多 API Key 轮换、代理、自定义请求头、模型白名单和路由前缀
- Responses API 支持多轮会话跟踪
- 前端构建产物嵌入后端，便于单二进制部署

## 赞助商

<table>
<tr>
<td width="180"><a href="https://www.compshare.cn/?ytag=GPU_YY_git_ccx"><img src="docs/sponsors/youyun-zhisuan.png" alt="优云智算" width="150"></a></td>
<td>感谢 <a href="https://www.compshare.cn/?ytag=GPU_YY_git_ccx">优云智算</a> 赞助了本项目！优云智算是 UCloud 旗下 AI 云平台，主打包月、按次的高性价比国模 Agent Plan 套餐，低至 49 元/月起。同时提供官转稳定海外模型，支持接入 Claude Code、Codex 及 API 调用，支持企业高并发、7*24 技术支持、自助开票。通过<a href="https://www.compshare.cn/?ytag=GPU_YY_git_ccx">此链接</a>注册的用户，可得免费 5 元平台体验金！</td>
</tr>
</table>

## 界面预览

### 渠道编排

可视化渠道管理，支持拖拽调整优先级，实时查看渠道健康状态和调度统计。

![渠道编排](docs/screenshots/channel-orchestration.png)

### 添加渠道

支持多种上游服务类型，灵活配置 API 密钥、模型映射和请求参数。

<img src="docs/screenshots/add-channel-modal.png" width="500" alt="添加渠道">

### 流量统计

实时监控各渠道的请求流量、成功率和响应延迟。

![流量统计](docs/screenshots/traffic-stats.png)

## 架构概览

CCX 对外提供一个统一后端入口：

```text
客户端 -> backend :3000 ->
  |- /                            -> Web 管理界面
  |- /api/*                       -> 管理 API
  |- /v1/messages                 -> Claude Messages 代理
  |- /v1/chat/completions         -> OpenAI Chat 代理
  |- /v1/responses                -> Codex Responses 代理
  |- /v1/images/{...}             -> OpenAI Images 代理
  |- /v1/models                   -> Models API
  `- /v1beta/models/*             -> Gemini 代理
```

当前 Images 入口包括：
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/images/variations`

详细设计请参考 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 快速开始

### 方式零：CCX Desktop

如果你希望在桌面端快速上手，可优先使用 [CCX Desktop](docs/guide/desktop)：

1. 下载并安装对应平台的桌面应用。
2. 打开应用并按向导配置 `PROXY_ACCESS_KEY`。
3. 启动服务、添加渠道、写入客户端配置后即可开始使用。

### 方式一：直接运行二进制

1. 从 [Releases](https://github.com/BenedictKing/ccx/releases/latest) 下载最新可执行文件；Windows 普通用户可优先使用 Microsoft Store 版本（上架后提供），Store 负责签名与更新
2. 在可执行文件同目录创建 `.env`：

```bash
PROXY_ACCESS_KEY=your-proxy-access-key
PORT=3000
ENABLE_WEB_UI=true
APP_UI_LANGUAGE=zh-CN
```

3. 启动后访问 `http://localhost:3000`

Windows 下如果客户端运行在 cmd、PowerShell、WSL 或 Docker 中，且 `localhost` 无法访问 CCX，建议改用 Windows 主机的局域网 IPv4 地址，例如 `http://192.168.1.23:3000`。CCX 默认通过 `:PORT` 监听所有网卡地址。

需要后台运行或开机自启动时，参考 [非 Docker 自启动](docs/service/README.md)。

### 方式二：Docker

```bash
docker run -d \
  --name ccx \
  -p 3000:3000 \
  -e PROXY_ACCESS_KEY=your-proxy-access-key \
  -e APP_UI_LANGUAGE=zh-CN \
  -v $(pwd)/.config:/app/.config \
  crpi-i19l8zl0ugidq97v.cn-hangzhou.personal.cr.aliyuncs.com/bene/ccx:latest
```

使用 Docker Compose 后台运行：

```bash
docker compose up -d
```

启用 Watchtower 自动更新：

```bash
docker compose -f docker-compose.yml -f docker-compose.watchtower.yml up -d
```

首次部署后如需立即拉取最新镜像：

```bash
docker compose pull ccx
docker compose up -d ccx
```

### 方式三：源码构建

```bash
git clone https://github.com/BenedictKing/ccx
cd ccx
cp backend-go/.env.example backend-go/.env
make run
```

常用命令：

```bash
make dev
make run
make build
make frontend-dev
```

## 主要环境变量

```bash
PORT=3000
ENV=production
ENABLE_WEB_UI=true
PROXY_ACCESS_KEY=your-proxy-access-key
ADMIN_ACCESS_KEY=your-admin-secret-key
APP_UI_LANGUAGE=zh-CN
LOG_LEVEL=info
REQUEST_TIMEOUT=300000
```

## 主要接口

- Web UI：`GET /`
- 健康检查：`GET /health`
- 管理 API：`/api/*`
- Claude Messages：`POST /v1/messages`
- OpenAI Chat：`POST /v1/chat/completions`
- Codex Responses：`POST /v1/responses`
- OpenAI Images：`POST /v1/images/generations`、`POST /v1/images/edits`、`POST /v1/images/variations`
- Gemini：`POST /v1beta/models/{model}:generateContent`
- 模型列表：`GET /v1/models`

## 开发

推荐本地开发方式：

```bash
make dev
```

仅前端：

```bash
cd "frontend"
bun install
bun run dev
```

仅后端：

```bash
cd "backend-go"
make dev
```

## 相关文档

- [CCX Desktop 用户教程](docs/guide/desktop)
- [客户端接入总览](docs/guide/clients)
- [README.md](README.md)
- [backend-go/README.md](backend-go/README.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [DEVELOPMENT.md](DEVELOPMENT.md)
- [ENVIRONMENT.md](ENVIRONMENT.md)
- [docs/service/README.md](docs/service/README.md) - 非 Docker 自启动
- [RELEASE.md](RELEASE.md)

## 社区交流

欢迎加入 QQ 群交流讨论：**642217364**

<img src="docs/qrcode_1769645166806.png" width="300" alt="QQ群二维码">

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=BenedictKing/ccx&type=Date)](https://www.star-history.com/#BenedictKing/ccx&Date)

## 许可证

MIT
