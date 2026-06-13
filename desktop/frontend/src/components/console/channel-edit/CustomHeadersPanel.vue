<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Plus, Trash2 } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

interface HeaderRow {
  id: number
  key: string
  value: string
}

const props = defineProps<{
  headerRows: HeaderRow[]
  newHeader: HeaderRow
}>()

const emit = defineEmits<{
  'update:newHeader': [value: Partial<HeaderRow>]
  'addHeaderRow': []
  'removeHeaderRow': [id: number]
  'updateHeaderRow': [id: number, field: 'key' | 'value', value: string]
}>()

const { tf } = useLanguage()
</script>

<template>
  <section class="space-y-4 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
    <h4 class="text-xs font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
      {{ tf('console.form.customHeaders', '自定义请求标头 Headers（JSON 注入）') }}
    </h4>

    <!-- 已有 Headers 列表 -->
    <div v-if="headerRows.length" class="space-y-2">
      <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/60 px-1">
        {{ tf('console.form.existingHeaders', '已配置标头') }}
      </div>
      <div
        v-for="row in headerRows"
        :key="row.id"
        class="flex items-center gap-2 p-2 rounded-lg border border-border/60 bg-background/60 hover:bg-background transition-colors"
      >
        <div class="flex-1 grid grid-cols-[140px_1fr] gap-2">
          <Input
            :model-value="row.key"
            class="h-9 font-mono text-xs"
            placeholder="Header-Name"
            @update:model-value="(val) => emit('updateHeaderRow', row.id, 'key', val as string)"
          />
          <Input
            :model-value="row.value"
            class="h-9 font-mono text-xs"
            placeholder="Header Value"
            @update:model-value="(val) => emit('updateHeaderRow', row.id, 'value', val as string)"
          />
        </div>
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          class="h-9 w-9 shrink-0 text-destructive hover:bg-destructive/10"
          @click="emit('removeHeaderRow', row.id)"
        >
          <Trash2 class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <!-- 添加新 Header -->
    <div>
      <Label class="text-xs font-semibold text-muted-foreground mb-2 block">
        {{ tf('console.form.addNewHeader', '添加新标头') }}
      </Label>
      <div class="flex gap-2">
        <Input
          :model-value="newHeader.key"
          class="h-9 w-40 font-mono text-xs"
          placeholder="Header-Name"
          @update:model-value="(val) => emit('update:newHeader', { key: val as string })"
          @keydown.enter.prevent="emit('addHeaderRow')"
        />
        <Input
          :model-value="newHeader.value"
          class="h-9 flex-1 font-mono text-xs"
          placeholder="标头携带对应的 Value 内容..."
          @update:model-value="(val) => emit('update:newHeader', { value: val as string })"
          @keydown.enter.prevent="emit('addHeaderRow')"
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="h-9 px-3.5 shadow-3xs"
          :disabled="!newHeader.key.trim() || !newHeader.value.trim()"
          @click="emit('addHeaderRow')"
        >
          <Plus class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <!-- 说明 -->
    <div class="text-xs text-muted-foreground bg-muted/30 p-3 rounded-lg border border-border/30">
      <p class="font-semibold mb-1">{{ tf('console.form.headersNote', '说明') }}</p>
      <ul class="space-y-1 text-[11px] leading-relaxed">
        <li>• 自定义 Headers 将在每个请求中注入到上游</li>
        <li>• 支持覆盖默认 Headers（如 User-Agent、Authorization 等）</li>
        <li>• 键名和值都支持模板变量（如果后端实现了变量替换）</li>
      </ul>
    </div>
  </section>
</template>
