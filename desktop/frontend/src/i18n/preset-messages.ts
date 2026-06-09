// preset-messages.ts
// 渠道预设/Plan/Target 文案的 i18n 翻译表（仅 EN）。
// zh-CN 直接 fallback 到 Go 后端 channelpreset/preset.go 中的原中文，
// 避免在前端重复维护一份中文表。
//
// Key 命名约定：
//   channel.preset.{providerId}.label
//   channel.preset.{providerId}.description
//   channel.preset.{providerId}.plan.{planId}.label
//   channel.preset.{providerId}.plan.{planId}.description
//   channel.preset.{providerId}.target.{type}.description  // provider 级覆盖（仅在需要差异化时定义）
//   channel.target.{type}.label                             // 共用 target 标签
//   channel.target.{type}.description                       // 共用 target 描述（默认值）

import type { SupportedLocale } from './messages'

export const presetMessages: Record<SupportedLocale, Record<string, string>> = {
  en: {
    // 共用 target
    'channel.target.messages.label': 'Messages native',
    'channel.target.chat.label': 'Chat passthrough',
    'channel.target.responses.label': 'Codex Responses',
    'channel.target.messages.description': 'Claude Code direct, or CCX messages channel',
    'channel.target.chat.description': 'OpenAI Chat protocol, for Chat clients',
    'channel.target.responses.description': 'OpenAI Responses protocol, for Codex',

    // DeepSeek
    'channel.preset.deepseek.description':
      'V4-Pro / V4-Flash dual-flagship reasoning models at breakthrough costs. Fully open-source (MIT), delivering frontier-level performance at unbeatable prices.',
    'channel.preset.deepseek.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.deepseek.plan.openai-chat.description': 'Common Chat / Responses endpoint',

    // MiMo
    'channel.preset.mimo.description':
      "Xiaomi's in-house AI platform for the Agent era. Flagship MiMo-V2.5-Pro with trillion parameters and 1M context, powering Xiaomi's smart ecosystem across devices, vehicles, and homes.",
    'channel.preset.mimo.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.mimo.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.mimo.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.mimo.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    'channel.preset.mimo.plan.token-cn.label': 'Token Plan - China (OpenAI)',
    'channel.preset.mimo.plan.token-cn.description': 'China Token Plan Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-sgp.label': 'Token Plan - Singapore (OpenAI)',
    'channel.preset.mimo.plan.token-sgp.description': 'Singapore Token Plan Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-ams.label': 'Token Plan - Europe (OpenAI)',
    'channel.preset.mimo.plan.token-ams.description': 'Europe Token Plan Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-cn-anthropic.label': 'Token Plan - China (Anthropic)',
    'channel.preset.mimo.plan.token-cn-anthropic.description':
      'China Token Plan Claude Messages native endpoint',
    'channel.preset.mimo.plan.token-sgp-anthropic.label': 'Token Plan - Singapore (Anthropic)',
    'channel.preset.mimo.plan.token-sgp-anthropic.description':
      'Singapore Token Plan Claude Messages native endpoint',
    'channel.preset.mimo.plan.token-ams-anthropic.label': 'Token Plan - Europe (Anthropic)',
    'channel.preset.mimo.plan.token-ams-anthropic.description':
      'Europe Token Plan Claude Messages native endpoint',

    // Compshare
    'channel.preset.compshare.label': 'Youyun Zhisuan Plans',
    'channel.preset.compshare.description':
      "Youyun Zhisuan is UCloud's AI cloud platform, offering cost-effective domestic AI model Agent Plan packages by monthly subscription or pay-as-you-go, starting from 49 CNY/month. It also provides stable access to official overseas models, supports Claude Code, Codex, and API integrations, and offers enterprise-grade high concurrency, 24/7 technical support, and self-service invoicing. Users who register through the promotion link can receive a free 5 CNY platform trial credit.",
    'channel.preset.compshare.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.compshare.plan.openai-chat.description':
      'Common Chat / Responses endpoint',

    // RunAPI
    'channel.preset.runapi.label': 'RunAPI',
    'channel.preset.runapi.description':
      "RunAPI is an efficient and stable API platform—an alternative to OpenRouter. A single API Key gives you access to 150+ leading models, including OpenAI, Claude, Gemini, DeepSeek, Grok, and more, at prices as low as 10% of the original (up to 90% off), with exceptional stability. It's seamlessly compatible with tools like Claude Code, OpenClaw, and others. RunAPI offers an exclusive perk for CCX users: register and contact an administrator to claim ¥7 in free credit.",
    'channel.preset.runapi.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.runapi.plan.openai-chat.description':
      'Common Chat / Responses endpoint',

    // Kimi
    'channel.preset.kimi.description':
      "Moonshot AI's intelligent assistant platform. Flagship Kimi K2.6 with native multimodal, renowned for ultra-long context and Agent Swarm capabilities.",
    'channel.preset.kimi.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.kimi.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.kimi.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.kimi.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    'channel.preset.kimi.plan.coding-anthropic.label': 'Coding Plan (Anthropic)',
    'channel.preset.kimi.plan.coding-anthropic.description': 'Coding Plan Claude Messages native endpoint',
    'channel.preset.kimi.plan.coding-openai-chat.label': 'Coding Plan (OpenAI)',
    'channel.preset.kimi.plan.coding-openai-chat.description': 'Coding Plan Chat / Responses shared endpoint',

    // GLM
    'channel.preset.glm.description':
      "Zhipu AI's model-as-a-service platform from Tsinghua. Flagship GLM-5.1 rivals frontier models in coding. Hong Kong-listed, trained entirely on Huawei Ascend chips.",
    'channel.preset.glm.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.glm.plan.coding.label': 'Coding Plan (OpenAI)',
    'channel.preset.glm.plan.coding.description': 'Coding Plan Chat / Responses shared endpoint',
    'channel.preset.glm.plan.openai-chat.label': 'General (OpenAI)',
    'channel.preset.glm.plan.openai-chat.description': 'General Chat / Responses endpoint',

    // MiniMax
    'channel.preset.minimax.description':
      'Global multimodal AI leader (HK-listed). Flagship M2.7 with Agent self-evolution. Hailuo video/audio models in global top tier. Over 70% revenue from international markets.',
    'channel.preset.minimax.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.minimax.plan.openai-chat.description': 'Common Chat / Responses endpoint',

    // DashScope
    'channel.preset.dashscope.label': 'Alibaba Cloud DashScope',
    'channel.preset.dashscope.description':
      "Alibaba Cloud's one-stop LLM platform integrating Qwen and third-party models. Full-modality coverage with pay-as-you-go, Coding Plan, and Token Plan options.",
    'channel.preset.dashscope.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.dashscope.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.dashscope.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.dashscope.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    'channel.preset.dashscope.plan.coding-anthropic.description':
      'Coding Plan Claude Messages native endpoint',
    'channel.preset.dashscope.plan.coding-openai-chat.description':
      'Coding Plan Chat / Responses shared endpoint',
    // Tencent Lkeap
    'channel.preset.tencent-lkeap.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.tencent-lkeap.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Volcengine Ark Coding Plan
    'channel.preset.volc-ark.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.volc-ark.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Baidu Qianfan Coding Plan
    'channel.preset.qianfan.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.qianfan.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Youyun OriginRouter
    'channel.preset.originrouter.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.originrouter.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    'channel.preset.dashscope.plan.token-plan-anthropic.description':
      'Token Plan Claude Messages native endpoint',
    'channel.preset.dashscope.plan.token-plan-openai-chat.description':
      'Token Plan Chat / Responses shared endpoint',

    // OpenCode Zen
    'channel.preset.opencode-zen.description':
      "OpenCode's curated gateway for AI coding models, benchmarked and optimized for coding agents. Zero-markup pay-per-request pricing.",
    'channel.preset.opencode-zen.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.opencode-zen.plan.openai-chat.description':
      'Common Chat / Responses endpoint',

    // OpenCode Go
    'channel.preset.opencode-go.description':
      "OpenCode's low-cost open-source coding model subscription (from $5/month), optimized for coding agent scenarios.",
    'channel.preset.opencode-go.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.opencode-go.plan.openai-chat.description': 'Common Chat / Responses endpoint',

    // OpenRouter
    'channel.preset.openrouter.description':
      "The largest AI model aggregator — access hundreds of models with a single API key. OpenAI-compatible with built-in guardrails for security, governance, and cost control.",
    'channel.preset.openrouter.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.openrouter.plan.openai-chat.description': 'Common Chat / Responses endpoint',

    // ModelScope
    'channel.preset.modelscope.label': 'ModelScope',
    'channel.preset.modelscope.description':
      "Alibaba DAMO Academy's open-source model community with 10,000+ models across text, image, and speech. Full-lifecycle support for model exploration, inference, and training.",
    'channel.preset.modelscope.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.modelscope.plan.openai-chat.description': 'Common Chat / Responses endpoint',
  },
  'zh-CN': {
    // 留空：所有 key 都通过 translateOrFallback 回退到 Go preset 中的原中文。
  },
}
