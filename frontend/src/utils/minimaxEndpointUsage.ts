import type { EndpointDetailItem } from '../services/api-types'

const hasUsableUsage = (endpoint: EndpointDetailItem): boolean =>
  !endpoint.miniMaxTokenPlanUsageError && Boolean(endpoint.miniMaxTokenPlanUsage?.models.length)

const usageTimestamp = (endpoint: EndpointDetailItem): number => {
  const value = endpoint.miniMaxTokenPlanUsage?.fetchedAt
  if (!value) return 0
  const timestamp = Date.parse(value)
  return Number.isNaN(timestamp) ? 0 : timestamp
}

const pickBestEndpoint = (endpoints: EndpointDetailItem[]): EndpointDetailItem | undefined =>
  [...endpoints].sort((left, right) => {
    const usageDifference = Number(hasUsableUsage(right)) - Number(hasUsableUsage(left))
    if (usageDifference !== 0) return usageDifference
    const timestampDifference = usageTimestamp(right) - usageTimestamp(left)
    if (timestampDifference !== 0) return timestampDifference
    return left.endpointUid.localeCompare(right.endpointUid)
  })[0]

export const sha256KeyHash = async (apiKey: string): Promise<string> => {
  if (!apiKey) return ''
  const bytes = new globalThis.TextEncoder().encode(apiKey)
  const digest = await globalThis.crypto.subtle.digest('SHA-256', bytes)
  return Array.from(new Uint8Array(digest), byte => byte.toString(16).padStart(2, '0')).join('').slice(0, 16)
}

export const selectMiniMaxTokenPlanEndpoint = (
  endpoints: EndpointDetailItem[],
  keyHash: string,
  keyMask: string,
): EndpointDetailItem | undefined => {
  const supported = endpoints.filter(endpoint => endpoint.tokenPlanUsageSupported)
  const hashMatches = keyHash ? supported.filter(endpoint => endpoint.keyHash === keyHash) : []
  if (hashMatches.length > 0) return pickBestEndpoint(hashMatches)

  // 兼容尚未升级、仍把掩码放在 keyHash 字段中的后端。
  return pickBestEndpoint(supported.filter(endpoint =>
    endpoint.keyMask === keyMask || endpoint.keyHash === keyMask,
  ))
}
