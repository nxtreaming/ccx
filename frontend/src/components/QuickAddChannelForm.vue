<template>
  <div class="quick-add-form d-flex flex-column ga-4">
    <!-- Provider 选择（模板化添加：选 provider + 输 key，系统自动判别 plan/baseURL） -->
    <div class="d-flex align-center ga-2">
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
        :disabled="providerTemplatesLoading"
        :loading="providerTemplatesLoading"
        :menu-props="{ contentClass: 'upstream-select-menu' }"
        class="provider-select"
        @update:model-value="clearSubmitError"
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

    <!-- 名称（仅自定义模式可选） -->
    <v-text-field
      v-if="!isProviderMode"
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
        <div v-for="(url, idx) in baseUrls" :key="'url-' + idx" class="d-flex align-center ga-2">
          <v-text-field
            v-model="baseUrls[idx]"
            :placeholder="t('addChannel.baseUrl') + ' ' + (idx + 1)"
            variant="outlined"
            density="compact"
            hide-details
            class="flex-grow-1"
            @input="validateForm"
          />
          <v-btn v-if="baseUrls.length > 1" size="small" icon variant="text" color="error" @click="removeBaseUrl(idx)">
            <v-icon size="small">mdi-close</v-icon>
          </v-btn>
        </div>
      </div>
    </div>

    <!-- API Key 输入 -->
    <div>
      <div class="d-flex align-center justify-space-between mb-2">
        <div class="d-flex align-center ga-2">
          <v-icon size="16" color="medium-emphasis">mdi-key</v-icon>
          <span class="text-body-2 font-weight-medium">{{ t('addChannel.apiKeys') }}</span>
        </div>
        <v-btn size="small" variant="text" color="primary" @click="addApiKey">
          <v-icon start size="small">mdi-plus</v-icon>
          {{ t('autopilot.quickAdd.addKey') }}
        </v-btn>
      </div>
      <div class="d-flex flex-column ga-2">
        <div v-for="(key, idx) in apiKeys" :key="'key-' + idx" class="d-flex align-center ga-2">
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
              <v-icon size="small" class="cursor-pointer" @click="toggleKeyVisibility(idx)">
                {{ showKeys[idx] ? 'mdi-eye-off' : 'mdi-eye' }}
              </v-icon>
            </template>
          </v-text-field>
          <v-btn v-if="apiKeys.length > 1" size="small" icon variant="text" color="error" @click="removeApiKey(idx)">
            <v-icon size="small">mdi-close</v-icon>
          </v-btn>
        </div>
      </div>
    </div>

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
    <v-alert v-if="submitError" color="error" variant="tonal" density="comfortable" icon="mdi-alert-circle-outline">
      {{ submitError }}
    </v-alert>

    <!-- 创建状态面板；发现任务在创建成功后转入后台 -->
    <v-card v-if="submitting" variant="outlined" class="discovery-card" rounded="lg">
      <v-card-text class="pa-4">
        <div class="d-flex align-center ga-3">
          <v-progress-circular indeterminate size="20" width="2" color="primary" />
          <span class="text-body-2 font-weight-medium">{{ t('autopilot.quickAdd.discovering') }}</span>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from '../i18n'
import api from '../services/api'
import { autoAddChannel, getProviderTemplates } from '../services/autopilot-api'
import type { ProviderTemplate } from '../services/autopilot-api'
import {
  buildQuickAddChannelName,
  defaultQuickAddServiceType,
  normalizeDiscoveredChannelKind,
  supportsQuickAddProtocolDiscovery
} from '../utils/quickAddChannel'

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
const providerTemplatesLoading = ref(true)

// ---- Provider 模板计算属性 ----
// 仅展示与当前渠道类型匹配的 provider；多 route provider 只要包含当前 tab 即可显示。
const availableProviders = computed(() =>
  providerTemplates.value.filter(p => providerSupportsChannel(p, props.channelType))
)

// 选择项：首项为「自定义」（value=''），其余为官方 provider
const providerItems = computed(() => [
  { title: t('autopilot.quickAdd.provider.custom'), value: '' },
  ...availableProviders.value.map(p => ({ title: p.displayName, value: p.providerId }))
])

const selectedProvider = computed(() => availableProviders.value.find(p => p.providerId === providerId.value))

const isProviderMode = computed(() => providerId.value !== '')

const isFormValid = computed(() => {
  const hasKey = apiKeys.value.some(k => k.trim() !== '')
  // provider 模式：baseURL 由后端判定，只需 key
  if (isProviderMode.value) return hasKey
  const hasUrl = baseUrls.value.some(u => u.trim() !== '')
  return hasUrl && hasKey
})

// ---- 方法 ----
function providerSupportsChannel(provider: ProviderTemplate, channelType: ChannelType): boolean {
  if (provider.routes?.some(route => route.channelKind === channelType)) return true
  return !provider.channelKind || provider.channelKind === channelType
}

function clearSubmitError() {
  submitError.value = ''
}

async function loadProviderTemplates() {
  providerTemplatesLoading.value = true
  try {
    providerTemplates.value = await getProviderTemplates()
  } catch {
    try {
      // 预取可能因瞬时认证状态失败；当前表单内受控重试一次。
      providerTemplates.value = await getProviderTemplates()
    } catch (err) {
      console.error('[QuickAdd-Provider] 加载 provider 模板失败:', err)
      providerTemplates.value = []
    }
  } finally {
    providerTemplatesLoading.value = false
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
  const filtered = getFilteredBaseUrls()
  return buildQuickAddChannelName(filtered[0] || '', generateRandomSuffix())
}

async function discoverCustomChannelKind(baseUrls: string[], apiKeys: string[]): Promise<ChannelType> {
  if (!supportsQuickAddProtocolDiscovery(props.channelType)) return props.channelType

  // 不传 channelKind，让真实探测结果决定落入哪个协议渠道，而不是沿用当前页签。
  const discovery = await api.discoverChannelConfig({
    serviceType: defaultQuickAddServiceType(props.channelType),
    baseUrls,
    apiKey: apiKeys[0]
  })
  const discoveredKind = normalizeDiscoveredChannelKind(discovery.recommendation.channelKind)
  if (!discoveredKind) {
    throw new Error(t('autopilot.quickAdd.discoveryFailed'))
  }
  return discoveredKind
}

async function handleSubmit() {
  if (!isFormValid.value || submitting.value) return

  submitting.value = true
  submitError.value = ''

  try {
    const filteredBaseUrls = getFilteredBaseUrls()
    const filteredApiKeys = getFilteredApiKeys()
    const targetChannelType = isProviderMode.value
      ? props.channelType
      : await discoverCustomChannelKind(filteredBaseUrls, filteredApiKeys)
    const result = await autoAddChannel(
      targetChannelType,
      isProviderMode.value
        ? {
            providerId: providerId.value,
            apiKeys: filteredApiKeys
          }
        : {
            name: channelName.value.trim() || getGeneratedName(),
            baseUrls: filteredBaseUrls,
            apiKeys: filteredApiKeys
          }
    )

    const currentChannel = result.channels?.find(ch => ch.channelKind === targetChannelType)
    const currentIndex = currentChannel?.index ?? result.index
    submitting.value = false
    emit('added', currentIndex)
  } catch (err) {
    submitting.value = false
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
  channelName.value = ''
  baseUrls.value = ['']
  apiKeys.value = ['']
  showKeys.value = [false]
  autoManaged.value = true
  submitting.value = false
  submitError.value = ''
}

// ---- 生命周期 ----
onMounted(() => {
  loadProviderTemplates()
})

// 暴露给父组件
defineExpose({ handleSubmit, resetForm, isFormValid, submitting })
</script>

<style scoped>
.quick-add-form {
  min-height: 0;
}

.provider-select {
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
