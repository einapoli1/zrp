package main

import (
	"encoding/json"
	"testing"
)

func TestBuildSearchSQL(t *testing.T) {
	tests := []struct {
		name        string
		filters     []SearchFilter
		entityType  string
		searchText  string
		wantContain string
		wantErr     bool
	}{
		{
			name: "Simple equals filter",
			filters: []SearchFilter{
				{Field: "status", Operator: "eq", Value: "open"},
			},
			entityType:  "workorders",
			wantContain: "status = ?",
			wantErr:     false,
		},
		{
			name: "Multiple AND filters",
			filters: []SearchFilter{
				{Field: "status", Operator: "eq", Value: "open", AndOr: "AND"},
				{Field: "priority", Operator: "eq", Value: "high", AndOr: "AND"},
			},
			entityType:  "workorders",
			wantContain: "status = ?",
			wantErr:     false,
		},
		{
			name: "Contains filter with wildcard",
			filters: []SearchFilter{
				{Field: "title", Operator: "contains", Value: "capacitor"},
			},
			entityType:  "ecos",
			wantContain: "title LIKE ?",
			wantErr:     false,
		},
		{
			name: "Greater than filter",
			filters: []SearchFilter{
				{Field: "qty_on_hand", Operator: "gt", Value: 10},
			},
			entityType:  "inventory",
			wantContain: "qty_on_hand > ?",
			wantErr:     false,
		},
		{
			name: "Between filter",
			filters: []SearchFilter{
				{Field: "created_at", Operator: "between", Value: []interface{}{"2026-01-01", "2026-12-31"}},
			},
			entityType:  "ecos",
			wantContain: "BETWEEN",
			wantErr:     false,
		},
		{
			name: "IN filter",
			filters: []SearchFilter{
				{Field: "status", Operator: "in", Value: []interface{}{"open", "in_progress"}},
			},
			entityType:  "workorders",
			wantContain: "IN",
			wantErr:     false,
		},
		{
			name:        "Multi-field text search",
			filters:     []SearchFilter{},
			entityType:  "parts",
			searchText:  "capacitor",
			wantContain: "ipn LIKE ?",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args, err := BuildSearchSQL(tt.filters, tt.entityType, tt.searchText)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSearchSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if tt.wantContain != "" && !stringContains(whereClause, tt.wantContain) {
					t.Errorf("BuildSearchSQL() whereClause = %v, want to contain %v", whereClause, tt.wantContain)
				}
				
				// Check args length matches expected
				if tt.searchText != "" && len(args) == 0 {
					t.Errorf("BuildSearchSQL() with searchText should have args")
				}
			}
		})
	}
}

func TestGetQuickFilters(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		wantCount  int
	}{
		{
			name:       "Parts quick filters",
			entityType: "parts",
			wantCount:  2,
		},
		{
			name:       "Work orders quick filters",
			entityType: "workorders",
			wantCount:  3,
		},
		{
			name:       "Inventory quick filters",
			entityType: "inventory",
			wantCount:  2,
		},
		{
			name:       "ECOs quick filters",
			entityType: "ecos",
			wantCount:  3,
		},
		{
			name:       "Unknown entity type",
			entityType: "unknown",
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := GetQuickFilters(tt.entityType)
			if len(filters) != tt.wantCount {
				t.Errorf("GetQuickFilters(%s) returned %d filters, want %d", tt.entityType, len(filters), tt.wantCount)
			}
		})
	}
}

func TestParseSearchOperators(t *testing.T) {
	tests := []struct {
		name       string
		searchText string
		wantCount  int
		wantFirst  *SearchFilter
	}{
		{
			name:       "Field equals operator",
			searchText: "status:open",
			wantCount:  1,
			wantFirst:  &SearchFilter{Field: "status", Operator: "contains", Value: "open"},
		},
		{
			name:       "Field greater than",
			searchText: "qty>10",
			wantCount:  1,
			wantFirst:  &SearchFilter{Field: "qty", Operator: "gt", Value: 10.0},
		},
		{
			name:       "Field less than",
			searchText: "priority<5",
			wantCount:  1,
			wantFirst:  &SearchFilter{Field: "priority", Operator: "lt", Value: 5.0},
		},
		{
			name:       "Multiple operators",
			searchText: "status:open priority:high",
			wantCount:  2,
		},
		{
			name:       "No operators",
			searchText: "just plain text",
			wantCount:  0,
		},
		{
			name:       "Wildcard in value",
			searchText: "title:*capacitor*",
			wantCount:  1,
			wantFirst:  &SearchFilter{Field: "title", Operator: "contains", Value: "*capacitor*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := ParseSearchOperators(tt.searchText)
			
			if len(filters) != tt.wantCount {
				t.Errorf("ParseSearchOperators() returned %d filters, want %d", len(filters), tt.wantCount)
			}
			
			if tt.wantFirst != nil && len(filters) > 0 {
				first := filters[0]
				if first.Field != tt.wantFirst.Field {
					t.Errorf("First filter field = %v, want %v", first.Field, tt.wantFirst.Field)
				}
				if first.Operator != tt.wantFirst.Operator {
					t.Errorf("First filter operator = %v, want %v", first.Operator, tt.wantFirst.Operator)
				}
			}
		})
	}
}

func TestInitSearchTables(t *testing.T) {
	// Setup test database
	if err := initDB(":memory:"); err != nil {
		t.Fatalf("Failed to init test db: %v", err)
	}
	defer db.Close()

	// Test table creation
	err := InitSearchTables(db)
	if err != nil {
		t.Fatalf("InitSearchTables() failed: %v", err)
	}

	// Verify tables exist
	tables := []string{"saved_searches", "search_history"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s not found", table)
		}
	}

	// Test that running again doesn't fail (IF NOT EXISTS)
	err = InitSearchTables(db)
	if err != nil {
		t.Errorf("InitSearchTables() second run failed: %v", err)
	}
}

func TestSavedSearchWorkflow(t *testing.T) {
	// Setup test database
	if err := initDB(":memory:"); err != nil {
		t.Fatalf("Failed to init test db: %v", err)
	}
	defer db.Close()

	if err := InitSearchTables(db); err != nil {
		t.Fatalf("Failed to init search tables: %v", err)
	}

	// Create test saved search
	filters := []SearchFilter{
		{Field: "status", Operator: "eq", Value: "open", AndOr: "AND"},
		{Field: "priority", Operator: "eq", Value: "high"},
	}
	filtersJSON, _ := json.Marshal(filters)

	searchID := "test-search-1"
	_, err := db.Exec(`INSERT INTO saved_searches 
		(id, name, entity_type, filters, sort_by, sort_order, created_by, is_public)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		searchID, "High Priority Open Items", "workorders", string(filtersJSON),
		"created_at", "desc", "testuser", true)

	if err != nil {
		t.Fatalf("Failed to insert saved search: %v", err)
	}

	// Retrieve saved search
	var retrievedSearch SavedSearch
	var filtersStr string
	err = db.QueryRow(`SELECT id, name, entity_type, filters, sort_by, sort_order, created_by, is_public
		FROM saved_searches WHERE id = ?`, searchID).Scan(
		&retrievedSearch.ID, &retrievedSearch.Name, &retrievedSearch.EntityType,
		&filtersStr, &retrievedSearch.SortBy, &retrievedSearch.SortOrder,
		&retrievedSearch.CreatedBy, &retrievedSearch.IsPublic)

	if err != nil {
		t.Fatalf("Failed to retrieve saved search: %v", err)
	}

	if retrievedSearch.Name != "High Priority Open Items" {
		t.Errorf("Retrieved search name = %v, want 'High Priority Open Items'", retrievedSearch.Name)
	}

	if retrievedSearch.EntityType != "workorders" {
		t.Errorf("Retrieved search entity_type = %v, want 'workorders'", retrievedSearch.EntityType)
	}

	// Parse filters
	var parsedFilters []SearchFilter
	if err := json.Unmarshal([]byte(filtersStr), &parsedFilters); err != nil {
		t.Fatalf("Failed to parse filters: %v", err)
	}

	if len(parsedFilters) != 2 {
		t.Errorf("Parsed filters count = %d, want 2", len(parsedFilters))
	}

	// Delete saved search
	result, err := db.Exec("DELETE FROM saved_searches WHERE id = ? AND created_by = ?", searchID, "testuser")
	if err != nil {
		t.Fatalf("Failed to delete saved search: %v", err)
	}

	rows, _ := result.RowsAffected()
	if rows != 1 {
		t.Errorf("Delete affected %d rows, want 1", rows)
	}
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && (s[:len(substr)] == substr || stringContains(s[1:], substr))))
}
