import { describe, expect, it } from 'vitest'

import type { EndpointDetailItem } from '../services/api-types'
import { selectMiniMaxTokenPlanEndpoint, sha256KeyHash } from './minimaxEndpointUsage'

const endpoint = (overrides: Partial<EndpointDetailItem>): EndpointDetailItem => ({
  endpointUid: 'endpoint-default',
  channelUid: 'channel-minimax',
  channelKind: 'messages',
  baseUrl: 'https://api.minimax.io',
  keyHash: 'hash-default',
  healthState: 'healthy',
  healthConfidence: 1,
  consecutiveFail: 0,
  tokenPlanUsageSupported: true,
  ...overrides,
})

describe('MiniMax endpoint usage matching', () => {
  it('生成与后端一致的 SHA-256 前 16 位', async () => {
    await expect(sha256KeyHash('sk-test')).resolves.toBe('f3abf2a6cc4f0098')
  })

  it('优先使用真实 keyHash，并兼容旧后端的掩码字段', () => {
    const exact = endpoint({ endpointUid: 'exact', keyHash: 'real-hash', keyMask: 'sk-rea***key' })
    const legacy = endpoint({ endpointUid: 'legacy', keyHash: 'sk-rea***key' })

    expect(selectMiniMaxTokenPlanEndpoint([legacy, exact], 'real-hash', 'sk-rea***key')?.endpointUid).toBe('exact')
    expect(selectMiniMaxTokenPlanEndpoint([legacy], 'real-hash', 'sk-rea***key')?.endpointUid).toBe('legacy')
  })

  it('同一 Key 存在多个 BaseURL 时稳定选择有用量且更新较新的 endpoint', () => {
    const older = endpoint({
      endpointUid: 'endpoint-b',
      keyHash: 'real-hash',
      miniMaxTokenPlanUsage: { models: [{
        modelName: 'MiniMax-M3',
        currentIntervalUsageCount: 1,
        currentIntervalTotalCount: 10,
        currentIntervalRemainingPercent: 90,
        currentWeeklyUsageCount: 1,
        currentWeeklyTotalCount: 100,
        currentWeeklyRemainingPercent: 99,
        remainsTimeMs: 60_000,
      }], fetchedAt: '2026-07-21T08:00:00Z', sourceUrl: 'https://api.minimax.io' },
    })
    const newer = endpoint({
      ...older,
      endpointUid: 'endpoint-a',
      miniMaxTokenPlanUsage: { ...older.miniMaxTokenPlanUsage!, fetchedAt: '2026-07-21T09:00:00Z' },
    })

    expect(selectMiniMaxTokenPlanEndpoint([older, newer], 'real-hash', '')?.endpointUid).toBe('endpoint-a')
  })
})
