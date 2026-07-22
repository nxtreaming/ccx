// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import ProtocolModelAvailability from './ProtocolModelAvailability.vue'

vi.mock('../../i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, number>) => {
      if (params?.count !== undefined) return `${key}:${params.count}`
      if (params?.available !== undefined) return `${key}:${params.available}/${params.total}`
      return key
    },
  }),
}))

const passthroughStub = defineComponent({
  template: '<span><slot /></span>',
})

describe('ProtocolModelAvailability', () => {
  it('按协议分组展示各自的可用模型', () => {
    const wrapper = mount(ProtocolModelAvailability, {
      props: {
        routes: [
          {
            kind: 'messages', index: 0, name: 'fastaitoken-claude', serviceType: 'claude',
            supportedModels: ['gpt-5.6-terra', 'gpt-5.6-sol', 'gpt-5.6-sol'],
          },
          {
            kind: 'chat', index: 0, name: 'fastaitoken-chat', serviceType: 'openai',
            supportedModels: ['gpt-5.6-sol'],
          },
          {
            kind: 'responses', index: 0, name: 'fastaitoken-codex', serviceType: 'responses',
            supportedModels: ['codex-auto-review'],
          },
        ],
      },
      global: {
        stubs: {
          VChip: passthroughStub,
          VIcon: passthroughStub,
        },
      },
    })

    const messages = wrapper.get('[data-kind="messages"]')
    const chat = wrapper.get('[data-kind="chat"]')
    const responses = wrapper.get('[data-kind="responses"]')

    expect(messages.text()).toContain('/v1/messages')
    expect(messages.text()).toContain('gpt-5.6-sol')
    expect(messages.text()).toContain('gpt-5.6-terra')
    expect(messages.text().match(/gpt-5\.6-sol/g)).toHaveLength(1)
    expect(chat.text()).toContain('/v1/chat/completions')
    expect(chat.text()).not.toContain('gpt-5.6-terra')
    expect(responses.text()).toContain('/v1/responses')
    expect(responses.text()).toContain('codex-auto-review')
  })

  it('区分未记录模型范围与协议不可用', () => {
    const wrapper = mount(ProtocolModelAvailability, {
      props: {
        routes: [{ kind: 'gemini', index: 0, name: 'gemini', serviceType: 'gemini' }],
      },
      global: {
        stubs: {
          VChip: passthroughStub,
          VIcon: passthroughStub,
        },
      },
    })

    const gemini = wrapper.get('[data-kind="gemini"]')
    expect(gemini.text()).toContain('channelEditor.protocolModels.empty')
    expect(gemini.text()).not.toContain('channelEditor.protocolModels.count:0')
  })

  it('已发现到空模型清单时不回退配置白名单', () => {
    const wrapper = mount(ProtocolModelAvailability, {
      props: {
        routes: [{
          kind: 'responses', upstreamKind: 'chat', index: 0, name: 'chat-through-responses', serviceType: 'openai',
          supportedModels: ['configured-model'], modelInventoryKnown: true, discoveredModels: [],
          modelBindings: [{ credentialUid: 'cred-empty', keyMask: 'sk-e***001', models: [] }],
        }],
      },
      global: {
        stubs: {
          VChip: passthroughStub,
          VIcon: passthroughStub,
        },
      },
    })

    const chat = wrapper.get('[data-kind="chat"]')
    expect(chat.text()).toContain('/v1/chat/completions')
    expect(chat.text()).toContain('channelEditor.protocolModels.count:0')
    expect(chat.text()).not.toContain('configured-model')
  })

  it('优先展示 endpoint profile 模型并标记 Key 差异', () => {
    const wrapper = mount(ProtocolModelAvailability, {
      props: {
        routes: [{
          kind: 'messages', index: 0, name: 'volcengine-claude', serviceType: 'claude',
          supportedModels: ['configured-model'],
          discoveredModels: ['actual-model'],
          modelBindings: [
            { credentialUid: 'cred-a', keyMask: 'ark-a***001', models: ['actual-model'] },
            { credentialUid: 'cred-b', keyMask: 'ark-b***002', models: ['other-model'] },
          ],
        }],
      },
      global: {
        stubs: {
          VChip: passthroughStub,
          VIcon: passthroughStub,
        },
      },
    })

    const messages = wrapper.get('[data-kind="messages"]')
    expect(messages.text()).toContain('actual-model')
    expect(messages.text()).not.toContain('configured-model')
    expect(messages.text()).toContain('channelEditor.protocolModels.diffCount:2')
    expect(messages.text()).toContain('ark-a***001')
    expect(messages.text()).toContain('ark-b***002')
    expect(messages.text()).toContain('channelEditor.protocolModels.coverage:1/2')
    // 全量模型列表折叠为一个 details，差异区直接列出缺失关系
    expect(messages.findAll('details')).toHaveLength(1)
    expect(messages.findAll('details')[0].text()).toContain('actual-model')
    expect(messages.findAll('details')[0].text()).toContain('other-model')
  })

  it('多 Key 模型一致时只显示一致提示，不展示差异区', () => {
    const wrapper = mount(ProtocolModelAvailability, {
      props: {
        routes: [{
          kind: 'messages', index: 0, name: 'multi-key', serviceType: 'claude',
          discoveredModels: ['model-a', 'model-b'],
          modelBindings: [
            { credentialUid: 'cred-a', keyMask: 'sk-a***001', models: ['model-a', 'model-b'] },
            { credentialUid: 'cred-b', keyMask: 'sk-b***002', models: ['model-a', 'model-b'] },
          ],
        }],
      },
      global: {
        stubs: {
          VChip: passthroughStub,
          VIcon: passthroughStub,
        },
      },
    })

    const messages = wrapper.get('[data-kind="messages"]')
    expect(messages.text()).toContain('channelEditor.protocolModels.consistent:2')
    expect(messages.text()).not.toContain('channelEditor.protocolModels.diffCount')
    expect(messages.find('.protocol-model-route__diffs').exists()).toBe(false)
  })

  it('展示模型清单的发现时间、来源和说明', () => {
    const wrapper = mount(ProtocolModelAvailability, {
      props: {
        routes: [{
          kind: 'messages', index: 0, name: 'volcengine-claude', serviceType: 'claude',
          modelInventoryKnown: true,
          discoveredModels: ['glm-5.2'],
          modelsDiscoveredAt: '2026-07-22T00:42:12Z',
          modelDiscoverySource: 'control_plane',
          modelDiscoveryMessage: '火山管控面 Coding Plan 模型清单',
        }],
      },
      global: {
        stubs: {
          VChip: passthroughStub,
          VIcon: passthroughStub,
        },
      },
    })

    const messages = wrapper.get('[data-kind="messages"]')
    expect(messages.text()).toContain('channelEditor.protocolModels.lastDiscovered')
    expect(messages.text()).toContain('channelEditor.protocolModels.source.controlPlane')
    expect(messages.text()).toContain('火山管控面 Coding Plan 模型清单')
  })
})
