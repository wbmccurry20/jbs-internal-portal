-- Migration: Create payment_application_line_items table (continuation sheet / Schedule of Values)
-- Created: 2026-05

CREATE TABLE IF NOT EXISTS payment_application_line_items (
    id                      SERIAL PRIMARY KEY,
    payment_application_id  INTEGER NOT NULL REFERENCES payment_applications(id) ON DELETE CASCADE,
    item_no                 VARCHAR(50),
    description             TEXT,

    -- User-entered values
    scheduled_value         DECIMAL(14,2) NOT NULL DEFAULT 0,
    prev_completed          DECIMAL(14,2) NOT NULL DEFAULT 0,
    this_period             DECIMAL(14,2) NOT NULL DEFAULT 0,
    materials_stored        DECIMAL(14,2) NOT NULL DEFAULT 0,

    -- Calculated values stored at submission time (for PDF stability)
    calc_total_completed    DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_percent_complete   DECIMAL(6,2)  NOT NULL DEFAULT 0,
    calc_balance_to_finish  DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_retainage          DECIMAL(14,2) NOT NULL DEFAULT 0,

    sort_order              INTEGER NOT NULL DEFAULT 0,        -- preserves user-entered display order
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pa_li_application_id ON payment_application_line_items(payment_application_id);
