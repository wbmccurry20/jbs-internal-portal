-- Compensating migration: PA migrations were originally numbered 020-023 and
-- renamed to 024-027 to resolve a duplicate prefix conflict with
-- 020_create_license_documents_table.sql.
--
-- On databases where the old filenames already ran (tracked in schema_migrations
-- as 020_create_pa_tenants.sql etc.), this migration marks the renamed filenames
-- as applied so the runner skips them and avoids duplicate-trigger errors.
--
-- On a fresh database, pa_tenants does not yet exist, so nothing is inserted
-- and migrations 024-027 run normally.

INSERT INTO schema_migrations (version)
SELECT unnest(ARRAY[
    '024_create_pa_tenants.sql',
    '025_create_payment_applications.sql',
    '026_create_payment_application_change_orders.sql',
    '027_create_payment_application_line_items.sql'
])
WHERE EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'pa_tenants'
)
ON CONFLICT DO NOTHING;
