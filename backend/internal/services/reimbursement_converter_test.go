package services

import (
	"bytes"
	"encoding/csv"
	"os"
	"strings"
	"testing"
)

func TestConvertReimbursementCSVAccountingExtract(t *testing.T) {
	fixture, err := os.ReadFile("../../../docs/data/USD_Synchronized_Accounting_Extract_2_20261016.csv")
	if err != nil {
		t.Fatal(err)
	}
	source, err := csv.NewReader(bytes.NewReader(fixture)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	conversion, err := ConvertReimbursementCSV(bytes.NewReader(fixture), "Employee Reimbursement")
	if err != nil {
		t.Fatal(err)
	}
	output, err := csv.NewReader(bytes.NewReader(conversion.CSV)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if conversion.LineCount != len(source)-1 || len(output) != len(source)-1+14 {
		t.Fatalf("source rows=%d output=%d lines=%d", len(source)-1, len(output), conversion.LineCount)
	}
	golden, err := os.ReadFile("../../../docs/data/9.22 Wyatt Bell Reimb.csv")
	if err != nil {
		t.Fatal(err)
	}
	template, err := csv.NewReader(bytes.NewReader(golden)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for rowIndex := 0; rowIndex < 14; rowIndex++ {
		for columnIndex, expected := range template[rowIndex] {
			if rowIndex == 0 && columnIndex == 0 {
				expected = strings.TrimPrefix(expected, "\ufeff")
			}
			if output[rowIndex][columnIndex] != expected {
				t.Errorf("template row %d column %d = %q, want %q", rowIndex+1, columnIndex+1, output[rowIndex][columnIndex], expected)
			}
		}
	}
	expectedLines := make(map[string]int)
	var expectedTotal, actualTotal int64
	for _, record := range source[1:] {
		merchant := record[13]
		if merchant == "" {
			merchant = record[14]
		}
		if merchant == "" {
			merchant = record[12]
		}
		if merchant == "" {
			merchant = record[9]
		}
		if utf8Length(merchant) > 30 {
			merchant = string([]rune(merchant)[:30])
		}
		merchant = strings.TrimSpace(merchant)
		amount, err := parseReimbursementCents(record[22])
		if err != nil {
			t.Fatal(err)
		}
		expectedTotal += amount
		date, err := formatReimbursementDate(record[10])
		if err != nil {
			t.Fatal(err)
		}
		expectedLines[strings.Join([]string{date, formatReimbursementCents(amount), record[23], record[32], record[30], record[29], merchant}, "|")]++
	}
	for _, record := range output[14:] {
		amount, err := parseReimbursementCents(record[30])
		if err != nil {
			t.Fatal(err)
		}
		actualTotal += amount
		key := strings.Join([]string{record[5], record[30], record[21], record[26], record[28], record[29], record[31]}, "|")
		if expectedLines[key] == 0 {
			t.Errorf("unexpected output line: %s", key)
		}
		expectedLines[key]--
		if record[4] != "10/16/2026" || record[16] != "10/21/2026" || record[3] != "138" || record[12] != "P" || record[13] != "EFT" {
			t.Errorf("incorrect dates or fixed values: %v", record)
		}
	}
	if expectedTotal != actualTotal {
		t.Errorf("total=%d want %d", actualTotal, expectedTotal)
	}
	if conversion.ShortenedDescriptions == 0 {
		t.Error("expected long merchant warnings")
	}
}

func TestReimbursementCSVSeparatesReportsAndRejectsUnsupportedRows(t *testing.T) {
	fixture, err := os.ReadFile("../../../docs/data/USD_Synchronized_Accounting_Extract_2_20261016.csv")
	if err != nil {
		t.Fatal(err)
	}
	source, err := csv.NewReader(bytes.NewReader(fixture)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	convert := func(records [][]string) (ReimbursementConversion, error) {
		var input bytes.Buffer
		writer := csv.NewWriter(&input)
		if err := writer.WriteAll(records); err != nil {
			t.Fatal(err)
		}
		return ConvertReimbursementCSV(&input, "")
	}
	for _, identityColumn := range []int{1, 4} {
		first := append([]string(nil), source[1]...)
		second := append([]string(nil), first...)
		second[identityColumn] = "different identity"
		result, err := convert([][]string{source[0], first, second})
		if err != nil {
			t.Fatal(err)
		}
		if result.VoucherCount != 2 {
			t.Errorf("identity column %d merged unrelated vouchers", identityColumn)
		}
	}
	for _, change := range []struct {
		column int
		value  string
	}{{5, "EUR"}, {11, "Y"}, {15, "CARD"}, {21, "Company"}, {24, "CR"}, {22, "bad amount"}, {22, "1.234"}, {22, "10000000000"}, {27, ""}} {
		record := append([]string(nil), source[1]...)
		record[change.column] = change.value
		if _, err := convert([][]string{source[0], record}); err == nil {
			t.Errorf("accepted invalid column %d=%q", change.column, change.value)
		}
	}
}

func TestConvertReimbursementWorkbookUsesHeaderCompatibleGroups(t *testing.T) {
	input, err := os.Open("../../../docs/data/9.22 Wyatt Bell Reimb.xlsm")
	if err != nil {
		t.Skipf("sample workbook unavailable: %v", err)
	}
	defer input.Close()

	conversion, err := ConvertReimbursementWorkbook(input, "Employee Reimbursement")
	if err != nil {
		t.Fatalf("ConvertReimbursementWorkbook() error = %v", err)
	}
	if conversion.LineCount != 13 {
		t.Fatalf("LineCount = %d, want 13", conversion.LineCount)
	}
	if conversion.VoucherCount != 4 {
		t.Fatalf("VoucherCount = %d, want 4", conversion.VoucherCount)
	}

	records, err := csv.NewReader(strings.NewReader(string(conversion.CSV))).ReadAll()
	if err != nil {
		t.Fatalf("parse output CSV: %v", err)
	}
	if len(records) != 27 {
		t.Fatalf("output has %d rows, want 27 template and detail rows", len(records))
	}

	expectedCounts := map[string]int{
		"5060|2026-502|1":     6,
		"5060|2026-1103|1":    5,
		"5160|2026-502|1":     1,
		"5160|2026-502|90000": 1,
	}
	expectedTotals := map[string]string{
		"5060|2026-502|1":     "395.98",
		"5060|2026-1103|1":    "306.50",
		"5160|2026-502|1":     "37.51",
		"5160|2026-502|90000": "118.35",
	}
	expectedMerchants := map[string]int{
		"MAVERIK":        2,
		"QuikTrip":       5,
		"Circle K":       1,
		"Loves#253":      1,
		"KROGER FUEL":    1,
		"Flying J":       1,
		"ACE Hardware":   1,
		"The Home Depot": 1,
	}
	groupCounts := make(map[string]int)
	groupReferences := make(map[string]string)
	groupReferenceCounts := make(map[string]int)
	merchantCounts := make(map[string]int)
	var distributionTotal int64

	for _, row := range records[14:] {
		if len(row) != reimbursementColumnCount {
			t.Fatalf("output data row has %d columns, want %d", len(row), reimbursementColumnCount)
		}
		key := strings.Join([]string{row[21], row[26], row[28]}, "|")
		groupCounts[key]++
		if row[7] != "Employee Reimbursement" {
			t.Errorf("invoice description = %q, want common description", row[7])
		}
		if row[3] != "138" || row[12] != "P" || row[13] != "EFT" {
			t.Errorf("fixed fields not applied: vendor=%q voucher=%q payment=%q", row[3], row[12], row[13])
		}
		if row[4] != "9/22/2026" || row[5] != "9/22/2026" || row[16] != "9/22/2026" {
			t.Errorf("dates were not normalized: invoice=%q transaction=%q payment=%q", row[4], row[5], row[16])
		}
		if row[31] == "" {
			t.Errorf("merchant detail description is empty")
		}
		merchantCounts[row[31]]++
		cents, err := parseReimbursementCents(row[30])
		if err != nil {
			t.Fatalf("invalid distribution amount %q: %v", row[30], err)
		}
		distributionTotal += cents
		if row[6] != expectedTotals[key] || row[15] != expectedTotals[key] {
			t.Errorf("group %s totals = header %q payment %q, want %q", key, row[6], row[15], expectedTotals[key])
		}
		if expectedCounts[key] > 1 {
			if row[1] != "Y" || len(row[2]) == 0 || len(row[2]) > 20 {
				t.Errorf("multi-line row has flag/reference %q/%q", row[1], row[2])
			}
			if prior := groupReferences[key]; prior != "" && prior != row[2] {
				t.Errorf("group %s has inconsistent voucher references", key)
			}
			groupReferences[key] = row[2]
			groupReferenceCounts[row[2]]++
		} else if row[1] != "N" || row[2] != "" {
			t.Errorf("single-line row has flag/reference %q/%q", row[1], row[2])
		}
	}

	if len(groupCounts) != len(expectedCounts) {
		t.Fatalf("found %d groups, want %d: %v", len(groupCounts), len(expectedCounts), groupCounts)
	}
	for key, expected := range expectedCounts {
		if groupCounts[key] != expected {
			t.Errorf("group %s has %d rows, want %d", key, groupCounts[key], expected)
		}
	}
	if distributionTotal != 85834 {
		t.Errorf("distribution total = %s, want 858.34", formatReimbursementCents(distributionTotal))
	}
	if len(groupReferenceCounts) != 2 {
		t.Errorf("found %d generated multi-line references, want 2", len(groupReferenceCounts))
	}
	if len(merchantCounts) != len(expectedMerchants) {
		t.Errorf("found %d merchant descriptions, want %d: %v", len(merchantCounts), len(expectedMerchants), merchantCounts)
	}
	for merchant, expected := range expectedMerchants {
		if merchantCounts[merchant] != expected {
			t.Errorf("merchant %q appears %d times, want %d", merchant, merchantCounts[merchant], expected)
		}
	}
	for reference, count := range groupReferenceCounts {
		if count < 2 {
			t.Errorf("reference %q appears on only %d rows", reference, count)
		}
	}
}
