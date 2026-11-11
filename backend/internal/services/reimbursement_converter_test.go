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
	if conversion.VoucherCount != conversion.LineCount {
		t.Errorf("voucher count=%d, want one per line (%d)", conversion.VoucherCount, conversion.LineCount)
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
		if record[1] != "N" || record[2] != "" {
			t.Errorf("distribution flag/reference = %q/%q, want N/empty", record[1], record[2])
		}
		if record[6] != record[30] || record[15] != record[30] {
			t.Errorf("single-line amounts differ: header=%q payment=%q distribution=%q", record[6], record[15], record[30])
		}
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
	for _, identityColumn := range []int{-1, 1, 4} {
		first := append([]string(nil), source[1]...)
		second := append([]string(nil), first...)
		if identityColumn >= 0 {
			second[identityColumn] = "different identity"
		}
		result, err := convert([][]string{source[0], first, second})
		if err != nil {
			t.Fatal(err)
		}
		if result.VoucherCount != 2 {
			t.Errorf("identity column %d merged unrelated vouchers", identityColumn)
		}
		output, err := csv.NewReader(bytes.NewReader(result.CSV)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range output[14:] {
			if row[1] != "N" || row[2] != "" {
				t.Errorf("identity column %d: expected N and blank reference, got %q/%q", identityColumn, row[1], row[2])
			}
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

func TestConvertConcurPayScreenCSVProducesHeaderlessJournal(t *testing.T) {
	header := []string{
		"DATE", "Employee ID", "Employee Default Currency Alpha Code", "Report Processing Payment Date",
		"Report Entry Is Personal Flag", "Report Entry Description", "Report Entry Vendor Description",
		"Report Entry Payment Code Code", "Journal Payee Payment Type Name", "Journal Amount",
		"Journal Account Code", "Journal Debit Or Credit", "Cost Class", "Cost Code", "Job",
	}
	makeRecord := func(values map[string]string) []string {
		record := make([]string, len(header))
		for index, name := range header {
			record[index] = values[name]
		}
		return record
	}

	var input bytes.Buffer
	writer := csv.NewWriter(&input)
	metadata := make([]string, len(header))
	metadata[0] = "H"
	if err := writer.Write(metadata); err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(header); err != nil {
		t.Fatal(err)
	}
	common := map[string]string{
		"Employee ID":                          "employee-1",
		"Employee Default Currency Alpha Code": "USD",
		"Report Processing Payment Date":       "10/30/2026",
		"Report Entry Is Personal Flag":        "N",
		"Report Entry Payment Code Code":       "CASH",
		"Journal Payee Payment Type Name":      "Employee",
		"Journal Debit Or Credit":              "DR",
	}
	first := make(map[string]string, len(common)+5)
	for name, value := range common {
		first[name] = value
	}
	for name, value := range map[string]string{
		"Report Entry Description":        "Team meal, approved",
		"Report Entry Vendor Description": "Vendor fallback",
		"Journal Amount":                  "12.5",
		"Journal Account Code":            "5100",
		"Cost Class":                      "7",
		"Cost Code":                       "1",
		"Job":                             "2026-303",
	} {
		first[name] = value
	}
	second := make(map[string]string, len(common)+5)
	for name, value := range common {
		second[name] = value
	}
	for name, value := range map[string]string{
		"Report Entry Vendor Description": "Tools, materials",
		"Journal Amount":                  "7.25",
		"Journal Account Code":            "5160",
	} {
		second[name] = value
	}
	for _, record := range [][]string{makeRecord(first), makeRecord(second), make([]string, len(header)), {"Notes"}} {
		if err := writer.Write(record); err != nil {
			t.Fatal(err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}

	conversion, err := ConvertReimbursementCSV(bytes.NewReader(input.Bytes()), "")
	if err != nil {
		t.Fatal(err)
	}
	if conversion.Filename != "reimbursement_journal.csv" {
		t.Errorf("Filename = %q, want reimbursement_journal.csv", conversion.Filename)
	}
	if conversion.LineCount != 2 {
		t.Errorf("LineCount = %d, want 2", conversion.LineCount)
	}
	output, err := csv.NewReader(bytes.NewReader(conversion.CSV)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 3 {
		t.Fatalf("output has %d rows, want two expenses and a balancing credit", len(output))
	}
	for rowIndex, row := range output {
		if len(row) != reimbursementJournalColumnCount {
			t.Fatalf("row %d has %d columns, want %d", rowIndex+1, len(row), reimbursementJournalColumnCount)
		}
		if rowIndex < 2 {
			if row[1] != "" || row[2] != "" || row[3] != "" || row[4] != "" ||
				row[8] != "" || row[10] != "" {
				t.Errorf("expense row %d has values in required blank columns: %v", rowIndex+1, row)
			}
		}
	}
	if output[0][0] != "5100" || output[0][5] != "D" || output[0][6] != "12.50" ||
		output[0][7] != "Team meal approved" || output[0][9] != "2026-303" ||
		output[0][11] != "1" || output[0][12] != "7" {
		t.Errorf("incorrect primary-description debit row: %v", output[0])
	}
	if output[1][0] != "5160" || output[1][5] != "D" || output[1][6] != "7.25" ||
		output[1][7] != "Tools materials" {
		t.Errorf("incorrect fallback-description debit row: %v", output[1])
	}
	if output[2][0] != "1002" || output[2][5] != "C" || output[2][6] != "19.75" ||
		output[2][7] != "10.30.26 reimbursement" {
		t.Errorf("incorrect balancing credit row: %v", output[2])
	}
}

func TestConvertReimbursementWorkbookUsesSingleLineDistributions(t *testing.T) {
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
	if conversion.VoucherCount != 13 {
		t.Fatalf("VoucherCount = %d, want 13", conversion.VoucherCount)
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
		if row[6] != row[30] || row[15] != row[30] {
			t.Errorf("single-line amounts differ: header=%q payment=%q distribution=%q", row[6], row[15], row[30])
		}
		if row[1] != "N" || row[2] != "" {
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
	if len(merchantCounts) != len(expectedMerchants) {
		t.Errorf("found %d merchant descriptions, want %d: %v", len(merchantCounts), len(expectedMerchants), merchantCounts)
	}
	for merchant, expected := range expectedMerchants {
		if merchantCounts[merchant] != expected {
			t.Errorf("merchant %q appears %d times, want %d", merchant, merchantCounts[merchant], expected)
		}
	}
}
