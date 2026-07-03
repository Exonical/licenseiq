package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FeatureFlag struct {
	Base
	Key                string
	Description        string
	Enabled            bool
	PercentageRollout  int
	TargetUserIDs      []uuid.UUID
	TargetRoles        []Role
	ScheduledEnableAt  *time.Time
	ScheduledDisableAt *time.Time
}

type FeatureFlagAudit struct {
	Base
	FeatureFlagID  uuid.UUID
	ActorUserID    *uuid.UUID
	Action         AuditAction
	PreviousValues []byte
	NewValues      []byte
}

func (f *FeatureFlag) Validate() error {
	if f == nil {
		return fmt.Errorf("%w: feature flag is nil", ErrValidation)
	}
	f.Key = strings.TrimSpace(f.Key)
	f.Description = strings.TrimSpace(f.Description)
	if f.Key == "" {
		return fmt.Errorf("%w: feature flag key is required", ErrValidation)
	}
	if f.PercentageRollout < 0 || f.PercentageRollout > 100 {
		return fmt.Errorf("%w: percentage rollout must be between 0 and 100", ErrValidation)
	}
	for _, role := range f.TargetRoles {
		if err := role.Validate(); err != nil {
			return err
		}
	}
	return nil
}
