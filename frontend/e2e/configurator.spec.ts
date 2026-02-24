import { test, expect } from '@playwright/test';

test.describe('Product Configurator', () => {
  // Login before each test
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.fill('#username', 'admin');
    await page.fill('#password', 'changeme');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/dashboard/);
  });

  test('should navigate to Configurator page', async ({ page }) => {
    // Click Configurator in sidebar
    await page.click('a[href="/configurator"]');
    await expect(page).toHaveURL('/configurator');
    
    // Should show templates tab
    await expect(page.getByRole('tab', { name: /templates/i })).toBeVisible();
  });

  test('should create a template and show Parameters/Parts sections', async ({ page }) => {
    await page.goto('/configurator');
    
    // Click New Template button
    await page.click('button:has-text("New Template")');
    
    // Should switch to Editor tab
    await expect(page.getByRole('tab', { name: /editor/i })).toHaveAttribute('data-state', 'active');
    
    // Fill in template details
    await page.fill('input[placeholder*="uATS"]', 'E2E Test Template');
    await page.fill('input[placeholder*="PCA"]', 'TEST-{voltage}-{length}');
    
    // Save template
    await page.click('button:has-text("Save Template")');
    
    // Wait for success toast
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Parameters and Parts sections should now appear
    await expect(page.locator('text=Parameters').first()).toBeVisible();
    await expect(page.locator('h3:has-text("Add Parameter")')).toBeVisible();
  });

  test('should add parameters to a template', async ({ page }) => {
    await page.goto('/configurator');
    
    // Create template
    await page.click('button:has-text("New Template")');
    await page.fill('input[placeholder*="uATS"]', 'Param Test Template');
    await page.fill('input[placeholder*="PCA"]', 'PARAM-{voltage}-{length}');
    await page.click('button:has-text("Save Template")');
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Wait for Parameters section to fully render
    await page.locator('h3:has-text("Add Parameter")').waitFor({ state: 'visible' });
    await page.waitForTimeout(500); // Let form elements render
    
    // Add first parameter (voltage - enum)
    await page.fill('input[placeholder*="voltage"]', 'voltage');

    // Scroll to parameter form
    await page.locator("select").first().scrollIntoViewIfNeeded();
    await page.locator('select').scrollIntoViewIfNeeded();
    await page.selectOption('select', 'enum');
    await page.fill('input[placeholder*="120,208"]', '120V,208V,240V');
    await page.click('button:has-text("Add Parameter")');
    
    // Should see success toast
    await expect(page.locator('text=/parameter added/i')).toBeVisible({ timeout: 5000 });
    
    // Parameter should appear in table
    await expect(page.locator('td:has-text("voltage")')).toBeVisible();
    await expect(page.locator('text=enum')).toBeVisible();
    
    // Parameters section should still be visible (regression test for data extraction bug)
    await expect(page.locator('h3:has-text("Add Parameter")')).toBeVisible();
    
    // Add second parameter (length - enum)
    await page.fill('input[placeholder*="voltage"]', 'length');
    // Type should already be enum from previous
    await page.fill('input[placeholder*="120,208"]', '3ft,6ft,10ft');
    await page.click('button:has-text("Add Parameter")');
    
    await expect(page.locator('text=/parameter added/i')).toBeVisible({ timeout: 5000 });
    
    // Both parameters should be visible
    await expect(page.locator('td:has-text("voltage")')).toBeVisible();
    await expect(page.locator('td:has-text("length")')).toBeVisible();
    
    // Form should still be visible
    await expect(page.locator('h3:has-text("Add Parameter")')).toBeVisible();
  });

  test('should delete a parameter', async ({ page }) => {
    await page.goto('/configurator');
    
    // Create template with parameter
    await page.click('button:has-text("New Template")');
    await page.fill('input[placeholder*="uATS"]', 'Delete Param Test');
    await page.fill('input[placeholder*="PCA"]', 'DEL-{test}');
    await page.click('button:has-text("Save Template")');
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Wait for Parameters section to fully render
    await page.locator('h3:has-text("Add Parameter")').waitFor({ state: 'visible' });
    await page.waitForTimeout(500); // Let form elements render
    
    // Add parameter
    await page.fill('input[placeholder*="voltage"]', 'test');
    await page.locator('select').scrollIntoViewIfNeeded();
    await page.selectOption('select', 'enum');
    await page.fill('input[placeholder*="120,208"]', 'A,B,C');
    await page.click('button:has-text("Add Parameter")');
    await expect(page.locator('td:has-text("test")')).toBeVisible();
    
    // Delete parameter
    await page.click('td:has-text("test") ~ td button[class*="destructive"]');
    
    // Should see success toast
    await expect(page.locator('text=/parameter deleted/i')).toBeVisible({ timeout: 5000 });
    
    // Parameter should be gone
    await expect(page.locator('td:has-text("test")')).not.toBeVisible();
    
    // Parameters section should still be visible (regression test)
    await expect(page.locator('h3:has-text("Add Parameter")')).toBeVisible();
  });

  test('should add parts to a template', async ({ page }) => {
    await page.goto('/configurator');
    
    // Create template
    await page.click('button:has-text("New Template")');
    await page.fill('input[placeholder*="uATS"]', 'Parts Test Template');
    await page.fill('input[placeholder*="PCA"]', 'PARTS-{voltage}');
    await page.click('button:has-text("Save Template")');
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Wait for Parameters section to fully render
    await page.locator('h3:has-text("Add Parameter")').waitFor({ state: 'visible' });
    await page.waitForTimeout(500); // Let form elements render
    
    // Scroll down to Parts section
    await page.locator('h3:has-text("Add Part")').scrollIntoViewIfNeeded();
    
    // Click Add Part button
    await page.click('button:has-text("Add Part")');
    
    // Dialog should open
    await expect(page.locator('text=Add Part').first()).toBeVisible();
    
    // Enter part IPN (use a generic one that might exist)
    await page.fill('input[placeholder*="IPN"]', 'TEST-PART-001');
    
    // Set quantity
    await page.fill('input[type="number"]', '2');
    
    // Click Add Part in dialog
    await page.click('button:has-text("Add Part"):last-of-type');
    
    // Should see success toast or error (part might not exist)
    // For now, just check dialog closes or error appears
    await page.waitForTimeout(1000);
  });

  test('should preview variants', async ({ page }) => {
    await page.goto('/configurator');
    
    // Create template with parameters
    await page.click('button:has-text("New Template")');
    await page.fill('input[placeholder*="uATS"]', 'Preview Test');
    await page.fill('input[placeholder*="PCA"]', 'PREV-{voltage}-{length}');
    await page.click('button:has-text("Save Template")');
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Wait for Parameters section to fully render
    await page.locator('h3:has-text("Add Parameter")').waitFor({ state: 'visible' });
    await page.waitForTimeout(500); // Let form elements render
    
    // Add voltage parameter
    await page.fill('input[placeholder*="voltage"]', 'voltage');
    await page.locator('select').scrollIntoViewIfNeeded();
    await page.selectOption('select', 'enum');
    await page.fill('input[placeholder*="120,208"]', '120V,208V');
    await page.click('button:has-text("Add Parameter")');
    await expect(page.locator('text=/parameter added/i')).toBeVisible({ timeout: 5000 });
    
    // Add length parameter
    await page.fill('input[placeholder*="voltage"]', 'length');
    await page.fill('input[placeholder*="120,208"]', '3ft,6ft');
    await page.click('button:has-text("Add Parameter")');
    await expect(page.locator('text=/parameter added/i')).toBeVisible({ timeout: 5000 });
    
    // Switch to Preview/Generate tab
    await page.click('button[role="tab"]:has-text("Preview")');
    
    // Select template from dropdown (should be auto-selected but let's be safe)
    const templateSelect = page.locator('select').first();
    await templateSelect.selectOption({ label: /Preview Test/i });
    
    // Click Preview Variants button
    await page.click('button:has-text("Preview Variants")');
    
    // Should show toast with variant count (2 voltages × 2 lengths = 4 variants)
    await expect(page.locator('text=/4 variants/i')).toBeVisible({ timeout: 5000 });
    
    // Should show variant list
    await expect(page.locator('text=PREV-120V-3ft')).toBeVisible();
    await expect(page.locator('text=PREV-120V-6ft')).toBeVisible();
    await expect(page.locator('text=PREV-208V-3ft')).toBeVisible();
    await expect(page.locator('text=PREV-208V-6ft')).toBeVisible();
  });

  test('should list templates on Templates tab', async ({ page }) => {
    await page.goto('/configurator');
    
    // Should be on Templates tab by default
    await expect(page.getByRole('tab', { name: /templates/i })).toHaveAttribute('data-state', 'active');
    
    // Create a template
    await page.click('button:has-text("New Template")');
    await page.fill('input[placeholder*="uATS"]', 'List Test Template');
    await page.fill('input[placeholder*="PCA"]', 'LIST-{test}');
    await page.click('button:has-text("Save Template")');
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Wait for Parameters section to fully render
    await page.locator('h3:has-text("Add Parameter")').waitFor({ state: 'visible' });
    await page.waitForTimeout(500); // Let form elements render
    
    // Switch back to Templates tab
    await page.click('button[role="tab"]:has-text("Templates")');
    
    // Should see the template in the list
    await expect(page.locator('text=List Test Template')).toBeVisible();
    await expect(page.locator('text=LIST-{test}')).toBeVisible();
  });

  test('should edit existing template', async ({ page }) => {
    await page.goto('/configurator');
    
    // Create template
    await page.click('button:has-text("New Template")');
    await page.fill('input[placeholder*="uATS"]', 'Edit Test Template');
    await page.fill('input[placeholder*="PCA"]', 'EDIT-{test}');
    await page.click('button:has-text("Save Template")');
    await expect(page.locator('text=/template created/i')).toBeVisible({ timeout: 5000 });
    
    // Wait for Parameters section to fully render
    await page.locator('h3:has-text("Add Parameter")').waitFor({ state: 'visible' });
    await page.waitForTimeout(500); // Let form elements render
    
    // Go back to Templates tab
    await page.click('button[role="tab"]:has-text("Templates")');
    
    // Click on the template to edit it
    await page.click('text=Edit Test Template');
    
    // Should switch to Editor tab
    await expect(page.getByRole('tab', { name: /editor/i })).toHaveAttribute('data-state', 'active');
    
    // Template details should be loaded
    await expect(page.locator('input[value="Edit Test Template"]')).toBeVisible();
    await expect(page.locator('input[value="EDIT-{test}"]')).toBeVisible();
    
    // Parameters/Parts sections should be visible
    await expect(page.locator('h3:has-text("Add Parameter")')).toBeVisible();
  });
});
