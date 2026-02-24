package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
)

// ==================== DATA INTEGRITY TESTS ====================

func TestECO_DBConstraints_StatusEnum(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test that database CHECK constraint enforces valid statuses
	invalidStatuses := []string{"invalid", "DRAFT", "Draft", "pending", "closed"}
	
	for _, status := range invalidStatuses {
		_, err := db.Exec(`INSERT INTO ecos (id, title, status) VALUES (?, 'Test', ?)`, "ECO-TEST-"+status, status)
		if err == nil {
			t.Errorf("Database allowed invalid status '%s' - CHECK constraint may be missing", status)
		} else {
			t.Logf("Correctly rejected invalid status '%s': %v", status, err)
		}
	}

	// Test valid statuses
	validStatuses := []string{"draft", "review", "approved", "implemented", "rejected", "cancelled"}
	for i, status := range validStatuses {
		_, err := db.Exec(`INSERT INTO ecos (id, title, status) VALUES (?, 'Test', ?)`, "ECO-VALID-"+string(rune(i)), status)
		if err != nil {
			t.Errorf("Database rejected valid status '%s': %v", status, err)
		}
	}
}

func TestECO_DBConstraints_PriorityEnum(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test that database CHECK constraint enforces valid priorities
	invalidPriorities := []string{"urgent", "CRITICAL", "medium"}
	
	for _, priority := range invalidPriorities {
		_, err := db.Exec(`INSERT INTO ecos (id, title, priority) VALUES (?, 'Test', ?)`, "ECO-TEST-"+priority, priority)
		if err == nil {
			t.Errorf("Database allowed invalid priority '%s' - CHECK constraint may be missing", priority)
		}
	}

	// Test valid priorities
	validPriorities := []string{"low", "normal", "high", "critical"}
	for i, priority := range validPriorities {
		_, err := db.Exec(`INSERT INTO ecos (id, title, priority) VALUES (?, 'Test', ?)`, "ECO-PRI-"+string(rune(i)), priority)
		if err != nil {
			t.Errorf("Database rejected valid priority '%s': %v", priority, err)
		}
	}
}

func TestECO_DBConstraints_NotNullTitle(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test that database enforces NOT NULL on title
	_, err := db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', NULL)`)
	if err == nil {
		t.Error("Database allowed NULL title - NOT NULL constraint may be missing")
	} else {
		t.Logf("Correctly rejected NULL title: %v", err)
	}

	// Empty string should be allowed by DB but caught by validation
	_, err = db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-002', '')`)
	if err != nil {
		t.Logf("Database also rejects empty title: %v", err)
	}
}

func TestECO_ForeignKey_ECORevisions(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Enable foreign key enforcement (SQLite requires this)
	db.Exec("PRAGMA foreign_keys = ON")

	// Try to create revision for non-existent ECO
	_, err := db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status) VALUES ('ECO-NONEXISTENT', 'A', 'created')`)
	if err == nil {
		t.Error("Database allowed revision for non-existent ECO - foreign key constraint may not be enforced")
	} else {
		t.Logf("Correctly enforced foreign key constraint: %v", err)
	}

	// Create ECO and then revision
	_, err = db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test')`)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	_, err = db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status) VALUES ('ECO-001', 'A', 'created')`)
	if err != nil {
		t.Errorf("Failed to create revision for valid ECO: %v", err)
	}
}

func TestECO_ForeignKey_OnDeleteBehavior(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Enable foreign key enforcement
	db.Exec("PRAGMA foreign_keys = ON")

	// Create ECO with revisions
	_, err := db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test')`)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}
	_, err = db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status) VALUES ('ECO-001', 'A', 'created'), ('ECO-001', 'B', 'approved')`)
	if err != nil {
		t.Fatalf("Failed to create revisions: %v", err)
	}

	// Try to delete ECO
	_, err = db.Exec(`DELETE FROM ecos WHERE id='ECO-001'`)
	
	if err != nil {
		// Foreign key prevents deletion (RESTRICT or NO ACTION)
		t.Logf("Foreign key prevents ECO deletion when revisions exist: %v", err)
		
		// Verify revisions still exist
		var revCount int
		db.QueryRow("SELECT COUNT(*) FROM eco_revisions WHERE eco_id='ECO-001'").Scan(&revCount)
		if revCount != 2 {
			t.Errorf("Expected 2 revisions to remain, got %d", revCount)
		}
	} else {
		// Deletion succeeded (CASCADE or no FK enforcement)
		var revCount int
		db.QueryRow("SELECT COUNT(*) FROM eco_revisions WHERE eco_id='ECO-001'").Scan(&revCount)
		
		if revCount == 0 {
			t.Log("Foreign key has CASCADE behavior - revisions deleted with ECO")
		} else {
			t.Errorf("ECO deleted but revisions remain - foreign key may not be enforced (revisions: %d)", revCount)
		}
	}
}

// ==================== REVISION LETTER SEQUENCE TESTS ====================

func TestECO_RevisionLetterSequence(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test')`)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	// Create initial revision
	reqBody := `{"changes_summary":"Initial revision"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/revisions", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleCreateECORevision(w, req, "ECO-001")

	if w.Code != 200 {
		t.Fatalf("Failed to create first revision: %d - %s", w.Code, w.Body.String())
	}

	var response1 struct {
		Data struct {
			Revision string `json:"revision"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response1)
	
	if response1.Data.Revision != "A" {
		t.Errorf("Expected first revision to be 'A', got '%s'", response1.Data.Revision)
	}

	// Create second revision
	reqBody2 := `{"changes_summary":"Second revision"}`
	req2 := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/revisions", bytes.NewBufferString(reqBody2))
	w2 := httptest.NewRecorder()
	handleCreateECORevision(w2, req2, "ECO-001")

	if w2.Code != 200 {
		t.Fatalf("Failed to create second revision: %d - %s", w2.Code, w2.Body.String())
	}

	var response2 struct {
		Data struct {
			Revision string `json:"revision"`
		} `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&response2)
	
	if response2.Data.Revision != "B" {
		t.Errorf("Expected second revision to be 'B', got '%s'", response2.Data.Revision)
	}
}

func TestECO_RevisionLetterOverflow(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test')`)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	// Manually insert revisions up to Z
	_, err = db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status) VALUES ('ECO-001', 'Z', 'created')`)
	if err != nil {
		t.Fatalf("Failed to create Z revision: %v", err)
	}

	// Try to create another revision (should overflow)
	reqBody := `{"changes_summary":"Post-Z revision"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/revisions", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleCreateECORevision(w, req, "ECO-001")

	if w.Code != 200 {
		t.Fatalf("Failed to create revision after Z: %d - %s", w.Code, w.Body.String())
	}
	
	var response struct {
		Data struct {
			Revision string `json:"revision"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)
	
	// After Z, should use AA (Excel-style column naming)
	if response.Data.Revision != "AA" {
		t.Errorf("Expected revision 'AA' after Z, got '%s'", response.Data.Revision)
	}
	
	// Test a few more to verify the pattern
	reqBody2 := `{"changes_summary":"Another revision"}`
	req2 := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/revisions", bytes.NewBufferString(reqBody2))
	w2 := httptest.NewRecorder()
	handleCreateECORevision(w2, req2, "ECO-001")
	
	if w2.Code == 200 {
		var response2 struct {
			Data struct {
				Revision string `json:"revision"`
			} `json:"data"`
		}
		json.NewDecoder(w2.Body).Decode(&response2)
		if response2.Data.Revision != "AB" {
			t.Errorf("Expected revision 'AB' after AA, got '%s'", response2.Data.Revision)
		}
	}
}

// ==================== NCR LINKING TESTS ====================

func TestECO_NCRLinking(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create NCR table for linking test
	db.Exec(`CREATE TABLE IF NOT EXISTS ncrs (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		status TEXT DEFAULT 'open'
	)`)
	db.Exec(`INSERT INTO ncrs (id, title) VALUES ('NCR-001', 'Test NCR')`)

	db.Exec("CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER)")

	// Create ECO linked to NCR
	reqBody := `{"title":"ECO from NCR","ncr_id":"NCR-001"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create ECO with NCR link: %d - %s", w.Code, w.Body.String())
	}

	var response struct {
		Data struct {
			ID    string `json:"id"`
			NcrID string `json:"ncr_id"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if response.Data.NcrID != "NCR-001" {
		t.Errorf("Expected ncr_id 'NCR-001', got '%s'", response.Data.NcrID)
	}

	// Note: Schema doesn't have FK constraint on ncr_id, so invalid NCR IDs are allowed
	// This might be intentional for flexibility
}

// ==================== CONCURRENT UPDATE TESTS ====================

func TestECO_ConcurrentUpdates_LastWriteWins(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, description, status, priority) VALUES ('ECO-001', 'Original', 'Original Desc', 'draft', 'normal')`)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	// Simulate two users updating concurrently
	update1 := `{"title":"Updated by User 1","description":"User 1 changes","status":"draft","priority":"high"}`
	update2 := `{"title":"Updated by User 2","description":"User 2 changes","status":"review","priority":"critical"}`

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		req := httptest.NewRequest("PUT", "/api/v1/ecos/ECO-001", bytes.NewBufferString(update1))
		w := httptest.NewRecorder()
		handleUpdateECO(w, req, "ECO-001")
	}()

	go func() {
		defer wg.Done()
		req := httptest.NewRequest("PUT", "/api/v1/ecos/ECO-001", bytes.NewBufferString(update2))
		w := httptest.NewRecorder()
		handleUpdateECO(w, req, "ECO-001")
	}()

	wg.Wait()

	// Check final state - last write wins (no optimistic locking)
	var title, status, priority string
	db.QueryRow("SELECT title, status, priority FROM ecos WHERE id='ECO-001'").Scan(&title, &status, &priority)

	t.Logf("Final state: title=%s, status=%s, priority=%s", title, status, priority)
	
	// Should be one of the two updates, but user might lose changes
	if title != "Updated by User 1" && title != "Updated by User 2" {
		t.Errorf("Unexpected final title: %s", title)
	}

	// Note: This test documents the lack of optimistic locking
	// In production, might want to implement version checking
}

// ==================== EDGE CASE: EMPTY REVISIONS ====================

func TestECO_NoRevisions(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Manually create ECO without calling ensureInitialRevision
	_, err := db.Exec(`INSERT INTO ecos (id, title, status) VALUES ('ECO-001', 'Test', 'draft')`)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	// Try to list revisions
	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-001/revisions", nil)
	w := httptest.NewRecorder()
	handleListECORevisions(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response struct {
		Data []ECORevision `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.Data) != 0 {
		t.Logf("Expected empty revisions list, got %d", len(response.Data))
	}
}

// ==================== PRIORITY FILTERING TESTS ====================

func TestECO_PriorityOrdering(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create ECOs with different priorities
	priorities := []string{"low", "normal", "high", "critical"}
	for i, p := range priorities {
		_, err := db.Exec(`INSERT INTO ecos (id, title, priority, created_at) VALUES (?, ?, ?, datetime('now', ? || ' seconds'))`,
			fmt.Sprintf("ECO-%03d", i+1), "Test "+p, p, fmt.Sprintf("+%d", i))
		if err != nil {
			t.Fatalf("Failed to create ECO with priority %s: %v", p, err)
		}
	}

	// List all ECOs (should be ordered by created_at DESC)
	req := httptest.NewRequest("GET", "/api/v1/ecos", nil)
	w := httptest.NewRecorder()
	handleListECOs(w, req)

	var response struct {
		Data []ECO `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.Data) != 4 {
		t.Errorf("Expected 4 ECOs, got %d", len(response.Data))
	}

	// Note: Current API doesn't support filtering by priority
	// Frontend would need to filter client-side
	// This test documents that limitation
}
