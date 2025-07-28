-- Market data files tracking
CREATE TABLE IF NOT EXISTS market_data_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64), -- SHA256 hash to detect duplicates
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
    status_message TEXT,
    row_count INTEGER,
    imported_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_market_data_files_status ON market_data_files(status);
CREATE INDEX idx_market_data_files_filename ON market_data_files(filename);
CREATE INDEX idx_market_data_files_file_hash ON market_data_files(file_hash);

-- OHLC data table (for bar data like 30-min, 1-hour, etc)
CREATE TABLE IF NOT EXISTS ohlc_data (
    id BIGSERIAL,
    file_id UUID REFERENCES market_data_files(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    time TIMESTAMP NOT NULL,
    open DECIMAL(18,8) NOT NULL,
    high DECIMAL(18,8) NOT NULL,
    low DECIMAL(18,8) NOT NULL,
    close DECIMAL(18,8) NOT NULL,
    volume BIGINT,
    trade_count INTEGER,
    ohlc_avg DECIMAL(18,8),
    hlc_avg DECIMAL(18,8),
    hl_avg DECIMAL(18,8),
    bid_volume BIGINT,
    ask_volume BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (symbol, time)
) PARTITION BY RANGE (time);

-- Create partitions for OHLC data
CREATE TABLE IF NOT EXISTS ohlc_data_2022_12 PARTITION OF ohlc_data
    FOR VALUES FROM ('2022-12-01') TO ('2023-01-01');

CREATE TABLE IF NOT EXISTS ohlc_data_2023_01 PARTITION OF ohlc_data
    FOR VALUES FROM ('2023-01-01') TO ('2023-02-01');

CREATE TABLE IF NOT EXISTS ohlc_data_2023_02 PARTITION OF ohlc_data
    FOR VALUES FROM ('2023-02-01') TO ('2023-03-01');

CREATE TABLE IF NOT EXISTS ohlc_data_2023_03 PARTITION OF ohlc_data
    FOR VALUES FROM ('2023-03-01') TO ('2023-04-01');

CREATE TABLE IF NOT EXISTS ohlc_data_2023_04 PARTITION OF ohlc_data
    FOR VALUES FROM ('2023-04-01') TO ('2023-05-01');

-- Create indexes
CREATE INDEX idx_ohlc_data_file_id ON ohlc_data(file_id);
CREATE INDEX idx_ohlc_data_symbol_time ON ohlc_data(symbol, time);

-- Technical indicators data (flexible schema for various indicators)
CREATE TABLE IF NOT EXISTS technical_indicators (
    id BIGSERIAL,
    file_id UUID REFERENCES market_data_files(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    time TIMESTAMP NOT NULL,
    indicator_name VARCHAR(100) NOT NULL,
    indicator_values JSONB NOT NULL, -- Store all indicator values as JSON
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (symbol, time, indicator_name)
) PARTITION BY RANGE (time);

-- Create partitions for technical indicators
CREATE TABLE IF NOT EXISTS technical_indicators_2022_12 PARTITION OF technical_indicators
    FOR VALUES FROM ('2022-12-01') TO ('2023-01-01');

CREATE TABLE IF NOT EXISTS technical_indicators_2023_01 PARTITION OF technical_indicators
    FOR VALUES FROM ('2023-01-01') TO ('2023-02-01');

CREATE TABLE IF NOT EXISTS technical_indicators_2023_02 PARTITION OF technical_indicators
    FOR VALUES FROM ('2023-02-01') TO ('2023-03-01');

CREATE TABLE IF NOT EXISTS technical_indicators_2023_03 PARTITION OF technical_indicators
    FOR VALUES FROM ('2023-03-01') TO ('2023-04-01');

CREATE TABLE IF NOT EXISTS technical_indicators_2023_04 PARTITION OF technical_indicators
    FOR VALUES FROM ('2023-04-01') TO ('2023-05-01');

-- Create indexes
CREATE INDEX idx_technical_indicators_file_id ON technical_indicators(file_id);
CREATE INDEX idx_technical_indicators_symbol_time ON technical_indicators(symbol, time);
CREATE INDEX idx_technical_indicators_name ON technical_indicators(indicator_name);

-- Update trigger for market_data_files
CREATE TRIGGER update_market_data_files_updated_at BEFORE UPDATE ON market_data_files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically create monthly partitions
CREATE OR REPLACE FUNCTION create_monthly_partitions(table_name text, start_date date, end_date date)
RETURNS void AS $$
DECLARE
    curr_date date := start_date;
    partition_name text;
    next_month date;
BEGIN
    WHILE curr_date < end_date LOOP
        next_month := curr_date + interval '1 month';
        partition_name := table_name || '_' || to_char(curr_date, 'YYYY_MM');
        
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
            partition_name, table_name, curr_date, next_month);
        
        curr_date := next_month;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create partitions for the next 12 months
SELECT create_monthly_partitions('ohlc_data', '2023-05-01'::date, '2024-05-01'::date);
SELECT create_monthly_partitions('technical_indicators', '2023-05-01'::date, '2024-05-01'::date);