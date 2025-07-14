package handlers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/omar-haris/cursor-buddy-mcp/internal/models"
	"github.com/omar-haris/cursor-buddy-mcp/internal/search"
)

// HistoryHandler manages implementation history
type HistoryHandler struct {
	path          string
	entries       []models.HistoryEntry
	searchManager *search.SearchManager
	mu            sync.RWMutex
}

// NewHistoryHandler creates a new history handler
func NewHistoryHandler(path string, searchManager *search.SearchManager) *HistoryHandler {
	return &HistoryHandler{
		path:          path,
		entries:       []models.HistoryEntry{},
		searchManager: searchManager,
	}
}

// Load loads all history entries
func (hh *HistoryHandler) Load() error {
	hh.mu.Lock()
	defer hh.mu.Unlock()

	hh.entries = []models.HistoryEntry{}

	// First, reindex all history
	if err := hh.searchManager.ReindexAll(search.IndexTypeHistory); err != nil {
		return fmt.Errorf("failed to reindex history: %w", err)
	}

	files, err := ioutil.ReadDir(hh.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			entry, err := hh.loadHistoryFile(filepath.Join(hh.path, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to load history %s: %w", file.Name(), err)
			}
			hh.entries = append(hh.entries, entry)

			// Index the entry in Bleve
			doc := search.FromHistoryEntry(entry)
			if err := hh.searchManager.IndexDocument(search.IndexTypeHistory, entry.ID, doc); err != nil {
				return fmt.Errorf("failed to index history %s: %w", entry.ID, err)
			}
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(hh.entries, func(i, j int) bool {
		return hh.entries[i].Timestamp.After(hh.entries[j].Timestamp)
	})

	return nil
}

// loadHistoryFile loads a single history file
func (hh *HistoryHandler) loadHistoryFile(filePath string) (models.HistoryEntry, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return models.HistoryEntry{}, err
	}

	var entry models.HistoryEntry
	if err := json.Unmarshal(content, &entry); err != nil {
		return models.HistoryEntry{}, err
	}

	return entry, nil
}

// AddEntry adds a new history entry
func (hh *HistoryHandler) AddEntry(feature, description, reasoning string, changes []models.Change) error {
	hh.mu.Lock()
	defer hh.mu.Unlock()

	entry := models.HistoryEntry{
		ID:          fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s-%d", feature, time.Now().UnixNano())))),
		Feature:     feature,
		Description: description,
		Reasoning:   reasoning,
		Changes:     changes,
		Timestamp:   time.Now(),
	}

	// Save to file
	filePath := filepath.Join(hh.path, fmt.Sprintf("%s.json", entry.ID))
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return err
	}

	// Add to memory
	hh.entries = append([]models.HistoryEntry{entry}, hh.entries...)

	// Index the entry in Bleve
	doc := search.FromHistoryEntry(entry)
	if err := hh.searchManager.IndexDocument(search.IndexTypeHistory, entry.ID, doc); err != nil {
		return fmt.Errorf("failed to index history %s: %w", entry.ID, err)
	}

	return nil
}

// GetRecentHistory returns the most recent history entries
func (hh *HistoryHandler) GetRecentHistory(limit int) []models.HistoryEntry {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	if limit > len(hh.entries) {
		limit = len(hh.entries)
	}

	return hh.entries[:limit]
}

// GetHistoryByFeature returns history entries for a specific feature
func (hh *HistoryHandler) GetHistoryByFeature(feature string) []models.HistoryEntry {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	var filtered []models.HistoryEntry
	for _, entry := range hh.entries {
		if strings.EqualFold(entry.Feature, feature) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// GetToolHandler returns the tool handler function for history
func (hh *HistoryHandler) GetToolHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		action, ok := args["action"].(string)
		if !ok {
			return nil, fmt.Errorf("action is required")
		}

		switch action {
		case "list":
			feature, _ := args["feature"].(string)
			limitFloat, ok := args["limit"].(float64)
			limit := 10
			if ok {
				limit = int(limitFloat)
			}

			var entries []models.HistoryEntry
			if feature != "" {
				entries = hh.GetHistoryByFeature(feature)
			} else {
				entries = hh.GetRecentHistory(limit)
			}

			result := hh.formatHistoryResults(entries)
			return mcp.NewToolResultText(result), nil

		case "add":
			feature, ok := args["feature"].(string)
			if !ok {
				return nil, fmt.Errorf("feature is required for add action")
			}

			description, ok := args["description"].(string)
			if !ok {
				return nil, fmt.Errorf("description is required for add action")
			}

			reasoning, ok := args["reasoning"].(string)
			if !ok {
				return nil, fmt.Errorf("reasoning is required for add action")
			}

			changesData, ok := args["changes"].([]interface{})
			if !ok {
				return nil, fmt.Errorf("changes array is required for add action")
			}

			var changes []models.Change
			for _, changeData := range changesData {
				if changeMap, ok := changeData.(map[string]interface{}); ok {
					change := models.Change{
						FilePath:   changeMap["file_path"].(string),
						ChangeType: changeMap["change_type"].(string),
					}
					if before, ok := changeMap["before"].(string); ok {
						change.Before = before
					}
					if after, ok := changeMap["after"].(string); ok {
						change.After = after
					}
					changes = append(changes, change)
				}
			}

			if err := hh.AddEntry(feature, description, reasoning, changes); err != nil {
				return nil, err
			}

			return mcp.NewToolResultText("Successfully added history entry"), nil

		case "search":
			query, ok := args["query"].(string)
			if !ok {
				return nil, fmt.Errorf("query is required for search action")
			}

			// Use Bleve search
			searchResults, err := hh.searchManager.Search(
				search.IndexTypeHistory,
				query,
				50, // Limit to 50 results
			)
			if err != nil {
				return nil, fmt.Errorf("search failed: %w", err)
			}

			// Convert search results to history entries
			var entries []models.HistoryEntry
			for _, hit := range searchResults.Hits {
				// Find the entry by ID
				for _, entry := range hh.entries {
					if entry.ID == hit.ID {
						entries = append(entries, entry)
						break
					}
				}
			}

			result := hh.formatSearchResults(query, entries)
			return mcp.NewToolResultText(result), nil

		default:
			return nil, fmt.Errorf("invalid action: %s", action)
		}
	}
}

// formatHistoryResults formats history entries for display
func (hh *HistoryHandler) formatHistoryResults(entries []models.HistoryEntry) string {
	if len(entries) == 0 {
		return "No history entries found"
	}

	result := fmt.Sprintf("Found %d history entries:\n", len(entries))

	for i, entry := range entries {
		result += fmt.Sprintf("\n%d. [%s] %s\n", i+1, entry.Feature, entry.Description)
		result += fmt.Sprintf("   Time: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
		result += fmt.Sprintf("   Reasoning: %s\n", entry.Reasoning)

		if len(entry.Changes) > 0 {
			result += "   Changes:\n"
			for _, change := range entry.Changes {
				emoji := hh.getChangeTypeEmoji(change.ChangeType)
				result += fmt.Sprintf("   %s %s (%s)\n", emoji, change.FilePath, change.ChangeType)
			}
		}

		if i < len(entries)-1 {
			result += "\n" + strings.Repeat("-", 60) + "\n"
		}
	}

	return result
}

// formatSearchResults formats search results with enhanced context
func (hh *HistoryHandler) formatSearchResults(query string, entries []models.HistoryEntry) string {
	if len(entries) == 0 {
		result := fmt.Sprintf("No history entries found for: %s\n", query)

		// Get document count
		count, _ := hh.searchManager.GetDocumentCount(search.IndexTypeHistory)
		if count > 0 {
			result += fmt.Sprintf("\nThere are %d history entries available. Try:\n", count)
			result += "- Different keywords or phrases\n"
			result += "- Feature names (e.g., 'authentication', 'database')\n"
			result += "- Change types (e.g., 'created', 'modified', 'deleted')\n"
			result += "- File names or paths\n"
		}

		// Show available features
		features := make(map[string]bool)
		allEntries := hh.GetRecentHistory(100)
		for _, entry := range allEntries {
			features[entry.Feature] = true
		}

		if len(features) > 0 {
			result += "\n\nAvailable features in history:"
			for feature := range features {
				result += fmt.Sprintf("\n- %s", feature)
			}
		}

		return result
	}

	result := fmt.Sprintf("Found %d history entries for: %s\n", len(entries), query)

	// Group by recency
	var today, thisWeek, older []models.HistoryEntry
	now := time.Now()

	for _, entry := range entries {
		daysSince := now.Sub(entry.Timestamp).Hours() / 24
		if daysSince < 1 {
			today = append(today, entry)
		} else if daysSince < 7 {
			thisWeek = append(thisWeek, entry)
		} else {
			older = append(older, entry)
		}
	}

	// Display by recency
	if len(today) > 0 {
		result += "\n📅 TODAY:\n"
		for i, entry := range today {
			result += hh.formatSingleEntry(i+1, entry)
		}
	}

	if len(thisWeek) > 0 {
		result += "\n📅 THIS WEEK:\n"
		for i, entry := range thisWeek {
			result += hh.formatSingleEntry(i+1, entry)
		}
	}

	if len(older) > 0 {
		result += "\n📅 OLDER:\n"
		for i, entry := range older {
			result += hh.formatSingleEntry(i+1, entry)
		}
	}

	return result
}

// formatSingleEntry formats a single history entry
func (hh *HistoryHandler) formatSingleEntry(num int, entry models.HistoryEntry) string {
	result := fmt.Sprintf("\n%d. [%s] %s\n", num, entry.Feature, entry.Description)
	result += fmt.Sprintf("   Time: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("   Reasoning: %s\n", entry.Reasoning)

	if len(entry.Changes) > 0 {
		result += "   Changes:\n"
		for _, change := range entry.Changes {
			emoji := hh.getChangeTypeEmoji(change.ChangeType)
			result += fmt.Sprintf("   %s %s (%s)\n", emoji, change.FilePath, change.ChangeType)
		}
	}

	return result + "\n"
}

// getChangeTypeEmoji returns an emoji for the change type
func (hh *HistoryHandler) getChangeTypeEmoji(changeType string) string {
	switch strings.ToLower(changeType) {
	case "created":
		return "➕"
	case "modified":
		return "📝"
	case "deleted":
		return "🗑️"
	case "renamed":
		return "📝"
	default:
		return "•"
	}
}
