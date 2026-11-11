package services

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

func TestLicenseAlertDelivery(t *testing.T) {
	url := os.Getenv("TEST_LICENSE_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_LICENSE_DATABASE_URL is required for PostgreSQL integration test")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	lock, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec("SELECT pg_advisory_xact_lock(1245852499, 1)"); err != nil {
		t.Fatal(err)
	}
	other, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	sent, err := sendNextLicenseAlert(other, licenseAlertConfig{}, func(licenseAlertConfig, string) error {
		t.Fatal("second worker sent while lock was held")
		return nil
	})
	if err != nil || sent {
		t.Fatalf("second worker should skip while locked: sent=%v, error=%v", sent, err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TEMP TABLE state_licenses (
			id integer primary key, state text, city text, license_type text,
			license_number text, expiration_date date, status text, is_active boolean
		);
		CREATE TEMP TABLE license_expiration_alerts (
			license_id integer, expiration_date date, threshold_days integer,
			PRIMARY KEY (license_id, expiration_date, threshold_days)
		);
		INSERT INTO state_licenses
		SELECT n, 'AL', 'Hueytown', 'Business License', '123',
		       (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date + n,
		       CASE WHEN n = 15 THEN 'pending' ELSE 'active' END, true
		FROM unnest(ARRAY[-1, 15, 20, 30, 31, 60, 61]) n;
		INSERT INTO state_licenses VALUES (99, 'AL', NULL, 'No date', '', NULL, 'active', true);
		INSERT INTO license_expiration_alerts
		SELECT id, expiration_date, 60 FROM state_licenses WHERE id = 30
	`)
	if err != nil {
		t.Fatal(err)
	}
	fail := func(licenseAlertConfig, string) error { return errors.New("SMTP unavailable") }
	if err := sendDueLicenseAlerts(db, licenseAlertConfig{}, fail); err == nil {
		t.Fatal("expected SMTP error")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM license_expiration_alerts").Scan(&count); err != nil || count != 1 {
		t.Fatalf("failed delivery recorded: count=%d, error=%v", count, err)
	}
	var messages []string
	send := func(_ licenseAlertConfig, body string) error {
		messages = append(messages, body)
		return nil
	}
	if err := sendDueLicenseAlerts(db, licenseAlertConfig{}, send); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 4 {
		t.Fatalf("expected 4 catch-up alerts, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "Hueytown, AL") {
		t.Fatalf("missing jurisdiction: %q", messages[0])
	}
	if err := sendDueLicenseAlerts(db, licenseAlertConfig{}, send); err != nil || len(messages) != 4 {
		t.Fatalf("duplicate alerts: %d, error=%v", len(messages), err)
	}
	var thresholds string
	if err := db.QueryRow("SELECT string_agg(threshold_days::text, ',' ORDER BY threshold_days) FROM license_expiration_alerts WHERE license_id = 20").Scan(&thresholds); err != nil || thresholds != "30,60" {
		t.Fatalf("30-day catch-up should cover both thresholds: %q, error=%v", thresholds, err)
	}
	if _, err := db.Exec(`UPDATE state_licenses SET expiration_date = (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date + 30 WHERE id = 31`); err != nil {
		t.Fatal(err)
	}
	if err := sendDueLicenseAlerts(db, licenseAlertConfig{}, send); err != nil || len(messages) != 5 {
		t.Fatalf("expected new expiration to trigger alert: %d, error=%v", len(messages), err)
	}
}
