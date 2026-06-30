<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  CheckCircle2,
  Copy,
  Loader2,
} from 'lucide-vue-next'
import { useConsoleChannels } from '@/composables/useConsoleChannels'
import { useLanguage } from '@/composables/useLanguage'
import { AdminApiError, useAdminApi } from '@/composables/useAdminApi'
import {
  COPILOT_OAUTH_DEVICE_CODE_PATH,
  COPILOT_OAUTH_TOKEN_PATH,
  type CopilotDeviceCodeResponse,
  type CopilotTokenResponse,
} from '@/services/admin-api'
import {
  buildChannelPayload,
  createModelCapabilityRow,
  modelCapabilitiesToRows,
  modelCapabilityRowsToRecord,
  resolveBuiltinUpstreamModelCapability,
  type ModelCapabilityRow,
} from '@/utils/channel-payload'
import { supportsAdvancedChannelOptions, supportsReasoningMapping } from '@/utils/channel-advanced-options'
import {
  extractChannelNamePrefix,
  filterValidSupportedModelPatterns,
  parseSupportedModelInput,
  syncBaseUrlsFormState,
} from '@/utils/channel-dialog-state'
import { getChannelTypeApi, type ManagedChannelType } from '@/utils/channel-type-api'
import { buildExpectedRequestUrls } from '@/utils/expected-request-urls'
import { sortModelNamesDesc } from '@/utils/model-priority'
import { parseQuickInput } from '@/utils/quick-input-parser'
import { defaultStreamTimeouts } from '@/utils/stream-timeout-presets'
import { claudeMessagesPresets } from '@/generated/claude-messages-presets'
import { codexResponsesPresets } from '@/generated/codex-responses-presets'
import { openaiMessagesPresets } from '@/generated/openai-messages-presets'
import { openExternalLink } from '@/lib/external-link'
import type { Channel, CompatDiagnoseResult, DisabledKeyInfo } from '@/services/admin-api'
import ChannelEditorHeader from './channel-edit/ChannelEditorHeader.vue'
import QuickCreatePanel from './channel-edit/QuickCreatePanel.vue'
import BasicConfigPanel from './channel-edit/BasicConfigPanel.vue'
import AuthPanel from './channel-edit/AuthPanel.vue'
import ModelMappingPanel from './channel-edit/ModelMappingPanel.vue'
import ModelCapabilityPanel from './channel-edit/ModelCapabilityPanel.vue'
import AdvancedPanel from './channel-edit/AdvancedPanel.vue'
import CustomHeadersPanel from './channel-edit/CustomHeadersPanel.vue'
import CustomParamsPanel from './channel-edit/CustomParamsPanel.vue'
import StreamTimeoutPanel from './channel-edit/StreamTimeoutPanel.vue'

interface Props {
  channel?: Channel | null
  channelType: ManagedChannelType
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
  (e: 'test-capability', channel: Channel): void
}>()

const { t } = useLanguage()
const { saveChannel, restoreApiKey } = useConsoleChannels()

const isEditMode = computed(() => !!props.channel)
const isMac = computed(() => typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform))
const saving = ref(false)
const restoringKey = ref('')
const error = ref('')
const success = ref('')
const diagnosingCompat = ref(false)
const quickInput = ref('')
const quickServiceTypeTouched = ref(false)
const existingApiKeys = ref<string[]>([])
const newApiKeysText = ref('')
const copiedKeyIndex = ref<number | null>(null)
const duplicateKeyIndex = ref<number | null>(null)
let duplicateKeyTimer: ReturnType<typeof setTimeout> | null = null
const localRestoredKeys = ref<Set<string>>(new Set())

// GitHub Copilot OAuth 状态
const adminApi = useAdminApi()
const copilotOAuthLoading = ref(false)
const copilotPolling = ref(false)
const copilotOAuthError = ref('')
const copilotOAuthSuccess = ref(false)
const copilotUserCode = ref('')
const copilotVerificationUri = ref('')
const copilotDeviceCode = ref('')
const copilotUserCodeCopied = ref(false)
let copilotPollTimer: ReturnType<typeof setTimeout> | null = null
let copilotCopyTimer: ReturnType<typeof setTimeout> | null = null

function clearCopilotPollTimer() {
  if (copilotPollTimer !== null) { clearTimeout(copilotPollTimer); copilotPollTimer = null }
}

function clearCopilotCopyTimer() {
  if (copilotCopyTimer !== null) { clearTimeout(copilotCopyTimer); copilotCopyTimer = null }
}

function clearCopilotAuthorizationCode() {
  copilotDeviceCode.value = ''
  copilotUserCode.value = ''
  copilotVerificationUri.value = ''
  copilotUserCodeCopied.value = false
  clearCopilotCopyTimer()
}

async function copyCopilotUserCode() {
  const userCode = copilotUserCode.value.trim()
  if (!userCode) return
  try {
    await navigator.clipboard.writeText(userCode)
    clearCopilotCopyTimer()
    copilotUserCodeCopied.value = true
    copilotCopyTimer = setTimeout(() => {
      copilotUserCodeCopied.value = false
      copilotCopyTimer = null
    }, 1200)
  } catch {
    // clipboard 不可用时静默
  }
}

async function pollCopilotToken(intervalSeconds: number) {
  if (!copilotDeviceCode.value) return
  copilotPolling.value = true
  try {
    const token = await adminApi.post<CopilotTokenResponse>(COPILOT_OAUTH_TOKEN_PATH, { deviceCode: copilotDeviceCode.value })
    if (token.accessToken) {
      if (!existingApiKeys.value.includes(token.accessToken)) existingApiKeys.value.push(token.accessToken)
      copilotOAuthSuccess.value = true
      copilotOAuthError.value = ''
      copilotPolling.value = false
      copilotOAuthLoading.value = false
      clearCopilotPollTimer()
      clearCopilotAuthorizationCode()
      return
    }
    if (token.error === 'expired_token') {
      copilotOAuthError.value = t('copilotOAuth.expired')
      copilotPolling.value = false; copilotOAuthLoading.value = false; clearCopilotPollTimer(); return
    }
    if (token.error && token.error !== 'authorization_pending') {
      copilotOAuthError.value = token.errorDescription || token.error
      copilotPolling.value = false; copilotOAuthLoading.value = false; clearCopilotPollTimer(); return
    }
  } catch (err) {
    copilotOAuthError.value = err instanceof Error ? err.message : String(err)
    copilotPolling.value = false; copilotOAuthLoading.value = false; clearCopilotPollTimer(); return
  }
  copilotPollTimer = setTimeout(() => pollCopilotToken(intervalSeconds), Math.max(intervalSeconds, 5) * 1000)
}

async function startCopilotOAuth() {
  clearCopilotPollTimer()
  clearCopilotAuthorizationCode()
  copilotOAuthLoading.value = true
  copilotOAuthError.value = ''
  copilotOAuthSuccess.value = false
  try {
    const device = await adminApi.post<CopilotDeviceCodeResponse>(COPILOT_OAUTH_DEVICE_CODE_PATH)
    copilotDeviceCode.value = device.deviceCode
    copilotUserCode.value = device.userCode
    copilotVerificationUri.value = device.verificationUri
    await openCopilotAuthorization()
    await pollCopilotToken(device.interval || 5)
  } catch (err) {
    copilotOAuthError.value = err instanceof Error ? err.message : String(err)
    copilotOAuthLoading.value = false; copilotPolling.value = false
  }
}

async function openCopilotAuthorization() {
  if (!copilotVerificationUri.value) return
  try {
    await openExternalLink(copilotVerificationUri.value)
  } catch (err) {
    copilotOAuthError.value = err instanceof Error ? err.message : String(err)
  }
}

type KeyModelsStatus = {
  loading?: boolean
  success?: boolean
  error?: string
  statusCode?: string | number
  modelCount?: number
}
const keyModelsStatus = ref<Map<string, KeyModelsStatus>>(new Map())

type ReasoningEffort = 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
interface ModelMappingRow {
  id: number
  source: string
  target: string
  reasoning: ReasoningEffort | ''
  noVision: boolean
}
interface HeaderRow {
  id: number
  key: string
  value: string
}

let rowId = 0
const activeSection = ref('basic')
const sectionRefs = ref<Record<string, HTMLElement | null>>({})
const dialogRef = ref<HTMLElement | null>(null)
let scrollRoot: Element | null = null
let scrollHandler: (() => void) | null = null

// 导航 section 定义（使用 computed 保证语言切换后更新）
const sections = computed(() => [
  { id: 'basic', label: t('channelEditor.nav.basic') },
  { id: 'auth', label: t('channelEditor.nav.auth') },
  { id: 'redirect', label: t('channelEditor.nav.redirect') },
  { id: 'advanced', label: t('channelEditor.nav.advanced') },
  { id: 'custom', label: t('channelEditor.nav.custom') },
])

function scrollToSection(id: string) {
  activeSection.value = id
  const el = sectionRefs.value[id]
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

function setSectionRef(id: string, el: any) {
  sectionRefs.value[id] = el as HTMLElement | null
}

function updateActiveSectionFromScroll() {
  if (!scrollRoot) return
  const rootTop = scrollRoot.getBoundingClientRect().top
  let current = sections.value[0]?.id || 'basic'

  // 遍历所有 section，找到最后一个进入视口顶部的 section
  for (const s of sections.value) {
    const el = sectionRefs.value[s.id]
    if (!el) continue
    const top = el.getBoundingClientRect().top - rootTop
    // 使用较小的阈值（60px），确保更灵敏的切换
    if (top <= 60) {
      current = s.id
    } else {
      break
    }
  }

  if (activeSection.value !== current) {
    activeSection.value = current
  }
}
const modelMappingRows = ref<ModelMappingRow[]>([])
const modelCapabilityRows = ref<ModelCapabilityRow[]>([])
const incompleteMappedTargetSuffix = /[._:/-]$/
const isCompleteMappedTargetModel = (model: string) => !!model && !incompleteMappedTargetSuffix.test(model)
const mappedTargetModels = computed(() => {
  const seen = new Set<string>()
  const models = [
    ...modelMappingRows.value.map(row => row.target.trim()),
    form.visionFallbackModel.trim(),
  ]

  return models.filter(model => {
    const key = model.toLowerCase()
    if (!isCompleteMappedTargetModel(model) || seen.has(key)) return false
    seen.add(key)
    return true
  })
})
const isMappingTargetEditing = ref(false)
const hasPendingModelCapabilitySync = ref(false)
const newModelMapping = reactive<ModelMappingRow>({ id: 0, source: '', target: '', reasoning: '', noVision: false })
const headerRows = ref<HeaderRow[]>([])
const newHeader = reactive<HeaderRow>({ id: 0, key: '', value: '' })

// 目标模型自动完成建议
const showTargetSuggestions = ref(false)
const activeTargetInputId = ref<string | null>(null)
const targetInputFilter = ref('')
const targetModelOptions = ref<string[]>([])
const fetchedModelsError = ref('')
const hasTriedFetchModels = ref(false)

function getFilteredTargetModels(filter: string): string[] {
  const models = targetModelDatalist.value
  const value = filter.trim()
  if (!value) return models.slice(0, 20)

  const lower = value.toLowerCase()

  // 查找完全匹配的项
  const exactIndex = models.findIndex(m => m.toLowerCase() === lower)

  // 如果找到完全匹配项，返回以该项为中心的窗口
  if (exactIndex >= 0) {
    return getTargetModelWindow(exactIndex)
  }

  // 否则过滤包含该值的项
  const filtered = models.filter(m => m.toLowerCase().includes(lower))

  // 如果只有一个匹配项，返回以该项为中心的窗口
  if (filtered.length === 1) {
    const index = models.findIndex(m => m === filtered[0])
    if (index >= 0) return getTargetModelWindow(index)
  }

  return filtered.slice(0, 20)
}

function getTargetModelWindow(index: number): string[] {
  const models = targetModelDatalist.value
  const limit = 20
  const before = 8
  const maxStart = Math.max(models.length - limit, 0)
  const start = Math.min(Math.max(index - before, 0), maxStart)
  return models.slice(start, start + limit)
}

const filteredTargetModels = computed(() => getFilteredTargetModels(targetInputFilter.value))

function showTargetDropdown(inputId: string, currentValue: string) {
  activeTargetInputId.value = inputId
  targetInputFilter.value = currentValue
  showTargetSuggestions.value = targetModelDatalist.value.length > 0
}

function hideTargetDropdown() {
  showTargetSuggestions.value = false
  activeTargetInputId.value = null
  finishMappingTargetEdit()
}

function handlePointerDown(e: PointerEvent) {
  const target = e.target as Element | null
  if (target?.closest('[data-target-model-picker]') || target?.closest('[data-source-model-picker]')) return
  hideTargetDropdown()
  hideSourceDropdown()
}

// 源模型自动完成建议
const showSourceSuggestions = ref(false)
const activeSourceInputId = ref<string | null>(null)
const sourceInputFilter = ref('')

function getFilteredSourceModels(filter: string): string[] {
  const models = sourceModelOptions.value
  const value = filter.trim()
  if (!value) return models.slice(0, 80)

  const lower = value.toLowerCase()
  const filtered = models.filter(m => m.toLowerCase().includes(lower))

  // 如果只有一个匹配项且完全匹配当前值，返回该项周围的窗口
  if (filtered.length === 1 && filtered[0].toLowerCase() === lower) {
    const index = models.findIndex(m => m === filtered[0])
    if (index >= 0) return getSourceModelWindow(index, models)
  }

  return filtered.slice(0, 80)
}

function getSourceModelWindow(index: number, models: string[]): string[] {
  const limit = 80
  const before = 30
  const maxStart = Math.max(models.length - limit, 0)
  const start = Math.min(Math.max(index - before, 0), maxStart)
  return models.slice(start, start + limit)
}

const filteredSourceModels = computed(() => getFilteredSourceModels(sourceInputFilter.value))

function showSourceDropdown(inputId: string, currentValue: string) {
  activeSourceInputId.value = inputId
  sourceInputFilter.value = currentValue
  showSourceSuggestions.value = true
}

function hideSourceDropdown() {
  showSourceSuggestions.value = false
  activeSourceInputId.value = null
}

function selectSourceModel(inputId: string, model: string) {
  if (inputId === 'new-source') {
    newModelMapping.source = model
  }
  showSourceSuggestions.value = false
  activeSourceInputId.value = null
}

function selectTargetModel(inputId: string, model: string) {
  // inputId 格式: 'row-{index}' 或 'new'
  if (inputId === 'new') {
    newModelMapping.target = model
  } else if (inputId.startsWith('row-')) {
    const index = parseInt(inputId.slice(4), 10)
    if (!isNaN(index) && modelMappingRows.value[index]) {
      modelMappingRows.value[index].target = model
    }
  }
  // 立即隐藏（不使用延迟）
  showTargetSuggestions.value = false
  activeTargetInputId.value = null
  finishMappingTargetEdit()
}

const reasoningParamStyleOptions = [
  { label: 'reasoning.effort', value: 'reasoning' },
  { label: 'reasoning_effort', value: 'reasoning_effort' },
  { label: 'thinking (JD/GLM)', value: 'thinking' },
]

const textVerbosityOptions = [
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
]

// 思考强度（effort）—— 模型映射第三列使用
// 注意：reka-ui 的 SelectItem 不允许空字符串 value，用 DEFAULT_SELECT_VALUE 哨兵代表"默认/不设置"
const DEFAULT_SELECT_VALUE = 'default'

const reasoningEffortOptions = computed(() => [
  { label: t('channelEditor.compat.selectDefault'), value: DEFAULT_SELECT_VALUE },
  { label: 'None', value: 'none' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
  { label: 'XHigh', value: 'xhigh' },
  { label: 'Max', value: 'max' },
])

const form = reactive({
  name: '',
  description: '',
  serviceType: '' as 'openai' | 'claude' | 'gemini' | 'responses' | 'copilot' | '',
  authHeader: 'auto' as 'auto' | 'bearer' | 'x-api-key' | '',
  baseUrl: '',
  baseUrlsText: '',
  website: '',
  proxyUrl: '',
  requestTimeoutMs: '' as string | number,
  responseHeaderTimeoutMs: '' as string | number,
  streamFirstContentTimeoutEnabled: false,
  streamFirstContentTimeoutMs: defaultStreamTimeouts.firstContentMs,
  streamInactivityTimeoutEnabled: false,
  streamInactivityTimeoutMs: defaultStreamTimeouts.inactivityMs,
  streamToolCallIdleTimeoutEnabled: false,
  streamToolCallIdleTimeoutMs: defaultStreamTimeouts.toolCallIdleMs,
  rateLimitRpm: '' as string | number,
  rateLimitWindowMinutes: '' as string | number,
  rateLimitMaxConcurrent: '' as string | number,
  rateLimitAutoFromHeaders: true,
  routePrefix: '',
  insecureSkipVerify: false,
  apiKeysText: '',
  customHeadersText: '{}',
  modelMappingText: '{}',
  modelCapabilitiesText: '',
  modelCapabilityRows: [] as ModelCapabilityRow[],
  defaultContextWindowTokens: '' as string | number,
  defaultMaxOutputTokens: '' as string | number,
  allowUnknownContext: false,
  reasoningMappingText: '{}',
  reasoningParamStyle: 'reasoning' as 'reasoning' | 'reasoning_effort' | 'thinking',
  textVerbosity: '' as 'low' | 'medium' | 'high' | '',
  supportedModelsText: '',
  visionFallbackModel: '',
  visionFallbackReasoningEffort: '' as ReasoningEffort | '',
  noVision: false,
  historicalImageTurnLimit: 0,
  passbackReasoningContent: false,
  passbackThinkingBlocks: false,
  fastMode: false,
  lowQuality: false,
  injectDummyThoughtSignature: false,
  stripThoughtSignature: false,
  stripEmptyTextBlocks: false,
  normalizeSystemRoleToTopLevel: false,
  normalizeMetadataUserId: true,
  stripBillingHeader: false,
  normalizeNonstandardChatRoles: false,
  autoBlacklistBalance: true,
  codexNativeToolPassthrough: false,
  codexToolCompat: false,
  stripCodexClientTools: false,
  stripImageGenerationTool: false,
})

const disabledApiKeys = computed<DisabledKeyInfo[]>(() => props.channel?.disabledApiKeys ?? [])
const historicalApiKeys = computed(() => props.channel?.historicalApiKeys ?? [])

const quickDetection = computed(() => parseQuickInput(quickInput.value, form.serviceType || undefined))
const detectedBaseUrls = computed(() => quickDetection.value.detectedBaseUrls)
const detectedApiKeys = computed(() => quickDetection.value.detectedApiKeys)
const detectedServiceType = computed(() => quickDetection.value.detectedServiceType)

// 生成随机字符串（用于渠道名称后缀）
function generateRandomString(length: number): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

const randomSuffix = ref(generateRandomString(6))

// 自动生成的渠道名称
const generatedChannelName = computed(() => {
  const firstUrl = detectedBaseUrls.value[0]
  if (!firstUrl) return `channel-${randomSuffix.value}`
  const prefix = extractChannelNamePrefix(firstUrl)
  return `${prefix}-${randomSuffix.value}`
})

watch(detectedServiceType, (serviceType) => {
  if (isEditMode.value || quickServiceTypeTouched.value || !serviceType) return
  form.serviceType = serviceType
})

function resetForm() {
  clearDuplicateKeyHighlight()
  randomSuffix.value = generateRandomString(6)
  form.name = ''
  form.description = ''
  form.serviceType = defaultServiceTypeForChannel()
  form.authHeader = 'auto'
  form.baseUrl = ''
  form.baseUrlsText = ''
  form.website = ''
  form.proxyUrl = ''
  form.requestTimeoutMs = ''
  form.responseHeaderTimeoutMs = ''
  form.streamFirstContentTimeoutEnabled = false
  form.streamFirstContentTimeoutMs = defaultStreamTimeouts.firstContentMs
  form.streamInactivityTimeoutEnabled = false
  form.streamInactivityTimeoutMs = defaultStreamTimeouts.inactivityMs
  form.streamToolCallIdleTimeoutEnabled = false
  form.streamToolCallIdleTimeoutMs = defaultStreamTimeouts.toolCallIdleMs
  form.rateLimitRpm = ''
  form.rateLimitWindowMinutes = ''
  form.rateLimitMaxConcurrent = ''
  form.rateLimitAutoFromHeaders = true
  form.routePrefix = ''
  form.insecureSkipVerify = false
  form.apiKeysText = ''
  form.customHeadersText = '{}'
  form.modelMappingText = '{}'
  form.modelCapabilitiesText = ''
  form.modelCapabilityRows = []
  form.defaultContextWindowTokens = ''
  form.defaultMaxOutputTokens = ''
  form.allowUnknownContext = false
  form.reasoningMappingText = '{}'
  form.reasoningParamStyle = 'reasoning'
  form.textVerbosity = ''
  form.supportedModelsText = ''
  form.visionFallbackModel = ''
  form.visionFallbackReasoningEffort = ''
  form.noVision = false
  form.historicalImageTurnLimit = 0
  form.passbackReasoningContent = false
  form.passbackThinkingBlocks = false
  form.fastMode = false
  form.lowQuality = false
  form.injectDummyThoughtSignature = false
  form.stripThoughtSignature = false
  form.stripEmptyTextBlocks = false
  form.normalizeSystemRoleToTopLevel = false
  form.normalizeMetadataUserId = true
  form.stripBillingHeader = false
  form.normalizeNonstandardChatRoles = false
  form.autoBlacklistBalance = true
  form.codexNativeToolPassthrough = false
  form.codexToolCompat = false
  form.stripCodexClientTools = false
  form.stripImageGenerationTool = false
  quickInput.value = ''
  quickServiceTypeTouched.value = false
  existingApiKeys.value = []
  newApiKeysText.value = ''
  copiedKeyIndex.value = null
  keyModelsStatus.value.clear()
  targetModelOptions.value = []
  hasTriedFetchModels.value = false
  localRestoredKeys.value = new Set()
  modelMappingRows.value = []
  modelCapabilityRows.value = []
  headerRows.value = []
  error.value = ''
  success.value = ''
}

function defaultServiceTypeForChannel() {
  if (props.channelType === 'gemini') return 'gemini'
  if (props.channelType === 'responses') return 'responses'
  if (props.channelType === 'messages') return 'claude'
  return 'openai'
}

function populateFromChannel(ch: Channel) {
  form.name = ch.name || ''
  form.description = ch.description || ''
  form.serviceType = ch.serviceType || defaultServiceTypeForChannel()
  form.authHeader = ch.authHeader || 'auto'
  // baseUrls 多 URL 时已包含主 URL；否则回退单个 baseUrl。form.baseUrl 由 watch 派生
  form.baseUrlsText = (ch.baseUrls?.length ? ch.baseUrls : [ch.baseUrl].filter(Boolean)).join('\n')
  form.website = ch.website || ''
  form.proxyUrl = ch.proxyUrl || ''
  form.requestTimeoutMs = ch.requestTimeoutMs || ''
  form.responseHeaderTimeoutMs = ch.responseHeaderTimeoutMs || ''
  form.streamFirstContentTimeoutEnabled = !!(ch.streamFirstContentTimeoutMs && ch.streamFirstContentTimeoutMs > 0)
  form.streamFirstContentTimeoutMs = ch.streamFirstContentTimeoutMs && ch.streamFirstContentTimeoutMs > 0 ? ch.streamFirstContentTimeoutMs : defaultStreamTimeouts.firstContentMs
  form.streamInactivityTimeoutEnabled = !!(ch.streamInactivityTimeoutMs && ch.streamInactivityTimeoutMs > 0)
  form.streamInactivityTimeoutMs = ch.streamInactivityTimeoutMs && ch.streamInactivityTimeoutMs > 0 ? ch.streamInactivityTimeoutMs : defaultStreamTimeouts.inactivityMs
  form.streamToolCallIdleTimeoutEnabled = !!(ch.streamToolCallIdleTimeoutMs && ch.streamToolCallIdleTimeoutMs > 0)
  form.streamToolCallIdleTimeoutMs = ch.streamToolCallIdleTimeoutMs && ch.streamToolCallIdleTimeoutMs > 0 ? ch.streamToolCallIdleTimeoutMs : defaultStreamTimeouts.toolCallIdleMs
  form.rateLimitRpm = (ch.rateLimitRpm && ch.rateLimitRpm > 0) ? ch.rateLimitRpm : ''
  form.rateLimitWindowMinutes = (ch.rateLimitWindowMinutes && ch.rateLimitWindowMinutes > 0) ? ch.rateLimitWindowMinutes : ''
  form.rateLimitMaxConcurrent = (ch.rateLimitMaxConcurrent && ch.rateLimitMaxConcurrent > 0) ? ch.rateLimitMaxConcurrent : ''
  form.rateLimitAutoFromHeaders = !!ch.rateLimitAutoFromHeaders
  form.routePrefix = ch.routePrefix || ''
  form.insecureSkipVerify = ch.insecureSkipVerify ?? false
  existingApiKeys.value = [...(ch.apiKeys || [])]
  form.apiKeysText = ''
  newApiKeysText.value = ''
  copiedKeyIndex.value = null
  keyModelsStatus.value.clear()
  targetModelOptions.value = []
  hasTriedFetchModels.value = false
  localRestoredKeys.value = new Set()
  modelMappingRows.value = modelMappingFromChannel(ch)
  modelCapabilityRows.value = modelCapabilitiesToRows(ch.modelCapabilities || {}, () => ++rowId)
  form.modelCapabilityRows = modelCapabilityRows.value
  headerRows.value = headerRowsFromChannel(ch)
  form.customHeadersText = stringifyJson(ch.customHeaders)
  form.modelMappingText = stringifyJson(ch.modelMapping)
  form.modelCapabilitiesText = stringifyJson(ch.modelCapabilities)
  form.defaultContextWindowTokens = ch.defaultCapability?.contextWindowTokens ?? ''
  form.defaultMaxOutputTokens = ch.defaultCapability?.maxOutputTokens ?? ''
  form.allowUnknownContext = ch.allowUnknownContext ?? false
  form.reasoningMappingText = stringifyJson(ch.reasoningMapping)
  form.reasoningParamStyle = ch.reasoningParamStyle || 'reasoning'
  form.textVerbosity = ch.textVerbosity || ''
  form.supportedModelsText = (ch.supportedModels || []).join('\n')
  // noVisionModels 中命中映射 target 的由行级 toggle 表示，其余保留在文本框，避免重复展示
  form.visionFallbackModel = ch.visionFallbackModel || ''
  form.visionFallbackReasoningEffort = (ch.reasoningMapping?.[form.visionFallbackModel] || '') as ReasoningEffort | ''
  form.noVision = ch.noVision ?? false
  form.historicalImageTurnLimit = ch.historicalImageTurnLimit ?? 0
  form.passbackReasoningContent = ch.passbackReasoningContent ?? false
  form.passbackThinkingBlocks = ch.passbackThinkingBlocks ?? false
  form.fastMode = ch.fastMode ?? false
  form.lowQuality = ch.lowQuality ?? false
  form.injectDummyThoughtSignature = ch.injectDummyThoughtSignature ?? false
  form.stripThoughtSignature = ch.stripThoughtSignature ?? false
  form.stripEmptyTextBlocks = ch.stripEmptyTextBlocks ?? false
  form.normalizeSystemRoleToTopLevel = ch.normalizeSystemRoleToTopLevel ?? false
  form.normalizeMetadataUserId = ch.normalizeMetadataUserId ?? true
  form.stripBillingHeader = ch.stripBillingHeader ?? false
  form.normalizeNonstandardChatRoles = ch.normalizeNonstandardChatRoles ?? false
  form.autoBlacklistBalance = ch.autoBlacklistBalance ?? true
  form.codexNativeToolPassthrough = ch.codexNativeToolPassthrough ?? false
  form.codexToolCompat = ch.codexToolCompat ?? ch.stripCodexClientTools ?? false
  form.stripCodexClientTools = ch.stripCodexClientTools ?? ch.codexToolCompat ?? false
  form.stripImageGenerationTool = ch.stripImageGenerationTool ?? false
}

watch(() => props.channel, (ch) => {
  console.log('[watch channel] 渠道变化', { 
    hasChannel: !!ch, 
    hasMappings: ch?.modelMapping ? Object.keys(ch.modelMapping).length : 0 
  })
  resetForm()
  if (ch) {
    populateFromChannel(ch)
    syncModelCapabilitiesFromMapping()
    // 如果有模型映射配置，主动触发一次模型列表获取
    // 使用 nextTick 确保表单数据已填充完成
    if (ch.modelMapping && Object.keys(ch.modelMapping).length > 0) {
      console.log('[watch channel] 检测到模型映射，准备预加载')
      nextTick(() => {
        console.log('[watch channel] nextTick 后触发预加载')
        void fetchTargetModels()
      })
    }
  }
}, { immediate: true })

// baseUrlsText 是唯一的 Base URL 输入（每行一个，第一行为主），派生 form.baseUrl / form.baseUrls（对齐 WebUI）
watch([() => form.baseUrlsText, () => form.serviceType], () => {
  const { baseUrl } = syncBaseUrlsFormState(form.baseUrlsText, form.serviceType)
  form.baseUrl = baseUrl
}, { immediate: true })

// API Key 是否满足必填：现有 + 新增；编辑模式下有可恢复 disabled key 也算
const hasConfigurableKeys = computed(() => {
  if (existingApiKeys.value.length > 0) return true
  if (parseLines(newApiKeysText.value).length > 0) return true
  if (isEditMode.value && visibleDisabledKeys.value.length > 0) return true
  return false
})

const errors = computed(() => {
  const errs: Record<string, string> = {}
  if (isEditMode.value && !form.name.trim()) errs.name = t('channelEditor.basic.name.required')
  if (!isEditMode.value && !generatedChannelName.value.trim()) errs.name = t('channelEditor.basic.name.required')
  if (!form.serviceType) errs.serviceType = t('channelEditor.basic.serviceType.required')
  if (!form.baseUrlsText.trim()) errs.baseUrl = t('channelEditor.basic.baseUrl.required')
  // copilot 渠道通过 OAuth 登录，apiKeys 由登录流程填充，此处豁免必填校验
  if (!hasConfigurableKeys.value && form.serviceType !== 'copilot') errs.apiKeys = t('channelEditor.auth.apiKeyRequired')
  if (String(form.requestTimeoutMs).trim()) {
    const timeout = Number(form.requestTimeoutMs)
    if (!Number.isInteger(timeout) || timeout < 1000 || timeout > 300000) {
      errs.requestTimeoutMs = t('channelEditor.transport.requestTimeout.invalid')
    }
  }
  if (String(form.responseHeaderTimeoutMs).trim()) {
    const timeout = Number(form.responseHeaderTimeoutMs)
    if (!Number.isInteger(timeout) || timeout < 1000 || timeout > 300000) {
      errs.responseHeaderTimeoutMs = t('channelEditor.transport.responseHeaderTimeout.invalid')
    }
  }
  if (modelCapabilityRowsToRecord(modelCapabilityRows.value) === null) {
    errs.modelCapabilitiesText = t('addChannel.modelCapabilitiesRowsInvalid')
  }
  return errs
})

const isValid = computed(() => Object.keys(errors.value).length === 0)

function stringifyJson(value?: Record<string, unknown>) {
  if (!value || Object.keys(value).length === 0) return '{}'
  return JSON.stringify(value, null, 2)
}

function parseJsonObject<T extends Record<string, unknown>>(text: string, label: string): T {
  const trimmed = text.trim()
  if (!trimmed) return {} as T
  const parsed = JSON.parse(trimmed)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(`${label} must be a JSON object`)
  }
  return parsed as T
}

function parseLines(text: string) {
  return text
    .split('\n')
    .map(s => s.trim())
    .filter(Boolean)
}

function removeExistingApiKey(index: number) {
  existingApiKeys.value.splice(index, 1)
}

function getSubmitApiKeys() {
  return [...existingApiKeys.value, ...parseLines(newApiKeysText.value || form.apiKeysText)]
}

function handleQuickPaste(text: string) {
  const result = parseQuickInput(text, form.serviceType || undefined)
  // 统一写入 baseUrlsText（每行一个，第一行为主），form.baseUrl 由 watch 派生
  const detectedUrls = result.detectedBaseUrls.length ? result.detectedBaseUrls : [result.detectedBaseUrl].filter(Boolean)
  if (detectedUrls.length) form.baseUrlsText = detectedUrls.join('\n')
  if (result.detectedApiKeys.length) {
    existingApiKeys.value = [...new Set([...existingApiKeys.value, ...result.detectedApiKeys])]
  }
  if (result.detectedServiceType && !quickServiceTypeTouched.value) form.serviceType = result.detectedServiceType
  if (!form.serviceType) form.serviceType = defaultServiceTypeForChannel()
}

function updateQuickServiceType(value: string) {
  form.serviceType = value as typeof form.serviceType
  quickServiceTypeTouched.value = true
}

function buildSubmitPayload() {
  const payload = isEditMode.value
    ? buildCurrentPayload()
    : buildChannelPayload({
        name: generatedChannelName.value,
        serviceType: form.serviceType,
        authHeader: form.authHeader,
        baseUrl: form.baseUrl,
        baseUrls: parseLines(form.baseUrlsText),
        website: form.website,
        insecureSkipVerify: form.insecureSkipVerify,
        lowQuality: form.lowQuality,
        injectDummyThoughtSignature: form.injectDummyThoughtSignature,
        stripThoughtSignature: form.stripThoughtSignature,
        passbackReasoningContent: form.passbackReasoningContent,
        passbackThinkingBlocks: form.passbackThinkingBlocks,
        description: form.description,
        apiKeys: getSubmitApiKeys(),
        modelMapping: parseJsonObject<Record<string, string>>(form.modelMappingText, 'Model mapping'),
        modelCapabilityRows: modelCapabilityRows.value,
        reasoningMapping: parseJsonObject<Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>>(form.reasoningMappingText, 'Reasoning mapping'),
        reasoningParamStyle: form.reasoningParamStyle,
        textVerbosity: form.textVerbosity,
        fastMode: form.fastMode,
        customHeaders: parseJsonObject<Record<string, string>>(form.customHeadersText, 'Custom headers'),
        proxyUrl: form.proxyUrl,
        requestTimeoutMs: form.requestTimeoutMs,
        responseHeaderTimeoutMs: form.responseHeaderTimeoutMs,
        streamFirstContentTimeoutMs: form.streamFirstContentTimeoutEnabled ? form.streamFirstContentTimeoutMs : undefined,
        streamInactivityTimeoutMs: form.streamInactivityTimeoutEnabled ? form.streamInactivityTimeoutMs : undefined,
        streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutEnabled ? form.streamToolCallIdleTimeoutMs : undefined,
        rateLimitWindowMinutes: form.rateLimitWindowMinutes,
        routePrefix: form.routePrefix,
        supportedModels: normalizedSupportedModelState.value.validPatterns,
        autoBlacklistBalance: form.autoBlacklistBalance,
        normalizeMetadataUserId: form.normalizeMetadataUserId,
        stripBillingHeader: form.stripBillingHeader,
        stripEmptyTextBlocks: form.stripEmptyTextBlocks,
        normalizeSystemRoleToTopLevel: form.normalizeSystemRoleToTopLevel,
        codexNativeToolPassthrough: form.codexNativeToolPassthrough,
        codexToolCompat: form.codexToolCompat,
        normalizeNonstandardChatRoles: form.normalizeNonstandardChatRoles,
        stripCodexClientTools: form.stripCodexClientTools,
        stripImageGenerationTool: form.stripImageGenerationTool,
        noVision: form.noVision,
        noVisionModels: getNoVisionModelsFromRows(),
        visionFallbackModel: form.visionFallbackModel,
        historicalImageTurnLimit: form.historicalImageTurnLimit,
      })

  applyVisionFallbackReasoning(payload)

  if (isEditMode.value && props.channel?.requestTimeoutMs && !String(form.requestTimeoutMs ?? '').trim()) {
    payload.requestTimeoutMs = 0
  }
  if (isEditMode.value && props.channel?.responseHeaderTimeoutMs && !String(form.responseHeaderTimeoutMs ?? '').trim()) {
    payload.responseHeaderTimeoutMs = 0
  }
  if (isEditMode.value && props.channel?.streamFirstContentTimeoutMs && !form.streamFirstContentTimeoutEnabled) {
    payload.streamFirstContentTimeoutMs = 0
  }
  if (isEditMode.value && props.channel?.streamInactivityTimeoutMs && !form.streamInactivityTimeoutEnabled) {
    payload.streamInactivityTimeoutMs = 0
  }
  if (isEditMode.value && props.channel?.streamToolCallIdleTimeoutMs && !form.streamToolCallIdleTimeoutEnabled) {
    payload.streamToolCallIdleTimeoutMs = 0
  }
  if (isEditMode.value && props.channel?.rateLimitRpm && !payload.rateLimitRpm) {
    payload.rateLimitRpm = 0
  }
  if (isEditMode.value && props.channel?.rateLimitMaxConcurrent && !payload.rateLimitMaxConcurrent) {
    payload.rateLimitMaxConcurrent = 0
  }

  return payload
}

async function persistCurrentDraft(options: { notifyParent?: boolean; close?: boolean } = {}) {
  syncModelCapabilitiesFromMapping()

  if (!isValid.value) {
    error.value = Object.values(errors.value)[0] || ''
    return false
  }

  saving.value = true
  error.value = ''
  success.value = ''
  try {
    await saveChannel(buildSubmitPayload(), props.channel?.index ?? null, {
      isQuickAdd: !isEditMode.value,
    }, props.channelType)
    if (options.notifyParent) emit('saved')
    if (options.close) emit('close')
    return true
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
    return false
  } finally {
    saving.value = false
  }
}

async function handleSubmit() {
  await persistCurrentDraft({ notifyParent: true, close: true })
}

// Keyboard shortcuts: Esc 取消，Cmd/Ctrl+Enter 保存（编辑/创建一致，避免多行文本内 Enter 误触发）
const handleGlobalKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    if (showTargetSuggestions.value) {
      e.preventDefault()
      e.stopPropagation()
      hideTargetDropdown()
      return
    }
    e.preventDefault()
    emit('close')
    return
  }

  if (e.key !== 'Enter') return
  if (saving.value) return

  // 统一 Cmd/Ctrl+Enter 保存（textarea 内也生效），普通 Enter 保留换行/原生行为
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey) {
    e.preventDefault()
    void handleSubmit()
  }
}

// 组件挂载即注册快捷键（新建和编辑模式都需要）
onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
  window.addEventListener('pointerdown', handlePointerDown)

  // 按滚动位置同步左侧导航高亮；长 section 内滚动也需要实时更新
  // 使用多次 nextTick + setTimeout 确保 Teleport + reka-ui 完全渲染
  nextTick(() => {
    nextTick(() => {
      setTimeout(() => {
        // 优先从 dialogRef 查找，失败则全局查找（Teleport 可能导致 ref 为 null）
        let viewport = dialogRef.value?.querySelector('[data-slot="scroll-area-viewport"]') as Element | null
        if (!viewport) {
          console.warn('[ChannelEditDialog] dialogRef 查询失败，尝试全局查询')
          // Teleport to body 后，需要从 document 查找；但可能有多个对话框，取最后一个（最新打开的）
          const all = document.querySelectorAll('[data-slot="scroll-area-viewport"]')
          viewport = all.length > 0 ? all[all.length - 1] : null
        }

        if (!viewport) {
          console.error('[ChannelEditDialog] 未找到滚动容器')
          return
        }

        scrollRoot = viewport
        console.log('[ChannelEditDialog] 滚动容器已绑定', scrollRoot)
        scrollHandler = () => updateActiveSectionFromScroll()
        scrollRoot.addEventListener('scroll', scrollHandler, { passive: true })
        updateActiveSectionFromScroll()
      }, 200)
    })
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
  window.removeEventListener('pointerdown', handlePointerDown)
  if (scrollRoot && scrollHandler) {
    scrollRoot.removeEventListener('scroll', scrollHandler)
  }
  clearDuplicateKeyHighlight()
  clearCopilotPollTimer()
  clearCopilotCopyTimer()
  scrollRoot = null
  scrollHandler = null
})

// ── API Key 操作 ──

function findDuplicateKeyIndex(newKey: string): number {
  return existingApiKeys.value.findIndex(k => k === newKey)
}

function clearDuplicateKeyHighlight() {
  if (duplicateKeyTimer) {
    clearTimeout(duplicateKeyTimer)
    duplicateKeyTimer = null
  }
  duplicateKeyIndex.value = null
}

function setDuplicateKeyHighlight(index: number) {
  clearDuplicateKeyHighlight()
  duplicateKeyIndex.value = index
  duplicateKeyTimer = setTimeout(() => {
    duplicateKeyIndex.value = null
    duplicateKeyTimer = null
  }, 3000)
}

async function addNewApiKeys() {
  const lines = parseLines(newApiKeysText.value)
  if (lines.length === 0) return

  // 去重粘贴内容内部的重复
  const uniqueLines = [...new Set(lines)]

  // 预检全部 key：检查与 existingApiKeys 的重复（用去重后的列表）
  for (const k of uniqueLines) {
    const duplicateIndex = findDuplicateKeyIndex(k)
    if (duplicateIndex !== -1) {
      error.value = t('addChannel.duplicateKey')
      setDuplicateKeyHighlight(duplicateIndex)
      return
    }
  }
  // 无重复，批量添加（去重后）
  for (const k of uniqueLines) {
    existingApiKeys.value.push(k)
  }
  clearDuplicateKeyHighlight()
  error.value = ''
  newApiKeysText.value = ''
}

async function copyApiKey(key: string, index: number) {
  try {
    await navigator.clipboard.writeText(key)
    copiedKeyIndex.value = index
    setTimeout(() => { copiedKeyIndex.value = null }, 1200)
  } catch {
    // clipboard 不可用时静默
  }
}

function moveApiKeyToTop(index: number) {
  if (index <= 0 || index >= existingApiKeys.value.length) return
  const [key] = existingApiKeys.value.splice(index, 1)
  existingApiKeys.value.unshift(key)
}

function moveApiKeyToBottom(index: number) {
  if (index < 0 || index >= existingApiKeys.value.length - 1) return
  const [key] = existingApiKeys.value.splice(index, 1)
  existingApiKeys.value.push(key)
}

const visibleDisabledKeys = computed(() => {
  if (!isEditMode.value) return []
  return disabledApiKeys.value.filter(dk => !localRestoredKeys.value.has(dk.key))
})

const hasDisabledKeys = computed(() => visibleDisabledKeys.value.length > 0)

async function handleDisabledKeyRestore(key: string) {
  if (!props.channel) return
  restoringKey.value = key
  error.value = ''
  try {
    await restoreApiKey(props.channel.index, key, props.channelType)
    localRestoredKeys.value.add(key)
    existingApiKeys.value.push(key)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    restoringKey.value = ''
  }
}

// ── Model Mapping 行操作 ──

function modelMappingFromChannel(ch: Channel) {
  const mapping = ch.modelMapping || {}
  const reasoning = ch.reasoningMapping || {}
  const noVision = new Set(ch.noVisionModels || [])
  return Object.entries(mapping).map(([source, target]) => ({
    id: ++rowId,
    source,
    target,
    reasoning: (reasoning[source] || '') as ModelMappingRow['reasoning'],
    noVision: noVision.has(target),
  }))
}

function addModelMappingRow() {
  if (!newModelMapping.source.trim() || !newModelMapping.target.trim() || sourceMappingError.value) return
  const target = newModelMapping.target.trim()
  modelMappingRows.value.push({
    id: ++rowId,
    source: newModelMapping.source.trim(),
    target,
    reasoning: newModelMapping.reasoning || '',
    noVision: findNoVisionForTarget(target) ?? newModelMapping.noVision,
  })
  newModelMapping.source = ''
  newModelMapping.target = ''
  newModelMapping.reasoning = ''
  newModelMapping.noVision = false
  finishMappingTargetEdit()
}

function removeModelMappingRow(id: number) {
  const index = modelMappingRows.value.findIndex(row => row.id === id)
  if (index >= 0) modelMappingRows.value.splice(index, 1)
}

// ── 预设模板 ──

const modelMappingPresets = openaiMessagesPresets
const claudeChannelPresets = claudeMessagesPresets

const serviceTypeOptions = computed(() => {
  const all = [
    { label: 'OpenAI Chat', value: 'openai' },
    { label: 'Claude', value: 'claude' },
    { label: 'Gemini', value: 'gemini' },
    { label: 'Responses (Codex)', value: 'responses' },
    { label: 'GitHub Copilot', value: 'copilot' },
  ]
  const first = props.channelType === 'messages' ? 'claude'
    : props.channelType === 'responses' ? 'responses'
    : props.channelType === 'gemini' ? 'gemini'
    : 'openai'
  if (props.channelType === 'images') return [{ label: 'OpenAI Images', value: 'openai' }]
  const primary = all.find(o => o.value === first)
  const rest = all.filter(o => o.value !== first)
  return primary ? [primary, ...rest] : all
})

// 推荐的上游类型：未手动选择时 = 已识别类型，否则回退默认类型；手动选择后不再推荐
const recommendedServiceType = computed<string | null>(() => {
  if (quickServiceTypeTouched.value) return null
  return detectedServiceType.value || defaultServiceTypeForChannel()
})

// 头部选择器选项：给推荐项的标签追加「· 推荐」后缀
const headerServiceTypeItems = computed(() => {
  const suffix = t('addChannel.serviceTypeRecommendedSuffix')
  return serviceTypeOptions.value.map(option =>
    option.value === recommendedServiceType.value
      ? { ...option, label: `${option.label}${suffix}` }
      : option,
  )
})

const supportsOpenAIAdvanced = computed(() => supportsAdvancedChannelOptions(form.serviceType))
const supportsOpenAIAdvancedOptions = computed(() => supportsAdvancedChannelOptions(form.serviceType))
const supportsReasoningMappingOptions = computed(() => supportsReasoningMapping(form.serviceType))
const supportsChatRoleNormalization = computed(() => {
  return props.channelType === 'chat' || (props.channelType === 'responses' && form.serviceType === 'openai')
})
const modelMappingHint = computed(() => {
  if (props.channelType === 'chat' || props.channelType === 'images') {
    return t('addChannel.modelMappingHintChat')
  }
  if (props.channelType === 'gemini') {
    return t('addChannel.modelMappingHintGemini')
  }
  if (props.channelType === 'responses') {
    return t('addChannel.modelMappingHintResponses')
  }
  return t('addChannel.modelMappingHintMessages')
})
const targetModelPlaceholder = computed(() => {
  if (props.channelType === 'chat' || props.channelType === 'images') {
    return t('addChannel.targetModelPlaceholderChat')
  }
  if (props.channelType === 'responses') {
    return t('addChannel.targetModelPlaceholderResponses')
  }
  if (props.channelType === 'gemini') {
    return t('addChannel.targetModelPlaceholderGemini')
  }
  return t('addChannel.targetModelPlaceholderMessages')
})
const showModelMappingPresets = computed(() => props.channelType === 'messages' && supportsOpenAIAdvanced.value)
const showMessagesOpenAIChannelPresets = computed(() => props.channelType === 'messages' && supportsOpenAIAdvanced.value)
const showClaudeChannelPresets = computed(() => form.serviceType === 'claude' && ['messages', 'chat', 'responses'].includes(props.channelType))
const showCodexResponsesPresets = computed(() => props.channelType === 'responses' && supportsOpenAIAdvanced.value)

function applyModelMappingPreset(name: string) {
  const preset = modelMappingPresets[name.toLowerCase()]
  if (!preset) return
  // 仅 OpenAI/Responses 上游应用 reasoning 映射（对齐 WebUI）
  const applyReasoning = supportsOpenAIAdvanced.value
  modelMappingRows.value = Object.entries(preset.modelMapping).map(([source, target]) => ({
    id: ++rowId,
    source,
    target,
    reasoning: applyReasoning ? (preset.reasoningMapping[source] || '') : '',
    noVision: false,
  }))
  form.fastMode = preset.fastMode
  form.textVerbosity = preset.textVerbosity as typeof form.textVerbosity
}

function applyClaudePreset(name: string) {
  const preset = claudeChannelPresets[name.toLowerCase()]
  if (!preset) return
  const noVisionSet = new Set(preset.noVisionModels)
  modelMappingRows.value = Object.entries(preset.modelMapping).map(([source, target]) => ({
    id: ++rowId,
    source,
    target,
    reasoning: preset.reasoningMapping[source] || '',
    noVision: noVisionSet.has(target),
  }))
  form.reasoningParamStyle = preset.reasoningParamStyle as typeof form.reasoningParamStyle
  form.passbackReasoningContent = preset.passbackReasoningContent
  form.passbackThinkingBlocks = preset.passbackThinkingBlocks
  form.stripEmptyTextBlocks = preset.stripEmptyTextBlocks
  form.normalizeSystemRoleToTopLevel = preset.normalizeSystemRoleToTopLevel
  form.stripImageGenerationTool = preset.stripImageGenerationTool
  form.noVision = preset.noVision
  form.visionFallbackModel = preset.visionFallbackModel
  form.visionFallbackReasoningEffort = ''
}

function applyCodexResponsesPreset(name: string) {
  const preset = codexResponsesPresets[name.toLowerCase()]
  if (!preset) return
  const noVisionSet = new Set(preset.noVisionModels)
  modelMappingRows.value = Object.entries(preset.modelMapping).map(([source, target]) => ({
    id: ++rowId,
    source,
    target,
    reasoning: preset.reasoningMapping[source] || '',
    noVision: noVisionSet.has(target),
  }))
  form.reasoningParamStyle = preset.reasoningParamStyle as typeof form.reasoningParamStyle
  if (preset.serviceType) {
    form.serviceType = preset.serviceType as typeof form.serviceType
  }
  form.codexNativeToolPassthrough = preset.codexNativeToolPassthrough
  form.codexToolCompat = preset.codexToolCompat
  form.stripCodexClientTools = preset.stripCodexClientTools
  form.stripImageGenerationTool = preset.stripImageGenerationTool
  form.normalizeNonstandardChatRoles = preset.normalizeNonstandardChatRoles
  form.noVision = preset.noVision
  form.visionFallbackModel = preset.visionFallbackModel
  form.visionFallbackReasoningEffort = ''
}

// ── 模型列表拉取 ──

const fetchingModels = ref(false)

// 切换渠道时重置拉取状态（独立于 resetForm，避免与早于本 ref 定义的 props.channel watch 产生 TDZ）
watch(() => props.channel?.index, () => {
  targetModelOptions.value = []
  fetchedModelsError.value = ''
  hasTriedFetchModels.value = false
})

// ── Source 模型预置列表（对齐 WebUI allSourceModelOptions） ──
const sourceModelPresetOptions = computed(() => {
  if (props.channelType === 'chat') {
    return ['codex', 'gpt', 'mini', 'gpt-5', 'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini']
  }
  if (props.channelType === 'images') {
    return ['gpt-image-2', 'gpt-image-1', 'dall-e-3', 'dall-e-2']
  }
  if (props.channelType === 'gemini') {
    return ['gemini-3.5-flash', 'gemini-3.1-pro-preview', 'gemini-3-pro-preview', 'gemini-3-flash-preview', 'gemini-3.1-flash-lite', 'gemini-2.5-pro', 'gemini-2.5-flash', 'gemini-2.5-flash-lite', 'gemini-2']
  }
  if (props.channelType === 'responses') {
    return ['codex', 'codex-auto-review', 'gpt-5', 'gpt', 'mini', 'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini']
  }
  // messages (Claude)
  return ['fable', 'opus', 'sonnet', 'haiku']
})

// 可选源模型：过滤掉已配置重定向的源模型，保持与 Web 端一致。
const sourceModelOptions = computed(() => {
  const configuredSources = new Set(modelMappingRows.value.map(row => row.source))
  return sourceModelPresetOptions.value.filter(model => !configuredSources.has(model))
})

const targetModelDatalist = computed(() => {
  const byLowercaseModel = new Map<string, string>()
  for (const model of targetModelOptions.value) {
    const trimmed = String(model || '').trim()
    if (!trimmed) continue
    const key = trimmed.toLowerCase()
    const existing = byLowercaseModel.get(key)
    if (!existing || trimmed === key) {
      byLowercaseModel.set(key, trimmed)
    }
  }
  return sortModelNamesDesc(Array.from(byLowercaseModel.values()))
})

const commonSupportedModelFilters = ['claude-*', 'gpt-5*', 'gpt-image-2', 'grok-4*', 'gemini-3*', '!*image*']
const normalizedSupportedModelState = computed(() => {
  const parsedPatterns = parseSupportedModelInput(form.supportedModelsText)
  return filterValidSupportedModelPatterns(parsedPatterns)
})
const supportedModelsError = computed(() => (
  normalizedSupportedModelState.value.hasInvalidPatterns
    ? t('addChannel.supportedModelsInvalidPattern')
    : ''
))
const selectedSupportedModelSet = computed(() => new Set(normalizedSupportedModelState.value.validPatterns))

const isPresetSourceModel = (value: string): boolean => sourceModelPresetOptions.value.includes(value)

const validateSourceModelName = (value: string): string => {
  const source = value.trim()
  if (!source) return ''
  if (!isPresetSourceModel(source) && source.length > 50) return t('addChannel.sourceModelNameTooLong')
  if (/\s/.test(source)) return t('addChannel.sourceModelNoSpaces')
  if (!/^[\w.\-/:@+]+$/.test(source)) return t('addChannel.sourceModelInvalidChars')
  return ''
}

const sourceMappingError = computed(() => {
  const source = newModelMapping.source.trim()
  if (!source) return ''
  const sourceNameError = validateSourceModelName(source)
  if (sourceNameError) return sourceNameError
  return modelMappingRows.value.some(row => row.source === source)
    ? t('channelEditor.mapping.source.duplicate')
    : ''
})

function toggleSupportedModelFilter(filter: string) {
  const current = [...normalizedSupportedModelState.value.validPatterns]
  const idx = current.indexOf(filter)
  if (idx !== -1) {
    current.splice(idx, 1)
  } else {
    current.push(filter)
  }
  form.supportedModelsText = current.join('\n')
}

// ── Base URL 预期请求预览 ──
const expectedRequestUrls = computed(() => {
  return buildExpectedRequestUrls(
    props.channelType as any,
    form.serviceType as any,
    form.baseUrl,
    parseLines(form.baseUrlsText),
  )
})

// 快速添加模式：基于 detectedBaseUrls 计算预期请求预览
const quickExpectedRequestUrls = computed(() => {
  return buildExpectedRequestUrls(
    props.channelType as any,
    (form.serviceType || detectedServiceType.value) as any,
    detectedBaseUrls.value[0] || '',
    detectedBaseUrls.value,
  )
})

async function fetchTargetModels() {
  console.log('[fetchTargetModels] 开始执行', { 
    hasChannel: !!props.channel, 
    baseUrl: form.baseUrl, 
    apiKeysCount: getSubmitApiKeys().length 
  })
  
  if (!props.channel) {
    console.log('[fetchTargetModels] 中断：无渠道')
    return
  }
  if (!form.baseUrl.trim() || getSubmitApiKeys().length === 0) {
    console.log('[fetchTargetModels] 中断：缺少配置')
    fetchedModelsError.value = t('addChannel.fillBaseUrlAndApiKey')
    return
  }

  const keys = getSubmitApiKeys()
  const uncheckedKeys = keys.filter(key => !keyModelsStatus.value.has(key))
  if (uncheckedKeys.length === 0) {
    showTargetSuggestions.value = !!activeTargetInputId.value && targetModelDatalist.value.length > 0
    return
  }

  fetchingModels.value = true
  fetchedModelsError.value = ''
  try {
    console.log('[fetchTargetModels] 开始请求 API')
    // 直接调用 API，不依赖于 persistCurrentDraft（避免表单验证失败导致无法获取模型）
    const effectiveServiceType = props.channelType === 'images'
      ? 'openai'
      : (form.serviceType || defaultServiceTypeForChannel())
    let modelsApiType: ManagedChannelType
    if (props.channelType === 'images') {
      modelsApiType = 'images'
    } else if (effectiveServiceType === 'gemini') {
      modelsApiType = 'gemini'
    } else if (effectiveServiceType === 'responses') {
      modelsApiType = 'responses'
    } else if (effectiveServiceType === 'openai') {
      modelsApiType = 'chat'
    } else {
      modelsApiType = 'messages'
    }

    const typeApi = getChannelTypeApi(modelsApiType)
    const channelId = props.channel.index
    const customHeaders = getHeadersAsObject()
    const results = await Promise.all(uncheckedKeys.map(async (key) => {
      keyModelsStatus.value.set(key, { loading: true, success: false })
      try {
        const resp = await typeApi.getChannelModels(channelId, {
          key,
          baseUrl: form.baseUrl,
          proxyUrl: form.proxyUrl,
          insecureSkipVerify: form.insecureSkipVerify,
          customHeaders: Object.keys(customHeaders).length ? customHeaders : undefined,
          authHeader: form.authHeader && form.authHeader !== 'auto' ? form.authHeader : undefined,
        })
        const list: any[] = Array.isArray(resp) ? resp : (resp?.data ?? [])
        keyModelsStatus.value.set(key, {
          loading: false,
          success: true,
          statusCode: 200,
          modelCount: list.length,
        })
        return list
      } catch (e) {
        keyModelsStatus.value.set(key, {
          loading: false,
          success: false,
          statusCode: e instanceof AdminApiError ? e.status : 'ERR',
          error: e instanceof Error ? e.message : String(e),
        })
        return []
      }
    }))
    const byLowercaseModel = new Map<string, string>()
    results
      .flat()
      .map((m: any) => m.id || m.name || String(m))
      .filter(Boolean)
      .forEach(model => {
        const trimmed = String(model).trim()
        if (!trimmed) return
        const key = trimmed.toLowerCase()
        const existing = byLowercaseModel.get(key)
        if (!existing || trimmed === key) {
          byLowercaseModel.set(key, trimmed)
        }
      })
    targetModelOptions.value = sortModelNamesDesc(Array.from(byLowercaseModel.values()))
    showTargetSuggestions.value = !!activeTargetInputId.value && targetModelDatalist.value.length > 0

    const allFailed = keys.every(key => {
      const status = keyModelsStatus.value.get(key)
      return status && !status.success
    })
    if (allFailed) {
      fetchedModelsError.value = t('addChannel.allApiKeysModelsFailed')
    }
    console.log('[fetchTargetModels] 成功获取模型', targetModelOptions.value.length)
  } catch (e) {
    console.error('[fetchTargetModels] 请求失败', e)
    fetchedModelsError.value = e instanceof Error 
      ? e.message 
      : typeof e === 'object' && e !== null 
        ? JSON.stringify(e, null, 2) 
        : String(e)
  } finally {
    fetchingModels.value = false
  }
}

// target 框首次聚焦时自动拉取真实模型（配置齐全时自动触发，新增/编辑均可）
function handleTargetFocus() {
  if (hasTriedFetchModels.value || fetchingModels.value) return
  if (!form.baseUrl.trim() || getSubmitApiKeys().length === 0) return
  hasTriedFetchModels.value = true
  void fetchTargetModels()
}

function getModelMappingAsObject(): Record<string, string> {
  const result: Record<string, string> = {}
  for (const row of modelMappingRows.value) {
    if (row.source && row.target) result[row.source] = row.target
  }
  return result
}

function getReasoningMappingAsObject(): Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'> {
  const result: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'> = {}
  for (const row of modelMappingRows.value) {
    if (row.source && row.target && row.reasoning) {
      result[row.source] = row.reasoning as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
    }
  }
  return result
}

function applyVisionFallbackReasoning(payload: Partial<Channel>) {
  const fallbackModel = form.visionFallbackModel.trim()
  if (!supportsReasoningMappingOptions.value || !fallbackModel) {
    return
  }

  const reasoningMapping = { ...(payload.reasoningMapping || {}) }
  if (form.visionFallbackReasoningEffort) {
    reasoningMapping[fallbackModel] = form.visionFallbackReasoningEffort
  } else if (!modelMappingRows.value.some(row => row.source === fallbackModel && row.reasoning)) {
    delete reasoningMapping[fallbackModel]
  }
  payload.reasoningMapping = reasoningMapping
}

function getNoVisionModelsFromRows(): string[] {
  return [...new Set(
    modelMappingRows.value
      .filter(row => row.noVision && row.target.trim())
      .map(row => row.target.trim())
  )]
}

function normalizeTargetKey(target: string): string {
  return target.trim()
}

function findNoVisionForTarget(target: string): boolean | undefined {
  const targetKey = normalizeTargetKey(target)
  const matched = modelMappingRows.value.find(row => normalizeTargetKey(row.target) === targetKey)
  return matched?.noVision
}

function setNoVisionForTarget(target: string, noVision: boolean) {
  const targetKey = normalizeTargetKey(target)
  if (!targetKey) return
  modelMappingRows.value.forEach(row => {
    if (normalizeTargetKey(row.target) === targetKey) {
      row.noVision = noVision
    }
  })
}

function applyPreset(presetName: string) {
  if (presetName === 'gpt-5.5' || presetName === 'gpt-5.4') {
    applyModelMappingPreset(presetName)
  } else if (form.serviceType === 'claude') {
    applyClaudePreset(presetName)
  } else if (props.channelType === 'responses') {
    applyCodexResponsesPreset(presetName)
  } else if (props.channelType === 'messages' || props.channelType === 'chat') {
    applyClaudePreset(presetName)
  }
}

function syncUpstreamModels() {
  // 同步上游模型列表
  void fetchTargetModels()
}

function updateMappingRow(id: number, field: keyof ModelMappingRow, value: any) {
  const row = modelMappingRows.value.find(r => r.id === id)
  if (!row) return

  // noVision 按 target 模型名聚合（后端 noVisionModels 是 string[]），
  // 切换一行时需同步所有相同 target 的行，保持视觉状态一致
  if (field === 'noVision') {
    setNoVisionForTarget(row.target, value)
  } else if (field === 'target') {
    const target = String(value).trim()
    const existingNoVision = findNoVisionForTarget(target)
    row.target = target
    row.noVision = existingNoVision ?? row.noVision
    setNoVisionForTarget(target, row.noVision)
  } else {
    ;(row as any)[field] = value
  }
}

function updateModelCapabilityRows(rows: ModelCapabilityRow[]) {
  modelCapabilityRows.value = rows
  form.modelCapabilityRows = rows
}

function syncModelCapabilitiesFromMapping() {
  const existingModels = new Set(
    modelCapabilityRows.value
      .map(row => row.model.trim())
      .map(model => model.toLowerCase())
      .filter(Boolean)
  )
  const rowsToAdd = mappedTargetModels.value
    .filter(isCompleteMappedTargetModel)
    .filter(model => !existingModels.has(model.toLowerCase()))
    .map(model => {
      const builtin = resolveBuiltinUpstreamModelCapability(model)
      return createModelCapabilityRow(
        ++rowId,
        model,
        builtin?.capability,
        builtin ? 'builtin' : 'custom',
        builtin?.pattern || '',
      )
    })
  if (!rowsToAdd.length) return
  updateModelCapabilityRows([...modelCapabilityRows.value, ...rowsToAdd])
}

function syncModelCapabilitiesFromMappingWhenIdle() {
  if (isMappingTargetEditing.value) {
    hasPendingModelCapabilitySync.value = true
    return
  }
  hasPendingModelCapabilitySync.value = false
  syncModelCapabilitiesFromMapping()
}

function startMappingTargetEdit() {
  isMappingTargetEditing.value = true
}

function finishMappingTargetEdit() {
  if (!isMappingTargetEditing.value) return
  isMappingTargetEditing.value = false
  if (!hasPendingModelCapabilitySync.value) return
  hasPendingModelCapabilitySync.value = false
  nextTick(syncModelCapabilitiesFromMapping)
}

watch(mappedTargetModels, () => {
  syncModelCapabilitiesFromMappingWhenIdle()
})

// ── Custom Headers 行操作 ──

function headerRowsFromChannel(ch: Channel) {
  const headers = ch.customHeaders || {}
  return Object.entries(headers).map(([k, v]) => ({ id: ++rowId, key: k, value: v }))
}

function addHeaderRow() {
  if (!newHeader.key.trim()) return
  headerRows.value.push({ id: ++rowId, key: newHeader.key.trim(), value: newHeader.value })
  newHeader.key = ''
  newHeader.value = ''
}

function removeHeaderRow(id: number) {
  headerRows.value = headerRows.value.filter(row => row.id !== id)
}

function updateHeaderRow(id: number, field: 'key' | 'value', value: string) {
  const row = headerRows.value.find(r => r.id === id)
  if (row) row[field] = value
}

function getHeadersAsObject(): Record<string, string> {
  const result: Record<string, string> = {}
  for (const h of headerRows.value) {
    if (h.key.trim()) result[h.key.trim()] = h.value
  }
  return result
}

// ── 编辑头部动作：noVision toggle + Test Capability ──

async function handleTestCapability() {
  if (!props.channel) return

  // 父组件收到 test-capability 后负责关闭编辑弹窗并刷新；这里不能先 emit saved，避免组件卸载后丢失事件。
  const saved = await persistCurrentDraft()
  if (!saved) return

  emit('test-capability', {
    ...props.channel,
    name: form.name || props.channel.name,
    index: props.channel.index,
  })
}

async function handleDiagnoseCompat() {
  if (!props.channel || props.channelType === 'images') return
  diagnosingCompat.value = true
  error.value = ''
  success.value = ''
  try {
    const result = await adminApi.post<CompatDiagnoseResult>(`/api/${props.channelType}/channels/${props.channel.index}/compat-diagnose`, {})
    const applied: string[] = []
    for (const [key, val] of Object.entries(result.recommendations)) {
      if (val !== undefined && (form as Record<string, unknown>)[key] !== val) {
        Object.assign(form, { [key]: val })
        applied.push(key)
      }
    }
    if (result.urlRecommendations?.recommended) {
      const current = result.urlRecommendations.current
      const recommended = result.urlRecommendations.recommended
      const lines = form.baseUrlsText.split('\n').map(line => line.trim()).filter(Boolean)
      const nextLines = lines.length > 0
        ? lines.map((line, index) => (index === 0 || line === current) ? recommended : line)
        : [recommended]
      form.baseUrlsText = Array.from(new Set(nextLines)).join('\n')
      applied.push('baseUrl')
    }
    success.value = applied.length
      ? t('channelEditor.compat.diagnoseApplied', { count: String(applied.length) })
      : t('channelEditor.compat.diagnoseNoChange')
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('channelEditor.compat.diagnoseFailed')
  } finally {
    diagnosingCompat.value = false
  }
}

function buildCurrentPayload() {
  const modelMapping = getModelMappingAsObject()
  const reasoningMapping = getReasoningMappingAsObject() as Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>

  return buildChannelPayload({
    name: form.name,
    serviceType: form.serviceType,
    authHeader: form.authHeader,
    baseUrl: form.baseUrl,
    baseUrls: parseLines(form.baseUrlsText),
    website: form.website,
    insecureSkipVerify: form.insecureSkipVerify,
    lowQuality: form.lowQuality,
    injectDummyThoughtSignature: form.injectDummyThoughtSignature,
    stripThoughtSignature: form.stripThoughtSignature,
    passbackReasoningContent: form.passbackReasoningContent,
    passbackThinkingBlocks: form.passbackThinkingBlocks,
    description: form.description,
    apiKeys: getSubmitApiKeys(),
    modelMapping,
    modelCapabilityRows: modelCapabilityRows.value,
    reasoningMapping,
    reasoningParamStyle: form.reasoningParamStyle,
    textVerbosity: form.textVerbosity,
    fastMode: form.fastMode,
    customHeaders: getHeadersAsObject(),
    proxyUrl: form.proxyUrl,
    requestTimeoutMs: form.requestTimeoutMs,
    responseHeaderTimeoutMs: form.responseHeaderTimeoutMs,
    streamFirstContentTimeoutMs: form.streamFirstContentTimeoutEnabled ? form.streamFirstContentTimeoutMs : undefined,
    streamInactivityTimeoutMs: form.streamInactivityTimeoutEnabled ? form.streamInactivityTimeoutMs : undefined,
    streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutEnabled ? form.streamToolCallIdleTimeoutMs : undefined,
    rateLimitRpm: form.rateLimitRpm,
    rateLimitWindowMinutes: form.rateLimitWindowMinutes,
    rateLimitMaxConcurrent: form.rateLimitMaxConcurrent,
    rateLimitAutoFromHeaders: form.rateLimitAutoFromHeaders,
    routePrefix: form.routePrefix,
    supportedModels: normalizedSupportedModelState.value.validPatterns,
    autoBlacklistBalance: form.autoBlacklistBalance,
    normalizeMetadataUserId: form.normalizeMetadataUserId,
    stripBillingHeader: form.stripBillingHeader,
    stripEmptyTextBlocks: form.stripEmptyTextBlocks,
    normalizeSystemRoleToTopLevel: form.normalizeSystemRoleToTopLevel,
    codexNativeToolPassthrough: form.codexNativeToolPassthrough,
    codexToolCompat: form.codexToolCompat,
    normalizeNonstandardChatRoles: form.normalizeNonstandardChatRoles,
    stripCodexClientTools: form.stripCodexClientTools,
    stripImageGenerationTool: form.stripImageGenerationTool,
    noVision: form.noVision,
    noVisionModels: getNoVisionModelsFromRows(),
    visionFallbackModel: form.visionFallbackModel,
    historicalImageTurnLimit: form.historicalImageTurnLimit,
  })
}

// 保留这些函数以备未来使用（模板迁移后的临时死代码）
void hasDisabledKeys
void hasTriedFetchModels
void showModelMappingPresets
void showClaudeChannelPresets
void showCodexResponsesPresets
void applyModelMappingPreset
void applyClaudePreset
void applyCodexResponsesPreset
void sourceModelOptions
void commonSupportedModelFilters
void selectedSupportedModelSet
void toggleSupportedModelFilter
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="true"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('close')" />

        <div
          ref="dialogRef"
          class="relative z-10 flex max-h-[90vh] w-[94vw] flex-col overflow-hidden rounded-xl border border-border/80 bg-background shadow-2xl backdrop-blur-md"
          :class="isEditMode ? 'max-w-6xl' : 'max-w-3xl'"
        >
          <ChannelEditorHeader
            :channel-type="channelType"
            :is-edit-mode="isEditMode"
            :no-vision="form.noVision"
            :saving="saving"
            @close="emit('close')"
            @toggle-no-vision="form.noVision = !form.noVision"
            @test-capability="handleTestCapability"
          />

          <!-- 创建模式：独立快速添加，不展示编辑器大纲和高级配置 -->
          <div v-if="!isEditMode" class="min-h-0 flex-1 overflow-hidden">
            <ScrollArea class="h-full">
              <form @submit.prevent="handleSubmit">
                <div v-if="error" class="mx-6 mt-6 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                  {{ error }}
                </div>

                <QuickCreatePanel
                  :quick-input="quickInput"
                  :service-type="form.serviceType"
                  :service-type-options="headerServiceTypeItems"
                  :detected-base-urls="detectedBaseUrls"
                  :detected-api-keys="detectedApiKeys"
                  :expected-request-urls="quickExpectedRequestUrls"
                  :generated-channel-name="generatedChannelName"
                  @update:quick-input="quickInput = $event"
                  @update:service-type="updateQuickServiceType"
                  @quick-paste="handleQuickPaste"
                />
              </form>
            </ScrollArea>
          </div>

          <!-- 编辑模式：完整渠道编辑器 -->
          <div v-else class="min-h-0 flex-1 flex">
            <!-- 左侧导航 -->
            <nav class="flex w-[180px] shrink-0 flex-col items-stretch gap-1 rounded-none border-r border-border/50 bg-card/20 p-4">
              <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/60 px-3 mb-2">{{ t('channelEditor.nav.outline') }}</div>
              <button
                v-for="s in sections"
                :key="s.id"
                class="flex items-center justify-start rounded-md border px-3 py-1.5 text-xs font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:ring-[3px] focus-visible:outline-1 disabled:pointer-events-none disabled:opacity-50"
                :class="activeSection === s.id
                  ? 'bg-background text-foreground shadow-sm border-input'
                  : 'text-muted-foreground border-transparent hover:text-foreground hover:bg-accent/50'"
                @click="scrollToSection(s.id)"
              >{{ s.label }}</button>
            </nav>

              <!-- 右侧内容面板 -->
              <div class="min-w-0 flex-1 overflow-hidden">
                <ScrollArea class="h-full">
                  <form class="p-6 space-y-6" @submit.prevent="handleSubmit">
                    <!-- 错误提示 -->
                    <div v-if="error" class="border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive rounded-lg">
                      {{ error }}
                    </div>
                    <div v-if="success" class="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-emerald-700 dark:text-emerald-300">
                      {{ success }}
                    </div>

                    <!-- Section: 基础配置 -->
                    <section :ref="(el: any) => setSectionRef('basic', el)" data-section-id="basic" class="scroll-mt-4">
                      <BasicConfigPanel
                        :form="form"
                        :errors="errors"
                        :service-type-options="serviceTypeOptions"
                        :expected-request-urls="expectedRequestUrls"
                        @update:form="(updates) => Object.assign(form, updates)"
                      />
                    </section>

                    <!-- Section: 认证管理 -->
                    <section :ref="(el: any) => setSectionRef('auth', el)" data-section-id="auth" class="scroll-mt-4">
                      <AuthPanel
                        :existing-api-keys="existingApiKeys"
                        :new-api-keys-text="newApiKeysText"
                        :copied-key-index="copiedKeyIndex"
                        :duplicate-key-index="duplicateKeyIndex"
                        :disabled-api-keys="disabledApiKeys"
                        :historical-api-keys="historicalApiKeys"
                        :restoring-key="restoringKey"
                        :local-restored-keys="localRestoredKeys"
                        :key-models-status="keyModelsStatus"
                        :errors="errors"
                        @update:new-api-keys-text="newApiKeysText = $event; clearDuplicateKeyHighlight()"
                        @add-new-api-keys="addNewApiKeys"
                        @remove-existing-api-key="removeExistingApiKey"
                        @move-api-key-to-top="moveApiKeyToTop"
                        @move-api-key-to-bottom="moveApiKeyToBottom"
                        @copy-api-key="copyApiKey"
                        @handle-disabled-key-restore="handleDisabledKeyRestore"
                      />

                      <!-- GitHub Copilot OAuth 登录（仅 copilot 渠道显示） -->
                      <div v-if="form.serviceType === 'copilot'" class="mt-4 rounded-xl border border-border/60 bg-card/40 p-5 space-y-3">
                        <h4 class="text-xs font-bold uppercase tracking-wider text-primary">GitHub Copilot</h4>
                        <div v-if="copilotUserCode" class="flex items-center gap-2 text-sm">
                          <span class="text-muted-foreground">{{ t('copilotOAuth.userCode') }}</span>
                          <code class="px-2 py-0.5 rounded bg-muted font-mono text-xs">{{ copilotUserCode }}</code>
                          <button
                            type="button"
                            class="inline-flex h-6 w-6 items-center justify-center rounded border border-border text-muted-foreground transition-colors hover:text-foreground"
                            :title="copilotUserCodeCopied ? t('common.copied') : t('common.copy')"
                            :aria-label="copilotUserCodeCopied ? t('common.copied') : t('common.copy')"
                            @click="copyCopilotUserCode"
                          >
                            <CheckCircle2 v-if="copilotUserCodeCopied" class="h-3.5 w-3.5 text-emerald-700 dark:text-emerald-400" />
                            <Copy v-else class="h-3.5 w-3.5" />
                          </button>
                          <button type="button" class="text-primary text-xs underline" @click="openCopilotAuthorization">{{ t('copilotOAuth.openAuthorize') }}</button>
                        </div>
                        <p v-if="copilotOAuthSuccess" class="text-xs text-emerald-600">{{ t('copilotOAuth.success') }}</p>
                        <p v-if="copilotOAuthError" class="text-xs text-destructive">{{ copilotOAuthError }}</p>
                        <div class="flex items-center gap-2">
                          <button
                            type="button"
                            class="inline-flex items-center gap-1.5 rounded-md border border-primary/40 bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary hover:bg-primary/20 disabled:opacity-50"
                            :disabled="copilotOAuthLoading || copilotPolling"
                            @click="startCopilotOAuth"
                          >
                            <span v-if="copilotOAuthLoading || copilotPolling" class="animate-spin">⏳</span>
                            {{ t('copilotOAuth.button') }}
                          </button>
                          <span v-if="copilotPolling" class="text-xs text-muted-foreground">{{ t('copilotOAuth.waiting') }}</span>
                          <button v-if="copilotPolling || copilotOAuthLoading" type="button" class="text-xs text-muted-foreground underline" @click="clearCopilotPollTimer(); copilotPolling = false; copilotOAuthLoading = false">{{ t('copilotOAuth.cancel') }}</button>
                        </div>
                      </div>
                    </section>

                    <!-- Section: 模型重定向 -->
                    <section :ref="(el: any) => setSectionRef('redirect', el)" data-section-id="redirect" class="scroll-mt-4">
                      <ModelMappingPanel
                        :model-mapping-rows="modelMappingRows"
                        :new-model-mapping="newModelMapping"
                        :source-model-options="sourceModelOptions"
                        :filtered-source-models="filteredSourceModels"
                        :reasoning-effort-options="reasoningEffortOptions"
                        :filtered-target-models="filteredTargetModels"
                        :channel-type="channelType"
                        :show-target-suggestions="showTargetSuggestions"
                        :active-target-input-id="activeTargetInputId"
                        :show-source-suggestions="showSourceSuggestions"
                        :active-source-input-id="activeSourceInputId"
                        :DEFAULT_SELECT_VALUE="DEFAULT_SELECT_VALUE"
                        :vision-fallback-model="form.visionFallbackModel"
                        :vision-fallback-reasoning-effort="form.visionFallbackReasoningEffort"
                        :supported-models-text="form.supportedModelsText"
                        :model-mapping-hint="modelMappingHint"
                        :target-model-placeholder="targetModelPlaceholder"
                        :show-model-mapping-presets="showModelMappingPresets"
                        :show-messages-open-a-i-channel-presets="showMessagesOpenAIChannelPresets"
                        :show-claude-channel-presets="showClaudeChannelPresets"
                        :show-codex-responses-presets="showCodexResponsesPresets"
                        :supports-reasoning-mapping-options="supportsReasoningMappingOptions"
                        :common-supported-model-filters="commonSupportedModelFilters"
                        :selected-supported-model-set="selectedSupportedModelSet"
                        :source-mapping-error="sourceMappingError"
                        :fetch-models-error="fetchedModelsError"
                        :supported-models-error="supportedModelsError"
                        @update:new-model-mapping="(updates) => Object.assign(newModelMapping, updates)"
                        @update:vision-fallback-model="form.visionFallbackModel = $event"
                        @update:vision-fallback-reasoning-effort="form.visionFallbackReasoningEffort = $event"
                        @update:supported-models-text="form.supportedModelsText = $event"
                        @add-model-mapping-row="addModelMappingRow"
                        @remove-model-mapping-row="removeModelMappingRow"
                        @update-mapping-row="updateMappingRow"
                        @sync-upstream-models="syncUpstreamModels"
                        @apply-preset="applyPreset"
                        @show-target-dropdown="showTargetDropdown"
                        @hide-target-dropdown="hideTargetDropdown"
                        @select-target-model="selectTargetModel"
                        @handle-target-focus="handleTargetFocus"
                        @target-edit-start="startMappingTargetEdit"
                        @target-edit-end="finishMappingTargetEdit"
                        @show-source-dropdown="showSourceDropdown"
                        @hide-source-dropdown="hideSourceDropdown"
                        @select-source-model="selectSourceModel"
                        @append-supported-model-filter="toggleSupportedModelFilter"
                      />
                      <ModelCapabilityPanel
                        v-if="channelType !== 'images'"
                        class="mt-6"
                        :rows="modelCapabilityRows"
                        :target-models="targetModelDatalist"
                        :mapped-target-models="mappedTargetModels"
                        :fetching-models="fetchingModels"
                        :fetch-models-error="fetchedModelsError"
                        :error="errors.modelCapabilitiesText"
                        @update:rows="updateModelCapabilityRows"
                        @sync-upstream-models="syncUpstreamModels"
                      />
                    </section>

                    <!-- Section: 高级选项 -->
                    <section :ref="(el: any) => setSectionRef('advanced', el)" data-section-id="advanced" class="scroll-mt-4">
                      <AdvancedPanel
                        :form="form"
                        :channel-type="channelType"
                        :supports-open-a-i-advanced-options="supportsOpenAIAdvancedOptions"
                        :supports-chat-role-normalization="supportsChatRoleNormalization"
                        :reasoning-param-style-options="reasoningParamStyleOptions"
                        :text-verbosity-options="textVerbosityOptions"
                        :diagnosing="diagnosingCompat"
                        @update:form="(updates) => Object.assign(form, updates)"
                        @diagnose="handleDiagnoseCompat"
                      />
                    </section>

                    <!-- Section: 自定义参数 -->
                    <section :ref="(el: any) => setSectionRef('custom', el)" data-section-id="custom" class="scroll-mt-4">
                      <CustomHeadersPanel
                        :header-rows="headerRows"
                        :new-header="newHeader"
                        @update:new-header="(updates) => Object.assign(newHeader, updates)"
                        @add-header-row="addHeaderRow"
                        @remove-header-row="removeHeaderRow"
                        @update-header-row="updateHeaderRow"
                      />
                      <div class="mt-6">
                        <StreamTimeoutPanel
                          :form="form"
                          @update:form="(updates) => Object.assign(form, updates)"
                        />
                      </div>
                      <div class="mt-6">
                        <CustomParamsPanel
                          :form="form"
                          @update:form="(updates) => Object.assign(form, updates)"
                        />
                      </div>
                    </section>
                  </form>
                </ScrollArea>
              </div>
          </div>

          <!-- 底部按钮栏 -->
          <div class="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-border bg-card/80 p-4 backdrop-blur-md">
            <Button variant="outline" class="hover:bg-muted hover:text-foreground dark:hover:bg-muted/50 hover:scale-[1.02] active:scale-[0.98]" @click="emit('close')">
              {{ t('common.cancel') }}
              <span class="ml-1 hidden sm:inline-flex h-4 select-none items-center gap-1 rounded border bg-transparent px-1.5 font-mono text-[9px] font-medium text-muted-foreground/80">Esc</span>
            </Button>
            <Button type="button" class="hover:shadow-lg hover:scale-[1.02] active:scale-[0.98]" :disabled="!isValid || saving" @click="handleSubmit">
              <Loader2 v-if="saving" class="mr-2 h-4 w-4 animate-spin" />
              {{ isEditMode
                ? t('channelEditor.actions.save')
                : t('channelEditor.actions.create')
              }}
              <span class="ml-1 hidden sm:inline-flex h-4 select-none items-center gap-1 rounded border border-primary-foreground/30 bg-primary-foreground/10 px-1.5 font-mono text-[9px] font-medium text-primary-foreground/90">{{ isMac ? '⌘ Enter' : 'Ctrl+Enter' }}</span>
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Range Slider 美化 */
input[type="range"].accent-primary::-webkit-slider-runnable-track {
  background: hsl(var(--primary) / 0.1);
  height: 4px;
  border-radius: 9999px;
}

input[type="range"].accent-primary::-webkit-slider-thumb {
  margin-top: -5px;
  background: hsl(var(--primary));
  border: 2px solid hsl(var(--background));
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
  width: 14px;
  height: 14px;
  border-radius: 9999px;
  transition: transform 0.1s;
}

input[type="range"].accent-primary::-webkit-slider-thumb:hover {
  transform: scale(1.2);
}
</style>
