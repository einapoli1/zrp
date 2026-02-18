// Record a UI walkthrough video of ZRP
// Usage: node tests/record-tour.js
const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

const BASE = 'http://localhost:9000';
const DEMO_DIR = path.join(__dirname, '..', 'demo');

async function dismissTour(page) {
  await page.evaluate(() => {
    localStorage.setItem('zrp-tour-seen', 'true');
    document.querySelectorAll('.zt-overlay-bg, .zt-overlay, .zt-popover').forEach(el => el.remove());
    if (window._tourCleanup) window._tourCleanup();
  });
}

async function navTo(page, route) {
  await page.evaluate((r) => { window.location.hash = '#/' + r; }, route);
  await page.waitForTimeout(500);
  await dismissTour(page);
  await page.waitForFunction(() => {
    const c = document.getElementById('content');
    return c && c.innerHTML.length > 0;
  }, { timeout: 5000 }).catch(() => {});
  await page.waitForTimeout(2500);
  await dismissTour(page);
}

(async () => {
  fs.mkdirSync(DEMO_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
    recordVideo: { dir: DEMO_DIR, size: { width: 1920, height: 1080 } },
  });
  const page = await context.newPage();

  // Login
  console.log('Logging in...');
  await page.goto(BASE);
  await page.waitForSelector('#login-page:not(.hidden)', { timeout: 15000 });
  await page.fill('#login-username', 'admin');
  await page.fill('#login-password', 'changeme');
  await page.click('#login-form button[type="submit"]');
  await page.waitForSelector('#app:not(.hidden)', { timeout: 10000 });
  await dismissTour(page);
  await page.waitForTimeout(2000);

  // Dashboard
  console.log('Dashboard...');
  await dismissTour(page);
  await page.waitForTimeout(3000);

  // Navigate through sidebar modules
  const modules = [
    'parts', 'ecos', 'inventory', 'workorders', 'ncr',
    'procurement', 'vendors', 'quotes', 'devices', 'firmware',
    'calendar', 'reports', 'audit'
  ];

  for (const mod of modules) {
    console.log(`Navigating to ${mod}...`);
    await navTo(page, mod);
  }

  // Toggle dark mode
  console.log('Toggling dark mode...');
  await dismissTour(page);
  await page.click('#dark-toggle', { force: true });
  await page.waitForTimeout(2000);

  // Navigate to ECOs and open create modal
  console.log('Opening ECO create modal...');
  await navTo(page, 'ecos');
  await dismissTour(page);
  const createBtn = await page.$('button:has-text("Create"), button:has-text("New"), button:has-text("Add")');
  if (createBtn) {
    await createBtn.click({ force: true });
    await page.waitForTimeout(2500);
    // Close modal
    const closeBtn = await page.$('#modal-close, .modal button:has-text("Cancel"), [onclick*="closeModal"]');
    if (closeBtn) {
      await closeBtn.click({ force: true });
    } else {
      await page.keyboard.press('Escape');
    }
    await page.waitForTimeout(1500);
  }

  // Toggle dark mode back
  console.log('Toggling dark mode back...');
  await dismissTour(page);
  await page.click('#dark-toggle', { force: true });
  await page.waitForTimeout(1500);

  // Finalize
  console.log('Finalizing video...');
  const video = page.video();
  await page.close();
  await context.close();

  if (video) {
    const videoPath = await video.path();
    const dest = path.join(DEMO_DIR, 'ui-walkthrough.webm');
    if (fs.existsSync(videoPath)) {
      fs.copyFileSync(videoPath, dest);
      console.log(`Video saved to: ${dest}`);
    }
  }

  await browser.close();
  console.log('Done!');
})().catch(err => {
  console.error('Recording failed:', err);
  process.exit(1);
});
