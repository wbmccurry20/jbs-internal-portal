package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

const reimbursementJournalColumnCount = 13

func ConvertConcurPayScreenCSV(input io.Reader) (ReimbursementConversion, error) {
	var result ReimbursementConversion
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1

	var header []string
	for range 3 {
		row, err := reader.Read()
		if err != nil {
			return result, fmt.Errorf("unable to find Concur pay screen CSV headers: %w", err)
		}
		if hasReimbursementCSVColumn(row, "DATE") &&
			hasReimbursementCSVColumn(row, "Journal Account Code") {
			header = row
			break
		}
	}
	if header == nil {
		return result, fmt.Errorf("Concur pay screen CSV is missing its column header row")
	}

	columns := make(map[string]int, len(header))
	for index, name := range header {
		name = strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))
		if _, exists := columns[name]; exists {
			return result, fmt.Errorf("duplicate Concur pay screen CSV column %q", name)
		}
		columns[name] = index
	}
	for _, name := range []string{
		"Employee ID", "Employee Default Currency Alpha Code", "Report Processing Payment Date",
		"Report Entry Is Personal Flag", "Report Entry Description", "Report Entry Vendor Description",
		"Report Entry Payment Code Code", "Journal Payee Payment Type Name", "Journal Amount",
		"Journal Account Code", "Journal Debit Or Credit", "Cost Class", "Cost Code", "Job",
	} {
		if _, exists := columns[name]; !exists {
			return result, fmt.Errorf("Concur pay screen CSV is missing column %q", name)
		}
	}

	get := func(record []string, name string, lineNumber int) (string, error) {
		index := columns[name]
		if index >= len(record) {
			return "", fmt.Errorf("row %d: missing value for %q", lineNumber, name)
		}
		return strings.TrimSpace(record[index]), nil
	}

	output := make([][]string, 0)
	var totalCents int64
	var processingDate string
	for lineNumber := 3; ; lineNumber++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, fmt.Errorf("Concur pay screen CSV row %d: %w", lineNumber, err)
		}
		if isEmptyReimbursementRow(record) {
			break
		}
		if strings.EqualFold(strings.TrimSpace(record[0]), "Notes") {
			break
		}

		getValue := func(name string) (string, error) { return get(record, name, lineNumber) }
		employeeID, err := getValue("Employee ID")
		if err != nil {
			return result, err
		}
		currency, err := getValue("Employee Default Currency Alpha Code")
		if err != nil {
			return result, err
		}
		personal, err := getValue("Report Entry Is Personal Flag")
		if err != nil {
			return result, err
		}
		paymentCode, err := getValue("Report Entry Payment Code Code")
		if err != nil {
			return result, err
		}
		payeeType, err := getValue("Journal Payee Payment Type Name")
		if err != nil {
			return result, err
		}
		debitOrCredit, err := getValue("Journal Debit Or Credit")
		if err != nil {
			return result, err
		}
		if employeeID == "" {
			return result, fmt.Errorf("row %d: employee ID is required", lineNumber)
		}
		if currency != "USD" || personal != "N" || paymentCode != "CASH" ||
			payeeType != "Employee" || debitOrCredit != "DR" {
			return result, fmt.Errorf("row %d: only non-personal USD CASH employee debit expenses are supported; no output generated", lineNumber)
		}

		amountText, err := getValue("Journal Amount")
		if err != nil {
			return result, err
		}
		amount, err := parseAccountingReimbursementCents(amountText)
		if err != nil || amount <= 0 {
			return result, fmt.Errorf("row %d: journal amount must be positive with at most two decimal places", lineNumber)
		}
		if amount > 0 && totalCents > int64(^uint64(0)>>1)-amount {
			return result, fmt.Errorf("row %d: total journal amount exceeds the supported range", lineNumber)
		}

		description, err := getValue("Report Entry Description")
		if err != nil {
			return result, err
		}
		if description == "" {
			description, err = getValue("Report Entry Vendor Description")
			if err != nil {
				return result, err
			}
		}
		if description == "" {
			return result, fmt.Errorf("row %d: entry description and vendor description are both blank", lineNumber)
		}
		description = strings.Join(strings.Fields(strings.ReplaceAll(description, ",", " ")), " ")

		dateText, err := getValue("Report Processing Payment Date")
		if err != nil {
			return result, err
		}
		formattedDate, err := formatReimbursementDate(dateText)
		if err != nil {
			return result, fmt.Errorf("row %d: invalid processing payment date: %w", lineNumber, err)
		}
		if processingDate == "" {
			processingDate = formattedDate
		} else if processingDate != formattedDate {
			return result, fmt.Errorf("row %d: all expenses must have the same processing payment date", lineNumber)
		}

		account, err := getValue("Journal Account Code")
		if err != nil {
			return result, err
		}
		job, err := getValue("Job")
		if err != nil {
			return result, err
		}
		costCode, err := getValue("Cost Code")
		if err != nil {
			return result, err
		}
		costClass, err := getValue("Cost Class")
		if err != nil {
			return result, err
		}
		row := make([]string, reimbursementJournalColumnCount)
		row[0] = account
		row[5] = "D"
		row[6] = formatReimbursementCents(amount)
		row[7] = description
		row[9] = job
		row[11] = costCode
		row[12] = costClass
		output = append(output, row)
		totalCents += amount
		result.LineCount++
	}
	if len(output) == 0 {
		return result, fmt.Errorf("no reimbursement line items found")
	}

	date, err := time.ParseInLocation("1/2/2006", processingDate, time.UTC)
	if err != nil {
		return result, fmt.Errorf("unable to format processing payment date: %w", err)
	}
	creditRow := make([]string, reimbursementJournalColumnCount)
	creditRow[0] = "1002"
	creditRow[5] = "C"
	creditRow[6] = formatReimbursementCents(totalCents)
	creditRow[7] = date.Format("01.02.06") + " reimbursement"
	output = append(output, creditRow)

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.WriteAll(output); err != nil {
		return result, fmt.Errorf("failed to write reimbursement journal CSV: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return result, fmt.Errorf("failed to finish reimbursement journal CSV: %w", err)
	}

	result.CSV = buffer.Bytes()
	result.Filename = "reimbursement_journal.csv"
	result.VoucherCount = result.LineCount
	return result, nil
}
