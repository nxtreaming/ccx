package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	claudeCodeProbeVersion       = "2.1.209"
	claudeCodeProbeUserAgent     = "claude-cli/" + claudeCodeProbeVersion + " (external, cli)"
	claudeCodeProbeBetaHeader    = "claude-code-20250219,adaptive-thinking-2026-01-28,prompt-caching-scope-2026-01-05,effort-2025-11-24"
	claudeCodeProbeBillingHeader = "x-anthropic-billing-header: cc_version=" + claudeCodeProbeVersion + ".2f9; cc_entrypoint=cli;"
)

var (
	claudeCodeProbeDeviceID    = uuid.NewString()
	claudeCodeProbeAccountUUID = uuid.NewString()
)

type claudeCodeProbeUserID struct {
	DeviceID    string `json:"device_id"`
	AccountUUID string `json:"account_uuid"`
	SessionID   string `json:"session_id"`
}

func newClaudeCodeProbeMetadata() (map[string]string, string) {
	sessionID := uuid.NewString()
	userID, _ := json.Marshal(claudeCodeProbeUserID{
		DeviceID:    claudeCodeProbeDeviceID,
		AccountUUID: claudeCodeProbeAccountUUID,
		SessionID:   sessionID,
	})
	return map[string]string{"user_id": string(userID)}, sessionID
}

// ensureClaudeCodeProbeBody 为所有 Messages 探针补齐 Claude Code 的计费归因块与
// metadata.user_id JSON 字符串，并返回与请求体一致的会话 ID。
func ensureClaudeCodeProbeBody(body []byte) ([]byte, string) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, uuid.NewString()
	}

	changed := ensureClaudeCodeProbeBillingBlock(payload)
	sessionID := claudeCodeProbeSessionID(payload)
	if sessionID == "" {
		metadata, generatedSessionID := newClaudeCodeProbeMetadata()
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
	var parsed claudeCodeProbeUserID
	if err := json.Unmarshal([]byte(userID), &parsed); err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.SessionID)
}

func ensureClaudeCodeProbeBillingBlock(payload map[string]interface{}) bool {
	switch system := payload["system"].(type) {
	case []interface{}:
		for _, raw := range system {
			block, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			text, _ := block["text"].(string)
			if strings.HasPrefix(text, "x-anthropic-billing-header") && strings.Contains(text, "cc_entrypoint=") {
				return false
			}
		}
	case string:
		if strings.Contains(system, "x-anthropic-billing-header") && strings.Contains(system, "cc_entrypoint=") {
			return false
		}
		payload["system"] = claudeCodeProbeBillingHeader + "\n\n" + system
		return true
	}

	billingBlock := map[string]interface{}{"type": "text", "text": claudeCodeProbeBillingHeader}
	switch system := payload["system"].(type) {
	case []interface{}:
		payload["system"] = append([]interface{}{billingBlock}, system...)
	default:
		payload["system"] = []interface{}{billingBlock}
	}
	return true
}

func applyClaudeCodeProbeHeaders(headers http.Header, sessionID string) {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = uuid.NewString()
	}
	headers.Set("Accept", "application/json")
	headers.Set("anthropic-version", "2023-06-01")
	headers.Set("anthropic-beta", claudeCodeProbeBetaHeader)
	headers.Set("anthropic-dangerous-direct-browser-access", "true")
	headers.Set("User-Agent", claudeCodeProbeUserAgent)
	headers.Set("X-App", "cli")
	headers.Set("X-Claude-Code-Session-Id", sessionID)
}
