package handlers

import (
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/omar-haris/cursor-buddy-mcp/internal/models"
	"github.com/omar-haris/cursor-buddy-mcp/internal/search"
)

// SearchResult represents a search result with score
type SearchResult struct {
	Knowledge models.Knowledge
	Score     float64
	Matches   []string
}

// KnowledgeHandler manages the knowledge base
type KnowledgeHandler struct {
	path          string
	knowledge     []models.Knowledge
	searchManager *search.SearchManager
	mu            sync.RWMutex
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(path string, searchManager *search.SearchManager) *KnowledgeHandler {
	return &KnowledgeHandler{
		path:          path,
		knowledge:     []models.Knowledge{},
		searchManager: searchManager,
	}
}

// Load loads all knowledge from the knowledge directory
func (kh *KnowledgeHandler) Load() error {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.knowledge = []models.Knowledge{}

	// First, reindex all knowledge
	if err := kh.searchManager.ReindexAll(search.IndexTypeKnowledge); err != nil {
		return fmt.Errorf("failed to reindex knowledge: %w", err)
	}

	err := filepath.Walk(kh.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			kb, err := kh.loadKnowledgeFile(path)
			if err != nil {
				return fmt.Errorf("failed to load knowledge %s: %w", path, err)
			}
			kh.knowledge = append(kh.knowledge, kb)

			// Index the knowledge in Bleve
			doc := search.FromKnowledge(kb)
			if err := kh.searchManager.IndexDocument(search.IndexTypeKnowledge, kb.ID, doc); err != nil {
				return fmt.Errorf("failed to index knowledge %s: %w", kb.ID, err)
			}
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// loadKnowledgeFile loads a single knowledge file
func (kh *KnowledgeHandler) loadKnowledgeFile(filePath string) (models.Knowledge, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return models.Knowledge{}, err
	}

	// Parse the knowledge file
	lines := strings.Split(string(content), "\n")
	var title, category string
	var tags []string
	var contentStart int

	// Extract metadata from the first few lines
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "Category: ") {
			category = strings.TrimPrefix(line, "Category: ")
		} else if strings.HasPrefix(line, "Tags: ") {
			tagStr := strings.TrimPrefix(line, "Tags: ")
			tags = strings.Split(tagStr, ", ")
		} else if line == "" && i > 0 {
			contentStart = i + 1
			break
		}
	}

	// Extract content
	contentText := ""
	if contentStart < len(lines) {
		contentText = strings.Join(lines[contentStart:], "\n")
	}

	// Generate ID from file path
	id := fmt.Sprintf("%x", md5.Sum([]byte(filePath)))

	// Determine category from path if not specified
	if category == "" {
		relPath, _ := filepath.Rel(kh.path, filePath)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) > 1 {
			category = parts[0]
		}
	}

	fileInfo, _ := os.Stat(filePath)

	return models.Knowledge{
		ID:        id,
		Title:     title,
		Category:  category,
		Content:   contentText,
		Tags:      tags,
		FilePath:  filePath,
		UpdatedAt: fileInfo.ModTime(),
	}, nil
}

// GetKnowledge returns all loaded knowledge
func (kh *KnowledgeHandler) GetKnowledge() []models.Knowledge {
	kh.mu.RLock()
	defer kh.mu.RUnlock()
	return kh.knowledge
}

// GetKnowledgeByCategory returns knowledge filtered by category
func (kh *KnowledgeHandler) GetKnowledgeByCategory(category string) []models.Knowledge {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	var filtered []models.Knowledge
	for _, kb := range kh.knowledge {
		if kb.Category == category {
			filtered = append(filtered, kb)
		}
	}
	return filtered
}

// GetToolHandler returns the tool handler function for knowledge
func (kh *KnowledgeHandler) GetToolHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		query, ok := args["query"].(string)
		if !ok {
			return nil, fmt.Errorf("query is required")
		}

		category, _ := args["category"].(string)

		// Use Bleve search
		filters := make(map[string]interface{})
		if category != "" {
			filters["category"] = category
		}

		searchResults, err := kh.searchManager.SearchWithFilters(
			search.IndexTypeKnowledge,
			query,
			filters,
			50, // Limit to 50 results
		)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		// Convert search results to knowledge entries
		var results []models.Knowledge
		for _, hit := range searchResults.Hits {
			// Find the knowledge by ID
			for _, kb := range kh.knowledge {
				if kb.ID == hit.ID {
					results = append(results, kb)
					break
				}
			}
		}

		// Enhanced result formatting
		result := kh.formatSearchResults(query, results)

		return mcp.NewToolResultText(result), nil
	}
}

// formatSearchResults formats search results with better context
func (kh *KnowledgeHandler) formatSearchResults(query string, results []models.Knowledge) string {
	if len(results) == 0 {
		result := fmt.Sprintf("No results found for: %s\n", query)

		// Get document count to show available knowledge
		count, _ := kh.searchManager.GetDocumentCount(search.IndexTypeKnowledge)
		if count > 0 {
			result += fmt.Sprintf("\nThere are %d knowledge entries available. Try:\n", count)
			result += "- Using different keywords\n"
			result += "- Searching for broader terms\n"
			result += "- Checking available categories\n"
		}

		result += "\nAvailable categories:"
		categories := make(map[string]bool)
		for _, kb := range kh.knowledge {
			categories[kb.Category] = true
		}
		for category := range categories {
			result += fmt.Sprintf("\n- %s", category)
		}

		return result
	}

	// Format results with relevance information
	result := fmt.Sprintf("Found %d knowledge entries for: %s\n", len(results), query)

	for i, kb := range results {
		result += fmt.Sprintf("\n%d. [%s] %s\n", i+1, kb.Category, kb.Title)
		if len(kb.Tags) > 0 {
			result += fmt.Sprintf("   Tags: %s\n", strings.Join(kb.Tags, ", "))
		}

		// Show content preview
		content := strings.TrimSpace(kb.Content)
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		result += fmt.Sprintf("   %s\n", content)

		// Add separator between results
		if i < len(results)-1 {
			result += "\n" + strings.Repeat("-", 50) + "\n"
		}
	}

	return result
}
