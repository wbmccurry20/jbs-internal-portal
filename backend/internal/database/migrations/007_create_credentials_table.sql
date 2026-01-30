-- Migration: Create credentials table (password manager)
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS credentials (
    id SERIAL PRIMARY KEY,
    category VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255),
    encrypted_password TEXT NOT NULL,
    url TEXT,
    notes TEXT,
    created_by INTEGER REFERENCES users(id),
    shared_with INTEGER[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_credentials_category ON credentials(category);
CREATE INDEX idx_credentials_name ON credentials(name);
CREATE INDEX idx_credentials_created_by ON credentials(created_by);

-- Create updated_at trigger
CREATE TRIGGER update_credentials_updated_at BEFORE UPDATE ON credentials
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comment
COMMENT ON TABLE credentials IS 'Encrypted password storage';
COMMENT ON COLUMN credentials.encrypted_password IS 'AES-256 encrypted password';
COMMENT ON COLUMN credentials.shared_with IS 'Array of user IDs who have access';
