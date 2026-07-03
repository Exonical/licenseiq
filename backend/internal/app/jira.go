package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/jira"
	"github.com/google/uuid"
)

var ErrJiraDisabled = fmt.Errorf("jira is not configured")

type jiraService struct {
	client      jira.Client
	projectKey  string
	issueType   string
	licenses    domain.LicenseRepository
	vendors     domain.VendorRepository
	products    domain.ProductRepository
	attachments domain.AttachmentRepository
	links       domain.LicenseIssueLinkRepository
	audits      domain.AuditRepository
}

func NewJiraService(client jira.Client, projectKey, issueType string, licenses domain.LicenseRepository, vendors domain.VendorRepository, products domain.ProductRepository, attachments domain.AttachmentRepository, links domain.LicenseIssueLinkRepository, audits domain.AuditRepository) JiraService {
	return &jiraService{
		client:      client,
		projectKey:  strings.TrimSpace(projectKey),
		issueType:   strings.TrimSpace(issueType),
		licenses:    licenses,
		vendors:     vendors,
		products:    products,
		attachments: attachments,
		links:       links,
		audits:      audits,
	}
}

func (s *jiraService) CreateRenewalTicket(ctx context.Context, licenseID uuid.UUID) (*domain.LicenseIssueLink, error) {
	if s.client == nil {
		return nil, ErrJiraDisabled
	}
	license, err := s.licenses.Get(ctx, licenseID)
	if err != nil {
		return nil, err
	}
	if license.RenewalDate == nil {
		return nil, fmt.Errorf("license renewal date is required")
	}
	if existing, err := s.links.GetByLicenseAndRenewalDate(ctx, license.ID, license.RenewalDate.UTC()); err == nil {
		return existing, nil
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	vendorName := ""
	if vendor, err := s.vendors.Get(ctx, license.VendorID); err == nil {
		vendorName = vendor.Name
	}
	productName := ""
	if product, err := s.products.Get(ctx, license.ProductID); err == nil {
		productName = product.Name
	}
	if s.projectKey == "" || s.issueType == "" {
		return nil, fmt.Errorf("jira project key and issue type are required")
	}
	resp, err := s.client.CreateIssue(ctx, jira.CreateIssueRequest{
		ProjectKey:  s.projectKey,
		IssueType:   s.issueType,
		Summary:     renewalTicketSummary(vendorName, productName, *license.RenewalDate),
		Description: renewalTicketDescription(license, vendorName, productName),
	})
	if err != nil {
		return nil, err
	}
	status := resp.Status
	if status == "" {
		status = "Created"
	}
	link := domain.LicenseIssueLink{
		LicenseID:   license.ID,
		IssueKey:    resp.Key,
		IssueURL:    resp.URL,
		Status:      status,
		RenewalDate: cloneTime(license.RenewalDate),
	}
	if err := s.links.Create(ctx, &link); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			if existing, lookupErr := s.links.GetByLicenseAndRenewalDate(ctx, license.ID, license.RenewalDate.UTC()); lookupErr == nil {
				return existing, nil
			} else if lookupErr != nil && !errors.Is(lookupErr, domain.ErrNotFound) {
				return nil, lookupErr
			}
		}
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "license_issue_link", link.ID, nil, link); err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *jiraService) LinkIssue(ctx context.Context, licenseID uuid.UUID, issueKey, issueURL string) (*domain.LicenseIssueLink, error) {
	if s.client == nil {
		return nil, ErrJiraDisabled
	}
	if _, err := s.licenses.Get(ctx, licenseID); err != nil {
		return nil, err
	}
	if existing, err := s.links.GetByLicenseAndIssueKey(ctx, licenseID, issueKey); err == nil {
		return existing, nil
	}
	if err := s.client.LinkIssue(ctx, jira.LinkIssueRequest{IssueKey: issueKey, Comment: fmt.Sprintf("Linked to LicenseIQ license %s", licenseID)}); err != nil {
		return nil, err
	}
	link := domain.LicenseIssueLink{LicenseID: licenseID, IssueKey: issueKey, IssueURL: issueURL, Status: "Linked"}
	if err := s.links.Create(ctx, &link); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "license_issue_link", link.ID, nil, link); err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *jiraService) ListIssueLinks(ctx context.Context, licenseID uuid.UUID) ([]domain.LicenseIssueLink, error) {
	if s.client == nil {
		return nil, ErrJiraDisabled
	}
	return s.links.ListByLicense(ctx, licenseID)
}

func (s *jiraService) UpdateIssueStatus(ctx context.Context, licenseID uuid.UUID, issueKey, status string) (*domain.LicenseIssueLink, error) {
	if s.client == nil {
		return nil, ErrJiraDisabled
	}
	link, err := s.links.GetByLicenseAndIssueKey(ctx, licenseID, issueKey)
	if err != nil {
		return nil, err
	}
	if err := s.client.TransitionIssue(ctx, jira.TransitionIssueRequest{IssueKey: issueKey, Status: status}); err != nil {
		return nil, err
	}
	previous := *link
	link.Status = status
	if err := s.links.Update(ctx, link); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "license_issue_link", link.ID, previous, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *jiraService) AttachIssueFile(ctx context.Context, licenseID uuid.UUID, issueKey string, attachmentID uuid.UUID) error {
	if s.client == nil {
		return ErrJiraDisabled
	}
	if _, err := s.links.GetByLicenseAndIssueKey(ctx, licenseID, issueKey); err != nil {
		return err
	}
	attachment, err := s.attachments.Get(ctx, attachmentID)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(attachment.StorageKey)
	if err != nil {
		return err
	}
	return s.client.AttachFile(ctx, jira.AttachFileRequest{
		IssueKey:    issueKey,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		Data:        data,
	})
}

func renewalTicketSummary(vendorName, productName string, renewalDate time.Time) string {
	parts := []string{"Renewal ticket"}
	if vendorName != "" {
		parts = append(parts, vendorName)
	}
	if productName != "" {
		parts = append(parts, productName)
	}
	return strings.Join(parts, " - ") + " - " + renewalDate.UTC().Format("2006-01-02")
}

func renewalTicketDescription(license *domain.License, vendorName, productName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "License ID: %s\n", license.ID)
	if vendorName != "" {
		fmt.Fprintf(&b, "Vendor: %s\n", vendorName)
	}
	if productName != "" {
		fmt.Fprintf(&b, "Product: %s\n", productName)
	}
	if license.Department != "" {
		fmt.Fprintf(&b, "Department: %s\n", license.Department)
	}
	if license.RenewalDate != nil {
		fmt.Fprintf(&b, "Renewal Date: %s\n", license.RenewalDate.UTC().Format(time.RFC3339))
	}
	return b.String()
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cloned := t.UTC()
	return &cloned
}
