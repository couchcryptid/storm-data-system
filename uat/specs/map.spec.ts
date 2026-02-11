import { test, expect } from '../fixtures';
import { EXPECTED_TOTAL } from '../constants';

test.describe('Event Map', () => {
  test('leaflet map container is rendered', async ({ dashboardPage: page }) => {
    await expect(page.locator('#map')).toBeVisible();
    // Leaflet adds its own container class.
    await expect(page.locator('.leaflet-container')).toBeVisible();
  });

  test('map tiles are loaded', async ({ dashboardPage: page }) => {
    const tiles = page.locator('.leaflet-tile-loaded');
    expect(await tiles.count()).toBeGreaterThan(0);
  });

  test('expected number of markers on map', async ({ dashboardPage: page }) => {
    // L.circleMarker renders as SVG <path> elements with the .leaflet-interactive class.
    const markers = page.locator('.leaflet-interactive');
    const count = await markers.count();
    expect(count).toBe(EXPECTED_TOTAL);
  });

  test('clicking a marker opens a popup', async ({ dashboardPage: page }) => {
    // Wait for fitBounds animation to settle before clicking.
    await page.waitForTimeout(500);
    const marker = page.locator('.leaflet-interactive').first();
    await marker.dispatchEvent('click');
    await expect(page.locator('.leaflet-popup-content')).toBeVisible();
  });

  test('color mode dropdown defaults to "By Type"', async ({ dashboardPage: page }) => {
    const sel = page.locator('#map-color-mode');
    await expect(sel).toBeVisible();
    await expect(sel).toHaveValue('type');
  });

  test('switching color mode to severity re-renders markers', async ({ dashboardPage: page }) => {
    const markersBefore = await page.locator('.leaflet-interactive').count();
    await page.locator('#map-color-mode').selectOption('severity');
    // Markers are re-rendered so count should remain the same.
    const markersAfter = await page.locator('.leaflet-interactive').count();
    expect(markersAfter).toBe(markersBefore);
  });

  test('popup contains event details', async ({ dashboardPage: page }) => {
    await page.waitForTimeout(500);
    const marker = page.locator('.leaflet-interactive').first();
    await marker.dispatchEvent('click');

    const popup = page.locator('.leaflet-popup-content');
    await expect(popup).toBeVisible();
    const text = await popup.textContent();
    expect(text).toBeTruthy();
    // Popups show event type (uppercase), location, and time.
    expect(text!.toLowerCase()).toMatch(/hail|tornado|wind/);
  });
});
