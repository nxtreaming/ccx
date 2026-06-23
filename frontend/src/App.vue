<template>
  <v-app>
    <!-- 自动认证加载提示 - 只在真正进行自动认证时显示 -->
    <v-overlay
      :model-value="authStore.isAutoAuthenticating && !authStore.isInitialized"
      persistent
      class="align-center justify-center"
      scrim="black"
    >
      <v-card class="pa-6 text-center" max-width="400" rounded="lg">
        <v-progress-circular indeterminate :size="64" :width="6" color="primary" class="mb-4" />
        <div class="text-h6 mb-2">{{ t('app.auth.verifyingTitle') }}</div>
        <div class="text-body-2 text-medium-emphasis">{{ t('app.auth.verifyingBody') }}</div>
      </v-card>
    </v-overlay>

    <!-- 认证界面 -->
    <v-dialog v-model="showAuthDialog" persistent max-width="500">
      <v-card class="pa-4">
        <v-card-title class="text-h5 text-center mb-4"> 🔐 API Proxy - CCX </v-card-title>

        <v-card-text>
          <v-alert v-if="authStore.authError" type="error" variant="tonal" class="mb-4">
            {{ authStore.authError }}
          </v-alert>

          <v-form @submit.prevent="handleAuthSubmit">
            <v-text-field
              v-model="authStore.authKeyInput"
              :label="t('app.auth.inputLabel')"
              type="password"
              variant="outlined"
              prepend-inner-icon="mdi-key"
              :rules="[(v: string) => !!v || t('app.auth.inputRequired')]"
              required
              autofocus
              @keyup.enter="handleAuthSubmit"
            />

            <v-btn type="submit" color="primary" block size="large" class="mt-4" :loading="authStore.authLoading">
              {{ t('app.auth.submit') }}
              <span class="shortcut-hint ml-2 text-xs opacity-50">Enter</span>
            </v-btn>
          </v-form>

          <v-divider class="my-4" />

          <v-alert type="info" variant="tonal" density="compact" class="mb-0" :icon="false">
            <div class="text-body-2">
              <p class="mb-2"><strong>🔒 {{ t('app.auth.securityTitle') }}</strong></p>
              <ul class="ml-4 mb-0">
                <li>{{ t('app.auth.securityItem1') }}</li>
                <li>{{ t('app.auth.securityItem2') }}</li>
                <li>{{ t('app.auth.securityItem3') }}</li>
                <li>{{ t('app.auth.securityItem4') }}</li>
                <li>{{ t('app.auth.securityItem5', { attempts: MAX_AUTH_ATTEMPTS }) }}</li>
              </ul>
            </div>
          </v-alert>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- 应用栏 - 毛玻璃效果 -->
    <v-app-bar elevation="0" :height="$vuetify.display.mobile ? 56 : 72" class="app-header">
      <template #prepend>
        <div class="app-logo d-flex align-center justify-center pa-0 overflow-hidden">
          <!-- 显著放大 Logo 尺寸（手机端 32px，电脑端 44px）让流转动画和发光更清晰夺目 -->
          <Logo :size="$vuetify.display.mobile ? 32 : 44" />
        </div>
      </template>

      <!-- 自定义标题容器 - 替代 v-app-bar-title -->
      <div class="header-title">
        <!-- 移动/平板端：下拉菜单（宽度小于 1000px 时进行折叠，防止菜单过窄挤压） -->
        <v-menu v-if="$vuetify.display.width < 1000">
          <template #activator="{ props: menuProps }">
            <v-btn
              v-bind="menuProps"
              variant="text"
              class="mobile-tab-selector text-body-2 font-weight-bold"
              append-icon="mdi-chevron-down"
            >
              {{ route.path === '/conversations' ? t('app.tabs.conversations') : translatedApiTabOptions.find(tab => tab.value === channelStore.activeTab)?.label }}
            </v-btn>
          </template>
          <v-list density="compact" nav>
            <v-list-item
              v-for="tab in translatedApiTabOptions"
              :key="tab.value"
              :active="tab.value === 'conversations' ? route.path === '/conversations' : channelStore.activeTab === tab.value"
              :to="tab.route"
            >
              <v-list-item-title>{{ tab.label }}</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>

        <!-- 桌面端：平铺链接 -->
        <div v-else class="text-h6 font-weight-bold d-flex align-center">
          <router-link to="/channels/messages" class="api-type-text" :class="{ active: channelStore.activeTab === 'messages' && route.path !== '/conversations' }">
            {{ t('app.tabs.messages') }}
          </router-link>
          <span class="api-type-text separator">/</span>
          <router-link to="/channels/chat" class="api-type-text" :class="{ active: channelStore.activeTab === 'chat' && route.path !== '/conversations' }">
            {{ t('app.tabs.chat') }}
          </router-link>
          <span class="api-type-text separator">/</span>
          <router-link to="/channels/images" class="api-type-text" :class="{ active: channelStore.activeTab === 'images' && route.path !== '/conversations' }">
            {{ t('app.tabs.images') }}
          </router-link>
          <span class="api-type-text separator">/</span>
          <router-link to="/channels/responses" class="api-type-text" :class="{ active: channelStore.activeTab === 'responses' && route.path !== '/conversations' }">
            {{ t('app.tabs.responses') }}
          </router-link>
          <span class="api-type-text separator">/</span>
          <router-link to="/channels/gemini" class="api-type-text" :class="{ active: channelStore.activeTab === 'gemini' && route.path !== '/conversations' }">
            {{ t('app.tabs.gemini') }}
          </router-link>
          <span class="api-type-text separator">/</span>
          <router-link to="/conversations" class="api-type-text" :class="{ active: route.path === '/conversations' }">
            {{ t('app.tabs.conversations') }}
          </router-link>
          <span class="brand-text d-none d-md-inline">API Proxy - CCX</span>
        </div>
      </div>

      <v-spacer/>

      <!-- 版本信息（< 500px 隐藏，避免在窄屏挤压右侧按钮） -->
      <div
        v-if="!isDesktopWebUI && $vuetify.display.width >= 500 && systemStore.versionInfo.currentVersion"
        class="version-badge"
        :class="{
          'version-clickable': systemStore.versionInfo.status === 'update-available' || systemStore.versionInfo.status === 'latest',
          'version-checking': systemStore.versionInfo.status === 'checking',
          'version-latest': systemStore.versionInfo.status === 'latest',
          'version-update': systemStore.versionInfo.status === 'update-available'
        }"
        @click="handleVersionClick"
      >
        <v-icon
          v-if="systemStore.versionInfo.status === 'checking'"
          size="14"
          class="mr-1"
        >mdi-clock-outline</v-icon>
        <v-icon
          v-else-if="systemStore.versionInfo.status === 'latest'"
          size="14"
          class="mr-1"
          color="success"
        >mdi-check-circle</v-icon>
        <v-icon
          v-else-if="systemStore.versionInfo.status === 'update-available'"
          size="14"
          class="mr-1"
          color="warning"
        >mdi-alert</v-icon>
        <span class="version-text">{{ systemStore.versionInfo.currentVersion }}</span>
        <template v-if="systemStore.versionInfo.status === 'update-available' && systemStore.versionInfo.latestVersion">
          <span class="version-arrow mx-1">→</span>
          <span class="version-latest-text">{{ systemStore.versionInfo.latestVersion }}</span>
        </template>
      </div>

      <!-- 语言切换 -->
      <v-menu location="bottom end">
        <template #activator="{ props: menuProps }">
          <v-btn
            v-bind="menuProps"
            icon
            variant="text"
            size="small"
            class="header-btn language-switch-btn"
          >
            <span class="language-switch-label">{{ currentLanguageShortLabel }}</span>
          </v-btn>
        </template>
        <v-list density="compact" nav>
          <v-list-item
            v-for="option in languageOptions"
            :key="option.value"
            :active="currentLocale === option.value"
            @click="setLocale(option.value)"
          >
            <v-list-item-title>{{ option.label }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>

      <!-- 新用户指引 -->
      <v-btn
        icon
        variant="text"
        size="small"
        class="header-btn"
        :title="t('guide.helpButton')"
        @click="openGuide"
      >
        <v-icon size="20">mdi-help-circle</v-icon>
      </v-btn>

      <!-- 暗色模式切换 -->
      <v-btn icon variant="text" size="small" class="header-btn" @click="toggleDarkMode">
        <v-icon size="20">{{
          theme.global.current.value.dark ? 'mdi-weather-night' : 'mdi-white-balance-sunny'
        }}</v-icon>
      </v-btn>

      <!-- 注销按钮 -->
      <v-btn
        v-if="isAuthenticated"
        icon
        variant="text"
        size="small"
        class="header-btn"
        :title="t('app.header.logout')"
        @click="handleLogout"
      >
        <v-icon size="20">mdi-logout</v-icon>
      </v-btn>
    </v-app-bar>


    <!-- 主要内容 -->
    <v-main>
      <v-container fluid class="pa-4 pa-md-6">
        <!-- 全局统计顶部可折叠卡片（根据当前 Tab 显示对应统计） -->
        <v-card v-if="isAuthenticated && route.path !== '/conversations'" class="mb-6 global-stats-panel">
          <div
            class="global-stats-header d-flex align-center justify-space-between px-4 py-2"
            style="cursor: pointer;"
            @click="preferencesStore.toggleGlobalStats()"
          >
            <div class="d-flex align-center">
              <v-icon size="20" class="mr-2">mdi-chart-areaspline</v-icon>
              <span class="text-subtitle-1 font-weight-bold">{{ activeTrafficTitle }}</span>
            </div>
            <v-btn icon size="small" variant="text">
              <v-icon>{{ preferencesStore.showGlobalStats ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
            </v-btn>
          </div>
          <v-expand-transition>
            <div v-if="preferencesStore.showGlobalStats">
              <v-divider />
              <GlobalStatsChart :api-type="channelStore.activeTab" />
            </div>
          </v-expand-transition>
        </v-card>

        <!-- 统计卡片 - 玻璃拟态风格 -->
        <v-row v-if="route.path !== '/conversations'" class="mb-6 stat-cards-row">
          <v-col cols="6" sm="4">
            <div class="stat-card stat-card-info">
              <div class="stat-card-icon">
                <v-icon size="28">mdi-server-network</v-icon>
              </div>
              <div class="stat-card-content">
                <div class="stat-card-value">{{ channelStore.currentChannelsData.channels?.length || 0 }}</div>
                <div class="stat-card-label">{{ t('app.stats.totalChannels') }}</div>
                <div class="stat-card-desc">{{ t('app.stats.totalChannelsDesc') }}</div>
              </div>
              <div class="stat-card-glow"></div>
            </div>
          </v-col>

          <v-col cols="6" sm="4">
            <div class="stat-card stat-card-success">
              <div class="stat-card-icon">
                <v-icon size="28">mdi-check-circle</v-icon>
              </div>
              <div class="stat-card-content">
                <div class="stat-card-value">
                  {{ channelStore.activeChannelCount }}<span class="stat-card-total">/{{ channelStore.failoverChannelCount }}</span>
                </div>
                <div class="stat-card-label">{{ t('app.stats.activeChannels') }}</div>
                <div class="stat-card-desc">{{ t('app.stats.activeChannelsDesc') }}</div>
              </div>
              <div class="stat-card-glow"></div>
            </div>
          </v-col>

          <v-col cols="6" sm="4">
            <div class="stat-card" :class="systemStore.systemStatus === 'running' ? 'stat-card-emerald' : 'stat-card-error'">
              <div class="stat-card-icon" :class="{ 'pulse-animation': systemStore.systemStatus === 'running' }">
                <v-icon size="28">{{ systemStore.systemStatus === 'running' ? 'mdi-heart-pulse' : 'mdi-alert-circle' }}</v-icon>
              </div>
              <div class="stat-card-content">
                <div class="stat-card-value">{{ systemStatusText }}</div>
                <div class="stat-card-label">{{ t('app.stats.systemStatus') }}</div>
                <div class="stat-card-desc">{{ systemStatusDesc }}</div>
              </div>
              <div class="stat-card-glow"></div>
            </div>
          </v-col>
        </v-row>

        <!-- 操作按钮区域 - 现代化设计 -->
        <div v-if="route.path !== '/conversations'" class="action-bar mb-6">
          <div class="action-bar-left">
            <v-btn
              color="primary"
              size="large"
              prepend-icon="mdi-plus"
              class="action-btn action-btn-primary"
              @click="openAddChannelModal"
            >
              {{ t('app.actions.addChannel') }}
            </v-btn>

            <v-btn
              color="info"
              size="large"
              prepend-icon="mdi-speedometer"
              variant="tonal"
              :loading="channelStore.isPingingAll"
              class="action-btn"
              @click="pingAllChannels"
            >
              {{ t('app.actions.ping') }}
            </v-btn>

            <v-btn size="large" prepend-icon="mdi-refresh" variant="text" class="action-btn" @click="refreshChannels">
              {{ t('app.actions.refresh') }}
            </v-btn>
          </div>

          <div class="action-bar-right">
            <!-- Fuzzy 模式切换按钮 -->
            <v-tooltip location="bottom" content-class="ccx-tooltip">
              <template #activator="{ props }">
                <v-btn
                  v-bind="props"
                  variant="tonal"
                  size="large"
                  :loading="systemStore.fuzzyModeLoading"
                  :disabled="systemStore.fuzzyModeLoadError"
                  :color="systemStore.fuzzyModeLoadError ? 'error' : (preferencesStore.fuzzyModeEnabled ? 'warning' : 'default')"
                  class="action-btn"
                  @click="toggleFuzzyMode"
                >
                  <v-icon start size="20">
                    {{ systemStore.fuzzyModeLoadError ? 'mdi-alert-circle-outline' : (preferencesStore.fuzzyModeEnabled ? 'mdi-shield-refresh' : 'mdi-shield-off-outline') }}
                  </v-icon>
                  Fuzzy
                </v-btn>
              </template>
              <span>{{ systemStore.fuzzyModeLoadError ? t('tooltip.loadFailedRefresh') : (preferencesStore.fuzzyModeEnabled ? t('tooltip.fuzzyEnabled') : t('tooltip.fuzzyDisabled')) }}</span>
            </v-tooltip>

            <!-- 熔断器配置按钮 -->
            <v-tooltip location="bottom" content-class="ccx-tooltip">
              <template #activator="{ props }">
                <v-btn
                  v-bind="props"
                  variant="tonal"
                  size="large"
                  color="default"
                  class="action-btn"
                  @click="openCircuitBreakerDialog"
                >
                  <v-icon start size="20">mdi-tune</v-icon>
                  TB
                </v-btn>
              </template>
              <span>{{ t('tooltip.circuitBreakerSettings') }}</span>
            </v-tooltip>

          </div>
        </div>
        <router-view
          @edit="editChannel"
          @delete="deleteChannel"
          @ping="pingChannel"
          @test-capability="testChannelCapability"
          @refresh="refreshChannels"
          @error="showErrorToast"
          @success="showSuccessToast"
        />
      </v-container>
    </v-main>

    <!-- 添加渠道模态框 -->
    <AddChannelModal
      v-model:show="dialogStore.showAddChannelModal"
      :channel-type="channelStore.activeTab"
      @save="saveChannel"
      @error="showErrorToast"
    />

    <!-- 编辑渠道模态框 -->
    <EditChannelModal
      v-model:show="dialogStore.showEditChannelModal"
      :channel="dialogStore.editingChannel"
      :channel-type="channelStore.activeTab"
      @save="saveChannel"
      @test-capability="testChannelCapability"
      @error="showErrorToast"
    />

    <!-- 能力测试对话框 -->
    <CapabilityTestDialog
      ref="capabilityTestDialogRef"
      v-model="showCapabilityTestDialog"
      :channel-name="capabilityTestChannelName"
      :current-tab="channelStore.activeTab"
      :capability-job="capabilityTestJob"
      :capability-rpm="capabilityTestRpm"
      @update:capability-rpm="capabilityTestRpm = $event"
      @copy-to-tab="handleCopyToTab"
      @cancel="handleCancelCapabilityTest"
      @retry-model="handleRetryCapabilityModel"
      @test-protocol="handleTestCapabilityProtocol"
    />

    <!-- OTA 更新对话框 -->
    <UpdateDialog v-model="systemStore.updateDialogOpen" />

    <!-- 新用户指引对话框 -->
    <UserGuideDialog v-model="showGuide" />

    <!-- 熔断器配置对话框 -->
    <v-dialog v-model="circuitBreakerDialogOpen" max-width="640">
      <v-card class="cb-dialog-card">
        <v-card-title class="cb-dialog-title">
          {{ t('dialog.circuitBreaker.title') }}
        </v-card-title>
        <v-divider />
        <v-card-text class="cb-dialog-body">
          <div class="cb-dialog-desc">
            {{ t('dialog.circuitBreaker.description') }}
          </div>

          <!-- 滑块区域 - 三列并排 -->
          <div class="cb-control-grid">
            <!-- 滑动窗口大小 -->
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.windowSize') }}</span>
                <span class="cb-slider-value">{{ cbForm.windowSize }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.windowSize"
                :min="3"
                :max="100"
                step="1"
                class="cb-slider w-100"
                @input="onSliderChange('windowSize', $event)"
              />
              <div class="cb-slider-range">
                <span>3</span><span>100</span>
              </div>
            </div>

            <!-- 失败率阈值 -->
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.failureThreshold') }}</span>
                <span class="cb-slider-value">{{ cbForm.failureThreshold.toFixed(2) }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.failureThreshold"
                :min="0.01"
                :max="1"
                step="0.01"
                class="cb-slider w-100"
                @input="onSliderChange('failureThreshold', $event)"
              />
              <div class="cb-slider-range">
                <span>0.01</span><span>1.00</span>
              </div>
            </div>

            <!-- 连续失败阈值 -->
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.consecutiveFailuresThreshold') }}</span>
                <span class="cb-slider-value">{{ cbForm.consecutiveFailuresThreshold }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.consecutiveFailuresThreshold"
                :min="1"
                :max="100"
                step="1"
                class="cb-slider w-100"
                @input="onSliderChange('consecutiveFailuresThreshold', $event)"
              />
              <div class="cb-slider-range">
                <span>1</span><span>100</span>
              </div>
            </div>
          </div>

          <!-- 上游请求生命周期超时 -->
          <div class="cb-control-grid cb-control-grid--two">
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.requestTimeout') }}</span>
                <span class="cb-slider-value">{{ (cbForm.requestTimeoutMs / 1000) + 's' }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.requestTimeoutMs"
                :min="1000"
                :max="300000"
                step="1000"
                class="cb-slider w-100"
                @input="onSliderChange('requestTimeoutMs', $event)"
              />
              <div class="cb-slider-range">
                <span>1s</span><span>300s</span>
              </div>
            </div>

            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.responseHeaderTimeout') }}</span>
                <span class="cb-slider-value">{{ (cbForm.responseHeaderTimeoutMs / 1000) + 's' }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.responseHeaderTimeoutMs"
                :min="1000"
                :max="300000"
                step="1000"
                class="cb-slider w-100"
                @input="onSliderChange('responseHeaderTimeoutMs', $event)"
              />
              <div class="cb-slider-range">
                <span>1s</span><span>300s</span>
              </div>
            </div>
          </div>

          <!-- 流式健康检测超时 -->
          <div class="cb-control-grid">
            <!-- 首字等待超时 -->
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.streamFirstContentTimeout') }}</span>
                <span class="cb-slider-value">{{ (cbForm.streamFirstContentTimeoutMs / 1000) + 's' }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.streamFirstContentTimeoutMs"
                :min="5000"
                :max="300000"
                step="1000"
                class="cb-slider w-100"
                @input="onSliderChange('streamFirstContentTimeoutMs', $event)"
              />
              <div class="cb-slider-range">
                <span>5s</span><span>300s</span>
              </div>
            </div>

            <!-- 首字后断流超时 -->
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.streamInactivityTimeout') }}</span>
                <span class="cb-slider-value">{{ (cbForm.streamInactivityTimeoutMs / 1000) + 's' }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.streamInactivityTimeoutMs"
                :min="1000"
                :max="180000"
                step="1000"
                class="cb-slider w-100"
                @input="onSliderChange('streamInactivityTimeoutMs', $event)"
              />
              <div class="cb-slider-range">
                <span>1s</span><span>180s</span>
              </div>
            </div>

            <!-- 工具调用空闲超时 -->
            <div class="cb-control">
              <div class="cb-control-header">
                <span class="cb-slider-label">{{ t('dialog.circuitBreaker.streamToolCallIdleTimeout') }}</span>
                <span class="cb-slider-value">{{ (cbForm.streamToolCallIdleTimeoutMs / 1000) + 's' }}</span>
              </div>
              <input
                type="range"
                :value="cbForm.streamToolCallIdleTimeoutMs"
                :min="30000"
                :max="300000"
                step="1000"
                class="cb-slider w-100"
                @input="onSliderChange('streamToolCallIdleTimeoutMs', $event)"
              />
              <div class="cb-slider-range">
                <span>30s</span><span>300s</span>
              </div>
            </div>
          </div>

          <!-- 预设按钮 -->
          <div class="cb-preset-grid">
            <v-btn
              v-for="preset in cbPresets"
              :key="preset.key"
              variant="flat"
              :color="activePreset === preset.key ? 'primary' : 'default'"
              size="small"
              class="cb-preset-btn"
              :class="{ 'cb-preset-active': activePreset === preset.key }"
              @click="applyPreset(preset)"
            >
              {{ t(preset.labelKey) }}
            </v-btn>
          </div>

          <!-- 历史图片轮次限制 -->
          <v-divider class="my-3" />
          <div class="cb-hl-section">
            <div class="cb-hl-copy">
              <div class="cb-hl-header">
                <v-icon size="18" class="cb-hl-icon">mdi-image-sync</v-icon>
                <span class="cb-hl-title">{{ t('dialog.historicalImageTurnLimit.title') }}</span>
              </div>
              <p class="cb-hl-hint">{{ t('dialog.historicalImageTurnLimit.hint') }}</p>
            </div>
            <v-text-field
              v-model.number="historicalImageForm.limit"
              :label="t('dialog.historicalImageTurnLimit.label')"
              type="number"
              min="3"
              variant="outlined"
              density="compact"
              hide-details
              class="cb-hl-input"
            />
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions class="cb-dialog-actions">
          <v-spacer />
          <v-btn variant="outlined" class="cb-dialog-btn" @click="circuitBreakerDialogOpen = false">
            {{ t('app.actions.cancel') }}
            <span class="shortcut-hint ml-2 text-xs opacity-50">Esc</span>
          </v-btn>
          <v-btn color="primary" variant="elevated" class="cb-dialog-btn cb-dialog-btn-primary" :loading="cbSaving" @click="saveCircuitBreaker">
            {{ t('app.actions.confirm') }}
            <span class="shortcut-hint ml-2 text-xs opacity-50">{{ isMac ? '⌘Enter' : 'Ctrl+Enter' }}</span>
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 添加API密钥对话框 -->
    <v-dialog v-model="dialogStore.showAddKeyModal" max-width="500">
      <v-card rounded="lg">
        <v-card-title class="d-flex align-center">
          <v-icon class="mr-3">mdi-key-plus</v-icon>
          {{ t('app.dialog.addApiKeyTitle') }}
        </v-card-title>
        <v-card-text>
          <v-text-field
            v-model="dialogStore.newApiKey"
            :label="t('app.dialog.apiKeyLabel')"
            type="password"
            variant="outlined"
            density="comfortable"
            :placeholder="t('app.dialog.apiKeyPlaceholder')"
            @keyup.enter="addApiKey"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn variant="outlined" @click="dialogStore.closeAddKeyModal()">{{ t('app.actions.cancel') }} <span class="shortcut-hint ml-2 text-xs opacity-50">Esc</span></v-btn>
          <v-btn :disabled="!dialogStore.newApiKey.trim()" color="primary" variant="elevated" @click="addApiKey">{{ t('app.actions.add') }} <span class="shortcut-hint ml-2 text-xs opacity-50">Enter</span></v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 通用确认对话框（替代 window.confirm，兼容 Wails 桌面 iframe） -->
    <v-dialog v-model="dialogStore.showConfirmDialog" max-width="420" persistent>
      <v-card rounded="lg">
        <v-card-title class="d-flex align-center pt-4">
          <v-icon class="mr-3" :color="dialogStore.confirmDialogColor">mdi-alert-circle-outline</v-icon>
          {{ t('app.dialog.confirmTitle') }}
        </v-card-title>
        <v-card-text class="text-body-1 pt-2">{{ dialogStore.confirmDialogMessage }}</v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn variant="text" @click="dialogStore.resolveConfirm(false)">
            {{ dialogStore.confirmDialogCancelText || t('app.actions.cancel') }}
            <span class="shortcut-hint ml-2 text-xs opacity-50">Esc</span>
          </v-btn>
          <v-btn :color="dialogStore.confirmDialogColor" variant="elevated" @click="dialogStore.resolveConfirm(true)">
            {{ dialogStore.confirmDialogConfirmText || t('app.actions.confirm') }}
            <span class="shortcut-hint ml-2 text-xs opacity-50">{{ isMac ? '⌘Enter' : 'Ctrl+Enter' }}</span>
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Toast通知 -->
    <v-snackbar
      v-for="toast in toasts"
      :key="toast.id"
      v-model="toast.show"
      :color="getToastColor(toast.type)"
      :timeout="3000"
      location="top right"
      variant="elevated"
    >
      <div class="d-flex align-center">
        <v-icon class="mr-3">{{ getToastIcon(toast.type) }}</v-icon>
        {{ toast.message }}
      </div>
    </v-snackbar>
  </v-app>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed, watch, defineAsyncComponent } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme } from 'vuetify'
import Logo from './components/Logo.vue'
import { api, fetchHealth, ApiError, type Channel, type CapabilityTestJob, type CapabilityTestJobStartResponse, type CapabilityProtocolJobResult, type CapabilityModelJobResult, type CapabilitySnapshot } from './services/api'
import { versionService } from './services/version'
import { useAuthStore } from './stores/auth'
import { useChannelStore } from './stores/channel'
import { usePreferencesStore } from './stores/preferences'
import { useDialogStore } from './stores/dialog'
import { useSystemStore } from './stores/system'
import { useI18n } from './i18n'
import type { SupportedLocale } from './i18n'
import AddChannelModal from './components/AddChannelModal.vue'
import EditChannelModal from './components/EditChannelModal.vue'
import CapabilityTestDialog from './components/CapabilityTestDialog.vue'
import UpdateDialog from './components/UpdateDialog.vue'
import UserGuideDialog from './components/UserGuideDialog.vue'
// 异步加载图表组件，减少首屏 JS 体积
const GlobalStatsChart = defineAsyncComponent(() => import('./components/GlobalStatsChart.vue'))
import { useAppTheme } from './composables/useTheme'
import { useCapabilityTestManager } from './composables/useCapabilityTestManager'
import { streamTimeoutPresets as sharedStreamPresets } from './utils/streamTimeoutPresets'

// 路由
const route = useRoute()

// Vuetify主题
const theme = useTheme()

// 应用主题系统
const { init: initTheme } = useAppTheme()

// 认证 Store
// 注意：as any 是 Pinia 3.x + Vue 3.5 + TS 6.x 兼容补丁——
// Vue 3.5 将 Ref<T> 改为 Ref<T, S>，Pinia 的 UnwrapRef<Ref<infer V, unknown>> 模式失效，
// 导致模板中访问 store 属性时类型未被自动解包。运行时行为正常。
const authStore = useAuthStore() as any

// 渠道 Store
const channelStore = useChannelStore() as any

// 偏好设置 Store
const preferencesStore = usePreferencesStore() as any

// 对话框 Store
const dialogStore = useDialogStore() as any

// 系统状态 Store
const systemStore = useSystemStore() as any
const { locale, t, setLocale } = useI18n()

const languageOptions: Array<{ value: SupportedLocale, label: string, shortLabel: string }> = [
  { value: 'en', label: 'English', shortLabel: 'EN' },
  { value: 'id', label: 'Bahasa Indonesia', shortLabel: 'ID' },
  { value: 'zh-CN', label: '简体中文', shortLabel: '中' },
]

const currentLocale = computed(() => locale.value)
const currentLanguageShortLabel = computed(() => {
  return languageOptions.find(option => option.value === currentLocale.value)?.shortLabel ?? currentLocale.value.slice(0, 2).toUpperCase()
})

// API 类型 Tab 选项（移动端下拉菜单使用）
const apiTabOptions = [
  { value: 'messages', labelKey: 'app.tabs.messages', route: '/channels/messages', icon: 'mdi-code-braces' },
  { value: 'chat', labelKey: 'app.tabs.chat', route: '/channels/chat', icon: 'mdi-chat-processing-outline' },
  { value: 'images', labelKey: 'app.tabs.images', route: '/channels/images', icon: 'mdi-image-outline' },
  { value: 'responses', labelKey: 'app.tabs.responses', route: '/channels/responses', icon: 'mdi-console' },
  { value: 'gemini', labelKey: 'app.tabs.gemini', route: '/channels/gemini', icon: 'mdi-google' },
  { value: 'conversations', labelKey: 'app.tabs.conversations', route: '/conversations', icon: 'mdi-view-dashboard-outline' },
] as const

const translatedApiTabOptions = computed(() => {
  return apiTabOptions.map(tab => ({
    ...tab,
    label: t(tab.labelKey),
  }))
})
const isDesktopWebUI = new URLSearchParams(window.location.search).get('ccx_desktop') === '1'

const currentTabLabel = computed(() => {
  return translatedApiTabOptions.value.find(tab => tab.value === channelStore.activeTab)?.label || channelStore.activeTab
})

const activeTrafficTitle = computed(() => t('app.stats.trafficTitle', { tab: currentTabLabel.value }))

const systemStatusText = computed(() => {
  switch (systemStore.systemStatus) {
    case 'running':
      return t('system.running')
    case 'error':
      return t('system.error')
    case 'connecting':
      return t('system.connecting')
    default:
      return t('system.unknown')
  }
})

const systemStatusDesc = computed(() => {
  switch (systemStore.systemStatus) {
    case 'running':
      return t('system.runningDesc')
    case 'error':
      return t('system.errorDesc')
    case 'connecting':
      return t('system.connectingDesc')
    default:
      return ''
  }
})

// 对话框状态已迁移到 DialogStore

// 主题和偏好设置已迁移到 PreferencesStore

// 系统状态已迁移到 SystemStore

// Toast通知系统
interface Toast {
  id: number
  message: string
  type: 'success' | 'error' | 'warning' | 'info'
  show?: boolean
}
const toasts = ref<Toast[]>([])
let toastId = 0

// Toast工具函数
const getToastColor = (type: string) => {
  const colorMap: Record<string, string> = {
    success: 'success',
    error: 'error',
    warning: 'warning',
    info: 'info'
  }
  return colorMap[type] || 'info'
}

const getToastIcon = (type: string) => {
  const iconMap: Record<string, string> = {
    success: 'mdi-check-circle',
    error: 'mdi-alert-circle',
    warning: 'mdi-alert',
    info: 'mdi-information'
  }
  return iconMap[type] || 'mdi-information'
}

// 工具函数
const showToast = (message: string, type: 'success' | 'error' | 'warning' | 'info' = 'info') => {
  const toast: Toast = { id: ++toastId, message, type, show: true }
  toasts.value.push(toast)
  setTimeout(() => {
    const index = toasts.value.findIndex(t => t.id === toast.id)
    if (index > -1) toasts.value.splice(index, 1)
  }, 3000)
}

const _handleError = (error: unknown, defaultMessage: string) => {
  const message = error instanceof Error ? error.message : defaultMessage
  showToast(message, 'error')
  console.error(error)
}

// 直接显示错误消息（供子组件事件使用）
const showErrorToast = (message: string) => {
  showToast(message, 'error')
}

// 直接显示成功消息（供子组件事件使用）
const showSuccessToast = (message: string) => {
  showToast(message, 'info')
}

// 主要功能函数 - 使用 ChannelStore
const refreshChannels = async () => {
  try {
    await channelStore.refreshChannels()
  } catch (error) {
    handleAuthError(error)
  }
}

const saveChannel = async (channel: Omit<Channel, 'index' | 'latency' | 'status'>, options?: { isQuickAdd?: boolean; triggerCapabilityTest?: boolean }) => {
  try {
    const result = await channelStore.saveChannel(channel, dialogStore.editingChannel?.index ?? null, options)
    showToast(result.message, 'success')
    if (result.quickAddMessage) {
      showToast(result.quickAddMessage, 'info')
    }
    dialogStore.closeAddChannelModal()
    dialogStore.closeEditChannelModal()
    await refreshChannels()

    if (options?.triggerCapabilityTest && result.channelId !== undefined) {
      testChannelCapability(result.channelId)
    }

    return result
  } catch (error) {
    handleAuthError(error)
    return undefined
  }
}

const editChannel = (channel: Channel) => {
  dialogStore.openEditChannelModal(channel)
}

const deleteChannel = async (channelId: number) => {
  const ok = await dialogStore.confirm({
    message: t('toast.confirmDeleteChannel'),
    confirmText: t('app.actions.delete'),
  })
  if (!ok) return

  try {
    const result = await channelStore.deleteChannel(channelId)
    showToast(result.message, 'success')
  } catch (error) {
    handleAuthError(error)
  }
}

const openAddChannelModal = () => {
  dialogStore.openAddChannelModal()
}

const _openAddKeyModal = (channelId: number) => {
  dialogStore.openAddKeyModal(channelId)
}

const addApiKey = async () => {
  if (!dialogStore.newApiKey.trim()) return

  try {
    if (channelStore.activeTab === 'chat') {
      await api.addChatApiKey(dialogStore.selectedChannelForKey, dialogStore.newApiKey.trim())
    } else if (channelStore.activeTab === 'images') {
      await api.addImagesApiKey(dialogStore.selectedChannelForKey, dialogStore.newApiKey.trim())
    } else if (channelStore.activeTab === 'gemini') {
      await api.addGeminiApiKey(dialogStore.selectedChannelForKey, dialogStore.newApiKey.trim())
    } else if (channelStore.activeTab === 'responses') {
      await api.addResponsesApiKey(dialogStore.selectedChannelForKey, dialogStore.newApiKey.trim())
    } else {
      await api.addApiKey(dialogStore.selectedChannelForKey, dialogStore.newApiKey.trim())
    }
    showToast(t('toast.apiKeyAdded'), 'success')
    dialogStore.closeAddKeyModal()
    await refreshChannels()
  } catch (error) {
    showToast(t('toast.apiKeyAddFailed', { message: error instanceof Error ? error.message : t('system.unknown') }), 'error')
  }
}

const _removeApiKey = async (channelId: number, apiKey: string) => {
  const ok = await dialogStore.confirm({
    message: t('toast.confirmDeleteApiKey'),
    confirmText: t('app.actions.delete'),
  })
  if (!ok) return

  try {
    if (channelStore.activeTab === 'chat') {
      await api.removeChatApiKey(channelId, apiKey)
    } else if (channelStore.activeTab === 'images') {
      await api.removeImagesApiKey(channelId, apiKey)
    } else if (channelStore.activeTab === 'gemini') {
      await api.removeGeminiApiKey(channelId, apiKey)
    } else if (channelStore.activeTab === 'responses') {
      await api.removeResponsesApiKey(channelId, apiKey)
    } else {
      await api.removeApiKey(channelId, apiKey)
    }
    showToast(t('toast.apiKeyDeleted'), 'success')
    await refreshChannels()
  } catch (error) {
    showToast(t('toast.apiKeyDeleteFailed', { message: error instanceof Error ? error.message : t('system.unknown') }), 'error')
  }
}

const pingChannel = async (channelId: number) => {
  try {
    await channelStore.pingChannel(channelId)
    // 不再使用 Toast，延迟结果直接显示在渠道列表中
  } catch (error) {
    showToast(t('toast.latencyFailed', { message: error instanceof Error ? error.message : t('system.unknown') }), 'error')
  }
}

// ============== 能力测试 ==============
const {
  showCapabilityTestDialog, capabilityTestChannelName, capabilityTestChannelId,
  capabilityTestChannelType, capabilityTestSourceTab, capabilityTestDialogRef,
  capabilityTestJobId, capabilityPollers, capabilityTestJob, capabilityTestRpm,
  capabilityTestPreviousJobId, capabilityRetryPendingUntil,
  isCapabilityChannelKind, capabilityPlaceholderModels, getPlaceholderModelsForProtocol,
  capabilityBaseProtocolOrder, capabilityNativeServiceTypeByProtocol,
  getCapabilityNativeServiceType, isCapabilityProtocol, buildCapabilityModels,
  buildCapabilityProtocolResult, toRetryingCapabilityModel, markCapabilityModelRetrying,
  applyCapabilityRetryPending, isIdleCapabilityTest, isActiveCapabilityTest,
  isBusyCapabilityTest, isPendingCapabilityTest, isSuccessfulCapabilityTest,
  getCapabilityAggregateState, buildCapabilityProgress, mergeCapabilityProtocolResult,
  normalizeCapabilityTests, buildCapabilityIdleJob, mergeCapabilityJob,
  getCapabilitySnapshotJobId, buildCapabilityJobFromSnapshot,
  collectActiveJobIds, isCapabilityJobTerminal, stopCapabilityPolling,
  stopAllCapabilityPolling, startCapabilityPolling, updateCapabilityJob,
  getCapabilityPreviousJobId, testChannelCapability, handleTestCapabilityProtocol,
  handleTestCapabilityProtocolWithModels, handleCancelCapabilityTest,
  handleRetryCapabilityModel, handleCopyToTab,
} = useCapabilityTestManager(channelStore, dialogStore, showToast, t, refreshChannels)

const pingAllChannels = async () => {
  try {
    await channelStore.pingAllChannels()
    // 不再使用 Toast，延迟结果直接显示在渠道列表中
  } catch (error) {
    showToast(t('toast.batchLatencyFailed', { message: error instanceof Error ? error.message : t('system.unknown') }), 'error')
  }
}

// Fuzzy 模式管理
const loadFuzzyModeStatus = async () => {
  systemStore.setFuzzyModeLoadError(false)
  try {
    const { fuzzyModeEnabled: enabled } = await api.getFuzzyMode()
    preferencesStore.setFuzzyMode(enabled)
  } catch (e) {
    console.error('Failed to load fuzzy mode status:', e)
    systemStore.setFuzzyModeLoadError(true)
    // 加载失败时不使用默认值，保持 UI 显示未知状态
    showToast(t('toast.loadFuzzyFailed'), 'warning')
  }
}

const toggleFuzzyMode = async () => {
  if (systemStore.fuzzyModeLoadError) {
    showToast(t('toast.fuzzyUnknown'), 'warning')
    return
  }
  systemStore.setFuzzyModeLoading(true)
  try {
    await api.setFuzzyMode(!preferencesStore.fuzzyModeEnabled)
    preferencesStore.toggleFuzzyMode()
    showToast(t('toast.fuzzyToggled', { state: preferencesStore.fuzzyModeEnabled ? t('common.enabled') : t('common.disabled') }), 'success')
  } catch (e) {
    showToast(t('toast.fuzzyToggleFailed', { message: e instanceof Error ? e.message : t('system.unknown') }), 'error')
  } finally {
    systemStore.setFuzzyModeLoading(false)
  }
}

// 历史图片轮次限制（已合并到熔断器对话框中）
const historicalImageForm = ref({ limit: 0 })

const loadHistoricalImageTurnLimit = async () => {
  systemStore.setHistoricalImageTurnLimitLoadError(false)
  try {
    const { historicalImageTurnLimit: limit } = await api.getHistoricalImageTurnLimit()
    preferencesStore.setHistoricalImageTurnLimit(limit)
  } catch (e) {
    console.error('Failed to load historical image turn limit:', e)
    systemStore.setHistoricalImageTurnLimitLoadError(true)
    showToast(t('toast.loadHistoricalImageTurnLimitFailed'), 'warning')
  }
}

// 新用户指引
const showGuide = ref(false)

function openGuide() {
  showGuide.value = true
}

// 指引关闭后标记已看过，避免下次自动弹出
watch(showGuide, (open) => {
  if (!open) preferencesStore.markGuideSeen()
})

// 首次认证成功后自动弹出一次指引（仅独立 WebUI，桌面端内嵌不打扰）
// 直接 watch authStore（在前文已定义），避免引用尚未声明的 isAuthenticated computed
watch(() => authStore.isAuthenticated, (authed) => {
  const isEmbedded = typeof window !== 'undefined' && window.self !== window.top
  if (authed && !preferencesStore.hasSeenGuide && !isEmbedded) {
    showGuide.value = true
  }
})

// 熔断器配置
const circuitBreakerDialogOpen = ref(false)
const cbSaving = ref(false)
const activePreset = ref('balanced')
const cbForm = reactive({
  windowSize: 10,
  failureThreshold: 0.5,
  consecutiveFailuresThreshold: 3,
  requestTimeoutMs: 120000,
  responseHeaderTimeoutMs: 60000,
  streamFirstContentTimeoutMs: 30000,
  streamInactivityTimeoutMs: 20000,
  streamToolCallIdleTimeoutMs: 120000,
})

// 工具调用 idle 预设按低速 5 TPS 粗估：60/120/300s 分别预留约 300/600/1500 token 的参数生成窗口。
const cbPresets = [
  { key: 'gentle', labelKey: 'dialog.circuitBreaker.presetGentle' as const, windowSize: 20, failureThreshold: 0.70, consecutiveFailuresThreshold: 5, requestTimeoutMs: 300000, responseHeaderTimeoutMs: 120000, streamFirstContentTimeoutMs: sharedStreamPresets.gentle.firstContentMs, streamInactivityTimeoutMs: sharedStreamPresets.gentle.inactivityMs, streamToolCallIdleTimeoutMs: sharedStreamPresets.gentle.toolCallIdleMs },
  { key: 'balanced', labelKey: 'dialog.circuitBreaker.presetBalanced' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, requestTimeoutMs: 120000, responseHeaderTimeoutMs: 60000, streamFirstContentTimeoutMs: sharedStreamPresets.balanced.firstContentMs, streamInactivityTimeoutMs: sharedStreamPresets.balanced.inactivityMs, streamToolCallIdleTimeoutMs: sharedStreamPresets.balanced.toolCallIdleMs },
  { key: 'aggressive', labelKey: 'dialog.circuitBreaker.presetAggressive' as const, windowSize: 5, failureThreshold: 0.30, consecutiveFailuresThreshold: 2, requestTimeoutMs: 60000, responseHeaderTimeoutMs: 30000, streamFirstContentTimeoutMs: sharedStreamPresets.aggressive.firstContentMs, streamInactivityTimeoutMs: sharedStreamPresets.aggressive.inactivityMs, streamToolCallIdleTimeoutMs: sharedStreamPresets.aggressive.toolCallIdleMs },
  { key: 'custom', labelKey: 'dialog.circuitBreaker.presetCustom' as const, windowSize: 10, failureThreshold: 0.50, consecutiveFailuresThreshold: 3, requestTimeoutMs: 120000, responseHeaderTimeoutMs: 60000, streamFirstContentTimeoutMs: sharedStreamPresets.balanced.firstContentMs, streamInactivityTimeoutMs: sharedStreamPresets.balanced.inactivityMs, streamToolCallIdleTimeoutMs: sharedStreamPresets.balanced.toolCallIdleMs },
]

const matchPreset = () => {
  for (const p of cbPresets) {
    if (p.key === 'custom') continue
    if (cbForm.windowSize === p.windowSize && cbForm.failureThreshold === p.failureThreshold && cbForm.consecutiveFailuresThreshold === p.consecutiveFailuresThreshold && cbForm.requestTimeoutMs === p.requestTimeoutMs && cbForm.responseHeaderTimeoutMs === p.responseHeaderTimeoutMs && cbForm.streamFirstContentTimeoutMs === p.streamFirstContentTimeoutMs && cbForm.streamInactivityTimeoutMs === p.streamInactivityTimeoutMs && cbForm.streamToolCallIdleTimeoutMs === p.streamToolCallIdleTimeoutMs) {
      activePreset.value = p.key
      return
    }
  }
  activePreset.value = 'custom'
}

const applyPreset = (preset: typeof cbPresets[number]) => {
  if (preset.key === 'custom') return
  cbForm.windowSize = preset.windowSize
  cbForm.failureThreshold = preset.failureThreshold
  cbForm.consecutiveFailuresThreshold = preset.consecutiveFailuresThreshold
  cbForm.requestTimeoutMs = preset.requestTimeoutMs
  cbForm.responseHeaderTimeoutMs = preset.responseHeaderTimeoutMs
  cbForm.streamFirstContentTimeoutMs = preset.streamFirstContentTimeoutMs
  cbForm.streamInactivityTimeoutMs = preset.streamInactivityTimeoutMs
  cbForm.streamToolCallIdleTimeoutMs = preset.streamToolCallIdleTimeoutMs
  activePreset.value = preset.key
}

const onSliderChange = (field: string, event: Event) => {
  const target = event.target
  if (!(target instanceof window.HTMLInputElement)) return
  const val = Number(target.value)
  if (field === 'failureThreshold') {
    cbForm.failureThreshold = Math.round(val * 100) / 100
  } else if (field === 'windowSize') {
    cbForm.windowSize = val
  } else if (field === 'consecutiveFailuresThreshold') {
    cbForm.consecutiveFailuresThreshold = val
  } else if (field === 'requestTimeoutMs') {
    cbForm.requestTimeoutMs = val
  } else if (field === 'responseHeaderTimeoutMs') {
    cbForm.responseHeaderTimeoutMs = val
  } else if (field === 'streamFirstContentTimeoutMs') {
    cbForm.streamFirstContentTimeoutMs = val
  } else if (field === 'streamInactivityTimeoutMs') {
    cbForm.streamInactivityTimeoutMs = val
  } else if (field === 'streamToolCallIdleTimeoutMs') {
    cbForm.streamToolCallIdleTimeoutMs = val
  }
  matchPreset()
}

const openCircuitBreakerDialog = async () => {
  historicalImageForm.value.limit = preferencesStore.historicalImageTurnLimit
  try {
    const params = await api.getCircuitBreaker()
    cbForm.windowSize = params.windowSize
    cbForm.failureThreshold = params.failureThreshold
    cbForm.consecutiveFailuresThreshold = params.consecutiveFailuresThreshold
    cbForm.requestTimeoutMs = params.requestTimeoutMs && params.requestTimeoutMs >= 1000 ? params.requestTimeoutMs : 120000
    cbForm.responseHeaderTimeoutMs = params.responseHeaderTimeoutMs && params.responseHeaderTimeoutMs >= 1000 ? params.responseHeaderTimeoutMs : 60000
    cbForm.streamFirstContentTimeoutMs = params.streamFirstContentTimeoutMs && params.streamFirstContentTimeoutMs >= 5000 ? params.streamFirstContentTimeoutMs : 60000
    cbForm.streamInactivityTimeoutMs = params.streamInactivityTimeoutMs && params.streamInactivityTimeoutMs >= 1000 ? params.streamInactivityTimeoutMs : 60000
    cbForm.streamToolCallIdleTimeoutMs = params.streamToolCallIdleTimeoutMs && params.streamToolCallIdleTimeoutMs >= 30000 ? params.streamToolCallIdleTimeoutMs : 180000
    matchPreset()
  } catch (e) {
    console.error('Failed to load circuit breaker config:', e)
  }
  circuitBreakerDialogOpen.value = true
}

const saveCircuitBreaker = async () => {
  cbSaving.value = true
  try {
    await Promise.all([
      api.setCircuitBreaker({
        windowSize: cbForm.windowSize,
        failureThreshold: cbForm.failureThreshold,
        consecutiveFailuresThreshold: cbForm.consecutiveFailuresThreshold,
        requestTimeoutMs: cbForm.requestTimeoutMs,
        responseHeaderTimeoutMs: cbForm.responseHeaderTimeoutMs,
        streamFirstContentTimeoutMs: cbForm.streamFirstContentTimeoutMs,
        streamInactivityTimeoutMs: cbForm.streamInactivityTimeoutMs,
        streamToolCallIdleTimeoutMs: cbForm.streamToolCallIdleTimeoutMs,
      }),
      api.setHistoricalImageTurnLimit(historicalImageForm.value.limit),
    ])
    preferencesStore.setHistoricalImageTurnLimit(historicalImageForm.value.limit)
    circuitBreakerDialogOpen.value = false
    showToast(t('toast.circuitBreakerSaved'), 'success')
  } catch (e) {
    showToast(t('toast.circuitBreakerFailed', { message: e instanceof Error ? e.message : t('system.unknown') }), 'error')
  } finally {
    cbSaving.value = false
  }
}

// 平台检测
const isMac = computed(() => typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform))

// 调校台弹窗键盘快捷键
const handleCircuitBreakerKeydown = (event: KeyboardEvent) => {
  if (!circuitBreakerDialogOpen.value) return

  if (event.key === 'Escape') {
    event.preventDefault()
    circuitBreakerDialogOpen.value = false
    return
  }

  // Cmd/Ctrl+Enter 确认提交
  if (event.key === 'Enter' && (event.metaKey || event.ctrlKey) && !event.shiftKey) {
    event.preventDefault()
    saveCircuitBreaker()
  }
}

// 添加API密钥弹窗键盘快捷键
const handleAddKeyKeydown = (event: KeyboardEvent) => {
  if (!dialogStore.showAddKeyModal) return

  if (event.key === 'Escape') {
    event.preventDefault()
    dialogStore.closeAddKeyModal()
    return
  }

  // Enter 确认添加
  if (event.key === 'Enter' && !event.metaKey && !event.ctrlKey && !event.shiftKey) {
    event.preventDefault()
    addApiKey()
  }
}

// 通用确认弹窗键盘快捷键
const handleConfirmKeydown = (event: KeyboardEvent) => {
  if (!dialogStore.showConfirmDialog) return

  if (event.key === 'Escape') {
    event.preventDefault()
    dialogStore.resolveConfirm(false)
    return
  }

  // Cmd/Ctrl+Enter 确认
  if (event.key === 'Enter' && (event.metaKey || event.ctrlKey) && !event.shiftKey) {
    event.preventDefault()
    dialogStore.resolveConfirm(true)
  }
}

// 主题管理
const toggleDarkMode = () => {
  const newMode = preferencesStore.darkModePreference === 'dark' ? 'light' : 'dark'
  setDarkMode(newMode)
}

const setDarkMode = (themeName: 'light' | 'dark' | 'auto') => {
  preferencesStore.setDarkMode(themeName)
  const apply = (isDark: boolean) => {
    // 使用 Vuetify 3.9+ 推荐的 theme.change() API
    theme.change(isDark ? 'dark' : 'light')
  }

  if (themeName === 'auto') {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    apply(prefersDark)
  } else {
    apply(themeName === 'dark')
  }
  // PreferencesStore 已通过 pinia-plugin-persistedstate 自动持久化，无需手动写入 localStorage
}

// 认证状态管理（使用 AuthStore）
const isAuthenticated = computed(() => authStore.isAuthenticated)
// 认证相关状态已迁移到 AuthStore

// 认证尝试限制
const MAX_AUTH_ATTEMPTS = 5

const getAuthLockoutRemainingSeconds = () => {
  const lockoutTime = authStore.authLockoutTime
  if (!lockoutTime) return 0

  const remainingSeconds = Math.ceil((lockoutTime - Date.now()) / 1000)
  if (remainingSeconds <= 0) {
    authStore.setAuthLockout(null)
    authStore.resetAuthAttempts()
    return 0
  }
  return remainingSeconds
}

// 控制认证对话框显示
const showAuthDialog = computed({
  get: () => {
    // 只有在初始化完成后，且未认证，且不在自动认证中时，才显示对话框
    return authStore.isInitialized && !isAuthenticated.value && !authStore.isAutoAuthenticating
  },
  set: () => {} // 防止外部修改，认证状态只能通过内部逻辑控制
})

// 自动验证保存的密钥
const autoAuthenticate = async () => {
  // 检查 AuthStore 中是否有保存的密钥
  if (!authStore.apiKey) {
    // 没有保存的密钥，显示登录对话框
    authStore.setAuthError(t('toast.enterAccessKeyContinue'))
    authStore.setAutoAuthenticating(false)
    authStore.setInitialized(true)
    return false
  }

  // 有保存的密钥，尝试自动认证
  try {
    // 尝试调用API验证密钥是否有效
    await api.getChannels()

    // 密钥有效，认证成功
    authStore.setAuthError('')
    return true
  } catch (error) {
    // 仅在明确 401 时视为密钥无效；其他错误（网络/5xx）不应清除密钥
    if (error instanceof ApiError && error.status === 401) {
      console.warn('自动认证失败: 认证失败(401)')
      authStore.clearAuth()
      authStore.setAuthError(t('toast.savedKeyInvalid'))
      return false
    }

    console.warn('自动认证暂时失败:', error)
    showToast(t('toast.cannotVerifyAccessKey', { message: error instanceof Error ? error.message : t('system.unknown') }), 'warning')
    // 非 401：保留密钥，继续尝试连接后端（后续刷新会更新系统状态）
    return true
  } finally {
    authStore.setAutoAuthenticating(false)
    authStore.setInitialized(true)
  }
}

// 手动设置密钥（用于重新认证）
const setAuthKey = (key: string) => {
  authStore.setApiKey(key)
  authStore.setAuthError('')
}

// 处理认证提交
const submitAuth = async (options: { countFailures?: boolean; ignoreLockout?: boolean } = {}) => {
  const countFailures = options.countFailures ?? true
  const ignoreLockout = options.ignoreLockout ?? false

  if (!authStore.authKeyInput.trim()) {
    authStore.setAuthError(t('toast.enterAccessKey'))
    return
  }

  // 检查是否被锁定；过期锁定会自动清理，避免显示负数倒计时
  // 桌面端 iframe 自动注入密钥时忽略前端本地 lockout，但仍会验证 key 是否有效
  const remainingSeconds = ignoreLockout ? 0 : getAuthLockoutRemainingSeconds()
  if (remainingSeconds > 0) {
    authStore.setAuthError(t('toast.tooManyAttemptsSeconds', { seconds: remainingSeconds }))
    return
  }

  authStore.setAuthLoading(true)
  authStore.setAuthError('')

  try {
    // 设置密钥
    setAuthKey(authStore.authKeyInput.trim())

    // 测试API调用以验证密钥
    await api.getChannels()

    // 认证成功，重置计数器
    authStore.resetAuthAttempts()
    authStore.setAuthLockout(null)

    // 如果成功，加载数据
    await refreshChannels()
    // 手动登录成功后同步系统状态，避免状态卡停留在 Connecting
    systemStore.setSystemStatus(channelStore.lastRefreshSuccess ? 'running' : 'error')

    authStore.setAuthKeyInput('')

    // 记录认证成功(前端日志)
    if (import.meta.env.DEV) {
      console.info('✅ 认证成功 - 时间:', new Date().toISOString())
    }
  } catch (error) {
    // 仅在明确 401 时计入认证失败；网络/5xx 不计入失败次数，也不清除已保存密钥
    if (error instanceof ApiError && error.status === 401) {
      if (countFailures) {
        authStore.incrementAuthAttempts()

        // 记录认证失败(前端日志)
        console.warn('🔒 认证失败 - 尝试次数:', authStore.authAttempts, '时间:', new Date().toISOString())

        // 如果尝试次数过多，锁定5分钟
        if (authStore.authAttempts >= MAX_AUTH_ATTEMPTS) {
          authStore.setAuthLockout(new Date(Date.now() + 5 * 60 * 1000))
          authStore.setAuthError(t('toast.tooManyAttempts'))
        } else {
          authStore.setAuthError(t('toast.accessKeyInvalidRemaining', { remaining: MAX_AUTH_ATTEMPTS - authStore.authAttempts }))
        }
      } else {
        // 桌面端 iframe 自动注入密钥时不消耗手动登录尝试次数，避免重复 postMessage 触发本地锁定
        authStore.setAuthError(t('toast.authInvalid'))
      }

      authStore.clearAuth()
      return
    }

    showToast(t('toast.cannotVerifyAccessKey', { message: error instanceof Error ? error.message : t('system.unknown') }), 'error')
  } finally {
    authStore.setAuthLoading(false)
  }
}

const handleAuthSubmit = async () => {
  await submitAuth()
}

// 处理注销
const handleLogout = () => {
  authStore.clearAuth()
  channelStore.clearChannels()
  authStore.setAuthError(t('toast.enterAccessKeyContinue'))
  showToast(t('toast.loggedOut'), 'info')
}

// 处理认证失败
const handleAuthError = (error: unknown) => {
  if (error instanceof ApiError && error.status === 401) {
    authStore.setAuthError(t('toast.authInvalid'))
  } else {
    showToast(t('toast.operationFailed', { message: error instanceof Error ? error.message : t('system.unknown') }), 'error')
  }
}

// 版本检查
const checkVersion = async () => {
  if (isDesktopWebUI || systemStore.isCheckingVersion) return

  systemStore.setCheckingVersion(true)
  try {
    // 直接通过 health 接口获取当前版本，再从 GitHub 检查是否有新版本
    const health = await fetchHealth()
    const currentVersion = health.version?.version || ''

    if (currentVersion) {
      versionService.setCurrentVersion(currentVersion)
      systemStore.setCurrentVersion(currentVersion)

      const result = await versionService.checkForUpdates()
      systemStore.setVersionInfo(result)
    } else {
      systemStore.setVersionInfo({
        currentVersion: systemStore.versionInfo.currentVersion,
        latestVersion: null,
        isLatest: false,
        hasUpdate: false,
        releaseUrl: null,
        lastCheckTime: 0,
        status: 'error',
      })
    }
  } catch (error) {
    console.warn('Version check failed:', error)
    systemStore.setVersionInfo({
      currentVersion: systemStore.versionInfo.currentVersion,
      latestVersion: null,
      isLatest: false,
      hasUpdate: false,
      releaseUrl: null,
      lastCheckTime: 0,
      status: 'error',
    })
  } finally {
    systemStore.setCheckingVersion(false)
  }
}

// 版本点击处理
const handleVersionClick = () => {
  systemStore.setUpdateDialogOpen(true)
}

// 监听系统主题变化（setup 阶段注册，onUnmounted 清理，避免泄漏）
// 守卫非浏览器环境（SSR / vitest 非 jsdom）：避免 ReferenceError: window is not defined
const mediaQuery = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null
const handlePref = () => {
  if (preferencesStore.darkModePreference === 'auto') setDarkMode('auto')
}
mediaQuery?.addEventListener('change', handlePref)

// 注册弹窗键盘快捷键（setup 阶段注册，onUnmounted 清理）
if (typeof window !== 'undefined') {
  window.addEventListener('keydown', handleCircuitBreakerKeydown)
  window.addEventListener('keydown', handleAddKeyKeydown)
  window.addEventListener('keydown', handleConfirmKeydown)
}

// 初始化
onMounted(async () => {
  // 初始化复古像素主题
  document.documentElement.dataset.theme = 'retro'
  initTheme()

  // 加载保存的暗色模式偏好（从 PreferencesStore 读取，已自动从 localStorage 恢复）
  setDarkMode(preferencesStore.darkModePreference)

  if (!isDesktopWebUI) {
    // 版本检查（独立于认证，静默执行）
    checkVersion()

    // 监听 UpdateDialog 手动触发的版本检查
    window.addEventListener('ccx-check-version', () => { checkVersion() })
  }

  const desktopAutoLogin = window.self !== window.top && isDesktopWebUI

  if (desktopAutoLogin) {
    authStore.clearAuth()
    authStore.setAutoAuthenticating(false)
    authStore.setInitialized(true)
    authStore.setAuthError(t('toast.enterAccessKeyContinue'))

    // 桌面端通过 postMessage 发送密钥，监听直到认证成功
    const handleDesktopAuth = async (event: MessageEvent) => {
      const data = event.data as { type?: string; accessKey?: string }
      if (data?.type !== 'ccx-desktop-auth' || !data.accessKey) return

      authStore.setAuthKeyInput(data.accessKey)
      await submitAuth({ countFailures: false, ignoreLockout: true })
      // 认证成功后移除监听器；失败时保留以便桌面端重试
      if (authStore.apiKey) {
        window.removeEventListener('message', handleDesktopAuth)
      }
    }

    window.addEventListener('message', handleDesktopAuth)
    return
  }

  // 桌面端嵌入但 ccx_desktop 参数缺失时，也注册 postMessage 监听器作为后备
  if (window.self !== window.top) {
    const handleDesktopAuthFallback = async (event: MessageEvent) => {
      const data = event.data as { type?: string; accessKey?: string }
      if (data?.type !== 'ccx-desktop-auth' || !data.accessKey) return

      window.removeEventListener('message', handleDesktopAuthFallback)
      authStore.setAuthKeyInput(data.accessKey)
      await submitAuth({ countFailures: false, ignoreLockout: true })
    }
    window.addEventListener('message', handleDesktopAuthFallback)
  }

  // 检查 AuthStore 中是否有保存的密钥
  if (authStore.apiKey) {
    // 有保存的密钥，开始自动认证
    authStore.setAutoAuthenticating(true)
    authStore.setInitialized(false)
  } else {
    // 没有保存的密钥，直接显示登录对话框
    authStore.setAutoAuthenticating(false)
    authStore.setInitialized(true)
  }

  // 尝试自动认证
  const authenticated = await autoAuthenticate()

  if (authenticated) {
    // 加载渠道数据
    await refreshChannels()
    // 加载 Fuzzy 模式状态
    await loadFuzzyModeStatus()
    // 加载历史图片轮次限制状态
    await loadHistoricalImageTurnLimit()
    // 启动自动刷新
    startAutoRefresh()
    // 初始化完成后根据最新刷新结果设置系统状态
    systemStore.setSystemStatus(channelStore.lastRefreshSuccess ? 'running' : 'error')
  }
})

// 启动自动刷新定时器
const startAutoRefresh = () => {
  channelStore.startAutoRefresh()
}

// 停止自动刷新定时器
const stopAutoRefresh = () => {
  channelStore.stopAutoRefresh()
}

// 监听 Tab 切换，刷新对应数据
watch(() => channelStore.activeTab, async () => {
  if (isAuthenticated.value) {
    try {
      await channelStore.refreshChannels()
    } catch (error) {
      console.error('切换 Tab 刷新失败:', error)
    }
  }
})

// 监听认证状态变化
watch(isAuthenticated, newValue => {
  if (newValue) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

// 监听自动刷新状态，更新 systemStatus
watch(() => channelStore.lastRefreshSuccess, (success) => {
  if (isAuthenticated.value) {
    systemStore.setSystemStatus(success ? 'running' : 'error')
  }
})

// 在组件卸载时清除定时器和事件监听器
onUnmounted(() => {
  channelStore.stopAutoRefresh()
  stopAllCapabilityPolling()
  mediaQuery?.removeEventListener('change', handlePref)
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleCircuitBreakerKeydown)
    window.removeEventListener('keydown', handleAddKeyKeydown)
    window.removeEventListener('keydown', handleConfirmKeydown)
  }
})
</script>

<style scoped src="./styles/app-retro.css"></style>

<!-- 全局样式 - 复古像素主题 -->
<style src="./styles/app-retro-global.css"></style>
