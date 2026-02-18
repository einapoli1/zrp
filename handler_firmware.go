package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,name,version,category,status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []FirmwareCampaign
	for rows.Next() {
		var f FirmwareCampaign
		var sa, ca sql.NullString
		rows.Scan(&f.ID, &f.Name, &f.Version, &f.Category, &f.Status, &f.TargetFilter, &f.Notes, &f.CreatedAt, &sa, &ca)
		f.StartedAt = sp(sa); f.CompletedAt = sp(ca)
		items = append(items, f)
	}
	if items == nil { items = []FirmwareCampaign{} }
	jsonResp(w, items)
}

func handleGetCampaign(w http.ResponseWriter, r *http.Request, id string) {
	var f FirmwareCampaign
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,name,version,category,status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns WHERE id=?", id).
		Scan(&f.ID, &f.Name, &f.Version, &f.Category, &f.Status, &f.TargetFilter, &f.Notes, &f.CreatedAt, &sa, &ca)
	if err != nil { jsonErr(w, "not found", 404); return }
	f.StartedAt = sp(sa); f.CompletedAt = sp(ca)
	jsonResp(w, f)
}

func handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var f FirmwareCampaign
	if err := decodeBody(r, &f); err != nil { jsonErr(w, "invalid body", 400); return }
	f.ID = nextID("FW", "firmware_campaigns", 3)
	if f.Status == "" { f.Status = "draft" }
	if f.Category == "" { f.Category = "public" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO firmware_campaigns (id,name,version,category,status,target_filter,notes,created_at) VALUES (?,?,?,?,?,?,?,?)",
		f.ID, f.Name, f.Version, f.Category, f.Status, f.TargetFilter, f.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	f.CreatedAt = now
	jsonResp(w, f)
}

func handleUpdateCampaign(w http.ResponseWriter, r *http.Request, id string) {
	var f FirmwareCampaign
	if err := decodeBody(r, &f); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE firmware_campaigns SET name=?,version=?,category=?,status=?,target_filter=?,notes=? WHERE id=?",
		f.Name, f.Version, f.Category, f.Status, f.TargetFilter, f.Notes, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetCampaign(w, r, id)
}

func handleLaunchCampaign(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	// Get all active devices and add them to campaign
	rows, err := db.Query("SELECT serial_number FROM devices WHERE status='active'")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	count := 0
	for rows.Next() {
		var sn string
		rows.Scan(&sn)
		db.Exec("INSERT OR IGNORE INTO campaign_devices (campaign_id,serial_number,status) VALUES (?,?,?)", id, sn, "pending")
		count++
	}
	db.Exec("UPDATE firmware_campaigns SET status='active',started_at=? WHERE id=?", now, id)
	jsonResp(w, map[string]interface{}{"launched": true, "devices_added": count})
}

func handleCampaignProgress(w http.ResponseWriter, r *http.Request, id string) {
	var pending, sent, updated, failed int
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='pending'", id).Scan(&pending)
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='sent'", id).Scan(&sent)
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='updated'", id).Scan(&updated)
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='failed'", id).Scan(&failed)
	total := pending + sent + updated + failed
	jsonResp(w, map[string]int{"total": total, "pending": pending, "sent": sent, "updated": updated, "failed": failed})
}
