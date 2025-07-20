-- Add job tracking fields to runs table
ALTER TABLE runs ADD COLUMN IF NOT EXISTS status_message TEXT DEFAULT '';
ALTER TABLE runs ADD COLUMN IF NOT EXISTS progress DECIMAL(5,2) DEFAULT 0.0;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS worker_id VARCHAR(255) DEFAULT '';
ALTER TABLE runs ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS last_error TEXT DEFAULT '';

-- Add new status values (PostgreSQL doesn't have ALTER TYPE for enums, so we'd need to recreate)
-- For now, we'll assume the application handles the new status values as strings

-- Add index for job queue operations
CREATE INDEX IF NOT EXISTS idx_runs_status_created ON runs(status, created_at) WHERE status IN ('queued', 'pending');
CREATE INDEX IF NOT EXISTS idx_runs_worker_id ON runs(worker_id) WHERE worker_id != '';

-- Add index for run filtering
CREATE INDEX IF NOT EXISTS idx_runs_symbol ON runs((config->>'symbol'));
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);