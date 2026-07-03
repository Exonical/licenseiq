package reporting

import (
	"fmt"
	"strings"
)

type Table struct {
	Title   string       `json:"title"`
	Columns []string     `json:"columns"`
	Rows    [][]string   `json:"rows"`
	Totals  []SummaryRow `json:"totals,omitempty"`
}

type SummaryRow struct {
	Label  string   `json:"label"`
	Values []string `json:"values"`
}

type Exporter interface {
	ContentType() string
	Extension() string
	Export(Table) ([]byte, error)
}

func Filename(title, ext string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	if base == "" {
		base = "report"
	}
	base = strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(base)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "report"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s%s", base, ext)
}
