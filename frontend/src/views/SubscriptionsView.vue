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
            <v-select
              v-model="form.originTier"
              :label="t('subscription.field.originTier')"
              :items="originTierOptions"
              variant="outlined"
              density="compact"
              class="mb-2"
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
import type { SubscriptionItem, SubscriptionCreateRequest, SubscriptionUpdateRequest } from '@/services/api-types'

const { t } = useI18n()

const subscriptions = ref<SubscriptionItem[]>([])
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
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

const originTypeOptions = computed(() => [
  { title: t('subscription.originType.official_api'), value: 'official_api' },
  { title: t('subscription.originType.official_token_plan'), value: 'official_token_plan' },
  { title: t('subscription.originType.relay'), value: 'relay' },
  { title: t('subscription.originType.public_benefit'), value: 'public_benefit' },
])

const originTierOptions = computed(() => [
  { title: t('subscription.originTier.first'), value: 'first' },
  { title: t('subscription.originTier.second'), value: 'second' },
  { title: t('subscription.originTier.third'), value: 'third' },
  { title: t('subscription.originTier.unknown'), value: 'unknown' },
])

const billingModeOptions = computed(() => [
  { title: t('subscription.billingMode.token_plan'), value: 'token_plan' },
  { title: t('subscription.billingMode.pay_as_you_go'), value: 'pay_as_you_go' },
  { title: t('subscription.billingMode.shared_free'), value: 'shared_free' },
  { title: t('subscription.billingMode.unknown'), value: 'unknown' },
])

const sourceOptions = computed(() => [
  { title: t('subscription.source.manual'), value: 'manual' },
  { title: t('subscription.source.auto_discovered'), value: 'auto_discovered' },
])

const filteredSubscriptions = computed(() => subscriptions.value)

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

  saving.value = true
  try {
    if (editingSubscription.value) {
      const updateData: SubscriptionUpdateRequest = {
        displayName: form.value.displayName,
        provider: form.value.provider || undefined,
        originType: form.value.originType || undefined,
        originTier: form.value.originTier || undefined,
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

onMounted(fetchSubscriptions)
</script>
