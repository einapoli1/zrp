package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListECOs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	query := "SELECT id,title,description,status,priority,COALESCE(affected_ipns,''),created_by,created_at,updated_at,approved_at,approved_by FROM ecos"
	var args []interface{}
	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500); return
	}
	defer rows.Close()
	var items []ECO
	for rows.Next() {
		var e ECO
		var aa, ab sql.NullString
		rows.Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab)
		e.ApprovedAt = sp(aa); e.ApprovedBy = sp(ab)
		items = append(items, e)
	}
	if items == nil { items = []ECO{} }
	jsonResp(w, items)
}

func handleGetECO(w http.ResponseWriter, r *http.Request, id string) {
	var e ECO
	var aa, ab sql.NullString
	err := db.QueryRow("SELECT id,title,description,status,priority,COALESCE(affected_ipns,''),created_by,created_at,updated_at,approved_at,approved_by FROM ecos WHERE id=?", id).
		Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab)
	if err != nil { jsonErr(w, "not found", 404); return }
	e.ApprovedAt = sp(aa); e.ApprovedBy = sp(ab)
	jsonResp(w, e)
}

func handleCreateECO(w http.ResponseWriter, r *http.Request) {
	var e ECO
	if err := decodeBody(r, &e); err != nil { jsonErr(w, "invalid body", 400); return }
	e.ID = nextID("ECO", "ecos", 3)
	if e.Status == "" { e.Status = "draft" }
	if e.Priority == "" { e.Priority = "normal" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)",
		e.ID, e.Title, e.Description, e.Status, e.Priority, e.AffectedIPNs, "engineer", now, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	e.CreatedAt = now; e.UpdatedAt = now; e.CreatedBy = "engineer"
	jsonResp(w, e)
}

func handleUpdateECO(w http.ResponseWriter, r *http.Request, id string) {
	var e ECO
	if err := decodeBody(r, &e); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE ecos SET title=?,description=?,status=?,priority=?,affected_ipns=?,updated_at=? WHERE id=?",
		e.Title, e.Description, e.Status, e.Priority, e.AffectedIPNs, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetECO(w, r, id)
}

func handleApproveECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by='engineer',updated_at=? WHERE id=?", now, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetECO(w, r, id)
}

func handleImplementECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetECO(w, r, id)
}
