import { test, expect } from '@playwright/test';

test.describe('Run Detail Page', () => {
  const mockRun = {
    id: 'test-run-123',
    config: {
      symbol: 'ES',
      timeframe: '5m',
      broker: {
        starting_balance: 10000
      }
    },
    status: 'completed',
    started_at: new Date().toISOString(),
    performance_metrics: {
      total_trades: 25,
      winning_trades: 15,
      losing_trades: 10,
      return_percentage: 12.5,
      win_rate: 60.0,
      max_drawdown: 500.0,
      max_drawdown_percent: 5.0,
      final_balance: 11250.0,
      total_profit: 1250.0,
      profit_factor: 2.1,
      average_win: 150.0,
      average_loss: -75.0,
      sharpe_ratio: 1.8,
      average_mfe: 200.0,
      average_mfe_percent: 2.0,
      average_mae: 100.0,
      average_mae_percent: 1.0,
      balance_history: [
        { time: new Date().toISOString(), balance: 10000 },
        { time: new Date().toISOString(), balance: 10500 },
        { time: new Date().toISOString(), balance: 11250 }
      ]
    }
  };

  const mockTrades = [
    {
      id: '1',
      symbol: 'ES',
      operation: 'Buy',
      quantity: 1,
      open_price: 4500.0,
      close_price: 4510.0,
      open_time: new Date().toISOString(),
      close_time: new Date().toISOString(),
      net_profit: 495.0,
      mfe: 250.0,
      mfe_percent: 1.1,
      mae: 125.0,
      mae_percent: 0.6
    },
    {
      id: '2',
      symbol: 'ES',
      operation: 'Buy',
      quantity: 1,
      open_price: 4520.0,
      close_price: 4515.0,
      open_time: new Date().toISOString(),
      close_time: new Date().toISOString(),
      net_profit: -255.0,
      mfe: 150.0,
      mfe_percent: 0.7,
      mae: 300.0,
      mae_percent: 1.3
    }
  ];

  test.beforeEach(async ({ page }) => {
    // Mock API responses
    await page.route('**/api/runs/test-run-123', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockRun)
      });
    });

    await page.route('**/api/runs/test-run-123/trades', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockTrades)
      });
    });

    await page.route('**/api/runs/test-run-123/signals', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([])
      });
    });
  });

  test('should display run details correctly', async ({ page }) => {
    await page.goto('/runs/test-run-123');

    // Check page title
    await expect(page.getByText('Backtest Run - ES')).toBeVisible();

    // Check status badge
    await expect(page.getByText('completed')).toBeVisible();

    // Check configuration section
    await expect(page.getByText('Configuration')).toBeVisible();
    await expect(page.getByText('ES')).toBeVisible();
    await expect(page.getByText('5m')).toBeVisible();
    await expect(page.getByText('$10,000.00')).toBeVisible();
  });

  test('should display performance metrics correctly', async ({ page }) => {
    await page.goto('/runs/test-run-123');

    // Check performance metrics section
    await expect(page.getByText('Performance Metrics')).toBeVisible();

    // Check key metrics
    await expect(page.getByText('12.50%')).toBeVisible(); // Total Return
    await expect(page.getByText('25')).toBeVisible(); // Total Trades
    await expect(page.getByText('60.00%')).toBeVisible(); // Win Rate
    await expect(page.getByText('$500.00')).toBeVisible(); // Max Drawdown

    // Check detailed metrics
    await expect(page.getByText('Final Balance')).toBeVisible();
    await expect(page.getByText('$11,250.00')).toBeVisible();
    
    await expect(page.getByText('Total Profit')).toBeVisible();
    await expect(page.getByText('$1,250.00')).toBeVisible();

    await expect(page.getByText('Profit Factor')).toBeVisible();
    await expect(page.getByText('2.10')).toBeVisible();

    // Check MFE/MAE metrics
    await expect(page.getByText('Average MFE')).toBeVisible();
    await expect(page.getByText('$200.00')).toBeVisible();
    await expect(page.getByText('(2.00%)')).toBeVisible();

    await expect(page.getByText('Average MAE')).toBeVisible();
    await expect(page.getByText('$100.00')).toBeVisible();
    await expect(page.getByText('(1.00%)')).toBeVisible();
  });

  test('should display performance chart', async ({ page }) => {
    await page.goto('/runs/test-run-123');

    // Check chart section
    await expect(page.getByText('Equity Curve & Drawdown')).toBeVisible();

    // Check chart container exists
    await expect(page.locator('.w-full > div').first()).toBeVisible();

    // Check chart legend
    await expect(page.getByText('Account Balance')).toBeVisible();
    await expect(page.getByText('Drawdown')).toBeVisible();
  });

  test('should display trades table correctly', async ({ page }) => {
    await page.goto('/runs/test-run-123');

    // Check trades section
    await expect(page.getByText('Trades (2)')).toBeVisible();

    // Check table headers
    await expect(page.getByText('Operation')).toBeVisible();
    await expect(page.getByText('Quantity')).toBeVisible();
    await expect(page.getByText('Open Price')).toBeVisible();
    await expect(page.getByText('Close Price')).toBeVisible();
    await expect(page.getByText('P&L')).toBeVisible();
    await expect(page.getByText('MFE')).toBeVisible();
    await expect(page.getByText('MAE')).toBeVisible();

    // Check trade data
    await expect(page.getByText('Buy').first()).toBeVisible();
    await expect(page.getByText('$4,500.00')).toBeVisible();
    await expect(page.getByText('$4,510.00')).toBeVisible();
    await expect(page.getByText('$495.00')).toBeVisible();
    await expect(page.getByText('$250.00')).toBeVisible();
    await expect(page.getByText('$125.00')).toBeVisible();

    // Check profit/loss colors
    const profitCell = page.locator('text=$495.00').locator('xpath=..');
    await expect(profitCell).toHaveClass(/text-green-600/);

    const lossCell = page.locator('text=-$255.00').locator('xpath=..');
    await expect(lossCell).toHaveClass(/text-red-600/);
  });

  test('should handle loading state', async ({ page }) => {
    // Delay the API response to test loading state
    await page.route('**/api/runs/test-run-123', async route => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockRun)
      });
    });

    await page.goto('/runs/test-run-123');

    // Should show loading state
    await expect(page.getByText('Loading run details...')).toBeVisible();
    
    // Eventually should show content
    await expect(page.getByText('Backtest Run - ES')).toBeVisible();
  });

  test('should handle error state', async ({ page }) => {
    // Mock error response
    await page.route('**/api/runs/test-run-123', async route => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Run not found' })
      });
    });

    await page.goto('/runs/test-run-123');

    // Should show error message
    await expect(page.getByText('Error loading run details')).toBeVisible();
  });
});