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

  test('county dropdown is disabled until state is selected', async ({ dashboardPage: page }) => {
    await expect(page.locator('#filter-county')).toBeDisabled();

    await applyFilters(page, { state: 'NE' });
    await expect(page.locator('#filter-county')).toBeEnabled();

    await applyFilters(page, { state: '' });
    await expect(page.locator('#filter-county')).toBeDisabled();
  });

  test('county filter narrows results within a state', async ({ dashboardPage: page }) => {
    await applyFilters(page, { state: 'NE' });

    // County dropdown should have more than just the "All" option.
    const countyOptions = page.locator('#filter-county option');
    expect(await countyOptions.count()).toBeGreaterThan(1);

    // Select the first actual county (after "All").
    const firstCounty = await countyOptions.nth(1).getAttribute('value');
    await applyFilters(page, { county: firstCounty! });

    const rows = page.locator('#report-body tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
    expect(count).toBeLessThan(EXPECTED_TOP_STATES['NE']);
  });
});
