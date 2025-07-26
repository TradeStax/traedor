import { useRouter } from 'next/router';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { useState } from 'react';
import Layout from '@/components/Layout';
import RunProgress from '@/components/RunProgress';
import { runsApi } from '@/lib/api';
import PerformanceChart from '@/components/PerformanceChart';

export default function RunDetailPage() {
  const router = useRouter();
  const { id } = router.query;
  const queryClient = useQueryClient();
  const [isCancelling, setIsCancelling] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const tradesPerPage = 100;

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

  const { data: tradesData, error: tradesError, isLoading: tradesLoading } = useQuery({
    queryKey: ['trades', id, currentPage],
    queryFn: () => runsApi.getTrades(id as string, tradesPerPage, (currentPage - 1) * tradesPerPage),
    enabled: !!id,
    retry: 1,
  });
  
  const trades = tradesData?.trades || [];
  const tradesTotalCount = tradesData?.pagination?.total || 0;
  const totalPages = Math.ceil(tradesTotalCount / tradesPerPage);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  const handlePrevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  const { data: signals, error: signalsError } = useQuery({
    queryKey: ['signals', id],
    queryFn: () => runsApi.getSignals(id as string),
    enabled: !!id,
    retry: 1,
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

  const handleCancelRun = async () => {
    if (!id || typeof id !== 'string') return;
    
    setIsCancelling(true);
    try {
      await runsApi.cancel(id);
      // Refresh the run data immediately
      queryClient.invalidateQueries({ queryKey: ['run', id] });
      queryClient.invalidateQueries({ queryKey: ['run-refresh', id] });
    } catch (error) {
      console.error('Failed to cancel run:', error);
      // Could add toast notification here if you have one
    } finally {
      setIsCancelling(false);
    }
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
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
              Backtest Run - {run.config.symbol}
            </h1>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              Started {format(new Date(run.started_at), 'MMM d, yyyy HH:mm')}
            </p>
          </div>
          <div className="flex items-center space-x-3">
            {/* Simple status badge for header */}
            <span
              className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium text-white ${
                run.status === 'completed' ? 'bg-green-500' :
                run.status === 'running' ? 'bg-blue-500' :
                run.status === 'failed' ? 'bg-red-500' :
                run.status === 'cancelled' ? 'bg-gray-500' :
                run.status === 'queued' ? 'bg-yellow-500' :
                run.status === 'retrying' ? 'bg-orange-500' :
                'bg-gray-300'
              }`}
            >
              {run.status === 'pending' ? 'Pending' :
               run.status === 'queued' ? 'Queued' :
               run.status === 'running' ? 'Running' :
               run.status === 'completed' ? 'Completed' :
               run.status === 'failed' ? 'Failed' :
               run.status === 'cancelled' ? 'Cancelled' :
               run.status === 'retrying' ? 'Retrying' :
               run.status
              }
            </span>
            
            {/* Cancel button - only show for active runs */}
            {(run.status === 'running' || run.status === 'queued') && (
              <button
                onClick={handleCancelRun}
                disabled={isCancelling}
                className="inline-flex items-center px-3 py-2 text-sm font-medium text-red-700 bg-red-100 border border-red-300 rounded-md hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-red-900/20 dark:text-red-400 dark:border-red-700 dark:hover:bg-red-900/30"
              >
                {isCancelling ? (
                  <>
                    <svg className="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Cancelling...
                  </>
                ) : (
                  <>
                    <svg className="-ml-1 mr-2 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path>
                    </svg>
                    Cancel
                  </>
                )}
              </button>
            )}
          </div>
        </div>

        {/* Configuration Summary */}
        <div className="card">
          <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Configuration</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Symbol</dt>
              <dd className="mt-1 text-sm text-gray-900 dark:text-gray-100">{run.config.symbol}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Timeframe</dt>
              <dd className="mt-1 text-sm text-gray-900 dark:text-gray-100">{run.config.timeframe}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Starting Balance</dt>
              <dd className="mt-1 text-sm text-gray-900 dark:text-gray-100">${formatNumber(run.config.broker.starting_balance)}</dd>
            </div>
          </div>
        </div>

        {/* Run Status Details */}
        <div className="card">
          <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Run Status</h2>
          <RunProgress run={run} showDetails={true} />
        </div>

        {/* Performance Metrics */}
        {run.performance_metrics ? (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Performance Metrics</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="text-center p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                  {formatPercentage(run.performance_metrics.return_percentage)}
                </div>
                <div className="text-sm text-gray-500 dark:text-gray-400">Total Return</div>
              </div>
              <div className="text-center p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                  {run.performance_metrics.total_trades}
                </div>
                <div className="text-sm text-gray-500 dark:text-gray-400">Total Trades</div>
              </div>
              <div className="text-center p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                  {formatPercentage(run.performance_metrics.win_rate)}
                </div>
                <div className="text-sm text-gray-500 dark:text-gray-400">Win Rate</div>
              </div>
              <div className="text-center p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <div className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                  ${formatNumber(run.performance_metrics.max_drawdown)}
                </div>
                <div className="text-sm text-gray-500 dark:text-gray-400">Max Drawdown</div>
                <div className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                  {formatPercentage(run.performance_metrics.max_drawdown_percent)}
                </div>
              </div>
            </div>
            
            <div className="mt-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 text-sm">
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Final Balance</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">${formatNumber(run.performance_metrics.final_balance)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Total Profit</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">${formatNumber(run.performance_metrics.total_profit)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Profit Factor</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">{formatNumber(run.performance_metrics.profit_factor)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Average Win</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">${formatNumber(run.performance_metrics.average_win)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Average Loss</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">${formatNumber(run.performance_metrics.average_loss)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Sharpe Ratio</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">{formatNumber(run.performance_metrics.sharpe_ratio)}</dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Average MFE</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">
                  ${formatNumber(run.performance_metrics.average_mfe)} 
                  <span className="text-xs text-gray-500 dark:text-gray-400 ml-1">({formatPercentage(run.performance_metrics.average_mfe_percent)})</span>
                </dd>
              </div>
              <div>
                <dt className="font-medium text-gray-500 dark:text-gray-400">Average MAE</dt>
                <dd className="mt-1 text-gray-900 dark:text-gray-100">
                  ${formatNumber(run.performance_metrics.average_mae)}
                  <span className="text-xs text-gray-500 dark:text-gray-400 ml-1">({formatPercentage(run.performance_metrics.average_mae_percent)})</span>
                </dd>
              </div>
            </div>
          </div>
        ) : (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Performance Metrics</h2>
            <div className="text-center py-8">
              <p className="text-gray-500 dark:text-gray-400">
                {run.status === 'completed' 
                  ? 'No performance metrics available. This may indicate an issue with the run completion or data processing.'
                  : `Performance metrics will be available when the run is complete (current status: ${run.status}).`
                }
              </p>
            </div>
          </div>
        )}

        {/* Performance Chart */}
        {run.performance_metrics ? (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Equity Curve & Drawdown</h2>
            <PerformanceChart 
              trades={trades || []} 
              startingBalance={run.config.broker.starting_balance}
              balanceHistory={run.performance_metrics.balance_history}
              showDrawdown={false}
            />
          </div>
        ) : (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Equity Curve & Drawdown</h2>
            <div className="text-center py-8">
              <p className="text-gray-500 dark:text-gray-400">
                Chart will be available when performance metrics are loaded.
              </p>
            </div>
          </div>
        )}

        {/* Debug Information - Remove this in production */}
        {process.env.NODE_ENV === 'development' && (
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Debug Information</h2>
            <div className="text-sm space-y-2 font-mono">
              <div>
                <span className="font-semibold">Run Status:</span> {run.status}
              </div>
              <div>
                <span className="font-semibold">Performance Metrics Available:</span> {run.performance_metrics ? 'Yes' : 'No'}
              </div>
              <div>
                <span className="font-semibold">Trades Data:</span> {trades ? `${trades.length} trades` : tradesError ? 'Error loading' : 'Loading...'}
              </div>
              <div>
                <span className="font-semibold">Balance History:</span> {run.performance_metrics?.balance_history ? `${run.performance_metrics.balance_history.length} points` : 'None'}
              </div>
              {run.performance_metrics && (
                <div className="mt-4 p-3 bg-gray-100 dark:bg-gray-700 rounded">
                  <div className="text-xs">
                    <div>Total Trades: {run.performance_metrics.total_trades}</div>
                    <div>Total Profit: ${formatNumber(run.performance_metrics.total_profit)}</div>
                    <div>Return %: {formatPercentage(run.performance_metrics.return_percentage)}</div>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Trades Table */}
        <div className="card">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">Trades</h2>
            {tradesLoading && (
              <div className="text-sm text-gray-500 dark:text-gray-400">Loading trades...</div>
            )}
            {tradesError && (
              <div className="text-sm text-red-600 dark:text-red-400">Error loading trades</div>
            )}
            {trades && (
              <div className="flex items-center space-x-4">
                <div className="text-sm text-gray-500 dark:text-gray-400">
                  Showing {(currentPage - 1) * tradesPerPage + 1}-{Math.min(currentPage * tradesPerPage, tradesTotalCount)} of {tradesTotalCount} trades
                </div>
                {totalPages > 1 && (
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={handlePrevPage}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </button>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      Page {currentPage} of {totalPages}
                    </span>
                    <button
                      onClick={handleNextPage}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
          {tradesError ? (
            <div className="text-red-600 dark:text-red-400 text-center py-8">
              <p>Failed to load trades: {(tradesError as any)?.message || 'Unknown error'}</p>
              <p className="text-sm mt-2">Check if the API is running and accessible.</p>
            </div>
          ) : trades && trades.length === 0 ? (
            <p className="text-gray-500 dark:text-gray-400 text-center py-8">No trades found</p>
          ) : trades && trades.length > 0 ? (
            <>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-600">
                  <thead className="bg-gray-50 dark:bg-gray-700">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Operation
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Quantity
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Open Price
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Close Price
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        P&L
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        MFE
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        MAE
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Open Time
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-600">
                    {trades.map((trade, index) => (
                      <tr key={index}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                          <span className={`${trade.operation === 'Buy' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                            {trade.operation}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                          {trade.quantity}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                          ${formatNumber(trade.open_price)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                          {trade.close_price ? `$${formatNumber(trade.close_price)}` : '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          {trade.net_profit !== undefined ? (
                            <span className={trade.net_profit >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}>
                              ${formatNumber(trade.net_profit)}
                            </span>
                          ) : (
                            '-'
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {trade.mfe !== undefined ? (
                            <span>
                              ${formatNumber(trade.mfe)}
                              {trade.mfe_percent !== undefined && (
                                <span className="text-xs ml-1">({formatNumber(trade.mfe_percent)}%)</span>
                              )}
                            </span>
                          ) : '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {trade.mae !== undefined ? (
                            <span>
                              ${formatNumber(trade.mae)}
                              {trade.mae_percent !== undefined && (
                                <span className="text-xs ml-1">({formatNumber(trade.mae_percent)}%)</span>
                              )}
                            </span>
                          ) : '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {format(new Date(trade.open_time), 'MMM d, HH:mm')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              
              {/* Bottom pagination */}
              {totalPages > 1 && (
                <div className="flex justify-between items-center pt-4 border-t border-gray-200 dark:border-gray-600">
                  <div className="text-sm text-gray-500 dark:text-gray-400">
                    Showing {(currentPage - 1) * tradesPerPage + 1}-{Math.min(currentPage * tradesPerPage, tradesTotalCount)} of {tradesTotalCount} trades
                  </div>
                  <div className="flex items-center space-x-1">
                    <button
                      onClick={() => handlePageChange(1)}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      First
                    </button>
                    <button
                      onClick={handlePrevPage}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      ←
                    </button>
                    
                    {/* Page numbers */}
                    {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                      let pageNum: number;
                      if (totalPages <= 5) {
                        pageNum = i + 1;
                      } else if (currentPage <= 3) {
                        pageNum = i + 1;
                      } else if (currentPage >= totalPages - 2) {
                        pageNum = totalPages - 4 + i;
                      } else {
                        pageNum = currentPage - 2 + i;
                      }
                      
                      return (
                        <button
                          key={pageNum}
                          onClick={() => handlePageChange(pageNum)}
                          className={`px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 ${
                            currentPage === pageNum 
                              ? 'bg-blue-500 text-white border-blue-500' 
                              : 'text-gray-700 dark:text-gray-300'
                          }`}
                        >
                          {pageNum}
                        </button>
                      );
                    })}
                    
                    <button
                      onClick={handleNextPage}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      →
                    </button>
                    <button
                      onClick={() => handlePageChange(totalPages)}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Last
                    </button>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="text-gray-500 dark:text-gray-400 text-center py-8">
              Loading...
            </div>
          )}
        </div>
      </div>
    </Layout>
  );
}