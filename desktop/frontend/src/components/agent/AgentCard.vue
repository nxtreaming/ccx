<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { ExternalLink } from 'lucide-vue-next'
import { openProviderConsole, providerConsoleLinks } from '@/lib/external-link'
import { DetectEditors, OpenFileInEditor } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'
import ProviderForm from '@/components/agent/ProviderForm.vue'
import type { AgentPlatform, AgentConfigStatus, AgentProvider } from '@/types'
import { useLanguage } from '@/composables/useLanguage'

interface EditorInfo { id: string; name: string; path: string }

const props = defineProps<{
  platform: AgentPlatform
  agentStatus: AgentConfigStatus | null
  configLoading: boolean
  agentLabel: string
  agentStatusText: string
  agentStatusClass: string
  canApply: boolean
  selectedClaudeProvider?: AgentProvider
  claudeProviderKeys?: Record<AgentProvider, string>
  savedProviderKeys?: Record<string, string>
  claudeMimoBaseUrl?: string
  selectedMimoPlan?: string
  selectedDashScopePlan?: string
  claudeProviderLabel?: (value?: string) => string
  claudeTargetBaseUrl?: () => string
  selectedCodexProvider?: AgentProvider
  codexMode?: 'quick' | 'plugin'
  codexOpenAIKey?: string
  codexProviderLabels?: Record<string, string>
  codexProviderLabel?: (value?: string) => string
  codexTargetBaseUrl?: () => string
  selectedOpenCodeProvider?: AgentProvider
  openCodeOpenAIKey?: string
  openCodeProviderLabels?: Record<string, string>
  openCodeProviderLabel?: (value?: string) => string
  openCodeTargetBaseUrl?: () => string
}>()

const emit = defineEmits<{
  apply: []
  restore: []
  'update:selectedClaudeProvider': [value: AgentProvider]
  'update:claudeProviderKeys': [value: Record<AgentProvider, string>]
  'update:claudeMimoBaseUrl': [value: string]
  'update:selectedMimoPlan': [value: string]
  'update:selectedDashScopePlan': [value: string]
  'update:selectedCodexProvider': [value: AgentProvider]
  'update:codexMode': [value: 'quick' | 'plugin']
  'update:codexOpenAIKey': [value: string]
  'update:selectedOpenCodeProvider': [value: AgentProvider]
  'update:openCodeOpenAIKey': [value: string]
}>()

// OpenAI 直连 API Key 勾选框状态（默认不显示输入框）
const showCodexOwnKey = ref(false)

const codexKeyRequired = computed(() => {
  const p = props.selectedCodexProvider
  return p !== 'ccx' && p !== 'openai'
})

// 支持快捷模式/插件模式切换的 provider 列表
const codexHasMode = computed(() => {
  const p = props.selectedCodexProvider
  return p === 'ccx' || p === 'dashscope' || p === 'opencode-zen' || p === 'opencode-go'
})

const { t } = useLanguage()

const badgeClass = computed(() => {
  if (props.agentStatusClass === 'running') return 'bg-accent text-accent-foreground border-0'
  if (props.agentStatusClass === 'starting') return 'bg-warning text-warning-foreground border-0'
  return 'bg-destructive text-destructive-foreground border-0'
})

const applyLabel = computed(() => t('agent.applyConfig'))

// 编辑器检测与文件打开
const editors = ref<EditorInfo[]>([])
const openingFile = ref('')

onMounted(async () => {
  // 已保存 OpenAI key 时自动勾选"我有 API Key"
  if (props.selectedCodexProvider === 'openai') {
    if (props.savedProviderKeys?.['codex:openai']) {
      showCodexOwnKey.value = true
    }
  }
  try {
    editors.value = (await DetectEditors()) as EditorInfo[] ?? []
  } catch { editors.value = [] }
})

const openFileInEditor = async (editorPath: string, filePath: string) => {
  if (!editorPath || !filePath) return
  openingFile.value = filePath
  try {
    await OpenFileInEditor(editorPath, filePath)
  } catch { /* ignore */ }
  finally { openingFile.value = '' }
}
</script>

<template>
  <Card>
    <CardHeader class="pb-3">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-xs text-muted-foreground uppercase tracking-widest">{{ platform }}</p>
          <h3 class="text-base font-semibold mt-1">{{ agentLabel }}</h3>
        </div>
        <Badge :class="badgeClass">{{ agentStatusText }}</Badge>
      </div>
    </CardHeader>

    <CardContent class="space-y-4">
      <div class="space-y-2 text-sm">
        <div v-for="detail in [
          { label: t('agent.currentProvider'), value: (platform === 'codex' ? codexProviderLabel : platform === 'opencode' ? openCodeProviderLabel : claudeProviderLabel)?.(agentStatus?.provider) || agentStatus?.provider || t('agent.notSet') },
          { label: t('agent.currentUrl'), value: agentStatus?.currentBaseUrl || t('agent.notSet') },
          { label: t('agent.targetUrl'), value: platform === 'claude' ? claudeTargetBaseUrl?.() : platform === 'opencode' ? openCodeTargetBaseUrl?.() : codexTargetBaseUrl?.() },
        ]" :key="detail.label" class="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3">
          <span class="text-muted-foreground">{{ detail.label }}</span>
          <div class="min-w-0 text-right">
            <code class="inline-block max-w-full rounded bg-secondary px-2 py-0.5 text-right text-xs break-all">{{ detail.value }}</code>
          </div>
        </div>
        <!-- 配置文件 — 带编辑器打开按钮 -->
        <div class="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3">
          <span class="text-muted-foreground">{{ t('agent.configPath') }}</span>
          <div class="flex min-w-0 items-center justify-end gap-2">
            <code class="inline-block min-w-0 max-w-full rounded bg-secondary px-2 py-0.5 text-right text-xs break-all">{{ agentStatus?.configPath || '--' }}</code>
            <div v-if="agentStatus?.configPath && editors.length > 0" class="relative shrink-0">
              <Button
                variant="ghost"
                size="icon-sm"
                :title="t('agent.openFileInEditor')"
                :disabled="openingFile === agentStatus.configPath"
                @click="editors.length === 1 && openFileInEditor(editors[0].path, agentStatus.configPath)"
              >
                <ExternalLink class="w-3.5 h-3.5" />
              </Button>
              <select
                v-if="editors.length > 1"
                class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                :disabled="openingFile === agentStatus.configPath"
                @change="($event.target as HTMLSelectElement).value && openFileInEditor(($event.target as HTMLSelectElement).value, agentStatus!.configPath!); ($event.target as HTMLSelectElement).selectedIndex = 0"
              >
                <option value="" disabled selected></option>
                <option v-for="ed in editors" :key="ed.id" :value="ed.path">{{ ed.name }}</option>
              </select>
            </div>
          </div>
        </div>
        <!-- 认证文件 — 带编辑器打开按钮 -->
        <div v-if="agentStatus?.authPath" class="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3">
          <span class="text-muted-foreground">{{ t('agent.authPath') }}</span>
          <div class="flex min-w-0 items-center justify-end gap-2">
            <code class="inline-block min-w-0 max-w-full rounded bg-secondary px-2 py-0.5 text-right text-xs break-all">{{ agentStatus.authPath }}</code>
            <div v-if="editors.length > 0" class="relative shrink-0">
              <Button
                variant="ghost"
                size="icon-sm"
                :title="t('agent.openFileInEditor')"
                :disabled="openingFile === agentStatus.authPath"
                @click="editors.length === 1 && openFileInEditor(editors[0].path, agentStatus.authPath!)"
              >
                <ExternalLink class="w-3.5 h-3.5" />
              </Button>
              <select
                v-if="editors.length > 1"
                class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                :disabled="openingFile === agentStatus.authPath"
                @change="($event.target as HTMLSelectElement).value && openFileInEditor(($event.target as HTMLSelectElement).value, agentStatus!.authPath!); ($event.target as HTMLSelectElement).selectedIndex = 0"
              >
                <option value="" disabled selected></option>
                <option v-for="ed in editors" :key="ed.id" :value="ed.path">{{ ed.name }}</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      <ProviderForm
        v-if="platform === 'claude'"
        :selected-provider="selectedClaudeProvider!"
        :provider-keys="claudeProviderKeys!"
        :saved-provider-keys="savedProviderKeys || {}"
        :mimo-base-url="claudeMimoBaseUrl!"
        :selected-mimo-plan="selectedMimoPlan || 'https://api.xiaomimimo.com/anthropic'"
        :selected-dash-scope-plan="selectedDashScopePlan || 'https://dashscope.aliyuncs.com/apps/anthropic'"
        @update:selected-provider="emit('update:selectedClaudeProvider', $event)"
        @update:provider-keys="emit('update:claudeProviderKeys', $event)"
        @update:mimo-base-url="emit('update:claudeMimoBaseUrl', $event)"
        @update:selected-mimo-plan="emit('update:selectedMimoPlan', $event)"
        @update:selected-dash-scope-plan="emit('update:selectedDashScopePlan', $event)"
      />
      <div v-else-if="platform === 'codex'" class="space-y-3">
        <div class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">Provider</Label>
          <select
            :value="selectedCodexProvider"
            class="w-full h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            @change="showCodexOwnKey = false; emit('update:selectedCodexProvider', ($event.target as HTMLSelectElement).value as AgentProvider)"
          >
            <option value="ccx">{{ t('agent.provider.localGateway') }}</option>
            <option value="openai">{{ t('agent.provider.openaiDirect') }}</option>
            <option value="dashscope">{{ t('agent.provider.dashscopeDirect') }}</option>
            <option value="opencode-zen">{{ t('agent.provider.opencodeZenDirect') }}</option>
            <option value="opencode-go">{{ t('agent.provider.opencodeGoDirect') }}</option>
          </select>
        </div>
        <div v-if="codexHasMode" class="space-y-2">
          <Label class="text-xs text-muted-foreground">{{ t('agent.codexMode') }}</Label>
          <div class="grid grid-cols-2 gap-2 rounded-lg bg-secondary/40 p-1">
            <button
              type="button"
              class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
              :class="codexMode === 'quick' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
              @click="emit('update:codexMode', 'quick')"
            >
              {{ t('agent.codexQuickMode') }}
            </button>
            <button
              type="button"
              class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
              :class="codexMode === 'plugin' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
              @click="emit('update:codexMode', 'plugin')"
            >
              {{ t('agent.codexPluginMode') }}
            </button>
          </div>
          <p class="text-xs text-muted-foreground leading-relaxed">
            {{ codexMode === 'plugin' ? t('agent.codexPluginModeHint') : t('agent.codexQuickModeHint') }}
          </p>
        </div>
        <div
          v-if="selectedCodexProvider && selectedCodexProvider !== 'ccx' && providerConsoleLinks[selectedCodexProvider]"
          class="inline-flex items-center gap-2"
        >
          <button
            type="button"
            class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground"
            @click="openProviderConsole(selectedCodexProvider)"
          >
            {{ t('agent.openConsole') }}
            <ExternalLink class="h-3 w-3" />
          </button>
          <span class="text-muted-foreground">|</span>
          <label
            v-if="selectedCodexProvider === 'openai'"
            class="inline-flex items-center gap-1 text-xs text-muted-foreground select-none cursor-pointer"
          >
            <input
              type="checkbox"
              class="h-3 w-3 rounded border-input accent-primary cursor-pointer"
              :checked="showCodexOwnKey"
              @change="showCodexOwnKey = ($event.target as HTMLInputElement).checked"
            >
            {{ t('agent.hasOwnApiKey') }}
          </label>
        </div>
        <div v-if="selectedCodexProvider !== 'ccx' && (codexKeyRequired || showCodexOwnKey)" class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">API Key <span v-if="codexKeyRequired" class="text-destructive">*</span></Label>
          <Input
            type="password"
            autocomplete="off"
            :placeholder="savedProviderKeys?.[`codex:${selectedCodexProvider}`] ? t('agent.codexPlaceholderSaved') : codexKeyRequired ? t('agent.codexPlaceholderRequired') : t('agent.codexPlaceholderWriteOnly')"
            :model-value="codexOpenAIKey || ''"
            @update:model-value="emit('update:codexOpenAIKey', String($event))"
          />
        </div>
      </div>

      <div v-else-if="platform === 'opencode'" class="space-y-3">
        <div class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">Provider</Label>
          <select
            :value="selectedOpenCodeProvider"
            class="w-full h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            @change="emit('update:selectedOpenCodeProvider', ($event.target as HTMLSelectElement).value as AgentProvider)"
          >
            <option value="ccx">{{ t('agent.provider.localGateway') }}</option>
            <option value="deepseek">{{ t('agent.provider.deepseekDirect') }}</option>
            <option value="kimi">{{ t('agent.provider.kimiDirect') }}</option>
            <option value="glm">{{ t('agent.provider.glmDirect') }}</option>
            <option value="minimax">{{ t('agent.provider.minimaxDirect') }}</option>
            <option value="opencode-zen">{{ t('agent.provider.opencodeZenDirect') }}</option>
            <option value="opencode-go">{{ t('agent.provider.opencodeGoDirect') }}</option>
          </select>
        </div>
        <button
          v-if="selectedOpenCodeProvider && selectedOpenCodeProvider !== 'ccx' && providerConsoleLinks[selectedOpenCodeProvider]"
          type="button"
          class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground"
          @click="openProviderConsole(selectedOpenCodeProvider)"
        >
          {{ t('agent.openConsole') }}
          <ExternalLink class="h-3 w-3" />
        </button>
        <div v-if="selectedOpenCodeProvider !== 'ccx'" class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">API Key</Label>
          <Input
            type="password"
            autocomplete="off"
            :placeholder="savedProviderKeys?.[`codex:${selectedOpenCodeProvider}`] ? t('agent.codexPlaceholderSaved') : t('agent.codexPlaceholderRequired')"
            :model-value="openCodeOpenAIKey || ''"
            @update:model-value="emit('update:openCodeOpenAIKey', String($event))"
          />
        </div>
      </div>

      <p v-if="agentStatus?.lastError" class="text-sm text-destructive-foreground">{{ agentStatus.lastError }}</p>

      <div class="flex flex-wrap gap-2">
        <Button size="sm" :disabled="!canApply" @click="emit('apply')">
          {{ applyLabel }}
        </Button>
        <Button size="sm" variant="secondary" :disabled="configLoading || !agentStatus?.hasState" @click="emit('restore')">
          {{ t('agent.restoreConfig') }}
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
