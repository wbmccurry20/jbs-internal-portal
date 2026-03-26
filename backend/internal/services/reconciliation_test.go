package services

import (
	"os"
	"testing"
	"time"

	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
)

func makeBankTx(date time.Time, desc string, amount float64) models.BankTransaction {
	return models.BankTransaction{
		Transaction: models.Transaction{
			Date:        date,
			Description: desc,
			Amount:      amount,
		},
	}
}

func makeFoundationTx(date time.Time, desc string, amount float64) models.FoundationTransaction {
	return models.FoundationTransaction{
		Transaction: models.Transaction{
			Date:        date,
			Description: desc,
			Amount:      amount,
		},
	}
}

func TestReconciliationEngine_ExactMatch(t *testing.T) {
	engine := NewReconciliationEngine(3, 0.01)
	
	bankTrans := []models.BankTransaction{
		makeBankTx(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Payment to ABC Corp", 1000.00),
	}
	
	foundationTrans := []models.FoundationTransaction{
		makeFoundationTx(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Payment to ABC Corp", 1000.00),
	}
	
	report := engine.Reconcile(bankTrans, foundationTrans)
	
	if report.TotalMatched() != 1 {
		t.Errorf("Expected 1 exact match, got %d", report.TotalMatched())
	}
	if report.TotalBankOnly() != 0 {
		t.Errorf("Expected 0 bank-only transactions, got %d", report.TotalBankOnly())
	}
	if report.TotalFoundationOnly() != 0 {
		t.Errorf("Expected 0 foundation-only transactions, got %d", report.TotalFoundationOnly())
	}
}

func TestReconciliationEngine_NoMatches(t *testing.T) {
	engine := NewReconciliationEngine(3, 0.01)
	
	bankTrans := []models.BankTransaction{
		makeBankTx(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Payment to ABC Corp", 1000.00),
	}
	
	foundationTrans := []models.FoundationTransaction{
		makeFoundationTx(time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), "Different payment", 2000.00),
	}
	
	report := engine.Reconcile(bankTrans, foundationTrans)
	
	if report.TotalMatched() != 0 {
		t.Errorf("Expected 0 matches, got %d", report.TotalMatched())
	}
	if report.TotalBankOnly() != 1 {
		t.Errorf("Expected 1 bank-only transaction, got %d", report.TotalBankOnly())
	}
	if report.TotalFoundationOnly() != 1 {
		t.Errorf("Expected 1 foundation-only transaction, got %d", report.TotalFoundationOnly())
	}
}

func TestReconciliationEngine_DateTolerance(t *testing.T) {
	engine := NewReconciliationEngine(3, 0.01)
	
	bankTrans := []models.BankTransaction{
		makeBankTx(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Payment", 1000.00),
	}
	
	foundationTrans := []models.FoundationTransaction{
		makeFoundationTx(time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), "Payment", 1000.00),
	}
	
	report := engine.Reconcile(bankTrans, foundationTrans)
	
	// Should match within 3-day tolerance
	if report.TotalMatched() != 1 {
		t.Errorf("Expected 1 match within tolerance, got %d", report.TotalMatched())
	}
}

// Simple unit test to verify model structure
func TestTransactionModels(t *testing.T) {
	// Test BankTransaction
	bankTx := makeBankTx(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Test", 100.00)
	if bankTx.Amount != 100.00 {
		t.Errorf("Expected amount 100.00, got %f", bankTx.Amount)
	}
	if bankTx.Description != "Test" {
		t.Errorf("Expected description 'Test', got %s", bankTx.Description)
	}
	
	// Test FoundationTransaction
	foundationTx := makeFoundationTx(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Test Foundation", 200.00)
	if foundationTx.Amount != 200.00 {
		t.Errorf("Expected amount 200.00, got %f", foundationTx.Amount)
	}
	if foundationTx.Description != "Test Foundation" {
		t.Errorf("Expected description 'Test Foundation', got %s", foundationTx.Description)
	}
}

func TestLoadBankTransactions_XLS(t *testing.T) {
	// Test loading the actual AmEx XLS file (if present in the project root)
	xlsPath := "../../../docs/data/reconciliation-samples/Statement_1002_Feb_2026 (2).xls"
	if _, err := os.Stat(xlsPath); os.IsNotExist(err) {
		t.Skip("AmEx XLS test file not present, skipping")
	}

	txns, err := LoadBankTransactions(xlsPath)
	if err != nil {
		t.Fatalf("Failed to load bank transactions from XLS: %v", err)
	}

	// The file has 912 Corporate Card rows per NodeJS conversion
	if len(txns) < 900 {
		t.Errorf("Expected at least 900 bank transactions, got %d (possible parsing issue)", len(txns))
	}

	// Verify first transaction has expected data
	if len(txns) > 0 {
		first := txns[0]
		if first.Date.IsZero() {
			t.Error("First transaction has zero date")
		}
		if first.Amount == 0 {
			t.Error("First transaction has zero amount")
		}
		if first.Description == "" {
			t.Error("First transaction has empty description")
		}
		// AMEX dates should be in February 2026
		if first.Date.Month() != time.February || first.Date.Year() != 2026 {
			t.Errorf("Expected February 2026 date, got %v", first.Date)
		}
	}

	t.Logf("Successfully loaded %d bank transactions from XLS", len(txns))
}

func TestLoadFoundationTransactions_XLSX(t *testing.T) {
	// Test loading the actual Concur XLSX file (if present in the project root)
	xlsxPath := "../../../docs/data/reconciliation-samples/Concur Report for Recon.xlsx"
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("Concur XLSX test file not present, skipping")
	}

	txns, err := LoadFoundationTransactions(xlsxPath)
	if err != nil {
		t.Fatalf("Failed to load foundation transactions from XLSX: %v", err)
	}

	if len(txns) < 100 {
		t.Errorf("Expected at least 100 foundation transactions, got %d", len(txns))
	}

	// Verify transactions have dates and amounts
	for i, txn := range txns[:minInt(5, len(txns))] {
		if txn.Date.IsZero() {
			t.Errorf("Transaction %d has zero date", i)
		}
		if txn.Amount == 0 {
			t.Errorf("Transaction %d has zero amount", i)
		}
	}

	t.Logf("Successfully loaded %d foundation transactions from XLSX", len(txns))
}

func TestParseDate_ExcelSerial(t *testing.T) {
	// Feb 28, 2026 = Excel serial 46082
	result, err := parseDate("46082")
	if err != nil {
		t.Fatalf("Failed to parse Excel serial date: %v", err)
	}
	if result.Year() != 2026 || result.Month() != time.February || result.Day() != 28 {
		t.Errorf("Expected 2026-02-28, got %v", result)
	}
}

func TestParseDate_ConcurFormat(t *testing.T) {
	// Concur exports dates like "2/26/26 0:00"
	result, err := parseDate("2/26/26 0:00")
	if err != nil {
		t.Fatalf("Failed to parse Concur date: %v", err)
	}
	if result.Year() != 2026 || result.Month() != time.February || result.Day() != 26 {
		t.Errorf("Expected 2026-02-26, got %v", result)
	}
}

func TestBillingCycleScoring(t *testing.T) {
	engine := NewReconciliationEngine(3, 0.01)

	// AmEx settlement date (end of billing cycle)
	bankTxn := makeBankTx(time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC), "PURCHASE", 100.00)
	// Concur entry posted mid-cycle (14 days before settlement)
	foundTxn := makeFoundationTx(time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC), "PURCHASE", 100.00)

	score := engine.calculateMatchScore(bankTxn, foundTxn)

	// With exact amount (0.5) and 14-day diff within billing cycle (0.12), score should be 0.62
	if score < 0.6 {
		t.Errorf("Expected billing-cycle match score >= 0.6, got %.2f", score)
	}
	t.Logf("Billing-cycle match score: %.2f", score)
}

func TestEndToEnd_Reconciliation(t *testing.T) {
	// Test full reconciliation with the actual files (if present)
	bankPath := "../../../docs/data/reconciliation-samples/Statement_1002_Feb_2026 (2).xls"
	foundPath := "../../../docs/data/reconciliation-samples/Concur Report for Recon.xlsx"

	if _, err := os.Stat(bankPath); os.IsNotExist(err) {
		t.Skip("Test files not present, skipping end-to-end test")
	}
	if _, err := os.Stat(foundPath); os.IsNotExist(err) {
		t.Skip("Test files not present, skipping end-to-end test")
	}

	bankTxns, err := LoadBankTransactions(bankPath)
	if err != nil {
		t.Fatalf("Failed to load bank transactions: %v", err)
	}

	foundTxns, err := LoadFoundationTransactions(foundPath)
	if err != nil {
		t.Fatalf("Failed to load foundation transactions: %v", err)
	}

	engine := NewReconciliationEngine(3, 0.01)
	report := engine.Reconcile(bankTxns, foundTxns)

	t.Logf("=== Reconciliation Results ===")
	t.Logf("Bank transactions loaded:       %d", len(bankTxns))
	t.Logf("Foundation transactions loaded:  %d", len(foundTxns))
	t.Logf("Matched:                         %d", report.TotalMatched())
	t.Logf("Potential matches:               %d", len(report.PotentialMatches))
	t.Logf("Bank-only:                       %d", report.TotalBankOnly())
	t.Logf("Foundation-only:                 %d", report.TotalFoundationOnly())
	t.Logf("Void pairs:                      %d", len(report.VoidPairs))
	t.Logf("Ambiguous voids:                 %d", len(report.AmbiguousVoids))
	t.Logf("Match rate (actionable):         %.1f%%", report.ActionableMatchRate())

	// We should have SOME matches if both files loaded correctly
	if report.TotalMatched() == 0 && len(report.PotentialMatches) == 0 {
		t.Error("Expected at least some matches or potential matches — possible parsing issue")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
