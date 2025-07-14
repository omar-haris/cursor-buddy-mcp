package handlers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/omar-haris/cursor-buddy-mcp/internal/models"
	"github.com/omar-haris/cursor-buddy-mcp/internal/search"
)

// BackupHandler manages file backups
type BackupHandler struct {
	path          string
	backups       []models.Backup
	searchManager *search.SearchManager
	mu            sync.RWMutex
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(path string, searchManager *search.SearchManager) *BackupHandler {
	return &BackupHandler{
		path:          path,
		backups:       []models.Backup{},
		searchManager: searchManager,
	}
}

// Load loads all backup records
func (bh *BackupHandler) Load() error {
	bh.mu.Lock()
	defer bh.mu.Unlock()

	bh.backups = []models.Backup{}

	// First, reindex all backups
	if err := bh.searchManager.ReindexAll(search.IndexTypeBackups); err != nil {
		return fmt.Errorf("failed to reindex backups: %w", err)
	}

	// Load backup metadata
	metadataPath := filepath.Join(bh.path, "metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		content, err := ioutil.ReadFile(metadataPath)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(content, &bh.backups); err != nil {
			return err
		}

		// Index all backups
		for _, backup := range bh.backups {
			doc := search.FromBackup(backup)
			if err := bh.searchManager.IndexDocument(search.IndexTypeBackups, backup.ID, doc); err != nil {
				fmt.Printf("failed to index backup %s: %v\n", backup.ID, err)
			}
		}
	}

	return nil
}

// save saves backup metadata
func (bh *BackupHandler) save() error {
	metadataPath := filepath.Join(bh.path, "metadata.json")
	data, err := json.MarshalIndent(bh.backups, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(metadataPath, data, 0644)
}

// CreateBackup creates a backup of a file
func (bh *BackupHandler) CreateBackup(originalPath, context, reasoning string) (*models.Backup, error) {
	bh.mu.Lock()
	defer bh.mu.Unlock()

	// Check if file exists
	fileInfo, err := os.Stat(originalPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Generate backup ID and path
	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s-%d", originalPath, time.Now().UnixNano()))))
	timestamp := time.Now()
	backupFileName := fmt.Sprintf("%s_%s%s",
		strings.TrimSuffix(filepath.Base(originalPath), filepath.Ext(originalPath)),
		timestamp.Format("20060102_150405"),
		filepath.Ext(originalPath))
	backupPath := filepath.Join(bh.path, id, backupFileName)

	// Create backup directory
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy file
	if err := bh.copyFile(originalPath, backupPath); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Create backup record
	backup := models.Backup{
		ID:            id,
		OriginalPath:  originalPath,
		BackupPath:    backupPath,
		Timestamp:     timestamp,
		ChangeContext: context,
		Reasoning:     reasoning,
		FileSize:      fileInfo.Size(),
	}

	// Add to list and save
	bh.backups = append(bh.backups, backup)
	if err := bh.save(); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// Index the backup
	doc := search.FromBackup(backup)
	if err := bh.searchManager.IndexDocument(search.IndexTypeBackups, backup.ID, doc); err != nil {
		fmt.Printf("failed to index backup %s: %v\n", backup.ID, err)
	}

	return &backup, nil
}

// copyFile copies a file from src to dst
func (bh *BackupHandler) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// RestoreBackup restores a backup
func (bh *BackupHandler) RestoreBackup(backupID string) error {
	bh.mu.RLock()
	var backup *models.Backup
	for _, b := range bh.backups {
		if b.ID == backupID {
			backup = &b
			break
		}
	}
	bh.mu.RUnlock()

	if backup == nil {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Check if backup file exists
	if _, err := os.Stat(backup.BackupPath); err != nil {
		return fmt.Errorf("backup file missing: %w", err)
	}

	// Copy backup to original location
	if err := bh.copyFile(backup.BackupPath, backup.OriginalPath); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// ListBackups returns all backups or filtered by file path
func (bh *BackupHandler) ListBackups(filePath string) []models.Backup {
	bh.mu.RLock()
	defer bh.mu.RUnlock()

	if filePath == "" {
		return bh.backups
	}

	var filtered []models.Backup
	for _, backup := range bh.backups {
		if backup.OriginalPath == filePath {
			filtered = append(filtered, backup)
		}
	}
	return filtered
}

// CleanOldBackups removes backups older than specified days
func (bh *BackupHandler) CleanOldBackups(maxAgeDays int) (int, error) {
	bh.mu.Lock()
	defer bh.mu.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	var retained []models.Backup
	removedCount := 0

	for _, backup := range bh.backups {
		if backup.Timestamp.Before(cutoffTime) {
			// Remove backup files
			if err := os.RemoveAll(filepath.Dir(backup.BackupPath)); err != nil {
				fmt.Printf("failed to remove backup %s: %v\n", backup.ID, err)
			}

			// Remove from index
			if err := bh.searchManager.DeleteDocument(search.IndexTypeBackups, backup.ID); err != nil {
				fmt.Printf("failed to remove backup from index %s: %v\n", backup.ID, err)
			}

			removedCount++
		} else {
			retained = append(retained, backup)
		}
	}

	bh.backups = retained
	if err := bh.save(); err != nil {
		return removedCount, fmt.Errorf("failed to save metadata: %w", err)
	}

	return removedCount, nil
}

// GetToolHandler returns the tool handler function for backups
func (bh *BackupHandler) GetToolHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		action, ok := args["action"].(string)
		if !ok {
			return nil, fmt.Errorf("action is required")
		}

		switch action {
		case "list":
			filePath, _ := args["file_path"].(string)
			query, _ := args["query"].(string)

			var backups []models.Backup

			if query != "" {
				// Use Bleve search
				searchResults, err := bh.searchManager.Search(
					search.IndexTypeBackups,
					query,
					50, // Limit to 50 results
				)
				if err != nil {
					return nil, fmt.Errorf("search failed: %w", err)
				}

				// Convert search results to backups
				for _, hit := range searchResults.Hits {
					// Find the backup by ID
					for _, backup := range bh.backups {
						if backup.ID == hit.ID {
							backups = append(backups, backup)
							break
						}
					}
				}
			} else {
				backups = bh.ListBackups(filePath)
			}

			result := bh.formatBackupList(backups, query)
			return mcp.NewToolResultText(result), nil

		case "create":
			filePath, ok := args["file_path"].(string)
			if !ok {
				return nil, fmt.Errorf("file_path is required for create action")
			}

			context, ok := args["context"].(string)
			if !ok {
				return nil, fmt.Errorf("context is required for create action")
			}

			reasoning, ok := args["reasoning"].(string)
			if !ok {
				return nil, fmt.Errorf("reasoning is required for create action")
			}

			backup, err := bh.CreateBackup(filePath, context, reasoning)
			if err != nil {
				return nil, err
			}

			result := fmt.Sprintf("✅ Backup created successfully\n\n")
			result += fmt.Sprintf("ID: %s\n", backup.ID)
			result += fmt.Sprintf("Original: %s\n", backup.OriginalPath)
			result += fmt.Sprintf("Backup: %s\n", backup.BackupPath)
			result += fmt.Sprintf("Size: %d bytes\n", backup.FileSize)
			result += fmt.Sprintf("Time: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))

			return mcp.NewToolResultText(result), nil

		case "restore":
			backupID, ok := args["backup_id"].(string)
			if !ok {
				return nil, fmt.Errorf("backup_id is required for restore action")
			}

			if err := bh.RestoreBackup(backupID); err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("✅ Backup %s restored successfully", backupID)), nil

		case "clean":
			maxAgeDaysFloat, ok := args["max_age_days"].(float64)
			if !ok {
				return nil, fmt.Errorf("max_age_days is required for clean action")
			}
			maxAgeDays := int(maxAgeDaysFloat)

			removedCount, err := bh.CleanOldBackups(maxAgeDays)
			if err != nil {
				return nil, err
			}

			result := fmt.Sprintf("🧹 Cleanup completed\n\n")
			result += fmt.Sprintf("Removed %d backups older than %d days\n", removedCount, maxAgeDays)
			result += fmt.Sprintf("Remaining backups: %d\n", len(bh.backups))

			return mcp.NewToolResultText(result), nil

		default:
			return nil, fmt.Errorf("invalid action: %s", action)
		}
	}
}

// formatBackupList formats backup list for display
func (bh *BackupHandler) formatBackupList(backups []models.Backup, query string) string {
	if len(backups) == 0 {
		result := "No backups found"
		if query != "" {
			result += fmt.Sprintf(" for search: %s", query)
		}

		// Get document count
		count, _ := bh.searchManager.GetDocumentCount(search.IndexTypeBackups)
		if count > 0 {
			result += fmt.Sprintf("\n\nThere are %d backups available. Try:\n", count)
			result += "- File names or paths\n"
			result += "- File types (e.g., 'js', 'go', 'css')\n"
			result += "- Time-based searches (e.g., 'recent', 'today', 'this week')\n"
			result += "- Context or reasoning keywords\n"
		}

		return result
	}

	result := fmt.Sprintf("Found %d backups", len(backups))
	if query != "" {
		result += fmt.Sprintf(" for search: %s", query)
	}
	result += "\n\n"

	// Group by recency
	var today, thisWeek, older []models.Backup
	now := time.Now()

	for _, backup := range backups {
		daysSince := now.Sub(backup.Timestamp).Hours() / 24
		if daysSince < 1 {
			today = append(today, backup)
		} else if daysSince < 7 {
			thisWeek = append(thisWeek, backup)
		} else {
			older = append(older, backup)
		}
	}

	// Display by recency
	if len(today) > 0 {
		result += "📅 TODAY:\n"
		for _, backup := range today {
			result += bh.formatBackupEntry(backup)
		}
		result += "\n"
	}

	if len(thisWeek) > 0 {
		result += "📅 THIS WEEK:\n"
		for _, backup := range thisWeek {
			result += bh.formatBackupEntry(backup)
		}
		result += "\n"
	}

	if len(older) > 0 {
		result += "📅 OLDER:\n"
		for _, backup := range older {
			result += bh.formatBackupEntry(backup)
		}
	}

	// Add restore instructions
	result += "\n💡 To restore a backup, use action 'restore' with the backup ID"

	return result
}

// formatBackupEntry formats a single backup entry
func (bh *BackupHandler) formatBackupEntry(backup models.Backup) string {
	result := fmt.Sprintf("\n📦 ID: %s\n", backup.ID)
	result += fmt.Sprintf("   File: %s\n", backup.OriginalPath)
	result += fmt.Sprintf("   Time: %s (%s)\n",
		backup.Timestamp.Format("2006-01-02 15:04:05"),
		bh.formatTimeAgo(backup.Timestamp))
	result += fmt.Sprintf("   Size: %s\n", bh.formatFileSize(backup.FileSize))
	result += fmt.Sprintf("   Context: %s\n", backup.ChangeContext)
	if backup.Reasoning != "" {
		result += fmt.Sprintf("   Reasoning: %s\n", backup.Reasoning)
	}
	return result
}

// formatTimeAgo formats time as relative duration
func (bh *BackupHandler) formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration.Hours() < 1 {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else if duration.Hours() < 24*7 {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration.Hours() < 24*30 {
		weeks := int(duration.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else {
		months := int(duration.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// formatFileSize formats file size in human-readable format
func (bh *BackupHandler) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
