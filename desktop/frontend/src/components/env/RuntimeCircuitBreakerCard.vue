<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert } from '@/components/ui/alert'
import { Save, RefreshCw, Zap } from 'lucide-vue-next'
import { useStatus } from '@/composables/useStatus'
import { useLanguage } from '@/composables/useLanguage'
import { GetAdminAccessKey } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

const { status } = useStatus()
const { t } = useLanguage()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')
let messageTimer: ReturnType<typeof setTimeout> | null = null

const activePreset = ref('balanced')
const form = reactive({
  windowSize: 10,
  failureThreshold: 0.5,
  consecutiveFailuresThreshold: 3,
  streamFirstContentTimeoutMs: 30000,
  streamInactivityTimeoutMs: 20000,
  streamToolCallIdleTimeoutMs: 120000,
})

// 工具调用 idle 预设按低速 5 TPS 粗估：60/120/300s 分别预留约 300/600/1500 token 的参数生成窗口。
const presets = [
  { key: 'gentle', labelKey: 'env.runtimeCbPresetGentle' as const, windowSize: 20, failureThreshold: 0.70, consecutiveFailuresThreshold: 5, streamFirstContentTimeoutMs: 90000, streamInactivityTimeoutMs: 90000, streamToolCallIdleTimeoutMs: 300000 },
  { key: 'balanced', labelKey: 'env.runtimeCbPresetBalanced' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, streamFirstContentTimeoutMs: 60000, streamInactivityTimeoutMs: 60000, streamToolCallIdleTimeoutMs: 180000 },
  { key: 'aggressive', labelKey: 'env.runtimeCbPresetAggressive' as const, windowSize: 5, failureThreshold: 0.30, consecutiveFailuresThreshold: 2, streamFirstContentTimeoutMs: 30000, streamInactivityTimeoutMs: 30000, streamToolCallIdleTimeoutMs: 60000 },
  { key: 'custom', labelKey: 'env.runtimeCbPresetCustom' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, streamFirstContentTimeoutMs: 60000, streamInactivityTimeoutMs: 60000, streamToolCallIdleTimeoutMs: 180000 },
]

// 历史图片轮次限制
const historicalImageLimit = ref(0)

const matchPreset = () => {
  for (const p of presets) {
    if (p.key === 'custom') continue
    if (form.windowSize === p.windowSize && form.failureThreshold === p.failureThreshold && form.consecutiveFailuresThreshold === p.consecutiveFailuresThreshold && form.streamFirstContentTimeoutMs === p.streamFirstContentTimeoutMs && form.streamInactivityTimeoutMs === p.streamInactivityTimeoutMs && form.streamToolCallIdleTimeoutMs === p.streamToolCallIdleTimeoutMs) {
      activePreset.value = p.key
      return
    }
  }
  activePreset.value = 'custom'
}

const applyPreset = (preset: typeof presets[number]) => {
  if (preset.key === 'custom') return
  form.windowSize = preset.windowSize
  form.failureThreshold = preset.failureThreshold
  form.consecutiveFailuresThreshold = preset.consecutiveFailuresThreshold
  form.streamFirstContentTimeoutMs = preset.streamFirstContentTimeoutMs
  form.streamInactivityTimeoutMs = preset.streamInactivityTimeoutMs
  form.streamToolCallIdleTimeoutMs = preset.streamToolCallIdleTimeoutMs
  activePreset.value = preset.key
}

const onSliderChange = (field: string, event: Event) => {
  const val = Number((event.target as HTMLInputElement).value)
  if (field === 'failureThreshold') {
    form.failureThreshold = Math.round(val * 100) / 100
  } else if (field === 'windowSize') {
    form.windowSize = val
  } else if (field === 'consecutiveFailuresThreshold') {
    form.consecutiveFailuresThreshold = val
  } else if (field === 'streamFirstContentTimeoutMs') {
    form.streamFirstContentTimeoutMs = val
  } else if (field === 'streamInactivityTimeoutMs') {
    form.streamInactivityTimeoutMs = val
  } else if (field === 'streamToolCallIdleTimeoutMs') {
    form.streamToolCallIdleTimeoutMs = val
  }
  matchPreset()
}

const clearMessages = () => {
  error.value = ''
  success.value = ''
  if (messageTimer) {
    clearTimeout(messageTimer)
    messageTimer = null
  }
}

const showMessage = (msg: string, type: 'success' | 'error') => {
  clearMessages()
  if (type === 'success') {
    success.value = msg
  } else {
    error.value = msg
  }
  messageTimer = setTimeout(clearMessages, 5000)
}

const buildApiUrl = async (path: string): Promise<string | null> => {
  if (!status.value.url) return null
  return `${status.value.url}${path}`
}

const fetchConfig = async () => {
  const url = await buildApiUrl('/api/settings/circuit-breaker')
  if (!url) return

  loading.value = true
  clearMessages()
  try {
    const adminKey = await GetAdminAccessKey()
    const resp = await fetch(url, {
      headers: { 'x-api-key': adminKey },
    })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    form.windowSize = data.windowSize ?? 10
    form.failureThreshold = data.failureThreshold ?? 0.5
    form.consecutiveFailuresThreshold = data.consecutiveFailuresThreshold ?? 3
    form.streamFirstContentTimeoutMs = data.streamFirstContentTimeoutMs && data.streamFirstContentTimeoutMs >= 5000 ? data.streamFirstContentTimeoutMs : 60000
    form.streamInactivityTimeoutMs = data.streamInactivityTimeoutMs && data.streamInactivityTimeoutMs >= 1000 ? data.streamInactivityTimeoutMs : 60000
    form.streamToolCallIdleTimeoutMs = data.streamToolCallIdleTimeoutMs && data.streamToolCallIdleTimeoutMs >= 30000 ? data.streamToolCallIdleTimeoutMs : 180000
    matchPreset()
  } catch (e) {
    showMessage(t('env.runtimeCbLoadFailed', { error: e instanceof Error ? e.message : String(e) }), 'error')
  } finally {
    loading.value = false
  }
}

const fetchHistoricalImageLimit = async () => {
  const url = await buildApiUrl('/api/settings/historical-image-turn-limit')
  if (!url) return

  try {
    const adminKey = await GetAdminAccessKey()
    const resp = await fetch(url, {
      headers: { 'x-api-key': adminKey },
    })
    if (resp.ok) {
      const data = await resp.json()
      historicalImageLimit.value = data.historicalImageTurnLimit ?? 0
    }
  } catch {
    // 非关键功能，静默忽略
  }
}

const saveConfig = async () => {
  const cbUrl = await buildApiUrl('/api/settings/circuit-breaker')
  const imgUrl = await buildApiUrl('/api/settings/historical-image-turn-limit')
  if (!cbUrl) {
    showMessage(t('env.runtimeCbNoBackend'), 'error')
    return
  }

  saving.value = true
  clearMessages()
  try {
    const adminKey = await GetAdminAccessKey()
    const promises: Promise<Response>[] = [
      fetch(cbUrl, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'x-api-key': adminKey },
        body: JSON.stringify({
          windowSize: form.windowSize,
          failureThreshold: form.failureThreshold,
          consecutiveFailuresThreshold: form.consecutiveFailuresThreshold,
          streamFirstContentTimeoutMs: form.streamFirstContentTimeoutMs,
          streamInactivityTimeoutMs: form.streamInactivityTimeoutMs,
          streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutMs,
        }),
      }),
    ]
    if (imgUrl) {
      promises.push(
        fetch(imgUrl, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', 'x-api-key': adminKey },
          body: JSON.stringify({ limit: historicalImageLimit.value }),
        }),
      )
    }
    const results = await Promise.all(promises)
    for (const resp of results) {
      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}))
        throw new Error(body.error || `HTTP ${resp.status}`)
      }
    }
    showMessage(t('env.runtimeCbSaved'), 'success')
  } catch (e) {
    showMessage(t('env.runtimeCbSaveFailed', { error: e instanceof Error ? e.message : String(e) }), 'error')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (status.value.running) {
    fetchConfig()
    fetchHistoricalImageLimit()
  }
})
</script>

<template>
  <Card>
    <CardHeader class="pb-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <CardTitle class="text-base flex items-center gap-2">
            <Zap class="w-4 h-4" />
            {{ t('env.runtimeCbTitle') }}
          </CardTitle>
          <p class="text-xs text-muted-foreground mt-1">{{ t('env.runtimeCbDesc') }}</p>
        </div>
        <div class="flex gap-2">
          <Button size="sm" variant="ghost" :disabled="loading || !status.running" @click="fetchConfig">
            <RefreshCw class="w-4 h-4 mr-1.5" :class="{ 'animate-spin': loading }" />
            {{ t('env.refresh') }}
          </Button>
          <Button size="sm" :disabled="saving || !status.running" @click="saveConfig">
            <Save class="w-4 h-4 mr-1.5" :class="{ 'animate-spin': saving }" />
            {{ saving ? t('env.saving') : t('env.save') }}
          </Button>
        </div>
      </div>

      <Alert v-if="!status.running" variant="default" class="mt-3">
        <p class="text-sm">{{ t('env.runtimeCbServiceStopped') }}</p>
      </Alert>
      <Alert v-if="error" variant="destructive" class="mt-3">
        <p class="text-sm">{{ error }}</p>
      </Alert>
      <Alert v-if="success" variant="default" class="mt-3">
        <p class="text-sm text-green-600">{{ success }}</p>
      </Alert>
    </CardHeader>

    <CardContent class="space-y-4">
      <!-- Sliders - 三列并排：基础参数 -->
      <div class="flex mb-4">
        <!-- 滑动窗口大小 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbWindowSize') }}</span>
            <span class="text-xs font-medium">{{ form.windowSize }}</span>
          </div>
          <input
            type="range"
            :value="form.windowSize"
            :min="3"
            :max="100"
            step="1"
            class="cb-slider w-full"
            :disabled="!status.running"
            @input="onSliderChange('windowSize', $event)"
          />
          <div class="flex justify-between text-xs text-muted-foreground"><span>3</span><span>100</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 失败率阈值 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbFailureThreshold') }}</span>
            <span class="text-xs font-medium">{{ form.failureThreshold.toFixed(2) }}</span>
          </div>
          <input
            type="range"
            :value="form.failureThreshold"
            :min="0.01"
            :max="1"
            step="0.01"
            class="cb-slider w-full"
            :disabled="!status.running"
            @input="onSliderChange('failureThreshold', $event)"
          />
          <div class="flex justify-between text-xs text-muted-foreground"><span>0.01</span><span>1.00</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 连续失败阈值 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbConsecutiveFailures') }}</span>
            <span class="text-xs font-medium">{{ form.consecutiveFailuresThreshold }}</span>
          </div>
          <input
            type="range"
            :value="form.consecutiveFailuresThreshold"
            :min="1"
            :max="100"
            step="1"
            class="cb-slider w-full"
            :disabled="!status.running"
            @input="onSliderChange('consecutiveFailuresThreshold', $event)"
          />
          <div class="flex justify-between text-xs text-muted-foreground"><span>1</span><span>100</span></div>
        </div>
      </div>

      <!-- Sliders - 流式健康检测超时 -->
      <div class="flex mb-4">
        <!-- 首字等待超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbStreamFirstContentTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.streamFirstContentTimeoutMs / 1000) + 's' }}</span>
          </div>
          <input
            type="range"
            :value="form.streamFirstContentTimeoutMs"
            :min="5000"
            :max="300000"
            step="1000"
            class="cb-slider w-full"
            :disabled="!status.running"
            @input="onSliderChange('streamFirstContentTimeoutMs', $event)"
          />
          <div class="flex justify-between text-xs text-muted-foreground"><span>5s</span><span>300s</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 首字后断流超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbStreamInactivityTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.streamInactivityTimeoutMs / 1000) + 's' }}</span>
          </div>
          <input
            type="range"
            :value="form.streamInactivityTimeoutMs"
            :min="1000"
            :max="180000"
            step="1000"
            class="cb-slider w-full"
            :disabled="!status.running"
            @input="onSliderChange('streamInactivityTimeoutMs', $event)"
          />
          <div class="flex justify-between text-xs text-muted-foreground"><span>1s</span><span>180s</span></div>
        </div>

        <div class="w-px bg-border mx-1 self-stretch" />

        <!-- 工具调用空闲超时 -->
        <div class="flex-1 px-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-muted-foreground">{{ t('env.runtimeCbStreamToolCallIdleTimeout') }}</span>
            <span class="text-xs font-medium">{{ (form.streamToolCallIdleTimeoutMs / 1000) + 's' }}</span>
          </div>
          <input
            type="range"
            :value="form.streamToolCallIdleTimeoutMs"
            :min="30000"
            :max="300000"
            step="1000"
            class="cb-slider w-full"
            :disabled="!status.running"
            @input="onSliderChange('streamToolCallIdleTimeoutMs', $event)"
          />
          <div class="flex justify-between text-xs text-muted-foreground"><span>30s</span><span>300s</span></div>
        </div>
      </div>

      <!-- Preset buttons -->
      <div class="flex gap-2">
        <Button
          v-for="p in presets"
          :key="p.key"
          size="sm"
          :variant="activePreset === p.key ? 'default' : 'outline'"
          :disabled="!status.running"
          @click="applyPreset(p)"
        >
          {{ t(p.labelKey) }}
        </Button>
      </div>

      <!-- 历史图片轮次限制 -->
      <div class="border-t border-border pt-4 mt-4">
        <div class="flex items-center justify-between mb-2">
          <div>
            <p class="text-sm font-medium">{{ t('env.historicalImageTurnLimitTitle') }}</p>
            <p class="text-xs text-muted-foreground mt-0.5">{{ t('env.historicalImageTurnLimitHint') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-muted-foreground">{{ t('env.historicalImageTurnLimitLabel') }}</span>
            <input
              v-model.number="historicalImageLimit"
              type="number"
              min="0"
              class="w-20 h-8 rounded border border-input bg-background px-2 text-sm text-center"
              :disabled="!status.running"
            />
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<style scoped>
.cb-slider {
  -webkit-appearance: none;
  appearance: none;
  height: 6px;
  border-radius: 3px;
  background: hsl(var(--muted));
  outline: none;
  cursor: pointer;
}
.cb-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: hsl(var(--primary));
  cursor: pointer;
  border: 2px solid hsl(var(--background));
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}
.cb-slider::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: hsl(var(--primary));
  cursor: pointer;
  border: 2px solid hsl(var(--background));
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}
.cb-slider:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
