//go:build integration

package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("LICENSEIQ_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LICENSEIQ_TEST_POSTGRES_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { truncateAll(t, db) })
	return db
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE TABLE renewal_reminder_logs, feature_flag_audits, feature_flags, audit_logs, api_keys, users, attachments, assignments, licenses, products, vendor_contacts, vendors RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestVendorRepositoryCRUD(t *testing.T) {
	db := testDB(t)
	repo := NewVendorRepository(db)

	vendor := &domain.Vendor{
		Name:       "Acme",
		SupportURL: "https://example.com",
		Contacts:   []domain.VendorContact{{Name: "Jane", Email: "jane@example.com"}},
	}
	if err := repo.Create(context.Background(), vendor); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(context.Background(), vendor.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != vendor.Name || len(got.Contacts) != 1 {
		t.Fatalf("unexpected vendor: %#v", got)
	}

	vendor.Name = "Acme Updated"
	vendor.Contacts = append(vendor.Contacts, domain.VendorContact{Name: "Bob", Email: "bob@example.com"})
	if err := repo.Update(context.Background(), vendor); err != nil {
		t.Fatalf("update: %v", err)
	}
	list, err := repo.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || len(list[0].Contacts) != 2 {
		t.Fatalf("unexpected list: %#v", list)
	}
	if err := repo.Delete(context.Background(), vendor.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(context.Background(), vendor.ID); err != domain.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	list, err = repo.List(context.Background(), domain.ListFilter{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("list include deleted: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected soft deleted row in list")
	}
}

func TestProductLicenseAssignmentAndFeatureFlagRepositories(t *testing.T) {
	db := testDB(t)
	vendorRepo := NewVendorRepository(db)
	productRepo := NewProductRepository(db)
	licenseRepo := NewLicenseRepository(db)
	assignmentRepo := NewAssignmentRepository(db)
	flagRepo := NewFeatureFlagRepository(db)

	vendor := &domain.Vendor{Name: "Acme"}
	if err := vendorRepo.Create(context.Background(), vendor); err != nil {
		t.Fatalf("vendor create: %v", err)
	}
	product := &domain.Product{Name: "Widget", VendorID: vendor.ID, Tags: []string{"cloud"}}
	if err := productRepo.Create(context.Background(), product); err != nil {
		t.Fatalf("product create: %v", err)
	}
	license := &domain.License{
		ProductID:     product.ID,
		VendorID:      vendor.ID,
		SeatCount:     1,
		AssignedSeats: 0,
		Cost:          decimal.RequireFromString("19.99"),
		Currency:      "USD",
		Type:          domain.LicenseTypeSubscription,
	}
	if err := licenseRepo.Create(context.Background(), license); err != nil {
		t.Fatalf("license create: %v", err)
	}

	assignment := &domain.Assignment{
		LicenseID:  license.ID,
		TargetType: domain.AssignmentTargetUser,
		TargetID:   uuid.NewString(),
		TargetName: "Jane",
		AssignedAt: time.Now().UTC(),
	}
	if err := assignmentRepo.Create(context.Background(), assignment); err != nil {
		t.Fatalf("assignment create: %v", err)
	}
	license, err := licenseRepo.Get(context.Background(), license.ID)
	if err != nil {
		t.Fatalf("license get: %v", err)
	}
	if license.AssignedSeats != 1 {
		t.Fatalf("assigned seats = %d", license.AssignedSeats)
	}

	overAlloc := &domain.Assignment{
		LicenseID:  license.ID,
		TargetType: domain.AssignmentTargetDevice,
		TargetID:   uuid.NewString(),
		TargetName: "Device 2",
		AssignedAt: time.Now().UTC(),
	}
	if err := assignmentRepo.Create(context.Background(), overAlloc); err != domain.ErrConflict {
		t.Fatalf("expected over-allocation conflict, got %v", err)
	}
	if err := assignmentRepo.Delete(context.Background(), assignment.ID); err != nil {
		t.Fatalf("assignment delete: %v", err)
	}
	license, err = licenseRepo.Get(context.Background(), license.ID)
	if err != nil {
		t.Fatalf("license get after delete: %v", err)
	}
	if license.AssignedSeats != 0 {
		t.Fatalf("assigned seats after delete = %d", license.AssignedSeats)
	}

	flag := &domain.FeatureFlag{Key: "new-ui", Description: "New UI", Enabled: true, PercentageRollout: 50, TargetRoles: []domain.Role{domain.RoleViewer}}
	if err := flagRepo.Create(context.Background(), flag); err != nil {
		t.Fatalf("flag create: %v", err)
	}
	flag.Enabled = false
	flag.PercentageRollout = 25
	if err := flagRepo.Update(context.Background(), flag); err != nil {
		t.Fatalf("flag update: %v", err)
	}
	flags, err := flagRepo.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("flag list: %v", err)
	}
	if len(flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(flags))
	}
	if err := flagRepo.Delete(context.Background(), flag.ID); err != nil {
		t.Fatalf("flag delete: %v", err)
	}
	if _, err := flagRepo.Get(context.Background(), flag.ID); err != domain.ErrNotFound {
		t.Fatalf("expected flag not found, got %v", err)
	}
}

func TestReportingServiceIntegration(t *testing.T) {
	db := testDB(t)
	vendorRepo := NewVendorRepository(db)
	productRepo := NewProductRepository(db)
	licenseRepo := NewLicenseRepository(db)

	vendor := &domain.Vendor{Name: "Acme"}
	if err := vendorRepo.Create(context.Background(), vendor); err != nil {
		t.Fatalf("vendor create: %v", err)
	}
	product := &domain.Product{Name: "Widget", VendorID: vendor.ID}
	if err := productRepo.Create(context.Background(), product); err != nil {
		t.Fatalf("product create: %v", err)
	}
	renewal := time.Now().UTC().AddDate(0, 0, 30)
	license := &domain.License{ProductID: product.ID, VendorID: vendor.ID, Department: "Finance", LicenseKey: "ACME-1", RenewalDate: &renewal, SeatCount: 10, AssignedSeats: 4, Cost: decimal.RequireFromString("12.50"), Currency: "USD", Type: domain.LicenseTypeSubscription}
	if err := licenseRepo.Create(context.Background(), license); err != nil {
		t.Fatalf("license create: %v", err)
	}
	service := app.NewReportingService(vendorRepo, productRepo, licenseRepo)
	renewals, err := service.UpcomingRenewals(context.Background(), app.UpcomingRenewalsParams{AsOf: time.Now().UTC(), WindowDays: 90})
	if err != nil {
		t.Fatalf("renewals: %v", err)
	}
	if len(renewals.Rows) != 1 || renewals.Rows[0][0] != "ACME-1" {
		t.Fatalf("unexpected renewals: %#v", renewals.Rows)
	}
	utilization, err := service.LicenseUtilization(context.Background(), app.ReportingAsOfParams{AsOf: time.Now().UTC()})
	if err != nil {
		t.Fatalf("utilization: %v", err)
	}
	if len(utilization.Rows) != 1 || utilization.Rows[0][0] != "ACME-1" {
		t.Fatalf("unexpected utilization: %#v", utilization.Rows)
	}
}
