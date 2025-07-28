-- Modify OHLC table to support tick data with duplicate timestamps
-- Drop the existing primary key constraint
ALTER TABLE ohlc_data DROP CONSTRAINT IF EXISTS ohlc_data_pkey CASCADE;

-- Add a sequence number to differentiate ticks at the same timestamp
ALTER TABLE ohlc_data ADD COLUMN IF NOT EXISTS tick_sequence BIGINT DEFAULT 0;

-- Create new composite primary key including tick_sequence
-- For partitioned tables, we must include the partition key (time)
ALTER TABLE ohlc_data ADD PRIMARY KEY (symbol, time, tick_sequence);

-- Create index for efficient queries
CREATE INDEX IF NOT EXISTS idx_ohlc_id ON ohlc_data(id);

-- Drop the now-redundant index
DROP INDEX IF EXISTS idx_ohlc_data_symbol_time;