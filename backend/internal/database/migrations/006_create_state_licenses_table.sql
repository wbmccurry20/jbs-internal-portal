-- Migration: Create state_licenses table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS state_licenses (
    id SERIAL PRIMARY KEY,
    state VARCHAR(2) NOT NULL,
    license_type VARCHAR(100) NOT NULL,
    license_number VARCHAR(100),
    entity_name VARCHAR(255),
    issue_date DATE,
    expiration_date DATE,
    status VARCHAR(50),
    renewal_fee DECIMAL(10, 2),
    notes TEXT,
    onedrive_file_path TEXT,
    last_synced_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_state_licenses_state ON state_licenses(state);
CREATE INDEX idx_state_licenses_status ON state_licenses(status);
CREATE INDEX idx_state_licenses_expiration_date ON state_licenses(expiration_date);
CREATE INDEX idx_state_licenses_license_type ON state_licenses(license_type);

-- Create updated_at trigger
CREATE TRIGGER update_state_licenses_updated_at BEFORE UPDATE ON state_licenses
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comment
COMMENT ON TABLE state_licenses IS 'State licensing data synced from OneDrive';
COMMENT ON COLUMN state_licenses.onedrive_file_path IS 'Original file path in OneDrive for reference';
