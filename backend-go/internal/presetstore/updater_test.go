package presetstore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestPresetUpdaterCheckOnceUpdatesStoreAndCache(t *testing.T) {
	cacheDir := t.TempDir()
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()

	server := newPresetTestServer(t, bundle, presetTestServerOptions{})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		CacheDir:                cacheDir,
		AllowInsecureForTesting: true,
	})

	if err := updater.CheckOnce(context.Background()); err != nil {
		t.Fatalf("CheckOnce() error = %v", err)
	}
	if store.DataVersion() != bundle.DataVersion {
		t.Fatalf("store version = %q, want %q", store.DataVersion(), bundle.DataVersion)
	}
	if updater.Status().Source != "remote" {
		t.Fatalf("status source = %q, want remote", updater.Status().Source)
	}
	loaded, err := LoadCache(cacheDir)
	if err != nil {
		t.Fatalf("LoadCache() error = %v", err)
	}
	if loaded.DataVersion != bundle.DataVersion {
		t.Fatalf("cached version = %q, want %q", loaded.DataVersion, bundle.DataVersion)
	}
}

func TestPresetUpdaterCheckOnceRejectsTamperedShard(t *testing.T) {
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()
	server := newPresetTestServer(t, bundle, presetTestServerOptions{tamperHash: true})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		AllowInsecureForTesting: true,
	})

	if err := updater.CheckOnce(context.Background()); err == nil {
		t.Fatal("CheckOnce() error = nil, want hash mismatch")
	}
	if store.DataVersion() != EmbeddedBundle().DataVersion {
		t.Fatalf("store version = %q, want embedded version %q", store.DataVersion(), EmbeddedBundle().DataVersion)
	}
}

func TestPresetUpdaterCheckOnceRejectsHigherSchema(t *testing.T) {
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()
	server := newPresetTestServer(t, bundle, presetTestServerOptions{higherSchema: true})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		AllowInsecureForTesting: true,
	})

	if err := updater.CheckOnce(context.Background()); err == nil {
		t.Fatal("CheckOnce() error = nil, want higher schema error")
	}
}

func TestPresetUpdaterCheckOnceSkipsOlderVersion(t *testing.T) {
	current := validBundle()
	current.ModelRegistry = validModelRegistryPreset()
	current.ChannelPresets = validChannelPresetsPreset()
	current.BuiltinModelsManifests = validBuiltinManifestPreset()
	store := NewPresetStore(current)

	incoming := cloneBundle(current)
	incoming.DataVersion = "2026.07.10-0"
	server := newPresetTestServer(t, incoming, presetTestServerOptions{})
	defer server.Close()

	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		AllowInsecureForTesting: true,
	})

	if err := updater.CheckOnce(context.Background()); err != nil {
		t.Fatalf("CheckOnce() error = %v", err)
	}
	if store.DataVersion() != "2026.07.10-1" {
		t.Fatalf("store version = %q, want unchanged 2026.07.10-1", store.DataVersion())
	}
}

func TestPresetUpdaterCheckOnceRejectsLegacySubscriptionOnlyIndex(t *testing.T) {
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	subscriptionBytes, err := json.Marshal(bundle.Subscription)
	if err != nil {
		t.Fatalf("json.Marshal(subscription) error = %v", err)
	}
	index := PresetIndex{
		SchemaVersion: bundle.SchemaVersion,
		DataVersion:   bundle.DataVersion,
		Shards: []PresetIndexShard{{
			Kind:   "subscription",
			URL:    "./subscription.json",
			SHA256: sha256Hex(subscriptionBytes),
		}},
	}
	indexBytes, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("json.Marshal(index) error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			_, _ = w.Write(indexBytes)
		case "/subscription.json":
			_, _ = w.Write(subscriptionBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		AllowInsecureForTesting: true,
	})

	if err := updater.CheckOnce(context.Background()); err == nil {
		t.Fatal("CheckOnce() error = nil, want legacy subscription rejection")
	}
	if store.DataVersion() != EmbeddedBundle().DataVersion {
		t.Fatalf("store version = %q, want embedded version %q", store.DataVersion(), EmbeddedBundle().DataVersion)
	}
}

func TestPresetUpdaterCheckOnceRequiresAllRemoteShards(t *testing.T) {
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()
	server := newPresetTestServer(t, bundle, presetTestServerOptions{omitKinds: map[string]bool{"builtinManifest": true}})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		AllowInsecureForTesting: true,
	})

	if err := updater.CheckOnce(context.Background()); err == nil {
		t.Fatal("CheckOnce() error = nil, want missing shard error")
	}
}

func TestPresetUpdaterLoadCacheAtStartupFallbackOnCorruption(t *testing.T) {
	cacheDir := t.TempDir()
	bundle := validBundle()
	if err := SaveCache(cacheDir, bundle); err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}
	bundlePath := filepath.Join(cacheDir, bundleFileName)
	if err := os.WriteFile(bundlePath, []byte("broken"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{CacheDir: cacheDir})
	if err := updater.LoadCacheAtStartup(); err == nil {
		t.Fatal("LoadCacheAtStartup() error = nil, want corruption error")
	}
	if updater.Status().CacheValid {
		t.Fatal("CacheValid = true, want false")
	}
	if store.DataVersion() != EmbeddedBundle().DataVersion {
		t.Fatalf("store version = %q, want embedded version %q", store.DataVersion(), EmbeddedBundle().DataVersion)
	}
}

func TestPresetUpdaterStopIsIdempotent(t *testing.T) {
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()
	server := newPresetTestServer(t, bundle, presetTestServerOptions{})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                20 * time.Millisecond,
		AllowInsecureForTesting: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updater.Start(ctx)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if updater.Status().LastCheckAt != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if updater.Status().LastCheckAt == nil {
		t.Fatal("updater did not perform initial check")
	}

	updater.Stop()
	updater.Stop()
	if updater.Status().Running {
		t.Fatal("Running = true, want false")
	}
}

func TestPresetUpdaterCheckOnceSerializesConcurrentRuns(t *testing.T) {
	cacheDir := t.TempDir()
	bundle := validBundle()
	bundle.DataVersion = "v2.10.0"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()

	release := make(chan struct{})
	server := newPresetTestServer(t, bundle, presetTestServerOptions{waitOnShardPath: "/subscription.json", release: release})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                time.Minute,
		CacheDir:                cacheDir,
		AllowInsecureForTesting: true,
	})

	errCh := make(chan error, 2)
	go func() { errCh <- updater.CheckOnce(context.Background()) }()
	go func() { errCh <- updater.CheckOnce(context.Background()) }()

	time.Sleep(50 * time.Millisecond)
	if !updater.Status().Checking {
		t.Fatal("Checking = false, want true while first run is blocked")
	}
	close(release)

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("CheckOnce() concurrent error = %v", err)
		}
	}
	if store.DataVersion() != bundle.DataVersion {
		t.Fatalf("store version = %q, want %q", store.DataVersion(), bundle.DataVersion)
	}
}

func TestPresetUpdaterStartStopCanRestart(t *testing.T) {
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-2"
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()
	server := newPresetTestServer(t, bundle, presetTestServerOptions{})
	defer server.Close()

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{
		Enabled:                 true,
		IndexURL:                server.URL + "/index.json",
		Interval:                15 * time.Millisecond,
		AllowInsecureForTesting: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updater.Start(ctx)
	updater.Stop()
	updater.Start(ctx)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if updater.Status().LastCheckAt != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if updater.Status().LastCheckAt == nil {
		t.Fatal("restart did not trigger a fresh check")
	}
	updater.Stop()
}

func TestPresetUpdaterCheckOnceRejectsCrossOriginRedirect(t *testing.T) {
	redirectTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{}"))
	}))
	defer redirectTarget.Close()

	redirector := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusFound)
	}))
	defer redirector.Close()

	updater := NewPresetUpdater(NewPresetStore(nil), UpdaterConfig{
		Enabled:    true,
		IndexURL:   redirector.URL,
		Interval:   time.Minute,
		HTTPClient: redirector.Client(),
	})

	if _, err := updater.fetchBytes(context.Background(), redirector.URL, 0); err == nil {
		t.Fatal("fetchBytes() error = nil, want cross-origin redirect rejection")
	}
}

func TestPresetUpdaterCheckOnceRejectsRedirectSchemeDowngrade(t *testing.T) {
	redirector := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com", http.StatusFound)
	}))
	defer redirector.Close()

	updater := NewPresetUpdater(NewPresetStore(nil), UpdaterConfig{
		Enabled:    true,
		IndexURL:   redirector.URL,
		Interval:   time.Minute,
		HTTPClient: redirector.Client(),
	})

	if _, err := updater.fetchBytes(context.Background(), redirector.URL, 0); err == nil {
		t.Fatal("fetchBytes() error = nil, want redirect scheme rejection")
	}
}

func TestStatusHandlerNilUpdater(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/presets/status", StatusHandler(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/presets/status", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("状态码=%d，期望 200", w.Code)
	}
	var status UpdaterStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("解析状态响应失败: %v", err)
	}
	if status.Enabled || status.Source != "embedded" {
		t.Fatalf("nil updater 状态=%+v，期望 disabled/embedded", status)
	}
}

type presetTestServerOptions struct {
	tamperHash      bool
	higherSchema    bool
	omitKinds       map[string]bool
	waitOnShardPath string
	release         <-chan struct{}
}

func newPresetTestServer(t *testing.T, bundle *PresetBundle, opts presetTestServerOptions) *httptest.Server {
	t.Helper()

	type shardFile struct {
		kind string
		path string
		body []byte
	}

	files := []shardFile{{kind: "subscriptionPreset", path: "/subscription.json", body: mustJSON(t, bundle.Subscription)}}
	if bundle.ModelRegistry != nil {
		files = append(files, shardFile{kind: "modelRegistry", path: "/model-registry.json", body: mustJSON(t, bundle.ModelRegistry)})
	}
	if bundle.ChannelPresets != nil {
		files = append(files, shardFile{kind: "channelPresets", path: "/channel-presets.json", body: mustJSON(t, bundle.ChannelPresets)})
	}
	if bundle.BuiltinModelsManifests != nil {
		files = append(files, shardFile{kind: "builtinManifest", path: "/builtin-manifest.json", body: mustJSON(t, bundle.BuiltinModelsManifests)})
	}

	shards := make([]PresetIndexShard, 0, len(files))
	fileMap := make(map[string][]byte, len(files))
	for _, file := range files {
		if opts.omitKinds != nil && opts.omitKinds[file.kind] {
			continue
		}
		fileMap[file.path] = file.body
		shards = append(shards, PresetIndexShard{Kind: file.kind, URL: "." + file.path, SHA256: sha256Hex(file.body)})
	}

	index := PresetIndex{
		SchemaVersion: bundle.SchemaVersion,
		DataVersion:   bundle.DataVersion,
		PublishedAt:   time.Now().UTC(),
		Shards:        shards,
	}
	if opts.tamperHash && len(index.Shards) > 0 {
		index.Shards[0].SHA256 = "deadbeef"
	}
	if opts.higherSchema {
		index.SchemaVersion = CurrentSchemaVersion + 1
	}
	indexBytes := mustJSON(t, index)

	var indexHits atomic.Int32
	var shardHits atomic.Int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			indexHits.Add(1)
			_, _ = w.Write(indexBytes)
		default:
			if body, ok := fileMap[r.URL.Path]; ok {
				if opts.waitOnShardPath == r.URL.Path && opts.release != nil {
					<-opts.release
				}
				shardHits.Add(1)
				_, _ = w.Write(body)
				return
			}
			http.NotFound(w, r)
		}
	}))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func TestCompareDataVersion(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal", a: "2026.07.10-1", b: "2026.07.10-1", want: 0},
		{name: "empty old", a: "2026.07.10-1", b: "", want: 1},
		{name: "older", a: "2026.07.10-0", b: "2026.07.10-1", want: -1},
		{name: "semantic numeric", a: "2.10.0", b: "2.9.37", want: 1},
		{name: "semantic reverse", a: "2.9.37", b: "2.10.0", want: -1},
		{name: "generated data version", a: "v2.9.37+20260721", b: "v2.9.37+20260718", want: 1},
		{name: "generated minor version", a: "v2.10.0+20260701", b: "v2.9.37+20260721", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareDataVersion(tt.a, tt.b); got != tt.want {
				t.Fatalf("compareDataVersion(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestPresetUpdaterLoadCacheAtStartupKeepsNewerEmbeddedBundle(t *testing.T) {
	cacheDir := t.TempDir()
	bundle := validBundle()
	bundle.DataVersion = "v0.0.1+19700101"
	if err := SaveCache(cacheDir, bundle); err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}

	store := NewPresetStore(nil)
	embeddedVersion := store.DataVersion()
	updater := NewPresetUpdater(store, UpdaterConfig{CacheDir: cacheDir})
	if err := updater.LoadCacheAtStartup(); err != nil {
		t.Fatalf("LoadCacheAtStartup() error = %v", err)
	}
	if store.DataVersion() != embeddedVersion {
		t.Fatalf("store version = %q, want newer embedded version %q", store.DataVersion(), embeddedVersion)
	}
	status := updater.Status()
	if status.Source != "embedded" || !status.CacheValid {
		t.Fatalf("status = %+v, want embedded source with valid stale cache", status)
	}
}

func TestPresetUpdaterLoadCacheAtStartupSetsCacheSource(t *testing.T) {
	cacheDir := t.TempDir()
	bundle := validBundle()
	bundle.DataVersion = "2026.07.10-9"
	if err := SaveCache(cacheDir, bundle); err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}

	store := NewPresetStore(nil)
	updater := NewPresetUpdater(store, UpdaterConfig{CacheDir: cacheDir})
	if err := updater.LoadCacheAtStartup(); err != nil {
		t.Fatalf("LoadCacheAtStartup() error = %v", err)
	}
	status := updater.Status()
	if status.Source != "cache" {
		t.Fatalf("source = %q, want cache", status.Source)
	}
	if status.DataVersion != bundle.DataVersion {
		t.Fatalf("dataVersion = %q, want %q", status.DataVersion, bundle.DataVersion)
	}
}

func TestPresetUpdaterCheckOnceRequiresHTTPSByDefault(t *testing.T) {
	bundle := validBundle()
	bundle.ModelRegistry = validModelRegistryPreset()
	bundle.ChannelPresets = validChannelPresetsPreset()
	bundle.BuiltinModelsManifests = validBuiltinManifestPreset()
	server := newPresetTestServer(t, bundle, presetTestServerOptions{})
	defer server.Close()

	updater := NewPresetUpdater(NewPresetStore(nil), UpdaterConfig{
		Enabled:  true,
		IndexURL: server.URL + "/index.json",
		Interval: time.Minute,
	})
	if err := updater.CheckOnce(context.Background()); err == nil {
		t.Fatal("CheckOnce() error = nil, want HTTPS validation error")
	}
}

func TestPresetUpdaterStartDisabledIsNoop(t *testing.T) {
	updater := NewPresetUpdater(NewPresetStore(nil), UpdaterConfig{Enabled: false})
	updater.Start(context.Background())
	if updater.Status().Running {
		t.Fatal("Running = true, want false")
	}
}

func validModelRegistryPreset() *ModelRegistryPreset {
	return &ModelRegistryPreset{
		SchemaVersion: 1,
		PricingUnit:   "per_1m_tokens",
		UpstreamCapabilities: []ModelRegistryCapabilityPreset{{
			Patterns: []string{"claude-sonnet-5"},
		}},
	}
}

func validChannelPresetsPreset() *ChannelPresetsPreset {
	return &ChannelPresetsPreset{
		SchemaVersion: 1,
		Collections: map[string]json.RawMessage{
			"claudeMessages": json.RawMessage(`{"schemaVersion":1,"items":[]}`),
			"openAIChat":     json.RawMessage(`{"schemaVersion":1,"items":[]}`),
			"codexResponses": json.RawMessage(`{"schemaVersion":1,"items":[]}`),
			"openAIMessages": json.RawMessage(`{"schemaVersion":1,"items":[]}`),
		},
	}
}

func validBuiltinManifestPreset() *BuiltinModelsManifestPreset {
	return &BuiltinModelsManifestPreset{
		SchemaVersion: 1,
		Manifests: []BuiltinModelsManifestEntryPreset{{
			BaseURLPattern: "api.example.com",
			ServiceType:    "messages",
			ModelIDs:       []string{"claude-sonnet-5"},
		}},
	}
}

func ExamplePresetUpdater_Status() {
	updater := NewPresetUpdater(NewPresetStore(nil), UpdaterConfig{Enabled: true})
	fmt.Println(updater.Status().Enabled)
	// Output: true
}
