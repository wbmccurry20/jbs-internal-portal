package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ─── Input types ───────────────────────────────────────────────────────────────

// PACreateInput carries all fields needed to insert one payment_applications row.
type PACreateInput struct {
	TenantID              int
	Token                 string
	CompanyName           string
	ContactName           string
	Email                 string
	Phone                 string
	AddressLine1          string
	AddressLine2          string
	City                  string
	State                 string
	Zip                   string
	ProjectName           string
	ProjectNumber         string
	Owner                 string
	Contractor            string
	ContractDate          *time.Time
	ApplicationNumber     int
	PeriodTo              *time.Time
	OriginalContractSum   float64
	RetainagePercent      float64
	PreviousCertificates  float64
	AdditionalNotes       string
	CalcNetChangeOrders   float64
	CalcContractSumToDate float64
	CalcTotalCompleted    float64
	CalcRetainageAmount   float64
	CalcEarnedLessRet     float64
	CalcCurrentPaymentDue float64
	CalcBalanceToFinish   float64
	// SnapshotJSON is the full request payload marshalled to a JSON string.
	// Stored as JSONB; passed as a string so lib/pq sends it correctly.
	SnapshotJSON string
	IPAddress    string
	UserAgent    string
}

// COCreateInput carries fields for one payment_application_change_orders row.
type COCreateInput struct {
	CONumber     string
	Description  string
	Amount       float64
	DateApproved *time.Time
	SortOrder    int
}

// LICreateInput carries fields for one payment_application_line_items row.
type LICreateInput struct {
	ItemNo              string
	Description         string
	ScheduledValue      float64
	PrevCompleted       float64
	ThisPeriod          float64
	MaterialsStored     float64
	CalcTotalCompleted  float64
	CalcPercentComplete float64
	CalcBalanceToFinish float64
	CalcRetainage       float64
	SortOrder           int
}

// ─── Response types ────────────────────────────────────────────────────────────

// PAFullResponse is the public-facing read model for one payment application.
// Sensitive fields (ip_address, user_agent, Stripe keys, internal review notes) are excluded.
type PAFullResponse struct {
	ID                       int          `json:"id"`
	SubmissionToken          string       `json:"submission_token"`
	CompanyName              string       `json:"company_name"`
	ContactName              string       `json:"contact_name"`
	Email                    string       `json:"email"`
	Phone                    string       `json:"phone"`
	AddressLine1             string       `json:"address_line1"`
	AddressLine2             string       `json:"address_line2"`
	City                     string       `json:"city"`
	State                    string       `json:"state"`
	Zip                      string       `json:"zip"`
	ProjectName              string       `json:"project_name"`
	ProjectNumber            string       `json:"project_number"`
	Owner                    string       `json:"owner"`
	Contractor               string       `json:"contractor"`
	ContractDate             *time.Time   `json:"contract_date"`
	ApplicationNumber        int          `json:"application_number"`
	PeriodTo                 *time.Time   `json:"period_to"`
	OriginalContractSum      float64      `json:"original_contract_sum"`
	RetainagePercent         float64      `json:"retainage_percent"`
	PreviousCertificates     float64      `json:"previous_certificates"`
	AdditionalNotes          string       `json:"additional_notes"`
	CalcNetChangeOrders      float64      `json:"calc_net_change_orders"`
	CalcContractSumToDate    float64      `json:"calc_contract_sum_to_date"`
	CalcTotalCompletedStored float64      `json:"calc_total_completed_stored"`
	CalcRetainageAmount      float64      `json:"calc_retainage_amount"`
	CalcEarnedLessRetainage  float64      `json:"calc_earned_less_retainage"`
	CalcCurrentPaymentDue    float64      `json:"calc_current_payment_due"`
	CalcBalanceToFinish      float64      `json:"calc_balance_to_finish"`
	PaymentStatus            string       `json:"payment_status"`
	ReviewStatus             string       `json:"review_status"`
	CreatedAt                time.Time    `json:"created_at"`
	ChangeOrders             []COResponse `json:"change_orders"`
	LineItems                []LIResponse `json:"line_items"`
}

// COResponse is one change order row in the GET /payment-applications/:token response.
type COResponse struct {
	ID           int        `json:"id"`
	CONumber     string     `json:"co_number"`
	Description  string     `json:"description"`
	Amount       float64    `json:"amount"`
	DateApproved *time.Time `json:"date_approved"`
	SortOrder    int        `json:"sort_order"`
	CreatedAt    time.Time  `json:"created_at"`
}

// LIResponse is one line item row in the GET /payment-applications/:token response.
type LIResponse struct {
	ID                  int       `json:"id"`
	ItemNo              string    `json:"item_no"`
	Description         string    `json:"description"`
	ScheduledValue      float64   `json:"scheduled_value"`
	PrevCompleted       float64   `json:"prev_completed"`
	ThisPeriod          float64   `json:"this_period"`
	MaterialsStored     float64   `json:"materials_stored"`
	CalcTotalCompleted  float64   `json:"calc_total_completed"`
	CalcPercentComplete float64   `json:"calc_percent_complete"`
	CalcBalanceToFinish float64   `json:"calc_balance_to_finish"`
	CalcRetainage       float64   `json:"calc_retainage"`
	SortOrder           int       `json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`
}

// ─── Repository functions ──────────────────────────────────────────────────────

// GetTenantIDBySlug returns the ID of an active tenant for the given slug.
// Returns sql.ErrNoRows if the tenant is not found or is inactive.
func GetTenantIDBySlug(slug string) (int, error) {
	var id int
	err := DB.QueryRow(
		"SELECT id FROM pa_tenants WHERE slug = $1 AND active = true",
		slug,
	).Scan(&id)
	return id, err
}

// CreatePaymentApplication inserts the parent application row, all change orders, and all line
// items inside a single database transaction. Returns the new payment_applications.id on success.
func CreatePaymentApplication(input PACreateInput, cos []COCreateInput, lis []LICreateInput) (int, error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op after Commit

	var paID int
	err = tx.QueryRow(`
		INSERT INTO payment_applications (
			tenant_id, submission_token,
			company_name, contact_name, email, phone,
			address_line1, address_line2, city, state, zip,
			project_name, project_number, owner, contractor,
			contract_date, application_number, period_to,
			original_contract_sum, retainage_percent, previous_certificates, additional_notes,
			calc_net_change_orders, calc_contract_sum_to_date, calc_total_completed_stored,
			calc_retainage_amount, calc_earned_less_retainage, calc_current_payment_due,
			calc_balance_to_finish,
			submission_snapshot,
			ip_address, user_agent
		) VALUES (
			$1,  $2,
			$3,  $4,  $5,  $6,
			$7,  $8,  $9,  $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18,
			$19, $20, $21, $22,
			$23, $24, $25,
			$26, $27, $28,
			$29,
			$30::jsonb,
			$31, $32
		) RETURNING id`,
		input.TenantID, input.Token,
		input.CompanyName, input.ContactName, input.Email, input.Phone,
		input.AddressLine1, input.AddressLine2, input.City, input.State, input.Zip,
		input.ProjectName, input.ProjectNumber, input.Owner, input.Contractor,
		input.ContractDate, input.ApplicationNumber, input.PeriodTo,
		input.OriginalContractSum, input.RetainagePercent, input.PreviousCertificates, input.AdditionalNotes,
		input.CalcNetChangeOrders, input.CalcContractSumToDate, input.CalcTotalCompleted,
		input.CalcRetainageAmount, input.CalcEarnedLessRet, input.CalcCurrentPaymentDue,
		input.CalcBalanceToFinish,
		input.SnapshotJSON,
		input.IPAddress, input.UserAgent,
	).Scan(&paID)
	if err != nil {
		return 0, fmt.Errorf("insert payment_application: %w", err)
	}

	for _, co := range cos {
		_, err = tx.Exec(`
			INSERT INTO payment_application_change_orders
				(payment_application_id, co_number, description, amount, date_approved, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			paID, co.CONumber, co.Description, co.Amount, co.DateApproved, co.SortOrder,
		)
		if err != nil {
			return 0, fmt.Errorf("insert change_order: %w", err)
		}
	}

	for _, li := range lis {
		_, err = tx.Exec(`
			INSERT INTO payment_application_line_items (
				payment_application_id, item_no, description,
				scheduled_value, prev_completed, this_period, materials_stored,
				calc_total_completed, calc_percent_complete, calc_balance_to_finish, calc_retainage,
				sort_order
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			paID, li.ItemNo, li.Description,
			li.ScheduledValue, li.PrevCompleted, li.ThisPeriod, li.MaterialsStored,
			li.CalcTotalCompleted, li.CalcPercentComplete, li.CalcBalanceToFinish, li.CalcRetainage,
			li.SortOrder,
		)
		if err != nil {
			return 0, fmt.Errorf("insert line_item: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}
	return paID, nil
}

// GetPaymentApplicationByToken fetches a full payment application by its public submission token,
// including all change orders and line items. Returns sql.ErrNoRows if the token is not found.
// Sensitive audit fields (ip_address, user_agent) and internal fields (Stripe, PDF, email keys)
// are never selected.
func GetPaymentApplicationByToken(token string) (*PAFullResponse, error) {
	var pa PAFullResponse
	err := DB.QueryRow(`
		SELECT
			id, submission_token,
			company_name, contact_name, email,
			COALESCE(phone,''),
			COALESCE(address_line1,''), COALESCE(address_line2,''),
			COALESCE(city,''), COALESCE(state,''), COALESCE(zip,''),
			project_name, COALESCE(project_number,''),
			COALESCE(owner,''), COALESCE(contractor,''),
			contract_date, application_number, period_to,
			original_contract_sum, retainage_percent, previous_certificates,
			COALESCE(additional_notes,''),
			calc_net_change_orders, calc_contract_sum_to_date, calc_total_completed_stored,
			calc_retainage_amount, calc_earned_less_retainage, calc_current_payment_due,
			calc_balance_to_finish,
			payment_status, review_status,
			created_at
		FROM payment_applications
		WHERE submission_token = $1`,
		token,
	).Scan(
		&pa.ID, &pa.SubmissionToken,
		&pa.CompanyName, &pa.ContactName, &pa.Email,
		&pa.Phone,
		&pa.AddressLine1, &pa.AddressLine2,
		&pa.City, &pa.State, &pa.Zip,
		&pa.ProjectName, &pa.ProjectNumber,
		&pa.Owner, &pa.Contractor,
		&pa.ContractDate, &pa.ApplicationNumber, &pa.PeriodTo,
		&pa.OriginalContractSum, &pa.RetainagePercent, &pa.PreviousCertificates,
		&pa.AdditionalNotes,
		&pa.CalcNetChangeOrders, &pa.CalcContractSumToDate, &pa.CalcTotalCompletedStored,
		&pa.CalcRetainageAmount, &pa.CalcEarnedLessRetainage, &pa.CalcCurrentPaymentDue,
		&pa.CalcBalanceToFinish,
		&pa.PaymentStatus, &pa.ReviewStatus,
		&pa.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch change orders ordered by sort_order, then insertion order
	coRows, err := DB.Query(`
		SELECT id, COALESCE(co_number,''), COALESCE(description,''),
		       amount, date_approved, sort_order, created_at
		FROM payment_application_change_orders
		WHERE payment_application_id = $1
		ORDER BY sort_order, id`,
		pa.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch change orders: %w", err)
	}
	defer coRows.Close()

	pa.ChangeOrders = []COResponse{}
	for coRows.Next() {
		var co COResponse
		if err := coRows.Scan(
			&co.ID, &co.CONumber, &co.Description,
			&co.Amount, &co.DateApproved, &co.SortOrder, &co.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan change order: %w", err)
		}
		pa.ChangeOrders = append(pa.ChangeOrders, co)
	}
	if err := coRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate change orders: %w", err)
	}

	// Fetch line items ordered by sort_order, then insertion order
	liRows, err := DB.Query(`
		SELECT
			id, COALESCE(item_no,''), COALESCE(description,''),
			scheduled_value, prev_completed, this_period, materials_stored,
			calc_total_completed, calc_percent_complete, calc_balance_to_finish, calc_retainage,
			sort_order, created_at
		FROM payment_application_line_items
		WHERE payment_application_id = $1
		ORDER BY sort_order, id`,
		pa.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch line items: %w", err)
	}
	defer liRows.Close()

	pa.LineItems = []LIResponse{}
	for liRows.Next() {
		var li LIResponse
		if err := liRows.Scan(
			&li.ID, &li.ItemNo, &li.Description,
			&li.ScheduledValue, &li.PrevCompleted, &li.ThisPeriod, &li.MaterialsStored,
			&li.CalcTotalCompleted, &li.CalcPercentComplete, &li.CalcBalanceToFinish, &li.CalcRetainage,
			&li.SortOrder, &li.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan line item: %w", err)
		}
		pa.LineItems = append(pa.LineItems, li)
	}
	if err := liRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate line items: %w", err)
	}

	return &pa, nil
}

// ─── Stripe helpers ────────────────────────────────────────────────────────────

// ─── PDF data types ────────────────────────────────────────────────────────────

// PDFChangeOrder holds the change-order fields required to render the PDF.
type PDFChangeOrder struct {
	CONumber     string
	Description  string
	Amount       float64
	DateApproved *time.Time
	SortOrder    int
}

// PDFLineItem holds the line-item fields required to render the PDF schedule of values.
type PDFLineItem struct {
	ItemNo              string
	Description         string
	ScheduledValue      float64
	PrevCompleted       float64
	ThisPeriod          float64
	MaterialsStored     float64
	CalcTotalCompleted  float64
	CalcPercentComplete float64
	CalcBalanceToFinish float64
	SortOrder           int
}

// PaymentApplicationPDFData aggregates all fields needed to generate the payment
// application PDF in a single flat struct. Loaded by GetPaymentApplicationForPDF.
type PaymentApplicationPDFData struct {
	// Identity
	ID                int
	SubmissionToken   string
	ApplicationNumber int
	PeriodTo          *time.Time
	CreatedAt         time.Time

	// Tenant
	TenantName string
	APEmail    string

	// Subcontractor contact (Step 1)
	CompanyName  string
	ContactName  string
	Email        string
	Phone        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Zip          string

	// Project (Step 2)
	ProjectName   string
	ProjectNumber string
	Owner         string
	Contractor    string
	ContractDate  *time.Time

	// Contract inputs (Step 3)
	OriginalContractSum  float64
	RetainagePercent     float64
	PreviousCertificates float64
	AdditionalNotes      string

	// Calculated totals (stored at submission time)
	CalcNetChangeOrders     float64
	CalcContractSumToDate   float64
	CalcTotalCompleted      float64
	CalcRetainageAmount     float64
	CalcEarnedLessRetainage float64
	CalcCurrentPaymentDue   float64
	CalcBalanceToFinish     float64

	// Children
	ChangeOrders []PDFChangeOrder
	LineItems    []PDFLineItem
}

// GetPaymentApplicationForPDF loads everything needed to generate a PDF for the
// given payment_applications.id. It performs a JOIN to pa_tenants for the AP email,
// then two follow-up queries for change orders and line items.
// Returns sql.ErrNoRows if the id does not exist.
func GetPaymentApplicationForPDF(id int) (*PaymentApplicationPDFData, error) {
	var pa PaymentApplicationPDFData

	err := DB.QueryRow(`
		SELECT
			pa.id, pa.submission_token, pa.application_number,
			pa.period_to, pa.created_at,
			t.name, COALESCE(t.ap_email, ''),
			pa.company_name, pa.contact_name, pa.email, COALESCE(pa.phone, ''),
			COALESCE(pa.address_line1, ''), COALESCE(pa.address_line2, ''),
			COALESCE(pa.city, ''), COALESCE(pa.state, ''), COALESCE(pa.zip, ''),
			pa.project_name, COALESCE(pa.project_number, ''),
			COALESCE(pa.owner, ''), COALESCE(pa.contractor, ''),
			pa.contract_date,
			pa.original_contract_sum, pa.retainage_percent, pa.previous_certificates,
			COALESCE(pa.additional_notes, ''),
			pa.calc_net_change_orders, pa.calc_contract_sum_to_date,
			pa.calc_total_completed_stored, pa.calc_retainage_amount,
			pa.calc_earned_less_retainage, pa.calc_current_payment_due, pa.calc_balance_to_finish
		FROM payment_applications pa
		JOIN pa_tenants t ON t.id = pa.tenant_id
		WHERE pa.id = $1`,
		id,
	).Scan(
		&pa.ID, &pa.SubmissionToken, &pa.ApplicationNumber,
		&pa.PeriodTo, &pa.CreatedAt,
		&pa.TenantName, &pa.APEmail,
		&pa.CompanyName, &pa.ContactName, &pa.Email, &pa.Phone,
		&pa.AddressLine1, &pa.AddressLine2,
		&pa.City, &pa.State, &pa.Zip,
		&pa.ProjectName, &pa.ProjectNumber,
		&pa.Owner, &pa.Contractor,
		&pa.ContractDate,
		&pa.OriginalContractSum, &pa.RetainagePercent, &pa.PreviousCertificates,
		&pa.AdditionalNotes,
		&pa.CalcNetChangeOrders, &pa.CalcContractSumToDate,
		&pa.CalcTotalCompleted, &pa.CalcRetainageAmount,
		&pa.CalcEarnedLessRetainage, &pa.CalcCurrentPaymentDue, &pa.CalcBalanceToFinish,
	)
	if err != nil {
		return nil, err
	}

	// Change orders
	coRows, err := DB.Query(`
		SELECT COALESCE(co_number,''), COALESCE(description,''), amount,
		       date_approved, sort_order
		FROM payment_application_change_orders
		WHERE payment_application_id = $1
		ORDER BY sort_order, id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch change orders for pdf: %w", err)
	}
	defer coRows.Close()
	for coRows.Next() {
		var co PDFChangeOrder
		if err := coRows.Scan(&co.CONumber, &co.Description, &co.Amount, &co.DateApproved, &co.SortOrder); err != nil {
			return nil, fmt.Errorf("scan change order for pdf: %w", err)
		}
		pa.ChangeOrders = append(pa.ChangeOrders, co)
	}
	if err := coRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate change orders for pdf: %w", err)
	}

	// Line items
	liRows, err := DB.Query(`
		SELECT COALESCE(item_no,''), COALESCE(description,''),
		       scheduled_value, prev_completed, this_period, materials_stored,
		       calc_total_completed, calc_percent_complete, calc_balance_to_finish,
		       sort_order
		FROM payment_application_line_items
		WHERE payment_application_id = $1
		ORDER BY sort_order, id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch line items for pdf: %w", err)
	}
	defer liRows.Close()
	for liRows.Next() {
		var li PDFLineItem
		if err := liRows.Scan(
			&li.ItemNo, &li.Description,
			&li.ScheduledValue, &li.PrevCompleted, &li.ThisPeriod, &li.MaterialsStored,
			&li.CalcTotalCompleted, &li.CalcPercentComplete, &li.CalcBalanceToFinish,
			&li.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan line item for pdf: %w", err)
		}
		pa.LineItems = append(pa.LineItems, li)
	}
	if err := liRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate line items for pdf: %w", err)
	}

	return &pa, nil
}



// PAStripeInfo holds the minimum fields fetched for a Stripe webhook lookup.
type PAStripeInfo struct {
	ID            int
	PaymentStatus string
}

// GetPaymentApplicationByStripeSession fetches a PAStripeInfo by Stripe checkout session ID.
// Returns sql.ErrNoRows if no matching row is found.
func GetPaymentApplicationByStripeSession(sessionID string) (*PAStripeInfo, error) {
	var info PAStripeInfo
	err := DB.QueryRow(
		`SELECT id, payment_status FROM payment_applications
		 WHERE stripe_checkout_session_id = $1`,
		sessionID,
	).Scan(&info.ID, &info.PaymentStatus)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// UpdatePaymentStatus sets payment_status, paid_at, and stripe_payment_intent_id
// on the payment_applications row with the given id.
func UpdatePaymentStatus(id int, status string, paidAt *time.Time, paymentIntentID string) error {
	_, err := DB.Exec(
		`UPDATE payment_applications
		 SET payment_status = $1, paid_at = $2, stripe_payment_intent_id = $3
		 WHERE id = $4`,
		status, paidAt, paymentIntentID, id,
	)
	return err
}

// UpdateStripeSessionID stores the Stripe checkout session ID on a payment_applications row.
func UpdateStripeSessionID(id int, sessionID string) error {
	_, err := DB.Exec(
		`UPDATE payment_applications SET stripe_checkout_session_id = $1 WHERE id = $2`,
		sessionID, id,
	)
	return err
}

// ─── Ensure sql.ErrNoRows is accessible to callers that import only this package ──
var _ = sql.ErrNoRows
