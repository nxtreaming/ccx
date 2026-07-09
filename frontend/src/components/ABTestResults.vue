<template>
  <v-card variant="outlined" rounded="lg">
    <v-card-title class="d-flex align-center text-subtitle-1 font-weight-bold pb-0">
      <v-icon size="20" class="mr-2" color="purple">mdi-flask-outline</v-icon>
      {{ t('autopilot.abTest.title') }}
    </v-card-title>

    <v-card-text>
      <!-- 状态指示 -->
      <v-alert
        v-if="results?.killSwitchActive"
        type="warning"
        variant="tonal"
        density="compact"
        class="mb-3"
        :text="t('autopilot.abTest.killSwitchActive')"
      />

      <v-alert
        v-if="!results?.enabled && !results?.killSwitchActive"
        type="info"
        variant="tonal"
        density="compact"
        class="mb-3"
        :text="t('autopilot.abTest.disabled')"
      />

      <!-- 配置概览 -->
      <v-row dense class="mb-3">
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">
              <v-icon v-if="results?.enabled" size="16" color="success">mdi-check-circle</v-icon>
              <v-icon v-else size="16" color="grey">mdi-close-circle</v-icon>
            </div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.enabled') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ sampleRatioDisplay }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.sampleRatio') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ results?.budgetUsed ?? 0 }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.budgetUsed') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ results?.budgetRemaining ?? 0 }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.budgetRemaining') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" color="warning" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">${{ totalCostDisplay }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.totalCost') }}</div>
          </v-card>
        </v-col>
      </v-row>

      <!-- 汇总统计 -->
      <v-row v-if="results?.stats && results.stats.totalRecords > 0" dense class="mb-3">
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ results.stats.totalRecords }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.totalRecords') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ results.stats.shadowSuccessCount }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.shadowSuccess') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ results.stats.shadowFailCount }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.shadowFail') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" :color="shadowSuccessRateColor" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ shadowSuccessRateDisplay }}</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.shadowSuccessRate') }}</div>
          </v-card>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-card variant="tonal" rounded="lg" class="pa-3 text-center">
            <div class="text-h5 font-weight-bold">{{ avgLatencyDisplay }}ms</div>
            <div class="text-caption text-medium-emphasis">{{ t('autopilot.abTest.avgShadowLatency') }}</div>
          </v-card>
        </v-col>
      </v-row>

      <!-- 按渠道分组统计 -->
      <div v-if="channelStatsItems.length > 0" class="mb-3">
        <div class="text-caption text-medium-emphasis mb-2">{{ t('autopilot.abTest.byChannel') }}</div>
        <v-table density="compact">
          <thead>
            <tr>
              <th class="text-caption">{{ t('autopilot.abTest.channel') }}</th>
              <th class="text-caption text-right">{{ t('autopilot.abTest.count') }}</th>
              <th class="text-caption text-right">{{ t('autopilot.abTest.successRate') }}</th>
              <th class="text-caption text-right">{{ t('autopilot.abTest.avgLatency') }}</th>
              <th class="text-caption text-right">{{ t('autopilot.abTest.cost') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in channelStatsItems" :key="item.channelUid">
              <td class="text-caption">{{ item.channelUid }}</td>
              <td class="text-caption text-right">{{ item.count }}</td>
              <td class="text-caption text-right">{{ (item.successRate * 100).toFixed(1) }}%</td>
              <td class="text-caption text-right">{{ item.avgLatencyMs.toFixed(0) }}ms</td>
              <td class="text-caption text-right">${{ item.totalCostUsd.toFixed(4) }}</td>
            </tr>
          </tbody>
        </v-table>
      </div>

      <!-- 最近记录 -->
      <div v-if="results?.recentRecords && results.recentRecords.length > 0" class="mb-3">
        <div class="text-caption text-medium-emphasis mb-2">{{ t('autopilot.abTest.recentRecords') }}</div>
        <v-table density="compact">
          <thead>
            <tr>
              <th class="text-caption">{{ t('autopilot.abTest.model') }}</th>
              <th class="text-caption">{{ t('autopilot.abTest.primary') }}</th>
              <th class="text-caption">{{ t('autopilot.abTest.shadow') }}</th>
              <th class="text-caption text-right">{{ t('autopilot.abTest.latency') }}</th>
              <th class="text-caption text-right">{{ t('autopilot.abTest.cost') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="record in results.recentRecords.slice(0, 20)" :key="record.recordUid">
              <td class="text-caption">{{ record.model }}</td>
              <td class="text-caption">
                <v-icon size="14" :color="record.primarySuccess ? 'success' : 'error'">
                  {{ record.primarySuccess ? 'mdi-check-circle' : 'mdi-close-circle' }}
                </v-icon>
                {{ record.primaryStatusCode }}
              </td>
              <td class="text-caption">
                <v-icon size="14" :color="record.shadowSuccess ? 'success' : 'error'">
                  {{ record.shadowSuccess ? 'mdi-check-circle' : 'mdi-close-circle' }}
                </v-icon>
                {{ record.shadowStatusCode || record.shadowError || '-' }}
              </td>
              <td class="text-caption text-right">
                {{ record.primaryLatencyMs }}ms / {{ record.shadowLatencyMs }}ms
              </td>
              <td class="text-caption text-right">${{ record.shadowCostUsd.toFixed(4) }}</td>
            </tr>
          </tbody>
        </v-table>
      </div>

      <!-- 紧急停止按钮 -->
      <v-btn
        v-if="results?.enabled"
        color="error"
        variant="tonal"
        size="small"
        class="mt-2"
        :loading="stopping"
        @click="handleEmergencyStop"
      >
        <v-icon size="16" class="mr-1">mdi-stop-circle-outline</v-icon>
        {{ t('autopilot.abTest.emergencyStop') }}
      </v-btn>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/services/api'
import type { ABTestResultsResponse } from '@/services/api-types'

const { t } = useI18n()

const results = ref<ABTestResultsResponse | null>(null)
const stopping = ref(false)

const sampleRatioDisplay = computed(() => {
  if (!results.value) return '-'
  return `${(results.value.sampleRatio * 100).toFixed(1)}%`
})

const totalCostDisplay = computed(() => {
  if (!results.value) return '0.00'
  return results.value.totalShadowCostUsd.toFixed(4)
})

const shadowSuccessRateDisplay = computed(() => {
  if (!results.value?.stats) return '-'
  return `${(results.value.stats.shadowSuccessRate * 100).toFixed(1)}%`
})

const shadowSuccessRateColor = computed(() => {
  if (!results.value?.stats) return undefined
  const rate = results.value.stats.shadowSuccessRate
  if (rate >= 0.95) return 'success'
  if (rate >= 0.8) return 'warning'
  return 'error'
})

const avgLatencyDisplay = computed(() => {
  if (!results.value?.stats) return '-'
  return results.value.stats.avgShadowLatencyMs.toFixed(0)
})

const channelStatsItems = computed(() => {
  if (!results.value?.stats?.byChannel) return []
  return Object.values(results.value.stats.byChannel).sort((a, b) => b.count - a.count)
})

async function loadResults() {
  try {
    results.value = await api.getABTestResults()
  } catch (e) {
    console.error('[ABTest-UI] 加载失败:', e)
  }
}

async function handleEmergencyStop() {
  stopping.value = true
  try {
    await api.emergencyStopABTest('UI emergency stop')
    await loadResults()
  } catch (e) {
    console.error('[ABTest-UI] 紧急停止失败:', e)
  } finally {
    stopping.value = false
  }
}

onMounted(() => {
  loadResults()
})
</script>
