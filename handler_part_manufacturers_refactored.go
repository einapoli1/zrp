package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// PartManufacturerRefactored uses normalized manufacturer_id instead of TEXT manufacturer
type PartManufacturerRefactored struct {
	ID             int    `json:"id"`
	PartID         string `json:"part_id"`
	ManufacturerID int    `json:"manufacturer_id"`
	Manufacturer   string `json:"manufacturer_name"` // Joined from manufacturers table
	MPN            string `json:"mpn"`
	IsPrimary      bool   `json:"is_primary"`
	Approved       bool   `json:"approved"`
	Notes          string `json:"notes"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// GET /api/v1/parts/:ipn/manufacturers (refactored to JOIN with manufacturers table)
func handleGetPartManufacturersRefactored(w http.ResponseWriter, r *http.Request) {
	ipn := extractPathParam(r.URL.Path, "/api/v1/parts/", "/manufacturers")
	if ipn == "" {
		http.Error(w, "IPN required", http.StatusBadRequest)
		return
	}

	// Verify part exists
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM parts WHERE ipn = ?)`, ipn).Scan(&exists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Part not found", http.StatusNotFound)
		return
	}

	// Query manufacturers with JOIN to manufacturers table
	rows, err := db.Query(`
		SELECT 
			pm.id, pm.part_id, pm.manufacturer_id, m.name, pm.mpn, 
			pm.is_primary, pm.approved, pm.notes, pm.created_at, pm.updated_at
		FROM part_manufacturers pm
		JOIN manufacturers m ON pm.manufacturer_id = m.id
		WHERE pm.part_id = ?
		ORDER BY pm.is_primary DESC, pm.created_at ASC
	`, ipn)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	manufacturers := []PartManufacturerRefactored{}
	for rows.Next() {
		var m PartManufacturerRefactored
		var isPrimary, approved int
		err := rows.Scan(&m.ID, &m.PartID, &m.ManufacturerID, &m.Manufacturer, &m.MPN, &isPrimary, &approved, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan manufacturer", http.StatusInternalServerError)
			return
		}
		m.IsPrimary = isPrimary == 1
		m.Approved = approved == 1
		manufacturers = append(manufacturers, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"manufacturers": manufacturers,
			"count":         len(manufacturers),
		},
	})
}

// POST /api/v1/parts/:ipn/manufacturers (refactored to use manufacturer_id)
func handleCreatePartManufacturerRefactored(w http.ResponseWriter, r *http.Request) {
	ipn := extractPathParam(r.URL.Path, "/api/v1/parts/", "/manufacturers")
	if ipn == "" {
		http.Error(w, "IPN required", http.StatusBadRequest)
		return
	}

	var req struct {
		ManufacturerID int    `json:"manufacturer_id"`
		MPN            string `json:"mpn"`
		IsPrimary      bool   `json:"is_primary"`
		Approved       bool   `json:"approved"`
		Notes          string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if req.ManufacturerID <= 0 {
		http.Error(w, "Valid manufacturer_id is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MPN) == "" {
		http.Error(w, "MPN cannot be empty", http.StatusBadRequest)
		return
	}

	// Verify manufacturer exists
	var mfrExists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM manufacturers WHERE id = ?)`, req.ManufacturerID).Scan(&mfrExists)
	if err != nil || !mfrExists {
		http.Error(w, "Manufacturer not found", http.StatusNotFound)
		return
	}

	// Verify part exists
	var partExists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM parts WHERE ipn = ?)`, ipn).Scan(&partExists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !partExists {
		http.Error(w, "Part not found", http.StatusNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// If is_primary=true, unset all other manufacturers for this part
	if req.IsPrimary {
		_, err = tx.Exec(`UPDATE part_manufacturers SET is_primary = 0 WHERE part_id = ?`, ipn)
		if err != nil {
			http.Error(w, "Failed to update primary status", http.StatusInternalServerError)
			return
		}
	}

	// Insert new manufacturer
	isPrimaryInt := 0
	if req.IsPrimary {
		isPrimaryInt = 1
	}
	approvedInt := 0
	if req.Approved {
		approvedInt = 1
	}

	result, err := tx.Exec(`
		INSERT INTO part_manufacturers (part_id, manufacturer_id, mpn, is_primary, approved, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ipn, req.ManufacturerID, req.MPN, isPrimaryInt, approvedInt, req.Notes, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			http.Error(w, "Duplicate manufacturer/MPN combination for this part", http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to create manufacturer: %v", err), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Audit log (after commit)
	username := getUsername(r)
	logAudit(db, username, "create", "part_manufacturers", strconv.FormatInt(id, 10), fmt.Sprintf("Added manufacturer ID %d (MPN: %s) to part %s", req.ManufacturerID, req.MPN, ipn))

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"id":      id,
			"message": "Manufacturer added successfully",
		},
	})
}

// PUT /api/v1/parts/:ipn/manufacturers/:id (refactored to use manufacturer_id)
func handleUpdatePartManufacturerRefactored(w http.ResponseWriter, r *http.Request) {
	ipn := extractPathParam(r.URL.Path, "/api/v1/parts/", "/manufacturers/")
	if ipn == "" {
		http.Error(w, "IPN required", http.StatusBadRequest)
		return
	}

	// Extract manufacturer ID from path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/parts/"+ipn+"/manufacturers/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid manufacturer ID", http.StatusBadRequest)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify manufacturer exists and belongs to this part
	var existingPartID string
	err = db.QueryRow(`SELECT part_id FROM part_manufacturers WHERE id = ?`, id).Scan(&existingPartID)
	if err == sql.ErrNoRows {
		http.Error(w, "Manufacturer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if existingPartID != ipn {
		http.Error(w, "Manufacturer does not belong to this part", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	// Handle manufacturer_id change
	if manufacturerID, ok := req["manufacturer_id"]; ok {
		mfrID := int(manufacturerID.(float64))
		if mfrID <= 0 {
			http.Error(w, "Invalid manufacturer_id", http.StatusBadRequest)
			return
		}
		
		// Verify new manufacturer exists
		var exists bool
		tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM manufacturers WHERE id = ?)`, mfrID).Scan(&exists)
		if !exists {
			http.Error(w, "Manufacturer not found", http.StatusNotFound)
			return
		}
		
		updates = append(updates, "manufacturer_id = ?")
		args = append(args, mfrID)
	}

	// Handle is_primary
	if isPrimary, ok := req["is_primary"]; ok {
		if isPrimary.(bool) {
			// Unset all other manufacturers for this part
			_, err = tx.Exec(`UPDATE part_manufacturers SET is_primary = 0 WHERE part_id = ? AND id != ?`, ipn, id)
			if err != nil {
				http.Error(w, "Failed to update primary status", http.StatusInternalServerError)
				return
			}
			updates = append(updates, "is_primary = ?")
			args = append(args, 1)
		} else {
			updates = append(updates, "is_primary = ?")
			args = append(args, 0)
		}
	}

	// Handle approved
	if approved, ok := req["approved"]; ok {
		approvedInt := 0
		if approved.(bool) {
			approvedInt = 1
		}
		updates = append(updates, "approved = ?")
		args = append(args, approvedInt)
	}

	// Handle mpn
	if mpn, ok := req["mpn"]; ok {
		mpnStr := mpn.(string)
		if strings.TrimSpace(mpnStr) == "" {
			http.Error(w, "MPN cannot be empty", http.StatusBadRequest)
			return
		}
		updates = append(updates, "mpn = ?")
		args = append(args, mpnStr)
	}

	// Handle notes
	if notes, ok := req["notes"]; ok {
		updates = append(updates, "notes = ?")
		args = append(args, notes.(string))
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	// Add updated_at
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now().Format(time.RFC3339))

	// Add ID for WHERE clause
	args = append(args, id)

	query := fmt.Sprintf("UPDATE part_manufacturers SET %s WHERE id = ?", strings.Join(updates, ", "))
	_, err = tx.Exec(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update manufacturer: %v", err), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Audit log (after commit)
	username := getUsername(r)
	logAudit(db, username, "update", "part_manufacturers", strconv.Itoa(id), fmt.Sprintf("Updated manufacturer for part %s", ipn))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"message": "Manufacturer updated successfully",
		},
	})
}

// DELETE remains the same - no changes needed
// (Reusing handleDeletePartManufacturer from original file)
