package persistence

import (
	"encoding/json"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

func vendorToModel(v domain.Vendor) VendorModel {
	contacts := make([]VendorContactModel, 0, len(v.Contacts))
	for _, c := range v.Contacts {
		contacts = append(contacts, vendorContactToModel(v.ID, c))
	}
	return VendorModel{
		BaseModel:      newBaseModel(v.Base),
		Name:           v.Name,
		SupportURL:     v.SupportURL,
		AccountManager: v.AccountManager,
		Notes:          v.Notes,
		Contacts:       contacts,
	}
}

func vendorContactToModel(vendorID uuid.UUID, c domain.VendorContact) VendorContactModel {
	return VendorContactModel{
		BaseModel: BaseModel{ID: uuid.New()},
		VendorID:  vendorID,
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		Role:      c.Role,
	}
}

func vendorToDomain(m VendorModel) domain.Vendor {
	contacts := make([]domain.VendorContact, 0, len(m.Contacts))
	for _, c := range m.Contacts {
		contacts = append(contacts, domain.VendorContact{
			Name:  c.Name,
			Email: c.Email,
			Phone: c.Phone,
			Role:  c.Role,
		})
	}
	return domain.Vendor{
		Base: domain.Base{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		Name:           m.Name,
		SupportURL:     m.SupportURL,
		AccountManager: m.AccountManager,
		Notes:          m.Notes,
		Contacts:       contacts,
	}
}

func productToModel(p domain.Product) ProductModel {
	return ProductModel{
		BaseModel:   newBaseModel(p.Base),
		Name:        p.Name,
		VendorID:    p.VendorID,
		Category:    p.Category,
		Version:     p.Version,
		Website:     p.Website,
		Description: p.Description,
		Tags:        append([]string(nil), p.Tags...),
	}
}

func productToDomain(m ProductModel) domain.Product {
	return domain.Product{
		Base: domain.Base{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		Name:        m.Name,
		VendorID:    m.VendorID,
		Category:    m.Category,
		Version:     m.Version,
		Website:     m.Website,
		Description: m.Description,
		Tags:        append([]string(nil), m.Tags...),
	}
}

func licenseToModel(l domain.License) LicenseModel {
	return LicenseModel{
		BaseModel:             newBaseModel(l.Base),
		ProductID:             l.ProductID,
		VendorID:              l.VendorID,
		LicenseKey:            l.LicenseKey,
		SubscriptionID:        l.SubscriptionID,
		ContractNumber:        l.ContractNumber,
		PurchaseOrder:         l.PurchaseOrder,
		Invoice:               l.Invoice,
		PurchaseDate:          l.PurchaseDate,
		RenewalDate:           l.RenewalDate,
		ExpirationDate:        l.ExpirationDate,
		MaintenanceExpiration: l.MaintenanceExpiration,
		SeatCount:             l.SeatCount,
		AssignedSeats:         l.AssignedSeats,
		Cost:                  l.Cost,
		Currency:              l.Currency,
		Notes:                 l.Notes,
		LicenseType:           l.Type.String(),
	}
}

func licenseToDomain(m LicenseModel) domain.License {
	return domain.License{
		Base: domain.Base{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		ProductID:             m.ProductID,
		VendorID:              m.VendorID,
		LicenseKey:            m.LicenseKey,
		SubscriptionID:        m.SubscriptionID,
		ContractNumber:        m.ContractNumber,
		PurchaseOrder:         m.PurchaseOrder,
		Invoice:               m.Invoice,
		PurchaseDate:          m.PurchaseDate,
		RenewalDate:           m.RenewalDate,
		ExpirationDate:        m.ExpirationDate,
		MaintenanceExpiration: m.MaintenanceExpiration,
		SeatCount:             m.SeatCount,
		AssignedSeats:         m.AssignedSeats,
		Cost:                  m.Cost,
		Currency:              m.Currency,
		Notes:                 m.Notes,
		Type:                  domain.LicenseType(m.LicenseType),
	}
}

func assignmentToModel(a domain.Assignment) AssignmentModel {
	return AssignmentModel{
		BaseModel:  newBaseModel(a.Base),
		LicenseID:  a.LicenseID,
		TargetType: string(a.TargetType),
		TargetID:   a.TargetID,
		TargetName: a.TargetName,
		AssignedAt: a.AssignedAt,
		Notes:      a.Notes,
	}
}

func assignmentToDomain(m AssignmentModel) domain.Assignment {
	return domain.Assignment{
		Base:       domain.Base{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
		LicenseID:  m.LicenseID,
		TargetType: domain.AssignmentTargetType(m.TargetType),
		TargetID:   m.TargetID,
		TargetName: m.TargetName,
		AssignedAt: m.AssignedAt,
		Notes:      m.Notes,
	}
}

func attachmentToModel(a domain.Attachment) AttachmentModel {
	return AttachmentModel{
		BaseModel:        newBaseModel(a.Base),
		OwnerType:        string(a.OwnerType),
		OwnerID:          a.OwnerID,
		Filename:         a.Filename,
		ContentType:      a.ContentType,
		SizeBytes:        a.SizeBytes,
		StorageKey:       a.StorageKey,
		UploadedByUserID: a.UploadedByUserID,
		UploadedAt:       a.UploadedAt,
	}
}

func attachmentToDomain(m AttachmentModel) domain.Attachment {
	return domain.Attachment{
		Base:             domain.Base{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
		OwnerType:        domain.AttachmentOwnerType(m.OwnerType),
		OwnerID:          m.OwnerID,
		Filename:         m.Filename,
		ContentType:      m.ContentType,
		SizeBytes:        m.SizeBytes,
		StorageKey:       m.StorageKey,
		UploadedByUserID: m.UploadedByUserID,
		UploadedAt:       m.UploadedAt,
	}
}

func userToModel(u domain.User) UserModel {
	return UserModel{
		BaseModel:        newBaseModel(u.Base),
		Email:            u.Email,
		DisplayName:      u.DisplayName,
		ExternalSubject:  u.ExternalSubject,
		Role:             u.Role.String(),
		IsServiceAccount: u.IsServiceAccount,
		Active:           u.Active,
	}
}

func userToDomain(m UserModel) domain.User {
	return domain.User{
		Base:             domain.Base{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
		Email:            m.Email,
		DisplayName:      m.DisplayName,
		ExternalSubject:  m.ExternalSubject,
		Role:             domain.Role(m.Role),
		IsServiceAccount: m.IsServiceAccount,
		Active:           m.Active,
	}
}

func apiKeyToModel(k domain.APIKey) APIKeyModel {
	return APIKeyModel{
		BaseModel:   newBaseModel(k.Base),
		KeyID:       k.KeyID,
		OwnerUserID: k.OwnerUserID,
		Name:        k.Name,
		HashedKey:   k.HashedKey,
		Scopes:      append([]string(nil), k.Scopes...),
		ExpiresAt:   k.ExpiresAt,
		LastUsedAt:  k.LastUsedAt,
	}
}

func apiKeyToDomain(m APIKeyModel) domain.APIKey {
	return domain.APIKey{
		Base:        domain.Base{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
		KeyID:       m.KeyID,
		OwnerUserID: m.OwnerUserID,
		Name:        m.Name,
		HashedKey:   m.HashedKey,
		Scopes:      append([]string(nil), m.Scopes...),
		ExpiresAt:   m.ExpiresAt,
		LastUsedAt:  m.LastUsedAt,
	}
}

func auditToModel(a domain.AuditLog) AuditLogModel {
	return AuditLogModel{
		AuditBaseModel: newAuditBaseModel(a.Base),
		ActorUserID:    a.ActorUserID,
		ActorAPIKeyID:  a.ActorAPIKeyID,
		Action:         string(a.Action),
		EntityType:     a.EntityType,
		EntityID:       a.EntityID,
		PreviousValues: append(json.RawMessage(nil), a.PreviousValues...),
		NewValues:      append(json.RawMessage(nil), a.NewValues...),
		IPAddress:      a.IPAddress,
		SessionID:      a.SessionID,
	}
}

func auditToDomain(m AuditLogModel) domain.AuditLog {
	return domain.AuditLog{
		Base:           domain.Base{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
		ActorUserID:    m.ActorUserID,
		ActorAPIKeyID:  m.ActorAPIKeyID,
		Action:         domain.AuditAction(m.Action),
		EntityType:     m.EntityType,
		EntityID:       m.EntityID,
		PreviousValues: append(json.RawMessage(nil), m.PreviousValues...),
		NewValues:      append(json.RawMessage(nil), m.NewValues...),
		IPAddress:      m.IPAddress,
		SessionID:      m.SessionID,
	}
}

func featureFlagToModel(f domain.FeatureFlag) FeatureFlagModel {
	roles := make([]string, 0, len(f.TargetRoles))
	for _, role := range f.TargetRoles {
		roles = append(roles, role.String())
	}
	return FeatureFlagModel{
		BaseModel:          newBaseModel(f.Base),
		Key:                f.Key,
		Description:        f.Description,
		Enabled:            f.Enabled,
		PercentageRollout:  f.PercentageRollout,
		TargetUserIDs:      append([]uuid.UUID(nil), f.TargetUserIDs...),
		TargetRoles:        roles,
		ScheduledEnableAt:  f.ScheduledEnableAt,
		ScheduledDisableAt: f.ScheduledDisableAt,
	}
}

func featureFlagToDomain(m FeatureFlagModel) domain.FeatureFlag {
	roles := make([]domain.Role, 0, len(m.TargetRoles))
	for _, role := range m.TargetRoles {
		roles = append(roles, domain.Role(role))
	}
	return domain.FeatureFlag{
		Base:               domain.Base{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt},
		Key:                m.Key,
		Description:        m.Description,
		Enabled:            m.Enabled,
		PercentageRollout:  m.PercentageRollout,
		TargetUserIDs:      append([]uuid.UUID(nil), m.TargetUserIDs...),
		TargetRoles:        roles,
		ScheduledEnableAt:  m.ScheduledEnableAt,
		ScheduledDisableAt: m.ScheduledDisableAt,
	}
}

func featureFlagAuditToModel(flagID uuid.UUID, actorID *uuid.UUID, action domain.AuditAction, prev, next []byte) FeatureFlagAuditModel {
	return FeatureFlagAuditModel{
		AuditBaseModel: newAuditBaseModel(domain.Base{}),
		FeatureFlagID:  flagID,
		ActorUserID:    actorID,
		Action:         string(action),
		PreviousValues: append(json.RawMessage(nil), prev...),
		NewValues:      append(json.RawMessage(nil), next...),
	}
}
