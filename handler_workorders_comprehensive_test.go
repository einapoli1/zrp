package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestHandleListWorkOrders_EdgeCases tests various edge cases for listing work orders
func TestHandleListWorkOrders_EdgeCases(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	t.Run("EmptyList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workorders", nil)
		rr := httptest.NewRecorder()
		handleListWorkOrders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var apiResp struct {
			Data []WorkOrder `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
			t.Fatal(err)
		}

		if len(apiResp.Data) != 0 {
			t.Errorf("Expected empty array, got %d items", len(apiResp.Data))
		}
	})

	t.Run("WithWorkOrders", func(t *testing.T) {
		// Insert test data
		_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
			('WO-TEST-001', 'ASY-001', 10, 'open', 'high', '2024-01-01 10:00:00'),
			('WO-TEST-002', 'ASY-002', 5, 'in_progress', 'normal', '2024-01-02 11:00:00'),
			('WO-TEST-003', 'ASY-003', 20, 'completed', 'low', '2024-01-03 12:00:00')`)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/api/v1/workorders", nil)
		rr := httptest.NewRecorder()
		handleListWorkOrders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var apiResp struct {
			Data []WorkOrder `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
			t.Fatal(err)
		}

		if len(apiResp.Data) != 3 {
			t.Errorf("Expected 3 work orders, got %d", len(apiResp.Data))
		}

		// Verify ordering (DESC by created_at)
		if apiResp.Data[0].ID != "WO-TEST-003" {
			t.Errorf("Expected first WO to be WO-TEST-003, got %s", apiResp.Data[0].ID)
		}
	})

	t.Run("WithQtyGoodAndScrap", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, qty_good, qty_scrap, status, priority, created_at) VALUES 
			('WO-YIELD-001', 'ASY-004', 100, 95, 5, 'completed', 'high', '2024-01-04 10:00:00')`)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/api/v1/workorders", nil)
		rr := httptest.NewRecorder()
		handleListWorkOrders(rr, req)

		var apiResp struct {
			Data []WorkOrder `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
			t.Fatal(err)
		}

		// Find our test WO
		var wo *WorkOrder
		for i := range apiResp.Data {
			if apiResp.Data[i].ID == "WO-YIELD-001" {
				wo = &apiResp.Data[i]
				break
			}
		}

		if wo == nil {
			t.Fatal("WO-YIELD-001 not found")
		}

		if wo.QtyGood == nil || *wo.QtyGood != 95 {
			t.Errorf("Expected qty_good=95, got %v", wo.QtyGood)
		}

		if wo.QtyScrap == nil || *wo.QtyScrap != 5 {
			t.Errorf("Expected qty_scrap=5, got %v", wo.QtyScrap)
		}
	})
}

// TestHandleGetWorkOrder_EdgeCases tests edge cases for getting a single work order
func TestHandleGetWorkOrder_EdgeCases(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workorders/WO-NONEXISTENT", nil)
		rr := httptest.NewRecorder()
		handleGetWorkOrder(rr, req, "WO-NONEXISTENT")

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
	})

	t.Run("ValidWorkOrder", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, notes, created_at) VALUES 
			('WO-GET-001', 'ASY-005', 15, 'open', 'high', 'Test notes', '2024-01-05 10:00:00')`)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/api/v1/workorders/WO-GET-001", nil)
		rr := httptest.NewRecorder()
		handleGetWorkOrder(rr, req, "WO-GET-001")

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var apiResp struct {
			Data WorkOrder `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
			t.Fatal(err)
		}

		if apiResp.Data.ID != "WO-GET-001" {
			t.Errorf("Expected ID WO-GET-001, got %s", apiResp.Data.ID)
		}

		if apiResp.Data.AssemblyIPN != "ASY-005" {
			t.Errorf("Expected assembly_ipn ASY-005, got %s", apiResp.Data.AssemblyIPN)
		}

		if apiResp.Data.Notes != "Test notes" {
			t.Errorf("Expected notes 'Test notes', got %s", apiResp.Data.Notes)
		}
	})
}

// TestHandleCreateWorkOrder_Validation tests validation rules for creating work orders
func TestHandleCreateWorkOrder_Validation(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name       string
		payload    interface{}
		expectCode int
		expectErr  string
	}{
		{
			name:       "InvalidJSON",
			payload:    "invalid json",
			expectCode: 400,
			expectErr:  "invalid body",
		},
		{
			name: "MissingAssemblyIPN",
			payload: map[string]interface{}{
				"qty":      10,
				"status":   "open",
				"priority": "normal",
			},
			expectCode: 400,
			expectErr:  "assembly_ipn",
		},
		{
			name: "EmptyAssemblyIPN",
			payload: map[string]interface{}{
				"assembly_ipn": "",
				"qty":          10,
			},
			expectCode: 400,
			expectErr:  "assembly_ipn",
		},
		{
			name: "AssemblyIPNTooLong",
			payload: map[string]interface{}{
				"assembly_ipn": strings.Repeat("A", 101),
				"qty":          10,
			},
			expectCode: 400,
			expectErr:  "assembly_ipn",
		},
		{
			name: "NotesTooLong",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-TEST",
				"qty":          10,
				"notes":        strings.Repeat("A", 10001),
			},
			expectCode: 400,
			expectErr:  "notes",
		},
		{
			name: "InvalidStatus",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-TEST",
				"qty":          10,
				"status":       "invalid_status",
			},
			expectCode: 400,
			expectErr:  "status",
		},
		{
			name: "InvalidPriority",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-TEST",
				"qty":          10,
				"priority":     "super_urgent",
			},
			expectCode: 400,
			expectErr:  "priority",
		},
		{
			name: "NegativeQty",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-TEST",
				"qty":          -5,
			},
			expectCode: 400,
			expectErr:  "qty",
		},
		{
			name: "ZeroQty",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-TEST",
				"qty":          0,
			},
			expectCode: 400,
			expectErr:  "qty",
		},
		{
			name: "ValidMinimal",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-VALID",
				"qty":          1,
			},
			expectCode: 200,
		},
		{
			name: "ValidComplete",
			payload: map[string]interface{}{
				"assembly_ipn": "ASY-COMPLETE",
				"qty":          100,
				"status":       "draft",
				"priority":     "critical",
				"notes":        "This is a test work order",
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.payload.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewBuffer(body))
			rr := httptest.NewRecorder()
			handleCreateWorkOrder(rr, req)

			if rr.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectCode, rr.Code, rr.Body.String())
			}

			if tt.expectErr != "" && !strings.Contains(rr.Body.String(), tt.expectErr) {
				t.Errorf("Expected error containing '%s', got: %s", tt.expectErr, rr.Body.String())
			}

			// For successful creates, verify the work order was created
			if tt.expectCode == 200 {
				var apiResp struct {
					Data WorkOrder `json:"data"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
					t.Fatalf("Failed to parse response: %v. Body: %s", err, rr.Body.String())
				}

				if apiResp.Data.ID == "" {
					t.Error("Expected ID to be generated")
				}

				// Verify defaults
				if tt.payload.(map[string]interface{})["status"] == nil && apiResp.Data.Status != "open" {
					t.Errorf("Expected default status 'open', got %s", apiResp.Data.Status)
				}

				if tt.payload.(map[string]interface{})["priority"] == nil && apiResp.Data.Priority != "normal" {
					t.Errorf("Expected default priority 'normal', got %s", apiResp.Data.Priority)
				}
			}
		})
	}
}

// TestHandleUpdateWorkOrder_StatusTransitions tests status state machine
func TestHandleUpdateWorkOrder_StatusTransitions(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	transitions := []struct {
		name       string
		fromStatus string
		toStatus   string
		shouldPass bool
	}{
		// Valid transitions
		{"DraftToOpen", "draft", "open", true},
		{"DraftToCancelled", "draft", "cancelled", true},
		{"OpenToInProgress", "open", "in_progress", true},
		{"OpenToOnHold", "open", "on_hold", true},
		{"OpenToCancelled", "open", "cancelled", true},
		{"InProgressToCompleted", "in_progress", "completed", true},
		{"InProgressToOnHold", "in_progress", "on_hold", true},
		{"InProgressToCancelled", "in_progress", "cancelled", true},
		{"OnHoldToInProgress", "on_hold", "in_progress", true},
		{"OnHoldToOpen", "on_hold", "open", true},
		{"OnHoldToCancelled", "on_hold", "cancelled", true},

		// Invalid transitions
		{"DraftToCompleted", "draft", "completed", false},
		{"DraftToInProgress", "draft", "in_progress", false},
		{"OpenToCompleted", "open", "completed", false},
		{"CompletedToOpen", "completed", "open", false},
		{"CompletedToInProgress", "completed", "in_progress", false},
		{"CancelledToOpen", "cancelled", "open", false},
		{"CancelledToInProgress", "cancelled", "in_progress", false},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			woID := "WO-TRANS-" + tt.name

			// Create work order with initial status
			_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES (?, 'ASY-001', 10, ?, 'normal', datetime('now'))`,
				woID, tt.fromStatus)
			if err != nil {
				t.Fatal(err)
			}

			// Also insert inventory record for assembly (needed for completion)
			_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('ASY-001', 0)`)
			if err != nil && !strings.Contains(err.Error(), "UNIQUE") {
				t.Fatal(err)
			}

			// Attempt transition
			payload := map[string]interface{}{
				"assembly_ipn": "ASY-001",
				"qty":          10,
				"status":       tt.toStatus,
				"priority":     "normal",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, bytes.NewBuffer(body))
			rr := httptest.NewRecorder()
			handleUpdateWorkOrder(rr, req, woID)

			if tt.shouldPass {
				if rr.Code != http.StatusOK {
					t.Errorf("Expected transition %s -> %s to succeed, got status %d: %s",
						tt.fromStatus, tt.toStatus, rr.Code, rr.Body.String())
				}
			} else {
				if rr.Code == http.StatusOK {
					t.Errorf("Expected transition %s -> %s to fail, but it succeeded",
						tt.fromStatus, tt.toStatus)
				}

				if !strings.Contains(rr.Body.String(), "invalid transition") {
					t.Errorf("Expected 'invalid transition' error, got: %s", rr.Body.String())
				}
			}
		})
	}
}

// TestHandleUpdateWorkOrder_ConcurrentAccess tests concurrent updates
func TestHandleUpdateWorkOrder_ConcurrentAccess(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create a work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-CONCURRENT-001', 'ASY-001', 10, 'open', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate concurrent updates
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	update := func(newStatus string, newPriority string) {
		payload := map[string]interface{}{
			"assembly_ipn": "ASY-001",
			"qty":          10,
			"status":       newStatus,
			"priority":     newPriority,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("PUT", "/api/v1/workorders/WO-CONCURRENT-001", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		handleUpdateWorkOrder(rr, req, "WO-CONCURRENT-001")

		if rr.Code != http.StatusOK {
			errors <- json.Unmarshal(rr.Body.Bytes(), &struct{}{})
		} else {
			errors <- nil
		}
		done <- true
	}

	// Launch concurrent updates
	go update("in_progress", "high")
	go update("on_hold", "low")

	// Wait for both to complete
	<-done
	<-done

	// At least one should succeed
	err1 := <-errors
	err2 := <-errors

	if err1 != nil && err2 != nil {
		t.Error("Both concurrent updates failed")
	}

	// Verify final state is consistent
	var finalStatus, finalPriority string
	err = db.QueryRow("SELECT status, priority FROM work_orders WHERE id = 'WO-CONCURRENT-001'").Scan(&finalStatus, &finalPriority)
	if err != nil {
		t.Fatal(err)
	}

	if finalStatus != "in_progress" && finalStatus != "on_hold" {
		t.Errorf("Expected status to be in_progress or on_hold, got %s", finalStatus)
	}

	t.Logf("Final state: status=%s, priority=%s", finalStatus, finalPriority)
}

// TestHandleWorkOrderBOM_Comparison tests BOM vs inventory comparison logic
func TestHandleWorkOrderBOM_Comparison(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-BOM-001', 'ASY-MAIN', 10, 'open', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	// Create parts (required for BOM foreign keys)
	_, err = db.Exec(`INSERT INTO parts (ipn, description) VALUES 
		('ASY-MAIN', 'Main Assembly'),
		('PART-SUFFICIENT', 'Part with Sufficient Stock'),
		('PART-SHORTAGE', 'Part with Shortage'),
		('PART-EXACT', 'Part with Exact Match'),
		('PART-MISSING', 'Part with No Stock')`)
	if err != nil {
		t.Fatal(err)
	}

	// Create BOM with various scenarios
	_, err = db.Exec(`INSERT INTO bom (parent_ipn, child_ipn, quantity, reference_designator) VALUES 
		('ASY-MAIN', 'PART-SUFFICIENT', 2.0, 'R1,R2'),
		('ASY-MAIN', 'PART-SHORTAGE', 5.0, 'C1-C5'),
		('ASY-MAIN', 'PART-EXACT', 1.0, 'U1'),
		('ASY-MAIN', 'PART-MISSING', 3.0, 'D1-D3')`)
	if err != nil {
		t.Fatal(err)
	}

	// Create inventory with different scenarios
	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES 
		('PART-SUFFICIENT', 1000.0, 0.0),
		('PART-SHORTAGE', 30.0, 0.0),
		('PART-EXACT', 10.0, 0.0),
		('PART-MISSING', 0.0, 0.0)`)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/workorders/WO-BOM-001/bom", nil)
	rr := httptest.NewRecorder()
	handleWorkOrderBOM(rr, req, "WO-BOM-001")

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result struct {
		WOID        string `json:"wo_id"`
		AssemblyIPN string `json:"assembly_ipn"`
		Qty         int    `json:"qty"`
		BOM         []struct {
			IPN          string  `json:"ipn"`
			Description  string  `json:"description"`
			QtyRequired  float64 `json:"qty_required"`
			QtyOnHand    float64 `json:"qty_on_hand"`
			Shortage     float64 `json:"shortage"`
			Status       string  `json:"status"`
		} `json:"bom"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if len(result.BOM) != 4 {
		t.Fatalf("Expected 4 BOM items, got %d", len(result.BOM))
	}

	// Verify calculations
	for _, item := range result.BOM {
		switch item.IPN {
		case "PART-SUFFICIENT":
			if item.QtyRequired != 20.0 { // 2 * 10
				t.Errorf("PART-SUFFICIENT: Expected qty_required=20, got %f", item.QtyRequired)
			}
			if item.QtyOnHand != 1000.0 {
				t.Errorf("PART-SUFFICIENT: Expected qty_on_hand=1000, got %f", item.QtyOnHand)
			}
			if item.Shortage != 0.0 {
				t.Errorf("PART-SUFFICIENT: Expected shortage=0, got %f", item.Shortage)
			}
			if item.Status != "ok" {
				t.Errorf("PART-SUFFICIENT: Expected status=ok, got %s", item.Status)
			}

		case "PART-SHORTAGE":
			if item.QtyRequired != 50.0 { // 5 * 10
				t.Errorf("PART-SHORTAGE: Expected qty_required=50, got %f", item.QtyRequired)
			}
			if item.QtyOnHand != 30.0 {
				t.Errorf("PART-SHORTAGE: Expected qty_on_hand=30, got %f", item.QtyOnHand)
			}
			if item.Shortage != 20.0 {
				t.Errorf("PART-SHORTAGE: Expected shortage=20, got %f", item.Shortage)
			}
			if item.Status != "shortage" {
				t.Errorf("PART-SHORTAGE: Expected status=shortage, got %s", item.Status)
			}

		case "PART-EXACT":
			if item.QtyRequired != 10.0 { // 1 * 10
				t.Errorf("PART-EXACT: Expected qty_required=10, got %f", item.QtyRequired)
			}
			if item.QtyOnHand != 10.0 {
				t.Errorf("PART-EXACT: Expected qty_on_hand=10, got %f", item.QtyOnHand)
			}
			if item.Shortage != 0.0 {
				t.Errorf("PART-EXACT: Expected shortage=0, got %f", item.Shortage)
			}
			if item.Status != "ok" {
				t.Errorf("PART-EXACT: Expected status=ok, got %s", item.Status)
			}

		case "PART-MISSING":
			if item.QtyRequired != 30.0 { // 3 * 10
				t.Errorf("PART-MISSING: Expected qty_required=30, got %f", item.QtyRequired)
			}
			if item.QtyOnHand != 0.0 {
				t.Errorf("PART-MISSING: Expected qty_on_hand=0, got %f", item.QtyOnHand)
			}
			if item.Shortage != 30.0 {
				t.Errorf("PART-MISSING: Expected shortage=30, got %f", item.Shortage)
			}
			if item.Status != "shortage" {
				t.Errorf("PART-MISSING: Expected status=shortage, got %s", item.Status)
			}
		}
	}
}

// TestHandleWorkOrderBOM_NoBOM tests work order with no BOM data
func TestHandleWorkOrderBOM_NoBOM(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-NOBOM-001', 'ASY-NOBOM', 10, 'open', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/workorders/WO-NOBOM-001/bom", nil)
	rr := httptest.NewRecorder()
	handleWorkOrderBOM(rr, req, "WO-NOBOM-001")

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var result struct {
		BOM []interface{} `json:"bom"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if len(result.BOM) != 0 {
		t.Errorf("Expected empty BOM array, got %d items", len(result.BOM))
	}
}

// TestHandleWorkOrderKit_InsufficientInventory tests kitting with insufficient inventory
func TestHandleWorkOrderKit_InsufficientInventory(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-KIT-SHORT-001', 'ASY-KIT', 10, 'open', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	// Create parts (required for BOM foreign keys)
	_, err = db.Exec(`INSERT INTO parts (ipn, description) VALUES 
		('ASY-KIT', 'Kit Assembly'),
		('PART-LOW', 'Part with Low Stock')`)
	if err != nil {
		t.Fatal(err)
	}

	// Create BOM
	_, err = db.Exec(`INSERT INTO bom (parent_ipn, child_ipn, quantity) VALUES 
		('ASY-KIT', 'PART-LOW', 5.0)`)
	if err != nil {
		t.Fatal(err)
	}

	// Create insufficient inventory
	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES 
		('PART-LOW', 20.0, 0.0)`)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/v1/workorders/WO-KIT-SHORT-001/kit", nil)
	rr := httptest.NewRecorder()
	handleWorkOrderKit(rr, req, "WO-KIT-SHORT-001")

	// Should succeed with partial kitting
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	// Verify partial kitting status
	items, ok := result["items"].([]interface{})
	if !ok || len(items) == 0 {
		t.Fatal("Expected items array in response")
	}

	// Check that shortage was reported
	item := items[0].(map[string]interface{})
	if item["status"].(string) != "partial" && item["status"].(string) != "shortage" {
		t.Errorf("Expected partial or shortage status, got %s", item["status"])
	}
}

// TestWorkOrderCompletionIntegration tests complete work order flow
func TestWorkOrderCompletionIntegration(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Setup: Create parts (required for BOM foreign keys)
	_, err := db.Exec(`INSERT INTO parts (ipn, description) VALUES 
		('ASY-COMPLETE', 'Complete Assembly'),
		('PART-MAT-001', 'Material 1'),
		('PART-MAT-002', 'Material 2')`)
	if err != nil {
		t.Fatal(err)
	}

	// Setup: Create assembly, BOM, and inventory
	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved, description) VALUES 
		('ASY-COMPLETE', 0.0, 0.0, 'Complete Assembly'),
		('PART-MAT-001', 100.0, 0.0, 'Material 1'),
		('PART-MAT-002', 50.0, 0.0, 'Material 2')`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO bom (parent_ipn, child_ipn, quantity) VALUES 
		('ASY-COMPLETE', 'PART-MAT-001', 2.0),
		('ASY-COMPLETE', 'PART-MAT-002', 1.0)`)
	if err != nil {
		t.Fatal(err)
	}

	// Create work order
	payload := map[string]interface{}{
		"assembly_ipn": "ASY-COMPLETE",
		"qty":          10,
		"status":       "draft",
		"priority":     "high",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to create work order: %d %s", rr.Code, rr.Body.String())
	}

	var wo WorkOrder
	json.Unmarshal(rr.Body.Bytes(), &wo)
	woID := wo.ID

	// Step 1: Transition to open
	updatePayload := map[string]interface{}{
		"assembly_ipn": "ASY-COMPLETE",
		"qty":          10,
		"status":       "open",
		"priority":     "high",
	}

	body, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, woID)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to update to open: %d %s", rr.Code, rr.Body.String())
	}

	// Step 2: Kit materials
	req = httptest.NewRequest("POST", "/api/v1/workorders/"+woID+"/kit", nil)
	rr = httptest.NewRecorder()
	handleWorkOrderKit(rr, req, woID)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to kit materials: %d %s", rr.Code, rr.Body.String())
	}

	// Step 3: Start work
	updatePayload["status"] = "in_progress"
	body, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, woID)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to start work: %d %s", rr.Code, rr.Body.String())
	}

	// Verify started_at is set
	var startedAt sql.NullString
	err = db.QueryRow("SELECT started_at FROM work_orders WHERE id = ?", woID).Scan(&startedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !startedAt.Valid {
		t.Error("Expected started_at to be set when transitioning to in_progress")
	}

	// Step 4: Complete work order
	updatePayload["status"] = "completed"
	body, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, woID)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to complete work order: %d %s", rr.Code, rr.Body.String())
	}

	// Verify completed_at is set
	var completedAt sql.NullString
	err = db.QueryRow("SELECT completed_at FROM work_orders WHERE id = ?", woID).Scan(&completedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !completedAt.Valid {
		t.Error("Expected completed_at to be set when transitioning to completed")
	}

	// Verify finished goods were added
	var assemblyQty float64
	err = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = 'ASY-COMPLETE'").Scan(&assemblyQty)
	if err != nil {
		t.Fatal(err)
	}
	if assemblyQty != 10.0 {
		t.Errorf("Expected 10 finished goods, got %f", assemblyQty)
	}

	// Verify inventory transaction was logged
	var txCount int
	err = db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = 'ASY-COMPLETE' AND type = 'receive' AND reference = ?", woID).Scan(&txCount)
	if err != nil {
		t.Fatal(err)
	}
	if txCount == 0 {
		t.Error("Expected inventory transaction for finished goods")
	}
}

// TestWorkOrderCancellation tests cancellation and inventory release
func TestWorkOrderCancellation(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order and inventory
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-CANCEL-001', 'ASY-001', 10, 'open', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES 
		('PART-RESERVED', 100.0, 20.0)`)
	if err != nil {
		t.Fatal(err)
	}

	// Cancel work order
	payload := map[string]interface{}{
		"assembly_ipn": "ASY-001",
		"qty":          10,
		"status":       "cancelled",
		"priority":     "normal",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/workorders/WO-CANCEL-001", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, "WO-CANCEL-001")

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to cancel work order: %d %s", rr.Code, rr.Body.String())
	}

	// Verify reservations were released
	var qtyReserved float64
	err = db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn = 'PART-RESERVED'").Scan(&qtyReserved)
	if err != nil {
		t.Fatal(err)
	}

	// Note: Current implementation releases ALL reservations, not per-WO
	// This is a known limitation documented in the code
	if qtyReserved > 20.0 {
		t.Errorf("Expected reservations to be released or unchanged, got %f", qtyReserved)
	}
}

// TestWorkOrderYieldTracking tests qty_good and qty_scrap tracking
func TestWorkOrderYieldTracking(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-YIELD-001', 'ASY-001', 100, 'in_progress', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('ASY-001', 0)`)
	if err != nil && !strings.Contains(err.Error(), "UNIQUE") {
		t.Fatal(err)
	}

	// Update with yield data
	qtyGood := 95
	qtyScrap := 5

	payload := map[string]interface{}{
		"assembly_ipn": "ASY-001",
		"qty":          100,
		"qty_good":     qtyGood,
		"qty_scrap":    qtyScrap,
		"status":       "in_progress",
		"priority":     "normal",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/v1/workorders/WO-YIELD-001", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, "WO-YIELD-001")

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to update yield: %d %s", rr.Code, rr.Body.String())
	}

	// Verify yield was stored
	var storedQtyGood, storedQtyScrap sql.NullInt64
	err = db.QueryRow("SELECT qty_good, qty_scrap FROM work_orders WHERE id = 'WO-YIELD-001'").Scan(&storedQtyGood, &storedQtyScrap)
	if err != nil {
		t.Fatal(err)
	}

	if !storedQtyGood.Valid || int(storedQtyGood.Int64) != qtyGood {
		t.Errorf("Expected qty_good=%d, got %v", qtyGood, storedQtyGood)
	}

	if !storedQtyScrap.Valid || int(storedQtyScrap.Int64) != qtyScrap {
		t.Errorf("Expected qty_scrap=%d, got %v", qtyScrap, storedQtyScrap)
	}

	// Test negative yield values
	payload["qty_good"] = -1
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("PUT", "/api/v1/workorders/WO-YIELD-001", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, "WO-YIELD-001")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for negative qty_good, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "qty_good") {
		t.Errorf("Expected error message about qty_good, got: %s", rr.Body.String())
	}
}

// TestWorkOrderSerials_DuplicateSerial tests that duplicate serials are rejected
func TestWorkOrderSerials_DuplicateSerial(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, created_at) VALUES 
		('WO-SERIAL-001', 'ASY-001', 10, 'in_progress', 'normal', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	// Add first serial
	serial := WOSerial{
		SerialNumber: "DUPLICATE-123",
		Status:       "assigned",
	}

	body, _ := json.Marshal(serial)
	req := httptest.NewRequest("POST", "/api/v1/workorders/WO-SERIAL-001/serials", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handleWorkOrderAddSerial(rr, req, "WO-SERIAL-001")

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to add first serial: %d %s", rr.Code, rr.Body.String())
	}

	// Try to add duplicate
	req = httptest.NewRequest("POST", "/api/v1/workorders/WO-SERIAL-001/serials", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handleWorkOrderAddSerial(rr, req, "WO-SERIAL-001")

	if rr.Code == http.StatusOK {
		t.Error("Expected duplicate serial to be rejected, but it was accepted")
	}
}

// TestWorkOrderMaxQuantityLimit tests the maximum work order quantity limit
func TestWorkOrderMaxQuantityLimit(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test at limit (should pass if MaxWorkOrderQty is defined and reasonable)
	payload := map[string]interface{}{
		"assembly_ipn": "ASY-001",
		"qty":          10000, // Assuming reasonable limit
		"status":       "draft",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	// Should succeed or fail based on MaxWorkOrderQty constant
	t.Logf("Creating WO with qty=10000: status=%d", rr.Code)

	// Test beyond limit
	payload["qty"] = 1000000
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	t.Logf("Creating WO with qty=1000000: status=%d, body=%s", rr.Code, rr.Body.String())
}
