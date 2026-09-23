package pdf_test

import (
	"testing"
	"time"

	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
	"github.com/wbmccurry20/jbs-internal-portal/internal/pdf"
)

func mockPDFData() *database.PaymentApplicationPDFData {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	periodTo := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	contractDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	coDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	return &database.PaymentApplicationPDFData{
		ID:                1,
		SubmissionToken:   "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
		ApplicationNumber: 3,
		PeriodTo:          &periodTo,
		CreatedAt:         now,

		TenantName: "JBS Construction",
		APEmail:    "info@jbsconstructiongroup.com",

		CompanyName:  "Acme Electrical LLC",
		ContactName:  "Jane Smith",
		Email:        "jane@acme-elec.com",
		Phone:        "(555) 867-5309",
		AddressLine1: "123 Main Street",
		City:         "Nashville",
		State:        "TN",
		Zip:          "37201",

		ProjectName:   "Retail Renovation — Store #42",
		ProjectNumber: "PRJ-2026-042",
		Owner:         "TN Retail Holdings LLC",
		Contractor:    "JBS Construction",
		ContractDate:  &contractDate,

		OriginalContractSum:  250000.00,
		RetainagePercent:     10.0,
		PreviousCertificates: 30000.00,
		AdditionalNotes:      "Work completed per approved drawings Rev B.",

		CalcNetChangeOrders:     15000.00,
		CalcContractSumToDate:   265000.00,
		CalcTotalCompleted:      90000.00,
		CalcRetainageAmount:     9000.00,
		CalcEarnedLessRetainage: 81000.00,
		CalcCurrentPaymentDue:   51000.00,
		CalcBalanceToFinish:     175000.00,

		ChangeOrders: []database.PDFChangeOrder{
			{CONumber: "CO-01", Description: "Additional conduit run to panel B", Amount: 8500.00, DateApproved: &coDate},
			{CONumber: "CO-02", Description: "LED fixture upgrade per owner request", Amount: 6500.00, DateApproved: &coDate},
		},
		LineItems: []database.PDFLineItem{
			{ItemNo: "01", Description: "Mobilization & temp power", ScheduledValue: 10000, PrevCompleted: 10000, CalcTotalCompleted: 10000, CalcPercentComplete: 100.0, CalcBalanceToFinish: 0},
			{ItemNo: "02", Description: "Rough-in wiring", ScheduledValue: 80000, PrevCompleted: 20000, ThisPeriod: 30000, CalcTotalCompleted: 50000, CalcPercentComplete: 62.5, CalcBalanceToFinish: 30000},
			{ItemNo: "03", Description: "Panel installation", ScheduledValue: 45000, CalcBalanceToFinish: 45000},
			{ItemNo: "04", Description: "Fixture installation", ScheduledValue: 60000, CalcBalanceToFinish: 60000},
		},
	}
}

func TestGeneratePDF_ReturnsNonEmptyBytes(t *testing.T) {
	data := mockPDFData()
	got, err := pdf.GeneratePDF(data)
	if err != nil {
		t.Fatalf("GeneratePDF returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("GeneratePDF returned empty bytes")
	}
	// A valid PDF starts with the %PDF header
	if len(got) < 4 || string(got[:4]) != "%PDF" {
		t.Fatalf("GeneratePDF output does not start with %%PDF: got %q", string(got[:min(20, len(got))]))
	}
}

func TestGeneratePDF_NoChangeOrders(t *testing.T) {
	data := mockPDFData()
	data.ChangeOrders = nil
	got, err := pdf.GeneratePDF(data)
	if err != nil {
		t.Fatalf("GeneratePDF (no COs) returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("GeneratePDF (no COs) returned empty bytes")
	}
}

func TestGeneratePDF_NoLineItems(t *testing.T) {
	data := mockPDFData()
	data.LineItems = nil
	got, err := pdf.GeneratePDF(data)
	if err != nil {
		t.Fatalf("GeneratePDF (no line items) returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("GeneratePDF (no line items) returned empty bytes")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
