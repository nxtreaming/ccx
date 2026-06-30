import { computed, ref, type ComputedRef } from 'vue'
import { ApiService, type Channel } from '../services/api'

type ChannelType = 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
type FormLike = {
  apiKeys: string[]
}

type DisabledApiKeyOptions = {
  apiService: ApiService
  channel: ComputedRef<Channel | null | undefined>
  channelType: ComputedRef<ChannelType>
  emitError: (message: string) => void
  form: FormLike
}

export function useDisabledApiKeys(options: DisabledApiKeyOptions) {
  const restoringKey = ref('')
  const localRestoredKeys = ref(new Set<string>())

  const disabledKeys = computed(() => options.channel.value?.disabledApiKeys || [])
  const visibleDisabledKeys = computed(() =>
    (options.channel.value?.disabledApiKeys || []).filter(dk => !localRestoredKeys.value.has(dk.key))
  )

  const resetRestoredKeys = () => {
    localRestoredKeys.value = new Set<string>()
    restoringKey.value = ''
  }

  const restoreDisabledKey = async (apiKey: string) => {
    const channel = options.channel.value
    if (!channel) return
    restoringKey.value = apiKey
    try {
      const channelId = channel.index
      switch (options.channelType.value) {
        case 'chat':
          await options.apiService.restoreChatApiKey(channelId, apiKey)
          break
        case 'images':
          await options.apiService.restoreImagesApiKey(channelId, apiKey)
          break
        case 'gemini':
          await options.apiService.restoreGeminiApiKey(channelId, apiKey)
          break
        case 'responses':
          await options.apiService.restoreResponsesApiKey(channelId, apiKey)
          break
        default:
          await options.apiService.restoreApiKey(channelId, apiKey)
      }
      localRestoredKeys.value.add(apiKey)
      options.form.apiKeys.push(apiKey)
    } catch (error) {
      options.emitError(error instanceof Error ? error.message : 'Restore failed')
    } finally {
      restoringKey.value = ''
    }
  }

  return {
    restoringKey,
    localRestoredKeys,
    disabledKeys,
    visibleDisabledKeys,
    resetRestoredKeys,
    restoreDisabledKey,
  }
}
