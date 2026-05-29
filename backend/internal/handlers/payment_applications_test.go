package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── calculateTotals unit tests (no DB, no HTTP) ──────────────────────────────

func TestCalculateTotals_BasicCalculation(t *testing.T) {
	req := &CreatePaymentApplicationRequest{
		OriginalContractSum:  100000.00,
		RetainagePercent:     10.0,
		PreviousCertificates: 20000.00,
		ChangeOrders: []ChangeOrderInput{
			{Amount: 5000.00},
			{Amount: -2000.00}, // credit change order
		},
		LineItems: []LineItemInput{
			{ScheduledValue: 50000, PrevCompleted: 20000, ThisPeriod: 10000, MaterialsStored: 0},
			{ScheduledValue: 50000, PrevCompleted: 5000, ThisPeriod: 5000, MaterialsStored: 2000},
		},
	}

	totals := calculateTotals(req)

	// net CO: 5000 + (-2000) = 3000
	assert.Equal(t, 3000.00, totals.NetChangeOrders)
	// contract sum to date: 100000 + 3000 = 103000
	assert.Equal(t, 103000.00, totals.ContractSumToDate)
	// total completed: (20000+10000+0) + (5000+5000+2000) = 42000
	assert.Equal(t, 42000.00, totals.TotalCompletedStored)
	// retainage: 42000 * 10% = 4200
	assert.InDelta(t, 4200.00, totals.RetainageAmount, 0.01)
	// earned less retainage: 42000 - 4200 = 37800
	assert.InDelta(t, 37800.00, totals.EarnedLessRetainage, 0.01)
	// current payment due: 37800 - 20000 (prev certs) = 17800
	assert.InDelta(t, 17800.00, totals.CurrentPaymentDue, 0.01)
	// balance to finish: 103000 - 42000 = 61000
	assert.InDelta(t, 61000.00, totals.BalanceToFinish, 0.01)
}

func TestCalculateTotals_NoChangeOrdersNoLineItems(t *testing.T) {
	req := &CreatePaymentApplicationRequest{
		OriginalContractSum:  50000.00,
		RetainagePercent:     5.0,
		PreviousCertificates: 0,
		ChangeOrders:         []ChangeOrderInput{},
		LineItems:            []LineItemInput{},
	}

	totals := calculateTotals(req)

	assert.Equal(t, 0.00, totals.NetChangeOrders)
	assert.Equal(t, 50000.00, totals.ContractSumToDate)
	assert.Equal(t, 0.00, totals.TotalCompletedStored)
	assert.Equal(t, 0.00, totals.RetainageAmount)
	assert.Equal(t, 0.00, totals.EarnedLessRetainage)
	assert.Equal(t, 0.00, totals.CurrentPaymentDue)
	assert.Equal(t, 50000.00, totals.BalanceToFinish)
}

func TestCalculateTotals_ZeroRetainage(t *testing.T) {
	req := &CreatePaymentApplicationRequest{
		OriginalContractSum:  10000.00,
		RetainagePercent:     0.0,
		PreviousCertificates: 0,
		LineItems: []LineItemInput{
			{ScheduledValue: 10000, PrevCompleted: 10000, ThisPeriod: 0, MaterialsStored: 0},
		},
	}

	totals := calculateTotals(req)

	assert.Equal(t, 0.00, totals.RetainageAmount)
	assert.Equal(t, 10000.00, totals.EarnedLessRetainage)
	assert.Equal(t, 10000.00, totals.CurrentPaymentDue)
	assert.Equal(t, 0.00, totals.BalanceToFinish)
}

// ─── calcLineItemValues unit tests (no DB, no HTTP) ───────────────────────────

func TestCalcLineItemValues_TypicalRow(t *testing.T) {
	li := LineItemInput{
		ScheduledValue:  20000,
		PrevCompleted:   8000,
		ThisPeriod:      4000,
		MaterialsStored: 1000,
	}
	ct, cp, cb, cr := calcLineItemValues(li, 10.0)

	assert.Equal(t, 13000.00, ct)        // 8000+4000+1000
	assert.InDelta(t, 65.0, cp, 0.01)    // 13000/20000*100
	assert.Equal(t, 7000.00, cb)         // 20000-13000
	assert.InDelta(t, 1300.00, cr, 0.01) // 13000*10%
}

func TestCalcLineItemValues_ZeroScheduledValue(t *testing.T) {
	li := LineItemInput{ScheduledValue: 0, PrevCompleted: 0, ThisPeriod: 0, MaterialsStored: 0}
	_, cp, _, _ := calcLineItemValues(li, 10.0)
	// No division by zero — percent should be 0
	assert.Equal(t, 0.00, cp)
}

// ─── parseDate unit tests (no DB, no HTTP) ────────────────────────────────────

func TestParseDate_ValidDate(t *testing.T) {
	result, err := parseDate("2026-05-29")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, 5, int(result.Month()))
	assert.Equal(t, 29, result.Day())
}

func TestParseDate_EmptyString(t *testing.T) {
	result, err := parseDate("")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseDate_InvalidFormat(t *testing.T) {
	result, err := parseDate("not-a-date")
	assert.Error(t, err)
	assert.Nil(t, result)

	result2, err2 := parseDate("2026/05/29") // wrong separator
	assert.Error(t, err2)
	assert.Nil(t, result2)
}

// ─── POST /api/v1/payment-applications binding validation tests ───────────────
// These tests exercise the ShouldBindJSON validation path and return before any
// database call, so no test database is required.

func newPARouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/payment-applications", CreatePaymentApplication)
	r.GET("/api/v1/payment-applications/:submissionToken", GetPaymentApplication)
	return r
}

// postPA sends a POST to /api/v1/payment-applications with the given raw JSON body string.
func postPA(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest("POST", "/api/v1/payment-applications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newPARouter().ServeHTTP(w, req)
	return w
}

func TestCreatePA_MissingCompanyName(t *testing.T) {
	w := postPA(t, `{
		"contact_name":"John Doe","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_MissingContactName(t *testing.T) {
	w := postPA(t, `{
		"company_name":"ACME","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_MissingProjectName(t *testing.T) {
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"application_number":1,"original_contract_sum":100000
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_InvalidEmail(t *testing.T) {
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"not-an-email",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_ApplicationNumberZero(t *testing.T) {
	// application_number must be >= 1
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":0,"original_contract_sum":100000
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_NegativeContractSum(t *testing.T) {
	// original_contract_sum must be > 0
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":-500
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_RetainagePercentTooHigh(t *testing.T) {
	// retainage_percent must be <= 50
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000,
		"retainage_percent":75
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_StateTooLong(t *testing.T) {
	// state must be at most 2 characters — validated in handler body after binding
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000,
		"state":"CAL"
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GET /api/v1/payment-applications/:token token format tests ───────────────

func TestGetPA_TokenTooShort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req, _ := http.NewRequest("GET", "/api/v1/payment-applications/shorttokenvalue", nil)
	w := httptest.NewRecorder()
	newPARouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPA_TokenExactlyWrongLength(t *testing.T) {
	// 63 chars — one short of valid 64-char hex token
	token := strings.Repeat("a", 63)
	req, _ := http.NewRequest("GET", "/api/v1/payment-applications/"+token, nil)
	w := httptest.NewRecorder()
	newPARouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Token hex validation tests ───────────────────────────────────────────────

func TestGetPA_NonHexToken64Chars(t *testing.T) {
	// 64 characters but 'g' is not valid hex (hex is 0-9, a-f, A-F)
	token := strings.Repeat("g", 64)
	req, _ := http.NewRequest("GET", "/api/v1/payment-applications/"+token, nil)
	w := httptest.NewRecorder()
	newPARouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPA_UppercaseNonHexToken64Chars(t *testing.T) {
	// 64 uppercase letters beyond F — not valid hex
	token := strings.Repeat("G", 64)
	req, _ := http.NewRequest("GET", "/api/v1/payment-applications/"+token, nil)
	w := httptest.NewRecorder()
	newPARouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPA_ValidHexToken_PassesFormatValidation(t *testing.T) {
	// 64 lowercase hex chars — passes isValidSubmissionToken.
	// DB is nil in test env so gin.Default() recovery returns 500, not 400.
	// This test verifies that a valid hex token is NOT rejected by format validation.
	gin.SetMode(gin.TestMode)
	r := gin.Default() // recovery middleware prevents nil-DB panic
	r.GET("/api/v1/payment-applications/:submissionToken", GetPaymentApplication)

	token := strings.Repeat("a", 64) // all lowercase 'a' — valid hex
	req, _ := http.NewRequest("GET", "/api/v1/payment-applications/"+token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Must NOT return 400; format is valid. Expect 500 due to nil DB in test env.
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

// ─── Date validation tests ────────────────────────────────────────────────────

func TestCreatePA_InvalidContractDate(t *testing.T) {
	// Wrong date separator — must return 400 before DB call
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000,
		"contract_date":"2026/05/29"
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_InvalidPeriodTo(t *testing.T) {
	// Human-readable date format — must return 400 before DB call
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000,
		"period_to":"May 2026"
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_InvalidCODateApproved(t *testing.T) {
	// Change order with wrong date format — must return 400 before DB call
	w := postPA(t, `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000,
		"change_orders":[{"co_number":"CO-001","description":"Extra work","amount":5000,"date_approved":"05/29/2026"}]
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_ValidDatesAllowed(t *testing.T) {
	// Valid YYYY-MM-DD dates must pass date validation (will fail later at DB call).
	// Uses gin.Default() recovery so nil-DB panic becomes 500, not a test crash.
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/payment-applications", CreatePaymentApplication)

	w := httptest.NewRecorder()
	body := `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000,
		"contract_date":"2026-05-01","period_to":"2026-05-31"
	}`
	req, _ := http.NewRequest("POST", "/api/v1/payment-applications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Must NOT return 400 — dates are valid. Expect 500 from nil DB.
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

func TestCreatePA_EmptyDatesAllowed(t *testing.T) {
	// Omitting date fields entirely must be accepted (dates are optional).
	// Uses gin.Default() recovery so nil-DB panic becomes 500, not a test crash.
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/payment-applications", CreatePaymentApplication)

	w := httptest.NewRecorder()
	body := `{
		"company_name":"ACME","contact_name":"John","email":"john@example.com",
		"project_name":"Test Project","application_number":1,"original_contract_sum":100000
	}`
	req, _ := http.NewRequest("POST", "/api/v1/payment-applications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Must NOT return 400 — no dates is perfectly valid.
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}
