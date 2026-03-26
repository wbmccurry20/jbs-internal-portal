package services

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
	"github.com/xuri/excelize/v2"
)

var requiredConcurColumns = []string{
	"Employee Name",
	"Expense Type",
	"Account Code",
	"Transaction Date",
	"SUM Allocation Claimed Amount (USD)",
}

type ConcurConverter struct {
	VendorID string
}

func NewConcurConverter(vendorID string) *ConcurConverter {
	if vendorID == "" {
		vendorID = "138" // Default vendor ID
	}
	return &ConcurConverter{VendorID: vendorID}
}

// ProcessFile converts a Concur Excel file to Foundation CSV format
func (c *ConcurConverter) ProcessFile(inputPath, outputPath string) (*models.ConversionResult, error) {
	result := &models.ConversionResult{
		Issues: make([]string, 0),
	}

	// Open the Excel file
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows found in sheet")
	}

	// Validate and map columns
	header := rows[0]
	columnMap, err := c.validateAndMapColumns(header)
	if err != nil {
		return nil, err
	}

	result.Issues = append(result.Issues, "Processing all expense rows (approval status ignored for conversion)")

	// Parse expenses
	expenses := make([]models.ConcurExpense, 0)
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		
		expense, err := c.parseExpenseRow(row, columnMap, rowIdx+1)
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("Row %d: %s", rowIdx+1, err.Error()))
			result.RowsSkipped++
			continue
		}

		expenses = append(expenses, *expense)
		result.RowsProcessed++
	}

	// Create Foundation CSV
	err = c.createFoundationCSV(expenses, outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Foundation CSV: %w", err)
	}

	result.OutputPath = outputPath
	return result, nil
}

// validateAndMapColumns checks for required columns and creates a mapping
func (c *ConcurConverter) validateAndMapColumns(header []string) (map[string]int, error) {
	columnMap := make(map[string]int)
	
	for idx, col := range header {
		columnMap[col] = idx
	}

	missing := make([]string, 0)
	for _, required := range requiredConcurColumns {
		if _, exists := columnMap[required]; !exists {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		available := strings.Join(header, ", ")
		return nil, fmt.Errorf("missing required columns: %s\n\nAvailable columns: %s", 
			strings.Join(missing, ", "), available)
	}

	return columnMap, nil
}

// parseExpenseRow parses a single expense row
func (c *ConcurConverter) parseExpenseRow(row []string, columnMap map[string]int, rowNum int) (*models.ConcurExpense, error) {
	getValue := func(colName string) string {
		if idx, ok := columnMap[colName]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	// Skip empty rows
	employeeName := getValue("Employee Name")
	if employeeName == "" {
		return nil, fmt.Errorf("empty row (no employee name)")
	}

	// Parse transaction date
	dateStr := getValue("Transaction Date")
	if dateStr == "" {
		return nil, fmt.Errorf("missing transaction date")
	}

	transDate, err := parseDate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format '%s': %w", dateStr, err)
	}

	// Parse amount
	amountStr := getValue("SUM Allocation Claimed Amount (USD)")
	if amountStr == "" {
		return nil, fmt.Errorf("missing claimed amount")
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount '%s': %w", amountStr, err)
	}

	expense := &models.ConcurExpense{
		EmployeeName:     employeeName,
		ExpenseType:      getValue("Expense Type"),
		AccountCode:      getValue("Account Code"),
		TransactionDate:  transDate,
		CostCode:         getValue("Allocation:Cost Code (Code)"),
		CostClass:        getValue("Allocation:Cost Class (Code)"),
		JobCode:          getValue("Allocation:Job (Code)"),
		ClaimedAmountUSD: amount,
	}

	return expense, nil
}

// parseDate attempts to parse date in multiple formats
func parseDate(dateStr string) (time.Time, error) {
	// Trim any surrounding whitespace
	dateStr = strings.TrimSpace(dateStr)
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"02/01/2006",
		"2/1/2006",
		"2006/01/02",
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"1/2/2006 15:04:05",
		// 2-digit year formats (e.g. Concur export: "2/6/26 00:00")
		"1/2/06 15:04",
		"01/02/06 15:04",
		"1/2/06",
		"01/02/06",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	// Fallback: Excel serial date number (days since 1899-12-30).
	// SheetJS or excelize may emit raw serial numbers for date cells when
	// the cell lacks a display format (e.g. "46082" for 02/28/2026).
	if serial, err := strconv.ParseFloat(dateStr, 64); err == nil && serial > 40000 && serial < 60000 {
		// Excel epoch: serial 1 = January 1, 1900, but the Lotus 1-2-3 bug
		// treats 1900 as a leap year (inserting a phantom Feb 29, 1900).
		// For serial > 60, subtract 1 to compensate.
		epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		days := int(serial)
		if days > 60 {
			days-- // compensate for Lotus 1-2-3 leap year bug
		}
		t := epoch.AddDate(0, 0, days)
		if t.Year() >= 2020 && t.Year() <= 2040 {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %q", dateStr)
}

// createFoundationCSV generates the Foundation import CSV file
func (c *ConcurConverter) createFoundationCSV(expenses []models.ConcurExpense, outputPath string) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write template header rows (rows 1-12)
	headerRows := c.createFoundationHeader()
	for _, row := range headerRows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write header row: %w", err)
		}
	}

	// Write expense data rows
	for _, expense := range expenses {
		row := c.expenseToFoundationRow(expense)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write expense row: %w", err)
		}
	}

	return nil
}

// createFoundationHeader creates the Foundation CSV template header (12 rows)
func (c *ConcurConverter) createFoundationHeader() [][]string {
	rows := make([][]string, 0)

	// Row 1: TRUE marker
	row1 := make([]string, 48)
	row1[0] = "TRUE"
	rows = append(rows, row1)

	// Rows 2-3: Empty
	for i := 0; i < 2; i++ {
		rows = append(rows, make([]string, 48))
	}

	// Row 4: HEADER/DETAIL labels
	row4 := make([]string, 48)
	row4[5] = "HEADER"
	row4[26] = "DETAIL"
	rows = append(rows, row4)

	// Rows 5-9: Empty
	for i := 0; i < 5; i++ {
		rows = append(rows, make([]string, 48))
	}

	// Row 10: Field names
	fieldNames := []string{
		"Field Name     * Indicates Required",
		"* Single or Multiple Distribution Detail lines",
		"Voucher Reference ID",
		"* Vendor No",
		"* Invoice Date",
		"* Transaction Date",
		"$  Header Amount",
		"Invoice Description",
		"Invoice No",
		"Due Date",
		"Retainage Amount",
		"PO/Sub No",
		"Voucher type",
		"Payment Type",
		"Check No",
		"Check Amount /\nPayment Amount",
		"Check Date / \nPayment Date",
		"Terms No",
		"Discount Date",
		"Discount Amount",
		"Scanned Invoice Link",
		"*  G/L Expense",
		"Div 1",
		"Div 2",
		"Div 3",
		"Div 4",
		"Job No",
		"Phase No",
		"Cost Code No",
		"Cost Class No",
		"$  Distribution Amount",
		"Description",
		"Equipment No",
		"Service Code",
		"Units",
		"Unit Price",
		"Original Contract",
		"Change Order",
		"Committed Cost",
		"Cost To Date",
		"Revenue",
		"Overhead",
		"Cash Flow",
		"Billed",
		"Billable Amount",
		"Sales Tax Amount",
		"Committed Sales Tax",
		"Sales Tax Code",
	}
	rows = append(rows, fieldNames)

	// Row 11: Field types
	fieldTypes := []string{
		"Field Type",
		"Text",
		"Text",
		"Text",
		"Date",
		"Date",
		"Currency",
		"Text",
		"Text",
		"Date",
		"Currency",
		"Text",
		"Text",
		"Text",
		"Text",
		"Currency",
		"Date",
		"Text",
		"Date",
		"Currency",
		"Text",
		"Text",
		"Text",
		"Text",
		"Text",
		"Text",
		"Text",
		"Text",
		"Text",
		"Text",
		"Currency",
		"Text",
		"Text",
		"Text",
		"Numeric",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Currency",
		"Text",
	}
	rows = append(rows, fieldTypes)

	// Row 12: Max Lengths
	row12 := make([]string, 48)
	row12[0] = "Max Length"
	rows = append(rows, row12)

	return rows
}

// expenseToFoundationRow converts a ConcurExpense to a Foundation CSV row
func (c *ConcurConverter) expenseToFoundationRow(expense models.ConcurExpense) []string {
	row := make([]string, 48)

	// Format date as M/D/YYYY (no leading zeros)
	dateStr := fmt.Sprintf("%d/%d/%d", 
		expense.TransactionDate.Month(),
		expense.TransactionDate.Day(),
		expense.TransactionDate.Year())

	// HEADER fields
	row[1] = "N"                                 // Single line
	row[3] = c.VendorID                          // Vendor No
	row[4] = dateStr                             // Invoice Date
	row[5] = dateStr                             // Transaction Date
	row[6] = fmt.Sprintf("%.2f", expense.ClaimedAmountUSD) // Header Amount
	row[7] = expense.EmployeeName                // Invoice Description
	row[12] = "P"                                // Voucher Type (Prepaid)
	row[13] = "CRC"                              // Payment Type (Credit Card)
	row[15] = fmt.Sprintf("%.2f", expense.ClaimedAmountUSD) // Check Amount
	row[16] = dateStr                            // Check Date

	// DETAIL fields
	row[21] = expense.AccountCode                // G/L Expense
	row[26] = expense.JobCode                    // Job No
	row[28] = expense.CostCode                   // Cost Code No
	row[29] = expense.CostClass                  // Cost Class No
	row[30] = fmt.Sprintf("%.2f", expense.ClaimedAmountUSD) // Distribution Amount

	return row
}
