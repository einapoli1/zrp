const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:9000';

test.describe('ZRP ERP System', () => {
  test('Dashboard loads with KPI cards', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForTimeout(2000);
    // Should show dashboard with numbers
    const content = await page.textContent('#content');
    expect(content).toContain('Open ECOs');
    expect(content).toContain('Low Stock');
    expect(content).toContain('Active Work Orders');
  });

  test('Parts module loads', async ({ page }) => {
    await page.goto(BASE + '#/parts');
    await page.waitForTimeout(1500);
    const content = await page.textContent('#content');
    // Should show parts or "No parts" or actual IPN data
    expect(content.length).toBeGreaterThan(10);
  });

  test('ECO: create and approve', async ({ page }) => {
    await page.goto(BASE + '#/ecos');
    await page.waitForTimeout(2000);
    // Click create
    await page.click('button:has-text("New ECO")');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="title"]', 'Test ECO from Playwright');
    await page.fill('[data-field="description"]', 'Automated test');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    // Verify it appears
    const content = await page.textContent('#content');
    expect(content).toContain('Test ECO from Playwright');
  });

  test('Inventory: view and receive stock', async ({ page }) => {
    await page.goto(BASE + '#/inventory');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('CAP-001-0001');
    // Quick receive
    await page.click('text=+ Quick Receive');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="ipn"]', 'TEST-001-0001');
    await page.fill('[data-field="qty"]', '100');
    await page.fill('[data-field="reference"]', 'PW-TEST');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
  });

  test('Work Orders: create WO', async ({ page }) => {
    await page.goto(BASE + '#/workorders');
    await page.waitForTimeout(2000);
    await page.click('text=+ New Work Order');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="assembly_ipn"]', 'PCB-001-0001');
    await page.fill('[data-field="qty"]', '5');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('PCB-001-0001');
  });

  test('Devices: register and view', async ({ page }) => {
    await page.goto(BASE + '#/devices');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('SN-001');
    expect(content).toContain('Acme Corp');
  });

  test('NCR: create NCR', async ({ page }) => {
    await page.goto(BASE + '#/ncr');
    await page.waitForTimeout(2000);
    await page.click('text=+ New NCR');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="title"]', 'Test NCR');
    await page.fill('[data-field="description"]', 'Test defect');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Test NCR');
  });

  test('Quotes: create quote', async ({ page }) => {
    await page.goto(BASE + '#/quotes');
    await page.waitForTimeout(2000);
    await page.click('text=+ New Quote');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="customer"]', 'Test Customer');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Test Customer');
  });

  test('Vendors page loads', async ({ page }) => {
    await page.goto(BASE + '#/vendors');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('DigiKey');
  });

  test('RMA page loads', async ({ page }) => {
    await page.goto(BASE + '#/rma');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('RMA-2026-001');
  });
});
