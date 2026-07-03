package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apilayer "github.com/Exonical/licenseiq/backend/internal/api"
	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"github.com/Exonical/licenseiq/backend/internal/logging"
	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/Exonical/licenseiq/backend/internal/platform/cache"
	"github.com/Exonical/licenseiq/backend/internal/platform/database"
	"github.com/Exonical/licenseiq/backend/internal/platform/database/persistence"
	"github.com/Exonical/licenseiq/backend/internal/server"
	"github.com/Exonical/licenseiq/backend/internal/telemetry"
	"github.com/Exonical/licenseiq/backend/internal/version"
	"github.com/Exonical/licenseiq/backend/internal/worker"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	logger, err := logging.New(cfg.Log.Level, cfg.Log.Dev)
	if err != nil {
		log.Fatalf("build logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrations(cfg, logger); err != nil {
			logger.Fatal("run migrations", zap.Error(err))
		}
		return
	}

	if err := runServer(cfg, logger); err != nil {
		logger.Fatal("run server", zap.Error(err))
	}
}

func runMigrations(cfg config.Config, logger *zap.Logger) error {
	db, err := openDatabaseWithRetry(cfg.Postgres, 5, 2*time.Second)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(db); err != nil {
			logger.Warn("close database", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	return persistence.Migrate(ctx, db)
}

func runServer(cfg config.Config, logger *zap.Logger) error {
	shutdownTelemetry, err := telemetry.New(context.Background(), telemetry.Config{
		Endpoint:    cfg.OTel.Endpoint,
		ServiceName: cfg.OTel.ServiceName,
	})
	if err != nil {
		logger.Fatal("initialize telemetry", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownTelemetry != nil {
			if err := shutdownTelemetry(ctx); err != nil {
				logger.Warn("shutdown telemetry", zap.Error(err))
			}
		}
	}()

	db, err := openDatabaseWithRetry(cfg.Postgres, 5, 2*time.Second)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(db); err != nil {
			logger.Warn("close database", zap.Error(err))
		}
	}()

	valkeyClient, cacheErr := cache.New(cfg.Valkey)
	if cacheErr != nil {
		logger.Warn("connect cache", zap.Error(cacheErr))
	}
	if valkeyClient != nil {
		defer func() {
			if err := valkeyClient.Close(); err != nil {
				logger.Warn("close cache", zap.Error(err))
			}
		}()
	}

	engine := server.NewEngine(server.Options{
		Logger:      logger,
		ServiceName: cfg.OTel.ServiceName,
		StartedAt:   time.Now().UTC(),
		Version:     version.Current(),
		Checkers: func() []server.HealthChecker {
			checkers := []server.HealthChecker{database.Checker{DB: db}}
			if valkeyClient != nil {
				checkers = append(checkers, cache.Checker{Cache: valkeyClient})
			}
			return checkers
		}(),
	})

	humaAPI := humagin.New(engine, apilayer.NewHumaConfig("LicenseIQ API", version.Current().Version))
	auditRepo := persistence.NewAuditRepository(db)
	userRepo := persistence.NewUserRepository(db)
	apiKeyRepo := persistence.NewAPIKeyRepository(db)
	identitySvc := app.NewIdentityService(userRepo, apiKeyRepo, auditRepo)
	authManager, err := auth.NewManager(context.Background(), cfg.Auth, identitySvc, userRepo, apiKeyRepo, logger)
	if err != nil {
		return err
	}
	if plain, err := authManager.Bootstrap(context.Background(), cfg.Auth.Bootstrap); err != nil {
		return err
	} else if plain != "" {
		logger.Warn("bootstrap administrator api key", zap.String("plaintext", plain))
	}
	featureFlagRepo := persistence.NewFeatureFlagRepository(db)
	featureFlagManager, err := featureflags.NewManager(context.Background(), cfg.FeatureFlags, featureFlagRepo, logger)
	if err != nil {
		return err
	}
	defer featureFlagManager.Close()

	notificationDispatcher, err := notify.NewDispatcher(cfg.Notifications)
	if err != nil {
		return err
	}
	scheduler := worker.NewScheduler(logger, cfg.Workers.Timeout)
	if cfg.Workers.Enabled {
		if cfg.Workers.Renewals.Enabled {
			scheduler.Register(worker.NewRenewalReminderJob(cfg.Workers.Renewals.Interval, featureFlagManager, persistence.NewLicenseRepository(db), persistence.NewProductRepository(db), persistence.NewVendorRepository(db), persistence.NewRenewalReminderLogRepository(db), notificationDispatcher, logger))
		}
		if cfg.Workers.Maintenance.Enabled {
			scheduler.Register(worker.NewMaintenanceJob(cfg.Workers.Maintenance.Interval, featureFlagManager, apiKeyRepo, logger))
		}
	}
	services := apilayer.Services{
		Vendors:       app.NewVendorService(persistence.NewVendorRepository(db), auditRepo),
		Products:      app.NewProductService(persistence.NewProductRepository(db), auditRepo),
		Licenses:      app.NewLicenseService(persistence.NewLicenseRepository(db), auditRepo),
		Assignments:   app.NewAssignmentService(persistence.NewAssignmentRepository(db), auditRepo),
		Attachments:   app.NewAttachmentService(persistence.NewAttachmentRepository(db), auditRepo),
		FeatureFlags:  app.NewFeatureFlagService(featureFlagRepo),
		Identity:      identitySvc,
		Notifications: notificationDispatcher,
	}
	apilayer.RegisterRoutes(humaAPI, services, logger, authManager, featureFlagManager)
	apilayer.MountOpenAPI(engine, humaAPI)
	apilayer.MountDocs(engine)

	httpServer := &http.Server{
		Addr:           cfg.HTTP.Addr,
		Handler:        engine,
		ReadTimeout:    cfg.HTTP.ReadTimeout,
		WriteTimeout:   cfg.HTTP.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", zap.String("addr", cfg.HTTP.Addr))
		serverErr <- httpServer.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	workerCtx, workerCancel := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	defer workerCancel()
	if cfg.Workers.Enabled {
		go func() {
			defer close(workerDone)
			if err := scheduler.Start(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("worker scheduler stopped", zap.Error(err))
			}
		}()
	}

	select {
	case <-ctx.Done():
		workerCancel()
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server exited", zap.Error(err))
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}
	workerCancel()
	if cfg.Workers.Enabled {
		select {
		case <-workerDone:
		case <-time.After(cfg.HTTP.ShutdownTimeout):
			logger.Warn("worker scheduler shutdown timed out")
		}
	}

	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("server stopped")
	return nil
}

func openDatabaseWithRetry(cfg config.PostgresConfig, attempts int, delay time.Duration) (*gorm.DB, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		db, err := database.Open(cfg)
		if err == nil {
			return db, nil
		}
		lastErr = err
		if i < attempts {
			time.Sleep(delay)
		}
	}
	return nil, lastErr
}
