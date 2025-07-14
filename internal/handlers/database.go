package handlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/omar-haris/cursor-buddy-mcp/internal/models"
	"github.com/omar-haris/cursor-buddy-mcp/internal/search"
)

// DatabaseHandler manages database schema information
type DatabaseHandler struct {
	path          string
	dbInfo        *models.DatabaseInfo
	searchManager *search.SearchManager
	mu            sync.RWMutex
}

// NewDatabaseHandler creates a new database handler
func NewDatabaseHandler(path string, searchManager *search.SearchManager) *DatabaseHandler {
	return &DatabaseHandler{
		path:          path,
		dbInfo:        nil,
		searchManager: searchManager,
	}
}

// Load loads database schema information
func (dh *DatabaseHandler) Load() error {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	// First, reindex all database tables
	if err := dh.searchManager.ReindexAll(search.IndexTypeDatabase); err != nil {
		return fmt.Errorf("failed to reindex database: %w", err)
	}

	dbInfo := &models.DatabaseInfo{
		Tables:    []models.Table{},
		UpdatedAt: time.Now(),
	}

	// Check for schema.sql
	schemaPath := filepath.Join(dh.path, "schema.sql")
	if _, err := os.Stat(schemaPath); err == nil {
		dbInfo.SchemaPath = schemaPath

		// Parse schema file
		if tables, err := dh.parseSchema(schemaPath); err == nil {
			dbInfo.Tables = tables

			// Index all tables
			for _, table := range tables {
				doc := search.FromTable(table)
				if err := dh.searchManager.IndexDocument(search.IndexTypeDatabase, table.Name, doc); err != nil {
					// Log error but continue
					fmt.Printf("failed to index table %s: %v\n", table.Name, err)
				}
			}
		}
	}

	// Check for ERD files
	erdFiles := []string{"erd.png", "erd.jpg", "erd.svg", "erd.pdf"}
	for _, erd := range erdFiles {
		erdPath := filepath.Join(dh.path, erd)
		if _, err := os.Stat(erdPath); err == nil {
			dbInfo.ERDPath = erdPath
			break
		}
	}

	// Load connection info
	connPath := filepath.Join(dh.path, "connection.md")
	if content, err := ioutil.ReadFile(connPath); err == nil {
		dbInfo.ConnectionInfo = string(content)

		// Try to determine database type
		connStr := strings.ToLower(string(content))
		if strings.Contains(connStr, "mysql") {
			dbInfo.Type = "mysql"
		} else if strings.Contains(connStr, "postgres") || strings.Contains(connStr, "postgresql") {
			dbInfo.Type = "postgresql"
		} else if strings.Contains(connStr, "sqlite") {
			dbInfo.Type = "sqlite"
		} else if strings.Contains(connStr, "mongodb") {
			dbInfo.Type = "mongodb"
		}
	}

	dh.dbInfo = dbInfo
	return nil
}

// parseSchema parses a SQL schema file
func (dh *DatabaseHandler) parseSchema(filePath string) ([]models.Table, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var tables []models.Table
	sql := string(content)

	// Find CREATE TABLE statements
	createTableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\((.*?)\);`)
	matches := createTableRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			tableDefinition := match[2]

			table := models.Table{
				Name:    tableName,
				Columns: dh.parseColumns(tableDefinition),
				Indexes: dh.parseIndexes(sql, tableName),
			}

			tables = append(tables, table)
		}
	}

	return tables, nil
}

// parseColumns parses column definitions from CREATE TABLE statement
func (dh *DatabaseHandler) parseColumns(definition string) []models.Column {
	var columns []models.Column

	// Split by commas, but be careful about nested parentheses
	lines := strings.Split(definition, ",")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToUpper(line), "PRIMARY KEY") ||
			strings.HasPrefix(strings.ToUpper(line), "FOREIGN KEY") ||
			strings.HasPrefix(strings.ToUpper(line), "UNIQUE") ||
			strings.HasPrefix(strings.ToUpper(line), "INDEX") ||
			strings.HasPrefix(strings.ToUpper(line), "KEY") ||
			strings.HasPrefix(strings.ToUpper(line), "CONSTRAINT") {
			continue
		}

		// Parse column definition
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			column := models.Column{
				Name:     parts[0],
				Type:     parts[1],
				Nullable: !strings.Contains(strings.ToUpper(line), "NOT NULL"),
			}

			// Check for DEFAULT value
			if strings.Contains(strings.ToUpper(line), "DEFAULT") {
				defaultRegex := regexp.MustCompile(`(?i)DEFAULT\s+([^,\s]+)`)
				if defaultMatch := defaultRegex.FindStringSubmatch(line); len(defaultMatch) > 1 {
					column.DefaultValue = defaultMatch[1]
				}
			}

			columns = append(columns, column)
		}
	}

	return columns
}

// parseIndexes extracts index information for a table
func (dh *DatabaseHandler) parseIndexes(sql, tableName string) []models.Index {
	var indexes []models.Index

	// Look for CREATE INDEX statements
	indexRegex := regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(\w+)\s+ON\s+` + tableName + `\s*\((.*?)\)`)
	matches := indexRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			index := models.Index{
				Name:    match[2],
				Unique:  strings.ToUpper(match[1]) == "UNIQUE",
				Columns: strings.Split(strings.ReplaceAll(match[3], " ", ""), ","),
			}
			indexes = append(indexes, index)
		}
	}

	return indexes
}

// GetDatabaseInfo returns the database information
func (dh *DatabaseHandler) GetDatabaseInfo() *models.DatabaseInfo {
	dh.mu.RLock()
	defer dh.mu.RUnlock()
	return dh.dbInfo
}

// GetTableByName returns a specific table by name
func (dh *DatabaseHandler) GetTableByName(name string) *models.Table {
	dh.mu.RLock()
	defer dh.mu.RUnlock()

	if dh.dbInfo == nil {
		return nil
	}

	for _, table := range dh.dbInfo.Tables {
		if strings.EqualFold(table.Name, name) {
			return &table
		}
	}

	return nil
}

// ValidateQuery validates a SQL query against the schema
func (dh *DatabaseHandler) ValidateQuery(query string) (bool, string) {
	dh.mu.RLock()
	defer dh.mu.RUnlock()

	if dh.dbInfo == nil || len(dh.dbInfo.Tables) == 0 {
		return true, "No schema loaded for validation"
	}

	lower := strings.ToLower(query)

	// Check for dangerous operations
	dangerous := []string{
		"drop table", "drop database", "truncate",
		"delete from", "alter table", "create database",
	}

	for _, danger := range dangerous {
		if strings.Contains(lower, danger) {
			// Check if it's a DELETE with WHERE clause
			if danger == "delete from" && strings.Contains(lower, "where") {
				continue
			}
			return false, fmt.Sprintf("Dangerous operation detected: %s", danger)
		}
	}

	// Extract table names from query
	tableNames := dh.extractTableNames(query)
	for _, tableName := range tableNames {
		if dh.GetTableByName(tableName) == nil {
			return false, fmt.Sprintf("Table '%s' not found in schema", tableName)
		}
	}

	return true, "Query validation passed"
}

// extractTableNames extracts table names from a SQL query
func (dh *DatabaseHandler) extractTableNames(query string) []string {
	var tableNames []string

	// Simple regex to find table names after FROM and JOIN
	fromRegex := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(\w+)`)
	matches := fromRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tableNames = append(tableNames, match[1])
		}
	}

	return tableNames
}

// GetToolHandler returns the tool handler function for database info
func (dh *DatabaseHandler) GetToolHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		tableName, _ := args["table_name"].(string)
		validateQuery, _ := args["validate_query"].(string)
		searchQuery, _ := args["search"].(string)

		dbInfo := dh.GetDatabaseInfo()
		if dbInfo == nil {
			return mcp.NewToolResultText("No database information loaded"), nil
		}

		// Handle search query using Bleve
		if searchQuery != "" {
			searchResults, err := dh.searchManager.Search(
				search.IndexTypeDatabase,
				searchQuery,
				20, // Limit to 20 results
			)
			if err != nil {
				return nil, fmt.Errorf("search failed: %w", err)
			}

			// Convert search results to tables
			var tables []models.Table
			for _, hit := range searchResults.Hits {
				// Find the table by name (ID)
				for _, table := range dbInfo.Tables {
					if table.Name == hit.ID {
						tables = append(tables, table)
						break
					}
				}
			}

			result := dh.formatSearchResults(searchQuery, tables)
			return mcp.NewToolResultText(result), nil
		}

		// Handle specific table request
		if tableName != "" {
			table := dh.GetTableByName(tableName)
			if table == nil {
				result := fmt.Sprintf("Table '%s' not found\n\n", tableName)
				result += "Available tables:\n"
				for _, t := range dbInfo.Tables {
					result += fmt.Sprintf("- %s\n", t.Name)
				}
				return mcp.NewToolResultText(result), nil
			}

			result := dh.formatTableDetails(*table)
			return mcp.NewToolResultText(result), nil
		}

		// Handle query validation
		if validateQuery != "" {
			valid, message := dh.ValidateQuery(validateQuery)
			result := fmt.Sprintf("Query Validation:\n")
			result += strings.Repeat("-", 20) + "\n\n"
			result += fmt.Sprintf("Query: %s\n\n", validateQuery)
			result += fmt.Sprintf("Valid: %v\n", valid)
			result += fmt.Sprintf("Message: %s\n", message)

			if !valid {
				result += "\nSuggestions:\n"
				result += "- Check table names are correct\n"
				result += "- Avoid dangerous operations like DROP or TRUNCATE\n"
				result += "- Use WHERE clauses with DELETE statements\n"
			}

			return mcp.NewToolResultText(result), nil
		}

		// Return general database info
		result := dh.formatDatabaseOverview()
		return mcp.NewToolResultText(result), nil
	}
}

// formatDatabaseOverview formats the database overview
func (dh *DatabaseHandler) formatDatabaseOverview() string {
	dbInfo := dh.GetDatabaseInfo()

	result := "Database Information\n"
	result += strings.Repeat("=", 25) + "\n\n"

	result += fmt.Sprintf("Type: %s\n", dbInfo.Type)
	result += fmt.Sprintf("Schema Path: %s\n", dbInfo.SchemaPath)
	result += fmt.Sprintf("ERD Path: %s\n", dbInfo.ERDPath)
	result += fmt.Sprintf("Has Connection Info: %v\n", dbInfo.ConnectionInfo != "")
	result += fmt.Sprintf("Total Tables: %d\n", len(dbInfo.Tables))
	result += fmt.Sprintf("Last Updated: %s\n\n", dbInfo.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(dbInfo.Tables) > 0 {
		result += "Tables Summary:\n"
		for _, table := range dbInfo.Tables {
			result += fmt.Sprintf("- %s (%d columns, %d indexes)\n",
				table.Name, len(table.Columns), len(table.Indexes))
		}
	}

	return result
}

// formatTableDetails formats detailed table information
func (dh *DatabaseHandler) formatTableDetails(table models.Table) string {
	result := fmt.Sprintf("Table: %s\n", table.Name)
	result += strings.Repeat("=", len(table.Name)+7) + "\n\n"

	if table.Description != "" {
		result += fmt.Sprintf("Description: %s\n\n", table.Description)
	}

	// Columns
	if len(table.Columns) > 0 {
		result += "Columns:\n"
		for _, col := range table.Columns {
			result += fmt.Sprintf("- %s %s", col.Name, col.Type)

			var attributes []string
			if !col.Nullable {
				attributes = append(attributes, "NOT NULL")
			}
			if col.DefaultValue != "" {
				attributes = append(attributes, fmt.Sprintf("DEFAULT %s", col.DefaultValue))
			}

			if len(attributes) > 0 {
				result += fmt.Sprintf(" (%s)", strings.Join(attributes, ", "))
			}
			result += "\n"
		}
	}

	// Indexes
	if len(table.Indexes) > 0 {
		result += "\nIndexes:\n"
		for _, idx := range table.Indexes {
			uniqueStr := ""
			if idx.Unique {
				uniqueStr = " (UNIQUE)"
			}
			result += fmt.Sprintf("- %s on (%s)%s\n",
				idx.Name, strings.Join(idx.Columns, ", "), uniqueStr)
		}
	}

	// Sample queries
	result += "\nSample Queries:\n"
	result += fmt.Sprintf("- SELECT * FROM %s LIMIT 10;\n", table.Name)
	result += fmt.Sprintf("- SELECT COUNT(*) FROM %s;\n", table.Name)

	return result
}

// formatSearchResults formats database search results
func (dh *DatabaseHandler) formatSearchResults(query string, tables []models.Table) string {
	if len(tables) == 0 {
		result := fmt.Sprintf("No tables found for search: %s\n", query)

		// Get document count
		count, _ := dh.searchManager.GetDocumentCount(search.IndexTypeDatabase)
		if count > 0 {
			result += fmt.Sprintf("\nThere are %d tables in the database. Try:\n", count)
			result += "- Table names or partial names\n"
			result += "- Column names or types\n"
			result += "- 'index' to find indexed tables\n"
			result += "- Data types (e.g., 'varchar', 'integer', 'timestamp')\n"
		}

		// Show available tables
		if dh.dbInfo != nil && len(dh.dbInfo.Tables) > 0 {
			result += "\nAvailable tables:\n"
			for _, table := range dh.dbInfo.Tables {
				result += fmt.Sprintf("- %s\n", table.Name)
			}
		}

		return result
	}

	result := fmt.Sprintf("Found %d tables for search: %s\n\n", len(tables), query)

	for i, table := range tables {
		result += fmt.Sprintf("%d. %s\n", i+1, table.Name)
		result += fmt.Sprintf("   %d columns, %d indexes\n", len(table.Columns), len(table.Indexes))

		// Show key columns
		if len(table.Columns) > 0 {
			result += "   Key columns: "
			for j, col := range table.Columns {
				if j >= 3 {
					result += fmt.Sprintf("... and %d more", len(table.Columns)-3)
					break
				}
				if j > 0 {
					result += ", "
				}
				result += fmt.Sprintf("%s (%s)", col.Name, col.Type)
			}
			result += "\n"
		}

		if table.Description != "" {
			result += fmt.Sprintf("   %s\n", table.Description)
		}

		result += "\n"
	}

	return result
}
