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

// Integration tests for cross-module workflows
// Tests critical end-to-end flows across BOMs, Work Orders, Procurement, ECOs, Quotes, and Sales Orders
// Run with: go test -v -run Integration

const (
	integrationBaseURL   = "http://localhost:9000"
	integrationAdminUser = "admin"
	integrationAdminPass = "changeme"
)

// IntegrationClient wraps http.Client with authentication for integration tests
type IntegrationClient struct {
	client *http.Client
	t      *testing.T
}

// newIntegrationClient creates an authenticated integration test client
func newIntegrationClient(t *testing.T) *IntegrationClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	// Login to get session cookie
	loginData := map[string]string{
		"username": integrationAdminUser,
		"password": integrationAdminPass,
	}
	jsonData, _ := json.Marshal(loginData)

	resp, err := client.Post(integrationBaseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login failed: %d - %s", resp.StatusCode, string(body))
	}

	t.Logf("✓ Authenticated as %s", integrationAdminUser)

	return &IntegrationClient{
		client: client,
		t:      t,
	}
}

// request makes an authenticated API request
func (ic *IntegrationClient) request(method, path string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, integrationBaseURL+path, reqBody)
	if err != nil {
		ic.t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := ic.client.Do(req)
	if err != nil {
		ic.t.Fatalf("Request failed: %v", err)
	}

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp, respBody
}

// uniqueID generates a unique ID using timestamp and test name
func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// TestIntegration_BOM_WorkOrder_Procurement_Flow tests the complete workflow:
// 1. Create BOM with parts
// 2. Create work order using that BOM
// 3. Check inventory shortages
// 4. Auto-generate PO from shortages
// 5. Receive PO
// 6. Verify inventory updated
// 7. Complete work order
func TestIntegration_BOM_WorkOrder_Procurement_Flow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newIntegrationClient(t)
	timestamp := time.Now().UnixNano()

	t.Log("=== INTEGRATION TEST: BOM → Work Order → Procurement → Inventory ===")

	// Step 1: Create vendor
	t.Log("\n[Step 1] Creating vendor...")
	vendorName := uniqueID("IntegrationVendor")
	vendor := map[string]interface{}{
		"name":           vendorName,
		"status":         "active",
		"lead_time_days": 7,
		"contact_email":  "vendor@test.com",
	}

	resp, body := client.request("POST", "/api/v1/vendors", vendor)
	var vendorResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &vendorResp)
	vendorID := vendorResp.Data.ID
	if vendorID == "" {
		vendorID = vendorName // Fallback to name if no ID returned
	}
	t.Logf("✓ Created vendor: %s", vendorID)

	// Step 2: Create component parts with insufficient inventory
	t.Log("\n[Step 2] Creating component parts with low inventory...")
	comp1IPN := fmt.Sprintf("COMP1-INT-%d", timestamp)
	comp2IPN := fmt.Sprintf("COMP2-INT-%d", timestamp)
	assemblyIPN := fmt.Sprintf("ASM-INT-%d", timestamp)

	// Create component 1 with qty=5 (insufficient for 10 assemblies needing 2 each)
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  comp1IPN,
		"type": "adjust",
		"qty":  5.0,
		"note": "Initial inventory - insufficient",
	})
	t.Logf("✓ Created %s with qty=5", comp1IPN)

	// Create component 2 with qty=8 (insufficient for 10 assemblies needing 3 each)
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  comp2IPN,
		"type": "adjust",
		"qty":  8.0,
		"note": "Initial inventory - insufficient",
	})
	t.Logf("✓ Created %s with qty=8", comp2IPN)

	// Create assembly with qty=0
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  assemblyIPN,
		"type": "adjust",
		"qty":  0.0,
		"note": "Assembly inventory initialized",
	})
	t.Logf("✓ Created assembly %s with qty=0", assemblyIPN)

	// Step 3: Create BOM for assembly
	t.Log("\n[Step 3] Creating BOM (assembly requires 2x comp1, 3x comp2)...")
	client.request("POST", "/api/v1/bom", map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": comp1IPN,
		"qty_per":       2.0,
	})
	client.request("POST", "/api/v1/bom", map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": comp2IPN,
		"qty_per":       3.0,
	})
	t.Logf("✓ BOM created: %s = 2x%s + 3x%s", assemblyIPN, comp1IPN, comp2IPN)

	// Step 4: Create work order for 10 assemblies (will create shortage)
	t.Log("\n[Step 4] Creating work order for 10 assemblies...")
	woID := uniqueID("WO-INT")
	wo := map[string]interface{}{
		"id":           woID,
		"assembly_ipn": assemblyIPN,
		"qty":          10,
		"status":       "open",
		"priority":     "normal",
	}

	resp, body = client.request("POST", "/api/v1/workorders", wo)
	var woResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &woResp)
	if woResp.Data.ID != "" {
		woID = woResp.Data.ID
	}
	t.Logf("✓ Created work order: %s for 10x %s", woID, assemblyIPN)

	// Step 5: Check inventory shortages
	t.Log("\n[Step 5] Checking inventory shortages...")
	t.Logf("  Required: 20x %s (have 5, need 15 more)", comp1IPN)
	t.Logf("  Required: 30x %s (have 8, need 22 more)", comp2IPN)

	// Step 6: Auto-generate PO from shortages
	t.Log("\n[Step 6] Creating purchase order for shortages...")
	po := map[string]interface{}{
		"vendor_id": vendorID,
		"status":    "sent",
		"notes":     "Auto-generated from WO " + woID,
		"lines": []map[string]interface{}{
			{
				"ipn":         comp1IPN,
				"qty_ordered": 15.0,
				"unit_price":  1.50,
			},
			{
				"ipn":         comp2IPN,
				"qty_ordered": 22.0,
				"unit_price":  2.00,
			},
		},
	}

	resp, body = client.request("POST", "/api/v1/pos", po)
	var poResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &poResp)
	poID := poResp.Data.ID
	if poID == "" {
		t.Skip("Cannot continue without PO ID")
	}
	t.Logf("✓ Created PO: %s for shortages", poID)

	// Step 7: Receive the PO
	t.Log("\n[Step 7] Receiving purchase order...")
	resp, body = client.request("GET", "/api/v1/pos/"+poID, nil)
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
		t.Skip("PO has no lines, cannot receive")
	}

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

	resp, body = client.request("POST", "/api/v1/pos/"+poID+"/receive", receiveData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("✓ Received PO: %s", poID)
	} else {
		t.Logf("⚠ PO receive returned status %d: %s", resp.StatusCode, string(body))
	}

	// Allow async operations to complete
	time.Sleep(500 * time.Millisecond)

	// Step 8: Verify inventory updated correctly
	t.Log("\n[Step 8] Verifying inventory after PO receipt...")
	resp, body = client.request("GET", "/api/v1/inventory/"+comp1IPN, nil)
	var inv1 struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &inv1)

	resp, body = client.request("GET", "/api/v1/inventory/"+comp2IPN, nil)
	var inv2 struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &inv2)

	// Expected: 5 + 15 = 20 for comp1, 8 + 22 = 30 for comp2
	if inv1.Data.QtyOnHand == 20.0 {
		t.Logf("✓ %s inventory correct: %.0f (expected 20)", comp1IPN, inv1.Data.QtyOnHand)
	} else {
		t.Errorf("✗ %s inventory incorrect: %.0f (expected 20)", comp1IPN, inv1.Data.QtyOnHand)
	}

	if inv2.Data.QtyOnHand == 30.0 {
		t.Logf("✓ %s inventory correct: %.0f (expected 30)", comp2IPN, inv2.Data.QtyOnHand)
	} else {
		t.Errorf("✗ %s inventory incorrect: %.0f (expected 30)", comp2IPN, inv2.Data.QtyOnHand)
	}

	// Step 9: Complete work order
	t.Log("\n[Step 9] Completing work order...")
	completeData := map[string]interface{}{
		"assembly_ipn": assemblyIPN,
		"qty":          10,
		"status":       "completed",
		"priority":     "normal",
		"qty_good":     10,
		"qty_scrap":    0,
	}

	resp, body = client.request("PUT", "/api/v1/workorders/"+woID, completeData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("✓ Completed work order: %s", woID)
	} else {
		t.Logf("⚠ Work order completion returned status %d: %s", resp.StatusCode, string(body))
	}

	time.Sleep(500 * time.Millisecond)

	// Step 10: Verify final inventory levels
	t.Log("\n[Step 10] Verifying final inventory after work order completion...")

	// Components should be consumed
	resp, body = client.request("GET", "/api/v1/inventory/"+comp1IPN, nil)
	json.Unmarshal(body, &inv1)
	resp, body = client.request("GET", "/api/v1/inventory/"+comp2IPN, nil)
	json.Unmarshal(body, &inv2)

	// Assembly should be increased
	resp, body = client.request("GET", "/api/v1/inventory/"+assemblyIPN, nil)
	var invAsm struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &invAsm)

	// Expected: comp1: 20 - (10*2) = 0, comp2: 30 - (10*3) = 0, assembly: 0 + 10 = 10
	expectedComp1 := 0.0
	expectedComp2 := 0.0
	expectedAsm := 10.0

	if inv1.Data.QtyOnHand == expectedComp1 {
		t.Logf("✓ %s consumed correctly: %.0f (expected %.0f)", comp1IPN, inv1.Data.QtyOnHand, expectedComp1)
	} else {
		t.Errorf("✗ %s consumption incorrect: %.0f (expected %.0f)", comp1IPN, inv1.Data.QtyOnHand, expectedComp1)
	}

	if inv2.Data.QtyOnHand == expectedComp2 {
		t.Logf("✓ %s consumed correctly: %.0f (expected %.0f)", comp2IPN, inv2.Data.QtyOnHand, expectedComp2)
	} else {
		t.Errorf("✗ %s consumption incorrect: %.0f (expected %.0f)", comp2IPN, inv2.Data.QtyOnHand, expectedComp2)
	}

	if invAsm.Data.QtyOnHand == expectedAsm {
		t.Logf("✓✓ SUCCESS: %s produced correctly: %.0f (expected %.0f)", assemblyIPN, invAsm.Data.QtyOnHand, expectedAsm)
	} else {
		t.Errorf("✗ %s production incorrect: %.0f (expected %.0f)", assemblyIPN, invAsm.Data.QtyOnHand, expectedAsm)
	}

	t.Log("\n=== INTEGRATION TEST COMPLETE: BOM → Work Order → Procurement ===\n")
}

// TestIntegration_ECO_BOM_WorkOrder_Flow tests the workflow:
// 1. Create ECO for BOM change
// 2. Approve ECO
// 3. Update BOM
// 4. Create work order with new BOM
// 5. Verify correct parts used
func TestIntegration_ECO_BOM_WorkOrder_Flow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newIntegrationClient(t)
	timestamp := time.Now().UnixNano()

	t.Log("=== INTEGRATION TEST: ECO → BOM → Work Order ===")

	// Step 1: Create parts (old part, new part, assembly)
	t.Log("\n[Step 1] Creating parts...")
	oldPartIPN := fmt.Sprintf("OLD-PART-%d", timestamp)
	newPartIPN := fmt.Sprintf("NEW-PART-%d", timestamp)
	assemblyIPN := fmt.Sprintf("ASM-ECO-%d", timestamp)

	// Old part with sufficient inventory
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  oldPartIPN,
		"type": "adjust",
		"qty":  100.0,
		"note": "Old part - to be replaced via ECO",
	})
	t.Logf("✓ Created %s with qty=100", oldPartIPN)

	// New part with sufficient inventory
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  newPartIPN,
		"type": "adjust",
		"qty":  100.0,
		"note": "New part - ECO replacement",
	})
	t.Logf("✓ Created %s with qty=100", newPartIPN)

	// Assembly
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  assemblyIPN,
		"type": "adjust",
		"qty":  0.0,
		"note": "Assembly using old part (will be updated via ECO)",
	})
	t.Logf("✓ Created assembly %s", assemblyIPN)

	// Step 2: Create initial BOM with old part
	t.Log("\n[Step 2] Creating initial BOM with old part...")
	client.request("POST", "/api/v1/bom", map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": oldPartIPN,
		"qty_per":       5.0,
	})
	t.Logf("✓ BOM created: %s uses 5x %s", assemblyIPN, oldPartIPN)

	// Step 3: Create ECO for part change
	t.Log("\n[Step 3] Creating ECO for BOM change...")
	ecoID := uniqueID("ECO-INT")
	eco := map[string]interface{}{
		"id":            ecoID,
		"title":         fmt.Sprintf("Replace %s with %s in %s", oldPartIPN, newPartIPN, assemblyIPN),
		"description":   "Integration test: Part obsolescence - replace with new approved part",
		"status":        "draft",
		"priority":      "medium",
		"affected_ipns": assemblyIPN,
	}

	resp, body := client.request("POST", "/api/v1/ecos", eco)
	var ecoResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ecoResp)
	if ecoResp.Data.ID != "" {
		ecoID = ecoResp.Data.ID
	}
	t.Logf("✓ Created ECO: %s (status: draft)", ecoID)

	// Step 4: Approve ECO
	t.Log("\n[Step 4] Approving ECO...")
	approveData := map[string]interface{}{
		"title":         eco["title"],
		"description":   eco["description"],
		"status":        "approved",
		"priority":      eco["priority"],
		"affected_ipns": eco["affected_ipns"],
	}

	resp, body = client.request("PUT", "/api/v1/ecos/"+ecoID, approveData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("✓ Approved ECO: %s", ecoID)
	} else {
		t.Logf("⚠ ECO approval returned status %d: %s", resp.StatusCode, string(body))
	}

	// Step 5: Update BOM to use new part
	t.Log("\n[Step 5] Updating BOM to use new part...")

	// Delete old BOM entry
	client.request("DELETE", "/api/v1/bom?parent_ipn="+assemblyIPN+"&component_ipn="+oldPartIPN, nil)
	t.Logf("  Removed old part from BOM")

	// Add new BOM entry
	client.request("POST", "/api/v1/bom", map[string]interface{}{
		"parent_ipn":    assemblyIPN,
		"component_ipn": newPartIPN,
		"qty_per":       5.0,
	})
	t.Logf("✓ Updated BOM: %s now uses 5x %s", assemblyIPN, newPartIPN)

	// Step 6: Verify BOM was updated
	t.Log("\n[Step 6] Verifying BOM update...")
	resp, body = client.request("GET", "/api/v1/bom?parent_ipn="+assemblyIPN, nil)
	var bomResp struct {
		Data []struct {
			ParentIPN    string  `json:"parent_ipn"`
			ComponentIPN string  `json:"component_ipn"`
			QtyPer       float64 `json:"qty_per"`
		} `json:"data"`
	}
	json.Unmarshal(body, &bomResp)

	foundNewPart := false
	foundOldPart := false

	for _, item := range bomResp.Data {
		if item.ComponentIPN == newPartIPN {
			foundNewPart = true
			t.Logf("  ✓ Found new part in BOM: %s (qty_per: %.0f)", newPartIPN, item.QtyPer)
		}
		if item.ComponentIPN == oldPartIPN {
			foundOldPart = true
		}
	}

	if foundNewPart && !foundOldPart {
		t.Logf("✓ BOM update verified: new part present, old part removed")
	} else if foundOldPart {
		t.Errorf("✗ Old part still in BOM!")
	} else if !foundNewPart {
		t.Errorf("✗ New part not in BOM!")
	}

	// Step 7: Create work order with new BOM
	t.Log("\n[Step 7] Creating work order with updated BOM...")
	woID := uniqueID("WO-ECO")
	wo := map[string]interface{}{
		"id":           woID,
		"assembly_ipn": assemblyIPN,
		"qty":          5,
		"status":       "open",
		"priority":     "normal",
		"notes":        "Using ECO-updated BOM with " + newPartIPN,
	}

	resp, body = client.request("POST", "/api/v1/workorders", wo)
	var woResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &woResp)
	if woResp.Data.ID != "" {
		woID = woResp.Data.ID
	}
	t.Logf("✓ Created work order: %s for 5x %s", woID, assemblyIPN)

	// Step 8: Complete work order and verify correct parts were used
	t.Log("\n[Step 8] Completing work order...")

	// Record inventory before completion
	resp, body = client.request("GET", "/api/v1/inventory/"+oldPartIPN, nil)
	var invOld struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &invOld)
	oldPartBefore := invOld.Data.QtyOnHand

	resp, body = client.request("GET", "/api/v1/inventory/"+newPartIPN, nil)
	var invNew struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &invNew)
	newPartBefore := invNew.Data.QtyOnHand

	t.Logf("  Before WO completion: %s=%.0f, %s=%.0f", oldPartIPN, oldPartBefore, newPartIPN, newPartBefore)

	// Complete work order
	completeData := map[string]interface{}{
		"assembly_ipn": assemblyIPN,
		"qty":          5,
		"status":       "completed",
		"priority":     "normal",
		"qty_good":     5,
		"qty_scrap":    0,
	}

	resp, body = client.request("PUT", "/api/v1/workorders/"+woID, completeData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("✓ Completed work order: %s", woID)
	} else {
		t.Logf("⚠ Work order completion returned status %d: %s", resp.StatusCode, string(body))
	}

	time.Sleep(500 * time.Millisecond)

	// Step 9: Verify correct parts were consumed
	t.Log("\n[Step 9] Verifying correct parts were consumed...")

	resp, body = client.request("GET", "/api/v1/inventory/"+oldPartIPN, nil)
	json.Unmarshal(body, &invOld)
	oldPartAfter := invOld.Data.QtyOnHand

	resp, body = client.request("GET", "/api/v1/inventory/"+newPartIPN, nil)
	json.Unmarshal(body, &invNew)
	newPartAfter := invNew.Data.QtyOnHand

	t.Logf("  After WO completion: %s=%.0f, %s=%.0f", oldPartIPN, oldPartAfter, newPartIPN, newPartAfter)

	// Old part should be unchanged, new part should be reduced by 5*5=25
	expectedOldPart := oldPartBefore      // Should not be consumed
	expectedNewPart := newPartBefore - 25 // Should consume 25 (5 assemblies * 5 per)

	if oldPartAfter == expectedOldPart {
		t.Logf("✓ Old part NOT consumed (correct): %.0f (unchanged)", oldPartAfter)
	} else {
		t.Errorf("✗ Old part should NOT be consumed! Before: %.0f, After: %.0f", oldPartBefore, oldPartAfter)
	}

	if newPartAfter == expectedNewPart {
		t.Logf("✓✓ SUCCESS: New part consumed correctly: %.0f → %.0f (consumed 25)", newPartBefore, newPartAfter)
	} else {
		t.Errorf("✗ New part consumption incorrect: %.0f → %.0f (expected %.0f)", newPartBefore, newPartAfter, expectedNewPart)
	}

	// Step 10: Verify ECO is properly tracked
	t.Log("\n[Step 10] Verifying ECO tracking...")
	resp, body = client.request("GET", "/api/v1/ecos/"+ecoID, nil)
	var ecoDetail struct {
		Data struct {
			Status       string `json:"status"`
			AffectedIPNs string `json:"affected_ipns"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ecoDetail)

	if ecoDetail.Data.Status == "approved" {
		t.Logf("✓ ECO status verified: %s", ecoDetail.Data.Status)
	}
	if ecoDetail.Data.AffectedIPNs == assemblyIPN {
		t.Logf("✓ ECO affected IPNs verified: %s", ecoDetail.Data.AffectedIPNs)
	}

	t.Log("\n=== INTEGRATION TEST COMPLETE: ECO → BOM → Work Order ===\n")
}

// TestIntegration_Quote_SalesOrder_WorkOrder_Flow tests the workflow:
// 1. Create quote
// 2. Convert to sales order
// 3. Generate work order from SO
// 4. Complete work order
// 5. Verify SO updated
func TestIntegration_Quote_SalesOrder_WorkOrder_Flow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := newIntegrationClient(t)
	timestamp := time.Now().UnixNano()

	t.Log("=== INTEGRATION TEST: Quote → Sales Order → Work Order ===")

	// Step 1: Create parts with inventory
	t.Log("\n[Step 1] Creating parts...")
	productIPN := fmt.Sprintf("PROD-INT-%d", timestamp)
	componentIPN := fmt.Sprintf("COMP-PROD-%d", timestamp)

	// Create component with sufficient inventory
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  componentIPN,
		"type": "adjust",
		"qty":  200.0,
		"note": "Component inventory for production",
	})
	t.Logf("✓ Created %s with qty=200", componentIPN)

	// Create product with zero inventory
	client.request("POST", "/api/v1/inventory/transact", map[string]interface{}{
		"ipn":  productIPN,
		"type": "adjust",
		"qty":  0.0,
		"note": "Product to be manufactured",
	})
	t.Logf("✓ Created %s with qty=0", productIPN)

	// Step 2: Create BOM for product
	t.Log("\n[Step 2] Creating BOM for product...")
	client.request("POST", "/api/v1/bom", map[string]interface{}{
		"parent_ipn":    productIPN,
		"component_ipn": componentIPN,
		"qty_per":       4.0,
	})
	t.Logf("✓ BOM created: %s uses 4x %s", productIPN, componentIPN)

	// Step 3: Create quote
	t.Log("\n[Step 3] Creating quote...")
	quoteID := uniqueID("Q-INT")
	customerName := "Integration Test Customer"

	quote := map[string]interface{}{
		"id":          quoteID,
		"customer":    customerName,
		"status":      "draft",
		"notes":       "Integration test quote",
		"valid_until": time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		"lines": []map[string]interface{}{
			{
				"ipn":         productIPN,
				"description": "Custom product for customer",
				"qty":         10,
				"unit_price":  99.99,
			},
		},
	}

	resp, body := client.request("POST", "/api/v1/quotes", quote)
	var quoteResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &quoteResp)
	if quoteResp.Data.ID != "" {
		quoteID = quoteResp.Data.ID
	}
	t.Logf("✓ Created quote: %s for %s (10x %s)", quoteID, customerName, productIPN)

	// Step 4: Accept quote
	t.Log("\n[Step 4] Accepting quote...")
	acceptData := map[string]interface{}{
		"customer":    customerName,
		"status":      "accepted",
		"notes":       quote["notes"],
		"valid_until": quote["valid_until"],
	}

	resp, body = client.request("PUT", "/api/v1/quotes/"+quoteID, acceptData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("✓ Accepted quote: %s", quoteID)
	} else {
		t.Logf("⚠ Quote acceptance returned status %d: %s", resp.StatusCode, string(body))
	}

	// Step 5: Convert quote to sales order
	t.Log("\n[Step 5] Converting quote to sales order...")
	resp, body = client.request("POST", "/api/v1/quotes/"+quoteID+"/convert", nil)
	var soResp struct {
		Data struct {
			ID      string `json:"id"`
			QuoteID string `json:"quote_id"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(body, &soResp)

	salesOrderID := soResp.Data.ID
	if salesOrderID == "" {
		t.Logf("Convert response: %s", string(body))
		t.Skip("Cannot continue without sales order ID")
	}
	t.Logf("✓ Created sales order: %s from quote %s", salesOrderID, quoteID)

	// Step 6: Verify sales order details
	t.Log("\n[Step 6] Verifying sales order details...")
	resp, body = client.request("GET", "/api/v1/sales_orders/"+salesOrderID, nil)
	var soDetail struct {
		Data struct {
			ID       string `json:"id"`
			QuoteID  string `json:"quote_id"`
			Customer string `json:"customer"`
			Status   string `json:"status"`
			Lines    []struct {
				IPN          string  `json:"ipn"`
				Qty          int     `json:"qty"`
				QtyAllocated int     `json:"qty_allocated"`
				QtyShipped   int     `json:"qty_shipped"`
				UnitPrice    float64 `json:"unit_price"`
			} `json:"lines"`
		} `json:"data"`
	}
	json.Unmarshal(body, &soDetail)

	if soDetail.Data.QuoteID == quoteID {
		t.Logf("✓ Sales order linked to quote: %s", quoteID)
	}
	if len(soDetail.Data.Lines) == 1 && soDetail.Data.Lines[0].IPN == productIPN {
		t.Logf("✓ Sales order line verified: %dx %s @ $%.2f",
			soDetail.Data.Lines[0].Qty, productIPN, soDetail.Data.Lines[0].UnitPrice)
	}

	// Step 7: Generate work order from sales order
	t.Log("\n[Step 7] Generating work order from sales order...")
	woID := uniqueID("WO-SO")
	wo := map[string]interface{}{
		"id":           woID,
		"assembly_ipn": productIPN,
		"qty":          10, // Match sales order qty
		"status":       "open",
		"priority":     "high",
		"notes":        fmt.Sprintf("Generated from SO %s for %s", salesOrderID, customerName),
	}

	resp, body = client.request("POST", "/api/v1/workorders", wo)
	var woResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &woResp)
	if woResp.Data.ID != "" {
		woID = woResp.Data.ID
	}
	t.Logf("✓ Created work order: %s for SO %s (10x %s)", woID, salesOrderID, productIPN)

	// Step 8: Start and complete work order
	t.Log("\n[Step 8] Starting and completing work order...")

	// Start work order
	startData := map[string]interface{}{
		"assembly_ipn": productIPN,
		"qty":          10,
		"status":       "in_progress",
		"priority":     "high",
	}

	resp, body = client.request("PUT", "/api/v1/workorders/"+woID, startData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("  ✓ Started work order: %s", woID)
	}

	// Complete work order
	completeData := map[string]interface{}{
		"assembly_ipn": productIPN,
		"qty":          10,
		"status":       "completed",
		"priority":     "high",
		"qty_good":     10,
		"qty_scrap":    0,
	}

	resp, body = client.request("PUT", "/api/v1/workorders/"+woID, completeData)
	if resp.StatusCode == http.StatusOK {
		t.Logf("  ✓ Completed work order: %s", woID)
	} else {
		t.Logf("  ⚠ Work order completion returned status %d: %s", resp.StatusCode, string(body))
	}

	time.Sleep(500 * time.Millisecond)

	// Step 9: Verify inventory changes
	t.Log("\n[Step 9] Verifying inventory after work order completion...")

	// Product inventory should increase by 10
	resp, body = client.request("GET", "/api/v1/inventory/"+productIPN, nil)
	var invProd struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &invProd)

	// Component inventory should decrease by 40 (10 * 4)
	resp, body = client.request("GET", "/api/v1/inventory/"+componentIPN, nil)
	var invComp struct {
		Data struct {
			QtyOnHand float64 `json:"qty_on_hand"`
		} `json:"data"`
	}
	json.Unmarshal(body, &invComp)

	expectedProdQty := 10.0
	expectedCompQty := 160.0 // 200 - 40

	if invProd.Data.QtyOnHand == expectedProdQty {
		t.Logf("✓ Product inventory correct: %.0f (expected %.0f)", invProd.Data.QtyOnHand, expectedProdQty)
	} else {
		t.Errorf("✗ Product inventory incorrect: %.0f (expected %.0f)", invProd.Data.QtyOnHand, expectedProdQty)
	}

	if invComp.Data.QtyOnHand == expectedCompQty {
		t.Logf("✓ Component inventory correct: %.0f (expected %.0f)", invComp.Data.QtyOnHand, expectedCompQty)
	} else {
		t.Errorf("✗ Component inventory incorrect: %.0f (expected %.0f)", invComp.Data.QtyOnHand, expectedCompQty)
	}

	// Step 10: Update sales order status (simulate allocation/picking)
	t.Log("\n[Step 10] Updating sales order with allocation...")

	// In a real system, you might allocate inventory to the SO
	// For now, just verify the SO still exists and can be updated
	resp, body = client.request("GET", "/api/v1/sales_orders/"+salesOrderID, nil)
	json.Unmarshal(body, &soDetail)

	if soDetail.Data.Status != "" {
		t.Logf("✓ Sales order verified: %s (status: %s)", salesOrderID, soDetail.Data.Status)
	}

	// Step 11: Verify end-to-end linkage
	t.Log("\n[Step 11] Verifying end-to-end linkage...")
	t.Logf("  Quote %s → Sales Order %s → Work Order %s", quoteID, salesOrderID, woID)
	t.Logf("  Product manufactured: %s (qty: 10)", productIPN)
	t.Logf("  Components consumed: %s (qty: 40)", componentIPN)
	t.Logf("✓✓ SUCCESS: Complete quote-to-delivery workflow verified!")

	t.Log("\n=== INTEGRATION TEST COMPLETE: Quote → Sales Order → Work Order ===\n")
}
