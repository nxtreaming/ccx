<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { ExternalLink } from 'lucide-vue-next'
import { DetectEditors, OpenFileInEditor } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'
import ProviderForm from '@/components/agent/ProviderForm.vue'
import type { AgentPlatform, AgentConfigStatus, AgentProvider } from '@/types'

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
  claudeProviderLabel?: (value?: string) => string
  claudeTargetBaseUrl?: () => string
  selectedCodexProvider?: AgentProvider
  codexOpenAIKey?: string
  codexProviderLabels?: Record<string, string>
  codexProviderLabel?: (value?: string) => string
  codexTargetBaseUrl?: () => string
}>()

const emit = defineEmits<{
  apply: []
  restore: []
  'update:selectedClaudeProvider': [value: AgentProvider]
  'update:claudeProviderKeys': [value: Record<AgentProvider, string>]
  'update:claudeMimoBaseUrl': [value: string]
  'update:selectedMimoPlan': [value: string]
  'update:selectedCodexProvider': [value: AgentProvider]
  'update:codexOpenAIKey': [value: string]
}>()

const codexKeyRequired = computed(() => {
  const p = props.selectedCodexProvider
  return p !== 'ccx' && p !== 'openai'
})

const badgeClass = computed(() => {
  if (props.agentStatusClass === 'running') return 'bg-accent text-accent-foreground border-0'
  if (props.agentStatusClass === 'starting') return 'bg-warning text-warning-foreground border-0'
  return 'bg-destructive text-destructive-foreground border-0'
})

const applyLabel = computed(() => '应用配置')

// 编辑器检测与文件打开
const editors = ref<EditorInfo[]>([])
const openingFile = ref('')

onMounted(async () => {
  try {
    editors.value = (await DetectEditors()) as EditorInfo[] ?? []
  } catch { editors.value = [] }
})

const openFileInEditor = async (filePath: string) => {
  if (editors.value.length === 0) return
  openingFile.value = filePath
  try {
    const editorPath = editors.value.length === 1 ? editors.value[0].path : ''
    if (editorPath) {
      await OpenFileInEditor(editorPath, filePath)
    }
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
          { label: '当前 Provider', value: (platform === 'codex' ? codexProviderLabel : claudeProviderLabel)?.(agentStatus?.provider) || agentStatus?.provider || '未设置' },
          { label: '当前 URL', value: agentStatus?.currentBaseUrl || '未设置' },
          { label: '目标 URL', value: platform === 'claude' ? claudeTargetBaseUrl?.() : codexTargetBaseUrl?.() },
        ]" :key="detail.label" class="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3">
          <span class="text-muted-foreground">{{ detail.label }}</span>
          <div class="min-w-0 text-right">
            <code class="inline-block max-w-full rounded bg-secondary px-2 py-0.5 text-right text-xs break-all">{{ detail.value }}</code>
          </div>
        </div>
        <!-- 配置文件 — 带编辑器打开按钮 -->
        <div class="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3">
          <span class="text-muted-foreground">配置文件</span>
          <div class="flex min-w-0 items-center justify-end gap-2">
            <code class="inline-block min-w-0 max-w-full rounded bg-secondary px-2 py-0.5 text-right text-xs break-all">{{ agentStatus?.configPath || '--' }}</code>
            <Button
              v-if="agentStatus?.configPath && editors.length > 0"
              variant="ghost"
              size="icon-sm"
              title="用编辑器打开"
              class="shrink-0"
              :disabled="openingFile === agentStatus.configPath"
              @click="openFileInEditor(agentStatus.configPath)"
            >
              <ExternalLink class="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
        <!-- 认证文件 — 带编辑器打开按钮 -->
        <div v-if="agentStatus?.authPath" class="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3">
          <span class="text-muted-foreground">认证文件</span>
          <div class="flex min-w-0 items-center justify-end gap-2">
            <code class="inline-block min-w-0 max-w-full rounded bg-secondary px-2 py-0.5 text-right text-xs break-all">{{ agentStatus.authPath }}</code>
            <Button
              v-if="editors.length > 0"
              variant="ghost"
              size="icon-sm"
              title="用编辑器打开"
              class="shrink-0"
              :disabled="openingFile === agentStatus.authPath"
              @click="openFileInEditor(agentStatus.authPath!)"
            >
              <ExternalLink class="w-3.5 h-3.5" />
            </Button>
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
        @update:selected-provider="emit('update:selectedClaudeProvider', $event)"
        @update:provider-keys="emit('update:claudeProviderKeys', $event)"
        @update:mimo-base-url="emit('update:claudeMimoBaseUrl', $event)"
        @update:selected-mimo-plan="emit('update:selectedMimoPlan', $event)"
      />
      <div v-else-if="platform === 'codex'" class="space-y-3">
        <div class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">Provider</Label>
          <select
            :value="selectedCodexProvider"
            class="w-full h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            @change="emit('update:selectedCodexProvider', ($event.target as HTMLSelectElement).value as AgentProvider)"
          >
            <option value="ccx">CCX 本地网关</option>
            <option value="openai">OpenAI 官方</option>
            <option value="dashscope">DashScope 直连</option>
            <option value="opencode-zen">OpenCode Zen 直连</option>
            <option value="opencode-go">OpenCode Go 直连</option>
          </select>
        </div>
        <div v-if="selectedCodexProvider !== 'ccx'" class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">API Key <span v-if="codexKeyRequired" class="text-destructive">*</span></Label>
          <Input
            type="password"
            autocomplete="off"
            :placeholder="savedProviderKeys?.[`codex:${selectedCodexProvider}`] ? '已保存，留空则使用已保存的 key' : codexKeyRequired ? '必填：输入 API Key' : '仅写入 Codex 配置'"
            :model-value="codexOpenAIKey || ''"
            @update:model-value="emit('update:codexOpenAIKey', String($event))"
          />
        </div>
      </div>

      <p v-if="agentStatus?.lastError" class="text-sm text-destructive-foreground">{{ agentStatus.lastError }}</p>

      <div class="flex flex-wrap gap-2">
        <Button size="sm" :disabled="!canApply" @click="emit('apply')">
          {{ applyLabel }}
        </Button>
        <Button size="sm" variant="secondary" :disabled="configLoading || !agentStatus?.hasState" @click="emit('restore')">
          恢复原始配置
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
