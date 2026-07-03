package api

import (
	"fmt"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ListInput struct {
	Limit          int  `query:"limit" minimum:"1" maximum:"500" default:"100" example:"100"`
	Offset         int  `query:"offset" minimum:"0" default:"0" example:"0"`
	IncludeDeleted bool `query:"includeDeleted" example:"false"`
}

type Page[T any] struct {
	Data   []T `json:"data"`
	Limit  int `json:"limit" example:"100"`
	Offset int `json:"offset" example:"0"`
	Total  int `json:"total" example:"1"`
}

type VendorContactBody struct {
	Name  string `json:"name" example:"Jane Doe"`
	Email string `json:"email,omitempty" example:"jane@example.com"`
	Phone string `json:"phone,omitempty" example:"+1-555-0100"`
	Role  string `json:"role,omitempty" example:"Account Manager"`
}

type VendorBody struct {
	Name           string              `json:"name" example:"Acme Software"`
	SupportURL     string              `json:"supportUrl,omitempty" example:"https://support.example.com"`
	AccountManager string              `json:"accountManager,omitempty" example:"Taylor"`
	Notes          string              `json:"notes,omitempty" example:"Preferred vendor"`
	Contacts       []VendorContactBody `json:"contacts,omitempty"`
}

type VendorResponse struct {
	ID             uuid.UUID           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt      time.Time           `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt      time.Time           `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt      *time.Time          `json:"deletedAt,omitempty"`
	Name           string              `json:"name" example:"Acme Software"`
	SupportURL     string              `json:"supportUrl,omitempty" example:"https://support.example.com"`
	AccountManager string              `json:"accountManager,omitempty" example:"Taylor"`
	Notes          string              `json:"notes,omitempty" example:"Preferred vendor"`
	Contacts       []VendorContactBody `json:"contacts,omitempty"`
}

type VendorCreateInput struct{ Body VendorBody }
type VendorUpdateInput struct{ Body VendorBody }
type VendorGetOutput struct{ Body VendorResponse }
type VendorListOutput struct{ Body Page[VendorResponse] }
type VendorDeleteOutput struct{}

type ProductBody struct {
	Name        string    `json:"name" example:"LicenseIQ"`
	VendorID    uuid.UUID `json:"vendorId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Category    string    `json:"category,omitempty" example:"Security"`
	Version     string    `json:"version,omitempty" example:"1.0.0"`
	Website     string    `json:"website,omitempty" example:"https://example.com"`
	Description string    `json:"description,omitempty" example:"Asset management"`
	Tags        []string  `json:"tags,omitempty" example:"[\"saas\",\"security\"]"`
}

type ProductResponse struct {
	ID          uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt   time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt   time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
	Name        string     `json:"name" example:"LicenseIQ"`
	VendorID    uuid.UUID  `json:"vendorId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Category    string     `json:"category,omitempty" example:"Security"`
	Version     string     `json:"version,omitempty" example:"1.0.0"`
	Website     string     `json:"website,omitempty" example:"https://example.com"`
	Description string     `json:"description,omitempty" example:"Asset management"`
	Tags        []string   `json:"tags,omitempty" example:"[\"saas\",\"security\"]"`
}

type ProductCreateInput struct{ Body ProductBody }
type ProductUpdateInput struct{ Body ProductBody }
type ProductGetOutput struct{ Body ProductResponse }
type ProductListOutput struct{ Body Page[ProductResponse] }
type ProductDeleteOutput struct{}

type LicenseBody struct {
	ProductID             uuid.UUID  `json:"productId" example:"550e8400-e29b-41d4-a716-446655440000"`
	VendorID              uuid.UUID  `json:"vendorId" example:"550e8400-e29b-41d4-a716-446655440000"`
	LicenseKey            string     `json:"licenseKey,omitempty" example:"AAAA-BBBB-CCCC"`
	SubscriptionID        string     `json:"subscriptionId,omitempty" example:"sub-123"`
	ContractNumber        string     `json:"contractNumber,omitempty" example:"ctr-123"`
	PurchaseOrder         string     `json:"purchaseOrder,omitempty" example:"po-123"`
	Invoice               string     `json:"invoice,omitempty" example:"inv-123"`
	PurchaseDate          *time.Time `json:"purchaseDate,omitempty" example:"2026-01-01T00:00:00Z"`
	RenewalDate           *time.Time `json:"renewalDate,omitempty" example:"2027-01-01T00:00:00Z"`
	ExpirationDate        *time.Time `json:"expirationDate,omitempty" example:"2027-12-31T00:00:00Z"`
	MaintenanceExpiration *time.Time `json:"maintenanceExpiration,omitempty" example:"2027-06-30T00:00:00Z"`
	SeatCount             int        `json:"seatCount" example:"10"`
	AssignedSeats         int        `json:"assignedSeats" example:"2"`
	Cost                  string     `json:"cost,omitempty" example:"1234.56"`
	Currency              string     `json:"currency,omitempty" example:"USD"`
	Notes                 string     `json:"notes,omitempty" example:"Annual subscription"`
	Type                  string     `json:"type" enum:"Subscription,Perpetual,PerUser,PerDevice,PerCore,Concurrent,Enterprise" example:"Subscription"`
}

type LicenseResponse struct {
	ID                    uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt             time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt             time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt             *time.Time `json:"deletedAt,omitempty"`
	ProductID             uuid.UUID  `json:"productId" example:"550e8400-e29b-41d4-a716-446655440000"`
	VendorID              uuid.UUID  `json:"vendorId" example:"550e8400-e29b-41d4-a716-446655440000"`
	LicenseKey            string     `json:"licenseKey,omitempty" example:"AAAA-BBBB-CCCC"`
	SubscriptionID        string     `json:"subscriptionId,omitempty" example:"sub-123"`
	ContractNumber        string     `json:"contractNumber,omitempty" example:"ctr-123"`
	PurchaseOrder         string     `json:"purchaseOrder,omitempty" example:"po-123"`
	Invoice               string     `json:"invoice,omitempty" example:"inv-123"`
	PurchaseDate          *time.Time `json:"purchaseDate,omitempty" example:"2026-01-01T00:00:00Z"`
	RenewalDate           *time.Time `json:"renewalDate,omitempty" example:"2027-01-01T00:00:00Z"`
	ExpirationDate        *time.Time `json:"expirationDate,omitempty" example:"2027-12-31T00:00:00Z"`
	MaintenanceExpiration *time.Time `json:"maintenanceExpiration,omitempty" example:"2027-06-30T00:00:00Z"`
	SeatCount             int        `json:"seatCount" example:"10"`
	AssignedSeats         int        `json:"assignedSeats" example:"2"`
	AvailableSeats        int        `json:"availableSeats" example:"8"`
	Cost                  string     `json:"cost,omitempty" example:"1234.56"`
	Currency              string     `json:"currency,omitempty" example:"USD"`
	Notes                 string     `json:"notes,omitempty" example:"Annual subscription"`
	Type                  string     `json:"type" enum:"Subscription,Perpetual,PerUser,PerDevice,PerCore,Concurrent,Enterprise" example:"Subscription"`
}

type LicenseCreateInput struct{ Body LicenseBody }
type LicenseUpdateInput struct{ Body LicenseBody }
type LicenseGetOutput struct{ Body LicenseResponse }
type LicenseListOutput struct{ Body Page[LicenseResponse] }
type LicenseDeleteOutput struct{}

type AssignmentBody struct {
	LicenseID  uuid.UUID `json:"licenseId" example:"550e8400-e29b-41d4-a716-446655440000"`
	TargetType string    `json:"targetType" enum:"User,Device,Server,VirtualMachine" example:"User"`
	TargetID   string    `json:"targetId" example:"user-123"`
	TargetName string    `json:"targetName" example:"Jane Doe"`
	AssignedAt time.Time `json:"assignedAt" example:"2026-01-01T00:00:00Z"`
	Notes      string    `json:"notes,omitempty" example:"Primary user"`
}

type AssignmentResponse struct {
	ID         uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt  time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt  time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
	LicenseID  uuid.UUID  `json:"licenseId" example:"550e8400-e29b-41d4-a716-446655440000"`
	TargetType string     `json:"targetType" enum:"User,Device,Server,VirtualMachine" example:"User"`
	TargetID   string     `json:"targetId" example:"user-123"`
	TargetName string     `json:"targetName" example:"Jane Doe"`
	AssignedAt time.Time  `json:"assignedAt" example:"2026-01-01T00:00:00Z"`
	Notes      string     `json:"notes,omitempty" example:"Primary user"`
}

type AssignmentCreateInput struct{ Body AssignmentBody }
type AssignmentUpdateInput struct{ Body AssignmentBody }
type AssignmentGetOutput struct{ Body AssignmentResponse }
type AssignmentListOutput struct{ Body Page[AssignmentResponse] }
type AssignmentDeleteOutput struct{}

type AttachmentBody struct {
	OwnerType        string     `json:"ownerType" enum:"license,vendor,product" example:"license"`
	OwnerID          uuid.UUID  `json:"ownerId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Filename         string     `json:"filename" example:"invoice.pdf"`
	ContentType      string     `json:"contentType,omitempty" example:"application/pdf"`
	SizeBytes        int64      `json:"sizeBytes" example:"12345"`
	StorageKey       string     `json:"storageKey" example:"attachments/uuid/invoice.pdf"`
	UploadedByUserID *uuid.UUID `json:"uploadedByUserId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	UploadedAt       time.Time  `json:"uploadedAt" example:"2026-01-01T00:00:00Z"`
}

type AttachmentResponse struct {
	ID               uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt        time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt        time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt        *time.Time `json:"deletedAt,omitempty"`
	OwnerType        string     `json:"ownerType" enum:"license,vendor,product" example:"license"`
	OwnerID          uuid.UUID  `json:"ownerId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Filename         string     `json:"filename" example:"invoice.pdf"`
	ContentType      string     `json:"contentType,omitempty" example:"application/pdf"`
	SizeBytes        int64      `json:"sizeBytes" example:"12345"`
	StorageKey       string     `json:"storageKey" example:"attachments/uuid/invoice.pdf"`
	UploadedByUserID *uuid.UUID `json:"uploadedByUserId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	UploadedAt       time.Time  `json:"uploadedAt" example:"2026-01-01T00:00:00Z"`
}

type AttachmentCreateInput struct{ Body AttachmentBody }
type AttachmentGetOutput struct{ Body AttachmentResponse }
type AttachmentListOutput struct{ Body Page[AttachmentResponse] }
type AttachmentDeleteOutput struct{}

func vendorToResponse(v domain.Vendor) VendorResponse {
	contacts := make([]VendorContactBody, 0, len(v.Contacts))
	for _, c := range v.Contacts {
		contacts = append(contacts, VendorContactBody{Name: c.Name, Email: c.Email, Phone: c.Phone, Role: c.Role})
	}
	return VendorResponse{ID: v.ID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, DeletedAt: v.DeletedAt, Name: v.Name, SupportURL: v.SupportURL, AccountManager: v.AccountManager, Notes: v.Notes, Contacts: contacts}
}

func vendorBodyToDomain(b VendorBody) domain.Vendor {
	contacts := make([]domain.VendorContact, 0, len(b.Contacts))
	for _, c := range b.Contacts {
		contacts = append(contacts, domain.VendorContact{Name: c.Name, Email: c.Email, Phone: c.Phone, Role: c.Role})
	}
	return domain.Vendor{Name: b.Name, SupportURL: b.SupportURL, AccountManager: b.AccountManager, Notes: b.Notes, Contacts: contacts}
}

func productToResponse(p domain.Product) ProductResponse {
	return ProductResponse{ID: p.ID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, DeletedAt: p.DeletedAt, Name: p.Name, VendorID: p.VendorID, Category: p.Category, Version: p.Version, Website: p.Website, Description: p.Description, Tags: append([]string(nil), p.Tags...)}
}

func productBodyToDomain(b ProductBody) domain.Product {
	return domain.Product{Name: b.Name, VendorID: b.VendorID, Category: b.Category, Version: b.Version, Website: b.Website, Description: b.Description, Tags: append([]string(nil), b.Tags...)}
}

func licenseToResponse(l domain.License) LicenseResponse {
	return LicenseResponse{ID: l.ID, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt, DeletedAt: l.DeletedAt, ProductID: l.ProductID, VendorID: l.VendorID, LicenseKey: l.LicenseKey, SubscriptionID: l.SubscriptionID, ContractNumber: l.ContractNumber, PurchaseOrder: l.PurchaseOrder, Invoice: l.Invoice, PurchaseDate: l.PurchaseDate, RenewalDate: l.RenewalDate, ExpirationDate: l.ExpirationDate, MaintenanceExpiration: l.MaintenanceExpiration, SeatCount: l.SeatCount, AssignedSeats: l.AssignedSeats, AvailableSeats: l.AvailableSeats(), Cost: l.Cost.StringFixedBank(2), Currency: l.Currency, Notes: l.Notes, Type: l.Type.String()}
}

func licenseBodyToDomain(b LicenseBody) (domain.License, error) {
	cost := decimal.Zero
	var err error
	if b.Cost != "" {
		cost, err = decimal.NewFromString(b.Cost)
		if err != nil {
			return domain.License{}, fmt.Errorf("invalid cost: %w", err)
		}
	}
	return domain.License{ProductID: b.ProductID, VendorID: b.VendorID, LicenseKey: b.LicenseKey, SubscriptionID: b.SubscriptionID, ContractNumber: b.ContractNumber, PurchaseOrder: b.PurchaseOrder, Invoice: b.Invoice, PurchaseDate: b.PurchaseDate, RenewalDate: b.RenewalDate, ExpirationDate: b.ExpirationDate, MaintenanceExpiration: b.MaintenanceExpiration, SeatCount: b.SeatCount, AssignedSeats: b.AssignedSeats, Cost: cost, Currency: b.Currency, Notes: b.Notes, Type: domain.LicenseType(b.Type)}, nil
}

func assignmentToResponse(a domain.Assignment) AssignmentResponse {
	return AssignmentResponse{ID: a.ID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, DeletedAt: a.DeletedAt, LicenseID: a.LicenseID, TargetType: string(a.TargetType), TargetID: a.TargetID, TargetName: a.TargetName, AssignedAt: a.AssignedAt, Notes: a.Notes}
}

func assignmentBodyToDomain(b AssignmentBody) domain.Assignment {
	return domain.Assignment{LicenseID: b.LicenseID, TargetType: domain.AssignmentTargetType(b.TargetType), TargetID: b.TargetID, TargetName: b.TargetName, AssignedAt: b.AssignedAt, Notes: b.Notes}
}

func attachmentToResponse(a domain.Attachment) AttachmentResponse {
	return AttachmentResponse{ID: a.ID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, DeletedAt: a.DeletedAt, OwnerType: string(a.OwnerType), OwnerID: a.OwnerID, Filename: a.Filename, ContentType: a.ContentType, SizeBytes: a.SizeBytes, StorageKey: a.StorageKey, UploadedByUserID: a.UploadedByUserID, UploadedAt: a.UploadedAt}
}

func attachmentBodyToDomain(b AttachmentBody) domain.Attachment {
	return domain.Attachment{OwnerType: domain.AttachmentOwnerType(b.OwnerType), OwnerID: b.OwnerID, Filename: b.Filename, ContentType: b.ContentType, SizeBytes: b.SizeBytes, StorageKey: b.StorageKey, UploadedByUserID: b.UploadedByUserID, UploadedAt: b.UploadedAt}
}
