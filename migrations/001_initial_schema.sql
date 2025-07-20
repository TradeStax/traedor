-- Create database if not exists
-- CREATE DATABASE IF NOT EXISTS traedor;

-- Runs table
CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    performance_metrics JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_symbol ON runs((config->>'symbol'));
CREATE INDEX idx_runs_created_at ON runs(created_at);

-- Trades table
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    quantity INTEGER NOT NULL,
    open_price DECIMAL(18,8) NOT NULL,
    close_price DECIMAL(18,8),
    open_time TIMESTAMP NOT NULL,
    close_time TIMESTAMP,
    net_profit DECIMAL(18,8),
    max_profit DECIMAL(18,8),
    max_drawdown DECIMAL(18,8),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trades_run_id ON trades(run_id);
CREATE INDEX idx_trades_symbol ON trades(symbol);
CREATE INDEX idx_trades_open_time ON trades(open_time);

-- Signals table
CREATE TABLE IF NOT EXISTS signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    signal_definition_id UUID,
    symbol VARCHAR(50) NOT NULL,
    direction INTEGER NOT NULL, -- 0=None, 1=Close, 2=Buy, 3=Sell
    price DECIMAL(18,8) NOT NULL,
    time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_signals_run_id ON signals(run_id);
CREATE INDEX idx_signals_symbol ON signals(symbol);
CREATE INDEX idx_signals_time ON signals(time);
CREATE INDEX idx_signals_definition_id ON signals(signal_definition_id);

-- Signal definitions table
CREATE TABLE IF NOT EXISTS signal_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- technical, ml, custom
    parameters JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_signal_definitions_type ON signal_definitions(type);
CREATE INDEX idx_signal_definitions_active ON signal_definitions(active);

-- Tick data table (partitioned by month for performance)
CREATE TABLE IF NOT EXISTS tick_data (
    symbol VARCHAR(50) NOT NULL,
    time TIMESTAMP NOT NULL,
    price DECIMAL(18,8) NOT NULL,
    volume BIGINT,
    bid DECIMAL(18,8),
    ask DECIMAL(18,8),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (symbol, time)
) PARTITION BY RANGE (time);

-- Create initial partitions for tick data (adjust as needed)
CREATE TABLE IF NOT EXISTS tick_data_2023_01 PARTITION OF tick_data
    FOR VALUES FROM ('2023-01-01') TO ('2023-02-01');

CREATE TABLE IF NOT EXISTS tick_data_2023_02 PARTITION OF tick_data
    FOR VALUES FROM ('2023-02-01') TO ('2023-03-01');

CREATE TABLE IF NOT EXISTS tick_data_2023_03 PARTITION OF tick_data
    FOR VALUES FROM ('2023-03-01') TO ('2023-04-01');

-- Add more partitions as needed...

-- Create indexes on tick_data
CREATE INDEX idx_tick_data_symbol_time ON tick_data(symbol, time);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_runs_updated_at BEFORE UPDATE ON runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_signal_definitions_updated_at BEFORE UPDATE ON signal_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();