package api

import (
	"context"
	"net/http"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerFeatureFlagRoutes(api huma.API, svc app.FeatureFlagService, manager *featureflags.Manager, logger *zap.Logger) {
	if svc != nil {
		huma.Post(api, "/feature-flags", func(ctx context.Context, input *FeatureFlagCreateInput) (*FeatureFlagGetOutput, error) {
			created, err := svc.Create(ctx, featureFlagBodyToDomain(input.Body))
			if err != nil {
				return nil, mapServiceError(err, logger, ctx)
			}
			return &FeatureFlagGetOutput{Body: featureFlagToResponse(*created)}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "createFeatureFlag"
			o.Summary = "Create feature flag"
			o.Description = "Create a feature flag with targeting, percentage rollout, and schedule settings."
			o.DefaultStatus = http.StatusCreated
			o.Tags = []string{"Feature Flags"}
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "write")(o)
		})

		huma.Get(api, "/feature-flags", func(ctx context.Context, input *struct{ ListInput }) (*FeatureFlagListOutput, error) {
			flags, err := svc.List(ctx, listFilterFromInput(input.ListInput))
			if err != nil {
				return nil, mapServiceError(err, logger, ctx)
			}
			out := make([]FeatureFlagResponse, 0, len(flags))
			for _, flag := range flags {
				out = append(out, featureFlagToResponse(flag))
			}
			return &FeatureFlagListOutput{Body: toPage(out, listFilterFromInput(input.ListInput))}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "listFeatureFlags"
			o.Summary = "List feature flags"
			o.Description = "List feature flags with pagination."
			o.Tags = []string{"Feature Flags"}
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "read")(o)
		})

		huma.Get(api, "/feature-flags/{id}", func(ctx context.Context, input *struct {
			ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		}) (*FeatureFlagGetOutput, error) {
			flag, err := svc.Get(ctx, input.ID)
			if err != nil {
				return nil, mapServiceError(err, logger, ctx)
			}
			return &FeatureFlagGetOutput{Body: featureFlagToResponse(*flag)}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "getFeatureFlag"
			o.Summary = "Get feature flag"
			o.Description = "Get a feature flag by id."
			o.Tags = []string{"Feature Flags"}
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "read")(o)
		})

		huma.Put(api, "/feature-flags/{id}", func(ctx context.Context, input *struct {
			ID   uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
			Body FeatureFlagBody
		}) (*FeatureFlagGetOutput, error) {
			updated, err := svc.Update(ctx, input.ID, featureFlagBodyToDomain(input.Body))
			if err != nil {
				return nil, mapServiceError(err, logger, ctx)
			}
			return &FeatureFlagGetOutput{Body: featureFlagToResponse(*updated)}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "updateFeatureFlag"
			o.Summary = "Update feature flag"
			o.Description = "Update a feature flag."
			o.Tags = []string{"Feature Flags"}
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "write")(o)
		})

		huma.Delete(api, "/feature-flags/{id}", func(ctx context.Context, input *struct {
			ID uuid.UUID `path:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
		}) (*FeatureFlagDeleteOutput, error) {
			if err := svc.Delete(ctx, input.ID); err != nil {
				return nil, mapServiceError(err, logger, ctx)
			}
			return &FeatureFlagDeleteOutput{}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "deleteFeatureFlag"
			o.Summary = "Delete feature flag"
			o.Description = "Delete a feature flag."
			o.Tags = []string{"Feature Flags"}
			o.DefaultStatus = http.StatusNoContent
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "write")(o)
		})
	}

	if manager != nil {
		huma.Get(api, "/feature-flags/evaluate", func(ctx context.Context, _ *struct{}) (*FeatureFlagEvaluateAllOutput, error) {
			flags, err := manager.EvaluateAll(ctx)
			if err != nil {
				return nil, mapServiceError(err, logger, ctx)
			}
			return &FeatureFlagEvaluateAllOutput{Body: FeatureFlagEvaluationAllResponse{Flags: flags}}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "evaluateAllFeatureFlags"
			o.Summary = "Evaluate feature flags"
			o.Description = "Evaluate all feature flags for the current authenticated principal."
			o.Tags = []string{"Feature Flags"}
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "read")(o)
		})

		huma.Get(api, "/feature-flags/{key}/evaluate", func(ctx context.Context, input *struct {
			Key string `path:"key" example:"new-dashboard"`
		}) (*FeatureFlagEvaluateOutput, error) {
			return &FeatureFlagEvaluateOutput{Body: FeatureFlagEvaluationResponse{Enabled: manager.Evaluate(ctx, input.Key, false)}}, nil
		}, func(o *huma.Operation) {
			o.OperationID = "evaluateFeatureFlag"
			o.Summary = "Evaluate feature flag"
			o.Description = "Evaluate a single feature flag for the current authenticated principal."
			o.Tags = []string{"Feature Flags"}
			o.Errors = operationErrors()
			protectedOperation("feature_flags", "read")(o)
		})
	}
}
