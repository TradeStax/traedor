import { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import Layout from '@/components/Layout';
import { runsApi, signalsApi, configApi } from '@/lib/api';
import { RunConfig } from '@/types';

interface BacktestFormData {
  symbol: string;
  startDate: string;
  endDate: string;
  startingBalance: number;
  trailingStopAmount: number;
  profitTarget: number;
  feePerSide: number;
  openSlippage: number;
  selectedSignals: string[];
}

interface SelectedSignal {
  id: string;
  name: string;
  type: string;
  baseSignalId: string;
  parameters: Record<string, any>;
}

export default function NewBacktestPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedSignals, setSelectedSignals] = useState<SelectedSignal[]>([]);
  const [showSignalModal, setShowSignalModal] = useState(false);
  const [selectedSignalForConfig, setSelectedSignalForConfig] = useState<any>(null);
  const [signalParameters, setSignalParameters] = useState<Record<string, any>>({});
  
  const { register, handleSubmit, formState: { errors }, watch, setValue } = useForm<BacktestFormData>({
    defaultValues: {
      symbol: '/MES',
      startingBalance: 10000,
      trailingStopAmount: 10,
      profitTarget: 20,
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
    console.log('Selected signals:', selectedSignals);

    // Prepare signals with their base definition IDs and parameters
    const signalsWithParams = selectedSignals.map(signal => ({
      signal_definition_id: signal.baseSignalId,
      parameters: signal.parameters
    }));

    // Get symbol configuration
    const symbolConfig = getSymbolConfig(data.symbol);

    const runConfig: RunConfig = {
      symbol: data.symbol,
      timeframe: 'tick',
      start_time: new Date(data.startDate).toISOString(),
      end_time: new Date(data.endDate).toISOString(),
      datafeeds: [
        {
          type: 'Database',
          symbol: data.symbol,
          data_path: '',
          interval: 'tick',
        },
      ],
      broker: {
        type: 'Futures',
        starting_balance: Number(data.startingBalance),
        weekly_withdrawl: 0,
        trailing_stop_amount: Number(data.trailingStopAmount),
        profit_target: Number(data.profitTarget),
        fee_per_side: Number(data.feePerSide),
        open_slippage: Number(data.openSlippage),
        symbol: {
          name: data.symbol,
          margin: symbolConfig.margin,
          point_price: symbolConfig.pointPrice,
        },
      },
      strategies: [],
      signals_with_params: signalsWithParams,
    };

    console.log('Final run config being sent:', JSON.stringify(runConfig, null, 2));
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

  const openSignalModal = (signal: any) => {
    setSelectedSignalForConfig(signal);
    setSignalParameters({ ...signal.parameters });
    setShowSignalModal(true);
  };

  const addSignalWithParameters = () => {
    if (!selectedSignalForConfig) return;
    
    const newSignal: SelectedSignal = {
      id: `${selectedSignalForConfig.id || selectedSignalForConfig.name}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      name: selectedSignalForConfig.name,
      type: selectedSignalForConfig.type,
      baseSignalId: selectedSignalForConfig.id || selectedSignalForConfig.name,
      parameters: { ...signalParameters }
    };
    setSelectedSignals(prev => [...prev, newSignal]);
    setShowSignalModal(false);
    setSelectedSignalForConfig(null);
    setSignalParameters({});
  };

  const removeSignal = (signalId: string) => {
    setSelectedSignals(prev => prev.filter(s => s.id !== signalId));
  };

  const getParameterDescription = (key: string, defaultValue: any): string => {
    const descriptions: Record<string, string> = {
      'period': 'Number of periods to use for calculation',
      'overbought_level': 'RSI level considered overbought (typically 70-80)',
      'oversold_level': 'RSI level considered oversold (typically 20-30)',
      'short_period': 'Number of periods for short moving average',
      'long_period': 'Number of periods for long moving average',
    };
    
    return descriptions[key] || `Default value: ${defaultValue}`;
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

              <div className="sm:col-span-2">
                <div className="rounded-md bg-blue-50 dark:bg-blue-900/20 p-3">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg className="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200">
                        Tick Data
                      </h3>
                      <div className="mt-1 text-sm text-blue-700 dark:text-blue-300">
                        All backtests are run using tick-level data for maximum accuracy.
                      </div>
                    </div>
                  </div>
                </div>
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
                  Trailing Stop Amount (points)
                </label>
                <input
                  type="number"
                  step="0.25"
                  {...register('trailingStopAmount', { 
                    required: 'Trailing stop amount is required',
                    min: { value: 0.25, message: 'Minimum trailing stop is 0.25 points' },
                    valueAsNumber: true
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                />
                {errors.trailingStopAmount && <p className="mt-1 text-sm text-red-600">{errors.trailingStopAmount.message}</p>}
              </div>

              <div>
                <label htmlFor="profitTarget" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Profit Target (points)
                </label>
                <input
                  type="number"
                  step="0.25"
                  {...register('profitTarget', { 
                    required: 'Profit target is required',
                    min: { value: 0.25, message: 'Minimum profit target is 0.25 points' },
                    valueAsNumber: true
                  })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
                />
                {errors.profitTarget && <p className="mt-1 text-sm text-red-600">{errors.profitTarget.message}</p>}
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
            
            {/* Available Signals */}
            <div className="mb-6">
              <h3 className="text-md font-medium text-gray-700 dark:text-gray-300 mb-3">Available Signals</h3>
              {isLoadingSignals ? (
                <div className="flex items-center text-sm text-gray-500 dark:text-gray-400">
                  <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  Loading available signals...
                </div>
              ) : (
                <div className="grid grid-cols-1 gap-3">
                  {availableSignals?.map((signal) => (
                    <div key={signal.name} className="border rounded-lg p-4 dark:border-gray-600">
                      <div className="flex justify-between items-start mb-2">
                        <div>
                          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">{signal.name}</h4>
                          <p className="text-sm text-gray-500 dark:text-gray-400">{signal.description}</p>
                        </div>
                      </div>
                      <div className="flex justify-end mt-3">
                        <button
                          type="button"
                          onClick={() => openSignalModal(signal)}
                          className="px-4 py-2 text-sm bg-primary-100 text-primary-800 dark:bg-primary-900/30 dark:text-primary-300 rounded hover:bg-primary-200 dark:hover:bg-primary-900/50 font-medium"
                        >
                          Add Signal
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Selected Signals */}
            <div>
              <h3 className="text-md font-medium text-gray-700 dark:text-gray-300 mb-3">Selected Signals</h3>
              {selectedSignals.length === 0 ? (
                <p className="text-sm text-gray-500 dark:text-gray-400">No signals selected</p>
              ) : (
                <div className="space-y-3">
                  {selectedSignals.map((signal) => (
                    <div key={signal.id} className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border">
                      <div className="flex justify-between items-start">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                              {signal.name}
                            </span>
                            <span className="px-2 py-0.5 text-xs bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300 rounded">
                              {signal.type}
                            </span>
                            {signal.parameters.aggregation_interval && (
                              <span className="px-2 py-0.5 text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 rounded">
                                {signal.parameters.aggregation_interval}m bars
                              </span>
                            )}
                          </div>
                          {Object.keys(signal.parameters).length > 0 && (
                            <div className="text-xs text-gray-500 dark:text-gray-400">
                              Parameters: {Object.entries(signal.parameters).map(([key, value]) => 
                                `${key}=${value}`
                              ).join(', ')}
                            </div>
                          )}
                        </div>
                        <button
                          type="button"
                          onClick={() => removeSignal(signal.id)}
                          className="ml-3 text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 p-1"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path>
                          </svg>
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
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

      {/* Signal Configuration Modal */}
      {showSignalModal && selectedSignalForConfig && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-md max-h-[90vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Configure {selectedSignalForConfig.name}
              </h3>
              <button
                onClick={() => setShowSignalModal(false)}
                className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
              </button>
            </div>
            
            {/* Modal Body */}
            <div className="p-4 space-y-4">
              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                  {selectedSignalForConfig.description}
                </p>
              </div>

              {/* Aggregation Interval */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Time Aggregation
                </label>
                <select
                  value={signalParameters.aggregation_interval || ''}
                  onChange={(e) => {
                    const value = e.target.value;
                    setSignalParameters(prev => ({
                      ...prev,
                      ...(value ? { aggregation_interval: parseInt(value) } : { aggregation_interval: undefined })
                    }));
                  }}
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm 
                           bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                           focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="">Tick Data (no aggregation)</option>
                  <option value="1">1 Minute Bars</option>
                  <option value="5">5 Minute Bars</option>
                  <option value="15">15 Minute Bars</option>
                  <option value="30">30 Minute Bars</option>
                  <option value="60">1 Hour Bars</option>
                </select>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Choose time-based aggregation or use raw tick data
                </p>
              </div>

              {/* Other Parameters */}
              {Object.entries(selectedSignalForConfig.parameters || {}).map(([key, defaultValue]) => {
                if (key === 'aggregation_interval') return null; // Already handled above
                
                const isNumber = typeof defaultValue === 'number';
                const currentValue = signalParameters[key] ?? defaultValue;
                
                return (
                  <div key={key}>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                    </label>
                    <input
                      type={isNumber ? 'number' : 'text'}
                      value={currentValue}
                      onChange={(e) => {
                        const value = isNumber 
                          ? (e.target.value === '' ? '' : parseFloat(e.target.value))
                          : e.target.value;
                        setSignalParameters(prev => ({
                          ...prev,
                          [key]: value
                        }));
                      }}
                      step={isNumber ? (key.includes('level') ? '0.1' : '1') : undefined}
                      min={isNumber ? '0' : undefined}
                      className="w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm 
                               bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                               focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                      placeholder={`Default: ${defaultValue}`}
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {getParameterDescription(key, defaultValue)}
                    </p>
                  </div>
                );
              })}
            </div>
            
            {/* Modal Footer */}
            <div className="flex flex-col-reverse sm:flex-row gap-3 p-4 border-t border-gray-200 dark:border-gray-700">
              <button
                type="button"
                onClick={() => setShowSignalModal(false)}
                className="w-full sm:w-auto px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 
                         bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md 
                         hover:bg-gray-50 dark:hover:bg-gray-600 focus:ring-2 focus:ring-gray-500"
              >
                Cancel
              </button>
              <button 
                type="button"
                onClick={addSignalWithParameters}
                className="w-full sm:w-auto px-4 py-2 text-sm font-medium text-white 
                         bg-primary-600 hover:bg-primary-700 rounded-md focus:ring-2 focus:ring-primary-500"
              >
                Add Signal
              </button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  );
}