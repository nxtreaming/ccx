<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type ComponentPublicInstance, type StyleValue } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { Alert } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { MessageSquare, Search } from 'lucide-vue-next'
import { useConversations } from '@/composables/useConversations'
import { useLanguage } from '@/composables/useLanguage'
import { useStatus } from '@/composables/useStatus'
import ConversationCard from './ConversationCard.vue'
import type { ChannelSequenceEntry } from '@/services/admin-api'

const { status } = useStatus()
const { tf } = useLanguage()
const {
  conversations,
  channelsByKind,
  overrides,
  loading,
  error,
  fetchConversations,
  setOverride,
  removeOverride,
} = useConversations()

const kindFilter = ref('')
const searchQuery = ref('')
const overrideDuration = ref('1800') // 默认 30min（对齐 OVERRIDE_TTL_MINUTES=30）
const nowMs = ref(Date.now())
const expandedCards = ref(new Set<string>())
const masonryEl = ref<HTMLElement | null>(null)
const masonryColumnCount = ref(1)
const masonryHeight = ref(0)
const masonryItemRects = ref<Record<string, { x: number; y: number; width: number }>>({})
const notice = ref<{ variant: 'success' | 'destructive'; message: string } | null>(null)

const MASONRY_MIN_COLUMN_WIDTH = 320
const MASONRY_GAP = 16
const MASONRY_MAX_COLUMNS = 3
let masonryResizeObserver: ResizeObserver | undefined
let masonryLayoutFrame = 0
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let refreshPromise: Promise<void> | null = null
const masonryItemElements = new Map<string, HTMLElement>()
const masonryItemObservers = new Map<string, ResizeObserver>()

const kindFilterOptions = [
  { label: 'ALL', value: '' },
  { label: 'MESSAGES', value: 'messages', class: 'border-purple-500/50 text-purple-500 data-[active=true]:bg-purple-500/10' },
  { label: 'CHAT', value: 'chat', class: 'border-blue-500/50 text-blue-500 data-[active=true]:bg-blue-500/10' },
  { label: 'IMAGES', value: 'images', class: 'border-pink-500/50 text-pink-500 data-[active=true]:bg-pink-500/10' },
  { label: 'RESPONSES', value: 'responses', class: 'border-teal-500/50 text-teal-500 data-[active=true]:bg-teal-500/10' },
  { label: 'GEMINI', value: 'gemini', class: 'border-orange-500/50 text-orange-500 data-[active=true]:bg-orange-500/10' },
]

const systemState = computed(() => {
  if (status.value.running) return 'running'
  if (status.value.starting) return 'connecting'
  if (status.value.lastError) return 'error'
  return 'unknown'
})

const systemStatusText = computed(() => {
  switch (systemState.value) {
    case 'running': return tf('system.running', '运行中')
    case 'error': return tf('system.error', '连接失败')
    case 'connecting': return tf('system.connecting', '连接中')
    default: return tf('system.unknown', '未知')
  }
})

const sortedConversations = computed(() => {
  return [...conversations.value].sort((a, b) => new Date(b.lastActiveAt).getTime() - new Date(a.lastActiveAt).getTime())
})

const visibleConversations = computed(() => {
  let list = sortedConversations.value
  if (kindFilter.value) {
    list = list.filter(conversation => conversation.kind === kindFilter.value)
  }

  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return list

  return list.filter(conversation =>
    (conversation.title || '').toLowerCase().includes(query)
    || (conversation.userId || '').toLowerCase().includes(query)
    || (conversation.rawUserId || '').toLowerCase().includes(query)
    || (conversation.lastModel || '').toLowerCase().includes(query)
    || (conversation.channelName || '').toLowerCase().includes(query)
  )
})

const overrideCount = computed(() => Object.keys(overrides.value).length)
const shouldRefresh = computed(() => status.value.running)

const durationOptions = computed(() => [
  { label: tf('cockpit.durationDefault', '30 min（默认）'), value: '1800' },
  { label: '15 min', value: '900' },
  { label: '1 hour', value: '3600' },
  { label: '2 hours', value: '7200' },
  { label: '4 hours', value: '14400' },
  { label: '8 hours', value: '28800' },
  { label: '12 hours', value: '43200' },
  { label: '24 hours', value: '86400' },
  { label: tf('cockpit.durationNever', '永不恢复'), value: '-1' },
])

function overrideDurationAsNumber(): number {
  return Number(overrideDuration.value)
}

function showNotice(variant: 'success' | 'destructive', message: string) {
  notice.value = { variant, message }
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => {
    notice.value = null
    noticeTimer = undefined
  }, 2400)
}

function getChannelsForKind(kind: string) {
  return channelsByKind.value[kind] || []
}

async function refreshConversations() {
  if (!shouldRefresh.value || refreshPromise) return refreshPromise
  refreshPromise = fetchConversations()
    .finally(() => {
      refreshPromise = null
    })
  return refreshPromise
}

function toggleExpand(id: string) {
  const next = new Set(expandedCards.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedCards.value = next
}

async function handleSetOverride(conversationId: string, sequence: ChannelSequenceEntry[]) {
  try {
    await setOverride(conversationId, sequence, overrideDurationAsNumber())
  } catch (e) {
    showNotice('destructive', e instanceof Error ? e.message : String(e))
  }
}

async function handleRemoveOverride(conversationId: string) {
  try {
    await removeOverride(conversationId)
  } catch (e) {
    showNotice('destructive', e instanceof Error ? e.message : String(e))
  }
}

function handleSuccess(message: string) {
  showNotice('success', message)
}

function handleError(message: string) {
  showNotice('destructive', message)
}

function updateMasonryColumnCount(width = masonryEl.value?.clientWidth || 0) {
  const fit = Math.floor((width + MASONRY_GAP) / (MASONRY_MIN_COLUMN_WIDTH + MASONRY_GAP))
  masonryColumnCount.value = Math.max(1, Math.min(fit, MASONRY_MAX_COLUMNS))
}

function scheduleMasonryLayout() {
  if (masonryLayoutFrame) return
  masonryLayoutFrame = window.requestAnimationFrame(() => {
    masonryLayoutFrame = 0
    layoutMasonryItems()
  })
}

function layoutMasonryItems() {
  const containerWidth = masonryEl.value?.clientWidth || 0
  if (!containerWidth) return

  const columnCount = Math.max(1, masonryColumnCount.value)
  const columnWidth = (containerWidth - MASONRY_GAP * (columnCount - 1)) / columnCount
  const columnHeights = Array.from({ length: columnCount }, () => 0)
  const nextRects: Record<string, { x: number; y: number; width: number }> = {}

  for (const conversation of visibleConversations.value) {
    let targetColumn = 0
    for (let i = 1; i < columnHeights.length; i++) {
      if (columnHeights[i] < columnHeights[targetColumn]) targetColumn = i
    }

    const element = masonryItemElements.get(conversation.id)
    const itemHeight = element?.offsetHeight || 0
    const x = targetColumn * (columnWidth + MASONRY_GAP)
    const y = columnHeights[targetColumn]
    nextRects[conversation.id] = { x, y, width: columnWidth }
    columnHeights[targetColumn] += itemHeight + MASONRY_GAP
  }

  masonryItemRects.value = nextRects
  masonryHeight.value = Math.max(0, Math.max(...columnHeights) - MASONRY_GAP)
}

function pruneMasonryItemRefs() {
  const visibleIds = new Set(visibleConversations.value.map(conversation => conversation.id))
  for (const id of masonryItemElements.keys()) {
    if (visibleIds.has(id)) continue
    masonryItemObservers.get(id)?.disconnect()
    masonryItemObservers.delete(id)
    masonryItemElements.delete(id)
  }
}

function getMasonryItemStyle(id: string): StyleValue {
  const rect = masonryItemRects.value[id]
  if (!rect) {
    return {
      width: masonryColumnCount.value > 1 ? `${MASONRY_MIN_COLUMN_WIDTH}px` : '100%',
      transform: 'translate3d(0, 0, 0)',
      visibility: 'hidden',
    }
  }
  return {
    width: `${rect.width}px`,
    transform: `translate3d(${rect.x}px, ${rect.y}px, 0)`,
  }
}

function setMasonryItemRef(id: string, el: Element | ComponentPublicInstance | null) {
  const element = el instanceof HTMLElement ? el : null
  const existing = masonryItemElements.get(id)
  if (existing === element) return

  const existingObserver = masonryItemObservers.get(id)
  existingObserver?.disconnect()
  masonryItemObservers.delete(id)
  masonryItemElements.delete(id)

  if (!element) {
    scheduleMasonryLayout()
    return
  }

  masonryItemElements.set(id, element)
  const observer = new ResizeObserver(() => scheduleMasonryLayout())
  observer.observe(element)
  masonryItemObservers.set(id, observer)
  scheduleMasonryLayout()
}

const { pause: pauseDataTick, resume: resumeDataTick } = useIntervalFn(() => {
  refreshConversations().catch(() => {})
}, 3000, { immediate: false })

const { pause: pauseClockTick, resume: resumeClockTick } = useIntervalFn(() => {
  nowMs.value = Date.now()
}, 1000, { immediate: false })

onMounted(() => {
  resumeDataTick()
  resumeClockTick()
  refreshConversations().catch(() => {})
})

onBeforeUnmount(() => {
  pauseDataTick()
  pauseClockTick()
  masonryResizeObserver?.disconnect()
  for (const observer of masonryItemObservers.values()) observer.disconnect()
  if (masonryLayoutFrame) window.cancelAnimationFrame(masonryLayoutFrame)
  if (noticeTimer) clearTimeout(noticeTimer)
})

watch(masonryEl, (el, _prev, onCleanup) => {
  if (!el) return
  updateMasonryColumnCount(el.clientWidth)
  const observer = new ResizeObserver(entries => {
    const entry = entries[0]
    if (!entry) return
    updateMasonryColumnCount(entry.contentRect.width)
  })
  observer.observe(el)
  masonryResizeObserver = observer
  nextTick(() => scheduleMasonryLayout())
  onCleanup(() => {
    observer.disconnect()
    if (masonryResizeObserver === observer) masonryResizeObserver = undefined
  })
}, { flush: 'post' })

watch(visibleConversations, async () => {
  pruneMasonryItemRefs()
  await nextTick()
  scheduleMasonryLayout()
})

watch(masonryColumnCount, () => scheduleMasonryLayout())
watch(shouldRefresh, running => {
  if (running) refreshConversations().catch(() => {})
})
</script>

<template>
  <div class="conversation-dashboard mx-auto w-full max-w-[1400px]">
    <Alert v-if="error" variant="destructive" class="mb-4">
      <p class="text-sm">{{ error }}</p>
    </Alert>

    <Alert v-if="notice" :variant="notice.variant" class="mb-4">
      <p class="text-sm">{{ notice.message }}</p>
    </Alert>

    <!-- 过滤栏：对齐 WebUI cockpit 顶部结构 -->
    <div class="mb-4 flex flex-wrap items-center gap-2">
      <div class="flex flex-wrap items-center gap-1.5">
        <button
          v-for="option in kindFilterOptions"
          :key="option.value"
          type="button"
          class="filter-chip border border-border px-2.5 py-1 text-[10px] font-bold tracking-[0.08em] text-muted-foreground transition-colors hover:bg-accent/40"
          :class="[option.class, { 'border-primary bg-primary/10 text-primary': kindFilter === option.value }]"
          :data-active="kindFilter === option.value"
          @click="kindFilter = option.value"
        >
          {{ option.label }}
        </button>
      </div>

      <div class="min-w-4 flex-1" />

      <div class="relative w-full min-w-[180px] sm:w-64 lg:w-80">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          :placeholder="tf('cockpit.searchPlaceholder', '搜索...')"
          class="h-9 pl-9"
        />
      </div>

      <Select v-model="overrideDuration">
        <SelectTrigger class="h-9 w-[140px] shrink-0">
          <SelectValue :placeholder="tf('cockpit.overrideDuration', 'Override 有效期')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="opt in durationOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </SelectItem>
        </SelectContent>
      </Select>

      <span class="system-status-indicator" :class="`status-${systemState}`">
        <span class="status-dot" />
        {{ systemStatusText }}
      </span>

      <span class="text-xs text-muted-foreground">
        Active: {{ visibleConversations.length }}
        <span v-if="overrideCount > 0" class="ml-2 text-amber-500">Override: {{ overrideCount }}</span>
      </span>
    </div>

    <div v-if="loading && !conversations.length" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      <Skeleton v-for="i in 6" :key="i" class="h-32 w-full rounded-none" />
    </div>

    <div v-else-if="!conversations.length" class="border border-border bg-card/60 px-6 py-12 text-center">
      <MessageSquare class="mx-auto h-12 w-12 text-muted-foreground/70" />
      <div class="mt-4 text-sm text-muted-foreground">
        {{ tf('cockpit.empty', '暂无活跃会话。请求经过网关后，会话会出现在驾驶舱雷达上。') }}
      </div>
    </div>

    <template v-else>
      <div v-if="!visibleConversations.length" class="mb-4 border border-border bg-card/60 px-6 py-8 text-center text-sm text-muted-foreground">
        {{ tf('cockpit.noMatches', '没有匹配当前过滤条件的会话') }}
      </div>

      <div
        ref="masonryEl"
        class="conversation-masonry"
        :style="{ height: `${masonryHeight}px` }"
      >
        <div
          v-for="conversation in visibleConversations"
          :key="conversation.id"
          :ref="el => setMasonryItemRef(conversation.id, el)"
          class="conversation-masonry-item"
          :style="getMasonryItemStyle(conversation.id)"
        >
          <ConversationCard
            :conversation="conversation"
            :override="overrides[conversation.id]"
            :available-channels="getChannelsForKind(conversation.kind)"
            :expanded="expandedCards.has(conversation.id)"
            :now-ms="nowMs"
            @toggle-expand="toggleExpand(conversation.id)"
            @set-override="handleSetOverride"
            @remove-override="handleRemoveOverride"
            @success="handleSuccess"
            @error="handleError"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.filter-chip {
  border-radius: 0;
}

.conversation-masonry {
  position: relative;
  box-sizing: border-box;
  padding-right: 6px;
  padding-bottom: 6px;
  transition: height 0.16s ease;
}

.conversation-masonry-item {
  position: absolute;
  left: 0;
  top: 0;
  min-width: 0;
  transition: transform 0.16s ease;
  will-change: transform;
}

.system-status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--color-border);
  background: var(--color-card);
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
}

.system-status-indicator .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-muted-foreground);
}

.system-status-indicator.status-running .status-dot {
  background: #10b981;
  animation: dot-pulse 2s ease-in-out infinite;
}

.system-status-indicator.status-error .status-dot {
  background: #ef4444;
}

.system-status-indicator.status-connecting .status-dot {
  background: #f59e0b;
}

@keyframes dot-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
