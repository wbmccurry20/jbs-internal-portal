-- Migration: Create payment_application_change_orders table
-- Created: 2026-05

CREATE TABLE IF NOT EXISTS payment_application_change_orders (
    id                      SERIAL PRIMARY KEY,
    payment_application_id  INTEGER NOT NULL REFERENCES payment_applications(id) ON DELETE CASCADE,
    co_number               VARCHAR(50),
    description             TEXT,
    amount                  DECIMAL(14,2) NOT NULL DEFAULT 0,
    date_approved           DATE,
    sort_order              INTEGER NOT NULL DEFAULT 0,        -- preserves user-entered display order
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pa_co_application_id ON payment_application_change_orders(payment_application_id);
