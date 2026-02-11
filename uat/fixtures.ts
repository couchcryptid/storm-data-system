import { test as base, expect as baseExpect, Page } from '@playwright/test';
import { DATA_READY_TEXT, DATA_READY_TIMEOUT } from './constants';

/** Wait for the dashboard to finish loading data and display the expected record count. */
export async function waitForDataReady(page: Page): Promise<void> {
  await baseExpect(page.locator('#status')).toHaveText(DATA_READY_TEXT, {
    timeout: DATA_READY_TIMEOUT,
  });
}

export interface FilterOptions {
  type?: string;
  state?: string;
  severity?: string;
  county?: string;
}

/** Set one or more filter dropdowns and wait briefly for client-side re-render. */
export async function applyFilters(page: Page, filters: FilterOptions): Promise<void> {
  if (filters.type !== undefined) {
    await page.locator('#filter-type').selectOption(filters.type);
  }
  if (filters.state !== undefined) {
    await page.locator('#filter-state').selectOption(filters.state);
  }
  if (filters.severity !== undefined) {
    await page.locator('#filter-severity').selectOption(filters.severity);
  }
  if (filters.county !== undefined) {
    await page.locator('#filter-county').selectOption(filters.county);
  }
  // Filters are applied synchronously via DOM manipulation,
  // but give the browser a rendering tick.
  await page.waitForTimeout(200);
}

/**
 * Custom test fixture that navigates to the dashboard and waits for data
 * before handing the page to the test body.
 */
export const test = base.extend<{ dashboardPage: Page }>({
  dashboardPage: async ({ page }, use) => {
    await page.goto('/');
    await waitForDataReady(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';
