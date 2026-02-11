import { test, expect } from '../fixtures';
import { TOOLBAR_LINKS } from '../constants';

test.describe('System Toolbar', () => {
  test('displays three tool links', async ({ dashboardPage: page }) => {
    const links = page.locator('.tool-link');
    expect(await links.count()).toBe(3);
  });

  test('GraphQL Playground link', async ({ dashboardPage: page }) => {
    const link = page.locator('.tool-link', { hasText: 'GraphQL Playground' });
    await expect(link).toHaveAttribute('href', TOOLBAR_LINKS[0].href);
  });

  test('Prometheus link', async ({ dashboardPage: page }) => {
    const link = page.locator('.tool-link', { hasText: 'Prometheus' });
    await expect(link).toHaveAttribute('href', TOOLBAR_LINKS[1].href);
  });

  test('Kafka UI link', async ({ dashboardPage: page }) => {
    const link = page.locator('.tool-link', { hasText: 'Kafka UI' });
    await expect(link).toHaveAttribute('href', TOOLBAR_LINKS[2].href);
  });

  test('all links open in new tab', async ({ dashboardPage: page }) => {
    const links = page.locator('.tool-link');
    const count = await links.count();
    for (let i = 0; i < count; i++) {
      await expect(links.nth(i)).toHaveAttribute('target', '_blank');
      await expect(links.nth(i)).toHaveAttribute('rel', 'noopener noreferrer');
    }
  });
});
