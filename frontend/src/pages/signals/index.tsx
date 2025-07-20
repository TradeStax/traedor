import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import Layout from '@/components/Layout';
import { signalsApi } from '@/lib/api';
import { SignalDefinition } from '@/types';

export default function SignalsPage() {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingSignal, setEditingSignal] = useState<SignalDefinition | null>(null);
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
      type: formData.get('type') as 'technical' | 'ml' | 'custom',
      parameters: JSON.parse(formData.get('parameters') as string || '{}'),
      active: formData.get('active') === 'on',
    };

    createSignalMutation.mutate(signalData);
  };

  const handleUpdateSignal = (id: string, formData: FormData) => {
    const signalData = {
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      type: formData.get('type') as 'technical' | 'ml' | 'custom',
      parameters: JSON.parse(formData.get('parameters') as string || '{}'),
      active: formData.get('active') === 'on',
    };

    updateSignalMutation.mutate({ id, signal: signalData });
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'technical':
        return 'bg-blue-100 text-blue-800';
      case 'ml':
        return 'bg-purple-100 text-purple-800';
      case 'custom':
        return 'bg-green-100 text-green-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  if (isLoading) {
    return (
      <Layout>
        <div className="text-center py-12">Loading signals...</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h1 className="text-2xl font-semibold text-gray-900">Signal Definitions</h1>
          <button
            onClick={() => setShowCreateForm(true)}
            className="btn-primary"
          >
            New Signal
          </button>
        </div>

        {/* Available Signal Templates */}
        <div className="card">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Available Signal Templates</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {availableSignals?.map((signal) => (
              <div key={signal.name} className="border rounded-lg p-4">
                <h3 className="font-medium text-gray-900">{signal.name}</h3>
                <p className="text-sm text-gray-600 mt-1">{signal.description}</p>
                <button
                  onClick={() => {
                    setEditingSignal({
                      name: signal.name,
                      description: signal.description,
                      type: 'technical',
                      parameters: signal.parameters,
                      active: true,
                    });
                  }}
                  className="mt-2 text-sm text-primary-600 hover:text-primary-900"
                >
                  Create from template
                </button>
              </div>
            ))}
          </div>
        </div>

        {/* Custom Signals */}
        <div className="card">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Custom Signals</h2>
          
          {signals && signals.length === 0 ? (
            <p className="text-gray-500 text-center py-8">
              No custom signals defined. Create one to get started.
            </p>
          ) : (
            <div className="space-y-4">
              {signals?.map((signal) => (
                <div key={signal.id} className="border rounded-lg p-4 flex justify-between items-start">
                  <div className="flex-1">
                    <div className="flex items-center space-x-3">
                      <h3 className="font-medium text-gray-900">{signal.name}</h3>
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getTypeColor(signal.type)}`}>
                        {signal.type}
                      </span>
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${signal.active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                        {signal.active ? 'Active' : 'Inactive'}
                      </span>
                    </div>
                    <p className="text-sm text-gray-600 mt-1">{signal.description}</p>
                    {Object.keys(signal.parameters).length > 0 && (
                      <div className="mt-2">
                        <span className="text-xs text-gray-500">Parameters: </span>
                        <code className="text-xs bg-gray-100 px-1 rounded">
                          {JSON.stringify(signal.parameters)}
                        </code>
                      </div>
                    )}
                  </div>
                  <div className="flex space-x-2 ml-4">
                    <button
                      onClick={() => setEditingSignal(signal)}
                      className="text-sm text-primary-600 hover:text-primary-900"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Are you sure you want to delete this signal?')) {
                          deleteSignalMutation.mutate(signal.id!);
                        }
                      }}
                      className="text-sm text-red-600 hover:text-red-900"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Create/Edit Form Modal */}
        {(showCreateForm || editingSignal) && (
          <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">
                {editingSignal ? 'Edit Signal' : 'Create Signal'}
              </h3>
              
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
                className="space-y-4"
              >
                <div>
                  <label className="block text-sm font-medium text-gray-700">Name</label>
                  <input
                    type="text"
                    name="name"
                    defaultValue={editingSignal?.name || ''}
                    required
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700">Description</label>
                  <textarea
                    name="description"
                    defaultValue={editingSignal?.description || ''}
                    rows={3}
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700">Type</label>
                  <select
                    name="type"
                    defaultValue={editingSignal?.type || 'technical'}
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                  >
                    <option value="technical">Technical</option>
                    <option value="ml">Machine Learning</option>
                    <option value="custom">Custom</option>
                  </select>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700">Parameters (JSON)</label>
                  <textarea
                    name="parameters"
                    defaultValue={JSON.stringify(editingSignal?.parameters || {}, null, 2)}
                    rows={4}
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 font-mono text-sm"
                  />
                </div>
                
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    name="active"
                    defaultChecked={editingSignal?.active !== false}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                  />
                  <label className="ml-2 block text-sm text-gray-900">Active</label>
                </div>
                
                <div className="flex justify-end space-x-3 pt-4">
                  <button
                    type="button"
                    onClick={() => {
                      setShowCreateForm(false);
                      setEditingSignal(null);
                    }}
                    className="btn-secondary"
                  >
                    Cancel
                  </button>
                  <button type="submit" className="btn-primary">
                    {editingSignal ? 'Update' : 'Create'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}