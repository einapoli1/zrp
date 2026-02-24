package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPartCreationWithSettingDisabled(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Ensure setting is false
	db.Exec("UPDATE settings SET value = 'false' WHERE key = 'require_eco_approval_for_creation'")

	// Create temp parts directory
	tmpDir := t.TempDir()
	partsDir = tmpDir
	defer func() { partsDir = "" }()

	// Create a test category CSV
	csvPath := filepath.Join(tmpDir, "test-category.csv")
	csvContent := "IPN,Description,Manufacturer\n"
	os.WriteFile(csvPath, []byte(csvContent), 0644)

	body := map[string]interface{}{
		"ipn":      "TEST-001",
		"category": "test-category",
		"fields": map[string]string{
			"Description":  "Test Part",
			"Manufacturer": "Test Mfg",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreatePart(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	// Should NOT have eco_created flag when setting is disabled
	if eco, ok := resp["eco_created"].(bool); ok && eco {
		t.Error("Expected direct part creation, got ECO creation instead")
	}

	// Verify part was added to CSV
	content, _ := os.ReadFile(csvPath)
	if !bytes.Contains(content, []byte("TEST-001")) {
		t.Error("Expected part to be written to CSV")
	}
}

func TestPartCreationWithSettingEnabled(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Enable ECO approval requirement
	db.Exec("UPDATE settings SET value = 'true' WHERE key = 'require_eco_approval_for_creation'")

	// Create temp parts directory
	tmpDir := t.TempDir()
	partsDir = tmpDir
	defer func() { partsDir = "" }()

	// Create a test category CSV
	csvPath := filepath.Join(tmpDir, "test-category.csv")
	csvContent := "IPN,Description,Manufacturer\n"
	os.WriteFile(csvPath, []byte(csvContent), 0644)

	body := map[string]interface{}{
		"ipn":      "TEST-002",
		"category": "test-category",
		"fields": map[string]string{
			"Description":  "Test Part 2",
			"Manufacturer": "Test Mfg",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreatePart(rec, req)

	bodyStr := rec.Body.String()
	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, bodyStr)
		return
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(bodyStr), &apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v, body: %s", err, bodyStr)
	}
	resp := apiResp.Data

	// Should have eco_created flag
	if eco, ok := resp["eco_created"].(bool); !ok || !eco {
		t.Errorf("Expected ECO creation, got direct part creation. Response: %v", resp)
		return
	}

	ecoID, ok := resp["eco_id"].(string)
	if !ok || ecoID == "" {
		t.Errorf("Expected eco_id in response. Response: %v", resp)
		return
	}

	// Verify ECO was created
	var ecoType, ecoStatus string
	err := db.QueryRow("SELECT type, status FROM ecos WHERE id = ?", ecoID).Scan(&ecoType, &ecoStatus)
	if err != nil {
		t.Fatalf("ECO not found: %v", err)
	}

	if ecoType != "creation" {
		t.Errorf("Expected ECO type 'creation', got %s", ecoType)
	}

	if ecoStatus != "pending" {
		t.Errorf("Expected ECO status 'pending', got %s", ecoStatus)
	}

	// Part should NOT be in CSV yet
	content, _ := os.ReadFile(csvPath)
	if bytes.Contains(content, []byte("TEST-002")) {
		t.Error("Part should not be in CSV before ECO approval")
	}
}

func TestECOApprovalCreatesPartSimple(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Create temp parts directory
	tmpDir := t.TempDir()
	partsDir = tmpDir
	defer func() { partsDir = "" }()

	// Create a test category CSV
	csvPath := filepath.Join(tmpDir, "test-category.csv")
	csvContent := "IPN,Description,Manufacturer\n"
	os.WriteFile(csvPath, []byte(csvContent), 0644)

	// Create a creation ECO manually
	ecoData := map[string]interface{}{
		"item_type": "part",
		"ipn":       "TEST-003",
		"category":  "test-category",
		"fields": map[string]string{
			"Description":  "Test Part 3",
			"Manufacturer": "Test Mfg",
		},
	}
	descBytes, _ := json.Marshal(ecoData)

	db.Exec("INSERT INTO ecos (id, title, description, type, status, priority, affected_ipns) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"ECO001", "Create Part: TEST-003", string(descBytes), "creation", "review", "normal", "TEST-003")

	// Approve the ECO
	req := httptest.NewRequest("POST", "/api/v1/ecos/ECO001/approve", nil)
	rec := httptest.NewRecorder()

	handleApproveECO(rec, req, "ECO001")

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		return
	}

	// Verify ECO is approved
	var status string
	err := db.QueryRow("SELECT status FROM ecos WHERE id = 'ECO001'").Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query ECO status: %v", err)
	}
	if status != "approved" {
		t.Errorf("Expected ECO status 'approved', got %s", status)
		return
	}

	// Verify part was created in CSV
	content, _ := os.ReadFile(csvPath)
	if !bytes.Contains(content, []byte("TEST-003")) {
		t.Error("Expected part to be created in CSV after ECO approval")
	}
}

func TestECODenialPreventsCreation(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Create temp parts directory
	tmpDir := t.TempDir()
	partsDir = tmpDir
	defer func() { partsDir = "" }()

	// Create a test category CSV
	csvPath := filepath.Join(tmpDir, "test-category.csv")
	csvContent := "IPN,Description,Manufacturer\n"
	os.WriteFile(csvPath, []byte(csvContent), 0644)

	// Create a creation ECO
	ecoData := map[string]interface{}{
		"item_type": "part",
		"ipn":       "TEST-004",
		"category":  "test-category",
		"fields": map[string]string{
			"Description":  "Test Part 4",
			"Manufacturer": "Test Mfg",
		},
	}
	descBytes, _ := json.Marshal(ecoData)

	db.Exec("INSERT INTO ecos (id, title, description, type, status, priority, affected_ipns) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"ECO002", "Create Part: TEST-004", string(descBytes), "creation", "review", "normal", "TEST-004")

	// Update ECO status to rejected
	db.Exec("UPDATE ecos SET status = 'rejected' WHERE id = 'ECO002'")

	// Verify part was NOT created
	content, _ := os.ReadFile(csvPath)
	if bytes.Contains(content, []byte("TEST-004")) {
		t.Error("Part should not be created when ECO is denied")
	}
}
