-- Migration: Create pa_tenants table (payment application white-label tenants)
-- Created: 2026-05

CREATE TABLE IF NOT EXISTS pa_tenants (
    id              SERIAL PRIMARY KEY,
    slug            VARCHAR(100) UNIQUE NOT NULL,       -- URL/API identifier, e.g. 'jbs'
    name            VARCHAR(255) NOT NULL,              -- Display name, e.g. 'JBS Construction'
    ap_email        VARCHAR(255),                       -- AP copy delivery address
    brand_config    JSONB NOT NULL DEFAULT '{}',        -- Logo URL, colors, PDF header (white-label)
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pa_tenants_slug   ON pa_tenants(slug);
CREATE INDEX IF NOT EXISTS idx_pa_tenants_active ON pa_tenants(active);

CREATE TRIGGER update_pa_tenants_updated_at
    BEFORE UPDATE ON pa_tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Seed JBS tenant
-- AP email confirmed by client (May 2026)
INSERT INTO pa_tenants (slug, name, ap_email, active)
VALUES ('jbs', 'JBS Construction', 'info@jbsconstructiongroup.com', true)
ON CONFLICT (slug) DO NOTHING;
