package api

import (
	"context"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerLicenseRoutes(api huma.API, svc app.LicenseService, logger *zap.Logger) {
	huma.Post(api, "/licenses", func(ctx context.Context, input *LicenseCreateInput) (*LicenseGetOutput, error) {
		body, err := licenseBodyToDomain(input.Body)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		created, err := svc.Create(ctx, body)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &LicenseGetOutput{Body: licenseToResponse(*created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createLicense"
		o.Summary = "Create license"
		o.Description = "Create a license record."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Licenses"}
		o.Errors = operationErrors()
		protectedOperation("licenses", "write")(o)
	})

	huma.Get(api, "/licenses", func(ctx context.Context, input *struct{ ListInput }) (*LicenseListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		items, err := svc.List(ctx, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]LicenseResponse, 0, len(items))
		for _, item := range items {
			out = append(out, licenseToResponse(item))
		}
		return &LicenseListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listLicenses"
		o.Summary = "List licenses"
		o.Description = "List licenses with pagination."
		o.Tags = []string{"Licenses"}
		o.Errors = operationErrors()
		protectedOperation("licenses", "read")(o)
	})

	huma.Get(api, "/licenses/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*LicenseGetOutput, error) {
		item, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &LicenseGetOutput{Body: licenseToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getLicense"
		o.Summary = "Get license"
		o.Description = "Get a single license by id."
		o.Tags = []string{"Licenses"}
		o.Errors = operationErrors()
		protectedOperation("licenses", "read")(o)
	})

	huma.Put(api, "/licenses/{id}", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body LicenseBody
	}) (*LicenseGetOutput, error) {
		body, err := licenseBodyToDomain(input.Body)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		item, err := svc.Update(ctx, input.ID, body)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &LicenseGetOutput{Body: licenseToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "updateLicense"
		o.Summary = "Update license"
		o.Description = "Update a license record."
		o.Tags = []string{"Licenses"}
		o.Errors = operationErrors()
		protectedOperation("licenses", "write")(o)
	})

	huma.Delete(api, "/licenses/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteLicense"
		o.Summary = "Delete license"
		o.Description = "Soft-delete a license."
		o.Tags = []string{"Licenses"}
		o.DefaultStatus = http.StatusNoContent
		o.Errors = operationErrors()
		protectedOperation("licenses", "write")(o)
	})
}
