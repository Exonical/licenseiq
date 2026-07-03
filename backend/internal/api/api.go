package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"
)

type Services struct {
	Vendors     app.VendorService
	Products    app.ProductService
	Licenses    app.LicenseService
	Assignments app.AssignmentService
	Attachments app.AttachmentService
}

func NewHumaConfig(title, version string) huma.Config {
	cfg := huma.DefaultConfig(title, version)
	cfg.OpenAPIPath = "/openapi"
	cfg.DocsPath = ""
	cfg.SchemasPath = "/schemas"
	if cfg.Info == nil {
		cfg.Info = &huma.Info{}
	}
	cfg.Info.Description = "LicenseIQ API"
	cfg.Servers = []*huma.Server{{URL: "/api/v1"}}
	return cfg
}

func RegisterRoutes(api huma.API, services Services, logger *zap.Logger) {
	group := huma.NewGroup(api, "/api/v1")
	group.UseMiddleware(requestContextMiddleware)
	registerVendorRoutes(group, services.Vendors, logger)
	registerProductRoutes(group, services.Products, logger)
	registerLicenseRoutes(group, services.Licenses, logger)
	registerAssignmentRoutes(group, services.Assignments, logger)
	registerAttachmentRoutes(group, services.Attachments, logger)
}

func requestContextMiddleware(ctx huma.Context, next func(huma.Context)) {
	rc := app.RequestContext{
		IPAddress: ctx.RemoteAddr(),
		SessionID: ctx.Header("X-Request-ID"),
	}
	next(huma.WithContext(ctx, app.WithRequestContext(ctx.Context(), rc)))
}

func mapServiceError(err error, logger *zap.Logger, ctx context.Context) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return huma.Error404NotFound("resource not found")
	case errors.Is(err, domain.ErrValidation):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, domain.ErrConflict):
		return huma.Error409Conflict("conflict")
	default:
		if logger != nil {
			rc := app.RequestContextFromContext(ctx)
			logger.Error("request failed", zap.String("request_id", rc.SessionID), zap.Error(err))
		}
		return huma.Error500InternalServerError("internal server error")
	}
}

func listFilterFromInput(input ListInput) domain.ListFilter {
	return domain.ListFilter{
		Limit:          input.Limit,
		Offset:         input.Offset,
		IncludeDeleted: input.IncludeDeleted,
	}
}

func toPage[T any](items []T, filter domain.ListFilter) Page[T] {
	return Page[T]{Data: items, Limit: filter.Limit, Offset: filter.Offset, Total: len(items)}
}

func operationErrors() []int {
	return []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity}
}
