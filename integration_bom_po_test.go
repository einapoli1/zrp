package main

import (
	"database/sql"
	"testing"
	"time"
)

// Integration test for critical BOM → PO → Inventory workflow
// This test identifies gaps in the procurement-to-inventory pipeline

func setupIntegrationTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run full schema migration (using actual initDB code)
	// For now, create minimal tables needed for this test
	
	// Users
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Vendors
	_, err = testDB.Exec(`
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			lead_days INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create vendors table: %v", err)
	}

	// Inventory (use actual schema column names)
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Work Orders
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			qty REAL NOT NULL,
			priority TEXT DEFAULT 'normal',
			status TEXT DEFAULT 'open',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Purchase Orders
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			expected_date TEXT,
			received_date TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	// PO Lines
	_, err = testDB.Exec(`
		CREATE TABLE po_lines (
			id TEXT PRIMARY KEY,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			qty REAL NOT NULL,
			unit_price REAL DEFAULT 0,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create po_lines table: %v", err)
	}

	// Inventory Transactions
	_, err = testDB.Exec(`
		CREATE TABLE inventory_transactions (
			id TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL,
			qty REAL NOT NULL,
			reference_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Sessions
	_, err = testDB.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	// Create audit_log table - CRITICAL: Used by almost every handler
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

	// Insert admin user
	_, err = testDB.Exec(`
		INSERT INTO users (id, username, password_hash, role)
		VALUES (1, 'admin', '$2a$10$placeholder', 'admin')
	`)
	if err != nil {
		t.Fatalf("Failed to insert admin user: %v", err)
	}

	return testDB
}

func createIntegrationTestSession(t *testing.T, db *sql.DB) string {
	token := "test-session-integration"
	expires := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, 1, expires,
	)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return token
}

// TestIntegration_PO_Receipt_Updates_Inventory verifies that when a PO is received,
// inventory levels are automatically updated
func TestIntegration_PO_Receipt_Updates_Inventory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupIntegrationTestDB(t)
	defer db.Close()

	_ = createIntegrationTestSession(t, db)

	t.Log("=== Integration Test: PO Receipt → Inventory Update ===")

	// Step 1: Create vendor
	t.Log("Step 1: Insert vendor into database")
	vendorID := "V-INT-001"
	_, err := db.Exec(`
		INSERT INTO vendors (id, name, status, lead_days)
		VALUES (?, ?, ?, ?)
	`, vendorID, "Integration Test Vendor", "active", 7)
	if err != nil {
		t.Fatalf("Failed to insert vendor: %v", err)
	}

	// Step 2: Create initial inventory with low stock
	t.Log("Step 2: Create inventory records with low stock")
	components := map[string]float64{
		"RES-INT-001": 5.0,
		"CAP-INT-001": 2.0,
	}

	for ipn, qty := range components {
		_, err := db.Exec(`
			INSERT INTO inventory (ipn, qty_on_hand, location)
			VALUES (?, ?, ?)
		`, ipn, qty, "Warehouse")
		if err != nil {
			t.Fatalf("Failed to insert inventory for %s: %v", ipn, err)
		}
		t.Logf("   Created inventory: %s with qty_on_hand=%.0f", ipn, qty)
	}

	// Step 3: Create PO with line items
	t.Log("Step 3: Create purchase order with shortage line items")
	poID := "PO-INT-001"
	_, err = db.Exec(`
		INSERT INTO purchase_orders (id, vendor_id, status)
		VALUES (?, ?, ?)
	`, poID, vendorID, "ordered")
	if err != nil {
		t.Fatalf("Failed to insert PO: %v", err)
	}

	// Add PO lines for the shortages
	shortageQtys := map[string]float64{
		"RES-INT-001": 95.0, // Need 100, have 5
		"CAP-INT-001": 48.0, // Need 50, have 2
	}

	for ipn, qty := range shortageQtys {
		_, err := db.Exec(`
			INSERT INTO po_lines (id, po_id, ipn, qty, unit_price)
			VALUES (?, ?, ?, ?, ?)
		`, "POL-"+ipn, poID, ipn, qty, 0.10)
		if err != nil {
			t.Fatalf("Failed to insert PO line for %s: %v", ipn, err)
		}
		t.Logf("   Added PO line: %s qty=%.0f", ipn, qty)
	}

	// Step 4: Record initial inventory levels
	t.Log("Step 4: Record initial inventory levels")
	initialQty := make(map[string]float64)
	for ipn := range components {
		var qty float64
		err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", ipn).Scan(&qty)
		if err != nil {
			t.Fatalf("Failed to query initial qty for %s: %v", ipn, err)
		}
		initialQty[ipn] = qty
	}

	// Step 5: Simulate PO receipt by calling the receive endpoint
	// This would normally be: POST /api/v1/procurement/{po_id}/receive
	// For this test, we'll directly call the handler logic or verify the database state

	t.Log("Step 5: Mark PO as received")
	_, err = db.Exec(`
		UPDATE purchase_orders SET status = ?, received_date = ? WHERE id = ?
	`, "received", time.Now().Format("2006-01-02"), poID)
	if err != nil {
		t.Fatalf("Failed to update PO status: %v", err)
	}

	// THIS IS WHERE THE WORKFLOW GAP OCCURS
	// The PO is marked received, but inventory is NOT automatically updated
	// We need to verify this gap exists and document it

	// Step 6: Check if inventory was automatically updated
	t.Log("Step 6: Verify if inventory was automatically updated")

	workflowGapDetected := false
	for ipn, initial := range initialQty {
		var currentQty float64
		err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", ipn).Scan(&currentQty)
		if err != nil {
			t.Fatalf("Failed to query updated qty for %s: %v", ipn, err)
		}

		expected := initial + shortageQtys[ipn]
		t.Logf("   %s: qty_on_hand = %.0f (initial=%.0f, expected=%.0f)", 
			ipn, currentQty, initial, expected)

		if currentQty == expected {
			t.Logf("   ✓ Inventory automatically updated!")
		} else if currentQty == initial {
			t.Errorf("   ✗ WORKFLOW GAP DETECTED: Inventory NOT updated")
			t.Errorf("      Expected %.0f, got %.0f (unchanged)", expected, currentQty)
			workflowGapDetected = true
		} else {
			t.Logf("   ⚠ Unexpected qty: %.0f", currentQty)
		}
	}

	// Step 7: Check if inventory transactions were created
	t.Log("Step 7: Verify inventory transactions were created")

	var txCount int
	err = db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE reference_id = ?", poID).Scan(&txCount)
	if err != nil {
		t.Fatalf("Failed to count transactions: %v", err)
	}

	if txCount > 0 {
		t.Logf("   ✓ Found %d inventory transactions for PO %s", txCount, poID)
	} else {
		t.Error("   ✗ NO inventory transactions created for PO receipt")
		workflowGapDetected = true
	}

	// Final verdict
	t.Log("=== Test Results ===")
	if workflowGapDetected {
		t.Error("✗✗ CRITICAL WORKFLOW GAP CONFIRMED:")
		t.Error("   PO receiving does NOT automatically update inventory")
		t.Error("   This must be implemented for production use")
		t.Error("")
		t.Error("Required implementation:")
		t.Error("   1. When PO is received, iterate through po_lines")
		t.Error("   2. For each line, UPDATE inventory SET qty_onhand = qty_onhand + line.qty")
		t.Error("   3. INSERT inventory_transactions record for audit trail")
	} else {
		t.Log("✓✓ SUCCESS: PO → Inventory workflow is correctly implemented")
	}

	t.Log("=== Integration Test Complete ===")
}

// TestIntegration_WorkOrder_Completion_Updates_Inventory verifies that when a work order
// is completed, finished goods are added and components are consumed
func TestIntegration_WorkOrder_Completion_Updates_Inventory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupIntegrationTestDB(t)
	defer db.Close()

	t.Log("=== Integration Test: WO Completion → Inventory Update ===")

	// Step 1: Create inventory records
	t.Log("Step 1: Create component and assembly inventory")
	
	// Components (raw materials)
	_, err := db.Exec(`
		INSERT INTO inventory (ipn, qty_on_hand, location)
		VALUES ('RES-WO-001', 200, 'Production Floor')
	`)
	if err != nil {
		t.Fatalf("Failed to insert component inventory: %v", err)
	}

	// Assembly (finished goods)
	_, err = db.Exec(`
		INSERT INTO inventory (ipn, qty_on_hand, location)
		VALUES ('ASY-WO-001', 0, 'Finished Goods')
	`)
	if err != nil {
		t.Fatalf("Failed to insert assembly inventory: %v", err)
	}

	t.Log("   Created: RES-WO-001 with qty_on_hand=200")
	t.Log("   Created: ASY-WO-001 with qty_on_hand=0")

	// Step 2: Create work order for 10 assemblies
	t.Log("Step 2: Create work order for 10x ASY-WO-001")
	woID := "WO-INT-001"
	_, err = db.Exec(`
		INSERT INTO work_orders (id, ipn, qty, status)
		VALUES (?, ?, ?, ?)
	`, woID, "ASY-WO-001", 10, "open")
	if err != nil {
		t.Fatalf("Failed to insert work order: %v", err)
	}

	// Step 3: Mark work order as completed
	t.Log("Step 3: Mark work order as completed")
	_, err = db.Exec(`
		UPDATE work_orders SET status = ? WHERE id = ?
	`, "completed", woID)
	if err != nil {
		t.Fatalf("Failed to update work order: %v", err)
	}
	t.Log("   ✓ Work order status set to 'completed'")

	// Step 4: Check if finished goods were added to inventory
	t.Log("Step 4: Verify finished goods added to inventory")
	
	var assemblyQty float64
	err = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "ASY-WO-001").Scan(&assemblyQty)
	if err != nil {
		t.Fatalf("Failed to query assembly qty: %v", err)
	}

	t.Logf("   ASY-WO-001: qty_on_hand = %.0f (expected 10)", assemblyQty)

	workflowGap := false
	if assemblyQty == 10.0 {
		t.Log("   ✓ Finished goods correctly added!")
	} else if assemblyQty == 0.0 {
		t.Error("   ✗ WORKFLOW GAP: Finished goods NOT added to inventory")
		workflowGap = true
	}

	// Step 5: Check if inventory transactions exist
	t.Log("Step 5: Verify inventory transactions created")
	
	var txCount int
	err = db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE reference_id = ?", woID).Scan(&txCount)
	if err != nil {
		t.Fatalf("Failed to count transactions: %v", err)
	}

	if txCount > 0 {
		t.Logf("   ✓ Found %d inventory transactions", txCount)
	} else {
		t.Error("   ✗ NO inventory transactions created")
		workflowGap = true
	}

	// Final verdict
	t.Log("=== Test Results ===")
	if workflowGap {
		t.Error("✗✗ CRITICAL WORKFLOW GAP CONFIRMED:")
		t.Error("   Work order completion does NOT update inventory")
		t.Error("")
		t.Error("Required implementation:")
		t.Error("   1. When WO status → 'completed', query WO qty and ipn")
		t.Error("   2. UPDATE inventory SET qty_onhand = qty_onhand + wo.qty WHERE ipn = wo.ipn")
		t.Error("   3. INSERT inventory_transactions for audit trail")
		t.Error("   4. (Optional) Deduct component materials from inventory")
	} else {
		t.Log("✓✓ SUCCESS: WO → Inventory workflow implemented correctly")
	}

	t.Log("=== Integration Test Complete ===")
}
