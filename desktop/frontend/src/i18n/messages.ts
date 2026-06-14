export type SupportedLocale = 'en' | 'zh-CN'

export const defaultLocale: SupportedLocale = 'en'

export const languageOptions: { locale: SupportedLocale; label: string }[] = [
  { locale: 'en', label: 'English' },
  { locale: 'zh-CN', label: '中文' },
]

export type MessageKey =
  | 'common.gatewayLabel'
  | 'common.online'
  | 'common.connecting'
  | 'common.offline'
  | 'common.refreshWebUI'
  | 'common.version'
  | 'common.gatewayPort'
  | 'common.daemonPid'
  | 'common.autoStart'
  | 'common.autoStartOn'
  | 'common.autoStartOff'
  | 'common.serviceHealthy'
  | 'common.serviceStarting'
  | 'common.serviceDisconnected'
  | 'common.settings'
  | 'common.save'
  | 'common.cancel'
  | 'common.retry'
  | 'nav.status'
  | 'nav.statusDesc'
  | 'nav.agent'
  | 'nav.agentDesc'
  | 'nav.channels'
  | 'nav.channelsDesc'
  | 'nav.env'
  | 'nav.envDesc'
  | 'nav.dashboard'
  | 'nav.dashboardDesc'
  | 'tab.statusTitle'
  | 'tab.agentTitle'
  | 'tab.channelsTitle'
  | 'tab.envTitle'
  | 'tab.dashboardTitle'
  | 'sidebar.language'
  | 'sidebar.languageEnglish'
  | 'sidebar.languageChinese'
  | 'setup.loading'
  | 'setup.title'
  | 'setup.description'
  | 'setup.regenerate'
  | 'setup.regenerateTitle'
  | 'setup.hide'
  | 'setup.show'
  | 'setup.copied'
  | 'setup.copyKey'
  | 'setup.configPath'
  | 'setup.copyPath'
  | 'setup.hint'
  | 'setup.saving'
  | 'setup.submit'
  | 'webui.notRunning'
  | 'webui.openInBrowser'
  | 'metrics.gatewayPort'
  | 'metrics.uptime'
  | 'metrics.channels'
  | 'metrics.version'
  | 'actions.start'
  | 'actions.stop'
  | 'actions.restart'
  | 'actions.openWebUI'
  | 'actions.openBrowser'
  | 'actions.refreshStatus'
  | 'details.title'
  | 'details.binary'
  | 'details.binaryMissing'
  | 'details.dataDir'
  | 'details.dataDirMissing'
  | 'details.healthStatus'
  | 'details.revealDir'
  | 'details.openDir'
  | 'channel.headerEyebrow'
  | 'channel.title'
  | 'channel.description'
  | 'channel.hasKey'
  | 'channel.promo'
  | 'channel.console'
  | 'channel.target'
  | 'channel.planLabel'
  | 'channel.apiKeyLabel'
  | 'channel.baseUrlPrefix'
  | 'channel.keySavedPlaceholder'
  | 'channel.keyInputPlaceholder'
  | 'channel.missingKey'
  | 'channel.reuseKey'
  | 'channel.name'
  | 'channel.nameHint'
  | 'channel.presetWrites'
  | 'channel.capabilityHint'
  | 'channel.addToCCX'
  | 'channel.badgeDirectAgent'
  | 'channel.badgeNativeMessages'
  // Provider labels & descriptions
  | 'channel.preset.deepseek.label'
  | 'channel.preset.deepseek.description'
  | 'channel.preset.mimo.label'
  | 'channel.preset.mimo.description'
  | 'channel.preset.compshare.label'
  | 'channel.preset.compshare.description'
  | 'channel.preset.runapi.label'
  | 'channel.preset.runapi.description'
  | 'channel.preset.kimi.label'
  | 'channel.preset.kimi.description'
  | 'channel.preset.glm.label'
  | 'channel.preset.glm.description'
  | 'channel.preset.minimax.label'
  | 'channel.preset.minimax.description'
  | 'channel.preset.dashscope.label'
  | 'channel.preset.dashscope.description'
  | 'channel.preset.opencode-zen.label'
  | 'channel.preset.opencode-zen.description'
  | 'channel.preset.opencode-go.label'
  | 'channel.preset.opencode-go.description'
  // DeepSeek plans
  | 'channel.preset.deepseek.plan.anthropic.label'
  | 'channel.preset.deepseek.plan.anthropic.description'
  | 'channel.preset.deepseek.plan.openai-chat.label'
  | 'channel.preset.deepseek.plan.openai-chat.description'
  // MiMo plans
  | 'channel.preset.mimo.plan.anthropic.label'
  | 'channel.preset.mimo.plan.anthropic.description'
  | 'channel.preset.mimo.plan.openai-chat.label'
  | 'channel.preset.mimo.plan.openai-chat.description'
  | 'channel.preset.mimo.plan.token-cn.label'
  | 'channel.preset.mimo.plan.token-cn.description'
  | 'channel.preset.mimo.plan.token-sgp.label'
  | 'channel.preset.mimo.plan.token-sgp.description'
  | 'channel.preset.mimo.plan.token-ams.label'
  | 'channel.preset.mimo.plan.token-ams.description'
  | 'channel.preset.mimo.plan.token-cn-anthropic.label'
  | 'channel.preset.mimo.plan.token-cn-anthropic.description'
  | 'channel.preset.mimo.plan.token-sgp-anthropic.label'
  | 'channel.preset.mimo.plan.token-sgp-anthropic.description'
  | 'channel.preset.mimo.plan.token-ams-anthropic.label'
  | 'channel.preset.mimo.plan.token-ams-anthropic.description'
  // Compshare plans
  | 'channel.preset.compshare.plan.anthropic.label'
  | 'channel.preset.compshare.plan.anthropic.description'
  | 'channel.preset.compshare.plan.openai-chat.label'
  | 'channel.preset.compshare.plan.openai-chat.description'
  // RunAPI plans
  | 'channel.preset.runapi.plan.anthropic.label'
  | 'channel.preset.runapi.plan.anthropic.description'
  | 'channel.preset.runapi.plan.openai-chat.label'
  | 'channel.preset.runapi.plan.openai-chat.description'
  | 'channel.preset.tencent-lkeap.plan.anthropic.label'
  | 'channel.preset.tencent-lkeap.plan.anthropic.description'
  | 'channel.preset.tencent-lkeap.plan.openai-chat.label'
  | 'channel.preset.tencent-lkeap.plan.openai-chat.description'
  | 'channel.preset.kimi-code.plan.anthropic.label'
  | 'channel.preset.kimi-code.plan.anthropic.description'
  | 'channel.preset.kimi-code.plan.openai-chat.label'
  | 'channel.preset.kimi-code.plan.openai-chat.description'
  | 'channel.preset.volc-ark.plan.anthropic.label'
  | 'channel.preset.volc-ark.plan.anthropic.description'
  | 'channel.preset.volc-ark.plan.openai-chat.label'
  | 'channel.preset.volc-ark.plan.openai-chat.description'
  | 'channel.preset.qianfan.plan.anthropic.label'
  | 'channel.preset.qianfan.plan.anthropic.description'
  | 'channel.preset.qianfan.plan.openai-chat.label'
  | 'channel.preset.qianfan.plan.openai-chat.description'
  | 'channel.preset.originrouter.plan.anthropic.label'
  | 'channel.preset.originrouter.plan.anthropic.description'
  | 'channel.preset.originrouter.plan.openai-chat.label'
  | 'channel.preset.originrouter.plan.openai-chat.description'
  // Kimi plans
  | 'channel.preset.kimi.plan.anthropic.label'
  | 'channel.preset.kimi.plan.anthropic.description'
  | 'channel.preset.kimi.plan.openai-chat.label'
  | 'channel.preset.kimi.plan.openai-chat.description'
  | 'channel.preset.kimi.plan.coding-anthropic.label'
  | 'channel.preset.kimi.plan.coding-anthropic.description'
  | 'channel.preset.kimi.plan.coding-openai-chat.label'
  | 'channel.preset.kimi.plan.coding-openai-chat.description'
  // GLM plans
  | 'channel.preset.glm.plan.anthropic.label'
  | 'channel.preset.glm.plan.anthropic.description'
  | 'channel.preset.glm.plan.coding.label'
  | 'channel.preset.glm.plan.coding.description'
  | 'channel.preset.glm.plan.openai-chat.label'
  | 'channel.preset.glm.plan.openai-chat.description'
  // MiniMax plans
  | 'channel.preset.minimax.plan.anthropic.label'
  | 'channel.preset.minimax.plan.anthropic.description'
  | 'channel.preset.minimax.plan.openai-chat.label'
  | 'channel.preset.minimax.plan.openai-chat.description'
  // DashScope plans
  | 'channel.preset.dashscope.plan.anthropic.label'
  | 'channel.preset.dashscope.plan.anthropic.description'
  | 'channel.preset.dashscope.plan.openai-chat.label'
  | 'channel.preset.dashscope.plan.openai-chat.description'
  | 'channel.preset.dashscope.plan.coding-anthropic.label'
  | 'channel.preset.dashscope.plan.coding-anthropic.description'
  | 'channel.preset.dashscope.plan.coding-openai-chat.label'
  | 'channel.preset.dashscope.plan.coding-openai-chat.description'
  | 'channel.preset.dashscope.plan.token-plan-anthropic.label'
  | 'channel.preset.dashscope.plan.token-plan-anthropic.description'
  | 'channel.preset.dashscope.plan.token-plan-openai-chat.label'
  | 'channel.preset.dashscope.plan.token-plan-openai-chat.description'
  // OpenCode Zen plans
  | 'channel.preset.opencode-zen.plan.anthropic.label'
  | 'channel.preset.opencode-zen.plan.anthropic.description'
  | 'channel.preset.opencode-zen.plan.openai-chat.label'
  | 'channel.preset.opencode-zen.plan.openai-chat.description'
  // OpenCode Go plans
  | 'channel.preset.opencode-go.plan.anthropic.label'
  | 'channel.preset.opencode-go.plan.anthropic.description'
  | 'channel.preset.opencode-go.plan.openai-chat.label'
  | 'channel.preset.opencode-go.plan.openai-chat.description'
  // ModelScope
  | 'channel.preset.modelscope.label'
  | 'channel.preset.modelscope.description'
  | 'channel.preset.modelscope.plan.anthropic.label'
  | 'channel.preset.modelscope.plan.anthropic.description'
  | 'channel.preset.modelscope.plan.openai-chat.label'
  | 'channel.preset.modelscope.plan.openai-chat.description'
  // OpenRouter
  | 'channel.preset.openrouter.label'
  | 'channel.preset.openrouter.description'
  | 'channel.preset.openrouter.plan.anthropic.label'
  | 'channel.preset.openrouter.plan.anthropic.description'
  | 'channel.preset.openrouter.plan.openai-chat.label'
  | 'channel.preset.openrouter.plan.openai-chat.description'
  // Targets (shared across providers)
  | 'channel.target.messages.label'
  | 'channel.target.messages.description'
  | 'channel.target.responses.label'
  | 'channel.target.responses.description'
  | 'channel.target.chat.label'
  | 'channel.target.chat.description'
  | 'env.title'
  | 'env.pathDetecting'
  | 'env.refresh'
  | 'env.openingEditor'
  | 'env.openWithEditor'
  | 'env.openInEditor'
  | 'env.selectEditor'
  | 'env.save'
  | 'env.saving'
  | 'env.saved'
  | 'env.failed'
  | 'env.hide'
  | 'env.show'
  | 'env.copied'
  | 'env.copy'
  | 'env.fieldRequired'
  | 'env.fieldDisallow'
  | 'env.fieldNumber'
  | 'env.fieldInteger'
  | 'env.fieldMin'
  | 'env.fieldMax'
  | 'env.loadFailed'
  | 'env.saveSuccess'
  | 'agent.statusDetecting'
  | 'agent.statusConfigured'
  | 'agent.statusPortMismatch'
  | 'agent.statusUnconfigured'
  | 'agent.localGateway'
  | 'agent.custom'
  | 'agent.currentProvider'
  | 'agent.currentUrl'
  | 'agent.targetUrl'
  | 'agent.notSet'
  | 'agent.configPath'
  | 'agent.authPath'
  | 'agent.openFileInEditor'
  | 'agent.applyConfig'
  | 'agent.restoreConfig'
  | 'agent.migrateSessions'
  | 'agent.migrateConfirmTitle'
  | 'agent.migrateConfirmDesc'
  | 'agent.migrateTarget'
  | 'agent.migrateRunning'
  | 'agent.migrateConfirm'
  | 'agent.migrateClose'
  | 'agent.migrateResultSuccess'
  | 'agent.migrateResultSkippedLabel'
  | 'agent.migrateResultFailedLabel'
  | 'agent.migrateSQLiteUpdated'
  | 'agent.migrateSQLiteSkipped'
  | 'agent.openConsole'
  | 'agent.codexPlaceholderSaved'
  | 'agent.codexPlaceholderRequired'
  | 'agent.codexPlaceholderWriteOnly'
  | 'agent.diffPreviewApply'
  | 'agent.diffPreviewRestore'
  | 'agent.diffConfirmApply'
  | 'agent.diffConfirmRestore'
  | 'agent.diffActionCreate'
  | 'agent.diffActionDelete'
  | 'agent.diffActionModify'
  | 'agent.diffComputing'
  | 'agent.diffNoChanges'
  | 'agent.diffExpandContext'
  | 'agent.diffCollapseContext'
  | 'agent.diffCancel'
  | 'agent.codexMode'
  | 'agent.codexQuickMode'
  | 'agent.codexPluginMode'
  | 'agent.codexQuickModeHint'
  | 'agent.codexPluginModeHint'
  | 'agent.sessionMigrationWarning'
  | 'agent.provider.localGateway'
  | 'agent.provider.deepseekDirect'
  | 'agent.provider.mimoDirect'
  | 'agent.provider.compshareDirect'
  | 'agent.provider.runapiDirect'
  | 'agent.provider.kimiDirect'
  | 'agent.provider.glmDirect'
  | 'agent.provider.minimaxDirect'
  | 'agent.provider.dashscopeDirect'
  | 'agent.provider.opencodeZenDirect'
  | 'agent.provider.opencodeGoDirect'
  | 'agent.provider.openaiDirect'
  | 'agent.provider.openrouterDirect'
  | 'agent.provider.modelscopeDirect'
  | 'agent.provider.xfyunDirect'
  | 'agent.provider.tencentLkeapDirect'
  | 'agent.provider.volcArkDirect'
  | 'agent.provider.qianfanDirect'
  | 'agent.hasOwnApiKey'
  | 'agent.promo'
  | 'agent.planPayAsYouGo'
  | 'agent.planChina'
  | 'agent.planSingapore'
  | 'agent.planEurope'
  | 'agent.planSubscription'
  | 'agent.billingModeMiMo'
  | 'agent.billingModeDashScope'
  | 'agent.placeholderSaved'
  | 'agent.placeholderMimo'
  | 'agent.placeholderDashScope'
  | 'agent.placeholderRequired'
  | 'agent.codexDiagnosticTitle'
  | 'agent.codexTroubleshoot'
  | 'agent.codexTroubleshooting'
  | 'agent.codexDiagnosticHealthy'
  | 'agent.codexDiagnosticMismatch'
  | 'agent.codexDiagnosticCheck'
  | 'agent.codexDiagnosticAuthMode'
  | 'agent.codexDiagnosticConfigPath'
  | 'agent.codexDiagnosticAuthPath'
  | 'agent.codexDiagnosticSuggestedApply'
  | 'agent.codexDiagnosticSuggestedRestore'
  | 'agent.codexDiagnosticStatusTitle'
  | 'agent.codexDiagnosticStatusHint'
  | 'agent.codexDiagnosticCodeMissingApiKey'
  | 'agent.codexDiagnosticCodeProxyKeyMismatch'
  | 'agent.codexDiagnosticCodeAuthModeMismatch'
  | 'agent.codexDiagnosticCodePluginMissingBearer'
  | 'agent.codexDiagnosticCodeLegacyIncomplete'
  | 'agent.codexDiagnosticCodeConfigPolluted'
  | 'agent.codexDiagnosticCodeAuthUnreadable'
  | 'agent.codexDiagnosticCodeModeMismatch'
  | 'agent.responsesDiagnosticTitle'
  | 'agent.responsesDiagnosticHealthy'
  | 'agent.responsesDiagnosticNoChannel'
  | 'agent.responsesDiagnosticAllDisabled'
  | 'agent.responsesDiagnosticNoApiKeys'
  | 'agent.responsesDiagnosticProtocolMismatch'
  | 'agent.responsesDiagnosticCircuitOpen'
  | 'agent.responsesDiagnosticSuggestedRefresh'
  | 'agent.responsesDiagnosticSuggestedCreate'
  | 'agent.responsesDiagnosticSuggestedEnable'
  | 'agent.responsesDiagnosticSuggestedAddKey'
  | 'agent.responsesDiagnosticSuggestedProtocolMismatch'
  | 'agent.responsesDiagnosticSuggestedResume'
  | 'agent.responsesLogsDiagnosticTitle'
  | 'agent.responsesLogsDiagnosticHealthy'
  | 'agent.responsesLogsDiagnosticAuth'
  | 'agent.responsesLogsDiagnosticRateLimit'
  | 'agent.responsesLogsDiagnosticUpstream5xx'
  | 'agent.responsesLogsDiagnosticTimeout'
  | 'agent.responsesLogsDiagnosticGeneric'
  | 'agent.responsesLogsDiagnosticSuggestedRefresh'
  | 'agent.responsesLogsDiagnosticSuggestedAuth'
  | 'agent.responsesLogsDiagnosticSuggestedRateLimit'
  | 'agent.responsesLogsDiagnosticSuggestedUpstream5xx'
  | 'agent.responsesLogsDiagnosticSuggestedTimeout'
  | 'agent.responsesLogsDiagnosticSuggestedGeneric'
  | 'agent.codexDiagnosticIntro'
  | 'agent.codexDiagnosticLayerConfig'
  | 'agent.codexDiagnosticLayerChannels'
  | 'agent.codexDiagnosticLayerLogs'
  | 'env.groupAccess'
  | 'env.groupAccessDesc'
  | 'env.fieldProxyAccessKey'
  | 'env.placeholderProxyAccessKey'
  | 'env.fieldAdminAccessKey'
  | 'env.placeholderAdminAccessKey'
  | 'env.descAdminAccessKey'
  | 'env.groupServer'
  | 'env.groupServerDesc'
  | 'env.fieldPort'
  | 'env.descPort'
  | 'env.fieldEnv'
  | 'env.descEnv'
  | 'env.groupWebUI'
  | 'env.groupWebUIDesc'
  | 'env.fieldEnableWebUI'
  | 'env.descEnableWebUI'
  | 'env.fieldAppUILanguage'
  | 'env.groupLogs'
  | 'env.groupLogsDesc'
  | 'env.fieldLogLevel'
  | 'env.fieldEnableRequestLogs'
  | 'env.fieldEnableResponseLogs'
  | 'env.descEnableResponseLogs'
  | 'env.fieldQuietPollingLogs'
  | 'env.fieldRawLogOutput'
  | 'env.fieldSseDebugLevel'
  | 'env.fieldRewriteResponseModel'
  | 'env.groupPerformance'
  | 'env.groupPerformanceDesc'
  | 'env.fieldRequestTimeout'
  | 'env.fieldServerReadTimeout'
  | 'env.fieldMaxRequestBodySize'
  | 'env.fieldResponseHeaderTimeout'
  | 'env.groupCors'
  | 'env.groupCorsDesc'
  | 'env.fieldEnableCors'
  | 'env.fieldCorsOrigin'
  | 'env.groupCircuitBreaker'
  | 'env.groupCircuitBreakerDesc'
  | 'env.fieldMetricsWindowSize'
  | 'env.fieldMetricsFailureThreshold'
  | 'env.runtimeCbTitle'
  | 'env.runtimeCbDesc'
  | 'env.runtimeCbWindowSize'
  | 'env.runtimeCbWindowSizeDesc'
  | 'env.runtimeCbFailureThreshold'
  | 'env.runtimeCbFailureThresholdDesc'
  | 'env.runtimeCbConsecutiveFailures'
  | 'env.runtimeCbConsecutiveFailuresDesc'
  | 'env.runtimeCbStreamFirstContentTimeout'
  | 'env.runtimeCbStreamFirstContentTimeoutDesc'
  | 'env.runtimeCbStreamInactivityTimeout'
  | 'env.runtimeCbStreamInactivityTimeoutDesc'
  | 'env.runtimeCbStreamToolCallIdleTimeout'
  | 'env.runtimeCbStreamToolCallIdleTimeoutDesc'
  | 'env.runtimeCbPresetGentle'
  | 'env.runtimeCbPresetBalanced'
  | 'env.runtimeCbPresetAggressive'
  | 'env.runtimeCbPresetCustom'
  | 'env.runtimeCbSaved'
  | 'env.runtimeCbSaveFailed'
  | 'env.runtimeCbLoadFailed'
  | 'env.runtimeCbNoBackend'
  | 'env.runtimeCbServiceStopped'
  | 'env.historicalImageTurnLimitTitle'
  | 'env.historicalImageTurnLimitLabel'
  | 'env.historicalImageTurnLimitHint'
  | 'env.historicalImageTurnLimitSaved'
  | 'env.groupMetricsPersistence'
  | 'env.groupMetricsPersistenceDesc'
  | 'env.fieldMetricsPersistenceEnabled'
  | 'env.fieldMetricsRetentionDays'
  | 'logs.searchPlaceholder'
  | 'logs.autoScroll'
  | 'logs.copied'
  | 'logs.copyAll'
  | 'logs.clear'
  | 'logs.noSearchResults'
  | 'logs.empty'
  | 'diagnostic.binaryTitle'
  | 'diagnostic.binarySuggestionBuild'
  | 'diagnostic.binarySuggestionCheckDataDir'
  | 'diagnostic.binarySuggestionDownload'
  | 'diagnostic.portTitle'
  | 'diagnostic.portSuggestionInstance'
  | 'diagnostic.portSuggestionEnv'
  | 'diagnostic.portSuggestionInspect'
  | 'diagnostic.healthTitle'
  | 'diagnostic.healthSuggestionLogs'
  | 'diagnostic.healthSuggestionEnv'
  | 'diagnostic.healthSuggestionChannels'
  | 'diagnostic.healthSuggestionRestart'
  | 'diagnostic.permissionTitle'
  | 'diagnostic.permissionSuggestionDataDir'
  | 'diagnostic.permissionSuggestionExecutable'
  | 'diagnostic.permissionSuggestionWindows'
  | 'diagnostic.genericTitle'
  | 'diagnostic.genericSuggestionLogs'
  | 'diagnostic.genericSuggestionRestart'
  | 'setup.errorEmptyKey'
  | 'env.saveSuccessHint'
  | 'env.openedInEditor'
  | 'sidebar.versionHintStore'
  | 'sidebar.versionHintTray'
  | 'sidebar.versionClickCheck'
  | 'sidebar.updateAvailable'
  | 'sidebar.updateAvailableHint'
  | 'sidebar.theme'
  | 'sidebar.themeAuto'
  | 'sidebar.themeLight'
  | 'sidebar.themeDark'
  | 'console.channelsTab'
  | 'console.conversationsTab'
  | 'console.nativeChannelsEyebrow'
  | 'console.nativeChannelsTitle'
  | 'console.quickAdd'
  | 'console.quickAddEyebrow'
  | 'console.quickAddTitle'
  | 'console.searchChannels'
  | 'console.actions.refresh'
  | 'console.pingAll'
  | 'console.addChannel'
  | 'console.noChannels'
  | 'console.noSearchResults'
  | 'console.mode.multi'
  | 'console.mode.single'
  | 'console.pool.active'
  | 'console.pool.current'
  | 'console.pool.inactive'
  | 'console.channelStatus.active'
  | 'console.channelStatus.suspended'
  | 'console.channelStatus.disabled'
  | 'console.channel.cacheWriteHigh'
  | 'console.channel.cacheWriteHighHint'
  | 'console.circuit.open'
  | 'console.circuit.halfOpen'
  | 'console.keys.active'
  | 'console.actions.label'
  | 'console.actions.edit'
  | 'console.actions.capability'
  | 'console.actions.logs'
  | 'console.actions.copy'
  | 'console.actions.website'
  | 'console.actions.ping'
  | 'console.actions.enable'
  | 'console.actions.suspend'
  | 'console.actions.resume'
  | 'console.actions.promote'
  | 'console.actions.disable'
  | 'console.actions.delete'
  | 'console.actions.resetCircuit'
  | 'console.form.addChannel'
  | 'console.form.editChannel'
  | 'console.form.basicInfo'
  | 'console.form.name'
  | 'console.form.nameRequired'
  | 'console.form.serviceType'
  | 'console.form.selectServiceType'
  | 'console.form.serviceTypeRequired'
  | 'console.form.description'
  | 'console.form.connection'
  | 'console.form.baseUrl'
  | 'console.form.baseUrlRequired'
  | 'console.form.additionalUrls'
  | 'console.form.proxyUrl'
  | 'console.form.routePrefix'
  | 'console.form.insecureSkipVerify'
  | 'console.form.authentication'
  | 'console.form.apiKeys'
  | 'console.form.disabledKeys'
  | 'console.form.restoreKey'
  | 'console.form.historicalKeys'
  | 'addChannel.oneClickSetup'
  | 'addChannel.expectedRequest'
  | 'console.form.apiKeyRequired'
  | 'console.form.reasoningEffort'
  | 'console.form.noVisionModels'
  | 'console.form.visionFallbackModel'
  | 'console.form.fetchModels'
  | 'console.form.fetchingModels'
  | 'console.form.modelFetchNeedsConfig'
  | 'console.form.visionEnabled'
  | 'console.form.visionDisabled'
  | 'console.form.models'
  | 'console.form.modelMapping'
  | 'console.form.supportedModels'
  | 'console.form.protocolOptions'
  | 'console.form.reasoningParamStyle'
  | 'console.form.textVerbosity'
  | 'console.form.advancedFlags'
  | 'console.form.noVision'
  | 'console.form.passbackReasoning'
  | 'console.form.passbackThinking'
  | 'console.form.fastMode'
  | 'console.form.lowQuality'
  | 'console.form.injectDummySignature'
  | 'console.form.stripThoughtSignature'
  | 'console.form.stripEmptyBlocks'
  | 'console.form.normalizeSystem'
  | 'console.form.normalizeUserId'
  | 'console.form.stripBillingHeader'
  | 'console.form.normalizeChatRoles'
  | 'console.form.autoBlacklist'
  | 'console.form.codexNativeTools'
  | 'console.form.codexCompat'
  | 'console.form.stripCodexTools'
  | 'console.form.customHeaders'
  | 'console.form.requestTimeoutMs'
  | 'console.form.requestTimeoutInvalid'
  | 'console.form.streamTimeouts'
  | 'console.form.streamFirstContentTimeoutOverrideLabel'
  | 'console.form.streamFirstContentTimeoutLabel'
  | 'console.form.streamInactivityTimeoutOverrideLabel'
  | 'console.form.streamInactivityTimeoutLabel'
  | 'console.form.streamToolCallIdleTimeoutOverrideLabel'
  | 'console.form.streamToolCallIdleTimeoutLabel'
  | 'console.form.streamTimeoutOverrideHint'
  | 'console.form.streamTimeoutInheritHint'
  | 'console.form.cancel'
  | 'console.form.create'
  | 'console.form.save'
  | 'console.form.website'
  | 'console.form.baseUrlPlaceholder'
  | 'console.form.sectionBasic'
  | 'console.form.sectionAuth'
  | 'console.form.sectionRedirect'
  | 'console.form.sectionAdvanced'
  | 'console.form.sectionHeaders'
  | 'console.form.outline'
  | 'console.form.compatibilityTitle'
  | 'console.form.runtimeTitle'
  | 'console.form.transportTitle'
  | 'console.form.rateLimitTitle'
  | 'console.form.streamTimeoutTitle'
  | 'console.form.fastModeHint'
  | 'console.form.textVerbosityStyle'
  | 'console.form.visionTitle'
  | 'console.form.selectDefaultLabel'
  | 'console.form.autoBlacklistBalanceLabel'
  | 'console.form.autoBlacklistBalanceHint'
  | 'console.form.rateLimitAutoLabel'
  | 'console.form.rateLimitAutoHint'
  | 'console.form.proxyUrlLabel'
  | 'console.form.requestTimeoutLabel'
  | 'console.form.routePrefixLabel'
  | 'console.form.rpmLabel'
  | 'console.form.rpmPlaceholder'
  | 'console.form.windowLabel'
  | 'console.form.windowPlaceholder'
  | 'console.form.maxConcurrentLabel'
  | 'console.form.maxConcurrentPlaceholder'
  | 'console.form.presetGentle'
  | 'console.form.presetBalanced'
  | 'console.form.presetAggressive'
  | 'console.form.firstByteWait'
  | 'console.form.firstByteWaitHint'
  | 'console.form.idleAfterFirstByte'
  | 'console.form.idleAfterFirstByteHint'
  | 'console.form.toolCallIdle'
  | 'console.form.toolCallIdleHint'
  | 'console.form.timeoutThreshold'
  | 'console.form.supportedModelsLabel'
  | 'console.form.basicInfoTitle'
  | 'console.form.connectionTitle'
  | 'console.form.multiLineFailover'
  | 'console.form.expectedEndpoint'
  | 'console.form.descriptionPlaceholder'
  | 'console.form.mappingHint'
  | 'console.form.visionFallbackHint'
  | 'console.form.modelRedirect'
  | 'console.form.modelMappingExisting'
  | 'console.form.modelMappingAdd'
  | 'console.form.generationParams'
  | 'console.form.textVerbosityPlaceholder'
  | 'console.form.modelScope'
  | 'console.form.selectDefault'
  | 'console.form.noVisionHint'
  | 'console.form.historicalImageTurnLimit'
  | 'console.form.historicalImageTurnLimitHint'
  | 'console.form.compactModel'
  | 'console.form.compactModelHint'
  | 'console.form.passbackReasoningHint'
  | 'console.form.passbackThinkingHint'
  | 'console.form.stripThoughtSignatureHint'
  | 'console.form.injectDummySignatureHint'
  | 'console.form.reasoningParamStyleHint'
  | 'console.form.codexNativeToolsHint'
  | 'console.form.codexCompatHint'
  | 'console.form.stripEmptyBlocksHint'
  | 'console.form.normalizeSystemHint'
  | 'console.form.normalizeUserIdHint'
  | 'console.form.stripBillingHeaderHint'
  | 'console.form.normalizeChatRolesHint'
  | 'console.form.lowQualityHint'
  | 'console.form.autoBlacklistHint'
  | 'console.form.insecureSkipVerifyHint'
  | 'console.form.proxyUrlHint'
  | 'console.form.routePrefixHint'
  | 'console.form.requestTimeoutMsHint'
  | 'console.form.rateLimitSectionLabel'
  | 'console.form.rateLimitSectionHint'
  | 'console.form.rateLimitRpmLabel'
  | 'console.form.rateLimitRpmHint'
  | 'console.form.rateLimitBurstLabel'
  | 'console.form.rateLimitBurstHint'
  | 'console.form.rateLimitMaxConcurrentLabel'
  | 'console.form.rateLimitMaxConcurrentHint'
  | 'console.form.rateLimitAutoFromHeadersLabel'
  | 'console.form.rateLimitAutoFromHeadersHint'
  | 'addChannel.duplicateKey'
  | 'addChannel.addNewApiKeyPlaceholder'
  | 'addChannel.quickMode'
  | 'addChannel.quickHint'
  | 'addChannel.applyDetected'
  | 'addChannel.noneDetected'
  | 'addChannel.fullSubtitle'
  | 'addChannel.editSubtitle'
  | 'orchestration.moveTop'
  | 'orchestration.moveBottom'
  | 'orchestration.keepOne'
  | 'orchestration.deleteActiveGuard'
  | 'console.keys.disabled'
  | 'console.conversations.search'
  | 'console.conversations.allKinds'
  | 'console.conversations.refresh'
  | 'console.conversations.total'
  | 'console.conversations.empty'
  | 'console.conversations.noSearchResults'
  | 'console.conversations.overrideTitle'
  | 'console.conversations.overrideHint'
  | 'console.conversations.overrideSequence'
  | 'console.conversations.overrideRequired'
  | 'console.conversations.saveOverride'
  | 'console.conversations.noChannelsForKind'
  | 'console.conversations.active'
  | 'console.conversations.override'
  | 'console.conversations.duration.default'
  | 'console.conversations.duration.15min'
  | 'console.conversations.duration.1hour'
  | 'console.conversations.duration.2hours'
  | 'console.conversations.duration.4hours'
  | 'console.conversations.duration.8hours'
  | 'console.conversations.duration.12hours'
  | 'console.conversations.duration.24hours'
  | 'console.conversations.duration.never'
  | 'console.capability.title'
  | 'console.capability.start'
  | 'console.capability.cancel'
  | 'console.capability.running'
  | 'console.capability.noResults'
  | 'console.capability.lastResults'
  | 'console.capability.models'
  | 'console.capability.duration'
  | 'console.capability.protocolResults'
  | 'console.capability.compatible'
  | 'capability.title'
  | 'capability.loadingTitle'
  | 'capability.startTest'
  | 'capability.cancel'
  | 'capability.cancelling'
  | 'capability.cancelled'
  | 'capability.partial'
  | 'capability.noCompatibleProtocols'
  | 'capability.modelQueued'
  | 'capability.protocolRunning'
  | 'capability.rpmLabel'
  | 'capability.progressSummary'
  | 'capability.supported'
  | 'capability.unsupported'
  | 'capability.table.protocol'
  | 'capability.table.status'
  | 'capability.table.successCount'
  | 'capability.table.latency'
  | 'capability.table.streaming'
  | 'capability.table.actions'
  | 'capability.copyToTab'
  | 'capability.currentTab'
  | 'capability.convert'
  | 'capability.modelsLabel'
  | 'capability.modelDetailsUnavailable'
  | 'capability.tooltipLatency'
  | 'capability.tooltipStreaming'
  | 'capability.modelStatus'
  | 'capability.actualModel'
  | 'capability.retryModel'
  | 'capability.testModel'
  | 'capability.runMode.reusedRunning'
  | 'capability.runMode.resumedCancelled'
  | 'capability.runMode.cacheHit'
  | 'capability.runMode.reusedPrevious'
  | 'capability.lastResults'
  | 'capability.noResults'
  | 'capability.models'
  | 'capability.duration'
  | 'capability.snapshotUpdated'
  | 'console.logs.title'
  | 'console.logs.time'
  | 'console.logs.model'
  | 'console.logs.baseUrl'
  | 'console.logs.key'
  | 'console.logs.statusCode'
  | 'console.logs.duration'
  | 'console.logs.error'
  | 'console.logs.entries'
  | 'console.logs.empty'
  | 'console.fuzzyEnabled'
  | 'console.fuzzyDisabled'
  | 'console.fuzzyLoadFailed'
  | 'console.cbSettings'
  | 'chart.totalRequests'
  | 'chart.successRate'
  | 'chart.inputTokens'
  | 'chart.outputTokens'
  | 'chart.cacheRw'
  | 'chart.noData'
  | 'channelEditor.title.create'
  | 'channelEditor.title.edit'
  | 'channelEditor.title.editSubtitle'
  | 'channelEditor.nav.basic'
  | 'channelEditor.nav.auth'
  | 'channelEditor.nav.redirect'
  | 'channelEditor.nav.advanced'
  | 'channelEditor.nav.custom'
  | 'channelEditor.basic.name.label'
  | 'channelEditor.basic.name.placeholder'
  | 'channelEditor.basic.name.required'
  | 'channelEditor.basic.serviceType.label'
  | 'channelEditor.basic.serviceType.required'
  | 'channelEditor.basic.baseUrl.label'
  | 'channelEditor.basic.baseUrl.placeholder'
  | 'channelEditor.basic.baseUrl.required'
  | 'channelEditor.basic.website.label'
  | 'channelEditor.basic.website.placeholder'
  | 'channelEditor.basic.description.label'
  | 'channelEditor.basic.description.placeholder'
  | 'channelEditor.auth.keys.label'
  | 'channelEditor.auth.apiKeyRequired'
  | 'channelEditor.auth.duplicateKey'
  | 'channelEditor.auth.addNewApiKey.placeholder'
  | 'channelEditor.mapping.redirect.title'
  | 'channelEditor.mapping.configured.label'
  | 'channelEditor.mapping.source.label'
  | 'channelEditor.mapping.source.placeholder'
  | 'channelEditor.mapping.target.label'
  | 'channelEditor.mapping.target.placeholder'
  | 'channelEditor.mapping.reasoningEffort.label'
  | 'channelEditor.actions.create'
  | 'channelEditor.actions.save'
  | 'channelEditor.actions.cancel'

export type Messages = Record<MessageKey, string>

export const messages: Record<SupportedLocale, Messages> = {
  en: {
    'common.gatewayLabel': 'CCX CORE',
    'common.online': 'GATEWAY ONLINE',
    'common.connecting': 'CONNECTING...',
    'common.offline': 'GATEWAY OFFLINE',
    'common.refreshWebUI': 'Refresh Web UI',
    'common.version': 'Version',
    'common.gatewayPort': 'Gateway port',
    'common.daemonPid': 'Daemon PID',
    'common.autoStart': 'Autostart',
    'common.autoStartOn': 'Enabled',
    'common.autoStartOff': 'Disabled',
    'common.serviceHealthy': 'Service healthy',
    'common.serviceStarting': 'Gateway starting',
    'common.serviceDisconnected': 'Service disconnected',
    'common.settings': 'Settings',
    'common.save': 'Save',
    'common.cancel': 'Cancel',
    'common.retry': 'Retry',
    'nav.status': 'Status',
    'nav.statusDesc': 'Live status and logs',
    'nav.agent': 'Agent',
    'nav.agentDesc': 'Local agent configuration',
    'nav.channels': 'Channels',
    'nav.channelsDesc': 'Protocol channel orchestration',
    'nav.env': 'Environment',
    'nav.envDesc': 'Edit gateway env file',
    'nav.dashboard': 'Dashboard',
    'nav.dashboardDesc': 'Native management dashboard',
    'tab.statusTitle': 'Gateway Status',
    'tab.agentTitle': 'Agent Config',
    'tab.channelsTitle': 'Channels Console',
    'tab.envTitle': 'Environment Settings',
    'tab.dashboardTitle': 'Management Dashboard',
    'sidebar.language': 'Language',
    'sidebar.languageEnglish': 'English',
    'sidebar.languageChinese': 'Chinese',
    'setup.loading': 'Initializing CCX Console',
    'setup.title': 'CCX Desktop Initial Setup',
    'setup.description': 'PROXY_ACCESS_KEY is the credential AI agents use to access upstream APIs through the CCX proxy. Every caller must have this key.',
    'setup.regenerate': 'Regenerate',
    'setup.regenerateTitle': 'Generate a new random key',
    'setup.hide': 'Hide',
    'setup.show': 'Show',
    'setup.copied': 'Copied',
    'setup.copyKey': 'Copy key',
    'setup.configPath': 'Config file path',
    'setup.copyPath': 'Copy path',
    'setup.hint': 'After saving, CCX will start automatically. You can adjust other settings later on the Environment page.',
    'setup.saving': 'Saving and starting...',
    'setup.submit': 'Finish setup and start',
    'webui.notRunning': 'CCX service is not running, so the Web UI cannot be displayed.',
    'webui.openInBrowser': 'Open in browser',
    'metrics.gatewayPort': 'Gateway port',
    'metrics.uptime': 'Uptime',
    'metrics.channels': 'Channels',
    'metrics.version': 'Gateway version',
    'actions.start': 'Start',
    'actions.stop': 'Stop',
    'actions.restart': 'Restart',
    'actions.openWebUI': 'Open Dashboard',
    'actions.openBrowser': 'Open in browser',
    'actions.refreshStatus': 'Refresh status',
    'details.title': 'Service details',
    'details.binary': 'Binary',
    'details.binaryMissing': 'Not found',
    'details.dataDir': 'Data dir',
    'details.dataDirMissing': 'Not configured',
    'details.healthStatus': 'Health',
    'details.revealDir': 'Reveal directory',
    'details.openDir': 'Open directory',
    'channel.headerEyebrow': 'Channel Preset Center',
    'channel.title': 'Channel Center',
    'channel.description': 'Use DeepSeek, MiMo, Kimi, GLM, MiniMax, DashScope, and more keys for both direct Agent routing and the unified CCX channel pool. Provider presets handle advanced switches automatically.',
    'channel.hasKey': 'Key saved',
    'channel.promo': 'Register via promotion link to claim the exclusive perk',
    'channel.console': 'Open official console',
    'channel.target': 'Target',
    'channel.planLabel': 'Token Plan / Base URL',
    'channel.apiKeyLabel': 'API Key',
    'channel.baseUrlPrefix': 'Base URL',
    'channel.keySavedPlaceholder': 'Saved locally; leave empty to reuse this Provider Key',
    'channel.keyInputPlaceholder': 'Enter API Key. It is stored only in local Desktop config',
    'channel.missingKey': 'Enter an API Key, or save this Provider key in Agent Config first.',
    'channel.reuseKey': 'Will reuse the saved local {provider} key',
    'channel.name': 'Channel name',
    'channel.nameHint': 'A channel with the same name will be overwritten. Use another name to create a separate channel.',
    'channel.presetWrites': 'Preset will write automatically',
    'channel.capabilityHint': 'Capability toggles: reasoning / vision / model list / compatibility fields are configured by Provider.',
    'channel.addToCCX': 'Add to CCX',
    'channel.badgeDirectAgent': 'Agent direct',
    'channel.badgeNativeMessages': 'Native Messages',
    // Provider labels & descriptions
    'channel.preset.deepseek.label': 'DeepSeek',
    'channel.preset.deepseek.description': 'V4-Pro / V4-Flash dual-flagship reasoning models at breakthrough costs. Fully open-source (MIT), delivering frontier-level performance at unbeatable prices.',
    'channel.preset.mimo.label': 'MiMo',
    'channel.preset.mimo.description': "Xiaomi's in-house AI platform for the Agent era. Flagship MiMo-V2.5-Pro with trillion parameters and 1M context, powering Xiaomi's smart ecosystem across devices, vehicles, and homes.",
    'channel.preset.compshare.label': 'Youyun Zhisuan Plans',
    'channel.preset.compshare.description': "Youyun Zhisuan is UCloud's AI cloud platform, offering cost-effective domestic AI model Agent Plan packages by monthly subscription or pay-as-you-go, starting from 49 CNY/month. It also provides stable access to official overseas models, supports Claude Code, Codex, and API integrations, and offers enterprise-grade high concurrency, 24/7 technical support, and self-service invoicing. Users who register through the promotion link can receive a free 5 CNY platform trial credit.",
    'channel.preset.runapi.label': 'RunAPI',
    'channel.preset.runapi.description': "RunAPI is an efficient and stable API platform—an alternative to OpenRouter. A single API Key gives you access to 150+ leading models, including OpenAI, Claude, Gemini, DeepSeek, Grok, and more, at prices as low as 10% of the original (up to 90% off), with exceptional stability. It's seamlessly compatible with tools like Claude Code, OpenClaw, and others. RunAPI offers an exclusive perk for CCX users: register and contact an administrator to claim ¥7 in free credit.",
    'channel.preset.kimi.label': 'Kimi / Moonshot',
    'channel.preset.kimi.description': "Moonshot AI's intelligent assistant platform. Flagship Kimi K2.6 with native multimodal, renowned for ultra-long context and Agent Swarm capabilities.",
    'channel.preset.glm.label': 'GLM / BigModel',
    'channel.preset.glm.description': "Zhipu AI's model-as-a-service platform from Tsinghua. Flagship GLM-5.1 rivals frontier models in coding. Hong Kong-listed, trained entirely on Huawei Ascend chips.",
    'channel.preset.minimax.label': 'MiniMax',
    'channel.preset.minimax.description': 'Global multimodal AI leader (HK-listed). Flagship M3 with 1M context, native vision, and Agent capabilities. Hailuo video/audio models in global top tier. Over 70% revenue from international markets.',
    'channel.preset.dashscope.label': 'Alibaba Cloud DashScope',
    'channel.preset.dashscope.description': "Alibaba Cloud's one-stop LLM platform integrating Qwen and third-party models. Full-modality coverage with pay-as-you-go, Coding Plan, and Token Plan options.",
    'channel.preset.opencode-zen.label': 'OpenCode Zen',
    'channel.preset.opencode-zen.description': 'Pay-as-you-go curated model gateway, supporting Messages, Chat, and Responses protocols.',
    'channel.preset.opencode-go.label': 'OpenCode Go',
    'channel.preset.opencode-go.description': 'Affordable open-source model subscription ($5/mo+), supporting Messages, Chat, and Responses protocols.',
    // DeepSeek plans
    'channel.preset.deepseek.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.deepseek.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.deepseek.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.deepseek.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // MiMo plans
    'channel.preset.mimo.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.mimo.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.mimo.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.mimo.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-cn.label': 'Token Plan - China (OpenAI)',
    'channel.preset.mimo.plan.token-cn.description': 'China Token Plan Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-sgp.label': 'Token Plan - Singapore (OpenAI)',
    'channel.preset.mimo.plan.token-sgp.description': 'Singapore Token Plan Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-ams.label': 'Token Plan - Europe (OpenAI)',
    'channel.preset.mimo.plan.token-ams.description': 'Europe Token Plan Chat / Responses shared endpoint',
    'channel.preset.mimo.plan.token-cn-anthropic.label': 'Token Plan - China (Anthropic)',
    'channel.preset.mimo.plan.token-cn-anthropic.description': 'China Token Plan Claude Messages native endpoint',
    'channel.preset.mimo.plan.token-sgp-anthropic.label': 'Token Plan - Singapore (Anthropic)',
    'channel.preset.mimo.plan.token-sgp-anthropic.description': 'Singapore Token Plan Claude Messages native endpoint',
    'channel.preset.mimo.plan.token-ams-anthropic.label': 'Token Plan - Europe (Anthropic)',
    'channel.preset.mimo.plan.token-ams-anthropic.description': 'Europe Token Plan Claude Messages native endpoint',
    // Compshare plans
    'channel.preset.compshare.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.compshare.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.compshare.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.compshare.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // RunAPI plans
    'channel.preset.runapi.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.runapi.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.runapi.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.runapi.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Kimi plans
    'channel.preset.kimi.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.kimi.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.kimi.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.kimi.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    'channel.preset.kimi.plan.coding-anthropic.label': 'Coding Plan (Anthropic)',
    'channel.preset.kimi.plan.coding-anthropic.description': 'Coding Plan Claude Messages native endpoint',
    'channel.preset.kimi.plan.coding-openai-chat.label': 'Coding Plan (OpenAI)',
    'channel.preset.kimi.plan.coding-openai-chat.description': 'Coding Plan Chat / Responses shared endpoint',
    // GLM plans
    'channel.preset.glm.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.glm.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.glm.plan.coding.label': 'Coding Plan (OpenAI)',
    'channel.preset.glm.plan.coding.description': 'Coding Plan Chat / Responses shared endpoint',
    'channel.preset.glm.plan.openai-chat.label': 'General (OpenAI)',
    'channel.preset.glm.plan.openai-chat.description': 'General Chat / Responses endpoint',
    // MiniMax plans
    'channel.preset.minimax.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.minimax.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.minimax.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.minimax.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // DashScope plans
    'channel.preset.dashscope.plan.anthropic.label': 'Pay-as-you-go (Anthropic)',
    'channel.preset.dashscope.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.dashscope.plan.openai-chat.label': 'Pay-as-you-go (OpenAI)',
    'channel.preset.dashscope.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    'channel.preset.dashscope.plan.coding-anthropic.label': 'Coding Plan (Anthropic)',
    'channel.preset.dashscope.plan.coding-anthropic.description': 'Coding Plan Claude Messages native endpoint',
    'channel.preset.dashscope.plan.coding-openai-chat.label': 'Coding Plan (OpenAI)',
    'channel.preset.dashscope.plan.coding-openai-chat.description': 'Coding Plan Chat / Responses shared endpoint',
    'channel.preset.dashscope.plan.token-plan-anthropic.label': 'Token Plan (Anthropic)',
    'channel.preset.dashscope.plan.token-plan-anthropic.description': 'Token Plan Claude Messages native endpoint',
    'channel.preset.dashscope.plan.token-plan-openai-chat.label': 'Token Plan (OpenAI)',
    'channel.preset.dashscope.plan.token-plan-openai-chat.description': 'Token Plan Chat / Responses shared endpoint',
    // Tencent Lkeap plans
    'channel.preset.tencent-lkeap.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.tencent-lkeap.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.tencent-lkeap.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.tencent-lkeap.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Kimi Code Plan plans
    'channel.preset.kimi-code.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.kimi-code.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.kimi-code.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.kimi-code.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Volcengine Ark Coding Plan plans
    'channel.preset.volc-ark.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.volc-ark.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.volc-ark.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.volc-ark.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Baidu Qianfan Coding Plan plans
    'channel.preset.qianfan.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.qianfan.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.qianfan.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.qianfan.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // Youyun OriginRouter plans
    'channel.preset.originrouter.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.originrouter.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.originrouter.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.originrouter.plan.openai-chat.description': 'Common Chat / Responses endpoint',
    // OpenCode Zen plans
    'channel.preset.opencode-zen.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.opencode-zen.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.opencode-zen.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.opencode-zen.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // OpenCode Go plans
    'channel.preset.opencode-go.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.opencode-go.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.opencode-go.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.opencode-go.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // ModelScope
    'channel.preset.modelscope.label': 'ModelScope',
    'channel.preset.modelscope.description': "Alibaba DAMO Academy's open-source model community with 10,000+ models across text, image, and speech. Full-lifecycle support for model exploration, inference, and training.",
    'channel.preset.modelscope.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.modelscope.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.modelscope.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.modelscope.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // OpenRouter
    'channel.preset.openrouter.label': 'OpenRouter',
    'channel.preset.openrouter.description': "The largest AI model aggregator — access hundreds of models with a single API key. OpenAI-compatible with built-in guardrails for security, governance, and cost control.",
    'channel.preset.openrouter.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.openrouter.plan.anthropic.description': 'Claude Messages native endpoint',
    'channel.preset.openrouter.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.openrouter.plan.openai-chat.description': 'Chat / Responses shared endpoint',
    // Targets (shared)
    'channel.target.messages.label': 'Claude Messages',
    'channel.target.messages.description': 'Native Claude Messages protocol, supports direct Claude Code connection or proxy via CCX',
    'channel.target.responses.label': 'Codex Responses',
    'channel.target.responses.description': 'OpenAI Responses protocol, designed for Codex CLI and compatible clients',
    'channel.target.chat.label': 'OpenAI Chat',
    'channel.target.chat.description': 'OpenAI Chat Completions protocol, compatible with various Chat clients and third-party tools',
    'env.title': 'Environment config',
    'env.pathDetecting': 'Detecting',
    'env.refresh': 'Refresh',
    'env.openingEditor': 'Opening…',
    'env.openWithEditor': 'Open in {editor}',
    'env.openInEditor': 'Open in editor',
    'env.selectEditor': 'Choose editor…',
    'env.save': 'Save',
    'env.saving': 'Saving',
    'env.saved': 'Saved',
    'env.failed': 'Failed',
    'env.hide': 'Hide',
    'env.show': 'Show',
    'env.copied': 'Copied',
    'env.copy': 'Copy',
    'env.fieldRequired': '{field} is required',
    'env.fieldDisallow': '{field} cannot use a sample placeholder value',
    'env.fieldNumber': '{field} must be a number',
    'env.fieldInteger': '{field} must be an integer',
    'env.fieldMin': '{field} must be at least {min}',
    'env.fieldMax': '{field} must be at most {max}',
    'env.loadFailed': 'Failed to load config: {error}',
    'env.saveSuccess': '.env saved; changes take effect after restarting the service',
    'agent.statusDetecting': 'Detecting',
    'agent.statusConfigured': 'Configured',
    'agent.statusPortMismatch': 'Port mismatch',
    'agent.statusUnconfigured': 'Not configured',
    'agent.localGateway': 'Current CCX gateway',
    'agent.custom': 'Custom',
    'agent.currentProvider': 'Current provider',
    'agent.currentUrl': 'Current URL',
    'agent.targetUrl': 'Target URL',
    'agent.notSet': 'Not configured',
    'agent.configPath': 'Config file',
    'agent.authPath': 'Auth file',
    'agent.openFileInEditor': 'Open in editor',
    'agent.applyConfig': 'Apply config',
    'agent.restoreConfig': 'Restore original config',
    'agent.migrateSessions': 'Migrate sessions',
    'agent.migrateConfirmTitle': 'Migrate Codex sessions',
    'agent.migrateConfirmDesc': 'This will rewrite Codex local session metadata to the current provider. It updates history, archived sessions, and Codex state database, but never deletes sessions.',
    'agent.migrateTarget': 'Target model_provider',
    'agent.migrateRunning': 'Migrating sessions…',
    'agent.migrateConfirm': 'Confirm migration',
    'agent.migrateClose': 'Close',
    'agent.migrateResultSuccess': 'Migration complete: {migrated}/{total} session files updated',
    'agent.migrateResultSkippedLabel': 'Skipped files',
    'agent.migrateResultFailedLabel': 'Failed files',
    'agent.migrateSQLiteUpdated': 'SQLite database updated: {count} rows',
    'agent.migrateSQLiteSkipped': 'SQLite database skipped',
    'agent.openConsole': 'Open official console',
    'agent.codexPlaceholderSaved': 'Saved locally; leave empty to reuse the key',
    'agent.codexPlaceholderRequired': 'Required: enter API Key',
    'agent.codexPlaceholderWriteOnly': 'Written to Codex config only',
    'agent.diffPreviewApply': 'Apply config preview',
    'agent.diffPreviewRestore': 'Restore config preview',
    'agent.diffConfirmApply': 'Confirm apply',
    'agent.diffConfirmRestore': 'Confirm restore',
    'agent.diffActionCreate': 'Create',
    'agent.diffActionDelete': 'Delete',
    'agent.diffActionModify': 'Modify',
    'agent.diffComputing': 'Computing changes…',
    'agent.diffNoChanges': 'No changes',
    'agent.diffExpandContext': 'Expand {count} unchanged lines',
    'agent.diffCollapseContext': 'Collapse {count} lines',
    'agent.diffCancel': 'Cancel',
    'agent.sessionMigrationWarning': 'Switching provider mode will make existing sessions invisible in Codex. Sessions are not deleted — switch back to the previous mode to access them.',
    'agent.codexMode': 'Mode',
    'agent.codexQuickMode': 'Quick mode',
    'agent.codexPluginMode': 'Plugin mode',
    'agent.codexQuickModeHint': 'Keeps OpenAI conversation history. Uses OpenAI as provider with the configured API endpoint.',
    'agent.codexPluginModeHint': 'Enables Codex plugins. Previous OpenAI sessions may not be visible — switch back to Quick mode to access them.',
    'agent.provider.localGateway': 'CCX local gateway',
    'agent.provider.deepseekDirect': 'DeepSeek direct',
    'agent.provider.mimoDirect': 'MiMo direct',
    'agent.provider.compshareDirect': 'Compshare direct',
    'agent.provider.runapiDirect': 'RunAPI direct',
    'agent.provider.kimiDirect': 'Kimi direct',
    'agent.provider.glmDirect': 'GLM direct',
    'agent.provider.minimaxDirect': 'MiniMax direct',
    'agent.provider.dashscopeDirect': 'DashScope direct',
    'agent.provider.opencodeZenDirect': 'OpenCode Zen direct',
    'agent.provider.opencodeGoDirect': 'OpenCode Go direct',
    'agent.provider.openaiDirect': 'OpenAI direct',
    'agent.provider.openrouterDirect': 'OpenRouter direct',
    'agent.provider.modelscopeDirect': 'ModelScope direct',
    'agent.provider.xfyunDirect': 'iFlytek Xinghuo MAAS direct',
    'agent.provider.tencentLkeapDirect': 'Tencent TokenHub direct',
    'agent.provider.volcArkDirect': 'Volc Ark direct',
    'agent.provider.qianfanDirect': 'Qianfan direct',
    'agent.hasOwnApiKey': 'I have my own API key',
    'agent.promo': 'Register via promotion link to claim the exclusive perk',
    'agent.planPayAsYouGo': 'Pay-as-you-go',
    'agent.planChina': 'Subscription - China',
    'agent.planSingapore': 'Subscription - Singapore',
    'agent.planEurope': 'Subscription - Europe',
    'agent.planSubscription': 'Subscription',
    'agent.billingModeMiMo': 'MiMo billing mode',
    'agent.billingModeDashScope': 'DashScope billing mode',
    'agent.placeholderSaved': 'Saved locally; leave empty to reuse the key',
    'agent.placeholderMimo': 'Required: MiMo API Key (tp-xxx or account key)',
    'agent.placeholderDashScope': 'Required: DashScope API Key (sk-xxx or sk-sp-xxx)',
    'agent.placeholderRequired': 'Required: enter API Key',
    'agent.codexDiagnosticTitle': 'Codex troubleshooting',
    'agent.codexTroubleshoot': 'Troubleshoot',
    'agent.codexTroubleshooting': 'Checking...',
    'agent.codexDiagnosticHealthy': 'Current Codex config looks consistent with CCX.',
    'agent.codexDiagnosticMismatch': 'config.toml and auth.json may be inconsistent.',
    'agent.codexDiagnosticCheck': 'Check item',
    'agent.codexDiagnosticAuthMode': 'Auth mode',
    'agent.codexDiagnosticConfigPath': 'config.toml',
    'agent.codexDiagnosticAuthPath': 'auth.json',
    'agent.codexDiagnosticSuggestedApply': 'Suggested action: re-apply Codex -> CCX config.',
    'agent.codexDiagnosticSuggestedRestore': 'If the issue persists, restore first and then apply again.',
    'agent.codexDiagnosticStatusTitle': 'Codex diagnostic summary',
    'agent.codexDiagnosticStatusHint': 'Open the Agent tab to inspect Codex config consistency.',
    'agent.codexDiagnosticCodeMissingApiKey': 'config.toml points to local CCX, but auth.json is missing OPENAI_API_KEY.',
    'agent.codexDiagnosticCodeProxyKeyMismatch': 'The proxy key stored in Codex no longer matches the current CCX proxy key.',
    'agent.codexDiagnosticCodeAuthModeMismatch': 'config.toml and auth.json disagree on the required auth mode.',
    'agent.codexDiagnosticCodePluginMissingBearer': 'Plugin mode is missing experimental_bearer_token.',
    'agent.codexDiagnosticCodeLegacyIncomplete': 'Legacy ccx provider config is incomplete.',
    'agent.codexDiagnosticCodeConfigPolluted': 'config.toml may have been polluted by another tool and no longer matches CCX expectations.',
    'agent.codexDiagnosticCodeAuthUnreadable': 'auth.json cannot be parsed and may be corrupted.',
    'agent.codexDiagnosticCodeModeMismatch': 'Current Codex config is {currentMode} mode, but you selected {targetMode} mode. Please re-apply the config.',
    'agent.responsesDiagnosticTitle': 'Responses channel diagnostic',
    'agent.responsesDiagnosticHealthy': 'Responses channels look available.',
    'agent.responsesDiagnosticNoChannel': 'No Responses channel is configured in CCX.',
    'agent.responsesDiagnosticAllDisabled': 'Responses channels exist, but none is currently active.',
    'agent.responsesDiagnosticNoApiKeys': 'Active Responses channels are missing API keys.',
    'agent.responsesDiagnosticProtocolMismatch': 'Responses channels exist, but their protocol routing does not look compatible with /v1/responses.',
    'agent.responsesDiagnosticCircuitOpen': 'All active Responses channels are currently circuit-open.',
    'agent.responsesDiagnosticSuggestedRefresh': 'Suggested action: refresh channel dashboard and logs.',
    'agent.responsesDiagnosticSuggestedCreate': 'Suggested action: add a Responses channel for your Codex provider.',
    'agent.responsesDiagnosticSuggestedEnable': 'Suggested action: enable or resume at least one Responses channel.',
    'agent.responsesDiagnosticSuggestedAddKey': 'Suggested action: add an API key to an active Responses channel.',
    'agent.responsesDiagnosticSuggestedProtocolMismatch': 'Suggested action: check serviceType and baseUrl so the channel resolves to a /responses endpoint.',
    'agent.responsesDiagnosticSuggestedResume': 'Suggested action: inspect logs and resume the affected channels after recovery.',
    'agent.responsesLogsDiagnosticTitle': 'Recent failed requests',
    'agent.responsesLogsDiagnosticHealthy': 'No recent failed Responses requests were found.',
    'agent.responsesLogsDiagnosticAuth': 'Recent Responses requests failed with authentication errors (401/403).',
    'agent.responsesLogsDiagnosticRateLimit': 'Recent Responses requests were rate limited (429).',
    'agent.responsesLogsDiagnosticUpstream5xx': 'Recent Responses requests failed with upstream 5xx errors.',
    'agent.responsesLogsDiagnosticTimeout': 'Recent Responses requests look stalled or timed out.',
    'agent.responsesLogsDiagnosticGeneric': 'Recent Responses requests failed. Check channel logs for details.',
    'agent.responsesLogsDiagnosticSuggestedRefresh': 'Suggested action: refresh the latest channel logs.',
    'agent.responsesLogsDiagnosticSuggestedAuth': 'Suggested action: verify the upstream API key, account quota, and auth settings.',
    'agent.responsesLogsDiagnosticSuggestedRateLimit': 'Suggested action: reduce request rate or wait for the upstream limit window to recover.',
    'agent.responsesLogsDiagnosticSuggestedUpstream5xx': 'Suggested action: switch channel, inspect upstream status, or retry later.',
    'agent.responsesLogsDiagnosticSuggestedTimeout': 'Suggested action: inspect upstream latency, timeout settings, and recent logs.',
    'agent.responsesLogsDiagnosticSuggestedGeneric': 'Suggested action: open the Responses channel logs and inspect the latest failures.',
    'agent.codexDiagnosticIntro': 'Troubleshooting runs on demand: config consistency → Responses channel availability → recent failed requests.',
    'agent.codexDiagnosticLayerConfig': 'Layer 1 · Codex config',
    'agent.codexDiagnosticLayerChannels': 'Layer 2 · Responses channels',
    'agent.codexDiagnosticLayerLogs': 'Layer 3 · Recent failed requests',
    'env.groupAccess': 'Access control',
    'env.groupAccessDesc': 'Access keys for the proxy and admin endpoints.',
    'env.fieldProxyAccessKey': 'Proxy access key',
    'env.placeholderProxyAccessKey': 'Enter a strong random key',
    'env.fieldAdminAccessKey': 'Admin API key',
    'env.placeholderAdminAccessKey': 'Leave empty to fall back to PROXY_ACCESS_KEY',
    'env.descAdminAccessKey': 'Used for the Web UI and /api/* endpoints.',
    'env.groupServer': 'Server config',
    'env.groupServerDesc': 'Desktop injects some runtime values on start; this section still matches .env.example.',
    'env.fieldPort': 'Port',
    'env.descPort': 'Preferred startup port. If occupied, the next available port is used automatically.',
    'env.fieldEnv': 'Environment',
    'env.descEnv': 'production is recommended.',
    'env.groupWebUI': 'Web UI config',
    'env.groupWebUIDesc': 'Control whether the Web UI is enabled and which language it uses by default.',
    'env.fieldEnableWebUI': 'Enable Web UI',
    'env.descEnableWebUI': 'Desktop mode usually forces this on.',
    'env.fieldAppUILanguage': 'Default language',
    'env.groupLogs': 'Logging config',
    'env.groupLogsDesc': 'Control request/response logs, SSE debugging, and response model rewriting.',
    'env.fieldLogLevel': 'Log level',
    'env.fieldEnableRequestLogs': 'Enable request logs',
    'env.fieldEnableResponseLogs': 'Enable response logs',
    'env.descEnableResponseLogs': 'Response logs may expose sensitive content.',
    'env.fieldQuietPollingLogs': 'Quiet polling logs',
    'env.fieldRawLogOutput': 'Raw log output',
    'env.fieldSseDebugLevel': 'SSE debug level',
    'env.fieldRewriteResponseModel': 'Rewrite response model',
    'env.groupPerformance': 'Performance config',
    'env.groupPerformanceDesc': 'Request-chain timeouts and request body size limits.',
    'env.fieldRequestTimeout': 'Request timeout (ms)',
    'env.fieldServerReadTimeout': 'Server read timeout (ms)',
    'env.fieldMaxRequestBodySize': 'Max request body size (MB)',
    'env.fieldResponseHeaderTimeout': 'Response header timeout (s)',
    'env.groupCors': 'CORS config',
    'env.groupCorsDesc': 'Cross-origin access control.',
    'env.fieldEnableCors': 'Enable CORS',
    'env.fieldCorsOrigin': 'Allowed Origin',
    'env.groupCircuitBreaker': 'Circuit-breaker metrics config',
    'env.groupCircuitBreakerDesc': 'Control scheduler metric windows and failure-rate thresholds.',
    'env.fieldMetricsWindowSize': 'Window size',
    'env.fieldMetricsFailureThreshold': 'Failure-rate threshold',
    'env.runtimeCbTitle': 'Tuning Bench',
    'env.runtimeCbDesc': 'Fine-tune circuit breaker and stream control parameters. Changes take effect immediately.',
    'env.runtimeCbWindowSize': 'Sliding Window Size',
    'env.runtimeCbWindowSizeDesc': 'Number of recent requests for failure rate (3-100)',
    'env.runtimeCbFailureThreshold': 'Failure Rate Threshold',
    'env.runtimeCbFailureThresholdDesc': 'Triggers circuit break when exceeded (0.01-1.0)',
    'env.runtimeCbConsecutiveFailures': 'Consecutive Failures Threshold',
    'env.runtimeCbConsecutiveFailuresDesc': 'Triggers circuit break on N consecutive retryable failures (1-100)',
    'env.runtimeCbStreamFirstContentTimeout': 'First Content Timeout',
    'env.runtimeCbStreamFirstContentTimeoutDesc': 'Wait for first valid content after HTTP 200 (0=disabled, 1000-300000ms)',
    'env.runtimeCbStreamInactivityTimeout': 'Inactivity Timeout',
    'env.runtimeCbStreamInactivityTimeoutDesc': 'Wait for subsequent activity after first content (0=disabled, 1000-180000ms)',
    'env.runtimeCbStreamToolCallIdleTimeout': 'Tool Call Idle Timeout',
    'env.runtimeCbStreamToolCallIdleTimeoutDesc': 'Wait for valid output while a tool call is pending (1000-180000ms)',
    'env.runtimeCbPresetGentle': 'Gentle',
    'env.runtimeCbPresetBalanced': 'Balanced',
    'env.runtimeCbPresetAggressive': 'Aggressive',
    'env.runtimeCbPresetCustom': 'Custom',
    'env.runtimeCbSaved': 'Tuning Bench settings saved. Changes take effect immediately.',
    'env.runtimeCbSaveFailed': 'Failed to save: {error}',
    'env.runtimeCbLoadFailed': 'Failed to load: {error}',
    'env.runtimeCbNoBackend': 'Backend service not running. Start the service first.',
    'env.runtimeCbServiceStopped': 'Service is stopped. Start it to configure runtime parameters.',
    'env.historicalImageTurnLimitTitle': 'Historical Image Turn Limit',
    'env.historicalImageTurnLimitLabel': 'Max history turns for image context',
    'env.historicalImageTurnLimitHint': 'Limit the number of previous conversation turns included when sending image context to upstream. 0 = no limit.',
    'env.historicalImageTurnLimitSaved': 'Historical image turn limit saved.',
    'env.groupMetricsPersistence': 'Metrics persistence config',
    'env.groupMetricsPersistenceDesc': 'Control SQLite metrics persistence and data retention.',
    'env.fieldMetricsPersistenceEnabled': 'Enable metrics persistence',
    'env.fieldMetricsRetentionDays': 'Metrics retention days',
    'logs.searchPlaceholder': 'Search logs...',
    'logs.autoScroll': 'Auto-scroll to bottom',
    'logs.copied': 'Copied!',
    'logs.copyAll': 'Copy all logs',
    'logs.clear': 'Clear log console',
    'logs.noSearchResults': 'No matching log lines found',
    'logs.empty': 'No logs yet. Start the service to view output.',
    'diagnostic.binaryTitle': 'Binary not found',
    'diagnostic.binarySuggestionBuild': 'Confirm the CCX binary has been built: cd backend-go && make build',
    'diagnostic.binarySuggestionCheckDataDir': 'Check whether ccx-go / ccx-go.exe exists in the Desktop data directory',
    'diagnostic.binarySuggestionDownload': 'Build the backend first, or download a prebuilt release from the Release page',
    'diagnostic.portTitle': 'Port conflict',
    'diagnostic.portSuggestionInstance': 'Check whether another CCX instance is already running',
    'diagnostic.portSuggestionEnv': 'Change the PORT field in .env to another port',
    'diagnostic.portSuggestionInspect': 'Use lsof -i :3688 (macOS/Linux) or netstat -ano | findstr :3688 (Windows) to inspect port usage',
    'diagnostic.healthTitle': 'Health check timeout',
    'diagnostic.healthSuggestionLogs': 'Check the log panel for startup errors',
    'diagnostic.healthSuggestionEnv': 'Check whether .env has syntax errors',
    'diagnostic.healthSuggestionChannels': 'Confirm upstream channel configuration is correct. First startup may take longer.',
    'diagnostic.healthSuggestionRestart': 'Try restarting the service manually',
    'diagnostic.permissionTitle': 'Insufficient permissions',
    'diagnostic.permissionSuggestionDataDir': 'Check whether the data directory is writable',
    'diagnostic.permissionSuggestionExecutable': 'macOS/Linux: confirm the binary is executable (chmod +x)',
    'diagnostic.permissionSuggestionWindows': 'Windows: try running as administrator',
    'diagnostic.genericTitle': 'Startup failed',
    'diagnostic.genericSuggestionLogs': 'Check the log panel below for detailed errors',
    'diagnostic.genericSuggestionRestart': 'Try restarting the service',
    'setup.errorEmptyKey': 'PROXY_ACCESS_KEY cannot be empty',
    'env.saveSuccessHint': '.env saved; changes take effect after restarting the service',
    'env.openedInEditor': '.env opened in editor',
    'sidebar.versionHintStore': 'Microsoft Store version updates automatically via Store',
    'sidebar.versionHintTray': 'Check for updates from the tray menu',
    'sidebar.versionClickCheck': 'Click to check for updates',
    'sidebar.updateAvailable': 'New {version}',
    'sidebar.updateAvailableHint': 'Click to view release notes',
    'sidebar.theme': 'Theme',
    'sidebar.themeAuto': 'System',
    'sidebar.themeLight': 'Light',
    'sidebar.themeDark': 'Dark',
    'console.channelsTab': 'Channels',
    'console.conversationsTab': 'Cockpit',
    'console.nativeChannelsEyebrow': 'Native channel orchestration',
    'console.nativeChannelsTitle': 'WebUI-aligned protocol workspace',
    'console.quickAdd': 'Quick Add',
    'console.quickAddEyebrow': 'Provider Presets',
    'console.quickAddTitle': 'Quick Add Channel',
    'console.searchChannels': 'Search channels...',
    'console.actions.refresh': 'Refresh',
    'console.pingAll': 'Ping all',
    'console.addChannel': 'Add channel',
    'console.noChannels': 'No channels yet. Click the button above to add one.',
    'console.noSearchResults': 'No matching channels',
    'console.mode.multi': 'Multi-channel',
    'console.mode.single': 'Single-channel',
    'console.pool.active': 'Failover Sequence',
    'console.pool.current': 'Current channel',
    'console.pool.inactive': 'Standby Pool',
    'console.channelStatus.active': 'Active',
    'console.channelStatus.suspended': 'Suspended',
    'console.channelStatus.disabled': 'Standby',
    'console.channel.cacheWriteHigh': 'Cache write high',
    'console.channel.cacheWriteHighHint': 'The cache write ratio is high in the last 15 minutes, which may indicate suboptimal cache-hit configuration.',
    'console.circuit.open': 'Circuit Open',
    'console.circuit.halfOpen': 'Half-Open',
    'console.keys.active': 'active keys',
    'console.actions.label': 'Actions',
    'console.actions.edit': 'Edit Channel',
    'console.actions.capability': 'Capability Test',
    'console.actions.logs': 'View Logs',
    'console.actions.copy': 'Copy Config',
    'console.actions.website': 'Visit Website',
    'console.actions.ping': 'Ping',
    'console.actions.enable': 'Enable',
    'console.actions.suspend': 'Suspend',
    'console.actions.resume': 'Resume',
    'console.actions.promote': 'Promote',
    'console.actions.disable': 'Move to Standby',
    'console.actions.delete': 'Delete Channel',
    'console.actions.resetCircuit': 'Reset Circuit Breaker',
    'orchestration.moveTop': 'Move to top',
    'orchestration.moveBottom': 'Move to bottom',
    'orchestration.keepOne': '(keep at least one)',
    'orchestration.deleteActiveGuard': 'Cannot delete: keep at least one active channel in the failover sequence',
    'console.keys.disabled': 'disabled keys',
    'console.form.addChannel': 'Add channel',
    'console.form.editChannel': 'Edit channel',
    'console.form.basicInfo': 'Basic info',
    'console.form.name': 'Name',
    'console.form.nameRequired': 'Channel name is required',
    'console.form.serviceType': 'Service type',
    'console.form.selectServiceType': 'Select service type',
    'console.form.serviceTypeRequired': 'Please select a service type',
    'console.form.description': 'Description',
    'console.form.connection': 'Connection',
    'console.form.baseUrl': 'Base URL',
    'console.form.baseUrlRequired': 'At least one Base URL is required',
    'console.form.additionalUrls': 'Additional URLs (one per line)',
    'console.form.proxyUrl': 'Proxy URL',
    'console.form.routePrefix': 'Route prefix',
    'console.form.insecureSkipVerify': 'Skip TLS verify',
    'console.form.authentication': 'Authentication',
    'console.form.apiKeys': 'API Keys (one per line)',
    'console.form.disabledKeys': 'Disabled keys',
    'console.form.restoreKey': 'Restore',
    'console.form.historicalKeys': 'historical keys recorded',
    'addChannel.oneClickSetup': 'One-click setup',
    'addChannel.expectedRequest': 'Expected request',
    'console.form.apiKeyRequired': 'At least one API Key required',
    'console.form.reasoningEffort': 'Reasoning effort',
    'console.form.noVisionModels': 'No vision models (one per line)',
    'console.form.visionFallbackModel': 'Vision fallback model',
    'console.form.fetchModels': 'Fetch models',
    'console.form.fetchingModels': 'Fetching...',
    'console.form.modelFetchNeedsConfig': 'Base URL and API Key required to fetch models',
    'console.form.visionEnabled': 'Vision enabled',
    'console.form.visionDisabled': 'Vision disabled',
    'console.form.models': 'Models',
    'console.form.modelMapping': 'Model mapping (JSON)',
    'console.form.supportedModels': 'Supported models (one per line, empty=all)',
    'console.form.protocolOptions': 'Protocol & model advanced options',
    'console.form.reasoningParamStyle': 'Reasoning param style',
    'console.form.textVerbosity': 'Text verbosity',
    'console.form.advancedFlags': 'Advanced flags',
    'console.form.noVision': 'No vision',
    'console.form.passbackReasoning': 'Passback reasoning content',
    'console.form.passbackThinking': 'Passback thinking blocks',
    'console.form.fastMode': 'Fast mode',
    'console.form.lowQuality': 'Low quality',
    'console.form.injectDummySignature': 'Inject dummy thought signature',
    'console.form.stripThoughtSignature': 'Strip thought signature',
    'console.form.stripEmptyBlocks': 'Strip empty text blocks',
    'console.form.normalizeSystem': 'Normalize system role',
    'console.form.normalizeUserId': 'Normalize user ID',
    'console.form.stripBillingHeader': 'Strip CCH billing parameter',
    'console.form.normalizeChatRoles': 'Normalize chat roles',
    'console.form.autoBlacklist': 'Auto-blacklist balance-anomaly keys',
    'console.form.codexNativeTools': 'Codex native tool passthrough',
    'console.form.codexCompat': 'Codex tool compat',
    'console.form.stripCodexTools': 'Strip Codex client tools',
    'console.form.customHeaders': 'Custom headers (JSON)',
    'console.form.requestTimeoutMs': 'Request timeout (ms)',
    'console.form.requestTimeoutInvalid': 'Timeout must be a positive integer in ms',
    'console.form.streamTimeouts': 'Stream timeouts',
    'console.form.streamFirstContentTimeoutOverrideLabel': 'Customize first content timeout',
    'console.form.streamFirstContentTimeoutLabel': 'First content timeout',
    'console.form.streamInactivityTimeoutOverrideLabel': 'Customize inactivity timeout',
    'console.form.streamInactivityTimeoutLabel': 'Inactivity timeout',
    'console.form.streamToolCallIdleTimeoutOverrideLabel': 'Customize tool call idle timeout',
    'console.form.streamToolCallIdleTimeoutLabel': 'Tool call idle timeout',
    'console.form.streamTimeoutOverrideHint': 'Custom value overrides the global stream timeout for this channel.',
    'console.form.streamTimeoutInheritHint': 'Disabled means this channel inherits the global stream timeout.',
    'console.form.cancel': 'Cancel',
    'console.form.create': 'Create',
    'console.form.save': 'Save',
    'console.form.website': 'Website',
    'console.form.baseUrlPlaceholder': 'One per line, first is primary, the rest are failover\nhttps://api.example.com\nhttps://backup.example.com',
    'console.form.sectionBasic': 'Basic config',
    'console.form.sectionAuth': 'Authentication',
    'console.form.sectionRedirect': 'Model redirect',
    'console.form.sectionAdvanced': 'Advanced options',
    'console.form.sectionHeaders': 'Custom parameters',
    'console.form.outline': 'Outline',
    'console.form.compatibilityTitle': 'Compatibility',
    'console.form.runtimeTitle': 'Runtime',
    'console.form.transportTitle': 'Transport',
    'console.form.rateLimitTitle': 'Rate Limit',
    'console.form.streamTimeoutTitle': 'Stream Timeout',
    'console.form.fastModeHint': 'Prefer low-latency lightweight edge routing',
    'console.form.textVerbosityStyle': 'Text Verbosity Style',
    'console.form.visionTitle': 'Vision',
    'console.form.selectDefaultLabel': 'Default',
    'console.form.autoBlacklistBalanceLabel': 'Auto blacklist abnormal balance keys',
    'console.form.autoBlacklistBalanceHint': 'Automatically trip/disable keys with balance errors',
    'console.form.rateLimitAutoLabel': 'Auto learn upstream rate limiting',
    'console.form.rateLimitAutoHint': 'Parse Retry-After / x-ratelimit-* headers and adjust cooldown dynamically',
    'console.form.proxyUrlLabel': 'Proxy URL',
    'console.form.requestTimeoutLabel': 'Request timeout (ms)',
    'console.form.routePrefixLabel': 'Route prefix',
    'console.form.rpmLabel': 'Max requests per minute (RPM)',
    'console.form.rpmPlaceholder': 'No limit',
    'console.form.windowLabel': 'Sliding window (s)',
    'console.form.windowPlaceholder': '60s',
    'console.form.maxConcurrentLabel': 'Max concurrent streams',
    'console.form.maxConcurrentPlaceholder': 'No limit',
    'console.form.presetGentle': 'Gentle',
    'console.form.presetBalanced': 'Balanced',
    'console.form.presetAggressive': 'Aggressive',
    'console.form.firstByteWait': 'First byte wait',
    'console.form.firstByteWaitHint': 'Auto disconnect if no response',
    'console.form.idleAfterFirstByte': 'Post-first-byte idle',
    'console.form.idleAfterFirstByteHint': 'Mid-generation stall timeout',
    'console.form.toolCallIdle': 'Tool call idle',
    'console.form.toolCallIdleHint': 'FunctionCall delay timeout',
    'console.form.timeoutThreshold': 'Timeout:',
    'console.form.supportedModelsLabel': 'Supported models (whitelist, leave empty for no restriction)',
    'console.form.basicInfoTitle': 'Basic info',
    'console.form.connectionTitle': 'Connection',
    'console.form.multiLineFailover': 'Multi-line for failover',
    'console.form.expectedEndpoint': 'Expected endpoint:',
    'console.form.descriptionPlaceholder': 'Optional: note about this provider...',
    'console.form.mappingHint': 'Intercept source aliases in requests and route to the real upstream target model',
    'console.form.visionFallbackHint': 'Models disabled via the vision toggle on redirect rows will auto-fallback to this model for image requests',
    'console.form.modelRedirect': 'Model redirect',
    'console.form.modelMappingExisting': 'Configured',
    'console.form.modelMappingAdd': 'Add redirect',
    'console.form.generationParams': 'Generation params',
    'console.form.textVerbosityPlaceholder': 'Default',
    'console.form.modelScope': 'Model scope',
    'console.form.selectDefault': 'Default',
    'console.form.noVisionHint': 'When enabled, requests containing images skip this channel and failover to the next one.',
    'console.form.historicalImageTurnLimit': 'Historical Image Turn Limit',
    'console.form.historicalImageTurnLimitHint': '0 = inherit global; backend applies a minimum of 3 for values > 0.',
    'console.form.compactModel': 'Compact Model',
    'console.form.compactModelHint': 'Upstream model name for local compact (compression/summary), bypassing model mapping. Always used for non-Responses protocol channels; for Responses channels, used as fallback when native compact fails.',
    'console.form.passbackReasoningHint': 'Convert thinking blocks into reasoning_content for passback, compatible with Claude-protocol upstreams that expect OpenAI-style reasoning_content such as MiMo.',
    'console.form.passbackThinkingHint': 'Project real reasoning_content into Claude content[].thinking, compatible with strict thinking-mode upstreams such as DeepSeek/GLM.',
    'console.form.stripThoughtSignatureHint': 'Remove the thought_signature field from functionCall, compatible with older Gemini APIs that do not support it.',
    'console.form.injectDummySignatureHint': 'Inject a dummy signature into functionCall, compatible with third-party APIs that require it (disable for official APIs).',
    'console.form.reasoningParamStyleHint': 'Choose whether OpenAI-style upstream requests use reasoning.effort or reasoning_effort.',
    'console.form.codexNativeToolsHint': 'In passthrough mode, convert Codex native tools (apply_patch, namespace, etc.) into OpenAI function format so the upstream model can call them.',
    'console.form.codexCompatHint': 'Enable Codex CLI compatibility: Responses passthrough upstreams strip client-only tools, while Chat/Claude/Gemini upstreams convert them into function proxy tools.',
    'console.form.stripEmptyBlocksHint': 'Strip bare empty text content blocks before forwarding, compatible with Claude-protocol upstreams that strictly reject Claude Code tool_use placeholder blocks.',
    'console.form.normalizeSystemHint': 'For new clients such as Opus 4.8 / Fable 5 that send system as a message role: pull it back into the top-level system field before forwarding, compatible with older Claude upstreams that only support user/assistant roles.',
    'console.form.normalizeUserIdHint': 'Automatically convert JSON-object user_id into a flat string to ensure upstream compatibility.',
    'console.form.stripBillingHeaderHint': 'Remove cch= billing parameters from system text blocks before forwarding, only for the current Messages channel.',
    'console.form.normalizeChatRolesHint': 'Convert non-standard roles such as developer into user before forwarding upstream. Domestic models usually do not support non-standard roles, so enabling this is recommended.',
    'console.form.lowQualityHint': 'When enabled, force local token estimation; use the local value when the deviation exceeds 5%.',
    'console.form.autoBlacklistHint': 'When the upstream returns insufficient balance, automatically move that key into the blacklist.',
    'console.form.insecureSkipVerifyHint': 'Only enable temporarily for self-signed or mismatched domains; disable in production.',
    'console.form.proxyUrlHint': 'Supports HTTP/HTTPS/SOCKS5 proxies for reaching upstream services through a proxy.',
    'console.form.routePrefixHint': 'Access this channel via /{prefix}/v1/messages; multiple channels can share the same prefix.',
    'console.form.requestTimeoutMsHint': 'Applies only to non-streaming upstream requests; leave empty to inherit the global REQUEST_TIMEOUT.',
    'console.form.rateLimitSectionLabel': 'Rate limiting',
    'console.form.rateLimitSectionHint': 'Proactively throttle requests before they reach the upstream to avoid 429 errors.',
    'console.form.rateLimitRpmLabel': 'RPM',
    'console.form.rateLimitRpmHint': 'Max requests/min upstream. 0 or empty = no limit.',
    'console.form.rateLimitBurstLabel': 'Burst',
    'console.form.rateLimitBurstHint': 'Token bucket burst capacity. Auto if empty.',
    'console.form.rateLimitMaxConcurrentLabel': 'Max concurrent',
    'console.form.rateLimitMaxConcurrentHint': 'Max in-flight upstream requests. 0 or empty = no limit.',
    'console.form.rateLimitAutoFromHeadersLabel': 'Auto-learn rate limits',
    'console.form.rateLimitAutoFromHeadersHint': 'Parse Retry-After / x-ratelimit-* headers for dynamic cooldown.',
    'addChannel.duplicateKey': 'Duplicate',
    'addChannel.addNewApiKeyPlaceholder': 'Enter a new API Key, press Enter to add',
    'addChannel.quickMode': 'Quick Paste',
    'addChannel.quickHint': 'Paste Base URL, API Key, or full config snippet to auto-detect and fill the form.',
    'addChannel.applyDetected': 'Apply detected',
    'addChannel.noneDetected': 'None detected',
    'addChannel.fullSubtitle': 'Configure channel, keys, models and advanced protocol options',
    'addChannel.editSubtitle': 'Modify channel configuration',
    'console.conversations.search': 'Search conversations...',
    'console.conversations.allKinds': 'All kinds',
    'console.conversations.refresh': 'Refresh',
    'console.conversations.total': '{count} conversations',
    'console.conversations.empty': 'No active conversations',
    'console.conversations.noSearchResults': 'No matching conversations',
    'console.conversations.overrideTitle': 'Channel override',
    'console.conversations.overrideHint': 'Select the channel sequence for this conversation (by priority):',
    'console.conversations.overrideSequence': 'Override sequence',
    'console.conversations.overrideRequired': 'Please select at least one channel',
    'console.conversations.saveOverride': 'Save override',
    'console.conversations.noChannelsForKind': 'No channels available for this type',
    'console.conversations.active': 'Active: {count}',
    'console.conversations.override': 'Override: {count}',
    'console.conversations.duration.default': '30 min (default)',
    'console.conversations.duration.15min': '15 min',
    'console.conversations.duration.1hour': '1 hour',
    'console.conversations.duration.2hours': '2 hours',
    'console.conversations.duration.4hours': '4 hours',
    'console.conversations.duration.8hours': '8 hours',
    'console.conversations.duration.12hours': '12 hours',
    'console.conversations.duration.24hours': '24 hours',
    'console.conversations.duration.never': 'Never restore',
    'console.capability.title': 'Capability test',
    'console.capability.start': 'Start test',
    'console.capability.cancel': 'Cancel',
    'console.capability.running': 'Test in progress...',
    'console.capability.noResults': 'No capability test results yet',
    'console.capability.lastResults': 'Last test results',
    'console.capability.models': 'Models',
    'console.capability.duration': 'Duration',
    'console.capability.protocolResults': 'Protocol results',
    'console.capability.compatible': 'Compatible protocols',
    'capability.title': 'Capability test',
    'capability.loadingTitle': 'Testing protocol compatibility...',
    'capability.startTest': 'Start test',
    'capability.cancel': 'Cancel',
    'capability.cancelling': 'Cancelling...',
    'capability.cancelled': 'Cancelled',
    'capability.partial': 'Partial',
    'capability.noCompatibleProtocols': 'No compatible protocols',
    'capability.modelQueued': 'Model queued',
    'capability.protocolRunning': 'Protocol testing',
    'capability.rpmLabel': 'RPM',
    'capability.progressSummary': '{done}/{total} models finished',
    'capability.supported': 'Supported',
    'capability.unsupported': 'Unsupported',
    'capability.table.protocol': 'Protocol',
    'capability.table.status': 'Status',
    'capability.table.successCount': 'Success',
    'capability.table.latency': 'Latency',
    'capability.table.streaming': 'SSE',
    'capability.table.actions': 'Actions',
    'capability.copyToTab': 'Copy to tab',
    'capability.currentTab': 'Current tab',
    'capability.convert': 'Convert',
    'capability.modelsLabel': 'Models',
    'capability.modelDetailsUnavailable': 'Model details unavailable',
    'capability.tooltipLatency': 'Latency',
    'capability.tooltipStreaming': 'Streaming',
    'capability.modelStatus': 'Status',
    'capability.actualModel': 'Actual model',
    'capability.retryModel': 'Retry',
    'capability.testModel': 'Test',
    'capability.runMode.reusedRunning': 'Reused running',
    'capability.runMode.resumedCancelled': 'Resumed cancelled',
    'capability.runMode.cacheHit': 'Cache hit',
    'capability.runMode.reusedPrevious': 'Reused previous',
    'capability.lastResults': 'Last results',
    'capability.noResults': 'No results yet',
    'capability.models': 'models',
    'capability.duration': 'Duration',
    'capability.snapshotUpdated': 'Updated',
    'console.logs.title': 'Channel logs',
    'console.logs.time': 'Time',
    'console.logs.model': 'Model',
    'console.logs.baseUrl': 'URL',
    'console.logs.key': 'Key',
    'console.logs.statusCode': 'Status',
    'console.logs.duration': 'Duration',
    'console.logs.error': 'Error',
    'console.logs.entries': 'entries',
    'console.logs.empty': 'No logs yet',
    'console.fuzzyEnabled': 'Fuzzy mode enabled',
    'console.fuzzyDisabled': 'Fuzzy mode disabled',
    'console.fuzzyLoadFailed': 'Failed to load fuzzy mode status',
    'console.cbSettings': 'Tuning Bench',
    'chart.totalRequests': 'Total requests',
    'chart.successRate': 'Availability',
    'chart.inputTokens': 'Input tokens',
    'chart.outputTokens': 'Output tokens',
    'chart.cacheRw': 'Cache R/W',
    'chart.noData': 'No request data',
    'channelEditor.title.create': 'Add new channel',
    'channelEditor.title.edit': 'Edit channel',
    'channelEditor.title.editSubtitle': 'Modify channel configuration',
    'channelEditor.nav.basic': 'Basic Info',
    'channelEditor.nav.auth': 'Authentication',
    'channelEditor.nav.redirect': 'Model Redirect',
    'channelEditor.nav.advanced': 'Advanced Options',
    'channelEditor.nav.custom': 'Custom Parameters',
    'channelEditor.basic.name.label': 'Channel name',
    'channelEditor.basic.name.placeholder': 'e.g. OpenAI Production',
    'channelEditor.basic.name.required': 'Channel name is required',
    'channelEditor.basic.serviceType.label': 'Service type',
    'channelEditor.basic.serviceType.required': 'Service type is required',
    'channelEditor.basic.baseUrl.label': 'Base URL',
    'channelEditor.basic.baseUrl.placeholder': 'https://api.openai.com/v1',
    'channelEditor.basic.baseUrl.required': 'At least one Base URL is required',
    'channelEditor.basic.website.label': 'Website',
    'channelEditor.basic.website.placeholder': 'https://openai.com',
    'channelEditor.basic.description.label': 'Description',
    'channelEditor.basic.description.placeholder': 'Optional: note about this provider...',
    'channelEditor.auth.keys.label': 'API Keys',
    'channelEditor.auth.apiKeyRequired': 'At least one API key is required',
    'channelEditor.auth.duplicateKey': 'Duplicate API key',
    'channelEditor.auth.addNewApiKey.placeholder': 'Enter new API key',
    'channelEditor.mapping.redirect.title': 'Model Redirect',
    'channelEditor.mapping.configured.label': 'Configured',
    'channelEditor.mapping.source.label': 'Source model',
    'channelEditor.mapping.source.placeholder': 'gpt-4o',
    'channelEditor.mapping.target.label': 'Target model',
    'channelEditor.mapping.target.placeholder': 'claude-sonnet-4-6',
    'channelEditor.mapping.reasoningEffort.label': 'Reasoning effort',
    'channelEditor.actions.create': 'Create channel',
    'channelEditor.actions.save': 'Save',
    'channelEditor.actions.cancel': 'Cancel',
  },
  'zh-CN': {
    'common.gatewayLabel': 'CCX CORE',
    'common.online': 'GATEWAY ONLINE',
    'common.connecting': 'CONNECTING...',
    'common.offline': 'GATEWAY OFFLINE',
    'common.refreshWebUI': '刷新 Web UI',
    'common.version': '当前版本',
    'common.gatewayPort': '网关端口',
    'common.daemonPid': '守护 PID',
    'common.autoStart': '开机自启',
    'common.autoStartOn': '已开启',
    'common.autoStartOff': '已关闭',
    'common.serviceHealthy': '运行正常',
    'common.serviceStarting': '网关启动中',
    'common.serviceDisconnected': '服务已断开',
    'common.settings': '设置',
    'common.save': '保存',
    'common.cancel': '取消',
    'common.retry': '重试',
    'nav.status': '网关监控',
    'nav.statusDesc': '实时状态及核心日志',
    'nav.agent': 'Agent 配置',
    'nav.agentDesc': '本地开发代理控制',
    'nav.channels': '渠道中心',
    'nav.channelsDesc': '协议级渠道编排',
    'nav.env': '环境参数',
    'nav.envDesc': '网关配置文件编辑',
    'nav.dashboard': '管理面板',
    'nav.dashboardDesc': '原生管理面板',
    'tab.statusTitle': '网关状态监控',
    'tab.agentTitle': 'Agent 代理配置',
    'tab.channelsTitle': '渠道管理控制台',
    'tab.envTitle': '环境参数管理',
    'tab.dashboardTitle': '管理面板',
    'sidebar.language': '语言',
    'sidebar.languageEnglish': 'English',
    'sidebar.languageChinese': '中文',
    'setup.loading': '正在初始化 CCX 控制台',
    'setup.title': 'CCX Desktop 初始配置',
    'setup.description': 'PROXY_ACCESS_KEY 是 AI Agent 通过 CCX 代理访问上游 API 的身份凭证，所有调用方必须持有此密钥。',
    'setup.regenerate': '重新生成',
    'setup.regenerateTitle': '重新生成随机密钥',
    'setup.hide': '隐藏',
    'setup.show': '显示',
    'setup.copied': '已复制',
    'setup.copyKey': '复制密钥',
    'setup.configPath': '配置文件路径',
    'setup.copyPath': '复制路径',
    'setup.hint': '保存后 CCX 将自动启动。后续可在主界面【环境参数】页继续调整其他配置。',
    'setup.saving': '正在保存并启动...',
    'setup.submit': '完成配置并启动',
    'webui.notRunning': 'CCX 服务尚未启动，无法显示 Web UI。',
    'webui.openInBrowser': '浏览器打开',
    'metrics.gatewayPort': '网关端口',
    'metrics.uptime': '运行时长',
    'metrics.channels': '调度信道',
    'metrics.version': '网关版本',
    'actions.start': '启动服务',
    'actions.stop': '停止服务',
    'actions.restart': '重启服务',
    'actions.openWebUI': '打开管理面板',
    'actions.openBrowser': '浏览器直达',
    'actions.refreshStatus': '刷新当前状态',
    'details.title': '服务详情',
    'details.binary': '二进制',
    'details.binaryMissing': '未发现',
    'details.dataDir': '数据目录',
    'details.dataDirMissing': '未设置',
    'details.healthStatus': '健康状态',
    'details.revealDir': '打开所在目录',
    'details.openDir': '打开目录',
    'channel.headerEyebrow': 'Channel Preset Center',
    'channel.title': '渠道中心',
    'channel.description': '统一把 DeepSeek、MiMo、Kimi、GLM、MiniMax、DashScope 等 Key 可同时用于 Agent 直连和 CCX 统一渠道池，复杂开关由预设自动处理。',
    'channel.hasKey': '已有 Key',
    'channel.promo': '通过推广链接注册领取专属福利',
    'channel.console': '访问官方控制台',
    'channel.target': '添加目标',
    'channel.planLabel': '套餐 / Base URL',
    'channel.apiKeyLabel': 'API Key',
    'channel.baseUrlPrefix': 'Base URL',
    'channel.keySavedPlaceholder': '已保存，留空则复用该 Provider Key',
    'channel.keyInputPlaceholder': '输入 API Key，仅保存在本机 Desktop 配置中',
    'channel.missingKey': '请填写 API Key，或先在 Agent 配置中保存该 Provider 的 key。',
    'channel.reuseKey': '将复用本机已保存的 {provider} Key',
    'channel.name': '渠道名称',
    'channel.nameHint': '同名渠道会被直接覆盖更新；如需新建独立渠道，请改用不同名称。',
    'channel.presetWrites': '预设将自动写入',
    'channel.capabilityHint': '能力开关：reasoning / vision / model list / 兼容字段会按 Provider 自动配置。',
    'channel.addToCCX': '添加到 CCX',
    'channel.badgeDirectAgent': 'Agent 直连',
    'channel.badgeNativeMessages': 'Messages 原生',
    // Provider labels & descriptions
    'channel.preset.deepseek.label': 'DeepSeek',
    'channel.preset.deepseek.description': 'V4-Pro / V4-Flash 双旗舰推理模型，突破性成本。完全开源（MIT），以极具竞争力的价格提供前沿级性能。',
    'channel.preset.mimo.label': 'MiMo',
    'channel.preset.mimo.description': '小米自研 AI 平台，面向 Agent 时代打造。旗舰 MiMo-V2.5-Pro 具备万亿参数与百万上下文窗口，赋能小米跨设备、车、家的智能生态。',
    'channel.preset.compshare.label': '优云智算套餐',
    'channel.preset.compshare.description': '优云智算是 UCloud 旗下 AI 云平台，提供高性价比国内 AI 模型 Agent Plan 套餐，支持包月订阅或按量付费（49 元/月起），同时提供官方海外模型稳定接入，支持 Claude Code、Codex 与 API 集成，具备企业级高并发、7×24 技术支持和自助开票能力；通过推广链接注册可领取 5 元平台试用金。',
    'channel.preset.runapi.label': 'RunAPI',
    'channel.preset.runapi.description': 'RunAPI 是高效稳定的API OpenRouter平替平台，一个 API Key 即可访问 OpenAI、Claude、Gemini、DeepSeek、Grok 等 150+ 主流模型，低至 1 折，极其稳定，可以无缝兼容 Claude Code、OpenClaw 等工具。RunAPI 为 CCX用户提供专属福利：注册联系管理员即可领取￥7的免费额度',
    'channel.preset.kimi.label': 'Kimi / Moonshot',
    'channel.preset.kimi.description': 'Moonshot AI 智能助手平台。旗舰 Kimi K2.6 支持原生多模态，以超长上下文和 Agent Swarm 能力著称。',
    'channel.preset.glm.label': 'GLM / BigModel',
    'channel.preset.glm.description': '清华系智谱 AI MaaS 平台。旗舰 GLM-5.1 在编程领域达到前沿模型水平，港股上市，全栈华为昇腾芯片训练。',
    'channel.preset.minimax.label': 'MiniMax',
    'channel.preset.minimax.description': '全球多模态 AI 领导者（港股上市）。旗舰 M3 具备百万上下文、原生视觉与 Agent 能力；海螺视频/音频模型位居全球前列；超 70% 收入来自海外市场。',
    'channel.preset.dashscope.label': '阿里云 DashScope',
    'channel.preset.dashscope.description': '阿里云一站式 LLM 平台，集成通义千问及第三方模型，全模态覆盖，提供按量、编程套餐与 Token Plan 选项。',
    'channel.preset.opencode-zen.label': 'OpenCode Zen',
    'channel.preset.opencode-zen.description': '按量付费精选模型网关，支持 Messages、Chat、Responses 三种协议。',
    'channel.preset.opencode-go.label': 'OpenCode Go',
    'channel.preset.opencode-go.description': '低成本开源编程模型订阅服务（$5/月起），支持 Messages、Chat、Responses 三种协议。',
    // DeepSeek plans
    'channel.preset.deepseek.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.deepseek.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.deepseek.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.deepseek.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // MiMo plans
    'channel.preset.mimo.plan.anthropic.label': '按量 (Anthropic)',
    'channel.preset.mimo.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.mimo.plan.openai-chat.label': '按量 (OpenAI)',
    'channel.preset.mimo.plan.openai-chat.description': 'Chat / Responses 通用入口',
    'channel.preset.mimo.plan.token-cn.label': 'Token Plan - 中国 (OpenAI)',
    'channel.preset.mimo.plan.token-cn.description': '中国区 Token Plan Chat / Responses 通用入口',
    'channel.preset.mimo.plan.token-sgp.label': 'Token Plan - 新加坡 (OpenAI)',
    'channel.preset.mimo.plan.token-sgp.description': '新加坡区 Token Plan Chat / Responses 通用入口',
    'channel.preset.mimo.plan.token-ams.label': 'Token Plan - 欧洲 (OpenAI)',
    'channel.preset.mimo.plan.token-ams.description': '欧洲区 Token Plan Chat / Responses 通用入口',
    'channel.preset.mimo.plan.token-cn-anthropic.label': 'Token Plan - 中国 (Anthropic)',
    'channel.preset.mimo.plan.token-cn-anthropic.description': '中国区 Token Plan Claude Messages 原生入口',
    'channel.preset.mimo.plan.token-sgp-anthropic.label': 'Token Plan - 新加坡 (Anthropic)',
    'channel.preset.mimo.plan.token-sgp-anthropic.description': '新加坡区 Token Plan Claude Messages 原生入口',
    'channel.preset.mimo.plan.token-ams-anthropic.label': 'Token Plan - 欧洲 (Anthropic)',
    'channel.preset.mimo.plan.token-ams-anthropic.description': '欧洲区 Token Plan Claude Messages 原生入口',
    // Compshare plans
    'channel.preset.compshare.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.compshare.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.compshare.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.compshare.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // RunAPI plans
    'channel.preset.runapi.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.runapi.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.runapi.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.runapi.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // Kimi plans
    'channel.preset.kimi.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.kimi.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.kimi.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.kimi.plan.openai-chat.description': 'Chat / Responses 通用入口',
    'channel.preset.kimi.plan.coding-anthropic.label': 'Coding Plan (Anthropic)',
    'channel.preset.kimi.plan.coding-anthropic.description': 'Coding Plan Claude Messages 原生入口',
    'channel.preset.kimi.plan.coding-openai-chat.label': 'Coding Plan (OpenAI)',
    'channel.preset.kimi.plan.coding-openai-chat.description': 'Coding Plan Chat / Responses 通用入口',
    // GLM plans
    'channel.preset.glm.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.glm.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.glm.plan.coding.label': 'Coding Plan (OpenAI)',
    'channel.preset.glm.plan.coding.description': 'Coding Plan Chat / Responses 通用入口',
    'channel.preset.glm.plan.openai-chat.label': '通用 (OpenAI)',
    'channel.preset.glm.plan.openai-chat.description': '通用 Chat / Responses 入口',
    // MiniMax plans
    'channel.preset.minimax.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.minimax.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.minimax.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.minimax.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // DashScope plans
    'channel.preset.dashscope.plan.anthropic.label': '按量 (Anthropic)',
    'channel.preset.dashscope.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.dashscope.plan.openai-chat.label': '按量 (OpenAI)',
    'channel.preset.dashscope.plan.openai-chat.description': 'Chat / Responses 通用入口',
    'channel.preset.dashscope.plan.coding-anthropic.label': 'Coding Plan (Anthropic)',
    'channel.preset.dashscope.plan.coding-anthropic.description': 'Coding Plan Claude Messages 原生入口',
    // Tencent Lkeap plans
    'channel.preset.tencent-lkeap.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.tencent-lkeap.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.tencent-lkeap.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.tencent-lkeap.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // Kimi Code Plan plans
    'channel.preset.kimi-code.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.kimi-code.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.kimi-code.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.kimi-code.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // Volcengine Ark Coding Plan plans
    'channel.preset.volc-ark.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.volc-ark.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.volc-ark.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.volc-ark.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // Baidu Qianfan Coding Plan plans
    'channel.preset.qianfan.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.qianfan.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.qianfan.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.qianfan.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // Youyun OriginRouter plans
    'channel.preset.originrouter.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.originrouter.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.originrouter.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.originrouter.plan.openai-chat.description': 'Chat / Responses 通用入口',
    'channel.preset.dashscope.plan.coding-openai-chat.label': 'Coding Plan (OpenAI)',
    'channel.preset.dashscope.plan.coding-openai-chat.description': 'Coding Plan Chat / Responses 通用入口',
    'channel.preset.dashscope.plan.token-plan-anthropic.label': 'Token Plan (Anthropic)',
    'channel.preset.dashscope.plan.token-plan-anthropic.description': 'Token Plan Claude Messages 原生入口',
    'channel.preset.dashscope.plan.token-plan-openai-chat.label': 'Token Plan (OpenAI)',
    'channel.preset.dashscope.plan.token-plan-openai-chat.description': 'Token Plan Chat / Responses 通用入口',
    // OpenCode Zen plans
    'channel.preset.opencode-zen.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.opencode-zen.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.opencode-zen.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.opencode-zen.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // OpenCode Go plans
    'channel.preset.opencode-go.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.opencode-go.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.opencode-go.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.opencode-go.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // ModelScope
    'channel.preset.modelscope.label': 'ModelScope',
    'channel.preset.modelscope.description': '阿里达摩院开源模型社区，涵盖文本、图像、语音等领域 10,000+ 模型，提供模型探索、推理与训练全链路支持。',
    'channel.preset.modelscope.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.modelscope.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.modelscope.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.modelscope.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // OpenRouter
    'channel.preset.openrouter.label': 'OpenRouter',
    'channel.preset.openrouter.description': '最大规模 AI 模型聚合平台，单个 API Key 即可访问数百款模型。OpenAI 兼容协议，内置安全治理与成本控制能力。',
    'channel.preset.openrouter.plan.anthropic.label': 'Anthropic-compatible',
    'channel.preset.openrouter.plan.anthropic.description': 'Claude Messages 原生入口',
    'channel.preset.openrouter.plan.openai-chat.label': 'OpenAI-compatible',
    'channel.preset.openrouter.plan.openai-chat.description': 'Chat / Responses 通用入口',
    // Targets (shared)
    'channel.target.messages.label': 'Claude Messages',
    'channel.target.messages.description': 'Claude 原生 Messages 协议，支持 Claude Code 直连或通过 CCX 代理',
    'channel.target.responses.label': 'Codex Responses',
    'channel.target.responses.description': 'OpenAI Responses 协议，专供 Codex CLI 及兼容客户端使用',
    'channel.target.chat.label': 'OpenAI Chat',
    'channel.target.chat.description': 'OpenAI Chat Completions 协议，兼容各类 Chat 客户端和第三方工具',
    'env.title': '环境配置',
    'env.pathDetecting': '检测中',
    'env.refresh': '刷新',
    'env.openingEditor': '打开中…',
    'env.openWithEditor': '用 {editor} 打开',
    'env.openInEditor': '用编辑器打开',
    'env.selectEditor': '选择编辑器…',
    'env.save': '保存',
    'env.saving': '保存中',
    'env.saved': '已保存',
    'env.failed': '失败',
    'env.hide': '隐藏',
    'env.show': '显示',
    'env.copied': '已复制',
    'env.copy': '复制',
    'env.fieldRequired': '{field}不能为空',
    'env.fieldDisallow': '{field}不能使用示例占位值',
    'env.fieldNumber': '{field}必须是数字',
    'env.fieldInteger': '{field}必须是整数',
    'env.fieldMin': '{field}不能小于 {min}',
    'env.fieldMax': '{field}不能大于 {max}',
    'env.loadFailed': '加载配置失败：{error}',
    'env.saveSuccess': '.env 已保存，重启服务后生效',
    'agent.statusDetecting': '检测中',
    'agent.statusConfigured': '已配置',
    'agent.statusPortMismatch': '端口不匹配',
    'agent.statusUnconfigured': '未配置',
    'agent.localGateway': '当前 CCX 网关',
    'agent.custom': '自定义',
    'agent.currentProvider': '当前 Provider',
    'agent.currentUrl': '当前 URL',
    'agent.targetUrl': '目标 URL',
    'agent.notSet': '未设置',
    'agent.configPath': '配置文件',
    'agent.authPath': '认证文件',
    'agent.openFileInEditor': '用编辑器打开',
    'agent.applyConfig': '应用配置',
    'agent.restoreConfig': '恢复原始配置',
    'agent.migrateSessions': '迁移会话',
    'agent.migrateConfirmTitle': '迁移 Codex 会话',
    'agent.migrateConfirmDesc': '将 Codex 本地会话元数据改写为当前 provider，覆盖历史会话、归档会话和 Codex 状态数据库，但不会删除任何会话。',
    'agent.migrateTarget': '目标 model_provider',
    'agent.migrateRunning': '正在迁移会话...',
    'agent.migrateConfirm': '确认迁移',
    'agent.migrateClose': '关闭',
    'agent.migrateResultSuccess': '迁移完成：{migrated}/{total} 个会话文件已更新',
    'agent.migrateResultSkippedLabel': '跳过文件',
    'agent.migrateResultFailedLabel': '失败文件',
    'agent.migrateSQLiteUpdated': 'SQLite 数据库已更新：{count} 行',
    'agent.migrateSQLiteSkipped': 'SQLite 数据库已跳过',
    'agent.openConsole': '访问官方控制台',
    'agent.codexPlaceholderSaved': '已保存，留空则使用已保存的 key',
    'agent.codexPlaceholderRequired': '必填：输入 API Key',
    'agent.codexPlaceholderWriteOnly': '仅写入 Codex 配置',
    'agent.diffPreviewApply': '应用配置预览',
    'agent.diffPreviewRestore': '恢复配置预览',
    'agent.diffConfirmApply': '确认应用',
    'agent.diffConfirmRestore': '确认恢复',
    'agent.diffActionCreate': '创建',
    'agent.diffActionDelete': '删除',
    'agent.diffActionModify': '修改',
    'agent.diffComputing': '计算变更中...',
    'agent.diffNoChanges': '无变更',
    'agent.diffExpandContext': '展开 {count} 行未变更内容',
    'agent.diffCollapseContext': '收起 {count} 行',
    'agent.diffCancel': '取消',
    'agent.codexMode': '模式',
    'agent.codexQuickMode': '快捷模式',
    'agent.codexPluginMode': '插件模式',
    'agent.codexQuickModeHint': '保留 OpenAI 对话历史。使用 OpenAI provider + 目标 API 端点。',
    'agent.codexPluginModeHint': '启用 Codex 插件功能。切换后之前的 OpenAI 对话可能不可见，切回快捷模式即可访问。',
    'agent.sessionMigrationWarning': '切换 Codex 模式后，已有会话可能不可见。会话不会被删除 — 切回之前的模式即可访问。',
    'agent.provider.localGateway': 'CCX 本地网关',
    'agent.provider.deepseekDirect': 'DeepSeek 直连',
    'agent.provider.mimoDirect': 'MiMo 直连',
    'agent.provider.compshareDirect': 'Compshare 直连',
    'agent.provider.runapiDirect': 'RunAPI 直连',
    'agent.provider.kimiDirect': 'Kimi 直连',
    'agent.provider.glmDirect': 'GLM 直连',
    'agent.provider.minimaxDirect': 'MiniMax 直连',
    'agent.provider.dashscopeDirect': 'DashScope 直连',
    'agent.provider.opencodeZenDirect': 'OpenCode Zen 直连',
    'agent.provider.opencodeGoDirect': 'OpenCode Go 直连',
    'agent.provider.openaiDirect': 'OpenAI 直连',
    'agent.provider.openrouterDirect': 'OpenRouter 直连',
    'agent.provider.modelscopeDirect': 'ModelScope 直连',
    'agent.provider.xfyunDirect': '讯飞星辰直连',
    'agent.provider.tencentLkeapDirect': '腾讯云 TokenHub 直连',
    'agent.provider.volcArkDirect': '火山方舟直连',
    'agent.provider.qianfanDirect': '百度千帆直连',
    'agent.hasOwnApiKey': '我有自己的 API Key',
    'agent.promo': '通过推广链接注册领取专属福利',
    'agent.planPayAsYouGo': '按量',
    'agent.planChina': '订阅套餐 - 中国',
    'agent.planSingapore': '订阅套餐 - 新加坡',
    'agent.planEurope': '订阅套餐 - 欧洲',
    'agent.planSubscription': '订阅套餐',
    'agent.billingModeMiMo': 'MiMo 计费模式',
    'agent.billingModeDashScope': 'DashScope 计费模式',
    'agent.placeholderSaved': '已保存，留空则使用已保存的 key',
    'agent.placeholderMimo': '必填：MiMo API Key（tp-xxx 或账号 key）',
    'agent.placeholderDashScope': '必填：DashScope API Key（sk-xxx 或 sk-sp-xxx）',
    'agent.placeholderRequired': '必填：输入 API Key',
    'agent.codexDiagnosticTitle': 'Codex 排障诊断',
    'agent.codexTroubleshoot': '排障检测',
    'agent.codexTroubleshooting': '检测中...',
    'agent.codexDiagnosticHealthy': '当前 Codex 配置与 CCX 看起来是一致的。',
    'agent.codexDiagnosticMismatch': 'config.toml 与 auth.json 可能不一致。',
    'agent.codexDiagnosticCheck': '检查项',
    'agent.codexDiagnosticAuthMode': '认证模式',
    'agent.codexDiagnosticConfigPath': 'config.toml',
    'agent.codexDiagnosticAuthPath': 'auth.json',
    'agent.codexDiagnosticSuggestedApply': '建议动作：重新应用 Codex -> CCX 配置。',
    'agent.codexDiagnosticSuggestedRestore': '如果问题仍在，先恢复原始配置，再重新应用。',
    'agent.codexDiagnosticStatusTitle': 'Codex 排障摘要',
    'agent.codexDiagnosticStatusHint': '请前往 Agent 页检查 Codex 配置一致性。',
    'agent.codexDiagnosticCodeMissingApiKey': 'config.toml 已指向本地 CCX，但 auth.json 缺少 OPENAI_API_KEY。',
    'agent.codexDiagnosticCodeProxyKeyMismatch': 'Codex 当前保存的代理密钥与 CCX 当前代理密钥不一致。',
    'agent.codexDiagnosticCodeAuthModeMismatch': 'config.toml 与 auth.json 要求的认证模式不一致。',
    'agent.codexDiagnosticCodePluginMissingBearer': '插件模式缺少 experimental_bearer_token。',
    'agent.codexDiagnosticCodeLegacyIncomplete': '旧式 ccx provider 配置不完整。',
    'agent.codexDiagnosticCodeConfigPolluted': 'config.toml 可能被其他工具污染，已不符合 CCX 预期。',
    'agent.codexDiagnosticCodeAuthUnreadable': 'auth.json 无法解析，可能已经损坏。',
    'agent.codexDiagnosticCodeModeMismatch': '当前 Codex 配置是{currentMode}模式，但你选择了{targetMode}模式。请重新应用配置。',
    'agent.responsesDiagnosticTitle': 'Responses 渠道诊断',
    'agent.responsesDiagnosticHealthy': 'Responses 渠道当前看起来可用。',
    'agent.responsesDiagnosticNoChannel': 'CCX 当前没有配置任何 Responses 渠道。',
    'agent.responsesDiagnosticAllDisabled': '已存在 Responses 渠道，但当前没有任何活跃渠道。',
    'agent.responsesDiagnosticNoApiKeys': '活跃的 Responses 渠道缺少 API Key。',
    'agent.responsesDiagnosticProtocolMismatch': '已存在 Responses 渠道，但它们的协议路由看起来并不兼容 /v1/responses。',
    'agent.responsesDiagnosticCircuitOpen': '所有活跃的 Responses 渠道当前都处于熔断状态。',
    'agent.responsesDiagnosticSuggestedRefresh': '建议动作：刷新渠道状态和日志。',
    'agent.responsesDiagnosticSuggestedCreate': '建议动作：为当前 Codex provider 新增一个 Responses 渠道。',
    'agent.responsesDiagnosticSuggestedEnable': '建议动作：启用或恢复至少一个 Responses 渠道。',
    'agent.responsesDiagnosticSuggestedAddKey': '建议动作：给活跃的 Responses 渠道补充 API Key。',
    'agent.responsesDiagnosticSuggestedProtocolMismatch': '建议动作：检查 serviceType 和 baseUrl，确保该渠道最终落到 /responses 端点。',
    'agent.responsesDiagnosticSuggestedResume': '建议动作：查看日志并在恢复后重新启用相关渠道。',
    'agent.responsesLogsDiagnosticTitle': '最近失败请求',
    'agent.responsesLogsDiagnosticHealthy': '未发现最近的 Responses 失败请求。',
    'agent.responsesLogsDiagnosticAuth': '最近的 Responses 请求出现了认证错误（401/403）。',
    'agent.responsesLogsDiagnosticRateLimit': '最近的 Responses 请求触发了限流（429）。',
    'agent.responsesLogsDiagnosticUpstream5xx': '最近的 Responses 请求出现了上游 5xx 错误。',
    'agent.responsesLogsDiagnosticTimeout': '最近的 Responses 请求看起来发生了超时或卡住。',
    'agent.responsesLogsDiagnosticGeneric': '最近的 Responses 请求失败了，请查看渠道日志获取细节。',
    'agent.responsesLogsDiagnosticSuggestedRefresh': '建议动作：刷新最近的渠道日志。',
    'agent.responsesLogsDiagnosticSuggestedAuth': '建议动作：检查上游 API Key、账户额度和认证设置。',
    'agent.responsesLogsDiagnosticSuggestedRateLimit': '建议动作：降低请求频率，或等待上游限流窗口恢复。',
    'agent.responsesLogsDiagnosticSuggestedUpstream5xx': '建议动作：切换渠道、检查上游状态，或稍后再试。',
    'agent.responsesLogsDiagnosticSuggestedTimeout': '建议动作：检查上游延迟、超时配置和最近日志。',
    'agent.responsesLogsDiagnosticSuggestedGeneric': '建议动作：打开 Responses 渠道日志并检查最近失败项。',
    'agent.codexDiagnosticIntro': '按需排障检测：配置一致性 → Responses 渠道可用性 → 最近失败请求。',
    'agent.codexDiagnosticLayerConfig': '第 1 层 · Codex 配置',
    'agent.codexDiagnosticLayerChannels': '第 2 层 · Responses 渠道',
    'agent.codexDiagnosticLayerLogs': '第 3 层 · 最近失败请求',
    'env.groupAccess': '访问控制',
    'env.groupAccessDesc': '代理入口与管理入口的访问密钥。',
    'env.fieldProxyAccessKey': '代理访问密钥',
    'env.placeholderProxyAccessKey': '请输入强随机密钥',
    'env.fieldAdminAccessKey': '管理 API 独立密钥',
    'env.placeholderAdminAccessKey': '留空则回退到 PROXY_ACCESS_KEY',
    'env.descAdminAccessKey': '用于管理界面和 /api/* 端点。',
    'env.groupServer': '服务器配置',
    'env.groupServerDesc': 'Desktop 会在启动时注入部分运行参数；这里仍完整覆盖 .env.example。',
    'env.fieldPort': '服务端口',
    'env.descPort': '启动时优先使用此端口，被占用时自动递增分配。',
    'env.fieldEnv': '运行环境',
    'env.descEnv': 'production 为推荐值。',
    'env.groupWebUI': 'Web UI 配置',
    'env.groupWebUIDesc': '控制管理界面是否启用以及默认语言。',
    'env.fieldEnableWebUI': '启用 Web UI',
    'env.descEnableWebUI': 'Desktop 模式通常会强制启用。',
    'env.fieldAppUILanguage': '默认语言',
    'env.groupLogs': '日志配置',
    'env.groupLogsDesc': '控制请求/响应日志、SSE 调试和模型字段改写。',
    'env.fieldLogLevel': '日志级别',
    'env.fieldEnableRequestLogs': '启用请求日志',
    'env.fieldEnableResponseLogs': '启用响应日志',
    'env.descEnableResponseLogs': '响应日志可能增加敏感内容暴露风险。',
    'env.fieldQuietPollingLogs': '静默轮询日志',
    'env.fieldRawLogOutput': '原始日志输出',
    'env.fieldSseDebugLevel': 'SSE 调试级别',
    'env.fieldRewriteResponseModel': '改写响应 model',
    'env.groupPerformance': '性能配置',
    'env.groupPerformanceDesc': '请求链路超时和请求体大小限制。',
    'env.fieldRequestTimeout': '请求超时（毫秒）',
    'env.fieldServerReadTimeout': '服务端读取超时（毫秒）',
    'env.fieldMaxRequestBodySize': '请求体最大大小（MB）',
    'env.fieldResponseHeaderTimeout': '响应头超时（秒）',
    'env.groupCors': 'CORS 配置',
    'env.groupCorsDesc': '跨域访问控制。',
    'env.fieldEnableCors': '启用 CORS',
    'env.fieldCorsOrigin': '允许的 Origin',
    'env.groupCircuitBreaker': '熔断指标配置',
    'env.groupCircuitBreakerDesc': '控制调度指标窗口与失败率阈值。',
    'env.fieldMetricsWindowSize': '滑动窗口大小',
    'env.fieldMetricsFailureThreshold': '失败率阈值',
    'env.runtimeCbTitle': '调校台',
    'env.runtimeCbDesc': '精调熔断与流控参数，修改立即生效',
    'env.runtimeCbWindowSize': '滑动窗口大小',
    'env.runtimeCbWindowSizeDesc': '计算失败率的最近请求数（3-100）',
    'env.runtimeCbFailureThreshold': '失败率阈值',
    'env.runtimeCbFailureThresholdDesc': '超过此值触发熔断（0.01-1.0）',
    'env.runtimeCbConsecutiveFailures': '连续失败阈值',
    'env.runtimeCbConsecutiveFailuresDesc': '连续可重试失败次数达此值触发熔断（1-100）',
    'env.runtimeCbStreamFirstContentTimeout': '首字等待超时',
    'env.runtimeCbStreamFirstContentTimeoutDesc': 'HTTP 200 后等待首个有效内容的时间（0=禁用, 1000-300000ms）',
    'env.runtimeCbStreamInactivityTimeout': '首字后断流超时',
    'env.runtimeCbStreamInactivityTimeoutDesc': '首字后等待后续活动的时间（0=禁用, 1000-180000ms）',
    'env.runtimeCbStreamToolCallIdleTimeout': '工具调用空闲超时',
    'env.runtimeCbStreamToolCallIdleTimeoutDesc': '工具调用 pending 时等待有效输出的时间（1000-180000ms）',
    'env.runtimeCbPresetGentle': '温和',
    'env.runtimeCbPresetBalanced': '均衡',
    'env.runtimeCbPresetAggressive': '激进',
    'env.runtimeCbPresetCustom': '自定义',
    'env.runtimeCbSaved': '调校台设置已保存，修改立即生效',
    'env.runtimeCbSaveFailed': '保存失败：{error}',
    'env.runtimeCbLoadFailed': '加载失败：{error}',
    'env.runtimeCbNoBackend': '后端服务未运行，请先启动服务',
    'env.runtimeCbServiceStopped': '服务已停止，启动后可配置运行时参数',
    'env.historicalImageTurnLimitTitle': '历史图片轮次限制',
    'env.historicalImageTurnLimitLabel': '图片上下文最大历史轮次',
    'env.historicalImageTurnLimitHint': '发送图片上下文时包含的历史对话轮次上限。0 = 不限制。',
    'env.historicalImageTurnLimitSaved': '历史图片轮次限制已保存',
    'env.groupMetricsPersistence': '指标持久化配置',
    'env.groupMetricsPersistenceDesc': '控制 SQLite 指标持久化与数据保留。',
    'env.fieldMetricsPersistenceEnabled': '启用指标持久化',
    'env.fieldMetricsRetentionDays': '指标保留天数',
    'logs.searchPlaceholder': '搜索日志...',
    'logs.autoScroll': '自动滚动到底部',
    'logs.copied': '已复制！',
    'logs.copyAll': '复制全部日志',
    'logs.clear': '清空日志控制台',
    'logs.noSearchResults': '未找到匹配的日志行',
    'logs.empty': '暂无日志输出，启动服务后即可查看',
    'diagnostic.binaryTitle': '二进制文件未找到',
    'diagnostic.binarySuggestionBuild': '确认 CCX 二进制已构建: cd backend-go && make build',
    'diagnostic.binarySuggestionCheckDataDir': '检查 Desktop 数据目录中是否存在 ccx-go / ccx-go.exe',
    'diagnostic.binarySuggestionDownload': '首次使用需先构建后端，或从 Release 页面下载预编译版本',
    'diagnostic.portTitle': '端口冲突',
    'diagnostic.portSuggestionInstance': '检查是否有其他 CCX 实例已在运行',
    'diagnostic.portSuggestionEnv': '修改 .env 中 PORT 字段使用其他端口',
    'diagnostic.portSuggestionInspect': '使用 lsof -i :3688 (macOS/Linux) 或 netstat -ano | findstr :3688 (Windows) 检查端口占用',
    'diagnostic.healthTitle': '健康检查超时',
    'diagnostic.healthSuggestionLogs': '查看日志面板中是否有启动错误信息',
    'diagnostic.healthSuggestionEnv': '检查 .env 配置是否有语法错误',
    'diagnostic.healthSuggestionChannels': '确认上游渠道配置正确，首次启动可能需要较长时间',
    'diagnostic.healthSuggestionRestart': '尝试手动重启服务',
    'diagnostic.permissionTitle': '权限不足',
    'diagnostic.permissionSuggestionDataDir': '检查数据目录是否有写入权限',
    'diagnostic.permissionSuggestionExecutable': 'macOS/Linux: 确认二进制文件有执行权限 (chmod +x)',
    'diagnostic.permissionSuggestionWindows': 'Windows: 尝试以管理员身份运行',
    'diagnostic.genericTitle': '启动失败',
    'diagnostic.genericSuggestionLogs': '查看下方日志面板获取详细错误信息',
    'diagnostic.genericSuggestionRestart': '尝试重启服务',
    'setup.errorEmptyKey': 'PROXY_ACCESS_KEY 不能为空',
    'env.saveSuccessHint': '.env 已保存，重启服务后生效',
    'env.openedInEditor': '已在编辑器中打开 .env 文件',
    'sidebar.versionHintStore': 'Microsoft Store 版本由 Store 自动更新',
    'sidebar.versionHintTray': '通过托盘菜单检查更新',
    'sidebar.versionClickCheck': '点击检查更新',
    'sidebar.updateAvailable': '新版 {version}',
    'sidebar.updateAvailableHint': '点击查看发布说明',
    'sidebar.theme': '主题',
    'sidebar.themeAuto': '跟随系统',
    'sidebar.themeLight': '亮色',
    'sidebar.themeDark': '暗色',
    'console.channelsTab': '渠道管理',
    'console.conversationsTab': '驾驶舱',
    'console.nativeChannelsEyebrow': '原生渠道编排',
    'console.nativeChannelsTitle': 'WebUI 对齐协议工作区',
    'console.quickAdd': '快速添加',
    'console.quickAddEyebrow': '渠道预设',
    'console.quickAddTitle': '快速添加渠道',
    'console.searchChannels': '搜索渠道...',
    'console.actions.refresh': '刷新',
    'console.pingAll': '批量测速',
    'console.addChannel': '添加渠道',
    'console.noChannels': '暂无渠道，点击上方按钮添加',
    'console.noSearchResults': '没有匹配的渠道',
    'console.mode.multi': '多渠道',
    'console.mode.single': '单渠道',
    'console.pool.active': '故障转移序列',
    'console.pool.current': '当前渠道',
    'console.pool.inactive': '备用池',
    'console.channelStatus.active': '活跃',
    'console.channelStatus.suspended': '暂停',
    'console.channelStatus.disabled': '备用',
    'console.channel.cacheWriteHigh': '缓存写偏高',
    'console.channel.cacheWriteHighHint': '过去 15 分钟缓存写入占比偏高，可能存在缓存命中配置不合理的问题。',
    'console.circuit.open': '熔断开启',
    'console.circuit.halfOpen': '半开',
    'console.keys.active': '活跃密钥',
    'console.actions.label': '操作',
    'console.actions.edit': '编辑渠道',
    'console.actions.capability': '能力测试',
    'console.actions.logs': '查看日志',
    'console.actions.copy': '复制配置',
    'console.actions.website': '访问网站',
    'console.actions.ping': '测速',
    'console.actions.enable': '启用',
    'console.actions.suspend': '暂停',
    'console.actions.resume': '恢复',
    'console.actions.promote': '提升优先级',
    'console.actions.disable': '移至备用池',
    'console.actions.delete': '删除渠道',
    'console.actions.resetCircuit': '重置熔断器',
    'orchestration.moveTop': '置顶',
    'orchestration.moveBottom': '置底',
    'orchestration.keepOne': '(至少保留一个)',
    'orchestration.deleteActiveGuard': '无法删除：故障转移序列中至少需要保留一个活跃渠道',
    'console.keys.disabled': '已禁用密钥',
    'console.form.addChannel': '添加渠道',
    'console.form.editChannel': '编辑渠道',
    'console.form.basicInfo': '基础信息',
    'console.form.name': '名称',
    'console.form.nameRequired': '渠道名称必填',
    'console.form.serviceType': '服务类型',
    'console.form.selectServiceType': '选择服务类型',
    'console.form.serviceTypeRequired': '请选择服务类型',
    'console.form.description': '描述',
    'console.form.connection': '连接',
    'console.form.baseUrl': 'Base URL',
    'console.form.baseUrlRequired': '至少需要一个 Base URL',
    'console.form.additionalUrls': '额外 URL（每行一个）',
    'console.form.proxyUrl': '代理 URL',
    'console.form.routePrefix': '路由前缀',
    'console.form.insecureSkipVerify': '跳过 TLS 验证',
    'console.form.authentication': '认证',
    'console.form.apiKeys': 'API Keys（每行一个）',
    'console.form.disabledKeys': '已禁用密钥',
    'console.form.restoreKey': '恢复',
    'console.form.historicalKeys': '历史密钥记录',
    'addChannel.oneClickSetup': '一键配置',
    'addChannel.expectedRequest': '预期请求',
    'console.form.apiKeyRequired': '至少需要一个 API Key',
    'console.form.reasoningEffort': '思考强度',
    'console.form.noVisionModels': '无视觉模型（每行一个）',
    'console.form.visionFallbackModel': '视觉回退模型',
    'console.form.fetchModels': '获取模型列表',
    'console.form.fetchingModels': '拉取中...',
    'console.form.modelFetchNeedsConfig': '需要 Base URL 和 API Key 才能获取模型列表',
    'console.form.visionEnabled': '视觉已启用',
    'console.form.visionDisabled': '视觉已禁用',
    'console.form.models': '模型',
    'console.form.modelMapping': '模型映射（JSON）',
    'console.form.supportedModels': '支持的模型（每行一个，留空=全部）',
    'console.form.protocolOptions': '协议与模型高级选项',
    'console.form.reasoningParamStyle': 'Reasoning 参数风格',
    'console.form.textVerbosity': 'Text verbosity',
    'console.form.advancedFlags': '高级选项',
    'console.form.noVision': '禁用视觉',
    'console.form.passbackReasoning': '回传推理内容',
    'console.form.passbackThinking': '回传思考块',
    'console.form.fastMode': '快速模式',
    'console.form.lowQuality': '低质量标记',
    'console.form.injectDummySignature': '注入假思考签名',
    'console.form.stripThoughtSignature': '移除思考签名',
    'console.form.stripEmptyBlocks': '移除空文本块',
    'console.form.normalizeSystem': '规范化系统角色',
    'console.form.normalizeUserId': '规范化用户 ID',
    'console.form.stripBillingHeader': '移除 CCH 计费参数',
    'console.form.normalizeChatRoles': '规范化 Chat 角色',
    'console.form.autoBlacklist': '自动黑名单余额异常 Key',
    'console.form.codexNativeTools': 'Codex 原生工具透传',
    'console.form.codexCompat': 'Codex 工具兼容',
    'console.form.stripCodexTools': '移除 Codex 客户端工具',
    'console.form.customHeaders': '自定义 Headers（JSON）',
    'console.form.requestTimeoutMs': '请求超时（ms）',
    'console.form.requestTimeoutInvalid': '请求超时必须是正整数毫秒',
    'console.form.streamTimeouts': '流式超时',
    'console.form.streamFirstContentTimeoutOverrideLabel': '自定义首字等待超时',
    'console.form.streamFirstContentTimeoutLabel': '首字等待超时',
    'console.form.streamInactivityTimeoutOverrideLabel': '自定义断流超时',
    'console.form.streamInactivityTimeoutLabel': '首字后断流超时',
    'console.form.streamToolCallIdleTimeoutOverrideLabel': '自定义工具调用空闲超时',
    'console.form.streamToolCallIdleTimeoutLabel': '工具调用空闲超时',
    'console.form.streamTimeoutOverrideHint': '自定义值会覆盖本渠道的全局流式超时。',
    'console.form.streamTimeoutInheritHint': '关闭表示本渠道继承全局流式超时。',
    'console.form.cancel': '取消',
    'console.form.create': '创建',
    'console.form.save': '保存',
    'console.form.website': '网站',
    'console.form.baseUrlPlaceholder': '每行一个，第一行为主地址，其余作为故障转移\nhttps://api.example.com\nhttps://backup.example.com',
    'console.form.sectionBasic': '基础配置',
    'console.form.sectionAuth': '认证管理',
    'console.form.sectionRedirect': '模型重定向',
    'console.form.sectionAdvanced': '高级选项',
    'console.form.sectionHeaders': '自定义参数',
    'console.form.outline': '配置大纲',
    'console.form.compatibilityTitle': '协议规范化',
    'console.form.runtimeTitle': '运行期策略',
    'console.form.transportTitle': '代理路由网络',
    'console.form.rateLimitTitle': '上游主动限速流量阀',
    'console.form.streamTimeoutTitle': '流式断流超时控制',
    'console.form.fastModeHint': '优先选取低延迟的轻量边缘路由链路',
    'console.form.textVerbosityStyle': '文本详细程度样式',
    'console.form.visionTitle': '视觉控制',
    'console.form.selectDefaultLabel': '默认',
    'console.form.autoBlacklistBalanceLabel': '自动熔断/黑名单余额异常 Key',
    'console.form.autoBlacklistBalanceHint': '自动禁用余额异常的 Key',
    'console.form.rateLimitAutoLabel': '自动学习上游限速',
    'console.form.rateLimitAutoHint': '解析 Retry-After / x-ratelimit-* 响应头动态调整 cooldown',
    'console.form.proxyUrlLabel': '代理通道 URL',
    'console.form.requestTimeoutLabel': '请求超时 (ms)',
    'console.form.routePrefixLabel': '接口路由前缀',
    'console.form.rpmLabel': '每分钟最大请求量 (RPM)',
    'console.form.rpmPlaceholder': '不设限制',
    'console.form.windowLabel': '检测窗口滑动时间 (s)',
    'console.form.windowPlaceholder': '60 秒',
    'console.form.maxConcurrentLabel': '全双工最大并发数限制',
    'console.form.maxConcurrentPlaceholder': '不设限制',
    'console.form.presetGentle': '温和',
    'console.form.presetBalanced': '均衡',
    'console.form.presetAggressive': '激进',
    'console.form.firstByteWait': '首字等待',
    'console.form.firstByteWaitHint': '未响应则自动断开',
    'console.form.idleAfterFirstByte': '首字后断流',
    'console.form.idleAfterFirstByteHint': '生成中途卡顿超时',
    'console.form.toolCallIdle': '工具调用空闲',
    'console.form.toolCallIdleHint': 'FunctionCall 延迟',
    'console.form.timeoutThreshold': '超时阈值:',
    'console.form.supportedModelsLabel': '限定可支持模型范围（白名单模式，留空表示不限制）',
    'console.form.basicInfoTitle': '基础信息',
    'console.form.connectionTitle': '连接终点',
    'console.form.multiLineFailover': '多行实现故障轮换',
    'console.form.expectedEndpoint': '预期终点:',
    'console.form.descriptionPlaceholder': '可选填，用于备注此节点供应商...',
    'console.form.mappingHint': '拦截调用请求中的 Source 别名并定向投递至上游 Target 真实模型',
    'console.form.visionFallbackHint': '已通过模型重定向行的视觉开关标记禁用视觉模型；这些模型遇到图像输入时会自动切换到此模型处理',
    'console.form.modelRedirect': '模型重定向',
    'console.form.modelMappingExisting': '已配置',
    'console.form.modelMappingAdd': '添加新重定向',
    'console.form.generationParams': '生成参数',
    'console.form.textVerbosityPlaceholder': '默认',
    'console.form.modelScope': '模型范围',
    'console.form.selectDefault': '默认',
    'console.form.noVisionHint': '启用后，包含图片的请求将跳过此渠道并 failover 到下一个渠道',
    'console.form.historicalImageTurnLimit': '历史图片轮次限制',
    'console.form.historicalImageTurnLimitHint': '0 = 继承全局；后端会对 >0 的值应用最低 3 约束',
    'console.form.compactModel': 'Compact 专用模型',
    'console.form.compactModelHint': '本地 compact（压缩摘要）时使用的上游模型名，不经过模型映射。对非 Responses 协议渠道生效；对 Responses 渠道，在原生 compact 失败时回退使用。',
    'console.form.passbackReasoningHint': '将 thinking 块转为 reasoning_content 回传，兼容 mimo 等要求 OpenAI 风格 reasoning_content 的 Claude 协议上游',
    'console.form.passbackThinkingHint': '将真实 reasoning_content 投影为 Claude 的 content[].thinking，兼容 DeepSeek/GLM 等严格 thinking mode 上游',
    'console.form.stripThoughtSignatureHint': '移除 functionCall 的 thought_signature 字段，兼容不支持该字段的旧版 Gemini API',
    'console.form.injectDummySignatureHint': '为 functionCall 注入 dummy signature，兼容需要该字段的第三方 API（官方 API 请关闭）',
    'console.form.reasoningParamStyleHint': '选择 OpenAI 风格上游请求使用 reasoning.effort 还是 reasoning_effort。',
    'console.form.codexNativeToolsHint': '透传模式下将 Codex 原生工具（apply_patch、namespace 等）转换为 OpenAI function 格式，使上游模型可调用。',
    'console.form.codexCompatHint': '启用 Codex CLI 兼容：Responses 透传上游会剥离客户端专属工具，Chat/Claude/Gemini 上游会转换为 function 代理工具。',
    'console.form.stripEmptyBlocksHint': '转发前移除裸空 text content block，兼容严格拒绝 Claude Code tool_use 占位块的 Claude 协议上游',
    'console.form.normalizeSystemHint': '针对 Opus 4.8 / Fable 5 等新客户端将 system 作为消息 role 发送的情况：转发前抽回顶层 system 字段，兼容仅支持 user/assistant role 的旧 Claude 上游',
    'console.form.normalizeUserIdHint': '自动将 JSON 对象格式的 user_id 转换为扁平字符串，确保上游兼容性。',
    'console.form.stripBillingHeaderHint': '转发前从 system 文本块中移除 cch= 计费参数，仅对当前 Messages 渠道生效。',
    'console.form.normalizeChatRolesHint': '将 developer 等非标准 role 统一转为 user 后转发给上游。国内模型通常不支持非标准 role，建议开启。',
    'console.form.lowQualityHint': '启用后强制本地估算 token 数量，偏差超过 5% 时使用本地值',
    'console.form.autoBlacklistHint': '当上游返回余额不足时，自动将该 Key 移入拉黑列表。',
    'console.form.insecureSkipVerifyHint': '仅在自签名或域名不匹配时临时启用，生产环境请关闭',
    'console.form.proxyUrlHint': '支持 HTTP/HTTPS/SOCKS5 代理，用于通过代理访问上游服务',
    'console.form.routePrefixHint': '通过 /{前缀}/v1/messages 访问此渠道，多个渠道可共享同一前缀',
    'console.form.requestTimeoutMsHint': '仅作用于非流式上游请求；留空表示继承全局 REQUEST_TIMEOUT。',
    'console.form.rateLimitSectionLabel': '主动限速',
    'console.form.rateLimitSectionHint': '在请求发往上游前主动限流，避免触发上游 429。',
    'console.form.rateLimitRpmLabel': '每分钟请求数',
    'console.form.rateLimitRpmHint': '渠道每分钟最多发往上游的请求数；留空或 0 表示不限。',
    'console.form.rateLimitBurstLabel': '突发容量',
    'console.form.rateLimitBurstHint': '令牌桶容量；留空时自动取每分钟请求数。',
    'console.form.rateLimitMaxConcurrentLabel': '最大并发数',
    'console.form.rateLimitMaxConcurrentHint': '同时进行的上游请求数上限；留空或 0 表示不限。',
    'console.form.rateLimitAutoFromHeadersLabel': '自动学习限流头',
    'console.form.rateLimitAutoFromHeadersHint': '解析上游 Retry-After / x-ratelimit-* 响应头，动态冷却。',
    'addChannel.duplicateKey': '重复',
    'addChannel.addNewApiKeyPlaceholder': '输入新 API Key，回车添加',
    'addChannel.quickMode': '快速粘贴',
    'addChannel.quickHint': '粘贴 Base URL、API Key 或完整配置片段，自动识别并填入表单。',
    'addChannel.applyDetected': '应用识别结果',
    'addChannel.noneDetected': '未识别',
    'addChannel.fullSubtitle': '配置渠道、密钥、模型与高级协议选项',
    'addChannel.editSubtitle': '修改渠道配置信息',
    'console.conversations.search': '搜索会话...',
    'console.conversations.allKinds': '所有类型',
    'console.conversations.refresh': '刷新',
    'console.conversations.total': '共 {count} 个会话',
    'console.conversations.empty': '暂无活跃会话',
    'console.conversations.noSearchResults': '没有匹配的会话',
    'console.conversations.overrideTitle': '会话渠道覆盖',
    'console.conversations.overrideHint': '选择该会话使用的渠道序列（按优先级排列）：',
    'console.conversations.overrideSequence': '覆盖序列',
    'console.conversations.overrideRequired': '请至少选择一个渠道',
    'console.conversations.saveOverride': '保存覆盖',
    'console.conversations.noChannelsForKind': '该类型暂无可用渠道',
    'console.conversations.active': '活跃: {count}',
    'console.conversations.override': '覆盖: {count}',
    'console.conversations.duration.default': '30 分钟（默认）',
    'console.conversations.duration.15min': '15 分钟',
    'console.conversations.duration.1hour': '1 小时',
    'console.conversations.duration.2hours': '2 小时',
    'console.conversations.duration.4hours': '4 小时',
    'console.conversations.duration.8hours': '8 小时',
    'console.conversations.duration.12hours': '12 小时',
    'console.conversations.duration.24hours': '24 小时',
    'console.conversations.duration.never': '永不恢复',
    'console.capability.title': '能力测试',
    'console.capability.start': '开始测试',
    'console.capability.cancel': '取消',
    'console.capability.running': '测试进行中...',
    'console.capability.noResults': '尚未进行能力测试',
    'console.capability.lastResults': '上次测试结果',
    'console.capability.models': '模型',
    'console.capability.duration': '总耗时',
    'console.capability.protocolResults': '协议结果',
    'console.capability.compatible': '兼容协议',
    'capability.title': '能力测试',
    'capability.loadingTitle': '正在测试协议兼容性...',
    'capability.startTest': '开始测试',
    'capability.cancel': '取消',
    'capability.cancelling': '取消中...',
    'capability.cancelled': '已取消',
    'capability.partial': '部分成功',
    'capability.noCompatibleProtocols': '无兼容协议',
    'capability.modelQueued': '模型排队中',
    'capability.protocolRunning': '协议测试中',
    'capability.rpmLabel': 'RPM',
    'capability.progressSummary': '{done}/{total} 模型已完成',
    'capability.supported': '支持',
    'capability.unsupported': '不支持',
    'capability.table.protocol': '协议',
    'capability.table.status': '状态',
    'capability.table.successCount': '成功',
    'capability.table.latency': '延迟',
    'capability.table.streaming': 'SSE',
    'capability.table.actions': '操作',
    'capability.copyToTab': '复制到当前 Tab',
    'capability.currentTab': '当前 Tab',
    'capability.convert': '转换',
    'capability.modelsLabel': '模型',
    'capability.modelDetailsUnavailable': '模型详情暂不可用',
    'capability.tooltipLatency': '延迟',
    'capability.tooltipStreaming': '流式',
    'capability.modelStatus': '状态',
    'capability.actualModel': '实际模型',
    'capability.retryModel': '重试',
    'capability.testModel': '测试',
    'capability.runMode.reusedRunning': '复用运行',
    'capability.runMode.resumedCancelled': '恢复取消',
    'capability.runMode.cacheHit': '缓存命中',
    'capability.runMode.reusedPrevious': '复用上次结果',
    'capability.lastResults': '上次测试结果',
    'capability.noResults': '尚未进行能力测试',
    'capability.models': '模型',
    'capability.duration': '总耗时',
    'capability.snapshotUpdated': '更新时间',
    'console.logs.title': '频道日志',
    'console.logs.time': '时间',
    'console.logs.model': '模型',
    'console.logs.baseUrl': 'URL',
    'console.logs.key': 'Key',
    'console.logs.statusCode': '状态码',
    'console.logs.duration': '耗时',
    'console.logs.error': '错误',
    'console.logs.entries': '条',
    'console.logs.empty': '暂无日志',
    'console.fuzzyEnabled': 'Fuzzy 模式已开启',
    'console.fuzzyDisabled': 'Fuzzy 模式已关闭',
    'console.fuzzyLoadFailed': '加载 Fuzzy 模式状态失败',
    'console.cbSettings': '调校台',
    'chart.totalRequests': '总请求数',
    'chart.successRate': '成功率',
    'chart.inputTokens': '输入 Token',
    'chart.outputTokens': '输出 Token',
    'chart.cacheRw': 'Cache R/W',
    'chart.noData': '暂无请求数据',
    'channelEditor.title.create': '添加新渠道',
    'channelEditor.title.edit': '编辑渠道',
    'channelEditor.title.editSubtitle': '修改渠道配置',
    'channelEditor.nav.basic': '基础信息',
    'channelEditor.nav.auth': '认证管理',
    'channelEditor.nav.redirect': '模型重定向',
    'channelEditor.nav.advanced': '高级选项',
    'channelEditor.nav.custom': '自定义参数',
    'channelEditor.basic.name.label': '渠道名称',
    'channelEditor.basic.name.placeholder': '例如 OpenAI 生产环境',
    'channelEditor.basic.name.required': '渠道名称不能为空',
    'channelEditor.basic.serviceType.label': '服务类型',
    'channelEditor.basic.serviceType.required': '请选择服务类型',
    'channelEditor.basic.baseUrl.label': 'Base URL',
    'channelEditor.basic.baseUrl.placeholder': 'https://api.openai.com/v1',
    'channelEditor.basic.baseUrl.required': '至少需要一个 Base URL',
    'channelEditor.basic.website.label': '官方网站',
    'channelEditor.basic.website.placeholder': 'https://openai.com',
    'channelEditor.basic.description.label': '描述',
    'channelEditor.basic.description.placeholder': '可选填，用于备注此节点供应商...',
    'channelEditor.auth.keys.label': 'API Keys',
    'channelEditor.auth.apiKeyRequired': '至少需要一个 API Key',
    'channelEditor.auth.duplicateKey': '重复的 API Key',
    'channelEditor.auth.addNewApiKey.placeholder': '输入新的 API Key',
    'channelEditor.mapping.redirect.title': '模型重定向',
    'channelEditor.mapping.configured.label': '已配置',
    'channelEditor.mapping.source.label': '源模型',
    'channelEditor.mapping.source.placeholder': 'gpt-4o',
    'channelEditor.mapping.target.label': '目标模型',
    'channelEditor.mapping.target.placeholder': 'claude-sonnet-4-6',
    'channelEditor.mapping.reasoningEffort.label': '推理强度',
    'channelEditor.actions.create': '创建渠道',
    'channelEditor.actions.save': '保存',
    'channelEditor.actions.cancel': '取消',
  },
}
