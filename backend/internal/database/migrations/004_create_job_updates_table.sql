-- Migration: Create job_updates table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS job_updates (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    author_id INTEGER REFERENCES users(id),
    author_name VARCHAR(255),
    update_text TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_job_updates_job_id ON job_updates(job_id);
CREATE INDEX idx_job_updates_created_at ON job_updates(created_at DESC);

-- Add comment
COMMENT ON TABLE job_updates IS 'Meeting minutes, status updates, and notes for jobs';
