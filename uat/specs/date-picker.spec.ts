import { test, expect } from '../fixtures';
import { EXPECTED_TOTAL, DATA_READY_TIMEOUT } from '../constants';

test.describe('Date Picker', () => {
  test('date input is visible and has a value', async ({ dashboardPage: page }) => {
    const dateFrom = page.locator('#date-from');
    await expect(dateFrom).toBeVisible();
    const value = await dateFrom.inputValue();
    // Should be a valid YYYY-MM-DD date string.
    expect(value).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  test('date is auto-detected from data', async ({ dashboardPage: page }) => {
    // The date picker should be set to the date of the loaded reports.
    const dateValue = await page.locator('#date-from').inputValue();
    // Verify the date-range sub text reflects this same date.
    const dateRange = await page.locator('#date-range').textContent();
    expect(dateRange).toContain(dateValue);
  });

  test('range toggle shows second date input', async ({ dashboardPage: page }) => {
    const dateTo = page.locator('#date-to');
    // Before toggle, end date input should be hidden.
    await expect(dateTo).not.toBeVisible();

    await page.locator('#date-range-toggle').check();
    await expect(dateTo).toBeVisible();

    // Both dates should start with the same value.
    const from = await page.locator('#date-from').inputValue();
    const to = await dateTo.inputValue();
    expect(to).toBe(from);
  });

  test('unchecking range toggle hides end date and reloads', async ({ dashboardPage: page }) => {
    await page.locator('#date-range-toggle').check();
    await expect(page.locator('#date-to')).toBeVisible();

    await page.locator('#date-range-toggle').uncheck();
    await expect(page.locator('#date-to')).not.toBeVisible();
    // Should still show loaded reports after collapsing range.
    await expect(page.locator('#status')).toHaveText(`${EXPECTED_TOTAL} reports loaded`, {
      timeout: DATA_READY_TIMEOUT,
    });
  });

  test('selecting a date with no data shows empty state', async ({ dashboardPage: page }) => {
    // Pick a date far in the past where no data exists.
    await page.locator('#date-from').fill('2000-01-01');
    await page.locator('#date-from').dispatchEvent('change');
    await expect(page.locator('#status')).toHaveText('No reports for selected date', {
      timeout: DATA_READY_TIMEOUT,
    });
    await expect(page.locator('#total')).toHaveText('0');
    await expect(page.locator('#date-range')).toHaveText('No data');
  });

  test('date label is visible', async ({ dashboardPage: page }) => {
    const label = page.locator('.date-label');
    await expect(label).toBeVisible();
    await expect(label).toHaveText('Date');
  });
});
