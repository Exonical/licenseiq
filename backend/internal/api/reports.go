package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/reporting"
	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"
)

func registerReportingRoutes(api huma.API, svc app.ReportingService, logger *zap.Logger) {
	if svc == nil {
		return
	}

	huma.Get(api, "/reports/renewals", func(ctx context.Context, input *struct{ UpcomingRenewalsReportInput }) (*ReportOutput, error) {
		windowDays := input.WindowDays
		if windowDays <= 0 {
			windowDays = 90
		}
		table, err := svc.UpcomingRenewals(ctx, app.UpcomingRenewalsParams{AsOf: input.AsOf, WindowDays: windowDays})
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return renderReport(table, input.Format)
	}, func(o *huma.Operation) {
		o.OperationID = "getRenewalReport"
		o.Summary = "Get upcoming renewals report"
		o.Description = "Generate an upcoming renewals report with optional export format."
		o.Tags = []string{"Reports"}
		o.Errors = operationErrors()
		protectedOperation("reports", "read")(o)
	})

	huma.Get(api, "/reports/expired", func(ctx context.Context, input *struct{ ExpiredLicensesReportInput }) (*ReportOutput, error) {
		table, err := svc.ExpiredLicenses(ctx, app.ExpiredLicensesParams{AsOf: input.AsOf})
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return renderReport(table, input.Format)
	}, func(o *huma.Operation) {
		o.OperationID = "getExpiredLicensesReport"
		o.Summary = "Get expired licenses report"
		o.Description = "Generate an expired licenses report with optional export format."
		o.Tags = []string{"Reports"}
		o.Errors = operationErrors()
		protectedOperation("reports", "read")(o)
	})

	huma.Get(api, "/reports/vendor-spend", func(ctx context.Context, input *struct{ VendorSpendReportInput }) (*ReportOutput, error) {
		table, err := svc.VendorSpend(ctx, app.ReportingAsOfParams{AsOf: input.AsOf})
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return renderReport(table, input.Format)
	}, func(o *huma.Operation) {
		o.OperationID = "getVendorSpendReport"
		o.Summary = "Get vendor spend report"
		o.Description = "Generate a vendor spend report with optional export format."
		o.Tags = []string{"Reports"}
		o.Errors = operationErrors()
		protectedOperation("reports", "financial")(o)
	})

	huma.Get(api, "/reports/utilization", func(ctx context.Context, input *struct{ LicenseUtilizationReportInput }) (*ReportOutput, error) {
		table, err := svc.LicenseUtilization(ctx, app.ReportingAsOfParams{AsOf: input.AsOf})
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return renderReport(table, input.Format)
	}, func(o *huma.Operation) {
		o.OperationID = "getLicenseUtilizationReport"
		o.Summary = "Get license utilization report"
		o.Description = "Generate a license utilization report with optional export format."
		o.Tags = []string{"Reports"}
		o.Errors = operationErrors()
		protectedOperation("reports", "read")(o)
	})

	huma.Get(api, "/reports/department-spend", func(ctx context.Context, input *struct{ DepartmentSpendReportInput }) (*ReportOutput, error) {
		table, err := svc.DepartmentSpend(ctx, app.ReportingAsOfParams{AsOf: input.AsOf})
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return renderReport(table, input.Format)
	}, func(o *huma.Operation) {
		o.OperationID = "getDepartmentSpendReport"
		o.Summary = "Get department spend report"
		o.Description = "Generate a department spend report with optional export format."
		o.Tags = []string{"Reports"}
		o.Errors = operationErrors()
		protectedOperation("reports", "financial")(o)
	})
}

func renderReport(table reporting.Table, format string) (*ReportOutput, error) {
	exporter, err := reporting.NewExporter(format)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}
	data, err := exporter.Export(table)
	if err != nil {
		return nil, err
	}
	output := &ReportOutput{ContentType: exporter.ContentType(), Body: data}
	if normalizeReportFormat(format) != "json" {
		output.ContentDisposition = fmt.Sprintf("attachment; filename=%q", reporting.Filename(table.Title, exporter.Extension()))
	}
	return output, nil
}

func normalizeReportFormat(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
