package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// migrateExistingManufacturers normalizes manufacturer data from denormalized to normalized schema.
// This migration handles the transition from part_manufacturers.manufacturer (TEXT) to manufacturer_id (INTEGER FK).
//
// Migration Steps:
// 1. Check if part_manufacturers has old 'manufacturer' TEXT column
// 2. Extract unique manufacturers and insert into manufacturers table (case-insensitive dedupe)
// 3. Add manufacturer_id column to part_manufacturers if it doesn't exist
// 4. Populate manufacturer_id by matching TEXT manufacturer names
// 5. Migrate any manufacturer/mpn data from parts table to part_manufacturers
// 6. Drop old manufacturer TEXT column (optional - can keep for rollback safety)
func migrateExistingManufacturers(database *sql.DB) error {
	log.Println("Starting manufacturer normalization migration...")

	// Step 1: Check if manufacturers table exists
	var manufacturersTableExists bool
	err := database.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM sqlite_master 
		WHERE type='table' AND name='manufacturers'
	`).Scan(&manufacturersTableExists)
	if err != nil {
		return fmt.Errorf("check manufacturers table: %w", err)
	}

	if !manufacturersTableExists {
		log.Println("Manufacturers table doesn't exist yet, skipping migration")
		return nil
	}

	// Step 2: Check if part_manufacturers table has old 'manufacturer' TEXT column
	var hasOldColumn bool
	err = database.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('part_manufacturers')
		WHERE name = 'manufacturer' AND type = 'TEXT'
	`).Scan(&hasOldColumn)
	if err != nil {
		return fmt.Errorf("check for old manufacturer column: %w", err)
	}

	if !hasOldColumn {
		log.Println("Part_manufacturers already using normalized schema, checking for data to migrate...")
	}

	// Step 3: Extract unique manufacturers from part_manufacturers if old column exists
	if hasOldColumn {
		log.Println("Found old manufacturer TEXT column, extracting unique manufacturers...")
		
		rows, err := database.Query(`
			SELECT DISTINCT TRIM(manufacturer) AS mfr_name
			FROM part_manufacturers
			WHERE manufacturer != '' AND manufacturer IS NOT NULL
			ORDER BY mfr_name
		`)
		if err != nil {
			return fmt.Errorf("query unique manufacturers: %w", err)
		}
		defer rows.Close()

		// Verify manufacturers table exists before inserting
		var mfrTableCount int
		err = database.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='manufacturers'`).Scan(&mfrTableCount)
		if err != nil {
			return fmt.Errorf("verify manufacturers table before insert: %w", err)
		}
		if mfrTableCount == 0 {
			return fmt.Errorf("manufacturers table does not exist, cannot proceed with migration")
		}

		tx, err := database.Begin()
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback()

		insertedCount := 0
		for rows.Next() {
			var mfrName string
			if err := rows.Scan(&mfrName); err != nil {
				return fmt.Errorf("scan manufacturer name: %w", err)
			}

			// Insert into manufacturers table (IGNORE duplicates due to UNIQUE constraint)
			_, err = tx.Exec(`
				INSERT OR IGNORE INTO manufacturers (name, approved, created_at, updated_at)
				VALUES (?, 1, ?, ?)
			`, mfrName, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("insert manufacturer %s: %w", mfrName, err)
			}
			insertedCount++
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit manufacturers insert: %w", err)
		}

		log.Printf("Inserted/verified %d unique manufacturers", insertedCount)
	}

	// Step 4: Check if manufacturer_id column exists in part_manufacturers
	var hasNewColumn bool
	err = database.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('part_manufacturers')
		WHERE name = 'manufacturer_id'
	`).Scan(&hasNewColumn)
	if err != nil {
		return fmt.Errorf("check for manufacturer_id column: %w", err)
	}

	// Step 5: Add manufacturer_id column if it doesn't exist
	if !hasNewColumn && hasOldColumn {
		log.Println("Adding manufacturer_id column...")
		_, err = database.Exec(`
			ALTER TABLE part_manufacturers 
			ADD COLUMN manufacturer_id INTEGER REFERENCES manufacturers(id) ON DELETE RESTRICT
		`)
		if err != nil {
			return fmt.Errorf("add manufacturer_id column: %w", err)
		}
		hasNewColumn = true // Column was just added
	}

	// Step 6: Populate manufacturer_id by matching manufacturer names (if old column exists)
	if hasOldColumn {
		log.Println("Populating manufacturer_id from manufacturer names...")
		
		// Update each row to set manufacturer_id based on manufacturer name
		_, err = database.Exec(`
			UPDATE part_manufacturers
			SET manufacturer_id = (
				SELECT id FROM manufacturers 
				WHERE LOWER(name) = LOWER(part_manufacturers.manufacturer)
				LIMIT 1
			)
			WHERE manufacturer_id IS NULL AND manufacturer != '' AND manufacturer IS NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("populate manufacturer_id: %w", err)
		}

		// Check for any rows that failed to match
		var unmatchedCount int
		err = database.QueryRow(`
			SELECT COUNT(*)
			FROM part_manufacturers
			WHERE manufacturer_id IS NULL AND manufacturer != '' AND manufacturer IS NOT NULL
		`).Scan(&unmatchedCount)
		if err == nil && unmatchedCount > 0 {
			log.Printf("WARNING: %d part_manufacturers rows could not be matched to manufacturers table", unmatchedCount)
		}
	}

	// Step 7: Migrate manufacturer/mpn from parts table to part_manufacturers + manufacturers
	// First check if parts table has manufacturer column (old schema)
	var partsHasManufacturer bool
	err = database.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('parts')
		WHERE name = 'manufacturer'
	`).Scan(&partsHasManufacturer)
	if err != nil {
		return fmt.Errorf("check parts table for manufacturer column: %w", err)
	}

	if !partsHasManufacturer {
		log.Println("Parts table doesn't have manufacturer column, skipping parts migration")
		return nil
	}

	log.Println("Migrating manufacturer/mpn data from parts table...")
	
	// First, ensure manufacturers from parts table exist in manufacturers table
	rows, err := database.Query(`
		SELECT DISTINCT TRIM(manufacturer) AS mfr_name
		FROM parts
		WHERE manufacturer != '' AND manufacturer IS NOT NULL
		AND ipn NOT IN (SELECT DISTINCT part_id FROM part_manufacturers WHERE part_id IS NOT NULL)
	`)
	if err != nil {
		return fmt.Errorf("query parts manufacturers: %w", err)
	}
	defer rows.Close()

	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin parts migration transaction: %w", err)
	}
	defer tx.Rollback()

	for rows.Next() {
		var mfrName string
		if err := rows.Scan(&mfrName); err != nil {
			return fmt.Errorf("scan parts manufacturer: %w", err)
		}

		_, err = tx.Exec(`
			INSERT OR IGNORE INTO manufacturers (name, approved, created_at, updated_at)
			VALUES (?, 1, ?, ?)
		`, mfrName, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("insert manufacturer from parts %s: %w", mfrName, err)
		}
	}
	rows.Close()

	// Now insert part_manufacturers records from parts table
	partsRows, err := tx.Query(`
		SELECT ipn, manufacturer, mpn
		FROM parts
		WHERE manufacturer != '' AND manufacturer IS NOT NULL
		AND mpn != '' AND mpn IS NOT NULL
		AND ipn NOT IN (SELECT DISTINCT part_id FROM part_manufacturers WHERE part_id IS NOT NULL)
	`)
	if err != nil {
		return fmt.Errorf("query parts for migration: %w", err)
	}
	defer partsRows.Close()

	migratedCount := 0
	for partsRows.Next() {
		var ipn, manufacturer, mpn string
		if err := partsRows.Scan(&ipn, &manufacturer, &mpn); err != nil {
			return fmt.Errorf("scan part row: %w", err)
		}

		// Get manufacturer_id
		var mfrID int
		err = tx.QueryRow(`
			SELECT id FROM manufacturers 
			WHERE LOWER(name) = LOWER(?)
			LIMIT 1
		`, manufacturer).Scan(&mfrID)
		if err != nil {
			log.Printf("WARNING: Could not find manufacturer '%s' for part %s: %v", manufacturer, ipn, err)
			continue
		}

		// Insert into part_manufacturers
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO part_manufacturers 
			(part_id, manufacturer_id, mpn, is_primary, approved, notes, created_at, updated_at)
			VALUES (?, ?, ?, 1, 1, 'Migrated from parts table', ?, ?)
		`, ipn, mfrID, mpn, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
		if err != nil {
			// Log but continue - duplicates are expected
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				log.Printf("WARNING: Failed to migrate part %s: %v", ipn, err)
			}
		} else {
			migratedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit parts migration: %w", err)
	}

	if migratedCount > 0 {
		log.Printf("Migrated %d manufacturer records from parts table", migratedCount)
	}

	// Step 8: (Optional) Drop old manufacturer column
	// We'll keep it for now for safety and rollback capability
	// Uncomment this section in a future release after confirming migration success
	/*
	if hasOldColumn {
		log.Println("Dropping old manufacturer TEXT column...")
		// SQLite doesn't support DROP COLUMN directly until 3.35+
		// Would need to recreate table - skip for now
	}
	*/

	log.Println("Manufacturer normalization migration complete!")
	return nil
}
