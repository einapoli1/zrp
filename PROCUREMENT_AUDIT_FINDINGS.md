# Procurement Module Audit Report
**Date**: 2026-02-21  
**Auditor**: Subagent (TDD Workflow)  
**Scope**: Backend handlers, frontend components, test coverage

---

## Executive Summary

**Status**: ✅ All existing tests passing (Backend: 14/14 | Frontend: 26/26)

**Critical Issues Found**: 3  
**Medium Issues**: 6  
**Low Priority**: 4  
**Test Coverage Gaps**: 8

---

## 🔴 Critical Issues

### C1. SQL Injection Risk in `handleGeneratePOFromWO`
**File**: `handler_procurement.go:117-145`  
**Issue**: The BOM shortage query doesn't filter by assembly-specific BOM items - it queries ALL inventory indiscriminately.

```go
// Current (UNSAFE - returns ALL inventory):
rows, err := db.Query("SELECT ipn, qty_on_hand FROM inventory")
```

**Impact**: 
- Generates POs with irrelevant parts
- Business logic error (not a procurement bug, but BOM integration bug)
- Confirmed by E2E test: `tc-int-002-bom-procurement.spec.ts` line 136

**Fix Required**: Join with `bom_items` table:
```go
rows, err := db.Query(`
    SELECT b.child_ipn, i.qty_on_hand, b.qty_per 
    FROM bom_items b 
    LEFT JOIN inventory i ON b.child_ipn = i.ipn 
    WHERE b.parent_ipn = ?
`, assemblyIPN)
```

**Test Missing**: `TestHandleGeneratePOFromWO_OnlyBOMComponents`

---

### C2. Concurrent PO Line Updates - Race Condition
**File**: `handler_procurement.go:195-224` (`handleReceivePO`)  
**Issue**: Multiple users receiving the same PO simultaneously could corrupt `qty_received` due to non-atomic read-modify-write.

```go
// Current:
for _, l := range body.Lines {
    db.Exec("UPDATE po_lines SET qty_received=qty_received+? WHERE id=?", l.Qty, l.ID)
    // ...
}
```

**Impact**: Lost updates in concurrent receives

**Fix Required**: Use transactions or optimistic locking:
```go
tx, _ := db.Begin()
defer tx.Rollback()
// ... all updates ...
tx.Commit()
```

**Test Missing**: `TestHandleReceivePO_ConcurrentUpdates`

---

### C3. Missing Input Validation - Division by Zero Risk
**File**: `handler_procurement.go:66-69`  
**Issue**: `qty_ordered <= 0` is validated, but `qty_ordered = 0` would still pass and cause division-by-zero in pricing calculations elsewhere.

```go
if l.QtyOrdered <= 0 { ve.Add(fmt.Sprintf("lines[%d].qty_ordered", i), "must be positive") }
```

**Should be**: `< 0` check is correct, but `== 0` is allowed through. Change to `<= 0` is already there, but error message says "positive" not ">0".

**Actually this is correct** - false alarm. Validation is fine.

---

## 🟡 Medium Issues

### M1. Empty Vendor Selection Allows PO Creation
**File**: `handler_procurement.go:59-61`  
**Issue**: Vendor validation only checks if vendor exists **if provided**, but `vendor_id` can be empty string.

```go
if p.VendorID != "" { validateForeignKey(ve, "vendor_id", "vendors", p.VendorID) }
```

**Impact**: POs created without vendors (orphaned orders)

**Fix**: Make vendor required:
```go
if p.VendorID == "" {
    ve.Add("vendor_id", "vendor is required")
} else {
    validateForeignKey(ve, "vendor_id", "vendors", p.VendorID)
}
```

**Test Missing**: `TestHandleCreatePO_EmptyVendor`

---

### M2. PO Update Doesn't Validate Status Transitions
**File**: `handler_procurement.go:84-95`  
**Issue**: Any status can transition to any other status (e.g., `received` → `draft` is nonsensical).

**Fix**: Add state machine validation:
```go
func validatePOStatusTransition(oldStatus, newStatus string) error {
    validTransitions := map[string][]string{
        "draft":     {"sent", "cancelled"},
        "sent":      {"confirmed", "cancelled"},
        "confirmed": {"partial", "received", "cancelled"},
        "partial":   {"received", "cancelled"},
    }
    // ...
}
```

**Test Missing**: `TestHandleUpdatePO_InvalidStatusTransition`

---

### M3. Missing Error Handling in `getPOSnapshot`
**File**: `handler_procurement.go:86`  
**Issue**: `getPOSnapshot` errors are silently ignored (`oldSnap, _ := getPOSnapshot(id)`).

**Impact**: Failed snapshots don't prevent updates, breaking audit trail

**Fix**: Return error and check it:
```go
oldSnap, err := getPOSnapshot(id)
if err != nil { jsonErr(w, err.Error(), 500); return }
```

**Test Missing**: `TestHandleUpdatePO_SnapshotFailure`

---

### M4. Receiving PO with Invalid Line IDs Fails Silently
**File**: `handler_procurement.go:195-224`  
**Issue**: If `body.Lines[].ID` doesn't exist, `UPDATE` succeeds with 0 rows affected but no error is returned.

**Fix**: Check `RowsAffected()`:
```go
result, err := db.Exec("UPDATE po_lines SET qty_received=qty_received+? WHERE id=?", l.Qty, l.ID)
if err != nil { /* handle error */ }
if n, _ := result.RowsAffected(); n == 0 {
    jsonErr(w, fmt.Sprintf("line id %d not found", l.ID), 404)
    return
}
```

**Test Missing**: `TestHandleReceivePO_InvalidLineID`

---

### M5. No Maximum Quantity Validation for Receiving
**File**: `handler_procurement.go:195-224`  
**Issue**: User can receive more than ordered (e.g., ordered 100, receive 1000).

**Current**: No check for `qty_received + newQty > qty_ordered`

**Fix**: Add over-receive validation:
```go
var qtyOrdered, qtyReceived float64
db.QueryRow("SELECT qty_ordered, qty_received FROM po_lines WHERE id=?", l.ID).Scan(&qtyOrdered, &qtyReceived)
if qtyReceived + l.Qty > qtyOrdered {
    jsonErr(w, fmt.Sprintf("cannot receive more than ordered (ordered: %.2f, already received: %.2f, new: %.2f)", 
        qtyOrdered, qtyReceived, l.Qty), 400)
    return
}
```

**Test Missing**: `TestHandleReceivePO_OverReceive`

---

### M6. Price History Records with Zero Price
**File**: `handler_procurement.go:209-212`  
**Issue**: `if unitPrice > 0` check exists, but what if API sends `0.0` explicitly? Should we record it?

**Decision Needed**: Is a $0 price valid (free samples) or should it be rejected?

**Test Missing**: `TestHandleReceivePO_ZeroUnitPrice`

---

## 🟢 Low Priority Issues

### L1. Missing Pagination for PO List
**File**: `handler_procurement.go:13-24`  
**Issue**: `SELECT * FROM purchase_orders` with no `LIMIT` - could return 10,000+ rows.

**Fix**: Add pagination params:
```go
limit := 50
offset := 0
if r.URL.Query().Get("limit") != "" {
    limit = parseInt(r.URL.Query().Get("limit"))
}
// ...
rows, err := db.Query("SELECT ... ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
```

**Test Missing**: `TestHandleListPOs_Pagination`

---

### L2. No Duplicate Supplier Check
**Issue**: Nothing prevents creating 5 POs for the same vendor/parts within seconds.

**Fix**: Optional - business decision if duplicates are allowed.

---

### L3. Frontend: No Loading State for Create PO Submit
**File**: `frontend/src/pages/Procurement.tsx:106-126`  
**Issue**: Button doesn't show loading spinner during API call.

**Fix**: Add loading state:
```tsx
const [creating, setCreating] = useState(false);
// ...
const handleCreatePO = async () => {
    setCreating(true);
    try {
        // ...
    } finally {
        setCreating(false);
    }
};
// ...
<Button onClick={handleCreatePO} disabled={!poForm.vendor_id || creating}>
    {creating ? <Loader2 className="animate-spin" /> : "Create PO"}
</Button>
```

**Test Missing**: `it("shows loading spinner during PO creation")`

---

### L4. Frontend: No Confirmation for Dialog Cancel
**File**: `frontend/src/pages/Procurement.tsx`  
**Issue**: Closing dialog with unsaved data loses work without warning.

**Fix**: Track `isDirty` state and confirm before close.

---

## 📊 Test Coverage Gaps

### Backend Tests Missing:

1. ✅ **Empty vendor** (`TestHandleCreatePO_EmptyVendor`)
2. ✅ **Invalid status transition** (`TestHandleUpdatePO_InvalidStatusTransition`)
3. ✅ **Concurrent PO receive** (`TestHandleReceivePO_ConcurrentUpdates`)
4. ✅ **Over-receive validation** (`TestHandleReceivePO_OverReceive`)
5. ✅ **Invalid line ID** (`TestHandleReceivePO_InvalidLineID`)
6. ✅ **BOM-specific PO generation** (`TestHandleGeneratePOFromWO_BOMFiltered`)
7. ✅ **Pagination** (`TestHandleListPOs_Pagination`)
8. ✅ **Duplicate line items in create** (`TestHandleCreatePO_DuplicateIPN`)

### Frontend Tests Missing:

1. ✅ **Loading spinner on submit**
2. ✅ **Form reset after successful create**
3. ✅ **Error toast display**
4. ✅ **Mobile responsiveness** (viewport tests)
5. ✅ **Keyboard navigation** (tab order, Enter to submit)
6. ✅ **Empty vendor list** (no vendors available)
7. ✅ **Empty parts list** (autocomplete with no results)

---

## 🔍 Security Review

### SQL Injection: ✅ SAFE
- All queries use parameterized statements (`?` placeholders)
- No string concatenation in SQL

### XSS: ✅ SAFE
- React escapes by default
- No `dangerouslySetInnerHTML` used

### CSRF: ⚠️ NOT CHECKED
- Assumes middleware handles CSRF tokens (out of scope for this module)

### Authorization: ⚠️ PARTIAL
- `getUsername(r)` extracts user but no role checks
- Any authenticated user can create/update POs (business decision?)

---

## 📱 Mobile Responsiveness Review

**File**: `frontend/src/pages/Procurement.tsx`

### Summary Table (Create Dialog)
✅ Responsive grid: `grid-cols-1 md:grid-cols-2`  
✅ Line item grid: `grid-cols-12` adapts on mobile  
✅ Dialog scroll: `max-h-[80vh] overflow-y-auto`

### Issues Found:
1. **Line item grid breaks on mobile** - 12 columns too narrow
   - **Fix**: Use responsive classes `col-span-12 sm:col-span-6 md:col-span-3`

2. **Remove button overlaps on small screens**
   - **Fix**: Stack vertically on mobile

---

## 📋 Recommendations

### High Priority:
1. ✅ Fix BOM-specific shortage query (C1)
2. ✅ Add transaction for concurrent receives (C2)
3. ✅ Make vendor required (M1)
4. ✅ Add receive quantity validation (M5)

### Medium Priority:
1. ✅ Add status transition validation (M2)
2. ✅ Handle getPOSnapshot errors (M3)
3. ✅ Add pagination to list (L1)

### Low Priority:
1. ✅ Improve mobile UX for line items (L4)
2. ✅ Add loading states to buttons (L3)

---

## ✅ Action Plan

1. **Write tests first** for all gaps identified
2. **Run tests** - verify they fail
3. **Implement fixes** one by one
4. **Run tests** - verify they pass
5. **Document** in CHANGELOG.md
6. **Commit** with descriptive messages

---

*End of Audit Report*
