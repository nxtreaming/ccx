import { describe, expect, it } from 'vitest'
import { normalizeAdvancedChannelOptions, supportsAdvancedChannelOptions, supportsReasoningMapping } from './channel-advanced-options'

describe('channel-advanced-options', () => {
  it('为 Claude 保留 reasoningMapping，但不保留 OpenAI 专属选项', () => {
    expect(supportsAdvancedChannelOptions('claude')).toBe(false)
    expect(supportsReasoningMapping('claude')).toBe(true)

    const result = normalizeAdvancedChannelOptions('claude', {
      reasoningMapping: { opus: 'high' },
      reasoningParamStyle: 'thinking',
      textVerbosity: 'high',
      fastMode: true
    })

    expect(result).toEqual({
      reasoningMapping: { opus: 'high' },
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false
    })
  })

  it('为不支持 reasoningMapping 的渠道清空映射', () => {
    expect(supportsReasoningMapping('gemini')).toBe(false)

    const result = normalizeAdvancedChannelOptions('gemini', {
      reasoningMapping: { gemini: 'high' },
      reasoningParamStyle: 'thinking',
      textVerbosity: 'high',
      fastMode: true
    })

    expect(result.reasoningMapping).toEqual({})
  })
})
