# Running ECO Module Tests

Quick reference for running the new ECO edge case tests.

## Quick Start

```bash
# Run all ECO-related tests
cd ~/.openclaw/workspace/zrp
go test -v -run "TestECO" -timeout 30s

# Run specific test files
go test -v handler_eco_edge_test.go handler_eco_test.go types.go validation.go db.go audit.go -timeout 30s
go test -v handler_eco_integrity_test.go handler_eco_test.go types.go validation.go db.go audit.go -timeout 30s

# Run only edge case tests
go test -v -run "TestECO.*edge" -timeout 30s

# Run only integrity tests  
go test -v -run "TestECO.*integrity" -timeout 30s

# Run all backend tests
go test ./... -timeout 60s

# Run frontend tests
cd frontend
npx vitest run

# Run specific frontend ECO tests
npx vitest run src/pages/ECO
```

## Test Categories

### Status Transition Tests
```bash
go test -v -run "TestECOStatusTransitions" -timeout 10s
```

### Approval Workflow Tests
```bash
go test -v -run "TestECOApproval" -timeout 10s
```

### Affected IPNs Tests
```bash
go test -v -run "TestECO_AffectedIPNs" -timeout 10s
```

### SQL Injection Safety Tests
```bash
go test -v -run "TestECO_SQLInjection" -timeout 10s
```

### Data Integrity Tests
```bash
go test -v -run "TestECO_DBConstraints|TestECO_ForeignKey" -timeout 10s
```

### Validation Tests
```bash
go test -v -run "TestECO_.*Title|TestECO_.*Description" -timeout 10s
```

### Audit Log Tests
```bash
go test -v -run "TestECO_AuditLog" -timeout 10s
```

## Expected Results

### Passing Tests (64)
All tests should pass except the 4 documented below.

### Expected Failures (4)
These tests **intentionally fail** to document bugs:

1. **TestECO_RevisionLetterOverflow**
   - Documents bug: Revision 'Z' + 1 = '[' instead of 'AA'
   - See: ECO_AUDIT_REPORT.md, Bug #1

2. **TestECOApproval_ConcurrentApprovals**
   - Documents bug: Race condition in concurrent approvals
   - See: ECO_AUDIT_REPORT.md, Bug #3
   - Note: Timing-sensitive, may pass occasionally

3. **TestECO_Implementation_AppliesPartChanges**
   - Integration test requiring parts directory setup
   - Not a bug, just needs environment setup

4. **TestECOStatusTransitions_InvalidFlow** (some subtests)
   - Documents bug: Missing state machine validation
   - See: ECO_AUDIT_REPORT.md, Bug #2

## Interpreting Results

### Success Output
```
=== RUN   TestECO_AffectedIPNs_JSONFormat
=== RUN   TestECO_AffectedIPNs_JSONFormat/Comma_separated
=== RUN   TestECO_AffectedIPNs_JSONFormat/JSON_array
--- PASS: TestECO_AffectedIPNs_JSONFormat (0.00s)
    --- PASS: TestECO_AffectedIPNs_JSONFormat/Comma_separated (0.00s)
    --- PASS: TestECO_AffectedIPNs_JSONFormat/JSON_array (0.00s)
```

### Expected Failure Output
```
=== RUN   TestECO_RevisionLetterOverflow
    handler_eco_integrity_test.go:141: Warning: Revision after Z is '[' (ASCII overflow) - should be handled differently
    handler_eco_integrity_test.go:144: Revision letter overflowed to '[' - should implement AA, AB, etc. or return error
--- FAIL: TestECO_RevisionLetterOverflow (0.00s)
```

## Test Coverage Summary

After running all tests, you should see:

```
Total ECO Tests: 68
Passed: 64
Failed: 4 (expected)
Pass Rate: 94%
```

## Debugging Failed Tests

### If a test fails unexpectedly:

1. **Check test isolation**
   ```bash
   go test -v -run "TestSpecificTest" -count=1
   ```

2. **Run with race detector**
   ```bash
   go test -race -v -run "TestECO" -timeout 30s
   ```

3. **Enable verbose logging**
   ```bash
   go test -v -run "TestECO" -timeout 30s 2>&1 | tee test.log
   ```

4. **Run single file**
   ```bash
   go test -v handler_eco_edge_test.go handler_eco_test.go types.go validation.go db.go audit.go
   ```

## Clean Test Environment

If tests are failing due to state issues:

```bash
# Clear any test databases
rm -f *.db test*.db

# Re-run tests
go test -v -run "TestECO" -timeout 30s
```

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Run ECO Tests
  run: |
    cd ~/.openclaw/workspace/zrp
    go test -v -run "TestECO" -timeout 30s -json | tee eco-test-results.json
```

### Expected CI Behavior
- **64 tests should pass**
- **4 tests expected to fail** (documented bugs)
- Exit code will be non-zero due to expected failures
- This is **normal and expected** until bugs are fixed

## After Bug Fixes

Once the 3 medium-severity bugs are fixed:

1. Update tests to reflect new behavior
2. Remove "expected failure" comments
3. All 68 tests should pass
4. Update this guide to remove expected failures section

## Questions?

See detailed bug descriptions and recommended fixes in:
- `ECO_AUDIT_REPORT.md` - Full audit report with bug details
- `ECO_POLISH_SUMMARY.md` - Task completion summary

---

**Last Updated:** 2026-02-21  
**Test Suite Version:** 1.0  
**Total Tests:** 68 (64 passing, 4 expected failures)
