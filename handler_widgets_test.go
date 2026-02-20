package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupWidgetsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create dashboard_widgets table
	_, err = testDB.Exec(`
		CREATE TABLE dashboard_widgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER DEFAULT 0,
			widget_type TEXT NOT NULL,
			position INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create dashboard_widgets table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

func insertTestWidget(t *testing.T, db *sql.DB, widgetType string, position, enabled int) int {
	res, err := db.Exec(
		"INSERT INTO dashboard_widgets (user_id, widget_type, position, enabled) VALUES (0, ?, ?, ?)",
		widgetType, position, enabled,
	)
	if err != nil {
		t.Fatalf("Failed to insert test widget: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// Test handleGetDashboardWidgets - Empty
func TestHandleGetDashboardWidgets_Empty(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()

	handleGetDashboardWidgets(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	widgets, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(widgets) != 0 {
		t.Errorf("Expected empty array, got %d widgets", len(widgets))
	}
}

// Test handleGetDashboardWidgets - With Data
func TestHandleGetDashboardWidgets_WithData(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "open_ecos", 1, 1)
	insertTestWidget(t, db, "low_stock", 2, 1)
	insertTestWidget(t, db, "active_wos", 3, 0)
	insertTestWidget(t, db, "open_ncrs", 4, 1)

	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()

	handleGetDashboardWidgets(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	widgetsData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(widgetsData) != 4 {
		t.Errorf("Expected 4 widgets, got %d", len(widgetsData))
	}

	// Verify they're ordered by position
	firstWidget := widgetsData[0].(map[string]interface{})
	if int(firstWidget["position"].(float64)) != 1 {
		t.Errorf("Expected first widget position 1, got %v", firstWidget["position"])
	}

	// Verify widget types are present
	for _, wData := range widgetsData {
		widget := wData.(map[string]interface{})
		if widget["widget_type"] == nil || widget["widget_type"] == "" {
			t.Errorf("Expected widget_type to be set")
		}
	}
}

// Test handleGetDashboardWidgets - Position Ordering
func TestHandleGetDashboardWidgets_Ordering(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert in non-sequential order
	insertTestWidget(t, db, "widget_c", 3, 1)
	insertTestWidget(t, db, "widget_a", 1, 1)
	insertTestWidget(t, db, "widget_d", 4, 1)
	insertTestWidget(t, db, "widget_b", 2, 1)

	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()

	handleGetDashboardWidgets(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	widgets := resp.Data.([]interface{})
	
	// Verify correct ascending position order
	for i, wData := range widgets {
		widget := wData.(map[string]interface{})
		expectedPos := i + 1
		actualPos := int(widget["position"].(float64))
		if actualPos != expectedPos {
			t.Errorf("Widget %d: expected position %d, got %d", i, expectedPos, actualPos)
		}
	}
}

// Test handleUpdateDashboardWidgets - Success
func TestHandleUpdateDashboardWidgets_Success(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "open_ecos", 1, 1)
	insertTestWidget(t, db, "low_stock", 2, 1)
	insertTestWidget(t, db, "active_wos", 3, 1)

	// Update widget configuration
	updates := []map[string]interface{}{
		{"widget_type": "open_ecos", "position": 3, "enabled": 0},
		{"widget_type": "low_stock", "position": 1, "enabled": 1},
		{"widget_type": "active_wos", "position": 2, "enabled": 0},
	}

	body, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDashboardWidgets(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify updates were applied
	var position, enabled int
	db.QueryRow("SELECT position, enabled FROM dashboard_widgets WHERE widget_type='open_ecos' AND user_id=0").Scan(&position, &enabled)
	if position != 3 {
		t.Errorf("Expected open_ecos position 3, got %d", position)
	}
	if enabled != 0 {
		t.Errorf("Expected open_ecos enabled 0, got %d", enabled)
	}

	db.QueryRow("SELECT position, enabled FROM dashboard_widgets WHERE widget_type='low_stock' AND user_id=0").Scan(&position, &enabled)
	if position != 1 {
		t.Errorf("Expected low_stock position 1, got %d", position)
	}
	if enabled != 1 {
		t.Errorf("Expected low_stock enabled 1, got %d", enabled)
	}
}

// Test handleUpdateDashboardWidgets - Partial Update
func TestHandleUpdateDashboardWidgets_PartialUpdate(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "widget_a", 1, 1)
	insertTestWidget(t, db, "widget_b", 2, 1)
	insertTestWidget(t, db, "widget_c", 3, 1)

	// Update only one widget
	updates := []map[string]interface{}{
		{"widget_type": "widget_b", "position": 5, "enabled": 0},
	}

	body, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDashboardWidgets(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify only widget_b was updated
	var position, enabled int
	db.QueryRow("SELECT position, enabled FROM dashboard_widgets WHERE widget_type='widget_b' AND user_id=0").Scan(&position, &enabled)
	if position != 5 || enabled != 0 {
		t.Errorf("Expected widget_b to be updated to position=5, enabled=0; got position=%d, enabled=%d", position, enabled)
	}

	// Verify other widgets unchanged
	db.QueryRow("SELECT position, enabled FROM dashboard_widgets WHERE widget_type='widget_a' AND user_id=0").Scan(&position, &enabled)
	if position != 1 || enabled != 1 {
		t.Errorf("Expected widget_a unchanged; got position=%d, enabled=%d", position, enabled)
	}
}

// Test handleUpdateDashboardWidgets - Toggle Enabled
func TestHandleUpdateDashboardWidgets_ToggleEnabled(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "test_widget", 1, 1)

	tests := []struct {
		name    string
		enabled int
	}{
		{"Disable widget", 0},
		{"Enable widget", 1},
		{"Disable again", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := []map[string]interface{}{
				{"widget_type": "test_widget", "position": 1, "enabled": tt.enabled},
			}

			body, _ := json.Marshal(updates)
			req := httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleUpdateDashboardWidgets(w, req)

			var enabled int
			db.QueryRow("SELECT enabled FROM dashboard_widgets WHERE widget_type='test_widget' AND user_id=0").Scan(&enabled)
			if enabled != tt.enabled {
				t.Errorf("Expected enabled %d, got %d", tt.enabled, enabled)
			}
		})
	}
}

// Test handleUpdateDashboardWidgets - Empty Update
func TestHandleUpdateDashboardWidgets_EmptyUpdate(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "test_widget", 1, 1)

	updates := []map[string]interface{}{}

	body, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDashboardWidgets(w, req)

	// Should succeed with empty update
	if w.Code != 200 {
		t.Errorf("Expected status 200 for empty update, got %d", w.Code)
	}
}

// Test handleUpdateDashboardWidgets - Invalid JSON
func TestHandleUpdateDashboardWidgets_InvalidJSON(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDashboardWidgets(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

// Test widget persistence across operations
func TestWidgetPersistence(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create initial widgets
	insertTestWidget(t, db, "widget_1", 1, 1)
	insertTestWidget(t, db, "widget_2", 2, 0)
	insertTestWidget(t, db, "widget_3", 3, 1)

	// Get initial state
	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)

	var resp1 APIResponse
	json.NewDecoder(w.Body).Decode(&resp1)
	widgets1 := resp1.Data.([]interface{})
	
	if len(widgets1) != 3 {
		t.Fatalf("Expected 3 widgets initially, got %d", len(widgets1))
	}

	// Update widgets
	updates := []map[string]interface{}{
		{"widget_type": "widget_1", "position": 3, "enabled": 0},
		{"widget_type": "widget_3", "position": 1, "enabled": 1},
	}

	body, _ := json.Marshal(updates)
	req = httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handleUpdateDashboardWidgets(w, req)

	// Get updated state
	req = httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w = httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)

	var resp2 APIResponse
	json.NewDecoder(w.Body).Decode(&resp2)
	widgets2 := resp2.Data.([]interface{})

	// Verify widget_1 changes persisted
	var widget1Found bool
	for _, wData := range widgets2 {
		widget := wData.(map[string]interface{})
		if widget["widget_type"] == "widget_1" {
			widget1Found = true
			if int(widget["position"].(float64)) != 3 {
				t.Errorf("Expected widget_1 position 3 after update, got %v", widget["position"])
			}
			if int(widget["enabled"].(float64)) != 0 {
				t.Errorf("Expected widget_1 enabled 0 after update, got %v", widget["enabled"])
			}
		}
	}

	if !widget1Found {
		t.Errorf("widget_1 not found after update")
	}
}

// Test valid widget types
func TestValidWidgetTypes(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	validTypes := []string{
		"open_ecos",
		"low_stock",
		"open_pos",
		"active_wos",
		"open_ncrs",
		"open_rmas",
		"total_parts",
		"total_devices",
		"recent_shipments",
		"pending_rfqs",
	}

	for i, widgetType := range validTypes {
		insertTestWidget(t, db, widgetType, i+1, 1)
	}

	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	widgets := resp.Data.([]interface{})
	if len(widgets) != len(validTypes) {
		t.Errorf("Expected %d widgets, got %d", len(validTypes), len(widgets))
	}

	// Verify all types are present
	foundTypes := make(map[string]bool)
	for _, wData := range widgets {
		widget := wData.(map[string]interface{})
		foundTypes[widget["widget_type"].(string)] = true
	}

	for _, widgetType := range validTypes {
		if !foundTypes[widgetType] {
			t.Errorf("Widget type '%s' not found in response", widgetType)
		}
	}
}

// Test user_id filtering (always 0 in current implementation)
func TestUserIDFiltering(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert widgets for different users
	db.Exec("INSERT INTO dashboard_widgets (user_id, widget_type, position, enabled) VALUES (0, 'widget_user0', 1, 1)")
	db.Exec("INSERT INTO dashboard_widgets (user_id, widget_type, position, enabled) VALUES (1, 'widget_user1', 2, 1)")
	db.Exec("INSERT INTO dashboard_widgets (user_id, widget_type, position, enabled) VALUES (2, 'widget_user2', 3, 1)")

	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	widgets := resp.Data.([]interface{})
	
	// Should only return widgets for user_id=0
	if len(widgets) != 1 {
		t.Errorf("Expected 1 widget (user_id=0), got %d", len(widgets))
	}

	if len(widgets) > 0 {
		widget := widgets[0].(map[string]interface{})
		if widget["widget_type"] != "widget_user0" {
			t.Errorf("Expected widget_user0, got %v", widget["widget_type"])
		}
	}
}

// Test layout persistence - reordering
func TestLayoutReordering(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "a", 1, 1)
	insertTestWidget(t, db, "b", 2, 1)
	insertTestWidget(t, db, "c", 3, 1)
	insertTestWidget(t, db, "d", 4, 1)

	// Reverse the order
	updates := []map[string]interface{}{
		{"widget_type": "a", "position": 4, "enabled": 1},
		{"widget_type": "b", "position": 3, "enabled": 1},
		{"widget_type": "c", "position": 2, "enabled": 1},
		{"widget_type": "d", "position": 1, "enabled": 1},
	}

	body, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/api/dashboard/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleUpdateDashboardWidgets(w, req)

	// Verify new order
	req = httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w = httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	widgets := resp.Data.([]interface{})

	expectedOrder := []string{"d", "c", "b", "a"}
	for i, wData := range widgets {
		widget := wData.(map[string]interface{})
		if widget["widget_type"] != expectedOrder[i] {
			t.Errorf("Position %d: expected '%s', got '%s'", i+1, expectedOrder[i], widget["widget_type"])
		}
	}
}

// Test mixed enabled/disabled widgets
func TestMixedEnabledDisabled(t *testing.T) {
	oldDB := db
	db = setupWidgetsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestWidget(t, db, "enabled_1", 1, 1)
	insertTestWidget(t, db, "disabled_1", 2, 0)
	insertTestWidget(t, db, "enabled_2", 3, 1)
	insertTestWidget(t, db, "disabled_2", 4, 0)

	req := httptest.NewRequest("GET", "/api/dashboard/widgets", nil)
	w := httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	widgets := resp.Data.([]interface{})

	// Should return all widgets (both enabled and disabled)
	if len(widgets) != 4 {
		t.Errorf("Expected 4 widgets, got %d", len(widgets))
	}

	// Verify enabled status is correct for each
	enabledCount := 0
	disabledCount := 0
	for _, wData := range widgets {
		widget := wData.(map[string]interface{})
		enabled := int(widget["enabled"].(float64))
		if enabled == 1 {
			enabledCount++
		} else {
			disabledCount++
		}
	}

	if enabledCount != 2 {
		t.Errorf("Expected 2 enabled widgets, got %d", enabledCount)
	}
	if disabledCount != 2 {
		t.Errorf("Expected 2 disabled widgets, got %d", disabledCount)
	}
}
