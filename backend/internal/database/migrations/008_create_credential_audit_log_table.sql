-- Migration: Create credential_audit_log table
-- Created: 2026-01-29

CREATE TABLE IF NOT EXISTS credential_audit_log (
    id SERIAL PRIMARY KEY,
    credential_id INTEGER NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_credential_audit_log_credential_id ON credential_audit_log(credential_id);
CREATE INDEX idx_credential_audit_log_user_id ON credential_audit_log(user_id);
CREATE INDEX idx_credential_audit_log_created_at ON credential_audit_log(created_at DESC);

-- Add check constraint for action
ALTER TABLE credential_audit_log ADD CONSTRAINT check_audit_action 
    CHECK (action IN ('create', 'view', 'update', 'delete', 'share'));

-- Add comment
COMMENT ON TABLE credential_audit_log IS 'Audit trail for password manager access';
