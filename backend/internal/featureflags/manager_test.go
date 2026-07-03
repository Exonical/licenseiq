package featureflags

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeFeatureFlagRepo struct {
	items []domain.FeatureFlag
}

func (r *fakeFeatureFlagRepo) Create(context.Context, *domain.FeatureFlag) error { return nil }
func (r *fakeFeatureFlagRepo) Get(context.Context, uuid.UUID) (*domain.FeatureFlag, error) {
	return nil, domain.ErrNotFound
}
func (r *fakeFeatureFlagRepo) Update(context.Context, *domain.FeatureFlag) error { return nil }
func (r *fakeFeatureFlagRepo) Delete(context.Context, uuid.UUID) error           { return nil }
func (r *fakeFeatureFlagRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.FeatureFlag, error) {
	out := make([]domain.FeatureFlag, len(r.items))
	copy(out, r.items)
	return out, nil
}

func TestToGOFFFlagMapping(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("disabled flag skips targeting", func(t *testing.T) {
		flag := domain.FeatureFlag{Key: "new-dashboard", Enabled: false, PercentageRollout: 100, TargetUserIDs: []uuid.UUID{userID}, TargetRoles: []domain.Role{domain.RoleViewer}}
		mapped := toGOFFFlag(flag, now)
		if mapped.DefaultRule.Variation != "disabled" {
			t.Fatalf("expected disabled default, got %+v", mapped.DefaultRule)
		}
		if len(mapped.Targeting) != 0 {
			t.Fatalf("expected disabled flag to skip targeting, got %d", len(mapped.Targeting))
		}
	})

	t.Run("enabled flag emits targeting and rollout", func(t *testing.T) {
		flag := domain.FeatureFlag{Key: "new-dashboard", Enabled: true, PercentageRollout: 0, TargetUserIDs: []uuid.UUID{userID}, TargetRoles: []domain.Role{domain.RoleViewer}}
		mapped := toGOFFFlag(flag, now)
		if mapped.DefaultRule.Variation != "disabled" {
			t.Fatalf("expected 0%% rollout to disable default, got %+v", mapped.DefaultRule)
		}
		if len(mapped.Targeting) != 2 {
			t.Fatalf("expected explicit targeting rules, got %d", len(mapped.Targeting))
		}
		if mapped.Targeting[0].Variation != "enabled" || !strings.Contains(mapped.Targeting[0].Query, userID.String()) {
			t.Fatalf("unexpected user targeting: %+v", mapped.Targeting[0])
		}

		flag.PercentageRollout = 100
		mapped = toGOFFFlag(flag, now)
		if mapped.DefaultRule.Variation != "enabled" {
			t.Fatalf("expected 100%% rollout to enable default, got %+v", mapped.DefaultRule)
		}
	})

	t.Run("future schedule disables the flag", func(t *testing.T) {
		future := now.Add(time.Hour)
		flag := domain.FeatureFlag{Key: "scheduled", Enabled: true, PercentageRollout: 100, ScheduledEnableAt: &future, TargetUserIDs: []uuid.UUID{userID}}
		mapped := toGOFFFlag(flag, now)
		if mapped.DefaultRule.Variation != "disabled" {
			t.Fatalf("expected future schedule to disable default, got %+v", mapped.DefaultRule)
		}
		if len(mapped.Targeting) != 0 {
			t.Fatalf("expected future schedule to suppress targeting, got %d", len(mapped.Targeting))
		}
	})
}

func TestRepositoryRetrieverRetrieve(t *testing.T) {
	repo := &fakeFeatureFlagRepo{items: []domain.FeatureFlag{{Key: "new-dashboard", Enabled: true, PercentageRollout: 100}}}
	data, err := (&RepositoryRetriever{repo: repo}).Retrieve(context.Background())
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !strings.Contains(string(data), "new-dashboard") {
		t.Fatalf("expected serialized flag data, got %s", string(data))
	}
}

func TestManagerEvaluatePrecedenceAndOverrides(t *testing.T) {
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	repo := &fakeFeatureFlagRepo{items: []domain.FeatureFlag{{
		Key:               "role-gated",
		Enabled:           true,
		PercentageRollout: 0,
		TargetRoles:       []domain.Role{domain.RoleViewer},
	}}}
	mgr, err := NewManager(context.Background(), config.FeatureFlagsConfig{Overrides: map[string]bool{"override-flag": true}}, repo, zap.NewNop())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(mgr.Close)

	ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserID: &userID, Role: domain.RoleViewer})
	if !mgr.Evaluate(ctx, "role-gated", false) {
		t.Fatalf("expected role targeting to win over rollout")
	}
	if !mgr.Evaluate(context.Background(), "override-flag", false) {
		t.Fatalf("expected override to win")
	}
	all, err := mgr.EvaluateAll(ctx)
	if err != nil {
		t.Fatalf("evaluate all: %v", err)
	}
	if !all["role-gated"] || !all["override-flag"] {
		t.Fatalf("unexpected evaluation map: %+v", all)
	}
}
