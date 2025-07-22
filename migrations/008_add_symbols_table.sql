-- Create symbols configuration table
CREATE TABLE IF NOT EXISTS symbols (
    name VARCHAR(50) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    margin DECIMAL(18,2) NOT NULL,
    point_price DECIMAL(18,8) NOT NULL,
    tick_size DECIMAL(18,8) NOT NULL DEFAULT 0.25,
    contract_size INTEGER NOT NULL DEFAULT 1,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    exchange VARCHAR(50) NOT NULL,
    trading_hours JSONB, -- Store trading hours as JSON
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for active symbols
CREATE INDEX idx_symbols_active ON symbols(active);
CREATE INDEX idx_symbols_exchange ON symbols(exchange);

-- Insert default symbol configurations
INSERT INTO symbols (name, description, margin, point_price, tick_size, exchange) VALUES
    ('/MES', 'Micro E-mini S&P 500', 1700.00, 5.00, 0.25, 'CME'),
    ('/MNQ', 'Micro E-mini NASDAQ-100', 2000.00, 2.00, 0.25, 'CME'),
    ('/MYM', 'Micro E-mini Dow', 1000.00, 0.50, 1.00, 'CME'),
    ('/M2K', 'Micro E-mini Russell 2000', 1100.00, 5.00, 0.10, 'CME'),
    ('/ES', 'E-mini S&P 500', 17000.00, 50.00, 0.25, 'CME'),
    ('/NQ', 'E-mini NASDAQ-100', 20000.00, 20.00, 0.25, 'CME'),
    ('/YM', 'E-mini Dow', 10000.00, 5.00, 1.00, 'CME'),
    ('/RTY', 'E-mini Russell 2000', 11000.00, 50.00, 0.10, 'CME')
ON CONFLICT (name) DO NOTHING;

-- Add trigger for updated_at
CREATE TRIGGER update_symbols_updated_at BEFORE UPDATE ON symbols
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create supported timeframes table
CREATE TABLE IF NOT EXISTS timeframes (
    value VARCHAR(10) PRIMARY KEY,
    description VARCHAR(50) NOT NULL,
    interval_seconds INTEGER NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert default timeframes
INSERT INTO timeframes (value, description, interval_seconds) VALUES
    ('1m', '1 minute', 60),
    ('5m', '5 minutes', 300),
    ('15m', '15 minutes', 900),
    ('30m', '30 minutes', 1800),
    ('1h', '1 hour', 3600),
    ('4h', '4 hours', 14400),
    ('1d', '1 day', 86400)
ON CONFLICT (value) DO NOTHING;

-- Create index for active timeframes
CREATE INDEX idx_timeframes_active ON timeframes(active);

-- Create a view to get available data ranges per symbol and timeframe
CREATE OR REPLACE VIEW symbol_data_availability AS
SELECT 
    symbol,
    EXTRACT(EPOCH FROM (MAX(time) - MIN(time)) / COUNT(DISTINCT DATE_TRUNC('day', time)))::INTEGER as avg_interval_seconds,
    MIN(time) as earliest_data,
    MAX(time) as latest_data,
    COUNT(*) as total_records
FROM ohlc_data
GROUP BY symbol;