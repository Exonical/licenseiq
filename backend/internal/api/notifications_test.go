package api

import (
	"testing"

	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestNotificationTestEndpointAuthorization(t *testing.T) {
	adminUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "admin@example.com", DisplayName: "Admin", Role: domain.RoleAdministrator, IsServiceAccount: true, Active: true}
	viewerUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "viewer@example.com", DisplayName: "Viewer", Role: domain.RoleViewer, IsServiceAccount: true, Active: true}
	adminToken := "liq_admin.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	viewerToken := "liq_viewer.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	authMgr := newAuthManager(t, adminUser, viewerUser, adminToken, viewerToken)
	dispatcher, err := notify.NewDispatcher(config.NotificationsConfig{})
	if err != nil {
		t.Fatalf("dispatcher: %v", err)
	}
	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{Notifications: dispatcher}, zap.NewNop(), authMgr, nil)

	resp := api.Post("/api/v1/notifications/test", "Authorization: Bearer "+viewerToken)
	if resp.Code != 403 {
		t.Fatalf("expected viewer forbidden, got %d", resp.Code)
	}
	resp = api.Post("/api/v1/notifications/test", "Authorization: Bearer "+adminToken)
	if resp.Code != 200 {
		t.Fatalf("expected admin success, got %d", resp.Code)
	}
}
