package autopilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/BenedictKing/ccx/internal/config"
)

const (
	compshareConsoleAPIURL       = "https://api.compshare.cn/"
	compshareConsoleAction       = "GetOpenAPIUserPlans"
	compshareConsoleOrigin       = "https://console.compshare.cn"
	compshareConsoleReferer      = "https://console.compshare.cn/light-gpu/model-manage"
	maxCompshareConsoleCookieLen = 32 << 10
	maxCompshareConsoleBodyLen   = 1 << 20
)

type CompshareConsoleClient struct {
	HTTPClient *http.Client
	BaseURL    string
	Now        func() time.Time
}

type compshareConsoleSession struct {
	Cookie    string
	UserEmail string
	CSRFToken string
	ProjectID string
}

type compshareConsoleRequest struct {
	ProjectID string `json:"ProjectId"`
	Action    string `json:"Action"`
	User      string `json:"_user"`
	Timestamp int64  `json:"_timestamp"`
}

type compshareConsoleResponse struct {
	RetCode          int                 `json:"RetCode"`
	Message          string              `json:"Message"`
	UserPlans        []compshareUserPlan `json:"UserPlans"`
	InvalidUserPlans []compshareUserPlan `json:"InvalidUserPlans"`
}

type compshareUserPlan struct {
	Code                     string             `json:"Code"`
	PlanCode                 string             `json:"PlanCode"`
	PlanName                 string             `json:"PlanName"`
	DisplayName              string             `json:"DisplayName"`
	LimitPer5h               int64              `json:"LimitPer5h"`
	LimitPerWeek             int64              `json:"LimitPerWeek"`
	LimitPerMonth            int64              `json:"LimitPerMonth"`
	ConcurrencyLimit         int64              `json:"ConcurrencyLimit"`
	UsagePer5h               int64              `json:"UsagePer5h"`
	UsagePerWeek             int64              `json:"UsagePerWeek"`
	UsagePerMonth            int64              `json:"UsagePerMonth"`
	UsagePer5hUpdatedAt      int64              `json:"UsagePer5hUpdatedAt"`
	UsagePerWeekUpdatedAt    int64              `json:"UsagePerWeekUpdatedAt"`
	UsagePerMonthUpdatedAt   int64              `json:"UsagePerMonthUpdatedAt"`
	UsagePer5hNextResetAt    int64              `json:"UsagePer5hNextResetAt"`
	UsagePerWeekNextResetAt  int64              `json:"UsagePerWeekNextResetAt"`
	UsagePerMonthNextResetAt int64              `json:"UsagePerMonthNextResetAt"`
	Status                   int                `json:"Status"`
	IsTeam                   bool               `json:"IsTeam"`
	ExpireAt                 int64              `json:"ExpireAt"`
	Keys                     []compsharePlanKey `json:"Keys"`
}

type compsharePlanKey struct {
	APIKey string `json:"APIKey"`
}

func (c *CompshareConsoleClient) Verify(ctx context.Context, cookie, currentAPIKey string) (*config.CompshareConsoleCredential, error) {
	session, err := parseCompshareConsoleSession(cookie)
	if err != nil {
		return nil, err
	}
	currentAPIKey = strings.TrimSpace(currentAPIKey)
	if currentAPIKey == "" {
		return nil, fmt.Errorf("当前托管 Key 为空")
	}

	now := time.Now()
	if c != nil && c.Now != nil {
		now = c.Now()
	}
	payload, err := json.Marshal(compshareConsoleRequest{
		ProjectID: session.ProjectID,
		Action:    compshareConsoleAction,
		User:      session.UserEmail,
		Timestamp: now.UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("构造优云智算套餐请求失败: %w", err)
	}

	endpoint, err := c.endpointURL()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("构造优云智算套餐请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", session.Cookie)
	req.Header.Set("Origin", compshareConsoleOrigin)
	req.Header.Set("Referer", compshareConsoleReferer)
	req.Header.Set("U-CSRF-Token", session.CSRFToken)
	req.Header.Set("x-api-lang", "zh_CN")

	client := http.DefaultClient
	if c != nil && c.HTTPClient != nil {
		client = c.HTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询优云智算套餐失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCompshareConsoleBodyLen+1))
	if err != nil {
		return nil, fmt.Errorf("读取优云智算套餐响应失败: %w", err)
	}
	if len(body) > maxCompshareConsoleBodyLen {
		return nil, fmt.Errorf("优云智算套餐响应超过大小限制")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("查询优云智算套餐失败: HTTP %d", resp.StatusCode)
	}

	var result compshareConsoleResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("优云智算套餐响应不是有效 JSON")
	}
	if result.RetCode != 0 {
		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = "未知错误"
		}
		return nil, fmt.Errorf("优云智算套餐接口返回错误 %d: %s", result.RetCode, message)
	}

	plans := append(result.UserPlans, result.InvalidUserPlans...)
	for _, plan := range plans {
		for _, key := range plan.Keys {
			if apiKeysEqual(currentAPIKey, strings.TrimSpace(key.APIKey)) {
				return compsharePlanSnapshot(session.Cookie, plan, now), nil
			}
		}
	}
	return nil, fmt.Errorf("控制台账号的套餐中未找到当前托管 Key，请确认 Cookie 与 Key 属于同一账号")
}

func (c *CompshareConsoleClient) endpointURL() (string, error) {
	baseURL := compshareConsoleAPIURL
	if c != nil && strings.TrimSpace(c.BaseURL) != "" {
		baseURL = strings.TrimSpace(c.BaseURL)
	}
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("优云智算套餐接口地址无效: %w", err)
	}
	query := endpoint.Query()
	query.Set("Action", compshareConsoleAction)
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func compsharePlanSnapshot(cookie string, plan compshareUserPlan, validatedAt time.Time) *config.CompshareConsoleCredential {
	return &config.CompshareConsoleCredential{
		Cookie:           cookie,
		UserPlanCode:     plan.Code,
		PlanCode:         plan.PlanCode,
		PlanName:         plan.PlanName,
		DisplayName:      plan.DisplayName,
		Status:           plan.Status,
		ConcurrencyLimit: plan.ConcurrencyLimit,
		IsTeam:           plan.IsTeam,
		ExpireAt:         plan.ExpireAt,
		FiveHourUsage: config.CompsharePlanUsageWindow{
			Used: plan.UsagePer5h, Limit: plan.LimitPer5h,
			UpdatedAt: plan.UsagePer5hUpdatedAt, NextResetAt: plan.UsagePer5hNextResetAt,
		},
		WeeklyUsage: config.CompsharePlanUsageWindow{
			Used: plan.UsagePerWeek, Limit: plan.LimitPerWeek,
			UpdatedAt: plan.UsagePerWeekUpdatedAt, NextResetAt: plan.UsagePerWeekNextResetAt,
		},
		MonthlyUsage: config.CompsharePlanUsageWindow{
			Used: plan.UsagePerMonth, Limit: plan.LimitPerMonth,
			UpdatedAt: plan.UsagePerMonthUpdatedAt, NextResetAt: plan.UsagePerMonthNextResetAt,
		},
		ValidatedAt: validatedAt,
	}
}

func parseCompshareConsoleSession(rawCookie string) (compshareConsoleSession, error) {
	cookie := strings.TrimSpace(rawCookie)
	if strings.HasPrefix(strings.ToLower(cookie), "cookie:") {
		cookie = strings.TrimSpace(cookie[len("cookie:"):])
	}
	if cookie == "" {
		return compshareConsoleSession{}, fmt.Errorf("Cookie 不能为空")
	}
	if len(cookie) > maxCompshareConsoleCookieLen {
		return compshareConsoleSession{}, fmt.Errorf("Cookie 长度超过限制")
	}
	if strings.ContainsAny(cookie, "\r\n") {
		return compshareConsoleSession{}, fmt.Errorf("Cookie 不能包含换行符")
	}

	values := make(map[string]string)
	for part := range strings.SplitSeq(cookie, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.TrimSpace(name) != "" {
			values[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
	}
	userEmail, err := decodeCompshareCookieValue(values["U_USER_EMAIL"])
	if err != nil || strings.TrimSpace(userEmail) == "" {
		return compshareConsoleSession{}, fmt.Errorf("Cookie 缺少有效的 U_USER_EMAIL")
	}
	csrfToken, err := decodeCompshareCookieValue(values["U_CSRF_TOKEN"])
	if err != nil || strings.TrimSpace(csrfToken) == "" {
		return compshareConsoleSession{}, fmt.Errorf("Cookie 缺少有效的 U_CSRF_TOKEN")
	}
	projectID, err := findCompshareProjectID(values, userEmail)
	if err != nil {
		return compshareConsoleSession{}, err
	}
	return compshareConsoleSession{
		Cookie: cookie, UserEmail: userEmail, CSRFToken: csrfToken, ProjectID: projectID,
	}, nil
}

func findCompshareProjectID(values map[string]string, userEmail string) (string, error) {
	preferredName := "c_project_" + compshareCookieIdentity(userEmail)
	if value, ok := values[preferredName]; ok {
		projectID, err := parseCompshareProjectID(value)
		if err != nil {
			return "", fmt.Errorf("Cookie 中的当前项目无效: %w", err)
		}
		return projectID, nil
	}

	projectIDs := make(map[string]struct{})
	for name, value := range values {
		if !strings.HasPrefix(name, "c_project_") {
			continue
		}
		projectID, err := parseCompshareProjectID(value)
		if err == nil && projectID != "" {
			projectIDs[projectID] = struct{}{}
		}
	}
	if len(projectIDs) == 0 {
		return "", fmt.Errorf("Cookie 缺少当前项目 c_project_* 信息")
	}
	if len(projectIDs) > 1 {
		return "", fmt.Errorf("Cookie 包含多个账号的项目上下文，请仅复制当前优云智算会话的 Cookie")
	}
	for projectID := range projectIDs {
		return projectID, nil
	}
	return "", fmt.Errorf("Cookie 缺少当前项目 c_project_* 信息")
}

func parseCompshareProjectID(rawValue string) (string, error) {
	value, err := decodeCompshareCookieValue(rawValue)
	if err != nil {
		return "", err
	}
	var project struct {
		ProjectID string `json:"ProjectId"`
	}
	if err := json.Unmarshal([]byte(value), &project); err != nil {
		return "", fmt.Errorf("无法解析 ProjectId")
	}
	project.ProjectID = strings.TrimSpace(project.ProjectID)
	if project.ProjectID == "" {
		return "", fmt.Errorf("ProjectId 为空")
	}
	return project.ProjectID, nil
}

func decodeCompshareCookieValue(value string) (string, error) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" {
		return "", nil
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(decoded), nil
}

func compshareCookieIdentity(value string) string {
	var builder strings.Builder
	for _, current := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			builder.WriteRune(current)
		} else {
			builder.WriteByte('_')
		}
	}
	return builder.String()
}
