package api

import (
	"context"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerAssignmentRoutes(api huma.API, svc app.AssignmentService, logger *zap.Logger) {
	huma.Post(api, "/assignments", func(ctx context.Context, input *AssignmentCreateInput) (*AssignmentGetOutput, error) {
		created, err := svc.Create(ctx, assignmentBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &AssignmentGetOutput{Body: assignmentToResponse(*created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createAssignment"
		o.Summary = "Create assignment"
		o.Description = "Create a license assignment."
		o.DefaultStatus = http.StatusCreated
		o.Tags = []string{"Assignments"}
		o.Errors = operationErrors()
		protectedOperation("assignments", "write")(o)
	})

	huma.Get(api, "/assignments", func(ctx context.Context, input *struct{ ListInput }) (*AssignmentListOutput, error) {
		filter := listFilterFromInput(input.ListInput)
		items, err := svc.List(ctx, filter)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		out := make([]AssignmentResponse, 0, len(items))
		for _, item := range items {
			out = append(out, assignmentToResponse(item))
		}
		return &AssignmentListOutput{Body: toPage(out, filter)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "listAssignments"
		o.Summary = "List assignments"
		o.Description = "List assignments with pagination."
		o.Tags = []string{"Assignments"}
		o.Errors = operationErrors()
		protectedOperation("assignments", "read")(o)
	})

	huma.Get(api, "/assignments/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*AssignmentGetOutput, error) {
		item, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &AssignmentGetOutput{Body: assignmentToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getAssignment"
		o.Summary = "Get assignment"
		o.Description = "Get a single assignment by id."
		o.Tags = []string{"Assignments"}
		o.Errors = operationErrors()
		protectedOperation("assignments", "read")(o)
	})

	huma.Put(api, "/assignments/{id}", func(ctx context.Context, input *struct {
		ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		Body AssignmentBody
	}) (*AssignmentGetOutput, error) {
		item, err := svc.Update(ctx, input.ID, assignmentBodyToDomain(input.Body))
		if err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return &AssignmentGetOutput{Body: assignmentToResponse(*item)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "updateAssignment"
		o.Summary = "Update assignment"
		o.Description = "Update an assignment record."
		o.Tags = []string{"Assignments"}
		o.Errors = operationErrors()
		protectedOperation("assignments", "write")(o)
	})

	huma.Delete(api, "/assignments/{id}", func(ctx context.Context, input *struct {
		ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	}) (*struct{}, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, mapServiceError(err, logger, ctx)
		}
		return nil, nil
	}, func(o *huma.Operation) {
		o.OperationID = "deleteAssignment"
		o.Summary = "Delete assignment"
		o.Description = "Soft-delete an assignment."
		o.DefaultStatus = http.StatusNoContent
		o.Tags = []string{"Assignments"}
		o.Errors = operationErrors()
		protectedOperation("assignments", "write")(o)
	})
}
