package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VendorRepository interface {
	Create(context.Context, *Vendor) error
	Get(context.Context, uuid.UUID) (*Vendor, error)
	Update(context.Context, *Vendor) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]Vendor, error)
}

type ProductRepository interface {
	Create(context.Context, *Product) error
	Get(context.Context, uuid.UUID) (*Product, error)
	Update(context.Context, *Product) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]Product, error)
}

type LicenseRepository interface {
	Create(context.Context, *License) error
	Get(context.Context, uuid.UUID) (*License, error)
	Update(context.Context, *License) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]License, error)
}

type AssignmentRepository interface {
	Create(context.Context, *Assignment) error
	Get(context.Context, uuid.UUID) (*Assignment, error)
	Update(context.Context, *Assignment) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]Assignment, error)
}

type AttachmentRepository interface {
	Create(context.Context, *Attachment) error
	Get(context.Context, uuid.UUID) (*Attachment, error)
	Update(context.Context, *Attachment) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]Attachment, error)
}

type UserRepository interface {
	Create(context.Context, *User) error
	Get(context.Context, uuid.UUID) (*User, error)
	GetByEmail(context.Context, string) (*User, error)
	GetByExternalSubject(context.Context, string) (*User, error)
	Update(context.Context, *User) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]User, error)
}

type APIKeyRepository interface {
	Create(context.Context, *APIKey) error
	Get(context.Context, uuid.UUID) (*APIKey, error)
	GetByKeyID(context.Context, string) (*APIKey, error)
	Update(context.Context, *APIKey) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]APIKey, error)
}

type RenewalReminderLogRepository interface {
	Create(context.Context, *RenewalReminderLog) error
	Get(context.Context, uuid.UUID) (*RenewalReminderLog, error)
	GetByLicenseThresholdAndRenewalDate(context.Context, uuid.UUID, int, time.Time) (*RenewalReminderLog, error)
	List(context.Context, ListFilter) ([]RenewalReminderLog, error)
}

type AuditRepository interface {
	Create(context.Context, *AuditLog) error
	Get(context.Context, uuid.UUID) (*AuditLog, error)
	List(context.Context, ListFilter) ([]AuditLog, error)
}

type FeatureFlagRepository interface {
	Create(context.Context, *FeatureFlag) error
	Get(context.Context, uuid.UUID) (*FeatureFlag, error)
	Update(context.Context, *FeatureFlag) error
	Delete(context.Context, uuid.UUID) error
	List(context.Context, ListFilter) ([]FeatureFlag, error)
}
