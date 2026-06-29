<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Clock, Globe, KeyRound, Loader2, ShieldCheck, Stethoscope, Zap } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

interface FormData {
  fastMode: boolean
  modelCapabilitiesText: string
  historicalImageTurnLimit: number
  insecureSkipVerify: boolean
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
  stripImageGenerationTool: boolean
  reasoningParamStyle: string
  textVerbosity: string
  serviceType: string
  authHeader: 'auto' | 'bearer' | 'x-api-key' | ''
  proxyUrl: string
  routePrefix: string
  rateLimitRpm: string | number
  rateLimitWindowMinutes: string | number
  rateLimitMaxConcurrent: string | number
  rateLimitAutoFromHeaders: boolean
  requestTimeoutMs: string | number
  responseHeaderTimeoutMs: string | number
}

defineProps<{
  form: FormData
  channelType: string
  supportsOpenAIAdvancedOptions: boolean
  supportsChatRoleNormalization: boolean
  reasoningParamStyleOptions: Array<{ label: string; value: string }>
  textVerbosityOptions: Array<{ label: string; value: string }>
  diagnosing?: boolean
}>()

const emit = defineEmits<{
  'update:form': [value: Partial<FormData>]
  'diagnose': []
}>()

const { t } = useLanguage()
const TEXT_VERBOSITY_DEFAULT_VALUE = 'default'
const authHeaderOptions = [
  { label: t('channelEditor.advanced.authHeader.auto'), value: 'auto' },
  { label: 'Authorization: Bearer', value: 'bearer' },
  { label: 'x-api-key', value: 'x-api-key' },
]

function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
  emit('update:form', { [key]: value } as Partial<FormData>)
}

function updateTextVerbosity(value: string) {
  updateField(
    'textVerbosity',
    (value === TEXT_VERBOSITY_DEFAULT_VALUE ? '' : value) as FormData['textVerbosity'],
  )
}
</script>

<template>
  <section class="space-y-6 rounded-xl border border-border/60 bg-gradient-to-br from-card/60 to-card/40 p-5 shadow-sm backdrop-blur-sm">
    <h4 class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2.5">
      <span class="flex h-5 w-5 items-center justify-center rounded-md bg-primary/10 text-primary">
        <ShieldCheck class="h-3 w-3" />
      </span>
      {{ t('channelEditor.nav.advanced') }}
    </h4>

    <div class="grid gap-3">
      <div class="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-gradient-to-r from-background/60 to-background/40 p-4 shadow-sm backdrop-blur-sm transition-all hover:shadow-md">
        <div class="min-w-0 space-y-0.5">
          <Label class="text-xs font-medium">{{ t('addChannel.skipTlsLabel') }}</Label>
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('addChannel.skipTlsHint') }}</p>
        </div>
        <Switch :model-value="form.insecureSkipVerify" @update:model-value="updateField('insecureSkipVerify', $event)" />
      </div>
      <div class="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-gradient-to-r from-background/60 to-background/40 p-4 shadow-sm backdrop-blur-sm transition-all hover:shadow-md">
        <div class="min-w-0 space-y-0.5">
          <Label class="text-xs font-medium">{{ t('addChannel.lowQualityLabel') }}</Label>
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('addChannel.lowQualityHint') }}</p>
        </div>
        <Switch :model-value="form.lowQuality" @update:model-value="updateField('lowQuality', $event)" />
      </div>
      <div class="grid gap-3 rounded-lg border border-border/60 bg-gradient-to-r from-background/60 to-background/40 p-4 shadow-sm backdrop-blur-sm md:grid-cols-[minmax(0,1fr)_160px]">
        <div class="min-w-0 space-y-0.5">
          <Label class="flex items-center gap-1.5 text-xs font-medium">
            <KeyRound class="h-3 w-3 text-primary" />
            {{ t('channelEditor.advanced.authHeader.label') }}
          </Label>
          <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.advanced.authHeader.hint') }}</p>
        </div>
        <Select
          :model-value="form.authHeader || 'auto'"
          @update:model-value="(val) => updateField('authHeader', String(val) as FormData['authHeader'])"
        >
          <SelectTrigger class="h-8 w-full text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="opt in authHeaderOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <div class="space-y-2.5">
      <!-- Runtime 运行期策略 -->
      <div class="p-4 rounded-lg border border-border/60 bg-gradient-to-br from-background/60 to-background/40 shadow-sm backdrop-blur-sm space-y-2.5">
        <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
          <Zap class="h-3 w-3" />
          {{ t('channelEditor.runtime.title') }}
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.runtime.autoBlacklist.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">
                {{ t('channelEditor.runtime.autoBlacklist.hint') }}
              </p>
            </div>
            <Switch :model-value="form.autoBlacklistBalance" @update:model-value="updateField('autoBlacklistBalance', $event)" />
          </div>
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.runtime.autoLearnRateLimits.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">
                {{ t('channelEditor.runtime.autoLearnRateLimits.hint') }}
              </p>
            </div>
            <Switch :model-value="form.rateLimitAutoFromHeaders" @update:model-value="updateField('rateLimitAutoFromHeaders', $event)" />
          </div>
        </div>
      </div>

      <!-- 协议规范化 -->
      <div class="p-4 rounded-lg border border-border/60 bg-gradient-to-br from-background/60 to-background/40 shadow-sm backdrop-blur-sm space-y-2.5">
        <div class="flex items-center justify-between gap-2 border-b border-border/40 pb-2">
          <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-primary">
            <ShieldCheck class="h-3 w-3" />
            {{ t('channelEditor.compat.title') }}
          </div>
          <Button type="button" variant="secondary" size="sm" class="h-6 gap-1 px-2 text-[10px]" :disabled="diagnosing" @click="$emit('diagnose')">
            <Loader2 v-if="diagnosing" class="h-3 w-3 animate-spin" />
            <Stethoscope v-else class="h-3 w-3" />
            {{ t('channelEditor.compat.diagnose') }}
          </Button>
        </div>
        <div class="compat-option-list space-y-2">
          <div v-if="channelType === 'responses'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.codexNativeTools.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.codexNativeTools.hint') }}</p>
            </div>
            <Switch :model-value="form.codexNativeToolPassthrough" @update:model-value="updateField('codexNativeToolPassthrough', $event)" />
          </div>
          <div v-if="channelType === 'responses'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.codexCompat.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.codexCompat.hint') }}</p>
            </div>
            <Switch :model-value="form.codexToolCompat" @update:model-value="updateField('codexToolCompat', $event)" />
          </div>
          <div v-if="channelType === 'responses' || channelType === 'chat'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.stripImageGen.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.stripImageGen.hint') }}</p>
            </div>
            <Switch :model-value="form.stripImageGenerationTool" @update:model-value="updateField('stripImageGenerationTool', $event)" />
          </div>
          <div v-if="channelType === 'messages'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.normalizeSystem.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.normalizeSystem.hint') }}</p>
            </div>
            <Switch :model-value="form.normalizeSystemRoleToTopLevel" @update:model-value="updateField('normalizeSystemRoleToTopLevel', $event)" />
          </div>
          <div v-if="['messages','responses'].includes(channelType)" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.normalizeUserId.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.normalizeUserId.hint') }}</p>
            </div>
            <Switch :model-value="form.normalizeMetadataUserId" @update:model-value="updateField('normalizeMetadataUserId', $event)" />
          </div>
          <div v-if="channelType === 'messages'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.stripBillingHeader.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.stripBillingHeader.hint') }}</p>
            </div>
            <Switch :model-value="form.stripBillingHeader" @update:model-value="updateField('stripBillingHeader', $event)" />
          </div>
          <div v-if="supportsChatRoleNormalization" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.normalizeRoles.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.normalizeRoles.hint') }}</p>
            </div>
            <Switch :model-value="form.normalizeNonstandardChatRoles" @update:model-value="updateField('normalizeNonstandardChatRoles', $event)" />
          </div>
          <div v-if="supportsOpenAIAdvancedOptions" class="flex items-center justify-between gap-3">
            <div class="min-w-0 flex-1 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.reasoningStyle.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.reasoningStyle.hint') }}</p>
            </div>
            <Select
              :model-value="form.reasoningParamStyle || 'reasoning'"
              @update:model-value="(val) => updateField('reasoningParamStyle', String(val))"
            >
              <SelectTrigger class="h-8 w-[200px] text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in reasoningParamStyleOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div v-if="supportsOpenAIAdvancedOptions" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.fastMode.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.fastMode.hint') }}</p>
            </div>
            <Switch :model-value="form.fastMode" @update:model-value="updateField('fastMode', $event)" />
          </div>
          <div v-if="supportsOpenAIAdvancedOptions" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.textVerbosity.label') }}</Label>
            </div>
            <Select
              :model-value="form.textVerbosity || TEXT_VERBOSITY_DEFAULT_VALUE"
              @update:model-value="(val) => updateTextVerbosity(String(val))"
            >
              <SelectTrigger class="h-8 w-[200px] text-xs">
                <SelectValue :placeholder="t('channelEditor.compat.textVerbosity.placeholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="TEXT_VERBOSITY_DEFAULT_VALUE">
                  {{ t('channelEditor.compat.textVerbosity.placeholder') }}
                </SelectItem>
                <SelectItem v-for="opt in textVerbosityOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div v-if="(channelType === 'gemini' || channelType === 'messages') && form.serviceType === 'gemini'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.injectDummySignature.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.injectDummySignature.hint') }}</p>
            </div>
            <Switch :model-value="form.injectDummyThoughtSignature" @update:model-value="updateField('injectDummyThoughtSignature', $event)" />
          </div>
          <div v-if="form.serviceType === 'gemini' && (channelType === 'gemini' || channelType === 'messages' || channelType === 'chat' || channelType === 'responses')" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.stripThoughtSignature.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.stripThoughtSignature.hint') }}</p>
            </div>
            <Switch :model-value="form.stripThoughtSignature" @update:model-value="updateField('stripThoughtSignature', $event)" />
          </div>
          <div v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.passbackReasoning.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.passbackReasoning.hint') }}</p>
            </div>
            <Switch :model-value="form.passbackReasoningContent" @update:model-value="updateField('passbackReasoningContent', $event)" />
          </div>
          <div v-if="(channelType === 'messages' || channelType === 'chat' || channelType === 'responses') && form.serviceType === 'claude'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.passbackThinking.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.passbackThinking.hint') }}</p>
            </div>
            <Switch :model-value="form.passbackThinkingBlocks" @update:model-value="updateField('passbackThinkingBlocks', $event)" />
          </div>
          <div v-if="channelType === 'messages' && form.serviceType === 'claude'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.stripEmptyBlocks.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.stripEmptyBlocks.hint') }}</p>
            </div>
            <Switch :model-value="form.stripEmptyTextBlocks" @update:model-value="updateField('stripEmptyTextBlocks', $event)" />
          </div>
          <div v-if="channelType !== 'images'" class="flex items-center justify-between gap-3">
            <div class="min-w-0 flex-1 space-y-0.5">
              <Label class="text-xs font-medium">{{ t('channelEditor.compat.historicalImageLimit.label') }}</Label>
              <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.compat.historicalImageLimit.hint') }}</p>
            </div>
            <Input
              :model-value="form.historicalImageTurnLimit"
              type="number"
              min="0"
              max="10"
              class="historical-image-limit-field h-8 w-[120px] text-xs"
              @update:model-value="updateField('historicalImageTurnLimit', Number($event))"
            />
          </div>
        </div>
      </div>

      <!-- Transport 代理路由网络 -->
      <div class="p-4 rounded-lg border border-border/60 bg-gradient-to-br from-background/60 to-background/40 shadow-sm backdrop-blur-sm space-y-3">
        <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
          <Globe class="h-3 w-3" />
          {{ t('channelEditor.transport.title') }}
        </div>
        <div class="grid gap-2">
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.proxyUrl.label') }}</Label>
            <Input
              :model-value="form.proxyUrl"
              class="h-8 w-full font-mono text-xs"
              :placeholder="t('channelEditor.transport.proxyUrl.placeholder')"
              @update:model-value="(val) => updateField('proxyUrl', val as string)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.proxyUrl.hint') }}</p>
          </div>
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.requestTimeout.label') }}</Label>
            <Input
              :model-value="form.requestTimeoutMs"
              type="number"
              min="1000"
              max="300000"
              step="1000"
              class="h-8 w-full text-xs"
              :placeholder="t('channelEditor.transport.requestTimeout.placeholder')"
              @update:model-value="(val) => updateField('requestTimeoutMs', val)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.requestTimeout.hint') }}</p>
          </div>
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.responseHeaderTimeout.label') }}</Label>
            <Input
              :model-value="form.responseHeaderTimeoutMs"
              type="number"
              min="1000"
              max="300000"
              step="1000"
              class="h-8 w-full text-xs"
              :placeholder="t('channelEditor.transport.responseHeaderTimeout.placeholder')"
              @update:model-value="(val) => updateField('responseHeaderTimeoutMs', val)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.responseHeaderTimeout.hint') }}</p>
          </div>
          <div class="space-y-1">
            <Label class="text-[9px] font-bold text-muted-foreground">{{ t('channelEditor.transport.routePrefix.label') }}</Label>
            <Input
              :model-value="form.routePrefix"
              class="h-8 w-full font-mono text-xs"
              :placeholder="t('channelEditor.transport.routePrefix.placeholder')"
              @update:model-value="(val) => updateField('routePrefix', val as string)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.transport.routePrefix.hint') }}</p>
          </div>
        </div>
      </div>

      <!-- Rate Limit -->
      <div class="p-4 rounded-lg border border-border/60 bg-gradient-to-br from-background/60 to-background/40 shadow-sm backdrop-blur-sm space-y-3">
        <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-primary border-b border-border/40 pb-2">
          <Clock class="h-3 w-3" />
          {{ t('channelEditor.rateLimit.title') }}
        </div>
        <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.section.hint') }}</p>
        <div class="grid gap-3 md:grid-cols-3">
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">{{ t('channelEditor.rateLimit.rpm.label') }}</Label>
            <Input
              :model-value="form.rateLimitRpm"
              type="number"
              class="h-9 text-xs"
              :placeholder="t('channelEditor.rateLimit.rpm.placeholder')"
              @update:model-value="updateField('rateLimitRpm', $event)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.rpm.hint') }}</p>
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">{{ t('channelEditor.rateLimit.window.label') }}</Label>
            <Input
              :model-value="form.rateLimitWindowMinutes"
              type="number"
              class="h-9 text-xs"
              :placeholder="t('channelEditor.rateLimit.window.placeholder')"
              @update:model-value="updateField('rateLimitWindowMinutes', $event)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.window.hint') }}</p>
          </div>
          <div class="space-y-1">
            <Label class="text-[10px] font-medium text-muted-foreground/80">{{ t('channelEditor.rateLimit.maxConcurrent.label') }}</Label>
            <Input
              :model-value="form.rateLimitMaxConcurrent"
              type="number"
              class="h-9 text-xs"
              :placeholder="t('channelEditor.rateLimit.maxConcurrent.placeholder')"
              @update:model-value="updateField('rateLimitMaxConcurrent', $event)"
            />
            <p class="text-[10px] leading-4 text-muted-foreground">{{ t('channelEditor.rateLimit.maxConcurrent.hint') }}</p>
          </div>
        </div>
      </div>

    </div>
  </section>
</template>

<style scoped>
.compat-option-list > div {
  align-items: flex-start;
  min-width: 0;
}

.compat-option-list > div > div:first-child {
  flex: 1 1 auto;
  min-width: 0;
  overflow-wrap: anywhere;
}

.compat-option-list :deep([data-slot='switch']) {
  flex: 0 0 2rem;
  margin-top: 0.125rem;
}

.compat-option-list :deep([data-slot='select-trigger']) {
  flex: 0 0 200px;
  max-width: 200px;
  min-width: 160px;
}

.historical-image-limit-field {
  flex: 0 0 120px;
  max-width: 120px;
  min-width: 120px;
}

@media (max-width: 640px) {
  .compat-option-list > div {
    flex-direction: column;
    align-items: stretch;
  }

  .compat-option-list :deep([data-slot='switch']) {
    align-self: flex-end;
  }

  .compat-option-list :deep([data-slot='select-trigger']),
  .historical-image-limit-field {
    flex-basis: auto;
    width: 100%;
    max-width: 100%;
  }
}
</style>
