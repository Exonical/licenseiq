//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

func TestLicenseIssueLinkRepositoryCRUD(t *testing.T) {
	db := testDB(t)
	vendorRepo := NewVendorRepository(db)
	productRepo := NewProductRepository(db)
	licenseRepo := NewLicenseRepository(db)
	linkRepo := NewLicenseIssueLinkRepository(db)

	vendor := &domain.Vendor{Name: "Acme"}
	if err := vendorRepo.Create(context.Background(), vendor); err != nil {
		t.Fatalf("vendor create: %v", err)
	}
	product := &domain.Product{Name: "Widget", VendorID: vendor.ID}
	if err := productRepo.Create(context.Background(), product); err != nil {
		t.Fatalf("product create: %v", err)
	}
	renewal := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	license := &domain.License{ProductID: product.ID, VendorID: vendor.ID, RenewalDate: &renewal, Type: domain.LicenseTypeSubscription}
	if err := licenseRepo.Create(context.Background(), license); err != nil {
		t.Fatalf("license create: %v", err)
	}

	link := &domain.LicenseIssueLink{LicenseID: license.ID, IssueKey: "ABC-1", IssueURL: "https://jira.example.com/browse/ABC-1", Status: "Open", RenewalDate: &renewal}
	if err := linkRepo.Create(context.Background(), link); err != nil {
		t.Fatalf("link create: %v", err)
	}
	if link.ID == uuid.Nil {
		t.Fatalf("expected id to be populated")
	}
	got, err := linkRepo.Get(context.Background(), link.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.IssueKey != link.IssueKey || got.Status != link.Status {
		t.Fatalf("unexpected link: %#v", got)
	}
	byIssue, err := linkRepo.GetByLicenseAndIssueKey(context.Background(), license.ID, "ABC-1")
	if err != nil {
		t.Fatalf("get by issue: %v", err)
	}
	if byIssue.ID != link.ID {
		t.Fatalf("unexpected link by issue: %#v", byIssue)
	}
	byRenewal, err := linkRepo.GetByLicenseAndRenewalDate(context.Background(), license.ID, renewal)
	if err != nil {
		t.Fatalf("get by renewal: %v", err)
	}
	if byRenewal.ID != link.ID {
		t.Fatalf("unexpected link by renewal: %#v", byRenewal)
	}
	list, err := linkRepo.ListByLicense(context.Background(), license.ID)
	if err != nil {
		t.Fatalf("list by license: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one link, got %d", len(list))
	}
}
