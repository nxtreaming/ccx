<script setup lang="ts">
import { ref } from 'vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import TabSwitcher from '@/components/layout/TabSwitcher.vue'
import StatusTab from '@/components/status/StatusTab.vue'
import AgentTab from '@/components/agent/AgentTab.vue'
import EnvTab from '@/components/env/EnvTab.vue'
import WebUITab from '@/components/webui/WebUITab.vue'
import { useStatus } from '@/composables/useStatus'
import { useWailsEvents } from '@/composables/useWailsEvents'

const activeTab = ref<'status' | 'agent' | 'env' | 'web'>('status')
const { status, actionError, syncStatus } = useStatus()

useWailsEvents(activeTab, actionError, syncStatus)

const switchToWeb = () => {
  activeTab.value = 'web'
}
</script>

<template>
  <div class="flex flex-col min-h-screen bg-background">
    <AppHeader />
    <main class="flex-1 flex flex-col px-6 py-5 overflow-hidden">
      <TabSwitcher v-model="activeTab">
        <template #status>
          <StatusTab @switch-to-web="switchToWeb" />
        </template>
        <template #agent>
          <AgentTab />
        </template>
        <template #env>
          <EnvTab />
        </template>
        <template #web>
          <WebUITab :status="status" :loading="false" />
        </template>
      </TabSwitcher>
    </main>
  </div>
</template>
