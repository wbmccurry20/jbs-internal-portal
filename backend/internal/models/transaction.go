package models

import "time"

// Transaction base model
type Transaction struct {
	Date        time.Time
	Amount      float64
	Description string
	Reference   string
}

// BankTransaction represents a bank account transaction (AmEx)
type BankTransaction struct {
	Transaction
	CardmemberName  string
	CardAccount     string
	AccountNumber   string
	TransactionType string // debit, credit
	Balance         float64
}

// FoundationTransaction represents a Foundation accounting system transaction
type FoundationTransaction struct {
	Transaction
	VendorID          string
	VendorName        string
	TransactionNumber int
	JobNumber         string
	AccountCode       string
	JobCode           string
	CostCode          string
	CostClass         string
	EmployeeName      string
}
