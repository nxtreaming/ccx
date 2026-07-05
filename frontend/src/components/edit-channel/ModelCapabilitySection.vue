<template>
  <div class="model-capability-section">
    <v-card variant="outlined" rounded="lg">
      <v-card-title class="d-flex align-center pa-4 pb-2">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-database</v-icon>
          <span class="section-title">{{ t('addChannel.contextCapabilityTitle') }}</span>
        </div>
      </v-card-title>

      <v-card-text class="pt-2">
        <div class="text-body-2 text-medium-emphasis mb-4">
          {{ t('addChannel.modelCapabilitiesRowsHint') }}
        </div>

        <div v-if="mappedTargetModels.length" class="d-flex align-center flex-wrap ga-2 mb-3">
          <span class="text-caption text-medium-emphasis">{{ t('addChannel.modelCapabilityRedirectTargets') }}</span>
          <v-chip
            v-for="model in mappedTargetModels"
            :key="model"
            size="x-small"
            variant="tonal"
            color="primary"
            class="font-mono"
          >
            {{ model }}
          </v-chip>
        </div>

        <div class="capability-container rounded-xl pa-3 mt-3">
          <div v-if="rows.length" class="d-flex flex-column ga-3">
            <div class="text-caption text-medium-emphasis d-flex align-center justify-space-between px-1">
              <span class="uppercase-label">{{ t('addChannel.modelCapabilitiesConfigured') }}</span>
              <v-chip size="x-small" variant="flat" color="primary" class="font-weight-bold px-2 font-mono">
                {{ rows.length }}
              </v-chip>
            </div>

            <div
              v-for="(row, index) in rows"
              :key="row.id"
              class="capability-card rounded-lg overflow-hidden"
            >
              <div class="capability-card-header d-flex align-center justify-space-between ga-3 px-3 py-2">
                <div class="d-flex align-center ga-2 min-width-0">
                  <div class="model-avatar flex-shrink-0">{{ modelInitial(row) }}</div>
                  <div class="min-width-0">
                    <div class="font-mono text-body-2 font-weight-bold text-truncate">
                      {{ row.model || t('addChannel.modelCapabilityModelPlaceholder') }}
                    </div>
                  </div>
                  <v-chip
                    v-if="row.source === 'builtin' && row.matchedPattern"
                    size="x-small"
                    color="primary"
                    variant="tonal"
                    class="preset-chip"
                  >
                    {{ t('addChannel.modelCapabilityBuiltinMatched', { pattern: row.matchedPattern }) }}
                  </v-chip>
                </div>
                <v-tooltip :text="t('app.actions.delete')" location="top" :open-delay="150">
                  <template #activator="{ props: tip }">
                    <v-btn
                      v-bind="tip"
                      size="small"
                      color="error"
                      icon
                      variant="text"
                      @click="removeRow(index)"
                    >
                      <v-icon size="16">mdi-close</v-icon>
                    </v-btn>
                  </template>
                </v-tooltip>
              </div>

              <div class="capability-card-body pa-3">
                <div class="capability-main">
                  <v-row dense>
                    <v-col cols="6">
                      <v-text-field
                        :model-value="row.contextWindowTokens"
                        :label="t('addChannel.contextTokensShort')"
                        type="number"
                        min="0"
                        variant="outlined"
                        density="compact"
                        hide-details
                        @update:model-value="updateRow(index, { contextWindowTokens: $event })"
                      />
                      <div v-if="formatTokenMeta(row.contextWindowTokens)" class="field-meta">
                        {{ formatTokenMeta(row.contextWindowTokens) }}
                      </div>
                    </v-col>
                    <v-col cols="6">
                      <v-text-field
                        :model-value="row.maxOutputTokens"
                        :label="t('addChannel.outputTokensShort')"
                        type="number"
                        min="0"
                        variant="outlined"
                        density="compact"
                        hide-details
                        @update:model-value="updateRow(index, { maxOutputTokens: $event })"
                      />
                      <div v-if="formatTokenMeta(row.maxOutputTokens)" class="field-meta">
                        {{ formatTokenMeta(row.maxOutputTokens) }}
                      </div>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-combobox
                        :model-value="row.thinkingMode"
                        :items="thinkingModeOptions"
                        :menu-props="modelCapabilityMenuProps"
                        :label="t('addChannel.thinkingModeLabel')"
                        variant="outlined"
                        density="compact"
                        hide-details
                        clearable
                        eager
                        @update:model-value="updateRow(index, { thinkingMode: normalizeSelectableString($event) })"
                        @update:menu="$emit('menu-update', $event)"
                      />
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-combobox
                        :model-value="reasoningEffortsToList(row.reasoningEffortsText)"
                        :items="reasoningEffortOptions"
                        :menu-props="modelCapabilityMenuProps"
                        :label="t('addChannel.reasoningEffortsLabel')"
                        placeholder="high, max"
                        variant="outlined"
                        density="compact"
                        hide-details
                        multiple
                        chips
                        closable-chips
                        clearable
                        eager
                        @update:model-value="updateReasoningEfforts(index, $event)"
                        @update:menu="$emit('menu-update', $event)"
                      />
                    </v-col>
                  </v-row>

                  <div v-if="row.description" class="text-caption text-medium-emphasis mt-2">
                    {{ row.description }}
                  </div>
                  <div
                    v-if="row.defaultOutputTokens || row.recommendedOutputTokens"
                    class="text-caption text-medium-emphasis mt-1"
                  >
                    <span v-if="row.defaultOutputTokens">
                      {{ t('addChannel.defaultOutputTokensMeta', { tokens: formatTokens(row.defaultOutputTokens) }) }}
                    </span>
                    <span v-if="row.defaultOutputTokens && row.recommendedOutputTokens"> · </span>
                    <span v-if="row.recommendedOutputTokens">
                      {{ t('addChannel.recommendedOutputTokensMeta', { tokens: formatTokens(row.recommendedOutputTokens) }) }}
                    </span>
                  </div>
                </div>

                <div class="pricing-panel rounded-lg pa-3">
                  <div class="d-flex align-center justify-space-between ga-2 mb-2">
                    <div class="d-flex align-center ga-1 text-caption font-weight-bold">
                      <v-icon size="14" color="primary">mdi-cash-multiple</v-icon>
                      <span>{{ t('addChannel.pricingTitle') }}</span>
                    </div>
                    <span class="text-caption text-medium-emphasis">{{ pricingUnitLabel() }}</span>
                  </div>
                  <v-row dense>
                    <v-col cols="6" md="2">
                      <v-text-field
                        :model-value="row.pricingCurrency"
                        :label="t('addChannel.pricingCurrencyLabel')"
                        variant="outlined"
                        density="compact"
                        hide-details
                        @update:model-value="updateRow(index, { pricingCurrency: String($event || '') })"
                      />
                    </v-col>
                    <v-col cols="12" sm="4" md="3">
                      <v-text-field
                        :model-value="row.inputCacheHitPrice"
                        :label="t('addChannel.inputCacheHitPriceLabel')"
                        type="number"
                        min="0"
                        step="0.000001"
                        variant="outlined"
                        density="compact"
                        hide-details
                        @update:model-value="updateRow(index, { inputCacheHitPrice: $event })"
                      />
                    </v-col>
                    <v-col cols="12" sm="4" md="3">
                      <v-text-field
                        :model-value="row.inputCacheMissPrice"
                        :label="t('addChannel.inputCacheMissPriceLabel')"
                        type="number"
                        min="0"
                        step="0.000001"
                        variant="outlined"
                        density="compact"
                        hide-details
                        @update:model-value="updateRow(index, { inputCacheMissPrice: $event })"
                      />
                    </v-col>
                    <v-col cols="12" sm="4" md="3">
                      <v-text-field
                        :model-value="row.outputPrice"
                        :label="t('addChannel.outputPriceLabel')"
                        type="number"
                        min="0"
                        step="0.000001"
                        variant="outlined"
                        density="compact"
                        hide-details
                        @update:model-value="updateRow(index, { outputPrice: $event })"
                      />
                    </v-col>
                  </v-row>
                </div>
              </div>
            </div>
          </div>

          <div class="add-capability-row d-flex align-center ga-3 pa-3 mt-3 rounded-lg">
            <v-combobox
              :model-value="newModel"
              :items="targetModelOptions"
              :menu-props="modelCapabilityMenuProps"
              item-title="title"
              item-value="value"
              :no-data-text="modelListNoDataText"
              :loading="fetchingModels"
              :label="t('addChannel.modelCapabilityModelLabel')"
              :placeholder="t('addChannel.modelCapabilityModelPlaceholder')"
              variant="outlined"
              density="compact"
              hide-details
              clearable
              eager
              class="flex-grow-1 font-mono"
              @focus="$emit('sync-upstream')"
              @update:model-value="handleNewModelUpdate"
              @update:menu="$emit('menu-update', $event)"
              @keyup.enter="addRow"
            />
            <v-btn
              color="primary"
              height="40"
              variant="flat"
              class="rounded-lg px-4"
              :disabled="!newModelName"
              @click="addRow"
            >
              <v-icon size="18" class="mr-1">mdi-plus</v-icon>
              {{ t('app.actions.add') }}
            </v-btn>
          </div>
        </div>

        <div v-if="error" class="text-error text-caption mt-2">
          {{ error }}
        </div>
        <div v-if="fetchModelsError" class="text-error text-caption mt-2">
          {{ fetchModelsError }}
        </div>

      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from '../../i18n'
import {
  createModelCapabilityRow,
  normalizeSelectableString,
  resolveBuiltinUpstreamModelCapability,
  type ModelCapabilityRow,
} from '../../utils/channelPayload'

type SelectableString = string | { title?: string; value?: unknown } | null | undefined
type ModelOptionValue = string | { title: string; value: string } | null

const props = defineProps<{
  rows: ModelCapabilityRow[]
  targetModelOptions: Array<{ title: string; value: string }>
  mappedTargetModels: string[]
  fetchingModels: boolean
  fetchModelsError: string
  error: string
}>()

const emit = defineEmits<{
  'update:rows': [ModelCapabilityRow[]]
  'sync-upstream': []
  'menu-update': [open: boolean]
}>()

const { t } = useI18n()
const newModel = ref<ModelOptionValue>('')
const thinkingModeOptions = ['thinking', 'extended', 'adaptive', 'adaptive_only', 'adaptive_always_on']
const reasoningEffortOptions = ['none', 'low', 'medium', 'high', 'xhigh', 'max']
const modelCapabilityMenuProps = { attach: 'body', zIndex: 2600 } as const
const newModelName = computed(() => normalizeSelectableString(newModel.value).trim())
const modelListNoDataText = computed(() => props.fetchModelsError || t('addChannel.modelListEmptyHint'))

function formatTokens(value?: number) {
  if (!value) return ''
  if (value >= 1000 && value % 1000 === 0) return `${value / 1000}k`
  return value.toLocaleString()
}

function formatTokenMeta(value: ModelCapabilityRow['contextWindowTokens']) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue) || numericValue <= 0) return ''
  const formatted = numericValue.toLocaleString()
  if (numericValue >= 1000000 && numericValue % 1000000 === 0) {
    return `${formatted} (${numericValue / 1000000}M)`
  }
  if (numericValue >= 1000 && numericValue % 1000 === 0) {
    return `${formatted} (${numericValue / 1000}K)`
  }
  return formatted
}

function pricingUnitLabel() {
  return t('addChannel.modelPricingUnitHint')
}

function modelInitial(row: ModelCapabilityRow) {
  const source = (row.model || '?').trim()
  return source ? source.slice(0, 1).toUpperCase() : '?'
}

const nextRowId = () => Date.now() + Math.floor(Math.random() * 1000)

function updateRows(rows: ModelCapabilityRow[]) {
  emit('update:rows', rows)
}

function updateRow(index: number, patch: Partial<ModelCapabilityRow>) {
  const rows = props.rows.map((row, rowIndex) => (
    rowIndex === index ? { ...row, ...patch, source: patch.source || row.source } : row
  ))
  updateRows(rows)
}

function reasoningEffortsToList(value: string) {
  return value
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
}

function updateReasoningEfforts(index: number, value: SelectableString[] | SelectableString) {
  const values = Array.isArray(value) ? value : [value]
  const seen = new Set<string>()
  const reasoningEffortsText = values
    .map(item => normalizeSelectableString(item).trim())
    .filter(item => {
      if (!item || seen.has(item)) return false
      seen.add(item)
      return true
    })
    .join(', ')
  updateRow(index, { reasoningEffortsText })
}

function addRow() {
  const model = newModelName.value
  if (!model) return
  if (props.rows.some(row => normalizeSelectableString(row.model).trim() === model)) {
    newModel.value = ''
    return
  }
  const builtin = resolveBuiltinUpstreamModelCapability(model)
  const row = createModelCapabilityRow(
    nextRowId(),
    model,
    builtin?.capability,
    builtin ? 'builtin' : 'custom',
    builtin?.pattern || '',
  )
  updateRows([...props.rows, row])
  newModel.value = ''
}

function handleNewModelUpdate(value: ModelOptionValue) {
  newModel.value = value
  const model = normalizeSelectableString(value).trim()
  const selectedKnownModel = props.targetModelOptions.some(option => (
    option.value.trim().toLowerCase() === model.toLowerCase()
  ))
  if (selectedKnownModel) {
    addRow()
  }
}

function removeRow(index: number) {
  updateRows(props.rows.filter((_, rowIndex) => rowIndex !== index))
}
</script>

<style scoped>
.section-title {
  font-size: 1.125rem;
  font-weight: 600;
}

.font-mono {
  font-family: 'SF Mono', 'Fira Code', Monaco, Consolas, monospace !important;
}

.capability-container {
  background: rgba(var(--v-border-color), 0.03);
  border: 1px solid rgba(var(--v-border-color), 0.08);
}

.capability-card {
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgba(var(--v-border-color), 0.12);
}

.capability-card-header {
  background: rgba(var(--v-border-color), 0.035);
  border-bottom: 1px solid rgba(var(--v-border-color), 0.08);
}

.capability-card-body {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}

.pricing-panel {
  background: rgba(var(--v-theme-primary), 0.035);
  border: 1px solid rgba(var(--v-theme-primary), 0.1);
}

.model-avatar {
  align-items: center;
  background: rgba(var(--v-theme-primary), 0.12);
  border: 1px solid rgba(var(--v-theme-primary), 0.2);
  border-radius: 999px;
  color: rgb(var(--v-theme-primary));
  display: inline-flex;
  font-size: 0.75rem;
  font-weight: 700;
  height: 28px;
  justify-content: center;
  width: 28px;
}

.preset-chip {
  max-width: 240px;
}

.field-meta {
  color: rgba(var(--v-theme-on-surface), 0.55);
  font-size: 0.72rem;
  line-height: 1rem;
  margin-top: 4px;
}

.min-width-0 {
  min-width: 0;
}

.add-capability-row {
  background: rgba(var(--v-theme-surface), 0.8);
  border: 1px solid rgba(var(--v-border-color), 0.15);
}

.uppercase-label {
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 600;
}

</style>
