package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestParts_ConcurrentCreateSameIPN tests concurrent part creation with the same IPN
// Should detect conflicts and reject all but the first
func TestParts_ConcurrentCreateSameIPN(t *testing.T) {
	oldPartsDir := partsDir
	defer func() { partsDir = oldPartsDir }()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create resistors category CSV
	os.WriteFile(filepath.Join(tmpDir, "resistors.csv"), []byte("IPN,description\n"), 0644)

	const goroutines = 10
	var wg sync.WaitGroup
	results := make([]int, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			body := map[string]interface{}{
				"ipn":      "R-CONCURRENT-001",
				"category": "resistors",
				"fields": map[string]string{
					"description": fmt.Sprintf("Test resistor %d", idx),
				},
			}

			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handleCreatePart(rr, req)

			results[idx] = rr.Code
		}(i)
	}

	wg.Wait()

	// Should have exactly 1 success (200) and rest conflicts (409)
	successCount := 0
	conflictCount := 0
	for _, code := range results {
		if code == 200 {
			successCount++
		} else if code == 409 {
			conflictCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 success, got %d", successCount)
	}

	if conflictCount != goroutines-1 {
		t.Errorf("Expected %d conflicts, got %d", goroutines-1, conflictCount)
	}
}

// TestParts_ConcurrentBOMUpdates tests concurrent BOM file reads/writes
func TestParts_ConcurrentBOMUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	oldPartsDir := partsDir
	partsDir = tmpDir
	defer func() { partsDir = oldPartsDir }()

	// Create component parts
	os.WriteFile(filepath.Join(tmpDir, "components.csv"), []byte("IPN,description\nRES-001,Resistor\n"), 0644)

	// Create assembly with BOM
	bomCSV := `IPN,qty,description
RES-001,10,Resistor`
	os.WriteFile(filepath.Join(tmpDir, "PCA-CONCURRENT.csv"), []byte(bomCSV), 0644)

	// Create assembly part entry
	os.WriteFile(filepath.Join(tmpDir, "assemblies.csv"), []byte("IPN,description\nPCA-CONCURRENT,Concurrent assembly\n"), 0644)

	const goroutines = 20
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/api/v1/parts/PCA-CONCURRENT/bom", nil)
			rr := httptest.NewRecorder()
			handlePartBOM(rr, req, "PCA-CONCURRENT")

			if rr.Code != 200 {
				errors <- fmt.Errorf("BOM request failed with code %d", rr.Code)
				return
			}

			var resp struct {
				Data BOMNode `json:"data"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				errors <- fmt.Errorf("Failed to decode response: %v", err)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent BOM request error: %v", err)
	}
}

// TestParts_SearchPerformance tests search performance with large dataset
func TestParts_SearchPerformance(t *testing.T) {
	oldPartsDir := partsDir
	defer func() { partsDir = oldPartsDir }()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create large parts dataset (1000 parts)
	f, err := os.Create(filepath.Join(tmpDir, "bulk.csv"))
	if err != nil {
		t.Fatalf("Failed to create bulk CSV: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"IPN", "description", "manufacturer", "status"})

	for i := 1; i <= 1000; i++ {
		w.Write([]string{
			fmt.Sprintf("PART-%04d", i),
			fmt.Sprintf("Test part %d", i),
			"TestMfg",
			"active",
		})
	}
	w.Flush()

	// Test search performance
	testCases := []struct {
		name  string
		query string
	}{
		{"Full scan", ""},
		{"IPN prefix", "PART-00"},
		{"Description search", "Test part"},
		{"No results", "NONEXISTENT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/v1/parts?q="+tc.query, nil)
			rr := httptest.NewRecorder()
			handleListParts(rr, req)

			duration := time.Since(start)

			if rr.Code != 200 {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			// Should complete in reasonable time (<500ms)
			if duration > 500*time.Millisecond {
				t.Errorf("Search took too long: %v", duration)
			}

			t.Logf("Search '%s' took %v", tc.name, duration)
		})
	}
}

// TestParts_IPNValidation tests IPN format validation
func TestParts_IPNValidation(t *testing.T) {
	oldPartsDir := partsDir
	defer func() { partsDir = oldPartsDir }()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	os.WriteFile(filepath.Join(tmpDir, "test.csv"), []byte("IPN,description\n"), 0644)

	testCases := []struct {
		name           string
		ipn            string
		expectedStatus int
	}{
		{"Valid IPN", "R-0402-10K", 200},
		{"Valid IPN with numbers", "IC-123-456", 200},
		{"Empty IPN", "", 400},
		{"IPN with spaces", "R 001", 200}, // Should still work
		{"IPN with special chars", "R-001#TEST", 200}, // Should still work
		{"Very long IPN", strings.Repeat("A", 200), 200}, // Should work but maybe warn
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]interface{}{
				"ipn":      tc.ipn,
				"category": "test",
				"fields": map[string]string{
					"description": "Test part",
				},
			}

			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handleCreatePart(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// TestBOM_QuantityCalculation tests BOM quantity rollup
func TestBOM_QuantityCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	oldPartsDir := partsDir
	partsDir = tmpDir
	defer func() { partsDir = oldPartsDir }()

	// Create components
	os.WriteFile(filepath.Join(tmpDir, "components.csv"), []byte("IPN,description\nRES-001,Resistor\n"), 0644)

	// Create sub-assembly that uses 5x RES-001
	os.WriteFile(filepath.Join(tmpDir, "PCA-SUB.csv"), []byte("IPN,qty\nRES-001,5\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "assemblies.csv"), []byte("IPN,description\nPCA-SUB,Sub assembly\nPCA-MAIN,Main assembly\n"), 0644)

	// Create main assembly that uses 3x PCA-SUB and 2x RES-001
	os.WriteFile(filepath.Join(tmpDir, "PCA-MAIN.csv"), []byte("IPN,qty\nPCA-SUB,3\nRES-001,2\n"), 0644)

	// Get BOM for main assembly
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-MAIN/bom", nil)
	rr := httptest.NewRecorder()
	handlePartBOM(rr, req, "PCA-MAIN")

	if rr.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data BOMNode `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	// Should have 2 children: 3x PCA-SUB and 2x RES-001
	if len(resp.Data.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(resp.Data.Children))
	}

	// Verify quantities
	for _, child := range resp.Data.Children {
		if child.IPN == "PCA-SUB" && child.Qty != 3 {
			t.Errorf("Expected PCA-SUB qty=3, got %f", child.Qty)
		}
		if child.IPN == "RES-001" && child.Qty != 2 {
			t.Errorf("Expected RES-001 qty=2 (direct), got %f", child.Qty)
		}

		// Check nested quantity
		if child.IPN == "PCA-SUB" {
			if len(child.Children) != 1 {
				t.Errorf("Expected 1 child in PCA-SUB, got %d", len(child.Children))
			}
			if len(child.Children) > 0 && child.Children[0].Qty != 5 {
				t.Errorf("Expected nested RES-001 qty=5, got %f", child.Children[0].Qty)
			}
		}
	}

	// Total RES-001 needed: 3*5 + 2 = 17 (but this is not automatically calculated)
	t.Logf("BOM tree shows direct quantities. Total calculation (flattened BOM) would require separate endpoint.")
}

// TestBOM_CostRollupAccuracy tests cost rollup calculation accuracy
func TestBOM_CostRollupAccuracy(t *testing.T) {
	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	tmpDir := t.TempDir()
	oldPartsDir := partsDir
	partsDir = tmpDir
	defer func() { partsDir = oldPartsDir }()

	// Create components with specific costs
	os.WriteFile(filepath.Join(tmpDir, "components.csv"), []byte("IPN,description\nRES-001,Resistor\nCAP-001,Capacitor\nIC-001,IC\n"), 0644)

	// Insert component costs
	insertPOLine(t, testDB, "PO-001", "RES-001", 0.05)
	insertPOLine(t, testDB, "PO-002", "CAP-001", 0.25)
	insertPOLine(t, testDB, "PO-003", "IC-001", 5.00)

	// Create assembly: 10x RES-001, 5x CAP-001, 1x IC-001
	os.WriteFile(filepath.Join(tmpDir, "PCA-COST.csv"), []byte("IPN,qty\nRES-001,10\nCAP-001,5\nIC-001,1\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "assemblies.csv"), []byte("IPN,description\nPCA-COST,Cost test assembly\n"), 0644)

	// Test cost endpoint
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-COST/cost", nil)
	rr := httptest.NewRecorder()
	handlePartCost(rr, req, "PCA-COST")

	if rr.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	bomCost, ok := resp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response")
	}

	// Expected: 10*0.05 + 5*0.25 + 1*5.00 = 0.50 + 1.25 + 5.00 = 6.75
	expected := 6.75
	tolerance := 0.001

	if bomCost < expected-tolerance || bomCost > expected+tolerance {
		t.Errorf("Expected BOM cost %.2f, got %.2f", expected, bomCost)
	}
}

// setupBOMCostTestDB creates test database (helper)
func setupPartsCompBOMCostTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	testDB.Exec("PRAGMA foreign_keys = ON")

	testDB.Exec(`CREATE TABLE purchase_orders (
		id TEXT PRIMARY KEY,
		supplier TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	testDB.Exec(`CREATE TABLE po_lines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		po_id TEXT NOT NULL,
		ipn TEXT NOT NULL,
		qty REAL NOT NULL,
		unit_price REAL NOT NULL,
		FOREIGN KEY (po_id) REFERENCES purchase_orders(id)
	)`)

	testDB.Exec(`CREATE TABLE audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		username TEXT,
		action TEXT,
		table_name TEXT,
		record_id TEXT,
		details TEXT
	)`)

	return testDB
}

// insertPOLine inserts a PO line for cost testing
func insertPOLine(t *testing.T, db *sql.DB, poID, ipn string, unitPrice float64) {
	t.Helper()
	db.Exec("INSERT OR IGNORE INTO purchase_orders (id, supplier) VALUES (?, 'Test Supplier')", poID)
	db.Exec("INSERT INTO po_lines (po_id, ipn, qty, unit_price) VALUES (?, ?, 1, ?)", poID, ipn, unitPrice)
}

// TestBOM_FlattenedView tests flattened (consolidated) BOM view
// This would require a new endpoint: GET /api/v1/parts/:ipn/bom/flattened
func TestBOM_FlattenedView(t *testing.T) {
	t.Skip("Flattened BOM endpoint not yet implemented - would consolidate quantities across tree")

	// Expected behavior:
	// - Consolidate duplicate IPNs across entire BOM tree
	// - Sum quantities
	// - Return flat list of {ipn, total_qty, description}
}

// TestBOM_WhereUsed tests where-used (inverse BOM) functionality
func TestBOM_WhereUsed(t *testing.T) {
	t.Skip("Where-used endpoint not found in handler_parts.go - may be in separate file or not implemented")

	// Expected behavior:
	// - Given a component IPN (e.g., RES-001)
	// - Return all assemblies that use it
	// - Show quantity used in each assembly
}

// TestParts_BulkImport tests bulk part import from CSV
func TestParts_BulkImport(t *testing.T) {
	t.Skip("Bulk import endpoint not found - parts are manually edited via CSV files")

	// Expected behavior:
	// - Upload CSV with many parts
	// - Validate all before inserting
	// - Return summary of created/failed/skipped
}

// TestParts_CategoryValidation tests category existence checks
func TestParts_CategoryValidation(t *testing.T) {
	oldPartsDir := partsDir
	defer func() { partsDir = oldPartsDir }()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Don't create "nonexistent" category

	body := map[string]interface{}{
		"ipn":      "R-TEST-001",
		"category": "nonexistent",
		"fields": map[string]string{
			"description": "Test part",
		},
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleCreatePart(rr, req)

	// Should return 404 for non-existent category
	if rr.Code != 404 {
		t.Errorf("Expected status 404 for non-existent category, got %d", rr.Code)
	}
}

// TestParts_SpecialCharactersInFields tests special character handling
func TestParts_SpecialCharactersInFields(t *testing.T) {
	oldPartsDir := partsDir
	defer func() { partsDir = oldPartsDir }()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	os.WriteFile(filepath.Join(tmpDir, "test.csv"), []byte("IPN,description\n"), 0644)

	specialChars := []string{
		"Test with \"quotes\"",
		"Test with 'apostrophe'",
		"Test with <html>",
		"Test with & ampersand",
		"Test with, comma",
		"Test\nwith\nnewline",
	}

	for i, desc := range specialChars {
		t.Run(fmt.Sprintf("Special_%d", i), func(t *testing.T) {
			body := map[string]interface{}{
				"ipn":      fmt.Sprintf("TEST-%03d", i),
				"category": "test",
				"fields": map[string]string{
					"description": desc,
				},
			}

			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handleCreatePart(rr, req)

			if rr.Code != 200 {
				t.Errorf("Failed to create part with special characters: %d - %s", rr.Code, rr.Body.String())
				return
			}

			// Verify part can be retrieved
			req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/parts/TEST-%03d", i), nil)
			rr2 := httptest.NewRecorder()
			handleGetPart(rr2, req2, fmt.Sprintf("TEST-%03d", i))

			if rr2.Code != 200 {
				t.Errorf("Failed to retrieve part with special characters: %d", rr2.Code)
				return
			}

			var resp struct {
				Data Part `json:"data"`
			}
			json.NewDecoder(rr2.Body).Decode(&resp)

			if resp.Data.Fields["description"] != desc {
				t.Errorf("Description corrupted: expected %q, got %q", desc, resp.Data.Fields["description"])
			}
		})
	}
}

// TestBOM_VisualizationDepthControl tests BOM depth limiting for visualization
func TestBOM_VisualizationDepthControl(t *testing.T) {
	t.Skip("BOM depth parameter not supported in API - hardcoded to maxDepth=5")

	// Expected behavior:
	// - GET /api/v1/parts/:ipn/bom?depth=2
	// - Limit expansion to specified depth
	// - Useful for large/complex BOMs
}

// TestParts_EmptyCategory tests handling of empty categories
func TestParts_EmptyCategory(t *testing.T) {
	oldPartsDir := partsDir
	defer func() { partsDir = oldPartsDir }()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create empty category
	os.WriteFile(filepath.Join(tmpDir, "empty.csv"), []byte("IPN,description\n"), 0644)

	req := httptest.NewRequest("GET", "/api/v1/parts?category=empty", nil)
	rr := httptest.NewRecorder()
	handleListParts(rr, req)

	if rr.Code != 200 {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Data []Part `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	if len(resp.Data) != 0 {
		t.Errorf("Expected 0 parts, got %d", len(resp.Data))
	}

	if resp.Meta.Total != 0 {
		t.Errorf("Expected total=0, got %d", resp.Meta.Total)
	}
}
