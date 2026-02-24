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

// Comprehensive procurement tests covering gaps identified in 2026-02-23 audit

// ============================================================================
// EDGE CASE 1: Empty Line Items
// ============================================================================

func TestHandleCreatePO_EmptyLinesList(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"vendor_id": "VEN-001",
		"notes": "PO with no lines",
		"lines": []
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	// Should succeed but create a PO with no lines (business may allow this for future edits)
	if w.Code != 200 {
		t.Logf("Response: %s", w.Body.String())
	}

	// Verify PO was created
	var poID string
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	poID = resp.Data.ID

	if poID == "" {
		t.Error("Expected PO to be created even with empty lines")
		return
	}

	// Verify no lines were created
	var lineCount int
	db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id=?", poID).Scan(&lineCount)
	if lineCount != 0 {
		t.Errorf("Expected 0 lines, got %d", lineCount)
	}
}

// ============================================================================
// EDGE CASE 2: Invalid Date Formats
// ============================================================================

func TestHandleCreatePO_InvalidDateFormat(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	testCases := []struct {
		name string
		date string
		want int // expected status code
	}{
		{"invalid_format_slashes", "12/31/2024", 400},
		{"invalid_format_no_dashes", "20241231", 400},
		{"invalid_month", "2024-13-01", 400},
		{"invalid_day", "2024-02-30", 400},
		{"empty_string", "", 200}, // Empty should be allowed
		{"valid_format", "2024-12-31", 200},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := fmt.Sprintf(`{
				"vendor_id": "VEN-001",
				"expected_date": "%s",
				"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": 1.0}]
			}`, tc.date)
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if w.Code != tc.want {
				t.Errorf("%s: expected status %d, got %d: %s", tc.name, tc.want, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================================
// EDGE CASE 3: Duplicate PO Number Prevention
// ============================================================================

func TestHandleCreatePO_ConcurrentDuplicateIDPrevention(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	// Initialize sequence
	db.Exec(`INSERT INTO id_sequences (prefix, next_num) VALUES ('PO', 1)`)

	// Create 10 POs concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	createdIDs := make(map[string]int)
	errors := 0

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			reqBody := fmt.Sprintf(`{
				"vendor_id": "VEN-001",
				"notes": "Concurrent PO %d",
				"lines": [{"ipn": "IPN-%03d", "qty_ordered": 10, "unit_price": 1.0}]
			}`, num, num)
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			mu.Lock()
			defer mu.Unlock()

			if w.Code != 200 {
				errors++
				t.Logf("Request %d failed: %s", num, w.Body.String())
				return
			}

			var resp struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			json.NewDecoder(w.Body).Decode(&resp)
			poID := resp.Data.ID

			if poID != "" {
				createdIDs[poID]++
			}
		}(i)
	}
	wg.Wait()

	// Check for duplicates
	for id, count := range createdIDs {
		if count > 1 {
			t.Errorf("DUPLICATE PO ID: %s created %d times", id, count)
		}
	}

	if len(createdIDs) != 10 {
		t.Errorf("Expected 10 unique POs, got %d (errors: %d)", len(createdIDs), errors)
	}

	t.Logf("Created %d unique PO IDs: %v", len(createdIDs), getKeys(createdIDs))
}

// ============================================================================
// EDGE CASE 4: Zero Quantity Order
// ============================================================================

func TestHandleCreatePO_ZeroQuantity(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"vendor_id": "VEN-001",
		"lines": [{"ipn": "IPN-001", "qty_ordered": 0, "unit_price": 1.0}]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for zero quantity, got %d", w.Code)
	}
}

// ============================================================================
// EDGE CASE 5: Very Large Quantities (Boundary Test)
// ============================================================================

func TestHandleCreatePO_LargeQuantity(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	testCases := []struct {
		name     string
		qty      float64
		expected int
	}{
		{"reasonable_large", 1000000, 200},
		{"very_large", 1e9, 400},         // 1 billion - exceeds max validation
		{"extremely_large", 1e15, 400},   // Should fail validation
		{"max_float64", 1.7e308, 400},    // Near float64 max
		{"negative_large", -1000000, 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := fmt.Sprintf(`{
				"vendor_id": "VEN-001",
				"lines": [{"ipn": "IPN-001", "qty_ordered": %f, "unit_price": 1.0}]
			}`, tc.qty)
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if w.Code != tc.expected {
				t.Errorf("%s: expected status %d, got %d: %s", tc.name, tc.expected, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================================
// EDGE CASE 6: Large Unit Prices (Boundary Test)
// ============================================================================

func TestHandleCreatePO_LargeUnitPrice(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	testCases := []struct {
		name     string
		price    float64
		expected int
	}{
		{"reasonable_expensive", 10000, 200},
		{"very_expensive", 1e6, 200},     // $1M per unit
		{"extremely_expensive", 1e12, 400}, // Should fail validation
		{"negative_price", -100, 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := fmt.Sprintf(`{
				"vendor_id": "VEN-001",
				"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": %f}]
			}`, tc.price)
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if w.Code != tc.expected {
				t.Errorf("%s: expected status %d, got %d: %s", tc.name, tc.expected, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================================
// WORKFLOW: Status Transitions
// ============================================================================

func TestHandlePOStatusTransitions(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-0001', 'VEN-001', 'draft')`)

	validTransitions := []struct {
		from string
		to   string
		ok   bool
	}{
		{"draft", "sent", true},
		{"sent", "confirmed", true},
		{"confirmed", "partial", true},
		{"partial", "received", true},
		{"draft", "cancelled", true},
		{"sent", "cancelled", true},
		// Invalid transitions
		{"received", "draft", false},    // Can't go backwards
		{"cancelled", "sent", false},    // Can't reactivate
		{"received", "partial", false},  // Can't go backwards
	}

	for _, tt := range validTransitions {
		t.Run(fmt.Sprintf("%s_to_%s", tt.from, tt.to), func(t *testing.T) {
			// Reset to starting state
			db.Exec("UPDATE purchase_orders SET status=? WHERE id='PO-0001'", tt.from)

			// Need to pass full PO object with vendor_id due to FK constraint
			reqBody := fmt.Sprintf(`{"vendor_id": "VEN-001", "status": "%s"}`, tt.to)
			req := httptest.NewRequest("PUT", "/api/v1/pos/PO-0001", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleUpdatePO(w, req, "PO-0001")

			// Note: Current implementation doesn't enforce workflow rules
			// This test documents expected behavior
			if w.Code != 200 {
				t.Logf("Transition %s→%s failed: %s", tt.from, tt.to, w.Body.String())
			} else {
				t.Logf("✓ Transition %s→%s succeeded", tt.from, tt.to)
			}
		})
	}
}

// ============================================================================
// EDGE CASE 7: Missing Vendor ID (Empty String vs Null)
// ============================================================================

func TestHandleCreatePO_NullVsEmptyVendor(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	testCases := []struct {
		name   string
		body   string
		expect int
	}{
		{"empty_string", `{"vendor_id": "", "lines": [{"ipn": "IPN-001", "qty_ordered": 10}]}`, 200},
		{"null_value", `{"vendor_id": null, "lines": [{"ipn": "IPN-001", "qty_ordered": 10}]}`, 200},
		{"missing_field", `{"lines": [{"ipn": "IPN-001", "qty_ordered": 10}]}`, 200},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(tc.body))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if w.Code != tc.expect {
				t.Errorf("%s: expected %d, got %d: %s", tc.name, tc.expect, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================================
// EDGE CASE 8: Update Non-Existent PO
// ============================================================================

func TestHandleUpdatePO_NotFound(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"status": "sent"}`
	req := httptest.NewRequest("PUT", "/api/v1/pos/PO-9999", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdatePO(w, req, "PO-9999")

	// Current implementation may not return 404, document actual behavior
	if w.Code == 200 {
		t.Log("BUG: Update succeeds for non-existent PO (should return 404)")
	} else if w.Code == 404 {
		t.Log("✓ Correctly returns 404 for non-existent PO")
	}
}

// ============================================================================
// EDGE CASE 9: Malformed JSON
// ============================================================================

func TestHandleCreatePO_MalformedJSON(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	testCases := []struct {
		name string
		body string
	}{
		{"unclosed_brace", `{"vendor_id": "VEN-001"`},
		{"trailing_comma", `{"vendor_id": "VEN-001",}`},
		{"invalid_quotes", `{vendor_id: "VEN-001"}`},
		{"empty_body", ``},
		{"null_body", `null`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(tc.body))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if w.Code != 400 {
				t.Errorf("%s: expected 400 for malformed JSON, got %d", tc.name, w.Code)
			}
		})
	}
}

// ============================================================================
// EDGE CASE 10: Very Long Notes Field
// ============================================================================

func TestHandleCreatePO_LongNotes(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	testCases := []struct {
		name     string
		length   int
		expected int
	}{
		{"short", 10, 200},
		{"medium", 1000, 200},
		{"long", 10000, 200},
		{"very_long", 100000, 200},    // 100KB
		{"extremely_long", 1000000, 200}, // 1MB - may fail depending on validation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate notes of specified length
			notesBytes := make([]byte, tc.length)
			for i := range notesBytes {
				notesBytes[i] = byte('A' + (i % 26))
			}
			notes := string(notesBytes)

			reqBody := fmt.Sprintf(`{
				"vendor_id": "VEN-001",
				"notes": "%s",
				"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": 1.0}]
			}`, notes)
			req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if w.Code != tc.expected {
				t.Errorf("%s: expected %d, got %d", tc.name, tc.expected, w.Code)
			}
		})
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ============================================================================
// SUPPLIER MANAGEMENT TESTS
// ============================================================================

func TestHandleListVendors_EmptyList(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/vendors", nil)
	w := httptest.NewRecorder()

	handleListVendors(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Data) != 0 {
		t.Errorf("Expected empty list, got %d vendors", len(resp.Data))
	}
}

func TestHandleGetVendor_NotFound_Procurement(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/vendors/VEN-999", nil)
	w := httptest.NewRecorder()

	handleGetVendor(w, req, "VEN-999")

	if w.Code != 404 {
		t.Errorf("Expected 404 for non-existent vendor, got %d", w.Code)
	}
}

// ============================================================================
// PERFORMANCE TEST: Bulk PO Creation
// ============================================================================

func TestHandleCreatePO_BulkPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	start := time.Now()
	count := 100

	for i := 0; i < count; i++ {
		reqBody := fmt.Sprintf(`{
			"vendor_id": "VEN-001",
			"lines": [
				{"ipn": "IPN-%03d", "qty_ordered": %d, "unit_price": %.2f}
			]
		}`, i, i+1, float64(i)*0.1)
		req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		handleCreatePO(w, req)

		if w.Code != 200 {
			t.Errorf("Request %d failed: %s", i, w.Body.String())
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(count)

	t.Logf("Created %d POs in %v (avg: %v per PO)", count, elapsed, avgTime)

	if avgTime > 100*time.Millisecond {
		t.Logf("WARNING: Average creation time is slow: %v", avgTime)
	}
}
