package main

import (
	"fmt"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

const bankFile = "/Users/will/ITWill/JBS/jbs-internal-portal/docs/data/reconciliation-samples/Statement_1002_Feb_2026 (2).xls"

func main() {
	// Test 1: Can excelize open this XLS file?
	fmt.Println("=== Testing excelize on .xls file ===")
	f, xlsxErr := excelize.OpenFile(bankFile)
	if xlsxErr != nil {
		fmt.Printf("excelize FAILED: %v\n", xlsxErr)
	} else {
		defer f.Close()
		sheets := f.GetSheetList()
		fmt.Printf("excelize SUCCESS! Sheets: %v\n", sheets)
		for _, sheet := range sheets {
			rows, _ := f.GetRows(sheet)
			fmt.Printf("  Sheet %q: %d rows\n", sheet, len(rows))
			// print first 3 rows
			for i, row := range rows {
				if i >= 3 {
					break
				}
				fmt.Printf("    Row %d: %v\n", i, row)
			}
		}
	}

	// Test 2: Does extrame/xls find the header row?
	fmt.Println("\n=== Testing extrame/xls header detection ===")
	xlsFile, xlsErr := xls.Open(bankFile, "utf-8")
	if xlsErr != nil {
		fmt.Printf("extrame/xls open FAILED: %v\n", xlsErr)
		return
	}
	fmt.Printf("extrame/xls opened OK, NumSheets: %d\n", xlsFile.NumSheets())
	sheet := xlsFile.GetSheet(0)
	fmt.Printf("Sheet MaxRow: %d\n", sheet.MaxRow)

	rows := make([][]string, 0)
	for i := 0; i <= int(sheet.MaxRow); i++ {
		if rowData := xlsRowSafe(sheet, i); rowData != nil {
			rows = append(rows, rowData)
		}
	}
	fmt.Printf("Rows after xlsRowSafe: %d\n", len(rows))

	// Find header
	headerIdx := findBankHeaderRow(rows)
	fmt.Printf("Header row index: %d\n", headerIdx)
	if headerIdx >= 0 {
		fmt.Printf("Header: %v\n", rows[headerIdx][:min(20, len(rows[headerIdx]))])
	}
}

func findBankHeaderRow(rows [][]string) int {
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "Transaction Date" || cell == "Transaction Amount USD" {
				return i
			}
		}
	}
	return -1
}

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
	rowData = make([]string, lastCol)
	for j := int(row.FirstCol()); j < lastCol; j++ {
		func() {
			defer func() { recover() }() //nolint:errcheck
			rowData[j] = row.Col(j)
		}()
	}
	return rowData
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
	}
	sheet := f.GetSheet(0)

	// Replicate exactly what loadBankFromXLS does
	rows := make([][]string, 0)
	for i := 0; i <= int(sheet.MaxRow); i++ {
		if rowData := xlsRowSafePadded(sheet, i); rowData != nil {
			rows = append(rows, rowData)
		}
	}
	fmt.Printf("Total rows after xlsRowSafe: %d\n", len(rows))

	// Find header row
	headerIdx := -1
	for i, row := range rows {
		for _, cell := range row {
			if strings.TrimSpace(cell) == "Transaction Date" || strings.TrimSpace(cell) == "Transaction Amount USD" {
				headerIdx = i
				break
			}
		}
		if headerIdx >= 0 {
			break
		}
	}
	fmt.Printf("Header row index: %d\n", headerIdx)
	if headerIdx < 0 {
		fmt.Println("ERROR: header not found")
		return
	}

	// Build colMap
	header := rows[headerIdx]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}
	fmt.Printf("colMap[Transaction Date]=%d, colMap[Transaction Amount USD]=%d\n",
		colMap["Transaction Date"], colMap["Transaction Amount USD"])

	// Simulate processing loop, count why rows are dropped
	loaded := 0
	droppedNoDateNoAmount := 0
	droppedFastPathBadDate := 0
	droppedFastPathBadAmount := 0  
	droppedScanFallbackFail := 0
	loadedViaFastPath := 0
	loadedViaFallback := 0
	fastPathGarbageAmount := 0  // loaded but with suspicious amount

	for i := headerIdx + 1; i < len(rows); i++ {
		row := rows[i]

		// Asterisk check at col 14
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

		// Get date and amount
		dateStr := getCol(row, colMap, "Transaction Date")
		amountStr := getCol(row, colMap, "Transaction Amount USD")

		if dateStr == "" && amountStr == "" {
			droppedNoDateNoAmount++
			continue
		}

		dateOk := false
		var date time.Time
		if dateStr != "" {
			if t, err := time.Parse("01/02/2006", dateStr); err == nil {
				date = t
				dateOk = true
			} else if t, err := time.Parse("1/2/2006", dateStr); err == nil {
				date = t
				dateOk = true
			}
		}

		amountOk := false
		var amount float64
		if amountStr != "" {
			clean := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(amountStr, "$", ""), ",", ""))
			if v, err := strconv.ParseFloat(clean, 64); err == nil {
				amount = v
				amountOk = true
			}
		}

		if dateOk && amountOk {
			// Fast path — check if amount is suspicious (not a real dollar amount)
			if amount > 1_000_000 || (amount > 0 && amount == float64(int64(amount)) && amount > 10_000) {
				fastPathGarbageAmount++
				// still count as loaded per current code
			}
			loaded++
			loadedViaFastPath++
			_ = date
		} else if !dateOk && !amountOk {
			// Both bad — try scan fallback
			scanOk := scanForDollarAmount(row)
			if scanOk {
				loaded++
				loadedViaFallback++
			} else {
				droppedScanFallbackFail++
			}
		} else if !dateOk {
			droppedFastPathBadDate++
			// Would try scan fallback
			scanOk := scanForDollarAmount(row)
			if scanOk {
				loaded++
				loadedViaFallback++
			} else {
				droppedScanFallbackFail++
			}
		} else {
			// dateOk but !amountOk
			droppedFastPathBadAmount++
			// Would try scan fallback
			scanOk := scanForDollarAmount(row)
			if scanOk {
				loaded++
				loadedViaFallback++
			} else {
				droppedScanFallbackFail++
			}
		}
	}

	fmt.Printf("\n=== Processing summary ===\n")
	fmt.Printf("  Total data rows processed: %d\n", len(rows)-headerIdx-1)
	fmt.Printf("  Loaded total: %d\n", loaded)
	fmt.Printf("    Via fast path: %d (of which garbage amount: %d)\n", loadedViaFastPath, fastPathGarbageAmount)
	fmt.Printf("    Via scan fallback: %d\n", loadedViaFallback)
	fmt.Printf("  DROPPED: %d total\n", len(rows)-headerIdx-1-loaded)
	fmt.Printf("    No date AND no amount: %d\n", droppedNoDateNoAmount)
	fmt.Printf("    Bad date only (fast path): %d\n", droppedFastPathBadDate)
	fmt.Printf("    Bad amount only (fast path): %d\n", droppedFastPathBadAmount)
	fmt.Printf("    Scan fallback failed: %d\n", droppedScanFallbackFail)
}

func getCol(row []string, colMap map[string]int, name string) string {
	idx, ok := colMap[name]
	if !ok || idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// scanForDollarAmount checks if any cell in row STARTS with $ or -$
func scanForDollarAmount(row []string) bool {
	for _, cell := range row {
		c := strings.TrimSpace(cell)
		if strings.HasPrefix(c, "$") || strings.HasPrefix(c, "-$") {
			// verify it's actually a number
			clean := strings.ReplaceAll(strings.ReplaceAll(c, "$", ""), ",", "")
			if _, err := strconv.ParseFloat(clean, 64); err == nil {
				return true
			}
		}
	}
	return false
}

// xlsRowSafePadded mirrors exactly what loadBankFromXLS uses
func xlsRowSafePadded(sheet *xls.WorkSheet, i int) (rowData []string) {
	defer func() { recover() }() //nolint:errcheck
	row := sheet.Row(i)
	if row == nil {
		return nil
	}
	lastCol := int(row.LastCol())
	if lastCol <= 0 {
		return nil
	}
	rowData = make([]string, lastCol)
	for j := int(row.FirstCol()); j < lastCol; j++ {
		func() {
			defer func() { recover() }() //nolint:errcheck
			rowData[j] = row.Col(j)
		}()
	}
	return rowData
}

func dumpRow(sheet *xls.WorkSheet, i int) {
	defer func() { recover() }() //nolint:errcheck
	row := sheet.Row(i)
	if row == nil {
		fmt.Printf("  Row %d: nil\n", i)
		return
	}
	fmt.Printf("  Row %d (first=%d last=%d): ", i, row.FirstCol(), row.LastCol())
	for j := int(row.FirstCol()); j < int(row.LastCol()) && j < 35; j++ {
		v := strings.TrimSpace(row.Col(j))
		if v != "" {
			fmt.Printf("[%d]=%q ", j, v)
		}
	}
	fmt.Println()
}
