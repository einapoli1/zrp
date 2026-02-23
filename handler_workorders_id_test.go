package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestWorkOrderIDGeneration_Concurrent tests that concurrent work order creation
// generates unique, sequential IDs without conflicts or duplicates
func TestWorkOrderIDGeneration_Concurrent(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Number of concurrent requests to create
	numRequests := 50
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Channel to collect created work order IDs
	idChan := make(chan string, numRequests)
	errorChan := make(chan error, numRequests)

	// Launch concurrent work order creation requests
	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()

			payload := `{"assembly_ipn":"ASY-CONCURRENT-TEST","qty":1}`
			req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handleCreateWorkOrder(rr, req)

			if rr.Code != 200 {
				errorChan <- &testError{msg: "Failed to create WO", code: rr.Code, body: rr.Body.String()}
				return
			}

			var apiResp struct {
				Data WorkOrder `json:"data"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
				errorChan <- err
				return
			}

			idChan <- apiResp.Data.ID
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(idChan)
	close(errorChan)

	// Check for errors
	if len(errorChan) > 0 {
		err := <-errorChan
		t.Fatalf("Concurrent creation failed: %v", err)
	}

	// Collect all IDs
	ids := make(map[string]bool)
	var idList []string
	for id := range idChan {
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
		idList = append(idList, id)
	}

	// Verify we got the expected number of unique IDs
	if len(ids) != numRequests {
		t.Errorf("Expected %d unique IDs, got %d", numRequests, len(ids))
	}

	// Verify all IDs follow the correct format (WO-YYYY-NNNN)
	for _, id := range idList {
		if !strings.HasPrefix(id, "WO-") {
			t.Errorf("ID %s does not have correct prefix", id)
		}
		if len(id) != 13 { // WO-YYYY-NNNN = 13 chars
			t.Errorf("ID %s has incorrect length %d, expected 13", id, len(id))
		}
	}

	t.Logf("Successfully created %d work orders with unique IDs", len(ids))
}

// TestWorkOrderIDGeneration_Sequential verifies that IDs increment properly
func TestWorkOrderIDGeneration_Sequential(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create 5 work orders sequentially
	var ids []string
	for i := 0; i < 5; i++ {
		payload := `{"assembly_ipn":"ASY-SEQ-TEST","qty":1}`
		req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handleCreateWorkOrder(rr, req)

		if rr.Code != 200 {
			t.Fatalf("Failed to create WO #%d: %d %s", i+1, rr.Code, rr.Body.String())
		}

		var apiResp struct {
			Data WorkOrder `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		ids = append(ids, apiResp.Data.ID)
	}

	// Verify sequential numbering (WO-2026-0001, WO-2026-0002, etc.)
	t.Logf("Generated IDs: %v", ids)
	
	for i := 1; i < len(ids); i++ {
		// Extract numeric part (last 4 digits)
		prevNum := ids[i-1][len(ids[i-1])-4:]
		currNum := ids[i][len(ids[i])-4:]
		
		if currNum <= prevNum {
			t.Errorf("IDs not sequential: %s should be greater than %s", ids[i], ids[i-1])
		}
	}
}

// TestWorkOrderIDGeneration_YearRollover tests ID generation across year boundaries
func TestWorkOrderIDGeneration_YearRollover(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Manually insert a sequence for a different year to simulate rollover
	_, err := db.Exec("INSERT INTO id_sequences (prefix, next_num) VALUES (?, ?)", "WO-2025", 9999)
	if err != nil {
		t.Fatalf("Failed to seed sequence: %v", err)
	}

	// Create work order in current year (2026)
	payload := `{"assembly_ipn":"ASY-ROLLOVER","qty":1}`
	req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	if rr.Code != 200 {
		t.Fatalf("Failed to create WO: %d %s", rr.Code, rr.Body.String())
	}

	var apiResp struct {
		Data WorkOrder `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should start at 0001 for new year, not continue from 2025
	id := apiResp.Data.ID
	if !strings.Contains(id, "2026") {
		t.Errorf("Expected ID to contain 2026, got: %s", id)
	}
	if !strings.HasSuffix(id, "0001") {
		t.Errorf("Expected first ID of year to be 0001, got: %s", id)
	}
}

// TestWorkOrderIDGeneration_Fallback tests that ID generation falls back gracefully on transaction failure
func TestWorkOrderIDGeneration_Fallback(t *testing.T) {
	// This test verifies the timestamp-based fallback when sequence table is unavailable
	// The fallback is already implemented in nextID() function
	
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Drop id_sequences table to force fallback
	_, err := db.Exec("DROP TABLE IF EXISTS id_sequences")
	if err != nil {
		t.Fatalf("Failed to drop sequences table: %v", err)
	}

	// Create work order - should use timestamp-based fallback
	payload := `{"assembly_ipn":"ASY-FALLBACK","qty":1}`
	req := httptest.NewRequest("POST", "/api/v1/workorders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handleCreateWorkOrder(rr, req)

	if rr.Code != 200 {
		t.Fatalf("Failed to create WO with fallback: %d %s", rr.Code, rr.Body.String())
	}

	var apiResp struct {
		Data WorkOrder `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify ID was still generated (using fallback)
	id := apiResp.Data.ID
	if !strings.HasPrefix(id, "WO-") {
		t.Errorf("Fallback ID should still have WO- prefix, got: %s", id)
	}
	
	t.Logf("Fallback ID generated: %s", id)
}

type testError struct {
	msg  string
	code int
	body string
}

func (e *testError) Error() string {
	return e.msg + " - " + e.body
}
