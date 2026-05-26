<template>
  <v-dialog :model-value="show" max-width="800" persistent @update:model-value="$emit('update:show', $event)">
    <v-card rounded="lg">
      <v-card-title class="d-flex align-center ga-3 pa-6" :class="headerClasses">
        <v-avatar :color="avatarColor" variant="flat" size="40">
          <v-icon :style="headerIconStyle" size="20">{{ isEditing ? 'mdi-pencil' : 'mdi-plus' }}</v-icon>
        </v-avatar>
        <div class="flex-grow-1 modal-header-text">
          <div class="modal-title">
            {{ isEditing ? t('addChannel.editTitle') : t('addChannel.createTitle') }}
          </div>
          <div class="modal-subtitle" :class="subtitleClasses">
            {{ isEditing ? t('addChannel.editSubtitle') : isQuickMode ? t('addChannel.quickSubtitle') : t('addChannel.fullSubtitle') }}
          </div>
        </div>
        <div v-if="isEditing && props.channelType !== 'images'" class="header-capability-actions">
          <v-tooltip location="bottom" :text="form.noVision ? t('channelCard.noVision') : t('channelCard.hasVision')" :open-delay="150" content-class="key-tooltip">
            <template #activator="{ props: tip }">
              <v-btn
                v-bind="tip"
                :color="form.noVision ? 'warning' : undefined"
                :variant="form.noVision ? 'tonal' : 'text'"
                size="small"
                icon
                rounded="lg"
                class="mr-2"
                @click="form.noVision = !form.noVision"
              >
                <v-icon size="18">{{ form.noVision ? 'mdi-eye-off' : 'mdi-eye' }}</v-icon>
              </v-btn>
            </template>
          </v-tooltip>
          <v-btn
            color="success"
            variant="flat"
            size="small"
            prepend-icon="mdi-test-tube"
            class="capability-test-btn"
            @click="handleTestCapability"
          >
            {{ t('addChannel.testCapability') }}
          </v-btn>
        </div>
        <!-- 模式切换按钮（仅在添加模式显示） -->
        <v-btn v-if="!isEditing" variant="outlined" size="small" class="mode-toggle-btn" @click="toggleMode">
          <v-icon start size="16">{{ isQuickMode ? 'mdi-form-textbox' : 'mdi-lightning-bolt' }}</v-icon>
          {{ isQuickMode ? t('addChannel.detailedMode') : t('addChannel.quickMode') }}
        </v-btn>
      </v-card-title>

      <v-card-text class="pa-6">
        <!-- 快速添加模式 -->
        <div v-if="!isEditing && isQuickMode">
          <v-textarea
            v-model="quickInput"
            :label="t('addChannel.quickInputLabel')"
            :placeholder="t('addChannel.quickInputPlaceholder')"
            variant="outlined"
            rows="10"
            no-resize
            autofocus
            class="quick-input-textarea"
            @input="parseQuickInput"
          />

          <!-- 检测状态提示 -->
          <v-card variant="outlined" class="mt-4 detection-status-card" rounded="lg">
            <v-card-text class="pa-4">
              <div class="d-flex flex-column ga-3">
                <!-- Base URL 检测 -->
                <div class="d-flex align-start ga-3">
                  <v-icon :color="detectedBaseUrls.length > 0 ? 'success' : 'error'" size="20" class="mt-1">
                    {{ detectedBaseUrls.length > 0 ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                  </v-icon>
                  <div class="flex-grow-1">
                    <div class="text-body-2 font-weight-medium">{{ t('addChannel.baseUrl') }}</div>
                    <div v-if="detectedBaseUrls.length === 0" class="text-caption text-error">
                      {{ t('addChannel.enterValidUrl') }}
                    </div>
                    <div v-else class="d-flex flex-column ga-2 mt-1">
                      <div v-for="url in detectedBaseUrls" :key="url" class="base-url-item">
                        <div class="text-caption text-success">{{ url }}</div>
                        <div class="text-caption text-medium-emphasis">{{ t('addChannel.expectedRequest') }} {{ getExpectedRequestUrl(url) }}</div>
                      </div>
                    </div>
                  </div>
                  <v-chip v-if="detectedBaseUrls.length > 0" size="x-small" color="success" variant="tonal">
                    {{ t('addChannel.count', { count: detectedBaseUrls.length }) }}
                  </v-chip>
                </div>

                <!-- API Keys 检测 -->
                <div class="d-flex align-center ga-3">
                  <v-icon :color="detectedApiKeys.length > 0 ? 'success' : 'error'" size="20">
                    {{ detectedApiKeys.length > 0 ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                  </v-icon>
                  <div class="flex-grow-1">
                    <div class="text-body-2 font-weight-medium">{{ t('addChannel.apiKeys') }}</div>
                    <div class="text-caption" :class="detectedApiKeys.length > 0 ? 'text-success' : 'text-error'">
                      {{
                        detectedApiKeys.length > 0
                          ? t('addChannel.detectedKeys', { count: detectedApiKeys.length })
                          : t('addChannel.enterApiKey')
                      }}
                    </div>
                  </div>
                  <v-chip v-if="detectedApiKeys.length > 0" size="x-small" color="success" variant="tonal">
                    {{ t('addChannel.count', { count: detectedApiKeys.length }) }}
                  </v-chip>
                </div>

                <!-- 渠道名称预览 -->
                <div class="d-flex align-center ga-3">
                  <v-icon color="primary" size="20">mdi-tag</v-icon>
                  <div class="flex-grow-1">
                    <div class="text-body-2 font-weight-medium">{{ t('addChannel.channelName') }}</div>
                    <div class="text-caption text-primary font-weight-medium">
                      {{ generatedChannelName }}
                    </div>
                  </div>
                  <v-chip size="x-small" color="primary" variant="tonal"> {{ t('common.autoGenerated') }} </v-chip>
                </div>

                <!-- 渠道类型提示 -->
                <div class="d-flex align-center ga-3">
                  <v-icon color="info" size="20">mdi-information</v-icon>
                  <div class="flex-grow-1">
                    <div class="text-body-2 font-weight-medium">{{ t('addChannel.channelType') }}</div>
                    <div class="text-caption text-medium-emphasis">
                      {{ props.channelType === 'chat' ? 'OpenAI Chat' : props.channelType === 'gemini' ? 'Gemini' : props.channelType === 'responses' ? 'Responses (Codex)' : props.channelType === 'images' ? 'Images' : 'Claude (Messages)' }} -
                      {{ getDefaultServiceType() }}
                    </div>
                  </div>
                </div>
              </div>
            </v-card-text>
          </v-card>
        </div>

        <!-- 详细表单模式（原有表单） -->
        <v-form v-else ref="formRef" @submit.prevent="handleSubmit">
          <v-row>
            <!-- 基本信息 -->
            <v-col cols="12" md="6">
              <v-text-field
                v-model="form.name"
                :label="t('addChannel.nameLabel')"
                :placeholder="t('addChannel.namePlaceholder')"
                prepend-inner-icon="mdi-tag"
                variant="outlined"
                density="comfortable"
                :rules="[rules.required]"
                required
                :error-messages="errors.name"
              />
            </v-col>

            <v-col cols="12" md="6">
              <v-select
                v-model="form.serviceType"
                :label="t('addChannel.serviceTypeLabel')"
                :items="serviceTypeOptions"
                prepend-inner-icon="mdi-cog"
                variant="outlined"
                density="comfortable"
                :rules="[rules.required]"
                required
                :error-messages="errors.serviceType"
                eager
                @update:menu="onMenuUpdate"
              />
            </v-col>

            <!-- 基础URL -->
            <v-col cols="12">
              <v-textarea
                v-model="baseUrlsText"
                :label="t('addChannel.baseUrlLabel')"
                :placeholder="t('addChannel.baseUrlPlaceholder')"
                prepend-inner-icon="mdi-web"
                variant="outlined"
                density="comfortable"
                rows="3"
                no-resize
                :rules="[rules.required, rules.baseUrls]"
                required
                :error-messages="errors.baseUrl"
                hide-details="auto"
              />
              <!-- 固定高度的提示区域，防止布局跳动；有错误时不显示 -->
              <div v-show="formExpectedRequestUrls.length > 0 && !baseUrlHasError" class="base-url-hint">
                <div v-for="(item, index) in formExpectedRequestUrls" :key="index" class="expected-request-item">
                  <span class="text-caption text-medium-emphasis"> {{ t('addChannel.expectedRequest') }} {{ item.expectedUrl }} </span>
                </div>
              </div>
            </v-col>

            <!-- 官网/控制台（可选） -->
            <v-col cols="12">
              <v-text-field
                v-model="form.website"
                :label="t('addChannel.websiteLabel')"
                :placeholder="t('addChannel.websitePlaceholder')"
                prepend-inner-icon="mdi-open-in-new"
                variant="outlined"
                density="comfortable"
                type="url"
                :rules="[rules.urlOptional]"
                :error-messages="errors.website"
              />
            </v-col>

            <!-- 模型重定向配置 -->
            <v-col v-if="form.serviceType" cols="12">
              <v-card variant="outlined" rounded="lg">
                <v-card-title class="d-flex align-center justify-space-between pa-4 pb-2">
                  <div class="d-flex align-center ga-2">
                    <v-icon color="primary">mdi-swap-horizontal</v-icon>
                    <span class="section-title">{{ t('addChannel.modelRedirect') }}</span>
                  </div>
                  <v-chip size="small" color="secondary" variant="tonal"> {{ t('addChannel.autoConvertModelNames') }} </v-chip>
                </v-card-title>

                <v-card-text class="pt-2">
                  <div class="text-body-2 text-medium-emphasis mb-4">
                    {{ modelMappingHint }}
                    <br/>
                    <span class="text-caption text-primary">💡 {{ t('addChannel.modelHintTip') }}</span>
                  </div>

                  <div v-if="showModelMappingPresets" class="d-flex align-center flex-wrap ga-2 mb-4">
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.oneClickSetup') }}</div>
                    <v-btn
                      size="small"
                      variant="tonal"
                      color="primary"
                      prepend-icon="mdi-lightning-bolt"
                      @click="applyModelMappingPreset('gpt-5.5')"
                    >
                      gpt-5.5
                    </v-btn>
                    <v-btn
                      size="small"
                      variant="tonal"
                      color="secondary"
                      prepend-icon="mdi-lightning-bolt"
                      @click="applyModelMappingPreset('gpt-5.4')"
                    >
                      gpt-5.4
                    </v-btn>
                    <v-btn
                      size="small"
                      variant="tonal"
                      color="secondary"
                      prepend-icon="mdi-lightning-bolt"
                      @click="applyModelMappingPreset('gpt-5.3-codex')"
                    >
                     gpt-5.3-codex
                    </v-btn>
                    <v-btn
                      size="small"
                      variant="tonal"
                      color="secondary"
                      prepend-icon="mdi-lightning-bolt"
                      @click="applyModelMappingPreset('gpt-5.2-codex')"
                    >
                      gpt-5.2 / gpt-5.2-codex
                    </v-btn>
                  </div>

                  <!-- 现有映射列表 -->
                  <div v-if="Object.keys(form.modelMapping).length" class="mb-4">
                    <v-list density="compact" class="bg-transparent">
                      <template v-for="[source, target] in Object.entries(form.modelMapping)" :key="source">
                      <v-list-item
                        class="mb-2"
                        rounded="lg"
                        variant="tonal"
                        color="surface-variant"
                      >
                        <template #prepend>
                          <v-icon size="small" color="primary">mdi-arrow-right</v-icon>
                        </template>

                      <v-list-item-title>
                          <div class="d-flex align-center ga-2 flex-wrap">
                            <code class="text-caption">{{ source }}</code>
                            <v-icon size="small" color="primary">mdi-arrow-right</v-icon>
                            <code class="text-caption">{{ target }}</code>
                            <v-chip
                              v-if="supportsOpenAIAdvancedOptions && form.reasoningMapping[source]"
                              size="x-small"
                              color="secondary"
                              variant="tonal"
                            >
                              reasoning: {{ form.reasoningMapping[source] }}
                            </v-chip>
                          </div>
                        </v-list-item-title>

                        <template #append>
                          <div class="d-flex align-center ga-1">
                            <v-tooltip :text="isModelNoVision(target) ? t('addChannel.visionDisabled') : t('addChannel.visionEnabled')" location="top" :open-delay="150" content-class="key-tooltip">
                              <template #activator="{ props: tip }">
                                <v-btn
                                  v-bind="tip"
                                  size="small"
                                  :color="isModelNoVision(target) ? 'warning' : undefined"
                                  icon
                                  variant="text"
                                  @click="toggleModelVision(target)"
                                >
                                  <v-icon size="small">{{ isModelNoVision(target) ? 'mdi-eye-off' : 'mdi-eye' }}</v-icon>
                                </v-btn>
                              </template>
                            </v-tooltip>
                            <v-btn size="small" color="error" icon variant="text" @click="removeModelMapping(source)">
                              <v-icon size="small" color="error">mdi-close</v-icon>
                            </v-btn>
                          </div>
                        </template>
                      </v-list-item>
                      </template>
                    </v-list>
                  </div>

                  <!-- 添加新映射 -->
                  <div class="d-flex align-center flex-wrap ga-2">
                    <v-combobox
                      v-model="newMapping.source"
                      :label="t('addChannel.sourceModelLabel')"
                      :items="sourceModelOptions"
                      variant="outlined"
                      density="comfortable"
                      hide-details
                      class="flex-1-1"
                      style="min-width: 160px"
                      :placeholder="t('addChannel.sourceModelPlaceholder')"
                      clearable
                      :error="!!sourceMappingError"
                      eager
                      @update:model-value="handleSourceModelChange"
                      @update:menu="onMenuUpdate"
                      @keyup.enter="addModelMapping"
                    />
                    <v-icon color="primary">mdi-arrow-right</v-icon>
                    <v-combobox
                      v-model="newMapping.target"
                      :label="t('addChannel.targetModelLabel')"
                      :placeholder="targetModelPlaceholder"
                      :items="targetModelOptions"
                      :loading="fetchingModels"
                      variant="outlined"
                      density="comfortable"
                      hide-details
                      class="flex-1-1"
                      style="min-width: 160px"
                      clearable
                      eager
                      @focus="handleTargetModelClick"
                      @update:menu="onMenuUpdate"
                      @keyup.enter="addModelMapping"
                    />
                    <v-select
                      v-if="supportsOpenAIAdvancedOptions"
                      v-model="newMapping.reasoningEffort"
                      :label="t('addChannel.reasoningEffortLabel')"
                      :items="reasoningEffortOptions"
                      variant="outlined"
                      density="comfortable"
                      hide-details
                      clearable
                      class="flex-1-1"
                      eager
                      @update:menu="onMenuUpdate"
                    />
                    <v-btn
                      color="secondary"
                      variant="elevated"
                      :disabled="!isMappingInputValid"
                      @click="addModelMapping"
                    >
                      {{ t('app.actions.add') }}
                    </v-btn>
                  </div>
                  <!-- 错误提示 -->
                  <div v-if="sourceMappingError" class="text-error text-caption mt-2">
                    {{ sourceMappingError }}
                  </div>
                  <div v-if="fetchModelsError" class="text-error text-caption mt-2">
                    {{ fetchModelsError }}
                  </div>
                  <v-row v-if="supportsOpenAIAdvancedOptions" class="mt-4">
                    <v-col cols="12" md="6">
                      <div class="d-flex align-center justify-space-between h-100 advanced-switch-row">
                        <div>
                          <div class="text-body-2 font-weight-medium">{{ t('addChannel.fastMode') }}</div>
                          <div class="text-caption text-medium-emphasis">{{ t('addChannel.fastModeHint') }}</div>
                        </div>
                        <v-switch
                          v-model="form.fastMode"
                          color="primary"
                          hide-details
                          inset
                        />
                      </div>
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="form.textVerbosity"
                        :label="t('addChannel.textVerbosityLabel')"
                        :items="textVerbosityOptions"
                        variant="outlined"
                        density="comfortable"
                        hide-details
                        clearable
                        eager
                        @update:menu="onMenuUpdate"
                      />
                    </v-col>
                  </v-row>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- Vision 回退模型（仅当有模型级 noVision 标记时显示） -->
            <v-col v-if="form.noVisionModels.length > 0" cols="12">
              <v-combobox
                v-model="form.visionFallbackModel"
                :label="t('addChannel.visionFallbackLabel')"
                :placeholder="t('addChannel.visionFallbackPlaceholder')"
                :hint="t('addChannel.visionFallbackHint')"
                :items="targetModelOptions"
                prepend-inner-icon="mdi-eye"
                persistent-hint
                clearable
                variant="outlined"
                density="comfortable"
                eager
                @focus="ensureTargetModelsLoaded"
                @update:menu="onMenuUpdate"
              />
            </v-col>

            <!-- 支持的模型白名单 -->
            <v-col cols="12">
              <v-combobox
                v-model="form.supportedModels"
                :label="t('addChannel.supportedModelsLabel')"
                :placeholder="t('addChannel.supportedModelsPlaceholder')"
                prepend-inner-icon="mdi-brain"
                :hint="t('addChannel.supportedModelsHint')"
                :error-messages="supportedModelsError ? [supportedModelsError] : []"
                persistent-hint
                clearable
                multiple
                chips
                closable-chips
                variant="outlined"
                density="comfortable"
                eager
                @update:model-value="handleSupportedModelsChange"
                @update:menu="onMenuUpdate"
              />
              <div class="d-flex align-center flex-wrap ga-2 mt-2">
                <div class="text-caption text-primary">{{ t('addChannel.commonFilters') }}</div>
                <v-chip
                  v-for="filter in commonSupportedModelFilters"
                  :key="filter"
                  size="small"
                  :color="isSupportedModelSelected(filter) ? 'primary' : 'default'"
                  :variant="isSupportedModelSelected(filter) ? 'flat' : 'tonal'"
                  @click="appendSupportedModelFilter(filter)"
                >
                  {{ filter }}
                </v-chip>
              </div>
            </v-col>

            <!-- API密钥管理 -->
            <v-col cols="12">
              <v-card variant="outlined" rounded="lg" :color="hasConfigurableKeys ? undefined : 'error'">
                <v-card-title class="d-flex align-center justify-space-between pa-4 pb-2">
                  <div class="d-flex align-center ga-2">
                    <v-icon :color="hasConfigurableKeys ? 'primary' : 'error'">mdi-key</v-icon>
                    <span class="section-title">{{ t('channelCard.apiKeyManagement') }} *</span>
                    <v-chip v-if="!hasConfigurableKeys" size="x-small" color="error" variant="tonal">
                      {{ t('addChannel.apiKeyRequired') }}
                    </v-chip>
                  </div>
                  <v-chip size="small" color="info" variant="tonal"> {{ t('addChannel.apiKeyLoadBalance') }} </v-chip>
                </v-card-title>

                <v-card-text class="pt-2">
                  <!-- 现有密钥列表 -->
                  <div v-if="form.apiKeys.length" class="mb-4">
                    <v-list density="compact" class="bg-transparent">
                      <v-list-item
                        v-for="(key, index) in form.apiKeys"
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
                                {{ t('addChannel.duplicateKey') }}
                              </v-chip>
                            </div>
                          </div>
                        </v-list-item-title>

                        <template #append>
                          <div class="d-flex align-center ga-1">
                            <!-- 置顶/置底：仅首尾密钥显示 -->
                            <v-tooltip
                              v-if="index === form.apiKeys.length - 1 && form.apiKeys.length > 1"
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
                                  @click="moveApiKeyToTop(index)"
                                >
                                  <v-icon size="small">mdi-arrow-up-bold</v-icon>
                                </v-btn>
                              </template>
                            </v-tooltip>
                            <v-tooltip
                              v-if="index === 0 && form.apiKeys.length > 1"
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
                                  @click="moveApiKeyToBottom(index)"
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
                                  @click="copyApiKey(key, index)"
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
                                  @click="removeApiKey(index)"
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

                  <!-- 添加新密钥 -->
                  <div class="d-flex align-start ga-3">
                    <v-text-field
                      v-model="newApiKey"
                      :label="t('addChannel.addNewApiKey')"
                      :placeholder="t('addChannel.addNewApiKeyPlaceholder')"
                      prepend-inner-icon="mdi-plus"
                      variant="outlined"
                      density="comfortable"
                      type="password"
                      :error="!!apiKeyError"
                      :error-messages="apiKeyError"
                      class="flex-grow-1"
                      @keyup.enter="addApiKey"
                      @input="handleApiKeyInput"
                    />
                    <v-btn
                      color="primary"
                      variant="elevated"
                      size="large"
                      height="40"
                      :disabled="!newApiKey.trim()"
                      class="mt-1"
                      @click="addApiKey"
                    >
                      {{ t('app.actions.add') }}
                    </v-btn>
                  </div>

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
                            {{ t(getRestoreDisabledKeyLabel(dk.reason)) }}
                          </v-chip>
                          <span class="text-caption">{{ new Date(dk.disabledAt).toLocaleDateString() }}</span>
                        </v-list-item-subtitle>
                        <template #append>
                          <v-btn size="x-small" color="success" variant="tonal" rounded="lg" :loading="restoringKey === dk.key" @click="restoreDisabledKey(dk.key)">
                            <v-icon start size="small">mdi-restore</v-icon>
                            {{ t('channelCard.restoreKey') }}
                          </v-btn>
                        </template>
                      </v-list-item>
                    </v-list>
                  </div>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- 描述 -->
            <v-col cols="12">
              <v-textarea
                v-model="form.description"
                :label="t('addChannel.descriptionLabel')"
                :hint="t('addChannel.descriptionHint')"
                persistent-hint
                prepend-inner-icon="mdi-text"
                variant="outlined"
                density="comfortable"
                rows="3"
                no-resize
              />
            </v-col>

            <!-- 跳过 TLS 证书验证 -->
            <v-col cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="warning">mdi-shield-alert</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.skipTlsLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.skipTlsHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.insecureSkipVerify" inset color="warning" hide-details />
              </div>
            </v-col>

            <!-- 低质量渠道标记 -->
            <v-col cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="info">mdi-speedometer-slow</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.lowQualityLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.lowQualityHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.lowQuality" inset color="info" hide-details />
              </div>
            </v-col>

            <v-col cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="warning">mdi-cash-remove</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.autoBlacklistBalanceLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.autoBlacklistBalanceHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.autoBlacklistBalance" inset color="warning" hide-details />
              </div>
            </v-col>

            <v-col v-if="props.channelType === 'responses'" cols="12">
              <div class="d-flex align-center justify-space-between ga-5">
                <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
                  <v-icon color="primary">mdi-cog</v-icon>
                  <div style="min-width: 0;">
                    <div class="section-title section-title--soft">{{ t('addChannel.codexNativeToolPassthroughLabel') }}</div>
                    <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.codexNativeToolPassthroughHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.codexNativeToolPassthrough" inset color="primary" hide-details style="flex-shrink: 0;" />
              </div>
            </v-col>

            <v-col v-if="props.channelType === 'responses'" cols="12">
              <div class="d-flex align-center justify-space-between ga-5">
                <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
                  <v-icon color="primary">mdi-cog</v-icon>
                  <div style="min-width: 0;">
                    <div class="section-title section-title--soft">{{ t('addChannel.codexToolCompatLabel') }}</div>
                    <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.codexToolCompatHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.codexToolCompat" inset color="primary" hide-details style="flex-shrink: 0;" />
              </div>
            </v-col>

            <v-col v-if="props.channelType === 'messages' || props.channelType === 'responses'" cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="primary">mdi-identifier</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.normalizeMetadataUserIdLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeMetadataUserIdHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.normalizeMetadataUserId" inset color="primary" hide-details />
              </div>
            </v-col>

            <v-col v-if="supportsChatRoleNormalization" cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="primary">mdi-account-switch</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.normalizeNonstandardChatRolesLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.normalizeNonstandardChatRolesHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.normalizeNonstandardChatRoles" inset color="primary" hide-details />
              </div>
            </v-col>

            <v-col v-if="supportsOpenAIAdvancedOptions" cols="12">
              <div class="d-flex align-center justify-space-between ga-4">
                <div class="d-flex align-center ga-2">
                  <v-icon color="primary">mdi-tune</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.reasoningParamStyleLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.reasoningParamStyleHint') }}</div>
                  </div>
                </div>
                <v-select
                  v-model="form.reasoningParamStyle"
                  :items="reasoningParamStyleOptions"
                  variant="outlined"
                  density="comfortable"
                  hide-details
                  class="channel-config-select"
                  eager
                  @update:menu="onMenuUpdate"
                />
              </div>
            </v-col>

            <!-- 注入 Dummy Thought Signature（上游为 Gemini 时显示） -->
            <!-- 注：chat/responses 渠道当前默认始终注入，开关无效，仅 messages/gemini 渠道下按开关生效 -->
            <v-col v-if="(props.channelType === 'gemini' || props.channelType === 'messages') && form.serviceType === 'gemini'" cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="secondary">mdi-signature</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.injectDummyThoughtSignatureLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.injectDummyThoughtSignatureHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.injectDummyThoughtSignature" inset color="secondary" hide-details />
              </div>
            </v-col>

            <!-- 移除 Thought Signature（上游为 Gemini 时显示，覆盖所有可能落到 Gemini 的入口） -->
            <v-col v-if="form.serviceType === 'gemini' && (props.channelType === 'gemini' || props.channelType === 'messages' || props.channelType === 'chat' || props.channelType === 'responses')" cols="12">
              <div class="d-flex align-center justify-space-between">
                <div class="d-flex align-center ga-2">
                  <v-icon color="error">mdi-close-circle</v-icon>
                  <div>
                    <div class="section-title section-title--soft">{{ t('addChannel.stripThoughtSignatureLabel') }}</div>
                    <div class="text-caption text-medium-emphasis">{{ t('addChannel.stripThoughtSignatureHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.stripThoughtSignature" inset color="error" hide-details />
              </div>
            </v-col>

            <!-- 回传 Reasoning Content（仅 Messages/Chat 渠道 + claude 服务类型显示） -->
            <v-col v-if="(props.channelType === 'messages' || props.channelType === 'chat') && form.serviceType === 'claude'" cols="12">
              <div class="d-flex align-center justify-space-between ga-5">
                <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
                  <v-icon color="secondary">mdi-brain</v-icon>
                  <div style="min-width: 0;">
                    <div class="section-title section-title--soft">{{ t('addChannel.passbackReasoningContentLabel') }}</div>
                    <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.passbackReasoningContentHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.passbackReasoningContent" inset color="secondary" hide-details style="flex-shrink: 0;" />
              </div>
            </v-col>

            <!-- 移除空 Text Block（仅 Messages 渠道 + claude 服务类型显示） -->
            <v-col v-if="props.channelType === 'messages' && form.serviceType === 'claude'" cols="12">
              <div class="d-flex align-center justify-space-between ga-5">
                <div class="d-flex align-center ga-2" style="min-width: 0; flex: 1 1 auto;">
                  <v-icon color="warning">mdi-filter-remove</v-icon>
                  <div style="min-width: 0;">
                    <div class="section-title section-title--soft">{{ t('addChannel.stripEmptyTextBlocksLabel') }}</div>
                    <div class="text-caption text-medium-emphasis" style="word-break: break-word;">{{ t('addChannel.stripEmptyTextBlocksHint') }}</div>
                  </div>
                </div>
                <v-switch v-model="form.stripEmptyTextBlocks" inset color="warning" hide-details style="flex-shrink: 0;" />
              </div>
            </v-col>

            <!-- 自定义请求头 -->
            <v-col cols="12">
              <v-card variant="outlined">
                <v-card-title class="section-card-title d-flex align-center ga-2">
                  <v-icon size="small">mdi-web</v-icon>
                  {{ t('addChannel.customHeadersLabel') }}
                </v-card-title>
                <v-card-text>
                  <div class="text-caption text-medium-emphasis mb-3">
                    {{ t('addChannel.customHeadersHint') }}
                  </div>

                  <!-- 已添加的请求头列表 -->
                  <v-list v-if="Object.keys(form.customHeaders).length > 0" density="compact" class="mb-3">
                    <v-list-item
                      v-for="(value, key) in form.customHeaders"
                      :key="key"
                      class="px-2"
                    >
                      <template #prepend>
                        <v-icon size="small" color="primary">mdi-tag</v-icon>
                      </template>
                      <v-list-item-title class="text-body-2">
                        <code>{{ key }}</code>: <span class="text-medium-emphasis">{{ value }}</span>
                      </v-list-item-title>
                      <template #append>
                        <v-btn
                          icon="mdi-delete"
                          size="x-small"
                          variant="text"
                          color="error"
                          @click="removeCustomHeader(key as string)"
                        />
                      </template>
                    </v-list-item>
                  </v-list>

                  <!-- 添加新请求头 -->
                  <div class="d-flex ga-2 align-center">
                    <v-text-field
                      v-model="newHeaderKey"
                      :label="t('addChannel.headerNameLabel')"
                      placeholder="X-Custom-Header"
                      variant="outlined"
                      density="compact"
                      hide-details
                      style="flex: 1"
                    />
                    <v-text-field
                      v-model="newHeaderValue"
                      :label="t('addChannel.headerValueLabel')"
                      placeholder="value"
                      variant="outlined"
                      density="compact"
                      hide-details
                      style="flex: 2"
                    />
                    <v-btn
                      icon="mdi-plus"
                      size="small"
                      color="primary"
                      variant="tonal"
                      :disabled="!newHeaderKey.trim() || !newHeaderValue.trim()"
                      @click="addCustomHeader"
                    />
                  </div>
                </v-card-text>
              </v-card>
            </v-col>

            <!-- 代理 URL -->
            <v-col cols="12">
              <v-text-field
                v-model="form.proxyUrl"
                :label="t('addChannel.proxyUrlLabel')"
                :placeholder="t('addChannel.proxyUrlPlaceholder')"
                prepend-inner-icon="mdi-shield-lock-outline"
                :hint="t('addChannel.proxyUrlHint')"
                persistent-hint
                clearable
                variant="outlined"
                density="comfortable"
              />
            </v-col>

            <!-- 路由前缀 -->
            <v-col cols="12">
              <v-text-field
                v-model="form.routePrefix"
                :label="t('addChannel.routePrefixLabel')"
                :placeholder="t('addChannel.routePrefixPlaceholder')"
                prepend-inner-icon="mdi-routes"
                :hint="t('addChannel.routePrefixHint')"
                persistent-hint
                clearable
                variant="outlined"
                density="comfortable"
              />
            </v-col>

          </v-row>
        </v-form>
      </v-card-text>

      <v-card-actions class="pa-6 pt-0">
        <v-spacer />
        <v-btn variant="text" @click="handleCancel"> {{ t('app.actions.cancel') }} </v-btn>
        <v-btn
          v-if="!isEditing && isQuickMode"
          color="primary"
          variant="elevated"
          :disabled="!isQuickFormValid"
          prepend-icon="mdi-check"
          @click="handleQuickSubmit"
        >
          {{ t('addChannel.createChannel') }}
        </v-btn>
        <v-btn
          v-else
          color="primary"
          variant="elevated"
          :disabled="!isFormValid"
          prepend-icon="mdi-check"
          @click="handleSubmit"
        >
          {{ isEditing ? t('addChannel.updateChannel') : t('addChannel.createChannel') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useTheme } from 'vuetify'
import type { Channel } from '../services/api'
import { ApiService, ApiError } from '../services/api'
import { useChannelStore } from '../stores/channel'
import { useDialogStore } from '../stores/dialog'
import {
  isValidApiKey as _isValidApiKey,
  isValidUrl as _isValidQuickInputUrl,
  parseQuickInput as parseQuickInputUtil
} from '../utils/quickInputParser'
import { buildExpectedRequestUrls } from '../utils/expectedRequestUrls'
import { supportsAdvancedChannelOptions } from '../utils/channelAdvancedOptions'
import { buildExpectedRequestUrl } from '../utils/baseUrlSemantics'
import { buildChannelPayload } from '../utils/channelPayload'
import {
  resolveChannelWatcherAction,
  syncBaseUrlsFormState,
  filterValidSupportedModelPatterns
} from '../utils/add-channel-modal-state'
import { useI18n } from '../i18n'

interface Props {
  show: boolean
  channel?: Channel | null
  channelType?: 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
}

const props = withDefaults(defineProps<Props>(), {
  channelType: 'messages'
})

const emit = defineEmits<{
  'update:show': [value: boolean]
  save: [channel: Omit<Channel, 'index' | 'latency' | 'status'>, options?: { isQuickAdd?: boolean; triggerCapabilityTest?: boolean }]
  testCapability: [channelId: number]
  error: [message: string]
}>()
const { t } = useI18n()
const apiService = new ApiService()
const channelStore = useChannelStore()
const dialogStore = useDialogStore()

// 主题
const theme = useTheme()

// 表单引用
const formRef = ref()

// 模式切换: 快速添加 vs 详细表单
const isQuickMode = ref(true)

// 快速添加模式的数据
const quickInput = ref('')
const detectedBaseUrl = ref('')
const detectedBaseUrls = ref<string[]>([])
const detectedRawBaseUrls = ref<string[]>([])
const detectedApiKeys = ref<string[]>([])
const detectedServiceType = ref<'openai' | 'gemini' | 'claude' | 'responses' | null>(null)

const getImagesServiceType = (_serviceType: 'openai' | 'gemini' | 'claude' | 'responses' | null | ''): 'openai' => {
  return 'openai'
}

// 详细表单预期请求 URL 预览（防止输入时抖动）
const formBaseUrlPreview = ref('')
let formBaseUrlPreviewTimer: number | null = null

// 切换模式时，将快速模式检测到的值同步到详细表单，但不清空快速模式输入
const toggleMode = () => {
  if (isQuickMode.value) {
    const effectiveServiceType = props.channelType === 'images'
      ? getImagesServiceType(detectedServiceType.value)
      : (detectedServiceType.value || getDefaultServiceTypeValue())
    const sourceUrls = detectedRawBaseUrls.value.length > 0
      ? detectedRawBaseUrls.value.join('\n')
      : (detectedBaseUrl.value || '')

    const { baseUrl, baseUrls } = syncBaseUrlsFormState(sourceUrls, effectiveServiceType)
    form.baseUrl = baseUrl
    form.baseUrls = baseUrls
    baseUrlsText.value = sourceUrls
    if (detectedApiKeys.value.length > 0) {
      form.apiKeys = [...detectedApiKeys.value]
    }
    if (generatedChannelName.value) {
      form.name = generatedChannelName.value
    }
    form.serviceType = effectiveServiceType
  }
  // 切换回快速模式时不做任何清理，保留 quickInput 原有内容
  isQuickMode.value = !isQuickMode.value
}

// 解析快速输入内容
const parseQuickInput = () => {
  const fallbackServiceType = props.channelType === 'images'
    ? getImagesServiceType(form.serviceType)
    : (form.serviceType || getDefaultServiceTypeValue())
  const result = parseQuickInputUtil(quickInput.value, fallbackServiceType)
  detectedBaseUrl.value = result.detectedBaseUrl
  detectedBaseUrls.value = result.detectedBaseUrls
  detectedRawBaseUrls.value = result.rawBaseUrls
  detectedApiKeys.value = result.detectedApiKeys
  detectedServiceType.value = props.channelType === 'images' ? 'openai' : result.detectedServiceType
}

// 获取默认服务类型
const getDefaultServiceType = (): string => {
  if (props.channelType === 'chat') {
    return 'OpenAI Chat'
  }
  if (props.channelType === 'gemini') {
    return 'Gemini'
  }
  if (props.channelType === 'responses') {
    return 'Responses (Codex)'
  }
  if (props.channelType === 'images') {
    return 'Images'
  }
  return 'Claude'
}

// 获取默认服务类型值
const getDefaultServiceTypeValue = (): 'openai' | 'gemini' | 'claude' | 'responses' => {
  if (props.channelType === 'chat') {
    return 'openai'
  }
  if (props.channelType === 'gemini') {
    return 'gemini'
  }
  if (props.channelType === 'responses') {
    return 'responses'
  }
  if (props.channelType === 'images') {
    return 'openai'
  }
  return 'claude'
}

// 获取默认 Base URL
const _getDefaultBaseUrl = (): string => {
  if (props.channelType === 'chat') {
    return 'https://api.openai.com/v1'
  }
  if (props.channelType === 'gemini') {
    return 'https://generativelanguage.googleapis.com'
  }
  if (props.channelType === 'responses') {
    return 'https://api.openai.com/v1'
  }
  if (props.channelType === 'images') {
    return 'https://api.openai.com/v1'
  }
  return 'https://api.anthropic.com'
}

// 快速模式表单验证
const isQuickFormValid = computed(() => {
  return detectedBaseUrls.value.length > 0 && detectedApiKeys.value.length > 0
})

// 生成随机字符串
const generateRandomString = (length: number): string => {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

// 从 URL 提取二级域名
const extractDomain = (url: string): string => {
  try {
    const hostname = new URL(url).hostname
    // 移除 www. 前缀
    const cleanHost = hostname.replace(/^www\./, '')
    const parts = cleanHost.split('.')

    // 处理特殊情况
    if (parts.length <= 1) {
      // localhost 等单段域名
      return cleanHost
    } else if (parts.length === 2) {
      // example.com → example
      return parts[0]
    } else {
      // api.openai.com → openai (取倒数第二段)
      return parts[parts.length - 2]
    }
  } catch {
    return 'channel'
  }
}

// 随机后缀和生成的渠道名称
const randomSuffix = ref(generateRandomString(6))

const generatedChannelName = computed(() => {
  if (!detectedBaseUrl.value) {
    return `channel-${randomSuffix.value}`
  }
  const domain = extractDomain(detectedBaseUrl.value)
  return `${domain}-${randomSuffix.value}`
})

// 预期请求 URL（模拟后端逻辑）
const _expectedRequestUrl = computed(() => {
  if (!detectedBaseUrl.value) return ''

  let baseUrl = detectedBaseUrl.value
  const skipVersion = baseUrl.endsWith('#')
  if (skipVersion) {
    baseUrl = baseUrl.slice(0, -1)
  }

  // 检查是否已包含版本号
  const hasVersion = /\/v\d+[a-z]*$/.test(baseUrl)

  // 根据渠道类型和服务类型确定端点（与后端逻辑一致）
  const serviceType = props.channelType === 'images'
    ? 'openai'
    : (detectedServiceType.value || getDefaultServiceTypeValue())
  const endpoint =
    props.channelType === 'images'
      ? '/images/generations'
      : props.channelType === 'responses'
        ? serviceType === 'responses'
          ? '/responses'
          : serviceType === 'claude'
            ? '/messages'
            : '/chat/completions'
        : serviceType === 'claude'
          ? '/messages'
          : serviceType === 'gemini'
            ? '/models/{model}:generateContent'
            : '/chat/completions'

  if (hasVersion || skipVersion) {
    return baseUrl + endpoint
  }
  // Gemini 使用 /v1beta，其他使用 /v1
  const versionPrefix = serviceType === 'gemini' ? '/v1beta' : '/v1'
  return baseUrl + versionPrefix + endpoint
})

// 生成单个 URL 的预期请求地址
const getExpectedRequestUrl = (inputBaseUrl: string): string => {
  if (!inputBaseUrl) return ''

  const serviceType = props.channelType === 'images'
    ? 'openai'
    : (detectedServiceType.value || getDefaultServiceTypeValue())
  const endpoint =
    props.channelType === 'images'
      ? '/images/generations'
      : props.channelType === 'responses'
        ? serviceType === 'responses'
          ? '/responses'
          : serviceType === 'claude'
            ? '/messages'
            : serviceType === 'gemini'
              ? '/models/{model}:generateContent'
              : '/chat/completions'
        : serviceType === 'claude'
          ? '/messages'
          : serviceType === 'gemini'
            ? '/models/{model}:generateContent'
            : serviceType === 'responses'
              ? props.channelType === 'chat'
                ? '/chat/completions'
                : '/responses'
              : '/chat/completions'

  return buildExpectedRequestUrl(serviceType, endpoint, inputBaseUrl)
}

// 检测 baseUrl 是否有验证错误
const baseUrlHasError = computed(() => {
  const value = form.baseUrl
  if (!value) return true
  try {
    new URL(value)
    return false
  } catch {
    return true
  }
})

// 详细模式所有 URL 的预期请求（支持多 BaseURL）
const formExpectedRequestUrls = computed(() => {
  const effectiveServiceType = props.channelType === 'images' ? 'openai' : form.serviceType
  return buildExpectedRequestUrls(props.channelType, effectiveServiceType, form.baseUrl, form.baseUrls)
})

// Workaround: Vuetify v-select menu 在 v-dialog 内首次打开时位置计算错误
// 通过 dispatch resize 强制重新计算菜单位置
const onMenuUpdate = (open: boolean) => {
  if (open) {
    setTimeout(() => window.dispatchEvent(new Event('resize')), 50)
  }
}

// 处理快速添加提交
const handleQuickSubmit = () => {
  if (!isQuickFormValid.value) return

  const channelData = {
    name: generatedChannelName.value,
    serviceType: props.channelType === 'images' ? 'openai' : (detectedServiceType.value || getDefaultServiceTypeValue()),
    baseUrl: detectedBaseUrl.value,
    baseUrls: detectedBaseUrls.value,
    apiKeys: detectedApiKeys.value,
    modelMapping: {}
  }

  // 传递 isQuickAdd 标志，让 App.vue 知道需要进行后续处理
  emit('save', channelData, { isQuickAdd: true })
}

// 服务类型选项 - 根据入口接口类型动态调整可用选项
const serviceTypeOptions = computed(() => {
  // 全部4种上游服务类型
  const allOptions = [
    { title: 'OpenAI Chat', value: 'openai' },
    { title: 'Claude', value: 'claude' },
    { title: 'Gemini', value: 'gemini' },
    { title: 'Responses (Codex)', value: 'responses' }
  ]

  // 根据入口接口类型调整排序（原生/默认类型排第一）
  const reorder = (options: typeof allOptions, first: string) => {
    const firstOption = options.find(o => o.value === first)
    const rest = options.filter(o => o.value !== first)
    return firstOption ? [firstOption, ...rest] : options
  }

  switch (props.channelType) {
    case 'messages':
      return reorder(allOptions, 'claude')
    case 'chat':
      // OpenAI Chat API 入口，OpenAI 原生排第一
      return reorder(allOptions, 'openai')
    case 'responses':
      // Responses API 入口，Responses 原生排第一
      return reorder(allOptions, 'responses')
    case 'images':
      return [{ title: 'OpenAI Images', value: 'openai' }]
    case 'gemini':
      // Gemini API 入口，Gemini 原生排第一
      return reorder(allOptions, 'gemini')
    default:
      return allOptions
  }
})

// 全部源模型选项 - 根据渠道类型动态显示
const allSourceModelOptions = computed(() => {
  if (props.channelType === 'chat') {
    // OpenAI Chat Completions 常用模型
    return [
      { title: 'codex', value: 'codex' },
      { title: 'gpt', value: 'gpt' },
      { title: 'mini', value: 'mini' },
      { title: 'gpt-5', value: 'gpt-5' },
      { title: 'gpt-5.5', value: 'gpt-5.5' },
      { title: 'gpt-5.4', value: 'gpt-5.4' },
      { title: 'gpt-5.3-codex', value: 'gpt-5.3-codex' },
      { title: 'gpt-5.2-codex', value: 'gpt-5.2-codex' },
      { title: 'gpt-5.2', value: 'gpt-5.2' }
    ]
  }
  if (props.channelType === 'images') {
    return [
      { title: 'gpt-image-2', value: 'gpt-image-2' },
      { title: 'gpt-image-1', value: 'gpt-image-1' },
      { title: 'dall-e-3', value: 'dall-e-3' },
      { title: 'dall-e-2', value: 'dall-e-2' }
    ]
  }
  if (props.channelType === 'gemini') {
    // Gemini API 常用模型别名
    return [
      { title: 'gemini-3.1-pro', value: 'gemini-3.1-pro' },
      { title: 'gemini-3-pro', value: 'gemini-3-pro' },
      { title: 'gemini-3-flash', value: 'gemini-3-flash' },
      { title: 'gemini-2.5-pro', value: 'gemini-2.5-pro' },
      { title: 'gemini-2.5-flash', value: 'gemini-2.5-flash' },
      { title: 'gemini-2.5-flash-lite', value: 'gemini-2.5-flash-lite' },
      { title: 'gemini-2', value: 'gemini-2' }
    ]
  }
  if (props.channelType === 'responses') {
    // Responses API (Codex) 常用模型名称
    return [
      { title: 'codex', value: 'codex' },
      { title: 'gpt-5', value: 'gpt-5' },
      { title: 'gpt', value: 'gpt' },
      { title: 'mini', value: 'mini' },
      { title: 'gpt-5.5', value: 'gpt-5.5' },
      { title: 'gpt-5.4', value: 'gpt-5.4' },
      { title: 'gpt-5.3-codex', value: 'gpt-5.3-codex' },
      { title: 'gpt-5.2-codex', value: 'gpt-5.2-codex' },
      { title: 'gpt-5.2', value: 'gpt-5.2' }
    ]
  } else {
    // Messages API (Claude) 常用模型别名
    return [
      { title: 'opus', value: 'opus' },
      { title: 'sonnet', value: 'sonnet' },
      { title: 'haiku', value: 'haiku' }
    ]
  }
})

// 可选的源模型选项 - 过滤掉已配置的模型
const sourceModelOptions = computed(() => {
  const configuredModels = Object.keys(form.modelMapping)
  return allSourceModelOptions.value.filter(opt => !configuredModels.includes(opt.value))
})

// 模型重定向的示例文本 - 根据渠道类型动态显示
const modelMappingHint = computed(() => {
  if (props.channelType === 'chat') {
    return t('addChannel.modelMappingHintChat')
  }
  if (props.channelType === 'images') {
    return t('addChannel.modelMappingHintChat')
  }
  if (props.channelType === 'gemini') {
    return t('addChannel.modelMappingHintGemini')
  }
  if (props.channelType === 'responses') {
    return t('addChannel.modelMappingHintResponses')
  } else {
    return t('addChannel.modelMappingHintMessages')
  }
})

const targetModelPlaceholder = computed(() => {
  if (props.channelType === 'chat') {
    return t('addChannel.targetModelPlaceholderChat')
  }
  if (props.channelType === 'images') {
    return t('addChannel.targetModelPlaceholderChat')
  }
  if (props.channelType === 'responses') {
    return t('addChannel.targetModelPlaceholderResponses')
  }
  if (props.channelType === 'gemini') {
    return t('addChannel.targetModelPlaceholderGemini')
  }
  return t('addChannel.targetModelPlaceholderMessages')
})

const reasoningEffortOptions = [
  { title: t('addChannel.reasoningDefault'), value: '' },
  { title: 'None', value: 'none' },
  { title: 'Low', value: 'low' },
  { title: 'Medium', value: 'medium' },
  { title: 'High', value: 'high' },
  { title: 'XHigh', value: 'xhigh' },
  { title: 'Max', value: 'max' }
]

const reasoningParamStyleOptions = [
  { title: 'reasoning.effort', value: 'reasoning' },
  { title: 'reasoning_effort', value: 'reasoning_effort' },
  { title: 'thinking (JD/GLM)', value: 'thinking' }
]

const textVerbosityOptions = [
  { title: 'Low', value: 'low' },
  { title: 'Medium', value: 'medium' },
  { title: 'High', value: 'high' }
]

const supportsOpenAIAdvancedOptions = computed(() => supportsAdvancedChannelOptions(form.serviceType))
const supportsChatRoleNormalization = computed(() => {
  return props.channelType === 'chat' || (props.channelType === 'responses' && form.serviceType === 'openai')
})

const showModelMappingPresets = computed(() => {
  return (props.channelType === 'messages' || props.channelType === 'responses') && (form.serviceType === 'openai' || form.serviceType === 'responses')
})
const modelNameCollator = new Intl.Collator('en', { numeric: true, sensitivity: 'base' })

const modelMappingPresets: Record<
  'gpt-5.5' | 'gpt-5.4' | 'gpt-5.3-codex' | 'gpt-5.2-codex',
  {
    modelMapping: Record<string, string>
    reasoningMapping: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>
    fastMode: boolean
    textVerbosity: 'low' | 'medium' | 'high'
  }
> = {
  'gpt-5.5': {
    modelMapping: {
      opus: 'gpt-5.5',
      sonnet: 'gpt-5.4',
      haiku: 'gpt-5.3-codex'
    },
    reasoningMapping: {
      opus: 'xhigh',
      sonnet: 'xhigh',
      haiku: 'high'
    },
    fastMode: true,
    textVerbosity: 'medium'
  },
  'gpt-5.4': {
    modelMapping: {
      opus: 'gpt-5.4',
      sonnet: 'gpt-5.4',
      haiku: 'gpt-5.3-codex'
    },
    reasoningMapping: {
      opus: 'xhigh',
      sonnet: 'xhigh',
      haiku: 'high'
    },
    fastMode: true,
    textVerbosity: 'medium'
  },
  'gpt-5.3-codex': {
    modelMapping: {
      opus: 'gpt-5.3-codex',
      sonnet: 'gpt-5.3-codex',
      haiku: 'gpt-5.3-codex'
    },
    reasoningMapping: {
      opus: 'xhigh',
      sonnet: 'xhigh',
      haiku: 'high'
    },
    fastMode: true,
    textVerbosity: 'medium'
  },
  'gpt-5.2-codex': {
    modelMapping: {
      opus: 'gpt-5.2',
      sonnet: 'gpt-5.2-codex',
      haiku: 'gpt-5.2-codex'
    },
    reasoningMapping: {
      opus: 'xhigh',
      sonnet: 'xhigh',
      haiku: 'high'
    },
    fastMode: true,
    textVerbosity: 'medium'
  }
}

const applyModelMappingPreset = (preset: keyof typeof modelMappingPresets) => {
  const presetConfig = modelMappingPresets[preset]
  form.modelMapping = { ...presetConfig.modelMapping }
  form.fastMode = presetConfig.fastMode
  form.textVerbosity = presetConfig.textVerbosity

  if (supportsOpenAIAdvancedOptions.value) {
    form.reasoningMapping = { ...presetConfig.reasoningMapping }
  } else {
    form.reasoningMapping = {}
  }
}

// 模型优先级排序规则（索引越小优先级越高）
// 规则顺序：先新后旧、先精确后宽松；同家族新版本在前，带 codex/pro/max 等精确后缀优先于通用名
// 数据基线：2026-05 各家官方在售模型
const modelPriorityPatterns: RegExp[] = [
  // Anthropic Claude（4.7 旗舰 / 4.6 Sonnet / 4.5 Haiku）
  /opus-4-7/i,
  /sonnet-4-7/i,
  /haiku-4-7/i,
  /opus-4-6/i,
  /sonnet-4-6/i,
  /haiku-4-6/i,
  /opus-4-5/i,
  /sonnet-4-5/i,
  /haiku-4-5/i,

  // OpenAI GPT-5 系列（pro / codex 变体优先匹配，再降级到主版本）
  /gpt-5\.5-pro/i,
  /gpt-5\.5/i,
  /gpt-5\.4-pro/i,
  /gpt-5\.4-mini/i,
  /gpt-5\.4-nano/i,
  /gpt-5\.4/i,
  /gpt-5\.3-codex/i,
  /gpt-5\.3/i,
  /gpt-5\.2-codex/i,
  /gpt-5\.2-pro/i,
  /gpt-5\.2/i,
  /gpt-5\.1-codex/i,
  /gpt-5\.1/i,
  /gpt-5-codex/i,
  /gpt-5-pro/i,
  /gpt-5/i,

  // Google Gemini（3.1 Pro 旗舰 → 3 Pro / Flash → 2.5 系列）
  /gemini-3\.1-pro/i,
  /gemini-3-pro/i,
  /gemini-3-flash/i,
  /gemini-3/i,
  /gemini-2\.5-pro/i,
  /gemini-2\.5-flash-lite/i,
  /gemini-2\.5-flash/i,

  // xAI Grok（4.3 当前旗舰；保留 4.2/4.1 以兼容旧 channel 命名）
  /grok-4\.3/i,
  /grok-4-3/i,
  /grok-4\.2/i,
  /grok-4\.1/i,
  /grok-4/i,

  // 智谱 GLM
  /glm-?5\.1/i,
  /glm-?5/i,
  /glm-?4\.7-flash/i,
  /glm-?4\.7/i,
  /glm-?4\.6/i,

  // 阿里 Qwen（3.6 / 3.5 / 3-Max）
  /qwen-?3\.6-plus/i,
  /qwen-?3\.6/i,
  /qwen-?3\.5/i,
  /qwen-?3-max/i,
  /qwen-?3-coder/i,
  /qwen-?3/i,

  // DeepSeek（V4 已发布；deepseek-chat / deepseek-reasoner 对应 V3.2）
  /deepseek-v4-pro/i,
  /deepseek-v4-flash/i,
  /deepseek-v4/i,
  /deepseek-v3\.2/i,
  /deepseek-reasoner/i,
  /deepseek-chat/i,
  /deepseek-v3/i,

  // Moonshot Kimi / MiniMax（带版本号 → 通用简写）
  /kimi-?k2\.6/i,
  /kimi-?k2\.5/i,
  /minimax-?m2\.7/i,
  /minimax-?m2\.5/i,
  /k2\.6/i,
  /k2\.5/i,
  /m2\.7/i,
  /m2\.5/i,

  // DeepSeek 兜底（匹配各种 deepseek- 前缀变体）
  /deepseek-/i,
]

const getModelPriority = (name: string): number => {
  for (let i = 0; i < modelPriorityPatterns.length; i++) {
    if (modelPriorityPatterns[i].test(name)) return i
  }
  return modelPriorityPatterns.length
}

const sortModelNamesDesc = (models: string[]): string[] => {
  return [...models].sort((a, b) => {
    const pa = getModelPriority(a)
    const pb = getModelPriority(b)
    if (pa !== pb) return pa - pb
    // 同优先级组内按自然降序
    return modelNameCollator.compare(b, a)
  })
}

// 表单数据
const form = reactive({
  name: '',
  serviceType: '' as 'openai' | 'gemini' | 'claude' | 'responses' | '',
  baseUrl: '',
  baseUrls: [] as string[],
  website: '',
  insecureSkipVerify: false,
  lowQuality: false,
  injectDummyThoughtSignature: false,
  stripThoughtSignature: false,
  passbackReasoningContent: false,
  stripEmptyTextBlocks: false,
  description: '',
  apiKeys: [] as string[],
  modelMapping: {} as Record<string, string>,
  reasoningMapping: {} as Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>,
  reasoningParamStyle: 'reasoning' as 'reasoning' | 'reasoning_effort' | 'thinking',
  textVerbosity: '' as 'low' | 'medium' | 'high' | '',
  fastMode: false,
  customHeaders: {} as Record<string, string>,
  proxyUrl: '',
  routePrefix: '',
  supportedModels: [] as string[],
  autoBlacklistBalance: true,
  normalizeMetadataUserId: true,
  codexNativeToolPassthrough: false,
  codexToolCompat: false,
  normalizeNonstandardChatRoles: false,
  stripCodexClientTools: false,
  noVision: false,
  noVisionModels: [] as string[],
  visionFallbackModel: '',
})

// 多 BaseURL 文本输入（独立变量，保留用户输入的换行）
const baseUrlsText = ref('')

// 监听 baseUrlsText 变化，同步到 form（去重等效 URL）
watch(baseUrlsText, val => {
  const { baseUrl, baseUrls } = syncBaseUrlsFormState(val, form.serviceType)
  form.baseUrl = baseUrl
  form.baseUrls = baseUrls
})

watch(() => form.serviceType, () => {
  const { baseUrl, baseUrls } = syncBaseUrlsFormState(baseUrlsText.value, form.serviceType)
  form.baseUrl = baseUrl
  form.baseUrls = baseUrls
})

// 原始密钥映射 (掩码密钥 -> 原始密钥)
const originalKeyMap = ref<Map<string, string>>(new Map())

// 新API密钥输入
const newApiKey = ref('')

// 密钥重复检测状态
const apiKeyError = ref('')
const duplicateKeyIndex = ref(-1)

// 处理 API 密钥输入事件
const handleApiKeyInput = () => {
  apiKeyError.value = ''
  duplicateKeyIndex.value = -1
}

// 复制功能相关状态
const copiedKeyIndex = ref<number | null>(null)

// 新模型映射输入
const newMapping = reactive({
  source: '',
  target: '',
  reasoningEffort: '' as 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max' | ''
})

// 自定义请求头输入
const newHeaderKey = ref('')
const newHeaderValue = ref('')

// 添加自定义请求头
const addCustomHeader = () => {
  const key = newHeaderKey.value.trim()
  const value = newHeaderValue.value.trim()
  if (key && value) {
    form.customHeaders[key] = value
    newHeaderKey.value = ''
    newHeaderValue.value = ''
  }
}

// 删除自定义请求头
const removeCustomHeader = (key: string) => {
  delete form.customHeaders[key]
}

function resetTransientUiState() {
  newApiKey.value = ''
  apiKeyError.value = ''
  duplicateKeyIndex.value = -1
  copiedKeyIndex.value = null
  newMapping.source = ''
  newMapping.target = ''
  newMapping.reasoningEffort = ''
  sourceMappingError.value = ''
  newHeaderKey.value = ''
  newHeaderValue.value = ''
  localRestoredKeys.value = new Set<string>()
  restoringKey.value = ''
  errors.name = ''
  errors.serviceType = ''
  errors.baseUrl = ''
  errors.website = ''
  formBaseUrlPreview.value = ''
}

// 安全地获取字符串值（处理 v-select/v-combobox 可能返回对象的情况）
const getStringValue = (val: string | { title: string; value: string } | null | undefined): string => {
  if (!val) return ''
  if (typeof val === 'string') return val
  return val.value || ''
}

// 源模型名验证错误
const sourceMappingError = ref('')

// 判断是否为内置源模型（内置选项允许更长名称）
const isPresetSourceModel = (val: string): boolean => {
  return allSourceModelOptions.value.some(opt => opt.value === val)
}

// 验证源模型名称（仅允许合法的模型名：字母、数字、连字符、下划线、点、斜杠）
const validateSourceModelName = (val: string): string => {
  if (!val) return ''
  if (!isPresetSourceModel(val) && val.length > 50) return t('addChannel.sourceModelNameTooLong')
  if (/\s/.test(val)) return t('addChannel.sourceModelNoSpaces')
  if (!/^[\w.\-/:@+]+$/.test(val)) return t('addChannel.sourceModelInvalidChars')
  return ''
}

// 检查映射输入是否有效
const isMappingInputValid = computed(() => {
  const source = getStringValue(newMapping.source).trim()
  const target = getStringValue(newMapping.target).trim()
  if (!source || !target) return false
  return !validateSourceModelName(source)
})

// 目标模型列表（从上游获取）
const targetModelOptions = ref<Array<{ title: string; value: string }>>([])
const fetchingModels = ref(false)
const fetchModelsError = ref('')
const hasTriedFetchModels = ref(false) // 标记是否已尝试获取过模型列表
const silentlySaving = ref(false)

// API Key 的 models 状态管理
interface KeyModelsStatus {
  loading: boolean
  success: boolean
  statusCode?: number
  error?: string
  modelCount?: number
}
const keyModelsStatus = ref<Map<string, KeyModelsStatus>>(new Map())

const restoreDisabledKeyLabelMap = {
  insufficient_balance: 'channelCard.blacklistReason.insufficient_balance',
  unavailable: 'channelCard.blacklistReason.unavailable',
  rate_limited: 'channelCard.blacklistReason.rate_limited',
  invalid: 'channelCard.blacklistReason.invalid',
  authentication_error: 'channelCard.blacklistReason.authentication_error',
  permission_error: 'channelCard.blacklistReason.permission_error',
  unknown: 'channelCard.blacklistReason.unknown',
} as const

const getRestoreDisabledKeyLabel = (reason?: string) => {
  return restoreDisabledKeyLabelMap[reason as keyof typeof restoreDisabledKeyLabelMap] || restoreDisabledKeyLabelMap.unknown
}

// 表单验证错误
const errors = reactive({
  name: '',
  serviceType: '',
  baseUrl: '',
  website: ''
})

// 验证规则
const rules = {
  required: (value: string) => !!value || t('addChannel.fieldRequired'),
  url: (value: string) => {
    try {
      new URL(value)
      return true
    } catch {
      return t('addChannel.invalidUrl')
    }
  },
  urlOptional: (value: string) => {
    if (!value) return true
    try {
      new URL(value)
      return true
    } catch {
      return t('addChannel.invalidUrl')
    }
  },
  baseUrls: (value: string) => {
    if (!value) return t('addChannel.fieldRequired')
    const urls = value
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
    if (urls.length === 0) return t('addChannel.atLeastOneUrl')
    for (const url of urls) {
      try {
        new URL(url)
      } catch {
        return t('addChannel.invalidUrlValue', { url })
      }
    }
    return true
  }
}

// 计算属性
const dialogMode = ref<'create' | 'edit'>('create')
const isEditing = computed(() => dialogMode.value === 'edit')
const hasDisabledKeysAvailable = computed(() => visibleDisabledKeys.value.length > 0)
const hasConfigurableKeys = computed(() => form.apiKeys.length > 0 || (isEditing.value && hasDisabledKeysAvailable.value))

const commonSupportedModelFilters = ['claude-*', 'gpt-5*', 'gpt-image-2', 'grok-4*', 'gemini-3*', '!*image*']

const selectedSupportedModelSet = computed(() => new Set(form.supportedModels))
const supportedModelsError = ref('')

// 动态header样式
const headerClasses = computed(() => {
  const isDark = theme.global.current.value.dark
  // Dark: keep neutral surface header; Light: use brand primary header
  return isDark ? 'bg-surface text-high-emphasis' : 'bg-primary text-white'
})

const avatarColor = computed(() => 'primary')

// Use Vuetify theme "on-primary" token so icon isn't fixed white
const headerIconStyle = computed(() => ({
  color: 'rgb(var(--v-theme-on-primary))'
}))

const subtitleClasses = computed(() => {
  const isDark = theme.global.current.value.dark
  // Dark mode: use medium emphasis; Light mode: use white with opacity for primary bg
  return isDark ? 'text-medium-emphasis' : 'text-white-subtitle'
})

const isFormValid = computed(() => {
  return (
    form.name.trim() && form.serviceType && form.baseUrl.trim() && isValidUrl(form.baseUrl) && hasConfigurableKeys.value
  )
})

// 工具函数
const isValidUrl = (url: string): boolean => {
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

const maskApiKey = (key: string): string => {
  if (key.length <= 10) return key.slice(0, 3) + '***' + key.slice(-2)
  return key.slice(0, 8) + '***' + key.slice(-5)
}

const normalizeStringArray = (values: string[]): string[] => values.map(v => v.trim()).filter(Boolean)

const handleSupportedModelsChange = (values: Array<string | { title: string; value: string }>) => {
  const normalizedValues = values
    .map(getStringValue)
    .map(v => v.trim())
    .filter(Boolean)

  const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(normalizedValues)
  form.supportedModels = validPatterns
  supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
}

const normalizeStringRecord = (record: Record<string, string>): Record<string, string> => {
  const normalized: Record<string, string> = {}
  Object.entries(record)
    .map(([key, value]) => [key.trim(), value.trim()] as const)
    .filter(([key, value]) => key && value)
    .sort(([keyA], [keyB]) => keyA.localeCompare(keyB))
    .forEach(([key, value]) => {
      normalized[key] = value
    })
  return normalized
}

const buildComparablePayload = () => {
  const payload = buildChannelPayload(form)
  return {
    ...payload,
    apiKeys: normalizeStringArray(payload.apiKeys),
    baseUrls: normalizeStringArray(payload.baseUrls || []),
    supportedModels: normalizeStringArray(payload.supportedModels || []),
    customHeaders: normalizeStringRecord(payload.customHeaders || {}),
    modelMapping: Object.fromEntries(Object.entries(payload.modelMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
    reasoningMapping: Object.fromEntries(Object.entries(payload.reasoningMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
    reasoningParamStyle: payload.reasoningParamStyle || 'reasoning'
  }
}

const hasEditableDraftChanges = computed(() => {
  if (!isEditing.value || !props.channel) return false
  const currentPayload = buildComparablePayload()
  const originalPayload = {
    name: props.channel.name.trim(),
    serviceType: props.channel.serviceType,
    baseUrl: props.channel.baseUrl || '',
    baseUrls: normalizeStringArray(props.channel.baseUrls || []),
    website: (props.channel.website || '').trim(),
    insecureSkipVerify: !!props.channel.insecureSkipVerify,
    lowQuality: !!props.channel.lowQuality,
    injectDummyThoughtSignature: !!props.channel.injectDummyThoughtSignature,
    stripThoughtSignature: !!props.channel.stripThoughtSignature,
    passbackReasoningContent: !!props.channel.passbackReasoningContent,
    stripEmptyTextBlocks: !!props.channel.stripEmptyTextBlocks,
    description: (props.channel.description || '').trim(),
    apiKeys: normalizeStringArray(props.channel.apiKeys || []),
    modelMapping: Object.fromEntries(Object.entries(props.channel.modelMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
    reasoningMapping: Object.fromEntries(Object.entries(props.channel.reasoningMapping || {}).sort(([a], [b]) => a.localeCompare(b))),
    reasoningParamStyle: props.channel.reasoningParamStyle || 'reasoning',
    textVerbosity: props.channel.textVerbosity || '',
    fastMode: !!props.channel.fastMode,
    customHeaders: normalizeStringRecord(props.channel.customHeaders || {}),
    proxyUrl: props.channel.proxyUrl || '',
    routePrefix: props.channel.routePrefix || '',
    supportedModels: normalizeStringArray(props.channel.supportedModels || []),
    autoBlacklistBalance: props.channel.autoBlacklistBalance ?? true,
    normalizeMetadataUserId: props.channel.normalizeMetadataUserId ?? true,
    codexNativeToolPassthrough: !!props.channel.codexNativeToolPassthrough,
    codexToolCompat: props.channel.codexToolCompat ?? props.channel.stripCodexClientTools ?? false,
    normalizeNonstandardChatRoles: !!props.channel.normalizeNonstandardChatRoles,
    stripCodexClientTools: props.channel.codexToolCompat ?? props.channel.stripCodexClientTools ?? false,
    noVision: !!props.channel.noVision,
    noVisionModels: [...(props.channel.noVisionModels || [])],
    visionFallbackModel: props.channel.visionFallbackModel || '',
  }

  return JSON.stringify(currentPayload) !== JSON.stringify(originalPayload)
})

const ensureLatestSavedChannel = async (): Promise<number | null> => {
  if (!isEditing.value || props.channel?.index === undefined || props.channel?.index === null) {
    return props.channel?.index ?? null
  }
  if (!hasEditableDraftChanges.value) {
    return props.channel.index
  }
  if (silentlySaving.value) {
    return null
  }

  if (formRef.value) {
    const { valid } = await formRef.value.validate()
    if (!valid) {
      return null
    }
  }

  silentlySaving.value = true
  try {
    const payload = buildChannelPayload(form)
    const result = await channelStore.saveChannel(payload, props.channel.index)
    await channelStore.refreshChannels()
    const latestChannel = (channelStore.currentChannelsData as any).channels?.find((ch: any) => ch.index === props.channel!.index) || null
    if (latestChannel) {
      dialogStore.editingChannel = latestChannel
    }
    return result.channelId ?? props.channel.index
  } catch (error) {
    const message = error instanceof Error ? error.message : t('system.unknown')
    emit('error', message)
    return null
  } finally {
    silentlySaving.value = false
  }
}

// 表单操作
const resetForm = () => {
  resetTransientUiState()
  form.name = ''
  form.serviceType = props.channelType === 'images' ? 'openai' : ''
  form.baseUrl = ''
  form.baseUrls = []
  form.website = ''
  form.insecureSkipVerify = false
  form.lowQuality = false
  form.injectDummyThoughtSignature = false
  form.stripThoughtSignature = false
  form.passbackReasoningContent = false
  form.stripEmptyTextBlocks = false
  form.description = ''
  form.apiKeys = []
  form.modelMapping = {}
  form.reasoningMapping = {}
  form.reasoningParamStyle = 'reasoning'
  form.textVerbosity = ''
  form.fastMode = false
  form.customHeaders = {}
  form.proxyUrl = ''
  form.routePrefix = ''
  form.supportedModels = []
  supportedModelsError.value = ''
  form.autoBlacklistBalance = true
  form.normalizeMetadataUserId = true
  form.codexNativeToolPassthrough = false
  form.codexToolCompat = false
  form.normalizeNonstandardChatRoles = false
  form.stripCodexClientTools = false
  form.noVision = false
  form.noVisionModels = []
  form.visionFallbackModel = ''

  // 重置 baseUrlsText
  baseUrlsText.value = ''

  // 清空原始密钥映射
  originalKeyMap.value.clear()

  // 清空模型缓存和状态
  targetModelOptions.value = []
  fetchingModels.value = false
  fetchModelsError.value = ''
  keyModelsStatus.value.clear()
  hasTriedFetchModels.value = false

  // 重置快速添加模式数据
  quickInput.value = ''
  detectedBaseUrl.value = ''
  detectedBaseUrls.value = []
  detectedRawBaseUrls.value = []
  detectedApiKeys.value = []
  detectedServiceType.value = null
  randomSuffix.value = generateRandomString(6)
}

const loadChannelData = (channel: Channel) => {
  resetTransientUiState()
  form.name = channel.name
  form.serviceType = props.channelType === 'images' ? 'openai' : channel.serviceType
  form.baseUrl = channel.baseUrl
  form.baseUrls = channel.baseUrls || []
  form.website = channel.website || ''
  form.insecureSkipVerify = !!channel.insecureSkipVerify
  form.lowQuality = !!channel.lowQuality
  form.injectDummyThoughtSignature = !!channel.injectDummyThoughtSignature
  form.stripThoughtSignature = !!channel.stripThoughtSignature
  form.passbackReasoningContent = !!channel.passbackReasoningContent
  form.stripEmptyTextBlocks = !!channel.stripEmptyTextBlocks
  form.description = channel.description || ''

  // 同步 baseUrlsText（优先使用 baseUrls，否则使用 baseUrl），保留用户显式配置的原始 URL 形式
  const rawUrls = channel.baseUrls && channel.baseUrls.length > 0
    ? channel.baseUrls
    : (channel.baseUrl ? [channel.baseUrl] : [])
  baseUrlsText.value = rawUrls.join('\n')

  // 直接存储原始密钥，不需要映射关系
  form.apiKeys = [...channel.apiKeys]

  // 清空原始密钥映射（现在不需要了）
  originalKeyMap.value.clear()

  form.modelMapping = { ...(channel.modelMapping || {}) }
  form.reasoningMapping = { ...(channel.reasoningMapping || {}) }
  form.reasoningParamStyle = channel.reasoningParamStyle || 'reasoning'
  form.textVerbosity = channel.textVerbosity || ''
  form.fastMode = !!channel.fastMode
  form.customHeaders = { ...(channel.customHeaders || {}) }
  form.proxyUrl = channel.proxyUrl || ''
  form.routePrefix = channel.routePrefix || ''
  const { validPatterns, hasInvalidPatterns } = filterValidSupportedModelPatterns(channel.supportedModels || [])
  form.supportedModels = validPatterns
  supportedModelsError.value = hasInvalidPatterns ? t('addChannel.supportedModelsInvalidPattern') : ''
  form.autoBlacklistBalance = channel.autoBlacklistBalance ?? true
  form.normalizeMetadataUserId = channel.normalizeMetadataUserId ?? true
  form.codexNativeToolPassthrough = !!channel.codexNativeToolPassthrough
  form.codexToolCompat = channel.codexToolCompat ?? channel.stripCodexClientTools ?? false
  form.normalizeNonstandardChatRoles = !!channel.normalizeNonstandardChatRoles
  form.stripCodexClientTools = channel.codexToolCompat ?? channel.stripCodexClientTools ?? false
  form.noVision = !!channel.noVision
  form.noVisionModels = [...(channel.noVisionModels || [])]
  form.visionFallbackModel = channel.visionFallbackModel || ''

  // 立即同步 baseUrl 到预览变量，避免等待 debounce
  formBaseUrlPreview.value = channel.baseUrl

  // 清空模型映射输入框
  newMapping.source = ''
  newMapping.target = ''

  // 清空模型缓存和状态（切换渠道时重置）
  targetModelOptions.value = []
  fetchingModels.value = false
  fetchModelsError.value = ''
  keyModelsStatus.value.clear()
  hasTriedFetchModels.value = false
}

const addApiKey = () => {
  const key = newApiKey.value.trim()
  if (!key) return

  // 重置错误状态
  apiKeyError.value = ''
  duplicateKeyIndex.value = -1

  // 检查是否与现有密钥重复
  const duplicateIndex = findDuplicateKeyIndex(key)
  if (duplicateIndex !== -1) {
    apiKeyError.value = t('addChannel.duplicateKeyExists')
    duplicateKeyIndex.value = duplicateIndex
    // 清除输入框，让用户重新输入
    newApiKey.value = ''
    return
  }

  // 直接存储原始密钥
  form.apiKeys.push(key)
  newApiKey.value = ''
}

// 检查密钥是否重复，返回重复密钥的索引，如果没有重复返回-1
const findDuplicateKeyIndex = (newKey: string): number => {
  return form.apiKeys.findIndex(existingKey => existingKey === newKey)
}

const removeApiKey = (index: number) => {
  form.apiKeys.splice(index, 1)

  // 如果删除的是当前高亮的重复密钥，清除高亮状态
  if (duplicateKeyIndex.value === index) {
    duplicateKeyIndex.value = -1
    apiKeyError.value = ''
  } else if (duplicateKeyIndex.value > index) {
    // 如果删除的密钥在高亮密钥之前，调整高亮索引
    duplicateKeyIndex.value--
  }
}

// 将指定密钥移到最上方
const moveApiKeyToTop = (index: number) => {
  if (index <= 0 || index >= form.apiKeys.length) return
  const [key] = form.apiKeys.splice(index, 1)
  form.apiKeys.unshift(key)
  duplicateKeyIndex.value = -1
  copiedKeyIndex.value = null
}

// 将指定密钥移到最下方
const moveApiKeyToBottom = (index: number) => {
  if (index < 0 || index >= form.apiKeys.length - 1) return
  const [key] = form.apiKeys.splice(index, 1)
  form.apiKeys.push(key)
  duplicateKeyIndex.value = -1
  copiedKeyIndex.value = null
}

// 恢复被拉黑的密钥
const restoringKey = ref('')
const localRestoredKeys = ref(new Set<string>())

// 本地过滤已恢复的 key，不直接修改 props
const visibleDisabledKeys = computed(() =>
  (props.channel?.disabledApiKeys || []).filter(dk => !localRestoredKeys.value.has(dk.key))
)

const restoreDisabledKey = async (apiKey: string) => {
  if (!props.channel) return
  restoringKey.value = apiKey
  try {
    const channelId = props.channel.index
    if (props.channelType === 'chat') {
      await apiService.restoreChatApiKey(channelId, apiKey)
    } else if (props.channelType === 'images') {
      await apiService.restoreImagesApiKey(channelId, apiKey)
    } else if (props.channelType === 'gemini') {
      await apiService.restoreGeminiApiKey(channelId, apiKey)
    } else if (props.channelType === 'responses') {
      await apiService.restoreResponsesApiKey(channelId, apiKey)
    } else {
      await apiService.restoreApiKey(channelId, apiKey)
    }
    // 本地标记已恢复，加入活跃列表
    localRestoredKeys.value.add(apiKey)
    form.apiKeys.push(apiKey)
  } catch (error) {
    apiKeyError.value = error instanceof Error ? error.message : 'Restore failed'
  } finally {
    restoringKey.value = ''
  }
}

// 复制API密钥到剪贴板
const copyApiKey = async (key: string, index: number) => {
  try {
    await navigator.clipboard.writeText(key)
    copiedKeyIndex.value = index

    // 2秒后重置复制状态
    setTimeout(() => {
      copiedKeyIndex.value = null
    }, 2000)
  } catch (err) {
    console.error('复制密钥失败:', err)
    // 降级方案：使用传统的复制方法
    const textArea = document.createElement('textarea')
    textArea.value = key
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    textArea.style.top = '-999999px'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()

    try {
      document.execCommand('copy')
      copiedKeyIndex.value = index

      setTimeout(() => {
        copiedKeyIndex.value = null
      }, 2000)
    } catch (err) {
      console.error('降级复制方案也失败:', err)
    } finally {
      textArea.remove()
    }
  }
}

// 处理源模型名输入变化，实时验证
const handleSourceModelChange = (val: string | { title: string; value: string } | null) => {
  const source = getStringValue(val).trim()
  if (!source) {
    sourceMappingError.value = ''
    return
  }
  sourceMappingError.value = validateSourceModelName(source)
}

const addModelMapping = () => {
  const source = getStringValue(newMapping.source).trim()
  const target = getStringValue(newMapping.target).trim()

  // 验证源模型名
  const sourceErr = validateSourceModelName(source)
  if (sourceErr) {
    sourceMappingError.value = sourceErr
    return
  }
  sourceMappingError.value = ''

  if (source && target && !form.modelMapping[source]) {
    form.modelMapping[source] = target
    if (supportsOpenAIAdvancedOptions.value && newMapping.reasoningEffort) {
      form.reasoningMapping[source] = newMapping.reasoningEffort
    } else {
      delete form.reasoningMapping[source]
    }
    newMapping.source = ''
    newMapping.target = ''
    newMapping.reasoningEffort = ''
  }
}

const removeModelMapping = (source: string) => {
  const target = form.modelMapping[source]
  delete form.modelMapping[source]
  delete form.reasoningMapping[source]
  if (target) {
    const idx = form.noVisionModels.indexOf(target)
    if (idx >= 0) form.noVisionModels.splice(idx, 1)
  }
}

const isModelNoVision = (model: string): boolean => {
  return form.noVisionModels.includes(model)
}

const toggleModelVision = (model: string) => {
  const idx = form.noVisionModels.indexOf(model)
  if (idx >= 0) {
    form.noVisionModels.splice(idx, 1)
  } else {
    form.noVisionModels.push(model)
  }
}

const isSupportedModelSelected = (filter: string): boolean => {
  return selectedSupportedModelSet.value.has(filter)
}

const appendSupportedModelFilter = (filter: string) => {
  if (isSupportedModelSelected(filter)) {
    return
  }
  form.supportedModels.push(filter)
  supportedModelsError.value = ''
}

// 处理目标模型输入框点击事件(仅在首次或有新 key 时触发请求)
const handleTargetModelClick = () => {
  // 如果已经尝试过获取且正在加载中,不重复触发
  if (hasTriedFetchModels.value || fetchingModels.value) {
    return
  }

  // 标记已尝试获取
  hasTriedFetchModels.value = true

  // 调用获取模型列表(内部有缓存逻辑)
  fetchTargetModels()
}

const ensureTargetModelsLoaded = () => {
  if (targetModelOptions.value.length === 0) {
    fetchTargetModels()
  }
}

const fetchTargetModels = async () => {
  const candidateKeys = form.apiKeys.length > 0
    ? form.apiKeys
    : (isEditing.value ? visibleDisabledKeys.value.map(dk => dk.key) : [])

  if (!form.baseUrl || candidateKeys.length === 0) {
    fetchModelsError.value = t('addChannel.fillBaseUrlAndApiKey')
    return
  }

  const channelId = props.channel?.index
  if (isEditing.value) {
    const savedChannelId = await ensureLatestSavedChannel()
    if (savedChannelId === null) {
      hasTriedFetchModels.value = false
      fetchingModels.value = false
      return
    }
  }

  // 仅为未检测过的 API Key 发起请求
  const uncheckedKeys = candidateKeys.filter(key => !keyModelsStatus.value.has(key))

  if (uncheckedKeys.length === 0) {
    return
  }

  fetchingModels.value = true
  fetchModelsError.value = ''

  // modelsApiType 决定请求协议（Bearer/x-goog-api-key、/v1/models vs /v1beta/models）
  // 对于 gemini 渠道组内配置为 openai/claude serviceType 的渠道，应走对应协议而非 Gemini 协议
  const effectiveServiceType = props.channelType === 'images'
    ? 'openai'
    : (form.serviceType || detectedServiceType.value || getDefaultServiceTypeValue())
  let modelsApiType: 'messages' | 'responses' | 'chat' | 'gemini' | 'images'
  if (props.channelType === 'images') {
    modelsApiType = 'images'
  } else if (effectiveServiceType === 'gemini') {
    modelsApiType = 'gemini'
  } else if (effectiveServiceType === 'responses') {
    modelsApiType = 'responses'
  } else if (effectiveServiceType === 'openai') {
    modelsApiType = 'chat'
  } else {
    modelsApiType = 'messages'
  }

  const requestOverrides = {
    baseUrl: form.baseUrl || undefined,
    proxyUrl: form.proxyUrl || undefined,
    insecureSkipVerify: form.insecureSkipVerify || undefined,
    customHeaders: Object.keys(form.customHeaders).length > 0 ? { ...form.customHeaders } : undefined,
  }

  // 每个 unchecked key 并发独立请求
  const keyPromises = uncheckedKeys.map(async (apiKey) => {
    keyModelsStatus.value.set(apiKey, { loading: true, success: false })

    try {
      let response: any
      const id = channelId ?? 0
      const request = { key: apiKey, ...requestOverrides }

      switch (modelsApiType) {
        case 'messages':
          response = await apiService.getChannelModels(id, request)
          break
        case 'responses':
          response = await apiService.getResponsesChannelModels(id, request)
          break
        case 'chat':
          response = await apiService.getChatChannelModels(id, request)
          break
        case 'images':
          response = await apiService.getImagesChannelModels(id, request)
          break
        case 'gemini':
          response = await apiService.getGeminiChannelModels(id, request)
          break
      }

      keyModelsStatus.value.set(apiKey, {
        loading: false,
        success: true,
        statusCode: 200,
        modelCount: response.data.length
      })
      return response.data as { id: string }[]
    } catch (error) {
      let errorMsg = t('addChannel.unknownError')
      let statusCode = 0
      if (error instanceof ApiError) {
        errorMsg = error.message
        statusCode = error.status
      } else if (error instanceof Error) {
        errorMsg = error.message
      }
      keyModelsStatus.value.set(apiKey, {
        loading: false,
        success: false,
        statusCode,
        error: errorMsg
      })
      return [] as { id: string }[]
    }
  })

  try {
    const results = await Promise.all(keyPromises)

    const allModels = new Set<string>(targetModelOptions.value.map(opt => opt.value))
    results.forEach(models => models.forEach(m => allModels.add(m.id)))

    targetModelOptions.value = sortModelNamesDesc(Array.from(allModels)).map(id => ({ title: id, value: id }))

    const allFailed = candidateKeys.every(key => {
      const s = keyModelsStatus.value.get(key)
      return s && !s.success
    })
    if (allFailed) {
      fetchModelsError.value = t('addChannel.allApiKeysModelsFailed')
    }
  } finally {
    fetchingModels.value = false
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return

  const { valid } = await formRef.value.validate()
  if (!valid) return

  const channelData = buildChannelPayload(form)

  emit('save', channelData)
}

const handleCancel = () => {
  emit('update:show', false)
  resetForm()
}

const PAYLOAD_KEYS = [
  'name', 'serviceType', 'baseUrl', 'baseUrls', 'website', 'insecureSkipVerify',
  'lowQuality', 'injectDummyThoughtSignature', 'stripThoughtSignature', 'description',
  'apiKeys', 'modelMapping', 'reasoningMapping', 'reasoningParamStyle', 'textVerbosity',
  'fastMode', 'customHeaders', 'proxyUrl', 'routePrefix', 'supportedModels',
  'autoBlacklistBalance', 'normalizeMetadataUserId', 'stripEmptyTextBlocks', 'codexNativeToolPassthrough',
  'codexToolCompat', 'normalizeNonstandardChatRoles', 'stripCodexClientTools'
] as const

function extractPayloadFields(channel: Channel): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const key of PAYLOAD_KEYS) {
    if (key in channel) {
      result[key] = channel[key as keyof Channel]
    }
  }
  return result
}

const handleTestCapability = async () => {
  if (props.channel?.index === undefined || props.channel?.index === null) {
    return
  }

  if (!formRef.value) return
  const { valid } = await formRef.value.validate()
  if (!valid) return

  const channelData = buildChannelPayload(form)
  const original = extractPayloadFields(props.channel)
  const hasChanges = JSON.stringify(channelData) !== JSON.stringify(original)

  if (hasChanges) {
    emit('save', channelData, { triggerCapabilityTest: true })
  } else {
    emit('testCapability', props.channel.index)
  }
}

// 监听props变化
watch(
  () => props.show,
  newShow => {
    if (newShow) {
      dialogMode.value = props.channel ? 'edit' : 'create'

      // 无论是编辑还是新增，都先清理密钥错误状态
      apiKeyError.value = ''
      duplicateKeyIndex.value = -1
      localRestoredKeys.value = new Set<string>()

      if (dialogMode.value === 'edit' && props.channel) {
        // 编辑模式：使用表单模式
        isQuickMode.value = false
        loadChannelData(props.channel)
      } else {
        // 添加模式：默认使用快速模式
        isQuickMode.value = true
        resetForm()
      }
    }
  }
)

watch(
  () => props.channel,
  (newChannel, oldChannel) => {
    const action = resolveChannelWatcherAction({
      show: props.show,
      newChannel,
      oldChannel,
    })

    if (action === 'load-edit-channel' && newChannel) {
      dialogMode.value = 'edit'
      isQuickMode.value = false
      loadChannelData(newChannel)
      return
    }

    if (action === 'reset-new-form') {
      dialogMode.value = 'create'
      isQuickMode.value = true
      resetForm()
    }
  }
)

watch(
  () => form.baseUrl,
  value => {
    if (formBaseUrlPreviewTimer !== null) {
      window.clearTimeout(formBaseUrlPreviewTimer)
    }
    formBaseUrlPreviewTimer = window.setTimeout(() => {
      formBaseUrlPreview.value = value
    }, 200)
  },
  { immediate: true }
)

watch(
  () => JSON.stringify({
    baseUrl: form.baseUrl,
    baseUrls: form.baseUrls,
    apiKeys: form.apiKeys,
    proxyUrl: form.proxyUrl,
    insecureSkipVerify: form.insecureSkipVerify,
    customHeaders: form.customHeaders,
    serviceType: form.serviceType,
    routePrefix: form.routePrefix,
  }),
  () => {
    targetModelOptions.value = []
    keyModelsStatus.value.clear()
    hasTriedFetchModels.value = false
    fetchModelsError.value = ''
  }
)

// ESC键监听
const handleKeydown = (event: Event) => {
  const keyboardEvent = event as KeyboardEvent
  if (keyboardEvent.key === 'Escape' && props.show) {
    handleCancel()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  if (formBaseUrlPreviewTimer !== null) {
    window.clearTimeout(formBaseUrlPreviewTimer)
  }
})
</script>

<style scoped>
/* 基础URL下方的提示区域 - 固定高度防止布局跳动 */
.base-url-hint {
  min-height: 20px;
  padding: 4px 12px 8px;
  line-height: 1.5;
}

.modal-header-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.modal-title {
  font-size: 1.125rem;
  line-height: 1.3;
  font-weight: 600;
  letter-spacing: 0;
}

.modal-subtitle {
  font-size: 0.8125rem;
  line-height: 1.5;
}

.section-title {
  font-size: 0.875rem;
  line-height: 1.4;
  font-weight: 600;
  letter-spacing: 0;
}

.section-title--soft {
  font-size: 0.875rem;
  font-weight: 500;
}

.section-card-title {
  font-size: 0.875rem !important;
  line-height: 1.4;
  font-weight: 600;
}

/* 多个预期请求项样式 */
.expected-request-item + .expected-request-item {
  margin-top: 2px;
}

/* 浅色模式下副标题使用白色带透明度 */
.text-white-subtitle {
  color: rgba(255, 255, 255, 0.78) !important;
}

.animate-pulse {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

:deep(.key-tooltip) {
  color: rgba(var(--v-theme-on-surface), 0.92);
  background-color: rgba(var(--v-theme-surface), 0.98);
  border: 1px solid rgba(var(--v-theme-primary), 0.45);
  font-weight: 600;
  letter-spacing: 0;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.06);
}

/* 快速添加模式样式 */
.quick-input-textarea :deep(textarea) {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.detection-status-card {
  background: rgba(var(--v-theme-surface-variant), 0.3);
}

/* 多 Base URL 项目样式 */
.base-url-item {
  padding: 6px 10px;
  background: rgba(var(--v-theme-surface-variant), 0.4);
  border-radius: 6px;
  border-left: 2px solid rgb(var(--v-theme-success));
}

.base-url-item + .base-url-item {
  margin-top: 4px;
}

.mode-toggle-btn {
  text-transform: none;
  font-size: 0.8125rem;
  font-weight: 600;
  letter-spacing: 0;
  padding-inline: 12px;
}

.mode-toggle-btn :deep(.v-btn__content) {
  gap: 4px;
  line-height: 1.5;
}

.capability-test-btn {
  text-transform: none;
  font-size: 0.8125rem;
  font-weight: 600;
  letter-spacing: 0;
  padding-inline: 12px;
}

.header-capability-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.capability-test-btn :deep(.v-btn__content) {
  gap: 4px;
  line-height: 1.5;
}

/* 高级选项中的右侧开关行 */
.advanced-switch-row {
  min-height: 56px;
}

.advanced-switch-row :deep(.v-selection-control) {
  justify-content: flex-end;
  margin-inline-start: 16px;
}

.channel-config-select {
  flex: 0 0 220px;
}

@media (max-width: 600px) {
  .channel-config-select {
    flex-basis: 100%;
  }
}

/* 暗色模式彩色按钮：深色文字，保证对比度 */
:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-primary),
:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-secondary),
:deep(.v-theme--dark .v-btn.v-btn--variant-flat.v-btn--color-success) {
  color: #1e1b4b !important;
}

:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-primary) {
  background-color: rgba(129, 140, 248, 0.92) !important;
}

:deep(.v-theme--dark .v-btn.v-btn--variant-elevated.v-btn--color-secondary) {
  background-color: rgba(167, 139, 250, 0.92) !important;
}

:deep(.v-theme--dark .v-btn.v-btn--variant-flat.v-btn--color-success) {
  background-color: rgba(52, 211, 153, 0.92) !important;
}

</style>
