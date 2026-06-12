<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  AlertCircle,
  ArrowDown,
  ArrowRight,
  ArrowUp,
  CheckCircle2,
  ClipboardPaste,
  Copy,
  Eye,
  EyeOff,
  Key,
  Loader2,
  Plus,
  RotateCcw,
  Trash2,
  X,
  Zap,
} from 'lucide-vue-next'
import { useConsoleChannels } from '@/composables/useConsoleChannels'
import { useLanguage } from '@/composables/useLanguage'
import { buildChannelPayload } from '@/utils/channel-payload'
import { syncBaseUrlsFormState } from '@/utils/channel-dialog-state'
import { getChannelTypeApi, type ManagedChannelType } from '@/utils/channel-type-api'
import { buildExpectedRequestUrls } from '@/utils/expected-request-urls'
import { parseQuickInput } from '@/utils/quick-input-parser'
import type { Channel, DisabledKeyInfo } from '@/services/admin-api'

interface Props {
  channel?: Channel | null
  channelType: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
  (e: 'test-capability', channel: Channel): void
}>()

const { tf } = useLanguage()
const { saveChannel, restoreApiKey } = useConsoleChannels()

const isEditMode = computed(() => !!props.channel)
const isMac = computed(() => typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform))
const saving = ref(false)
const restoringKey = ref('')
const error = ref('')
const quickInput = ref('')
const existingApiKeys = ref<string[]>([])
const newApiKeysText = ref('')
const copiedKeyIndex = ref<number | null>(null)
const localRestoredKeys = ref<Set<string>>(new Set())

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
const modelMappingRows = ref<ModelMappingRow[]>([])
const newModelMapping = reactive<ModelMappingRow>({ id: 0, source: '', target: '', reasoning: '', noVision: false })
const headerRows = ref<HeaderRow[]>([])
const newHeader = reactive<HeaderRow>({ id: 0, key: '', value: '' })

const reasoningParamStyleOptions = [
  { label: 'reasoning.effort', value: 'reasoning' },
  { label: 'reasoning_effort', value: 'reasoning_effort' },
  { label: 'thinking (JD/GLM)', value: 'thinking' },
]

// 思考强度（effort）—— 模型映射第三列使用
// 注意：reka-ui 的 SelectItem 不允许空字符串 value，用 DEFAULT_SELECT_VALUE 哨兵代表"默认/不设置"
const DEFAULT_SELECT_VALUE = 'default'

const reasoningEffortOptions = computed(() => [
  { label: tf('console.form.selectDefault', '默认'), value: DEFAULT_SELECT_VALUE },
  { label: 'None', value: 'none' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
  { label: 'XHigh', value: 'xhigh' },
  { label: 'Max', value: 'max' },
])

const textVerbosityOptions = computed(() => [
  { label: tf('console.form.selectDefault', '默认'), value: DEFAULT_SELECT_VALUE },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
])

// 空字符串 ↔ 哨兵值互转：form 内部保持空串语义，Select 层使用哨兵值
function toSelectValue(value: string) {
  return value === '' ? DEFAULT_SELECT_VALUE : value
}

function fromSelectValue(value: unknown) {
  return value === DEFAULT_SELECT_VALUE ? '' : String(value ?? '')
}

const form = reactive({
  name: '',
  description: '',
  serviceType: '' as 'openai' | 'claude' | 'gemini' | 'responses' | '',
  baseUrl: '',
  baseUrlsText: '',
  website: '',
  proxyUrl: '',
  requestTimeoutMs: '' as string | number,
  streamFirstContentTimeoutEnabled: false,
  streamFirstContentTimeoutMs: 30000,
  streamInactivityTimeoutEnabled: false,
  streamInactivityTimeoutMs: 20000,
  streamToolCallIdleTimeoutEnabled: false,
  streamToolCallIdleTimeoutMs: 30000,
  rateLimitRpm: '' as string | number,
  rateLimitMaxConcurrent: '' as string | number,
  rateLimitAutoFromHeaders: false,
  routePrefix: '',
  insecureSkipVerify: false,
  apiKeysText: '',
  customHeadersText: '{}',
  modelMappingText: '{}',
  reasoningMappingText: '{}',
  reasoningParamStyle: 'reasoning' as 'reasoning' | 'reasoning_effort' | 'thinking',
  textVerbosity: '' as 'low' | 'medium' | 'high' | '',
  supportedModelsText: '',
  noVisionModelsText: '',
  visionFallbackModel: '',
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

function resetForm() {
  form.name = ''
  form.description = ''
  form.serviceType = defaultServiceTypeForChannel()
  form.baseUrl = ''
  form.baseUrlsText = ''
  form.website = ''
  form.proxyUrl = ''
  form.requestTimeoutMs = ''
  form.streamFirstContentTimeoutEnabled = false
  form.streamFirstContentTimeoutMs = 30000
  form.streamInactivityTimeoutEnabled = false
  form.streamInactivityTimeoutMs = 20000
  form.streamToolCallIdleTimeoutEnabled = false
  form.streamToolCallIdleTimeoutMs = 30000
  form.rateLimitRpm = ''
  form.rateLimitMaxConcurrent = ''
  form.rateLimitAutoFromHeaders = false
  form.routePrefix = ''
  form.insecureSkipVerify = false
  form.apiKeysText = ''
  form.customHeadersText = '{}'
  form.modelMappingText = '{}'
  form.reasoningMappingText = '{}'
  form.reasoningParamStyle = 'reasoning'
  form.textVerbosity = ''
  form.supportedModelsText = ''
  form.noVisionModelsText = ''
  form.visionFallbackModel = ''
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
  existingApiKeys.value = []
  newApiKeysText.value = ''
  copiedKeyIndex.value = null
  localRestoredKeys.value = new Set()
  modelMappingRows.value = []
  headerRows.value = []
  error.value = ''
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
  // baseUrls 多 URL 时已包含主 URL；否则回退单个 baseUrl。form.baseUrl 由 watch 派生
  form.baseUrlsText = (ch.baseUrls?.length ? ch.baseUrls : [ch.baseUrl].filter(Boolean)).join('\n')
  form.website = ch.website || ''
  form.proxyUrl = ch.proxyUrl || ''
  form.requestTimeoutMs = ch.requestTimeoutMs || ''
  form.streamFirstContentTimeoutEnabled = !!(ch.streamFirstContentTimeoutMs && ch.streamFirstContentTimeoutMs > 0)
  form.streamFirstContentTimeoutMs = ch.streamFirstContentTimeoutMs && ch.streamFirstContentTimeoutMs > 0 ? ch.streamFirstContentTimeoutMs : 30000
  form.streamInactivityTimeoutEnabled = !!(ch.streamInactivityTimeoutMs && ch.streamInactivityTimeoutMs > 0)
  form.streamInactivityTimeoutMs = ch.streamInactivityTimeoutMs && ch.streamInactivityTimeoutMs > 0 ? ch.streamInactivityTimeoutMs : 20000
  form.streamToolCallIdleTimeoutEnabled = !!(ch.streamToolCallIdleTimeoutMs && ch.streamToolCallIdleTimeoutMs > 0)
  form.streamToolCallIdleTimeoutMs = ch.streamToolCallIdleTimeoutMs && ch.streamToolCallIdleTimeoutMs > 0 ? ch.streamToolCallIdleTimeoutMs : 30000
  form.rateLimitRpm = ch.rateLimitRpm || ''
  form.rateLimitMaxConcurrent = ch.rateLimitMaxConcurrent || ''
  form.rateLimitAutoFromHeaders = !!ch.rateLimitAutoFromHeaders
  form.routePrefix = ch.routePrefix || ''
  form.insecureSkipVerify = ch.insecureSkipVerify ?? false
  existingApiKeys.value = [...(ch.apiKeys || [])]
  form.apiKeysText = ''
  newApiKeysText.value = ''
  copiedKeyIndex.value = null
  localRestoredKeys.value = new Set()
  modelMappingRows.value = modelMappingFromChannel(ch)
  headerRows.value = headerRowsFromChannel(ch)
  form.customHeadersText = stringifyJson(ch.customHeaders)
  form.modelMappingText = stringifyJson(ch.modelMapping)
  form.reasoningMappingText = stringifyJson(ch.reasoningMapping)
  form.reasoningParamStyle = ch.reasoningParamStyle || 'reasoning'
  form.textVerbosity = ch.textVerbosity || ''
  form.supportedModelsText = (ch.supportedModels || []).join('\n')
  // noVisionModels 中命中映射 target 的由行级 toggle 表示，其余保留在文本框，避免重复展示
  const mappedTargets = new Set(Object.values(ch.modelMapping || {}))
  form.noVisionModelsText = (ch.noVisionModels || []).filter(m => !mappedTargets.has(m)).join('\n')
  form.visionFallbackModel = ch.visionFallbackModel || ''
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
  resetForm()
  if (ch) populateFromChannel(ch)
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
  if (!form.name.trim()) errs.name = tf('console.form.nameRequired', '渠道名称必填')
  if (!form.serviceType) errs.serviceType = tf('console.form.serviceTypeRequired', '请选择服务类型')
  if (!form.baseUrlsText.trim()) errs.baseUrl = tf('console.form.baseUrlRequired', '至少需要一个 Base URL')
  // API Key 必填：现有 key + 新增 key，编辑模式下可恢复的 disabled key 也算
  if (!hasConfigurableKeys.value) errs.apiKeys = tf('console.form.apiKeyRequired', '至少需要一个 API Key')
  if (String(form.requestTimeoutMs).trim()) {
    const timeout = Number(form.requestTimeoutMs)
    if (!Number.isInteger(timeout) || timeout <= 0) {
      errs.requestTimeoutMs = tf('console.form.requestTimeoutInvalid', '请求超时必须是正整数毫秒')
    }
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

function maskApiKey(key: string): string {
  if (key.length <= 10) return `${key.slice(0, 3)}***${key.slice(-2)}`
  return `${key.slice(0, 8)}***${key.slice(-5)}`
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
  if (result.detectedServiceType && !form.serviceType) form.serviceType = result.detectedServiceType
  if (!form.serviceType) form.serviceType = defaultServiceTypeForChannel()
  if (!form.name.trim()) {
    const st = form.serviceType || 'channel'
    form.name = `${props.channelType}-${st}-${Date.now().toString(36)}`
  }
}

function buildSubmitPayload() {
  const payload = isEditMode.value
    ? buildCurrentPayload()
    : buildChannelPayload({
        name: form.name,
        serviceType: form.serviceType,
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
        reasoningMapping: parseJsonObject<Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>>(form.reasoningMappingText, 'Reasoning mapping'),
        reasoningParamStyle: form.reasoningParamStyle,
        textVerbosity: form.textVerbosity,
        fastMode: form.fastMode,
        customHeaders: parseJsonObject<Record<string, string>>(form.customHeadersText, 'Custom headers'),
        proxyUrl: form.proxyUrl,
        requestTimeoutMs: form.requestTimeoutMs,
        streamFirstContentTimeoutMs: form.streamFirstContentTimeoutEnabled ? form.streamFirstContentTimeoutMs : undefined,
        streamInactivityTimeoutMs: form.streamInactivityTimeoutEnabled ? form.streamInactivityTimeoutMs : undefined,
        streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutEnabled ? form.streamToolCallIdleTimeoutMs : undefined,
        routePrefix: form.routePrefix,
        supportedModels: parseLines(form.supportedModelsText),
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
        noVisionModels: parseLines(form.noVisionModelsText),
        visionFallbackModel: form.visionFallbackModel,
        historicalImageTurnLimit: form.historicalImageTurnLimit,
      })

  if (isEditMode.value && props.channel?.requestTimeoutMs && !String(form.requestTimeoutMs ?? '').trim()) {
    payload.requestTimeoutMs = 0
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
  }
  if (isEditMode.value && props.channel?.rateLimitMaxConcurrent && !payload.rateLimitMaxConcurrent) {
    payload.rateLimitMaxConcurrent = 0
  }

  return payload
}

async function persistCurrentDraft(options: { notifyParent?: boolean; close?: boolean } = {}) {
  if (!isValid.value) {
    error.value = Object.values(errors.value)[0] || ''
    return false
  }

  saving.value = true
  error.value = ''
  try {
    await saveChannel(buildSubmitPayload(), props.channel?.index ?? null)
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
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})

// ── API Key 操作 ──

function findDuplicateKeyIndex(newKey: string): number {
  return existingApiKeys.value.findIndex(k => k === newKey)
}

async function addNewApiKeys() {
  const lines = parseLines(newApiKeysText.value)
  const errors: string[] = []
  for (const k of lines) {
    if (findDuplicateKeyIndex(k) !== -1) {
      errors.push(maskApiKey(k))
    } else {
      existingApiKeys.value.push(k)
    }
  }
  if (errors.length) {
    error.value = `重复 key: ${errors.join(', ')}`
  }
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
    await restoreApiKey(props.channel.index, key)
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
  if (!newModelMapping.source.trim() || !newModelMapping.target.trim()) return
  modelMappingRows.value.push({
    id: ++rowId,
    source: newModelMapping.source.trim(),
    target: newModelMapping.target.trim(),
    reasoning: newModelMapping.reasoning || '',
    noVision: newModelMapping.noVision,
  })
  newModelMapping.source = ''
  newModelMapping.target = ''
  newModelMapping.reasoning = ''
  newModelMapping.noVision = false
}

function removeModelMappingRow(index: number) {
  modelMappingRows.value.splice(index, 1)
}

// ── 预设模板 ──

type ModelMappingPresetEntry = { source: string; target: string; reasoning?: ModelMappingRow['reasoning'] }

const modelMappingPresets: Record<string, { mapping: ModelMappingPresetEntry[]; fastMode?: boolean; textVerbosity?: string }> = {
  'gpt-5.5': {
    mapping: [
      { source: 'fable', target: 'gpt-5.5', reasoning: 'xhigh' },
      { source: 'opus', target: 'gpt-5.5', reasoning: 'xhigh' },
      { source: 'sonnet', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'haiku', target: 'gpt-5.4-mini', reasoning: 'high' },
    ],
    fastMode: true,
    textVerbosity: 'medium',
  },
  'gpt-5.4': {
    mapping: [
      { source: 'fable', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'opus', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'sonnet', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'haiku', target: 'gpt-5.4-mini', reasoning: 'high' },
    ],
    fastMode: true,
    textVerbosity: 'medium',
  },
}

type ClaudePresetEntry = { source: string; target: string }
const claudeChannelPresets: Record<string, {
  mapping: ClaudePresetEntry[]
  passbackReasoningContent: boolean
  passbackThinkingBlocks: boolean
  stripEmptyTextBlocks: boolean
  normalizeSystemRoleToTopLevel: boolean
  noVision: boolean
  noVisionModels: string[]
  visionFallbackModel: string
}> = {
  mimo: {
    mapping: [
      { source: 'fable', target: 'mimo-v2.5-pro' },
      { source: 'opus', target: 'mimo-v2.5-pro' },
      { source: 'sonnet', target: 'mimo-v2.5-pro' },
      { source: 'haiku', target: 'mimo-v2.5-pro' },
    ],
    passbackReasoningContent: true,
    passbackThinkingBlocks: false,
    stripEmptyTextBlocks: false,
    normalizeSystemRoleToTopLevel: false,
    noVision: false,
    noVisionModels: ['mimo-v2.5-pro'],
    visionFallbackModel: 'mimo-v2.5',
  },
  deepseek: {
    mapping: [
      { source: 'fable', target: 'deepseek-v4-pro' },
      { source: 'opus', target: 'deepseek-v4-pro' },
      { source: 'sonnet', target: 'deepseek-v4-pro' },
      { source: 'haiku', target: 'deepseek-v4-flash' },
    ],
    passbackReasoningContent: true,
    passbackThinkingBlocks: true,
    stripEmptyTextBlocks: true,
    normalizeSystemRoleToTopLevel: false,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  minimax: {
    mapping: [
      { source: 'fable', target: 'MiniMax-M3' },
      { source: 'opus', target: 'MiniMax-M3' },
      { source: 'sonnet', target: 'MiniMax-M3' },
      { source: 'haiku', target: 'MiniMax-M2.7' },
    ],
    passbackReasoningContent: true,
    passbackThinkingBlocks: false,
    stripEmptyTextBlocks: false,
    normalizeSystemRoleToTopLevel: false,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: '',
  },
}

const codexResponsesPresets: Record<string, {
  mapping: { source: string; target: string; reasoning?: ModelMappingRow['reasoning'] }[]
  reasoningParamStyle: string
  codexNativeToolPassthrough: boolean
  codexToolCompat: boolean
  stripCodexClientTools: boolean
  stripImageGenerationTool: boolean
  normalizeNonstandardChatRoles: boolean
  noVision: boolean
  noVisionModels: string[]
  visionFallbackModel: string
}> = {
  mimo: {
    mapping: [
      { source: 'codex', target: 'mimo-v2.5' },
      { source: 'gpt', target: 'mimo-v2.5-pro' },
    ],
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: false,
    codexToolCompat: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: ['mimo-v2.5-pro'],
    visionFallbackModel: 'mimo-v2.5',
  },
  deepseek: {
    mapping: [
      { source: 'codex', target: 'deepseek-v4-flash' },
      { source: 'gpt', target: 'deepseek-v4-pro', reasoning: 'max' },
      { source: 'mini', target: 'deepseek-v4-flash' },
    ],
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: true,
    codexToolCompat: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: true,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  compshare: {
    mapping: [
      { source: 'codex', target: 'deepseek-v4-flash' },
      { source: 'gpt', target: 'glm-5.1' },
      { source: 'mini', target: 'deepseek-v4-flash' },
    ],
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: true,
    codexToolCompat: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: true,
    noVision: false,
    noVisionModels: ['deepseek-v4-flash'],
    visionFallbackModel: 'MiniMax-M2.7',
  },
  minimax: {
    mapping: [
      { source: 'codex', target: 'MiniMax-M2.7' },
      { source: 'gpt', target: 'MiniMax-M3' },
      { source: 'mini', target: 'MiniMax-M2.7' },
    ],
    reasoningParamStyle: '',
    codexNativeToolPassthrough: true,
    codexToolCompat: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: true,
    noVision: false,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  dashscope: {
    mapping: [
      { source: 'codex', target: 'deepseek-v4-flash' },
      { source: 'gpt', target: 'deepseek-v4-pro', reasoning: 'max' },
      { source: 'mini', target: 'deepseek-v4-flash' },
    ],
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: false,
    codexToolCompat: true,
    stripCodexClientTools: true,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  kimi: {
    mapping: [
      { source: 'codex', target: 'kimi-k2.6' },
      { source: 'gpt', target: 'kimi-k2.6' },
    ],
    reasoningParamStyle: '',
    codexNativeToolPassthrough: false,
    codexToolCompat: true,
    stripCodexClientTools: true,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  glm: {
    mapping: [
      { source: 'codex', target: 'glm-5.1' },
      { source: 'gpt', target: 'glm-5.1' },
    ],
    reasoningParamStyle: '',
    codexNativeToolPassthrough: false,
    codexToolCompat: true,
    stripCodexClientTools: true,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  'opencode-zen': {
    mapping: [
      { source: 'codex', target: 'glm-5.1' },
      { source: 'gpt', target: 'glm-5.1' },
    ],
    reasoningParamStyle: '',
    codexNativeToolPassthrough: false,
    codexToolCompat: true,
    stripCodexClientTools: true,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: [],
    visionFallbackModel: '',
  },
  'opencode-go': {
    mapping: [
      { source: 'codex', target: 'glm-5.1' },
      { source: 'gpt', target: 'glm-5.1' },
    ],
    reasoningParamStyle: '',
    codexNativeToolPassthrough: false,
    codexToolCompat: true,
    stripCodexClientTools: true,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: [],
    visionFallbackModel: '',
  },
}

const serviceTypeOptions = computed(() => {
  const all = [
    { label: 'OpenAI Chat', value: 'openai' },
    { label: 'Claude', value: 'claude' },
    { label: 'Gemini', value: 'gemini' },
    { label: 'Responses (Codex)', value: 'responses' },
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

const supportsOpenAIAdvanced = computed(() => form.serviceType === 'openai' || form.serviceType === 'responses')
const showModelMappingPresets = computed(() => props.channelType === 'messages' && supportsOpenAIAdvanced.value)
const showClaudeChannelPresets = computed(() => form.serviceType === 'claude' && ['messages', 'chat', 'responses'].includes(props.channelType))
const showCodexResponsesPresets = computed(() => props.channelType === 'responses' && supportsOpenAIAdvanced.value)

function applyModelMappingPreset(name: string) {
  const preset = modelMappingPresets[name]
  if (!preset) return
  // 仅 OpenAI/Responses 上游应用 reasoning 映射（对齐 WebUI）
  const applyReasoning = supportsOpenAIAdvanced.value
  modelMappingRows.value = preset.mapping.map(m => ({
    id: ++rowId,
    source: m.source,
    target: m.target,
    reasoning: applyReasoning ? (m.reasoning || '') : '',
    noVision: false,
  }))
  if (preset.fastMode !== undefined) form.fastMode = preset.fastMode
  if (preset.textVerbosity !== undefined) form.textVerbosity = preset.textVerbosity as typeof form.textVerbosity
}

function applyClaudePreset(name: string) {
  const preset = claudeChannelPresets[name]
  if (!preset) return
  modelMappingRows.value = preset.mapping.map(m => ({ id: ++rowId, source: m.source, target: m.target, reasoning: '', noVision: false }))
  form.passbackReasoningContent = preset.passbackReasoningContent
  form.passbackThinkingBlocks = preset.passbackThinkingBlocks
  form.stripEmptyTextBlocks = preset.stripEmptyTextBlocks
  form.normalizeSystemRoleToTopLevel = preset.normalizeSystemRoleToTopLevel
  form.noVision = preset.noVision
  form.noVisionModelsText = preset.noVisionModels.join('\n')
  form.visionFallbackModel = preset.visionFallbackModel
}

function applyCodexResponsesPreset(name: string) {
  const preset = codexResponsesPresets[name]
  if (!preset) return
  modelMappingRows.value = preset.mapping.map(m => ({ id: ++rowId, source: m.source, target: m.target, reasoning: m.reasoning || '', noVision: false }))
  form.reasoningParamStyle = preset.reasoningParamStyle as typeof form.reasoningParamStyle
  form.codexNativeToolPassthrough = preset.codexNativeToolPassthrough
  form.codexToolCompat = preset.codexToolCompat
  form.stripCodexClientTools = preset.stripCodexClientTools
  form.stripImageGenerationTool = preset.stripImageGenerationTool
  form.normalizeNonstandardChatRoles = preset.normalizeNonstandardChatRoles
  form.noVision = preset.noVision
  form.noVisionModelsText = preset.noVisionModels.join('\n')
  form.visionFallbackModel = preset.visionFallbackModel
}

// ── 模型列表拉取 ──

const fetchingModels = ref(false)
const targetModelOptions = ref<string[]>([])
const fetchedModelsError = ref('')
const hasTriedFetchModels = ref(false)

// 切换渠道时重置拉取状态（独立于 resetForm，避免与早于本 ref 定义的 props.channel watch 产生 TDZ）
watch(() => props.channel?.index, () => {
  targetModelOptions.value = []
  fetchedModelsError.value = ''
  hasTriedFetchModels.value = false
})

// ── Source 模型预置列表（对齐 WebUI allSourceModelOptions） ──
const sourceModelOptions = computed(() => {
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

// ── Target 模型预置列表（未拉取真实模型前的候选 fallback） ──
const targetModelPresets = computed(() => {
  if (props.channelType === 'images') {
    return ['gpt-image-2', 'gpt-image-1', 'dall-e-3', 'dall-e-2']
  }
  if (props.channelType === 'gemini' || form.serviceType === 'gemini') {
    return ['gemini-3-pro-preview', 'gemini-3-flash-preview', 'gemini-2.5-pro', 'gemini-2.5-flash', 'gemini-2.5-flash-lite']
  }
  if (form.serviceType === 'claude') {
    return ['claude-opus-4-1', 'claude-sonnet-4-5', 'claude-haiku-4-5', 'mimo-v2.5-pro', 'mimo-v2.5', 'deepseek-v4-pro', 'deepseek-v4-flash']
  }
  // openai / responses 等 OpenAI 兼容上游
  return ['gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini', 'deepseek-v4-pro', 'deepseek-v4-flash', 'mimo-v2.5-pro', 'mimo-v2.5']
})

// 拉取到真实模型则优先，否则用预置候选；datalist 始终有内容
const targetModelDatalist = computed(() => targetModelOptions.value.length ? targetModelOptions.value : targetModelPresets.value)

const commonSupportedModelFilters = ['claude-*', 'gpt-5*', 'gpt-image-2', 'grok-4*', 'gemini-3*', '!*image*']
const selectedSupportedModelSet = computed(() => new Set(parseLines(form.supportedModelsText)))

function toggleSupportedModelFilter(filter: string) {
  const current = parseLines(form.supportedModelsText)
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

async function fetchTargetModels() {
  if (!props.channel) return
  if (!form.baseUrl.trim() || getSubmitApiKeys().length === 0) {
    fetchedModelsError.value = tf('console.form.modelFetchNeedsConfig', '需要 Base URL 和 API Key 才能获取模型列表')
    return
  }

  fetchingModels.value = true
  fetchedModelsError.value = ''
  try {
    const saved = await persistCurrentDraft()
    if (!saved) {
      fetchedModelsError.value = error.value
      return
    }

    const typeApi = getChannelTypeApi(props.channelType as ManagedChannelType)
    const keys = getSubmitApiKeys()
    const resp = await typeApi.getChannelModels(props.channel.index, {
      key: keys[0],
      baseUrl: form.baseUrl,
      proxyUrl: form.proxyUrl,
      insecureSkipVerify: form.insecureSkipVerify,
    })
    // 上游原始响应：Claude/OpenAI 返回 { data: [...] }，部分返回裸数组
    const list: any[] = Array.isArray(resp) ? resp : (resp?.data ?? [])
    targetModelOptions.value = [...new Set<string>(list.map((m: any) => m.id || m.name || String(m)).filter(Boolean))].sort()
  } catch (e) {
    fetchedModelsError.value = e instanceof Error ? e.message : String(e)
  } finally {
    fetchingModels.value = false
  }
}

// target 框首次聚焦时自动拉取真实模型（仅编辑模式、配置齐全、未拉取过）
function handleTargetFocus() {
  if (hasTriedFetchModels.value || fetchingModels.value) return
  if (!props.channel || !form.baseUrl.trim() || getSubmitApiKeys().length === 0) return
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

function getNoVisionModelsFromRows(): string[] {
  // 合并：模型行勾选的 noVision target + 高级选项里手动维护的列表
  const set = new Set<string>(parseLines(form.noVisionModelsText))
  for (const row of modelMappingRows.value) {
    if (row.noVision && row.target) set.add(row.target)
  }
  return [...set]
}

// 生成参数分组是否有可见内容（fastMode/textVerbosity 仅 OpenAI/Responses；vision fallback 仅有 noVision 模型时）
const hasGenerationParams = computed(() => supportsOpenAIAdvanced.value || getNoVisionModelsFromRows().length > 0)

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

function removeHeaderRow(index: number) {
  headerRows.value.splice(index, 1)
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

function buildCurrentPayload() {
  const modelMapping = getModelMappingAsObject()
  const reasoningMapping = getReasoningMappingAsObject() as Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>

  return buildChannelPayload({
    name: form.name,
    serviceType: form.serviceType,
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
    reasoningMapping,
    reasoningParamStyle: form.reasoningParamStyle,
    textVerbosity: form.textVerbosity,
    fastMode: form.fastMode,
    customHeaders: getHeadersAsObject(),
    proxyUrl: form.proxyUrl,
    requestTimeoutMs: form.requestTimeoutMs,
    streamFirstContentTimeoutMs: form.streamFirstContentTimeoutEnabled ? form.streamFirstContentTimeoutMs : undefined,
    streamInactivityTimeoutMs: form.streamInactivityTimeoutEnabled ? form.streamInactivityTimeoutMs : undefined,
    streamToolCallIdleTimeoutMs: form.streamToolCallIdleTimeoutEnabled ? form.streamToolCallIdleTimeoutMs : undefined,
    rateLimitRpm: form.rateLimitRpm,
    rateLimitMaxConcurrent: form.rateLimitMaxConcurrent,
    rateLimitAutoFromHeaders: form.rateLimitAutoFromHeaders,
    routePrefix: form.routePrefix,
    supportedModels: parseLines(form.supportedModelsText),
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
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="true"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('close')" />

        <div class="relative z-10 flex max-h-[90vh] w-[94vw] max-w-6xl flex-col overflow-hidden border border-border bg-card shadow-2xl">
          <div class="flex shrink-0 items-start justify-between gap-3 border-b border-border p-4">
            <div class="min-w-0 space-y-1">
              <div class="text-xs font-bold uppercase tracking-[0.18em] text-primary">
                {{ channelType }} CHANNEL
              </div>
              <h3 class="text-lg font-semibold">
                {{ isEditMode
                  ? tf('console.form.editChannel', '编辑渠道')
                  : tf('console.form.addChannel', '添加渠道')
                }}
              </h3>
            </div>
            <div class="flex shrink-0 items-center gap-1">
              <template v-if="isEditMode">
                <Button variant="ghost" size="icon-sm" :title="form.noVision ? tf('console.form.visionDisabled', '视觉已禁用') : tf('console.form.visionEnabled', '视觉已启用')" @click="form.noVision = !form.noVision">
                  <EyeOff v-if="form.noVision" class="h-4 w-4 text-amber-500" />
                  <Eye v-else class="h-4 w-4 text-muted-foreground" />
                </Button>
                <Button v-if="channelType !== 'images'" variant="outline" size="sm" :disabled="saving" @click="handleTestCapability">
                  <Zap class="h-3.5 w-3.5" />
                  {{ tf('console.actions.capability', '能力测试') }}
                </Button>
              </template>
              <Button variant="ghost" size="icon-sm" class="shrink-0" @click="emit('close')">
                <X class="h-4 w-4" />
              </Button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto">
            <form class="grid gap-5 p-4 lg:grid-cols-[1fr_1fr]" @submit.prevent="handleSubmit">
              <div v-if="error" class="lg:col-span-2 border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                {{ error }}
              </div>

              <!-- ── 创建模式：仅保留快速粘贴 ── -->
              <section v-if="!isEditMode" class="space-y-3 border border-primary/20 bg-primary/5 p-4 lg:col-span-2">
                <div>
                  <h4 class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-primary">
                    <ClipboardPaste class="h-3.5 w-3.5" />
                    {{ tf('addChannel.quickMode', '快速粘贴') }}
                  </h4>
                  <p class="mt-1 text-xs text-muted-foreground">
                    {{ tf('addChannel.quickHint', '粘贴 Base URL、API Key 或完整配置片段，自动识别并填入表单。') }}
                  </p>
                </div>
                <Textarea
                  v-model="quickInput"
                  rows="10"
                  class="!field-sizing-none min-h-[14rem] font-mono text-xs"
                  placeholder="https://api.example.com/v1&#10;sk-..."
                  @paste="handleQuickPaste(($event.clipboardData?.getData('text/plain') || ''))"
                />
                <div class="grid gap-2 md:grid-cols-2">
                  <div class="border border-border bg-background/70 p-2 text-xs">
                    <div class="mb-1 flex items-center gap-1.5 font-semibold">
                      <CheckCircle2 v-if="detectedBaseUrls.length" class="h-3.5 w-3.5 text-emerald-500" />
                      <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
                      Base URLs
                    </div>
                    <p class="truncate text-muted-foreground">
                      {{ detectedBaseUrls.length ? detectedBaseUrls.join(' · ') : tf('addChannel.noneDetected', '未识别') }}
                    </p>
                  </div>
                  <div class="border border-border bg-background/70 p-2 text-xs">
                    <div class="mb-1 flex items-center gap-1.5 font-semibold">
                      <CheckCircle2 v-if="detectedApiKeys.length" class="h-3.5 w-3.5 text-emerald-500" />
                      <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
                      {{ tf('console.form.apiKeys', 'API Keys') }}
                    </div>
                    <p class="text-muted-foreground">
                      {{ detectedApiKeys.length ? `${detectedApiKeys.length} ${tf('console.keys.active', 'active keys')}` : tf('addChannel.noneDetected', '未识别') }}
                    </p>
                  </div>
                </div>
              </section>

              <!-- ── 编辑模式：完整表单 ── -->
              <template v-if="isEditMode">
                <section class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.basicInfo', '基础信息') }}
                  </h4>
                  <div class="grid grid-cols-[2fr_1fr] gap-3">
                    <div class="space-y-1.5">
                      <Label>{{ tf('console.form.name', '名称') }} *</Label>
                      <Input v-model="form.name" :class="{ 'border-destructive': errors.name }" />
                      <p v-if="errors.name" class="text-[10px] text-destructive">{{ errors.name }}</p>
                    </div>
                    <div class="space-y-1.5">
                      <Label>{{ tf('console.form.serviceType', '服务类型') }} *</Label>
                      <Select v-model="form.serviceType">
                        <SelectTrigger :class="['w-full', { 'border-destructive': errors.serviceType }]">
                          <SelectValue :placeholder="tf('console.form.selectServiceType', '选择服务类型')" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="opt in serviceTypeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
                        </SelectContent>
                      </Select>
                      <p v-if="errors.serviceType" class="text-[10px] text-destructive">{{ errors.serviceType }}</p>
                    </div>
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.description', '描述') }}</Label>
                    <Textarea v-model="form.description" rows="2" />
                  </div>
                </section>

                <section class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.connection', '连接') }}
                  </h4>
                  <div class="space-y-1.5">
                    <div class="flex items-center justify-between">
                      <Label>{{ tf('console.form.baseUrl', 'Base URL') }} *</Label>
                      <span class="text-[10px] text-muted-foreground">{{ tf('console.form.baseUrlHint', '支持多个 Base URL 轮换，每行一个，第一行为主地址') }}</span>
                    </div>
                    <Textarea
                      v-model="form.baseUrlsText"
                      class="min-h-20 font-mono text-xs"
                      :placeholder="tf('console.form.baseUrlPlaceholder', '每行一个，第一行为主地址，其余作为故障转移\nhttps://api.example.com\nhttps://backup.example.com')"
                      :class="{ 'border-destructive': errors.baseUrl }"
                    />
                    <p v-if="errors.baseUrl" class="text-[10px] text-destructive">{{ errors.baseUrl }}</p>
                    <div v-if="expectedRequestUrls.length" class="space-y-0.5">
                      <div v-for="(item, index) in expectedRequestUrls" :key="index" class="text-[10px] text-muted-foreground">
                        {{ tf('addChannel.expectedRequest', '预期请求') }} {{ item.expectedUrl }}
                      </div>
                    </div>
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.website', '网站') }}</Label>
                    <Input v-model="form.website" placeholder="https://example.com" />
                  </div>
                </section>

                <section class="space-y-3 border bg-background/40 p-4 lg:col-span-2" :class="errors.apiKeys ? 'border-destructive/40' : 'border-border'">
                  <h4 class="text-xs font-semibold uppercase tracking-wider" :class="errors.apiKeys ? 'text-destructive' : 'text-muted-foreground'">
                    {{ tf('console.form.authentication', '认证') }} *
                  </h4>
                  <div class="space-y-2">
                    <div class="flex items-center justify-between gap-2">
                      <Label>{{ tf('console.form.apiKeys', 'API Keys') }}</Label>
                      <span class="text-[10px] text-muted-foreground">{{ existingApiKeys.length }} {{ tf('console.keys.active', 'active keys') }}</span>
                    </div>
                    <p v-if="errors.apiKeys" class="text-[10px] text-destructive">{{ errors.apiKeys }}</p>
                    <div v-if="existingApiKeys.length" class="space-y-1.5">
                      <div
                        v-for="(key, index) in existingApiKeys"
                        :key="`${index}-${key}`"
                        class="flex items-center justify-between gap-2 border border-border bg-background/60 px-2 py-1.5 text-xs"
                      >
                        <div class="flex min-w-0 items-center gap-2">
                          <Key class="h-3.5 w-3.5 shrink-0 text-primary" />
                          <code class="truncate font-mono text-muted-foreground">{{ maskApiKey(key) }}</code>
                          <span v-if="findDuplicateKeyIndex(key) !== index && existingApiKeys.indexOf(key) !== index" class="text-[10px] text-amber-600">{{ tf('addChannel.duplicateKey', '重复') }}</span>
                        </div>
                        <div class="flex shrink-0 items-center gap-0.5">
                          <Button size="icon-sm" variant="ghost" :class="copiedKeyIndex === index ? 'text-emerald-500' : 'text-muted-foreground'" @click="copyApiKey(key, index)">
                            <CheckCircle2 v-if="copiedKeyIndex === index" class="h-3.5 w-3.5" />
                            <Copy v-else class="h-3.5 w-3.5" />
                          </Button>
                          <Button v-if="index > 0" size="icon-sm" variant="ghost" class="text-muted-foreground" @click="moveApiKeyToTop(index)">
                            <ArrowUp class="h-3.5 w-3.5" />
                          </Button>
                          <Button v-if="index < existingApiKeys.length - 1" size="icon-sm" variant="ghost" class="text-muted-foreground" @click="moveApiKeyToBottom(index)">
                            <ArrowDown class="h-3.5 w-3.5" />
                          </Button>
                          <Button size="icon-sm" variant="ghost" class="text-destructive" @click="removeExistingApiKey(index)">
                            <Trash2 class="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </div>
                    </div>
                    <div class="flex gap-2">
                      <Input
                        v-model="newApiKeysText"
                        class="flex-1 font-mono text-xs"
                        :placeholder="tf('addChannel.addNewApiKeyPlaceholder', '输入新 API Key，回车添加')"
                        @keydown.enter.prevent="addNewApiKeys"
                      />
                      <Button type="button" variant="outline" size="sm" :disabled="!newApiKeysText.trim()" @click="addNewApiKeys">
                        <Plus class="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                  <div v-if="hasDisabledKeys" class="space-y-2 border border-amber-500/20 bg-amber-500/10 p-2">
                    <div class="text-[10px] font-bold uppercase tracking-wider text-amber-700 dark:text-amber-300">
                      {{ tf('console.form.disabledKeys', 'Disabled keys') }} ({{ visibleDisabledKeys.length }})
                    </div>
                    <div v-for="item in visibleDisabledKeys" :key="item.key" class="flex items-center justify-between gap-2 text-xs">
                      <div class="min-w-0 space-y-0.5">
                        <div class="flex min-w-0 items-center gap-1.5">
                          <span class="truncate font-mono text-muted-foreground">{{ maskApiKey(item.key) }}</span>
                          <span v-if="item.reason" class="shrink-0 rounded bg-amber-500/15 px-1 text-[9px] text-amber-700 dark:text-amber-300">{{ item.reason }}</span>
                        </div>
                        <div v-if="item.disabledAt" class="text-[10px] text-muted-foreground">{{ item.disabledAt }}</div>
                      </div>
                      <Button type="button" size="sm" variant="outline" :disabled="restoringKey === item.key" @click="handleDisabledKeyRestore(item.key)">
                        <Loader2 v-if="restoringKey === item.key" class="h-3 w-3 animate-spin" />
                        <RotateCcw v-else class="h-3 w-3" />
                        {{ tf('console.form.restoreKey', 'Restore') }}
                      </Button>
                    </div>
                  </div>
                  <div v-if="historicalApiKeys.length" class="text-xs text-muted-foreground">
                    {{ historicalApiKeys.length }} {{ tf('console.form.historicalKeys', 'historical keys recorded') }}
                  </div>
                </section>

                <section class="space-y-3 border border-border bg-background/40 p-4 lg:col-span-2">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.modelRedirect', '模型重定向') }}
                  </h4>

                  <!-- 预设按钮 -->
                  <div v-if="showModelMappingPresets" class="flex flex-wrap items-center gap-1.5">
                    <span class="text-[10px] text-muted-foreground">{{ tf('addChannel.oneClickSetup', '一键配置') }}</span>
                    <Button v-for="name in Object.keys(modelMappingPresets)" :key="name" type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyModelMappingPreset(name)">
                      <Zap class="mr-1 h-3 w-3" />
                      {{ name }}
                    </Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('mimo')"><Zap class="mr-1 h-3 w-3" />MiMo</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('deepseek')"><Zap class="mr-1 h-3 w-3" />DeepSeek</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('minimax')"><Zap class="mr-1 h-3 w-3" />MiniMax</Button>
                  </div>
                  <div v-if="showClaudeChannelPresets" class="flex flex-wrap items-center gap-1.5">
                    <span class="text-[10px] text-muted-foreground">{{ tf('addChannel.oneClickSetup', '一键配置') }}</span>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('mimo')"><Zap class="mr-1 h-3 w-3" />MiMo</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('deepseek')"><Zap class="mr-1 h-3 w-3" />DeepSeek</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('minimax')"><Zap class="mr-1 h-3 w-3" />MiniMax</Button>
                  </div>
                  <div v-if="showCodexResponsesPresets" class="flex flex-wrap items-center gap-1.5">
                    <span class="text-[10px] text-muted-foreground">{{ tf('addChannel.oneClickSetup', '一键配置') }}</span>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('mimo')"><Zap class="mr-1 h-3 w-3" />MiMo</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('deepseek')"><Zap class="mr-1 h-3 w-3" />DeepSeek</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('compshare')"><Zap class="mr-1 h-3 w-3" />Compshare</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('minimax')"><Zap class="mr-1 h-3 w-3" />MiniMax</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('dashscope')"><Zap class="mr-1 h-3 w-3" />DashScope</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('kimi')"><Zap class="mr-1 h-3 w-3" />Kimi</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('glm')"><Zap class="mr-1 h-3 w-3" />GLM</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('opencode-zen')"><Zap class="mr-1 h-3 w-3" />OpenCode Zen</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('opencode-go')"><Zap class="mr-1 h-3 w-3" />OpenCode Go</Button>
                  </div>

                  <!-- 结构化模型映射行 -->
                  <div class="space-y-2">
                    <div class="flex items-center justify-between">
                      <Label>{{ tf('console.form.modelMapping', '模型映射') }}</Label>
                      <Button v-if="channel" type="button" variant="ghost" size="sm" class="h-6 text-[10px]" :disabled="fetchingModels" @click="fetchTargetModels">
                        <Loader2 v-if="fetchingModels" class="mr-1 h-3 w-3 animate-spin" />
                        {{ fetchingModels ? tf('console.form.fetchingModels', '拉取中...') : tf('console.form.fetchModels', '获取模型列表') }}
                      </Button>
                    </div>
                    <p v-if="fetchedModelsError" class="text-[10px] text-destructive">{{ fetchedModelsError }}</p>

                    <!-- 已配置的重定向 -->
                    <div v-if="modelMappingRows.length" class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                        {{ tf('console.form.modelMappingExisting', '已配置') }} ({{ modelMappingRows.length }})
                      </div>
                      <div v-for="(row, index) in modelMappingRows" :key="row.id" class="flex items-center gap-2 border border-border bg-background/60 px-2 py-1.5 text-xs">
                        <Input v-model="row.source" class="h-7 flex-1 font-mono text-xs" placeholder="source-model" :list="`source-models-${index}`" />
                        <datalist :id="`source-models-${index}`"><option v-for="m in sourceModelOptions" :key="m" :value="m" /></datalist>
                        <ArrowRight class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                        <Input v-model="row.target" class="h-7 flex-1 font-mono text-xs" placeholder="target-model" :list="`target-models-${index}`" @focus="handleTargetFocus" />
                        <datalist :id="`target-models-${index}`">
                          <option v-for="m in targetModelDatalist" :key="m" :value="m" />
                        </datalist>
                        <Select v-if="supportsOpenAIAdvanced" :model-value="toSelectValue(row.reasoning)" @update:model-value="row.reasoning = fromSelectValue($event) as ReasoningEffort | ''">
                          <SelectTrigger class="h-7 w-28 text-xs"><SelectValue :placeholder="tf('console.form.reasoningEffort', '思考强度')" /></SelectTrigger>
                          <SelectContent>
                            <SelectItem v-for="opt in reasoningEffortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
                          </SelectContent>
                        </Select>
                        <Button type="button" size="icon-sm" variant="ghost" :class="row.noVision ? 'text-amber-500' : 'text-muted-foreground'" :title="tf('console.form.noVision', '禁用视觉')" @click="row.noVision = !row.noVision">
                          <EyeOff v-if="row.noVision" class="h-3.5 w-3.5" />
                          <Eye v-else class="h-3.5 w-3.5" />
                        </Button>
                        <Button type="button" size="icon-sm" variant="ghost" class="text-destructive" @click="removeModelMappingRow(index)">
                          <Trash2 class="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </div>

                    <!-- 添加新重定向 -->
                    <div class="space-y-2 border-t border-dashed border-border pt-3">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-primary">
                        {{ tf('console.form.modelMappingAdd', '添加新重定向') }}
                      </div>
                      <div class="flex items-center gap-2 border border-primary/30 bg-primary/5 px-2 py-1.5 text-xs">
                        <Input v-model="newModelMapping.source" class="h-7 flex-1 font-mono text-xs" placeholder="source" list="source-models-new" @keydown.enter.prevent="addModelMappingRow" />
                        <datalist id="source-models-new"><option v-for="m in sourceModelOptions" :key="m" :value="m" /></datalist>
                        <ArrowRight class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                        <Input v-model="newModelMapping.target" class="h-7 flex-1 font-mono text-xs" placeholder="target" list="target-models-new" @focus="handleTargetFocus" @keydown.enter.prevent="addModelMappingRow" />
                        <datalist id="target-models-new">
                          <option v-for="m in targetModelDatalist" :key="m" :value="m" />
                        </datalist>
                        <Select v-if="supportsOpenAIAdvanced" :model-value="toSelectValue(newModelMapping.reasoning)" @update:model-value="newModelMapping.reasoning = fromSelectValue($event) as ReasoningEffort | ''">
                          <SelectTrigger class="h-7 w-28 text-xs"><SelectValue :placeholder="tf('console.form.reasoningEffort', '思考强度')" /></SelectTrigger>
                          <SelectContent>
                            <SelectItem v-for="opt in reasoningEffortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
                          </SelectContent>
                        </Select>
                        <Button type="button" variant="outline" size="sm" :disabled="!newModelMapping.source.trim() || !newModelMapping.target.trim()" @click="addModelMappingRow">
                          <Plus class="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </div>
                  </div>
                </section>

                <!-- ── 生成参数：快速模式 / Text verbosity / 视觉回退 ── -->
                <section v-if="hasGenerationParams" class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.generationParams', '生成参数') }}
                  </h4>

                  <!-- fastMode + textVerbosity（仅 OpenAI/Responses，对齐 WebUI 模型卡片内布局） -->
                  <div v-if="supportsOpenAIAdvanced" class="grid items-end gap-3 md:grid-cols-2">
                    <div class="flex h-9 items-center gap-2">
                      <Switch v-model="form.fastMode" />
                      <Label class="text-xs">{{ tf('console.form.fastMode', '快速模式') }}</Label>
                    </div>
                    <div class="space-y-1">
                      <Label class="text-[10px]">{{ tf('console.form.textVerbosity', 'Text verbosity') }}</Label>
                      <Select :model-value="toSelectValue(form.textVerbosity)" @update:model-value="form.textVerbosity = fromSelectValue($event) as 'low' | 'medium' | 'high' | ''">
                        <SelectTrigger class="h-9 w-full"><SelectValue :placeholder="tf('console.form.textVerbosityPlaceholder', '默认')" /></SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="item in textVerbosityOptions" :key="item.value" :value="item.value">{{ item.label }}</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <!-- Vision fallback model（仅当有模型级 noVision 标记时显示，对齐 WebUI） -->
                  <div v-if="getNoVisionModelsFromRows().length > 0" class="space-y-1.5">
                    <Label>{{ tf('console.form.visionFallbackModel', 'Vision fallback model') }}</Label>
                    <Input v-model="form.visionFallbackModel" class="h-7 text-xs" placeholder="mimo-v2.5" list="vision-fallback-models" @focus="handleTargetFocus" />
                    <datalist id="vision-fallback-models">
                      <option v-for="m in targetModelDatalist" :key="m" :value="m" />
                    </datalist>
                  </div>
                </section>

                <!-- ── 模型范围：支持的模型白名单 ── -->
                <section class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.modelScope', '模型范围') }}
                  </h4>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.supportedModels', '支持的模型（每行一个，留空=全部）') }}</Label>
                    <Textarea v-model="form.supportedModelsText" rows="3" placeholder="gpt-4*&#10;claude-3*" class="font-mono text-xs" />
                    <div class="flex flex-wrap gap-1">
                      <Button
                        v-for="filter in commonSupportedModelFilters"
                        :key="filter"
                        type="button"
                        variant="outline"
                        size="sm"
                        class="h-5 px-1.5 text-[10px]"
                        :class="selectedSupportedModelSet.has(filter) ? 'border-primary bg-primary/10 text-primary' : ''"
                        @click="toggleSupportedModelFilter(filter)"
                      >
                        {{ filter }}
                      </Button>
                    </div>
                  </div>
                </section>

                <section class="space-y-3 border border-border bg-background/40 p-4 lg:col-span-2">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.advancedFlags', '高级选项') }}
                  </h4>

                  <div class="space-y-5">
                    <!-- Vision -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Vision</div>
                      <div class="grid gap-3 md:grid-cols-2">
                        <div class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.noVision" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.noVision', '禁用视觉') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.noVisionHint', '启用后，包含图片的请求将跳过此渠道并 failover 到下一个渠道') }}</p>
                          </div>
                        </div>
                        <div class="space-y-1 md:col-span-2"><Label class="text-[10px]">{{ tf('console.form.noVisionModels', 'No vision models（每行一个）') }}</Label><Textarea v-model="form.noVisionModelsText" rows="2" class="font-mono text-xs" /></div>
                        <div class="space-y-1">
                          <Label class="text-[10px]">{{ tf('console.form.historicalImageTurnLimit', '历史图片轮次限制') }}</Label>
                          <Input v-model.number="form.historicalImageTurnLimit" type="number" min="0" class="h-7 text-xs" placeholder="0" />
                          <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.historicalImageTurnLimitHint', '0 = 继承全局；后端会对 >0 的值应用最低 3 约束') }}</p>
                        </div>
                      </div>
                    </div>

                    <!-- Reasoning / Thinking -->
                    <div class="space-y-2" v-if="form.serviceType === 'claude' || form.serviceType === 'gemini' || supportsOpenAIAdvanced">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Reasoning / Thinking</div>
                      <div class="grid gap-3 md:grid-cols-2">
                        <div v-if="form.serviceType === 'claude' && channelType !== 'images'" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.passbackReasoningContent" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.passbackReasoning', '回传推理内容') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.passbackReasoningHint', '将 thinking 块转为 reasoning_content 回传，兼容 mimo 等要求 OpenAI 风格 reasoning_content 的 Claude 协议上游') }}</p>
                          </div>
                        </div>
                        <div v-if="form.serviceType === 'claude' && channelType !== 'images'" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.passbackThinkingBlocks" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.passbackThinking', '回传思考块') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.passbackThinkingHint', '将真实 reasoning_content 投影为 Claude 的 content[].thinking，兼容 DeepSeek/GLM 等严格 thinking mode 上游') }}</p>
                          </div>
                        </div>
                        <div v-if="form.serviceType === 'gemini' && ['gemini','messages','chat','responses'].includes(channelType)" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.stripThoughtSignature" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.stripThoughtSignature', '移除思考签名') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.stripThoughtSignatureHint', '移除 functionCall 的 thought_signature 字段，兼容不支持该字段的旧版 Gemini API') }}</p>
                          </div>
                        </div>
                        <div v-if="form.serviceType === 'gemini' && ['gemini','messages'].includes(channelType)" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.injectDummyThoughtSignature" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.injectDummySignature', '注入假思考签名') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.injectDummySignatureHint', '为 functionCall 注入 dummy signature，兼容需要该字段的第三方 API（官方 API 请关闭）') }}</p>
                          </div>
                        </div>
                        <div v-if="supportsOpenAIAdvanced" class="flex items-center justify-between gap-3">
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.reasoningParamStyle', '思考方式') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.reasoningParamStyleHint', '选择 OpenAI 风格上游请求使用 reasoning.effort 还是 reasoning_effort。') }}</p>
                          </div>
                          <Select v-model="form.reasoningParamStyle"><SelectTrigger class="h-8 w-40 shrink-0 text-xs"><SelectValue /></SelectTrigger><SelectContent><SelectItem v-for="item in reasoningParamStyleOptions" :key="item.value" :value="item.value">{{ item.label }}</SelectItem></SelectContent></Select>
                        </div>
                      </div>
                    </div>

                    <!-- Codex / Responses -->
                    <div class="space-y-2" v-if="channelType === 'responses'">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Codex / Responses</div>
                      <div class="grid gap-3 md:grid-cols-2">
                        <div class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.codexNativeToolPassthrough" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.codexNativeTools', 'Codex 原生工具透传') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.codexNativeToolsHint', '透传模式下将 Codex 原生工具（apply_patch、namespace 等）转换为 OpenAI function 格式，使上游模型可调用。') }}</p>
                          </div>
                        </div>
                        <div class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.codexToolCompat" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.codexCompat', 'Codex 工具兼容') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.codexCompatHint', '启用 Codex CLI 兼容：Responses 透传上游会剥离客户端专属工具，Chat/Claude/Gemini 上游会转换为 function 代理工具。') }}</p>
                          </div>
                        </div>
                        <div v-if="channelType === 'responses' || channelType === 'chat'" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.stripImageGenerationTool" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.stripImageGenTool', '去除 image_generation 工具') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.stripImageGenToolHint', '从请求的 tools 数组中移除 image_generation 类型，避免未开通图片生成权限的上游返回权限错误。') }}</p>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Compatibility / Normalization -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Compatibility / Normalization</div>
                      <div class="grid gap-3 md:grid-cols-2">
                        <div v-if="form.serviceType === 'claude' && channelType === 'messages'" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.stripEmptyTextBlocks" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.stripEmptyBlocks', '移除空文本块') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.stripEmptyBlocksHint', '转发前移除裸空 text content block，兼容严格拒绝 Claude Code tool_use 占位块的 Claude 协议上游') }}</p>
                          </div>
                        </div>
                        <div v-if="channelType === 'messages'" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.normalizeSystemRoleToTopLevel" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.normalizeSystem', '规范化系统角色') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.normalizeSystemHint', '针对 Opus 4.8 / Fable 5 等新客户端将 system 作为消息 role 发送的情况：转发前抽回顶层 system 字段，兼容仅支持 user/assistant role 的旧 Claude 上游') }}</p>
                          </div>
                        </div>
                        <div v-if="['messages','responses'].includes(channelType)" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.normalizeMetadataUserId" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.normalizeUserId', '规范化用户 ID') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.normalizeUserIdHint', '自动将 JSON 对象格式的 user_id 转换为扁平字符串，确保上游兼容性。') }}</p>
                          </div>
                        </div>
                        <div v-if="channelType === 'messages'" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.stripBillingHeader" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.stripBillingHeader', '移除 CCH 计费参数') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.stripBillingHeaderHint', '转发前从 system 文本块中移除 cch= 计费参数，仅对当前 Messages 渠道生效。') }}</p>
                          </div>
                        </div>
                        <div v-if="channelType === 'chat' || (channelType === 'responses' && form.serviceType === 'openai')" class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.normalizeNonstandardChatRoles" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.normalizeChatRoles', '规范化 Chat 角色') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.normalizeChatRolesHint', '将 developer 等非标准 role 统一转为 user 后转发给上游。国内模型通常不支持非标准 role，建议开启。') }}</p>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Runtime -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Runtime</div>
                      <div class="grid gap-3 md:grid-cols-2">
                        <div class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.lowQuality" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.lowQuality', '低质量标记') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.lowQualityHint', '启用后强制本地估算 token 数量，偏差超过 5% 时使用本地值') }}</p>
                          </div>
                        </div>
                        <div class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.autoBlacklistBalance" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.autoBlacklist', '自动黑名单') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.autoBlacklistHint', '当上游返回余额不足时，自动将该 Key 移入拉黑列表。') }}</p>
                          </div>
                        </div>
                        <div class="flex flex-row-reverse items-center justify-between gap-3">
                          <Switch v-model="form.insecureSkipVerify" class="shrink-0" />
                          <div class="min-w-0 space-y-0.5">
                            <Label class="text-xs">{{ tf('console.form.insecureSkipVerify', '跳过 TLS 验证') }}</Label>
                            <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.insecureSkipVerifyHint', '仅在自签名或域名不匹配时临时启用，生产环境请关闭') }}</p>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Transport -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Transport</div>
                      <div class="grid gap-3 md:grid-cols-3">
                        <div class="space-y-1">
                          <Label class="text-[10px]">{{ tf('console.form.proxyUrl', '代理 URL') }}</Label>
                          <Input v-model="form.proxyUrl" class="h-7 text-xs" placeholder="socks5://..." />
                          <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.proxyUrlHint', '支持 HTTP/HTTPS/SOCKS5 代理，用于通过代理访问上游服务') }}</p>
                        </div>
                        <div class="space-y-1">
                          <Label class="text-[10px]">{{ tf('console.form.routePrefix', '路由前缀') }}</Label>
                          <Input v-model="form.routePrefix" class="h-7 text-xs" placeholder="kimi" />
                          <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.routePrefixHint', '通过 /{前缀}/v1/messages 访问此渠道，多个渠道可共享同一前缀') }}</p>
                        </div>
                        <div class="space-y-1">
                          <Label class="text-[10px]">{{ tf('console.form.requestTimeoutMs', '请求超时（ms）') }}</Label>
                          <Input v-model="form.requestTimeoutMs" type="number" class="h-7 text-xs" placeholder="60000" :class="{ 'border-destructive': errors.requestTimeoutMs }" />
                          <p v-if="errors.requestTimeoutMs" class="text-[10px] text-destructive">{{ errors.requestTimeoutMs }}</p>
                          <p v-else class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.requestTimeoutMsHint', '仅作用于非流式上游请求；留空表示继承全局 REQUEST_TIMEOUT。') }}</p>
                        </div>
                        <div class="space-y-1">
                          <p class="text-[10px] font-medium text-foreground">{{ tf('console.form.rateLimitSectionLabel', '主动限速') }}</p>
                          <p class="text-[10px] leading-4 text-muted-foreground mb-2">{{ tf('console.form.rateLimitSectionHint', '在请求发往上游前主动限流，避免触发上游 429。') }}</p>
                          <div class="grid grid-cols-2 gap-2 mb-3">
                            <div class="space-y-1">
                              <Label class="text-[10px]">{{ tf('console.form.rateLimitRpmLabel', 'RPM') }}</Label>
                              <Input v-model="form.rateLimitRpm" type="number" class="h-7 text-xs" placeholder="留空=不限" />
                              <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.rateLimitRpmHint', '每分钟请求数上限') }}</p>
                            </div>
                            <div class="space-y-1">
                              <Label class="text-[10px]">{{ tf('console.form.rateLimitMaxConcurrentLabel', '最大并发') }}</Label>
                              <Input v-model="form.rateLimitMaxConcurrent" type="number" class="h-7 text-xs" placeholder="留空=不限" />
                              <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.rateLimitMaxConcurrentHint', '并发上限') }}</p>
                            </div>
                          </div>
                          <div class="flex items-center gap-2">
                            <Switch v-model="form.rateLimitAutoFromHeaders" />
                            <div class="space-y-0.5">
                              <Label class="text-[10px]">{{ tf('console.form.rateLimitAutoFromHeadersLabel', '自动学习上游限速') }}</Label>
                              <p class="text-[10px] leading-4 text-muted-foreground">{{ tf('console.form.rateLimitAutoFromHeadersHint', '解析 Retry-After / x-ratelimit-* 响应头动态调整 cooldown。') }}</p>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Stream Timeouts -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">{{ tf('console.form.streamTimeouts', '流式超时') }}</div>
                      <div class="grid gap-3 md:grid-cols-2">
                        <div class="border border-border bg-background/60 p-3 space-y-2">
                          <div class="flex items-center justify-between gap-3">
                            <Switch v-model="form.streamFirstContentTimeoutEnabled" class="shrink-0" />
                            <div class="min-w-0 space-y-0.5">
                              <Label class="text-xs">{{ tf('console.form.streamFirstContentTimeoutOverrideLabel', '自定义首字等待超时') }}</Label>
                              <p class="text-[10px] leading-4 text-muted-foreground">{{ form.streamFirstContentTimeoutEnabled ? tf('console.form.streamTimeoutOverrideHint', '自定义值覆盖全局流式超时') : tf('console.form.streamTimeoutInheritHint', '继承全局流式超时') }}</p>
                            </div>
                          </div>
                          <div :class="{ 'opacity-50 pointer-events-none': !form.streamFirstContentTimeoutEnabled }">
                            <div class="flex items-center justify-between mb-1">
                              <span class="text-[10px] text-muted-foreground">{{ tf('console.form.streamFirstContentTimeoutLabel', '首字等待超时') }}</span>
                              <span class="text-[10px] font-medium">{{ (form.streamFirstContentTimeoutMs / 1000) }}s</span>
                            </div>
                            <input
                              v-model.number="form.streamFirstContentTimeoutMs"
                              type="range"
                              min="5000"
                              max="300000"
                              step="1000"
                              class="cb-slider w-full"
                              :disabled="!form.streamFirstContentTimeoutEnabled"
                            />
                            <div class="flex justify-between text-[10px] text-muted-foreground"><span>5s</span><span>300s</span></div>
                          </div>
                        </div>
                        <div class="border border-border bg-background/60 p-3 space-y-2">
                          <div class="flex items-center justify-between gap-3">
                            <Switch v-model="form.streamInactivityTimeoutEnabled" class="shrink-0" />
                            <div class="min-w-0 space-y-0.5">
                              <Label class="text-xs">{{ tf('console.form.streamInactivityTimeoutOverrideLabel', '自定义断流超时') }}</Label>
                              <p class="text-[10px] leading-4 text-muted-foreground">{{ form.streamInactivityTimeoutEnabled ? tf('console.form.streamTimeoutOverrideHint', '自定义值覆盖全局流式超时') : tf('console.form.streamTimeoutInheritHint', '继承全局流式超时') }}</p>
                            </div>
                          </div>
                          <div :class="{ 'opacity-50 pointer-events-none': !form.streamInactivityTimeoutEnabled }">
                            <div class="flex items-center justify-between mb-1">
                              <span class="text-[10px] text-muted-foreground">{{ tf('console.form.streamInactivityTimeoutLabel', '首字后断流超时') }}</span>
                              <span class="text-[10px] font-medium">{{ (form.streamInactivityTimeoutMs / 1000) }}s</span>
                            </div>
                            <input
                              v-model.number="form.streamInactivityTimeoutMs"
                              type="range"
                              min="1000"
                              max="180000"
                              step="1000"
                              class="cb-slider w-full"
                              :disabled="!form.streamInactivityTimeoutEnabled"
                            />
                            <div class="flex justify-between text-[10px] text-muted-foreground"><span>1s</span><span>180s</span></div>
                          </div>
                        </div>
                        <div class="border border-border bg-background/60 p-3 space-y-2">
                          <div class="flex items-center justify-between gap-3">
                            <Switch v-model="form.streamToolCallIdleTimeoutEnabled" class="shrink-0" />
                            <div class="min-w-0 space-y-0.5">
                              <Label class="text-xs">{{ tf('console.form.streamToolCallIdleTimeoutOverrideLabel', '自定义工具调用空闲超时') }}</Label>
                              <p class="text-[10px] leading-4 text-muted-foreground">{{ form.streamToolCallIdleTimeoutEnabled ? tf('console.form.streamTimeoutOverrideHint', '自定义值覆盖全局流式超时') : tf('console.form.streamTimeoutInheritHint', '继承全局流式超时') }}</p>
                            </div>
                          </div>
                          <div :class="{ 'opacity-50 pointer-events-none': !form.streamToolCallIdleTimeoutEnabled }">
                            <div class="flex items-center justify-between mb-1">
                              <span class="text-[10px] text-muted-foreground">{{ tf('console.form.streamToolCallIdleTimeoutLabel', '工具调用空闲超时') }}</span>
                              <span class="text-[10px] font-medium">{{ (form.streamToolCallIdleTimeoutMs / 1000) }}s</span>
                            </div>
                            <input
                              v-model.number="form.streamToolCallIdleTimeoutMs"
                              type="range"
                              min="1000"
                              max="180000"
                              step="1000"
                              class="cb-slider w-full"
                              :disabled="!form.streamToolCallIdleTimeoutEnabled"
                            />
                            <div class="flex justify-between text-[10px] text-muted-foreground"><span>1s</span><span>180s</span></div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </section>

                <section class="space-y-3 border border-border bg-background/40 p-4 lg:col-span-2">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.customHeaders', '自定义 Headers') }}
                  </h4>
                  <div v-if="headerRows.length" class="space-y-1.5">
                    <div v-for="(h, index) in headerRows" :key="h.id" class="flex items-center gap-2 border border-border bg-background/60 px-2 py-1.5 text-xs">
                      <code class="shrink-0 font-mono font-semibold text-primary">{{ h.key }}</code>
                      <span class="shrink-0 text-muted-foreground">:</span>
                      <Input v-model="h.value" class="flex-1 font-mono text-xs" />
                      <Button type="button" size="icon-sm" variant="ghost" class="shrink-0 text-destructive" @click="removeHeaderRow(index)">
                        <Trash2 class="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                  <div class="flex items-center gap-2">
                    <Input v-model="newHeader.key" class="h-7 w-40 font-mono text-xs" placeholder="Header-Name" @keydown.enter.prevent="addHeaderRow" />
                    <Input v-model="newHeader.value" class="flex-1 font-mono text-xs" placeholder="value" @keydown.enter.prevent="addHeaderRow" />
                    <Button type="button" variant="outline" size="sm" :disabled="!newHeader.key.trim()" @click="addHeaderRow">
                      <Plus class="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </section>
              </template>
            </form>
          </div>

          <div class="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-border bg-card p-4">
            <Button variant="ghost" @click="emit('close')">
              {{ tf('common.cancel', '取消') }} <span class="ml-2 text-xs opacity-60">Esc</span>
            </Button>
            <Button type="button" :disabled="!isValid || saving" @click="handleSubmit">
              <Loader2 v-if="saving" class="mr-2 h-4 w-4 animate-spin" />
              {{ isEditMode
                ? tf('console.form.save', '保存')
                : tf('console.form.create', '创建')
              }}
              <span class="ml-2 text-xs opacity-60">{{ isMac ? '⌘ Enter' : 'Ctrl+Enter' }}</span>
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
</style>
