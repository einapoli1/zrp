package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQualityWorkflowIntegration(t *testing.T) {
	// Initialize test database
	if err := initDB(":memory:"); err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Seed test data
	seedTestUsers(t)

	t.Run("NCR Creation with created_by", testNCRCreationWithCreatedBy)
	t.Run("Create CAPA from NCR via API", testCreateCAPAFromNCRAPI)
	t.Run("Create ECO from NCR via API", testCreateECOFromNCRAPI)
	t.Run("CAPA Approval Security", testCAPAApprovalSecurity)
	t.Run("CAPA Status Auto-advancement", testCAPAStatusAutoAdvancement)
}

func seedTestUsers(t *testing.T) {
	// Create test users with different roles
	users := []struct {
		username string
		role     string
	}{
		{"test_qe", "qe"},
		{"test_manager", "manager"},
		{"test_user", "user"},
		{"test_admin", "admin"},
	}

	for _, user := range users {
		_, err := db.Exec(`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
			user.username, "test_hash", user.role)
		if err != nil {
			t.Fatalf("Failed to create test user %s: %v", user.username, err)
		}

		// Create session for the user
		_, err = db.Exec(`INSERT INTO sessions (token, user_id) 
			SELECT ?, id FROM users WHERE username = ?`,
			user.username+"_token", user.username)
		if err != nil {
			t.Fatalf("Failed to create session for %s: %v", user.username, err)
		}
	}
}

func testNCRCreationWithCreatedBy(t *testing.T) {
	// Create NCR request
	ncrData := map[string]interface{}{
		"title":       "Test NCR with created_by",
		"description": "Testing created_by field",
		"severity":    "minor",
		"defect_type": "material",
	}

	body, _ := json.Marshal(ncrData)
	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handleCreateNCR(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var result NCR
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify created_by is set
	if result.CreatedBy != "test_qe" {
		t.Errorf("Expected created_by to be 'test_qe', got '%s'", result.CreatedBy)
	}

	// Verify in database
	var createdBy string
	err := db.QueryRow("SELECT created_by FROM ncrs WHERE id = ?", result.ID).Scan(&createdBy)
	if err != nil {
		t.Errorf("Failed to query created_by from database: %v", err)
	}
	if createdBy != "test_qe" {
		t.Errorf("Database created_by should be 'test_qe', got '%s'", createdBy)
	}
}

func testCreateCAPAFromNCRAPI(t *testing.T) {
	// First create an NCR
	ncrID := "NCR-2026-TEST1"
	_, err := db.Exec(`INSERT INTO ncrs (id, title, description, status, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		ncrID, "Test NCR", "Test description", "resolved", "test_qe")
	if err != nil {
		t.Fatalf("Failed to create test NCR: %v", err)
	}

	// Test creating CAPA from NCR
	capaData := map[string]interface{}{
		"owner":    "test_manager",
		"due_date": "2026-03-01",
	}

	body, _ := json.Marshal(capaData)
	req := httptest.NewRequest("POST", "/api/v1/ncrs/"+ncrID+"/create-capa", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handleCreateCAPAFromNCR(w, req, ncrID)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var result CAPA
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify CAPA is linked to NCR
	if result.LinkedNCRID != ncrID {
		t.Errorf("Expected linked_ncr_id to be '%s', got '%s'", ncrID, result.LinkedNCRID)
	}

	// Verify auto-populated title
	expectedTitle := "CAPA for NCR " + ncrID + ": Test NCR"
	if result.Title != expectedTitle {
		t.Errorf("Expected title to be '%s', got '%s'", expectedTitle, result.Title)
	}
}

func testCreateECOFromNCRAPI(t *testing.T) {
	// First create an NCR
	ncrID := "NCR-2026-TEST2"
	_, err := db.Exec(`INSERT INTO ncrs (id, title, description, status, corrective_action, ipn, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		ncrID, "Test NCR for ECO", "Test description", "resolved", "Replace component", "TEST-001", "test_qe")
	if err != nil {
		t.Fatalf("Failed to create test NCR: %v", err)
	}

	// Test creating ECO from NCR
	req := httptest.NewRequest("POST", "/api/v1/ncrs/"+ncrID+"/create-eco", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})

	w := httptest.NewRecorder()
	handleCreateECOFromNCR(w, req, ncrID)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify ECO is linked to NCR
	if result["ncr_id"] != ncrID {
		t.Errorf("Expected ncr_id to be '%s', got '%v'", ncrID, result["ncr_id"])
	}

	// Verify auto-populated fields
	expectedTitle := "[NCR " + ncrID + "] Test NCR for ECO — Corrective Action"
	if result["title"] != expectedTitle {
		t.Errorf("Expected title to be '%s', got '%s'", expectedTitle, result["title"])
	}

	if result["affected_ipns"] != "TEST-001" {
		t.Errorf("Expected affected_ipns to be 'TEST-001', got '%v'", result["affected_ipns"])
	}
}

func testCAPAApprovalSecurity(t *testing.T) {
	// Create a test CAPA
	capaID := "CAPA-2026-TEST"
	_, err := db.Exec(`INSERT INTO capas (id, title, status, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`,
		capaID, "Test CAPA", "open")
	if err != nil {
		t.Fatalf("Failed to create test CAPA: %v", err)
	}

	t.Run("QE approval by non-QE user should fail", func(t *testing.T) {
		updateData := map[string]interface{}{
			"approved_by_qe": "approve",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_user_token"}) // regular user, not QE
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handleUpdateCAPA(w, req, capaID)

		if w.Code != 403 {
			t.Errorf("Expected status 403 for non-QE user, got %d", w.Code)
		}
	})

	t.Run("QE approval by QE user should succeed", func(t *testing.T) {
		updateData := map[string]interface{}{
			"approved_by_qe": "approve",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handleUpdateCAPA(w, req, capaID)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for QE user, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Manager approval by non-manager should fail", func(t *testing.T) {
		updateData := map[string]interface{}{
			"approved_by_mgr": "approve",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"}) // QE user, not manager
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handleUpdateCAPA(w, req, capaID)

		if w.Code != 403 {
			t.Errorf("Expected status 403 for non-manager user, got %d", w.Code)
		}
	})
}

func testCAPAStatusAutoAdvancement(t *testing.T) {
	// Create a test CAPA
	capaID := "CAPA-2026-AUTO"
	_, err := db.Exec(`INSERT INTO capas (id, title, status, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`,
		capaID, "Auto-advancement test CAPA", "open")
	if err != nil {
		t.Fatalf("Failed to create test CAPA: %v", err)
	}

	// First, QE approves
	updateData := map[string]interface{}{
		"approved_by_qe": "approve",
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handleUpdateCAPA(w, req, capaID)

	if w.Code != 200 {
		t.Errorf("QE approval failed: %d: %s", w.Code, w.Body.String())
		return
	}

	// Now manager approves - this should auto-advance status to "approved"
	updateData = map[string]interface{}{
		"approved_by_mgr": "approve",
	}

	body, _ = json.Marshal(updateData)
	req = httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_manager_token"})
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handleUpdateCAPA(w, req, capaID)

	if w.Code != 200 {
		t.Errorf("Manager approval failed: %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify status was auto-advanced
	var result CAPA
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Status != "approved" {
		t.Errorf("Expected status to auto-advance to 'approved', got '%s'", result.Status)
	}

	// Verify in database
	var dbStatus string
	err = db.QueryRow("SELECT status FROM capas WHERE id = ?", capaID).Scan(&dbStatus)
	if err != nil {
		t.Errorf("Failed to query status from database: %v", err)
	}
	if dbStatus != "approved" {
		t.Errorf("Database status should be 'approved', got '%s'", dbStatus)
	}
}