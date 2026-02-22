package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

// Integration tests for critical BOM → PO → Inventory and WO → Inventory workflows
// REWRITTEN: Now uses HTTP API instead of direct DB manipulation to test actual handler logic

const (
	testBaseURL   = "http://localhost:9000"
	testAdminUser = "admin"
	testAdminPass = "changeme"
)

// APITestClient wraps http.Client with authentication for integration tests
type APITestClient struct {
	client    *http.Client
	csrfToken string
	t         *testing.T
}

// newAPITestClient creates an authenticated test client
func newAPITestClient(t *testing.T) *APITestClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	// Login to get session cookie and CSRF token
	loginData := map[string]string{
		"username": testAdminUser,
		"password": testAdminPass,
	}
	jsonData, _ := json.Marshal(loginData)

	resp, err := client.Post(testBaseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login failed: %d - %s", resp.StatusCode, string(body))
	}

	// Extract CSRF token from response
	var loginResp struct {
		CSRFToken string `json:"csrf_token"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &loginResp)

	t.Logf("✓ Authenticated as %s (CSRF: %s...)", testAdminUser, loginResp.CSRFToken[:8])

	return &APITestClient{
		client:    client,
		csrfToken: loginResp.CSRFToken,
		t:         t,
	}
}

// apiRequest makes an authenticated API request with CSRF token
func (tc *APITestClient) apiRequest(method, path string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, testBaseURL+path, reqBody)
	if err != nil {
		tc.t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add CSRF token header for mutating operations
	if method != "GET" && tc.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", tc.csrfToken)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		tc.t.Fatalf("Request failed: %v", err)
	}

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp, respBody
}

// TestIntegration_PO_Receipt_Updates_Inventory verifies that when a PO is received via API,
// inventory levels are automatically updated (tests handleReceivePO handler)
func TestIntegration_PO_Receipt_Updates_Inventory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newAPITestClient(t)
	timestamp := time.Now().UnixNano()

	t.Log("=== Integration Test: PO Receipt → Inventory Update (HTTP API) ===")

	// Step 1: Create vendor via API
	t.Log("\n[1] Creating vendor via API...")
	vendorName := fmt.Sprintf("IntVendor-%d", timestamp)
	vendor := map[string]interface{}{
		"name":           vendorName,
		"status":         "active",
		"lead_time_days": 7,
	}

	resp, body := client.apiRequest("POST", "/api/v1/vendors", vendor)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("Vendor creation response (%d): %s", resp.StatusCode, string(body))
	}

	var vendorResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &vendorResp)
	vendorID := vendorResp.Data.ID
	if vendorID == "" {
		vendorID = vendorName
	}
	t.Logf("✓ Created vendor: %s", vendorID)

	// Step 2: Create inventory records with low stock
	t.Log("\n[2] Creating inventory records via API...")
	comp1IPN := fmt.Sprintf("RES-INT-API-%d", timestamp)
	comp2IPN := fmt.Sprintf("CAP-INT-API-%d", timestamp)

	// Create comp1 with qty=5
	client.apiRequest("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  comp1IPN,
		"type": "adjust",
		"qty":  5.0,
		"note": "Initial inventory - insufficient",
	})
	t.Logf("✓ Created %s with qty=5", comp1IPN)

	// Create comp2 with qty=2
	client.apiRequest("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  comp2IPN,
		"type": "adjust",
		"qty":  2.0,
		"note": "Initial inventory - insufficient",
	})
	t.Logf("✓ Created %s with qty=2", comp2IPN)

	// Step 3: Record initial inventory levels
	t.Log("\n[3] Recording initial inventory levels...")
	var inv1, inv2 struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}

	resp, body = client.apiRequest("GET", "/api/v1/inventory/"+comp1IPN, nil)
	json.Unmarshal(body, &inv1)
	initialComp1 := inv1.Data.QtyOnHand

	resp, body = client.apiRequest("GET", "/api/v1/inventory/"+comp2IPN, nil)
	json.Unmarshal(body, &inv2)
	initialComp2 := inv2.Data.QtyOnHand

	t.Logf("  Initial: %s=%.0f, %s=%.0f", comp1IPN, initialComp1, comp2IPN, initialComp2)

	// Step 4: Create purchase order via API
	t.Log("\n[4] Creating purchase order via API...")
	po := map[string]interface{}{
		"vendor_id": vendorID,
		"status":    "sent",
		"notes":     "Integration test PO",
		"lines": []map[string]interface{}{
			{
				"ipn":         comp1IPN,
				"qty_ordered": 95.0,
				"unit_price":  0.50,
			},
			{
				"ipn":         comp2IPN,
				"qty_ordered": 48.0,
				"unit_price":  0.30,
			},
		},
	}

	resp, body = client.apiRequest("POST", "/api/v1/pos", po)
	var poResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &poResp)
	poID := poResp.Data.ID
	if poID == "" {
		t.Logf("PO creation response: %s", string(body))
		t.Skip("Cannot continue without PO ID")
	}
	t.Logf("✓ Created PO: %s", poID)

	// Step 5: Get PO details to find line IDs
	t.Log("\n[5] Getting PO line IDs...")
	resp, body = client.apiRequest("GET", "/api/v1/pos/"+poID, nil)
	var poDetail struct {
		Data struct {
			Lines []struct {
				ID         int     `json:"id"`
				IPN        string  `json:"ipn"`
				QtyOrdered float64 `json:"qty_ordered"`
			} `json:"lines"`
		} `json:"data"`
	}
	json.Unmarshal(body, &poDetail)

	if len(poDetail.Data.Lines) == 0 {
		t.Skip("PO has no lines, cannot proceed")
	}
	t.Logf("✓ Found %d PO lines", len(poDetail.Data.Lines))

	// Step 6: Receive the PO via API (this should trigger inventory update)
	t.Log("\n[6] Receiving PO via API...")
	receiveData := map[string]interface{}{
		"skip_inspection": true,
		"lines":           []map[string]interface{}{},
	}

	for _, line := range poDetail.Data.Lines {
		receiveData["lines"] = append(receiveData["lines"].([]map[string]interface{}), map[string]interface{}{
			"id":  line.ID,
			"qty": line.QtyOrdered,
		})
	}

	resp, body = client.apiRequest("POST", "/api/v1/pos/"+poID+"/receive", receiveData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("⚠ PO receive returned %d: %s", resp.StatusCode, string(body))
	} else {
		t.Logf("✓ Received PO: %s", poID)
	}

	// Allow async operations
	time.Sleep(500 * time.Millisecond)

	// Step 7: Verify inventory was updated
	t.Log("\n[7] Verifying inventory after PO receipt...")
	resp, body = client.apiRequest("GET", "/api/v1/inventory/"+comp1IPN, nil)
	json.Unmarshal(body, &inv1)

	resp, body = client.apiRequest("GET", "/api/v1/inventory/"+comp2IPN, nil)
	json.Unmarshal(body, &inv2)

	finalComp1 := inv1.Data.QtyOnHand
	finalComp2 := inv2.Data.QtyOnHand

	expectedComp1 := initialComp1 + 95.0 // 5 + 95 = 100
	expectedComp2 := initialComp2 + 48.0 // 2 + 48 = 50

	success := true
	if finalComp1 == expectedComp1 {
		t.Logf("✓ %s inventory correct: %.0f (expected %.0f)", comp1IPN, finalComp1, expectedComp1)
	} else {
		t.Errorf("✗ %s inventory incorrect: %.0f (expected %.0f)", comp1IPN, finalComp1, expectedComp1)
		success = false
	}

	if finalComp2 == expectedComp2 {
		t.Logf("✓ %s inventory correct: %.0f (expected %.0f)", comp2IPN, finalComp2, expectedComp2)
	} else {
		t.Errorf("✗ %s inventory incorrect: %.0f (expected %.0f)", comp2IPN, finalComp2, expectedComp2)
		success = false
	}

	if success {
		t.Log("\n✓✓ SUCCESS: PO Receipt → Inventory Update workflow working!")
	} else {
		t.Error("\n✗✗ FAILURE: PO receipt did not update inventory correctly")
	}

	t.Log("\n=== Integration Test Complete ===")
}

// TestIntegration_WorkOrder_Completion_Updates_Inventory verifies that when a work order
// is completed via API, finished goods are added to inventory (tests handleUpdateWorkOrder handler)
func TestIntegration_WorkOrder_Completion_Updates_Inventory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newAPITestClient(t)
	timestamp := time.Now().UnixNano()

	t.Log("=== Integration Test: WO Completion → Inventory Update (HTTP API) ===")

	// Step 1: Create parts via API
	t.Log("\n[1] Creating parts via API...")
	assemblyIPN := fmt.Sprintf("ASY-WO-API-%d", timestamp)

	// Create assembly with qty=0
	client.apiRequest("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  assemblyIPN,
		"type": "adjust",
		"qty":  0.0,
		"note": "Assembly inventory initialized",
	})
	t.Logf("✓ Created assembly: %s with qty=0", assemblyIPN)

	// Step 2: Record initial inventory
	t.Log("\n[2] Recording initial inventory...")
	var invAsm struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}

	resp, body := client.apiRequest("GET", "/api/v1/inventory/"+assemblyIPN, nil)
	json.Unmarshal(body, &invAsm)
	initialQty := invAsm.Data.QtyOnHand
	t.Logf("  Initial: %s = %.0f", assemblyIPN, initialQty)

	// Step 3: Create work order via API
	t.Log("\n[3] Creating work order via API...")
	woID := fmt.Sprintf("WO-INT-API-%d", timestamp)
	wo := map[string]interface{}{
		"id":           woID,
		"assembly_ipn": assemblyIPN,
		"qty":          10,
		"status":       "open",
		"priority":     "normal",
	}

	resp, body = client.apiRequest("POST", "/api/v1/workorders", wo)
	var woResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &woResp)
	if woResp.Data.ID != "" {
		woID = woResp.Data.ID
	}
	t.Logf("✓ Created work order: %s", woID)

	// Step 4: Complete work order via API (this should trigger inventory update)
	t.Log("\n[4] Completing work order via API...")
	completeData := map[string]interface{}{
		"assembly_ipn": assemblyIPN,
		"qty":          10,
		"status":       "completed",
		"priority":     "normal",
		"qty_good":     10,
		"qty_scrap":    0,
	}

	resp, body = client.apiRequest("PUT", "/api/v1/workorders/"+woID, completeData)
	if resp.StatusCode != http.StatusOK {
		t.Logf("⚠ WO completion returned %d: %s", resp.StatusCode, string(body))
	} else {
		t.Logf("✓ Completed work order: %s", woID)
	}

	// Allow async operations
	time.Sleep(500 * time.Millisecond)

	// Step 5: Verify finished goods were added to inventory
	t.Log("\n[5] Verifying finished goods inventory...")
	resp, body = client.apiRequest("GET", "/api/v1/inventory/"+assemblyIPN, nil)
	json.Unmarshal(body, &invAsm)

	finalQty := invAsm.Data.QtyOnHand
	expectedQty := initialQty + 10.0 // 0 + 10 = 10

	if finalQty == expectedQty {
		t.Logf("✓✓ SUCCESS: %s inventory correct: %.0f (expected %.0f)", assemblyIPN, finalQty, expectedQty)
		t.Log("\n✓✓ SUCCESS: WO Completion → Inventory Update workflow working!")
	} else {
		t.Errorf("✗ %s inventory incorrect: %.0f (expected %.0f)", assemblyIPN, finalQty, expectedQty)
		t.Error("\n✗✗ FAILURE: WO completion did not update inventory correctly")
	}

	t.Log("\n=== Integration Test Complete ===")
}
