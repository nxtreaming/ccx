package converters

import (
	"testing"

	"github.com/BenedictKing/ccx/internal/types"
	"github.com/stretchr/testify/assert"
)

// summaryText 从 ResponsesItem.Summary（interface{} 包装的 []interface{}）中提取 summary_text。
func summaryText(t *testing.T, item types.ResponsesItem) string {
	t.Helper()
	arr, ok := item.Summary.([]interface{})
	if !ok {
		t.Fatalf("Summary is not []interface{}, got %T: %v", item.Summary, item.Summary)
	}
	if len(arr) == 0 {
		t.Fatal("Summary is empty")
	}
	m, ok := arr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Summary[0] is not map: %v", arr[0])
	}
	text, _ := m["text"].(string)
	return text
}

// firstContentText 从 ResponsesItem.Content（interface{} 包装的 []ContentBlock 或 []interface{}）中
// 提取第一段 output_text 的 text。
func firstContentText(t *testing.T, item types.ResponsesItem) string {
	t.Helper()
	switch v := item.Content.(type) {
	case []types.ContentBlock:
		if len(v) == 0 {
			t.Fatal("Content blocks is empty")
		}
		return v[0].Text
	case []interface{}:
		if len(v) == 0 {
			t.Fatal("Content blocks is empty")
		}
		m, ok := v[0].(map[string]interface{})
		if !ok {
			t.Fatalf("Content[0] is not map: %v", v[0])
		}
		text, _ := m["text"].(string)
		return text
	default:
		t.Fatalf("Content has unexpected type %T: %v", item.Content, item.Content)
		return ""
	}
}

// TestOpenAIChatResponseToResponses_ThinkAtStart 验证 content 开头的 <think>...</think>
// 会被提取为 reasoning ResponsesItem，剩余文本作为 message ResponsesItem。
func TestOpenAIChatResponseToResponses_ThinkAtStart(t *testing.T) {
	openaiResp := map[string]interface{}{
		"model": "MiniMax-M2.7",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "<think>let me think</think>final answer",
				},
			},
		},
	}

	resp, err := OpenAIChatResponseToResponses(openaiResp, "sess_test")
	assert.NoError(t, err)
	assert.Len(t, resp.Output, 2)

	assert.Equal(t, "reasoning", resp.Output[0].Type)
	assert.Equal(t, "let me think", summaryText(t, resp.Output[0]))

	assert.Equal(t, "message", resp.Output[1].Type)
	assert.Equal(t, "assistant", resp.Output[1].Role)
	assert.Equal(t, "final answer", firstContentText(t, resp.Output[1]))
}

// TestOpenAIChatResponseToResponses_ThinkMergesWithReasoningContent 验证原生
// reasoning_content 与 content 头部 <think>...</think> 共存时被合并到单个 reasoning item。
func TestOpenAIChatResponseToResponses_ThinkMergesWithReasoningContent(t *testing.T) {
	openaiResp := map[string]interface{}{
		"model": "MiniMax-M2.7",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":              "assistant",
					"reasoning_content": "native-",
					"content":           "<think>tagged</think>visible",
				},
			},
		},
	}

	resp, err := OpenAIChatResponseToResponses(openaiResp, "sess_test")
	assert.NoError(t, err)
	assert.Len(t, resp.Output, 2)
	assert.Equal(t, "reasoning", resp.Output[0].Type)
	assert.Equal(t, "native-tagged", summaryText(t, resp.Output[0]))
	assert.Equal(t, "message", resp.Output[1].Type)
	assert.Equal(t, "visible", firstContentText(t, resp.Output[1]))
}

// TestOpenAIChatResponseToResponses_ThinkInMiddleNotStripped 验证 content 非起始位置出现
// 的 <think> 字面不应被剥离。
func TestOpenAIChatResponseToResponses_ThinkInMiddleNotStripped(t *testing.T) {
	openaiResp := map[string]interface{}{
		"model": "gpt-4o",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "hello <think>not-reasoning</think> world",
				},
			},
		},
	}

	resp, err := OpenAIChatResponseToResponses(openaiResp, "sess_test")
	assert.NoError(t, err)
	assert.Len(t, resp.Output, 1)
	assert.Equal(t, "message", resp.Output[0].Type)
	assert.Equal(t, "hello <think>not-reasoning</think> world", firstContentText(t, resp.Output[0]))
}

// TestOpenAIChatResponseToResponses_ThinkOnlyToolCalls 验证 content 只含 <think>...</think>
// （提取后正文为空）+ tool_calls 时，输出为 reasoning + function_call 两项，
// 不应出现空的 message item。
func TestOpenAIChatResponseToResponses_ThinkOnlyToolCalls(t *testing.T) {
	openaiResp := map[string]interface{}{
		"model": "MiniMax-M2.7",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "<think>decide to call tool</think>",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call_42",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "lookup",
								"arguments": `{"q":"x"}`,
							},
						},
					},
				},
			},
		},
	}

	resp, err := OpenAIChatResponseToResponses(openaiResp, "sess_test")
	assert.NoError(t, err)
	assert.Len(t, resp.Output, 2)
	assert.Equal(t, "reasoning", resp.Output[0].Type)
	assert.Equal(t, "function_call", resp.Output[1].Type)
	assert.Equal(t, "call_42", resp.Output[1].CallID)
	assert.Equal(t, "lookup", resp.Output[1].Name)
	// 不应有空 message item
	for _, item := range resp.Output {
		assert.NotEqual(t, "message", item.Type, "should not emit empty message item")
	}
}

// TestOpenAIChatResponseToResponses_UnclosedThink 验证未闭合 <think> 时所有内容都进入 reasoning。
func TestOpenAIChatResponseToResponses_UnclosedThink(t *testing.T) {
	openaiResp := map[string]interface{}{
		"model": "MiniMax-M2.7",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "<think>incomplete thought",
				},
			},
		},
	}

	resp, err := OpenAIChatResponseToResponses(openaiResp, "sess_test")
	assert.NoError(t, err)
	assert.Len(t, resp.Output, 1)
	assert.Equal(t, "reasoning", resp.Output[0].Type)
	assert.Equal(t, "incomplete thought", summaryText(t, resp.Output[0]))
}
