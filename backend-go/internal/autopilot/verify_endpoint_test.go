package autopilot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

func TestBuildClaudeProbeURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{"anthropic 无版本后缀补 /v1/messages", "https://api.xiaomimimo.com/anthropic", "https://api.xiaomimimo.com/anthropic/v1/messages"},
		{"已含 /v1 直接拼 /messages", "https://api.deepseek.com/anthropic/v1", "https://api.deepseek.com/anthropic/v1/messages"},
		{"尾部斜杠归一化", "https://api.moonshot.cn/anthropic/", "https://api.moonshot.cn/anthropic/v1/messages"},
		{"# 结尾跳过版本前缀", "https://custom.example.com/relay#", "https://custom.example.com/relay/messages"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildClaudeProbeURL(tc.baseURL); got != tc.want {
				t.Errorf("buildClaudeProbeURL(%q) = %q, want %q", tc.baseURL, got, tc.want)
			}
		})
	}
}

func TestBuildOpenAIChatProbeURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{"无版本后缀补 /v1/chat/completions", "https://api.xiaomimimo.com", "https://api.xiaomimimo.com/v1/chat/completions"},
		{"已含 /v1 直接拼 /chat/completions", "https://api.xiaomimimo.com/v1", "https://api.xiaomimimo.com/v1/chat/completions"},
		{"尾部斜杠归一化", "https://token-plan-cn.xiaomimimo.com/v1/", "https://token-plan-cn.xiaomimimo.com/v1/chat/completions"},
		{"# 结尾跳过版本前缀", "https://custom.example.com/relay#", "https://custom.example.com/relay/chat/completions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildOpenAIChatProbeURL(tc.baseURL); got != tc.want {
				t.Errorf("buildOpenAIChatProbeURL(%q) = %q, want %q", tc.baseURL, got, tc.want)
			}
		})
	}
}

func TestVerifyClaudeEndpoint(t *testing.T) {
	cases := []struct {
		name           string
		statusCode     int
		wantOK         bool
		wantAuthFailed bool
	}{
		{"200 鉴权通过", http.StatusOK, true, false},
		{"400 服务可达鉴权通过", http.StatusBadRequest, true, false},
		{"422 服务可达鉴权通过", http.StatusUnprocessableEntity, true, false},
		{"401 鉴权失败", http.StatusUnauthorized, false, true},
		{"403 鉴权失败", http.StatusForbidden, false, true},
		{"404 端点不可用", http.StatusNotFound, false, false},
		{"500 端点不可用", http.StatusInternalServerError, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/v1/messages") {
					t.Errorf("探测路径应以 /v1/messages 结尾，实际 %q", r.URL.Path)
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			res := VerifyClaudeEndpoint(context.Background(), srv.URL, "sk-test", "")
			if res.OK != tc.wantOK {
				t.Errorf("OK = %v, want %v (status %d)", res.OK, tc.wantOK, tc.statusCode)
			}
			if res.AuthFailed != tc.wantAuthFailed {
				t.Errorf("AuthFailed = %v, want %v (status %d)", res.AuthFailed, tc.wantAuthFailed, tc.statusCode)
			}
		})
	}
}

func TestVerifyOpenAIChatEndpoint(t *testing.T) {
	cases := []struct {
		name           string
		statusCode     int
		wantOK         bool
		wantAuthFailed bool
	}{
		{"200 鉴权通过", http.StatusOK, true, false},
		{"400 服务可达鉴权通过", http.StatusBadRequest, true, false},
		{"422 服务可达鉴权通过", http.StatusUnprocessableEntity, true, false},
		{"401 鉴权失败", http.StatusUnauthorized, false, true},
		{"403 鉴权失败", http.StatusForbidden, false, true},
		{"404 端点不可用", http.StatusNotFound, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/v1/chat/completions") {
					t.Errorf("探测路径应以 /v1/chat/completions 结尾，实际 %q", r.URL.Path)
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			res := VerifyOpenAIChatEndpoint(context.Background(), srv.URL, "sk-test", "")
			if res.OK != tc.wantOK {
				t.Errorf("OK = %v, want %v (status %d)", res.OK, tc.wantOK, tc.statusCode)
			}
			if res.AuthFailed != tc.wantAuthFailed {
				t.Errorf("AuthFailed = %v, want %v (status %d)", res.AuthFailed, tc.wantAuthFailed, tc.statusCode)
			}
		})
	}
}

func TestVerifyClaudeEndpointNetworkError(t *testing.T) {
	// 指向一个不可达地址，期望 Err 非空、OK=false
	res := VerifyClaudeEndpoint(context.Background(), "http://127.0.0.1:1/anthropic", "sk-test", "")
	if res.OK {
		t.Error("网络错误时 OK 应为 false")
	}
	if res.Err == nil {
		t.Error("网络错误时 Err 应非空")
	}
}

func TestVerifyProviderKeys(t *testing.T) {
	// okSrv 恒返回 200（鉴权通过），authSrv 恒返回 401（鉴权失败）
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer authSrv.Close()

	t.Run("per-key 按前缀绑定候选端点", func(t *testing.T) {
		tmpl := &config.ProviderTemplate{
			ProviderID:  "test",
			ServiceType: "claude",
			KeyPrefixRules: []config.KeyPrefixRule{
				{Prefix: "sk-", PlanTag: "payg"},
				{Prefix: "tp-", PlanTag: "token_plan"},
			},
			Candidates: []config.ProviderCandidate{
				{BaseURL: okSrv.URL + "/anthropic", PlanTag: "payg", Priority: 0},
				{BaseURL: authSrv.URL + "/anthropic", PlanTag: "token_plan", Priority: 0},
			},
		}
		// sk- key 命中 payg 候选（okSrv，200 通过）；tp- key 命中 token_plan 候选后回退到 payg 候选（okSrv）
		keyConfigs, baseURLs, err := verifyProviderKeys(context.Background(), tmpl, []string{"sk-a", "tp-b"})
		if err != nil {
			t.Fatalf("verifyProviderKeys 意外失败: %v", err)
		}
		if len(keyConfigs) != 2 {
			t.Fatalf("keyConfigs 数量 = %d, want 2", len(keyConfigs))
		}
		wantURL := okSrv.URL + "/anthropic"
		// sk-a 绑定 okSrv
		if keyConfigs[0].Key != "sk-a" || keyConfigs[0].BaseURL != wantURL {
			t.Errorf("keyConfigs[0] = %+v, want key=sk-a baseURL=%s", keyConfigs[0], wantURL)
		}
		// tp-b 首选 token_plan 候选（authSrv 401）失败后回退到 payg 候选（okSrv 200）
		if keyConfigs[1].Key != "tp-b" || keyConfigs[1].BaseURL != wantURL {
			t.Errorf("keyConfigs[1] = %+v, want key=tp-b baseURL=%s", keyConfigs[1], wantURL)
		}
		// 两 key 均绑定 okSrv，去重后仅 1 个渠道级 baseURL
		if len(baseURLs) != 1 || baseURLs[0] != wantURL {
			t.Errorf("baseURLs = %v, want [%s]", baseURLs, wantURL)
		}
	})

	t.Run("全部候选鉴权失败时报错", func(t *testing.T) {
		tmpl := &config.ProviderTemplate{
			ProviderID:  "test",
			ServiceType: "claude",
			Candidates: []config.ProviderCandidate{
				{BaseURL: authSrv.URL + "/anthropic", Priority: 0},
			},
		}
		_, _, err := verifyProviderKeys(context.Background(), tmpl, []string{"sk-bad"})
		if err == nil {
			t.Fatal("所有候选鉴权失败时应返回错误")
		}
	})

	t.Run("openai route 使用 chat completions 探测", func(t *testing.T) {
		var gotPath string
		chatSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer chatSrv.Close()

		tmpl := &config.ProviderTemplate{ProviderID: "x"}
		route := config.ProviderRoute{
			ChannelKind: "chat",
			ServiceType: "openai",
			Candidates:  []config.ProviderCandidate{{BaseURL: chatSrv.URL + "/v1", Priority: 0}},
		}
		keyConfigs, baseURLs, err := verifyProviderRouteKeys(context.Background(), tmpl, route, []string{"sk-a"})
		if err != nil {
			t.Fatalf("openai route 应支持模板化验证: %v", err)
		}
		if gotPath != "/v1/chat/completions" {
			t.Fatalf("openai route 探测路径=%q, want /v1/chat/completions", gotPath)
		}
		if len(keyConfigs) != 1 || len(baseURLs) != 1 || baseURLs[0] != chatSrv.URL+"/v1" {
			t.Fatalf("验证结果不符合预期: keyConfigs=%+v baseURLs=%v", keyConfigs, baseURLs)
		}
	})

	t.Run("不支持的 serviceType 拒绝", func(t *testing.T) {
		tmpl := &config.ProviderTemplate{ProviderID: "x"}
		route := config.ProviderRoute{ChannelKind: "gemini", ServiceType: "gemini"}
		if _, _, err := verifyProviderRouteKeys(context.Background(), tmpl, route, []string{"sk-a"}); err == nil {
			t.Fatal("不支持的 serviceType 应返回错误")
		}
	})
}
