-- Migration: Create payment_applications table
-- Created: 2026-05

CREATE TABLE IF NOT EXISTS payment_applications (
    -- Identity
    id                          SERIAL PRIMARY KEY,
    tenant_id                   INTEGER NOT NULL REFERENCES pa_tenants(id),
    submission_token            VARCHAR(64) UNIQUE NOT NULL,    -- server-generated secure random token; no-auth lookup

    -- Step 1: Subcontractor contact
    company_name                VARCHAR(255) NOT NULL,
    contact_name                VARCHAR(255) NOT NULL,
    email                       VARCHAR(255) NOT NULL,
    phone                       VARCHAR(30),
    address_line1               VARCHAR(255),
    address_line2               VARCHAR(255),
    city                        VARCHAR(100),
    state                       VARCHAR(2),
    zip                         VARCHAR(10),

    -- Step 2: Project info
    project_name                VARCHAR(255) NOT NULL,
    project_number              VARCHAR(100),
    owner                       VARCHAR(255),
    contractor                  VARCHAR(255),
    contract_date               DATE,
    application_number          INTEGER NOT NULL DEFAULT 1,
    period_to                   DATE,

    -- Step 3: Contract summary inputs
    original_contract_sum       DECIMAL(14,2) NOT NULL DEFAULT 0,
    retainage_percent           DECIMAL(5,2)  NOT NULL DEFAULT 10,
    previous_certificates       DECIMAL(14,2) NOT NULL DEFAULT 0,
    additional_notes            TEXT,

    -- Calculated totals stored at submission time (for PDF stability and admin queries)
    calc_net_change_orders      DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_contract_sum_to_date   DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_total_completed_stored DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_retainage_amount       DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_earned_less_retainage  DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_current_payment_due    DECIMAL(14,2) NOT NULL DEFAULT 0,
    calc_balance_to_finish      DECIMAL(14,2) NOT NULL DEFAULT 0,

    -- Full raw payload snapshot (audit trail; ensures PDF reflects data as-submitted)
    submission_snapshot         JSONB,

    -- Stripe payment (placeholder; not active in v1)
    payment_status              VARCHAR(30) NOT NULL DEFAULT 'pending',
    stripe_checkout_session_id  VARCHAR(255),
    stripe_payment_intent_id    VARCHAR(255),
    payment_amount_cents        INTEGER NOT NULL DEFAULT 999,   -- $9.99
    paid_at                     TIMESTAMP,

    -- PDF generation (placeholder; not active in v1)
    pdf_status                  VARCHAR(30) NOT NULL DEFAULT 'not_generated',
    pdf_storage_key             VARCHAR(500),                   -- R2/S3 object key
    pdf_generated_at            TIMESTAMP,
    pdf_download_token          VARCHAR(64),                    -- short-lived secure download token
    pdf_download_expires_at     TIMESTAMP,

    -- Subcontractor email delivery (placeholder; not active in v1)
    email_status                VARCHAR(30) NOT NULL DEFAULT 'not_sent',
    email_sent_at               TIMESTAMP,
    email_resend_message_id     VARCHAR(255),

    -- AP copy delivery (placeholder; not active in v1)
    ap_email_status             VARCHAR(30) NOT NULL DEFAULT 'not_sent',
    ap_email_sent_at            TIMESTAMP,
    ap_email_resend_message_id  VARCHAR(255),

    -- Admin review (placeholder; not active in v1)
    review_status               VARCHAR(30) NOT NULL DEFAULT 'unreviewed',
    reviewed_by                 INTEGER REFERENCES users(id),   -- nullable; set when auth is added
    reviewed_at                 TIMESTAMP,
    review_notes                TEXT,

    -- Audit / error tracking
    error_log                   JSONB NOT NULL DEFAULT '[]',    -- array of {timestamp, step, message}
    ip_address                  VARCHAR(45),
    user_agent                  TEXT,

    created_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Status check constraints
ALTER TABLE payment_applications
    ADD CONSTRAINT check_pa_payment_status
        CHECK (payment_status IN ('pending', 'paid', 'failed', 'refunded')),
    ADD CONSTRAINT check_pa_pdf_status
        CHECK (pdf_status IN ('not_generated', 'generating', 'generated', 'failed')),
    ADD CONSTRAINT check_pa_email_status
        CHECK (email_status IN ('not_sent', 'sent', 'failed', 'bounced')),
    ADD CONSTRAINT check_pa_ap_email_status
        CHECK (ap_email_status IN ('not_sent', 'sent', 'failed', 'bounced')),
    ADD CONSTRAINT check_pa_review_status
        CHECK (review_status IN ('unreviewed', 'reviewed', 'flagged', 'archived')),
    ADD CONSTRAINT check_pa_retainage_percent
        CHECK (retainage_percent >= 0 AND retainage_percent <= 50),
    ADD CONSTRAINT check_pa_application_number
        CHECK (application_number >= 1),
    ADD CONSTRAINT check_pa_payment_amount
        CHECK (payment_amount_cents > 0);

CREATE INDEX IF NOT EXISTS idx_pa_tenant_id        ON payment_applications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pa_submission_token ON payment_applications(submission_token);
CREATE INDEX IF NOT EXISTS idx_pa_email            ON payment_applications(email);
CREATE INDEX IF NOT EXISTS idx_pa_payment_status   ON payment_applications(payment_status);
CREATE INDEX IF NOT EXISTS idx_pa_review_status    ON payment_applications(review_status);
CREATE INDEX IF NOT EXISTS idx_pa_stripe_session   ON payment_applications(stripe_checkout_session_id);
CREATE INDEX IF NOT EXISTS idx_pa_created_at       ON payment_applications(created_at DESC);

CREATE TRIGGER update_payment_applications_updated_at
    BEFORE UPDATE ON payment_applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
