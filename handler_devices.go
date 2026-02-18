package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT serial_number,ipn,COALESCE(firmware_version,''),COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),last_seen,COALESCE(notes,''),created_at FROM devices ORDER BY serial_number")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []Device
	for rows.Next() {
		var d Device
		var ls sql.NullString
		rows.Scan(&d.SerialNumber, &d.IPN, &d.FirmwareVersion, &d.Customer, &d.Location, &d.Status, &d.InstallDate, &ls, &d.Notes, &d.CreatedAt)
		d.LastSeen = sp(ls)
		items = append(items, d)
	}
	if items == nil { items = []Device{} }
	jsonResp(w, items)
}

func handleGetDevice(w http.ResponseWriter, r *http.Request, serial string) {
	var d Device
	var ls sql.NullString
	err := db.QueryRow("SELECT serial_number,ipn,COALESCE(firmware_version,''),COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),last_seen,COALESCE(notes,''),created_at FROM devices WHERE serial_number=?", serial).
		Scan(&d.SerialNumber, &d.IPN, &d.FirmwareVersion, &d.Customer, &d.Location, &d.Status, &d.InstallDate, &ls, &d.Notes, &d.CreatedAt)
	if err != nil { jsonErr(w, "not found", 404); return }
	d.LastSeen = sp(ls)
	jsonResp(w, d)
}

func handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var d Device
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	if d.Status == "" { d.Status = "active" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO devices (serial_number,ipn,firmware_version,customer,location,status,install_date,notes,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		d.SerialNumber, d.IPN, d.FirmwareVersion, d.Customer, d.Location, d.Status, d.InstallDate, d.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	d.CreatedAt = now
	jsonResp(w, d)
}

func handleUpdateDevice(w http.ResponseWriter, r *http.Request, serial string) {
	var d Device
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE devices SET ipn=?,firmware_version=?,customer=?,location=?,status=?,install_date=?,notes=? WHERE serial_number=?",
		d.IPN, d.FirmwareVersion, d.Customer, d.Location, d.Status, d.InstallDate, d.Notes, serial)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	handleGetDevice(w, r, serial)
}

func handleDeviceHistory(w http.ResponseWriter, r *http.Request, serial string) {
	// Get test records
	tests := []TestRecord{}
	rows, _ := db.Query("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records WHERE serial_number=? ORDER BY tested_at DESC", serial)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t TestRecord
			rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
			tests = append(tests, t)
		}
	}
	// Get campaign updates
	campaigns := []CampaignDevice{}
	rows2, _ := db.Query("SELECT campaign_id,serial_number,status,updated_at FROM campaign_devices WHERE serial_number=?", serial)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var cd CampaignDevice
			var ua sql.NullString
			rows2.Scan(&cd.CampaignID, &cd.SerialNumber, &cd.Status, &ua)
			cd.UpdatedAt = sp(ua)
			campaigns = append(campaigns, cd)
		}
	}
	jsonResp(w, map[string]interface{}{"tests": tests, "campaigns": campaigns})
}
