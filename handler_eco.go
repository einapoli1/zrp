package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
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

	// Enrich with affected parts details
	var affectedParts []map[string]string
	var ipns []string
	// Try JSON array first, then comma-separated
	if strings.HasPrefix(strings.TrimSpace(e.AffectedIPNs), "[") {
		json.Unmarshal([]byte(e.AffectedIPNs), &ipns)
	} else if e.AffectedIPNs != "" {
		for _, s := range strings.Split(e.AffectedIPNs, ",") {
			s = strings.TrimSpace(s)
			if s != "" { ipns = append(ipns, s) }
		}
	}
	for _, ipn := range ipns {
		fields, err := getPartByIPN(partsDir, ipn)
		if err == nil {
			part := make(map[string]string)
			part["ipn"] = ipn
			for k, v := range fields {
				part[strings.ToLower(k)] = v
			}
			affectedParts = append(affectedParts, part)
		} else {
			affectedParts = append(affectedParts, map[string]string{"ipn": ipn, "error": "not found"})
		}
	}
	if affectedParts == nil { affectedParts = []map[string]string{} }

	// Build enriched response
	resp := map[string]interface{}{
		"id": e.ID, "title": e.Title, "description": e.Description,
		"status": e.Status, "priority": e.Priority, "affected_ipns": e.AffectedIPNs,
		"affected_parts": affectedParts, "created_by": e.CreatedBy,
		"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
		"approved_at": e.ApprovedAt, "approved_by": e.ApprovedBy,
	}
	jsonResp(w, resp)
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
	logAudit(db, getUsername(r), "created", "eco", e.ID, "Created "+e.ID+": "+e.Title)
	jsonResp(w, e)
}

func handleUpdateECO(w http.ResponseWriter, r *http.Request, id string) {
	var e ECO
	if err := decodeBody(r, &e); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE ecos SET title=?,description=?,status=?,priority=?,affected_ipns=?,updated_at=? WHERE id=?",
		e.Title, e.Description, e.Status, e.Priority, e.AffectedIPNs, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "eco", id, "Updated "+id+": "+e.Title)
	handleGetECO(w, r, id)
}

func handleApproveECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by='engineer',updated_at=? WHERE id=?", now, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "approved", "eco", id, "Approved "+id)
	handleGetECO(w, r, id)
}

func handleImplementECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "implemented", "eco", id, "Implemented "+id)
	handleGetECO(w, r, id)
}
