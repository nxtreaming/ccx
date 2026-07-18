import type { DisabledKeyInfo } from '@/services/admin-api'

export interface ChannelApiKeyRow {
  key: string
  activeIndex: number
  disabled?: DisabledKeyInfo
}

export function buildChannelApiKeyRows(
  apiKeys: string[] | null | undefined = [],
  disabledKeys: DisabledKeyInfo[] | null | undefined = [],
): ChannelApiKeyRow[] {
  const activeKeys = apiKeys ?? []
  const disabledItems = disabledKeys ?? []
  const disabledByKey = new Map(disabledItems.filter(item => item.key).map(item => [item.key, item]))
  const seen = new Set<string>()
  const rows: ChannelApiKeyRow[] = []

  activeKeys.forEach((key, activeIndex) => {
    if (!key || seen.has(key)) return
    seen.add(key)
    rows.push({ key, activeIndex, disabled: disabledByKey.get(key) })
  })

  for (const disabled of disabledItems) {
    if (!disabled.key || seen.has(disabled.key)) continue
    seen.add(disabled.key)
    rows.push({ key: disabled.key, activeIndex: -1, disabled })
  }

  return rows
}

type ChannelKeyState = {
  apiKeys?: string[] | null
  disabledApiKeys?: DisabledKeyInfo[] | null
}

export function availableChannelApiKeyCount(channel: ChannelKeyState): number {
  return buildChannelApiKeyRows(channel.apiKeys, channel.disabledApiKeys).filter(row => !row.disabled).length
}

export function disabledChannelApiKeyCount(channel: ChannelKeyState): number {
  return buildChannelApiKeyRows(channel.apiKeys, channel.disabledApiKeys).filter(row => !!row.disabled).length
}
