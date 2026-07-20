<template>
  <div class="compshare-plan-section mb-5">
    <v-divider class="mb-4" />
    <div class="d-flex align-center justify-space-between ga-3 flex-wrap mb-2">
      <div class="d-flex align-center ga-2">
        <v-icon color="primary" size="small">mdi-gauge</v-icon>
        <span class="text-body-2 font-weight-medium">{{ t('compshareConsoleCookie.title') }}</span>
      </div>
      <v-btn
        href="https://console.compshare.cn/light-gpu/model-manage"
        target="_blank"
        rel="noopener noreferrer"
        size="small"
        variant="text"
        color="primary"
      >
        <v-icon start size="small">mdi-open-in-new</v-icon>
        {{ t('compshareConsoleCookie.openConsole') }}
      </v-btn>
    </div>
    <div class="text-caption text-medium-emphasis mb-3">{{ t('compshareConsoleCookie.hint') }}</div>

    <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-3" />
    <v-alert v-if="loadError" color="error" variant="tonal" density="compact" class="mb-3">
      {{ loadError }}
    </v-alert>

    <div v-for="credential in credentials" :key="credential.credentialUid" class="compshare-credential py-3">
      <div class="d-flex align-center justify-space-between ga-3 flex-wrap mb-3">
        <code class="text-caption">{{ credential.keyMask }}</code>
        <div class="d-flex align-center ga-2 flex-wrap">
          <v-chip
            v-if="credential.compsharePlan"
            :color="credential.compsharePlan.status === 1 ? 'success' : 'error'"
            size="x-small"
            variant="tonal"
          >
            {{ planDisplayName(credential.compsharePlan) }}
          </v-chip>
          <v-chip :color="credential.hasCompshareConsoleCookie ? 'info' : 'warning'" size="x-small" variant="tonal">
            {{
              credential.hasCompshareConsoleCookie
                ? t('compshareConsoleCookie.configured')
                : t('compshareConsoleCookie.notConfigured')
            }}
          </v-chip>
        </div>
      </div>

      <template v-if="credential.compsharePlan">
        <div class="compshare-usage-grid mb-3">
          <div v-for="item in usageItems(credential.compsharePlan)" :key="item.label" class="compshare-usage-item">
            <div class="text-caption text-medium-emphasis">{{ t(item.label) }}</div>
            <div class="text-body-2 font-weight-medium">{{ formatRemaining(item.window) }}</div>
            <v-progress-linear
              :model-value="usagePercent(item.window)"
              :color="usageColor(item.window)"
              height="4"
              rounded
              class="my-2"
            />
            <div class="text-caption text-disabled">
              {{ t('compshareConsoleCookie.nextReset') }} {{ formatEpoch(item.window.nextResetAt) }}
            </div>
          </div>
        </div>
        <div class="compshare-plan-meta mb-2">
          <div>
            <div class="text-caption text-medium-emphasis">{{ t('compshareConsoleCookie.concurrency') }}</div>
            <div class="text-body-2 font-weight-medium">{{ credential.compsharePlan.concurrencyLimit }}</div>
          </div>
          <div>
            <div class="text-caption text-medium-emphasis">{{ t('compshareConsoleCookie.accountType') }}</div>
            <div class="text-body-2 font-weight-medium">
              {{
                credential.compsharePlan.isTeam
                  ? t('compshareConsoleCookie.team')
                  : t('compshareConsoleCookie.personal')
              }}
            </div>
          </div>
          <div>
            <div class="text-caption text-medium-emphasis">{{ t('compshareConsoleCookie.expiresAt') }}</div>
            <div class="text-body-2 font-weight-medium">{{ formatEpoch(credential.compsharePlan.expireAt) }}</div>
          </div>
        </div>
        <div class="text-caption text-disabled mb-3">
          {{ t('compshareConsoleCookie.validatedAt') }} {{ formatDateTime(credential.compsharePlan.validatedAt) }}
        </div>
      </template>

      <div v-if="forms[credential.credentialUid]" class="d-flex flex-column ga-2">
        <v-text-field
          v-model="forms[credential.credentialUid].cookie"
          :label="t('compshareConsoleCookie.cookie')"
          :placeholder="t('compshareConsoleCookie.cookiePlaceholder')"
          type="password"
          variant="outlined"
          density="compact"
          autocomplete="new-password"
          hide-details
        />
        <v-alert v-if="forms[credential.credentialUid].error" color="error" variant="tonal" density="compact">
          {{ forms[credential.credentialUid].error }}
        </v-alert>
        <div class="d-flex align-center justify-end ga-2 flex-wrap">
          <v-tooltip
            v-if="credential.hasCompshareConsoleCookie"
            :text="t('compshareConsoleCookie.refresh')"
            location="top"
            :open-delay="150"
            content-class="key-tooltip"
          >
            <template #activator="{ props: tooltipProps }">
              <v-btn
                v-bind="tooltipProps"
                icon
                size="small"
                variant="text"
                color="secondary"
                :loading="forms[credential.credentialUid].refreshing"
                :aria-label="t('compshareConsoleCookie.refresh')"
                @click="refreshCookie(credential)"
              >
                <v-icon size="small">mdi-refresh</v-icon>
              </v-btn>
            </template>
          </v-tooltip>
          <v-tooltip
            v-if="credential.hasCompshareConsoleCookie"
            :text="t('compshareConsoleCookie.clear')"
            location="top"
            :open-delay="150"
            content-class="key-tooltip"
          >
            <template #activator="{ props: tooltipProps }">
              <v-btn
                v-bind="tooltipProps"
                icon
                size="small"
                variant="text"
                color="error"
                :loading="forms[credential.credentialUid].clearing"
                :aria-label="t('compshareConsoleCookie.clear')"
                @click="clearCookie(credential)"
              >
                <v-icon size="small">mdi-link-off</v-icon>
              </v-btn>
            </template>
          </v-tooltip>
          <v-btn
            size="small"
            variant="tonal"
            color="primary"
            :loading="forms[credential.credentialUid].saving"
            :disabled="!forms[credential.credentialUid].cookie.trim()"
            @click="saveCookie(credential)"
          >
            <v-icon start size="small">mdi-check-decagram-outline</v-icon>
            {{ t('compshareConsoleCookie.verifyAndSave') }}
          </v-btn>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from '../../i18n'
import { ApiService } from '../../services/api'
import type {
  CompsharePlanSnapshot,
  CompsharePlanUsageWindow,
  ManagedAccountCredential
} from '../../services/api-types'

interface Props {
  accountUid: string
}

interface CredentialForm {
  cookie: string
  saving: boolean
  refreshing: boolean
  clearing: boolean
  error: string
}

const props = defineProps<Props>()
const { t } = useI18n()
const apiService = new ApiService()
const credentials = ref<ManagedAccountCredential[]>([])
const forms = ref<Record<string, CredentialForm>>({})
const loading = ref(false)
const loadError = ref('')
let loadRequest = 0

const loadCredentials = async () => {
  const request = ++loadRequest
  credentials.value = []
  loadError.value = ''
  if (!props.accountUid) return
  loading.value = true
  try {
    const response = await apiService.getManagedAccounts()
    if (request !== loadRequest) return
    const account = response.accounts.find(item => item.accountUid === props.accountUid)
    if (!account) {
      loadError.value = t('compshareConsoleCookie.accountNotFound')
      return
    }
    credentials.value = account.credentials
    const nextForms: Record<string, CredentialForm> = {}
    for (const credential of account.credentials) {
      nextForms[credential.credentialUid] = forms.value[credential.credentialUid] ?? {
        cookie: '',
        saving: false,
        refreshing: false,
        clearing: false,
        error: ''
      }
    }
    forms.value = nextForms
  } catch (error) {
    if (request === loadRequest) loadError.value = errorMessage(error)
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

watch(
  () => props.accountUid,
  () => {
    void loadCredentials()
  },
  { immediate: true }
)

const saveCookie = async (credential: ManagedAccountCredential) => {
  const form = forms.value[credential.credentialUid]
  if (!form?.cookie.trim()) return
  form.saving = true
  form.error = ''
  try {
    const response = await apiService.setCompshareConsoleCookie(
      props.accountUid,
      credential.credentialUid,
      form.cookie.trim()
    )
    credential.hasCompshareConsoleCookie = true
    credential.compsharePlan = response.plan
    form.cookie = ''
  } catch (error) {
    form.error = errorMessage(error)
  } finally {
    form.saving = false
  }
}

const refreshCookie = async (credential: ManagedAccountCredential) => {
  const form = forms.value[credential.credentialUid]
  if (!form) return
  form.refreshing = true
  form.error = ''
  try {
    const response = await apiService.refreshCompshareConsoleCookie(props.accountUid, credential.credentialUid)
    credential.compsharePlan = response.plan
  } catch (error) {
    form.error = errorMessage(error)
  } finally {
    form.refreshing = false
  }
}

const clearCookie = async (credential: ManagedAccountCredential) => {
  if (!window.confirm(t('compshareConsoleCookie.clearConfirm'))) return
  const form = forms.value[credential.credentialUid]
  if (!form) return
  form.clearing = true
  form.error = ''
  try {
    await apiService.clearCompshareConsoleCookie(props.accountUid, credential.credentialUid)
    credential.hasCompshareConsoleCookie = false
    credential.compsharePlan = undefined
  } catch (error) {
    form.error = errorMessage(error)
  } finally {
    form.clearing = false
  }
}

const usageItems = (plan: CompsharePlanSnapshot) => [
  { label: 'compshareConsoleCookie.fiveHourRemaining', window: plan.fiveHourUsage },
  { label: 'compshareConsoleCookie.weeklyRemaining', window: plan.weeklyUsage },
  { label: 'compshareConsoleCookie.monthlyRemaining', window: plan.monthlyUsage }
]

const numberFormat = new Intl.NumberFormat()
const dateTimeFormat = new Intl.DateTimeFormat(undefined, {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit'
})

const formatRemaining = (window: CompsharePlanUsageWindow) => {
  const remaining = Math.max(0, window.limit - window.used)
  return `${numberFormat.format(remaining)} / ${numberFormat.format(window.limit)}`
}

const usagePercent = (window: CompsharePlanUsageWindow) => {
  if (window.limit <= 0) return 0
  return Math.max(0, Math.min(100, (window.used / window.limit) * 100))
}

const usageColor = (window: CompsharePlanUsageWindow) => {
  const percent = usagePercent(window)
  if (percent >= 90) return 'error'
  if (percent >= 70) return 'warning'
  return 'success'
}

const formatEpoch = (value?: number) => (value && value > 0 ? dateTimeFormat.format(new Date(value * 1000)) : '-')

const formatDateTime = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : dateTimeFormat.format(date)
}

const planDisplayName = (plan: CompsharePlanSnapshot) => plan.displayName || plan.planName || plan.planCode
const errorMessage = (error: unknown) => (error instanceof Error ? error.message : String(error))
</script>

<style scoped>
.compshare-credential + .compshare-credential {
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.compshare-usage-grid,
.compshare-plan-meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.compshare-usage-item {
  min-width: 0;
}

@media (max-width: 700px) {
  .compshare-usage-grid,
  .compshare-plan-meta {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
