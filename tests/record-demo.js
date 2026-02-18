const { chromium } = require('@playwright/test');
const path = require('path');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    recordVideo: { dir: path.join(__dirname, '..', 'demo-video'), size: { width: 1440, height: 900 } }
  });
  const page = await context.newPage();
  const BASE = 'http://localhost:9000';

  // Dashboard
  await page.goto(BASE);
  await page.waitForTimeout(2000);

  // Parts
  await page.click('[data-route="parts"]');
  await page.waitForTimeout(2000);

  // ECOs
  await page.click('[data-route="ecos"]');
  await page.waitForTimeout(2000);

  // Inventory
  await page.click('[data-route="inventory"]');
  await page.waitForTimeout(2000);

  // Work Orders
  await page.click('[data-route="workorders"]');
  await page.waitForTimeout(2000);

  // Devices
  await page.click('[data-route="devices"]');
  await page.waitForTimeout(2000);

  // NCRs
  await page.click('[data-route="ncr"]');
  await page.waitForTimeout(2000);

  // Vendors
  await page.click('[data-route="vendors"]');
  await page.waitForTimeout(2000);

  // Quotes
  await page.click('[data-route="quotes"]');
  await page.waitForTimeout(2000);

  // RMAs
  await page.click('[data-route="rma"]');
  await page.waitForTimeout(2000);

  // Firmware
  await page.click('[data-route="firmware"]');
  await page.waitForTimeout(2000);

  // Back to dashboard
  await page.click('[data-route="dashboard"]');
  await page.waitForTimeout(2000);

  await context.close();
  await browser.close();

  // Find the video file
  const fs = require('fs');
  const dir = path.join(__dirname, '..', 'demo-video');
  const files = fs.readdirSync(dir);
  const videoFile = files.find(f => f.endsWith('.webm'));
  if (videoFile) {
    console.log('Video: ' + path.join(dir, videoFile));
  }
})();
