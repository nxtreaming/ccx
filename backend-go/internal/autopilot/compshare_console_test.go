package autopilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testCompshareCookie = "U_USER_EMAIL=test_user%40console.compshare.cn; U_CSRF_TOKEN=test-csrf; c_project_test_user_console_compshare_cn={%22ProjectId%22:%22org-test%22%2C%22ProjectName%22:%22Default%22}; U_JWT_TOKEN=test-session"

func TestCompshareConsoleClientVerify(t *testing.T) {
	const currentKey = "sk-cp-test-current"
	now := time.Date(2026, 7, 20, 13, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Query().Get("Action") != compshareConsoleAction {
			t.Fatalf("请求地址错误: %s %s", r.Method, r.URL.String())
		}
		if r.Header.Get("Cookie") != testCompshareCookie || r.Header.Get("U-CSRF-Token") != "test-csrf" {
			t.Fatalf("控制台凭证请求头错误")
		}
		if r.Header.Get("Origin") != compshareConsoleOrigin || r.Header.Get("Referer") != compshareConsoleReferer {
			t.Fatalf("控制台来源请求头错误")
		}
		var request compshareConsoleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.ProjectID != "org-test" || request.User != "test_user@console.compshare.cn" || request.Timestamp != now.UnixMilli() {
			t.Fatalf("请求体错误: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "RetCode": 0,
  "Message": "success",
  "UserPlans": [{
    "Code": "cppkg-test",
    "PlanCode": "cp-plan-test",
    "PlanName": "Pro",
    "DisplayName": "Pro 增强版",
    "LimitPer5h": 3000,
    "LimitPerWeek": 7500,
    "LimitPerMonth": 19000,
    "ConcurrencyLimit": 10,
    "UsagePer5h": 120,
    "UsagePerWeek": 2300,
    "UsagePerMonth": 6496,
    "UsagePer5hUpdatedAt": 1784512800,
    "UsagePerWeekUpdatedAt": 1784477100,
    "UsagePerMonthUpdatedAt": 1784400000,
    "UsagePer5hNextResetAt": 1784530800,
    "UsagePerWeekNextResetAt": 1785081600,
    "UsagePerMonthNextResetAt": 1785037981,
    "Status": 1,
    "IsTeam": false,
    "ExpireAt": 1785037981,
    "Keys": [{"APIKey": "sk-cp-other"}, {"APIKey": "` + currentKey + `"}]
  }],
  "InvalidUserPlans": []
}`))
	}))
	defer server.Close()

	client := &CompshareConsoleClient{
		HTTPClient: server.Client(), BaseURL: server.URL, Now: func() time.Time { return now },
	}
	snapshot, err := client.Verify(context.Background(), "Cookie: "+testCompshareCookie, currentKey)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Cookie != testCompshareCookie || snapshot.PlanCode != "cp-plan-test" || snapshot.DisplayName != "Pro 增强版" {
		t.Fatalf("套餐快照错误: %+v", snapshot)
	}
	if snapshot.ConcurrencyLimit != 10 || snapshot.FiveHourUsage.Used != 120 || snapshot.MonthlyUsage.Limit != 19000 {
		t.Fatalf("套餐用量错误: %+v", snapshot)
	}
	if snapshot.ValidatedAt != now {
		t.Fatalf("ValidatedAt=%v", snapshot.ValidatedAt)
	}
}

func TestCompshareConsoleClientRejectsIncompleteCookie(t *testing.T) {
	tests := []string{
		"U_CSRF_TOKEN=test-csrf; c_project_test={%22ProjectId%22:%22org-test%22}",
		"U_USER_EMAIL=test%40example.com; c_project_test_example_com={%22ProjectId%22:%22org-test%22}",
		"U_USER_EMAIL=test%40example.com; U_CSRF_TOKEN=test-csrf",
	}
	for _, cookie := range tests {
		if _, err := (&CompshareConsoleClient{}).Verify(context.Background(), cookie, "sk-cp-test"); err == nil {
			t.Fatalf("不完整 Cookie 应被拒绝: %s", cookie)
		}
	}
}

func TestCompshareConsoleClientRequiresCurrentKeyInPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"RetCode":0,"UserPlans":[{"PlanCode":"test","Keys":[{"APIKey":"sk-cp-other"}]}]}`))
	}))
	defer server.Close()
	client := &CompshareConsoleClient{HTTPClient: server.Client(), BaseURL: server.URL}
	_, err := client.Verify(context.Background(), testCompshareCookie, "sk-cp-current")
	if err == nil || !strings.Contains(err.Error(), "未找到当前托管 Key") {
		t.Fatalf("应拒绝不属于当前套餐的 Key: %v", err)
	}
}

func TestParseCompshareConsoleSessionRejectsAmbiguousProjects(t *testing.T) {
	cookie := "U_USER_EMAIL=missing%40example.com; U_CSRF_TOKEN=test-csrf; " +
		"c_project_first={%22ProjectId%22:%22org-first%22}; c_project_second={%22ProjectId%22:%22org-second%22}"
	_, err := parseCompshareConsoleSession(cookie)
	if err == nil || !strings.Contains(err.Error(), "多个账号") {
		t.Fatalf("应拒绝歧义项目 Cookie: %v", err)
	}
}
