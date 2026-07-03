package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

type auditRepoFake struct{ logs []domain.AuditLog }

func (r *auditRepoFake) Create(_ context.Context, log *domain.AuditLog) error {
	r.logs = append(r.logs, *log)
	return nil
}
func (r *auditRepoFake) Get(context.Context, uuid.UUID) (*domain.AuditLog, error) {
	return nil, errors.New("not implemented")
}
func (r *auditRepoFake) List(context.Context, domain.ListFilter) ([]domain.AuditLog, error) {
	return nil, errors.New("not implemented")
}

type vendorRepoFake struct{ items map[uuid.UUID]domain.Vendor }

func (r *vendorRepoFake) Create(_ context.Context, v *domain.Vendor) error {
	if r.items == nil {
		r.items = map[uuid.UUID]domain.Vendor{}
	}
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	v.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v.UpdatedAt = v.CreatedAt
	r.items[v.ID] = *v
	return nil
}
func (r *vendorRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Vendor, error) {
	v, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &v, nil
}
func (r *vendorRepoFake) Update(_ context.Context, v *domain.Vendor) error {
	if r.items == nil {
		r.items = map[uuid.UUID]domain.Vendor{}
	}
	v.UpdatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	r.items[v.ID] = *v
	return nil
}
func (r *vendorRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *vendorRepoFake) List(context.Context, domain.ListFilter) ([]domain.Vendor, error) {
	out := make([]domain.Vendor, 0, len(r.items))
	for _, v := range r.items {
		out = append(out, v)
	}
	return out, nil
}

type assignmentRepoFake struct {
	items     map[uuid.UUID]domain.Assignment
	createErr error
}

func (r *assignmentRepoFake) Create(_ context.Context, a *domain.Assignment) error {
	if r.createErr != nil {
		return r.createErr
	}
	if r.items == nil {
		r.items = map[uuid.UUID]domain.Assignment{}
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.UpdatedAt = a.CreatedAt
	r.items[a.ID] = *a
	return nil
}
func (r *assignmentRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Assignment, error) {
	a, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &a, nil
}
func (r *assignmentRepoFake) Update(_ context.Context, a *domain.Assignment) error {
	if r.items == nil {
		r.items = map[uuid.UUID]domain.Assignment{}
	}
	r.items[a.ID] = *a
	return nil
}
func (r *assignmentRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *assignmentRepoFake) List(context.Context, domain.ListFilter) ([]domain.Assignment, error) {
	out := make([]domain.Assignment, 0, len(r.items))
	for _, a := range r.items {
		out = append(out, a)
	}
	return out, nil
}

func TestVendorServiceAuditLifecycle(t *testing.T) {
	t.Parallel()
	ctx := WithRequestContext(context.Background(), RequestContext{IPAddress: "127.0.0.1", SessionID: "req-1"})
	vendorID := uuid.New()
	base := domain.Vendor{Base: domain.Base{ID: vendorID}, Name: "Acme", Contacts: []domain.VendorContact{{Name: "Jane"}}}
	for _, tc := range []struct {
		name string
		run  func(t *testing.T, svc VendorService, audit *auditRepoFake)
	}{
		{name: "create", run: func(t *testing.T, svc VendorService, audit *auditRepoFake) {
			created, err := svc.Create(ctx, domain.Vendor{Name: "Acme", Contacts: []domain.VendorContact{{Name: "Jane"}}})
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if len(audit.logs) != 1 {
				t.Fatalf("expected 1 audit, got %d", len(audit.logs))
			}
			if audit.logs[0].Action != domain.AuditActionCreate {
				t.Fatalf("unexpected action: %s", audit.logs[0].Action)
			}
			if audit.logs[0].EntityID != created.ID {
				t.Fatalf("unexpected entity id")
			}
			if len(audit.logs[0].PreviousValues) != 0 {
				t.Fatalf("expected no previous values")
			}
			if len(audit.logs[0].NewValues) == 0 {
				t.Fatalf("expected new values")
			}
		}},
		{name: "update", run: func(t *testing.T, svc VendorService, audit *auditRepoFake) {
			created, _ := svc.Create(ctx, domain.Vendor{Name: "Acme", Contacts: []domain.VendorContact{{Name: "Jane"}}})
			audit.logs = nil
			updated, err := svc.Update(ctx, created.ID, domain.Vendor{Name: "Acme 2", Contacts: []domain.VendorContact{{Name: "John"}}})
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			if updated.Name != "Acme 2" {
				t.Fatalf("unexpected update")
			}
			if len(audit.logs) != 1 {
				t.Fatalf("expected 1 audit, got %d", len(audit.logs))
			}
			if audit.logs[0].Action != domain.AuditActionUpdate {
				t.Fatalf("unexpected action: %s", audit.logs[0].Action)
			}
			if !json.Valid(audit.logs[0].PreviousValues) || !json.Valid(audit.logs[0].NewValues) {
				t.Fatalf("expected valid json")
			}
		}},
		{name: "delete", run: func(t *testing.T, svc VendorService, audit *auditRepoFake) {
			created, _ := svc.Create(ctx, domain.Vendor{Name: "Acme", Contacts: []domain.VendorContact{{Name: "Jane"}}})
			audit.logs = nil
			if err := svc.Delete(ctx, created.ID); err != nil {
				t.Fatalf("delete: %v", err)
			}
			if len(audit.logs) != 1 {
				t.Fatalf("expected 1 audit, got %d", len(audit.logs))
			}
			if audit.logs[0].Action != domain.AuditActionDelete {
				t.Fatalf("unexpected action: %s", audit.logs[0].Action)
			}
			if len(audit.logs[0].NewValues) != 0 {
				t.Fatalf("expected no new values")
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vendorRepo := &vendorRepoFake{items: map[uuid.UUID]domain.Vendor{vendorID: base}}
			auditRepo := &auditRepoFake{}
			svc := NewVendorService(vendorRepo, auditRepo)
			tc.run(t, svc, auditRepo)
		})
	}
}

func TestAssignmentServicePropagatesConflictAndSkipsAudit(t *testing.T) {
	t.Parallel()
	assignmentRepo := &assignmentRepoFake{createErr: domain.ErrConflict}
	auditRepo := &auditRepoFake{}
	svc := NewAssignmentService(assignmentRepo, auditRepo)
	_, err := svc.Create(context.Background(), domain.Assignment{LicenseID: uuid.New(), TargetType: domain.AssignmentTargetUser, TargetID: "user-1", TargetName: "Jane", AssignedAt: time.Now().UTC()})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if len(auditRepo.logs) != 0 {
		t.Fatalf("expected no audit entries")
	}
}
