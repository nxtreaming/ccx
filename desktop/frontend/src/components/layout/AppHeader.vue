<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { useStatus } from '@/composables/useStatus'

const { status } = useStatus()

const statusText = computed(() => {
  if (status.value.running) return '运行中'
  if (status.value.starting) return '启动中'
  return '已停止'
})

const statusVariant = computed(() => {
  if (status.value.running) return 'default' as const
  if (status.value.starting) return 'secondary' as const
  return 'destructive' as const
})

const statusColor = computed(() => {
  if (status.value.running) return 'bg-accent text-accent-foreground'
  if (status.value.starting) return 'bg-warning text-warning-foreground'
  return 'bg-destructive text-destructive-foreground'
})
</script>

<template>
  <header class="sticky top-0 z-50 w-full bg-background/80 backdrop-blur-md border-b border-border">
    <div class="flex h-14 items-center justify-between pr-6 pl-24">
      <h1 class="text-lg font-semibold tracking-tight">CCX Desktop</h1>
      <Badge :class="statusColor" variant="outline" class="border-0 font-bold">
        {{ statusText }}
      </Badge>
    </div>
  </header>
</template>
