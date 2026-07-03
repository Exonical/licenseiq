package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/reporting"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ReportingService interface {
	UpcomingRenewals(context.Context, UpcomingRenewalsParams) (reporting.Table, error)
	ExpiredLicenses(context.Context, ExpiredLicensesParams) (reporting.Table, error)
	VendorSpend(context.Context, ReportingAsOfParams) (reporting.Table, error)
	LicenseUtilization(context.Context, ReportingAsOfParams) (reporting.Table, error)
	DepartmentSpend(context.Context, ReportingAsOfParams) (reporting.Table, error)
}

type ReportingAsOfParams struct {
	AsOf time.Time
}

type UpcomingRenewalsParams struct {
	AsOf       time.Time
	WindowDays int
}

type ExpiredLicensesParams struct {
	AsOf time.Time
}

type reportingService struct {
	vendors  domain.VendorRepository
	products domain.ProductRepository
	licenses domain.LicenseRepository
}

func NewReportingService(vendors domain.VendorRepository, products domain.ProductRepository, licenses domain.LicenseRepository) ReportingService {
	return &reportingService{vendors: vendors, products: products, licenses: licenses}
}

func (s *reportingService) UpcomingRenewals(ctx context.Context, params UpcomingRenewalsParams) (reporting.Table, error) {
	asOf := normalizeAsOf(params.AsOf)
	windowDays := params.WindowDays
	if windowDays <= 0 {
		windowDays = 90
	}
	licenses, vendors, products, err := s.loadCatalog(ctx)
	if err != nil {
		return reporting.Table{}, err
	}
	upper := asOf.AddDate(0, 0, windowDays)
	rows := make([][]string, 0)
	for _, lic := range licenses {
		if lic.RenewalDate == nil {
			continue
		}
		renewal := lic.RenewalDate.UTC()
		if renewal.Before(asOf) || renewal.After(upper) {
			continue
		}
		rows = append(rows, []string{
			licenseLabel(lic),
			vendorName(vendors, lic.VendorID),
			productName(products, lic.ProductID),
			renewal.Format(time.RFC3339),
			fmt.Sprintf("%d", daysUntil(asOf, renewal)),
			lic.Cost.StringFixedBank(2),
			lic.Currency,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][3] < rows[j][3] })
	return reporting.Table{
		Title:   fmt.Sprintf("Upcoming Renewals (%d days)", windowDays),
		Columns: []string{"License", "Vendor", "Product", "Renewal Date", "Days Until", "Cost", "Currency"},
		Rows:    rows,
	}, nil
}

func (s *reportingService) ExpiredLicenses(ctx context.Context, params ExpiredLicensesParams) (reporting.Table, error) {
	asOf := normalizeAsOf(params.AsOf)
	licenses, vendors, products, err := s.loadCatalog(ctx)
	if err != nil {
		return reporting.Table{}, err
	}
	rows := make([][]string, 0)
	for _, lic := range licenses {
		expiredAt, expiredBy := expiredAt(lic, asOf)
		if expiredAt == nil {
			continue
		}
		rows = append(rows, []string{
			licenseLabel(lic),
			vendorName(vendors, lic.VendorID),
			productName(products, lic.ProductID),
			formatTime(lic.ExpirationDate),
			formatTime(lic.MaintenanceExpiration),
			fmt.Sprintf("%d", int(asOf.Sub(expiredAt.UTC()).Hours()/24)),
			expiredBy,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	return reporting.Table{
		Title:   "Expired Licenses",
		Columns: []string{"License", "Vendor", "Product", "Expiration Date", "Maintenance Expiration", "Days Past", "Expired By"},
		Rows:    rows,
	}, nil
}

func (s *reportingService) VendorSpend(ctx context.Context, params ReportingAsOfParams) (reporting.Table, error) {
	_, vendors, _, err := s.loadCatalog(ctx)
	if err != nil {
		return reporting.Table{}, err
	}
	licenses, err := s.listLicenses(ctx)
	if err != nil {
		return reporting.Table{}, err
	}
	totals := map[string]map[string]summaryBucket{}
	for _, lic := range licenses {
		vendor := vendorName(vendors, lic.VendorID)
		currency := currencyBucket(lic.Currency)
		if _, ok := totals[vendor]; !ok {
			totals[vendor] = map[string]summaryBucket{}
		}
		bucket := totals[vendor][currency]
		bucket.count++
		bucket.total = bucket.total.Add(lic.Cost)
		totals[vendor][currency] = bucket
	}
	rows := make([][]string, 0)
	vendorsSorted := sortedKeys(totals)
	for _, vendor := range vendorsSorted {
		currencies := sortedCurrencyKeys(totals[vendor])
		for _, currency := range currencies {
			bucket := totals[vendor][currency]
			rows = append(rows, []string{vendor, currency, bucket.total.StringFixedBank(2), fmt.Sprintf("%d", bucket.count)})
		}
	}
	return reporting.Table{
		Title:   "Vendor Spend",
		Columns: []string{"Vendor", "Currency", "Total Cost", "License Count"},
		Rows:    rows,
		Totals:  summaryRowsForCurrencyTotals(aggregateCurrencyTotals(licenses)),
	}, nil
}

func (s *reportingService) LicenseUtilization(ctx context.Context, params ReportingAsOfParams) (reporting.Table, error) {
	licenses, vendors, products, err := s.loadCatalog(ctx)
	if err != nil {
		return reporting.Table{}, err
	}
	rows := make([][]string, 0, len(licenses))
	var totalSeats, totalAssigned int
	for _, lic := range licenses {
		available := lic.AvailableSeats()
		if available < 0 {
			available = 0
		}
		utilization := 0.0
		if lic.SeatCount > 0 {
			utilization = (float64(lic.AssignedSeats) / float64(lic.SeatCount)) * 100
		}
		rows = append(rows, []string{
			licenseLabel(lic),
			vendorName(vendors, lic.VendorID),
			productName(products, lic.ProductID),
			fmt.Sprintf("%d", lic.SeatCount),
			fmt.Sprintf("%d", lic.AssignedSeats),
			fmt.Sprintf("%d", available),
			fmt.Sprintf("%.2f%%", utilization),
		})
		totalSeats += lic.SeatCount
		totalAssigned += lic.AssignedSeats
	}
	totalAvailable := totalSeats - totalAssigned
	if totalAvailable < 0 {
		totalAvailable = 0
	}
	overall := 0.0
	if totalSeats > 0 {
		overall = (float64(totalAssigned) / float64(totalSeats)) * 100
	}
	return reporting.Table{
		Title:   "License Utilization",
		Columns: []string{"License", "Vendor", "Product", "Seat Count", "Assigned Seats", "Available Seats", "Utilization"},
		Rows:    rows,
		Totals:  []reporting.SummaryRow{{Label: "Total", Values: []string{fmt.Sprintf("%d", totalSeats), fmt.Sprintf("%d", totalAssigned), fmt.Sprintf("%d", totalAvailable), fmt.Sprintf("%.2f%%", overall)}}},
	}, nil
}

func (s *reportingService) DepartmentSpend(ctx context.Context, params ReportingAsOfParams) (reporting.Table, error) {
	licenses, _, _, err := s.loadCatalog(ctx)
	if err != nil {
		return reporting.Table{}, err
	}
	totals := map[string]map[string]summaryBucket{}
	for _, lic := range licenses {
		department := strings.TrimSpace(lic.Department)
		if department == "" {
			department = "Unassigned"
		}
		currency := currencyBucket(lic.Currency)
		if _, ok := totals[department]; !ok {
			totals[department] = map[string]summaryBucket{}
		}
		bucket := totals[department][currency]
		bucket.count++
		bucket.total = bucket.total.Add(lic.Cost)
		totals[department][currency] = bucket
	}
	rows := make([][]string, 0)
	departments := sortedKeys(totals)
	for _, department := range departments {
		currencies := sortedCurrencyKeys(totals[department])
		for _, currency := range currencies {
			bucket := totals[department][currency]
			rows = append(rows, []string{department, currency, bucket.total.StringFixedBank(2), fmt.Sprintf("%d", bucket.count)})
		}
	}
	return reporting.Table{
		Title:   "Department Spend",
		Columns: []string{"Department", "Currency", "Total Cost", "License Count"},
		Rows:    rows,
		Totals:  summaryRowsForCurrencyTotals(aggregateCurrencyTotals(licenses)),
	}, nil
}

type summaryBucket struct {
	total decimal.Decimal
	count int
}

func (s *reportingService) loadCatalog(ctx context.Context) ([]domain.License, map[uuid.UUID]string, map[uuid.UUID]string, error) {
	licenses, err := s.listLicenses(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	vendors, err := s.listVendors(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	products, err := s.listProducts(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	vendorNames := make(map[uuid.UUID]string, len(vendors))
	for _, vendor := range vendors {
		vendorNames[vendor.ID] = vendor.Name
	}
	productNames := make(map[uuid.UUID]string, len(products))
	for _, product := range products {
		productNames[product.ID] = product.Name
	}
	return licenses, vendorNames, productNames, nil
}

func (s *reportingService) listLicenses(ctx context.Context) ([]domain.License, error) {
	if s.licenses == nil {
		return nil, fmt.Errorf("license repository is required")
	}
	return listAll(ctx, s.licenses.List)
}

func (s *reportingService) listVendors(ctx context.Context) ([]domain.Vendor, error) {
	if s.vendors == nil {
		return nil, fmt.Errorf("vendor repository is required")
	}
	return listAll(ctx, s.vendors.List)
}

func (s *reportingService) listProducts(ctx context.Context) ([]domain.Product, error) {
	if s.products == nil {
		return nil, fmt.Errorf("product repository is required")
	}
	return listAll(ctx, s.products.List)
}

func listAll[T any](ctx context.Context, fn func(context.Context, domain.ListFilter) ([]T, error)) ([]T, error) {
	const pageSize = 500
	var out []T
	for offset := 0; ; offset += pageSize {
		batch, err := fn(ctx, domain.ListFilter{Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return out, nil
}

func normalizeAsOf(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func licenseLabel(lic domain.License) string {
	if strings.TrimSpace(lic.LicenseKey) != "" {
		return lic.LicenseKey
	}
	return lic.ID.String()
}

func productName(products map[uuid.UUID]string, id uuid.UUID) string {
	if name := strings.TrimSpace(products[id]); name != "" {
		return name
	}
	return id.String()
}

func vendorName(vendors map[uuid.UUID]string, id uuid.UUID) string {
	if name := strings.TrimSpace(vendors[id]); name != "" {
		return name
	}
	return id.String()
}

func currencyBucket(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return "Unspecified"
	}
	return value
}

func daysUntil(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	return int(to.Sub(from).Hours() / 24)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func expiredAt(lic domain.License, asOf time.Time) (*time.Time, string) {
	var candidates []struct {
		when  *time.Time
		label string
	}
	if lic.ExpirationDate != nil && !lic.ExpirationDate.After(asOf) {
		candidates = append(candidates, struct {
			when  *time.Time
			label string
		}{lic.ExpirationDate, "Expiration Date"})
	}
	if lic.MaintenanceExpiration != nil && !lic.MaintenanceExpiration.After(asOf) {
		candidates = append(candidates, struct {
			when  *time.Time
			label string
		}{lic.MaintenanceExpiration, "Maintenance Expiration"})
	}
	if len(candidates) == 0 {
		return nil, ""
	}
	if len(candidates) == 1 {
		return candidates[0].when, candidates[0].label
	}
	when := candidates[0].when
	label := "Both"
	if candidates[1].when.Before(*when) {
		when = candidates[1].when
	}
	return when, label
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedCurrencyKeys[V any](m map[string]V) []string {
	return sortedKeys(m)
}

func aggregateCurrencyTotals(licenses []domain.License) map[string]summaryBucket {
	totals := map[string]summaryBucket{}
	for _, lic := range licenses {
		currency := currencyBucket(lic.Currency)
		bucket := totals[currency]
		bucket.count++
		bucket.total = bucket.total.Add(lic.Cost)
		totals[currency] = bucket
	}
	return totals
}

func summaryRowsForCurrencyTotals(totals map[string]summaryBucket) []reporting.SummaryRow {
	if len(totals) == 0 {
		return nil
	}
	rows := make([]reporting.SummaryRow, 0, len(totals))
	for _, currency := range sortedKeys(totals) {
		bucket := totals[currency]
		rows = append(rows, reporting.SummaryRow{Label: fmt.Sprintf("%s total", currency), Values: []string{bucket.total.StringFixedBank(2), fmt.Sprintf("%d", bucket.count)}})
	}
	return rows
}
