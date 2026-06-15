<template>
  <div class="api-key-management-section">
    <v-card variant="outlined" rounded="lg" :color="hasConfigurableKeys ? undefined : 'error'">
      <v-card-title class="d-flex align-center justify-space-between pa-4 pb-2">
        <div class="d-flex align-center ga-2">
          <v-icon :color="hasConfigurableKeys ? 'primary' : 'error'">mdi-key</v-icon>
          <span class="section-title">{{ t('channelCard.apiKeyManagement') }} *</span>
          <v-chip v-if="!hasConfigurableKeys" size="x-small" color="error" variant="tonal">
            {{ t('channelEditor.auth.apiKeyRequired') }}
          </v-chip>
        </div>
        <v-chip size="small" color="info" variant="tonal">
          {{ t('addChannel.apiKeyLoadBalance') }}
        </v-chip>
      </v-card-title>

      <v-card-text class="pt-2">
        <!-- 现有密钥列表 -->
        <div v-if="apiKeys.length" class="mb-4">
          <v-list density="compact" class="bg-transparent">
            <v-list-item
              v-for="(key, index) in apiKeys"
              :key="index"
              class="mb-2"
              rounded="lg"
              variant="tonal"
              :color="duplicateKeyIndex === index ? 'error' : 'surface-variant'"
              :class="{ 'animate-pulse': duplicateKeyIndex === index }"
            >
              <template #prepend>
                <v-icon size="small" :color="duplicateKeyIndex === index ? 'error' : 'primary'">
                  {{ duplicateKeyIndex === index ? 'mdi-alert' : 'mdi-key' }}
                </v-icon>
              </template>

              <v-list-item-title>
                <div class="d-flex align-center justify-space-between">
                  <code class="text-caption">{{ maskApiKey(key) }}</code>
                  <div class="d-flex align-center ga-1">
                    <!-- Models 状态标签 -->
                    <v-chip
                      v-if="keyModelsStatus.get(key)?.loading"
                      size="x-small"
                      color="info"
                      variant="tonal"
                    >
                      <v-icon start size="12">mdi-loading</v-icon>
                      {{ t('addChannel.checking') }}
                    </v-chip>
                    <v-chip
                      v-else-if="keyModelsStatus.get(key)?.success"
                      size="x-small"
                      color="success"
                      variant="tonal"
                    >
                      {{ t('addChannel.modelsCount', { statusCode: keyModelsStatus.get(key)?.statusCode ?? 'OK', count: keyModelsStatus.get(key)?.modelCount ?? 0 }) }}
                    </v-chip>
                    <v-tooltip
                      v-else-if="keyModelsStatus.get(key)?.error"
                      :text="keyModelsStatus.get(key)?.error"
                      location="top"
                      max-width="300"
                      content-class="key-tooltip"
                    >
                      <template #activator="{ props: tooltipProps }">
                        <v-chip
                          v-bind="tooltipProps"
                          size="x-small"
                          color="error"
                          variant="tonal"
                        >
                          models {{ keyModelsStatus.get(key)?.statusCode || 'ERR' }}
                        </v-chip>
                      </template>
                    </v-tooltip>
                    <!-- 重复密钥标签 -->
                    <v-chip v-if="duplicateKeyIndex === index" size="x-small" color="error" variant="text">
                      {{ t('channelEditor.auth.duplicateKey') }}
                    </v-chip>
                  </div>
                </div>
              </v-list-item-title>

              <template #append>
                <div class="d-flex align-center ga-1">
                  <!-- 置顶/置底：仅首尾密钥显示 -->
                  <v-tooltip
                    v-if="index === apiKeys.length - 1 && apiKeys.length > 1"
                    :text="t('channelCard.moveTop')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        color="warning"
                        icon
                        variant="text"
                        rounded="md"
                        @click="moveToTop(index)"
                      >
                        <v-icon size="small">mdi-arrow-up-bold</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                  <v-tooltip
                    v-if="index === 0 && apiKeys.length > 1"
                    :text="t('channelCard.moveBottom')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        color="warning"
                        icon
                        variant="text"
                        rounded="md"
                        @click="moveToBottom(index)"
                      >
                        <v-icon size="small">mdi-arrow-down-bold</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                  <v-tooltip
                    :text="copiedKeyIndex === index ? t('channelCard.copied') : t('channelCard.copyKey')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        :color="copiedKeyIndex === index ? 'success' : 'primary'"
                        icon
                        variant="text"
                        @click="copyKey(key, index)"
                      >
                        <v-icon size="small">{{
                          copiedKeyIndex === index ? 'mdi-check' : 'mdi-content-copy'
                        }}</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                  <v-tooltip :text="t('addChannel.deleteKey')" location="top" :open-delay="150" content-class="key-tooltip">
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        color="error"
                        icon
                        variant="text"
                        @click="removeKey(index)"
                      >
                        <v-icon size="small" color="error">mdi-close</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                </div>
              </template>
            </v-list-item>
          </v-list>
        </div>

        <!-- 添加新密钥 -->
        <div class="d-flex align-start ga-3">
          <v-text-field
            v-model="newApiKey"
            :label="t('addChannel.addNewApiKey')"
            :placeholder="t('channelEditor.auth.addNewApiKey.placeholder')"
            prepend-inner-icon="mdi-plus"
            variant="outlined"
            density="comfortable"
            type="password"
            :error="!!apiKeyError"
            :error-messages="apiKeyError"
            class="flex-grow-1"
            @keyup.enter="handleAddKey"
            @input="handleInput"
          />
          <v-btn
            color="primary"
            variant="elevated"
            size="large"
            height="40"
            :disabled="!newApiKey.trim()"
            class="mt-1"
            @click="handleAddKey"
          >
            {{ t('app.actions.add') }}
          </v-btn>
        </div>

        <!-- 被拉黑的密钥（仅编辑模式） -->
        <div v-if="isEditing && visibleDisabledKeys.length" class="mt-4">
          <div class="d-flex align-center ga-2 mb-2">
            <v-icon size="small" color="error">mdi-key-remove</v-icon>
            <span class="text-body-2 font-weight-medium text-error">{{ t('channelCard.disabledKeys') }}</span>
            <v-chip size="x-small" color="error" variant="tonal">{{ visibleDisabledKeys.length }}</v-chip>
          </div>
          <v-list density="compact" class="rounded-lg" style="max-height: 150px; overflow-y: auto;">
            <v-list-item
              v-for="(dk, dkIdx) in visibleDisabledKeys"
              :key="'disabled-' + dkIdx"
              class="px-3"
              style="background: rgba(var(--v-theme-error), 0.04);"
            >
              <template #prepend>
                <v-icon size="small" color="error" class="mr-2">mdi-key-alert</v-icon>
              </template>
              <v-list-item-title class="text-caption font-weight-mono">
                {{ dk.key.length > 20 ? dk.key.slice(0, 8) + '***' + dk.key.slice(-5) : dk.key }}
              </v-list-item-title>
              <v-list-item-subtitle class="d-flex align-center ga-1">
                <v-chip size="x-small" :color="dk.reason === 'insufficient_balance' ? 'warning' : 'error'" variant="tonal">
                  {{ t(getDisabledKeyLabel(dk.reason)) }}
                </v-chip>
                <span class="text-caption">{{ new Date(dk.disabledAt).toLocaleDateString() }}</span>
              </v-list-item-subtitle>
              <template #append>
                <v-btn
                  size="x-small"
                  color="success"
                  variant="tonal"
                  rounded="lg"
                  :loading="restoringKey === dk.key"
                  @click="$emit('restore-key', dk.key)"
                >
                  <v-icon start size="small">mdi-restore</v-icon>
                  {{ t('channelCard.restoreKey') }}
                </v-btn>
              </template>
            </v-list-item>
          </v-list>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from '../../i18n'
import { maskApiKey } from '../../utils/apiKeyMask'

interface KeyModelsStatus {
  loading?: boolean
  success?: boolean
  error?: string
  statusCode?: string | number
  modelCount?: number
}

interface DisabledKey {
  key: string
  reason: string
  disabledAt: string
}

interface Props {
  apiKeys: string[]
  disabledKeys: DisabledKey[]
  keyModelsStatus: Map<string, KeyModelsStatus>
  isEditing: boolean
  restoringKey: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:apiKeys': [string[]]
  'restore-key': [string]
}>()

const { t } = useI18n()

const newApiKey = ref('')
const apiKeyError = ref('')
const duplicateKeyIndex = ref<number | null>(null)
const copiedKeyIndex = ref<number | null>(null)

const hasConfigurableKeys = computed(() => props.apiKeys.length > 0)

const visibleDisabledKeys = computed(() => {
  return props.disabledKeys.filter(dk => !props.apiKeys.includes(dk.key))
})

const handleInput = () => {
  apiKeyError.value = ''
  duplicateKeyIndex.value = null
}

const handleAddKey = () => {
  const trimmed = newApiKey.value.trim()
  if (!trimmed) return

  // 检查重复
  const existingIndex = props.apiKeys.indexOf(trimmed)
  if (existingIndex !== -1) {
    apiKeyError.value = t('addChannel.duplicateKey')
    duplicateKeyIndex.value = existingIndex

    // 3秒后清除高亮
    setTimeout(() => {
      duplicateKeyIndex.value = null
    }, 3000)
    return
  }

  emit('update:apiKeys', [...props.apiKeys, trimmed])
  newApiKey.value = ''
  apiKeyError.value = ''
}

const removeKey = (index: number) => {
  const updated = props.apiKeys.filter((_, i) => i !== index)
  emit('update:apiKeys', updated)
}

const moveToTop = (index: number) => {
  const updated = [...props.apiKeys]
  const [key] = updated.splice(index, 1)
  updated.unshift(key)
  emit('update:apiKeys', updated)
}

const moveToBottom = (index: number) => {
  const updated = [...props.apiKeys]
  const [key] = updated.splice(index, 1)
  updated.push(key)
  emit('update:apiKeys', updated)
}

const copyKey = (key: string, index: number) => {
  navigator.clipboard.writeText(key)
  copiedKeyIndex.value = index
  setTimeout(() => {
    copiedKeyIndex.value = null
  }, 2000)
}

const getDisabledKeyLabel = (reason: string) => {
  const map: Record<string, string> = {
    'insufficient_balance': 'channelCard.blacklistReason.insufficient_balance',
    'unauthorized': 'channelCard.blacklistReason.authentication_error',
    'invalid': 'channelCard.blacklistReason.invalid',
  }
  return (map[reason] || 'channelCard.blacklistReason.unknown') as any
}
</script>

<style scoped>
.section-title {
  font-size: 1.125rem;
  font-weight: 600;
}

.animate-pulse {
  animation: pulse 1s ease-in-out 3;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

.font-weight-mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Courier New', monospace;
}


</style>
