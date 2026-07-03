package api

import (
	"context"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerAttachmentRoutes(api huma.API, svc app.AttachmentService, logger *zap.Logger) {
	huma.Post(api, "/attachments", func(ctx context.Context, input *AttachmentCreateInput) (*AttachmentGetOutput, error) {
		created, err := svc.Create(ctx, attachmentBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &AttachmentGetOutput{Body: attachmentToResponse(*created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createAttachment"
		o.Summary = "Create attachment metadata"
		o.Description = "Create attachment metadata. Binary file storage is handled elsewhere."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Attachments"}
		o.Errors = operationErrors()
	})

	huma.Get(api, "/attachments", func(ctx context.Context, input *struct{ ListInput }) (*AttachmentListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		items, err := svc.List(ctx, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]AttachmentResponse, 0, len(items))
		for _, item := range items {
			out = append(out, attachmentToResponse(item))
		}
		return &AttachmentListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listAttachments"
		o.Summary = "List attachments"
		o.Description = "List attachment metadata with pagination."
		o.Tags = []string{"Attachments"}
		o.Errors = operationErrors()
	})

	huma.Get(api, "/attachments/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*AttachmentGetOutput, error) {
		item, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &AttachmentGetOutput{Body: attachmentToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getAttachment"
		o.Summary = "Get attachment"
		o.Description = "Get attachment metadata by id."
		o.Tags = []string{"Attachments"}
		o.Errors = operationErrors()
	})

	huma.Delete(api, "/attachments/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteAttachment"
		o.Summary = "Delete attachment metadata"
		o.Description = "Soft-delete attachment metadata."
		o.DefaultStatus = http.StatusNoContent
		o.Tags = []string{"Attachments"}
		o.Errors = operationErrors()
	})
}
