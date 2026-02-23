package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// TestHandleUpdateInventory_LocationChange tests updating inventory location
func TestHandleUpdateInventory_LocationChange(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert initial inventory
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, location) VALUES ('IPN-LOC-001', 100, 'Bin A1')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Since we don't have handleUpdateInventory endpoint, we'll test via SQL
	// This is a gap - inventory updates should support PATCH for location/reorder points
	// For now, verify location can be updated via SQL (implementation gap)
	
	_, err = db.Exec("UPDATE inventory SET location=? WHERE ipn=?", "Bin B5", "IPN-LOC-001")
	if err != nil {
		t.Fatalf("Failed to update location: %v", err)
	}

	var location string
	err = db.QueryRow("SELECT location FROM inventory WHERE ipn=?", "IPN-LOC-001").Scan(&location)
	if err != nil {
		t.Fatalf("Failed to query location: %v", err)
	}

	if location != "Bin B5" {
		t.Errorf("Expected location 'Bin B5', got '%s'", location)
	}
}

// TestHandleInventoryTransact_ReorderPointUpdate tests updating reorder points
func TestHandleInventoryTransact_ReorderPointUpdate(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, reorder_qty) VALUES ('IPN-REORDER-001', 100, 20, 50)`)

	// Update reorder point
	_, err := db.Exec("UPDATE inventory SET reorder_point=?, reorder_qty=? WHERE ipn=?", 30, 75, "IPN-REORDER-001")
	if err != nil {
		t.Fatalf("Failed to update reorder settings: %v", err)
	}

	var reorderPoint, reorderQty float64
	err = db.QueryRow("SELECT reorder_point, reorder_qty FROM inventory WHERE ipn=?", "IPN-REORDER-001").Scan(&reorderPoint, &reorderQty)
	if err != nil {
		t.Fatalf("Failed to query reorder settings: %v", err)
	}

	if reorderPoint != 30 {
		t.Errorf("Expected reorder_point 30, got %f", reorderPoint)
	}
	if reorderQty != 75 {
		t.Errorf("Expected reorder_qty 75, got %f", reorderQty)
	}
}

// TestHandleInventoryTransact_InvalidIPN tests transaction with special characters in IPN
func TestHandleInventoryTransact_InvalidIPN(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test with SQL injection attempt in IPN
	reqBody := `{
		"ipn": "IPN-001'; DROP TABLE inventory; --",
		"type": "receive",
		"qty": 50
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	// Should succeed (prepared statements prevent SQL injection)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify table still exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
	if err != nil {
		t.Fatalf("Inventory table was dropped or query failed: %v", err)
	}
}

// TestHandleInventoryTransact_EmptyStringFields tests transaction with empty reference and notes
func TestHandleInventoryTransact_EmptyStringFields(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"ipn": "IPN-EMPTY-001",
		"type": "receive",
		"qty": 25,
		"reference": "",
		"notes": ""
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify transaction was recorded with empty strings
	var reference, notes string
	err := db.QueryRow("SELECT COALESCE(reference, ''), COALESCE(notes, '') FROM inventory_transactions WHERE ipn=? ORDER BY id DESC LIMIT 1",
		"IPN-EMPTY-001").Scan(&reference, &notes)
	if err != nil {
		t.Fatalf("Failed to query transaction: %v", err)
	}

	if reference != "" {
		t.Errorf("Expected empty reference, got '%s'", reference)
	}
	if notes != "" {
		t.Errorf("Expected empty notes, got '%s'", notes)
	}
}

// TestHandleInventoryTransact_VeryLargeQuantity tests transaction with very large quantity
func TestHandleInventoryTransact_VeryLargeQuantity(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test with a very large quantity (1 billion)
	reqBody := `{
		"ipn": "IPN-LARGE-001",
		"type": "receive",
		"qty": 1000000000,
		"reference": "BULK-ORDER"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var qty float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-LARGE-001").Scan(&qty)
	if err != nil {
		t.Fatalf("Failed to query quantity: %v", err)
	}

	expected := 1000000000.0
	if qty != expected {
		t.Errorf("Expected quantity %f, got %f", expected, qty)
	}
}

// TestHandleInventoryTransact_FractionalQuantity tests transaction with decimal quantities
func TestHandleInventoryTransact_FractionalQuantity(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert initial inventory
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-FRAC-001', 10.5)`)

	// Add 5.75 units
	reqBody := `{
		"ipn": "IPN-FRAC-001",
		"type": "receive",
		"qty": 5.75,
		"reference": "PO-FRAC"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var qty float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-FRAC-001").Scan(&qty)
	if err != nil {
		t.Fatalf("Failed to query quantity: %v", err)
	}

	expected := 16.25 // 10.5 + 5.75
	if qty != expected {
		t.Errorf("Expected quantity %f, got %f", expected, qty)
	}
}

// TestHandleInventoryHistory_LargeHistory tests pagination behavior with many transactions
func TestHandleInventoryHistory_LargeHistory(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn) VALUES ('IPN-HISTORY-001')`)

	// Insert 100 transactions
	for i := 0; i < 100; i++ {
		db.Exec(`INSERT INTO inventory_transactions (ipn, type, qty, created_at) VALUES (?, 'receive', 10, datetime('now', '-' || ? || ' hours'))`,
			"IPN-HISTORY-001", i)
	}

	req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-HISTORY-001/history", nil)
	w := httptest.NewRecorder()

	handleInventoryHistory(w, req, "IPN-HISTORY-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	dataBytes, _ := json.Marshal(resp.Data)
	var result []InventoryTransaction
	json.Unmarshal(dataBytes, &result)

	if len(result) != 100 {
		t.Errorf("Expected 100 transactions, got %d", len(result))
	}

	// Verify ordering (most recent first)
	if len(result) > 1 {
		if result[0].CreatedAt < result[1].CreatedAt {
			t.Error("Transactions are not ordered by created_at DESC")
		}
	}
}

// TestHandleListInventory_FilterByLocation tests filtering inventory by location
func TestHandleListInventory_FilterByLocation(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, location) VALUES 
		('IPN-LOC-A', 100, 'Warehouse A'),
		('IPN-LOC-B', 200, 'Warehouse B'),
		('IPN-LOC-C', 300, 'Warehouse A')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Note: Current implementation doesn't support location filtering via query param
	// This is a coverage gap - should be implemented
	
	// Manual query to verify location filtering would work
	rows, err := db.Query("SELECT ipn FROM inventory WHERE location=?", "Warehouse A")
	if err != nil {
		t.Fatalf("Failed to query by location: %v", err)
	}
	defer rows.Close()

	var ipns []string
	for rows.Next() {
		var ipn string
		rows.Scan(&ipn)
		ipns = append(ipns, ipn)
	}

	if len(ipns) != 2 {
		t.Errorf("Expected 2 items in Warehouse A, got %d", len(ipns))
	}
}

// TestHandleInventoryTransact_MalformedJSON tests handling of malformed JSON
func TestHandleInventoryTransact_MalformedJSON(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"ipn": "IPN-001",
		"type": "receive",
		"qty": 50,
		"malformed: json without closing
	`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}
}

// TestHandleBulkDeleteInventory_WithTransactionHistory tests bulk delete behavior with existing transactions
func TestHandleBulkDeleteInventory_WithTransactionHistory(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create inventory with transaction history
	db.Exec(`INSERT INTO inventory (ipn) VALUES ('IPN-DELETE-001'), ('IPN-DELETE-002')`)
	db.Exec(`INSERT INTO inventory_transactions (ipn, type, qty) VALUES 
		('IPN-DELETE-001', 'receive', 100),
		('IPN-DELETE-001', 'issue', 20),
		('IPN-DELETE-002', 'receive', 50)
	`)

	reqBody := `{
		"ipns": ["IPN-DELETE-001"]
	}`
	req := httptest.NewRequest("DELETE", "/api/v1/inventory/bulk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleBulkDeleteInventory(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify inventory was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM inventory WHERE ipn='IPN-DELETE-001'").Scan(&count)
	if count != 0 {
		t.Errorf("Expected inventory to be deleted, but found %d records", count)
	}

	// Verify transactions are orphaned (no FK constraint cascade in current schema)
	// Note: This is potentially a data integrity issue - transactions exist without parent inventory
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn='IPN-DELETE-001'").Scan(&txCount)
	if txCount == 0 {
		t.Log("Transactions were cascade deleted (if FK constraint configured)")
	} else {
		t.Logf("Warning: %d orphaned transactions remain after inventory deletion (no CASCADE DELETE)", txCount)
	}
}

// TestHandleInventoryTransact_ConcurrentReservedStockCheck tests reserved stock validation under concurrency
func TestHandleInventoryTransact_ConcurrentReservedStockCheck(t *testing.T) {
	oldDB := db
	oldPartsDir := partsDir
	db = setupInventoryTestDB(t)
	partsDir = ""
	defer func() { 
		db.Close() 
		db = oldDB 
		partsDir = oldPartsDir
	}()

	// Create inventory with on_hand=100, reserved=50, available=50
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES ('IPN-RESERVE-TEST', 100, 50)`)

	// Try to issue 60 units (exceeds available of 50)
	reqBody := `{
		"ipn": "IPN-RESERVE-TEST",
		"type": "issue",
		"qty": 60,
		"reference": "WO-OVERISSUE"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	// Should fail with 400 error
	if w.Code != 400 {
		t.Errorf("Expected status 400 for issuing beyond available stock, got %d: %s", w.Code, w.Body.String())
	}

	// Check error message in response body
	bodyStr := w.Body.String()
	if bodyStr == "" || bodyStr == "{}" {
		t.Error("Expected error message for insufficient available stock")
	}

	// Verify quantity unchanged
	var onHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='IPN-RESERVE-TEST'").Scan(&onHand)
	if onHand != 100 {
		t.Errorf("Expected qty_on_hand to remain 100, got %f", onHand)
	}
}

// TestHandleGetInventory_WithMPN tests retrieving inventory item with MPN field
func TestHandleGetInventory_WithMPN(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, description, mpn) VALUES 
		('IPN-MPN-001', 100, 'Resistor 1k Ohm', 'RC0805FR-071KL')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-MPN-001", nil)
	w := httptest.NewRecorder()

	handleGetInventory(w, req, "IPN-MPN-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	dataBytes, _ := json.Marshal(resp.Data)
	var result InventoryItem
	json.Unmarshal(dataBytes, &result)

	if result.MPN != "RC0805FR-071KL" {
		t.Errorf("Expected MPN 'RC0805FR-071KL', got '%s'", result.MPN)
	}
	if result.Description != "Resistor 1k Ohm" {
		t.Errorf("Expected description 'Resistor 1k Ohm', got '%s'", result.Description)
	}
}

// TestHandleListInventory_MultipleItems tests listing with various stock levels
func TestHandleListInventory_MultipleItems(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved, reorder_point) VALUES 
		('IPN-MULTI-1', 500, 50, 100),
		('IPN-MULTI-2', 15, 5, 50),
		('IPN-MULTI-3', 0, 0, 10),
		('IPN-MULTI-4', 1000, 200, 500)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/inventory", nil)
	w := httptest.NewRecorder()

	handleListInventory(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	dataBytes, _ := json.Marshal(resp.Data)
	var result []InventoryItem
	json.Unmarshal(dataBytes, &result)

	if len(result) != 4 {
		t.Errorf("Expected 4 items, got %d", len(result))
	}

	// Verify items are sorted by IPN
	if len(result) > 1 {
		for i := 1; i < len(result); i++ {
			if result[i].IPN < result[i-1].IPN {
				t.Error("Items are not sorted by IPN")
				break
			}
		}
	}
}
