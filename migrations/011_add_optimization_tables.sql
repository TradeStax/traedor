-- Add optimization support tables
CREATE TABLE IF NOT EXISTS optimizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    status_message TEXT,
    progress FLOAT DEFAULT 0.0,
    total_permutations INTEGER DEFAULT 0,
    completed_runs INTEGER DEFAULT 0,
    failed_runs INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    results JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    worker_id VARCHAR(255),
    parameter_sequence JSONB
);

CREATE INDEX IF NOT EXISTS idx_optimizations_status ON optimizations(status);
CREATE INDEX IF NOT EXISTS idx_optimizations_created_at ON optimizations(created_at);
CREATE INDEX IF NOT EXISTS idx_optimizations_worker_id ON optimizations(worker_id);

-- Table to track individual optimization runs
CREATE TABLE IF NOT EXISTS optimization_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    optimization_id UUID NOT NULL REFERENCES optimizations(id) ON DELETE CASCADE,
    parameter_index INTEGER NOT NULL,
    parameters JSONB NOT NULL,
    run_config JSONB NOT NULL,
    backtest_run_id UUID REFERENCES runs(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimization_runs_optimization_id ON optimization_runs(optimization_id);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_parameter_index ON optimization_runs(parameter_index);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_status ON optimization_runs(status);
CREATE INDEX IF NOT EXISTS idx_optimization_runs_backtest_run_id ON optimization_runs(backtest_run_id);

-- Table to store optimization results for fast querying and sorting
CREATE TABLE IF NOT EXISTS optimization_run_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    optimization_id UUID NOT NULL REFERENCES optimizations(id) ON DELETE CASCADE,
    optimization_run_id UUID NOT NULL REFERENCES optimization_runs(id) ON DELETE CASCADE,
    parameter_index INTEGER NOT NULL,
    parameters JSONB NOT NULL,
    backtest_run_id UUID NOT NULL REFERENCES runs(id),
    performance_metrics JSONB,
    optimization_score FLOAT,
    rank_position INTEGER,
    completed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimization_run_results_optimization_id ON optimization_run_results(optimization_id);
CREATE INDEX IF NOT EXISTS idx_optimization_run_results_optimization_score ON optimization_run_results(optimization_score DESC);
CREATE INDEX IF NOT EXISTS idx_optimization_run_results_rank ON optimization_run_results(rank_position);
CREATE INDEX IF NOT EXISTS idx_optimization_run_results_backtest_run_id ON optimization_run_results(backtest_run_id);