import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ConversationDashboard from './ConversationDashboard.vue'
import type { ConversationChannelInfo, ConversationInfo, SequenceOverrideInfo } from '@/services/admin-api'

const status = ref({ running: true, starting: false })
const isConsoleConversationsActive = ref(false)
const conversations = ref<ConversationInfo[]>([])
const channelsByKind = ref<Record<string, ConversationChannelInfo[]>>({})
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const loading = ref(false)
const error = ref('')
const fetchConversations = vi.fn().mockResolvedValue(undefined)
const setOverride = vi.fn().mockResolvedValue(undefined)
const removeOverride = vi.fn().mockResolvedValue(undefined)
const apiGet = vi.fn().mockResolvedValue({ overrideTtlMinutes: 0 })
const apiPut = vi.fn().mockResolvedValue(undefined)

vi.mock('@/composables/useStatus', () => ({
  useStatus: () => ({ status }),
}))

vi.mock('@/composables/useDesktopActivity', () => ({
  useDesktopActivity: () => ({ isConsoleConversationsActive }),
}))

vi.mock('@/composables/useConversations', () => ({
  useConversations: () => ({
    conversations,
    channelsByKind,
    overrides,
    loading,
    error,
    fetchConversations,
    setOverride,
    removeOverride,
  }),
}))

vi.mock('@/composables/useAdminApi', () => ({
  useAdminApi: () => ({
    get: apiGet,
    put: apiPut,
  }),
}))

vi.mock('@/composables/useLanguage', () => ({
  useLanguage: () => ({
    t: (key: string, params?: Record<string, string>) => formatText(key, params),
    tf: (_key: string, fallback: string, params?: Record<string, string>) => formatText(fallback, params),
  }),
}))

vi.mock('@/components/ui/alert', () => ({
  Alert: defineComponent({
    props: {
      variant: { type: String, default: 'default' },
    },
    setup(_props, { slots }) {
      return () => h('div', { role: 'alert' }, slots.default?.())
    },
  }),
}))

vi.mock('@/components/ui/input', () => ({
  Input: defineComponent({
    props: {
      modelValue: { type: String, default: '' },
      placeholder: { type: String, default: '' },
    },
    emits: ['update:modelValue'],
    setup(props, { emit, attrs }) {
      return () =>
        h('input', {
          ...attrs,
          value: props.modelValue,
          placeholder: props.placeholder,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        })
    },
  }),
}))

vi.mock('@/components/ui/select', () => ({
  Select: defineComponent({
    props: {
      modelValue: { type: String, default: '' },
    },
    emits: ['update:modelValue'],
    setup(_props, { slots }) {
      return () => h('div', { 'data-testid': 'select-root' }, slots.default?.())
    },
  }),
  SelectContent: defineComponent({
    setup(_props, { slots }) {
      return () => h('div', { 'data-testid': 'select-content' }, slots.default?.())
    },
  }),
  SelectItem: defineComponent({
    props: {
      value: { type: String, required: true },
    },
    setup(props, { slots }) {
      return () => h('button', { type: 'button', 'data-value': props.value }, slots.default?.())
    },
  }),
  SelectTrigger: defineComponent({
    setup(_props, { slots }) {
      return () => h('div', { 'data-testid': 'select-trigger' }, slots.default?.())
    },
  }),
  SelectValue: defineComponent({
    props: {
      placeholder: { type: String, default: '' },
    },
    setup(props) {
      return () => h('span', props.placeholder)
    },
  }),
}))

vi.mock('@/components/ui/skeleton', () => ({
  Skeleton: defineComponent({
    setup() {
      return () => h('div', { 'data-testid': 'skeleton' })
    },
  }),
}))

vi.mock('./ConversationCard.vue', () => ({
  default: defineComponent({
    props: {
      conversation: { type: Object, required: true },
      subagents: { type: Array, default: () => [] },
      subagentSummary: { type: Object, default: undefined },
      override: { type: Object, default: undefined },
      availableChannels: { type: Array, default: () => [] },
      expanded: { type: Boolean, default: false },
      nowMs: { type: Number, required: true },
      relatedParentTitle: { type: String, default: '' },
    },
    emits: ['navigateConversation'],
    setup(props, { emit }) {
      return () => {
        const conversation = props.conversation as ConversationInfo
        const children: any[] = [
          h('span', { 'data-testid': 'conversation-card-title' }, conversation.title || conversation.userId),
        ]
        for (const subagent of props.subagents as ConversationInfo[]) {
          children.push(h('span', { 'data-testid': `subagent-${subagent.id}` }, subagent.title || subagent.userId))
        }
        if (conversation.parentConversationId) {
          children.push(
            h('button', {
              type: 'button',
              'data-testid': `navigate-parent-${conversation.id}`,
              onClick: () => emit('navigateConversation', conversation.parentConversationId),
            }, '主对话'),
          )
        }
        return h(
          'article',
          {
            'data-testid': 'conversation-card',
            'data-id': conversation.id,
            'data-kind': conversation.kind,
            'data-expanded': String(props.expanded),
          },
          children,
        )
      }
    },
  }),
}))

describe('ConversationDashboard', () => {
  let root: HTMLDivElement
  let app: ReturnType<typeof createApp> | undefined
  let errors: unknown[]

  beforeEach(() => {
    status.value = { running: true, starting: false }
    isConsoleConversationsActive.value = false
    conversations.value = []
    channelsByKind.value = {}
    overrides.value = {}
    loading.value = false
    error.value = ''
    fetchConversations.mockClear()
    setOverride.mockClear()
    removeOverride.mockClear()
    apiGet.mockResolvedValue({ overrideTtlMinutes: 0 })
    apiPut.mockResolvedValue(undefined)
    root = document.createElement('div')
    document.body.append(root)
    errors = []
    window.addEventListener('unhandledrejection', captureUnhandledRejection)
  })

  afterEach(() => {
    window.removeEventListener('unhandledrejection', captureUnhandledRejection)
    app?.unmount()
    app = undefined
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  function captureUnhandledRejection(event: PromiseRejectionEvent) {
    errors.push(event.reason)
  }

  function mountDashboard() {
    const vueErrors: unknown[] = []
    app = createApp(ConversationDashboard)
    app.config.errorHandler = error => vueErrors.push(error)
    app.mount(root)
    return { vueErrors }
  }

  function createConversation(overrides: Partial<ConversationInfo>): ConversationInfo {
    return {
      id: overrides.id ?? `conv-${Math.random().toString(36).slice(2, 8)}`,
      kind: overrides.kind ?? 'messages',
      userId: overrides.userId ?? 'user-1',
      rawUserId: overrides.rawUserId,
      title: overrides.title,
      currentChannel: overrides.currentChannel ?? 1,
      status: overrides.status ?? 'active',
      models: overrides.models ?? ['gpt-4.1'],
      lastModel: overrides.lastModel ?? 'gpt-4.1',
      requestCount: overrides.requestCount ?? 1,
      channelName: overrides.channelName ?? 'Channel 1',
      lastRequestId: overrides.lastRequestId ?? 'req-1',
      createdAt: overrides.createdAt ?? '2026-06-23T08:00:00.000Z',
      lastActiveAt: overrides.lastActiveAt ?? '2026-06-23T08:00:00.000Z',
      parentThreadId: overrides.parentThreadId,
      parentConversationId: overrides.parentConversationId,
      childConversationIds: overrides.childConversationIds,
      hasSubagents: overrides.hasSubagents,
      subagentCount: overrides.subagentCount,
      mainChannel: overrides.mainChannel,
      subagentChannel: overrides.subagentChannel,
    }
  }

  function getVisibleTitles(columnKey: 'working' | 'idle') {
    const column = root.querySelector(`[data-testid="cockpit-column-${columnKey}"]`)
    expect(column).toBeTruthy()
    return [...column!.querySelectorAll('[data-testid="conversation-card"]')]
      .map(node => node.querySelector('[data-testid="conversation-card-title"]')?.textContent?.trim() || '')
  }

  function getAllColumnCards() {
    return [...root.querySelectorAll('[data-testid^="cockpit-column-"] [data-testid="conversation-card"]')]
      .map(node => node.querySelector('[data-testid="conversation-card-title"]')?.textContent?.trim() || '')
  }

  function clickButton(text: string) {
    const button = [...root.querySelectorAll('button')]
      .find(node => node.textContent?.trim() === text)
    expect(button).toBeTruthy()
    ;(button as HTMLButtonElement).click()
  }

  function setSearch(value: string) {
    const input = root.querySelector('input')
    expect(input).toBeTruthy()
    ;(input as HTMLInputElement).value = value
    input!.dispatchEvent(new Event('input', { bubbles: true }))
  }

  it('groups conversations into task-board columns', async () => {
    conversations.value = [
      createConversation({
        id: 'root',
        title: 'Root Conversation',
        kind: 'messages',
        status: 'idle',
        childConversationIds: ['subagents', 'streaming-subagent'],
        lastActiveAt: '2026-06-23T10:00:00.000Z',
      }),
      createConversation({
        id: 'subagents',
        title: 'Subagent One',
        kind: 'chat',
        status: 'active',
        hasSubagents: true,
        parentConversationId: 'root',
        lastActiveAt: '2026-06-23T09:00:00.000Z',
      }),
      createConversation({
        id: 'streaming-subagent',
        title: 'Streaming Subagent',
        kind: 'responses',
        status: 'streaming',
        hasSubagents: true,
        parentConversationId: 'root',
        lastActiveAt: '2026-06-23T08:30:00.000Z',
      }),
      createConversation({
        id: 'active',
        title: 'Active One',
        kind: 'images',
        status: 'active',
        lastActiveAt: '2026-06-23T08:00:00.000Z',
      }),
      createConversation({
        id: 'idle',
        title: 'Idle One',
        kind: 'gemini',
        status: 'idle',
        lastActiveAt: '2026-06-23T07:00:00.000Z',
      }),
    ]

    const { vueErrors } = mountDashboard()
    await nextTick()

    expect(getVisibleTitles('working')).toEqual(['Root Conversation', 'Active One'])
    expect(getVisibleTitles('idle')).toEqual(['Idle One'])
    expect(root.textContent).toContain('Subagent One')
    expect(root.textContent).toContain('Streaming Subagent')
    expect(vueErrors).toEqual([])
    expect(errors).toEqual([])
  })

  it('filters conversations by kind and search query', async () => {
    conversations.value = [
      createConversation({
        id: 'alpha',
        title: 'Alpha Stream',
        kind: 'messages',
        status: 'streaming',
        lastActiveAt: '2026-06-23T10:00:00.000Z',
      }),
      createConversation({
        id: 'beta',
        title: 'Beta Chat',
        kind: 'chat',
        status: 'active',
        lastActiveAt: '2026-06-23T09:00:00.000Z',
      }),
    ]

    const { vueErrors } = mountDashboard()
    await nextTick()

    expect(getAllColumnCards()).toEqual(['Alpha Stream', 'Beta Chat'])

    clickButton('MESSAGES')
    await nextTick()
    expect(getAllColumnCards()).toEqual(['Alpha Stream'])

    clickButton('ALL')
    await nextTick()
    setSearch('Beta')
    await nextTick()
    expect(getAllColumnCards()).toEqual(['Beta Chat'])

    setSearch('Nope')
    await nextTick()
    expect(root.textContent).toContain('cockpit.noMatches')
    expect(root.querySelectorAll('[data-testid^="cockpit-column-"]').length).toBe(0)
    expect(vueErrors).toEqual([])
    expect(errors).toEqual([])
  })

  it('shows the empty state when there are no conversations', async () => {
    conversations.value = []

    const { vueErrors } = mountDashboard()
    await nextTick()

    expect(root.textContent).toContain('cockpit.empty')
    expect(root.querySelectorAll('[data-testid^="cockpit-column-"]').length).toBe(0)
    expect(vueErrors).toEqual([])
    expect(errors).toEqual([])
  })

  it('keeps parent conversation visible when filtering by a subagent kind', async () => {
    const scrollIntoView = vi.fn()
    Object.defineProperty(Element.prototype, 'scrollIntoView', {
      configurable: true,
      value: scrollIntoView,
    })
    conversations.value = [
      createConversation({
        id: 'parent',
        title: 'Parent Conversation',
        kind: 'messages',
        childConversationIds: ['child'],
        lastActiveAt: '2026-06-23T10:00:00.000Z',
      }),
      createConversation({
        id: 'child',
        title: 'Child Agent',
        kind: 'responses',
        parentConversationId: 'parent',
        parentThreadId: 'parent-thread',
        lastActiveAt: '2026-06-23T11:00:00.000Z',
      }),
    ]

    const { vueErrors } = mountDashboard()
    await nextTick()

    clickButton('RESPONSES')
    await nextTick()
    expect(getAllColumnCards()).toEqual(['Parent Conversation'])
    expect(root.textContent).toContain('Child Agent')

    const parent = root.querySelector('[data-testid="conversation-card"][data-id="parent"]')
    expect(parent).toBeTruthy()
    expect(scrollIntoView).not.toHaveBeenCalled()
    expect(vueErrors).toEqual([])
    expect(errors).toEqual([])
  })
})

function formatText(template: string, params?: Record<string, string>) {
  if (!params) return template
  return Object.entries(params).reduce((acc, [key, value]) => acc.replaceAll(`{${key}}`, value), template)
}
