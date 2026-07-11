<template>
  <div class="quick-add-form d-flex flex-column ga-4">
    <!-- Provider 选择（模板化添加：选 provider + 输 key，系统自动判别 plan/baseURL） -->
    <div v-if="providerItems.length > 1" class="d-flex align-center ga-2">
      <v-icon color="primary" size="20">mdi-shape-outline</v-icon>
      <div class="text-caption text-medium-emphasis flex-shrink-0">
        {{ t('autopilot.quickAdd.provider.label') }}
      </div>
      <v-spacer />
      <v-select
        v-model="providerId"
        :items="providerItems"
        item-title="title"
        item-value="value"
        variant="outlined"
        density="compact"
        hide-details
        :menu-props="{ contentClass: 'upstream-select-menu' }"
        class="service-type-select"
        @update:model-value="onProviderChange"
      />
    </div>

    <!-- Provider 说明 -->
    <v-alert
      v-if="isProviderMode && selectedProvider?.description"
      color="primary"
      variant="tonal"
      density="comfortable"
      icon="mdi-information"
    >
      {{ selectedProvider.description }}
    </v-alert>

    <!-- 服务类型选择（自定义模式；provider 模式下由模板锁定，隐藏） -->
    <div v-if="!isProviderMode" class="d-flex align-center ga-2">
      <v-icon :color="serviceTypeColor" size="20">{{ serviceTypeIcon }}</v-icon>
      <div class="text-caption text-medium-emphasis flex-shrink-0">
        {{ t('channelEditor.basic.serviceType.label') }}
      </div>
      <v-spacer />
      <v-select
        v-model="serviceType"
        :items="serviceTypeItems"
        item-title="title"
        item-value="value"
        variant="outlined"
        density="compact"
        hide-details
        :menu-props="{ contentClass: 'upstream-select-menu' }"
        class="service-type-select"
        @update:model-value="onServiceTypeChange"
      />
    </div>

    <!-- 名称（可选） -->
    <v-text-field
      v-model="channelName"
      :label="t('addChannel.channelName')"
      :placeholder="t('autopilot.quickAdd.namePlaceholder')"
      variant="outlined"
      density="compact"
      hide-details
      prepend-inner-icon="mdi-tag"
    />

    <!-- Base URL 输入（provider 模式下由后端按 key 前缀判定，隐藏） -->
    <div v-if="!isProviderMode">
      <div class="d-flex align-center justify-space-between mb-2">
        <div class="d-flex align-center ga-2">
          <v-icon size="16" color="medium-emphasis">mdi-web</v-icon>
          <span class="text-body-2 font-weight-medium">{{ t('addChannel.baseUrl') }}</span>
        </div>
        <v-btn size="small" variant="text" color="primary" @click="addBaseUrl">
          <v-icon start size="small">mdi-plus</v-icon>
          {{ t('autopilot.quickAdd.addUrl') }}
        </v-btn>
      </div>
      <div class="d-flex flex-column ga-2">
        <div
          v-for="(url, idx) in baseUrls"
          :key="'url-' + idx"
          class="d-flex align-center ga-2"
        >
          <v-text-field
            v-model="baseUrls[idx]"
            :placeholder="t('addChannel.baseUrl') + ' ' + (idx + 1)"
            variant="outlined"
            density="compact"
            hide-details
            class="flex-grow-1"
            @input="validateForm"
          />
          <v-btn
            v-if="baseUrls.length > 1"
            size="small"
            icon
            variant="text"
            color="error"
            @click="removeBaseUrl(idx)"
          >
            <v-icon size="small">mdi-close</v-icon>
          </v-btn>
        </div>
      </div>
    </div>

    <!-- API Key 输入（Copilot 模式下可选） -->
    <div>
      <div class="d-flex align-center justify-space-between mb-2">
        <div class="d-flex align-center ga-2">
          <v-icon size="16" color="medium-emphasis">mdi-key</v-icon>
          <span class="text-body-2 font-weight-medium">{{ t('addChannel.apiKeys') }}</span>
          <v-chip
            v-if="isCopilot"
            size="x-small"
            color="info"
            variant="tonal"
          >
            {{ t('autopilot.quickAdd.optionalForCopilot') }}
          </v-chip>
        </div>
        <v-btn size="small" variant="text" color="primary" @click="addApiKey">
          <v-icon start size="small">mdi-plus</v-icon>
          {{ t('autopilot.quickAdd.addKey') }}
        </v-btn>
      </div>
      <div class="d-flex flex-column ga-2">
        <div
          v-for="(key, idx) in apiKeys"
          :key="'key-' + idx"
          class="d-flex align-center ga-2"
        >
          <v-text-field
            v-model="apiKeys[idx]"
            :placeholder="'sk-...' + (idx + 1)"
            variant="outlined"
            density="compact"
            hide-details
            :type="showKeys[idx] ? 'text' : 'password'"
            class="flex-grow-1"
            @input="validateForm"
          >
            <template #append-inner>
              <v-icon
                size="small"
                class="cursor-pointer"
                @click="toggleKeyVisibility(idx)"
              >
                {{ showKeys[idx] ? 'mdi-eye-off' : 'mdi-eye' }}
              </v-icon>
            </template>
          </v-text-field>
          <v-btn
            v-if="apiKeys.length > 1"
            size="small"
            icon
            variant="text"
            color="error"
            @click="removeApiKey(idx)"
          >
            <v-icon size="small">mdi-close</v-icon>
          </v-btn>
        </div>
      </div>
    </div>

    <!-- Copilot OAuth 提示 -->
    <v-alert
      v-if="isCopilot"
      color="info"
      variant="tonal"
      density="comfortable"
    >
      {{ t('copilotOAuth.quickAddHint') }}
    </v-alert>

    <!-- 自动托管开关 -->
    <v-card variant="outlined" class="auto-managed-card" rounded="lg">
      <v-card-text class="pa-3">
        <div class="d-flex align-center ga-3">
          <v-checkbox
            v-model="autoManaged"
            color="primary"
            density="compact"
            hide-details
            class="ma-0 pa-0 flex-shrink-0"
          />
          <div class="flex-grow-1">
            <div class="text-body-2 font-weight-medium">
              {{ t('autopilot.quickAdd.autoManaged') }}
            </div>
            <div class="text-caption text-medium-emphasis">
              {{ t('autopilot.quickAdd.autoManagedHint') }}
            </div>
          </div>
          <v-icon color="primary" size="24">mdi-auto-fix</v-icon>
        </div>
      </v-card-text>
    </v-card>

    <!-- 提交错误（provider 模式 key 无效等） -->
    <v-alert
      v-if="submitError"
      color="error"
      variant="tonal"
      density="comfortable"
      icon="mdi-alert-circle-outline"
    >
      {{ submitError }}
    </v-alert>

    <!-- 发现状态面板 -->
    <v-card v-if="submitting" variant="outlined" class="discovery-card" rounded="lg">
      <v-card-text class="pa-4">
        <div class="d-flex align-center ga-3 mb-3">
          <v-progress-circular
            v-if="autoStatus.status === 'discovering'"
            indeterminate
            size="20"
            width="2"
            color="primary"
          />
          <v-icon v-else-if="autoStatus.status === 'done'" color="success" size="20">mdi-check-circle</v-icon>
          <v-icon v-else-if="autoStatus.status === 'failed'" color="error" size="20">mdi-alert-circle</v-icon>
          <span class="text-body-2 font-weight-medium">{{ statusText }}</span>
        </div>

        <template v-if="autoStatus.endpoints.length > 0">
          <v-divider class="mb-3" />
          <div class="d-flex flex-column ga-2">
            <div
              v-for="(ep, idx) in autoStatus.endpoints"
              :key="idx"
              class="d-flex align-center ga-2 text-caption"
            >
              <v-icon size="14" :color="ep.protocolOk ? 'success' : 'error'">
                {{ ep.protocolOk ? 'mdi-check-circle' : 'mdi-close-circle' }}
              </v-icon>
              <code class="text-caption">{{ ep.keyMask }}</code>
              <v-spacer />
              <span v-if="ep.modelsCount > 0" class="text-success">
                {{ ep.modelsCount }} {{ t('autopilot.quickAdd.models') }}
              </span>
            </div>
          </div>
        </template>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from '../i18n'
import {
  autoAddChannel,
  getChannelAutoStatus,
  getProviderTemplates,
} from '../services/autopilot-api'
import type { ProviderTemplate } from '../services/autopilot-api'

type ServiceType = 'openai' | 'gemini' | 'claude' | 'responses' | 'copilot'
type ChannelType = 'messages' | 'chat' | 'responses' | 'gemini' | 'images' | 'vectors'

interface Props {
  channelType: ChannelType
}

const props = defineProps<Props>()

const emit = defineEmits<{
  added: [channelId: number]
  close: []
}>()

const { t } = useI18n()

// ---- 表单状态 ----
const serviceType = ref<ServiceType>(getDefaultServiceType())
const channelName = ref('')
const baseUrls = ref<string[]>([''])
const apiKeys = ref<string[]>([''])
const showKeys = ref<boolean[]>([false])
const autoManaged = ref(true)
const submitting = ref(false)
const submitError = ref('')

// ---- Provider 模板状态 ----
// '' 表示自定义模式（手填 baseURL）；非空表示选中某官方 provider（模板化添加）
const providerId = ref('')
const providerTemplates = ref<ProviderTemplate[]>([])

// ---- 发现状态 ----
import type { AutoEndpointStatus } from '../services/autopilot-api'

const autoStatus = reactive({
  status: '' as 'discovering' | 'done' | 'failed' | '',
  endpoints: [] as AutoEndpointStatus[],
})

let pollTimer: ReturnType<typeof setInterval> | null = null

// ---- 常量 ----
const serviceTypeItems = computed(() => {
  if (props.channelType === 'images' || props.channelType === 'vectors') {
    return [{ title: 'OpenAI Images', value: 'openai' as const }]
  }
  return [
    { title: 'Claude', value: 'claude' as const },
    { title: 'OpenAI Chat', value: 'openai' as const },
    { title: 'Gemini', value: 'gemini' as const },
    { title: 'Responses (Codex)', value: 'responses' as const },
    { title: 'GitHub Copilot', value: 'copilot' as const },
  ]
})

const serviceTypeIcon = computed(() => {
  const map: Record<ServiceType, string> = {
    openai: 'mdi-robot',
    claude: 'mdi-message-processing',
    gemini: 'mdi-diamond-stone',
    responses: 'mdi-rocket-launch',
    copilot: 'mdi-code-braces',
  }
  return map[serviceType.value] || 'mdi-api'
})

const serviceTypeColor = computed(() => {
  const map: Record<ServiceType, string> = {
    openai: 'info',
    claude: 'orange',
    gemini: 'purple',
    responses: 'success',
    copilot: 'grey-darken-2',
  }
  return map[serviceType.value] || 'primary'
})

const isCopilot = computed(() => serviceType.value === 'copilot')

// ---- Provider 模板计算属性 ----
// 仅展示与当前渠道类型匹配的 provider；多 route provider 只要包含当前 tab 即可显示。
const availableProviders = computed(() =>
  providerTemplates.value.filter(p => providerSupportsChannel(p, props.channelType)),
)

// 选择项：首项为「自定义」（value=''），其余为官方 provider
const providerItems = computed(() => [
  { title: t('autopilot.quickAdd.provider.custom'), value: '' },
  ...availableProviders.value.map(p => ({ title: p.displayName, value: p.providerId })),
])

const selectedProvider = computed(() =>
  availableProviders.value.find(p => p.providerId === providerId.value),
)

const isProviderMode = computed(() => providerId.value !== '')

const isFormValid = computed(() => {
  const hasKey = apiKeys.value.some(k => k.trim() !== '')
  // provider 模式：baseURL 由后端判定，只需 key
  if (isProviderMode.value) return hasKey
  const hasUrl = baseUrls.value.some(u => u.trim() !== '')
  if (isCopilot.value) return hasUrl
  return hasUrl && hasKey
})

const statusText = computed(() => {
  switch (autoStatus.status) {
    case 'discovering': return t('autopilot.quickAdd.discovering')
    case 'done': return t('autopilot.quickAdd.discoveryDone')
    case 'failed': return t('autopilot.quickAdd.discoveryFailed')
    default: return ''
  }
})

// ---- 方法 ----
function providerRouteForChannel(provider: ProviderTemplate | undefined, channelType: ChannelType) {
  if (!provider) return undefined
  const route = provider.routes?.find(item => item.channelKind === channelType)
  if (route) return route
  if (!provider.channelKind || provider.channelKind === channelType) {
    return {
      channelKind: provider.channelKind,
      serviceType: provider.serviceType,
    }
  }
  return undefined
}

function providerSupportsChannel(provider: ProviderTemplate, channelType: ChannelType): boolean {
  return providerRouteForChannel(provider, channelType) !== undefined
}

function getDefaultServiceType(): ServiceType {
  if (props.channelType === 'chat') return 'openai'
  if (props.channelType === 'gemini') return 'gemini'
  if (props.channelType === 'responses') return 'responses'
  if (props.channelType === 'images' || props.channelType === 'vectors') return 'openai'
  return 'claude'
}

function onServiceTypeChange() {
  if (isCopilot.value && baseUrls.value.length === 1 && !baseUrls.value[0]) {
    baseUrls.value = ['https://api.githubcopilot.com']
  }
}

// 切换 provider：选中官方 provider 时锁定其 serviceType（baseURL 由后端判定）
function onProviderChange() {
  submitError.value = ''
  const route = providerRouteForChannel(selectedProvider.value, props.channelType)
  if (route?.serviceType) {
    serviceType.value = route.serviceType as ServiceType
  }
}

async function loadProviderTemplates() {
  try {
    providerTemplates.value = await getProviderTemplates()
  } catch (err) {
    console.error('[QuickAdd-Provider] 加载 provider 模板失败:', err)
    providerTemplates.value = []
  }
}

function addBaseUrl() {
  baseUrls.value.push('')
}

function removeBaseUrl(idx: number) {
  baseUrls.value.splice(idx, 1)
}

function addApiKey() {
  apiKeys.value.push('')
  showKeys.value.push(false)
}

function removeApiKey(idx: number) {
  apiKeys.value.splice(idx, 1)
  showKeys.value.splice(idx, 1)
}

function toggleKeyVisibility(idx: number) {
  showKeys.value[idx] = !showKeys.value[idx]
}

function validateForm() {
  // 触发响应式更新
}

function getFilteredBaseUrls(): string[] {
  return baseUrls.value.filter(u => u.trim() !== '')
}

function getFilteredApiKeys(): string[] {
  return apiKeys.value.filter(k => k.trim() !== '')
}

function generateRandomSuffix(length = 6): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

function getGeneratedName(): string {
  // provider 模式：baseURL 由后端判定，用 providerId 生成名称
  if (isProviderMode.value) {
    return `${providerId.value}-${generateRandomSuffix()}`
  }
  const filtered = getFilteredBaseUrls()
  const first = filtered[0] || ''
  try {
    const host = new URL(first).hostname.replace(/\./g, '-')
    return `${host}-${generateRandomSuffix()}`
  } catch {
    return `channel-${generateRandomSuffix()}`
  }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function startPolling(kind: ChannelType, channelId: number) {
  let attempts = 0
  const maxAttempts = 60 // 最多 5 分钟 (5s * 60)

  pollTimer = setInterval(async () => {
    attempts++
    if (attempts > maxAttempts) {
      stopPolling()
      autoStatus.status = 'failed'
      submitting.value = false
      return
    }

    try {
      const result = await getChannelAutoStatus(kind, channelId)
      const discovery = result.discovery
      if (!discovery) return // 尚未触发发现
      if (discovery.status === 'done') {
        stopPolling()
        autoStatus.status = 'done'
        autoStatus.endpoints = discovery.endpoints || []
        submitting.value = false
        emit('added', channelId)
      } else if (discovery.status === 'failed') {
        stopPolling()
        autoStatus.status = 'failed'
        autoStatus.endpoints = discovery.endpoints || []
        submitting.value = false
      } else {
        autoStatus.endpoints = discovery.endpoints || []
      }
    } catch {
      // 忽略轮询错误，继续尝试
    }
  }, 5000)
}

async function handleSubmit() {
  if (!isFormValid.value || submitting.value) return

  submitting.value = true
  submitError.value = ''
  autoStatus.status = 'discovering'
  autoStatus.endpoints = []

  const finalName = channelName.value.trim() || getGeneratedName()

  try {
    const result = await autoAddChannel(
      props.channelType,
      isProviderMode.value
        ? {
            name: finalName,
            providerId: providerId.value,
            apiKeys: getFilteredApiKeys(),
          }
        : {
            name: finalName,
            baseUrls: getFilteredBaseUrls(),
            apiKeys: getFilteredApiKeys(),
          },
    )

    const currentChannel = result.channels?.find(ch => ch.channelKind === props.channelType)
    const currentIndex = currentChannel?.index ?? result.index
    const discoveryStarted = currentChannel?.discoveryStarted ?? result.discoveryStarted

    if (discoveryStarted) {
      startPolling(props.channelType, currentIndex)
    } else {
      autoStatus.status = ''
      submitting.value = false
      emit('added', currentIndex)
    }
  } catch (err) {
    stopPolling()
    submitting.value = false
    autoStatus.status = 'failed'
    // provider 模式下后端会对无效 key 返回 400（含明确原因），提取给用户
    submitError.value = extractErrorMessage(err)
    console.error('[QuickAdd-Submit] 自动添加渠道失败:', err)
  }
}

// 从 auto-add 抛出的 Error 中提取后端返回的错误正文
function extractErrorMessage(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err)
  // autopilot-api 抛出格式：`auto-add failed (400): {"error":"..."}`
  const jsonStart = raw.indexOf('{')
  if (jsonStart >= 0) {
    try {
      const parsed = JSON.parse(raw.slice(jsonStart))
      if (parsed?.error) return String(parsed.error)
    } catch {
      // 非 JSON 正文，回退到原始消息
    }
  }
  return raw
}

function resetForm() {
  providerId.value = ''
  serviceType.value = getDefaultServiceType()
  channelName.value = ''
  baseUrls.value = ['']
  apiKeys.value = ['']
  showKeys.value = [false]
  autoManaged.value = true
  submitting.value = false
  submitError.value = ''
  autoStatus.status = ''
  autoStatus.endpoints = []
  stopPolling()
}

// ---- 生命周期 ----
onMounted(() => {
  loadProviderTemplates()
})

onUnmounted(() => {
  stopPolling()
})

// 暴露给父组件
defineExpose({ handleSubmit, resetForm, isFormValid, submitting })
</script>

<style scoped>
.quick-add-form {
  min-height: 0;
}

.service-type-select {
  min-width: 200px;
  max-width: 260px;
}

.auto-managed-card {
  border-color: rgba(var(--v-theme-primary), 0.3);
  background: rgba(var(--v-theme-primary), 0.03);
}

.discovery-card {
  border-color: rgba(var(--v-theme-outline), 0.32);
}
</style>
