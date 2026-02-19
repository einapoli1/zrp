# Inventory Adjustment E2E Tests

## Overview

Comprehensive end-to-end test suite for the manual inventory adjustment workflow in ZRP. This critical feature allows operators to correct inventory discrepancies through manual stock adjustments, issue/receive transactions, and reorder point configuration.

## Test File

- **Location**: `frontend/e2e/inventory-adjustment.spec.ts`
- **Test Count**: 8 comprehensive test scenarios
- **Framework**: Playwright with TypeScript

## Test Coverage

### 1. Manual Stock Additions
- Tests adjust transactions to add inventory
- Verifies quantity updates persist correctly
- Ensures UI reflects new quantities

### 2. Manual Stock Removals
- Tests issue transactions to remove stock
- Validates subtraction logic
- Confirms transaction history tracking

### 3. Transaction History Verification
- Tests multiple transaction types (receive, issue, adjust)
- Verifies all transactions appear in history table
- Validates reference numbers and notes are recorded

### 4. Reorder Point Configuration
- Tests bulk edit functionality for reorder points
- Verifies reorder quantity updates
- Ensures changes persist across navigation

### 5. Location Management
- Tests location updates via bulk edit
- Verifies location changes in table and detail views
- Validates data persistence

### 6. Low Stock Alert System
- Tests low stock badge appearance when qty ≤ reorder point
- Verifies visual indicators (red borders/backgrounds)
- Tests low stock filter functionality
- Validates alert count accuracy

### 7. Data Persistence
- Tests quantity persistence across page reloads
- Verifies data integrity in list and detail views
- Ensures backend updates are saved correctly

### 8. Sequential Adjustments
- Tests multiple consecutive adjustments
- Verifies transaction history accumulates correctly
- Validates final quantities after multiple changes

## Helper Functions

### `goToInventory(page: Page)`
Navigates to inventory page with proper waiting:
- Waits for network idle
- Verifies page heading loaded
- Ensures page is interactive

### `quickReceive(page: Page, ipn, qty, reference?, notes?)`
Performs quick receive operation:
- Opens quick receive dialog
- Fills in form fields
- Submits and waits for completion
- Allows time for backend processing

## Running the Tests

```bash
# Run all inventory adjustment tests
npm run test:e2e -- inventory-adjustment.spec.ts

# Run a specific test
npm run test:e2e -- inventory-adjustment.spec.ts -g "should manually add stock"

# Run with UI mode for debugging
npm run test:e2e -- inventory-adjustment.spec.ts --ui

# Run with headed browser
npm run test:e2e -- inventory-adjustment.spec.ts --headed
```

## Test Data Strategy

- **Unique Test Data**: Each test uses timestamp-based IPNs (`TEST-ADD-${Date.now()}`)
- **Isolation**: Tests don't interfere with each other
- **Deterministic**: Tests produce consistent results
- **Cleanup**: Test database is reset via global setup

## Key Assertions

### Quantity Verification
```typescript
await expect(page.locator('text="On Hand"').locator('..').locator('text="50"')).toBeVisible();
```

### Transaction History
```typescript
await expect(page.locator('table').locator('text="ADJUST"')).toBeVisible();
await expect(page.locator('table').locator('text="INV-ADJ-001"')).toBeVisible();
```

### Low Stock Alert
```typescript
await expect(page.locator('text="LOW"')).toBeVisible();
await expect(onHandCard).toHaveClass(/border-red-200|bg-red-50/);
```

## Waiting Strategies

The tests use multiple waiting strategies for reliability:

1. **Network Idle**: Waits for network requests to complete
2. **Explicit Waits**: Waits for specific elements to appear
3. **Dialog State**: Waits for dialogs to open/close completely
4. **Timeout Buffers**: Brief waits after actions for backend processing

## Known Considerations

- Tests require ZRP backend running on `localhost:9000`
- Admin credentials must be `admin`/`changeme`
- Test database is at `/tmp/zrp-test/zrp-test.db`
- Tests run sequentially (workers: 1) to avoid conflicts

## Troubleshooting

### Timeouts
If tests timeout, check:
- Backend server is running on port 9000
- Database is properly initialized
- Network latency isn't causing slow responses

### Selector Issues
If elements aren't found:
- Verify UI hasn't changed (check screenshots in `test-results/`)
- Check browser console for JavaScript errors
- Ensure proper login and navigation

### Data Issues
If quantities don't match:
- Check transaction history for all operations
- Verify backend logs for errors
- Ensure database transactions are committing

## Future Enhancements

Potential additions to test coverage:
- Negative quantity validation
- Concurrent user scenarios
- Barcode scanning integration
- Export/import functionality
- Audit trail verification
- Permission-based access control

## Success Criteria Met

✅ Stock adjustments (add/remove) tested  
✅ Transaction history verified  
✅ Reorder point/quantity changes tested  
✅ Location changes tested  
✅ Tests are reliable and deterministic  
✅ Comprehensive coverage of critical workflow  
✅ Clear documentation for maintenance  

## Contact

For questions or issues with these tests, refer to the ZRP development team or check the main project documentation.
