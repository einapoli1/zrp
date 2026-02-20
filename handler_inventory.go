package main

import (
	"net/http"
	"strings"
	"time"
)

func handleListInventory(w http.ResponseWriter, r *http.Request) {
	lowStock := r.URL.Query().Get("low_stock")
	query := "SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory"
	if lowStock == "true" {
		query += " WHERE qty_on_hand <= reorder_point AND reorder_point > 0"
	}
	query += " ORDER BY ipn"
	rows, err := db.Query(query)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []InventoryItem
	for rows.Next() {
		var i InventoryItem
		rows.Scan(&i.IPN, &i.QtyOnHand, &i.QtyReserved, &i.Location, &i.ReorderPoint, &i.ReorderQty, &i.Description, &i.MPN, &i.UpdatedAt)
		items = append(items, i)
	}
	if items == nil { items = []InventoryItem{} }
	jsonResp(w, items)
}

func handleGetInventory(w http.ResponseWriter, r *http.Request, ipn string) {
	var i InventoryItem
	err := db.QueryRow("SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory WHERE ipn=?", ipn).
		Scan(&i.IPN, &i.QtyOnHand, &i.QtyReserved, &i.Location, &i.ReorderPoint, &i.ReorderQty, &i.Description, &i.MPN, &i.UpdatedAt)
	if err != nil { jsonErr(w, "not found", 404); return }
	jsonResp(w, i)
}

func handleInventoryTransact(w http.ResponseWriter, r *http.Request) {
	var t InventoryTransaction
	if err := decodeBody(r, &t); err != nil { jsonErr(w, "invalid body", 400); return }

	ve := &ValidationErrors{}
	requireField(ve, "ipn", t.IPN)
	requireField(ve, "type", t.Type)
	validateEnum(ve, "type", t.Type, validInventoryTypes)
	if t.Type != "adjust" && t.Qty <= 0 { ve.Add("qty", "must be positive") }
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	now := time.Now().Format("2006-01-02 15:04:05")

	// Ensure inventory record exists, enriching with parts DB data
	var desc, mpn string
	fields, err2 := getPartByIPN(partsDir, t.IPN)
	if err2 == nil {
		for k, v := range fields {
			kl := strings.ToLower(k)
			if kl == "description" || kl == "desc" {
				desc = v
			}
			if kl == "mpn" {
				mpn = v
			}
		}
	}

	// Begin transaction to ensure atomicity
	tx, err := db.Begin()
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer tx.Rollback() // Rollback if not committed

	// Ensure inventory record exists
	_, err = tx.Exec("INSERT OR IGNORE INTO inventory (ipn, description, mpn) VALUES (?, ?, ?)", t.IPN, desc, mpn)
	if err != nil { jsonErr(w, err.Error(), 500); return }

	// Insert transaction
	_, err = tx.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
		t.IPN, t.Type, t.Qty, t.Reference, t.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }

	// Update inventory quantity
	switch t.Type {
	case "receive", "return":
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", t.Qty, now, t.IPN)
	case "issue":
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand-?,updated_at=? WHERE ipn=?", t.Qty, now, t.IPN)
	case "adjust":
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand=?,updated_at=? WHERE ipn=?", t.Qty, now, t.IPN)
	}
	if err != nil { jsonErr(w, err.Error(), 500); return }

	// Commit transaction
	if err = tx.Commit(); err != nil { jsonErr(w, err.Error(), 500); return }

	logAudit(db, getUsername(r), t.Type, "inventory", t.IPN, "Inventory "+t.Type+": "+t.IPN)
	// Capture db and ipn to avoid race with test cleanup
	currentDB := db
	ipnCopy := t.IPN
	go func() {
		if currentDB == nil || db == nil {
			return
		}
		// Check email config enabled
		var enabled int
		if err := currentDB.QueryRow("SELECT enabled FROM email_config WHERE id=1").Scan(&enabled); err != nil || enabled != 1 {
			return
		}
		// Check inventory levels
		var qtyOnHand, reorderPoint int
		if err := currentDB.QueryRow("SELECT qty_on_hand, reorder_point FROM inventory WHERE ipn=?", ipnCopy).Scan(&qtyOnHand, &reorderPoint); err != nil || reorderPoint <= 0 || qtyOnHand > reorderPoint {
			return
		}
		// Send low stock email (only if global db still valid)
		if db != nil {
			emailOnLowStock(ipnCopy)
		}
	}()
	jsonResp(w, map[string]string{"status": "ok"})
}

func handleInventoryHistory(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := db.Query("SELECT id,ipn,type,qty,COALESCE(reference,''),COALESCE(notes,''),created_at FROM inventory_transactions WHERE ipn=? ORDER BY created_at DESC", ipn)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []InventoryTransaction
	for rows.Next() {
		var t InventoryTransaction
		rows.Scan(&t.ID, &t.IPN, &t.Type, &t.Qty, &t.Reference, &t.Notes, &t.CreatedAt)
		items = append(items, t)
	}
	if items == nil { items = []InventoryTransaction{} }
	jsonResp(w, items)
}

func handleBulkDeleteInventory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IPNs []string `json:"ipns"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if len(body.IPNs) == 0 {
		jsonErr(w, "ipns required", 400)
		return
	}
	deleted := 0
	for _, ipn := range body.IPNs {
		res, err := db.Exec("DELETE FROM inventory WHERE ipn=?", ipn)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		deleted += int(n)
	}
	jsonResp(w, map[string]int{"deleted": deleted})
}
