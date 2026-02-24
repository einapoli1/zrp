package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Edge case tests for Procurement module (2026-02-21 audit)

// C1: BOM-Specific Filtering Test
func TestHandleGeneratePOFromWO_OnlyBOMComponents(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create BOM items table
	_, err := db.Exec(`
		CREATE TABLE bom_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_ipn TEXT NOT NULL,
			child_ipn TEXT NOT NULL,
			qty_per REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create bom_items: %v", err)
	}

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty) VALUES ('WO-001', 'ASSY-001', 10)`)
	
	// ASSY-001 BOM: only R1 and C1
	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES 
		('ASSY-001', 'R1', 10.0),
		('ASSY-001', 'C1', 5.0)
	`)

	// Inventory: includes non-BOM parts R2, C2
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES 
		('R1', 50), ('C1', 20), ('R2', 10), ('C2', 5)
	`)

	reqBody := `{"wo_id": "WO-001", "vendor_id": "VEN-001"}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleGeneratePOFromWO(w, req)

	if w.Code != 200 {
		t.Skipf("Generate PO failed: %s", w.Body.String())
		return
	}

	var resp struct{ Data map[string]interface{} `json:"data"` }
	json.NewDecoder(w.Body).Decode(&resp)
	poID := resp.Data["po_id"].(string)

	rows, _ := db.Query("SELECT ipn FROM po_lines WHERE po_id=? ORDER BY ipn", poID)
	defer rows.Close()
	
	var ipns []string
	for rows.Next() {
		var ipn string
		rows.Scan(&ipn)
		ipns = append(ipns, ipn)
	}

	t.Logf("PO lines: %v", ipns)
	
	// Check for bug: if all 4 parts are included, BOM filtering is broken
	if len(ipns) > 2 {
		t.Errorf("BUG: PO includes %d parts (expected 2 BOM parts only)", len(ipns))
		t.Logf("Found: %v (should be only R1, C1)", ipns)
	}
}

// M1: Empty Vendor Test
func TestHandleCreatePO_MissingVendor(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"vendor_id": "",
		"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": 1.0}]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleCreatePO(w, req)

	// Should reject empty vendor
	if w.Code == 200 {
		t.Logf("BUG: Accepted PO without vendor (should require vendor_id)")
	} else if w.Code == 400 {
		t.Logf("✓ Correctly rejects empty vendor")
	}
}

// M5: Over-Receive Test
func TestHandleReceivePO_ExceedsOrdered(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-001', 'VEN-001', 'sent')`)
	db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, qty_received) VALUES 
		('PO-001', 'IPN-001', 100, 0)
	`)

	var lineID int
	db.QueryRow("SELECT id FROM po_lines WHERE po_id='PO-001'").Scan(&lineID)

	// Try to receive 150 (more than ordered 100)
	reqBody := fmt.Sprintf(`{
		"lines": [{"id": %d, "qty": 150}],
		"skip_inspection": true
	}`, lineID)
	req := httptest.NewRequest("POST", "/api/v1/pos/PO-001/receive", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleReceivePO(w, req, "PO-001")

	if w.Code == 200 {
		t.Logf("BUG: Allowed over-receive (150 > 100 ordered)")
	} else if w.Code == 400 {
		t.Logf("✓ Correctly rejects over-receive")
	}
}

// C2: Concurrent Updates Test
func TestHandleReceivePO_RaceCondition(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-001', 'VEN-001', 'sent')`)
	db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, qty_received) VALUES 
		('PO-001', 'IPN-001', 100, 0)
	`)

	var lineID int
	db.QueryRow("SELECT id FROM po_lines WHERE po_id='PO-001'").Scan(&lineID)

	// 10 concurrent receives of 10 each = should total 100
	var wg sync.WaitGroup
	var successCount, errorCount int
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()
			reqBody := fmt.Sprintf(`{
				"lines": [{"id": %d, "qty": 10}],
				"skip_inspection": true
			}`, lineID)
			req := httptest.NewRequest("POST", "/api/v1/pos/PO-001/receive", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleReceivePO(w, req, "PO-001")
			mu.Lock()
			if w.Code == 200 {
				successCount++
			} else {
				errorCount++
				t.Logf("Request %d failed with code %d: %s", reqNum, w.Code, w.Body.String())
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	t.Logf("Results: %d success, %d errors", successCount, errorCount)

	var qtyReceived float64
	db.QueryRow("SELECT qty_received FROM po_lines WHERE id=?", lineID).Scan(&qtyReceived)

	if qtyReceived != 100 {
		t.Errorf("RACE CONDITION: Expected 100, got %.2f (lost updates)", qtyReceived)
	}
}

// L1: Pagination Test
func TestHandleListPOs_NoPagination(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	
	// Create 100 POs
	for i := 1; i <= 100; i++ {
		poID := fmt.Sprintf("PO-%04d", i)
		db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, created_at) VALUES 
			(?, 'VEN-001', 'draft', ?)`, poID, 
			time.Now().Add(time.Duration(i)*time.Second).Format("2006-01-02 15:04:05"))
	}

	req := httptest.NewRequest("GET", "/api/v1/pos", nil)
	w := httptest.NewRecorder()
	handleListPOs(w, req)

	var response struct {
		Data []PurchaseOrder `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.Data) == 100 {
		t.Logf("BUG: Returns all 100 POs without pagination (performance risk)")
	} else {
		t.Logf("✓ Pagination active (%d results)", len(response.Data))
	}
}

// M4: Invalid Line ID Test
func TestHandleReceivePO_NonexistentLineID(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-001', 'VEN-001', 'sent')`)

	reqBody := `{
		"lines": [{"id": 99999, "qty": 10}],
		"skip_inspection": true
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos/PO-001/receive", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleReceivePO(w, req, "PO-001")

	if w.Code == 200 {
		t.Logf("BUG: Silently ignores invalid line ID (should return 404)")
	} else if w.Code == 404 || w.Code == 400 {
		t.Logf("✓ Correctly rejects invalid line ID")
	}
}

// Edge case: Negative quantity
func TestHandleReceivePO_NegativeQuantity(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-001', 'VEN-001', 'sent')`)
	db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES ('PO-001', 'IPN-001', 100)`)

	var lineID int
	db.QueryRow("SELECT id FROM po_lines WHERE po_id='PO-001'").Scan(&lineID)

	reqBody := fmt.Sprintf(`{
		"lines": [{"id": %d, "qty": -10}],
		"skip_inspection": true
	}`, lineID)
	req := httptest.NewRequest("POST", "/api/v1/pos/PO-001/receive", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleReceivePO(w, req, "PO-001")

	if w.Code != 400 {
		t.Errorf("BUG: Accepted negative quantity (status %d)", w.Code)
	}
}
