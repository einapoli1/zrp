package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// SQL INJECTION SAFETY TESTS
// =============================================================================

func TestSalesOrderSQLInjectionList(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	// Create a normal order
	body := `{"customer":"Normal Corp","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)

	// SQL injection attempts in query params
	injectionTests := []struct {
		name   string
		param  string
		value  string
	}{
		{"status injection", "status", "draft' OR '1'='1"},
		{"customer injection", "customer", "'; DROP TABLE sales_orders; --"},
		{"customer union", "customer", "' UNION SELECT * FROM users --"},
	}

	for _, tc := range injectionTests {
		t.Run(tc.name, func(t *testing.T) {
			urlPath := "/api/v1/sales-orders?" + tc.param + "=" + url.QueryEscape(tc.value)
			req := authedRequest("GET", urlPath, nil, cookie)
			w := httptest.NewRecorder()
			handleListSalesOrders(w, req)
			// Should not crash and should return valid response
			if w.Code != 200 {
				t.Logf("Warning: status %d for %s", w.Code, tc.name)
			}
			// Verify table still exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sales_orders").Scan(&count)
			if err != nil {
				t.Fatalf("table dropped or damaged: %v", err)
			}
		})
	}
}

func TestSalesOrderSQLInjectionCreate(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	injectionTests := []struct {
		name string
		body string
	}{
		{
			"customer field",
			`{"customer":"'; DROP TABLE sales_orders; --","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`,
		},
		{
			"notes field",
			`{"customer":"Test","notes":"'; DELETE FROM sales_orders; --","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`,
		},
		{
			"IPN field",
			`{"customer":"Test","lines":[{"ipn":"'; DROP TABLE inventory; --","qty":1,"unit_price":10}]}`,
		},
	}

	for _, tc := range injectionTests {
		t.Run(tc.name, func(t *testing.T) {
			req := authedRequest("POST", "/api/v1/sales-orders", []byte(tc.body), cookie)
			w := httptest.NewRecorder()
			handleCreateSalesOrder(w, req)
			// Should not crash
			if w.Code >= 500 {
				t.Fatalf("server error: %d %s", w.Code, w.Body.String())
			}
			// Verify table integrity
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sales_orders").Scan(&count)
			if err != nil {
				t.Fatalf("table damaged: %v", err)
			}
		})
	}
}

// =============================================================================
// LINE ITEM VALIDATION EDGE CASES
// =============================================================================

func TestSalesOrderLineValidation(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	validationTests := []struct {
		name       string
		body       string
		shouldFail bool
		errorMsg   string
	}{
		{
			"negative quantity",
			`{"customer":"Test","lines":[{"ipn":"IPN-001","qty":-5,"unit_price":10}]}`,
			true,
			"qty",
		},
		{
			"zero quantity",
			`{"customer":"Test","lines":[{"ipn":"IPN-001","qty":0,"unit_price":10}]}`,
			true,
			"qty",
		},
		{
			"negative unit price",
			`{"customer":"Test","lines":[{"ipn":"IPN-001","qty":10,"unit_price":-25.5}]}`,
			true,
			"unit_price",
		},
		{
			"missing customer",
			`{"lines":[{"ipn":"IPN-001","qty":10,"unit_price":25.5}]}`,
			true,
			"customer",
		},
		{
			"empty lines array",
			`{"customer":"Test","lines":[]}`,
			false,
			"",
		},
		{
			"null lines",
			`{"customer":"Test"}`,
			false,
			"",
		},
		{
			"very large quantity",
			`{"customer":"Test","lines":[{"ipn":"IPN-001","qty":999999999,"unit_price":0.01}]}`,
			false,
			"",
		},
		{
			"multiple lines mixed valid/invalid",
			`{"customer":"Test","lines":[{"ipn":"IPN-001","qty":10,"unit_price":25},{"ipn":"IPN-002","qty":-5,"unit_price":10}]}`,
			true,
			"qty",
		},
	}

	for _, tc := range validationTests {
		t.Run(tc.name, func(t *testing.T) {
			req := authedRequest("POST", "/api/v1/sales-orders", []byte(tc.body), cookie)
			w := httptest.NewRecorder()
			handleCreateSalesOrder(w, req)

			if tc.shouldFail {
				if w.Code == 200 {
					t.Errorf("expected failure but got success: %s", w.Body.String())
				}
				if tc.errorMsg != "" && !strings.Contains(w.Body.String(), tc.errorMsg) {
					t.Errorf("expected error containing '%s', got: %s", tc.errorMsg, w.Body.String())
				}
			} else {
				if w.Code != 200 {
					t.Errorf("expected success but got %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
}

// =============================================================================
// LINE ITEM TOTALS ACCURACY
// =============================================================================

func TestSalesOrderTotalsAccuracy(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "WIDGET-01", 1000, 0, "A1")

	tests := []struct {
		name          string
		lines         string
		expectedTotal float64
	}{
		{
			"simple total",
			`[{"ipn":"WIDGET-01","qty":10,"unit_price":25.50}]`,
			255.00,
		},
		{
			"multiple lines",
			`[{"ipn":"WIDGET-01","qty":10,"unit_price":25.50},{"ipn":"WIDGET-01","qty":5,"unit_price":10.00}]`,
			305.00,
		},
		{
			"fractional prices",
			`[{"ipn":"WIDGET-01","qty":3,"unit_price":33.33}]`,
			99.99,
		},
		{
			"large quantities",
			`[{"ipn":"WIDGET-01","qty":1000,"unit_price":1.99}]`,
			1990.00,
		},
		{
			"zero price",
			`[{"ipn":"WIDGET-01","qty":10,"unit_price":0}]`,
			0.00,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"customer":"Test Corp","lines":%s}`, tc.lines)
			req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
			w := httptest.NewRecorder()
			handleCreateSalesOrder(w, req)
			if w.Code != 200 {
				t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
			}

			so := extractSalesOrder(t, w.Body.Bytes())
			
			// Transition through to invoiced
			transitionOrderToInvoiced(t, so.ID, cookie)

			// Verify invoice total
			var invTotal float64
			err := db.QueryRow("SELECT total FROM invoices WHERE sales_order_id=?", so.ID).Scan(&invTotal)
			if err != nil {
				t.Fatalf("failed to get invoice: %v", err)
			}

			if invTotal != tc.expectedTotal {
				t.Errorf("expected total %.2f, got %.2f", tc.expectedTotal, invTotal)
			}
		})
	}
}

// Helper to transition order to invoiced state
func transitionOrderToInvoiced(t *testing.T, orderID string, cookie string) {
	// Confirm
	req := authedRequest("POST", "/api/v1/sales-orders/"+orderID+"/confirm", nil, cookie)
	w := httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("confirm failed: %d %s", w.Code, w.Body.String())
	}

	// Allocate
	req = authedRequest("POST", "/api/v1/sales-orders/"+orderID+"/allocate", nil, cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("allocate failed: %d %s", w.Code, w.Body.String())
	}

	// Pick
	req = authedRequest("POST", "/api/v1/sales-orders/"+orderID+"/pick", nil, cookie)
	w = httptest.NewRecorder()
	handlePickSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("pick failed: %d %s", w.Code, w.Body.String())
	}

	// Ship
	req = authedRequest("POST", "/api/v1/sales-orders/"+orderID+"/ship", nil, cookie)
	w = httptest.NewRecorder()
	handleShipSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("ship failed: %d %s", w.Code, w.Body.String())
	}

	// Invoice
	req = authedRequest("POST", "/api/v1/sales-orders/"+orderID+"/invoice", nil, cookie)
	w = httptest.NewRecorder()
	handleInvoiceSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("invoice failed: %d %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// STATUS VALIDATION
// =============================================================================

func TestSalesOrderStatusValidation(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	// Create with invalid status
	invalidStatuses := []string{"pending", "cancelled", "invalid", "DRAFT", "Draft"}
	for _, status := range invalidStatuses {
		t.Run("create with "+status, func(t *testing.T) {
			body := fmt.Sprintf(`{"customer":"Test","status":"%s","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`, status)
			req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
			w := httptest.NewRecorder()
			handleCreateSalesOrder(w, req)
			if w.Code != 400 {
				t.Errorf("expected 400 for invalid status %s, got %d", status, w.Code)
			}
		})
	}

	// Create with valid status
	validStatuses := []string{"draft", "confirmed", "allocated", "picked", "shipped", "invoiced", "closed"}
	for _, status := range validStatuses {
		t.Run("create with "+status, func(t *testing.T) {
			body := fmt.Sprintf(`{"customer":"Test","status":"%s","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`, status)
			req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
			w := httptest.NewRecorder()
			handleCreateSalesOrder(w, req)
			if w.Code != 200 {
				t.Errorf("expected 200 for valid status %s, got %d: %s", status, w.Code, w.Body.String())
			}
		})
	}
}

// =============================================================================
// CONCURRENT UPDATE TESTS
// =============================================================================

func TestSalesOrderConcurrentUpdates(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	// Create order
	body := `{"customer":"Concurrent Test","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so := extractSalesOrder(t, w.Body.Bytes())
	id := so.ID

	// Concurrent updates
	var wg sync.WaitGroup
	results := make([]int, 10)
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			updateBody := fmt.Sprintf(`{"customer":"Updated %d","status":"draft","notes":"Update %d"}`, idx, idx)
			req := authedRequest("PUT", "/api/v1/sales-orders/"+id, []byte(updateBody), cookie)
			w := httptest.NewRecorder()
			handleUpdateSalesOrder(w, req, id)
			results[idx] = w.Code
		}(i)
	}
	wg.Wait()

	// Verify all succeeded (no race conditions)
	for i, code := range results {
		if code != 200 {
			t.Errorf("update %d failed with code %d", i, code)
		}
	}

	// Verify final state is consistent
	req = authedRequest("GET", "/api/v1/sales-orders/"+id, nil, cookie)
	w = httptest.NewRecorder()
	handleGetSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("failed to get order: %d", w.Code)
	}
}

func TestSalesOrderConcurrentAllocations(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "WIDGET-01", 100, 0, "A1")

	// Create multiple orders
	orderIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		body := fmt.Sprintf(`{"customer":"Concurrent %d","lines":[{"ipn":"WIDGET-01","qty":15,"unit_price":25}]}`, i)
		req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
		w := httptest.NewRecorder()
		handleCreateSalesOrder(w, req)
		so := extractSalesOrder(t, w.Body.Bytes())
		orderIDs[i] = so.ID
		
		// Confirm each
		req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/confirm", nil, cookie)
		w = httptest.NewRecorder()
		handleConfirmSalesOrder(w, req, so.ID)
	}

	// Concurrently allocate all (total = 75, but only 100 available)
	var wg sync.WaitGroup
	successes := 0
	failures := 0
	var mu sync.Mutex
	
	for _, id := range orderIDs {
		wg.Add(1)
		go func(orderID string) {
			defer wg.Done()
			req := authedRequest("POST", "/api/v1/sales-orders/"+orderID+"/allocate", nil, cookie)
			w := httptest.NewRecorder()
			handleAllocateSalesOrder(w, req, orderID)
			mu.Lock()
			if w.Code == 200 {
				successes++
			} else {
				failures++
			}
			mu.Unlock()
		}(id)
	}
	wg.Wait()

	// Should have some successes and some failures
	t.Logf("Allocations: %d succeeded, %d failed", successes, failures)
	
	// Verify inventory hasn't gone negative
	var qtyReserved float64
	db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn='WIDGET-01'").Scan(&qtyReserved)
	if qtyReserved > 100 {
		t.Errorf("inventory over-reserved: %.0f (max should be 100)", qtyReserved)
	}
	if qtyReserved < 0 {
		t.Errorf("negative reserved inventory: %.0f", qtyReserved)
	}
}

// =============================================================================
// QUOTE CONVERSION EDGE CASES
// =============================================================================

func TestConvertQuoteWithNoLines(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	// Create accepted quote with no lines
	body := `{"customer":"Test","status":"accepted"}`
	req := authedRequest("POST", "/api/v1/quotes", []byte(body), cookie)
	w := httptest.NewRecorder()
	handleCreateQuote(w, req)
	var qResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &qResp)
	qb, _ := json.Marshal(qResp.Data)
	var q Quote
	json.Unmarshal(qb, &q)

	// Convert
	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", nil, cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 200 {
		t.Fatalf("conversion failed: %d %s", w.Code, w.Body.String())
	}

	// Verify created order has no lines
	so := extractSalesOrder(t, w.Body.Bytes())
	if len(so.Lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(so.Lines))
	}
}

// =============================================================================
// INVENTORY RESERVATION TESTS
// =============================================================================

func TestMultipleOrdersInventoryReservation(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)
	
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "WIDGET-01", 100, 0, "A1")

	// Create and allocate first order
	body1 := `{"customer":"Order 1","lines":[{"ipn":"WIDGET-01","qty":30,"unit_price":25}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", []byte(body1), cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so1 := extractSalesOrder(t, w.Body.Bytes())

	req = authedRequest("POST", "/api/v1/sales-orders/"+so1.ID+"/confirm", nil, cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, so1.ID)

	req = authedRequest("POST", "/api/v1/sales-orders/"+so1.ID+"/allocate", nil, cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so1.ID)

	// Create and allocate second order
	body2 := `{"customer":"Order 2","lines":[{"ipn":"WIDGET-01","qty":40,"unit_price":25}]}`
	req = authedRequest("POST", "/api/v1/sales-orders", []byte(body2), cookie)
	w = httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so2 := extractSalesOrder(t, w.Body.Bytes())

	req = authedRequest("POST", "/api/v1/sales-orders/"+so2.ID+"/confirm", nil, cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, so2.ID)

	req = authedRequest("POST", "/api/v1/sales-orders/"+so2.ID+"/allocate", nil, cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so2.ID)

	// Try third order (should fail - only 30 left)
	body3 := `{"customer":"Order 3","lines":[{"ipn":"WIDGET-01","qty":35,"unit_price":25}]}`
	req = authedRequest("POST", "/api/v1/sales-orders", []byte(body3), cookie)
	w = httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so3 := extractSalesOrder(t, w.Body.Bytes())

	req = authedRequest("POST", "/api/v1/sales-orders/"+so3.ID+"/confirm", nil, cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, so3.ID)

	req = authedRequest("POST", "/api/v1/sales-orders/"+so3.ID+"/allocate", nil, cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so3.ID)

	if w.Code == 200 {
		t.Error("should not allow allocation exceeding available inventory")
	}

	// Verify total reservations
	var qtyReserved float64
	db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn='WIDGET-01'").Scan(&qtyReserved)
	if qtyReserved != 70 {
		t.Errorf("expected 70 reserved (30+40), got %.0f", qtyReserved)
	}
}

// =============================================================================
// AUDIT TRAIL TESTS
// =============================================================================

func TestSalesOrderAuditTrail(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "WIDGET-01", 100, 0, "A1")

	// Create order
	body := `{"customer":"Audit Test","lines":[{"ipn":"WIDGET-01","qty":10,"unit_price":25}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so := extractSalesOrder(t, w.Body.Bytes())

	// Go through workflow
	transitionOrderToInvoiced(t, so.ID, cookie)

	// Check audit logs
	rows, err := db.Query("SELECT action FROM audit_log WHERE module='sales_order' AND record_id=? ORDER BY created_at", so.ID)
	if err != nil {
		t.Fatalf("failed to query audit: %v", err)
	}
	defer rows.Close()

	var actions []string
	for rows.Next() {
		var action string
		rows.Scan(&action)
		actions = append(actions, action)
	}

	expectedActions := []string{"created", "confirmed", "allocated", "picked", "shipped", "invoiced"}
	if len(actions) < len(expectedActions) {
		t.Errorf("expected at least %d audit entries, got %d: %v", len(expectedActions), len(actions), actions)
	}
}

// =============================================================================
// TIMESTAMP CONSISTENCY TESTS
// =============================================================================

func TestSalesOrderTimestamps(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	cookie := loginAdmin(t, db)

	// Create order
	body := `{"customer":"Timestamp Test","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", []byte(body), cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so := extractSalesOrder(t, w.Body.Bytes())

	createdAt := so.CreatedAt
	updatedAt := so.UpdatedAt

	if createdAt != updatedAt {
		t.Error("created_at should equal updated_at on creation")
	}

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Update
	updateBody := `{"customer":"Timestamp Updated","status":"draft","notes":"updated"}`
	req = authedRequest("PUT", "/api/v1/sales-orders/"+so.ID, []byte(updateBody), cookie)
	w = httptest.NewRecorder()
	handleUpdateSalesOrder(w, req, so.ID)
	updated := extractSalesOrder(t, w.Body.Bytes())

	if updated.CreatedAt != createdAt {
		t.Error("created_at should not change on update")
	}
	if updated.UpdatedAt == updatedAt {
		t.Error("updated_at should change on update")
	}
}
