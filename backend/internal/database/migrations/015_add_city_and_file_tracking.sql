-- Add city support and file tracking to licenses
ALTER TABLE state_licenses 
  ADD COLUMN IF NOT EXISTS city VARCHAR(100),
  ADD COLUMN IF NOT EXISTS is_city_license BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS notes TEXT;

-- Create table to track multiple files per license
CREATE TABLE IF NOT EXISTS license_files (
    id SERIAL PRIMARY KEY,
    license_id INTEGER REFERENCES state_licenses(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_type VARCHAR(50), -- 'license', 'renewal', 'application', 'supporting_doc'
    year_extracted INTEGER, -- Year found in filename
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for performance
CREATE INDEX IF NOT EXISTS idx_license_files_license_id ON license_files(license_id);
CREATE INDEX IF NOT EXISTS idx_state_licenses_city ON state_licenses(state, city);
CREATE INDEX IF NOT EXISTS idx_state_licenses_active ON state_licenses(is_active);
