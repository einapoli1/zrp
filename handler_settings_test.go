package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestGetAllSettings(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Seed a test setting
	db.Exec("INSERT INTO settings (key, value, description) VALUES (?, ?, ?)",
		"test_setting", "test_value", "Test description")

	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	rec := httptest.NewRecorder()

	handleListSettings(rec, req)

	bodyStr := rec.Body.String()
	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, bodyStr)
		return
	}

	var resp struct {
		Data []Setting `json:"data"`
	}
	if err := json.Unmarshal([]byte(bodyStr), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v, body: %s", err, bodyStr)
	}
	settings := resp.Data

	if len(settings) == 0 {
		t.Error("Expected at least one setting")
	}

	// Verify the default setting exists
	found := false
	for _, s := range settings {
		if s.Key == "require_eco_approval_for_creation" {
			found = true
			if s.Value != "false" {
				t.Errorf("Expected default value 'false', got %s", s.Value)
			}
		}
	}
	if !found {
		t.Error("Expected to find require_eco_approval_for_creation setting")
	}
}

func TestGetSpecificSetting(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/settings/require_eco_approval_for_creation", nil)
	rec := httptest.NewRecorder()

	handleGetSetting(rec, req, "require_eco_approval_for_creation")

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		return
	}

	var resp struct {
		Data Setting `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v, body: %s", err, rec.Body.String())
	}
	setting := resp.Data

	if setting.Key != "require_eco_approval_for_creation" {
		t.Errorf("Expected key 'require_eco_approval_for_creation', got %s", setting.Key)
	}

	if setting.Value != "false" {
		t.Errorf("Expected default value 'false', got %s", setting.Value)
	}
}

func TestUpdateSetting(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	body := map[string]string{"value": "true"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/settings/require_eco_approval_for_creation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateSetting(rec, req, "require_eco_approval_for_creation")

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		return
	}

	var resp struct {
		Data Setting `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	setting := resp.Data

	if setting.Value != "true" {
		t.Errorf("Expected updated value 'true', got %s", setting.Value)
	}

	// Verify in database
	var dbValue string
	db.QueryRow("SELECT value FROM settings WHERE key = ?", "require_eco_approval_for_creation").Scan(&dbValue)
	if dbValue != "true" {
		t.Errorf("Expected database value 'true', got %s", dbValue)
	}
}

func TestGetSettingInvalidKey(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/settings/nonexistent_key", nil)
	rec := httptest.NewRecorder()

	handleGetSetting(rec, req, "nonexistent_key")

	if rec.Code != 404 {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdateSettingInvalidValue(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	// Try to set invalid value for boolean setting
	body := map[string]string{"value": "invalid"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/settings/require_eco_approval_for_creation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateSetting(rec, req, "require_eco_approval_for_creation")

	if rec.Code != 400 {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestUpdateSettingNonexistentKey(t *testing.T) {
	db = setupTestDB(t)
	defer db.Close()

	body := map[string]string{"value": "test"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/settings/nonexistent_key", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateSetting(rec, req, "nonexistent_key")

	if rec.Code != 404 {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}
