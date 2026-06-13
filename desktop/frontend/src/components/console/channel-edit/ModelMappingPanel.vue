<script setup lang="ts">
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ArrowRight, Eye, EyeOff, Plus, RotateCcw, Trash2, Zap } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

type ReasoningEffort = 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'

interface ModelMappingRow {
  id: number
  source: string
  target: string
  reasoning: ReasoningEffort | ''
  noVision: boolean
}

const props = defineProps<{
  modelMappingRows: ModelMappingRow[]
  newModelMapping: ModelMappingRow
  reasoningEffortOptions: Array<{ label: string; value: string }>
  targetModelDatalist: string[]
  channelType: string
  showTargetSuggestions: boolean
  activeTargetInputId: string | null
  targetInputFilter: string
  DEFAULT_SELECT_VALUE: string
}>()

const emit = defineEmits<{
  'update:newModelMapping': [value: Partial<ModelMappingRow>]
  'addModelMappingRow': []
  'removeModelMappingRow': [id: number]
  'updateMappingRow': [id: number, field: keyof ModelMappingRow, value: any]
  'syncUpstreamModels': []
  'applyPreset': [name: string]
  'showTargetDropdown': [inputId: string, currentValue: string]
  'hideTargetDropdown': []
  'selectTargetModel': [inputId: string, model: string]
  'handleTargetFocus': []
}>()

const { tf } = useLanguage()

const presetTags = ref(['gpt-5.5', 'gpt-5.4', 'MiMo', 'DeepSeek', 'MiniMax'])

function toSelectValue(effort: ReasoningEffort | ''): string {
  return effort === '' ? props.DEFAULT_SELECT_VALUE : effort
}

function fromSelectValue(value: string): ReasoningEffort | '' {
  return value === props.DEFAULT_SELECT_VALUE ? '' : (value as ReasoningEffort)
}
</script>

<template>
  <section class="space-y-4 rounded-xl border border-primary/20 bg-gradient-to-b from-primary/[0.02] to-transparent p-5 shadow-sm">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border/40 pb-3">
      <div class="space-y-0.5">
        <h4 class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-foreground">
          <span class="h-2 w-2 rounded-full bg-primary shadow-[0_0_10px_hsl(var(--primary))] animate-pulse"></span>
          {{ tf('console.form.modelMapping', '模型重定向映射') }}
        </h4>
        <p class="text-[10px] text-muted-foreground">
          拦截调用请求中的 Source 别名并定向投递至上游 Target 真实模型
        </p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="h-8 rounded-full border border-border/80 bg-background/60 px-3 text-xs font-medium text-muted-foreground hover:text-foreground hover:border-primary/40 hover:bg-primary/5 shadow-2xs"
        @click="emit('syncUpstreamModels')"
      >
        <RotateCcw class="h-3.5 w-3.5 mr-1 text-primary" />
        {{ tf('console.form.syncModels', '同步上游可用模型') }}
      </Button>
    </div>

    <!-- 预设快速注入 -->
    <div class="rounded-lg border border-border/50 bg-card/40 p-3 space-y-2">
      <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/80">
        <Zap class="h-3 w-3 text-primary" />
        {{ tf('console.form.presets', '预设快速注入') }}
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <Button
          v-for="tag in presetTags"
          :key="tag"
          type="button"
          variant="outline"
          size="sm"
          class="h-6 rounded-md border border-border/70 bg-background px-2.5 text-[10px] font-medium text-muted-foreground hover:border-primary/40 hover:text-primary hover:bg-primary/5 shadow-3xs"
          @click="emit('applyPreset', tag)"
        >
          {{ tag }}
        </Button>
      </div>
    </div>

    <!-- 已有映射列表 -->
    <div v-if="modelMappingRows.length" class="space-y-2.5">
      <div class="flex items-center justify-between text-[10px] font-bold uppercase tracking-wider text-muted-foreground/60 px-1">
        <span>{{ tf('console.form.existingMappings', '已有条目映射') }}</span>
        <span class="rounded-full bg-primary/10 border border-primary/20 px-2 py-0.5 font-mono text-primary">
          {{ modelMappingRows.length }} {{ tf('console.form.mappings', '条') }}
        </span>
      </div>

      <div class="grid gap-2">
        <div
          v-for="(row, index) in modelMappingRows"
          :key="row.id"
          class="group grid grid-cols-[1fr_auto_1.5fr_auto_auto] items-center gap-3 rounded-xl border border-border/60 bg-background/50 p-2 shadow-2xs transition-all hover:border-primary/30 hover:bg-background/80"
        >
          <!-- SOURCE -->
          <div class="min-w-0 rounded-lg border border-border/50 bg-muted/40 px-3 py-1.5 h-10 flex flex-col justify-center">
            <span class="text-[8px] font-bold tracking-wider text-muted-foreground/50 leading-none">SOURCE</span>
            <span class="truncate font-mono text-xs font-semibold text-foreground mt-0.5" :title="row.source">
              {{ row.source }}
            </span>
          </div>

          <!-- Arrow -->
          <div class="flex justify-center">
            <span class="flex size-7 items-center justify-center rounded-full border border-primary/10 bg-primary/5 text-primary/70 transition-colors group-hover:border-primary/30 group-hover:bg-primary/10">
              <ArrowRight class="h-3.5 w-3.5" />
            </span>
          </div>

          <!-- TARGET -->
          <div class="space-y-0.5 relative">
            <span class="text-[8px] font-bold tracking-wider text-muted-foreground/50 block pl-1">TARGET</span>
            <Input
              :model-value="row.target"
              class="h-8 w-full rounded-lg border border-border/70 bg-background px-2.5 font-mono text-xs outline-none transition-all focus:border-primary/60 focus:ring-2 focus:ring-primary/10"
              placeholder="target-model"
              @update:model-value="(val) => emit('updateMappingRow', row.id, 'target', val as string)"
              @focus="emit('handleTargetFocus'); emit('showTargetDropdown', `row-${index}`, row.target)"
            />
          </div>

          <!-- VERBOSITY (Reasoning) -->
          <div class="space-y-0.5">
            <span class="text-[8px] font-bold tracking-wider text-muted-foreground/50 block pl-1">VERBOSITY</span>
            <Select
              :model-value="toSelectValue(row.reasoning)"
              @update:model-value="(val) => emit('updateMappingRow', row.id, 'reasoning', fromSelectValue(val as string))"
            >
              <SelectTrigger class="h-8 rounded-lg border border-border/70 bg-background/60 px-2.5 text-xs text-muted-foreground hover:text-foreground">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in reasoningEffortOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-1 pl-1">
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              class="h-7 w-7 rounded-full border border-border/50 bg-background/40 text-muted-foreground hover:border-amber-500/40 hover:bg-amber-500/5 hover:text-amber-500"
              :title="tf('console.form.toggleVision', '切换视觉通道')"
              @click="emit('updateMappingRow', row.id, 'noVision', !row.noVision)"
            >
              <EyeOff v-if="row.noVision" class="h-3.5 w-3.5" />
              <Eye v-else class="h-3.5 w-3.5" />
            </Button>
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              class="h-7 w-7 rounded-full border border-border/50 bg-background/40 text-muted-foreground hover:border-destructive/30 hover:bg-destructive/5 hover:text-destructive"
              @click="emit('removeModelMappingRow', row.id)"
            >
              <Trash2 class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加新映射 -->
    <div class="rounded-xl border border-primary/20 bg-primary/[0.02] p-3 space-y-2">
      <div class="text-[10px] font-bold uppercase tracking-wider text-primary">
        {{ tf('console.form.addNewMapping', '添加新映射关系') }}
      </div>
      <div class="grid grid-cols-[1fr_auto_1.5fr_auto_auto] items-end gap-3">
        <div class="space-y-1">
          <span class="text-[8px] font-bold tracking-wider text-muted-foreground/60 block pl-1">SOURCE</span>
          <Input
            :model-value="newModelMapping.source"
            class="h-9 w-full rounded-lg border border-primary/20 bg-background px-3 font-mono text-xs outline-none focus:border-primary/50"
            placeholder="source-model"
            @update:model-value="(val) => emit('update:newModelMapping', { source: val as string })"
          />
        </div>
        <div class="flex h-9 items-center text-primary/60">
          <ArrowRight class="h-3.5 w-3.5" />
        </div>
        <div class="space-y-1">
          <span class="text-[8px] font-bold tracking-wider text-muted-foreground/60 block pl-1">TARGET</span>
          <Input
            :model-value="newModelMapping.target"
            class="h-9 w-full rounded-lg border border-primary/20 bg-background px-3 font-mono text-xs outline-none focus:border-primary/50"
            placeholder="target-model"
            @update:model-value="(val) => emit('update:newModelMapping', { target: val as string })"
          />
        </div>
        <div class="space-y-1">
          <span class="text-[8px] font-bold tracking-wider text-muted-foreground/60 block pl-1">VERBOSITY</span>
          <Select
            :model-value="toSelectValue(newModelMapping.reasoning)"
            @update:model-value="(val) => emit('update:newModelMapping', { reasoning: fromSelectValue(val as string) as any })"
          >
            <SelectTrigger class="h-9 rounded-lg border border-primary/20 bg-background px-3 text-xs text-muted-foreground">
              <SelectValue :placeholder="tf('console.form.selectDefault', '默认')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="opt in reasoningEffortOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button
          type="button"
          variant="default"
          size="sm"
          class="h-9 rounded-lg bg-primary hover:bg-primary/90 px-4 text-xs font-semibold text-primary-foreground shadow-sm"
          :disabled="!newModelMapping.source.trim() || !newModelMapping.target.trim()"
          @click="emit('addModelMappingRow')"
        >
          <Plus class="h-3.5 w-3.5 mr-1" />
          {{ tf('console.form.createMapping', '建立映射') }}
        </Button>
      </div>
    </div>
  </section>
</template>
