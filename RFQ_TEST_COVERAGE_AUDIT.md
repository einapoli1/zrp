# RFQ Test Coverage Audit - 2026-02-23

## Current Coverage Analysis

### Backend Tests (handler_rfq_test.go)
**Covered:**
- ✓ List RFQs (empty and with data)
- ✓ Get RFQ (success with full data, not found)
- ✓ Create RFQ (success with lines/vendors, validation)
- ✓ Update RFQ (success, replaces lines/vendors)
- ✓ Delete RFQ (success, cascade deletes)
- ✓ Send RFQ (draft→sent, invalid status rejection)
- ✓ Award RFQ (creates PO, validates vendor)
- ✓ Close RFQ (awarded→closed, invalid status rejection)
- ✓ Create/Update RFQ Quote (vendor status changes to quoted)
- ✓ Compare RFQ (matrix with lines/vendors/quotes)
- ✓ Basic workflow state transitions

### Frontend Tests
**RFQs.test.tsx:**
- ✓ Loading state
- ✓ List rendering
- ✓ Empty state
- ✓ Create RFQ button and dialog
- ✓ Status badges

**RFQDetail.test.tsx:**
- ✓ Loading state
- ✓ Detail rendering
- ✓ API calls (getRFQ, compareRFQ)

## Coverage Gaps (Missing Tests)

### Backend - Critical Missing Tests

#### 1. ID Generation Pattern
- [ ] Verify RFQ IDs use `nextID("RFQ", "rfqs", 4)` pattern
- [ ] Test sequential ID generation
- [ ] Test ID collision handling

#### 2. Edge Cases - Required Fields
- [ ] Create RFQ with empty title (should fail)
- [ ] Create RFQ with null/missing fields
- [ ] Line items with invalid qty (negative, zero, null)
- [ ] Line items with empty IPN
- [ ] Vendor assignment with non-existent vendor

#### 3. Edge Cases - Business Logic
- [ ] RFQ with no lines (should allow?)
- [ ] RFQ with no vendors (should allow?)
- [ ] RFQ with duplicate lines (same IPN multiple times)
- [ ] RFQ with duplicate vendors (same vendor multiple times)
- [ ] Multi-vendor RFQ with different quotes per line
- [ ] Partial quotes (some lines quoted, others not)

#### 4. Due Date Handling
- [ ] Create RFQ with due_date in the past
- [ ] Update due_date after RFQ is sent
- [ ] Quote submission after due_date
- [ ] Dashboard filtering by overdue RFQs

#### 5. Quote Management
- [ ] Multiple quotes from same vendor for same line (should replace or fail?)
- [ ] Quote with zero or negative unit_price
- [ ] Quote with invalid lead_time_days
- [ ] Delete quote behavior
- [ ] Update quote after RFQ is awarded

#### 6. Award Functionality
- [ ] Award RFQ with no quotes
- [ ] Award RFQ to vendor with partial quotes
- [ ] Award RFQ per-line (split award across vendors)
- [ ] Verify PO creation details (vendor_id, notes, status)
- [ ] Verify PO lines match quote data
- [ ] Award already-awarded RFQ (should fail)

#### 7. Status Transitions
- [ ] Invalid transitions: draft→closed, sent→draft, closed→sent
- [ ] Close RFQ without awarding (sent→closed)
- [ ] Vendor status: pending→quoted→awarded
- [ ] Concurrent status updates (race conditions)

#### 8. Integration with Other Modules
- [ ] PO creation from RFQ (verify all fields)
- [ ] PO lines creation (IPN, qty, unit_price from quotes)
- [ ] Vendor lookup in RFQ vendor assignment
- [ ] Audit log entries for all RFQ actions

#### 9. Email Body Generation
- [ ] handleRFQEmailBody with full data
- [ ] Email body with no lines
- [ ] Email body with no due_date
- [ ] Email body formatting

#### 10. Dashboard
- [ ] Dashboard counts (open, pending, awarded this month)
- [ ] Dashboard RFQ list with stats
- [ ] Total quoted value calculation
- [ ] Response rate calculation

#### 11. Concurrency & Data Integrity
- [ ] Simultaneous updates to same RFQ
- [ ] Cascade delete verification (lines, vendors, quotes)
- [ ] Foreign key constraints
- [ ] Transaction rollback on error

### Frontend - Missing Tests

#### RFQs.tsx
- [ ] Filter by status
- [ ] Search RFQs
- [ ] Sort by date, status
- [ ] Delete RFQ
- [ ] Error handling (API failures)

#### RFQDetail.tsx
- [ ] Edit RFQ (title, notes, due_date)
- [ ] Add/remove line items
- [ ] Add/remove vendors
- [ ] Send RFQ action (draft→sent)
- [ ] Submit quotes as vendor
- [ ] Compare quotes view
- [ ] Award RFQ action
- [ ] Close RFQ action
- [ ] Generate email body
- [ ] Status-based action visibility
- [ ] Validation error display

## Recommended Test Additions

### Priority 1 (Critical)
1. **handler_rfq_comprehensive_test.go** - Edge cases and validation
2. **handler_rfq_integration_test.go** - RFQ→Quote→PO workflow
3. **handler_rfq_concurrency_test.go** - Race conditions

### Priority 2 (High)
4. Frontend: RFQDetail line/vendor management tests
5. Frontend: Award workflow tests
6. Frontend: Validation error tests

### Priority 3 (Medium)
7. Dashboard functionality tests
8. Email body generation tests
9. Per-line award tests

## Test Execution Plan

1. Create comprehensive backend test file
2. Run tests: `go test -v -run TestRFQ`
3. Fix any failing tests
4. Add frontend component tests
5. Run frontend tests: `cd frontend && npx vitest run --grep RFQ`
6. Update CHANGELOG.md with findings
7. Commit all changes together
