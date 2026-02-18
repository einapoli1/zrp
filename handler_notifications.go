package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Notification struct {
	ID        int     `json:"id"`
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Title     string  `json:"title"`
	Message   *string `json:"message"`
	RecordID  *string `json:"record_id"`
	Module    *string `json:"module"`
	ReadAt    *string `json:"read_at"`
	CreatedAt string  `json:"created_at"`
}

func handleListNotifications(w http.ResponseWriter, r *http.Request) {
	unread := r.URL.Query().Get("unread")
	q := `SELECT id, type, severity, title, message, record_id, module, read_at, created_at FROM notifications`
	if unread == "true" {
		q += ` WHERE read_at IS NULL`
	}
	q += ` ORDER BY created_at DESC LIMIT 50`

	rows, err := db.Query(q)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var notifs []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Severity, &n.Title, &n.Message, &n.RecordID, &n.Module, &n.ReadAt, &n.CreatedAt); err != nil {
			continue
		}
		notifs = append(notifs, n)
	}
	if notifs == nil {
		notifs = []Notification{}
	}
	jsonResp(w, notifs)
}

func handleMarkNotificationRead(w http.ResponseWriter, r *http.Request, id string) {
	_, err := db.Exec("UPDATE notifications SET read_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "read"})
}

type pendingNotif struct {
	ntype, severity, title string
	message, recordID, module *string
}

func generateNotifications() {
	log.Println("Generating notifications...")
	var pending []pendingNotif

	// Low stock
	func() {
		rows, err := db.Query(`SELECT ipn, qty_on_hand, reorder_point FROM inventory WHERE reorder_point > 0 AND qty_on_hand < reorder_point`)
		if err != nil { return }
		defer rows.Close()
		for rows.Next() {
			var ipn string; var qty, rp float64
			rows.Scan(&ipn, &qty, &rp)
			pending = append(pending, pendingNotif{"low_stock", "warning", "Low Stock: " + ipn,
				stringPtr(fmt.Sprintf("%.0f on hand, reorder point %.0f", qty, rp)), stringPtr(ipn), stringPtr("inventory")})
		}
	}()

	// Overdue work orders
	func() {
		rows, err := db.Query(`SELECT id, assembly_ipn FROM work_orders WHERE status = 'in_progress' AND started_at < datetime('now', '-7 days')`)
		if err != nil { return }
		defer rows.Close()
		for rows.Next() {
			var id, ipn string
			rows.Scan(&id, &ipn)
			pending = append(pending, pendingNotif{"overdue_wo", "warning", "Overdue WO: " + id,
				stringPtr("In progress for >7 days: " + ipn), stringPtr(id), stringPtr("workorders")})
		}
	}()

	// Open NCRs > 14 days
	func() {
		rows, err := db.Query(`SELECT id, title FROM ncrs WHERE status = 'open' AND created_at < datetime('now', '-14 days')`)
		if err != nil { return }
		defer rows.Close()
		for rows.Next() {
			var id, title string
			rows.Scan(&id, &title)
			t := title
			pending = append(pending, pendingNotif{"open_ncr", "error", "Open NCR >14d: " + id,
				&t, stringPtr(id), stringPtr("ncr")})
		}
	}()

	// New RMAs in last hour
	func() {
		rows, err := db.Query(`SELECT id, serial_number, customer FROM rmas WHERE created_at > datetime('now', '-1 hour')`)
		if err != nil { return }
		defer rows.Close()
		for rows.Next() {
			var id, sn string; var cust *string
			rows.Scan(&id, &sn, &cust)
			msg := "SN: " + sn
			if cust != nil { msg += " — " + *cust }
			pending = append(pending, pendingNotif{"new_rma", "info", "New RMA: " + id,
				&msg, stringPtr(id), stringPtr("rma")})
		}
	}()

	// Now insert all collected notifications
	for _, p := range pending {
		createNotificationIfNew(p.ntype, p.severity, p.title, p.message, p.recordID, p.module)
	}
	log.Printf("Notification check complete: %d candidates", len(pending))
}

func createNotificationIfNew(ntype, severity, title string, message, recordID, module *string) {
	// Dedup: don't create if same type+record exists within 24h
	var count int
	if recordID != nil {
		db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type = ? AND record_id = ? AND created_at > datetime('now', '-24 hours')`,
			ntype, *recordID).Scan(&count)
	} else {
		db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type = ? AND title = ? AND created_at > datetime('now', '-24 hours')`,
			ntype, title).Scan(&count)
	}
	if count > 0 {
		return
	}
	_, err := db.Exec(`INSERT INTO notifications (type, severity, title, message, record_id, module) VALUES (?, ?, ?, ?, ?, ?)`,
		ntype, severity, title, message, recordID, module)
	if err != nil {
		log.Println("Failed to insert notification:", err)
	}
}

func stringPtr(s string) *string { return &s }
