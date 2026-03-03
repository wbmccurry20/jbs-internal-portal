-- Add city support to licenses
ALTER TABLE state_licenses 
  ADD COLUMN IF NOT EXISTS city VARCHAR(100),
  ADD COLUMN IF NOT EXISTS is_city_license BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

-- Index for performance
CREATE INDEX IF NOT EXISTS idx_state_licenses_city ON state_licenses(state, city);
CREATE INDEX IF NOT EXISTS idx_state_licenses_active ON state_licenses(is_active);
