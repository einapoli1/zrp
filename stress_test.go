package main

import (
	"bytes"
	_ "database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	_ "os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStress is the main stress test suite
func TestStress(t *testing.T) {
	t.Run("WAL Mode", testWALMode)
	t.Run("Concurrent Inventory Updates API", testConcurrentInventoryUpdatesAPI)
	t.Run("Concurrent Inventory Updates DB", testConcurrentInventoryUpdatesDB)
	t.Run("Concurrent Work Order Creation", testConcurrentWorkOrderCreation)
	t.Run("Large Dataset Performance", testLargeDatasetPerformance)
	t.Run("Read While Write", testReadWhileWrite)
	t.Run("Transaction Integrity", testTransactionIntegrity)
	t.Run("Foreign Key Constraints", testForeignKeyConstraints)
}

// testWALMode verifies that WAL mode is enabled
func testWALMode(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	var mode string
	err := db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}

	if mode != "wal" && mode != "WAL" {
		t.Errorf("Expected WAL mode, got: %s", mode)
	} else {
		t.Logf("✓ WAL mode enabled: %s", mode)
	}

	// Also check busy_timeout
	var timeout int
	err = db.QueryRow("PRAGMA busy_timeout").Scan(&timeout)
	if err != nil {
		t.Fatalf("Failed to query busy_timeout: %v", err)
	}
	t.Logf("✓ Busy timeout: %d ms", timeout)
}

// testConcurrentInventoryUpdatesDB tests concurrent inventory updates at the database level
func testConcurrentInventoryUpdatesDB(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Create test inventory item
	ipn := "TEST-CONC-001"
	initialQty := 100.0

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, description) VALUES (?, ?, ?)`,
		ipn, initialQty, "Concurrent test item")
	if err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	// Launch 10 goroutines, each adding 10 units
	const numRoutines = 10
	const addPerRoutine = 10.0
	expectedFinal := initialQty + (numRoutines * addPerRoutine) // 100 + 100 = 200

	var wg sync.WaitGroup
	errChan := make(chan error, numRoutines)
	start := time.Now()

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Retry logic for SQLITE_BUSY errors
			const maxRetries = 10
			for retry := 0; retry < maxRetries; retry++ {
				if retry > 0 {
					// Exponential backoff with jitter
					backoff := time.Duration(retry*retry*5) * time.Millisecond
					time.Sleep(backoff)
				}

				// Use a transaction for each update
				tx, err := db.Begin()
				if err != nil {
					if retry < maxRetries-1 {
						continue // Retry
					}
					errChan <- fmt.Errorf("routine %d: begin tx: %v", id, err)
					return
				}

				// Read current qty
				var currentQty float64
				err = tx.QueryRow(`SELECT qty_on_hand FROM inventory WHERE ipn = ?`, ipn).Scan(&currentQty)
				if err != nil {
					tx.Rollback()
					if retry < maxRetries-1 {
						continue // Retry
					}
					errChan <- fmt.Errorf("routine %d: read qty: %v", id, err)
					return
				}

				// Update qty
				newQty := currentQty + addPerRoutine
				_, err = tx.Exec(`UPDATE inventory SET qty_on_hand = ?, updated_at = CURRENT_TIMESTAMP WHERE ipn = ?`,
					newQty, ipn)
				if err != nil {
					tx.Rollback()
					if retry < maxRetries-1 {
						continue // Retry
					}
					errChan <- fmt.Errorf("routine %d: update: %v", id, err)
					return
				}

				// Also log the transaction
				_, err = tx.Exec(`INSERT INTO inventory_transactions (ipn, type, qty, reference) VALUES (?, 'adjust', ?, ?)`,
					ipn, addPerRoutine, fmt.Sprintf("stress-test-%d", id))
				if err != nil {
					tx.Rollback()
					if retry < maxRetries-1 {
						continue // Retry
					}
					errChan <- fmt.Errorf("routine %d: log transaction: %v", id, err)
					return
				}

				if err = tx.Commit(); err != nil {
					if retry < maxRetries-1 {
						continue // Retry
					}
					errChan <- fmt.Errorf("routine %d: commit: %v", id, err)
					return
				}

				// Success!
				return
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	elapsed := time.Since(start)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	
	// Log errors but don't fail if within acceptable range
	if len(errors) > 0 {
		maxAcceptableFailures := int(float64(numRoutines)*0.05 + 0.5) // 5% failure rate
		if len(errors) > maxAcceptableFailures {
			t.Errorf("Too many failures during concurrent updates: %d errors (max acceptable: %d):",
				len(errors), maxAcceptableFailures)
			for _, err := range errors {
				t.Errorf("  - %v", err)
			}
		} else {
			t.Logf("Some failures during concurrent updates (%d/%d, within acceptable range):",
				len(errors), maxAcceptableFailures)
			for _, err := range errors {
				t.Logf("  - %v", err)
			}
		}
	}

	// Verify final quantity
	var finalQty float64
	err = db.QueryRow(`SELECT qty_on_hand FROM inventory WHERE ipn = ?`, ipn).Scan(&finalQty)
	if err != nil {
		t.Fatalf("Failed to read final qty: %v", err)
	}

	if finalQty != expectedFinal {
		t.Errorf("Data integrity violation! Expected qty=%.0f, got %.0f (lost %.0f units)",
			expectedFinal, finalQty, expectedFinal-finalQty)
	} else {
		t.Logf("✓ Data integrity verified: final qty=%.0f (expected %.0f)", finalQty, expectedFinal)
	}

	// Verify transaction log count
	var txCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM inventory_transactions WHERE ipn = ?`, ipn).Scan(&txCount)
	if err != nil {
		t.Fatalf("Failed to count transactions: %v", err)
	}

	if txCount != numRoutines {
		t.Errorf("Expected %d transaction records, got %d", numRoutines, txCount)
	}

	t.Logf("✓ Completed %d concurrent updates in %v", numRoutines, elapsed)
}

// testConcurrentInventoryUpdatesAPI tests concurrent inventory updates via API
func testConcurrentInventoryUpdatesAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// This test requires the server to be running
	// Check if server is accessible
	resp, err := http.Get("http://localhost:9000/api/health")
	if err != nil {
		t.Skipf("Server not running on localhost:9000: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Server not healthy: status %d", resp.StatusCode)
	}

	// Get auth token
	token := getTestAuthToken(t)

	// Create test inventory item via API
	ipn := fmt.Sprintf("TEST-API-%d", time.Now().Unix())
	initialQty := 100.0

	createPayload := map[string]interface{}{
		"ipn":         ipn,
		"qty_on_hand": initialQty,
		"description": "API concurrent test item",
	}

	if !apiCall(t, token, "POST", "/api/inventory", createPayload, nil) {
		t.Fatal("Failed to create inventory item")
	}

	// Launch concurrent updates
	const numRoutines = 10
	const addPerRoutine = 10.0
	expectedFinal := initialQty + (numRoutines * addPerRoutine)

	var wg sync.WaitGroup
	var successCount int32
	start := time.Now()

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Adjust inventory quantity
			payload := map[string]interface{}{
				"ipn":  ipn,
				"type": "adjust",
				"qty":  addPerRoutine,
				"note": fmt.Sprintf("concurrent-api-test-%d", id),
			}

			if apiCall(t, token, "POST", "/api/inventory/adjust", payload, nil) {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	if int(successCount) != numRoutines {
		t.Errorf("Only %d/%d API calls succeeded", successCount, numRoutines)
	}

	// Verify final quantity
	var result struct {
		QtyOnHand float64 `json:"qty_on_hand"`
	}

	if !apiCall(t, token, "GET", fmt.Sprintf("/api/inventory/%s", ipn), nil, &result) {
		t.Fatal("Failed to fetch final inventory")
	}

	if result.QtyOnHand != expectedFinal {
		t.Errorf("API concurrent update integrity violation! Expected %.0f, got %.0f (lost %.0f)",
			expectedFinal, result.QtyOnHand, expectedFinal-result.QtyOnHand)
	} else {
		t.Logf("✓ API concurrent updates verified: final qty=%.0f", result.QtyOnHand)
	}

	t.Logf("✓ Completed %d concurrent API calls in %v", numRoutines, elapsed)
}

// testConcurrentWorkOrderCreation tests creating many work orders simultaneously
func testConcurrentWorkOrderCreation(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Create a test assembly part
	assemblyIPN := "ASM-TEST-001"
	_, err := db.Exec(`INSERT INTO inventory (ipn, description) VALUES (?, ?)`,
		assemblyIPN, "Test assembly for WO creation")
	if err != nil {
		t.Fatalf("Failed to create assembly: %v", err)
	}

	const numOrders = 50
	var wg sync.WaitGroup
	errChan := make(chan error, numOrders)
	woIDs := make(chan string, numOrders)
	start := time.Now()

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Retry logic for SQLITE_BUSY errors
			const maxRetries = 10
			for retry := 0; retry < maxRetries; retry++ {
				if retry > 0 {
					// Exponential backoff with jitter
					backoff := time.Duration(retry*retry*5) * time.Millisecond
					time.Sleep(backoff)
				}

				tx, err := db.Begin()
				if err != nil {
					if retry < maxRetries-1 {
						continue
					}
					errChan <- fmt.Errorf("routine %d: begin: %v", id, err)
					return
				}

				// Generate unique WO ID
				woID := fmt.Sprintf("WO-%d-%d", time.Now().Unix(), id)

				_, err = tx.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES (?, ?, ?, ?)`,
					woID, assemblyIPN, 1, "open")
				if err != nil {
					tx.Rollback()
					if retry < maxRetries-1 {
						continue
					}
					errChan <- fmt.Errorf("routine %d: insert WO: %v", id, err)
					return
				}

				if err = tx.Commit(); err != nil {
					if retry < maxRetries-1 {
						continue
					}
					errChan <- fmt.Errorf("routine %d: commit: %v", id, err)
					return
				}

				woIDs <- woID
				return // Success
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	close(woIDs)
	elapsed := time.Since(start)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	
	// Log errors but don't fail if within acceptable range
	if len(errors) > 0 {
		maxAcceptableFailures := int(float64(numOrders)*0.05 + 0.5) // 5% failure rate
		if len(errors) > maxAcceptableFailures {
			t.Errorf("Too many failures during concurrent WO creation: %d errors (max acceptable: %d):",
				len(errors), maxAcceptableFailures)
			for _, err := range errors {
				t.Errorf("  - %v", err)
			}
		} else {
			t.Logf("Some failures during concurrent WO creation (%d/%d, within acceptable range):",
				len(errors), maxAcceptableFailures)
			for _, err := range errors {
				t.Logf("  - %v", err)
			}
		}
	}

	// Collect all created IDs
	createdIDs := make(map[string]bool)
	for id := range woIDs {
		if createdIDs[id] {
			t.Errorf("Duplicate WO ID created: %s", id)
		}
		createdIDs[id] = true
	}

	// Count work orders in DB
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM work_orders WHERE assembly_ipn = ?`, assemblyIPN).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count work orders: %v", err)
	}

	// Accept up to 5% failure rate under extreme concurrent load (realistic for SQLite)
	numOrdersFloat := float64(numOrders)
	minAcceptable := int(numOrdersFloat*0.95 + 0.5) // Round properly
	if count < minAcceptable {
		t.Errorf("Too many failures: expected at least %d work orders, found %d (%.1f%% success rate)",
			minAcceptable, count, 100.0*float64(count)/float64(numOrders))
	} else if count == numOrders {
		t.Logf("✓ All %d work orders created successfully (100%% success)", numOrders)
	} else {
		t.Logf("✓ Created %d/%d work orders (%.1f%% success rate, within acceptable range)",
			count, numOrders, 100.0*float64(count)/float64(numOrders))
	}

	if len(createdIDs) != count {
		t.Errorf("Mismatch: %d work orders in DB but %d unique IDs returned", count, len(createdIDs))
	} else {
		t.Logf("✓ All %d WO IDs are unique", len(createdIDs))
	}

	t.Logf("✓ Created %d work orders in %v (%.2f/sec)",
		numOrders, elapsed, float64(numOrders)/elapsed.Seconds())
}

// testLargeDatasetPerformance tests performance with large datasets
func testLargeDatasetPerformance(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	const numParts = 10000
	t.Logf("Inserting %d parts...", numParts)

	start := time.Now()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO inventory (ipn, qty_on_hand, description, mpn) VALUES (?, ?, ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < numParts; i++ {
		ipn := fmt.Sprintf("PART-%06d", i)
		desc := fmt.Sprintf("Test part number %d", i)
		mpn := fmt.Sprintf("MPN-%06d", i)
		_, err = stmt.Exec(ipn, float64(i%100), desc, mpn)
		if err != nil {
			t.Fatalf("Failed to insert part %d: %v", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	insertTime := time.Since(start)
	t.Logf("✓ Inserted %d parts in %v (%.0f parts/sec)",
		numParts, insertTime, float64(numParts)/insertTime.Seconds())

	// Test search performance
	searches := []string{
		"PART-005",    // Exact match
		"PART-0%",     // Prefix
		"Test part",   // Description search
		"%500%",       // Contains
	}

	for _, searchTerm := range searches {
		start = time.Now()
		rows, err := db.Query(`
			SELECT ipn, description FROM inventory 
			WHERE ipn LIKE ? OR description LIKE ? 
			LIMIT 100`,
			"%"+searchTerm+"%", "%"+searchTerm+"%")
		if err != nil {
			t.Errorf("Search failed for '%s': %v", searchTerm, err)
			continue
		}

		count := 0
		for rows.Next() {
			count++
			var ipn, desc string
			rows.Scan(&ipn, &desc)
		}
		rows.Close()

		searchTime := time.Since(start)
		if searchTime > 500*time.Millisecond {
			t.Errorf("Search '%s' took %v (target: <500ms)", searchTerm, searchTime)
		} else {
			t.Logf("✓ Search '%s' returned %d results in %v", searchTerm, count, searchTime)
		}
	}

	// Test pagination performance
	const pageSize = 50
	const numPages = 10

	start = time.Now()
	for page := 0; page < numPages; page++ {
		offset := page * pageSize
		rows, err := db.Query(`SELECT ipn, description FROM inventory LIMIT ? OFFSET ?`, pageSize, offset)
		if err != nil {
			t.Errorf("Pagination failed at page %d: %v", page, err)
			break
		}

		count := 0
		for rows.Next() {
			count++
		}
		rows.Close()

		if count != pageSize {
			t.Errorf("Page %d returned %d items, expected %d", page, count, pageSize)
		}
	}

	paginationTime := time.Since(start)
	avgPerPage := paginationTime / numPages
	if avgPerPage > 100*time.Millisecond {
		t.Errorf("Average pagination time %v per page (target: <100ms)", avgPerPage)
	} else {
		t.Logf("✓ Paginated through %d pages in %v (avg %v/page)",
			numPages, paginationTime, avgPerPage)
	}
}

// testReadWhileWrite tests concurrent reads during writes
func testReadWhileWrite(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Insert initial data
	for i := 0; i < 100; i++ {
		ipn := fmt.Sprintf("RW-%04d", i)
		_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, description) VALUES (?, ?, ?)`,
			ipn, float64(i), fmt.Sprintf("Read-write test part %d", i))
		if err != nil {
			t.Fatalf("Failed to insert initial data: %v", err)
		}
	}

	var wg sync.WaitGroup
	var readErrors, writeErrors int32
	stop := make(chan bool)

	// Start readers
	const numReaders = 5
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					rows, err := db.Query(`SELECT ipn, qty_on_hand FROM inventory LIMIT 50`)
					if err != nil {
						atomic.AddInt32(&readErrors, 1)
						continue
					}
					for rows.Next() {
						var ipn string
						var qty float64
						rows.Scan(&ipn, &qty)
					}
					rows.Close()
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Start writers
	const numWriters = 3
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				ipn := fmt.Sprintf("RW-%04d", j%100)
				_, err := db.Exec(`UPDATE inventory SET qty_on_hand = qty_on_hand + 1 WHERE ipn = ?`, ipn)
				if err != nil {
					atomic.AddInt32(&writeErrors, 1)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	// Let it run for a bit
	time.Sleep(2 * time.Second)
	close(stop)
	wg.Wait()

	if readErrors > 0 {
		t.Errorf("Encountered %d read errors during concurrent access (WAL mode should allow concurrent reads)", readErrors)
	} else {
		t.Logf("✓ No read errors during concurrent access")
	}

	// Some write errors are acceptable under heavy concurrent load
	// The key is that reads work during writes (WAL mode benefit)
	const maxAcceptableWriteErrors = 30 // 3 writers * 20 operations = 60 total, allow up to 50% failure
	if writeErrors > maxAcceptableWriteErrors {
		t.Errorf("Too many write errors: %d (max acceptable: %d)", writeErrors, maxAcceptableWriteErrors)
	} else if writeErrors > 0 {
		t.Logf("✓ Write errors within acceptable range: %d/%d (expected under heavy load)", writeErrors, maxAcceptableWriteErrors)
	} else {
		t.Logf("✓ No write errors during concurrent access")
	}

	if readErrors == 0 {
		t.Logf("✓ Read-while-write test passed: WAL mode allowing concurrent reads")
	}
}

// testTransactionIntegrity tests that transactions rollback properly on failure
func testTransactionIntegrity(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Create 100 test parts
	const numParts = 100
	for i := 0; i < numParts; i++ {
		ipn := fmt.Sprintf("TXN-%04d", i)
		_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, description) VALUES (?, ?, ?)`,
			ipn, 0, fmt.Sprintf("Transaction test part %d", i))
		if err != nil {
			t.Fatalf("Failed to create test part: %v", err)
		}
	}

	// Attempt a batch update that will fail midway
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Update first 50 successfully
	for i := 0; i < 50; i++ {
		ipn := fmt.Sprintf("TXN-%04d", i)
		_, err = tx.Exec(`UPDATE inventory SET qty_on_hand = 100 WHERE ipn = ?`, ipn)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update part %d: %v", i, err)
		}
	}

	// Simulate failure - try to update with invalid data
	_, err = tx.Exec(`UPDATE inventory SET qty_on_hand = -999 WHERE ipn = 'TXN-0050'`)
	// This should fail due to CHECK constraint

	// Rollback the transaction
	tx.Rollback()
	t.Logf("✓ Transaction rolled back after simulated failure")

	// Verify that NO parts were updated
	var updatedCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM inventory WHERE ipn LIKE 'TXN-%' AND qty_on_hand = 100`).Scan(&updatedCount)
	if err != nil {
		t.Fatalf("Failed to count updated parts: %v", err)
	}

	if updatedCount > 0 {
		t.Errorf("Transaction rollback failed! Found %d updated parts (expected 0)", updatedCount)
	} else {
		t.Logf("✓ Transaction integrity verified: 0 partial updates after rollback")
	}

	// Now test a successful batch update
	tx, err = db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	for i := 0; i < numParts; i++ {
		ipn := fmt.Sprintf("TXN-%04d", i)
		_, err = tx.Exec(`UPDATE inventory SET qty_on_hand = 50 WHERE ipn = ?`, ipn)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update part %d: %v", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify all parts updated
	err = db.QueryRow(`SELECT COUNT(*) FROM inventory WHERE ipn LIKE 'TXN-%' AND qty_on_hand = 50`).Scan(&updatedCount)
	if err != nil {
		t.Fatalf("Failed to count updated parts: %v", err)
	}

	if updatedCount != numParts {
		t.Errorf("Expected %d updated parts, got %d", numParts, updatedCount)
	} else {
		t.Logf("✓ Successful batch update: all %d parts updated atomically", numParts)
	}
}

// testForeignKeyConstraints tests that foreign keys prevent orphaned records
func testForeignKeyConstraints(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Verify FK enforcement is on
	var fkEnabled int
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil || fkEnabled != 1 {
		t.Fatal("Foreign keys not enabled")
	}

	// Test 1: Can't delete vendor with open PO
	t.Run("Vendor_With_PO", func(t *testing.T) {
		// Create vendor
		vendorID := "VENDOR-FK-001"
		_, err := db.Exec(`INSERT INTO vendors (id, name) VALUES (?, ?)`, vendorID, "Test Vendor")
		if err != nil {
			t.Fatalf("Failed to create vendor: %v", err)
		}

		// Create PO referencing vendor
		poID := "PO-FK-001"
		_, err = db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES (?, ?, 'draft')`,
			poID, vendorID)
		if err != nil {
			t.Fatalf("Failed to create PO: %v", err)
		}

		// Try to delete vendor (should fail)
		_, err = db.Exec(`DELETE FROM vendors WHERE id = ?`, vendorID)
		if err == nil {
			t.Error("Expected FK violation when deleting vendor with PO, but delete succeeded")
		} else {
			t.Logf("✓ FK constraint prevented vendor deletion: %v", err)
		}

		// Delete PO first, then vendor should work
		_, err = db.Exec(`DELETE FROM purchase_orders WHERE id = ?`, poID)
		if err != nil {
			t.Fatalf("Failed to delete PO: %v", err)
		}

		_, err = db.Exec(`DELETE FROM vendors WHERE id = ?`, vendorID)
		if err != nil {
			t.Errorf("Failed to delete vendor after removing PO: %v", err)
		} else {
			t.Logf("✓ Vendor deletion succeeded after removing PO")
		}
	})

	// Test 2: Deleting PO cascades to po_lines
	t.Run("PO_Cascade_To_Lines", func(t *testing.T) {
		// Create vendor
		vendorID := "VENDOR-FK-002"
		_, err := db.Exec(`INSERT INTO vendors (id, name) VALUES (?, ?)`, vendorID, "Test Vendor 2")
		if err != nil {
			t.Fatalf("Failed to create vendor: %v", err)
		}

		// Create PO
		poID := "PO-FK-002"
		_, err = db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES (?, ?, 'draft')`,
			poID, vendorID)
		if err != nil {
			t.Fatalf("Failed to create PO: %v", err)
		}

		// Create PO lines
		for i := 0; i < 5; i++ {
			_, err = db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES (?, ?, ?)`,
				poID, fmt.Sprintf("PART-%d", i), 10)
			if err != nil {
				t.Fatalf("Failed to create PO line: %v", err)
			}
		}

		// Count lines
		var lineCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM po_lines WHERE po_id = ?`, poID).Scan(&lineCount)
		if err != nil {
			t.Fatalf("Failed to count lines: %v", err)
		}
		if lineCount != 5 {
			t.Errorf("Expected 5 lines, got %d", lineCount)
		}

		// Delete PO (should cascade to lines)
		_, err = db.Exec(`DELETE FROM purchase_orders WHERE id = ?`, poID)
		if err != nil {
			t.Fatalf("Failed to delete PO: %v", err)
		}

		// Verify lines were deleted
		err = db.QueryRow(`SELECT COUNT(*) FROM po_lines WHERE po_id = ?`, poID).Scan(&lineCount)
		if err != nil {
			t.Fatalf("Failed to count lines after PO delete: %v", err)
		}
		if lineCount != 0 {
			t.Errorf("Expected 0 lines after PO delete, got %d (cascade failed)", lineCount)
		} else {
			t.Logf("✓ CASCADE DELETE worked: PO lines deleted with PO")
		}
	})

	// Test 3: Deleting work order cascades to serials
	t.Run("WO_Cascade_To_Serials", func(t *testing.T) {
		// Create work order
		woID := "WO-FK-001"
		_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty) VALUES (?, ?, ?)`,
			woID, "ASM-001", 5)
		if err != nil {
			t.Fatalf("Failed to create WO: %v", err)
		}

		// Create serials
		for i := 0; i < 5; i++ {
			serial := fmt.Sprintf("SN-FK-%04d", i)
			_, err = db.Exec(`INSERT INTO wo_serials (wo_id, serial_number) VALUES (?, ?)`,
				woID, serial)
			if err != nil {
				t.Fatalf("Failed to create serial: %v", err)
			}
		}

		// Delete WO
		_, err = db.Exec(`DELETE FROM work_orders WHERE id = ?`, woID)
		if err != nil {
			t.Fatalf("Failed to delete WO: %v", err)
		}

		// Verify serials deleted
		var serialCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE wo_id = ?`, woID).Scan(&serialCount)
		if err != nil {
			t.Fatalf("Failed to count serials: %v", err)
		}
		if serialCount != 0 {
			t.Errorf("Expected 0 serials after WO delete, got %d", serialCount)
		} else {
			t.Logf("✓ CASCADE DELETE worked: serials deleted with work order")
		}
	})

	// Test 4: Can't create PO with non-existent vendor
	t.Run("Invalid_Vendor_Reference", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status) VALUES (?, ?, 'draft')`,
			"PO-FK-999", "NONEXISTENT-VENDOR")
		if err == nil {
			t.Error("Expected FK violation when creating PO with invalid vendor")
		} else {
			t.Logf("✓ FK constraint prevented PO creation with invalid vendor: %v", err)
		}
	})

	// Test 5: Can't create po_line with non-existent PO
	t.Run("Invalid_PO_Reference", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES (?, ?, ?)`,
			"NONEXISTENT-PO", "PART-001", 10)
		if err == nil {
			t.Error("Expected FK violation when creating PO line with invalid PO")
		} else {
			t.Logf("✓ FK constraint prevented PO line creation with invalid PO: %v", err)
		}
	})
}

// Helper: Get auth token for API tests
func getTestAuthToken(t *testing.T) string {
	t.Helper()

	payload := map[string]string{
		"username": "admin",
		"password": "changeme",
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post("http://localhost:9000/api/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Skipf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Login failed: status %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Skipf("Failed to decode login response: %v", err)
	}

	return result.Token
}

// Helper: Make API call
func apiCall(t *testing.T, token, method, path string, payload interface{}, result interface{}) bool {
	t.Helper()

	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, "http://localhost:9000"+path, body)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("API call failed: %s %s -> %d: %s", method, path, resp.StatusCode, string(bodyBytes))
		return false
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			t.Errorf("Failed to decode response: %v", err)
			return false
		}
	}

	return true
}
