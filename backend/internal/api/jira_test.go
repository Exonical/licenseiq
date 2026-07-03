package api

import (
	"context"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type jiraAPIFake struct {
	link domain.LicenseIssueLink
}

func (f *jiraAPIFake) CreateRenewalTicket(context.Context, uuid.UUID) (*domain.LicenseIssueLink, error) {
	link := f.link
	return &link, nil
}
func (f *jiraAPIFake) LinkIssue(context.Context, uuid.UUID, string, string) (*domain.LicenseIssueLink, error) {
	link := f.link
	return &link, nil
}
func (f *jiraAPIFake) ListIssueLinks(context.Context, uuid.UUID) ([]domain.LicenseIssueLink, error) {
	return []domain.LicenseIssueLink{f.link}, nil
}
func (f *jiraAPIFake) UpdateIssueStatus(context.Context, uuid.UUID, string, string) (*domain.LicenseIssueLink, error) {
	link := f.link
	link.Status = "Done"
	return &link, nil
}
func (f *jiraAPIFake) AttachIssueFile(context.Context, uuid.UUID, string, uuid.UUID) error {
	return nil
}

func newJiraAuthManager(t *testing.T) (*auth.Manager, string, string) {
	t.Helper()
	managerUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "manager@example.com", DisplayName: "Manager", Role: domain.RoleLicenseManager, IsServiceAccount: true, Active: true}
	viewerUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "viewer@example.com", DisplayName: "Viewer", Role: domain.RoleViewer, IsServiceAccount: true, Active: true}
	managerToken := "liq_manager.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	viewerToken := "liq_viewer.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	userRepo := &authUserRepo{items: map[uuid.UUID]domain.User{managerUser.ID: managerUser, viewerUser.ID: viewerUser}}
	keyRepo := &authKeyRepo{itemsByKeyID: map[string]domain.APIKey{
		"manager": {Base: domain.Base{ID: uuid.New()}, OwnerUserID: managerUser.ID, KeyID: "manager", HashedKey: sha256Hex(managerToken), Name: "manager", Active: true},
		"viewer":  {Base: domain.Base{ID: uuid.New()}, OwnerUserID: viewerUser.ID, KeyID: "viewer", HashedKey: sha256Hex(viewerToken), Name: "viewer", Active: true},
	}}
	identity := app.NewIdentityService(userRepo, keyRepo, nil)
	mgr, err := auth.NewManager(context.Background(), config.AuthConfig{}, identity, userRepo, keyRepo, zap.NewNop())
	if err != nil {
		t.Fatalf("new auth manager: %v", err)
	}
	return mgr, managerToken, viewerToken
}

func TestJiraEndpointsAuthorizationAndResponses(t *testing.T) {
	mgr, managerToken, viewerToken := newJiraAuthManager(t)
	link := domain.LicenseIssueLink{Base: domain.Base{ID: uuid.New(), CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, LicenseID: uuid.New(), IssueKey: "ABC-1", IssueURL: "https://jira.example.com/browse/ABC-1", Status: "Open"}
	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{Jira: &jiraAPIFake{link: link}}, zap.NewNop(), mgr, nil)

	resp := api.Post("/api/v1/licenses/"+link.LicenseID.String()+"/jira/renewal-tickets", "Authorization: Bearer "+viewerToken)
	if resp.Code != 403 {
		t.Fatalf("expected viewer forbidden, got %d", resp.Code)
	}

	resp = api.Post("/api/v1/licenses/"+link.LicenseID.String()+"/jira/renewal-tickets", "Authorization: Bearer "+managerToken)
	if resp.Code != 201 {
		t.Fatalf("expected manager create success, got %d", resp.Code)
	}

	resp = api.Post("/api/v1/licenses/"+link.LicenseID.String()+"/jira/issues", map[string]any{"issueKey": "ABC-1", "issueUrl": "https://jira.example.com/browse/ABC-1"}, "Authorization: Bearer "+managerToken)
	if resp.Code != 201 {
		t.Fatalf("expected link success, got %d", resp.Code)
	}

	resp = api.Get("/api/v1/licenses/"+link.LicenseID.String()+"/jira/issues", "Authorization: Bearer "+managerToken)
	if resp.Code != 200 {
		t.Fatalf("expected list success, got %d", resp.Code)
	}

	resp = api.Put("/api/v1/licenses/"+link.LicenseID.String()+"/jira/issues/ABC-1/status", map[string]any{"status": "Done"}, "Authorization: Bearer "+managerToken)
	if resp.Code != 200 {
		t.Fatalf("expected update success, got %d", resp.Code)
	}
}
