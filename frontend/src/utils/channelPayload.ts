import type { Channel, UpstreamModelCapability } from '../services/api'
import { normalizeAdvancedChannelOptions } from './channelAdvancedOptions'
import { deduplicateEquivalentBaseUrls } from './baseUrlSemantics'

export interface ChannelFormLike {
  name: string
  serviceType: 'openai' | 'gemini' | 'claude' | 'responses' | ''
  baseUrl: string
  baseUrls: string[]
  website: string
  insecureSkipVerify: boolean
  lowQuality: boolean
  injectDummyThoughtSignature: boolean
  stripThoughtSignature: boolean
  passbackReasoningContent: boolean
  passbackThinkingBlocks: boolean
  description: string
  apiKeys: string[]
  modelMapping: Record<string, string>
  modelCapabilitiesText?: string
  defaultContextWindowTokens?: string | number | null
  defaultMaxOutputTokens?: string | number | null
  allowUnknownContext?: boolean
  reasoningMapping: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>
  reasoningParamStyle: 'reasoning' | 'reasoning_effort' | 'thinking'
  textVerbosity: 'low' | 'medium' | 'high' | ''
  fastMode: boolean
  customHeaders: Record<string, string>
  proxyUrl: string
  requestTimeoutMs?: string | number | null
  responseHeaderTimeoutMs?: string | number | null
  streamFirstContentTimeoutMs?: string | number | null
  streamInactivityTimeoutMs?: string | number | null
  streamToolCallIdleTimeoutMs?: string | number | null
  rateLimitRpm?: string | number | null
  rateLimitWindowMinutes?: string | number | null
  rateLimitBurst?: string | number | null
  rateLimitMaxConcurrent?: string | number | null
  rateLimitAutoFromHeaders?: boolean
  routePrefix: string
  supportedModels: string[]
  autoBlacklistBalance: boolean
  normalizeMetadataUserId: boolean
  stripBillingHeader?: boolean
  stripEmptyTextBlocks: boolean
  normalizeSystemRoleToTopLevel: boolean
  codexNativeToolPassthrough: boolean
  codexToolCompat: boolean
  normalizeNonstandardChatRoles?: boolean
  stripCodexClientTools?: boolean
  stripImageGenerationTool?: boolean
  convertImageUrlToB64Json?: boolean
  noVision: boolean
  noVisionModels: string[]
  visionFallbackModel: string
  historicalImageTurnLimit?: string | number | null

}

export function parseModelCapabilitiesText(text?: string): Record<string, UpstreamModelCapability> | null {
  const trimmed = (text || '').trim()
  if (!trimmed) return {}

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    return null
  }

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return null
  }

  const result: Record<string, UpstreamModelCapability> = {}
  for (const [model, rawCapability] of Object.entries(parsed as Record<string, unknown>)) {
    const modelName = model.trim()
    if (!modelName || !rawCapability || typeof rawCapability !== 'object' || Array.isArray(rawCapability)) {
      return null
    }

    const capability = rawCapability as Record<string, unknown>
    const normalized: UpstreamModelCapability = {}
    const contextWindowTokens = capability.contextWindowTokens
    if (contextWindowTokens !== undefined) {
      if (typeof contextWindowTokens !== 'number' || !Number.isInteger(contextWindowTokens) || contextWindowTokens < 0) return null
      normalized.contextWindowTokens = contextWindowTokens
    }
    const maxOutputTokens = capability.maxOutputTokens
    if (maxOutputTokens !== undefined) {
      if (typeof maxOutputTokens !== 'number' || !Number.isInteger(maxOutputTokens) || maxOutputTokens < 0) return null
      normalized.maxOutputTokens = maxOutputTokens
    }
    if (capability.thinkingMode !== undefined) {
      if (typeof capability.thinkingMode !== 'string') return null
      normalized.thinkingMode = capability.thinkingMode
    }
    if (capability.reasoningEfforts !== undefined) {
      if (!Array.isArray(capability.reasoningEfforts) || !capability.reasoningEfforts.every(v => typeof v === 'string')) return null
      normalized.reasoningEfforts = capability.reasoningEfforts
    }

    result[modelName] = normalized
  }

  return result
}

type SelectableString = string | { title?: string; value?: unknown } | null | undefined

export function normalizeSelectableString(value: SelectableString): string {
  if (!value) return ''
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed.startsWith('{')) return value
    try {
      const parsed = JSON.parse(trimmed)
      if (parsed && typeof parsed === 'object' && 'value' in parsed) {
        return normalizeSelectableString(parsed as SelectableString)
      }
    } catch {
      return value
    }
    return value
  }
  if (typeof value.value === 'string') return value.value
  if (value.value != null) return String(value.value)
  return ''
}

export function buildChannelPayload(form: ChannelFormLike): Omit<Channel, 'index' | 'latency' | 'status'> {
  const processedApiKeys = form.apiKeys.filter(key => key.trim())
  const advancedOptions = normalizeAdvancedChannelOptions(form.serviceType, {
    reasoningMapping: form.reasoningMapping,
    reasoningParamStyle: form.reasoningParamStyle,
    textVerbosity: form.textVerbosity,
    fastMode: form.fastMode
  })

  const sourceUrls = form.baseUrls.length > 0 ? form.baseUrls : [form.baseUrl]
  const deduplicatedUrls = deduplicateEquivalentBaseUrls(sourceUrls, form.serviceType)

  // 清洗 modelMapping：v-combobox 选中下拉后 key/value 都可能是 { title, value } 对象。
  const cleanModelMapping: Record<string, string> = {}
  for (const [source, target] of Object.entries(form.modelMapping)) {
    const cleanSource = normalizeSelectableString(source).trim()
    const cleanTarget = normalizeSelectableString(target as SelectableString).trim()
    if (cleanSource && cleanTarget) {
      cleanModelMapping[cleanSource] = cleanTarget
    }
  }

  const modelCapabilities = parseModelCapabilitiesText(form.modelCapabilitiesText)
  const defaultContextWindowTokens = Number(form.defaultContextWindowTokens)
  const defaultMaxOutputTokens = Number(form.defaultMaxOutputTokens)

  const channelData: Omit<Channel, 'index' | 'latency' | 'status'> = {
    name: form.name.trim(),
    serviceType: form.serviceType as 'openai' | 'gemini' | 'claude' | 'responses',
    baseUrl: deduplicatedUrls[0] || '',
    website: form.website.trim(),
    insecureSkipVerify: form.insecureSkipVerify,
    lowQuality: form.lowQuality,
    injectDummyThoughtSignature: form.injectDummyThoughtSignature,
    stripThoughtSignature: form.stripThoughtSignature,
    passbackReasoningContent: form.passbackReasoningContent,
    passbackThinkingBlocks: form.passbackThinkingBlocks,
    description: form.description.trim(),
    apiKeys: processedApiKeys,
    modelMapping: cleanModelMapping,
    modelCapabilities: modelCapabilities || {},
    defaultCapability: {},
    allowUnknownContext: !!form.allowUnknownContext,
    reasoningMapping: advancedOptions.reasoningMapping,
    reasoningParamStyle: advancedOptions.reasoningParamStyle,
    textVerbosity: advancedOptions.textVerbosity,
    fastMode: advancedOptions.fastMode,
    customHeaders: form.customHeaders,
    proxyUrl: form.proxyUrl.trim(),
    routePrefix: form.routePrefix.trim(),
    supportedModels: form.supportedModels,
    autoBlacklistBalance: form.autoBlacklistBalance,
    normalizeMetadataUserId: form.normalizeMetadataUserId,
    stripBillingHeader: !!form.stripBillingHeader,
    stripEmptyTextBlocks: form.stripEmptyTextBlocks,
    normalizeSystemRoleToTopLevel: form.normalizeSystemRoleToTopLevel,
    codexNativeToolPassthrough: form.codexNativeToolPassthrough,
    codexToolCompat: form.codexToolCompat,
    normalizeNonstandardChatRoles: !!form.normalizeNonstandardChatRoles,
    stripCodexClientTools: form.codexToolCompat,
    stripImageGenerationTool: !!form.stripImageGenerationTool,
    convertImageUrlToB64Json: !!form.convertImageUrlToB64Json,
    noVision: form.noVision,
    noVisionModels: form.noVisionModels,
    visionFallbackModel: normalizeSelectableString(form.visionFallbackModel as SelectableString),
  }

  if (Number.isInteger(defaultContextWindowTokens) && defaultContextWindowTokens > 0) {
    channelData.defaultCapability!.contextWindowTokens = defaultContextWindowTokens
  }
  if (Number.isInteger(defaultMaxOutputTokens) && defaultMaxOutputTokens > 0) {
    channelData.defaultCapability!.maxOutputTokens = defaultMaxOutputTokens
  }

  // 历史图片轮次限制：始终发送（含 0），使编辑场景能把渠道级覆盖清回 0（继承全局）。
  // 0=继承全局；后端会对 >0 的值应用最低 3 约束。空/非整数/负数归一为 0。
  const historicalImageTurnLimit = Number(form.historicalImageTurnLimit)
  ;(channelData as any).historicalImageTurnLimit =
    Number.isInteger(historicalImageTurnLimit) && historicalImageTurnLimit > 0
      ? historicalImageTurnLimit
      : 0

  if (deduplicatedUrls.length > 1) {
    channelData.baseUrls = deduplicatedUrls
  }

  const requestTimeoutMs = Number(form.requestTimeoutMs)
  if (Number.isInteger(requestTimeoutMs) && requestTimeoutMs >= 1000 && requestTimeoutMs <= 300000) {
    channelData.requestTimeoutMs = requestTimeoutMs
  }

  const responseHeaderTimeoutMs = Number(form.responseHeaderTimeoutMs)
  if (Number.isInteger(responseHeaderTimeoutMs) && responseHeaderTimeoutMs >= 1000 && responseHeaderTimeoutMs <= 300000) {
    channelData.responseHeaderTimeoutMs = responseHeaderTimeoutMs
  }

  const streamFirstContentTimeoutMs = Number(form.streamFirstContentTimeoutMs)
  if (Number.isInteger(streamFirstContentTimeoutMs) && streamFirstContentTimeoutMs > 0) {
    channelData.streamFirstContentTimeoutMs = streamFirstContentTimeoutMs
  }

  const streamInactivityTimeoutMs = Number(form.streamInactivityTimeoutMs)
  if (Number.isInteger(streamInactivityTimeoutMs) && streamInactivityTimeoutMs > 0) {
    channelData.streamInactivityTimeoutMs = streamInactivityTimeoutMs
  }

  const streamToolCallIdleTimeoutMs = Number(form.streamToolCallIdleTimeoutMs)
  if (Number.isInteger(streamToolCallIdleTimeoutMs) && streamToolCallIdleTimeoutMs >= 30000) {
    channelData.streamToolCallIdleTimeoutMs = streamToolCallIdleTimeoutMs
  }

  const rateLimitRpm = Number(form.rateLimitRpm)
  if (Number.isInteger(rateLimitRpm) && rateLimitRpm > 0) {
    channelData.rateLimitRpm = rateLimitRpm
  }

  const rateLimitWindowMinutes = Number(form.rateLimitWindowMinutes)
  if (Number.isInteger(rateLimitWindowMinutes) && rateLimitWindowMinutes > 0) {
    channelData.rateLimitWindowMinutes = rateLimitWindowMinutes
  }

  const rateLimitBurst = Number(form.rateLimitBurst)
  if (Number.isInteger(rateLimitBurst) && rateLimitBurst > 0) {
    channelData.rateLimitBurst = rateLimitBurst
  }

  const rateLimitMaxConcurrent = Number(form.rateLimitMaxConcurrent)
  if (Number.isInteger(rateLimitMaxConcurrent) && rateLimitMaxConcurrent > 0) {
    channelData.rateLimitMaxConcurrent = rateLimitMaxConcurrent
  }

  channelData.rateLimitAutoFromHeaders = !!form.rateLimitAutoFromHeaders

  return channelData
}
