<script setup lang="ts">
import { computed, reactive, onBeforeUnmount } from 'vue'
import { Loader2, CheckCircle2, XCircle, ArrowRight, Clock } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'
import type { CapabilityProtocolJobResult, CapabilityModelJobResult } from '@/services/admin-api'

interface Props {
  test: CapabilityProtocolJobResult
  pendingText?: string
  retryEnabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  pendingText: '',
  retryEnabled: true,
})

const emit = defineEmits<{
  retryModel: [protocol: string, model: string]
}>()

const { t } = useLanguage()

const modelResults = computed(() => props.test.modelResults ?? [])

// 是否显示 pending placeholder
const shouldShowPendingPlaceholder = computed(() => {
  if (modelResults.value.length > 0) return false
  if (props.test.lifecycle === 'cancelled' || props.test.outcome === 'cancelled') return false
  return props.test.lifecycle === 'active' || props.test.lifecycle === 'pending' || props.test.status === 'running' || props.test.status === 'queued'
})

// 是否显示"details unavailable"
const shouldShowDetailsUnavailable = computed(() => {
  if (modelResults.value.length > 0) return false
  if (shouldShowPendingPlaceholder.value) return false
  return true
})

function getModelBadgeClasses(result: CapabilityModelJobResult) {
  const base = 'relative inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-mono cursor-default select-none'
  if (result.status === 'success') return `${base} bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/20`
  if (result.status === 'failed') {
    const retry = props.retryEnabled && canRetry(result)
    return `${base} bg-rose-500/15 text-rose-700 dark:text-rose-300 border border-rose-500/20 ${retry ? 'cursor-pointer hover:border-blue-500/40 hover:bg-blue-500/10' : ''}`
  }
  if (isModelCancelledOrSkipped(result)) {
    const retry = props.retryEnabled && canRetry(result)
    return `${base} bg-muted/20 text-muted-foreground border border-border ${retry ? 'cursor-pointer hover:border-blue-500/40 hover:bg-blue-500/10' : ''}`
  }
  if (result.status === 'running') return `${base} bg-blue-500/15 text-blue-700 dark:text-blue-300 border border-blue-500/20`
  if (result.status === 'queued') return `${base} bg-muted/30 text-muted-foreground border border-border`
  // idle / skipped / cancelled
  const retry = props.retryEnabled && canRetry(result)
  return `${base} bg-muted/20 text-muted-foreground border border-border ${retry ? 'cursor-pointer hover:border-blue-500/40 hover:bg-blue-500/10' : ''}`
}

function canRetry(result: CapabilityModelJobResult) {
  return result.status === 'failed' || result.status === 'idle' || result.status === 'skipped' || result.lifecycle === 'cancelled' || result.outcome === 'cancelled'
}

function getRetryHint(result: CapabilityModelJobResult) {
  return result.status === 'idle'
    ? t('capability.testModel')
    : t('capability.retryModel')
}

function isRedirected(result: CapabilityModelJobResult) {
  return result.actualModel && result.actualModel !== result.model
}

function getModelTooltipView(result: CapabilityModelJobResult) {
  if (result.status === 'success') return 'success'
  if (!isModelCancelledOrSkipped(result) && (result.status === 'running' || result.status === 'queued')) return 'pending'
  return 'failed'
}

function isModelCancelledOrSkipped(result: CapabilityModelJobResult) {
  return result.lifecycle === 'cancelled' || result.outcome === 'cancelled' || result.status === 'skipped'
}

function isModelPending(result: CapabilityModelJobResult) {
  return !isModelCancelledOrSkipped(result) && (result.status === 'running' || result.status === 'queued')
}

function formatCodexImageGeneration(result: CapabilityModelJobResult): string {
  const probe = result.codexImageGeneration
  if (!probe) return '-'
  const total = probe.supportedKeys + probe.unsupportedKeys + probe.inconclusiveKeys
  const summary = t('capability.codexImageGenerationSummary', {
    supported: probe.supportedKeys,
    total,
  })
  return probe.compatibleViaStrip
    ? `${summary} · ${t('capability.codexImageGenerationStripped')}`
    : summary
}

function formatStreaming(result: CapabilityModelJobResult) {
  if (result.streamingSupported === true) return t('capability.supported')
  if (result.streamingSupported === false) return t('capability.unsupported')
  return '—'
}

function getModelTooltipLatencyText(result: CapabilityModelJobResult) {
  return result.latency >= 0 ? `${result.latency}ms` : '—'
}

function handleBadgeClick(result: CapabilityModelJobResult) {
  if (canRetry(result) && props.retryEnabled) {
    emit('retryModel', props.test.protocol, result.model)
  }
}

// 浮动 tooltip：使用 fixed 定位 + 动态计算坐标，逃出父级 overflow:hidden
const tooltipState = reactive({
  visible: false,
  hoveredKey: '',
  x: 0,
  y: 0,
  placement: 'top' as 'top' | 'bottom',
})

function showTooltip(event: MouseEvent | FocusEvent, result: CapabilityModelJobResult) {
  const target = event.currentTarget as HTMLElement | null
  if (!target) return
  const rect = target.getBoundingClientRect()
  const tooltipWidth = 240
  const tooltipHeight = 132
  const margin = 8
  // 默认在 badge 上方
  let placement: 'top' | 'bottom' = 'top'
  let x = rect.left + rect.width / 2
  let y = rect.top - margin
  // 上方空间不够就改到下方
  if (rect.top - tooltipHeight - margin < 0) {
    placement = 'bottom'
    y = rect.bottom + margin
  }
  // 横向贴边修正
  const halfWidth = tooltipWidth / 2
  if (x - halfWidth < margin) x = halfWidth + margin
  if (x + halfWidth > window.innerWidth - margin) x = window.innerWidth - halfWidth - margin
  tooltipState.placement = placement
  tooltipState.x = x
  tooltipState.y = y
  tooltipState.hoveredKey = `${props.test.protocol}-${result.model}`
  tooltipState.visible = true
}

function hideTooltip() {
  tooltipState.visible = false
  tooltipState.hoveredKey = ''
}

onBeforeUnmount(hideTooltip)
</script>

<template>
  <div>
    <!-- pending placeholder -->
    <div v-if="shouldShowPendingPlaceholder" class="flex items-center gap-2 py-2">
      <Loader2 class="h-4 w-4 animate-spin text-primary" />
      <span class="text-xs text-muted-foreground">{{ pendingText || t('capability.modelQueued') }}</span>
    </div>

    <!-- details unavailable -->
    <div v-else-if="shouldShowDetailsUnavailable" class="py-1 text-xs text-muted-foreground">
      {{ t('capability.modelDetailsUnavailable') }}
    </div>

    <!-- model badges -->
    <div v-else class="flex flex-wrap gap-1.5 py-1">
      <div
        v-for="result in modelResults"
        :key="`${test.protocol}-${result.model}`"
        :class="getModelBadgeClasses(result)"
        tabindex="0"
        @click="handleBadgeClick(result)"
        @mouseenter="showTooltip($event, result)"
        @mouseleave="hideTooltip"
        @focus="showTooltip($event, result)"
        @blur="hideTooltip"
      >
        <span>{{ result.model }}</span>
        <ArrowRight v-if="isRedirected(result)" class="h-3 w-3 text-muted-foreground" />
        <CheckCircle2
          v-if="result.status === 'success'"
          class="h-3 w-3"
        />
        <XCircle
          v-else-if="result.status === 'failed'"
          class="h-3 w-3"
        />
        <Loader2
          v-else-if="isModelPending(result)"
          class="h-3 w-3 animate-spin"
        />
        <Clock
          v-else
          class="h-3 w-3"
        />
      </div>
    </div>

    <!-- 浮动 tooltip：使用 fixed 定位 + 动态计算坐标，逃出父级 overflow:hidden -->
    <Teleport to="body">
      <div
        v-if="tooltipState.visible"
        class="model-tooltip-content"
        :class="`model-tooltip-${tooltipState.placement}`"
        :style="{ left: `${tooltipState.x}px`, top: `${tooltipState.y}px` }"
      >
        <template v-for="result in modelResults" :key="`tt-${test.protocol}-${result.model}`">
          <div v-if="tooltipState.hoveredKey === `${test.protocol}-${result.model}`">
            <div class="model-tooltip-title">{{ result.model }}</div>
            <div v-if="isRedirected(result)" class="model-tooltip-row">
              <span class="model-tooltip-label">{{ t('capability.actualModel') }}</span>
              <span class="model-tooltip-value">{{ result.actualModel }}</span>
            </div>
            <template v-if="getModelTooltipView(result) === 'success'">
              <div class="model-tooltip-row">
                <span class="model-tooltip-label">{{ t('capability.tooltipLatency') }}</span>
                <span class="model-tooltip-value">{{ getModelTooltipLatencyText(result) }}</span>
              </div>
              <div class="model-tooltip-row">
                <span class="model-tooltip-label">{{ t('capability.tooltipStreaming') }}</span>
                <span class="model-tooltip-value">{{ formatStreaming(result) }}</span>
              </div>
            </template>
            <div class="model-tooltip-row">
              <span class="model-tooltip-label">{{ t('capability.modelStatus') }}</span>
              <span class="model-tooltip-value">{{ result.status }}</span>
            </div>
            <div v-if="result.codexImageGeneration?.tested" class="model-tooltip-row">
              <span class="model-tooltip-label">{{ t('capability.codexImageGeneration') }}</span>
              <span class="model-tooltip-value">{{ formatCodexImageGeneration(result) }}</span>
            </div>
            <div v-if="getModelTooltipView(result) === 'failed' && result.error" class="model-tooltip-error">
              {{ result.error }}
            </div>
            <div v-if="getModelTooltipView(result) === 'failed' && canRetry(result)" class="model-tooltip-retry">
              {{ getRetryHint(result) }}
            </div>
          </div>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.model-tooltip-content {
  position: fixed;
  z-index: 9999;
  width: max-content;
  max-width: 280px;
  padding: 8px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-popover);
  color: var(--color-popover-foreground);
  box-shadow: 0 12px 32px rgb(0 0 0 / 18%);
  font-size: 12px;
  line-height: 1.5;
  pointer-events: none;
}

.model-tooltip-top {
  transform: translate(-50%, -100%);
}

.model-tooltip-bottom {
  transform: translate(-50%, 0);
}

.model-tooltip-content::after {
  position: absolute;
  left: 50%;
  width: 8px;
  height: 8px;
  border-right: 1px solid var(--color-border);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-popover);
  content: '';
}

.model-tooltip-top::after {
  top: 100%;
  transform: translate(-50%, -50%) rotate(45deg);
}

.model-tooltip-bottom::after {
  bottom: 100%;
  transform: translate(-50%, 50%) rotate(225deg);
}

.model-tooltip-title {
  margin-bottom: 4px;
  font-weight: 600;
}

.model-tooltip-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.model-tooltip-label {
  color: var(--color-muted-foreground);
  white-space: nowrap;
}

.model-tooltip-value {
  text-align: right;
  word-break: break-word;
}

.model-tooltip-error {
  margin-top: 4px;
  color: var(--color-destructive);
  word-break: break-word;
}

.model-tooltip-retry {
  margin-top: 4px;
  color: var(--color-muted-foreground);
}
</style>
