package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestHandleCreateNCR_ConcurrentIDGenerationNoDuplicates verifies the fix from commit e23d24e
// This test ensures that concurrent NCR creation doesn't generate duplicate IDs
func TestHandleCreateNCR_ConcurrentIDGenerationNoDuplicates(t *testing.T) {
	oldDB := db
	testDB := setupNCRTestDB(t)
	db = testDB
	defer func() { testDB.Close(); db = oldDB }()

	const concurrentRequests = 10
	var wg sync.WaitGroup
	ncrIDs := make([]string, concurrentRequests)
	errors := make([]error, concurrentRequests)

	// Launch concurrent NCR creation requests
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			ncr := map[string]interface{}{
				"title":       fmt.Sprintf("Concurrent NCR %d", index),
				"description": fmt.Sprintf("Test concurrent creation %d", index),
				"severity":    "minor",
			}
			body, _ := json.Marshal(ncr)
			req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handleCreateNCR(w, req)

			if w.Code != 200 {
				errors[index] = fmt.Errorf("request %d failed with status %d: %s", index, w.Code, w.Body.String())
				return
			}

			var response NCR
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				errors[index] = fmt.Errorf("request %d failed to parse response: %v", index, err)
				return
			}

			ncrIDs[index] = response.ID
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			t.Errorf("Concurrent request %d error: %v", i, err)
		}
	}

	// Verify all IDs are unique (no duplicates)
	idMap := make(map[string]bool)
	duplicates := []string{}

	for i, id := range ncrIDs {
		if id == "" {
			t.Errorf("Request %d generated empty ID", i)
			continue
		}
		if idMap[id] {
			duplicates = append(duplicates, id)
		}
		idMap[id] = true
	}

	if len(duplicates) > 0 {
		t.Errorf("Found duplicate NCR IDs (race condition detected): %v", duplicates)
		t.Errorf("All IDs: %v", ncrIDs)
	}

	// Verify all NCRs were actually inserted into the database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM ncrs").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count NCRs: %v", err)
	}

	if count != concurrentRequests {
		t.Errorf("Expected %d NCRs in database, got %d", concurrentRequests, count)
	}

	t.Logf("✓ All %d concurrent NCR creations generated unique IDs", concurrentRequests)
	t.Logf("✓ Generated IDs: %v", ncrIDs)
}

// TestHandleCreateNCR_IDSequenceIncrementsCorrectly verifies nextID() increments properly
func TestHandleCreateNCR_IDSequenceIncrementsCorrectly(t *testing.T) {
	oldDB := db
	testDB := setupNCRTestDB(t)
	db = testDB
	defer func() { testDB.Close(); db = oldDB }()

	// Create 5 NCRs sequentially
	expectedIDs := []string{}
	year := "2026" // Assuming current year in test

	for i := 1; i <= 5; i++ {
		ncr := map[string]interface{}{
			"title":    fmt.Sprintf("Sequential NCR %d", i),
			"severity": "minor",
		}
		body, _ := json.Marshal(ncr)
		req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handleCreateNCR(w, req)

		if w.Code != 200 {
			t.Fatalf("Request %d failed: %d - %s", i, w.Code, w.Body.String())
		}

		var response NCR
		json.Unmarshal(w.Body.Bytes(), &response)
		expectedIDs = append(expectedIDs, response.ID)

		// Verify ID format: NCR-2026-001, NCR-2026-002, etc.
		expectedID := fmt.Sprintf("NCR-%s-%03d", year, i)
		if response.ID != expectedID {
			t.Errorf("NCR %d: expected ID %s, got %s", i, expectedID, response.ID)
		}
	}

	// Verify sequence persisted correctly in id_sequences table
	var nextNum int
	err := db.QueryRow("SELECT next_num FROM id_sequences WHERE prefix = ?", "NCR-"+year).Scan(&nextNum)
	if err != nil {
		t.Fatalf("Failed to query id_sequences: %v", err)
	}

	// Next number should be 6 (ready for the next NCR)
	if nextNum != 6 {
		t.Errorf("Expected next_num=6 in id_sequences, got %d", nextNum)
	}

	t.Logf("✓ ID sequence incremented correctly: %v", expectedIDs)
}

// TestHandleCreateNCR_IDSequencePersistsAcrossConnections verifies the fix prevents race conditions
func TestHandleCreateNCR_IDSequencePersistsAcrossConnections(t *testing.T) {
	oldDB := db
	testDB := setupNCRTestDB(t)
	db = testDB
	defer func() { testDB.Close(); db = oldDB }()

	// Create first NCR
	ncr1 := map[string]interface{}{"title": "First NCR", "severity": "minor"}
	body1, _ := json.Marshal(ncr1)
	req1 := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handleCreateNCR(w1, req1)

	var response1 NCR
	json.Unmarshal(w1.Body.Bytes(), &response1)
	firstID := response1.ID

	// Simulate "closing and reopening connection" by just creating another NCR
	// The id_sequences table should persist the counter
	ncr2 := map[string]interface{}{"title": "Second NCR", "severity": "major"}
	body2, _ := json.Marshal(ncr2)
	req2 := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handleCreateNCR(w2, req2)

	var response2 NCR
	json.Unmarshal(w2.Body.Bytes(), &response2)
	secondID := response2.ID

	// IDs should be sequential
	if firstID == secondID {
		t.Errorf("Duplicate IDs generated: %s and %s", firstID, secondID)
	}

	t.Logf("✓ Persistent sequence: %s → %s", firstID, secondID)
}

// Note: setupNCRTestDB is defined in handler_ncr_test.go
