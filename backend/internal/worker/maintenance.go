package worker

import (
	"context"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"go.uber.org/zap"
)

const maintenanceFlagKey = "workers-maintenance"

type MaintenanceJob struct {
	interval time.Duration
	flags    *featureflags.Manager
	keys     domain.APIKeyRepository
	logger   *zap.Logger
	clock    func() time.Time
}

func NewMaintenanceJob(interval time.Duration, flags *featureflags.Manager, keys domain.APIKeyRepository, logger *zap.Logger) *MaintenanceJob {
	if interval <= 0 {
		interval = time.Hour
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &MaintenanceJob{interval: interval, flags: flags, keys: keys, logger: logger, clock: time.Now}
}

func (j *MaintenanceJob) Name() string            { return "maintenance" }
func (j *MaintenanceJob) Interval() time.Duration { return j.interval }

func (j *MaintenanceJob) Run(ctx context.Context) error {
	if j == nil {
		return nil
	}
	if j.logger == nil {
		j.logger = zap.NewNop()
	}
	if j.flags != nil && !j.flags.Evaluate(ctx, maintenanceFlagKey, true) {
		return nil
	}
	keys, err := listAPIKeys(ctx, j.keys)
	if err != nil {
		return err
	}
	now := j.clock().UTC()
	for i := range keys {
		key := &keys[i]
		if !key.Active || key.ExpiresAt == nil || !now.After(key.ExpiresAt.UTC()) {
			continue
		}
		key.Active = false
		if err := j.keys.Update(ctx, key); err != nil {
			return err
		}
		j.logger.Info("deactivated expired api key", zap.String("api_key_id", key.ID.String()))
	}
	return nil
}

func listAPIKeys(ctx context.Context, repo domain.APIKeyRepository) ([]domain.APIKey, error) {
	if repo == nil {
		return nil, nil
	}
	return listAll(func(filter domain.ListFilter) ([]domain.APIKey, error) { return repo.List(ctx, filter) })
}
