package main

import (
	"net/http"
	"time"
)

func handleListDocs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []Document
	for rows.Next() {
		var d Document
		rows.Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
		items = append(items, d)
	}
	if items == nil { items = []Document{} }
	jsonResp(w, items)
}

func handleGetDoc(w http.ResponseWriter, r *http.Request, id string) {
	var d Document
	err := db.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", id).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil { jsonErr(w, "not found", 404); return }
	jsonResp(w, d)
}

func handleCreateDoc(w http.ResponseWriter, r *http.Request) {
	var d Document
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	d.ID = nextID("DOC", "documents", 3)
	if d.Revision == "" { d.Revision = "A" }
	if d.Status == "" { d.Status = "draft" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO documents (id,title,category,ipn,revision,status,content,file_path,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		d.ID, d.Title, d.Category, d.IPN, d.Revision, d.Status, d.Content, d.FilePath, "engineer", now, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	d.CreatedAt = now; d.UpdatedAt = now; d.CreatedBy = "engineer"
	jsonResp(w, d)
}

func handleUpdateDoc(w http.ResponseWriter, r *http.Request, id string) {
	var d Document
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE documents SET title=?,category=?,ipn=?,revision=?,status=?,content=?,file_path=?,updated_at=? WHERE id=?",
		d.Title, d.Category, d.IPN, d.Revision, d.Status, d.Content, d.FilePath, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetDoc(w, r, id)
}

func handleApproveDoc(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE documents SET status='approved',updated_at=? WHERE id=?", now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetDoc(w, r, id)
}
