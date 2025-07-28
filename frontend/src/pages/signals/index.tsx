import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import Layout from '@/components/Layout';
import { signalsApi } from '@/lib/api';
import { SignalDefinition } from '@/types';

export default function SignalsPage() {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingSignal, setEditingSignal] = useState<SignalDefinition | null>(null);
  const [selectedTab, setSelectedTab] = useState<'available' | 'custom'>('available');
  const queryClient = useQueryClient();

  const { data: signals, isLoading } = useQuery({
    queryKey: ['signalDefinitions'],
    queryFn: signalsApi.list,
  });

  const { data: availableSignals } = useQuery({
    queryKey: ['availableSignals'],
    queryFn: signalsApi.getAvailable,
  });

  const createSignalMutation = useMutation({
    mutationFn: signalsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signalDefinitions'] });
      setShowCreateForm(false);
    },
  });

  const updateSignalMutation = useMutation({
    mutationFn: ({ id, signal }: { id: string; signal: SignalDefinition }) =>
      signalsApi.update(id, signal),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signalDefinitions'] });
      setEditingSignal(null);
    },
  });

  const deleteSignalMutation = useMutation({
    mutationFn: signalsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signalDefinitions'] });
    },
  });

  const handleCreateSignal = (formData: FormData) => {
    const signalData = {
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      type: formData.get('type') as string,
      parameters: JSON.parse(formData.get('parameters') as string || '{}'),
      active: formData.get('active') === 'on',
    };

    createSignalMutation.mutate(signalData);
  };

  const handleUpdateSignal = (id: string, formData: FormData) => {
    const signalData = {
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      type: formData.get('type') as string,
      parameters: JSON.parse(formData.get('parameters') as string || '{}'),
      active: formData.get('active') === 'on',
    };

    updateSignalMutation.mutate({ id, signal: signalData });
  };

  const getTypeColor = (type: string) => {
    switch (type.toLowerCase()) {
      case 'rsi':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300';
      case 'sma_crossover':
        return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300';
      case 'macd':
        return 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300';
      case 'technical':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300';
      case 'ml':
        return 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300';
      case 'custom':
        return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300';
    }
  };

  const getAggregationDisplay = (parameters: any) => {
    if (parameters?.aggregation_interval) {
      return `${parameters.aggregation_interval}m`;
    }
    return 'Tick';
  };

  if (isLoading) {
    return (
      <Layout>
        <div className="flex justify-center items-center py-16">
          <div className="text-center">
            <div className="animate-spin h-8 w-8 border-2 border-primary-600 border-t-transparent rounded-full mx-auto mb-4"></div>
            <p className="text-gray-600 dark:text-gray-400">Loading signals...</p>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        {/* Header */}
        <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-40">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center h-16">
              <div>
                <h1 className="text-xl sm:text-2xl font-semibold text-gray-900 dark:text-gray-100">Signals</h1>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 hidden sm:block">
                  Manage trading signal definitions and parameters
                </p>
              </div>
              <button
                onClick={() => setShowCreateForm(true)}
                className="btn-primary text-sm px-3 py-2 sm:px-4 sm:py-2"
              >
                <span className="hidden sm:inline">New Signal</span>
                <span className="sm:hidden">New</span>
              </button>
            </div>
          </div>
        </div>

        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          {/* Tabs */}
          <div className="mb-6">
            <div className="border-b border-gray-200 dark:border-gray-700">
              <nav className="-mb-px flex space-x-8">
                <button
                  onClick={() => setSelectedTab('available')}
                  className={`py-2 px-1 border-b-2 font-medium text-sm ${
                    selectedTab === 'available'
                      ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                  }`}
                >
                  Available ({availableSignals?.length || 0})
                </button>
                <button
                  onClick={() => setSelectedTab('custom')}
                  className={`py-2 px-1 border-b-2 font-medium text-sm ${
                    selectedTab === 'custom'
                      ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                  }`}
                >
                  Custom ({signals?.length || 0})
                </button>
              </nav>
            </div>
          </div>

          {/* Available Signals Tab */}
          {selectedTab === 'available' && (
            <div className="space-y-4">
              {availableSignals?.length === 0 ? (
                <div className="text-center py-12">
                  <div className="w-16 h-16 mx-auto mb-4 bg-gray-100 dark:bg-gray-800 rounded-full flex items-center justify-center">
                    <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path>
                    </svg>
                  </div>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No signals available</h3>
                  <p className="text-gray-500 dark:text-gray-400">Check back later for available signal templates.</p>
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {availableSignals?.map((signal) => (
                    <div key={signal.id || signal.name} className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-shadow">
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex-1 min-w-0">
                          <h3 className="font-medium text-gray-900 dark:text-gray-100 truncate">{signal.name}</h3>
                          <div className="flex items-center gap-2 mt-1">
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${getTypeColor(signal.type)}`}>
                              {signal.type?.toUpperCase() || 'N/A'}
                            </span>
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                              {getAggregationDisplay(signal.parameters)}
                            </span>
                          </div>
                        </div>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mb-4 line-clamp-2">{signal.description}</p>
                      <button
                        onClick={() => {
                          setEditingSignal({
                            name: `${signal.name} Copy`,
                            description: signal.description,
                            type: signal.type || 'technical',
                            parameters: signal.parameters || {},
                            active: true,
                          });
                        }}
                        className="w-full bg-primary-50 text-primary-700 hover:bg-primary-100 dark:bg-primary-900/20 dark:text-primary-400 dark:hover:bg-primary-900/30 px-3 py-2 rounded-md text-sm font-medium transition-colors"
                      >
                        Create Copy
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Custom Signals Tab */}
          {selectedTab === 'custom' && (
            <div className="space-y-4">
              {signals?.length === 0 ? (
                <div className="text-center py-12">
                  <div className="w-16 h-16 mx-auto mb-4 bg-gray-100 dark:bg-gray-800 rounded-full flex items-center justify-center">
                    <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                    </svg>
                  </div>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No custom signals</h3>
                  <p className="text-gray-500 dark:text-gray-400 mb-4">Create your first custom signal to get started.</p>
                  <button
                    onClick={() => setShowCreateForm(true)}
                    className="btn-primary"
                  >
                    Create Signal
                  </button>
                </div>
              ) : (
                <div className="space-y-3">
                  {signals?.map((signal) => (
                    <div key={signal.id} className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
                      <div className="flex items-start gap-3">
                        <div className="flex-1 min-w-0">
                          {/* Header with name and action buttons */}
                          <div className="flex items-start justify-between mb-2">
                            <h3 className="font-medium text-gray-900 dark:text-gray-100 text-sm sm:text-base pr-2">
                              {signal.name}
                            </h3>
                            <div className="flex items-center gap-1 flex-shrink-0">
                              <button
                                onClick={() => setEditingSignal(signal)}
                                className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700"
                                title="Edit"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
                                </svg>
                              </button>
                              <button
                                onClick={() => {
                                  if (confirm('Are you sure you want to delete this signal?')) {
                                    deleteSignalMutation.mutate(signal.id!);
                                  }
                                }}
                                className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700"
                                title="Delete"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                                </svg>
                              </button>
                            </div>
                          </div>

                          {/* Badges - responsive layout */}
                          <div className="flex flex-wrap items-center gap-2 mb-3">
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${getTypeColor(signal.type)}`}>
                              {signal.type?.toUpperCase() || 'N/A'}
                            </span>
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                              {getAggregationDisplay(signal.parameters)}
                            </span>
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                              signal.active ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300' : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
                            }`}>
                              {signal.active ? 'Active' : 'Inactive'}
                            </span>
                          </div>

                          {/* Description */}
                          <p className="text-sm text-gray-600 dark:text-gray-400 mb-3 leading-relaxed">
                            {signal.description}
                          </p>

                          {/* Parameters - mobile friendly */}
                          {Object.keys(signal.parameters).length > 0 && (
                            <div className="bg-gray-50 dark:bg-gray-700/50 rounded-md p-3">
                              <div className="text-xs font-medium text-gray-700 dark:text-gray-300 mb-2">
                                Parameters:
                              </div>
                              <div className="space-y-1">
                                {Object.entries(signal.parameters).map(([key, value]) => (
                                  <div key={key} className="flex items-center justify-between text-xs">
                                    <span className="text-gray-600 dark:text-gray-400 font-medium">
                                      {key}:
                                    </span>
                                    <span className="font-mono bg-white dark:bg-gray-600 px-2 py-0.5 rounded text-gray-900 dark:text-gray-100">
                                      {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                                    </span>
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Create/Edit Form Modal */}
      {(showCreateForm || editingSignal) && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-lg max-h-[90vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {editingSignal ? 'Edit Signal' : 'Create Signal'}
              </h3>
              <button
                onClick={() => {
                  setShowCreateForm(false);
                  setEditingSignal(null);
                }}
                className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
              </button>
            </div>
            
            {/* Modal Body */}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                const formData = new FormData(e.currentTarget);
                if (editingSignal?.id) {
                  handleUpdateSignal(editingSignal.id, formData);
                } else {
                  handleCreateSignal(formData);
                }
              }}
              className="p-4 space-y-4"
            >
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Name
                </label>
                <input
                  type="text"
                  name="name"
                  defaultValue={editingSignal?.name || ''}
                  required
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm 
                           bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                           focus:ring-2 focus:ring-primary-500 focus:border-primary-500
                           placeholder-gray-400 dark:placeholder-gray-400"
                  placeholder="Enter signal name"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Description
                </label>
                <textarea
                  name="description"
                  defaultValue={editingSignal?.description || ''}
                  rows={3}
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm 
                           bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                           focus:ring-2 focus:ring-primary-500 focus:border-primary-500
                           placeholder-gray-400 dark:placeholder-gray-400 resize-none"
                  placeholder="Describe what this signal does"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Type
                </label>
                <select
                  name="type"
                  defaultValue={editingSignal?.type || 'rsi'}
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm 
                           bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                           focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="rsi">RSI</option>
                  <option value="sma_crossover">SMA Crossover</option>
                  <option value="macd">MACD</option>
                  <option value="technical">Technical</option>
                  <option value="ml">Machine Learning</option>
                  <option value="custom">Custom</option>
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Parameters (JSON)
                </label>
                <textarea
                  name="parameters"
                  defaultValue={JSON.stringify(editingSignal?.parameters || {}, null, 2)}
                  rows={6}
                  className="w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm 
                           bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                           focus:ring-2 focus:ring-primary-500 focus:border-primary-500
                           font-mono resize-none"
                  placeholder='{"period": 14, "aggregation_interval": 5}'
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Enter parameters as valid JSON. Add "aggregation_interval" for time-based signals.
                </p>
              </div>
              
              <div className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-md">
                <input
                  type="checkbox"
                  name="active"
                  id="active"
                  defaultChecked={editingSignal?.active !== false}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 dark:border-gray-600 rounded"
                />
                <label htmlFor="active" className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  Active Signal
                </label>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Inactive signals won't be available for selection
                </p>
              </div>
              
              {/* Modal Footer */}
              <div className="flex flex-col-reverse sm:flex-row gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateForm(false);
                    setEditingSignal(null);
                  }}
                  className="w-full sm:w-auto px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 
                           bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md 
                           hover:bg-gray-50 dark:hover:bg-gray-600 focus:ring-2 focus:ring-gray-500"
                >
                  Cancel
                </button>
                <button 
                  type="submit" 
                  disabled={createSignalMutation.isPending || updateSignalMutation.isPending}
                  className="w-full sm:w-auto px-4 py-2 text-sm font-medium text-white 
                           bg-primary-600 hover:bg-primary-700 disabled:bg-primary-400 
                           rounded-md focus:ring-2 focus:ring-primary-500 disabled:cursor-not-allowed"
                >
                  {createSignalMutation.isPending || updateSignalMutation.isPending ? (
                    <span className="flex items-center justify-center gap-2">
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      {editingSignal ? 'Updating...' : 'Creating...'}
                    </span>
                  ) : (
                    editingSignal ? 'Update Signal' : 'Create Signal'
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </Layout>
  );
}