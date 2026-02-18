package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListNCRs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),created_at,resolved_at FROM ncrs ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []NCR
	for rows.Next() {
		var n NCR
		var ra sql.NullString
		rows.Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedAt, &ra)
		n.ResolvedAt = sp(ra)
		items = append(items, n)
	}
	if items == nil { items = []NCR{} }
	jsonResp(w, items)
}

func handleGetNCR(w http.ResponseWriter, r *http.Request, id string) {
	var n NCR
	var ra sql.NullString
	err := db.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),created_at,resolved_at FROM ncrs WHERE id=?", id).
		Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedAt, &ra)
	if err != nil { jsonErr(w, "not found", 404); return }
	n.ResolvedAt = sp(ra)
	jsonResp(w, n)
}

func handleCreateNCR(w http.ResponseWriter, r *http.Request) {
	var n NCR
	if err := decodeBody(r, &n); err != nil { jsonErr(w, "invalid body", 400); return }
	n.ID = nextID("NCR", "ncrs", 3)
	if n.Status == "" { n.Status = "open" }
	if n.Severity == "" { n.Severity = "minor" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		n.ID, n.Title, n.Description, n.IPN, n.SerialNumber, n.DefectType, n.Severity, n.Status, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	n.CreatedAt = now
	logAudit(db, getUsername(r), "created", "ncr", n.ID, "Created "+n.ID+": "+n.Title)
	jsonResp(w, n)
}

func handleUpdateNCR(w http.ResponseWriter, r *http.Request, id string) {
	var n NCR
	if err := decodeBody(r, &n); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	var resolvedAt interface{}
	if n.Status == "resolved" || n.Status == "closed" {
		resolvedAt = now
	}
	_, err := db.Exec("UPDATE ncrs SET title=?,description=?,ipn=?,serial_number=?,defect_type=?,severity=?,status=?,root_cause=?,corrective_action=?,resolved_at=COALESCE(?,resolved_at) WHERE id=?",
		n.Title, n.Description, n.IPN, n.SerialNumber, n.DefectType, n.Severity, n.Status, n.RootCause, n.CorrectiveAction, resolvedAt, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "ncr", id, "Updated "+id+": "+n.Title)
	handleGetNCR(w, r, id)
}
