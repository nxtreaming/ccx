import { ref, computed } from 'vue'
import type { AgentPlatform, AgentProvider, AgentConfigStatus, ApplyAgentConfigRequest, ConfigDiffResult } from '@/types'
import { useLanguage } from '@/composables/useLanguage'
import {
  GetAgentConfigStatus,
  ApplyAgentConfig,
  RestoreAgentConfig,
  GetSavedProviderKeys,
  PreviewAgentConfigDiff,
  PreviewRestoreConfigDiff,
} from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

const { t } = useLanguage()

const agentLabels: Record<AgentPlatform, string> = {
  claude: 'Claude Code',
  codex: 'Codex',
  opencode: 'OpenCode',
}

const claudeProviderLabels: Record<AgentProvider | 'custom', string> = {
  ccx: 'CCX',
  deepseek: 'DeepSeek',
  mimo: 'MiMo',
  compshare: 'Compshare',
  kimi: 'Kimi',
  glm: 'GLM',
  minimax: 'MiniMax',
  dashscope: 'DashScope',
  'opencode-zen': 'OpenCode Zen',
  'opencode-go': 'OpenCode Go',
  openai: 'OpenAI',
  custom: t('agent.custom'),
}

const codexProviderLabels = computed<Record<AgentProvider | 'custom', string>>(() => ({
  ccx: t('agent.localGateway'),
  openai: 'OpenAI',
  deepseek: 'DeepSeek',
  mimo: 'MiMo',
  compshare: 'Compshare',
  kimi: 'Kimi',
  glm: 'GLM',
  minimax: 'MiniMax',
  dashscope: 'DashScope',
  'opencode-zen': 'OpenCode Zen',
  'opencode-go': 'OpenCode Go',
  custom: t('agent.custom'),
}))

const agentPlatforms: AgentPlatform[] = ['claude', 'codex', 'opencode']

// Module-level singletons
const agentStatuses = ref<Record<AgentPlatform, AgentConfigStatus | null>>({
  claude: null,
  codex: null,
  opencode: null,
})
const configLoading = ref(false)
const selectedClaudeProvider = ref<AgentProvider>('ccx')
const claudeProviderKeys = ref<Record<AgentProvider, string>>({
  ccx: '',
  deepseek: '',
  mimo: '',
  compshare: '',
  kimi: '',
  glm: '',
  minimax: '',
  dashscope: '',
  'opencode-zen': '',
  'opencode-go': '',
  openai: '',
})
const savedProviderKeys = ref<Record<string, string>>({})
const codexOpenAIKey = ref('')
const openCodeOpenAIKey = ref('')
const claudeMimoBaseUrl = ref('https://api.xiaomimimo.com/anthropic')
const selectedMimoPlan = ref('https://api.xiaomimimo.com/anthropic')
const selectedDashScopePlan = ref('https://dashscope.aliyuncs.com/apps/anthropic')
const selectedCodexProvider = ref<AgentProvider>('ccx')
const codexMode = ref<'quick' | 'plugin'>('quick')
const selectedOpenCodeProvider = ref<AgentProvider>('ccx')

// Diff preview dialog state
const diffDialogOpen = ref(false)
const diffResult = ref<ConfigDiffResult | null>(null)
const diffMode = ref<'apply' | 'restore'>('apply')
const diffLoading = ref(false)
const diffPendingPlatform = ref<AgentPlatform>('claude')
const diffWarning = ref<string | undefined>(undefined)

const isClaudeProvider = (value?: string): value is AgentProvider => {
  return value === 'ccx' || value === 'deepseek' || value === 'mimo' || value === 'compshare' || value === 'kimi' || value === 'glm' || value === 'minimax' || value === 'dashscope' || value === 'opencode-zen' || value === 'opencode-go'
}

// Codex 支持快捷模式/插件模式切换的第三方 provider
const isCodexThirdPartyWithMode = (provider?: string) => {
  return provider === 'dashscope' || provider === 'opencode-zen' || provider === 'opencode-go'
}

const claudeProviderLabel = (value?: string) => {
  if (!value) return t('agent.statusDetecting')
  return claudeProviderLabels[value as AgentProvider | 'custom'] || value
}

const codexProviderLabel = (value?: string) => {
  if (!value) return t('agent.statusDetecting')
  return codexProviderLabels.value[value as AgentProvider | 'custom'] || value
}

const openCodeProviderLabel = (value?: string) => {
  if (!value) return t('agent.statusDetecting')
  return codexProviderLabels.value[value as AgentProvider | 'custom'] || value
}

const claudeTargetBaseUrl = () => {
  switch (selectedClaudeProvider.value) {
    case 'ccx':
      return agentStatuses.value.claude?.targetBaseUrl || t('agent.localGateway')
    case 'deepseek':
      return 'https://api.deepseek.com/anthropic'
    case 'mimo':
      return claudeMimoBaseUrl.value || 'https://api.xiaomimimo.com/anthropic'
    case 'compshare':
      return 'https://cp.compshare.cn'
    case 'kimi':
      return 'https://api.moonshot.cn/anthropic'
    case 'glm':
      return 'https://open.bigmodel.cn/api/anthropic'
    case 'minimax':
      return 'https://api.minimaxi.com/anthropic'
    case 'dashscope':
      return selectedDashScopePlan.value
    case 'opencode-zen':
      return 'https://opencode.ai/zen'
    case 'opencode-go':
      return 'https://opencode.ai/zen/go'
    default:
      return ''
  }
}

const codexTargetBaseUrl = () => {
  switch (selectedCodexProvider.value) {
    case 'ccx':
      return agentStatuses.value.codex?.targetBaseUrl || t('agent.localGateway')
    case 'openai':
      return 'https://api.openai.com/v1'
    case 'dashscope':
      return 'https://dashscope.aliyuncs.com/compatible-mode/v1'
    case 'opencode-zen':
      return 'https://opencode.ai/zen/v1'
    case 'opencode-go':
      return 'https://opencode.ai/zen/go/v1'
    default:
      return ''
  }
}

const openCodeTargetBaseUrl = () => {
  switch (selectedOpenCodeProvider.value) {
    case 'ccx':
      return agentStatuses.value.opencode?.targetBaseUrl || t('agent.localGateway')
    case 'deepseek':
      return 'https://api.deepseek.com/v1'
    case 'kimi':
      return 'https://api.moonshot.cn/v1'
    case 'glm':
      return 'https://open.bigmodel.cn/api/paas/v4'
    case 'minimax':
      return 'https://api.minimaxi.com/v1'
    case 'opencode-zen':
      return 'https://opencode.ai/zen/v1'
    case 'opencode-go':
      return 'https://opencode.ai/zen/go/v1'
    default:
      return ''
  }
}

const agentStatusText = (item: AgentConfigStatus | null) => {
  if (!item) return t('agent.statusDetecting')
  if (item.configured) return t('agent.statusConfigured')
  if (item.needsUpdate) return t('agent.statusPortMismatch')
  return t('agent.statusUnconfigured')
}

const agentStatusClass = (item: AgentConfigStatus | null) => {
  if (!item) return 'starting'
  if (item.configured) return 'running'
  if (item.needsUpdate) return 'starting'
  return 'stopped'
}

const resolveMiMoPlan = (url: string): string => {
  const known = [
    'https://api.xiaomimimo.com/anthropic',
    'https://token-plan-cn.xiaomimimo.com/anthropic',
    'https://token-plan-sgp.xiaomimimo.com/anthropic',
    'https://token-plan-ams.xiaomimimo.com/anthropic',
  ]
  return known.includes(url) ? url : ''
}

const resolveDashScopePlan = (url: string): string => {
  const known = [
    'https://dashscope.aliyuncs.com/apps/anthropic',
    'https://coding.dashscope.aliyuncs.com/apps/anthropic',
  ]
  return known.includes(url) ? url : ''
}

const loadAgentStatuses = async () => {
  configLoading.value = true
  try {
    const [claude, codex, opencode, keys] = await Promise.all([
      GetAgentConfigStatus('claude') as Promise<AgentConfigStatus>,
      GetAgentConfigStatus('codex') as Promise<AgentConfigStatus>,
      GetAgentConfigStatus('opencode') as Promise<AgentConfigStatus>,
      GetSavedProviderKeys(),
    ])
    agentStatuses.value = { claude, codex, opencode }
    savedProviderKeys.value = Object.fromEntries(
      Object.entries(keys).filter((entry): entry is [string, string] => typeof entry[1] === 'string')
    )
    if (isClaudeProvider(claude.provider)) {
      selectedClaudeProvider.value = claude.provider
    }
    if (claude.provider === 'mimo' && claude.currentBaseUrl) {
      claudeMimoBaseUrl.value = claude.currentBaseUrl
      selectedMimoPlan.value = resolveMiMoPlan(claude.currentBaseUrl)
    }
    if (claude.provider === 'dashscope' && claude.currentBaseUrl) {
      selectedDashScopePlan.value = resolveDashScopePlan(claude.currentBaseUrl)
    }
    if (codex.provider && codex.provider !== 'ccx' && codex.provider !== '') {
      selectedCodexProvider.value = codex.provider as AgentProvider
    } else {
      selectedCodexProvider.value = 'ccx'
    }
    // 恢复所有支持 mode 切换的 provider 的 mode 状态
    if (codex.provider === 'ccx' || codex.provider === 'dashscope' || codex.provider === 'opencode-zen' || codex.provider === 'opencode-go') {
      codexMode.value = codex.mode === 'plugin' ? 'plugin' : 'quick'
    }
    if (opencode.provider && opencode.provider !== 'ccx' && opencode.provider !== '') {
      selectedOpenCodeProvider.value = opencode.provider as AgentProvider
    } else {
      selectedOpenCodeProvider.value = 'ccx'
    }
  } catch (error) {
    // error is handled by caller
  } finally {
    configLoading.value = false
  }
}

const findSavedKey = (provider: string, planID?: string): string => {
  if (planID) {
    const planKey = savedProviderKeys.value[`claude:${provider}:${planID}`]
    if (planKey) return planKey
  }
  return savedProviderKeys.value[`claude:${provider}`] || ''
}

const canApplyAgent = (platform: AgentPlatform) => {
  if (configLoading.value) return false
  if (platform === 'codex') {
    // CCX 和 OpenAI 不需要验证，后端会使用代理 key 或 auth.json 中现有的 key
    if (selectedCodexProvider.value === 'ccx' || selectedCodexProvider.value === 'openai') {
      return true
    }
    // 第三方 provider 必须有输入的 key 或已保存的 key
    const inputKey = codexOpenAIKey.value.trim()
    const hasSaved = !!savedProviderKeys.value[`codex:${selectedCodexProvider.value}`]
    return inputKey !== '' || hasSaved
  }
  if (platform === 'opencode') {
    if (selectedOpenCodeProvider.value === 'ccx') {
      return true
    }
    const inputKey = openCodeOpenAIKey.value.trim()
    const hasSaved = !!savedProviderKeys.value[`codex:${selectedOpenCodeProvider.value}`]
    return inputKey !== '' || hasSaved
  }
  if (selectedClaudeProvider.value === 'ccx') return true
  const provider = selectedClaudeProvider.value
  const inputKey = claudeProviderKeys.value[provider].trim()
  const planID = provider === 'mimo' ? selectedMimoPlan.value
    : provider === 'dashscope' ? selectedDashScopePlan.value
    : undefined
  const hasSaved = !!findSavedKey(provider, planID)
  return inputKey !== '' || hasSaved
}

const applyAgent = async (platform: AgentPlatform) => {
  configLoading.value = true
  try {
    const request: ApplyAgentConfigRequest = { platform }
    if (platform === 'claude') {
      request.provider = selectedClaudeProvider.value
      if (selectedClaudeProvider.value !== 'ccx') {
        const inputKey = claudeProviderKeys.value[selectedClaudeProvider.value].trim()
        const planID = selectedClaudeProvider.value === 'mimo' ? selectedMimoPlan.value
          : selectedClaudeProvider.value === 'dashscope' ? selectedDashScopePlan.value
          : undefined
        request.apiKey = inputKey || findSavedKey(selectedClaudeProvider.value, planID)
      }
      if (selectedClaudeProvider.value === 'mimo') {
        request.baseUrl = claudeMimoBaseUrl.value.trim()
      }
      if (selectedClaudeProvider.value === 'dashscope') {
        request.baseUrl = selectedDashScopePlan.value.trim()
      }
    }
    if (platform === 'codex') {
      request.provider = selectedCodexProvider.value
      if (selectedCodexProvider.value === 'ccx' || isCodexThirdPartyWithMode(selectedCodexProvider.value)) {
        request.mode = codexMode.value
      }
      if (selectedCodexProvider.value !== 'ccx') {
        const inputKey = codexOpenAIKey.value.trim()
        request.apiKey = inputKey || savedProviderKeys.value[`codex:${selectedCodexProvider.value}`] || ''
      }
    }
    if (platform === 'opencode') {
      request.provider = selectedOpenCodeProvider.value
      if (selectedOpenCodeProvider.value !== 'ccx') {
        const inputKey = openCodeOpenAIKey.value.trim()
        request.apiKey = inputKey || savedProviderKeys.value[`codex:${selectedOpenCodeProvider.value}`] || ''
      }
    }
    await ApplyAgentConfig(request)
    await loadAgentStatuses()
  } finally {
    configLoading.value = false
  }
}

const showApplyPreview = async (platform: AgentPlatform) => {
  const request: ApplyAgentConfigRequest = { platform }
  if (platform === 'claude') {
    request.provider = selectedClaudeProvider.value
    if (selectedClaudeProvider.value !== 'ccx') {
      const inputKey = claudeProviderKeys.value[selectedClaudeProvider.value].trim()
      request.apiKey = inputKey || findSavedKey(selectedClaudeProvider.value, selectedMimoPlan.value)
    }
    if (selectedClaudeProvider.value === 'mimo') {
      request.baseUrl = claudeMimoBaseUrl.value.trim()
    }
    if (selectedClaudeProvider.value === 'dashscope') {
      request.baseUrl = selectedDashScopePlan.value.trim()
    }
  }
  if (platform === 'codex') {
    request.provider = selectedCodexProvider.value
    if (selectedCodexProvider.value === 'ccx' || isCodexThirdPartyWithMode(selectedCodexProvider.value)) {
      request.mode = codexMode.value
    }
    if (selectedCodexProvider.value !== 'ccx') {
      const inputKey = codexOpenAIKey.value.trim()
      request.apiKey = inputKey || savedProviderKeys.value[`codex:${selectedCodexProvider.value}`] || ''
    }
  }
  if (platform === 'opencode') {
    request.provider = selectedOpenCodeProvider.value
    if (selectedOpenCodeProvider.value !== 'ccx') {
      const inputKey = openCodeOpenAIKey.value.trim()
      request.apiKey = inputKey || savedProviderKeys.value[`codex:${selectedOpenCodeProvider.value}`] || ''
    }
  }
  // 检测模式切换，显示会话迁移警告
  diffWarning.value = undefined
  if (platform === 'codex' && (selectedCodexProvider.value === 'ccx' || isCodexThirdPartyWithMode(selectedCodexProvider.value))) {
    const currentMode = agentStatuses.value.codex?.mode === 'plugin' ? 'plugin' : 'quick'
    if (currentMode !== codexMode.value) {
      diffWarning.value = t('agent.sessionMigrationWarning')
    }
  }
  diffPendingPlatform.value = platform
  diffMode.value = 'apply'
  diffDialogOpen.value = true
  diffLoading.value = true
  diffResult.value = null
  try {
    diffResult.value = await PreviewAgentConfigDiff(request) as ConfigDiffResult
  } catch {
    diffResult.value = null
  } finally {
    diffLoading.value = false
  }
}

const confirmApply = async () => {
  diffDialogOpen.value = false
  await applyAgent(diffPendingPlatform.value)
}

const showRestorePreview = async (platform: AgentPlatform) => {
  diffPendingPlatform.value = platform
  diffMode.value = 'restore'
  diffDialogOpen.value = true
  diffLoading.value = true
  diffResult.value = null
  try {
    diffResult.value = await PreviewRestoreConfigDiff(platform) as ConfigDiffResult
  } catch {
    diffResult.value = null
  } finally {
    diffLoading.value = false
  }
}

const confirmRestore = async () => {
  diffDialogOpen.value = false
  await restoreAgent(diffPendingPlatform.value)
}

const closeDiffDialog = () => {
  diffDialogOpen.value = false
}

const restoreAgent = async (platform: AgentPlatform) => {
  configLoading.value = true
  try {
    await RestoreAgentConfig(platform)
  } finally {
    configLoading.value = false
  }
}

export function useAgentConfig() {
  return {
    agentStatuses,
    configLoading,
    selectedClaudeProvider,
    claudeProviderKeys,
    savedProviderKeys,
    codexOpenAIKey,
    codexMode,
    openCodeOpenAIKey,
    claudeMimoBaseUrl,
    selectedMimoPlan,
    selectedDashScopePlan,
    agentLabels,
    claudeProviderLabels,
    codexProviderLabels,
    agentPlatforms,
    isClaudeProvider,
    claudeProviderLabel,
    claudeTargetBaseUrl,
    agentStatusText,
    agentStatusClass,
    loadAgentStatuses,
    canApplyAgent,
    applyAgent,
    restoreAgent,
    selectedCodexProvider,
    selectedOpenCodeProvider,
    codexProviderLabel,
    codexTargetBaseUrl,
    openCodeProviderLabel,
    openCodeTargetBaseUrl,
    // Diff preview
    diffDialogOpen,
    diffResult,
    diffMode,
    diffLoading,
    diffPendingPlatform,
    diffWarning,
    showApplyPreview,
    showRestorePreview,
    confirmApply,
    confirmRestore,
    closeDiffDialog,
  }
}
