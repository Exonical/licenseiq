package reporting

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

type PDFExporter struct{}

func (PDFExporter) ContentType() string { return "application/pdf" }
func (PDFExporter) Extension() string   { return ".pdf" }
func (PDFExporter) Export(report Table) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(report.Title, false)
	pdf.SetAuthor("LicenseIQ", false)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, report.Title, "", 1, "L", false, 0, "")
	pdf.Ln(2)
	if len(report.Columns) > 0 {
		pdf.SetFont("Helvetica", "B", 9)
		widths := columnWidths(pdf, report.Columns)
		for i, column := range report.Columns {
			pdf.CellFormat(widths[i], 7, column, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Helvetica", "", 9)
		for _, row := range report.Rows {
			for i, value := range row {
				if i >= len(widths) {
					break
				}
				pdf.CellFormat(widths[i], 7, value, "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
		}
		if len(report.Totals) > 0 {
			pdf.Ln(2)
			pdf.SetFont("Helvetica", "B", 10)
			pdf.CellFormat(0, 7, "Summary", "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 9)
			for _, total := range report.Totals {
				parts := append([]string{total.Label}, total.Values...)
				pdf.CellFormat(0, 6, strings.Join(parts, " | "), "", 1, "L", false, 0, "")
			}
		}
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func columnWidths(pdf *fpdf.Fpdf, columns []string) []float64 {
	pageWidth, _ := pdf.GetPageSize()
	marginLeft, _, marginRight, _ := pdf.GetMargins()
	usable := pageWidth - marginLeft - marginRight
	if len(columns) == 0 {
		return nil
	}
	width := usable / float64(len(columns))
	widths := make([]float64, len(columns))
	for i := range widths {
		widths[i] = width
	}
	return widths
}
