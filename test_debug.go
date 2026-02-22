package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	_ "modernc.org/sqlite"
)

func TestDebugInventoryReport(t *testing.T) {
	oldDB := db
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	db = testDB
	defer func() { db.Close(); db = oldDB }()

	// Create tables
	testDB.Exec("PRAGMA foreign_keys = ON")
	testDB.Exec(`CREATE TABLE inventory (
		ipn TEXT PRIMARY KEY,
		description TEXT,
		mpn TEXT,
		qty_on_hand REAL DEFAULT 0,
		reorder_point REAL DEFAULT 0,
		reorder_qty REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	testDB.Exec(`CREATE TABLE purchase_orders (
		id TEXT PRIMARY KEY,
		vendor_id TEXT,
		status TEXT DEFAULT 'draft',
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expected_date TEXT,
		received_at DATETIME
	)`)
	testDB.Exec(`CREATE TABLE po_lines (
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
	)`)

	// Insert test data
	testDB.Exec("INSERT INTO inventory (ipn, description, mpn, qty_on_hand) VALUES ('RES-001', '100 Ohm Resistor', 'RC0805FR-07100RL', 5000)")
	testDB.Exec("INSERT INTO purchase_orders (id, vendor_id, status, created_at) VALUES ('PO-001', 'VENDOR-001', 'received', '2024-01-01 10:00:00')")
	testDB.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES ('PO-001', 'RES-001', 100, 0.05)")

	// Make request
	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()
	handleReportInventoryValuation(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Body: %s\n", w.Body.String())

	var report InvValuationReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		fmt.Printf("Decode error: %v\n", err)
	} else {
		fmt.Printf("Groups: %d\n", len(report.Groups))
		fmt.Printf("GrandTotal: %.2f\n", report.GrandTotal)
	}
}
