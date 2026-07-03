//go:build integration

package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/Exonical/licenseiq/backend/internal/platform/database/persistence"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func workerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("LICENSEIQ_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LICENSEIQ_TEST_POSTGRES_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := persistence.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec(`TRUNCATE TABLE renewal_reminder_logs, api_keys, users, assignments, attachments, licenses, products, vendor_contacts, vendors RESTART IDENTITY CASCADE`).Error
	})
	return db
}

func TestWorkerIntegrationRenewalLogAndApiKeyMaintenance(t *testing.T) {
	db := workerTestDB(t)
	vendorRepo := persistence.NewVendorRepository(db)
	productRepo := persistence.NewProductRepository(db)
	licenseRepo := persistence.NewLicenseRepository(db)
	reminderRepo := persistence.NewRenewalReminderLogRepository(db)
	apiKeyRepo := persistence.NewAPIKeyRepository(db)

	vendor := &domain.Vendor{Name: "Vendor"}
	if err := vendorRepo.Create(context.Background(), vendor); err != nil {
		t.Fatalf("vendor: %v", err)
	}
	product := &domain.Product{Name: "Product", VendorID: vendor.ID}
	if err := productRepo.Create(context.Background(), product); err != nil {
		t.Fatalf("product: %v", err)
	}
	renewal := time.Now().UTC().AddDate(0, 0, 30)
	license := &domain.License{VendorID: vendor.ID, ProductID: product.ID, RenewalDate: &renewal, LicenseKey: "LIC-INT", Cost: decimal.RequireFromString("10.00"), Currency: "USD"}
	if err := licenseRepo.Create(context.Background(), license); err != nil {
		t.Fatalf("license: %v", err)
	}
	job := &RenewalReminderJob{licenses: licenseRepo, products: productRepo, vendors: vendorRepo, logs: reminderRepo, dispatcher: noopDispatcher{}, clock: func() time.Time { return time.Now().UTC() }, thresholds: []int{30}}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("reminder run: %v", err)
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("reminder rerun: %v", err)
	}
	logs, err := reminderRepo.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("list reminder logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one reminder log, got %d", len(logs))
	}

	expired := time.Now().UTC().Add(-time.Hour)
	key := &domain.APIKey{OwnerUserID: uuid.New(), Name: "expired", HashedKey: "x", Active: true, ExpiresAt: &expired}
	if err := apiKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("api key create: %v", err)
	}
	maint := &MaintenanceJob{keys: apiKeyRepo, clock: func() time.Time { return time.Now().UTC() }}
	if err := maint.Run(context.Background()); err != nil {
		t.Fatalf("maintenance run: %v", err)
	}
	stored, err := apiKeyRepo.Get(context.Background(), key.ID)
	if err != nil {
		t.Fatalf("api key get: %v", err)
	}
	if stored.Active {
		t.Fatalf("expected key inactive after maintenance")
	}
}

type noopDispatcher struct{}

func (noopDispatcher) Send(context.Context, notify.Message) error { return nil }
