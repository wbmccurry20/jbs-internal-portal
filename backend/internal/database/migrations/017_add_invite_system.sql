-- Add status column to users table for invite flow
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- Invite tokens table
CREATE TABLE IF NOT EXISTS invite_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT invite_tokens_expiry_future CHECK (expires_at > created_at)
);

-- Index for fast token lookup
CREATE INDEX IF NOT EXISTS idx_invite_tokens_token ON invite_tokens(token);
CREATE INDEX IF NOT EXISTS idx_invite_tokens_user_id ON invite_tokens(user_id);

-- Set all existing users to active (they already have passwords)
UPDATE users SET status = 'active' WHERE status IS NULL OR status = '';
