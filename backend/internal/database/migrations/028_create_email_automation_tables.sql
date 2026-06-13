-- Migration: Create email automation tables
-- Created: 2026-06

-- 1. Folder mapping cache: tracks Outlook folders synced from the shared mailbox via Graph API
CREATE TABLE IF NOT EXISTS email_folder_mappings (
    id                   SERIAL PRIMARY KEY,
    project_name         VARCHAR(255)  NOT NULL,
    folder_id            VARCHAR(255)  UNIQUE NOT NULL,  -- Outlook folder ID from Graph API
    folder_name          VARCHAR(255)  NOT NULL,          -- Outlook display name
    folder_path          VARCHAR(500),
    parent_folder_id     VARCHAR(255),
    parent_folder_name   VARCHAR(50),
    is_bid_submitted_sub BOOLEAN       DEFAULT FALSE,
    bid_id               INTEGER       REFERENCES bids(id) ON DELETE SET NULL,
    created_at           TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_folder_mappings_bid_id
    ON email_folder_mappings(bid_id);

-- Reuse the existing update_updated_at_column() function (defined in earlier migrations)
CREATE TRIGGER update_email_folder_mappings_updated_at
    BEFORE UPDATE ON email_folder_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Audit log: one row per message processed by the automation engine
CREATE TABLE IF NOT EXISTS email_automation_log (
    id                  SERIAL PRIMARY KEY,
    message_id          VARCHAR(500)  NOT NULL,
    subject             TEXT,
    sender              VARCHAR(255),
    action_taken        VARCHAR(50),
    destination_folder  VARCHAR(255),
    rule_matched        VARCHAR(100),
    error               TEXT,
    processed_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_automation_log_processed_at
    ON email_automation_log(processed_at);

CREATE INDEX IF NOT EXISTS idx_email_automation_log_action
    ON email_automation_log(action_taken);

-- 3. Key/value config store for email automation settings
CREATE TABLE IF NOT EXISTS email_automation_config (
    key         VARCHAR(100) PRIMARY KEY,
    value       TEXT,
    updated_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- Default configuration rows.
-- Credential and address values are intentionally left as empty strings;
-- they are populated at runtime via the UI or the OAuth flow — never hardcoded here.
INSERT INTO email_automation_config (key, value, updated_at) VALUES
    ('enabled',                  'false', NOW()),
    ('polling_interval_minutes', '15',    NOW()),
    ('mailbox_address',          '',      NOW()),  -- PLACEHOLDER: set via the portal settings UI
    ('last_processed_at',        '',      NOW()),
    ('oauth_access_token',       '',      NOW()),  -- PLACEHOLDER: populated by OAuth flow
    ('oauth_refresh_token',      '',      NOW()),  -- PLACEHOLDER: populated by OAuth flow
    ('oauth_expires_at',         '',      NOW())   -- PLACEHOLDER: populated by OAuth flow
ON CONFLICT (key) DO NOTHING;
