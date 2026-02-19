# Missing E2E Test Coverage - Advanced Features Audit

**Date**: 2026-02-19  
**Scope**: Advanced/Admin Features (Quotes, RMAs, NCRs, CAPAs, Reports, Users, Permissions, API Keys, Bulk Operations)  
**Current Coverage**: Basic smoke tests (navigation only) exist for these modules  
**Gap**: No comprehensive workflow tests for any advanced features

---

## Summary

**Current State**: 
- ✅ Smoke tests exist for navigation to: `/quotes`, `/rmas`, `/ncrs`, `/users`, `/api-keys`, `/reports`
- ✅ Integration tests for basic workflows: Work Orders, Procurement, Inventory
- ❌ **NO workflow tests** for any advanced/admin features

**Priority Breakdown**:
- **P0 (Critical)**: 18 missing tests - core business workflows
- **P1 (Important)**: 15 missing tests - important features
- **P2 (Nice-to-have)**: 8 missing tests - edge cases and optimizations

---

## Advanced Features Audit

### 1. Quotes Module

**Current**: Navigation test only (`smoke.spec.ts`)  
**Missing Coverage**:

- ❌ **[P0] Create quote from scratch**
  - Fill customer name, notes, valid until date
  - Add multiple line items (IPN, description, qty, unit price)
  - Verify total calculation
  - Save and verify quote appears in list

- ❌ **[P0] Generate PDF quote**
  - Create quote with line items
  - Click "Export PDF" button
  - Verify PDF download with correct filename format

- ❌ **[P1] Edit existing quote**
  - Navigate to quote detail
  - Enable editing mode
  - Modify customer, notes, line items
  - Save and verify changes persist

- ❌ **[P1] Quote status workflow**
  - Create draft quote
  - Change status: draft → sent → accepted
  - Verify status badges update correctly

- ❌ **[P1] Delete line item from quote**
  - Open quote with multiple lines
  - Remove middle line item
  - Verify total recalculates correctly

- ❌ **[P2] Quote expiration validation**
  - Create quote with past expiration date
  - Verify status shows as "expired"

- ❌ **[P2] Convert quote to sales order**
  - Accept quote
  - Click "Convert to Order" (if feature exists)
  - Verify order created with same line items

---

### 2. RMAs (Return Merchandise Authorization)

**Current**: Navigation test only (`smoke.spec.ts`)  
**Missing Coverage**:

- ❌ **[P0] Create RMA**
  - Fill serial number, customer, reason, defect description
  - Submit form
  - Verify RMA appears in list with "open" status

- ❌ **[P0] RMA status workflow progression**
  - Create RMA (status: open)
  - Update to "received"
  - Update to "investigating"
  - Update to "resolved"
  - Update to "shipped"
  - Final status: "closed"
  - Verify status badges and icons change correctly

- ❌ **[P1] Edit RMA details**
  - Navigate to RMA detail page
  - Enable editing
  - Update defect description, resolution notes
  - Save and verify changes

- ❌ **[P1] Link RMA to device/serial number**
  - Verify serial number lookup/validation
  - Display device history if linked

- ❌ **[P2] RMA filtering by status**
  - Filter RMAs by open/closed/shipped
  - Verify correct results displayed

---

### 3. NCRs (Non-Conformance Reports)

**Current**: Navigation test only (`smoke.spec.ts`)  
**Missing Coverage**:

- ❌ **[P0] Create NCR**
  - Fill title, description, IPN
  - Select severity: critical/major/minor
  - Submit and verify NCR created

- ❌ **[P0] Severity badge display**
  - Verify critical NCRs show red badge
  - Verify major NCRs show orange badge
  - Verify minor NCRs show yellow badge

- ❌ **[P1] Link NCR to part (IPN)**
  - Create NCR with specific IPN
  - Navigate to NCR detail
  - Verify IPN displays and links correctly

- ❌ **[P1] Create CAPA from NCR**
  - Navigate to NCR detail
  - Click "Create CAPA" button
  - Verify redirect to CAPA creation with pre-filled NCR link

- ❌ **[P2] NCR status progression**
  - Update NCR status: open → investigating → resolved
  - Verify status tracking

---

### 4. CAPAs (Corrective and Preventive Actions)

**Current**: Navigation test only (via smoke tests)  
**Missing Coverage**:

- ❌ **[P0] Create CAPA manually**
  - Fill title, type (corrective/preventive)
  - Add root cause, action plan, owner, due date
  - Submit and verify creation

- ❌ **[P0] Create CAPA from NCR (workflow integration)**
  - Start from NCR detail page
  - Click "Create CAPA"
  - Verify CAPA form pre-populated with NCR ID and title
  - Submit and verify link back to NCR

- ❌ **[P1] CAPA status workflow**
  - Progress through: open → in-progress → verification → closed
  - Verify status badges update

- ❌ **[P1] Link CAPA to RMA**
  - Create CAPA with RMA ID
  - Verify link displays on CAPA detail

- ❌ **[P1] CAPA dashboard metrics**
  - Create multiple CAPAs with different statuses
  - Verify dashboard shows correct counts

- ❌ **[P2] Overdue CAPA detection**
  - Create CAPA with past due date
  - Verify overdue indicator shows

---

### 5. Reports Module

**Current**: Navigation test only (`smoke.spec.ts`)  
**Missing Coverage**:

- ❌ **[P0] Generate Inventory Valuation report**
  - Click report card
  - Verify report data loads
  - Verify table shows: category, location, parts count, total value
  - Verify summary total displays

- ❌ **[P1] Generate Low Stock Alert report**
  - Click report card
  - Verify parts below minimum displayed
  - Verify alert indicators

- ❌ **[P1] Generate Vendor Performance report**
  - Generate report
  - Verify on-time delivery metrics display

- ❌ **[P1] Export report to CSV/PDF**
  - Generate any report
  - Click export button
  - Verify file downloads

- ❌ **[P2] Generate User Activity report**
  - Verify login frequency and module usage data

- ❌ **[P2] Filter reports by date range**
  - Apply date filter to time-based reports
  - Verify correct data returned

---

### 6. User Management

**Current**: Navigation test only (`smoke.spec.ts`)  
**Missing Coverage**:

- ❌ **[P0] Create new user**
  - Fill username, email, password
  - Select role: admin/user/readonly
  - Submit and verify user created
  - Verify user appears in list with correct role badge

- ❌ **[P0] Edit user role**
  - Select existing user
  - Change role from "user" to "admin"
  - Save and verify role updated

- ❌ **[P0] Deactivate user account**
  - Edit user
  - Change status to "inactive"
  - Verify status badge changes to inactive

- ❌ **[P1] Reactivate inactive user**
  - Select inactive user
  - Change status to "active"
  - Verify user can log in again

- ❌ **[P1] User role badge display**
  - Verify admin shows red shield badge
  - Verify user shows blue badge
  - Verify readonly shows gray eye badge

- ❌ **[P2] Display last login timestamp**
  - Verify relative time display ("2 days ago")

- ❌ **[P2] Prevent self-deactivation**
  - Logged in user tries to deactivate own account
  - Verify prevented or warning shown

---

### 7. Permissions (Role-Based Access Control)

**Current**: Basic unit test exists (`Permissions.test.tsx`)  
**Missing Coverage**:

- ❌ **[P0] Verify admin has full permissions**
  - Select "admin" role
  - Verify all modules show all actions checked

- ❌ **[P0] Verify readonly has view-only permissions**
  - Select "readonly" role
  - Verify only "view" actions checked
  - Verify create/edit/delete unchecked

- ❌ **[P0] Modify role permissions**
  - Select "user" role
  - Toggle specific permission (e.g., parts:delete)
  - Save changes
  - Reload and verify change persisted

- ❌ **[P1] Toggle all permissions for module**
  - Click module header checkbox
  - Verify all actions toggle on/off together

- ❌ **[P1] Verify permission changes apply to user sessions**
  - Change user role permissions
  - Log in as that user
  - Verify UI reflects new permissions (buttons disabled, etc.)

- ❌ **[P2] Reset permissions to default**
  - Modify permissions
  - Click reset/restore defaults
  - Verify original state restored

---

### 8. API Keys Management

**Current**: Navigation test only (`smoke.spec.ts`)  
**Missing Coverage**:

- ❌ **[P0] Generate new API key**
  - Click "Create API Key"
  - Enter key name/description
  - Submit and verify key generated
  - Verify full key displayed (one-time view)

- ❌ **[P0] Copy API key to clipboard**
  - Generate key
  - Click copy button
  - Verify toast notification
  - Verify clipboard contains key

- ❌ **[P0] Revoke API key**
  - Select active key
  - Click revoke/delete
  - Confirm action
  - Verify status changes to "revoked"

- ❌ **[P1] API key prefix masking**
  - After initial creation, navigate away and return
  - Verify full key no longer visible
  - Verify only prefix shown (e.g., "zrp_abc123...")

- ❌ **[P1] Last used timestamp tracking**
  - Use API key in request
  - Reload API keys page
  - Verify "last used" timestamp updated

- ❌ **[P2] Prevent duplicate key names**
  - Create key with name "Production API"
  - Try to create another with same name
  - Verify error or warning

---

### 9. Bulk Operations

**Current**: No tests found  
**Missing Coverage**:

- ❌ **[P0] Bulk edit inventory items**
  - Select multiple inventory items via checkboxes
  - Click "Bulk Edit" button
  - Update common field (e.g., location)
  - Apply changes and verify all items updated

- ❌ **[P1] Bulk update pricing**
  - Select multiple parts in Pricing module
  - Apply percentage markup/discount
  - Verify calculations correct

- ❌ **[P2] Bulk delete with confirmation**
  - Select multiple items
  - Click delete
  - Verify confirmation dialog shows count
  - Confirm and verify deletion

- ❌ **[P2] Barcode scan for inventory**
  - Open barcode scanner dialog
  - Simulate barcode scan
  - Verify inventory item found and updated

---

## Integration Test Gaps

These advanced features also need integration testing across modules:

- ❌ **[P0] NCR → CAPA → Close workflow**
  - Create NCR (critical severity)
  - Create CAPA from NCR
  - Complete CAPA action plan
  - Close CAPA
  - Verify NCR status updates

- ❌ **[P0] Quote → Sales Order → Shipment**
  - Create and accept quote
  - Convert to sales order
  - Fulfill and ship
  - Track end-to-end

- ❌ **[P1] RMA → NCR → CAPA chain**
  - Create RMA for defective device
  - Create NCR from RMA
  - Create CAPA from NCR
  - Verify all linked correctly

- ❌ **[P1] Permission enforcement across UI**
  - Set user to readonly
  - Verify create/edit buttons hidden throughout app

---

## Test Infrastructure Improvements Needed

To support advanced feature testing:

1. **Test data factories** for:
   - Quotes with line items
   - RMAs with status history
   - NCRs with linked parts
   - CAPAs with linked NCRs/RMAs
   - Users with different roles
   - API keys

2. **Authentication contexts**:
   - Test as admin user
   - Test as readonly user
   - Test as standard user

3. **API mocking/seeding**:
   - Pre-populate test database with baseline data
   - Mock PDF generation endpoints
   - Mock report data endpoints

4. **Snapshot testing** for:
   - Permission matrices
   - Report outputs
   - PDF export structures

---

## Priority Recommendations

**Immediate (Sprint 1)**:
1. User Management CRUD (P0)
2. Permissions verification (P0)
3. API Key generation/revocation (P0)
4. Quote creation and PDF export (P0)
5. RMA status workflow (P0)

**Short-term (Sprint 2)**:
1. NCR → CAPA integration workflow (P0)
2. Reports generation (P0)
3. Bulk operations (P0)
4. Quote/RMA editing (P1)

**Medium-term (Sprint 3)**:
1. All P1 items
2. Cross-module integration tests
3. Permission enforcement UI verification

**Long-term (Sprint 4+)**:
1. All P2 items
2. Edge case coverage
3. Performance tests for bulk operations

---

## Test File Structure Recommendation

```
frontend/e2e/
├── advanced/
│   ├── quotes.spec.ts          # Quote CRUD, PDF export
│   ├── quote-workflow.spec.ts  # Status changes, conversions
│   ├── rmas.spec.ts            # RMA CRUD
│   ├── rma-workflow.spec.ts    # Status progression
│   ├── ncrs.spec.ts            # NCR creation, severity
│   ├── capas.spec.ts           # CAPA workflows
│   ├── ncr-capa-flow.spec.ts   # NCR → CAPA integration
│   ├── reports.spec.ts         # Report generation
│   ├── users.spec.ts           # User management
│   ├── permissions.spec.ts     # RBAC verification
│   ├── api-keys.spec.ts        # API key lifecycle
│   └── bulk-operations.spec.ts # Bulk editing
└── integration/
    ├── tc-int-004-quote-to-order.spec.ts
    ├── tc-int-005-rma-ncr-capa.spec.ts
    └── tc-int-006-permission-enforcement.spec.ts
```

---

## Estimated Coverage Impact

**Before**: ~15% e2e coverage of advanced features (navigation only)  
**After implementing P0 tests**: ~60% coverage  
**After implementing P0 + P1 tests**: ~85% coverage  
**After implementing all tests**: ~95% coverage

---

**Report compiled by**: Eva (AI Assistant)  
**Next Action**: Prioritize P0 tests and create test implementation tickets

---
---

# Error Handling & Edge Cases Audit

**Date:** 2026-02-19  
**Scope:** Error scenarios, validation, edge cases, negative testing  
**Auditor:** Eva (Subagent)

## Executive Summary

Reviewed all e2e tests in `/tests/` and backend handler tests. Current coverage is **basic** for happy paths and **partial** for error scenarios. Critical gaps exist around:
- Inventory constraints during WO operations
- Dependency-based deletion failures
- Network/timeout failures
- Concurrent edit conflicts
- UI-level form validation feedback

---

## ✅ Already Covered

**Permissions** (permissions.spec.js):
- ✅ Readonly user blocked from POST/PUT/DELETE on all modules
- ✅ Unauthenticated API access returns 401

**Basic Validation** (validation.spec.js):
- ✅ Create records with missing required fields (ECO title, WO assembly_ipn, etc.)
- ✅ Invalid JSON body handling
- ✅ Nonexistent record GET/PUT/DELETE

**Duplicate Prevention** (handler_parts_create_test.go):
- ✅ Duplicate IPN in same category
- ✅ Duplicate IPN across categories
- ✅ Duplicate category prefix

**Delete Dependencies** (handler_vendors.go):
- ✅ Cannot delete vendor with active POs
- ✅ Cannot delete vendor with active RFQs

**Edge Input** (edge-cases.spec.js):
- ✅ Very long strings (5000 chars)
- ✅ Special characters/XSS attempts
- ✅ SQL injection attempts
- ✅ Unicode in names
- ✅ Empty search queries

**Concurrency** (edge-cases.spec.js):
- ✅ Rapid navigation doesn't crash
- ✅ Multiple parallel API calls succeed

---

## ❌ CRITICAL GAPS - Must Test

### Form Validation (UI-level)

**Parts:**
- ❌ Create part with missing IPN in UI modal → should show inline error
- ❌ Create part with invalid IPN format (special chars) → validation message
- ❌ Create part with negative/zero reorder point → reject or warn
- ❌ Upload invalid CSV format → clear error message
- ❌ Upload CSV with duplicate IPNs → report which rows failed

**ECOs:**
- ❌ Create ECO without title in UI form → validation error shown
- ❌ Approve ECO without required approvals → blocked with message
- ❌ Implement ECO without parts list → validation or warning

**Work Orders:**
- ❌ Create WO with missing assembly IPN → validation error
- ❌ Create WO with zero quantity → reject with message
- ❌ Create WO with negative quantity → reject with message
- ❌ Create WO for non-existent assembly IPN → error message

**Purchase Orders:**
- ❌ Create PO without vendor → validation error
- ❌ Create PO with negative unit price → reject
- ❌ Create PO with zero quantity → reject
- ❌ Add PO line with invalid IPN → error message

**Vendors:**
- ❌ Create vendor without name → validation error
- ❌ Create vendor with invalid website URL → format validation
- ❌ Create vendor with invalid email → format validation

**Users:**
- ❌ Create user without username → validation error
- ❌ Create user without password → validation error
- ❌ Create user with weak password → strength validation
- ❌ Create user with duplicate username → 409 conflict message
- ❌ Update user to invalid role → validation error

**NCR:**
- ❌ Create NCR without severity → validation error
- ❌ Create NCR without description → validation or warning

**RMA:**
- ❌ Create RMA without serial number → validation error
- ❌ Create RMA without customer → validation error
- ❌ Create RMA with non-existent serial → error message

**Devices:**
- ❌ Create device without serial number → validation error
- ❌ Create device with duplicate serial → 409 conflict
- ❌ Upload firmware without version number → validation error

**Inventory:**
- ❌ Adjust inventory with negative resulting quantity → reject
- ❌ Adjust inventory with invalid transaction type → error

---

### Delete Dependencies (Foreign Key Constraints)

**Parts:**
- ❌ **CRITICAL**: Delete part used in active BOM → 409 with clear message listing dependencies
- ❌ **CRITICAL**: Delete part used in open work order → 409 conflict
- ❌ **CRITICAL**: Delete part with open PO lines → 409 conflict
- ❌ Delete part with inventory transactions → 409 or cascade warning

**Work Orders:**
- ❌ Delete WO with reserved inventory → unreserve or block
- ❌ Delete WO with linked serials → cascade or block

**Purchase Orders:**
- ❌ Delete PO that's already received → block with message
- ❌ Delete PO with partial receipts → block or warn

**Vendors:**
- ✅ Already tested: vendor with POs/RFQs

**ECOs:**
- ❌ Delete ECO in 'approved' state → block or require confirmation
- ❌ Delete ECO with linked parts changes → cascade warning

**Categories:**
- ❌ Delete category with existing parts → 409 conflict

**Users:**
- ❌ Delete user with audit log entries → block or anonymize
- ❌ Delete last admin user → must prevent

---

### Inventory & Material Constraints

**Work Order Operations:**
- ❌ **CRITICAL**: Complete work order with insufficient inventory → error with shortage details
- ❌ **CRITICAL**: Kit work order when parts unavailable → shortage report shown
- ❌ **CRITICAL**: Start WO that would exceed available inventory → blocked
- ❌ Reserve more inventory than available → error message
- ❌ Complete WO with partial inventory → scrap handling or error

**Inventory Adjustments:**
- ❌ Withdraw more than qty_on_hand → reject with available qty
- ❌ Adjust to negative inventory → validation error
- ❌ Transfer inventory to non-existent location → error

**Sales Orders:**
- ❌ Ship sales order with insufficient inventory → blocked (already handled in code, needs e2e test)
- ❌ Create SO line for part not in inventory → warning or error

**Purchase Order Receiving:**
- ❌ Receive more than ordered → warning or accept with note
- ❌ Receive with invalid serial numbers → validation error
- ❌ Receive into non-existent location → error or create

---

### Permission Denials (Extended)

**Readonly User:**
- ✅ Already covered: cannot POST/PUT/DELETE
- ❌ Readonly cannot approve ECO → 403
- ❌ Readonly cannot kit work order → 403
- ❌ Readonly cannot adjust inventory → 403
- ❌ Readonly cannot receive PO → 403

**Regular User (non-admin):**
- ❌ Regular user cannot create users → 403
- ❌ Regular user cannot delete users → 403
- ❌ Regular user cannot access audit log → 403
- ❌ Regular user cannot create API keys → 403
- ❌ Regular user cannot modify system settings → 403

**QE Role:**
- ❌ Non-QE cannot approve CAPA as QE → 403 (code exists, needs e2e)
- ❌ Non-QE cannot close NCR without approval → 403

**Manager Role:**
- ❌ Non-manager cannot approve CAPA as manager → 403 (code exists, needs e2e)

---

### Network & Timeout Failures

**Network Errors:**
- ❌ API call fails mid-request → graceful error shown
- ❌ Offline mode → show offline indicator and queue/block operations
- ❌ Slow network (timeout) → loading indicator, eventual timeout message
- ❌ 500 server error → user-friendly error message (not stack trace)
- ❌ 502/503 gateway errors → retry or clear message

**WebSocket:**
- ❌ WebSocket disconnect → reconnect automatically
- ❌ WebSocket message delivery failure → graceful degradation

**File Uploads:**
- ❌ Upload interrupted → clear error and retry option
- ❌ Upload file too large → size validation message
- ❌ Upload unsupported file type → format validation

---

### Empty States & Zero Data

**Lists:**
- ✅ Partially covered: empty state rendering in list-features.spec.js
- ❌ Each module shows helpful empty state with CTA (test all modules)
- ❌ Search with no results → "No results found" message
- ❌ Filter that returns zero items → clear empty state

**Reports:**
- ❌ Generate report with no data → empty report or message
- ❌ Export CSV with zero rows → valid CSV with headers only

**Dashboard:**
- ❌ Dashboard with no ECOs/WOs/NCRs → all KPIs show zero gracefully

---

### Concurrent Edits & Race Conditions

**Edit Conflicts:**
- ❌ **CRITICAL**: Two users edit same ECO → last-write-wins or conflict detection
- ❌ Two users edit same part → conflict warning
- ❌ Two users approve same ECO simultaneously → only one succeeds
- ❌ User edits record deleted by another user → 404 with clear message

**Inventory Race:**
- ❌ Two users adjust same inventory item simultaneously → atomic transaction or lock
- ❌ WO kitting while inventory adjusted → consistent state

**Session Conflicts:**
- ❌ User logged in on multiple devices → both sessions valid or conflict
- ❌ Password changed while user is active → session invalidation

---

### Data Format & Type Validation

**Email Fields:**
- ❌ Invalid email format in vendor contact → validation error
- ❌ Invalid email format in user creation → validation error
- ❌ Email settings with malformed SMTP config → clear validation

**Date Fields:**
- ❌ Invalid date format → validation error
- ❌ Date in past when future required (expected delivery) → warning
- ❌ End date before start date → validation error

**Number Fields:**
- ❌ Non-numeric input in quantity field → validation error
- ❌ Negative prices → validation error
- ❌ Prices with > 2 decimal places → round or reject
- ❌ Extremely large numbers (overflow) → validation

**Text Fields:**
- ❌ Script tags in text inputs → sanitize or escape
- ❌ Null bytes in strings → reject
- ❌ Emoji in fields that don't support it → handle gracefully

---

### Module-Specific Edge Cases

**BOM:**
- ❌ Create BOM with circular dependency → detect and reject
- ❌ Create BOM with non-existent child part → validation error
- ❌ BOM with zero quantity per assembly → reject

**ECO Workflow:**
- ❌ Approve already-approved ECO → idempotent or error
- ❌ Implement already-implemented ECO → idempotent or error
- ❌ Cancel ECO with active work → require confirmation

**Receiving:**
- ❌ Receive PO that's already fully received → error or warning
- ❌ Receive into wrong PO → validation
- ❌ Receive with missing required inspection data → block

**Quotes:**
- ❌ Convert expired quote to sales order → warning
- ❌ Convert quote with invalid customer → validation error

**Firmware:**
- ❌ Upload firmware without version → validation error
- ❌ Upload duplicate firmware version → 409 conflict
- ❌ Deploy firmware to device already at that version → idempotent or skip

**CAPA:**
- ❌ Close CAPA without all approvals → blocked
- ❌ Reopen closed CAPA → permission check

**RFQ:**
- ❌ Close RFQ without responses → warning or allow
- ❌ Award RFQ to non-responding vendor → validation

---

## Testing Recommendations

### High Priority (P0)
1. **Inventory constraints** during WO operations (complete with shortage)
2. **Delete dependencies** (part in BOM, WO, PO)
3. **Concurrent edit conflicts** (same record, multiple users)
4. **Permission denials** for role-based operations (QE, manager)
5. **Network failure** handling (timeout, offline, 500 errors)

### Medium Priority (P1)
1. **Form validation** UI feedback (all modules)
2. **Email/date format** validation
3. **Empty states** comprehensive testing
4. **Duplicate handling** (users, serials, vendors)
5. **Negative/zero quantity** validation across all modules

### Low Priority (P2)
1. **Edge input** (emoji, null bytes, overflow)
2. **WebSocket** reconnection
3. **File upload** edge cases
4. **Report generation** with zero data
5. **Session conflicts** across devices

---

## Suggested Test Structure

```javascript
// Example: Inventory shortage test
test('complete WO with insufficient inventory shows shortage error', async ({ page }) => {
  await login(page);
  
  // Create WO for 100 units
  const wo = await apiFetch(page, '/api/v1/workorders', {
    method: 'POST',
    body: JSON.stringify({ assembly_ipn: 'PCB-001-0001', qty: 100 })
  });
  
  // Verify inventory is insufficient
  const bom = await apiFetch(page, `/api/v1/workorders/${wo.body.data.id}/bom`);
  // Ensure we don't have enough inventory for at least one component
  
  // Try to complete
  const complete = await apiFetch(page, `/api/v1/workorders/${wo.body.data.id}/complete`, {
    method: 'POST'
  });
  
  expect(complete.status).toBe(400);
  expect(complete.body.error).toContain('insufficient inventory');
  expect(complete.body.shortages).toBeTruthy(); // Should list what's missing
});
```

---

## Implementation Priority Matrix

| Category | Risk | Frequency | Priority |
|----------|------|-----------|----------|
| Inventory constraints | HIGH | HIGH | **P0** |
| Delete dependencies | HIGH | MEDIUM | **P0** |
| Concurrent edits | MEDIUM | MEDIUM | **P0** |
| Form validation UI | LOW | HIGH | P1 |
| Network failures | MEDIUM | LOW | P1 |
| Permission edge cases | MEDIUM | LOW | P1 |
| Data format validation | LOW | MEDIUM | P1 |
| Empty states | LOW | MEDIUM | P2 |
| Upload edge cases | LOW | LOW | P2 |

---

## Notes

- Backend unit tests cover **some** error cases (duplicates, missing fields)
- E2E tests focus on **happy paths** - error scenarios are under-tested
- **No network/offline simulation** in current test suite
- **No concurrent access testing** exists
- Permission tests are basic (readonly only, not role-based operations)

**Recommendation:** Start with P0 items. These represent actual bugs users will encounter (inventory shortages, delete conflicts, concurrent edits). Add at least one test per P0 category in the next sprint.

---

**End of Error Handling & Edge Cases Audit**
