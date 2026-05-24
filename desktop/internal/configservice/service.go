package configservice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	PlatformClaude = "claude"
	PlatformCodex  = "codex"

	ProviderCCX         = "ccx"
	ProviderDeepSeek    = "deepseek"
	ProviderMiMo        = "mimo"
	ProviderKimi        = "kimi"
	ProviderGLM         = "glm"
	ProviderMiniMax     = "minimax"
	ProviderDashScope   = "dashscope"
	ProviderOpenCodeZen = "opencode-zen"
	ProviderOpenCodeGo  = "opencode-go"
	ProviderCustom      = "custom"
	ProviderOpenAI      = "openai"

	deepSeekClaudeBaseURL    = "https://api.deepseek.com/anthropic"
	defaultMiMoBaseURL       = "https://api.xiaomimimo.com/anthropic"
	kimiClaudeBaseURL        = "https://api.moonshot.cn/anthropic"
	glmClaudeBaseURL         = "https://open.bigmodel.cn/api/anthropic"
	miniMaxClaudeBaseURL     = "https://api.minimaxi.com/anthropic"
	dashScopeClaudeBaseURL   = "https://dashscope.aliyuncs.com/apps/anthropic"
	openCodeZenClaudeBaseURL = "https://opencode.ai/zen"
	openCodeGoClaudeBaseURL  = "https://opencode.ai/zen/go"
	stateVersion             = 1
)

type AgentConfigStatus struct {
	Platform           string `json:"platform"`
	Provider           string `json:"provider,omitempty"`
	TargetProvider     string `json:"targetProvider,omitempty"`
	Configured         bool   `json:"configured"`
	MatchesCurrentPort bool   `json:"matchesCurrentPort"`
	NeedsUpdate        bool   `json:"needsUpdate"`
	CurrentBaseURL     string `json:"currentBaseUrl"`
	TargetBaseURL      string `json:"targetBaseUrl"`
	ConfigPath         string `json:"configPath"`
	AuthPath           string `json:"authPath,omitempty"`
	HasState           bool   `json:"hasState"`
	LastError          string `json:"lastError,omitempty"`
}

type ApplyAgentConfigRequest struct {
	Platform string `json:"platform"`
	Provider string `json:"provider,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
}

type Service struct {
	homeDir  string
	stateDir string
}

type ClaudeProxyState struct {
	Version           int     `json:"version"`
	TargetPath        string  `json:"targetPath"`
	FileExisted       bool    `json:"fileExisted"`
	EnvExisted        bool    `json:"envExisted"`
	OriginalBaseURL   *string `json:"originalBaseUrl,omitempty"`
	OriginalAuthToken *string `json:"originalAuthToken,omitempty"`
	OriginalAPIKey    *string `json:"originalApiKey,omitempty"`
	InjectedProvider  string  `json:"injectedProvider"`
	InjectedBaseURL   string  `json:"injectedBaseUrl"`
	InjectedAuthToken string  `json:"injectedAuthToken,omitempty"`
	InjectedAPIKey    string  `json:"injectedApiKey,omitempty"`
}

type CodexProxyState struct {
	Version               int     `json:"version"`
	ConfigPath            string  `json:"configPath"`
	AuthPath              string  `json:"authPath"`
	ConfigFileExisted     bool    `json:"configFileExisted"`
	AuthFileExisted       bool    `json:"authFileExisted"`
	OriginalModelProvider *string `json:"originalModelProvider,omitempty"`
	OriginalProviderBlock *string `json:"originalProviderBlock,omitempty"`
	OriginalOpenAIAPIKey  *string `json:"originalOpenaiApiKey,omitempty"`
	InjectedBaseURL       string  `json:"injectedBaseUrl"`
	InjectedAPIKey        string  `json:"injectedApiKey"`
}

type ProviderKeyStore struct {
	Version int                         `json:"version"`
	Keys    map[string]string           `json:"keys"`
	Assets  map[string]ProviderKeyAsset `json:"assets,omitempty"`
}

type ProviderKeyAsset struct {
	Provider string   `json:"provider"`
	APIKey   string   `json:"apiKey"`
	BaseURL  string   `json:"baseUrl,omitempty"`
	PlanID   string   `json:"planId,omitempty"`
	Usages   []string `json:"usages,omitempty"`
}

func New(dataDir string) (*Service, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return nil, fmt.Errorf("无法定位用户主目录")
	}
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".config", "ccx-desktop")
	}
	stateDir := filepath.Join(dataDir, "agent-config-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	return &Service{homeDir: homeDir, stateDir: stateDir}, nil
}

func (s *Service) GetStatus(platform string, port int) (AgentConfigStatus, error) {
	switch platform {
	case PlatformClaude:
		return s.getClaudeStatus(port)
	case PlatformCodex:
		return s.getCodexStatus(port)
	default:
		return AgentConfigStatus{}, fmt.Errorf("不支持的 agent 平台: %s", platform)
	}
}

func (s *Service) Apply(req ApplyAgentConfigRequest, port int, accessKey string) error {
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		return fmt.Errorf("agent 平台不能为空")
	}
	switch platform {
	case PlatformClaude:
		return s.applyClaude(req, port, accessKey)
	case PlatformCodex:
		provider := strings.TrimSpace(req.Provider)
		if provider == ProviderOpenAI {
			return s.applyCodexOpenAI(req.APIKey)
		}
		if responsesURL, ok := codexResponsesBaseURL(provider); ok {
			return s.applyCodexThirdParty(provider, responsesURL, req.APIKey)
		}
		if port == 0 {
			return fmt.Errorf("CCX 端口未设置")
		}
		if accessKey == "" {
			return fmt.Errorf("PROXY_ACCESS_KEY 为空")
		}
		return s.applyCodex(port, accessKey)
	default:
		return fmt.Errorf("不支持的 agent 平台: %s", platform)
	}
}

func (s *Service) GetSavedProviderKeys() map[string]string {
	store := s.readProviderKeyStore()
	keys := map[string]string{}
	for name, key := range store.Keys {
		keys[name] = key
	}
	legacyCandidates := map[string]string{}
	assetKeys := make([]string, 0, len(store.Assets))
	for k := range store.Assets {
		assetKeys = append(assetKeys, k)
	}
	sort.Strings(assetKeys)
	for _, assetKey := range assetKeys {
		asset := store.Assets[assetKey]
		if strings.TrimSpace(asset.APIKey) == "" {
			continue
		}
		provider := asset.Provider
		planID := asset.PlanID
		keys["channel:"+provider] = asset.APIKey
		switch provider {
		case ProviderDeepSeek, ProviderMiMo:
			if planID != "" {
				keys[PlatformClaude+":"+provider+":"+planID] = asset.APIKey
			}
			if legacyCandidates[PlatformClaude+":"+provider] == "" {
				legacyCandidates[PlatformClaude+":"+provider] = asset.APIKey
			}
		case ProviderOpenAI:
			if planID != "" {
				keys[PlatformCodex+":"+provider+":"+planID] = asset.APIKey
			}
			if legacyCandidates[PlatformCodex+":"+provider] == "" {
				legacyCandidates[PlatformCodex+":"+provider] = asset.APIKey
			}
		}
	}
	for k, v := range legacyCandidates {
		if keys[k] == "" {
			keys[k] = v
		}
	}
	return keys
}

func (s *Service) GetProviderKeyAssets() []ProviderKeyAsset {
	store := s.readProviderKeyStore()
	assets := make([]ProviderKeyAsset, 0, len(store.Assets))
	for _, asset := range store.Assets {
		if asset.Provider == "" || asset.APIKey == "" {
			continue
		}
		assets = append(assets, asset)
	}
	return assets
}

func (s *Service) SaveProviderKeyAsset(asset ProviderKeyAsset) error {
	provider := strings.TrimSpace(asset.Provider)
	key := strings.TrimSpace(asset.APIKey)
	if provider == "" || key == "" {
		return nil
	}
	store := s.readProviderKeyStore()
	store.Version = stateVersion
	asset.Provider = provider
	asset.APIKey = key
	asset.BaseURL = strings.TrimSpace(asset.BaseURL)
	asset.PlanID = strings.TrimSpace(asset.PlanID)
	assetKey := provider
	if asset.PlanID != "" {
		assetKey = provider + ":" + asset.PlanID
	}
	existing := store.Assets[assetKey]
	if existing.Usages != nil {
		asset.Usages = appendUniqueMany(existing.Usages, asset.Usages)
	}
	store.Assets[assetKey] = asset
	store.Keys["channel:"+provider] = key
	switch provider {
	case ProviderDeepSeek, ProviderMiMo:
		store.Keys[PlatformClaude+":"+provider] = key
	case ProviderOpenAI:
		store.Keys[PlatformCodex+":"+provider] = key
	}
	return writeJSONAtomic(s.providerKeysPath(), store)
}

func (s *Service) Restore(platform string) error {
	switch platform {
	case PlatformClaude:
		return s.restoreClaude()
	case PlatformCodex:
		return s.restoreCodex()
	default:
		return fmt.Errorf("不支持的 agent 平台: %s", platform)
	}
}

func (s *Service) getClaudeStatus(port int) (AgentConfigStatus, error) {
	path := s.claudeSettingsPath()
	target := claudeBaseURL(port)
	status := AgentConfigStatus{
		Platform:       PlatformClaude,
		Provider:       ProviderCustom,
		TargetProvider: ProviderCCX,
		TargetBaseURL:  target,
		ConfigPath:     path,
		HasState:       fileExists(s.claudeStatePath()),
	}
	data, _, err := readJSONMap(path)
	if err != nil {
		status.LastError = err.Error()
		return status, nil
	}
	baseURL, _ := getNestedString(data, "env", "ANTHROPIC_BASE_URL")
	status.CurrentBaseURL = baseURL
	status.Provider = detectClaudeProvider(baseURL)
	status.MatchesCurrentPort = baseURL == target
	status.Configured = status.MatchesCurrentPort || status.Provider == ProviderDeepSeek || status.Provider == ProviderMiMo
	status.NeedsUpdate = baseURL != "" && isLocalBaseURL(baseURL) && !status.MatchesCurrentPort
	return status, nil
}

func (s *Service) getCodexStatus(port int) (AgentConfigStatus, error) {
	path := s.codexConfigPath()
	authPath := s.codexAuthPath()
	target := codexBaseURL(port)
	status := AgentConfigStatus{
		Platform:       PlatformCodex,
		Provider:       ProviderCustom,
		TargetProvider: ProviderCCX,
		TargetBaseURL:  target,
		ConfigPath:     path,
		AuthPath:       authPath,
		HasState:       fileExists(s.codexStatePath()),
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return status, nil
		}
		status.LastError = err.Error()
		return status, nil
	}
	text := string(content)
	baseURL, _ := extractTomlStringField(extractCodexProviderBlock(text), "base_url")
	modelProvider, _ := extractTopLevelTomlString(text, "model_provider")
	status.CurrentBaseURL = baseURL
	if modelProvider == ProviderCCX {
		status.Provider = ProviderCCX
	} else {
		status.Provider = ProviderOpenAI
		// OpenAI 官方模式，不需要目标 URL
		status.TargetBaseURL = ""
		status.TargetProvider = ProviderOpenAI
	}
	status.MatchesCurrentPort = modelProvider == ProviderCCX && baseURL == target
	status.Configured = status.MatchesCurrentPort || status.Provider == ProviderOpenAI
	status.NeedsUpdate = (modelProvider == ProviderCCX || isLocalBaseURL(baseURL)) && !status.MatchesCurrentPort
	return status, nil
}

func (s *Service) applyClaude(req ApplyAgentConfigRequest, port int, accessKey string) error {
	provider := normalizeClaudeProvider(req.Provider)
	baseURL, authToken, apiKey, err := resolveClaudeProvider(req, port, accessKey)
	if err != nil {
		return err
	}
	path := s.claudeSettingsPath()
	data, existed, err := readJSONMap(path)
	if err != nil {
		return err
	}
	env, envExisted := data["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
		data["env"] = env
	}
	originalBaseURL, baseOK := env["ANTHROPIC_BASE_URL"].(string)
	originalAuthToken, authOK := env["ANTHROPIC_AUTH_TOKEN"].(string)
	originalAPIKey, apiOK := env["ANTHROPIC_API_KEY"].(string)
	state := ClaudeProxyState{
		Version:           stateVersion,
		TargetPath:        path,
		FileExisted:       existed,
		EnvExisted:        envExisted,
		OriginalBaseURL:   optionalString(originalBaseURL, baseOK),
		OriginalAuthToken: optionalString(originalAuthToken, authOK),
		OriginalAPIKey:    optionalString(originalAPIKey, apiOK),
		InjectedProvider:  provider,
		InjectedBaseURL:   baseURL,
		InjectedAuthToken: authToken,
		InjectedAPIKey:    apiKey,
	}
	if existing, ok := s.readClaudeState(); ok {
		state = existing
		state.InjectedProvider = provider
		state.InjectedBaseURL = baseURL
		state.InjectedAuthToken = authToken
		state.InjectedAPIKey = apiKey
	}
	if provider != ProviderCCX && req.APIKey != "" {
		if err := s.saveProviderKey(PlatformClaude, provider, req.APIKey); err != nil {
			return err
		}
	}
	if err := writeJSONAtomic(s.claudeStatePath(), state); err != nil {
		return err
	}
	env["ANTHROPIC_BASE_URL"] = state.InjectedBaseURL
	if state.InjectedAuthToken != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = state.InjectedAuthToken
	} else {
		delete(env, "ANTHROPIC_AUTH_TOKEN")
	}
	if state.InjectedAPIKey != "" {
		env["ANTHROPIC_API_KEY"] = state.InjectedAPIKey
	} else {
		delete(env, "ANTHROPIC_API_KEY")
	}
	if err := writeJSONAtomic(path, data); err != nil {
		return err
	}
	if provider != ProviderCCX {
		return nil
	}
	if err := mergeJSONFile(s.claudeConfigPath(), map[string]any{"primaryApiKey": "ccx"}); err != nil {
		return err
	}
	return mergeJSONFile(s.claudeRootConfigPath(), map[string]any{"hasCompletedOnboarding": true})
}

func (s *Service) restoreClaude() error {
	var state ClaudeProxyState
	if err := readJSONFile(s.claudeStatePath(), &state); err != nil {
		return err
	}
	if !state.FileExisted {
		if err := os.Remove(state.TargetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Remove(s.claudeStatePath())
	}
	data, _, err := readJSONMap(state.TargetPath)
	if err != nil {
		return err
	}
	env, _ := data["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
		data["env"] = env
	}
	restoreStringField(env, "ANTHROPIC_BASE_URL", state.OriginalBaseURL)
	restoreStringField(env, "ANTHROPIC_AUTH_TOKEN", state.OriginalAuthToken)
	restoreStringField(env, "ANTHROPIC_API_KEY", state.OriginalAPIKey)
	if !state.EnvExisted && len(env) == 0 {
		delete(data, "env")
	}
	if err := writeJSONAtomic(state.TargetPath, data); err != nil {
		return err
	}
	return os.Remove(s.claudeStatePath())
}

func (s *Service) applyCodex(port int, accessKey string) error {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, configExisted, err := readTextFile(configPath)
	if err != nil {
		return err
	}
	authData, authExisted, err := readJSONMap(authPath)
	if err != nil {
		return err
	}
	modelProvider, mpOK := extractTopLevelTomlString(configContent, "model_provider")
	providerBlock, blockOK := extractNamedTomlBlock(configContent, "model_providers.ccx")
	apiKey, keyOK := authData["OPENAI_API_KEY"].(string)
	state := CodexProxyState{
		Version:               stateVersion,
		ConfigPath:            configPath,
		AuthPath:              authPath,
		ConfigFileExisted:     configExisted,
		AuthFileExisted:       authExisted,
		OriginalModelProvider: optionalString(modelProvider, mpOK),
		OriginalProviderBlock: optionalString(providerBlock, blockOK),
		OriginalOpenAIAPIKey:  optionalString(apiKey, keyOK),
		InjectedBaseURL:       codexBaseURL(port),
		InjectedAPIKey:        accessKey,
	}
	if existing, ok := s.readCodexState(); ok {
		state = existing
		state.InjectedBaseURL = codexBaseURL(port)
		state.InjectedAPIKey = accessKey
	}
	if err := writeJSONAtomic(s.codexStatePath(), state); err != nil {
		return err
	}
	updated := upsertTopLevelTomlString(configContent, "model_provider", "ccx")
	updated = upsertNamedTomlBlock(updated, "model_providers.ccx", codexProviderBlock(state.InjectedBaseURL))
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	authData["OPENAI_API_KEY"] = accessKey
	return writeJSONAtomic(authPath, authData)
}

func (s *Service) applyCodexOpenAI(apiKey string) error {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return err
	}
	key := strings.TrimSpace(apiKey)
	// 优先级 1: 用户输入的 key
	if key == "" {
		// 优先级 2: 之前保存的 OpenAI key
		key = s.GetSavedProviderKeys()["codex:"+ProviderOpenAI]
	}
	// 优先级 3: 从 codex state 中恢复原始的 OpenAI key
	if key == "" {
		if state, ok := s.readCodexState(); ok && state.OriginalOpenAIAPIKey != nil && *state.OriginalOpenAIAPIKey != "" {
			key = *state.OriginalOpenAIAPIKey
		}
	}
	// 优先级 4: auth.json 中现有的 key（可能是 CCX 的 accessKey）
	if key == "" {
		if existingKey, ok := authData["OPENAI_API_KEY"].(string); ok && strings.TrimSpace(existingKey) != "" {
			key = strings.TrimSpace(existingKey)
		}
	}
	if key == "" {
		return fmt.Errorf("OpenAI API Key 不能为空")
	}
	if err := s.saveProviderKey(PlatformCodex, ProviderOpenAI, key); err != nil {
		return err
	}
	updated := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	// OpenAI 是内置 provider，不需要显式配置块
	updated = restoreNamedTomlBlock(updated, "model_providers.openai", nil)
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	authData["OPENAI_API_KEY"] = key
	return writeJSONAtomic(authPath, authData)
}

func codexResponsesBaseURL(provider string) (string, bool) {
	switch provider {
	case ProviderDashScope:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1", true
	case ProviderOpenCodeZen:
		return "https://opencode.ai/zen/v1", true
	case ProviderOpenCodeGo:
		return "https://opencode.ai/zen/go/v1", true
	default:
		return "", false
	}
}

func (s *Service) applyCodexThirdParty(provider, baseURL, apiKey string) error {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return err
	}
	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = s.GetSavedProviderKeys()["codex:"+provider]
	}
	if key == "" {
		return fmt.Errorf("%s API Key 不能为空", provider)
	}
	if err := s.saveProviderKey(PlatformCodex, provider, key); err != nil {
		return err
	}
	block := fmt.Sprintf(`[model_providers.%s]
name = %q
base_url = %q
wire_api = "responses"
`, provider, provider, baseURL)
	updated := upsertTopLevelTomlString(configContent, "model_provider", provider)
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	updated = restoreNamedTomlBlock(updated, "model_providers.openai", nil)
	updated = upsertNamedTomlBlock(updated, "model_providers."+provider, block)
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	authData["OPENAI_API_KEY"] = key
	return writeJSONAtomic(authPath, authData)
}
func (s *Service) restoreCodex() error {
	var state CodexProxyState
	if err := readJSONFile(s.codexStatePath(), &state); err != nil {
		return err
	}
	if state.ConfigFileExisted {
		content, _, err := readTextFile(state.ConfigPath)
		if err != nil {
			return err
		}
		content = restoreTopLevelTomlString(content, "model_provider", state.OriginalModelProvider)
		content = restoreNamedTomlBlock(content, "model_providers.ccx", state.OriginalProviderBlock)
		if err := writeTextAtomic(state.ConfigPath, content); err != nil {
			return err
		}
	} else if err := os.Remove(state.ConfigPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if state.AuthFileExisted {
		authData, _, err := readJSONMap(state.AuthPath)
		if err != nil {
			return err
		}
		restoreStringField(authData, "OPENAI_API_KEY", state.OriginalOpenAIAPIKey)
		if err := writeJSONAtomic(state.AuthPath, authData); err != nil {
			return err
		}
	} else if err := os.Remove(state.AuthPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Remove(s.codexStatePath())
}

func (s *Service) claudeSettingsPath() string {
	return filepath.Join(s.homeDir, ".claude", "settings.json")
}

func (s *Service) claudeConfigPath() string {
	return filepath.Join(s.homeDir, ".claude", "config.json")
}

func (s *Service) claudeRootConfigPath() string {
	return filepath.Join(s.homeDir, ".claude.json")
}

func (s *Service) codexConfigPath() string {
	return filepath.Join(s.homeDir, ".codex", "config.toml")
}

func (s *Service) codexAuthPath() string {
	return filepath.Join(s.homeDir, ".codex", "auth.json")
}

func (s *Service) claudeStatePath() string {
	return filepath.Join(s.stateDir, "claude.json")
}

func (s *Service) codexStatePath() string {
	return filepath.Join(s.stateDir, "codex.json")
}

func (s *Service) providerKeysPath() string {
	return filepath.Join(s.stateDir, "provider-keys.json")
}

func (s *Service) readProviderKeyStore() ProviderKeyStore {
	store := ProviderKeyStore{Version: stateVersion, Keys: map[string]string{}, Assets: map[string]ProviderKeyAsset{}}
	_ = readJSONFile(s.providerKeysPath(), &store)
	if store.Keys == nil {
		store.Keys = map[string]string{}
	}
	if store.Assets == nil {
		store.Assets = map[string]ProviderKeyAsset{}
	}
	for name, key := range store.Keys {
		provider := providerFromStoreKey(name)
		if provider == "" || strings.TrimSpace(key) == "" {
			continue
		}
		asset := store.Assets[provider]
		asset.Provider = provider
		if asset.APIKey == "" {
			asset.APIKey = key
		}
		asset.Usages = appendUnique(asset.Usages, usageFromStoreKey(name))
		store.Assets[provider] = asset
	}
	return store
}

func (s *Service) saveProviderKey(platform string, provider string, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	store := s.readProviderKeyStore()
	store.Version = stateVersion
	store.Keys[platform+":"+provider] = key
	asset := store.Assets[provider]
	asset.Provider = provider
	asset.APIKey = key
	asset.Usages = appendUnique(asset.Usages, usageFromStoreKey(platform+":"+provider))
	store.Assets[provider] = asset
	return writeJSONAtomic(s.providerKeysPath(), store)
}

func providerFromStoreKey(name string) string {
	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(name)
	}
	return strings.TrimSpace(parts[1])
}

func usageFromStoreKey(name string) string {
	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 {
		return "manual"
	}
	switch parts[0] {
	case PlatformClaude:
		return "agent-direct"
	case PlatformCodex:
		return "codex-direct"
	default:
		return parts[0]
	}
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueMany(values []string, additions []string) []string {
	for _, value := range additions {
		values = appendUnique(values, value)
	}
	return values
}

func (s *Service) readClaudeState() (ClaudeProxyState, bool) {
	var state ClaudeProxyState
	if err := readJSONFile(s.claudeStatePath(), &state); err != nil {
		return ClaudeProxyState{}, false
	}
	return state, true
}

func (s *Service) readCodexState() (CodexProxyState, bool) {
	var state CodexProxyState
	if err := readJSONFile(s.codexStatePath(), &state); err != nil {
		return CodexProxyState{}, false
	}
	return state, true
}

func claudeBaseURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func normalizeClaudeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", ProviderCCX:
		return ProviderCCX
	case ProviderDeepSeek:
		return ProviderDeepSeek
	case ProviderMiMo:
		return ProviderMiMo
	case ProviderKimi:
		return ProviderKimi
	case ProviderGLM:
		return ProviderGLM
	case ProviderMiniMax:
		return ProviderMiniMax
	case ProviderDashScope:
		return ProviderDashScope
	case ProviderOpenCodeZen:
		return ProviderOpenCodeZen
	case ProviderOpenCodeGo:
		return ProviderOpenCodeGo
	default:
		return provider
	}
}

func resolveClaudeProvider(req ApplyAgentConfigRequest, port int, accessKey string) (string, string, string, error) {
	provider := normalizeClaudeProvider(req.Provider)
	switch provider {
	case ProviderCCX:
		if port == 0 {
			return "", "", "", fmt.Errorf("CCX 端口未设置")
		}
		if accessKey == "" {
			return "", "", "", fmt.Errorf("PROXY_ACCESS_KEY 为空")
		}
		return claudeBaseURL(port), accessKey, "", nil
	case ProviderDeepSeek:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("DeepSeek API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = deepSeekClaudeBaseURL
		}
		return baseURL, "", apiKey, nil
	case ProviderMiMo:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("MiMo API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = defaultMiMoBaseURL
		}
		return baseURL, "", apiKey, nil
	case ProviderKimi:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("Kimi API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = kimiClaudeBaseURL
		}
		return baseURL, "", apiKey, nil
	case ProviderGLM:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("GLM API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = glmClaudeBaseURL
		}
		return baseURL, "", apiKey, nil
	case ProviderMiniMax:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("MiniMax API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = miniMaxClaudeBaseURL
		}
		return baseURL, "", apiKey, nil
	case ProviderDashScope:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("DashScope API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = dashScopeClaudeBaseURL
		}
		return baseURL, "", apiKey, nil
	case ProviderOpenCodeZen:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("OpenCode Zen API Key 不能为空")
		}
		return openCodeZenClaudeBaseURL, "", apiKey, nil
	case ProviderOpenCodeGo:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("OpenCode Go API Key 不能为空")
		}
		return openCodeGoClaudeBaseURL, "", apiKey, nil
	default:
		return "", "", "", fmt.Errorf("不支持的 Claude Code provider: %s", provider)
	}
}

func detectClaudeProvider(baseURL string) string {
	value := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case value == "":
		return ""
	case isLocalBaseURL(value):
		return ProviderCCX
	case strings.Contains(value, "deepseek.com"):
		return ProviderDeepSeek
	case strings.Contains(value, "mimo.xiaomi.com") || strings.Contains(value, "xiaomimimo.com"):
		return ProviderMiMo
	case strings.Contains(value, "moonshot.cn"):
		return ProviderKimi
	case strings.Contains(value, "bigmodel.cn"):
		return ProviderGLM
	case strings.Contains(value, "minimaxi.com") || strings.Contains(value, "minimax.chat"):
		return ProviderMiniMax
	case strings.Contains(value, "dashscope.aliyuncs.com"):
		return ProviderDashScope
	case strings.Contains(value, "opencode.ai/zen/go"):
		return ProviderOpenCodeGo
	case strings.Contains(value, "opencode.ai/zen"):
		return ProviderOpenCodeZen
	default:
		return ProviderCustom
	}
}

func codexBaseURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/v1", port)
}

func codexProviderBlock(baseURL string) string {
	return fmt.Sprintf(`[model_providers.ccx]
name = "CCX Proxy"
base_url = %q
wire_api = "responses"
temp_env_key = "OPENAI_API_KEY"
requires_openai_auth = false
`, baseURL)
}

func readJSONMap(path string) (map[string]any, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, false, nil
		}
		return nil, false, err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return map[string]any{}, true, nil
	}
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, true, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return data, true, nil
}

func readJSONFile(path string, dest any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, dest)
}

func readTextFile(path string) (string, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(content), true, nil
}

func mergeJSONFile(path string, patch map[string]any) error {
	data, _, err := readJSONMap(path)
	if err != nil {
		return err
	}
	for key, value := range patch {
		data[key] = value
	}
	return writeJSONAtomic(path, data)
}

func writeJSONAtomic(path string, data any) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return writeBytesAtomic(path, content)
}

func writeTextAtomic(path string, content string) error {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return writeBytesAtomic(path, []byte(content))
}

func writeBytesAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func getNestedString(data map[string]any, keys ...string) (string, bool) {
	var current any = data
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = m[key]
		if !ok {
			return "", false
		}
	}
	value, ok := current.(string)
	return value, ok
}

func optionalString(value string, ok bool) *string {
	if !ok {
		return nil
	}
	return &value
}

func restoreStringField(data map[string]any, key string, value *string) {
	if value == nil {
		delete(data, key)
		return
	}
	data[key] = *value
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isLocalBaseURL(value string) bool {
	return strings.Contains(value, "127.0.0.1") || strings.Contains(value, "localhost")
}

func extractCodexProviderBlock(content string) string {
	block, _ := extractNamedTomlBlock(content, "model_providers.ccx")
	return block
}

// extractTopLevelTomlString 从 TOML 内容中提取顶层字符串值。
// 注意：仅适用于简单格式（key = "value"），不支持多行字符串、转义引号或 inline table。
// 当前仅用于 Codex config.toml 的 model_provider 字段，该字段始终为简单字符串。
func extractTopLevelTomlString(content string, key string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"\s*(?:#.*)?$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", false
	}
	return match[1], true
}

func extractTomlStringField(content string, key string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"\s*(?:#.*)?$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", false
	}
	return match[1], true
}

// findNamedTomlBlock 返回 [table] 块的起止偏移量。
// 注意：仅支持标准 [table] 格式，不支持带引号的 key（如 [providers."ccx"]）或 inline table。
// 当前仅用于 Codex config.toml 的 [model_providers.ccx] 块。
func findNamedTomlBlock(content string, table string) (int, int, bool) {
	header := "[" + table + "]"
	for lineStart := 0; lineStart < len(content); {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += lineStart
		}
		if strings.TrimSpace(content[lineStart:lineEnd]) == header {
			for nextLineStart := lineEnd + 1; nextLineStart < len(content); {
				nextLineEnd := strings.IndexByte(content[nextLineStart:], '\n')
				if nextLineEnd < 0 {
					nextLineEnd = len(content)
				} else {
					nextLineEnd += nextLineStart
				}
				nextLine := strings.TrimSpace(content[nextLineStart:nextLineEnd])
				if strings.HasPrefix(nextLine, "[") && strings.Contains(nextLine, "]") {
					return lineStart, nextLineStart, true
				}
				if nextLineEnd == len(content) {
					break
				}
				nextLineStart = nextLineEnd + 1
			}
			return lineStart, len(content), true
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
	}
	return 0, 0, false
}

func extractNamedTomlBlock(content string, table string) (string, bool) {
	start, end, ok := findNamedTomlBlock(content, table)
	if !ok {
		return "", false
	}
	return strings.TrimRight(content[start:end], "\n"), true
}

func upsertTopLevelTomlString(content string, key string, value string) string {
	line := fmt.Sprintf("%s = %q", key, value)
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=.*$`)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, line)
	}
	if strings.TrimSpace(content) == "" {
		return line + "\n"
	}
	return line + "\n" + content
}

func restoreTopLevelTomlString(content string, key string, original *string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=.*(?:\n|$)`)
	if original == nil {
		return re.ReplaceAllString(content, "")
	}
	line := fmt.Sprintf("%s = %q", key, *original)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, line+"\n")
	}
	return line + "\n" + content
}

func upsertNamedTomlBlock(content string, table string, block string) string {
	block = strings.TrimRight(block, "\n")
	if start, end, ok := findNamedTomlBlock(content, table); ok {
		return content[:start] + block + "\n" + content[end:]
	}
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return block + "\n"
	}
	return content + "\n\n" + block + "\n"
}

func restoreNamedTomlBlock(content string, table string, original *string) string {
	start, end, ok := findNamedTomlBlock(content, table)
	if original == nil {
		if !ok {
			return strings.TrimRight(content, "\n") + "\n"
		}
		return strings.TrimRight(content[:start]+content[end:], "\n") + "\n"
	}
	block := strings.TrimRight(*original, "\n") + "\n"
	if ok {
		return content[:start] + block + content[end:]
	}
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return block
	}
	return content + "\n\n" + block
}
