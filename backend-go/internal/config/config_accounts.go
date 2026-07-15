package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/BenedictKing/ccx/internal/utils"
)

// AccountChannel 是账号级管理 API 使用的渠道快照。
type AccountChannel struct {
	Kind     string
	Upstream UpstreamConfig
}

// AccountChannelUpdate 描述一次账号更新中单条协议渠道的新凭证绑定。
type AccountChannelUpdate struct {
	ChannelUID   string
	Name         string
	APIKeys      []string
	APIKeyConfig []APIKeyConfig
	BaseURLs     []string
}

// AccountChannelAddition 描述账号事务中需要新增的一条协议渠道。
type AccountChannelAddition struct {
	Kind     string
	Upstream UpstreamConfig
}

// GetAccountChannels 返回账号下全部协议渠道的深拷贝。
func (cm *ConfigManager) GetAccountChannels(accountUID string) []AccountChannel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []AccountChannel
	visit := func(kind string, channels []UpstreamConfig) {
		for i := range channels {
			if channels[i].AccountUID != accountUID {
				continue
			}
			result = append(result, AccountChannel{Kind: kind, Upstream: *channels[i].Clone()})
		}
	}
	visit("messages", cm.config.Upstream)
	visit("chat", cm.config.ChatUpstream)
	visit("responses", cm.config.ResponsesUpstream)
	visit("gemini", cm.config.GeminiUpstream)
	visit("images", cm.config.ImagesUpstream)
	visit("vectors", cm.config.VectorsUpstream)
	return result
}

// GetManagedAccountCredential 返回账号凭证的副本。
func (cm *ConfigManager) GetManagedAccountCredential(accountUID, credentialUID string) (ManagedAccountCredential, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, account := range cm.config.ManagedAccounts {
		if account.AccountUID != accountUID {
			continue
		}
		for _, credential := range account.Credentials {
			if credential.CredentialUID == credentialUID {
				if credential.VolcengineAccessKey != nil {
					pair := *credential.VolcengineAccessKey
					credential.VolcengineAccessKey = &pair
				}
				if credential.MiMoConsole != nil {
					console := *credential.MiMoConsole
					credential.MiMoConsole = &console
				}
				return credential, true
			}
		}
	}
	return ManagedAccountCredential{}, false
}

// BindManagedAccountMiMoConsole 绑定 MiMo 控制台 Cookie，并可原子采用 Cookie 所属的 Token Plan Key。
func (cm *ConfigManager) BindManagedAccountMiMoConsole(accountUID, credentialUID, replacementKey string, console MiMoConsoleCredential) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	var credential *ManagedAccountCredential
	for i := range cm.config.ManagedAccounts {
		account := &cm.config.ManagedAccounts[i]
		if account.AccountUID != accountUID {
			continue
		}
		if account.ProviderID != "mimo" {
			return fmt.Errorf("仅 MiMo 自动托管账号支持绑定控制台 Cookie")
		}
		for j := range account.Credentials {
			if account.Credentials[j].CredentialUID == credentialUID {
				credential = &account.Credentials[j]
				break
			}
		}
		if credential == nil {
			return fmt.Errorf("凭证 %s 不存在", credentialUID)
		}
		break
	}
	if credential == nil {
		return fmt.Errorf("账号 %s 不存在", accountUID)
	}
	replacementKey = strings.TrimSpace(replacementKey)
	oldKey := credential.APIKey
	if replacementKey != "" && replacementKey != oldKey {
		for _, account := range cm.config.ManagedAccounts {
			if account.AccountUID != accountUID {
				continue
			}
			for _, existing := range account.Credentials {
				if existing.CredentialUID != credentialUID && existing.APIKey == replacementKey {
					return fmt.Errorf("Cookie 所属 Key 已存在于当前账号")
				}
			}
		}
		replaceKey := func(channels []UpstreamConfig) {
			for i := range channels {
				channel := &channels[i]
				if channel.AccountUID != accountUID {
					continue
				}
				replaced := false
				for j := range channel.APIKeys {
					if channel.APIKeys[j] == oldKey {
						channel.APIKeys[j] = replacementKey
						replaced = true
					}
				}
				for j := range channel.APIKeyConfigs {
					cfg := &channel.APIKeyConfigs[j]
					if cfg.CredentialUID == credentialUID || cfg.Key == oldKey {
						cfg.Key = replacementKey
						cfg.CredentialUID = credentialUID
						replaced = true
					}
				}
				if replaced && oldKey != "" && !accountContainsString(channel.HistoricalAPIKeys, oldKey) {
					channel.HistoricalAPIKeys = append(channel.HistoricalAPIKeys, oldKey)
				}
			}
		}
		replaceKey(cm.config.Upstream)
		replaceKey(cm.config.ChatUpstream)
		replaceKey(cm.config.ResponsesUpstream)
		replaceKey(cm.config.GeminiUpstream)
		replaceKey(cm.config.ImagesUpstream)
		replaceKey(cm.config.VectorsUpstream)
		credential.APIKey = replacementKey
	}
	console.Cookie = strings.TrimSpace(console.Cookie)
	credential.MiMoConsole = &console
	return cm.saveConfigLocked(cm.config)
}

func (cm *ConfigManager) ClearManagedAccountMiMoConsole(accountUID, credentialUID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for i := range cm.config.ManagedAccounts {
		account := &cm.config.ManagedAccounts[i]
		if account.AccountUID != accountUID {
			continue
		}
		for j := range account.Credentials {
			if account.Credentials[j].CredentialUID == credentialUID {
				account.Credentials[j].MiMoConsole = nil
				return cm.saveConfigLocked(cm.config)
			}
		}
		return fmt.Errorf("凭证 %s 不存在", credentialUID)
	}
	return fmt.Errorf("账号 %s 不存在", accountUID)
}

func accountContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// SetManagedAccountVolcengineAccessKey 为一个推理 Key 绑定火山云签名凭证。
func (cm *ConfigManager) SetManagedAccountVolcengineAccessKey(accountUID, credentialUID, accessKeyID, secretAccessKey string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	accessKeyID = strings.TrimSpace(accessKeyID)
	secretAccessKey = strings.TrimSpace(secretAccessKey)
	if accessKeyID == "" || secretAccessKey == "" {
		return fmt.Errorf("Access Key ID 和 Secret Access Key 均不能为空")
	}
	for i := range cm.config.ManagedAccounts {
		account := &cm.config.ManagedAccounts[i]
		if account.AccountUID != accountUID {
			continue
		}
		for j := range account.Credentials {
			if account.Credentials[j].CredentialUID != credentialUID {
				continue
			}
			account.Credentials[j].VolcengineAccessKey = &VolcengineAccessKeyPair{
				AccessKeyID: accessKeyID, SecretAccessKey: secretAccessKey,
			}
			return cm.saveConfigLocked(cm.config)
		}
		return fmt.Errorf("凭证 %s 不存在", credentialUID)
	}
	return fmt.Errorf("账号 %s 不存在", accountUID)
}

// ClearManagedAccountVolcengineAccessKey 删除推理 Key 绑定的火山云签名凭证。
func (cm *ConfigManager) ClearManagedAccountVolcengineAccessKey(accountUID, credentialUID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for i := range cm.config.ManagedAccounts {
		account := &cm.config.ManagedAccounts[i]
		if account.AccountUID != accountUID {
			continue
		}
		for j := range account.Credentials {
			if account.Credentials[j].CredentialUID != credentialUID {
				continue
			}
			account.Credentials[j].VolcengineAccessKey = nil
			return cm.saveConfigLocked(cm.config)
		}
		return fmt.Errorf("凭证 %s 不存在", credentialUID)
	}
	return fmt.Errorf("账号 %s 不存在", accountUID)
}

// SetManagedAccountVolcenginePlan 保存由火山管控面自动识别出的套餐信息。
func (cm *ConfigManager) SetManagedAccountVolcenginePlan(accountUID, credentialUID, plan, tier, status string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for i := range cm.config.ManagedAccounts {
		account := &cm.config.ManagedAccounts[i]
		if account.AccountUID != accountUID {
			continue
		}
		for j := range account.Credentials {
			credential := &account.Credentials[j]
			if credential.CredentialUID != credentialUID || credential.VolcengineAccessKey == nil {
				continue
			}
			credential.VolcengineAccessKey.Plan = strings.TrimSpace(plan)
			credential.VolcengineAccessKey.PlanTier = strings.TrimSpace(tier)
			credential.VolcengineAccessKey.PlanStatus = strings.TrimSpace(status)
			return cm.saveConfigLocked(cm.config)
		}
		return fmt.Errorf("凭证 %s 不存在或未绑定火山 Access Key", credentialUID)
	}
	return fmt.Errorf("账号 %s 不存在", accountUID)
}

// SetManagedAccountVolcenginePlanUsage 保存火山管控面查询到的套餐用量快照。
func (cm *ConfigManager) SetManagedAccountVolcenginePlanUsage(accountUID, credentialUID string, usage *VolcenginePlanUsage) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for i := range cm.config.ManagedAccounts {
		account := &cm.config.ManagedAccounts[i]
		if account.AccountUID != accountUID {
			continue
		}
		for j := range account.Credentials {
			credential := &account.Credentials[j]
			if credential.CredentialUID != credentialUID || credential.VolcengineAccessKey == nil {
				continue
			}
			credential.VolcengineAccessKey.Usage = usage
			return cm.saveConfigLocked(cm.config)
		}
		return fmt.Errorf("凭证 %s 不存在或未绑定火山 Access Key", credentialUID)
	}
	return fmt.Errorf("账号 %s 不存在", accountUID)
}

// mergeManagedProviderAccounts 将同一官方 provider 的历史自动托管账号归并为一个凭证池。
// provider 模板已经描述完整协议 routes，因此账号身份应是 provider 级，而不是每个 Key 一份。
func (cm *ConfigManager) mergeManagedProviderAccounts() bool {
	canonicalUID := make(map[string]string)
	canonicalName := make(map[string]string)
	providerAccounts := make(map[string]map[string]bool)
	for _, account := range cm.config.ManagedAccounts {
		if account.ProviderID == "" || account.AccountUID == "" {
			continue
		}
		// 保留最后创建的账号身份和名称，符合用户最近一次添加时看到的名称。
		canonicalUID[account.ProviderID] = account.AccountUID
		canonicalName[account.ProviderID] = account.Name
		if providerAccounts[account.ProviderID] == nil {
			providerAccounts[account.ProviderID] = make(map[string]bool)
		}
		providerAccounts[account.ProviderID][account.AccountUID] = true
	}

	updated := false
	providerKinds := make(map[string]map[string]bool)
	collectKinds := func(channels []UpstreamConfig, kind string) {
		for i := range channels {
			channel := &channels[i]
			if channel.AutoManaged && channel.ProviderID != "" {
				if providerKinds[channel.ProviderID] == nil {
					providerKinds[channel.ProviderID] = make(map[string]bool)
				}
				providerKinds[channel.ProviderID][kind] = true
				if providerAccounts[channel.ProviderID] == nil {
					providerAccounts[channel.ProviderID] = make(map[string]bool)
				}
				if channel.AccountUID != "" {
					providerAccounts[channel.ProviderID][channel.AccountUID] = true
				}
			}
		}
	}
	collectKinds(cm.config.Upstream, "messages")
	collectKinds(cm.config.ChatUpstream, "chat")
	collectKinds(cm.config.ResponsesUpstream, "responses")
	collectKinds(cm.config.GeminiUpstream, "gemini")
	collectKinds(cm.config.ImagesUpstream, "images")
	collectKinds(cm.config.VectorsUpstream, "vectors")

	mergeKind := func(channels []UpstreamConfig, kind string) []UpstreamConfig {
		out := make([]UpstreamConfig, 0, len(channels))
		providerIndex := make(map[string]int)
		providerHasCanonicalRoute := make(map[string]bool)
		for i := range channels {
			channel := channels[i]
			if !channel.AutoManaged || channel.ProviderID == "" {
				out = append(out, channel)
				continue
			}
			providerID := channel.ProviderID
			if len(providerAccounts[providerID]) < 2 {
				out = append(out, channel)
				continue
			}
			originalAccountUID := channel.AccountUID
			uid := canonicalUID[providerID]
			if uid == "" {
				uid = channel.AccountUID
				if uid == "" {
					uid = GenerateAccountUID()
				}
				canonicalUID[providerID] = uid
			}
			baseName := canonicalName[providerID]
			if baseName == "" {
				baseName = managedAccountName(channel.Name)
				canonicalName[providerID] = baseName
			}
			targetName := baseName
			if len(providerKinds[providerID]) > 1 {
				targetName += accountChannelSuffix(kind)
			}
			if channel.AccountUID != uid || channel.Name != targetName {
				updated = true
			}
			channel.AccountUID = uid
			channel.Name = targetName
			channel.APIKeyConfigs = normalizeAPIKeyConfigs(channel.APIKeys, channel.APIKeyConfigs)
			for j := range channel.APIKeyConfigs {
				channel.APIKeyConfigs[j].CredentialUID = GenerateCredentialUID(uid, channel.APIKeyConfigs[j].Key)
			}

			idx, exists := providerIndex[providerID]
			if !exists {
				providerIndex[providerID] = len(out)
				providerHasCanonicalRoute[providerID] = originalAccountUID == uid
				out = append(out, channel)
				continue
			}

			merged := &out[idx]
			if originalAccountUID == uid && !providerHasCanonicalRoute[providerID] {
				previous := *merged
				*merged = channel
				channel = previous
				providerHasCanonicalRoute[providerID] = true
			}
			configs := make(map[string]APIKeyConfig, len(merged.APIKeyConfigs)+len(channel.APIKeyConfigs))
			for _, cfg := range merged.APIKeyConfigs {
				configs[cfg.Key] = cfg
			}
			for _, cfg := range channel.APIKeyConfigs {
				cfg.CredentialUID = GenerateCredentialUID(uid, cfg.Key)
				configs[cfg.Key] = cfg
			}
			merged.APIKeys = deduplicateStrings(append(merged.APIKeys, channel.APIKeys...))
			merged.APIKeyConfigs = make([]APIKeyConfig, 0, len(merged.APIKeys))
			for _, key := range merged.APIKeys {
				cfg := configs[key]
				cfg.Key = key
				cfg.CredentialUID = GenerateCredentialUID(uid, key)
				merged.APIKeyConfigs = append(merged.APIKeyConfigs, cfg)
			}
			incomingBaseURLs := append([]string(nil), channel.BaseURLs...)
			if channel.BaseURL != "" {
				incomingBaseURLs = append([]string{channel.BaseURL}, incomingBaseURLs...)
			}
			merged.BaseURLs = deduplicateBaseURLs(append(merged.BaseURLs, incomingBaseURLs...), merged.ServiceType)
			if merged.BaseURL != "" {
				merged.BaseURLs = deduplicateBaseURLs(append([]string{merged.BaseURL}, merged.BaseURLs...), merged.ServiceType)
			}
			if len(merged.BaseURLs) > 0 {
				merged.BaseURL = merged.BaseURLs[0]
			}
			if channel.Status == "active" {
				merged.Status = "active"
			}
			updated = true
		}
		return out
	}

	cm.config.Upstream = mergeKind(cm.config.Upstream, "messages")
	cm.config.ChatUpstream = mergeKind(cm.config.ChatUpstream, "chat")
	cm.config.ResponsesUpstream = mergeKind(cm.config.ResponsesUpstream, "responses")
	cm.config.GeminiUpstream = mergeKind(cm.config.GeminiUpstream, "gemini")
	cm.config.ImagesUpstream = mergeKind(cm.config.ImagesUpstream, "images")
	cm.config.VectorsUpstream = mergeKind(cm.config.VectorsUpstream, "vectors")
	if updated {
		accounts := cm.config.ManagedAccounts[:0]
		for _, account := range cm.config.ManagedAccounts {
			canonical := canonicalUID[account.ProviderID]
			if canonical != "" && account.AccountUID != canonical {
				continue
			}
			accounts = append(accounts, account)
		}
		cm.config.ManagedAccounts = accounts
		cm.config.syncManagedAccountsFromChannels()
		log.Printf("[Config-AccountMerge] 已按 provider 合并历史自动托管账号")
	}
	return updated
}

// UpdateAccountChannels 原子更新账号下所有协议渠道的 Key -> BaseURL 绑定。
func (cm *ConfigManager) UpdateAccountChannels(accountUID string, updates []AccountChannelUpdate) error {
	return cm.ApplyAccountChannelChanges(accountUID, updates, nil)
}

// ApplyAccountChannelChanges 在一次配置写入中更新现有渠道并新增缺失协议渠道。
func (cm *ConfigManager) ApplyAccountChannelChanges(accountUID string, updates []AccountChannelUpdate, additions []AccountChannelAddition) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	next := cm.config.deepCopy()
	if err := applyAccountChannelChanges(&next, accountUID, updates, additions); err != nil {
		return err
	}
	return cm.saveConfigLocked(next)
}

func applyAccountChannelChanges(cfg *Config, accountUID string, updates []AccountChannelUpdate, additions []AccountChannelAddition) error {
	if cfg == nil {
		return fmt.Errorf("配置为空")
	}
	accountUID = strings.TrimSpace(accountUID)
	if accountUID == "" {
		return fmt.Errorf("accountUID 不能为空")
	}
	byChannel := make(map[string]AccountChannelUpdate, len(updates))
	for _, update := range updates {
		if update.ChannelUID == "" {
			return fmt.Errorf("账号 %s 包含空 channelUID 更新", accountUID)
		}
		if _, exists := byChannel[update.ChannelUID]; exists {
			return fmt.Errorf("账号 %s 包含重复渠道更新: %s", accountUID, update.ChannelUID)
		}
		byChannel[update.ChannelUID] = update
	}
	known := 0
	total := 0
	providerID := ""
	providerMismatch := false
	countKnown := func(channels []UpstreamConfig) {
		for i := range channels {
			if channels[i].AccountUID == accountUID {
				total++
				if providerID == "" {
					providerID = channels[i].ProviderID
				} else if channels[i].ProviderID != providerID {
					providerMismatch = true
				}
				if _, ok := byChannel[channels[i].ChannelUID]; ok {
					known++
				}
			}
		}
	}
	countKnown(cfg.Upstream)
	countKnown(cfg.ChatUpstream)
	countKnown(cfg.ResponsesUpstream)
	countKnown(cfg.GeminiUpstream)
	countKnown(cfg.ImagesUpstream)
	countKnown(cfg.VectorsUpstream)
	if providerMismatch {
		return fmt.Errorf("账号 %s 包含不一致的 provider", accountUID)
	}
	for _, addition := range additions {
		additionProvider := strings.TrimSpace(addition.Upstream.ProviderID)
		if providerID == "" {
			providerID = additionProvider
		} else if additionProvider != providerID {
			return fmt.Errorf("账号 %s 的新增渠道 provider 不一致", accountUID)
		}
	}
	if total == 0 && len(additions) == 0 {
		return fmt.Errorf("账号 %s 不存在或没有可更新渠道", accountUID)
	}
	if total == 0 && len(updates) != 0 {
		return fmt.Errorf("账号 %s 不存在，不能应用渠道更新", accountUID)
	}
	if total > 0 && (known != total || len(updates) != total) {
		return fmt.Errorf("账号 %s 渠道更新不完整: matched=%d total=%d updates=%d", accountUID, known, total, len(updates))
	}

	matched := 0
	apply := func(channels []UpstreamConfig) {
		for i := range channels {
			channel := &channels[i]
			if channel.AccountUID != accountUID {
				continue
			}
			update, ok := byChannel[channel.ChannelUID]
			if !ok {
				continue
			}
			channel.Name = update.Name
			channel.APIKeys = deduplicateStrings(update.APIKeys)
			channel.APIKeyConfigs = normalizeAPIKeyConfigs(channel.APIKeys, update.APIKeyConfig)
			for j := range channel.APIKeyConfigs {
				if channel.APIKeyConfigs[j].CredentialUID == "" {
					channel.APIKeyConfigs[j].CredentialUID = GenerateCredentialUID(accountUID, channel.APIKeyConfigs[j].Key)
				}
			}
			channel.BaseURLs = deduplicateBaseURLs(update.BaseURLs, channel.ServiceType)
			if len(channel.BaseURLs) > 0 {
				channel.BaseURL = channel.BaseURLs[0]
			}
			if len(channel.APIKeys) > 0 && channel.Status == "suspended" {
				channel.Status = "active"
			}
			matched++
		}
	}
	apply(cfg.Upstream)
	apply(cfg.ChatUpstream)
	apply(cfg.ResponsesUpstream)
	apply(cfg.GeminiUpstream)
	apply(cfg.ImagesUpstream)
	apply(cfg.VectorsUpstream)

	if matched != known {
		return fmt.Errorf("账号 %s 渠道更新计数异常: matched=%d known=%d", accountUID, matched, known)
	}
	for i := range cfg.ManagedAccounts {
		if cfg.ManagedAccounts[i].AccountUID == accountUID && len(updates) > 0 {
			cfg.ManagedAccounts[i].Name = managedAccountName(updates[0].Name)
		}
	}
	for _, addition := range additions {
		if err := appendAccountChannelAddition(cfg, accountUID, addition); err != nil {
			return err
		}
	}
	return nil
}

func appendAccountChannelAddition(cfg *Config, accountUID string, addition AccountChannelAddition) error {
	channels, fallback, err := accountChannelSlice(cfg, addition.Kind)
	if err != nil {
		return err
	}
	upstream := *addition.Upstream.Clone()
	if upstream.AccountUID != accountUID || !upstream.AutoManaged || strings.TrimSpace(upstream.ProviderID) == "" {
		return fmt.Errorf("新增渠道必须属于账号 %s 的官方自动托管 provider", accountUID)
	}
	if strings.TrimSpace(upstream.ChannelUID) == "" || strings.TrimSpace(upstream.Name) == "" {
		return fmt.Errorf("新增 %s 渠道缺少 name 或 channelUID", addition.Kind)
	}
	if len(upstream.APIKeys) == 0 {
		return fmt.Errorf("新增 %s 渠道缺少 API Key", addition.Kind)
	}
	for _, existing := range *channels {
		if existing.Name == upstream.Name {
			return fmt.Errorf("渠道名称 '%s' 已存在", upstream.Name)
		}
	}
	if configHasChannelUID(cfg, upstream.ChannelUID) {
		return fmt.Errorf("channelUID %s 已存在", upstream.ChannelUID)
	}

	upstream.ServiceType = normalizeUpstreamServiceType(upstream.ServiceType, fallback)
	if addition.Kind == "images" {
		upstream.ServiceType, err = normalizeImagesServiceType(upstream.ServiceType)
	} else if addition.Kind == "vectors" {
		upstream.ServiceType, err = normalizeVectorsServiceType(upstream.ServiceType)
	}
	if err != nil {
		return err
	}
	upstream.AuthHeader, err = applyAuthHeader(upstream.AuthHeader)
	if err != nil {
		return err
	}
	if err := validateRequestTimeoutMs(upstream.RequestTimeoutMs); err != nil {
		return err
	}
	if err := validateResponseHeaderTimeoutMs(upstream.ResponseHeaderTimeoutMs); err != nil {
		return err
	}
	if upstream.RateLimitRPM < 0 || upstream.RateLimitBurst < 0 || upstream.RateLimitMaxConcurrent < 0 {
		return fmt.Errorf("限速参数不能为负数")
	}
	if err := validateStreamTimeouts(upstream.StreamFirstContentTimeoutMs, upstream.StreamInactivityTimeoutMs, upstream.StreamToolCallIdleTimeoutMs); err != nil {
		return err
	}
	if upstream.Status == "" {
		upstream.Status = "active"
	}
	upstream.APIKeys = deduplicateStrings(upstream.APIKeys)
	upstream.APIKeyConfigs = normalizeAPIKeyConfigs(upstream.APIKeys, upstream.APIKeyConfigs)
	for i := range upstream.APIKeyConfigs {
		if upstream.APIKeyConfigs[i].CredentialUID == "" {
			upstream.APIKeyConfigs[i].CredentialUID = GenerateCredentialUID(accountUID, upstream.APIKeyConfigs[i].Key)
		}
	}
	upstream.BaseURL = utils.CanonicalBaseURL(upstream.BaseURL, upstream.ServiceType)
	upstream.BaseURLs = deduplicateBaseURLs(upstream.BaseURLs, upstream.ServiceType)
	applyDefaultBaseURL(&upstream)
	*channels = append([]UpstreamConfig{upstream}, (*channels)...)
	return nil
}

func accountChannelSlice(cfg *Config, kind string) (*[]UpstreamConfig, string, error) {
	switch kind {
	case "messages":
		return &cfg.Upstream, "claude", nil
	case "chat":
		return &cfg.ChatUpstream, "openai", nil
	case "responses":
		return &cfg.ResponsesUpstream, "responses", nil
	case "gemini":
		return &cfg.GeminiUpstream, "gemini", nil
	case "images":
		return &cfg.ImagesUpstream, "openai", nil
	case "vectors":
		return &cfg.VectorsUpstream, "openai", nil
	default:
		return nil, "", fmt.Errorf("不支持的渠道类型: %s", kind)
	}
}

func configHasChannelUID(cfg *Config, channelUID string) bool {
	found := false
	visit := func(channels []UpstreamConfig) {
		for _, channel := range channels {
			if channel.ChannelUID == channelUID {
				found = true
				return
			}
		}
	}
	visit(cfg.Upstream)
	visit(cfg.ChatUpstream)
	visit(cfg.ResponsesUpstream)
	visit(cfg.GeminiUpstream)
	visit(cfg.ImagesUpstream)
	visit(cfg.VectorsUpstream)
	return found
}

// DeleteAccountChannels 原子删除账号下全部协议渠道，返回被删除的 channelUid。
func (cm *ConfigManager) DeleteAccountChannels(accountUID string) ([]string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var removed []string
	filter := func(channels []UpstreamConfig) []UpstreamConfig {
		kept := channels[:0]
		for _, channel := range channels {
			if channel.AccountUID == accountUID {
				removed = append(removed, channel.ChannelUID)
				continue
			}
			kept = append(kept, channel)
		}
		return kept
	}
	cm.config.Upstream = filter(cm.config.Upstream)
	cm.config.ChatUpstream = filter(cm.config.ChatUpstream)
	cm.config.ResponsesUpstream = filter(cm.config.ResponsesUpstream)
	cm.config.GeminiUpstream = filter(cm.config.GeminiUpstream)
	cm.config.ImagesUpstream = filter(cm.config.ImagesUpstream)
	cm.config.VectorsUpstream = filter(cm.config.VectorsUpstream)
	if len(removed) == 0 {
		return nil, fmt.Errorf("账号 %s 不存在", accountUID)
	}
	accounts := cm.config.ManagedAccounts[:0]
	for _, account := range cm.config.ManagedAccounts {
		if account.AccountUID != accountUID {
			accounts = append(accounts, account)
		}
	}
	cm.config.ManagedAccounts = accounts
	if err := cm.saveConfigLocked(cm.config); err != nil {
		return nil, err
	}
	return removed, nil
}

// RenameManagedAccount 原子重命名账号及其全部协议渠道。
func (cm *ConfigManager) RenameManagedAccount(accountUID, baseName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		return fmt.Errorf("账号名称不能为空")
	}
	total := 0
	count := func(channels []UpstreamConfig) {
		for i := range channels {
			if channels[i].AccountUID == accountUID {
				total++
			}
		}
	}
	count(cm.config.Upstream)
	count(cm.config.ChatUpstream)
	count(cm.config.ResponsesUpstream)
	count(cm.config.GeminiUpstream)
	count(cm.config.ImagesUpstream)
	count(cm.config.VectorsUpstream)
	matched := 0
	rename := func(kind string, channels []UpstreamConfig) {
		for i := range channels {
			if channels[i].AccountUID == accountUID {
				channels[i].Name = baseName
				if total > 1 {
					channels[i].Name += accountChannelSuffix(kind)
				}
				matched++
			}
		}
	}
	rename("messages", cm.config.Upstream)
	rename("chat", cm.config.ChatUpstream)
	rename("responses", cm.config.ResponsesUpstream)
	rename("gemini", cm.config.GeminiUpstream)
	rename("images", cm.config.ImagesUpstream)
	rename("vectors", cm.config.VectorsUpstream)
	if matched == 0 {
		return fmt.Errorf("账号 %s 不存在", accountUID)
	}
	for i := range cm.config.ManagedAccounts {
		if cm.config.ManagedAccounts[i].AccountUID == accountUID {
			cm.config.ManagedAccounts[i].Name = baseName
		}
	}
	return cm.saveConfigLocked(cm.config)
}

func accountChannelSuffix(kind string) string {
	switch kind {
	case "messages":
		return "-claude"
	case "chat":
		return "-chat"
	case "responses":
		return "-codex"
	case "gemini":
		return "-gemini"
	default:
		return "-" + kind
	}
}

func (c *Config) syncManagedAccountsFromChannels() {
	existingOrder := append([]ManagedAccountConfig(nil), c.ManagedAccounts...)
	existingCredentials := make(map[string]map[string]ManagedAccountCredential, len(c.ManagedAccounts))
	accounts := make(map[string]ManagedAccountConfig, len(c.ManagedAccounts))
	for _, account := range c.ManagedAccounts {
		byUID := make(map[string]ManagedAccountCredential, len(account.Credentials))
		for _, credential := range account.Credentials {
			byUID[credential.CredentialUID] = credential
		}
		existingCredentials[account.AccountUID] = byUID
		account.Credentials = nil
		accounts[account.AccountUID] = account
	}
	credentialSeen := make(map[string]map[string]bool, len(accounts))
	visit := func(channels []UpstreamConfig) {
		for i := range channels {
			channel := &channels[i]
			if !channel.AutoManaged || channel.AccountUID == "" || channel.ProviderID == "" {
				continue
			}
			account := accounts[channel.AccountUID]
			account.AccountUID = channel.AccountUID
			account.ProviderID = channel.ProviderID
			if account.Name == "" {
				account.Name = managedAccountName(channel.Name)
			}
			seen := credentialSeen[channel.AccountUID]
			if seen == nil {
				seen = make(map[string]bool, len(channel.APIKeys))
				credentialSeen[channel.AccountUID] = seen
			}
			for _, apiKey := range channel.APIKeys {
				uid := channel.CredentialUIDForKey(apiKey)
				if !seen[uid] {
					credential := existingCredentials[channel.AccountUID][uid]
					credential.CredentialUID = uid
					credential.APIKey = apiKey
					account.Credentials = append(account.Credentials, credential)
					seen[uid] = true
				}
			}
			accounts[channel.AccountUID] = account
		}
	}
	visit(c.Upstream)
	visit(c.ChatUpstream)
	visit(c.ResponsesUpstream)
	visit(c.GeminiUpstream)
	visit(c.ImagesUpstream)
	visit(c.VectorsUpstream)
	c.ManagedAccounts = c.ManagedAccounts[:0]
	seen := make(map[string]bool, len(accounts))
	for _, existing := range existingOrder {
		if account, ok := accounts[existing.AccountUID]; ok {
			c.ManagedAccounts = append(c.ManagedAccounts, account)
			seen[existing.AccountUID] = true
		}
	}
	for uid, account := range accounts {
		if !seen[uid] {
			c.ManagedAccounts = append(c.ManagedAccounts, account)
		}
	}
}

func (c *Config) hydrateManagedAccountCredentials() {
	credentials := make(map[string]map[string]string, len(c.ManagedAccounts))
	for _, account := range c.ManagedAccounts {
		byUID := make(map[string]string, len(account.Credentials))
		for _, credential := range account.Credentials {
			byUID[credential.CredentialUID] = credential.APIKey
		}
		credentials[account.AccountUID] = byUID
	}
	visit := func(channels []UpstreamConfig) {
		for i := range channels {
			channel := &channels[i]
			byUID := credentials[channel.AccountUID]
			if len(byUID) == 0 {
				continue
			}
			channel.APIKeys = channel.APIKeys[:0]
			for j := range channel.APIKeyConfigs {
				if apiKey := byUID[channel.APIKeyConfigs[j].CredentialUID]; apiKey != "" {
					channel.APIKeyConfigs[j].Key = apiKey
					channel.APIKeys = append(channel.APIKeys, apiKey)
				}
			}
		}
	}
	visit(c.Upstream)
	visit(c.ChatUpstream)
	visit(c.ResponsesUpstream)
	visit(c.GeminiUpstream)
	visit(c.ImagesUpstream)
	visit(c.VectorsUpstream)
}

func (c *Config) stripManagedChannelSecrets() {
	visit := func(channels []UpstreamConfig) {
		for i := range channels {
			channel := &channels[i]
			if !channel.AutoManaged || channel.AccountUID == "" || channel.ProviderID == "" {
				continue
			}
			channel.APIKeys = nil
			for j := range channel.APIKeyConfigs {
				channel.APIKeyConfigs[j].Key = ""
			}
		}
	}
	visit(c.Upstream)
	visit(c.ChatUpstream)
	visit(c.ResponsesUpstream)
	visit(c.GeminiUpstream)
	visit(c.ImagesUpstream)
	visit(c.VectorsUpstream)
}

func managedAccountName(channelName string) string {
	for _, suffix := range []string{"-claude", "-chat", "-codex", "-gemini"} {
		channelName = strings.TrimSuffix(channelName, suffix)
	}
	return channelName
}
