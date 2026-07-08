package autopilot

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// ── LocalRuntimeStore CRUD 测试 ──

func newTestLocalRuntimeStore(t *testing.T) *LocalRuntimeStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	store, err := NewLocalRuntimeStoreWithDB(db)
	if err != nil {
		t.Fatalf("创建 LocalRuntimeStore 失败: %v", err)
	}
	t.Cleanup(func() { store.Close(); db.Close() })
	return store
}

func TestLocalRuntimeStore_CRUD(t *testing.T) {
	store := newTestLocalRuntimeStore(t)

	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "Upsert + Get",
			fn: func(t *testing.T) {
				profile := &LocalModelRuntimeProfile{
					RuntimeUID:  "lr_test0001",
					Name:        "本机 Ollama",
					RuntimeType: RuntimeTypeOllama,
					BaseURL:     "http://localhost:11434",
					Status:      LocalRuntimeUnknown,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
				if err := store.Upsert(profile); err != nil {
					t.Fatalf("Upsert 失败: %v", err)
				}

				got := store.Get("lr_test0001")
				if got == nil {
					t.Fatal("Get 返回 nil")
				}
				if got.Name != "本机 Ollama" {
					t.Errorf("Name = %q, 期望 %q", got.Name, "本机 Ollama")
				}
				if got.RuntimeType != RuntimeTypeOllama {
					t.Errorf("RuntimeType = %q, 期望 %q", got.RuntimeType, RuntimeTypeOllama)
				}
				if got.BaseURL != "http://localhost:11434" {
					t.Errorf("BaseURL = %q, 期望 %q", got.BaseURL, "http://localhost:11434")
				}
			},
		},
		{
			name: "Get 不存在",
			fn: func(t *testing.T) {
				got := store.Get("lr_notexist")
				if got != nil {
					t.Errorf("期望 nil，实际 %+v", got)
				}
			},
		},
		{
			name: "ListAll",
			fn: func(t *testing.T) {
				// 再插入一条
				profile := &LocalModelRuntimeProfile{
					RuntimeUID:  "lr_test0002",
					Name:        "LM Studio",
					RuntimeType: RuntimeTypeLMStudio,
					BaseURL:     "http://localhost:1234/v1",
					Status:      LocalRuntimeUnknown,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
				if err := store.Upsert(profile); err != nil {
					t.Fatalf("Upsert 失败: %v", err)
				}

				all := store.ListAll()
				if len(all) < 2 {
					t.Errorf("ListAll 返回 %d 条, 期望 >=2", len(all))
				}
			},
		},
		{
			name: "Update (Upsert 覆盖)",
			fn: func(t *testing.T) {
				profile := store.Get("lr_test0001")
				if profile == nil {
					t.Skip("lr_test0001 不存在")
				}
				profile.Name = "本机 Ollama (已更新)"
				profile.Status = LocalRuntimeHealthy
				profile.DiscoveredModels = []string{"llama3:8b", "qwen2:7b"}
				if err := store.Upsert(profile); err != nil {
					t.Fatalf("Upsert 更新失败: %v", err)
				}

				got := store.Get("lr_test0001")
				if got.Name != "本机 Ollama (已更新)" {
					t.Errorf("更新后 Name = %q, 期望 %q", got.Name, "本机 Ollama (已更新)")
				}
				if got.Status != LocalRuntimeHealthy {
					t.Errorf("更新后 Status = %q, 期望 %q", got.Status, LocalRuntimeHealthy)
				}
				if len(got.DiscoveredModels) != 2 {
					t.Errorf("更新后 DiscoveredModels 长度 = %d, 期望 2", len(got.DiscoveredModels))
				}
			},
		},
		{
			name: "Delete",
			fn: func(t *testing.T) {
				if err := store.Delete("lr_test0001"); err != nil {
					t.Fatalf("Delete 失败: %v", err)
				}
				if got := store.Get("lr_test0001"); got != nil {
					t.Errorf("删除后 Get 仍返回非 nil: %+v", got)
				}
			},
		},
		{
			name: "Delete 不存在",
			fn: func(t *testing.T) {
				// 删除不存在的 ID 不应 panic（数据库会删 0 行）
				err := store.Delete("lr_notexist")
				if err != nil {
					t.Errorf("删除不存在的 ID 不应报错: %v", err)
				}
			},
		},
		{
			name: "Flush 持久化后重新加载",
			fn: func(t *testing.T) {
				// 重新插入并 flush
				profile := &LocalModelRuntimeProfile{
					RuntimeUID:       "lr_flush001",
					Name:             "Flush Test",
					RuntimeType:      RuntimeTypeOpenAICompatible,
					BaseURL:          "http://localhost:8080/v1",
					Status:           LocalRuntimeHealthy,
					DiscoveredModels: []string{"model-a"},
					CreatedAt:        now,
					UpdatedAt:        now,
				}
				if err := store.Upsert(profile); err != nil {
					t.Fatalf("Upsert 失败: %v", err)
				}
				if err := store.Flush(); err != nil {
					t.Fatalf("Flush 失败: %v", err)
				}

				// 直接从数据库验证持久化
				var count int
				err := store.db.QueryRow("SELECT COUNT(*) FROM autopilot_local_runtimes WHERE runtime_uid = ?", "lr_flush001").Scan(&count)
				if err != nil {
					t.Fatalf("查询数据库失败: %v", err)
				}
				if count != 1 {
					t.Errorf("Flush 后数据库记录数 = %d, 期望 1", count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

// ── ProbeRuntime 探测测试 ──

func TestProbeRuntime_Ollama(t *testing.T) {
	// 模拟 ollama /api/tags 接口
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		resp := ollamaTagsResponse{
			Models: []ollamaModel{
				{Name: "llama3:8b"},
				{Name: "qwen2:7b"},
				{Name: "codellama:13b"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	profile := &LocalModelRuntimeProfile{
		RuntimeUID:  "lr_ollama01",
		RuntimeType: RuntimeTypeOllama,
		BaseURL:     mockServer.URL,
	}

	err := ProbeRuntime(context.Background(), profile)
	if err != nil {
		t.Fatalf("ProbeRuntime 失败: %v", err)
	}

	if profile.Status != LocalRuntimeHealthy {
		t.Errorf("Status = %q, 期望 %q", profile.Status, LocalRuntimeHealthy)
	}
	if len(profile.DiscoveredModels) != 3 {
		t.Errorf("DiscoveredModels 长度 = %d, 期望 3", len(profile.DiscoveredModels))
	}
	if profile.DiscoveredModels != nil && profile.DiscoveredModels[0] != "llama3:8b" {
		t.Errorf("DiscoveredModels[0] = %q, 期望 %q", profile.DiscoveredModels[0], "llama3:8b")
	}
	if profile.LatencyMs < 0 {
		t.Errorf("LatencyMs = %d, 期望 >= 0", profile.LatencyMs)
	}
	if profile.LastProbeAt == nil {
		t.Error("LastProbeAt 不应为 nil")
	}
}

func TestProbeRuntime_OpenAICompatible(t *testing.T) {
	// 模拟 openai-compatible /v1/models 接口
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		resp := openAIModelsResponse{
			Data: []openAIModel{
				{ID: "gpt-4o"},
				{ID: "gpt-4o-mini"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	tests := []struct {
		name        string
		runtimeType RuntimeType
	}{
		{"lmstudio", RuntimeTypeLMStudio},
		{"llama_server", RuntimeTypeLlamaServer},
		{"openai_compatible", RuntimeTypeOpenAICompatible},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &LocalModelRuntimeProfile{
				RuntimeUID:  "lr_openai01",
				RuntimeType: tt.runtimeType,
				BaseURL:     mockServer.URL,
			}

			err := ProbeRuntime(context.Background(), profile)
			if err != nil {
				t.Fatalf("ProbeRuntime 失败: %v", err)
			}

			if profile.Status != LocalRuntimeHealthy {
				t.Errorf("Status = %q, 期望 %q", profile.Status, LocalRuntimeHealthy)
			}
			if len(profile.DiscoveredModels) != 2 {
				t.Errorf("DiscoveredModels 长度 = %d, 期望 2", len(profile.DiscoveredModels))
			}
		})
	}
}

func TestProbeRuntime_Unavailable(t *testing.T) {
	// 使用一个已关闭的服务器模拟不可达
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	mockServer.Close() // 立即关闭

	profile := &LocalModelRuntimeProfile{
		RuntimeUID:  "lr_dead0001",
		RuntimeType: RuntimeTypeOllama,
		BaseURL:     mockServer.URL,
	}

	err := ProbeRuntime(context.Background(), profile)
	if err == nil {
		t.Fatal("期望 ProbeRuntime 返回错误，实际为 nil")
	}

	if profile.Status != LocalRuntimeUnavailable {
		t.Errorf("Status = %q, 期望 %q", profile.Status, LocalRuntimeUnavailable)
	}
	if profile.DiscoveredModels != nil {
		t.Errorf("不可达时 DiscoveredModels 应为 nil，实际 %v", profile.DiscoveredModels)
	}
}

func TestProbeRuntime_SlowResponse(t *testing.T) {
	// 模拟慢响应（>2s 触发 slow 状态）
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2100 * time.Millisecond)
		resp := openAIModelsResponse{
			Data: []openAIModel{{ID: "slow-model"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	profile := &LocalModelRuntimeProfile{
		RuntimeUID:  "lr_slow0001",
		RuntimeType: RuntimeTypeOpenAICompatible,
		BaseURL:     mockServer.URL,
	}

	err := ProbeRuntime(context.Background(), profile)
	if err != nil {
		t.Fatalf("ProbeRuntime 失败: %v", err)
	}

	if profile.Status != LocalRuntimeSlow {
		t.Errorf("Status = %q, 期望 %q", profile.Status, LocalRuntimeSlow)
	}
}

func TestProbeRuntime_EmptyBaseURL(t *testing.T) {
	profile := &LocalModelRuntimeProfile{
		RuntimeUID:  "lr_empty01",
		RuntimeType: RuntimeTypeOllama,
		BaseURL:     "",
	}

	err := ProbeRuntime(context.Background(), profile)
	if err == nil {
		t.Fatal("空 BaseURL 应返回错误")
	}
}

func TestProbeRuntime_NonOKStatus(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	profile := &LocalModelRuntimeProfile{
		RuntimeUID:  "lr_50000001",
		RuntimeType: RuntimeTypeOllama,
		BaseURL:     mockServer.URL,
	}

	err := ProbeRuntime(context.Background(), profile)
	if err == nil {
		t.Fatal("500 响应应返回错误")
	}
	if profile.Status != LocalRuntimeUnavailable {
		t.Errorf("Status = %q, 期望 %q", profile.Status, LocalRuntimeUnavailable)
	}
}

// ── 辅助函数测试 ──

func TestGenerateRuntimeUID(t *testing.T) {
	uid := GenerateRuntimeUID()
	if len(uid) < 3 || uid[:3] != "lr_" {
		t.Errorf("GenerateRuntimeUID = %q, 期望 lr_ 前缀", uid)
	}

	// 验证唯一性
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateRuntimeUID()
		if seen[id] {
			t.Fatalf("重复 UID: %q", id)
		}
		seen[id] = true
	}
}

func TestIsValidRuntimeType(t *testing.T) {
	tests := []struct {
		input RuntimeType
		want  bool
	}{
		{RuntimeTypeOllama, true},
		{RuntimeTypeLMStudio, true},
		{RuntimeTypeLlamaServer, true},
		{RuntimeTypeOpenAICompatible, true},
		{"invalid_type", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := IsValidRuntimeType(tt.input)
			if got != tt.want {
				t.Errorf("IsValidRuntimeType(%q) = %v, 期望 %v", tt.input, got, tt.want)
			}
		})
	}
}
