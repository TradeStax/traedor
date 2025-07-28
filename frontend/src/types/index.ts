export interface Run {
  id: string;
  config: RunConfig;
  status: RunStatus;
  status_message: string;
  progress: number; // 0.0 to 100.0
  started_at: string;
  completed_at?: string;
  performance_metrics?: PerformanceMetrics;
  created_at: string;
  updated_at: string;
  worker_id?: string;
  retry_count?: number;
  last_error?: string;
}

export type RunStatus = 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled' | 'retrying';

export interface RunConfig {
  symbol: string;
  timeframe: string;
  start_time: string;
  end_time: string;
  datafeeds: DatafeedConfig[];
  broker: BrokerConfig;
  strategies: StrategyConfig[];
  signals?: string[];
  signals_with_params?: SignalWithParams[];
  signal_definitions?: Partial<SignalDefinition>[];
}

export interface SignalWithParams {
  signal_definition_id: string;
  parameters: Record<string, any>;
}

export interface DatafeedConfig {
  type: string;
  symbol: string;
  data_path: string;
  interval: string;
}

export interface BrokerConfig {
  type: string;
  starting_balance: number;
  weekly_withdrawl: number;
  trailing_stop_amount: number;
  profit_target: number;
  fee_per_side: number;
  open_slippage: number;
  symbol: SymbolConfig;
}

export interface SymbolConfig {
  name: string;
  margin: number;
  point_price: number;
}

export interface StrategyConfig {
  type: string;
  symbol: string;
  params: Record<string, any>;
}

export interface PerformanceMetrics {
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  total_profit: number;
  max_drawdown: number;
  max_drawdown_percent: number;
  sharpe_ratio: number;
  win_rate: number;
  average_win: number;
  average_loss: number;
  profit_factor: number;
  final_balance: number;
  return_percentage: number;
  average_mfe: number;
  average_mfe_percent: number;
  average_mae: number;
  average_mae_percent: number;
  balance_history: BalancePoint[];
  drawdown_history: DrawdownPoint[];
}

export interface BalancePoint {
  time: string;
  balance: number;
}

export interface DrawdownPoint {
  time: string;
  drawdown: number;
  drawdown_percent: number;
}

export interface Trade {
  id: string;
  symbol: string;
  operation: 'Buy' | 'Sell' | 'Close';
  quantity: number;
  open_price: number;
  close_price?: number;
  open_time: string;
  close_time?: string;
  net_profit?: number;
  max_profit?: number;
  max_drawdown?: number;
  mfe?: number;
  mfe_percent?: number;
  mae?: number;
  mae_percent?: number;
}

export interface Signal {
  id: string;
  run_id: string;
  signal_id?: string;
  symbol: string;
  direction: number;
  price: number;
  time: string;
  created_at: string;
}

export interface SignalDefinition {
  id?: string;
  name: string;
  description: string;
  type: string;
  parameters: Record<string, any>;
  active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface SignalOption {
  name: string;
  description: string;
  type: string;
  parameters: Record<string, any>;
  aggregation_intervals?: number[];
}