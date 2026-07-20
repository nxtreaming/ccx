package utils

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// Claude Code 探针使用的请求特征。它仅用于主动兼容性和端点验证探测，
// 与实际客户端请求中的 Claude Code 身份字段保持一致。
const (
	ClaudeCodeProbeVersion       = "2.1.209"
	ClaudeCodeProbeUserAgent     = "claude-cli/" + ClaudeCodeProbeVersion + " (external, cli)"
	ClaudeCodeProbeBetaHeader    = "claude-code-20250219,adaptive-thinking-2026-01-28,prompt-caching-scope-2026-01-05,effort-2025-11-24"
	ClaudeCodeProbeBillingHeader = "x-anthropic-billing-header: cc_version=" + ClaudeCodeProbeVersion + ".2f9; cc_entrypoint=cli;"
	ClaudeCodeProbeIdentity      = "You are Claude Code, Anthropic's official CLI for Claude."
)

// ClaudeCodeProbeUserID 是 Claude Code 探针 metadata.user_id 的 JSON 结构。
type ClaudeCodeProbeUserID struct {
	DeviceID    string `json:"device_id"`
	AccountUUID string `json:"account_uuid"`
	SessionID   string `json:"session_id"`
}

var (
	claudeCodeProbeDeviceID    = uuid.NewString()
	claudeCodeProbeAccountUUID = uuid.NewString()
)

// NewClaudeCodeProbeMetadata 创建与请求头一致的 Claude Code 会话元数据。
func NewClaudeCodeProbeMetadata() (map[string]string, string) {
	sessionID := uuid.NewString()
	userID, _ := json.Marshal(ClaudeCodeProbeUserID{
		DeviceID:    claudeCodeProbeDeviceID,
		AccountUUID: claudeCodeProbeAccountUUID,
		SessionID:   sessionID,
	})
	return map[string]string{"user_id": string(userID)}, sessionID
}

// NewClaudeCodeProbeBillingBlock 返回 Claude Code 账单身份系统块。
func NewClaudeCodeProbeBillingBlock() map[string]interface{} {
	return map[string]interface{}{"type": "text", "text": ClaudeCodeProbeBillingHeader}
}

// NewClaudeCodeProbeIdentityBlock 返回 Claude Code 身份系统块。
func NewClaudeCodeProbeIdentityBlock() map[string]interface{} {
	return map[string]interface{}{
		"type":          "text",
		"text":          ClaudeCodeProbeIdentity,
		"cache_control": map[string]string{"type": "ephemeral"},
	}
}

// EnsureClaudeCodeProbeBody 为 Messages 探针补齐 Claude Code 的系统身份和 metadata。
// 返回值中的 session ID 必须传给 ApplyClaudeCodeProbeHeaders。
func EnsureClaudeCodeProbeBody(body []byte) ([]byte, string) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, uuid.NewString()
	}

	changed := ensureClaudeCodeProbeSystem(payload)
	sessionID := claudeCodeProbeSessionID(payload)
	if sessionID == "" {
		metadata, generatedSessionID := NewClaudeCodeProbeMetadata()
		payload["metadata"] = metadata
		sessionID = generatedSessionID
		changed = true
	}
	if !changed {
		return body, sessionID
	}

	updated, err := json.Marshal(payload)
	if err != nil {
		return body, sessionID
	}
	return updated, sessionID
}

func claudeCodeProbeSessionID(payload map[string]interface{}) string {
	metadata, ok := payload["metadata"].(map[string]interface{})
	if !ok {
		return ""
	}
	userID, ok := metadata["user_id"].(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return ""
	}
	var parsed ClaudeCodeProbeUserID
	if err := json.Unmarshal([]byte(userID), &parsed); err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.SessionID)
}

func ensureClaudeCodeProbeSystem(payload map[string]interface{}) bool {
	switch system := payload["system"].(type) {
	case []interface{}:
		var billingBlock interface{}
		var identityBlock interface{}
		billingCount := 0
		identityCount := 0
		rest := make([]interface{}, 0, len(system))
		for _, raw := range system {
			text := claudeCodeProbeSystemBlockText(raw)
			switch {
			case isClaudeCodeProbeBillingBlock(text):
				billingCount++
				if billingBlock == nil {
					billingBlock = raw
				}
			case isClaudeCodeProbeIdentityBlock(text):
				identityCount++
				if identityBlock == nil {
					identityBlock = raw
				}
			default:
				rest = append(rest, raw)
			}
		}
		if billingCount == 1 && identityCount == 1 && len(system) >= 2 &&
			isClaudeCodeProbeBillingBlock(claudeCodeProbeSystemBlockText(system[0])) &&
			isClaudeCodeProbeIdentityBlock(claudeCodeProbeSystemBlockText(system[1])) {
			return false
		}
		if billingBlock == nil {
			billingBlock = NewClaudeCodeProbeBillingBlock()
		}
		if identityBlock == nil {
			identityBlock = NewClaudeCodeProbeIdentityBlock()
		}
		payload["system"] = append([]interface{}{billingBlock, identityBlock}, rest...)
		return true
	case string:
		hasBilling := isClaudeCodeProbeBillingBlock(system)
		hasIdentity := strings.Contains(system, ClaudeCodeProbeIdentity)
		if hasBilling && hasIdentity {
			return false
		}
		switch {
		case hasBilling:
			payload["system"] = system + "\n\n" + ClaudeCodeProbeIdentity
		case hasIdentity:
			payload["system"] = ClaudeCodeProbeBillingHeader + "\n\n" + system
		default:
			parts := []string{ClaudeCodeProbeBillingHeader, ClaudeCodeProbeIdentity}
			if strings.TrimSpace(system) != "" {
				parts = append(parts, system)
			}
			payload["system"] = strings.Join(parts, "\n\n")
		}
		return true
	}

	payload["system"] = []interface{}{
		NewClaudeCodeProbeBillingBlock(),
		NewClaudeCodeProbeIdentityBlock(),
	}
	return true
}

func isClaudeCodeProbeBillingBlock(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "x-anthropic-billing-header") && strings.Contains(trimmed, "cc_entrypoint=")
}

func isClaudeCodeProbeIdentityBlock(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), ClaudeCodeProbeIdentity)
}

func claudeCodeProbeSystemBlockText(raw interface{}) string {
	block, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	text, _ := block["text"].(string)
	return text
}

// ApplyClaudeCodeProbeHeaders 为已补齐身份的 Messages 探针设置 Claude Code 请求头。
func ApplyClaudeCodeProbeHeaders(headers http.Header, sessionID string) {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = uuid.NewString()
	}
	headers.Set("Accept", "application/json")
	headers.Set("anthropic-version", "2023-06-01")
	headers.Set("anthropic-beta", ClaudeCodeProbeBetaHeader)
	headers.Set("anthropic-dangerous-direct-browser-access", "true")
	headers.Set("User-Agent", ClaudeCodeProbeUserAgent)
	headers.Set("X-App", "cli")
	headers.Set("X-Claude-Code-Session-Id", sessionID)
}
