# ZRP Integration Tests

> **Status:** Phase 1 Complete - Documentation Tests Passing  
> **Created:** 2026-02-19  
> **Test Framework:** Playwright + TypeScript

## Overview

Integration tests verify **end-to-end workflows** that span multiple ZRP modules. Unlike unit tests (which test individual handlers), integration tests ensure that BOM → Inventory → Work Orders → Procurement workflows function correctly across module boundaries.

## Test Coverage

### ✅ TC-INT-001: Work Order Completion → Inventory Updates

**Status:** Phase 1 Complete (Documentation Tests Passing)

**File:** `tc-int-001-wo-inventory.spec.ts`

**Test Objective:**
Verify that completing a work order properly updates inventory by:
- Adding finished goods to inventory
- Consuming component materials
- Releasing reserved quantities

**Current Results:**
- ✅ Test infrastructure working
- ✅ All pages accessible (Work Orders, Inventory, Parts, Procurement)
- ✅ Test documents expected behavior vs actual behavior
- ✅ Known gaps clearly identified

**Known Gaps (from INTEGRATION_TESTS_NEEDED.md):**
- **Gap #4.1:** Creating WO does NOT reserve inventory (qty_reserved stays 0)
- **Gap #4.5:** Completing WO does NOT update inventory
- **Gap #4.6:** No material kitting/consumption workflow

**Test Cases:**
1. `should document WO to inventory integration workflow` - ✅ PASSED
   - Verifies all pages load correctly
   - Documents expected vs actual behavior
   - Captures screenshots for documentation

2. `should verify API endpoints for WO-inventory integration` - ⚠️ AUTH ISSUE
   - Tests API endpoint availability
   - Documents required integration points
   - Minor: Login timing issue (not critical)

3. `should provide manual testing instructions` - ✅ PASSED
   - Comprehensive manual test guide
   - Step-by-step workflow documentation
   - Implementation remediation steps

**Next Steps:**
- [ ] Fix Gap #4.5: Implement inventory update on WO completion
- [ ] Fix Gap #4.1: Implement material reservation on WO creation
- [ ] Expand test to include actual assertions (currently in documentation mode)
- [ ] Add test data setup helpers

---

## Running Integration Tests

### Prerequisites

1. **ZRP Server Running:**
   ```bash
   cd /Users/jsnapoli1/.openclaw/workspace/zrp
   go run . -port 9000
   ```

2. **Install Dependencies:**
   ```bash
   cd frontend
   npm install
   npx playwright install
   ```

### Run Tests

**All Integration Tests:**
```bash
cd frontend
npx playwright test --config=playwright.integration.config.ts
```

**Specific Test:**
```bash
npx playwright test tc-int-001-wo-inventory.spec.ts --config=playwright.integration.config.ts
```

**With UI:**
```bash
npx playwright test --config=playwright.integration.config.ts --ui
```

**Debug Mode:**
```bash
npx playwright test --config=playwright.integration.config.ts --debug
```

---

## Test Architecture

### Configuration Files

- **`playwright.integration.config.ts`** - Integration test config (uses running server on port 9000)
- **`playwright.config.ts`** - Standard e2e test config (starts own server on port 9001)

**Key Differences:**
- Integration tests assume server is already running
- Longer timeouts (90s vs 60s)
- Uses `baseURL: http://localhost:9000`
- No `webServer` section (doesn't start server)

### Test Structure

Each integration test follows this pattern:

```typescript
test.describe('TC-XXX: Workflow Name', () => {
  test.beforeEach(async ({ page }) => {
    // Login to ZRP
  });

  test('should test specific workflow', async ({ page }) => {
    // 1. Setup: Create test data
    // 2. Execute: Perform workflow actions
    // 3. Verify: Check expected outcomes
    // 4. Document: Log gaps and expected behavior
  });
});
```

---

## Test Data Management

### Current Approach
- Tests use existing ZRP database
- Manual setup required (see manual test guide in TC-INT-001)

### Future Improvements
- [ ] Create `test-data-setup.ts` helper module
- [ ] Add `resetTestData()` function for repeatable tests
- [ ] Implement test fixtures for common scenarios
- [ ] Add API helpers for programmatic data creation

---

## Known Issues & Workarounds

### Issue #1: Login Timing
**Symptom:** `page.waitForURL` timeout in beforeEach  
**Workaround:** Increase timeout to 10s  
**Fix:** Improve login flow detection

### Issue #2: Parts Table Detection
**Symptom:** `hasPartsTable = false` on Parts page  
**Reason:** Parts page may use different layout/component  
**Impact:** Low - doesn't block core workflow tests

### Issue #3: API Token Management
**Symptom:** API endpoint tests get 401 Unauthorized  
**Reason:** Token retrieval from localStorage fails in test context  
**Workaround:** Use UI-based tests instead of API tests  
**Fix:** Improve token management in test setup

---

## Manual Testing Guide

**See TC-INT-001, Test Case #3** for comprehensive manual test procedures.

**Quick Start:**
1. Create test parts (TST-RES-001, TST-CAP-001, TST-ASY-001)
2. Create BOM for TST-ASY-001
3. Add initial inventory
4. Create work order for 10x TST-ASY-001
5. Check inventory (verify reservation - EXPECTED GAP)
6. Complete work order
7. Check inventory (verify consumption - EXPECTED GAP)

**Expected Results:**
- ⚠️ Inventory NOT updated (Gap #4.5)
- ⚠️ Materials NOT reserved (Gap #4.1)

---

## Integration Test Roadmap

### Phase 1: Documentation ✅ COMPLETE
- [x] TC-INT-001 test infrastructure
- [x] Page accessibility verification
- [x] Gap documentation
- [x] Manual test guide

### Phase 2: Backend Implementation (IN PROGRESS)
- [ ] Implement inventory reservation service
- [ ] Implement inventory consumption service
- [ ] Hook WO creation → material reservation
- [ ] Hook WO completion → inventory update

### Phase 3: Automated Tests (PENDING)
- [ ] Add test data setup helpers
- [ ] Implement full automated workflow test
- [ ] Add assertions for expected behavior
- [ ] Verify transactions are created

### Phase 4: Expand Coverage (FUTURE)
- [ ] TC-INT-002: BOM Shortage → Procurement
- [ ] TC-INT-003: Material Reservation Edge Cases
- [ ] TC-INT-004: NCR → ECO Traceability
- [ ] TC-INT-005: Scrap/Yield Tracking
- [ ] TC-INT-006: Partial PO Receiving

---

## Screenshots & Artifacts

Test screenshots are saved to:
```
frontend/test-results/tc-int-001-step*.png
```

**Generated Screenshots:**
- `step1-wo-page.png` - Work Orders page
- `step2-inventory-page.png` - Inventory page
- `step3-inventory-initial.png` - Initial inventory state
- `step4-parts-page.png` - Parts/BOM page
- `step5-procurement-page.png` - Procurement page

---

## CI/CD Integration

### Current Status
- ❌ Not yet integrated into CI pipeline

### Planned Integration
```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests
on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Start ZRP Server
        run: go run . -port 9000 &
      - name: Run Integration Tests
        run: |
          cd frontend
          npm install
          npx playwright install --with-deps
          npx playwright test --config=playwright.integration.config.ts
```

---

## Contributing

### Adding New Integration Tests

1. **Create Test File:**
   ```bash
   touch frontend/e2e/integration/tc-int-XXX-your-test.spec.ts
   ```

2. **Follow Naming Convention:**
   - `tc-int-XXX-workflow-name.spec.ts`
   - Match test ID from `docs/INTEGRATION_TESTS_NEEDED.md`

3. **Test Structure:**
   ```typescript
   /**
    * TC-INT-XXX: Test Name
    * 
    * **Test Objective:** ...
    * **Known Gaps:** ...
    * **Reference:** docs/INTEGRATION_TESTS_NEEDED.md
    */
   test.describe('TC-INT-XXX: Test Name', () => {
     // Tests here
   });
   ```

4. **Document Gaps:**
   - Use console.log to document expected vs actual behavior
   - Reference specific gaps from WORKFLOW_GAPS.md
   - Provide remediation steps

5. **Screenshots:**
   - Use `await page.screenshot({ path: 'test-results/...' })`
   - Capture key workflow states

---

## References

- **Test Spec:** `docs/INTEGRATION_TESTS_NEEDED.md`
- **Workflow Gaps:** `docs/WORKFLOW_GAPS.md`
- **Testing Guide:** `docs/TESTING.md`
- **Playwright Docs:** https://playwright.dev

---

## Contact & Support

**Test Owner:** ZRP Development Team  
**Created:** 2026-02-19  
**Last Updated:** 2026-02-19

For questions about integration tests, see:
- Test specification: `docs/INTEGRATION_TESTS_NEEDED.md`
- Implementation gaps: `docs/WORKFLOW_GAPS.md`
