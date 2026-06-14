<template>
  <div class="model-stats-chart">
    <!-- Header -->
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div class="flex items-center gap-2">
        <!-- Duration selector -->
        <div class="inline-flex rounded-md border border-border divide-x divide-border">
          <button
            v-for="opt in durationOptions"
            :key="opt.value"
            type="button"
            class="px-2 py-1 text-[11px] font-semibold transition-colors hover:bg-accent/40 disabled:opacity-50"
            :class="{ 'bg-accent text-accent-foreground': selectedDuration === opt.value }"
            :disabled="isLoading"
            @click="selectedDuration = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>

        <button
          type="button"
          class="p-1 text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
          :disabled="isLoading"
          @click="refreshData"
        >
          <svg v-if="!isLoading" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <svg v-else class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </button>
      </div>

      <!-- View switcher -->
      <div class="inline-flex rounded-md border border-border divide-x divide-border">
        <button
          v-for="view in viewOptions"
          :key="view.value"
          type="button"
          class="px-2 py-1 text-[11px] font-semibold transition-colors hover:bg-accent/40 disabled:opacity-50 flex items-center gap-1"
          :class="{ 'bg-accent text-accent-foreground': selectedView === view.value }"
          :disabled="isLoading"
          @click="selectedView = view.value"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path v-if="view.value === 'requests'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z" />
            <path v-else-if="view.value === 'tokens'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 8v8m-4-5v5m-4-2v2m-2 4h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ view.label }}
        </button>
      </div>
    </div>

    <!-- Compact summary (top models) -->
    <div v-if="topModels.length" class="flex flex-wrap items-center gap-3 mb-2 text-xs bg-secondary/20 dark:bg-secondary/10 rounded px-2 py-1.5">
      <span v-for="(m, i) in topModels" :key="m.name" class="flex items-center gap-1">
        <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: MODEL_COLORS[i % MODEL_COLORS.length] }" />
        <span class="font-medium">{{ m.name }}</span>
        <span class="text-muted-foreground">{{ formatNumber(m.count) }} {{ t('chart.requestUnit') }}</span>
      </span>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center" style="height: 200px">
      <div class="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!hasData"
      class="flex flex-col items-center justify-center text-muted-foreground"
      style="height: 200px"
    >
      <div class="text-2xl mb-2 opacity-40">&#x1F4CA;</div>
      <div class="text-xs">{{ t('chart.noModelData') }}</div>
    </div>

    <!-- Chart -->
    <div v-else>
      <VueApexCharts
        ref="chartRef"
        :key="`model-stats-${selectedView}`"
        type="area"
        :height="200"
        :options="chartOptions"
        :series="chartSeries"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import VueApexCharts from 'vue3-apexcharts'
import type { ApexOptions } from 'apexcharts'
import { useTheme } from '@/composables/useTheme'
import { useI18n } from '@/i18n'
import type { ModelHistoryDataPoint } from '@/services/admin-api'

const props = withDefaults(
  defineProps<{
    apiType: 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
  }>(),
  {},
)

const emit = defineEmits<{
  refresh: [duration: string]
}>()

type Duration = '1h' | '6h' | '24h' | 'today' | '7d' | '30d'
type ViewMode = 'requests' | 'tokens' | 'cache'

// Persisted preferences
const storageKey = (key: string) => `modelStats:${props.apiType}:${key}`
const loadPref = (apiType: string) => ({
  duration: (localStorage.getItem(`modelStats:${apiType}:duration`) as Duration) || '6h',
  view: (localStorage.getItem(`modelStats:${apiType}:view`) as ViewMode) || 'requests'
})

const { theme } = useTheme()
const { t } = useI18n()

const isDark = computed(() => {
  if (theme.value === 'dark') return true
  if (theme.value === 'auto') return window.matchMedia('(prefers-color-scheme: dark)').matches
  return false
})

const saved = loadPref(props.apiType)
const selectedDuration = ref<Duration>(saved.duration)
const selectedView = ref<ViewMode>(saved.view)
const isLoading = ref(false)
const historyData = ref<{ models: Record<string, ModelHistoryDataPoint[]> } | null>(null)

const textColor = computed(() => (isDark.value ? '#9ca3af' : '#6b7280'))
const gridBorder = computed(() => (isDark.value ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'))

const MODEL_COLORS = [
  '#3b82f6', '#10b981', '#f97316', '#8b5cf6', '#ef4444',
  '#06b6d4', '#ec4899', '#84cc16', '#f59e0b', '#6366f1',
]

const durationOptions = computed(() => [
  { label: t('chart.1h'), value: '1h' as Duration },
  { label: t('chart.6h'), value: '6h' as Duration },
  { label: t('chart.24h'), value: '24h' as Duration },
  { label: t('chart.today'), value: 'today' as Duration },
  { label: t('chart.7d'), value: '7d' as Duration },
  { label: t('chart.30d'), value: '30d' as Duration },
])

const viewOptions = computed(() => [
  { label: t('chart.traffic'), value: 'requests' as ViewMode },
  { label: t('chart.tokens'), value: 'tokens' as ViewMode },
  { label: 'Cache', value: 'cache' as ViewMode },
])

const sortedModels = computed(() => {
  if (!historyData.value?.models) return []
  return Object.entries(historyData.value.models)
    .map(([name, points]) => ({
      name,
      points,
      totalRequests: points.reduce((s, p) => s + p.requestCount, 0),
      totalTokens: points.reduce((s, p) => s + p.inputTokens + p.outputTokens, 0)
    }))
    .sort((a, b) => b.totalRequests - a.totalRequests)
})

const topModels = computed(() =>
  sortedModels.value.slice(0, 5).map(m => ({ name: m.name, count: m.totalRequests })),
)

const hasData = computed(() => sortedModels.value.some(m => m.totalRequests > 0))

const xLabelFormat = computed(() =>
  selectedDuration.value === '7d' || selectedDuration.value === '30d' ? 'MM-dd HH:mm' : 'HH:mm'
)

const chartOptions = computed<ApexOptions>(() => ({
  chart: {
    toolbar: { show: false },
    zoom: { enabled: false },
    background: 'transparent',
    fontFamily: 'inherit',
    animations: { enabled: true, speed: 400 }
  },
  theme: { mode: isDark.value ? 'dark' : 'light' },
  colors: MODEL_COLORS.slice(0, sortedModels.value.length),
  fill: {
    type: 'gradient',
    gradient: { shadeIntensity: 1, opacityFrom: 0.3, opacityTo: 0.05, stops: [0, 90, 100] }
  },
  dataLabels: { enabled: false },
  stroke: { curve: 'smooth', width: 2 },
  grid: { borderColor: gridBorder.value, padding: { left: 10, right: 10 } },
  xaxis: {
    type: 'datetime',
    labels: {
      datetimeUTC: false,
      format: xLabelFormat.value,
      style: { fontSize: '10px', colors: textColor.value }
    },
    axisBorder: { show: false },
    axisTicks: { show: false }
  },
  yaxis: {
    labels: {
      formatter: (val: number) => selectedView.value === 'requests' ? Math.round(val).toString() : formatNumber(val),
      style: { fontSize: '11px', colors: textColor.value }
    },
    min: 0
  },
  tooltip: {
    x: { format: 'MM-dd HH:mm' },
    y: {
      formatter: (val: number) => selectedView.value === 'requests'
        ? `${Math.round(val)} ${t('chart.requestUnit')}`
        : `${formatNumber(val)} ${t('chart.tokenUnit')}`
    }
  },
  legend: {
    show: true,
    position: 'top',
    horizontalAlign: 'right',
    fontSize: '11px',
    markers: { size: 4 },
    labels: { colors: textColor.value }
  }
}))

const chartSeries = computed(() => {
  return sortedModels.value.map(m => ({
    name: m.name,
    data: m.points.map(p => ({
      x: new Date(p.timestamp).getTime(),
      y: selectedView.value === 'requests'
        ? p.requestCount
        : selectedView.value === 'tokens'
          ? p.inputTokens + p.outputTokens
          : (p.cacheReadTokens || 0) + (p.cacheCreationTokens || 0)
    }))
  }))
})

function formatNumber(num: number): string {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toFixed(0)
}

// Expose data for parent to update
const updateData = (models: Record<string, ModelHistoryDataPoint[]>) => {
  historyData.value = { models }
}

const refreshData = () => {
  emit('refresh', selectedDuration.value)
}

const setLoading = (loading: boolean) => {
  isLoading.value = loading
}

watch(selectedDuration, (v) => {
  localStorage.setItem(storageKey('duration'), v)
  refreshData()
})

watch(selectedView, (v) => {
  localStorage.setItem(storageKey('view'), v)
})

watch(() => props.apiType, (t) => {
  const p = loadPref(t)
  selectedDuration.value = p.duration as Duration
  selectedView.value = p.view as ViewMode
  refreshData()
})

onMounted(() => {
  refreshData()
})

defineExpose({
  updateData,
  setLoading,
  refreshData,
})
</script>
