import { test, expect } from '@playwright/test';

test.describe('Signals Page', () => {
  const mockSignals = [
    {
      id: 'sma-crossover-1',
      name: 'SMA Crossover',
      description: 'Simple Moving Average crossover signal',
      type: 'technical',
      parameters: {
        short_period: 5,
        long_period: 20
      },
      active: true,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    },
    {
      id: 'rsi-overbought-1',
      name: 'RSI Overbought/Oversold',
      description: 'RSI based overbought and oversold signals',
      type: 'technical',
      parameters: {
        period: 14,
        overbought: 70,
        oversold: 30
      },
      active: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    }
  ];

  test.beforeEach(async ({ page }) => {
    // Mock API responses
    await page.route('**/api/signals', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockSignals)
      });
    });
  });

  test('should display signals list correctly', async ({ page }) => {
    await page.goto('/signals');

    // Check page title and heading
    await expect(page).toHaveTitle(/Signals.*Traedor/);
    await expect(page.getByText('Signal Definitions')).toBeVisible();

    // Check for create new signal button
    await expect(page.getByRole('button', { name: /new signal/i })).toBeVisible();

    // Check table headers
    await expect(page.getByText('Name')).toBeVisible();
    await expect(page.getByText('Type')).toBeVisible();
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Actions')).toBeVisible();

    // Check signal data
    await expect(page.getByText('SMA Crossover')).toBeVisible();
    await expect(page.getByText('Simple Moving Average crossover signal')).toBeVisible();
    await expect(page.getByText('technical')).toBeVisible();

    // Check active/inactive status
    await expect(page.getByText('Active').first()).toBeVisible();
    await expect(page.getByText('Inactive')).toBeVisible();
  });

  test('should filter signals by type', async ({ page }) => {
    await page.goto('/signals');

    // Click on type filter
    const typeFilter = page.getByLabel(/type/i);
    await typeFilter.selectOption('technical');

    // Should show filtered results
    await expect(page.getByText('SMA Crossover')).toBeVisible();
    await expect(page.getByText('RSI Overbought/Oversold')).toBeVisible();
  });

  test('should filter signals by status', async ({ page }) => {
    await page.goto('/signals');

    // Click on status filter
    const statusFilter = page.getByLabel(/status/i);
    await statusFilter.selectOption('active');

    // Should show only active signals
    await expect(page.getByText('SMA Crossover')).toBeVisible();
  });

  test('should create new signal', async ({ page }) => {
    await page.goto('/signals');

    // Mock create signal API
    await page.route('**/api/signals', async route => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'new-signal-id',
            name: 'Test Signal',
            type: 'custom'
          })
        });
      }
    });

    // Click new signal button
    await page.getByRole('button', { name: /new signal/i }).click();

    // Should show modal or navigate to form
    await expect(page.getByText('Create New Signal')).toBeVisible();

    // Fill out form
    await page.getByLabel(/name/i).fill('Test Signal');
    await page.getByLabel(/description/i).fill('A test signal for validation');
    await page.getByLabel(/type/i).selectOption('custom');

    // Submit form
    await page.getByRole('button', { name: /create/i }).click();

    // Should show success message
    await expect(page.getByText(/signal.*created/i)).toBeVisible();
  });

  test('should edit signal', async ({ page }) => {
    await page.goto('/signals');

    // Mock update signal API
    await page.route('**/api/signals/sma-crossover-1', async route => {
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...mockSignals[0],
            name: 'Updated SMA Crossover'
          })
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockSignals[0])
        });
      }
    });

    // Click edit button for first signal
    await page.getByRole('button', { name: /edit/i }).first().click();

    // Should show edit form
    await expect(page.getByText('Edit Signal')).toBeVisible();

    // Update name
    const nameInput = page.getByLabel(/name/i);
    await nameInput.clear();
    await nameInput.fill('Updated SMA Crossover');

    // Submit form
    await page.getByRole('button', { name: /save/i }).click();

    // Should show success message
    await expect(page.getByText(/signal.*updated/i)).toBeVisible();
  });

  test('should toggle signal active status', async ({ page }) => {
    await page.goto('/signals');

    // Mock update signal API
    await page.route('**/api/signals/sma-crossover-1', async route => {
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...mockSignals[0],
            active: false
          })
        });
      }
    });

    // Click toggle button for first signal
    await page.getByRole('button', { name: /toggle/i }).first().click();

    // Should update status
    await expect(page.getByText('Inactive').first()).toBeVisible();
  });

  test('should delete signal', async ({ page }) => {
    await page.goto('/signals');

    // Mock delete signal API
    await page.route('**/api/signals/sma-crossover-1', async route => {
      if (route.request().method() === 'DELETE') {
        await route.fulfill({
          status: 204
        });
      }
    });

    // Mock confirmation dialog
    page.on('dialog', dialog => dialog.accept());

    // Click delete button for first signal
    await page.getByRole('button', { name: /delete/i }).first().click();

    // Should show confirmation dialog and delete
    await expect(page.getByText(/signal.*deleted/i)).toBeVisible();
  });

  test('should display signal parameters', async ({ page }) => {
    await page.goto('/signals');

    // Click on a signal to view details
    await page.getByText('SMA Crossover').click();

    // Should show signal details
    await expect(page.getByText('Signal Details')).toBeVisible();
    await expect(page.getByText('Parameters')).toBeVisible();
    
    // Check parameters are displayed
    await expect(page.getByText('short_period: 5')).toBeVisible();
    await expect(page.getByText('long_period: 20')).toBeVisible();
  });

  test('should validate signal form', async ({ page }) => {
    await page.goto('/signals');

    // Click new signal button
    await page.getByRole('button', { name: /new signal/i }).click();

    // Try to submit empty form
    await page.getByRole('button', { name: /create/i }).click();

    // Should show validation errors
    await expect(page.getByText(/name.*required/i)).toBeVisible();
    await expect(page.getByText(/type.*required/i)).toBeVisible();
  });
});