package configservice

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BenedictKing/ccx/desktop/internal/appdirs"
	_ "modernc.org/sqlite"
)

const (
	PlatformClaude   = "claude"
	PlatformCodex    = "codex"
	PlatformOpenCode = "opencode"

	ProviderCCX         = "ccx"
	ProviderDeepSeek    = "deepseek"
	ProviderMiMo        = "mimo"
	ProviderCompshare   = "compshare"
	ProviderRunAPI      = "runapi"
	ProviderKimi        = "kimi"
	ProviderGLM         = "glm"
	ProviderMiniMax     = "minimax"
	ProviderDashScope   = "dashscope"
	ProviderOpenCodeZen = "opencode-zen"
	ProviderOpenCodeGo  = "opencode-go"
	ProviderCustom      = "custom"
	ProviderOpenAI      = "openai"
	ProviderXFyun       = "xfyun"
	ProviderTencentLkeap = "tencent-lkeap"
	ProviderVolcArk      = "volc-ark"
	ProviderQianfan      = "qianfan"
	ProviderModelScope   = "modelscope"

	deepSeekClaudeBaseURL            = "https://api.deepseek.com/anthropic"
	defaultMiMoBaseURL               = "https://api.xiaomimimo.com/anthropic"
	compshareClaudeBaseURL           = "https://cp.compshare.cn"
	runAPIBaseURL                    = "https://runapi.co/v1"
	kimiClaudeBaseURL                = "https://api.moonshot.cn/anthropic"
	glmClaudeBaseURL                 = "https://open.bigmodel.cn/api/anthropic"
	miniMaxClaudeBaseURL             = "https://api.minimaxi.com/anthropic"
	dashScopeClaudeBaseURL           = "https://dashscope.aliyuncs.com/apps/anthropic"
	dashScopeCodingPlanClaudeBaseURL = "https://coding.dashscope.aliyuncs.com/apps/anthropic"
	openCodeZenClaudeBaseURL         = "https://opencode.ai/zen"
	openCodeGoClaudeBaseURL          = "https://opencode.ai/zen/go"
	xfyunClaudeBaseURL               = "https://maas-api.cn-huabei-1.xf-yun.com/anthropic"
	xfyunCodexBaseURL                = "https://maas-api.cn-huabei-1.xf-yun.com/v2"
	tencentLkeapClaudeBaseURL        = "https://api.lkeap.cloud.tencent.com/plan/anthropic"
	volcArkClaudeBaseURL             = "https://ark.cn-beijing.volces.com/api/coding"
	qianfanClaudeBaseURL             = "https://qianfan.baidubce.com/anthropic/coding"
	tencentLkeapCodexBaseURL         = "https://api.lkeap.cloud.tencent.com/plan/v3"
	volcArkCodexBaseURL              = "https://ark.cn-beijing.volces.com/api/coding/v3"
	qianfanCodexBaseURL              = "https://qianfan.baidubce.com/v2/coding"
	modelScopeCodexBaseURL           = "https://api-inference.modelscope.cn/v1"
	stateVersion                     = 1

	openCodeZenBaseURL = "https://opencode.ai/zen/v1"
	openCodeGoBaseURL  = "https://opencode.ai/zen/go/v1"
)

type AgentConfigStatus struct {
	Platform           string `json:"platform"`
	Provider           string `json:"provider,omitempty"`
	Mode               string `json:"mode,omitempty"` // "quick" | "plugin"
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

	// Codex 专属：config.toml 与 auth.json 一致性诊断。
	// 用于识别 CCS 等工具或手工编辑导致的配置污染，避免用户只看到上游 503。
	AuthMode          string `json:"authMode,omitempty"`          // auth.json 的 auth_mode："apikey" | "chatgpt"
	ConfigConsistent  bool   `json:"configConsistent"`            // config.toml 与 auth.json 语义是否一致
	DiagnosticCode    string `json:"diagnosticCode,omitempty"`    // 不一致时的诊断码，如 codex.missing_api_key
	DiagnosticMessage string `json:"diagnosticMessage,omitempty"` // 面向用户的诊断说明
}

type ApplyAgentConfigRequest struct {
	Platform string `json:"platform"`
	Provider string `json:"provider,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Mode     string `json:"mode,omitempty"` // "quick" | "plugin"
}

type MigrateCodexSessionsRequest struct {
	Provider string `json:"provider,omitempty"`
	Mode     string `json:"mode,omitempty"` // "quick" | "plugin"
}

type MigrateCodexSessionsResult struct {
	TargetProvider    string `json:"targetProvider"`
	TotalFiles        int    `json:"totalFiles"`
	MigratedFiles     int    `json:"migratedFiles"`
	SkippedFiles      int    `json:"skippedFiles"`
	FailedFiles       int    `json:"failedFiles"`
	SQLiteRowsUpdated int64  `json:"sqliteRowsUpdated"`
	SQLiteSkipped     bool   `json:"sqliteSkipped"`
	SQLiteError       string `json:"sqliteError,omitempty"`
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
	OriginalOpenAIBaseURL *string `json:"originalOpenaiBaseUrl,omitempty"`
	InjectedProvider      string  `json:"injectedProvider,omitempty"`
	InjectedBaseURL       string  `json:"injectedBaseUrl"`
	InjectedAPIKey        string  `json:"injectedApiKey"`
	ThirdPartyQuickMode   bool    `json:"thirdPartyQuickMode,omitempty"`
}

type DiffLine struct {
	Type    string `json:"type"` // "context" | "added" | "removed"
	Content string `json:"content"`
}

type FileDiff struct {
	Path   string     `json:"path"`
	Action string     `json:"action"` // "modify" | "create" | "delete"
	Lines  []DiffLine `json:"lines"`
}

type ConfigDiffResult struct {
	Files []FileDiff `json:"files"`
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
		dataDir = appdirs.DataDirForHome(homeDir)
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
	case PlatformOpenCode:
		return s.getOpenCodeStatus(port)
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
			if req.Mode == "quick" {
				return s.applyCodexThirdPartyQuick(provider, responsesURL, req.APIKey)
			}
			return s.applyCodexThirdParty(provider, responsesURL, req.APIKey)
		}
		if port == 0 {
			return fmt.Errorf("CCX 端口未设置")
		}
		if accessKey == "" {
			return fmt.Errorf("PROXY_ACCESS_KEY 为空")
		}
		return s.applyCodex(port, accessKey, req.Mode)
	case PlatformOpenCode:
		return s.applyOpenCode(req, port, accessKey)
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
		case ProviderDeepSeek, ProviderMiMo, ProviderCompshare, ProviderRunAPI, ProviderXFyun:
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
	case ProviderDeepSeek, ProviderMiMo, ProviderCompshare, ProviderRunAPI, ProviderXFyun:
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
	case PlatformOpenCode:
		return s.restoreOpenCode()
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
	status.Configured = status.MatchesCurrentPort || status.Provider == ProviderDeepSeek || status.Provider == ProviderMiMo || status.Provider == ProviderCompshare || status.Provider == ProviderRunAPI || status.Provider == ProviderXFyun
	status.NeedsUpdate = baseURL != "" && isLocalBaseURL(baseURL) && !status.MatchesCurrentPort
	return status, nil
}

func (s *Service) getCodexStatus(port int) (AgentConfigStatus, error) {
	path := s.codexConfigPath()
	authPath := s.codexAuthPath()
	target := codexBaseURL(port)
	status := AgentConfigStatus{
		Platform:         PlatformCodex,
		Provider:         ProviderCustom,
		TargetProvider:   ProviderCCX,
		TargetBaseURL:    target,
		ConfigPath:       path,
		AuthPath:         authPath,
		HasState:         fileExists(s.codexStatePath()),
		ConfigConsistent: true,
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
	ccxBlockBaseURL, _ := extractTomlStringField(extractCodexProviderBlock(text), "base_url")
	modelProvider, _ := extractTopLevelTomlString(text, "model_provider")
	openaiBaseURL, _ := extractTopLevelTomlString(text, "openai_base_url")

	// 兼容新旧两种 CCX proxy 格式
	isNewStyleCCX := strings.EqualFold(modelProvider, "openai") && isLocalBaseURL(openaiBaseURL)
	isOldStyleCCX := strings.EqualFold(modelProvider, ProviderCCX)

	// 检测插件模式
	isNativeCCX := false
	isLegacyQuickCCX := false
	ccxBearerToken := ""
	if isOldStyleCCX {
		ccxBlock, hasBlock := extractNamedTomlBlock(text, "model_providers.ccx")
		if hasBlock {
			requiresOpenAIAuth, hasRequiresOpenAIAuth := extractTomlBoolField(ccxBlock, "requires_openai_auth")
			if hasRequiresOpenAIAuth && requiresOpenAIAuth {
				isNativeCCX = true
			}
			if _, hasEnvKey := extractTomlStringField(ccxBlock, "env_key"); hasEnvKey || (hasRequiresOpenAIAuth && !requiresOpenAIAuth) {
				isLegacyQuickCCX = true
			}
			if bearerToken, hasBearerToken := extractTomlStringField(ccxBlock, "experimental_bearer_token"); hasBearerToken {
				ccxBearerToken = strings.TrimSpace(bearerToken)
			}
		}
	}

	// 检测第三方 provider 快捷模式：model_provider="openai" + 非本地 openai_base_url
	isThirdPartyQuickMode := false
	var thirdPartyProvider string
	if strings.EqualFold(modelProvider, "openai") && openaiBaseURL != "" && !isLocalBaseURL(openaiBaseURL) {
		if tp, ok := codexThirdPartyQuickBaseURL(openaiBaseURL); ok {
			isThirdPartyQuickMode = true
			thirdPartyProvider = tp
		}
	}

	// 旧格式优先取 [model_providers.ccx].base_url，新格式取 openai_base_url
	effectiveBaseURL := ccxBlockBaseURL
	if effectiveBaseURL == "" {
		effectiveBaseURL = openaiBaseURL
	}

	normalized := normalizeCodexProvider(modelProvider)
	if isNewStyleCCX || isNativeCCX || isOldStyleCCX {
		status.Provider = ProviderCCX
		if isNativeCCX || ccxBearerToken != "" {
			status.Mode = "plugin"
		} else if isOldStyleCCX {
			status.Mode = "quick"
		}
	} else if isThirdPartyQuickMode {
		status.Provider = thirdPartyProvider
		status.Mode = "quick"
		status.TargetProvider = thirdPartyProvider
	} else {
		status.Provider = normalized
	}
	if status.Provider != ProviderCCX && !isThirdPartyQuickMode {
		status.TargetProvider = status.Provider
	}
	if normalized == ProviderOpenAI && !isNewStyleCCX && !isThirdPartyQuickMode {
		status.TargetBaseURL = ""
	}
	if isThirdPartyQuickMode {
		status.TargetBaseURL = openaiBaseURL
	}
	status.CurrentBaseURL = effectiveBaseURL
	status.MatchesCurrentPort = (isNewStyleCCX || isOldStyleCCX) && effectiveBaseURL == target
	status.Configured = status.MatchesCurrentPort || (normalized == ProviderOpenAI && !isNewStyleCCX && !isThirdPartyQuickMode) || isCodexThirdPartyProvider(normalized) || isThirdPartyQuickMode
	status.NeedsUpdate = (isOldStyleCCX || isLocalBaseURL(effectiveBaseURL)) && !status.MatchesCurrentPort

	s.diagnoseCodexConfigAuth(&status, text, modelProvider, openaiBaseURL, isNewStyleCCX, isOldStyleCCX, isNativeCCX, isLegacyQuickCCX, ccxBearerToken)
	return status, nil
}

// diagnoseCodexConfigAuth 识别 config.toml 与 auth.json 的常见语义冲突。
// 这类冲突多由 CCS 等工具或手工编辑导致：config.toml 改成指向本地 CCX，
// 但 auth.json 的 auth_mode / OPENAI_API_KEY 没有同步，运行时表现为上游 503/鉴权失败。
// 仅针对“当前意图指向本地 CCX”的配置做判定，避免误报第三方直连。
func (s *Service) diagnoseCodexConfigAuth(
	status *AgentConfigStatus,
	configText string,
	modelProvider string,
	openaiBaseURL string,
	isNewStyleCCX bool,
	isOldStyleCCX bool,
	isNativeCCX bool,
	isLegacyQuickCCX bool,
	ccxBearerToken string,
) {
	// 默认视为一致，无法读取 auth.json 时不武断报错。
	status.ConfigConsistent = true

	authData, authExisted, err := readJSONMap(status.AuthPath)
	if err != nil {
		// auth.json 存在但解析失败本身就是污染信号。
		status.ConfigConsistent = false
		status.DiagnosticCode = "codex.auth_unreadable"
		status.DiagnosticMessage = "auth.json 解析失败，可能已损坏；建议重新应用 Codex -> CCX 配置"
		return
	}

	authMode, _ := authData["auth_mode"].(string)
	status.AuthMode = authMode
	apiKey, _ := authData["OPENAI_API_KEY"].(string)
	apiKey = strings.TrimSpace(apiKey)
	expectedProxyKey := strings.TrimSpace(s.readCurrentProxyAccessKey())

	// 仅诊断“指向本地 CCX”的两类配置；第三方/OpenAI 直连不在此判定范围。
	pointsToLocalCCX := isNewStyleCCX || isOldStyleCCX

	switch {
	case isNativeCCX || ccxBearerToken != "":
		// 插件模式：依赖 ChatGPT OAuth + experimental_bearer_token。
		if ccxBearerToken == "" {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.plugin_missing_bearer"
			status.DiagnosticMessage = "插件模式缺少 experimental_bearer_token；建议重新应用 Codex -> CCX 插件模式"
			return
		}
		if expectedProxyKey != "" && ccxBearerToken != expectedProxyKey {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.proxy_key_mismatch"
			status.DiagnosticMessage = "config.toml 中的 experimental_bearer_token 与当前 CCX 代理密钥不一致；建议重新应用 Codex -> CCX 配置"
			return
		}
		if !strings.EqualFold(authMode, "chatgpt") {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.auth_mode_mismatch"
			status.DiagnosticMessage = "插件模式要求 auth.json 的 auth_mode 为 chatgpt，但当前为 " + authMode + "；建议重新应用 Codex -> CCX 配置"
			return
		}
	case isLegacyQuickCCX:
		// 旧式 quick 配置：model_provider="ccx" + [model_providers.ccx]（env_key/requires_openai_auth=false），
		// 仍由 OPENAI_API_KEY 驱动，可正常工作；缺 key 或 auth_mode 缺失/错误时才需提示。
		if !authExisted || apiKey == "" {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.missing_api_key"
			status.DiagnosticMessage = "config.toml 指向本地 CCX，但 auth.json 缺少 OPENAI_API_KEY；建议重新应用 Codex -> CCX 配置"
			return
		}
		if expectedProxyKey != "" && apiKey != expectedProxyKey {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.proxy_key_mismatch"
			status.DiagnosticMessage = "auth.json 中的 OPENAI_API_KEY 与当前 CCX 代理密钥不一致；建议重新应用 Codex -> CCX 配置"
			return
		}
		if !strings.EqualFold(authMode, "apikey") {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.auth_mode_mismatch"
			status.DiagnosticMessage = "快捷模式要求 auth.json 的 auth_mode 为 apikey，但当前为 " + authMode + "；建议重新应用 Codex -> CCX 配置"
			return
		}
	case isNewStyleCCX:
		// 快捷模式：openai_base_url 指向本地 CCX，需要 OPENAI_API_KEY + auth_mode=apikey。
		if !authExisted || apiKey == "" {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.missing_api_key"
			status.DiagnosticMessage = "config.toml 指向本地 CCX，但 auth.json 缺少 OPENAI_API_KEY；建议重新应用 Codex -> CCX 配置"
			return
		}
		if expectedProxyKey != "" && apiKey != expectedProxyKey {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.proxy_key_mismatch"
			status.DiagnosticMessage = "auth.json 中的 OPENAI_API_KEY 与当前 CCX 代理密钥不一致；建议重新应用 Codex -> CCX 配置"
			return
		}
		if !strings.EqualFold(authMode, "apikey") {
			status.ConfigConsistent = false
			status.DiagnosticCode = "codex.auth_mode_mismatch"
			status.DiagnosticMessage = "快捷模式要求 auth.json 的 auth_mode 为 apikey，但当前为 " + authMode + "；建议重新应用 Codex -> CCX 配置"
			return
		}
	case isOldStyleCCX:
		// 旧格式 model_provider="ccx"，但既不是插件块也不是旧式 quick 块，说明配置残缺。
		status.ConfigConsistent = false
		status.DiagnosticCode = "codex.legacy_incomplete"
		status.DiagnosticMessage = "config.toml 使用旧式 ccx provider 但缺少必要字段；建议重新应用 Codex -> CCX 配置"
		return
	}

	// model_provider 指向本地但格式无法识别为任何已知 CCX 形态：典型的 CCS 污染。
	if !pointsToLocalCCX && isLocalBaseURL(openaiBaseURL) && !strings.EqualFold(modelProvider, "openai") {
		status.ConfigConsistent = false
		status.DiagnosticCode = "codex.config_polluted"
		status.DiagnosticMessage = "config.toml 指向本地端口但 model_provider 配置异常；建议先恢复再重新应用 Codex -> CCX 配置"
	}
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

func (s *Service) applyCodex(port int, accessKey string, mode string) error {
	if mode == "plugin" {
		return s.applyCodexPlugin(port, accessKey)
	}
	return s.applyCodexQuick(port, accessKey)
}

func (s *Service) applyCodexQuick(port int, accessKey string) error {
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
	openaiBaseURL, obOK := extractTopLevelTomlString(configContent, "openai_base_url")
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
		OriginalOpenAIBaseURL: optionalString(openaiBaseURL, obOK),
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
	// config.toml: model_provider = "openai" + openai_base_url
	updated := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updated = upsertTopLevelTomlString(updated, "openai_base_url", codexBaseURL(port))
	// 清理插件模式残留
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	// auth.json: OPENAI_API_KEY = accessKey, auth_mode = "apikey"
	authData["OPENAI_API_KEY"] = accessKey
	authData["auth_mode"] = "apikey"
	return writeJSONAtomic(authPath, authData)
}

func (s *Service) applyCodexPlugin(port int, accessKey string) error {
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
	openaiBaseURL, obOK := extractTopLevelTomlString(configContent, "openai_base_url")
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
		OriginalOpenAIBaseURL: optionalString(openaiBaseURL, obOK),
		InjectedProvider:      ProviderCCX,
		InjectedBaseURL:       codexBaseURL(port),
		InjectedAPIKey:        accessKey,
	}
	if existing, ok := s.readCodexState(); ok {
		state = existing
		state.InjectedProvider = ProviderCCX
		state.InjectedBaseURL = codexBaseURL(port)
		state.InjectedAPIKey = accessKey
	}
	if err := writeJSONAtomic(s.codexStatePath(), state); err != nil {
		return err
	}
	// config.toml: model_provider = "ccx" + [model_providers.ccx] 块 + experimental_bearer_token
	block := fmt.Sprintf(`[model_providers.ccx]
name = "CCX Proxy"
base_url = %q
wire_api = "responses"
requires_openai_auth = true
experimental_bearer_token = %q
`, codexBaseURL(port), accessKey)
	updated := upsertTopLevelTomlString(configContent, "model_provider", "ccx")
	updated = restoreTopLevelTomlString(updated, "openai_base_url", nil)
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	updated = upsertNamedTomlBlock(updated, "model_providers.ccx", block)
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	// auth.json: OPENAI_API_KEY = accessKey, auth_mode = "chatgpt"（插件模式依赖 ChatGPT OAuth）
	authData["OPENAI_API_KEY"] = accessKey
	authData["auth_mode"] = "chatgpt"
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
	updated := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updated = restoreTopLevelTomlString(updated, "openai_base_url", nil) // 清理 CCX proxy 残留
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	// OpenAI 是内置 provider，不需要显式配置块
	updated = restoreNamedTomlBlock(updated, "model_providers.openai", nil)
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	key := strings.TrimSpace(apiKey)
	if key != "" {
		// API Key 模式：写入 key + auth_mode = "apikey"
		// OpenAI 直连的 key 直接落在 auth.json，不再单独保存 provider key
		authData["OPENAI_API_KEY"] = key
		authData["auth_mode"] = "apikey"
	} else {
		// OAuth 登录模式：auth_mode = "chatgpt"，OPENAI_API_KEY = null
		authData["auth_mode"] = "chatgpt"
		authData["OPENAI_API_KEY"] = nil
	}
	return writeJSONAtomic(authPath, authData)
}

func codexResponsesBaseURL(provider string) (string, bool) {
	switch provider {
	case ProviderDeepSeek:
		return "https://api.deepseek.com/v1", true
	case ProviderMiMo:
		return "https://api.xiaomimimo.com/v1", true
	case ProviderCompshare:
		return "https://cp.compshare.cn/v1", true
	case ProviderKimi:
		return "https://api.moonshot.cn/v1", true
	case ProviderGLM:
		return "https://open.bigmodel.cn/api/paas/v4", true
	case ProviderMiniMax:
		return "https://api.minimax.chat/v1", true
	case ProviderDashScope:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1", true
	case ProviderRunAPI:
		return runAPIBaseURL, true
	case ProviderOpenCodeZen:
		return "https://opencode.ai/zen/v1", true
	case ProviderOpenCodeGo:
		return "https://opencode.ai/zen/go/v1", true
	case ProviderXFyun:
		return xfyunCodexBaseURL, true
	case ProviderTencentLkeap:
		return tencentLkeapCodexBaseURL, true
	case ProviderVolcArk:
		return volcArkCodexBaseURL, true
	case ProviderQianfan:
		return qianfanCodexBaseURL, true
	case ProviderModelScope:
		return modelScopeCodexBaseURL, true
	default:
		return "", false
	}
}

func (s *Service) applyCodexThirdParty(provider, baseURL, apiKey string) error {
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
	openaiKey, keyOK := authData["OPENAI_API_KEY"].(string)
	state := CodexProxyState{
		Version:               stateVersion,
		ConfigPath:            configPath,
		AuthPath:              authPath,
		ConfigFileExisted:     configExisted,
		AuthFileExisted:       authExisted,
		OriginalModelProvider: optionalString(modelProvider, mpOK),
		OriginalProviderBlock: optionalString(providerBlock, blockOK),
		OriginalOpenAIAPIKey:  optionalString(openaiKey, keyOK),
		InjectedProvider:      provider,
		InjectedBaseURL:       baseURL,
	}
	if existing, ok := s.readCodexState(); ok {
		state = existing
		state.InjectedProvider = provider
		state.InjectedBaseURL = baseURL
	}
	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = s.GetSavedProviderKeys()["codex:"+provider]
	}
	if key == "" {
		return fmt.Errorf("%s API Key 不能为空", provider)
	}
	state.InjectedAPIKey = key
	if err := s.saveProviderKey(PlatformCodex, provider, key); err != nil {
		return err
	}
	if err := writeJSONAtomic(s.codexStatePath(), state); err != nil {
		return err
	}
	block := fmt.Sprintf(`[model_providers.%s]
name = %q
base_url = %q
wire_api = "responses"
requires_openai_auth = true
experimental_bearer_token = %q
`, provider, provider, baseURL, key)
	updated := upsertTopLevelTomlString(configContent, "model_provider", provider)
	updated = restoreTopLevelTomlString(updated, "openai_base_url", nil) // 清理 CCX proxy 残留
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	updated = restoreNamedTomlBlock(updated, "model_providers.openai", nil)
	updated = upsertNamedTomlBlock(updated, "model_providers."+provider, block)
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	authData["OPENAI_API_KEY"] = key
	authData["auth_mode"] = "chatgpt"
	return writeJSONAtomic(authPath, authData)
}

// codexThirdPartyQuickBaseURL 通过 openai_base_url 反查已知第三方 provider。
func codexThirdPartyQuickBaseURL(baseURL string) (string, bool) {
	switch {
	case strings.Contains(baseURL, "deepseek.com"):
		return ProviderDeepSeek, true
	case strings.Contains(baseURL, "xiaomimimo.com"):
		return ProviderMiMo, true
	case strings.Contains(baseURL, "cp.compshare.cn"):
		return ProviderCompshare, true
	case strings.Contains(baseURL, "moonshot.cn"):
		return ProviderKimi, true
	case strings.Contains(baseURL, "bigmodel.cn"):
		return ProviderGLM, true
	case strings.Contains(baseURL, "minimax.chat") || strings.Contains(baseURL, "minimaxi.com"):
		return ProviderMiniMax, true
	case strings.Contains(baseURL, "dashscope.aliyuncs.com"):
		return ProviderDashScope, true
	case strings.Contains(baseURL, "runapi.co"):
		return ProviderRunAPI, true
	case strings.Contains(baseURL, "opencode.ai/zen/go"):
		return ProviderOpenCodeGo, true
	case strings.Contains(baseURL, "opencode.ai/zen"):
		return ProviderOpenCodeZen, true
	case strings.Contains(baseURL, "xf-yun.com"):
		return ProviderXFyun, true
	case strings.Contains(baseURL, "lkeap.cloud.tencent.com"):
		return ProviderTencentLkeap, true
	case strings.Contains(baseURL, "volces.com"):
		return ProviderVolcArk, true
	case strings.Contains(baseURL, "baidubce.com"):
		return ProviderQianfan, true
	case strings.Contains(baseURL, "modelscope.cn"):
		return ProviderModelScope, true
	default:
		return "", false
	}
}

// applyCodexThirdPartyQuick 以快捷模式配置第三方 provider。
// 使用 model_provider="openai" + openai_base_url=<第三方 URL>。
func (s *Service) applyCodexThirdPartyQuick(provider, baseURL, apiKey string) error {
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
	openaiBaseURL, obOK := extractTopLevelTomlString(configContent, "openai_base_url")
	openaiKey, keyOK := authData["OPENAI_API_KEY"].(string)
	state := CodexProxyState{
		Version:               stateVersion,
		ConfigPath:            configPath,
		AuthPath:              authPath,
		ConfigFileExisted:     configExisted,
		AuthFileExisted:       authExisted,
		OriginalModelProvider: optionalString(modelProvider, mpOK),
		OriginalProviderBlock: optionalString(providerBlock, blockOK),
		OriginalOpenAIAPIKey:  optionalString(openaiKey, keyOK),
		OriginalOpenAIBaseURL: optionalString(openaiBaseURL, obOK),
		ThirdPartyQuickMode:   true,
		InjectedProvider:      provider,
		InjectedBaseURL:       baseURL,
	}
	if existing, ok := s.readCodexState(); ok {
		state = existing
		state.ThirdPartyQuickMode = true
		state.InjectedProvider = provider
		state.InjectedBaseURL = baseURL
	}
	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = s.GetSavedProviderKeys()["codex:"+provider]
	}
	if key == "" {
		return fmt.Errorf("%s API Key 不能为空", provider)
	}
	state.InjectedAPIKey = key
	if err := s.saveProviderKey(PlatformCodex, provider, key); err != nil {
		return err
	}
	if err := writeJSONAtomic(s.codexStatePath(), state); err != nil {
		return err
	}
	// config.toml: model_provider="openai" + openai_base_url=<第三方 URL>
	updated := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updated = upsertTopLevelTomlString(updated, "openai_base_url", baseURL)
	updated = restoreNamedTomlBlock(updated, "model_providers.ccx", nil)
	// 清理插件模式残留的第三方 provider 块
	if isCodexThirdPartyProvider(provider) {
		updated = restoreNamedTomlBlock(updated, "model_providers."+provider, nil)
	}
	if err := writeTextAtomic(configPath, updated); err != nil {
		return err
	}
	authData["OPENAI_API_KEY"] = key
	authData["auth_mode"] = "apikey"
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
		content = restoreTopLevelTomlString(content, "openai_base_url", state.OriginalOpenAIBaseURL)
		content = restoreNamedTomlBlock(content, "model_providers.ccx", state.OriginalProviderBlock)
		if state.InjectedProvider != "" && state.InjectedProvider != ProviderCCX && state.InjectedProvider != ProviderOpenAI {
			content = restoreNamedTomlBlock(content, "model_providers."+state.InjectedProvider, nil)
		}
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

func (s *Service) MigrateCodexSessions(req MigrateCodexSessionsRequest) (MigrateCodexSessionsResult, error) {
	targetProvider, err := resolveCodexSessionModelProvider(req)
	result := MigrateCodexSessionsResult{TargetProvider: targetProvider}
	if err != nil {
		return result, err
	}
	for _, dir := range []string{s.codexSessionsDir(), s.codexArchivedSessionsDir()} {
		if err := s.migrateCodexSessionDir(dir, targetProvider, &result); err != nil {
			return result, err
		}
	}
	s.migrateCodexStateDB(targetProvider, &result)
	return result, nil
}

func resolveCodexSessionModelProvider(req MigrateCodexSessionsRequest) (string, error) {
	provider := normalizeCodexProvider(req.Provider)
	mode := strings.TrimSpace(req.Mode)
	if mode != "plugin" {
		mode = "quick"
	}
	switch provider {
	case ProviderOpenAI:
		return ProviderOpenAI, nil
	case ProviderCCX:
		if mode == "plugin" {
			return ProviderCCX, nil
		}
		return ProviderOpenAI, nil
	case ProviderDashScope, ProviderRunAPI, ProviderOpenCodeZen, ProviderOpenCodeGo, ProviderXFyun:
		if mode == "plugin" {
			return provider, nil
		}
		return ProviderOpenAI, nil
	default:
		return "", fmt.Errorf("不支持的 Codex provider: %s", req.Provider)
	}
}

func (s *Service) migrateCodexSessionDir(dir, targetProvider string, result *MigrateCodexSessionsResult) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			result.FailedFiles++
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".jsonl") {
			return nil
		}
		result.TotalFiles++
		migrated, err := migrateCodexSessionFile(path, targetProvider)
		if err != nil {
			result.FailedFiles++
			return nil
		}
		if migrated {
			result.MigratedFiles++
		} else {
			result.SkippedFiles++
		}
		return nil
	})
}

func migrateCodexSessionFile(path, targetProvider string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	lineEnd := bytes.IndexByte(content, '\n')
	firstLine := content
	rest := []byte(nil)
	if lineEnd >= 0 {
		firstLine = content[:lineEnd]
		rest = content[lineEnd:]
	}
	lineSuffix := []byte(nil)
	if bytes.HasSuffix(firstLine, []byte("\r")) {
		firstLine = firstLine[:len(firstLine)-1]
		lineSuffix = []byte("\r")
	}
	var meta map[string]any
	if err := json.Unmarshal(firstLine, &meta); err != nil {
		return false, nil
	}
	if typ, ok := meta["type"].(string); !ok || typ != "session_meta" {
		return false, nil
	}
	payload, ok := meta["payload"].(map[string]any)
	if !ok {
		return false, nil
	}
	currentProvider, ok := payload["model_provider"].(string)
	if !ok || currentProvider == targetProvider {
		return false, nil
	}
	payload["model_provider"] = targetProvider
	updatedFirstLine, err := json.Marshal(meta)
	if err != nil {
		return false, err
	}
	updated := make([]byte, 0, len(updatedFirstLine)+len(lineSuffix)+len(rest))
	updated = append(updated, updatedFirstLine...)
	updated = append(updated, lineSuffix...)
	updated = append(updated, rest...)
	mode := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return true, writeBytesAtomicWithMode(path, updated, mode)
}

func (s *Service) migrateCodexStateDB(targetProvider string, result *MigrateCodexSessionsResult) {
	path := s.codexStateDBPath()
	if _, err := os.Stat(path); err != nil {
		result.SQLiteSkipped = true
		if !os.IsNotExist(err) {
			result.SQLiteError = err.Error()
		}
		return
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		result.SQLiteSkipped = true
		result.SQLiteError = err.Error()
		return
	}
	defer db.Close()
	_, _ = db.Exec("PRAGMA busy_timeout = 500")
	updateResult, err := db.Exec(`UPDATE threads SET model_provider = ? WHERE model_provider IS NULL OR model_provider <> ?`, targetProvider, targetProvider)
	if err != nil {
		result.SQLiteSkipped = true
		result.SQLiteError = err.Error()
		return
	}
	rows, err := updateResult.RowsAffected()
	if err == nil {
		result.SQLiteRowsUpdated = rows
	}
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

func (s *Service) codexSessionsDir() string {
	return filepath.Join(s.homeDir, ".codex", "sessions")
}

func (s *Service) codexArchivedSessionsDir() string {
	return filepath.Join(s.homeDir, ".codex", "archived_sessions")
}

func (s *Service) codexStateDBPath() string {
	return filepath.Join(s.homeDir, ".codex", "state_5.sqlite")
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

func (s *Service) currentDataDir() string {
	return filepath.Dir(s.stateDir)
}

func (s *Service) readCurrentProxyAccessKey() string {
	if key := readEnvValueFromFile(filepath.Join(s.currentDataDir(), ".env"), "PROXY_ACCESS_KEY"); strings.TrimSpace(key) != "" {
		return key
	}
	if key := strings.TrimSpace(os.Getenv("PROXY_ACCESS_KEY")); key != "" {
		return key
	}
	return ""
}

type OpenCodeProxyState struct {
	Version              int     `json:"version"`
	ProviderID           string  `json:"providerId"`
	ConfigPath           string  `json:"configPath"`
	AuthPath             string  `json:"authPath"`
	ConfigFileExisted    bool    `json:"configFileExisted"`
	AuthFileExisted      bool    `json:"authFileExisted"`
	OriginalProviderJSON *string `json:"originalProviderJson,omitempty"`
	OriginalAuthType     *string `json:"originalAuthType,omitempty"`
	OriginalAuthKey      *string `json:"originalAuthKey,omitempty"`
	InjectedBaseURL      string  `json:"injectedBaseUrl"`
	InjectedAPIKey       string  `json:"injectedApiKey"`
}

func (s *Service) openCodeConfigPath() string {
	return filepath.Join(s.homeDir, ".config", "opencode", "opencode.jsonc")
}

func (s *Service) openCodeAuthPath() string {
	if dir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); dir != "" {
		return filepath.Join(dir, "opencode", "auth.json")
	}
	return filepath.Join(s.homeDir, ".local", "share", "opencode", "auth.json")
}

func (s *Service) openCodeStatePath() string {
	return filepath.Join(s.stateDir, "opencode.json")
}

func (s *Service) readOpenCodeState() (OpenCodeProxyState, bool) {
	var state OpenCodeProxyState
	if err := readJSONFile(s.openCodeStatePath(), &state); err != nil {
		return OpenCodeProxyState{}, false
	}
	return state, true
}

func (s *Service) writeOpenCodeState(state OpenCodeProxyState) error {
	return writeJSONAtomic(s.openCodeStatePath(), state)
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
	case ProviderCompshare:
		return ProviderCompshare
	case ProviderRunAPI:
		return ProviderRunAPI
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
	case ProviderXFyun:
		return ProviderXFyun
	case ProviderTencentLkeap:
		return ProviderTencentLkeap
	case ProviderVolcArk:
		return ProviderVolcArk
	case ProviderQianfan:
		return ProviderQianfan
	default:
		return provider
	}
}

func normalizeCodexProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", ProviderOpenAI:
		return ProviderOpenAI
	case ProviderCCX:
		return ProviderCCX
	case ProviderDeepSeek:
		return ProviderDeepSeek
	case ProviderMiMo:
		return ProviderMiMo
	case ProviderCompshare:
		return ProviderCompshare
	case ProviderRunAPI:
		return ProviderRunAPI
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
	case ProviderXFyun:
		return ProviderXFyun
	case ProviderTencentLkeap:
		return ProviderTencentLkeap
	case ProviderVolcArk:
		return ProviderVolcArk
	case ProviderQianfan:
		return ProviderQianfan
	case ProviderModelScope:
		return ProviderModelScope
	default:
		return ProviderCustom
	}
}

func isCodexThirdPartyProvider(provider string) bool {
	return provider == ProviderDeepSeek || provider == ProviderMiMo || provider == ProviderCompshare || provider == ProviderRunAPI || provider == ProviderKimi || provider == ProviderGLM || provider == ProviderMiniMax || provider == ProviderDashScope || provider == ProviderOpenCodeZen || provider == ProviderOpenCodeGo || provider == ProviderXFyun || provider == ProviderTencentLkeap || provider == ProviderVolcArk || provider == ProviderQianfan || provider == ProviderModelScope
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
		return baseURL, apiKey, "", nil
	case ProviderMiMo:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("MiMo API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = defaultMiMoBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderCompshare:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("Compshare API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = compshareClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderRunAPI:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("RunAPI API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = runAPIBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderKimi:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("Kimi API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = kimiClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderGLM:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("GLM API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = glmClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderMiniMax:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("MiniMax API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = miniMaxClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderDashScope:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("DashScope API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = dashScopeClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderOpenCodeZen:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("OpenCode Zen API Key 不能为空")
		}
		return openCodeZenClaudeBaseURL, apiKey, "", nil
	case ProviderOpenCodeGo:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("OpenCode Go API Key 不能为空")
		}
		return openCodeGoClaudeBaseURL, apiKey, "", nil
	case ProviderXFyun:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("讯飞星辰 API Key 不能为空")
		}
		return xfyunClaudeBaseURL, apiKey, "", nil
	case ProviderTencentLkeap:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("腾讯云 TokenHub API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = tencentLkeapClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderVolcArk:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("火山方舟 API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = volcArkClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
	case ProviderQianfan:
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("百度千帆 API Key 不能为空")
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = qianfanClaudeBaseURL
		}
		return baseURL, apiKey, "", nil
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
	case strings.Contains(value, "cp.compshare.cn"):
		return ProviderCompshare
	case strings.Contains(value, "runapi.co"):
		return ProviderRunAPI
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
	case strings.Contains(value, "xf-yun.com"):
		return ProviderXFyun
	case strings.Contains(value, "lkeap.cloud.tencent.com"):
		return ProviderTencentLkeap
	case strings.Contains(value, "volces.com"):
		return ProviderVolcArk
	case strings.Contains(value, "baidubce.com"):
		return ProviderQianfan
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
env_key = "OPENAI_API_KEY"
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
	return writeBytesAtomicWithMode(path, content, 0o600)
}

func writeBytesAtomicWithMode(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if mode != 0 {
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			return err
		}
	}
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

func extractTomlBoolField(content string, key string) (bool, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*(true|false)\s*(?:#.*)?$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return false, false
	}
	return strings.EqualFold(match[1], "true"), true
}

func readEnvValueFromFile(path, key string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(rest)
		}
		k, value, _ := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if k != key {
			continue
		}
		value = strings.TrimSpace(value)
		return strings.Trim(value, `"'`)
	}
	return ""
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

// PreviewApply 预览 Apply 操作的变更，不实际写入文件。
func (s *Service) PreviewApply(req ApplyAgentConfigRequest, port int, accessKey string) (ConfigDiffResult, error) {
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		return ConfigDiffResult{}, fmt.Errorf("agent 平台不能为空")
	}
	switch platform {
	case PlatformClaude:
		return s.previewApplyClaude(req, port, accessKey)
	case PlatformCodex:
		provider := strings.TrimSpace(req.Provider)
		if provider == ProviderOpenAI {
			return s.previewApplyCodexOpenAI(req.APIKey)
		}
		if responsesURL, ok := codexResponsesBaseURL(provider); ok {
			if req.Mode == "quick" {
				return s.previewApplyCodexThirdPartyQuick(provider, responsesURL, req.APIKey)
			}
			return s.previewApplyCodexThirdParty(provider, responsesURL, req.APIKey)
		}
		return s.previewApplyCodex(port, accessKey, req.Mode)
	case PlatformOpenCode:
		return s.previewApplyOpenCode(req, port, accessKey)
	default:
		return ConfigDiffResult{}, fmt.Errorf("不支持的 agent 平台: %s", platform)
	}
}

// PreviewRestore 预览 Restore 操作的变更，不实际写入文件。
func (s *Service) PreviewRestore(platform string) (ConfigDiffResult, error) {
	switch platform {
	case PlatformClaude:
		return s.previewRestoreClaude()
	case PlatformCodex:
		return s.previewRestoreCodex()
	case PlatformOpenCode:
		return s.previewRestoreOpenCode()
	default:
		return ConfigDiffResult{}, fmt.Errorf("不支持的 agent 平台: %s", platform)
	}
}

func (s *Service) previewApplyClaude(req ApplyAgentConfigRequest, port int, accessKey string) (ConfigDiffResult, error) {
	provider := normalizeClaudeProvider(req.Provider)
	baseURL, authToken, apiKey, err := resolveClaudeProvider(req, port, accessKey)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	path := s.claudeSettingsPath()
	data, _, err := readJSONMap(path)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	oldData := copyJSONMap(data)

	env, _ := data["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
		data["env"] = env
	}
	env["ANTHROPIC_BASE_URL"] = baseURL
	if authToken != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = authToken
	} else {
		delete(env, "ANTHROPIC_AUTH_TOKEN")
	}
	if apiKey != "" {
		env["ANTHROPIC_API_KEY"] = apiKey
	} else {
		delete(env, "ANTHROPIC_API_KEY")
	}
	newData := data

	files := []FileDiff{computeJSONDiffWithMask(path, oldData, newData, sensitiveFieldKeys...)}

	if provider == ProviderCCX {
		configPath := s.claudeConfigPath()
		oldConfig, _, _ := readJSONMap(configPath)
		newConfig := copyJSONMap(oldConfig)
		newConfig["primaryApiKey"] = "ccx"
		files = append(files, computeJSONDiff(configPath, oldConfig, newConfig))

		rootPath := s.claudeRootConfigPath()
		oldRoot, _, _ := readJSONMap(rootPath)
		newRoot := copyJSONMap(oldRoot)
		newRoot["hasCompletedOnboarding"] = true
		files = append(files, computeJSONDiff(rootPath, oldRoot, newRoot))
	}
	return ConfigDiffResult{Files: files}, nil
}

func (s *Service) previewApplyCodex(port int, accessKey string, mode string) (ConfigDiffResult, error) {
	if mode == "plugin" {
		return s.previewApplyCodexPlugin(port, accessKey)
	}
	return s.previewApplyCodexQuick(port, accessKey)
}

func (s *Service) previewApplyCodexQuick(port int, accessKey string) (ConfigDiffResult, error) {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()

	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}

	targetURL := codexBaseURL(port)
	updatedConfig := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updatedConfig = upsertTopLevelTomlString(updatedConfig, "openai_base_url", targetURL)
	// 清理插件模式残留
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.ccx", nil)

	newAuthData := copyJSONMap(authData)
	newAuthData["OPENAI_API_KEY"] = accessKey
	newAuthData["auth_mode"] = "apikey"

	return ConfigDiffResult{Files: []FileDiff{
		computeTextDiffWithSensitiveFields(configPath, configContent, updatedConfig, "experimental_bearer_token"),
		computeJSONDiffWithMask(authPath, authData, newAuthData, "OPENAI_API_KEY"),
	}}, nil
}

func (s *Service) previewApplyCodexPlugin(port int, accessKey string) (ConfigDiffResult, error) {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}

	block := fmt.Sprintf(`[model_providers.ccx]
name = "CCX Proxy"
base_url = %q
wire_api = "responses"
requires_openai_auth = true
experimental_bearer_token = %q
`, codexBaseURL(port), accessKey)

	updatedConfig := upsertTopLevelTomlString(configContent, "model_provider", "ccx")
	updatedConfig = restoreTopLevelTomlString(updatedConfig, "openai_base_url", nil)
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.ccx", nil)
	updatedConfig = upsertNamedTomlBlock(updatedConfig, "model_providers.ccx", block)

	newAuthData := copyJSONMap(authData)
	newAuthData["OPENAI_API_KEY"] = accessKey
	newAuthData["auth_mode"] = "chatgpt"

	return ConfigDiffResult{Files: []FileDiff{
		computeTextDiffWithSensitiveFields(configPath, configContent, updatedConfig, "experimental_bearer_token"),
		computeJSONDiffWithMask(authPath, authData, newAuthData, "OPENAI_API_KEY"),
	}}, nil
}

func (s *Service) previewApplyCodexOpenAI(apiKey string) (ConfigDiffResult, error) {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}

	key := strings.TrimSpace(apiKey)
	newAuthData := copyJSONMap(authData)
	if key != "" {
		// API Key 模式：写入 key + auth_mode = "apikey"
		newAuthData["OPENAI_API_KEY"] = key
		newAuthData["auth_mode"] = "apikey"
	} else {
		// OAuth 登录模式：auth_mode = "chatgpt"，OPENAI_API_KEY = null
		newAuthData["auth_mode"] = "chatgpt"
		newAuthData["OPENAI_API_KEY"] = nil
	}

	updatedConfig := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updatedConfig = restoreTopLevelTomlString(updatedConfig, "openai_base_url", nil)
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.ccx", nil)
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.openai", nil)

	return ConfigDiffResult{Files: []FileDiff{
		computeTextDiffWithSensitiveFields(configPath, configContent, updatedConfig, "experimental_bearer_token"),
		computeJSONDiffWithMask(authPath, authData, newAuthData, "OPENAI_API_KEY"),
	}}, nil
}

func (s *Service) previewApplyCodexThirdParty(provider, baseURL, apiKey string) (ConfigDiffResult, error) {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = s.GetSavedProviderKeys()["codex:"+provider]
	}
	if key == "" {
		key = "[未配置]"
	}

	block := fmt.Sprintf(`[model_providers.%s]
name = %q
base_url = %q
wire_api = "responses"
requires_openai_auth = true
experimental_bearer_token = %q
`, provider, provider, baseURL, key)

	updatedConfig := upsertTopLevelTomlString(configContent, "model_provider", provider)
	updatedConfig = restoreTopLevelTomlString(updatedConfig, "openai_base_url", nil) // 清理 CCX proxy 残留
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.ccx", nil)
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.openai", nil)
	updatedConfig = upsertNamedTomlBlock(updatedConfig, "model_providers."+provider, block)

	newAuthData := copyJSONMap(authData)
	newAuthData["OPENAI_API_KEY"] = key
	newAuthData["auth_mode"] = "chatgpt"

	return ConfigDiffResult{Files: []FileDiff{
		computeTextDiffWithSensitiveFields(configPath, configContent, updatedConfig, "experimental_bearer_token"),
		computeJSONDiffWithMask(authPath, authData, newAuthData, "OPENAI_API_KEY"),
	}}, nil
}

func (s *Service) previewApplyCodexThirdPartyQuick(provider, baseURL, apiKey string) (ConfigDiffResult, error) {
	configPath := s.codexConfigPath()
	authPath := s.codexAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = s.GetSavedProviderKeys()["codex:"+provider]
	}
	if key == "" {
		key = "[未配置]"
	}

	// config.toml: model_provider="openai" + openai_base_url=<第三方 URL>
	updatedConfig := upsertTopLevelTomlString(configContent, "model_provider", "openai")
	updatedConfig = upsertTopLevelTomlString(updatedConfig, "openai_base_url", baseURL)
	updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers.ccx", nil)
	// 清理插件模式残留的第三方 provider 块
	if isCodexThirdPartyProvider(provider) {
		updatedConfig = restoreNamedTomlBlock(updatedConfig, "model_providers."+provider, nil)
	}

	newAuthData := copyJSONMap(authData)
	newAuthData["OPENAI_API_KEY"] = key
	newAuthData["auth_mode"] = "apikey"

	return ConfigDiffResult{Files: []FileDiff{
		computeTextDiffWithSensitiveFields(configPath, configContent, updatedConfig, "experimental_bearer_token"),
		computeJSONDiffWithMask(authPath, authData, newAuthData, "OPENAI_API_KEY"),
	}}, nil
}

func (s *Service) previewRestoreClaude() (ConfigDiffResult, error) {
	var state ClaudeProxyState
	if err := readJSONFile(s.claudeStatePath(), &state); err != nil {
		return ConfigDiffResult{}, fmt.Errorf("未找到 Claude 配置状态，请先应用配置")
	}
	path := state.TargetPath
	if !state.FileExisted {
		data, _, _ := readJSONMap(path)
		return ConfigDiffResult{Files: []FileDiff{
			computeJSONDiffWithMask(path, data, nil, sensitiveFieldKeys...),
		}}, nil
	}
	data, _, err := readJSONMap(path)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	oldData := copyJSONMap(data)

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

	return ConfigDiffResult{Files: []FileDiff{
		computeJSONDiffWithMask(path, oldData, data, sensitiveFieldKeys...),
	}}, nil
}

func (s *Service) previewRestoreCodex() (ConfigDiffResult, error) {
	var state CodexProxyState
	if err := readJSONFile(s.codexStatePath(), &state); err != nil {
		return ConfigDiffResult{}, fmt.Errorf("未找到 Codex 配置状态，请先应用配置")
	}

	var files []FileDiff

	// config.toml
	if state.ConfigFileExisted {
		content, _, err := readTextFile(state.ConfigPath)
		if err != nil {
			return ConfigDiffResult{}, err
		}
		restoredContent := restoreTopLevelTomlString(content, "model_provider", state.OriginalModelProvider)
		restoredContent = restoreTopLevelTomlString(restoredContent, "openai_base_url", state.OriginalOpenAIBaseURL)
		restoredContent = restoreNamedTomlBlock(restoredContent, "model_providers.ccx", state.OriginalProviderBlock)
		if state.InjectedProvider != "" && state.InjectedProvider != ProviderCCX && state.InjectedProvider != ProviderOpenAI {
			restoredContent = restoreNamedTomlBlock(restoredContent, "model_providers."+state.InjectedProvider, nil)
		}
		files = append(files, computeTextDiff(state.ConfigPath, content, restoredContent))
	} else {
		content, _, _ := readTextFile(state.ConfigPath)
		files = append(files, computeTextDiff(state.ConfigPath, content, ""))
	}

	// auth.json
	if state.AuthFileExisted {
		authData, _, err := readJSONMap(state.AuthPath)
		if err != nil {
			return ConfigDiffResult{}, err
		}
		restoredAuth := copyJSONMap(authData)
		restoreStringField(restoredAuth, "OPENAI_API_KEY", state.OriginalOpenAIAPIKey)
		files = append(files, computeJSONDiffWithMask(state.AuthPath, authData, restoredAuth, "OPENAI_API_KEY"))
	} else {
		authData, _, _ := readJSONMap(state.AuthPath)
		files = append(files, computeJSONDiffWithMask(state.AuthPath, authData, nil, "OPENAI_API_KEY"))
	}

	return ConfigDiffResult{Files: files}, nil
}

// copyJSONMap 深拷贝一个 JSON map，避免修改原始数据。
func copyJSONMap(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	b, _ := json.Marshal(data)
	var result map[string]any
	_ = json.Unmarshal(b, &result)
	return result
}

func findJSONCStringValue(content string, key string) (string, bool) {
	re := regexp.MustCompile(`(?m)^(\s*)` + `"` + regexp.QuoteMeta(key) + `"` + `\s*:\s*\"([^\"\\]*(?:\\.[^\"\\]*)*)\"`)
	m := re.FindStringSubmatch(content)
	if len(m) < 3 {
		return "", false
	}
	return m[2], true
}

func extractJSONObjectRange(content string, key string) (int, int, bool) {
	re := regexp.MustCompile(`(?m)^(\s*)` + `"` + regexp.QuoteMeta(key) + `"` + `\s*:\s*\{`)
	loc := re.FindStringIndex(content)
	if loc == nil {
		return 0, 0, false
	}
	start := loc[0]
	pos := strings.IndexByte(content[start:], '{')
	if pos < 0 {
		return 0, 0, false
	}
	pos += start
	depth := 0
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false
	i := pos
	for i < len(content) {
		ch := content[i]
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			i++
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(content) && content[i+1] == '/' {
				inBlockComment = false
				i += 2
				continue
			}
			i++
			continue
		}
		if inString {
			if escaped {
				escaped = false
				i++
				continue
			}
			if ch == '\\' {
				escaped = true
				i++
				continue
			}
			if ch == '"' {
				inString = false
			}
			i++
			continue
		}
		if ch == '/' && i+1 < len(content) {
			next := content[i+1]
			if next == '/' {
				inLineComment = true
				i += 2
				continue
			}
			if next == '*' {
				inBlockComment = true
				i += 2
				continue
			}
		}
		if ch == '"' {
			inString = true
			i++
			continue
		}
		if ch == '{' {
			depth++
			i++
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return start, i + 1, true
			}
			i++
			continue
		}
		i++
	}
	return 0, 0, false
}

func extractJSONObjectString(content string, key string) (string, bool) {
	start, end, ok := extractJSONObjectRange(content, key)
	if !ok {
		return "", false
	}
	return strings.TrimRight(content[start:end], "\n"), true
}

func ensureJSONObjectKey(content string, parentKey string, childKey string, childJSON string) string {
	parentStart, parentEnd, ok := extractJSONObjectRange(content, parentKey)
	if !ok {
		block := fmt.Sprintf("  %q: %s", parentKey, childJSON)
		if idx := strings.LastIndex(content, "}"); idx >= 0 {
			head := content[:idx]
			tail := content[idx:]
			if strings.TrimSpace(head) != "" && !strings.HasSuffix(strings.TrimRight(head, " \t\r\n"), ",") && !strings.HasSuffix(strings.TrimRight(head, " \t\r\n"), "{") {
				head = strings.TrimRight(head, " \t\r\n") + ","
			}
			if !strings.HasSuffix(head, "\n") {
				head += "\n"
			}
			return head + block + "\n" + tail
		}
		if strings.TrimSpace(content) == "" {
			return "{\n" + block + "\n}"
		}
		if !strings.HasSuffix(content, "\n") {
			return content + "\n" + block + "\n"
		}
		return content + block + "\n"
	}
	childStart, childEnd, childOK := extractJSONObjectRange(content[parentStart:parentEnd], childKey)
	if childOK {
		absoluteStart := parentStart + childStart
		absoluteEnd := parentStart + childEnd
		return content[:absoluteStart] + childJSON + content[absoluteEnd:]
	}
	parentInner := content[parentStart:parentEnd]
	childBlock := fmt.Sprintf("  %q: %s", childKey, childJSON)
	insertPos := strings.LastIndex(parentInner, "}")
	if insertPos < 0 {
		return content
	}
	absInsert := parentStart + insertPos
	head := content[:absInsert]
	tail := content[absInsert:]
	if strings.TrimSpace(head) != "" && !strings.HasSuffix(strings.TrimRight(head, " \t\r\n"), ",") && !strings.HasSuffix(strings.TrimRight(head, " \t\r\n"), "{") {
		head = strings.TrimRight(head, " \t\r\n") + ","
	}
	if !strings.HasSuffix(head, "\n") {
		head += "\n"
	}
	return head + childBlock + "\n" + tail
}

func patchOpenCodeProviderJSONC(content string, providerID string, providerJSON string) string {
	return ensureJSONObjectKey(content, "provider", providerID, providerJSON)
}

func removeJSONCObjectKey(content string, key string) string {
	re := regexp.MustCompile(`(?m)^(\s*)` + `"` + regexp.QuoteMeta(key) + `"` + `\s*:\s*\{`)
	loc := re.FindStringIndex(content)
	if loc == nil {
		return content
	}
	start, end, ok := extractJSONObjectRange(content, key)
	if !ok {
		return content
	}
	if end < len(content) && content[end] == ',' {
		end++
	}
	if start > 0 && content[start-1] == ',' {
		start--
	}
	if start > 0 && (content[start-1] == '\n' || content[start-1] == '\r') {
		start--
		if start > 0 && content[start-1] == '\r' {
			start--
		}
	}
	if end < len(content) && (content[end] == '\n' || content[end] == '\r') {
		end++
	}
	return content[:start] + content[end:]
}

func openCodeAuthKeyFromMap(authData map[string]any, provider string) (string, string) {
	obj, _ := authData[provider].(map[string]any)
	if obj == nil {
		return "", ""
	}
	authType, _ := obj["type"].(string)
	key, _ := obj["key"].(string)
	return authType, key
}

func upsertOpenCodeAuthKey(authData map[string]any, provider string, key string) map[string]any {
	if authData == nil {
		authData = map[string]any{}
	}
	existing, _ := authData[provider].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	existing["type"] = "api"
	existing["key"] = key
	authData[provider] = existing
	return authData
}

func restoreOpenCodeAuthKey(authData map[string]any, provider string, origType *string, origKey *string) map[string]any {
	if authData == nil {
		authData = map[string]any{}
	}
	if origType == nil && origKey == nil {
		delete(authData, provider)
		return authData
	}
	existing, _ := authData[provider].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	if origType != nil {
		existing["type"] = *origType
	} else {
		delete(existing, "type")
	}
	if origKey != nil {
		existing["key"] = *origKey
	} else {
		delete(existing, "key")
	}
	authData[provider] = existing
	return authData
}

func normalizeOpenCodeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", ProviderCCX:
		return ProviderCCX
	case ProviderDeepSeek:
		return ProviderDeepSeek
	case ProviderMiMo:
		return ProviderMiMo
	case ProviderCompshare:
		return ProviderCompshare
	case ProviderRunAPI:
		return ProviderRunAPI
	case ProviderKimi:
		return ProviderKimi
	case ProviderGLM:
		return ProviderGLM
	case ProviderMiniMax:
		return ProviderMiniMax
	case ProviderDashScope:
		return ProviderDashScope
	case ProviderXFyun:
		return ProviderXFyun
	case ProviderOpenCodeZen:
		return ProviderOpenCodeZen
	case ProviderOpenCodeGo:
		return ProviderOpenCodeGo
	case ProviderTencentLkeap:
		return ProviderTencentLkeap
	case ProviderVolcArk:
		return ProviderVolcArk
	case ProviderQianfan:
		return ProviderQianfan
	default:
		return provider
	}
}

func isOpenCodeDirectProvider(provider string) bool {
	switch provider {
	case ProviderDeepSeek, ProviderMiMo, ProviderCompshare, ProviderRunAPI, ProviderKimi, ProviderGLM, ProviderMiniMax, ProviderDashScope, ProviderXFyun, ProviderOpenCodeZen, ProviderOpenCodeGo, ProviderTencentLkeap, ProviderVolcArk, ProviderQianfan:
		return true
	default:
		return false
	}
}

func openCodeDirectBaseURL(provider string) (string, bool) {
	switch provider {
	case ProviderDeepSeek:
		return "https://api.deepseek.com/v1", true
	case ProviderMiMo:
		return "https://api.xiaomimimo.com/v1", true
	case ProviderCompshare:
		return "https://cp.compshare.cn/v1", true
	case ProviderRunAPI:
		return runAPIBaseURL, true
	case ProviderKimi:
		return "https://api.moonshot.cn/v1", true
	case ProviderGLM:
		return "https://open.bigmodel.cn/api/paas/v4", true
	case ProviderMiniMax:
		return "https://api.minimax.chat/v1", true
	case ProviderDashScope:
		return "https://dashscope.aliyuncs.com/compatible-mode/v1", true
	case ProviderXFyun:
		return xfyunCodexBaseURL, true
	case ProviderOpenCodeZen:
		return openCodeZenBaseURL, true
	case ProviderOpenCodeGo:
		return openCodeGoBaseURL, true
	case ProviderTencentLkeap:
		return tencentLkeapCodexBaseURL, true
	case ProviderVolcArk:
		return volcArkCodexBaseURL, true
	case ProviderQianfan:
		return qianfanCodexBaseURL, true
	default:
		return "", false
	}
}

func openCodeDirectLabel(provider string) string {
	switch provider {
	case ProviderDeepSeek:
		return "DeepSeek"
	case ProviderMiMo:
		return "MiMo"
	case ProviderCompshare:
		return "Compshare"
	case ProviderRunAPI:
		return "RunAPI"
	case ProviderKimi:
		return "Kimi"
	case ProviderGLM:
		return "GLM"
	case ProviderMiniMax:
		return "MiniMax"
	case ProviderDashScope:
		return "DashScope"
	case ProviderXFyun:
		return "讯飞星辰"
	case ProviderOpenCodeZen:
		return "OpenCode Zen"
	case ProviderOpenCodeGo:
		return "OpenCode Go"
	case ProviderTencentLkeap:
		return "腾讯云 TokenHub"
	case ProviderVolcArk:
		return "火山方舟"
	case ProviderQianfan:
		return "百度千帆"
	default:
		return provider
	}
}

func openCodeProviderBlockJSON(providerID string, label string, baseURL string) string {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("      \"npm\": \"@ai-sdk/openai-compatible\",\n"))
	b.WriteString(fmt.Sprintf("      \"name\": %q,\n", label))
	b.WriteString("      \"options\": {\n")
	b.WriteString(fmt.Sprintf("        \"baseURL\": %q\n", baseURL))
	b.WriteString("      },\n")
	b.WriteString("      \"models\": {}\n")
	b.WriteString("    }")
	return b.String()
}

func detectOpenCodeProvider(configContent string, providerID string) (string, string) {
	if strings.TrimSpace(configContent) == "" || strings.TrimSpace(providerID) == "" {
		return "", ""
	}
	block, ok := extractJSONObjectString(configContent, providerID)
	if !ok {
		return "", ""
	}
	baseURL, _ := findJSONCStringValue(block, "baseURL")
	return baseURL, providerID
}

func resolveOpenCodeProvider(req ApplyAgentConfigRequest, port int, accessKey string) (string, string, string, string, string, error) {
	provider := normalizeOpenCodeProvider(req.Provider)
	switch provider {
	case ProviderCCX:
		if port == 0 {
			return "", "", "", "", "", fmt.Errorf("CCX 端口未设置")
		}
		if accessKey == "" {
			return "", "", "", "", "", fmt.Errorf("PROXY_ACCESS_KEY 为空")
		}
		return ProviderCCX, ProviderCCX, codexBaseURL(port), accessKey, ProviderCCX, nil
	default:
		if !isOpenCodeDirectProvider(provider) {
			return "", "", "", "", "", fmt.Errorf("不支持的 OpenCode provider: %s", provider)
		}
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return "", "", "", "", "", fmt.Errorf("%s API Key 不能为空", provider)
		}
		baseURL, ok := openCodeDirectBaseURL(provider)
		if !ok {
			return "", "", "", "", "", fmt.Errorf("%s 缺少 OpenCode Base URL", provider)
		}
		return provider, provider, baseURL, apiKey, provider, nil
	}
}

func (s *Service) getOpenCodeStatus(port int) (AgentConfigStatus, error) {
	configPath := s.openCodeConfigPath()
	authPath := s.openCodeAuthPath()
	target := codexBaseURL(port)
	status := AgentConfigStatus{
		Platform:       PlatformOpenCode,
		Provider:       ProviderCCX,
		TargetProvider: ProviderCCX,
		TargetBaseURL:  target,
		ConfigPath:     configPath,
		AuthPath:       authPath,
		HasState:       fileExists(s.openCodeStatePath()),
	}
	if existing, ok := s.readOpenCodeState(); ok {
		status.Provider = existing.ProviderID
		if existing.ProviderID != ProviderCCX {
			status.TargetProvider = existing.ProviderID
		}
	}
	configContent, configExists, err := readTextFile(configPath)
	if err != nil {
		status.LastError = err.Error()
		return status, nil
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		status.LastError = err.Error()
		return status, nil
	}
	providerID := status.Provider
	if providerID == "" {
		providerID = ProviderCCX
	}
	baseURL, _ := detectOpenCodeProvider(configContent, providerID)
	_, authKey := openCodeAuthKeyFromMap(authData, providerID)
	status.CurrentBaseURL = baseURL
	if providerID == ProviderCCX {
		status.TargetBaseURL = target
	} else if wantURL, ok := openCodeDirectBaseURL(providerID); ok {
		status.TargetBaseURL = wantURL
	} else {
		status.TargetBaseURL = ""
	}
	envAccessKey := strings.TrimSpace(os.Getenv("PROXY_ACCESS_KEY"))
	if providerID == ProviderCCX {
		status.Configured = configExists && baseURL == target && strings.TrimSpace(authKey) != "" && envAccessKey != "" && strings.TrimSpace(authKey) == envAccessKey
		status.MatchesCurrentPort = status.Configured
		status.NeedsUpdate = configExists && (isLocalBaseURL(baseURL) || strings.TrimSpace(authKey) != "") && !status.MatchesCurrentPort
	} else {
		status.Configured = configExists && baseURL != "" && strings.TrimSpace(authKey) != ""
		status.MatchesCurrentPort = status.Configured
		status.NeedsUpdate = configExists && (isLocalBaseURL(baseURL) || strings.TrimSpace(authKey) != "") && !status.MatchesCurrentPort
	}
	return status, nil
}

func (s *Service) applyOpenCode(req ApplyAgentConfigRequest, port int, accessKey string) error {
	providerID, providerLabel, targetURL, apiKey, storedProvider, err := resolveOpenCodeProvider(req, port, accessKey)
	if err != nil {
		return err
	}
	configPath := s.openCodeConfigPath()
	authPath := s.openCodeAuthPath()
	configContent, configExisted, err := readTextFile(configPath)
	if err != nil {
		return err
	}
	authData, authExisted, err := readJSONMap(authPath)
	if err != nil {
		return err
	}
	origProviderJSON, _ := extractJSONObjectString(configContent, providerID)
	origAuthType, origAuthKey := openCodeAuthKeyFromMap(authData, providerID)
	state := OpenCodeProxyState{
		Version:              stateVersion,
		ProviderID:           storedProvider,
		ConfigPath:           configPath,
		AuthPath:             authPath,
		ConfigFileExisted:    configExisted,
		AuthFileExisted:      authExisted,
		OriginalProviderJSON: optionalString(origProviderJSON, origProviderJSON != ""),
		OriginalAuthType:     optionalString(origAuthType, origAuthType != ""),
		OriginalAuthKey:      optionalString(origAuthKey, origAuthKey != ""),
		InjectedBaseURL:      targetURL,
		InjectedAPIKey:       apiKey,
	}
	if existing, ok := s.readOpenCodeState(); ok {
		state.ConfigFileExisted = existing.ConfigFileExisted
		state.AuthFileExisted = existing.AuthFileExisted
		if existing.OriginalProviderJSON != nil {
			state.OriginalProviderJSON = existing.OriginalProviderJSON
		}
		if existing.OriginalAuthType != nil {
			state.OriginalAuthType = existing.OriginalAuthType
		}
		if existing.OriginalAuthKey != nil {
			state.OriginalAuthKey = existing.OriginalAuthKey
		}
	}
	if err := s.writeOpenCodeState(state); err != nil {
		return err
	}
	providerJSON := openCodeProviderBlockJSON(providerID, providerLabel, targetURL)
	updatedConfig := patchOpenCodeProviderJSONC(configContent, providerID, providerJSON)
	if err := writeTextAtomic(configPath, updatedConfig); err != nil {
		return err
	}
	authData = upsertOpenCodeAuthKey(authData, providerID, apiKey)
	return writeJSONAtomic(authPath, authData)
}

func (s *Service) restoreOpenCode() error {
	var state OpenCodeProxyState
	if err := readJSONFile(s.openCodeStatePath(), &state); err != nil {
		return err
	}
	if state.ConfigFileExisted {
		content, _, err := readTextFile(state.ConfigPath)
		if err != nil {
			return err
		}
		if state.OriginalProviderJSON != nil {
			content = patchOpenCodeProviderJSONC(content, state.ProviderID, *state.OriginalProviderJSON)
		} else {
			content = removeJSONCObjectKey(content, state.ProviderID)
		}
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
		authData = restoreOpenCodeAuthKey(authData, state.ProviderID, state.OriginalAuthType, state.OriginalAuthKey)
		if err := writeJSONAtomic(state.AuthPath, authData); err != nil {
			return err
		}
	} else if err := os.Remove(state.AuthPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Remove(s.openCodeStatePath())
}

func (s *Service) previewApplyOpenCode(req ApplyAgentConfigRequest, port int, accessKey string) (ConfigDiffResult, error) {
	providerID, providerLabel, targetURL, apiKey, _, err := resolveOpenCodeProvider(req, port, accessKey)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	configPath := s.openCodeConfigPath()
	authPath := s.openCodeAuthPath()
	configContent, _, err := readTextFile(configPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	authData, _, err := readJSONMap(authPath)
	if err != nil {
		return ConfigDiffResult{}, err
	}
	providerJSON := openCodeProviderBlockJSON(providerID, providerLabel, targetURL)
	updatedConfig := patchOpenCodeProviderJSONC(configContent, providerID, providerJSON)
	newAuth := copyJSONMap(authData)
	newAuth = upsertOpenCodeAuthKey(newAuth, providerID, apiKey)
	return ConfigDiffResult{Files: []FileDiff{
		computeTextDiff(configPath, configContent, updatedConfig),
		computeJSONDiffWithMask(authPath, authData, newAuth, "key"),
	}}, nil
}

func (s *Service) previewRestoreOpenCode() (ConfigDiffResult, error) {
	var state OpenCodeProxyState
	if err := readJSONFile(s.openCodeStatePath(), &state); err != nil {
		return ConfigDiffResult{}, fmt.Errorf("未找到 OpenCode 配置状态，请先应用配置")
	}
	var files []FileDiff
	if state.ConfigFileExisted {
		content, _, err := readTextFile(state.ConfigPath)
		if err != nil {
			return ConfigDiffResult{}, err
		}
		var restored string
		if state.OriginalProviderJSON != nil {
			restored = patchOpenCodeProviderJSONC(content, state.ProviderID, *state.OriginalProviderJSON)
		} else {
			restored = removeJSONCObjectKey(content, state.ProviderID)
		}
		files = append(files, computeTextDiff(state.ConfigPath, content, restored))
	} else {
		content, _, _ := readTextFile(state.ConfigPath)
		files = append(files, computeTextDiff(state.ConfigPath, content, ""))
	}
	if state.AuthFileExisted {
		authData, _, err := readJSONMap(state.AuthPath)
		if err != nil {
			return ConfigDiffResult{}, err
		}
		restoredAuth := copyJSONMap(authData)
		restoredAuth = restoreOpenCodeAuthKey(restoredAuth, state.ProviderID, state.OriginalAuthType, state.OriginalAuthKey)
		files = append(files, computeJSONDiffWithMask(state.AuthPath, authData, restoredAuth, "key"))
	} else {
		authData, _, _ := readJSONMap(state.AuthPath)
		files = append(files, computeJSONDiffWithMask(state.AuthPath, authData, nil, "key"))
	}
	return ConfigDiffResult{Files: files}, nil
}
