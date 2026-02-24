package main

import (
	"encoding/json"
	"net/http"
)

type DashboardWidget struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	WidgetType string `json:"widget_type"`
	Position   int    `json:"position"`
	Enabled    int    `json:"enabled"`
}

func handleGetDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, user_id, widget_type, position, enabled FROM dashboard_widgets WHERE user_id=0 ORDER BY position ASC")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var widgets []DashboardWidget
	for rows.Next() {
		var wg DashboardWidget
		rows.Scan(&wg.ID, &wg.UserID, &wg.WidgetType, &wg.Position, &wg.Enabled)
		widgets = append(widgets, wg)
	}
	if widgets == nil {
		widgets = []DashboardWidget{}
	}
	jsonResp(w, widgets)
}

func handleUpdateDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	var updates []struct {
		WidgetType string `json:"widget_type"`
		Position   int    `json:"position"`
		Enabled    int    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	for _, u := range updates {
		db.Exec("UPDATE dashboard_widgets SET position=?, enabled=? WHERE widget_type=? AND user_id=0",
			u.Position, u.Enabled, u.WidgetType)
	}
	logAudit(db, getUsername(r), "updated", "dashboard", "widgets", "Updated dashboard widget layout")
	handleGetDashboardWidgets(w, r)
}
