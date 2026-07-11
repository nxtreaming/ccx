import { useAuthStore } from '@/stores/auth'
import { API_BASE } from './api-helpers'

// ─── 类型定义 ───

/** 自动添加渠道请求 */
export interface AutoAddChannelRequest {
  name?: string
  /** provider 模板模式：带 providerId 时 baseURL 由后端按 key 前缀探测判定，无需填 baseUrls */
  providerId?: string
  baseUrls?: string[]
  apiKeys: string[]
  subscriptionUid?: string
}

/** Provider 模板 key 前缀规则 */
export interface ProviderKeyPrefixRule {
  prefix: string
  planTag: string
}

/** Provider 候选 baseURL */
export interface ProviderCandidate {
  baseUrl: string
  planTag?: string
  region?: string
  priority?: number
}

/** 官方 provider 模板 */
export interface ProviderTemplate {
  providerId: string
  displayName: string
  description?: string
  channelKind: string
  serviceType: string
  originType?: string
  originTier?: string
  keyPrefixRules?: ProviderKeyPrefixRule[]
  candidates?: ProviderCandidate[]
}

/** 自动添加渠道响应 */
export interface AutoAddChannelResponse {
  channelUid: string
  index: number
  discoveryStarted: boolean
}

/** Endpoint 发现信息 */
export interface AutoEndpointStatus {
  keyMask: string
  baseUrl: string
  modelsCount: number
  protocolOk: boolean
}

/** 发现状态信息 */
export interface AutoDiscoveryStatus {
  status: 'pending' | 'running' | 'done' | 'failed'
  startedAt?: string
  finishedAt?: string
  error?: string
  endpoints?: AutoEndpointStatus[]
}

/** 自动托管状态响应 */
export interface ChannelAutoStatusResponse {
  autoManaged: boolean
  autoManagedAt?: string
  discovery?: AutoDiscoveryStatus
}

// ─── 辅助方法 ───

function getAuthHeaders(): Record<string, string> {
  const authStore = useAuthStore()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  const apiKey = authStore.apiKey as unknown as string | null
  if (apiKey) {
    headers['x-api-key'] = apiKey
  }
  return headers
}

// ─── API 方法 ───

/**
 * 快速添加渠道（自动托管模式）
 * POST /api/{kind}/channels/auto-add
 */
export async function autoAddChannel(
  kind: string,
  request: AutoAddChannelRequest,
): Promise<AutoAddChannelResponse> {
  const url = `${API_BASE}/${kind}/channels/auto-add`
  const response = await fetch(url, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(request),
  })

  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText)
    throw new Error(`auto-add failed (${response.status}): ${text}`)
  }

  return response.json()
}

/**
 * 查询渠道自动托管状态
 * GET /api/{kind}/channels/{id}/auto-status
 */
export async function getChannelAutoStatus(
  kind: string,
  channelId: number,
): Promise<ChannelAutoStatusResponse> {
  const url = `${API_BASE}/${kind}/channels/${channelId}/auto-status`
  const response = await fetch(url, {
    method: 'GET',
    headers: getAuthHeaders(),
  })

  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText)
    throw new Error(`auto-status failed (${response.status}): ${text}`)
  }

  return response.json()
}

/**
 * 获取内置 provider 模板（模板化添加：选 provider + 输 key）
 * GET /api/channels/provider-templates
 */
export async function getProviderTemplates(): Promise<ProviderTemplate[]> {
  const url = `${API_BASE}/channels/provider-templates`
  const response = await fetch(url, {
    method: 'GET',
    headers: getAuthHeaders(),
  })

  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText)
    throw new Error(`provider-templates failed (${response.status}): ${text}`)
  }

  const data = await response.json()
  return data.providers ?? []
}
