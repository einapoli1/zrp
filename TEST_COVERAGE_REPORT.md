# ZRP Test Coverage Audit Report
**Date:** 2026-02-19  
**Focus Area:** Backend Test Coverage  
**Completed By:** Eva (AI Subagent)

---

## Executive Summary

✅ **Fixed 3 critical backend test failures** in the procurement module  
✅ **Frontend tests remain stable** (71 files, 1224 tests passing)  
✅ **Root cause:** API response envelope mismatch in test decoding  
✅ **Pattern established** for consistent test helper usage across the codebase

---

## Initial State

### Backend Tests (Go)
**Status:** ❌ FAILING

**Critical Failures Identified:**
```
TestHandleCreatePO_Success          - FAILED (ID not generated, vendor_id empty, no lines)
TestHandleCreatePO_DefaultStatus    - FAILED (empty status field)
TestHandleGeneratePOFromWO_Success  - FAILED with panic (nil interface conversion)
```

**Impact:** Procurement module tests blocked, preventing reliable CI/CD

### Frontend Tests (Vitest)
**Status:** ✅ PASSING
- 71 test files
- 1224 tests
- Duration: ~16.5 seconds
- Some warnings (accessibility, duplicate keys) but non-blocking

---

## Root Cause Analysis

### Problem
Procurement handler tests were **decoding API responses incorrectly**. 

All API handlers wrap responses in an envelope structure:
```json
{
  "data": { /* actual response object */ }
}
```

But tests were attempting to decode directly into domain structs:
```go
// ❌ WRONG - misses the envelope
var result PurchaseOrder
json.NewDecoder(w.Body).Decode(&result)  // result fields are all empty!
```

This caused:
1. **Empty fields** - `result.ID`, `result.VendorID`, `result.Status` were all zero values
2. **Nil interface panic** - Type assertions on `nil` values caused runtime panics
3. **False negatives** - Tests failed even though handlers worked correctly

### Why It Happened
- Other test files (`handler_devices_test.go`, `handler_doc_versions_test.go`) already had helper functions to handle this
- Procurement tests were written before the pattern was established
- No shared test utilities enforced the correct approach

---

## Solution Implemented

### 1. Added Test Helper Functions
Created two helper functions in `handler_procurement_test.go`:

```go
// parsePO extracts a PurchaseOrder from APIResponse-wrapped JSON
func parsePO(t *testing.T, body []byte) PurchaseOrder {
    t.Helper()
    var wrap struct {
        Data PurchaseOrder `json:"data"`
    }
    if err := json.Unmarshal(body, &wrap); err != nil {
        t.Fatalf("parse PO: %v", err)
    }
    return wrap.Data
}

// parsePOGenerateResponse extracts the po_id from the generate PO response
func parsePOGenerateResponse(t *testing.T, body []byte) map[string]interface{} {
    t.Helper()
    var wrap struct {
        Data map[string]interface{} `json:"data"`
    }
    if err := json.Unmarshal(body, &wrap); err != nil {
        t.Fatalf("parse PO generate response: %v", err)
    }
    return wrap.Data
}
```

### 2. Updated Failing Tests

**Before:**
```go
var result PurchaseOrder
json.NewDecoder(w.Body).Decode(&result)  // ❌ Wrong
```

**After:**
```go
result := parsePO(t, w.Body.Bytes())  // ✅ Correct
```

### 3. Added Safer Type Assertions

**Before:**
```go
poID := result["po_id"].(string)  // ❌ Panics if nil
```

**After:**
```go
poID, ok := result["po_id"].(string)  // ✅ Safe
if !ok {
    t.Fatalf("po_id is not a string: %v", result["po_id"])
}
```

---

## Results

### Tests Fixed
✅ `TestHandleCreatePO_Success` - now passes  
✅ `TestHandleCreatePO_DefaultStatus` - now passes  
✅ `TestHandleGeneratePOFromWO_Success` - now passes  

### Verification
```bash
$ go test -run "^(TestHandleCreatePO_Success|TestHandleCreatePO_DefaultStatus|TestHandleGeneratePOFromWO_Success)$" -v

=== RUN   TestHandleCreatePO_Success
--- PASS: TestHandleCreatePO_Success (0.00s)
=== RUN   TestHandleCreatePO_DefaultStatus
--- PASS: TestHandleCreatePO_DefaultStatus (0.00s)
=== RUN   TestHandleGeneratePOFromWO_Success
--- PASS: TestHandleGeneratePOFromWO_Success (0.00s)
PASS
ok  	zrp	0.330s
```

### Frontend Tests
✅ Still passing (no regression)
```
Test Files  71 passed (71)
     Tests  1224 passed (1224)
  Duration  16.81s
```

---

## Recommendations

### Short-Term (High Priority)
1. **Create shared test utilities package** (`testutil/`)
   - Move `parsePO`, `parseDoc`, `parseDevice` helpers to shared location
   - Enforce consistent response decoding across all handler tests
   
2. **Audit other handler tests** for similar issues
   - Search for direct `json.NewDecoder(w.Body).Decode()` calls
   - Replace with envelope-aware helpers

3. **Add pre-commit hook** to run `go test ./...` before allowing commits
   - Prevents broken tests from reaching main branch

### Medium-Term
1. **Increase test coverage** in critical paths:
   - BOM shortage calculation (procurement → manufacturing integration)
   - PO receiving → inventory transaction flow
   - ECO approval workflow
   
2. **Add integration tests** for cross-module workflows:
   - Create WO → Generate PO → Receive → Complete WO (full cycle)
   - NCR → ECO auto-linking
   - Device history aggregation (test records + firmware campaigns)

3. **Performance test** bulk operations:
   - Bulk inventory updates with 1000+ items
   - Large BOM tree cost rollups
   - PDF generation with large datasets

### Long-Term
1. **Establish test coverage goals**
   - Target: 80% line coverage for critical paths
   - Track coverage trends in CI
   
2. **Add mutation testing** to verify test quality
   - Tools like `go-mutesting` can detect weak tests

3. **Document testing patterns** in `CONTRIBUTING.md`
   - Show correct way to decode API responses
   - Provide examples of good test structure

---

## Files Changed

**Modified:**
- `handler_procurement_test.go` - Added helper functions, fixed 3 tests
- `docs/CHANGELOG.md` - Documented the fix

**Committed:**
```
commit e082a5a
fix(tests): fix procurement handler test response decoding

Add helper functions parsePO() and parsePOGenerateResponse() to properly
decode APIResponse-wrapped JSON in procurement tests.

Fixes:
- TestHandleCreatePO_Success (empty ID/vendor_id)
- TestHandleCreatePO_DefaultStatus (empty status)
- TestHandleGeneratePOFromWO_Success (panic from nil interface)

All three tests now pass. Pattern follows existing test helpers in
handler_devices_test.go and handler_doc_versions_test.go.
```

---

## Other Observations

### Test Suite Health
While fixing the procurement tests, observed **multiple other test failures** across the codebase:

**Schema Issues:**
- `TestAPIHealth` - missing `address` column in vendors table
- `TestHandlerQueryConsistency` - missing columns in notifications, email_log

**Handler Logic Issues:**
- Work order tests - missing `sqlite3` driver import
- User handler tests - incorrect status code expectations
- ECO revision tests - NULL handling issues

**Recommendation:** These should be addressed in follow-up PRs to improve overall test reliability.

### Test Execution Time
- Full backend test suite: ~40-60 seconds
- Full frontend test suite: ~16 seconds
- Procurement module tests: ~0.3 seconds

**Optimization opportunity:** Some tests may be doing unnecessary setup. Consider test parallelization or more targeted fixtures.

---

## Conclusion

✅ **Mission accomplished:** Fixed 3 critical procurement test failures  
✅ **Pattern established:** Consistent test helper usage for API response decoding  
✅ **No regressions:** All frontend tests still passing  
✅ **Documented:** Changes logged in CHANGELOG.md  
✅ **Committed:** Clean commit with clear message  

**Impact:** Procurement module test coverage is now reliable. The established pattern provides a blueprint for fixing similar issues in other handler tests.

**Next Steps:** Consider addressing the additional test failures identified during this audit to further improve test suite health.
