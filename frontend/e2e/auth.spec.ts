import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  // Clear cookies before each test to ensure clean state
  test.beforeEach(async ({ context }) => {
    await context.clearCookies();
  });

  test('should show login page when not authenticated', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('h1')).toContainText('ZRP');
    await expect(page.locator('text=Sign in to your account')).toBeVisible();
  });

  test('should reject invalid credentials', async ({ page }) => {
    await page.goto('/login');
    
    // Fill in invalid credentials using correct selectors
    await page.fill('#username', 'wronguser');
    await page.fill('#password', 'wrongpass');
    await page.click('button[type="submit"]');
    
    // Should show error message
    await expect(page.locator('text=/invalid/i')).toBeVisible();
  });

  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/login');
    
    // Fill in default admin credentials
    await page.fill('#username', 'admin');
    await page.fill('#password', 'changeme');
    await page.click('button[type="submit"]');
    
    // Should redirect to dashboard
    await expect(page).toHaveURL(/dashboard/);
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });

  test('should logout successfully', async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.fill('#username', 'admin');
    await page.fill('#password', 'changeme');
    await page.click('button[type="submit"]');
    
    // Wait for dashboard
    await expect(page).toHaveURL(/dashboard/);
    
    // Click user menu button (has aria-label="User menu")
    await page.click('button[aria-label="User menu"]');
    
    // Click logout in the dropdown
    await page.click('text=Log out');
    
    // Should redirect to login
    await expect(page).toHaveURL(/login/);
  });
});
