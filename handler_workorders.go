package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListWorkOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []WorkOrder
	for rows.Next() {
		var wo WorkOrder
		var sa, ca sql.NullString
		rows.Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
		wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
		items = append(items, wo)
	}
	if items == nil { items = []WorkOrder{} }
	jsonResp(w, items)
}

func handleGetWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	var wo WorkOrder
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders WHERE id=?", id).
		Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
	if err != nil { jsonErr(w, "not found", 404); return }
	wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
	jsonResp(w, wo)
}

func handleCreateWorkOrder(w http.ResponseWriter, r *http.Request) {
	var wo WorkOrder
	if err := decodeBody(r, &wo); err != nil { jsonErr(w, "invalid body", 400); return }
	wo.ID = nextID("WO", "work_orders", 4)
	if wo.Status == "" { wo.Status = "open" }
	if wo.Priority == "" { wo.Priority = "normal" }
	if wo.Qty == 0 { wo.Qty = 1 }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO work_orders (id,assembly_ipn,qty,status,priority,notes,created_at) VALUES (?,?,?,?,?,?,?)",
		wo.ID, wo.AssemblyIPN, wo.Qty, wo.Status, wo.Priority, wo.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	wo.CreatedAt = now
	jsonResp(w, wo)
}

func handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	var wo WorkOrder
	if err := decodeBody(r, &wo); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE work_orders SET assembly_ipn=?,qty=?,status=?,priority=?,notes=?,started_at=CASE WHEN ?='in_progress' AND started_at IS NULL THEN ? ELSE started_at END,completed_at=CASE WHEN ?='completed' THEN ? ELSE completed_at END WHERE id=?",
		wo.AssemblyIPN, wo.Qty, wo.Status, wo.Priority, wo.Notes, wo.Status, now, wo.Status, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetWorkOrder(w, r, id)
}

func handleWorkOrderBOM(w http.ResponseWriter, r *http.Request, id string) {
	// Get the assembly IPN for this WO
	var assemblyIPN string
	var qty int
	err := db.QueryRow("SELECT assembly_ipn,qty FROM work_orders WHERE id=?", id).Scan(&assemblyIPN, &qty)
	if err != nil { jsonErr(w, "not found", 404); return }

	// Return a simple BOM structure (would need gitplm BOM data for real implementation)
	type BOMLine struct {
		IPN       string  `json:"ipn"`
		Qty       float64 `json:"qty_required"`
		OnHand    float64 `json:"qty_on_hand"`
		Available bool    `json:"available"`
	}
	// Check inventory for any items we know about
	rows, _ := db.Query("SELECT ipn, qty_on_hand FROM inventory")
	var bom []BOMLine
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var bl BOMLine
			rows.Scan(&bl.IPN, &bl.OnHand)
			bl.Qty = float64(qty) // simplified
			bl.Available = bl.OnHand >= bl.Qty
			bom = append(bom, bl)
		}
	}
	if bom == nil { bom = []BOMLine{} }
	jsonResp(w, map[string]interface{}{"assembly_ipn": assemblyIPN, "wo_qty": qty, "bom": bom})
}
