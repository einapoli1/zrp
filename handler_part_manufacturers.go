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

type PartManufacturer struct {
	ID           int    `json:"id"`
	PartID       string `json:"part_id"`
	Manufacturer string `json:"manufacturer"`
	MPN          string `json:"mpn"`
	IsPrimary    bool   `json:"is_primary"`
	Approved     bool   `json:"approved"`
	Notes        string `json:"notes"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// GET /api/v1/parts/:ipn/manufacturers
func handleGetPartManufacturers(w http.ResponseWriter, r *http.Request) {
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

	// Query manufacturers ordered by is_primary DESC, created_at ASC
	rows, err := db.Query(`
		SELECT id, part_id, manufacturer, mpn, is_primary, approved, notes, created_at, updated_at
		FROM part_manufacturers
		WHERE part_id = ?
		ORDER BY is_primary DESC, created_at ASC
	`, ipn)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	manufacturers := []PartManufacturer{}
	for rows.Next() {
		var m PartManufacturer
		var isPrimary, approved int
		err := rows.Scan(&m.ID, &m.PartID, &m.Manufacturer, &m.MPN, &isPrimary, &approved, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
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

// POST /api/v1/parts/:ipn/manufacturers
func handleCreatePartManufacturer(w http.ResponseWriter, r *http.Request) {
	ipn := extractPathParam(r.URL.Path, "/api/v1/parts/", "/manufacturers")
	if ipn == "" {
		http.Error(w, "IPN required", http.StatusBadRequest)
		return
	}

	var req struct {
		Manufacturer string `json:"manufacturer"`
		MPN          string `json:"mpn"`
		IsPrimary    bool   `json:"is_primary"`
		Approved     bool   `json:"approved"`
		Notes        string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(req.Manufacturer) == "" {
		http.Error(w, "Manufacturer cannot be empty", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MPN) == "" {
		http.Error(w, "MPN cannot be empty", http.StatusBadRequest)
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
		INSERT INTO part_manufacturers (part_id, manufacturer, mpn, is_primary, approved, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ipn, req.Manufacturer, req.MPN, isPrimaryInt, approvedInt, req.Notes, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
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
	logAudit(db, username, "create", "part_manufacturers", strconv.FormatInt(id, 10), fmt.Sprintf("Added manufacturer %s (%s) to part %s", req.Manufacturer, req.MPN, ipn))

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"id":      id,
			"message": "Manufacturer added successfully",
		},
	})
}

// PUT /api/v1/parts/:ipn/manufacturers/:id
func handleUpdatePartManufacturer(w http.ResponseWriter, r *http.Request) {
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

	// Handle manufacturer
	if manufacturer, ok := req["manufacturer"]; ok {
		mfrStr := manufacturer.(string)
		if strings.TrimSpace(mfrStr) == "" {
			http.Error(w, "Manufacturer cannot be empty", http.StatusBadRequest)
			return
		}
		updates = append(updates, "manufacturer = ?")
		args = append(args, mfrStr)
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

// DELETE /api/v1/parts/:ipn/manufacturers/:id
func handleDeletePartManufacturer(w http.ResponseWriter, r *http.Request) {
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

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Verify manufacturer exists and belongs to this part
	var existingPartID string
	var isPrimary int
	err = tx.QueryRow(`SELECT part_id, is_primary FROM part_manufacturers WHERE id = ?`, id).Scan(&existingPartID, &isPrimary)
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

	// Check if this is the last manufacturer for the part
	var count int
	err = tx.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE part_id = ?`, ipn).Scan(&count)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count <= 1 {
		http.Error(w, "Cannot delete the last manufacturer. Each part must have at least one manufacturer.", http.StatusBadRequest)
		return
	}

	// If deleting primary manufacturer and others exist, promote oldest secondary to primary
	if isPrimary == 1 {
		// SQLite doesn't support LIMIT in UPDATE, so we need to find the ID first
		var promoteID int
		err = tx.QueryRow(`
			SELECT id FROM part_manufacturers 
			WHERE part_id = ? AND id != ? 
			ORDER BY created_at ASC 
			LIMIT 1
		`, ipn, id).Scan(&promoteID)
		if err == nil {
			_, err = tx.Exec(`UPDATE part_manufacturers SET is_primary = 1 WHERE id = ?`, promoteID)
			if err != nil {
				http.Error(w, "Failed to promote secondary manufacturer", http.StatusInternalServerError)
				return
			}
		}
	}

	// Delete the manufacturer
	_, err = tx.Exec(`DELETE FROM part_manufacturers WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete manufacturer", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Audit log (after commit)
	username := getUsername(r)
	logAudit(db, username, "delete", "part_manufacturers", strconv.Itoa(id), fmt.Sprintf("Deleted manufacturer from part %s", ipn))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"message": "Manufacturer deleted successfully",
		},
	})
}

// Helper function to extract path parameter
func extractPathParam(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	path = strings.TrimPrefix(path, prefix)
	if idx := strings.Index(path, suffix); idx != -1 {
		return path[:idx]
	}
	return path
}

// Migration function moved to db_manufacturer_migration.go
