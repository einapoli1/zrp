package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleListFieldReports(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id,title,COALESCE(report_type,''),status,COALESCE(priority,''),
		COALESCE(customer_name,''),COALESCE(site_location,''),COALESCE(device_ipn,''),
		COALESCE(device_serial,''),COALESCE(reported_by,''),COALESCE(reported_at,''),
		COALESCE(description,''),COALESCE(root_cause,''),COALESCE(resolution,''),
		resolved_at,COALESCE(ncr_id,''),COALESCE(eco_id,''),created_at,updated_at
		FROM field_reports WHERE 1=1`
	var args []interface{}

	if v := r.URL.Query().Get("status"); v != "" {
		query += " AND status=?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("priority"); v != "" {
		query += " AND priority=?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("report_type"); v != "" {
		query += " AND report_type=?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("from"); v != "" {
		query += " AND created_at >= ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("to"); v != "" {
		query += " AND created_at <= ?"
		args = append(args, v)
	}

	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []FieldReport
	for rows.Next() {
		var fr FieldReport
		var ra sql.NullString
		rows.Scan(&fr.ID, &fr.Title, &fr.ReportType, &fr.Status, &fr.Priority,
			&fr.CustomerName, &fr.SiteLocation, &fr.DeviceIPN, &fr.DeviceSerial,
			&fr.ReportedBy, &fr.ReportedAt, &fr.Description, &fr.RootCause,
			&fr.Resolution, &ra, &fr.NcrID, &fr.EcoID, &fr.CreatedAt, &fr.UpdatedAt)
		fr.ResolvedAt = sp(ra)
		items = append(items, fr)
	}
	if items == nil {
		items = []FieldReport{}
	}
	jsonResp(w, items)
}

func handleGetFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	var fr FieldReport
	var ra sql.NullString
	err := db.QueryRow(`SELECT id,title,COALESCE(report_type,''),status,COALESCE(priority,''),
		COALESCE(customer_name,''),COALESCE(site_location,''),COALESCE(device_ipn,''),
		COALESCE(device_serial,''),COALESCE(reported_by,''),COALESCE(reported_at,''),
		COALESCE(description,''),COALESCE(root_cause,''),COALESCE(resolution,''),
		resolved_at,COALESCE(ncr_id,''),COALESCE(eco_id,''),created_at,updated_at
		FROM field_reports WHERE id=?`, id).
		Scan(&fr.ID, &fr.Title, &fr.ReportType, &fr.Status, &fr.Priority,
			&fr.CustomerName, &fr.SiteLocation, &fr.DeviceIPN, &fr.DeviceSerial,
			&fr.ReportedBy, &fr.ReportedAt, &fr.Description, &fr.RootCause,
			&fr.Resolution, &ra, &fr.NcrID, &fr.EcoID, &fr.CreatedAt, &fr.UpdatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	fr.ResolvedAt = sp(ra)
	jsonResp(w, fr)
}

func handleCreateFieldReport(w http.ResponseWriter, r *http.Request) {
	var fr FieldReport
	if err := decodeBody(r, &fr); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if strings.TrimSpace(fr.Title) == "" {
		jsonErr(w, "title is required", 400)
		return
	}
	fr.ID = nextID("FR", "field_reports", 3)
	if fr.Status == "" {
		fr.Status = "open"
	}
	if fr.Priority == "" {
		fr.Priority = "medium"
	}
	if fr.ReportType == "" {
		fr.ReportType = "failure"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	fr.CreatedAt = now
	fr.UpdatedAt = now
	if fr.ReportedAt == "" {
		fr.ReportedAt = now
	}

	_, err := db.Exec(`INSERT INTO field_reports (id,title,report_type,status,priority,customer_name,
		site_location,device_ipn,device_serial,reported_by,reported_at,description,
		root_cause,resolution,ncr_id,eco_id,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		fr.ID, fr.Title, fr.ReportType, fr.Status, fr.Priority, fr.CustomerName,
		fr.SiteLocation, fr.DeviceIPN, fr.DeviceSerial, fr.ReportedBy, fr.ReportedAt,
		fr.Description, fr.RootCause, fr.Resolution, fr.NcrID, fr.EcoID, fr.CreatedAt, fr.UpdatedAt)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logAudit(db, getUsername(r), "created", "field_report", fr.ID, "Created "+fr.ID+": "+fr.Title)
	jsonResp(w, fr)
}

func handleUpdateFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	// Check exists
	var existing FieldReport
	err := db.QueryRow("SELECT id FROM field_reports WHERE id=?", id).Scan(&existing.ID)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	var body map[string]interface{}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	getString := func(key string) string {
		if v, ok := body[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	sets := []string{}
	args := []interface{}{}
	fields := []string{"title", "report_type", "status", "priority", "customer_name",
		"site_location", "device_ipn", "device_serial", "reported_by",
		"description", "root_cause", "resolution", "ncr_id", "eco_id"}
	for _, f := range fields {
		if _, ok := body[f]; ok {
			sets = append(sets, f+"=?")
			args = append(args, getString(f))
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	// Auto-set resolved_at when status becomes resolved
	if status := getString("status"); status == "resolved" {
		sets = append(sets, "resolved_at=?")
		args = append(args, now)
	}

	sets = append(sets, "updated_at=?")
	args = append(args, now)
	args = append(args, id)

	if len(sets) > 0 {
		_, err = db.Exec("UPDATE field_reports SET "+strings.Join(sets, ",")+" WHERE id=?", args...)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
	}

	logAudit(db, getUsername(r), "updated", "field_report", id, "Updated "+id)
	handleGetFieldReport(w, r, id)
}

func handleDeleteFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	res, err := db.Exec("DELETE FROM field_reports WHERE id=?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonErr(w, "not found", 404)
		return
	}
	logAudit(db, getUsername(r), "deleted", "field_report", id, "Deleted "+id)
	jsonResp(w, map[string]string{"status": "ok"})
}

func handleFieldReportCreateNCR(w http.ResponseWriter, r *http.Request, id string) {
	var fr FieldReport
	var ra sql.NullString
	err := db.QueryRow(`SELECT id,title,COALESCE(report_type,''),status,COALESCE(priority,''),
		COALESCE(customer_name,''),COALESCE(site_location,''),COALESCE(device_ipn,''),
		COALESCE(device_serial,''),COALESCE(reported_by,''),COALESCE(reported_at,''),
		COALESCE(description,''),COALESCE(root_cause,''),COALESCE(resolution,''),
		resolved_at,COALESCE(ncr_id,''),COALESCE(eco_id,''),created_at,updated_at
		FROM field_reports WHERE id=?`, id).
		Scan(&fr.ID, &fr.Title, &fr.ReportType, &fr.Status, &fr.Priority,
			&fr.CustomerName, &fr.SiteLocation, &fr.DeviceIPN, &fr.DeviceSerial,
			&fr.ReportedBy, &fr.ReportedAt, &fr.Description, &fr.RootCause,
			&fr.Resolution, &ra, &fr.NcrID, &fr.EcoID, &fr.CreatedAt, &fr.UpdatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	// Create NCR from field report data
	ncrID := nextID("NCR", "ncrs", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	severity := "minor"
	if fr.Priority == "critical" {
		severity = "critical"
	} else if fr.Priority == "high" {
		severity = "major"
	}

	_, err = db.Exec(`INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		ncrID, fr.Title, fr.Description, fr.DeviceIPN, fr.DeviceSerial, "field_report", severity, "open", now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Link NCR back to field report
	db.Exec("UPDATE field_reports SET ncr_id=?, updated_at=? WHERE id=?", ncrID, now, id)

	logAudit(db, getUsername(r), "created", "ncr", ncrID, fmt.Sprintf("Created %s from field report %s", ncrID, id))

	jsonResp(w, map[string]string{"id": ncrID, "title": fr.Title, "status": "open", "severity": severity})
}
