-- Create superintendent_lodging table for tracking AirBnb and temporary housing
CREATE TABLE superintendent_lodging (
    id SERIAL PRIMARY KEY,
    superintendent_id INTEGER REFERENCES superintendents(id) ON DELETE CASCADE,
    job_id INTEGER REFERENCES jobs(id) ON DELETE SET NULL,
    location VARCHAR(255),
    check_in_date DATE,
    check_out_date DATE,
    property_link TEXT,
    address VARCHAR(255),
    jobsite_address VARCHAR(255),
    cost_per_night DECIMAL(10, 2),
    total_cost DECIMAL(10, 2),
    booking_confirmation VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for common queries
CREATE INDEX idx_lodging_superintendent ON superintendent_lodging(superintendent_id);
CREATE INDEX idx_lodging_job ON superintendent_lodging(job_id);
CREATE INDEX idx_lodging_dates ON superintendent_lodging(check_in_date, check_out_date);
CREATE INDEX idx_lodging_check_in ON superintendent_lodging(check_in_date);
CREATE INDEX idx_lodging_check_out ON superintendent_lodging(check_out_date);

-- Create trigger for updated_at
CREATE TRIGGER update_superintendent_lodging_updated_at
    BEFORE UPDATE ON superintendent_lodging
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE superintendent_lodging IS 'Temporary housing (AirBnb) for superintendents on jobs';
COMMENT ON COLUMN superintendent_lodging.job_id IS 'Optional link to specific job';
COMMENT ON COLUMN superintendent_lodging.total_cost IS 'Auto-calculated or manually entered';
COMMENT ON COLUMN superintendent_lodging.property_link IS 'AirBnb or booking URL';
