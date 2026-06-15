package images

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func newImagesTestConfigManager(t *testing.T) *config.ConfigManager {
	t.Helper()
	cfgFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(cfgFile, []byte(`{"upstream":[],"imagesUpstream":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgManager, err := config.NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	return cfgManager
}

func newImagesTestEnvConfig() *config.EnvConfig {
	envCfg := config.NewEnvConfig()
	envCfg.ProxyAccessKey = "test-key"
	return envCfg
}

func captureImagesLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	oldWriter := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
	})
	return &buf
}

func TestBuildProviderRequest_URLVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"image-default","prompt":"hello"}`))

	upstream := &config.UpstreamConfig{ServiceType: "openai"}
	req, err := buildProviderRequest(c, upstream, "https://api.openai.com", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err != nil {
		t.Fatalf("buildProviderRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/v1/images/generations" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}

	req, err = buildProviderRequest(c, upstream, "https://api.openai.com#", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err != nil {
		t.Fatalf("buildProviderRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/images/generations" {
		t.Fatalf("unexpected # url: %s", req.URL.String())
	}

	req, err = buildProviderRequest(c, upstream, "https://api.openai.com/#", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err != nil {
		t.Fatalf("buildProviderRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/images/generations" {
		t.Fatalf("unexpected /# url: %s", req.URL.String())
	}
}

func TestBuildProviderRequest_RejectsUnsupportedServiceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logBuf := captureImagesLogs(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"image-default","prompt":"hello"}`))

	upstream := &config.UpstreamConfig{ServiceType: "gemini"}
	_, err := buildProviderRequest(c, upstream, "https://api.openai.com", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err == nil {
		t.Fatal("expected error for unsupported serviceType")
	}
	if !strings.Contains(err.Error(), "仅支持 openai serviceType") {
		t.Fatalf("unexpected error: %v", err)
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "[Images-BuildRequest]") {
		t.Fatalf("expected build request log, got: %s", logs)
	}
	if !strings.Contains(logs, "reason=invalid_service_type") {
		t.Fatalf("expected invalid_service_type log, got: %s", logs)
	}
	if strings.Contains(logs, "sk-test") {
		t.Fatalf("expected API key to be masked in logs, got: %s", logs)
	}
}

func TestAddUpstream_RejectsUnsupportedServiceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(cfgFile, []byte(`{"upstream":[],"imagesUpstream":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgManager, err := config.NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.POST("/api/images/channels", AddUpstream(cfgManager))

	body := strings.NewReader(`{"name":"images-gemini","serviceType":"gemini","baseUrl":"https://example.com","apiKeys":["test-key"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/images/channels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Images 渠道仅支持 openai serviceType") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestHandler_MissingModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(cfgFile, []byte(`{"upstream":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgManager, err := config.NewConfigManager(cfgFile, "")
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	defer cfgManager.Close()

	envCfg := config.NewEnvConfig()
	envCfg.ProxyAccessKey = "test-key"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"prompt":"hello"}`))
	c.Request.Header.Set("Authorization", "Bearer test-key")
	Handler(envCfg, cfgManager, nil)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_MissingPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgManager := newImagesTestConfigManager(t)
	defer cfgManager.Close()

	envCfg := newImagesTestEnvConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1"}`))
	c.Request.Header.Set("Authorization", "Bearer test-key")
	Handler(envCfg, cfgManager, nil)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_InvalidMultipartEditsReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgManager := newImagesTestConfigManager(t)
	defer cfgManager.Close()

	envCfg := newImagesTestEnvConfig()
	logBuf := captureImagesLogs(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", strings.NewReader("broken"))
	c.Request.Header.Set("Authorization", "Bearer test-key")
	c.Request.Header.Set("Content-Type", "multipart/form-data")
	Handler(envCfg, cfgManager, nil)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid_multipart") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "[Images-Multipart]") {
		t.Fatalf("expected multipart diagnostic log, got: %s", logs)
	}
	if !strings.Contains(logs, "operation=edits") {
		t.Fatalf("expected operation in logs, got: %s", logs)
	}
	if !strings.Contains(logs, "reason=missing_boundary") {
		t.Fatalf("expected missing_boundary in logs, got: %s", logs)
	}
	if strings.Contains(logs, "broken") {
		t.Fatalf("expected multipart body content to stay out of logs, got: %s", logs)
	}
}

func TestShouldConvertImageURLToB64JSON(t *testing.T) {
	upstream := &config.UpstreamConfig{ConvertImageURLToB64JSON: true}

	if !shouldConvertImageURLToB64JSON(upstream, []byte(`{"response_format":"b64_json"}`), "application/json") {
		t.Fatal("expected conversion for b64_json request")
	}
	if shouldConvertImageURLToB64JSON(upstream, []byte(`{"response_format":"url"}`), "application/json") {
		t.Fatal("did not expect conversion for url response_format")
	}
	if shouldConvertImageURLToB64JSON(&config.UpstreamConfig{}, []byte(`{"response_format":"b64_json"}`), "application/json") {
		t.Fatal("did not expect conversion when channel flag is disabled")
	}
	if shouldConvertImageURLToB64JSON(upstream, []byte(`broken`), "application/json") {
		t.Fatal("did not expect conversion for invalid request body")
	}

	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	if err := writer.WriteField("response_format", "b64_json"); err != nil {
		t.Fatalf("write multipart field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	if !shouldConvertImageURLToB64JSON(upstream, multipartBody.Bytes(), writer.FormDataContentType()) {
		t.Fatal("expected conversion for multipart b64_json request")
	}
}

func TestConvertImageURLResponseToB64JSON(t *testing.T) {
	imageBytes := []byte("image-bytes")
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	}))
	defer imageServer.Close()

	respMap := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"url":            imageServer.URL + "/image.png",
				"revised_prompt": "keep",
			},
			map[string]interface{}{
				"url":      imageServer.URL + "/already.png",
				"b64_json": "already",
			},
		},
	}

	body, err := convertImageURLResponseToB64JSON(context.Background(), respMap, &config.UpstreamConfig{})
	if err != nil {
		t.Fatalf("convertImageURLResponseToB64JSON() error = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal converted body: %v", err)
	}
	data := got["data"].([]interface{})
	first := data[0].(map[string]interface{})
	if first["b64_json"] != base64.StdEncoding.EncodeToString(imageBytes) {
		t.Fatalf("b64_json = %v, want encoded image bytes", first["b64_json"])
	}
	if _, exists := first["url"]; exists {
		t.Fatalf("url should be removed after conversion: %#v", first)
	}
	if first["revised_prompt"] != "keep" {
		t.Fatalf("revised_prompt = %v, want keep", first["revised_prompt"])
	}
	second := data[1].(map[string]interface{})
	if second["b64_json"] != "already" {
		t.Fatalf("existing b64_json = %v, want already", second["b64_json"])
	}
}

func TestConvertImageURLResponseToB64JSONRejectsNonImage(t *testing.T) {
	textServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("not image"))
	}))
	defer textServer.Close()

	respMap := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"url": textServer.URL + "/not-image.txt"},
		},
	}

	_, err := convertImageURLResponseToB64JSON(context.Background(), respMap, &config.UpstreamConfig{})
	if err == nil {
		t.Fatal("expected non-image response to fail conversion")
	}
	if !strings.Contains(err.Error(), "non-image") {
		t.Fatalf("unexpected error: %v", err)
	}
}
