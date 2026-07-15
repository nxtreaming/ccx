import { isValidUrl } from './quickInputParser'
import { buildExpectedRequestUrl, type ServiceType } from './baseUrlSemantics'

export type ChannelType = 'messages' | 'chat' | 'responses' | 'gemini' | 'images' | 'vectors'

export interface ExpectedRequestUrlItem {
  baseUrl: string
  expectedUrl: string
}

export type DiscoveryProtocol = 'messages' | 'chat' | 'responses' | 'gemini'

export interface DiscoveryExpectedRequestUrlItem extends ExpectedRequestUrlItem {
  protocol: DiscoveryProtocol
}

const discoveryProtocolTargets: Array<{
  protocol: DiscoveryProtocol
  channelType: DiscoveryProtocol
  serviceType: ServiceType
}> = [
  { protocol: 'messages', channelType: 'messages', serviceType: 'claude' },
  { protocol: 'chat', channelType: 'chat', serviceType: 'openai' },
  { protocol: 'responses', channelType: 'responses', serviceType: 'responses' },
  { protocol: 'gemini', channelType: 'gemini', serviceType: 'gemini' }
]

export function buildDiscoveryExpectedRequestUrls(baseUrl: string): DiscoveryExpectedRequestUrlItem[] {
  return discoveryProtocolTargets.flatMap(target =>
    buildExpectedRequestUrls(target.channelType, target.serviceType, baseUrl).map(item => ({
      ...item,
      protocol: target.protocol
    }))
  )
}

export function buildExpectedRequestUrls(
  channelType: ChannelType,
  serviceType: ServiceType,
  baseUrl?: string,
  baseUrls?: string[]
): ExpectedRequestUrlItem[] {
  if (!serviceType) return []

  const urls: string[] = []
  if (baseUrls && baseUrls.length > 0) {
    urls.push(...baseUrls)
  } else if (baseUrl) {
    urls.push(baseUrl)
  }

  if (urls.length === 0) return []

  let endpoint = ''
  if (channelType === 'vectors') {
    endpoint = '/embeddings'
  } else if (channelType === 'images') {
    endpoint = '/images/generations'
  } else if (channelType === 'responses') {
    if (serviceType === 'responses' || serviceType === 'copilot') {
      endpoint = '/responses'
    } else if (serviceType === 'claude') {
      endpoint = '/messages'
    } else if (serviceType === 'gemini') {
      endpoint = '/models/{model}:generateContent'
    } else {
      endpoint = '/chat/completions'
    }
  } else {
    if (serviceType === 'claude') {
      endpoint = '/messages'
    } else if (serviceType === 'gemini') {
      endpoint = '/models/{model}:generateContent'
    } else if (serviceType === 'responses' || serviceType === 'copilot') {
      endpoint = '/responses'
    } else {
      endpoint = '/chat/completions'
    }
  }

  return urls
    .filter(url => url && isValidUrl(url.replace(/#$/, '')))
    .map(rawUrl => ({
      baseUrl: rawUrl,
      expectedUrl: buildExpectedRequestUrl(serviceType, endpoint, rawUrl)
    }))
}
