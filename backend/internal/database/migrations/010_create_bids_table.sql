-- Create bids table for tracking bid opportunities and pipeline
CREATE TABLE bids (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    location VARCHAR(255) NOT NULL,
    city VARCHAR(100),
    state VARCHAR(2),
    due_date DATE,
    assigned_to_id INTEGER REFERENCES users(id),
    assigned_to_name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'in_progress',
    building_connected_date DATE,
    plan_hub_date DATE,
    awarded VARCHAR(10),
    bid_amount DECIMAL(12, 2),
    notes TEXT,
    job_id INTEGER REFERENCES jobs(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    archived BOOLEAN DEFAULT FALSE
);

-- Create indexes for common queries
CREATE INDEX idx_bids_client ON bids(client_id);
CREATE INDEX idx_bids_status ON bids(status);
CREATE INDEX idx_bids_archived ON bids(archived);
CREATE INDEX idx_bids_awarded ON bids(awarded);
CREATE INDEX idx_bids_assigned_to ON bids(assigned_to_id);
CREATE INDEX idx_bids_due_date ON bids(due_date);
CREATE INDEX idx_bids_job ON bids(job_id);

-- Create trigger for updated_at
CREATE TRIGGER update_bids_updated_at
    BEFORE UPDATE ON bids
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE bids IS 'Bid opportunities and pipeline tracking';
COMMENT ON COLUMN bids.status IS 'in_progress, complete, no_access, did_not_bid, not_started, not_bidding';
COMMENT ON COLUMN bids.awarded IS 'Yes, No, or NULL (pending)';
COMMENT ON COLUMN bids.archived IS 'TRUE = moved to Completed Bids';
COMMENT ON COLUMN bids.job_id IS 'Link to job if bid was awarded';
