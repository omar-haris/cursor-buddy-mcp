package search

import (
	"strings"
	"time"

	"github.com/omar-haris/cursor-buddy-mcp/internal/models"
)

// RuleDocument represents a rule document for indexing
type RuleDocument struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Content     string `json:"content"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
}

// FromRule creates a RuleDocument from a models.Rule
func FromRule(rule models.Rule) RuleDocument {
	return RuleDocument{
		ID:          rule.ID,
		Title:       rule.Title,
		Category:    rule.Category,
		Content:     rule.Content,
		Priority:    rule.Priority,
		Description: rule.Description,
	}
}

// KnowledgeDocument represents a knowledge document for indexing
type KnowledgeDocument struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Content  string `json:"content"`
	Tags     string `json:"tags"` // Comma-separated for better search
}

// FromKnowledge creates a KnowledgeDocument from a models.Knowledge
func FromKnowledge(knowledge models.Knowledge) KnowledgeDocument {
	return KnowledgeDocument{
		ID:       knowledge.ID,
		Title:    knowledge.Title,
		Category: knowledge.Category,
		Content:  knowledge.Content,
		Tags:     strings.Join(knowledge.Tags, ", "),
	}
}

// TodoDocument represents a todo document for indexing
type TodoDocument struct {
	ID        string `json:"id"`
	Task      string `json:"task"`
	Feature   string `json:"feature"`
	Completed bool   `json:"completed"`
	Status    string `json:"status"` // "completed" or "pending" for text search
}

// FromTodo creates a TodoDocument from a models.Todo
func FromTodo(todo models.Todo) TodoDocument {
	status := "pending"
	if todo.Completed {
		status = "completed"
	}

	return TodoDocument{
		ID:        todo.ID,
		Task:      todo.Task,
		Feature:   todo.Feature,
		Completed: todo.Completed,
		Status:    status,
	}
}

// HistoryDocument represents a history document for indexing
type HistoryDocument struct {
	ID          string    `json:"id"`
	Feature     string    `json:"feature"`
	Description string    `json:"description"`
	Reasoning   string    `json:"reasoning"`
	Files       string    `json:"files"` // Comma-separated file paths
	Timestamp   time.Time `json:"timestamp"`
}

// FromHistoryEntry creates a HistoryDocument from a models.HistoryEntry
func FromHistoryEntry(entry models.HistoryEntry) HistoryDocument {
	var files []string
	for _, change := range entry.Changes {
		files = append(files, change.FilePath)
	}

	return HistoryDocument{
		ID:          entry.ID,
		Feature:     entry.Feature,
		Description: entry.Description,
		Reasoning:   entry.Reasoning,
		Files:       strings.Join(files, ", "),
		Timestamp:   entry.Timestamp,
	}
}

// DatabaseDocument represents a database table document for indexing
type DatabaseDocument struct {
	ID          string `json:"id"`
	TableName   string `json:"table_name"`
	Columns     string `json:"columns"` // Comma-separated column names
	Indexes     string `json:"indexes"` // Comma-separated index names
	Description string `json:"description"`
}

// FromTable creates a DatabaseDocument from a models.Table
func FromTable(table models.Table) DatabaseDocument {
	var columnNames []string
	for _, col := range table.Columns {
		columnNames = append(columnNames, col.Name+" "+col.Type)
	}

	var indexNames []string
	for _, idx := range table.Indexes {
		indexNames = append(indexNames, idx.Name)
	}

	return DatabaseDocument{
		ID:          table.Name, // Use table name as ID
		TableName:   table.Name,
		Columns:     strings.Join(columnNames, ", "),
		Indexes:     strings.Join(indexNames, ", "),
		Description: table.Description,
	}
}

// BackupDocument represents a backup document for indexing
type BackupDocument struct {
	ID           string    `json:"id"`
	OriginalPath string    `json:"original_path"`
	Context      string    `json:"context"`
	Reasoning    string    `json:"reasoning"`
	Timestamp    time.Time `json:"timestamp"`
}

// FromBackup creates a BackupDocument from a models.Backup
func FromBackup(backup models.Backup) BackupDocument {
	return BackupDocument{
		ID:           backup.ID,
		OriginalPath: backup.OriginalPath,
		Context:      backup.ChangeContext,
		Reasoning:    backup.Reasoning,
		Timestamp:    backup.Timestamp,
	}
}
