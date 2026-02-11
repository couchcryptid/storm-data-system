import { test, expect } from '../fixtures';
import {
  EXPECTED_TOTAL,
  EXPECTED_HAIL,
  EXPECTED_TORNADO,
  EXPECTED_WIND,
} from '../constants';

test.describe('Stats Cards', () => {
  test('total reports count', async ({ dashboardPage: page }) => {
    await expect(page.locator('#total')).toHaveText(String(EXPECTED_TOTAL));
  });

  test('hail count', async ({ dashboardPage: page }) => {
    await expect(page.locator('#hail-count')).toHaveText(String(EXPECTED_HAIL));
  });

  test('tornado count', async ({ dashboardPage: page }) => {
    await expect(page.locator('#tornado-count')).toHaveText(String(EXPECTED_TORNADO));
  });

  test('wind count', async ({ dashboardPage: page }) => {
    await expect(page.locator('#wind-count')).toHaveText(String(EXPECTED_WIND));
  });

  test('hail max magnitude formatted as inches', async ({ dashboardPage: page }) => {
    // Hail max should contain a double-quote (inches), e.g. 'max 4.25"'
    await expect(page.locator('#hail-max')).toContainText('max');
    await expect(page.locator('#hail-max')).toContainText('"');
  });

  test('tornado max magnitude formatted as EF scale', async ({ dashboardPage: page }) => {
    await expect(page.locator('#tornado-max')).toContainText('max EF');
  });

  test('wind max magnitude formatted as mph', async ({ dashboardPage: page }) => {
    await expect(page.locator('#wind-max')).toContainText('mph');
  });

  test('date range format and single-day constraint', async ({ dashboardPage: page }) => {
    const text = await page.locator('#date-range').textContent();
    expect(text).toBeTruthy();
    // Format: "YYYY-MM-DD — YYYY-MM-DD"
    const match = text!.match(/^(\d{4}-\d{2}-\d{2}) — (\d{4}-\d{2}-\d{2})$/);
    expect(match).not.toBeNull();
    // All mock data is processed in a single run, so both ends should be the same date.
    expect(match![1]).toBe(match![2]);
  });
});
