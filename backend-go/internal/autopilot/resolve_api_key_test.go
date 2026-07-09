package autopilot

import (
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

// newTestManagerForResolveAPIKey 构造一个只挂了 cfgManager 的最小 Manager，
// 用于测试 ResolveAPIKey（不依赖 profiler/metrics 等重量组件）。
func newTestManagerForResolveAPIKey(t *testing.T, cfg config.Config) *Manager {
	t.Helper()
	cfgManager, cleanup := createTestConfigManager(t, cfg)
	t.Cleanup(cleanup)
	return &Manager{cfgManager: cfgManager}
}

func TestManager_ResolveAPIKey_Hit(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				ChannelUID:  "ch-abc",
				BaseURL:     "https://example.com",
				APIKeys:     []string{"sk-real-key-1", "sk-real-key-2"},
				ServiceType: "claude",
			},
		},
	}
	mgr := newTestManagerForResolveAPIKey(t, cfg)

	keyHash := KeyHashFromAPIKey("sk-real-key-2")
	key, ok := mgr.ResolveAPIKey("ch-abc", keyHash)
	if !ok {
		t.Fatal("应命中 ch-abc 下的 sk-real-key-2")
	}
	if key != "sk-real-key-2" {
		t.Errorf("解析出的 key: got %q, want sk-real-key-2", key)
	}
}

func TestManager_ResolveAPIKey_ChannelUIDMismatch(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				ChannelUID:  "ch-abc",
				BaseURL:     "https://example.com",
				APIKeys:     []string{"sk-real-key-1"},
				ServiceType: "claude",
			},
		},
	}
	mgr := newTestManagerForResolveAPIKey(t, cfg)

	// channelUID 不存在于配置中（渠道已被删除）
	_, ok := mgr.ResolveAPIKey("ch-deleted", KeyHashFromAPIKey("sk-real-key-1"))
	if ok {
		t.Error("channelUID 不匹配应返回 ok=false")
	}
}

func TestManager_ResolveAPIKey_KeyHashMismatch(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				ChannelUID:  "ch-abc",
				BaseURL:     "https://example.com",
				APIKeys:     []string{"sk-real-key-1"},
				ServiceType: "claude",
			},
		},
	}
	mgr := newTestManagerForResolveAPIKey(t, cfg)

	// key 已轮换：keyHash 对应的旧 key 已不在 APIKeys 列表中
	_, ok := mgr.ResolveAPIKey("ch-abc", KeyHashFromAPIKey("sk-rotated-out"))
	if ok {
		t.Error("keyHash 不匹配（key 已轮换）应返回 ok=false")
	}
}

func TestManager_ResolveAPIKey_EmptyArgs(t *testing.T) {
	mgr := newTestManagerForResolveAPIKey(t, config.Config{})

	if _, ok := mgr.ResolveAPIKey("", "somehash"); ok {
		t.Error("channelUID 为空应返回 ok=false")
	}
	if _, ok := mgr.ResolveAPIKey("ch-abc", ""); ok {
		t.Error("keyHash 为空应返回 ok=false")
	}
}

// 覆盖跨渠道类型：ResolveAPIKey 内部通过 gatherChannelEntries 遍历全部 6 类渠道列表，
// 应能在非 messages 列表（如 chat）中命中。
func TestManager_ResolveAPIKey_AcrossChannelKinds(t *testing.T) {
	cfg := config.Config{
		ChatUpstream: []config.UpstreamConfig{
			{
				ChannelUID:  "ch-chat-1",
				BaseURL:     "https://chat.example.com",
				APIKeys:     []string{"sk-chat-key"},
				ServiceType: "openai",
			},
		},
	}
	mgr := newTestManagerForResolveAPIKey(t, cfg)

	key, ok := mgr.ResolveAPIKey("ch-chat-1", KeyHashFromAPIKey("sk-chat-key"))
	if !ok || key != "sk-chat-key" {
		t.Errorf("应在 ChatUpstream 中命中: key=%q ok=%v", key, ok)
	}
}
