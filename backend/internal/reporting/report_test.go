package reporting

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/xuri/excelize/v2"
)

func sampleReport() Table {
	return Table{
		Title:   "License Utilization",
		Columns: []string{"License", "Seat Count"},
		Rows:    [][]string{{"AAA-BBB", "10"}},
		Totals:  []SummaryRow{{Label: "Total", Values: []string{"10"}}},
	}
}

func TestJSONExporter(t *testing.T) {
	data, err := JSONExporter{}.Export(sampleReport())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !bytes.Contains(data, []byte(`"title": "License Utilization"`)) {
		t.Fatalf("expected json payload")
	}
}

func TestCSVExporter(t *testing.T) {
	data, err := CSVExporter{}.Export(sampleReport())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if rows[0][0] != "License Utilization" {
		t.Fatalf("unexpected title row: %#v", rows[0])
	}
	if rows[0][0] != "License Utilization" || rows[1][0] != "License" || rows[2][0] != "AAA-BBB" {
		t.Fatalf("unexpected csv body: %#v", rows)
	}
}

func TestXLSXExporter(t *testing.T) {
	data, err := XLSXExporter{}.Export(sampleReport())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	if sheet != "License Utilization" {
		t.Fatalf("unexpected sheet: %s", sheet)
	}
	value, err := f.GetCellValue(sheet, "A1")
	if err != nil {
		t.Fatalf("get cell: %v", err)
	}
	if value != "License Utilization" {
		t.Fatalf("unexpected xlsx title: %s", value)
	}
}

func TestPDFExporter(t *testing.T) {
	data, err := PDFExporter{}.Export(sampleReport())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("expected pdf header")
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty pdf bytes")
	}
}
