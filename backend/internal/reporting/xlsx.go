package reporting

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

type XLSXExporter struct{}

func (XLSXExporter) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}
func (XLSXExporter) Extension() string { return ".xlsx" }
func (XLSXExporter) Export(report Table) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	name := "Report"
	if report.Title != "" {
		name = sanitizeSheetName(report.Title)
	}
	if index, err := f.GetSheetIndex("Sheet1"); err != nil {
		return nil, err
	} else if index != -1 && name != "Sheet1" {
		if err := f.SetSheetName("Sheet1", name); err != nil {
			return nil, err
		}
	}
	if err := f.SetCellValue(name, "A1", report.Title); err != nil {
		return nil, err
	}
	for i, column := range report.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		if err := f.SetCellValue(name, cell, column); err != nil {
			return nil, err
		}
	}
	for r, row := range report.Rows {
		for c, value := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+4)
			if err := f.SetCellValue(name, cell, value); err != nil {
				return nil, err
			}
		}
	}
	if len(report.Totals) > 0 {
		start := len(report.Rows) + 6
		if err := f.SetCellValue(name, fmt.Sprintf("A%d", start), "Summary"); err != nil {
			return nil, err
		}
		for i, total := range report.Totals {
			row := start + i + 1
			if err := f.SetCellValue(name, fmt.Sprintf("A%d", row), total.Label); err != nil {
				return nil, err
			}
			for c, value := range total.Values {
				cell, _ := excelize.CoordinatesToCellName(c+2, row)
				if err := f.SetCellValue(name, cell, value); err != nil {
					return nil, err
				}
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sanitizeSheetName(value string) string {
	if value == "" {
		return "Report"
	}
	out := strings.NewReplacer("\\", "-", "/", "-", "?", "-", "*", "-", "[", "-", "]", "-", ":", "-").Replace(value)
	if len(out) > 31 {
		out = out[:31]
	}
	if out == "" {
		return "Report"
	}
	return out
}
