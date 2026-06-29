<script setup lang="ts">
import { reactive, ref, watch, computed, onMounted, onBeforeUnmount } from 'vue'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Alert } from '@/components/ui/alert'
import { Loader2, Zap } from 'lucide-vue-next'
import { useStatus } from '@/composables/useStatus'
import { useLanguage } from '@/composables/useLanguage'
import { streamTimeoutPresets } from '@/utils/stream-timeout-presets'
import { GetAdminAccessKey } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

interface Props {
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ (e: 'close'): void }>()

const { status } = useStatus()
const { t } = useLanguage()

const isMac = computed(() => typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform))
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')

const activePreset = ref('balanced')
const form = reactive({
  windowSize: 10,
  failureThreshold: 0.5,
  consecutiveFailuresThreshold: 3,
  requestTimeoutMs: 120000,
  responseHeaderTimeoutMs: 60000,
  streamFirstContentTimeoutMs: streamTimeoutPresets.balanced.firstContentMs,
  streamInactivityTimeoutMs: streamTimeoutPresets.balanced.inactivityMs,
  streamToolCallIdleTimeoutMs: streamTimeoutPresets.balanced.toolCallIdleMs,
})

const sliderStyle = (value: number, min: number, max: number) => {
  const percent = ((value - min) / (max - min)) * 100
  return { '--cb-slider-progress': `${Math.min(100, Math.max(0, percent))}%` }
}

// 工具调用 idle 预设按低速 5 TPS 粗估：60/120/300s 分别预留约 300/600/1500 token 的参数生成窗口。
const presets = [
  { key: 'gentle', labelKey: 'env.runtimeCbPresetGentle' as const, windowSize: 20, failureThreshold: 0.70, consecutiveFailuresThreshold: 5, requestTimeoutMs: 300000, responseHeaderTimeoutMs: 120000, streamFirstContentTimeoutMs: streamTimeoutPresets.gentle.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.gentle.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.gentle.toolCallIdleMs },
  { key: 'balanced', labelKey: 'env.runtimeCbPresetBalanced' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, requestTimeoutMs: 120000, responseHeaderTimeoutMs: 60000, streamFirstContentTimeoutMs: streamTimeoutPresets.balanced.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.balanced.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.balanced.toolCallIdleMs },
  { key: 'aggressive', labelKey: 'env.runtimeCbPresetAggressive' as const, windowSize: 5, failureThreshold: 0.30, consecutiveFailuresThreshold: 2, requestTimeoutMs: 60000, responseHeaderTimeoutMs: 30000, streamFirstContentTimeoutMs: streamTimeoutPresets.aggressive.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.aggressive.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.aggressive.toolCallIdleMs },
  { key: 'custom', labelKey: 'env.runtimeCbPresetCustom' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, requestTimeoutMs: 120000, responseHeaderTimeoutMs: 60000, streamFirstContentTimeoutMs: streamTimeoutPresets.balanced.firstContentMs, streamInactivityTimeoutMs: streamTimeoutPresets.balanced.inactivityMs, streamToolCallIdleTimeoutMs: streamTimeoutPresets.balanced.toolCallIdleMs },
]

const matchPreset = () => {
  for (const p of presets) {
    if (p.key === 'custom') continue
    if (form.windowSize === p.windowSize && form.failureThreshold === p.failureThreshold && form.consecutiveFailuresThreshold === p.consecutiveFailuresThreshold && form.requestTimeoutMs === p.requestTimeoutMs && form.responseHeaderTimeoutMs === p.responseHeaderTimeoutMs && form.streamFirstContentTimeoutMs === p.streamFirstContentTimeoutMs && form.streamInactivityTimeoutMs === p.streamInactivityTimeoutMs && form.streamToolCallIdleTimeoutMs === p.streamToolCallIdleTimeoutMs) {
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
  form.requestTimeoutMs = preset.requestTimeoutMs
  form.responseHeaderTimeoutMs = preset.responseHeaderTimeoutMs
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
  } else if (field === 'requestTimeoutMs') {
    form.requestTimeoutMs = val
  } else if (field === 'responseHeaderTimeoutMs') {
    form.responseHeaderTimeoutMs = val
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
    const resp = await fetch(url, { headers: { 'x-api-key': adminKey } })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = await resp.json()
    form.windowSize = data.windowSize ?? 10
    form.failureThreshold = data.failureThreshold ?? 0.5
    form.consecutiveFailuresThreshold = data.consecutiveFailuresThreshold ?? 3
    form.requestTimeoutMs = data.requestTimeoutMs && data.requestTimeoutMs >= 1000 ? data.requestTimeoutMs : 120000
    form.responseHeaderTimeoutMs = data.responseHeaderTimeoutMs && data.responseHeaderTimeoutMs >= 1000 ? data.responseHeaderTimeoutMs : 60000
    form.streamFirstContentTimeoutMs = data.streamFirstContentTimeoutMs && data.streamFirstContentTimeoutMs >= 5000 ? data.streamFirstContentTimeoutMs : 60000
    form.streamInactivityTimeoutMs = data.streamInactivityTimeoutMs && data.streamInactivityTimeoutMs >= 1000 ? data.streamInactivityTimeoutMs : 60000
    form.streamToolCallIdleTimeoutMs = data.streamToolCallIdleTimeoutMs && data.streamToolCallIdleTimeoutMs >= 30000 ? data.streamToolCallIdleTimeoutMs : 180000
    matchPreset()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  const cbUrl = await buildApiUrl('/api/settings/circuit-breaker')
  if (!cbUrl) {
    error.value = t('env.runtimeCbNoBackend')
    return
  }

  saving.value = true
  clearMessages()
  try {
    const adminKey = await GetAdminAccessKey()
    const resp = await fetch(cbUrl, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'x-api-key': adminKey },
      body: JSON.stringify({
        windowSize: form.windowSize,
        failureThreshold: form.failureThreshold,
        consecutiveFailuresThreshold: form.consecutiveFailuresThreshold,
        requestTimeoutMs: form.requestTimeoutMs,
        responseHeaderTimeoutMs: form.responseHeaderTimeoutMs,
        streamFirstContentTimeoutMs: form.streamFirstContentTimeoutMs,
        streamInactivityTimeoutMs: form.streamInactivityTimeoutMs,
        streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutMs,
      }),
    })
    if (!resp.ok) {
      const body = await resp.json().catch(() => ({}))
      throw new Error(body.error || `HTTP ${resp.status}`)
    }
    success.value = t('env.runtimeCbSaved')
    setTimeout(() => { success.value = '' }, 3000)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (!props.open) return

  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
    return
  }

  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey) && !e.shiftKey && !saving.value) {
    e.preventDefault()
    void saveConfig()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDown)
})

// 弹窗打开时加载配置
const loadOnOpen = async () => {
  if (props.open && status.value.running) {
    await fetchConfig()
  }
}

// 监听 open 变化
watch(() => props.open, (isOpen) => {
  if (isOpen) loadOnOpen()
}, { immediate: true })
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('close')" />

        <div class="cb-dialog-shell relative z-10 flex max-h-[calc(100vh-48px)] w-[560px] max-w-[90vw] flex-col overflow-hidden rounded-2xl border border-border/80 bg-gradient-to-br from-card/95 to-card/85 shadow-2xl backdrop-blur-md">
          <!-- Header -->
          <div class="shrink-0 border-b border-border/60 bg-background shadow-[0_6px_18px_rgba(15,23,42,0.08)]">
            <div class="flex items-center justify-between px-5 py-3">
              <div class="flex items-center gap-2.5">
                <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 ring-1 ring-primary/20">
                  <Zap class="h-4 w-4 fill-primary/20 text-primary" />
                </div>
                <h3 class="text-base font-bold tracking-tight">{{ t('env.runtimeCbTitle') }}</h3>
              </div>
            </div>
          </div>

          <!-- Body -->
          <ScrollArea type="auto" class="min-h-0 w-full flex-1">
            <div class="space-y-4 px-4 py-4">
                <Alert v-if="error" variant="destructive" class="shadow-sm">
                  <p class="text-xs">{{ error }}</p>
                </Alert>
                <Alert v-if="success" variant="default" class="shadow-sm">
                  <p class="text-xs text-green-600">{{ success }}</p>
                </Alert>

                <!-- 基础参数：三列 -->
                <div class="rounded-xl border border-border/60 bg-gradient-to-br from-background/60 to-background/40 p-3 shadow-sm backdrop-blur-sm">
                  <div class="mb-2 flex items-center gap-1.5 border-b border-border/40 pb-2">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
                    <span class="text-[10px] font-bold uppercase tracking-wider text-primary">Circuit Breaker</span>
                  </div>
                  <div class="flex gap-2">
                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbWindowSize') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ form.windowSize }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.windowSize, 3, 100)">
                      <input
                        type="range"
                        :value="form.windowSize"
                        :min="3"
                        :max="100"
                        step="1"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbWindowSize')"
                        @input="onSliderChange('windowSize', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>3</span><span>100</span></div>
                  </div>

                  <div class="w-px bg-border self-stretch" />

                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbFailureThreshold') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ form.failureThreshold.toFixed(2) }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.failureThreshold, 0.01, 1)">
                      <input
                        type="range"
                        :value="form.failureThreshold"
                        :min="0.01"
                        :max="1"
                        step="0.01"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbFailureThreshold')"
                        @input="onSliderChange('failureThreshold', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>0.01</span><span>1.00</span></div>
                  </div>
                  <div class="w-px bg-border self-stretch" />

                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbConsecutiveFailures') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ form.consecutiveFailuresThreshold }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.consecutiveFailuresThreshold, 1, 100)">
                      <input
                        type="range"
                        :value="form.consecutiveFailuresThreshold"
                        :min="1"
                        :max="100"
                        step="1"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbConsecutiveFailures')"
                        @input="onSliderChange('consecutiveFailuresThreshold', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>1</span><span>100</span></div>
                  </div>
                </div>
                </div>

                <!-- 请求生命周期超时：两列 -->
                <div class="rounded-xl border border-border/60 bg-gradient-to-br from-background/60 to-background/40 p-3 shadow-sm backdrop-blur-sm">
                  <div class="mb-2 flex items-center gap-1.5 border-b border-border/40 pb-2">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2v2"/><path d="M14 2v2"/><path d="M16 8a6 6 0 1 1-8 0"/><path d="M12 14v-4"/></svg>
                    <span class="text-[10px] font-bold uppercase tracking-wider text-primary">Request Timeout</span>
                  </div>
                  <div class="flex gap-2">
                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbRequestTimeout') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ (form.requestTimeoutMs / 1000) + 's' }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.requestTimeoutMs, 1000, 300000)">
                      <input
                        type="range"
                        :value="form.requestTimeoutMs"
                        :min="1000"
                        :max="300000"
                        step="1000"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbRequestTimeout')"
                        @input="onSliderChange('requestTimeoutMs', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>1s</span><span>300s</span></div>
                  </div>

                  <div class="w-px bg-border self-stretch" />

                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbResponseHeaderTimeout') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ (form.responseHeaderTimeoutMs / 1000) + 's' }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.responseHeaderTimeoutMs, 1000, 300000)">
                      <input
                        type="range"
                        :value="form.responseHeaderTimeoutMs"
                        :min="1000"
                        :max="300000"
                        step="1000"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbResponseHeaderTimeout')"
                        @input="onSliderChange('responseHeaderTimeoutMs', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>1s</span><span>300s</span></div>
                  </div>
                </div>
                </div>

                <!-- 流式超时：三列 -->
                <div class="rounded-xl border border-border/60 bg-gradient-to-br from-background/60 to-background/40 p-3 shadow-sm backdrop-blur-sm">
                  <div class="mb-2 flex items-center gap-1.5 border-b border-border/40 pb-2">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v20M2 12h20"/></svg>
                    <span class="text-[10px] font-bold uppercase tracking-wider text-primary">Stream Timeout</span>
                  </div>
                  <div class="flex gap-2">
                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbStreamFirstContentTimeout') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ (form.streamFirstContentTimeoutMs / 1000) + 's' }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.streamFirstContentTimeoutMs, 5000, 300000)">
                      <input
                        type="range"
                        :value="form.streamFirstContentTimeoutMs"
                        :min="5000"
                        :max="300000"
                        step="1000"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbStreamFirstContentTimeout')"
                        @input="onSliderChange('streamFirstContentTimeoutMs', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>5s</span><span>300s</span></div>
                  </div>

                  <div class="w-px bg-border self-stretch" />

                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbStreamInactivityTimeout') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ (form.streamInactivityTimeoutMs / 1000) + 's' }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.streamInactivityTimeoutMs, 1000, 180000)">
                      <input
                        type="range"
                        :value="form.streamInactivityTimeoutMs"
                        :min="1000"
                        :max="180000"
                        step="1000"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbStreamInactivityTimeout')"
                        @input="onSliderChange('streamInactivityTimeoutMs', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>1s</span><span>180s</span></div>
                  </div>

                  <div class="w-px bg-border self-stretch" />

                  <div class="flex-1 px-1">
                    <div class="flex items-center justify-between mb-1">
                      <span class="text-[11px] text-muted-foreground">{{ t('env.runtimeCbStreamToolCallIdleTimeout') }}</span>
                      <span class="text-[11px] font-mono font-medium">{{ (form.streamToolCallIdleTimeoutMs / 1000) + 's' }}</span>
                    </div>
                    <div class="cb-slider-shell" :style="sliderStyle(form.streamToolCallIdleTimeoutMs, 30000, 300000)">
                      <input
                        type="range"
                        :value="form.streamToolCallIdleTimeoutMs"
                        :min="30000"
                        :max="300000"
                        step="1000"
                        class="cb-slider-input"
                        :aria-label="t('env.runtimeCbStreamToolCallIdleTimeout')"
                        @input="onSliderChange('streamToolCallIdleTimeoutMs', $event)"
                      />
                      <div class="cb-slider-visual" aria-hidden="true">
                        <div class="cb-slider-track">
                          <div class="cb-slider-fill" />
                        </div>
                        <div class="cb-slider-thumb" />
                      </div>
                    </div>
                    <div class="flex justify-between text-[10px] text-muted-foreground"><span>30s</span><span>300s</span></div>
                  </div>
                </div>
                </div>

                <!-- 预设按钮 -->
                <div class="flex gap-2 px-1">
                  <Button
                    v-for="p in presets"
                    :key="p.key"
                    size="sm"
                    :variant="activePreset === p.key ? 'default' : 'outline'"
                    class="text-xs"
                    @click="applyPreset(p)"
                  >
                    {{ t(p.labelKey) }}
                  </Button>
                </div>

            </div>
          </ScrollArea>

          <!-- Footer -->
          <div class="cb-dialog-glass cb-dialog-glass-bottom shrink-0 border-t border-border/60">
            <div class="flex items-center justify-end gap-2.5 p-3">
              <Button variant="outline" size="sm" class="text-xs shadow-sm hover:shadow-md transition-all" @click="emit('close')">
                {{ t('common.cancel') }}
                <span class="ml-1.5 text-xs opacity-60">Esc</span>
              </Button>
              <Button size="sm" :disabled="saving" class="text-xs shadow-sm hover:shadow-lg transition-all" @click="saveConfig">
                <Loader2 v-if="saving" class="h-3 w-3 mr-1.5 animate-spin" />
                {{ t('env.save') }}
                <span class="ml-1.5 text-xs opacity-60">{{ isMac ? '⌘ Enter' : 'Ctrl+Enter' }}</span>
              </Button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.cb-dialog-shell {
  --cb-dialog-glass-bg: rgb(255 255 255 / 0.08);
  --cb-dialog-glass-shadow: rgb(15 23 42 / 0.06);
  --cb-dialog-glass-highlight: rgb(255 255 255 / 0.14);
}

:global(.dark) .cb-dialog-shell {
  --cb-dialog-glass-bg: rgb(255 255 255 / 0.04);
  --cb-dialog-glass-shadow: rgb(0 0 0 / 0.18);
  --cb-dialog-glass-highlight: rgb(255 255 255 / 0.06);
}

.cb-dialog-glass {
  background: var(--cb-dialog-glass-bg);
  backdrop-filter: blur(24px) saturate(1.45) brightness(1.03);
  -webkit-backdrop-filter: blur(24px) saturate(1.45) brightness(1.03);
  box-shadow:
    0 0 0 1px var(--cb-dialog-glass-highlight) inset,
    0 14px 30px var(--cb-dialog-glass-shadow);
}

.cb-dialog-glass-bottom {
  box-shadow:
    0 0 0 1px var(--cb-dialog-glass-highlight) inset,
    0 -14px 30px var(--cb-dialog-glass-shadow);
}

.cb-slider-shell {
  --cb-slider-progress: 0%;
  --cb-slider-track-bg: color-mix(in srgb, var(--color-input) 76%, var(--color-muted) 24%);
  --cb-slider-fill-bg: color-mix(in srgb, var(--color-primary) 72%, transparent);
  --cb-slider-thumb-ring: color-mix(in srgb, var(--color-primary) 18%, var(--color-border));
  --cb-slider-thumb-hover-ring: color-mix(in srgb, var(--color-primary) 42%, var(--color-border));
  --cb-slider-thumb-hover-glow: color-mix(in srgb, var(--color-primary) 14%, transparent);
  --cb-slider-thumb-active-ring: color-mix(in srgb, var(--color-primary) 52%, var(--color-border));
  --cb-slider-thumb-active-glow: color-mix(in srgb, var(--color-primary) 16%, transparent);
  position: relative;
  width: 100%;
  height: 28px;
}
.cb-slider-input {
  -webkit-appearance: none;
  appearance: none;
  position: absolute;
  inset: 0;
  z-index: 2;
  width: 100%;
  height: 28px;
  margin: 0;
  background: transparent;
  opacity: 0;
  outline: none;
  cursor: pointer;
}
.cb-slider-input::-webkit-slider-runnable-track {
  height: 28px;
  background: transparent;
}
.cb-slider-input::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 26px;
  height: 28px;
  background: transparent;
  border: 0;
}
.cb-slider-input::-moz-range-track {
  height: 28px;
  background: transparent;
  border: 0;
}
.cb-slider-input::-moz-range-thumb {
  width: 26px;
  height: 28px;
  background: transparent;
  border: 0;
}
.cb-slider-input:disabled {
  cursor: not-allowed;
}
.cb-slider-visual {
  position: absolute;
  top: 0;
  right: 13px;
  bottom: 0;
  left: 13px;
  pointer-events: none;
}
.cb-slider-track {
  position: absolute;
  top: 50%;
  right: 0;
  left: 0;
  height: 4px;
  transform: translateY(-50%);
  border-radius: 999px;
  background: var(--cb-slider-track-bg);
  box-shadow: inset 0 1px 1px rgb(15 23 42 / 0.08);
}
.cb-slider-fill {
  width: var(--cb-slider-progress);
  height: 100%;
  border-radius: inherit;
  background: var(--cb-slider-fill-bg);
}
.cb-slider-thumb {
  position: absolute;
  top: 50%;
  left: var(--cb-slider-progress);
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--color-primary);
  cursor: pointer;
  border: 3px solid var(--color-background);
  box-shadow:
    0 0 0 1px var(--cb-slider-thumb-ring),
    0 2px 5px rgb(15 23 42 / 0.16);
  transform: translate(-50%, -50%);
  transition: all 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}
.cb-slider-input:hover + .cb-slider-visual .cb-slider-thumb,
.cb-slider-input:focus-visible + .cb-slider-visual .cb-slider-thumb {
  box-shadow:
    0 0 0 1px var(--cb-slider-thumb-hover-ring),
    0 0 0 4px var(--cb-slider-thumb-hover-glow),
    0 3px 7px rgb(15 23 42 / 0.18);
  transform: translate(-50%, -50%) scale(1.1);
}
.cb-slider-input:active + .cb-slider-visual .cb-slider-thumb {
  box-shadow:
    0 0 0 1px var(--cb-slider-thumb-active-ring),
    0 0 0 5px var(--cb-slider-thumb-active-glow),
    0 2px 5px rgb(15 23 42 / 0.16);
  transform: translate(-50%, -50%) scale(1.04);
}
.cb-slider-input:disabled + .cb-slider-visual {
  opacity: 0.5;
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
