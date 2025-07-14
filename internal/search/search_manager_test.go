package search

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearchManager(t *testing.T) {
	tempDir := t.TempDir()

	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	require.NotNil(t, sm)

	// Verify all index types are created
	indexTypes := []IndexType{
		IndexTypeRules,
		IndexTypeKnowledge,
		IndexTypeTodos,
		IndexTypeHistory,
		IndexTypeDatabase,
		IndexTypeBackups,
	}

	for _, indexType := range indexTypes {
		assert.Contains(t, sm.indexes, indexType)
		assert.NotNil(t, sm.indexes[indexType])
	}

	// Cleanup
	sm.Close()
}

func TestSearchManager_IndexDocument(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Test indexing a knowledge document
	doc := &KnowledgeDocument{
		ID:       "test-knowledge-1",
		Title:    "Test Knowledge",
		Category: "testing",
		Tags:     "test, knowledge",
		Content:  "This is test knowledge content about testing procedures",
	}

	err = sm.IndexDocument(IndexTypeKnowledge, doc.ID, doc)
	assert.NoError(t, err)

	// Test indexing a rule document
	ruleDoc := &RuleDocument{
		ID:          "test-rule-1",
		Title:       "Test Rule",
		Category:    "coding",
		Priority:    "critical",
		Description: "This is a test rule for code quality",
		Content:     "Always write unit tests for your code",
	}

	err = sm.IndexDocument(IndexTypeRules, ruleDoc.ID, ruleDoc)
	assert.NoError(t, err)
}

func TestSearchManager_SearchWithFilters(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Index test documents
	docs := []*KnowledgeDocument{
		{
			ID:       "kb-1",
			Title:    "Testing Best Practices",
			Category: "testing",
			Tags:     "testing, best-practices",
			Content:  "Unit testing is essential for code quality",
		},
		{
			ID:       "kb-2",
			Title:    "Code Review Guidelines",
			Category: "development",
			Tags:     "code-review, guidelines",
			Content:  "Code reviews help maintain code quality and share knowledge",
		},
		{
			ID:       "kb-3",
			Title:    "Testing Strategies",
			Category: "testing",
			Tags:     "testing, strategies",
			Content:  "Integration testing complements unit testing",
		},
	}

	for _, doc := range docs {
		err = sm.IndexDocument(IndexTypeKnowledge, doc.ID, doc)
		require.NoError(t, err)
	}

	// Wait for indexing to complete
	time.Sleep(100 * time.Millisecond)

	// Test basic search
	results, err := sm.SearchWithFilters(IndexTypeKnowledge, "testing", nil, 10)
	assert.NoError(t, err)
	assert.True(t, len(results.Hits) >= 2) // Should find at least 2 documents

	// Test search with category filter
	filters := map[string]interface{}{
		"category": "testing",
	}
	results, err = sm.SearchWithFilters(IndexTypeKnowledge, "testing", filters, 10)
	assert.NoError(t, err)
	assert.True(t, len(results.Hits) >= 2) // Should find documents in testing category

	// Test search with non-existent category
	filters = map[string]interface{}{
		"category": "nonexistent",
	}
	results, err = sm.SearchWithFilters(IndexTypeKnowledge, "testing", filters, 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results.Hits)) // Should find no documents
}

func TestSearchManager_Search(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Index test documents
	ruleDoc := &RuleDocument{
		ID:          "rule-1",
		Title:       "Code Quality Rule",
		Category:    "coding",
		Priority:    "critical",
		Description: "Always write unit tests",
		Content:     "Unit tests are essential for maintaining code quality",
	}

	err = sm.IndexDocument(IndexTypeRules, ruleDoc.ID, ruleDoc)
	require.NoError(t, err)

	// Wait for indexing
	time.Sleep(100 * time.Millisecond)

	// Test basic search
	results, err := sm.Search(IndexTypeRules, "unit tests", 10)
	assert.NoError(t, err)
	assert.True(t, len(results.Hits) >= 1)

	// Verify search result structure
	if len(results.Hits) > 0 {
		hit := results.Hits[0]
		assert.Equal(t, "rule-1", hit.ID)
		assert.Greater(t, hit.Score, 0.0)
	}
}

func TestSearchManager_DeleteDocument(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Index a document
	doc := &KnowledgeDocument{
		ID:       "kb-delete-test",
		Title:    "Document to Delete",
		Category: "testing",
		Tags:     "delete, test",
		Content:  "This document will be deleted",
	}

	err = sm.IndexDocument(IndexTypeKnowledge, doc.ID, doc)
	require.NoError(t, err)

	// Wait for indexing
	time.Sleep(100 * time.Millisecond)

	// Verify document can be found
	results, err := sm.Search(IndexTypeKnowledge, "delete", 10)
	assert.NoError(t, err)
	assert.True(t, len(results.Hits) >= 1)

	// Delete the document
	err = sm.DeleteDocument(IndexTypeKnowledge, doc.ID)
	assert.NoError(t, err)

	// Wait for deletion to complete
	time.Sleep(100 * time.Millisecond)

	// Verify document is no longer found
	results, err = sm.Search(IndexTypeKnowledge, "delete", 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results.Hits))
}

func TestSearchManager_ReindexAll(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Test reindexing (should not error even if no documents)
	err = sm.ReindexAll(IndexTypeKnowledge)
	assert.NoError(t, err)

	// Index some documents first
	doc := &KnowledgeDocument{
		ID:       "kb-reindex-test",
		Title:    "Reindex Test",
		Category: "testing",
		Tags:     "reindex, test",
		Content:  "This document tests reindexing",
	}

	err = sm.IndexDocument(IndexTypeKnowledge, doc.ID, doc)
	require.NoError(t, err)

	// Wait for indexing
	time.Sleep(100 * time.Millisecond)

	// Reindex again (this clears the index)
	err = sm.ReindexAll(IndexTypeKnowledge)
	assert.NoError(t, err)

	// Wait for reindexing to complete
	time.Sleep(100 * time.Millisecond)

	// Verify document is no longer searchable (reindex clears all documents)
	results, err := sm.Search(IndexTypeKnowledge, "reindex", 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results.Hits))
}

func TestSearchManager_GetDocumentCount(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Initially should be 0
	count, err := sm.GetDocumentCount(IndexTypeKnowledge)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	// Index a document
	doc := &KnowledgeDocument{
		ID:       "kb-count-test",
		Title:    "Count Test",
		Category: "testing",
		Tags:     "count, test",
		Content:  "This document tests counting",
	}

	err = sm.IndexDocument(IndexTypeKnowledge, doc.ID, doc)
	require.NoError(t, err)

	// Wait for indexing
	time.Sleep(100 * time.Millisecond)

	// Count should be 1
	count, err = sm.GetDocumentCount(IndexTypeKnowledge)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestSearchManager_DocumentTypes(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Test all document types with correct field names
	tests := []struct {
		name      string
		indexType IndexType
		document  interface{}
		docID     string
	}{
		{
			name:      "RuleDocument",
			indexType: IndexTypeRules,
			document: &RuleDocument{
				ID:          "rule-1",
				Title:       "Test Rule",
				Category:    "coding",
				Priority:    "critical",
				Description: "Test rule description",
				Content:     "Test rule content",
			},
			docID: "rule-1",
		},
		{
			name:      "KnowledgeDocument",
			indexType: IndexTypeKnowledge,
			document: &KnowledgeDocument{
				ID:       "kb-1",
				Title:    "Test Knowledge",
				Category: "testing",
				Tags:     "test",
				Content:  "Test knowledge content",
			},
			docID: "kb-1",
		},
		{
			name:      "TodoDocument",
			indexType: IndexTypeTodos,
			document: &TodoDocument{
				ID:        "todo-1",
				Task:      "Test Todo Task",
				Feature:   "testing",
				Status:    "pending",
				Completed: false,
			},
			docID: "todo-1",
		},
		{
			name:      "HistoryDocument",
			indexType: IndexTypeHistory,
			document: &HistoryDocument{
				ID:          "hist-1",
				Feature:     "testing",
				Description: "Test history description",
				Reasoning:   "Test reasoning",
				Files:       "test.go, main.go",
				Timestamp:   time.Now(),
			},
			docID: "hist-1",
		},
		{
			name:      "DatabaseDocument",
			indexType: IndexTypeDatabase,
			document: &DatabaseDocument{
				ID:          "db-1",
				TableName:   "test_table",
				Columns:     "id INTEGER, name TEXT",
				Indexes:     "idx_name",
				Description: "Test database table",
			},
			docID: "db-1",
		},
		{
			name:      "BackupDocument",
			indexType: IndexTypeBackups,
			document: &BackupDocument{
				ID:           "backup-1",
				OriginalPath: "/test/path.go",
				Context:      "test context",
				Reasoning:    "test reasoning",
				Timestamp:    time.Now(),
			},
			docID: "backup-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.IndexDocument(tt.indexType, tt.docID, tt.document)
			assert.NoError(t, err)

			// Wait for indexing
			time.Sleep(50 * time.Millisecond)

			// Verify document can be found
			results, err := sm.Search(tt.indexType, "test", 10)
			assert.NoError(t, err)
			assert.True(t, len(results.Hits) >= 1)

			// Clean up
			err = sm.DeleteDocument(tt.indexType, tt.docID)
			assert.NoError(t, err)
		})
	}
}

func TestSearchManager_Close(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)

	// Close should not error
	err = sm.Close()
	assert.NoError(t, err)
}

func TestSearchManager_InvalidIndexType(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Test with invalid index type
	invalidType := IndexType("invalid")

	doc := &KnowledgeDocument{
		ID:       "test",
		Title:    "Test",
		Category: "test",
		Content:  "test content",
	}

	err = sm.IndexDocument(invalidType, doc.ID, doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "index invalid not found")
}

func TestSearchManager_EmptySearch(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Search with empty query
	results, err := sm.Search(IndexTypeKnowledge, "", 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results.Hits))

	// Search with empty query and filters
	results, err = sm.SearchWithFilters(IndexTypeKnowledge, "", nil, 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results.Hits))
}

func TestSearchManager_IndexesDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create search manager
	sm, err := NewSearchManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Verify indexes directory is created
	indexesPath := tempDir + "/indexes"
	_, err = os.Stat(indexesPath)
	assert.NoError(t, err)

	// Verify individual index directories exist
	indexTypes := []IndexType{
		IndexTypeRules,
		IndexTypeKnowledge,
		IndexTypeTodos,
		IndexTypeHistory,
		IndexTypeDatabase,
		IndexTypeBackups,
	}

	for _, indexType := range indexTypes {
		indexPath := indexesPath + "/" + string(indexType)
		_, err = os.Stat(indexPath)
		assert.NoError(t, err, "Index directory should exist for %s", indexType)
	}
}
