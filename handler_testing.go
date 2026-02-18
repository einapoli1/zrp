package main

import (
	"net/http"
	"time"
)

func handleListTests(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records ORDER BY tested_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []TestRecord
	for rows.Next() {
		var t TestRecord
		rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
		items = append(items, t)
	}
	if items == nil { items = []TestRecord{} }
	jsonResp(w, items)
}

func handleGetTests(w http.ResponseWriter, r *http.Request, serial string) {
	rows, err := db.Query("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records WHERE serial_number=? ORDER BY tested_at DESC", serial)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []TestRecord
	for rows.Next() {
		var t TestRecord
		rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
		items = append(items, t)
	}
	if items == nil { items = []TestRecord{} }
	jsonResp(w, items)
}

func handleCreateTest(w http.ResponseWriter, r *http.Request) {
	var t TestRecord
	if err := decodeBody(r, &t); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec("INSERT INTO test_records (serial_number,ipn,firmware_version,test_type,result,measurements,notes,tested_by,tested_at) VALUES (?,?,?,?,?,?,?,?,?)",
		t.SerialNumber, t.IPN, t.FirmwareVersion, t.TestType, t.Result, t.Measurements, t.Notes, "operator", now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	id, _ := res.LastInsertId()
	t.ID = int(id)
	t.TestedAt = now
	t.TestedBy = "operator"
	logAudit(db, getUsername(r), "created", "test", t.SerialNumber, "Test "+t.Result+" for "+t.SerialNumber)
	jsonResp(w, t)
}
