const { expect } = require('@playwright/test');

const BASE = 'http://localhost:9000';

async function dismissTour(page) {
  await page.evaluate(() => {
    localStorage.setItem('zrp-tour-seen', 'true');
    document.querySelectorAll('.zt-overlay-bg, .zt-overlay, .zt-popover').forEach(el => el.remove());
    if (window._tourCleanup) window._tourCleanup();
  });
}

async function login(page, user = 'admin', pass = 'changeme') {
  for (let attempt = 0; attempt < 3; attempt++) {
    await page.goto(BASE);
    await page.evaluate(() => localStorage.setItem('zrp-tour-seen', 'true'));
    await page.waitForSelector('#login-page:not(.hidden), #app:not(.hidden)', { timeout: 15000 });
    const alreadyLoggedIn = await page.$('#app:not(.hidden)');
    if (alreadyLoggedIn) {
      await dismissTour(page);
      return;
    }
    await page.fill('#login-username', user);
    await page.fill('#login-password', pass);
    await page.click('#login-form button[type="submit"]');
    try {
      await page.waitForSelector('#app:not(.hidden)', { timeout: 10000 });
      await dismissTour(page);
      await page.waitForTimeout(1000);
      return;
    } catch (e) {
      if (attempt === 2) throw e;
      await page.waitForTimeout(1000);
    }
  }
}

async function nav(page, route) {
  await page.evaluate((r) => window.navigate(r), route);
  await page.waitForFunction(() => {
    const content = document.getElementById('content');
    return content && content.children.length > 0 && !content.textContent.includes('Loading');
  }, { timeout: 10000 });
  await dismissTour(page);
  await page.waitForTimeout(500);
}

async function getContent(page) {
  return page.textContent('#content');
}

async function fillField(page, field, value) {
  const sel = `[data-field="${field}"]`;
  const tag = await page.$eval(sel, el => el.tagName.toLowerCase());
  if (tag === 'select') {
    await page.selectOption(sel, value);
  } else {
    await page.fill(sel, value);
  }
}

async function saveModal(page) {
  await page.click('#modal-save');
  await page.waitForFunction(() => !document.querySelector('.modal-overlay'), { timeout: 10000 }).catch(() => {});
  await page.waitForTimeout(500);
}

// API helper: make authenticated fetch from within page context
async function apiFetch(page, path, options = {}) {
  return page.evaluate(async ([p, opts]) => {
    const resp = await fetch(p, opts);
    const status = resp.status;
    const contentType = resp.headers.get('content-type') || '';
    let body;
    if (contentType.includes('json')) {
      body = await resp.json();
    } else {
      body = await resp.text();
    }
    return { status, body };
  }, [path, options]);
}

// API helper: external fetch with bearer token (no page session needed)
async function apiRequest(request, path, options = {}) {
  const url = `${BASE}${path}`;
  const resp = await request.fetch(url, options);
  return resp;
}

module.exports = { BASE, login, nav, getContent, fillField, saveModal, apiFetch, apiRequest, dismissTour };
