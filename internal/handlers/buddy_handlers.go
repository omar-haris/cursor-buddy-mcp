package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/omar-haris/cursor-buddy-mcp/internal/search"
)

// marshalFunc is a test hook for json.Marshal
var marshalFunc = json.Marshal

// BuddyHandlers manages all buddy system handlers
type BuddyHandlers struct {
	buddyPath        string
	searchManager    *search.SearchManager
	rulesHandler     *RulesHandler
	knowledgeHandler *KnowledgeHandler
	databaseHandler  *DatabaseHandler
	todoHandler      *TodoHandler
	historyHandler   *HistoryHandler
	backupHandler    *BackupHandler
	mu               sync.RWMutex
}

// NewBuddyHandlers creates a new instance of BuddyHandlers
func NewBuddyHandlers(buddyPath string) (*BuddyHandlers, error) {
	// Create buddy directory structure if it doesn't exist
	if err := createBuddyStructure(buddyPath); err != nil {
		return nil, fmt.Errorf("failed to create buddy structure: %w", err)
	}

	// Initialize search manager
	searchManager, err := search.NewSearchManager(buddyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create search manager: %w", err)
	}

	bh := &BuddyHandlers{
		buddyPath:     buddyPath,
		searchManager: searchManager,
	}

	// Initialize all handlers with search manager
	bh.rulesHandler = NewRulesHandler(filepath.Join(buddyPath, "rules"), searchManager)
	bh.knowledgeHandler = NewKnowledgeHandler(filepath.Join(buddyPath, "knowledge"), searchManager)
	bh.databaseHandler = NewDatabaseHandler(filepath.Join(buddyPath, "database"), searchManager)
	bh.todoHandler = NewTodoHandler(filepath.Join(buddyPath, "todos"), searchManager)
	bh.historyHandler = NewHistoryHandler(filepath.Join(buddyPath, "history"), searchManager)
	bh.backupHandler = NewBackupHandler(filepath.Join(buddyPath, "backups"), searchManager)

	// Load initial data
	if err := bh.loadAllData(); err != nil {
		return nil, fmt.Errorf("failed to load initial data: %w", err)
	}

	return bh, nil
}

// createBuddyStructure creates the necessary directory structure
func createBuddyStructure(buddyPath string) error {
	dirs := []string{
		"rules",
		"knowledge",
		"todos",
		"database",
		"history",
		"backups",
		"indexes", // For Bleve indexes
	}

	for _, dir := range dirs {
		path := filepath.Join(buddyPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	return nil
}

// loadAllData loads all data from disk
func (bh *BuddyHandlers) loadAllData() error {
	// Load rules
	if err := bh.rulesHandler.Load(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	// Load knowledge
	if err := bh.knowledgeHandler.Load(); err != nil {
		return fmt.Errorf("failed to load knowledge: %w", err)
	}

	// Load database
	if err := bh.databaseHandler.Load(); err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Load todos
	if err := bh.todoHandler.Load(); err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	// Load history
	if err := bh.historyHandler.Load(); err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Load backups
	if err := bh.backupHandler.Load(); err != nil {
		return fmt.Errorf("failed to load backups: %w", err)
	}

	return nil
}

// ReloadData reloads data when files change
func (bh *BuddyHandlers) ReloadData() error {
	return bh.loadAllData()
}

// GetRulesToolHandler returns the tool handler for rules management
func (bh *BuddyHandlers) GetRulesToolHandler() server.ToolHandlerFunc {
	return bh.rulesHandler.GetToolHandler()
}

// GetKnowledgeToolHandler returns the tool handler for knowledge base
func (bh *BuddyHandlers) GetKnowledgeToolHandler() server.ToolHandlerFunc {
	return bh.knowledgeHandler.GetToolHandler()
}

// GetDatabaseToolHandler returns the tool handler for database management
func (bh *BuddyHandlers) GetDatabaseToolHandler() server.ToolHandlerFunc {
	return bh.databaseHandler.GetToolHandler()
}

// GetTodoToolHandler returns the tool handler for todo management
func (bh *BuddyHandlers) GetTodoToolHandler() server.ToolHandlerFunc {
	return bh.todoHandler.GetToolHandler()
}

// GetHistoryToolHandler returns the tool handler for history tracking
func (bh *BuddyHandlers) GetHistoryToolHandler() server.ToolHandlerFunc {
	return bh.historyHandler.GetToolHandler()
}

// GetBackupToolHandler returns the tool handler for backup management
func (bh *BuddyHandlers) GetBackupToolHandler() server.ToolHandlerFunc {
	return bh.backupHandler.GetToolHandler()
}

// GetProjectContextResourceHandler returns the resource handler for project context
func (bh *BuddyHandlers) GetProjectContextResourceHandler() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Gather all project context
		projectContext := map[string]interface{}{
			"rules":     bh.rulesHandler.GetRules(),
			"knowledge": bh.knowledgeHandler.GetKnowledge(),
			"todos":     bh.todoHandler.GetTodos(),
			"database":  bh.databaseHandler.GetDatabaseInfo(),
			"history":   bh.historyHandler.GetRecentHistory(10),
		}

		// Marshal to JSON
		data, err := marshalFunc(projectContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "buddy://project-context",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

// Close closes all resources including the search manager
func (bh *BuddyHandlers) Close() error {
	if bh.searchManager != nil {
		return bh.searchManager.Close()
	}
	return nil
}
