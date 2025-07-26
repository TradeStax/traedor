import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { format } from 'date-fns';
import Layout from '@/components/Layout';
import { runsApi } from '@/lib/api';
import { Run } from '@/types';

export default function HomePage() {
  const [selectedStatus, setSelectedStatus] = useState<string>('');
  
  const { data: runs, isLoading, error } = useQuery({
    queryKey: ['runs', selectedStatus],
    queryFn: () => runsApi.list(selectedStatus ? { status: selectedStatus } : undefined),
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

  const formatNumber = (num: number) => {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(num);
  };

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">Backtest Runs</h1>
          <Link href="/runs/new" className="btn-primary w-full sm:w-auto text-center">
            New Backtest
          </Link>
        </div>

        <div className="flex flex-col sm:flex-row gap-4">
          <select
            value={selectedStatus}
            onChange={(e) => setSelectedStatus(e.target.value)}
            className="w-full sm:w-auto rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
          >
            <option value="">All Statuses</option>
            <option value="pending">Pending</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        {isLoading && (
          <div className="text-center py-12">
            <div className="inline-flex items-center">
              <svg className="animate-spin h-5 w-5 mr-3 text-primary-600 dark:text-primary-400" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <span className="text-gray-900 dark:text-gray-100">Loading runs...</span>
            </div>
          </div>
        )}

        {error && (
          <div className="text-red-600 dark:text-red-400 text-center py-12">
            Error loading runs. Please try again.
          </div>
        )}

        {runs && runs.length === 0 && (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            No runs found. Create your first backtest!
          </div>
        )}

        {runs && runs.length > 0 && (
          <>
            {/* Desktop table view */}
            <div className="hidden sm:block bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
              <div className="flex bg-gray-50 dark:bg-gray-700 py-3 px-6 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                <div className="w-32">Symbol</div>
                <div className="w-32">Timeframe</div>
                <div className="w-32">Status</div>
                <div className="w-32">Return %</div>
                <div className="flex-1">Started</div>
                <div className="w-20">View</div>
              </div>
              {runs.map((run: Run) => (
                <div key={run.id} className="flex py-4 px-6 border-t border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700">
                  <div className="w-32 text-sm font-medium text-gray-900 dark:text-gray-100">
                    {run.config.symbol}
                  </div>
                  <div className="w-32 text-sm text-gray-500 dark:text-gray-400">
                    {run.config.timeframe}
                  </div>
                  <div className="w-32">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(run.status)}`}>
                      {run.status}
                    </span>
                  </div>
                  <div className="w-32 text-sm text-gray-500 dark:text-gray-400">
                    {run.performance_metrics ? (
                      <span className={run.performance_metrics.return_percentage >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}>
                        {formatNumber(run.performance_metrics.return_percentage)}%
                      </span>
                    ) : (
                      '-'
                    )}
                  </div>
                  <div className="flex-1 text-sm text-gray-500 dark:text-gray-400">
                    {format(new Date(run.started_at), 'MMM d, yyyy HH:mm')}
                  </div>
                  <div className="w-20 text-right text-sm font-medium">
                    <Link href={`/runs/${run.id}`} className="text-primary-600 dark:text-primary-400 hover:text-primary-900 dark:hover:text-primary-300">
                      View
                    </Link>
                  </div>
                </div>
              ))}
            </div>

            {/* Mobile card view */}
            <div className="sm:hidden space-y-4">
              {runs.map((run: Run) => (
                <div key={run.id} className="card">
                  <div className="flex justify-between items-start mb-3">
                    <div>
                      <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                        {run.config.symbol}
                      </h3>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {run.config.timeframe}
                      </p>
                    </div>
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(run.status)}`}>
                      {run.status}
                    </span>
                  </div>
                  
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-sm text-gray-500 dark:text-gray-400">Return:</span>
                      <span className="text-sm">
                        {run.performance_metrics ? (
                          <span className={run.performance_metrics.return_percentage >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}>
                            {formatNumber(run.performance_metrics.return_percentage)}%
                          </span>
                        ) : (
                          <span className="text-gray-500 dark:text-gray-400">-</span>
                        )}
                      </span>
                    </div>
                    
                    <div className="flex justify-between">
                      <span className="text-sm text-gray-500 dark:text-gray-400">Started:</span>
                      <span className="text-sm text-gray-900 dark:text-gray-100">
                        {format(new Date(run.started_at), 'MMM d, yyyy HH:mm')}
                      </span>
                    </div>
                  </div>
                  
                  <div className="mt-4">
                    <Link href={`/runs/${run.id}`} className="btn-primary w-full text-center">
                      View Details
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </Layout>
  );
}