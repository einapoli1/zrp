import { test, expect } from '@playwright/test';

test.describe('ZRP Smoke Tests', () => {
  test('dashboard loads', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to parts page', async ({ page }) => {
    await page.goto('/parts');
    await expect(page.locator('text=Parts')).toBeVisible();
  });

  test('navigate to ECOs page', async ({ page }) => {
    await page.goto('/ecos');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to work orders page', async ({ page }) => {
    await page.goto('/work-orders');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to inventory page', async ({ page }) => {
    await page.goto('/inventory');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to vendors page', async ({ page }) => {
    await page.goto('/vendors');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to procurement page', async ({ page }) => {
    await page.goto('/procurement');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to devices page', async ({ page }) => {
    await page.goto('/devices');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to firmware page', async ({ page }) => {
    await page.goto('/firmware');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to testing page', async ({ page }) => {
    await page.goto('/testing');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to NCRs page', async ({ page }) => {
    await page.goto('/ncrs');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to RMAs page', async ({ page }) => {
    await page.goto('/rmas');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to quotes page', async ({ page }) => {
    await page.goto('/quotes');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to documents page', async ({ page }) => {
    await page.goto('/documents');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to calendar page', async ({ page }) => {
    await page.goto('/calendar');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to users page', async ({ page }) => {
    await page.goto('/users');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to audit page', async ({ page }) => {
    await page.goto('/audit');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to API keys page', async ({ page }) => {
    await page.goto('/api-keys');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to email settings page', async ({ page }) => {
    await page.goto('/email-settings');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('navigate to reports page', async ({ page }) => {
    await page.goto('/reports');
    await expect(page.locator('h1')).toBeVisible();
  });
});
