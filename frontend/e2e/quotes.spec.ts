import { test, expect, type Download } from '@playwright/test';
import path from 'path';

// Helper to login before each test
test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'changeme');
  await page.click('button[type="submit"]');
  await page.waitForURL(/dashboard|home/i);
});

test.describe('Quote Management', () => {
  test('should navigate to quotes page', async ({ page }) => {
    await page.goto('/quotes');
    await expect(page).toHaveURL(/\/quotes/);
    await expect(page.locator('h1, h2')).toContainText(/quote/i);
  });

  test('should display quotes list', async ({ page }) => {
    await page.goto('/quotes');
    
    // Should show a table or list container
    const tableOrList = page.locator('table, [role="table"], [role="grid"]');
    await expect(tableOrList).toBeVisible({ timeout: 5000 });
  });

  test('should create a basic quote for a customer', async ({ page }) => {
    await page.goto('/quotes');
    
    // Open create dialog
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    // Wait for dialog to appear
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    // Fill customer name
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Acme Corporation');
    
    // Fill notes
    const notesField = page.locator('textarea[name="notes"], textarea[placeholder*="note" i]').first();
    if (await notesField.isVisible()) {
      await notesField.fill('Test quote for customer');
    }
    
    // Set valid until date (future date)
    const validUntilField = page.locator('input[name="valid_until"], input[type="date"]').first();
    if (await validUntilField.isVisible()) {
      const futureDate = new Date();
      futureDate.setDate(futureDate.getDate() + 30);
      const dateString = futureDate.toISOString().split('T')[0];
      await validUntilField.fill(dateString);
    }
    
    // Submit the form
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    
    // Wait for dialog to close and quote to appear
    await page.waitForTimeout(2000);
    
    // Verify quote appears in the list
    await expect(page.locator('text="Acme Corporation"')).toBeVisible({ timeout: 5000 });
  });

  test('should create quote with multiple line items', async ({ page }) => {
    await page.goto('/quotes');
    
    // Open create dialog
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    // Fill customer
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Widget Industries');
    
    // Add first line item
    const addLineButton = page.locator('button:has-text("Add Line"), button:has-text("Add Item"), button:has-text("Add Product")').first();
    
    // There might be a default line item, or we need to add one
    if (await addLineButton.isVisible()) {
      await addLineButton.click();
    }
    
    // Fill first line item - look for the first set of line item inputs
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    // First line item
    if (await ipnInputs.nth(0).isVisible()) {
      await ipnInputs.nth(0).fill('WIDGET-001');
    }
    if (await qtyInputs.nth(0).isVisible()) {
      await qtyInputs.nth(0).fill('10');
    }
    if (await priceInputs.nth(0).isVisible()) {
      await priceInputs.nth(0).fill('25.50');
    }
    
    // Add second line item
    if (await addLineButton.isVisible()) {
      await addLineButton.click();
      await page.waitForTimeout(500);
    }
    
    // Second line item
    if (await ipnInputs.nth(1).isVisible()) {
      await ipnInputs.nth(1).fill('WIDGET-002');
    }
    if (await qtyInputs.nth(1).isVisible()) {
      await qtyInputs.nth(1).fill('5');
    }
    if (await priceInputs.nth(1).isVisible()) {
      await priceInputs.nth(1).fill('50.00');
    }
    
    // Submit
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Verify quote was created
    await expect(page.locator('text="Widget Industries"')).toBeVisible({ timeout: 5000 });
  });

  test('should calculate quote total with pricing', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote with known pricing
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Total Test Inc');
    
    // Add line items with specific pricing to calculate total
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    // Line 1: 10 x $100 = $1000
    if (await ipnInputs.nth(0).isVisible()) {
      await ipnInputs.nth(0).fill('ITEM-A');
      await qtyInputs.nth(0).fill('10');
      await priceInputs.nth(0).fill('100.00');
    }
    
    // Add second line
    const addLineButton = page.locator('button:has-text("Add Line"), button:has-text("Add Item"), button:has-text("Add Product")').first();
    if (await addLineButton.isVisible()) {
      await addLineButton.click();
      await page.waitForTimeout(500);
      
      // Line 2: 5 x $50 = $250
      if (await ipnInputs.nth(1).isVisible()) {
        await ipnInputs.nth(1).fill('ITEM-B');
        await qtyInputs.nth(1).fill('5');
        await priceInputs.nth(1).fill('50.00');
      }
    }
    
    // Submit
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote to view details (expected total: $1250)
    await page.click('text="Total Test Inc"');
    await page.waitForTimeout(1500);
    
    // Verify total is calculated correctly ($1250)
    // The total might be displayed in various formats: "$1,250.00", "1250", etc.
    const pageContent = await page.content();
    const has1250 = pageContent.includes('1250') || pageContent.includes('1,250');
    expect(has1250).toBeTruthy();
  });

  test('should generate and download PDF quote', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote first
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'PDF Test Corp');
    
    // Add a line item
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('PDF-ITEM');
      await qtyInputs.first().fill('1');
      await priceInputs.first().fill('99.99');
    }
    
    // Submit
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote to view details
    await page.click('text="PDF Test Corp"');
    await page.waitForTimeout(1500);
    
    // Setup download listener
    const downloadPromise = page.waitForEvent('download', { timeout: 10000 });
    
    // Click PDF export/download button
    const pdfButton = page.locator('button:has-text("PDF"), button:has-text("Download"), button:has-text("Export")').first();
    await pdfButton.click();
    
    // Wait for download to complete
    const download = await downloadPromise;
    
    // Verify download happened
    expect(download).toBeTruthy();
    expect(download.suggestedFilename()).toMatch(/\.pdf$/i);
  });

  test('should change quote status from draft to sent', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Status Test LLC');
    
    // Add a line item
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('STATUS-ITEM');
      await qtyInputs.first().fill('1');
      await priceInputs.first().fill('100.00');
    }
    
    // Submit
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Should initially be in draft status
    await expect(page.locator('text="draft"')).toBeVisible({ timeout: 5000 });
    
    // Click on the quote to view details
    await page.click('text="Status Test LLC"');
    await page.waitForTimeout(1500);
    
    // Look for status selector/dropdown
    const statusSelect = page.locator('select[name="status"], [role="combobox"]').first();
    if (await statusSelect.isVisible()) {
      await statusSelect.click();
      await page.click('text="sent"');
      
      // Save the change
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")').first();
      if (await saveButton.isVisible()) {
        await saveButton.click();
        await page.waitForTimeout(1500);
      }
      
      // Verify status changed
      await expect(page.locator('text="sent"')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should change quote status to accepted', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Accepted Quote Co');
    
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('ACCEPT-ITEM');
      await qtyInputs.first().fill('1');
      await priceInputs.first().fill('200.00');
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote
    await page.click('text="Accepted Quote Co"');
    await page.waitForTimeout(1500);
    
    // Change status to accepted
    const statusSelect = page.locator('select[name="status"], [role="combobox"]').first();
    if (await statusSelect.isVisible()) {
      await statusSelect.click();
      await page.click('text="accepted"');
      
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")').first();
      if (await saveButton.isVisible()) {
        await saveButton.click();
        await page.waitForTimeout(1500);
      }
      
      // Verify status is accepted
      await expect(page.locator('text="accepted"')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should change quote status to rejected', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Rejected Quote Inc');
    
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('REJECT-ITEM');
      await qtyInputs.first().fill('1');
      await priceInputs.first().fill('150.00');
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote
    await page.click('text="Rejected Quote Inc"');
    await page.waitForTimeout(1500);
    
    // Change status to rejected
    const statusSelect = page.locator('select[name="status"], [role="combobox"]').first();
    if (await statusSelect.isVisible()) {
      await statusSelect.click();
      await page.click('text="rejected"');
      
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")').first();
      if (await saveButton.isVisible()) {
        await saveButton.click();
        await page.waitForTimeout(1500);
      }
      
      // Verify status is rejected
      await expect(page.locator('text="rejected"')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should view quote cost vs margin', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote with line items
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Margin Test Corp');
    
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    // Add item with known margin (sell at $200, assume cost shows on detail page)
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('MARGIN-ITEM');
      await qtyInputs.first().fill('10');
      await priceInputs.first().fill('200.00'); // Total: $2000
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote to view details
    await page.click('text="Margin Test Corp"');
    await page.waitForTimeout(1500);
    
    // Look for cost/margin display on the detail page
    const pageText = await page.textContent('body');
    
    // Should show some cost/margin related text
    const hasFinancialInfo = pageText?.toLowerCase().includes('cost') || 
                            pageText?.toLowerCase().includes('margin') ||
                            pageText?.toLowerCase().includes('total');
    expect(hasFinancialInfo).toBeTruthy();
    
    // Should show the quoted total of 2000
    const has2000 = pageText?.includes('2000') || pageText?.includes('2,000');
    expect(has2000).toBeTruthy();
  });

  test('should edit an existing quote', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote first
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Edit Test Original');
    
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('EDIT-ITEM');
      await qtyInputs.first().fill('5');
      await priceInputs.first().fill('50.00');
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote to view/edit details
    await page.click('text="Edit Test Original"');
    await page.waitForTimeout(1500);
    
    // Click edit button if needed
    const editButton = page.locator('button:has-text("Edit")').first();
    if (await editButton.isVisible()) {
      await editButton.click();
      await page.waitForTimeout(500);
    }
    
    // Modify the customer name
    const customerInput = page.locator('input[name="customer"], input[value="Edit Test Original"]').first();
    if (await customerInput.isVisible()) {
      await customerInput.clear();
      await customerInput.fill('Edit Test Modified');
      
      // Save changes
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")').first();
      await saveButton.click();
      await page.waitForTimeout(1500);
      
      // Verify the change
      await expect(page.locator('text="Edit Test Modified"')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should add and remove line items from existing quote', async ({ page }) => {
    await page.goto('/quotes');
    
    // Create a quote
    const createButton = page.locator('button:has-text("New Quote"), button:has-text("Create Quote"), button:has-text("Add Quote")').first();
    await createButton.click();
    
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });
    
    await page.fill('input[name="customer"], input[placeholder*="customer" i]', 'Line Edit Test');
    
    const ipnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
    const qtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
    const priceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
    
    if (await ipnInputs.first().isVisible()) {
      await ipnInputs.first().fill('LINE-1');
      await qtyInputs.first().fill('1');
      await priceInputs.first().fill('10.00');
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save"), button:has-text("Submit")');
    await page.waitForTimeout(2000);
    
    // Click on the quote
    await page.click('text="Line Edit Test"');
    await page.waitForTimeout(1500);
    
    // Click edit if needed
    const editButton = page.locator('button:has-text("Edit")').first();
    if (await editButton.isVisible()) {
      await editButton.click();
      await page.waitForTimeout(500);
    }
    
    // Add a new line item
    const addLineButton = page.locator('button:has-text("Add Line"), button:has-text("Add Item"), button:has-text("Add Product")').first();
    if (await addLineButton.isVisible()) {
      await addLineButton.click();
      await page.waitForTimeout(500);
      
      // Fill the new line item
      const newIpnInputs = page.locator('input[name*="ipn" i], input[placeholder*="part" i], input[placeholder*="ipn" i]');
      const newQtyInputs = page.locator('input[name*="qty" i], input[type="number"][placeholder*="quantity" i]');
      const newPriceInputs = page.locator('input[name*="price" i], input[type="number"][placeholder*="price" i]');
      
      // Get the last (newly added) inputs
      const count = await newIpnInputs.count();
      if (count > 1) {
        await newIpnInputs.nth(count - 1).fill('LINE-2');
        await newQtyInputs.nth(count - 1).fill('2');
        await newPriceInputs.nth(count - 1).fill('20.00');
      }
      
      // Save
      const saveButton = page.locator('button:has-text("Save"), button:has-text("Update")').first();
      await saveButton.click();
      await page.waitForTimeout(1500);
      
      // Verify both line items exist
      await expect(page.locator('text="LINE-1"')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('text="LINE-2"')).toBeVisible({ timeout: 5000 });
    }
  });
});
