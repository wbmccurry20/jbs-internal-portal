package services

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

//go:embed reimbursement_template.csv
var reimbursementTemplate string

func ConvertReimbursementCSV(input io.Reader, invoiceDescription string) (ReimbursementConversion, error) {
	var result ReimbursementConversion
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("unable to read accounting CSV header: %w", err)
	}
	columns := make(map[string]int)
	for index, name := range header {
		name = strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))
		if _, exists := columns[name]; exists {
			return result, fmt.Errorf("duplicate accounting CSV column %q", name)
		}
		columns[name] = index
	}
	for _, name := range []string{
		"Employee ID", "Report ID", "Employee Default Currency Alpha Code",
		"Report Processing Payment Date", "Report Entry Transaction Date", "Estimated Payment Date",
		"Report Entry Is Personal Flag", "Report Entry Payment Code Code", "Journal Payee Payment Type Name",
		"Journal Amount", "Journal Account Code", "Journal Debit Or Credit",
		"Report Entry Vendor Name", "Report Entry Vendor Description", "Report Entry Description",
		"Report Entry Expense Type Name", "Job", "Cost Code", "Cost Class",
	} {
		if _, exists := columns[name]; !exists {
			return result, fmt.Errorf("accounting CSV is missing column %q", name)
		}
	}
	rows, err := csv.NewReader(strings.NewReader(reimbursementTemplate)).ReadAll()
	if err != nil {
		return result, fmt.Errorf("unable to read Foundation template: %w", err)
	}
	shortened := 0
	for lineNumber := 2; ; lineNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return result, fmt.Errorf("accounting CSV row %d: %w", lineNumber, readErr)
		}
		get := func(name string) string { return strings.TrimSpace(record[columns[name]]) }
		if get("Employee ID") == "" || get("Report ID") == "" || get("Journal Account Code") == "" {
			return result, fmt.Errorf("row %d: employee, report and journal account are required", lineNumber)
		}
		if get("Employee Default Currency Alpha Code") != "USD" || get("Report Entry Is Personal Flag") != "N" ||
			get("Report Entry Payment Code Code") != "CASH" || get("Journal Payee Payment Type Name") != "Employee" ||
			get("Journal Debit Or Credit") != "DR" {
			return result, fmt.Errorf("row %d: only non-personal USD CASH employee debit expenses are supported; no output generated", lineNumber)
		}
		amount, amountErr := parseAccountingReimbursementCents(get("Journal Amount"))
		if amountErr != nil || amount <= 0 {
			return result, fmt.Errorf("row %d: journal amount must be positive with at most two decimal places", lineNumber)
		}
		merchant := ""
		for _, name := range []string{"Report Entry Vendor Name", "Report Entry Vendor Description", "Report Entry Description", "Report Entry Expense Type Name"} {
			if merchant = get(name); merchant != "" {
				break
			}
		}
		if utf8Length(merchant) > 30 {
			merchant = string([]rune(merchant)[:30])
			shortened++
		}
		row := make([]string, reimbursementColumnCount)
		row[1], row[3], row[12], row[13] = "N", "138", "P", "EFT"
		row[4], row[5], row[16] = get("Report Processing Payment Date"), get("Report Entry Transaction Date"), get("Estimated Payment Date")
		for _, index := range []int{4, 5, 16} {
			if row[index] == "" {
				return result, fmt.Errorf("row %d: processing, transaction and estimated payment dates are required", lineNumber)
			}
		}
		row[6], row[15], row[7] = formatReimbursementCents(amount), formatReimbursementCents(amount), merchant
		row[21], row[26], row[28], row[29] = get("Journal Account Code"), get("Job"), get("Cost Code"), get("Cost Class")
		rows = append(rows, row)
	}
	result, err = convertReimbursementRows(rows, invoiceDescription)
	result.ShortenedDescriptions = shortened
	return result, err
}

func parseAccountingReimbursementCents(value string) (int64, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid journal amount")
	}
	for _, part := range parts {
		if part == "" {
			return 0, fmt.Errorf("invalid journal amount")
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return 0, fmt.Errorf("invalid journal amount")
			}
		}
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, fmt.Errorf("fractional cents are not supported")
		}
		fraction = (parts[1] + "00")[:2]
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > 9999999999 {
		return 0, fmt.Errorf("amount exceeds Foundation numeric limit")
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return whole*100 + cents, nil
}
