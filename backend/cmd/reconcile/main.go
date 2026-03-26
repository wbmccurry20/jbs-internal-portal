// cmd/reconcile/main.go — standalone CLI for reconciling bank vs Concur/Foundation files.
// Usage: go run ./cmd/reconcile -bank <bank_file> -foundation <foundation_file> [-out <output.xlsx>] [-days <tolerance_days>]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wbmccurry20/jbs-internal-portal/internal/services"
)

func main() {
	bankFile := flag.String("bank", "", "Path to bank statement file (.xls, .xlsx, or .csv)")
	foundationFile := flag.String("foundation", "", "Path to foundation/Concur file (.xlsx, .csv, or .xls)")
	outFile := flag.String("out", "", "Output Excel report path (default: reconciliation_<timestamp>.xlsx in current dir)")
	toleranceDays := flag.Int("days", 3, "Date tolerance in days for matching (default: 3)")
	flag.Parse()

	if *bankFile == "" || *foundationFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(*bankFile); err != nil {
		log.Fatalf("Bank file not found: %s", *bankFile)
	}
	if _, err := os.Stat(*foundationFile); err != nil {
		log.Fatalf("Foundation/Concur file not found: %s", *foundationFile)
	}

	if *outFile == "" {
		timestamp := time.Now().Format("20060102_150405")
		*outFile = fmt.Sprintf("reconciliation_%s.xlsx", timestamp)
	}
	// Ensure output directory exists
	if dir := filepath.Dir(*outFile); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	fmt.Printf("Loading bank transactions from: %s\n", *bankFile)
	bankTrans, err := services.LoadBankTransactions(*bankFile)
	if err != nil {
		log.Fatalf("Failed to load bank transactions: %v", err)
	}
	fmt.Printf("  Loaded %d bank transactions\n", len(bankTrans))

	fmt.Printf("Loading foundation/Concur transactions from: %s\n", *foundationFile)
	foundationTrans, err := services.LoadFoundationTransactions(*foundationFile)
	if err != nil {
		log.Fatalf("Failed to load foundation transactions: %v", err)
	}
	fmt.Printf("  Loaded %d foundation transactions\n", len(foundationTrans))

	fmt.Printf("Running reconciliation (tolerance: %d day(s))...\n", *toleranceDays)
	engine := services.NewReconciliationEngine(*toleranceDays, 0.01)
	report := engine.Reconcile(bankTrans, foundationTrans)

	fmt.Printf("\nDate ranges:\n")
	if report.BankMinDate != nil {
		fmt.Printf("  Bank statement:    %s  →  %s\n",
			report.BankMinDate.Format("Jan 2, 2006"), report.BankMaxDate.Format("Jan 2, 2006"))
	}
	if report.FoundationMinDate != nil {
		fmt.Printf("  Concur/Foundation: %s  →  %s\n",
			report.FoundationMinDate.Format("Jan 2, 2006"), report.FoundationMaxDate.Format("Jan 2, 2006"))
	}
	if report.OverlapStart != nil {
		fmt.Printf("  Reconcile window:  %s  →  %s\n",
			report.OverlapStart.Format("Jan 2, 2006"), report.OverlapEnd.Format("Jan 2, 2006"))
	}

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Matched:                         %d\n", report.TotalMatched())
	fmt.Printf("  Potential matches:               %d\n", len(report.PotentialMatches))
	fmt.Printf("  Bank payments/credits (skipped): %d\n", len(report.BankPayments))
	fmt.Printf("  Bank out-of-range:               %d\n", len(report.BankOutOfRange))
	fmt.Printf("  Bank true discrepancies:         %d\n", len(report.BankTrueDiscrepancies))
	fmt.Printf("  Concur out-of-range:             %d\n", len(report.FoundationOutOfRange))
	fmt.Printf("  Concur true discrepancies:       %d\n", len(report.FoundationTrueDiscrepancies))
	fmt.Printf("  Void pairs:                      %d\n", len(report.VoidPairs))
	fmt.Printf("  Match rate (actionable):         %s\n", fmt.Sprintf("%.1f%%", report.ActionableMatchRate()))

	fmt.Printf("\nGenerating Excel report: %s\n", *outFile)
	generator := services.NewExcelReportGenerator()
	if err := generator.GenerateReconciliationExcel(report, *outFile); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	absOut, _ := filepath.Abs(*outFile)
	fmt.Printf("\nDone! Report saved to:\n  %s\n", absOut)
}
