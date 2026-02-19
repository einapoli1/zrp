package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupVendorsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create vendors table
	_, err = testDB.Exec(`
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 0,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create vendors table: %v", err)
	}

	// Create purchase_orders table for FK checks
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	// Create rfq_vendors table for FK checks
	_, err = testDB.Exec(`
		CREATE TABLE rfq_vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			vendor_id TEXT NOT NULL,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rfq_vendors table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			timestamp TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Create changes table
	_, err = testDB.Exec(`
		CREATE TABLE changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			action TEXT NOT NULL,
			old_value TEXT,
			new_value TEXT,
			timestamp TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create changes table: %v", err)
	}

	// Create undo_stack table
	_, err = testDB.Exec(`
		CREATE TABLE undo_stack (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			timestamp TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create undo_stack table: %v", err)
	}

	return testDB
}

func createTestVendor(t *testing.T, db *sql.DB, id, name, website, contactName, contactEmail, contactPhone, notes, status string, leadTimeDays int) {
	_, err := db.Exec(`
		INSERT INTO vendors (id, name, website, contact_name, contact_email, contact_phone, notes, status, lead_time_days)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, name, website, contactName, contactEmail, contactPhone, notes, status, leadTimeDays)
	if err != nil {
		t.Fatalf("Failed to create test vendor: %v", err)
	}
}

func TestHandleListVendors(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	createTestVendor(t, db, "V-001", "Acme Corp", "https://acme.com", "John Doe", "john@acme.com", "555-1234", "Test notes", "active", 7)
	createTestVendor(t, db, "V-002", "Widget Inc", "https://widget.com", "Jane Smith", "jane@widget.com", "555-5678", "", "inactive", 14)

	req := httptest.NewRequest("GET", "/api/v1/vendors", nil)
	w := httptest.NewRecorder()

	handleListVendors(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	vendorsJSON, _ := json.Marshal(resp.Data)
	var vendors []Vendor
	if err := json.Unmarshal(vendorsJSON, &vendors); err != nil {
		t.Fatalf("Failed to unmarshal vendors: %v", err)
	}

	if len(vendors) != 2 {
		t.Errorf("Expected 2 vendors, got %d", len(vendors))
	}

	// Vendors should be sorted by name
	if vendors[0].Name != "Acme Corp" {
		t.Errorf("Expected first vendor to be Acme Corp, got %s", vendors[0].Name)
	}
	if vendors[0].Status != "active" {
		t.Errorf("Expected status active, got %s", vendors[0].Status)
	}
	if vendors[0].LeadTimeDays != 7 {
		t.Errorf("Expected lead time 7, got %d", vendors[0].LeadTimeDays)
	}
}

func TestHandleListVendors_Empty(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	req := httptest.NewRequest("GET", "/api/v1/vendors", nil)
	w := httptest.NewRecorder()

	handleListVendors(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	vendorsJSON, _ := json.Marshal(resp.Data)
	var vendors []Vendor
	if err := json.Unmarshal(vendorsJSON, &vendors); err != nil {
		t.Fatalf("Failed to unmarshal vendors: %v", err)
	}

	if len(vendors) != 0 {
		t.Errorf("Expected 0 vendors, got %d", len(vendors))
	}
}

func TestHandleGetVendor(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	createTestVendor(t, db, "V-001", "Acme Corp", "https://acme.com", "John Doe", "john@acme.com", "555-1234", "Test notes", "active", 7)

	req := httptest.NewRequest("GET", "/api/v1/vendors/V-001", nil)
	w := httptest.NewRecorder()

	handleGetVendor(w, req, "V-001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	vendorJSON, _ := json.Marshal(resp.Data)
	var vendor Vendor
	if err := json.Unmarshal(vendorJSON, &vendor); err != nil {
		t.Fatalf("Failed to unmarshal vendor: %v", err)
	}

	if vendor.ID != "V-001" {
		t.Errorf("Expected ID V-001, got %s", vendor.ID)
	}
	if vendor.Name != "Acme Corp" {
		t.Errorf("Expected name Acme Corp, got %s", vendor.Name)
	}
	if vendor.Website != "https://acme.com" {
		t.Errorf("Expected website https://acme.com, got %s", vendor.Website)
	}
	if vendor.ContactEmail != "john@acme.com" {
		t.Errorf("Expected email john@acme.com, got %s", vendor.ContactEmail)
	}
}

func TestHandleGetVendor_NotFound(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	req := httptest.NewRequest("GET", "/api/v1/vendors/NONEXISTENT", nil)
	w := httptest.NewRecorder()

	handleGetVendor(w, req, "NONEXISTENT")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateVendor(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	vendor := Vendor{
		Name:          "Acme Corp",
		Website:       "https://acme.com",
		ContactName:   "John Doe",
		ContactEmail:  "john@acme.com",
		ContactPhone:  "555-1234",
		Notes:         "Test vendor",
		Status:        "active",
		LeadTimeDays:  7,
	}

	body, _ := json.Marshal(vendor)
	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	vendorJSON, _ := json.Marshal(resp.Data)
	var created Vendor
	if err := json.Unmarshal(vendorJSON, &created); err != nil {
		t.Fatalf("Failed to unmarshal vendor: %v", err)
	}

	if created.ID != "V-001" {
		t.Errorf("Expected ID V-001, got %s", created.ID)
	}
	if created.Name != "Acme Corp" {
		t.Errorf("Expected name Acme Corp, got %s", created.Name)
	}

	// Verify vendor was created in DB
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id = ?", "V-001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 vendor in DB, got %d", count)
	}
}

func TestHandleCreateVendor_DefaultStatus(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	vendor := Vendor{
		Name: "Acme Corp",
		// Status not provided - should default to "active"
	}

	body, _ := json.Marshal(vendor)
	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	vendorJSON, _ := json.Marshal(resp.Data)
	var created Vendor
	if err := json.Unmarshal(vendorJSON, &created); err != nil {
		t.Fatalf("Failed to unmarshal vendor: %v", err)
	}

	if created.Status != "active" {
		t.Errorf("Expected default status 'active', got %s", created.Status)
	}
}

func TestHandleCreateVendor_AutoIncrementID(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	// Create V-001
	vendor1 := Vendor{Name: "Vendor 1"}
	body1, _ := json.Marshal(vendor1)
	req1 := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handleCreateVendor(w1, req1)

	// Create V-002
	vendor2 := Vendor{Name: "Vendor 2"}
	body2, _ := json.Marshal(vendor2)
	req2 := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handleCreateVendor(w2, req2)

	var resp APIResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	vendorJSON, _ := json.Marshal(resp.Data)
	var created Vendor
	json.Unmarshal(vendorJSON, &created)

	if created.ID != "V-002" {
		t.Errorf("Expected ID V-002, got %s", created.ID)
	}
}

func TestHandleCreateVendor_MissingName(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	vendor := Vendor{
		// Missing Name
		Website: "https://test.com",
	}

	body, _ := json.Marshal(vendor)
	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateVendor_InvalidEmail(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	vendor := Vendor{
		Name:         "Acme Corp",
		ContactEmail: "not-an-email",
	}

	body, _ := json.Marshal(vendor)
	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Verify error message mentions email
	if !strings.Contains(w.Body.String(), "email") && !strings.Contains(w.Body.String(), "contact_email") {
		t.Errorf("Expected error message to mention email, got: %s", w.Body.String())
	}
}

func TestHandleCreateVendor_NegativeLeadTime(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	vendor := Vendor{
		Name:         "Acme Corp",
		LeadTimeDays: -5,
	}

	body, _ := json.Marshal(vendor)
	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateVendor_InvalidJSON(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	req := httptest.NewRequest("POST", "/api/v1/vendors", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleUpdateVendor(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	createTestVendor(t, db, "V-001", "Acme Corp", "https://acme.com", "John Doe", "john@acme.com", "555-1234", "Old notes", "active", 7)

	updated := Vendor{
		Name:          "Updated Corp",
		Website:       "https://updated.com",
		ContactName:   "Jane Smith",
		ContactEmail:  "jane@updated.com",
		ContactPhone:  "555-9999",
		Notes:         "New notes",
		Status:        "inactive",
		LeadTimeDays:  14,
	}

	body, _ := json.Marshal(updated)
	req := httptest.NewRequest("PUT", "/api/v1/vendors/V-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateVendor(w, req, "V-001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	vendorJSON, _ := json.Marshal(resp.Data)
	var vendor Vendor
	if err := json.Unmarshal(vendorJSON, &vendor); err != nil {
		t.Fatalf("Failed to unmarshal vendor: %v", err)
	}

	if vendor.Name != "Updated Corp" {
		t.Errorf("Expected name Updated Corp, got %s", vendor.Name)
	}
	if vendor.Status != "inactive" {
		t.Errorf("Expected status inactive, got %s", vendor.Status)
	}
	if vendor.LeadTimeDays != 14 {
		t.Errorf("Expected lead time 14, got %d", vendor.LeadTimeDays)
	}
}

func TestHandleDeleteVendor(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	createTestVendor(t, db, "V-001", "Acme Corp", "https://acme.com", "John Doe", "john@acme.com", "555-1234", "Test notes", "active", 7)

	req := httptest.NewRequest("DELETE", "/api/v1/vendors/V-001", nil)
	w := httptest.NewRecorder()

	handleDeleteVendor(w, req, "V-001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	if result["deleted"] != "V-001" {
		t.Errorf("Expected deleted V-001, got %v", result["deleted"])
	}

	// Verify vendor was deleted from DB
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id = ?", "V-001").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 vendors in DB, got %d", count)
	}
}

func TestHandleDeleteVendor_WithPurchaseOrders(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	createTestVendor(t, db, "V-001", "Acme Corp", "https://acme.com", "John Doe", "john@acme.com", "555-1234", "Test notes", "active", 7)
	
	// Create a purchase order referencing this vendor
	db.Exec("INSERT INTO purchase_orders (id, vendor_id, status) VALUES (?, ?, ?)", "PO-001", "V-001", "draft")

	req := httptest.NewRequest("DELETE", "/api/v1/vendors/V-001", nil)
	w := httptest.NewRecorder()

	handleDeleteVendor(w, req, "V-001")

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	// Verify error message mentions purchase orders
	if !strings.Contains(w.Body.String(), "purchase orders") {
		t.Errorf("Expected error message to mention purchase orders, got: %s", w.Body.String())
	}

	// Verify vendor was NOT deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id = ?", "V-001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected vendor to still exist, got %d vendors", count)
	}
}

func TestHandleDeleteVendor_WithRFQs(t *testing.T) {
	origDB := db
	db = setupVendorsTestDB(t)
	defer func() { db.Close(); db = origDB }()

	createTestVendor(t, db, "V-001", "Acme Corp", "https://acme.com", "John Doe", "john@acme.com", "555-1234", "Test notes", "active", 7)
	
	// Create an RFQ vendor entry referencing this vendor
	db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id) VALUES (?, ?)", "RFQ-001", "V-001")

	req := httptest.NewRequest("DELETE", "/api/v1/vendors/V-001", nil)
	w := httptest.NewRecorder()

	handleDeleteVendor(w, req, "V-001")

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	// Verify error message mentions RFQs
	if !strings.Contains(w.Body.String(), "RFQ") {
		t.Errorf("Expected error message to mention RFQs, got: %s", w.Body.String())
	}

	// Verify vendor was NOT deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id = ?", "V-001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected vendor to still exist, got %d vendors", count)
	}
}
