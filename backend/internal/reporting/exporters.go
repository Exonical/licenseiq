package reporting

import (
	"fmt"
	"strings"
)

func NewExporter(format string) (Exporter, error) {
	switch normalizeFormat(format) {
	case "", "json":
		return JSONExporter{}, nil
	case "csv":
		return CSVExporter{}, nil
	case "xlsx":
		return XLSXExporter{}, nil
	case "pdf":
		return PDFExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func normalizeFormat(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
