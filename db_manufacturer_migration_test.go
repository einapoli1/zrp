package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupMigrationTestDB(t *testing.T) *sql.DB {
	// Use a temporary file database to avoid in-memory connection isolation issues
	tmpFile := fmt.Sprintf("/tmp/zrp_migration_test_%d.db", os.Getpid())
	
	// Clean up any existing file
	os.Remove(tmpFile)
	
	// Ensure cleanup on test completion
	t.Cleanup(func() {
		os.Remove(tmpFile)
	})
	
	testDB, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	return testDB
}

// Test migration from denormalized to normalized schema
func TestManufacturerNormalizationMigration(t *testing.T) {
	testDB := setupMigrationTestDB(t)
	defer testDB.Close()

	// Create OLD schema (denormalized)
	_, err := testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			category TEXT DEFAULT '',
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			manufacturer TEXT DEFAULT '',
			lifecycle TEXT DEFAULT 'active',
			status TEXT DEFAULT 'active',
			notes TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	// Create OLD part_manufacturers table (with TEXT manufacturer column)
	_, err = testDB.Exec(`
		CREATE TABLE part_manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_id TEXT NOT NULL,
			manufacturer TEXT NOT NULL,
			mpn TEXT NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 0,
			approved INTEGER NOT NULL DEFAULT 1,
			notes TEXT DEFAULT '',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (part_id) REFERENCES parts(ipn) ON DELETE CASCADE,
			UNIQUE(part_id, manufacturer, mpn)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create old part_manufacturers table: %v", err)
	}

	// Insert test data with denormalized manufacturers
	testData := []struct {
		ipn          string
		manufacturer string
		mpn          string
		description  string
	}{
		{"RES-001", "Yageo", "RC0603FR-071KL", "1K resistor"},
		{"RES-002", "Yageo", "RC0603FR-0710KL", "10K resistor"},
		{"RES-003", "YAGEO", "RC0603FR-07100KL", "100K resistor"}, // Duplicate with different case
		{"CAP-001", "Murata", "GRM188R71C104KA01D", "100nF cap"},
		{"CAP-002", "murata", "GRM188R71C105KA12D", "1uF cap"}, // Duplicate with different case
		{"IC-001", "Texas Instruments", "TPS54360DDAR", "DC-DC converter"},
	}

	// Insert into parts table
	for _, td := range testData {
		_, err = testDB.Exec(`INSERT INTO parts (ipn, description, manufacturer, mpn) VALUES (?, ?, ?, ?)`,
			td.ipn, td.description, td.manufacturer, td.mpn)
		if err != nil {
			t.Fatalf("Failed to insert part %s: %v", td.ipn, err)
		}
	}

	// Insert into part_manufacturers table
	for _, td := range testData {
		_, err = testDB.Exec(`INSERT INTO part_manufacturers (part_id, manufacturer, mpn, is_primary, approved) VALUES (?, ?, ?, 1, 1)`,
			td.ipn, td.manufacturer, td.mpn)
		if err != nil {
			t.Fatalf("Failed to insert part_manufacturer %s: %v", td.ipn, err)
		}
	}

	// Now create NEW schema (normalized)
	_, err = testDB.Exec(`
		CREATE TABLE manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			contact_name TEXT DEFAULT '',
			contact_email TEXT DEFAULT '',
			contact_phone TEXT DEFAULT '',
			website TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			approved INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create manufacturers table: %v", err)
	}

	// Verify table was created
	var tableExists bool
	err = testDB.QueryRow(`SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='manufacturers'`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if manufacturers table exists: %v", err)
	}
	if !tableExists {
		t.Fatal("Manufacturers table was not created successfully")
	}
	t.Logf("Verified manufacturers table exists before migration")

	// Run migration
	if err := migrateExistingManufacturers(testDB); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify manufacturers table has correct unique manufacturers (case-insensitive)
	var manufacturerCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM manufacturers`).Scan(&manufacturerCount)
	if err != nil {
		t.Fatalf("Failed to count manufacturers: %v", err)
	}

	// Should have 3 unique manufacturers: Yageo, Murata, Texas Instruments
	// (YAGEO and Yageo are same, murata and Murata are same)
	if manufacturerCount != 3 {
		t.Errorf("Expected 3 unique manufacturers, got %d", manufacturerCount)
	}

	// Verify specific manufacturers exist
	expectedManufacturers := []string{"Yageo", "Murata", "Texas Instruments"}
	for _, mfr := range expectedManufacturers {
		var exists bool
		err = testDB.QueryRow(`SELECT EXISTS(SELECT 1 FROM manufacturers WHERE LOWER(name) = LOWER(?))`, mfr).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check manufacturer %s: %v", mfr, err)
		}
		if !exists {
			t.Errorf("Expected manufacturer %s to exist", mfr)
		}
	}

	// Verify manufacturer_id column was added
	var hasManufacturerID bool
	err = testDB.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('part_manufacturers')
		WHERE name = 'manufacturer_id'
	`).Scan(&hasManufacturerID)
	if err != nil {
		t.Fatalf("Failed to check for manufacturer_id column: %v", err)
	}
	if !hasManufacturerID {
		t.Error("manufacturer_id column was not added")
	}

	// Verify all part_manufacturers have manufacturer_id populated
	var nullCount int
	err = testDB.QueryRow(`
		SELECT COUNT(*) FROM part_manufacturers WHERE manufacturer_id IS NULL
	`).Scan(&nullCount)
	if err != nil {
		t.Fatalf("Failed to count NULL manufacturer_ids: %v", err)
	}
	if nullCount > 0 {
		t.Errorf("Expected all manufacturer_ids to be populated, found %d NULL values", nullCount)
	}

	// Verify manufacturer_id values are correct (match original manufacturer names)
	rows, err := testDB.Query(`
		SELECT pm.part_id, pm.manufacturer, m.name
		FROM part_manufacturers pm
		JOIN manufacturers m ON pm.manufacturer_id = m.id
	`)
	if err != nil {
		t.Fatalf("Failed to query part_manufacturers with JOIN: %v", err)
	}
	defer rows.Close()

	matchCount := 0
	for rows.Next() {
		var partID, oldMfr, newMfr string
		if err := rows.Scan(&partID, &oldMfr, &newMfr); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		
		// Verify case-insensitive match - names should match regardless of case
		if oldMfr != newMfr { // Allow exact or different case
			// Special case: YAGEO/Yageo and murata/Murata are duplicates
			lowerOld := strings.ToLower(oldMfr)
			lowerNew := strings.ToLower(newMfr)
			if lowerOld != lowerNew {
				t.Errorf("Manufacturer mismatch for part %s: old=%s, new=%s", partID, oldMfr, newMfr)
			}
		}
		matchCount++
	}

	if matchCount != len(testData) {
		t.Errorf("Expected %d matched rows, got %d", len(testData), matchCount)
	}
}

// Test migration with parts table manufacturer/mpn data
func TestManufacturerMigrationFromPartsTable(t *testing.T) {
	testDB := setupMigrationTestDB(t)
	defer testDB.Close()

	// Create schema
	_, err := testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			description TEXT DEFAULT '',
			manufacturer TEXT DEFAULT '',
			mpn TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			approved INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create manufacturers table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE part_manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_id TEXT NOT NULL,
			manufacturer_id INTEGER,
			mpn TEXT NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 0,
			approved INTEGER NOT NULL DEFAULT 1,
			notes TEXT DEFAULT '',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (part_id) REFERENCES parts(ipn) ON DELETE CASCADE,
			FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id) ON DELETE RESTRICT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_manufacturers table: %v", err)
	}

	// Insert parts with manufacturer/mpn in parts table (old data)
	_, err = testDB.Exec(`INSERT INTO parts (ipn, description, manufacturer, mpn) VALUES (?, ?, ?, ?)`,
		"PART-001", "Test Part 1", "Acme Corp", "AC-12345")
	if err != nil {
		t.Fatalf("Failed to insert part: %v", err)
	}

	_, err = testDB.Exec(`INSERT INTO parts (ipn, description, manufacturer, mpn) VALUES (?, ?, ?, ?)`,
		"PART-002", "Test Part 2", "Beta Inc", "BI-67890")
	if err != nil {
		t.Fatalf("Failed to insert part: %v", err)
	}

	// Run migration
	if err := migrateExistingManufacturers(testDB); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify manufacturers were created
	var mfrCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM manufacturers`).Scan(&mfrCount)
	if err != nil {
		t.Fatalf("Failed to count manufacturers: %v", err)
	}
	if mfrCount != 2 {
		t.Errorf("Expected 2 manufacturers, got %d", mfrCount)
	}

	// Verify part_manufacturers records were created
	var pmCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM part_manufacturers`).Scan(&pmCount)
	if err != nil {
		t.Fatalf("Failed to count part_manufacturers: %v", err)
	}
	if pmCount != 2 {
		t.Errorf("Expected 2 part_manufacturers records, got %d", pmCount)
	}

	// Verify all part_manufacturers have manufacturer_id populated
	var nullCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE manufacturer_id IS NULL`).Scan(&nullCount)
	if err != nil {
		t.Fatalf("Failed to count NULL manufacturer_ids: %v", err)
	}
	if nullCount > 0 {
		t.Errorf("Expected all manufacturer_ids to be populated, found %d NULL values", nullCount)
	}

	// Verify is_primary is set correctly
	var primaryCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM part_manufacturers WHERE is_primary = 1`).Scan(&primaryCount)
	if err != nil {
		t.Fatalf("Failed to count primary manufacturers: %v", err)
	}
	if primaryCount != 2 {
		t.Errorf("Expected 2 primary manufacturers, got %d", primaryCount)
	}
}

// Test migration with empty database
func TestManufacturerMigrationEmpty(t *testing.T) {
	testDB := setupMigrationTestDB(t)
	defer testDB.Close()

	// Create new schema (no data)
	_, err := testDB.Exec(`
		CREATE TABLE manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			approved INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create manufacturers table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			description TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE part_manufacturers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_id TEXT NOT NULL,
			manufacturer_id INTEGER,
			mpn TEXT NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 0,
			approved INTEGER NOT NULL DEFAULT 1,
			FOREIGN KEY (part_id) REFERENCES parts(ipn) ON DELETE CASCADE,
			FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id) ON DELETE RESTRICT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_manufacturers table: %v", err)
	}

	// Run migration on empty database (should not fail)
	if err := migrateExistingManufacturers(testDB); err != nil {
		t.Errorf("Migration should not fail on empty database: %v", err)
	}

	// Verify no data was created
	var mfrCount, pmCount int
	testDB.QueryRow(`SELECT COUNT(*) FROM manufacturers`).Scan(&mfrCount)
	testDB.QueryRow(`SELECT COUNT(*) FROM part_manufacturers`).Scan(&pmCount)

	if mfrCount != 0 {
		t.Errorf("Expected 0 manufacturers, got %d", mfrCount)
	}
	if pmCount != 0 {
		t.Errorf("Expected 0 part_manufacturers, got %d", pmCount)
	}
}
