package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListPOs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,COALESCE(vendor_id,''),status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []PurchaseOrder
	for rows.Next() {
		var p PurchaseOrder
		var ra sql.NullString
		rows.Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
		p.ReceivedAt = sp(ra)
		items = append(items, p)
	}
	if items == nil { items = []PurchaseOrder{} }
	jsonResp(w, items)
}

func handleGetPO(w http.ResponseWriter, r *http.Request, id string) {
	var p PurchaseOrder
	var ra sql.NullString
	err := db.QueryRow("SELECT id,COALESCE(vendor_id,''),status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders WHERE id=?", id).
		Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
	if err != nil { jsonErr(w, "not found", 404); return }
	p.ReceivedAt = sp(ra)

	// Load lines
	rows, _ := db.Query("SELECT id,po_id,ipn,COALESCE(mpn,''),COALESCE(manufacturer,''),qty_ordered,qty_received,COALESCE(unit_price,0),COALESCE(notes,'') FROM po_lines WHERE po_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l POLine
			rows.Scan(&l.ID, &l.POID, &l.IPN, &l.MPN, &l.Manufacturer, &l.QtyOrdered, &l.QtyReceived, &l.UnitPrice, &l.Notes)
			p.Lines = append(p.Lines, l)
		}
	}
	if p.Lines == nil { p.Lines = []POLine{} }
	jsonResp(w, p)
}

func handleCreatePO(w http.ResponseWriter, r *http.Request) {
	var p PurchaseOrder
	if err := decodeBody(r, &p); err != nil { jsonErr(w, "invalid body", 400); return }
	p.ID = nextID("PO", "purchase_orders", 4)
	if p.Status == "" { p.Status = "draft" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO purchase_orders (id,vendor_id,status,notes,created_at,expected_date) VALUES (?,?,?,?,?,?)",
		p.ID, p.VendorID, p.Status, p.Notes, now, p.ExpectedDate)
	if err != nil { jsonErr(w, err.Error(), 500); return }

	for _, l := range p.Lines {
		db.Exec("INSERT INTO po_lines (po_id,ipn,mpn,manufacturer,qty_ordered,unit_price,notes) VALUES (?,?,?,?,?,?,?)",
			p.ID, l.IPN, l.MPN, l.Manufacturer, l.QtyOrdered, l.UnitPrice, l.Notes)
	}
	p.CreatedAt = now
	jsonResp(w, p)
}

func handleUpdatePO(w http.ResponseWriter, r *http.Request, id string) {
	var p PurchaseOrder
	if err := decodeBody(r, &p); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE purchase_orders SET vendor_id=?,status=?,notes=?,expected_date=? WHERE id=?",
		p.VendorID, p.Status, p.Notes, p.ExpectedDate, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetPO(w, r, id)
}

func handleReceivePO(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Lines []struct {
			ID  int     `json:"id"`
			Qty float64 `json:"qty"`
		} `json:"lines"`
	}
	if err := decodeBody(r, &body); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range body.Lines {
		db.Exec("UPDATE po_lines SET qty_received=qty_received+? WHERE id=?", l.Qty, l.ID)
		// Also update inventory
		var ipn string
		db.QueryRow("SELECT ipn FROM po_lines WHERE id=?", l.ID).Scan(&ipn)
		if ipn != "" {
			db.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", ipn)
			db.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", l.Qty, now, ipn)
			db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,created_at) VALUES (?,?,?,?,?)", ipn, "receive", l.Qty, id, now)
		}
	}
	// Check if all received
	var totalOrdered, totalReceived float64
	db.QueryRow("SELECT COALESCE(SUM(qty_ordered),0),COALESCE(SUM(qty_received),0) FROM po_lines WHERE po_id=?", id).Scan(&totalOrdered, &totalReceived)
	if totalReceived >= totalOrdered {
		db.Exec("UPDATE purchase_orders SET status='received',received_at=? WHERE id=?", now, id)
	} else {
		db.Exec("UPDATE purchase_orders SET status='partial' WHERE id=?", id)
	}
	handleGetPO(w, r, id)
}
