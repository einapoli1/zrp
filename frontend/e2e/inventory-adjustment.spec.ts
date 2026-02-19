import { test, expect } from '@playwright/test';

/**
 * E2E Tests for Manual Inventory Adjustment
 * 
 * Tests the critical workflow for manually adjusting inventory quantities,
 * tracking transactions, and managing reorder points/locations.
 * 
 * Test Coverage:
 * - Manual stock additions (receive/adjust transactions)
 * - Manual stock removals (issue transactions)
 * - Inventory transaction history tracking
 * - Reorder point and quantity configuration
 * - Location changes
 * - Low stock alert triggers
 */

// Helper to login before each test
test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'changeme');
  await page.click('button[type="submit"]');
  await page.waitForURL(/dashboard|home/i);
});

test.describe('Inventory Adjustment', () => {
  
  test('should manually add stock to inventory item using adjust transaction', async ({ page }) => {
    // Navigate to inventory page
    await page.goto('/inventory');
    
    // Wait for page to load completely
    await page.waitForLoadState('networkidle');
    await expect(page.locator('h1')).toContainText(/inventory/i, { timeout: 10000 });
    
    // Create a test part first if needed (using Quick Receive to ensure inventory exists)
    const testIPN = `TEST-ADD-${Date.now()}`;
    
    // Click Quick Receive button
    const quickReceiveButton = page.locator('button:has-text("Quick Receive")');
    await quickReceiveButton.waitFor({ state: 'visible', timeout: 10000 });
    await quickReceiveButton.click();
    
    // Wait for dialog to open
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    
    // Fill in the quick receive form
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '10');
    await page.fill('input#reference', 'Initial stock');
    
    // Submit the form - find the button within the dialog
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    
    // Wait for dialog to close
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    
    // Wait for success toast
    await expect(page.locator('text=/received.*units/i')).toBeVisible({ timeout: 5000 });
    
    // Navigate to the inventory detail page
    await page.goto(`/inventory/${testIPN}`);
    
    // Wait for page to load
    await expect(page.locator('h1')).toContainText(testIPN);
    
    // Verify initial quantity
    await expect(page.locator('text="On Hand"').locator('..').locator('text="10"')).toBeVisible();
    
    // Click New Transaction button to add more stock
    await page.click('button:has-text("New Transaction")');
    
    // Wait for dialog to open
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    
    // Select "Adjust" transaction type
    await page.click('button[role="combobox"]');
    await page.click('text="Adjust"');
    
    // Enter new total quantity (adding 15 more: 10 + 15 = 25)
    await page.fill('input#qty', '25');
    await page.fill('input#reference', 'Manual adjustment - add stock');
    await page.fill('textarea#notes', 'Adding stock after physical count');
    
    // Submit the transaction
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    
    // Wait for the dialog to close and page to refresh
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Verify the new quantity is displayed
    await expect(page.locator('text="On Hand"').locator('..').locator('text="25"')).toBeVisible();
  });

  test('should manually remove stock from inventory item using issue transaction', async ({ page }) => {
    // Navigate to inventory page
    await page.goto('/inventory');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('h1')).toContainText(/inventory/i, { timeout: 10000 });
    
    // Create a test part with initial stock
    const testIPN = `TEST-REMOVE-${Date.now()}`;
    
    // Click Quick Receive button
    const quickReceiveButton = page.locator('button:has-text("Quick Receive")');
    await quickReceiveButton.waitFor({ state: 'visible', timeout: 10000 });
    await quickReceiveButton.click();
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    
    // Fill in the quick receive form with initial stock
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '50');
    await page.fill('input#reference', 'Initial stock for removal test');
    
    // Submit the form
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    
    // Wait for success
    await page.waitForTimeout(1500);
    
    // Navigate to the inventory detail page
    await page.goto(`/inventory/${testIPN}`);
    
    // Verify initial quantity
    await expect(page.locator('text="On Hand"').locator('..').locator('text="50"')).toBeVisible();
    
    // Click New Transaction button to remove stock
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    
    // Select "Issue" transaction type
    await page.click('button[role="combobox"]');
    await page.click('text="Issue"');
    
    // Enter quantity to remove
    await page.fill('input#qty', '20');
    await page.fill('input#reference', 'WO-12345');
    await page.fill('textarea#notes', 'Issued for production work order');
    
    // Submit the transaction
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    
    // Wait for the dialog to close and page to refresh
    await page.waitForTimeout(1500);
    
    // Verify the new quantity (50 - 20 = 30)
    await expect(page.locator('text="On Hand"').locator('..').locator('text="30"')).toBeVisible();
  });

  test('should verify inventory transaction history is recorded correctly', async ({ page }) => {
    // Create a test part and perform multiple transactions
    const testIPN = `TEST-HISTORY-${Date.now()}`;
    
    await page.goto('/inventory');
    
    // Initial receive
    await page.click('button:has-text("Quick Receive")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '100');
    await page.fill('input#reference', 'PO-001');
    await page.fill('input#notes', 'Initial inventory');
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Navigate to detail page
    await page.goto(`/inventory/${testIPN}`);
    
    // Perform an issue transaction
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Issue"');
    await page.fill('input#qty', '30');
    await page.fill('input#reference', 'WO-100');
    await page.fill('textarea#notes', 'Production issue');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Perform an adjust transaction
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Adjust"');
    await page.fill('input#qty', '75');
    await page.fill('input#reference', 'INV-ADJ-001');
    await page.fill('textarea#notes', 'Physical count adjustment');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Verify transaction history table shows all transactions
    await expect(page.locator('text="Transaction History"')).toBeVisible();
    
    // Check for RECEIVE transaction
    await expect(page.locator('table').locator('text="RECEIVE"')).toBeVisible();
    await expect(page.locator('table').locator('text="PO-001"')).toBeVisible();
    await expect(page.locator('table').locator('text="Initial inventory"')).toBeVisible();
    
    // Check for ISSUE transaction
    await expect(page.locator('table').locator('text="ISSUE"')).toBeVisible();
    await expect(page.locator('table').locator('text="WO-100"')).toBeVisible();
    await expect(page.locator('table').locator('text="Production issue"')).toBeVisible();
    
    // Check for ADJUST transaction
    await expect(page.locator('table').locator('text="ADJUST"')).toBeVisible();
    await expect(page.locator('table').locator('text="INV-ADJ-001"')).toBeVisible();
    await expect(page.locator('table').locator('text="Physical count adjustment"')).toBeVisible();
    
    // Verify final quantity is correct (75 after adjustment)
    await expect(page.locator('text="On Hand"').locator('..').locator('text="75"')).toBeVisible();
  });

  test('should set reorder point and reorder quantity using bulk edit', async ({ page }) => {
    // Create a test part
    const testIPN = `TEST-REORDER-${Date.now()}`;
    
    await page.goto('/inventory');
    
    // Create initial inventory
    await page.click('button:has-text("Quick Receive")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '50');
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Find and select the inventory item checkbox
    await page.goto('/inventory');
    
    // Find the row with our test IPN and click its checkbox
    const row = page.locator(`tr:has-text("${testIPN}")`);
    await row.locator('input[type="checkbox"]').click();
    
    // Verify selection count
    await expect(page.locator('text="Selected"').locator('..').locator('text="1"')).toBeVisible();
    
    // Click Bulk Edit button
    await page.click('button:has-text("Bulk Edit")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    
    // Fill in reorder point
    await page.fill('input[name="reorder_point"]', '20');
    
    // Fill in reorder quantity
    await page.fill('input[name="reorder_qty"]', '50');
    
    // Submit bulk edit
    await page.locator('div[role="dialog"]').locator('button:has-text("Update")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    
    // Wait for success
    await page.waitForTimeout(1500);
    
    // Navigate to detail page to verify
    await page.goto(`/inventory/${testIPN}`);
    
    // Verify reorder point is displayed
    await expect(page.locator('text="Reorder Point"').locator('..').locator('text="20"')).toBeVisible();
    
    // Verify reorder quantity in the details section
    await expect(page.locator('text="Reorder Quantity"').locator('..').locator('text="50"')).toBeVisible();
  });

  test('should change location for inventory item using bulk edit', async ({ page }) => {
    // Create a test part
    const testIPN = `TEST-LOCATION-${Date.now()}`;
    const newLocation = 'A-15-03';
    
    await page.goto('/inventory');
    
    // Create initial inventory
    await page.click('button:has-text("Quick Receive")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '25');
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Go back to inventory list
    await page.goto('/inventory');
    
    // Find and select the inventory item
    const row = page.locator(`tr:has-text("${testIPN}")`);
    await row.locator('input[type="checkbox"]').click();
    
    // Click Bulk Edit button
    await page.click('button:has-text("Bulk Edit")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    
    // Fill in new location
    await page.fill('input[name="location"]', newLocation);
    
    // Submit bulk edit
    await page.locator('div[role="dialog"]').locator('button:has-text("Update")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    
    // Wait for success
    await page.waitForTimeout(1500);
    
    // Verify location in the table
    await expect(page.locator(`tr:has-text("${testIPN}")`).locator(`text="${newLocation}"`)).toBeVisible();
    
    // Navigate to detail page to verify
    await page.goto(`/inventory/${testIPN}`);
    
    // Verify location in item details
    await expect(page.locator('text="Location"').locator('..').locator(`text="${newLocation}"`)).toBeVisible();
  });

  test('should verify low stock alerts trigger correctly when quantity falls below reorder point', async ({ page }) => {
    // Create a test part with reorder point
    const testIPN = `TEST-LOWSTOCK-${Date.now()}`;
    
    await page.goto('/inventory');
    
    // Create initial inventory with sufficient quantity
    await page.click('button:has-text("Quick Receive")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '30');
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Set reorder point to 25 (current qty is 30, so not low stock yet)
    await page.goto('/inventory');
    const row = page.locator(`tr:has-text("${testIPN}")`);
    await row.locator('input[type="checkbox"]').click();
    await page.click('button:has-text("Bulk Edit")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input[name="reorder_point"]', '25');
    await page.fill('input[name="reorder_qty"]', '50');
    await page.locator('div[role="dialog"]').locator('button:has-text("Update")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Navigate to detail page - should NOT show low stock alert yet
    await page.goto(`/inventory/${testIPN}`);
    
    // Verify no LOW badge initially (30 > 25)
    const lowBadgeCount = await page.locator('text="LOW"').count();
    expect(lowBadgeCount).toBe(0);
    
    // Issue stock to bring it below reorder point (30 - 10 = 20, which is < 25)
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Issue"');
    await page.fill('input#qty', '10');
    await page.fill('input#reference', 'Low stock test');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Verify LOW badge now appears
    await expect(page.locator('text="LOW"')).toBeVisible();
    
    // Verify the card has red styling (bg-red-50 or border-red-200)
    const onHandCard = page.locator('text="On Hand"').locator('../..');
    await expect(onHandCard).toHaveClass(/border-red-200|bg-red-50/);
    
    // Go to inventory list and click "Low Stock" filter
    await page.goto('/inventory');
    await page.click('button:has-text("Low Stock")');
    
    // Wait for filter to apply
    await page.waitForTimeout(1000);
    
    // Verify our test item appears in the filtered list
    await expect(page.locator(`text="${testIPN}"`)).toBeVisible();
    
    // Verify low stock count is updated
    await expect(page.locator('text="Low Stock Items"').locator('..').locator('text=/[1-9]/').first()).toBeVisible();
  });

  test('should persist quantity changes across page reloads', async ({ page }) => {
    // Create a test part and perform adjustments
    const testIPN = `TEST-PERSIST-${Date.now()}`;
    
    await page.goto('/inventory');
    
    // Create initial inventory
    await page.click('button:has-text("Quick Receive")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '100');
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Navigate to detail page
    await page.goto(`/inventory/${testIPN}`);
    
    // Perform adjustment
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Adjust"');
    await page.fill('input#qty', '150');
    await page.fill('input#reference', 'Persistence test');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Verify quantity
    await expect(page.locator('text="On Hand"').locator('..').locator('text="150"')).toBeVisible();
    
    // Reload the page
    await page.reload();
    
    // Wait for page to load
    await expect(page.locator('h1')).toContainText(testIPN);
    
    // Verify quantity persists after reload
    await expect(page.locator('text="On Hand"').locator('..').locator('text="150"')).toBeVisible();
    
    // Go back to inventory list
    await page.goto('/inventory');
    
    // Verify quantity persists in the table
    const tableRow = page.locator(`tr:has-text("${testIPN}")`);
    await expect(tableRow.locator('text="150"').first()).toBeVisible();
    
    // Reload inventory list page
    await page.reload();
    await page.waitForTimeout(1000);
    
    // Verify quantity still persists
    await expect(page.locator(`tr:has-text("${testIPN}")`).locator('text="150"').first()).toBeVisible();
  });

  test('should handle multiple sequential adjustments correctly', async ({ page }) => {
    // Test that multiple adjustments in sequence work properly
    const testIPN = `TEST-MULTI-${Date.now()}`;
    
    await page.goto('/inventory');
    
    // Create initial inventory
    await page.click('button:has-text("Quick Receive")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.fill('input#ipn', testIPN);
    await page.fill('input#qty', '10');
    await page.locator('div[role="dialog"]').locator('button:has-text("Receive")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    
    // Navigate to detail page
    await page.goto(`/inventory/${testIPN}`);
    
    // First adjustment: add stock
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Adjust"');
    await page.fill('input#qty', '50');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    await expect(page.locator('text="On Hand"').locator('..').locator('text="50"')).toBeVisible();
    
    // Second adjustment: reduce stock
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Adjust"');
    await page.fill('input#qty', '35');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    await expect(page.locator('text="On Hand"').locator('..').locator('text="35"')).toBeVisible();
    
    // Third adjustment: increase again
    await page.click('button:has-text("New Transaction")');
    await page.waitForSelector('div[role="dialog"]', { state: 'visible' });
    await page.click('button[role="combobox"]');
    await page.click('text="Adjust"');
    await page.fill('input#qty', '100');
    await page.locator('div[role="dialog"]').locator('button:has-text("Create Transaction")').last().click();
    await page.waitForSelector('div[role="dialog"]', { state: 'hidden', timeout: 5000 });
    await page.waitForTimeout(1500);
    await expect(page.locator('text="On Hand"').locator('..').locator('text="100"')).toBeVisible();
    
    // Verify all transactions appear in history
    const transactionTable = page.locator('table').filter({ hasText: 'Transaction History' });
    
    // Should have 4 transactions total (1 receive + 3 adjusts)
    const adjustRows = await transactionTable.locator('text="ADJUST"').count();
    expect(adjustRows).toBe(3);
    
    const receiveRows = await transactionTable.locator('text="RECEIVE"').count();
    expect(receiveRows).toBe(1);
  });
});
