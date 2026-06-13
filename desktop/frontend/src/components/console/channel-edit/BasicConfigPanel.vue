<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AlertCircle, CheckCircle2, ClipboardPaste } from 'lucide-vue-next'
import { useLanguage } from '@/composables/useLanguage'

interface FormData {
  name: string
  description: string
  serviceType: 'openai' | 'claude' | 'gemini' | 'responses' | ''
  baseUrl: string
  baseUrlsText: string
  website: string
}

interface Errors {
  name?: string
  serviceType?: string
  baseUrl?: string
}

const props = defineProps<{
  form: FormData
  errors: Errors
  isEditMode: boolean
  channelType: string
  serviceTypeOptions: Array<{ label: string; value: string }>
  expectedRequestUrls: any[]
  quickInput?: string
  detectedBaseUrls?: string[]
  detectedApiKeys?: string[]
}>()

const emit = defineEmits<{
  'update:form': [value: Partial<FormData>]
  'update:quickInput': [value: string]
  'quickPaste': [text: string]
}>()

const { tf } = useLanguage()

function updateField<K extends keyof FormData>(key: K, value: FormData[K]) {
  emit('update:form', { [key]: value } as Partial<FormData>)
}
</script>

<template>
  <!-- 创建模式：快速粘贴 -->
  <section v-if="!isEditMode" class="space-y-3 rounded-xl border border-primary/20 bg-primary/5 p-4">
    <div>
      <h4 class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-primary">
        <ClipboardPaste class="h-3.5 w-3.5" />
        {{ tf('addChannel.quickMode', '快速粘贴') }}
      </h4>
      <p class="mt-1 text-xs text-muted-foreground">
        {{ tf('addChannel.quickHint', '粘贴 Base URL、API Key 或完整配置片段，自动识别并填入表单。') }}
      </p>
    </div>
    <Textarea
      :model-value="quickInput"
      rows="10"
      class="!field-sizing-none min-h-[14rem] font-mono text-xs"
      placeholder="https://api.example.com/v1&#10;sk-..."
      @update:model-value="(val) => emit('update:quickInput', val as string)"
      @paste="emit('quickPaste', $event.clipboardData?.getData('text/plain') || '')"
    />
    <div class="grid gap-2 md:grid-cols-2">
      <div class="border border-border bg-background/70 p-2 text-xs rounded-lg">
        <div class="mb-1 flex items-center gap-1.5 font-semibold">
          <CheckCircle2 v-if="detectedBaseUrls?.length" class="h-3.5 w-3.5 text-emerald-500" />
          <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
          Base URLs
        </div>
        <p class="truncate text-muted-foreground">
          {{ detectedBaseUrls?.length ? detectedBaseUrls.join(' · ') : tf('addChannel.noneDetected', '未识别') }}
        </p>
      </div>
      <div class="border border-border bg-background/70 p-2 text-xs rounded-lg">
        <div class="mb-1 flex items-center gap-1.5 font-semibold">
          <CheckCircle2 v-if="detectedApiKeys?.length" class="h-3.5 w-3.5 text-emerald-500" />
          <AlertCircle v-else class="h-3.5 w-3.5 text-muted-foreground" />
          {{ tf('console.form.apiKeys', 'API Keys') }}
        </div>
        <p class="text-muted-foreground">
          {{ detectedApiKeys?.length ? `${detectedApiKeys.length} ${tf('console.keys.active', 'active keys')}` : tf('addChannel.noneDetected', '未识别') }}
        </p>
      </div>
    </div>
  </section>

  <!-- 编辑模式：基础信息 + 连接 -->
  <template v-if="isEditMode">
    <div class="grid gap-4 lg:grid-cols-2">
      <!-- 基础信息 -->
      <section class="space-y-4 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
        <h4 class="text-xs font-bold uppercase tracking-wider text-primary">基础信息</h4>
        <div class="grid grid-cols-[2fr_1fr] gap-3">
          <div class="space-y-1.5">
            <Label class="text-xs font-semibold text-muted-foreground">
              {{ tf('console.form.name', '名称') }} <span class="text-destructive">*</span>
            </Label>
            <Input
              :model-value="form.name"
              class="h-9"
              :class="{ 'border-destructive': errors.name }"
              @update:model-value="(val) => updateField('name', val as string)"
            />
            <p v-if="errors.name" class="text-[10px] text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <Label class="text-xs font-semibold text-muted-foreground">
              {{ tf('console.form.serviceType', '服务类型') }} <span class="text-destructive">*</span>
            </Label>
            <Select :model-value="form.serviceType" @update:model-value="updateField('serviceType', $event as any)">
              <SelectTrigger class="h-9" :class="{ 'border-destructive': errors.serviceType }">
                <SelectValue :placeholder="tf('console.form.selectServiceType', '选择服务类型')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in serviceTypeOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.serviceType" class="text-[10px] text-destructive">{{ errors.serviceType }}</p>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label class="text-xs font-semibold text-muted-foreground">{{ tf('console.form.description', '描述') }}</Label>
          <Textarea
            :model-value="form.description"
            rows="2"
            class="min-h-[74px] resize-none"
            placeholder="可选填，用于备注此节点供应商..."
            @update:model-value="(val) => updateField('description', val as string)"
          />
        </div>
      </section>

      <!-- 连接终点 -->
      <section class="space-y-4 rounded-xl border border-border/60 bg-card/40 p-5 shadow-xs">
        <h4 class="text-xs font-bold uppercase tracking-wider text-primary">连接终点</h4>
        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <Label class="text-xs font-semibold text-muted-foreground">
              Base URL <span class="text-destructive">*</span>
            </Label>
            <span class="text-[10px] text-muted-foreground/80 scale-95 origin-right">多行实现故障轮换</span>
          </div>
          <Textarea
            :model-value="form.baseUrlsText"
            placeholder="https://api.example.com&#10;https://backup.example.com"
            class="w-full rounded-lg border border-input bg-background/50 px-3 py-2 font-mono text-xs shadow-inner outline-none transition-all focus:border-primary focus:ring-2 focus:ring-primary/20 min-h-[74px] leading-relaxed"
            :class="{ 'border-destructive': errors.baseUrl }"
            @update:model-value="(val) => updateField('baseUrlsText', val as string)"
          />
          <div class="flex items-center gap-1.5 text-[10px] text-muted-foreground/70 bg-accent/40 px-2 py-1 rounded-md border border-border/30">
            <span class="inline-block size-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            <span class="font-mono truncate">
              预期终点: {{ expectedRequestUrls[0]?.expectedUrl || 'N/A' }}
            </span>
          </div>
          <p v-if="errors.baseUrl" class="text-[10px] text-destructive">{{ errors.baseUrl }}</p>
        </div>
        <div class="space-y-1.5">
          <Label class="text-xs font-semibold text-muted-foreground">{{ tf('console.form.website', '官方网站') }}</Label>
          <Input
            :model-value="form.website"
            class="h-9"
            placeholder="https://example.com"
            @update:model-value="(val) => updateField('website', val as string)"
          />
        </div>
      </section>
    </div>
  </template>
</template>
