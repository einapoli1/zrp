package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveBOM_ValidIPNs(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create test parts
	createTestPartCSV(t, tmpDir, "components.csv", [][]string{
		{"IPN", "description"},
		{"RES-001", "10K Resistor"},
		{"CAP-001", "10uF Capacitor"},
		{"IC-001", "ATmega328P"},
	})

	// Valid BOM request
	bomReq := BOMSaveRequest{
		AssemblyIPN: "PCA-TEST",
		Items: []BOMItem{
			{IPN: "RES-001", Description: "10K Resistor", Quantity: 2, RefDes: "R1,R2"},
			{IPN: "CAP-001", Description: "10uF Capacitor", Quantity: 1, RefDes: "C1"},
			{IPN: "IC-001", Description: "ATmega328P", Quantity: 1, RefDes: "U1"},
		},
	}

	body, _ := json.Marshal(bomReq)
	req, err := http.NewRequest("POST", "/api/v1/parts/PCA-TEST/bom", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "PCA-TEST")
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var response struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, rr.Body.String())
	}

	if success, ok := response.Data["success"].(bool); !ok || !success {
		t.Errorf("expected success=true, got %v", response.Data["success"])
	}

	// Verify BOM file was created
	bomPath := filepath.Join(tmpDir, "PCA-TEST.csv")
	if _, err := os.Stat(bomPath); os.IsNotExist(err) {
		t.Errorf("BOM file was not created: %s", bomPath)
	}

	// Verify BOM contents
	content, err := os.ReadFile(bomPath)
	if err != nil {
		t.Fatalf("Failed to read BOM file: %v", err)
	}

	contentStr := string(content)
	
	// Check headers
	if !contains(contentStr, "IPN,description,qty,ref") {
		t.Errorf("BOM file missing headers, content: %s", contentStr)
	}
	
	// Check for IPNs (in any format)
	for _, ipn := range []string{"RES-001", "CAP-001", "IC-001"} {
		if !contains(contentStr, ipn) {
			t.Errorf("BOM file missing IPN %s, content: %s", ipn, contentStr)
		}
	}
}

func TestSaveBOM_InvalidIPNRejected(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create test parts
	createTestPartCSV(t, tmpDir, "components.csv", [][]string{
		{"IPN", "description"},
		{"RES-001", "10K Resistor"},
		{"CAP-001", "10uF Capacitor"},
	})

	// BOM with invalid IPN
	bomReq := BOMSaveRequest{
		AssemblyIPN: "PCA-TEST",
		Items: []BOMItem{
			{IPN: "RES-001", Description: "10K Resistor", Quantity: 2, RefDes: "R1,R2"},
			{IPN: "INVALID-999", Description: "Does Not Exist", Quantity: 1, RefDes: "U1"},
		},
	}

	body, _ := json.Marshal(bomReq)
	req, err := http.NewRequest("POST", "/api/v1/parts/PCA-TEST/bom", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "PCA-TEST")
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	errorMsg := response["error"].(string)
	if !contains(errorMsg, "INVALID-999") || !contains(errorMsg, "not found") {
		t.Errorf("expected error message about INVALID-999 not found, got: %s", errorMsg)
	}

	// Verify BOM file was NOT created
	bomPath := filepath.Join(tmpDir, "PCA-TEST.csv")
	if _, err := os.Stat(bomPath); !os.IsNotExist(err) {
		t.Errorf("BOM file should not have been created for invalid BOM")
	}
}

func TestSaveBOM_NonAssemblyIPNRejected(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create test parts
	createTestPartCSV(t, tmpDir, "components.csv", [][]string{
		{"IPN", "description"},
		{"RES-001", "10K Resistor"},
	})

	// Try to save BOM for non-assembly part
	bomReq := BOMSaveRequest{
		AssemblyIPN: "RES-001",
		Items: []BOMItem{
			{IPN: "RES-001", Description: "10K Resistor", Quantity: 1, RefDes: "R1"},
		},
	}

	body, _ := json.Marshal(bomReq)
	req, err := http.NewRequest("POST", "/api/v1/parts/RES-001/bom", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "RES-001")
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	errorMsg := response["error"].(string)
	if !contains(errorMsg, "assembly") {
		t.Errorf("expected error about assembly IPNs, got: %s", errorMsg)
	}
}

func TestSaveBOM_EmptyBOM(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Empty BOM request
	bomReq := BOMSaveRequest{
		AssemblyIPN: "PCA-EMPTY",
		Items:       []BOMItem{},
	}

	body, _ := json.Marshal(bomReq)
	req, err := http.NewRequest("POST", "/api/v1/parts/PCA-EMPTY/bom", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "PCA-EMPTY")
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify empty BOM file was created with headers only
	bomPath := filepath.Join(tmpDir, "PCA-EMPTY.csv")
	content, err := os.ReadFile(bomPath)
	if err != nil {
		t.Fatalf("Failed to read BOM file: %v", err)
	}

	if !contains(string(content), "IPN,description,qty,ref") {
		t.Errorf("BOM file should contain headers")
	}
}

func TestSaveBOM_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create test parts
	createTestPartCSV(t, tmpDir, "components.csv", [][]string{
		{"IPN", "description"},
		{"RES-001", "10K Resistor"},
		{"RES-002", "1K Resistor"},
		{"CAP-001", "10uF Capacitor"},
	})

	// Create initial BOM
	initialBOM := BOMSaveRequest{
		AssemblyIPN: "ASY-UPDATE",
		Items: []BOMItem{
			{IPN: "RES-001", Description: "10K Resistor", Quantity: 1, RefDes: "R1"},
		},
	}

	body, _ := json.Marshal(initialBOM)
	req, _ := http.NewRequest("POST", "/api/v1/parts/ASY-UPDATE/bom", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "ASY-UPDATE")
	})
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to create initial BOM: %v", rr.Body.String())
	}

	// Update with new BOM
	updatedBOM := BOMSaveRequest{
		AssemblyIPN: "ASY-UPDATE",
		Items: []BOMItem{
			{IPN: "RES-002", Description: "1K Resistor", Quantity: 2, RefDes: "R1,R2"},
			{IPN: "CAP-001", Description: "10uF Capacitor", Quantity: 1, RefDes: "C1"},
		},
	}

	body, _ = json.Marshal(updatedBOM)
	req, _ = http.NewRequest("PUT", "/api/v1/parts/ASY-UPDATE/bom", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleUpdateBOM(w, r, "ASY-UPDATE")
	})
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to update BOM: %v", rr.Body.String())
	}

	// Verify BOM was updated (not appended)
	bomPath := filepath.Join(tmpDir, "ASY-UPDATE.csv")
	content, err := os.ReadFile(bomPath)
	if err != nil {
		t.Fatalf("Failed to read BOM file: %v", err)
	}

	contentStr := string(content)
	// Should contain new items
	if !contains(contentStr, "RES-002") || !contains(contentStr, "CAP-001") {
		t.Errorf("Updated BOM should contain new items")
	}
	// Should NOT contain old item
	if contains(contentStr, "RES-001") {
		t.Errorf("Updated BOM should not contain old item RES-001")
	}
}

func TestSaveBOM_InvalidRequestBody(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	req, err := http.NewRequest("POST", "/api/v1/parts/PCA-TEST/bom", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "PCA-TEST")
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestSaveBOM_IPNMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// BOM request with mismatched IPN
	bomReq := BOMSaveRequest{
		AssemblyIPN: "PCA-DIFFERENT",
		Items:       []BOMItem{},
	}

	body, _ := json.Marshal(bomReq)
	req, err := http.NewRequest("POST", "/api/v1/parts/PCA-TEST/bom", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSaveBOM(w, r, "PCA-TEST")
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	errorMsg := response["error"].(string)
	if !contains(errorMsg, "mismatch") {
		t.Errorf("expected error about IPN mismatch, got: %s", errorMsg)
	}
}

func TestValidateBOMIPNs(t *testing.T) {
	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create test parts
	createTestPartCSV(t, tmpDir, "components.csv", [][]string{
		{"IPN", "description"},
		{"RES-001", "10K Resistor"},
		{"CAP-001", "10uF Capacitor"},
	})

	tests := []struct {
		name        string
		items       []BOMItem
		expectError bool
		errorIPNString    string
	}{
		{
			name: "all valid IPNs",
			items: []BOMItem{
				{IPN: "RES-001", Quantity: 1},
				{IPN: "CAP-001", Quantity: 1},
			},
			expectError: false,
		},
		{
			name: "one invalid IPN",
			items: []BOMItem{
				{IPN: "RES-001", Quantity: 1},
				{IPN: "INVALID-123", Quantity: 1},
			},
			expectError: true,
			errorIPNString:    "INVALID-123",
		},
		{
			name: "empty IPN skipped",
			items: []BOMItem{
				{IPN: "", Quantity: 1},
				{IPN: "RES-001", Quantity: 1},
			},
			expectError: false,
		},
		{
			name:        "empty BOM",
			items:       []BOMItem{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBOMIPNs(tt.items)
			
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			
			if tt.expectError && err != nil && tt.errorIPNString != "" {
				if !contains(err.Error(), tt.errorIPNString) {
					t.Errorf("expected error to contain %s, got: %v", tt.errorIPNString, err)
				}
			}
		})
	}
}

// contains is defined in handler_query_profiler_test.go - using that implementation
