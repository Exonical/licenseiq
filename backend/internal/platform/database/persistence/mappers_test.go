package persistence

import (
	"reflect"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestVendorRoundTripMapping(t *testing.T) {
	vendor := domain.Vendor{
		Base: domain.Base{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		Name: "Acme",
		Contacts: []domain.VendorContact{{
			Name:  "Jane",
			Email: "jane@example.com",
		}},
	}
	got := vendorToDomain(vendorToModel(vendor))
	if !reflect.DeepEqual(vendor.Name, got.Name) || !reflect.DeepEqual(vendor.Contacts, got.Contacts) {
		t.Fatalf("vendor round trip mismatch: %#v != %#v", vendor, got)
	}
}

func TestLicenseRoundTripMapping(t *testing.T) {
	now := time.Now().UTC()
	license := domain.License{
		Base:          domain.Base{ID: uuid.New(), CreatedAt: now},
		ProductID:     uuid.New(),
		VendorID:      uuid.New(),
		SeatCount:     4,
		AssignedSeats: 2,
		Cost:          decimal.RequireFromString("12.34"),
		Currency:      "USD",
		Type:          domain.LicenseTypeSubscription,
		PurchaseDate:  &now,
	}
	got := licenseToDomain(licenseToModel(license))
	if got.SeatCount != license.SeatCount || !got.Cost.Equal(license.Cost) || got.Type != license.Type {
		t.Fatalf("license round trip mismatch: %#v != %#v", license, got)
	}
}

func TestFeatureFlagRoundTripMapping(t *testing.T) {
	now := time.Now().UTC()
	flag := domain.FeatureFlag{
		Base:        domain.Base{ID: uuid.New(), CreatedAt: now},
		Key:         "new-ui",
		TargetRoles: []domain.Role{domain.RoleViewer},
	}
	got := featureFlagToDomain(featureFlagToModel(flag))
	if got.Key != flag.Key || !reflect.DeepEqual(got.TargetRoles, flag.TargetRoles) {
		t.Fatalf("feature flag round trip mismatch: %#v != %#v", flag, got)
	}
}
