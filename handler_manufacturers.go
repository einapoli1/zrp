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

type Manufacturer struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	ContactName  string `json:"contact_name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Website      string `json:"website"`
	Notes        string `json:"notes"`
	Approved     bool   `json:"approved"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// GET /api/v1/manufacturers - List all manufacturers with optional search/filter
func handleListManufacturers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	search := r.URL.Query().Get("search")
	approvedOnly := r.URL.Query().Get("approved") == "true"
	limit := 100 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Build query
	query := `SELECT id, name, contact_name, contact_email, contact_phone, website, notes, approved, created_at, updated_at FROM manufacturers WHERE 1=1`
	args := []interface{}{}

	if search != "" {
		query += ` AND LOWER(name) LIKE LOWER(?)`
		args = append(args, "%"+search+"%")
	}

	if approvedOnly {
		query += ` AND approved = 1`
	}

	query += ` ORDER BY name ASC LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	manufacturers := []Manufacturer{}
	for rows.Next() {
		var m Manufacturer
		var approved int
		err := rows.Scan(&m.ID, &m.Name, &m.ContactName, &m.ContactEmail, &m.ContactPhone, &m.Website, &m.Notes, &approved, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan manufacturer", http.StatusInternalServerError)
			return
		}
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

// GET /api/v1/manufacturers/:id - Get one manufacturer
func handleGetManufacturer(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/manufacturers/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid manufacturer ID", http.StatusBadRequest)
		return
	}

	var m Manufacturer
	var approved int
	err = db.QueryRow(`
		SELECT id, name, contact_name, contact_email, contact_phone, website, notes, approved, created_at, updated_at
		FROM manufacturers WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.ContactName, &m.ContactEmail, &m.ContactPhone, &m.Website, &m.Notes, &approved, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Manufacturer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	m.Approved = approved == 1

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: m,
	})
}

// POST /api/v1/manufacturers - Create manufacturer with duplicate detection and fuzzy matching
func handleCreateManufacturer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		ContactName  string `json:"contact_name"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
		Website      string `json:"website"`
		Notes        string `json:"notes"`
		Approved     bool   `json:"approved"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Manufacturer name is required", http.StatusBadRequest)
		return
	}

	// Check for EXACT match (case-insensitive)
	var existingID int
	var existingName string
	err := db.QueryRow(`SELECT id, name FROM manufacturers WHERE LOWER(name) = LOWER(?)`, req.Name).Scan(&existingID, &existingName)
	if err == nil {
		// Exact match found
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "Duplicate manufacturer name",
			"existing_id":   existingID,
			"existing_name": existingName,
			"match_type":    "exact",
		})
		return
	}

	// Check for FUZZY matches (Levenshtein distance < 3)
	rows, err := db.Query(`SELECT id, name FROM manufacturers`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	suggestions := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}

		distance := levenshteinDistance(req.Name, name)
		if distance > 0 && distance < 3 { // Close match (1-2 character difference)
			suggestions = append(suggestions, map[string]interface{}{
				"id":       id,
				"name":     name,
				"distance": distance,
			})
		}
	}

	// If fuzzy matches found, suggest instead of creating
	if len(suggestions) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "Similar manufacturer names found",
			"match_type":  "fuzzy",
			"suggestions": suggestions,
		})
		return
	}

	// No duplicates or close matches, create new manufacturer
	approvedInt := 0
	if req.Approved {
		approvedInt = 1
	}

	result, err := db.Exec(`
		INSERT INTO manufacturers (name, contact_name, contact_email, contact_phone, website, notes, approved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.Name, req.ContactName, req.ContactEmail, req.ContactPhone, req.Website, req.Notes, approvedInt, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create manufacturer: %v", err), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	// Audit log
	username := getUsername(r)
	logAudit(db, username, "create", "manufacturers", strconv.FormatInt(id, 10), fmt.Sprintf("Created manufacturer: %s", req.Name))

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"id":      id,
			"message": "Manufacturer created successfully",
		},
	})
}

// PUT /api/v1/manufacturers/:id - Update manufacturer
func handleUpdateManufacturer(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/manufacturers/")
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

	// Verify manufacturer exists
	var exists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM manufacturers WHERE id = ?)`, id).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Manufacturer not found", http.StatusNotFound)
		return
	}

	// If updating name, check for duplicates
	if name, ok := req["name"]; ok {
		nameStr := name.(string)
		if strings.TrimSpace(nameStr) == "" {
			http.Error(w, "Manufacturer name cannot be empty", http.StatusBadRequest)
			return
		}

		// Check for existing manufacturer with same name (excluding self)
		var existingID int
		err = db.QueryRow(`SELECT id FROM manufacturers WHERE LOWER(name) = LOWER(?) AND id != ?`, nameStr, id).Scan(&existingID)
		if err == nil {
			http.Error(w, "Manufacturer name already exists", http.StatusConflict)
			return
		}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	if name, ok := req["name"]; ok {
		updates = append(updates, "name = ?")
		args = append(args, name.(string))
	}
	if contactName, ok := req["contact_name"]; ok {
		updates = append(updates, "contact_name = ?")
		args = append(args, contactName.(string))
	}
	if contactEmail, ok := req["contact_email"]; ok {
		updates = append(updates, "contact_email = ?")
		args = append(args, contactEmail.(string))
	}
	if contactPhone, ok := req["contact_phone"]; ok {
		updates = append(updates, "contact_phone = ?")
		args = append(args, contactPhone.(string))
	}
	if website, ok := req["website"]; ok {
		updates = append(updates, "website = ?")
		args = append(args, website.(string))
	}
	if notes, ok := req["notes"]; ok {
		updates = append(updates, "notes = ?")
		args = append(args, notes.(string))
	}
	if approved, ok := req["approved"]; ok {
		approvedInt := 0
		if approved.(bool) {
			approvedInt = 1
		}
		updates = append(updates, "approved = ?")
		args = append(args, approvedInt)
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

	query := fmt.Sprintf("UPDATE manufacturers SET %s WHERE id = ?", strings.Join(updates, ", "))
	_, err = db.Exec(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update manufacturer: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit log
	username := getUsername(r)
	logAudit(db, username, "update", "manufacturers", strconv.Itoa(id), "Updated manufacturer")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"message": "Manufacturer updated successfully",
		},
	})
}

// DELETE /api/v1/manufacturers/:id - Delete manufacturer (only if no parts reference it)
func handleDeleteManufacturer(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/manufacturers/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid manufacturer ID", http.StatusBadRequest)
		return
	}

	// Verify manufacturer exists
	var mfrName string
	err = db.QueryRow(`SELECT name FROM manufacturers WHERE id = ?`, id).Scan(&mfrName)
	if err == sql.ErrNoRows {
		http.Error(w, "Manufacturer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if any parts reference this manufacturer
	var partCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE manufacturer_id = ?`, id).Scan(&partCount)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if partCount > 0 {
		http.Error(w, fmt.Sprintf("Cannot delete manufacturer: %d parts reference it", partCount), http.StatusConflict)
		return
	}

	// Delete manufacturer
	_, err = db.Exec(`DELETE FROM manufacturers WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete manufacturer", http.StatusInternalServerError)
		return
	}

	// Audit log
	username := getUsername(r)
	logAudit(db, username, "delete", "manufacturers", strconv.Itoa(id), fmt.Sprintf("Deleted manufacturer: %s", mfrName))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Data: map[string]interface{}{
			"message": "Manufacturer deleted successfully",
		},
	})
}
