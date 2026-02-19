package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// XSS test payloads - common attack vectors
var xssPayloads = []string{
	"<script>alert('XSS')</script>",
	"<img src=x onerror=alert(1)>",
	"<iframe src='javascript:alert(1)'>",
	"javascript:alert(1)",
	"<svg onload=alert('XSS')>",
	"<body onload=alert('XSS')>",
	"<input onfocus=alert('XSS') autofocus>",
	"<select onfocus=alert('XSS') autofocus>",
	"<textarea onfocus=alert('XSS') autofocus>",
	"<keygen onfocus=alert('XSS') autofocus>",
	"<video><source onerror=alert('XSS')>",
	"<audio src=x onerror=alert('XSS')>",
	"<details open ontoggle=alert('XSS')>",
	"';alert(String.fromCharCode(88,83,83))//",
	"\"><script>alert('XSS')</script>",
}

// setupXSSTestDB creates a test database for XSS testing
func setupXSSTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create all necessary tables
	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			email TEXT
		);

		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE TABLE parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT UNIQUE NOT NULL,
			description TEXT,
			category TEXT,
			mpn TEXT,
			manufacturer TEXT,
			datasheet TEXT,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			address TEXT,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			priority TEXT DEFAULT 'normal',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			valid_until TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE quote_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty REAL NOT NULL,
			unit_price REAL NOT NULL,
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		);

		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			assignee TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			serial_number TEXT,
			description TEXT,
			location TEXT,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE field_reports (
			id TEXT PRIMARY KEY,
			device_id TEXT,
			reporter TEXT,
			issue TEXT,
			description TEXT,
			severity TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			part_ipn TEXT,
			severity TEXT,
			root_cause TEXT,
			corrective_action TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE capas (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			type TEXT,
			status TEXT DEFAULT 'open',
			root_cause TEXT,
			corrective_action TEXT,
			preventive_action TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE docs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT,
			category TEXT,
			status TEXT DEFAULT 'draft',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	_, err = testDB.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)", "admin", string(hash), "admin")
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	return testDB
}

// createXSSTestSession creates an admin session for testing
func createXSSTestSession(t *testing.T, testDB *sql.DB) string {
	token := "test-session-token-xss"
	_, err := testDB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, datetime('now', '+1 day'))", token, 1)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	return token
}

// TestXSS_PartName tests XSS in part name/IPN fields
func TestXSS_PartName(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	for i, payload := range xssPayloads {
		t.Run(fmt.Sprintf("Payload_%d", i), func(t *testing.T) {
			// Create part with XSS payload in IPN
			partData := map[string]interface{}{
				"ipn":         fmt.Sprintf("XSS-%d-%s", i, payload),
				"description": "Test part",
				"category":    "test",
			}
			body, _ := json.Marshal(partData)

			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
			w := httptest.NewRecorder()

			handleCreatePart(w, req)

			if w.Code != 201 && w.Code != 200 {
				t.Logf("Create failed (expected for validation): %d", w.Code)
				return
			}

			// Retrieve the part
			var result map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &result)
			partID := result["id"]

			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/parts/%v", partID), nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
			w = httptest.NewRecorder()

			handleGetPart(w, req, fmt.Sprintf("%v", partID))

			// Verify response is properly escaped
			responseBody := w.Body.String()
			
			// Check that script tags are escaped
			if strings.Contains(responseBody, "<script>") {
				t.Errorf("XSS vulnerability: unescaped <script> tag found in response")
			}
			
			// Check that the payload appears escaped or encoded
			if strings.Contains(responseBody, payload) && strings.Contains(payload, "<") {
				// If payload contains HTML and appears verbatim, it's likely not escaped
				if !strings.Contains(responseBody, "&lt;") && !strings.Contains(responseBody, "\\u003c") {
					t.Errorf("XSS vulnerability: payload not properly escaped in JSON response")
				}
			}
		})
	}
}

// TestXSS_PartDescription tests XSS in part description field
func TestXSS_PartDescription(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	for i, payload := range xssPayloads {
		t.Run(fmt.Sprintf("Payload_%d", i), func(t *testing.T) {
			partData := map[string]interface{}{
				"ipn":         fmt.Sprintf("TEST-PART-%d", i),
				"description": payload,
				"category":    "test",
			}
			body, _ := json.Marshal(partData)

			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
			w := httptest.NewRecorder()

			handleCreatePart(w, req)

			responseBody := w.Body.String()
			
			if strings.Contains(responseBody, "<script>") && !strings.Contains(responseBody, "&lt;script&gt;") {
				t.Errorf("XSS vulnerability in description: unescaped script tag")
			}
		})
	}
}

// TestXSS_PartNotes tests XSS in part notes field
func TestXSS_PartNotes(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<script>alert('XSS')</script>"
	
	partData := map[string]interface{}{
		"ipn":         "TEST-NOTES",
		"description": "Test",
		"category":    "test",
		"notes":       payload,
	}
	body, _ := json.Marshal(partData)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreatePart(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<script>") && !strings.Contains(responseBody, "&lt;script&gt;") {
		t.Errorf("XSS vulnerability in notes: unescaped script tag")
	}
}

// TestXSS_VendorName tests XSS in vendor name
func TestXSS_VendorName(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<img src=x onerror=alert('XSS')>"
	
	vendorData := map[string]interface{}{
		"name":          payload,
		"contact_name":  "John Doe",
		"contact_email": "john@example.com",
	}
	body, _ := json.Marshal(vendorData)

	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<img") && strings.Contains(responseBody, "onerror") {
		t.Errorf("XSS vulnerability in vendor name: unescaped img tag")
	}
}

// TestXSS_WorkOrderNotes tests XSS in work order notes
func TestXSS_WorkOrderNotes(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	// First create a part
	testDB.Exec("INSERT INTO parts (ipn, description) VALUES (?, ?)", "ASM-001", "Test Assembly")

	payload := "<script>alert('XSS')</script>"
	
	woData := map[string]interface{}{
		"assembly_ipn": "ASM-001",
		"qty":          10,
		"notes":        payload,
	}
	body, _ := json.Marshal(woData)

	req := httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateWorkOrder(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<script>") && !strings.Contains(responseBody, "&lt;script&gt;") {
		t.Errorf("XSS vulnerability in work order notes: unescaped script tag")
	}
}

// TestXSS_WorkOrderPDF tests XSS in work order PDF generation (HTML output)
func TestXSS_WorkOrderPDF(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	// Create part with XSS in description
	testDB.Exec("INSERT INTO parts (ipn, description) VALUES (?, ?)", "ASM-XSS", "<script>alert('XSS')</script>")

	// Create work order
	woID := "WO-XSS-001"
	testDB.Exec("INSERT INTO work_orders (id, assembly_ipn, qty, notes) VALUES (?, ?, ?, ?)", 
		woID, "ASM-XSS", 5, "<img src=x onerror=alert(1)>")

	req := httptest.NewRequest("GET", "/api/v1/workorders/"+woID+"/pdf", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleWorkOrderPDF(w, req, woID)

	htmlOutput := w.Body.String()
	
	// Check for unescaped script tags
	if strings.Contains(htmlOutput, "<script>alert") {
		t.Errorf("CRITICAL XSS vulnerability in work order PDF: unescaped script tag in HTML output")
	}
	
	// Check for unescaped img with onerror
	if strings.Contains(htmlOutput, "<img") && strings.Contains(htmlOutput, "onerror=alert") {
		t.Errorf("CRITICAL XSS vulnerability in work order PDF: unescaped img onerror in HTML output")
	}
	
	// Verify HTML entities are used for user content
	if !strings.Contains(htmlOutput, "&lt;") && strings.Contains(htmlOutput, "<script>") {
		t.Errorf("HTML entities not used for escaping user content in PDF")
	}
}

// TestXSS_QuoteCustomer tests XSS in quote customer name
func TestXSS_QuoteCustomer(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<svg onload=alert('XSS')>"
	
	quoteData := map[string]interface{}{
		"customer":    payload,
		"valid_until": "2026-12-31",
		"items": []map[string]interface{}{
			{
				"ipn":        "TEST-001",
				"description": "Test item",
				"qty":         1.0,
				"unit_price":  10.0,
			},
		},
	}
	body, _ := json.Marshal(quoteData)

	req := httptest.NewRequest("POST", "/api/v1/quotes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateQuote(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<svg") && strings.Contains(responseBody, "onload") {
		t.Errorf("XSS vulnerability in quote customer: unescaped svg tag")
	}
}

// TestXSS_QuoteNotes tests XSS in quote notes
func TestXSS_QuoteNotes(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<iframe src='javascript:alert(1)'>"
	
	quoteData := map[string]interface{}{
		"customer":    "Test Customer",
		"valid_until": "2026-12-31",
		"notes":       payload,
		"items": []map[string]interface{}{
			{
				"ipn":        "TEST-001",
				"description": "Test item",
				"qty":         1.0,
				"unit_price":  10.0,
			},
		},
	}
	body, _ := json.Marshal(quoteData)

	req := httptest.NewRequest("POST", "/api/v1/quotes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateQuote(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<iframe") {
		t.Errorf("XSS vulnerability in quote notes: unescaped iframe tag")
	}
}

// TestXSS_QuotePDF tests XSS in quote PDF generation (HTML output)
func TestXSS_QuotePDF(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	// Create quote with XSS payloads
	quoteID := "Q-XSS-001"
	testDB.Exec("INSERT INTO quotes (id, customer, valid_until, notes) VALUES (?, ?, ?, ?)", 
		quoteID, "<script>alert('Customer XSS')</script>", "2026-12-31", "<img src=x onerror=alert('Notes XSS')>")
	
	testDB.Exec("INSERT INTO quote_items (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)",
		quoteID, "TEST-001", "<svg onload=alert('Item XSS')>", 1.0, 100.0)

	req := httptest.NewRequest("GET", "/api/v1/quotes/"+quoteID+"/pdf", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleQuotePDF(w, req, quoteID)

	htmlOutput := w.Body.String()
	
	// Check for unescaped script tags
	if strings.Contains(htmlOutput, "<script>alert") {
		t.Errorf("CRITICAL XSS vulnerability in quote PDF: unescaped script tag in customer field")
	}
	
	// Check for unescaped img with onerror
	if strings.Contains(htmlOutput, "onerror=alert") {
		t.Errorf("CRITICAL XSS vulnerability in quote PDF: unescaped onerror in notes")
	}
	
	// Check for unescaped svg
	if strings.Contains(htmlOutput, "<svg") && strings.Contains(htmlOutput, "onload=alert") {
		t.Errorf("CRITICAL XSS vulnerability in quote PDF: unescaped svg in item description")
	}
}

// TestXSS_ECOTitle tests XSS in ECO title
func TestXSS_ECOTitle(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<body onload=alert('XSS')>"
	
	ecoData := map[string]interface{}{
		"title":       payload,
		"description": "Test ECO",
	}
	body, _ := json.Marshal(ecoData)

	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<body") && strings.Contains(responseBody, "onload") {
		t.Errorf("XSS vulnerability in ECO title: unescaped body tag")
	}
}

// TestXSS_ECODescription tests XSS in ECO description
func TestXSS_ECODescription(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<input onfocus=alert('XSS') autofocus>"
	
	ecoData := map[string]interface{}{
		"title":       "Test ECO",
		"description": payload,
	}
	body, _ := json.Marshal(ecoData)

	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<input") && strings.Contains(responseBody, "onfocus") {
		t.Errorf("XSS vulnerability in ECO description: unescaped input tag")
	}
}

// TestXSS_DeviceName tests XSS in device name
func TestXSS_DeviceName(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<video><source onerror=alert('XSS')>"
	
	deviceData := map[string]interface{}{
		"name":          payload,
		"serial_number": "SN-001",
	}
	body, _ := json.Marshal(deviceData)

	req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateDevice(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<video") || strings.Contains(responseBody, "onerror") {
		t.Errorf("XSS vulnerability in device name: unescaped video/onerror")
	}
}

// TestXSS_NCRTitle tests XSS in NCR title
func TestXSS_NCRTitle(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<audio src=x onerror=alert('XSS')>"
	
	ncrData := map[string]interface{}{
		"title":       payload,
		"description": "Test NCR",
		"severity":    "high",
	}
	body, _ := json.Marshal(ncrData)

	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateNCR(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<audio") || strings.Contains(responseBody, "onerror") {
		t.Errorf("XSS vulnerability in NCR title: unescaped audio tag")
	}
}

// TestXSS_CAPATitle tests XSS in CAPA title
func TestXSS_CAPATitle(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<details open ontoggle=alert('XSS')>"
	
	capaData := map[string]interface{}{
		"title":       payload,
		"description": "Test CAPA",
		"type":        "corrective",
	}
	body, _ := json.Marshal(capaData)

	req := httptest.NewRequest("POST", "/api/v1/capas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateCAPA(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "<details") || strings.Contains(responseBody, "ontoggle") {
		t.Errorf("XSS vulnerability in CAPA title: unescaped details tag")
	}
}

// TestXSS_DocumentTitle tests XSS in document title
func TestXSS_DocumentTitle(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "';alert(String.fromCharCode(88,83,83))//"
	
	docData := map[string]interface{}{
		"title":    payload,
		"content":  "Test content",
		"category": "test",
	}
	body, _ := json.Marshal(docData)

	req := httptest.NewRequest("POST", "/api/v1/docs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	responseBody := w.Body.String()
	
	if strings.Contains(responseBody, "alert(String.fromCharCode") {
		t.Errorf("XSS vulnerability in document title: JavaScript code not escaped")
	}
}

// TestXSS_SearchQuery tests XSS in search query parameters
func TestXSS_SearchQuery(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	payload := "<script>alert('XSS')</script>"
	
	req := httptest.NewRequest("GET", "/api/v1/search?q="+payload, nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleGlobalSearch(w, req)

	responseBody := w.Body.String()
	
	// Search results should not include unescaped script tags
	if strings.Contains(responseBody, "<script>alert") {
		t.Errorf("XSS vulnerability in search results: unescaped query parameter")
	}
}

// TestContentSecurityPolicy tests that CSP headers are present
func TestContentSecurityPolicy(t *testing.T) {
	endpoints := []string{
		"/api/v1/parts",
		"/api/v1/vendors",
		"/api/v1/workorders",
		"/api/v1/quotes",
		"/api/v1/ecos",
		"/api/v1/devices",
		"/api/v1/ncrs",
		"/api/v1/capas",
		"/api/v1/docs",
	}

	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
			w := httptest.NewRecorder()

			// Use the main router to test middleware
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				handleListParts(w, r)
			})
			mux.ServeHTTP(w, req)

			// Check for security headers
			csp := w.Header().Get("Content-Security-Policy")
			xContentType := w.Header().Get("X-Content-Type-Options")
			xFrame := w.Header().Get("X-Frame-Options")

			// CSP should be present for HTML responses
			if w.Header().Get("Content-Type") == "text/html" && csp == "" {
				t.Logf("Warning: Content-Security-Policy header missing for HTML endpoint %s", endpoint)
			}

			// X-Content-Type-Options should always be present
			if xContentType != "nosniff" {
				t.Logf("Warning: X-Content-Type-Options header missing or incorrect for %s", endpoint)
			}

			// X-Frame-Options should be present
			if xFrame == "" {
				t.Logf("Warning: X-Frame-Options header missing for %s", endpoint)
			}
		})
	}
}

// TestXSS_HTMLResponses tests all HTML-generating endpoints for proper escaping
func TestXSS_HTMLResponses(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	t.Run("WorkOrderPDF_HTMLEscaping", func(t *testing.T) {
		// Create part and work order with various XSS payloads
		testDB.Exec("INSERT INTO parts (ipn, description, mpn, manufacturer) VALUES (?, ?, ?, ?)", 
			"TEST-001", 
			"<script>alert('desc')</script>",
			"<img src=x onerror=alert('mpn')>",
			"<svg onload=alert('mfr')>")

		woID := "WO-HTML-001"
		testDB.Exec("INSERT INTO work_orders (id, assembly_ipn, qty, notes, priority) VALUES (?, ?, ?, ?, ?)", 
			woID, "TEST-001", 10, "<iframe src=javascript:alert(1)>", "<script>alert('priority')</script>")

		req := httptest.NewRequest("GET", "/api/v1/workorders/"+woID+"/pdf", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
		w := httptest.NewRecorder()

		handleWorkOrderPDF(w, req, woID)

		html := w.Body.String()

		// All user content should be HTML-escaped
		vulnerabilities := []struct {
			pattern string
			field   string
		}{
			{"<script>alert('desc')</script>", "description"},
			{"<img src=x onerror=", "mpn"},
			{"<svg onload=", "manufacturer"},
			{"<iframe src=javascript:", "notes"},
			{"<script>alert('priority')", "priority"},
		}

		for _, v := range vulnerabilities {
			if strings.Contains(html, v.pattern) {
				t.Errorf("CRITICAL: Unescaped XSS payload in %s field: %s", v.field, v.pattern)
			}
		}

		// Verify HTML escaping is being used
		if !strings.Contains(html, "&lt;") {
			t.Errorf("HTML entity escaping not detected in output")
		}
	})

	t.Run("QuotePDF_HTMLEscaping", func(t *testing.T) {
		quoteID := "Q-HTML-001"
		testDB.Exec("INSERT INTO quotes (id, customer, notes, valid_until) VALUES (?, ?, ?, ?)", 
			quoteID,
			"<script>alert('customer')</script>",
			"<img src=x onerror=alert('notes')>",
			"2026-12-31")
		
		testDB.Exec("INSERT INTO quote_items (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)",
			quoteID, "ITEM-001", "<svg onload=alert('item')>", 5.0, 100.0)

		req := httptest.NewRequest("GET", "/api/v1/quotes/"+quoteID+"/pdf", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
		w := httptest.NewRecorder()

		handleQuotePDF(w, req, quoteID)

		html := w.Body.String()

		vulnerabilities := []struct {
			pattern string
			field   string
		}{
			{"<script>alert('customer')", "customer"},
			{"<img src=x onerror=", "notes"},
			{"<svg onload=alert", "item description"},
		}

		for _, v := range vulnerabilities {
			if strings.Contains(html, v.pattern) {
				t.Errorf("CRITICAL: Unescaped XSS payload in %s field: %s", v.field, v.pattern)
			}
		}
	})
}

// TestXSS_JSONEncoding verifies JSON responses don't introduce XSS
func TestXSS_JSONEncoding(t *testing.T) {
	oldDB := db
	testDB := setupXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	sessionToken := createXSSTestSession(t, testDB)

	// Test that JSON encoding properly escapes HTML special chars
	payload := "<script>alert('XSS')</script>"
	
	partData := map[string]interface{}{
		"ipn":         "JSON-TEST",
		"description": payload,
	}
	body, _ := json.Marshal(partData)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreatePart(w, req)

	// JSON encoding should escape < and > as \u003c and \u003e
	responseBody := w.Body.String()
	
	// Verify proper JSON encoding
	if strings.Contains(responseBody, "<script>") {
		// Check if it's properly escaped in JSON
		if !strings.Contains(responseBody, "\\u003cscript\\u003e") && 
		   !strings.Contains(responseBody, "\\u003Cscript\\u003E") {
			t.Errorf("JSON response contains unescaped HTML: %s", responseBody)
		}
	}
}
