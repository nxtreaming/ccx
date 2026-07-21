<template>
  <div class="cockpit-board-page">
    <div class="cockpit-toolbar">
      <div class="cockpit-controls">
        <div class="cockpit-filter-controls">
          <v-select
            v-if="isCompactFilter"
            v-model="kindFilter"
            :items="kindFilterOptions"
            density="compact"
            variant="outlined"
            hide-details
            class="kind-filter-select"
          />
          <v-chip-group v-else v-model="kindFilter" mandatory selected-class="filter-chip-selected" class="cockpit-kind-filter">
            <v-chip value="" variant="outlined" size="small" class="filter-chip kind-all" filter>ALL</v-chip>
            <v-chip value="messages" variant="outlined" size="small" class="filter-chip kind-messages" filter>MESSAGES</v-chip>
            <v-chip value="chat" variant="outlined" size="small" class="filter-chip kind-chat" filter>CHAT</v-chip>
            <v-chip value="images" variant="outlined" size="small" class="filter-chip kind-images" filter>IMAGES</v-chip>
            <v-chip value="vectors" variant="outlined" size="small" class="filter-chip kind-vectors" filter>VECTORS</v-chip>
            <v-chip value="responses" variant="outlined" size="small" class="filter-chip kind-responses" filter>RESPONSES</v-chip>
            <v-chip value="gemini" variant="outlined" size="small" class="filter-chip kind-gemini" filter>GEMINI</v-chip>
          </v-chip-group>
        </div>

        <div class="cockpit-tool-controls">
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
        </div>
      </div>
    </div>

    <div v-if="loading && !conversations.length" class="d-flex justify-center py-12">
      <v-progress-circular indeterminate color="primary" />
    </div>

    <v-card v-else-if="!conversations.length" variant="outlined" class="text-center pa-12 cockpit-empty">
      <v-icon size="48" color="grey">mdi-chat-outline</v-icon>
      <div class="text-body-1 mt-4 text-medium-emphasis">
        {{ t('cockpit.empty') }}
      </div>
    </v-card>

    <template v-else>
      <v-card v-if="!visibleBoardItems.length" variant="outlined" class="text-center pa-8 mb-4 cockpit-empty">
        <div class="text-body-2 text-medium-emphasis">
          {{ t('cockpit.noMatches') }}
        </div>
      </v-card>

      <div v-else class="cockpit-board">
        <section
          v-for="column in boardColumns"
          :key="column.key"
          class="cockpit-column"
        >
          <div class="cockpit-column-head">
            <div class="cockpit-column-title">
              <span class="cockpit-column-dot" :style="{ background: column.color }"></span>
              <span>{{ column.label }}</span>
            </div>
            <v-tooltip :text="t('cockpit.tooltip.columnCount', { column: column.label, count: column.items.length })" location="top" :open-delay="150" content-class="ccx-tooltip">
              <template #activator="{ props: tooltipProps }">
                <span v-bind="tooltipProps" class="cockpit-column-count">{{ column.items.length }}</span>
              </template>
            </v-tooltip>
          </div>

          <div class="cockpit-column-body">
            <div v-if="!column.items.length" class="cockpit-column-empty">--</div>
            <div
              v-for="item in column.items"
              :key="item.conversation.id"
              :ref="el => setConversationCardRef(item.conversation.id, el as Element | null)"
              class="cockpit-board-card"
            >
              <ConversationCard
                :conversation="item.conversation"
                :subagents="item.subagents"
                :subagent-summary="item.subagentSummary"
                :override="overrides[item.conversation.id]"
                :available-channels="getChannelsForKind(item.conversation.kind)"
                :expanded="expandedCards.has(item.conversation.id)"
                :now-ms="nowMs"
                :related-parent-title="getConversationTitle(item.conversation.parentConversationId)"
                @toggle-expand="toggleExpand(item.conversation.id)"
                @set-override="handleSetOverride"
                @remove-override="handleRemoveOverride"
                @navigate-conversation="handleNavigateConversation"
                @success="(msg: string) => emit('success', msg)"
                @error="(msg: string) => emit('error', msg)"
              />
            </div>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { useDisplay } from 'vuetify'
import { api, type ConversationInfo, type SequenceOverrideInfo, type ChannelSequenceEntry } from '@/services/api'
import { useGlobalTick } from '@/composables/useGlobalTick'
import { useI18n } from '@/i18n'
import { buildConversationBoardItems, filterConversationBoardItems, type BoardColumnKey, type ConversationBoardItem } from '@/utils/conversationDashboard'
import ConversationCard from './ConversationCard.vue'

const { t } = useI18n()
const { width } = useDisplay()
const emit = defineEmits<{
  success: [message: string]
  error: [message: string]
}>()

type DashboardChannel = { index: number; name: string; priority: number; status: string; circuitOpen?: boolean }

const loading = ref(true)
const conversations = ref<ConversationInfo[]>([])
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const channelsByKind = ref<Record<string, DashboardChannel[]>>({})
const kindFilter = ref('')
const searchQuery = ref('')
const overrideDuration = ref(1800)
const nowMs = ref(Date.now())
const expandedCards = ref(new Set<string>())
const pinnedConversationOrder = ref<string[]>([])
const cardElements = new Map<string, HTMLElement>()

const isCompactFilter = computed(() => width.value < 400)

const boardColumnMeta = computed<Array<{ key: BoardColumnKey; label: string; color: string }>>(() => [
  { key: 'working', label: t('cockpit.column.working'), color: '#6366f1' },
  { key: 'idle', label: t('cockpit.column.idle'), color: '#10b981' },
])

const kindFilterOptions = [
  { title: 'ALL', value: '' },
  { title: 'MESSAGES', value: 'messages' },
  { title: 'CHAT', value: 'chat' },
  { title: 'IMAGES', value: 'images' },
  { title: 'VECTORS', value: 'vectors' },
  { title: 'RESPONSES', value: 'responses' },
  { title: 'GEMINI', value: 'gemini' },
]

const durationOptions = computed(() => [
  { title: t('cockpit.durationDefault'), value: 1800 },
  { title: t('cockpit.duration15min'), value: 900 },
  { title: t('cockpit.duration1hour'), value: 3600 },
  { title: t('cockpit.duration2hours'), value: 7200 },
  { title: t('cockpit.duration4hours'), value: 14400 },
  { title: t('cockpit.duration8hours'), value: 28800 },
  { title: t('cockpit.duration12hours'), value: 43200 },
  { title: t('cockpit.duration24hours'), value: 86400 },
  { title: t('cockpit.durationNever'), value: -1 },
])

const sortedConversations = computed(() => {
  if (expandedCards.value.size > 0) {
    return getPinnedOrderedConversations(conversations.value)
  }
  return sortConversationsByLastActive(conversations.value)
})

const boardItems = computed(() => buildConversationBoardItems(sortedConversations.value))

const visibleBoardItems = computed(() => {
  return filterConversationBoardItems(boardItems.value, kindFilter.value, searchQuery.value)
})

const boardColumns = computed(() => {
  const buckets = bucketBoardItems(visibleBoardItems.value)
  return boardColumnMeta.value.map(column => ({
    ...column,
    items: buckets[column.key],
  }))
})

function bucketBoardItems(items: ConversationBoardItem[]): Record<BoardColumnKey, ConversationBoardItem[]> {
  return items.reduce<Record<BoardColumnKey, ConversationBoardItem[]>>((buckets, item) => {
    buckets[item.aggregateStatus].push(item)
    return buckets
  }, { working: [], idle: [] })
}

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

function getChannelsForKind(kind: string): DashboardChannel[] {
  return channelsByKind.value[kind] || []
}

function getConversationTitle(id?: string) {
  if (!id) return ''
  const conversation = conversations.value.find(item => item.id === id)
  return conversation?.title || conversation?.userId || id
}

function getConversationTime(conversation: ConversationInfo) {
  return new Date(conversation.lastActiveAt).getTime()
}

function sortConversationsByLastActive(items: ConversationInfo[]) {
  return [...items].sort((a, b) => getConversationTime(b) - getConversationTime(a))
}

function getPinnedOrderedConversations(items: ConversationInfo[]) {
  const byID = new Map(items.map(item => [item.id, item]))
  const pinned = [...expandedCards.value]
    .filter(id => byID.has(id))
    .map(id => byID.get(id)!)
  const seen = new Set(pinned.map(item => item.id))
  const fresh = sortConversationsByLastActive(items.filter(item => !seen.has(item.id)))
  return [...pinned, ...fresh]
}

function applyConversationOrder(items: ConversationInfo[]) {
  const itemIDs = new Set(items.map(item => item.id))
  const expanded = [...expandedCards.value].filter(id => itemIDs.has(id))
  if (expanded.length !== expandedCards.value.size) {
    expandedCards.value = new Set(expanded)
  }

  if (expanded.length === 0) {
    const sorted = sortConversationsByLastActive(items)
    pinnedConversationOrder.value = sorted.map(item => item.id)
    return sorted
  }

  const ordered = getPinnedOrderedConversations(items)
  pinnedConversationOrder.value = ordered.map(item => item.id)
  return ordered
}

watch(expandedCards, expanded => {
  if (expanded.size === 0) {
    conversations.value = applyConversationOrder(conversations.value)
  }
})

async function fetchAllChannels() {
  const kinds = ['messages', 'chat', 'responses', 'gemini', 'images', 'vectors'] as const
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
    conversations.value = applyConversationOrder(resp.conversations || [])
    overrides.value = resp.overrides || {}
    if (resp.channelsByKind) channelsByKind.value = normalizeChannelsByKind(resp.channelsByKind)
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
    if (expandedCards.value.size === 0) {
      conversations.value = applyConversationOrder(conversations.value)
    }
    next.add(id)
  }
  expandedCards.value = next
}

function setConversationCardRef(id: string, el: Element | null) {
  if (el instanceof HTMLElement) {
    cardElements.set(id, el)
    return
  }
  cardElements.delete(id)
}

async function handleNavigateConversation(id: string) {
  if (!id) return

  const target = conversations.value.find(item => item.id === id)
  if (!target) {
    emit('error', `未找到关联对话 ${id.slice(0, 8)}`)
    return
  }

  pinnedConversationOrder.value = [
    id,
    ...pinnedConversationOrder.value.filter(itemID => itemID !== id),
  ]
  conversations.value = applyConversationOrder(conversations.value)
  kindFilter.value = ''
  searchQuery.value = ''

  const next = new Set(expandedCards.value)
  next.add(id)
  expandedCards.value = next

  await nextTick()
  const el = cardElements.get(id)
  if (!el) return

  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('conversation-card-target')
  window.setTimeout(() => {
    el.classList.remove('conversation-card-target')
  }, 1800)
}

async function handleSetOverride(
  convId: string,
  sequence: ChannelSequenceEntry[],
  subagentSequence?: ChannelSequenceEntry[],
  clearSubagentSequence = false,
) {
  try {
    await api.setConversationOverride(convId, sequence, overrideDuration.value, subagentSequence, clearSubagentSequence)
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

const tick = useGlobalTick(3000, 'ConversationDashboard')
tick.onTick(() => fetchConversations())
const clockTick = useGlobalTick(1000, 'ConversationDashboardClock')
clockTick.onTick(() => { nowMs.value = Date.now() })
fetchConversations()
fetchAllChannels()
</script>

<style scoped>
.cockpit-board-page {
  max-width: 1680px;
  margin: 0 auto;
}

.cockpit-toolbar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.cockpit-column-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex: 0 0 auto;
}

.cockpit-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.cockpit-filter-controls {
  display: flex;
  min-width: 0;
  flex: 0 1 auto;
}

.cockpit-tool-controls {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-left: auto;
  min-width: 0;
  flex: 1 1 420px;
}

.cockpit-kind-filter {
  flex: 0 1 auto;
}

.filter-chip {
  border-radius: 0 !important;
  font-size: 10px !important;
  font-weight: 700;
  letter-spacing: 0;
  transition: background-color 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

.kind-all {
  color: rgb(var(--v-theme-primary)) !important;
}

.kind-messages {
  color: #8b5cf6 !important;
}

.kind-chat {
  color: #3b82f6 !important;
}

.kind-images {
  color: #ec4899 !important;
}

.kind-vectors {
  color: #0ea5e9 !important;
}

.kind-responses {
  color: #14b8a6 !important;
}

.kind-gemini {
  color: #f97316 !important;
}

.filter-chip-selected.kind-all {
  background: rgb(var(--v-theme-primary) / 10%) !important;
  border-color: rgb(var(--v-theme-primary)) !important;
}

.filter-chip-selected.kind-messages {
  background: rgb(139 92 246 / 10%) !important;
  border-color: rgb(139 92 246 / 60%) !important;
}

.filter-chip-selected.kind-chat {
  background: rgb(59 130 246 / 10%) !important;
  border-color: rgb(59 130 246 / 60%) !important;
}

.filter-chip-selected.kind-images {
  background: rgb(236 72 153 / 10%) !important;
  border-color: rgb(236 72 153 / 60%) !important;
}

.filter-chip-selected.kind-vectors {
  background: rgb(14 165 233 / 10%) !important;
  border-color: rgb(14 165 233 / 60%) !important;
}

.filter-chip-selected.kind-responses {
  background: rgb(20 184 166 / 10%) !important;
  border-color: rgb(20 184 166 / 60%) !important;
}

.filter-chip-selected.kind-gemini {
  background: rgb(249 115 22 / 10%) !important;
  border-color: rgb(249 115 22 / 60%) !important;
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
  flex: 1 1 240px;
}

.cockpit-empty {
  border-radius: 0 !important;
}

.cockpit-board {
  display: grid;
  grid-template-columns: minmax(420px, 1.35fr) minmax(320px, 1fr);
  gap: 14px;
  align-items: start;
}

.cockpit-column {
  min-width: 0;
}

.cockpit-column-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 6px 4px 10px;
}

.cockpit-column-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

.cockpit-column-count {
  color: rgb(var(--v-theme-on-surface) / 50%);
  font-size: 12px;
  font-weight: 700;
}

.cockpit-column-body {
  min-height: 60vh;
  padding: 8px;
  background: rgba(0, 0, 0, 0.025);
  border: 1px dashed rgba(var(--v-border-color), var(--v-border-opacity));
}

.v-theme--dark .cockpit-column-body {
  background: rgba(255, 255, 255, 0.035);
}

.cockpit-column-empty {
  padding: 14px 8px;
  color: rgb(var(--v-theme-on-surface) / 38%);
  font-size: 12px;
  text-align: center;
}

.cockpit-board-card {
  margin-bottom: 10px;
}

.conversation-card-target :deep(.conversation-card) {
  border-color: rgb(var(--v-theme-primary));
  box-shadow: 0 0 0 2px rgb(var(--v-theme-primary) / 24%);
}

@media (max-width: 959px) {
  .cockpit-board {
    grid-template-columns: 1fr;
  }

  .cockpit-controls,
  .cockpit-tool-controls {
    gap: 8px;
  }

  .cockpit-tool-controls {
    flex: 1 1 320px;
  }

  .conversation-search-field {
    max-width: 260px;
    min-width: 150px;
    flex-basis: 190px;
  }

  .override-duration-select {
    max-width: 164px;
  }

  .filter-chip {
    padding-inline: 8px !important;
  }
}

@media (max-width: 600px) {
  .cockpit-toolbar {
    gap: 8px;
  }

  .cockpit-controls,
  .cockpit-tool-controls {
    gap: 6px;
  }

  .filter-chip {
    padding-inline: 7px !important;
  }
}

@media (max-width: 400px) {
  .cockpit-tool-controls {
    margin-left: 0;
    width: 100%;
    flex-basis: 100%;
    flex-wrap: nowrap;
  }

  .kind-filter-select {
    max-width: 170px;
    width: 170px;
    flex-basis: 170px;
  }

  .conversation-search-field {
    min-width: 0;
    max-width: none;
    flex: 1 1 0;
  }

  .override-duration-select {
    max-width: 154px;
    flex: 0 0 154px;
  }
}

@media (max-width: 340px) {
  .cockpit-tool-controls {
    flex-wrap: wrap;
  }

  .conversation-search-field,
  .override-duration-select {
    max-width: none;
    width: 100%;
    flex-basis: 100%;
  }
}
</style>
