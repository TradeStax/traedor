import axios from 'axios';
import { Run, RunConfig, Trade, Signal, SignalDefinition } from '@/types';

const API_BASE_URL = '/api';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const runsApi = {
  list: async (params?: { symbol?: string; status?: string; search?: string; limit?: number; offset?: number }) => {
    const response = await api.get<{
      runs: Run[];
      pagination: {
        total: number;
        limit: number;
        offset: number;
      };
    }>('/runs', { params });
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

  getTrades: async (runId: string, limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit) params.append('limit', limit.toString());
    if (offset) params.append('offset', offset.toString());
    
    const response = await api.get<{
      trades: any[]; // Raw backend trades format
      pagination: {
        total: number;
        limit: number;
        offset: number;
      };
    }>(`/runs/${runId}/trades?${params.toString()}`);
    return response.data;
  },

  getTradesStream: async (runId: string) => {
    const response = await api.get<{
      trades: any[]; // Raw backend trades format
      total: number;
    }>(`/runs/${runId}/trades/stream`);
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
    const response = await api.get<{
      name: string;
      description: string;
      margin: number;
      point_price: number;
      tick_size: number;
      contract_size: number;
      currency: string;
      exchange: string;
      active: boolean;
    }[]>('/config/symbols');
    return response.data;
  },

  getTimeframes: async () => {
    const response = await api.get<{
      value: string;
      description: string;
      interval_seconds: number;
      active: boolean;
    }[]>('/config/timeframes');
    return response.data;
  },

  getSymbolDataAvailability: async (symbol: string) => {
    const response = await api.get<{
      symbol: string;
      earliest_data: string;
      latest_data: string;
      total_records: number;
      avg_interval_seconds: number;
    }>('/config/symbols/availability', {
      params: { symbol }
    });
    return response.data;
  },
};

export const dataApi = {
  getFiles: async () => {
    const response = await api.get('/data/files');
    return response.data;
  },

  scanFiles: async () => {
    const response = await api.post('/data/scan');
    return response.data;
  },

  importFile: async (filePath: string) => {
    const response = await api.post('/data/import/new', { file_path: filePath });
    return response.data;
  },

  deleteFile: async (fileId: string) => {
    const response = await api.delete(`/data/files/${fileId}`);
    return response.data;
  },

  deleteFailedImports: async () => {
    const response = await api.delete('/data/files/failed');
    return response.data;
  },

  deletePendingImports: async () => {
    const response = await api.delete('/data/files/pending');
    return response.data;
  },

  retryFile: async (fileId: string) => {
    const response = await api.post(`/data/files/${fileId}/retry`);
    return response.data;
  },

  getOHLCData: async (symbol: string, start?: string, end?: string) => {
    const params = new URLSearchParams();
    params.append('symbol', symbol);
    if (start) params.append('start', start);
    if (end) params.append('end', end);
    
    const response = await api.get(`/data/ohlc?${params.toString()}`);
    return response.data;
  },
};

export const optimizationApi = {
  create: async (config: any) => {
    const response = await api.post('/optimizations', config);
    return response.data;
  },

  list: async (params?: { status?: string; limit?: number; offset?: number }) => {
    const response = await api.get('/optimizations', { params });
    return response.data;
  },

  get: async (id: string) => {
    const response = await api.get(`/optimizations/${id}`);
    return response.data;
  },

  cancel: async (id: string) => {
    const response = await api.post(`/optimizations/${id}/cancel`);
    return response.data;
  },

  pause: async (id: string) => {
    const response = await api.post(`/optimizations/${id}/pause`);
    return response.data;
  },

  resume: async (id: string) => {
    const response = await api.post(`/optimizations/${id}/resume`);
    return response.data;
  },

  getResults: async (id: string) => {
    const response = await api.get(`/optimizations/${id}/results`);
    return response.data;
  },
};