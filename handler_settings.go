package main

import (
	"database/sql"
	"net/http"
)

type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

// handleListSettings returns all settings
func handleListSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT key, value, COALESCE(description, ''), COALESCE(updated_at, '') FROM settings ORDER BY key")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		settings = append(settings, s)
	}
	if settings == nil {
		settings = []Setting{}
	}
	jsonResp(w, settings)
}

// handleGetSetting returns a specific setting by key
func handleGetSetting(w http.ResponseWriter, r *http.Request, key string) {
	var s Setting
	err := db.QueryRow("SELECT key, value, COALESCE(description, ''), COALESCE(updated_at, '') FROM settings WHERE key = ?", key).
		Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		jsonErr(w, "setting not found", 404)
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, s)
}

// handleUpdateSetting updates a setting's value
func handleUpdateSetting(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Value string `json:"value"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid request body", 400)
		return
	}

	// Verify setting exists
	var exists int
	err := db.QueryRow("SELECT 1 FROM settings WHERE key = ?", key).Scan(&exists)
	if err == sql.ErrNoRows {
		jsonErr(w, "setting not found", 404)
		return
	}
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Validate value for specific settings
	if key == "require_eco_approval_for_creation" {
		if body.Value != "true" && body.Value != "false" {
			jsonErr(w, "value must be 'true' or 'false'", 400)
			return
		}
	}

	// Update the setting
	_, err = db.Exec("UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?", body.Value, key)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Return updated setting
	var s Setting
	db.QueryRow("SELECT key, value, COALESCE(description, ''), COALESCE(updated_at, '') FROM settings WHERE key = ?", key).
		Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt)
	jsonResp(w, s)
}

// getSetting retrieves a setting value by key (helper function)
func getSetting(key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	return value, err
}
