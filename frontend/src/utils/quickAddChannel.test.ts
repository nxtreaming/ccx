import { describe, expect, it } from 'vitest'

import {
  buildQuickAddChannelName,
  defaultQuickAddServiceType,
  normalizeDiscoveredChannelKind,
  supportsQuickAddProtocolDiscovery
} from './quickAddChannel'

describe('buildQuickAddChannelName', () => {
  it('省略域名前导 www 并保留其余主机名', () => {
    expect(buildQuickAddChannelName('https://www.fastaitoken.com/v1', 'ivpp0p')).toBe('fastaitoken-com-ivpp0p')
  })

  it('不误删主机名中间的 www', () => {
    expect(buildQuickAddChannelName('https://api.www-example.com', 'abc123')).toBe('api-www-example-com-abc123')
  })

  it('无效地址回退到通用名称', () => {
    expect(buildQuickAddChannelName('not a url', 'abc123')).toBe('channel-abc123')
  })
})

describe('quick add protocol discovery', () => {
  it('仅对四类 LLM 协议执行发现', () => {
    expect(supportsQuickAddProtocolDiscovery('messages')).toBe(true)
    expect(supportsQuickAddProtocolDiscovery('responses')).toBe(true)
    expect(supportsQuickAddProtocolDiscovery('images')).toBe(false)
    expect(supportsQuickAddProtocolDiscovery('vectors')).toBe(false)
  })

  it('按渠道类型提供探测所需的默认 serviceType', () => {
    expect(defaultQuickAddServiceType('messages')).toBe('claude')
    expect(defaultQuickAddServiceType('responses')).toBe('responses')
  })

  it('只接受发现接口支持的协议类型', () => {
    expect(normalizeDiscoveredChannelKind('responses')).toBe('responses')
    expect(normalizeDiscoveredChannelKind('images')).toBeNull()
    expect(normalizeDiscoveredChannelKind('')).toBeNull()
  })
})
