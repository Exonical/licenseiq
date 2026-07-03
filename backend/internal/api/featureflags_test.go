package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type featureFlagAPIFake struct {
	items map[uuid.UUID]domain.FeatureFlag
}

func (s *featureFlagAPIFake) List(context.Context, domain.ListFilter) ([]domain.FeatureFlag, error) {
	out := make([]domain.FeatureFlag, 0, len(s.items))
	for _, flag := range s.items {
		out = append(out, flag)
	}
	return out, nil
}

func (s *featureFlagAPIFake) Get(_ context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	flag, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &flag, nil
}

func (s *featureFlagAPIFake) Create(_ context.Context, flag domain.FeatureFlag) (*domain.FeatureFlag, error) {
	if s.items == nil {
		s.items = map[uuid.UUID]domain.FeatureFlag{}
	}
	if flag.ID == uuid.Nil {
		flag.ID = uuid.New()
	}
	flag.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	flag.UpdatedAt = flag.CreatedAt
	s.items[flag.ID] = flag
	return &flag, nil
}

func (s *featureFlagAPIFake) Update(_ context.Context, id uuid.UUID, flag domain.FeatureFlag) (*domain.FeatureFlag, error) {
	prev, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	flag.Base = prev.Base
	flag.ID = id
	flag.UpdatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	s.items[id] = flag
	return &flag, nil
}

func (s *featureFlagAPIFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(s.items, id)
	return nil
}

type featureFlagRepoAdapter struct {
	items map[uuid.UUID]domain.FeatureFlag
}

func (r *featureFlagRepoAdapter) Create(_ context.Context, flag *domain.FeatureFlag) error {
	if r.items == nil {
		r.items = map[uuid.UUID]domain.FeatureFlag{}
	}
	if flag.ID == uuid.Nil {
		flag.ID = uuid.New()
	}
	flag.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	flag.UpdatedAt = flag.CreatedAt
	r.items[flag.ID] = *flag
	return nil
}

func (r *featureFlagRepoAdapter) Get(_ context.Context, id uuid.UUID) (*domain.FeatureFlag, error) {
	flag, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &flag, nil
}

func (r *featureFlagRepoAdapter) Update(_ context.Context, flag *domain.FeatureFlag) error {
	prev, ok := r.items[flag.ID]
	if !ok {
		return domain.ErrNotFound
	}
	flag.Base = prev.Base
	flag.UpdatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	r.items[flag.ID] = *flag
	return nil
}

func (r *featureFlagRepoAdapter) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}

func (r *featureFlagRepoAdapter) List(_ context.Context, _ domain.ListFilter) ([]domain.FeatureFlag, error) {
	out := make([]domain.FeatureFlag, 0, len(r.items))
	for _, flag := range r.items {
		out = append(out, flag)
	}
	return out, nil
}

type authUserRepo struct {
	items map[uuid.UUID]domain.User
}

func (r *authUserRepo) Create(context.Context, *domain.User) error { return nil }
func (r *authUserRepo) Get(_ context.Context, id uuid.UUID) (*domain.User, error) {
	user, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &user, nil
}
func (r *authUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *authUserRepo) GetByExternalSubject(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *authUserRepo) Update(context.Context, *domain.User) error { return nil }
func (r *authUserRepo) Delete(context.Context, uuid.UUID) error    { return nil }
func (r *authUserRepo) List(context.Context, domain.ListFilter) ([]domain.User, error) {
	return nil, nil
}

type authKeyRepo struct {
	itemsByKeyID map[string]domain.APIKey
}

func (r *authKeyRepo) Create(context.Context, *domain.APIKey) error { return nil }
func (r *authKeyRepo) Get(context.Context, uuid.UUID) (*domain.APIKey, error) {
	return nil, domain.ErrNotFound
}
func (r *authKeyRepo) GetByKeyID(_ context.Context, keyID string) (*domain.APIKey, error) {
	key, ok := r.itemsByKeyID[keyID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &key, nil
}
func (r *authKeyRepo) Update(context.Context, *domain.APIKey) error { return nil }
func (r *authKeyRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (r *authKeyRepo) List(context.Context, domain.ListFilter) ([]domain.APIKey, error) {
	return nil, nil
}

func newAuthManager(t *testing.T, adminUser, viewerUser domain.User, adminToken, viewerToken string) *auth.Manager {
	t.Helper()
	userRepo := &authUserRepo{items: map[uuid.UUID]domain.User{adminUser.ID: adminUser, viewerUser.ID: viewerUser}}
	keyRepo := &authKeyRepo{itemsByKeyID: map[string]domain.APIKey{
		"admin":  {Base: domain.Base{ID: uuid.New()}, OwnerUserID: adminUser.ID, KeyID: "admin", HashedKey: sha256Hex(adminToken), Name: "admin", Active: true},
		"viewer": {Base: domain.Base{ID: uuid.New()}, OwnerUserID: viewerUser.ID, KeyID: "viewer", HashedKey: sha256Hex(viewerToken), Name: "viewer", Active: true},
	}}
	identity := app.NewIdentityService(userRepo, keyRepo, nil)
	mgr, err := auth.NewManager(context.Background(), config.AuthConfig{}, identity, userRepo, keyRepo, zap.NewNop())
	if err != nil {
		t.Fatalf("new auth manager: %v", err)
	}
	return mgr
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TestFeatureFlagAPIEvaluateAndAuthorization(t *testing.T) {
	store := map[uuid.UUID]domain.FeatureFlag{}
	service := &featureFlagAPIFake{items: store}
	repo := &featureFlagRepoAdapter{items: store}
	flag := domain.FeatureFlag{Key: "new-dashboard", Enabled: true, PercentageRollout: 100}
	if err := repo.Create(context.Background(), &flag); err != nil {
		t.Fatalf("seed flag: %v", err)
	}

	mgr, err := featureflags.NewManager(context.Background(), config.FeatureFlagsConfig{}, repo, zap.NewNop())
	if err != nil {
		t.Fatalf("new feature manager: %v", err)
	}
	t.Cleanup(mgr.Close)

	adminUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "admin@example.com", DisplayName: "Admin", Role: domain.RoleAdministrator, IsServiceAccount: true, Active: true}
	viewerUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "viewer@example.com", DisplayName: "Viewer", Role: domain.RoleViewer, IsServiceAccount: true, Active: true}
	adminToken := "liq_admin.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	viewerToken := "liq_viewer.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	authMgr := newAuthManager(t, adminUser, viewerUser, adminToken, viewerToken)

	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{FeatureFlags: service}, zap.NewNop(), authMgr, mgr)

	resp := api.Get("/api/v1/feature-flags/new-dashboard/evaluate", "Authorization: Bearer "+viewerToken)
	if resp.Code != 200 {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var evaluation FeatureFlagEvaluationResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &evaluation); err != nil {
		t.Fatalf("unmarshal evaluate: %v", err)
	}
	if !evaluation.Enabled {
		t.Fatalf("expected evaluation to be enabled")
	}

	resp = api.Post("/api/v1/feature-flags", "Authorization: Bearer "+viewerToken, map[string]any{
		"key":               "beta-ui",
		"description":       "Beta UI",
		"enabled":           true,
		"percentageRollout": 100,
	})
	if resp.Code != 403 {
		t.Fatalf("expected viewer create to be forbidden, got %d", resp.Code)
	}

	resp = api.Post("/api/v1/feature-flags", "Authorization: Bearer "+adminToken, map[string]any{
		"key":               "beta-ui",
		"description":       "Beta UI",
		"enabled":           true,
		"percentageRollout": 100,
	})
	if resp.Code != 201 {
		t.Fatalf("expected admin create to succeed, got %d", resp.Code)
	}

	resp = api.Get("/api/v1/feature-flags")
	if resp.Code != 401 {
		t.Fatalf("expected unauthenticated list to be rejected, got %d", resp.Code)
	}

}
