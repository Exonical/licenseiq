package api

import (
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

type FeatureFlagBody struct {
	Key                string      `json:"key" example:"new-dashboard"`
	Description        string      `json:"description,omitempty" example:"Enable the new dashboard"`
	Enabled            bool        `json:"enabled" example:"true"`
	PercentageRollout  int         `json:"percentageRollout" minimum:"0" maximum:"100" example:"50"`
	TargetUserIDs      []uuid.UUID `json:"targetUserIds,omitempty" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	TargetRoles        []string    `json:"targetRoles,omitempty" example:"[\"Viewer\",\"LicenseManager\"]"`
	ScheduledEnableAt  *time.Time  `json:"scheduledEnableAt,omitempty" example:"2026-01-01T00:00:00Z"`
	ScheduledDisableAt *time.Time  `json:"scheduledDisableAt,omitempty" example:"2026-12-31T00:00:00Z"`
}

type FeatureFlagResponse struct {
	ID                 uuid.UUID   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt          time.Time   `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt          time.Time   `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
	DeletedAt          *time.Time  `json:"deletedAt,omitempty"`
	Key                string      `json:"key" example:"new-dashboard"`
	Description        string      `json:"description,omitempty" example:"Enable the new dashboard"`
	Enabled            bool        `json:"enabled" example:"true"`
	PercentageRollout  int         `json:"percentageRollout" example:"50"`
	TargetUserIDs      []uuid.UUID `json:"targetUserIds,omitempty"`
	TargetRoles        []string    `json:"targetRoles,omitempty" example:"[\"Viewer\",\"LicenseManager\"]"`
	ScheduledEnableAt  *time.Time  `json:"scheduledEnableAt,omitempty"`
	ScheduledDisableAt *time.Time  `json:"scheduledDisableAt,omitempty"`
}

type FeatureFlagEvaluationResponse struct {
	Enabled bool `json:"enabled" example:"true"`
}

type FeatureFlagEvaluationAllResponse struct {
	Flags map[string]bool `json:"flags"`
}

type FeatureFlagCreateInput struct{ Body FeatureFlagBody }
type FeatureFlagUpdateInput struct{ Body FeatureFlagBody }
type FeatureFlagGetOutput struct{ Body FeatureFlagResponse }
type FeatureFlagListOutput struct{ Body Page[FeatureFlagResponse] }
type FeatureFlagDeleteOutput struct{}
type FeatureFlagEvaluateOutput struct{ Body FeatureFlagEvaluationResponse }
type FeatureFlagEvaluateAllOutput struct {
	Body FeatureFlagEvaluationAllResponse
}

func featureFlagBodyToDomain(body FeatureFlagBody) domain.FeatureFlag {
	roles := make([]domain.Role, 0, len(body.TargetRoles))
	for _, role := range body.TargetRoles {
		roles = append(roles, domain.Role(role))
	}
	return domain.FeatureFlag{
		Key:                body.Key,
		Description:        body.Description,
		Enabled:            body.Enabled,
		PercentageRollout:  body.PercentageRollout,
		TargetUserIDs:      append([]uuid.UUID(nil), body.TargetUserIDs...),
		TargetRoles:        roles,
		ScheduledEnableAt:  body.ScheduledEnableAt,
		ScheduledDisableAt: body.ScheduledDisableAt,
	}
}

func featureFlagToResponse(flag domain.FeatureFlag) FeatureFlagResponse {
	roles := make([]string, 0, len(flag.TargetRoles))
	for _, role := range flag.TargetRoles {
		roles = append(roles, role.String())
	}
	return FeatureFlagResponse{
		ID:                 flag.ID,
		CreatedAt:          flag.CreatedAt,
		UpdatedAt:          flag.UpdatedAt,
		DeletedAt:          flag.DeletedAt,
		Key:                flag.Key,
		Description:        flag.Description,
		Enabled:            flag.Enabled,
		PercentageRollout:  flag.PercentageRollout,
		TargetUserIDs:      append([]uuid.UUID(nil), flag.TargetUserIDs...),
		TargetRoles:        roles,
		ScheduledEnableAt:  flag.ScheduledEnableAt,
		ScheduledDisableAt: flag.ScheduledDisableAt,
	}
}
