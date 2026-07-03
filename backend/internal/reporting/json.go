package reporting

import (
	"encoding/json"
)

type JSONExporter struct{}

func (JSONExporter) ContentType() string { return "application/json; charset=utf-8" }
func (JSONExporter) Extension() string   { return ".json" }
func (JSONExporter) Export(report Table) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
