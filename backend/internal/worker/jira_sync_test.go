package worker

import (
	"context"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"github.com/Exonical/licenseiq/backend/internal/jira"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type jiraSyncFakeJiraClient struct {
	createCalls int
}

func (f *jiraSyncFakeJiraClient) CreateIssue(context.Context, jira.CreateIssueRequest) (*jira.CreateIssueResponse, error) {
	f.createCalls++
	return &jira.CreateIssueResponse{Key: "ABC-1", URL: "https://jira.example.com/browse/ABC-1", Status: "Open"}, nil
}
func (f *jiraSyncFakeJiraClient) TransitionIssue(context.Context, jira.TransitionIssueRequest) error {
	return nil
}
func (f *jiraSyncFakeJiraClient) LinkIssue(context.Context, jira.LinkIssueRequest) error { return nil }
func (f *jiraSyncFakeJiraClient) AttachFile(context.Context, jira.AttachFileRequest) error {
	return nil
}

type jiraSyncFakeLicenseRepo struct{ licenses []domain.License }

func (f jiraSyncFakeLicenseRepo) Create(context.Context, *domain.License) error { return nil }
func (f jiraSyncFakeLicenseRepo) Get(context.Context, uuid.UUID) (*domain.License, error) {
	if len(f.licenses) > 0 {
		lic := f.licenses[0]
		return &lic, nil
	}
	return nil, domain.ErrNotFound
}
func (f jiraSyncFakeLicenseRepo) Update(context.Context, *domain.License) error { return nil }
func (f jiraSyncFakeLicenseRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f jiraSyncFakeLicenseRepo) List(context.Context, domain.ListFilter) ([]domain.License, error) {
	return append([]domain.License(nil), f.licenses...), nil
}

type jiraSyncFakeVendorRepo struct{ vendor domain.Vendor }

func (f jiraSyncFakeVendorRepo) Create(context.Context, *domain.Vendor) error { return nil }
func (f jiraSyncFakeVendorRepo) Get(context.Context, uuid.UUID) (*domain.Vendor, error) {
	v := f.vendor
	return &v, nil
}
func (f jiraSyncFakeVendorRepo) Update(context.Context, *domain.Vendor) error { return nil }
func (f jiraSyncFakeVendorRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (f jiraSyncFakeVendorRepo) List(context.Context, domain.ListFilter) ([]domain.Vendor, error) {
	return []domain.Vendor{f.vendor}, nil
}

type jiraSyncFakeProductRepo struct{ product domain.Product }

func (f jiraSyncFakeProductRepo) Create(context.Context, *domain.Product) error { return nil }
func (f jiraSyncFakeProductRepo) Get(context.Context, uuid.UUID) (*domain.Product, error) {
	p := f.product
	return &p, nil
}
func (f jiraSyncFakeProductRepo) Update(context.Context, *domain.Product) error { return nil }
func (f jiraSyncFakeProductRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f jiraSyncFakeProductRepo) List(context.Context, domain.ListFilter) ([]domain.Product, error) {
	return []domain.Product{f.product}, nil
}

type jiraSyncFakeLinkRepo struct {
	links map[string]domain.LicenseIssueLink
}

func (f *jiraSyncFakeLinkRepo) key(licenseID uuid.UUID, renewalDate time.Time) string {
	return licenseID.String() + "|" + renewalDate.UTC().Format(time.RFC3339)
}
func (f *jiraSyncFakeLinkRepo) Create(_ context.Context, link *domain.LicenseIssueLink) error {
	if f.links == nil {
		f.links = map[string]domain.LicenseIssueLink{}
	}
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	if link.RenewalDate != nil {
		f.links[f.key(link.LicenseID, *link.RenewalDate)] = *link
	}
	return nil
}
func (f *jiraSyncFakeLinkRepo) Get(context.Context, uuid.UUID) (*domain.LicenseIssueLink, error) {
	return nil, domain.ErrNotFound
}
func (f *jiraSyncFakeLinkRepo) Update(_ context.Context, link *domain.LicenseIssueLink) error {
	if link.RenewalDate != nil {
		f.links[f.key(link.LicenseID, *link.RenewalDate)] = *link
	}
	return nil
}
func (f *jiraSyncFakeLinkRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f *jiraSyncFakeLinkRepo) List(context.Context, domain.ListFilter) ([]domain.LicenseIssueLink, error) {
	out := make([]domain.LicenseIssueLink, 0, len(f.links))
	for _, link := range f.links {
		out = append(out, link)
	}
	return out, nil
}
func (f *jiraSyncFakeLinkRepo) ListByLicense(_ context.Context, licenseID uuid.UUID) ([]domain.LicenseIssueLink, error) {
	out := []domain.LicenseIssueLink{}
	for _, link := range f.links {
		if link.LicenseID == licenseID {
			out = append(out, link)
		}
	}
	return out, nil
}
func (f *jiraSyncFakeLinkRepo) GetByLicenseAndIssueKey(_ context.Context, licenseID uuid.UUID, issueKey string) (*domain.LicenseIssueLink, error) {
	for _, link := range f.links {
		if link.LicenseID == licenseID && link.IssueKey == issueKey {
			copy := link
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *jiraSyncFakeLinkRepo) GetByLicenseAndRenewalDate(_ context.Context, licenseID uuid.UUID, renewalDate time.Time) (*domain.LicenseIssueLink, error) {
	if f.links == nil {
		return nil, domain.ErrNotFound
	}
	if link, ok := f.links[f.key(licenseID, renewalDate)]; ok {
		copy := link
		return &copy, nil
	}
	return nil, domain.ErrNotFound
}

type jiraSyncFakeAuditRepo struct{}

func (jiraSyncFakeAuditRepo) Create(context.Context, *domain.AuditLog) error { return nil }
func (jiraSyncFakeAuditRepo) Get(context.Context, uuid.UUID) (*domain.AuditLog, error) {
	return nil, domain.ErrNotFound
}
func (jiraSyncFakeAuditRepo) List(context.Context, domain.ListFilter) ([]domain.AuditLog, error) {
	return nil, nil
}

type jiraSyncFakeFlagRepo struct{}

func (jiraSyncFakeFlagRepo) Create(context.Context, *domain.FeatureFlag) error { return nil }
func (jiraSyncFakeFlagRepo) Get(context.Context, uuid.UUID) (*domain.FeatureFlag, error) {
	return nil, domain.ErrNotFound
}
func (jiraSyncFakeFlagRepo) Update(context.Context, *domain.FeatureFlag) error { return nil }
func (jiraSyncFakeFlagRepo) Delete(context.Context, uuid.UUID) error           { return nil }
func (jiraSyncFakeFlagRepo) List(context.Context, domain.ListFilter) ([]domain.FeatureFlag, error) {
	return nil, nil
}

func TestJiraSyncJobCreatesTicketAndDedups(t *testing.T) {
	renewal := time.Now().UTC().AddDate(0, 0, 30)
	licenseID := uuid.New()
	vendorID := uuid.New()
	productID := uuid.New()
	client := &jiraSyncFakeJiraClient{}
	licenseRepo := jiraSyncFakeLicenseRepo{licenses: []domain.License{{Base: domain.Base{ID: licenseID}, VendorID: vendorID, ProductID: productID, RenewalDate: &renewal, LicenseKey: "LIC-1"}}}
	vendorRepo := jiraSyncFakeVendorRepo{vendor: domain.Vendor{Base: domain.Base{ID: vendorID}, Name: "Acme"}}
	productRepo := jiraSyncFakeProductRepo{product: domain.Product{Base: domain.Base{ID: productID}, Name: "Widget"}}
	linkRepo := &jiraSyncFakeLinkRepo{}
	svc := app.NewJiraService(client, "PROJ", "Task", licenseRepo, vendorRepo, productRepo, nil, linkRepo, jiraSyncFakeAuditRepo{})
	job := NewJiraSyncJob(24*time.Hour, 90, nil, licenseRepo, linkRepo, svc, zap.NewNop())
	job.clock = func() time.Time { return renewal.Add(-24 * time.Hour) }

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", client.createCalls)
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected deduped create call, got %d", client.createCalls)
	}
}

func TestJiraSyncJobNoopWhenDisabledOrFlagOff(t *testing.T) {
	renewal := time.Now().UTC().AddDate(0, 0, 30)
	licenseID := uuid.New()
	licenseRepo := jiraSyncFakeLicenseRepo{licenses: []domain.License{{Base: domain.Base{ID: licenseID}, RenewalDate: &renewal}}}
	linkRepo := &jiraSyncFakeLinkRepo{}
	client := &jiraSyncFakeJiraClient{}
	svc := app.NewJiraService(client, "PROJ", "Task", licenseRepo, jiraSyncFakeVendorRepo{}, jiraSyncFakeProductRepo{}, nil, linkRepo, jiraSyncFakeAuditRepo{})

	if err := NewJiraSyncJob(24*time.Hour, 90, nil, licenseRepo, linkRepo, nil, zap.NewNop()).Run(context.Background()); err != nil {
		t.Fatalf("nil svc run: %v", err)
	}
	if client.createCalls != 0 {
		t.Fatalf("expected no calls for nil svc")
	}

	mgr, err := featureflags.NewManager(context.Background(), config.FeatureFlagsConfig{Overrides: map[string]bool{jiraSyncFlagKey: false}}, jiraSyncFakeFlagRepo{}, zap.NewNop())
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer mgr.Close()
	job := NewJiraSyncJob(24*time.Hour, 90, mgr, licenseRepo, linkRepo, svc, zap.NewNop())
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("flag-off run: %v", err)
	}
	if client.createCalls != 0 {
		t.Fatalf("expected no calls when flag off")
	}
}
