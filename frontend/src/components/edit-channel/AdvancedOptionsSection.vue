<template>
  <div class="advanced-options-section">
    <v-row dense>
      <!-- 描述 -->
      <v-col cols="12">
        <v-textarea
          :model-value="form.description"
          :label="t('addChannel.descriptionLabel')"
          :hint="t('addChannel.descriptionHint')"
          persistent-hint
          prepend-inner-icon="mdi-text"
          variant="outlined"
          density="comfortable"
          rows="3"
          no-resize
          @update:model-value="updateField('description', $event)"
        />
      </v-col>

      <!-- 跳过 TLS 证书验证 -->
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="warning">mdi-shield-alert</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.skipTlsLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.skipTlsHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.insecureSkipVerify" inset color="warning" hide-details @update:model-value="updateField('insecureSkipVerify', $event)" />
        </div>
      </v-col>

      <!-- 低质量渠道标记 -->
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between">
          <div class="d-flex align-center ga-2">
            <v-icon color="info">mdi-speedometer-slow</v-icon>
            <div>
              <div class="section-title section-title--soft">{{ t('addChannel.lowQualityLabel') }}</div>
              <div class="text-caption text-medium-emphasis">{{ t('addChannel.lowQualityHint') }}</div>
            </div>
          </div>
          <v-switch :model-value="form.lowQuality" inset color="info" hide-details @update:model-value="updateField('lowQuality', $event)" />
        </div>
      </v-col>

      <!-- Runtime 运行期策略 -->
      <v-col cols="12">
        <RuntimeSwitchGroup :form="form" @update:field="updateField" />
      </v-col>

      <!-- Compatibility 协议规范化 -->
      <v-col cols="12">
        <CompatibilitySwitchGroup
          :form="form"
          :channel-type="channelType"
          :supports-chat-role-normalization="supportsChatRoleNormalization"
          :supports-open-a-i-advanced-options="supportsOpenAIAdvancedOptions"
          :reasoning-param-style-options="reasoningParamStyleOptions"
          @update:field="updateField"
          @menu-update="$emit('menu-update', $event)"
        />
      </v-col>

      <slot name="custom-headers" />

      <!-- Transport 代理路由网络 -->
      <v-col cols="12">
        <TransportConfigGroup :form="form" :rules="rules" @update:field="updateField">
          <template #stream-timeout>
            <slot name="stream-timeout" />
          </template>
        </TransportConfigGroup>
      </v-col>

      <!-- 主动限速 -->
      <v-col cols="12">
        <RateLimitGroup :form="form" @update:field="updateField" />
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from '../../i18n'
import CompatibilitySwitchGroup from './CompatibilitySwitchGroup.vue'
import RateLimitGroup from './RateLimitGroup.vue'
import RuntimeSwitchGroup from './RuntimeSwitchGroup.vue'
import TransportConfigGroup from './TransportConfigGroup.vue'

interface FormData {
  description: string
  insecureSkipVerify: boolean
  lowQuality: boolean
  autoBlacklistBalance: boolean
  codexNativeToolPassthrough?: boolean
  codexToolCompat?: boolean
  stripImageGenerationTool?: boolean
  convertImageUrlToB64Json?: boolean
  normalizeMetadataUserId?: boolean
  stripBillingHeader?: boolean
  normalizeNonstandardChatRoles?: boolean
  reasoningParamStyle?: string
  fastMode?: boolean
  injectDummyThoughtSignature?: boolean
  stripThoughtSignature?: boolean
  passbackReasoningContent?: boolean
  passbackThinkingBlocks?: boolean
  stripEmptyTextBlocks?: boolean
  normalizeSystemRoleToTopLevel?: boolean
  historicalImageTurnLimit?: number
  proxyUrl: string
  requestTimeoutMs: string | number | null
  rateLimitRpm: string | number | null
  rateLimitWindowMinutes: string | number | null
  rateLimitMaxConcurrent: string | number | null
  rateLimitAutoFromHeaders: boolean
  routePrefix?: string
  serviceType: string
}

interface Props {
  form: FormData
  channelType: string
  supportsChatRoleNormalization: boolean
  supportsOpenAIAdvancedOptions: boolean
  reasoningParamStyleOptions: Array<{ title: string; value: string }>
  rules: Record<string, (v: any) => boolean | string>
}

defineProps<Props>()

const emit = defineEmits<{
  'update:form': [Partial<FormData>]
  'menu-update': [boolean]
}>()

const { t } = useI18n()

const updateField = (field: keyof FormData, value: any) => {
  emit('update:form', { [field]: value })
}
</script>

<style scoped>
.section-title--soft {
  font-weight: 500;
  font-size: 0.875rem;
}

.channel-config-select {
  max-width: 200px;
}

.rate-limit-card {
  background: rgba(var(--v-theme-surface-variant), 0.3);
}
</style>
