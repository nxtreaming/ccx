import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useAdminApi } from '@/composables/useAdminApi'
import { useStatus } from '@/composables/useStatus'
import { mergeChannelsWithLocalData } from '@/utils/channel-merge'
import { getChannelTypeApi, type ManagedChannelType } from '@/utils/channel-type-api'
import type {
  Channel,
  ChannelsResponse,
  ChannelMetrics,
  ChannelDashboardResponse,
  ChannelRecentActivity,
  SchedulerStatsResponse,
  PingResult,
} from '@/services/admin-api'

/**
 * 管理控制台频道状态层
 *
 * 使用 composable 单例模式（模块级 state），与 Desktop 现有架构一致。
 * 替代根 frontend 的 Pinia channel store，通过 HTTP 直接调后端 /api/*。
 */

type ChannelType = ManagedChannelType
type DashboardCache = {
  metrics: ChannelMetrics[]
  stats: SchedulerStatsResponse | undefined
  recentActivity: ChannelRecentActivity[] | undefined
}

const EMPTY_CACHE: DashboardCache = { metrics: [], stats: undefined, recentActivity: undefined }

// ===== 模块级单例状态 =====

const activeTab = ref<ChannelType>('messages')

// 五种协议的频道数据（独立缓存，切换不闪烁）
const channelsByType = ref<Record<ChannelType, ChannelsResponse>>({
  messages: { channels: [], current: -1 },
  chat: { channels: [], current: -1 },
  responses: { channels: [], current: -1 },
  gemini: { channels: [], current: -1 },
  images: { channels: [], current: -1 },
})

// 五种协议的 dashboard 缓存
const dashboardCache = ref<Record<ChannelType, DashboardCache>>({
  messages: { ...EMPTY_CACHE },
  chat: { ...EMPTY_CACHE },
  responses: { ...EMPTY_CACHE },
  gemini: { ...EMPTY_CACHE },
  images: { ...EMPTY_CACHE },
})

const isPingingAll = ref(false)
const refreshError = ref('')
let refreshLoopPromise: Promise<void> | null = null
let refreshRequested = false

// ===== 计算属性 =====

const currentChannelsData = computed(() => channelsByType.value[activeTab.value])
const currentDashboardMetrics = computed(() => dashboardCache.value[activeTab.value].metrics)
const currentDashboardStats = computed(() => dashboardCache.value[activeTab.value].stats)
const currentDashboardRecentActivity = computed(() => dashboardCache.value[activeTab.value].recentActivity)

const activeChannelCount = computed(() => {
  const chs = currentChannelsData.value.channels
  return chs.filter(ch => ch.status === 'active' || ch.status === undefined || ch.status === '').length
})

const failoverChannelCount = computed(() => {
  const chs = currentChannelsData.value.channels
  return chs.filter(ch => ch.status !== 'disabled').length
})

// ===== 刷新逻辑 =====

async function doRefresh(tab: ChannelType) {
  const api = useAdminApi()
  try {
    // 统一 dashboard 接口：GET /api/messages/channels/dashboard?type=<tab>
    const dashboard = await api.get<ChannelDashboardResponse>(
      `/api/messages/channels/dashboard?type=${tab}`
    )
    const existing = channelsByType.value[tab].channels
    channelsByType.value[tab] = {
      channels: mergeChannelsWithLocalData(dashboard.channels, existing),
      current: channelsByType.value[tab].current,
    }
    dashboardCache.value[tab] = {
      metrics: dashboard.metrics,
      stats: dashboard.stats,
      recentActivity: dashboard.recentActivity,
    }
    refreshError.value = ''
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    // 网络层 TypeError 包装为友好提示
    refreshError.value = msg.includes('Failed to fetch')
      ? '服务未运行或网络不可达，请检查后端是否已启动'
      : msg
  }
}

async function refreshChannels() {
  refreshRequested = true
  if (refreshLoopPromise) return refreshLoopPromise

  refreshLoopPromise = (async () => {
    try {
      while (refreshRequested) {
        refreshRequested = false
        await doRefresh(activeTab.value)
      }
    } finally {
      refreshLoopPromise = null
    }
  })()

  return refreshLoopPromise
}

// ===== 频道 CRUD 操作 =====

async function saveChannel(
  payload: Omit<Channel, 'index' | 'latency' | 'status'>,
  editingIndex: number | null,
) {
  const typeApi = getChannelTypeApi(activeTab.value)
  if (editingIndex !== null) {
    await typeApi.updateChannel(editingIndex, payload)
    await refreshChannels()
    return { success: true, message: '频道已更新' }
  }
  await typeApi.addChannel(payload)
  await refreshChannels()
  return { success: true, message: '频道已添加' }
}

async function deleteChannel(channelId: number) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.deleteChannel(channelId)
  await refreshChannels()
}

async function pingChannel(channelId: number) {
  const typeApi = getChannelTypeApi(activeTab.value)
  const result = await typeApi.pingChannel(channelId) as PingResult
  const channel = channelsByType.value[activeTab.value].channels.find(c => c.index === channelId)
  if (channel) {
    channel.latency = result.latency
    channel.latencyTestTime = Date.now()
  }
  return result
}

async function pingAllChannels() {
  if (isPingingAll.value) return
  isPingingAll.value = true
  try {
    const typeApi = getChannelTypeApi(activeTab.value)
    const results = await typeApi.pingAll() as PingResult[]
    const now = Date.now()
    const channels = channelsByType.value[activeTab.value].channels
    for (const result of results) {
      const ch = channels.find(c => c.index === (result as any).id)
      if (ch) {
        ch.latency = result.latency
        ch.latencyTestTime = now
      }
    }
  } finally {
    isPingingAll.value = false
  }
}

async function reorderChannels(order: number[]) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.reorder(order)
  await refreshChannels()
}

async function setChannelStatus(channelId: number, status: 'active' | 'suspended' | 'disabled') {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.setStatus(channelId, status)
  await refreshChannels()
}

async function resumeChannel(channelId: number) {
  const typeApi = getChannelTypeApi(activeTab.value)
  const result = await typeApi.resume(channelId)
  await refreshChannels()
  return result
}

async function promoteChannel(channelId: number, durationSeconds: number) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.promote(channelId, durationSeconds)
  await refreshChannels()
}

// ===== Key 管理 =====

async function addApiKey(channelId: number, key: string) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.addApiKey(channelId, key)
  await refreshChannels()
}

async function removeApiKey(channelId: number, key: string) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.removeApiKey(channelId, key)
  await refreshChannels()
}

async function restoreApiKey(channelId: number, key: string) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.restoreApiKey(channelId, key)
  await refreshChannels()
}

async function moveApiKeyToTop(channelId: number, key: string) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.moveApiKeyToTop(channelId, key)
  await refreshChannels()
}

async function moveApiKeyToBottom(channelId: number, key: string) {
  const typeApi = getChannelTypeApi(activeTab.value)
  await typeApi.moveApiKeyToBottom(channelId, key)
  await refreshChannels()
}

// ===== 自动刷新 =====

const AUTO_REFRESH_INTERVAL = 5000
let autoRefreshRunning = false

export function useConsoleChannels() {
  const { status } = useStatus()

  // 仅在服务运行时刷新（桌面 App Webview 始终在前台，无需可见性检查）
  const shouldRefresh = computed(() => status.value.running)

  const { pause, resume } = useIntervalFn(() => {
    if (autoRefreshRunning && shouldRefresh.value) {
      refreshChannels().catch(() => {})
    }
  }, AUTO_REFRESH_INTERVAL)

  function startAutoRefresh() {
    autoRefreshRunning = true
    resume()
    // 立即刷新一次
    if (shouldRefresh.value) {
      refreshChannels().catch(() => {})
    }
  }

  function stopAutoRefresh() {
    autoRefreshRunning = false
    pause()
  }

  onMounted(() => {
    if (status.value.running) {
      startAutoRefresh()
    }
  })

  onBeforeUnmount(() => {
    stopAutoRefresh()
  })

  // 服务状态变化时自动启停刷新
  watch(() => status.value.running, (running) => {
    if (running) startAutoRefresh()
    else stopAutoRefresh()
  })

  return {
    // 状态
    activeTab,
    channelsByType,
    dashboardCache,
    isPingingAll,
    refreshError,

    // 当前 tab 的计算属性
    currentChannelsData,
    currentDashboardMetrics,
    currentDashboardStats,
    currentDashboardRecentActivity,
    activeChannelCount,
    failoverChannelCount,

    // 操作
    refreshChannels,
    saveChannel,
    deleteChannel,
    pingChannel,
    pingAllChannels,
    reorderChannels,
    setChannelStatus,
    resumeChannel,
    promoteChannel,

    // Key 管理
    addApiKey,
    removeApiKey,
    restoreApiKey,
    moveApiKeyToTop,
    moveApiKeyToBottom,

    // 刷新控制
    startAutoRefresh,
    stopAutoRefresh,
  }
}
