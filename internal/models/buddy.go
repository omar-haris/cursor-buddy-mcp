package models

import "time"

// Rule represents a coding rule or guideline
type Rule struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"` // critical, recommended, optional
	Content     string    `json:"content"`
	FilePath    string    `json:"file_path"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Knowledge represents a knowledge base entry
type Knowledge struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	FilePath  string    `json:"file_path"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DatabaseInfo represents database schema and connection information
type DatabaseInfo struct {
	Type           string    `json:"type"`
	SchemaPath     string    `json:"schema_path"`
	ERDPath        string    `json:"erd_path"`
	ConnectionInfo string    `json:"connection_info"`
	Tables         []Table   `json:"tables"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Table represents a database table
type Table struct {
	Name        string   `json:"name"`
	Schema      string   `json:"schema"`
	Columns     []Column `json:"columns"`
	Indexes     []Index  `json:"indexes"`
	Description string   `json:"description"`
}

// Column represents a database column
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value"`
	Description  string `json:"description"`
}

// Index represents a database index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// Todo represents a task item
type Todo struct {
	ID         string    `json:"id"`
	Feature    string    `json:"feature"`
	Task       string    `json:"task"`
	Completed  bool      `json:"completed"`
	FilePath   string    `json:"file_path"`
	LineNumber int       `json:"line_number"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// HistoryEntry represents a change history record
type HistoryEntry struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Feature     string    `json:"feature"`
	Description string    `json:"description"`
	Changes     []Change  `json:"changes"`
	Reasoning   string    `json:"reasoning"`
	FilePath    string    `json:"file_path"`
}

// Change represents a single file change
type Change struct {
	FilePath   string `json:"file_path"`
	ChangeType string `json:"change_type"` // created, modified, deleted
	Before     string `json:"before"`
	After      string `json:"after"`
}

// Backup represents a file backup
type Backup struct {
	ID            string    `json:"id"`
	OriginalPath  string    `json:"original_path"`
	BackupPath    string    `json:"backup_path"`
	Timestamp     time.Time `json:"timestamp"`
	ChangeContext string    `json:"change_context"`
	Reasoning     string    `json:"reasoning"`
	FileSize      int64     `json:"file_size"`
}

// ProjectContext represents the overall project context
type ProjectContext struct {
	ProjectName   string         `json:"project_name"`
	Rules         []Rule         `json:"rules"`
	Knowledge     []Knowledge    `json:"knowledge"`
	Database      *DatabaseInfo  `json:"database,omitempty"`
	Todos         []Todo         `json:"todos"`
	RecentHistory []HistoryEntry `json:"recent_history"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
