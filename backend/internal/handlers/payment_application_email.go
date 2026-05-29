package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
)

// sanitizeHeaderValue removes CR and LF characters from a string before it is
// placed into an email header. Without this, user-supplied data (e.g. company
// names stored in the database) could inject extra headers (header injection).
func sanitizeHeaderValue(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// sendPDFEmail sends an email with a PDF file attached.
//
// It re-uses the same SMTP environment variables as sendInviteEmail:
//
//	SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM
//
// If SMTP is not configured the function returns a non-nil error so the
// caller can log the failure without aborting the webhook response.
func sendPDFEmail(toEmail, subject, htmlBody, attachmentName string, attachmentData []byte) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP not configured (SMTP_HOST, SMTP_USER, SMTP_PASSWORD required)")
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if fromEmail == "" {
		fromEmail = smtpUser
	}

	// Sanitize header values to prevent email header injection.
	// CompanyName (and therefore subject) is user-supplied data from the DB.
	safeSubject := sanitizeHeaderValue(subject)
	safeTo := sanitizeHeaderValue(toEmail)
	safeAttachment := sanitizeHeaderValue(attachmentName)

	// Build a MIME multipart/mixed message with an HTML part and a PDF attachment.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	// Top-level headers
	fmt.Fprintf(&buf, "From: %s\r\n", fromEmail)
	fmt.Fprintf(&buf, "To: %s\r\n", safeTo)
	fmt.Fprintf(&buf, "Subject: %s\r\n", safeSubject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n", boundary)
	fmt.Fprintf(&buf, "\r\n")

	// — HTML body part —
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return fmt.Errorf("create html part: %w", err)
	}
	if _, err := fmt.Fprint(htmlPart, htmlBody); err != nil {
		return fmt.Errorf("write html part: %w", err)
	}

	// — PDF attachment part —
	pdfHeader := make(textproto.MIMEHeader)
	pdfHeader.Set("Content-Type", "application/pdf")
	pdfHeader.Set("Content-Transfer-Encoding", "base64")
	pdfHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeAttachment))
	pdfPart, err := writer.CreatePart(pdfHeader)
	if err != nil {
		return fmt.Errorf("create pdf part: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(attachmentData)
	// RFC 2045: lines should be no longer than 76 characters.
	for len(encoded) > 76 {
		if _, err := fmt.Fprintf(pdfPart, "%s\r\n", encoded[:76]); err != nil {
			return fmt.Errorf("write pdf chunk: %w", err)
		}
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		if _, err := fmt.Fprintf(pdfPart, "%s\r\n", encoded); err != nil {
			return fmt.Errorf("write pdf tail: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close mime writer: %w", err)
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	return smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		fromEmail,
		[]string{safeTo},
		buf.Bytes(),
	)
}
