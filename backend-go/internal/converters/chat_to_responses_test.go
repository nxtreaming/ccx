package converters

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func extractResponseCompletedUsage(t *testing.T, events []string) map[string]interface{} {
	t.Helper()
	for _, event := range events {
		if !strings.Contains(event, "event: response.completed") {
			continue
		}
		for _, line := range strings.Split(event, "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			jsonStr := strings.TrimPrefix(line, "data: ")
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
				continue
			}
			response, ok := payload["response"].(map[string]interface{})
			if !ok {
				continue
			}
			usage, ok := response["usage"].(map[string]interface{})
			if ok {
				return usage
			}
		}
	}
	t.Fatalf("未找到 response.completed usage 事件: %v", events)
	return nil
}

func TestConvertResponsesToOpenAIChatRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		model    string
		stream   bool
		validate func(t *testing.T, result []byte)
	}{
		{
			name: "基本文本输入",
			input: `{
				"model": "gpt-4",
				"input": "Hello, world!",
				"instructions": "You are a helpful assistant."
			}`,
			model:  "gpt-4o",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				if root.Get("model").String() != "gpt-4o" {
					t.Errorf("model should be gpt-4o, got %s", root.Get("model").String())
				}
				if root.Get("stream").Bool() != false {
					t.Error("stream should be false")
				}
				messages := root.Get("messages").Array()
				if len(messages) != 2 {
					t.Errorf("should have 2 messages (system + user), got %d", len(messages))
				}
				if messages[0].Get("role").String() != "system" {
					t.Error("first message should be system")
				}
				if messages[1].Get("role").String() != "user" {
					t.Error("second message should be user")
				}
			},
		},
		{
			name: "带 tools 的请求",
			input: `{
				"model": "gpt-4",
				"input": [{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "What's the weather?"}]}],
				"tools": [
					{
						"name": "get_weather",
						"description": "Get weather info",
						"parameters": {"type": "object", "properties": {"location": {"type": "string"}}}
					}
				]
			}`,
			model:  "gpt-4o",
			stream: true,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				if root.Get("stream").Bool() != true {
					t.Error("stream should be true")
				}
				tools := root.Get("tools").Array()
				if len(tools) != 1 {
					t.Errorf("should have 1 tool, got %d", len(tools))
				}
				if tools[0].Get("function.name").String() != "get_weather" {
					t.Error("tool name should be get_weather")
				}
			},
		},
		{
			name: "function_call 和 function_call_output",
			input: `{
				"model": "gpt-4",
				"input": [
					{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "What's the weather in NYC?"}]},
					{"type": "function_call", "call_id": "call_123", "name": "get_weather", "arguments": "{\"location\": \"NYC\"}"},
					{"type": "function_call_output", "call_id": "call_123", "output": "Sunny, 72°F"}
				]
			}`,
			model:  "gpt-4o",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				messages := root.Get("messages").Array()
				if len(messages) != 3 {
					t.Errorf("should have 3 messages, got %d", len(messages))
				}
				// 第二条消息应该是 assistant with tool_calls
				if messages[1].Get("role").String() != "assistant" {
					t.Error("second message should be assistant")
				}
				if !messages[1].Get("tool_calls").Exists() {
					t.Error("assistant message should have tool_calls")
				}
				// 第三条消息应该是 tool
				if messages[2].Get("role").String() != "tool" {
					t.Error("third message should be tool")
				}
			},
		},
		{
			name: "多模态图片输入保留为 Chat content array",
			input: `{
				"model": "mimo-v2.5-pro",
				"input": [{"type": "message", "role": "user", "content": [
					{"type": "input_text", "text": "描述这张图片"},
					{"type": "input_image", "image_url": "data:image/png;base64,abc", "detail": "high"}
				]}]
			}`,
			model:  "mimo-v2.5-pro",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				content := root.Get("messages.0.content")
				if !content.IsArray() {
					t.Fatalf("content should be array, got %s", content.Raw)
				}
				if content.Get("0.type").String() != "text" || content.Get("0.text").String() != "描述这张图片" {
					t.Fatalf("text block mismatch: %s", content.Get("0").Raw)
				}
				if content.Get("1.type").String() != "image_url" {
					t.Fatalf("image block type mismatch: %s", content.Get("1").Raw)
				}
				if content.Get("1.image_url.url").String() != "data:image/png;base64,abc" {
					t.Fatalf("image url mismatch: %s", content.Get("1").Raw)
				}
				if content.Get("1.image_url.detail").String() != "high" {
					t.Fatalf("image detail mismatch: %s", content.Get("1").Raw)
				}
			},
		},
		{
			name: "tool_choice object 保真",
			input: `{
				"model": "gpt-4",
				"input": "Call a tool",
				"tool_choice": {"type": "function", "function": {"name": "get_weather"}}
			}`,
			model:  "gpt-4o",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				if root.Get("tool_choice.type").String() != "function" {
					t.Fatalf("tool_choice.type should be function, got %s", root.Get("tool_choice.type").String())
				}
				if root.Get("tool_choice.function.name").String() != "get_weather" {
					t.Fatalf("tool_choice.function.name should be get_weather, got %s", root.Get("tool_choice.function.name").String())
				}
			},
		},

		{
			name: "tools 缺失 required 字段时自动补齐 []",
			input: `{
				"model": "gpt-5-codex",
				"input": "list mcp resources",
				"tools": [
					{
						"type": "function",
						"name": "list_mcp_resources",
						"description": "Lists resources provided by MCP servers.",
						"strict": false,
						"parameters": {
							"type": "object",
							"properties": {
								"cursor": {"type": "string"},
								"server": {"type": "string"}
							},
							"additionalProperties": false
						}
					}
				]
			}`,
			model:  "gpt-5-codex",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				tools := root.Get("tools").Array()
				if len(tools) != 1 {
					t.Fatalf("should have 1 tool, got %d", len(tools))
				}
				tool := tools[0]
				if tool.Get("function.name").String() != "list_mcp_resources" {
					t.Fatalf("tool name mismatch: %s", tool.Raw)
				}
				params := tool.Get("function.parameters")
				if params.Get("type").String() != "object" {
					t.Fatalf("parameters.type should be object: %s", params.Raw)
				}
				required := params.Get("required")
				if !required.Exists() || !required.IsArray() {
					t.Fatalf("parameters.required should exist and be array, got %s", params.Raw)
				}
				if params.Get("additionalProperties").Bool() != false {
					t.Fatalf("additionalProperties should be preserved: %s", params.Raw)
				}
			},
		},
		{
			name: "非 function 类型的工具应被跳过",
			input: `{
				"model": "gpt-5-codex",
				"input": "search the web",
				"tools": [
					{"type": "web_search"},
					{"type": "custom", "name": "grep"},
					{"type": "function", "name": "do_thing", "parameters": {"type": "object", "properties": {}}}
				]
			}`,
			model:  "gpt-5-codex",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				tools := root.Get("tools").Array()
				if len(tools) != 1 {
					t.Fatalf("expected 1 tool after filtering, got %d (%s)", len(tools), root.Get("tools").Raw)
				}
				if tools[0].Get("function.name").String() != "do_thing" {
					t.Fatalf("should keep only function tool, got %s", tools[0].Raw)
				}
			},
		},
		{
			name: "reasoning effort 转换",
			input: `{
				"model": "o1-mini",
				"input": "Think about this",
				"reasoning": {"effort": "high"}
			}`,
			model:  "o1-mini",
			stream: false,
			validate: func(t *testing.T, result []byte) {
				root := gjson.ParseBytes(result)
				if root.Get("reasoning_effort").String() != "high" {
					t.Errorf("reasoning_effort should be high, got %s", root.Get("reasoning_effort").String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertResponsesToOpenAIChatRequest(tt.model, []byte(tt.input), tt.stream)
			tt.validate(t, result)
		})
	}
}

func TestConvertOpenAIChatToResponses_Stream(t *testing.T) {
	ctx := context.Background()

	// 模拟 OpenAI Chat Completions SSE 流
	sseLines := []string{
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" world!"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
		`data: [DONE]`,
	}

	originalReq := []byte(`{"model":"gpt-4o","input":"Hi"}`)

	var state any
	var allEvents []string

	for _, line := range sseLines {
		events := ConvertOpenAIChatToResponses(ctx, "gpt-4o", originalReq, nil, []byte(line), &state)
		allEvents = append(allEvents, events...)
	}

	// 验证事件序列
	if len(allEvents) == 0 {
		t.Fatal("should produce events")
	}

	// 检查是否有 response.created 事件
	hasCreated := false
	hasInProgress := false
	hasCompleted := false
	hasTextDelta := false

	for _, ev := range allEvents {
		if strings.Contains(ev, "response.created") {
			hasCreated = true
		}
		if strings.Contains(ev, "response.in_progress") {
			hasInProgress = true
		}
		if strings.Contains(ev, "response.completed") {
			hasCompleted = true
		}
		if strings.Contains(ev, "response.output_text.delta") {
			hasTextDelta = true
		}
	}

	if !hasCreated {
		t.Error("should have response.created event")
	}
	if !hasInProgress {
		t.Error("should have response.in_progress event")
	}
	if !hasCompleted {
		t.Error("should have response.completed event")
	}
	if !hasTextDelta {
		t.Error("should have response.output_text.delta event")
	}
}

func TestConvertOpenAIChatToResponses_StreamReasoningContent(t *testing.T) {
	ctx := context.Background()
	sseLines := []string{
		`data: {"id":"chatcmpl-ds","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-v4-pro","choices":[{"index":0,"delta":{"reasoning_content":"chat reasoning"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-ds","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-v4-pro","choices":[{"index":0,"delta":{"content":"chat text"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-ds","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-v4-pro","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	var state any
	var allEvents []string
	for _, line := range sseLines {
		events := ConvertOpenAIChatToResponses(ctx, "deepseek-v4-pro", []byte(`{"model":"deepseek-v4-pro","input":"hello"}`), nil, []byte(line), &state)
		allEvents = append(allEvents, events...)
	}

	joined := strings.Join(allEvents, "\n")
	if !strings.Contains(joined, `"type":"reasoning"`) {
		t.Fatalf("expected reasoning item, got %v", allEvents)
	}
	if !strings.Contains(joined, `"text":"chat reasoning"`) {
		t.Fatalf("expected reasoning summary text, got %v", allEvents)
	}
	if !strings.Contains(joined, `"delta":"chat text"`) {
		t.Fatalf("expected text delta after reasoning, got %v", allEvents)
	}
}

func TestConvertOpenAIChatToResponses_ToolCall(t *testing.T) {
	ctx := context.Background()

	// 模拟带 tool_call 的 SSE 流
	sseLines := []string{
		`data: {"id":"chatcmpl-456","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-456","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-456","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\": \"NYC\"}"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-456","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	}

	originalReq := []byte(`{"model":"gpt-4o","input":"What's the weather?","tools":[{"name":"get_weather"}]}`)

	var state any
	var allEvents []string

	for _, line := range sseLines {
		events := ConvertOpenAIChatToResponses(ctx, "gpt-4o", originalReq, nil, []byte(line), &state)
		allEvents = append(allEvents, events...)
	}

	// 验证是否有 function_call 相关事件
	hasFuncAdded := false
	hasFuncDelta := false
	hasFuncDone := false

	for _, ev := range allEvents {
		if strings.Contains(ev, "response.output_item.added") && strings.Contains(ev, "function_call") {
			hasFuncAdded = true
		}
		if strings.Contains(ev, "response.function_call_arguments.delta") {
			hasFuncDelta = true
		}
		if strings.Contains(ev, "response.function_call_arguments.done") {
			hasFuncDone = true
		}
	}

	if !hasFuncAdded {
		t.Error("should have function_call output_item.added event")
	}
	if !hasFuncDelta {
		t.Error("should have function_call_arguments.delta event")
	}
	if !hasFuncDone {
		t.Error("should have function_call_arguments.done event")
	}
}

func TestConvertOpenAIChatToResponses_CustomToolCall(t *testing.T) {
	ctx := context.Background()
	sseLines := []string{
		`data: {"id":"chatcmpl-custom","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_patch","type":"function","function":{"name":"apply_patch_add_file","arguments":""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-custom","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"docs/test.md\",\"content\":\"# Test\\n\"}"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-custom","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	}
	originalReq := []byte(`{"model":"gpt-4o","input":"edit","tools":[{"type":"custom","name":"apply_patch"}],"transformer_metadata":{"codex_tool_compat_enabled":true}}`)

	var state any
	var allEvents []string
	for _, line := range sseLines {
		allEvents = append(allEvents, ConvertOpenAIChatToResponses(ctx, "gpt-4o", originalReq, nil, []byte(line), &state)...)
	}
	joined := strings.Join(allEvents, "\n")
	if strings.Contains(joined, `"type":"function_call"`) {
		t.Fatalf("custom tool stream leaked function_call events: %s", joined)
	}
	if strings.Contains(joined, "response.function_call_arguments.") {
		t.Fatalf("custom tool stream leaked function argument events: %s", joined)
	}
	if !strings.Contains(joined, `"type":"custom_tool_call"`) {
		t.Fatalf("missing custom_tool_call item: %s", joined)
	}
	if !strings.Contains(joined, "response.custom_tool_call_input.delta") {
		t.Fatalf("missing custom tool input delta: %s", joined)
	}
	if !strings.Contains(joined, `*** Add File: docs/test.md`) || strings.Contains(joined, `\n+\n*** End Patch`) {
		t.Fatalf("unexpected patch input: %s", joined)
	}
}

func TestConvertOpenAIChatToResponsesNonStream(t *testing.T) {
	ctx := context.Background()

	// 模拟 OpenAI Chat Completions 非流式响应
	chatResponse := `{
		"id": "chatcmpl-789",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello! How can I help you today?"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 8,
			"total_tokens": 18
		}
	}`

	originalReq := []byte(`{"model":"gpt-4o","input":"Hi","instructions":"Be helpful"}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	// 解析结果
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// 验证基本字段
	if resp["object"] != "response" {
		t.Errorf("object should be response, got %v", resp["object"])
	}
	if resp["status"] != "completed" {
		t.Errorf("status should be completed, got %v", resp["status"])
	}

	// 验证 output
	output, ok := resp["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("output should have items")
	}

	msgItem := output[0].(map[string]interface{})
	if msgItem["type"] != "message" {
		t.Errorf("first output item should be message, got %v", msgItem["type"])
	}

	// 验证 usage
	usage, ok := resp["usage"].(map[string]interface{})
	if !ok {
		t.Fatal("usage should exist")
	}
	if usage["input_tokens"].(float64) != 10 {
		t.Errorf("input_tokens should be 10, got %v", usage["input_tokens"])
	}
	if usage["output_tokens"].(float64) != 8 {
		t.Errorf("output_tokens should be 8, got %v", usage["output_tokens"])
	}
}

func TestConvertOpenAIChatToResponsesNonStream_ToolCalls(t *testing.T) {
	ctx := context.Background()

	// 模拟带 tool_calls 的响应
	chatResponse := `{
		"id": "chatcmpl-tool",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [
					{
						"id": "call_xyz",
						"type": "function",
						"function": {
							"name": "search",
							"arguments": "{\"query\": \"test\"}"
						}
					}
				]
			},
			"finish_reason": "tool_calls"
		}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
	}`

	originalReq := []byte(`{"model":"gpt-4o","input":"Search for test"}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	output, ok := resp["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("output should have items")
	}

	// 查找 function_call item
	var funcItem map[string]interface{}
	for _, item := range output {
		itemMap := item.(map[string]interface{})
		if itemMap["type"] == "function_call" {
			funcItem = itemMap
			break
		}
	}

	if funcItem == nil {
		t.Fatal("should have function_call item")
	}

	if funcItem["name"] != "search" {
		t.Errorf("function name should be search, got %v", funcItem["name"])
	}
	if funcItem["call_id"] != "call_xyz" {
		t.Errorf("call_id should be call_xyz, got %v", funcItem["call_id"])
	}
}

func TestConvertOpenAIChatToResponsesNonStream_CustomToolCall(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id": "chatcmpl-custom",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call_patch",
					"type": "function",
					"function": {
						"name": "apply_patch_add_file",
						"arguments": "{\"path\":\"docs/test.md\",\"content\":\"# Test\\n\"}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
	}`
	originalReq := []byte(`{"model":"gpt-4o","input":"edit","tools":[{"type":"custom","name":"apply_patch"}],"transformer_metadata":{"codex_tool_compat_enabled":true}}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatal(err)
	}
	output := resp["output"].([]interface{})
	item := output[0].(map[string]interface{})
	if item["type"] != "custom_tool_call" {
		t.Fatalf("item type = %v, result = %s", item["type"], result)
	}
	if item["name"] != "apply_patch" {
		t.Fatalf("name = %v", item["name"])
	}
	if _, ok := item["output"]; ok {
		t.Fatalf("custom_tool_call should not contain output: %#v", item)
	}
	if got := item["input"].(string); got != "*** Begin Patch\n*** Add File: docs/test.md\n+# Test\n*** End Patch" {
		t.Fatalf("input = %q", got)
	}
}

func TestConvertOpenAIChatToResponses_Stream_ClaudeCacheTotalTokens(t *testing.T) {
	ctx := context.Background()
	sseLines := []string{
		`data: {"id":"msg-claude-cache","object":"chat.completion.chunk","created":1234567890,"model":"claude","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":null}]}`,
		`data: {"id":"msg-claude-cache","object":"chat.completion.chunk","created":1234567890,"model":"claude","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":10,"cache_read_input_tokens":30}}`,
		`data: [DONE]`,
	}

	originalReq := []byte(`{"model":"claude","input":"hi"}`)
	var state any
	var allEvents []string
	for _, line := range sseLines {
		allEvents = append(allEvents, ConvertOpenAIChatToResponses(ctx, "claude", originalReq, nil, []byte(line), &state)...)
	}

	usage := extractResponseCompletedUsage(t, allEvents)
	if got := int(usage["total_tokens"].(float64)); got != 160 {
		t.Fatalf("total_tokens = %d, want 160", got)
	}
	if got := int(usage["cache_creation_input_tokens"].(float64)); got != 10 {
		t.Fatalf("cache_creation_input_tokens = %d, want 10", got)
	}
	if got := int(usage["cache_read_input_tokens"].(float64)); got != 30 {
		t.Fatalf("cache_read_input_tokens = %d, want 30", got)
	}
}

func TestConvertOpenAIChatToResponses_Stream_OpenAICacheDetailsNormalizesInput(t *testing.T) {
	ctx := context.Background()
	sseLines := []string{
		`data: {"id":"chatcmpl-openai-cache","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.5","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-openai-cache","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":38451,"completion_tokens":1275,"total_tokens":39726,"prompt_tokens_details":{"cached_tokens":36608}}}`,
		`data: [DONE]`,
	}

	originalReq := []byte(`{"model":"gpt-5.5","input":"hi"}`)
	var state any
	var allEvents []string
	for _, line := range sseLines {
		allEvents = append(allEvents, ConvertOpenAIChatToResponses(ctx, "gpt-5.5", originalReq, nil, []byte(line), &state)...)
	}

	usage := extractResponseCompletedUsage(t, allEvents)
	if got := int(usage["input_tokens"].(float64)); got != 1843 {
		t.Fatalf("input_tokens = %d, want 1843", got)
	}
	if got := int(usage["total_tokens"].(float64)); got != 39726 {
		t.Fatalf("total_tokens = %d, want 39726", got)
	}
	if _, exists := usage["cache_read_input_tokens"]; exists {
		t.Fatalf("cache_read_input_tokens should not be emitted for OpenAI cache details: %#v", usage)
	}
	details, ok := usage["input_tokens_details"].(map[string]interface{})
	if !ok {
		t.Fatalf("input_tokens_details missing: %#v", usage)
	}
	if got := int(details["cached_tokens"].(float64)); got != 36608 {
		t.Fatalf("cached_tokens = %d, want 36608", got)
	}
}

func TestConvertOpenAIChatToResponsesNonStream_ClaudeCacheTTLTotalFallback(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id":"chatcmpl-claude-cache",
		"object":"chat.completion",
		"created":1234567890,
		"model":"claude",
		"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{
			"input_tokens":100,
			"output_tokens":20,
			"cache_read_input_tokens":30,
			"cache_creation_5m_input_tokens":7,
			"cache_creation_1h_input_tokens":3
		}
	}`

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "claude", []byte(`{"model":"claude","input":"hi"}`), nil, []byte(chatResponse), nil)
	if got := gjson.Get(result, "usage.total_tokens").Int(); got != 160 {
		t.Fatalf("usage.total_tokens = %d, want 160", got)
	}
}

func TestConvertOpenAIChatToResponsesNonStream_OpenAICacheDetailsNormalizesInput(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id":"chatcmpl-openai-cache",
		"object":"chat.completion",
		"created":1234567890,
		"model":"gpt-5.5",
		"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{
			"prompt_tokens":38451,
			"completion_tokens":1275,
			"total_tokens":39726,
			"prompt_tokens_details":{"cached_tokens":36608}
		}
	}`

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-5.5", []byte(`{"model":"gpt-5.5","input":"hi"}`), nil, []byte(chatResponse), nil)
	if got := gjson.Get(result, "usage.input_tokens").Int(); got != 1843 {
		t.Fatalf("usage.input_tokens = %d, want 1843", got)
	}
	if got := gjson.Get(result, "usage.total_tokens").Int(); got != 39726 {
		t.Fatalf("usage.total_tokens = %d, want 39726", got)
	}
	if gjson.Get(result, "usage.cache_read_input_tokens").Exists() {
		t.Fatalf("usage.cache_read_input_tokens should not be emitted: %s", result)
	}
	if got := gjson.Get(result, "usage.input_tokens_details.cached_tokens").Int(); got != 36608 {
		t.Fatalf("usage.input_tokens_details.cached_tokens = %d, want 36608", got)
	}
}

// ============== <think> 标签提取测试 ==============

// collectThinkStreamText 聚合事件中的 reasoning 与 text delta 内容。
func collectThinkStreamText(events []string) (reasoning, text string) {
	for _, ev := range events {
		for _, line := range strings.Split(ev, "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			typ := gjson.Get(payload, "type").String()
			switch typ {
			case "response.reasoning_summary_text.delta":
				reasoning += gjson.Get(payload, "text").String()
			case "response.output_text.delta":
				text += gjson.Get(payload, "delta").String()
			}
		}
	}
	return reasoning, text
}

func runThinkStream(t *testing.T, chunks []string) []string {
	t.Helper()
	ctx := context.Background()
	originalReq := []byte(`{"model":"MiniMax-M2.7","input":"hi"}`)
	var state any
	var allEvents []string
	for _, chunk := range chunks {
		events := ConvertOpenAIChatToResponses(ctx, "MiniMax-M2.7", originalReq, nil, []byte(chunk), &state)
		allEvents = append(allEvents, events...)
	}
	return allEvents
}

// TestThinkTag_StreamCrossChunkBoundary 验证跨 chunk 边界的 <think>...</think> 能被正确切分。
func TestThinkTag_StreamCrossChunkBoundary(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<thi"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"nk>思考"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"内容</thi"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"nk>正文"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "思考内容" {
		t.Errorf("reasoning = %q, want %q", reasoning, "思考内容")
	}
	if text != "正文" {
		t.Errorf("text = %q, want %q", text, "正文")
	}
	joined := strings.Join(events, "\n")
	if strings.Contains(joined, "<think>") || strings.Contains(joined, "</think>") {
		t.Errorf("events should not contain raw think tags: %s", joined)
	}
}

// TestThinkTag_StreamSingleChunk 验证单个 chunk 内含完整 <think>...</think>正文的拆分。
func TestThinkTag_StreamSingleChunk(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<think>full-think</think>answer"}}]}`,
		`data: {"id":"cc-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "full-think" {
		t.Errorf("reasoning = %q, want %q", reasoning, "full-think")
	}
	if text != "answer" {
		t.Errorf("text = %q, want %q", text, "answer")
	}
}

// TestThinkTag_StreamUnclosedFallback 验证未闭合 <think> 会被视为推理内容兜底刷出。
func TestThinkTag_StreamUnclosedFallback(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-3","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<think>仅有思考"}}]}`,
		`data: {"id":"cc-3","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "仅有思考" {
		t.Errorf("reasoning = %q, want %q", reasoning, "仅有思考")
	}
	if text != "" {
		t.Errorf("text should be empty, got %q", text)
	}
}

// TestThinkTag_StreamPlainTextWithLiteralTag 验证正文中误用 <think> 不应被剥离。
func TestThinkTag_StreamPlainTextWithLiteralTag(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-4","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello <think>not-reasoning</think>"}}]}`,
		`data: {"id":"cc-4","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "" {
		t.Errorf("reasoning should be empty when <think> is not at the start, got %q", reasoning)
	}
	if text != "Hello <think>not-reasoning</think>" {
		t.Errorf("text = %q, want %q", text, "Hello <think>not-reasoning</think>")
	}
}

// TestThinkTag_NonStreamSplit 验证非流式接口能从 content 头部提取 <think>。
func TestThinkTag_NonStreamSplit(t *testing.T) {
	ctx := context.Background()
	body := `{
		"id":"cc-ns-1",
		"object":"chat.completion",
		"created":1,
		"model":"MiniMax-M2.7",
		"choices":[{"index":0,"message":{"role":"assistant","content":"<think>think-here</think>answer-here"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}
	}`
	result := ConvertOpenAIChatToResponsesNonStream(ctx, "MiniMax-M2.7", []byte(`{"model":"MiniMax-M2.7","input":"hi"}`), nil, []byte(body), nil)

	outputs := gjson.Get(result, "output").Array()
	if len(outputs) < 2 {
		t.Fatalf("expected at least 2 outputs (reasoning + message), got %d: %s", len(outputs), result)
	}
	if outputs[0].Get("type").String() != "reasoning" {
		t.Errorf("first output type = %q, want reasoning", outputs[0].Get("type").String())
	}
	if got := outputs[0].Get("summary.0.text").String(); got != "think-here" {
		t.Errorf("reasoning text = %q, want %q", got, "think-here")
	}
	if outputs[1].Get("type").String() != "message" {
		t.Errorf("second output type = %q, want message", outputs[1].Get("type").String())
	}
	if got := outputs[1].Get("content.0.text").String(); got != "answer-here" {
		t.Errorf("message text = %q, want %q", got, "answer-here")
	}
}

// TestThinkTag_StreamThenToolCalls 验证 <think>...</think> 后紧跟 tool_calls 的 SSE：
//
//   - reasoning 内容应正确进入 reasoning_summary_text.delta
//   - function_call 的 output_index 计算应正确（reasoning 占位后的 fc 索引应 +1）
//   - 不应有任何 response.output_text.delta（content 全部被 think 吸收）
//   - 不应留下空的 message item
func TestThinkTag_StreamThenToolCalls(t *testing.T) {
	chunks := []string{
		// 第一帧：role + 完整 <think>...</think>
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<think>let me call a tool</think>"}}]}`,
		// 第二帧：tool_calls 的开头（id + name）
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_tt","type":"function","function":{"name":"lookup","arguments":""}}]}}]}`,
		// 第三帧：tool_calls 的 arguments 分片
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":\"x\"}"}}]}}]}`,
		// 第四帧：finish
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "let me call a tool" {
		t.Errorf("reasoning = %q, want %q", reasoning, "let me call a tool")
	}
	if text != "" {
		t.Errorf("text should be empty when content is entirely <think>, got %q", text)
	}

	// 解析关键事件
	var (
		reasoningAddedIdx     = -1
		funcCallAddedIdx      = -1
		funcCallArgsDelta     string
		funcCallOutputIndex   = -1
		funcCallItemID        string
		funcCallArgsOutputIdx = -1
		sawOutputTextDelta    = false
		sawEmptyMessage       = false
	)

	for _, ev := range events {
		for _, line := range strings.Split(ev, "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			typ := gjson.Get(payload, "type").String()
			switch typ {
			case "response.output_item.added":
				itemType := gjson.Get(payload, "item.type").String()
				outIdx := int(gjson.Get(payload, "output_index").Int())
				if itemType == "reasoning" && reasoningAddedIdx < 0 {
					reasoningAddedIdx = outIdx
				}
				if itemType == "function_call" && funcCallAddedIdx < 0 {
					funcCallAddedIdx = outIdx
					funcCallItemID = gjson.Get(payload, "item.id").String()
				}
				if itemType == "message" {
					// 状态机吸收完 <think> 后正文为空 → 不应发射 message item
					sawEmptyMessage = true
				}
			case "response.output_text.delta":
				sawOutputTextDelta = true
			case "response.function_call_arguments.delta":
				funcCallArgsDelta += gjson.Get(payload, "delta").String()
				funcCallArgsOutputIdx = int(gjson.Get(payload, "output_index").Int())
			case "response.output_item.done":
				if gjson.Get(payload, "item.type").String() == "function_call" && funcCallOutputIndex < 0 {
					funcCallOutputIndex = int(gjson.Get(payload, "output_index").Int())
				}
			}
		}
	}

	if reasoningAddedIdx != 0 {
		t.Errorf("reasoning output_index = %d, want 0", reasoningAddedIdx)
	}
	if funcCallAddedIdx != 1 {
		t.Errorf("function_call output_index = %d, want 1 (reasoning + fc)", funcCallAddedIdx)
	}
	if funcCallArgsOutputIdx != 1 {
		t.Errorf("function_call_arguments.delta output_index = %d, want 1", funcCallArgsOutputIdx)
	}
	if funcCallArgsDelta != `{"q":"x"}` {
		t.Errorf("function arguments = %q, want %q", funcCallArgsDelta, `{"q":"x"}`)
	}
	if funcCallItemID == "" {
		t.Error("function_call item.id should not be empty")
	}
	if sawOutputTextDelta {
		t.Error("should not emit response.output_text.delta when content is fully absorbed by <think>")
	}
	if sawEmptyMessage {
		t.Error("should not emit empty message item when content is fully absorbed by <think>")
	}
}

// TestThinkTag_StreamWithLeadingWhitespace 验证流式开头有空白字符时，<think> 仍能被正确提取。
func TestThinkTag_StreamWithLeadingWhitespace(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-ws-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"  \n  <thi"}}]}`,
		`data: {"id":"cc-ws-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"nk>思考内容</think>正文"}}]}`,
		`data: {"id":"cc-ws-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "思考内容" {
		t.Errorf("reasoning = %q, want %q", reasoning, "思考内容")
	}
	if text != "正文" {
		t.Errorf("text = %q, want %q", text, "正文")
	}
}

// TestThinkTag_StreamWithLeadingWhitespaceNoThink 验证流式开头有空白字符但最终没有 <think> 时，空白字符能被正确作为正文输出。
func TestThinkTag_StreamWithLeadingWhitespaceNoThink(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-ws-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"  \n  "}}]}`,
		`data: {"id":"cc-ws-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"正文"}}]}`,
		`data: {"id":"cc-ws-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "" {
		t.Errorf("reasoning should be empty, got %q", reasoning)
	}
	if text != "  \n  正文" {
		t.Errorf("text = %q, want %q", text, "  \n  正文")
	}
}

// TestExtractThinkTag_TableDriven 覆盖共享函数 extractThinkTag 的核心分支。
func TestExtractThinkTag_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantText  string
		wantThink string
		wantHas   bool
	}{
		{"no think", "plain answer", "plain answer", "", false},
		{"think at start", "<think>t</think>a", "a", "t", true},
		{"think with leading whitespace", "  \n<think>t</think>a", "a", "t", true},
		{"unclosed think", "<think>only thinking", "", "only thinking", true},
		{"think in middle", "head <think>x</think>tail", "head <think>x</think>tail", "", false},
		{"empty input", "", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotText, gotThink, gotHas := extractThinkTag(tc.input)
			if gotText != tc.wantText || gotThink != tc.wantThink || gotHas != tc.wantHas {
				t.Errorf("extractThinkTag(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.input, gotText, gotThink, gotHas, tc.wantText, tc.wantThink, tc.wantHas)
			}
		})
	}
}
