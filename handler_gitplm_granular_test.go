package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupGitPLMTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create app_settings table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create app_settings table: %v", err)
	}

	// Create audit_log table - CRITICAL: Used by almost every handler
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			username TEXT,
			action TEXT,
			table_name TEXT,
			record_id TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

func TestHandleGetGitPLMConfig_NoConfig(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupGitPLMTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/settings/gitplm", nil)
	w := httptest.NewRecorder()

	handleGetGitPLMConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}

	configData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %T", apiResp.Data)
	}

	baseURL, _ := configData["base_url"].(string)
	if baseURL != "" {
		t.Errorf("Expected empty BaseURL, got %q", baseURL)
	}
}

func TestHandleGetGitPLMConfig_WithConfig(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupGitPLMTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO app_settings (key, value) VALUES ('gitplm_base_url', 'https://gitplm.example.com')")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/settings/gitplm", nil)
	w := httptest.NewRecorder()

	handleGetGitPLMConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}

	configData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %T", apiResp.Data)
	}

	baseURL, ok := configData["base_url"].(string)
	if !ok || baseURL != "https://gitplm.example.com" {
		t.Errorf("Expected URL 'https://gitplm.example.com', got %q", baseURL)
	}
}

func TestHandleUpdateGitPLMConfig_Create(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupGitPLMTestDB(t)
	defer db.Close()

	config := GitPLMConfig{BaseURL: "https://gitplm.example.com"}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest("PUT", "/api/settings/gitplm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateGitPLMConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}

	configData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %T", apiResp.Data)
	}

	baseURL, ok := configData["base_url"].(string)
	if !ok {
		t.Fatalf("base_url is not a string: %T", configData["base_url"])
	}

	if baseURL != "https://gitplm.example.com" {
		t.Errorf("Expected URL 'https://gitplm.example.com', got %q", baseURL)
	}
}

func TestHandleUpdateGitPLMConfig_TrimSlash(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupGitPLMTestDB(t)
	defer db.Close()

	config := GitPLMConfig{BaseURL: "https://gitplm.example.com/"}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest("PUT", "/api/settings/gitplm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateGitPLMConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}

	configData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %T", apiResp.Data)
	}

	baseURL, ok := configData["base_url"].(string)
	if !ok || baseURL != "https://gitplm.example.com" {
		t.Errorf("Expected trailing slash trimmed, got %q", baseURL)
	}
}

func TestHandleGetGitPLMURL_NotConfigured(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupGitPLMTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/parts/RES-0001/gitplm-url", nil)
	w := httptest.NewRecorder()

	handleGetGitPLMURL(w, req, "RES-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}

	responseData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %T", apiResp.Data)
	}

	configured, _ := responseData["configured"].(bool)
	url, _ := responseData["url"].(string)

	if configured {
		t.Error("Expected Configured=false when no config")
	}
	if url != "" {
		t.Errorf("Expected empty URL, got %q", url)
	}
}

func TestHandleGetGitPLMURL_Configured(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupGitPLMTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO app_settings (key, value) VALUES ('gitplm_base_url', 'https://gitplm.example.com')")

	req := httptest.NewRequest("GET", "/api/parts/RES-0001/gitplm-url", nil)
	w := httptest.NewRecorder()

	handleGetGitPLMURL(w, req, "RES-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}

	responseData, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %T", apiResp.Data)
	}

	configured, _ := responseData["configured"].(bool)
	url, _ := responseData["url"].(string)

	if !configured {
		t.Error("Expected Configured=true")
	}

	expectedURL := "https://gitplm.example.com/parts/RES-0001"
	if url != expectedURL {
		t.Errorf("Expected URL %q, got %q", expectedURL, url)
	}
}
