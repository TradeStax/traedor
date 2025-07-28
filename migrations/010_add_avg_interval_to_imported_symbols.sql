-- Add average interval to imported_symbols table for faster lookups
ALTER TABLE imported_symbols 
ADD COLUMN IF NOT EXISTS avg_interval_seconds INTEGER DEFAULT 30;

-- Update existing records with calculated average interval
UPDATE imported_symbols
SET avg_interval_seconds = subquery.avg_seconds
FROM (
    SELECT 
        symbol,
        COALESCE(
            EXTRACT(EPOCH FROM (
                MAX(time) - MIN(time)
            )) / NULLIF(COUNT(*) - 1, 0),
            30
        )::INTEGER as avg_seconds
    FROM (
        SELECT symbol, time 
        FROM ohlc_data 
        WHERE symbol IN (SELECT symbol FROM imported_symbols)
        ORDER BY symbol, time
        LIMIT 10000  -- Sample for performance
    ) sampled
    GROUP BY symbol
) subquery
WHERE imported_symbols.symbol = subquery.symbol;

-- Update the trigger function to maintain avg_interval_seconds
CREATE OR REPLACE FUNCTION update_imported_symbols_stats()
RETURNS TRIGGER AS $$
DECLARE
    current_avg_interval INTEGER;
    new_avg_interval INTEGER;
BEGIN
    -- Get current average interval if record exists
    SELECT avg_interval_seconds INTO current_avg_interval
    FROM imported_symbols
    WHERE symbol = NEW.symbol;

    IF current_avg_interval IS NULL THEN
        -- First record, default to 30 seconds
        current_avg_interval := 30;
    END IF;

    INSERT INTO imported_symbols (symbol, total_records, earliest_data, latest_data, avg_interval_seconds)
    VALUES (NEW.symbol, 1, NEW.time, NEW.time, current_avg_interval)
    ON CONFLICT (symbol) DO UPDATE SET
        total_records = imported_symbols.total_records + 1,
        earliest_data = LEAST(imported_symbols.earliest_data, NEW.time),
        latest_data = GREATEST(imported_symbols.latest_data, NEW.time),
        last_import_date = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;