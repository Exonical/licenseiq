package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type vendorAPIFake struct{ items map[uuid.UUID]domain.Vendor }

func (s *vendorAPIFake) List(context.Context, domain.ListFilter) ([]domain.Vendor, error) {
	out := make([]domain.Vendor, 0, len(s.items))
	for _, v := range s.items {
		out = append(out, v)
	}
	return out, nil
}
func (s *vendorAPIFake) Get(_ context.Context, id uuid.UUID) (*domain.Vendor, error) {
	v, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &v, nil
}
func (s *vendorAPIFake) Create(_ context.Context, v domain.Vendor) (*domain.Vendor, error) {
	if s.items == nil {
		s.items = map[uuid.UUID]domain.Vendor{}
	}
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	v.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v.UpdatedAt = v.CreatedAt
	s.items[v.ID] = v
	return &v, nil
}
func (s *vendorAPIFake) Update(_ context.Context, id uuid.UUID, v domain.Vendor) (*domain.Vendor, error) {
	if s.items == nil {
		s.items = map[uuid.UUID]domain.Vendor{}
	}
	prev, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	v.Base = prev.Base
	v.ID = id
	v.UpdatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	s.items[id] = v
	return &v, nil
}
func (s *vendorAPIFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(s.items, id)
	return nil
}

type assignmentAPIFake struct{ createErr error }

func (s *assignmentAPIFake) List(context.Context, domain.ListFilter) ([]domain.Assignment, error) {
	return nil, nil
}
func (s *assignmentAPIFake) Get(context.Context, uuid.UUID) (*domain.Assignment, error) {
	return nil, domain.ErrNotFound
}
func (s *assignmentAPIFake) Create(context.Context, domain.Assignment) (*domain.Assignment, error) {
	return nil, s.createErr
}
func (s *assignmentAPIFake) Update(context.Context, uuid.UUID, domain.Assignment) (*domain.Assignment, error) {
	return nil, domain.ErrNotFound
}
func (s *assignmentAPIFake) Delete(context.Context, uuid.UUID) error { return nil }

func TestVendorAPIHappyPath(t *testing.T) {
	t.Parallel()
	vendorSvc := &vendorAPIFake{}
	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{Vendors: vendorSvc}, zap.NewNop(), nil, nil)

	resp := api.Post("/api/v1/vendors", map[string]any{
		"name":     "Acme",
		"contacts": []map[string]any{{"name": "Jane"}},
	})
	if resp.Code != 201 {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	var created VendorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	if created.Name != "Acme" || len(created.Contacts) != 1 {
		t.Fatalf("unexpected create payload: %+v", created)
	}

	resp = api.Get("/api/v1/vendors/" + created.ID.String())
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var got VendorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("unexpected id")
	}

	resp = api.Get("/api/v1/vendors")
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var list Page[VendorResponse]
	if err := json.Unmarshal(resp.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 vendor, got %d", len(list.Data))
	}
}

func TestAPIErrorMappings(t *testing.T) {
	t.Parallel()
	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{
		Vendors:     &vendorAPIFake{items: map[uuid.UUID]domain.Vendor{}},
		Assignments: &assignmentAPIFake{createErr: domain.ErrConflict},
	}, zap.NewNop(), nil, nil)

	resp := api.Get("/api/v1/vendors/550e8400-e29b-41d4-a716-446655440000")
	if resp.Code != 404 {
		t.Fatalf("expected 404, got %d", resp.Code)
	}

	resp = api.Post("/api/v1/assignments", map[string]any{
		"licenseId":  "550e8400-e29b-41d4-a716-446655440000",
		"targetType": "User",
		"targetId":   "user-1",
		"targetName": "Jane",
		"assignedAt": "2026-01-01T00:00:00Z",
	})
	if resp.Code != 409 {
		t.Fatalf("expected 409, got %d", resp.Code)
	}
}
