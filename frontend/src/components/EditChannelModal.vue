<template>
  <v-dialog :model-value="show" max-width="1200" persistent scrollable @update:model-value="$emit('update:show', $event)">
    <v-card rounded="lg" class="add-channel-dialog channel-editor-dialog">
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
      <v-card-text class="pa-0 channel-editor-body">
        <!-- 左侧导航 + 右侧面板 -->
        <div class="content-row">
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

            <!-- 身份认证 -->
            <section :ref="(el: any) => setSectionRef('auth', el)" data-section-id="auth" class="pa-6 scroll-mt-4">
              <ApiKeyManagementSection
                :api-keys="form.apiKeys"
                :disabled-keys="disabledKeys"
                :key-models-status="keyModelsStatus"
                :is-editing="isEditing"
                :restoring-key="restoringKey"
                :service-type="form.serviceType"
                :channel-id="props.channel?.index"
                @update:api-keys="form.apiKeys = $event"
                @restore-key="restoreDisabledKey"
              />
            </section>

            <!-- 模型重定向（模型映射 + Vision 回退 + 模型过滤） -->
            <section :ref="(el: any) => setSectionRef('redirect', el)" data-section-id="redirect" class="pa-6 scroll-mt-4">
              <ModelMappingSection
                v-if="form.serviceType"
                :mapping-rows="modelMappingRows"
                :source-model-options="sourceModelOptions"
                :target-model-options="targetModelOptions"
                :fetching-models="fetchingModels"
                :source-mapping-error="sourceMappingError"
                :fetch-models-error="fetchModelsError"
                :model-mapping-hint="modelMappingHint"
                :target-model-placeholder="targetModelPlaceholder"
                :show-model-mapping-presets="showModelMappingPresets"
                :show-messages-open-a-i-channel-presets="showMessagesOpenAIChannelPresets"
                :show-claude-channel-presets="showClaudeChannelPresets"
                :show-codex-responses-channel-presets="showCodexResponsesChannelPresets"
                :supports-reasoning-mapping-options="supportsReasoningMappingOptions"
                :reasoning-effort-options="reasoningEffortOptions"
                @update:mapping-rows="modelMappingRows = ($event as any)"
                @sync-upstream="syncUpstreamModels"
                @apply-preset="applyPreset"
                @menu-update="onMenuUpdate"
                @target-edit-start="startMappingTargetEdit"
                @target-edit-end="finishMappingTargetEdit"
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
                          @focus="startMappingTargetEdit(); ensureTargetModelsLoaded()"
                          @blur="finishMappingTargetEdit"
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
                <ModelCapabilitySection
                  v-model:rows="form.modelCapabilityRows"
                  :target-model-options="targetModelOptions"
                  :mapped-target-models="mappedTargetModels"
                  :fetching-models="fetchingModels"
                  :fetch-models-error="fetchModelsError"
                  :error="modelCapabilitiesError"
                  @sync-upstream="syncUpstreamModels"
                  @menu-update="onMenuUpdate"
                />
              </div>
            </section>

            <!-- 高级选项 -->
            <section :ref="(el: any) => setSectionRef('advanced', el)" data-section-id="advanced" class="pa-6 scroll-mt-4">
              <AdvancedOptionsSection
                :form="form"
                :channel-type="props.channelType"
                :supports-chat-role-normalization="supportsChatRoleNormalization"
                :supports-open-a-i-advanced-options="supportsOpenAIAdvancedOptions"
                :reasoning-param-style-options="reasoningParamStyleOptions"
                :text-verbosity-options="textVerbosityOptions"
                :diagnosing="diagnosingCompat"
                @update:form="updateForm"
                @menu-update="onMenuUpdate"
                @diagnose="handleDiagnoseCompat"
              />
            </section>

            <!-- 自定义参数（自定义请求头 + 流式超时） -->
            <section :ref="(el: any) => setSectionRef('custom', el)" data-section-id="custom" class="pa-6 scroll-mt-4">
              <CustomHeadersSection
                :headers="customHeadersArray"
                @update:headers="updateCustomHeaders"
              />

              <div class="mt-6">
                <TransportConfigGroup :form="form" @update:field="(field, value) => updateForm({ [field]: value })" />
              </div>

              <div class="mt-6">
                <StreamTimeoutSection
                  :request-timeout-ms="form.requestTimeoutMs"
                  :response-header-timeout-ms="form.responseHeaderTimeoutMs"
                  :selected-strategy="selectedStreamTimeoutStrategy"
                  :first-content-enabled="form.streamFirstContentTimeoutEnabled"
                  :first-content-ms="form.streamFirstContentTimeoutMs"
                  :inactivity-enabled="form.streamInactivityTimeoutEnabled"
                  :inactivity-ms="form.streamInactivityTimeoutMs"
                  :tool-call-idle-enabled="form.streamToolCallIdleTimeoutEnabled"
                  :tool-call-idle-ms="form.streamToolCallIdleTimeoutMs"
                  @update:request-timeout-ms="form.requestTimeoutMs = $event"
                  @update:response-header-timeout-ms="form.responseHeaderTimeoutMs = $event"
                  @apply-strategy="applyStreamTimeoutStrategy"
                  @update:first-content-ms="form.streamFirstContentTimeoutMs = $event"
                  @update:inactivity-ms="form.streamInactivityTimeoutMs = $event"
                  @update:tool-call-idle-ms="form.streamToolCallIdleTimeoutMs = $event"
                />
              </div>

              <div class="mt-6">
                <RateLimitGroup :form="form" @update:field="(field, value) => updateForm({ [field]: value })" />
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
import { buildExpectedRequestUrls } from '../utils/expectedRequestUrls'
import { supportsAdvancedChannelOptions, supportsReasoningMapping } from '../utils/channelAdvancedOptions'
import {
  buildChannelPayload,
  createModelCapabilityRow,
  modelCapabilitiesToRows,
  modelCapabilityRowsToRecord,
  normalizeSelectableString,
  resolveBuiltinUpstreamModelCapability,
  type ModelCapabilityRow,
} from '../utils/channelPayload'
import {
  resolveChannelWatcherAction,
  syncBaseUrlsFormState,
  filterValidSupportedModelPatterns,
  parseSupportedModelInput
} from '../utils/add-channel-modal-state'
import { streamTimeoutPresets } from '../utils/streamTimeoutPresets'
import { sortModelNamesDesc } from '../utils/modelPriority'
import { claudeMessagesPresets } from '../generated/claudeMessagesPresets'
import { codexResponsesPresets } from '../generated/codexResponsesPresets'
import { openaiMessagesPresets } from '../generated/openaiMessagesPresets'
import { useI18n } from '../i18n'

// 子组件导入
import AddChannelHeader from './edit-channel/AddChannelHeader.vue'
import AddChannelSidebarNav from './edit-channel/AddChannelSidebarNav.vue'
import BasicInfoSection from './edit-channel/BasicInfoSection.vue'
import ApiKeyManagementSection from './edit-channel/ApiKeyManagementSection.vue'
import ModelMappingSection from './edit-channel/ModelMappingSection.vue'
import ModelCapabilitySection from './edit-channel/ModelCapabilitySection.vue'
import SupportedModelsFilter from './edit-channel/SupportedModelsFilter.vue'
import CustomHeadersSection from './edit-channel/CustomHeadersSection.vue'
import StreamTimeoutSection from './edit-channel/StreamTimeoutSection.vue'
import AdvancedOptionsSection from './edit-channel/AdvancedOptionsSection.vue'
import TransportConfigGroup from './edit-channel/TransportConfigGroup.vue'
import RateLimitGroup from './edit-channel/RateLimitGroup.vue'

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
  success: [message: string]
}>()
const { t } = useI18n()
const apiService = new ApiService()

// 主题
const theme = useTheme()

// 表单引用
const formRef = ref()

const defaultServiceTypeValueFallback = (): 'openai' | 'gemini' | 'claude' | 'responses' | 'copilot' => {
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
  { id: 'auth', label: t('channelEditor.nav.auth') },
  { id: 'redirect', label: t('channelEditor.nav.redirect') },
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
const isAnySelectMenuOpen = ref(false)
const suppressDialogEscapeUntil = ref(0)

const onMenuUpdate = (open: boolean) => {
  isAnySelectMenuOpen.value = open
  if (!open) {
    suppressDialogEscapeUntil.value = Date.now() + 150
  }
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
    { title: 'Responses (Codex)', value: 'responses' },
    { title: 'GitHub Copilot', value: 'copilot' }
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
      { title: 'gpt-5.6-sol', value: 'gpt-5.6-sol' },
      { title: 'gpt-5.6-terra', value: 'gpt-5.6-terra' },
      { title: 'gpt-5.6-luna', value: 'gpt-5.6-luna' },
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
      { title: 'gpt-5.6-sol', value: 'gpt-5.6-sol' },
      { title: 'gpt-5.6-terra', value: 'gpt-5.6-terra' },
      { title: 'gpt-5.6-luna', value: 'gpt-5.6-luna' },
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

const modelMappingPresets = openaiMessagesPresets

const applyModelMappingPreset = (preset: keyof typeof modelMappingPresets) => {
  const presetConfig = modelMappingPresets[preset]
  form.modelMapping = { ...presetConfig.modelMapping }
  form.fastMode = presetConfig.fastMode
  form.textVerbosity = presetConfig.textVerbosity

  if (supportsOpenAIAdvancedOptions.value) {
    form.reasoningMapping = { ...presetConfig.reasoningMapping } as typeof form.reasoningMapping
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

const claudeChannelPresets = claudeMessagesPresets

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
  form.modelMapping = { ...presetConfig.modelMapping }
  form.reasoningMapping = { ...presetConfig.reasoningMapping } as typeof form.reasoningMapping
  form.reasoningParamStyle = presetConfig.reasoningParamStyle as typeof form.reasoningParamStyle
  syncModelMappingRowsFromForm()
}

// Codex Responses 转 OpenAI 兼容上游的一键预设（MiMo / DeepSeek）
const showCodexResponsesChannelPresets = computed(() => {
  return props.channelType === 'responses' && supportsOpenAIAdvancedOptions.value
})

const applyCodexResponsesChannelPreset = (preset: string) => {
  const presetConfig = codexResponsesPresets[preset.toLowerCase()]
  if (!presetConfig) return

  form.modelMapping = { ...presetConfig.modelMapping }
  form.reasoningMapping = { ...presetConfig.reasoningMapping } as typeof form.reasoningMapping
  form.reasoningParamStyle = presetConfig.reasoningParamStyle as typeof form.reasoningParamStyle
  if (presetConfig.serviceType) {
    form.serviceType = presetConfig.serviceType as typeof form.serviceType
  }
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
// 表单数据：balanced 预设值作为渠道级默认回退值
const defaultStreamTimeouts = { ...streamTimeoutPresets.balanced }

type StreamTimeoutPresetKey = 'gentle' | 'balanced' | 'aggressive' | 'custom'

const form = reactive({
  name: '',
  serviceType: '' as 'openai' | 'gemini' | 'claude' | 'responses' | 'copilot' | '',
  authHeader: 'auto' as 'auto' | 'bearer' | 'x-api-key' | '',
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
  apiKeyConfigs: undefined as Channel['apiKeyConfigs'],
  modelMapping: {} as Record<string, string>,
  modelCapabilitiesText: '',
  modelCapabilityRows: [] as ModelCapabilityRow[],
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
let capabilityRowIdCounter = 0
const nextCapabilityRowId = () => ++capabilityRowIdCounter

const incompleteMappedTargetSuffix = /[._:/-]$/
const isCompleteMappedTargetModel = (model: string) => !!model && !incompleteMappedTargetSuffix.test(model)
const hasNoVisionRows = computed(() => modelMappingRows.value.some(row => row.noVision && row.target.trim()))
const mappedTargetModels = computed(() => {
  const seen = new Set<string>()
  const models = [
    ...modelMappingRows.value.map(row => normalizeSelectableString(row.target).trim()),
    normalizeSelectableString(form.visionFallbackModel).trim(),
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

function resetTransientUiState() {
  sourceMappingError.value = ''
  localRestoredKeys.value = new Set<string>()
  restoringKey.value = ''
  errors.name = ''
  errors.serviceType = ''
  errors.baseUrl = ''
  errors.website = ''
  formBaseUrlPreview.value = ''
}

// 源模型名验证错误
const sourceMappingError = ref('')

// 目标模型列表只展示上游 /models 真实返回值；手动输入由 combobox 自身支持
const targetModelOptions = ref<Array<{ title: string; value: string }>>([])
const upstreamTargetModels = ref<string[]>([])

const normalizeTargetModelNames = (models: string[]) => {
  const byLowercaseModel = new Map<string, string>()
  for (const model of models) {
    const trimmed = String(model || '').trim()
    if (!trimmed) continue
    const key = trimmed.toLowerCase()
    const existing = byLowercaseModel.get(key)
    if (!existing || trimmed === key) {
      byLowercaseModel.set(key, trimmed)
    }
  }
  return sortModelNamesDesc(Array.from(byLowercaseModel.values()))
}

const toTargetModelOptions = (models: string[]) => {
  return normalizeTargetModelNames(models).map((id: string) => ({ title: id, value: id }))
}

const resetTargetModelOptions = () => {
  upstreamTargetModels.value = []
  targetModelOptions.value = []
}

const mergeUpstreamTargetModelOptions = (models: string[]) => {
  upstreamTargetModels.value = normalizeTargetModelNames([...upstreamTargetModels.value, ...models])
  targetModelOptions.value = toTargetModelOptions(upstreamTargetModels.value)
}

resetTargetModelOptions()
const fetchingModels = ref(false)
const fetchModelsError = ref('')

// API Key 的 models 状态管理
interface KeyModelsStatus {
  loading: boolean
  success: boolean
  statusCode?: number
  error?: string
  modelCount?: number
}
const keyModelsStatus = ref<Map<string, KeyModelsStatus>>(new Map())

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
const modelCapabilitiesError = computed(() => {
  return modelCapabilityRowsToRecord(form.modelCapabilityRows) === null
    ? t('addChannel.modelCapabilitiesRowsInvalid')
    : ''
})

const syncModelCapabilitiesFromMapping = () => {
  const existingModels = new Set(
    form.modelCapabilityRows
      .map(row => normalizeSelectableString(row.model).trim().toLowerCase())
      .filter(Boolean)
  )
  const rowsToAdd = mappedTargetModels.value
    .filter(isCompleteMappedTargetModel)
    .filter(model => !existingModels.has(model.toLowerCase()))
    .map(model => {
      const builtin = resolveBuiltinUpstreamModelCapability(model)
      return createModelCapabilityRow(
        nextCapabilityRowId(),
        model,
        builtin?.capability,
        builtin ? 'builtin' : 'custom',
        builtin?.pattern || '',
      )
    })
  if (!rowsToAdd.length) return
  form.modelCapabilityRows = [...form.modelCapabilityRows, ...rowsToAdd]
}

const syncModelCapabilitiesFromMappingWhenIdle = () => {
  if (isMappingTargetEditing.value) {
    hasPendingModelCapabilitySync.value = true
    return
  }
  hasPendingModelCapabilitySync.value = false
  syncModelCapabilitiesFromMapping()
}

const startMappingTargetEdit = () => {
  isMappingTargetEditing.value = true
}

const finishMappingTargetEdit = () => {
  if (!isMappingTargetEditing.value) return
  isMappingTargetEditing.value = false
  if (!hasPendingModelCapabilitySync.value) return
  hasPendingModelCapabilitySync.value = false
  nextTick(syncModelCapabilitiesFromMapping)
}

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

const handleSupportedModelsChange = (values: Array<string | { title: string; value: string }>) => {
  // 用户可能把多条规则用顿号/逗号粘进同一项，这里统一按分隔符拆分
  const normalizedValues = values
    .map(normalizeSelectableString)
    .flatMap(parseSupportedModelInput)

  const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(normalizedValues)
  form.supportedModels = validPatterns
  supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
}

const normalizeModelCapabilities = (record: Channel['modelCapabilities'] = {}): Channel['modelCapabilities'] => {
  return Object.fromEntries(Object.entries(record).sort(([a], [b]) => a.localeCompare(b)))
}

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

// 表单操作
const resetForm = () => {
  resetTransientUiState()
  form.name = ''
  form.serviceType = props.channelType === 'images' ? 'openai' : ''
  form.authHeader = 'auto'
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
  form.apiKeyConfigs = undefined
  form.modelMapping = {}
  form.modelCapabilitiesText = ''
  form.modelCapabilityRows = []
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

  // 清空模型缓存和状态
  resetTargetModelOptions()
  fetchingModels.value = false
  fetchModelsError.value = ''
  keyModelsStatus.value.clear()

  }

const loadChannelData = (channel: Channel) => {
  resetTransientUiState()
  form.name = channel.name
  form.serviceType = props.channelType === 'images' ? 'openai' : channel.serviceType
  form.authHeader = channel.authHeader || 'auto'
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
  form.apiKeyConfigs = channel.apiKeyConfigs
    ? channel.apiKeyConfigs.map(cfg => ({
        ...cfg,
        models: cfg.models ? [...cfg.models] : undefined,
      }))
    : undefined

  form.modelMapping = { ...(channel.modelMapping || {}) }
  form.modelCapabilitiesText = Object.keys(channel.modelCapabilities || {}).length > 0
    ? JSON.stringify(normalizeModelCapabilities(channel.modelCapabilities), null, 2)
    : ''
  form.modelCapabilityRows = modelCapabilitiesToRows(channel.modelCapabilities || {}, nextCapabilityRowId)
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
  form.rateLimitAutoFromHeaders = channel.rateLimitAutoFromHeaders !== false
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

  // 清空模型缓存和状态（切换渠道时重置）
  resetTargetModelOptions()
  fetchingModels.value = false
  fetchModelsError.value = ''
  keyModelsStatus.value.clear()

  // 如果有模型映射配置，主动预加载模型列表
  if (channel.modelMapping && Object.keys(channel.modelMapping).length > 0) {
    nextTick(() => {
      fetchTargetModels()
    })
  }
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
    emit('error', error instanceof Error ? error.message : 'Restore failed')
  } finally {
    restoringKey.value = ''
  }
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

const ensureTargetModelsLoaded = () => {
  if (upstreamTargetModels.value.length === 0) {
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
    authHeader: form.authHeader && form.authHeader !== 'auto' ? form.authHeader : undefined,
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
    mergeUpstreamTargetModelOptions(results.flatMap(models => models.map(m => m.id)))

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
    applyCodexResponsesChannelPreset(presetName)
  } else {
    applyClaudeChannelPreset(presetName as 'mimo' | 'deepseek' | 'minimax')
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return

  syncModelCapabilitiesFromMapping()

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
  'apiKeys', 'apiKeyConfigs', 'modelMapping', 'modelCapabilities', 'defaultCapability', 'allowUnknownContext',
  'reasoningMapping', 'reasoningParamStyle', 'textVerbosity',
  'fastMode', 'customHeaders', 'proxyUrl', 'authHeader', 'requestTimeoutMs', 'responseHeaderTimeoutMs', 'streamFirstContentTimeoutMs', 'streamInactivityTimeoutMs', 'streamToolCallIdleTimeoutMs', 'routePrefix', 'supportedModels',
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
  syncModelCapabilitiesFromMapping()
  const { valid } = await formRef.value.validate()
  if (!valid) return
  if (modelCapabilitiesError.value) return

  // 与 handleSubmit 保持一致：先把已添加的模型映射行同步到 form，
  // 否则刚点“添加”进列表、尚未保存的重定向不会进入 payload，导致能力测试漏掉这些改动
  syncModelMappingToForm()

  const channelData = buildSubmitPayload()
  const original = extractPayloadFields(props.channel)
  const hasChanges = JSON.stringify(channelData) !== JSON.stringify(original)

  if (hasChanges) {
    emit('save', channelData, { triggerCapabilityTest: true })
  } else {
    emit('testCapability', props.channel.index)
  }
}

const diagnosingCompat = ref(false)

const handleDiagnoseCompat = async () => {
  if (props.channel?.index === undefined || props.channel?.index === null) return
  if (props.channelType === 'images') return

  diagnosingCompat.value = true
  try {
    const type = props.channelType as 'messages' | 'chat' | 'responses' | 'gemini'
    const result = await apiService.diagnoseChannelCompat(type, props.channel.index)
    const applied: string[] = []
    for (const [key, val] of Object.entries(result.recommendations)) {
      if (val !== undefined && (form as Record<string, unknown>)[key] !== val) {
        updateForm({ [key]: val })
        applied.push(key)
      }
    }
    if (result.urlRecommendations?.recommended) {
      const current = result.urlRecommendations.current
      const recommended = result.urlRecommendations.recommended
      const lines = baseUrlsText.value.split('\n').map(line => line.trim()).filter(Boolean)
      const nextLines = lines.length > 0
        ? lines.map((line, index) => (index === 0 || line === current) ? recommended : line)
        : [recommended]
      baseUrlsText.value = Array.from(new Set(nextLines)).join('\n')
      applied.push('baseUrl')
    }
    emit('success', applied.length
      ? t('channelEditor.compat.diagnoseApplied', { count: applied.length })
      : t('channelEditor.compat.diagnoseNoChange'))
  } catch (e) {
    emit('error', e instanceof Error ? e.message : t('channelEditor.compat.diagnoseFailed'))
  } finally {
    diagnosingCompat.value = false
  }
}

// 监听props变化
watch(
  () => props.show,
  newShow => {
    if (newShow) {
      dialogMode.value = props.channel ? 'edit' : 'create'
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
  mappedTargetModels,
  () => {
    syncModelCapabilitiesFromMappingWhenIdle()
  }
)

watch(
  () => JSON.stringify({
    baseUrl: form.baseUrl,
    baseUrls: form.baseUrls,
    apiKeys: form.apiKeys,
    proxyUrl: form.proxyUrl,
    insecureSkipVerify: form.insecureSkipVerify,
    customHeaders: form.customHeaders,
    authHeader: form.authHeader,
    serviceType: form.serviceType,
    routePrefix: form.routePrefix,
  }),
  () => {
    resetTargetModelOptions()
    keyModelsStatus.value.clear()
    fetchModelsError.value = ''
  }
)

// ESC键监听 & Cmd/Ctrl+Enter 确认
const handleKeydown = (event: Event) => {
  const keyboardEvent = event as KeyboardEvent
  if (!props.show) return

  if (keyboardEvent.key === 'Escape') {
    if (isAnySelectMenuOpen.value || Date.now() < suppressDialogEscapeUntil.value) {
      keyboardEvent.preventDefault()
      keyboardEvent.stopPropagation()
      return
    }
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

<style scoped src="./edit-channel/edit-channel-modal.css"></style>
