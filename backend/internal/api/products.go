package api

import (
	"context"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerProductRoutes(api huma.API, svc app.ProductService, logger *zap.Logger) {
	if svc == nil {
		return
	}

	huma.Post(api, "/products", func(ctx context.Context, input *ProductCreateInput) (*ProductGetOutput, error) {
		created, err := svc.Create(ctx, productBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &ProductGetOutput{Body: productToResponse(*created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createProduct"
		o.Summary = "Create product"
		o.Description = "Create a product record."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Products"}
		o.Errors = operationErrors()
		protectedOperation("products", "write")(o)
	})

	huma.Get(api, "/products", func(ctx context.Context, input *struct{ ListInput }) (*ProductListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		items, err := svc.List(ctx, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]ProductResponse, 0, len(items))
		for _, item := range items {
			out = append(out, productToResponse(item))
		}
		return &ProductListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listProducts"
		o.Summary = "List products"
		o.Description = "List products with pagination."
		o.Tags = []string{"Products"}
		o.Errors = operationErrors()
		protectedOperation("products", "read")(o)
	})

	huma.Get(api, "/products/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*ProductGetOutput, error) {
		item, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &ProductGetOutput{Body: productToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getProduct"
		o.Summary = "Get product"
		o.Description = "Get a single product by id."
		o.Tags = []string{"Products"}
		o.Errors = operationErrors()
		protectedOperation("products", "read")(o)
	})

	huma.Put(api, "/products/{id}", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body ProductBody
	}) (*ProductGetOutput, error) {
		item, err := svc.Update(ctx, input.ID, productBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &ProductGetOutput{Body: productToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "updateProduct"
		o.Summary = "Update product"
		o.Description = "Update a product."
		o.Tags = []string{"Products"}
		o.Errors = operationErrors()
		protectedOperation("products", "write")(o)
	})

	huma.Delete(api, "/products/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteProduct"
		o.Summary = "Delete product"
		o.Description = "Soft-delete a product."
		o.Tags = []string{"Products"}
		o.DefaultStatus = http.StatusNoContent
		o.Errors = operationErrors()
		protectedOperation("products", "write")(o)
	})
}
