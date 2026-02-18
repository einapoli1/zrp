package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListQuotes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []Quote
	for rows.Next() {
		var q Quote
		var aa sql.NullString
		rows.Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
		q.AcceptedAt = sp(aa)
		items = append(items, q)
	}
	if items == nil { items = []Quote{} }
	jsonResp(w, items)
}

func handleGetQuote(w http.ResponseWriter, r *http.Request, id string) {
	var q Quote
	var aa sql.NullString
	err := db.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", id).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil { jsonErr(w, "not found", 404); return }
	q.AcceptedAt = sp(aa)

	rows, _ := db.Query("SELECT id,quote_id,ipn,COALESCE(description,''),qty,COALESCE(unit_price,0),COALESCE(notes,'') FROM quote_lines WHERE quote_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l QuoteLine
			rows.Scan(&l.ID, &l.QuoteID, &l.IPN, &l.Description, &l.Qty, &l.UnitPrice, &l.Notes)
			q.Lines = append(q.Lines, l)
		}
	}
	if q.Lines == nil { q.Lines = []QuoteLine{} }
	jsonResp(w, q)
}

func handleCreateQuote(w http.ResponseWriter, r *http.Request) {
	var q Quote
	if err := decodeBody(r, &q); err != nil { jsonErr(w, "invalid body", 400); return }
	q.ID = nextID("Q", "quotes", 3)
	if q.Status == "" { q.Status = "draft" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO quotes (id,customer,status,notes,created_at,valid_until) VALUES (?,?,?,?,?,?)",
		q.ID, q.Customer, q.Status, q.Notes, now, q.ValidUntil)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	for _, l := range q.Lines {
		db.Exec("INSERT INTO quote_lines (quote_id,ipn,description,qty,unit_price,notes) VALUES (?,?,?,?,?,?)",
			q.ID, l.IPN, l.Description, l.Qty, l.UnitPrice, l.Notes)
	}
	q.CreatedAt = now
	jsonResp(w, q)
}

func handleUpdateQuote(w http.ResponseWriter, r *http.Request, id string) {
	var q Quote
	if err := decodeBody(r, &q); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE quotes SET customer=?,status=?,notes=?,valid_until=? WHERE id=?",
		q.Customer, q.Status, q.Notes, q.ValidUntil, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetQuote(w, r, id)
}

func handleQuoteCost(w http.ResponseWriter, r *http.Request, id string) {
	rows, err := db.Query("SELECT ipn,description,qty,COALESCE(unit_price,0) FROM quote_lines WHERE quote_id=?", id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	type CostLine struct {
		IPN         string  `json:"ipn"`
		Description string  `json:"description"`
		Qty         int     `json:"qty"`
		UnitPrice   float64 `json:"unit_price"`
		LineTotal   float64 `json:"line_total"`
	}
	var lines []CostLine
	total := 0.0
	for rows.Next() {
		var l CostLine
		rows.Scan(&l.IPN, &l.Description, &l.Qty, &l.UnitPrice)
		l.LineTotal = float64(l.Qty) * l.UnitPrice
		total += l.LineTotal
		lines = append(lines, l)
	}
	if lines == nil { lines = []CostLine{} }
	jsonResp(w, map[string]interface{}{"lines": lines, "total": total})
}
