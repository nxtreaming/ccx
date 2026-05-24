<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import StatusTab from '@/components/status/StatusTab.vue'
import AgentTab from '@/components/agent/AgentTab.vue'
import EnvTab from '@/components/env/EnvTab.vue'
import WebUITab from '@/components/webui/WebUITab.vue'
import ChannelTab from '@/components/channel/ChannelTab.vue'
import UpdateDialog from '@/components/update/UpdateDialog.vue'
import { useStatus } from '@/composables/useStatus'
import { useUpdater } from '@/composables/useUpdater'
import { useWailsEvents } from '@/composables/useWailsEvents'
import { RefreshCw } from 'lucide-vue-next'

import type { TabValue } from '@/types'

const activeTab = ref<TabValue>('status')
const { status, actionError, syncStatus } = useStatus()
useUpdater()

useWailsEvents(activeTab, actionError, syncStatus)

const switchToWeb = () => {
  activeTab.value = 'web'
}

const webTabRef = ref<InstanceType<typeof WebUITab> | null>(null)
const refreshWebUI = () => webTabRef.value?.refreshIframe()

// 切换到 Web UI 标签时自动刷新 iframe，避免 v-show 隐藏状态下浏览器挂起加载导致白屏
watch(activeTab, async (tab) => {
  if (tab === 'web') {
    await nextTick()
    refreshWebUI()
  }
})

// 选项卡标题映射
const tabTitles: Record<TabValue, string> = {
  status: '网关状态监控',
  agent: 'Agent 代理配置',
  channels: '渠道中心',
  env: '环境参数管理',
  web: '内置控制台 Web UI'
}
</script>

<template>
  <div class="flex h-screen w-screen bg-[#060a13] text-slate-100 overflow-hidden font-sans">
    <!-- 常驻左侧高级磨砂侧边栏 -->
    <Sidebar v-model="activeTab" />

    <!-- 右侧内容主展区 -->
    <main class="flex-1 flex flex-col min-w-0 h-full relative">
      <!-- 右侧顶部精细页眉 -->
      <header class="h-14 border-b border-slate-900/60 bg-slate-950/25 backdrop-blur-md flex items-center justify-between px-8 shrink-0" data-wails-drag>
        <div class="flex items-center gap-3">
          <span class="text-xs bg-blue-500/10 text-blue-400 font-semibold px-2 py-0.5 rounded border border-blue-500/15">
            CCX CORE
          </span>
          <h2 class="text-sm font-bold text-slate-200 tracking-wide uppercase">
            {{ tabTitles[activeTab] }}
          </h2>
          <button
            v-if="activeTab === 'web'"
            class="text-slate-400 hover:text-slate-200 transition-colors p-1 rounded hover:bg-slate-800/50"
            title="刷新 Web UI"
            @click="refreshWebUI"
          >
            <RefreshCw class="w-3.5 h-3.5" />
          </button>
        </div>

        <div class="flex items-center gap-2">
          <!-- 实时网关状态指示微标 -->
          <span
            v-if="status.running"
            class="text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-bold px-2 py-0.5 rounded-full"
          >
            GATEWAY ONLINE
          </span>
          <span
            v-else-if="status.starting"
            class="text-[10px] bg-amber-500/10 text-amber-400 border border-amber-500/20 font-bold px-2 py-0.5 rounded-full animate-pulse"
          >
            CONNECTING...
          </span>
          <span
            v-else
            class="text-[10px] bg-rose-500/10 text-rose-400 border border-rose-500/20 font-bold px-2 py-0.5 rounded-full"
          >
            GATEWAY OFFLINE
          </span>
        </div>
      </header>

      <!-- 独立内容滚动区域 -->
      <div
        class="flex-1 overflow-y-auto"
        :class="activeTab === 'web' ? 'p-0' : 'px-8 py-7'"
      >
        <div :class="activeTab === 'web' ? 'h-full' : 'max-w-5xl mx-auto h-full'">
          <!-- v-show 常驻缓存各 Tab，切换时保留内部状态与滚动位置 -->
          <div v-show="activeTab === 'status'" class="h-full">
            <StatusTab @switch-to-web="switchToWeb" />
          </div>
          <div v-show="activeTab === 'agent'" class="h-full">
            <AgentTab />
          </div>
          <div v-show="activeTab === 'channels'" class="h-full">
            <ChannelTab />
          </div>
          <div v-show="activeTab === 'env'" class="h-full">
            <EnvTab />
          </div>
          <div v-show="activeTab === 'web'" class="h-full">
            <WebUITab ref="webTabRef" :status="status" :loading="false" />
          </div>
        </div>
      </div>
    </main>

    <UpdateDialog />
  </div>
</template>
