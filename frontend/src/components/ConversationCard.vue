<template>
  <v-card
    :class="['conversation-card', { 'override-active': hasOverride }]"
    :style="{ '--ccx-kind-color': kindCssColor }"
    elevation="0"
    role="button"
    tabindex="0"
    :aria-expanded="expanded"
    @click="$emit('toggleExpand')"
    @keydown.enter.prevent="$emit('toggleExpand')"
    @keydown.space.prevent="$emit('toggleExpand')"
  >
    <v-card-text class="pa-4">
      <div class="task-card-id">{{ shortConversationId }}</div>
      <div class="task-card-title-row">
        <span :class="['status-led', `status-led--${conversation.status}`]"></span>
        <span class="task-card-title">
          <v-tooltip :text="tooltipText" location="top" :open-delay="150" content-class="ccx-tooltip">
            <template #activator="{ props: tp }">
              <span v-bind="tp" class="display-label-text">{{ displayLabel }}</span>
            </template>
          </v-tooltip>
        </span>
      </div>

      <div class="task-card-meta">
        <v-chip class="kind-chip" :color="kindColor" size="x-small" variant="outlined">{{ kindLabel }}</v-chip>
        <span class="task-meta-item">
          <span class="task-meta-dot"></span>
          {{ conversation.requestCount }}x
        </span>
        <span class="task-meta-item">{{ duration }}</span>
        <v-chip :color="statusColor" size="x-small" variant="tonal" class="task-status-chip">
          {{ conversation.status.toUpperCase() }}
        </v-chip>
      </div>

      <div v-if="hasSubagentActivity" class="subagent-summary" @click.stop="$emit('toggleExpand')">
        <div class="subagent-summary-main">
          <v-icon size="16">mdi-source-branch</v-icon>
          <span>{{ t('cockpit.subagents') }}</span>
          <strong>{{ displaySubagentCount }}</strong>
        </div>
        <div class="subagent-summary-route">
          <span v-if="subagentSummary.total > 0">{{ subagentSummary.streaming }} streaming / {{ subagentSummary.active }} active</span>
          <template v-else>
          <span>{{ mainChannelLabel }}</span>
          <v-icon size="13">mdi-arrow-right</v-icon>
          <span>{{ subagentChannelLabel }}</span>
          </template>
        </div>
      </div>

      <div v-if="subagents.length > 0" class="subagent-list" @click.stop>
        <div class="subagent-list-head">
          <span>Subagents</span>
          <span>{{ subagentSummary.streaming }} streaming · {{ subagentSummary.active }} active · {{ subagentSummary.idle }} idle</span>
        </div>
        <div v-for="agent in visibleSubagents" :key="agent.id" class="subagent-row">
          <span :class="['subagent-dot', `subagent-dot--${agent.status}`]"></span>
          <div class="subagent-row-main">
            <span class="subagent-row-title">{{ agent.title || agent.userId }}</span>
            <span class="subagent-row-meta">{{ agent.lastModel }} · {{ agent.channelName || `Channel ${agent.currentChannel}` }}</span>
          </div>
          <v-chip size="x-small" variant="tonal" :color="subagentStatusColor(agent.status)">
            {{ agent.status }}
          </v-chip>
        </div>
        <button v-if="subagents.length > visibleSubagents.length" type="button" class="subagent-more" @click.stop="$emit('toggleExpand')">
          +{{ subagents.length - visibleSubagents.length }} more
        </button>
      </div>

      <div class="task-card-notes">
        <span>{{ conversation.lastModel }}</span>
        <span class="task-card-channel">{{ conversation.channelName || `Channel ${conversation.currentChannel}` }}</span>
      </div>

      <!-- Row 2: Model + Channel chips (collapsed) -->
      <div v-if="!expanded" class="d-flex align-center ga-2 flex-wrap">
        <v-tooltip v-for="ch in visibleChannels" :key="ch.index" :text="getChannelTooltip(ch)" location="top" :open-delay="150" content-class="ccx-tooltip">
          <template #activator="{ props: tip }">
            <v-chip
              v-bind="tip"
              :class="{
                'current-channel-chip': ch.index === conversation.currentChannel && !hasOverride,
                'next-channel-chip': ch.index === nextChannel,
              }"
              :color="ch.index === conversation.currentChannel ? 'primary' : ch.index === nextChannel ? (nextChannelCircuitOpen ? 'error' : 'success') : undefined"
              :variant="ch.index === conversation.currentChannel ? 'flat' : ch.index === nextChannel ? 'flat' : 'outlined'"
              size="x-small"
              @click.stop="handleQuickOverride(ch)"
            >
              {{ ch.name }}
              <template v-if="ch.index === conversation.currentChannel" #append>
                <v-icon size="10">mdi-check</v-icon>
              </template>
              <template v-else-if="ch.index === nextChannel" #append>
                <span class="next-label">| {{ nextChannelCircuitOpen ? 'TRIPPED' : 'NEXT' }}</span>
              </template>
            </v-chip>
          </template>
        </v-tooltip>
        <v-chip v-if="hiddenCount > 0" size="x-small" variant="text" @click.stop="$emit('toggleExpand')">+{{ hiddenCount }}</v-chip>
      </div>

      <!-- Expanded: Override alert -->
      <v-alert v-if="expanded && hasOverride" type="warning" density="compact" variant="tonal" class="override-alert mb-2 mt-2">
        <div class="d-flex align-center">
          <span class="alert-bang">[!]</span>
          <span v-if="override?.isPerpetual" class="text-caption">{{ t('cockpit.overrideActivePerpetual') }}</span>
          <span v-else class="text-caption">{{ t('cockpit.overrideActive', { time: remainingTime }) }}</span>
          <v-spacer />
          <v-btn size="x-small" variant="text" @click.stop="$emit('removeOverride', conversation.id)">{{ t('cockpit.restoreDefault') }}</v-btn>
        </div>
      </v-alert>

      <!-- Expanded: Full channel sequence -->
      <div v-if="expanded" class="mt-3">
        <div class="text-caption text-medium-emphasis mb-1">{{ conversation.lastModel }}</div>
        <ConversationChannelSequence
          :channels="channelSequence"
          :current-channel="conversation.currentChannel"
          :next-channel="nextChannel"
          :next-channel-circuit-open="nextChannelCircuitOpen"
          :override-active="hasOverride"
          @move-to-top="handleMoveToTop"
          @demote="handleDemote"
        />

        <!-- Subagent Routing：为主对话与 subagent 分别指定渠道 -->
        <div v-if="showSubagentSection" class="subagent-routing mt-3" @click.stop>
          <div class="d-flex align-center mb-1">
            <span class="text-caption text-medium-emphasis">Subagent routing</span>
            <span v-if="hasSubagentOverride" class="text-caption text-warning ml-2">[{{ t('cockpit.subagentOverride') }}]</span>
            <span v-else class="text-caption text-medium-emphasis ml-2">[{{ t('cockpit.subagentFollowMain') }}]</span>
            <v-spacer />
            <v-btn v-if="hasSubagentOverride" size="x-small" variant="text" @click.stop="handleClearSubagentOverride">{{ t('cockpit.subagentClearOverride') }}</v-btn>
          </div>
          <ConversationChannelSequence
            :channels="subagentSequence"
            :current-channel="subagentCurrentChannel"
            :next-channel="subagentNextChannel"
            :next-channel-circuit-open="subagentNextChannelCircuitOpen"
            :override-active="hasOverride"
            @move-to-top="handleSubagentMoveToTop"
            @demote="handleSubagentDemote"
          />
        </div>
      </div>

      <!-- Row 3: Raw User ID -->
      <div v-if="conversation.rawUserId" class="raw-user-id mt-2 d-flex align-center">
        <span class="text-caption text-medium-emphasis font-weight-mono raw-user-id-text" @click.stop="copyRawUserId">{{ conversation.rawUserId }}</span>
        <v-btn icon size="x-small" variant="text" class="copy-btn" aria-label="Copy conversation ID" @click.stop="copyRawUserId">
          <v-icon size="12">mdi-content-copy</v-icon>
        </v-btn>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ConversationInfo, SequenceOverrideInfo, ChannelSequenceEntry } from '@/services/api'
import type { SubagentSummary } from '@/utils/conversationDashboard'
import { useI18n } from '@/i18n'
import ConversationChannelSequence from './ConversationChannelSequence.vue'

const { t } = useI18n()

interface ChannelInfo {
  index: number
  name: string
  status: string
  circuitOpen?: boolean
}

const props = defineProps<{
  conversation: ConversationInfo
  subagents?: ConversationInfo[]
  subagentSummary?: SubagentSummary
  override?: SequenceOverrideInfo
  availableChannels: ChannelInfo[]
  expanded: boolean
  nowMs: number
}>()

const emit = defineEmits<{
  toggleExpand: []
  setOverride: [convId: string, sequence: ChannelSequenceEntry[], subagentSequence?: ChannelSequenceEntry[]]
  removeOverride: [convId: string]
  success: [message: string]
  error: [message: string]
}>()

const MAX_VISIBLE = 6

const conversation = computed(() => props.conversation)
const emptySubagentSummary: SubagentSummary = { total: 0, streaming: 0, active: 0, idle: 0 }
const subagents = computed(() => props.subagents ?? [])
const subagentSummary = computed(() => props.subagentSummary ?? emptySubagentSummary)
const hasSubagentActivity = computed(() => props.conversation.hasSubagents || subagents.value.length > 0)
const displaySubagentCount = computed(() => subagentSummary.value.total || props.conversation.subagentCount || subagents.value.length || 1)
const visibleSubagents = computed(() => subagents.value.slice(0, props.expanded ? 12 : 4))
const hasOverride = computed(() => !!props.override)
const kindLabel = computed(() => `[ ${props.conversation.kind.toUpperCase()} ]`)

const kindColor = computed(() => {
  switch (props.conversation.kind) {
    case 'messages': return 'purple'
    case 'chat': return 'blue'
    case 'responses': return 'teal'
    case 'gemini': return 'orange'
    case 'images': return 'pink'
    default: return 'grey'
  }
})

const statusColor = computed(() => {
  switch (props.conversation.status) {
    case 'streaming': return 'error'
    case 'active': return 'primary'
    case 'idle': return 'success'
    default: return 'grey'
  }
})

function subagentStatusColor(status: ConversationInfo['status']): string {
  switch (status) {
    case 'streaming': return 'error'
    case 'active': return 'primary'
    case 'idle': return 'success'
    default: return 'grey'
  }
}

const kindCssColor = computed(() => {
  const map: Record<string, string> = {
    messages: 'var(--ccx-kind-messages)',
    chat: 'var(--ccx-kind-chat)',
    responses: 'var(--ccx-kind-responses)',
    gemini: 'var(--ccx-kind-gemini)',
    images: 'var(--ccx-kind-images)',
  }
  return map[props.conversation.kind] ?? 'rgb(var(--v-theme-on-surface))'
})

const displayLabel = computed(() => props.conversation.title || props.conversation.userId)
const shortConversationId = computed(() => props.conversation.id.slice(0, 12))

const tooltipText = computed(() => {
  if (props.conversation.title) return props.conversation.title
  return props.conversation.userId
})

const duration = computed(() => {
  const start = new Date(props.conversation.createdAt).getTime()
  const mins = Math.floor((props.nowMs - start) / 60000)
  if (mins < 1) return '<1m'
  if (mins < 60) return `${mins}m`
  return `${Math.floor(mins / 60)}h${mins % 60}m`
})

const remainingTime = computed(() => {
  if (!props.override) return ''
  if (props.override.isPerpetual) return t('cockpit.durationNever')
  const expires = new Date(props.override.expiresAt).getTime()
  const remaining = Math.max(0, expires - props.nowMs)
  const mins = Math.floor(remaining / 60000)
  const secs = Math.floor((remaining % 60000) / 1000)
  return `${mins}:${secs.toString().padStart(2, '0')}`
})

const fallbackChannels = computed((): ChannelInfo[] => {
  const channels: ChannelInfo[] = []
  const pushUnique = (channel: ChannelInfo) => {
    if (!channels.some(ch => ch.index === channel.index)) channels.push(channel)
  }
  if (props.override?.sequence) {
    for (const entry of props.override.sequence) {
      pushUnique({ index: entry.channelIndex, name: entry.channelName || `Channel ${entry.channelIndex}`, status: 'active' })
    }
  }
  pushUnique({ index: props.conversation.currentChannel, name: props.conversation.channelName || `Channel ${props.conversation.currentChannel}`, status: 'active' })
  return channels
})

const channelSequence = computed((): ChannelInfo[] => {
  if (props.override?.sequence) {
    return props.override.sequence.map(entry => {
      const ch = props.availableChannels.find(c => c.index === entry.channelIndex)
      return { index: entry.channelIndex, name: entry.channelName || ch?.name || `Channel ${entry.channelIndex}`, status: ch?.status || 'active', circuitOpen: ch?.circuitOpen }
    })
  }
  const channels = props.availableChannels.filter(ch => ch.status !== 'disabled')
  return channels.length > 0 ? channels : fallbackChannels.value
})

// subagent 渠道序列：优先用 override.subagentSequence，否则 fallback 到主序列
const subagentSequence = computed((): ChannelInfo[] => {
  if (props.override?.subagentSequence && props.override.subagentSequence.length > 0) {
    return props.override.subagentSequence.map(entry => {
      const ch = props.availableChannels.find(c => c.index === entry.channelIndex)
      return { index: entry.channelIndex, name: entry.channelName || ch?.name || `Channel ${entry.channelIndex}`, status: ch?.status || 'active', circuitOpen: ch?.circuitOpen }
    })
  }
  return channelSequence.value
})

const hasSubagentOverride = computed(() => !!props.override?.subagentSequence && props.override.subagentSequence.length > 0)
const showSubagentSection = computed(() => hasSubagentActivity.value || hasSubagentOverride.value)
const subagentCurrentChannel = computed(() => props.conversation.subagentChannel ?? -1)

const mainChannelLabel = computed(() => {
  const index = props.conversation.mainChannel ?? props.conversation.currentChannel
  return getChannelName(index)
})

const subagentChannelLabel = computed(() => {
  const index = props.conversation.subagentChannel
  return index === undefined ? 'fallback' : getChannelName(index)
})

function getChannelName(index: number): string {
  return props.availableChannels.find(ch => ch.index === index)?.name || `Channel ${index}`
}

const currentChannelInfo = computed(() => {
  const existing = channelSequence.value.find(ch => ch.index === props.conversation.currentChannel)
    ?? props.availableChannels.find(ch => ch.index === props.conversation.currentChannel)
  if (existing) return existing
  return { index: props.conversation.currentChannel, name: props.conversation.channelName || `Channel ${props.conversation.currentChannel}`, status: 'active' }
})

const nextChannel = computed(() => {
  const candidate = props.override?.sequence?.[0]?.channelIndex
  return candidate !== undefined && candidate !== props.conversation.currentChannel ? candidate : undefined
})

const nextChannelInfo = computed(() => {
  if (nextChannel.value === undefined) return undefined
  const existing = channelSequence.value.find(ch => ch.index === nextChannel.value)
    ?? props.availableChannels.find(ch => ch.index === nextChannel.value)
  if (existing) return existing
  const entry = props.override?.sequence?.[0]
  return { index: nextChannel.value!, name: entry?.channelName || `Channel ${nextChannel.value}`, status: 'active' }
})

const nextChannelCircuitOpen = computed(() => {
  if (!nextChannelInfo.value) return false
  return nextChannelInfo.value.circuitOpen === true
})

const subagentNextChannel = computed(() => {
  const candidate = props.override?.subagentSequence?.[0]?.channelIndex ?? props.override?.sequence?.[0]?.channelIndex
  return candidate !== undefined && candidate !== subagentCurrentChannel.value ? candidate : undefined
})

const subagentNextChannelInfo = computed(() => {
  if (subagentNextChannel.value === undefined) return undefined
  const existing = subagentSequence.value.find(ch => ch.index === subagentNextChannel.value)
    ?? props.availableChannels.find(ch => ch.index === subagentNextChannel.value)
  if (existing) return existing
  const entry = props.override?.subagentSequence?.[0]
  return { index: subagentNextChannel.value!, name: entry?.channelName || `Channel ${subagentNextChannel.value}`, status: 'active' }
})

const subagentNextChannelCircuitOpen = computed(() => {
  if (!subagentNextChannelInfo.value) return false
  return subagentNextChannelInfo.value.circuitOpen === true
})

const visibleChannels = computed(() => {
  const result: ChannelInfo[] = []
  const required = [currentChannelInfo.value, nextChannelInfo.value].filter((ch): ch is ChannelInfo => !!ch)
  const requiredIndexes = new Set(required.map(ch => ch.index))
  const pushUnique = (channel?: ChannelInfo) => {
    if (!channel || result.some(ch => ch.index === channel.index)) return
    result.push(channel)
  }
  for (const ch of channelSequence.value) {
    if (result.length >= MAX_VISIBLE) break
    pushUnique(ch)
  }
  for (const channel of required) {
    if (result.some(ch => ch.index === channel.index)) continue
    if (result.length >= MAX_VISIBLE) {
      let removeIndex = result.length - 1
      for (let i = result.length - 1; i >= 0; i--) {
        if (!requiredIndexes.has(result[i].index)) { removeIndex = i; break }
      }
      result.splice(removeIndex, 1)
    }
    pushUnique(channel)
  }
  return result
})

const hiddenCount = computed(() => Math.max(0, channelSequence.value.length - visibleChannels.value.length))

function buildSequence(channels: ChannelInfo[]): ChannelSequenceEntry[] {
  return channels.map(ch => ({ channelIndex: ch.index, channelName: ch.name }))
}

function getChannelTooltip(ch: ChannelInfo): string {
  if (ch.index === props.conversation.currentChannel && !hasOverride.value) return 'Current channel'
  if (ch.index === nextChannel.value) return 'Next override target'
  return 'Click to set as next'
}

function handleQuickOverride(ch: ChannelInfo) {
  if (!hasOverride.value && ch.index === props.conversation.currentChannel) return
  const rest = channelSequence.value.filter(c => c.index !== ch.index)
  emit('setOverride', props.conversation.id, buildSequence([ch, ...rest]))
}

// subagent 渠道：移到最前（等同主对话的 moveToTop）
function handleSubagentMoveToTop(ch: ChannelInfo, currentIdx: number) {
  if (currentIdx === 0) return
  const current = [...subagentSequence.value]
  const [item] = current.splice(currentIdx, 1)
  current.unshift(item)
  emit('setOverride', props.conversation.id, buildSequence(channelSequence.value), buildSequence(current))
}

function handleSubagentDemote(index: number) {
  const current = [...subagentSequence.value]
  if (index >= current.length - 1) return
  const [item] = current.splice(index, 1)
  current.push(item)
  emit('setOverride', props.conversation.id, buildSequence(channelSequence.value), buildSequence(current))
}

// 清除 subagent override：API 会省略空 subagentSequence，后端重建主 override 时移除旧子序列。
function handleClearSubagentOverride() {
  emit('setOverride', props.conversation.id, buildSequence(channelSequence.value), [])
}

function handleMoveToTop(ch: ChannelInfo, currentIdx: number) {
  if (currentIdx === 0) return
  const current = [...channelSequence.value]
  const [item] = current.splice(currentIdx, 1)
  current.unshift(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}

function handleDemote(index: number) {
  const current = [...channelSequence.value]
  if (index >= current.length - 1) return
  const [item] = current.splice(index, 1)
  current.push(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}

async function copyRawUserId() {
  if (!props.conversation.rawUserId) return
  try {
    await navigator.clipboard.writeText(props.conversation.rawUserId)
    emit('success', t('cockpit.rawUserIdCopied'))
  } catch {
    emit('error', t('cockpit.rawUserIdCopyFailed'))
  }
}

</script>

<style scoped>
.conversation-card {
  cursor: pointer;
  position: relative;
  transition: all 0.1s ease;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  box-shadow: none;
  background:
    radial-gradient(circle, var(--ccx-dot-grid-color) 1px, transparent 1px) 0 0 / var(--ccx-dot-grid-size) var(--ccx-dot-grid-size),
    rgb(var(--v-theme-surface));
  border-radius: 0;
}
.conversation-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  bottom: 8px;
  width: 3px;
  background: var(--ccx-kind-color);
  border: 0;
  pointer-events: none;
  z-index: 1;
}
.conversation-card:hover {
  border-color: var(--ccx-kind-color);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08);
}
.conversation-card:active {
  transform: translateY(1px);
}
.conversation-card.override-active {
  border-color: rgb(var(--v-theme-warning));
}
.conversation-card.override-active:hover {
  border-color: rgb(var(--v-theme-warning));
}
.v-theme--dark .conversation-card {
  border-color: rgba(255, 255, 255, 0.16);
  box-shadow: none;
}
.v-theme--dark .conversation-card:hover {
  border-color: var(--ccx-kind-color);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.34);
}

.task-card-id {
  margin-bottom: 4px;
  color: rgb(var(--v-theme-on-surface) / 42%);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 1.2;
}

.task-card-title-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}

.task-card-title {
  flex: 1 1 auto;
  min-width: 0;
  color: rgb(var(--v-theme-on-surface));
  font-size: 13px;
  font-weight: 800;
  line-height: 1.4;
}

.task-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.task-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: rgb(var(--v-theme-on-surface) / 64%);
  font-size: 11px;
  font-weight: 700;
}

.task-meta-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--ccx-kind-color);
}

.task-status-chip {
  margin-left: auto;
  font-weight: 800;
}

.task-card-notes {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
  color: rgb(var(--v-theme-on-surface) / 62%);
  font-size: 12px;
  line-height: 1.45;
}

.task-card-channel {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: rgb(var(--v-theme-on-surface) / 44%);
}

/* Status LED */
.status-led {
  display: inline-block;
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-led--streaming {
  background: var(--ccx-led-streaming-color);
  animation: ccx-led-pulse 1.4s ease-in-out infinite;
}
.status-led--active {
  background: var(--ccx-status-breaker-half-open-dot-bg);
  box-shadow: 0 0 4px 1px var(--ccx-status-breaker-half-open-dot-glow);
}
.status-led--idle {
  background: var(--ccx-status-disabled-dot-bg);
}

/* Kind chip */
.kind-chip {
  border-radius: 0 !important;
  font-size: 9px !important;
  font-weight: 700;
  letter-spacing: 0.08em;
}

/* Display label (title/userId) */
.display-label {
  min-width: 0;
  flex: 1;
}
.display-label-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subagent-summary {
  margin-top: 10px;
  padding: 8px 10px;
  border: 1px solid rgba(var(--v-theme-warning), 0.45);
  background: rgba(var(--v-theme-warning), 0.1);
}

.subagent-summary-main,
.subagent-summary-route {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.subagent-summary-main {
  color: rgb(var(--v-theme-warning));
  font-size: 12px;
  font-weight: 800;
}

.subagent-summary-route {
  margin-top: 5px;
  color: rgb(var(--v-theme-on-surface) / 62%);
  font-size: 11px;
}

.subagent-summary-route span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subagent-list {
  margin-top: 8px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  background: rgb(var(--v-theme-surface) / 72%);
}

.subagent-list-head {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 10px;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  color: rgb(var(--v-theme-on-surface) / 62%);
  font-size: 11px;
  font-weight: 700;
}

.subagent-row {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  padding: 7px 10px;
  border-bottom: 1px solid rgba(var(--v-border-color), 0.5);
}

.subagent-row:last-of-type {
  border-bottom: 0;
}

.subagent-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
}

.subagent-dot--streaming {
  background: var(--ccx-led-streaming-color);
  animation: ccx-led-pulse 1.4s ease-in-out infinite;
}

.subagent-dot--active {
  background: var(--ccx-status-breaker-half-open-dot-bg);
}

.subagent-dot--idle {
  background: var(--ccx-status-disabled-dot-bg);
}

.subagent-row-main {
  min-width: 0;
}

.subagent-row-title,
.subagent-row-meta {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subagent-row-title {
  color: rgb(var(--v-theme-on-surface) / 86%);
  font-size: 12px;
  font-weight: 700;
}

.subagent-row-meta {
  color: rgb(var(--v-theme-on-surface) / 48%);
  font-size: 11px;
}

.subagent-more {
  width: 100%;
  padding: 6px 10px;
  color: rgb(var(--v-theme-on-surface) / 58%);
  font-size: 11px;
  text-align: left;
}

/* Next label */
.next-label {
  display: inline-block;
  margin-left: 6px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.05em;
}

/* Override alert */
.override-alert {
  border: 2px solid rgb(var(--v-theme-warning)) !important;
  border-radius: 0 !important;
}
.alert-bang {
  font-weight: 900;
  font-size: 11px;
  letter-spacing: 0.1em;
  margin-right: 6px;
  animation: ccx-alert-blink 0.8s step-end infinite;
  color: rgb(var(--v-theme-warning));
}

.current-channel-chip {
  cursor: default !important;
  opacity: 0.85;
}

.font-weight-mono {
  font-family: monospace;
}

/* Raw User ID */
.raw-user-id {
  border-top: 1px dashed rgba(var(--v-border-color), var(--v-border-opacity));
  padding-top: 6px;
  opacity: 0.6;
  cursor: pointer;
}
.raw-user-id:hover {
  opacity: 0.9;
}
.raw-user-id-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.raw-user-id .copy-btn {
  flex-shrink: 0;
  opacity: 0.5;
}
.raw-user-id:hover .copy-btn {
  opacity: 1;
}

</style>
