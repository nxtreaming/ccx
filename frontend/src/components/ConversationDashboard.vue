<template>
  <div class="cockpit-board-page">
    <div class="cockpit-toolbar">
      <div class="cockpit-toolbar-main">
        <div class="cockpit-title">
          <v-icon size="20">mdi-view-dashboard-outline</v-icon>
          <span>{{ t('app.tabs.conversations') }}</span>
        </div>
        <div class="cockpit-stats">
          <div v-for="stat in boardStats" :key="stat.key" class="cockpit-stat">
            <span class="cockpit-stat-dot" :style="{ background: stat.color }"></span>
            <span class="cockpit-stat-label">{{ stat.label }}</span>
            <span class="cockpit-stat-value">{{ stat.count }}</span>
          </div>
        </div>
      </div>

      <div class="cockpit-controls">
        <v-select
          v-if="xs"
          v-model="kindFilter"
          :items="kindFilterOptions"
          density="compact"
          variant="outlined"
          hide-details
          class="kind-filter-select"
        />
        <v-chip-group v-else v-model="kindFilter" mandatory selected-class="text-primary" class="cockpit-kind-filter">
          <v-chip value="" variant="outlined" size="small" class="filter-chip" filter>ALL</v-chip>
          <v-chip value="messages" variant="outlined" size="small" color="purple" class="filter-chip" filter>MESSAGES</v-chip>
          <v-chip value="chat" variant="outlined" size="small" color="blue" class="filter-chip" filter>CHAT</v-chip>
          <v-chip value="images" variant="outlined" size="small" color="pink" class="filter-chip" filter>IMAGES</v-chip>
          <v-chip value="responses" variant="outlined" size="small" color="teal" class="filter-chip" filter>RESPONSES</v-chip>
          <v-chip value="gemini" variant="outlined" size="small" color="orange" class="filter-chip" filter>GEMINI</v-chip>
        </v-chip-group>

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
            <span class="cockpit-column-count">{{ column.items.length }}</span>
          </div>

          <div class="cockpit-column-body">
            <div v-if="!column.items.length" class="cockpit-column-empty">--</div>
            <ConversationCard
              v-for="item in column.items"
              :key="item.conversation.id"
              class="cockpit-board-card"
              :conversation="item.conversation"
              :subagents="item.subagents"
              :subagent-summary="item.subagentSummary"
              :override="overrides[item.conversation.id]"
              :available-channels="getChannelsForKind(item.conversation.kind)"
              :expanded="expandedCards.has(item.conversation.id)"
              :now-ms="nowMs"
              @toggle-expand="toggleExpand(item.conversation.id)"
              @set-override="handleSetOverride"
              @remove-override="handleRemoveOverride"
              @success="(msg: string) => emit('success', msg)"
              @error="(msg: string) => emit('error', msg)"
            />
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDisplay } from 'vuetify'
import { api, type ConversationInfo, type SequenceOverrideInfo, type ChannelSequenceEntry } from '@/services/api'
import { useGlobalTick } from '@/composables/useGlobalTick'
import { useI18n } from '@/i18n'
import { buildConversationBoardItems, filterConversationBoardItems, type BoardColumnKey, type ConversationBoardItem } from '@/utils/conversationDashboard'
import ConversationCard from './ConversationCard.vue'

const { t } = useI18n()
const { xs } = useDisplay()
const emit = defineEmits<{
  success: [message: string]
  error: [message: string]
}>()

type DashboardChannel = { index: number; name: string; priority: number; status: string; circuitOpen?: boolean }
type CockpitStat = { key: string; label: string; color: string; count: number }

const loading = ref(true)
const conversations = ref<ConversationInfo[]>([])
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const channelsByKind = ref<Record<string, DashboardChannel[]>>({})
const kindFilter = ref('')
const searchQuery = ref('')
const overrideDuration = ref(1800)
const nowMs = ref(Date.now())
const expandedCards = ref(new Set<string>())

const boardColumnMeta: Array<{ key: BoardColumnKey; label: string; color: string }> = [
  { key: 'working', label: 'Working', color: '#6366f1' },
  { key: 'idle', label: t('cockpit.column.idle'), color: '#10b981' },
]

const kindFilterOptions = [
  { title: 'ALL', value: '' },
  { title: 'MESSAGES', value: 'messages' },
  { title: 'CHAT', value: 'chat' },
  { title: 'IMAGES', value: 'images' },
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
  return [...conversations.value].sort((a, b) => new Date(b.lastActiveAt).getTime() - new Date(a.lastActiveAt).getTime())
})

const boardItems = computed(() => buildConversationBoardItems(sortedConversations.value))

const visibleBoardItems = computed(() => {
  return filterConversationBoardItems(boardItems.value, kindFilter.value, searchQuery.value)
})

const boardStats = computed<CockpitStat[]>(() => {
  const counts = bucketBoardItems(visibleBoardItems.value)
  const subagentTotal = visibleBoardItems.value.reduce((total, item) => total + item.subagentSummary.total, 0)
  const streamingTotal = visibleBoardItems.value.reduce((total, item) => total + (item.conversation.status === 'streaming' ? 1 : 0) + item.subagentSummary.streaming, 0)
  return [
    ...boardColumnMeta.map(column => ({
    ...column,
    label: column.label,
    count: counts[column.key].length,
    })),
    { key: 'subagents', label: t('cockpit.subagents'), color: '#f59e0b', count: subagentTotal },
    { key: 'streaming', label: t('cockpit.column.streaming'), color: '#ef4444', count: streamingTotal },
  ]
})

const boardColumns = computed(() => {
  const buckets = bucketBoardItems(visibleBoardItems.value)
  return boardColumnMeta.map(column => ({
    ...column,
    label: column.label,
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
    if (resp.channelsByKind) channelsByKind.value = normalizeChannelsByKind(resp.channelsByKind)
  } catch (e) {
    console.error('[ConversationDashboard] fetch error:', e)
  } finally {
    loading.value = false
  }
}

function toggleExpand(id: string) {
  const next = new Set(expandedCards.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedCards.value = next
}

async function handleSetOverride(convId: string, sequence: ChannelSequenceEntry[], subagentSequence?: ChannelSequenceEntry[]) {
  try {
    await api.setConversationOverride(convId, sequence, overrideDuration.value, subagentSequence)
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

.cockpit-toolbar-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.cockpit-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 800;
  color: rgb(var(--v-theme-on-surface));
}

.cockpit-stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.cockpit-stat {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 92px;
  padding: 7px 10px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  background: rgb(var(--v-theme-surface));
  font-size: 12px;
}

.cockpit-stat-dot,
.cockpit-column-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex: 0 0 auto;
}

.cockpit-stat-label {
  color: rgb(var(--v-theme-on-surface) / 68%);
}

.cockpit-stat-value {
  margin-left: auto;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.cockpit-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.cockpit-kind-filter {
  flex: 0 1 auto;
}

.filter-chip {
  border-radius: 0 !important;
  font-size: 10px !important;
  font-weight: 700;
  letter-spacing: 0;
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

@media (max-width: 1280px) {
  .cockpit-board {
    grid-template-columns: repeat(2, minmax(260px, 1fr));
  }
}

@media (max-width: 720px) {
  .cockpit-board {
    grid-template-columns: 1fr;
  }

  .conversation-search-field,
  .override-duration-select,
  .kind-filter-select {
    max-width: none;
    width: 100%;
    flex-basis: 100%;
  }
}
</style>
