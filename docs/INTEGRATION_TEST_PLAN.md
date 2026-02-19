# ZRP Integration Test Plan

> Created: 2026-02-19  
> Purpose: Document critical cross-module workflows that need integration testing

## Executive Summary

ZRP has excellent unit test coverage (1,224 frontend tests + 40 backend test files, all passing). However, **integration tests for cross-module workflows are missing**. This document identifies the highest-priority integration flows that need testing and provides test cases for each.

## Priority Integration Flows

### 1. BOM Shortage → Procurement → PO → Receiving → Inventory (P0)

**Workflow:** Create WO for assembly → Check BOM shortages → Generate PO from shortages → Receive PO → Verify inventory updated

**Current State:** ⚠️ **FRAGILE** (WORKFLOW_GAPS.md #3.1)
- No end-to-end test exists
- Multiple handoff points with unclear behavior
- Unclear if receiving updates inventory automatically

**Test Cases Needed:**

#### TC-INT-001: Complete BOM-to-Procurement Flow
```
GIVEN an assembly ASY-001 with BOM requiring:
  - 10x RES-001 (resistor)
  - 5x CAP-001 (capacitor)
AND inventory has insufficient stock:
  - RES-001: 5 on hand (shortage: 50 for WO qty 10)
  - CAP-001: 2 on hand (shortage: 48 for WO qty 10)
  
WHEN:
  1. Create work order WO-001 for 10x ASY-001
  2. Call GET /api/v1/workorders/WO-001/bom-check
  3. Call POST /api/v1/workorders/WO-001/generate-po with vendor V-001
  4. Call POST /api/v1/pos/{po_id}/receive
  
THEN:
  - Step 2 returns shortages array with 2 items
  - Step 3 creates PO with 2 line items (50x RES-001, 48x CAP-001)
  - Step 4 succeeds (200 OK)
  - Inventory qty_on_hand increased:
    * RES-001: 5 → 55
    * CAP-001: 2 → 50
  - Step 2 (repeated) returns empty shortages array
```

#### TC-INT-002: Partial PO Receiving
```
GIVEN PO-001 with line item: 100x RES-001
  
WHEN:
  1. Receive 50 units via POST /api/v1/pos/PO-001/receive-partial {line_id: 1, qty: 50}
  2. Query inventory for RES-001
  3. Receive remaining 50 units
  4. Query PO status
  
THEN:
  - After step 1: inventory += 50, PO status = 'partial'
  - After step 3: inventory += 50 (total +100), PO status = 'received'
  
NOTES: Currently NOT supported (WORKFLOW_GAPS.md #3.4)
```

---

### 2. Work Order → Material Reservation → Completion → Inventory Update (P0)

**Workflow:** Create WO → Reserve materials → Build → Complete → Update inventory

**Current State:** ⚠️ **BROKEN** (WORKFLOW_GAPS.md #4.1, #4.5)
- Creating WO does NOT reserve inventory
- Completing WO does NOT update inventory
- No material kitting step

**Test Cases Needed:**

#### TC-INT-003: Material Reservation on WO Creation
```
GIVEN:
  - BOM: ASY-001 requires 10x RES-001
  - Inventory: RES-001 has qty_on_hand=100, qty_reserved=0
  
WHEN:
  1. Create WO-001 for 5x ASY-001 (requires 50x RES-001)
  2. Query inventory for RES-001
  
THEN:
  - qty_reserved = 50
  - qty_on_hand = 100 (unchanged)
  - Available qty (on_hand - reserved) = 50
  
CURRENT BEHAVIOR: qty_reserved stays 0 (KNOWN GAP #4.1)
```

#### TC-INT-004: Inventory Update on WO Completion
```
GIVEN:
  - WO-001 for 10x ASY-001, status='in_progress'
  - BOM: ASY-001 requires 10x RES-001, 5x CAP-001
  - Inventory before:
    * ASY-001: qty_on_hand=0
    * RES-001: qty_on_hand=100, qty_reserved=100
    * CAP-001: qty_on_hand=50, qty_reserved=50
  
WHEN:
  1. Update WO-001 status to 'completed'
  2. Query all three inventory records
  
THEN:
  - ASY-001: qty_on_hand = 10 (finished goods added)
  - RES-001: qty_on_hand = 0, qty_reserved = 0 (consumed 100)
  - CAP-001: qty_on_hand = 0, qty_reserved = 0 (consumed 50)
  
CURRENT BEHAVIOR: No inventory changes (KNOWN GAP #4.5)
```

#### TC-INT-005: WO Scrap/Yield Tracking
```
GIVEN WO-001 for qty=100, status='in_progress'
  
WHEN:
  1. Update WO-001 with qty_good=95, qty_scrap=5
  2. Complete WO-001
  3. Query inventory for assembly
  
THEN:
  - Inventory += 95 (not 100)
  - Scrap recorded in WO record
  
CURRENT BEHAVIOR: Unknown - need to verify qty_good/qty_scrap handling
```

---

### 3. NCR → ECO / CAPA Creation (P1)

**Workflow:** Detect defect → Create NCR → Generate ECO or CAPA → Implement fix

**Current State:** ⚠️ **URL-PARAM BASED** (WORKFLOW_GAPS.md #9.1, #5.1, #2.7)
- "Create ECO from NCR" navigates to `/ecos?from_ncr={id}`
- "Create CAPA from NCR" navigates to `/capas?from_ncr={id}`
- Frontend must parse query params and auto-populate forms
- No database relation between NCR and ECO/CAPA

**Test Cases Needed:**

#### TC-INT-006: NCR to ECO Linking
```
GIVEN NCR-001 with:
  - ipn = 'ASY-001'
  - defect_type = 'design'
  - corrective_action = 'Redesign required'
  
WHEN:
  1. User clicks "Create ECO from NCR" (navigates to /ecos?from_ncr=NCR-001)
  2. ECO list page loads
  3. User clicks "New ECO" button
  
THEN:
  - Create ECO dialog opens
  - Title pre-filled: "ECO for NCR-001"
  - Description pre-filled from NCR corrective_action
  - affected_ipns pre-filled: 'ASY-001'
  
CURRENT STATUS: ⚠️ Unclear if frontend implements query param parsing
RECOMMENDATION: Add source_ncr_id field to ECO table for proper relation
```

#### TC-INT-007: NCR to CAPA Workflow
```
GIVEN NCR-002 with severity='critical', defect_type='process'
  
WHEN:
  1. Navigate to /capas?from_ncr=NCR-002
  2. Create CAPA with title, description, corrective/preventive actions
  3. Save CAPA
  4. Navigate back to NCR-002 detail page
  
THEN:
  - CAPA created successfully
  - NCR-002 shows link to created CAPA (if relation exists)
  
CURRENT STATUS: ⚠️ No database relation - need to verify frontend behavior
```

---

### 4. Device → RMA → Repair → Return (P1)

**Workflow:** Device fails in field → Create RMA → Diagnose → Repair → Ship back → Update device

**Current State:** ⚠️ **URL-PARAM BASED** (WORKFLOW_GAPS.md #7.5, #7.4, #7.1)
- No automatic device status update
- No link from device to field reports
- URL-param based RMA creation

**Test Cases Needed:**

#### TC-INT-008: Device to RMA Creation
```
GIVEN Device SN-12345 with:
  - status = 'active'
  - firmware_version = '1.0.0'
  - customer = 'Acme Corp'
  
WHEN:
  1. Navigate to device detail page
  2. Click "Create RMA"
  3. RMA form opens (via /rmas?device=SN-12345)
  4. Fill remaining fields and submit
  5. Query device status
  
THEN:
  - RMA created with serial_number = SN-12345
  - Device status auto-updated to 'maintenance' or 'rma' (DESIRED)
  
CURRENT BEHAVIOR: Device status stays 'active' (KNOWN GAP #7.4)
```

#### TC-INT-009: RMA Workflow Complete Cycle
```
GIVEN RMA-001 for device SN-12345, status='open'
  
WHEN:
  1. Update RMA status: open → received
  2. Update RMA status: received → diagnosing
  3. Add diagnosis findings and resolution
  4. Update RMA status: diagnosing → repaired
  5. Update RMA status: repaired → shipped
  6. Update RMA status: shipped → closed
  7. Query device SN-12345
  
THEN:
  - All status transitions succeed
  - Device status returns to 'active'
  - Device history shows RMA event
  
CURRENT STATUS: Need to verify status transition logic and device update
```

---

### 5. Quote → Sales Order → Work Order → Shipment (P0 BLOCKER)

**Workflow:** Create quote → Accept → Generate order → Build → Ship → Invoice

**Current State:** 🔴 **BLOCKER** (WORKFLOW_GAPS.md #8.1, #8.3)
- NO sales order module exists
- NO invoicing module exists
- Quote acceptance is a dead end

**Test Cases Needed:**

#### TC-INT-010: Quote to Sales Order (BLOCKED)
```
GIVEN Quote Q-001 with:
  - customer = 'Acme Corp'
  - lines: [{ipn: 'ASY-001', qty: 10, unit_price: 100}]
  - status = 'draft'
  
WHEN:
  1. Update quote status to 'sent'
  2. Update quote status to 'accepted'
  3. Query for sales orders with quote_id = Q-001
  
THEN:
  - Sales order SO-001 created automatically
  - SO-001 has same line items as Q-001
  - SO-001 status = 'pending' or 'confirmed'
  
CURRENT STATUS: 🔴 /api/v1/salesorders endpoint does not exist
ACTION REQUIRED: Implement sales order module
```

#### TC-INT-011: Sales Order to Work Order Generation
```
GIVEN Sales Order SO-001 for 10x ASY-001 (assembly)
  
WHEN:
  1. Click "Generate Work Order" on SO-001
  2. System creates WO-001 for 10x ASY-001
  3. Complete WO-001
  4. Update SO status to 'fulfilled'
  
THEN:
  - WO-001 linked to SO-001
  - Inventory for ASY-001 incremented by 10
  - SO status reflects WO completion
  
CURRENT STATUS: ⚠️ No SO→WO integration exists
```

---

## Testing Strategy

### Phase 1: Document Current Behavior (THIS DOCUMENT)
- ✅ Identify critical workflows
- ✅ Document expected vs actual behavior
- ✅ Flag known gaps from WORKFLOW_GAPS.md

### Phase 2: Create Integration Test Scaffolding
- Create `handler_integration_bom_test.go` for procurement flow
- Create `handler_integration_quality_test.go` for NCR/ECO/CAPA
- Create `handler_integration_sales_test.go` for quote/order (blocked)
- Use existing test patterns (httptest, in-memory SQLite)

### Phase 3: Implement Tests for Working Flows
- Focus on flows that SHOULD work but lack test coverage
- Document gaps where implementation is missing
- Use `t.Skip()` for blocked tests with clear reasons

### Phase 4: Fix Gaps and Update Tests
- Address P0 gaps (sales orders, WO inventory updates)
- Address P1 gaps (material reservation, URL-param linking)
- Update tests from Skip → Pass as features are implemented

---

## Test Implementation Notes

### Database Setup Pattern
```go
func setupIntegrationTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    
    // Create minimal schema needed for the test
    // Use actual table definitions from db.go migrations
    
    return db
}
```

### HTTP Test Pattern
```go
func TestWorkflow(t *testing.T) {
    testDB := setupIntegrationTestDB(t)
    defer testDB.Close()
    db = testDB  // Set global db
    
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/endpoint", handleEndpoint)
    
    req := httptest.NewRequest("GET", "/api/v1/endpoint", nil)
    w := httptest.NewRecorder()
    mux.ServeHTTP(w, req)
    
    // Assert response
}
```

### Documenting Known Gaps
```go
if actualBehavior != expectedBehavior {
    t.Logf("⚠️  KNOWN GAP #X.Y: Brief description")
    t.Logf("   Expected: %v", expectedBehavior)
    t.Logf("   Actual: %v", actualBehavior)
    // Don't fail the test if gap is documented
}
```

---

## Metrics

| Workflow | Priority | Test Status | Implementation Status |
|----------|----------|-------------|----------------------|
| BOM → Procurement → PO | P0 | ⚠️ Partial | ⚠️ Fragile |
| WO Material Reservation | P0 | ❌ Missing | ❌ Not implemented |
| WO Completion → Inventory | P0 | ❌ Missing | ❌ Not implemented |
| NCR → ECO | P1 | ❌ Missing | ⚠️ URL-param based |
| NCR → CAPA | P1 | ❌ Missing | ⚠️ URL-param based |
| Device → RMA | P1 | ❌ Missing | ⚠️ No auto status update |
| Quote → Sales Order | P0 | 🔴 Blocked | 🔴 Not implemented |
| Sales Order → WO | P0 | 🔴 Blocked | 🔴 Not implemented |

**Legend:**
- ✅ Passing tests
- ⚠️ Partial or documented gaps
- ❌ No tests exist
- 🔴 Blocked by missing features

---

## Recommendations

### Immediate Actions (P0)
1. **Implement sales order module** — Blocker for quote acceptance workflow
2. **Fix WO completion inventory update** — Manufacturing workflow broken
3. **Implement material reservation** — Prevents double-allocation of components
4. **Add integration test suite** — Catch regressions across module boundaries

### Short-term (P1)
1. **Replace URL-param linking with database relations** — Add `source_ncr_id` to ECO/CAPA tables
2. **Auto-update device status on RMA creation** — Improve field service tracking
3. **Add partial PO receiving** — Support multi-shipment deliveries

### Long-term (P2+)
1. **Add invoicing module** — Complete quote-to-cash cycle
2. **Implement WO-to-WO dependencies** — Support complex build sequences
3. **Add serial number genealogy** — Full component traceability

---

## Appendix: Test Data Seeds

### Minimal BOM Test Data
```sql
-- Vendor
INSERT INTO vendors VALUES ('V-001', 'Test Vendor', 'active', 7);

-- Parts
INSERT INTO parts VALUES 
  ('ASY-001', 'assembly', 'production', 'Test Assembly'),
  ('RES-001', 'resistor', 'production', 'Test Resistor'),
  ('CAP-001', 'capacitor', 'production', 'Test Capacitor');

-- BOM
INSERT INTO boms (parent_ipn, child_ipn, quantity) VALUES
  ('ASY-001', 'RES-001', 10.0),
  ('ASY-001', 'CAP-001', 5.0);

-- Inventory (low stock to trigger shortages)
INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES
  ('RES-001', 5.0, 50.0),
  ('CAP-001', 2.0, 25.0),
  ('ASY-001', 0.0, 10.0);
```
