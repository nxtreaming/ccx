import type { Channel } from '@/services/api'

export const VOLCENGINE_AGENT_PLAN_CONSOLE_URL = 'https://console.volcengine.com/ark/region:cn-beijing/subscription/agent-plan'
export const VOLCENGINE_CODING_PLAN_CONSOLE_URL = 'https://console.volcengine.com/ark/region:cn-beijing/subscription/coding-plan'
export const COMPSHARE_CONSOLE_URL = 'https://console.compshare.cn/light-gpu/model-manage'

export type ChannelWebsiteKind = 'custom' | 'agent_plan' | 'coding_plan' | 'provider_console' | 'upstream'

export interface ChannelWebsiteLink {
  kind: ChannelWebsiteKind
  url: string
}

type WebsiteChannel = Pick<Channel, 'website' | 'providerId' | 'baseUrl' | 'baseUrls' | 'apiKeyConfigs'>

const VOLCENGINE_CONSOLE_URLS = {
  agent_plan: VOLCENGINE_AGENT_PLAN_CONSOLE_URL,
  coding_plan: VOLCENGINE_CODING_PLAN_CONSOLE_URL,
} as const

export function getVolcenginePlanConsoleURL(plan?: string): string | undefined {
  const normalized = plan?.trim().toLowerCase()
  if (normalized === 'agent_plan' || normalized === 'coding_plan') {
    return VOLCENGINE_CONSOLE_URLS[normalized]
  }
  return undefined
}

const normalizeURL = (value?: string): string => value?.trim() ?? ''

const volcenginePlanFromEndpoint = (value: string): 'agent_plan' | 'coding_plan' | null => {
  try {
    const pathname = new URL(value).pathname.toLowerCase().replace(/\/+$/, '')
    if (pathname === '/api/coding' || pathname.startsWith('/api/coding/')) return 'coding_plan'
    if (pathname === '/api/plan' || pathname.startsWith('/api/plan/')) return 'agent_plan'
  } catch {
    return null
  }
  return null
}

const knownVolcengineConsoleKind = (value: string): 'agent_plan' | 'coding_plan' | null => {
  const normalized = normalizeURL(value).replace(/\/+$/, '')
  if (normalized === VOLCENGINE_AGENT_PLAN_CONSOLE_URL) return 'agent_plan'
  if (normalized === VOLCENGINE_CODING_PLAN_CONSOLE_URL) return 'coding_plan'
  return null
}

const isKnownProviderConsole = (value: string): boolean => {
  const normalized = normalizeURL(value).replace(/\/+$/, '')
  return knownVolcengineConsoleKind(normalized) !== null || normalized === COMPSHARE_CONSOLE_URL
}

export function getVolcenginePlanWebsiteLinks(channel: WebsiteChannel): ChannelWebsiteLink[] {
  const providerId = channel.providerId?.trim().toLowerCase()
  if (providerId !== 'volcengine' && providerId !== 'volc-ark') return []

  const endpointURLs = [
    ...(channel.baseUrls ?? []),
    channel.baseUrl,
    ...(channel.apiKeyConfigs ?? []).map(config => config.baseUrl ?? ''),
  ]
  const plans = new Set<'agent_plan' | 'coding_plan'>()
  for (const endpointURL of endpointURLs) {
    const plan = volcenginePlanFromEndpoint(normalizeURL(endpointURL))
    if (plan) plans.add(plan)
  }

  const websitePlan = knownVolcengineConsoleKind(channel.website ?? '')
  if (websitePlan) plans.add(websitePlan)

  return (['agent_plan', 'coding_plan'] as const)
    .filter(plan => plans.has(plan))
    .map(plan => ({ kind: plan, url: VOLCENGINE_CONSOLE_URLS[plan] }))
}

export function getManagedProviderWebsiteLinks(channel: WebsiteChannel): ChannelWebsiteLink[] {
  const volcengineLinks = getVolcenginePlanWebsiteLinks(channel)
  if (volcengineLinks.length > 0) return volcengineLinks

  if (channel.providerId?.trim().toLowerCase() === 'compshare') {
    return [{ kind: 'provider_console', url: COMPSHARE_CONSOLE_URL }]
  }
  return []
}

export function getChannelWebsiteLinks(channel: WebsiteChannel): ChannelWebsiteLink[] {
  const managedProviderLinks = getManagedProviderWebsiteLinks(channel)
  const customWebsite = normalizeURL(channel.website)
  if (customWebsite && !isKnownProviderConsole(customWebsite)) {
    return [{ kind: 'custom', url: customWebsite }]
  }
  if (managedProviderLinks.length > 0) return managedProviderLinks
  if (customWebsite) return [{ kind: 'custom', url: customWebsite }]

  const upstreamURL = normalizeURL(channel.baseUrl) || normalizeURL(channel.baseUrls?.[0])
  if (!upstreamURL) return []
  try {
    const url = new URL(upstreamURL)
    return [{ kind: 'upstream', url: `${url.protocol}//${url.host}` }]
  } catch {
    return [{ kind: 'upstream', url: upstreamURL }]
  }
}
