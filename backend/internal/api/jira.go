package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type LicenseIssueLinkResponse struct {
	ID          uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	LicenseID   uuid.UUID  `json:"licenseId" example:"550e8400-e29b-41d4-a716-446655440000"`
	IssueKey    string     `json:"issueKey" example:"ABC-123"`
	IssueURL    string     `json:"issueUrl,omitempty" example:"https://jira.example.com/browse/ABC-123"`
	Status      string     `json:"status,omitempty" example:"Open"`
	RenewalDate *time.Time `json:"renewalDate,omitempty" example:"2026-01-01T00:00:00Z"`
	CreatedAt   time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt   time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
}

type JiraLinkIssueBody struct {
	IssueKey string `json:"issueKey" example:"ABC-123"`
	IssueURL string `json:"issueUrl,omitempty" example:"https://jira.example.com/browse/ABC-123"`
}

type JiraIssueStatusBody struct {
	Status string `json:"status" example:"Done"`
}

type JiraCreateRenewalTicketOutput struct{ Body LicenseIssueLinkResponse }
type JiraLinkIssueOutput struct{ Body LicenseIssueLinkResponse }
type JiraListIssueLinksOutput struct {
	Body Page[LicenseIssueLinkResponse]
}
type JiraUpdateIssueStatusOutput struct{ Body LicenseIssueLinkResponse }

func registerJiraRoutes(api huma.API, svc app.JiraService, logger *zap.Logger) {
	if svc == nil {
		return
	}

	huma.Post(api, "/licenses/{id}/jira/renewal-tickets", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*JiraCreateRenewalTicketOutput, error) {
		link, err := svc.CreateRenewalTicket(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &JiraCreateRenewalTicketOutput{Body: licenseIssueLinkToResponse(*link)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createLicenseRenewalTicket"
		o.Summary = "Create Jira renewal ticket"
		o.Description = "Create a Jira renewal ticket for the specified license."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Jira"}
		o.Errors = operationErrors()
		protectedOperation("jira", "manage")(o)
	})

	huma.Post(api, "/licenses/{id}/jira/issues", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body JiraLinkIssueBody
	}) (*JiraLinkIssueOutput, error) {
		link, err := svc.LinkIssue(ctx, input.ID, input.Body.IssueKey, input.Body.IssueURL)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &JiraLinkIssueOutput{Body: licenseIssueLinkToResponse(*link)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "linkLicenseIssue"
		o.Summary = "Link Jira issue"
		o.Description = "Link an existing Jira issue to a license."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Jira"}
		o.Errors = operationErrors()
		protectedOperation("jira", "manage")(o)
	})

	huma.Get(api, "/licenses/{id}/jira/issues", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*JiraListIssueLinksOutput, error) {
		links, err := svc.ListIssueLinks(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]LicenseIssueLinkResponse, 0, len(links))
		for _, link := range links {
			out = append(out, licenseIssueLinkToResponse(link))
		}
		return &JiraListIssueLinksOutput{Body: Page[LicenseIssueLinkResponse]{Data: out, Limit: len(out), Offset: 0, Total: len(out)}}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listLicenseIssueLinks"
		o.Summary = "List Jira links for a license"
		o.Description = "List Jira issue links associated with a license."
		o.Tags = []string{"Jira"}
		o.Errors = operationErrors()
		protectedOperation("jira", "manage")(o)
	})

	huma.Put(api, "/licenses/{id}/jira/issues/{issueKey}/status", func(ctx context.Context, input *struct {
		ID       uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		IssueKey string    `path:"issueKey" example:"ABC-123"`
		Body     JiraIssueStatusBody
	}) (*JiraUpdateIssueStatusOutput, error) {
		link, err := svc.UpdateIssueStatus(ctx, input.ID, input.IssueKey, input.Body.Status)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &JiraUpdateIssueStatusOutput{Body: licenseIssueLinkToResponse(*link)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "updateLicenseIssueStatus"
		o.Summary = "Update Jira issue status"
		o.Description = "Transition a linked Jira issue to the requested status."
		o.Tags = []string{"Jira"}
		o.Errors = operationErrors()
		protectedOperation("jira", "manage")(o)
	})
}

func licenseIssueLinkToResponse(link domain.LicenseIssueLink) LicenseIssueLinkResponse {
	return LicenseIssueLinkResponse{
		ID:          link.ID,
		LicenseID:   link.LicenseID,
		IssueKey:    link.IssueKey,
		IssueURL:    link.IssueURL,
		Status:      link.Status,
		RenewalDate: link.RenewalDate,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}
}
