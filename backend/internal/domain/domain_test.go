package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestEnumValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		validate func() error
		wantErr  bool
	}{
		{name: "role valid", validate: RoleAdministrator.Validate},
		{name: "role invalid", validate: func() error { return Role("bad").Validate() }, wantErr: true},
		{name: "license valid", validate: LicenseTypeSubscription.Validate},
		{name: "license invalid", validate: func() error { return LicenseType("bad").Validate() }, wantErr: true},
		{name: "assignment valid", validate: AssignmentTargetUser.Validate},
		{name: "assignment invalid", validate: func() error { return AssignmentTargetType("bad").Validate() }, wantErr: true},
		{name: "attachment valid", validate: AttachmentOwnerVendor.Validate},
		{name: "attachment invalid", validate: func() error { return AttachmentOwnerType("bad").Validate() }, wantErr: true},
		{name: "audit valid", validate: AuditActionCreate.Validate},
		{name: "audit invalid", validate: func() error { return AuditAction("bad").Validate() }, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLicenseSeatMath(t *testing.T) {
	t.Parallel()

	license := License{
		ProductID:     uuid.New(),
		VendorID:      uuid.New(),
		SeatCount:     10,
		AssignedSeats: 3,
		Cost:          decimal.NewFromInt(1234),
		Currency:      "usd",
		Type:          LicenseTypeSubscription,
	}
	if err := license.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := license.AvailableSeats(); got != 7 {
		t.Fatalf("available seats = %d", got)
	}

	license.AssignedSeats = 11
	if err := license.Validate(); err == nil {
		t.Fatal("expected validation error for over-allocation")
	}
}

func TestVendorAndProductValidation(t *testing.T) {
	t.Parallel()

	vendor := Vendor{Name: "Acme", SupportURL: "https://example.com"}
	if err := vendor.Validate(); err != nil {
		t.Fatalf("vendor validate: %v", err)
	}
	product := Product{
		Name:     "Widget",
		VendorID: uuid.New(),
		Website:  "https://example.com",
		Tags:     []string{" a ", "b", "a"},
	}
	if err := product.Validate(); err != nil {
		t.Fatalf("product validate: %v", err)
	}
	if len(product.Tags) != 2 {
		t.Fatalf("tags not normalized: %#v", product.Tags)
	}
}

func TestAssignmentAndAttachmentValidation(t *testing.T) {
	t.Parallel()

	assignedAt := time.Now().UTC()
	assignment := Assignment{
		LicenseID:  uuid.New(),
		TargetType: AssignmentTargetServer,
		TargetID:   "srv-1",
		TargetName: "Server 1",
		AssignedAt: assignedAt,
	}
	if err := assignment.Validate(); err != nil {
		t.Fatalf("assignment validate: %v", err)
	}

	attachment := Attachment{
		OwnerType:   AttachmentOwnerLicense,
		OwnerID:     uuid.New(),
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		SizeBytes:   1024,
		StorageKey:  "objects/1",
		UploadedAt:  assignedAt,
	}
	if err := attachment.Validate(); err != nil {
		t.Fatalf("attachment validate: %v", err)
	}
}
