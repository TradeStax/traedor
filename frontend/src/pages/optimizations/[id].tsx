import { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import Layout from '@/components/Layout';
import { optimizationApi } from '@/lib/api';
import { Optimization, OptimizationRunResult, OptimizationStatus } from '@/types';

export default function OptimizationDetailPage() {
  const router = useRouter();
  const { id } = router.query;
  const [selectedTab, setSelectedTab] = useState<'overview' | 'results' | 'parameters'>('overview');
  const [sortField, setSortField] = useState<keyof OptimizationRunResult>('optimization_score');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');

  const { data: optimization, isLoading: isLoadingOptimization, refetch: refetchOptimization } = useQuery({
    queryKey: ['optimization', id],
    queryFn: () => optimizationApi.get(id as string),
    enabled: !!id,
    refetchInterval: (data) => {
      // Refetch every 5 seconds if optimization is running or queued
      const opt = data as unknown as Optimization;
      if (opt?.status === 'running' || opt?.status === 'queued') {
        return 5000;
      }
      return false;
    },
  });

  const { data: results, isLoading: isLoadingResults, refetch: refetchResults } = useQuery({
    queryKey: ['optimizationResults', id],
    queryFn: () => optimizationApi.getResults(id as string),
    enabled: !!id && optimization?.status !== 'pending' && optimization?.status !== 'queued',
    refetchInterval: () => {
      // Refetch every 10 seconds if optimization is running
      if (optimization?.status === 'running') {
        return 10000;
      }
      return false;
    },
  });

  const getStatusColor = (status: OptimizationStatus) => {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300';
      case 'running':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300';
      case 'cancelled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300';
      case 'queued':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300';
      case 'paused':
        return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300';
    }
  };

  const formatDuration = (startTime: string, endTime?: string) => {
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const durationMs = end.getTime() - start.getTime();
    
    if (durationMs < 60000) {
      return `${Math.floor(durationMs / 1000)}s`;
    } else if (durationMs < 3600000) {
      return `${Math.floor(durationMs / 60000)}m ${Math.floor((durationMs % 60000) / 1000)}s`;
    } else {
      return `${Math.floor(durationMs / 3600000)}h ${Math.floor((durationMs % 3600000) / 60000)}m`;
    }
  };

  const sortResults = (results: OptimizationRunResult[]) => {
    if (!results) return [];
    
    const sorted = [...results].sort((a, b) => {
      let aValue = a[sortField];
      let bValue = b[sortField];
      
      // Handle nested values for performance metrics
      if (sortField.includes('.')) {
        const [parent, child] = sortField.split('.');
        aValue = (a as any)[parent]?.[child];
        bValue = (b as any)[parent]?.[child];
      }
      
      if (aValue == null) return 1;
      if (bValue == null) return -1;
      
      if (typeof aValue === 'number' && typeof bValue === 'number') {
        return sortDirection === 'asc' ? aValue - bValue : bValue - aValue;
      }
      
      return sortDirection === 'asc' 
        ? String(aValue).localeCompare(String(bValue))
        : String(bValue).localeCompare(String(aValue));
    });
    
    return sorted;
  };

  const handleSort = (field: keyof OptimizationRunResult) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const getSortIcon = (field: keyof OptimizationRunResult) => {
    if (sortField !== field) {
      return (
        <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
        </svg>
      );
    }
    
    return sortDirection === 'asc' ? (
      <svg className="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4l6 6 4-4 8 8" />
      </svg>
    ) : (
      <svg className="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 20l-6-6-4 4-8-8" />
      </svg>
    );
  };

  if (isLoadingOptimization) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-12">
          <div className="flex items-center space-x-2">
            <svg className="animate-spin h-5 w-5 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span className="text-gray-500">Loading optimization...</span>
          </div>
        </div>
      </Layout>
    );
  }

  if (!optimization) {
    return (
      <Layout>
        <div className="text-center py-12">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">Optimization not found</h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            The optimization you're looking for doesn't exist or has been deleted.
          </p>
          <div className="mt-6">
            <Link href="/optimizations" className="btn-primary">
              Back to Optimizations
            </Link>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center space-x-3">
              <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                {optimization.config.name}
              </h1>
              <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(optimization.status)}`}>
                {optimization.status}
              </span>
            </div>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              {optimization.config.description || 'No description provided'}
            </p>
          </div>
          <div className="flex space-x-3">
            <Link
              href="/optimizations"
              className="btn-secondary"
            >
              Back to List
            </Link>
            {(optimization.status === 'running' || optimization.status === 'queued') && (
              <div className="flex space-x-2">
                <button
                  onClick={async () => {
                    await optimizationApi.pause(optimization.id);
                    refetchOptimization();
                  }}
                  className="px-4 py-2 text-sm font-medium text-orange-700 bg-orange-100 border border-orange-300 rounded-md hover:bg-orange-200 focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2 dark:bg-orange-900/30 dark:text-orange-300 dark:border-orange-600 dark:hover:bg-orange-900/50"
                >
                  Pause
                </button>
                <button
                  onClick={async () => {
                    if (window.confirm('Are you sure you want to cancel this optimization? This action cannot be undone.')) {
                      await optimizationApi.cancel(optimization.id);
                      refetchOptimization();
                    }
                  }}
                  className="btn-danger"
                >
                  Cancel
                </button>
              </div>
            )}
            {optimization.status === 'paused' && (
              <div className="flex space-x-2">
                <button
                  onClick={async () => {
                    await optimizationApi.resume(optimization.id);
                    refetchOptimization();
                  }}
                  className="px-4 py-2 text-sm font-medium text-green-700 bg-green-100 border border-green-300 rounded-md hover:bg-green-200 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 dark:bg-green-900/30 dark:text-green-300 dark:border-green-600 dark:hover:bg-green-900/50"
                >
                  Resume
                </button>
                <button
                  onClick={async () => {
                    if (window.confirm('Are you sure you want to cancel this optimization? This action cannot be undone.')) {
                      await optimizationApi.cancel(optimization.id);
                      refetchOptimization();
                    }
                  }}
                  className="btn-danger"
                >
                  Cancel
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Progress Bar */}
        {(optimization.status === 'running' || optimization.status === 'queued' || optimization.status === 'paused') && (
          <div className="card">
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Progress</h3>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
              <div
                className="bg-blue-600 h-3 rounded-full transition-all duration-300"
                style={{ width: `${Math.max(0, Math.min(100, optimization.progress))}%` }}
              />
            </div>
            <div className="flex justify-between text-sm text-gray-500 dark:text-gray-400 mt-2">
              <span>{optimization.completed_runs}/{optimization.total_permutations} completed</span>
              <span>{Math.round(optimization.progress)}%</span>
            </div>
            {optimization.status_message && (
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                {optimization.status_message}
              </p>
            )}
          </div>
        )}

        {/* Tabs */}
        <div className="border-b border-gray-200 dark:border-gray-700">
          <nav className="-mb-px flex space-x-8">
            <button
              onClick={() => setSelectedTab('overview')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                selectedTab === 'overview'
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              }`}
            >
              Overview
            </button>
            <button
              onClick={() => setSelectedTab('results')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                selectedTab === 'results'
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              }`}
            >
              Results {results && results.length > 0 && `(${results.length})`}
            </button>
            <button
              onClick={() => setSelectedTab('parameters')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                selectedTab === 'parameters'
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              }`}
            >
              Parameters
            </button>
          </nav>
        </div>

        {/* Tab Content */}
        {selectedTab === 'overview' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Configuration Summary */}
            <div className="card">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Configuration</h3>
              <dl className="space-y-3">
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Symbol</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">{optimization.config.base_run_config.symbol}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Date Range</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    {new Date(optimization.config.base_run_config.start_time).toLocaleDateString()} - {new Date(optimization.config.base_run_config.end_time).toLocaleDateString()}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Starting Balance</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    ${optimization.config.base_run_config.broker.starting_balance.toLocaleString()}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Optimization Metric</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    {optimization.config.optimization_metric.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Execution Order</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    {optimization.config.random_order ? 'Random' : 'Sequential'}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Total Permutations</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    {optimization.total_permutations?.toLocaleString() || 0}
                  </dd>
                </div>
              </dl>
            </div>

            {/* Statistics */}
            <div className="card">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Statistics</h3>
              <dl className="space-y-3">
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Status</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">{optimization.status}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Duration</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    {formatDuration(optimization.started_at, optimization.completed_at)}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Completed</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">
                    {optimization.completed_runs} / {optimization.total_permutations}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Failed</dt>
                  <dd className="text-sm text-gray-900 dark:text-gray-100">{optimization.failed_runs || 0}</dd>
                </div>
                {optimization.results && (
                  <>
                    <div>
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Best Return</dt>
                      <dd className={`text-sm font-medium ${
                        optimization.results.best_result?.performance_metrics?.return_percentage >= 0 
                          ? 'text-green-600 dark:text-green-400' 
                          : 'text-red-600 dark:text-red-400'
                      }`}>
                        {optimization.results.best_result?.performance_metrics?.return_percentage?.toFixed(2) || 'N/A'}%
                      </dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Average Return</dt>
                      <dd className={`text-sm ${
                        optimization.results.average_return >= 0 
                          ? 'text-green-600 dark:text-green-400' 
                          : 'text-red-600 dark:text-red-400'
                      }`}>
                        {optimization.results.average_return?.toFixed(2) || 'N/A'}%
                      </dd>
                    </div>
                  </>
                )}
              </dl>
            </div>
          </div>
        )}

        {selectedTab === 'results' && (
          <div className="space-y-4">
            {isLoadingResults ? (
              <div className="flex items-center justify-center py-12">
                <div className="flex items-center space-x-2">
                  <svg className="animate-spin h-5 w-5 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span className="text-gray-500">Loading results...</span>
                </div>
              </div>
            ) : results && results.length > 0 ? (
              <div className="card overflow-hidden">
                <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                  <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                    Optimization Results ({results.length})
                  </h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    Click column headers to sort results
                  </p>
                </div>
                
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-800">
                      <tr>
                        <th 
                          className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                          onClick={() => handleSort('rank')}
                        >
                          <div className="flex items-center space-x-1">
                            <span>Rank</span>
                            {getSortIcon('rank')}
                          </div>
                        </th>
                        <th 
                          className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                          onClick={() => handleSort('optimization_score')}
                        >
                          <div className="flex items-center space-x-1">
                            <span>Score</span>
                            {getSortIcon('optimization_score')}
                          </div>
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Return %
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Total Profit
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Win Rate
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Max DD
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Trades
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Parameters
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                      {sortResults(results).map((result, index) => (
                        <tr key={result.optimization_run_id} className={`hover:bg-gray-50 dark:hover:bg-gray-800 ${
                          index === 0 && sortField === 'optimization_score' && sortDirection === 'desc' 
                            ? 'bg-green-50 dark:bg-green-900/10' 
                            : ''
                        }`}>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                            {index === 0 && sortField === 'optimization_score' && sortDirection === 'desc' && (
                              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 mr-2">
                                🏆 Best
                              </span>
                            )}
                            #{result.rank}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-gray-100">
                            {result.optimization_score.toFixed(2)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm">
                            {result.performance_metrics ? (
                              <span className={`font-medium ${
                                result.performance_metrics.return_percentage >= 0 
                                  ? 'text-green-600 dark:text-green-400' 
                                  : 'text-red-600 dark:text-red-400'
                              }`}>
                                {result.performance_metrics.return_percentage.toFixed(2)}%
                              </span>
                            ) : (
                              <span className="text-gray-400">-</span>
                            )}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm">
                            {result.performance_metrics ? (
                              <span className={`${
                                result.performance_metrics.total_profit >= 0 
                                  ? 'text-green-600 dark:text-green-400' 
                                  : 'text-red-600 dark:text-red-400'
                              }`}>
                                ${result.performance_metrics.total_profit.toFixed(2)}
                              </span>
                            ) : (
                              <span className="text-gray-400">-</span>
                            )}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                            {result.performance_metrics?.win_rate?.toFixed(1) || '-'}%
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-red-600 dark:text-red-400">
                            {result.performance_metrics?.max_drawdown_percent?.toFixed(2) || '-'}%
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                            {result.performance_metrics?.total_trades || '-'}
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">
                            <div className="max-w-xs">
                              {Object.entries(result.parameters).map(([key, value]) => (
                                <div key={key} className="text-xs">
                                  <span className="text-gray-500 dark:text-gray-400">{key}:</span> {String(value)}
                                </div>
                              ))}
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                            <Link
                              href={`/runs/${result.backtest_run_id}`}
                              className="text-primary-600 dark:text-primary-400 hover:text-primary-900 dark:hover:text-primary-300"
                            >
                              View Backtest
                            </Link>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : (
              <div className="card text-center py-12">
                <svg
                  className="mx-auto h-12 w-12 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                  />
                </svg>
                <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">No results yet</h3>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Results will appear here as the optimization progresses.
                </p>
              </div>
            )}
          </div>
        )}

        {selectedTab === 'parameters' && (
          <div className="card">
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Parameter Ranges</h3>
            
            {optimization.config.parameter_ranges.length > 0 ? (
              <div className="space-y-4">
                {optimization.config.parameter_ranges.map((range: any, index: number) => (
                  <div key={index} className="border rounded-lg p-4 bg-gray-50 dark:bg-gray-800">
                    <div className="flex justify-between items-start mb-2">
                      <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                        {range.parameter_path}
                      </h4>
                      <span className="px-2 py-1 text-xs bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300 rounded">
                        {range.parameter_type}
                      </span>
                    </div>
                    
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Min:</span>
                        <span className="ml-1 text-gray-900 dark:text-gray-100">{String(range.lower_bound)}</span>
                      </div>
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Max:</span>
                        <span className="ml-1 text-gray-900 dark:text-gray-100">{String(range.upper_bound)}</span>
                      </div>
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Step:</span>
                        <span className="ml-1 text-gray-900 dark:text-gray-100">{String(range.step)}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">No parameter ranges defined.</p>
            )}
          </div>
        )}
      </div>
    </Layout>
  );
}