package models

import (
	"encoding/json"
	"time"
)

// PATenant is a white-label tenant for the payment application tool.
// JBS Construction is the only tenant in v1; additional GCs can be seeded
// as rows without schema changes.
type PATenant struct {
	ID          int             `json:"id"`
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	APEmail     string          `json:"ap_email"`
	BrandConfig json.RawMessage `json:"brand_config"`
	Active      bool            `json:"active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// PaymentApplication is the core submission record.
// One row is created per submitted application. The submission_token is the
// public-facing no-auth identifier included in confirmation emails.
type PaymentApplication struct {
	ID              int    `json:"id"`
	TenantID        int    `json:"tenant_id"`
	SubmissionToken string `json:"submission_token"`

	// Step 1 — Subcontractor contact
	CompanyName  string `json:"company_name"`
	ContactName  string `json:"contact_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	Zip          string `json:"zip"`

	// Step 2 — Project info
	ProjectName       string     `json:"project_name"`
	ProjectNumber     string     `json:"project_number"`
	Owner             string     `json:"owner"`
	Contractor        string     `json:"contractor"`
	ContractDate      *time.Time `json:"contract_date"`
	ApplicationNumber int        `json:"application_number"`
	PeriodTo          *time.Time `json:"period_to"`

	// Step 3 — Contract summary inputs
	OriginalContractSum  float64 `json:"original_contract_sum"`
	RetainagePercent     float64 `json:"retainage_percent"`
	PreviousCertificates float64 `json:"previous_certificates"`
	AdditionalNotes      string  `json:"additional_notes"`

	// Calculated totals stored at submission time
	CalcNetChangeOrders      float64 `json:"calc_net_change_orders"`
	CalcContractSumToDate    float64 `json:"calc_contract_sum_to_date"`
	CalcTotalCompletedStored float64 `json:"calc_total_completed_stored"`
	CalcRetainageAmount      float64 `json:"calc_retainage_amount"`
	CalcEarnedLessRetainage  float64 `json:"calc_earned_less_retainage"`
	CalcCurrentPaymentDue    float64 `json:"calc_current_payment_due"`
	CalcBalanceToFinish      float64 `json:"calc_balance_to_finish"`

	// Full raw payload stored at submission time for audit and PDF stability
	SubmissionSnapshot json.RawMessage `json:"submission_snapshot,omitempty"`

	// Stripe payment status (placeholder; not active in v1)
	PaymentStatus           string     `json:"payment_status"`
	StripeCheckoutSessionID string     `json:"stripe_checkout_session_id,omitempty"`
	StripePaymentIntentID   string     `json:"stripe_payment_intent_id,omitempty"`
	PaymentAmountCents      int        `json:"payment_amount_cents"`
	PaidAt                  *time.Time `json:"paid_at,omitempty"`

	// PDF generation status (placeholder; not active in v1)
	PDFStatus          string     `json:"pdf_status"`
	PDFStorageKey      string     `json:"pdf_storage_key,omitempty"`
	PDFGeneratedAt     *time.Time `json:"pdf_generated_at,omitempty"`
	PDFDownloadToken   string     `json:"pdf_download_token,omitempty"`
	PDFDownloadExpires *time.Time `json:"pdf_download_expires_at,omitempty"`

	// Subcontractor email delivery status (placeholder; not active in v1)
	EmailStatus          string     `json:"email_status"`
	EmailSentAt          *time.Time `json:"email_sent_at,omitempty"`
	EmailResendMessageID string     `json:"email_resend_message_id,omitempty"`

	// AP copy delivery status (placeholder; not active in v1)
	APEmailStatus          string     `json:"ap_email_status"`
	APEmailSentAt          *time.Time `json:"ap_email_sent_at,omitempty"`
	APEmailResendMessageID string     `json:"ap_email_resend_message_id,omitempty"`

	// Admin review (placeholder; not active in v1)
	ReviewStatus string     `json:"review_status"`
	ReviewedBy   *int       `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes  string     `json:"review_notes,omitempty"`

	// Audit / error tracking
	ErrorLog  json.RawMessage `json:"error_log,omitempty"`
	IPAddress string          `json:"-"` // never sent to client
	UserAgent string          `json:"-"` // never sent to client

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships — populated via joins, not stored in this table
	ChangeOrders []PAChangeOrder `json:"change_orders,omitempty"`
	LineItems    []PALineItem    `json:"line_items,omitempty"`
}

// PAChangeOrder is one change order row from Step 4 of the form.
type PAChangeOrder struct {
	ID                   int        `json:"id"`
	PaymentApplicationID int        `json:"payment_application_id"`
	CONumber             string     `json:"co_number"`
	Description          string     `json:"description"`
	Amount               float64    `json:"amount"`
	DateApproved         *time.Time `json:"date_approved,omitempty"`
	SortOrder            int        `json:"sort_order"`
	CreatedAt            time.Time  `json:"created_at"`
}

// PALineItem is one Schedule of Values row from Step 5 (continuation sheet).
type PALineItem struct {
	ID                   int     `json:"id"`
	PaymentApplicationID int     `json:"payment_application_id"`
	ItemNo               string  `json:"item_no"`
	Description          string  `json:"description"`
	ScheduledValue       float64 `json:"scheduled_value"`
	PrevCompleted        float64 `json:"prev_completed"`
	ThisPeriod           float64 `json:"this_period"`
	MaterialsStored      float64 `json:"materials_stored"`

	// Stored calculated values (computed at submission time for PDF stability)
	CalcTotalCompleted  float64 `json:"calc_total_completed"`
	CalcPercentComplete float64 `json:"calc_percent_complete"`
	CalcBalanceToFinish float64 `json:"calc_balance_to_finish"`
	CalcRetainage       float64 `json:"calc_retainage"`

	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// PaymentApplicationListItem is a lightweight projection for admin list views.
type PaymentApplicationListItem struct {
	ID                    int        `json:"id"`
	SubmissionToken       string     `json:"submission_token"`
	CompanyName           string     `json:"company_name"`
	ProjectName           string     `json:"project_name"`
	ApplicationNumber     int        `json:"application_number"`
	PeriodTo              *time.Time `json:"period_to"`
	CalcCurrentPaymentDue float64    `json:"calc_current_payment_due"`
	PaymentStatus         string     `json:"payment_status"`
	ReviewStatus          string     `json:"review_status"`
	CreatedAt             time.Time  `json:"created_at"`
}
