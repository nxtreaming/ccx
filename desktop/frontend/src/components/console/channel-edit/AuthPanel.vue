<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ArrowDown,
  ArrowUp,
  CheckCircle2,
  Copy,
  Key,
  Loader2,
  Plus,
  RotateCcw,
  Trash2,
} from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

interface DisabledKeyInfo {
  key: string
  reason?: string
  disabledAt?: string
}

const props = defineProps<{
  existingApiKeys: string[]
  newApiKeysText: string
  copiedKeyIndex: number | null
  disabledApiKeys: DisabledKeyInfo[]
  historicalApiKeys: string[]
  restoringKey: string
  localRestoredKeys: Set<string>
  errors: { apiKeys?: string }
}>()

const emit = defineEmits<{
  'update:newApiKeysText': [value: string]
  'addNewApiKeys': []
  'removeExistingApiKey': [index: number]
  'moveApiKeyToTop': [index: number]
  'moveApiKeyToBottom': [index: number]
  'copyApiKey': [key: string, index: number]
  'handleDisabledKeyRestore': [key: string]
}>()

const { tf } = useLanguage()

function maskApiKey(key: string): string {
  if (!key) return ''
  if (key.length <= 8) return key
  return key.slice(0, 4) + '***' + key.slice(-4)
}

function findDuplicateKeyIndex(key: string): number {
  return props.existingApiKeys.indexOf(key)
}

const hasDisabledKeys = computed(() => props.disabledApiKeys.length > 0)

const visibleDisabledKeys = computed(() => {
  return props.disabledApiKeys.filter(item => !props.localRestoredKeys.has(item.key))
})

const usableKeys = computed(() => {
  return props.existingApiKeys.filter(k => k.trim())
})
</script>

<template>
  <section class="space-y-4 rounded-xl border bg-card/40 p-5 shadow-xs" :class="errors.apiKeys ? 'border-destructive/40' : 'border-border/60'">
    <div class="flex items-center justify-between border-b border-border/40 pb-2">
      <h4 class="text-xs font-bold uppercase tracking-wider text-primary">
        {{ tf('console.form.authentication', '鉴权认证') }} *
      </h4>
      <span class="text-[10px] bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 font-semibold px-2 py-0.5 rounded-full">
        {{ usableKeys.length }} {{ tf('console.keys.active', '个有效活跃密钥') }}
      </span>
    </div>

    <p v-if="errors.apiKeys" class="text-[10px] text-destructive">{{ errors.apiKeys }}</p>

    <!-- API Keys 列表 -->
    <div v-if="existingApiKeys.length" class="grid gap-2 max-h-[160px] overflow-y-auto pr-1">
      <div
        v-for="(key, index) in existingApiKeys"
        :key="`${index}-${key}`"
        class="flex items-center justify-between gap-4 border border-border/60 bg-background/60 hover:bg-background rounded-lg px-3 py-2 text-xs shadow-2xs transition-all group"
      >
        <div class="flex min-w-0 items-center gap-2.5">
          <Key class="h-3.5 w-3.5 shrink-0 text-primary/70" />
          <code class="font-mono text-muted-foreground font-medium select-all">{{ maskApiKey(key) }}</code>
          <span
            v-if="findDuplicateKeyIndex(key) !== index && existingApiKeys.indexOf(key) !== index"
            class="text-[10px] text-amber-600 shrink-0"
          >
            {{ tf('addChannel.duplicateKey', '重复') }}
          </span>
        </div>
        <div class="flex shrink-0 items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            class="h-7 w-7 rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
            :class="copiedKeyIndex === index ? 'text-emerald-500' : ''"
            :title="tf('common.copy', '复制密钥')"
            @click="emit('copyApiKey', key, index)"
          >
            <CheckCircle2 v-if="copiedKeyIndex === index" class="h-3.5 w-3.5" />
            <Copy v-else class="h-3.5 w-3.5" />
          </Button>
          <Button
            v-if="index > 0"
            type="button"
            size="icon-sm"
            variant="ghost"
            class="h-7 w-7 rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="emit('moveApiKeyToTop', index)"
          >
            <ArrowUp class="h-3.5 w-3.5" />
          </Button>
          <Button
            v-if="index < existingApiKeys.length - 1"
            type="button"
            size="icon-sm"
            variant="ghost"
            class="h-7 w-7 rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="emit('moveApiKeyToBottom', index)"
          >
            <ArrowDown class="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            class="h-7 w-7 rounded-md text-destructive hover:bg-destructive/10"
            :title="tf('common.delete', '删除密钥')"
            @click="emit('removeExistingApiKey', index)"
          >
            <Trash2 class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </div>

    <!-- 添加新 API Key -->
    <div class="flex gap-2">
      <Input
        :model-value="newApiKeysText"
        class="h-9 flex-1 rounded-lg border border-input bg-background/40 px-3 font-mono text-xs placeholder:text-muted-foreground/60 outline-none transition-all focus:border-primary focus:ring-2 focus:ring-primary/20"
        :placeholder="tf('addChannel.addNewApiKeyPlaceholder', '追加输入新 API Key，按下回车或点击右侧添加')"
        @update:model-value="(val) => emit('update:newApiKeysText', val as string)"
        @keydown.enter.prevent="emit('addNewApiKeys')"
      />
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="h-9 rounded-lg border border-input bg-background/80 hover:bg-accent px-4 text-xs font-semibold shadow-2xs"
        :disabled="!newApiKeysText.trim()"
        @click="emit('addNewApiKeys')"
      >
        <Plus class="h-4 w-4 mr-1" />
        {{ tf('common.add', '添加 Key') }}
      </Button>
    </div>

    <!-- Disabled Keys -->
    <div v-if="hasDisabledKeys && visibleDisabledKeys.length" class="space-y-2 border border-amber-500/20 bg-amber-500/10 p-3 rounded-lg">
      <div class="text-[10px] font-bold uppercase tracking-wider text-amber-700 dark:text-amber-300">
        {{ tf('console.form.disabledKeys', 'Disabled keys') }} ({{ visibleDisabledKeys.length }})
      </div>
      <div v-for="item in visibleDisabledKeys" :key="item.key" class="flex items-center justify-between gap-2 text-xs">
        <div class="min-w-0 space-y-0.5">
          <div class="flex min-w-0 items-center gap-1.5">
            <span class="truncate font-mono text-muted-foreground">{{ maskApiKey(item.key) }}</span>
            <span
              v-if="item.reason"
              class="shrink-0 rounded bg-amber-500/15 px-1 text-[9px] text-amber-700 dark:text-amber-300"
            >
              {{ item.reason }}
            </span>
          </div>
          <div v-if="item.disabledAt" class="text-[10px] text-muted-foreground">{{ item.disabledAt }}</div>
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          :disabled="restoringKey === item.key"
          @click="emit('handleDisabledKeyRestore', item.key)"
        >
          <Loader2 v-if="restoringKey === item.key" class="h-3 w-3 animate-spin" />
          <RotateCcw v-else class="h-3 w-3" />
          {{ tf('console.form.restoreKey', 'Restore') }}
        </Button>
      </div>
    </div>

    <!-- Historical Keys Info -->
    <div v-if="historicalApiKeys.length" class="text-xs text-muted-foreground">
      {{ historicalApiKeys.length }} {{ tf('console.form.historicalKeys', 'historical keys recorded') }}
    </div>
  </section>
</template>
