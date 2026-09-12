package autopilot

import "testing"

// 协议端点学习闭环（路由侧读取）的回归测试。
// 写入侧识别器见 handlers/common/protocol_unsupported_signal_test.go。

func TestLearnedProtocolUnsupported(t *testing.T) {
	orig := learnedProtocolUnsupportedLookup
	defer func() { learnedProtocolUnsupportedLookup = orig }()

	learned := map[string]bool{}
	learnedProtocolUnsupportedLookup = func(channelUID, protocol, model string) bool {
		return learned[channelUID+"|"+protocol+"|"+model]
	}

	if learnedProtocolUnsupported("ch_a", "responses", "m1") {
		t.Fatal("无记忆时应 fail-open 返回 false")
	}
	if learnedProtocolUnsupported("", "responses", "m1") ||
		learnedProtocolUnsupported("ch_a", "", "m1") ||
		learnedProtocolUnsupported("ch_a", "responses", "") {
		t.Fatal("空参数应直接返回 false")
	}

	learned["ch_a|responses|m1"] = true
	if !learnedProtocolUnsupported("ch_a", "responses", "m1") {
		t.Fatal("有记忆时应返回 true")
	}
	// 协议维隔离：同模型 chat 端点不受 responses 结论影响
	if learnedProtocolUnsupported("ch_a", "chat", "m1") {
		t.Fatal("其他协议端点不应命中")
	}
	if learnedProtocolUnsupported("ch_a", "responses", "m2") {
		t.Fatal("其他模型不应命中")
	}
}

func TestFilterLearnedToolCallCapable(t *testing.T) {
	orig := learnedToolCallUnsupportedLookup
	defer func() { learnedToolCallUnsupportedLookup = orig }()
	learnedToolCallUnsupportedLookup = func(channelUID, model string) bool {
		return channelUID == "ch_a" && model == "bad-model"
	}

	profiles := []ModelProfile{
		{ModelID: "bad-model"},
		{ModelID: "good-model"},
		{ModelID: "also-good"},
	}
	got := filterLearnedToolCallCapable(profiles, "ch_a")
	if len(got) != 2 || got[0].ModelID != "good-model" || got[1].ModelID != "also-good" {
		t.Fatalf("应剔除 bad-model，保留其余，got %v", got)
	}

	all := []ModelProfile{{ModelID: "bad-model"}}
	if got := filterLearnedToolCallCapable(all, "ch_a"); len(got) != 0 {
		t.Fatalf("全部剔除时应为空，got %v", got)
	}
}

// ── 工具调用正向白名单（运行期证据）──

func TestFilterLearnedToolCallCapableWhitelistMode(t *testing.T) {
	origUnsupported := learnedToolCallUnsupportedLookup
	origVerified := verifiedToolCallModelsLookup
	defer func() {
		learnedToolCallUnsupportedLookup = origUnsupported
		verifiedToolCallModelsLookup = origVerified
	}()
	learnedToolCallUnsupportedLookup = func(channelUID, model string) bool {
		return channelUID == "ch_a" && model == "bad-model"
	}

	profiles := []ModelProfile{
		{ModelID: "bad-model"},   // 黑名单
		{ModelID: "plain-model"}, // 无记录（fail-open 基线：保留）
		{ModelID: "Good-Model"},  // 大小写归一验证
	}

	// 无白名单记录：黑名单模式（fail-open，不堵冷启动渠道）
	verifiedToolCallModelsLookup = func(string) map[string]bool { return nil }
	got := filterLearnedToolCallCapable(profiles, "ch_a")
	if len(got) != 2 || got[0].ModelID != "plain-model" || got[1].ModelID != "Good-Model" {
		t.Fatalf("无白名单时应保持黑名单模式，got %v", got)
	}

	// 白名单存在：候选只从验证组合产生（大小写不敏感），黑名单/未验证全部出局
	verifiedToolCallModelsLookup = func(string) map[string]bool {
		return map[string]bool{"good-model": true}
	}
	got = filterLearnedToolCallCapable(profiles, "ch_a")
	if len(got) != 1 || got[0].ModelID != "Good-Model" {
		t.Fatalf("白名单模式应只留验证组合，got %v", got)
	}

	// 验证组合与候选无交集：回退黑名单逻辑，不空转
	verifiedToolCallModelsLookup = func(string) map[string]bool {
		return map[string]bool{"other-model": true}
	}
	got = filterLearnedToolCallCapable(profiles, "ch_a")
	if len(got) != 2 {
		t.Fatalf("白名单无交集时应回退黑名单模式，got %v", got)
	}
}

// ── 渠道间排他（白名单渠道集合）──

func TestVerifiedToolCallChannelExclusivity(t *testing.T) {
	origModels := verifiedToolCallModelsLookup
	origChannels := verifiedToolCallChannelsLookup
	defer func() {
		verifiedToolCallModelsLookup = origModels
		verifiedToolCallChannelsLookup = origChannels
	}()

	profiles := []ModelProfile{{ModelID: "some-model"}}

	// 全局无白名单渠道：fail-open，一切照旧（黑名单逻辑）
	verifiedToolCallChannelsLookup = func() map[string]bool { return nil }
	verifiedToolCallModelsLookup = func(string) map[string]bool { return nil }
	if got := filterLearnedToolCallCapable(profiles, "ch_any"); len(got) != 1 {
		t.Fatalf("无白名单渠道时应 fail-open，got %v", got)
	}
}
