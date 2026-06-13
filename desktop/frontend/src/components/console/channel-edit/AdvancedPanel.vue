<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useLanguage } from '@/composables/useLanguage'

interface FormData {
  fastMode: boolean
  textVerbosity: 'low' | 'medium' | 'high' | ''
  supportedModelsText: string
  noVisionModelsText: string
  visionFallbackModel: string
  noVision: boolean
  historicalImageTurnLimit: number
  passbackReasoningContent: boolean
  passbackThinkingBlocks: boolean
  lowQuality: boolean
  injectDummyThoughtSignature: boolean
  stripThoughtSignature: boolean
  stripEmptyTextBlocks: boolean
  normalizeSystemRoleToTopLevel: boolean
  normalizeMetadataUserId: boolean
  stripBillingHeader: boolean
  normalizeNonstandardChatRoles: boolean
  autoBlacklistBalance: boolean
  codexNativeToolPassthrough: boolean
  codexToolCompat: boolean
  stripCodexClientTools: boolean
  stripImageGenerationTool: boolean
  proxyUrl: string
  routePrefix: string
  rateLimitRpm: string | number
  rateLimitWindowMinutes: string | number
  rateLimitMaxConcurrent: string | number
  rateLimitAutoFromHeaders: boolean
  requestTimeoutMs: string | number
  streamFirstContentTimeoutEnabled: boolean
  streamFirstContentTimeoutMs: number
  streamInactivityTimeoutEnabled: boolean
  streamInactivityTimeoutMs: number
  streamToolCallIdleTimeoutEnabled: boolean
  streamToolCallIdleTimeoutMs: number
}

const props = defineProps<{
  form: FormData
  channelType: string
  textVerbosityOptions: Array<{ label: string; value: string }>
  supportsOpenAIAdvanced: boolean
  DEFAULT_SELECT_VALUE: string
}>()

const emit = defineEmits<{
  'update:form': [value: Partial<FormData>]
}>()

const { tf } = useLanguage()

function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
  emit('update:form', { [key]: value } as Partial<FormData>)
}

function toSelectValue(value: string): string {
  return value === '' ? props.DEFAULT_SELECT_VALUE : value
}

function fromSelectValue(value: string): string {
  return value === props.DEFAULT_SELECT_VALUE ? '' : value
}
</script>

<template>
  <div class="space-y-6">
    <!-- 生成参数 -->
    <section v-if="supportsOpenAIAdvanced || channelType === 'responses' || channelType === 'chat'" class="space-y-3 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
      <h4 class="text-xs font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
        {{ tf('console.form.generationParams', '生成参数') }}
      </h4>

      <div v-if="supportsOpenAIAdvanced" class="grid gap-4 md:grid-cols-2 bg-background/30 p-3 rounded-lg border border-border/40">
        <div class="flex items-center justify-between p-2 rounded-md hover:bg-accent/40 transition-colors">
          <div class="space-y-0.5">
            <Label class="text-xs font-semibold">{{ tf('console.form.fastMode', '快速模式') }}</Label>
            <p class="text-[10px] text-muted-foreground">优先选取低延迟的轻量边缘路由链路</p>
          </div>
          <Switch :model-value="form.fastMode" @update:model-value="updateField('fastMode', $event)" />
        </div>

        <div class="space-y-1 p-1">
          <Label class="text-[10px] font-bold text-muted-foreground uppercase">Text Verbosity Style</Label>
          <Select :model-value="toSelectValue(form.textVerbosity)" @update:model-value="(val) => updateField('textVerbosity', fromSelectValue(val as string) as any)">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="tf('console.form.selectDefault', '默认')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="opt in textVerbosityOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <!-- Vision 控制 -->
      <div v-if="['messages', 'chat'].includes(channelType)" class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          Vision 视觉控制
        </div>
        <div class="space-y-3">
          <div class="flex flex-row-reverse items-center justify-between gap-3">
            <Switch :model-value="form.noVision" @update:model-value="updateField('noVision', $event)" class="shrink-0" />
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ tf('console.form.noVision', '跳过含图请求') }}</Label>
              <p class="text-[10px] text-muted-foreground">
                {{ tf('console.form.noVisionHint', '启用后，包含图片的请求将跳过此渠道并 failover 到下一个渠道') }}
              </p>
            </div>
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-bold text-muted-foreground">
              {{ tf('console.form.historicalImageTurnLimit', '历史图片轮次限制') }}
            </Label>
            <Input
              :model-value="form.historicalImageTurnLimit"
              type="number"
              min="0"
              class="h-8 text-xs"
              placeholder="0"
              @update:model-value="updateField('historicalImageTurnLimit', Number($event))"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">
              {{ tf('console.form.historicalImageTurnLimitHint', '0 = 继承全局；后端会对 >0 的值应用最低 3 约束') }}
            </p>
          </div>
        </div>
      </div>
    </section>

    <!-- 高级扩展选项 -->
    <section class="space-y-6 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
      <h4 class="text-xs font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
        {{ tf('console.form.advancedOptions', '高级扩展选项') }}
      </h4>

      <!-- 协议规范化 -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-2.5">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          Compatibility 协议规范化
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <Label class="text-xs font-medium">{{ tf('console.form.normalizeSystemRole', '规范化 System 角色域') }}</Label>
            <Switch :model-value="form.normalizeSystemRoleToTopLevel" @update:model-value="updateField('normalizeSystemRoleToTopLevel', $event)" />
          </div>
          <div v-if="['messages','responses'].includes(channelType)" class="flex items-center justify-between">
            <Label class="text-xs font-medium">{{ tf('console.form.normalizeUserId', '平铺扁平化用户 ID') }}</Label>
            <Switch :model-value="form.normalizeMetadataUserId" @update:model-value="updateField('normalizeMetadataUserId', $event)" />
          </div>
          <div v-if="channelType === 'messages'" class="flex items-center justify-between">
            <Label class="text-xs font-medium">{{ tf('console.form.stripBillingHeader', '抽离并剔除 CCH 计费尾缀') }}</Label>
            <Switch :model-value="form.stripBillingHeader" @update:model-value="updateField('stripBillingHeader', $event)" />
          </div>
        </div>
      </div>

      <!-- Runtime 运行期策略 -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-2.5">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          Runtime 运行期策略
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <Label class="text-xs font-medium">自动熔断/黑名单余额异常 Key</Label>
            <Switch :model-value="form.autoBlacklistBalance" @update:model-value="updateField('autoBlacklistBalance', $event)" />
          </div>
        </div>
      </div>

      <!-- Transport 代理路由网络 -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary/80 border-b border-border/30 pb-1">
          Transport 代理路由网络
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">代理通道 URL</Label>
            <Input
              :model-value="form.proxyUrl"
              class="h-8 w-full font-mono text-xs"
              placeholder="socks5://..."
              @update:model-value="(val) => updateField('proxyUrl', val as string)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">接口路由前缀</Label>
            <Input
              :model-value="form.routePrefix"
              class="h-8 w-full font-mono text-xs"
              placeholder="kimi"
              @update:model-value="(val) => updateField('routePrefix', val as string)"
            />
          </div>
        </div>
      </div>

      <!-- Rate Limit -->
      <div class="p-4 rounded-xl border border-border/50 bg-background/40 space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          Rate Limit 上游主动限速流量阀
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">每分钟最大请求量 (RPM)</Label>
            <Input
              :model-value="form.rateLimitRpm"
              type="number"
              class="h-9 text-xs"
              placeholder="不设限制"
              @update:model-value="updateField('rateLimitRpm', $event)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">检测窗口滑动时间 (s)</Label>
            <Input
              :model-value="form.rateLimitWindowMinutes"
              type="number"
              class="h-9 text-xs"
              placeholder="60 秒"
              @update:model-value="updateField('rateLimitWindowMinutes', $event)"
            />
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">全双工最大并发数限制</Label>
            <Input
              :model-value="form.rateLimitMaxConcurrent"
              type="number"
              class="h-9 text-xs"
              placeholder="不设限制"
              @update:model-value="updateField('rateLimitMaxConcurrent', $event)"
            />
          </div>
        </div>
        <div class="flex items-center gap-2">
          <Switch :model-value="form.rateLimitAutoFromHeaders" @update:model-value="updateField('rateLimitAutoFromHeaders', $event)" />
          <div class="space-y-0.5">
            <Label class="text-[10px]">自动学习上游限速</Label>
            <p class="text-[10px] leading-4 text-muted-foreground">
              解析 Retry-After / x-ratelimit-* 响应头动态调整 cooldown。
            </p>
          </div>
        </div>
      </div>

      <!-- 流式断流超时控制 -->
      <div class="space-y-3">
        <div class="text-[10px] font-bold uppercase tracking-wider text-primary">流式断流超时控制</div>
        <div class="grid gap-3 md:grid-cols-3">
          <!-- 首字等待 -->
          <div class="border border-border/60 bg-background/60 p-3 rounded-xl space-y-2.5">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <Label class="text-xs font-semibold block">首字等待</Label>
                <span class="text-[9px] text-muted-foreground leading-none">未响应则自动断开</span>
              </div>
              <Switch
                :model-value="form.streamFirstContentTimeoutEnabled"
                @update:model-value="updateField('streamFirstContentTimeoutEnabled', $event)"
              />
            </div>
            <div class="space-y-1" :class="{ 'opacity-50 pointer-events-none': !form.streamFirstContentTimeoutEnabled }">
              <div class="flex items-center justify-between text-[10px] font-mono font-medium text-muted-foreground">
                <span>超时阈值:</span>
                <span class="text-primary font-bold">{{ (form.streamFirstContentTimeoutMs / 1000) }}s</span>
              </div>
              <input
                :value="form.streamFirstContentTimeoutMs"
                type="range"
                min="5000"
                max="300000"
                step="1000"
                class="w-full accent-primary h-1 bg-muted rounded-lg appearance-none cursor-pointer"
                :disabled="!form.streamFirstContentTimeoutEnabled"
                @input="updateField('streamFirstContentTimeoutMs', Number(($event.target as HTMLInputElement).value))"
              />
            </div>
          </div>

          <!-- 首字后断流 -->
          <div class="border border-border/60 bg-background/60 p-3 rounded-xl space-y-2.5">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <Label class="text-xs font-semibold block">首字后断流</Label>
                <span class="text-[9px] text-muted-foreground leading-none">生成中途卡顿超时</span>
              </div>
              <Switch
                :model-value="form.streamInactivityTimeoutEnabled"
                @update:model-value="updateField('streamInactivityTimeoutEnabled', $event)"
              />
            </div>
            <div class="space-y-1" :class="{ 'opacity-50 pointer-events-none': !form.streamInactivityTimeoutEnabled }">
              <div class="flex items-center justify-between text-[10px] font-mono font-medium text-muted-foreground">
                <span>超时阈值:</span>
                <span class="text-primary font-bold">{{ (form.streamInactivityTimeoutMs / 1000) }}s</span>
              </div>
              <input
                :value="form.streamInactivityTimeoutMs"
                type="range"
                min="1000"
                max="180000"
                step="1000"
                class="w-full accent-primary h-1 bg-muted rounded-lg appearance-none cursor-pointer"
                :disabled="!form.streamInactivityTimeoutEnabled"
                @input="updateField('streamInactivityTimeoutMs', Number(($event.target as HTMLInputElement).value))"
              />
            </div>
          </div>

          <!-- 工具调用空闲 -->
          <div class="border border-border/60 bg-background/60 p-3 rounded-xl space-y-2.5">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <Label class="text-xs font-semibold block">工具调用空闲</Label>
                <span class="text-[9px] text-muted-foreground leading-none">FunctionCall 延迟</span>
              </div>
              <Switch
                :model-value="form.streamToolCallIdleTimeoutEnabled"
                @update:model-value="updateField('streamToolCallIdleTimeoutEnabled', $event)"
              />
            </div>
            <div class="space-y-1" :class="{ 'opacity-50 pointer-events-none': !form.streamToolCallIdleTimeoutEnabled }">
              <div class="flex items-center justify-between text-[10px] font-mono font-medium text-muted-foreground">
                <span>超时阈值:</span>
                <span class="text-primary font-bold">{{ (form.streamToolCallIdleTimeoutMs / 1000) }}s</span>
              </div>
              <input
                :value="form.streamToolCallIdleTimeoutMs"
                type="range"
                min="1000"
                max="180000"
                step="1000"
                class="w-full accent-primary h-1 bg-muted rounded-lg appearance-none cursor-pointer"
                :disabled="!form.streamToolCallIdleTimeoutEnabled"
                @input="updateField('streamToolCallIdleTimeoutMs', Number(($event.target as HTMLInputElement).value))"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- 限定可支持模型范围 -->
      <div class="space-y-2">
        <Label class="text-xs font-semibold text-muted-foreground">
          限定可支持模型范围（白名单模式，留空表示不限制）
        </Label>
        <Textarea
          :model-value="form.supportedModelsText"
          placeholder="gpt-4*&#10;claude-3*"
          class="w-full font-mono text-xs min-h-[64px]"
          @update:model-value="(val) => updateField('supportedModelsText', val as string)"
        />
      </div>
    </section>
  </div>
</template>
