package services

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	reimbursementSheet       = "Basic Invoice"
	reimbursementColumnCount = 35
)

type ReimbursementConversion struct {
	CSV          []byte
	VoucherCount int
	LineCount    int
}

type reimbursementLine struct {
	row         []string
	amountCents int64
}

type reimbursementGroup struct {
	lines      []reimbursementLine
	totalCents int64
}

// ConvertReimbursementWorkbook creates a Foundation CSV from the Basic Invoice sheet.
func ConvertReimbursementWorkbook(input io.Reader, invoiceDescription string) (ReimbursementConversion, error) {
	var result ReimbursementConversion

	invoiceDescription = strings.TrimSpace(invoiceDescription)
	if invoiceDescription == "" {
		invoiceDescription = "Employee Reimbursement"
	}
	if utf8Length(invoiceDescription) > 30 {
		return result, fmt.Errorf("invoice description must be 30 characters or fewer")
	}

	workbook, err := excelize.OpenReader(input)
	if err != nil {
		return result, fmt.Errorf("unable to open Excel workbook: %w", err)
	}
	defer workbook.Close()

	rows, err := workbook.GetRows(reimbursementSheet)
	if err != nil {
		return result, fmt.Errorf("unable to read %q sheet: %w", reimbursementSheet, err)
	}
	if len(rows) == 0 {
		return result, fmt.Errorf("%q sheet is empty", reimbursementSheet)
	}

	headerRowIndex := -1
	for rowIndex, row := range rows {
		if len(row) > 0 && normalizeReimbursementHeader(row[0]) == "field name indicates required" {
			headerRowIndex = rowIndex
			break
		}
	}
	if headerRowIndex < 0 {
		return result, fmt.Errorf("%q sheet does not contain the Foundation field header", reimbursementSheet)
	}
	header := padReimbursementRow(rows[headerRowIndex])
	columns := make(map[string]int, len(header))
	for index, name := range header {
		key := normalizeReimbursementHeader(name)
		if key != "" {
			columns[key] = index
		}
	}

	columnNames := map[string]string{
		"multiple":            "single or multiple distribution detail lines",
		"voucher_reference":   "voucher reference id",
		"vendor":              "vendor no",
		"invoice_date":        "invoice date",
		"transaction_date":    "transaction date",
		"header_amount":       "header amount",
		"invoice_description": "invoice description",
		"voucher_type":        "voucher type",
		"payment_type":        "payment type",
		"payment_amount":      "check amount / payment amount",
		"payment_date":        "check date / payment date",
		"gl_expense":          "g/l expense",
		"job":                 "job no",
		"cost_code":           "cost code no",
		"cost_class":          "cost class no",
		"distribution_amount": "distribution amount",
		"detail_description":  "description",
	}
	indices := make(map[string]int, len(columnNames))
	for key, name := range columnNames {
		index, ok := columns[name]
		if !ok {
			return result, fmt.Errorf("Foundation template is missing column %q", name)
		}
		indices[key] = index
	}
	if len(header) < reimbursementColumnCount {
		return result, fmt.Errorf("Foundation template has %d columns; expected at least %d", len(header), reimbursementColumnCount)
	}

	detailStart := -1
	for rowIndex := 0; rowIndex < headerRowIndex; rowIndex++ {
		for columnIndex, value := range rows[rowIndex] {
			if normalizeReimbursementHeader(value) == "detail" {
				detailStart = columnIndex
				break
			}
		}
		if detailStart >= 0 {
			break
		}
	}
	if detailStart <= 0 || detailStart >= reimbursementColumnCount {
		return result, fmt.Errorf("Foundation template is missing its DETAIL section marker")
	}

	dataStart := headerRowIndex + 5
	if dataStart > len(rows) {
		return result, fmt.Errorf("Foundation template has no reimbursement rows")
	}

	ignoredHeaderFields := map[int]bool{
		indices["multiple"]:            true,
		indices["voucher_reference"]:   true,
		indices["header_amount"]:       true,
		indices["invoice_description"]: true,
		indices["payment_amount"]:      true,
	}
	groups := make([]reimbursementGroup, 0)
	groupByKey := make(map[string]int)
	for rowIndex := dataStart; rowIndex < len(rows); rowIndex++ {
		row := padReimbursementRow(rows[rowIndex])
		if isEmptyReimbursementRow(row) {
			continue
		}

		lineNumber := rowIndex + 1
		merchant := strings.TrimSpace(row[indices["invoice_description"]])
		if merchant == "" {
			return result, fmt.Errorf("row %d: missing merchant description", lineNumber)
		}
		if utf8Length(merchant) > 30 {
			return result, fmt.Errorf("row %d: merchant description exceeds 30 characters", lineNumber)
		}
		amountCents, err := parseReimbursementCents(row[indices["header_amount"]])
		if err != nil {
			return result, fmt.Errorf("row %d: invalid header amount: %w", lineNumber, err)
		}

		for _, key := range []string{"invoice_date", "transaction_date", "payment_date"} {
			index := indices[key]
			dateValue := strings.TrimSpace(row[index])
			if dateValue == "" && key != "payment_date" {
				return result, fmt.Errorf("row %d: missing %s", lineNumber, columnNamesForError(key))
			}
			if dateValue != "" {
				formatted, dateErr := formatReimbursementDate(dateValue)
				if dateErr != nil {
					return result, fmt.Errorf("row %d: invalid %s: %w", lineNumber, columnNamesForError(key), dateErr)
				}
				row[index] = formatted
			}
		}

		row[indices["vendor"]] = "138"
		row[indices["voucher_type"]] = "P"
		row[indices["payment_type"]] = "EFT"

		var groupKey strings.Builder
		for index := 0; index < detailStart; index++ {
			if ignoredHeaderFields[index] {
				continue
			}
			value := strings.ReplaceAll(strings.ReplaceAll(row[index], "\r\n", "\n"), "\r", "\n")
			fmt.Fprintf(&groupKey, "%d:%s;", len(value), value)
		}
		key := groupKey.String()
		groupIndex, exists := groupByKey[key]
		if !exists {
			groupIndex = len(groups)
			groupByKey[key] = groupIndex
			groups = append(groups, reimbursementGroup{})
		}
		groups[groupIndex].lines = append(groups[groupIndex].lines, reimbursementLine{
			row:         row,
			amountCents: amountCents,
		})
		groups[groupIndex].totalCents += amountCents
		result.LineCount++
	}
	if len(groups) == 0 {
		return result, fmt.Errorf("no reimbursement line items found")
	}

	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	for rowIndex := 0; rowIndex < dataStart; rowIndex++ {
		templateRow := padReimbursementRow(rows[rowIndex])
		for index := range templateRow {
			templateRow[index] = strings.ReplaceAll(strings.ReplaceAll(templateRow[index], "\r\n", "\n"), "\r", "\n")
		}
		if rowIndex == headerRowIndex+2 {
			for _, key := range []string{"invoice_date", "transaction_date", "payment_date"} {
				index := indices[key]
				if formatted, dateErr := formatReimbursementDate(templateRow[index]); dateErr == nil {
					templateRow[index] = formatted
				}
			}
		}
		if err := writer.Write(templateRow); err != nil {
			return result, fmt.Errorf("failed to write template row: %w", err)
		}
	}

	for _, group := range groups {
		voucherReference := ""
		if len(group.lines) > 1 {
			voucherReference, err = newReimbursementVoucherReference()
			if err != nil {
				return result, fmt.Errorf("failed to generate voucher reference: %w", err)
			}
		}
		for _, line := range group.lines {
			row := append([]string(nil), line.row...)
			row[indices["multiple"]] = "N"
			if len(group.lines) > 1 {
				row[indices["multiple"]] = "Y"
			}
			row[indices["voucher_reference"]] = voucherReference
			row[indices["header_amount"]] = formatReimbursementCents(group.totalCents)
			row[indices["invoice_description"]] = invoiceDescription
			row[indices["payment_amount"]] = formatReimbursementCents(group.totalCents)
			row[indices["distribution_amount"]] = formatReimbursementCents(line.amountCents)
			row[indices["detail_description"]] = strings.TrimSpace(line.row[indices["invoice_description"]])
			if err := writer.Write(row); err != nil {
				return result, fmt.Errorf("failed to write reimbursement row: %w", err)
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return result, fmt.Errorf("failed to finish reimbursement CSV: %w", err)
	}

	result.CSV = output.Bytes()
	result.VoucherCount = len(groups)
	return result, nil
}

func normalizeReimbursementHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "*", "")
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}

func padReimbursementRow(row []string) []string {
	padded := make([]string, reimbursementColumnCount)
	copy(padded, row)
	return padded
}

func isEmptyReimbursementRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func parseReimbursementCents(value string) (int64, error) {
	cleaned := strings.NewReplacer("$", "", ",", "", " ", "").Replace(strings.TrimSpace(value))
	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, fmt.Errorf("%q is not a valid amount", value)
	}
	return int64(math.Round(amount * 100)), nil
}

func formatReimbursementCents(value int64) string {
	return fmt.Sprintf("%d.%02d", value/100, absReimbursementCents(value)%100)
}

func absReimbursementCents(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func formatReimbursementDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"1/2/06", "01/02/06", "1/2/2006", "01/02/2006", "1-2-06", "01-02-06", "1-2-2006", "01-02-2006", "2006-01-02", "1/2/06 15:04", "1/2/2006 15:04"} {
		parsed, err := time.ParseInLocation(layout, value, time.UTC)
		if err == nil {
			return fmt.Sprintf("%d/%d/%d", parsed.Month(), parsed.Day(), parsed.Year()), nil
		}
	}
	return "", fmt.Errorf("%q is not a supported date", value)
}

func columnNamesForError(key string) string {
	switch key {
	case "invoice_date":
		return "invoice date"
	case "transaction_date":
		return "transaction date"
	default:
		return "payment date"
	}
}

func utf8Length(value string) int {
	return len([]rune(value))
}

func newReimbursementVoucherReference() (string, error) {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return "RB" + time.Now().UTC().Format("0601021504") + hex.EncodeToString(randomBytes), nil
}
