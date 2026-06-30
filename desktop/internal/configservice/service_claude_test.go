package configservice

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNew(t *testing.T) {
	svc := newTestService(t)
	if svc.homeDir == "" {
		t.Error("homeDir should not be empty")
	}
	if svc.stateDir == "" {
		t.Error("stateDir should not be empty")
	}
	if _, err := os.Stat(svc.stateDir); os.IsNotExist(err) {
		t.Error("stateDir should be created")
	}
}

func TestNewWithDefaultDataDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_STATE_HOME", "")
	}
	svc, err := New("")
	if err != nil {
		t.Fatalf("New('') failed: %v", err)
	}
	expected := filepath.Join(tmpHome, ".config", "ccx-desktop", "agent-config-state")
	if runtime.GOOS == "linux" {
		expected = filepath.Join(tmpHome, ".local", "state", "ccx", "agent-config-state")
	}
	if svc.stateDir != expected {
		t.Errorf("stateDir = %q, want %q", svc.stateDir, expected)
	}
}

func TestGetStatusClaude_NoConfig(t *testing.T) {
	svc := newTestService(t)
	status, err := svc.GetStatus(PlatformClaude, 3688)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Configured {
		t.Error("should not be configured when no settings.json exists")
	}
	// 空 base_url 时 detectClaudeProvider 返回 ""
	if status.Provider != "" {
		t.Errorf("provider = %q, want empty", status.Provider)
	}
}

func TestGetStatusClaude_CCXProvider(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	data := map[string]any{
		"env": map[string]any{
			"ANTHROPIC_BASE_URL": "http://127.0.0.1:3688",
		},
	}
	writeJSON(settingsPath, data)

	status, err := svc.GetStatus(PlatformClaude, 3688)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Provider != ProviderCCX {
		t.Errorf("provider = %q, want %q", status.Provider, ProviderCCX)
	}
	if !status.MatchesCurrentPort {
		t.Error("should match current port")
	}
	if status.NeedsUpdate {
		t.Error("should not need update when port matches")
	}
}

func TestGetStatusClaude_DeepSeekProvider(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	data := map[string]any{
		"env": map[string]any{
			"ANTHROPIC_BASE_URL": "https://api.deepseek.com/anthropic",
		},
	}
	writeJSON(settingsPath, data)

	status, err := svc.GetStatus(PlatformClaude, 3688)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Provider != ProviderDeepSeek {
		t.Errorf("provider = %q, want %q", status.Provider, ProviderDeepSeek)
	}
	if !status.Configured {
		t.Error("should be configured for DeepSeek")
	}
}

func TestGetStatusClaude_CompshareProvider(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	data := map[string]any{
		"env": map[string]any{
			"ANTHROPIC_BASE_URL": "https://cp.compshare.cn",
		},
	}
	writeJSON(settingsPath, data)

	status, err := svc.GetStatus(PlatformClaude, 3688)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Provider != ProviderCompshare {
		t.Errorf("provider = %q, want %q", status.Provider, ProviderCompshare)
	}
	if !status.Configured {
		t.Error("should be configured for Compshare")
	}
}

func TestGetStatusClaude_RunAPIProvider(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	data := map[string]any{
		"env": map[string]any{
			"ANTHROPIC_BASE_URL": "https://runapi.co/v1",
		},
	}
	writeJSON(settingsPath, data)

	status, err := svc.GetStatus(PlatformClaude, 3688)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Provider != ProviderRunAPI {
		t.Errorf("provider = %q, want %q", status.Provider, ProviderRunAPI)
	}
	if !status.Configured {
		t.Error("should be configured for RunAPI")
	}
}

func TestApplyAndRestoreClaudeXFyunProvider(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	original := map[string]any{
		"env": map[string]any{
			"ANTHROPIC_MODEL":            "original-model",
			"ANTHROPIC_SMALL_FAST_MODEL": "original-small",
		},
	}
	writeJSON(settingsPath, original)

	err := svc.Apply(ApplyAgentConfigRequest{Platform: PlatformClaude, Provider: ProviderXFyun, APIKey: "xf-test-key"}, 3688, "proxy-key")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	var after map[string]any
	readJSON(settingsPath, &after)
	env := after["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != xfyunClaudeBaseURL {
		t.Errorf("base_url = %v, want %s", env["ANTHROPIC_BASE_URL"], xfyunClaudeBaseURL)
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "xf-test-key" {
		t.Errorf("auth_token = %v", env["ANTHROPIC_AUTH_TOKEN"])
	}
	if env["ANTHROPIC_MODEL"] != "astron-code-latest" {
		t.Errorf("model = %v", env["ANTHROPIC_MODEL"])
	}
	if env["ANTHROPIC_SMALL_FAST_MODEL"] != "astron-code-latest" {
		t.Errorf("small_fast_model = %v", env["ANTHROPIC_SMALL_FAST_MODEL"])
	}

	err = svc.Restore(PlatformClaude)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	var restored map[string]any
	readJSON(settingsPath, &restored)
	env = restored["env"].(map[string]any)
	if env["ANTHROPIC_MODEL"] != "original-model" {
		t.Errorf("restored model = %v", env["ANTHROPIC_MODEL"])
	}
	if env["ANTHROPIC_SMALL_FAST_MODEL"] != "original-small" {
		t.Errorf("restored small_fast_model = %v", env["ANTHROPIC_SMALL_FAST_MODEL"])
	}
	if _, ok := env["ANTHROPIC_BASE_URL"]; ok {
		t.Errorf("base_url should be removed after restore, got %v", env["ANTHROPIC_BASE_URL"])
	}
}

func TestApplyAndRestoreClaude(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	configPath := filepath.Join(svc.homeDir, ".claude", "config.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	original := map[string]any{
		"env": map[string]any{
			"ANTHROPIC_BASE_URL": "https://original.example.com",
		},
	}
	writeJSON(settingsPath, original)
	writeJSON(configPath, map[string]any{"primaryApiKey": "sk-original"})

	err := svc.Apply(ApplyAgentConfigRequest{Platform: PlatformClaude}, 3688, "test-key")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// 验证注入后配置
	var after map[string]any
	readJSON(settingsPath, &after)
	env := after["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:3688" {
		t.Errorf("base_url = %v", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "test-key" {
		t.Errorf("auth_token = %v", env["ANTHROPIC_AUTH_TOKEN"])
	}
	var configAfter map[string]any
	readJSON(configPath, &configAfter)
	if configAfter["primaryApiKey"] != "sk-original" {
		t.Errorf("primaryApiKey should be preserved, got %v", configAfter["primaryApiKey"])
	}

	// Restore
	err = svc.Restore(PlatformClaude)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	var restored map[string]any
	readJSON(settingsPath, &restored)
	env = restored["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != "https://original.example.com" {
		t.Errorf("restored base_url = %v", env["ANTHROPIC_BASE_URL"])
	}
}

func TestPreviewApplyClaudeDoesNotTouchPrimaryAPIKey(t *testing.T) {
	svc := newTestService(t)
	settingsPath := filepath.Join(svc.homeDir, ".claude", "settings.json")
	configPath := filepath.Join(svc.homeDir, ".claude", "config.json")
	writeJSON(settingsPath, map[string]any{"env": map[string]any{}})
	writeJSON(configPath, map[string]any{"primaryApiKey": "sk-original"})

	diff, err := svc.PreviewApply(ApplyAgentConfigRequest{Platform: PlatformClaude}, 3688, "test-key")
	if err != nil {
		t.Fatalf("PreviewApply failed: %v", err)
	}
	for _, file := range diff.Files {
		if file.Path == configPath {
			t.Fatalf("preview should not include config.json diff: %+v", file)
		}
	}
}

func TestSaveAndLoadProviderKeys(t *testing.T) {
	svc := newTestService(t)

	err := svc.SaveProviderKeyAsset(ProviderKeyAsset{
		Provider: ProviderDeepSeek,
		APIKey:   "sk-test-key",
		BaseURL:  "https://api.deepseek.com",
	})
	if err != nil {
		t.Fatalf("SaveProviderKeyAsset failed: %v", err)
	}
	err = svc.SaveProviderKeyAsset(ProviderKeyAsset{
		Provider: ProviderCompshare,
		APIKey:   "cs-test-key",
		BaseURL:  "https://cp.compshare.cn",
	})
	if err != nil {
		t.Fatalf("SaveProviderKeyAsset failed: %v", err)
	}
	err = svc.SaveProviderKeyAsset(ProviderKeyAsset{
		Provider: ProviderRunAPI,
		APIKey:   "runapi-test-key",
		BaseURL:  "https://runapi.co/v1",
	})
	if err != nil {
		t.Fatalf("SaveProviderKeyAsset failed: %v", err)
	}

	keys := svc.GetSavedProviderKeys()
	if keys["channel:"+ProviderDeepSeek] != "sk-test-key" {
		t.Errorf("channel key = %q", keys["channel:"+ProviderDeepSeek])
	}
	if keys[PlatformClaude+":"+ProviderDeepSeek] != "sk-test-key" {
		t.Errorf("claude key = %q", keys[PlatformClaude+":"+ProviderDeepSeek])
	}
	if keys["channel:"+ProviderCompshare] != "cs-test-key" {
		t.Errorf("compshare channel key = %q", keys["channel:"+ProviderCompshare])
	}
	if keys[PlatformClaude+":"+ProviderCompshare] != "cs-test-key" {
		t.Errorf("compshare claude key = %q", keys[PlatformClaude+":"+ProviderCompshare])
	}
	if keys["channel:"+ProviderRunAPI] != "runapi-test-key" {
		t.Errorf("runapi channel key = %q", keys["channel:"+ProviderRunAPI])
	}
	if keys[PlatformClaude+":"+ProviderRunAPI] != "runapi-test-key" {
		t.Errorf("runapi claude key = %q", keys[PlatformClaude+":"+ProviderRunAPI])
	}

	assets := svc.GetProviderKeyAssets()
	foundDeepSeek := false
	foundCompshare := false
	foundRunAPI := false
	for _, a := range assets {
		switch a.Provider {
		case ProviderDeepSeek:
			foundDeepSeek = true
			if a.APIKey != "sk-test-key" {
				t.Errorf("asset APIKey = %q", a.APIKey)
			}
		case ProviderCompshare:
			foundCompshare = true
			if a.APIKey != "cs-test-key" {
				t.Errorf("compshare asset APIKey = %q", a.APIKey)
			}
		case ProviderRunAPI:
			foundRunAPI = true
			if a.APIKey != "runapi-test-key" {
				t.Errorf("runapi asset APIKey = %q", a.APIKey)
			}
		}
	}
	if !foundDeepSeek {
		t.Error("DeepSeek asset not found")
	}
	if !foundCompshare {
		t.Error("Compshare asset not found")
	}
	if !foundRunAPI {
		t.Error("RunAPI asset not found")
	}
}

// ── P1 回归: Codex 第三方 provider 状态识别 ──
