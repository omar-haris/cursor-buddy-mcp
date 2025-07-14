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

// RulesHandler manages coding rules and guidelines
type RulesHandler struct {
	path          string
	rules         []models.Rule
	searchManager *search.SearchManager
	mu            sync.RWMutex
}

// NewRulesHandler creates a new rules handler
func NewRulesHandler(path string, searchManager *search.SearchManager) *RulesHandler {
	return &RulesHandler{
		path:          path,
		rules:         []models.Rule{},
		searchManager: searchManager,
	}
}

// Load loads all rules from the rules directory
func (rh *RulesHandler) Load() error {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.rules = []models.Rule{}

	// First, reindex all rules
	if err := rh.searchManager.ReindexAll(search.IndexTypeRules); err != nil {
		return fmt.Errorf("failed to reindex rules: %w", err)
	}

	files, err := ioutil.ReadDir(rh.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			rule, err := rh.loadRuleFile(filepath.Join(rh.path, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to load rule %s: %w", file.Name(), err)
			}
			rh.rules = append(rh.rules, rule)

			// Index the rule in Bleve
			doc := search.FromRule(rule)
			if err := rh.searchManager.IndexDocument(search.IndexTypeRules, rule.ID, doc); err != nil {
				return fmt.Errorf("failed to index rule %s: %w", rule.ID, err)
			}
		}
	}

	return nil
}

// loadRuleFile loads a single rule file
func (rh *RulesHandler) loadRuleFile(filePath string) (models.Rule, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return models.Rule{}, err
	}

	// Parse the rule file
	lines := strings.Split(string(content), "\n")
	var title, category, priority string
	var descriptionStart int

	// Extract metadata from the first few lines
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "Category: ") {
			category = strings.TrimPrefix(line, "Category: ")
		} else if strings.HasPrefix(line, "Priority: ") {
			priority = strings.TrimPrefix(line, "Priority: ")
		} else if line == "" && i > 0 {
			descriptionStart = i + 1
			break
		}
	}

	// Extract description
	description := ""
	if descriptionStart < len(lines) {
		description = strings.Join(lines[descriptionStart:], "\n")
	}

	// Generate ID from file path
	id := fmt.Sprintf("%x", md5.Sum([]byte(filePath)))

	fileInfo, _ := os.Stat(filePath)

	return models.Rule{
		ID:          id,
		Category:    category,
		Title:       title,
		Description: description,
		Priority:    priority,
		Content:     string(content),
		FilePath:    filePath,
		UpdatedAt:   fileInfo.ModTime(),
	}, nil
}

// GetRules returns all loaded rules
func (rh *RulesHandler) GetRules() []models.Rule {
	rh.mu.RLock()
	defer rh.mu.RUnlock()
	return rh.rules
}

// GetRulesByCategory returns rules filtered by category
func (rh *RulesHandler) GetRulesByCategory(category string) []models.Rule {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	var filtered []models.Rule
	for _, rule := range rh.rules {
		if rule.Category == category {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// GetRulesByPriority returns rules filtered by priority
func (rh *RulesHandler) GetRulesByPriority(priority string) []models.Rule {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	var filtered []models.Rule
	for _, rule := range rh.rules {
		if rule.Priority == priority {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// GetToolHandler returns the tool handler function for rules
func (rh *RulesHandler) GetToolHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use GetArguments() method to access arguments
		args := request.GetArguments()
		category, _ := args["category"].(string)
		priority, _ := args["priority"].(string)
		searchQuery, _ := args["search"].(string)

		var rules []models.Rule

		// If search query is provided, use Bleve search
		if searchQuery != "" {
			filters := make(map[string]interface{})
			if category != "" {
				filters["category"] = category
			}
			if priority != "" {
				filters["priority"] = priority
			}

			searchResults, err := rh.searchManager.SearchWithFilters(
				search.IndexTypeRules,
				searchQuery,
				filters,
				50, // Limit to 50 results
			)
			if err != nil {
				return nil, fmt.Errorf("search failed: %w", err)
			}

			// Convert search results to rules
			for _, hit := range searchResults.Hits {
				// Find the rule by ID
				for _, rule := range rh.rules {
					if rule.ID == hit.ID {
						rules = append(rules, rule)
						break
					}
				}
			}
		} else {
			// Use traditional filtering
			rules = rh.GetRules()

			// Apply filters
			if category != "" {
				rules = rh.GetRulesByCategory(category)
			}
			if priority != "" {
				var filtered []models.Rule
				for _, rule := range rules {
					if rule.Priority == priority {
						filtered = append(filtered, rule)
					}
				}
				rules = filtered
			}
		}

		// Enhanced result formatting
		result := rh.formatRulesResults(category, priority, rules, searchQuery)

		return mcp.NewToolResultText(result), nil
	}
}

// formatRulesResults formats rules results with enhanced context
func (rh *RulesHandler) formatRulesResults(category, priority string, rules []models.Rule, searchQuery string) string {
	if len(rules) == 0 {
		result := "No rules found"
		if searchQuery != "" {
			result += fmt.Sprintf(" for search: %s", searchQuery)
		}
		if category != "" {
			result += fmt.Sprintf(" in category: %s", category)
		}
		if priority != "" {
			result += fmt.Sprintf(" with priority: %s", priority)
		}
		result += "\n\nAvailable categories:"

		// Show available categories
		categories := make(map[string]bool)
		allRules := rh.GetRules()
		for _, rule := range allRules {
			categories[rule.Category] = true
		}
		for cat := range categories {
			result += fmt.Sprintf("\n- %s", cat)
		}

		result += "\n\nAvailable priorities: critical, recommended, optional"
		return result
	}

	result := fmt.Sprintf("Found %d rules", len(rules))
	if searchQuery != "" {
		result += fmt.Sprintf(" for search: %s", searchQuery)
	}
	if category != "" {
		result += fmt.Sprintf(" in category: %s", category)
	}
	if priority != "" {
		result += fmt.Sprintf(" with priority: %s", priority)
	}
	result += "\n"

	// Group rules by priority for better organization
	priorityGroups := make(map[string][]models.Rule)
	for _, rule := range rules {
		priority := rule.Priority
		if priority == "" {
			priority = "unspecified"
		}
		priorityGroups[priority] = append(priorityGroups[priority], rule)
	}

	// Display in priority order
	priorityOrder := []string{"critical", "recommended", "optional", "unspecified"}
	for _, pri := range priorityOrder {
		if rulesInPriority, exists := priorityGroups[pri]; exists {
			result += fmt.Sprintf("\n=== %s PRIORITY ===\n", strings.ToUpper(pri))

			for i, rule := range rulesInPriority {
				result += fmt.Sprintf("\n%d. [%s] %s\n", i+1, rule.Category, rule.Title)

				// Show description with better formatting
				description := strings.TrimSpace(rule.Description)
				if len(description) > 300 {
					description = description[:300] + "..."
				}

				// Format multiline descriptions better
				lines := strings.Split(description, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						result += fmt.Sprintf("   %s\n", strings.TrimSpace(line))
					}
				}

				if i < len(rulesInPriority)-1 {
					result += "\n" + strings.Repeat("-", 40) + "\n"
				}
			}
		}
	}

	return result
}
