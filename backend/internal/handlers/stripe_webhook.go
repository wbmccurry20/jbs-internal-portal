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

	log.Printf("INFO: pa id=%d marked as paid (session=%q, intent=%q) — PDF generation not yet implemented",
		pa.ID, cs.ID, paymentIntentID)
	return nil
}
