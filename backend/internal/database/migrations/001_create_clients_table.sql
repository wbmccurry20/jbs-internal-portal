-- Migration: Create clients table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    contact_name VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster lookups
CREATE INDEX idx_clients_name ON clients(name);
CREATE INDEX idx_clients_active ON clients(active);

-- Insert common clients from Smartsheet data
INSERT INTO clients (name, active) VALUES
    ('Auto Zone', true),
    ('Driven Brands', true),
    ('Mister Car Wash', true),
    ('Valvoline', true),
    ('Grease Monkey', true),
    ('Take 5', true),
    ('Franchisee', true),
    ('Misc', true)
ON CONFLICT (name) DO NOTHING;
