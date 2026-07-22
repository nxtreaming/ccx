<template>
  <div v-if="normalizedRoutes.length" class="protocol-model-availability">
    <div class="protocol-model-availability__header">
      <v-icon color="primary" size="20">mdi-routes</v-icon>
      <div>
        <div class="text-subtitle-2 font-weight-medium">
          {{ t('channelEditor.protocolModels.title') }}
        </div>
        <div class="text-caption text-medium-emphasis">
          {{ t('channelEditor.protocolModels.hint') }}
        </div>
      </div>
      <v-progress-circular v-if="loading" class="ml-auto" color="primary" indeterminate size="18" width="2" />
    </div>

    <div class="protocol-model-availability__rows">
      <div
        v-for="route in normalizedRoutes"
        :key="`${route.kind}:${route.channelUid || route.index}`"
        class="protocol-model-route"
        :data-kind="route.upstreamKind"
      >
        <div class="protocol-model-route__identity">
          <v-icon size="18" color="primary">{{ route.icon }}</v-icon>
          <div class="protocol-model-route__label">
            <span class="text-body-2 font-weight-medium">{{ route.label }}</span>
            <code class="protocol-model-route__path">{{ route.path }}</code>
          </div>
          <v-chip v-if="route.hasInventory" size="x-small" variant="tonal" color="primary">
            {{ t('channelEditor.protocolModels.count', { count: route.models.length }) }}
          </v-chip>
        </div>

        <div v-if="route.hasInventory" class="protocol-model-route__discovery-meta">
          <span class="text-caption text-medium-emphasis">
            {{ t('channelEditor.protocolModels.lastDiscovered') }}
            {{ route.discoveryTime || t('channelEditor.protocolModels.discoveryTimeUnknown') }}
          </span>
          <v-chip size="x-small" variant="tonal" color="secondary">
            {{ route.discoverySourceLabel }}
          </v-chip>
          <span v-if="route.modelDiscoveryMessage" class="text-caption text-medium-emphasis">
            {{ route.modelDiscoveryMessage }}
          </span>
        </div>

        <!-- 多 Key 场景只显示差异：列出未被全部 Key 覆盖的模型及缺失的 Key -->
        <div v-if="route.diffModels.length" class="protocol-model-route__diffs">
          <div class="protocol-model-route__diffs-header">
            <v-icon color="warning" size="16">mdi-key-alert</v-icon>
            <span class="text-caption font-weight-medium text-warning">
              {{ t('channelEditor.protocolModels.diffCount', { count: route.diffModels.length }) }}
            </span>
          </div>
          <div class="protocol-model-route__diff-list">
            <div
              v-for="diff in route.diffModels"
              :key="diff.model"
              class="protocol-model-diff"
            >
              <code class="protocol-model-diff__model">{{ diff.model }}</code>
              <span class="text-caption text-medium-emphasis text-no-wrap">
                {{ t('channelEditor.protocolModels.missingFor') }}
              </span>
              <code
                v-for="key in diff.missingKeys"
                :key="key.credentialUid || key.keyMask"
                class="protocol-model-diff__key"
              >
                {{ key.keyMask }}
              </code>
            </div>
          </div>
          <div class="protocol-model-route__coverage">
            <v-chip
              v-for="binding in route.bindings"
              :key="binding.credentialUid || binding.keyMask"
              size="x-small"
              variant="tonal"
              :color="binding.models.length === route.models.length ? 'success' : 'warning'"
            >
              {{ binding.keyMask }} ·
              {{ t('channelEditor.protocolModels.coverage', { available: binding.models.length, total: route.models.length }) }}
            </v-chip>
          </div>
        </div>
        <div v-else-if="route.bindings.length > 1" class="protocol-model-route__consistent">
          <v-icon color="success" size="14">mdi-check-all</v-icon>
          <span class="text-caption text-medium-emphasis">
            {{ t('channelEditor.protocolModels.consistent', { count: route.bindings.length }) }}
          </span>
        </div>

        <details v-if="route.models.length" class="protocol-model-route__all">
          <summary class="protocol-model-route__all-summary">
            <span class="text-caption">
              {{ t('channelEditor.protocolModels.viewAll', { count: route.models.length }) }}
            </span>
            <v-icon class="protocol-model-route__all-chevron" size="16">mdi-chevron-down</v-icon>
          </summary>
          <div class="protocol-model-route__models">
            <v-chip
              v-for="model in route.models"
              :key="model"
              size="small"
              variant="outlined"
              class="protocol-model-route__model"
            >
              {{ model }}
            </v-chip>
          </div>
        </details>
        <div v-else class="text-caption text-medium-emphasis">
          {{ t('channelEditor.protocolModels.empty') }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import { useI18n } from '../../i18n'
import type { ChannelKind, ChannelProtocolRoute } from '../../services/api'

interface ProtocolDefinition {
  labelKey: string
  path: string
  icon: string
}

const protocolDefinitions: Record<ChannelKind, ProtocolDefinition> = {
  messages: {
    labelKey: 'channelEditor.protocolModels.messages',
    path: '/v1/messages',
    icon: 'mdi-message-text-outline',
  },
  chat: {
    labelKey: 'channelEditor.protocolModels.chat',
    path: '/v1/chat/completions',
    icon: 'mdi-forum-outline',
  },
  responses: {
    labelKey: 'channelEditor.protocolModels.responses',
    path: '/v1/responses',
    icon: 'mdi-code-json',
  },
  gemini: {
    labelKey: 'channelEditor.protocolModels.gemini',
    path: '/v1beta/models/{model}:generateContent',
    icon: 'mdi-creation-outline',
  },
  images: {
    labelKey: 'channelEditor.protocolModels.images',
    path: '/v1/images/*',
    icon: 'mdi-image-outline',
  },
  vectors: {
    labelKey: 'channelEditor.protocolModels.vectors',
    path: '/v1/embeddings',
    icon: 'mdi-vector-polyline',
  },
}

const props = withDefaults(defineProps<{
  routes?: ChannelProtocolRoute[]
  loading?: boolean
}>(), {
  loading: false,
})

const { t } = useI18n()

const discoverySourceKey: Record<string, string> = {
  control_plane: 'channelEditor.protocolModels.source.controlPlane',
  models_api: 'channelEditor.protocolModels.source.modelsApi',
  builtin_manifest: 'channelEditor.protocolModels.source.builtinManifest',
  builtin_fallback: 'channelEditor.protocolModels.source.builtinFallback',
  mixed: 'channelEditor.protocolModels.source.mixed',
}

const discoverySourceLabel = (source?: string) => {
  const key = source ? discoverySourceKey[source] : undefined
  return t(key ?? 'channelEditor.protocolModels.source.unknown')
}

const discoveryDateTimeFormat = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'medium',
})

const formatDiscoveryTime = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : discoveryDateTimeFormat.format(date)
}

const normalizeModels = (models?: string[]) => Array.from(new Set(
  (models ?? []).map(model => model.trim()).filter(Boolean),
)).sort((left, right) => left.localeCompare(right))

const normalizedRoutes = computed(() => (props.routes ?? []).map((route) => {
  const upstreamKind = route.upstreamKind ?? route.kind
  const definition = protocolDefinitions[upstreamKind]
  const hasDiscoveredInventory = route.modelInventoryKnown === true || Array.isArray(route.discoveredModels)
  const inventoryModels = hasDiscoveredInventory
    ? normalizeModels(route.discoveredModels)
    : normalizeModels(route.supportedModels)
  const bindings = (route.modelBindings ?? []).map(binding => ({
    ...binding,
    models: normalizeModels(binding.models),
  }))
  const models = normalizeModels([
    ...inventoryModels,
    ...bindings.flatMap(binding => binding.models),
  ])
  const bindingSignatures = new Set(bindings.map(binding => binding.models.join('\u0000')))
  const hasBindingDifferences = bindings.length > 1 && bindingSignatures.size > 1
  // 只保留未被全部 Key 覆盖的模型，并列出缺失该模型的 Key。
  const diffModels = hasBindingDifferences
    ? models
        .map(model => ({
          model,
          missingKeys: bindings.filter(binding => !binding.models.includes(model)),
        }))
        .filter(diff => diff.missingKeys.length > 0)
    : []

  return {
    ...route,
    upstreamKind,
    label: t(definition.labelKey),
    path: definition.path,
    icon: definition.icon,
    models,
    bindings,
    diffModels,
    hasInventory: hasDiscoveredInventory || models.length > 0,
    discoveryTime: formatDiscoveryTime(route.modelsDiscoveredAt),
    discoverySourceLabel: discoverySourceLabel(route.modelDiscoverySource),
  }
}))
</script>

<style scoped>
.protocol-model-availability {
  margin-top: 8px;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.12);
}

.protocol-model-availability__header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 18px 0 12px;
}

.protocol-model-availability__rows {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 6px;
  overflow: hidden;
}

.protocol-model-route {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
}

.protocol-model-route + .protocol-model-route {
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.1);
}

.protocol-model-route__identity {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}

.protocol-model-route__label {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.protocol-model-route__discovery-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.protocol-model-route__path {
  overflow-wrap: anywhere;
  color: rgba(var(--v-theme-on-surface), 0.62);
  font-size: 0.72rem;
  line-height: 1.35;
}

.protocol-model-route__diffs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border: 1px dashed rgba(var(--v-theme-warning), 0.4);
  border-radius: 6px;
  background: rgba(var(--v-theme-warning), 0.04);
}

.protocol-model-route__diffs-header {
  display: flex;
  align-items: center;
  gap: 6px;
}

.protocol-model-route__diff-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.protocol-model-diff {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 4px 8px;
}

.protocol-model-diff__model {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.protocol-model-diff__key {
  padding: 0 6px;
  border-radius: 4px;
  background: rgba(var(--v-theme-warning), 0.12);
  color: rgb(var(--v-theme-warning));
  font-size: 0.72rem;
}

.protocol-model-route__coverage {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding-top: 6px;
  border-top: 1px dashed rgba(var(--v-theme-warning), 0.25);
}

.protocol-model-route__consistent {
  display: flex;
  align-items: center;
  gap: 6px;
}

.protocol-model-route__all-summary {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  width: fit-content;
  cursor: pointer;
  list-style: none;
  color: rgba(var(--v-theme-on-surface), 0.62);
}

.protocol-model-route__all-summary::-webkit-details-marker {
  display: none;
}

.protocol-model-route__all-chevron {
  transition: transform 0.16s ease;
}

.protocol-model-route__all[open] .protocol-model-route__all-chevron {
  transform: rotate(180deg);
}

.protocol-model-route__models {
  display: flex;
  align-items: flex-start;
  align-content: flex-start;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
  padding-top: 8px;
}

.protocol-model-route__model {
  height: auto;
  min-height: 24px;
  max-width: 100%;
}

.protocol-model-route__model :deep(.v-chip__content) {
  overflow-wrap: anywhere;
  white-space: normal;
  line-height: 1.35;
}
</style>
