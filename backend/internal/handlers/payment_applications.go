package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v82"
	stripesession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

// ─── Request DTOs ──────────────────────────────────────────────────────────────

// ChangeOrderInput represents one change order row sent by the frontend form (Step 4).
type ChangeOrderInput struct {
	CONumber     string  `json:"co_number"`
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	DateApproved string  `json:"date_approved"` // ISO date "YYYY-MM-DD"; empty = no date
	SortOrder    int     `json:"sort_order"`
}

// LineItemInput represents one Schedule of Values row sent by the frontend form (Step 5).
type LineItemInput struct {
	ItemNo          string  `json:"item_no"`
	Description     string  `json:"description"`
	ScheduledValue  float64 `json:"scheduled_value"`
	PrevCompleted   float64 `json:"prev_completed"`
	ThisPeriod      float64 `json:"this_period"`
	MaterialsStored float64 `json:"materials_stored"`
	SortOrder       int     `json:"sort_order"`
}

// CreatePaymentApplicationRequest is the request body for POST /api/v1/payment-applications.
type CreatePaymentApplicationRequest struct {
	// TenantSlug identifies the GC tenant. Defaults to "jbs" if omitted.
	TenantSlug string `json:"tenant_slug"`

	// Step 1 — Subcontractor contact
	CompanyName  string `json:"company_name"  binding:"required"`
	ContactName  string `json:"contact_name"  binding:"required"`
	Email        string `json:"email"         binding:"required,email"`
	Phone        string `json:"phone"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	Zip          string `json:"zip"`

	// Step 2 — Project info
	ProjectName       string `json:"project_name"        binding:"required"`
	ProjectNumber     string `json:"project_number"`
	Owner             string `json:"owner"`
	Contractor        string `json:"contractor"`
	ContractDate      string `json:"contract_date"` // YYYY-MM-DD
	ApplicationNumber int    `json:"application_number"  binding:"required,min=1"`
	PeriodTo          string `json:"period_to"` // YYYY-MM-DD

	// Step 3 — Contract summary
	OriginalContractSum  float64 `json:"original_contract_sum"  binding:"required,gt=0"`
	RetainagePercent     float64 `json:"retainage_percent"      binding:"min=0,max=50"`
	PreviousCertificates float64 `json:"previous_certificates"`
	AdditionalNotes      string  `json:"additional_notes"`

	// Step 4 — Change orders (optional; may be empty)
	ChangeOrders []ChangeOrderInput `json:"change_orders"`

	// Step 5 — Line items / Schedule of Values (optional; may be empty)
	LineItems []LineItemInput `json:"line_items"`
}

// ─── Response DTOs ─────────────────────────────────────────────────────────────

// CalculatedTotals contains every server-side computed total for a payment application.
type CalculatedTotals struct {
	NetChangeOrders      float64 `json:"net_change_orders"`
	ContractSumToDate    float64 `json:"contract_sum_to_date"`
	TotalCompletedStored float64 `json:"total_completed_stored"`
	RetainageAmount      float64 `json:"retainage_amount"`
	EarnedLessRetainage  float64 `json:"earned_less_retainage"`
	CurrentPaymentDue    float64 `json:"current_payment_due"`
	BalanceToFinish      float64 `json:"balance_to_finish"`
}

// CreatePaymentApplicationResponse is returned on successful POST.
type CreatePaymentApplicationResponse struct {
	ID              int              `json:"id"`
	SubmissionToken string           `json:"submission_token"`
	Message         string           `json:"message"`
	Totals          CalculatedTotals `json:"totals"`
	// CheckoutURL is the Stripe Checkout Session URL. Empty when STRIPE_SECRET_KEY is not set.
	CheckoutURL string `json:"checkout_url,omitempty"`
}

// ─── Calculation helpers ───────────────────────────────────────────────────────

// calculateTotals computes all server-side totals from the validated request.
// Frontend-supplied totals are intentionally ignored; only the inputs are trusted.
func calculateTotals(req *CreatePaymentApplicationRequest) CalculatedTotals {
	var netCO float64
	for _, co := range req.ChangeOrders {
		netCO += co.Amount
	}

	contractSumToDate := req.OriginalContractSum + netCO

	var totalCompleted float64
	for _, li := range req.LineItems {
		totalCompleted += li.PrevCompleted + li.ThisPeriod + li.MaterialsStored
	}

	retainageAmount := totalCompleted * req.RetainagePercent / 100.0
	earnedLessRet := totalCompleted - retainageAmount
	currentPaymentDue := earnedLessRet - req.PreviousCertificates
	balanceToFinish := contractSumToDate - totalCompleted

	return CalculatedTotals{
		NetChangeOrders:      netCO,
		ContractSumToDate:    contractSumToDate,
		TotalCompletedStored: totalCompleted,
		RetainageAmount:      retainageAmount,
		EarnedLessRetainage:  earnedLessRet,
		CurrentPaymentDue:    currentPaymentDue,
		BalanceToFinish:      balanceToFinish,
	}
}

// calcLineItemValues computes the four stored calculated fields for one line item.
func calcLineItemValues(li LineItemInput, retainagePct float64) (calcTotal, calcPct, calcBalance, calcRetainage float64) {
	calcTotal = li.PrevCompleted + li.ThisPeriod + li.MaterialsStored
	if li.ScheduledValue > 0 {
		calcPct = calcTotal / li.ScheduledValue * 100.0
	}
	calcBalance = li.ScheduledValue - calcTotal
	calcRetainage = calcTotal * retainagePct / 100.0
	return
}

// parseDate converts an ISO date string ("YYYY-MM-DD") to *time.Time.
// Returns (nil, nil) for an empty string (date fields are optional).
// Returns (nil, error) if the string is non-empty but cannot be parsed as YYYY-MM-DD.
func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: expected YYYY-MM-DD format", s)
	}
	return &t, nil
}

// isValidSubmissionToken returns true if s is exactly 64 hexadecimal characters.
// Submission tokens are 64-char hex strings produced by generateSecureToken().
func isValidSubmissionToken(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// CreatePaymentApplication handles POST /api/v1/payment-applications.
// Public endpoint — no JWT required. Accepts the full form payload, computes server-side
// totals, persists all rows in a transaction, and returns the submission token.
func CreatePaymentApplication(c *gin.Context) {
	var req CreatePaymentApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize inputs
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.TenantSlug == "" {
		req.TenantSlug = "jbs"
	}
	// State codes must be at most 2 characters
	if len(req.State) > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state must be a 2-letter US state code"})
		return
	}

	// Validate optional date fields before the DB call so we can return 400 immediately
	if _, err := parseDate(req.ContractDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contract_date: " + err.Error()})
		return
	}
	if _, err := parseDate(req.PeriodTo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period_to: " + err.Error()})
		return
	}
	for i, co := range req.ChangeOrders {
		if _, err := parseDate(co.DateApproved); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("change_orders[%d].date_approved: %s", i, err.Error())})
			return
		}
	}

	// Look up tenant
	tenantID, err := database.GetTenantIDBySlug(req.TenantSlug)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown tenant"})
		return
	}
	if err != nil {
		log.Printf("ERROR: pa tenant lookup slug=%q: %v", req.TenantSlug, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process submission"})
		return
	}

	// Generate cryptographically random submission token (64-char hex, 32 bytes entropy)
	token, err := generateSecureToken()
	if err != nil {
		log.Printf("ERROR: pa token generation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process submission"})
		return
	}

	// Compute server-side totals — frontend-supplied totals are never trusted
	totals := calculateTotals(&req)

	// Capture submission snapshot for audit trail and future PDF stability
	snapshotBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("ERROR: pa snapshot marshal: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process submission"})
		return
	}

	// Build the parent row input — dates were already validated above; errors discarded safely
	contractDate, _ := parseDate(req.ContractDate)
	periodTo, _ := parseDate(req.PeriodTo)
	paInput := database.PACreateInput{
		TenantID:              tenantID,
		Token:                 token,
		CompanyName:           req.CompanyName,
		ContactName:           req.ContactName,
		Email:                 req.Email,
		Phone:                 req.Phone,
		AddressLine1:          req.AddressLine1,
		AddressLine2:          req.AddressLine2,
		City:                  req.City,
		State:                 req.State,
		Zip:                   req.Zip,
		ProjectName:           req.ProjectName,
		ProjectNumber:         req.ProjectNumber,
		Owner:                 req.Owner,
		Contractor:            req.Contractor,
		ContractDate:          contractDate,
		ApplicationNumber:     req.ApplicationNumber,
		PeriodTo:              periodTo,
		OriginalContractSum:   req.OriginalContractSum,
		RetainagePercent:      req.RetainagePercent,
		PreviousCertificates:  req.PreviousCertificates,
		AdditionalNotes:       req.AdditionalNotes,
		CalcNetChangeOrders:   totals.NetChangeOrders,
		CalcContractSumToDate: totals.ContractSumToDate,
		CalcTotalCompleted:    totals.TotalCompletedStored,
		CalcRetainageAmount:   totals.RetainageAmount,
		CalcEarnedLessRet:     totals.EarnedLessRetainage,
		CalcCurrentPaymentDue: totals.CurrentPaymentDue,
		CalcBalanceToFinish:   totals.BalanceToFinish,
		SnapshotJSON:          string(snapshotBytes),
		IPAddress:             c.ClientIP(),
		UserAgent:             c.GetHeader("User-Agent"),
	}

	// Build change order inputs — dates were already validated above; errors discarded safely
	coInputs := make([]database.COCreateInput, len(req.ChangeOrders))
	for i, co := range req.ChangeOrders {
		dateApproved, _ := parseDate(co.DateApproved)
		coInputs[i] = database.COCreateInput{
			CONumber:     co.CONumber,
			Description:  co.Description,
			Amount:       co.Amount,
			DateApproved: dateApproved,
			SortOrder:    co.SortOrder,
		}
	}

	// Build line item inputs with per-row server-side calculations
	liInputs := make([]database.LICreateInput, len(req.LineItems))
	for i, li := range req.LineItems {
		ct, cp, cb, cr := calcLineItemValues(li, req.RetainagePercent)
		liInputs[i] = database.LICreateInput{
			ItemNo:              li.ItemNo,
			Description:         li.Description,
			ScheduledValue:      li.ScheduledValue,
			PrevCompleted:       li.PrevCompleted,
			ThisPeriod:          li.ThisPeriod,
			MaterialsStored:     li.MaterialsStored,
			CalcTotalCompleted:  ct,
			CalcPercentComplete: cp,
			CalcBalanceToFinish: cb,
			CalcRetainage:       cr,
			SortOrder:           li.SortOrder,
		}
	}

	// Persist all rows in a single transaction
	paID, err := database.CreatePaymentApplication(paInput, coInputs, liInputs)
	if err != nil {
		log.Printf("ERROR: pa create token=%q: %v", token, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// Optionally create a Stripe Checkout Session (skipped when STRIPE_SECRET_KEY is unset)
	checkoutURL := ""
	if stripeKey := os.Getenv("STRIPE_SECRET_KEY"); stripeKey != "" {
		stripe.Key = stripeKey
		params := &stripe.CheckoutSessionParams{
			Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency: stripe.String("usd"),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name: stripe.String("JBS Application for Payment"),
						},
						UnitAmount: stripe.Int64(999),
					},
					Quantity: stripe.Int64(1),
				},
			},
			SuccessURL: stripe.String("https://buildwithjbs.com/client-portal/payment-application/success?session_id={CHECKOUT_SESSION_ID}"),
			CancelURL:  stripe.String("https://buildwithjbs.com/client-portal/payment-application?cancelled=true"),
			Metadata: map[string]string{
				"submission_token": token,
			},
		}
		sess, stripeErr := stripesession.New(params)
		if stripeErr != nil {
			log.Printf("WARN: stripe checkout session creation failed for pa id=%d: %v", paID, stripeErr)
		} else {
			checkoutURL = sess.URL
			if dbErr := database.UpdateStripeSessionID(paID, sess.ID); dbErr != nil {
				log.Printf("WARN: failed to store stripe session ID for pa id=%d: %v", paID, dbErr)
			}
		}
	}

	c.JSON(http.StatusCreated, CreatePaymentApplicationResponse{
		ID:              paID,
		SubmissionToken: token,
		Message:         "Payment application submitted successfully.",
		Totals:          totals,
		CheckoutURL:     checkoutURL,
	})
}

// GetPaymentApplication handles GET /api/v1/payment-applications/:submissionToken.
// Public endpoint — no JWT required. Returns the full submission with change orders and
// line items. Sensitive fields are never selected from the database.
func GetPaymentApplication(c *gin.Context) {
	token := c.Param("submissionToken")

	// Submission tokens are 64-char lowercase hex strings (32 bytes from generateSecureToken).
	// Reject tokens with wrong length or non-hex characters before hitting the database.
	if !isValidSubmissionToken(token) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission token format"})
		return
	}

	pa, err := database.GetPaymentApplicationByToken(token)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}
	if err != nil {
		log.Printf("ERROR: pa fetch token=%q: %v", token, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve submission"})
		return
	}

	c.JSON(http.StatusOK, pa)
}
