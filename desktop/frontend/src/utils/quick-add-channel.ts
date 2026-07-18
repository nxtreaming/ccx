import type { Channel } from '@/services/admin-api'
import { canonicalBaseUrl, type ServiceType } from './base-url-semantics'

const QUICK_ADD_URL_SERVICE_TYPES: ServiceType[] = ['claude', 'openai', 'responses', 'gemini']

export interface ExistingQuickAddChannelMatch {
  channel: Channel
  inputBaseUrl: string
  existingBaseUrl: string
}

function normalizeQuickAddURLIdentity(rawUrl: string): string {
  const hasHash = rawUrl.endsWith('#')
  const withoutHash = hasHash ? rawUrl.slice(0, -1) : rawUrl
  try {
    const parsed = new URL(withoutHash)
    parsed.protocol = parsed.protocol.toLowerCase()
    parsed.hostname = parsed.hostname.toLowerCase()
    parsed.hash = ''
    const normalized = parsed.toString().replace(/\/+$/, '')
    return hasHash ? `${normalized}#` : normalized
  } catch {
    return withoutHash.trim().replace(/\/+$/, '') + (hasHash ? '#' : '')
  }
}

function equivalentQuickAddURLIdentities(rawUrl: string): Set<string> {
  const identities = new Set<string>()
  for (const serviceType of QUICK_ADD_URL_SERVICE_TYPES) {
    const canonical = canonicalBaseUrl(rawUrl, serviceType)
    if (canonical) identities.add(normalizeQuickAddURLIdentity(canonical))
  }
  return identities
}

export function findExistingQuickAddChannel(
  inputBaseUrls: string[],
  existingChannels: Channel[]
): ExistingQuickAddChannelMatch | null {
  const inputs = inputBaseUrls
    .map(inputBaseUrl => ({ inputBaseUrl, identities: equivalentQuickAddURLIdentities(inputBaseUrl) }))
    .filter(item => item.identities.size > 0)

  for (const channel of existingChannels) {
    const channelBaseUrls = Array.from(new Set([channel.baseUrl, ...(channel.baseUrls ?? [])].filter(Boolean)))
    for (const existingBaseUrl of channelBaseUrls) {
      const existingIdentities = equivalentQuickAddURLIdentities(existingBaseUrl)
      for (const input of inputs) {
        if ([...input.identities].some(identity => existingIdentities.has(identity))) {
          return { channel, inputBaseUrl: input.inputBaseUrl, existingBaseUrl }
        }
      }
    }
  }
  return null
}
