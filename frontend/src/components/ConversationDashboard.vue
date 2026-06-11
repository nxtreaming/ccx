<template>
  <div class="conversation-dashboard">
    <!-- 过滤栏 -->
    <div class="d-flex align-center mb-4 flex-wrap ga-2">
      <!-- 手机端：下拉选择 -->
      <v-select
        v-if="xs"
        v-model="kindFilter"
        :items="kindFilterOptions"
        density="compact"
        variant="outlined"
        hide-details
        class="kind-filter-select"
      />
      <!-- 桌面端：chip 组 -->
      <v-chip-group v-else v-model="kindFilter" mandatory selected-class="text-primary">
        <v-chip value="" variant="outlined" size="small" class="filter-chip" filter>ALL</v-chip>
        <v-chip value="messages" variant="outlined" size="small" color="purple" class="filter-chip" filter>MESSAGES</v-chip>
        <v-chip value="chat" variant="outlined" size="small" color="blue" class="filter-chip" filter>CHAT</v-chip>
        <v-chip value="images" variant="outlined" size="small" color="pink" class="filter-chip" filter>IMAGES</v-chip>
        <v-chip value="responses" variant="outlined" size="small" color="teal" class="filter-chip" filter>RESPONSES</v-chip>
        <v-chip value="gemini" variant="outlined" size="small" color="orange" class="filter-chip" filter>GEMINI</v-chip>
      </v-chip-group>
      <v-spacer />
      <v-text-field
        v-model="searchQuery"
        density="compact"
        variant="outlined"
        hide-details
        clearable
        prepend-inner-icon="mdi-magnify"
        :placeholder="t('cockpit.searchPlaceholder')"
        class="conversation-search-field"
      />
      <v-select
        v-model="overrideDuration"
        :items="durationOptions"
        density="compact"
        variant="outlined"
        hide-details
        class="override-duration-select"
        :label="t('cockpit.overrideDuration')"
      />
      <span class="system-status-indicator" :class="'status-' + systemStore.systemStatus">
        <span class="status-dot"></span>
        {{ systemStatusText }}
      </span>
      <span class="text-caption text-medium-emphasis">
        Active: {{ visibleConversations.length }}
        <span v-if="overrideCount > 0" class="ml-2 text-warning">Override: {{ overrideCount }}</span>
      </span>
    </div>

    <!-- Loading -->
    <div v-if="loading && !conversations.length" class="d-flex justify-center py-12">
      <v-progress-circular indeterminate color="primary" />
    </div>

    <!-- Empty (no conversations at all) -->
    <v-card v-else-if="!conversations.length" variant="outlined" class="text-center pa-12">
      <v-icon size="48" color="grey">mdi-chat-outline</v-icon>
      <div class="text-body-1 mt-4 text-medium-emphasis">
        {{ t('cockpit.empty') }}
      </div>
    </v-card>

    <!-- Conversation cards -->
    <template v-else>
      <v-card v-if="!visibleConversations.length" variant="outlined" class="text-center pa-8 mb-4">
        <div class="text-body-2 text-medium-emphasis">
          {{ t('cockpit.noMatches') }}
        </div>
      </v-card>
      <div
        ref="masonryEl"
        class="conversation-masonry"
        :style="{ height: `${masonryHeight}px` }"
      >
        <div
          v-for="conv in visibleConversations"
          :key="conv.id"
          :ref="el => setMasonryItemRef(conv.id, el)"
          class="conversation-masonry-item"
          :style="getMasonryItemStyle(conv.id)"
        >
          <ConversationCard
            :conversation="conv"
            :override="overrides[conv.id]"
            :available-channels="getChannelsForKind(conv.kind)"
            :expanded="expandedCards.has(conv.id)"
            :now-ms="nowMs"
            @toggle-expand="toggleExpand(conv.id)"
            @set-override="handleSetOverride"
            @remove-override="handleRemoveOverride"
            @success="(msg: string) => emit('success', msg)"
            @error="(msg: string) => emit('error', msg)"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onBeforeUnmount, watch, type ComponentPublicInstance, type StyleValue } from 'vue'
import { useDisplay } from 'vuetify'
import { api, type ConversationInfo, type SequenceOverrideInfo, type ChannelSequenceEntry } from '@/services/api'
import { useGlobalTick } from '@/composables/useGlobalTick'
import { useI18n } from '@/i18n'
import { useSystemStore } from '@/stores/system'
import ConversationCard from './ConversationCard.vue'

const { t } = useI18n()
const { xs } = useDisplay()
const systemStore = useSystemStore() as any

const emit = defineEmits<{
  success: [message: string]
  error: [message: string]
}>()

const systemStatusText = computed(() => {
  switch (systemStore.systemStatus) {
    case 'running': return t('system.running')
    case 'error': return t('system.error')
    case 'connecting': return t('system.connecting')
    default: return t('system.unknown')
  }
})

const loading = ref(true)
const conversations = ref<ConversationInfo[]>([])
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const kindFilter = ref('')
const searchQuery = ref('')
const overrideDuration = ref(1800) // 默认 30min（对齐 OVERRIDE_TTL_MINUTES=30）
const nowMs = ref(Date.now())
const masonryEl = ref<HTMLElement | null>(null)
const masonryColumnCount = ref(1)
const masonryHeight = ref(0)
const masonryItemRects = ref<Record<string, { x: number; y: number; width: number }>>({})

const MASONRY_MIN_COLUMN_WIDTH = 320
const MASONRY_GAP = 16
const MASONRY_MAX_COLUMNS = 3
let masonryResizeObserver: ResizeObserver | undefined
let masonryLayoutFrame = 0
const masonryItemElements = new Map<string, HTMLElement>()
const masonryItemObservers = new Map<string, ResizeObserver>()

const kindFilterOptions = [
  { title: 'ALL', value: '' },
  { title: 'MESSAGES', value: 'messages' },
  { title: 'CHAT', value: 'chat' },
  { title: 'IMAGES', value: 'images' },
  { title: 'RESPONSES', value: 'responses' },
  { title: 'GEMINI', value: 'gemini' },
]
const expandedCards = ref(new Set<string>())
type DashboardChannel = { index: number; name: string; priority: number; status: string; circuitOpen?: boolean }

const channelsByKind = ref<Record<string, DashboardChannel[]>>({})

function normalizeChannel(ch: any): DashboardChannel {
  const index = ch.index ?? ch.Index ?? 0
  return {
    index,
    name: ch.name ?? ch.Name ?? `Channel ${index}`,
    priority: ch.priority ?? ch.Priority ?? index,
    status: ch.status ?? ch.Status ?? 'active',
    circuitOpen: ch.circuitOpen ?? ch.CircuitOpen ?? false,
  }
}

function normalizeChannelsByKind(value: Record<string, any[]>): Record<string, DashboardChannel[]> {
  return Object.fromEntries(
    Object.entries(value).map(([kind, channels]) => [
      kind,
      (channels || [])
        .map(normalizeChannel)
        .sort((a, b) => (a.priority - b.priority) || (a.index - b.index)),
    ]),
  )
}

const sortedConversations = computed(() => {
  return [...conversations.value].sort((a, b) => new Date(b.lastActiveAt).getTime() - new Date(a.lastActiveAt).getTime())
})

const visibleConversations = computed(() => {
  let list = sortedConversations.value
  const kind = kindFilter.value
  if (kind) list = list.filter(c => c.kind === kind)
  const q = (searchQuery.value || '').trim().toLowerCase()
  if (q) {
    list = list.filter(c =>
      (c.title || '').toLowerCase().includes(q) ||
      (c.userId || '').toLowerCase().includes(q) ||
      (c.rawUserId || '').toLowerCase().includes(q) ||
      (c.lastModel || '').toLowerCase().includes(q) ||
      (c.channelName || '').toLowerCase().includes(q),
    )
  }
  return list
})

const overrideCount = computed(() => Object.keys(overrides.value).length)

const durationOptions = computed(() => [
  { title: t('cockpit.durationDefault'), value: 1800 },
  { title: '15 min', value: 900 },
  { title: '1 hour', value: 3600 },
  { title: '2 hours', value: 7200 },
  { title: '4 hours', value: 14400 },
  { title: '8 hours', value: 28800 },
  { title: '12 hours', value: 43200 },
  { title: '24 hours', value: 86400 },
  { title: t('cockpit.durationNever'), value: -1 },
])

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
  masonryHeight.value = Math.max(0, ...columnHeights) - MASONRY_GAP
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

function getChannelsForKind(kind: string): DashboardChannel[] {
  return channelsByKind.value[kind] || []
}

async function fetchAllChannels() {
  const kinds = ['messages', 'chat', 'responses', 'gemini', 'images'] as const
  for (const kind of kinds) {
    try {
      const dashboard = await api.getChannelDashboard(kind)
      if (!channelsByKind.value[kind]?.length) {
        channelsByKind.value[kind] = (dashboard.channels || [])
          .map(normalizeChannel)
          .sort((a, b) => (a.priority - b.priority) || (a.index - b.index))
      }
    } catch (e) {
      console.error(`[ConversationDashboard] fetch ${kind} channels error:`, e)
    }
  }
}

async function fetchConversations() {
  try {
    const resp = await api.getConversations(undefined)
    conversations.value = resp.conversations || []
    overrides.value = resp.overrides || {}
    if (resp.channelsByKind) {
      channelsByKind.value = normalizeChannelsByKind(resp.channelsByKind)
    }
  } catch (e) {
    console.error('[ConversationDashboard] fetch error:', e)
  } finally {
    loading.value = false
  }
}

function toggleExpand(id: string) {
  const next = new Set(expandedCards.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  expandedCards.value = next
}

async function handleSetOverride(convId: string, sequence: ChannelSequenceEntry[]) {
  try {
    await api.setConversationOverride(convId, sequence, overrideDuration.value)
    await fetchConversations()
  } catch (e) {
    console.error('[ConversationDashboard] set override error:', e)
    emit('error', e instanceof Error ? e.message : 'Override failed')
  }
}

async function handleRemoveOverride(convId: string) {
  try {
    await api.removeConversationOverride(convId)
    await fetchConversations()
  } catch (e) {
    console.error('[ConversationDashboard] remove override error:', e)
    emit('error', e instanceof Error ? e.message : 'Remove override failed')
  }
}

// masonryEl 是条件渲染节点（v-else 分支），onMounted 时可能尚未挂载
// 因此用 watch 监听节点出现/消失，确保 RO 在节点存在时启动、消失时清理
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

onBeforeUnmount(() => {
  masonryResizeObserver?.disconnect()
  for (const observer of masonryItemObservers.values()) observer.disconnect()
  if (masonryLayoutFrame) window.cancelAnimationFrame(masonryLayoutFrame)
})

watch(visibleConversations, async () => {
  pruneMasonryItemRefs()
  await nextTick()
  scheduleMasonryLayout()
})

watch(masonryColumnCount, () => scheduleMasonryLayout())

// Polling (3s for data, 1s for clock)
const tick = useGlobalTick(3000, 'ConversationDashboard')
tick.onTick(() => fetchConversations())
const clockTick = useGlobalTick(1000, 'ConversationDashboardClock')
clockTick.onTick(() => { nowMs.value = Date.now() })
fetchConversations()
fetchAllChannels()
</script>

<style scoped>
.conversation-dashboard {
  max-width: 1400px;
  margin: 0 auto;
}
.filter-chip {
  border-radius: 0 !important;
  font-size: 10px !important;
  font-weight: 700;
  letter-spacing: 0.06em;
}
.kind-filter-select {
  max-width: 160px;
  flex: 0 0 auto;
}
.override-duration-select {
  max-width: 180px;
  flex: 0 0 auto;
}
.conversation-search-field {
  max-width: 320px;
  min-width: 180px;
  flex: 1 1 auto;
}
@media (max-width: 499px) {
  .conversation-search-field {
    max-width: 160px;
    min-width: 120px;
  }
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
  top: 0;
  left: 0;
  min-width: 0;
  transition: transform 0.16s ease;
  will-change: transform;
}
.system-status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  margin-right: 12px;
  border: 1px solid rgb(var(--v-theme-on-surface));
  background: rgb(var(--v-theme-surface));
}
.system-status-indicator .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #9ca3af;
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
