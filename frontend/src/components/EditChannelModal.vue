<template>
  <v-dialog :model-value="show" max-width="1200" persistent @update:model-value="$emit('update:show', $event)">
    <v-card rounded="lg" class="add-channel-dialog">
      <!-- 头部 -->
      <AddChannelHeader
        :is-editing="isEditing"
        :channel-type="props.channelType"
        :no-vision="form.noVision"
        :header-classes="headerClasses"
        :avatar-color="avatarColor"
        :header-icon-style="headerIconStyle"
        :subtitle-classes="subtitleClasses"
        :edit-title="t('addChannel.editTitle')"
        :create-title="t('addChannel.createTitle')"
        :edit-subtitle="t('addChannel.editSubtitle')"
        :create-subtitle="t('addChannel.quickSubtitle')"
        :test-capability-label="t('addChannel.testCapability')"
        :vision-tooltip="form.noVision ? t('channelCard.noVision') : t('channelCard.hasVision')"
        @toggle-no-vision="form.noVision = !form.noVision"
        @test-capability="handleTestCapability"
      />

      <!-- 主体内容 -->
      <v-card-text class="pa-0" style="height: 600px;">
        <!-- 左侧导航 + 右侧面板 -->
        <div class="content-row" style="height: 100%;">
          <!-- 左侧垂直导航 -->
          <AddChannelSidebarNav
            :title="t('addChannel.outline')"
            :sections="sections"
            :active-section="activeSection"
            @navigate="scrollToSection"
          />

          <!-- 右侧内容面板 -->
          <v-form ref="formRef" class="content-area" @submit.prevent="handleSubmit">
            <!-- 基本信息 -->
            <section :ref="(el: any) => setSectionRef('basic', el)" data-section-id="basic" class="pa-6 scroll-mt-4">
              <BasicInfoSection
                :form="form"
                :base-urls-text="baseUrlsText"
                :expected-request-urls="expectedRequestUrls"
                :base-url-has-error="baseUrlHasError"
                :service-type-options="serviceTypeOptions"
                :errors="errors"
                :rules="rules"
                @update:form="updateForm"
                @update:base-urls-text="baseUrlsText = $event"
                @menu-update="onMenuUpdate"
              />
            </section>

            <!-- 模型重定向（模型映射 + Vision 回退 + 模型过滤） -->
            <section :ref="(el: any) => setSectionRef('redirect', el)" data-section-id="redirect" class="pa-6 scroll-mt-4">
              <ModelMappingSection
                v-if="form.serviceType"
                :mappingRows="modelMappingRows"
                :sourceModelOptions="sourceModelOptions"
                :targetModelOptions="targetModelOptions"
                :fetchingModels="fetchingModels"
                :sourceMappingError="sourceMappingError"
                :fetchModelsError="fetchModelsError"
                :modelMappingHint="modelMappingHint"
                :targetModelPlaceholder="targetModelPlaceholder"
                :showModelMappingPresets="showModelMappingPresets"
                :showMessagesOpenAIChannelPresets="showMessagesOpenAIChannelPresets"
                :showClaudeChannelPresets="showClaudeChannelPresets"
                :showCodexResponsesChannelPresets="showCodexResponsesChannelPresets"
                :supportsReasoningMappingOptions="supportsReasoningMappingOptions"
                :reasoningEffortOptions="reasoningEffortOptions"
                @update:mappingRows="modelMappingRows = ($event as any)"
                @sync-upstream="syncUpstreamModels"
                @apply-preset="applyPreset"
                @menu-update="onMenuUpdate"
              >
                <template #vision-fallback>
                  <div v-if="hasNoVisionRows" class="mt-6">
                    <v-row dense>
                      <v-col cols="12" :md="supportsReasoningMappingOptions ? 8 : 12">
                        <v-combobox
                          v-model="form.visionFallbackModel"
                          :label="t('addChannel.visionFallbackLabel')"
                          :placeholder="t('addChannel.visionFallbackPlaceholder')"
                          :hint="t('addChannel.visionFallbackHint')"
                          :items="targetModelOptions"
                          prepend-inner-icon="mdi-eye"
                          persistent-hint
                          clearable
                          variant="outlined"
                          density="comfortable"
                          eager
                          @focus="ensureTargetModelsLoaded"
                          @update:menu="onMenuUpdate"
                        />
                      </v-col>
                      <v-col v-if="supportsReasoningMappingOptions" cols="12" md="4">
                        <v-select
                          v-model="form.visionFallbackReasoningEffort"
                          :label="t('addChannel.visionFallbackReasoningLabel')"
                          :items="reasoningEffortOptions"
                          variant="outlined"
                          density="comfortable"
                          clearable
                          persistent-hint
                          :hint="t('addChannel.visionFallbackReasoningHint')"
                          eager
                          @update:menu="onMenuUpdate"
                        />
                      </v-col>
                    </v-row>
                  </div>
                </template>
              </ModelMappingSection>

              <!-- 模型过滤 -->
              <div class="mt-4">
                <SupportedModelsFilter
                  :model-value="form.supportedModels"
                  :error="supportedModelsError"
                  :common-filters="commonSupportedModelFilters"
                  :selected-filters="Array.from(selectedSupportedModelSet)"
                  @update:model-value="handleSupportedModelsChange($event as any)"
                  @append-filter="appendSupportedModelFilter"
                  @menu-update="onMenuUpdate"
                />
              </div>

              <div v-if="props.channelType !== 'images'" class="mt-6">
                <div class="text-subtitle-2 mb-3">{{ t('addChannel.contextCapabilityTitle') }}</div>
                <v-row dense>
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="form.defaultContextWindowTokens"
                      :label="t('addChannel.defaultContextWindowLabel')"
                      :hint="t('addChannel.defaultContextWindowHint')"
                      type="number"
                      min="0"
                      prepend-inner-icon="mdi-database"
                      persistent-hint
                      variant="outlined"
                      density="comfortable"
                    />
                  </v-col>
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="form.defaultMaxOutputTokens"
                      :label="t('addChannel.defaultMaxOutputLabel')"
                      :hint="t('addChannel.defaultMaxOutputHint')"
                      type="number"
                      min="0"
                      prepend-inner-icon="mdi-text"
                      persistent-hint
                      variant="outlined"
                      density="comfortable"
                    />
                  </v-col>
                </v-row>

                <v-textarea
                  v-model="form.modelCapabilitiesText"
                  :label="t('addChannel.modelCapabilitiesJsonLabel')"
                  :placeholder="modelCapabilitiesJsonPlaceholder"
                  :hint="t('addChannel.modelCapabilitiesJsonHint')"
                  :error-messages="modelCapabilitiesError ? [modelCapabilitiesError] : []"
                  prepend-inner-icon="mdi-code-braces"
                  persistent-hint
                  auto-grow
                  rows="4"
                  variant="outlined"
                  density="comfortable"
                  class="mt-2"
                />

                <v-switch
                  v-model="form.allowUnknownContext"
                  :label="t('addChannel.allowUnknownContextLabel')"
                  :hint="t('addChannel.allowUnknownContextHint')"
                  color="primary"
                  inset
                  persistent-hint
                  hide-details="auto"
                  class="mt-2"
                />
              </div>
            </section>

            <!-- 身份认证 -->
            <section :ref="(el: any) => setSectionRef('auth', el)" data-section-id="auth" class="pa-6 scroll-mt-4">
              <ApiKeyManagementSection
                :api-keys="form.apiKeys"
                :disabled-keys="disabledKeys"
                :key-models-status="keyModelsStatus"
                :is-editing="isEditing"
                :restoring-key="restoringKey"
                @update:api-keys="form.apiKeys = $event"
                @restore-key="restoreDisabledKey"
              />
            </section>

            <!-- 高级选项 -->
            <section :ref="(el: any) => setSectionRef('advanced', el)" data-section-id="advanced" class="pa-6 scroll-mt-4">
              <AdvancedOptionsSection
                :form="form"
                :channelType="props.channelType"
                :supportsChatRoleNormalization="supportsChatRoleNormalization"
                :supportsOpenAIAdvancedOptions="supportsOpenAIAdvancedOptions"
                :reasoningParamStyleOptions="reasoningParamStyleOptions"
                :textVerbosityOptions="textVerbosityOptions"
                :rules="rules"
                @update:form="updateForm"
                @menu-update="onMenuUpdate"
              />
            </section>

            <!-- 自定义参数（自定义请求头 + 流式超时） -->
            <section :ref="(el: any) => setSectionRef('custom', el)" data-section-id="custom" class="pa-6 scroll-mt-4">
              <CustomHeadersSection
                :headers="customHeadersArray"
                @update:headers="updateCustomHeaders"
              />

              <div class="mt-6">
                <StreamTimeoutSection
                  :selected-strategy="selectedStreamTimeoutStrategy"
                  :first-content-enabled="form.streamFirstContentTimeoutEnabled"
                  :first-content-ms="form.streamFirstContentTimeoutMs"
                  :inactivity-enabled="form.streamInactivityTimeoutEnabled"
                  :inactivity-ms="form.streamInactivityTimeoutMs"
                  :tool-call-idle-enabled="form.streamToolCallIdleTimeoutEnabled"
                  :tool-call-idle-ms="form.streamToolCallIdleTimeoutMs"
                  @apply-strategy="applyStreamTimeoutStrategy"
                  @update:first-content-ms="form.streamFirstContentTimeoutMs = $event"
                  @update:inactivity-ms="form.streamInactivityTimeoutMs = $event"
                  @update:tool-call-idle-ms="form.streamToolCallIdleTimeoutMs = $event"
                />
              </div>
            </section>
          </v-form>
        </div>
      </v-card-text>

      <!-- 底部按钮 -->
      <v-card-actions class="pa-6 pt-2">
        <v-spacer />
        <v-btn variant="outlined" @click="handleCancel">
          {{ t('app.actions.cancel') }}<span class="shortcut-hint ml-2 text-xs opacity-50">Esc</span>
        </v-btn>
        <v-btn
          color="primary"
          variant="elevated"
          :disabled="!isFormValid"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ t('app.actions.save') }}<span class="shortcut-hint ml-2 text-xs opacity-50">{{ isMac ? '⌘Enter' : 'Ctrl+Enter' }}</span>
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useTheme } from 'vuetify'
import type { Channel } from '../services/api'
import { ApiService, ApiError } from '../services/api'
import { useChannelStore } from '../stores/channel'
import { useDialogStore } from '../stores/dialog'
import { buildExpectedRequestUrls } from '../utils/expectedRequestUrls'
import { supportsAdvancedChannelOptions, supportsReasoningMapping } from '../utils/channelAdvancedOptions'
import { buildChannelPayload, normalizeSelectableString, parseModelCapabilitiesText } from '../utils/channelPayload'
import { maskApiKey } from '../utils/apiKeyMask'
import {
  resolveChannelWatcherAction,
  syncBaseUrlsFormState,
  filterValidSupportedModelPatterns,
  parseSupportedModelInput
} from '../utils/add-channel-modal-state'
import { streamTimeoutPresets } from '../utils/streamTimeoutPresets'
import { useI18n } from '../i18n'

// 子组件导入
import AddChannelHeader from './edit-channel/AddChannelHeader.vue'
import AddChannelSidebarNav from './edit-channel/AddChannelSidebarNav.vue'
import BasicInfoSection from './edit-channel/BasicInfoSection.vue'
import ApiKeyManagementSection from './edit-channel/ApiKeyManagementSection.vue'
import ModelMappingSection from './edit-channel/ModelMappingSection.vue'
import SupportedModelsFilter from './edit-channel/SupportedModelsFilter.vue'
import CustomHeadersSection from './edit-channel/CustomHeadersSection.vue'
import StreamTimeoutSection from './edit-channel/StreamTimeoutSection.vue'
import AdvancedOptionsSection from './edit-channel/AdvancedOptionsSection.vue'

interface Props {
  show: boolean
  channel?: Channel | null
  channelType?: 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
}

const props = withDefaults(defineProps<Props>(), {
  channelType: 'messages'
})

const emit = defineEmits<{
  'update:show': [value: boolean]
  save: [channel: Omit<Channel, 'index' | 'latency' | 'status'>, options?: { isQuickAdd?: boolean; triggerCapabilityTest?: boolean }]
  testCapability: [channelId: number]
  error: [message: string]
}>()
const { t } = useI18n()
const apiService = new ApiService()
const channelStore = useChannelStore()
const dialogStore = useDialogStore()

// 主题
const theme = useTheme()

// 表单引用
const formRef = ref()


const defaultServiceTypeValueFallback = (): 'openai' | 'gemini' | 'claude' | 'responses' => {
  if (props.channelType === 'chat') return 'openai'
  if (props.channelType === 'gemini') return 'gemini'
  if (props.channelType === 'responses') return 'responses'
  return 'claude'
}

// 详细表单预期请求 URL 预览（防止输入时抖动）
const formBaseUrlPreview = ref('')
let formBaseUrlPreviewTimer: number | null = null

// 垂直导航激活 Section
const activeSection = ref('basic')
const sectionRefs = ref<Record<string, HTMLElement | null>>({})
let scrollRoot: Element | null = null
let scrollHandler: (() => void) | null = null

function detachScrollListener() {
  if (scrollRoot && scrollHandler) {
    scrollRoot.removeEventListener('scroll', scrollHandler)
  }
  scrollRoot = null
  scrollHandler = null
}

// 导航 section 定义（与桌面端保持一致）
const sections = [
  { id: 'basic', label: t('channelEditor.nav.basic') },
  { id: 'redirect', label: t('channelEditor.nav.redirect') },
  { id: 'auth', label: t('channelEditor.nav.auth') },
  { id: 'advanced', label: t('channelEditor.nav.advanced') },
  { id: 'custom', label: t('channelEditor.nav.custom') },
]

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
  let current = sections[0]?.id || 'basic'

  for (const s of sections) {
    const el = sectionRefs.value[s.id]
    if (!el) continue
    const top = el.getBoundingClientRect().top - rootTop
    if (top <= 120) {
      current = s.id
    } else {
      break
    }
  }

  activeSection.value = current
}

// 检测 baseUrl 是否有验证错误
const baseUrlHasError = computed(() => {
  const value = form.baseUrl
  if (!value) return true
  try {
    new URL(value)
    return false
  } catch {
    return true
  }
})

// Workaround: Vuetify v-select menu 在 v-dialog 内首次打开时位置计算错误
// 通过 dispatch resize 强制重新计算菜单位置
const onMenuUpdate = (open: boolean) => {
  if (open) {
    setTimeout(() => window.dispatchEvent(new Event('resize')), 50)
  }
}



// 服务类型选项 - 根据入口接口类型动态调整可用选项
const serviceTypeOptions = computed(() => {
  // 全部4种上游服务类型
  const allOptions = [
    { title: 'OpenAI Chat', value: 'openai' },
    { title: 'Claude', value: 'claude' },
    { title: 'Gemini', value: 'gemini' },
    { title: 'Responses (Codex)', value: 'responses' }
  ]

  // 根据入口接口类型调整排序（原生/默认类型排第一）
  const reorder = (options: typeof allOptions, first: string) => {
    const firstOption = options.find(o => o.value === first)
    const rest = options.filter(o => o.value !== first)
    return firstOption ? [firstOption, ...rest] : options
  }

  switch (props.channelType) {
    case 'messages':
      return reorder(allOptions, 'claude')
    case 'chat':
      // OpenAI Chat API 入口，OpenAI 原生排第一
      return reorder(allOptions, 'openai')
    case 'responses':
      // Responses API 入口，Responses 原生排第一
      return reorder(allOptions, 'responses')
    case 'images':
      return [{ title: 'OpenAI Images', value: 'openai' }]
    case 'gemini':
      // Gemini API 入口，Gemini 原生排第一
      return reorder(allOptions, 'gemini')
    default:
      return allOptions
  }
})

// 全部源模型选项 - 根据渠道类型动态显示
const allSourceModelOptions = computed(() => {
  if (props.channelType === 'chat') {
    // OpenAI Chat Completions 常用模型
    return [
      { title: 'codex', value: 'codex' },
      { title: 'gpt', value: 'gpt' },
      { title: 'mini', value: 'mini' },
      { title: 'gpt-5', value: 'gpt-5' },
      { title: 'gpt-5.5', value: 'gpt-5.5' },
      { title: 'gpt-5.4', value: 'gpt-5.4' },
      { title: 'gpt-5.4-mini', value: 'gpt-5.4-mini' },
    ]
  }
  if (props.channelType === 'images') {
    return [
      { title: 'gpt-image-2', value: 'gpt-image-2' },
      { title: 'gpt-image-1', value: 'gpt-image-1' },
      { title: 'dall-e-3', value: 'dall-e-3' },
      { title: 'dall-e-2', value: 'dall-e-2' }
    ]
  }
  if (props.channelType === 'gemini') {
    // Gemini API 常用模型别名
    return [
      { title: 'gemini-3.5-flash', value: 'gemini-3.5-flash' },
      { title: 'gemini-3.1-pro-preview', value: 'gemini-3.1-pro-preview' },
      { title: 'gemini-3-pro-preview', value: 'gemini-3-pro-preview' },
      { title: 'gemini-3-flash-preview', value: 'gemini-3-flash-preview' },
      { title: 'gemini-3.1-flash-lite', value: 'gemini-3.1-flash-lite' },
      { title: 'gemini-2.5-pro', value: 'gemini-2.5-pro' },
      { title: 'gemini-2.5-flash', value: 'gemini-2.5-flash' },
      { title: 'gemini-2.5-flash-lite', value: 'gemini-2.5-flash-lite' },
      { title: 'gemini-2', value: 'gemini-2' }
    ]
  }
  if (props.channelType === 'responses') {
    // Responses API (Codex) 常用模型名称
    return [
      { title: 'codex', value: 'codex' },
      { title: 'codex-auto-review', value: 'codex-auto-review' },
      { title: 'gpt-5', value: 'gpt-5' },
      { title: 'gpt', value: 'gpt' },
      { title: 'mini', value: 'mini' },
      { title: 'gpt-5.5', value: 'gpt-5.5' },
      { title: 'gpt-5.4', value: 'gpt-5.4' },
      { title: 'gpt-5.4-mini', value: 'gpt-5.4-mini' },
    ]
  } else {
    // Messages API (Claude) 常用模型别名
    return [
      { title: 'fable', value: 'fable' },
      { title: 'opus', value: 'opus' },
      { title: 'sonnet', value: 'sonnet' },
      { title: 'haiku', value: 'haiku' }
    ]
  }
})

// 可选的源模型选项 - 过滤掉已配置的模型
const sourceModelOptions = computed(() => {
  const configuredModels = Object.keys(form.modelMapping)
  return allSourceModelOptions.value.filter(opt => !configuredModels.includes(opt.value))
})

// 模型重定向的示例文本 - 根据渠道类型动态显示
const modelMappingHint = computed(() => {
  if (props.channelType === 'chat') {
    return t('addChannel.modelMappingHintChat')
  }
  if (props.channelType === 'images') {
    return t('addChannel.modelMappingHintChat')
  }
  if (props.channelType === 'gemini') {
    return t('addChannel.modelMappingHintGemini')
  }
  if (props.channelType === 'responses') {
    return t('addChannel.modelMappingHintResponses')
  } else {
    return t('addChannel.modelMappingHintMessages')
  }
})

const targetModelPlaceholder = computed(() => {
  if (props.channelType === 'chat') {
    return t('addChannel.targetModelPlaceholderChat')
  }
  if (props.channelType === 'images') {
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

const reasoningEffortOptions = [
  { title: t('addChannel.reasoningDefault'), value: '' },
  { title: 'None', value: 'none' },
  { title: 'Low', value: 'low' },
  { title: 'Medium', value: 'medium' },
  { title: 'High', value: 'high' },
  { title: 'XHigh', value: 'xhigh' },
  { title: 'Max', value: 'max' }
]

const reasoningParamStyleOptions = [
  { title: 'reasoning.effort', value: 'reasoning' },
  { title: 'reasoning_effort', value: 'reasoning_effort' },
  { title: 'thinking (JD/GLM)', value: 'thinking' }
]

const textVerbosityOptions = [
  { title: 'Low', value: 'low' },
  { title: 'Medium', value: 'medium' },
  { title: 'High', value: 'high' }
]

const supportsOpenAIAdvancedOptions = computed(() => supportsAdvancedChannelOptions(form.serviceType))
const supportsReasoningMappingOptions = computed(() => supportsReasoningMapping(form.serviceType))
const supportsChatRoleNormalization = computed(() => {
  return props.channelType === 'chat' || (props.channelType === 'responses' && form.serviceType === 'openai')
})

const showModelMappingPresets = computed(() => {
  // gpt-5.x 预设只配置 fable/opus/sonnet/haiku 重定向，限定在 Messages 入口展示。
  return props.channelType === 'messages' && (form.serviceType === 'openai' || form.serviceType === 'responses')
})

const showMessagesOpenAIChannelPresets = computed(() => {
  return props.channelType === 'messages' && (form.serviceType === 'openai' || form.serviceType === 'responses')
})
const modelNameCollator = new Intl.Collator('en', { numeric: true, sensitivity: 'base' })

const modelMappingPresets: Record<
  'gpt-5.5' | 'gpt-5.4',
  {
    modelMapping: Record<string, string>
    reasoningMapping: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>
    fastMode: boolean
    textVerbosity: 'low' | 'medium' | 'high'
  }
> = {
  'gpt-5.5': {
    modelMapping: {
      fable: 'gpt-5.5',
      opus: 'gpt-5.5',
      sonnet: 'gpt-5.4',
      haiku: 'gpt-5.4-mini'
    },
    reasoningMapping: {
      fable: 'xhigh',
      opus: 'xhigh',
      sonnet: 'xhigh',
      haiku: 'high'
    },
    fastMode: true,
    textVerbosity: 'medium'
  },
  'gpt-5.4': {
    modelMapping: {
      fable: 'gpt-5.4',
      opus: 'gpt-5.4',
      sonnet: 'gpt-5.4',
      haiku: 'gpt-5.4-mini'
    },
    reasoningMapping: {
      fable: 'xhigh',
      opus: 'xhigh',
      sonnet: 'xhigh',
      haiku: 'high'
    },
    fastMode: true,
    textVerbosity: 'medium'
  }
}

const applyModelMappingPreset = (preset: keyof typeof modelMappingPresets) => {
  const presetConfig = modelMappingPresets[preset]
  form.modelMapping = { ...presetConfig.modelMapping }
  form.fastMode = presetConfig.fastMode
  form.textVerbosity = presetConfig.textVerbosity

  if (supportsOpenAIAdvancedOptions.value) {
    form.reasoningMapping = { ...presetConfig.reasoningMapping }
  } else {
    form.reasoningMapping = {}
  }

  syncModelMappingRowsFromForm()
}

// fable/opus/sonnet/haiku 模型别名的一键预设（MiMo / DeepSeek）
const showClaudeChannelPresets = computed(() => {
  return form.serviceType === 'claude'
    && (props.channelType === 'messages' || props.channelType === 'chat' || props.channelType === 'responses')
})

const claudeChannelPresets: Record<
  'mimo' | 'deepseek' | 'minimax',
  {
    passbackReasoningContent: boolean
    passbackThinkingBlocks: boolean
    stripEmptyTextBlocks: boolean
    normalizeSystemRoleToTopLevel: boolean
    stripImageGenerationTool: boolean
    noVision: boolean
    noVisionModels: string[]
    visionFallbackModel: string
    modelMapping?: Record<string, string>
  }
> = {
  mimo: {
    passbackReasoningContent: true,
    passbackThinkingBlocks: false,
    stripEmptyTextBlocks: false,
    normalizeSystemRoleToTopLevel: false,
    stripImageGenerationTool: false,
    noVision: false,
    noVisionModels: ['mimo-v2.5-pro'],
    visionFallbackModel: 'mimo-v2.5',
    modelMapping: {
      fable: 'mimo-v2.5-pro',
      haiku: 'mimo-v2.5-pro',
      opus: 'mimo-v2.5-pro',
      sonnet: 'mimo-v2.5-pro'
    }
  },
  deepseek: {
    passbackReasoningContent: true,
    passbackThinkingBlocks: true,
    stripEmptyTextBlocks: true,
    normalizeSystemRoleToTopLevel: false,
    stripImageGenerationTool: true,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: '',
    modelMapping: {
      fable: 'deepseek-v4-pro',
      haiku: 'deepseek-v4-flash',
      opus: 'deepseek-v4-pro',
      sonnet: 'deepseek-v4-pro'
    }
  },
  minimax: {
    passbackReasoningContent: true,
    passbackThinkingBlocks: false,
    stripEmptyTextBlocks: false,
    normalizeSystemRoleToTopLevel: false,
    stripImageGenerationTool: false,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: '',
    modelMapping: {
      fable: 'MiniMax-M3',
      haiku: 'MiniMax-M2.7',
      opus: 'MiniMax-M3',
      sonnet: 'MiniMax-M3'
    }
  }
}

const applyClaudeChannelPreset = (preset: keyof typeof claudeChannelPresets) => {
  const presetConfig = claudeChannelPresets[preset]
  form.passbackReasoningContent = presetConfig.passbackReasoningContent
  form.passbackThinkingBlocks = presetConfig.passbackThinkingBlocks
  form.stripEmptyTextBlocks = presetConfig.stripEmptyTextBlocks
  form.normalizeSystemRoleToTopLevel = presetConfig.normalizeSystemRoleToTopLevel
  form.stripImageGenerationTool = presetConfig.stripImageGenerationTool
  form.noVision = presetConfig.noVision
  form.noVisionModels = [...presetConfig.noVisionModels]
  form.visionFallbackModel = presetConfig.visionFallbackModel
  form.visionFallbackReasoningEffort = ''
  if (presetConfig.modelMapping) {
    form.modelMapping = { ...presetConfig.modelMapping }
    form.reasoningMapping = {}
    syncModelMappingRowsFromForm()
  }
}

// Codex Responses 转 OpenAI 兼容上游的一键预设（MiMo / DeepSeek）
const showCodexResponsesChannelPresets = computed(() => {
  return props.channelType === 'responses' && supportsOpenAIAdvancedOptions.value
})

const codexResponsesChannelPresets: Record<
  'mimo' | 'deepseek' | 'minimax',
  {
    modelMapping: Record<string, string>
    reasoningMapping: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>
    reasoningParamStyle: 'reasoning' | 'reasoning_effort' | 'thinking'
    codexNativeToolPassthrough: boolean
    codexToolCompat: boolean
    stripCodexClientTools: boolean
    stripImageGenerationTool: boolean
    normalizeNonstandardChatRoles: boolean
    noVision: boolean
    noVisionModels: string[]
    visionFallbackModel: string
  }
> = {
  mimo: {
    modelMapping: {
      'codex': 'mimo-v2.5',
      'gpt': 'mimo-v2.5-pro'
    },
    reasoningMapping: {},
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: false,
    codexToolCompat: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: false,
    noVision: false,
    noVisionModels: ['mimo-v2.5-pro'],
    visionFallbackModel: 'mimo-v2.5'
  },
  deepseek: {
    modelMapping: {
      codex: 'deepseek-v4-flash',
      gpt: 'deepseek-v4-pro',
      mini: 'deepseek-v4-flash'
    },
    reasoningMapping: {
      gpt: 'max'
    },
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: true,
    codexToolCompat: false,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: true,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: ''
  },
  minimax: {
    modelMapping: {
      codex: 'MiniMax-M2.7',
      gpt: 'MiniMax-M3',
      mini: 'MiniMax-M2.7'
    },
    reasoningMapping: {},
    reasoningParamStyle: 'reasoning',
    codexNativeToolPassthrough: false,
    codexToolCompat: true,
    stripCodexClientTools: false,
    stripImageGenerationTool: false,
    normalizeNonstandardChatRoles: true,
    noVision: true,
    noVisionModels: [],
    visionFallbackModel: ''
  }
}

const applyCodexResponsesChannelPreset = (preset: keyof typeof codexResponsesChannelPresets) => {
  const presetConfig = codexResponsesChannelPresets[preset]
  form.modelMapping = { ...presetConfig.modelMapping }
  form.reasoningMapping = { ...presetConfig.reasoningMapping }
  form.reasoningParamStyle = presetConfig.reasoningParamStyle
  form.codexNativeToolPassthrough = presetConfig.codexNativeToolPassthrough
  form.codexToolCompat = presetConfig.codexToolCompat
  form.stripCodexClientTools = presetConfig.stripCodexClientTools
  form.stripImageGenerationTool = presetConfig.stripImageGenerationTool
  form.normalizeNonstandardChatRoles = presetConfig.normalizeNonstandardChatRoles
  form.noVision = presetConfig.noVision
  form.noVisionModels = [...presetConfig.noVisionModels]
  form.visionFallbackModel = presetConfig.visionFallbackModel
  form.visionFallbackReasoningEffort = ''

  syncModelMappingRowsFromForm()
}

// 模型优先级排序规则（索引越小优先级越高）
// 规则顺序：先新后旧、先精确后宽松；同家族新版本在前，带 codex/pro/max 等精确后缀优先于通用名
// 数据基线：2026-05 各家官方在售模型
const modelPriorityPatterns: RegExp[] = [
  // Anthropic Claude（Fable 5 / 4.8 旗舰 / 4.7 / 4.6 Sonnet / 4.5 Haiku）
  /fable-5/i,
  /opus-4-8/i,
  /opus-4-7/i,
  /sonnet-4-7/i,
  /haiku-4-7/i,
  /opus-4-6/i,
  /sonnet-4-6/i,
  /haiku-4-6/i,
  /opus-4-5/i,
  /sonnet-4-5/i,
  /haiku-4-5/i,

  // OpenAI GPT-5 系列（pro / codex 变体优先匹配，再降级到主版本）
  /gpt-5\.5-pro/i,
  /gpt-5\.5/i,
  /gpt-5\.4-pro/i,
  /gpt-5\.4-mini/i,
  /gpt-5\.4-nano/i,
  /gpt-5\.4/i,
  /gpt-5\.3-codex/i,
  /gpt-5\.3/i,
  /gpt-5\.2-codex/i,
  /gpt-5\.2-pro/i,
  /gpt-5\.2/i,
  /gpt-5\.1-codex/i,
  /gpt-5\.1/i,
  /gpt-5-codex/i,
  /gpt-5-pro/i,
  /gpt-5/i,

  // Google Gemini（3.5 Flash → 3.1 Pro Preview → 3 Pro / Flash Preview → 3.1 Flash Lite → 2.5 系列）
  /gemini-3\.5-flash/i,
  /gemini-3\.1-pro/i,
  /gemini-3\.1-flash-lite/i,
  /gemini-3-pro/i,
  /gemini-3-flash/i,
  /gemini-3/i,
  /gemini-2\.5-pro/i,
  /gemini-2\.5-flash-lite/i,
  /gemini-2\.5-flash/i,

  // xAI Grok（4.3 当前旗舰；保留 4.2/4.1 以兼容旧 channel 命名）
  /grok-4\.3/i,
  /grok-4-3/i,
  /grok-4\.2/i,
  /grok-4\.1/i,
  /grok-4/i,

  // 智谱 GLM
  /glm-?5\.1/i,
  /glm-?5/i,
  /glm-?4\.7-flash/i,
  /glm-?4\.7/i,
  /glm-?4\.6/i,

  // 阿里 Qwen（3.6 / 3.5 / 3-Max）
  /qwen-?3\.6-plus/i,
  /qwen-?3\.6/i,
  /qwen-?3\.5/i,
  /qwen-?3-max/i,
  /qwen-?3-coder/i,
  /qwen-?3/i,

  // DeepSeek（V4 已发布；deepseek-chat / deepseek-reasoner 对应 V3.2）
  /deepseek-v4-pro/i,
  /deepseek-v4-flash/i,
  /deepseek-v4/i,
  /deepseek-v3\.2/i,
  /deepseek-reasoner/i,
  /deepseek-chat/i,
  /deepseek-v3/i,

  // Moonshot Kimi / MiniMax（带版本号 → 通用简写）
  /kimi-?k2\.6/i,
  /kimi-?k2\.5/i,
  /minimax-?m3/i,
  /minimax-?m2\.7/i,
  /minimax-?m2\.5/i,
  /k2\.6/i,
  /k2\.5/i,
  /m3/i,
  /m2\.7/i,
  /m2\.5/i,

  // DeepSeek 兜底（匹配各种 deepseek- 前缀变体）
  /deepseek-/i,
]

const getModelPriority = (name: string): number => {
  for (let i = 0; i < modelPriorityPatterns.length; i++) {
    if (modelPriorityPatterns[i].test(name)) return i
  }
  return modelPriorityPatterns.length
}

const sortModelNamesDesc = (models: string[]): string[] => {
  return [...models].sort((a, b) => {
    const pa = getModelPriority(a)
    const pb = getModelPriority(b)
    if (pa !== pb) return pa - pb
    // 同优先级组内按自然降序
    return modelNameCollator.compare(b, a)
  })
}

// 表单数据：balanced 预设值作为渠道级默认回退值
const defaultStreamTimeouts = { ...streamTimeoutPresets.balanced }

type StreamTimeoutPresetKey = 'gentle' | 'balanced' | 'aggressive' | 'custom'

const form = reactive({
  name: '',
  serviceType: '' as 'openai' | 'gemini' | 'claude' | 'responses' | '',
  baseUrl: '',
  baseUrls: [] as string[],
  website: '',
  insecureSkipVerify: false,
  lowQuality: false,
  injectDummyThoughtSignature: false,
  stripThoughtSignature: false,
  passbackReasoningContent: false,
  passbackThinkingBlocks: false,
  stripEmptyTextBlocks: false,
  normalizeSystemRoleToTopLevel: false,
  description: '',
  apiKeys: [] as string[],
  modelMapping: {} as Record<string, string>,
  modelCapabilitiesText: '',
  defaultContextWindowTokens: null as string | number | null,
  defaultMaxOutputTokens: null as string | number | null,
  allowUnknownContext: false,
  reasoningMapping: {} as Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>,
  reasoningParamStyle: 'reasoning' as 'reasoning' | 'reasoning_effort' | 'thinking',
  textVerbosity: '' as 'low' | 'medium' | 'high' | '',
  fastMode: false,
  customHeaders: {} as Record<string, string>,
  proxyUrl: '',
  requestTimeoutMs: null as string | number | null,
  responseHeaderTimeoutMs: null as string | number | null,
  streamFirstContentTimeoutEnabled: false,
  streamFirstContentTimeoutMs: defaultStreamTimeouts.firstContentMs as number,
  streamInactivityTimeoutEnabled: false,
  streamInactivityTimeoutMs: defaultStreamTimeouts.inactivityMs as number,
  streamToolCallIdleTimeoutEnabled: false,
  streamToolCallIdleTimeoutMs: defaultStreamTimeouts.toolCallIdleMs as number,
  rateLimitRpm: null as string | number | null,
  rateLimitWindowMinutes: null as string | number | null,
  rateLimitMaxConcurrent: null as string | number | null,
  rateLimitAutoFromHeaders: true,
  routePrefix: '',
  supportedModels: [] as string[],
  autoBlacklistBalance: true,
  normalizeMetadataUserId: true,
  stripBillingHeader: false,
  codexNativeToolPassthrough: false,
  codexToolCompat: false,
  normalizeNonstandardChatRoles: false,
  stripCodexClientTools: false,
  stripImageGenerationTool: false,
  convertImageUrlToB64Json: false,
  noVision: false,
  noVisionModels: [] as string[],
  visionFallbackModel: '',
  visionFallbackReasoningEffort: '' as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max' | '',
  historicalImageTurnLimit: 0,
})

// 多 BaseURL 文本输入（独立变量，保留用户输入的换行）
const baseUrlsText = ref('')

// 监听 baseUrlsText 变化，同步到 form（去重等效 URL）
watch(baseUrlsText, val => {
  const { baseUrl, baseUrls } = syncBaseUrlsFormState(val, form.serviceType)
  form.baseUrl = baseUrl
  form.baseUrls = baseUrls
})

watch(() => form.serviceType, () => {
  const { baseUrl, baseUrls } = syncBaseUrlsFormState(baseUrlsText.value, form.serviceType)
  form.baseUrl = baseUrl
  form.baseUrls = baseUrls
})

// 原始密钥映射 (掩码密钥 -> 原始密钥)
const originalKeyMap = ref<Map<string, string>>(new Map())

// 新API密钥输入
const newApiKey = ref('')

// 密钥重复检测状态
const apiKeyError = ref('')
const duplicateKeyIndex = ref(-1)

// 处理 API 密钥输入事件
const handleApiKeyInput = () => {
  apiKeyError.value = ''
  duplicateKeyIndex.value = -1
}

// 复制功能相关状态
const copiedKeyIndex = ref<number | null>(null)

// 新模型映射输入
const newMapping = reactive({
  source: '',
  target: '',
  reasoningEffort: '' as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max' | ''
})

// 模型映射行数据结构（改用数组存储，支持直接编辑）
interface ModelMappingRow {
  id: number
  source: string
  target: string
  reasoning: '' | 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
  noVision: boolean
}

let rowIdCounter = 0
const modelMappingRows = ref<ModelMappingRow[]>([])

const hasNoVisionRows = computed(() => modelMappingRows.value.some(row => row.noVision && row.target.trim()))

// 模型映射编辑状态（已废弃，保留以防需要恢复）
const editingMapping = ref<string | null>(null)
const editMappingForm = reactive({
  targetModel: '',
  reasoning: '' as '' | 'off' | 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
})

// 自定义请求头输入
const newHeaderKey = ref('')
const newHeaderValue = ref('')

// 添加自定义请求头
const addCustomHeader = () => {
  const key = newHeaderKey.value.trim()
  const value = newHeaderValue.value.trim()
  if (key && value) {
    form.customHeaders[key] = value
    newHeaderKey.value = ''
    newHeaderValue.value = ''
  }
}

// 删除自定义请求头
const removeCustomHeader = (key: string) => {
  delete form.customHeaders[key]
}

function resetTransientUiState() {
  newApiKey.value = ''
  apiKeyError.value = ''
  duplicateKeyIndex.value = -1
  copiedKeyIndex.value = null
  newMapping.source = ''
  newMapping.target = ''
  newMapping.reasoningEffort = ''
  sourceMappingError.value = ''
  newHeaderKey.value = ''
  newHeaderValue.value = ''
  localRestoredKeys.value = new Set<string>()
  restoringKey.value = ''
  errors.name = ''
  errors.serviceType = ''
  errors.baseUrl = ''
  errors.website = ''
  formBaseUrlPreview.value = ''
}

// 安全地获取字符串值（处理 v-select/v-combobox 可能返回对象的情况）
const getStringValue = (val: string | { title: string; value: string } | null | undefined): string => {
  if (!val) return ''
  if (typeof val === 'string') return val
  return val.value || ''
}

// 源模型名验证错误
const sourceMappingError = ref('')

// 判断是否为内置源模型（内置选项允许更长名称）
const isPresetSourceModel = (val: string): boolean => {
  return allSourceModelOptions.value.some(opt => opt.value === val)
}

// 验证源模型名称（仅允许合法的模型名：字母、数字、连字符、下划线、点、斜杠）
const validateSourceModelName = (val: string): string => {
  if (!val) return ''
  if (!isPresetSourceModel(val) && val.length > 50) return t('addChannel.sourceModelNameTooLong')
  if (/\s/.test(val)) return t('addChannel.sourceModelNoSpaces')
  if (!/^[\w.\-/:@+]+$/.test(val)) return t('addChannel.sourceModelInvalidChars')
  return ''
}

// 检查映射输入是否有效
const isMappingInputValid = computed(() => {
  const source = getStringValue(newMapping.source).trim()
  const target = getStringValue(newMapping.target).trim()
  if (!source || !target) return false
  return !validateSourceModelName(source)
})

// 目标模型列表（从上游获取）
const targetModelOptions = ref<Array<{ title: string; value: string }>>([])
const fetchingModels = ref(false)
const fetchModelsError = ref('')
const hasTriedFetchModels = ref(false) // 标记是否已尝试获取过模型列表
const silentlySaving = ref(false)

// API Key 的 models 状态管理
interface KeyModelsStatus {
  loading: boolean
  success: boolean
  statusCode?: number
  error?: string
  modelCount?: number
}
const keyModelsStatus = ref<Map<string, KeyModelsStatus>>(new Map())

const restoreDisabledKeyLabelMap = {
  insufficient_balance: 'channelCard.blacklistReason.insufficient_balance',
  unavailable: 'channelCard.blacklistReason.unavailable',
  rate_limited: 'channelCard.blacklistReason.rate_limited',
  invalid: 'channelCard.blacklistReason.invalid',
  authentication_error: 'channelCard.blacklistReason.authentication_error',
  permission_error: 'channelCard.blacklistReason.permission_error',
  unknown: 'channelCard.blacklistReason.unknown',
} as const

const getRestoreDisabledKeyLabel = (reason?: string) => {
  return restoreDisabledKeyLabelMap[reason as keyof typeof restoreDisabledKeyLabelMap] || restoreDisabledKeyLabelMap.unknown
}

// 表单验证错误
const errors = reactive({
  name: '',
  serviceType: '',
  baseUrl: '',
  website: ''
})

// 验证规则
const rules = {
  required: (value: string) => !!value || t('addChannel.fieldRequired'),
  url: (value: string) => {
    try {
      new URL(value)
      return true
    } catch {
      return t('addChannel.invalidUrl')
    }
  },
  urlOptional: (value: string) => {
    if (!value) return true
    try {
      new URL(value)
      return true
    } catch {
      return t('addChannel.invalidUrl')
    }
  },
  baseUrls: (value: string) => {
    if (!value) return t('addChannel.fieldRequired')
    const urls = value
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
    if (urls.length === 0) return t('addChannel.atLeastOneUrl')
    for (const url of urls) {
      try {
        new URL(url)
      } catch {
        return t('addChannel.invalidUrlValue', { url })
      }
    }
    return true
  },
  requestTimeoutMs: (value: string | number | null) => {
    if (value === null || value === undefined || value === '') return true
    const timeout = Number(value)
    return (Number.isInteger(timeout) && timeout >= 1000 && timeout <= 300000) || t('addChannel.requestTimeoutMsInvalid')
  },
  responseHeaderTimeoutMs: (value: string | number | null) => {
    if (value === null || value === undefined || value === '') return true
    const timeout = Number(value)
    return (Number.isInteger(timeout) && timeout >= 1000 && timeout <= 300000) || t('addChannel.responseHeaderTimeoutMsInvalid')
  }
}

// 计算属性
const dialogMode = ref<'create' | 'edit'>('create')
const isEditing = computed(() => dialogMode.value === 'edit')
const isMac = computed(() => typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform))
const hasDisabledKeysAvailable = computed(() => visibleDisabledKeys.value.length > 0)
const hasConfigurableKeys = computed(() => form.apiKeys.length > 0 || (isEditing.value && hasDisabledKeysAvailable.value))

const selectedStreamTimeoutStrategy = computed(() => {
  if (!form.streamFirstContentTimeoutEnabled && !form.streamInactivityTimeoutEnabled && !form.streamToolCallIdleTimeoutEnabled) {
    return 'inherit'
  }
  for (const [key, preset] of Object.entries(streamTimeoutPresets) as Array<[StreamTimeoutPresetKey, { firstContentMs: number; inactivityMs: number; toolCallIdleMs: number }]>) {
    if (
      form.streamFirstContentTimeoutEnabled
      && form.streamInactivityTimeoutEnabled
      && form.streamToolCallIdleTimeoutEnabled
      && form.streamFirstContentTimeoutMs === preset.firstContentMs
      && form.streamInactivityTimeoutMs === preset.inactivityMs
      && form.streamToolCallIdleTimeoutMs === preset.toolCallIdleMs
    ) {
      return key
    }
  }
  return 'custom'
})

const applyStreamTimeoutStrategy = (strategy: string | null) => {
  if (!strategy) return
  if (strategy === 'inherit') {
    form.streamFirstContentTimeoutEnabled = false
    form.streamInactivityTimeoutEnabled = false
    form.streamToolCallIdleTimeoutEnabled = false
    return
  }

  const preset = streamTimeoutPresets[strategy as keyof typeof streamTimeoutPresets]
  if (!preset) return
  form.streamFirstContentTimeoutEnabled = true
  form.streamFirstContentTimeoutMs = preset.firstContentMs
  form.streamInactivityTimeoutEnabled = true
  form.streamInactivityTimeoutMs = preset.inactivityMs
  form.streamToolCallIdleTimeoutEnabled = true
  form.streamToolCallIdleTimeoutMs = preset.toolCallIdleMs
}

const commonSupportedModelFilters = ['claude-*', 'gpt-5*', 'gpt-image-2', 'grok-4*', 'gemini-3*', '!*image*']

const selectedSupportedModelSet = computed(() => new Set(form.supportedModels))
const supportedModelsError = ref('')
const modelCapabilitiesJsonPlaceholder = `{
  "claude-sonnet-4-6": {
    "contextWindowTokens": 1000000,
    "maxOutputTokens": 64000
  }
}`
const modelCapabilitiesError = computed(() => {
  return parseModelCapabilitiesText(form.modelCapabilitiesText) === null
    ? t('addChannel.modelCapabilitiesJsonInvalid')
    : ''
})

// 动态header样式
const headerClasses = computed(() => {
  const isDark = theme.global.current.value.dark
  // Dark: keep neutral surface header; Light: use brand primary header
  return isDark ? 'bg-surface text-high-emphasis' : 'bg-primary text-white'
})

const avatarColor = computed(() => 'primary')

// Use Vuetify theme "on-primary" token so icon isn't fixed white
const headerIconStyle = computed(() => ({
  color: 'rgb(var(--v-theme-on-primary))'
}))

const subtitleClasses = computed(() => {
  const isDark = theme.global.current.value.dark
  // Dark mode: use medium emphasis; Light mode: use white with opacity for primary bg
  return isDark ? 'text-medium-emphasis' : 'text-white-subtitle'
})

const isFormValid = computed(() => {
  return (
    form.name.trim() && form.serviceType && form.baseUrl.trim() && isValidUrl(form.baseUrl) && hasConfigurableKeys.value && !modelCapabilitiesError.value
  )
})

// 工具函数
const isValidUrl = (url: string): boolean => {
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

const normalizeStringArray = (values: string[]): string[] => values.map(v => v.trim()).filter(Boolean)

const handleSupportedModelsChange = (values: Array<string | { title: string; value: string }>) => {
  // 用户可能把多条规则用顿号/逗号粘进同一项，这里统一按分隔符拆分
  const normalizedValues = values
    .map(getStringValue)
    .flatMap(parseSupportedModelInput)

  const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(normalizedValues)
  form.supportedModels = validPatterns
  supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
}

const normalizeStringRecord = (record: Record<string, string>): Record<string, string> => {
  const normalized: Record<string, string> = {}
  Object.entries(record)
    .map(([key, value]) => [key.trim(), value.trim()] as const)
    .filter(([key, value]) => key && value)
    .sort(([keyA], [keyB]) => keyA.localeCompare(keyB))
    .forEach(([key, value]) => {
      normalized[key] = value
    })
  return normalized
}

const normalizeModelCapabilities = (record: Channel['modelCapabilities'] = {}): Channel['modelCapabilities'] => {
  return Object.fromEntries(Object.entries(record).sort(([a], [b]) => a.localeCompare(b)))
}

const buildComparablePayload = () => {
  const payload = buildSubmitPayload()
  return normalizeComparablePayload(payload)
}

const normalizeComparablePayload = (payload: Partial<Channel>) => ({
  ...payload,
  apiKeys: normalizeStringArray(payload.apiKeys || []),
  baseUrls: normalizeStringArray(payload.baseUrls || []),
  supportedModels: normalizeStringArray(payload.supportedModels || []),
  customHeaders: normalizeStringRecord(payload.customHeaders || {}),
  modelMapping: Object.fromEntries(Object.entries(payload.modelMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
  modelCapabilities: normalizeModelCapabilities(payload.modelCapabilities || {}),
  defaultCapability: payload.defaultCapability || {},
  allowUnknownContext: !!payload.allowUnknownContext,
  reasoningMapping: Object.fromEntries(Object.entries(payload.reasoningMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
  reasoningParamStyle: payload.reasoningParamStyle || 'reasoning',
  requestTimeoutMs: payload.requestTimeoutMs || undefined,
  responseHeaderTimeoutMs: payload.responseHeaderTimeoutMs || undefined,
  streamFirstContentTimeoutMs: payload.streamFirstContentTimeoutMs || undefined,
  streamInactivityTimeoutMs: payload.streamInactivityTimeoutMs || undefined,
  streamToolCallIdleTimeoutMs: payload.streamToolCallIdleTimeoutMs || undefined,
  rateLimitRpm: payload.rateLimitRpm || undefined,
  rateLimitWindowMinutes: payload.rateLimitWindowMinutes || undefined,
  rateLimitMaxConcurrent: payload.rateLimitMaxConcurrent || undefined,
  rateLimitAutoFromHeaders: !!payload.rateLimitAutoFromHeaders,
})

const buildSubmitPayload = () => {
  const payload = buildChannelPayload(form)
  applyVisionFallbackReasoning(payload)
  if (!form.streamFirstContentTimeoutEnabled) {
    delete payload.streamFirstContentTimeoutMs
    if (isEditing.value && props.channel?.streamFirstContentTimeoutMs) {
      payload.streamFirstContentTimeoutMs = 0
    }
  }
  if (!form.streamInactivityTimeoutEnabled) {
    delete payload.streamInactivityTimeoutMs
    if (isEditing.value && props.channel?.streamInactivityTimeoutMs) {
      payload.streamInactivityTimeoutMs = 0
    }
  }
  if (!form.streamToolCallIdleTimeoutEnabled) {
    delete payload.streamToolCallIdleTimeoutMs
    if (isEditing.value && props.channel?.streamToolCallIdleTimeoutMs) {
      payload.streamToolCallIdleTimeoutMs = 0
    }
  }
  if (isEditing.value && props.channel?.requestTimeoutMs && !payload.requestTimeoutMs) {
    payload.requestTimeoutMs = 0
  }
  if (isEditing.value && props.channel?.responseHeaderTimeoutMs && !payload.responseHeaderTimeoutMs) {
    payload.responseHeaderTimeoutMs = 0
  }
  if (isEditing.value && props.channel?.rateLimitRpm && !payload.rateLimitRpm) {
    payload.rateLimitRpm = 0
  }
  if (isEditing.value && props.channel?.rateLimitWindowMinutes && !payload.rateLimitWindowMinutes) {
    payload.rateLimitWindowMinutes = 0
  }
  if (isEditing.value && props.channel?.rateLimitMaxConcurrent && !payload.rateLimitMaxConcurrent) {
    payload.rateLimitMaxConcurrent = 0
  }
  return payload
}

const applyVisionFallbackReasoning = (payload: Partial<Channel>) => {
  const fallbackModel = normalizeSelectableString(form.visionFallbackModel).trim()
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

const hasEditableDraftChanges = computed(() => {
  if (!isEditing.value || !props.channel) return false
  const currentPayload = buildComparablePayload()
  const originalPayload = {
    name: props.channel.name.trim(),
    serviceType: props.channel.serviceType,
    baseUrl: props.channel.baseUrl || '',
    baseUrls: normalizeStringArray(props.channel.baseUrls || []),
    website: (props.channel.website || '').trim(),
    insecureSkipVerify: !!props.channel.insecureSkipVerify,
    lowQuality: !!props.channel.lowQuality,
    injectDummyThoughtSignature: !!props.channel.injectDummyThoughtSignature,
    stripThoughtSignature: !!props.channel.stripThoughtSignature,
    passbackReasoningContent: !!props.channel.passbackReasoningContent,
    passbackThinkingBlocks: !!props.channel.passbackThinkingBlocks,
    stripEmptyTextBlocks: !!props.channel.stripEmptyTextBlocks,
    normalizeSystemRoleToTopLevel: !!props.channel.normalizeSystemRoleToTopLevel,
    description: (props.channel.description || '').trim(),
    apiKeys: normalizeStringArray(props.channel.apiKeys || []),
    modelMapping: Object.fromEntries(Object.entries(props.channel.modelMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
    modelCapabilities: normalizeModelCapabilities(props.channel.modelCapabilities || {}),
    defaultCapability: props.channel.defaultCapability || {},
    allowUnknownContext: !!props.channel.allowUnknownContext,
    reasoningMapping: Object.fromEntries(Object.entries(props.channel.reasoningMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
    reasoningParamStyle: props.channel.reasoningParamStyle || 'reasoning',
    textVerbosity: props.channel.textVerbosity || '',
    fastMode: !!props.channel.fastMode,
    customHeaders: normalizeStringRecord(props.channel.customHeaders || {}),
    proxyUrl: props.channel.proxyUrl || '',
    requestTimeoutMs: props.channel.requestTimeoutMs || undefined,
    responseHeaderTimeoutMs: props.channel.responseHeaderTimeoutMs || undefined,
    streamFirstContentTimeoutMs: props.channel.streamFirstContentTimeoutMs || undefined,
    streamInactivityTimeoutMs: props.channel.streamInactivityTimeoutMs || undefined,
    streamToolCallIdleTimeoutMs: props.channel.streamToolCallIdleTimeoutMs || undefined,
    rateLimitRpm: props.channel.rateLimitRpm || undefined,
    rateLimitWindowMinutes: props.channel.rateLimitWindowMinutes || undefined,
    rateLimitMaxConcurrent: props.channel.rateLimitMaxConcurrent || undefined,
    rateLimitAutoFromHeaders: !!props.channel.rateLimitAutoFromHeaders,
    routePrefix: props.channel.routePrefix || '',
    supportedModels: normalizeStringArray(props.channel.supportedModels || []),
    autoBlacklistBalance: props.channel.autoBlacklistBalance ?? true,
    normalizeMetadataUserId: props.channel.normalizeMetadataUserId ?? true,
    stripBillingHeader: props.channel.stripBillingHeader ?? false,
    codexNativeToolPassthrough: !!props.channel.codexNativeToolPassthrough,
    codexToolCompat: props.channel.codexToolCompat ?? props.channel.stripCodexClientTools ?? false,
    normalizeNonstandardChatRoles: !!props.channel.normalizeNonstandardChatRoles,
    stripCodexClientTools: props.channel.codexToolCompat ?? props.channel.stripCodexClientTools ?? false,
    stripImageGenerationTool: !!props.channel.stripImageGenerationTool,
    convertImageUrlToB64Json: !!props.channel.convertImageUrlToB64Json,
    noVision: !!props.channel.noVision,
    noVisionModels: [...(props.channel.noVisionModels || [])],
    visionFallbackModel: props.channel.visionFallbackModel || '',
    historicalImageTurnLimit: props.channel.historicalImageTurnLimit ?? 0,
  }

  return JSON.stringify(currentPayload) !== JSON.stringify(normalizeComparablePayload(originalPayload as Partial<Channel>))
})

const ensureLatestSavedChannel = async (): Promise<number | null> => {
  if (!isEditing.value || props.channel?.index === undefined || props.channel?.index === null) {
    return props.channel?.index ?? null
  }
  if (!hasEditableDraftChanges.value) {
    return props.channel.index
  }
  if (silentlySaving.value) {
    return null
  }

  if (formRef.value) {
    const { valid } = await formRef.value.validate()
    if (!valid) {
      return null
    }
  }
  if (modelCapabilitiesError.value) {
    return null
  }

  silentlySaving.value = true
  try {
    const payload = buildSubmitPayload()
    const result = await channelStore.saveChannel(payload, props.channel.index)
    await channelStore.refreshChannels()
    const latestChannel = (channelStore.currentChannelsData as any).channels?.find((ch: any) => ch.index === props.channel!.index) || null
    if (latestChannel) {
      dialogStore.editingChannel = latestChannel
    }
    return result.channelId ?? props.channel.index
  } catch (error) {
    const message = error instanceof Error ? error.message : t('system.unknown')
    emit('error', message)
    return null
  } finally {
    silentlySaving.value = false
  }
}

// 表单操作
const resetForm = () => {
  resetTransientUiState()
  form.name = ''
  form.serviceType = props.channelType === 'images' ? 'openai' : ''
  form.baseUrl = ''
  form.baseUrls = []
  form.website = ''
  form.insecureSkipVerify = false
  form.lowQuality = false
  form.injectDummyThoughtSignature = false
  form.stripThoughtSignature = false
  form.passbackReasoningContent = false
  form.passbackThinkingBlocks = false
  form.stripEmptyTextBlocks = false
  form.normalizeSystemRoleToTopLevel = false
  form.description = ''
  form.apiKeys = []
  form.modelMapping = {}
  form.modelCapabilitiesText = ''
  form.defaultContextWindowTokens = null
  form.defaultMaxOutputTokens = null
  form.allowUnknownContext = false
  form.reasoningMapping = {}

  // 清空模型映射行
  modelMappingRows.value = []

  form.reasoningParamStyle = 'reasoning'
  form.textVerbosity = ''
  form.fastMode = false
  form.customHeaders = {}
  form.proxyUrl = ''
  form.requestTimeoutMs = null
  form.responseHeaderTimeoutMs = null
  form.streamFirstContentTimeoutEnabled = false
  form.streamFirstContentTimeoutMs = defaultStreamTimeouts.firstContentMs
  form.streamInactivityTimeoutEnabled = false
  form.streamInactivityTimeoutMs = defaultStreamTimeouts.inactivityMs
  form.streamToolCallIdleTimeoutEnabled = false
  form.streamToolCallIdleTimeoutMs = defaultStreamTimeouts.toolCallIdleMs
  form.rateLimitRpm = null
  form.rateLimitWindowMinutes = null
  form.rateLimitMaxConcurrent = null
  form.rateLimitAutoFromHeaders = true
  form.routePrefix = ''
  form.supportedModels = []
  supportedModelsError.value = ''
  form.autoBlacklistBalance = true
  form.normalizeMetadataUserId = true
  form.stripBillingHeader = false
  form.codexNativeToolPassthrough = false
  form.codexToolCompat = false
  form.normalizeNonstandardChatRoles = false
  form.stripCodexClientTools = false
  form.stripImageGenerationTool = false
  form.convertImageUrlToB64Json = false
  form.noVision = false
  form.noVisionModels = []
  form.visionFallbackModel = ''
  form.visionFallbackReasoningEffort = ''
  form.historicalImageTurnLimit = 0

  // 重置 baseUrlsText
  baseUrlsText.value = ''

  // 清空原始密钥映射
  originalKeyMap.value.clear()

  // 清空模型缓存和状态
  targetModelOptions.value = []
  fetchingModels.value = false
  fetchModelsError.value = ''
  keyModelsStatus.value.clear()
  hasTriedFetchModels.value = false

  }

const loadChannelData = (channel: Channel) => {
  resetTransientUiState()
  form.name = channel.name
  form.serviceType = props.channelType === 'images' ? 'openai' : channel.serviceType
  form.baseUrl = channel.baseUrl
  form.baseUrls = channel.baseUrls || []
  form.website = channel.website || ''
  form.insecureSkipVerify = !!channel.insecureSkipVerify
  form.lowQuality = !!channel.lowQuality
  form.injectDummyThoughtSignature = !!channel.injectDummyThoughtSignature
  form.stripThoughtSignature = !!channel.stripThoughtSignature
  form.passbackReasoningContent = !!channel.passbackReasoningContent
  form.passbackThinkingBlocks = !!channel.passbackThinkingBlocks
  form.stripEmptyTextBlocks = !!channel.stripEmptyTextBlocks
  form.normalizeSystemRoleToTopLevel = !!channel.normalizeSystemRoleToTopLevel
  form.description = channel.description || ''

  // 同步 baseUrlsText（优先使用 baseUrls，否则使用 baseUrl），保留用户显式配置的原始 URL 形式
  const rawUrls = channel.baseUrls && channel.baseUrls.length > 0
    ? channel.baseUrls
    : (channel.baseUrl ? [channel.baseUrl] : [])
  baseUrlsText.value = rawUrls.join('\n')

  // 直接存储原始密钥，不需要映射关系
  form.apiKeys = [...channel.apiKeys]

  // 清空原始密钥映射（现在不需要了）
  originalKeyMap.value.clear()

  form.modelMapping = { ...(channel.modelMapping || {}) }
  form.modelCapabilitiesText = Object.keys(channel.modelCapabilities || {}).length > 0
    ? JSON.stringify(normalizeModelCapabilities(channel.modelCapabilities), null, 2)
    : ''
  form.defaultContextWindowTokens = channel.defaultCapability?.contextWindowTokens || null
  form.defaultMaxOutputTokens = channel.defaultCapability?.maxOutputTokens || null
  form.allowUnknownContext = !!channel.allowUnknownContext
  form.reasoningMapping = { ...(channel.reasoningMapping || {}) }

  // 加载模型映射行
  loadModelMappingRows(channel)

  form.reasoningParamStyle = channel.reasoningParamStyle || 'reasoning'
  form.textVerbosity = channel.textVerbosity || ''
  form.fastMode = !!channel.fastMode
  form.customHeaders = { ...(channel.customHeaders || {}) }
  form.proxyUrl = channel.proxyUrl || ''
  form.requestTimeoutMs = channel.requestTimeoutMs || null
  form.responseHeaderTimeoutMs = channel.responseHeaderTimeoutMs || null
  form.streamFirstContentTimeoutEnabled = !!(channel.streamFirstContentTimeoutMs && channel.streamFirstContentTimeoutMs > 0)
  form.streamFirstContentTimeoutMs = channel.streamFirstContentTimeoutMs && channel.streamFirstContentTimeoutMs > 0 ? channel.streamFirstContentTimeoutMs : defaultStreamTimeouts.firstContentMs
  form.streamInactivityTimeoutEnabled = !!(channel.streamInactivityTimeoutMs && channel.streamInactivityTimeoutMs > 0)
  form.streamInactivityTimeoutMs = channel.streamInactivityTimeoutMs && channel.streamInactivityTimeoutMs > 0 ? channel.streamInactivityTimeoutMs : defaultStreamTimeouts.inactivityMs
  form.streamToolCallIdleTimeoutEnabled = !!(channel.streamToolCallIdleTimeoutMs && channel.streamToolCallIdleTimeoutMs >= 30000)
  form.streamToolCallIdleTimeoutMs = channel.streamToolCallIdleTimeoutMs && channel.streamToolCallIdleTimeoutMs >= 30000 ? channel.streamToolCallIdleTimeoutMs : defaultStreamTimeouts.toolCallIdleMs
  form.rateLimitRpm = (channel.rateLimitRpm && channel.rateLimitRpm > 0) ? channel.rateLimitRpm : null
  form.rateLimitWindowMinutes = (channel.rateLimitWindowMinutes && channel.rateLimitWindowMinutes > 0) ? channel.rateLimitWindowMinutes : null
  form.rateLimitMaxConcurrent = (channel.rateLimitMaxConcurrent && channel.rateLimitMaxConcurrent > 0) ? channel.rateLimitMaxConcurrent : null
  form.rateLimitAutoFromHeaders = !!channel.rateLimitAutoFromHeaders
  form.routePrefix = channel.routePrefix || ''
  const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(channel.supportedModels || [])
  form.supportedModels = validPatterns
  supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
  form.autoBlacklistBalance = channel.autoBlacklistBalance ?? true
  form.normalizeMetadataUserId = channel.normalizeMetadataUserId ?? true
  form.stripBillingHeader = channel.stripBillingHeader ?? false
  form.codexNativeToolPassthrough = !!channel.codexNativeToolPassthrough
  form.codexToolCompat = channel.codexToolCompat ?? channel.stripCodexClientTools ?? false
  form.normalizeNonstandardChatRoles = !!channel.normalizeNonstandardChatRoles
  form.stripCodexClientTools = channel.codexToolCompat ?? channel.stripCodexClientTools ?? false
  form.stripImageGenerationTool = !!channel.stripImageGenerationTool
  form.convertImageUrlToB64Json = !!channel.convertImageUrlToB64Json
  form.noVision = !!channel.noVision
  form.noVisionModels = [...(channel.noVisionModels || [])]
  form.visionFallbackModel = channel.visionFallbackModel || ''
  form.visionFallbackReasoningEffort = (channel.reasoningMapping?.[form.visionFallbackModel] || '') as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max' | ''
  form.historicalImageTurnLimit = channel.historicalImageTurnLimit ?? 0

  // 立即同步 baseUrl 到预览变量，避免等待 debounce
  formBaseUrlPreview.value = channel.baseUrl

  // 清空模型映射输入框
  newMapping.source = ''
  newMapping.target = ''

  // 清空模型缓存和状态（切换渠道时重置）
  targetModelOptions.value = []
  fetchingModels.value = false
  fetchModelsError.value = ''
  keyModelsStatus.value.clear()
  hasTriedFetchModels.value = false

  // 如果有模型映射配置，主动预加载模型列表
  if (channel.modelMapping && Object.keys(channel.modelMapping).length > 0) {
    nextTick(() => {
      fetchTargetModels()
    })
  }
}

const addApiKey = () => {
  const key = newApiKey.value.trim()
  if (!key) return

  // 重置错误状态
  apiKeyError.value = ''
  duplicateKeyIndex.value = -1

  // 检查是否与现有密钥重复
  const duplicateIndex = findDuplicateKeyIndex(key)
  if (duplicateIndex !== -1) {
    apiKeyError.value = t('addChannel.duplicateKeyExists')
    duplicateKeyIndex.value = duplicateIndex
    // 清除输入框，让用户重新输入
    newApiKey.value = ''
    return
  }

  // 直接存储原始密钥
  form.apiKeys.push(key)
  newApiKey.value = ''
}

// 检查密钥是否重复，返回重复密钥的索引，如果没有重复返回-1
const findDuplicateKeyIndex = (newKey: string): number => {
  return form.apiKeys.findIndex(existingKey => existingKey === newKey)
}

const removeApiKey = (index: number) => {
  form.apiKeys.splice(index, 1)

  // 如果删除的是当前高亮的重复密钥，清除高亮状态
  if (duplicateKeyIndex.value === index) {
    duplicateKeyIndex.value = -1
    apiKeyError.value = ''
  } else if (duplicateKeyIndex.value > index) {
    // 如果删除的密钥在高亮密钥之前，调整高亮索引
    duplicateKeyIndex.value--
  }
}

// 将指定密钥移到最上方
const moveApiKeyToTop = (index: number) => {
  if (index <= 0 || index >= form.apiKeys.length) return
  const [key] = form.apiKeys.splice(index, 1)
  form.apiKeys.unshift(key)
  duplicateKeyIndex.value = -1
  copiedKeyIndex.value = null
}

// 将指定密钥移到最下方
const moveApiKeyToBottom = (index: number) => {
  if (index < 0 || index >= form.apiKeys.length - 1) return
  const [key] = form.apiKeys.splice(index, 1)
  form.apiKeys.push(key)
  duplicateKeyIndex.value = -1
  copiedKeyIndex.value = null
}

// 恢复被拉黑的密钥
const restoringKey = ref('')
const localRestoredKeys = ref(new Set<string>())

// 提交状态
const submitting = ref(false)

// 被拉黑的密钥（直接从 props.channel 读取）
const disabledKeys = computed(() => props.channel?.disabledApiKeys || [])

// 本地过滤已恢复的 key，不直接修改 props
const visibleDisabledKeys = computed(() =>
  (props.channel?.disabledApiKeys || []).filter(dk => !localRestoredKeys.value.has(dk.key))
)

// 预期请求 URL 列表
const expectedRequestUrls = computed(() => {
  if (!baseUrlsText.value || !form.serviceType) return []
  return buildExpectedRequestUrls(
    props.channelType,
    form.serviceType,
    undefined,
    baseUrlsText.value.split('\n').map(url => url.trim()).filter(Boolean),
  )
})

// 自定义请求头转换（Record <-> Array）
const customHeadersArray = computed(() => {
  return Object.entries(form.customHeaders).map(([key, value]) => ({ key, value }))
})

const updateCustomHeaders = (headers: Array<{ key: string; value: string }>) => {
  const newHeaders: Record<string, string> = {}
  headers.forEach(h => {
    if (h.key && h.value) {
      newHeaders[h.key] = h.value
    }
  })
  form.customHeaders = newHeaders
}


const restoreDisabledKey = async (apiKey: string) => {
  if (!props.channel) return
  restoringKey.value = apiKey
  try {
    const channelId = props.channel.index
    if (props.channelType === 'chat') {
      await apiService.restoreChatApiKey(channelId, apiKey)
    } else if (props.channelType === 'images') {
      await apiService.restoreImagesApiKey(channelId, apiKey)
    } else if (props.channelType === 'gemini') {
      await apiService.restoreGeminiApiKey(channelId, apiKey)
    } else if (props.channelType === 'responses') {
      await apiService.restoreResponsesApiKey(channelId, apiKey)
    } else {
      await apiService.restoreApiKey(channelId, apiKey)
    }
    // 本地标记已恢复，加入活跃列表
    localRestoredKeys.value.add(apiKey)
    form.apiKeys.push(apiKey)
  } catch (error) {
    apiKeyError.value = error instanceof Error ? error.message : 'Restore failed'
  } finally {
    restoringKey.value = ''
  }
}

// 复制API密钥到剪贴板
const copyApiKey = async (key: string, index: number) => {
  try {
    await navigator.clipboard.writeText(key)
    copiedKeyIndex.value = index

    // 2秒后重置复制状态
    setTimeout(() => {
      copiedKeyIndex.value = null
    }, 2000)
  } catch (err) {
    console.error('复制密钥失败:', err)
    // 降级方案：使用传统的复制方法
    const textArea = document.createElement('textarea')
    textArea.value = key
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    textArea.style.top = '-999999px'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()

    try {
      document.execCommand('copy')
      copiedKeyIndex.value = index

      setTimeout(() => {
        copiedKeyIndex.value = null
      }, 2000)
    } catch (err) {
      console.error('降级复制方案也失败:', err)
    } finally {
      textArea.remove()
    }
  }
}

// 处理源模型名输入变化，实时验证
const handleSourceModelChange = (val: string | { title: string; value: string } | null) => {
  const source = getStringValue(val).trim()
  if (!source) {
    sourceMappingError.value = ''
    return
  }
  sourceMappingError.value = validateSourceModelName(source)
}

const addModelMapping = () => {
  const source = getStringValue(newMapping.source).trim()
  const target = getStringValue(newMapping.target).trim()

  // 验证源模型名
  const sourceErr = validateSourceModelName(source)
  if (sourceErr) {
    sourceMappingError.value = sourceErr
    return
  }
  sourceMappingError.value = ''

  if (source && target) {
    // 检查是否已存在
    const exists = modelMappingRows.value.some(row => row.source === source)
    if (exists) {
      sourceMappingError.value = '该源模型已存在映射'
      return
    }

    modelMappingRows.value.push({
      id: ++rowIdCounter,
      source,
      target,
      reasoning: newMapping.reasoningEffort || '',
      noVision: false
    })

    newMapping.source = ''
    newMapping.target = ''
    newMapping.reasoningEffort = ''
  }
}

const removeModelMappingRow = (index: number) => {
  modelMappingRows.value.splice(index, 1)
}

const toggleRowVision = (row: ModelMappingRow) => {
  row.noVision = !row.noVision
}

// 将 modelMappingRows 转换为 form.modelMapping 对象（保存时使用）
const syncModelMappingToForm = () => {
  form.modelMapping = {}
  form.reasoningMapping = {}
  form.noVisionModels = []
  const noVisionModels = new Set<string>()

  modelMappingRows.value.forEach(row => {
    if (row.source && row.target) {
      form.modelMapping[row.source] = row.target
      if (row.reasoning) {
        form.reasoningMapping[row.source] = row.reasoning
      }
      if (row.noVision) {
        noVisionModels.add(row.target)
      }
    }
  })

  form.noVisionModels = [...noVisionModels]
}

// 从渠道数据初始化 modelMappingRows
const loadModelMappingRows = (channel: Channel) => {
  const mapping = channel.modelMapping || {}
  const reasoning = channel.reasoningMapping || {}
  const noVisionSet = new Set(channel.noVisionModels || [])

  modelMappingRows.value = Object.entries(mapping).map(([source, target]) => ({
    id: ++rowIdCounter,
    source,
    target,
    reasoning: (reasoning[source] || '') as ModelMappingRow['reasoning'],
    noVision: noVisionSet.has(target)
  }))
}

const syncModelMappingRowsFromForm = () => {
  const noVisionSet = new Set(form.noVisionModels || [])

  modelMappingRows.value = Object.entries(form.modelMapping || {}).map(([source, target]) => ({
    id: ++rowIdCounter,
    source,
    target,
    reasoning: (form.reasoningMapping[source] || '') as ModelMappingRow['reasoning'],
    noVision: noVisionSet.has(target)
  }))
}

// 开始编辑模型映射（已废弃）
const startEditMapping = (source: string) => {
  editingMapping.value = source
  editMappingForm.targetModel = form.modelMapping[source] || ''
  editMappingForm.reasoning = (form.reasoningMapping[source] || '') as '' | 'off' | 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
}

// 取消编辑模型映射（已废弃）
const cancelEditMapping = () => {
  editingMapping.value = null
  editMappingForm.targetModel = ''
  editMappingForm.reasoning = ''
}

// 保存编辑的模型映射（已废弃，保留以防需要恢复）
// saveEditMapping - 已废弃，改用直接编辑模式
// const saveEditMapping = async () => { ... }

const isSupportedModelSelected = (filter: string): boolean => {
  return selectedSupportedModelSet.value.has(filter)
}

const appendSupportedModelFilter = (filter: string) => {
  if (isSupportedModelSelected(filter)) {
    return
  }
  form.supportedModels.push(filter)
  supportedModelsError.value = ''
}

// 处理目标模型输入框点击事件(仅在首次或有新 key 时触发请求)
const handleTargetModelClick = () => {
  // 如果已经尝试过获取且正在加载中,不重复触发
  if (hasTriedFetchModels.value || fetchingModels.value) {
    return
  }

  // 标记已尝试获取
  hasTriedFetchModels.value = true

  // 调用获取模型列表(内部有缓存逻辑)
  fetchTargetModels()
}

const ensureTargetModelsLoaded = () => {
  if (targetModelOptions.value.length === 0) {
    fetchTargetModels()
  }
}

const fetchTargetModels = async () => {
  const candidateKeys = form.apiKeys.length > 0
    ? form.apiKeys
    : (isEditing.value ? visibleDisabledKeys.value.map(dk => dk.key) : [])

  if (!form.baseUrl || candidateKeys.length === 0) {
    fetchModelsError.value = t('addChannel.fillBaseUrlAndApiKey')
    return
  }

  const channelId = props.channel?.index

  // 仅为未检测过的 API Key 发起请求
  const uncheckedKeys = candidateKeys.filter(key => !keyModelsStatus.value.has(key))

  if (uncheckedKeys.length === 0) {
    return
  }

  fetchingModels.value = true
  fetchModelsError.value = ''

  // modelsApiType 决定请求协议（Bearer/x-goog-api-key、/v1/models vs /v1beta/models）
  // 对于 gemini 渠道组内配置为 openai/claude serviceType 的渠道，应走对应协议而非 Gemini 协议
  const effectiveServiceType = props.channelType === 'images'
    ? 'openai'
    : (form.serviceType || defaultServiceTypeValueFallback())
  let modelsApiType: 'messages' | 'responses' | 'chat' | 'gemini' | 'images'
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

  const requestOverrides = {
    baseUrl: form.baseUrl || undefined,
    proxyUrl: form.proxyUrl || undefined,
    insecureSkipVerify: form.insecureSkipVerify || undefined,
    customHeaders: Object.keys(form.customHeaders).length > 0 ? { ...form.customHeaders } : undefined,
  }

  // 每个 unchecked key 并发独立请求
  const keyPromises = uncheckedKeys.map(async (apiKey) => {
    keyModelsStatus.value.set(apiKey, { loading: true, success: false })

    try {
      let response: any
      const id = channelId ?? 0
      const request = { key: apiKey, ...requestOverrides }

      switch (modelsApiType) {
        case 'messages':
          response = await apiService.getChannelModels(id, request)
          break
        case 'responses':
          response = await apiService.getResponsesChannelModels(id, request)
          break
        case 'chat':
          response = await apiService.getChatChannelModels(id, request)
          break
        case 'images':
          response = await apiService.getImagesChannelModels(id, request)
          break
        case 'gemini':
          response = await apiService.getGeminiChannelModels(id, request)
          break
      }

      keyModelsStatus.value.set(apiKey, {
        loading: false,
        success: true,
        statusCode: 200,
        modelCount: response.data.length
      })
      return response.data as { id: string }[]
    } catch (error) {
      let errorMsg = t('addChannel.unknownError')
      let statusCode = 0
      if (error instanceof ApiError) {
        errorMsg = error.message
        statusCode = error.status
      } else if (error instanceof Error) {
        errorMsg = error.message
      }
      keyModelsStatus.value.set(apiKey, {
        loading: false,
        success: false,
        statusCode,
        error: errorMsg
      })
      return [] as { id: string }[]
    }
  })

  try {
    const results = await Promise.all(keyPromises)

    const allModels = new Set<string>(targetModelOptions.value.map(opt => opt.value))
    results.forEach(models => models.forEach(m => allModels.add(m.id)))

    targetModelOptions.value = sortModelNamesDesc(Array.from(allModels)).map(id => ({ title: id, value: id }))

    const allFailed = candidateKeys.every(key => {
      const s = keyModelsStatus.value.get(key)
      return s && !s.success
    })
    if (allFailed) {
      fetchModelsError.value = t('addChannel.allApiKeysModelsFailed')
    }
  } finally {
    fetchingModels.value = false
  }
}

// 辅助函数：更新表单字段
const updateForm = (partial: Record<string, any>) => {
  Object.assign(form, partial)
}

// 辅助函数：同步上游模型
const syncUpstreamModels = () => {
  fetchTargetModels()
}

// 辅助函数：应用预设
const applyPreset = (presetName: string) => {
  // 根据预设名称判断调用哪个函数
  if (presetName === 'gpt-5.5' || presetName === 'gpt-5.4') {
    applyModelMappingPreset(presetName)
  } else if (form.serviceType === 'claude') {
    applyClaudeChannelPreset(presetName as 'mimo' | 'deepseek' | 'minimax')
  } else if (props.channelType === 'responses') {
    applyCodexResponsesChannelPreset(presetName as 'mimo' | 'deepseek' | 'minimax')
  } else {
    applyClaudeChannelPreset(presetName as 'mimo' | 'deepseek' | 'minimax')
  }
}



const handleSubmit = async () => {
  if (!formRef.value) return

  const { valid } = await formRef.value.validate()
  if (!valid) return
  if (modelCapabilitiesError.value) return

  // 将模型映射行同步到 form 对象
  syncModelMappingToForm()

  const channelData = buildSubmitPayload()

  emit('save', channelData)
}

const handleCancel = () => {
  emit('update:show', false)
  resetForm()
}

const PAYLOAD_KEYS = [
  'name', 'serviceType', 'baseUrl', 'baseUrls', 'website', 'insecureSkipVerify',
  'lowQuality', 'injectDummyThoughtSignature', 'stripThoughtSignature', 'description',
  'apiKeys', 'modelMapping', 'modelCapabilities', 'defaultCapability', 'allowUnknownContext',
  'reasoningMapping', 'reasoningParamStyle', 'textVerbosity',
  'fastMode', 'customHeaders', 'proxyUrl', 'requestTimeoutMs', 'responseHeaderTimeoutMs', 'streamFirstContentTimeoutMs', 'streamInactivityTimeoutMs', 'streamToolCallIdleTimeoutMs', 'routePrefix', 'supportedModels',
  'rateLimitRpm', 'rateLimitWindowMinutes', 'rateLimitMaxConcurrent', 'rateLimitAutoFromHeaders',
  'autoBlacklistBalance', 'normalizeMetadataUserId', 'stripBillingHeader', 'passbackThinkingBlocks', 'stripEmptyTextBlocks', 'normalizeSystemRoleToTopLevel', 'codexNativeToolPassthrough',
  'codexToolCompat', 'normalizeNonstandardChatRoles', 'stripCodexClientTools', 'stripImageGenerationTool', 'convertImageUrlToB64Json'
] as const

function extractPayloadFields(channel: Channel): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const key of PAYLOAD_KEYS) {
    if (key in channel) {
      result[key] = channel[key as keyof Channel]
    }
  }
  return result
}

const handleTestCapability = async () => {
  if (props.channel?.index === undefined || props.channel?.index === null) {
    return
  }

  if (!formRef.value) return
  const { valid } = await formRef.value.validate()
  if (!valid) return
  if (modelCapabilitiesError.value) return

  const channelData = buildSubmitPayload()
  const original = extractPayloadFields(props.channel)
  const hasChanges = JSON.stringify(channelData) !== JSON.stringify(original)

  if (hasChanges) {
    emit('save', channelData, { triggerCapabilityTest: true })
  } else {
    emit('testCapability', props.channel.index)
  }
}

// 监听props变化
watch(
  () => props.show,
  newShow => {
    if (newShow) {
      dialogMode.value = props.channel ? 'edit' : 'create'

      // 无论是编辑还是新增，都先清理密钥错误状态
      apiKeyError.value = ''
      duplicateKeyIndex.value = -1
      localRestoredKeys.value = new Set<string>()

      if (dialogMode.value === 'edit' && props.channel) {
        // 编辑模式：使用完整表单
        loadChannelData(props.channel)
      } else {
        // 添加模式：固定使用快速添加
        resetForm()
      }

      // dialog 渲染完成后绑定滚动监听，同步左侧导航高亮
      nextTick(() => {
        detachScrollListener()
        scrollRoot = document.querySelector('.content-area')
        if (!scrollRoot) return
        scrollHandler = () => updateActiveSectionFromScroll()
        scrollRoot.addEventListener('scroll', scrollHandler, { passive: true })
        updateActiveSectionFromScroll()
      })
    } else {
      detachScrollListener()
    }
  }
)

watch(
  () => props.channel,
  (newChannel, oldChannel) => {
    const action = resolveChannelWatcherAction({
      show: props.show,
      newChannel,
      oldChannel,
    })

    if (action === 'load-edit-channel' && newChannel) {
      dialogMode.value = 'edit'
      loadChannelData(newChannel)
      return
    }

    if (action === 'reset-new-form') {
      dialogMode.value = 'create'
      resetForm()
    }
  }
)

watch(
  () => form.baseUrl,
  value => {
    if (formBaseUrlPreviewTimer !== null) {
      window.clearTimeout(formBaseUrlPreviewTimer)
    }
    formBaseUrlPreviewTimer = window.setTimeout(() => {
      formBaseUrlPreview.value = value
    }, 200)
  },
  { immediate: true }
)

watch(
  () => JSON.stringify({
    baseUrl: form.baseUrl,
    baseUrls: form.baseUrls,
    apiKeys: form.apiKeys,
    proxyUrl: form.proxyUrl,
    insecureSkipVerify: form.insecureSkipVerify,
    customHeaders: form.customHeaders,
    serviceType: form.serviceType,
    routePrefix: form.routePrefix,
  }),
  () => {
    targetModelOptions.value = []
    keyModelsStatus.value.clear()
    hasTriedFetchModels.value = false
    fetchModelsError.value = ''
  }
)

// ESC键监听 & Cmd/Ctrl+Enter 确认
const handleKeydown = (event: Event) => {
  const keyboardEvent = event as KeyboardEvent
  if (!props.show) return

  if (keyboardEvent.key === 'Escape') {
    keyboardEvent.preventDefault()
    handleCancel()
    return
  }

  // Cmd/Ctrl+Enter 确认提交
  if (keyboardEvent.key === 'Enter' && (keyboardEvent.metaKey || keyboardEvent.ctrlKey) && !keyboardEvent.shiftKey) {
    keyboardEvent.preventDefault()
    handleSubmit()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  detachScrollListener()
  if (formBaseUrlPreviewTimer !== null) {
    window.clearTimeout(formBaseUrlPreviewTimer)
  }
})
</script>

<style scoped>
/* 基础URL下方的提示区域 - 固定高度防止布局跳动 */
.base-url-hint {
  min-height: 20px;
  padding: 4px 12px 8px;
  line-height: 1.5;
}



/* 高级选项中的右侧开关行 */
.advanced-switch-row {
  min-height: 56px;
}

.advanced-switch-row :deep(.v-selection-control) {
  justify-content: flex-end;
  margin-inline-start: 16px;
}

.channel-config-select {
  flex: 0 0 220px;
}

.stream-timeout-strategy-toggle {
  display: flex;
  flex-wrap: wrap;
  width: 100%;
}

.stream-timeout-strategy-toggle :deep(.v-btn) {
  flex: 1 1 120px;
  min-width: 0;
}

@media (max-width: 600px) {
  .channel-config-select {
    flex-basis: 100%;
  }
}

/* 暗色模式彩色按钮：深色文字，保证对比度 */
:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-primary),
:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-secondary),
:deep(.v-theme--dark .v-btn.v-btn--variant-flat.v-btn--color-success) {
  color: #1e1b4b !important;
}

:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-primary) {
  background-color: rgba(129, 140, 248, 0.92) !important;
}

:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-secondary) {
  background-color: rgba(167, 139, 250, 0.92) !important;
}

:deep(.v-theme--dark .v-btn.v-btn--variant-flat.v-btn--color-success) {
  background-color: rgba(52, 211, 153, 0.92) !important;
}

/* 超时配置 - Neo-Brutalism 风格（沿用调校台设计） */
.timeout-control-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0;
  margin-bottom: 16px;
  border: 2px solid rgb(var(--v-theme-on-surface));
  background: rgb(var(--v-theme-surface));
}

.v-theme--dark .timeout-control-grid {
  border-color: rgba(255, 255, 255, 0.6);
}

.timeout-control {
  padding: 12px 14px;
  position: relative;
  transition: opacity 0.2s ease;
}

.timeout-control--disabled {
  opacity: 0.4;
}

.timeout-control:not(:last-child)::after {
  content: '';
  position: absolute;
  right: 0;
  top: 8px;
  bottom: 8px;
  width: 2px;
  background: rgb(var(--v-theme-on-surface));
  opacity: 0.18;
}

.v-theme--dark .timeout-control:not(:last-child)::after {
  background: rgba(255, 255, 255, 0.6);
}

.timeout-control-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  gap: 6px;
}

.timeout-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: rgb(var(--v-theme-on-surface) / 70%);
  text-transform: uppercase;
  letter-spacing: 0;
  line-height: 1.3;
}

.timeout-value {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.8125rem;
  font-weight: 700;
  color: rgb(var(--v-theme-primary));
  padding: 2px 8px;
  border: 2px solid rgb(var(--v-theme-on-surface));
  background: rgb(var(--v-theme-surface));
  flex-shrink: 0;
  min-width: 40px;
  text-align: center;
}

.v-theme--dark .timeout-value {
  border-color: rgba(255, 255, 255, 0.5);
}

.timeout-slider {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 8px;
  border-radius: 0;
  border: 2px solid rgb(var(--v-theme-on-surface) / 20%);
  background: rgb(var(--v-theme-on-surface) / 8%);
  outline: none;
  cursor: pointer;
}

.timeout-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 20px;
  height: 20px;
  border-radius: 0;
  background: rgb(var(--v-theme-primary));
  cursor: pointer;
  border: 2px solid rgb(var(--v-theme-on-surface));
  box-shadow: 2px 2px 0 0 rgb(var(--v-theme-on-surface));
  transition: box-shadow 0.1s ease;
}

.timeout-slider::-webkit-slider-thumb:hover {
  transform: translate(-1px, -1px);
  box-shadow: 3px 3px 0 0 rgb(var(--v-theme-on-surface));
}

.timeout-slider::-moz-range-thumb {
  width: 20px;
  height: 20px;
  border-radius: 0;
  background: rgb(var(--v-theme-primary));
  cursor: pointer;
  border: 2px solid rgb(var(--v-theme-on-surface));
  box-shadow: 2px 2px 0 0 rgb(var(--v-theme-on-surface));
}

.timeout-slider:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.v-theme--dark .timeout-slider {
  border-color: rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.06);
}

.v-theme--dark .timeout-slider::-webkit-slider-thumb {
  border-color: rgba(255, 255, 255, 0.6);
  box-shadow: 2px 2px 0 0 rgba(255, 255, 255, 0.6);
}

.v-theme--dark .timeout-slider::-moz-range-thumb {
  border-color: rgba(255, 255, 255, 0.6);
  box-shadow: 2px 2px 0 0 rgba(255, 255, 255, 0.6);
}

.timeout-range {
  display: flex;
  justify-content: space-between;
  font-size: 0.6875rem;
  color: rgb(var(--v-theme-on-surface) / 50%);
  margin-top: 4px;
}

@media (max-width: 768px) {
  .timeout-control-grid {
    grid-template-columns: 1fr;
  }

  .timeout-control:not(:last-child)::after {
    display: none;
  }
}

/* 模型映射区域样式优化 */
.uppercase-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: rgba(var(--v-theme-on-surface), 0.6);
}

.mapping-container {
  background: rgba(var(--v-border-color), 0.03);
  border: 1px solid rgba(var(--v-border-color), 0.08);
}

.add-mapping-row {
  background: rgba(var(--v-theme-surface), 0.8);
  border: 1px solid rgba(var(--v-border-color), 0.15);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
}

.mapping-item {
  background: rgb(var(--v-theme-surface)) !important;
  border: 1px solid rgba(var(--v-border-color), 0.12) !important;
  transition: all 0.2s ease-in-out;
}

.mapping-item:hover {
  border-color: rgba(var(--v-theme-primary), 0.3) !important;
  box-shadow: 0 4px 12px rgba(var(--v-theme-primary), 0.04);
}

.mapping-item:hover .arrow-icon {
  transform: translateX(2px);
  opacity: 0.9;
}

.model-badge {
  min-width: 130px;
  max-width: 160px;
  background: rgba(var(--v-border-color), 0.05);
  border: 1px solid rgba(var(--v-border-color), 0.1);
  height: 40px;
}

.badge-title {
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: rgba(var(--v-theme-on-surface), 0.4);
  line-height: 1;
  margin-bottom: 2px;
}

.inner-label {
  position: absolute;
  top: -6px;
  left: 10px;
  background: rgb(var(--v-theme-surface));
  padding: 0 4px;
  z-index: 2;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: rgba(var(--v-theme-on-surface), 0.4);
}

.target-wrapper {
  position: relative;
}

.model-name {
  font-size: 13px;
  font-weight: 600;
  color: rgba(var(--v-theme-on-surface), 0.85);
}

.arrow-icon {
  opacity: 0.5;
  transition: transform 0.2s, opacity 0.2s;
}

.font-mono {
  font-family: 'SF Mono', 'Fira Code', Monaco, Consolas, monospace !important;
}

/* 垂直导航布局 */
.content-row {
  display: flex;
  height: 100%;
  min-height: 0;
}

.nav-sidebar {
  width: 220px;
  min-width: 220px;
  flex-shrink: 0;
  border-right: 1px solid rgba(var(--v-border-color), 0.12);
  background: rgba(var(--v-theme-surface-variant), 0.3);
  overflow-y: auto;
  padding: 16px 8px;
}

.nav-sidebar-title {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.4);
  padding: 0 12px 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 12px 12px;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  letter-spacing: normal;
  text-transform: none;
  color: rgba(var(--v-theme-on-surface), 0.7);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
  text-align: left;
}

.nav-item:hover {
  background: rgba(var(--v-theme-primary), 0.06);
  color: rgb(var(--v-theme-on-surface));
}

.nav-item--active {
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
  font-weight: 600;
}

.nav-item--active .nav-item-icon {
  color: rgb(var(--v-theme-primary));
}

.nav-item-icon {
  margin-right: 8px;
  opacity: 0.7;
}

.nav-item--active .nav-item-icon {
  opacity: 1;
}

.content-area {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
}

@media (max-width: 960px) {
  .content-row {
    flex-direction: column;
  }

  .nav-sidebar {
    display: none;
  }
}
</style>
