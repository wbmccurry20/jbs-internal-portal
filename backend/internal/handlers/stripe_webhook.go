package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/pdf"
)

// constructStripeEvent is set to webhook.ConstructEvent at startup.
// Unit tests replace it to inject pre-built events without needing a real Stripe signature.
var constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sigHeader, secret)
}

// dbGetPAByStripeSession and dbUpdatePaymentStatus are function vars so unit tests can
// inject stubs without a live database connection.
var dbGetPAByStripeSession = func(sessionID string) (*database.PAStripeInfo, error) {
	return database.GetPaymentApplicationByStripeSession(sessionID)
}

var dbUpdatePaymentStatus = func(id int, status string, paidAt *time.Time, paymentIntentID string) error {
	return database.UpdatePaymentStatus(id, status, paidAt, paymentIntentID)
}

// dbGetPAForPDF, generatePDF, and sendPAEmail are function vars so unit tests can inject
// stubs without a live database, PDF library, or SMTP server.
var dbGetPAForPDF = func(id int) (*database.PaymentApplicationPDFData, error) {
	return database.GetPaymentApplicationForPDF(id)
}

var generatePDF = func(data *database.PaymentApplicationPDFData) ([]byte, error) {
	return pdf.GeneratePDF(data)
}

var sendPAEmail = func(toEmail, subject, htmlBody, attachmentName string, pdfBytes []byte) error {
	return sendPDFEmail(toEmail, subject, htmlBody, attachmentName, pdfBytes)
}

// StripeWebhook handles POST /api/v1/stripe/webhook.
//
// This route must receive the raw request body to pass to Stripe's signature
// verifier. Do NOT apply any JSON body-parsing or CSRF middleware to this route.
func StripeWebhook(c *gin.Context) {
	signingSecret := os.Getenv("STRIPE_WEBHOOK_SIGNING_SECRET")
	if signingSecret == "" {
		log.Println("WARN: STRIPE_WEBHOOK_SIGNING_SECRET not set — webhook rejected")
		c.Status(http.StatusBadRequest)
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	event, err := constructStripeEvent(payload, sigHeader, signingSecret)
	if err != nil {
		log.Printf("WARN: stripe webhook signature verification failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook signature"})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		if err := handleCheckoutSessionCompleted(event); err != nil {
			log.Printf("ERROR: stripe checkout.session.completed handling: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	default:
		log.Printf("INFO: stripe webhook event type %q received and ignored", event.Type)
	}

	c.Status(http.StatusOK)
}

// handleCheckoutSessionCompleted processes a checkout.session.completed Stripe event.
// It looks up the submission by Stripe session ID, guards against duplicate delivery,
// and marks the submission as paid.
func handleCheckoutSessionCompleted(event stripe.Event) error {
	var cs stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
		return fmt.Errorf("unmarshal checkout session: %w", err)
	}
	if cs.ID == "" {
		return fmt.Errorf("checkout session has empty ID")
	}

	pa, err := dbGetPAByStripeSession(cs.ID)
	if err == sql.ErrNoRows {
		// Stripe may fire events for sessions unrelated to payment applications.
		log.Printf("INFO: checkout.session.completed for unknown session %q — ignored", cs.ID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup by stripe session %q: %w", cs.ID, err)
	}

	// Idempotency: a network retry should not re-update an already-paid record.
	if pa.PaymentStatus == "paid" {
		log.Printf("INFO: checkout.session.completed for already-paid pa id=%d — ignored", pa.ID)
		return nil
	}

	paymentIntentID := ""
	if cs.PaymentIntent != nil {
		paymentIntentID = cs.PaymentIntent.ID
	}

	now := time.Now().UTC()
	if err := dbUpdatePaymentStatus(pa.ID, "paid", &now, paymentIntentID); err != nil {
		return fmt.Errorf("update payment status pa id=%d: %w", pa.ID, err)
	}

	log.Printf("INFO: pa id=%d marked as paid (session=%q, intent=%q) — generating PDF and sending email",
		pa.ID, cs.ID, paymentIntentID)

	// Load full data for PDF generation. If this fails we log and continue — do not
	// fail the webhook response, which would cause Stripe to retry.
	paData, err := dbGetPAForPDF(pa.ID)
	if err != nil {
		log.Printf("ERROR: pa id=%d GetPaymentApplicationForPDF: %v", pa.ID, err)
		return nil
	}

	// Generate the PDF in memory.
	pdfBytes, err := generatePDF(paData)
	if err != nil {
		log.Printf("ERROR: pa id=%d GeneratePDF: %v", pa.ID, err)
		return nil
	}

	// Determine the AP recipient address.
	apEmail := paData.APEmail
	if apEmail == "" {
		apEmail = os.Getenv("OWNER_EMAIL")
	}
	if apEmail == "" {
		log.Printf("WARN: pa id=%d no AP email configured — skipping delivery", pa.ID)
		return nil
	}

	subject := fmt.Sprintf("Payment Application Received — %s — Application #%d",
		paData.CompanyName, paData.ApplicationNumber)
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#333;">
  <div style="background:#1e2937;padding:24px;border-radius:8px 8px 0 0;text-align:center;">
    <h1 style="color:white;margin:0;font-size:22px;">%s</h1>
    <p style="color:#00a0e0;margin:8px 0 0;font-size:13px;text-transform:uppercase;letter-spacing:0.08em;">Application for Payment</p>
  </div>
  <div style="background:#f9fafb;padding:32px;border:1px solid #e5e7eb;border-top:none;border-radius:0 0 8px 8px;">
    <p>A payment application has been submitted and payment confirmed.</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0;">
      <tr><td style="padding:6px 0;color:#6b7280;font-size:13px;">Company</td><td style="padding:6px 0;font-size:13px;">%s</td></tr>
      <tr><td style="padding:6px 0;color:#6b7280;font-size:13px;">Application #</td><td style="padding:6px 0;font-size:13px;">%d</td></tr>
      <tr><td style="padding:6px 0;color:#6b7280;font-size:13px;">Current Payment Due</td><td style="padding:6px 0;font-size:13px;font-weight:600;color:#00a0e0;">$%.2f</td></tr>
    </table>
    <p style="color:#6b7280;font-size:13px;">The full PDF is attached to this email.</p>
  </div>
</body>
</html>`,
		paData.TenantName,
		paData.CompanyName,
		paData.ApplicationNumber,
		paData.CalcCurrentPaymentDue,
	)

	attachmentName := fmt.Sprintf("payment-application-%s.pdf", paData.SubmissionToken)

	if err := sendPAEmail(apEmail, subject, htmlBody, attachmentName, pdfBytes); err != nil {
		log.Printf("ERROR: pa id=%d email delivery to %q failed: %v", pa.ID, apEmail, err)
	} else {
		log.Printf("INFO: pa id=%d PDF emailed to %q (%s)", pa.ID, apEmail, attachmentName)
	}

	return nil
}
