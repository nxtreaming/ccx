<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Check, ChevronDown, ChevronRight, Minus, RefreshCw } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useLanguage } from '@/composables/useLanguage'
import type { RoutingDecisionTrace } from '@/services/admin-api'

const props = defineProps<{
  traces: RoutingDecisionTrace[]
  loading: boolean
}>()

const emit = defineEmits<{
  refresh: []
}>()

const { t } = useLanguage()

const mismatchOnly = ref(false)
const expanded = reactive<Record<string, boolean>>({})

const filteredTraces = computed(() => {
  if (!mismatchOnly.value) return props.traces
  return props.traces.filter((tr) => !tr.match && tr.shadowChannelUid && tr.actualChannelUid)
})

function toggleExpand(uid: string) {
  expanded[uid] = !expanded[uid]
}

function formatTime(iso: string): string {
  if (!iso) return '-'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function shortenUid(uid?: string): string {
  if (!uid) return '-'
  const stripped = uid.replace(/^ch_/, '')
  return stripped.length > 8 ? `${stripped.slice(0, 8)}...` : stripped
}

function isComparable(trace: RoutingDecisionTrace): boolean {
  return trace.mode === 'shadow' && !!trace.shadowChannelUid && !!trace.actualChannelUid
}

function outcomeClass(outcome?: RoutingDecisionTrace['outcome']): string {
  if (outcome === 'success') return 'border-emerald-500 text-emerald-600'
  if (outcome === 'cancelled') return 'border-muted-foreground text-muted-foreground'
  if (outcome === 'attempt_failed') return 'border-amber-500 text-amber-600'
  return 'border-red-500 text-red-600'
}
</script>

<template>
  <div class="rounded-xl border border-border/60 bg-card/40 p-4">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-sm font-bold">{{ t('autopilot.traceTable.title') }}</h4>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <Switch
            :model-value="mismatchOnly"
            @update:model-value="mismatchOnly = Boolean($event)"
          />
          <span class="text-xs text-muted-foreground">{{ t('autopilot.traceTable.mismatchOnly') }}</span>
        </div>
        <Button variant="outline" size="sm" :disabled="loading" @click="emit('refresh')">
          <RefreshCw class="size-3.5" :class="{ 'animate-spin': loading }" />
          {{ t('app.actions.refresh') }}
        </Button>
      </div>
    </div>

    <div v-if="filteredTraces.length === 0 && !loading" class="py-8 text-center text-sm text-muted-foreground">
      {{ t('autopilot.traceTable.empty') }}
    </div>

    <div v-else class="overflow-hidden rounded-lg border border-border/50">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead class="w-8" />
            <TableHead>{{ t('autopilot.traceTable.col.time') }}</TableHead>
            <TableHead>{{ t('autopilot.traceTable.col.kind') }}</TableHead>
            <TableHead>{{ t('autopilot.traceTable.col.taskClass') }}</TableHead>
            <TableHead>{{ t('autopilot.traceTable.col.model') }}</TableHead>
            <TableHead>{{ t('autopilot.traceTable.col.shadowVsActual') }}</TableHead>
            <TableHead>{{ t('autopilot.traceTable.col.match') }}</TableHead>
            <TableHead>{{ t('autopilot.traceTable.col.mode') }}</TableHead>
            <TableHead class="w-[90px]">{{ t('autopilot.traceTable.col.outcome') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-for="trace in filteredTraces" :key="trace.traceUid">
            <TableRow class="cursor-pointer" @click="toggleExpand(trace.traceUid)">
              <TableCell>
                <component :is="expanded[trace.traceUid] ? ChevronDown : ChevronRight" class="size-4 text-muted-foreground" />
              </TableCell>
              <TableCell class="text-xs">{{ formatTime(trace.createdAt) }}</TableCell>
              <TableCell>
                <Badge variant="outline">{{ trace.requestKind }}</Badge>
              </TableCell>
              <TableCell>
                <Badge variant="secondary">{{ trace.taskClass || '-' }}</Badge>
              </TableCell>
              <TableCell class="max-w-[160px] truncate text-xs">{{ trace.requestedModel || '-' }}</TableCell>
              <TableCell>
                <div class="flex items-center gap-1 text-xs">
                  <Badge variant="secondary">{{ shortenUid(trace.shadowChannelUid) }}</Badge>
                  <span class="text-muted-foreground">→</span>
                  <Badge variant="outline">{{ shortenUid(trace.actualChannelUid) }}</Badge>
                </div>
              </TableCell>
              <TableCell>
                <Badge v-if="isComparable(trace)" :variant="trace.match ? 'default' : 'destructive'">
                  {{ trace.match ? t('autopilot.traceTable.yes') : t('autopilot.traceTable.no') }}
                </Badge>
                <span v-else class="text-xs text-muted-foreground">-</span>
              </TableCell>
              <TableCell>
                <Badge variant="outline">{{ t(`autopilot.mode.${trace.mode}`) || trace.mode }}</Badge>
              </TableCell>
              <TableCell>
                <Badge
                  v-if="trace.outcomeRecorded"
                  variant="outline"
                  :class="outcomeClass(trace.outcome)"
                >
                  {{ trace.outcome }}
                </Badge>
                <span v-else class="text-xs text-muted-foreground">-</span>
              </TableCell>
            </TableRow>
            <TableRow v-if="expanded[trace.traceUid]">
              <TableCell colspan="9" class="bg-muted/20 p-4">
                <div v-if="trace.candidates && trace.candidates.length > 0" class="mb-3">
                  <div class="mb-2 text-xs font-bold">{{ t('autopilot.traceTable.candidates') }}</div>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead class="text-[11px]">Channel UID</TableHead>
                        <TableHead class="text-[11px]">Origin Tier</TableHead>
                        <TableHead class="text-[11px]">Health</TableHead>
                        <TableHead class="text-[11px]">Score</TableHead>
                        <TableHead class="text-[11px]">Domain</TableHead>
                        <TableHead class="text-[11px]">Selected</TableHead>
                        <TableHead class="text-[11px]">Filter Reasons</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      <TableRow v-for="(cand, ci) in trace.candidates" :key="ci">
                        <TableCell class="text-[11px]">{{ cand.channelUid }}</TableCell>
                        <TableCell class="text-[11px]">{{ cand.originTier || '-' }}</TableCell>
                        <TableCell class="text-[11px]">{{ cand.healthState || '-' }}</TableCell>
                        <TableCell class="text-[11px]">{{ cand.totalScore.toFixed(3) }}</TableCell>
                        <TableCell class="text-[11px]">
                          <div>{{ cand.domainEvidence?.source || '-' }}</div>
                          <div v-if="cand.domainEvidence?.canonicalModel" class="text-muted-foreground">
                            {{ cand.domainEvidence.canonicalModel }} / {{ cand.domainEvidence.benchmarkCategory }}
                            · {{ cand.domainEvidence.canonicalCeiling?.toFixed(3) }} ×
                            {{ cand.domainEvidence.providerQualityFactor?.toFixed(3) }}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Check v-if="cand.selected" class="size-3.5 text-emerald-500" />
                          <Minus v-else class="size-3.5 text-muted-foreground" />
                        </TableCell>
                        <TableCell class="text-[11px]">{{ cand.filterReasons?.join('; ') || '-' }}</TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>
                </div>

                <div v-if="trace.sortReasons && trace.sortReasons.length > 0">
                  <div class="mb-1 text-xs font-bold">{{ t('autopilot.traceTable.sortReasons') }}</div>
                  <ul class="ml-4 list-disc text-[11px] text-muted-foreground">
                    <li v-for="(reason, ri) in trace.sortReasons" :key="ri">{{ reason }}</li>
                  </ul>
                </div>
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </div>
  </div>
</template>
