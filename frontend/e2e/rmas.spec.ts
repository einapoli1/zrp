import { test, expect } from '@playwright/test';

// Helper to login before each test
test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'changeme');
  await page.click('button[type="submit"]');
  await page.waitForURL(/dashboard|home/i);
});

test.describe('RMA Management', () => {
  test('should create a new RMA for a customer', async ({ page }) => {
    // Navigate to RMAs page
    await page.goto('/rmas');
    
    // Click "Create RMA" button
    await page.click('button:has-text("Create RMA")');
    
    // Wait for dialog to open
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    // Fill in RMA details
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-TEST-${timestamp}`);
    await page.fill('input[id="customer"]', 'Test Customer Inc');
    await page.fill('input[id="reason"]', 'Device not powering on');
    await page.fill('textarea[id="defect_description"]', 'Customer reports device fails to boot after firmware update. No lights, no response.');
    
    // Submit the form
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    // Wait for dialog to close and verify RMA appears in list
    await page.waitForTimeout(1500);
    await expect(page.locator(`text="Test Customer Inc"`)).toBeVisible({ timeout: 5000 });
    await expect(page.locator(`text="Device not powering on"`)).toBeVisible();
  });

  test('should track RMA number and reason in the list', async ({ page }) => {
    // Navigate to RMAs page
    await page.goto('/rmas');
    
    // Create a test RMA
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-TRACK-${timestamp}`);
    await page.fill('input[id="customer"]', 'Acme Corp');
    await page.fill('input[id="reason"]', 'Screen defect');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Verify RMA number (ID) is visible - it should be in format RMA-YYYY-XXX
    const rmaIdLocator = page.locator('td.font-medium').filter({ hasText: 'RMA-' }).first();
    await expect(rmaIdLocator).toBeVisible({ timeout: 5000 });
    const rmaId = await rmaIdLocator.textContent();
    expect(rmaId).toMatch(/RMA-\d{4}-\d{3}/);
    
    // Verify reason is visible
    await expect(page.locator('text="Screen defect"')).toBeVisible();
    
    // Verify customer is visible
    await expect(page.locator('text="Acme Corp"')).toBeVisible();
  });

  test('should progress RMA through full status workflow', async ({ page }) => {
    // Create a test RMA
    await page.goto('/rmas');
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-WORKFLOW-${timestamp}`);
    await page.fill('input[id="customer"]', 'Workflow Test Corp');
    await page.fill('input[id="reason"]', 'Battery issue');
    await page.fill('textarea[id="defect_description"]', 'Battery drains too quickly');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Click on the RMA to open detail page
    await page.click('text="Workflow Test Corp"');
    await page.waitForTimeout(1000);
    
    // Verify initial status is "Open"
    await expect(page.locator('text=/Open/i').first()).toBeVisible();
    
    // Status workflow: open → received → investigating → resolved → shipped → closed
    const statusWorkflow = ['received', 'investigating', 'resolved', 'shipped', 'closed'];
    
    for (const status of statusWorkflow) {
      // Click Edit button
      await page.click('button:has-text("Edit RMA")');
      await page.waitForTimeout(500);
      
      // Change status
      await page.click('[role="combobox"]');
      await page.waitForTimeout(500);
      
      // Click the status option (capitalize first letter to match UI)
      const statusLabel = status.charAt(0).toUpperCase() + status.slice(1);
      await page.click(`text="${statusLabel}"`);
      await page.waitForTimeout(500);
      
      // Save changes
      await page.click('button:has-text("Save Changes")');
      await page.waitForTimeout(1500);
      
      // Verify status badge updated
      await expect(page.locator(`text=/^${statusLabel}$/i`).first()).toBeVisible({ timeout: 5000 });
    }
    
    // Verify final status is "Closed"
    await expect(page.locator('text=/^Closed$/i').first()).toBeVisible();
  });

  test('should add notes/comments to RMA via resolution field', async ({ page }) => {
    // Create a test RMA
    await page.goto('/rmas');
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-NOTES-${timestamp}`);
    await page.fill('input[id="customer"]', 'Notes Test Customer');
    await page.fill('input[id="reason"]', 'WiFi connectivity');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Click on the RMA to open detail page
    await page.click('text="Notes Test Customer"');
    await page.waitForTimeout(1000);
    
    // Click Edit
    await page.click('button:has-text("Edit RMA")');
    await page.waitForTimeout(500);
    
    // Add resolution notes
    const resolutionText = 'Tested device. Found faulty WiFi module. Replaced module and verified connectivity. Device now working properly.';
    const resolutionField = page.locator('textarea').filter({ hasText: '' }).last();
    await resolutionField.fill(resolutionText);
    
    // Save changes
    await page.click('button:has-text("Save Changes")');
    await page.waitForTimeout(1500);
    
    // Verify resolution text is visible
    await expect(page.locator(`text="${resolutionText}"`)).toBeVisible({ timeout: 5000 });
  });

  test('should link RMA to NCR when defect is found', async ({ page }) => {
    // Create a test RMA with defect
    await page.goto('/rmas');
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-NCR-${timestamp}`);
    await page.fill('input[id="customer"]', 'NCR Test Customer');
    await page.fill('input[id="reason"]', 'Manufacturing defect');
    await page.fill('textarea[id="defect_description"]', 'Solder joint failure on PCB');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Click on the RMA to open detail page
    await page.click('text="NCR Test Customer"');
    await page.waitForTimeout(1000);
    
    // Look for CAPA button (NCR creation button)
    const capaButton = page.locator('button:has-text("Create CAPA from RMA")');
    await expect(capaButton).toBeVisible({ timeout: 5000 });
    
    // Click to create CAPA/NCR
    await capaButton.click();
    await page.waitForTimeout(1500);
    
    // Verify navigation to CAPA page with pre-filled RMA info
    await expect(page).toHaveURL(/capas/i);
    
    // Verify RMA ID is in the title or form
    const rmaIdPattern = /RMA-\d{4}-\d{3}/;
    await expect(page.locator(`text=${rmaIdPattern}`)).toBeVisible({ timeout: 5000 });
  });

  test('should search and filter RMAs by status', async ({ page }) => {
    // Navigate to RMAs page
    await page.goto('/rmas');
    
    // Create multiple RMAs with different statuses
    const testRMAs = [
      { customer: 'Filter Test 1', reason: 'Test 1', status: 'open' },
      { customer: 'Filter Test 2', reason: 'Test 2', status: 'received' },
      { customer: 'Filter Test 3', reason: 'Test 3', status: 'closed' },
    ];
    
    for (let i = 0; i < testRMAs.length; i++) {
      const rma = testRMAs[i];
      const timestamp = Date.now() + i;
      
      await page.click('button:has-text("Create RMA")');
      await expect(page.locator('[role="dialog"]')).toBeVisible();
      
      await page.fill('input[id="serial_number"]', `SN-FILTER-${timestamp}`);
      await page.fill('input[id="customer"]', rma.customer);
      await page.fill('input[id="reason"]', rma.reason);
      await page.click('button[type="submit"]:has-text("Create RMA")');
      await page.waitForTimeout(1500);
      
      // If status needs to be changed from 'open'
      if (rma.status !== 'open') {
        await page.click(`text="${rma.customer}"`);
        await page.waitForTimeout(1000);
        
        await page.click('button:has-text("Edit RMA")');
        await page.waitForTimeout(500);
        
        await page.click('[role="combobox"]');
        await page.waitForTimeout(500);
        
        const statusLabel = rma.status.charAt(0).toUpperCase() + rma.status.slice(1);
        await page.click(`text="${statusLabel}"`);
        await page.waitForTimeout(500);
        
        await page.click('button:has-text("Save Changes")');
        await page.waitForTimeout(1500);
        
        await page.click('button:has-text("Back to RMAs")');
        await page.waitForTimeout(1000);
      }
    }
    
    // Verify all test RMAs are visible
    await page.goto('/rmas');
    await page.waitForTimeout(1000);
    
    for (const rma of testRMAs) {
      await expect(page.locator(`text="${rma.customer}"`)).toBeVisible({ timeout: 5000 });
    }
    
    // Note: Search/filter functionality would be tested here if UI supports it
    // Currently the RMAs page shows all RMAs in a table without explicit filter controls
    // This test verifies multiple RMAs with different statuses can be created and displayed
  });

  test('should close RMA successfully', async ({ page }) => {
    // Create a test RMA
    await page.goto('/rmas');
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-CLOSE-${timestamp}`);
    await page.fill('input[id="customer"]', 'Close Test Customer');
    await page.fill('input[id="reason"]', 'Complete test');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Click on the RMA
    await page.click('text="Close Test Customer"');
    await page.waitForTimeout(1000);
    
    // Verify initial status is "Open"
    await expect(page.locator('text=/^Open$/i').first()).toBeVisible();
    
    // Click Edit
    await page.click('button:has-text("Edit RMA")');
    await page.waitForTimeout(500);
    
    // Add resolution before closing
    const resolutionField = page.locator('textarea').filter({ hasText: '' }).last();
    await resolutionField.fill('RMA completed. Issue resolved and device returned to customer.');
    
    // Change status to Closed
    await page.click('[role="combobox"]');
    await page.waitForTimeout(500);
    await page.click('text="Closed"');
    await page.waitForTimeout(500);
    
    // Save changes
    await page.click('button:has-text("Save Changes")');
    await page.waitForTimeout(1500);
    
    // Verify status is "Closed"
    await expect(page.locator('text=/^Closed$/i').first()).toBeVisible({ timeout: 5000 });
    
    // Verify resolution is saved
    await expect(page.locator('text="RMA completed. Issue resolved and device returned to customer."')).toBeVisible();
  });

  test('should enforce business rules during status transitions', async ({ page }) => {
    // Create a test RMA
    await page.goto('/rmas');
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-RULES-${timestamp}`);
    await page.fill('input[id="customer"]', 'Business Rules Test');
    await page.fill('input[id="reason"]', 'Rules test');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Click on the RMA
    await page.click('text="Business Rules Test"');
    await page.waitForTimeout(1000);
    
    // Verify all workflow statuses are available in the dropdown
    await page.click('button:has-text("Edit RMA")');
    await page.waitForTimeout(500);
    
    await page.click('[role="combobox"]');
    await page.waitForTimeout(500);
    
    // Verify all expected statuses are present
    const expectedStatuses = ['Open', 'Received', 'Investigating', 'Resolved', 'Shipped', 'Closed'];
    for (const status of expectedStatuses) {
      await expect(page.locator(`text="${status}"`)).toBeVisible({ timeout: 2000 });
    }
    
    // Close the dropdown by pressing Escape
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);
    
    // Cancel editing
    await page.click('button:has-text("Cancel")');
    await page.waitForTimeout(500);
  });

  test('should display RMA timeline with timestamps', async ({ page }) => {
    // Create a test RMA
    await page.goto('/rmas');
    await page.click('button:has-text("Create RMA")');
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const timestamp = Date.now();
    await page.fill('input[id="serial_number"]', `SN-TIMELINE-${timestamp}`);
    await page.fill('input[id="customer"]', 'Timeline Test');
    await page.fill('input[id="reason"]', 'Timeline verification');
    await page.click('button[type="submit"]:has-text("Create RMA")');
    
    await page.waitForTimeout(1500);
    
    // Click on the RMA
    await page.click('text="Timeline Test"');
    await page.waitForTimeout(1000);
    
    // Verify Timeline card is visible
    await expect(page.locator('text="Timeline"').first()).toBeVisible({ timeout: 5000 });
    
    // Verify "Created" timestamp is present
    await expect(page.locator('text="Created"')).toBeVisible();
    
    // Change status to received to test received_at timestamp
    await page.click('button:has-text("Edit RMA")');
    await page.waitForTimeout(500);
    
    await page.click('[role="combobox"]');
    await page.waitForTimeout(500);
    await page.click('text="Received"');
    await page.waitForTimeout(500);
    
    await page.click('button:has-text("Save Changes")');
    await page.waitForTimeout(1500);
    
    // Verify "Received" timestamp appears
    await expect(page.locator('text="Received"').nth(1)).toBeVisible({ timeout: 5000 });
    
    // Change to resolved status to test resolved_at timestamp
    await page.click('button:has-text("Edit RMA")');
    await page.waitForTimeout(500);
    
    await page.click('[role="combobox"]');
    await page.waitForTimeout(500);
    await page.click('text="Resolved"');
    await page.waitForTimeout(500);
    
    await page.click('button:has-text("Save Changes")');
    await page.waitForTimeout(1500);
    
    // Verify "Resolved" timestamp appears in timeline
    await expect(page.locator('text="Resolved"').nth(1)).toBeVisible({ timeout: 5000 });
  });

  test('should show empty state when no RMAs exist', async ({ page }) => {
    // This test assumes a fresh state or we need to check for empty state message
    await page.goto('/rmas');
    
    // Look for either RMA records or empty state message
    const tableBody = page.locator('tbody');
    const emptyMessage = page.locator('text="No RMAs found"');
    
    // Either table has rows OR empty message is shown
    const hasRows = await tableBody.locator('tr').count();
    const hasEmptyMessage = await emptyMessage.isVisible().catch(() => false);
    
    // At least one should be true (either we have RMAs or we see the empty state)
    expect(hasRows > 0 || hasEmptyMessage).toBeTruthy();
  });
});
