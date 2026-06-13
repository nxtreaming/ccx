import { describe, expect, it } from 'vitest'
import { buildChannelPayload } from './channelPayload'

describe('buildChannelPayload', () => {
  it('应序列化 reasoningMapping 与渠道级 verbosity/fastMode', () => {
    const result = buildChannelPayload({
      name: '  test-channel  ',
      serviceType: 'openai',
      baseUrl: 'https://api.example.com/v1#',
      baseUrls: [],
      website: ' https://platform.openai.com ',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '  desc  ',
      apiKeys: ['sk-1', '  ', 'sk-2'],
      modelMapping: { 'gpt-5': 'gpt-5.4' },
      reasoningMapping: { 'gpt-5': 'max' },
      reasoningParamStyle: 'reasoning_effort',
      textVerbosity: 'medium',
      fastMode: true,
      customHeaders: { 'x-test': '1' },
      proxyUrl: ' http://127.0.0.1:7890 ',
      requestTimeoutMs: 15000,
      routePrefix: '',
      supportedModels: ['gpt-5'],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: '',
      historicalImageTurnLimit: 3
    })

    expect(result.name).toBe('test-channel')
    expect(result.baseUrl).toBe('https://api.example.com/v1#')
    expect(result.website).toBe('https://platform.openai.com')
    expect(result.description).toBe('desc')
    expect(result.apiKeys).toEqual(['sk-1', 'sk-2'])
    expect(result.modelMapping).toEqual({ 'gpt-5': 'gpt-5.4' })
    expect(result.reasoningMapping).toEqual({ 'gpt-5': 'max' })
    expect(result.reasoningParamStyle).toBe('reasoning_effort')
    expect(result.textVerbosity).toBe('medium')
    expect(result.fastMode).toBe(true)
    expect(result.proxyUrl).toBe('http://127.0.0.1:7890')
    expect(result.requestTimeoutMs).toBe(15000)
    expect((result as any).historicalImageTurnLimit).toBe(3)
  })

  it('应对多个 baseUrls 去重并保留 baseUrls 输出', () => {
    const result = buildChannelPayload({
      name: 'multi',
      serviceType: 'responses',
      baseUrl: '',
      baseUrls: ['https://api.example.com/v1/', 'https://api.example.com/v1#', 'https://backup.example.com/v1'],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.baseUrl).toBe('https://api.example.com')
    expect(result.baseUrls).toEqual([
      'https://api.example.com',
      'https://api.example.com/v1#',
      'https://backup.example.com'
    ])
  })

  it('应将根域名与默认版本前缀 URL 去重为最短形式', () => {
    const result = buildChannelPayload({
      name: 'multi',
      serviceType: 'openai',
      baseUrl: '',
      baseUrls: ['https://new.timefiles.online/v1', 'https://new.timefiles.online'],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.baseUrl).toBe('https://new.timefiles.online')
    expect(result.baseUrls).toBeUndefined()
  })

  it('应保留带 # 的 URL 与普通 URL 分离', () => {
    const result = buildChannelPayload({
      name: 'multi',
      serviceType: 'openai',
      baseUrl: '',
      baseUrls: ['https://new.timefiles.online/v1', 'https://new.timefiles.online#'],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.baseUrl).toBe('https://new.timefiles.online')
    expect(result.baseUrls).toEqual(['https://new.timefiles.online', 'https://new.timefiles.online#'])
  })

  it('应清空 claude 渠道不支持的高级参数', () => {
    const result = buildChannelPayload({
      name: 'claude-channel',
      serviceType: 'claude',
      baseUrl: 'https://api.anthropic.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-ant'],
      modelMapping: { opus: 'claude-3-7-sonnet' },
      reasoningMapping: { opus: 'high' },
      reasoningParamStyle: 'reasoning_effort',
      textVerbosity: 'high',
      fastMode: true,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: ['opus'],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.modelMapping).toEqual({ opus: 'claude-3-7-sonnet' })
    expect(result.reasoningMapping).toEqual({})
    expect(result.reasoningParamStyle).toBe('reasoning')
    expect(result.textVerbosity).toBe('')
    expect(result.fastMode).toBe(false)
  })

  it('应携带 autoBlacklistBalance 开关', () => {
    const result = buildChannelPayload({
      name: 'balance-guard',
      serviceType: 'responses',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: false,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.autoBlacklistBalance).toBe(false)
  })

  it('应携带 normalizeMetadataUserId 开关', () => {
    const result = buildChannelPayload({
      name: 'metadata-guard',
      serviceType: 'responses',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: false,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.normalizeMetadataUserId).toBe(false)
  })

  it('应携带 stripBillingHeader 开关', () => {
    const result = buildChannelPayload({
      name: 'claude-cch-strip',
      serviceType: 'claude',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripBillingHeader: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.stripBillingHeader).toBe(true)
  })

  it('应携带 stripEmptyTextBlocks 开关', () => {
    const result = buildChannelPayload({
      name: 'claude-strict-upstream',
      serviceType: 'claude',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: true,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.stripEmptyTextBlocks).toBe(true)
  })

  it('应携带 passbackThinkingBlocks 开关', () => {
    const result = buildChannelPayload({
      name: 'claude-thinking-passback',
      serviceType: 'claude',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: true,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.passbackThinkingBlocks).toBe(true)
  })

  it('应携带 normalizeSystemRoleToTopLevel 开关', () => {
    const result = buildChannelPayload({
      name: 'claude-system-normalize',
      serviceType: 'claude',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: true,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.normalizeSystemRoleToTopLevel).toBe(true)
  })

  it('应携带 normalizeNonstandardChatRoles 开关', () => {
    const result = buildChannelPayload({
      name: 'chat-role-guard',
      serviceType: 'openai',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      normalizeNonstandardChatRoles: true,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.normalizeNonstandardChatRoles).toBe(true)
  })

  it('空请求超时不写入 payload，继承全局配置', () => {
    const result = buildChannelPayload({
      name: 'inherit-timeout',
      serviceType: 'openai',
      baseUrl: 'https://api.example.com/v1',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      modelMapping: {},
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      requestTimeoutMs: null,
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: true,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    expect(result.requestTimeoutMs).toBeUndefined()
  })

  it('应清洗 modelMapping 中的对象值为字符串', () => {
    const result = buildChannelPayload({
      name: 'test',
      serviceType: 'claude',
      baseUrl: 'https://api.example.com',
      baseUrls: [],
      website: '',
      insecureSkipVerify: false,
      lowQuality: false,
      injectDummyThoughtSignature: false,
      stripThoughtSignature: false,
      passbackReasoningContent: false,
      passbackThinkingBlocks: false,
      description: '',
      apiKeys: ['sk-1'],
      // v-combobox 选中下拉后可能产生对象值
      modelMapping: {
        'fable': 'claude-3-5-sonnet',
        'haiku': { title: 'claude-3-5-haiku', value: 'claude-3-5-haiku' }
      } as any,
      reasoningMapping: {},
      reasoningParamStyle: 'reasoning',
      textVerbosity: '',
      fastMode: false,
      customHeaders: {},
      proxyUrl: '',
      routePrefix: '',
      supportedModels: [],
      autoBlacklistBalance: true,
      normalizeMetadataUserId: true,
      stripEmptyTextBlocks: false,
      normalizeSystemRoleToTopLevel: false,
      codexNativeToolPassthrough: false,
      codexToolCompat: false,
      stripImageGenerationTool: false,
      noVision: false,
      noVisionModels: [],
      visionFallbackModel: ''
    })

    // 确保所有 modelMapping 值都是字符串
    expect(result.modelMapping).toEqual({
      'fable': 'claude-3-5-sonnet',
      'haiku': 'claude-3-5-haiku'
    })
    expect(result.modelMapping).toBeDefined()
    expect(typeof result.modelMapping!.fable).toBe('string')
    expect(typeof result.modelMapping!.haiku).toBe('string')
  })
})
