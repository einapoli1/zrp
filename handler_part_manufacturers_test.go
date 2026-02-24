package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupManufacturersTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create parts table
	_, err = testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			category TEXT DEFAULT '',
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			manufacturer TEXT DEFAULT '',
			lifecycle TEXT DEFAULT 'active',
			status TEXT DEFAULT 'active',
			notes TEXT DEFAULT '',
			fields TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	// Create part_manufacturers table
	_, err = testDB.Exec(`
		CREATE TABLE part_manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_id TEXT NOT NULL,
			manufacturer TEXT NOT NULL,
			mpn TEXT NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 0,
			approved INTEGER NOT NULL DEFAULT 1,
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (part_id) REFERENCES parts(ipn) ON DELETE CASCADE,
			UNIQUE(part_id, manufacturer, mpn)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_manufacturers table: %v", err)
	}

	// Create index on part_id
	_, err = testDB.Exec(`CREATE INDEX idx_part_manufacturers_part_id ON part_manufacturers(part_id)`)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Create audit_log table (used by handlers)
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			username TEXT,
			action TEXT,
			table_name TEXT,
			record_id TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

// Test 1: List manufacturers for a part (empty list)
func TestListManufacturers_Empty(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with NO manufacturers in part_manufacturers table
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-001', 'Test Part')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/parts/TEST-001/manufacturers", nil)
	w := httptest.NewRecorder()

	handleGetPartManufacturers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data := response.Data.(map[string]interface{})
	manufacturers := data["manufacturers"].([]interface{})
	count := int(data["count"].(float64))

	if count != 0 {
		t.Errorf("Expected count=0, got %d", count)
	}

	if len(manufacturers) != 0 {
		t.Errorf("Expected empty list, got %d items", len(manufacturers))
	}
}

// Test 2: List manufacturers for a part (with data)
func TestListManufacturers_WithData(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-002', 'Test Part 2')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	// Add manufacturers (primary first, then secondaries)
	_, err = db.Exec(`
		INSERT INTO part_manufacturers (part_id, manufacturer, mpn, is_primary, approved, notes) 
		VALUES 
		('TEST-002', 'TI', 'TPS54360', 1, 1, 'Primary source'),
		('TEST-002', 'Analog Devices', 'LT1234', 0, 1, 'Secondary'),
		('TEST-002', 'Maxim', 'MAX5678', 0, 0, 'Not approved')
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturers: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/parts/TEST-002/manufacturers", nil)
	
	w := httptest.NewRecorder()

	handleGetPartManufacturers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data := response.Data.(map[string]interface{})
	manufacturers := data["manufacturers"].([]interface{})
	count := int(data["count"].(float64))

	if count != 3 {
		t.Errorf("Expected count=3, got %d", count)
	}

	if len(manufacturers) != 3 {
		t.Errorf("Expected 3 manufacturers, got %d", len(manufacturers))
	}

	// Check ordering: primary should be first
	first := manufacturers[0].(map[string]interface{})
	if first["manufacturer"] != "TI" {
		t.Errorf("Expected primary manufacturer 'TI' first, got %v", first["manufacturer"])
	}
	if !first["is_primary"].(bool) {
		t.Errorf("Expected first item to be primary")
	}
}

// Test 3: Add manufacturer (valid)
func TestAddManufacturer_Valid(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-003', 'Test Part 3')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	payload := map[string]interface{}{
		"manufacturer": "Texas Instruments",
		"mpn":          "TPS54360DDAR",
		"is_primary":   true,
		"approved":     true,
		"notes":        "Primary vendor",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/parts/TEST-003/manufacturers", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleCreatePartManufacturer(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify in database
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = 'TEST-003'`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 manufacturer in DB, got %d", count)
	}
}

// Test 4: Add manufacturer (duplicate should fail)
func TestAddManufacturer_Duplicate(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-004', 'Test Part 4')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	// Add manufacturer first time
	_, err = db.Exec(`
		INSERT INTO part_manufacturers (part_id, manufacturer, mpn, is_primary, approved) 
		VALUES ('TEST-004', 'TI', 'TPS123', 1, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}

	// Try to add duplicate
	payload := map[string]interface{}{
		"manufacturer": "TI",
		"mpn":          "TPS123",
		"is_primary":   false,
		"approved":     true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/parts/TEST-004/manufacturers", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleCreatePartManufacturer(w, req)

	if w.Code == http.StatusCreated {
		t.Errorf("Expected error for duplicate, got status %d", w.Code)
	}
}

// Test 5: Add manufacturer (primary handling - unsets old primary)
func TestAddManufacturer_PrimaryHandling(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with existing primary manufacturer
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-005', 'Test Part 5')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (part_id, manufacturer, mpn, is_primary, approved) 
		VALUES ('TEST-005', 'OldPrimary', 'OLD123', 1, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}

	// Add new manufacturer as primary
	payload := map[string]interface{}{
		"manufacturer": "NewPrimary",
		"mpn":          "NEW123",
		"is_primary":   true,
		"approved":     true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/parts/TEST-005/manufacturers", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleCreatePartManufacturer(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify only one primary manufacturer exists
	var primaryCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = 'TEST-005' AND is_primary = 1`).Scan(&primaryCount)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if primaryCount != 1 {
		t.Errorf("Expected 1 primary manufacturer, got %d", primaryCount)
	}

	// Verify the new one is primary
	var mfr string
	err = db.QueryRow(`SELECT manufacturer FROM part_manufacturers WHERE part_id = 'TEST-005' AND is_primary = 1`).Scan(&mfr)
	if err != nil {
		t.Fatalf("Failed to query primary manufacturer: %v", err)
	}

	if mfr != "NewPrimary" {
		t.Errorf("Expected 'NewPrimary' to be primary, got '%s'", mfr)
	}
}

// Test 6: Update manufacturer (change primary)
func TestUpdateManufacturer_ChangePrimary(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with two manufacturers
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-006', 'Test Part 6')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved) 
		VALUES 
		(1, 'TEST-006', 'Primary1', 'P1-123', 1, 1),
		(2, 'TEST-006', 'Secondary1', 'S1-123', 0, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturers: %v", err)
	}

	// Update secondary to be primary
	payload := map[string]interface{}{
		"is_primary": true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/parts/TEST-006/manufacturers/2", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleUpdatePartManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify only one primary
	var primaryCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = 'TEST-006' AND is_primary = 1`).Scan(&primaryCount)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if primaryCount != 1 {
		t.Errorf("Expected 1 primary manufacturer, got %d", primaryCount)
	}

	// Verify the correct one is primary
	var mfr string
	err = db.QueryRow(`SELECT manufacturer FROM part_manufacturers WHERE part_id = 'TEST-006' AND is_primary = 1`).Scan(&mfr)
	if err != nil {
		t.Fatalf("Failed to query primary manufacturer: %v", err)
	}

	if mfr != "Secondary1" {
		t.Errorf("Expected 'Secondary1' to be primary, got '%s'", mfr)
	}
}

// Test 7: Update manufacturer (change approved status)
func TestUpdateManufacturer_ChangeApproved(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with manufacturer
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-007', 'Test Part 7')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved) 
		VALUES (1, 'TEST-007', 'TestMfr', 'TM-123', 1, 0)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}

	// Update approved status
	payload := map[string]interface{}{
		"approved": true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/parts/TEST-007/manufacturers/1", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleUpdatePartManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify approved status changed
	var approved int
	err = db.QueryRow(`SELECT approved FROM part_manufacturers WHERE id = 1`).Scan(&approved)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if approved != 1 {
		t.Errorf("Expected approved=1, got %d", approved)
	}
}

// Test 8: Update manufacturer (change MPN and notes)
func TestUpdateManufacturer_ChangeMPNAndNotes(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with manufacturer
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-008', 'Test Part 8')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved, notes) 
		VALUES (1, 'TEST-008', 'TestMfr', 'OLD-MPN', 1, 1, 'Old notes')
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}

	// Update MPN and notes
	payload := map[string]interface{}{
		"mpn":   "NEW-MPN",
		"notes": "Updated notes",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/parts/TEST-008/manufacturers/1", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleUpdatePartManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify changes
	var mpn, notes string
	err = db.QueryRow(`SELECT mpn, notes FROM part_manufacturers WHERE id = 1`).Scan(&mpn, &notes)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if mpn != "NEW-MPN" {
		t.Errorf("Expected mpn='NEW-MPN', got '%s'", mpn)
	}

	if notes != "Updated notes" {
		t.Errorf("Expected notes='Updated notes', got '%s'", notes)
	}
}

// Test 9: Update manufacturer (validation - empty MPN should fail)
func TestUpdateManufacturer_ValidationEmptyMPN(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with manufacturer
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-009', 'Test Part 9')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved) 
		VALUES (1, 'TEST-009', 'TestMfr', 'VALID-MPN', 1, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}

	// Try to update with empty MPN
	payload := map[string]interface{}{
		"mpn": "",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/parts/TEST-009/manufacturers/1", bytes.NewReader(body))
	
	w := httptest.NewRecorder()

	handleUpdatePartManufacturer(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Expected error for empty MPN, got status %d", w.Code)
	}
}

// Test 10: Delete manufacturer (valid delete)
func TestDeleteManufacturer_Valid(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with two manufacturers
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-010', 'Test Part 10')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved) 
		VALUES 
		(1, 'TEST-010', 'Primary', 'P-123', 1, 1),
		(2, 'TEST-010', 'Secondary', 'S-123', 0, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturers: %v", err)
	}

	// Delete secondary manufacturer
	req := httptest.NewRequest("DELETE", "/api/v1/parts/TEST-010/manufacturers/2", nil)
	
	w := httptest.NewRecorder()

	handleDeletePartManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = 'TEST-010'`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 manufacturer remaining, got %d", count)
	}
}

// Test 11: Delete manufacturer (cannot delete last manufacturer)
func TestDeleteManufacturer_LastManufacturerProtection(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with only one manufacturer
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-011', 'Test Part 11')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved) 
		VALUES (1, 'TEST-011', 'OnlyOne', 'O-123', 1, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}

	// Try to delete the only manufacturer
	req := httptest.NewRequest("DELETE", "/api/v1/parts/TEST-011/manufacturers/1", nil)
	
	w := httptest.NewRecorder()

	handleDeletePartManufacturer(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Expected error when deleting last manufacturer, got status %d", w.Code)
	}

	// Verify it still exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = 'TEST-011'`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected manufacturer to still exist, got count=%d", count)
	}
}

// Test 12: Delete manufacturer (promote oldest secondary to primary)
func TestDeleteManufacturer_PromoteToPrimary(t *testing.T) {
	db = setupManufacturersTestDB(t)
	defer db.Close()

	// Create a test part with primary and secondaries
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-012', 'Test Part 12')`)
	if err != nil {
		t.Fatalf("Failed to insert test part: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO part_manufacturers (id, part_id, manufacturer, mpn, is_primary, approved, created_at) 
		VALUES 
		(1, 'TEST-012', 'Primary', 'P-123', 1, 1, '2024-01-01 10:00:00'),
		(2, 'TEST-012', 'Secondary1', 'S1-123', 0, 1, '2024-01-02 10:00:00'),
		(3, 'TEST-012', 'Secondary2', 'S2-123', 0, 1, '2024-01-03 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert manufacturers: %v", err)
	}

	// Delete primary manufacturer
	req := httptest.NewRequest("DELETE", "/api/v1/parts/TEST-012/manufacturers/1", nil)
	
	w := httptest.NewRecorder()

	handleDeletePartManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify oldest secondary is now primary
	var mfr string
	var isPrimary int
	err = db.QueryRow(`SELECT manufacturer, is_primary FROM part_manufacturers WHERE id = 2`).Scan(&mfr, &isPrimary)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if isPrimary != 1 {
		t.Errorf("Expected Secondary1 to be promoted to primary, got is_primary=%d", isPrimary)
	}

	if mfr != "Secondary1" {
		t.Errorf("Expected 'Secondary1' to be promoted, got '%s'", mfr)
	}

	// Verify only one primary exists
	var primaryCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = 'TEST-012' AND is_primary = 1`).Scan(&primaryCount)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if primaryCount != 1 {
		t.Errorf("Expected 1 primary manufacturer, got %d", primaryCount)
	}
}

// Test 13: Migration test - verify existing data migrated
func TestMigration_ExistingDataMigrated(t *testing.T) {
	testDB := setupManufacturersTestDB(t)
	defer testDB.Close()

	// Verify part_manufacturers table exists
	var tableName string
	err := testDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='part_manufacturers'`).Scan(&tableName)
	if err != nil {
		t.Fatalf("part_manufacturers table does not exist: %v", err)
	}

	// Test direct INSERT to part_manufacturers to verify it works
	// First create a test part
	_, err = testDB.Exec(`INSERT INTO parts (ipn, description) VALUES ('TEST-X', 'Test Part')`)
	if err != nil {
		t.Fatalf("Failed to insert test part for FK test: %v", err)
	}
	
	_, err = testDB.Exec(`INSERT INTO part_manufacturers (part_id, manufacturer, mpn, is_primary, approved) VALUES ('TEST-X', 'TestMfr', 'TEST-MPN', 1, 1)`)
	if err != nil {
		t.Fatalf("Direct INSERT to part_manufacturers failed: %v", err)
	}

	// Delete test records
	testDB.Exec(`DELETE FROM part_manufacturers WHERE part_id = 'TEST-X'`)
	testDB.Exec(`DELETE FROM parts WHERE ipn = 'TEST-X'`)

	// Create parts with old manufacturer/mpn fields
	_, err = testDB.Exec(`
		INSERT INTO parts (ipn, description, manufacturer, mpn) 
		VALUES 
		('MIGR-001', 'Migrated Part 1', 'OldMfr1', 'OLDMPN-001'),
		('MIGR-002', 'Migrated Part 2', 'OldMfr2', 'OLDMPN-002')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test parts: %v", err)
	}

	// Run migration function
	err = migrateExistingManufacturers(testDB)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify part_manufacturers table has the migrated data
	var count int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM part_manufacturers`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query part_manufacturers: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 migrated manufacturers, got %d", count)
	}

	// Verify they're marked as primary
	var primaryCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE is_primary = 1`).Scan(&primaryCount)
	if err != nil {
		t.Fatalf("Failed to query primary count: %v", err)
	}

	if primaryCount != 2 {
		t.Errorf("Expected 2 primary manufacturers (migrated as primary), got %d", primaryCount)
	}

	// Verify specific migrated data
	var mfr, mpn string
	err = testDB.QueryRow(`SELECT manufacturer, mpn FROM part_manufacturers WHERE part_id = 'MIGR-001'`).Scan(&mfr, &mpn)
	if err != nil {
		t.Fatalf("Failed to query migrated data: %v", err)
	}

	if mfr != "OldMfr1" || mpn != "OLDMPN-001" {
		t.Errorf("Expected OldMfr1/OLDMPN-001, got %s/%s", mfr, mpn)
	}
}
