package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func setupManufacturersHandlerTestDB(t *testing.T) *sql.DB {
	tmpFile := fmt.Sprintf("/tmp/zrp_manufacturers_handler_test_%d.db", os.Getpid())
	os.Remove(tmpFile)
	t.Cleanup(func() { os.Remove(tmpFile) })

	testDB, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create manufacturers table
	_, err = testDB.Exec(`
		CREATE TABLE manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			contact_name TEXT DEFAULT '',
			contact_email TEXT DEFAULT '',
			contact_phone TEXT DEFAULT '',
			website TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			approved INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create manufacturers table: %v", err)
	}

	// Create part_manufacturers table for FK constraints
	_, err = testDB.Exec(`
		CREATE TABLE part_manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_id TEXT NOT NULL,
			manufacturer_id INTEGER,
			mpn TEXT NOT NULL,
			FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id) ON DELETE RESTRICT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_manufacturers table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT DEFAULT CURRENT_TIMESTAMP,
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

// Test 1: List manufacturers - empty list
func TestManufacturersHandler_ListEmpty(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/manufacturers", nil)
	w := httptest.NewRecorder()

	handleListManufacturers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	manufacturers := data["manufacturers"].([]interface{})

	if len(manufacturers) != 0 {
		t.Errorf("Expected 0 manufacturers, got %d", len(manufacturers))
	}
}

// Test 2: List manufacturers - with data
func TestManufacturersHandler_ListWithData(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	// Insert test manufacturers
	manufacturers := []string{"Texas Instruments", "Murata", "Yageo", "Analog Devices"}
	for _, name := range manufacturers {
		_, err := db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, name)
		if err != nil {
			t.Fatalf("Failed to insert test manufacturer: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/manufacturers", nil)
	w := httptest.NewRecorder()

	handleListManufacturers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	mfrList := data["manufacturers"].([]interface{})

	if len(mfrList) != 4 {
		t.Errorf("Expected 4 manufacturers, got %d", len(mfrList))
	}

	// Verify sorted by name (Analog Devices should be first)
	firstMfr := mfrList[0].(map[string]interface{})
	if firstMfr["name"].(string) != "Analog Devices" {
		t.Errorf("Expected first manufacturer to be Analog Devices, got %s", firstMfr["name"])
	}
}

// Test 3: List manufacturers - with search filter
func TestManufacturersHandler_ListSearch(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	manufacturers := []string{"Texas Instruments", "Murata", "Yageo"}
	for _, name := range manufacturers {
		db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, name)
	}

	req := httptest.NewRequest("GET", "/api/v1/manufacturers?search=texas", nil)
	w := httptest.NewRecorder()

	handleListManufacturers(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	mfrList := data["manufacturers"].([]interface{})

	if len(mfrList) != 1 {
		t.Errorf("Expected 1 manufacturer, got %d", len(mfrList))
	}

	if len(mfrList) > 0 {
		firstMfr := mfrList[0].(map[string]interface{})
		if firstMfr["name"].(string) != "Texas Instruments" {
			t.Errorf("Expected Texas Instruments, got %s", firstMfr["name"])
		}
	}
}

// Test 4: Get manufacturer by ID
func TestManufacturersHandler_Get(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	result, err := db.Exec(`INSERT INTO manufacturers (name, contact_email, website) VALUES (?, ?, ?)`,
		"Texas Instruments", "support@ti.com", "https://www.ti.com")
	if err != nil {
		t.Fatalf("Failed to insert manufacturer: %v", err)
	}
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/manufacturers/%d", id), nil)
	w := httptest.NewRecorder()

	handleGetManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	mfr := resp.Data.(map[string]interface{})

	if mfr["name"].(string) != "Texas Instruments" {
		t.Errorf("Expected Texas Instruments, got %s", mfr["name"])
	}
	if mfr["contact_email"].(string) != "support@ti.com" {
		t.Errorf("Expected support@ti.com, got %s", mfr["contact_email"])
	}
}

// Test 5: Get manufacturer - not found
func TestManufacturersHandler_GetNotFound(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/manufacturers/999", nil)
	w := httptest.NewRecorder()

	handleGetManufacturer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test 6: Create manufacturer - success
func TestManufacturersHandler_CreateSuccess(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	payload := map[string]interface{}{
		"name":          "Texas Instruments",
		"contact_email": "support@ti.com",
		"website":       "https://www.ti.com",
		"approved":      true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/manufacturers", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateManufacturer(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify manufacturer was created
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM manufacturers WHERE name = ?`, "Texas Instruments").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 manufacturer in database, got %d", count)
	}
}

// Test 7: Create manufacturer - duplicate (exact match case-insensitive)
func TestManufacturersHandler_CreateDuplicateExact(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	// Insert existing manufacturer
	db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, "Texas Instruments")

	// Try to create duplicate with different case
	payload := map[string]interface{}{
		"name": "TEXAS INSTRUMENTS",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/manufacturers", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateManufacturer(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["match_type"].(string) != "exact" {
		t.Errorf("Expected exact match type, got %s", resp["match_type"])
	}
}

// Test 8: Create manufacturer - fuzzy match suggestions
func TestManufacturersHandler_CreateFuzzyMatch(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	// Insert existing manufacturer
	db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, "Murata")

	// Try to create similar name (1 character difference)
	payload := map[string]interface{}{
		"name": "Murataa", // Extra 'a'
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/manufacturers", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateManufacturer(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["match_type"].(string) != "fuzzy" {
		t.Errorf("Expected fuzzy match type, got %s", resp["match_type"])
	}

	suggestions := resp["suggestions"].([]interface{})
	if len(suggestions) == 0 {
		t.Error("Expected fuzzy match suggestions")
	}
}

// Test 9: Create manufacturer - validation (empty name)
func TestManufacturersHandler_CreateEmptyName(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	payload := map[string]interface{}{
		"name": "",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/manufacturers", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateManufacturer(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test 10: Update manufacturer - success
func TestManufacturersHandler_UpdateSuccess(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	result, _ := db.Exec(`INSERT INTO manufacturers (name, contact_email) VALUES (?, ?)`,
		"Murata", "old@murata.com")
	id, _ := result.LastInsertId()

	payload := map[string]interface{}{
		"contact_email": "new@murata.com",
		"website":       "https://www.murata.com",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/manufacturers/%d", id), bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleUpdateManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify update
	var email, website string
	db.QueryRow(`SELECT contact_email, website FROM manufacturers WHERE id = ?`, id).Scan(&email, &website)
	if email != "new@murata.com" {
		t.Errorf("Expected new@murata.com, got %s", email)
	}
	if website != "https://www.murata.com" {
		t.Errorf("Expected https://www.murata.com, got %s", website)
	}
}

// Test 11: Update manufacturer - not found
func TestManufacturersHandler_UpdateNotFound(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	payload := map[string]interface{}{
		"name": "Updated Name",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("PUT", "/api/v1/manufacturers/999", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleUpdateManufacturer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test 12: Delete manufacturer - success
func TestManufacturersHandler_DeleteSuccess(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	result, _ := db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, "Obsolete Manufacturer")
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/manufacturers/%d", id), nil)
	w := httptest.NewRecorder()

	handleDeleteManufacturer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deletion
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM manufacturers WHERE id = ?`, id).Scan(&count)
	if count != 0 {
		t.Error("Manufacturer was not deleted")
	}
}

// Test 13: Delete manufacturer - fail when parts reference it
func TestManufacturersHandler_DeleteHasParts(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	result, _ := db.Exec(`INSERT INTO manufacturers (name) VALUES (?)`, "Active Manufacturer")
	id, _ := result.LastInsertId()

	// Create a part_manufacturer referencing this manufacturer
	_, _ = db.Exec(`INSERT INTO part_manufacturers (part_id, manufacturer_id, mpn) VALUES (?, ?, ?)`,
		"PART-001", id, "MPN-12345")

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/manufacturers/%d", id), nil)
	w := httptest.NewRecorder()

	handleDeleteManufacturer(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	// Verify manufacturer still exists
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM manufacturers WHERE id = ?`, id).Scan(&count)
	if count != 1 {
		t.Error("Manufacturer should not have been deleted")
	}
}

// Test 14: Delete manufacturer - not found
func TestManufacturersHandler_DeleteNotFound(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("DELETE", "/api/v1/manufacturers/999", nil)
	w := httptest.NewRecorder()

	handleDeleteManufacturer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test 15: List manufacturers - approved filter
func TestManufacturersHandler_ListApprovedFilter(t *testing.T) {
	db = setupManufacturersHandlerTestDB(t)
	defer db.Close()

	// Insert approved and unapproved manufacturers
	db.Exec(`INSERT INTO manufacturers (name, approved) VALUES (?, ?)`, "Approved Inc", 1)
	db.Exec(`INSERT INTO manufacturers (name, approved) VALUES (?, ?)`, "Pending Corp", 0)

	req := httptest.NewRequest("GET", "/api/v1/manufacturers?approved=true", nil)
	w := httptest.NewRecorder()

	handleListManufacturers(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	mfrList := data["manufacturers"].([]interface{})

	if len(mfrList) != 1 {
		t.Errorf("Expected 1 approved manufacturer, got %d", len(mfrList))
	}

	if len(mfrList) > 0 {
		mfr := mfrList[0].(map[string]interface{})
		if mfr["name"].(string) != "Approved Inc" {
			t.Errorf("Expected Approved Inc, got %s", mfr["name"])
		}
	}
}
