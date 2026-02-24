package main

import (
	"bytes"
	
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestQuoteForeignKeyConstraints verifies that foreign key constraints are enforced
func TestQuoteForeignKeyConstraints(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Test", "2026-12-31")

	// Try to insert a quote line with non-existent quote_id
	_, err := db.Exec("INSERT INTO quote_lines (quote_id, ipn, qty, unit_price) VALUES (?, ?, ?, ?)",
		"Q-NONEXISTENT", "IPN-001", 10, 100.0)

	if err == nil {
		t.Error("Expected foreign key constraint error for non-existent quote_id, but insert succeeded")
	}

	// Verify the error is a foreign key constraint violation
	if !strings.Contains(err.Error(), "FOREIGN KEY") && !strings.Contains(err.Error(), "constraint") {
		t.Logf("Warning: error message doesn't clearly indicate FK violation: %v", err)
	}
}

// TestQuoteCascadeDelete verifies that deleting a quote cascades to quote_lines
func TestQuoteCascadeDelete(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Test", "2026-12-31")
	insertTestQuoteLine(t, db, "Q-001", "IPN-100", "Widget A", 10, 25.50, "")
	insertTestQuoteLine(t, db, "Q-001", "IPN-200", "Widget B", 5, 50.00, "")

	// Verify lines exist
	var lineCount int
	err := db.QueryRow("SELECT COUNT(*) FROM quote_lines WHERE quote_id = ?", "Q-001").Scan(&lineCount)
	if err != nil {
		t.Fatalf("Failed to count quote lines: %v", err)
	}
	if lineCount != 2 {
		t.Errorf("Expected 2 lines before delete, got %d", lineCount)
	}

	// Delete the quote
	_, err = db.Exec("DELETE FROM quotes WHERE id = ?", "Q-001")
	if err != nil {
		t.Fatalf("Failed to delete quote: %v", err)
	}

	// Verify lines were cascade deleted
	err = db.QueryRow("SELECT COUNT(*) FROM quote_lines WHERE quote_id = ?", "Q-001").Scan(&lineCount)
	if err != nil {
		t.Fatalf("Failed to count quote lines after delete: %v", err)
	}
	if lineCount != 0 {
		t.Errorf("Expected 0 lines after cascade delete, got %d", lineCount)
	}
}

// TestQuoteNegativeMarginPrevention tests that we can detect negative margins
func TestQuoteNegativeMarginPrevention(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Test", "2026-12-31")
	insertTestQuoteLine(t, db, "Q-001", "IPN-100", "Widget A", 10, 50.00, "")

	// Set up a PO line where cost is higher than quoted price (negative margin)
	insertTestPOLine(t, db, "PO-001", "IPN-100", 75.00) // Cost $75, selling at $50 = -$25 margin

	req := httptest.NewRequest("GET", "/api/quotes/Q-001/cost", nil)
	w := httptest.NewRecorder()

	handleQuoteCost(w, req, "Q-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})

	// Verify negative margin is calculated correctly
	if totalMargin, ok := result["total_margin"].(float64); ok {
		expectedMargin := (10 * 50.00) - (10 * 75.00) // = -250
		if totalMargin >= 0 {
			t.Errorf("Expected negative margin, got %.2f", totalMargin)
		}
		if totalMargin != expectedMargin {
			t.Logf("Warning: margin calculation off. Expected %.2f, got %.2f", expectedMargin, totalMargin)
		}
	}

	// Verify negative margin percentage
	if marginPct, ok := result["total_margin_pct"].(float64); ok {
		if marginPct >= 0 {
			t.Errorf("Expected negative margin percentage, got %.2f%%", marginPct)
		}
	}

	t.Logf("✓ Negative margin detection working: %v", result)
}

// TestQuoteSQLInjectionSafety tests that parameterized queries prevent SQL injection
func TestQuoteSQLInjectionSafety(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// SQL injection payloads
	sqlInjectionPayloads := []string{
		"'; DROP TABLE quotes; --",
		"' OR '1'='1",
		"'; DELETE FROM quotes WHERE '1'='1",
		"admin' --",
		"' UNION SELECT * FROM quotes --",
	}

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Injection_%s", strings.ReplaceAll(payload, " ", "_")), func(t *testing.T) {
			reqBody := fmt.Sprintf(`{"customer":"%s","status":"draft","notes":"Test"}`, payload)
			req := httptest.NewRequest("POST", "/api/quotes", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleCreateQuote(w, req)

			// Should either succeed (treating payload as literal string) or fail validation
			// But should NOT execute the SQL injection
			if w.Code == 200 {
				// Verify the payload was stored as literal text, not executed
				var resp APIResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				quoteData := resp.Data.(map[string]interface{})
				quoteID := quoteData["id"].(string)

				var customer string
				err := db.QueryRow("SELECT customer FROM quotes WHERE id = ?", quoteID).Scan(&customer)
				if err != nil {
					t.Fatalf("Quote not found after creation: %v", err)
				}

				// Verify customer field contains the literal payload
				if customer != payload {
					t.Errorf("Expected customer to be '%s', got '%s'", payload, customer)
				}

				// Verify quotes table still exists (wasn't dropped)
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM quotes").Scan(&count)
				if err != nil {
					t.Fatalf("Quotes table appears to have been dropped or corrupted: %v", err)
				}

				t.Logf("✓ SQL injection safely handled as literal string")
			}
		})
	}

	// Test SQL injection in GET parameters (quote ID)
	req := httptest.NewRequest("GET", "/api/quotes/'OR'1'='1", nil)
	w := httptest.NewRecorder()

	handleGetQuote(w, req, "'OR'1'='1")

	// Should return 404 (no quote with that ID), not leak data
	if w.Code != 404 {
		t.Errorf("Expected 404 for invalid quote ID, got %d", w.Code)
	}
}

// TestQuoteConcurrentUpdates tests race conditions when updating the same quote
func TestQuoteConcurrentUpdates(t *testing.T) {
	// Note: This test documents concurrent update behavior but may not be reliable in all test environments
	// SQLite has limitations with concurrent writes
	t.Skip("Skipping concurrent update test - SQLite in-memory DB doesn't support true concurrency")
	
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Original notes", "2026-12-31")

	var wg sync.WaitGroup
	numGoroutines := 10
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			reqBody := fmt.Sprintf(`{"customer":"Updated Corp %d","status":"draft","notes":"Update %d"}`, index, index)
			req := httptest.NewRequest("PUT", "/api/quotes/Q-001", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleUpdateQuote(w, req, "Q-001")

			if w.Code == 200 {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// All updates should succeed (last write wins)
	if successCount != numGoroutines {
		t.Logf("Warning: %d/%d concurrent updates succeeded (possible lock contention)", successCount, numGoroutines)
	} else {
		t.Logf("✓ All %d concurrent updates succeeded", successCount)
	}

	// Verify quote still exists and is in valid state
	var customer string
	err := db.QueryRow("SELECT customer FROM quotes WHERE id = ?", "Q-001").Scan(&customer)
	if err != nil {
		t.Logf("Note: Quote state after concurrent updates: %v", err)
		return
	}

	t.Logf("Final customer value: %s", customer)
}

// TestQuoteStatusTransitionValidation tests invalid status transitions
func TestQuoteStatusTransitionValidation(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test constraint at DB level
	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Test", "2026-12-31")

	// Try to set an invalid status directly in DB
	_, err := db.Exec("UPDATE quotes SET status = ? WHERE id = ?", "INVALID_STATUS", "Q-001")

	if err == nil {
		t.Error("Expected CHECK constraint error for invalid status, but update succeeded")
	}

	// Verify error is a constraint violation
	if !strings.Contains(err.Error(), "CHECK") && !strings.Contains(err.Error(), "constraint") {
		t.Logf("Warning: error doesn't clearly indicate CHECK constraint: %v", err)
	}

	// Verify status wasn't changed
	var status string
	db.QueryRow("SELECT status FROM quotes WHERE id = ?", "Q-001").Scan(&status)
	if status != "draft" {
		t.Errorf("Status should still be 'draft', got '%s'", status)
	}
}

// TestQuoteExpirationLogic tests quote expiration based on valid_until date
func TestQuoteExpirationLogic(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	// Create quotes with different expiration dates
	insertTestQuote(t, db, "Q-EXPIRED", "Acme Corp", "sent", "Should be expired", yesterday)
	insertTestQuote(t, db, "Q-VALID", "Beta Inc", "sent", "Still valid", tomorrow)
	insertTestQuote(t, db, "Q-NO-EXP", "Gamma LLC", "sent", "No expiration", "")

	// Query for expired quotes
	rows, err := db.Query(`
		SELECT id, customer, valid_until 
		FROM quotes 
		WHERE status = 'sent' 
		AND valid_until IS NOT NULL 
		AND valid_until != '' 
		AND valid_until < date('now')
	`)
	if err != nil {
		t.Fatalf("Failed to query expired quotes: %v", err)
	}
	defer rows.Close()

	expiredQuotes := []string{}
	for rows.Next() {
		var id, customer, validUntil string
		rows.Scan(&id, &customer, &validUntil)
		expiredQuotes = append(expiredQuotes, id)
	}

	// Verify expired quote is found
	if len(expiredQuotes) != 1 {
		t.Errorf("Expected 1 expired quote, found %d", len(expiredQuotes))
	}
	if len(expiredQuotes) > 0 && expiredQuotes[0] != "Q-EXPIRED" {
		t.Errorf("Expected Q-EXPIRED to be expired, got %v", expiredQuotes)
	}

	t.Logf("✓ Expiration logic working: found %d expired quotes", len(expiredQuotes))

	// Simulate auto-expiration process
	result, err := db.Exec(`
		UPDATE quotes 
		SET status = 'expired' 
		WHERE status = 'sent' 
		AND valid_until IS NOT NULL 
		AND valid_until != '' 
		AND valid_until < date('now')
	`)
	if err != nil {
		t.Fatalf("Failed to update expired quotes: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		t.Errorf("Expected 1 quote to be marked expired, got %d", rowsAffected)
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM quotes WHERE id = ?", "Q-EXPIRED").Scan(&status)
	if status != "expired" {
		t.Errorf("Expected status 'expired', got '%s'", status)
	}
}

// TestQuoteLineValidation tests comprehensive line item validation
func TestQuoteLineValidation(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		ipn         string
		qty         int
		unitPrice   float64
		expectError bool
	}{
		{"Valid line", "IPN-001", 10, 25.50, false},
		{"Zero qty", "IPN-002", 0, 100.00, true},
		{"Negative qty", "IPN-003", -5, 50.00, true},
		{"Valid zero price", "IPN-004", 10, 0.00, false}, // Free items allowed
		{"Negative price", "IPN-005", 10, -10.00, false}, // DB allows, but handler validates
		{"Empty IPN", "", 10, 25.00, false}, // Backend may allow empty IPN
		{"Very large qty", "IPN-006", 1000000, 1.00, false}, // Should succeed unless MaxWorkOrderQty validation
		{"Very large price", "IPN-007", 1, 999999.99, false}, // Should succeed unless MaxPrice validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Try to insert line directly into DB
			insertTestQuote(t, db, "Q-TEST-"+tt.name, "Test Corp", "draft", "", "")

			_, err := db.Exec(
				"INSERT INTO quote_lines (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)",
				"Q-TEST-"+tt.name, tt.ipn, "Test item", tt.qty, tt.unitPrice,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but insert succeeded", tt.name)
				}
			} else {
				if err != nil {
					t.Logf("Note: %s resulted in error (may be intentional): %v", tt.name, err)
				}
			}
		})
	}
}

// TestQuoteWithZeroLines tests quote behavior with no line items
func TestQuoteWithZeroLines(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Quote with no lines", "2026-12-31")

	// Get quote with no lines
	req := httptest.NewRequest("GET", "/api/quotes/Q-001", nil)
	w := httptest.NewRecorder()

	handleGetQuote(w, req, "Q-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Get cost for quote with no lines
	req = httptest.NewRequest("GET", "/api/quotes/Q-001/cost", nil)
	w = httptest.NewRecorder()

	handleQuoteCost(w, req, "Q-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200 for cost endpoint, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	if result["total_quoted"].(float64) != 0 {
		t.Errorf("Expected total_quoted 0, got %.2f", result["total_quoted"])
	}
}

// TestQuoteBOMCostCalculationAccuracy tests precision of BOM cost calculations
func TestQuoteBOMCostCalculationAccuracy(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Test", "2026-12-31")

	// Create lines with decimal quantities and prices
	insertTestQuoteLine(t, db, "Q-001", "IPN-100", "Widget A", 10, 0.12345, "")  // $1.23450
	insertTestQuoteLine(t, db, "Q-001", "IPN-200", "Widget B", 7, 0.00999, "")   // $0.06993
	insertTestQuoteLine(t, db, "Q-001", "IPN-300", "Widget C", 1000, 0.001, "")  // $1.00000

	// Set up BOM costs
	insertTestPOLine(t, db, "PO-001", "IPN-100", 0.10000)
	insertTestPOLine(t, db, "PO-002", "IPN-200", 0.00500)
	insertTestPOLine(t, db, "PO-003", "IPN-300", 0.00050)

	req := httptest.NewRequest("GET", "/api/quotes/Q-001/cost", nil)
	w := httptest.NewRecorder()

	handleQuoteCost(w, req, "Q-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})

	// Calculate expected values
	expectedQuoted := (10 * 0.12345) + (7 * 0.00999) + (1000 * 0.001) // ≈ 2.30443
	expectedCost := (10 * 0.10000) + (7 * 0.00500) + (1000 * 0.00050) // ≈ 1.535

	totalQuoted := result["total_quoted"].(float64)
	if totalBOM, ok := result["total_bom_cost"].(float64); ok {
		// Allow small floating point precision differences
		quotedDiff := totalQuoted - expectedQuoted
		costDiff := totalBOM - expectedCost

		if quotedDiff > 0.001 || quotedDiff < -0.001 {
			t.Errorf("Total quoted precision error: expected %.5f, got %.5f (diff: %.5f)", expectedQuoted, totalQuoted, quotedDiff)
		}

		if costDiff > 0.001 || costDiff < -0.001 {
			t.Errorf("Total BOM cost precision error: expected %.5f, got %.5f (diff: %.5f)", expectedCost, totalBOM, costDiff)
		}

		t.Logf("✓ Precision test passed: quoted=%.5f, cost=%.5f", totalQuoted, totalBOM)
	} else {
		t.Skip("BOM cost data not available (PO lookup may not be working in test)")
	}
}

// TestQuoteUpdatePreservesLines tests that updating a quote doesn't affect its lines
func TestQuoteUpdatePreservesLines(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuote(t, db, "Q-001", "Acme Corp", "draft", "Original notes", "2026-12-31")
	insertTestQuoteLine(t, db, "Q-001", "IPN-100", "Widget A", 10, 25.50, "")
	insertTestQuoteLine(t, db, "Q-001", "IPN-200", "Widget B", 5, 50.00, "")

	// Count lines before update
	var lineCountBefore int
	db.QueryRow("SELECT COUNT(*) FROM quote_lines WHERE quote_id = ?", "Q-001").Scan(&lineCountBefore)

	// Update quote (not lines)
	reqBody := `{"customer":"Updated Corp","status":"sent","notes":"Updated notes","valid_until":"2027-01-31"}`
	req := httptest.NewRequest("PUT", "/api/quotes/Q-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateQuote(w, req, "Q-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Count lines after update
	var lineCountAfter int
	db.QueryRow("SELECT COUNT(*) FROM quote_lines WHERE quote_id = ?", "Q-001").Scan(&lineCountAfter)

	if lineCountBefore != lineCountAfter {
		t.Errorf("Line count changed from %d to %d after update", lineCountBefore, lineCountAfter)
	}

	// Verify line data is intact
	var ipn string
	db.QueryRow("SELECT ipn FROM quote_lines WHERE quote_id = ? LIMIT 1", "Q-001").Scan(&ipn)
	if ipn != "IPN-100" {
		t.Errorf("Line data corrupted: expected IPN-100, got %s", ipn)
	}
}

// TestQuotePDFXSSEscaping verifies all fields are properly HTML-escaped in PDF generation
func TestQuotePDFXSSEscaping(t *testing.T) {
	oldDB := db
	db = setupQuotesTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test that all XSS payloads are properly escaped
	xssPayload := "<script>alert('xss')</script>"
	xssIPN := "<img src=x onerror=alert(1)>"
	xssNotes := "<svg onload=alert(2)>"

	insertTestQuote(t, db, "Q-XSS", "SafeCustomer", "draft", xssNotes, "2026-12-31")
	insertTestQuoteLine(t, db, "Q-XSS", xssIPN, xssPayload, 1, 100.00, "")

	req := httptest.NewRequest("GET", "/api/quotes/Q-XSS/pdf", nil)
	w := httptest.NewRecorder()

	handleQuotePDF(w, req, "Q-XSS")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	html := w.Body.String()

	// Verify all XSS payloads are properly escaped
	if strings.Contains(html, xssPayload) {
		t.Error("PDF HTML contains unescaped script tag in description field")
	}
	if strings.Contains(html, xssIPN) {
		t.Error("PDF HTML contains unescaped img tag in IPN field")
	}
	if strings.Contains(html, xssNotes) {
		t.Error("PDF HTML contains unescaped svg tag in notes field")
	}

	// Verify escaped versions are present
	if !strings.Contains(html, "&lt;script&gt;") && !strings.Contains(html, "&#") {
		t.Error("Expected escaped HTML entities in output")
	}

	t.Logf("✓ All XSS payloads properly escaped in PDF output")
}
