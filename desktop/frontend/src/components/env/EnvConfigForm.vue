<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RefreshCw, Save } from 'lucide-vue-next'
import { useEnvFile } from '@/composables/useEnvFile'
import { detectEnvNewline, getEnvFieldValue, parseEnvFile, serializeEnvFile, type EnvEntry } from '@/lib/env-file'

type FieldType = 'text' | 'password' | 'number' | 'select'

type EnvField = {
  key: string
  label: string
  type: FieldType
  defaultValue: string
  description?: string
  placeholder?: string
  options?: Array<{ label: string; value: string }>
  min?: number
  max?: number
  step?: number | string
  required?: boolean
  disallow?: string[]
}

type EnvGroup = {
  title: string
  description: string
  fields: EnvField[]
}

const booleanOptions = [
  { label: 'true', value: 'true' },
  { label: 'false', value: 'false' },
]

const envGroups: EnvGroup[] = [
  {
    title: '服务器配置',
    description: 'Desktop 会在启动时注入部分运行参数；这里仍完整覆盖 .env.example。',
    fields: [
      { key: 'PORT', label: '服务端口', type: 'number', defaultValue: '3688', min: 1, max: 65535, description: 'Desktop 启动器可能会覆盖为当前分配端口。' },
      { key: 'ENV', label: '运行环境', type: 'select', defaultValue: 'production', options: [{ label: 'production', value: 'production' }, { label: 'development', value: 'development' }], description: 'production 为推荐值。' },
    ],
  },
  {
    title: 'Web UI 配置',
    description: '控制管理界面是否启用以及默认语言。',
    fields: [
      { key: 'ENABLE_WEB_UI', label: '启用 Web UI', type: 'select', defaultValue: 'true', options: booleanOptions, description: 'Desktop 模式通常会强制启用。' },
      { key: 'APP_UI_LANGUAGE', label: '默认语言', type: 'select', defaultValue: 'en', options: [{ label: 'English', value: 'en' }, { label: 'Bahasa Indonesia', value: 'id' }, { label: '简体中文', value: 'zh-CN' }] },
    ],
  },
  {
    title: '访问控制',
    description: '代理入口与管理入口的访问密钥。',
    fields: [
      { key: 'PROXY_ACCESS_KEY', label: '代理访问密钥', type: 'password', defaultValue: '', required: true, disallow: ['your-proxy-access-key'], placeholder: '请输入强随机密钥' },
      { key: 'ADMIN_ACCESS_KEY', label: '管理 API 独立密钥', type: 'password', defaultValue: '', placeholder: '留空则回退到 PROXY_ACCESS_KEY', description: '用于管理界面和 /api/* 端点。' },
    ],
  },
  {
    title: '日志配置',
    description: '控制请求/响应日志、SSE 调试和模型字段改写。',
    fields: [
      { key: 'LOG_LEVEL', label: '日志级别', type: 'select', defaultValue: 'info', options: [{ label: 'error', value: 'error' }, { label: 'warn', value: 'warn' }, { label: 'info', value: 'info' }, { label: 'debug', value: 'debug' }] },
      { key: 'ENABLE_REQUEST_LOGS', label: '启用请求日志', type: 'select', defaultValue: 'false', options: booleanOptions },
      { key: 'ENABLE_RESPONSE_LOGS', label: '启用响应日志', type: 'select', defaultValue: 'false', options: booleanOptions, description: '响应日志可能增加敏感内容暴露风险。' },
      { key: 'QUIET_POLLING_LOGS', label: '静默轮询日志', type: 'select', defaultValue: 'true', options: booleanOptions },
      { key: 'RAW_LOG_OUTPUT', label: '原始日志输出', type: 'select', defaultValue: 'false', options: booleanOptions },
      { key: 'SSE_DEBUG_LEVEL', label: 'SSE 调试级别', type: 'select', defaultValue: 'off', options: [{ label: 'off', value: 'off' }, { label: 'summary', value: 'summary' }, { label: 'full', value: 'full' }] },
      { key: 'REWRITE_RESPONSE_MODEL', label: '改写响应 model', type: 'select', defaultValue: 'false', options: booleanOptions },
    ],
  },
  {
    title: '性能配置',
    description: '请求链路超时和请求体大小限制。',
    fields: [
      { key: 'REQUEST_TIMEOUT', label: '请求超时（毫秒）', type: 'number', defaultValue: '300000', min: 1 },
      { key: 'SERVER_READ_TIMEOUT', label: '服务端读取超时（毫秒）', type: 'number', defaultValue: '60000', min: 10000, max: 300000 },
      { key: 'MAX_REQUEST_BODY_SIZE_MB', label: '请求体最大大小（MB）', type: 'number', defaultValue: '50', min: 1 },
      { key: 'RESPONSE_HEADER_TIMEOUT', label: '响应头超时（秒）', type: 'number', defaultValue: '60', min: 30, max: 120 },
    ],
  },
  {
    title: 'CORS 配置',
    description: '跨域访问控制。',
    fields: [
      { key: 'ENABLE_CORS', label: '启用 CORS', type: 'select', defaultValue: 'false', options: booleanOptions },
      { key: 'CORS_ORIGIN', label: '允许的 Origin', type: 'text', defaultValue: '*', placeholder: '*' },
    ],
  },
  {
    title: '熔断指标配置',
    description: '控制调度指标窗口与失败率阈值。',
    fields: [
      { key: 'METRICS_WINDOW_SIZE', label: '滑动窗口大小', type: 'number', defaultValue: '10', min: 3 },
      { key: 'METRICS_FAILURE_THRESHOLD', label: '失败率阈值', type: 'number', defaultValue: '0.5', min: 0, max: 1, step: '0.01' },
    ],
  },
  {
    title: '指标持久化配置',
    description: '控制 SQLite 指标持久化与数据保留。',
    fields: [
      { key: 'METRICS_PERSISTENCE_ENABLED', label: '启用指标持久化', type: 'select', defaultValue: 'true', options: booleanOptions },
      { key: 'METRICS_RETENTION_DAYS', label: '指标保留天数', type: 'number', defaultValue: '30', min: 3, max: 90 },
    ],
  },
]

const supportedKeys = envGroups.flatMap((group) => group.fields.map((field) => field.key))
const allFields = envGroups.flatMap((group) => group.fields)
const fieldMap = new Map(allFields.map((field) => [field.key, field]))

const { envFile, envLoading, envSaving, envMessage, envError, loadEnvFile, saveEnvFile } = useEnvFile()

const entries = ref<EnvEntry[]>([])
const newline = ref('\n')
const showSecret = reactive<Record<string, boolean>>({})
const form = reactive<Record<string, string>>(Object.fromEntries(allFields.map((field) => [field.key, field.defaultValue])))

const fieldErrors = computed(() => {
  const errors: Record<string, string> = {}
  for (const field of allFields) {
    const value = String(form[field.key] ?? '').trim()
    if (field.required && !value) {
      errors[field.key] = `${field.label}不能为空`
      continue
    }
    if (field.disallow?.includes(value)) {
      errors[field.key] = `${field.label}不能使用示例占位值`
      continue
    }
    if (field.type === 'number') {
      const numberValue = Number(value)
      if (!Number.isFinite(numberValue)) {
        errors[field.key] = `${field.label}必须是数字`
        continue
      }
      if (field.step !== '0.01' && !Number.isInteger(numberValue)) {
        errors[field.key] = `${field.label}必须是整数`
        continue
      }
      if (field.min !== undefined && numberValue < field.min) {
        errors[field.key] = `${field.label}不能小于 ${field.min}`
        continue
      }
      if (field.max !== undefined && numberValue > field.max) {
        errors[field.key] = `${field.label}不能大于 ${field.max}`
      }
    }
  }
  return errors
})

const validationError = computed(() => Object.values(fieldErrors.value)[0] || '')

const load = async () => {
  await loadEnvFile()
  const content = envFile.value.content || ''
  newline.value = detectEnvNewline(content)
  entries.value = parseEnvFile(content)
  for (const field of allFields) {
    form[field.key] = getEnvFieldValue(entries.value, field.key, field.defaultValue)
  }
}

const save = async () => {
  if (validationError.value) return
  const content = serializeEnvFile(entries.value, form, supportedKeys, newline.value)
  await saveEnvFile(content)
  entries.value = parseEnvFile(envFile.value.content || content)
}

const inputType = (field: EnvField) => {
  if (field.type === 'password') return showSecret[field.key] ? 'text' : 'password'
  if (field.type === 'number') return 'number'
  return 'text'
}

onMounted(load)
</script>

<template>
  <Card>
    <CardHeader class="pb-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <CardTitle class="text-base">环境配置</CardTitle>
          <p class="text-xs text-muted-foreground mt-1 break-all">
            {{ envFile.path || '检测中' }}
          </p>
        </div>
        <div class="flex gap-2">
          <Button size="sm" variant="ghost" :disabled="envLoading || envSaving" @click="load">
            <RefreshCw class="w-4 h-4 mr-1.5" />
            刷新
          </Button>
          <Button size="sm" :disabled="envLoading || envSaving || Boolean(validationError)" @click="save">
            <Save class="w-4 h-4 mr-1.5" />
            保存
          </Button>
        </div>
      </div>
    </CardHeader>

    <CardContent class="space-y-6">
      <section v-for="group in envGroups" :key="group.title" class="space-y-3">
        <div>
          <h3 class="text-sm font-semibold">{{ group.title }}</h3>
          <p class="text-xs text-muted-foreground mt-1">{{ group.description }}</p>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div v-for="field in group.fields" :key="field.key" class="space-y-1.5">
            <Label class="text-xs text-muted-foreground">{{ field.key }}</Label>

            <div v-if="field.type === 'password'" class="flex gap-2">
              <Input v-model="form[field.key]" :type="inputType(field)" :placeholder="field.placeholder" />
              <Button type="button" variant="secondary" size="sm" @click="showSecret[field.key] = !showSecret[field.key]">
                {{ showSecret[field.key] ? '隐藏' : '显示' }}
              </Button>
            </div>

            <select
              v-else-if="field.type === 'select'"
              v-model="form[field.key]"
              class="w-full h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              <option v-for="option in field.options" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>

            <Input
              v-else
              v-model="form[field.key]"
              :type="inputType(field)"
              :min="field.min"
              :max="field.max"
              :step="field.step"
              :placeholder="field.placeholder"
            />

            <p v-if="field.description" class="text-xs text-muted-foreground">{{ field.description }}</p>
            <p v-if="fieldErrors[field.key]" class="text-xs text-destructive-foreground">{{ fieldErrors[field.key] }}</p>
          </div>
        </div>
      </section>

      <p v-if="validationError" class="text-xs text-destructive-foreground">{{ validationError }}</p>
      <p v-if="envMessage" class="text-xs text-accent-foreground">{{ envMessage }}</p>
      <p v-if="envError" class="text-xs text-destructive-foreground">{{ envError }}</p>
    </CardContent>
  </Card>
</template>
