package common

import (
	"regexp"
	"strings"

	"github.com/BenedictKing/ccx/internal/autopilot"
)

const maxAutopilotSignalRunes = 32_768

var autopilotFileExtensionPattern = regexp.MustCompile(`(?i)\.[a-z][a-z0-9]{0,9}\b`)

type autopilotPromptAnalysis struct {
	Complexity  autopilot.TaskComplexity
	DomainHints autopilot.DomainHints
}

func analyzeAutopilotPrompt(req map[string]interface{}, estTokens int, explicitDomain string) autopilotPromptAnalysis {
	if req == nil {
		return autopilotPromptAnalysis{Complexity: autopilot.TaskComplexityUnknown}
	}

	systemTexts := make([]string, 0, 4)
	for _, key := range []string{"system", "instructions", "system_instruction", "systemInstruction"} {
		appendAutopilotText(&systemTexts, req[key])
	}

	userTexts := make([]string, 0, 8)
	messageCount := 0
	collectAutopilotRoleTexts(req["messages"], &userTexts, &messageCount)
	collectAutopilotRoleTexts(req["contents"], &userTexts, &messageCount)
	if input, ok := req["input"].(string); ok && strings.TrimSpace(input) != "" {
		userTexts = append(userTexts, input)
		messageCount++
	} else {
		collectAutopilotRoleTexts(req["input"], &userTexts, &messageCount)
	}
	if prompt, ok := req["prompt"].(string); ok && strings.TrimSpace(prompt) != "" {
		userTexts = append(userTexts, prompt)
		messageCount++
	}

	toolNames := extractAutopilotToolNames(req["tools"])
	complexityTexts := userTexts
	if len(complexityTexts) > 3 {
		complexityTexts = complexityTexts[len(complexityTexts)-3:]
	}
	complexityText := joinAutopilotSignalText(complexityTexts)
	domainText := joinAutopilotSignalText(append(append([]string{}, systemTexts...), userTexts...))
	hasDiff := strings.Contains(domainText, "diff --git") || strings.Contains(domainText, "@@ -")

	return autopilotPromptAnalysis{
		Complexity: autopilot.InferTaskComplexity(autopilot.ComplexitySignals{
			PromptText:     complexityText,
			MessageCount:   messageCount,
			ToolCount:      len(toolNames),
			EstTokens:      estTokens,
			HasDiffContext: hasDiff,
		}),
		DomainHints: autopilot.DomainHints{
			ExplicitDomain: strings.TrimSpace(explicitDomain),
			SystemPrompt:   domainText,
			ToolNames:      toolNames,
			FileExtensions: extractAutopilotFileExtensions(domainText),
			HasDiffContext: hasDiff,
		},
	}
}

func collectAutopilotRoleTexts(value interface{}, texts *[]string, count *int) {
	items, ok := value.([]interface{})
	if !ok {
		return
	}
	for _, item := range items {
		message, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(stringValue(message["role"])))
		if role != "" && role != "user" {
			continue
		}
		before := len(*texts)
		if content, exists := message["content"]; exists {
			appendAutopilotText(texts, content)
		} else if parts, exists := message["parts"]; exists {
			appendAutopilotText(texts, parts)
		} else {
			appendAutopilotText(texts, message)
		}
		if len(*texts) > before {
			(*count)++
		}
	}
}

func appendAutopilotText(texts *[]string, value interface{}) {
	switch typed := value.(type) {
	case string:
		if text := strings.TrimSpace(typed); text != "" {
			*texts = append(*texts, text)
		}
	case []interface{}:
		for _, item := range typed {
			appendAutopilotText(texts, item)
		}
	case map[string]interface{}:
		blockType := strings.ToLower(strings.TrimSpace(stringValue(typed["type"])))
		if strings.Contains(blockType, "image") || strings.Contains(blockType, "tool_result") || blockType == "tool" {
			return
		}
		if text, ok := typed["text"].(string); ok {
			appendAutopilotText(texts, text)
			return
		}
		for _, key := range []string{"content", "parts"} {
			if nested, exists := typed[key]; exists {
				appendAutopilotText(texts, nested)
			}
		}
	}
}

func extractAutopilotToolNames(value interface{}) []string {
	seen := make(map[string]bool)
	var names []string
	var visit func(interface{})
	visit = func(current interface{}) {
		switch typed := current.(type) {
		case []interface{}:
			for _, item := range typed {
				visit(item)
			}
		case map[string]interface{}:
			if name := strings.TrimSpace(stringValue(typed["name"])); name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
			for _, key := range []string{"function", "functionDeclarations"} {
				if nested, exists := typed[key]; exists {
					visit(nested)
				}
			}
		}
	}
	visit(value)
	return names
}

func extractAutopilotFileExtensions(text string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, 8)
	for _, extension := range autopilotFileExtensionPattern.FindAllString(text, 32) {
		extension = strings.ToLower(extension)
		if !seen[extension] {
			seen[extension] = true
			result = append(result, extension)
		}
	}
	return result
}

func joinAutopilotSignalText(parts []string) string {
	text := strings.Join(parts, "\n")
	runes := []rune(text)
	if len(runes) <= maxAutopilotSignalRunes {
		return text
	}
	return string(runes[len(runes)-maxAutopilotSignalRunes:])
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}
