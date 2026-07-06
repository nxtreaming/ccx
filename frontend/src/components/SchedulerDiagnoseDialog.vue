<template>
  <v-dialog :model-value="modelValue" max-width="820" @update:model-value="$emit('update:modelValue', $event)">
    <v-card>
      <v-card-title class="d-flex align-center justify-space-between">
        <span class="dialog-title">{{ t('schedulerDiagnose.title') }}</span>
        <v-tooltip :text="t('app.actions.close') + ' (Esc)'" location="bottom" content-class="ccx-tooltip">
          <template #activator="{ props: tooltipProps }">
            <v-btn icon size="small" variant="text" v-bind="tooltipProps" @click="$emit('update:modelValue', false)">
              <v-icon>mdi-close</v-icon>
            </v-btn>
          </template>
        </v-tooltip>
      </v-card-title>
      <v-divider />

      <v-card-text>
        <v-row dense>
          <v-col cols="12" sm="6">
            <v-text-field v-model="model" :label="t('schedulerDiagnose.model')" density="compact" variant="outlined" hide-details />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="userId" :label="t('schedulerDiagnose.userId')" density="compact" variant="outlined" hide-details />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="routePrefix" :label="t('schedulerDiagnose.routePrefix')" density="compact" variant="outlined" hide-details />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="channelName" :label="t('schedulerDiagnose.channelName')" density="compact" variant="outlined" hide-details />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="failedChannelsText" :label="t('schedulerDiagnose.failedChannels')" density="compact" variant="outlined" hide-details />
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              v-model="agentRole"
              :items="agentRoleItems"
              :label="t('schedulerDiagnose.agentRole')"
              density="compact"
              variant="outlined"
              hide-details
            />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field v-model="inputTokens" :label="t('schedulerDiagnose.inputTokens')" density="compact" variant="outlined" type="number" hide-details />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field v-model="outputTokens" :label="t('schedulerDiagnose.outputTokens')" density="compact" variant="outlined" type="number" hide-details />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field v-model="requiredTokens" :label="t('schedulerDiagnose.requiredTokens')" density="compact" variant="outlined" type="number" hide-details />
          </v-col>
          <v-col cols="12" class="d-flex align-center ga-4 flex-wrap">
            <v-checkbox v-model="hasImageContent" :label="t('schedulerDiagnose.hasImageContent')" density="compact" hide-details />
            <v-checkbox v-model="explicitOutputMax" :label="t('schedulerDiagnose.explicitOutputMax')" density="compact" hide-details />
            <v-checkbox v-model="skipWindowValidation" :label="t('schedulerDiagnose.skipWindowValidation')" density="compact" hide-details />
          </v-col>
        </v-row>

        <div class="d-flex align-center ga-2 mt-4">
          <v-btn color="primary" :loading="isRunning" @click="runDiagnose">
            <v-icon start>mdi-routes</v-icon>
            {{ t('schedulerDiagnose.run') }}
          </v-btn>
          <v-btn variant="text" :disabled="isRunning" @click="clearResult">
            {{ t('schedulerDiagnose.clear') }}
          </v-btn>
        </div>

        <v-alert v-if="result?.ok === false" type="error" variant="tonal" density="compact" class="mt-4">
          {{ result.error }}
        </v-alert>

        <div v-if="result && hasTraceDetails" class="mt-4 scheduler-diagnose-result">
          <div v-if="result.ok" class="d-flex align-center ga-2 flex-wrap mb-2">
            <v-chip size="small" color="success" variant="tonal">
              {{ t('schedulerDiagnose.selected') }} {{ result.selected?.channelIndex }}:{{ result.selected?.channelName }}
            </v-chip>
            <v-chip v-if="result.reason" size="small" color="secondary" variant="tonal">
              {{ result.reason }}
            </v-chip>
          </div>

          <div v-if="result.summary" class="mb-3">
            <div class="scheduler-diagnose-label">{{ t('schedulerDiagnose.summary') }}</div>
            <code class="scheduler-diagnose-code">{{ result.summary }}</code>
          </div>

          <div v-if="result.trace?.stages?.length" class="mb-3">
            <div class="scheduler-diagnose-label">{{ t('schedulerDiagnose.stages') }}</div>
            <div class="d-flex ga-2 flex-wrap">
              <v-chip v-for="stage in result.trace.stages" :key="stage.name" size="small" variant="outlined">
                {{ stage.name }}: {{ stage.count }}
              </v-chip>
            </div>
          </div>

          <div v-if="result.trace?.candidates?.length">
            <div class="scheduler-diagnose-label">{{ t('schedulerDiagnose.candidates') }}</div>
            <v-table density="compact" class="scheduler-diagnose-table">
              <thead>
                <tr>
                  <th>{{ t('schedulerDiagnose.channel') }}</th>
                  <th>{{ t('schedulerDiagnose.stage') }}</th>
                  <th>{{ t('schedulerDiagnose.reason') }}</th>
                  <th>{{ t('schedulerDiagnose.details') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="candidate in result.trace.candidates" :key="`${candidate.stage}:${candidate.channelIndex}:${candidate.reason}`">
                  <td>{{ candidate.channelIndex }}:{{ candidate.channelName }}</td>
                  <td>{{ candidate.stage }}</td>
                  <td>{{ candidate.reason }}</td>
                  <td>{{ candidate.details }}</td>
                </tr>
              </tbody>
            </v-table>
          </div>
        </div>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api, type ChannelKind, type SchedulerDiagnoseRequest, type SchedulerDiagnoseResponse } from '../services/api'
import { useI18n } from '../i18n'

const props = defineProps<{
  modelValue: boolean
  channelType: ChannelKind
}>()

const emit = defineEmits<{
  (_e: 'update:modelValue', _v: boolean): void
}>()

const { t } = useI18n()

const model = ref('')
const userId = ref('')
const routePrefix = ref('')
const channelName = ref('')
const failedChannelsText = ref('')
const agentRole = ref('')
const hasImageContent = ref(false)
const explicitOutputMax = ref(false)
const skipWindowValidation = ref(false)
const inputTokens = ref('')
const outputTokens = ref('')
const requiredTokens = ref('')
const isRunning = ref(false)
const result = ref<SchedulerDiagnoseResponse | null>(null)

const hasTraceDetails = computed(() => Boolean(
  result.value?.summary ||
  result.value?.trace?.stages?.length ||
  result.value?.trace?.candidates?.length
))

const agentRoleItems = computed(() => [
  { title: t('schedulerDiagnose.agentRoleDefault'), value: '' },
  { title: t('schedulerDiagnose.agentRoleMain'), value: 'main' },
  { title: t('schedulerDiagnose.agentRoleSubagent'), value: 'subagent' }
])

const parseOptionalNumber = (value: string): number | undefined => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number.parseInt(trimmed, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

const parseFailedChannels = (): number[] => {
  return failedChannelsText.value
    .split(/[\s,，]+/)
    .map(part => Number.parseInt(part, 10))
    .filter(index => Number.isInteger(index) && index >= 0)
}

const buildPayload = (): SchedulerDiagnoseRequest => {
  const input = parseOptionalNumber(inputTokens.value)
  const output = parseOptionalNumber(outputTokens.value)
  const required = parseOptionalNumber(requiredTokens.value)
  const hasContext = input !== undefined || output !== undefined || required !== undefined || explicitOutputMax.value || skipWindowValidation.value

  return {
    model: model.value.trim() || undefined,
    userId: userId.value.trim() || undefined,
    routePrefix: routePrefix.value.trim() || undefined,
    channelName: channelName.value.trim() || undefined,
    failedChannels: parseFailedChannels(),
    agentRole: agentRole.value || undefined,
    hasImageContent: hasImageContent.value,
    contextRequirement: hasContext
      ? {
          inputTokens: input,
          outputTokens: output,
          requiredTokens: required,
          explicitOutputMax: explicitOutputMax.value,
          skipWindowValidation: skipWindowValidation.value
        }
      : undefined
  }
}

const runDiagnose = async () => {
  isRunning.value = true
  try {
    result.value = await api.diagnoseSchedulerSelection(props.channelType, buildPayload())
  } catch (err) {
    result.value = {
      ok: false,
      kind: props.channelType,
      error: err instanceof Error ? err.message : String(err)
    }
  } finally {
    isRunning.value = false
  }
}

const clearResult = () => {
  result.value = null
}

watch(() => props.modelValue, (open) => {
  if (!open) return
  result.value = null
})
</script>

<style scoped>
.scheduler-diagnose-result {
  font-size: 0.875rem;
}

.scheduler-diagnose-label {
  color: rgba(var(--v-theme-on-surface), 0.68);
  font-size: 0.75rem;
  font-weight: 600;
  margin-bottom: 4px;
}

.scheduler-diagnose-code {
  display: block;
  padding: 8px;
  border-radius: 6px;
  background: rgba(var(--v-theme-surface-variant), 0.42);
  white-space: pre-wrap;
  word-break: break-all;
}

.scheduler-diagnose-table {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 6px;
}
</style>
