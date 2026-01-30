#!/bin/bash
# Export local data to production-ready SQL
# This creates INSERT statements that are safe to run in production

set -e

LOCAL_DB="postgresql://jbs_user:jbs_password@localhost:5433/jbs_portal?sslmode=disable"
OUTPUT_DIR="./production_export"

mkdir -p "$OUTPUT_DIR"

echo "Exporting data from local database..."

# Export clients
psql "$LOCAL_DB" -c "\COPY (SELECT * FROM clients ORDER BY id) TO STDOUT WITH CSV HEADER" > "$OUTPUT_DIR/clients.csv"
echo "✓ Exported clients"

# Export superintendents  
psql "$LOCAL_DB" -c "\COPY (SELECT * FROM superintendents ORDER BY id) TO STDOUT WITH CSV HEADER" > "$OUTPUT_DIR/superintendents.csv"
echo "✓ Exported superintendents"

# Export jobs
psql "$LOCAL_DB" -c "\COPY (SELECT * FROM jobs ORDER BY id) TO STDOUT WITH CSV HEADER" > "$OUTPUT_DIR/jobs.csv"
echo "✓ Exported jobs"

# Export bids
psql "$LOCAL_DB" -c "\COPY (SELECT * FROM bids ORDER BY id) TO STDOUT WITH CSV HEADER" > "$OUTPUT_DIR/bids.csv"
echo "✓ Exported bids"

# Export lodging
psql "$LOCAL_DB" -c "\COPY (SELECT * FROM superintendent_lodging ORDER BY id) TO STDOUT WITH CSV HEADER" > "$OUTPUT_DIR/lodging.csv"
echo "✓ Exported lodging"

echo ""
echo "✅ Export complete! Files saved to: $OUTPUT_DIR"
echo ""
echo "File sizes:"
ls -lh "$OUTPUT_DIR" | tail -n +2
echo ""
echo "Total size:"
du -sh "$OUTPUT_DIR"
echo ""
echo "Next steps:"
echo "1. Review CSV files in $OUTPUT_DIR"
echo "2. Run import_to_production.sh with your Railway DATABASE_URL"
