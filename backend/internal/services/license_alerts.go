package services

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type licenseAlertConfig struct {
	host, port, user, password, from string
	recipients                       []string
}

func licenseAlertSettings() (licenseAlertConfig, error) {
	config := licenseAlertConfig{
		host: os.Getenv("SMTP_HOST"), port: os.Getenv("SMTP_PORT"),
		user: os.Getenv("SMTP_USER"), password: os.Getenv("SMTP_PASSWORD"),
		from: os.Getenv("SMTP_FROM"),
	}
	if config.host == "" || config.user == "" || config.password == "" {
		return config, fmt.Errorf("SMTP_HOST, SMTP_USER and SMTP_PASSWORD are required")
	}
	if config.port == "" {
		config.port = "587"
	}
	if config.from == "" {
		config.from = config.user
	}
	from, err := mail.ParseAddress(config.from)
	if err != nil {
		return config, fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	config.from = from.Address
	for _, raw := range strings.Split(os.Getenv("LICENSE_ALERT_RECIPIENTS"), ",") {
		address, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			return config, fmt.Errorf("invalid license alert recipient: %w", err)
		}
		config.recipients = append(config.recipients, address.Address)
	}
	return config, nil
}

// StartLicenseAlerts enables the worker only when recipients are explicitly configured.
func StartLicenseAlerts(db *sql.DB) {
	if strings.TrimSpace(os.Getenv("LICENSE_ALERT_RECIPIENTS")) == "" {
		log.Print("License expiration alerts disabled: LICENSE_ALERT_RECIPIENTS is not set")
		return
	}
	config, err := licenseAlertSettings()
	if err != nil {
		log.Printf("License expiration alerts disabled: %v", err)
		return
	}
	log.Print("License expiration alerts enabled")
	go func() {
		run := func() {
			if err := sendDueLicenseAlerts(db, config, sendLicenseAlertEmail); err != nil {
				log.Printf("License expiration alert run failed; will retry: %v", err)
			}
		}
		run()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}

func sendDueLicenseAlerts(db *sql.DB, config licenseAlertConfig, send func(licenseAlertConfig, string) error) error {
	for {
		sent, err := sendNextLicenseAlert(db, config, send)
		if err != nil || !sent {
			return err
		}
	}
}

func sendNextLicenseAlert(db *sql.DB, config licenseAlertConfig, send func(licenseAlertConfig, string) error) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var locked bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock(1245852499, 1)").Scan(&locked); err != nil {
		return false, fmt.Errorf("acquire license alert lock: %w", err)
	}
	if !locked {
		return false, nil
	}

	var id, days, threshold int
	var state, city, licenseType, number, expiration string
	err = tx.QueryRow(`
		SELECT sl.id, sl.state, COALESCE(sl.city, ''), sl.license_type,
		       COALESCE(sl.license_number, ''), sl.expiration_date::text,
		       sl.expiration_date - (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date AS days,
		       CASE WHEN sl.expiration_date - (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date <= 30 THEN 30 ELSE 60 END AS threshold
		FROM state_licenses sl
		WHERE sl.is_active IS DISTINCT FROM FALSE
		  AND COALESCE(sl.status, '') <> 'pending'
		  AND sl.expiration_date BETWEEN (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date
		      AND (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date + 60
		  AND NOT EXISTS (
		      SELECT 1 FROM license_expiration_alerts a
		      WHERE a.license_id = sl.id AND a.expiration_date = sl.expiration_date
		        AND a.threshold_days = CASE WHEN sl.expiration_date - (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date <= 30 THEN 30 ELSE 60 END
		  )
		ORDER BY sl.expiration_date, sl.id
		FOR UPDATE OF sl SKIP LOCKED
		LIMIT 1
	`).Scan(&id, &state, &city, &licenseType, &number, &expiration, &days, &threshold)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("find due license: %w", err)
	}
	jurisdiction := state
	if city != "" {
		jurisdiction = city + ", " + state
	}
	body := fmt.Sprintf("A license is expiring in %d days.\r\n\r\nJurisdiction: %s\r\nLicense type: %s\r\nLicense number: %s\r\nExpiration date: %s\r\n\r\nReview this license in the JBS portal.\r\n", days, jurisdiction, licenseType, number, expiration)
	if err := send(config, body); err != nil {
		return false, fmt.Errorf("send alert for license %d: %w", id, err)
	}
	_, err = tx.Exec(`
		INSERT INTO license_expiration_alerts (license_id, expiration_date, threshold_days)
		SELECT $1, $2::date, threshold FROM unnest($3::int[]) AS threshold
		ON CONFLICT DO NOTHING
	`, id, expiration, alertThresholds(threshold))
	if err != nil {
		return false, fmt.Errorf("record alert for license %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit alert for license %d: %w", id, err)
	}
	return true, nil
}

func alertThresholds(threshold int) string {
	if threshold == 30 {
		return "{30,60}"
	}
	return "{60}"
}

func sendLicenseAlertEmail(config licenseAlertConfig, body string) error {
	address := net.JoinHostPort(config.host, config.port)
	conn, err := net.DialTimeout("tcp", address, 15*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, config.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: config.host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if err := client.Auth(smtp.PlainAuth("", config.user, config.password, config.host)); err != nil {
		return err
	}
	if err := client.Mail(config.from); err != nil {
		return err
	}
	for _, recipient := range config.recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: JBS license expiration reminder\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", config.from, strings.Join(config.recipients, ", "), body)
	if _, err := fmt.Fprint(writer, message); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := client.Quit(); err != nil {
		log.Printf("License alert accepted but SMTP quit failed: %v", err)
	}
	return nil
}
