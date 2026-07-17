package handlers

import (
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

func TestBuildCapabilityCacheKeyIncludesModelsDimension(t *testing.T) {
	keyA := buildCapabilityCacheKey(
		"https://example.com",
		"sk-test",
		"responses",
		[]string{"responses", "messages"},
		[]string{"gpt-4o", "claude-3-7-sonnet"},
		"",
	)
	keyB := buildCapabilityCacheKey(
		"https://example.com",
		"sk-test",
		"responses",
		[]string{"messages", "responses"},
		[]string{"claude-3-7-sonnet", "gpt-4o"},
		"",
	)
	keyC := buildCapabilityCacheKey(
		"https://example.com",
		"sk-test",
		"responses",
		[]string{"messages", "responses"},
		[]string{"gpt-4o-mini"},
		"",
	)

	if keyA != keyB {
		t.Fatalf("same model set should yield same key, got %q != %q", keyA, keyB)
	}
	if keyA == keyC {
		t.Fatalf("different model set should yield different key, got %q", keyA)
	}
}

func TestBuildCapabilityCacheKeyIncludesModelMappingHash(t *testing.T) {
	base := buildCapabilityCacheKey(
		"https://example.com",
		"sk-test",
		"messages",
		[]string{"messages"},
		nil,
		"",
	)
	withHash := buildCapabilityCacheKey(
		"https://example.com",
		"sk-test",
		"messages",
		[]string{"messages"},
		nil,
		hashModelMapping(map[string]string{"claude-opus-4-7": "anthropic/claude-opus-4"}),
	)
	withDifferentHash := buildCapabilityCacheKey(
		"https://example.com",
		"sk-test",
		"messages",
		[]string{"messages"},
		nil,
		hashModelMapping(map[string]string{"claude-opus-4-7": "anthropic/claude-opus-4-v2"}),
	)

	if base == withHash {
		t.Fatalf("empty vs non-empty mapping should yield different keys, got %q", base)
	}
	if withHash == withDifferentHash {
		t.Fatalf("different mappings should yield different keys, got %q", withHash)
	}
}

func TestHashModelMappingStable(t *testing.T) {
	a := hashModelMapping(map[string]string{"a": "x", "b": "y"})
	b := hashModelMapping(map[string]string{"b": "y", "a": "x"})
	if a != b {
		t.Fatalf("hash should be order-independent, got %q vs %q", a, b)
	}
	if hashModelMapping(nil) != "" {
		t.Fatalf("nil mapping should produce empty hash")
	}
	if hashModelMapping(map[string]string{}) != "" {
		t.Fatalf("empty mapping should produce empty hash")
	}
}

func TestHashCapabilityProbePoolTracksAllKeysAndBaseURLs(t *testing.T) {
	single := &config.UpstreamConfig{
		BaseURL: "https://one.example.com",
		APIKeys: []string{"sk-one"},
	}
	if got := hashCapabilityProbePool(single); got != "" {
		t.Fatalf("单 Key 单 BaseURL 不应增加缓存维度，got %q", got)
	}

	multi := &config.UpstreamConfig{
		BaseURLs: []string{"https://one.example.com", "https://two.example.com"},
		APIKeys:  []string{"sk-one", "sk-two"},
	}
	same := multi.Clone()
	changedKey := multi.Clone()
	changedKey.APIKeys[1] = "sk-three"
	changedURL := multi.Clone()
	changedURL.BaseURLs[1] = "https://three.example.com"

	baseHash := hashCapabilityProbePool(multi)
	if baseHash == "" || baseHash != hashCapabilityProbePool(same) {
		t.Fatalf("多 Key/BaseURL 哈希不稳定: %q vs %q", baseHash, hashCapabilityProbePool(same))
	}
	if baseHash == hashCapabilityProbePool(changedKey) {
		t.Fatal("Key 池变化后缓存哈希未变化")
	}
	if baseHash == hashCapabilityProbePool(changedURL) {
		t.Fatal("BaseURL 池变化后缓存哈希未变化")
	}
	if capabilityProbeCacheAPIKey(multi, multi.APIKeys[0]) == multi.APIKeys[0] {
		t.Fatal("多 Key 探测缓存身份未包含池哈希")
	}
}
