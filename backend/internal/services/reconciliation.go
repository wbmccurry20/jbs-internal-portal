package services

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
	"github.com/xuri/excelize/v2"
)

type ReconciliationEngine struct {
	DateToleranceDays     int
	AmountTolerancePercent float64
	MinDate               *time.Time
	MaxDate               *time.Time
	ambiguousVoids        []models.AmbiguousVoid
}

func NewReconciliationEngine(dateToleranceDays int, amountTolerancePercent float64) *ReconciliationEngine {
	return &ReconciliationEngine{
		DateToleranceDays:     dateToleranceDays,
		AmountTolerancePercent: amountTolerancePercent,
		ambiguousVoids:        make([]models.AmbiguousVoid, 0),
	}
}

// Reconcile performs the reconciliation between bank and foundation transactions
func (e *ReconciliationEngine) Reconcile(bankTrans []models.BankTransaction, foundationTrans []models.FoundationTransaction) *models.ReconciliationReport {
	// Filter transactions
	bankFiltered := e.filterBankTransactions(bankTrans)
	foundationFiltered := e.filterFoundationTransactions(foundationTrans)

	// Handle voids in Foundation
	foundationCleaned, voidPairs := e.handleVoids(foundationFiltered)

	// Match transactions
	matched := make([]models.MatchedPair, 0)
	potentialMatches := make([]models.PotentialMatch, 0)
	bankUnmatched := make([]models.BankTransaction, len(bankFiltered))
	copy(bankUnmatched, bankFiltered)
	foundationUnmatched := make([]models.FoundationTransaction, len(foundationCleaned))
	copy(foundationUnmatched, foundationCleaned)

	// Pass 1: Find best matches
	for i := 0; i < len(bankFiltered); i++ {
		bankTxn := bankFiltered[i]
		
		// Check if already matched
		alreadyMatched := false
		for _, m := range matched {
			if transactionsEqual(m.BankTransaction, bankTxn) {
				alreadyMatched = true
				break
			}
		}
		if alreadyMatched {
			continue
		}

		// Find all possible matches
		type candidate struct {
			foundTxn     models.FoundationTransaction
			score        float64
			dateDiff     int
			amountDiff   float64
			foundIndex   int
		}
		candidates := make([]candidate, 0)

		for j := 0; j < len(foundationUnmatched); j++ {
			foundTxn := foundationUnmatched[j]
			if e.isMatch(bankTxn, foundTxn) {
				dateDiff := int(math.Abs(float64(bankTxn.Date.Sub(foundTxn.Date).Hours() / 24)))
				amountDiff := math.Abs(bankTxn.Amount - foundTxn.Amount)
				score := float64(dateDiff) + (amountDiff * 10)

				candidates = append(candidates, candidate{
					foundTxn:   foundTxn,
					score:      score,
					dateDiff:   dateDiff,
					amountDiff: amountDiff,
					foundIndex: j,
				})
			}
		}

		// If we found matches, take the best one
		if len(candidates) > 0 {
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].score < candidates[j].score
			})

			best := candidates[0]
			matched = append(matched, models.MatchedPair{
				BankTransaction:       bankTxn,
				FoundationTransaction: best.foundTxn,
				DateDifference:        best.dateDiff,
				AmountDifference:      best.amountDiff,
			})

			// Remove from unmatched lists
			bankUnmatched = removeBank(bankUnmatched, bankTxn)
			foundationUnmatched = removeFoundation(foundationUnmatched, best.foundTxn)
		}
	}

	// Pass 2: Find potential matches for remaining unmatched
	potentialBank := make([]models.BankTransaction, 0)
	potentialFoundation := make([]models.FoundationTransaction, 0)

	for _, bankTxn := range bankUnmatched {
		var bestMatch *models.FoundationTransaction
		var bestScore float64 = 0

		for _, foundTxn := range foundationUnmatched {
			score := e.calculateMatchScore(bankTxn, foundTxn)
			if score > 0.6 && score > bestScore {
				bestScore = score
				foundTxn := foundTxn
				bestMatch = &foundTxn
			}
		}

		if bestMatch != nil {
			potentialMatches = append(potentialMatches, models.PotentialMatch{
				BankTransaction:       bankTxn,
				FoundationTransaction: *bestMatch,
				Score:                 bestScore,
			})
			potentialBank = append(potentialBank, bankTxn)
			potentialFoundation = append(potentialFoundation, *bestMatch)
		}
	}

	// Remove potential matches from final unmatched lists
	finalBankUnmatched := make([]models.BankTransaction, 0)
	for _, txn := range bankUnmatched {
		found := false
		for _, pot := range potentialBank {
			if transactionsEqual(txn, pot) {
				found = true
				break
			}
		}
		if !found {
			finalBankUnmatched = append(finalBankUnmatched, txn)
		}
	}

	finalFoundationUnmatched := make([]models.FoundationTransaction, 0)
	for _, txn := range foundationUnmatched {
		found := false
		for _, pot := range potentialFoundation {
			if foundationTransactionsEqual(txn, pot) {
				found = true
				break
			}
		}
		if !found {
			finalFoundationUnmatched = append(finalFoundationUnmatched, txn)
		}
	}

	report := &models.ReconciliationReport{
		MatchedTransactions:        matched,
		BankOnlyTransactions:       finalBankUnmatched,
		FoundationOnlyTransactions: finalFoundationUnmatched,
		PotentialMatches:           potentialMatches,
		VoidPairs:                  voidPairs,
		AmbiguousVoids:             e.ambiguousVoids,
		GeneratedAt:                time.Now(),
	}

	// Compute date ranges and classify unmatched transactions
	e.classifyUnmatched(report, bankFiltered, foundationCleaned)

	return report
}

func (e *ReconciliationEngine) filterBankTransactions(transactions []models.BankTransaction) []models.BankTransaction {
	filtered := make([]models.BankTransaction, 0)
	for _, txn := range transactions {
		if e.MinDate != nil && txn.Date.Before(*e.MinDate) {
			continue
		}
		if e.MaxDate != nil && txn.Date.After(*e.MaxDate) {
			continue
		}
		filtered = append(filtered, txn)
	}
	return filtered
}

func (e *ReconciliationEngine) filterFoundationTransactions(transactions []models.FoundationTransaction) []models.FoundationTransaction {
	filtered := make([]models.FoundationTransaction, 0)
	for _, txn := range transactions {
		if e.MinDate != nil && txn.Date.Before(*e.MinDate) {
			continue
		}
		if e.MaxDate != nil && txn.Date.After(*e.MaxDate) {
			continue
		}
		filtered = append(filtered, txn)
	}
	return filtered
}

// classifyUnmatched categorizes unmatched transactions into payments, out-of-range, and true discrepancies
func (e *ReconciliationEngine) classifyUnmatched(report *models.ReconciliationReport, bankAll []models.BankTransaction, foundAll []models.FoundationTransaction) {
	// Compute date ranges from all loaded transactions
	if len(bankAll) > 0 {
		minD, maxD := bankAll[0].Date, bankAll[0].Date
		for _, t := range bankAll {
			if t.Date.Before(minD) {
				minD = t.Date
			}
			if t.Date.After(maxD) {
				maxD = t.Date
			}
		}
		report.BankMinDate = &minD
		report.BankMaxDate = &maxD
	}
	if len(foundAll) > 0 {
		minD, maxD := foundAll[0].Date, foundAll[0].Date
		for _, t := range foundAll {
			// Consider AltDate when computing the Concur/Foundation date range
			d := t.Date
			if t.AltDate != nil && t.AltDate.After(t.Date) {
				d = *t.AltDate
			}
			if t.Date.Before(minD) {
				minD = t.Date
			}
			if d.After(maxD) {
				maxD = d
			}
		}
		report.FoundationMinDate = &minD
		report.FoundationMaxDate = &maxD
	}

	// Build the reconciliation window anchored on the BANK statement.
	// AmEx and similar bank statements use settlement/batch dates (typically the
	// last few days of the billing period) rather than individual purchase dates.
	// The full billing cycle typically spans ~35 days before the statement
	// closing date. We use bankMaxDate as the anchor and reach back 35 days so
	// that Concur/Foundation entries made throughout the billing cycle are all
	// considered in-range — not just those whose date happens to land within a
	// few days of the batch settlement date.
	if report.BankMaxDate != nil {
		tolerance := time.Duration(e.DateToleranceDays) * 24 * time.Hour
		// Extend look-back to cover the full billing cycle (~35 days) plus tolerance
		billingCycleStart := report.BankMaxDate.Add(-(35 * 24 * time.Hour)).Add(-tolerance)
		report.OverlapStart = &billingCycleStart
		adjustedEnd := report.BankMaxDate.Add(tolerance)
		report.OverlapEnd = &adjustedEnd
	}

	// Classify bank-only transactions
	report.BankPayments = make([]models.BankTransaction, 0)
	report.BankOutOfRange = make([]models.BankTransaction, 0)
	report.BankTrueDiscrepancies = make([]models.BankTransaction, 0)

	for _, txn := range report.BankOnlyTransactions {
		if isPaymentTransaction(txn) {
			report.BankPayments = append(report.BankPayments, txn)
		} else if report.OverlapStart != nil && report.OverlapEnd != nil &&
			(txn.Date.Before(*report.OverlapStart) || txn.Date.After(*report.OverlapEnd)) {
			report.BankOutOfRange = append(report.BankOutOfRange, txn)
		} else {
			report.BankTrueDiscrepancies = append(report.BankTrueDiscrepancies, txn)
		}
	}

	// Classify foundation-only transactions.
	// A Foundation/Concur entry is in-range if EITHER its primary date OR its
	// AltDate falls within the billing cycle window.
	report.FoundationOutOfRange = make([]models.FoundationTransaction, 0)
	report.FoundationTrueDiscrepancies = make([]models.FoundationTransaction, 0)

	for _, txn := range report.FoundationOnlyTransactions {
		inWindow := false
		if report.OverlapStart != nil && report.OverlapEnd != nil {
			if !txn.Date.Before(*report.OverlapStart) && !txn.Date.After(*report.OverlapEnd) {
				inWindow = true
			}
			if !inWindow && txn.AltDate != nil {
				if !txn.AltDate.Before(*report.OverlapStart) && !txn.AltDate.After(*report.OverlapEnd) {
					inWindow = true
				}
			}
		} else {
			inWindow = true
		}
		if inWindow {
			report.FoundationTrueDiscrepancies = append(report.FoundationTrueDiscrepancies, txn)
		} else {
			report.FoundationOutOfRange = append(report.FoundationOutOfRange, txn)
		}
	}
}

// isPaymentTransaction detects AmEx payment/credit transactions (not purchases)
func isPaymentTransaction(txn models.BankTransaction) bool {
	desc := strings.ToLower(txn.Description)
	return strings.Contains(desc, "payment") ||
		strings.Contains(desc, "autopay") ||
		strings.Contains(desc, "online pmt") ||
		strings.Contains(desc, "rebilling") ||
		strings.Contains(desc, "membership cancelled") ||
		strings.Contains(desc, "late fee") ||
		strings.Contains(desc, "interest charge") ||
		(txn.Amount < 0 && math.Abs(txn.Amount) > 5000)
}

func (e *ReconciliationEngine) handleVoids(transactions []models.FoundationTransaction) ([]models.FoundationTransaction, []models.VoidPair) {
	voidPairs := make([]models.VoidPair, 0)
	cleaned := make([]models.FoundationTransaction, 0)
	usedIndices := make(map[int]bool)

	// Group by amount and date to find ambiguous situations
	type amountDateKey struct {
		amount float64
		date   string
	}
	groups := make(map[amountDateKey][]struct {
		index int
		txn   models.FoundationTransaction
	})

	for i, txn := range transactions {
		key := amountDateKey{
			amount: math.Round(math.Abs(txn.Amount)*100) / 100,
			date:   txn.Date.Format("2006-01-02"),
		}
		groups[key] = append(groups[key], struct {
			index int
			txn   models.FoundationTransaction
		}{i, txn})
	}

	// Check for ambiguous situations
	for _, group := range groups {
		if len(group) >= 2 {
			positives := make([]struct {
				index int
				txn   models.FoundationTransaction
			}, 0)
			negatives := make([]struct {
				index int
				txn   models.FoundationTransaction
			}, 0)

			for _, item := range group {
				if item.txn.Amount > 0 {
					positives = append(positives, item)
				} else if item.txn.Amount < 0 {
					negatives = append(negatives, item)
				}
			}

			// Ambiguous if more positives than negatives
			if len(negatives) > 0 && len(positives) > len(negatives) {
				txnNumbers := make([]string, 0)
				txns := make([]models.FoundationTransaction, 0)
				for _, item := range group {
					if item.txn.Reference != "" {
						txnNumbers = append(txnNumbers, item.txn.Reference)
					}
					txns = append(txns, item.txn)
				}

				e.ambiguousVoids = append(e.ambiguousVoids, models.AmbiguousVoid{
					Amount:             group[0].txn.Amount,
					Date:               group[0].txn.Date,
					TransactionCount:   len(group),
					TransactionNumbers: txnNumbers,
					Transactions:       txns,
					PositiveCount:      len(positives),
					NegativeCount:      len(negatives),
				})
			}
		}
	}

	// Find void pairs
	for i, txn1 := range transactions {
		if usedIndices[i] {
			continue
		}

		foundPair := false
		for j := i + 1; j < len(transactions); j++ {
			if usedIndices[j] {
				continue
			}

			txn2 := transactions[j]

			// Same transaction number and opposite amounts
			if txn1.TransactionNumber == txn2.TransactionNumber &&
				txn1.TransactionNumber != 0 &&
				math.Abs(txn1.Amount+txn2.Amount) < 0.01 {
				voidPairs = append(voidPairs, models.VoidPair{
					Transaction1: txn1,
					Transaction2: txn2,
				})
				usedIndices[i] = true
				usedIndices[j] = true
				foundPair = true
				break
			}
		}

		if !foundPair {
			cleaned = append(cleaned, txn1)
		}
	}

	return cleaned, voidPairs
}

func (e *ReconciliationEngine) isMatch(bankTxn models.BankTransaction, foundTxn models.FoundationTransaction) bool {
	// Check amount match
	amountDiff := math.Abs(bankTxn.Amount - foundTxn.Amount)
	amountThreshold := math.Max(math.Abs(bankTxn.Amount*e.AmountTolerancePercent), 0.02)

	if amountDiff > amountThreshold {
		return false
	}

	// Check date match — try primary date first, then AltDate (e.g. Concur
	// transaction_date vs invoice_date) so AmEx settlement-date statements
	// can match Concur entries whose purchase date is within the billing cycle.
	dateDiff := int(math.Abs(float64(bankTxn.Date.Sub(foundTxn.Date).Hours() / 24)))
	if dateDiff <= e.DateToleranceDays {
		return true
	}
	if foundTxn.AltDate != nil {
		altDiff := int(math.Abs(float64(bankTxn.Date.Sub(*foundTxn.AltDate).Hours() / 24)))
		if altDiff <= e.DateToleranceDays {
			return true
		}
	}
	return false
}

func (e *ReconciliationEngine) calculateMatchScore(bankTxn models.BankTransaction, foundTxn models.FoundationTransaction) float64 {
	score := 0.0

	// Amount similarity (50% weight)
	amountDiff := math.Abs(bankTxn.Amount - foundTxn.Amount)
	if amountDiff < 0.01 {
		score += 0.5
	} else if amountDiff < math.Abs(bankTxn.Amount*0.05) {
		score += 0.3
	} else if amountDiff < math.Abs(bankTxn.Amount*0.1) {
		score += 0.1
	}

	// Date proximity (50% weight) — try both primary date and AltDate
	dateDiff := int(math.Abs(float64(bankTxn.Date.Sub(foundTxn.Date).Hours() / 24)))
	if foundTxn.AltDate != nil {
		altDiff := int(math.Abs(float64(bankTxn.Date.Sub(*foundTxn.AltDate).Hours() / 24)))
		if altDiff < dateDiff {
			dateDiff = altDiff
		}
	}
	if dateDiff == 0 {
		score += 0.5
	} else if dateDiff <= 1 {
		score += 0.4
	} else if dateDiff <= 3 {
		score += 0.3
	} else if dateDiff <= 7 {
		score += 0.15
	} else if dateDiff <= 35 && amountDiff < 0.01 {
		// Billing-cycle match: AmEx statements use settlement/batch dates which
		// cluster in the last few days of the billing period. The actual purchase
		// date may be up to ~35 days earlier. When the amount is an exact match,
		// score high enough to surface as a potential match for review.
		score += 0.12
	}

	return score
}

// LoadBankTransactions loads bank transactions from CSV or Excel file
func LoadBankTransactions(filePath string) ([]models.BankTransaction, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".xlsx" {
		result, err := loadBankFromExcel(filePath)
		if err != nil {
			return nil, err
		}
		log.Printf("[reconciliation] Loaded %d bank transactions from XLSX: %s", len(result), filepath.Base(filePath))
		return result, nil
	} else if ext == ".xls" {
		// Preferred: use Node.js/SheetJS which handles all BIFF variants and gets all rows.
		if result, err := loadBankFromXLSViaNode(filePath); err == nil {
			log.Printf("[reconciliation] Loaded %d bank transactions via Node.js/SheetJS: %s", len(result), filepath.Base(filePath))
			return result, nil
		} else {
			log.Printf("[reconciliation] Node.js XLS conversion failed for %s: %v — trying extrame/xls fallback", filepath.Base(filePath), err)
		}
		// Fallback: extrame/xls (may miss rows on some files)
		result, err := loadBankFromXLS(filePath)
		if err != nil {
			log.Printf("[reconciliation] extrame/xls failed for %s: %v — trying excelize fallback", filepath.Base(filePath), err)
			if xlsxResult, xlsxErr := loadBankFromExcel(filePath); xlsxErr == nil {
				log.Printf("[reconciliation] Loaded %d bank transactions via excelize fallback: %s", len(xlsxResult), filepath.Base(filePath))
				return xlsxResult, nil
			}
			return nil, err
		}
		log.Printf("[reconciliation] Loaded %d bank transactions via extrame/xls: %s", len(result), filepath.Base(filePath))
		return result, nil
	}
	return loadBankFromCSV(filePath)
}

// loadBankFromXLSViaNode converts the XLS to CSV using Node.js/SheetJS, which handles
// all BIFF variants correctly (including cases where extrame/xls only reads a subset of rows).
func loadBankFromXLSViaNode(filePath string) ([]models.BankTransaction, error) {
	scriptPath, err := findXLSConvertScript(filePath)
	if err != nil {
		return nil, fmt.Errorf("xls_to_csv.js not found: %v", err)
	}

	// Run: node xls_to_csv.js <xlsFile> and capture CSV output
	cmd := exec.Command("node", scriptPath, filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("node xls_to_csv.js failed: %v", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("node xls_to_csv.js produced no output")
	}

	// Write to a temp CSV file and reuse the existing CSV parser
	tmp, err := os.CreateTemp("", "bank_xls_*.csv")
	if err != nil {
		return nil, fmt.Errorf("could not create temp file: %v", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("could not write temp CSV: %v", err)
	}
	tmp.Close()

	return loadBankFromCSV(tmpName)
}

// findXLSConvertScript locates scripts/xls_to_csv.js using multiple strategies:
//  1. XLS_CONVERT_SCRIPT env var (explicit override)
//  2. Relative to the running executable (works in deployed containers)
//  3. Relative to the current working directory (works during development)
//  4. Walking up from the XLS file's directory (legacy fallback)
func findXLSConvertScript(xlsFilePath string) (string, error) {
	// Strategy 1: Explicit env var
	if envPath := os.Getenv("XLS_CONVERT_SCRIPT"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// Strategy 2: Relative to the running executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		// Try scripts/ next to the executable and one level up (common layout:
		// backend/server + scripts/xls_to_csv.js both under project root)
		for _, rel := range []string{
			filepath.Join(exeDir, "scripts", "xls_to_csv.js"),
			filepath.Join(exeDir, "..", "scripts", "xls_to_csv.js"),
			filepath.Join(exeDir, "..", "..", "scripts", "xls_to_csv.js"),
		} {
			if abs, err := filepath.Abs(rel); err == nil {
				if _, err := os.Stat(abs); err == nil {
					return abs, nil
				}
			}
		}
	}

	// Strategy 3: Relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		for _, rel := range []string{
			filepath.Join(cwd, "scripts", "xls_to_csv.js"),
			filepath.Join(cwd, "..", "scripts", "xls_to_csv.js"),
		} {
			if abs, err := filepath.Abs(rel); err == nil {
				if _, err := os.Stat(abs); err == nil {
					return abs, nil
				}
			}
		}
	}

	// Strategy 4: Walk up from the XLS file's directory (legacy fallback)
	dir, err := filepath.Abs(filepath.Dir(xlsFilePath))
	if err != nil {
		return "", err
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "scripts", "xls_to_csv.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("scripts/xls_to_csv.js not found (checked env XLS_CONVERT_SCRIPT, exe dir, cwd, and near %s)", xlsFilePath)
}

func loadBankFromCSV(filePath string) ([]models.BankTransaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Allow variable number of fields per row
	reader.FieldsPerRecord = -1
	// Be more flexible with CSV parsing
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV file: %v", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// Find header row (skip AmEx preamble)
	headerRowIdx := findBankHeaderRow(records)
	if headerRowIdx == -1 {
		return nil, fmt.Errorf("could not find header row in bank statement")
	}

	// Map headers
	header := records[headerRowIdx]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.BankTransaction, 0)

	// Start parsing from row after header
	for i := headerRowIdx + 1; i < len(records); i++ {
		row := records[i]
		
		// Get date and amount with AmEx column name support
		dateStr := getBankColumn(row, colMap, "Date", "Transaction Date")
		amountStr := getBankColumn(row, colMap, "Amount", "Transaction Amount USD")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		// Parse amount, handling currency symbols, commas, and any residual whitespace
		cleanAmount := strings.TrimSpace(amountStr)
		cleanAmount = strings.ReplaceAll(cleanAmount, "$", "")
		cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
		cleanAmount = strings.TrimSpace(cleanAmount)
		amount, err := strconv.ParseFloat(cleanAmount, 64)
		if err != nil {
			continue
		}

		// Build description from multiple AmEx fields or single field
		description := buildBankDescription(row, colMap)
		
		// Get cardmember name (combine first/last for AmEx format)
		cardmemberName := getBankCardmember(row, colMap)

		transactions = append(transactions, models.BankTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: description,
				Reference:   getBankColumn(row, colMap, "Transaction Reference No.", "Transaction Reference No."),
			},
			CardmemberName: cardmemberName,
			CardAccount:    getBankColumn(row, colMap, "Card Account No."),
		})
	}

	return transactions, nil
}

func loadBankFromExcel(filePath string) ([]models.BankTransaction, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		// Provide more helpful error message
		if strings.Contains(err.Error(), "not supported") || strings.Contains(err.Error(), "workbook") {
			return nil, fmt.Errorf("Excel file format not supported - please ensure file is saved as .xlsx (not .xls). Error: %v", err)
		}
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty Excel file")
	}

	// Find header row (skip AmEx preamble)
	headerRowIdx := findBankHeaderRow(rows)
	if headerRowIdx == -1 {
		return nil, fmt.Errorf("could not find header row in bank statement")
	}

	// Map headers
	header := rows[headerRowIdx]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.BankTransaction, 0)

	// Start parsing from row after header
	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]
		
		dateStr := getBankColumn(row, colMap, "Date", "Transaction Date")
		amountStr := getBankColumn(row, colMap, "Amount", "Transaction Amount USD")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		// Parse amount, handling currency symbols, commas, and any residual whitespace
		cleanAmount := strings.TrimSpace(amountStr)
		cleanAmount = strings.ReplaceAll(cleanAmount, "$", "")
		cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
		cleanAmount = strings.TrimSpace(cleanAmount)
		amount, err := strconv.ParseFloat(cleanAmount, 64)
		if err != nil {
			continue
		}

		// Build description from multiple AmEx fields or single field
		description := buildBankDescription(row, colMap)
		
		// Get cardmember name (combine first/last for AmEx format)
		cardmemberName := getBankCardmember(row, colMap)

		transactions = append(transactions, models.BankTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: description,
				Reference:   getBankColumn(row, colMap, "Transaction Reference No.", "Transaction Reference No."),
			},
			CardmemberName: cardmemberName,
			CardAccount:    getBankColumn(row, colMap, "Card Account No."),
		})
	}

	return transactions, nil
}

// xlsRowSafe reads a single XLS row; returns nil if the library panics (e.g. merged/empty cells).
// IMPORTANT: always returns a slice indexed from column 0 (padding leading empty columns with "")
// so that the column indices in colMap (built from the header row) stay aligned with data rows.
// The extrame/xls library's Row.Col() can be called for any column index but FirstCol() varies
// per row — using FirstCol() as slice offset 0 caused massive column misalignment.
// Per-cell panics (e.g. unusual cell types like ######JmKohk reference numbers) are caught
// individually so only that one cell is blanked out rather than dropping the entire row.
func xlsRowSafe(sheet *xls.WorkSheet, i int) (rowData []string) {
	defer func() { recover() }() //nolint:errcheck
	row := sheet.Row(i)
	if row == nil {
		return nil
	}
	lastCol := int(row.LastCol())
	if lastCol <= 0 {
		return nil
	}
	// Allocate from 0 so index j in the slice == column j in the sheet
	rowData = make([]string, lastCol)
	for j := int(row.FirstCol()); j < lastCol; j++ {
		func() {
			defer func() { recover() }() //nolint:errcheck
			rowData[j] = row.Col(j)
		}()
	}
	return rowData
}

func loadBankFromXLS(filePath string) ([]models.BankTransaction, error) {
	xlsFile, err := xls.Open(filePath, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("failed to open XLS file: %v", err)
	}

	if xlsFile.NumSheets() == 0 {
		return nil, fmt.Errorf("no sheets in XLS file")
	}

	sheet := xlsFile.GetSheet(0)
	if sheet == nil {
		return nil, fmt.Errorf("failed to get first sheet")
	}

	// Convert to [][]string format, skipping rows that cause the library to panic.
	rows := make([][]string, 0)
	for i := 0; i <= int(sheet.MaxRow); i++ {
		if rowData := xlsRowSafe(sheet, i); rowData != nil {
			rows = append(rows, rowData)
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty XLS file")
	}

	// Find header row (skip AmEx preamble)
	headerRowIdx := findBankHeaderRow(rows)
	if headerRowIdx == -1 {
		return nil, fmt.Errorf("could not find header row in bank statement")
	}

	// Map headers
	header := rows[headerRowIdx]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.BankTransaction, 0)

	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]

		// AmEx XLS files occasionally insert an extra asterisk-wrapped tracking ID
		// at the Transaction Reference No. column, shifting amount/desc right by one.
		refColIdx := colMap["Transaction Reference No."]
		if refColIdx > 0 && refColIdx < len(row) {
			cell := strings.TrimSpace(row[refColIdx])
			if len(cell) >= 2 && cell[0] == '*' && cell[len(cell)-1] == '*' {
				adjusted := make([]string, len(row)-1)
				copy(adjusted, row[:refColIdx])
				copy(adjusted[refColIdx:], row[refColIdx+1:])
				row = adjusted
			}
		}

		// --- Fast path: use column-map indices (works for well-aligned rows) ---
		dateStr := getBankColumn(row, colMap, "Date", "Transaction Date")
		amountStr := getBankColumn(row, colMap, "Amount", "Transaction Amount USD")

		date, dateOk := func() (time.Time, bool) {
			if dateStr == "" {
				return time.Time{}, false
			}
			t, err := parseDate(dateStr)
			return t, err == nil
		}()

		amount, amountOk := func() (float64, bool) {
			if amountStr == "" {
				return 0, false
			}
			clean := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(amountStr, "$", ""), ",", ""))
			clean = strings.TrimSpace(clean)
			v, err := strconv.ParseFloat(clean, 64)
			return v, err == nil
		}()

		var description, cardmemberName string

		if dateOk && amountOk {
			// Normal path — column map is aligned
			description = buildBankDescription(row, colMap)
			cardmemberName = getBankCardmember(row, colMap)
		} else {
			// --- Content-based fallback for AmEx rows where the extrame/xls library  ---
			// --- returns cells at wrong column indices due to sparse/shifted BIFF rows ---
			var ok bool
			date, amount, cardmemberName, description, ok = scanXLSRowForBankFields(row)
			if !ok {
				continue // not a real transaction row (continuation/hash-only row)
			}
		}

		transactions = append(transactions, models.BankTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: description,
				Reference:   getBankColumn(row, colMap, "Transaction Reference No.", "Transaction Reference No."),
			},
			CardmemberName: cardmemberName,
			CardAccount:    getBankColumn(row, colMap, "Card Account No."),
		})
	}

	return transactions, nil
}

// scanXLSRowForBankFields performs content-based extraction from XLS rows where
// the extrame/xls library has returned cells at wrong column positions (common in
// AmEx exports where sparse BIFF rows start at col 12+, making cols 0-11 all empty).
//
// Strategy:
//   - Find MM/DD/YYYY dates (4-digit year only; skips 2-digit-year description dates)
//   - Find the first $-prefixed amount
//   - Use the 2nd full date as Transaction Date (1st is Business Process Date)
//   - Collect cells after the amount as description
//   - If no dollar amount found → not a real transaction row → return ok=false
func scanXLSRowForBankFields(row []string) (date time.Time, amount float64, cardmember, desc string, ok bool) {
	var fullDates []time.Time
	amtIdx := -1

	for i, raw := range row {
		cell := strings.TrimSpace(raw)
		if cell == "" {
			continue
		}
		// Skip AmEx asterisk-wrapped hash cells (e.g. "*875145409423900000012634*")
		if len(cell) >= 2 && cell[0] == '*' && cell[len(cell)-1] == '*' {
			continue
		}
		// Detect MM/DD/YYYY date — must have exactly 4-digit year to avoid grabbing
		// description date strings like "02/28/26" (2-digit year).
		if parts := strings.Split(cell, "/"); len(parts) == 3 && len(parts[2]) == 4 {
			if t, err := parseDate(cell); err == nil && t.Year() >= 2020 {
				fullDates = append(fullDates, t)
				continue
			}
		}
		// Detect dollar-prefixed amount ($X.XX, $X,XXX.XX, -$X.XX)
		if amtIdx == -1 {
			rawAmt := cell
			neg := strings.HasPrefix(rawAmt, "-$")
			if strings.HasPrefix(rawAmt, "$") || neg {
				clean := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(rawAmt, "$", ""), ",", ""))
				if v, err := strconv.ParseFloat(clean, 64); err == nil {
					amount = v
					amtIdx = i
				}
			}
		}
	}

	// A real transaction row must have at least one date AND a dollar amount
	if len(fullDates) == 0 || amtIdx == -1 {
		return
	}

	// Business Process Date comes before Transaction Date in the AmEx column order.
	// Use the 2nd full date encountered; if only one exists use it.
	if len(fullDates) >= 2 {
		date = fullDates[1]
	} else {
		date = fullDates[0]
	}

	// Cardmember name: available only when col 0 has "Corporate Card" (aligned rows).
	// Shifted rows (col0="") won't have names here, which is acceptable.
	if len(row) > 2 && strings.TrimSpace(row[0]) == "Corporate Card" {
		last := strings.TrimSpace(row[1])
		first := strings.TrimSpace(row[2])
		if last != "" {
			cardmember = strings.TrimSpace(first + " " + last)
		}
	}

	// Description: collect cells after the amount, skipping hash values
	var descParts []string
	for _, raw := range row[amtIdx+1:] {
		cell := strings.TrimSpace(raw)
		if cell == "" {
			continue
		}
		if len(cell) >= 2 && cell[0] == '*' && cell[len(cell)-1] == '*' {
			continue
		}
		descParts = append(descParts, cell)
	}
	desc = strings.Join(descParts, " ")
	ok = true
	return
}


// LoadFoundationTransactions loads foundation transactions from CSV or Excel file
func LoadFoundationTransactions(filePath string) ([]models.FoundationTransaction, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	var result []models.FoundationTransaction
	var err error

	if ext == ".xlsx" {
		result, err = loadFoundationFromExcel(filePath)
	} else if ext == ".xls" {
		// Try legacy XLS format first; fall back to XLSX in case the file was
		// saved with a .xls extension but is actually XLSX (common on macOS).
		result, err = loadFoundationFromXLS(filePath)
		if err != nil {
			if xlsxResult, xlsxErr := loadFoundationFromExcel(filePath); xlsxErr == nil {
				result = xlsxResult
				err = nil
			}
		}
	} else {
		result, err = loadFoundationFromCSV(filePath)
	}

	if err != nil {
		return nil, err
	}
	log.Printf("[reconciliation] Loaded %d foundation transactions from %s: %s", len(result), ext, filepath.Base(filePath))
	return result, nil
}

func loadFoundationFromCSV(filePath string) ([]models.FoundationTransaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Allow variable number of fields per row
	reader.FieldsPerRecord = -1
	// Be more flexible with CSV parsing
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV file: %v", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// Map headers
	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.FoundationTransaction, 0)

	for i := 1; i < len(records); i++ {
		row := records[i]
		
		// Prefer Inv Date / invoice_date (actual charge date) over Trx Date / transaction_date
		// (Foundation posting date) so dates align with bank/AmEx statement dates.
		// Column names support Foundation Title Case, Foundation snake_case, and Concur exports.
		// Also capture the secondary date so isMatch can try both against the bank.
		primaryDateStr := getBankColumn(row, colMap, "Date", "Inv Date", "invoice_date")
		secondaryDateStr := getBankColumn(row, colMap, "Trx Date", "transaction_date", "Transaction Date")
		// Fall back: if neither 'invoice_date' nor 'transaction_date' is present, accept 'Date'
		if primaryDateStr == "" {
			primaryDateStr = secondaryDateStr
			secondaryDateStr = ""
		}
		dateStr := primaryDateStr
		amountStr := getBankColumn(row, colMap, "Amount", "Trx Amount", "transaction_amount", "SUM Allocation Claimed Amount (USD)")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		// Parse optional secondary/alt date
		var altDate *time.Time
		if secondaryDateStr != "" && secondaryDateStr != dateStr {
			if ad, aerr := parseDate(secondaryDateStr); aerr == nil && !ad.Equal(date) {
				altDate = &ad
			}
		}

		// Parse amount, handling currency symbols and commas
		cleanAmount := strings.TrimSpace(amountStr)
		cleanAmount = strings.ReplaceAll(cleanAmount, "$", "")
		cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
		amount, err := strconv.ParseFloat(cleanAmount, 64)
		if err != nil {
			continue
		}

		trxNoStr := getBankColumn(row, colMap, "Trx No", "Transaction Number", "voucher")
		var trxNo int
		if trxNoStr != "" {
			trxNo, _ = strconv.Atoi(strings.TrimSpace(trxNoStr))
		}

		transactions = append(transactions, models.FoundationTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getBankColumn(row, colMap, "Description", "description", "Trx Description", "Expense Type"),
				Reference:   getBankColumn(row, colMap, "Trx No", "Transaction Number", "voucher"),
			},
			AltDate:           altDate,
			VendorName:        getBankColumn(row, colMap, "Vendor Name", "vendor_name", "VendorName", "Employee Name"),
			TransactionNumber: trxNo,
			JobNumber:         getBankColumn(row, colMap, "Job No", "job_no", "Job Number"),
		})
	}

	return transactions, nil
}

func loadFoundationFromExcel(filePath string) ([]models.FoundationTransaction, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		// Provide more helpful error message
		if strings.Contains(err.Error(), "not supported") || strings.Contains(err.Error(), "workbook") {
			return nil, fmt.Errorf("Excel file format not supported - please ensure file is saved as .xlsx (not .xls). Error: %v", err)
		}
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty Excel file")
	}

	// Map headers
	header := rows[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.FoundationTransaction, 0)

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Prefer Inv Date / invoice_date. Also capture the secondary date as AltDate.
		primaryDateStr := getBankColumn(row, colMap, "Date", "Inv Date", "invoice_date")
		secondaryDateStr := getBankColumn(row, colMap, "Trx Date", "transaction_date", "Transaction Date")
		if primaryDateStr == "" {
			primaryDateStr = secondaryDateStr
			secondaryDateStr = ""
		}
		dateStr := primaryDateStr
		amountStr := getBankColumn(row, colMap, "Amount", "Trx Amount", "transaction_amount", "SUM Allocation Claimed Amount (USD)")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		var altDate *time.Time
		if secondaryDateStr != "" && secondaryDateStr != dateStr {
			if ad, aerr := parseDate(secondaryDateStr); aerr == nil && !ad.Equal(date) {
				altDate = &ad
			}
		}

		// Parse amount, handling currency symbols and commas
		cleanAmount := strings.TrimSpace(amountStr)
		cleanAmount = strings.ReplaceAll(cleanAmount, "$", "")
		cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
		amount, err := strconv.ParseFloat(cleanAmount, 64)
		if err != nil {
			continue
		}

		trxNoStr := getBankColumn(row, colMap, "Trx No", "Transaction Number", "voucher")
		var trxNo int
		if trxNoStr != "" {
			trxNo, _ = strconv.Atoi(strings.TrimSpace(trxNoStr))
		}

		transactions = append(transactions, models.FoundationTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getBankColumn(row, colMap, "Description", "description", "Trx Description", "Expense Type"),
				Reference:   getBankColumn(row, colMap, "Trx No", "Transaction Number", "voucher"),
			},
			AltDate:           altDate,
			VendorName:        getBankColumn(row, colMap, "Vendor Name", "vendor_name", "VendorName", "Employee Name"),
			TransactionNumber: trxNo,
			JobNumber:         getBankColumn(row, colMap, "Job No", "job_no", "Job Number"),
		})
	}

	return transactions, nil
}

func loadFoundationFromXLS(filePath string) ([]models.FoundationTransaction, error) {
	xlsFile, err := xls.Open(filePath, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("failed to open XLS file: %v", err)
	}

	if xlsFile.NumSheets() == 0 {
		return nil, fmt.Errorf("no sheets in XLS file")
	}

	sheet := xlsFile.GetSheet(0)
	if sheet == nil {
		return nil, fmt.Errorf("failed to get first sheet")
	}

	// Convert to [][]string format, skipping rows that cause the library to panic.
	rows := make([][]string, 0)
	for i := 0; i <= int(sheet.MaxRow); i++ {
		if rowData := xlsRowSafe(sheet, i); rowData != nil {
			rows = append(rows, rowData)
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty XLS file")
	}

	// Map headers
	header := rows[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.FoundationTransaction, 0)

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Prefer Inv Date / invoice_date (actual charge date) over Trx Date / transaction_date.
		// Column names support Foundation Title Case, Foundation snake_case, and Concur exports.
		dateStr := getBankColumn(row, colMap, "Date", "Inv Date", "invoice_date", "Trx Date", "transaction_date", "Transaction Date")
		amountStr := getBankColumn(row, colMap, "Amount", "Trx Amount", "transaction_amount", "SUM Allocation Claimed Amount (USD)")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		// Parse amount, handling currency symbols and commas
		cleanAmount := strings.TrimSpace(amountStr)
		cleanAmount = strings.ReplaceAll(cleanAmount, "$", "")
		cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
		amount, err := strconv.ParseFloat(cleanAmount, 64)
		if err != nil {
			continue
		}

		trxNoStr := getBankColumn(row, colMap, "Trx No", "Transaction Number", "voucher")
		var trxNo int
		if trxNoStr != "" {
			trxNo, _ = strconv.Atoi(strings.TrimSpace(trxNoStr))
		}

		transactions = append(transactions, models.FoundationTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getBankColumn(row, colMap, "Description", "description", "Trx Description", "Expense Type"),
				Reference:   getBankColumn(row, colMap, "Trx No", "Transaction Number", "voucher"),
			},
			VendorName:        getBankColumn(row, colMap, "Vendor Name", "vendor_name", "VendorName", "Employee Name"),
			TransactionNumber: trxNo,
			JobNumber:         getBankColumn(row, colMap, "Job No", "job_no", "Job Number"),
		})
	}

	return transactions, nil
}

// Helper functions
func getColumn(row []string, colMap map[string]int, colName string) string {
	// Exact match first
	if idx, ok := colMap[colName]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	// Case-insensitive fallback (handles snake_case from Concur exports, Title Case from Foundation, etc.)
	lower := strings.ToLower(colName)
	for k, idx := range colMap {
		if strings.ToLower(k) == lower && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
	}
	return ""
}

// getBankColumn tries multiple column names (for AmEx format compatibility)
func getBankColumn(row []string, colMap map[string]int, primaryName string, alternateName ...string) string {
	// Try primary name first
	if val := getColumn(row, colMap, primaryName); val != "" {
		return val
	}
	// Try alternate names
	for _, name := range alternateName {
		if val := getColumn(row, colMap, name); val != "" {
			return val
		}
	}
	return ""
}

// findBankHeaderRow locates the header row in bank statement (skips AmEx preamble)
func findBankHeaderRow(rows [][]string) int {
	// Look for row containing key column names
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		// Check for AmEx format headers
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "Transaction Date" || cell == "Transaction Amount USD" ||
				cell == "Cardmember Last Name" || cell == "Business Process Date" {
				return i
			}
		}
		// Check for simple format headers
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "Date" && len(row) > 2 {
				// Verify it's actually a header by checking if next row has date-like content
				if i+1 < len(rows) && len(rows[i+1]) > 0 {
					return i
				}
			}
		}
	}
	return -1
}

// buildBankDescription combines multiple description fields for AmEx format
func buildBankDescription(row []string, colMap map[string]int) string {
	// Try single Description field first
	if desc := getColumn(row, colMap, "Description"); desc != "" {
		return desc
	}
	
	// Try single "Transaction Description" column (some AmEx formats use this)
	if desc := getColumn(row, colMap, "Transaction Description"); desc != "" {
		return desc
	}
	
	// For AmEx format: combine Transaction Description 1-16
	parts := make([]string, 0)
	for i := 1; i <= 16; i++ {
		fieldName := fmt.Sprintf("Transaction Description %d", i)
		if val := getColumn(row, colMap, fieldName); val != "" {
			parts = append(parts, val)
		}
	}
	
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	
	return ""
}

// getBankCardmember gets cardmember name, combining first/last for AmEx format
func getBankCardmember(row []string, colMap map[string]int) string {
	// Try single Cardmember field first
	if name := getColumn(row, colMap, "Cardmember"); name != "" {
		return name
	}
	
	// For AmEx format: combine Last, First, Middle names
	last := getColumn(row, colMap, "Cardmember Last Name")
	first := getColumn(row, colMap, "Cardmember First Name")
	middle := getColumn(row, colMap, "Cardmember Middle Name")
	
	parts := make([]string, 0)
	if last != "" {
		parts = append(parts, last)
	}
	if first != "" {
		parts = append(parts, first)
	}
	if middle != "" {
		parts = append(parts, middle)
	}
	
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	
	return ""
}

func transactionsEqual(t1, t2 models.BankTransaction) bool {
	return t1.Date.Equal(t2.Date) && t1.Amount == t2.Amount && t1.Description == t2.Description
}

func foundationTransactionsEqual(t1, t2 models.FoundationTransaction) bool {
	return t1.Date.Equal(t2.Date) && t1.Amount == t2.Amount && t1.TransactionNumber == t2.TransactionNumber
}

func removeBank(slice []models.BankTransaction, item models.BankTransaction) []models.BankTransaction {
	result := make([]models.BankTransaction, 0)
	for _, txn := range slice {
		if !transactionsEqual(txn, item) {
			result = append(result, txn)
		}
	}
	return result
}

func removeFoundation(slice []models.FoundationTransaction, item models.FoundationTransaction) []models.FoundationTransaction {
	result := make([]models.FoundationTransaction, 0)
	for _, txn := range slice {
		if !foundationTransactionsEqual(txn, item) {
			result = append(result, txn)
		}
	}
	return result
}
