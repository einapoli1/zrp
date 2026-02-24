package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestECO_IDGeneration_UsesNextID verifies that ECO creation uses the fixed nextID() function
// This test ensures the fix from commit e23d24e is working correctly
func TestECO_IDGeneration_UsesNextID(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create id_sequences table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER NOT NULL DEFAULT 1)`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences: %v", err)
	}

	// Create first ECO - should get ECO-001
	reqBody := `{"title":"First ECO"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleCreateECO(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create first ECO: %d - %s", w.Code, w.Body.String())
	}

	var response1 struct {
		Data ECO `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response1)

	// ID format is ECO-YYYY-NNN
	year := time.Now().Format("2006")
	expectedID1 := fmt.Sprintf("ECO-%s-001", year)
	if response1.Data.ID != expectedID1 {
		t.Errorf("Expected first ECO ID to be %s, got %s", expectedID1, response1.Data.ID)
	}

	// Create second ECO - should get ECO-YYYY-002
	reqBody2 := `{"title":"Second ECO"}`
	req2 := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody2))
	w2 := httptest.NewRecorder()
	handleCreateECO(w2, req2)

	var response2 struct {
		Data ECO `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&response2)

	expectedID2 := fmt.Sprintf("ECO-%s-002", year)
	if response2.Data.ID != expectedID2 {
		t.Errorf("Expected second ECO ID to be %s, got %s", expectedID2, response2.Data.ID)
	}

	// Verify id_sequences table was updated (prefix is ECO-YYYY)
	var nextNum int
	seqKey := fmt.Sprintf("ECO-%s", year)
	err = db.QueryRow("SELECT next_num FROM id_sequences WHERE prefix=?", seqKey).Scan(&nextNum)
	if err != nil {
		t.Errorf("id_sequences table not updated: %v", err)
	}
	if nextNum != 3 {
		t.Errorf("Expected next_num to be 3, got %d", nextNum)
	}
}

// TestECO_IDGeneration_ConcurrentCreation verifies that concurrent ECO creation
// doesn't create duplicate IDs using the nextID() function
func TestECO_IDGeneration_ConcurrentCreation(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER NOT NULL DEFAULT 1)`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	ecoIDs := make([]string, numGoroutines)
	var mu sync.Mutex

	// Create 10 ECOs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			reqBody := `{"title":"Concurrent ECO ` + string(rune('A'+idx)) + `"}`
			req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			mu.Lock()
			handleCreateECO(w, req)
			mu.Unlock()

			if w.Code == 200 {
				var response struct {
					Data ECO `json:"data"`
				}
				json.NewDecoder(w.Body).Decode(&response)
				ecoIDs[idx] = response.Data.ID
			}
		}(i)
	}

	wg.Wait()

	// Verify all IDs are unique
	idMap := make(map[string]bool)
	for _, id := range ecoIDs {
		if id == "" {
			continue
		}
		if idMap[id] {
			t.Errorf("Duplicate ECO ID detected: %s", id)
		}
		idMap[id] = true
	}

	// Should have created 10 unique IDs
	if len(idMap) != numGoroutines {
		t.Errorf("Expected %d unique IDs, got %d", numGoroutines, len(idMap))
	}
}

// TestECO_IDGeneration_SequencePersistence verifies that the ID sequence
// persists across database operations
func TestECO_IDGeneration_SequencePersistence(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER NOT NULL DEFAULT 1)`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences: %v", err)
	}

	// Create first ECO
	reqBody := `{"title":"First"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleCreateECO(w, req)

	var response1 struct {
		Data ECO `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response1)
	firstID := response1.Data.ID

	// Delete the ECO
	db.Exec("DELETE FROM eco_revisions WHERE eco_id=?", firstID)
	db.Exec("DELETE FROM ecos WHERE id=?", firstID)

	// Create another ECO - should still increment sequence, not reuse
	reqBody2 := `{"title":"Second"}`
	req2 := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody2))
	w2 := httptest.NewRecorder()
	handleCreateECO(w2, req2)

	var response2 struct {
		Data ECO `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&response2)

	year := time.Now().Format("2006")
	expectedID2 := fmt.Sprintf("ECO-%s-002", year)
	if response2.Data.ID != expectedID2 {
		t.Errorf("Expected %s (sequence should not reuse), got %s", expectedID2, response2.Data.ID)
	}
}

// TestECO_IDGeneration_PaddingFormat verifies that IDs are zero-padded to 3 digits
func TestECO_IDGeneration_PaddingFormat(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER NOT NULL DEFAULT 1)`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences: %v", err)
	}

	// Set sequence to 99 (prefix includes year)
	year := time.Now().Format("2006")
	seqKey := fmt.Sprintf("ECO-%s", year)
	db.Exec("INSERT OR REPLACE INTO id_sequences (prefix, next_num) VALUES (?, 99)", seqKey)

	reqBody := `{"title":"Test"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleCreateECO(w, req)

	var response struct {
		Data ECO `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	expectedID1 := fmt.Sprintf("ECO-%s-099", year)
	if response.Data.ID != expectedID1 {
		t.Errorf("Expected %s (zero-padded), got %s", expectedID1, response.Data.ID)
	}

	// Create another - should be ECO-YYYY-100
	reqBody2 := `{"title":"Test2"}`
	req2 := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody2))
	w2 := httptest.NewRecorder()
	handleCreateECO(w2, req2)

	var response2 struct {
		Data ECO `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&response2)

	expectedID2 := fmt.Sprintf("ECO-%s-100", year)
	if response2.Data.ID != expectedID2 {
		t.Errorf("Expected %s, got %s", expectedID2, response2.Data.ID)
	}
}
