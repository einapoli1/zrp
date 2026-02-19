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

// Test helper to set up database
func setupInputValidationDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create minimal schema for testing
	schemas := []string{
		`CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			customer TEXT,
			location TEXT,
			status TEXT DEFAULT 'active',
			install_date TEXT,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			ipn TEXT,
			serial_number TEXT,
			defect_type TEXT,
			severity TEXT DEFAULT 'minor',
			status TEXT DEFAULT 'open',
			root_cause TEXT,
			corrective_action TEXT,
			created_by TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL CHECK(qty > 0),
			qty_good INTEGER,
			qty_scrap INTEGER,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			location TEXT,
			description TEXT,
			mpn TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE field_reports (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			report_type TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			priority TEXT DEFAULT 'medium',
			customer_name TEXT,
			site_location TEXT,
			device_ipn TEXT,
			device_serial TEXT,
			reported_by TEXT,
			reported_at TEXT,
			description TEXT,
			root_cause TEXT,
			resolution TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_by TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE capas (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			root_cause TEXT,
			corrective_action TEXT,
			preventive_action TEXT,
			created_by TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE rmas (
			id TEXT PRIMARY KEY,
			serial_number TEXT NOT NULL,
			customer TEXT,
			reason TEXT,
			status TEXT DEFAULT 'open',
			defect_description TEXT,
			resolution TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user',
			password_hash TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, schema := range schemas {
		if _, err := testDB.Exec(schema); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	return testDB
}

// Test Vendor Name Field - Max 255 chars
func TestVendorNameLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name       string
		vendorName string
		wantErr    bool
	}{
		{"Empty name", "", true}, // Required field
		{"Single char", "A", false},
		{"Max valid (255)", strings.Repeat("A", 255), false},
		{"Over max (256)", strings.Repeat("A", 256), true},
		{"Way over max (10k)", strings.Repeat("A", 10000), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":         tt.vendorName,
				"contact_name": "Test Contact",
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/vendors", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateVendor(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test Vendor Notes Field - Max 10000 chars
func TestVendorNotesLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		notes   string
		wantErr bool
	}{
		{"Empty notes", "", false}, // Optional field
		{"Single char", "A", false},
		{"Max valid (10000)", strings.Repeat("A", 10000), false},
		{"Over max (10001)", strings.Repeat("A", 10001), true},
		{"Way over (50k)", strings.Repeat("A", 50000), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":  "Test Vendor",
				"notes": tt.notes,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/vendors", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateVendor(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test Device Notes Field
func TestDeviceNotesLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		notes   string
		wantErr bool
	}{
		{"Empty notes", "", false},
		{"Single char", "N", false},
		{"Max valid (10000)", strings.Repeat("N", 10000), false},
		{"Over max (10001)", strings.Repeat("N", 10001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"serial_number": "TEST-001",
				"ipn":           "TEST-IPN",
				"notes":         tt.notes,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/devices", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateDevice(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test NCR Description Field - Max 1000 chars
func TestNCRDescriptionLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{"Empty description", "", false}, // Optional
		{"Single char", "D", false},
		{"Max valid (1000)", strings.Repeat("D", 1000), false},
		{"Over max (1001)", strings.Repeat("D", 1001), true},
		{"Way over (10k)", strings.Repeat("D", 10000), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title":       "Test NCR",
				"description": tt.description,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/ncrs", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateNCR(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test NCR Title Field - Max 255 chars
func TestNCRTitleLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"Empty title", "", true}, // Required
		{"Single char", "T", false},
		{"Max valid (255)", strings.Repeat("T", 255), false},
		{"Over max (256)", strings.Repeat("T", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title": tt.title,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/ncrs", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateNCR(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test Work Order Notes Field - Max 10000 chars
func TestWorkOrderNotesLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		notes   string
		wantErr bool
	}{
		{"Empty notes", "", false},
		{"Max valid (10000)", strings.Repeat("W", 10000), false},
		{"Over max (10001)", strings.Repeat("W", 10001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"assembly_ipn": "ASM-001",
				"qty":          10,
				"notes":        tt.notes,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/work_orders", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateWorkOrder(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test Field Report Title - Max 255 chars
func TestFieldReportTitleLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"Empty title", "", true}, // Required
		{"Max valid (255)", strings.Repeat("F", 255), false},
		{"Over max (256)", strings.Repeat("F", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title":       tt.title,
				"report_type": "failure",
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/field_reports", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateFieldReport(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test Field Report Description - Max 1000 chars
func TestFieldReportDescriptionLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{"Empty description", "", false},
		{"Max valid (1000)", strings.Repeat("D", 1000), false},
		{"Over max (1001)", strings.Repeat("D", 1001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title":       "Test Report",
				"report_type": "failure",
				"description": tt.description,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/field_reports", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateFieldReport(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test ECO Title - Max 255 chars
func TestECOTitleLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"Empty title", "", true}, // Required
		{"Max valid (255)", strings.Repeat("E", 255), false},
		{"Over max (256)", strings.Repeat("E", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title": tt.title,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/ecos", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateECO(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test ECO Description - Max 1000 chars
func TestECODescriptionLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{"Empty description", "", false},
		{"Max valid (1000)", strings.Repeat("D", 1000), false},
		{"Over max (1001)", strings.Repeat("D", 1001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title":       "Test ECO",
				"description": tt.description,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/ecos", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateECO(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test CAPA Title - Max 255 chars
func TestCAPATitleLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"Empty title", "", true}, // Required
		{"Max valid (255)", strings.Repeat("C", 255), false},
		{"Over max (256)", strings.Repeat("C", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title": tt.title,
				"type":  "corrective",
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/capas", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateCAPA(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test RMA Defect Description - Max 1000 chars
func TestRMADefectDescriptionLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name               string
		defectDescription  string
		wantErr            bool
	}{
		{"Empty description", "", false},
		{"Max valid (1000)", strings.Repeat("R", 1000), false},
		{"Over max (1001)", strings.Repeat("R", 1001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"serial_number":      "SN-001",
				"defect_description": tt.defectDescription,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/rmas", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateRMA(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test User Email Validation (format + length)
func TestUserEmailValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Empty email", "", false}, // Optional
		{"Valid email", "test@example.com", false},
		{"Invalid format", "not-an-email", true},
		{"Very long valid email", strings.Repeat("a", 240) + "@example.com", false},
		{"Exceeds max (255)", strings.Repeat("a", 250) + "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"username": "testuser" + tt.name,
				"email":    tt.email,
				"password": "testpass123",
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateUser(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test IPN Field Length - Max 100 chars
func TestIPNLengthValidation(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name    string
		ipn     string
		wantErr bool
	}{
		{"Empty IPN", "", true}, // Required for device
		{"Single char", "A", false},
		{"Max valid (100)", strings.Repeat("A", 100), false},
		{"Over max (101)", strings.Repeat("A", 101), true},
		{"Way over (1000)", strings.Repeat("A", 1000), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"serial_number": "SN-TEST-" + tt.name,
				"ipn":           tt.ipn,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/devices", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateDevice(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// Test comprehensive validation for all text field types
func TestAllTextFieldLimits(t *testing.T) {
	t.Run("Name fields max 255", func(t *testing.T) {
		// Already covered in TestVendorNameLengthValidation
	})

	t.Run("Description fields max 1000", func(t *testing.T) {
		// Already covered in multiple tests
	})

	t.Run("Notes/Comments fields max 10000", func(t *testing.T) {
		// Already covered in multiple tests
	})

	t.Run("IPN fields max 100", func(t *testing.T) {
		// Already covered in TestIPNLengthValidation
	})
}

// Test that validation errors return proper HTTP 400 with error messages
func TestValidationErrorResponseFormat(t *testing.T) {
	oldDB := db
	db = setupInputValidationDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test with oversized name
	payload := map[string]interface{}{
		"name": strings.Repeat("X", 1000),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}

	// Check response contains error message
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if errorMsg, ok := response["error"].(string); ok {
			if !strings.Contains(errorMsg, "name") && !strings.Contains(errorMsg, "length") {
				t.Errorf("Error message should mention field name and length issue, got: %s", errorMsg)
			}
		}
	}
}
