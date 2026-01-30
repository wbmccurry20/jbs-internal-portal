-- Migration: Create change_orders table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS change_orders (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    amount DECIMAL(12, 2),
    status VARCHAR(50) DEFAULT 'pending',
    created_by INTEGER REFERENCES users(id),
    approved_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_change_orders_job_id ON change_orders(job_id);
CREATE INDEX idx_change_orders_status ON change_orders(status);
CREATE INDEX idx_change_orders_created_at ON change_orders(created_at DESC);

-- Add check constraint for status
ALTER TABLE change_orders ADD CONSTRAINT check_change_order_status 
    CHECK (status IN ('pending', 'approved', 'rejected'));
