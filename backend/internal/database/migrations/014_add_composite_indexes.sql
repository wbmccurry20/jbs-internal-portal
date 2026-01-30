-- Migration: Add composite indexes for common query patterns
-- Created: 2026-01-30

-- Jobs table composite indexes
CREATE INDEX IF NOT EXISTS idx_jobs_status_archived ON jobs(status, archived);
CREATE INDEX IF NOT EXISTS idx_jobs_client_status ON jobs(client_id, status);
CREATE INDEX IF NOT EXISTS idx_jobs_superintendent_archived ON jobs(superintendent_id, archived);

-- Bids table composite indexes  
CREATE INDEX IF NOT EXISTS idx_bids_status_archived ON bids(status, archived);
CREATE INDEX IF NOT EXISTS idx_bids_client_status ON bids(client_id, status);
CREATE INDEX IF NOT EXISTS idx_bids_due_archived ON bids(due_date, archived) WHERE archived = false;

-- Lodging table composite indexes
CREATE INDEX IF NOT EXISTS idx_lodging_super_checkin ON superintendent_lodging(superintendent_id, check_in_date);
CREATE INDEX IF NOT EXISTS idx_lodging_job_checkin ON superintendent_lodging(job_id, check_in_date);

-- Comments
COMMENT ON INDEX idx_jobs_status_archived IS 'Speeds up filtered job lists (status + archived)';
COMMENT ON INDEX idx_bids_due_archived IS 'Partial index for active bids by due date';
