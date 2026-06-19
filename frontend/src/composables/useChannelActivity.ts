import { computed, ref, watch, markRaw, type Ref } from 'vue'
import type { ChannelRecentActivity, ActivitySegment } from '../services/api'
import { expandSparseSegments } from '../services/api-helpers'

/**
 * Activity 可视化相关的状态和计算逻辑。
 * 从 ChannelOrchestration.vue 抽出，降低单文件行数。
 */
export function useChannelActivity(recentActivity: Ref<ChannelRecentActivity[]>) {
  const activityMap = computed(() => {
    const map = new Map<number, ChannelRecentActivity>()
    for (const a of recentActivity.value) {
      map.set(a.channelIndex, a)
    }
    return map
  })

  const maxRequestsHistory = ref(new Map<number, { max: number; updatedAt: number }>())
  const DECAY_HALF_LIFE = 5 * 60 * 1000  // Half-life: 5 minutes
  const MIN_MAX_REQUESTS = 1  // Minimum baseline value to avoid division by zero

  const getDecayedMax = (record: { max: number; updatedAt: number }, now: number): number => {
    const elapsed = now - record.updatedAt
    const decayFactor = Math.pow(0.5, elapsed / DECAY_HALF_LIFE)
    return Math.max(MIN_MAX_REQUESTS, record.max * decayFactor)
  }

  watch(activityMap, (newMap) => {
    const now = Date.now()
    for (const [channelIndex, activity] of newMap.entries()) {
      const currentMax = activity.rpm
      if (currentMax <= 0) continue

      const record = maxRequestsHistory.value.get(channelIndex)
      if (!record) {
        maxRequestsHistory.value.set(channelIndex, { max: currentMax, updatedAt: now })
        continue
      }

      const decayedMax = getDecayedMax(record, now)
      if (currentMax > decayedMax) {
        maxRequestsHistory.value.set(channelIndex, { max: currentMax, updatedAt: now })
      } else {
        maxRequestsHistory.value.set(channelIndex, { max: decayedMax, updatedAt: now })
      }
    }
    // Clean up stale entries
    for (const key of maxRequestsHistory.value.keys()) {
      if (!newMap.has(key)) {
        maxRequestsHistory.value.delete(key)
      }
    }
  })

  const getChannelActivity = (channelIndex: number): ChannelRecentActivity | undefined => {
    return activityMap.value.get(channelIndex)
  }

  type ActivityBar = { x: number; y: number; width: number; height: number; radius: number; g: number; v: 0 | 1 }

  const activityBarsPersistentCache = new Map<number, { segments: ChannelRecentActivity['segments'], bars: ActivityBar[] }>()

  const activityBarsCache = computed(() => {
    const cache = new Map<number, ActivityBar[]>()

    for (const [channelIndex, activity] of activityMap.value.entries()) {
      const segments = activity.segments
      if (!segments || Object.keys(segments).length === 0) {
        cache.set(channelIndex, [])
        continue
      }

      const existing = activityBarsPersistentCache.get(channelIndex)
      if (existing && existing.segments === segments) {
        cache.set(channelIndex, existing.bars)
        continue
      }

      const now = Date.now()
      const record = maxRequestsHistory.value.get(channelIndex)
      const maxRPM = activity.rpm || 1
      const currentMax = maxRPM
      const maxRequests = record ? Math.max(getDecayedMax(record, now), currentMax) : currentMax

      let bars: ActivityBar[]

      const expanded = expandSparseSegments(activity)
      const totalSegs = activity.totalSegs || 150
      const barWidth = 1 / totalSegs

      bars = expanded.map((seg, i) => {
        const x = i * barWidth
        const successRate = seg.requestCount > 0 ? seg.successCount / seg.requestCount : 0
        const height = maxRequests > 0 ? (seg.requestCount / (maxRequests * barWidth * 60)) : 0
        const clampedHeight = Math.min(1, height)
        const g = successRate
        const v: 0 | 1 = seg.failureCount > 0 ? 1 : 0
        return { x, y: 1 - clampedHeight, width: barWidth, height: clampedHeight, radius: 0, g, v }
      })

      activityBarsPersistentCache.set(channelIndex, { segments: activity.segments, bars: markRaw(bars) })
      cache.set(channelIndex, bars)
    }

    for (const key of activityBarsPersistentCache.keys()) {
      if (!activityMap.value.has(key)) {
        activityBarsPersistentCache.delete(key)
      }
    }

    return cache
  })

  const getActivityBars = (channelIndex: number): ActivityBar[] => {
    return activityBarsCache.value.get(channelIndex) || []
  }

  const getActivityPath = (channelIndex: number): string => {
    const bars = getActivityBars(channelIndex)
    if (bars.length === 0) return ''

    const points: { x: number; y: number }[] = []
    const step = 3
    for (let i = 0; i < bars.length; i += step) {
      points.push({ x: bars[i].x, y: bars[i].y })
    }
    if (points.length < 2) return ''

    const pathParts: string[] = [`M ${points[0].x} ${points[0].y}`]
    for (let i = 1; i < points.length; i++) {
      const prev = points[i - 1]
      const curr = points[i]
      const cpx = (prev.x + curr.x) / 2
      pathParts.push(`C ${cpx} ${prev.y}, ${cpx} ${curr.y}, ${curr.x} ${curr.y}`)
    }
    return pathParts.join(' ')
  }

  function catmullRomToPath(points: { x: number; y: number }[]): string {
    if (points.length < 2) return ''
    const parts: string[] = [`M ${points[0].x.toFixed(4)} ${points[0].y.toFixed(4)}`]
    for (let i = 0; i < points.length - 1; i++) {
      const p0 = points[Math.max(0, i - 1)]
      const p1 = points[i]
      const p2 = points[i + 1]
      const p3 = points[Math.min(points.length - 1, i + 2)]
      const cp1x = p1.x + (p2.x - p0.x) / 6
      const cp1y = p1.y + (p2.y - p0.y) / 6
      const cp2x = p2.x - (p3.x - p1.x) / 6
      const cp2y = p2.y - (p3.y - p1.y) / 6
      parts.push(`C ${cp1x.toFixed(4)} ${cp1y.toFixed(4)}, ${cp2x.toFixed(4)} ${cp2y.toFixed(4)}, ${p2.x.toFixed(4)} ${p2.y.toFixed(4)}`)
    }
    return parts.join(' ')
  }

  const _getActivityAreaPath = (channelIndex: number): string => {
    const bars = getActivityBars(channelIndex)
    if (bars.length === 0) return ''
    const points: { x: number; y: number }[] = []
    const step = 3
    for (let i = 0; i < bars.length; i += step) {
      points.push({ x: bars[i].x, y: bars[i].y })
    }
    if (points.length < 2) return ''
    const areaPoints = [
      { x: points[0].x, y: 1 },
      ...points,
      { x: points[points.length - 1].x, y: 1 },
    ]
    return catmullRomToPath(areaPoints) + ' Z'
  }

  const _getActivityGradient = (channelIndex: number): string => {
    const bars = getActivityBars(channelIndex)
    if (bars.length === 0) return ''
    let totalG = 0
    let count = 0
    for (const bar of bars) {
      if (bar.height > 0.01) {
        totalG += bar.g
        count++
      }
    }
    if (count === 0) return 'rgba(100, 100, 100, 0.3)'
    const avgG = totalG / count
    const r = Math.round(255 * (1 - avgG))
    const g = Math.round(255 * avgG)
    return `rgba(${r}, ${g}, 50, 0.6)`
  }

  const formatRPM = (channelIndex: number): string => {
    const activity = getChannelActivity(channelIndex)
    if (!activity) return '0'
    return activity.rpm >= 1000 ? `${(activity.rpm / 1000).toFixed(1)}k` : Math.round(activity.rpm).toString()
  }

  const formatTPM = (channelIndex: number): string => {
    const activity = getChannelActivity(channelIndex)
    if (!activity) return '0'
    return activity.tpm >= 1000000
      ? `${(activity.tpm / 1000000).toFixed(1)}M`
      : activity.tpm >= 1000
        ? `${(activity.tpm / 1000).toFixed(1)}k`
        : Math.round(activity.tpm).toString()
  }

  const hasActivityData = (channelIndex: number): boolean => {
    const activity = getChannelActivity(channelIndex)
    if (!activity) return false
    return activity.rpm > 0 || activity.tpm > 0
  }

  return {
    activityMap,
    getChannelActivity,
    getActivityBars,
    getActivityPath,
    _getActivityAreaPath,
    _getActivityGradient,
    formatRPM,
    formatTPM,
    hasActivityData,
  }
}
