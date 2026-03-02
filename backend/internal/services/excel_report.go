package services

import (
	"fmt"
	"math"
	"strings"

	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
	"github.com/xuri/excelize/v2"
)

type ExcelReportGenerator struct{}

func NewExcelReportGenerator() *ExcelReportGenerator {
	return &ExcelReportGenerator{}
}

// GenerateReconciliationExcel creates a comprehensive Excel report
func (g *ExcelReportGenerator) GenerateReconciliationExcel(report *models.ReconciliationReport, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Create format styles
	formats := g.createFormats(f)

	// Create sheets in priority order
	g.createActionItemsSheet(f, report, formats)
	g.createSummarySheet(f, report, formats)
	g.createMatchedSheet(f, report, formats)
	g.createDiscrepanciesSheet(f, report, formats)
	g.createPotentialMatchesSheet(f, report, formats)
	g.createVoidsSheet(f, report, formats)
	g.createAmbiguousVoidsSheet(f, report, formats)

	// Delete default Sheet1
	f.DeleteSheet("Sheet1")

	// Save file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

func (g *ExcelReportGenerator) createFormats(f *excelize.File) map[string]int {
	formats := make(map[string]int)

	// Header format
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4472C4"}},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	formats["header"] = headerStyle

	// Matched (green)
	matchedStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C6EFCE"}},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["matched"] = matchedStyle

	// Bank only (red)
	bankOnlyStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFC7CE"}},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["bank_only"] = bankOnlyStyle

	// Foundation only (yellow)
	foundationOnlyStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFEB9C"}},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["foundation_only"] = foundationOnlyStyle

	// Potential (gray)
	potentialStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E7E6E6"}},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["potential"] = potentialStyle

	// Action required (light orange)
	actionStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF2CC"}},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["action_required"] = actionStyle

	// Currency format
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4, // $#,##0.00
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["currency"] = currencyStyle

	// Date format
	dateStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 14, // mm/dd/yyyy
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	formats["date"] = dateStyle

	return formats
}

func (g *ExcelReportGenerator) createActionItemsSheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "⚠️ ACTION ITEMS"
	f.NewSheet(sheetName)

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "B", 12)
	f.SetColWidth(sheetName, "C", "C", 12)
	f.SetColWidth(sheetName, "D", "D", 12)
	f.SetColWidth(sheetName, "E", "E", 35)
	f.SetColWidth(sheetName, "F", "F", 25)
	f.SetColWidth(sheetName, "G", "G", 60)
	f.SetColWidth(sheetName, "H", "H", 10)
	f.SetColWidth(sheetName, "I", "I", 12)
	f.SetColWidth(sheetName, "J", "J", 20)
	f.SetColWidth(sheetName, "K", "K", 40)

	row := 1

	// Instructions header
	f.MergeCell(sheetName, "A1", "K1")
	f.SetCellValue(sheetName, "A1", "👉 INSTRUCTIONS: Review each item below. Mark STATUS as ✓ when completed. Focus on CRITICAL/HIGH priority items first.")
	f.SetCellStyle(sheetName, "A1", "K1", formats["action_required"])
	row = 3

	// Column headers
	headers := []string{"STATUS", "PRIORITY", "AMOUNT", "DATE", "TYPE", "CARDMEMBER/VENDOR", "DESCRIPTION", "TRX#", "JOB#", "REFERENCE", "NOTES"}
	for col, header := range headers {
		cell := fmt.Sprintf("%s%d", string(rune('A'+col)), row)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}
	row++

	// Collect action items
	type actionItem struct {
		priority     string
		prioritySort int
		amount       float64
		date         string
		itemType     string
		cardmember   string
		description  string
		trxNum       string
		jobNum       string
		reference    string
		notes        string
		format       int
	}

	items := make([]actionItem, 0)

	// CRITICAL: Ambiguous voids
	for _, situation := range report.AmbiguousVoids {
		items = append(items, actionItem{
			priority:     "🚨 CRITICAL",
			prioritySort: 0,
			amount:       situation.Amount,
			date:         situation.Date.Format("01/02/2006"),
			itemType:     fmt.Sprintf("⚠️ AMBIGUOUS VOID - %d identical transactions", situation.TransactionCount),
			description:  "Multiple transactions with same amount/date where some void out",
			trxNum:       strings.Join(situation.TransactionNumbers, ", "),
			notes:        "MANUAL REVIEW REQUIRED: Verify which transactions net to zero and which are actual charges. See 'Ambiguous Voids' tab for details.",
			format:       formats["action_required"],
		})
	}

	// HIGH: True bank discrepancies (in overlapping date range, not payments)
	for _, txn := range report.BankTrueDiscrepancies {
		items = append(items, actionItem{
			priority:     "HIGH",
			prioritySort: 1,
			amount:       txn.Amount,
			date:         txn.Date.Format("01/02/2006"),
			itemType:     "🏦 BANK ONLY - Not in Foundation",
			cardmember:   txn.CardmemberName,
			description:  txn.Description,
			reference:    txn.Reference,
			notes:        "Card charge in bank statement but not found in Foundation/Concur",
			format:       formats["bank_only"],
		})
	}

	// MEDIUM/HIGH: True foundation discrepancies (in overlapping date range)
	for _, txn := range report.FoundationTrueDiscrepancies {
		isPrepaid := g.isPrepaidCardTransaction(txn)
		priority := "MEDIUM"
		prioritySort := 2
		itemType := "📊 FOUNDATION ONLY - Not in Bank"
		notes := "In Foundation but not in bank statement for this period"

		if isPrepaid {
			itemType = "💳 PRE-PAID CARD - Add merchant details"
			notes = "ConCur pre-paid transaction needs merchant info"
		} else if math.Abs(txn.Amount) > 1000 {
			priority = "HIGH"
			prioritySort = 1
		}

		items = append(items, actionItem{
			priority:     priority,
			prioritySort: prioritySort,
			amount:       txn.Amount,
			date:         txn.Date.Format("01/02/2006"),
			itemType:     itemType,
			cardmember:   txn.VendorName,
			description:  txn.Description,
			trxNum:       fmt.Sprintf("%d", txn.TransactionNumber),
			jobNum:       txn.JobNumber,
			notes:        notes,
			format:       formats["foundation_only"],
		})
	}

	// REVIEW: Potential matches
	for _, pm := range report.PotentialMatches {
		isPrepaid := g.isPrepaidCardTransaction(pm.FoundationTransaction)
		notes := "Review if these transactions are the same"
		if isPrepaid {
			notes = "Pre-paid card - may match when merchant details added"
		}

		items = append(items, actionItem{
			priority:     "REVIEW",
			prioritySort: 3,
			amount:       pm.BankTransaction.Amount,
			date:         pm.BankTransaction.Date.Format("01/02/2006"),
			itemType:     fmt.Sprintf("🔍 POTENTIAL MATCH (%.0f%%) - Verify", pm.Score*100),
			cardmember:   pm.BankTransaction.CardmemberName,
			description:  pm.BankTransaction.Description,
			trxNum:       fmt.Sprintf("%d", pm.FoundationTransaction.TransactionNumber),
			jobNum:       pm.FoundationTransaction.JobNumber,
			reference:    pm.BankTransaction.Reference,
			notes:        notes,
			format:       formats["potential"],
		})
	}

	// INFO: Payments to AmEx
	for _, txn := range report.BankPayments {
		items = append(items, actionItem{
			priority:     "INFO",
			prioritySort: 4,
			amount:       txn.Amount,
			date:         txn.Date.Format("01/02/2006"),
			itemType:     "💳 PAYMENT TO AMEX",
			cardmember:   txn.CardmemberName,
			description:  txn.Description,
			reference:    txn.Reference,
			notes:        "Payment/credit to card account — not a purchase, no Foundation match expected",
			format:       formats["potential"],
		})
	}

	// INFO: Out of date range (bank)
	for _, txn := range report.BankOutOfRange {
		items = append(items, actionItem{
			priority:     "INFO",
			prioritySort: 5,
			amount:       txn.Amount,
			date:         txn.Date.Format("01/02/2006"),
			itemType:     "📅 OUTSIDE DATE RANGE",
			cardmember:   txn.CardmemberName,
			description:  txn.Description,
			reference:    txn.Reference,
			notes:        "Bank transaction outside Foundation date range — check prior/next period",
			format:       formats["potential"],
		})
	}

	// INFO: Out of date range (foundation)
	for _, txn := range report.FoundationOutOfRange {
		items = append(items, actionItem{
			priority:     "INFO",
			prioritySort: 5,
			amount:       txn.Amount,
			date:         txn.Date.Format("01/02/2006"),
			itemType:     "📅 OUTSIDE DATE RANGE",
			cardmember:   txn.VendorName,
			description:  txn.Description,
			trxNum:       fmt.Sprintf("%d", txn.TransactionNumber),
			jobNum:       txn.JobNumber,
			notes:        "Foundation transaction outside bank statement range — check prior/next period",
			format:       formats["potential"],
		})
	}

	// Sort items: by priority, then by amount (descending)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].prioritySort > items[j].prioritySort ||
				(items[i].prioritySort == items[j].prioritySort && math.Abs(items[i].amount) < math.Abs(items[j].amount)) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Write data
	for _, item := range items {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "☐")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.priority)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), item.amount)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), item.date)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), item.itemType)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.cardmember)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), truncate(item.description, 100))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), item.trxNum)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), item.jobNum)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), item.reference)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), item.notes)

		// Apply formats
		for col := 'B'; col <= 'K'; col++ {
			cell := fmt.Sprintf("%s%d", string(col), row)
			f.SetCellStyle(sheetName, cell, cell, item.format)
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), formats["currency"])

		row++
	}

	// Add summary
	row += 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "SUMMARY:")
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("Total Action Items: %d", len(items)))
	row++

	// Count by priority
	highCount := 0
	mediumCount := 0
	reviewCount := 0
	infoCount := 0
	for _, item := range items {
		switch {
		case item.prioritySort <= 1:
			highCount++
		case item.prioritySort == 2:
			mediumCount++
		case item.prioritySort == 3:
			reviewCount++
		default:
			infoCount++
		}
	}

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("HIGH/CRITICAL (True Discrepancies): %d", highCount))
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("MEDIUM (Foundation Only - Verify): %d", mediumCount))
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("REVIEW (Potential Matches): %d", reviewCount))
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("INFO (Payments/Out-of-Range - No Action): %d", infoCount))
}

func (g *ExcelReportGenerator) createSummarySheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "Summary"
	f.NewSheet(sheetName)

	f.SetColWidth(sheetName, "A", "A", 45)
	f.SetColWidth(sheetName, "B", "B", 25)

	row := 1
	f.SetCellValue(sheetName, "A1", "Reconciliation Summary Report")
	f.SetCellStyle(sheetName, "A1", "A1", formats["header"])
	f.SetCellValue(sheetName, "A2", fmt.Sprintf("Generated: %s", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	// Date Range Section
	row = 4
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "📅 Date Range Analysis")
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), formats["header"])
	row++

	if report.BankMinDate != nil && report.BankMaxDate != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Bank Statement Period")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("%s to %s", report.BankMinDate.Format("01/02/2006"), report.BankMaxDate.Format("01/02/2006")))
		row++
	}
	if report.FoundationMinDate != nil && report.FoundationMaxDate != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Foundation Period")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("%s to %s", report.FoundationMinDate.Format("01/02/2006"), report.FoundationMaxDate.Format("01/02/2006")))
		row++
	}
	if report.OverlapStart != nil && report.OverlapEnd != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Overlapping Period (used for matching)")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("%s to %s", report.OverlapStart.Format("01/02/2006"), report.OverlapEnd.Format("01/02/2006")))
		row++
	}

	// Transaction counts
	row += 1
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "📊 Transaction Counts")
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), formats["header"])
	row++

	totalBank := report.TotalMatched() + report.TotalBankOnly()
	totalFoundation := report.TotalMatched() + report.TotalFoundationOnly()

	metrics := []struct {
		label string
		value interface{}
	}{
		{"Total Bank Transactions", totalBank},
		{"Total Foundation Transactions", totalFoundation},
	}

	for _, metric := range metrics {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), metric.label)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), metric.value)
		row++
	}

	// Matching results
	row += 1
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "✅ Matching Results")
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), formats["header"])
	row++

	matchResults := []struct {
		label  string
		value  interface{}
		style  int
	}{
		{"Matched Transactions", report.TotalMatched(), formats["matched"]},
		{"Match Rate (Actionable)", fmt.Sprintf("%.1f%%", report.ActionableMatchRate()), formats["matched"]},
		{"Match Rate (Overall)", fmt.Sprintf("%.1f%%", report.MatchRate()), 0},
		{"", "", 0},
		{"🔍 Potential Matches (Need Review)", len(report.PotentialMatches), formats["potential"]},
		{"♻️  Void Pairs Identified", len(report.VoidPairs), 0},
		{"⚠️  Ambiguous Voids (Manual Review)", len(report.AmbiguousVoids), 0},
	}

	for _, m := range matchResults {
		if m.label != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), m.label)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), m.value)
			if m.style != 0 {
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), m.style)
			}
		}
		row++
	}

	// Discrepancy breakdown
	row += 1
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "🔍 Discrepancy Breakdown")
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), formats["header"])
	row++

	discrepancies := []struct {
		label string
		value interface{}
		style int
	}{
		{"🏦 Bank Only — True Discrepancies (need action)", len(report.BankTrueDiscrepancies), formats["bank_only"]},
		{"📊 Foundation Only — True Discrepancies (need action)", len(report.FoundationTrueDiscrepancies), formats["foundation_only"]},
		{"", "", 0},
		{"💳 Bank Payments/Credits (no action needed)", len(report.BankPayments), formats["potential"]},
		{"📅 Bank — Outside Foundation Date Range", len(report.BankOutOfRange), formats["potential"]},
		{"📅 Foundation — Outside Bank Date Range", len(report.FoundationOutOfRange), formats["potential"]},
	}

	for _, d := range discrepancies {
		if d.label != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), d.label)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), d.value)
			if d.style != 0 {
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), d.style)
			}
		}
		row++
	}
}

func (g *ExcelReportGenerator) createMatchedSheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "Matched"
	f.NewSheet(sheetName)

	headers := []string{"Bank Date", "Bank Amount", "Bank Description", "Bank Cardmember", "Foundation Date", "Foundation Amount", "Foundation Vendor", "Trx#", "Job#", "Date Diff", "Amount Diff"}
	
	for col, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+col)))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}

	row := 2
	for _, match := range report.MatchedTransactions {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), match.BankTransaction.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), match.BankTransaction.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), match.BankTransaction.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), match.BankTransaction.CardmemberName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), match.FoundationTransaction.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), match.FoundationTransaction.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), match.FoundationTransaction.VendorName)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), match.FoundationTransaction.TransactionNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), match.FoundationTransaction.JobNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), match.DateDifference)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), match.AmountDifference)

		// Apply matched formatting
		for col := 'A'; col <= 'K'; col++ {
			cell := fmt.Sprintf("%s%d", string(col), row)
			f.SetCellStyle(sheetName, cell, cell, formats["matched"])
		}
		row++
	}
}

func (g *ExcelReportGenerator) createDiscrepanciesSheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "Discrepancies"
	f.NewSheet(sheetName)

	f.SetCellValue(sheetName, "A1", "Bank Only Transactions")
	f.SetCellStyle(sheetName, "A1", "A1", formats["header"])

	headers := []string{"Date", "Amount", "Description", "Cardmember", "Reference"}
	for col, header := range headers {
		cell := fmt.Sprintf("%s2", string(rune('A'+col)))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}

	row := 3
	for _, txn := range report.BankOnlyTransactions {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), txn.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), txn.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), txn.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), txn.CardmemberName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), txn.Reference)

		for col := 'A'; col <= 'E'; col++ {
			cell := fmt.Sprintf("%s%d", string(col), row)
			f.SetCellStyle(sheetName, cell, cell, formats["bank_only"])
		}
		row++
	}

	row += 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Foundation Only Transactions")
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), formats["header"])
	row++

	headers = []string{"Date", "Amount", "Description", "Vendor", "Trx#", "Job#"}
	for col, header := range headers {
		cell := fmt.Sprintf("%s%d", string(rune('A'+col)), row)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}
	row++

	for _, txn := range report.FoundationOnlyTransactions {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), txn.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), txn.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), txn.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), txn.VendorName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), txn.TransactionNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), txn.JobNumber)

		for col := 'A'; col <= 'F'; col++ {
			cell := fmt.Sprintf("%s%d", string(col), row)
			f.SetCellStyle(sheetName, cell, cell, formats["foundation_only"])
		}
		row++
	}
}

func (g *ExcelReportGenerator) createPotentialMatchesSheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "Potential Matches"
	f.NewSheet(sheetName)

	headers := []string{"Score", "Bank Date", "Bank Amount", "Bank Description", "Foundation Date", "Foundation Amount", "Foundation Vendor", "Trx#", "Job#"}
	
	for col, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+col)))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}

	row := 2
	for _, pm := range report.PotentialMatches {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("%.0f%%", pm.Score*100))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), pm.BankTransaction.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), pm.BankTransaction.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), pm.BankTransaction.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), pm.FoundationTransaction.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), pm.FoundationTransaction.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), pm.FoundationTransaction.VendorName)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), pm.FoundationTransaction.TransactionNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), pm.FoundationTransaction.JobNumber)

		for col := 'A'; col <= 'I'; col++ {
			cell := fmt.Sprintf("%s%d", string(col), row)
			f.SetCellStyle(sheetName, cell, cell, formats["potential"])
		}
		row++
	}
}

func (g *ExcelReportGenerator) createVoidsSheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "Voids"
	f.NewSheet(sheetName)

	headers := []string{"Trx#", "Date 1", "Amount 1", "Description 1", "Date 2", "Amount 2", "Description 2", "Net Amount"}
	
	for col, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+col)))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}

	row := 2
	for _, vp := range report.VoidPairs {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), vp.Transaction1.TransactionNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), vp.Transaction1.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), vp.Transaction1.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), vp.Transaction1.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), vp.Transaction2.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), vp.Transaction2.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), vp.Transaction2.Description)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), vp.Transaction1.Amount+vp.Transaction2.Amount)
		row++
	}
}

func (g *ExcelReportGenerator) createAmbiguousVoidsSheet(f *excelize.File, report *models.ReconciliationReport, formats map[string]int) {
	sheetName := "Ambiguous Voids"
	f.NewSheet(sheetName)

	f.SetCellValue(sheetName, "A1", "⚠️ CRITICAL: These situations require manual review")
	f.SetCellStyle(sheetName, "A1", "A1", formats["action_required"])

	headers := []string{"Amount", "Date", "Count", "Positives", "Negatives", "Transaction Numbers", "Action Required"}
	
	for col, header := range headers {
		cell := fmt.Sprintf("%s2", string(rune('A'+col)))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, formats["header"])
	}

	row := 3
	for _, av := range report.AmbiguousVoids {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), av.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), av.Date.Format("01/02/2006"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), av.TransactionCount)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), av.PositiveCount)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), av.NegativeCount)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), strings.Join(av.TransactionNumbers, ", "))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), "Verify which transactions net to zero and which are actual charges")

		for col := 'A'; col <= 'G'; col++ {
			cell := fmt.Sprintf("%s%d", string(col), row)
			f.SetCellStyle(sheetName, cell, cell, formats["action_required"])
		}
		row++
	}
}

// Helper functions
func (g *ExcelReportGenerator) isPaymentTransaction(txn models.BankTransaction) bool {
	return isPaymentTransaction(txn)
}

func (g *ExcelReportGenerator) isPrepaidCardTransaction(txn models.FoundationTransaction) bool {
	desc := strings.ToLower(txn.Description)
	return strings.Contains(desc, "pre-paid") || strings.Contains(desc, "prepaid") ||
		strings.Contains(desc, "concur")
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
