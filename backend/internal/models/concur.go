package models

import "time"

// ConcurExpense represents a single expense from Concur export
type ConcurExpense struct {
	EmployeeName     string
	ExpenseType      string
	AccountCode      string
	TransactionDate  time.Time
	CostCode         string
	CostClass        string
	JobCode          string
	ClaimedAmountUSD float64
}

// FoundationRow represents a row in the Foundation CSV import
type FoundationRow struct {
	// HEADER fields (columns 0-20)
	SingleMultiple       string  // Column 1: 'N' for single line
	VoucherReferenceID   string  // Column 2
	VendorNo             string  // Column 3: Required
	InvoiceDate          string  // Column 4: Required (M/D/YYYY)
	TransactionDate      string  // Column 5: Required (M/D/YYYY)
	HeaderAmount         float64 // Column 6: Header total
	InvoiceDescription   string  // Column 7
	InvoiceNo            string  // Column 8
	DueDate              string  // Column 9
	RetainageAmount      float64 // Column 10
	POSubNo              string  // Column 11
	VoucherType          string  // Column 12: 'P' for Prepaid
	PaymentType          string  // Column 13: 'CRC' for Credit Card
	CheckNo              string  // Column 14
	CheckAmount          float64 // Column 15
	CheckDate            string  // Column 16 (M/D/YYYY)
	TermsNo              string  // Column 17
	DiscountDate         string  // Column 18
	DiscountAmount       float64 // Column 19
	ScannedInvoiceLink   string  // Column 20

	// DETAIL fields (columns 21-47)
	GLExpense            string  // Column 21: Required - Account Code
	Div1                 string  // Column 22
	Div2                 string  // Column 23
	Div3                 string  // Column 24
	Div4                 string  // Column 25
	JobNo                string  // Column 26
	PhaseNo              string  // Column 27
	CostCodeNo           string  // Column 28
	CostClassNo          string  // Column 29
	DistributionAmount   float64 // Column 30
	Description          string  // Column 31
	EquipmentNo          string  // Column 32
	ServiceCode          string  // Column 33
	Units                float64 // Column 34
	UnitPrice            float64 // Column 35
	OriginalContract     float64 // Column 36
	ChangeOrder          float64 // Column 37
	CommittedCost        float64 // Column 38
	CostToDate           float64 // Column 39
	Revenue              float64 // Column 40
	Overhead             float64 // Column 41
	CashFlow             float64 // Column 42
	Billed               float64 // Column 43
	BillableAmount       float64 // Column 44
	SalesTaxAmount       float64 // Column 45
	CommittedSalesTax    float64 // Column 46
	SalesTaxCode         string  // Column 47
}

// ConversionJob tracks a Concur to Foundation conversion
type ConversionJob struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	Filename       string    `json:"filename"`
	Status         string    `json:"status"` // pending, processing, completed, failed
	VendorID       string    `json:"vendor_id"`
	RowsProcessed  int       `json:"rows_processed"`
	RowsSkipped    int       `json:"rows_skipped"`
	OutputFilePath string    `json:"output_file_path"`
	ErrorLog       string    `json:"error_log"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

// ConversionResult contains the result of a conversion
type ConversionResult struct {
	OutputPath    string   `json:"output_path"`
	RowsProcessed int      `json:"rows_processed"`
	RowsSkipped   int      `json:"rows_skipped"`
	Issues        []string `json:"issues"`
}
