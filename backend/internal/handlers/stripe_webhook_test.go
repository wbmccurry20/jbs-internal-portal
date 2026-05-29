package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stretchr/testify/assert"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

func newWebhookRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/stripe/webhook", StripeWebhook)
	return r
}

// postWebhook sends a raw POST to the webhook endpoint.
func postWebhook(t *testing.T, body, sigHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest("POST", "/api/v1/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if sigHeader != "" {
		req.Header.Set("Stripe-Signature", sigHeader)
	}
	w := httptest.NewRecorder()
	newWebhookRouter().ServeHTTP(w, req)
	return w
}

// buildCheckoutSessionEvent builds a stripe.Event for checkout.session.completed.
// paymentIntentID may be empty to simulate a session with no PaymentIntent (e.g. free).
func buildCheckoutSessionEvent(sessionID, paymentIntentID string) stripe.Event {
	cs := map[string]interface{}{
		"id":             sessionID,
		"payment_intent": paymentIntentID,
	}
	raw, _ := json.Marshal(cs)
	return stripe.Event{
		Type: "checkout.session.completed",
		Data: &stripe.EventData{Raw: raw},
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestStripeWebhook_InvalidSignature verifies that a bad Stripe-Signature header returns 400.
func TestStripeWebhook_InvalidSignature(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SIGNING_SECRET", "whsec_test")

	origCons := constructStripeEvent
	defer func() { constructStripeEvent = origCons }()
	constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
		return stripe.Event{}, fmt.Errorf("computed signature does not match any known signature")
	}

	w := postWebhook(t, `{}`, "t=1,v1=badsig")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestStripeWebhook_ValidSignature_CheckoutCompleted verifies the happy path:
// a valid event for a pending PA marks it as paid and calls UpdatePaymentStatus.
func TestStripeWebhook_ValidSignature_CheckoutCompleted(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SIGNING_SECRET", "whsec_test")

	origCons := constructStripeEvent
	origGet := dbGetPAByStripeSession
	origUpdate := dbUpdatePaymentStatus
	origGetPDF := dbGetPAForPDF
	origGenPDF := generatePDF
	origSend := sendPAEmail
	defer func() {
		constructStripeEvent = origCons
		dbGetPAByStripeSession = origGet
		dbUpdatePaymentStatus = origUpdate
		dbGetPAForPDF = origGetPDF
		generatePDF = origGenPDF
		sendPAEmail = origSend
	}()

	constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
		return buildCheckoutSessionEvent("cs_test_001", "pi_test_001"), nil
	}
	dbGetPAByStripeSession = func(sessionID string) (*database.PAStripeInfo, error) {
		assert.Equal(t, "cs_test_001", sessionID)
		return &database.PAStripeInfo{ID: 42, PaymentStatus: "pending"}, nil
	}

	updateCalled := false
	dbUpdatePaymentStatus = func(id int, status string, paidAt *time.Time, paymentIntentID string) error {
		updateCalled = true
		assert.Equal(t, 42, id)
		assert.Equal(t, "paid", status)
		assert.NotNil(t, paidAt)
		assert.Equal(t, "pi_test_001", paymentIntentID)
		return nil
	}

	// Stub downstream PDF/email steps so this test stays focused on status update.
	dbGetPAForPDF = func(id int) (*database.PaymentApplicationPDFData, error) {
		return &database.PaymentApplicationPDFData{ID: 42, SubmissionToken: "tok42", ApplicationNumber: 1, CompanyName: "Test", APEmail: "ap@test.com"}, nil
	}
	generatePDF = func(data *database.PaymentApplicationPDFData) ([]byte, error) {
		return []byte("%PDF-stub"), nil
	}
	sendPAEmail = func(_, _, _, _ string, _ []byte) error { return nil }

	w := postWebhook(t, `{}`, "t=valid,v1=sig")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, updateCalled, "UpdatePaymentStatus should have been called")
}

// TestStripeWebhook_DuplicateEvent_AlreadyPaid verifies idempotency:
// a second checkout.session.completed for an already-paid PA must not call UpdatePaymentStatus.
func TestStripeWebhook_DuplicateEvent_AlreadyPaid(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SIGNING_SECRET", "whsec_test")

	origCons := constructStripeEvent
	origGet := dbGetPAByStripeSession
	origUpdate := dbUpdatePaymentStatus
	defer func() {
		constructStripeEvent = origCons
		dbGetPAByStripeSession = origGet
		dbUpdatePaymentStatus = origUpdate
	}()

	constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
		return buildCheckoutSessionEvent("cs_test_002", "pi_test_002"), nil
	}
	dbGetPAByStripeSession = func(sessionID string) (*database.PAStripeInfo, error) {
		return &database.PAStripeInfo{ID: 99, PaymentStatus: "paid"}, nil
	}

	updateCalled := false
	dbUpdatePaymentStatus = func(id int, status string, paidAt *time.Time, paymentIntentID string) error {
		updateCalled = true
		return nil
	}

	w := postWebhook(t, `{}`, "t=valid,v1=sig")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, updateCalled, "UpdatePaymentStatus must NOT be called for already-paid PA")
}

// TestStripeWebhook_UnknownEventType verifies that unhandled event types return 200 (not 4xx).
func TestStripeWebhook_UnknownEventType(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SIGNING_SECRET", "whsec_test")

	origCons := constructStripeEvent
	defer func() { constructStripeEvent = origCons }()
	constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
		return stripe.Event{Type: "customer.created"}, nil
	}

	w := postWebhook(t, `{}`, "t=valid,v1=sig")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestStripeWebhook_PDFEmailSent verifies that after marking a PA as paid, the handler
// loads the full PA data, generates a PDF, and calls sendPAEmail with the correct
// recipient, subject format, and attachment filename.
func TestStripeWebhook_PDFEmailSent(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SIGNING_SECRET", "whsec_test")

	origCons := constructStripeEvent
	origGet := dbGetPAByStripeSession
	origUpdate := dbUpdatePaymentStatus
	origGetPDF := dbGetPAForPDF
	origGenPDF := generatePDF
	origSend := sendPAEmail
	defer func() {
		constructStripeEvent = origCons
		dbGetPAByStripeSession = origGet
		dbUpdatePaymentStatus = origUpdate
		dbGetPAForPDF = origGetPDF
		generatePDF = origGenPDF
		sendPAEmail = origSend
	}()

	constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
		return buildCheckoutSessionEvent("cs_pdf_001", "pi_pdf_001"), nil
	}
	dbGetPAByStripeSession = func(sessionID string) (*database.PAStripeInfo, error) {
		return &database.PAStripeInfo{ID: 77, PaymentStatus: "pending"}, nil
	}
	dbUpdatePaymentStatus = func(id int, status string, paidAt *time.Time, paymentIntentID string) error {
		return nil
	}

	fakePAData := &database.PaymentApplicationPDFData{
		ID:                77,
		SubmissionToken:   "tok_test_0000000000000000000000000000000000000000000000000000000000",
		ApplicationNumber: 2,
		CompanyName:       "Acme Electrical LLC",
		APEmail:           "ap@jbstest.com",
		CalcCurrentPaymentDue: 12345.67,
	}
	dbGetPAForPDF = func(id int) (*database.PaymentApplicationPDFData, error) {
		assert.Equal(t, 77, id)
		return fakePAData, nil
	}

	fakePDF := []byte("%PDF-fake")
	generatePDF = func(data *database.PaymentApplicationPDFData) ([]byte, error) {
		return fakePDF, nil
	}

	var capturedTo, capturedSubject, capturedAttachmentName string
	var capturedPDFBytes []byte
	sendPAEmail = func(toEmail, subject, htmlBody, attachmentName string, pdfBytes []byte) error {
		capturedTo = toEmail
		capturedSubject = subject
		capturedAttachmentName = attachmentName
		capturedPDFBytes = pdfBytes
		return nil
	}

	w := postWebhook(t, `{}`, "t=valid,v1=sig")
	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "ap@jbstest.com", capturedTo)
	assert.Contains(t, capturedSubject, "Acme Electrical LLC")
	assert.Contains(t, capturedSubject, "Application #2")
	expectedFilename := "payment-application-tok_test_0000000000000000000000000000000000000000000000000000000000.pdf"
	assert.Equal(t, expectedFilename, capturedAttachmentName)
	assert.Equal(t, fakePDF, capturedPDFBytes)
}

// TestStripeWebhook_PDFGenerationFails_Returns200 verifies that if PDF generation fails,
// the webhook handler still returns 200 so Stripe does not retry.
func TestStripeWebhook_PDFGenerationFails_Returns200(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SIGNING_SECRET", "whsec_test")

	origCons := constructStripeEvent
	origGet := dbGetPAByStripeSession
	origUpdate := dbUpdatePaymentStatus
	origGetPDF := dbGetPAForPDF
	origGenPDF := generatePDF
	origSend := sendPAEmail
	defer func() {
		constructStripeEvent = origCons
		dbGetPAByStripeSession = origGet
		dbUpdatePaymentStatus = origUpdate
		dbGetPAForPDF = origGetPDF
		generatePDF = origGenPDF
		sendPAEmail = origSend
	}()

	constructStripeEvent = func(payload []byte, sigHeader, secret string) (stripe.Event, error) {
		return buildCheckoutSessionEvent("cs_fail_001", "pi_fail_001"), nil
	}
	dbGetPAByStripeSession = func(_ string) (*database.PAStripeInfo, error) {
		return &database.PAStripeInfo{ID: 88, PaymentStatus: "pending"}, nil
	}
	dbUpdatePaymentStatus = func(_ int, _ string, _ *time.Time, _ string) error { return nil }
	dbGetPAForPDF = func(id int) (*database.PaymentApplicationPDFData, error) {
		return &database.PaymentApplicationPDFData{
			ID: 88, SubmissionToken: "tok88", ApplicationNumber: 1,
			CompanyName: "Test Co", APEmail: "ap@test.com",
		}, nil
	}
	generatePDF = func(data *database.PaymentApplicationPDFData) ([]byte, error) {
		return nil, fmt.Errorf("simulated pdf generation failure")
	}

	emailCalled := false
	sendPAEmail = func(_, _, _, _ string, _ []byte) error {
		emailCalled = true
		return nil
	}

	w := postWebhook(t, `{}`, "t=valid,v1=sig")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, emailCalled, "email should not be called when PDF generation fails")
}

// Suppress "declared and not used" for sql.ErrNoRows import — used via database package only.
var _ = sql.ErrNoRows
