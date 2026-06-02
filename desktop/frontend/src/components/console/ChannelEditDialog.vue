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
  ArrowUp,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  ClipboardPaste,
  Copy,
  Eye,
  EyeOff,
  Key,
  Loader2,
  Plus,
  RotateCcw,
  Trash2,
  Wand2,
  X,
  Zap,
} from 'lucide-vue-next'
import { useConsoleChannels } from '@/composables/useConsoleChannels'
import { useLanguage } from '@/composables/useLanguage'
import { buildChannelPayload } from '@/utils/channel-payload'
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
const showAdvanced = ref(false)
const showProtocolOptions = ref(false)
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
const reasoningEffortOptions = [
  { label: 'Default', value: '' },
  { label: 'None', value: 'none' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
  { label: 'XHigh', value: 'xhigh' },
  { label: 'Max', value: 'max' },
]

const textVerbosityOptions = [
  { label: 'Default', value: '' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
]

const form = reactive({
  name: '',
  description: '',
  serviceType: '' as 'openai' | 'claude' | 'gemini' | 'responses' | '',
  baseUrl: '',
  baseUrlsText: '',
  website: '',
  proxyUrl: '',
  requestTimeoutMs: '' as string | number,
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
  passbackReasoningContent: false,
  passbackThinkingBlocks: false,
  fastMode: false,
  lowQuality: false,
  injectDummyThoughtSignature: false,
  stripThoughtSignature: false,
  stripEmptyTextBlocks: false,
  normalizeSystemRoleToTopLevel: false,
  normalizeMetadataUserId: true,
  normalizeNonstandardChatRoles: false,
  autoBlacklistBalance: true,
  codexNativeToolPassthrough: false,
  codexToolCompat: false,
  stripCodexClientTools: false,
})

const disabledApiKeys = computed<DisabledKeyInfo[]>(() => props.channel?.disabledApiKeys ?? [])
const historicalApiKeys = computed(() => props.channel?.historicalApiKeys ?? [])

const quickDetection = computed(() => parseQuickInput(quickInput.value, form.serviceType || undefined))
const detectedBaseUrls = computed(() => quickDetection.value.detectedBaseUrls)
const detectedApiKeys = computed(() => quickDetection.value.detectedApiKeys)
const quickHasDetections = computed(() => detectedBaseUrls.value.length > 0 || detectedApiKeys.value.length > 0 || !!quickDetection.value.detectedServiceType)

function resetForm() {
  form.name = ''
  form.description = ''
  form.serviceType = defaultServiceTypeForChannel()
  form.baseUrl = ''
  form.baseUrlsText = ''
  form.website = ''
  form.proxyUrl = ''
  form.requestTimeoutMs = ''
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
  form.passbackReasoningContent = false
  form.passbackThinkingBlocks = false
  form.fastMode = false
  form.lowQuality = false
  form.injectDummyThoughtSignature = false
  form.stripThoughtSignature = false
  form.stripEmptyTextBlocks = false
  form.normalizeSystemRoleToTopLevel = false
  form.normalizeMetadataUserId = true
  form.normalizeNonstandardChatRoles = false
  form.autoBlacklistBalance = true
  form.codexNativeToolPassthrough = false
  form.codexToolCompat = false
  form.stripCodexClientTools = false
  quickInput.value = ''
  existingApiKeys.value = []
  newApiKeysText.value = ''
  copiedKeyIndex.value = null
  localRestoredKeys.value = new Set()
  modelMappingRows.value = []
  headerRows.value = []
  error.value = ''
  showAdvanced.value = false
  showProtocolOptions.value = false
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
  form.baseUrl = ch.baseUrl || ''
  form.baseUrlsText = (ch.baseUrls || []).join('\n')
  form.website = ch.website || ''
  form.proxyUrl = ch.proxyUrl || ''
  form.requestTimeoutMs = ch.requestTimeoutMs || ''
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
  form.noVisionModelsText = (ch.noVisionModels || []).join('\n')
  form.visionFallbackModel = ch.visionFallbackModel || ''
  form.noVision = ch.noVision ?? false
  form.passbackReasoningContent = ch.passbackReasoningContent ?? false
  form.passbackThinkingBlocks = ch.passbackThinkingBlocks ?? false
  form.fastMode = ch.fastMode ?? false
  form.lowQuality = ch.lowQuality ?? false
  form.injectDummyThoughtSignature = ch.injectDummyThoughtSignature ?? false
  form.stripThoughtSignature = ch.stripThoughtSignature ?? false
  form.stripEmptyTextBlocks = ch.stripEmptyTextBlocks ?? false
  form.normalizeSystemRoleToTopLevel = ch.normalizeSystemRoleToTopLevel ?? false
  form.normalizeMetadataUserId = ch.normalizeMetadataUserId ?? true
  form.normalizeNonstandardChatRoles = ch.normalizeNonstandardChatRoles ?? false
  form.autoBlacklistBalance = ch.autoBlacklistBalance ?? true
  form.codexNativeToolPassthrough = ch.codexNativeToolPassthrough ?? false
  form.codexToolCompat = ch.codexToolCompat ?? ch.stripCodexClientTools ?? false
  form.stripCodexClientTools = ch.stripCodexClientTools ?? ch.codexToolCompat ?? false
}

watch(() => props.channel, (ch) => {
  resetForm()
  if (ch) populateFromChannel(ch)
}, { immediate: true })

const errors = computed(() => {
  const errs: Record<string, string> = {}
  if (!form.name.trim()) errs.name = tf('console.form.nameRequired', '渠道名称必填')
  if (!form.serviceType) errs.serviceType = tf('console.form.serviceTypeRequired', '请选择服务类型')
  if (!form.baseUrl.trim() && !form.baseUrlsText.trim()) errs.baseUrl = tf('console.form.baseUrlRequired', '至少需要一个 Base URL')
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

function mergeLineText(currentText: string, additions: string[]) {
  const merged = [...parseLines(currentText)]
  for (const addition of additions.map(item => item.trim()).filter(Boolean)) {
    if (!merged.includes(addition)) merged.push(addition)
  }
  return merged.join('\n')
}

function applyQuickInput() {
  const result = quickDetection.value
  if (result.detectedServiceType) form.serviceType = result.detectedServiceType
  if (result.detectedBaseUrl) form.baseUrl = result.detectedBaseUrl
  if (result.detectedBaseUrls.length > 1) form.baseUrlsText = result.detectedBaseUrls.join('\n')
  if (result.detectedApiKeys.length) form.apiKeysText = mergeLineText(form.apiKeysText, result.detectedApiKeys)
  if (!form.name.trim() && result.detectedServiceType) {
    form.name = `${props.channelType}-${result.detectedServiceType}-channel`
  }
}

function handleQuickPaste(text: string) {
  const result = parseQuickInput(text, form.serviceType || undefined)
  if (result.detectedBaseUrl) form.baseUrl = result.detectedBaseUrl
  if (result.detectedBaseUrls.length > 1) form.baseUrlsText = result.detectedBaseUrls.join('\n')
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

async function handleRestoreKey(key: string) {
  if (!props.channel) return
  restoringKey.value = key
  error.value = ''
  try {
    await restoreApiKey(props.channel.index, key)
    emit('saved')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    restoringKey.value = ''
  }
}

async function handleSubmit() {
  if (!isValid.value) return
  saving.value = true
  error.value = ''

  try {
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
          routePrefix: form.routePrefix,
          supportedModels: parseLines(form.supportedModelsText),
          autoBlacklistBalance: form.autoBlacklistBalance,
          normalizeMetadataUserId: form.normalizeMetadataUserId,
          stripEmptyTextBlocks: form.stripEmptyTextBlocks,
          normalizeSystemRoleToTopLevel: form.normalizeSystemRoleToTopLevel,
          codexNativeToolPassthrough: form.codexNativeToolPassthrough,
          codexToolCompat: form.codexToolCompat,
          normalizeNonstandardChatRoles: form.normalizeNonstandardChatRoles,
          stripCodexClientTools: form.stripCodexClientTools,
          noVision: form.noVision,
          noVisionModels: parseLines(form.noVisionModelsText),
          visionFallbackModel: form.visionFallbackModel,
        })

    if (isEditMode.value && props.channel?.requestTimeoutMs && !String(form.requestTimeoutMs ?? '').trim()) {
      payload.requestTimeoutMs = 0
    }

    await saveChannel(payload, props.channel?.index ?? null)
    emit('saved')
    emit('close')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

function shouldSkipEnterSubmit(target: EventTarget | null) {
  const el = target instanceof Element ? target : null
  if (!el) return false

  // textarea / contenteditable 内保留原生编辑行为；按钮和选择器内保留自身 Enter 行为。
  if (el.closest('textarea, button, [contenteditable]')) return true
  const interactiveRole = el.closest('[role="button"], [role="combobox"], [role="listbox"], [role="option"], [role="switch"], [role="checkbox"]')
  return Boolean(interactiveRole)
}

// Keyboard shortcuts: Esc 取消，创建模式 Cmd/Ctrl+Enter / 编辑模式 Enter 保存
const handleGlobalKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
    return
  }

  if (e.key !== 'Enter') return
  if (saving.value) return

  // 编辑模式：直接 Enter 保存（textarea 内不拦截）
  if (isEditMode.value && !e.shiftKey) {
    const el = e.target instanceof Element ? e.target : null
    if (el?.closest('button, [role="combobox"], [role="listbox"], [role="option"], [role="switch"], [role="checkbox"]')) return
    e.preventDefault()
    void handleSubmit()
    return
  }

  // 创建模式：需要 Cmd/Ctrl+Enter 保存（textarea 内也生效）
  if (!isEditMode.value && (e.metaKey || e.ctrlKey) && !e.shiftKey) {
    e.preventDefault()
    void handleSubmit()
    return
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
      { source: 'opus', target: 'gpt-5.5', reasoning: 'xhigh' },
      { source: 'sonnet', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'haiku', target: 'gpt-5.3-codex', reasoning: 'high' },
    ],
    fastMode: true,
    textVerbosity: 'medium',
  },
  'gpt-5.4': {
    mapping: [
      { source: 'opus', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'sonnet', target: 'gpt-5.4', reasoning: 'xhigh' },
      { source: 'haiku', target: 'gpt-5.3-codex', reasoning: 'high' },
    ],
    fastMode: true,
    textVerbosity: 'medium',
  },
  'gpt-5.3-codex': {
    mapping: [
      { source: 'opus', target: 'gpt-5.3-codex', reasoning: 'xhigh' },
      { source: 'sonnet', target: 'gpt-5.3-codex', reasoning: 'xhigh' },
      { source: 'haiku', target: 'gpt-5.3-codex', reasoning: 'high' },
    ],
    fastMode: true,
    textVerbosity: 'medium',
  },
  'gpt-5.2-codex': {
    mapping: [
      { source: 'opus', target: 'gpt-5.2', reasoning: 'xhigh' },
      { source: 'sonnet', target: 'gpt-5.2-codex', reasoning: 'xhigh' },
      { source: 'haiku', target: 'gpt-5.2-codex', reasoning: 'high' },
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
}

const codexResponsesPresets: Record<string, {
  mapping: { source: string; target: string; reasoning?: ModelMappingRow['reasoning'] }[]
  reasoningParamStyle: string
  codexNativeToolPassthrough: boolean
  codexToolCompat: boolean
  stripCodexClientTools: boolean
  normalizeNonstandardChatRoles: boolean
  noVision: boolean
  noVisionModels: string[]
  visionFallbackModel: string
}> = {
  mimo: {
    mapping: [
      { source: 'gpt-5', target: 'mimo-v2.5-pro' },
      { source: 'codex-auto-review', target: 'mimo-v2.5' },
    ],
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: false,
    codexToolCompat: false,
    stripCodexClientTools: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: ['mimo-v2.5-pro'],
    visionFallbackModel: 'mimo-v2.5',
  },
  deepseek: {
    mapping: [
      { source: 'gpt', target: 'deepseek-v4-pro', reasoning: 'max' },
      { source: 'mini', target: 'deepseek-v4-flash' },
      { source: 'codex-auto-review', target: 'deepseek-v4-flash' },
    ],
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: true,
    codexToolCompat: false,
    stripCodexClientTools: false,
    normalizeNonstandardChatRoles: true,
    noVision: true,
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
  form.normalizeNonstandardChatRoles = preset.normalizeNonstandardChatRoles
  form.noVision = preset.noVision
  form.noVisionModelsText = preset.noVisionModels.join('\n')
  form.visionFallbackModel = preset.visionFallbackModel
}

// ── 模型列表拉取 ──

const fetchingModels = ref(false)
const targetModelOptions = ref<string[]>([])
const fetchedModelsError = ref('')

// ── Source 模型预置列表（对齐 WebUI allSourceModelOptions） ──
const sourceModelOptions = computed(() => {
  if (props.channelType === 'chat') {
    return ['codex', 'gpt', 'mini', 'gpt-5', 'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.3-codex', 'gpt-5.2-codex', 'gpt-5.2']
  }
  if (props.channelType === 'images') {
    return ['gpt-image-2', 'gpt-image-1', 'dall-e-3', 'dall-e-2']
  }
  if (props.channelType === 'gemini') {
    return ['gemini-3.5-flash', 'gemini-3.1-pro-preview', 'gemini-3-pro-preview', 'gemini-3-flash-preview', 'gemini-3.1-flash-lite', 'gemini-2.5-pro', 'gemini-2.5-flash', 'gemini-2.5-flash-lite', 'gemini-2']
  }
  if (props.channelType === 'responses') {
    return ['codex', 'codex-auto-review', 'gpt-5', 'gpt', 'mini', 'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.3-codex', 'gpt-5.2-codex', 'gpt-5.2']
  }
  // messages (Claude)
  return ['opus', 'sonnet', 'haiku']
})

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
    const typeApi = getChannelTypeApi(props.channelType as ManagedChannelType)
    const models = await typeApi.getChannelModels(props.channel.index, {
      baseUrl: form.baseUrl,
      proxyUrl: form.proxyUrl,
      insecureSkipVerify: form.insecureSkipVerify,
    })
    targetModelOptions.value = [...new Set<string>((models as any[]).map((m: any) => m.id || m.name || String(m)).filter(Boolean))].sort()
  } catch (e) {
    fetchedModelsError.value = e instanceof Error ? e.message : String(e)
  } finally {
    fetchingModels.value = false
  }
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
  const set = new Set<string>()
  for (const row of modelMappingRows.value) {
    if (row.noVision && row.target) set.add(row.target)
  }
  return [...set]
}

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

  // 先保存当前编辑内容
  saving.value = true
  error.value = ''
  try {
    const payload = buildCurrentPayload()
    await saveChannel(payload, props.channel.index)
    emit('saved')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
    saving.value = false
    return
  }
  saving.value = false

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
    routePrefix: form.routePrefix,
    supportedModels: parseLines(form.supportedModelsText),
    autoBlacklistBalance: form.autoBlacklistBalance,
    normalizeMetadataUserId: form.normalizeMetadataUserId,
    stripEmptyTextBlocks: form.stripEmptyTextBlocks,
    normalizeSystemRoleToTopLevel: form.normalizeSystemRoleToTopLevel,
    codexNativeToolPassthrough: form.codexNativeToolPassthrough,
    codexToolCompat: form.codexToolCompat,
    normalizeNonstandardChatRoles: form.normalizeNonstandardChatRoles,
    stripCodexClientTools: form.stripCodexClientTools,
    noVision: form.noVision,
    noVisionModels: getNoVisionModelsFromRows(),
    visionFallbackModel: form.visionFallbackModel,
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
                      API Keys
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
                  <div class="grid grid-cols-2 gap-3">
                    <div class="space-y-1.5">
                      <Label>{{ tf('console.form.name', '名称') }} *</Label>
                      <Input v-model="form.name" :class="{ 'border-destructive': errors.name }" />
                      <p v-if="errors.name" class="text-[10px] text-destructive">{{ errors.name }}</p>
                    </div>
                    <div class="space-y-1.5">
                      <Label>{{ tf('console.form.serviceType', '服务类型') }} *</Label>
                      <Select v-model="form.serviceType">
                        <SelectTrigger :class="{ 'border-destructive': errors.serviceType }">
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
                  <div class="space-y-1.5">
                    <Label>Website</Label>
                    <Input v-model="form.website" placeholder="https://example.com" />
                  </div>
                </section>

                <section class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.connection', '连接') }}
                  </h4>
                  <div class="space-y-1.5">
                    <Label>{{ tf('console.form.baseUrl', 'Base URL') }} *</Label>
                    <Input
                      v-model="form.baseUrl"
                      placeholder="https://api.example.com"
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
                    <Label>{{ tf('console.form.additionalUrls', '额外 URL（每行一个）') }}</Label>
                    <Textarea v-model="form.baseUrlsText" rows="3" placeholder="https://backup.example.com" />
                  </div>
                </section>

                <section class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.authentication', '认证') }}
                  </h4>
                  <div class="space-y-2">
                    <div class="flex items-center justify-between gap-2">
                      <Label>{{ tf('console.form.apiKeys', 'API Keys') }}</Label>
                      <span class="text-[10px] text-muted-foreground">{{ existingApiKeys.length }} keys</span>
                    </div>
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
                      <Button size="sm" variant="outline" :disabled="restoringKey === item.key" @click="handleDisabledKeyRestore(item.key)">
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

                <section class="space-y-3 border border-border bg-background/40 p-4">
                  <h4 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {{ tf('console.form.models', '模型') }}
                  </h4>

                  <!-- 预设按钮 -->
                  <div v-if="showModelMappingPresets" class="flex flex-wrap items-center gap-1.5">
                    <span class="text-[10px] text-muted-foreground">{{ tf('addChannel.oneClickSetup', '一键配置') }}</span>
                    <Button v-for="(preset, name) in modelMappingPresets" :key="name" type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyModelMappingPreset(String(name))">
                      <Zap class="mr-1 h-3 w-3" />
                      {{ name }}
                    </Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('mimo')"><Zap class="mr-1 h-3 w-3" />MiMo</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('deepseek')"><Zap class="mr-1 h-3 w-3" />DeepSeek</Button>
                  </div>
                  <div v-if="showClaudeChannelPresets" class="flex flex-wrap items-center gap-1.5">
                    <span class="text-[10px] text-muted-foreground">{{ tf('addChannel.oneClickSetup', '一键配置') }}</span>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('mimo')"><Zap class="mr-1 h-3 w-3" />MiMo</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyClaudePreset('deepseek')"><Zap class="mr-1 h-3 w-3" />DeepSeek</Button>
                  </div>
                  <div v-if="showCodexResponsesPresets" class="flex flex-wrap items-center gap-1.5">
                    <span class="text-[10px] text-muted-foreground">{{ tf('addChannel.oneClickSetup', '一键配置') }}</span>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('mimo')"><Zap class="mr-1 h-3 w-3" />MiMo</Button>
                    <Button type="button" variant="outline" size="sm" class="h-6 text-[10px]" @click="applyCodexResponsesPreset('deepseek')"><Zap class="mr-1 h-3 w-3" />DeepSeek</Button>
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
                    <div v-for="(row, index) in modelMappingRows" :key="row.id" class="flex items-center gap-2 border border-border bg-background/60 px-2 py-1.5 text-xs">
                      <Input v-model="row.source" class="h-7 flex-1 font-mono text-xs" placeholder="source-model" :list="`source-models-${index}`" />
                      <datalist :id="`source-models-${index}`"><option v-for="m in sourceModelOptions" :key="m" :value="m" /></datalist>
                      <ArrowRight class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                      <Input v-model="row.target" class="h-7 flex-1 font-mono text-xs" placeholder="target-model" :list="targetModelOptions.length ? `target-models-${index}` : undefined" />
                      <datalist v-if="targetModelOptions.length" :id="`target-models-${index}`">
                        <option v-for="m in targetModelOptions" :key="m" :value="m" />
                      </datalist>
                      <Select v-model="row.reasoning">
                        <SelectTrigger class="h-7 w-28 text-xs"><SelectValue :placeholder="tf('console.form.reasoningEffort', '思考强度')" /></SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="opt in reasoningEffortOptions" :key="opt.value || 'default'" :value="opt.value">{{ opt.label }}</SelectItem>
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
                    <div class="flex items-center gap-2">
                      <Input v-model="newModelMapping.source" class="h-7 flex-1 font-mono text-xs" placeholder="source" list="source-models-new" @keydown.enter.prevent="addModelMappingRow" />
                      <datalist id="source-models-new"><option v-for="m in sourceModelOptions" :key="m" :value="m" /></datalist>
                      <ArrowRight class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                      <Input v-model="newModelMapping.target" class="h-7 flex-1 font-mono text-xs" placeholder="target" :list="targetModelOptions.length ? 'target-models-new' : undefined" @keydown.enter.prevent="addModelMappingRow" />
                      <datalist v-if="targetModelOptions.length" id="target-models-new">
                        <option v-for="m in targetModelOptions" :key="m" :value="m" />
                      </datalist>
                      <Button type="button" variant="outline" size="sm" :disabled="!newModelMapping.source.trim() || !newModelMapping.target.trim()" @click="addModelMappingRow">
                        <Plus class="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
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
                  <button type="button" class="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground" @click="showAdvanced = !showAdvanced">
                    <ChevronDown v-if="!showAdvanced" class="h-3.5 w-3.5" />
                    <ChevronUp v-else class="h-3.5 w-3.5" />
                    {{ tf('console.form.advancedFlags', '高级选项') }}
                  </button>

                  <div v-if="showAdvanced" class="space-y-4">
                    <!-- Vision -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Vision</div>
                      <div class="grid gap-2 md:grid-cols-2">
                        <div class="flex items-center gap-2"><Switch v-model="form.noVision" /><Label class="text-xs">{{ tf('console.form.noVision', '禁用视觉') }}</Label></div>
                        <div class="space-y-1"><Label class="text-[10px]">Vision fallback model</Label><Input v-model="form.visionFallbackModel" class="h-7 text-xs" placeholder="mimo-v2.5" /></div>
                        <div class="space-y-1 md:col-span-2"><Label class="text-[10px]">No vision models（每行一个）</Label><Textarea v-model="form.noVisionModelsText" rows="2" class="font-mono text-xs" /></div>
                      </div>
                    </div>

                    <!-- Reasoning / Thinking -->
                    <div class="space-y-2" v-if="form.serviceType === 'claude' || form.serviceType === 'gemini' || supportsOpenAIAdvanced">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Reasoning / Thinking</div>
                      <div class="grid gap-2 md:grid-cols-2">
                        <div v-if="form.serviceType === 'claude' && channelType !== 'images'" class="flex items-center gap-2"><Switch v-model="form.passbackReasoningContent" /><Label class="text-xs">{{ tf('console.form.passbackReasoning', '回传推理内容') }}</Label></div>
                        <div v-if="form.serviceType === 'claude' && channelType !== 'images'" class="flex items-center gap-2"><Switch v-model="form.passbackThinkingBlocks" /><Label class="text-xs">{{ tf('console.form.passbackThinking', '回传思考块') }}</Label></div>
                        <div v-if="form.serviceType === 'gemini' && ['gemini','messages','chat','responses'].includes(channelType)" class="flex items-center gap-2"><Switch v-model="form.stripThoughtSignature" /><Label class="text-xs">{{ tf('console.form.stripThoughtSignature', '移除思考签名') }}</Label></div>
                        <div v-if="form.serviceType === 'gemini' && ['gemini','messages'].includes(channelType)" class="flex items-center gap-2"><Switch v-model="form.injectDummyThoughtSignature" /><Label class="text-xs">{{ tf('console.form.injectDummySignature', '注入假思考签名') }}</Label></div>
                        <div v-if="supportsOpenAIAdvanced" class="space-y-1"><Label class="text-[10px]">{{ tf('console.form.reasoningParamStyle', 'Reasoning 参数风格') }}</Label><Select v-model="form.reasoningParamStyle"><SelectTrigger class="h-7"><SelectValue /></SelectTrigger><SelectContent><SelectItem v-for="item in reasoningParamStyleOptions" :key="item.value" :value="item.value">{{ item.label }}</SelectItem></SelectContent></Select></div>
                        <div v-if="supportsOpenAIAdvanced" class="space-y-1"><Label class="text-[10px]">{{ tf('console.form.textVerbosity', 'Text verbosity') }}</Label><Select v-model="form.textVerbosity"><SelectTrigger class="h-7"><SelectValue /></SelectTrigger><SelectContent><SelectItem v-for="item in textVerbosityOptions" :key="item.value || 'default'" :value="item.value">{{ item.label }}</SelectItem></SelectContent></Select></div>
                        <div class="flex items-center gap-2"><Switch v-model="form.fastMode" /><Label class="text-xs">{{ tf('console.form.fastMode', '快速模式') }}</Label></div>
                      </div>
                    </div>

                    <!-- Codex / Responses -->
                    <div class="space-y-2" v-if="channelType === 'responses'">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Codex / Responses</div>
                      <div class="grid gap-2 md:grid-cols-2">
                        <div class="flex items-center gap-2"><Switch v-model="form.codexNativeToolPassthrough" /><Label class="text-xs">{{ tf('console.form.codexNativeTools', 'Codex 原生工具透传') }}</Label></div>
                        <div class="flex items-center gap-2"><Switch v-model="form.codexToolCompat" /><Label class="text-xs">{{ tf('console.form.codexCompat', 'Codex 工具兼容') }}</Label></div>
                        <div class="flex items-center gap-2"><Switch v-model="form.stripCodexClientTools" /><Label class="text-xs">{{ tf('console.form.stripCodexTools', '移除 Codex 客户端工具') }}</Label></div>
                      </div>
                    </div>

                    <!-- Compatibility / Normalization -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Compatibility / Normalization</div>
                      <div class="grid gap-2 md:grid-cols-2">
                        <div v-if="form.serviceType === 'claude' && channelType === 'messages'" class="flex items-center gap-2"><Switch v-model="form.stripEmptyTextBlocks" /><Label class="text-xs">{{ tf('console.form.stripEmptyBlocks', '移除空文本块') }}</Label></div>
                        <div v-if="form.serviceType === 'claude' && channelType === 'messages'" class="flex items-center gap-2"><Switch v-model="form.normalizeSystemRoleToTopLevel" /><Label class="text-xs">{{ tf('console.form.normalizeSystem', '规范化系统角色') }}</Label></div>
                        <div v-if="['messages','responses'].includes(channelType)" class="flex items-center gap-2"><Switch v-model="form.normalizeMetadataUserId" /><Label class="text-xs">{{ tf('console.form.normalizeUserId', '规范化用户 ID') }}</Label></div>
                        <div v-if="channelType === 'chat' || (channelType === 'responses' && form.serviceType === 'openai')" class="flex items-center gap-2"><Switch v-model="form.normalizeNonstandardChatRoles" /><Label class="text-xs">{{ tf('console.form.normalizeChatRoles', '规范化 Chat 角色') }}</Label></div>
                      </div>
                    </div>

                    <!-- Runtime / Transport -->
                    <div class="space-y-2">
                      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Runtime</div>
                      <div class="grid gap-2 md:grid-cols-3">
                        <div class="flex items-center gap-2"><Switch v-model="form.lowQuality" /><Label class="text-xs">{{ tf('console.form.lowQuality', '低质量标记') }}</Label></div>
                        <div class="flex items-center gap-2"><Switch v-model="form.autoBlacklistBalance" /><Label class="text-xs">{{ tf('console.form.autoBlacklist', '自动黑名单') }}</Label></div>
                        <div class="flex items-center gap-2"><Switch v-model="form.insecureSkipVerify" /><Label class="text-xs">{{ tf('console.form.insecureSkipVerify', '跳过 TLS 验证') }}</Label></div>
                        <div class="space-y-1"><Label class="text-[10px]">{{ tf('console.form.proxyUrl', '代理 URL') }}</Label><Input v-model="form.proxyUrl" class="h-7 text-xs" placeholder="socks5://..." /></div>
                        <div class="space-y-1"><Label class="text-[10px]">{{ tf('console.form.routePrefix', '路由前缀') }}</Label><Input v-model="form.routePrefix" class="h-7 text-xs" placeholder="kimi" /></div>
                        <div class="space-y-1"><Label class="text-[10px]">{{ tf('console.form.requestTimeoutMs', '请求超时（ms）') }}</Label><Input v-model="form.requestTimeoutMs" type="number" class="h-7 text-xs" placeholder="60000" :class="{ 'border-destructive': errors.requestTimeoutMs }" /><p v-if="errors.requestTimeoutMs" class="text-[10px] text-destructive">{{ errors.requestTimeoutMs }}</p></div>
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
              {{ tf('common.cancel', '取消') }} <span class="ml-1.5 text-xs opacity-60">Esc</span>
            </Button>
            <Button type="button" :disabled="!isValid || saving" @click="handleSubmit">
              <Loader2 v-if="saving" class="mr-2 h-4 w-4 animate-spin" />
              {{ isEditMode
                ? tf('console.form.save', '保存')
                : tf('console.form.create', '创建')
              }}
              <span class="ml-1.5 text-xs opacity-60">{{ isEditMode ? 'Enter' : isMac ? '⌘ Enter' : 'Ctrl+Enter' }}</span>
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
