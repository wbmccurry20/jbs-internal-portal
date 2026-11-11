CREATE TABLE IF NOT EXISTS license_expiration_alerts (
    license_id INTEGER NOT NULL REFERENCES state_licenses(id) ON DELETE CASCADE,
    expiration_date DATE NOT NULL,
    threshold_days INTEGER NOT NULL CHECK (threshold_days IN (30, 60)),
    sent_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (license_id, expiration_date, threshold_days)
);
