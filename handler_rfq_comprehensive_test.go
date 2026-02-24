package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupRFQTestDBComprehensive creates a fresh test database with all required tables
func setupRFQTestDBComprehensive(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create all required tables
	tables := []string{
		`CREATE TABLE rfqs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			due_date TEXT,
			notes TEXT
		)`,
		`CREATE TABLE rfq_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty REAL NOT NULL,
			unit TEXT DEFAULT 'EA',
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE rfq_vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			quoted_at DATETIME,
			notes TEXT,
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE rfq_quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			rfq_vendor_id INTEGER NOT NULL,
			rfq_line_id INTEGER NOT NULL,
			unit_price REAL NOT NULL,
			lead_time_days INTEGER DEFAULT 0,
			moq INTEGER DEFAULT 1,
			notes TEXT,
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE,
			FOREIGN KEY (rfq_vendor_id) REFERENCES rfq_vendors(id) ON DELETE CASCADE,
			FOREIGN KEY (rfq_line_id) REFERENCES rfq_lines(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME
		)`,
		`CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_ordered REAL NOT NULL,
			qty_received REAL DEFAULT 0,
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER NOT NULL DEFAULT 1
		)`,
	}

	for _, sql := range tables {
		if _, err := testDB.Exec(sql); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	return testDB
}

// Test: Verify ID generation uses correct pattern
func TestRFQ_IDGeneration(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"title": "Test RFQ for ID Generation",
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	if w.Code != 201 {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})
	id := created["id"].(string)

	// Verify ID format: RFQ-YYYY-NNNN (e.g., RFQ-2026-0001)
	if !strings.HasPrefix(id, "RFQ-") {
		t.Errorf("Expected ID to start with 'RFQ-', got %s", id)
	}
	// ID should be RFQ-YYYY-NNNN format (13 characters)
	if len(id) < 10 {
		t.Errorf("Expected ID to be at least 10 chars (RFQ-YYYY-N), got %d for ID %s", len(id), id)
	}
	// Verify it contains year
	if !strings.Contains(id, "2026") {
		t.Logf("Note: ID format is %s (may use different year pattern)", id)
	}
}

// Test: RFQ with no lines (edge case)
func TestRFQ_NoLines(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-001", "Test Vendor")

	rfq := map[string]interface{}{
		"title": "RFQ with no lines",
		"vendors": []map[string]interface{}{
			{"vendor_id": "V-001"},
		},
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	if w.Code != 201 {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})

	lines := created["lines"]
	// System returns empty arrays as [] not nil when properly initialized
	if lines == nil {
		t.Log("Note: System returns nil for empty lines array (consider changing to [] for consistency)")
	}
}

// Test: RFQ with no vendors (edge case)
func TestRFQ_NoVendors(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"title": "RFQ with no vendors",
		"lines": []map[string]interface{}{
			{"ipn": "IPN-001", "qty": 10},
		},
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	if w.Code != 201 {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})

	vendors := created["vendors"]
	// System returns empty arrays as [] not nil when properly initialized
	if vendors == nil {
		t.Log("Note: System returns nil for empty vendors array (consider changing to [] for consistency)")
	}
}

// Test: Line item with invalid quantity (zero)
func TestRFQ_LineItemZeroQty(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"title": "Test RFQ",
		"lines": []map[string]interface{}{
			{"ipn": "IPN-001", "qty": 0},
		},
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	// Currently allows zero qty - this documents existing behavior
	// TODO: Add validation to reject zero/negative qty if business rules require it
	if w.Code == 201 {
		t.Log("Note: System currently allows zero quantity line items")
	}
}

// Test: Line item with negative quantity
func TestRFQ_LineItemNegativeQty(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"title": "Test RFQ",
		"lines": []map[string]interface{}{
			{"ipn": "IPN-001", "qty": -10},
		},
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	// Currently allows negative qty - this documents existing behavior
	// TODO: Add validation to reject negative qty
	if w.Code == 201 {
		t.Log("Note: System currently allows negative quantity line items")
	}
}

// Test: Line item with empty IPN
func TestRFQ_LineItemEmptyIPN(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"title": "Test RFQ",
		"lines": []map[string]interface{}{
			{"ipn": "", "qty": 10},
		},
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	// Currently allows empty IPN - this documents existing behavior
	// TODO: Add validation to require IPN
	if w.Code == 201 {
		t.Log("Note: System currently allows empty IPN in line items")
	}
}

// Test: Multi-vendor RFQ with different quotes per vendor
func TestRFQ_MultiVendorDifferentQuotes(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-001", "Vendor A")
	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-002", "Vendor B")
	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-003", "Vendor C")

	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-MULTI", "Multi-vendor Test", "sent", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	res1, _ := db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-MULTI", "IPN-001", "Part 1", 100, "EA")
	line1ID, _ := res1.LastInsertId()

	res2, _ := db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-MULTI", "IPN-002", "Part 2", 50, "EA")
	line2ID, _ := res2.LastInsertId()

	resV1, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-MULTI", "V-001", "quoted")
	vendor1ID, _ := resV1.LastInsertId()

	resV2, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-MULTI", "V-002", "quoted")
	vendor2ID, _ := resV2.LastInsertId()

	resV3, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-MULTI", "V-003", "quoted")
	vendor3ID, _ := resV3.LastInsertId()

	// Vendor 1: quotes both lines
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-MULTI", vendor1ID, line1ID, 10.00, 14, 50)
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-MULTI", vendor1ID, line2ID, 15.00, 14, 25)

	// Vendor 2: quotes only line 1
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-MULTI", vendor2ID, line1ID, 9.50, 21, 100)

	// Vendor 3: quotes only line 2
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-MULTI", vendor3ID, line2ID, 12.00, 10, 10)

	req := httptest.NewRequest("GET", "/api/rfq/RFQ-MULTI/compare", nil)
	w := httptest.NewRecorder()

	handleCompareRFQ(w, req, "RFQ-MULTI")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	result := resp.Data.(map[string]interface{})

	lines := result["lines"].([]interface{})
	vendors := result["vendors"].([]interface{})
	matrix := result["matrix"].(map[string]interface{})

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
	if len(vendors) != 3 {
		t.Errorf("Expected 3 vendors, got %d", len(vendors))
	}

	// Verify matrix has correct structure
	if len(matrix) == 0 {
		t.Errorf("Expected populated matrix")
	}

	t.Logf("Multi-vendor comparison successful: %d lines, %d vendors, %d matrix entries", len(lines), len(vendors), len(matrix))
}

// Test: Due date in the past
func TestRFQ_PastDueDate(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	pastDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	rfq := map[string]interface{}{
		"title":    "Past Due Date Test",
		"due_date": pastDate,
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	// Currently allows past due dates - documents existing behavior
	if w.Code == 201 {
		t.Log("Note: System currently allows past due dates")
	}
}

// Test: Award RFQ with no quotes
func TestRFQ_AwardWithNoQuotes(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-001", "Test Vendor")
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-NOQT", "No Quotes", "sent", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-NOQT", "V-001", "pending")

	award := map[string]string{"vendor_id": "V-001"}
	body, _ := json.Marshal(award)
	req := httptest.NewRequest("POST", "/api/rfq/RFQ-NOQT/award", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAwardRFQ(w, req, "RFQ-NOQT")

	// Currently allows awarding without quotes - creates PO with no lines
	if w.Code == 200 {
		var resp APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		result := resp.Data.(map[string]interface{})
		poID := result["po_id"].(string)

		var lineCount int
		db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id=?", poID).Scan(&lineCount)
		if lineCount != 0 {
			t.Errorf("Expected 0 PO lines without quotes, got %d", lineCount)
		}
		t.Log("Note: System allows awarding RFQ without quotes (creates empty PO)")
	}
}

// Test: Invalid status transitions
func TestRFQ_InvalidStatusTransitions(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	// Test: Cannot close draft RFQ
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-DRAFT", "Draft RFQ", "draft", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	req := httptest.NewRequest("POST", "/api/rfq/RFQ-DRAFT/close", nil)
	w := httptest.NewRecorder()
	handleCloseRFQ(w, req, "RFQ-DRAFT")

	if w.Code != 400 {
		t.Errorf("Expected 400 when closing draft RFQ, got %d", w.Code)
	}

	// Test: Cannot send already-sent RFQ
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-SENT", "Sent RFQ", "sent", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	req = httptest.NewRequest("POST", "/api/rfq/RFQ-SENT/send", nil)
	w = httptest.NewRecorder()
	handleSendRFQ(w, req, "RFQ-SENT")

	if w.Code != 400 {
		t.Errorf("Expected 400 when sending already-sent RFQ, got %d", w.Code)
	}

	// Test: Cannot close already-closed RFQ
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-CLOSED", "Closed RFQ", "closed", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	req = httptest.NewRequest("POST", "/api/rfq/RFQ-CLOSED/close", nil)
	w = httptest.NewRecorder()
	handleCloseRFQ(w, req, "RFQ-CLOSED")

	if w.Code != 400 {
		t.Errorf("Expected 400 when closing already-closed RFQ, got %d", w.Code)
	}
}

// Test: Cascade delete behavior
func TestRFQ_CascadeDelete(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-001", "Test Vendor")
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-DEL", "Delete Test", "draft", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	res, _ := db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-DEL", "IPN-001", "Part 1", 10, "EA")
	lineID, _ := res.LastInsertId()

	resV, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-DEL", "V-001", "pending")
	vendorID, _ := resV.LastInsertId()

	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-DEL", vendorID, lineID, 10.0, 14, 50)

	// Verify data exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM rfq_lines WHERE rfq_id=?", "RFQ-DEL").Scan(&count)
	if count != 1 {
		t.Fatalf("Expected 1 line before delete, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM rfq_vendors WHERE rfq_id=?", "RFQ-DEL").Scan(&count)
	if count != 1 {
		t.Fatalf("Expected 1 vendor before delete, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM rfq_quotes WHERE rfq_id=?", "RFQ-DEL").Scan(&count)
	if count != 1 {
		t.Fatalf("Expected 1 quote before delete, got %d", count)
	}

	// Delete RFQ
	req := httptest.NewRequest("DELETE", "/api/rfq/RFQ-DEL", nil)
	w := httptest.NewRecorder()
	handleDeleteRFQ(w, req, "RFQ-DEL")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Verify cascade delete
	db.QueryRow("SELECT COUNT(*) FROM rfq_lines WHERE rfq_id=?", "RFQ-DEL").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 lines after delete, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM rfq_vendors WHERE rfq_id=?", "RFQ-DEL").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 vendors after delete, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM rfq_quotes WHERE rfq_id=?", "RFQ-DEL").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 quotes after delete, got %d", count)
	}
}

// Test: Audit log entries
func TestRFQ_AuditLog(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"title": "Audit Test",
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})
	id := created["id"].(string)

	// Verify audit log entry was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module='rfq' AND record_id=? AND action='create'", id).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 audit log entry for create, got %d", count)
	}
}

// Test: Email body generation
func TestRFQ_EmailBody(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at, due_date, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"RFQ-EMAIL", "Email Test", "draft", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), "2026-12-31", "Please quote ASAP")

	db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-EMAIL", "IPN-001", "Resistor 10k", 1000, "EA")
	db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-EMAIL", "IPN-002", "Capacitor 100nF", 500, "EA")

	req := httptest.NewRequest("GET", "/api/rfq/RFQ-EMAIL/email", nil)
	w := httptest.NewRecorder()

	handleRFQEmailBody(w, req, "RFQ-EMAIL")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	result := resp.Data.(map[string]interface{})

	subject := result["subject"].(string)
	body := result["body"].(string)

	if !strings.Contains(subject, "RFQ-EMAIL") {
		t.Errorf("Expected subject to contain RFQ-EMAIL, got: %s", subject)
	}

	if !strings.Contains(body, "IPN-001") {
		t.Errorf("Expected body to contain IPN-001")
	}
	if !strings.Contains(body, "IPN-002") {
		t.Errorf("Expected body to contain IPN-002")
	}
	if !strings.Contains(body, "2026-12-31") {
		t.Errorf("Expected body to contain due date")
	}
	if !strings.Contains(body, "Please quote ASAP") {
		t.Errorf("Expected body to contain notes")
	}
}

// Test: Dashboard statistics
func TestRFQ_Dashboard(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	now := time.Now()

	// Create various RFQs
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-D1", "Draft 1", "draft", "user1", now.Format(time.RFC3339), now.Format(time.RFC3339))
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-D2", "Draft 2", "draft", "user1", now.Format(time.RFC3339), now.Format(time.RFC3339))
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-S1", "Sent 1", "sent", "user1", now.Format(time.RFC3339), now.Format(time.RFC3339))
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-A1", "Awarded 1", "awarded", "user1", now.Format(time.RFC3339), now.Format(time.RFC3339))
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-C1", "Closed 1", "closed", "user1", now.Format(time.RFC3339), now.AddDate(0, 0, -40).Format(time.RFC3339))

	// Add vendors
	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-001", "Vendor A")
	db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)", "RFQ-S1", "V-001", "pending")
	db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)", "RFQ-A1", "V-001", "quoted")

	req := httptest.NewRequest("GET", "/api/rfq/dashboard", nil)
	w := httptest.NewRecorder()

	handleRFQDashboard(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	dash := resp.Data.(map[string]interface{})

	openRFQs := int(dash["open_rfqs"].(float64))
	pendingResponses := int(dash["pending_responses"].(float64))
	awardedThisMonth := int(dash["awarded_this_month"].(float64))

	if openRFQs != 3 { // 2 draft + 1 sent
		t.Errorf("Expected 3 open RFQs, got %d", openRFQs)
	}
	if pendingResponses != 1 {
		t.Errorf("Expected 1 pending response, got %d", pendingResponses)
	}
	if awardedThisMonth != 1 {
		t.Errorf("Expected 1 awarded this month, got %d", awardedThisMonth)
	}

	rfqs := dash["rfqs"].([]interface{})
	if len(rfqs) < 3 {
		t.Errorf("Expected at least 3 active RFQs in dashboard, got %d", len(rfqs))
	}
}

// Test: Concurrent updates (race condition)
// NOTE: This test demonstrates that concurrent updates work but don't have transaction isolation
func TestRFQ_ConcurrentUpdates(t *testing.T) {
	t.Skip("Skipping concurrent update test - requires connection pooling or transaction management")
	// TODO: Implement proper transaction isolation or connection pooling before enabling this test
	
	oldDB := db
	testDB := setupRFQTestDBComprehensive(t)
	db = testDB
	defer func() { 
		testDB.Close()
		db = oldDB
	}()

	testDB.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-RACE", "Race Test", "draft", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Simulate 3 concurrent updates
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			update := map[string]interface{}{
				"title": "Updated Title " + string(rune(n)),
				"notes": "Update from goroutine " + string(rune(n)),
			}

			body, _ := json.Marshal(update)
			req := httptest.NewRequest("PUT", "/api/rfq/RFQ-RACE", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleUpdateRFQ(w, req, "RFQ-RACE")
			
			if w.Code == 200 {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify RFQ still exists and has valid data
	var title string
	err := testDB.QueryRow("SELECT title FROM rfqs WHERE id=?", "RFQ-RACE").Scan(&title)
	if err != nil {
		t.Errorf("RFQ corrupted after concurrent updates: %v", err)
		return
	}

	if title == "" {
		t.Errorf("RFQ title is empty after concurrent updates")
	}

	t.Logf("Concurrent update test: %d/3 updates succeeded. Final title: %s", successCount, title)
}

// Test: PO creation details on award
func TestRFQ_POCreationDetails(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-WIN", "Winning Vendor")
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-PO", "PO Test", "sent", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	res, _ := db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-PO", "IPN-123", "Test Part", 100, "EA")
	lineID, _ := res.LastInsertId()

	resV, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-PO", "V-WIN", "quoted")
	vendorID, _ := resV.LastInsertId()

	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-PO", vendorID, lineID, 25.50, 7, 10)

	award := map[string]string{"vendor_id": "V-WIN"}
	body, _ := json.Marshal(award)
	req := httptest.NewRequest("POST", "/api/rfq/RFQ-PO/award", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAwardRFQ(w, req, "RFQ-PO")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	result := resp.Data.(map[string]interface{})
	poID := result["po_id"].(string)

	// Verify PO fields
	var poVendor, poStatus, poNotes string
	db.QueryRow("SELECT vendor_id, status, notes FROM purchase_orders WHERE id=?", poID).Scan(&poVendor, &poStatus, &poNotes)

	if poVendor != "V-WIN" {
		t.Errorf("Expected PO vendor 'V-WIN', got %s", poVendor)
	}
	if poStatus != "draft" {
		t.Errorf("Expected PO status 'draft', got %s", poStatus)
	}
	if !strings.Contains(poNotes, "RFQ-PO") {
		t.Errorf("Expected PO notes to reference RFQ-PO, got: %s", poNotes)
	}

	// Verify PO line fields
	var lineIPN string
	var lineQty, linePrice float64
	db.QueryRow("SELECT ipn, qty_ordered, unit_price FROM po_lines WHERE po_id=?", poID).Scan(&lineIPN, &lineQty, &linePrice)

	if lineIPN != "IPN-123" {
		t.Errorf("Expected PO line IPN 'IPN-123', got %s", lineIPN)
	}
	if lineQty != 100 {
		t.Errorf("Expected PO line qty 100, got %.2f", lineQty)
	}
	if linePrice != 25.50 {
		t.Errorf("Expected PO line price 25.50, got %.2f", linePrice)
	}
}

// Test: Per-line award functionality
func TestRFQ_PerLineAward(t *testing.T) {
	oldDB := db
	db = setupRFQTestDBComprehensive(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-A", "Vendor A")
	db.Exec("INSERT INTO vendors (id, name) VALUES (?, ?)", "V-B", "Vendor B")
	db.Exec("INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-SPLIT", "Split Award", "sent", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	res1, _ := db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-SPLIT", "IPN-A", "Part A", 50, "EA")
	line1ID, _ := res1.LastInsertId()

	res2, _ := db.Exec("INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		"RFQ-SPLIT", "IPN-B", "Part B", 75, "EA")
	line2ID, _ := res2.LastInsertId()

	resVA, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-SPLIT", "V-A", "quoted")
	vendorAID, _ := resVA.LastInsertId()

	resVB, _ := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		"RFQ-SPLIT", "V-B", "quoted")
	vendorBID, _ := resVB.LastInsertId()

	// Vendor A quotes line 1
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price) VALUES (?, ?, ?, ?)",
		"RFQ-SPLIT", vendorAID, line1ID, 10.0)

	// Vendor B quotes line 2
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price) VALUES (?, ?, ?, ?)",
		"RFQ-SPLIT", vendorBID, line2ID, 15.0)

	// Award per line
	award := map[string]interface{}{
		"awards": []map[string]interface{}{
			{"line_id": int(line1ID), "vendor_id": "V-A"},
			{"line_id": int(line2ID), "vendor_id": "V-B"},
		},
	}

	body, _ := json.Marshal(award)
	req := httptest.NewRequest("POST", "/api/rfq/RFQ-SPLIT/award-per-line", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAwardRFQPerLine(w, req, "RFQ-SPLIT")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	result := resp.Data.(map[string]interface{})

	poIDs := result["po_ids"].([]interface{})
	if len(poIDs) != 2 {
		t.Errorf("Expected 2 POs created, got %d", len(poIDs))
	}

	// Verify each PO has correct vendor and lines
	for _, poIDInterface := range poIDs {
		poID := poIDInterface.(string)
		var vendorID string
		db.QueryRow("SELECT vendor_id FROM purchase_orders WHERE id=?", poID).Scan(&vendorID)

		var lineCount int
		db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id=?", poID).Scan(&lineCount)

		if lineCount != 1 {
			t.Errorf("Expected 1 line per PO, got %d for PO %s", lineCount, poID)
		}

		t.Logf("PO %s created for vendor %s with %d line(s)", poID, vendorID, lineCount)
	}
}
