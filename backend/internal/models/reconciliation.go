package models

import "time"

// ReconciliationReport contains the results of a reconciliation
type ReconciliationReport struct {
	MatchedTransactions         []MatchedPair
	BankOnlyTransactions        []BankTransaction
	FoundationOnlyTransactions  []FoundationTransaction
	PotentialMatches            []PotentialMatch
	VoidPairs                   []VoidPair
	AmbiguousVoids              []AmbiguousVoid
	GeneratedAt                 time.Time
}

// MatchedPair represents a matched bank and foundation transaction
type MatchedPair struct {
	BankTransaction       BankTransaction
	FoundationTransaction FoundationTransaction
	DateDifference        int
	AmountDifference      float64
}

// PotentialMatch represents a possible match that needs review
type PotentialMatch struct {
	BankTransaction       BankTransaction
	FoundationTransaction FoundationTransaction
	Score                 float64 // 0-1, higher is better match
}

// VoidPair represents a void/reversal transaction pair
type VoidPair struct {
	Transaction1 FoundationTransaction
	Transaction2 FoundationTransaction
}

// AmbiguousVoid represents a situation with multiple transactions where void pairing is unclear
type AmbiguousVoid struct {
	Amount             float64
	Date               time.Time
	TransactionCount   int
	TransactionNumbers []string
	Transactions       []FoundationTransaction
	PositiveCount      int
	NegativeCount      int
}

// ReconciliationJob tracks a reconciliation operation
type ReconciliationJob struct {
	ID                    int        `json:"id"`
	UserID                int        `json:"user_id"`
	BankFilename          string     `json:"bank_filename"`
	FoundationFilename    string     `json:"foundation_filename"`
	Status                string     `json:"status"` // pending, processing, completed, failed
	ToleranceDays         int        `json:"tolerance_days"`
	MatchedCount          int        `json:"matched_count"`
	VoidCount             int        `json:"void_count"`
	AmbiguousVoidCount    int        `json:"ambiguous_void_count"`
	OutputFilePath        string     `json:"output_file_path"`
	ErrorLog              string     `json:"error_log"`
	CreatedAt             time.Time  `json:"created_at"`
	CompletedAt           *time.Time `json:"completed_at"`
}

// Summary methods for ReconciliationReport
func (r *ReconciliationReport) TotalMatched() int {
	return len(r.MatchedTransactions)
}

func (r *ReconciliationReport) TotalBankOnly() int {
	return len(r.BankOnlyTransactions)
}

func (r *ReconciliationReport) TotalFoundationOnly() int {
	return len(r.FoundationOnlyTransactions)
}

func (r *ReconciliationReport) TotalDiscrepancies() int {
	return r.TotalBankOnly() + r.TotalFoundationOnly()
}

func (r *ReconciliationReport) MatchRate() float64 {
	total := r.TotalMatched() + r.TotalDiscrepancies()
	if total == 0 {
		return 0.0
	}
	return (float64(r.TotalMatched()) / float64(total)) * 100.0
}
