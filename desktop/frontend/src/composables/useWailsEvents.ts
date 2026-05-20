import { onMounted, onBeforeUnmount, type Ref } from 'vue'
import { Events } from '@wailsio/runtime'

export function useWailsEvents(
  activeTab: Ref<'status' | 'agent' | 'env' | 'web'>,
  actionError: Ref<string>,
  syncStatus: () => Promise<void>,
) {
  let unsubscribeTab: (() => void) | undefined
  let unsubscribeTrayError: (() => void) | undefined

  onMounted(() => {
    unsubscribeTab = Events.On('desktop:show-tab', (event: { data: string }) => {
      const validTabs = ['status', 'agent', 'env', 'web'] as const
      if (validTabs.includes(event.data as typeof validTabs[number])) {
        activeTab.value = event.data as typeof validTabs[number]
      }
    })
    unsubscribeTrayError = Events.On('desktop:tray-error', (event: { data: string }) => {
      actionError.value = event.data
      void syncStatus()
    })
  })

  onBeforeUnmount(() => {
    unsubscribeTab?.()
    unsubscribeTrayError?.()
  })
}
