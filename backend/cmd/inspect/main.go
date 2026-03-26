package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

func safeRow(sheet *xls.WorkSheet, i int) (cells []string) {
	defer func() { recover() }() //nolint:errcheck
	row := sheet.Row(i)
	if row == nil {
		return nil
	}
	cells = make([]string, 0, row.LastCol()-row.FirstCol())
	for j := row.FirstCol(); j < row.LastCol(); j++ {
		cells = append(cells, row.Col(j))
	}
	return cells
}

func main() {
	// Dump all raw XLS rows and search for 8185
	inspectXLSRaw("/Users/will/ITWill/JBS/jbs-internal-portal/docs/data/reconciliation-samples/Statement_1002_Feb_2026 (2).xls")
}

func inspectXLSX(path string) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		// might be legacy XLS masquerading as xlsx
		fmt.Printf("XLSX open error for %s: %v\n", path, err)
		return
	}
	defer f.Close()
	sheets := f.GetSheetList()
	fmt.Printf("\n=== XLSX: %s ===\nSheets: %v\n", path, sheets)
	for _, sheet := range sheets {
		rows, _ := f.GetRows(sheet)
		if len(rows) == 0 {
			continue
		}
		fmt.Printf("Sheet %q: %d rows\n", sheet, len(rows))
		fmt.Printf("  Headers: %v\n", rows[0])
		for i := 1; i <= 5 && i < len(rows); i++ {
			fmt.Printf("  Row %d: %v\n", i, rows[i])
		}
	}
}

func inspectXLS(path string) {
	fmt.Printf("\n=== XLS: %s ===\n", path)
	f, err := excelize.OpenFile(path)
	if err == nil {
		defer f.Close()
		sheets := f.GetSheetList()
		fmt.Printf("(Opened as XLSX) Sheets: %v\n", sheets)
		for _, sheet := range sheets {
			rows, _ := f.GetRows(sheet)
			fmt.Printf("Sheet %q: %d rows\n", sheet, len(rows))
			// Find header row (look for Transaction Date)
			headerIdx := -1
			for i, row := range rows {
				for _, cell := range row {
					if strings.TrimSpace(cell) == "Transaction Date" {
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
				fmt.Println("Header row not found")
				return
			}
			header := rows[headerIdx]
			fmt.Printf("Header: %v\n\n", header[:min(20, len(header))])

			// Find Transaction Date column index
			txnDateIdx := -1
			amtIdx := -1
			for i, h := range header {
				switch strings.TrimSpace(h) {
				case "Transaction Date":
					txnDateIdx = i
				case "Transaction Amount USD":
					amtIdx = i
				}
			}
			fmt.Printf("Transaction Date col: %d, Amount col: %d\n", txnDateIdx, amtIdx)

			// Collect all unique dates and their counts
			dateCounts := make(map[string]int)
			totalData := 0
			for _, row := range rows[headerIdx+1:] {
				if txnDateIdx >= 0 && txnDateIdx < len(row) {
					d := strings.TrimSpace(row[txnDateIdx])
					if d != "" {
						dateCounts[d]++
						totalData++
					}
				}
			}
			fmt.Printf("\nData rows with Transaction Date: %d\n", totalData)
			fmt.Printf("Unique dates:\n")
			// Sort dates
			sortedDates := make([]string, 0, len(dateCounts))
			for d := range dateCounts {
				sortedDates = append(sortedDates, d)
			}
			sort.Strings(sortedDates)
			for _, d := range sortedDates {
				fmt.Printf("  %s: %d transactions\n", d, dateCounts[d])
			}
			// Show a few sample rows
			fmt.Println("\nFirst 5 data rows:")
			for i, row := range rows[headerIdx+1:] {
				if i >= 5 {
					break
				}
				if txnDateIdx < len(row) && amtIdx < len(row) {
					desc := ""
					if txnDateIdx+3 < len(row) {
						desc = strings.TrimSpace(row[txnDateIdx+3])
					}
					fmt.Printf("  Row %d: date=%q amount=%q desc=%q\n",
						i+1, row[txnDateIdx], row[amtIdx], desc)
				}
			}
		}
		return
	}
	// Legacy XLS path
	xlsFile, xlsErr := xls.Open(path, "utf-8")
	if xlsErr != nil {
		fmt.Printf("XLS open error: %v\n", xlsErr)
		return
	}
	fmt.Printf("(Opened as legacy XLS) NumSheets: %d\n", xlsFile.NumSheets())
	sheet := xlsFile.GetSheet(0)
	if sheet == nil {
		return
	}
	fmt.Printf("MaxRow: %d\n", sheet.MaxRow)
	printed := 0
	for i := 0; i <= int(sheet.MaxRow) && printed < 30; i++ {
		cells := safeRow(sheet, i)
		if cells != nil && strings.Join(cells, "") != "" {
			fmt.Printf("  Row %d: %v\n", i, cells)
			printed++
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func inspectXLSRaw(path string) {
	fmt.Printf("\n=== RAW XLS SEARCH: %s ===\n", path)
	xlsFile, err := xls.Open(path, "utf-8")
	if err != nil {
		// Try as XLSX
		f, err2 := excelize.OpenFile(path)
		if err2 != nil {
			fmt.Printf("Cannot open: %v / %v\n", err, err2)
			return
		}
		defer f.Close()
		sheets := f.GetSheetList()
		for _, sheet := range sheets {
			rows, _ := f.GetRows(sheet)
			for i, row := range rows {
				for _, cell := range row {
					if strings.Contains(cell, "8185") {
						fmt.Printf("  XLSX Row %d: %v\n", i, row)
					}
				}
			}
		}
		return
	}
	sheet := xlsFile.GetSheet(0)
	if sheet == nil {
		fmt.Println("no sheet")
		return
	}
	fmt.Printf("MaxRow: %d\n", sheet.MaxRow)
	foundCount := 0
	for i := 0; i <= int(sheet.MaxRow); i++ {
		cells := safeRow(sheet, i)
		if cells == nil {
			continue
		}
		for _, c := range cells {
			if strings.Contains(c, "8185") {
				fmt.Printf("  Row %d: %v\n", i, cells)
				foundCount++
				break
			}
		}
	}
	if foundCount == 0 {
		fmt.Println("  No row contains '8185' in any cell")
		// Also dump the amount column for all rows to see what values exist
		fmt.Println("\n  Dumping all rows with amount-like values > $5000:")
		// Find header first
		rows := make([][]string, 0)
		for i := 0; i <= int(sheet.MaxRow); i++ {
			if row := safeRow(sheet, i); row != nil {
				rows = append(rows, row)
			}
		}
		amtIdx := -1
		dateIdx := -1
		for i, row := range rows {
			for j, cell := range row {
				if strings.TrimSpace(cell) == "Transaction Amount USD" {
					amtIdx = j
					fmt.Printf("  Header row: %d, Amount col: %d\n", i, j)
				}
				if strings.TrimSpace(cell) == "Transaction Date" {
					dateIdx = j
				}
			}
			if amtIdx >= 0 {
				break
			}
		}
		for _, row := range rows {
			if amtIdx >= 0 && amtIdx < len(row) {
				raw := strings.TrimSpace(row[amtIdx])
				clean := strings.ReplaceAll(strings.ReplaceAll(raw, "$", ""), ",", "")
				if v, perr := fmt.Sscanf(clean, "%f", new(float64)); v == 1 && perr == nil {
					var amt float64
					fmt.Sscanf(clean, "%f", &amt)
					if amt > 5000 {
						date := ""
						if dateIdx >= 0 && dateIdx < len(row) {
							date = row[dateIdx]
						}
						fmt.Printf("  Date=%s  Amount=%s\n", date, raw)
					}
				}
			}
		}
	}
}
