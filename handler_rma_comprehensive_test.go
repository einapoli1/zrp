package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// =============================================================================
// CONCURRENCY & RACE CONDITION TESTS
// =============================================================================

func TestHandleUpdateRMA_ConcurrentStatusUpdates(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "Test", "")

	// Simulate 10 concurrent status updates
	var wg sync.WaitGroup
	statusUpdates := []string{"received", "diagnosing", "repairing", "resolved", "closed"}
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status := statusUpdates[idx%len(statusUpdates)]
			reqBody := fmt.Sprintf(`{"serial_number":"SN12345","reason":"Test","status":"%s"}`, status)
			req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleUpdateRMA(w, req, "RMA-001")

			if w.Code != 200 {
				errors <- fmt.Errorf("Update %d failed with status %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check if any errors occurred
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent update error: %v", err)
		errorCount++
	}

	// Verify final state is consistent
	var finalStatus string
	err := db.QueryRow("SELECT status FROM rmas WHERE id = ?", "RMA-001").Scan(&finalStatus)
	if err != nil {
		t.Fatalf("Failed to query final status: %v", err)
	}

	// Final status should be one of the valid statuses
	validFinalStatus := false
	for _, s := range statusUpdates {
		if finalStatus == s {
			validFinalStatus = true
			break
		}
	}
	if !validFinalStatus {
		t.Errorf("Final status '%s' is not one of the expected values", finalStatus)
	}

	t.Logf("Concurrent updates completed. Errors: %d/10, Final status: %s", errorCount, finalStatus)
}

func TestHandleUpdateRMA_ConcurrentWithRead(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "Test", "")

	var wg sync.WaitGroup
	stopReading := make(chan bool)
	readErrors := make(chan error, 100)
	writeErrors := make(chan error, 10)

	// Reader goroutines (simulating users viewing the RMA)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for {
				select {
				case <-stopReading:
					return
				default:
					req := httptest.NewRequest("GET", "/api/rmas/RMA-001", nil)
					w := httptest.NewRecorder()
					handleGetRMA(w, req, "RMA-001")
					if w.Code != 200 {
						readErrors <- fmt.Errorf("Reader %d got status %d", idx, w.Code)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Writer goroutines (simulating concurrent updates)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			statuses := []string{"received", "diagnosing", "repairing"}
			status := statuses[idx%len(statuses)]
			reqBody := fmt.Sprintf(`{"serial_number":"SN12345","reason":"Test","status":"%s"}`, status)
			req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleUpdateRMA(w, req, "RMA-001")

			if w.Code != 200 {
				writeErrors <- fmt.Errorf("Writer %d failed with status %d", idx, w.Code)
			}
		}(i)
	}

	// Let readers and writers run concurrently
	time.Sleep(100 * time.Millisecond)
	close(stopReading)

	wg.Wait()
	close(readErrors)
	close(writeErrors)

	// Count errors
	readErrCount := 0
	for err := range readErrors {
		t.Logf("Read error: %v", err)
		readErrCount++
	}

	writeErrCount := 0
	for err := range writeErrors {
		t.Logf("Write error: %v", err)
		writeErrCount++
	}

	t.Logf("Concurrent read/write test: Read errors: %d, Write errors: %d", readErrCount, writeErrCount)

	// At least some operations should succeed
	if readErrCount > 50 {
		t.Errorf("Too many read errors: %d", readErrCount)
	}
}

func TestHandleCreateRMA_ConcurrentCreates(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	var wg sync.WaitGroup
	errors := make(chan error, 20)
	createdIDs := make(chan string, 20)

	// Create 20 RMAs concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reqBody := fmt.Sprintf(`{
				"serial_number": "SN%05d",
				"reason": "Concurrent test %d"
			}`, idx, idx)
			req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreateRMA(w, req)

			if w.Code != 200 {
				errors <- fmt.Errorf("Create %d failed with status %d: %s", idx, w.Code, w.Body.String())
			} else {
				var resp APIResponse
				json.NewDecoder(w.Body).Decode(&resp)
				rmaData := resp.Data.(map[string]interface{})
				createdIDs <- rmaData["id"].(string)
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(createdIDs)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent create error: %v", err)
		errorCount++
	}

	// Collect created IDs
	ids := make(map[string]bool)
	for id := range createdIDs {
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	// Verify all IDs are unique
	expectedCount := 20 - errorCount
	if len(ids) != expectedCount {
		t.Errorf("Expected %d unique IDs, got %d", expectedCount, len(ids))
	}

	t.Logf("Concurrent creates: %d succeeded, %d failed, %d unique IDs", len(ids), errorCount, len(ids))
}

// =============================================================================
// INVENTORY INTEGRATION TESTS (DOCUMENTING MISSING FUNCTIONALITY)
// =============================================================================

func TestHandleUpdateRMA_InventoryReturnFlow_MISSING(t *testing.T) {
	t.Skip("MISSING FEATURE: RMA → Inventory return flow not implemented")

	// This test documents the expected behavior for inventory integration:
	// When an RMA is marked as "received" and the defective unit is returned to inventory:
	// 1. RMA should have optional fields: returned_to_inventory (bool), returned_ipn (string), returned_qty (float)
	// 2. On status update to specific states (e.g., "scrapped", "resolved"), should:
	//    - Create inventory transaction (type: "rma_return" or "rma_scrap")
	//    - Update inventory.qty_on_hand for the returned part
	//    - Link the transaction to the RMA ID
	// 3. Should prevent marking as scrapped/resolved without specifying returned_ipn
	// 4. Should validate returned_ipn exists in parts table

	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Would need to create inventory and parts tables
	// db.Exec(`CREATE TABLE inventory (ipn TEXT PRIMARY KEY, qty_on_hand REAL DEFAULT 0)`)
	// db.Exec(`CREATE TABLE inventory_transactions (id INTEGER PRIMARY KEY, ipn TEXT, type TEXT, qty REAL, reference TEXT)`)

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "DOA unit", "received", "Unit dead", "")

	// Expected workflow:
	reqBody := `{
		"serial_number": "SN12345",
		"reason": "DOA unit",
		"status": "scrapped",
		"returned_to_inventory": true,
		"returned_ipn": "PART-123",
		"returned_qty": 1.0,
		"defect_description": "Unit dead"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	// Should succeed and create inventory transaction
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify inventory transaction was created
	// var txnType, reference string
	// var qty float64
	// err := db.QueryRow("SELECT type, qty, reference FROM inventory_transactions WHERE reference = ?", "RMA-001").
	// 	Scan(&txnType, &qty, &reference)
	// if err != nil {
	// 	t.Errorf("Expected inventory transaction to be created: %v", err)
	// }
	// if txnType != "rma_scrap" {
	// 	t.Errorf("Expected transaction type 'rma_scrap', got '%s'", txnType)
	// }
	// if qty != 1.0 {
	// 	t.Errorf("Expected qty 1.0, got %.1f", qty)
	// }
}

func TestHandleUpdateRMA_PreventScrapWithoutInventoryInfo_MISSING(t *testing.T) {
	t.Skip("MISSING FEATURE: Validation for inventory return not implemented")

	// Should prevent marking as scrapped without returned_ipn
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "received", "Test", "")

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"status": "scrapped"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	// Should fail validation (missing returned_ipn)
	if w.Code != 400 {
		t.Errorf("Expected status 400 for missing returned_ipn, got %d", w.Code)
	}
}

// =============================================================================
// REFUND/REPLACEMENT WORKFLOW TESTS (DOCUMENTING MISSING FUNCTIONALITY)
// =============================================================================

func TestHandleUpdateRMA_RefundWorkflow_MISSING(t *testing.T) {
	t.Skip("MISSING FEATURE: Refund/replacement workflow not implemented")

	// This test documents the expected behavior for refund workflow:
	// RMA should have fields: resolution_type (enum: 'refund', 'replacement', 'repair'), refund_amount (decimal), refund_issued_at (datetime)
	// When resolution_type = 'refund':
	// 1. Require refund_amount to be set
	// 2. Set refund_issued_at timestamp when status transitions to 'closed'
	// 3. Optionally integrate with accounting system (create credit memo, etc.)

	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "DOA - customer wants refund", "diagnosing", "Unit defective", "")

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "DOA - customer wants refund",
		"status": "closed",
		"resolution_type": "refund",
		"refund_amount": 299.99,
		"resolution": "Full refund issued to customer"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify refund was recorded
	// var resolutionType string
	// var refundAmount sql.NullFloat64
	// var refundIssuedAt sql.NullString
	// err := db.QueryRow("SELECT resolution_type, refund_amount, refund_issued_at FROM rmas WHERE id = ?", "RMA-001").
	// 	Scan(&resolutionType, &refundAmount, &refundIssuedAt)
	// if err != nil {
	// 	t.Fatalf("Failed to query refund data: %v", err)
	// }
	// if resolutionType != "refund" {
	// 	t.Errorf("Expected resolution_type 'refund', got '%s'", resolutionType)
	// }
	// if !refundAmount.Valid || refundAmount.Float64 != 299.99 {
	// 	t.Errorf("Expected refund_amount 299.99, got %v", refundAmount)
	// }
	// if !refundIssuedAt.Valid {
	// 	t.Error("Expected refund_issued_at to be set")
	// }
}

func TestHandleUpdateRMA_ReplacementWorkflow_MISSING(t *testing.T) {
	t.Skip("MISSING FEATURE: Replacement workflow not implemented")

	// This test documents the expected behavior for replacement workflow:
	// RMA should have fields: resolution_type, replacement_serial_number, replacement_shipped_at
	// When resolution_type = 'replacement':
	// 1. Require replacement_serial_number to be set
	// 2. Set replacement_shipped_at timestamp when status transitions to 'shipped'
	// 3. Optionally create a new device record for the replacement unit
	// 4. Link replacement device to original RMA

	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Defective unit - needs replacement", "diagnosing", "Power supply failed", "")

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Defective unit - needs replacement",
		"status": "shipped",
		"resolution_type": "replacement",
		"replacement_serial_number": "SN67890",
		"resolution": "Replacement unit shipped"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify replacement was recorded
	// var resolutionType, replacementSN string
	// var replacementShippedAt sql.NullString
	// err := db.QueryRow("SELECT resolution_type, replacement_serial_number, replacement_shipped_at FROM rmas WHERE id = ?", "RMA-001").
	// 	Scan(&resolutionType, &replacementSN, &replacementShippedAt)
	// if err != nil {
	// 	t.Fatalf("Failed to query replacement data: %v", err)
	// }
	// if resolutionType != "replacement" {
	// 	t.Errorf("Expected resolution_type 'replacement', got '%s'", resolutionType)
	// }
	// if replacementSN != "SN67890" {
	// 	t.Errorf("Expected replacement_serial_number 'SN67890', got '%s'", replacementSN)
	// }
	// if !replacementShippedAt.Valid {
	// 	t.Error("Expected replacement_shipped_at to be set")
	// }
}

func TestHandleCreateRMA_RequireResolutionType_MISSING(t *testing.T) {
	t.Skip("MISSING FEATURE: Resolution type validation not implemented")

	// When creating RMA, should allow specifying expected resolution_type upfront
	// Validation should ensure:
	// - resolution_type is one of: 'refund', 'replacement', 'repair', 'pending'
	// - Default to 'pending' if not specified

	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Customer wants refund",
		"resolution_type": "refund"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// var resolutionType string
	// db.QueryRow("SELECT resolution_type FROM rmas WHERE serial_number = ?", "SN12345").Scan(&resolutionType)
	// if resolutionType != "refund" {
	// 	t.Errorf("Expected resolution_type 'refund', got '%s'", resolutionType)
	// }
}

// =============================================================================
// ADDITIONAL EDGE CASES
// =============================================================================

func TestHandleUpdateRMA_TimestampImmutability(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert RMA with received_at already set
	timestamp := "2024-01-01 10:00:00"
	_, err := db.Exec(`
		INSERT INTO rmas (id, serial_number, reason, status, received_at, created_at) 
		VALUES ('RMA-001', 'SN12345', 'Test', 'received', ?, datetime('now'))
	`, timestamp)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	// Update to diagnosing (should NOT change received_at)
	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"status": "diagnosing"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify received_at was NOT changed (COALESCE preserves existing value)
	var receivedAt string
	err = db.QueryRow("SELECT received_at FROM rmas WHERE id = ?", "RMA-001").Scan(&receivedAt)
	if err != nil {
		t.Fatalf("Failed to query received_at: %v", err)
	}

	if !strings.Contains(receivedAt, "2024-01-01") {
		t.Errorf("Expected received_at to be preserved (2024-01-01), got '%s'", receivedAt)
	}
}

func TestHandleUpdateRMA_StatusTransitionToSameStatus(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "received", "Test", "")

	// Update to same status (should succeed - idempotent)
	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"status": "received"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200 for idempotent update, got %d", w.Code)
	}
}

func TestHandleUpdateRMA_UnicodeAndEmoji(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "", "")

	// Update with Unicode and emoji characters
	reqBody := `{
		"serial_number": "SN12345",
		"customer": "Société Française 🇫🇷",
		"reason": "Устройство не работает",
		"defect_description": "屏幕破裂 💔",
		"resolution": "Problema resuelto ✅",
		"status": "closed"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200 for Unicode/emoji, got %d: %s", w.Code, w.Body.String())
	}

	// Verify data was stored correctly
	var customer, reason, defect, resolution string
	err := db.QueryRow("SELECT customer, reason, defect_description, resolution FROM rmas WHERE id = ?", "RMA-001").
		Scan(&customer, &reason, &defect, &resolution)
	if err != nil {
		t.Fatalf("Failed to query RMA: %v", err)
	}

	if !strings.Contains(customer, "🇫🇷") {
		t.Errorf("Emoji not preserved in customer field: %s", customer)
	}
	if reason != "Устройство не работает" {
		t.Errorf("Cyrillic not preserved in reason field: %s", reason)
	}
	if !strings.Contains(defect, "屏幕破裂") {
		t.Errorf("Chinese characters not preserved in defect_description: %s", defect)
	}
	if !strings.Contains(resolution, "✅") {
		t.Errorf("Emoji not preserved in resolution field: %s", resolution)
	}
}

func TestHandleCreateRMA_VeryLongSerialNumber(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test with exactly 100 characters (max allowed)
	longSN := strings.Repeat("X", 100)
	reqBody := fmt.Sprintf(`{
		"serial_number": "%s",
		"reason": "Test"
	}`, longSN)
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for 100-char serial number, got %d", w.Code)
	}

	// Verify it was stored
	var count int
	db.QueryRow("SELECT COUNT(*) FROM rmas WHERE serial_number = ?", longSN).Scan(&count)
	if count != 1 {
		t.Errorf("Expected RMA to be created with long serial number")
	}
}

func TestHandleListRMAs_LargeDataset(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert 100 RMAs
	for i := 1; i <= 100; i++ {
		insertTestRMA(t, db, fmt.Sprintf("RMA-%03d", i), fmt.Sprintf("SN%05d", i), "Test Corp", "Test", "open", "", "")
	}

	req := httptest.NewRequest("GET", "/api/rmas", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	handleListRMAs(w, req)
	duration := time.Since(start)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmas := resp.Data.([]interface{})

	if len(rmas) != 100 {
		t.Errorf("Expected 100 RMAs, got %d", len(rmas))
	}

	t.Logf("List 100 RMAs took %v", duration)

	// Performance check: should complete in < 100ms for 100 records
	if duration > 100*time.Millisecond {
		t.Logf("Warning: List operation took %v (>100ms) for 100 records", duration)
	}
}

func TestHandleGetRMA_NonexistentAfterDelete(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "", "")

	// Delete the RMA (simulating manual database cleanup)
	_, err := db.Exec("DELETE FROM rmas WHERE id = ?", "RMA-001")
	if err != nil {
		t.Fatalf("Failed to delete RMA: %v", err)
	}

	// Try to get the deleted RMA
	req := httptest.NewRequest("GET", "/api/rmas/RMA-001", nil)
	w := httptest.NewRecorder()

	handleGetRMA(w, req, "RMA-001")

	if w.Code != 404 {
		t.Errorf("Expected status 404 for deleted RMA, got %d", w.Code)
	}
}

func TestHandleUpdateRMA_EmptyStringFields(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Original reason", "open", "Original defect", "")

	// Update with empty strings (should be allowed for optional fields)
	reqBody := `{
		"serial_number": "SN12345",
		"customer": "",
		"reason": "Updated reason",
		"defect_description": "",
		"resolution": "",
		"status": "received"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200 for empty optional fields, got %d: %s", w.Code, w.Body.String())
	}

	// Verify empty strings were saved
	var customer, defect string
	err := db.QueryRow("SELECT customer, defect_description FROM rmas WHERE id = ?", "RMA-001").
		Scan(&customer, &defect)
	if err != nil {
		t.Fatalf("Failed to query RMA: %v", err)
	}

	if customer != "" {
		t.Errorf("Expected empty customer, got '%s'", customer)
	}
	if defect != "" {
		t.Errorf("Expected empty defect_description, got '%s'", defect)
	}
}

func TestHandleCreateRMA_SpecialCharactersInFields(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test various special characters that might cause issues
	reqBody := `{
		"serial_number": "SN-123/456\\ABC",
		"customer": "O'Reilly & Sons, Inc.",
		"reason": "Device \"broken\" - won't start",
		"defect_description": "Error: [FATAL] System\ncrash\tat boot\r\n"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for special characters, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})

	if !strings.Contains(rmaData["serial_number"].(string), "/") {
		t.Error("Forward slash not preserved in serial number")
	}
	if !strings.Contains(rmaData["customer"].(string), "'") {
		t.Error("Apostrophe not preserved in customer")
	}
	if !strings.Contains(rmaData["reason"].(string), "\"") {
		t.Error("Quote not preserved in reason")
	}
}

// =============================================================================
// STATUS VALIDATION & SCHEMA CONSTRAINT TESTS
// =============================================================================

func TestHandleCreateRMA_DatabaseConstraint_InvalidStatus(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Try to insert directly with invalid status (bypassing validation)
	_, err := db.Exec(`
		INSERT INTO rmas (id, serial_number, reason, status, created_at) 
		VALUES ('RMA-001', 'SN12345', 'Test', 'invalid_status', datetime('now'))
	`)

	// Should fail due to CHECK constraint
	if err == nil {
		t.Error("Expected database constraint to reject invalid status, but insert succeeded")
	}

	if !strings.Contains(err.Error(), "CHECK constraint") {
		t.Logf("Error message: %v", err)
		// SQLite should enforce the CHECK constraint
	}
}

func TestHandleUpdateRMA_AllStatusTransitions(t *testing.T) {
	// Test all possible status transitions to verify workflow
	transitions := []struct {
		from string
		to   string
		ok   bool
	}{
		{"open", "received", true},
		{"open", "diagnosing", true}, // Allow skipping received
		{"open", "closed", true},     // Allow direct close (e.g., duplicate)
		{"received", "diagnosing", true},
		{"received", "scrapped", true},
		{"diagnosing", "repairing", true},
		{"diagnosing", "resolved", true}, // Allow skipping repair if not needed
		{"diagnosing", "scrapped", true},
		{"repairing", "resolved", true},
		{"repairing", "scrapped", true},
		{"resolved", "closed", true},
		{"closed", "open", true}, // Allow reopening (though unusual)
	}

	for _, tt := range transitions {
		t.Run(fmt.Sprintf("%s_to_%s", tt.from, tt.to), func(t *testing.T) {
			oldDB := db
			db = setupRMATestDB(t)
			defer func() { db.Close(); db = oldDB }()

			insertTestRMA(t, db, "RMA-001", "SN12345", "Test Corp", "Test", tt.from, "Test", "")

			reqBody := fmt.Sprintf(`{
				"serial_number": "SN12345",
				"reason": "Test",
				"status": "%s"
			}`, tt.to)
			req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleUpdateRMA(w, req, "RMA-001")

			if tt.ok {
				if w.Code != 200 {
					t.Errorf("Expected transition %s → %s to succeed, got status %d: %s",
						tt.from, tt.to, w.Code, w.Body.String())
				}
			} else {
				if w.Code == 200 {
					t.Errorf("Expected transition %s → %s to fail, but it succeeded", tt.from, tt.to)
				}
			}
		})
	}
}

func TestHandleListRMAs_PerformanceWithComplexData(t *testing.T) {
	oldDB := db
	db = setupRMATestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert RMAs with complex data (long strings, special chars)
	for i := 1; i <= 50; i++ {
		defect := strings.Repeat(fmt.Sprintf("Line %d of detailed defect description. ", i), 25) // Make it long enough
		if len(defect) > 900 {
			defect = defect[:900]
		}
		insertTestRMA(t, db, fmt.Sprintf("RMA-%03d", i), fmt.Sprintf("SN%05d", i),
			"Very Long Customer Name Corp., Inc. & Associates", "Reason "+strings.Repeat("X", 200),
			"received", defect, "")
	}

	req := httptest.NewRequest("GET", "/api/rmas", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	handleListRMAs(w, req)
	duration := time.Since(start)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	t.Logf("List 50 complex RMAs took %v", duration)
}
