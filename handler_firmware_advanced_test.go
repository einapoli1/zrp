package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// Test SQL injection safety in all firmware handlers
func TestFirmwareSQLInjectionSafety(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "Test Campaign", "v1.0.0", "public", "draft")
	insertTestDevice(t, db, "SN-001", "IPN-001", "active")
	insertCampaignDevice(t, db, "FW-001", "SN-001", "pending")

	sqlInjectionTests := []struct {
		name    string
		handler func(w *httptest.ResponseRecorder)
	}{
		{
			name: "GetCampaign with SQL injection in ID",
			handler: func(w *httptest.ResponseRecorder) {
				req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001", nil)
				handleGetCampaign(w, req, "FW-001'; DROP TABLE firmware_campaigns; --")
			},
		},
		{
			name: "CampaignProgress with SQL injection in ID",
			handler: func(w *httptest.ResponseRecorder) {
				req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/progress", nil)
				handleCampaignProgress(w, req, "FW-001' OR '1'='1")
			},
		},
		{
			name: "CampaignDevices with SQL injection in ID",
			handler: func(w *httptest.ResponseRecorder) {
				req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/devices", nil)
				handleCampaignDevices(w, req, "FW-001'; DELETE FROM campaign_devices; --")
			},
		},
		{
			name: "MarkCampaignDevice with SQL injection in campaign ID",
			handler: func(w *httptest.ResponseRecorder) {
				body := map[string]string{"status": "updated"}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001/devices/SN-001", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				handleMarkCampaignDevice(w, req, "FW-001' OR '1'='1", "SN-001")
			},
		},
		{
			name: "MarkCampaignDevice with SQL injection in serial number",
			handler: func(w *httptest.ResponseRecorder) {
				body := map[string]string{"status": "updated"}
				bodyBytes, _ := json.Marshal(body)
				req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001/devices/SN-001", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				handleMarkCampaignDevice(w, req, "FW-001", "SN-001'; DROP TABLE campaign_devices; --")
			},
		},
		{
			name: "CreateCampaign with SQL injection in name field",
			handler: func(w *httptest.ResponseRecorder) {
				campaign := map[string]interface{}{
					"name":    "Test'; DROP TABLE firmware_campaigns; --",
					"version": "v1.0.0",
				}
				bodyBytes, _ := json.Marshal(campaign)
				req := httptest.NewRequest("POST", "/api/firmware/campaigns", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				handleCreateCampaign(w, req)
			},
		},
		{
			name: "CreateCampaign with SQL injection in target_filter field",
			handler: func(w *httptest.ResponseRecorder) {
				campaign := map[string]interface{}{
					"name":          "Safe Name",
					"version":       "v1.0.0",
					"target_filter": "ipn:'; DELETE FROM devices; --",
				}
				bodyBytes, _ := json.Marshal(campaign)
				req := httptest.NewRequest("POST", "/api/firmware/campaigns", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				handleCreateCampaign(w, req)
			},
		},
	}

	for _, tt := range sqlInjectionTests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.handler(w)

			// Verify tables still exist and data is intact
			var campaignCount, deviceCount int
			db.QueryRow("SELECT COUNT(*) FROM firmware_campaigns").Scan(&campaignCount)
			db.QueryRow("SELECT COUNT(*) FROM campaign_devices").Scan(&deviceCount)

			if campaignCount == 0 {
				t.Error("firmware_campaigns table was affected by SQL injection attempt")
			}
			if deviceCount == 0 {
				t.Error("campaign_devices table was affected by SQL injection attempt")
			}

			// Verify test data still exists
			var exists int
			db.QueryRow("SELECT COUNT(*) FROM firmware_campaigns WHERE id='FW-001'").Scan(&exists)
			if exists == 0 {
				t.Error("Test campaign was deleted by SQL injection attempt")
			}
		})
	}
}

// Test concurrent campaign updates
func TestConcurrentCampaignUpdates(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "Concurrent Test", "v1.0.0", "public", "draft")

	// Simulate 10 concurrent updates to the same campaign
	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(iteration int) {
			defer wg.Done()

			update := map[string]interface{}{
				"name":    fmt.Sprintf("Updated Campaign %d", iteration),
				"version": "v1.0.0",
				"status":  "draft",
			}

			bodyBytes, _ := json.Marshal(update)
			req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleUpdateCampaign(w, req, "FW-001")

			if w.Code != 200 && w.Code != 500 {
				errors <- fmt.Errorf("unexpected status code %d for goroutine %d", w.Code, iteration)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check if there were any unexpected errors
	for err := range errors {
		t.Error(err)
	}

	// Verify the campaign still exists and has a valid name
	var name string
	err := db.QueryRow("SELECT name FROM firmware_campaigns WHERE id='FW-001'").Scan(&name)
	if err != nil {
		t.Fatalf("Campaign was corrupted by concurrent updates: %v", err)
	}

	if !strings.HasPrefix(name, "Updated Campaign") {
		t.Errorf("Campaign name was corrupted: %s", name)
	}
}

// Test status transition validation
func TestCampaignStatusTransitions(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name           string
		initialStatus  string
		targetStatus   string
		shouldSucceed  bool
	}{
		// Valid transitions
		{"draft to active", "draft", "active", true},
		{"active to paused", "active", "paused", true},
		{"paused to active", "paused", "active", true},
		{"active to completed", "active", "completed", true},
		{"active to cancelled", "active", "cancelled", true},
		{"paused to completed", "paused", "completed", true},
		{"paused to cancelled", "paused", "cancelled", true},
		
		// Invalid status values (should be caught by validation)
		{"draft to invalid", "draft", "invalid_status", false},
		{"active to random", "active", "random", false},
		
		// Edge cases
		{"draft to draft", "draft", "draft", true},  // Idempotent
		{"completed to completed", "completed", "completed", true},  // Idempotent
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaignID := fmt.Sprintf("FW-%03d", i+1)
			insertTestCampaign(t, db, campaignID, "Status Test", "v1.0.0", "public", tt.initialStatus)

			update := map[string]interface{}{
				"name":          "Status Test",
				"version":       "v1.0.0",
				"category":      "public",
				"status":        tt.targetStatus,
				"target_filter": "",
				"notes":         "",
			}

			bodyBytes, _ := json.Marshal(update)
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/firmware/campaigns/%s", campaignID), bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleUpdateCampaign(w, req, campaignID)

			if tt.shouldSucceed {
				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d for valid transition %s -> %s: %s", 
						w.Code, tt.initialStatus, tt.targetStatus, w.Body.String())
				}

				// Verify status was actually updated
				var currentStatus string
				db.QueryRow("SELECT status FROM firmware_campaigns WHERE id=?", campaignID).Scan(&currentStatus)
				if currentStatus != tt.targetStatus {
					t.Errorf("Status not updated correctly. Expected %s, got %s", tt.targetStatus, currentStatus)
				}
			} else {
				if w.Code == 200 {
					t.Errorf("Expected failure for invalid status transition %s -> %s", tt.initialStatus, tt.targetStatus)
				}
			}
		})
	}
}

// Test duplicate device prevention
func TestDuplicateDeviceInCampaign(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "Duplicate Test", "v1.0.0", "public", "draft")
	insertTestDevice(t, db, "SN-001", "IPN-001", "active")

	// Add device first time - should succeed
	_, err := db.Exec("INSERT INTO campaign_devices (campaign_id, serial_number, status) VALUES (?, ?, ?)", 
		"FW-001", "SN-001", "pending")
	if err != nil {
		t.Fatalf("Failed to insert first device: %v", err)
	}

	// Try to add same device again - should fail due to PRIMARY KEY constraint
	_, err = db.Exec("INSERT INTO campaign_devices (campaign_id, serial_number, status) VALUES (?, ?, ?)", 
		"FW-001", "SN-001", "pending")
	if err == nil {
		t.Error("Expected error when inserting duplicate device, but got none")
	}

	// Verify only one record exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND serial_number=?", 
		"FW-001", "SN-001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 device record, got %d", count)
	}

	// Verify INSERT OR IGNORE works (used in launch handler)
	result, err := db.Exec("INSERT OR IGNORE INTO campaign_devices (campaign_id, serial_number, status) VALUES (?, ?, ?)", 
		"FW-001", "SN-001", "pending")
	if err != nil {
		t.Errorf("INSERT OR IGNORE failed: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 0 {
		t.Error("INSERT OR IGNORE should not insert duplicate device")
	}

	// Verify still only one record
	db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND serial_number=?", 
		"FW-001", "SN-001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 device record after INSERT OR IGNORE, got %d", count)
	}
}

// Test foreign key constraints for firmware campaigns
func TestFirmwareForeignKeyConstraints(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "FK Test", "v1.0.0", "public", "draft")
	insertTestDevice(t, db, "SN-001", "IPN-001", "active")
	insertCampaignDevice(t, db, "FW-001", "SN-001", "pending")

	t.Run("Delete campaign should cascade to campaign_devices", func(t *testing.T) {
		// Insert test data
		insertTestCampaign(t, db, "FW-002", "Cascade Test", "v1.0.0", "public", "draft")
		insertCampaignDevice(t, db, "FW-002", "SN-001", "pending")

		// Verify device is in campaign
		var count int
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id='FW-002'").Scan(&count)
		if count != 1 {
			t.Fatalf("Expected 1 device in campaign FW-002, got %d", count)
		}

		// Delete campaign
		_, err := db.Exec("DELETE FROM firmware_campaigns WHERE id='FW-002'")
		if err != nil {
			t.Fatalf("Failed to delete campaign: %v", err)
		}

		// Verify campaign_devices were cascaded
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id='FW-002'").Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 devices after cascade delete, got %d", count)
		}
	})

	t.Run("Cannot add device to non-existent campaign", func(t *testing.T) {
		// Try to insert device with non-existent campaign_id
		_, err := db.Exec("INSERT INTO campaign_devices (campaign_id, serial_number, status) VALUES (?, ?, ?)", 
			"FW-NONEXISTENT", "SN-001", "pending")
		
		if err == nil {
			t.Error("Expected error when inserting device with non-existent campaign_id")
		}
	})

	t.Run("Cannot add non-existent device to campaign", func(t *testing.T) {
		// Try to insert device with non-existent serial_number
		_, err := db.Exec("INSERT INTO campaign_devices (campaign_id, serial_number, status) VALUES (?, ?, ?)", 
			"FW-001", "SN-NONEXISTENT", "pending")
		
		if err == nil {
			t.Error("Expected error when inserting non-existent device to campaign")
		}
	})

	t.Run("Delete device should cascade to campaign_devices", func(t *testing.T) {
		// Insert test device
		insertTestDevice(t, db, "SN-002", "IPN-002", "active")
		insertCampaignDevice(t, db, "FW-001", "SN-002", "pending")

		// Verify device is in campaign
		var count int
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE serial_number='SN-002'").Scan(&count)
		if count != 1 {
			t.Fatalf("Expected 1 campaign with device SN-002, got %d", count)
		}

		// Delete device
		_, err := db.Exec("DELETE FROM devices WHERE serial_number='SN-002'")
		if err != nil {
			t.Fatalf("Failed to delete device: %v", err)
		}

		// Verify campaign_devices were cascaded
		db.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE serial_number='SN-002'").Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 campaign_devices after cascade delete, got %d", count)
		}
	})
}

// Test progress tracking accuracy
func TestProgressTrackingAccuracy(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "Progress Test", "v1.0.0", "public", "active")
	
	// Insert devices in different states
	devices := []struct {
		serial string
		status string
	}{
		{"SN-001", "pending"},
		{"SN-002", "pending"},
		{"SN-003", "in_progress"},
		{"SN-004", "in_progress"},
		{"SN-005", "success"},
		{"SN-006", "success"},
		{"SN-007", "success"},
		{"SN-008", "failed"},
		{"SN-009", "skipped"},
		{"SN-010", "pending"},
	}

	for _, d := range devices {
		insertTestDevice(t, db, d.serial, "IPN-001", "active")
		insertCampaignDevice(t, db, "FW-001", d.serial, d.status)
	}

	// Test progress endpoint
	req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/progress", nil)
	w := httptest.NewRecorder()
	handleCampaignProgress(w, req, "FW-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	progress := resp.Data.(map[string]interface{})

	// Verify counts
	expectedCounts := map[string]int{
		"total":       10,
		"pending":     3,
		"in_progress": 2,
		"success":     3,
		"failed":      1,
	}

	for key, expectedValue := range expectedCounts {
		actualValue := int(progress[key].(float64))
		if actualValue != expectedValue {
			t.Errorf("Progress count mismatch for %s: expected %d, got %d", key, expectedValue, actualValue)
		}
	}
}

// Test campaign launch with no active devices
func TestLaunchCampaignNoActiveDevices(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "No Devices Test", "v1.0.0", "public", "draft")
	
	// Insert only inactive devices
	insertTestDevice(t, db, "SN-001", "IPN-001", "inactive")
	insertTestDevice(t, db, "SN-002", "IPN-001", "decommissioned")
	insertTestDevice(t, db, "SN-003", "IPN-002", "rma")

	req := httptest.NewRequest("POST", "/api/firmware/campaigns/FW-001/launch", nil)
	w := httptest.NewRecorder()

	handleLaunchCampaign(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	devicesAdded := int(result["devices_added"].(float64))

	if devicesAdded != 0 {
		t.Errorf("Expected 0 devices added (no active devices), got %d", devicesAdded)
	}

	// Verify campaign status is still updated to active
	var status string
	db.QueryRow("SELECT status FROM firmware_campaigns WHERE id='FW-001'").Scan(&status)
	if status != "active" {
		t.Errorf("Expected campaign status 'active', got '%s'", status)
	}
}

// Test campaign device update timestamp
func TestCampaignDeviceUpdateTimestamp(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "Timestamp Test", "v1.0.0", "public", "active")
	insertTestDevice(t, db, "SN-001", "IPN-001", "active")
	insertCampaignDevice(t, db, "FW-001", "SN-001", "pending")

	// Wait a moment to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Mark device as success
	body := map[string]string{"status": "success"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001/devices/SN-001", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleMarkCampaignDevice(w, req, "FW-001", "SN-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify updated_at timestamp was set
	var updatedAt sql.NullString
	err := db.QueryRow("SELECT updated_at FROM campaign_devices WHERE campaign_id='FW-001' AND serial_number='SN-001'").Scan(&updatedAt)
	if err != nil {
		t.Fatalf("Failed to query updated_at: %v", err)
	}

	if !updatedAt.Valid || updatedAt.String == "" {
		t.Error("updated_at timestamp should be set after marking device")
	}

	// Verify timestamp is recent (within last minute)
	// SQLite may use different timestamp formats
	var timestamp time.Time
	var parseErr error
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}
	
	for _, format := range formats {
		timestamp, parseErr = time.Parse(format, updatedAt.String)
		if parseErr == nil {
			break
		}
	}
	
	if parseErr != nil {
		t.Fatalf("Failed to parse timestamp with any format: %v (value: %s)", parseErr, updatedAt.String)
	}

	if time.Since(timestamp) > time.Minute {
		t.Error("updated_at timestamp is not recent")
	}
}

// Test invalid campaign device status values
func TestInvalidCampaignDeviceStatus(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(t, db, "FW-001", "Invalid Status Test", "v1.0.0", "public", "active")
	insertTestDevice(t, db, "SN-001", "IPN-001", "active")
	insertCampaignDevice(t, db, "FW-001", "SN-001", "pending")

	invalidStatuses := []string{
		"completed",  // Not allowed for campaign_devices (should be "success")
		"active",
		"draft",
		"random_status",
		"'; DROP TABLE campaign_devices; --",
	}

	for _, invalidStatus := range invalidStatuses {
		t.Run(fmt.Sprintf("Status: %s", invalidStatus), func(t *testing.T) {
			body := map[string]string{"status": invalidStatus}
			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001/devices/SN-001", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleMarkCampaignDevice(w, req, "FW-001", "SN-001")

			if w.Code == 200 {
				t.Errorf("Expected error for invalid status '%s', but got success", invalidStatus)
			}

			// Verify status wasn't changed
			var currentStatus string
			db.QueryRow("SELECT status FROM campaign_devices WHERE campaign_id='FW-001' AND serial_number='SN-001'").Scan(&currentStatus)
			if currentStatus == invalidStatus {
				t.Errorf("Status was incorrectly changed to invalid value: %s", invalidStatus)
			}
		})
	}
}

// Test empty campaign name/version validation
func TestEmptyCampaignFields(t *testing.T) {
	oldDB := db
	db = setupFirmwareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name     string
		campaign map[string]interface{}
	}{
		{
			name:     "Empty name",
			campaign: map[string]interface{}{"name": "", "version": "v1.0.0"},
		},
		{
			name:     "Whitespace-only name",
			campaign: map[string]interface{}{"name": "   ", "version": "v1.0.0"},
		},
		{
			name:     "Empty version",
			campaign: map[string]interface{}{"name": "Test Campaign", "version": ""},
		},
		{
			name:     "Whitespace-only version",
			campaign: map[string]interface{}{"name": "Test Campaign", "version": "   "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.campaign)
			req := httptest.NewRequest("POST", "/api/firmware/campaigns", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateCampaign(w, req)

			if w.Code != 400 {
				t.Errorf("Expected status 400 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// Benchmark campaign progress calculation
func BenchmarkCampaignProgress(b *testing.B) {
	oldDB := db
	db = setupFirmwareTestDB(&testing.T{})
	defer func() { db.Close(); db = oldDB }()

	insertTestCampaign(&testing.T{}, db, "FW-001", "Benchmark Test", "v1.0.0", "public", "active")

	// Insert 1000 devices
	for i := 0; i < 1000; i++ {
		serial := fmt.Sprintf("SN-%04d", i)
		insertTestDevice(&testing.T{}, db, serial, "IPN-001", "active")
		status := []string{"pending", "in_progress", "success", "failed"}[i%4]
		insertCampaignDevice(&testing.T{}, db, "FW-001", serial, status)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/progress", nil)
		w := httptest.NewRecorder()
		handleCampaignProgress(w, req, "FW-001")
	}
}
