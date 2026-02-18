package main

import (
	"os"
	"strings"
	"testing"
)

func TestNextID(t *testing.T) {
	// Setup test DB
	os.Remove("test.db")
	defer os.Remove("test.db")
	if err := initDB("test.db"); err != nil {
		t.Fatal(err)
	}
	seedDB()

	id := nextID("ECO", "ecos", 3)
	if !strings.HasPrefix(id, "ECO-2026-") {
		t.Errorf("unexpected id format: %s", id)
	}
	// Should be 003 since we seeded 001 and 002
	if !strings.HasSuffix(id, "003") {
		t.Errorf("expected suffix 003, got: %s", id)
	}
}

func TestDBMigrations(t *testing.T) {
	os.Remove("test2.db")
	defer os.Remove("test2.db")
	if err := initDB("test2.db"); err != nil {
		t.Fatal("migration failed:", err)
	}
	// Run again - should be idempotent
	if err := runMigrations(); err != nil {
		t.Fatal("re-migration failed:", err)
	}
}

func TestSeedDB(t *testing.T) {
	os.Remove("test3.db")
	defer os.Remove("test3.db")
	if err := initDB("test3.db"); err != nil {
		t.Fatal(err)
	}
	seedDB()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 ecos, got %d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 vendors, got %d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 devices, got %d", count)
	}
}

func TestSPHelper(t *testing.T) {
	s := "hello"
	ns := ns(&s)
	if !ns.Valid || ns.String != "hello" {
		t.Error("ns failed")
	}
	result := sp(ns)
	if result == nil || *result != "hello" {
		t.Error("sp failed")
	}
	nilResult := sp(ns)
	if nilResult == nil {
		t.Error("sp nil failed")
	}
}
