import { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import Layout from '@/components/Layout';
import { runsApi, signalsApi, configApi } from '@/lib/api';
import { RunConfig } from '@/types';

interface BacktestFormData {
  symbol: string;
  timeframe: string;
  startDate: string;
  endDate: string;
  startingBalance: number;
  trailingStopAmount: number;
  feePerSide: number;
  openSlippage: number;
  selectedSignals: string[];
}

export default function NewBacktestPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const { register, handleSubmit, formState: { errors }, watch, setValue } = useForm<BacktestFormData>({
    defaultValues: {
      symbol: '/MES',
      timeframe: '30m',
      startingBalance: 1700,
      trailingStopAmount: 10,
      feePerSide: 1,
      openSlippage: 0.25,
      selectedSignals: [],
    },
  });

  const selectedSymbol = watch('symbol');

  // Query available options
  const { data: symbols, isLoading: isLoadingSymbols } = useQuery({
    queryKey: ['symbols'],
    queryFn: configApi.getSymbols,
  });

  const { data: timeframes, isLoading: isLoadingTimeframes } = useQuery({
    queryKey: ['timeframes'],
    queryFn: configApi.getTimeframes,
  });

  const { data: availableSignals, isLoading: isLoadingSignals } = useQuery({
    queryKey: ['availableSignals'],
    queryFn: signalsApi.getAvailable,
  });

  const { data: signalDefinitions } = useQuery({
    queryKey: ['signalDefinitions'],
    queryFn: signalsApi.list,
  });

  // Fetch data availability when symbol changes
  const { data: dataAvailability, isLoading: isLoadingDataAvailability } = useQuery({
    queryKey: ['dataAvailability', selectedSymbol],
    queryFn: () => configApi.getSymbolDataAvailability(selectedSymbol),
    enabled: !!selectedSymbol,
    staleTime: 0, // Force fresh data every time
  });


  const createRunMutation = useMutation({
    mutationFn: runsApi.create,
    onSuccess: (data) => {
      router.push(`/runs/${data.id}`);
    },
    onError: (error: any) => {
      console.error('Failed to create run:', error);
      console.error('Error response:', error.response);
      console.error('Error config:', error.config);
      if (error.response) {
        console.error('Response data:', error.response.data);
        console.error('Response status:', error.response.status);
        console.error('Response headers:', error.response.headers);
      }
      setIsSubmitting(false);
    },
  });

  const onSubmit = async (data: BacktestFormData) => {
    setIsSubmitting(true);
    
    // Debug form data
    console.log('Form data on submit:', data);
    console.log('Available timeframes:', timeframes);
    console.log('Current form timeframe value:', data.timeframe);

    const runConfig: RunConfig = {
      symbol: data.symbol,
      timeframe: data.timeframe,
      start_time: new Date(data.startDate).toISOString(),
      end_time: new Date(data.endDate).toISOString(),
      datafeeds: [
        {
          type: 'Database',
          symbol: data.symbol,
          data_path: '',
          interval: data.timeframe,
        },
      ],
      broker: {
        type: 'Futures',
        starting_balance: Number(data.startingBalance),
        weekly_withdrawl: 0,
        trailing_stop_amount: Number(data.trailingStopAmount),
        fee_per_side: Number(data.feePerSide),
        open_slippage: Number(data.openSlippage),
        symbol: {
          name: data.symbol,
          margin: 0,
          point_price: 0,
        },
      },
      strategies: [],
      signals: data.selectedSignals,
    };

    console.log('Final run config being sent:', JSON.stringify(runConfig, null, 2));
    console.log('Timeframe in config:', runConfig.timeframe);
    console.log('Interval in datafeed:', runConfig.datafeeds[0].interval);
    createRunMutation.mutate(runConfig);
  };

  // Update date range when data availability loads
  useEffect(() => {
    if (dataAvailability) {
      setValue('startDate', new Date(dataAvailability.earliest_data).toISOString().split('T')[0]);
      setValue('endDate', new Date(dataAvailability.latest_data).toISOString().split('T')[0]);
    }
  }, [dataAvailability, setValue]);

  const getSymbolConfig = (symbolName: string) => {
    const symbol = symbols?.find(s => s.name === symbolName);
    return {
      margin: symbol?.margin || 1200,
      pointPrice: symbol?.point_price || 5,
    };
  };

  const handleSignalSelection = (signalId: string, checked: boolean) => {
    const currentSignals = watch('selectedSignals');
    if (checked) {
      setValue('selectedSignals', [...currentSignals, signalId]);
    } else {
      setValue('selectedSignals', currentSignals.filter(id => id !== signalId));
    }
  };

  return (
    <Layout>
      <div className="max-w-2xl mx-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">New Backtest</h1>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Configure and start a new backtesting run.
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Market Configuration</h2>
            
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
              <div>
                <label htmlFor="symbol" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Symbol
                </label>
                <div className="relative">
                  <select
                    {...register('symbol', { required: 'Symbol is required' })}
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                    disabled={isLoadingSymbols}
                  >
                    {isLoadingSymbols ? (
                      <option>Loading symbols...</option>
                    ) : (
                      symbols?.map((symbol) => (
                        <option key={symbol.name} value={symbol.name}>
                          {symbol.name} - {symbol.description}
                        </option>
                      ))
                    )}
                  </select>
                  {isLoadingSymbols && (
                    <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
                      <svg className="animate-spin h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                    </div>
                  )}
                </div>
                {errors.symbol && <p className="mt-1 text-sm text-red-600">{errors.symbol.message}</p>}
              </div>

              <div>
                <label htmlFor="timeframe" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Timeframe
                </label>
                <select
                  {...register('timeframe', { required: 'Timeframe is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                  disabled={isLoadingTimeframes}
                >
                  {isLoadingTimeframes ? (
                    <option>Loading timeframes...</option>
                  ) : (
                    timeframes?.map((tf) => (
                      <option key={tf.value} value={tf.value}>
                        {tf.description}
                      </option>
                    ))
                  )}
                </select>
                {errors.timeframe && <p className="mt-1 text-sm text-red-600">{errors.timeframe.message}</p>}
              </div>

              <div>
                <label htmlFor="startDate" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Start Date
                </label>
                <input
                  type="date"
                  {...register('startDate', { required: 'Start date is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                  disabled={!dataAvailability || isLoadingDataAvailability}
                />
                {isLoadingDataAvailability && (
                  <div className="mt-1 text-sm text-blue-600 dark:text-blue-400">
                    <div className="flex items-center">
                      <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      Loading date availability...
                    </div>
                  </div>
                )}
                {errors.startDate && <p className="mt-1 text-sm text-red-600">{errors.startDate.message}</p>}
              </div>

              <div>
                <label htmlFor="endDate" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  End Date
                </label>
                <input
                  type="date"
                  {...register('endDate', { required: 'End date is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                  disabled={!dataAvailability || isLoadingDataAvailability}
                />
                {errors.endDate && <p className="mt-1 text-sm text-red-600">{errors.endDate.message}</p>}
              </div>

              {dataAvailability && (
                <div className="sm:col-span-2 text-sm text-gray-500 dark:text-gray-400">
                  Data available from {new Date(dataAvailability.earliest_data).toLocaleDateString()} to {new Date(dataAvailability.latest_data).toLocaleDateString()}
                  ({dataAvailability.total_records.toLocaleString()} records)
                </div>
              )}
              
              {!dataAvailability && !isLoadingDataAvailability && selectedSymbol && (
                <div className="sm:col-span-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-md">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg className="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <h3 className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
                        No market data available
                      </h3>
                      <div className="mt-1 text-sm text-yellow-700 dark:text-yellow-300">
                        No market data has been imported for {selectedSymbol}. Please import data for this symbol before running a backtest.
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Broker Configuration</h2>
            
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
              <div>
                <label htmlFor="startingBalance" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Starting Balance ($)
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('startingBalance', { 
                    required: 'Starting balance is required',
                    min: { value: 100, message: 'Minimum balance is $100' },
                    valueAsNumber: true
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                />
                {errors.startingBalance && <p className="mt-1 text-sm text-red-600">{errors.startingBalance.message}</p>}
              </div>

              <div>
                <label htmlFor="trailingStopAmount" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Trailing Stop Amount
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('trailingStopAmount', { 
                    required: 'Trailing stop amount is required',
                    min: { value: 0.1, message: 'Minimum trailing stop is 0.1' },
                    valueAsNumber: true
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                />
                {errors.trailingStopAmount && <p className="mt-1 text-sm text-red-600">{errors.trailingStopAmount.message}</p>}
              </div>

              <div>
                <label htmlFor="feePerSide" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Fee Per Side ($)
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('feePerSide', { 
                    required: 'Fee per side is required',
                    min: { value: 0, message: 'Fee cannot be negative' },
                    valueAsNumber: true
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                />
                {errors.feePerSide && <p className="mt-1 text-sm text-red-600">{errors.feePerSide.message}</p>}
              </div>

              <div>
                <label htmlFor="openSlippage" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Open Slippage
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('openSlippage', { 
                    required: 'Open slippage is required',
                    min: { value: 0, message: 'Slippage cannot be negative' },
                    valueAsNumber: true
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                />
                {errors.openSlippage && <p className="mt-1 text-sm text-red-600">{errors.openSlippage.message}</p>}
              </div>
            </div>
          </div>

          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Signal Configuration</h2>
            
            <div className="space-y-3">
              {isLoadingSignals ? (
                <div className="flex items-center text-sm text-gray-500 dark:text-gray-400">
                  <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  Loading available signals...
                </div>
              ) : (
                availableSignals?.map((signal) => (
                  <div key={signal.name} className="flex items-center">
                    <input
                      type="checkbox"
                      id={signal.name}
                      onChange={(e) => handleSignalSelection(signal.name, e.target.checked)}
                      className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                    />
                    <label htmlFor={signal.name} className="ml-3">
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{signal.name}</span>
                      <span className="text-sm text-gray-500 dark:text-gray-400 ml-2">{signal.description}</span>
                    </label>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="flex justify-end space-x-3">
            <button
              type="button"
              onClick={() => router.back()}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting ? 'Starting Backtest...' : 'Start Backtest'}
            </button>
          </div>
        </form>
      </div>
    </Layout>
  );
}