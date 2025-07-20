export interface Run {
  id: string;
  config: RunConfig;
  status: RunStatus;
  started_at: string;
  completed_at?: string;
  performance_metrics?: PerformanceMetrics;
  created_at: string;
  updated_at: string;
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface RunConfig {
  symbol: string;
  timeframe: string;
  start_time: string;
  end_time: string;
  datafeeds: DatafeedConfig[];
  broker: BrokerConfig;
  strategies: StrategyConfig[];
  signals: string[];
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
  sharpe_ratio: number;
  win_rate: number;
  average_win: number;
  average_loss: number;
  profit_factor: number;
  final_balance: number;
  return_percentage: number;
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
  type: 'technical' | 'ml' | 'custom';
  parameters: Record<string, any>;
  active: boolean;
  created_at?: string;
  updated_at?: string;
}