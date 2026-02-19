package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// UndoLogEntry represents a stored undo action
type UndoLogEntry struct {
	ID           int    `json:"id"`
	UserID       string `json:"user_id"`
	Action       string `json:"action"` // delete, update, status_change
	EntityType   string `json:"entity_type"`
	EntityID     string `json:"entity_id"`
	PreviousData string `json:"previous_data"` // JSON snapshot
	CreatedAt    string `json:"created_at"`
	ExpiresAt    string `json:"expires_at"`
}

// snapshotEntity captures the current state of an entity before destructive action.
// Returns the JSON snapshot or error.
func snapshotEntity(entityType, entityID string) (string, error) {
	var data interface{}
	var err error

	switch entityType {
	case "eco":
		data, err = getECOSnapshot(entityID)
	case "workorder":
		data, err = getWorkOrderSnapshot(entityID)
	case "ncr":
		data, err = getNCRSnapshot(entityID)
	case "device":
		data, err = getDeviceSnapshot(entityID)
	case "inventory":
		data, err = getInventorySnapshot(entityID)
	case "rma":
		data, err = getRMASnapshot(entityID)
	case "vendor":
		data, err = getVendorSnapshot(entityID)
	case "quote":
		data, err = getQuoteSnapshot(entityID)
	case "po":
		data, err = getPOSnapshot(entityID)
	default:
		return "", fmt.Errorf("unsupported entity type: %s", entityType)
	}

	if err != nil {
		return "", err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// createUndoEntry snapshots an entity and inserts into undo_log. Returns the entry ID.
func createUndoEntry(username, action, entityType, entityID string) (int64, error) {
	snapshot, err := snapshotEntity(entityType, entityID)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	res, err := db.Exec(
		`INSERT INTO undo_log (user_id, action, entity_type, entity_id, previous_data, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		username, action, entityType, entityID, snapshot,
		now.Format("2006-01-02 15:04:05"),
		expires.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// performUndo restores an entity from a snapshot
func performUndo(entry UndoLogEntry) error {
	switch entry.EntityType {
	case "eco":
		return restoreECO(entry.PreviousData)
	case "workorder":
		return restoreWorkOrder(entry.PreviousData)
	case "ncr":
		return restoreNCR(entry.PreviousData)
	case "device":
		return restoreDevice(entry.PreviousData)
	case "inventory":
		return restoreInventory(entry.PreviousData)
	case "rma":
		return restoreRMA(entry.PreviousData)
	case "vendor":
		return restoreVendor(entry.PreviousData)
	case "quote":
		return restoreQuote(entry.PreviousData)
	case "po":
		return restorePO(entry.PreviousData)
	default:
		return fmt.Errorf("unsupported entity type: %s", entry.EntityType)
	}
}

// --- Snapshot functions ---

func getECOSnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM ecos WHERE id=?", id)
}

func getWorkOrderSnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM work_orders WHERE id=?", id)
}

func getNCRSnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM ncrs WHERE id=?", id)
}

func getDeviceSnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM devices WHERE serial_number=?", id)
}

func getInventorySnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM inventory WHERE ipn=?", id)
}

func getRMASnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM rmas WHERE id=?", id)
}

func getVendorSnapshot(id string) (map[string]interface{}, error) {
	return getRowAsMap("SELECT * FROM vendors WHERE id=?", id)
}

func getQuoteSnapshot(id string) (map[string]interface{}, error) {
	row, err := getRowAsMap("SELECT * FROM quotes WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	// Also snapshot quote lines
	lines, err := getRowsAsMapSlice("SELECT * FROM quote_lines WHERE quote_id=?", id)
	if err == nil {
		row["_lines"] = lines
	}
	return row, nil
}

func getPOSnapshot(id string) (map[string]interface{}, error) {
	row, err := getRowAsMap("SELECT * FROM purchase_orders WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	// Also snapshot PO lines
	lines, err := getRowsAsMapSlice("SELECT * FROM po_lines WHERE po_id=?", id)
	if err == nil {
		row["_lines"] = lines
	}
	return row, nil
}

// getRowAsMap runs a query expecting one row and returns it as a map
func getRowAsMap(query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, fmt.Errorf("entity not found")
	}

	values := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range cols {
		v := values[i]
		if b, ok := v.([]byte); ok {
			result[col] = string(b)
		} else {
			result[col] = v
		}
	}
	return result, nil
}

// getRowsAsMapSlice runs a query and returns all rows as a slice of maps
func getRowsAsMapSlice(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		results = append(results, row)
	}
	return results, nil
}

// --- Restore functions ---

func restoreECO(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO ecos (id, title, description, status, priority, affected_ipns, created_by, created_at, updated_at, approved_at, approved_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["title"], m["description"], m["status"], m["priority"],
		m["affected_ipns"], m["created_by"], m["created_at"], m["updated_at"],
		m["approved_at"], m["approved_by"],
	)
	return err
}

func restoreWorkOrder(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO work_orders (id, assembly_ipn, qty, status, priority, notes, created_at, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["assembly_ipn"], m["qty"], m["status"], m["priority"],
		m["notes"], m["created_at"], m["started_at"], m["completed_at"],
	)
	return err
}

func restoreNCR(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO ncrs (id, title, description, ipn, serial_number, defect_type, severity, status, root_cause, corrective_action, created_at, resolved_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["title"], m["description"], m["ipn"], m["serial_number"],
		m["defect_type"], m["severity"], m["status"], m["root_cause"], m["corrective_action"],
		m["created_at"], m["resolved_at"],
	)
	return err
}

func restoreDevice(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO devices (serial_number, ipn, firmware_version, customer, location, status, install_date, last_seen, notes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["serial_number"], m["ipn"], m["firmware_version"], m["customer"],
		m["location"], m["status"], m["install_date"], m["last_seen"], m["notes"], m["created_at"],
	)
	return err
}

func restoreInventory(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO inventory (ipn, qty_on_hand, qty_reserved, location, reorder_point, reorder_qty, description, mpn, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["ipn"], m["qty_on_hand"], m["qty_reserved"], m["location"],
		m["reorder_point"], m["reorder_qty"], m["description"], m["mpn"], m["updated_at"],
	)
	return err
}

func restoreRMA(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO rmas (id, serial_number, customer, reason, status, defect_description, resolution, created_at, received_at, resolved_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["serial_number"], m["customer"], m["reason"], m["status"],
		m["defect_description"], m["resolution"], m["created_at"], m["received_at"], m["resolved_at"],
	)
	return err
}

func restoreVendor(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO vendors (id, name, website, contact_name, contact_email, contact_phone, notes, status, lead_time_days, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["name"], m["website"], m["contact_name"], m["contact_email"],
		m["contact_phone"], m["notes"], m["status"], m["lead_time_days"], m["created_at"],
	)
	return err
}

func restoreQuote(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO quotes (id, customer, status, notes, created_at, valid_until, accepted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["customer"], m["status"], m["notes"],
		m["created_at"], m["valid_until"], m["accepted_at"],
	)
	if err != nil {
		return err
	}
	// Restore lines if present
	if linesRaw, ok := m["_lines"]; ok {
		if lines, ok := linesRaw.([]interface{}); ok {
			for _, lineRaw := range lines {
				if line, ok := lineRaw.(map[string]interface{}); ok {
					db.Exec(
						`INSERT OR REPLACE INTO quote_lines (id, quote_id, ipn, description, qty, unit_price, notes)
						 VALUES (?, ?, ?, ?, ?, ?, ?)`,
						line["id"], line["quote_id"], line["ipn"], line["description"],
						line["qty"], line["unit_price"], line["notes"],
					)
				}
			}
		}
	}
	return nil
}

func restorePO(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT OR REPLACE INTO purchase_orders (id, vendor_id, status, notes, created_at, expected_date, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["vendor_id"], m["status"], m["notes"],
		m["created_at"], m["expected_date"], m["received_at"],
	)
	if err != nil {
		return err
	}
	// Restore lines if present
	if linesRaw, ok := m["_lines"]; ok {
		if lines, ok := linesRaw.([]interface{}); ok {
			for _, lineRaw := range lines {
				if line, ok := lineRaw.(map[string]interface{}); ok {
					db.Exec(
						`INSERT OR REPLACE INTO po_lines (id, po_id, ipn, mpn, manufacturer, qty_ordered, qty_received, unit_price, notes)
						 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
						line["id"], line["po_id"], line["ipn"], line["mpn"], line["manufacturer"],
						line["qty_ordered"], line["qty_received"], line["unit_price"], line["notes"],
					)
				}
			}
		}
	}
	return nil
}

// --- HTTP Handlers ---

func handleListUndo(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	rows, err := db.Query(
		`SELECT id, user_id, action, entity_type, entity_id, previous_data, created_at, expires_at
		 FROM undo_log WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP
		 ORDER BY created_at DESC LIMIT ?`,
		username, limit,
	)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var entries []UndoLogEntry
	for rows.Next() {
		var e UndoLogEntry
		rows.Scan(&e.ID, &e.UserID, &e.Action, &e.EntityType, &e.EntityID, &e.PreviousData, &e.CreatedAt, &e.ExpiresAt)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []UndoLogEntry{}
	}
	jsonResp(w, entries)
}

func handlePerformUndo(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonErr(w, "invalid undo id", 400)
		return
	}

	username := getUsername(r)
	var entry UndoLogEntry
	err = db.QueryRow(
		`SELECT id, user_id, action, entity_type, entity_id, previous_data, created_at, expires_at
		 FROM undo_log WHERE id = ? AND user_id = ? AND expires_at > CURRENT_TIMESTAMP`,
		id, username,
	).Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.PreviousData, &entry.CreatedAt, &entry.ExpiresAt)
	if err != nil {
		jsonErr(w, "undo entry not found or expired", 404)
		return
	}

	if err := performUndo(entry); err != nil {
		jsonErr(w, fmt.Sprintf("undo failed: %v", err), 500)
		return
	}

	// Remove the undo entry after successful restore
	db.Exec("DELETE FROM undo_log WHERE id = ?", id)

	logAudit(db, username, "undo", entry.EntityType, entry.EntityID,
		fmt.Sprintf("Undid %s on %s %s", entry.Action, entry.EntityType, entry.EntityID))

	jsonResp(w, map[string]string{
		"status":      "restored",
		"entity_type": entry.EntityType,
		"entity_id":   entry.EntityID,
	})
}

// cleanExpiredUndo removes expired undo entries. Run periodically.
func cleanExpiredUndo() {
	for {
		time.Sleep(1 * time.Hour)
		db.Exec("DELETE FROM undo_log WHERE expires_at < CURRENT_TIMESTAMP")
	}
}

// suppress unused import
var _ = strings.TrimSpace
