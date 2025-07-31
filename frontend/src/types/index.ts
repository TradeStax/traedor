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

// Signal Optimization Types
export interface OptimizationConfig {
  name: string;
  description: string;
  base_run_config: RunConfig;
  parameter_ranges: OptimizationParameterRange[];
  random_order: boolean;
  optimization_metric: string; // "cumulative_return", "sharpe_ratio", etc.
}

export interface OptimizationParameterRange {
  parameter_path: string;  // e.g., "signals_with_params.0.parameters.period"
  lower_bound: any;
  upper_bound: any;
  step: any;
  parameter_type: string;  // "int", "float", "string"
}

export interface Optimization {
  id: string;
  config: OptimizationConfig;
  status: OptimizationStatus;
  status_message: string;
  progress: number;  // 0.0 to 100.0
  total_permutations: number;
  completed_runs: number;
  failed_runs: number;
  started_at: string;
  completed_at?: string;
  results?: OptimizationResults;
  created_at: string;
  updated_at: string;
  worker_id?: string;
  parameter_sequence?: Record<string, any>[];
}

export type OptimizationStatus = 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled' | 'paused';

export interface OptimizationResults {
  best_result?: OptimizationRunResult;
  worst_result?: OptimizationRunResult;
  average_return: number;
  median_return: number;
  best_parameters: Record<string, any>;
  completion_time: number;
  total_backtests: number;
  successful_backtests: number;
  failed_backtests: number;
}

export interface OptimizationRun {
  id: string;
  optimization_id: string;
  parameter_index: number;
  parameters: Record<string, any>;
  run_config: RunConfig;
  backtest_run_id: string;
  status: RunStatus;
  created_at: string;
  updated_at: string;
}

export interface OptimizationRunResult {
  optimization_run_id: string;
  parameter_index: number;
  parameters: Record<string, any>;
  backtest_run_id: string;
  performance_metrics?: PerformanceMetrics;
  optimization_score: number;
  rank: number;
  completed_at: string;
}