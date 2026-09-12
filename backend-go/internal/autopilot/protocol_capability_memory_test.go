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
