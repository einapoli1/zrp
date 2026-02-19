package main

import (
	"database/sql"
	"os"
	"testing"
)

// freshTestDB creates an in-memory database with all migrations applied.
func freshTestDB(t *testing.T) func() {
	t.Helper()
	// Use a temp file so FOREIGN KEY pragmas work reliably
	f, err := os.CreateTemp("", "zrp-integrity-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := initDB(f.Name()); err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}
	return func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func TestForeignKeysEnabled(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatal(err)
	}
	if fkEnabled != 1 {
		t.Fatal("foreign_keys pragma is not enabled")
	}
}

func TestFK_PORequiresVendor(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Inserting a PO with a non-existent vendor should fail
	_, err := db.Exec("INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-TEST-001', 'NONEXISTENT', 'draft')")
	if err == nil {
		t.Fatal("expected FK violation inserting PO with non-existent vendor")
	}
}

func TestFK_POLineRequiresPO(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES ('NONEXISTENT', 'PART-1', 10)")
	if err == nil {
		t.Fatal("expected FK violation inserting PO line with non-existent PO")
	}
}

func TestFK_CascadeDeletePOLines(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Create vendor, PO, and PO line
	db.Exec("INSERT INTO vendors (id, name) VALUES ('V-TEST', 'Test Vendor')")
	db.Exec("INSERT INTO purchase_orders (id, vendor_id) VALUES ('PO-TEST', 'V-TEST')")
	db.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES ('PO-TEST', 'PART-1', 100)")

	// Delete PO should cascade to lines
	if _, err := db.Exec("DELETE FROM purchase_orders WHERE id = 'PO-TEST'"); err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id = 'PO-TEST'").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 PO lines after cascade delete, got %d", count)
	}
}

func TestFK_RestrictVendorDeleteWithPO(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO vendors (id, name) VALUES ('V-TEST', 'Test Vendor')")
	db.Exec("INSERT INTO purchase_orders (id, vendor_id) VALUES ('PO-TEST', 'V-TEST')")

	// Should fail: vendor has POs referencing it
	_, err := db.Exec("DELETE FROM vendors WHERE id = 'V-TEST'")
	if err == nil {
		t.Fatal("expected FK RESTRICT violation deleting vendor with POs")
	}
}

func TestFK_QuoteLineCascade(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO quotes (id, customer) VALUES ('Q-TEST', 'Acme')")
	db.Exec("INSERT INTO quote_lines (quote_id, ipn, qty) VALUES ('Q-TEST', 'PART-1', 5)")

	if _, err := db.Exec("DELETE FROM quotes WHERE id = 'Q-TEST'"); err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM quote_lines WHERE quote_id = 'Q-TEST'").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 quote lines after cascade, got %d", count)
	}
}

func TestFK_WOSerialRequiresWO(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO wo_serials (wo_id, serial_number) VALUES ('NONEXISTENT', 'SN-1')")
	if err == nil {
		t.Fatal("expected FK violation for wo_serials with non-existent WO")
	}
}

func TestFK_SessionRequiresUser(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES ('tok', 99999, '2099-01-01')")
	if err == nil {
		t.Fatal("expected FK violation for session with non-existent user")
	}
}

func TestFK_ShipmentLineCascade(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO shipments (id) VALUES ('SHIP-TEST')")
	db.Exec("INSERT INTO shipment_lines (shipment_id, ipn, qty) VALUES ('SHIP-TEST', 'P1', 1)")

	db.Exec("DELETE FROM shipments WHERE id = 'SHIP-TEST'")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM shipment_lines WHERE shipment_id = 'SHIP-TEST'").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 shipment lines after cascade, got %d", count)
	}
}

func TestFK_EcoRevisionRequiresEco(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO eco_revisions (eco_id, revision) VALUES ('NONEXISTENT', 'A')")
	if err == nil {
		t.Fatal("expected FK violation for eco_revision with non-existent ECO")
	}
}

func TestFK_DocumentVersionRequiresDocument(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO document_versions (document_id, revision) VALUES ('NONEXISTENT', 'A')")
	if err == nil {
		t.Fatal("expected FK violation for document_version with non-existent document")
	}
}

func TestFK_RFQVendorRequiresVendor(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO rfqs (id, title) VALUES ('RFQ-TEST', 'Test RFQ')")
	_, err := db.Exec("INSERT INTO rfq_vendors (rfq_id, vendor_id) VALUES ('RFQ-TEST', 'NONEXISTENT')")
	if err == nil {
		t.Fatal("expected FK violation for rfq_vendor with non-existent vendor")
	}
}

func TestFK_CampaignDeviceRequiresCampaign(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO campaign_devices (campaign_id, serial_number) VALUES ('NONEXISTENT', 'SN-1')")
	if err == nil {
		t.Fatal("expected FK violation for campaign_device with non-existent campaign")
	}
}

func TestFK_ReceivingInspectionRequiresPO(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received) VALUES ('NONEXISTENT', 1, 'P1', 10)")
	if err == nil {
		t.Fatal("expected FK violation for receiving_inspection with non-existent PO")
	}
}

// CHECK constraint tests

func TestCheck_ECOInvalidStatus(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO ecos (id, title, status) VALUES ('ECO-TEST', 'Test', 'bogus')")
	if err == nil {
		t.Fatal("expected CHECK violation for invalid ECO status")
	}
}

func TestCheck_WorkOrderInvalidPriority(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO work_orders (id, assembly_ipn, qty, priority) VALUES ('WO-TEST', 'P1', 1, 'super_ultra')")
	if err == nil {
		t.Fatal("expected CHECK violation for invalid WO priority")
	}
}

func TestCheck_NegativeInventory(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO inventory (ipn, qty_on_hand) VALUES ('TEST-PART', -5)")
	if err == nil {
		t.Fatal("expected CHECK violation for negative inventory qty")
	}
}

func TestCheck_NegativeUnitPrice(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO vendors (id, name) VALUES ('V-TEST', 'Test')")
	db.Exec("INSERT INTO purchase_orders (id, vendor_id) VALUES ('PO-TEST', 'V-TEST')")
	_, err := db.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES ('PO-TEST', 'P1', 10, -1.50)")
	if err == nil {
		t.Fatal("expected CHECK violation for negative unit price")
	}
}

func TestCheck_ZeroQtyOrdered(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO vendors (id, name) VALUES ('V-TEST', 'Test')")
	db.Exec("INSERT INTO purchase_orders (id, vendor_id) VALUES ('PO-TEST', 'V-TEST')")
	_, err := db.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES ('PO-TEST', 'P1', 0)")
	if err == nil {
		t.Fatal("expected CHECK violation for zero qty_ordered")
	}
}

func TestCheck_TestRecordInvalidResult(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO test_records (serial_number, ipn, result) VALUES ('SN-1', 'P1', 'maybe')")
	if err == nil {
		t.Fatal("expected CHECK violation for invalid test result")
	}
}

func TestCheck_VendorNegativeLeadTime(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO vendors (id, name, lead_time_days) VALUES ('V-TEST', 'Test', -3)")
	if err == nil {
		t.Fatal("expected CHECK violation for negative lead_time_days")
	}
}

func TestCheck_NCRInvalidSeverity(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO ncrs (id, title, severity) VALUES ('NCR-TEST', 'Test', 'apocalyptic')")
	if err == nil {
		t.Fatal("expected CHECK violation for invalid NCR severity")
	}
}

func TestCheck_DeviceInvalidStatus(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	_, err := db.Exec("INSERT INTO devices (serial_number, ipn, status) VALUES ('SN-X', 'P1', 'exploded')")
	if err == nil {
		t.Fatal("expected CHECK violation for invalid device status")
	}
}

// UNIQUE constraint tests

func TestUnique_WOSerialNumber(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO work_orders (id, assembly_ipn, qty) VALUES ('WO-TEST', 'P1', 1)")
	db.Exec("INSERT INTO wo_serials (wo_id, serial_number) VALUES ('WO-TEST', 'SN-UNIQUE')")

	_, err := db.Exec("INSERT INTO wo_serials (wo_id, serial_number) VALUES ('WO-TEST', 'SN-UNIQUE')")
	if err == nil {
		t.Fatal("expected UNIQUE violation for duplicate serial number")
	}
}

func TestUnique_MarketPricing(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO market_pricing (part_ipn, mpn, distributor, fetched_at) VALUES ('P1', 'MPN1', 'DigiKey', '2026-01-01')")
	_, err := db.Exec("INSERT INTO market_pricing (part_ipn, mpn, distributor, fetched_at) VALUES ('P1', 'MPN1', 'DigiKey', '2026-01-02')")
	if err == nil {
		t.Fatal("expected UNIQUE violation for duplicate market_pricing (part_ipn, distributor)")
	}
}

// Index existence tests

func TestIndexesExist(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	expectedIndexes := []string{
		"idx_ecos_status",
		"idx_purchase_orders_vendor_id",
		"idx_po_lines_po_id",
		"idx_work_orders_status",
		"idx_test_records_serial_number",
		"idx_ncrs_status",
		"idx_devices_ipn",
		"idx_audit_log_created_at",
		"idx_change_history_table_record",
		"idx_notifications_read_at",
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	existing := map[string]bool{}
	for rows.Next() {
		var name sql.NullString
		rows.Scan(&name)
		if name.Valid {
			existing[name.String] = true
		}
	}

	for _, idx := range expectedIndexes {
		if !existing[idx] {
			t.Errorf("missing index: %s", idx)
		}
	}
}

// Valid data should work fine

func TestValidDataInserts(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	// Full valid chain: vendor → PO → PO lines → receiving inspection
	mustExec(t, "INSERT INTO vendors (id, name, status) VALUES ('V-1', 'Vendor', 'active')")
	mustExec(t, "INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO-1', 'V-1', 'draft')")
	mustExec(t, "INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES ('PO-1', 'P1', 100, 0.50)")

	// Work order chain
	mustExec(t, "INSERT INTO work_orders (id, assembly_ipn, qty, status, priority) VALUES ('WO-1', 'ASM-1', 10, 'open', 'high')")
	mustExec(t, "INSERT INTO wo_serials (wo_id, serial_number, status) VALUES ('WO-1', 'SN-001', 'building')")

	// Quote chain
	mustExec(t, "INSERT INTO quotes (id, customer, status) VALUES ('Q-1', 'Acme', 'draft')")
	mustExec(t, "INSERT INTO quote_lines (quote_id, ipn, qty, unit_price) VALUES ('Q-1', 'P1', 5, 10.00)")

	// Inventory with valid values
	mustExec(t, "INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES ('P1', 100, 10)")

	// ECO chain
	mustExec(t, "INSERT INTO ecos (id, title, status) VALUES ('ECO-1', 'Test ECO', 'draft')")
	mustExec(t, "INSERT INTO eco_revisions (eco_id, revision) VALUES ('ECO-1', 'A')")

	// Document chain
	mustExec(t, "INSERT INTO documents (id, title) VALUES ('DOC-1', 'Test Doc')")
	mustExec(t, "INSERT INTO document_versions (document_id, revision) VALUES ('DOC-1', 'A')")
}

func mustExec(t *testing.T, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("mustExec failed: %s\nQuery: %s", err, query)
	}
}
