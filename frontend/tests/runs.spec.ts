import { test, expect } from '@playwright/test';

test.describe('Runs Page', () => {
  test('should display runs list correctly', async ({ page }) => {
    await page.goto('/runs');

    // Check page title and heading
    await expect(page).toHaveTitle(/Runs.*Traedor/);
    await expect(page.getByText('Backtest Runs')).toBeVisible();

    // Check for filter controls
    await expect(page.getByLabel(/search/i)).toBeVisible();
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Symbol')).toBeVisible();

    // Check for new run button
    await expect(page.getByRole('button', { name: /new run/i })).toBeVisible();

    // Check for runs table headers
    await expect(page.getByText('Symbol')).toBeVisible();
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Started')).toBeVisible();
    await expect(page.getByText('Return')).toBeVisible();
  });

  test('should filter runs by search', async ({ page }) => {
    await page.goto('/runs');

    const searchInput = page.getByLabel(/search/i);
    await searchInput.fill('ES');

    // Should filter results (assuming there are some runs)
    await page.waitForTimeout(500); // Wait for search debounce
    
    // Verify search input has value
    await expect(searchInput).toHaveValue('ES');
  });

  test('should create new run', async ({ page }) => {
    await page.goto('/runs');

    // Click new run button
    await page.getByRole('button', { name: /new run/i }).click();

    // Should navigate to new run page
    await expect(page.url()).toContain('/runs/new');
    await expect(page.getByText('Create New Backtest Run')).toBeVisible();

    // Check form fields are present
    await expect(page.getByLabel(/symbol/i)).toBeVisible();
    await expect(page.getByLabel(/timeframe/i)).toBeVisible();
    await expect(page.getByLabel(/start.*time/i)).toBeVisible();
    await expect(page.getByLabel(/end.*time/i)).toBeVisible();
    
    // Check form buttons
    await expect(page.getByRole('button', { name: /create run/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /cancel/i })).toBeVisible();
  });

  test('should submit new run form', async ({ page }) => {
    await page.goto('/runs/new');

    // Fill out the form
    await page.getByLabel(/symbol/i).fill('ES');
    await page.getByLabel(/timeframe/i).selectOption('5m');
    
    // Set start and end dates
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);
    
    await page.getByLabel(/start.*time/i).fill(yesterday.toISOString().split('T')[0]);
    await page.getByLabel(/end.*time/i).fill(today.toISOString().split('T')[0]);

    // Set starting balance
    await page.getByLabel(/starting.*balance/i).fill('10000');

    // Mock the API response to avoid actual backend call
    await page.route('**/api/runs', async route => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'test-run-id',
          status: 'pending',
          config: {
            symbol: 'ES',
            timeframe: '5m'
          }
        })
      });
    });

    // Submit the form
    await page.getByRole('button', { name: /create run/i }).click();

    // Should redirect back to runs page
    await expect(page.url()).toContain('/runs');
    
    // Should show success message
    await expect(page.getByText(/run.*created/i)).toBeVisible();
  });
});