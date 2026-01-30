#!/bin/bash
# Import data to Railway production database
# Usage: ./import_to_production.sh <RAILWAY_DATABASE_URL>

set -e

if [ -z "$1" ]; then
    echo "❌ Error: Railway DATABASE_URL required"
    echo "Usage: ./import_to_production.sh 'postgresql://user:pass@host:port/dbname'"
    echo ""
    echo "Get your DATABASE_URL from:"
    echo "  Railway Dashboard → Postgres Service → Variables → DATABASE_URL"
    exit 1
fi

RAILWAY_DB="$1"
IMPORT_DIR="./production_export"

if [ ! -d "$IMPORT_DIR" ]; then
    echo "❌ Error: $IMPORT_DIR not found"
    echo "Run ./export_to_production.sh first"
    exit 1
fi

echo "⚠️  WARNING: This will import data into your production database"
echo "Database: $RAILWAY_DB"
echo ""
echo "Choose import mode:"
echo "1) CLEAR & IMPORT - Delete all existing data and import fresh (destructive)"
echo "2) UPSERT - Update existing records, insert new ones (safe)"
echo "3) CANCEL"
echo ""
read -p "Enter choice (1/2/3): " choice

if [ "$choice" = "3" ] || [ -z "$choice" ]; then
    echo "Aborted"
    exit 0
fi

if [ "$choice" != "1" ] && [ "$choice" != "2" ]; then
    echo "Invalid choice"
    exit 1
fi

echo ""

if [ "$choice" = "1" ]; then
    echo "⚠️  CLEARING existing data..."
    psql "$RAILWAY_DB" << 'EOF'
TRUNCATE TABLE superintendent_lodging, bids, jobs, superintendents, clients RESTART IDENTITY CASCADE;
EOF
    echo "✓ Tables cleared"
    echo ""
    echo "Importing data to production..."
    
    # Simple import for fresh database
    echo "→ Importing clients..."
    psql "$RAILWAY_DB" -c "\COPY clients FROM STDIN WITH CSV HEADER" < "$IMPORT_DIR/clients.csv"
    
    echo "→ Importing superintendents..."
    psql "$RAILWAY_DB" -c "\COPY superintendents FROM STDIN WITH CSV HEADER" < "$IMPORT_DIR/superintendents.csv"
    
    echo "→ Importing jobs..."
    psql "$RAILWAY_DB" -c "\COPY jobs FROM STDIN WITH CSV HEADER" < "$IMPORT_DIR/jobs.csv"
    
    echo "→ Importing bids..."
    psql "$RAILWAY_DB" -c "\COPY bids FROM STDIN WITH CSV HEADER" < "$IMPORT_DIR/bids.csv"
    
    echo "→ Importing lodging..."
    psql "$RAILWAY_DB" -c "\COPY superintendent_lodging FROM STDIN WITH CSV HEADER" < "$IMPORT_DIR/lodging.csv"
    
else
    echo "Importing data with UPSERT (updating existing, inserting new)..."
    
    # Upsert using a single psql session per table with a temp CSV import
    echo "→ Upserting clients..."
    {
        echo "BEGIN;"
        echo "CREATE TEMP TABLE temp_clients (LIKE clients INCLUDING ALL);"
        echo "\COPY temp_clients FROM STDIN WITH CSV HEADER"
        cat "$IMPORT_DIR/clients.csv"
        echo "\."
        cat << 'SQL'
INSERT INTO clients SELECT * FROM temp_clients
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    address = EXCLUDED.address,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    zip = EXCLUDED.zip,
    phone = EXCLUDED.phone,
    email = EXCLUDED.email,
    notes = EXCLUDED.notes,
    updated_at = CURRENT_TIMESTAMP;
COMMIT;
SQL
    } | psql "$RAILWAY_DB" -q
    
    echo "→ Upserting superintendents..."
    {
        echo "BEGIN;"
        echo "CREATE TEMP TABLE temp_supers (LIKE superintendents INCLUDING ALL);"
        echo "\COPY temp_supers FROM STDIN WITH CSV HEADER"
        cat "$IMPORT_DIR/superintendents.csv"
        echo "\."
        cat << 'SQL'
INSERT INTO superintendents SELECT * FROM temp_supers
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    phone = EXCLUDED.phone,
    email = EXCLUDED.email,
    notes = EXCLUDED.notes,
    updated_at = CURRENT_TIMESTAMP;
COMMIT;
SQL
    } | psql "$RAILWAY_DB" -q
    
    echo "→ Upserting jobs..."
    {
        echo "BEGIN;"
        echo "CREATE TEMP TABLE temp_jobs (LIKE jobs INCLUDING ALL);"
        echo "\COPY temp_jobs FROM STDIN WITH CSV HEADER"
        cat "$IMPORT_DIR/jobs.csv"
        echo "\."
        cat << 'SQL'
INSERT INTO jobs SELECT * FROM temp_jobs
ON CONFLICT (id) DO UPDATE SET
    client_id = EXCLUDED.client_id,
    job_number = EXCLUDED.job_number,
    name = EXCLUDED.name,
    address = EXCLUDED.address,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    zip = EXCLUDED.zip,
    superintendent_id = EXCLUDED.superintendent_id,
    status = EXCLUDED.status,
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date,
    notes = EXCLUDED.notes,
    updated_at = CURRENT_TIMESTAMP;
COMMIT;
SQL
    } | psql "$RAILWAY_DB" -q
    
    echo "→ Upserting bids..."
    {
        echo "BEGIN;"
        echo "CREATE TEMP TABLE temp_bids (LIKE bids INCLUDING ALL);"
        echo "\COPY temp_bids FROM STDIN WITH CSV HEADER"
        cat "$IMPORT_DIR/bids.csv"
        echo "\."
        cat << 'SQL'
INSERT INTO bids SELECT * FROM temp_bids
ON CONFLICT (id) DO UPDATE SET
    job_id = EXCLUDED.job_id,
    bid_number = EXCLUDED.bid_number,
    description = EXCLUDED.description,
    amount = EXCLUDED.amount,
    status = EXCLUDED.status,
    submitted_date = EXCLUDED.submitted_date,
    notes = EXCLUDED.notes,
    updated_at = CURRENT_TIMESTAMP;
COMMIT;
SQL
    } | psql "$RAILWAY_DB" -q
    
    echo "→ Upserting lodging..."
    {
        echo "BEGIN;"
        echo "CREATE TEMP TABLE temp_lodging (LIKE superintendent_lodging INCLUDING ALL);"
        echo "\COPY temp_lodging FROM STDIN WITH CSV HEADER"
        cat "$IMPORT_DIR/lodging.csv"
        echo "\."
        cat << 'SQL'
INSERT INTO superintendent_lodging SELECT * FROM temp_lodging
ON CONFLICT (id) DO UPDATE SET
    superintendent_id = EXCLUDED.superintendent_id,
    job_id = EXCLUDED.job_id,
    hotel_name = EXCLUDED.hotel_name,
    address = EXCLUDED.address,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    zip = EXCLUDED.zip,
    phone = EXCLUDED.phone,
    check_in_date = EXCLUDED.check_in_date,
    check_out_date = EXCLUDED.check_out_date,
    nightly_rate = EXCLUDED.nightly_rate,
    notes = EXCLUDED.notes,
    updated_at = CURRENT_TIMESTAMP;
COMMIT;
SQL
    } | psql "$RAILWAY_DB" -q
fi

echo ""
echo "→ Updating sequences..."
psql "$RAILWAY_DB" << 'EOF'
SELECT setval('clients_id_seq', (SELECT MAX(id) FROM clients));
SELECT setval('superintendents_id_seq', (SELECT MAX(id) FROM superintendents));
SELECT setval('jobs_id_seq', (SELECT MAX(id) FROM jobs));
SELECT setval('bids_id_seq', (SELECT MAX(id) FROM bids));
SELECT setval('superintendent_lodging_id_seq', (SELECT MAX(id) FROM superintendent_lodging));
EOF

echo ""
echo "✅ Import complete!"
echo ""
echo "Verifying counts:"
psql "$RAILWAY_DB" -c "SELECT 
    (SELECT COUNT(*) FROM clients) as clients,
    (SELECT COUNT(*) FROM superintendents) as superintendents,
    (SELECT COUNT(*) FROM jobs) as jobs,
    (SELECT COUNT(*) FROM bids) as bids,
    (SELECT COUNT(*) FROM superintendent_lodging) as lodging;"

echo ""
echo "Database size:"
psql "$RAILWAY_DB" -c "SELECT pg_size_pretty(pg_database_size(current_database())) as size;"
