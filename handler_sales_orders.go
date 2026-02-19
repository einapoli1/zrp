package main

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"
)

func handleListSalesOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	customer := r.URL.Query().Get("customer")

	query := "SELECT id,COALESCE(quote_id,''),customer,status,COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM sales_orders"
	var conditions []string
	var args []interface{}

	if status != "" {
		conditions = append(conditions, "status=?")
		args = append(args, status)
	}
	if customer != "" {
		conditions = append(conditions, "customer LIKE ?")
		args = append(args, "%"+customer+"%")
	}
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []SalesOrder
	for rows.Next() {
		var o SalesOrder
		rows.Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
		items = append(items, o)
	}
	if items == nil {
		items = []SalesOrder{}
	}
	jsonResp(w, items)
}

func handleGetSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o SalesOrder
	err := db.QueryRow("SELECT id,COALESCE(quote_id,''),customer,status,COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM sales_orders WHERE id=?", id).
		Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	o.Lines = getSalesOrderLines(id)

	// Attach shipment/invoice IDs if they exist
	var shipID sql.NullString
	db.QueryRow("SELECT DISTINCT sl.shipment_id FROM shipment_lines sl WHERE sl.sales_order_id=? LIMIT 1", id).Scan(&shipID)
	if shipID.Valid {
		o.ShipmentID = &shipID.String
	}
	var invID sql.NullString
	db.QueryRow("SELECT id FROM invoices WHERE sales_order_id=? LIMIT 1", id).Scan(&invID)
	if invID.Valid {
		o.InvoiceID = &invID.String
	}

	jsonResp(w, o)
}

func getSalesOrderLines(orderID string) []SalesOrderLine {
	rows, err := db.Query("SELECT id,sales_order_id,ipn,COALESCE(description,''),qty,qty_allocated,qty_picked,qty_shipped,COALESCE(unit_price,0),COALESCE(notes,'') FROM sales_order_lines WHERE sales_order_id=?", orderID)
	if err != nil {
		return []SalesOrderLine{}
	}
	defer rows.Close()
	var lines []SalesOrderLine
	for rows.Next() {
		var l SalesOrderLine
		rows.Scan(&l.ID, &l.SalesOrderID, &l.IPN, &l.Description, &l.Qty, &l.QtyAllocated, &l.QtyPicked, &l.QtyShipped, &l.UnitPrice, &l.Notes)
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []SalesOrderLine{}
	}
	return lines
}

func handleCreateSalesOrder(w http.ResponseWriter, r *http.Request) {
	var o SalesOrder
	if err := decodeBody(r, &o); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "customer", o.Customer)
	if o.Status != "" {
		validateEnum(ve, "status", o.Status, validSalesOrderStatuses)
	}
	for i, l := range o.Lines {
		if l.Qty <= 0 {
			ve.Add(fmt.Sprintf("lines[%d].qty", i), "must be positive")
		}
		if l.UnitPrice < 0 {
			ve.Add(fmt.Sprintf("lines[%d].unit_price", i), "must be non-negative")
		}
	}
	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	o.ID = nextID("SO", "sales_orders", 4)
	if o.Status == "" {
		o.Status = "draft"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	o.CreatedBy = getUsername(r)
	_, err := db.Exec("INSERT INTO sales_orders (id,quote_id,customer,status,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)",
		o.ID, o.QuoteID, o.Customer, o.Status, o.Notes, o.CreatedBy, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	for _, l := range o.Lines {
		db.Exec("INSERT INTO sales_order_lines (sales_order_id,ipn,description,qty,unit_price,notes) VALUES (?,?,?,?,?,?)",
			o.ID, l.IPN, l.Description, l.Qty, l.UnitPrice, l.Notes)
	}
	o.CreatedAt = now
	o.UpdatedAt = now
	logAudit(db, getUsername(r), "created", "sales_order", o.ID, "Created "+o.ID+" for "+o.Customer)
	recordChangeJSON(getUsername(r), "sales_orders", o.ID, "create", nil, o)
	jsonResp(w, o)
}

func handleUpdateSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o SalesOrder
	if err := decodeBody(r, &o); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE sales_orders SET customer=?,status=?,notes=?,updated_at=? WHERE id=?",
		o.Customer, o.Status, o.Notes, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logAudit(db, getUsername(r), "updated", "sales_order", id, "Updated "+id+": status="+o.Status)
	handleGetSalesOrder(w, r, id)
}

func handleConvertQuoteToOrder(w http.ResponseWriter, r *http.Request, quoteID string) {
	// Fetch quote
	var q Quote
	var aa sql.NullString
	err := db.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", quoteID).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil {
		jsonErr(w, "quote not found", 404)
		return
	}
	if q.Status != "accepted" {
		jsonErr(w, "quote must be in 'accepted' status to convert", 400)
		return
	}

	// Check if already converted
	var existingID string
	err = db.QueryRow("SELECT id FROM sales_orders WHERE quote_id=?", quoteID).Scan(&existingID)
	if err == nil {
		jsonErr(w, fmt.Sprintf("quote already converted to order %s", existingID), 409)
		return
	}

	// Get quote lines
	rows, _ := db.Query("SELECT ipn,COALESCE(description,''),qty,COALESCE(unit_price,0),COALESCE(notes,'') FROM quote_lines WHERE quote_id=?", quoteID)
	var lines []QuoteLine
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l QuoteLine
			rows.Scan(&l.IPN, &l.Description, &l.Qty, &l.UnitPrice, &l.Notes)
			lines = append(lines, l)
		}
	}

	// Create sales order
	orderID := nextID("SO", "sales_orders", 4)
	now := time.Now().Format("2006-01-02 15:04:05")
	username := getUsername(r)
	_, err = db.Exec("INSERT INTO sales_orders (id,quote_id,customer,status,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)",
		orderID, quoteID, q.Customer, "draft", q.Notes, username, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	for _, l := range lines {
		db.Exec("INSERT INTO sales_order_lines (sales_order_id,ipn,description,qty,unit_price,notes) VALUES (?,?,?,?,?,?)",
			orderID, l.IPN, l.Description, l.Qty, l.UnitPrice, l.Notes)
	}

	logAudit(db, username, "created", "sales_order", orderID, fmt.Sprintf("Converted quote %s to order %s", quoteID, orderID))
	recordChangeJSON(username, "sales_orders", orderID, "create", nil, map[string]string{"quote_id": quoteID})
	handleGetSalesOrder(w, r, orderID)
}

func handleConfirmSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	transitionSalesOrder(w, r, id, "draft", "confirmed")
}

func handleAllocateSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	// Check inventory availability
	lines := getSalesOrderLines(id)
	for _, l := range lines {
		var qtyOnHand, qtyReserved float64
		err := db.QueryRow("SELECT COALESCE(qty_on_hand,0), COALESCE(qty_reserved,0) FROM inventory WHERE ipn=?", l.IPN).Scan(&qtyOnHand, &qtyReserved)
		if err != nil {
			jsonErr(w, fmt.Sprintf("inventory record not found for %s", l.IPN), 400)
			return
		}
		available := qtyOnHand - qtyReserved
		if available < float64(l.Qty) {
			jsonErr(w, fmt.Sprintf("insufficient inventory for %s: need %d, available %.0f", l.IPN, l.Qty, available), 400)
			return
		}
	}

	// Reserve inventory
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range lines {
		db.Exec("UPDATE inventory SET qty_reserved = qty_reserved + ?, updated_at = ? WHERE ipn=?", l.Qty, now, l.IPN)
		db.Exec("UPDATE sales_order_lines SET qty_allocated=? WHERE id=?", l.Qty, l.ID)
		db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			l.IPN, "adjust", 0, fmt.Sprintf("SO:%s", id), fmt.Sprintf("Reserved %d for %s", l.Qty, id), now)
	}

	transitionSalesOrder(w, r, id, "confirmed", "allocated")
}

func handlePickSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	lines := getSalesOrderLines(id)
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range lines {
		db.Exec("UPDATE sales_order_lines SET qty_picked=? WHERE id=?", l.Qty, l.ID)
		_ = now
	}
	transitionSalesOrder(w, r, id, "allocated", "picked")
}

func handleShipSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o SalesOrder
	err := db.QueryRow("SELECT id,COALESCE(quote_id,''),customer,status FROM sales_orders WHERE id=?", id).
		Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if o.Status != "picked" {
		jsonErr(w, "order must be in 'picked' status to ship", 400)
		return
	}

	lines := getSalesOrderLines(id)
	now := time.Now().Format("2006-01-02 15:04:05")
	username := getUsername(r)

	// Create outbound shipment
	shipID := nextID("SH", "shipments", 4)
	db.Exec("INSERT INTO shipments (id,type,status,to_address,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)",
		shipID, "outbound", "packed", o.Customer, fmt.Sprintf("Shipment for %s", id), username, now, now)

	for _, l := range lines {
		// Create shipment line
		db.Exec("INSERT INTO shipment_lines (shipment_id,ipn,qty,sales_order_id) VALUES (?,?,?,?)",
			shipID, l.IPN, l.Qty, id)
		// Reduce inventory (issue)
		db.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand - ?, qty_reserved = qty_reserved - ?, updated_at = ? WHERE ipn=?",
			l.Qty, l.Qty, now, l.IPN)
		db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			l.IPN, "issue", float64(l.Qty), fmt.Sprintf("SO:%s", id), fmt.Sprintf("Shipped %d for %s", l.Qty, id), now)
		db.Exec("UPDATE sales_order_lines SET qty_shipped=? WHERE id=?", l.Qty, l.ID)
	}

	db.Exec("UPDATE sales_orders SET status='shipped',updated_at=? WHERE id=?", now, id)
	logAudit(db, username, "shipped", "sales_order", id, fmt.Sprintf("Shipped %s via shipment %s", id, shipID))
	handleGetSalesOrder(w, r, id)
}

func handleInvoiceSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o SalesOrder
	err := db.QueryRow("SELECT id,COALESCE(quote_id,''),customer,status FROM sales_orders WHERE id=?", id).
		Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if o.Status != "shipped" {
		jsonErr(w, "order must be in 'shipped' status to invoice", 400)
		return
	}

	lines := getSalesOrderLines(id)
	var total float64
	for _, l := range lines {
		total += float64(l.Qty) * l.UnitPrice
	}
	total = math.Round(total*100) / 100

	now := time.Now().Format("2006-01-02 15:04:05")
	invID := nextID("INV", "invoices", 4)
	issueDate := time.Now().Format("2006-01-02")
	dueDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	username := getUsername(r)

	_, err = db.Exec("INSERT INTO invoices (id,invoice_number,sales_order_id,customer,status,total,created_at,issue_date,due_date) VALUES (?,?,?,?,?,?,?,?,?)",
		invID, invID, id, o.Customer, "draft", total, now, issueDate, dueDate)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	db.Exec("UPDATE sales_orders SET status='invoiced',updated_at=? WHERE id=?", now, id)
	logAudit(db, username, "invoiced", "sales_order", id, fmt.Sprintf("Created invoice %s for %s (%.2f)", invID, id, total))
	handleGetSalesOrder(w, r, id)
}

func transitionSalesOrder(w http.ResponseWriter, r *http.Request, id, fromStatus, toStatus string) {
	var currentStatus string
	err := db.QueryRow("SELECT status FROM sales_orders WHERE id=?", id).Scan(&currentStatus)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if currentStatus != fromStatus {
		jsonErr(w, fmt.Sprintf("order must be in '%s' status (currently '%s')", fromStatus, currentStatus), 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("UPDATE sales_orders SET status=?,updated_at=? WHERE id=?", toStatus, now, id)
	logAudit(db, getUsername(r), toStatus, "sales_order", id, fmt.Sprintf("Transitioned %s from %s to %s", id, fromStatus, toStatus))
	handleGetSalesOrder(w, r, id)
}
