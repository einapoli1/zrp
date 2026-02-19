package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// NotificationTypeInfo describes an available notification type
type NotificationTypeInfo struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Icon             string   `json:"icon"`
	HasThreshold     bool     `json:"has_threshold"`
	ThresholdLabel   *string  `json:"threshold_label,omitempty"`
	ThresholdDefault *float64 `json:"threshold_default,omitempty"`
}

// NotificationPreference represents a user's preference for a notification type
type NotificationPreference struct {
	ID             int      `json:"id"`
	UserID         int      `json:"user_id"`
	Type           string   `json:"notification_type"`
	Enabled        bool     `json:"enabled"`
	DeliveryMethod string   `json:"delivery_method"`
	ThresholdValue *float64 `json:"threshold_value"`
}

var notificationTypes = []NotificationTypeInfo{
	{Type: "low_stock", Name: "Low Stock", Description: "When inventory drops below the minimum quantity threshold", Icon: "package", HasThreshold: true, ThresholdLabel: stringPtr("Minimum Qty"), ThresholdDefault: float64Ptr(10)},
	{Type: "overdue_wo", Name: "Overdue Work Order", Description: "When a work order has been in progress longer than the threshold days", Icon: "clock", HasThreshold: true, ThresholdLabel: stringPtr("Days Overdue"), ThresholdDefault: float64Ptr(7)},
	{Type: "open_ncr", Name: "Open NCR", Description: "When an NCR has been open longer than 14 days", Icon: "alert-triangle", HasThreshold: false},
	{Type: "eco_approval", Name: "ECO Approval", Description: "When an ECO requires your approval or is approved", Icon: "check-circle", HasThreshold: false},
	{Type: "eco_implemented", Name: "ECO Implemented", Description: "When an ECO is marked as implemented", Icon: "check-square", HasThreshold: false},
	{Type: "po_received", Name: "PO Received", Description: "When a purchase order is received", Icon: "truck", HasThreshold: false},
	{Type: "wo_completed", Name: "Work Order Completed", Description: "When a work order is completed", Icon: "check", HasThreshold: false},
	{Type: "field_report_critical", Name: "Critical Field Report", Description: "When a field report is marked as critical", Icon: "alert-circle", HasThreshold: false},
}

func float64Ptr(f float64) *float64 { return &f }

func initNotificationPrefsTable() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS notification_preferences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		notification_type TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		delivery_method TEXT DEFAULT 'in_app',
		threshold_value REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, notification_type)
	)`)
	if err != nil {
		log.Println("Failed to create notification_preferences table:", err)
	}
}

// ensureDefaultPreferences creates default preferences for a user if they don't exist
func ensureDefaultPreferences(userID int) {
	for _, nt := range notificationTypes {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM notification_preferences WHERE user_id=? AND notification_type=?", userID, nt.Type).Scan(&count)
		if count == 0 {
			db.Exec("INSERT INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value) VALUES (?, ?, 1, 'in_app', ?)",
				userID, nt.Type, nt.ThresholdDefault)
		}
	}
}

func handleGetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxUserID).(int)
	if !ok || userID == 0 {
		jsonErr(w, "unauthorized", 401)
		return
	}
	ensureDefaultPreferences(userID)

	prefs := getNotificationPrefsForUser(userID)
	jsonResp(w, prefs)
}

func getNotificationPrefsForUser(userID int) []NotificationPreference {
	rows, err := db.Query("SELECT id, user_id, notification_type, enabled, delivery_method, threshold_value FROM notification_preferences WHERE user_id=? ORDER BY notification_type", userID)
	if err != nil {
		return []NotificationPreference{}
	}
	defer rows.Close()

	var prefs []NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		var enabled int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Type, &enabled, &p.DeliveryMethod, &p.ThresholdValue); err != nil {
			continue
		}
		p.Enabled = enabled == 1
		prefs = append(prefs, p)
	}
	if prefs == nil {
		prefs = []NotificationPreference{}
	}
	return prefs
}

func handleUpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxUserID).(int)
	if !ok || userID == 0 {
		jsonErr(w, "unauthorized", 401)
		return
	}
	ensureDefaultPreferences(userID)

	var prefs []NotificationPreference
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	for _, p := range prefs {
		if !isValidNotificationType(p.Type) {
			continue
		}
		if !isValidDeliveryMethod(p.DeliveryMethod) {
			p.DeliveryMethod = "in_app"
		}
		enabled := 0
		if p.Enabled {
			enabled = 1
		}
		db.Exec(`INSERT OR REPLACE INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			userID, p.Type, enabled, p.DeliveryMethod, p.ThresholdValue)
	}

	username := getUsername(r)
	logAudit(db, username, "updated", "notification_preferences", "", "Updated notification preferences")
	handleGetNotificationPreferences(w, r)
}

func handleUpdateSingleNotificationPreference(w http.ResponseWriter, r *http.Request, notifType string) {
	userID, ok := r.Context().Value(ctxUserID).(int)
	if !ok || userID == 0 {
		jsonErr(w, "unauthorized", 401)
		return
	}

	if !isValidNotificationType(notifType) {
		jsonErr(w, "invalid notification type", 400)
		return
	}

	ensureDefaultPreferences(userID)

	var p NotificationPreference
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	if !isValidDeliveryMethod(p.DeliveryMethod) {
		p.DeliveryMethod = "in_app"
	}
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		userID, notifType, enabled, p.DeliveryMethod, p.ThresholdValue)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	handleGetNotificationPreferences(w, r)
}

func handleListNotificationTypes(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, notificationTypes)
}

func isValidNotificationType(t string) bool {
	for _, nt := range notificationTypes {
		if nt.Type == t {
			return true
		}
	}
	return false
}

func isValidDeliveryMethod(m string) bool {
	return m == "in_app" || m == "email" || m == "both"
}

// getUserNotifPref returns the preference for a given user and type.
func getUserNotifPref(userID int, notifType string) (enabled bool, deliveryMethod string, threshold *float64) {
	var e int
	err := db.QueryRow("SELECT enabled, delivery_method, threshold_value FROM notification_preferences WHERE user_id=? AND notification_type=?",
		userID, notifType).Scan(&e, &deliveryMethod, &threshold)
	if err != nil {
		return true, "in_app", nil
	}
	return e == 1, deliveryMethod, threshold
}

// generateNotificationsFiltered generates notifications respecting per-user preferences
func generateNotificationsFiltered() {
	log.Println("Generating notifications (filtered)...")

	rows, err := db.Query("SELECT id FROM users WHERE active=1")
	if err != nil {
		log.Println("Failed to get users for notification generation:", err)
		generateNotifications()
		return
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		generateNotifications()
		return
	}

	for _, uid := range userIDs {
		ensureDefaultPreferences(uid)
		generateNotificationsForUser(uid)
	}
}

func generateNotificationsForUser(userID int) {
	var pending []pendingNotif

	// Low stock
	enabled, deliveryMethod, threshold := getUserNotifPref(userID, "low_stock")
	if enabled {
		rows, err := db.Query(`SELECT ipn, qty_on_hand, reorder_point FROM inventory WHERE reorder_point > 0 AND qty_on_hand < reorder_point`)
		if err == nil {
			for rows.Next() {
				var ipn string
				var qty, rp float64
				rows.Scan(&ipn, &qty, &rp)
				if threshold != nil && qty >= *threshold {
					continue
				}
				p := pendingNotif{
					ntype:          "low_stock",
					severity:       "warning",
					title:          "Low Stock: " + ipn,
					message:        stringPtr(fmt.Sprintf("%.0f on hand, reorder point %.0f", qty, rp)),
					recordID:       stringPtr(ipn),
					module:         stringPtr("inventory"),
					deliveryMethod: deliveryMethod,
					userID:         userID,
				}
				pending = append(pending, p)
			}
			rows.Close()
		}
	}

	// Overdue work orders
	enabled, deliveryMethod, threshold = getUserNotifPref(userID, "overdue_wo")
	if enabled {
		days := 7
		if threshold != nil {
			days = int(*threshold)
		}
		q := fmt.Sprintf(`SELECT id, assembly_ipn FROM work_orders WHERE status = 'in_progress' AND started_at < datetime('now', '-%d days')`, days)
		rows, err := db.Query(q)
		if err == nil {
			for rows.Next() {
				var id, ipn string
				rows.Scan(&id, &ipn)
				p := pendingNotif{
					ntype:          "overdue_wo",
					severity:       "warning",
					title:          "Overdue WO: " + id,
					message:        stringPtr(fmt.Sprintf("In progress for >%d days: %s", days, ipn)),
					recordID:       stringPtr(id),
					module:         stringPtr("workorders"),
					deliveryMethod: deliveryMethod,
					userID:         userID,
				}
				pending = append(pending, p)
			}
			rows.Close()
		}
	}

	// Open NCRs > 14 days
	enabled, deliveryMethod, _ = getUserNotifPref(userID, "open_ncr")
	if enabled {
		rows, err := db.Query(`SELECT id, title FROM ncrs WHERE status = 'open' AND created_at < datetime('now', '-14 days')`)
		if err == nil {
			for rows.Next() {
				var id, title string
				rows.Scan(&id, &title)
				t := title
				p := pendingNotif{
					ntype:          "open_ncr",
					severity:       "error",
					title:          "Open NCR >14d: " + id,
					message:        &t,
					recordID:       stringPtr(id),
					module:         stringPtr("ncr"),
					deliveryMethod: deliveryMethod,
					userID:         userID,
				}
				pending = append(pending, p)
			}
			rows.Close()
		}
	}

	for _, p := range pending {
		createNotificationIfNew(p.ntype, p.severity, p.title, p.message, p.recordID, p.module)
		if p.deliveryMethod == "email" || p.deliveryMethod == "both" {
			if emailConfigEnabled() {
				msg := ""
				if p.message != nil {
					msg = *p.message
				}
				go sendNotificationEmail(0, p.title, msg)
			}
		}
	}
}
