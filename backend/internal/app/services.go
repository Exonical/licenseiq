package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

type VendorService interface {
	List(context.Context, domain.ListFilter) ([]domain.Vendor, error)
	Get(context.Context, uuid.UUID) (*domain.Vendor, error)
	Create(context.Context, domain.Vendor) (*domain.Vendor, error)
	Update(context.Context, uuid.UUID, domain.Vendor) (*domain.Vendor, error)
	Delete(context.Context, uuid.UUID) error
}

type ProductService interface {
	List(context.Context, domain.ListFilter) ([]domain.Product, error)
	Get(context.Context, uuid.UUID) (*domain.Product, error)
	Create(context.Context, domain.Product) (*domain.Product, error)
	Update(context.Context, uuid.UUID, domain.Product) (*domain.Product, error)
	Delete(context.Context, uuid.UUID) error
}

type LicenseService interface {
	List(context.Context, domain.ListFilter) ([]domain.License, error)
	Get(context.Context, uuid.UUID) (*domain.License, error)
	Create(context.Context, domain.License) (*domain.License, error)
	Update(context.Context, uuid.UUID, domain.License) (*domain.License, error)
	Delete(context.Context, uuid.UUID) error
}

type AssignmentService interface {
	List(context.Context, domain.ListFilter) ([]domain.Assignment, error)
	Get(context.Context, uuid.UUID) (*domain.Assignment, error)
	Create(context.Context, domain.Assignment) (*domain.Assignment, error)
	Update(context.Context, uuid.UUID, domain.Assignment) (*domain.Assignment, error)
	Delete(context.Context, uuid.UUID) error
}

type AttachmentService interface {
	List(context.Context, domain.ListFilter) ([]domain.Attachment, error)
	Get(context.Context, uuid.UUID) (*domain.Attachment, error)
	Create(context.Context, domain.Attachment) (*domain.Attachment, error)
	Delete(context.Context, uuid.UUID) error
}

type vendorService struct {
	repo   domain.VendorRepository
	audits domain.AuditRepository
}

type productService struct {
	repo   domain.ProductRepository
	audits domain.AuditRepository
}

type licenseService struct {
	repo   domain.LicenseRepository
	audits domain.AuditRepository
}

type assignmentService struct {
	repo   domain.AssignmentRepository
	audits domain.AuditRepository
}

type attachmentService struct {
	repo   domain.AttachmentRepository
	audits domain.AuditRepository
}

func NewVendorService(repo domain.VendorRepository, audits domain.AuditRepository) VendorService {
	return &vendorService{repo: repo, audits: audits}
}

func NewProductService(repo domain.ProductRepository, audits domain.AuditRepository) ProductService {
	return &productService{repo: repo, audits: audits}
}

func NewLicenseService(repo domain.LicenseRepository, audits domain.AuditRepository) LicenseService {
	return &licenseService{repo: repo, audits: audits}
}

func NewAssignmentService(repo domain.AssignmentRepository, audits domain.AuditRepository) AssignmentService {
	return &assignmentService{repo: repo, audits: audits}
}

func NewAttachmentService(repo domain.AttachmentRepository, audits domain.AuditRepository) AttachmentService {
	return &attachmentService{repo: repo, audits: audits}
}

func (s *vendorService) List(ctx context.Context, filter domain.ListFilter) ([]domain.Vendor, error) {
	return s.repo.List(ctx, filter)
}
func (s *vendorService) Get(ctx context.Context, id uuid.UUID) (*domain.Vendor, error) {
	return s.repo.Get(ctx, id)
}
func (s *productService) List(ctx context.Context, filter domain.ListFilter) ([]domain.Product, error) {
	return s.repo.List(ctx, filter)
}
func (s *productService) Get(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return s.repo.Get(ctx, id)
}
func (s *licenseService) List(ctx context.Context, filter domain.ListFilter) ([]domain.License, error) {
	return s.repo.List(ctx, filter)
}
func (s *licenseService) Get(ctx context.Context, id uuid.UUID) (*domain.License, error) {
	return s.repo.Get(ctx, id)
}
func (s *assignmentService) List(ctx context.Context, filter domain.ListFilter) ([]domain.Assignment, error) {
	return s.repo.List(ctx, filter)
}
func (s *assignmentService) Get(ctx context.Context, id uuid.UUID) (*domain.Assignment, error) {
	return s.repo.Get(ctx, id)
}
func (s *attachmentService) List(ctx context.Context, filter domain.ListFilter) ([]domain.Attachment, error) {
	return s.repo.List(ctx, filter)
}
func (s *attachmentService) Get(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	return s.repo.Get(ctx, id)
}

func (s *vendorService) Create(ctx context.Context, input domain.Vendor) (*domain.Vendor, error) {
	created := input
	if err := s.repo.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "vendor", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *vendorService) Update(ctx context.Context, id uuid.UUID, input domain.Vendor) (*domain.Vendor, error) {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	input.Base = previous.Base
	input.ID = id
	updated := input
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "vendor", updated.ID, previous, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *vendorService) Delete(ctx context.Context, id uuid.UUID) error {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "vendor", id, previous, nil)
}

func (s *productService) Create(ctx context.Context, input domain.Product) (*domain.Product, error) {
	created := input
	if err := s.repo.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "product", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *productService) Update(ctx context.Context, id uuid.UUID, input domain.Product) (*domain.Product, error) {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	input.Base = previous.Base
	input.ID = id
	updated := input
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "product", updated.ID, previous, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *productService) Delete(ctx context.Context, id uuid.UUID) error {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "product", id, previous, nil)
}

func (s *licenseService) Create(ctx context.Context, input domain.License) (*domain.License, error) {
	created := input
	if err := s.repo.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "license", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *licenseService) Update(ctx context.Context, id uuid.UUID, input domain.License) (*domain.License, error) {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	input.Base = previous.Base
	input.ID = id
	updated := input
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "license", updated.ID, previous, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *licenseService) Delete(ctx context.Context, id uuid.UUID) error {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "license", id, previous, nil)
}

func (s *assignmentService) Create(ctx context.Context, input domain.Assignment) (*domain.Assignment, error) {
	created := input
	if err := s.repo.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "assignment", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *assignmentService) Update(ctx context.Context, id uuid.UUID, input domain.Assignment) (*domain.Assignment, error) {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	input.Base = previous.Base
	input.ID = id
	updated := input
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "assignment", updated.ID, previous, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *assignmentService) Delete(ctx context.Context, id uuid.UUID) error {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "assignment", id, previous, nil)
}

func (s *attachmentService) Create(ctx context.Context, input domain.Attachment) (*domain.Attachment, error) {
	created := input
	if err := s.repo.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "attachment", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *attachmentService) Delete(ctx context.Context, id uuid.UUID) error {
	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "attachment", id, previous, nil)
}

func writeAudit(ctx context.Context, repo domain.AuditRepository, action domain.AuditAction, entityType string, entityID uuid.UUID, previous any, current any) error {
	if repo == nil {
		return errors.New("audit repository not configured")
	}
	rc := RequestContextFromContext(ctx)
	audit := domain.AuditLog{
		ActorUserID:    rc.ActorUserID,
		ActorAPIKeyID:  rc.ActorAPIKeyID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		PreviousValues: rawJSON(previous),
		NewValues:      rawJSON(current),
		IPAddress:      rc.IPAddress,
		SessionID:      rc.SessionID,
	}
	if err := repo.Create(ctx, &audit); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func rawJSON(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return b
}
