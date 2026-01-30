-- Migration: Add archived column to superintendent_lodging table
-- Created: 2026-01-30

ALTER TABLE superintendent_lodging ADD COLUMN IF NOT EXISTS archived BOOLEAN DEFAULT false;

-- Add index for archived column
CREATE INDEX IF NOT EXISTS idx_superintendent_lodging_archived ON superintendent_lodging(archived);

-- Update default archived filter queries
COMMENT ON COLUMN superintendent_lodging.archived IS 'Soft delete flag - archived lodging records are hidden by default';
