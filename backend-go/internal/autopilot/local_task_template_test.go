package autopilot

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestTemplateStore(t *testing.T) *LocalTaskTemplateStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewLocalTaskTemplateStoreWithDB(db)
	if err != nil {
		t.Fatalf("创建 LocalTaskTemplateStore 失败: %v", err)
	}
	return store
}

func TestLocalTaskTemplate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    LocalTaskTemplate
		wantErr string
	}{
		{
			name: "valid template",
			tmpl: LocalTaskTemplate{
				Name:           "Code Summary",
				OutputMode:     OutputModeSummarize,
				PromptTemplate: "Summarize: {{original_content}}",
				Priority:       10,
			},
			wantErr: "",
		},
		{
			name: "empty name",
			tmpl: LocalTaskTemplate{
				OutputMode:     OutputModeSummarize,
				PromptTemplate: "Summarize: {{original_content}}",
			},
			wantErr: "name 不能为空",
		},
		{
			name: "empty prompt",
			tmpl: LocalTaskTemplate{
				Name:       "Test",
				OutputMode: OutputModeSummarize,
			},
			wantErr: "promptTemplate 不能为空",
		},
		{
			name: "invalid output mode",
			tmpl: LocalTaskTemplate{
				Name:           "Test",
				OutputMode:     "invalid",
				PromptTemplate: "test",
			},
			wantErr: "outputMode 无效",
		},
		{
			name: "negative priority",
			tmpl: LocalTaskTemplate{
				Name:           "Test",
				OutputMode:     OutputModeRewrite,
				PromptTemplate: "test",
				Priority:       -1,
			},
			wantErr: "priority 不能为负数",
		},
		{
			name: "invalid task class",
			tmpl: LocalTaskTemplate{
				Name:             "Test",
				OutputMode:       OutputModeExtract,
				PromptTemplate:   "test",
				MatchTaskClasses: []string{"invalid_class"},
			},
			wantErr: "matchTaskClasses 包含无效值",
		},
		{
			name: "invalid domain",
			tmpl: LocalTaskTemplate{
				Name:           "Test",
				OutputMode:     OutputModeSummarize,
				PromptTemplate: "test",
				MatchDomains:   []string{"invalid_domain"},
			},
			wantErr: "matchDomains 包含无效值",
		},
		{
			name: "valid with filters",
			tmpl: LocalTaskTemplate{
				Name:             "Worker Coding",
				OutputMode:       OutputModeSummarize,
				PromptTemplate:   "Code summary: {{original_content}}",
				MatchTaskClasses: []string{"worker", "supervisor"},
				MatchDomains:     []string{"coding", "code_review"},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("Validate() expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Validate() error = %q, want containing %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

// PLACEHOLDER_MORE_TESTS

func TestIsValidOutputMode(t *testing.T) {
	tests := []struct {
		mode OutputMode
		want bool
	}{
		{OutputModeSummarize, true},
		{OutputModeRewrite, true},
		{OutputModeExtract, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsValidOutputMode(tt.mode)
		if got != tt.want {
			t.Errorf("IsValidOutputMode(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter []string
		value  string
		want   bool
	}{
		{"nil filter matches anything", nil, "worker", true},
		{"nil filter matches empty", nil, "", true},
		{"empty filter matches anything", []string{}, "worker", true},
		{"filter matches value", []string{"worker", "supervisor"}, "worker", true},
		{"filter matches case insensitive", []string{"Worker"}, "worker", true},
		{"filter does not match", []string{"worker"}, "supervisor", false},
		{"non-empty filter does not match empty", []string{"worker"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(tt.filter, tt.value)
			if got != tt.want {
				t.Errorf("matchesFilter(%v, %q) = %v, want %v", tt.filter, tt.value, got, tt.want)
			}
		})
	}
}

func TestLocalTaskTemplateStore_CRUD(t *testing.T) {
	store := newTestTemplateStore(t)

	// 初始为空
	all := store.ListAll()
	if len(all) != 0 {
		t.Fatalf("expected 0 templates, got %d", len(all))
	}

	// 创建
	enabled := true
	tmpl := &LocalTaskTemplate{
		TemplateID:       GenerateTemplateID(),
		Name:             "Test Template",
		OutputMode:       OutputModeSummarize,
		PromptTemplate:   "Summarize this: {{original_content}}",
		MatchTaskClasses: []string{"worker"},
		Priority:         10,
		Enabled:          enabled,
	}
	if err := store.Upsert(tmpl); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Get
	got := store.Get(tmpl.TemplateID)
	if got == nil {
		t.Fatal("Get returned nil after Upsert")
	}
	if got.Name != "Test Template" {
		t.Fatalf("name = %q, want %q", got.Name, "Test Template")
	}

	// List
	all = store.ListAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 template, got %d", len(all))
	}

	// Update
	got.Name = "Updated Template"
	if err := store.Upsert(got); err != nil {
		t.Fatalf("Upsert update failed: %v", err)
	}
	updated := store.Get(tmpl.TemplateID)
	if updated.Name != "Updated Template" {
		t.Fatalf("updated name = %q, want %q", updated.Name, "Updated Template")
	}

	// Delete
	if err := store.Delete(tmpl.TemplateID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if store.Get(tmpl.TemplateID) != nil {
		t.Fatal("Get should return nil after Delete")
	}
}

func TestLocalTaskTemplateStore_FindBestPrompt(t *testing.T) {
	store := newTestTemplateStore(t)

	// 创建多个模板
	templates := []*LocalTaskTemplate{
		{
			TemplateID:       "tpl_coding",
			Name:             "Coding Summary",
			OutputMode:       OutputModeSummarize,
			PromptTemplate:   "Coding summary: {{original_content}}",
			MatchTaskClasses: []string{"worker"},
			MatchDomains:     []string{"coding"},
			Priority:         10,
			Enabled:          true,
		},
		{
			TemplateID:       "tpl_default",
			Name:             "Default",
			OutputMode:       OutputModeSummarize,
			PromptTemplate:   "Default summary: {{original_content}}",
			Priority:         0,
			Enabled:          true,
		},
		{
			TemplateID:       "tpl_disabled",
			Name:             "Disabled",
			OutputMode:       OutputModeRewrite,
			PromptTemplate:   "Should not match: {{original_content}}",
			MatchTaskClasses: []string{"worker"},
			Priority:         100, // 最高优先级，但禁用
			Enabled:          false,
		},
		{
			TemplateID:       "tpl_code_review",
			Name:             "Code Review",
			OutputMode:       OutputModeExtract,
			PromptTemplate:   "Review: {{original_content}}",
			MatchTaskClasses: []string{"supervisor", "worker"},
			MatchDomains:     []string{"code_review"},
			Priority:         5,
			Enabled:          true,
		},
	}

	for _, tmpl := range templates {
		if err := store.Upsert(tmpl); err != nil {
			t.Fatalf("Upsert(%s) failed: %v", tmpl.TemplateID, err)
		}
	}

	transcript := "Hello, this is the conversation."

	tests := []struct {
		name     string
		taskClass string
		domain   string
		wantPrefix string
		wantEmpty bool
	}{
		{
			name:       "exact match: worker + coding -> coding template (highest priority)",
			taskClass:  "worker",
			domain:     "coding",
			wantPrefix: "Coding summary:",
		},
		{
			name:       "supervisor + code_review -> code_review template",
			taskClass:  "supervisor",
			domain:     "code_review",
			wantPrefix: "Review:",
		},
		{
			name:       "vision + general -> default (no class/domain filter)",
			taskClass:  "vision",
			domain:     "general",
			wantPrefix: "Default summary:",
		},
		{
			name:       "disabled template never matches",
			taskClass:  "worker",
			domain:     "coding",
			wantPrefix: "Coding summary:", // coding template, not disabled
		},
		{
			name:       "no match -> empty (all have non-empty filters except default)",
			taskClass:  "",
			domain:     "",
			wantPrefix: "Default summary:", // default has no filters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.FindBestPrompt(tt.taskClass, tt.domain, transcript)
			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("FindBestPrompt() = %q, want empty", got)
				}
				return
			}
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Fatalf("FindBestPrompt() prefix = %q, want %q", got[:min(50, len(got))], tt.wantPrefix)
			}
			// 验证变量替换生效
			if !strings.Contains(got, transcript) {
				t.Fatalf("FindBestPrompt() should contain transcript content")
			}
		})
	}
}

func TestLocalTaskTemplateStore_Persistence(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	defer db.Close()

	// 创建并写入
	store1, err := NewLocalTaskTemplateStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewLocalTaskTemplateStoreWithDB (1st) failed: %v", err)
	}

	tmpl := &LocalTaskTemplate{
		TemplateID:     "tpl_persist",
		Name:           "Persistent",
		OutputMode:     OutputModeSummarize,
		PromptTemplate: "Persist: {{original_content}}",
		Priority:       5,
		Enabled:        true,
	}
	if err := store1.Upsert(tmpl); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// 新 store 从同一 db 加载，应能看到
	store2, err := NewLocalTaskTemplateStoreWithDB(db)
	if err != nil {
		t.Fatalf("NewLocalTaskTemplateStoreWithDB (2nd) failed: %v", err)
	}

	got := store2.Get("tpl_persist")
	if got == nil {
		t.Fatal("store2 should have loaded persisted template")
	}
	if got.Name != "Persistent" {
		t.Fatalf("persisted name = %q, want %q", got.Name, "Persistent")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
