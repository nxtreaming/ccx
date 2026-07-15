package autopilot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

func TestDeepSeekClientFetchBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user/balance" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"cny","total_balance":"110.00","granted_balance":"10.00","topped_up_balance":"100.00"}]}`))
	}))
	defer server.Close()

	client := NewDeepSeekClient(server.Client())
	client.BaseURL = server.URL
	balance, err := client.FetchBalance(t.Context(), "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if !balance.IsAvailable || len(balance.BalanceInfos) != 1 || balance.BalanceInfos[0].Currency != "CNY" || balance.BalanceInfos[0].TotalBalance != "110.00" {
		t.Fatalf("balance = %+v", balance)
	}
}

func TestHandleDeepSeekBalanceMasksCredentialsAndKeepsPartialErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Bearer sk-good-secret":
			_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"2.50","granted_balance":"0.50","topped_up_balance":"2.00"}]}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts":[{"accountUid":"acct_ds","providerId":"deepseek","name":"deepseek","credentials":[
    {"credentialUid":"cred_good","apiKey":"sk-good-secret"},
    {"credentialUid":"cred_bad","apiKey":"sk-bad-secret"}
  ]}],
  "upstream":[{"accountUid":"acct_ds","channelUid":"ch_ds","providerId":"deepseek","name":"deepseek-claude","serviceType":"claude","baseUrl":"https://api.deepseek.com/anthropic","apiKeys":["sk-good-secret","sk-bad-secret"],"apiKeyConfigs":[
    {"credentialUid":"cred_good","key":"sk-good-secret","baseUrl":"https://api.deepseek.com/anthropic"},
    {"credentialUid":"cred_bad","key":"sk-bad-secret","baseUrl":"https://api.deepseek.com/anthropic"}
  ],"autoManaged":true,"status":"active"}],
  "chatUpstream":[],"responsesUpstream":[],"geminiUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := config.NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	client := NewDeepSeekClient(server.Client())
	client.BaseURL = server.URL
	router := setupAutoManagedRouter(&AutoManagedDeps{CfgManager: manager, DeepSeekClient: client})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/accounts/acct_ds/deepseek-balance", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "sk-good-secret") || strings.Contains(recorder.Body.String(), "sk-bad-secret") {
		t.Fatalf("响应泄漏明文 Key: %s", recorder.Body.String())
	}
	var response struct {
		Balances []managedDeepSeekCredentialBalance `json:"balances"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Balances) != 2 || !response.Balances[0].IsAvailable || response.Balances[1].Error == "" {
		t.Fatalf("balances=%+v", response.Balances)
	}
}
