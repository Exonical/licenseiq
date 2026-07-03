package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Base struct {
	ID        uuid.UUID  `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type ListFilter struct {
	Limit          int
	Offset         int
	IncludeDeleted bool
}

func (f ListFilter) Normalize(defaultLimit int) ListFilter {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = defaultLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f
}

type Role string

const (
	RoleAdministrator  Role = "Administrator"
	RoleLicenseManager Role = "LicenseManager"
	RoleAuditor        Role = "Auditor"
	RoleFinance        Role = "Finance"
	RoleViewer         Role = "Viewer"
)

func (r Role) String() string { return string(r) }

func (r Role) Validate() error {
	switch r {
	case RoleAdministrator, RoleLicenseManager, RoleAuditor, RoleFinance, RoleViewer:
		return nil
	default:
		return fmt.Errorf("%w: invalid role %q", ErrValidation, string(r))
	}
}

func ParseRole(v string) (Role, error) {
	r := Role(strings.TrimSpace(v))
	return r, r.Validate()
}

type LicenseType string

const (
	LicenseTypeSubscription LicenseType = "Subscription"
	LicenseTypePerpetual    LicenseType = "Perpetual"
	LicenseTypePerUser      LicenseType = "PerUser"
	LicenseTypePerDevice    LicenseType = "PerDevice"
	LicenseTypePerCore      LicenseType = "PerCore"
	LicenseTypeConcurrent   LicenseType = "Concurrent"
	LicenseTypeEnterprise   LicenseType = "Enterprise"
)

func (t LicenseType) String() string { return string(t) }

func (t LicenseType) Validate() error {
	switch t {
	case LicenseTypeSubscription, LicenseTypePerpetual, LicenseTypePerUser, LicenseTypePerDevice, LicenseTypePerCore, LicenseTypeConcurrent, LicenseTypeEnterprise:
		return nil
	default:
		return fmt.Errorf("%w: invalid license type %q", ErrValidation, string(t))
	}
}

type AssignmentTargetType string

const (
	AssignmentTargetUser           AssignmentTargetType = "User"
	AssignmentTargetDevice         AssignmentTargetType = "Device"
	AssignmentTargetServer         AssignmentTargetType = "Server"
	AssignmentTargetVirtualMachine AssignmentTargetType = "VirtualMachine"
)

func (t AssignmentTargetType) Validate() error {
	switch t {
	case AssignmentTargetUser, AssignmentTargetDevice, AssignmentTargetServer, AssignmentTargetVirtualMachine:
		return nil
	default:
		return fmt.Errorf("%w: invalid assignment target type %q", ErrValidation, string(t))
	}
}

type AttachmentOwnerType string

const (
	AttachmentOwnerLicense AttachmentOwnerType = "license"
	AttachmentOwnerVendor  AttachmentOwnerType = "vendor"
	AttachmentOwnerProduct AttachmentOwnerType = "product"
)

func (t AttachmentOwnerType) Validate() error {
	switch t {
	case AttachmentOwnerLicense, AttachmentOwnerVendor, AttachmentOwnerProduct:
		return nil
	default:
		return fmt.Errorf("%w: invalid attachment owner type %q", ErrValidation, string(t))
	}
}

type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
)

func (a AuditAction) Validate() error {
	switch a {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete:
		return nil
	default:
		return fmt.Errorf("%w: invalid audit action %q", ErrValidation, string(a))
	}
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
