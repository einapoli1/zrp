# PO Auto-Generation from BOM Shortages - Test Implementation Report

**Date**: 2026-02-19  
**Task**: Implement tests for PO auto-generation from work order BOM shortages (Critical Gap #4)  
**Status**: ✅ **COMPLETE** - All tests passing

---

## Summary

Successfully implemented comprehensive test suite for PO auto-generation from BOM shortages feature. All 6 tests pass, covering the complete workflow from shortage detection through PO creation.

## Test Coverage

### 1. **TestPOAutogen_BOMShortageDetection** ✅
**Purpose**: Verify that BOM shortages are correctly calculated  
**Scenario**: 
- Work order requires 100 units (10 WO qty × 10 per assembly)
- Inventory has 30 units
- Expected shortage: 70 units

**Result**: PO suggestion correctly identifies 70-unit shortage

### 2. **TestPOAutogen_MultiVendorSplitting** ✅
**Purpose**: Verify suggestions are split by preferred vendor  
**Scenario**:
- 3 components required from 2 different vendors
- DigiKey: Resistors + Capacitors (2 parts)
- Mouser: MCU IC (1 part)

**Result**: Creates 2 separate PO suggestions, one per vendor

### 3. **TestPOAutogen_SuggestedPOIncludesCorrectDetails** ✅
**Purpose**: Verify PO suggestions contain accurate vendor, quantity, pricing  
**Validations**:
- ✅ Correct vendor ID
- ✅ Correct MPN (Manufacturer Part Number)
- ✅ Correct manufacturer name
- ✅ Accurate shortage quantity (15 units)
- ✅ Correct unit price from part_vendors table

### 4. **TestPOAutogen_ApproveRejectWorkflow** ✅
**Purpose**: Verify user can approve or reject PO suggestions  
**Scenarios**:
- ✅ Reject suggestion with reason
- ✅ Approve suggestion
- ✅ Status updates correctly in database
- ✅ Audit trail captured

### 5. **TestPOAutogen_ApprovedPOCreated** ✅
**Purpose**: Verify approved suggestions create actual POs  
**Validations**:
- ✅ PO created with correct vendor
- ✅ PO contains correct number of lines (2 lines)
- ✅ Each line has correct IPN, quantity, price, MPN
- ✅ PO status set to 'draft'
- ✅ PO linked back to suggestion

**Detailed Line Verification**:
- CAP-002: 40 units @ $0.12, MPN: CAP-MPN-002
- RES-002: 30 units @ $0.05, MPN: RES-MPN-002

### 6. **TestPOAutogen_NoSuggestionsWhenNoShortage** ✅
**Purpose**: Verify no suggestions created when inventory is sufficient  
**Scenario**:
- Work order needs 10 units
- Inventory has 100 units
- No shortage exists

**Result**: Returns "No shortages found", creates 0 suggestions

---

## Implementation Details

### New Handler Functions

#### `handleGeneratePOSuggestions`
- Analyzes BOM requirements vs. inventory levels
- Calculates shortages: Required - OnHand
- Groups shortages by preferred vendor
- Creates po_suggestions records with status='pending'
- Inserts po_suggestion_lines with qty_needed and pricing

**Key Logic**:
```go
req.Required = req.QtyPer * float64(woQty)
req.Shortage = req.Required - req.OnHand
if req.Shortage > 0 {
    // Add to suggestions
}
```

#### `handleReviewPOSuggestion`
- Allows approval or rejection of suggestions
- Updates suggestion status and audit trail
- Optionally creates PO from approved suggestion
- Copies suggestion lines to po_lines
- Links created PO back to suggestion

**Workflow**:
1. Validate suggestion exists and is 'pending'
2. Update status to 'approved' or 'rejected'
3. If approved + create_po=true:
   - Generate PO ID
   - Create purchase_orders record
   - Copy po_suggestion_lines to po_lines
   - Link PO to suggestion

### Database Schema

**New Tables**:
```sql
CREATE TABLE po_suggestions (
    id INTEGER PRIMARY KEY,
    wo_id TEXT NOT NULL,
    vendor_id TEXT NOT NULL,
    status TEXT DEFAULT 'pending', -- pending/approved/rejected
    notes TEXT,
    created_at DATETIME,
    reviewed_by TEXT,
    reviewed_at DATETIME,
    po_id TEXT -- Links to created PO
);

CREATE TABLE po_suggestion_lines (
    id INTEGER PRIMARY KEY,
    suggestion_id INTEGER NOT NULL,
    ipn TEXT NOT NULL,
    mpn TEXT,
    manufacturer TEXT,
    qty_needed REAL NOT NULL,
    estimated_unit_price REAL,
    notes TEXT
);
```

**Existing Tables Used**:
- `bom_items`: Maps parent assemblies to child components with qty_per
- `inventory`: Tracks qty_on_hand for each IPN
- `part_vendors`: Provides vendor info, pricing, preferred vendor flag
- `work_orders`: Source of assembly IPN and build quantity

---

## Bug Fixes Applied

### 1. **Database Connection Issue**
**Problem**: Tests using `:memory:` SQLite databases had connection isolation - handler couldn't see test data  
**Solution**: Changed to `file:test_po_autogen.db?mode=memory&cache=shared` for shared in-memory database

### 2. **NULL Handling in PO Creation**
**Problem**: NULL values in po_suggestion_lines caused scanning errors  
**Solution**: Added `COALESCE()` to handle NULL mpn, manufacturer, notes

### 3. **Zero Quantity Check**
**Problem**: CHECK constraint failed when qty_needed was 0  
**Solution**: Added skip logic for lines with qtyNeeded <= 0

---

## Test Execution Results

```bash
$ go test -run TestPOAutogen -v

=== RUN   TestPOAutogen_BOMShortageDetection
--- PASS: TestPOAutogen_BOMShortageDetection (0.00s)
=== RUN   TestPOAutogen_MultiVendorSplitting
--- PASS: TestPOAutogen_MultiVendorSplitting (0.00s)
=== RUN   TestPOAutogen_SuggestedPOIncludesCorrectDetails
--- PASS: TestPOAutogen_SuggestedPOIncludesCorrectDetails (0.00s)
=== RUN   TestPOAutogen_ApproveRejectWorkflow
--- PASS: TestPOAutogen_ApproveRejectWorkflow (0.00s)
=== RUN   TestPOAutogen_ApprovedPOCreated
--- PASS: TestPOAutogen_ApprovedPOCreated (0.00s)
=== RUN   TestPOAutogen_NoSuggestionsWhenNoShortage
--- PASS: TestPOAutogen_NoSuggestionsWhenNoShortage (0.00s)
PASS
ok      zrp     0.322s
```

**Success Rate**: 6/6 tests passing (100%)

---

## Files Modified

1. **handler_po_autogen_test.go** (564 lines)
   - 6 comprehensive test functions
   - setupPOAutogenTestDB() helper
   - Covers all user stories from FEATURE_TEST_MATRIX.md

2. **handler_procurement.go**
   - Added handleGeneratePOSuggestions() function
   - Added handleReviewPOSuggestion() function
   - Fixed NULL handling and zero-quantity checks

---

## Commit

```
commit 8831e83
test: Add PO auto-generation from BOM shortage tests
```

---

## Next Steps (Future Enhancements)

1. **API Route Registration**: Add routes for new handlers to main.go
2. **Frontend Integration**: Build UI for viewing/approving PO suggestions
3. **Email Notifications**: Alert purchasing when suggestions are created
4. **Price Optimization**: Compare prices across multiple vendor options
5. **Lead Time Analysis**: Factor vendor lead times into suggestions
6. **MOQ Handling**: Respect Minimum Order Quantities from part_vendors
7. **Batch Suggestion Review**: Allow bulk approve/reject operations

---

## Conclusion

✅ **All acceptance criteria met**:
- [x] BOM shortage detection working correctly
- [x] Multi-vendor PO splitting implemented
- [x] Suggested POs include correct vendor, quantity, pricing
- [x] Approve/reject workflow functional
- [x] Approved POs created in system
- [x] All tests passing
- [x] Changes committed

The PO auto-generation feature is now **fully tested** and ready for production deployment after route registration and frontend integration.
