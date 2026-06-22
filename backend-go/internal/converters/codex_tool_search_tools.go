package converters

import (
	"encoding/json"
	"fmt"

	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/types"
)

// MergeCodexToolSearchOutputTools merges tools returned by Codex tool_search_output
// history items into the current request tools for Chat-style upstreams.
func MergeCodexToolSearchOutputTools(rawTools []interface{}, sess *session.Session, input interface{}) []interface{} {
	merged := make([]interface{}, 0, len(rawTools))
	seen := make(map[string]struct{}, len(rawTools))
	for _, tool := range rawTools {
		merged = appendToolIfNew(merged, seen, tool)
	}

	if sess != nil {
		merged = mergeToolSearchOutputItems(merged, seen, sess.Messages)
	}

	inputItems, err := types.ParseResponsesInput(input)
	if err == nil {
		merged = mergeToolSearchOutputItems(merged, seen, inputItems)
	}

	return merged
}

func mergeToolSearchOutputItems(merged []interface{}, seen map[string]struct{}, items []types.ResponsesItem) []interface{} {
	for _, item := range items {
		if item.Type != "tool_search_output" {
			continue
		}
		if item.Execution != "" && item.Execution != "client" {
			continue
		}
		for _, tool := range item.Tools {
			merged = appendToolIfNew(merged, seen, tool)
		}
	}
	return merged
}

func appendToolIfNew(tools []interface{}, seen map[string]struct{}, tool interface{}) []interface{} {
	key := codexToolIdentity(tool)
	if key == "" {
		return tools
	}
	if _, exists := seen[key]; exists {
		return tools
	}
	seen[key] = struct{}{}
	return append(tools, tool)
}

func codexToolIdentity(tool interface{}) string {
	if name, ok := tool.(string); ok {
		if name == "" {
			return ""
		}
		return "string:" + name
	}

	toolMap, ok := tool.(map[string]interface{})
	if !ok {
		return fallbackToolIdentity(tool)
	}

	toolType, _ := toolMap["type"].(string)
	name, _ := toolMap["name"].(string)
	if toolType != "" || name != "" {
		return fmt.Sprintf("%s:%s", toolType, name)
	}
	return fallbackToolIdentity(tool)
}

func fallbackToolIdentity(tool interface{}) string {
	data, err := json.Marshal(tool)
	if err != nil {
		return ""
	}
	return string(data)
}
