import type { Channel } from '../services/api'
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
  reasoningMapping: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>
  reasoningParamStyle: 'reasoning' | 'reasoning_effort' | 'thinking'
  textVerbosity: 'low' | 'medium' | 'high' | ''
  fastMode: boolean
  customHeaders: Record<string, string>
  proxyUrl: string
  requestTimeoutMs?: string | number | null
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
  noVision: boolean
  noVisionModels: string[]
  visionFallbackModel: string
  historicalImageTurnLimit?: string | number | null

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

  // 清洗 modelMapping：确保所有 value 都是字符串
  // v-combobox 选中下拉后 v-model 可能是 { title, value } 对象，需规整为字符串
  const cleanModelMapping: Record<string, string> = {}
  for (const [source, target] of Object.entries(form.modelMapping)) {
    if (typeof target === 'string') {
      cleanModelMapping[source] = target
    } else if (target && typeof target === 'object' && 'value' in target) {
      cleanModelMapping[source] = String((target as any).value || '')
    }
  }

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
    noVision: form.noVision,
    noVisionModels: form.noVisionModels,
    visionFallbackModel: typeof form.visionFallbackModel === 'object' && form.visionFallbackModel !== null
      ? (form.visionFallbackModel as unknown as { value: string }).value || ''
      : form.visionFallbackModel || '',
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
  if (Number.isInteger(requestTimeoutMs) && requestTimeoutMs > 0) {
    channelData.requestTimeoutMs = requestTimeoutMs
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
