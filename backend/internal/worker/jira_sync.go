package worker

import (
	"context"
	"errors"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"go.uber.org/zap"
)

const jiraSyncFlagKey = "jira-sync"

type JiraSyncJob struct {
	interval   time.Duration
	windowDays int
	flags      *featureflags.Manager
	licenses   domain.LicenseRepository
	links      domain.LicenseIssueLinkRepository
	svc        app.JiraService
	logger     *zap.Logger
	clock      func() time.Time
}

func NewJiraSyncJob(interval time.Duration, windowDays int, flags *featureflags.Manager, licenses domain.LicenseRepository, links domain.LicenseIssueLinkRepository, svc app.JiraService, logger *zap.Logger) *JiraSyncJob {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if windowDays <= 0 {
		windowDays = 90
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &JiraSyncJob{interval: interval, windowDays: windowDays, flags: flags, licenses: licenses, links: links, svc: svc, logger: logger, clock: time.Now}
}

func (j *JiraSyncJob) Name() string            { return "jira-sync" }
func (j *JiraSyncJob) Interval() time.Duration { return j.interval }

func (j *JiraSyncJob) Run(ctx context.Context) error {
	if j == nil || j.svc == nil {
		return nil
	}
	if j.flags != nil && !j.flags.Evaluate(ctx, jiraSyncFlagKey, true) {
		return nil
	}
	now := utcDateOnly(j.clock())
	licenses, err := listLicenses(ctx, j.licenses)
	if err != nil {
		return err
	}
	for _, lic := range licenses {
		if lic.RenewalDate == nil {
			continue
		}
		renewal := utcDateOnly(*lic.RenewalDate)
		daysUntil := int(renewal.Sub(now).Hours() / 24)
		if daysUntil < 0 || daysUntil > j.windowDays {
			continue
		}
		if j.links != nil {
			if existing, err := j.links.GetByLicenseAndRenewalDate(ctx, lic.ID, renewal); err == nil && existing != nil {
				continue
			} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return err
			}
		}
		if _, err := j.svc.CreateRenewalTicket(ctx, lic.ID); err != nil {
			if errors.Is(err, app.ErrJiraDisabled) {
				return nil
			}
			return err
		}
		j.logger.Info("created jira renewal ticket", zap.String("license_id", lic.ID.String()), zap.Int("days_until", daysUntil))
	}
	return nil
}
