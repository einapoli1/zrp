package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func unmarshalConfigResponse(t *testing.T, body []byte) ConfigurationTemplate {
	var resp struct {
		Data ConfigurationTemplate `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, string(body))
	}
	return resp.Data
}

func unmarshalConfigParameterResponse(t *testing.T, body []byte) ConfigurationParameter {
	var resp struct {
		Data ConfigurationParameter `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, string(body))
	}
	return resp.Data
}

func unmarshalConfigPartResponse(t *testing.T, body []byte) ConfigurationPart {
	var resp struct {
		Data ConfigurationPart `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, string(body))
	}
	return resp.Data
}

func setupConfiguratorTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create configuration_templates table
	_, err = testDB.Exec(`
		CREATE TABLE configuration_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			model_format TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create configuration_templates table: %v", err)
	}

	// Create configuration_parameters table
	_, err = testDB.Exec(`
		CREATE TABLE configuration_parameters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('enum','range')),
			values_json TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (template_id) REFERENCES configuration_templates(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create configuration_parameters table: %v", err)
	}

	// Create configuration_parts table
	_, err = testDB.Exec(`
		CREATE TABLE configuration_parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id INTEGER NOT NULL,
			ipn TEXT NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1 CHECK(quantity > 0),
			include_all_variants INTEGER NOT NULL DEFAULT 0 CHECK(include_all_variants IN (0,1)),
			constraints_json TEXT DEFAULT '{}',
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (template_id) REFERENCES configuration_templates(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create configuration_parts table: %v", err)
	}

	// Create configuration_generations table
	_, err = testDB.Exec(`
		CREATE TABLE configuration_generations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id INTEGER NOT NULL,
			eco_id TEXT NOT NULL,
			generated_at TEXT DEFAULT (datetime('now')),
			variant_count INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (template_id) REFERENCES configuration_templates(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create configuration_generations table: %v", err)
	}

	// Create parts table for validation
	_, err = testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			description TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	// Create ecos table for generation
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			type TEXT DEFAULT 'change',
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_at TEXT,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			action TEXT,
			module TEXT,
			record_id TEXT,
			summary TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Create id_sequences table for nextID
	_, err = testDB.Exec(`
		CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences table: %v", err)
	}

	return testDB
}

// Template CRUD Tests (5 tests)

func TestCreateConfigTemplate(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	template := map[string]interface{}{
		"name":         "uATS 1.2kVA",
		"model_format": "PCA-uATS-{voltage}-{amperage}-{length}",
	}
	body, _ := json.Marshal(template)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	handleCreateConfigTemplate(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data ConfigurationTemplate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}
	result := resp.Data
	if result.Name != "uATS 1.2kVA" {
		t.Errorf("Expected name 'uATS 1.2kVA', got '%s'", result.Name)
	}
	if result.ModelFormat != "PCA-uATS-{voltage}-{amperage}-{length}" {
		t.Errorf("Expected model_format with placeholders, got '%s'", result.ModelFormat)
	}
}

func TestCreateConfigTemplateValidation(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	// Missing model_format placeholder
	template := map[string]interface{}{
		"name":         "Bad Template",
		"model_format": "NO-PLACEHOLDERS",
	}
	body, _ := json.Marshal(template)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	handleCreateConfigTemplate(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid model_format, got %d", w.Code)
	}
}

func TestGetConfigTemplate(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	// Create template first
	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")

	req := httptest.NewRequest("GET", "/api/v1/configurator/templates/1", nil)
	w := httptest.NewRecorder()

	handleGetConfigTemplate(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// var result ConfigurationTemplate
	result := unmarshalConfigResponse(t, w.Body.Bytes())
	if result.Name != "Test" {
		t.Errorf("Expected name 'Test', got '%s'", result.Name)
	}
}

func TestUpdateConfigTemplate(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	// Create template first
	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Old', 'PCA-{v}', '2026-01-01', '2026-01-01')")

	update := map[string]interface{}{
		"name":         "Updated Name",
		"model_format": "NEW-{param}",
	}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/v1/configurator/templates/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	handleUpdateConfigTemplate(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// var result ConfigurationTemplate
	result := unmarshalConfigResponse(t, w.Body.Bytes())
	if result.Name != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", result.Name)
	}
}

func TestDeleteConfigTemplate(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	// Create template first
	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'ToDelete', 'PCA-{v}', '2026-01-01', '2026-01-01')")

	req := httptest.NewRequest("DELETE", "/api/v1/configurator/templates/1", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	handleDeleteConfigTemplate(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM configuration_templates WHERE id=1").Scan(&count)
	if count != 0 {
		t.Error("Template not deleted")
	}
}

// Parameter CRUD Tests (5 tests)

func TestCreateConfigParameter(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	// Create template first
	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{voltage}', '2026-01-01', '2026-01-01')")

	param := map[string]interface{}{
		"name":        "voltage",
		"type":        "enum",
		"values_json": `["120V","208V","240V"]`,
	}
	body, _ := json.Marshal(param)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/parameters", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	handleCreateConfigParameter(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// var result ConfigurationParameter
	result := unmarshalConfigParameterResponse(t, w.Body.Bytes())
	if result.Name != "voltage" {
		t.Errorf("Expected name 'voltage', got '%s'", result.Name)
	}
	if result.Type != "enum" {
		t.Errorf("Expected type 'enum', got '%s'", result.Type)
	}
}

func TestCreateConfigParameterValidation(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")

	// Invalid parameter name with special characters
	param := map[string]interface{}{
		"name":        "volt@ge!",
		"type":        "enum",
		"values_json": `["120V"]`,
	}
	body, _ := json.Marshal(param)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/parameters", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateConfigParameter(w, req, "1")

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid parameter name, got %d", w.Code)
	}
}

func TestCreateRangeParameter(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{amperage}', '2026-01-01', '2026-01-01')")

	param := map[string]interface{}{
		"name":        "amperage",
		"type":        "range",
		"values_json": `{"min":10,"max":100,"unit":"A"}`,
	}
	body, _ := json.Marshal(param)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/parameters", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateConfigParameter(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateConfigParameter(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (id, template_id, name, type, values_json, created_at) VALUES (1, 1, 'voltage', 'enum', '[\"120V\"]', '2026-01-01')")

	update := map[string]interface{}{
		"name":        "voltage",
		"type":        "enum",
		"values_json": `["120V","208V","240V","480V"]`,
	}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/v1/configurator/parameters/1", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleUpdateConfigParameter(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteConfigParameter(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (id, template_id, name, type, values_json, created_at) VALUES (1, 1, 'voltage', 'enum', '[\"120V\"]', '2026-01-01')")

	req := httptest.NewRequest("DELETE", "/api/v1/configurator/parameters/1", nil)
	w := httptest.NewRecorder()

	handleDeleteConfigParameter(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM configuration_parameters WHERE id=1").Scan(&count)
	if count != 0 {
		t.Error("Parameter not deleted")
	}
}

// Part Pool CRUD Tests (5 tests)

func TestCreateConfigPart(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO parts (ipn, description) VALUES ('CAP-001', 'Capacitor')")

	part := map[string]interface{}{
		"ipn":                  "CAP-001",
		"quantity":             2,
		"include_all_variants": 1,
		"constraints_json":     "{}",
	}
	body, _ := json.Marshal(part)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/parts", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateConfigPart(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// var result ConfigurationPart
	result := unmarshalConfigPartResponse(t, w.Body.Bytes())
	if result.IPN != "CAP-001" {
		t.Errorf("Expected IPN 'CAP-001', got '%s'", result.IPN)
	}
	if result.Quantity != 2 {
		t.Errorf("Expected quantity 2, got %d", result.Quantity)
	}
}

func TestCreateConfigPartValidation(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")

	// Non-existent IPN
	part := map[string]interface{}{
		"ipn":      "NONEXISTENT",
		"quantity": 1,
	}
	body, _ := json.Marshal(part)
	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/parts", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateConfigPart(w, req, "1")

	if w.Code != 400 {
		t.Errorf("Expected 400 for non-existent part, got %d", w.Code)
	}
}

func TestUpdateConfigPart(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO parts (ipn, description) VALUES ('CAP-001', 'Capacitor')")
	db.Exec("INSERT INTO configuration_parts (id, template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (1, 1, 'CAP-001', 1, 0, '{}', '2026-01-01')")

	update := map[string]interface{}{
		"quantity":             3,
		"include_all_variants": 1,
		"constraints_json":     `{"voltage":["120V"]}`,
	}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/v1/configurator/parts/1", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleUpdateConfigPart(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// var result ConfigurationPart
	result := unmarshalConfigPartResponse(t, w.Body.Bytes())
	if result.Quantity != 3 {
		t.Errorf("Expected quantity 3, got %d", result.Quantity)
	}
}

func TestDeleteConfigPart(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO parts (ipn, description) VALUES ('CAP-001', 'Capacitor')")
	db.Exec("INSERT INTO configuration_parts (id, template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (1, 1, 'CAP-001', 1, 0, '{}', '2026-01-01')")

	req := httptest.NewRequest("DELETE", "/api/v1/configurator/parts/1", nil)
	w := httptest.NewRecorder()

	handleDeleteConfigPart(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM configuration_parts WHERE id=1").Scan(&count)
	if count != 0 {
		t.Error("Part not deleted")
	}
}

func TestConfigPartCascadeDelete(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO parts (ipn, description) VALUES ('CAP-001', 'Capacitor')")
	db.Exec("INSERT INTO configuration_parts (id, template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (1, 1, 'CAP-001', 1, 0, '{}', '2026-01-01')")

	// Delete template should cascade delete parts
	db.Exec("DELETE FROM configuration_templates WHERE id=1")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM configuration_parts WHERE template_id=1").Scan(&count)
	if count != 0 {
		t.Error("Parts not cascade deleted")
	}
}

// Generation with Enum Parameters (3 tests)

func TestGenerateEnumVariants(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'uATS', 'PCA-uATS-{voltage}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'voltage', 'enum', '[\"120V\",\"208V\"]', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	if len(variants) != 2 {
		t.Errorf("Expected 2 variants, got %d", len(variants))
	}

	ipns := []string{}
	for _, v := range variants {
		ipns = append(ipns, v["ipn"].(string))
	}

	expectedIPNs := []string{"PCA-uATS-120V", "PCA-uATS-208V"}
	for _, expected := range expectedIPNs {
		found := false
		for _, ipn := range ipns {
			if ipn == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected IPN '%s' not found", expected)
		}
	}
}

func TestGenerateMultiEnumVariants(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'uATS', 'PCA-{voltage}-{length}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'voltage', 'enum', '[\"120V\",\"208V\"]', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'length', 'enum', '[\"6ft\",\"10ft\"]', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	// 2 voltages × 2 lengths = 4 variants
	if len(variants) != 4 {
		t.Errorf("Expected 4 variants, got %d", len(variants))
	}
}

func TestGenerateEnumVariantsPreview(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'v', 'enum', '[\"A\",\"B\",\"C\",\"D\",\"E\",\"F\",\"G\",\"H\",\"I\",\"J\",\"K\",\"L\"]', '2026-01-01')")

	// Preview limited to 10
	variants, err := generateVariants(1, 10)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	if len(variants) != 10 {
		t.Errorf("Expected preview of 10 variants, got %d", len(variants))
	}
}

// Generation with Range Parameters (3 tests)

func TestGenerateRangeVariants(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{amperage}A', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'amperage', 'range', '{\"min\":10,\"max\":20,\"unit\":\"A\"}', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	// Range generates min and max as discrete values
	if len(variants) != 2 {
		t.Errorf("Expected 2 variants (min and max), got %d", len(variants))
	}
}

func TestGenerateMixedEnumRangeVariants(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{voltage}-{amperage}A', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'voltage', 'enum', '[\"120V\",\"208V\"]', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'amperage', 'range', '{\"min\":10,\"max\":20}', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	// 2 voltages × 2 amperage (min, max) = 4 variants
	if len(variants) != 4 {
		t.Errorf("Expected 4 variants, got %d", len(variants))
	}
}

func TestGenerateRangeSameMinMax(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{amp}A', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'amp', 'range', '{\"min\":15,\"max\":15}', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	// Same min/max should generate 1 variant
	if len(variants) != 1 {
		t.Errorf("Expected 1 variant when min=max, got %d", len(variants))
	}
}

// "All Variants" Parts Tests (2 tests)

func TestGenerateWithAllVariantsPart(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'v', 'enum', '[\"A\",\"B\"]', '2026-01-01')")
	db.Exec("INSERT INTO parts (ipn, description) VALUES ('COMMON-PART', 'Always included')")
	db.Exec("INSERT INTO configuration_parts (template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (1, 'COMMON-PART', 1, 1, '{}', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	if len(variants) != 2 {
		t.Fatalf("Expected 2 variants, got %d", len(variants))
	}

	// Both variants should have the common part
	for _, v := range variants {
		bom := v["bom"].([]map[string]interface{})
		if len(bom) != 1 {
			t.Errorf("Expected 1 BOM item, got %d", len(bom))
			continue
		}
		if bom[0]["ipn"] != "COMMON-PART" {
			t.Errorf("Expected COMMON-PART, got %s", bom[0]["ipn"])
		}
	}
}

func TestGenerateWithMixedIncludeAllAndConstraints(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'v', 'enum', '[\"120V\",\"208V\"]', '2026-01-01')")
	db.Exec("INSERT INTO parts (ipn, description) VALUES ('COMMON', 'Common'), ('ONLY-120V', 'Only for 120V')")
	db.Exec("INSERT INTO configuration_parts (template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (1, 'COMMON', 1, 1, '{}', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parts (template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (1, 'ONLY-120V', 1, 0, '{\"v\":[\"120V\"]}', '2026-01-01')")

	variants, err := generateVariants(1, 0)
	if err != nil {
		t.Fatalf("Error generating variants: %v", err)
	}

	// Check 120V variant
	var variant120V map[string]interface{}
	for _, v := range variants {
		if v["ipn"].(string) == "PCA-120V" {
			variant120V = v
			break
		}
	}
	if variant120V == nil {
		t.Fatal("120V variant not found")
	}

	bom := variant120V["bom"].([]map[string]interface{})
	if len(bom) != 2 {
		t.Errorf("Expected 2 BOM items for 120V variant, got %d", len(bom))
	}

	// Check 208V variant
	var variant208V map[string]interface{}
	for _, v := range variants {
		if v["ipn"].(string) == "PCA-208V" {
			variant208V = v
			break
		}
	}
	if variant208V == nil {
		t.Fatal("208V variant not found")
	}

	bom208V := variant208V["bom"].([]map[string]interface{})
	if len(bom208V) != 1 {
		t.Errorf("Expected 1 BOM item for 208V variant (COMMON only), got %d", len(bom208V))
	}
}

// Constraint Matching Logic Tests (8 tests)

func TestMatchesConstraintsEnum(t *testing.T) {
	params := map[string]interface{}{"voltage": "120V"}

	// Match
	if !matchesConstraints(`{"voltage":["120V","208V"]}`, params) {
		t.Error("Should match when value is in constraint array")
	}

	// No match
	if matchesConstraints(`{"voltage":["208V","240V"]}`, params) {
		t.Error("Should not match when value not in constraint array")
	}
}

func TestMatchesConstraintsRange(t *testing.T) {
	params := map[string]interface{}{"amperage": 15.0}

	// Within range
	if !matchesConstraints(`{"amperage":{"min":10,"max":20}}`, params) {
		t.Error("Should match when value is within range")
	}

	// Below min
	if matchesConstraints(`{"amperage":{"min":20,"max":30}}`, params) {
		t.Error("Should not match when value is below min")
	}

	// Above max
	if matchesConstraints(`{"amperage":{"min":5,"max":10}}`, params) {
		t.Error("Should not match when value is above max")
	}
}

func TestMatchesConstraintsMultiple(t *testing.T) {
	params := map[string]interface{}{
		"voltage":  "120V",
		"amperage": 15.0,
	}

	// Both match
	if !matchesConstraints(`{"voltage":["120V"],"amperage":{"min":10,"max":20}}`, params) {
		t.Error("Should match when all constraints satisfied")
	}

	// One doesn't match
	if matchesConstraints(`{"voltage":["208V"],"amperage":{"min":10,"max":20}}`, params) {
		t.Error("Should not match when any constraint fails")
	}
}

func TestMatchesConstraintsEmpty(t *testing.T) {
	params := map[string]interface{}{"voltage": "120V"}

	// Empty constraints should always match
	if !matchesConstraints(`{}`, params) {
		t.Error("Empty constraints should match")
	}
	if !matchesConstraints(``, params) {
		t.Error("Empty string constraints should match")
	}
}

func TestMatchesConstraintsMissingParam(t *testing.T) {
	params := map[string]interface{}{"voltage": "120V"}

	// Constraint references param not in variant
	if matchesConstraints(`{"amperage":{"min":10,"max":20}}`, params) {
		t.Error("Should not match when constraint parameter doesn't exist in variant")
	}
}

func TestMatchesConstraintsRangeEdgeCases(t *testing.T) {
	// Exact min
	if !matchesConstraints(`{"amp":{"min":10,"max":20}}`, map[string]interface{}{"amp": 10.0}) {
		t.Error("Should match at exact min")
	}

	// Exact max
	if !matchesConstraints(`{"amp":{"min":10,"max":20}}`, map[string]interface{}{"amp": 20.0}) {
		t.Error("Should match at exact max")
	}
}

func TestMatchesConstraintsIntegerValues(t *testing.T) {
	params := map[string]interface{}{"count": 5}

	// Integer comparison
	if !matchesConstraints(`{"count":{"min":1,"max":10}}`, params) {
		t.Error("Should handle integer parameter values")
	}
}

func TestMatchesConstraintsStringNumericValues(t *testing.T) {
	params := map[string]interface{}{"length": "6"}

	// String with numeric value in range
	if !matchesConstraints(`{"length":{"min":5,"max":10}}`, params) {
		t.Error("Should parse numeric strings for range constraints")
	}
}

// ECO Creation Verification Tests (3 tests)

func TestGenerateCreatesECO(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'uATS 1.2kVA', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'v', 'enum', '[\"120V\",\"208V\"]', '2026-01-01')")

	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/generate", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	handleGenerateConfigVariants(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	result := resp.Data

	ecoID, ok := result["eco_id"].(string)
	if !ok {
		t.Fatalf("eco_id not found in response or wrong type: %+v", result)
	}
	if ecoID == "" {
		t.Fatal("ECO ID not returned")
	}

	// Verify ECO exists
	var title, status string
	err := db.QueryRow("SELECT title, status FROM ecos WHERE id=?", ecoID).Scan(&title, &status)
	if err != nil {
		t.Fatalf("ECO not created: %v", err)
	}

	if status != "pending" {
		t.Errorf("Expected ECO status 'pending', got '%s'", status)
	}

	if !strings.Contains(title, "uATS 1.2kVA") {
		t.Errorf("ECO title should contain template name, got '%s'", title)
	}
}

func TestGenerateRecordsGeneration(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'v', 'enum', '[\"A\",\"B\"]', '2026-01-01')")

	req := httptest.NewRequest("POST", "/api/v1/configurator/templates/1/generate", nil)
	w := httptest.NewRecorder()

	handleGenerateConfigVariants(w, req, "1")

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	// Verify generation record
	var count, variantCount int
	db.QueryRow("SELECT COUNT(*), MAX(variant_count) FROM configuration_generations WHERE template_id=1").Scan(&count, &variantCount)

	if count != 1 {
		t.Errorf("Expected 1 generation record, got %d", count)
	}
	if variantCount != 2 {
		t.Errorf("Expected variant_count=2, got %d", variantCount)
	}
}

func TestGeneratePreviewDoesNotCreateECO(t *testing.T) {
	db = setupConfiguratorTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO configuration_templates (id, name, model_format, created_at, updated_at) VALUES (1, 'Test', 'PCA-{v}', '2026-01-01', '2026-01-01')")
	db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (1, 'v', 'enum', '[\"A\",\"B\"]', '2026-01-01')")

	req := httptest.NewRequest("GET", "/api/v1/configurator/templates/1/preview", nil)
	w := httptest.NewRecorder()

	handlePreviewConfigVariants(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Verify no ECO was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&count)
	if count != 0 {
		t.Error("Preview should not create ECO")
	}

	// Verify no generation record
	db.QueryRow("SELECT COUNT(*) FROM configuration_generations").Scan(&count)
	if count != 0 {
		t.Error("Preview should not create generation record")
	}
}
