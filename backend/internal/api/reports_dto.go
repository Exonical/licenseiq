package api

import "time"

type ReportFormatInput struct {
	Format string    `query:"format" default:"json" example:"json"`
	AsOf   time.Time `query:"asOf,omitempty" example:"2026-01-01T00:00:00Z"`
}

type UpcomingRenewalsReportInput struct {
	ReportFormatInput
	WindowDays int `query:"windowDays" default:"90" example:"90"`
}

type ExpiredLicensesReportInput struct {
	ReportFormatInput
}

type VendorSpendReportInput struct {
	ReportFormatInput
}

type LicenseUtilizationReportInput struct {
	ReportFormatInput
}

type DepartmentSpendReportInput struct {
	ReportFormatInput
}

type ReportOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}
