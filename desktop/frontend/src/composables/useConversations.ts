import { ref } from 'vue'
import { useAdminApi } from '@/composables/useAdminApi'
import type {
  ConversationInfo,
  ConversationsResponse,
  ChannelSequenceEntry,
  ConversationChannelInfo,
  SequenceOverrideInfo,
} from '@/services/admin-api'

// Module-level singletons
const conversations = ref<ConversationInfo[]>([])
const total = ref(0)
const channelsByKind = ref<Record<string, ConversationChannelInfo[]>>({})
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const loading = ref(false)
const error = ref('')

export function useConversations() {
  const api = useAdminApi()

  function clearError() {
    error.value = ''
  }

  async function fetchConversations(kind?: string) {
    loading.value = true
    clearError()
    try {
      const params = kind ? `?kind=${encodeURIComponent(kind)}` : ''
      const data = await api.get<ConversationsResponse>(`/api/conversations${params}`)
      conversations.value = data.conversations
      total.value = data.total
      channelsByKind.value = data.channelsByKind || {}
      overrides.value = data.overrides
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function setOverride(conversationId: string, sequence: ChannelSequenceEntry[], duration?: number) {
    await api.post(`/api/conversations/${encodeURIComponent(conversationId)}/override`, { sequence, duration })
    await fetchConversations()
  }

  async function removeOverride(conversationId: string) {
    await api.del(`/api/conversations/${encodeURIComponent(conversationId)}/override`)
    await fetchConversations()
  }

  return {
    conversations,
    total,
    channelsByKind,
    overrides,
    loading,
    error,
    fetchConversations,
    setOverride,
    removeOverride,
  }
}
