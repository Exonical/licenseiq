package api

import (
	"context"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerVendorRoutes(api huma.API, svc app.VendorService, logger *zap.Logger) {
	huma.Post(api, "/vendors", func(ctx context.Context, input *VendorCreateInput) (*VendorGetOutput, error) {
		created, err := svc.Create(ctx, vendorBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &VendorGetOutput{Body: vendorToResponse(*created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createVendor"
		o.Summary = "Create vendor"
		o.Description = "Create a vendor and its contacts."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Vendors"}
		o.Errors = operationErrors()
		protectedOperation("vendors", "write")(o)
	})

	huma.Get(api, "/vendors", func(ctx context.Context, input *struct{ ListInput }) (*VendorListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		vendors, err := svc.List(ctx, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]VendorResponse, 0, len(vendors))
		for _, v := range vendors {
			out = append(out, vendorToResponse(v))
		}
		return &VendorListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listVendors"
		o.Summary = "List vendors"
		o.Description = "List vendors with pagination."
		o.Tags = []string{"Vendors"}
		o.Errors = operationErrors()
		protectedOperation("vendors", "read")(o)
	})

	huma.Get(api, "/vendors/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*VendorGetOutput, error) {
		vendor, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &VendorGetOutput{Body: vendorToResponse(*vendor)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getVendor"
		o.Summary = "Get vendor"
		o.Description = "Get a single vendor by id."
		o.Tags = []string{"Vendors"}
		o.Errors = operationErrors()
		protectedOperation("vendors", "read")(o)
	})

	huma.Put(api, "/vendors/{id}", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body VendorBody
	}) (*VendorGetOutput, error) {
		updated, err := svc.Update(ctx, input.ID, vendorBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &VendorGetOutput{Body: vendorToResponse(*updated)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "updateVendor"
		o.Summary = "Update vendor"
		o.Description = "Update a vendor and replace its contacts."
		o.Tags = []string{"Vendors"}
		o.Errors = operationErrors()
		protectedOperation("vendors", "write")(o)
	})

	huma.Delete(api, "/vendors/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteVendor"
		o.Summary = "Delete vendor"
		o.Description = "Soft-delete a vendor."
		o.Tags = []string{"Vendors"}
		o.DefaultStatus = http.StatusNoContent
		o.Errors = operationErrors()
		protectedOperation("vendors", "write")(o)
	})
}
