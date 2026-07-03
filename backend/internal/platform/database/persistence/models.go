package persistence

import (
	"encoding/json"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type AuditBaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VendorModel struct {
	BaseModel
	Name           string
	SupportURL     string
	AccountManager string
	Notes          string
	Contacts       []VendorContactModel `gorm:"foreignKey:VendorID;constraint:OnDelete:CASCADE"`
}

func (VendorModel) TableName() string { return "vendors" }

type VendorContactModel struct {
	BaseModel
	VendorID uuid.UUID `gorm:"type:uuid;index"`
	Name     string
	Email    string
	Phone    string
	Role     string
}

func (VendorContactModel) TableName() string { return "vendor_contacts" }

type ProductModel struct {
	BaseModel
	Name        string
	VendorID    uuid.UUID `gorm:"type:uuid;index"`
	Category    string
	Version     string
	Website     string
	Description string
	Tags        []string `gorm:"type:jsonb;serializer:json"`
}

func (ProductModel) TableName() string { return "products" }

type LicenseModel struct {
	BaseModel
	ProductID             uuid.UUID `gorm:"type:uuid;index"`
	VendorID              uuid.UUID `gorm:"type:uuid;index"`
	LicenseKey            string
	SubscriptionID        string
	ContractNumber        string
	PurchaseOrder         string
	Invoice               string
	PurchaseDate          *time.Time
	RenewalDate           *time.Time
	ExpirationDate        *time.Time
	MaintenanceExpiration *time.Time
	SeatCount             int
	AssignedSeats         int
	Cost                  decimal.Decimal `gorm:"type:numeric(20,2)"`
	Currency              string
	Notes                 string
	LicenseType           string `gorm:"column:license_type"`
}

func (LicenseModel) TableName() string { return "licenses" }

type AssignmentModel struct {
	BaseModel
	LicenseID  uuid.UUID `gorm:"type:uuid;index"`
	TargetType string
	TargetID   string
	TargetName string
	AssignedAt time.Time
	Notes      string
}

func (AssignmentModel) TableName() string { return "assignments" }

type AttachmentModel struct {
	BaseModel
	OwnerType        string
	OwnerID          uuid.UUID `gorm:"type:uuid;index"`
	Filename         string
	ContentType      string
	SizeBytes        int64
	StorageKey       string
	UploadedByUserID *uuid.UUID `gorm:"type:uuid;index"`
	UploadedAt       time.Time
}

func (AttachmentModel) TableName() string { return "attachments" }

type UserModel struct {
	BaseModel
	Email            string `gorm:"uniqueIndex"`
	DisplayName      string
	ExternalSubject  string
	Role             string
	IsServiceAccount bool
	Active           bool
	APIKeys          []APIKeyModel `gorm:"foreignKey:OwnerUserID;constraint:OnDelete:CASCADE"`
}

func (UserModel) TableName() string { return "users" }

type APIKeyModel struct {
	BaseModel
	KeyID       string    `gorm:"uniqueIndex"`
	OwnerUserID uuid.UUID `gorm:"type:uuid;index"`
	Owner       UserModel `gorm:"foreignKey:OwnerUserID;references:ID;constraint:OnDelete:CASCADE"`
	Name        string
	HashedKey   string   `gorm:"uniqueIndex"`
	Scopes      []string `gorm:"type:jsonb;serializer:json"`
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
}

func (APIKeyModel) TableName() string { return "api_keys" }

type AuditLogModel struct {
	AuditBaseModel
	ActorUserID    *uuid.UUID `gorm:"type:uuid;index"`
	ActorAPIKeyID  *uuid.UUID `gorm:"type:uuid;index"`
	Action         string
	EntityType     string
	EntityID       uuid.UUID       `gorm:"type:uuid;index"`
	PreviousValues json.RawMessage `gorm:"type:jsonb"`
	NewValues      json.RawMessage `gorm:"type:jsonb"`
	IPAddress      string
	SessionID      string
}

func (AuditLogModel) TableName() string { return "audit_logs" }

type FeatureFlagModel struct {
	BaseModel
	Key                string `gorm:"uniqueIndex"`
	Description        string
	Enabled            bool
	PercentageRollout  int
	TargetUserIDs      []uuid.UUID `gorm:"type:jsonb;serializer:json"`
	TargetRoles        []string    `gorm:"type:jsonb;serializer:json"`
	ScheduledEnableAt  *time.Time
	ScheduledDisableAt *time.Time
	Audits             []FeatureFlagAuditModel `gorm:"foreignKey:FeatureFlagID;constraint:OnDelete:CASCADE"`
}

func (FeatureFlagModel) TableName() string { return "feature_flags" }

type FeatureFlagAuditModel struct {
	AuditBaseModel
	FeatureFlagID  uuid.UUID  `gorm:"type:uuid;index"`
	ActorUserID    *uuid.UUID `gorm:"type:uuid;index"`
	Action         string
	PreviousValues json.RawMessage `gorm:"type:jsonb"`
	NewValues      json.RawMessage `gorm:"type:jsonb"`
}

func (FeatureFlagAuditModel) TableName() string { return "feature_flag_audits" }

func ensureUUID(id uuid.UUID) uuid.UUID {
	if id == uuid.Nil {
		return uuid.New()
	}
	return id
}

func (m *VendorModel) BeforeCreate(tx *gorm.DB) error        { m.ID = ensureUUID(m.ID); return nil }
func (m *VendorContactModel) BeforeCreate(tx *gorm.DB) error { m.ID = ensureUUID(m.ID); return nil }
func (m *ProductModel) BeforeCreate(tx *gorm.DB) error       { m.ID = ensureUUID(m.ID); return nil }
func (m *LicenseModel) BeforeCreate(tx *gorm.DB) error       { m.ID = ensureUUID(m.ID); return nil }
func (m *AssignmentModel) BeforeCreate(tx *gorm.DB) error    { m.ID = ensureUUID(m.ID); return nil }
func (m *AttachmentModel) BeforeCreate(tx *gorm.DB) error    { m.ID = ensureUUID(m.ID); return nil }
func (m *UserModel) BeforeCreate(tx *gorm.DB) error          { m.ID = ensureUUID(m.ID); return nil }
func (m *APIKeyModel) BeforeCreate(tx *gorm.DB) error        { m.ID = ensureUUID(m.ID); return nil }
func (m *AuditLogModel) BeforeCreate(tx *gorm.DB) error      { m.ID = ensureUUID(m.ID); return nil }
func (m *FeatureFlagModel) BeforeCreate(tx *gorm.DB) error   { m.ID = ensureUUID(m.ID); return nil }
func (m *FeatureFlagAuditModel) BeforeCreate(tx *gorm.DB) error {
	m.ID = ensureUUID(m.ID)
	return nil
}

func newBaseModel(base domain.Base) BaseModel {
	return BaseModel{
		ID:        base.ID,
		CreatedAt: base.CreatedAt,
		UpdatedAt: base.UpdatedAt,
	}
}

func newAuditBaseModel(base domain.Base) AuditBaseModel {
	return AuditBaseModel{
		ID:        base.ID,
		CreatedAt: base.CreatedAt,
		UpdatedAt: base.UpdatedAt,
	}
}
