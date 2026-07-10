<template>
  <div class="subscriptions-view">
    <!-- Header -->
    <div class="d-flex align-center justify-space-between mb-4">
      <div class="d-flex align-center">
        <v-icon size="28" class="mr-2" color="primary">mdi-cash-multiple</v-icon>
        <span class="text-h5 font-weight-bold">{{ t('subscription.title') }}</span>
      </div>
      <div class="d-flex ga-2">
        <v-btn
          variant="tonal"
          prepend-icon="mdi-refresh"
          :loading="loading"
          @click="fetchSubscriptions"
        >
          {{ t('app.actions.refresh') }}
        </v-btn>
        <v-btn
          color="primary"
          prepend-icon="mdi-plus"
          @click="openCreateDialog"
        >
          {{ t('subscription.add') }}
        </v-btn>
        <v-btn
          variant="tonal"
          color="primary"
          prepend-icon="mdi-connection"
          @click="showNewApiDialog = true"
        >
          {{ t('subscription.newApi.connect') }}
        </v-btn>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="loading && subscriptions.length === 0" class="text-center py-12">
      <v-progress-circular indeterminate color="primary" size="48" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!loading && subscriptions.length === 0" class="text-center py-12 text-medium-emphasis">
      <v-icon size="64" class="mb-4" color="grey">mdi-cash-multiple</v-icon>
      <div class="text-body-1">{{ t('subscription.empty') }}</div>
    </div>

    <!-- Subscription table -->
    <SubscriptionPlanTable
      v-else
      :subscriptions="filteredSubscriptions"
      @edit="openEditDialog"
      @delete="handleDelete"
      @refresh="handleRefresh"
    />

    <!-- Create/Edit dialog -->
    <v-dialog v-model="showDialog" max-width="600" persistent>
      <v-card class="pa-4">
        <v-card-title class="text-h5 mb-2">
          {{ editingSubscription ? t('subscription.edit') : t('subscription.add') }}
        </v-card-title>

        <v-card-text>
          <v-form @submit.prevent="handleSubmit">
            <v-text-field
              v-model="form.subscriptionUid"
              :label="t('subscription.field.uid')"
              variant="outlined"
              density="compact"
              class="mb-2"
              :disabled="!!editingSubscription"
              required
            />
            <v-text-field
              v-model="form.displayName"
              :label="t('subscription.field.name')"
              variant="outlined"
              density="compact"
              class="mb-2"
              required
            />
            <v-text-field
              v-model="form.provider"
              :label="t('subscription.field.provider')"
              variant="outlined"
              density="compact"
              class="mb-2"
            />
            <v-select
              v-model="form.originType"
              :label="t('subscription.field.originType')"
              :items="originTypeOptions"
              variant="outlined"
              density="compact"
              class="mb-2"
            />
            <!-- 来源等级：由来源类型系统推导，只读展示 -->
            <v-text-field
              :model-value="derivedOriginTierLabel"
              :label="t('subscription.field.originTier')"
              variant="outlined"
              density="compact"
              class="mb-2"
              readonly
              :hint="t('subscription.field.originTierHint')"
              persistent-hint
            />
            <v-select
              v-model="form.billingMode"
              :label="t('subscription.field.billingMode')"
              :items="billingModeOptions"
              variant="outlined"
              density="compact"
              class="mb-2"
            />
            <v-text-field
              v-model="form.currency"
              :label="t('subscription.field.currency')"
              variant="outlined"
              density="compact"
              class="mb-2"
              placeholder="CNY / USD"
            />
            <v-text-field
              v-model.number="form.balance"
              :label="t('subscription.field.balance')"
              variant="outlined"
              density="compact"
              type="number"
              class="mb-2"
            />
            <v-text-field
              v-model.number="form.rechargeMultiplier"
              :label="t('subscription.field.rechargeMultiplier')"
              variant="outlined"
              density="compact"
              type="number"
              step="0.1"
              class="mb-2"
            />
            <v-textarea
              v-model="form.notes"
              :label="t('subscription.field.notes')"
              variant="outlined"
              density="compact"
              rows="2"
              class="mb-2"
            />
            <v-divider class="my-3" />
            <div class="text-subtitle-2 mb-2 text-medium-emphasis">{{ t('subscription.field.autoRefreshSection') }}</div>
            <v-text-field
              v-model="form.billingApiKey"
              :label="t('subscription.field.billingApiKey')"
              variant="outlined"
              density="compact"
              class="mb-2"
              type="password"
              :placeholder="t('subscription.field.billingApiKeyPlaceholder')"
              :hint="t('subscription.field.billingApiKeyHint')"
              persistent-hint
            />
            <v-switch
              v-model="form.autoRefreshEnabled"
              :label="t('subscription.field.autoRefreshEnabled')"
              color="primary"
              density="compact"
              class="mb-2"
              :disabled="!form.billingApiKey"
              :hint="t('subscription.field.autoRefreshHint')"
              persistent-hint
            />
            <v-select
              v-model="form.source"
              :label="t('subscription.field.source')"
              :items="sourceOptions"
              variant="outlined"
              density="compact"
              class="mb-2"
            />
          </v-form>
        </v-card-text>

        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeDialog">
            {{ t('app.actions.cancel') }}
          </v-btn>
          <v-btn color="primary" variant="flat" :loading="saving" @click="handleSubmit">
            {{ t('app.actions.save') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete confirmation dialog -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card class="pa-4">
        <v-card-title class="text-h6">{{ t('subscription.delete') }}</v-card-title>
        <v-card-text>
          {{ t('subscription.deleteConfirm', { name: deletingSubscription?.displayName || '' }) }}
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showDeleteDialog = false">
            {{ t('app.actions.cancel') }}
          </v-btn>
          <v-btn color="error" variant="flat" :loading="deleting" @click="confirmDelete">
            {{ t('app.actions.delete') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- new-api 接入 dialog -->
    <v-dialog v-model="showNewApiDialog" max-width="700">
      <v-card class="pa-4">
        <v-card-title class="text-h5 mb-2">
          {{ t('subscription.newApi.connect') }}
        </v-card-title>
        <v-card-text>
          <NewApiSubscriptionForm
            @created="handleNewApiCreated"
            @error="handleNewApiError"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showNewApiDialog = false">
            {{ t('app.actions.close') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Snackbar -->
    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="3000">
      {{ snackbar.message }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from '@/i18n'
import { api } from '@/services/api'
import SubscriptionPlanTable from '@/components/SubscriptionPlanTable.vue'
import NewApiSubscriptionForm from '@/components/NewApiSubscriptionForm.vue'
import type {
  SubscriptionItem,
  SubscriptionCreateRequest,
  SubscriptionUpdateRequest,
  NewApiProvisionResponse,
  SubscriptionPreset,
} from '@/services/api-types'

const { t } = useI18n()

// 预置兜底：/api/presets 不可用时保证表单选项仍可渲染。
// 与后端 shared/subscription-preset 编译期兜底保持一致。
function fallbackPreset(): SubscriptionPreset {
  return {
    originTypes: [
      { value: 'official_api', tier: 'first' },
      { value: 'official_token_plan', tier: 'first' },
      { value: 'relay', tier: 'second' },
      { value: 'community', tier: 'third' },
      { value: 'local_runtime', tier: 'local' },
      { value: 'unknown', tier: 'unknown' },
    ],
    billingModes: ['token_plan', 'pay_as_you_go', 'shared_free', 'unknown'],
    sources: ['manual', 'auto_discovered'],
    autoRefreshProviders: ['openai', 'anthropic', 'google'],
    newApiDefaults: { originType: 'relay', originTier: 'second', billingMode: 'token_plan' },
    originTypeAliases: { public_benefit: 'community' },
  }
}

const subscriptions = ref<SubscriptionItem[]>([])
// 订阅预置：来源类型/计费模式/来源等表单选项来源，替代前端硬编码副本。
// 后端 /api/presets 提供，失败时回退到 fallbackPreset 保证表单可用。
const preset = ref<SubscriptionPreset>(fallbackPreset())
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
const showNewApiDialog = ref(false)
const editingSubscription = ref<SubscriptionItem | null>(null)
const deletingSubscription = ref<SubscriptionItem | null>(null)

const snackbar = ref({ show: false, message: '', color: 'success' })

const form = ref<SubscriptionCreateRequest>({
  subscriptionUid: '',
  displayName: '',
  provider: '',
  originType: '',
  originTier: '',
  billingMode: '',
  currency: '',
  balance: 0,
  rechargeMultiplier: 1,
  notes: '',
  source: 'manual',
  billingApiKey: '',
  autoRefreshEnabled: false,
})

const originTypeOptions = computed(() =>
  preset.value.originTypes.map((o) => ({
    title: t(`subscription.originType.${o.value}`),
    value: o.value,
  })),
)

// 来源等级由来源类型推导，语义与后端 InferOriginTier 对齐：
// 先按预置别名归一化（如 public_benefit -> community），再查预置 tier；
// 未命中回退 unknown。等级映射随预置更新，无需改前端。
function inferOriginTier(originType?: string): string {
  if (!originType) return 'unknown'
  const canonical = preset.value.originTypeAliases[originType] ?? originType
  const entry = preset.value.originTypes.find((o) => o.value === canonical)
  return entry?.tier ?? 'unknown'
}

const derivedOriginTier = computed(() => inferOriginTier(form.value.originType))

const derivedOriginTierLabel = computed(() =>
  t(`subscription.originTier.${derivedOriginTier.value}`),
)

const billingModeOptions = computed(() =>
  preset.value.billingModes.map((m) => ({
    title: t(`subscription.billingMode.${m}`),
    value: m,
  })),
)

const sourceOptions = computed(() =>
  preset.value.sources.map((s) => ({
    title: t(`subscription.source.${s}`),
    value: s,
  })),
)

const filteredSubscriptions = computed(() => subscriptions.value)

async function fetchPresets() {
  try {
    const bundle = await api.getPresets()
    if (bundle?.subscription) {
      preset.value = bundle.subscription
    }
  } catch {
    // 预置拉取失败不阻断页面：保留 fallbackPreset 选项。
  }
}

async function fetchSubscriptions() {
  loading.value = true
  try {
    const resp = await api.getSubscriptions()
    subscriptions.value = resp.subscriptions || []
  } catch (e) {
    showSnackbar(e instanceof Error ? e.message : 'Unknown error', 'error')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = {
    subscriptionUid: '',
    displayName: '',
    provider: '',
    originType: '',
    originTier: '',
    billingMode: '',
    currency: '',
    balance: 0,
    rechargeMultiplier: 1,
    notes: '',
    source: 'manual',
    billingApiKey: '',
    autoRefreshEnabled: false,
  }
}

function openCreateDialog() {
  editingSubscription.value = null
  resetForm()
  showDialog.value = true
}

function openEditDialog(item: SubscriptionItem) {
  editingSubscription.value = item
  form.value = {
    subscriptionUid: item.subscriptionUid,
    displayName: item.displayName,
    provider: item.provider || '',
    originType: item.originType || '',
    originTier: item.originTier || '',
    billingMode: item.billingMode || '',
    currency: item.currency || '',
    balance: item.balance || 0,
    groupMultipliers: item.groupMultipliers,
    rechargeMultiplier: item.rechargeMultiplier || 1,
    notes: item.notes || '',
    source: item.source || 'manual',
    billingApiKey: item.billingApiKey || '',
    autoRefreshEnabled: item.autoRefreshEnabled || false,
  }
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editingSubscription.value = null
}

async function handleSubmit() {
  if (!form.value.subscriptionUid.trim() || !form.value.displayName.trim()) return

  // 来源等级始终由来源类型推导，提交时同步，不接受手动值
  form.value.originTier = derivedOriginTier.value

  saving.value = true
  try {
    if (editingSubscription.value) {
      const updateData: SubscriptionUpdateRequest = {
        displayName: form.value.displayName,
        provider: form.value.provider || undefined,
        originType: form.value.originType || undefined,
        originTier: derivedOriginTier.value,
        billingMode: form.value.billingMode || undefined,
        currency: form.value.currency || undefined,
        balance: form.value.balance,
        rechargeMultiplier: form.value.rechargeMultiplier,
        notes: form.value.notes || undefined,
        source: form.value.source || undefined,
        billingApiKey: form.value.billingApiKey || undefined,
        autoRefreshEnabled: form.value.autoRefreshEnabled,
      }
      await api.updateSubscription(editingSubscription.value.subscriptionUid, updateData)
      showSnackbar(t('app.actions.save') + ' - OK', 'success')
    } else {
      await api.createSubscription(form.value)
      showSnackbar(t('subscription.add') + ' - OK', 'success')
    }
    closeDialog()
    await fetchSubscriptions()
  } catch (e) {
    showSnackbar(e instanceof Error ? e.message : 'Unknown error', 'error')
  } finally {
    saving.value = false
  }
}

function handleDelete(item: SubscriptionItem) {
  deletingSubscription.value = item
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingSubscription.value) return

  deleting.value = true
  try {
    await api.deleteSubscription(deletingSubscription.value.subscriptionUid)
    showSnackbar(t('app.actions.delete') + ' - OK', 'success')
    showDeleteDialog.value = false
    deletingSubscription.value = null
    await fetchSubscriptions()
  } catch (e) {
    showSnackbar(e instanceof Error ? e.message : 'Unknown error', 'error')
  } finally {
    deleting.value = false
  }
}

async function handleRefresh(item: SubscriptionItem) {
  try {
    const result = await api.refreshSubscription(item.subscriptionUid)
    if (result.refreshResult.success) {
      showSnackbar(`${item.displayName}: ${t('app.actions.refresh')} OK`, 'success')
    } else {
      showSnackbar(`${item.displayName}: ${result.refreshResult.errorMessage || 'Refresh failed'}`, 'error')
    }
    await fetchSubscriptions()
  } catch (e) {
    showSnackbar(e instanceof Error ? e.message : 'Unknown error', 'error')
  }
}

function showSnackbar(message: string, color: string) {
  snackbar.value = { show: true, message, color }
}

async function handleNewApiCreated(result: NewApiProvisionResponse) {
  showNewApiDialog.value = false
  const suffix = result.discoveryStarted
    ? t('subscription.newApi.discoveryStarted')
    : ''
  showSnackbar(`${t('subscription.newApi.provisionSuccess')} ${suffix}`.trim(), 'success')
  await fetchSubscriptions()
}

function handleNewApiError(message: string) {
  showSnackbar(message, 'error')
}

onMounted(() => {
  fetchPresets()
  fetchSubscriptions()
})
</script>
