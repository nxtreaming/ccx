package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestChannelUID_GenerateOnMissing 测试缺失 ChannelUID 时自动生成
func TestChannelUID_GenerateOnMissing(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	// 写入不含 channelUid 的配置
	initial := `{"upstream":[{"baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"ch1"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	if err := os.WriteFile(cfgFile, []byte(initial), 0600); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	cm, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	defer cm.Close()

	cfg := cm.GetConfig()

	if len(cfg.Upstream) != 1 {
		t.Fatalf("期望 1 个 upstream，得到 %d", len(cfg.Upstream))
	}
	uid := cfg.Upstream[0].ChannelUID
	if uid == "" {
		t.Fatal("ChannelUID 不应为空")
	}
	if len(uid) != 3+len("123456789012") {
		// "ch_" (3) + 12 hex = 15
		t.Fatalf("ChannelUID 长度异常: %q (len=%d)", uid, len(uid))
	}
	if uid[:3] != "ch_" {
		t.Fatalf("ChannelUID 应以 'ch_' 开头: %q", uid)
	}
}

// TestChannelUID_PreserveExisting 测试已有 ChannelUID 不被覆盖
func TestChannelUID_PreserveExisting(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	initial := `{"upstream":[{"channelUid":"ch_custom_abc123","baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"ch1"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	if err := os.WriteFile(cfgFile, []byte(initial), 0600); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	cm, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	defer cm.Close()

	cfg := cm.GetConfig()
	if cfg.Upstream[0].ChannelUID != "ch_custom_abc123" {
		t.Fatalf("已有 ChannelUID 不应被覆盖: got %q", cfg.Upstream[0].ChannelUID)
	}
}

// TestChannelUID_StableOnReorder 测试渠道重排后 ChannelUID 不变
func TestChannelUID_StableOnReorder(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	// 两个渠道，加载后获取各自的 UID
	initial := `{"upstream":[{"baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"first"},{"baseUrl":"https://b.example.com","apiKeys":["k2"],"name":"second"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	if err := os.WriteFile(cfgFile, []byte(initial), 0600); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	cm, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	defer cm.Close()

	cfg := cm.GetConfig()
	uidA := cfg.Upstream[0].ChannelUID
	uidB := cfg.Upstream[1].ChannelUID
	if uidA == "" || uidB == "" {
		t.Fatal("ChannelUID 不应为空")
	}
	if uidA == uidB {
		t.Fatal("不同渠道的 ChannelUID 不应相同")
	}

	// 关闭后写入重排的配置（second 在前）
	reordered := `{"upstream":[{"channelUid":"` + uidB + `","baseUrl":"https://b.example.com","apiKeys":["k2"],"name":"second"},{"channelUid":"` + uidA + `","baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"first"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	cm.Close()
	if err := os.WriteFile(cfgFile, []byte(reordered), 0600); err != nil {
		t.Fatalf("写入重排配置失败: %v", err)
	}

	cm2, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}
	defer cm2.Close()

	cfg2 := cm2.GetConfig()
	// second 现在在索引 0，first 在索引 1
	if cfg2.Upstream[0].ChannelUID != uidB {
		t.Fatalf("重排后 second 的 ChannelUID 应为 %q, got %q", uidB, cfg2.Upstream[0].ChannelUID)
	}
	if cfg2.Upstream[1].ChannelUID != uidA {
		t.Fatalf("重排后 first 的 ChannelUID 应为 %q, got %q", uidA, cfg2.Upstream[1].ChannelUID)
	}
}

// TestChannelUID_NewChannelGetsNewUID 测试新增渠道获得新 UID，不影响已有渠道
func TestChannelUID_NewChannelGetsNewUID(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	// 先加载一个渠道
	initial := `{"upstream":[{"channelUid":"ch_existing000001","baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"existing"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	if err := os.WriteFile(cfgFile, []byte(initial), 0600); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	cm, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	defer cm.Close()

	// 在配置中追加一个新渠道（没有 channelUid）
	cfg := cm.GetConfig()
	existingUID := cfg.Upstream[0].ChannelUID

	cfg.Upstream = append(cfg.Upstream, UpstreamConfig{
		BaseURL: "https://b.example.com",
		APIKeys: []string{"k2"},
		Name:    "newcomer",
	})

	// 保存后重新加载
	cm.Close()
	// 序列化 + 写入
	cfgJSON, _ := os.ReadFile(cfgFile)
	_ = cfgJSON // 通过 SaveConfig 或手动写
	updatedJSON := `{"upstream":[{"channelUid":"ch_existing000001","baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"existing"},{"baseUrl":"https://b.example.com","apiKeys":["k2"],"name":"newcomer"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	if err := os.WriteFile(cfgFile, []byte(updatedJSON), 0600); err != nil {
		t.Fatalf("写入更新配置失败: %v", err)
	}

	cm2, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}
	defer cm2.Close()

	cfg2 := cm2.GetConfig()
	if len(cfg2.Upstream) != 2 {
		t.Fatalf("期望 2 个 upstream，得到 %d", len(cfg2.Upstream))
	}
	// 已有渠道 UID 不变
	if cfg2.Upstream[0].ChannelUID != existingUID {
		t.Fatalf("已有渠道 UID 不应改变: expected %q, got %q", existingUID, cfg2.Upstream[0].ChannelUID)
	}
	// 新渠道获得新 UID
	newUID := cfg2.Upstream[1].ChannelUID
	if newUID == "" {
		t.Fatal("新渠道的 ChannelUID 不应为空")
	}
	if newUID == existingUID {
		t.Fatal("新渠道的 ChannelUID 不应与已有渠道相同")
	}
	if newUID[:3] != "ch_" {
		t.Fatalf("新渠道 ChannelUID 应以 'ch_' 开头: %q", newUID)
	}
}

// TestChannelUID_AllChannelKinds 测试全部六类渠道都覆盖了 ChannelUID 补齐
func TestChannelUID_AllChannelKinds(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	initial := `{
		"upstream":[{"baseUrl":"https://msg.example.com","apiKeys":["k1"],"name":"msg"}],
		"responsesUpstream":[{"baseUrl":"https://resp.example.com","apiKeys":["k2"],"name":"resp"}],
		"geminiUpstream":[{"baseUrl":"https://gem.example.com","apiKeys":["k3"],"name":"gem"}],
		"chatUpstream":[{"baseUrl":"https://chat.example.com","apiKeys":["k4"],"name":"chat"}],
		"imagesUpstream":[{"baseUrl":"https://img.example.com","apiKeys":["k5"],"name":"img"}],
		"vectorsUpstream":[{"baseUrl":"https://vec.example.com","apiKeys":["k6"],"name":"vec"}]
	}`
	if err := os.WriteFile(cfgFile, []byte(initial), 0600); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	cm, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	defer cm.Close()

	cfg := cm.GetConfig()

	type channelKind struct {
		name     string
		channels []UpstreamConfig
	}
	kinds := []channelKind{
		{"Messages", cfg.Upstream},
		{"Responses", cfg.ResponsesUpstream},
		{"Gemini", cfg.GeminiUpstream},
		{"Chat", cfg.ChatUpstream},
		{"Images", cfg.ImagesUpstream},
		{"Vectors", cfg.VectorsUpstream},
	}

	for _, k := range kinds {
		if len(k.channels) != 1 {
			t.Fatalf("%s: 期望 1 个渠道, got %d", k.name, len(k.channels))
		}
		uid := k.channels[0].ChannelUID
		if uid == "" {
			t.Fatalf("%s: ChannelUID 不应为空", k.name)
		}
		if uid[:3] != "ch_" {
			t.Fatalf("%s: ChannelUID 应以 'ch_' 开头: %q", k.name, uid)
		}
	}
}

// TestChannelUID_PersistedAfterBackfill 测试补齐后持久化到文件
func TestChannelUID_PersistedAfterBackfill(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	initial := `{"upstream":[{"baseUrl":"https://a.example.com","apiKeys":["k1"],"name":"ch1"}],"responsesUpstream":[],"geminiUpstream":[],"chatUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]}`
	if err := os.WriteFile(cfgFile, []byte(initial), 0600); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	cm, err := NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("NewConfigManager 失败: %v", err)
	}
	uid := cm.GetConfig().Upstream[0].ChannelUID
	cm.Close()

	// 重新从文件读取，验证 UID 已持久化
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	cfgStr := string(data)
	if !containsSubstring(cfgStr, uid) {
		t.Fatalf("配置文件中应包含 ChannelUID %q, 文件内容:\n%s", uid, cfgStr)
	}
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestChannelUID_Unique 测试生成的 UID 具有唯一性（批量生成 100 个无重复）
func TestChannelUID_Unique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		uid := generateChannelUID()
		if seen[uid] {
			t.Fatalf("第 %d 次生成的 ChannelUID 与之前的重复: %q", i, uid)
		}
		seen[uid] = true
	}
}
