import type { Channel, ChannelDiscoveryKind, ChannelKind } from '@/services/api-types'

const DISCOVERABLE_CHANNEL_KINDS = new Set<ChannelDiscoveryKind>(['messages', 'chat', 'responses', 'gemini'])

const DEFAULT_SERVICE_TYPES: Record<ChannelKind, Channel['serviceType']> = {
  messages: 'claude',
  chat: 'openai',
  responses: 'responses',
  gemini: 'gemini',
  images: 'openai',
  vectors: 'openai'
}

export function buildQuickAddChannelName(baseUrl: string, suffix: string): string {
  try {
    const hostname = new URL(baseUrl).hostname.replace(/^www\./i, '').replace(/\./g, '-')
    return `${hostname || 'channel'}-${suffix}`
  } catch {
    return `channel-${suffix}`
  }
}

export function supportsQuickAddProtocolDiscovery(kind: ChannelKind): kind is ChannelDiscoveryKind {
  return DISCOVERABLE_CHANNEL_KINDS.has(kind as ChannelDiscoveryKind)
}

export function defaultQuickAddServiceType(kind: ChannelKind): Channel['serviceType'] {
  return DEFAULT_SERVICE_TYPES[kind]
}

export function normalizeDiscoveredChannelKind(kind: string): ChannelDiscoveryKind | null {
  return DISCOVERABLE_CHANNEL_KINDS.has(kind as ChannelDiscoveryKind) ? (kind as ChannelDiscoveryKind) : null
}
