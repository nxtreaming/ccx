---
name: upstream-check
description: 检查 Claude Code / Codex 上游版本变更，对比本地版本，识别协议/工具/用法相关更新并追加 TODO 提醒
version: 1.0.0
author: https://github.com/BenedictKing/ccx/
allowed-tools: Bash, Read, Write, Edit
context: fork
---

# 上游版本检查技能

当用户输入包含以下关键词时，自动触发上游版本检查流程：

## 触发条件

- "检查上游版本"、"上游检查"、"检查更新"、"upstream check"、"check upstream"

## 执行步骤

### 1. 运行检查脚本

```bash
bash .claude/skills/upstream-check/scripts/upstream-check.sh
```

### 2. 读取当前状态

```bash
cat .claude/skills/upstream-check/scripts/upstream-state.json
```

### 3. 分析结果

根据脚本输出的 JSON 判断：

- **`claude_code.up_to_date`** / **`codex.up_to_date`**：版本是否相同
- **`claude_code.protocol_changes`** / **`codex.protocol_changes`**：是否有协议/工具/用法相关变更
- **`matched_keywords`**：具体匹配了哪些关键字

### 4. 升级建议逻辑

| 条件 | 输出 |
|------|------|
| `up_to_date: true` 且 `protocol_changes: false` | "✅ 已是最新版本，无需关注。" |
| `up_to_date: true` 且 `protocol_changes: true` | "✅ 版本已是最新。以下版本发布说明涉及协议/工具/用法变更，值得阅读了解：[关键字列表]" |
| `up_to_date: false` 且 `protocol_changes: false` | "⬆️ 有新版本 [remote_version] 可用（本地: [local_version]）。非紧急，可在方便时升级。" |
| `up_to_date: false` 且 `protocol_changes: true` | "⬆️ 有新版本 [remote_version] 可用（本地: [local_version]），涉及协议/工具/用法变更：[关键字列表]。**建议关注并评估对 ccx 的影响。**" |

### 5. 更新 TODO.md

**仅当 `protocol_changes: true` 且远程 tag 不在 `seen_tags` 中时**，追加 TODO 条目。

**去重检查（必须）**：

1. 读取 `.claude/skills/upstream-check/scripts/upstream-state.json`，检查远程 tag 是否已在 `seen_tags` 中
2. 如已在，跳过 TODO 追加
3. 追加后，将 tag 加入 `seen_tags`（上限 20 条，超出时删除最早的）

**TODO.md 更新**：

- 检查 TODO.md 是否已有 `## 上游版本变更` 标题，如无则追加
- 追加格式：

```markdown
- [Claude Code vX.Y.Z] 发现协议/工具/用法变更：keyword1, keyword2。请评估对 ccx Messages 渠道的影响。
- [Codex rust-vX.Y.Z] 发现协议/工具/用法变更：keyword1, keyword2。请评估对 ccx Responses 渠道的影响。
```

### 6. 更新状态文件

将新的远程 tag 追加到 `.claude/skills/upstream-check/scripts/upstream-state.json` 的 `seen_tags` 数组中：

```json
{
  "last_checked_at": "2026-06-07T12:00:00Z",
  "claude_code": {
    "remote_tag": "v2.1.168",
    "local_version": "2.1.168",
    "seen_tags": ["v2.1.168"]
  },
  "codex": {
    "remote_tag": "rust-v0.137.0",
    "local_version": "0.137.0",
    "seen_tags": ["rust-v0.137.0"]
  }
}
```

### 7. 输出摘要报告

格式示例：

```
📋 上游版本检查结果

## Claude Code
- 本地版本: 2.1.168
- 远程版本: 2.1.168
- 状态: ✅ 已是最新
- 协议/工具变更: 无

## Codex
- 本地版本: 0.137.0
- 远程版本: 0.137.0
- 状态: ✅ 已是最新
- 协议/工具变更: 发现变更 - multi-agent, skills, plugins, web search
```

## 说明

- 脚本只输出 JSON 到 stdout，不写磁盘
- 状态持久化由 skill 负责
- 脚本网络失败时（exit 1），不更新状态，仅输出错误
- `matched_keywords` 包含协议兼容性、新工具/能力、用法变化三类关键字
