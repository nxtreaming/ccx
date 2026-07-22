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

        <div v-if="route.models.length" class="protocol-model-route__models">
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
        <div v-else class="text-caption text-medium-emphasis">
          {{ t('channelEditor.protocolModels.empty') }}
        </div>
        <div v-if="route.hasBindingDifferences" class="protocol-model-route__bindings">
          <div class="protocol-model-route__bindings-header">
            <v-icon color="warning" size="16">mdi-key-alert</v-icon>
            <div>
              <div class="text-caption font-weight-medium text-warning">
                {{ t('channelEditor.protocolModels.keyDifferences') }}
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ t('channelEditor.protocolModels.keyDifferencesHint') }}
              </div>
            </div>
          </div>
          <div class="protocol-model-route__binding-list">
            <details
              v-for="binding in route.bindings"
              :key="binding.credentialUid || binding.keyMask"
              class="protocol-model-binding"
            >
              <summary class="protocol-model-binding__summary">
                <code>{{ binding.keyMask }}</code>
                <v-chip class="protocol-model-binding__coverage" size="x-small" variant="tonal" color="warning">
                  {{ t('channelEditor.protocolModels.coverage', { available: binding.models.length, total: route.models.length }) }}
                </v-chip>
                <v-icon class="protocol-model-binding__chevron" size="16">mdi-chevron-down</v-icon>
              </summary>
              <div class="protocol-model-binding__detail">
                <div class="protocol-model-binding__discovery-meta text-caption text-medium-emphasis">
                  <span>
                    {{ t('channelEditor.protocolModels.lastDiscovered') }}
                    {{ formatDiscoveryTime(binding.modelsDiscoveredAt) || t('channelEditor.protocolModels.discoveryTimeUnknown') }}
                  </span>
                  <span v-if="binding.modelDiscoverySource">
                    {{ discoverySourceLabel(binding.modelDiscoverySource) }}
                  </span>
                  <span v-if="binding.modelDiscoveryMessage">{{ binding.modelDiscoveryMessage }}</span>
                </div>
                <div>
                  <div class="protocol-model-binding__label text-success">
                    {{ t('channelEditor.protocolModels.availableModels') }} · {{ binding.models.length }}
                  </div>
                  <div v-if="binding.models.length" class="protocol-model-binding__models">
                    <v-chip
                      v-for="model in binding.models"
                      :key="`available:${model}`"
                      size="x-small"
                      variant="outlined"
                      color="success"
                      class="protocol-model-binding__model"
                    >
                      {{ model }}
                    </v-chip>
                  </div>
                  <div v-else class="text-caption text-medium-emphasis">
                    {{ t('channelEditor.protocolModels.empty') }}
                  </div>
                </div>
                <div v-if="binding.missingModels.length">
                  <div class="protocol-model-binding__label text-warning">
                    {{ t('channelEditor.protocolModels.unavailableModels') }} · {{ binding.missingModels.length }}
                  </div>
                  <div class="protocol-model-binding__models">
                    <v-chip
                      v-for="model in binding.missingModels"
                      :key="`unavailable:${model}`"
                      size="x-small"
                      variant="tonal"
                      color="warning"
                      class="protocol-model-binding__model"
                    >
                      {{ model }}
                    </v-chip>
                  </div>
                </div>
              </div>
            </details>
          </div>
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
  const bindingsWithDifferences = bindings.map(binding => {
    const availableModels = new Set(binding.models)
    return {
      ...binding,
      missingModels: models.filter(model => !availableModels.has(model)),
    }
  })

  return {
    ...route,
    upstreamKind,
    label: t(definition.labelKey),
    path: definition.path,
    icon: definition.icon,
    models,
    bindings: bindingsWithDifferences,
    hasInventory: hasDiscoveredInventory || models.length > 0,
    hasBindingDifferences: bindings.length > 1 && bindingSignatures.size > 1,
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
  display: grid;
  grid-template-columns: minmax(220px, 0.8fr) minmax(0, 2fr);
  gap: 16px;
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
  grid-column: 1 / -1;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: -6px;
}

.protocol-model-route__path {
  overflow-wrap: anywhere;
  color: rgba(var(--v-theme-on-surface), 0.62);
  font-size: 0.72rem;
  line-height: 1.35;
}

.protocol-model-route__models {
  display: flex;
  align-items: flex-start;
  align-content: flex-start;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.protocol-model-route__bindings {
  display: flex;
  grid-column: 1 / -1;
  flex-direction: column;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px dashed rgba(var(--v-theme-warning), 0.35);
}

.protocol-model-route__bindings-header {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.protocol-model-route__binding-list {
  overflow: hidden;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 4px;
}

.protocol-model-binding + .protocol-model-binding {
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.1);
}

.protocol-model-binding__summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto 20px;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 7px 10px;
  cursor: pointer;
  list-style: none;
  background: rgba(var(--v-theme-warning), 0.04);
}

.protocol-model-binding__summary::-webkit-details-marker {
  display: none;
}

.protocol-model-binding__summary code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.protocol-model-binding__chevron {
  transition: transform 0.16s ease;
}

.protocol-model-binding[open] .protocol-model-binding__chevron {
  transform: rotate(180deg);
}

.protocol-model-binding__detail {
  display: grid;
  gap: 12px;
  padding: 12px;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.08);
}

.protocol-model-binding__discovery-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
}

.protocol-model-binding__label {
  margin-bottom: 6px;
  font-size: 0.72rem;
  font-weight: 600;
}

.protocol-model-binding__models {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 0;
}

.protocol-model-binding__model {
  height: auto;
  min-height: 20px;
  max-width: 100%;
}

.protocol-model-binding__model :deep(.v-chip__content) {
  overflow-wrap: anywhere;
  white-space: normal;
  line-height: 1.3;
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

@media (max-width: 700px) {
  .protocol-model-route {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .protocol-model-binding__summary {
    grid-template-columns: minmax(0, 1fr) auto 18px;
    gap: 6px;
  }
}

@media (max-width: 480px) {
  .protocol-model-binding__summary {
    grid-template-columns: minmax(0, 1fr) 18px;
  }

  .protocol-model-binding__coverage {
    grid-column: 1;
    justify-self: start;
  }

  .protocol-model-binding__chevron {
    grid-row: 1 / span 2;
    grid-column: 2;
    align-self: center;
  }
}
</style>
