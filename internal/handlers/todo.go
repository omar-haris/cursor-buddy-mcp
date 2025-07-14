package handlers

import (
	"context"
	"crypto/md5"
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

// TodoHandler manages todo items
type TodoHandler struct {
	path          string
	todos         []models.Todo
	searchManager *search.SearchManager
	mu            sync.RWMutex
}

// NewTodoHandler creates a new todo handler
func NewTodoHandler(path string, searchManager *search.SearchManager) *TodoHandler {
	return &TodoHandler{
		path:          path,
		todos:         []models.Todo{},
		searchManager: searchManager,
	}
}

// Load loads all todos from the todos directory
func (th *TodoHandler) Load() error {
	th.mu.Lock()
	defer th.mu.Unlock()

	th.todos = []models.Todo{}

	// First, reindex all todos
	if err := th.searchManager.ReindexAll(search.IndexTypeTodos); err != nil {
		return fmt.Errorf("failed to reindex todos: %w", err)
	}

	err := filepath.Walk(th.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			todos, err := th.loadTodoFile(path)
			if err != nil {
				return fmt.Errorf("failed to load todo file %s: %w", path, err)
			}

			// Add todos and index them
			for _, todo := range todos {
				th.todos = append(th.todos, todo)

				// Index the todo in Bleve
				doc := search.FromTodo(todo)
				if err := th.searchManager.IndexDocument(search.IndexTypeTodos, todo.ID, doc); err != nil {
					return fmt.Errorf("failed to index todo %s: %w", todo.ID, err)
				}
			}
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// loadTodoFile loads todos from a single file
func (th *TodoHandler) loadTodoFile(filePath string) ([]models.Todo, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var todos []models.Todo
	lines := strings.Split(string(content), "\n")

	// Extract feature name from first heading
	feature := filepath.Base(filePath)
	feature = strings.TrimSuffix(feature, ".md")

	for i, line := range lines {
		if strings.HasPrefix(line, "# Feature: ") {
			feature = strings.TrimPrefix(line, "# Feature: ")
		} else if strings.HasPrefix(line, "# ") {
			feature = strings.TrimPrefix(line, "# ")
		}

		// Look for checkbox items
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") {
			completed := strings.HasPrefix(line, "- [x]")
			task := strings.TrimPrefix(line, "- [ ]")
			task = strings.TrimPrefix(task, "- [x]")
			task = strings.TrimSpace(task)

			if task != "" {
				// Generate unique ID
				id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s-%s-%d", filePath, task, i))))

				todo := models.Todo{
					ID:        id,
					Task:      task,
					Feature:   feature,
					Completed: completed,
					FilePath:  filePath,
					UpdatedAt: time.Now(),
				}

				todos = append(todos, todo)
			}
		}
	}

	return todos, nil
}

// GetTodos returns all todos
func (th *TodoHandler) GetTodos() []models.Todo {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.todos
}

// GetTodosByFeature returns todos for a specific feature
func (th *TodoHandler) GetTodosByFeature(feature string) []models.Todo {
	th.mu.RLock()
	defer th.mu.RUnlock()

	var filtered []models.Todo
	for _, todo := range th.todos {
		if strings.EqualFold(todo.Feature, feature) {
			filtered = append(filtered, todo)
		}
	}
	return filtered
}

// GetIncompleteTodos returns all incomplete todos
func (th *TodoHandler) GetIncompleteTodos() []models.Todo {
	th.mu.RLock()
	defer th.mu.RUnlock()

	var filtered []models.Todo
	for _, todo := range th.todos {
		if !todo.Completed {
			filtered = append(filtered, todo)
		}
	}
	return filtered
}

// UpdateTodoStatus updates a todo's completion status
func (th *TodoHandler) UpdateTodoStatus(todoID string, completed bool) error {
	th.mu.Lock()
	defer th.mu.Unlock()

	for i, todo := range th.todos {
		if todo.ID == todoID {
			th.todos[i].Completed = completed
			th.todos[i].UpdatedAt = time.Now()

			// Update the file
			if err := th.updateTodoFile(&th.todos[i]); err != nil {
				return err
			}

			// Update the index
			doc := search.FromTodo(th.todos[i])
			if err := th.searchManager.UpdateDocument(search.IndexTypeTodos, todoID, doc); err != nil {
				return fmt.Errorf("failed to update todo in index: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("todo with ID %s not found", todoID)
}

// updateTodoFile updates a todo in its file
func (th *TodoHandler) updateTodoFile(todo *models.Todo) error {
	content, err := ioutil.ReadFile(todo.FilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.Contains(line, todo.Task) {
			if todo.Completed {
				lines[i] = strings.Replace(line, "- [ ]", "- [x]", 1)
			} else {
				lines[i] = strings.Replace(line, "- [x]", "- [ ]", 1)
			}
			break
		}
	}

	newContent := strings.Join(lines, "\n")
	return ioutil.WriteFile(todo.FilePath, []byte(newContent), 0644)
}

// GetProgress calculates completion progress with enhanced metrics
func (th *TodoHandler) GetProgress() map[string]interface{} {
	th.mu.RLock()
	defer th.mu.RUnlock()

	total := len(th.todos)
	completed := 0
	byFeature := make(map[string]map[string]int)
	recentActivity := make(map[string]int)

	for _, todo := range th.todos {
		if todo.Completed {
			completed++
		}

		// By feature stats
		if _, ok := byFeature[todo.Feature]; !ok {
			byFeature[todo.Feature] = map[string]int{"total": 0, "completed": 0}
		}
		byFeature[todo.Feature]["total"]++
		if todo.Completed {
			byFeature[todo.Feature]["completed"]++
		}

		// Recent activity (last 7 days)
		if todo.UpdatedAt.After(time.Now().AddDate(0, 0, -7)) {
			recentActivity[todo.Feature]++
		}
	}

	return map[string]interface{}{
		"total":     total,
		"completed": completed,
		"percentage": func() float64 {
			if total > 0 {
				return float64(completed) / float64(total) * 100
			} else {
				return 0
			}
		}(),
		"by_feature":      byFeature,
		"recent_activity": recentActivity,
	}
}

// GetToolHandler returns the tool handler function for todos
func (th *TodoHandler) GetToolHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		action, ok := args["action"].(string)
		if !ok {
			return nil, fmt.Errorf("action is required")
		}

		switch action {
		case "list":
			feature, _ := args["feature"].(string)
			onlyIncomplete, _ := args["only_incomplete"].(bool)
			query, _ := args["query"].(string)

			var todos []models.Todo

			if query != "" {
				// Use Bleve search
				filters := make(map[string]interface{})
				if feature != "" {
					filters["feature"] = feature
				}
				if onlyIncomplete {
					filters["completed"] = false
				}

				searchResults, err := th.searchManager.SearchWithFilters(
					search.IndexTypeTodos,
					query,
					filters,
					100, // Limit to 100 results
				)
				if err != nil {
					return nil, fmt.Errorf("search failed: %w", err)
				}

				// Convert search results to todos
				for _, hit := range searchResults.Hits {
					// Find the todo by ID
					for _, todo := range th.todos {
						if todo.ID == hit.ID {
							todos = append(todos, todo)
							break
						}
					}
				}
			} else if feature != "" {
				todos = th.GetTodosByFeature(feature)
			} else if onlyIncomplete {
				todos = th.GetIncompleteTodos()
			} else {
				todos = th.GetTodos()
			}

			// Enhanced result formatting
			result := th.formatTodoResults(query, todos)
			return mcp.NewToolResultText(result), nil

		case "update":
			todoID, ok := args["todo_id"].(string)
			if !ok {
				return nil, fmt.Errorf("todo_id is required for update action")
			}

			completed, ok := args["completed"].(bool)
			if !ok {
				return nil, fmt.Errorf("completed status is required for update action")
			}

			if err := th.UpdateTodoStatus(todoID, completed); err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("Successfully updated todo %s to completed=%v", todoID, completed)), nil

		case "progress":
			progress := th.GetProgress()
			result := th.formatProgressResults(progress)
			return mcp.NewToolResultText(result), nil

		default:
			return nil, fmt.Errorf("invalid action: %s", action)
		}
	}
}

// formatTodoResults formats todo results with enhanced context
func (th *TodoHandler) formatTodoResults(query string, todos []models.Todo) string {
	if len(todos) == 0 {
		result := "No todos found"
		if query != "" {
			result += fmt.Sprintf(" for query: %s", query)
		}

		// Get document count
		count, _ := th.searchManager.GetDocumentCount(search.IndexTypeTodos)
		if count > 0 {
			result += fmt.Sprintf("\n\nThere are %d todos in the system. Try:\n", count)
			result += "- 'incomplete' or 'pending' - for open tasks\n"
			result += "- 'completed' or 'done' - for finished tasks\n"
			result += "- Feature names (e.g., 'authentication', 'ui')\n"
			result += "- Task keywords (e.g., 'bug', 'feature', 'test')\n"
		}

		// Show available features
		features := make(map[string]bool)
		allTodos := th.GetTodos()
		for _, todo := range allTodos {
			features[todo.Feature] = true
		}

		if len(features) > 0 {
			result += "\n\nAvailable features:"
			for feature := range features {
				result += fmt.Sprintf("\n- %s", feature)
			}
		}

		return result
	}

	result := fmt.Sprintf("Found %d todos", len(todos))
	if query != "" {
		result += fmt.Sprintf(" for query: %s", query)
	}
	result += "\n"

	// Group by feature and status
	byFeature := make(map[string][]models.Todo)
	for _, todo := range todos {
		byFeature[todo.Feature] = append(byFeature[todo.Feature], todo)
	}

	for feature, featureTodos := range byFeature {
		result += fmt.Sprintf("\n=== %s ===\n", strings.ToUpper(feature))

		// Separate completed and incomplete
		var incomplete, completed []models.Todo
		for _, todo := range featureTodos {
			if todo.Completed {
				completed = append(completed, todo)
			} else {
				incomplete = append(incomplete, todo)
			}
		}

		// Show incomplete first
		if len(incomplete) > 0 {
			result += "\n📝 PENDING:\n"
			for i, todo := range incomplete {
				result += fmt.Sprintf("  %d. [ ] %s (ID: %s)\n", i+1, todo.Task, todo.ID)
			}
		}

		// Show completed
		if len(completed) > 0 {
			result += "\n✅ COMPLETED:\n"
			for i, todo := range completed {
				result += fmt.Sprintf("  %d. [x] %s (ID: %s)\n", i+1, todo.Task, todo.ID)
			}
		}

		// Feature summary
		totalFeatureTodos := len(featureTodos)
		completedFeatureTodos := len(completed)
		if totalFeatureTodos > 0 {
			percentage := float64(completedFeatureTodos) / float64(totalFeatureTodos) * 100
			result += fmt.Sprintf("\nProgress: %d/%d (%.1f%%)\n", completedFeatureTodos, totalFeatureTodos, percentage)
		}
	}

	return result
}

// formatProgressResults formats progress results with enhanced metrics
func (th *TodoHandler) formatProgressResults(progress map[string]interface{}) string {
	result := "📊 Todo Progress Summary\n"
	result += strings.Repeat("=", 30) + "\n\n"

	result += fmt.Sprintf("Overall Progress:\n")
	result += fmt.Sprintf("├─ Total: %v\n", progress["total"])
	result += fmt.Sprintf("├─ Completed: %v\n", progress["completed"])
	result += fmt.Sprintf("└─ Percentage: %.1f%%\n\n", progress["percentage"])

	if byFeature, ok := progress["by_feature"].(map[string]map[string]int); ok {
		result += "📋 By Feature:\n"

		// Sort features by completion percentage
		type featureStats struct {
			name       string
			completed  int
			total      int
			percentage float64
		}

		var features []featureStats
		for feature, stats := range byFeature {
			percentage := float64(stats["completed"]) / float64(stats["total"]) * 100
			features = append(features, featureStats{
				name:       feature,
				completed:  stats["completed"],
				total:      stats["total"],
				percentage: percentage,
			})
		}

		sort.Slice(features, func(i, j int) bool {
			return features[i].percentage > features[j].percentage
		})

		for _, feature := range features {
			status := "🔴"
			if feature.percentage >= 80 {
				status = "🟢"
			} else if feature.percentage >= 50 {
				status = "🟡"
			}

			result += fmt.Sprintf("├─ %s %s: %d/%d (%.1f%%)\n",
				status, feature.name, feature.completed, feature.total, feature.percentage)
		}
	}

	if recentActivity, ok := progress["recent_activity"].(map[string]int); ok && len(recentActivity) > 0 {
		result += "\n🔥 Recent Activity (Last 7 Days):\n"
		for feature, count := range recentActivity {
			result += fmt.Sprintf("├─ %s: %d updates\n", feature, count)
		}
	}

	return result
}
