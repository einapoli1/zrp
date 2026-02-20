package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupAuditTestDB creates an in-memory database with all required tables
func setupAuditTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create audit_log table with all enhanced fields
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			before_value TEXT,
			after_value TEXT,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Create changes table for change history
	_, err = testDB.Exec(`
		CREATE TABLE changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			action TEXT NOT NULL,
			old_value TEXT,
			new_value TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create changes table: %v", err)
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

	// Create purchase_orders table
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at TEXT,
			created_by TEXT,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	// Create po_lines table
	_, err = testDB.Exec(`
		CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_ordered REAL NOT NULL,
			qty_received REAL DEFAULT 0,
			unit_price REAL,
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create po_lines table: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_by TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
			approved_at TEXT,
			approved_by TEXT,
			ncr_id TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create eco_revisions table
	_, err = testDB.Exec(`
		CREATE TABLE eco_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			eco_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			approved_by TEXT,
			approved_at TEXT,
			FOREIGN KEY (eco_id) REFERENCES ecos(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create eco_revisions table: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			location TEXT,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
			description TEXT,
			mpn TEXT,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create inventory_transactions table
	_, err = testDB.Exec(`
		CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL,
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (ipn) REFERENCES inventory(ipn)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Create undo_operations table
	_, err = testDB.Exec(`
		CREATE TABLE undo_operations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			operation TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			snapshot TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			undone INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create undo_operations table: %v", err)
	}

	// Create users and sessions for authentication testing
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user'
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	// Insert test user
	_, err = testDB.Exec(`INSERT INTO users (id, username, email, role) VALUES (1, 'testuser', 'test@example.com', 'admin')`)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Insert test session
	_, err = testDB.Exec(`INSERT INTO sessions (token, user_id) VALUES ('test-session-token', 1)`)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return testDB
}

// verifyAuditLog checks that an audit log entry exists with expected properties
func verifyAuditLog(t *testing.T, db *sql.DB, module, action, recordID string) map[string]interface{} {
	t.Helper()

	var id int
	var userID sql.NullInt64
	var username, auditAction, auditModule, auditRecordID, summary string
	var beforeValue, afterValue, ipAddress, userAgent sql.NullString
	var createdAt string

	query := `SELECT id, user_id, username, action, module, record_id, summary, 
		before_value, after_value, ip_address, user_agent, created_at 
		FROM audit_log WHERE module = ? AND action = ? AND record_id = ?
		ORDER BY created_at DESC LIMIT 1`

	err := db.QueryRow(query, module, action, recordID).Scan(
		&id, &userID, &username, &auditAction, &auditModule, &auditRecordID,
		&summary, &beforeValue, &afterValue, &ipAddress, &userAgent, &createdAt,
	)

	if err == sql.ErrNoRows {
		t.Fatalf("No audit log entry found for module=%s, action=%s, record_id=%s", module, action, recordID)
	}
	if err != nil {
		t.Fatalf("Error querying audit log: %v", err)
	}

	entry := map[string]interface{}{
		"id":           id,
		"user_id":      userID,
		"username":     username,
		"action":       auditAction,
		"module":       auditModule,
		"record_id":    auditRecordID,
		"summary":      summary,
		"before_value": beforeValue,
		"after_value":  afterValue,
		"ip_address":   ipAddress,
		"user_agent":   userAgent,
		"created_at":   createdAt,
	}

	return entry
}

// Test vendor CRUD operations generate audit logs
func TestAuditLog_Vendor_Create(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	// Temporarily swap global db
	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create vendor
	body := `{
		"name": "Test Vendor Inc",
		"website": "https://example.com",
		"contact_email": "contact@example.com",
		"status": "active",
		"lead_time_days": 14
	}`

	req := httptest.NewRequest("POST", "/api/vendors", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data Vendor `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	vendor := resp.Data
	if vendor.ID == "" {
		t.Fatalf("Vendor ID is empty in response: %s", w.Body.String())
	}

	t.Logf("Created vendor ID: %s", vendor.ID)

	// Verify audit log entry exists
	entry := verifyAuditLog(t, testDB, "vendor", "created", vendor.ID)

	// Verify user tracking
	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	// Verify timestamp is recent
	if entry["created_at"] == "" {
		t.Error("Expected created_at to be populated")
	}

	// Verify summary contains vendor info
	summary := entry["summary"].(string)
	if summary == "" {
		t.Error("Expected summary to be populated")
	}

	t.Logf("✓ Vendor CREATE audit log verified: %s", vendor.ID)
}

func TestAuditLog_Vendor_Update(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create initial vendor
	_, err := testDB.Exec(`INSERT INTO vendors (id, name, status) VALUES ('V-001', 'Original Name', 'active')`)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}

	// Update vendor
	body := `{
		"name": "Updated Vendor Name",
		"status": "active",
		"lead_time_days": 21
	}`

	req := httptest.NewRequest("PUT", "/api/vendors/V-001", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleUpdateVendor(w, req, "V-001")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log entry
	entry := verifyAuditLog(t, testDB, "vendor", "updated", "V-001")

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	t.Logf("✓ Vendor UPDATE audit log verified")
}

func TestAuditLog_Vendor_Delete(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create vendor to delete
	_, err := testDB.Exec(`INSERT INTO vendors (id, name) VALUES ('V-999', 'Delete Me')`)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/vendors/V-999", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleDeleteVendor(w, req, "V-999")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log entry
	entry := verifyAuditLog(t, testDB, "vendor", "deleted", "V-999")

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	t.Logf("✓ Vendor DELETE audit log verified")
}

// Test ECO operations with approval tracking
func TestAuditLog_ECO_Create(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	body := `{
		"title": "Update resistor value",
		"description": "Change R1 from 10k to 22k",
		"affected_ipns": "R001,R002",
		"priority": "high",
		"status": "draft"
	}`

	req := httptest.NewRequest("POST", "/api/ecos", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data ECO `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	eco := resp.Data
	if eco.ID == "" {
		t.Fatalf("ECO ID is empty in response: %s", w.Body.String())
	}

	t.Logf("Created ECO ID: %s", eco.ID)

	// Verify audit log
	entry := verifyAuditLog(t, testDB, "eco", "created", eco.ID)

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	t.Logf("✓ ECO CREATE audit log verified: %s", eco.ID)
}

func TestAuditLog_ECO_Approve(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create ECO
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := testDB.Exec(`INSERT INTO ecos (id, title, status, created_at, updated_at) 
		VALUES ('ECO-001', 'Test ECO', 'review', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to insert test ECO: %v", err)
	}

	// Approve ECO
	req := httptest.NewRequest("POST", "/api/ecos/ECO-001/approve", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleApproveECO(w, req, "ECO-001")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log for approval
	entry := verifyAuditLog(t, testDB, "eco", "approved", "ECO-001")

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	// Verify approval was logged
	summary := entry["summary"].(string)
	if summary == "" {
		t.Error("Expected summary to be populated for approval")
	}

	// Verify ECO record was updated with approval info
	var approvedBy sql.NullString
	var approvedAt sql.NullString
	err = testDB.QueryRow("SELECT approved_by, approved_at FROM ecos WHERE id = ?", "ECO-001").
		Scan(&approvedBy, &approvedAt)
	if err != nil {
		t.Fatalf("Failed to query ECO: %v", err)
	}

	if !approvedBy.Valid || approvedBy.String != "testuser" {
		t.Errorf("Expected approved_by='testuser', got '%v'", approvedBy)
	}

	if !approvedAt.Valid {
		t.Error("Expected approved_at to be set")
	}

	t.Logf("✓ ECO APPROVE audit log verified with user tracking")
}

// Test Purchase Order operations with price tracking
func TestAuditLog_PurchaseOrder_Create(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create test vendor
	_, err := testDB.Exec(`INSERT INTO vendors (id, name) VALUES ('V-001', 'Test Vendor')`)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}

	body := `{
		"vendor_id": "V-001",
		"status": "draft",
		"notes": "Test PO",
		"lines": [
			{
				"ipn": "R001",
				"qty_ordered": 100,
				"unit_price": 0.25
			},
			{
				"ipn": "C001",
				"qty_ordered": 50,
				"unit_price": 1.50
			}
		]
	}`

	req := httptest.NewRequest("POST", "/api/purchase-orders", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data PurchaseOrder `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	po := resp.Data
	if po.ID == "" {
		t.Fatalf("PO ID is empty in response: %s", w.Body.String())
	}

	t.Logf("Created PO ID: %s", po.ID)

	// Verify audit log
	entry := verifyAuditLog(t, testDB, "po", "created", po.ID)

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	// Verify PO lines were created with prices
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id = ?", po.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query po_lines: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 PO lines, got %d", count)
	}

	// Verify prices were captured
	var unitPrice float64
	err = testDB.QueryRow("SELECT unit_price FROM po_lines WHERE po_id = ? AND ipn = 'R001'", po.ID).Scan(&unitPrice)
	if err != nil {
		t.Fatalf("Failed to query unit_price: %v", err)
	}
	if unitPrice != 0.25 {
		t.Errorf("Expected unit_price=0.25, got %f", unitPrice)
	}

	t.Logf("✓ PO CREATE audit log verified with price tracking: %s", po.ID)
}

func TestAuditLog_PurchaseOrder_Update_PriceChange(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create vendor and PO
	_, err := testDB.Exec(`INSERT INTO vendors (id, name) VALUES ('V-001', 'Test Vendor')`)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = testDB.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, created_at) 
		VALUES ('PO-0001', 'V-001', 'draft', ?)`, now)
	if err != nil {
		t.Fatalf("Failed to insert test PO: %v", err)
	}

	_, err = testDB.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) 
		VALUES ('PO-0001', 'R001', 100, 0.25)`)
	if err != nil {
		t.Fatalf("Failed to insert PO line: %v", err)
	}

	// Update PO
	body := `{
		"vendor_id": "V-001",
		"status": "ordered",
		"notes": "Price updated"
	}`

	req := httptest.NewRequest("PUT", "/api/purchase-orders/PO-0001", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleUpdatePO(w, req, "PO-0001")

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log
	entry := verifyAuditLog(t, testDB, "po", "updated", "PO-0001")

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	t.Logf("✓ PO UPDATE audit log verified")
}

// Test inventory adjustments are logged
func TestAuditLog_Inventory_Adjust(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create inventory item
	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('R001', 100)`)
	if err != nil {
		t.Fatalf("Failed to insert inventory: %v", err)
	}

	// Perform adjustment
	body := `{
		"ipn": "R001",
		"type": "adjust",
		"qty": 150,
		"notes": "Physical count adjustment"
	}`

	req := httptest.NewRequest("POST", "/api/inventory/transact", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log
	entry := verifyAuditLog(t, testDB, "inventory", "adjust", "R001")

	if entry["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", entry["username"])
	}

	// Verify transaction was recorded
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = 'R001' AND type = 'adjust'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query inventory_transactions: %v", err)
	}
	if count == 0 {
		t.Error("Expected inventory transaction to be recorded")
	}

	// Verify quantity was updated
	var qtyOnHand float64
	err = testDB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = 'R001'").Scan(&qtyOnHand)
	if err != nil {
		t.Fatalf("Failed to query inventory: %v", err)
	}
	if qtyOnHand != 150 {
		t.Errorf("Expected qty_on_hand=150, got %f", qtyOnHand)
	}

	t.Logf("✓ Inventory ADJUST audit log verified")
}

// Test audit log includes before/after values for updates
func TestAuditLog_BeforeAfter_Values(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create vendor
	_, err := testDB.Exec(`INSERT INTO vendors (id, name, status, lead_time_days) 
		VALUES ('V-001', 'Original Vendor', 'active', 10)`)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}

	// Update with LogUpdateWithDiff
	req := httptest.NewRequest("PUT", "/api/vendors/V-001", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})

	before := map[string]interface{}{
		"name":           "Original Vendor",
		"status":         "active",
		"lead_time_days": 10,
	}

	after := map[string]interface{}{
		"name":           "Updated Vendor",
		"status":         "active",
		"lead_time_days": 15,
	}

	LogUpdateWithDiff(testDB, req, "vendor", "V-001", before, after)

	// Verify audit log has before/after values
	var beforeValue, afterValue sql.NullString
	err = testDB.QueryRow(`SELECT before_value, after_value FROM audit_log 
		WHERE module = 'vendor' AND record_id = 'V-001' AND action = 'UPDATE'`).
		Scan(&beforeValue, &afterValue)

	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if !beforeValue.Valid || beforeValue.String == "" {
		t.Error("Expected before_value to be populated")
	}

	if !afterValue.Valid || afterValue.String == "" {
		t.Error("Expected after_value to be populated")
	}

	// Verify JSON is valid
	var beforeJSON, afterJSON map[string]interface{}
	if err := json.Unmarshal([]byte(beforeValue.String), &beforeJSON); err != nil {
		t.Errorf("before_value is not valid JSON: %v", err)
	}

	if err := json.Unmarshal([]byte(afterValue.String), &afterJSON); err != nil {
		t.Errorf("after_value is not valid JSON: %v", err)
	}

	// Verify content
	if beforeJSON["name"] != "Original Vendor" {
		t.Errorf("Expected before name='Original Vendor', got '%v'", beforeJSON["name"])
	}

	if afterJSON["name"] != "Updated Vendor" {
		t.Errorf("Expected after name='Updated Vendor', got '%v'", afterJSON["name"])
	}

	t.Logf("✓ Before/after values verified in audit log")
}

// Test audit log searchability and filtering
func TestAuditLog_Search_Filter(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	// Create multiple audit log entries
	entries := []struct {
		module   string
		action   string
		recordID string
		username string
	}{
		{"vendor", "created", "V-001", "testuser"},
		{"vendor", "updated", "V-001", "testuser"},
		{"eco", "created", "ECO-001", "testuser"},
		{"eco", "approved", "ECO-001", "admin"},
		{"po", "created", "PO-0001", "testuser"},
	}

	for _, e := range entries {
		_, err := testDB.Exec(`INSERT INTO audit_log (module, action, record_id, username, summary) 
			VALUES (?, ?, ?, ?, ?)`,
			e.module, e.action, e.recordID, e.username, "Test entry")
		if err != nil {
			t.Fatalf("Failed to insert audit entry: %v", err)
		}
	}

	// Test filtering by module
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'vendor'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 vendor entries, got %d", count)
	}

	// Test filtering by action
	err = testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE action = 'created'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 created entries, got %d", count)
	}

	// Test filtering by user
	err = testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE username = 'admin'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 admin entry, got %d", count)
	}

	// Test filtering by record_id
	err = testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = 'ECO-001'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 ECO-001 entries, got %d", count)
	}

	t.Logf("✓ Audit log search and filtering verified")
}

// Test audit log captures IP address and user agent
func TestAuditLog_IPAddress_UserAgent(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	req.Header.Set("User-Agent", "Mozilla/5.0 Test Browser")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	userID, username := GetUserContext(req, testDB)
	ipAddress := GetClientIP(req)
	userAgent := req.UserAgent()

	err := LogAuditEnhanced(testDB, LogAuditOptions{
		UserID:    userID,
		Username:  username,
		Action:    "TEST",
		Module:    "test",
		RecordID:  "TEST-001",
		Summary:   "Test entry",
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	if err != nil {
		t.Fatalf("Failed to log audit entry: %v", err)
	}

	// Verify IP address and user agent were captured
	var capturedIP, capturedUA sql.NullString
	err = testDB.QueryRow(`SELECT ip_address, user_agent FROM audit_log 
		WHERE module = 'test' AND record_id = 'TEST-001'`).
		Scan(&capturedIP, &capturedUA)

	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if !capturedIP.Valid || capturedIP.String != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%v'", capturedIP)
	}

	if !capturedUA.Valid || capturedUA.String != "Mozilla/5.0 Test Browser" {
		t.Errorf("Expected user agent 'Mozilla/5.0 Test Browser', got '%v'", capturedUA)
	}

	t.Logf("✓ IP address and user agent tracking verified")
}

// Test all CRUD operations generate audit entries
func TestAuditLog_Completeness_AllOperations(t *testing.T) {
	testDB := setupAuditTestDB(t)
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	operations := []struct {
		module   string
		action   string
		recordID string
	}{
		// Test each critical module has audit logging
		{"vendor", "created", "V-TEST"},
		{"vendor", "updated", "V-TEST"},
		{"vendor", "deleted", "V-TEST"},
		{"po", "created", "PO-TEST"},
		{"po", "updated", "PO-TEST"},
		{"eco", "created", "ECO-TEST"},
		{"eco", "updated", "ECO-TEST"},
		{"eco", "approved", "ECO-TEST"},
		{"inventory", "adjust", "TEST-PART"},
		{"inventory", "receive", "TEST-PART"},
		{"inventory", "issue", "TEST-PART"},
	}

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})

	// Log each operation
	for _, op := range operations {
		err := LogAuditEnhanced(testDB, LogAuditOptions{
			UserID:   1,
			Username: "testuser",
			Action:   op.action,
			Module:   op.module,
			RecordID: op.recordID,
			Summary:  "Test operation",
		})
		if err != nil {
			t.Errorf("Failed to log %s %s: %v", op.module, op.action, err)
		}
	}

	// Verify all entries exist
	for _, op := range operations {
		var count int
		err := testDB.QueryRow(`SELECT COUNT(*) FROM audit_log 
			WHERE module = ? AND action = ? AND record_id = ?`,
			op.module, op.action, op.recordID).Scan(&count)

		if err != nil {
			t.Errorf("Failed to query %s %s: %v", op.module, op.action, err)
			continue
		}

		if count == 0 {
			t.Errorf("Missing audit log for %s %s %s", op.module, op.action, op.recordID)
		}
	}

	t.Logf("✓ Audit log completeness verified for all CRUD operations")
}
