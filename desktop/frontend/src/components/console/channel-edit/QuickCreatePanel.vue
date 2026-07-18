<script setup lang="ts">
import { computed, type FunctionalComponent } from 'vue'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { AlertCircle, ArrowDownToLine, Bot, CheckCircle2, Code2, Gem, Info, MessageSquare, Sparkles, Tag } from 'lucide-vue-next'
import { Alert } from '@/components/ui/alert'
import { useLanguage } from '@/composables/useLanguage'
import { useChannelPlacementPreference } from '@/composables/useChannelPlacementPreference'
import { findExistingQuickAddChannel } from '@/utils/quick-add-channel'
import type { Channel } from '@/services/admin-api'

const props = defineProps<{
  quickInput: string
  serviceType: string
  serviceTypeOptions: Array<{ label: string; value: string }>
  detectedBaseUrls: string[]
  detectedApiKeys: string[]
  expectedRequestUrls: Array<{ baseUrl: string; expectedUrl: string }>
  generatedChannelName: string
  existingChannels?: Channel[]
}>()

const emit = defineEmits<{
  (e: 'update:quick-input', value: string): void
  (e: 'update:service-type', value: string): void
  (e: 'quick-paste', text: string): void
}>()

const { t } = useLanguage()
const { newChannelPlacement, setNewChannelPlacement } = useChannelPlacementPreference()

// 上游类型品牌色 + 图标，与 ChannelCard 的服务类型视觉体系保持一致
interface ServiceMeta {
  icon: FunctionalComponent
  iconClass: string
  badgeClass: string
}
const serviceMetaMap: Record<string, ServiceMeta> = {
  openai: {
    icon: Bot,
    iconClass: 'text-blue-500',
    badgeClass: 'border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-300',
  },
  claude: {
    icon: MessageSquare,
    iconClass: 'text-orange-500',
    badgeClass: 'border-orange-500/30 bg-orange-500/10 text-orange-600 dark:text-orange-300',
  },
  gemini: {
    icon: Gem,
    iconClass: 'text-purple-500',
    badgeClass: 'border-purple-500/30 bg-purple-500/10 text-purple-600 dark:text-purple-300',
  },
  responses: {
    icon: Sparkles,
    iconClass: 'text-emerald-500',
    badgeClass: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300',
  },
  copilot: {
    icon: Code2,
    iconClass: 'text-slate-600 dark:text-slate-300',
    badgeClass: 'border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-200',
  },
}
const serviceMeta = computed<ServiceMeta>(() => serviceMetaMap[props.serviceType] || serviceMetaMap.openai)
const isCopilot = computed(() => props.serviceType === 'copilot')

const duplicateChannel = computed(() =>
  props.existingChannels?.length
    ? findExistingQuickAddChannel(props.detectedBaseUrls, props.existingChannels)
    : null
)
</script>

<template>
  <section class="space-y-4 p-6">
    <!-- 输入区域 -->
    <Textarea
      :model-value="quickInput"
      rows="10"
      class="!field-sizing-none min-h-[14rem] font-mono text-xs"
      :placeholder="t('addChannel.quickInputPlaceholder')"
      @update:model-value="(val) => emit('update:quick-input', val as string)"
      @paste="emit('quick-paste', $event.clipboardData?.getData('text/plain') || '')"
    />

    <!-- 检测结果 -->
    <div class="rounded-xl border border-border/60 p-4">
      <div class="space-y-3">
        <!-- Base URLs（独占一行） -->
        <div class="space-y-2 rounded-lg border border-border bg-background/70 p-3">
          <div class="flex items-center gap-2 text-xs font-semibold">
            <CheckCircle2 v-if="detectedBaseUrls.length" class="h-4 w-4 text-emerald-500" />
            <AlertCircle v-else class="h-4 w-4 text-muted-foreground" />
            Base URLs
          </div>
          <template v-if="detectedBaseUrls.length">
            <div v-for="item in expectedRequestUrls" :key="item.baseUrl" class="space-y-0.5">
              <p class="truncate text-[11px] font-medium text-emerald-600">{{ item.baseUrl }}</p>
              <p class="truncate text-[10px] text-muted-foreground">
                {{ t('addChannel.expectedRequest') }}: {{ item.expectedUrl }}
              </p>
            </div>
          </template>
          <p v-else class="text-xs text-muted-foreground">
            {{ t('addChannel.noneDetected') }}
          </p>
        </div>

        <!-- 重复渠道提示 -->
        <Alert v-if="duplicateChannel" variant="default" class="border-amber-500/50 bg-amber-500/10 text-amber-700 dark:text-amber-400">
          <AlertCircle class="mr-2 inline size-4" />
          <p class="inline text-xs">
            {{ t('autopilot.quickAdd.alreadyAdded', { name: duplicateChannel.channel.name }) }}
          </p>
        </Alert>

        <!-- 两栏：左（渠道名称+上游类型）右（API Keys） -->
        <div class="grid items-start gap-3 md:grid-cols-12">
          <!-- 左栏：7/12 -->
          <div class="space-y-3 md:col-span-7">
            <!-- 渠道名称预览 -->
            <div class="flex items-center gap-3 rounded-lg border-0 bg-muted/10 p-3">
              <Tag class="h-4 w-4 shrink-0 text-primary" />
              <div class="min-w-0 flex-1 space-y-0.5">
                <div class="text-[11px] text-muted-foreground">{{ t('addChannel.channelName') }}</div>
                <p class="text-xs font-semibold text-primary break-all">{{ generatedChannelName }}</p>
              </div>
              <Badge variant="secondary" class="shrink-0 text-[9px]">
                {{ t('common.autoGenerated') }}
              </Badge>
            </div>

            <!-- 上游类型选择器 -->
            <div class="flex items-center gap-3 rounded-lg bg-muted/10 p-3" :class="serviceMeta.badgeClass">
              <component :is="serviceMeta.icon" class="h-4 w-4 shrink-0" :class="serviceMeta.iconClass" />
              <div class="shrink-0 text-[11px] text-muted-foreground">
                {{ t('channelEditor.basic.serviceType.label') }}
              </div>
              <div class="flex-1" />
              <Select :model-value="serviceType" @update:model-value="(val) => emit('update:service-type', String(val))">
                <SelectTrigger class="h-7 w-auto border-0 bg-transparent p-0 shadow-none focus:ring-0">
                  <SelectValue :placeholder="t('channelEditor.basic.serviceType.placeholder')" class="text-xs font-semibold" />
                </SelectTrigger>
                <SelectContent class="min-w-[180px] max-w-[220px]">
                  <SelectItem v-for="opt in serviceTypeOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <!-- 右栏：API Keys（5/12） -->
          <div class="space-y-2 md:col-span-5">
            <div class="rounded-lg border border-border bg-background/70 p-3">
              <div class="flex items-center gap-2 text-xs font-semibold mb-1.5">
                <CheckCircle2 v-if="detectedApiKeys.length" class="h-4 w-4 text-emerald-500" />
                <Info v-else-if="isCopilot" class="h-4 w-4 text-sky-500" />
                <AlertCircle v-else class="h-4 w-4 text-muted-foreground" />
                {{ t('addChannel.apiKeys') }}
              </div>
              <p v-if="detectedApiKeys.length" class="text-xs text-emerald-600">
                {{ t('addChannel.detectedKeys', { count: String(detectedApiKeys.length) }) }}
              </p>
              <p v-else-if="isCopilot" class="text-xs text-sky-600 dark:text-sky-300">
                {{ t('copilotOAuth.quickAddKeyHint') }}
              </p>
              <p v-else class="text-xs text-muted-foreground">
                {{ t('addChannel.enterApiKey') }}
              </p>
            </div>

            <div
              class="flex items-center gap-3 rounded-lg border border-border bg-background/70 p-3 cursor-pointer hover:border-primary/40 hover:bg-primary/5 transition-colors"
              @click="setNewChannelPlacement(newChannelPlacement === 'bottom' ? 'top' : 'bottom')"
            >
              <ArrowDownToLine class="h-4 w-4 shrink-0 text-primary" />
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium">{{ t('addChannel.newChannelPlacementLabel') }}</div>
              </div>
              <Switch
                :model-value="newChannelPlacement === 'bottom'"
                readonly
                class="pointer-events-none shrink-0"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
