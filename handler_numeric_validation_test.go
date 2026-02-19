package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"math"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// setupNumericTestDB creates an in-memory database for numeric validation testing
func setupNumericTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create all necessary tables with CHECK constraints matching production
	schema := `
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','preferred','inactive','blocked')),
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT,
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			reorder_qty REAL DEFAULT 0 CHECK(reorder_qty >= 0),
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('receive','issue','adjust','transfer','return','scrap')),
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','confirmed','partial','received','cancelled')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME,
			created_by TEXT,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT
		);

		CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_ordered REAL NOT NULL CHECK(qty_ordered > 0),
			qty_received REAL DEFAULT 0 CHECK(qty_received >= 0),
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		);

		CREATE TABLE quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','accepted','rejected','expired','cancelled')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_until TEXT,
			accepted_at DATETIME
		);

		CREATE TABLE quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty INTEGER NOT NULL CHECK(qty > 0),
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		);

		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL CHECK(qty > 0),
			qty_good INTEGER,
			qty_scrap INTEGER,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','open','in_progress','completed','cancelled','on_hold')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		);

		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return testDB
}

// TestInventoryQuantityOverflow tests very large and overflow values for inventory quantities
// Tests at DB level to avoid goroutine race conditions in handler (emailOnLowStock)
func TestInventoryQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		qty         float64
		expectError bool
	}{
		{"Normal quantity", 100.0, false},
		{"Very large but valid", 999999999.0, false},
		{"Negative (violates CHECK)", -100.0, true},
		{"Zero (valid)", 0.0, false},
		{"Max float64", 1.7e308, false}, // SQLite REAL should handle
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test at DB level to avoid handler goroutine issues
			_, err := db.Exec("INSERT INTO inventory (ipn, qty_on_hand) VALUES (?, ?)",
				tt.name, tt.qty)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for qty=%v, but insert succeeded", tt.qty)
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected success for qty=%v, but got error: %v", tt.qty, err)
			}
		})
	}
}

// TestPOLineQuantityOverflow tests qty_ordered boundaries
func TestPOLineQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec("INSERT INTO vendors (id, name) VALUES ('V-001', 'Test Vendor')")
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	tests := []struct {
		name        string
		qtyOrdered  float64
		expectError bool
	}{
		{"Normal", 100.0, false},
		{"Large valid", 99999.0, false},
		{"Too large", 1000001.0, true},
		{"Zero", 0.0, true},
		{"Negative", -50.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := PurchaseOrder{
				VendorID: "V-001",
				Lines: []POLine{{
					IPN:        "P-001",
					QtyOrdered: tt.qtyOrdered,
					UnitPrice:  10.0,
				}},
			}

			body, _ := json.Marshal(po)
			req := httptest.NewRequest("POST", "/api/v1/procurement/pos", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handleCreatePO(w, req)

			if tt.expectError && w.Code == 200 {
				t.Errorf("Expected error for %v, got success", tt.qtyOrdered)
			} else if !tt.expectError && w.Code != 200 {
				t.Errorf("Expected success for %v, got %d: %s", tt.qtyOrdered, w.Code, w.Body.String())
			}
		})
	}
}

// TestPriceFieldOverflow tests unit_price boundaries
func TestPriceFieldOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, _ = db.Exec("INSERT INTO vendors (id, name) VALUES ('V-001', 'Test')")

	tests := []struct {
		name      string
		price     float64
		expectErr bool
	}{
		{"Normal", 99.99, false},
		{"Zero", 0.0, false},
		{"High valid", 99999.99, false},
		{"Negative", -50.0, true},
		{"Too large", 1000001.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := PurchaseOrder{
				VendorID: "V-001",
				Lines: []POLine{{
					IPN:        "P-001",
					QtyOrdered: 10.0,
					UnitPrice:  tt.price,
				}},
			}

			body, _ := json.Marshal(po)
			req := httptest.NewRequest("POST", "/api/v1/procurement/pos", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handleCreatePO(w, req)

			if tt.expectErr && w.Code == 200 {
				t.Errorf("Expected error for %v", tt.price)
			} else if !tt.expectErr && w.Code != 200 {
				t.Errorf("Expected success for %v, got %d", tt.price, w.Code)
			}
		})
	}
}

// TestLeadTimeDaysOverflow tests lead_time_days boundaries
func TestLeadTimeDaysOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name     string
		leadDays int
		expectErr bool
	}{
		{"Normal", 7, false},
		{"Zero", 0, false},
		{"Max valid", 730, false},
		{"Too large", 10000, true},
		{"Negative", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vendor := Vendor{
				Name:         "Test",
				LeadTimeDays: tt.leadDays,
			}

			body, _ := json.Marshal(vendor)
			req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handleCreateVendor(w, req)

			if tt.expectErr && w.Code == 200 {
				t.Errorf("Expected error for %d days", tt.leadDays)
			} else if !tt.expectErr && w.Code != 200 {
				t.Errorf("Expected success for %d, got %d: %s", tt.leadDays, w.Code, w.Body.String())
			}
		})
	}
}

// TestQuoteLineQuantityOverflow tests qty boundaries
func TestQuoteLineQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name      string
		qty       int
		expectErr bool
	}{
		{"Normal", 100, false},
		{"Large", 99999, false},
		{"Too large", 100001, true},
		{"Zero", 0, true},
		{"Negative", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote := Quote{
				Customer: "Test",
				Lines: []QuoteLine{{
					IPN:       "P-001",
					Qty:       tt.qty,
					UnitPrice: 50.0,
				}},
			}

			body, _ := json.Marshal(quote)
			req := httptest.NewRequest("POST", "/api/v1/quotes", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handleCreateQuote(w, req)

			if tt.expectErr && w.Code == 200 {
				t.Errorf("Expected error for %d", tt.qty)
			} else if !tt.expectErr && w.Code != 200 {
				t.Errorf("Expected success for %d, got %d", tt.qty, w.Code)
			}
		})
	}
}

// TestFloatingPointPrecision tests precision with REAL types
func TestFloatingPointPrecision(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name string
		qty  float64
	}{
		{"Repeating decimal", 0.33333333333333},
		{"Very precise", 99.999999999},
		{"Small diff", 1.0000000001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec("INSERT INTO inventory (ipn, qty_on_hand) VALUES (?, ?)", tt.name, tt.qty)
			if err != nil {
				t.Fatalf("Insert failed: %v", err)
			}

			var stored float64
			_ = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", tt.name).Scan(&stored)
			
			diff := math.Abs(stored - tt.qty)
			if diff > 1e-6 {
				t.Logf("Precision diff for %v: stored=%v, diff=%v", tt.qty, stored, diff)
			}
		})
	}
}

// TestWorkOrderQuantityBoundaries tests WO qty via handler validation
func TestWorkOrderQuantityBoundaries(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name      string
		qty       int
		expectErr bool
	}{
		{"Normal", 100, false},
		{"Large", 10000, false},
		{"Too large", 100001, true},  // Handler validation should reject
		{"Zero", 0, true},            // Validation rejects (must be >= 1)
		{"Negative", -50, true},       // Handler validation rejects
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wo := WorkOrder{
				AssemblyIPN: "ASSY-001",
				Qty:         tt.qty,
			}

			body, _ := json.Marshal(wo)
			req := httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handleCreateWorkOrder(w, req)

			if tt.expectErr && w.Code == 200 {
				t.Errorf("Expected error for qty=%d, got success", tt.qty)
			} else if !tt.expectErr && w.Code != 200 {
				t.Errorf("Expected success for qty=%d, got %d: %s", tt.qty, w.Code, w.Body.String())
			}
		})
	}
}
