import { useState } from 'react';
import { useRouter } from 'next/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import Layout from '@/components/Layout';
import { runsApi, signalsApi, configApi } from '@/lib/api';
import { RunConfig } from '@/types';

interface BacktestFormData {
  symbol: string;
  timeframe: string;
  dataPath: string;
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
      router.push(`/runs/${data.run_id}`);
    },
    onError: (error) => {
      console.error('Failed to create run:', error);
      setIsSubmitting(false);
    },
  });

  const onSubmit = async (data: BacktestFormData) => {
    setIsSubmitting(true);

    const runConfig: RunConfig = {
      symbol: data.symbol,
      timeframe: data.timeframe,
      start_time: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days ago
      end_time: new Date().toISOString(),
      datafeeds: [
        {
          type: 'SC',
          symbol: data.symbol,
          data_path: data.dataPath,
          interval: '0ns',
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
          margin: getMarginForSymbol(data.symbol),
          point_price: getPointPriceForSymbol(data.symbol),
        },
      },
      strategies: [
        {
          type: 'SC',
          symbol: data.symbol,
          params: {
            data_path: data.dataPath.replace('.txt', '-1Min.txt'),
            values: ['12B'],
          },
        },
      ],
      signals: data.selectedSignals,
    };

    createRunMutation.mutate(runConfig);
  };

  const getMarginForSymbol = (symbol: string): number => {
    const margins: Record<string, number> = {
      '/MES': 1200,
      '/MNQ': 1800,
      '/MYM': 800,
      '/M2K': 900,
      '/ES': 12000,
      '/NQ': 18000,
    };
    return margins[symbol] || 1200;
  };

  const getPointPriceForSymbol = (symbol: string): number => {
    const pointPrices: Record<string, number> = {
      '/MES': 5,
      '/MNQ': 2,
      '/MYM': 0.5,
      '/M2K': 5,
      '/ES': 50,
      '/NQ': 20,
    };
    return pointPrices[symbol] || 5;
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
          <h1 className="text-2xl font-semibold text-gray-900">New Backtest</h1>
          <p className="mt-1 text-sm text-gray-600">
            Configure and start a new backtesting run.
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Market Configuration</h2>
            
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
              <div>
                <label htmlFor="symbol" className="block text-sm font-medium text-gray-700">
                  Symbol
                </label>
                <select
                  {...register('symbol', { required: 'Symbol is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
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
                <label htmlFor="timeframe" className="block text-sm font-medium text-gray-700">
                  Timeframe
                </label>
                <select
                  {...register('timeframe', { required: 'Timeframe is required' })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                >
                  {timeframes?.map((tf) => (
                    <option key={tf.value} value={tf.value}>
                      {tf.description}
                    </option>
                  ))}
                </select>
                {errors.timeframe && <p className="mt-1 text-sm text-red-600">{errors.timeframe.message}</p>}
              </div>

              <div className="sm:col-span-2">
                <label htmlFor="dataPath" className="block text-sm font-medium text-gray-700">
                  Data Path
                </label>
                <input
                  type="text"
                  {...register('dataPath', { required: 'Data path is required' })}
                  placeholder="./data/MESH23_FUT_CME.txt"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                />
                {errors.dataPath && <p className="mt-1 text-sm text-red-600">{errors.dataPath.message}</p>}
              </div>
            </div>
          </div>

          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Broker Configuration</h2>
            
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
              <div>
                <label htmlFor="startingBalance" className="block text-sm font-medium text-gray-700">
                  Starting Balance ($)
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('startingBalance', { 
                    required: 'Starting balance is required',
                    min: { value: 100, message: 'Minimum balance is $100' }
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                />
                {errors.startingBalance && <p className="mt-1 text-sm text-red-600">{errors.startingBalance.message}</p>}
              </div>

              <div>
                <label htmlFor="trailingStopAmount" className="block text-sm font-medium text-gray-700">
                  Trailing Stop Amount
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('trailingStopAmount', { 
                    required: 'Trailing stop amount is required',
                    min: { value: 0.1, message: 'Minimum trailing stop is 0.1' }
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                />
                {errors.trailingStopAmount && <p className="mt-1 text-sm text-red-600">{errors.trailingStopAmount.message}</p>}
              </div>

              <div>
                <label htmlFor="feePerSide" className="block text-sm font-medium text-gray-700">
                  Fee Per Side ($)
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('feePerSide', { 
                    required: 'Fee per side is required',
                    min: { value: 0, message: 'Fee cannot be negative' }
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                />
                {errors.feePerSide && <p className="mt-1 text-sm text-red-600">{errors.feePerSide.message}</p>}
              </div>

              <div>
                <label htmlFor="openSlippage" className="block text-sm font-medium text-gray-700">
                  Open Slippage
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...register('openSlippage', { 
                    required: 'Open slippage is required',
                    min: { value: 0, message: 'Slippage cannot be negative' }
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                />
                {errors.openSlippage && <p className="mt-1 text-sm text-red-600">{errors.openSlippage.message}</p>}
              </div>
            </div>
          </div>

          <div className="card">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Signal Configuration</h2>
            
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
                    <span className="text-sm font-medium text-gray-700">{signal.name}</span>
                    <span className="text-sm text-gray-500 ml-2">{signal.description}</span>
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