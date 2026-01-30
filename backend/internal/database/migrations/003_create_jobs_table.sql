-- Migration: Create jobs table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY,
    job_number VARCHAR(50) UNIQUE NOT NULL,
    job_name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(2),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    client_id INTEGER REFERENCES clients(id),
    status VARCHAR(50),
    contract_value DECIMAL(12, 2),
    revised_contract_value DECIMAL(12, 2),
    start_date DATE,
    projected_end_date DATE,
    actual_end_date DATE,
    project_manager_id INTEGER REFERENCES users(id),
    apm_id INTEGER REFERENCES users(id),
    superintendent_id INTEGER REFERENCES superintendents(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for common queries
CREATE INDEX idx_jobs_job_number ON jobs(job_number);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_client_id ON jobs(client_id);
CREATE INDEX idx_jobs_superintendent_id ON jobs(superintendent_id);
CREATE INDEX idx_jobs_project_manager_id ON jobs(project_manager_id);
CREATE INDEX idx_jobs_start_date ON jobs(start_date);
CREATE INDEX idx_jobs_state ON jobs(state);

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
