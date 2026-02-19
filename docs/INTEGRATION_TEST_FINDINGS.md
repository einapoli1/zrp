# Integration Test Findings & Workflow Gap Report

**Date:** 2026-02-19  
**Scope:** Critical cross-module workflow integration tests  
**Test Files:** `integration_bom_po_test.go`

## Executive Summary

Created comprehensive integration tests for ZRP's two most critical business workflows:
1. **BOM Shortage → PO → Inventory** (TC-INT-001)
2. **Work Order Completion → Inventory** (TC-INT-002)

**Result:** Both tests **FAILED**, confirming **TWO CRITICAL P0 WORKFLOW GAPS** that prevent ZRP from functioning correctly in production manufacturing environments.

---

## Test Results

### ✗ TC-INT-001: PO Receipt Does NOT Update Inventory

**Test:** `TestIntegration_PO_Receipt_Updates_Inventory`

**Expected Workflow:**
1. Create PO with line items (RES-INT-001 qty=95, CAP-INT-001 qty=48)
2. Mark PO as received (UPDATE status='received')
3. Inventory automatically updates (qty_on_hand increases by received qty)
4. Inventory transactions created for audit trail

**Actual Behavior:**
- ✗ PO marked as received successfully
- ✗ Inventory qty_on_hand **UNCHANGED** (remained at initial values)
- ✗ NO inventory_transactions created

**Impact:**
- **BLOCKER** for procurement operations
- Received parts don't appear in inventory
- Manufacturing cannot consume materials that were ordered
- Manual inventory adjustments required for every PO (error-prone)

**Root Cause:**
Code in `handler_procurement.go` has inventory update logic but it's conditional on `SkipInspection` flag:
```go
if body.SkipInspection {
    // Only this path updates inventory
    db.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", l.Qty, now, ipn)
} else {
    // Default path creates receiving_inspection record but does NOT update inventory
    db.Exec(`INSERT INTO receiving_inspections (...)`)
}
```

The default workflow creates `receiving_inspections` records and expects a separate inspection completion step to update inventory. However, there is **no automatic completion** of inspections, leaving inventory in a permanently stale state.

**Required Fix:**
```go
// Option 1: Default to SkipInspection=true for direct inventory update
// Option 2: Add auto-complete inspection after PO receipt if no inspection config
// Option 3: Always update inventory on PO receipt, separate inspection tracking

// Recommended: Option 1 for simplicity
if body.SkipInspection || !hasInspectionWorkflow {
    db.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", ipn)
    db.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", l.Qty, now, ipn)
    db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,created_at) VALUES (?,?,?,?,?)", 
        ipn, "receive", l.Qty, id, now)
}
```

---

### ✗ TC-INT-002: Work Order Completion Does NOT Update Inventory

**Test:** `TestIntegration_WorkOrder_Completion_Updates_Inventory`

**Expected Workflow:**
1. Create WO for 10x ASY-WO-001 (status='open')
2. Mark WO as completed (UPDATE status='completed')
3. Finished goods automatically added to inventory (qty_on_hand += 10)
4. Component materials consumed (optional but expected)
5. Inventory transactions created

**Actual Behavior:**
- ✓ WO status updated to 'completed' successfully
- ✗ Assembly inventory **UNCHANGED** (qty_on_hand remained 0)
- ✗ NO inventory_transactions created
- ✗ Component materials NOT consumed

**Impact:**
- **BLOCKER** for manufacturing operations
- Finished goods don't appear in finished goods inventory
- Cannot ship products that were built
- Inventory accuracy completely broken
- Raw material consumption not tracked (ghost inventory)

**Root Cause:**
Work order completion handler (`handler_workorders.go`) only updates the `status` field:
```go
func handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
    var body map[string]interface{}
    json.NewDecoder(r.Body).Decode(&body)
    
    if status, ok := body["status"].(string); ok {
        db.Exec("UPDATE work_orders SET status=? WHERE id=?", status, id)
    }
    
    // NO inventory update logic exists
}
```

There is **no code** to update inventory when a work order is completed. This is a complete missing feature.

**Required Fix:**
```go
func handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
    var body map[string]interface{}
    json.NewDecoder(r.Body).Decode(&body)
    
    if status, ok := body["status"].(string); ok {
        // Update status
        db.Exec("UPDATE work_orders SET status=?,completed_at=? WHERE id=?", status, time.Now(), id)
        
        // NEW: If completing WO, update inventory
        if status == "completed" {
            var wo WorkOrder
            db.QueryRow("SELECT ipn, qty FROM work_orders WHERE id=?", id).Scan(&wo.IPN, &wo.Qty)
            
            // Add finished goods to inventory
            db.Exec("INSERT OR IGNORE INTO inventory (ipn,qty_on_hand) VALUES (?,0)", wo.IPN)
            db.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+? WHERE ipn=?", wo.Qty, wo.IPN)
            
            // Create transaction for audit trail
            db.Exec(`INSERT INTO inventory_transactions (ipn,type,qty,reference,created_at,created_by)
                VALUES (?,?,?,?,?,?)`, wo.IPN, "receive", wo.Qty, id, time.Now(), getUsername(r))
            
            // TODO: Consume component materials from BOM
            // This requires BOM lookup and may need gitplm integration
        }
    }
    
    logAudit(db, getUsername(r), "updated", "work_order", id, fmt.Sprintf("Status: %s", status))
    handleGetWorkOrder(w, r, id)
}
```

---

## Priority & Severity

**Both gaps are P0 BLOCKERS for production use:**

| Gap | Priority | Severity | Business Impact |
|-----|----------|----------|-----------------|
| PO Receipt → Inventory | **P0** | Critical | Cannot restock inventory; procurement workflow broken |
| WO Completion → Inventory | **P0** | Critical | Cannot track finished goods; manufacturing workflow broken |

**Risk if not fixed:**
- Manual inventory adjustments required for EVERY transaction
- High error rate (typos, missed entries)
- No audit trail of inventory movements
- Inventory counts permanently inaccurate
- Cannot ship orders (inventory shows 0 even after production)
- Cannot detect shortages (BOM checks see wrong inventory levels)

---

## Test Coverage Added

### New Test File: `integration_bom_po_test.go`

**Test Functions:**
1. `TestIntegration_PO_Receipt_Updates_Inventory` (184 lines)
   - Creates vendor, inventory, PO with line items
   - Simulates PO receipt
   - Verifies inventory updated and transactions created
   - **Status:** FAILING (as expected - workflow gap confirmed)

2. `TestIntegration_WorkOrder_Completion_Updates_Inventory` (102 lines)
   - Creates component and assembly inventory
   - Creates and completes work order
   - Verifies finished goods added and transactions created
   - **Status:** FAILING (as expected - workflow gap confirmed)

**Test Infrastructure:**
- `setupIntegrationTestDB()` - Creates in-memory SQLite test database
- `createIntegrationTestSession()` - Creates authenticated session
- Uses actual database schema (matching `db.go` migrations)
- Isolated tests (no shared state, each test gets fresh DB)

**Running Tests:**
```bash
# Run integration tests (not in short mode)
go test -v -run TestIntegration_

# Skip integration tests in short mode
go test -short -run TestIntegration_  # No tests run

# All tests
go test ./...
```

---

## Recommendations

### Immediate Actions (Required for Production)

1. **Implement PO → Inventory Update** (Est: 1 hour)
   - Modify `handler_procurement.go` receive endpoint
   - Default to direct inventory update
   - Create inventory_transactions
   - Add transaction rollback on error

2. **Implement WO Completion → Inventory Update** (Est: 2 hours)
   - Modify `handler_workorders.go` update endpoint
   - Add finished goods to inventory on status='completed'
   - Create inventory_transactions
   - (Optional) Deduct component materials based on BOM

3. **Verify Fixes with Integration Tests** (Est: 15 minutes)
   - Run `go test -v -run TestIntegration_`
   - Both tests should PASS after fixes

4. **Add E2E Tests** (Est: 3 hours)
   - Playwright test: Create PO → Receive → Check inventory page
   - Playwright test: Create WO → Complete → Check inventory page

### Future Enhancements

1. **BOM-Based Material Consumption**
   - When WO completes, deduct components from inventory
   - Requires gitplm BOM integration
   - Prevents ghost inventory (shows components that were used)

2. **Inventory Reservations**
   - When WO is created, reserve materials (qty_reserved)
   - Prevents selling components needed for production
   - Release reservations on WO cancel/complete

3. **Receiving Inspection Workflow**
   - Optional quality gate between PO receipt and inventory
   - Reject/accept logic with qty_passed and qty_rejected
   - Failed inspections create NCRs automatically

4. **Work Order Material Kitting**
   - Explicit "kit materials" step before WO start
   - Moves materials from warehouse to production floor
   - Tracks material location throughout production

---

## Conclusion

✅ **Integration tests successfully identified TWO CRITICAL workflow gaps**  
✗ **Both workflows are currently BROKEN in production**  
🔧 **Fixes are straightforward and well-defined**  
⏱️ **Estimated fix time: 3 hours total**

These integration tests provide:
- **Regression protection** - Prevents these gaps from recurring
- **Documentation** - Clear specification of expected behavior
- **Confidence** - Automated verification that fixes work correctly

**Next Steps:**
1. Implement fixes for both workflow gaps
2. Verify with `go test -run TestIntegration_` (both should PASS)
3. Add to CI/CD pipeline
4. Document in CHANGELOG.md
