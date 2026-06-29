<script setup lang="ts">
import { TooltipContent, TooltipPortal, type TooltipContentProps } from 'reka-ui'
import { cn } from '@/lib/utils'

interface Props extends TooltipContentProps {
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  sideOffset: 4,
  collisionPadding: 8,
})
</script>

<template>
  <TooltipPortal>
    <TooltipContent
      v-bind="props"
      :class="cn(
        'z-50 max-w-[min(28rem,calc(100vw-1rem))] overflow-hidden rounded-md border bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md whitespace-normal break-words animate-in fade-in-0 zoom-in-95',
        props.class
      )"
    >
      <slot />
    </TooltipContent>
  </TooltipPortal>
</template>
