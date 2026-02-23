package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWorkOrderValidation_RequiredFields tests all required field validation
func TestWorkOrderValidation_RequiredFields(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name     string
		payload  string
		wantCode int
		wantErr  string
	}{
		{
			name:     "Missing assembly_ipn",
			payload:  `{"qty":10}`,
			wantCode: 400,
			wantErr:  "assembly_ipn",
		},
		{
			name:     "Empty assembly_ipn",
			payload:  `{"assembly_ipn":"","qty":10}`,
			wantCode: 400,
			wantErr:  "assembly_ipn",
		},
		{
			name:     "Whitespace-only assembly_ipn",
			payload:  `{"assembly_ipn":"   ","qty":10}`,
			wantCode: 400,
			wantErr:  "assembly_ipn",
		},
		{
			name:     "Assembly IPN too long (>100 chars)",
			payload:  `{"assembly_ipn":"` + strings.Repeat("A", 101) + `","qty":10}`,
			wantCode: 400,
			wantErr:  "assembly_ipn",
		},
		{
			name:     "Valid minimal payload",
			payload:  `{"assembly_ipn":"ASY-001","qty":1}`,
			wantCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handleCreateWorkOrder(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.wantCode, rr.Code, rr.Body.String())
			}

			if tt.wantErr != "" && !strings.Contains(rr.Body.String(), tt.wantErr) {
				t.Errorf("Expected error to mention %q, got: %s", tt.wantErr, rr.Body.String())
			}
		})
	}
}

// TestWorkOrderValidation_SpecialCharacters tests handling of special characters
func TestWorkOrderValidation_SpecialCharacters(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		assemblyIPN string
		notes       string
		shouldPass  bool
	}{
		{
			name:        "Hyphen and underscore in IPN",
			assemblyIPN: "ASY-TEST_001",
			notes:       "Normal notes",
			shouldPass:  true,
		},
		{
			name:        "Dots in IPN",
			assemblyIPN: "ASY.V1.0",
			notes:       "Version 1.0",
			shouldPass:  true,
		},
		{
			name:        "HTML in notes (should be escaped)",
			assemblyIPN: "ASY-HTML",
			notes:       "<script>alert('xss')</script>",
			shouldPass:  true, // Should accept but escape
		},
		{
			name:        "Newlines in notes",
			assemblyIPN: "ASY-NEWLINE",
			notes:       "Line 1\nLine 2\nLine 3",
			shouldPass:  true,
		},
		{
			name:        "Unicode in notes",
			assemblyIPN: "ASY-UNICODE",
			notes:       "测试 🚀 Тест",
			shouldPass:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"assembly_ipn": tt.assemblyIPN,
				"qty":          1,
				"notes":        tt.notes,
			}
			payloadJSON, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(string(payloadJSON)))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handleCreateWorkOrder(rr, req)

			if tt.shouldPass && rr.Code != 200 {
				t.Errorf("Expected success, got %d: %s", rr.Code, rr.Body.String())
			} else if !tt.shouldPass && rr.Code == 200 {
				t.Errorf("Expected failure, got success")
			}

			// If HTML in notes, verify it's escaped on retrieval
			if strings.Contains(tt.notes, "<script>") && rr.Code == 200 {
				var apiResp struct {
					Data WorkOrder `json:"data"`
				}
				json.Unmarshal(rr.Body.Bytes(), &apiResp)
				
				// Verify the WO was created
				if apiResp.Data.ID == "" {
					t.Error("Work order not created")
					return
				}
				
				// Get the work order and check notes are escaped
				req2 := httptest.NewRequest("GET", "/api/v1/workorders/"+apiResp.Data.ID, nil)
				rr2 := httptest.NewRecorder()
				handleGetWorkOrder(rr2, req2, apiResp.Data.ID)
				
				var getResp struct {
					Data WorkOrder `json:"data"`
				}
				json.Unmarshal(rr2.Body.Bytes(), &getResp)
				
				if strings.Contains(getResp.Data.Notes, "<script>") {
					t.Error("XSS detected: script tags not escaped in notes")
				}
			}
		})
	}
}

// TestWorkOrderValidation_QuantityEdgeCases tests qty_good and qty_scrap validation
func TestWorkOrderValidation_QuantityEdgeCases(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create a work order first
	payload := `{"assembly_ipn":"ASY-QTY-TEST","qty":100,"status":"in_progress"}`
	req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	if rr.Code != 200 {
		t.Fatalf("Failed to create test WO: %s", rr.Body.String())
	}

	var createResp struct {
		Data WorkOrder `json:"data"`
	}
	json.Unmarshal(rr.Body.Bytes(), &createResp)
	woID := createResp.Data.ID

	tests := []struct {
		name      string
		qtyGood   *int
		qtyScrap  *int
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "Negative qty_good",
			qtyGood:   intPtr(-5),
			shouldErr: true,
			errMsg:    "qty_good",
		},
		{
			name:      "Negative qty_scrap",
			qtyScrap:  intPtr(-3),
			shouldErr: true,
			errMsg:    "qty_scrap",
		},
		{
			name:      "Valid yield: 95 good, 5 scrap = 100 total",
			qtyGood:   intPtr(95),
			qtyScrap:  intPtr(5),
			shouldErr: false,
		},
		{
			name:      "Overage: good + scrap > qty (allowed)",
			qtyGood:   intPtr(60),
			qtyScrap:  intPtr(50),
			shouldErr: false, // Currently no constraint, but worth noting
		},
		{
			name:      "Zero good, all scrap",
			qtyGood:   intPtr(0),
			qtyScrap:  intPtr(100),
			shouldErr: false,
		},
		{
			name:      "All good, zero scrap",
			qtyGood:   intPtr(100),
			qtyScrap:  intPtr(0),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatePayload := map[string]interface{}{
				"assembly_ipn": "ASY-QTY-TEST",
				"qty":          100,
				"status":       "in_progress",
			}
			if tt.qtyGood != nil {
				updatePayload["qty_good"] = *tt.qtyGood
			}
			if tt.qtyScrap != nil {
				updatePayload["qty_scrap"] = *tt.qtyScrap
			}

			payloadJSON, _ := json.Marshal(updatePayload)
			req := httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, strings.NewReader(string(payloadJSON)))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handleUpdateWorkOrder(rr, req, woID)

			if tt.shouldErr {
				if rr.Code == 200 {
					t.Errorf("Expected error, but got success")
				}
				if !strings.Contains(rr.Body.String(), tt.errMsg) {
					t.Errorf("Expected error to mention %q, got: %s", tt.errMsg, rr.Body.String())
				}
			} else {
				if rr.Code != 200 {
					t.Errorf("Expected success, got %d: %s", rr.Code, rr.Body.String())
				}
			}
		})
	}
}

// TestWorkOrderStatusTransitions_OnHoldToOpen tests missing transition
func TestWorkOrderStatusTransitions_OnHoldToOpen(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create WO in on_hold status
	payload := `{"assembly_ipn":"ASY-HOLD","qty":10,"status":"open"}`
	req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	var createResp struct {
		Data WorkOrder `json:"data"`
	}
	json.Unmarshal(rr.Body.Bytes(), &createResp)
	woID := createResp.Data.ID

	// Transition to on_hold
	updatePayload := `{"assembly_ipn":"ASY-HOLD","qty":10,"status":"on_hold"}`
	req = httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handleUpdateWorkOrder(rr, req, woID)

	if rr.Code != 200 {
		t.Fatalf("Failed to set on_hold: %s", rr.Body.String())
	}

	// Test on_hold → open transition (should be valid)
	updatePayload2 := `{"assembly_ipn":"ASY-HOLD","qty":10,"status":"open"}`
	req2 := httptest.NewRequest("PUT", "/api/v1/workorders/"+woID, strings.NewReader(updatePayload2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handleUpdateWorkOrder(rr2, req2, woID)

	if rr2.Code != 200 {
		t.Errorf("on_hold → open transition should be valid, got: %s", rr2.Body.String())
	}

	var updateResp struct {
		Data WorkOrder `json:"data"`
	}
	json.Unmarshal(rr2.Body.Bytes(), &updateResp)

	if updateResp.Data.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", updateResp.Data.Status)
	}
}

// TestWorkOrderBOM_EdgeCases tests BOM comparison edge cases
func TestWorkOrderBOM_EdgeCases(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create work order
	payload := `{"assembly_ipn":"ASY-BOM-TEST","qty":10}`
	req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	var createResp struct {
		Data WorkOrder `json:"data"`
	}
	json.Unmarshal(rr.Body.Bytes(), &createResp)
	woID := createResp.Data.ID

	t.Run("No BOM entries", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workorders/"+woID+"/bom", nil)
		rr := httptest.NewRecorder()
		handleWorkOrderBOM(rr, req, woID)

		if rr.Code != 200 {
			t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var apiResp struct {
			Data interface{} `json:"data"`
		}
		json.Unmarshal(rr.Body.Bytes(), &apiResp)
		// Should return empty array, not error
	})

	t.Run("Inventory with NULL qty", func(t *testing.T) {
		// Insert BOM and inventory with NULL
		_, err := db.Exec(`INSERT INTO bom (parent_ipn, child_ipn, quantity, reference_designator) VALUES ('ASY-BOM-TEST', 'PART-NULL', 2, 'U1')`)
		if err != nil {
			t.Fatalf("Failed to insert BOM: %v", err)
		}

		_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('PART-NULL', NULL)`)
		if err != nil {
			// SQLite might not allow NULL on qty_on_hand if there's a NOT NULL constraint
			t.Logf("Cannot insert NULL qty (expected due to schema constraints)")
		}

		req := httptest.NewRequest("GET", "/api/v1/workorders/"+woID+"/bom", nil)
		rr := httptest.NewRecorder()
		handleWorkOrderBOM(rr, req, woID)

		if rr.Code != 200 {
			t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("BOM with zero required qty", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO bom (parent_ipn, child_ipn, quantity, reference_designator) VALUES ('ASY-BOM-TEST', 'PART-ZERO', 0, 'DNP1')`)
		if err != nil {
			t.Fatalf("Failed to insert BOM: %v", err)
		}

		_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, description) VALUES ('PART-ZERO', 100, 'Do Not Populate')`)
		if err != nil {
			t.Fatalf("Failed to insert inventory: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/workorders/"+woID+"/bom", nil)
		rr := httptest.NewRecorder()
		handleWorkOrderBOM(rr, req, woID)

		if rr.Code != 200 {
			t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		// Zero-qty BOM items should still appear in comparison
	})
}

func intPtr(i int) *int {
	return &i
}
