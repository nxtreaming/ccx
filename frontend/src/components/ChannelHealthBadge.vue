<template>
  <div
    v-if="health"
    class="channel-health-badge"
    @mouseenter="hovered = true"
    @mouseleave="hovered = false"
    @focusin="hovered = true"
    @focusout="hovered = false"
  >
    <div class="health-dot-wrapper" :class="`health-${health.aggState}`">
      <div class="health-dot"></div>
    </div>
    <v-tooltip
      v-if="hovered"
      v-model="hovered"
      activator="parent"
      location="top"
      :open-delay="0"
      content-class="ccx-tooltip"
    >
      <div class="health-tooltip">
        <div class="health-tooltip-title font-weight-bold mb-1">
          {{ t('channelHealth.stateLabel') }}: {{ stateLabel }}
        </div>
        <div class="health-tooltip-breakdown text-caption mb-1">
          {{ healthyCount }}/{{ health.endpointCount }}
          {{ t('channelHealth.endpointsHealthy') }}
        </div>
        <div v-if="health.avgSuccessRate != null" class="text-caption mb-1">
          {{ t('channelHealth.avgSuccessRate') }}: {{ (health.avgSuccessRate * 100).toFixed(1) }}%
        </div>
        <div v-if="warningCount > 0" class="health-tooltip-warning text-caption mt-1">
          <v-icon size="12" color="warning" class="mr-1">mdi-alert-circle-outline</v-icon>
          {{ t('channelHealth.inconsistentWarning', { count: warningCount }) }}
        </div>
      </div>
    </v-tooltip>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ChannelHealthItem } from '../services/api-types'
import { useI18n } from '../i18n'

const props = defineProps<{
  health?: ChannelHealthItem | null
}>()

const { t } = useI18n()
const hovered = ref(false)

const stateLabel = computed(() => {
  const state = props.health?.aggState || 'unknown'
  return t(`healthCenter.state.${state}`)
})

const healthyCount = computed(() => {
  if (!props.health) return 0
  return props.health.healthyCount
})

// Inconsistent = endpoints that are NOT healthy (degraded + limited + dead + unknown)
const warningCount = computed(() => {
  if (!props.health) return 0
  const h = props.health
  return h.degradedCount + h.limitedCount + h.deadCount + h.unknownCount
})
</script>

<style scoped>
.channel-health-badge {
  display: inline-flex;
  align-items: center;
  cursor: help;
  flex-shrink: 0;
}

.health-dot-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  position: relative;
}

.health-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* State colors - matching HealthCenter view palette */
.health-healthy .health-dot {
  background: rgb(var(--v-theme-success));
  box-shadow: 0 0 4px rgba(var(--v-theme-success), 0.5);
}

.health-degraded .health-dot {
  background: #f59e0b;
  box-shadow: 0 0 4px rgba(245, 158, 11, 0.5);
}

.health-limited .health-dot {
  background: #f97316;
  box-shadow: 0 0 4px rgba(249, 115, 22, 0.5);
}

.health-misconfigured .health-dot {
  background: #a855f7;
  box-shadow: 0 0 4px rgba(168, 85, 247, 0.5);
}

.health-dead .health-dot {
  background: rgb(var(--v-theme-error));
  box-shadow: 0 0 4px rgba(var(--v-theme-error), 0.5);
  animation: health-pulse-dead 1.5s ease-in-out infinite;
}

.health-unknown .health-dot {
  background: #94a3b8;
  box-shadow: 0 0 3px rgba(148, 163, 184, 0.4);
}

@keyframes health-pulse-dead {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Tooltip */
.health-tooltip {
  min-width: 120px;
}

.health-tooltip-title {
  font-size: 13px;
}

.health-tooltip-warning {
  color: rgb(var(--v-theme-warning));
}

/* Hide on very small screens */
@media (max-width: 480px) {
  .channel-health-badge {
    display: none;
  }
}
</style>
