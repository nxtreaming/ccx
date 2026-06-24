import { isValidUrl } from './quick-input-parser'
import { buildExpectedRequestUrl, type ServiceType } from './base-url-semantics'

export type ChannelType = 'messages' | 'chat' | 'responses' | 'gemini' | 'images'

export interface ExpectedRequestUrlItem {
  baseUrl: string
  expectedUrl: string
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
  if (channelType === 'images') {
    endpoint = '/images/generations'
  } else if (channelType === 'responses') {
    if (serviceType === 'responses' || serviceType === 'copilot') endpoint = '/responses'
    else if (serviceType === 'claude') endpoint = '/messages'
    else if (serviceType === 'gemini') endpoint = '/models/{model}:generateContent'
    else endpoint = '/chat/completions'
  } else {
    if (serviceType === 'claude') endpoint = '/messages'
    else if (serviceType === 'gemini') endpoint = '/models/{model}:generateContent'
    else if (serviceType === 'responses' || serviceType === 'copilot') endpoint = '/responses'
    else endpoint = '/chat/completions'
  }

  return urls
    .filter(url => url && isValidUrl(url.replace(/#$/, '')))
    .map(rawUrl => ({
      baseUrl: rawUrl,
      expectedUrl: buildExpectedRequestUrl(serviceType, endpoint, rawUrl)
    }))
}
