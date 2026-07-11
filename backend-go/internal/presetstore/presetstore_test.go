package presetstore

import (
	"sync"
	"testing"
)

// validBundle 返回一份通过校验的最小 bundle，供各用例按需改坏。
func validBundle() *PresetBundle {
	return &PresetBundle{
		SchemaVersion: CurrentSchemaVersion,
		DataVersion:   "2026.07.10-1",
		Subscription: SubscriptionPreset{
			OriginTypes: []OriginTypeEntry{
				{Value: "official_api", Tier: "first"},
				{Value: "relay", Tier: "second"},
				{Value: "community", Tier: "third"},
				{Value: "unknown", Tier: "unknown"},
			},
			BillingModes:         []string{"token_plan", "unknown"},
			Sources:              []string{"manual", "auto_discovered"},
			AutoRefreshProviders: []string{"openai", "anthropic", "google"},
			NewApiDefaults:       NewApiDefaults{OriginType: "relay", OriginTier: "second", BillingMode: "token_plan"},
			OriginTypeAliases:    map[string]string{"public_benefit": "community"},
		},
	}
}

func TestEmbeddedBundleValid(t *testing.T) {
	b := EmbeddedBundle()
	if err := Validate(b); err != nil {
		t.Fatalf("内置 bundle 应通过校验，得到: %v", err)
	}
	if b.DataVersion != "" {
		t.Errorf("内置 bundle DataVersion 应为空串，得到 %q", b.DataVersion)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*PresetBundle){
		"schema 过高":      func(b *PresetBundle) { b.SchemaVersion = CurrentSchemaVersion + 1 },
		"originTypes 空":  func(b *PresetBundle) { b.Subscription.OriginTypes = nil },
		"tier 非法":        func(b *PresetBundle) { b.Subscription.OriginTypes[0].Tier = "platinum" },
		"value 空":        func(b *PresetBundle) { b.Subscription.OriginTypes[0].Value = "" },
		"重复 value":       func(b *PresetBundle) { b.Subscription.OriginTypes[1].Value = "official_api" },
		"billingModes 空": func(b *PresetBundle) { b.Subscription.BillingModes = nil },
		"sources 空":      func(b *PresetBundle) { b.Subscription.Sources = nil },
		"newApi 引用未知类型":  func(b *PresetBundle) { b.Subscription.NewApiDefaults.OriginType = "ghost" },
		"别名目标未知":         func(b *PresetBundle) { b.Subscription.OriginTypeAliases["x"] = "ghost" },
	}
	for name, corrupt := range cases {
		t.Run(name, func(t *testing.T) {
			b := validBundle()
			corrupt(b)
			if err := Validate(b); err == nil {
				t.Errorf("%s：期望校验失败，却通过了", name)
			}
		})
	}
}

func TestValidateAcceptsOlderSchema(t *testing.T) {
	b := validBundle()
	b.SchemaVersion = CurrentSchemaVersion - 1 // 更旧 schema 仍可读
	if CurrentSchemaVersion-1 >= 0 {
		if err := Validate(b); err != nil {
			t.Errorf("更旧 schema 应可接受，得到: %v", err)
		}
	}
}

func TestValidateAcceptsBuiltinManifestOpenAIServiceType(t *testing.T) {
	b := validBundle()
	b.BuiltinModelsManifests = &BuiltinModelsManifestPreset{
		SchemaVersion: 1,
		Manifests: []BuiltinModelsManifestEntryPreset{
			{
				BaseURLPattern: "api.example.com/v1",
				ServiceType:    "openai",
				ModelIDs:       []string{"model-a"},
			},
		},
	}
	if err := Validate(b); err != nil {
		t.Fatalf("openai serviceType 应可用于 builtin manifest: %v", err)
	}
}

func TestTierForAndCanonicalize(t *testing.T) {
	sub := validBundle().Subscription
	if got := sub.TierFor("relay"); got != "second" {
		t.Errorf("relay tier=%q，期望 second", got)
	}
	// 别名归一化：public_benefit -> community -> third
	if got := sub.TierFor("public_benefit"); got != "third" {
		t.Errorf("public_benefit tier=%q，期望 third（经别名归一化）", got)
	}
	if got := sub.Canonicalize("public_benefit"); got != "community" {
		t.Errorf("Canonicalize(public_benefit)=%q，期望 community", got)
	}
	if got := sub.TierFor("nonexistent"); got != "unknown" {
		t.Errorf("未知类型 tier=%q，期望 unknown", got)
	}
}

func TestSupportsAutoRefresh(t *testing.T) {
	sub := validBundle().Subscription
	if !sub.SupportsAutoRefresh("openai") {
		t.Error("openai 应支持自动刷新")
	}
	if sub.SupportsAutoRefresh("relay_x") {
		t.Error("relay_x 不应支持自动刷新")
	}
}

func TestStoreSwapAndObserver(t *testing.T) {
	s := NewPresetStore(nil) // 回退内置
	if s.DataVersion() != "" {
		t.Errorf("初始应为内置（DataVersion 空），得到 %q", s.DataVersion())
	}

	var mu sync.Mutex
	var notified string
	s.RegisterOnChange(func(b *PresetBundle) {
		mu.Lock()
		notified = b.DataVersion
		mu.Unlock()
	})

	next := validBundle()
	s.Swap(next)

	if s.DataVersion() != "2026.07.10-1" {
		t.Errorf("Swap 后 DataVersion=%q，期望 2026.07.10-1", s.DataVersion())
	}
	mu.Lock()
	if notified != "2026.07.10-1" {
		t.Errorf("观察者应收到新版本，得到 %q", notified)
	}
	mu.Unlock()

	// nil swap 应被忽略，不改变当前值
	s.Swap(nil)
	if s.DataVersion() != "2026.07.10-1" {
		t.Errorf("nil Swap 后 DataVersion 应不变，得到 %q", s.DataVersion())
	}
}

func TestStoreGetReturnsImmutableClone(t *testing.T) {
	s := NewPresetStore(validBundle())
	got := s.Get()
	got.DataVersion = "mutated"
	got.Subscription.OriginTypes[0].Value = "mutated"

	again := s.Get()
	if again.DataVersion != "2026.07.10-1" {
		t.Fatalf("DataVersion = %q, want 2026.07.10-1", again.DataVersion)
	}
	if again.Subscription.OriginTypes[0].Value != "official_api" {
		t.Fatalf("OriginTypes[0].Value = %q, want official_api", again.Subscription.OriginTypes[0].Value)
	}
}

func TestStoreObserverGetsImmutableClone(t *testing.T) {
	s := NewPresetStore(validBundle())
	var first *PresetBundle
	s.RegisterOnChange(func(b *PresetBundle) {
		first = b
		b.DataVersion = "observer-mutated"
	})
	next := validBundle()
	next.DataVersion = "2026.07.10-2"
	s.Swap(next)
	if first == nil {
		t.Fatal("observer not invoked")
	}
	if s.DataVersion() != "2026.07.10-2" {
		t.Fatalf("store DataVersion = %q, want 2026.07.10-2", s.DataVersion())
	}
}
