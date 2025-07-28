import { test, expect } from '@playwright/test';

test.describe('Homepage', () => {
  test('should display the homepage correctly', async ({ page }) => {
    await page.goto('/');

    // Check that the title is correct
    await expect(page).toHaveTitle(/Traedor/);

    // Check for main navigation elements
    await expect(page.locator('nav')).toBeVisible();
    await expect(page.getByText('Traedor')).toBeVisible();

    // Check for main content areas
    await expect(page.getByText('Recent Runs')).toBeVisible();
    await expect(page.getByText('Quick Actions')).toBeVisible();

    // Check for new run button
    await expect(page.getByRole('button', { name: /new run/i })).toBeVisible();
  });

  test('should navigate to runs page', async ({ page }) => {
    await page.goto('/');

    // Click on runs navigation link
    await page.getByRole('link', { name: /runs/i }).click();

    // Should navigate to runs page
    await expect(page.url()).toContain('/runs');
    await expect(page.getByText('Backtest Runs')).toBeVisible();
  });

  test('should navigate to signals page', async ({ page }) => {
    await page.goto('/');

    // Click on signals navigation link  
    await page.getByRole('link', { name: /signals/i }).click();

    // Should navigate to signals page
    await expect(page.url()).toContain('/signals');
    await expect(page.getByText('Signal Definitions')).toBeVisible();
  });
});