package app

import (
	"context"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type reportingVendorRepoFake struct{ items map[uuid.UUID]domain.Vendor }

func (r *reportingVendorRepoFake) Create(_ context.Context, v *domain.Vendor) error {
	if r.items == nil {
		r.items = map[uuid.UUID]domain.Vendor{}
	}
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	r.items[v.ID] = *v
	return nil
}
func (r *reportingVendorRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Vendor, error) {
	v, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &v, nil
}
func (r *reportingVendorRepoFake) Update(_ context.Context, v *domain.Vendor) error {
	r.items[v.ID] = *v
	return nil
}
func (r *reportingVendorRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *reportingVendorRepoFake) List(context.Context, domain.ListFilter) ([]domain.Vendor, error) {
	out := make([]domain.Vendor, 0, len(r.items))
	for _, v := range r.items {
		out = append(out, v)
	}
	return out, nil
}

type reportingProductRepoFake struct{ items map[uuid.UUID]domain.Product }

func (r *reportingProductRepoFake) Create(_ context.Context, p *domain.Product) error {
	if r.items == nil {
		r.items = map[uuid.UUID]domain.Product{}
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	r.items[p.ID] = *p
	return nil
}
func (r *reportingProductRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	p, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &p, nil
}
func (r *reportingProductRepoFake) Update(_ context.Context, p *domain.Product) error {
	r.items[p.ID] = *p
	return nil
}
func (r *reportingProductRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *reportingProductRepoFake) List(context.Context, domain.ListFilter) ([]domain.Product, error) {
	out := make([]domain.Product, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, p)
	}
	return out, nil
}

type reportingLicenseRepoFake struct{ items []domain.License }

func (r *reportingLicenseRepoFake) Create(context.Context, *domain.License) error { return nil }
func (r *reportingLicenseRepoFake) Get(context.Context, uuid.UUID) (*domain.License, error) {
	return nil, domain.ErrNotFound
}
func (r *reportingLicenseRepoFake) Update(context.Context, *domain.License) error { return nil }
func (r *reportingLicenseRepoFake) Delete(context.Context, uuid.UUID) error       { return nil }
func (r *reportingLicenseRepoFake) List(context.Context, domain.ListFilter) ([]domain.License, error) {
	return append([]domain.License(nil), r.items...), nil
}

func TestReportingServiceUpcomingRenewalsAndExpired(t *testing.T) {
	asOf := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	vendorID := uuid.New()
	productID := uuid.New()
	repo := &reportingLicenseRepoFake{items: []domain.License{
		{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, LicenseKey: "in-window", RenewalDate: timePtr(asOf.AddDate(0, 0, 30)), Cost: decimal.RequireFromString("10.00"), Currency: "usd", Type: domain.LicenseTypeSubscription},
		{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, LicenseKey: "upper-bound", RenewalDate: timePtr(asOf.AddDate(0, 0, 90)), Cost: decimal.RequireFromString("20.00"), Currency: "EUR", Type: domain.LicenseTypeSubscription},
		{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, LicenseKey: "expired", ExpirationDate: timePtr(asOf.AddDate(0, 0, -2)), MaintenanceExpiration: timePtr(asOf.AddDate(0, 0, -1)), Cost: decimal.RequireFromString("30.00"), Currency: "USD", Type: domain.LicenseTypeSubscription},
		{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, LicenseKey: "future", RenewalDate: timePtr(asOf.AddDate(0, 0, 91)), Cost: decimal.RequireFromString("40.00"), Currency: "USD", Type: domain.LicenseTypeSubscription},
	}}
	vendors := &reportingVendorRepoFake{items: map[uuid.UUID]domain.Vendor{vendorID: {Base: domain.Base{ID: vendorID}, Name: "Acme"}}}
	products := &reportingProductRepoFake{items: map[uuid.UUID]domain.Product{productID: {Base: domain.Base{ID: productID}, Name: "Widget"}}}
	svc := NewReportingService(vendors, products, repo)

	renewals, err := svc.UpcomingRenewals(context.Background(), UpcomingRenewalsParams{AsOf: asOf, WindowDays: 90})
	if err != nil {
		t.Fatalf("renewals: %v", err)
	}
	if len(renewals.Rows) != 2 {
		t.Fatalf("expected 2 upcoming renewals, got %d", len(renewals.Rows))
	}
	if renewals.Rows[0][0] != "in-window" || renewals.Rows[1][0] != "upper-bound" {
		t.Fatalf("unexpected renewals: %#v", renewals.Rows)
	}

	expired, err := svc.ExpiredLicenses(context.Background(), ExpiredLicensesParams{AsOf: asOf})
	if err != nil {
		t.Fatalf("expired: %v", err)
	}
	if len(expired.Rows) != 1 || expired.Rows[0][0] != "expired" {
		t.Fatalf("unexpected expired rows: %#v", expired.Rows)
	}
	if expired.Rows[0][6] != "Both" {
		t.Fatalf("expected both expiration sources, got %q", expired.Rows[0][6])
	}
}

func TestReportingServiceAggregationsAndUtilization(t *testing.T) {
	vendorID := uuid.New()
	otherVendorID := uuid.New()
	productID := uuid.New()
	repo := &reportingLicenseRepoFake{items: []domain.License{
		{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, Department: "Finance", SeatCount: 10, AssignedSeats: 4, Cost: decimal.RequireFromString("12.50"), Currency: "USD", Type: domain.LicenseTypeSubscription},
		{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, Department: "", SeatCount: 0, AssignedSeats: 0, Cost: decimal.RequireFromString("7.50"), Currency: "USD", Type: domain.LicenseTypeSubscription},
		{Base: domain.Base{ID: uuid.New()}, VendorID: otherVendorID, ProductID: productID, Department: "Finance", SeatCount: 5, AssignedSeats: 5, Cost: decimal.RequireFromString("9.00"), Currency: "EUR", Type: domain.LicenseTypeSubscription},
	}}
	vendors := &reportingVendorRepoFake{items: map[uuid.UUID]domain.Vendor{vendorID: {Base: domain.Base{ID: vendorID}, Name: "Acme"}, otherVendorID: {Base: domain.Base{ID: otherVendorID}, Name: "Beta"}}}
	products := &reportingProductRepoFake{items: map[uuid.UUID]domain.Product{productID: {Base: domain.Base{ID: productID}, Name: "Widget"}}}
	svc := NewReportingService(vendors, products, repo)

	vendorSpend, err := svc.VendorSpend(context.Background(), ReportingAsOfParams{})
	if err != nil {
		t.Fatalf("vendor spend: %v", err)
	}
	if len(vendorSpend.Rows) != 2 {
		t.Fatalf("expected 2 vendor spend rows, got %d", len(vendorSpend.Rows))
	}
	if vendorSpend.Rows[0][0] != "Acme" || vendorSpend.Rows[0][1] != "USD" {
		t.Fatalf("unexpected vendor spend row: %#v", vendorSpend.Rows[0])
	}
	departmentSpend, err := svc.DepartmentSpend(context.Background(), ReportingAsOfParams{})
	if err != nil {
		t.Fatalf("department spend: %v", err)
	}
	if len(departmentSpend.Rows) != 3 {
		t.Fatalf("expected 3 department spend rows, got %d", len(departmentSpend.Rows))
	}
	if departmentSpend.Rows[0][0] != "Finance" || departmentSpend.Rows[1][0] != "Finance" || departmentSpend.Rows[2][0] != "Unassigned" {
		t.Fatalf("unexpected department rows: %#v", departmentSpend.Rows)
	}
	utilization, err := svc.LicenseUtilization(context.Background(), ReportingAsOfParams{})
	if err != nil {
		t.Fatalf("utilization: %v", err)
	}
	if len(utilization.Rows) != 3 {
		t.Fatalf("expected 3 utilization rows, got %d", len(utilization.Rows))
	}
	if utilization.Rows[1][6] != "0.00%" {
		t.Fatalf("expected zero-seat license utilization to be safe, got %s", utilization.Rows[1][6])
	}
	if len(utilization.Totals) != 1 || utilization.Totals[0].Values[3] != "60.00%" {
		t.Fatalf("unexpected totals: %#v", utilization.Totals)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
