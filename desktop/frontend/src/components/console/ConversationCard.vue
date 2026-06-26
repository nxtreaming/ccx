<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Check, Copy, CornerUpLeft, GitBranch } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'
import type {
  ChannelSequenceEntry,
  ConversationChannelInfo,
  ConversationInfo,
  SequenceOverrideInfo,
} from '@/services/admin-api'
import type { SubagentSummary } from '@/utils/conversation-dashboard'
import ConversationChannelSequence from './ConversationChannelSequence.vue'

interface ChannelInfo {
  index: number
  name: string
  status: string
  priority?: number
  circuitOpen?: boolean
}

const props = defineProps<{
  conversation: ConversationInfo
  subagents?: ConversationInfo[]
  subagentSummary?: SubagentSummary
  override?: SequenceOverrideInfo
  availableChannels: ConversationChannelInfo[]
  expanded: boolean
  nowMs: number
  relatedParentTitle?: string
}>()

const emit = defineEmits<{
  toggleExpand: []
  setOverride: [conversationId: string, sequence: ChannelSequenceEntry[], subagentSequence?: ChannelSequenceEntry[]]
  removeOverride: [conversationId: string]
  navigateConversation: [conversationId: string]
  success: [message: string]
  error: [message: string]
}>()

const { t } = useLanguage()
const MAX_VISIBLE = 6
const emptySubagentSummary: SubagentSummary = { total: 0, streaming: 0, active: 0, idle: 0 }

const conversation = computed(() => props.conversation)
const subagents = computed(() => props.subagents ?? [])
const subagentSummary = computed(() => props.subagentSummary ?? emptySubagentSummary)
const hasSubagentActivity = computed(() => props.conversation.hasSubagents || subagents.value.length > 0)
const displaySubagentCount = computed(() => subagentSummary.value.total || props.conversation.subagentCount || subagents.value.length || 1)
const visibleSubagents = computed(() => subagents.value.slice(0, props.expanded ? 12 : 4))
const hasOverride = computed(() => !!props.override)
const kindLabel = computed(() => props.conversation.kind.toUpperCase())

function splitConversationTurns(text: string): string[] {
  const turns = text
    .split(/\s+\/\s+/)
    .map((turn) => turn.trim())
    .filter(Boolean)

  return turns.length > 0 ? turns : [text]
}

const kindStyle = computed(() => {
  switch (props.conversation.kind) {
    case 'messages': return { color: '#a855f7', chip: 'border-purple-500/60 text-purple-500 bg-purple-500/10' }
    case 'chat': return { color: '#3b82f6', chip: 'border-blue-500/60 text-blue-500 bg-blue-500/10' }
    case 'responses': return { color: '#14b8a6', chip: 'border-teal-500/60 text-teal-500 bg-teal-500/10' }
    case 'gemini': return { color: '#f97316', chip: 'border-orange-500/60 text-orange-500 bg-orange-500/10' }
    case 'images': return { color: '#ec4899', chip: 'border-pink-500/60 text-pink-500 bg-pink-500/10' }
    default: return { color: 'var(--color-foreground)', chip: 'border-border text-muted-foreground bg-muted/20' }
  }
})

const displayLabel = computed(() => props.conversation.title || props.conversation.userId)
const mainConversationText = computed(() => props.conversation.lastUserMessage || displayLabel.value)
const mainConversationTurns = computed(() => splitConversationTurns(mainConversationText.value))
const tooltipText = computed(() => props.conversation.title || props.conversation.userId)
const childConversationCount = computed(() => props.conversation.childConversationIds?.length ?? 0)
const firstChildConversationId = computed(() => props.conversation.childConversationIds?.[0])
const parentThreadLabel = computed(() => props.conversation.parentThreadId ? shortId(props.conversation.parentThreadId) : '')

const duration = computed(() => {
  const start = new Date(props.conversation.createdAt).getTime()
  if (!Number.isFinite(start)) return '<1m'
  const mins = Math.floor((props.nowMs - start) / 60000)
  if (mins < 1) return '<1m'
  if (mins < 60) return `${mins}m`
  return `${Math.floor(mins / 60)}h${mins % 60}m`
})

const mainDetailRows = computed(() => [
  { label: t('cockpit.detail.requests'), value: `${props.conversation.requestCount}x` },
  { label: t('cockpit.detail.duration'), value: duration.value },
])

const remainingTime = computed(() => {
  if (!props.override?.expiresAt) return '--:--'
  const expires = new Date(props.override.expiresAt).getTime()
  if (!Number.isFinite(expires)) return '--:--'
  const remaining = Math.max(0, expires - props.nowMs)
  const mins = Math.floor(remaining / 60000)
  const secs = Math.floor((remaining % 60000) / 1000)
  return `${mins}:${secs.toString().padStart(2, '0')}`
})

const normalizedAvailableChannels = computed<ChannelInfo[]>(() => {
  return props.availableChannels
    .map(channel => ({
      index: channel.index,
      name: channel.name,
      status: channel.status || 'active',
      priority: channel.priority,
      circuitOpen: channel.circuitOpen,
    }))
    .sort((a, b) => ((a.priority ?? a.index) - (b.priority ?? b.index)) || (a.index - b.index))
})

const fallbackChannels = computed((): ChannelInfo[] => {
  const channels: ChannelInfo[] = []
  const pushUnique = (channel: ChannelInfo) => {
    if (!channels.some(item => item.index === channel.index)) channels.push(channel)
  }

  if (props.override?.sequence) {
    for (const entry of props.override.sequence) {
      pushUnique({
        index: entry.channelIndex,
        name: entry.channelName || `Channel ${entry.channelIndex}`,
        status: 'active',
      })
    }
  }

  pushUnique({
    index: props.conversation.currentChannel,
    name: props.conversation.channelName || `Channel ${props.conversation.currentChannel}`,
    status: 'active',
  })

  return channels
})

const channelSequence = computed((): ChannelInfo[] => {
  if (props.override?.sequence?.length) {
    return props.override.sequence.map(entry => {
      const channel = normalizedAvailableChannels.value.find(item => item.index === entry.channelIndex)
      return {
        index: entry.channelIndex,
        name: entry.channelName || channel?.name || `Channel ${entry.channelIndex}`,
        status: channel?.status || 'active',
        priority: channel?.priority,
        circuitOpen: channel?.circuitOpen,
      }
    })
  }

  const channels = normalizedAvailableChannels.value.filter(channel => channel.status !== 'disabled')
  return channels.length > 0 ? channels : fallbackChannels.value
})

const currentChannelInfo = computed(() => {
  const existing = channelSequence.value.find(channel => channel.index === props.conversation.currentChannel)
    ?? normalizedAvailableChannels.value.find(channel => channel.index === props.conversation.currentChannel)
  if (existing) return existing
  return {
    index: props.conversation.currentChannel,
    name: props.conversation.channelName || `Channel ${props.conversation.currentChannel}`,
    status: 'active',
  }
})

const nextChannel = computed(() => {
  const candidate = props.override?.sequence?.[0]?.channelIndex
  return candidate !== undefined && candidate !== props.conversation.currentChannel ? candidate : undefined
})

const nextChannelInfo = computed(() => {
  if (nextChannel.value === undefined) return undefined
  const existing = channelSequence.value.find(channel => channel.index === nextChannel.value)
    ?? normalizedAvailableChannels.value.find(channel => channel.index === nextChannel.value)
  if (existing) return existing
  const entry = props.override?.sequence?.[0]
  return {
    index: nextChannel.value,
    name: entry?.channelName || `Channel ${nextChannel.value}`,
    status: 'active',
  }
})

const nextChannelCircuitOpen = computed(() => nextChannelInfo.value?.circuitOpen === true)

const visibleChannels = computed(() => {
  const result: ChannelInfo[] = []
  const required = [currentChannelInfo.value, nextChannelInfo.value].filter((channel): channel is ChannelInfo => !!channel)
  const requiredIndexes = new Set(required.map(channel => channel.index))
  const pushUnique = (channel?: ChannelInfo) => {
    if (!channel || result.some(item => item.index === channel.index)) return
    result.push(channel)
  }

  for (const channel of channelSequence.value) {
    if (result.length >= MAX_VISIBLE) break
    pushUnique(channel)
  }

  for (const channel of required) {
    if (result.some(item => item.index === channel.index)) continue
    if (result.length >= MAX_VISIBLE) {
      let removeIndex = result.length - 1
      for (let i = result.length - 1; i >= 0; i--) {
        if (!requiredIndexes.has(result[i].index)) {
          removeIndex = i
          break
        }
      }
      result.splice(removeIndex, 1)
    }
    pushUnique(channel)
  }

  return result
})

const hiddenCount = computed(() => Math.max(0, channelSequence.value.length - visibleChannels.value.length))

// subagent 渠道序列：优先用 override.subagentSequence，否则 fallback 到主序列
const subagentSequence = computed((): ChannelInfo[] => {
  if (props.override?.subagentSequence?.length) {
    return props.override.subagentSequence.map(entry => {
      const ch = normalizedAvailableChannels.value.find(c => c.index === entry.channelIndex)
      return { index: entry.channelIndex, name: entry.channelName || ch?.name || `Channel ${entry.channelIndex}`, status: ch?.status || 'active', circuitOpen: ch?.circuitOpen }
    })
  }
  return channelSequence.value
})
const hasSubagentOverride = computed(() => !!props.override?.subagentSequence?.length)
const showSubagentSection = computed(() => hasSubagentActivity.value || hasSubagentOverride.value)
const subagentCurrentChannel = computed(() => props.conversation.subagentChannel ?? -1)
const subagentNextChannel = computed(() => {
  const candidate = props.override?.subagentSequence?.[0]?.channelIndex ?? props.override?.sequence?.[0]?.channelIndex
  return candidate !== undefined && candidate !== subagentCurrentChannel.value ? candidate : undefined
})

const subagentNextChannelInfo = computed(() => {
  if (subagentNextChannel.value === undefined) return undefined
  const existing = subagentSequence.value.find(channel => channel.index === subagentNextChannel.value)
    ?? normalizedAvailableChannels.value.find(channel => channel.index === subagentNextChannel.value)
  if (existing) return existing
  const entry = props.override?.subagentSequence?.[0]
  return {
    index: subagentNextChannel.value,
    name: entry?.channelName || `Channel ${subagentNextChannel.value}`,
    status: 'active',
  }
})

const subagentNextChannelCircuitOpen = computed(() => subagentNextChannelInfo.value?.circuitOpen === true)

const subagentRouteLabel = computed(() => {
  if (props.conversation.subagentChannel === undefined) return t('cockpit.subagentFollowMain')
  return t('cockpit.subagentOverride')
})


function subagentStatusClass(status: ConversationInfo['status']): string {
  switch (status) {
    case 'streaming': return 'bg-rose-500'
    case 'active': return 'bg-blue-500'
    case 'idle': return 'bg-muted-foreground'
    default: return 'bg-muted-foreground'
  }
}

function buildSequence(channels: ChannelInfo[]): ChannelSequenceEntry[] {
  return channels.map(channel => ({
    channelIndex: channel.index,
    channelName: channel.name,
  }))
}

function getChannelTooltip(channel: ChannelInfo): string {
  if (channel.index === props.conversation.currentChannel && !hasOverride.value) return 'Current channel'
  if (channel.index === nextChannel.value) return 'Next override target'
  if (channel.status === 'suspended' || channel.circuitOpen) return 'Click to resume and set as next'
  return 'Click to set as next'
}

function channelChipClass(channel: ChannelInfo): string {
  if (channel.index === props.conversation.currentChannel) {
    return 'border-primary bg-primary text-primary-foreground shadow-sm'
  }
  if (channel.index === nextChannel.value) {
    return nextChannelCircuitOpen.value
      ? 'next-channel-chip border-red-500 bg-red-500 text-white'
      : 'next-channel-chip border-emerald-500 bg-emerald-500 text-white'
  }
  return 'border-border bg-background/50 text-foreground hover:border-primary/40 hover:bg-accent/30'
}

function handleQuickOverride(channel: ChannelInfo) {
  if (!hasOverride.value && channel.index === props.conversation.currentChannel && channel.status !== 'suspended' && !channel.circuitOpen) return
  const rest = channelSequence.value.filter(item => item.index !== channel.index)
  emit('setOverride', props.conversation.id, buildSequence([channel, ...rest]))
}

function handleSubagentMoveToTop(channel: ChannelInfo, currentIndex: number) {
  if (currentIndex === 0) return
  const current = [...subagentSequence.value]
  const [item] = current.splice(currentIndex, 1)
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

function handleMoveToTop(channel: ChannelInfo, currentIndex: number) {
  if (currentIndex === 0 && channel.status !== 'suspended' && !channel.circuitOpen) return
  const current = [...channelSequence.value]
  const [item] = current.splice(currentIndex, 1)
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

function navigateConversation(id?: string) {
  if (!id) return
  emit('navigateConversation', id)
}

function shortId(value: string): string {
  if (value.length <= 12) return value
  return `${value.slice(0, 8)}...${value.slice(-4)}`
}

</script>

<template>
  <div
    class="conversation-card border bg-card/85 p-4 text-card-foreground"
    :class="{ 'override-active': hasOverride }"
    :style="{ '--ccx-kind-color': kindStyle.color }"
    role="button"
    tabindex="0"
    :aria-expanded="expanded"
    @click="emit('toggleExpand')"
    @keydown.enter.prevent="emit('toggleExpand')"
    @keydown.space.prevent="emit('toggleExpand')"
  >
    <!-- Row 1: LED + Kind + Title/User + Stats -->
    <div class="mb-3 flex min-w-0 items-center gap-2">
      <span class="status-led" :class="`status-led--${conversation.status}`" />
      <span class="kind-chip border px-1 py-0.5 text-[9px] font-bold tracking-[0.08em]" :class="kindStyle.chip">
        {{ kindLabel }}
      </span>
      <span class="display-label min-w-0 flex-1 font-mono text-xs text-muted-foreground" :title="tooltipText">
        <span class="display-label-text" :class="{ 'display-label-text--expanded': expanded }">{{ displayLabel }}</span>
      </span>
      <span class="shrink-0 text-xs text-muted-foreground">{{ conversation.requestCount }}x</span>
      <span class="shrink-0 text-xs text-muted-foreground">{{ duration }}</span>
      <span
        v-if="hasSubagentActivity"
        class="inline-flex shrink-0 items-center border border-amber-500/50 bg-amber-500/10 px-1 py-0.5 text-[10px] font-medium text-amber-500"
      >
        SA {{ displaySubagentCount }}
      </span>
    </div>

    <div
      v-if="conversation.parentConversationId || conversation.parentThreadId || (!expanded && childConversationCount > 0)"
      class="relation-row mb-3 flex flex-wrap items-center gap-1.5"
      @click.stop
    >
      <button
        v-if="conversation.parentConversationId"
        type="button"
        class="relation-chip border border-sky-500/50 bg-sky-500/10 text-sky-600 hover:bg-sky-500/20 dark:text-sky-400"
        :title="relatedParentTitle || conversation.parentConversationId"
        @click="navigateConversation(conversation.parentConversationId)"
      >
        <CornerUpLeft class="h-3 w-3" />
        <span>{{ t('cockpit.relation.parent') }}</span>
      </button>
      <span
        v-else-if="conversation.parentThreadId"
        class="relation-chip border border-border bg-muted/30 text-muted-foreground"
        :title="conversation.parentThreadId"
      >
        <CornerUpLeft class="h-3 w-3" />
        <span>{{ t('cockpit.relation.parentThread', { id: parentThreadLabel }) }}</span>
      </span>

      <button
        v-if="!expanded && childConversationCount > 0 && firstChildConversationId"
        type="button"
        class="relation-chip border border-amber-500/50 bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 dark:text-amber-400"
        :title="firstChildConversationId"
        @click="navigateConversation(firstChildConversationId)"
      >
        <GitBranch class="h-3 w-3" />
        <span>{{ t('cockpit.relation.children', { count: String(childConversationCount) }) }}</span>
      </button>
    </div>

    <div v-if="expanded" class="main-conversation-detail mb-3 border border-border bg-background/60 p-2.5">
      <div class="conversation-section-head flex items-center gap-2 text-[10px] font-bold uppercase tracking-[0.04em] text-muted-foreground">
        <span>{{ t('cockpit.mainConversation') }}</span>
      </div>
      <div class="main-conversation-turns mt-1.5 flex flex-col gap-1.5">
        <div
          v-for="(turn, index) in mainConversationTurns"
          :key="`${index}-${turn}`"
          :class="[
            'main-conversation-turn text-xs font-semibold leading-relaxed text-foreground',
            { 'main-conversation-turn--single': mainConversationTurns.length === 1 },
          ]"
        >
          {{ turn }}
        </div>
      </div>
      <div class="mt-2 grid grid-cols-2 gap-2">
        <div v-for="row in mainDetailRows" :key="row.label" class="min-w-0 border-t border-dashed border-border pt-1.5">
          <span class="block truncate text-[10px] font-semibold text-muted-foreground">{{ row.label }}</span>
          <strong class="mt-0.5 block truncate text-[11px] font-bold text-foreground/85">{{ row.value }}</strong>
        </div>
      </div>
    </div>

    <!-- Row 2: Model + Channel chips (collapsed) -->
    <div v-if="!expanded" class="flex flex-wrap items-center gap-2">
      <span class="mr-2 min-w-0 max-w-full truncate text-sm font-medium">{{ conversation.lastModel }}</span>
      <button
        v-for="channel in visibleChannels"
        :key="channel.index"
        type="button"
        :title="getChannelTooltip(channel)"
        class="channel-chip inline-flex max-w-[160px] items-center gap-1 truncate border px-2 py-0.5 text-[10px] font-medium transition-colors"
        :class="channelChipClass(channel)"
        @click.stop="handleQuickOverride(channel)"
      >
        <span class="truncate">{{ channel.name }}</span>
        <Check v-if="channel.index === conversation.currentChannel" class="h-2.5 w-2.5 shrink-0" />
        <span v-else-if="channel.index === nextChannel" class="next-label shrink-0">
          | {{ nextChannelCircuitOpen ? 'TRIPPED' : 'NEXT' }}
        </span>
      </button>
      <button
        v-if="hiddenCount > 0"
        type="button"
        class="px-1.5 py-0.5 text-[10px] text-muted-foreground hover:text-foreground"
        @click.stop="emit('toggleExpand')"
      >
        +{{ hiddenCount }}
      </button>
    </div>

    <!-- Expanded: Override alert -->
    <div v-if="expanded && hasOverride" class="override-alert mt-3 border border-amber-500/70 bg-amber-500/10 p-2">
      <div class="flex items-center gap-2">
        <span class="alert-bang">[!]</span>
        <span v-if="override?.isPerpetual" class="text-xs text-amber-600 dark:text-amber-400">
          {{ t('cockpit.overrideActivePerpetual') }}
        </span>
        <span v-else class="text-xs text-amber-600 dark:text-amber-400">
          {{ t('cockpit.overrideActive', { time: remainingTime }) }}
        </span>
        <div class="flex-1" />
        <Button variant="ghost" size="sm" class="h-6 px-2 text-xs" @click.stop="emit('removeOverride', conversation.id)">
          {{ t('cockpit.restoreDefault') }}
        </Button>
      </div>
    </div>

    <!-- Expanded: Full channel sequence -->
    <div v-if="expanded" class="main-routing-section mt-3">
      <div class="mb-1 text-xs text-muted-foreground">{{ t('cockpit.mainRouting') }} · {{ conversation.lastModel }}</div>
      <ConversationChannelSequence
        :channels="channelSequence"
        :current-channel="conversation.currentChannel"
        :next-channel="nextChannel"
        :next-channel-circuit-open="nextChannelCircuitOpen"
        :override-active="hasOverride"
        @move-to-top="handleMoveToTop"
        @demote="handleDemote"
      />
    </div>

    <div v-if="expanded && showSubagentSection" class="subagent-expanded-section mt-3 border-t border-dashed border-border pt-2" @click.stop>
      <div v-if="childConversationCount > 0 && firstChildConversationId" class="relation-row mb-2 flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          class="relation-chip border border-amber-500/50 bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 dark:text-amber-400"
          :title="firstChildConversationId"
          @click="navigateConversation(firstChildConversationId)"
        >
          <GitBranch class="h-3 w-3" />
          <span>{{ t('cockpit.relation.children', { count: String(childConversationCount) }) }}</span>
        </button>
      </div>

      <div class="subagent-summary border border-amber-500/50 bg-amber-500/10 p-2 text-amber-500">
        <div class="flex items-center gap-1.5 text-xs font-bold">
          <GitBranch class="h-3.5 w-3.5" />
          <span>{{ t('cockpit.board.subagents') }}</span>
          <strong>{{ displaySubagentCount }}</strong>
        </div>
        <div class="mt-1 flex min-w-0 items-center gap-1.5 text-[10px] text-muted-foreground">
          <span v-if="subagentSummary.total > 0" class="truncate">
            {{ subagentSummary.streaming }} {{ t('cockpit.board.streaming') }} / {{ subagentSummary.active }} {{ t('cockpit.board.active') }} / {{ subagentSummary.idle }} {{ t('cockpit.board.idle') }}
          </span>
          <span v-else class="truncate">{{ subagentRouteLabel }}</span>
        </div>
      </div>

      <div v-if="subagents.length > 0" class="subagent-list mt-2 border border-border bg-background/60">
        <div class="flex items-center justify-between gap-2 border-b border-border px-2 py-1.5 text-[10px] font-semibold text-muted-foreground">
          <span>{{ t('cockpit.board.subagents') }}</span>
          <span>{{ subagentSummary.streaming }} {{ t('cockpit.board.streaming') }} · {{ subagentSummary.active }} {{ t('cockpit.board.active') }} · {{ subagentSummary.idle }} {{ t('cockpit.board.idle') }}</span>
        </div>
        <div
          v-for="agent in visibleSubagents"
          :key="agent.id"
          class="grid grid-cols-[8px_minmax(0,1fr)_auto] items-center gap-2 border-b border-border/60 px-2 py-1.5 last:border-b-0"
        >
          <span class="h-2 w-2 rounded-full" :class="subagentStatusClass(agent.status)" />
          <div class="min-w-0">
            <div class="truncate text-xs font-semibold text-foreground">{{ agent.title || agent.userId }}</div>
            <div class="truncate text-[10px] text-muted-foreground">{{ agent.lastModel }} · {{ agent.channelName || `Channel ${agent.currentChannel}` }}</div>
          </div>
          <span class="border border-border bg-muted/30 px-1.5 py-0.5 text-[10px] font-semibold text-muted-foreground">{{ agent.status }}</span>
        </div>
        <button
          v-if="subagents.length > visibleSubagents.length"
          type="button"
          class="w-full px-2 py-1 text-left text-[10px] text-muted-foreground hover:text-foreground"
          @click.stop="emit('toggleExpand')"
        >
          +{{ subagents.length - visibleSubagents.length }} more
        </button>
      </div>

      <!-- Subagent Routing：为主对话与 subagent 分别指定渠道 -->
      <div class="subagent-routing mt-3">
        <div class="mb-1 flex items-center">
          <span class="text-xs text-muted-foreground">{{ t('cockpit.subagentRouting') }}</span>
          <span v-if="hasSubagentOverride" class="ml-2 text-xs text-amber-500">[{{ t('cockpit.subagentOverride') }}]</span>
          <span v-else class="ml-2 text-xs text-muted-foreground">[{{ t('cockpit.subagentFollowMain') }}]</span>
          <span class="flex-1" />
          <Button v-if="hasSubagentOverride" variant="ghost" size="sm" class="h-6 px-2 text-xs" @click.stop="handleClearSubagentOverride">
            {{ t('cockpit.subagentClearOverride') }}
          </Button>
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
    <div v-if="conversation.rawUserId" class="raw-user-id mt-2 flex items-center gap-1 border-t border-dashed border-border pt-2">
      <button
        type="button"
        class="raw-user-id-text min-w-0 flex-1 truncate text-left font-mono text-xs text-muted-foreground"
        @click.stop="copyRawUserId"
      >
        {{ conversation.rawUserId }}
      </button>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button variant="ghost" size="icon-sm" class="copy-btn h-6 w-6" :aria-label="t('cockpit.copyRawUserId')" @click.stop="copyRawUserId">
            <Copy class="h-3 w-3" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top">
          {{ t('cockpit.copyRawUserId') }}
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<style scoped>
.conversation-card {
  cursor: pointer;
  position: relative;
  border-width: 2px;
  border-color: color-mix(in srgb, var(--color-foreground) 55%, transparent);
  border-radius: 0;
  box-shadow: 4px 4px 0 0 color-mix(in srgb, var(--color-foreground) 80%, transparent);
  background:
    radial-gradient(circle, color-mix(in srgb, var(--color-foreground) 12%, transparent) 1px, transparent 1px) 0 0 / 16px 16px,
    var(--color-card);
  transition: all 0.1s ease;
}

.conversation-card::before,
.conversation-card::after {
  content: '';
  position: absolute;
  width: 14px;
  height: 14px;
  pointer-events: none;
  opacity: 0.45;
}

.conversation-card::before {
  left: 6px;
  top: 6px;
  border-left: 2px solid var(--ccx-kind-color);
  border-top: 2px solid var(--ccx-kind-color);
}

.conversation-card::after {
  right: 6px;
  bottom: 6px;
  border-bottom: 2px solid var(--ccx-kind-color);
  border-right: 2px solid var(--ccx-kind-color);
}

.conversation-card:hover {
  border-color: var(--ccx-kind-color);
  box-shadow: 5px 5px 0 0 var(--ccx-kind-color);
  transform: translate(-1px, -1px);
}

.conversation-card:active {
  box-shadow: 2px 2px 0 0 color-mix(in srgb, var(--color-foreground) 80%, transparent);
  transform: translate(1px, 1px);
}

.conversation-card.override-active {
  border-color: var(--color-warning);
  box-shadow: 4px 4px 0 0 var(--color-warning);
}

.conversation-card.override-active:hover {
  border-color: var(--color-warning);
  box-shadow: 5px 5px 0 0 var(--color-warning);
}

.status-led {
  display: inline-block;
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 999px;
}

.status-led--streaming {
  background: #22d3ee;
  animation: ccx-led-pulse 1.4s ease-in-out infinite;
}

.status-led--active {
  background: #10b981;
  box-shadow: 0 0 4px 1px rgba(16, 185, 129, 0.45);
}

.status-led--idle {
  background: var(--color-muted-foreground);
}

.kind-chip,
.channel-chip,
.sequence-badge {
  border-radius: 0;
}

.display-label-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.main-conversation-detail,
.subagent-summary,
.subagent-list {
  border-radius: 0;
}

.main-conversation-turn {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.main-conversation-turn--single {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 5;
}

.next-label {
  display: inline-block;
  margin-left: 4px;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.05em;
}

.next-channel-chip {
  animation: ccx-breathe 2s ease-in-out infinite;
}

.relation-chip {
  display: inline-flex;
  min-height: 22px;
  max-width: 100%;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}

.override-alert {
  border-radius: 0;
}

.alert-bang {
  color: var(--color-warning);
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.1em;
  animation: ccx-alert-blink 0.8s step-end infinite;
}

.raw-user-id {
  opacity: 0.65;
}

.raw-user-id:hover {
  opacity: 0.95;
}

.copy-btn {
  opacity: 0.55;
}

.raw-user-id:hover .copy-btn {
  opacity: 1;
}

@keyframes ccx-led-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 5px rgba(34, 211, 238, 0.55); }
  50% { opacity: 0.4; box-shadow: 0 0 12px rgba(34, 211, 238, 0.9); }
}

@keyframes ccx-breathe {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

@keyframes ccx-alert-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.2; }
}
</style>
