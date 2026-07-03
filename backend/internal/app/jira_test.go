package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/jira"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeJiraClient struct {
	createCalls int
	lastCreate  jira.CreateIssueRequest
	response    *jira.CreateIssueResponse
}

func (f *fakeJiraClient) CreateIssue(_ context.Context, req jira.CreateIssueRequest) (*jira.CreateIssueResponse, error) {
	f.createCalls++
	f.lastCreate = req
	if f.response != nil {
		return f.response, nil
	}
	return &jira.CreateIssueResponse{Key: "ABC-1", URL: "https://jira.example.com/browse/ABC-1", Status: "Open"}, nil
}
func (f *fakeJiraClient) TransitionIssue(context.Context, jira.TransitionIssueRequest) error {
	return nil
}
func (f *fakeJiraClient) LinkIssue(context.Context, jira.LinkIssueRequest) error   { return nil }
func (f *fakeJiraClient) AttachFile(context.Context, jira.AttachFileRequest) error { return nil }

type fakeVendorRepo struct{ vendor domain.Vendor }

func (f fakeVendorRepo) Create(context.Context, *domain.Vendor) error { return nil }
func (f fakeVendorRepo) Get(context.Context, uuid.UUID) (*domain.Vendor, error) {
	v := f.vendor
	return &v, nil
}
func (f fakeVendorRepo) Update(context.Context, *domain.Vendor) error { return nil }
func (f fakeVendorRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (f fakeVendorRepo) List(context.Context, domain.ListFilter) ([]domain.Vendor, error) {
	return []domain.Vendor{f.vendor}, nil
}

type fakeProductRepo struct{ product domain.Product }

func (f fakeProductRepo) Create(context.Context, *domain.Product) error { return nil }
func (f fakeProductRepo) Get(context.Context, uuid.UUID) (*domain.Product, error) {
	p := f.product
	return &p, nil
}
func (f fakeProductRepo) Update(context.Context, *domain.Product) error { return nil }
func (f fakeProductRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f fakeProductRepo) List(context.Context, domain.ListFilter) ([]domain.Product, error) {
	return []domain.Product{f.product}, nil
}

type fakeLicenseRepo struct{ license domain.License }

func (f fakeLicenseRepo) Create(context.Context, *domain.License) error { return nil }
func (f fakeLicenseRepo) Get(context.Context, uuid.UUID) (*domain.License, error) {
	l := f.license
	return &l, nil
}
func (f fakeLicenseRepo) Update(context.Context, *domain.License) error { return nil }
func (f fakeLicenseRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f fakeLicenseRepo) List(context.Context, domain.ListFilter) ([]domain.License, error) {
	return []domain.License{f.license}, nil
}

type fakeAttachmentRepo struct{}

func (f fakeAttachmentRepo) Create(context.Context, *domain.Attachment) error { return nil }
func (f fakeAttachmentRepo) Get(context.Context, uuid.UUID) (*domain.Attachment, error) {
	return nil, errors.New("not used")
}
func (f fakeAttachmentRepo) Update(context.Context, *domain.Attachment) error { return nil }
func (f fakeAttachmentRepo) Delete(context.Context, uuid.UUID) error          { return nil }
func (f fakeAttachmentRepo) List(context.Context, domain.ListFilter) ([]domain.Attachment, error) {
	return nil, nil
}

type fakeLinkRepo struct {
	links map[string]domain.LicenseIssueLink
}

func (f *fakeLinkRepo) key(licenseID uuid.UUID, renewalDate *time.Time, issueKey string) string {
	if renewalDate != nil {
		return licenseID.String() + "|renewal|" + renewalDate.UTC().Format(time.RFC3339)
	}
	return licenseID.String() + "|issue|" + issueKey
}
func (f *fakeLinkRepo) Create(_ context.Context, link *domain.LicenseIssueLink) error {
	if f.links == nil {
		f.links = map[string]domain.LicenseIssueLink{}
	}
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	f.links[f.key(link.LicenseID, link.RenewalDate, link.IssueKey)] = *link
	return nil
}
func (f *fakeLinkRepo) Get(context.Context, uuid.UUID) (*domain.LicenseIssueLink, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeLinkRepo) Update(_ context.Context, link *domain.LicenseIssueLink) error {
	f.links[f.key(link.LicenseID, link.RenewalDate, link.IssueKey)] = *link
	return nil
}
func (f *fakeLinkRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f *fakeLinkRepo) List(context.Context, domain.ListFilter) ([]domain.LicenseIssueLink, error) {
	out := make([]domain.LicenseIssueLink, 0, len(f.links))
	for _, link := range f.links {
		out = append(out, link)
	}
	return out, nil
}
func (f *fakeLinkRepo) ListByLicense(_ context.Context, licenseID uuid.UUID) ([]domain.LicenseIssueLink, error) {
	out := []domain.LicenseIssueLink{}
	for _, link := range f.links {
		if link.LicenseID == licenseID {
			out = append(out, link)
		}
	}
	return out, nil
}
func (f *fakeLinkRepo) GetByLicenseAndIssueKey(_ context.Context, licenseID uuid.UUID, issueKey string) (*domain.LicenseIssueLink, error) {
	if link, ok := f.links[f.key(licenseID, nil, issueKey)]; ok {
		return &link, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeLinkRepo) GetByLicenseAndRenewalDate(_ context.Context, licenseID uuid.UUID, renewalDate time.Time) (*domain.LicenseIssueLink, error) {
	for _, link := range f.links {
		if link.LicenseID == licenseID && link.RenewalDate != nil && link.RenewalDate.UTC().Equal(renewalDate.UTC()) {
			copy := link
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

type fakeAuditRepo struct{ audits []domain.AuditLog }

func (f *fakeAuditRepo) Create(_ context.Context, audit *domain.AuditLog) error {
	if audit.ID == uuid.Nil {
		audit.ID = uuid.New()
	}
	f.audits = append(f.audits, *audit)
	return nil
}
func (f *fakeAuditRepo) Get(context.Context, uuid.UUID) (*domain.AuditLog, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAuditRepo) List(context.Context, domain.ListFilter) ([]domain.AuditLog, error) {
	return append([]domain.AuditLog(nil), f.audits...), nil
}

func TestJiraServiceCreateRenewalTicket(t *testing.T) {
	renewal := time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC)
	client := &fakeJiraClient{response: &jira.CreateIssueResponse{Key: "ABC-1", URL: "https://jira.example.com/browse/ABC-1", Status: "Open"}}
	vendorID := uuid.New()
	productID := uuid.New()
	license := domain.License{Base: domain.Base{ID: uuid.New()}, VendorID: vendorID, ProductID: productID, RenewalDate: &renewal, LicenseKey: "LIC-1", Department: "IT", Cost: decimal.RequireFromString("10.00"), Currency: "USD"}
	svc := NewJiraService(client, "PROJ", "Task", fakeLicenseRepo{license: license}, fakeVendorRepo{vendor: domain.Vendor{Base: domain.Base{ID: vendorID}, Name: "Acme"}}, fakeProductRepo{product: domain.Product{Base: domain.Base{ID: productID}, Name: "Widget"}}, fakeAttachmentRepo{}, &fakeLinkRepo{}, &fakeAuditRepo{})

	link, err := svc.CreateRenewalTicket(context.Background(), license.ID)
	if err != nil {
		t.Fatalf("create renewal ticket: %v", err)
	}
	if link.IssueKey != "ABC-1" || link.Status != "Open" || link.RenewalDate == nil || !link.RenewalDate.Equal(renewal.UTC()) {
		t.Fatalf("unexpected link: %#v", link)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", client.createCalls)
	}
	if !strings.Contains(client.lastCreate.Summary, "Acme") || !strings.Contains(client.lastCreate.Summary, "Widget") {
		t.Fatalf("unexpected summary: %q", client.lastCreate.Summary)
	}
}

func TestJiraServiceCreateRenewalTicketDedupAndDisabled(t *testing.T) {
	renewal := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	client := &fakeJiraClient{}
	licenseID := uuid.New()
	vendorID := uuid.New()
	productID := uuid.New()
	links := &fakeLinkRepo{}
	svc := NewJiraService(client, "PROJ", "Task", fakeLicenseRepo{license: domain.License{Base: domain.Base{ID: licenseID}, VendorID: vendorID, ProductID: productID, RenewalDate: &renewal}}, fakeVendorRepo{vendor: domain.Vendor{Base: domain.Base{ID: vendorID}, Name: "Acme"}}, fakeProductRepo{product: domain.Product{Base: domain.Base{ID: productID}, Name: "Widget"}}, fakeAttachmentRepo{}, links, &fakeAuditRepo{})

	first, err := svc.CreateRenewalTicket(context.Background(), licenseID)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.CreateRenewalTicket(context.Background(), licenseID)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected deduped link")
	}
	if client.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", client.createCalls)
	}

	disabled := NewJiraService(nil, "PROJ", "Task", fakeLicenseRepo{license: domain.License{Base: domain.Base{ID: licenseID}, VendorID: vendorID, ProductID: productID, RenewalDate: &renewal}}, fakeVendorRepo{vendor: domain.Vendor{Base: domain.Base{ID: vendorID}, Name: "Acme"}}, fakeProductRepo{product: domain.Product{Base: domain.Base{ID: productID}, Name: "Widget"}}, fakeAttachmentRepo{}, links, &fakeAuditRepo{})
	if _, err := disabled.CreateRenewalTicket(context.Background(), licenseID); !errors.Is(err, ErrJiraDisabled) {
		t.Fatalf("expected disabled error, got %v", err)
	}
}
