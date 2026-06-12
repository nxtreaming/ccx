import type { Channel } from '@/services/admin-api'
import { normalizeAdvancedChannelOptions } from './channel-advanced-options'
import { deduplicateEquivalentBaseUrls } from './base-url-semantics'

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
    modelMapping: form.modelMapping,
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

  if (deduplicatedUrls.length > 1) {
    channelData.baseUrls = deduplicatedUrls
  }

  // 历史图片轮次限制：始终发送（含 0），使编辑场景能把渠道级覆盖清回 0（继承全局）。
  // 0=继承全局；后端会对 >0 的值应用最低 3 约束。空/非整数/负数归一为 0。
  const historicalImageTurnLimit = Number(form.historicalImageTurnLimit)
  ;(channelData as any).historicalImageTurnLimit =
    Number.isInteger(historicalImageTurnLimit) && historicalImageTurnLimit > 0
      ? historicalImageTurnLimit
      : 0

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
  if (Number.isInteger(streamToolCallIdleTimeoutMs) && streamToolCallIdleTimeoutMs > 0) {
    channelData.streamToolCallIdleTimeoutMs = streamToolCallIdleTimeoutMs
  }

  const rateLimitRpm = Number(form.rateLimitRpm)
  if (Number.isInteger(rateLimitRpm) && rateLimitRpm > 0) {
    channelData.rateLimitRpm = rateLimitRpm
  }

  const rateLimitMaxConcurrent = Number(form.rateLimitMaxConcurrent)
  if (Number.isInteger(rateLimitMaxConcurrent) && rateLimitMaxConcurrent > 0) {
    channelData.rateLimitMaxConcurrent = rateLimitMaxConcurrent
  }

  channelData.rateLimitAutoFromHeaders = !!form.rateLimitAutoFromHeaders

  return channelData
}
