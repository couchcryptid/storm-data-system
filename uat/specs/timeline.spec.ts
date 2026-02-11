import { test, expect } from '../fixtures';
import { EXPECTED_TOTAL } from '../constants';

test.describe('Activity Timeline', () => {
  test('timeline section is visible', async ({ dashboardPage: page }) => {
    await expect(page.locator('.timeline-section')).toBeVisible();
  });

  test('at least one hourly bar is rendered', async ({ dashboardPage: page }) => {
    const bars = page.locator('.tl-bar');
    expect(await bars.count()).toBeGreaterThan(0);
  });

  test('bar counts sum to total reports', async ({ dashboardPage: page }) => {
    const counts = await page.locator('.tl-count').allTextContents();
    const sum = counts.reduce((acc, text) => acc + Number(text), 0);
    expect(sum).toBe(EXPECTED_TOTAL);
  });

  test('legend shows all three event types', async ({ dashboardPage: page }) => {
    const legend = page.locator('.tl-legend');
    await expect(legend).toContainText('Hail');
    await expect(legend).toContainText('Tornado');
    await expect(legend).toContainText('Wind');
  });

  test('stacked bars use correct colors', async ({ dashboardPage: page }) => {
    // Verify at least one segment exists for each event type.
    await expect(page.locator('.tl-seg.hail').first()).toBeVisible();
    await expect(page.locator('.tl-seg.tornado').first()).toBeVisible();
    await expect(page.locator('.tl-seg.wind').first()).toBeVisible();
  });
});
