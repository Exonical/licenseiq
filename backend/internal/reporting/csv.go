package reporting

import (
	"bytes"
	"encoding/csv"
)

type CSVExporter struct{}

func (CSVExporter) ContentType() string { return "text/csv; charset=utf-8" }
func (CSVExporter) Extension() string   { return ".csv" }
func (CSVExporter) Export(report Table) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{report.Title})
	_ = w.Write(nil)
	_ = w.Write(report.Columns)
	for _, row := range report.Rows {
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	if len(report.Totals) > 0 {
		_ = w.Write(nil)
		_ = w.Write([]string{"Summary"})
		for _, total := range report.Totals {
			values := append([]string{total.Label}, total.Values...)
			if err := w.Write(values); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
