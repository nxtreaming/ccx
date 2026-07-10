package presetstore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/presets", Handler(Default()))
	return r
}

// GET /api/presets 返回完整 bundle 且带 ETag。
func TestHandler_ReturnsBundleWithETag(t *testing.T) {
	r := newTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/presets", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", w.Code)
	}
	if etag := w.Header().Get("ETag"); etag == "" {
		t.Fatal("缺少 ETag 头")
	}

	var bundle PresetBundle
	if err := json.Unmarshal(w.Body.Bytes(), &bundle); err != nil {
		t.Fatalf("响应体解析失败: %v", err)
	}
	if len(bundle.Subscription.OriginTypes) == 0 {
		t.Error("originTypes 为空")
	}
	if bundle.Subscription.NewApiDefaults.OriginType == "" {
		t.Error("newApiDefaults.originType 为空")
	}
}

// If-None-Match 命中当前 ETag 时返回 304。
func TestHandler_NotModifiedOnMatchingETag(t *testing.T) {
	r := newTestRouter()

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/api/presets", nil))
	etag := w1.Header().Get("ETag")

	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/presets", nil)
	req.Header.Set("If-None-Match", etag)
	r.ServeHTTP(w2, req)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("状态码 = %d，期望 304", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Errorf("304 响应体应为空，实际 %d 字节", w2.Body.Len())
	}
}
