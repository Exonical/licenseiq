package api

import (
	"context"
	"testing"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/reporting"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type reportingAPIFake struct{}

func (reportingAPIFake) UpcomingRenewals(context.Context, app.UpcomingRenewalsParams) (reporting.Table, error) {
	return reportTable("Upcoming Renewals"), nil
}
func (reportingAPIFake) ExpiredLicenses(context.Context, app.ExpiredLicensesParams) (reporting.Table, error) {
	return reportTable("Expired Licenses"), nil
}
func (reportingAPIFake) VendorSpend(context.Context, app.ReportingAsOfParams) (reporting.Table, error) {
	return reportTable("Vendor Spend"), nil
}
func (reportingAPIFake) LicenseUtilization(context.Context, app.ReportingAsOfParams) (reporting.Table, error) {
	return reportTable("License Utilization"), nil
}
func (reportingAPIFake) DepartmentSpend(context.Context, app.ReportingAsOfParams) (reporting.Table, error) {
	return reportTable("Department Spend"), nil
}

func reportTable(title string) reporting.Table {
	return reporting.Table{Title: title, Columns: []string{"A"}, Rows: [][]string{{"1"}}}
}

func newReportingAuthManager(t *testing.T) (*auth.Manager, string, string, string) {
	t.Helper()
	adminUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "admin@example.com", DisplayName: "Admin", Role: domain.RoleAdministrator, IsServiceAccount: true, Active: true}
	viewerUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "viewer@example.com", DisplayName: "Viewer", Role: domain.RoleViewer, IsServiceAccount: true, Active: true}
	financeUser := domain.User{Base: domain.Base{ID: uuid.New()}, Email: "finance@example.com", DisplayName: "Finance", Role: domain.RoleFinance, IsServiceAccount: true, Active: true}
	adminToken := "liq_admin.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	viewerToken := "liq_viewer.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	financeToken := "liq_finance.cccccccccccccccccccccccccccccccc"
	userRepo := &authUserRepo{items: map[uuid.UUID]domain.User{adminUser.ID: adminUser, viewerUser.ID: viewerUser, financeUser.ID: financeUser}}
	keyRepo := &authKeyRepo{itemsByKeyID: map[string]domain.APIKey{
		"admin":   {Base: domain.Base{ID: uuid.New()}, OwnerUserID: adminUser.ID, KeyID: "admin", HashedKey: sha256Hex(adminToken), Name: "admin", Active: true},
		"viewer":  {Base: domain.Base{ID: uuid.New()}, OwnerUserID: viewerUser.ID, KeyID: "viewer", HashedKey: sha256Hex(viewerToken), Name: "viewer", Active: true},
		"finance": {Base: domain.Base{ID: uuid.New()}, OwnerUserID: financeUser.ID, KeyID: "finance", HashedKey: sha256Hex(financeToken), Name: "finance", Active: true},
	}}
	identity := app.NewIdentityService(userRepo, keyRepo, nil)
	mgr, err := auth.NewManager(context.Background(), config.AuthConfig{}, identity, userRepo, keyRepo, zap.NewNop())
	if err != nil {
		t.Fatalf("new auth manager: %v", err)
	}
	return mgr, adminToken, viewerToken, financeToken
}

func TestReportingEndpointsAuthorizationAndFormats(t *testing.T) {
	mgr, adminToken, viewerToken, financeToken := newReportingAuthManager(t)
	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{Reports: reportingAPIFake{}}, zap.NewNop(), mgr, nil)

	resp := api.Get("/api/v1/reports/renewals?format=csv", "Authorization: Bearer "+viewerToken)
	if resp.Code != 200 {
		t.Fatalf("expected viewer operational report success, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", got)
	}

	resp = api.Get("/api/v1/reports/vendor-spend?format=xlsx", "Authorization: Bearer "+viewerToken)
	if resp.Code != 403 {
		t.Fatalf("expected viewer financial report forbidden, got %d", resp.Code)
	}

	resp = api.Get("/api/v1/reports/vendor-spend?format=xlsx", "Authorization: Bearer "+financeToken)
	if resp.Code != 200 {
		t.Fatalf("expected finance report success, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("unexpected xlsx content-type: %s", got)
	}
	if got := resp.Header().Get("Content-Disposition"); got == "" {
		t.Fatalf("expected attachment disposition")
	}

	resp = api.Get("/api/v1/reports/department-spend?format=pdf", "Authorization: Bearer "+adminToken)
	if resp.Code != 200 {
		t.Fatalf("expected admin report success, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("unexpected pdf content-type: %s", got)
	}
}

func TestReportingEndpointsRequireAuth(t *testing.T) {
	mgr, _, _, _ := newReportingAuthManager(t)
	_, api := humatest.New(t, NewHumaConfig("LicenseIQ API", "test"))
	RegisterRoutes(api, Services{Reports: reportingAPIFake{}}, zap.NewNop(), mgr, nil)
	resp := api.Get("/api/v1/reports/utilization")
	if resp.Code != 401 {
		t.Fatalf("expected unauthenticated report request rejected, got %d", resp.Code)
	}
}
