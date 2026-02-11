import { test, expect, applyFilters } from '../fixtures';
import {
  EXPECTED_TOTAL,
  EXPECTED_HAIL,
  EXPECTED_TORNADO,
  EXPECTED_STATE_COUNT,
  EXPECTED_TOP_STATES,
} from '../constants';

test.describe('Reports Table', () => {
  test('table shows all reports initially', async ({ dashboardPage: page }) => {
    const rows = page.locator('#report-body tr');
    expect(await rows.count()).toBe(EXPECTED_TOTAL);
  });

  test('type filter narrows to hail reports', async ({ dashboardPage: page }) => {
    await applyFilters(page, { type: 'hail' });
    const rows = page.locator('#report-body tr');
    expect(await rows.count()).toBe(EXPECTED_HAIL);
  });

  test('state filter narrows to NE reports', async ({ dashboardPage: page }) => {
    await applyFilters(page, { state: 'NE' });
    const rows = page.locator('#report-body tr');
    expect(await rows.count()).toBe(EXPECTED_TOP_STATES['NE']);
  });

  test('severity filter shows only matching rows', async ({ dashboardPage: page }) => {
    await applyFilters(page, { severity: 'severe' });
    const rows = page.locator('#report-body tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
    expect(count).toBeLessThan(EXPECTED_TOTAL);
  });

  test('combined filters work', async ({ dashboardPage: page }) => {
    await applyFilters(page, { type: 'tornado', state: 'IA' });
    const rows = page.locator('#report-body tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
    expect(count).toBeLessThan(EXPECTED_TORNADO);
  });

  test('state dropdown has expected number of states', async ({ dashboardPage: page }) => {
    const options = page.locator('#filter-state option');
    // +1 for the "All" option.
    expect(await options.count()).toBe(EXPECTED_STATE_COUNT + 1);
  });

  test('resetting filters shows all reports', async ({ dashboardPage: page }) => {
    await applyFilters(page, { type: 'hail' });
    expect(await page.locator('#report-body tr').count()).toBe(EXPECTED_HAIL);

    await applyFilters(page, { type: '' });
    expect(await page.locator('#report-body tr').count()).toBe(EXPECTED_TOTAL);
  });
});
