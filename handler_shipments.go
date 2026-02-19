package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

func handleListShipments(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,type,status,COALESCE(tracking_number,''),COALESCE(carrier,''),ship_date,delivery_date,COALESCE(from_address,''),COALESCE(to_address,''),COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM shipments ORDER BY created_at DESC")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []Shipment
	for rows.Next() {
		var s Shipment
		var sd, dd sql.NullString
		rows.Scan(&s.ID, &s.Type, &s.Status, &s.TrackingNumber, &s.Carrier, &sd, &dd, &s.FromAddress, &s.ToAddress, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
		s.ShipDate = sp(sd)
		s.DeliveryDate = sp(dd)
		items = append(items, s)
	}
	if items == nil {
		items = []Shipment{}
	}
	jsonResp(w, items)
}

func handleGetShipment(w http.ResponseWriter, r *http.Request, id string) {
	var s Shipment
	var sd, dd sql.NullString
	err := db.QueryRow("SELECT id,type,status,COALESCE(tracking_number,''),COALESCE(carrier,''),ship_date,delivery_date,COALESCE(from_address,''),COALESCE(to_address,''),COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM shipments WHERE id=?", id).
		Scan(&s.ID, &s.Type, &s.Status, &s.TrackingNumber, &s.Carrier, &sd, &dd, &s.FromAddress, &s.ToAddress, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	s.ShipDate = sp(sd)
	s.DeliveryDate = sp(dd)
	s.Lines = getShipmentLines(id)
	jsonResp(w, s)
}

func getShipmentLines(shipmentID string) []ShipmentLine {
	rows, err := db.Query("SELECT id,shipment_id,COALESCE(ipn,''),COALESCE(serial_number,''),qty,COALESCE(work_order_id,''),COALESCE(rma_id,'') FROM shipment_lines WHERE shipment_id=?", shipmentID)
	if err != nil {
		return []ShipmentLine{}
	}
	defer rows.Close()
	var lines []ShipmentLine
	for rows.Next() {
		var l ShipmentLine
		rows.Scan(&l.ID, &l.ShipmentID, &l.IPN, &l.SerialNumber, &l.Qty, &l.WorkOrderID, &l.RMAID)
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []ShipmentLine{}
	}
	return lines
}

func handleCreateShipment(w http.ResponseWriter, r *http.Request) {
	var s Shipment
	if err := decodeBody(r, &s); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	if s.Type != "" { validateEnum(ve, "type", s.Type, validShipmentTypes) }
	if s.Status != "" { validateEnum(ve, "status", s.Status, validShipmentStatuses) }
	for i, line := range s.Lines {
		if line.Qty <= 0 { ve.Add(fmt.Sprintf("lines[%d].qty", i), "must be positive") }
	}
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	s.ID = nextID("SHP", "shipments", 4)
	if s.Type == "" {
		s.Type = "outbound"
	}
	if s.Status == "" {
		s.Status = "draft"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	s.CreatedBy = getUsername(r)
	_, err := db.Exec("INSERT INTO shipments (id,type,status,tracking_number,carrier,from_address,to_address,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		s.ID, s.Type, s.Status, s.TrackingNumber, s.Carrier, s.FromAddress, s.ToAddress, s.Notes, s.CreatedBy, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	s.CreatedAt = now
	s.UpdatedAt = now

	// Insert lines if provided
	for _, line := range s.Lines {
		_, err := db.Exec("INSERT INTO shipment_lines (shipment_id,ipn,serial_number,qty,work_order_id,rma_id) VALUES (?,?,?,?,?,?)",
			s.ID, line.IPN, line.SerialNumber, line.Qty, line.WorkOrderID, line.RMAID)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
	}

	logAudit(db, getUsername(r), "created", "shipment", s.ID, "Created shipment "+s.ID)
	s.Lines = getShipmentLines(s.ID)
	jsonResp(w, s)
}

func handleUpdateShipment(w http.ResponseWriter, r *http.Request, id string) {
	var s Shipment
	if err := decodeBody(r, &s); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE shipments SET type=?,status=?,tracking_number=?,carrier=?,from_address=?,to_address=?,notes=?,updated_at=? WHERE id=?",
		s.Type, s.Status, s.TrackingNumber, s.Carrier, s.FromAddress, s.ToAddress, s.Notes, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Replace lines if provided
	if s.Lines != nil {
		db.Exec("DELETE FROM shipment_lines WHERE shipment_id=?", id)
		for _, line := range s.Lines {
			db.Exec("INSERT INTO shipment_lines (shipment_id,ipn,serial_number,qty,work_order_id,rma_id) VALUES (?,?,?,?,?,?)",
				id, line.IPN, line.SerialNumber, line.Qty, line.WorkOrderID, line.RMAID)
		}
	}

	logAudit(db, getUsername(r), "updated", "shipment", id, fmt.Sprintf("Updated shipment %s: status=%s", id, s.Status))
	handleGetShipment(w, r, id)
}

func handleShipShipment(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		TrackingNumber string `json:"tracking_number"`
		Carrier        string `json:"carrier"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	// Verify shipment exists and is in valid state
	var status string
	err := db.QueryRow("SELECT status FROM shipments WHERE id=?", id).Scan(&status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if status == "shipped" || status == "delivered" {
		jsonErr(w, "shipment already "+status, 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE shipments SET status='shipped',tracking_number=?,carrier=?,ship_date=?,updated_at=? WHERE id=?",
		body.TrackingNumber, body.Carrier, now, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "shipped", "shipment", id, fmt.Sprintf("Shipped %s via %s tracking %s", id, body.Carrier, body.TrackingNumber))
	handleGetShipment(w, r, id)
}

func handleDeliverShipment(w http.ResponseWriter, r *http.Request, id string) {
	// Verify shipment exists
	var status, shipType string
	err := db.QueryRow("SELECT status, type FROM shipments WHERE id=?", id).Scan(&status, &shipType)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if status == "delivered" {
		jsonErr(w, "already delivered", 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE shipments SET status='delivered',delivery_date=?,updated_at=? WHERE id=?", now, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Update inventory for inbound shipments
	if shipType == "inbound" {
		lines := getShipmentLines(id)
		for _, line := range lines {
			if line.IPN != "" && line.Qty > 0 {
				db.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ?, updated_at = ? WHERE ipn = ?", line.Qty, now, line.IPN)
				db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,'receive',?,?,?,?)",
					line.IPN, line.Qty, "SHP:"+id, "Inbound shipment delivered", now)
			}
		}
	}

	logAudit(db, getUsername(r), "delivered", "shipment", id, "Marked "+id+" as delivered")
	handleGetShipment(w, r, id)
}

func handleShipmentPackList(w http.ResponseWriter, r *http.Request, id string) {
	// Verify shipment exists
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM shipments WHERE id=?", id).Scan(&exists)
	if err != nil || exists == 0 {
		jsonErr(w, "not found", 404)
		return
	}

	lines := getShipmentLines(id)

	// Auto-create pack list record
	now := time.Now().Format("2006-01-02 15:04:05")
	res, _ := db.Exec("INSERT INTO pack_lists (shipment_id,created_at) VALUES (?,?)", id, now)
	plID, _ := res.LastInsertId()

	pl := PackList{
		ID:         int(plID),
		ShipmentID: id,
		CreatedAt:  now,
		Lines:      lines,
	}
	jsonResp(w, pl)
}
