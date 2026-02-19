import { test, expect } from '@playwright/test';

// Helper to login before each test
test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'changeme');
  await page.click('button[type="submit"]');
  await page.waitForURL(/dashboard|home/i);
});

test.describe('API Key Management', () => {
  test('should display API keys page', async ({ page }) => {
    // Navigate to API keys page
    await page.goto('/api-keys');
    
    // Verify page loaded
    await expect(page.locator('h1')).toContainText('API Keys');
    await expect(page.locator('text=Manage API keys for programmatic access')).toBeVisible();
    
    // Verify stats cards are present
    await expect(page.locator('text=Total Keys')).toBeVisible();
    await expect(page.locator('text=Active')).toBeVisible();
    await expect(page.locator('text=Revoked')).toBeVisible();
  });

  test('should generate new API key and display full key once', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Record initial key count
    const initialCountText = await page.locator('text=Total Keys').locator('..').locator('div').first().textContent();
    const initialCount = parseInt(initialCountText || '0');
    
    // Click generate new key button
    await page.click('button:has-text("Generate New Key")');
    
    // Verify dialog opened
    await expect(page.locator('text=Generate New API Key')).toBeVisible();
    await expect(page.locator('text=The full API key will only be shown once')).toBeVisible();
    
    // Fill in key name
    const keyName = `Test Key ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    
    // Click generate button
    await page.click('button:has-text("Generate Key")');
    
    // Wait for success message
    await expect(page.locator('text=API Key Generated Successfully')).toBeVisible({ timeout: 10000 });
    
    // Verify full key is displayed (zrp_ + 32 hex chars = 36 total)
    const fullKeyElement = page.locator('code').filter({ hasText: /^zrp_[a-f0-9]{32}$/ });
    await expect(fullKeyElement).toBeVisible();
    
    // Get the full key text
    const fullKey = await fullKeyElement.textContent();
    expect(fullKey).toMatch(/^zrp_[a-f0-9]{32}$/);
    
    // Verify key count increased
    const newCountText = await page.locator('text=Total Keys').locator('..').locator('div').first().textContent();
    const newCount = parseInt(newCountText || '0');
    expect(newCount).toBe(initialCount + 1);
    
    // Verify key appears in table with prefix masking
    await page.click('button:has-text("I\'ve copied the key")');
    
    // The key should now show only the prefix in the table
    const tableRow = page.locator(`tr:has-text("${keyName}")`);
    await expect(tableRow).toBeVisible();
    
    // Extract the prefix from the full key
    const keyPrefix = fullKey?.slice(0, 12) || ''; // zrp_ + first 8 hex chars
    const maskedKey = tableRow.locator('code').first();
    const maskedKeyText = await maskedKey.textContent();
    
    // Verify the key is masked (should show only prefix, not full key)
    expect(maskedKeyText).toBe(keyPrefix);
    expect(maskedKeyText).not.toBe(fullKey);
  });

  test('should copy API key to clipboard', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Generate a new key
    await page.click('button:has-text("Generate New Key")');
    const keyName = `Clipboard Test ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    
    // Wait for success message
    await expect(page.locator('text=API Key Generated Successfully')).toBeVisible({ timeout: 10000 });
    
    // Get the full key
    const fullKeyElement = page.locator('code').filter({ hasText: /^zrp_[a-f0-9]{32}$/ });
    const fullKey = await fullKeyElement.textContent();
    
    // Grant clipboard permissions
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
    
    // Click the copy button (within the success card)
    const successCard = page.locator('.border-green-200');
    const copyButton = successCard.locator('button').last();
    await copyButton.click();
    
    // Verify clipboard content
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toBe(fullKey);
  });

  test('should verify prefix masking in list view', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Generate a new key
    await page.click('button:has-text("Generate New Key")');
    const keyName = `Masking Test ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    
    // Get the full key
    const fullKeyElement = page.locator('code').filter({ hasText: /^zrp_[a-f0-9]{32}$/ });
    await expect(fullKeyElement).toBeVisible({ timeout: 10000 });
    const fullKey = await fullKeyElement.textContent();
    const keyPrefix = fullKey?.slice(0, 12) || ''; // zrp_ + first 8 hex chars
    
    // Hide the full key display
    await page.click('button:has-text("I\'ve copied the key")');
    
    // Verify the key is masked in the table
    const tableRow = page.locator(`tr:has-text("${keyName}")`);
    const maskedKey = tableRow.locator('code').first();
    const maskedKeyText = await maskedKey.textContent();
    
    // Should show prefix but not full key
    expect(maskedKeyText).toBe(keyPrefix);
    expect(maskedKeyText).not.toBe(fullKey);
    
    // Verify it follows the pattern: zrp_ + 8 hex chars
    expect(maskedKeyText).toMatch(/^zrp_[a-f0-9]{8}$/);
  });

  test('should authenticate API call with generated key', async ({ page, request }) => {
    await page.goto('/api-keys');
    
    // Generate a new key
    await page.click('button:has-text("Generate New Key")');
    const keyName = `API Auth Test ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    
    // Get the full key
    const fullKeyElement = page.locator('code').filter({ hasText: /^zrp_[a-f0-9]{32}$/ });
    await expect(fullKeyElement).toBeVisible({ timeout: 10000 });
    const fullKey = await fullKeyElement.textContent();
    
    // Make an authenticated API call using the key
    const apiResponse = await request.get('http://localhost:9000/api/api-keys', {
      headers: {
        'Authorization': `Bearer ${fullKey}`,
      },
    });
    
    // Verify the API call succeeds
    expect(apiResponse.status()).toBe(200);
    
    // Verify we get a valid response
    const data = await apiResponse.json();
    expect(Array.isArray(data)).toBe(true);
    
    // Verify our new key is in the response (check by name)
    const ourKey = data.find((k: any) => k.name === keyName);
    expect(ourKey).toBeDefined();
    expect(ourKey.enabled).toBe(1); // enabled=1 means active
  });

  test('should revoke API key', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Generate a new key to revoke
    await page.click('button:has-text("Generate New Key")');
    const keyName = `Revoke Test ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    
    // Wait for success message and then hide it
    await expect(page.locator('text=API Key Generated Successfully')).toBeVisible({ timeout: 10000 });
    await page.click('button:has-text("I\'ve copied the key")');
    
    // Find the key in the table
    const tableRow = page.locator(`tr:has-text("${keyName}")`);
    await expect(tableRow).toBeVisible();
    
    // Note: The UI shows badges, not plain text. Let's check for the badge
    const activeBadge = tableRow.locator('.bg-green-100');
    await expect(activeBadge).toBeVisible();
    
    // Click revoke button
    await tableRow.locator('button:has-text("Revoke")').click();
    
    // Verify revoke confirmation dialog
    await expect(page.locator('text=Revoke API Key')).toBeVisible();
    await expect(page.locator(`text=${keyName}`)).toBeVisible();
    await expect(page.locator('text=This action cannot be undone')).toBeVisible();
    
    // Confirm revocation
    await page.click('button:has-text("Yes, Revoke Key")');
    
    // Wait for dialog to close
    await expect(page.locator('text=Revoke API Key')).not.toBeVisible({ timeout: 5000 });
    
    // Verify status changed to revoked (check for red badge)
    const revokedBadge = tableRow.locator('.bg-red-100');
    await expect(revokedBadge).toBeVisible();
    
    // Verify revoke button is no longer available for this key
    await expect(tableRow.locator('button:has-text("Revoke")')).not.toBeVisible();
  });

  test('should verify revoked key no longer works', async ({ page, request }) => {
    await page.goto('/api-keys');
    
    // Generate a new key
    await page.click('button:has-text("Generate New Key")');
    const keyName = `Auth Revoke Test ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    
    // Get the full key
    const fullKeyElement = page.locator('code').filter({ hasText: /^zrp_[a-f0-9]{32}$/ });
    await expect(fullKeyElement).toBeVisible({ timeout: 10000 });
    const fullKey = await fullKeyElement.textContent();
    
    // Verify key works initially
    const initialResponse = await request.get('http://localhost:9000/api/api-keys', {
      headers: {
        'Authorization': `Bearer ${fullKey}`,
      },
    });
    expect(initialResponse.status()).toBe(200);
    
    // Hide the full key display
    await page.click('button:has-text("I\'ve copied the key")');
    
    // Revoke the key
    const tableRow = page.locator(`tr:has-text("${keyName}")`);
    await tableRow.locator('button:has-text("Revoke")').click();
    await page.click('button:has-text("Yes, Revoke Key")');
    
    // Wait for revocation to complete (check for red badge)
    const revokedBadge = tableRow.locator('.bg-red-100');
    await expect(revokedBadge).toBeVisible({ timeout: 5000 });
    
    // Try to use the revoked key
    const revokedResponse = await request.get('http://localhost:9000/api/api-keys', {
      headers: {
        'Authorization': `Bearer ${fullKey}`,
      },
    });
    
    // Verify the API call is rejected
    expect(revokedResponse.status()).toBe(401);
  });

  test('should list all API keys for user', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Get initial count
    const initialCountText = await page.locator('text=Total Keys').locator('..').locator('div').first().textContent();
    const initialCount = parseInt(initialCountText || '0');
    
    // Generate multiple keys
    const keyNames: string[] = [];
    for (let i = 0; i < 3; i++) {
      await page.click('button:has-text("Generate New Key")');
      const keyName = `List Test ${Date.now()}-${i}`;
      keyNames.push(keyName);
      await page.fill('input#key-name', keyName);
      await page.click('button:has-text("Generate Key")');
      await page.click('button:has-text("I\'ve copied the key")');
      
      // Small delay to ensure unique timestamps
      await page.waitForTimeout(100);
    }
    
    // Verify all keys are visible in the table
    for (const keyName of keyNames) {
      const row = page.locator(`tr:has-text("${keyName}")`);
      await expect(row).toBeVisible();
      
      // Verify each key has required columns
      await expect(row.locator('code')).toBeVisible(); // Key prefix
      await expect(row.locator('text=active')).toBeVisible(); // Status
      await expect(row.locator('text=admin')).toBeVisible(); // Created by
    }
    
    // Verify count updated correctly
    const finalCountText = await page.locator('text=Total Keys').locator('..').locator('div').first().textContent();
    const finalCount = parseInt(finalCountText || '0');
    expect(finalCount).toBe(initialCount + 3);
  });

  test('should show created_at timestamp for keys', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Generate a new key
    await page.click('button:has-text("Generate New Key")');
    const keyName = `Timestamp Test ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    await page.click('button:has-text("I\'ve copied the key")');
    
    // Find the key in the table
    const tableRow = page.locator(`tr:has-text("${keyName}")`);
    
    // Verify created timestamp is present and reasonable
    const createdCell = tableRow.locator('td').nth(3); // Created column
    const createdText = await createdCell.textContent();
    
    expect(createdText).toBeTruthy();
    
    // Should be today's date
    const today = new Date().toLocaleDateString();
    expect(createdText).toContain(today.split('/')[2]); // Year should match
  });

  test('should show "Never" for unused keys in last_used column', async ({ page }) => {
    await page.goto('/api-keys');
    
    // Generate a new key
    await page.click('button:has-text("Generate New Key")');
    const keyName = `Unused Key ${Date.now()}`;
    await page.fill('input#key-name', keyName);
    await page.click('button:has-text("Generate Key")');
    await page.click('button:has-text("I\'ve copied the key")');
    
    // Find the key in the table
    const tableRow = page.locator(`tr:has-text("${keyName}")`);
    
    // Verify last used shows "Never"
    const lastUsedCell = tableRow.locator('td').nth(4); // Last Used column
    await expect(lastUsedCell).toContainText('Never');
  });
});
