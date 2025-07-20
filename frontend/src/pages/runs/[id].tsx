import { useRouter } from 'next/router';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import Layout from '@/components/Layout';
import RunProgress from '@/components/RunProgress';
import { runsApi } from '@/lib/api';
import PerformanceChart from '@/components/PerformanceChart';

export default function RunDetailPage() {
  const router = useRouter();
  const { id } = router.query;

  const { data: run, isLoading, error } = useQuery({
    queryKey: ['run', id],
    queryFn: () => runsApi.get(id as string),
    enabled: !!id,
  });
  
  // Auto-refresh for active runs
  useQuery({
    queryKey: ['run-refresh', id],
    queryFn: () => runsApi.get(id as string),
    enabled: !!id && (run?.status === 'running' || run?.status === 'queued'),
    refetchInterval: 2000,
  });

  const { data: trades } = useQuery({
    queryKey: ['trades', id],
    queryFn: () => runsApi.getTrades(id as string),
    enabled: !!id,
  });

  const { data: signals } = useQuery({
    queryKey: ['signals', id],
    queryFn: () => runsApi.getSignals(id as string),
    enabled: !!id,
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'text-green-600 bg-green-100';
      case 'running':
        return 'text-blue-600 bg-blue-100';
      case 'failed':
        return 'text-red-600 bg-red-100';
      default:
        return 'text-gray-600 bg-gray-100';
    }
  };

  const formatNumber = (num?: number) => {
    if (num === undefined || num === null) return '-';
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(num);
  };

  const formatPercentage = (num?: number) => {
    if (num === undefined || num === null) return '-';
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
      style: 'percent',
    }).format(num / 100);
  };

  if (isLoading) {
    return (
      <Layout>
        <div className="text-center py-12">
          <div className="inline-flex items-center">
            <svg className="animate-spin h-5 w-5 mr-3 text-primary-600" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            Loading run details...
          </div>
        </div>
      </Layout>
    );
  }

  if (error || !run) {
    return (
      <Layout>
        <div className="text-red-600 text-center py-12">
          Error loading run details. Please try again.
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">
              Backtest Run - {run.config.symbol}
            </h1>
            <p className="mt-1 text-sm text-gray-600">
              Started {format(new Date(run.started_at), 'MMM d, yyyy HH:mm')}
            </p>
          </div>
          <RunProgress run={run} showDetails={true} />
        </div>

        {/* Configuration Summary */}
        <div className="card">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Configuration</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <dt className="text-sm font-medium text-gray-500">Symbol</dt>
              <dd className="mt-1 text-sm text-gray-900">{run.config.symbol}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Timeframe</dt>
              <dd className="mt-1 text-sm text-gray-900">{run.config.timeframe}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Starting Balance</dt>
              <dd className="mt-1 text-sm text-gray-900">${formatNumber(run.config.broker.starting_balance)}</dd>
            </div>
          </div>
        </div>

        {/* Performance Metrics */}
        {run.performance_metrics && (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Performance Metrics</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900">
                  {formatPercentage(run.performance_metrics.return_percentage)}
                </div>
                <div className="text-sm text-gray-500">Total Return</div>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900">
                  {run.performance_metrics.total_trades}
                </div>
                <div className="text-sm text-gray-500">Total Trades</div>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900">
                  {formatPercentage(run.performance_metrics.win_rate)}
                </div>
                <div className="text-sm text-gray-500">Win Rate</div>
              </div>
              <div className="text-center p-4 bg-gray-50 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900">
                  ${formatNumber(run.performance_metrics.max_drawdown)}
                </div>
                <div className="text-sm text-gray-500">Max Drawdown</div>
                <div className="text-xs text-gray-400 mt-1">
                  {formatPercentage(run.performance_metrics.max_drawdown_percent)}
                </div>
              </div>
            </div>
            
            <div className="mt-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 text-sm">
              <div>
                <dt className="font-medium text-gray-500">Final Balance</dt>
                <dd className="mt-1 text-gray-900">${formatNumber(run.performance_metrics.final_balance)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Total Profit</dt>
                <dd className="mt-1 text-gray-900">${formatNumber(run.performance_metrics.total_profit)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Profit Factor</dt>
                <dd className="mt-1 text-gray-900">{formatNumber(run.performance_metrics.profit_factor)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Average Win</dt>
                <dd className="mt-1 text-gray-900">${formatNumber(run.performance_metrics.average_win)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Average Loss</dt>
                <dd className="mt-1 text-gray-900">${formatNumber(run.performance_metrics.average_loss)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Sharpe Ratio</dt>
                <dd className="mt-1 text-gray-900">{formatNumber(run.performance_metrics.sharpe_ratio)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Average MFE</dt>
                <dd className="mt-1 text-gray-900">
                  ${formatNumber(run.performance_metrics.average_mfe)} 
                  <span className="text-xs text-gray-500 ml-1">({formatPercentage(run.performance_metrics.average_mfe_percent)})</span>
                </dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500">Average MAE</dt>
                <dd className="mt-1 text-gray-900">
                  ${formatNumber(run.performance_metrics.average_mae)}
                  <span className="text-xs text-gray-500 ml-1">({formatPercentage(run.performance_metrics.average_mae_percent)})</span>
                </dd>
              </div>
            </div>
          </div>
        )}

        {/* Performance Chart */}
        {run.performance_metrics && (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Equity Curve & Drawdown</h2>
            <PerformanceChart 
              trades={trades || []} 
              startingBalance={run.config.broker.starting_balance}
              balanceHistory={run.performance_metrics.balance_history}
              showDrawdown={true}
            />
          </div>
        )}

        {/* Trades Table */}
        {trades && (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Trades ({trades.length})</h2>
            {trades.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No trades found</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Operation
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Quantity
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Open Price
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Close Price
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        P&L
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        MFE
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        MAE
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Open Time
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {trades.map((trade, index) => (
                      <tr key={index}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                          <span className={`${trade.operation === 'Buy' ? 'text-green-600' : 'text-red-600'}`}>
                            {trade.operation}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {trade.quantity}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          ${formatNumber(trade.open_price)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {trade.close_price ? `$${formatNumber(trade.close_price)}` : '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          {trade.net_profit !== undefined ? (
                            <span className={trade.net_profit >= 0 ? 'text-green-600' : 'text-red-600'}>
                              ${formatNumber(trade.net_profit)}
                            </span>
                          ) : (
                            '-'
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {trade.mfe !== undefined ? (
                            <span>
                              ${formatNumber(trade.mfe)}
                              {trade.mfe_percent !== undefined && (
                                <span className="text-xs ml-1">({formatNumber(trade.mfe_percent)}%)</span>
                              )}
                            </span>
                          ) : '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {trade.mae !== undefined ? (
                            <span>
                              ${formatNumber(trade.mae)}
                              {trade.mae_percent !== undefined && (
                                <span className="text-xs ml-1">({formatNumber(trade.mae_percent)}%)</span>
                              )}
                            </span>
                          ) : '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {format(new Date(trade.open_time), 'MMM d, HH:mm')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </Layout>
  );
}