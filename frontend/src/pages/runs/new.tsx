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
      timeframe: '1m',
      startingBalance: 1700,
      trailingStopAmount: 10,
      feePerSide: 1,
      openSlippage: 0.25,
      selectedSignals: [],
    },
  });

  const selectedSymbol = watch('symbol');

  // Query available options
  const { data: symbols } = useQuery({
    queryKey: ['symbols'],
    queryFn: configApi.getSymbols,
  });

  const { data: timeframes } = useQuery({
    queryKey: ['timeframes'],
    queryFn: configApi.getTimeframes,
  });

  const { data: availableSignals } = useQuery({
    queryKey: ['availableSignals'],
    queryFn: signalsApi.getAvailable,
  });

  const { data: signalDefinitions } = useQuery({
    queryKey: ['signalDefinitions'],
    queryFn: signalsApi.list,
  });

  const createRunMutation = useMutation({
    mutationFn: runsApi.create,
    onSuccess: (data) => {
      router.push(`/runs/${data.id}`);
    },
    onError: (error) => {
      console.error('Failed to create run:', error);
      setIsSubmitting(false);
    },
  });

  const onSubmit = async (data: BacktestFormData) => {
    setIsSubmitting(true);

    const symbolConfig = getSymbolConfig(data.symbol);
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
        starting_balance: data.startingBalance,
        weekly_withdrawl: 0,
        trailing_stop_amount: data.trailingStopAmount,
        fee_per_side: data.feePerSide,
        open_slippage: data.openSlippage,
        symbol: {
          name: data.symbol,
          margin: symbolConfig.margin,
          point_price: symbolConfig.pointPrice,
        },
      },
      strategies: [
        {
          type: 'SC',
          symbol: data.symbol,
          params: {
            values: ['12B'],
          },
        },
      ],
      signals: data.selectedSignals,
    };

    createRunMutation.mutate(runConfig);
  };

  // Fetch data availability when symbol changes
  const { data: dataAvailability } = useQuery({
    queryKey: ['dataAvailability', selectedSymbol],
    queryFn: () => configApi.getSymbolDataAvailability(selectedSymbol),
    enabled: !!selectedSymbol,
  });

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
                <select
                  {...register('symbol', { required: 'Symbol is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                >
                  {symbols?.map((symbol) => (
                    <option key={symbol.name} value={symbol.name}>
                      {symbol.name} - {symbol.description}
                    </option>
                  ))}
                </select>
                {errors.symbol && <p className="mt-1 text-sm text-red-600">{errors.symbol.message}</p>}
              </div>

              <div>
                <label htmlFor="timeframe" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Timeframe
                </label>
                <select
                  {...register('timeframe', { required: 'Timeframe is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                >
                  {timeframes?.map((tf) => (
                    <option key={tf.value} value={tf.value}>
                      {tf.description}
                    </option>
                  ))}
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
                  disabled={!dataAvailability}
                />
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
                  disabled={!dataAvailability}
                />
                {errors.endDate && <p className="mt-1 text-sm text-red-600">{errors.endDate.message}</p>}
              </div>

              {dataAvailability && (
                <div className="sm:col-span-2 text-sm text-gray-500 dark:text-gray-400">
                  Data available from {new Date(dataAvailability.earliest_data).toLocaleDateString()} to {new Date(dataAvailability.latest_data).toLocaleDateString()}
                  ({dataAvailability.total_records.toLocaleString()} records)
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
                    min: { value: 100, message: 'Minimum balance is $100' }
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
                    min: { value: 0.1, message: 'Minimum trailing stop is 0.1' }
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
                    min: { value: 0, message: 'Fee cannot be negative' }
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
                    min: { value: 0, message: 'Slippage cannot be negative' }
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
              {availableSignals?.map((signal) => (
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
              ))}
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