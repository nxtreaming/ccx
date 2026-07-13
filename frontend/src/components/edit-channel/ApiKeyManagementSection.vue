<template>
  <div class="api-key-management-section">
    <v-card variant="outlined" rounded="lg" :color="hasConfigurableKeys ? undefined : 'error'">
      <v-card-title class="d-flex align-center justify-space-between pa-4 pb-2">
        <div class="d-flex align-center ga-2">
          <v-icon :color="hasConfigurableKeys ? 'primary' : 'error'">mdi-key</v-icon>
          <span class="section-title">
            {{ t('channelCard.apiKeyManagement') }}<span v-if="serviceType !== 'copilot'"> *</span>
          </span>
          <v-chip v-if="!hasConfigurableKeys" size="x-small" color="error" variant="tonal">
            {{ t('channelEditor.auth.apiKeyRequired') }}
          </v-chip>
        </div>
        <v-chip size="small" color="info" variant="tonal">
          {{ t('addChannel.apiKeyLoadBalance') }}
        </v-chip>
      </v-card-title>

      <v-card-text class="pt-2">
        <!-- 现有密钥列表 -->
        <div v-if="apiKeys.length" class="mb-4">
          <v-list density="compact" class="bg-transparent">
            <v-list-item
              v-for="(key, index) in apiKeys"
              :key="index"
              class="mb-2"
              rounded="lg"
              variant="tonal"
              :color="duplicateKeyIndex === index ? 'error' : 'surface-variant'"
              :class="{ 'animate-pulse': duplicateKeyIndex === index }"
            >
              <template #prepend>
                <v-icon size="small" :color="duplicateKeyIndex === index ? 'error' : 'primary'">
                  {{ duplicateKeyIndex === index ? 'mdi-alert' : 'mdi-key' }}
                </v-icon>
              </template>

              <v-list-item-title>
                <div class="d-flex align-center justify-space-between">
                  <code class="text-caption">{{ maskApiKey(key) }}</code>
                  <div class="d-flex align-center ga-1">
                    <!-- Models 状态标签 -->
                    <v-chip
                      v-if="keyModelsStatus.get(key)?.loading"
                      size="x-small"
                      color="info"
                      variant="tonal"
                    >
                      <v-icon start size="12">mdi-loading</v-icon>
                      {{ t('addChannel.checking') }}
                    </v-chip>
                    <v-chip
                      v-else-if="keyModelsStatus.get(key)?.success"
                      size="x-small"
                      color="success"
                      variant="tonal"
                    >
                      {{ t('addChannel.modelsCount', { statusCode: keyModelsStatus.get(key)?.statusCode ?? 'OK', count: keyModelsStatus.get(key)?.modelCount ?? 0 }) }}
                    </v-chip>
                    <v-tooltip
                      v-else-if="keyModelsStatus.get(key)?.error"
                      :text="keyModelsStatus.get(key)?.error"
                      location="top"
                      max-width="300"
                      content-class="key-tooltip"
                    >
                      <template #activator="{ props: tooltipProps }">
                        <v-chip
                          v-bind="tooltipProps"
                          size="x-small"
                          color="error"
                          variant="tonal"
                        >
                          models {{ keyModelsStatus.get(key)?.statusCode || 'ERR' }}
                        </v-chip>
                      </template>
                    </v-tooltip>
                    <!-- 重复密钥标签 -->
                    <v-chip v-if="duplicateKeyIndex === index" size="x-small" color="error" variant="text">
                      {{ t('channelEditor.auth.duplicateKey') }}
                    </v-chip>
                  </div>
                </div>
              </v-list-item-title>

              <template #append>
                <div class="d-flex align-center ga-1">
                  <!-- 置顶/置底：仅首尾密钥显示 -->
                  <v-tooltip
                    v-if="!isAutoManaged && index === apiKeys.length - 1 && apiKeys.length > 1"
                    :text="t('channelCard.moveTop')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        color="warning"
                        icon
                        variant="text"
                        rounded="md"
                        @click="moveToTop(index)"
                      >
                        <v-icon size="small">mdi-arrow-up-bold</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                  <v-tooltip
                    v-if="!isAutoManaged && index === 0 && apiKeys.length > 1"
                    :text="t('channelCard.moveBottom')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        color="warning"
                        icon
                        variant="text"
                        rounded="md"
                        @click="moveToBottom(index)"
                      >
                        <v-icon size="small">mdi-arrow-down-bold</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                  <v-tooltip
                    :text="copiedKeyIndex === index ? t('channelCard.copied') : t('channelCard.copyKey')"
                    location="top"
                    :open-delay="150"
                    content-class="key-tooltip"
                  >
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        :color="copiedKeyIndex === index ? 'success' : 'primary'"
                        icon
                        variant="text"
                        @click="copyKey(key, index)"
                      >
                        <v-icon size="small">{{
                          copiedKeyIndex === index ? 'mdi-check' : 'mdi-content-copy'
                        }}</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                  <v-tooltip :text="t('addChannel.deleteKey')" location="top" :open-delay="150" content-class="key-tooltip">
                    <template #activator="{ props: tooltipProps }">
                      <v-btn
                        v-bind="tooltipProps"
                        size="small"
                        color="error"
                        icon
                        variant="text"
                        @click="removeKey(index)"
                      >
                        <v-icon size="small" color="error">mdi-close</v-icon>
                      </v-btn>
                    </template>
                  </v-tooltip>
                </div>
              </template>
            </v-list-item>
          </v-list>
        </div>

        <div v-if="providerId === 'volcengine' && accountUid" class="volcengine-access-keys mb-5">
          <v-divider class="mb-4" />
          <div class="d-flex align-center justify-space-between ga-3 flex-wrap mb-2">
            <div class="d-flex align-center ga-2">
              <v-icon color="primary" size="small">mdi-shield-key-outline</v-icon>
              <span class="text-body-2 font-weight-medium">{{ t('volcengineAccessKey.title') }}</span>
            </div>
            <v-btn
              href="https://console.volcengine.com/iam/keymanage"
              target="_blank"
              rel="noopener noreferrer"
              size="small"
              variant="text"
              color="primary"
            >
              <v-icon start size="small">mdi-open-in-new</v-icon>
              {{ t('volcengineAccessKey.openConsole') }}
            </v-btn>
          </div>
          <div class="text-caption text-medium-emphasis mb-3">
            {{ t('volcengineAccessKey.hint') }}
          </div>
          <v-progress-linear v-if="volcengineCredentialsLoading" indeterminate color="primary" class="mb-3" />
          <v-alert v-if="volcengineCredentialsError" color="error" variant="tonal" density="compact" class="mb-3">
            {{ volcengineCredentialsError }}
          </v-alert>
          <div
            v-for="credential in volcengineCredentials"
            :key="credential.credentialUid"
            class="volcengine-credential py-3"
          >
            <div class="d-flex align-center justify-space-between ga-3 flex-wrap mb-3">
              <code class="text-caption">{{ credential.keyMask }}</code>
              <div class="d-flex align-center ga-2 flex-wrap">
                <v-chip v-if="credential.volcenginePlan" size="x-small" color="success" variant="tonal">
                  {{ planDisplayName(credential.volcenginePlan) }}
                  <span v-if="credential.volcenginePlanTier"> · {{ credential.volcenginePlanTier }}</span>
                </v-chip>
                <v-chip
                  :color="credential.hasVolcengineAccessKey ? 'info' : 'warning'"
                  size="x-small"
                  variant="tonal"
                >
                  {{ credential.hasVolcengineAccessKey
                    ? credential.volcengineAccessKeyIdMask
                    : t('volcengineAccessKey.notConfigured') }}
                </v-chip>
              </div>
            </div>
            <div v-if="credential.hasVolcengineAccessKey" class="volcengine-usage mb-3">
              <div class="d-flex align-center justify-space-between ga-2 mb-2">
                <span class="text-caption text-medium-emphasis">{{ t('volcengineAccessKey.usageTitle') }}</span>
                <div class="d-flex align-center ga-2">
                  <span
                    v-if="credential.volcenginePlanUsage?.fetchedAt && !credential.volcenginePlanUsage.error"
                    class="text-caption text-disabled"
                  >
                    {{ t('volcengineAccessKey.usageFetchedAt') }} {{ formatVolcengineTime(credential.volcenginePlanUsage.fetchedAt) }}
                  </span>
                  <v-btn
                    icon
                    size="x-small"
                    variant="text"
                    :loading="volcengineUsageRefreshing[credential.credentialUid]"
                    :title="t('volcengineAccessKey.refresh')"
                    @click="refreshVolcengineUsage(credential)"
                  >
                    <v-icon size="small">mdi-refresh</v-icon>
                  </v-btn>
                </div>
              </div>
              <div
                v-if="hasVolcengineUsageData(credential.volcenginePlanUsage)"
                class="volcengine-usage-grid"
              >
                <div v-for="win in volcengineUsageWindows(credential.volcenginePlanUsage)" :key="win.labelKey">
                  <div class="text-caption text-medium-emphasis">{{ t(win.labelKey) }}</div>
                  <div class="text-body-2 font-weight-medium" :class="win.colorClass">{{ win.text }}</div>
                </div>
              </div>
              <div v-else class="text-caption text-disabled">{{ t('volcengineAccessKey.noUsageData') }}</div>
            </div>
            <div v-if="volcengineForms[credential.credentialUid]" class="d-flex flex-column ga-2">
              <div class="volcengine-key-fields">
                <v-text-field
                  v-model="volcengineForms[credential.credentialUid].accessKeyId"
                  :label="t('volcengineAccessKey.accessKeyId')"
                  variant="outlined"
                  density="compact"
                  autocomplete="off"
                  hide-details
                />
                <v-text-field
                  v-model="volcengineForms[credential.credentialUid].secretAccessKey"
                  :label="t('volcengineAccessKey.secretAccessKey')"
                  type="password"
                  variant="outlined"
                  density="compact"
                  autocomplete="new-password"
                  hide-details
                />
              </div>
              <v-alert
                v-if="volcengineForms[credential.credentialUid].error"
                color="error"
                variant="tonal"
                density="compact"
              >
                {{ volcengineForms[credential.credentialUid].error }}
              </v-alert>
              <div class="d-flex align-center justify-end ga-2">
                <v-btn
                  v-if="credential.hasVolcengineAccessKey"
                  size="small"
                  variant="text"
                  color="error"
                  :loading="volcengineForms[credential.credentialUid].clearing"
                  @click="clearVolcengineAccessKey(credential)"
                >
                  <v-icon start size="small">mdi-delete-outline</v-icon>
                  {{ t('volcengineAccessKey.clear') }}
                </v-btn>
                <v-btn
                  size="small"
                  variant="tonal"
                  color="primary"
                  :loading="volcengineForms[credential.credentialUid].saving"
                  :disabled="!canSaveVolcengineCredential(credential.credentialUid)"
                  @click="saveVolcengineAccessKey(credential)"
                >
                  <v-icon start size="small">mdi-content-save-outline</v-icon>
                  {{ t('volcengineAccessKey.verifyAndSave') }}
                </v-btn>
              </div>
            </div>
          </div>
        </div>

        <div v-if="providerId === 'mimo' && accountUid" class="mimo-console-cookies mb-5">
          <v-divider class="mb-4" />
          <div class="d-flex align-center justify-space-between ga-3 flex-wrap mb-2">
            <div class="d-flex align-center ga-2">
              <v-icon color="primary" size="small">mdi-cookie-cog-outline</v-icon>
              <span class="text-body-2 font-weight-medium">{{ t('mimoConsoleCookie.title') }}</span>
            </div>
            <v-btn
              href="https://platform.xiaomimimo.com/console/plan-manage"
              target="_blank"
              rel="noopener noreferrer"
              size="small"
              variant="text"
              color="primary"
            >
              <v-icon start size="small">mdi-open-in-new</v-icon>
              {{ t('mimoConsoleCookie.openConsole') }}
            </v-btn>
          </div>
          <div class="text-caption text-medium-emphasis mb-3">{{ t('mimoConsoleCookie.hint') }}</div>
          <v-progress-linear v-if="mimoCredentialsLoading" indeterminate color="primary" class="mb-3" />
          <v-alert v-if="mimoCredentialsError" color="error" variant="tonal" density="compact" class="mb-3">
            {{ mimoCredentialsError }}
          </v-alert>
          <div v-for="credential in mimoCredentials" :key="credential.credentialUid" class="volcengine-credential py-3">
            <div class="d-flex align-center justify-space-between ga-3 flex-wrap mb-3">
              <code class="text-caption">{{ credential.keyMask }}</code>
              <div class="d-flex align-center ga-2">
                <v-chip v-if="credential.mimoTokenPlan" size="x-small" color="success" variant="tonal">
                  {{ credential.mimoTokenPlan.planName }}
                </v-chip>
                <v-chip :color="credential.hasMiMoConsoleCookie ? 'info' : 'warning'" size="x-small" variant="tonal">
                  {{ credential.hasMiMoConsoleCookie
                    ? t('mimoConsoleCookie.configured')
                    : t('mimoConsoleCookie.notConfigured') }}
                </v-chip>
              </div>
            </div>
            <div v-if="credential.mimoTokenPlan" class="mimo-usage-grid mb-3">
              <div>
                <div class="text-caption text-medium-emphasis">{{ t('mimoConsoleCookie.currentRemaining') }}</div>
                <div class="text-body-2 font-weight-medium">{{ formatMiMoQuota(credential.mimoTokenPlan.currentUsage) }}</div>
              </div>
              <div>
                <div class="text-caption text-medium-emphasis">{{ t('mimoConsoleCookie.monthRemaining') }}</div>
                <div class="text-body-2 font-weight-medium">{{ formatMiMoQuota(credential.mimoTokenPlan.monthUsage) }}</div>
              </div>
              <div>
                <div class="text-caption text-medium-emphasis">{{ t('mimoConsoleCookie.expiresAt') }}</div>
                <div class="text-body-2 font-weight-medium">{{ credential.mimoTokenPlan.currentPeriodEnd }}</div>
              </div>
            </div>
            <div v-if="mimoForms[credential.credentialUid]" class="d-flex flex-column ga-2">
              <v-text-field
                v-model="mimoForms[credential.credentialUid].cookie"
                :label="t('mimoConsoleCookie.cookie')"
                :placeholder="t('mimoConsoleCookie.cookiePlaceholder')"
                type="password"
                variant="outlined"
                density="compact"
                autocomplete="new-password"
                hide-details
              />
              <v-alert v-if="mimoForms[credential.credentialUid].error" color="error" variant="tonal" density="compact">
                {{ mimoForms[credential.credentialUid].error }}
              </v-alert>
              <div class="d-flex align-center justify-end ga-2 flex-wrap">
                <v-btn
                  v-if="credential.hasMiMoConsoleCookie"
                  size="small"
                  variant="text"
                  color="secondary"
                  :loading="mimoForms[credential.credentialUid].refreshing"
                  @click="refreshMiMoConsoleCookie(credential)"
                >
                  <v-icon start size="small">mdi-refresh</v-icon>
                  {{ t('mimoConsoleCookie.refresh') }}
                </v-btn>
                <v-btn
                  v-if="credential.hasMiMoConsoleCookie"
                  size="small"
                  variant="text"
                  color="error"
                  :loading="mimoForms[credential.credentialUid].clearing"
                  @click="clearMiMoConsoleCookie(credential)"
                >
                  <v-icon start size="small">mdi-delete-outline</v-icon>
                  {{ t('mimoConsoleCookie.clear') }}
                </v-btn>
                <v-btn
                  size="small"
                  variant="tonal"
                  color="primary"
                  :loading="mimoForms[credential.credentialUid].saving"
                  :disabled="!mimoForms[credential.credentialUid].cookie.trim()"
                  @click="saveMiMoConsoleCookie(credential)"
                >
                  <v-icon start size="small">mdi-check-decagram-outline</v-icon>
                  {{ t('mimoConsoleCookie.verifyAndSave') }}
                </v-btn>
              </div>
            </div>
          </div>
        </div>

        <!-- 添加新密钥 -->
        <div class="d-flex align-start ga-3">
          <v-text-field
            v-model="newApiKey"
            :label="t('addChannel.addNewApiKey')"
            :placeholder="t('channelEditor.auth.addNewApiKey.placeholder')"
            prepend-inner-icon="mdi-plus"
            variant="outlined"
            density="comfortable"
            type="password"
            :error="!!apiKeyError"
            :error-messages="apiKeyError"
            class="flex-grow-1"
            @keyup.enter="handleAddKey"
            @input="handleInput"
          />
          <v-btn
            color="primary"
            variant="elevated"
            size="large"
            height="40"
            :disabled="!newApiKey.trim()"
            class="mt-1"
            @click="handleAddKey"
          >
            {{ t('app.actions.add') }}
          </v-btn>
        </div>

        <v-alert
          v-if="serviceType === 'copilot'"
          class="mt-3"
          color="info"
          variant="tonal"
          density="comfortable"
        >
          <div class="d-flex flex-column ga-2">
            <div class="text-body-2">{{ t('copilotOAuth.description') }}</div>
            <v-text-field
              :model-value="proxyUrl || ''"
              :label="t('channelEditor.transport.proxyUrl.label')"
              :placeholder="t('channelEditor.transport.proxyUrl.placeholder')"
              :hint="t('channelEditor.transport.proxyUrl.hint')"
              prepend-inner-icon="mdi-shield-lock-outline"
              variant="outlined"
              density="compact"
              clearable
              persistent-hint
              @update:model-value="$emit('update:proxyUrl', String($event || ''))"
            />
            <div v-if="copilotUserCode" class="d-flex align-center flex-wrap ga-2">
              <span class="text-body-2">{{ t('copilotOAuth.userCode') }}</span>
              <code class="px-2 py-1 rounded bg-surface">{{ copilotUserCode }}</code>
              <v-btn
                size="small"
                color="primary"
                variant="text"
                :href="copilotVerificationUri"
                target="_blank"
                rel="noopener noreferrer"
              >
                <v-icon start size="small">mdi-open-in-new</v-icon>
                {{ t('copilotOAuth.openAuthorize') }}
              </v-btn>
            </div>
            <v-alert v-if="copilotOAuthSuccess" color="success" variant="tonal" density="compact">
              {{ t('copilotOAuth.success') }}
            </v-alert>
            <v-alert v-if="copilotOAuthError" color="error" variant="tonal" density="compact">
              {{ copilotOAuthError }}
            </v-alert>
            <div class="d-flex align-center ga-2">
              <v-btn
                color="primary"
                variant="tonal"
                :loading="copilotOAuthLoading"
                @click="startCopilotOAuth"
              >
                <v-icon start>mdi-code-braces</v-icon>
                {{ t('copilotOAuth.button') }}
              </v-btn>
              <span v-if="copilotPolling" class="text-caption text-medium-emphasis">
                {{ t('copilotOAuth.waiting') }}
              </span>
            </div>
            <div class="d-flex align-center ga-2 mt-1">
              <v-btn
                v-if="copilotPolling || copilotOAuthLoading"
                size="small"
                color="warning"
                variant="text"
                @click="cancelCopilotOAuth"
              >
                {{ t('copilotOAuth.cancel') }}
              </v-btn>
              <v-btn
                v-if="copilotOAuthError"
                size="small"
                color="primary"
                variant="text"
                @click="retryCopilotOAuth"
              >
                {{ t('copilotOAuth.retry') }}
              </v-btn>
            </div>
            <div v-if="serviceType === 'copilot' && isEditing && channelId !== undefined" class="mt-3 d-flex flex-column ga-2">
              <div class="d-flex align-center ga-2">
                <v-btn
                  color="secondary"
                  variant="tonal"
                  :loading="copilotDiagnoseLoading"
                  @click="diagnoseCopilotChannel"
                >
                  {{ t('copilotDiagnose.button') }}
                </v-btn>
                <span v-if="copilotDiagnoseLoading" class="text-caption text-medium-emphasis">
                  {{ t('copilotDiagnose.loading') }}
                </span>
              </div>
              <v-alert v-if="copilotDiagnoseError" color="error" variant="tonal" density="compact">
                {{ copilotDiagnoseError }}
              </v-alert>
              <v-alert v-if="copilotDiagnoseResult" color="info" variant="tonal" density="compact" class="text-caption">
                <div class="d-flex flex-column ga-2">
                  <div class="d-flex align-center ga-2">
                    <v-chip :color="copilotDiagnoseResult.githubUser ? 'success' : 'warning'" size="small" variant="tonal">
                      GitHub
                    </v-chip>
                    <span v-if="copilotDiagnoseResult.githubUser">{{ copilotDiagnoseResult.githubUser.login }}</span>
                    <span v-else>{{ copilotDiagnoseResult.githubUserError }}</span>
                  </div>
                  <div class="d-flex align-center ga-2">
                    <v-chip :color="copilotDiagnoseResult.tokenError ? 'error' : 'success'" size="small" variant="tonal">
                      Token
                    </v-chip>
                    <span v-if="copilotDiagnoseResult.tokenError">{{ copilotDiagnoseResult.tokenError }}</span>
                    <span v-else>{{ copilotDiagnoseResult.copilotBaseUrl }}</span>
                  </div>
                  <div class="d-flex align-center ga-2">
                    <v-chip :color="copilotDiagnoseResult.modelsError ? 'error' : (copilotDiagnoseResult.modelsStatus && copilotDiagnoseResult.modelsStatus < 400 ? 'success' : 'warning')" size="small" variant="tonal">
                      Models
                    </v-chip>
                    <span v-if="copilotDiagnoseResult.modelsError">{{ copilotDiagnoseResult.modelsError }}</span>
                    <span v-else>{{ copilotDiagnoseResult.modelsStatus }} {{ copilotDiagnoseResult.modelsUrl }}</span>
                  </div>
                </div>
              </v-alert>
            </div>
          </div>
        </v-alert>

        <!-- 被拉黑的密钥（仅编辑模式） -->
        <div v-if="isEditing && visibleDisabledKeys.length" class="mt-4">
          <div class="d-flex align-center ga-2 mb-2">
            <v-icon size="small" color="error">mdi-key-remove</v-icon>
            <span class="text-body-2 font-weight-medium text-error">{{ t('channelCard.disabledKeys') }}</span>
            <v-chip size="x-small" color="error" variant="tonal">{{ visibleDisabledKeys.length }}</v-chip>
          </div>
          <v-list density="compact" class="rounded-lg" style="max-height: 150px; overflow-y: auto;">
            <v-list-item
              v-for="(dk, dkIdx) in visibleDisabledKeys"
              :key="'disabled-' + dkIdx"
              class="px-3"
              style="background: rgba(var(--v-theme-error), 0.04);"
            >
              <template #prepend>
                <v-icon size="small" color="error" class="mr-2">mdi-key-alert</v-icon>
              </template>
              <v-list-item-title class="text-caption font-weight-mono">
                {{ dk.key.length > 20 ? dk.key.slice(0, 8) + '***' + dk.key.slice(-5) : dk.key }}
              </v-list-item-title>
              <v-list-item-subtitle class="d-flex align-center ga-1">
                <v-chip size="x-small" :color="dk.reason === 'insufficient_balance' ? 'warning' : 'error'" variant="tonal">
                  {{ t(getDisabledKeyLabel(dk.reason)) }}
                </v-chip>
                <span class="text-caption">{{ new Date(dk.disabledAt).toLocaleDateString() }}</span>
              </v-list-item-subtitle>
              <template #append>
                <v-btn
                  size="x-small"
                  color="success"
                  variant="tonal"
                  rounded="lg"
                  :loading="restoringKey === dk.key"
                  @click="$emit('restore-key', dk.key)"
                >
                  <v-icon start size="small">mdi-restore</v-icon>
                  {{ t('channelCard.restoreKey') }}
                </v-btn>
              </template>
            </v-list-item>
          </v-list>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from '../../i18n'
import { ApiError, ApiService } from '../../services/api'
import type { ManagedAccountCredential, MiMoTokenPlanQuota, VolcenginePlanUsage, VolcenginePlanUsageWindow } from '../../services/api-types'
import { maskApiKey } from '../../utils/apiKeyMask'

interface KeyModelsStatus {
  loading?: boolean
  success?: boolean
  error?: string
  statusCode?: string | number
  modelCount?: number
}

interface DisabledKey {
  key: string
  reason: string
  disabledAt: string
}

interface Props {
  apiKeys: string[]
  disabledKeys: DisabledKey[]
  keyModelsStatus: Map<string, KeyModelsStatus>
  isEditing: boolean
  restoringKey: string
  serviceType?: string
  isAutoManaged?: boolean
  channelId?: number
  proxyUrl?: string
  accountUid?: string
  providerId?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:apiKeys': [string[]]
  'update:proxyUrl': [string]
  'restore-key': [string]
}>()

const { t } = useI18n()
const apiService = new ApiService()

const newApiKey = ref('')
const apiKeyError = ref('')
const duplicateKeyIndex = ref<number | null>(null)
const copiedKeyIndex = ref<number | null>(null)
interface VolcengineCredentialForm {
  accessKeyId: string
  secretAccessKey: string
  saving: boolean
  clearing: boolean
  error: string
}

const volcengineCredentials = ref<ManagedAccountCredential[]>([])
const volcengineCredentialsLoading = ref(false)
const volcengineCredentialsError = ref('')
const volcengineForms = ref<Record<string, VolcengineCredentialForm>>({})
const volcengineUsageRefreshing = ref<Record<string, boolean>>({})
interface MiMoCredentialForm {
  cookie: string
  saving: boolean
  refreshing: boolean
  clearing: boolean
  error: string
}

const mimoCredentials = ref<ManagedAccountCredential[]>([])
const mimoCredentialsLoading = ref(false)
const mimoCredentialsError = ref('')
const mimoForms = ref<Record<string, MiMoCredentialForm>>({})
interface CopilotDiagnoseResponse {
  githubUser?: {
    login?: string
    id?: number
  }
  githubUserError?: string
  copilotBaseUrl?: string
  tokenError?: string
  tokenErrorKind?: string
  modelsUrl?: string
  modelsStatus?: number
  modelsError?: string
  modelsBodyPrefix?: string
}

const copilotOAuthLoading = ref(false)
const copilotPolling = ref(false)
const copilotOAuthError = ref('')
const copilotOAuthSuccess = ref(false)
const copilotDeviceCode = ref('')
const copilotUserCode = ref('')
const copilotVerificationUri = ref('')
const copilotDiagnoseLoading = ref(false)
const copilotDiagnoseResult = ref<CopilotDiagnoseResponse | null>(null)
const copilotDiagnoseError = ref('')
let copilotPollTimer: number | null = null

const hasConfigurableKeys = computed(() => props.serviceType === 'copilot' || props.apiKeys.length > 0)

const visibleDisabledKeys = computed(() => {
  return props.disabledKeys.filter(dk => !props.apiKeys.includes(dk.key))
})

const loadVolcengineCredentials = async () => {
  volcengineCredentials.value = []
  volcengineCredentialsError.value = ''
  if (props.providerId !== 'volcengine' || !props.accountUid) return
  volcengineCredentialsLoading.value = true
  try {
    const response = await apiService.getManagedAccounts()
    const account = response.accounts.find(item => item.accountUid === props.accountUid)
    if (!account) {
      volcengineCredentialsError.value = t('volcengineAccessKey.accountNotFound')
      return
    }
    volcengineCredentials.value = account.credentials
    const nextForms: Record<string, VolcengineCredentialForm> = {}
    for (const credential of account.credentials) {
      nextForms[credential.credentialUid] = volcengineForms.value[credential.credentialUid] ?? {
        accessKeyId: '',
        secretAccessKey: '',
        saving: false,
        clearing: false,
        error: '',
      }
    }
    volcengineForms.value = nextForms
    // 打开编辑框时对已绑定 AK 的凭证自动刷新用量（后端 TTL 兜底避免频繁请求）。
    void Promise.allSettled(
      account.credentials
        .filter(credential => credential.hasVolcengineAccessKey)
        .map(credential => refreshVolcengineUsage(credential))
    )
  } catch (err) {
    volcengineCredentialsError.value = err instanceof Error ? err.message : String(err)
  } finally {
    volcengineCredentialsLoading.value = false
  }
}

watch(
  () => [props.providerId, props.accountUid],
  () => { void loadVolcengineCredentials() },
  { immediate: true }
)

const loadMiMoCredentials = async () => {
  mimoCredentials.value = []
  mimoCredentialsError.value = ''
  if (props.providerId !== 'mimo' || !props.accountUid) return
  mimoCredentialsLoading.value = true
  try {
    const response = await apiService.getManagedAccounts()
    const account = response.accounts.find(item => item.accountUid === props.accountUid)
    if (!account) {
      mimoCredentialsError.value = t('mimoConsoleCookie.accountNotFound')
      return
    }
    mimoCredentials.value = account.credentials
    const nextForms: Record<string, MiMoCredentialForm> = {}
    for (const credential of account.credentials) {
      nextForms[credential.credentialUid] = mimoForms.value[credential.credentialUid] ?? {
        cookie: '', saving: false, refreshing: false, clearing: false, error: '',
      }
    }
    mimoForms.value = nextForms
  } catch (err) {
    mimoCredentialsError.value = err instanceof Error ? err.message : String(err)
  } finally {
    mimoCredentialsLoading.value = false
  }
}

watch(
  () => [props.providerId, props.accountUid],
  () => { void loadMiMoCredentials() },
  { immediate: true }
)

const applyMiMoCookieResponse = (credential: ManagedAccountCredential, response: Awaited<ReturnType<ApiService['setMiMoConsoleCookie']>>) => {
  if (response.keyAdopted && response.adoptedApiKey) {
    const keyIndex = props.apiKeys.findIndex(key => maskApiKey(key) === credential.keyMask)
    if (keyIndex >= 0) {
      const updated = [...props.apiKeys]
      updated[keyIndex] = response.adoptedApiKey
      emit('update:apiKeys', updated)
    }
    credential.keyMask = response.keyMask
  }
  credential.hasMiMoConsoleCookie = true
  credential.mimoTokenPlan = response.tokenPlan
  mimoForms.value[credential.credentialUid].cookie = ''
}

const bindMiMoConsoleCookie = async (credential: ManagedAccountCredential, adoptCookieKey: boolean) => {
  if (!props.accountUid) return
  const form = mimoForms.value[credential.credentialUid]
  const response = await apiService.setMiMoConsoleCookie(props.accountUid, credential.credentialUid, {
    cookie: form.cookie.trim(),
    adoptCookieKey,
  })
  applyMiMoCookieResponse(credential, response)
}

const saveMiMoConsoleCookie = async (credential: ManagedAccountCredential) => {
  const form = mimoForms.value[credential.credentialUid]
  if (!form?.cookie.trim()) return
  form.saving = true
  form.error = ''
  try {
    await bindMiMoConsoleCookie(credential, false)
  } catch (err) {
    const details = err instanceof ApiError && typeof err.details === 'object' && err.details
      ? err.details as { code?: string; currentKeyMask?: string; cookieKeyMask?: string }
      : null
    if (err instanceof ApiError && err.status === 409 && details?.code === 'mimo_cookie_key_mismatch') {
      const confirmed = window.confirm(t('mimoConsoleCookie.keyMismatchConfirm', {
        currentKey: details.currentKeyMask ?? '-',
        cookieKey: details.cookieKeyMask ?? '-',
      }))
      if (confirmed) {
        try {
          await bindMiMoConsoleCookie(credential, true)
          return
        } catch (adoptError) {
          form.error = adoptError instanceof Error ? adoptError.message : String(adoptError)
          return
        }
      }
    }
    form.error = err instanceof Error ? err.message : String(err)
  } finally {
    form.saving = false
  }
}

const refreshMiMoConsoleCookie = async (credential: ManagedAccountCredential) => {
  if (!props.accountUid) return
  const form = mimoForms.value[credential.credentialUid]
  form.refreshing = true
  form.error = ''
  try {
    const response = await apiService.refreshMiMoConsoleCookie(props.accountUid, credential.credentialUid)
    credential.mimoTokenPlan = response.tokenPlan
  } catch (err) {
    form.error = err instanceof Error ? err.message : String(err)
  } finally {
    form.refreshing = false
  }
}

const clearMiMoConsoleCookie = async (credential: ManagedAccountCredential) => {
  if (!props.accountUid || !window.confirm(t('mimoConsoleCookie.clearConfirm'))) return
  const form = mimoForms.value[credential.credentialUid]
  form.clearing = true
  form.error = ''
  try {
    await apiService.clearMiMoConsoleCookie(props.accountUid, credential.credentialUid)
    credential.hasMiMoConsoleCookie = false
    credential.mimoTokenPlan = undefined
  } catch (err) {
    form.error = err instanceof Error ? err.message : String(err)
  } finally {
    form.clearing = false
  }
}

const formatMiMoQuota = (quota: MiMoTokenPlanQuota) => {
  const remainingPercent = Math.max(0, Math.min(100, (1 - quota.usedPercent) * 100)).toFixed(1)
  const remaining = Math.max(0, quota.limit - quota.used)
  return `${remainingPercent}% · ${Intl.NumberFormat().format(remaining)} tokens`
}

const planDisplayName = (plan?: string) => plan === 'agent_plan' ? 'Agent Plan' : 'Coding Plan'

interface VolcengineUsageCell {
  labelKey: string
  text: string
  colorClass: string
}

const numberFmt = new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 })

const hasVolcengineUsageData = (usage?: VolcenginePlanUsage) => {
  if (!usage || usage.error) return false
  return !!(usage.fiveHour || usage.daily || usage.weekly || usage.monthly)
}

// 根据剩余百分比着色：低于 20% 红，低于 50% 橙。
const volcengineUsageColor = (remainingPercent: number): string => {
  if (remainingPercent < 20) return 'text-error'
  if (remainingPercent < 50) return 'text-warning'
  return ''
}

// 单窗口展示：Agent Plan 显示剩余%与已用/额度，Coding Plan 显示剩余%与已用%。
const volcengineWindowCell = (labelKey: string, win?: VolcenginePlanUsageWindow): VolcengineUsageCell | null => {
  if (!win) return null
  if (typeof win.usedPercent === 'number' && Number.isFinite(win.usedPercent)) {
    const usedPercent = Math.max(0, Math.min(100, win.usedPercent))
    const remainingPercent = 100 - usedPercent
    return {
      labelKey,
      text: `${t('volcengineAccessKey.remaining')} ${remainingPercent.toFixed(1)}% · ${t('volcengineAccessKey.used')} ${usedPercent.toFixed(1)}%`,
      colorClass: volcengineUsageColor(remainingPercent),
    }
  }
  if (win.quota && win.quota > 0) {
    const remaining = Math.max(0, win.quota - win.used)
    const remainingPercent = Math.max(0, Math.min(100, (remaining / win.quota) * 100))
    return {
      labelKey,
      text: `${remainingPercent.toFixed(1)}% · ${numberFmt.format(win.used)}/${numberFmt.format(win.quota)}`,
      colorClass: volcengineUsageColor(remainingPercent),
    }
  }
  return {
    labelKey,
    text: `${t('volcengineAccessKey.used')} ${numberFmt.format(win.used)}`,
    colorClass: '',
  }
}

const volcengineUsageWindows = (usage?: VolcenginePlanUsage): VolcengineUsageCell[] => {
  if (!usage) return []
  return [
    volcengineWindowCell('volcengineAccessKey.fiveHourWindow', usage.fiveHour),
    volcengineWindowCell('volcengineAccessKey.dailyWindow', usage.daily),
    volcengineWindowCell('volcengineAccessKey.weeklyWindow', usage.weekly),
    volcengineWindowCell('volcengineAccessKey.monthlyWindow', usage.monthly),
  ].filter((cell): cell is VolcengineUsageCell => cell !== null)
}

const formatVolcengineTime = (iso: string): string => {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString()
}

const refreshVolcengineUsage = async (credential: ManagedAccountCredential) => {
  if (!props.accountUid || !credential.hasVolcengineAccessKey) return
  volcengineUsageRefreshing.value[credential.credentialUid] = true
  try {
    const response = await apiService.refreshVolcenginePlanUsage(props.accountUid, credential.credentialUid)
    credential.volcenginePlanUsage = response.usage
  } catch (err) {
    // 用量刷新失败降级为快照 error，不打断编辑框其它操作。
    credential.volcenginePlanUsage = {
      fetchedAt: new Date().toISOString(),
      error: err instanceof Error ? err.message : String(err),
    }
  } finally {
    volcengineUsageRefreshing.value[credential.credentialUid] = false
  }
}

const canSaveVolcengineCredential = (credentialUid: string) => {
  const form = volcengineForms.value[credentialUid]
  return !!form?.accessKeyId.trim() && !!form?.secretAccessKey.trim() && !form.saving
}

const saveVolcengineAccessKey = async (credential: ManagedAccountCredential) => {
  if (!props.accountUid) return
  const form = volcengineForms.value[credential.credentialUid]
  if (!form || !canSaveVolcengineCredential(credential.credentialUid)) return
  form.saving = true
  form.error = ''
  try {
    const response = await apiService.setVolcengineAccessKey(props.accountUid, credential.credentialUid, {
      accessKeyId: form.accessKeyId.trim(),
      secretAccessKey: form.secretAccessKey.trim(),
    })
    credential.hasVolcengineAccessKey = true
    credential.volcengineAccessKeyIdMask = response.accessKeyIdMask
    credential.volcenginePlan = response.plan
    credential.volcenginePlanTier = response.planTier
    credential.volcenginePlanStatus = response.planStatus
    credential.volcenginePlanUsage = response.usage
    form.accessKeyId = ''
    form.secretAccessKey = ''
  } catch (err) {
    form.error = err instanceof Error ? err.message : String(err)
  } finally {
    form.saving = false
  }
}

const clearVolcengineAccessKey = async (credential: ManagedAccountCredential) => {
  if (!props.accountUid || !window.confirm(t('volcengineAccessKey.clearConfirm'))) return
  const form = volcengineForms.value[credential.credentialUid]
  if (!form) return
  form.clearing = true
  form.error = ''
  try {
    await apiService.clearVolcengineAccessKey(props.accountUid, credential.credentialUid)
    credential.hasVolcengineAccessKey = false
    credential.volcengineAccessKeyIdMask = undefined
    credential.volcenginePlan = undefined
    credential.volcenginePlanTier = undefined
    credential.volcenginePlanStatus = undefined
    credential.volcenginePlanUsage = undefined
  } catch (err) {
    form.error = err instanceof Error ? err.message : String(err)
  } finally {
    form.clearing = false
  }
}

const handleInput = () => {
  apiKeyError.value = ''
  duplicateKeyIndex.value = null
}

const handleAddKey = () => {
  const trimmed = newApiKey.value.trim()
  if (!trimmed) return

  // 检查重复
  const existingIndex = props.apiKeys.indexOf(trimmed)
  if (existingIndex !== -1) {
    apiKeyError.value = t('addChannel.duplicateKey')
    duplicateKeyIndex.value = existingIndex

    // 3秒后清除高亮
    setTimeout(() => {
      duplicateKeyIndex.value = null
    }, 3000)
    return
  }

  emit('update:apiKeys', [...props.apiKeys, trimmed])
  newApiKey.value = ''
  apiKeyError.value = ''
}

const removeKey = (index: number) => {
  const updated = props.apiKeys.filter((_, i) => i !== index)
  emit('update:apiKeys', updated)
}

const moveToTop = (index: number) => {
  const updated = [...props.apiKeys]
  const [key] = updated.splice(index, 1)
  updated.unshift(key)
  emit('update:apiKeys', updated)
}

const moveToBottom = (index: number) => {
  const updated = [...props.apiKeys]
  const [key] = updated.splice(index, 1)
  updated.push(key)
  emit('update:apiKeys', updated)
}

const clearCopilotPollTimer = () => {
  if (copilotPollTimer !== null) {
    window.clearTimeout(copilotPollTimer)
    copilotPollTimer = null
  }
}

const appendOAuthKey = (accessToken: string) => {
  if (!props.apiKeys.includes(accessToken)) {
    emit('update:apiKeys', [...props.apiKeys, accessToken])
  }
}

const oauthProxyUrl = () => props.proxyUrl?.trim() || undefined

const pollCopilotAccessToken = async (intervalSeconds: number) => {
  if (!copilotDeviceCode.value) return
  copilotPolling.value = true
  try {
    const token = await apiService.pollCopilotAccessToken(copilotDeviceCode.value, oauthProxyUrl())
    if (token.accessToken) {
      appendOAuthKey(token.accessToken)
      copilotOAuthError.value = ''
      copilotOAuthSuccess.value = true
      copilotPolling.value = false
      copilotOAuthLoading.value = false
      clearCopilotPollTimer()
      return
    }
    if (token.error === 'expired_token') {
      copilotOAuthError.value = t('copilotOAuth.expired')
      copilotOAuthSuccess.value = false
      copilotPolling.value = false
      copilotOAuthLoading.value = false
      clearCopilotPollTimer()
      return
    }
    if (token.error && token.error !== 'authorization_pending') {
      copilotOAuthError.value = token.errorDescription || token.error
      copilotOAuthSuccess.value = false
      copilotPolling.value = false
      copilotOAuthLoading.value = false
      clearCopilotPollTimer()
      return
    }
  } catch (err) {
    copilotOAuthError.value = err instanceof Error ? err.message : String(err)
    copilotOAuthSuccess.value = false
    copilotPolling.value = false
    copilotOAuthLoading.value = false
    clearCopilotPollTimer()
    return
  }

  copilotPollTimer = window.setTimeout(() => {
    void pollCopilotAccessToken(intervalSeconds)
  }, Math.max(intervalSeconds, 5) * 1000)
}

const startCopilotOAuth = async () => {
  clearCopilotPollTimer()
  copilotOAuthLoading.value = true
  copilotOAuthError.value = ''
  copilotOAuthSuccess.value = false
  try {
    const device = await apiService.requestCopilotDeviceCode(oauthProxyUrl())
    copilotDeviceCode.value = device.deviceCode
    copilotUserCode.value = device.userCode
    copilotVerificationUri.value = device.verificationUri
    window.open(device.verificationUri, '_blank', 'noopener,noreferrer')
    await pollCopilotAccessToken(device.interval || 5)
  } catch (err) {
    copilotOAuthError.value = err instanceof Error ? err.message : String(err)
    copilotOAuthLoading.value = false
    copilotPolling.value = false
  }
}

onBeforeUnmount(clearCopilotPollTimer)

const cancelCopilotOAuth = () => {
  clearCopilotPollTimer()
  copilotPolling.value = false
  copilotOAuthLoading.value = false
}

const retryCopilotOAuth = () => {
  void startCopilotOAuth()
}

const diagnoseCopilotChannel = async () => {
  if (!props.channelId) return
  copilotDiagnoseLoading.value = true
  copilotDiagnoseError.value = ''
  copilotDiagnoseResult.value = null
  try {
    const latestKey = props.apiKeys[0]
    copilotDiagnoseResult.value = await apiService.diagnoseCopilotChannel(props.channelId, latestKey) as unknown as CopilotDiagnoseResponse
  } catch (err) {
    copilotDiagnoseError.value = err instanceof Error ? err.message : String(err)
  } finally {
    copilotDiagnoseLoading.value = false
  }
}

const copyKey = (key: string, index: number) => {
  navigator.clipboard.writeText(key)
  copiedKeyIndex.value = index
  setTimeout(() => {
    copiedKeyIndex.value = null
  }, 2000)
}

const getDisabledKeyLabel = (reason: string) => {
  const map: Record<string, string> = {
    'insufficient_balance': 'channelCard.blacklistReason.insufficient_balance',
    'unauthorized': 'channelCard.blacklistReason.authentication_error',
    'invalid': 'channelCard.blacklistReason.invalid',
  }
  return (map[reason] || 'channelCard.blacklistReason.unknown') as any
}
</script>

<style scoped>
.section-title {
  font-size: 1.125rem;
  font-weight: 600;
}

.animate-pulse {
  animation: pulse 1s ease-in-out 3;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

.font-weight-mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Courier New', monospace;
}

.volcengine-credential + .volcengine-credential {
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.volcengine-key-fields {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

.mimo-usage-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.volcengine-usage-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
  gap: 12px;
}

@media (max-width: 700px) {
  .volcengine-key-fields {
    grid-template-columns: minmax(0, 1fr);
  }

  .mimo-usage-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

</style>
