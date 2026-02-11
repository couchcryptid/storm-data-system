import { test, expect, waitForDataReady } from '../fixtures';
import { DATA_READY_TEXT } from '../constants';

test.describe('Dashboard Health', () => {
  test('page loads successfully', async ({ page }) => {
    const response = await page.goto('/');
    expect(response?.status()).toBe(200);
  });

  test('status shows expected record count', async ({ dashboardPage: page }) => {
    await expect(page.locator('#status')).toHaveText(DATA_READY_TEXT);
  });

  test('status indicator is green', async ({ dashboardPage: page }) => {
    await expect(page.locator('#status')).toHaveClass(/\bok\b/);
  });

  test('freshness badge is visible', async ({ dashboardPage: page }) => {
    const freshness = page.locator('#freshness');
    await expect(freshness).toBeVisible();
  });

  test('freshness badge shows a lag value', async ({ dashboardPage: page }) => {
    const value = page.locator('#freshness-value');
    const text = await value.textContent();
    expect(text).toBeTruthy();
    // Lag is displayed as either "Xm" or "Xh Ym".
    expect(text).toMatch(/^\d+m$|^\d+h \d+m$/);
  });

  test('freshness badge has a severity class', async ({ dashboardPage: page }) => {
    const freshness = page.locator('#freshness');
    const cls = await freshness.getAttribute('class');
    expect(cls).toMatch(/\bfreshness-(ok|warn|stale|err)\b/);
  });

  test('no console errors during load', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') errors.push(msg.text());
    });

    await page.goto('/');
    await waitForDataReady(page);

    expect(errors).toEqual([]);
  });
});
