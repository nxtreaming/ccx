package common

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

func extractReasoningEffortForLog(bodyBytes []byte) string {
	if len(bytes.TrimSpace(bodyBytes)) == 0 || !gjson.ValidBytes(bodyBytes) {
		return ""
	}

	if value := strings.TrimSpace(gjson.GetBytes(bodyBytes, "thinking.type").String()); strings.EqualFold(value, "disabled") {
		return "none"
	}

	for _, path := range []string{
		"reasoning_effort",
		"reasoning.effort",
		"reasoning",
		"thinking.effort",
		"output_config.effort",
		"generationConfig.thinkingConfig.thinkingLevel",
		"thinkingConfig.thinkingLevel",
		"thinking.thinkingLevel",
	} {
		if value := stringReasoningValue(bodyBytes, path); value != "" {
			return value
		}
	}

	for _, path := range []string{
		"generationConfig.thinkingConfig.thinkingBudget",
		"thinkingConfig.thinkingBudget",
		"thinking.budget_tokens",
		"thinking.budgetTokens",
	} {
		if value := gjson.GetBytes(bodyBytes, path); value.Exists() {
			return "budget=" + formatReasoningNumber(value)
		}
	}

	for _, path := range []string{
		"generationConfig.thinkingConfig.includeThoughts",
		"thinkingConfig.includeThoughts",
	} {
		if value := gjson.GetBytes(bodyBytes, path); value.Exists() && value.Bool() {
			return "enabled"
		}
	}

	if value := strings.TrimSpace(gjson.GetBytes(bodyBytes, "thinking.type").String()); value != "" {
		return value
	}

	return ""
}

func extractActualReasoningEffortForLog(req *http.Request) string {
	bodyBytes := snapshotRequestBodyForLog(req)
	if len(bodyBytes) == 0 {
		return ""
	}
	return extractReasoningEffortForLog(bodyBytes)
}

func snapshotRequestBodyForLog(req *http.Request) []byte {
	if req == nil || req.Body == nil {
		return nil
	}
	contentType := strings.ToLower(req.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return nil
	}

	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil
		}
		defer body.Close()
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return nil
		}
		return bodyBytes
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes
}

func formatReasoningNumber(value gjson.Result) string {
	if value.Type == gjson.Number {
		number := value.Float()
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return strings.TrimSpace(value.String())
}

func stringReasoningValue(bodyBytes []byte, path string) string {
	value := gjson.GetBytes(bodyBytes, path)
	if !value.Exists() {
		return ""
	}
	switch value.Type {
	case gjson.String, gjson.Number, gjson.True, gjson.False:
		return strings.TrimSpace(value.String())
	default:
		return ""
	}
}
