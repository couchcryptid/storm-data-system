import { test, expect } from '../fixtures';

test.describe('GraphQL Query Panel', () => {
  test('panel is collapsed by default', async ({ dashboardPage: page }) => {
    const panel = page.locator('#query-panel');
    await expect(panel).not.toHaveClass(/expanded/);
  });

  test('clicking query bar expands panel', async ({ dashboardPage: page }) => {
    await page.locator('#query-bar').click();
    await expect(page.locator('#query-panel')).toHaveClass(/expanded/);
  });

  test('clicking query bar again collapses panel', async ({ dashboardPage: page }) => {
    await page.locator('#query-bar').click();
    await expect(page.locator('#query-panel')).toHaveClass(/expanded/);

    await page.locator('#query-bar').click();
    await expect(page.locator('#query-panel')).not.toHaveClass(/expanded/);
  });

  test('pressing Enter on query bar toggles panel', async ({ dashboardPage: page }) => {
    const bar = page.locator('#query-bar');
    await bar.focus();
    await bar.press('Enter');
    await expect(page.locator('#query-panel')).toHaveClass(/expanded/);
    await expect(bar).toHaveAttribute('aria-expanded', 'true');

    await bar.press('Enter');
    await expect(page.locator('#query-panel')).not.toHaveClass(/expanded/);
    await expect(bar).toHaveAttribute('aria-expanded', 'false');
  });

  test('pressing Space on query bar toggles panel', async ({ dashboardPage: page }) => {
    const bar = page.locator('#query-bar');
    await bar.focus();
    await bar.press('Space');
    await expect(page.locator('#query-panel')).toHaveClass(/expanded/);

    await bar.press('Space');
    await expect(page.locator('#query-panel')).not.toHaveClass(/expanded/);
  });

  test('query text is populated with initial query', async ({ dashboardPage: page }) => {
    const value = await page.locator('#query-text').inputValue();
    expect(value).toContain('stormReports');
  });

  test('edit checkbox enables textarea and run button', async ({ dashboardPage: page }) => {
    // Before checking the box, textarea is readonly and run is disabled.
    await expect(page.locator('#query-text')).toHaveAttribute('readonly', '');
    await expect(page.locator('#query-run')).toBeDisabled();

    await page.locator('#query-editable').check();

    await expect(page.locator('#query-text')).not.toHaveAttribute('readonly', '');
    await expect(page.locator('#query-run')).toBeEnabled();
  });

  test('running a query shows results and timing', async ({ dashboardPage: page }) => {
    // Expand panel, enable editing, run the pre-filled query.
    await page.locator('#query-bar').click();
    await page.locator('#query-editable').check();
    await page.locator('#query-run').click();

    // Wait for result to appear (not "Running...").
    await page.waitForFunction(
      () => {
        const el = document.querySelector('#query-result');
        return el?.textContent && !el.textContent.includes('Running...');
      },
      { timeout: 15_000 },
    );

    const result = await page.locator('#query-result').textContent();
    expect(result).toContain('stormReports');

    const timing = await page.locator('#query-time').textContent();
    expect(timing).toMatch(/\d+ms/);
  });

  test('running a custom query returns data', async ({ dashboardPage: page }) => {
    await page.locator('#query-bar').click();
    await page.locator('#query-editable').check();

    const customQuery = `{
      stormReports(filter: {
        timeRange: { from: "2020-01-01T00:00:00Z", to: "2030-01-01T00:00:00Z" }
        eventTypes: [HAIL]
        limit: 1
      }) { totalCount }
    }`;
    await page.locator('#query-text').fill(customQuery);
    await page.locator('#query-run').click();

    await page.waitForFunction(
      () => {
        const el = document.querySelector('#query-result');
        return el?.textContent && !el.textContent.includes('Running...');
      },
      { timeout: 15_000 },
    );

    const result = await page.locator('#query-result').textContent();
    expect(result).toContain('totalCount');
  });
});
