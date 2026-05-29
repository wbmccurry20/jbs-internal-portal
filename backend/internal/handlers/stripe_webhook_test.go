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
	defer func() {
		constructStripeEvent = origCons
		dbGetPAByStripeSession = origGet
		dbUpdatePaymentStatus = origUpdate
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

// Suppress "declared and not used" for sql.ErrNoRows import — used via database package only.
var _ = sql.ErrNoRows
