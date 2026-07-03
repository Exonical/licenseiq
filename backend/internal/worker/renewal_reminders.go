package worker

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var renewalReminderThresholds = []int{90, 60, 30, 14, 7, 1}

const renewalReminderFlagKey = "workers-renewal-reminders"

type messageSender interface {
	Send(context.Context, notify.Message) error
}

type RenewalReminderJob struct {
	interval   time.Duration
	flags      *featureflags.Manager
	licenses   domain.LicenseRepository
	products   domain.ProductRepository
	vendors    domain.VendorRepository
	logs       domain.RenewalReminderLogRepository
	dispatcher messageSender
	logger     *zap.Logger
	clock      func() time.Time
	thresholds []int
}

func NewRenewalReminderJob(interval time.Duration, flags *featureflags.Manager, licenses domain.LicenseRepository, products domain.ProductRepository, vendors domain.VendorRepository, logs domain.RenewalReminderLogRepository, dispatcher messageSender, logger *zap.Logger) *RenewalReminderJob {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RenewalReminderJob{interval: interval, flags: flags, licenses: licenses, products: products, vendors: vendors, logs: logs, dispatcher: dispatcher, logger: logger, clock: time.Now, thresholds: append([]int(nil), renewalReminderThresholds...)}
}

func (j *RenewalReminderJob) Name() string            { return "renewal-reminders" }
func (j *RenewalReminderJob) Interval() time.Duration { return j.interval }

func (j *RenewalReminderJob) Run(ctx context.Context) error {
	if j == nil {
		return nil
	}
	if j.flags != nil && !j.flags.Evaluate(ctx, renewalReminderFlagKey, true) {
		return nil
	}
	now := utcDateOnly(j.clock())
	licenses, err := listLicenses(ctx, j.licenses)
	if err != nil {
		return err
	}
	products, err := listProducts(ctx, j.products)
	if err != nil {
		return err
	}
	vendors, err := listVendors(ctx, j.vendors)
	if err != nil {
		return err
	}
	productByID := make(map[uuid.UUID]domain.Product, len(products))
	for _, p := range products {
		productByID[p.ID] = p
	}
	vendorByID := make(map[uuid.UUID]domain.Vendor, len(vendors))
	for _, v := range vendors {
		vendorByID[v.ID] = v
	}
	for _, lic := range licenses {
		if lic.RenewalDate == nil {
			continue
		}
		renewal := utcDateOnly(*lic.RenewalDate)
		daysUntil := int(renewal.Sub(now).Hours() / 24)
		if daysUntil < 0 {
			continue
		}
		if !containsInt(j.thresholds, daysUntil) {
			continue
		}
		if j.logs != nil {
			if existing, err := j.logs.GetByLicenseThresholdAndRenewalDate(ctx, lic.ID, daysUntil, renewal); err == nil && existing != nil {
				continue
			}
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return err
			}
		}
		prod := productByID[lic.ProductID]
		vend := vendorByID[lic.VendorID]
		msg, err := notify.RenderRenewalReminder(notify.RenewalReminderData{
			VendorName:  strings.TrimSpace(vend.Name),
			ProductName: strings.TrimSpace(prod.Name),
			LicenseName: strings.TrimSpace(lic.LicenseKey),
			RenewalDate: renewal,
			DaysUntil:   daysUntil,
		})
		if err != nil {
			return err
		}
		if j.dispatcher != nil {
			if err := j.dispatcher.Send(ctx, msg); err != nil {
				return err
			}
		}
		if j.logs != nil {
			log := &domain.RenewalReminderLog{LicenseID: lic.ID, ThresholdDays: daysUntil, RenewalDate: renewal, SentAt: j.clock().UTC()}
			if err := j.logs.Create(ctx, log); err != nil {
				return err
			}
		}
		j.logger.Info("sent renewal reminder", zap.String("license_id", lic.ID.String()), zap.Int("days_until", daysUntil))
	}
	return nil
}

func utcDateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func containsInt(values []int, want int) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func listLicenses(ctx context.Context, repo domain.LicenseRepository) ([]domain.License, error) {
	if repo == nil {
		return nil, nil
	}
	return listAll(func(filter domain.ListFilter) ([]domain.License, error) { return repo.List(ctx, filter) })
}
func listProducts(ctx context.Context, repo domain.ProductRepository) ([]domain.Product, error) {
	if repo == nil {
		return nil, nil
	}
	return listAll(func(filter domain.ListFilter) ([]domain.Product, error) { return repo.List(ctx, filter) })
}
func listVendors(ctx context.Context, repo domain.VendorRepository) ([]domain.Vendor, error) {
	if repo == nil {
		return nil, nil
	}
	return listAll(func(filter domain.ListFilter) ([]domain.Vendor, error) { return repo.List(ctx, filter) })
}

func listAll[T any](fetch func(domain.ListFilter) ([]T, error)) ([]T, error) {
	if fetch == nil {
		return nil, nil
	}
	const pageSize = 500
	var out []T
	for offset := 0; ; offset += pageSize {
		items, err := fetch(domain.ListFilter{Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(items) < pageSize {
			return out, nil
		}
	}
}
