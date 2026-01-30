-- Migration: Add archived column to jobs table
-- Created: 2026-01-30

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS archived BOOLEAN DEFAULT false;

-- Add index for archived column
CREATE INDEX IF NOT EXISTS idx_jobs_archived ON jobs(archived);

-- Update default archived filter queries
COMMENT ON COLUMN jobs.archived IS 'Soft delete flag - archived jobs are hidden by default';
