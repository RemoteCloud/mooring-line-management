// Package report renders condition reports (CSV / PDF) from inspection data.
package report

import (
	"bytes"
	"encoding/csv"

	"github.com/go-pdf/fpdf"

	"github.com/ncl/mooring-api/internal/store"
)

// CSV renders the condition report rows as a CSV byte slice.
func CSV(rows []store.InspReportRow) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Line", "Serial", "Condition", "Last Inspected"})
	for _, r := range rows {
		_ = w.Write([]string{r.LineName, r.SerialNumber, r.ConditionStatus, r.LastInspected})
	}
	w.Flush()
	return buf.Bytes()
}

// PDF renders the condition report as a simple titled table.
func PDF(vesselName string, rows []store.InspReportRow) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "Condition Report — "+vesselName, "", 1, "L", false, 0, "")
	pdf.Ln(2)

	headers := []string{"Line", "Serial", "Condition", "Last Inspected"}
	widths := []float64{55, 50, 35, 40}

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetFillColor(235, 235, 235)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 8, h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 10)
	for _, r := range rows {
		cells := []string{r.LineName, r.SerialNumber, r.ConditionStatus, r.LastInspected}
		for i, c := range cells {
			pdf.CellFormat(widths[i], 7, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
