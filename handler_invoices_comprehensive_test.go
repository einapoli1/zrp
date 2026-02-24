package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ===== Manual Invoice Creation Tests =====

func TestCreateInvoiceManually(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create sales order first (required)
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Test Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoice := Invoice{
		SalesOrderID: soID,
		Customer:     "Test Customer",
		IssueDate:    "2026-02-01",
		DueDate:      "2026-03-01",
		Status:       "draft",
		Notes:        "Manual test invoice",
		Lines: []InvoiceLine{
			{IPN: "PART-001", Description: "Test Part 1", Quantity: 10, UnitPrice: 50.0},
			{IPN: "PART-002", Description: "Test Part 2", Quantity: 5, UnitPrice: 100.0},
		},
	}

	body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreateInvoice(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := extractInvoice(t, rec.Body.Bytes())

	// Verify ID generation
	if !strings.HasPrefix(result.ID, "INV-2026-") {
		t.Errorf("Expected ID to start with INV-2026-, got %s", result.ID)
	}

	// Verify invoice number generation
	if !strings.HasPrefix(result.InvoiceNumber, "INV-2026-") {
		t.Errorf("Expected invoice number to start with INV-2026-, got %s", result.InvoiceNumber)
	}

	// Verify totals calculation
	expectedSubtotal := 1000.0 // (10 * 50) + (5 * 100)
	expectedTax := expectedSubtotal * 0.10
	expectedTotal := expectedSubtotal + expectedTax

	if result.Tax != expectedTax {
		t.Errorf("Expected tax %.2f, got %.2f", expectedTax, result.Tax)
	}

	if result.Total != expectedTotal {
		t.Errorf("Expected total %.2f, got %.2f", expectedTotal, result.Total)
	}

	// Verify lines
	if len(result.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result.Lines))
	}
}

func TestCreateInvoiceWithoutRequiredFields(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name     string
		invoice  Invoice
		expected string
	}{
		{
			name:     "missing sales_order_id",
			invoice:  Invoice{Customer: "Test Customer"},
			expected: "sales_order_id and customer are required",
		},
		{
			name:     "missing customer",
			invoice:  Invoice{SalesOrderID: "SO-001"},
			expected: "sales_order_id and customer are required",
		},
		{
			name:     "both missing",
			invoice:  Invoice{},
			expected: "sales_order_id and customer are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handleCreateInvoice(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rec.Code)
			}

			if !strings.Contains(rec.Body.String(), tt.expected) {
				t.Errorf("Expected error message containing '%s', got '%s'", tt.expected, rec.Body.String())
			}
		})
	}
}

func TestCreateInvoiceWithInvalidJSON(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req, _ := http.NewRequest("POST", "/api/v1/invoices", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreateInvoice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "invalid JSON") {
		t.Error("Expected 'invalid JSON' error message")
	}
}

// ===== Update Invoice Tests =====

func TestUpdateInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create test invoice
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 100.0, 10.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Update invoice
	update := Invoice{
		Customer:  "Updated Customer",
		IssueDate: "2026-03-01",
		DueDate:   "2026-04-01",
		Status:    "draft",
		Notes:     "Updated notes",
		Lines: []InvoiceLine{
			{IPN: "NEW-001", Description: "New Part", Quantity: 3, UnitPrice: 75.0},
		},
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/invoices/%s", invoiceID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateInvoice(rec, req, invoiceID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := extractInvoice(t, rec.Body.Bytes())

	if result.Customer != "Updated Customer" {
		t.Errorf("Expected customer 'Updated Customer', got %s", result.Customer)
	}

	if result.Notes != "Updated notes" {
		t.Errorf("Expected notes 'Updated notes', got %s", result.Notes)
	}

	// Verify totals recalculated
	expectedSubtotal := 225.0 // 3 * 75
	expectedTax := expectedSubtotal * 0.10
	expectedTotal := expectedSubtotal + expectedTax

	if result.Tax != expectedTax {
		t.Errorf("Expected tax %.2f, got %.2f", expectedTax, result.Tax)
	}

	if result.Total != expectedTotal {
		t.Errorf("Expected total %.2f, got %.2f", expectedTotal, result.Total)
	}
}

func TestUpdatePaidInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create paid invoice
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, paid_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "paid", 100.0, 10.0, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Attempt to update paid invoice
	update := Invoice{Customer: "New Customer"}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/invoices/%s", invoiceID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateInvoice(rec, req, invoiceID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "cannot edit paid or cancelled invoices") {
		t.Error("Expected error about editing paid invoices")
	}
}

func TestUpdateCancelledInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create cancelled invoice
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "cancelled", 100.0, 10.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Attempt to update cancelled invoice
	update := Invoice{Customer: "New Customer"}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/invoices/%s", invoiceID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateInvoice(rec, req, invoiceID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "cannot edit paid or cancelled invoices") {
		t.Error("Expected error about editing cancelled invoices")
	}
}

func TestUpdateNonExistentInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	update := Invoice{Customer: "New Customer"}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/v1/invoices/INV-2026-999999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleUpdateInvoice(rec, req, "INV-2026-999999")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "invoice not found") {
		t.Error("Expected 'invoice not found' error message")
	}
}

// ===== Edge Cases: Line Items =====

func TestCreateInvoiceWithZeroQuantity(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoice := Invoice{
		SalesOrderID: soID,
		Customer:     "Test Customer",
		Lines: []InvoiceLine{
			{IPN: "PART-001", Description: "Test Part", Quantity: 0, UnitPrice: 50.0},
		},
	}

	body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreateInvoice(rec, req)

	// Database has CHECK constraint: quantity > 0, so this should fail
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 (constraint violation), got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "constraint") {
		t.Errorf("Expected constraint error, got: %s", rec.Body.String())
	}
}

func TestCreateInvoiceWithEmptyLines(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoice := Invoice{
		SalesOrderID: soID,
		Customer:     "Test Customer",
		Lines:        []InvoiceLine{}, // Empty lines
	}

	body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreateInvoice(rec, req)

	// Should succeed with empty lines
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := extractInvoice(t, rec.Body.Bytes())

	if result.Total != 0.0 {
		t.Errorf("Expected total 0.0 for empty lines, got %.2f", result.Total)
	}

	if len(result.Lines) != 0 {
		t.Errorf("Expected 0 lines, got %d", len(result.Lines))
	}
}

func TestCreateInvoiceWithManyLines(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create invoice with 50 lines
	var lines []InvoiceLine
	for i := 1; i <= 50; i++ {
		lines = append(lines, InvoiceLine{
			IPN:         fmt.Sprintf("PART-%03d", i),
			Description: fmt.Sprintf("Part %d", i),
			Quantity:    i,
			UnitPrice:   10.0,
		})
	}

	invoice := Invoice{
		SalesOrderID: soID,
		Customer:     "Test Customer",
		Lines:        lines,
	}

	body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreateInvoice(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := extractInvoice(t, rec.Body.Bytes())

	if len(result.Lines) != 50 {
		t.Errorf("Expected 50 lines, got %d", len(result.Lines))
	}

	// Verify total calculation: sum(1+2+3+...+50) * 10 = 1275 * 10 = 12750
	expectedSubtotal := 12750.0
	expectedTax := expectedSubtotal * 0.10
	expectedTotal := expectedSubtotal + expectedTax

	if result.Total != expectedTotal {
		t.Errorf("Expected total %.2f, got %.2f", expectedTotal, result.Total)
	}
}

// ===== Status Workflow Tests =====

func TestInvoiceStatusWorkflow(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create draft invoice
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 100.0, 10.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Test workflow: draft -> sent -> paid
	t.Run("draft_to_sent", func(t *testing.T) {
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/invoices/%s/send", invoiceID), nil)
		rec := httptest.NewRecorder()

		handleSendInvoice(rec, req, invoiceID)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var status string
		db.QueryRow("SELECT status FROM invoices WHERE id = ?", invoiceID).Scan(&status)
		if status != "sent" {
			t.Errorf("Expected status 'sent', got %s", status)
		}
	})

	t.Run("sent_to_paid", func(t *testing.T) {
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/invoices/%s/mark-paid", invoiceID), nil)
		rec := httptest.NewRecorder()

		handleMarkInvoicePaid(rec, req, invoiceID)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var status string
		db.QueryRow("SELECT status FROM invoices WHERE id = ?", invoiceID).Scan(&status)
		if status != "paid" {
			t.Errorf("Expected status 'paid', got %s", status)
		}
	})
}

func TestSendNonDraftInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create sent invoice
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "sent", 100.0, 10.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/invoices/%s/send", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleSendInvoice(rec, req, invoiceID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "only draft invoices can be sent") {
		t.Error("Expected error about only draft invoices being sendable")
	}
}

func TestMarkCancelledInvoicePaid(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create cancelled invoice
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "cancelled", 100.0, 10.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/invoices/%s/mark-paid", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleMarkInvoicePaid(rec, req, invoiceID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "cannot mark cancelled invoice as paid") {
		t.Error("Expected error about cancelled invoices")
	}
}

// ===== Tax and Calculation Tests =====

func TestTaxCalculationAccuracy(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	tests := []struct {
		name          string
		lines         []InvoiceLine
		expectedSub   float64
		expectedTax   float64
		expectedTotal float64
	}{
		{
			name: "simple_calculation",
			lines: []InvoiceLine{
				{IPN: "P1", Description: "Product 1", Quantity: 2, UnitPrice: 100.0},
			},
			expectedSub:   200.0,
			expectedTax:   20.0,
			expectedTotal: 220.0,
		},
		{
			name: "decimal_quantities",
			lines: []InvoiceLine{
				{IPN: "P1", Description: "Product 1", Quantity: 3, UnitPrice: 33.33},
			},
			expectedSub:   99.99,
			expectedTax:   9.999,
			expectedTotal: 109.989,
		},
		{
			name: "large_amounts",
			lines: []InvoiceLine{
				{IPN: "P1", Description: "Product 1", Quantity: 100, UnitPrice: 999.99},
			},
			expectedSub:   99999.0,
			expectedTax:   9999.9,
			expectedTotal: 109998.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoice := Invoice{
				SalesOrderID: soID,
				Customer:     "Test Customer",
				Lines:        tt.lines,
			}

			body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handleCreateInvoice(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
			}

			result := extractInvoice(t, rec.Body.Bytes())

			// Allow small floating point errors
			if absFloat(result.Tax-tt.expectedTax) > 0.01 {
				t.Errorf("Expected tax %.4f, got %.4f", tt.expectedTax, result.Tax)
			}

			if absFloat(result.Total-tt.expectedTotal) > 0.01 {
				t.Errorf("Expected total %.4f, got %.4f", tt.expectedTotal, result.Total)
			}
		})
	}
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ===== Filter Tests =====

func TestListInvoicesWithFilters(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	now := time.Now()

	// Create multiple invoices with different customers and dates
	createTestInvoice := func(customer, status, issueDate string) string {
		soID := nextID("SO", "sales_orders", 3)
		_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			soID, customer, "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("Failed to create sales order: %v", err)
		}

		invoiceID := nextID("INV", "invoices", 6)
		_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			invoiceID, generateInvoiceNumber(), soID, customer, issueDate, now.AddDate(0, 0, 30).Format("2006-01-02"), status, 100.0, 10.0, now.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("Failed to create invoice: %v", err)
		}
		return invoiceID
	}

	createTestInvoice("Acme Corp", "draft", "2026-02-01")
	createTestInvoice("Acme Corp", "sent", "2026-02-15")
	createTestInvoice("Beta Inc", "paid", "2026-02-20")
	createTestInvoice("Charlie LLC", "draft", "2026-03-01")

	tests := []struct {
		name          string
		queryParams   string
		expectedCount int
	}{
		{"filter_by_status_draft", "?status=draft", 2},
		{"filter_by_status_sent", "?status=sent", 1},
		{"filter_by_status_paid", "?status=paid", 1},
		{"filter_by_customer", "?customer=Acme", 2},
		{"filter_by_customer_partial", "?customer=Beta", 1},
		{"filter_by_from_date", "?from_date=2026-02-15", 3},
		{"filter_by_to_date", "?to_date=2026-02-15", 2},
		{"filter_by_date_range", "?from_date=2026-02-01&to_date=2026-02-20", 3},
		{"combined_filters", "?customer=Acme&status=sent", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/invoices"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			handleListInvoices(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rec.Code)
			}

			invoices := extractInvoices(t, rec.Body.Bytes())

			if len(invoices) != tt.expectedCount {
				t.Errorf("Expected %d invoices, got %d", tt.expectedCount, len(invoices))
			}
		})
	}
}

// ===== ID Generation Tests =====

func TestInvoiceIDGeneration(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create multiple invoices and verify IDs are sequential
	var ids []string
	for i := 0; i < 3; i++ {
		invoice := Invoice{
			SalesOrderID: soID,
			Customer:     "Test Customer",
			Lines: []InvoiceLine{
				{IPN: "PART-001", Description: "Test Part", Quantity: 1, UnitPrice: 10.0},
			},
		}

		body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handleCreateInvoice(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		result := extractInvoice(t, rec.Body.Bytes())
		ids = append(ids, result.ID)
	}

	// Verify all IDs follow pattern INV-YYYY-NNNNNN
	year := time.Now().Year()
	expectedPrefix := fmt.Sprintf("INV-%d-", year)

	for i, id := range ids {
		if !strings.HasPrefix(id, expectedPrefix) {
			t.Errorf("Invoice %d: Expected ID to start with %s, got %s", i, expectedPrefix, id)
		}

		// Verify 6-digit suffix
		parts := strings.Split(id, "-")
		if len(parts) != 3 {
			t.Errorf("Invoice %d: Expected ID format INV-YYYY-NNNNNN, got %s", i, id)
			continue
		}

		if len(parts[2]) != 6 {
			t.Errorf("Invoice %d: Expected 6-digit sequence, got %s", i, parts[2])
		}
	}

	// Verify IDs are unique
	uniqueMap := make(map[string]bool)
	for _, id := range ids {
		if uniqueMap[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		uniqueMap[id] = true
	}
}

func TestInvoiceNumberGeneration(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create sales order first
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Generate and save multiple invoice numbers to DB
	var numbers []string
	for i := 0; i < 3; i++ {
		num := generateInvoiceNumber()
		numbers = append(numbers, num)
		
		// Insert the invoice into DB so next call sees it
		invoiceID := nextID("INV", "invoices", 6)
		_, err := db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			invoiceID, num, soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 100.0, 10.0, now.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("Failed to create invoice: %v", err)
		}
	}

	year := time.Now().Year()
	expectedPrefix := fmt.Sprintf("INV-%d-", year)

	for i, num := range numbers {
		if !strings.HasPrefix(num, expectedPrefix) {
			t.Errorf("Invoice number %d: Expected to start with %s, got %s", i, expectedPrefix, num)
		}

		// Verify 5-digit suffix
		parts := strings.Split(num, "-")
		if len(parts) != 3 {
			t.Errorf("Invoice number %d: Expected format INV-YYYY-NNNNN, got %s", i, num)
			continue
		}

		if len(parts[2]) != 5 {
			t.Errorf("Invoice number %d: Expected 5-digit sequence, got %s", i, parts[2])
		}
	}

	// Verify uniqueness
	if numbers[0] == numbers[1] || numbers[1] == numbers[2] || numbers[0] == numbers[2] {
		t.Error("Invoice numbers should be unique")
	}
}

// ===== Integration: Sales Order to Invoice =====

func TestSalesOrderStatusAfterInvoiceCreation(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	salesOrderID, _ := setupInvoiceTestData(t)

	// Create invoice from sales order
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/sales-orders/%s/create-invoice", salesOrderID), nil)
	rec := httptest.NewRecorder()

	handleCreateInvoiceFromSalesOrder(rec, req, salesOrderID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify sales order status updated to 'invoiced'
	var status string
	err := db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", salesOrderID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query sales order status: %v", err)
	}

	if status != "invoiced" {
		t.Errorf("Expected sales order status 'invoiced', got %s", status)
	}
}

// ===== Get Invoice Tests =====

func TestGetNonExistentInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req, _ := http.NewRequest("GET", "/api/v1/invoices/INV-2026-999999", nil)
	rec := httptest.NewRecorder()

	handleGetInvoice(rec, req, "INV-2026-999999")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "invoice not found") {
		t.Error("Expected 'invoice not found' error message")
	}
}

// ===== Default Values Tests =====

func TestCreateInvoiceWithDefaults(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "confirmed", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create invoice without dates or status
	invoice := Invoice{
		SalesOrderID: soID,
		Customer:     "Test Customer",
		Lines: []InvoiceLine{
			{IPN: "P1", Description: "Product", Quantity: 1, UnitPrice: 100.0},
		},
	}

	body, _ := json.Marshal(invoice)
	req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handleCreateInvoice(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := extractInvoice(t, rec.Body.Bytes())

	// Verify defaults
	if result.Status != "draft" {
		t.Errorf("Expected default status 'draft', got %s", result.Status)
	}

	if result.IssueDate != now.Format("2006-01-02") {
		t.Errorf("Expected issue_date to be today (%s), got %s", now.Format("2006-01-02"), result.IssueDate)
	}

	expectedDueDate := now.AddDate(0, 0, 30).Format("2006-01-02")
	if result.DueDate != expectedDueDate {
		t.Errorf("Expected due_date to be 30 days from now (%s), got %s", expectedDueDate, result.DueDate)
	}
}

// ===== PDF Generation Edge Cases =====

func TestGeneratePDFForNonExistentInvoice(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req, _ := http.NewRequest("GET", "/api/v1/invoices/INV-2026-999999/pdf", nil)
	rec := httptest.NewRecorder()

	handleGenerateInvoicePDF(rec, req, "INV-2026-999999")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "invoice not found") {
		t.Error("Expected 'invoice not found' error message")
	}
}

func TestGeneratePDFWithoutLines(t *testing.T) {
	oldDB := db
	db = setupTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create invoice without lines
	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 0.0, 0.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/invoices/%s/pdf", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleGenerateInvoicePDF(rec, req, invoiceID)

	// Should still generate PDF successfully
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.Bytes()
	if len(body) < 4 || string(body[:4]) != "%PDF" {
		t.Error("Response should start with PDF header")
	}
}
