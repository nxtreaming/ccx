<template>
  <div class="model-mapping-section">
    <v-card variant="outlined" rounded="lg">
      <v-card-title class="d-flex align-center justify-space-between pa-4 pb-2">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-swap-horizontal</v-icon>
          <span class="section-title">{{ t('channelEditor.mapping.redirect.title') }}</span>
        </div>
        <v-chip size="small" color="secondary" variant="tonal">
          {{ t('addChannel.autoConvertModelNames') }}
        </v-chip>
      </v-card-title>

      <v-card-text class="pt-2">
        <div class="text-body-2 text-medium-emphasis mb-4">
          {{ modelMappingHint }}
          <br/>
          <span class="text-caption text-primary">💡 {{ t('addChannel.modelHintTip') }}</span>
        </div>

        <!-- 预设按钮组 -->
        <div v-if="showModelMappingPresets" class="d-flex align-center flex-wrap ga-2 mb-4">
          <div class="text-caption text-medium-emphasis">{{ t('addChannel.oneClickSetup') }}</div>
          <v-btn
            size="small"
            variant="tonal"
            color="primary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'gpt-5.5')"
          >
            gpt-5.5
          </v-btn>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'gpt-5.4')"
          >
            gpt-5.4
          </v-btn>
          <v-btn
            v-if="showMessagesOpenAIChannelPresets"
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'mimo')"
          >
            MiMo
          </v-btn>
          <v-btn
            v-if="showMessagesOpenAIChannelPresets"
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'deepseek')"
          >
            DeepSeek
          </v-btn>
        </div>

        <div v-if="showClaudeChannelPresets" class="d-flex align-center flex-wrap ga-2 mb-4">
          <div class="text-caption text-medium-emphasis">{{ t('addChannel.oneClickSetup') }}</div>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'mimo')"
          >
            MiMo
          </v-btn>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'deepseek')"
          >
            DeepSeek
          </v-btn>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'minimax')"
          >
            MiniMax
          </v-btn>
        </div>

        <div v-if="showCodexResponsesChannelPresets" class="d-flex align-center flex-wrap ga-2 mb-4">
          <div class="text-caption text-medium-emphasis">{{ t('addChannel.oneClickSetup') }}</div>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'mimo')"
          >
            MiMo
          </v-btn>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'deepseek')"
          >
            DeepSeek
          </v-btn>
          <v-btn
            size="small"
            variant="tonal"
            color="secondary"
            prepend-icon="mdi-lightning-bolt"
            @click="$emit('apply-preset', 'minimax')"
          >
            MiniMax
          </v-btn>
        </div>

        <!-- 映射容器 -->
        <div class="mapping-container rounded-xl pa-3">
          <!-- 已配置映射列表 -->
          <div v-if="mappingRows.length">
            <div class="text-caption text-medium-emphasis mb-3 d-flex align-center justify-space-between px-1">
              <span class="uppercase-label">{{ t('channelEditor.mapping.configured.label') }}</span>
              <v-chip size="x-small" variant="flat" color="primary" class="font-weight-bold px-2 font-mono">
                {{ mappingRows.length }}
              </v-chip>
            </div>

            <div class="d-flex flex-column ga-2">
              <div
                v-for="(row, index) in mappingRows"
                :key="row.id"
                class="mapping-item d-flex align-center justify-space-between pa-3 rounded-lg"
              >
                <div class="d-flex align-center ga-3 flex-grow-1">
                  <!-- 源模型徽章 -->
                  <div class="model-badge source-badge pa-2 rounded-lg d-flex flex-column justify-center">
                    <span class="badge-title">SOURCE</span>
                    <span class="model-name text-truncate font-mono" :title="formatModelName(row.source)">
                      {{ formatModelName(row.source) || 'source-model' }}
                    </span>
                  </div>

                  <v-icon color="primary" class="arrow-icon" size="18">mdi-arrow-right</v-icon>

                  <!-- 目标模型输入包装 -->
                  <div class="target-wrapper flex-grow-1" style="position: relative;">
                    <span class="badge-title inner-label">TARGET</span>
                    <v-combobox
                      v-model="row.target"
                      :items="targetModelOptions"
                      :loading="fetchingModels"
                      density="compact"
                      variant="outlined"
                      hide-details
                      placeholder="target-model"
                      class="font-mono"
                      eager
                      @focus="$emit('sync-upstream')"
                      @update:menu="$emit('menu-update', $event)"
                    />
                  </div>

                  <!-- Reasoning 选择器 -->
                  <v-select
                    v-if="supportsOpenAIAdvancedOptions"
                    v-model="row.reasoning"
                    :items="[
                      { title: '无', value: '' },
                      { title: 'None', value: 'none' },
                      { title: 'Low', value: 'low' },
                      { title: 'Medium', value: 'medium' },
                      { title: 'High', value: 'high' },
                      { title: 'XHigh', value: 'xhigh' },
                      { title: 'Max', value: 'max' }
                    ]"
                    density="compact"
                    variant="outlined"
                    hide-details
                    placeholder="reasoning"
                    class="flex-shrink-0"
                    style="max-width: 120px;"
                  />
                </div>

                <!-- 操作按钮组 -->
                <div class="action-group d-flex align-center ga-1 ml-3">
                  <v-tooltip
                    :text="row.noVision ? t('addChannel.visionDisabled') : t('addChannel.visionEnabled')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tip }">
                      <v-btn
                        v-bind="tip"
                        size="small"
                        :color="row.noVision ? 'warning' : 'primary'"
                        icon
                        :variant="row.noVision ? 'tonal' : 'text'"
                        class="rounded-lg"
                        @click="toggleVision(index)"
                      >
                        <v-icon :size="16">{{ row.noVision ? 'mdi-eye-off' : 'mdi-eye' }}</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>

                  <v-btn
                    size="small"
                    color="error"
                    icon
                    variant="text"
                    @click="removeMapping(index)"
                  >
                    <v-icon :size="16">mdi-close</v-icon>
                  </v-btn>
                </div>
              </div>
            </div>
          </div>

          <!-- Vision 回退模型（由父组件通过 slot 注入） -->
          <slot name="vision-fallback" />

          <!-- 添加新映射 -->
          <div class="add-mapping-row d-flex align-center ga-3 pa-3 mt-3 rounded-lg">
            <v-combobox
              v-model="newMapping.source"
              :label="t('channelEditor.mapping.source.label')"
              :items="sourceModelOptions"
              variant="outlined"
              density="compact"
              hide-details
              class="flex-grow-1 font-mono"
              :placeholder="t('channelEditor.mapping.source.placeholder')"
              clearable
              :error="!!sourceMappingError"
              eager
              @update:model-value="handleSourceChange"
              @update:menu="$emit('menu-update', $event)"
              @keyup.enter="handleAddMapping"
            />

            <v-icon color="primary" size="18" class="arrow-icon">mdi-arrow-right</v-icon>

            <v-combobox
              v-model="newMapping.target"
              :label="t('channelEditor.mapping.target.label')"
              :placeholder="targetModelPlaceholder"
              :items="targetModelOptions"
              :loading="fetchingModels"
              variant="outlined"
              density="compact"
              hide-details
              class="flex-grow-1 font-mono"
              clearable
              eager
              @focus="$emit('sync-upstream')"
              @update:menu="$emit('menu-update', $event)"
              @keyup.enter="handleAddMapping"
            />

            <v-select
              v-if="supportsOpenAIAdvancedOptions"
              v-model="newMapping.reasoningEffort"
              :label="t('channelEditor.mapping.reasoningEffort.label')"
              :items="reasoningEffortOptions"
              variant="outlined"
              density="compact"
              hide-details
              clearable
              class="flex-shrink-0"
              style="min-width: 120px;"
              eager
              @update:menu="$emit('menu-update', $event)"
            />

            <v-btn
              color="primary"
              height="40"
              variant="flat"
              class="rounded-lg px-4"
              :disabled="!isMappingInputValid"
              @click="handleAddMapping"
            >
              <v-icon size="18" class="mr-1">mdi-plus</v-icon>
              {{ t('app.actions.add') }}
            </v-btn>
          </div>
        </div>

        <!-- 错误提示 -->
        <div v-if="sourceMappingError" class="text-error text-caption mt-2">
          {{ sourceMappingError }}
        </div>
        <div v-if="fetchModelsError" class="text-error text-caption mt-2">
          {{ fetchModelsError }}
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from '../../i18n'
import { normalizeSelectableString } from '../../utils/channelPayload'

interface MappingRow {
  id: number
  source: string
  target: string
  reasoning?: '' | 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
  noVision?: boolean
}

interface NewMapping {
  source: string | { title: string; value: string }
  target: string | { title: string; value: string }
  reasoningEffort?: string
}

interface Props {
  mappingRows: MappingRow[]
  sourceModelOptions: Array<{ title: string; value: string }>
  targetModelOptions: Array<{ title: string; value: string }>
  fetchingModels: boolean
  sourceMappingError: string
  fetchModelsError: string
  modelMappingHint: string
  targetModelPlaceholder: string
  showModelMappingPresets: boolean
  showMessagesOpenAIChannelPresets: boolean
  showClaudeChannelPresets: boolean
  showCodexResponsesChannelPresets: boolean
  supportsOpenAIAdvancedOptions: boolean
  reasoningEffortOptions: Array<{ title: string; value: string }>
}


const emit = defineEmits<{
  'update:mappingRows': [MappingRow[]]
  'sync-upstream': []
  'apply-preset': [string]
  'menu-update': [boolean]
}>()

const { t } = useI18n()

const newMapping = ref<NewMapping>({
  source: '',
  target: '',
  reasoningEffort: undefined,
})

const isMappingInputValid = computed(() => {
  return !!(normalizeSelectableString(newMapping.value.source).trim() && normalizeSelectableString(newMapping.value.target).trim())
})

const handleSourceChange = () => {
  // 源模型变化时的处理逻辑
}

const handleAddMapping = () => {
  if (!isMappingInputValid.value) return
  const source = normalizeSelectableString(newMapping.value.source).trim()
  const target = normalizeSelectableString(newMapping.value.target).trim()

  const row: MappingRow = {
    id: Date.now(),
    source,
    target,
    reasoning: (newMapping.value.reasoningEffort || '') as '' | 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max',
    noVision: false,
  }

  emit('update:mappingRows', [...normalizeMappingRows(props.mappingRows), row])

  // 重置输入
  newMapping.value = {
    source: '',
    target: '',
    reasoningEffort: undefined,
  }
}

const removeMapping = (index: number) => {
  const updated = normalizeMappingRows(props.mappingRows).filter((_, i) => i !== index)
  emit('update:mappingRows', updated)
}

const toggleVision = (index: number) => {
  const updated = normalizeMappingRows(props.mappingRows)
  updated[index] = { ...updated[index], noVision: !updated[index].noVision }
  emit('update:mappingRows', updated)
}

const normalizeMappingRows = (rows: MappingRow[]): MappingRow[] => rows.map(row => ({
  ...row,
  source: normalizeSelectableString(row.source),
  target: normalizeSelectableString(row.target),
}))

const formatModelName = (value: MappingRow['source']) => normalizeSelectableString(value)

const props = defineProps<Props>()
</script>

<style scoped>
.section-title {
  font-size: 1.125rem;
  font-weight: 600;
}

.font-mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Courier New', monospace;
}

.mapping-container {
  background: rgba(var(--v-theme-surface-variant), 0.3);
}

.add-mapping-row {
  background: rgba(var(--v-theme-primary), 0.05);
  border: 2px dashed rgba(var(--v-theme-primary), 0.3);
}

.arrow-icon {
  flex-shrink: 0;
}

.uppercase-label {
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 600;
}

.mapping-item {
  background: rgba(var(--v-theme-surface), 0.8);
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  transition: all 0.2s ease;
}

.mapping-item:hover {
  background: rgba(var(--v-theme-primary), 0.08);
  border-color: rgba(var(--v-theme-primary), 0.3);
}

.model-badge {
  min-width: 140px;
  background: rgba(var(--v-theme-surface-variant), 0.6);
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
}

.source-badge {
  background: linear-gradient(135deg, rgba(var(--v-theme-primary), 0.1), rgba(var(--v-theme-secondary), 0.05));
}

.badge-title {
  font-size: 0.625rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 2px;
}

.model-name {
  font-size: 0.8125rem;
  font-weight: 500;
  max-width: 120px;
}

.target-wrapper {
  position: relative;
}

.inner-label {
  position: absolute;
  top: -8px;
  left: 12px;
  z-index: 1;
  padding: 0 4px;
  background: rgb(var(--v-theme-surface));
  font-size: 0.625rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgba(var(--v-theme-on-surface), 0.6);
}

.action-group {
  flex-shrink: 0;
}
</style>
