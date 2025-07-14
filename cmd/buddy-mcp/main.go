package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/omar-haris/cursor-buddy-mcp/internal/handlers"
	"github.com/omar-haris/cursor-buddy-mcp/internal/monitor"
)

// runServer contains the main server logic that can be tested
func runServer(ctx context.Context, buddyPath string) error {
	// Initialize the buddy handlers
	buddyHandlers, err := handlers.NewBuddyHandlers(buddyPath)
	if err != nil {
		return fmt.Errorf("failed to initialize buddy handlers: %w", err)
	}

	// Start file monitoring
	fileMonitor := monitor.NewFileMonitor(buddyPath, buddyHandlers)
	go fileMonitor.Start(ctx)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"Cursor Buddy MCP",
		"1.0.0",
	)

	// Register tool handlers
	// Rules tool
	rulesTool := mcp.NewTool("buddy_get_rules",
		mcp.WithDescription("Get coding rules and guidelines from the project's buddy system"),
		mcp.WithString("category",
			mcp.Description("Filter rules by category (optional)"),
		),
		mcp.WithString("priority",
			mcp.Description("Filter rules by priority: critical, recommended, optional (optional)"),
			mcp.Enum("critical", "recommended", "optional"),
		),
	)
	mcpServer.AddTool(rulesTool, buddyHandlers.GetRulesToolHandler())

	// Knowledge search tool
	knowledgeTool := mcp.NewTool("buddy_search_knowledge",
		mcp.WithDescription("Search the project knowledge base for context and documentation"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query to find relevant knowledge"),
		),
		mcp.WithString("category",
			mcp.Description("Filter by category (optional)"),
		),
	)
	mcpServer.AddTool(knowledgeTool, buddyHandlers.GetKnowledgeToolHandler())

	// Database info tool
	databaseTool := mcp.NewTool("buddy_get_database_info",
		mcp.WithDescription("Get database schema and connection information"),
		mcp.WithString("table_name",
			mcp.Description("Get info for specific table (optional)"),
		),
		mcp.WithString("validate_query",
			mcp.Description("SQL query to validate against schema (optional)"),
		),
	)
	mcpServer.AddTool(databaseTool, buddyHandlers.GetDatabaseToolHandler())

	// Todo management tool
	todoTool := mcp.NewTool("buddy_manage_todos",
		mcp.WithDescription("Manage project todos and track feature implementation progress"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform: list, update, progress"),
			mcp.Enum("list", "update", "progress"),
		),
		mcp.WithString("feature",
			mcp.Description("Filter by feature name (optional for list)"),
		),
		mcp.WithString("todo_id",
			mcp.Description("Todo ID (required for update)"),
		),
		mcp.WithBoolean("completed",
			mcp.Description("New completion status (required for update)"),
		),
		mcp.WithBoolean("only_incomplete",
			mcp.Description("Show only incomplete todos (optional for list)"),
		),
	)
	mcpServer.AddTool(todoTool, buddyHandlers.GetTodoToolHandler())

	// History tool
	historyTool := mcp.NewTool("buddy_history",
		mcp.WithDescription("Track and search implementation history"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform: list, add, search"),
			mcp.Enum("list", "add", "search"),
		),
		mcp.WithString("feature",
			mcp.Description("Feature name (for filtering or adding)"),
		),
		mcp.WithString("description",
			mcp.Description("Description of changes (required for add)"),
		),
		mcp.WithString("reasoning",
			mcp.Description("Reasoning behind changes (required for add)"),
		),
		mcp.WithArray("changes",
			mcp.Description("List of file changes (required for add)"),
		),
		mcp.WithString("query",
			mcp.Description("Search query (required for search)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Limit results (default: 10)"),
		),
	)
	mcpServer.AddTool(historyTool, buddyHandlers.GetHistoryToolHandler())

	// Backup tool
	backupTool := mcp.NewTool("buddy_backup",
		mcp.WithDescription("Manage file backups for safe code changes"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform: list, create, restore, clean"),
			mcp.Enum("list", "create", "restore", "clean"),
		),
		mcp.WithString("file_path",
			mcp.Description("Original file path (for create or list by file)"),
		),
		mcp.WithString("backup_id",
			mcp.Description("Backup ID (required for restore)"),
		),
		mcp.WithString("context",
			mcp.Description("Context of the change (required for create)"),
		),
		mcp.WithString("reasoning",
			mcp.Description("Reasoning for the backup (required for create)"),
		),
		mcp.WithNumber("max_age_days",
			mcp.Description("Maximum age in days for cleanup (required for clean)"),
		),
	)
	mcpServer.AddTool(backupTool, buddyHandlers.GetBackupToolHandler())

	// Add project context resource
	projectResource := mcp.NewResource(
		"buddy://project-context",
		"Project Context",
		mcp.WithResourceDescription("Complete project context including rules, knowledge, database schema, and todos"),
		mcp.WithMIMEType("application/json"),
	)
	mcpServer.AddResource(projectResource, buddyHandlers.GetProjectContextResourceHandler())

	// Start server with context-aware serving
	fmt.Println("Starting Cursor Buddy MCP server...")

	log.Println("Cursor Buddy MCP server started")

	// Serve stdio directly - this will block until stdin is closed or context is cancelled
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Printf("MCP server error: %v", err)
		return fmt.Errorf("MCP server error: %w", err)
	}

	log.Println("Server completed successfully")
	return nil
}

func main() {
	var (
		buddyPath = flag.String("buddy-path", os.Getenv("BUDDY_PATH"), "Path to the .buddy directory")
		version   = flag.Bool("version", false, "Show version information")
		help      = flag.Bool("help", false, "Show help information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Cursor Buddy MCP Server\n")
		fmt.Fprintf(os.Stderr, "A Model Context Protocol server for development workflow management\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BUDDY_PATH    Path to the .buddy directory (default: .buddy)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --buddy-path=/home/user/project/.buddy\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  BUDDY_PATH=/home/user/project/.buddy %s\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Println("Cursor Buddy MCP Server v1.0.0")
		os.Exit(0)
	}

	// Set default buddy path if not provided
	if *buddyPath == "" {
		*buddyPath = ".buddy"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run the server
	if err := runServer(ctx, *buddyPath); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")
	cancel()
}
