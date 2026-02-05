package services

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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

	return &models.ReconciliationReport{
		MatchedTransactions:        matched,
		BankOnlyTransactions:       finalBankUnmatched,
		FoundationOnlyTransactions: finalFoundationUnmatched,
		PotentialMatches:           potentialMatches,
		VoidPairs:                  voidPairs,
		AmbiguousVoids:             e.ambiguousVoids,
		GeneratedAt:                time.Now(),
	}
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

	// Check date match
	dateDiff := int(math.Abs(float64(bankTxn.Date.Sub(foundTxn.Date).Hours() / 24)))
	if dateDiff > e.DateToleranceDays {
		return false
	}

	return true
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

	// Date proximity (50% weight)
	dateDiff := int(math.Abs(float64(bankTxn.Date.Sub(foundTxn.Date).Hours() / 24)))
	if dateDiff == 0 {
		score += 0.5
	} else if dateDiff <= 1 {
		score += 0.4
	} else if dateDiff <= 3 {
		score += 0.3
	} else if dateDiff <= 7 {
		score += 0.1
	}

	return score
}

// LoadBankTransactions loads bank transactions from CSV or Excel file
func LoadBankTransactions(filePath string) ([]models.BankTransaction, error) {
	ext := strings.ToLower(filePath[strings.LastIndex(filePath, "."):])
	
	if ext == ".xlsx" {
		return loadBankFromExcel(filePath)
	} else if ext == ".xls" {
		return nil, fmt.Errorf("legacy .xls format is not supported - please convert to .xlsx or save as CSV")
	}
	return loadBankFromCSV(filePath)
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

	// Map headers
	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.BankTransaction, 0)

	for i := 1; i < len(records); i++ {
		row := records[i]
		
		dateStr := getColumn(row, colMap, "Date")
		amountStr := getColumn(row, colMap, "Amount")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		amount, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
		if err != nil {
			continue
		}

		transactions = append(transactions, models.BankTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getColumn(row, colMap, "Description"),
				Reference:   getColumn(row, colMap, "Transaction Reference No."),
			},
			CardmemberName: getColumn(row, colMap, "Cardmember"),
			CardAccount:    getColumn(row, colMap, "Card Account No."),
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

	// Map headers
	header := rows[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	transactions := make([]models.BankTransaction, 0)

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		
		dateStr := getColumn(row, colMap, "Date")
		amountStr := getColumn(row, colMap, "Amount")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		amount, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
		if err != nil {
			continue
		}

		transactions = append(transactions, models.BankTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getColumn(row, colMap, "Description"),
				Reference:   getColumn(row, colMap, "Transaction Reference No."),
			},
			CardmemberName: getColumn(row, colMap, "Cardmember"),
			CardAccount:    getColumn(row, colMap, "Card Account No."),
		})
	}

	return transactions, nil
}

// LoadFoundationTransactions loads foundation transactions from CSV or Excel file
func LoadFoundationTransactions(filePath string) ([]models.FoundationTransaction, error) {
	ext := strings.ToLower(filePath[strings.LastIndex(filePath, "."):])
	
	if ext == ".xlsx" {
		return loadFoundationFromExcel(filePath)
	} else if ext == ".xls" {
		return nil, fmt.Errorf("legacy .xls format is not supported - please convert to .xlsx or save as CSV")
	}
	return loadFoundationFromCSV(filePath)
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
		
		dateStr := getColumn(row, colMap, "Date")
		amountStr := getColumn(row, colMap, "Amount")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		amount, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
		if err != nil {
			continue
		}

		trxNoStr := getColumn(row, colMap, "Trx No")
		var trxNo int
		if trxNoStr != "" {
			trxNo, _ = strconv.Atoi(strings.TrimSpace(trxNoStr))
		}

		transactions = append(transactions, models.FoundationTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getColumn(row, colMap, "Description"),
				Reference:   getColumn(row, colMap, "Trx No"),
			},
			VendorName:        getColumn(row, colMap, "Vendor Name"),
			TransactionNumber: trxNo,
			JobNumber:         getColumn(row, colMap, "Job No"),
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
		
		dateStr := getColumn(row, colMap, "Date")
		amountStr := getColumn(row, colMap, "Amount")

		if dateStr == "" || amountStr == "" {
			continue
		}

		date, err := parseDate(dateStr)
		if err != nil {
			continue
		}

		amount, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
		if err != nil {
			continue
		}

		trxNoStr := getColumn(row, colMap, "Trx No")
		var trxNo int
		if trxNoStr != "" {
			trxNo, _ = strconv.Atoi(strings.TrimSpace(trxNoStr))
		}

		transactions = append(transactions, models.FoundationTransaction{
			Transaction: models.Transaction{
				Date:        date,
				Amount:      amount,
				Description: getColumn(row, colMap, "Description"),
				Reference:   getColumn(row, colMap, "Trx No"),
			},
			VendorName:        getColumn(row, colMap, "Vendor Name"),
			TransactionNumber: trxNo,
			JobNumber:         getColumn(row, colMap, "Job No"),
		})
	}

	return transactions, nil
}

// Helper functions
func getColumn(row []string, colMap map[string]int, colName string) string {
	if idx, ok := colMap[colName]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
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
