package handlers

import (
	"net/http"

	"github.com/BenedictKing/ccx/internal/utils"
)

const (
	claudeCodeProbeVersion       = utils.ClaudeCodeProbeVersion
	claudeCodeProbeUserAgent     = utils.ClaudeCodeProbeUserAgent
	claudeCodeProbeBetaHeader    = utils.ClaudeCodeProbeBetaHeader
	claudeCodeProbeBillingHeader = utils.ClaudeCodeProbeBillingHeader
	claudeCodeProbeIdentity      = utils.ClaudeCodeProbeIdentity
)

type claudeCodeProbeUserID = utils.ClaudeCodeProbeUserID

func newClaudeCodeProbeMetadata() (map[string]string, string) {
	return utils.NewClaudeCodeProbeMetadata()
}

func newClaudeCodeProbeBillingBlock() map[string]interface{} {
	return utils.NewClaudeCodeProbeBillingBlock()
}

func newClaudeCodeProbeIdentityBlock() map[string]interface{} {
	return utils.NewClaudeCodeProbeIdentityBlock()
}

// ensureClaudeCodeProbeBody 为所有 Messages 探针补齐 Claude Code 的 system 指纹与
// metadata.user_id JSON 字符串，并返回与请求体一致的会话 ID。
func ensureClaudeCodeProbeBody(body []byte) ([]byte, string) {
	return utils.EnsureClaudeCodeProbeBody(body)
}

func applyClaudeCodeProbeHeaders(headers http.Header, sessionID string) {
	utils.ApplyClaudeCodeProbeHeaders(headers, sessionID)
}
