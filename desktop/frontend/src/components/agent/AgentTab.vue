<script setup lang="ts">
import { onMounted } from 'vue'
import AgentCard from '@/components/agent/AgentCard.vue'
import ConfigDiffDialog from '@/components/agent/ConfigDiffDialog.vue'
import MigrateSessionsDialog from '@/components/agent/MigrateSessionsDialog.vue'
import { useStatus } from '@/composables/useStatus'
import { useAgentConfig } from '@/composables/useAgentConfig'
import { useResponsesDiagnostics } from '@/composables/useResponsesDiagnostics'
import type { AgentPlatform } from '@/types'

const { actionError } = useStatus()
const {
  codexDiagnosticVisible,
  codexDiagnosticSeverity,
  codexDiagnosticSummary,
  codexDiagnosticSuggestions,
  responsesChannelDiagnosticVisible,
  responsesChannelDiagnosticSeverity,
  responsesChannelDiagnosticSummary,
  responsesChannelDiagnosticSuggestions,
  recentFailedLogsDiagnosticVisible,
  recentFailedLogsDiagnosticSeverity,
  recentFailedLogsDiagnosticSummary,
  recentFailedLogsDiagnosticSuggestions,
  codexTroubleshootingLoading,
  runCodexTroubleshooting,
} = useResponsesDiagnostics()
const {
  agentStatuses,
  configLoading,
  selectedClaudeProvider,
  claudeProviderKeys,
  savedProviderKeys,
  codexOpenAIKey,
  codexOpenAIUseOwnKey,
  openCodeOpenAIKey,
  claudeMimoBaseUrl,
  selectedMimoPlan,
  selectedDashScopePlan,
  agentLabels,
  agentPlatforms,
  claudeProviderLabel,
  claudeTargetBaseUrl,
  agentStatusText,
  agentStatusClass,
  loadAgentStatuses,
  canApplyAgent,
  selectedCodexProvider,
  codexMode,
  selectedOpenCodeProvider,
  codexProviderLabels,
  codexProviderLabel,
  codexTargetBaseUrl,
  openCodeProviderLabel,
  openCodeTargetBaseUrl,
  // Diff preview
  diffDialogOpen,
  diffResult,
  diffMode,
  diffLoading,
  diffPendingPlatform,
  diffWarning,
  showApplyPreview,
  showRestorePreview,
  confirmApply,
  confirmRestore,
  closeDiffDialog,
  migrateDialogOpen,
  migrateLoading,
  migrateResult,
  migrateError,
  codexSessionTargetProvider,
  showMigrateDialog,
  confirmMigrate,
  closeMigrateDialog,
} = useAgentConfig()

onMounted(() => {
  loadAgentStatuses()
})

const handleApply = async (platform: AgentPlatform) => {
  actionError.value = ''
  try {
    await showApplyPreview(platform)
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : String(error)
  }
}

const handleRestore = async (platform: AgentPlatform) => {
  actionError.value = ''
  try {
    await showRestorePreview(platform)
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : String(error)
  }
}

const handleMigrate = () => {
  actionError.value = ''
  showMigrateDialog()
}

const handleTroubleshoot = async () => {
  actionError.value = ''
  try {
    await runCodexTroubleshooting()
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : String(error)
  }
}

const handleConfirm = async () => {
  actionError.value = ''
  try {
    if (diffMode.value === 'apply') {
      await confirmApply()
    } else {
      await confirmRestore()
    }
    await loadAgentStatuses()
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : String(error)
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <AgentCard
        v-for="platform in agentPlatforms"
        :key="platform"
        :platform="platform"
        :agent-status="agentStatuses[platform]"
        :config-loading="configLoading"
        :agent-label="agentLabels[platform]"
        :agent-status-text="agentStatusText(agentStatuses[platform])"
        :agent-status-class="agentStatusClass(agentStatuses[platform])"
        :can-apply="canApplyAgent(platform)"
        :selected-claude-provider="selectedClaudeProvider"
        :claude-provider-keys="claudeProviderKeys"
        :saved-provider-keys="savedProviderKeys"
        :claude-mimo-base-url="claudeMimoBaseUrl"
        :selected-mimo-plan="selectedMimoPlan"
        :selected-dash-scope-plan="selectedDashScopePlan"
        :claude-provider-label="claudeProviderLabel"
        :claude-target-base-url="claudeTargetBaseUrl"
        :selected-codex-provider="selectedCodexProvider"
        :codex-mode="codexMode"
        :codex-open-a-i-key="codexOpenAIKey"
        :codex-open-a-i-use-own-key="codexOpenAIUseOwnKey"
        :codex-provider-labels="codexProviderLabels"
        :codex-provider-label="codexProviderLabel"
        :codex-target-base-url="codexTargetBaseUrl"
        :selected-open-code-provider="selectedOpenCodeProvider"
        :open-code-open-a-i-key="openCodeOpenAIKey"
        :open-code-provider-labels="codexProviderLabels"
        :open-code-provider-label="openCodeProviderLabel"
        :open-code-target-base-url="openCodeTargetBaseUrl"
        :migrate-loading="migrateLoading"
        :codex-diagnostic-visible="platform === 'codex' ? codexDiagnosticVisible : false"
        :codex-diagnostic-severity="platform === 'codex' ? codexDiagnosticSeverity : 'ok'"
        :codex-diagnostic-summary="platform === 'codex' ? codexDiagnosticSummary : ''"
        :codex-diagnostic-suggestions="platform === 'codex' ? codexDiagnosticSuggestions : []"
        :responses-channel-diagnostic-visible="platform === 'codex' ? responsesChannelDiagnosticVisible : false"
        :responses-channel-diagnostic-severity="platform === 'codex' ? responsesChannelDiagnosticSeverity : 'ok'"
        :responses-channel-diagnostic-summary="platform === 'codex' ? responsesChannelDiagnosticSummary : ''"
        :responses-channel-diagnostic-suggestions="platform === 'codex' ? responsesChannelDiagnosticSuggestions : []"
        :recent-failed-logs-diagnostic-visible="platform === 'codex' ? recentFailedLogsDiagnosticVisible : false"
        :recent-failed-logs-diagnostic-severity="platform === 'codex' ? recentFailedLogsDiagnosticSeverity : 'ok'"
        :recent-failed-logs-diagnostic-summary="platform === 'codex' ? recentFailedLogsDiagnosticSummary : ''"
        :recent-failed-logs-diagnostic-suggestions="platform === 'codex' ? recentFailedLogsDiagnosticSuggestions : []"
        :codex-troubleshooting-loading="platform === 'codex' ? codexTroubleshootingLoading : false"
        @apply="handleApply(platform)"
        @restore="handleRestore(platform)"
        @migrate="handleMigrate"
        @troubleshoot="handleTroubleshoot"
        @update:selected-claude-provider="selectedClaudeProvider = $event"
        @update:claude-provider-keys="claudeProviderKeys = $event"
        @update:claude-mimo-base-url="claudeMimoBaseUrl = $event"
        @update:selected-mimo-plan="selectedMimoPlan = $event"
        @update:selected-dash-scope-plan="selectedDashScopePlan = $event"
        @update:selected-codex-provider="selectedCodexProvider = $event"
        @update:codex-mode="codexMode = $event"
        @update:codex-open-a-i-key="codexOpenAIKey = $event"
        @update:codex-open-a-i-use-own-key="codexOpenAIUseOwnKey = $event"
        @update:selected-open-code-provider="selectedOpenCodeProvider = $event"
        @update:open-code-open-a-i-key="openCodeOpenAIKey = $event"
      />
    </div>
    <p v-if="actionError" class="text-sm text-destructive-foreground">{{ actionError }}</p>

    <ConfigDiffDialog
      :open="diffDialogOpen"
      :mode="diffMode"
      :platform="diffPendingPlatform"
      :result="diffResult"
      :loading="diffLoading"
      :warning="diffWarning"
      @confirm="handleConfirm"
      @cancel="closeDiffDialog"
    />

    <MigrateSessionsDialog
      :open="migrateDialogOpen"
      :loading="migrateLoading"
      :target-provider="codexSessionTargetProvider"
      :result="migrateResult"
      :error="migrateError"
      @confirm="confirmMigrate"
      @cancel="closeMigrateDialog"
    />
  </div>
</template>
