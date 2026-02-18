package main

import (
	"net/http"
	"time"
)

func handleListDocs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT d.id, d.title, COALESCE(d.category,''), COALESCE(d.ipn,''), d.revision, d.status,
		COALESCE(d.content,''), COALESCE(d.file_path,''), d.created_by, d.created_at, d.updated_at,
		COALESCE(a.cnt, 0)
		FROM documents d
		LEFT JOIN (SELECT record_id, COUNT(*) as cnt FROM attachments WHERE module='document' GROUP BY record_id) a ON a.record_id = d.id
		ORDER BY d.created_at DESC`)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	type DocWithCount struct {
		Document
		AttachmentCount int `json:"attachment_count"`
	}
	var items []DocWithCount
	for rows.Next() {
		var d DocWithCount
		rows.Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.AttachmentCount)
		items = append(items, d)
	}
	if items == nil { items = []DocWithCount{} }
	jsonResp(w, items)
}

func handleGetDoc(w http.ResponseWriter, r *http.Request, id string) {
	var d Document
	err := db.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", id).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil { jsonErr(w, "not found", 404); return }

	// Fetch attachments
	attRows, err := db.Query("SELECT id, module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by, created_at FROM attachments WHERE module='document' AND record_id=? ORDER BY created_at DESC", id)
	var atts []Attachment
	if err == nil {
		defer attRows.Close()
		for attRows.Next() {
			var a Attachment
			attRows.Scan(&a.ID, &a.Module, &a.RecordID, &a.Filename, &a.OriginalName, &a.SizeBytes, &a.MimeType, &a.UploadedBy, &a.CreatedAt)
			atts = append(atts, a)
		}
	}
	if atts == nil { atts = []Attachment{} }

	type DocWithAttachments struct {
		Document
		Attachments []Attachment `json:"attachments"`
	}
	jsonResp(w, DocWithAttachments{Document: d, Attachments: atts})
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
	logAudit(db, getUsername(r), "created", "document", d.ID, "Created "+d.ID+": "+d.Title)
	jsonResp(w, d)
}

func handleUpdateDoc(w http.ResponseWriter, r *http.Request, id string) {
	var d Document
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE documents SET title=?,category=?,ipn=?,revision=?,status=?,content=?,file_path=?,updated_at=? WHERE id=?",
		d.Title, d.Category, d.IPN, d.Revision, d.Status, d.Content, d.FilePath, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "document", id, "Updated "+id+": "+d.Title)
	handleGetDoc(w, r, id)
}

func handleApproveDoc(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE documents SET status='approved',updated_at=? WHERE id=?", now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "approved", "document", id, "Approved document "+id)
	handleGetDoc(w, r, id)
}
