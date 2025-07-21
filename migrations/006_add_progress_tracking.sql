-- Add detailed progress tracking to market_data_files table
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS progress_percentage INTEGER DEFAULT 0;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS lines_processed BIGINT DEFAULT 0;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS total_lines BIGINT DEFAULT 0;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS processing_start_time TIMESTAMP;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS estimated_completion_time TIMESTAMP;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS processing_rate DECIMAL(10,2); -- rows per second
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS current_batch INTEGER DEFAULT 0;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS total_batches INTEGER DEFAULT 0;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS last_processed_line_preview TEXT;
ALTER TABLE market_data_files ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0;

-- Add index for progress queries
CREATE INDEX IF NOT EXISTS idx_market_data_files_processing ON market_data_files(status, processing_start_time);