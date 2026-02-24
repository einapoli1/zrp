package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// BOMItem represents a single line item in a BOM
type BOMItem struct {
	IPN         string  `json:"ipn"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	RefDes      string  `json:"ref_des"`
}

// BOMSaveRequest represents the request to save a BOM
type BOMSaveRequest struct {
	AssemblyIPN string    `json:"assembly_ipn"`
	Items       []BOMItem `json:"items"`
}

// handleSaveBOM saves a BOM for an assembly part
// POST /api/v1/parts/:ipn/bom
func handleSaveBOM(w http.ResponseWriter, r *http.Request, ipn string) {
	if partsDir == "" {
		jsonErr(w, "parts directory not configured", 500)
		return
	}

	var req BOMSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid request body", 400)
		return
	}

	if req.AssemblyIPN == "" {
		req.AssemblyIPN = ipn
	}

	// Validate assembly IPN matches route parameter
	if req.AssemblyIPN != ipn {
		jsonErr(w, "assembly IPN mismatch", 400)
		return
	}

	// Validate that assembly IPN is actually an assembly
	upperIPN := strings.ToUpper(ipn)
	if !strings.HasPrefix(upperIPN, "PCA-") && !strings.HasPrefix(upperIPN, "ASY-") {
		jsonErr(w, "BOM can only be saved for assembly IPNs (PCA-, ASY- prefix)", 400)
		return
	}

	// Validate that all IPNs in the BOM exist in the parts database
	if err := validateBOMIPNs(req.Items); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	// Save BOM to CSV file
	if err := saveBOMToCSV(ipn, req.Items); err != nil {
		jsonErr(w, fmt.Sprintf("failed to save BOM: %v", err), 500)
		return
	}

	jsonResp(w, map[string]interface{}{
		"success":      true,
		"assembly_ipn": ipn,
		"item_count":   len(req.Items),
	})
}

// validateBOMIPNs checks that all IPNs in the BOM exist in the parts database
func validateBOMIPNs(items []BOMItem) error {
	if len(items) == 0 {
		return nil
	}

	// Load all parts from the database
	cats, _, _, err := loadPartsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load parts database: %w", err)
	}

	// Build a set of all valid IPNs
	validIPNs := make(map[string]bool)
	for _, parts := range cats {
		for _, p := range parts {
			validIPNs[p.IPN] = true
		}
	}

	// Check each BOM item
	for _, item := range items {
		if item.IPN == "" {
			continue // Skip empty IPNs
		}
		if !validIPNs[item.IPN] {
			return fmt.Errorf("part %s not found", item.IPN)
		}
	}

	return nil
}

// saveBOMToCSV writes BOM items to a CSV file
func saveBOMToCSV(assemblyIPN string, items []BOMItem) error {
	// Determine BOM file path
	bomPath := filepath.Join(partsDir, assemblyIPN+".csv")

	// Create or overwrite the file
	f, err := os.Create(bomPath)
	if err != nil {
		return fmt.Errorf("failed to create BOM file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write headers
	if err := writer.Write([]string{"IPN", "description", "qty", "ref"}); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write BOM items
	for _, item := range items {
		if item.IPN == "" {
			continue // Skip empty items
		}

		row := []string{
			item.IPN,
			item.Description,
			fmt.Sprintf("%.0f", item.Quantity),
			item.RefDes,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write BOM item: %w", err)
		}
	}

	return nil
}

// handleUpdateBOM updates an existing BOM
// PUT /api/v1/parts/:ipn/bom
func handleUpdateBOM(w http.ResponseWriter, r *http.Request, ipn string) {
	// For now, update is the same as save (overwrites)
	handleSaveBOM(w, r, ipn)
}
