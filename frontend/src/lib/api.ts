import axios from 'axios';
import { Run, RunConfig, Trade, Signal, SignalDefinition } from '@/types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const runsApi = {
  list: async (params?: { symbol?: string; status?: string; search?: string; limit?: number; offset?: number }) => {
    const response = await api.get<Run[]>('/runs', { params });
    return response.data;
  },

  get: async (id: string) => {
    const response = await api.get<Run>(`/runs/${id}`);
    return response.data;
  },

  create: async (config: RunConfig) => {
    const response = await api.post<Run>('/runs', config);
    return response.data;
  },

  cancel: async (id: string) => {
    const response = await api.post(`/runs/${id}/cancel`);
    return response.data;
  },

  retry: async (id: string) => {
    const response = await api.post(`/runs/${id}/retry`);
    return response.data;
  },

  getTrades: async (runId: string) => {
    const response = await api.get<Trade[]>(`/runs/${runId}/trades`);
    return response.data;
  },

  getSignals: async (runId: string) => {
    const response = await api.get<Signal[]>(`/runs/${runId}/signals`);
    return response.data;
  },
};

export const signalsApi = {
  list: async () => {
    const response = await api.get<SignalDefinition[]>('/signals');
    return response.data;
  },

  create: async (signal: SignalDefinition) => {
    const response = await api.post<SignalDefinition>('/signals', signal);
    return response.data;
  },

  update: async (id: string, signal: SignalDefinition) => {
    const response = await api.put<SignalDefinition>(`/signals/${id}`, signal);
    return response.data;
  },

  delete: async (id: string) => {
    await api.delete(`/signals/${id}`);
  },

  getAvailable: async () => {
    const response = await api.get<any[]>('/signals/available');
    return response.data;
  },
};

export const configApi = {
  getSymbols: async () => {
    const response = await api.get<{ name: string; description: string }[]>('/config/symbols');
    return response.data;
  },

  getTimeframes: async () => {
    const response = await api.get<{ value: string; description: string }[]>('/config/timeframes');
    return response.data;
  },
};