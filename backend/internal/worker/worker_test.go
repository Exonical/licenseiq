package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type fakeNotifier struct {
	mu       sync.Mutex
	messages []notify.Message
	err      error
}

func (f *fakeNotifier) Send(_ context.Context, msg notify.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msg)
	return f.err
}

func (f *fakeNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.messages)
}

type fakeLicenseRepo struct{ licenses []domain.License }

func (f fakeLicenseRepo) Create(context.Context, *domain.License) error { return nil }
func (f fakeLicenseRepo) Get(context.Context, uuid.UUID) (*domain.License, error) {
	return nil, domain.ErrNotFound
}
func (f fakeLicenseRepo) Update(context.Context, *domain.License) error { return nil }
func (f fakeLicenseRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f fakeLicenseRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.License, error) {
	return append([]domain.License(nil), f.licenses...), nil
}

type fakeProductRepo struct{ products []domain.Product }

func (f fakeProductRepo) Create(context.Context, *domain.Product) error { return nil }
func (f fakeProductRepo) Get(context.Context, uuid.UUID) (*domain.Product, error) {
	return nil, domain.ErrNotFound
}
func (f fakeProductRepo) Update(context.Context, *domain.Product) error { return nil }
func (f fakeProductRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (f fakeProductRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.Product, error) {
	return append([]domain.Product(nil), f.products...), nil
}

type fakeVendorRepo struct{ vendors []domain.Vendor }

func (f fakeVendorRepo) Create(context.Context, *domain.Vendor) error { return nil }
func (f fakeVendorRepo) Get(context.Context, uuid.UUID) (*domain.Vendor, error) {
	return nil, domain.ErrNotFound
}
func (f fakeVendorRepo) Update(context.Context, *domain.Vendor) error { return nil }
func (f fakeVendorRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (f fakeVendorRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.Vendor, error) {
	return append([]domain.Vendor(nil), f.vendors...), nil
}

type fakeAPIKeyRepo struct {
	keys    []domain.APIKey
	updates []*domain.APIKey
}

func (f fakeAPIKeyRepo) Create(context.Context, *domain.APIKey) error { return nil }
func (f fakeAPIKeyRepo) Get(context.Context, uuid.UUID) (*domain.APIKey, error) {
	return nil, domain.ErrNotFound
}
func (f fakeAPIKeyRepo) GetByKeyID(context.Context, string) (*domain.APIKey, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeAPIKeyRepo) Update(_ context.Context, key *domain.APIKey) error {
	copied := *key
	f.updates = append(f.updates, &copied)
	return nil
}
func (f fakeAPIKeyRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f fakeAPIKeyRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.APIKey, error) {
	return append([]domain.APIKey(nil), f.keys...), nil
}

type fakeReminderLogRepo struct {
	logs    map[string]domain.RenewalReminderLog
	creates []*domain.RenewalReminderLog
}

func (f *fakeReminderLogRepo) key(licenseID uuid.UUID, threshold int, renewalDate time.Time) string {
	return fmt.Sprintf("%s|%s|%d", licenseID.String(), time.Date(renewalDate.Year(), renewalDate.Month(), renewalDate.Day(), 0, 0, 0, 0, time.UTC).Format(time.DateOnly), threshold)
}
func (f *fakeReminderLogRepo) Create(_ context.Context, log *domain.RenewalReminderLog) error {
	copied := *log
	f.creates = append(f.creates, &copied)
	if f.logs == nil {
		f.logs = map[string]domain.RenewalReminderLog{}
	}
	f.logs[f.key(log.LicenseID, log.ThresholdDays, log.RenewalDate)] = copied
	return nil
}
func (f *fakeReminderLogRepo) Get(context.Context, uuid.UUID) (*domain.RenewalReminderLog, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeReminderLogRepo) GetByLicenseThresholdAndRenewalDate(_ context.Context, licenseID uuid.UUID, threshold int, renewalDate time.Time) (*domain.RenewalReminderLog, error) {
	if f.logs == nil {
		return nil, domain.ErrNotFound
	}
	if log, ok := f.logs[f.key(licenseID, threshold, renewalDate)]; ok {
		copied := log
		return &copied, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeReminderLogRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.RenewalReminderLog, error) {
	out := make([]domain.RenewalReminderLog, 0, len(f.logs))
	for _, v := range f.logs {
		out = append(out, v)
	}
	return out, nil
}

type fakeReminderDispatcher struct{ notifier *fakeNotifier }

func (f fakeReminderDispatcher) Send(ctx context.Context, msg notify.Message) error {
	return f.notifier.Send(ctx, msg)
}

func TestRenewalReminderJobThresholdsAndDedup(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	thresholds := []int{90, 60, 30, 14, 7, 1}
	for _, threshold := range thresholds {
		t.Run(time.Duration(threshold).String(), func(t *testing.T) {
			licID := uuid.New()
			vendorID := uuid.New()
			productID := uuid.New()
			renewal := now.AddDate(0, 0, threshold)
			notifier := &fakeNotifier{}
			logs := &fakeReminderLogRepo{}
			job := &RenewalReminderJob{
				flags:      nil,
				licenses:   fakeLicenseRepo{licenses: []domain.License{{Base: domain.Base{ID: licID}, VendorID: vendorID, ProductID: productID, RenewalDate: &renewal, LicenseKey: "LIC-1", Cost: decimal.RequireFromString("12.00"), Currency: "USD"}}},
				products:   fakeProductRepo{products: []domain.Product{{Base: domain.Base{ID: productID}, Name: "Product"}}},
				vendors:    fakeVendorRepo{vendors: []domain.Vendor{{Base: domain.Base{ID: vendorID}, Name: "Vendor"}}},
				logs:       logs,
				dispatcher: fakeReminderDispatcher{notifier: notifier},
				logger:     zap.NewNop(),
				clock:      func() time.Time { return now },
				thresholds: []int{threshold},
			}
			if err := job.Run(context.Background()); err != nil {
				t.Fatalf("run: %v", err)
			}
			if notifier.count() != 1 {
				t.Fatalf("expected one reminder, got %d", notifier.count())
			}
			if len(logs.creates) != 1 {
				t.Fatalf("expected one log row, got %d", len(logs.creates))
			}
		})
	}
}

func TestRenewalReminderJobDedupAndNextCycle(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	licID := uuid.New()
	vendorID := uuid.New()
	productID := uuid.New()
	firstRenewal := now.AddDate(0, 0, 30)
	secondRenewal := now.AddDate(1, 0, 0).AddDate(0, 0, 30)
	notifier := &fakeNotifier{}
	logs := &fakeReminderLogRepo{}
	job := &RenewalReminderJob{
		licenses:   fakeLicenseRepo{licenses: []domain.License{{Base: domain.Base{ID: licID}, VendorID: vendorID, ProductID: productID, RenewalDate: &firstRenewal, LicenseKey: "LIC-1"}}},
		products:   fakeProductRepo{products: []domain.Product{{Base: domain.Base{ID: productID}, Name: "Product"}}},
		vendors:    fakeVendorRepo{vendors: []domain.Vendor{{Base: domain.Base{ID: vendorID}, Name: "Vendor"}}},
		logs:       logs,
		dispatcher: fakeReminderDispatcher{notifier: notifier},
		logger:     zap.NewNop(),
		clock:      func() time.Time { return now },
		thresholds: []int{30},
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if notifier.count() != 1 {
		t.Fatalf("expected 1 send, got %d", notifier.count())
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("dedup run: %v", err)
	}
	if notifier.count() != 1 {
		t.Fatalf("expected no duplicate send, got %d", notifier.count())
	}
	job.licenses = fakeLicenseRepo{licenses: []domain.License{{Base: domain.Base{ID: licID}, VendorID: vendorID, ProductID: productID, RenewalDate: &secondRenewal, LicenseKey: "LIC-1"}}}
	job.clock = func() time.Time { return now.AddDate(1, 0, 0) }
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("next cycle: %v", err)
	}
	if notifier.count() != 2 {
		t.Fatalf("expected second-cycle send, got %d", notifier.count())
	}
}

func TestRenewalReminderJobOffThresholdDoesNotSend(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	renewal := now.AddDate(0, 0, 2)
	notifier := &fakeNotifier{}
	job := &RenewalReminderJob{
		licenses:   fakeLicenseRepo{licenses: []domain.License{{Base: domain.Base{ID: uuid.New()}, VendorID: uuid.New(), ProductID: uuid.New(), RenewalDate: &renewal, LicenseKey: "LIC-1"}}},
		products:   fakeProductRepo{},
		vendors:    fakeVendorRepo{},
		logs:       &fakeReminderLogRepo{},
		dispatcher: fakeReminderDispatcher{notifier: notifier},
		logger:     zap.NewNop(),
		clock:      func() time.Time { return now },
		thresholds: []int{30},
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if notifier.count() != 0 {
		t.Fatalf("expected no send, got %d", notifier.count())
	}
}

func TestMaintenanceJobDeactivatesExpiredKeys(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	repo := &fakeAPIKeyRepo{keys: []domain.APIKey{{Base: domain.Base{ID: uuid.New()}, Active: true, ExpiresAt: &expired}, {Base: domain.Base{ID: uuid.New()}, Active: true, ExpiresAt: &future}, {Base: domain.Base{ID: uuid.New()}, Active: false, ExpiresAt: &expired}}}
	job := &MaintenanceJob{keys: repo, logger: zap.NewNop(), clock: func() time.Time { return now }}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(repo.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(repo.updates))
	}
	if repo.updates[0].Active {
		t.Fatalf("expected expired key to be inactive")
	}
}

func TestSchedulerRunsJobsAndRecoversPanics(t *testing.T) {
	t.Run("runs", func(t *testing.T) {
		var mu sync.Mutex
		count := 0
		job := jobFunc{name: "tick", interval: 10 * time.Millisecond, fn: func(context.Context) error { mu.Lock(); count++; mu.Unlock(); return nil }}
		s := NewScheduler(zap.NewNop(), 100*time.Millisecond)
		s.Register(job)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- s.Start(ctx) }()
		time.Sleep(50 * time.Millisecond)
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled, got %v", err)
		}
		mu.Lock()
		got := count
		mu.Unlock()
		if got == 0 {
			t.Fatalf("expected job to run")
		}
	})

	t.Run("panic contained", func(t *testing.T) {
		job := jobFunc{name: "panic", interval: 10 * time.Millisecond, fn: func(context.Context) error { panic("boom") }}
		s := NewScheduler(zap.NewNop(), 100*time.Millisecond)
		s.Register(job)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- s.Start(ctx) }()
		time.Sleep(30 * time.Millisecond)
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled, got %v", err)
		}
	})
}

type jobFunc struct {
	name     string
	interval time.Duration
	fn       func(context.Context) error
}

func (j jobFunc) Name() string                  { return j.name }
func (j jobFunc) Interval() time.Duration       { return j.interval }
func (j jobFunc) Run(ctx context.Context) error { return j.fn(ctx) }
