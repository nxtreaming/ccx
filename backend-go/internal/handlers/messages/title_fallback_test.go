package messages

import (
	"testing"

	"github.com/BenedictKing/ccx/internal/types"
)

func TestExtractLastUserMessageSkipsSystemReminder(t *testing.T) {
	messages := []types.ClaudeMessage{{
		Role: "user",
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "<system-reminder>\n# MCP Server Instructions\n</system-reminder>\n真实用户问题",
			},
		},
	}}

	got := extractLastUserMessage(messages)
	if got != "真实用户问题" {
		t.Fatalf("extractLastUserMessage() = %q, want %q", got, "真实用户问题")
	}
}

func TestExtractLastUserMessageSkipsInjectedAgentsInstructions(t *testing.T) {
	messages := []types.ClaudeMessage{{
		Role: "user",
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "# AGENTS.md instructions for /Users/example/project\n<INSTRUCTIONS>\nAlways respond in Chinese\n</INSTRUCTIONS>",
			},
			map[string]interface{}{
				"type": "text",
				"text": "这个展开的对话卡片应该优化",
			},
		},
	}}

	got := extractLastUserMessage(messages)
	if got != "这个展开的对话卡片应该优化" {
		t.Fatalf("extractLastUserMessage() = %q, want %q", got, "这个展开的对话卡片应该优化")
	}
}

func TestExtractLastUserMessageSkipsClaudeCommandDocs(t *testing.T) {
	messages := []types.ClaudeMessage{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "<system-reminder>\ncontext\n</system-reminder>\n",
				},
				map[string]interface{}{
					"type": "text",
					"text": "提交",
				},
			},
		},
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "# Claude Command: Commit (Git-only)\n\n## Usage\n\n/git-commit\n\n### Options\n\n- --no-verify",
				},
			},
		},
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "继续",
				},
			},
		},
	}

	got := extractLastUserMessage(messages)
	want := "提交 / 继续"
	if got != want {
		t.Fatalf("extractLastUserMessage() = %q, want %q", got, want)
	}
}

func TestExtractLastUserMessageIgnoresOnlyTaggedContent(t *testing.T) {
	messages := []types.ClaudeMessage{{
		Role:    "user",
		Content: "<system-reminder>\n# MCP Server Instructions\n</system-reminder>",
	}}

	if got := extractLastUserMessage(messages); got != "" {
		t.Fatalf("extractLastUserMessage() = %q, want empty", got)
	}
}

func TestExtractLastUserMessageSkipsClaudeMetaRetryPrompt(t *testing.T) {
	messages := []types.ClaudeMessage{
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "真实用户问题"}}},
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "[Your previous response had no visible output. Please continue and produce a user-visible response.]"}}},
	}

	got := extractLastUserMessage(messages)
	if got != "真实用户问题" {
		t.Fatalf("extractLastUserMessage() = %q, want %q", got, "真实用户问题")
	}
	if got := countUserMessages(messages); got != 1 {
		t.Fatalf("countUserMessages() = %d, want 1", got)
	}
}

func TestExtractLastUserMessageJoinsRecentShortInputs(t *testing.T) {
	messages := []types.ClaudeMessage{
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "第一个短问题"}}},
		{Role: "assistant", Content: []interface{}{map[string]interface{}{"type": "text", "text": "回答"}}},
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "第二个短问题"}}},
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "tool_result", "content": "工具结果"}}},
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "最后一个短问题"}}},
	}

	got := extractLastUserMessage(messages)
	want := "第一个短问题 / 第二个短问题 / 最后一个短问题"
	if got != want {
		t.Fatalf("extractLastUserMessage() = %q, want %q", got, want)
	}
}

func TestCountUserMessagesIgnoresToolResults(t *testing.T) {
	messages := []types.ClaudeMessage{
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "用户问题"}}},
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "tool_result", "content": "工具结果"}}},
	}

	if got := countUserMessages(messages); got != 1 {
		t.Fatalf("countUserMessages() = %d, want 1", got)
	}
}

func TestClaudeCodeRecapRequestIsDetectedAndSkippedFromPreview(t *testing.T) {
	const recapPrompt = "The user stepped away and is coming back. Recap in under 40 words, 1-2 plain sentences, no markdown."
	body := []byte(`{
		"context_management":{"edits":[{"keep":"all","type":"clear_thinking_20251015"}]},
		"messages":[
			{"role":"user","content":[{"type":"text","text":"真实用户问题"}]},
			{"role":"assistant","content":[{"type":"text","text":"回答"}]},
			{"role":"user","content":"` + recapPrompt + `"}
		]
	}`)

	if !isClaudeCodeRecapRequest(body) {
		t.Fatal("expected recap request to be detected")
	}

	messages := []types.ClaudeMessage{
		{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "真实用户问题"}}},
		{Role: "assistant", Content: []interface{}{map[string]interface{}{"type": "text", "text": "回答"}}},
		{Role: "user", Content: recapPrompt},
	}
	if got := extractLastUserMessage(messages); got != "真实用户问题" {
		t.Fatalf("extractLastUserMessage() = %q, want %q", got, "真实用户问题")
	}
	if got := countUserMessages(messages); got != 1 {
		t.Fatalf("countUserMessages() = %d, want 1", got)
	}
}
