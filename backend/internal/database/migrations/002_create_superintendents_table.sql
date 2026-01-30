-- Migration: Create superintendents table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS superintendents (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    certifications TEXT[],
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_superintendents_name ON superintendents(name);
CREATE INDEX idx_superintendents_active ON superintendents(active);

-- Insert known superintendents from Smartsheet data
INSERT INTO superintendents (name, active) VALUES
    ('Alan Tyminski', true),
    ('James Elliott', true),
    ('Martin Wade', true),
    ('Enrique McFarlane', true),
    ('Graham Dickson', true),
    ('Wolf Ishcomer', true),
    ('Cody Goatley', true),
    ('Jason Buchanan', true),
    ('Sam Clark', true),
    ('Wally Logan', true)
ON CONFLICT DO NOTHING;
