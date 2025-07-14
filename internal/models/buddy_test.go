package models

import (
	"testing"
	"time"
)

func TestRule(t *testing.T) {
	rule := Rule{
		ID:          "test-id",
		Category:    "test-category",
		Title:       "Test Rule",
		Description: "This is a test rule",
		Priority:    "critical",
		Content:     "Test content",
		FilePath:    "/test/path",
		UpdatedAt:   time.Now(),
	}

	if rule.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got %s", rule.ID)
	}

	if rule.Priority != "critical" {
		t.Errorf("Expected Priority to be 'critical', got %s", rule.Priority)
	}
}

func TestKnowledge(t *testing.T) {
	knowledge := Knowledge{
		ID:        "knowledge-id",
		Title:     "Test Knowledge",
		Category:  "Architecture",
		Content:   "Knowledge content",
		Tags:      []string{"test", "architecture"},
		FilePath:  "/test/knowledge.md",
		UpdatedAt: time.Now(),
	}

	if len(knowledge.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(knowledge.Tags))
	}

	if knowledge.Tags[0] != "test" {
		t.Errorf("Expected first tag to be 'test', got %s", knowledge.Tags[0])
	}
}

func TestTable(t *testing.T) {
	table := Table{
		Name:        "users",
		Schema:      "public",
		Description: "User table",
		Columns: []Column{
			{
				Name:         "id",
				Type:         "UUID",
				Nullable:     false,
				DefaultValue: "gen_random_uuid()",
				Description:  "Primary key",
			},
		},
		Indexes: []Index{
			{
				Name:    "users_pkey",
				Columns: []string{"id"},
				Unique:  true,
			},
		},
	}

	if table.Name != "users" {
		t.Errorf("Expected table name to be 'users', got %s", table.Name)
	}

	if len(table.Columns) != 1 {
		t.Errorf("Expected 1 column, got %d", len(table.Columns))
	}

	if table.Columns[0].Name != "id" {
		t.Errorf("Expected column name to be 'id', got %s", table.Columns[0].Name)
	}
}

func TestTodo(t *testing.T) {
	todo := Todo{
		ID:         "todo-1",
		Feature:    "Authentication",
		Task:       "Implement login",
		Completed:  false,
		FilePath:   "/todos/auth.md",
		LineNumber: 5,
		UpdatedAt:  time.Now(),
	}

	if todo.Completed {
		t.Error("Expected todo to be incomplete")
	}

	if todo.Feature != "Authentication" {
		t.Errorf("Expected feature to be 'Authentication', got %s", todo.Feature)
	}
}

func TestHistoryEntry(t *testing.T) {
	changes := []Change{
		{
			FilePath:   "/test/file.go",
			ChangeType: "modified",
			Before:     "old content",
			After:      "new content",
		},
	}

	entry := HistoryEntry{
		ID:          "history-1",
		Timestamp:   time.Now(),
		Feature:     "Authentication",
		Description: "Added login functionality",
		Changes:     changes,
		Reasoning:   "User requested feature",
		FilePath:    "/history/auth.json",
	}

	if len(entry.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(entry.Changes))
	}

	if entry.Changes[0].ChangeType != "modified" {
		t.Errorf("Expected change type to be 'modified', got %s", entry.Changes[0].ChangeType)
	}
}

func TestBackup(t *testing.T) {
	backup := Backup{
		ID:            "backup-1",
		OriginalPath:  "/test/file.go",
		BackupPath:    "/backups/file.go.bak",
		Timestamp:     time.Now(),
		ChangeContext: "Before refactoring",
		Reasoning:     "Safety backup",
		FileSize:      1024,
	}

	if backup.FileSize != 1024 {
		t.Errorf("Expected file size to be 1024, got %d", backup.FileSize)
	}

	if backup.ChangeContext != "Before refactoring" {
		t.Errorf("Expected change context to be 'Before refactoring', got %s", backup.ChangeContext)
	}
}

func TestProjectContext(t *testing.T) {
	rules := []Rule{
		{ID: "rule-1", Title: "Test Rule", Priority: "critical"},
	}

	knowledge := []Knowledge{
		{ID: "kb-1", Title: "Test Knowledge", Category: "Architecture"},
	}

	todos := []Todo{
		{ID: "todo-1", Feature: "Auth", Task: "Login", Completed: false},
	}

	context := ProjectContext{
		ProjectName:   "test-project",
		Rules:         rules,
		Knowledge:     knowledge,
		Todos:         todos,
		RecentHistory: []HistoryEntry{},
		UpdatedAt:     time.Now(),
	}

	if context.ProjectName != "test-project" {
		t.Errorf("Expected project name to be 'test-project', got %s", context.ProjectName)
	}

	if len(context.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(context.Rules))
	}

	if len(context.Knowledge) != 1 {
		t.Errorf("Expected 1 knowledge entry, got %d", len(context.Knowledge))
	}

	if len(context.Todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(context.Todos))
	}
}
