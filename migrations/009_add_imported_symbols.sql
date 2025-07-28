-- Create a table to efficiently track which symbols have imported data
CREATE TABLE IF NOT EXISTS imported_symbols (
    symbol VARCHAR(50) PRIMARY KEY,
    first_import_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_import_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    total_records BIGINT NOT NULL DEFAULT 0,
    earliest_data TIMESTAMP,
    latest_data TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for efficient queries
CREATE INDEX idx_imported_symbols_updated_at ON imported_symbols(updated_at);

-- Add trigger for updated_at
CREATE TRIGGER update_imported_symbols_updated_at BEFORE UPDATE ON imported_symbols
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Populate the table with existing data
INSERT INTO imported_symbols (symbol, total_records, earliest_data, latest_data)
SELECT 
    symbol,
    COUNT(*) as total_records,
    MIN(time) as earliest_data,
    MAX(time) as latest_data
FROM ohlc_data
GROUP BY symbol
ON CONFLICT (symbol) DO NOTHING;

-- Create function to update imported_symbols when new data is inserted
CREATE OR REPLACE FUNCTION update_imported_symbols_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO imported_symbols (symbol, total_records, earliest_data, latest_data)
    VALUES (NEW.symbol, 1, NEW.time, NEW.time)
    ON CONFLICT (symbol) DO UPDATE SET
        total_records = imported_symbols.total_records + 1,
        earliest_data = LEAST(imported_symbols.earliest_data, NEW.time),
        latest_data = GREATEST(imported_symbols.latest_data, NEW.time),
        last_import_date = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger on ohlc_data to maintain imported_symbols
CREATE TRIGGER update_imported_symbols_on_insert
    AFTER INSERT ON ohlc_data
    FOR EACH ROW
    EXECUTE FUNCTION update_imported_symbols_stats();