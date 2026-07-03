package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultLimit = 100

type VendorRepository struct{ db *gorm.DB }
type ProductRepository struct{ db *gorm.DB }
type LicenseRepository struct{ db *gorm.DB }
type AssignmentRepository struct{ db *gorm.DB }
type AttachmentRepository struct{ db *gorm.DB }
type UserRepository struct{ db *gorm.DB }
type APIKeyRepository struct{ db *gorm.DB }
type AuditRepository struct{ db *gorm.DB }
type FeatureFlagRepository struct{ db *gorm.DB }

func NewVendorRepository(db *gorm.DB) *VendorRepository         { return &VendorRepository{db: db} }
func NewProductRepository(db *gorm.DB) *ProductRepository       { return &ProductRepository{db: db} }
func NewLicenseRepository(db *gorm.DB) *LicenseRepository       { return &LicenseRepository{db: db} }
func NewAssignmentRepository(db *gorm.DB) *AssignmentRepository { return &AssignmentRepository{db: db} }
func NewAttachmentRepository(db *gorm.DB) *AttachmentRepository { return &AttachmentRepository{db: db} }
func NewUserRepository(db *gorm.DB) *UserRepository             { return &UserRepository{db: db} }
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository         { return &APIKeyRepository{db: db} }
func NewAuditRepository(db *gorm.DB) *AuditRepository           { return &AuditRepository{db: db} }
func NewFeatureFlagRepository(db *gorm.DB) *FeatureFlagRepository {
	return &FeatureFlagRepository{db: db}
}

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return domain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return domain.ErrConflict
	default:
		return err
	}
}

func normalize(filter domain.ListFilter) domain.ListFilter {
	return filter.Normalize(defaultLimit)
}

func baseQuery(db *gorm.DB, filter domain.ListFilter) *gorm.DB {
	filter = normalize(filter)
	q := db.Offset(filter.Offset).Limit(filter.Limit)
	if filter.IncludeDeleted {
		q = q.Unscoped()
	}
	return q
}

func marshalJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// Vendor
func (r *VendorRepository) Create(ctx context.Context, entity *domain.Vendor) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := vendorToModel(*entity)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Contacts").Create(&model).Error; err != nil {
			return mapError(err)
		}
		for i := range model.Contacts {
			model.Contacts[i].VendorID = model.ID
			if err := tx.Create(&model.Contacts[i]).Error; err != nil {
				return mapError(err)
			}
		}
		*entity = vendorToDomain(model)
		return nil
	})
}

func (r *VendorRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Vendor, error) {
	var model VendorModel
	err := r.db.WithContext(ctx).Preload("Contacts").First(&model, "id = ?", id).Error
	if err != nil {
		return nil, mapError(err)
	}
	entity := vendorToDomain(model)
	return &entity, nil
}

func (r *VendorRepository) Update(ctx context.Context, entity *domain.Vendor) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := vendorToModel(*entity)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Contacts").Save(&model).Error; err != nil {
			return mapError(err)
		}
		if err := tx.Where("vendor_id = ?", model.ID).Delete(&VendorContactModel{}).Error; err != nil {
			return mapError(err)
		}
		for i := range model.Contacts {
			model.Contacts[i].VendorID = model.ID
			if err := tx.Create(&model.Contacts[i]).Error; err != nil {
				return mapError(err)
			}
		}
		*entity = vendorToDomain(model)
		return nil
	})
}

func (r *VendorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(r.db.WithContext(ctx).Delete(&VendorModel{}, "id = ?", id).Error)
}

func (r *VendorRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Vendor, error) {
	var models []VendorModel
	q := baseQuery(r.db.WithContext(ctx), filter).Preload("Contacts").Order("created_at DESC")
	if err := q.Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.Vendor, 0, len(models))
	for _, m := range models {
		out = append(out, vendorToDomain(m))
	}
	return out, nil
}

// Product
func (r *ProductRepository) Create(ctx context.Context, entity *domain.Product) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := productToModel(*entity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = productToDomain(model)
	return nil
}

func (r *ProductRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var model ProductModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := productToDomain(model)
	return &entity, nil
}

func (r *ProductRepository) Update(ctx context.Context, entity *domain.Product) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := productToModel(*entity)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = productToDomain(model)
	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(r.db.WithContext(ctx).Delete(&ProductModel{}, "id = ?", id).Error)
}

func (r *ProductRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Product, error) {
	var models []ProductModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.Product, 0, len(models))
	for _, m := range models {
		out = append(out, productToDomain(m))
	}
	return out, nil
}

// License
func (r *LicenseRepository) Create(ctx context.Context, entity *domain.License) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := licenseToModel(*entity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = licenseToDomain(model)
	return nil
}

func (r *LicenseRepository) Get(ctx context.Context, id uuid.UUID) (*domain.License, error) {
	var model LicenseModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := licenseToDomain(model)
	return &entity, nil
}

func (r *LicenseRepository) Update(ctx context.Context, entity *domain.License) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := licenseToModel(*entity)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = licenseToDomain(model)
	return nil
}

func (r *LicenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(r.db.WithContext(ctx).Delete(&LicenseModel{}, "id = ?", id).Error)
}

func (r *LicenseRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.License, error) {
	var models []LicenseModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.License, 0, len(models))
	for _, m := range models {
		out = append(out, licenseToDomain(m))
	}
	return out, nil
}

// Assignment
func (r *AssignmentRepository) Create(ctx context.Context, entity *domain.Assignment) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var license LicenseModel
		if err := tx.First(&license, "id = ?", entity.LicenseID).Error; err != nil {
			return mapError(err)
		}
		var count int64
		if err := tx.Model(&AssignmentModel{}).Where("license_id = ?", entity.LicenseID).Count(&count).Error; err != nil {
			return mapError(err)
		}
		if license.SeatCount > 0 && int(count) >= license.SeatCount {
			return domain.ErrConflict
		}
		model := assignmentToModel(*entity)
		if err := tx.Create(&model).Error; err != nil {
			return mapError(err)
		}
		license.AssignedSeats = int(count) + 1
		if err := tx.Model(&LicenseModel{}).Where("id = ?", license.ID).Update("assigned_seats", license.AssignedSeats).Error; err != nil {
			return mapError(err)
		}
		*entity = assignmentToDomain(model)
		return nil
	})
}

func (r *AssignmentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Assignment, error) {
	var model AssignmentModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := assignmentToDomain(model)
	return &entity, nil
}

func (r *AssignmentRepository) Update(ctx context.Context, entity *domain.Assignment) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := assignmentToModel(*entity)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = assignmentToDomain(model)
	return nil
}

func (r *AssignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assignment AssignmentModel
		if err := tx.First(&assignment, "id = ?", id).Error; err != nil {
			return mapError(err)
		}
		if err := tx.Delete(&assignment).Error; err != nil {
			return mapError(err)
		}
		var count int64
		if err := tx.Model(&AssignmentModel{}).Where("license_id = ?", assignment.LicenseID).Count(&count).Error; err != nil {
			return mapError(err)
		}
		if err := tx.Model(&LicenseModel{}).Where("id = ?", assignment.LicenseID).Update("assigned_seats", int(count)).Error; err != nil {
			return mapError(err)
		}
		return nil
	})
}

func (r *AssignmentRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Assignment, error) {
	var models []AssignmentModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.Assignment, 0, len(models))
	for _, m := range models {
		out = append(out, assignmentToDomain(m))
	}
	return out, nil
}

// Attachment
func (r *AttachmentRepository) Create(ctx context.Context, entity *domain.Attachment) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := attachmentToModel(*entity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = attachmentToDomain(model)
	return nil
}

func (r *AttachmentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	var model AttachmentModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := attachmentToDomain(model)
	return &entity, nil
}

func (r *AttachmentRepository) Update(ctx context.Context, entity *domain.Attachment) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := attachmentToModel(*entity)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = attachmentToDomain(model)
	return nil
}

func (r *AttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(r.db.WithContext(ctx).Delete(&AttachmentModel{}, "id = ?", id).Error)
}

func (r *AttachmentRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Attachment, error) {
	var models []AttachmentModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.Attachment, 0, len(models))
	for _, m := range models {
		out = append(out, attachmentToDomain(m))
	}
	return out, nil
}

// User
func (r *UserRepository) Create(ctx context.Context, entity *domain.User) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := userToModel(*entity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = userToDomain(model)
	return nil
}

func (r *UserRepository) Get(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := userToDomain(model)
	return &entity, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).First(&model, "email = ?", email).Error; err != nil {
		return nil, mapError(err)
	}
	entity := userToDomain(model)
	return &entity, nil
}

func (r *UserRepository) GetByExternalSubject(ctx context.Context, subject string) (*domain.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).First(&model, "external_subject = ?", subject).Error; err != nil {
		return nil, mapError(err)
	}
	entity := userToDomain(model)
	return &entity, nil
}

func (r *UserRepository) Update(ctx context.Context, entity *domain.User) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := userToModel(*entity)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = userToDomain(model)
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(r.db.WithContext(ctx).Delete(&UserModel{}, "id = ?", id).Error)
}

func (r *UserRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.User, error) {
	var models []UserModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.User, 0, len(models))
	for _, m := range models {
		out = append(out, userToDomain(m))
	}
	return out, nil
}

// API keys
func (r *APIKeyRepository) Create(ctx context.Context, entity *domain.APIKey) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := apiKeyToModel(*entity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = apiKeyToDomain(model)
	return nil
}

func (r *APIKeyRepository) Get(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	var model APIKeyModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := apiKeyToDomain(model)
	return &entity, nil
}

func (r *APIKeyRepository) GetByKeyID(ctx context.Context, keyID string) (*domain.APIKey, error) {
	var model APIKeyModel
	if err := r.db.WithContext(ctx).First(&model, "key_id = ?", keyID).Error; err != nil {
		return nil, mapError(err)
	}
	entity := apiKeyToDomain(model)
	return &entity, nil
}

func (r *APIKeyRepository) Update(ctx context.Context, entity *domain.APIKey) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := apiKeyToModel(*entity)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = apiKeyToDomain(model)
	return nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(r.db.WithContext(ctx).Delete(&APIKeyModel{}, "id = ?", id).Error)
}

func (r *APIKeyRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.APIKey, error) {
	var models []APIKeyModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.APIKey, 0, len(models))
	for _, m := range models {
		out = append(out, apiKeyToDomain(m))
	}
	return out, nil
}

// Audit
func (r *AuditRepository) Create(ctx context.Context, entity *domain.AuditLog) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := auditToModel(*entity)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return mapError(err)
	}
	*entity = auditToDomain(model)
	return nil
}

func (r *AuditRepository) Get(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	var model AuditLogModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := auditToDomain(model)
	return &entity, nil
}

func (r *AuditRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.AuditLog, error) {
	var models []AuditLogModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.AuditLog, 0, len(models))
	for _, m := range models {
		out = append(out, auditToDomain(m))
	}
	return out, nil
}

// Feature flags
func (r *FeatureFlagRepository) Create(ctx context.Context, entity *domain.FeatureFlag) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	model := featureFlagToModel(*entity)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return mapError(err)
		}
		audit := featureFlagAuditToModel(model.ID, nil, domain.AuditActionCreate, nil, marshalJSON(model))
		if err := tx.Create(&audit).Error; err != nil {
			return mapError(err)
		}
		*entity = featureFlagToDomain(model)
		return nil
	})
}

func (r *FeatureFlagRepository) Get(ctx context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	var model FeatureFlagModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		return nil, mapError(err)
	}
	entity := featureFlagToDomain(model)
	return &entity, nil
}

func (r *FeatureFlagRepository) Update(ctx context.Context, entity *domain.FeatureFlag) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing FeatureFlagModel
		if err := tx.First(&existing, "id = ?", entity.ID).Error; err != nil {
			return mapError(err)
		}
		model := featureFlagToModel(*entity)
		if err := tx.Save(&model).Error; err != nil {
			return mapError(err)
		}
		prev := marshalJSON(existing)
		next := marshalJSON(model)
		audit := featureFlagAuditToModel(model.ID, nil, domain.AuditActionUpdate, prev, next)
		if err := tx.Create(&audit).Error; err != nil {
			return mapError(err)
		}
		*entity = featureFlagToDomain(model)
		return nil
	})
}

func (r *FeatureFlagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing FeatureFlagModel
		if err := tx.First(&existing, "id = ?", id).Error; err != nil {
			return mapError(err)
		}
		if err := tx.Delete(&existing).Error; err != nil {
			return mapError(err)
		}
		audit := featureFlagAuditToModel(existing.ID, nil, domain.AuditActionDelete, marshalJSON(existing), nil)
		if err := tx.Create(&audit).Error; err != nil {
			return mapError(err)
		}
		return nil
	})
}

func (r *FeatureFlagRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.FeatureFlag, error) {
	var models []FeatureFlagModel
	if err := baseQuery(r.db.WithContext(ctx), filter).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.FeatureFlag, 0, len(models))
	for _, m := range models {
		out = append(out, featureFlagToDomain(m))
	}
	return out, nil
}
