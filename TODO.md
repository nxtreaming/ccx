# TODO

## [x] 优化 200 超时重试机制

当前代理层在上游返回 HTTP 200 但实际超时（如流式响应中断、空响应体等场景）时的重试行为需要优化，确保能够正确识别"伪成功"响应并触发跨渠道故障转移。

**关键提交：**
- `017597c7` feat(stream): 新增流式 200 伪成功检测与 failover 机制
- `e15e9fc4` feat(chat): Chat handler 流式两阶段 preflight 检测
- `30bc1535` feat(responses): Responses handler 流式两阶段 preflight 检测
- `4fca03a6` fix(stream): 捕捉已提交流式响应断流

## [x] 渠道级配置超时参数

支持为每个渠道独立配置超时参数（连接超时、读取超时、总超时），替代当前的全局统一超时设置，以适应不同上游服务商的响应速度差异。

**关键提交：**
- `c7091345` feat(channels): 支持渠道级请求超时配置
- `29549dea` fix(stream): 移除流式超时 off 档并为渠道新增超时覆盖开关
- `1c14aea2` feat(web): Web 前端熔断器配置新增流式超时滑块
- `b1e1bb0a` feat(desktop): Desktop 前端熔断器配置新增流式超时滑块

## [x] Windows 桌面应用图标透明度

Windows 系统下桌面应用图标周边没有透明背景，需要修正图标资源以确保图标在 Windows 任务栏和桌面显示时具有正确的透明边缘。

**关键提交：**
- `cf1d867b` fix(desktop): 修复 Windows 图标透明度并补全 Codex 预设模板

## [ ] 桌面端渠道中心成功提示清理

桌面端渠道中心中，一个渠道添加成功后切换到另一个渠道时，之前的"添加成功"提示没有清除。需要在渠道切换或表单重置时同步清理成功提示状态，避免误导用户。

## [ ] GPT 类型上游模型测试覆盖 codex-auto-review

GPT 类型的上游模型测试用例应包含 `codex-auto-review`，确保 Codex 自动评审相关模型能力在 GPT 类渠道中被覆盖。

## [x] 桌面端同步 stripImageGenerationTool 开关

Web 端与后端已支持渠道级「去除 image_generation 工具」开关（Responses/Chat 透传路径剥离 image_generation 工具，规避无图片生成权限上游的 permission_error 误拉黑）。桌面端尚未同步，需补：`desktop/internal/channelpreset/preset.go` 已加字段，仍需 `desktop/frontend/src/services/admin-api.ts` 类型、`desktop/frontend/src/utils/channel-payload.ts`、`desktop/frontend/src/components/console/ChannelEditDialog.vue` 表单/UI/预设。

**关键提交：**
- `2a46906c` feat(channel): 新增 stripImageGenerationTool 渠道开关（Web 端 + 后端）
- 本次提交完成桌面端同步

## 疑似bug

issues #162 #188 #187
