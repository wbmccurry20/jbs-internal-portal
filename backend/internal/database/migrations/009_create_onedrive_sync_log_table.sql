-- Migration: Create onedrive_sync_log table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS onedrive_sync_log (
    id SERIAL PRIMARY KEY,
    sync_type VARCHAR(50),
    records_processed INTEGER DEFAULT 0,
    records_created INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    errors TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- Create index
CREATE INDEX idx_onedrive_sync_log_started_at ON onedrive_sync_log(started_at DESC);

-- Add comment
COMMENT ON TABLE onedrive_sync_log IS 'Track OneDrive synchronization jobs for state licenses';
