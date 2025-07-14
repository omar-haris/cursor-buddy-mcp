package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

// No need for custom analyzer registration - using standard analyzer

// IndexType represents the type of index
type IndexType string

const (
	IndexTypeRules     IndexType = "rules"
	IndexTypeKnowledge IndexType = "knowledge"
	IndexTypeTodos     IndexType = "todos"
	IndexTypeHistory   IndexType = "history"
	IndexTypeDatabase  IndexType = "database"
	IndexTypeBackups   IndexType = "backups"
)

// SearchManager manages all Bleve indexes
type SearchManager struct {
	basePath string
	indexes  map[IndexType]bleve.Index
	mu       sync.RWMutex
}

// NewSearchManager creates a new search manager
func NewSearchManager(basePath string) (*SearchManager, error) {
	sm := &SearchManager{
		basePath: basePath,
		indexes:  make(map[IndexType]bleve.Index),
	}

	// Create indexes directory if it doesn't exist
	indexesPath := filepath.Join(basePath, "indexes")
	if err := os.MkdirAll(indexesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create indexes directory: %w", err)
	}

	// Initialize all indexes
	indexTypes := []IndexType{
		IndexTypeRules,
		IndexTypeKnowledge,
		IndexTypeTodos,
		IndexTypeHistory,
		IndexTypeDatabase,
		IndexTypeBackups,
	}

	for _, indexType := range indexTypes {
		if err := sm.initializeIndex(indexType); err != nil {
			return nil, fmt.Errorf("failed to initialize %s index: %w", indexType, err)
		}
	}

	return sm, nil
}

// initializeIndex initializes or opens an index
func (sm *SearchManager) initializeIndex(indexType IndexType) error {
	indexPath := filepath.Join(sm.basePath, "indexes", string(indexType))

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Create new index with custom mapping
		mapping := sm.createIndexMapping(indexType)
		index, err := bleve.New(indexPath, mapping)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
		sm.indexes[indexType] = index
	} else {
		// Open existing index
		index, err := bleve.Open(indexPath)
		if err != nil {
			return fmt.Errorf("failed to open index: %w", err)
		}
		sm.indexes[indexType] = index
	}

	return nil
}

// createIndexMapping creates a custom mapping for an index type
func (sm *SearchManager) createIndexMapping(indexType IndexType) mapping.IndexMapping {
	// Create mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = "standard"

	// Create document mappings based on type
	switch indexType {
	case IndexTypeRules:
		ruleMapping := bleve.NewDocumentMapping()

		// ID field
		idField := bleve.NewTextFieldMapping()
		idField.Store = true
		idField.Index = false
		ruleMapping.AddFieldMappingsAt("id", idField)

		// Title field with higher weight
		titleField := bleve.NewTextFieldMapping()
		titleField.Store = true
		titleField.IncludeInAll = true
		ruleMapping.AddFieldMappingsAt("title", titleField)

		// Category field
		categoryField := bleve.NewTextFieldMapping()
		categoryField.Store = true
		categoryField.IncludeInAll = true
		ruleMapping.AddFieldMappingsAt("category", categoryField)

		// Content field
		contentField := bleve.NewTextFieldMapping()
		contentField.Store = true
		contentField.IncludeInAll = true
		ruleMapping.AddFieldMappingsAt("content", contentField)

		// Priority field
		priorityField := bleve.NewTextFieldMapping()
		priorityField.Store = true
		priorityField.IncludeInAll = true
		ruleMapping.AddFieldMappingsAt("priority", priorityField)

		indexMapping.AddDocumentMapping("rule", ruleMapping)
		indexMapping.DefaultMapping = ruleMapping

	case IndexTypeKnowledge:
		knowledgeMapping := bleve.NewDocumentMapping()

		// ID field
		idField := bleve.NewTextFieldMapping()
		idField.Store = true
		idField.Index = false
		knowledgeMapping.AddFieldMappingsAt("id", idField)

		// Title field
		titleField := bleve.NewTextFieldMapping()
		titleField.Store = true
		titleField.IncludeInAll = true
		knowledgeMapping.AddFieldMappingsAt("title", titleField)

		// Category field
		categoryField := bleve.NewTextFieldMapping()
		categoryField.Store = true
		categoryField.IncludeInAll = true
		knowledgeMapping.AddFieldMappingsAt("category", categoryField)

		// Content field
		contentField := bleve.NewTextFieldMapping()
		contentField.Store = true
		contentField.IncludeInAll = true
		knowledgeMapping.AddFieldMappingsAt("content", contentField)

		// Tags field
		tagsField := bleve.NewTextFieldMapping()
		tagsField.Store = true
		tagsField.IncludeInAll = true
		knowledgeMapping.AddFieldMappingsAt("tags", tagsField)

		indexMapping.AddDocumentMapping("knowledge", knowledgeMapping)
		indexMapping.DefaultMapping = knowledgeMapping

	case IndexTypeTodos:
		todoMapping := bleve.NewDocumentMapping()

		// ID field
		idField := bleve.NewTextFieldMapping()
		idField.Store = true
		idField.Index = false
		todoMapping.AddFieldMappingsAt("id", idField)

		// Task field
		taskField := bleve.NewTextFieldMapping()
		taskField.Store = true
		taskField.IncludeInAll = true
		todoMapping.AddFieldMappingsAt("task", taskField)

		// Feature field
		featureField := bleve.NewTextFieldMapping()
		featureField.Store = true
		featureField.IncludeInAll = true
		todoMapping.AddFieldMappingsAt("feature", featureField)

		// Completed field
		completedField := bleve.NewBooleanFieldMapping()
		completedField.Store = true
		completedField.IncludeInAll = false
		todoMapping.AddFieldMappingsAt("completed", completedField)

		// Status field for text search
		statusField := bleve.NewTextFieldMapping()
		statusField.Store = true
		statusField.IncludeInAll = true
		todoMapping.AddFieldMappingsAt("status", statusField)

		indexMapping.AddDocumentMapping("todo", todoMapping)
		indexMapping.DefaultMapping = todoMapping

	case IndexTypeHistory:
		historyMapping := bleve.NewDocumentMapping()

		// ID field
		idField := bleve.NewTextFieldMapping()
		idField.Store = true
		idField.Index = false
		historyMapping.AddFieldMappingsAt("id", idField)

		// Feature field
		featureField := bleve.NewTextFieldMapping()
		featureField.Store = true
		featureField.IncludeInAll = true
		historyMapping.AddFieldMappingsAt("feature", featureField)

		// Description field
		descriptionField := bleve.NewTextFieldMapping()
		descriptionField.Store = true
		descriptionField.IncludeInAll = true
		historyMapping.AddFieldMappingsAt("description", descriptionField)

		// Reasoning field
		reasoningField := bleve.NewTextFieldMapping()
		reasoningField.Store = true
		reasoningField.IncludeInAll = true
		historyMapping.AddFieldMappingsAt("reasoning", reasoningField)

		// Files field
		filesField := bleve.NewTextFieldMapping()
		filesField.Store = true
		filesField.IncludeInAll = true
		historyMapping.AddFieldMappingsAt("files", filesField)

		// Timestamp field
		timestampField := bleve.NewDateTimeFieldMapping()
		timestampField.Store = true
		timestampField.IncludeInAll = false
		historyMapping.AddFieldMappingsAt("timestamp", timestampField)

		indexMapping.AddDocumentMapping("history", historyMapping)
		indexMapping.DefaultMapping = historyMapping

	case IndexTypeDatabase:
		databaseMapping := bleve.NewDocumentMapping()

		// Table name field
		tableNameField := bleve.NewTextFieldMapping()
		tableNameField.Store = true
		tableNameField.IncludeInAll = true
		databaseMapping.AddFieldMappingsAt("table_name", tableNameField)

		// Columns field
		columnsField := bleve.NewTextFieldMapping()
		columnsField.Store = true
		columnsField.IncludeInAll = true
		databaseMapping.AddFieldMappingsAt("columns", columnsField)

		// Indexes field
		indexesField := bleve.NewTextFieldMapping()
		indexesField.Store = true
		indexesField.IncludeInAll = true
		databaseMapping.AddFieldMappingsAt("indexes", indexesField)

		// Description field
		descriptionField := bleve.NewTextFieldMapping()
		descriptionField.Store = true
		descriptionField.IncludeInAll = true
		databaseMapping.AddFieldMappingsAt("description", descriptionField)

		indexMapping.AddDocumentMapping("database", databaseMapping)
		indexMapping.DefaultMapping = databaseMapping

	case IndexTypeBackups:
		backupMapping := bleve.NewDocumentMapping()

		// ID field
		idField := bleve.NewTextFieldMapping()
		idField.Store = true
		idField.Index = false
		backupMapping.AddFieldMappingsAt("id", idField)

		// Original path field
		pathField := bleve.NewTextFieldMapping()
		pathField.Store = true
		pathField.IncludeInAll = true
		backupMapping.AddFieldMappingsAt("original_path", pathField)

		// Context field
		contextField := bleve.NewTextFieldMapping()
		contextField.Store = true
		contextField.IncludeInAll = true
		backupMapping.AddFieldMappingsAt("context", contextField)

		// Reasoning field
		reasoningField := bleve.NewTextFieldMapping()
		reasoningField.Store = true
		reasoningField.IncludeInAll = true
		backupMapping.AddFieldMappingsAt("reasoning", reasoningField)

		// Timestamp field
		timestampField := bleve.NewDateTimeFieldMapping()
		timestampField.Store = true
		timestampField.IncludeInAll = false
		backupMapping.AddFieldMappingsAt("timestamp", timestampField)

		indexMapping.AddDocumentMapping("backup", backupMapping)
		indexMapping.DefaultMapping = backupMapping
	}

	return indexMapping
}

// IndexDocument indexes a document
func (sm *SearchManager) IndexDocument(indexType IndexType, id string, doc interface{}) error {
	sm.mu.RLock()
	index, exists := sm.indexes[indexType]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("index %s not found", indexType)
	}

	return index.Index(id, doc)
}

// UpdateDocument updates a document in the index
func (sm *SearchManager) UpdateDocument(indexType IndexType, id string, doc interface{}) error {
	// Bleve's Index method automatically updates if document exists
	return sm.IndexDocument(indexType, id, doc)
}

// DeleteDocument deletes a document from the index
func (sm *SearchManager) DeleteDocument(indexType IndexType, id string) error {
	sm.mu.RLock()
	index, exists := sm.indexes[indexType]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("index %s not found", indexType)
	}

	return index.Delete(id)
}

// Search performs a search on an index
func (sm *SearchManager) Search(indexType IndexType, queryStr string, size int) (*bleve.SearchResult, error) {
	sm.mu.RLock()
	index, exists := sm.indexes[indexType]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", indexType)
	}

	// Create a query
	var q query.Query

	// If query is empty, match all
	if queryStr == "" || queryStr == "*" {
		q = bleve.NewMatchAllQuery()
	} else {
		// Use a disjunction query to search across multiple fields with different boosts
		disjunction := bleve.NewDisjunctionQuery()

		// Fuzzy match query for typo tolerance
		fuzzyQuery := bleve.NewFuzzyQuery(queryStr)
		fuzzyQuery.SetFuzziness(2)
		disjunction.AddQuery(fuzzyQuery)

		// Match query for exact terms
		matchQuery := bleve.NewMatchQuery(queryStr)
		matchQuery.SetBoost(2.0)
		disjunction.AddQuery(matchQuery)

		// Prefix query for partial matches
		prefixQuery := bleve.NewPrefixQuery(queryStr)
		prefixQuery.SetBoost(1.5)
		disjunction.AddQuery(prefixQuery)

		// Wildcard query for more flexibility
		wildcardQuery := bleve.NewWildcardQuery("*" + queryStr + "*")
		disjunction.AddQuery(wildcardQuery)

		q = disjunction
	}

	// Create search request
	searchRequest := bleve.NewSearchRequest(q)
	searchRequest.Size = size
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Fields = []string{"*"} // Return all stored fields

	// Add facets for better filtering
	if indexType == IndexTypeRules || indexType == IndexTypeKnowledge {
		searchRequest.AddFacet("category", bleve.NewFacetRequest("category", 10))
	}
	if indexType == IndexTypeRules {
		searchRequest.AddFacet("priority", bleve.NewFacetRequest("priority", 5))
	}

	return index.Search(searchRequest)
}

// SearchWithFilters performs a search with additional filters
func (sm *SearchManager) SearchWithFilters(indexType IndexType, queryStr string, filters map[string]interface{}, size int) (*bleve.SearchResult, error) {
	sm.mu.RLock()
	index, exists := sm.indexes[indexType]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", indexType)
	}

	// Build main query
	var mainQuery query.Query
	if queryStr == "" || queryStr == "*" {
		mainQuery = bleve.NewMatchAllQuery()
	} else {
		// Same as Search method
		disjunction := bleve.NewDisjunctionQuery()

		fuzzyQuery := bleve.NewFuzzyQuery(queryStr)
		fuzzyQuery.SetFuzziness(2)
		disjunction.AddQuery(fuzzyQuery)

		matchQuery := bleve.NewMatchQuery(queryStr)
		matchQuery.SetBoost(2.0)
		disjunction.AddQuery(matchQuery)

		prefixQuery := bleve.NewPrefixQuery(queryStr)
		prefixQuery.SetBoost(1.5)
		disjunction.AddQuery(prefixQuery)

		wildcardQuery := bleve.NewWildcardQuery("*" + queryStr + "*")
		disjunction.AddQuery(wildcardQuery)

		mainQuery = disjunction
	}

	// Apply filters
	if len(filters) > 0 {
		conjunctionQuery := bleve.NewConjunctionQuery()
		conjunctionQuery.AddQuery(mainQuery)

		for field, value := range filters {
			switch v := value.(type) {
			case string:
				termQuery := bleve.NewTermQuery(v)
				termQuery.SetField(field)
				conjunctionQuery.AddQuery(termQuery)
			case bool:
				boolQuery := bleve.NewBoolFieldQuery(v)
				boolQuery.SetField(field)
				conjunctionQuery.AddQuery(boolQuery)
			}
		}

		mainQuery = conjunctionQuery
	}

	// Create search request
	searchRequest := bleve.NewSearchRequest(mainQuery)
	searchRequest.Size = size
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Fields = []string{"*"}

	return index.Search(searchRequest)
}

// ReindexAll reindexes all documents in an index
func (sm *SearchManager) ReindexAll(indexType IndexType) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Close existing index
	if index, exists := sm.indexes[indexType]; exists {
		index.Close()
	}

	// Delete index directory
	indexPath := filepath.Join(sm.basePath, "indexes", string(indexType))
	if err := os.RemoveAll(indexPath); err != nil {
		return fmt.Errorf("failed to remove index directory: %w", err)
	}

	// Reinitialize index
	return sm.initializeIndex(indexType)
}

// Close closes all indexes
func (sm *SearchManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, index := range sm.indexes {
		if err := index.Close(); err != nil {
			return err
		}
	}

	return nil
}

// GetDocumentCount returns the number of documents in an index
func (sm *SearchManager) GetDocumentCount(indexType IndexType) (uint64, error) {
	sm.mu.RLock()
	index, exists := sm.indexes[indexType]
	sm.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("index %s not found", indexType)
	}

	return index.DocCount()
}
