// Package pdf generates JBS-branded Application for Payment PDFs in memory.
// No files are written to disk; the generated PDF is returned as []byte.
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/wbmccurry20/jbs-internal-portal/internal/database"
)

// ─── colour palette ────────────────────────────────────────────────────────────

var (
	colDark   = [3]int{30, 41, 59}   // jbs-dark  #1e2937
	colBlue   = [3]int{0, 160, 224}  // jbs-blue  #00a0e0
	colGray   = [3]int{107, 114, 128} // gray-500
	colWhite  = [3]int{255, 255, 255}
	colLight  = [3]int{243, 244, 246} // gray-100 — table header fill
	colAlt    = [3]int{249, 250, 251} // gray-50  — alternating row fill
)

// ─── helpers ───────────────────────────────────────────────────────────────────

func fmtUSD(v float64) string {
	if v < 0 {
		return fmt.Sprintf("-$%.2f", -v)
	}
	return fmt.Sprintf("$%.2f", v)
}

func fmtDate(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("01/02/2006")
}

func setFill(pdf *fpdf.Fpdf, rgb [3]int) {
	pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
}

func setTextColor(pdf *fpdf.Fpdf, rgb [3]int) {
	pdf.SetTextColor(rgb[0], rgb[1], rgb[2])
}

func setDrawColor(pdf *fpdf.Fpdf, rgb [3]int) {
	pdf.SetDrawColor(rgb[0], rgb[1], rgb[2])
}

// sectionHeader draws a coloured section heading row spanning the full page width.
func sectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.Ln(3)
	setFill(pdf, colDark)
	setTextColor(pdf, colWhite)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(0, 7, "  "+title, "", 1, "L", true, 0, "")
	setTextColor(pdf, colDark)
	setFill(pdf, colWhite)
}

// kv renders a two-column label:value pair.
func kv(pdf *fpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "B", 8)
	setTextColor(pdf, colGray)
	pdf.CellFormat(45, 5, label+":", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	setTextColor(pdf, colDark)
	pdf.MultiCell(0, 5, value, "", "L", false)
}

// ─── GeneratePDF ───────────────────────────────────────────────────────────────

// GeneratePDF generates a JBS-branded Application for Payment PDF from data and
// returns it as a byte slice. No files are written to disk.
func GeneratePDF(data *database.PaymentApplicationPDFData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	contentW := pageW - lm - rm

	// ── Header banner ──────────────────────────────────────────────────────────
	setFill(pdf, colDark)
	setDrawColor(pdf, colDark)
	pdf.Rect(lm, 12, contentW, 22, "F")

	pdf.SetFont("Arial", "B", 16)
	setTextColor(pdf, colWhite)
	pdf.SetXY(lm+4, 14)
	pdf.CellFormat(contentW/2, 8, data.TenantName, "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	setTextColor(pdf, colBlue)
	pdf.SetXY(lm+4, 22)
	pdf.CellFormat(contentW/2, 6, "APPLICATION FOR PAYMENT", "", 0, "L", false, 0, "")

	// right column: application number + period
	pdf.SetFont("Arial", "B", 8)
	setTextColor(pdf, colWhite)
	pdf.SetXY(pageW-rm-80, 14)
	pdf.CellFormat(76, 5, fmt.Sprintf("Application No: %d", data.ApplicationNumber), "", 1, "R", false, 0, "")
	pdf.SetXY(pageW-rm-80, 19)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(76, 5, fmt.Sprintf("Period To: %s", fmtDate(data.PeriodTo)), "", 1, "R", false, 0, "")
	pdf.SetXY(pageW-rm-80, 24)
	pdf.CellFormat(76, 5, fmt.Sprintf("Date: %s", data.CreatedAt.Format("01/02/2006")), "", 1, "R", false, 0, "")

	pdf.SetXY(lm, 36)

	// ── Subcontractor info ─────────────────────────────────────────────────────
	sectionHeader(pdf, "Subcontractor Information")
	pdf.Ln(1)

	halfW := contentW / 2
	leftX := lm

	// Two-column layout: left = contact, right = address
	startY := pdf.GetY()
	pdf.SetXY(leftX, startY)
	kv(pdf, "Company", data.CompanyName)
	kv(pdf, "Contact", data.ContactName)
	kv(pdf, "Email", data.Email)
	kv(pdf, "Phone", data.Phone)

	// right column
	pdf.SetXY(leftX+halfW+4, startY)
	pdf.SetFont("Arial", "B", 8)
	setTextColor(pdf, colGray)
	pdf.CellFormat(45, 5, "Address:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	setTextColor(pdf, colDark)
	addrParts := []string{}
	if data.AddressLine1 != "" {
		addrParts = append(addrParts, data.AddressLine1)
	}
	if data.AddressLine2 != "" {
		addrParts = append(addrParts, data.AddressLine2)
	}
	cityStateZip := ""
	if data.City != "" || data.State != "" || data.Zip != "" {
		cityStateZip = data.City
		if data.State != "" {
			if cityStateZip != "" {
				cityStateZip += ", "
			}
			cityStateZip += data.State
		}
		if data.Zip != "" {
			cityStateZip += " " + data.Zip
		}
		addrParts = append(addrParts, cityStateZip)
	}
	addr := "—"
	for i, p := range addrParts {
		if i == 0 {
			addr = p
		} else {
			addr += "\n" + p
		}
	}
	pdf.SetXY(leftX+halfW+49, startY)
	pdf.MultiCell(halfW-53, 5, addr, "", "L", false)

	pdf.Ln(2)

	// ── Project info ───────────────────────────────────────────────────────────
	sectionHeader(pdf, "Project Information")
	pdf.Ln(1)

	startY = pdf.GetY()
	pdf.SetXY(leftX, startY)
	kv(pdf, "Project Name", data.ProjectName)
	kv(pdf, "Project Number", data.ProjectNumber)
	kv(pdf, "Owner", data.Owner)
	kv(pdf, "Contractor", data.Contractor)

	pdf.SetXY(leftX+halfW+4, startY)
	kv(pdf, "Contract Date", fmtDate(data.ContractDate))

	pdf.Ln(2)

	// ── Contract summary ───────────────────────────────────────────────────────
	sectionHeader(pdf, "Contract Summary")

	type summaryRow struct {
		label string
		value string
		bold  bool
		blue  bool
	}

	rows := []summaryRow{
		{"Original Contract Sum", fmtUSD(data.OriginalContractSum), false, false},
		{"Net Change Orders", fmtUSD(data.CalcNetChangeOrders), false, false},
		{"Contract Sum to Date", fmtUSD(data.CalcContractSumToDate), true, false},
		{"Total Completed & Stored", fmtUSD(data.CalcTotalCompleted), false, false},
		{fmt.Sprintf("Retainage (%.1f%%)", data.RetainagePercent), fmtUSD(data.CalcRetainageAmount), false, false},
		{"Total Earned Less Retainage", fmtUSD(data.CalcEarnedLessRetainage), false, false},
		{"Previous Certificates", fmtUSD(data.PreviousCertificates), false, false},
		{"Current Payment Due", fmtUSD(data.CalcCurrentPaymentDue), true, true},
		{"Balance to Finish", fmtUSD(data.CalcBalanceToFinish), false, false},
	}

	labelW := 90.0
	valueW := contentW - labelW

	for i, r := range rows {
		if i%2 == 1 {
			setFill(pdf, colAlt)
		} else {
			setFill(pdf, colWhite)
		}
		if r.bold {
			setFill(pdf, colLight)
		}

		style := ""
		if r.bold {
			style = "B"
		}
		pdf.SetFont("Arial", style, 8)

		if r.blue {
			setTextColor(pdf, colBlue)
		} else {
			setTextColor(pdf, colDark)
		}
		setDrawColor(pdf, colLight)
		pdf.CellFormat(labelW, 6, "  "+r.label, "B", 0, "L", true, 0, "")
		pdf.CellFormat(valueW, 6, r.value+"  ", "B", 1, "R", true, 0, "")
	}
	setTextColor(pdf, colDark)
	setDrawColor(pdf, colDark)

	pdf.Ln(2)

	// ── Change orders ──────────────────────────────────────────────────────────
	if len(data.ChangeOrders) > 0 {
		sectionHeader(pdf, "Change Orders")

		// Table header
		coHeaders := []struct {
			label string
			w     float64
		}{
			{"CO #", 18},
			{"Description", contentW - 18 - 30 - 28},
			{"Amount", 30},
			{"Date Approved", 28},
		}

		setFill(pdf, colLight)
		pdf.SetFont("Arial", "B", 7.5)
		setTextColor(pdf, colDark)
		for _, h := range coHeaders {
			pdf.CellFormat(h.w, 6, "  "+h.label, "B", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		for i, co := range data.ChangeOrders {
			if i%2 == 0 {
				setFill(pdf, colWhite)
			} else {
				setFill(pdf, colAlt)
			}
			pdf.SetFont("Arial", "", 7.5)
			setDrawColor(pdf, colLight)
			pdf.CellFormat(coHeaders[0].w, 5.5, "  "+co.CONumber, "B", 0, "L", true, 0, "")
			pdf.CellFormat(coHeaders[1].w, 5.5, "  "+co.Description, "B", 0, "L", true, 0, "")
			pdf.CellFormat(coHeaders[2].w, 5.5, fmtUSD(co.Amount)+"  ", "B", 0, "R", true, 0, "")
			pdf.CellFormat(coHeaders[3].w, 5.5, "  "+fmtDate(co.DateApproved), "B", 1, "L", true, 0, "")
		}
		setDrawColor(pdf, colDark)
		pdf.Ln(2)
	}

	// ── Schedule of values ─────────────────────────────────────────────────────
	if len(data.LineItems) > 0 {
		sectionHeader(pdf, "Schedule of Values")

		descW := contentW - 14 - 22 - 22 - 22 - 22 - 20 - 20
		if descW < 20 {
			descW = 20
		}
		liHeaders := []struct {
			label string
			w     float64
		}{
			{"#", 14},
			{"Description", descW},
			{"Sched. Value", 22},
			{"Prev %", 22},
			{"This Period", 22},
			{"Mat. Stored", 22},
			{"Total %", 20},
			{"Bal. to Finish", 20},
		}

		setFill(pdf, colLight)
		pdf.SetFont("Arial", "B", 6.5)
		setTextColor(pdf, colDark)
		setDrawColor(pdf, colLight)
		for _, h := range liHeaders {
			pdf.CellFormat(h.w, 6, " "+h.label, "B", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		for i, li := range data.LineItems {
			if i%2 == 0 {
				setFill(pdf, colWhite)
			} else {
				setFill(pdf, colAlt)
			}
			pdf.SetFont("Arial", "", 6.5)
			pdf.CellFormat(liHeaders[0].w, 5, " "+li.ItemNo, "B", 0, "C", true, 0, "")
			pdf.CellFormat(liHeaders[1].w, 5, " "+li.Description, "B", 0, "L", true, 0, "")
			pdf.CellFormat(liHeaders[2].w, 5, fmtUSD(li.ScheduledValue), "B", 0, "R", true, 0, "")
			pdf.CellFormat(liHeaders[3].w, 5, fmtUSD(li.PrevCompleted), "B", 0, "R", true, 0, "")
			pdf.CellFormat(liHeaders[4].w, 5, fmtUSD(li.ThisPeriod), "B", 0, "R", true, 0, "")
			pdf.CellFormat(liHeaders[5].w, 5, fmtUSD(li.MaterialsStored), "B", 0, "R", true, 0, "")
			pdf.CellFormat(liHeaders[6].w, 5, fmt.Sprintf("%.1f%%", li.CalcPercentComplete), "B", 0, "R", true, 0, "")
			pdf.CellFormat(liHeaders[7].w, 5, fmtUSD(li.CalcBalanceToFinish), "B", 1, "R", true, 0, "")
		}
		setDrawColor(pdf, colDark)
		pdf.Ln(2)
	}

	// ── Notes ──────────────────────────────────────────────────────────────────
	if data.AdditionalNotes != "" {
		sectionHeader(pdf, "Additional Notes")
		pdf.SetFont("Arial", "", 8)
		setTextColor(pdf, colDark)
		pdf.Ln(1)
		pdf.MultiCell(0, 5, data.AdditionalNotes, "", "L", false)
		pdf.Ln(2)
	}

	// ── Footer ─────────────────────────────────────────────────────────────────
	_, pageH := pdf.GetPageSize()
	_, _, _, bm := pdf.GetMargins()
	pdf.SetXY(lm, pageH-bm-6)
	setDrawColor(pdf, colBlue)
	pdf.Line(lm, pageH-bm-7, pageW-rm, pageH-bm-7)
	pdf.SetFont("Arial", "I", 7)
	setTextColor(pdf, colGray)
	pdf.CellFormat(contentW/2, 5,
		fmt.Sprintf("Generated by JBS Client Portal — %s", time.Now().UTC().Format("January 2, 2006 15:04 UTC")),
		"", 0, "L", false, 0, "")
	pdf.CellFormat(contentW/2, 5,
		fmt.Sprintf("Submission token: %s", data.SubmissionToken),
		"", 0, "R", false, 0, "")

	// ── Output ─────────────────────────────────────────────────────────────────
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}
