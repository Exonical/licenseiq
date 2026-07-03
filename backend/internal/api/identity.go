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

type ServiceAccountBody struct {
	Email            string `json:"email" example:"admin@example.com"`
	DisplayName      string `json:"displayName,omitempty" example:"Bootstrap Admin"`
	Role             string `json:"role" enum:"Administrator,LicenseManager,Auditor,Finance,Viewer" example:"Administrator"`
	Active           bool   `json:"active" example:"true"`
	IsServiceAccount bool   `json:"isServiceAccount" example:"true"`
}

type UserResponse struct {
	ID               uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt        time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt        time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt        *time.Time `json:"deletedAt,omitempty"`
	Email            string     `json:"email" example:"admin@example.com"`
	DisplayName      string     `json:"displayName,omitempty" example:"Bootstrap Admin"`
	ExternalSubject  string     `json:"externalSubject,omitempty" example:"oidc-subject-123"`
	Role             string     `json:"role" enum:"Administrator,LicenseManager,Auditor,Finance,Viewer" example:"Administrator"`
	IsServiceAccount bool       `json:"isServiceAccount" example:"true"`
	Active           bool       `json:"active" example:"true"`
}

type ServiceAccountCreateInput struct{ Body ServiceAccountBody }
type ServiceAccountUpdateInput struct{ Body ServiceAccountBody }
type ServiceAccountGetOutput struct{ Body UserResponse }
type ServiceAccountListOutput struct{ Body Page[UserResponse] }
type ServiceAccountDeleteOutput struct{}

type APIKeyBody struct {
	Name      string     `json:"name" example:"cli"`
	Scopes    []string   `json:"scopes,omitempty" example:"[\"read\"]"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty" example:"2027-01-01T00:00:00Z"`
}

type APIKeyResponse struct {
	ID          uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt   time.Time  `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt   time.Time  `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
	KeyID       string     `json:"keyId" example:"abc123def456"`
	OwnerUserID uuid.UUID  `json:"ownerUserId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string     `json:"name" example:"cli"`
	Active      bool       `json:"active" example:"true"`
	Scopes      []string   `json:"scopes,omitempty" example:"[\"read\"]"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty" example:"2027-01-01T00:00:00Z"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty" example:"2026-01-01T00:00:00Z"`
}

type APIKeyCreateResponse struct {
	APIKeyResponse
	Plaintext string `json:"plaintext" example:"liq_abc123def456.abcdef"`
}

type APIKeyCreateInput struct{ Body APIKeyBody }
type APIKeyCreateOutput struct{ Body APIKeyCreateResponse }
type APIKeyListOutput struct{ Body Page[APIKeyResponse] }
type APIKeyDeleteOutput struct{}

func registerIdentityRoutes(api huma.API, svc app.IdentityService, logger *zap.Logger) {
	if svc == nil {
		return
	}
	// service accounts
	huma.Post(api, "/service-accounts", func(ctx context.Context, input *ServiceAccountCreateInput) (*ServiceAccountGetOutput, error) {
		created, err := svc.CreateServiceAccount(ctx, serviceAccountBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &ServiceAccountGetOutput{Body: userToResponse(*created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createServiceAccount"
		o.Summary = "Create service account"
		o.Description = "Create a service account user."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Users"}
		o.Errors = operationErrors()
		protectedOperation("service_accounts", "write")(o)
	})

	huma.Get(api, "/service-accounts", func(ctx context.Context, input *struct{ ListInput }) (*ServiceAccountListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		items, err := svc.ListServiceAccounts(ctx, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]UserResponse, 0, len(items))
		for _, item := range items {
			out = append(out, userToResponse(item))
		}
		return &ServiceAccountListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listServiceAccounts"
		o.Summary = "List service accounts"
		o.Description = "List service account users with pagination."
		o.Tags = []string{"Users"}
		o.Errors = operationErrors()
		protectedOperation("service_accounts", "read")(o)
	})

	huma.Get(api, "/service-accounts/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*ServiceAccountGetOutput, error) {
		item, err := svc.GetServiceAccount(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &ServiceAccountGetOutput{Body: userToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getServiceAccount"
		o.Summary = "Get service account"
		o.Description = "Get a service account by id."
		o.Tags = []string{"Users"}
		o.Errors = operationErrors()
		protectedOperation("service_accounts", "read")(o)
	})

	huma.Put(api, "/service-accounts/{id}", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body ServiceAccountBody
	}) (*ServiceAccountGetOutput, error) {
		updated, err := svc.UpdateServiceAccount(ctx, input.ID, serviceAccountBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &ServiceAccountGetOutput{Body: userToResponse(*updated)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "updateServiceAccount"
		o.Summary = "Update service account"
		o.Description = "Update a service account."
		o.Tags = []string{"Users"}
		o.Errors = operationErrors()
		protectedOperation("service_accounts", "write")(o)
	})

	huma.Delete(api, "/service-accounts/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.DeleteServiceAccount(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteServiceAccount"
		o.Summary = "Delete service account"
		o.Description = "Soft-delete a service account."
		o.DefaultStatus = http.StatusNoContent
		o.Tags = []string{"Users"}
		o.Errors = operationErrors()
		protectedOperation("service_accounts", "admin")(o)
	})

	// api keys
	huma.Get(api, "/service-accounts/{id}/api-keys", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		ListInput
	}) (*APIKeyListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		items, err := svc.ListAPIKeys(ctx, input.ID, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]APIKeyResponse, 0, len(items))
		for _, item := range items {
			out = append(out, apiKeyToResponse(item))
		}
		return &APIKeyListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listServiceAccountAPIKeys"
		o.Summary = "List API keys"
		o.Description = "List API keys for a service account."
		o.Tags = []string{"API Keys"}
		o.Errors = operationErrors()
		protectedOperation("api_keys", "read")(o)
	})

	huma.Post(api, "/service-accounts/{id}/api-keys", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body APIKeyBody
	}) (*APIKeyCreateOutput, error) {
		created, plain, err := svc.CreateAPIKey(ctx, apiKeyBodyToDomain(input.ID, input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &APIKeyCreateOutput{Body: APIKeyCreateResponse{APIKeyResponse: apiKeyToResponse(*created), Plaintext: plain}}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createServiceAccountAPIKey"
		o.Summary = "Create API key"
		o.Description = "Create an API key for a service account."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"API Keys"}
		o.Errors = operationErrors()
		protectedOperation("api_keys", "write")(o)
	})

	huma.Delete(api, "/api-keys/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.DeleteAPIKey(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteAPIKey"
		o.Summary = "Delete API key"
		o.Description = "Revoke an API key."
		o.DefaultStatus = http.StatusNoContent
		o.Tags = []string{"API Keys"}
		o.Errors = operationErrors()
		protectedOperation("api_keys", "admin")(o)
	})
}

func userToResponse(u domain.User) UserResponse {
	return UserResponse{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, DeletedAt: u.DeletedAt, Email: u.Email, DisplayName: u.DisplayName, ExternalSubject: u.ExternalSubject, Role: u.Role.String(), IsServiceAccount: u.IsServiceAccount, Active: u.Active}
}

func serviceAccountBodyToDomain(b ServiceAccountBody) domain.User {
	role, err := domain.ParseRole(b.Role)
	if err != nil {
		role = domain.RoleAdministrator
	}
	return domain.User{Email: b.Email, DisplayName: b.DisplayName, Role: role, IsServiceAccount: true, Active: b.Active}
}

func apiKeyToResponse(k domain.APIKey) APIKeyResponse {
	return APIKeyResponse{ID: k.ID, CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt, DeletedAt: k.DeletedAt, KeyID: k.KeyID, OwnerUserID: k.OwnerUserID, Name: k.Name, Active: k.Active, Scopes: append([]string(nil), k.Scopes...), ExpiresAt: k.ExpiresAt, LastUsedAt: k.LastUsedAt}
}

func apiKeyBodyToDomain(ownerID uuid.UUID, b APIKeyBody) domain.APIKey {
	return domain.APIKey{OwnerUserID: ownerID, Name: b.Name, Active: true, Scopes: append([]string(nil), b.Scopes...), ExpiresAt: b.ExpiresAt}
}
