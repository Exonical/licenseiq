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

	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/logging"
	"github.com/Exonical/licenseiq/backend/internal/platform/cache"
	"github.com/Exonical/licenseiq/backend/internal/platform/database"
	"github.com/Exonical/licenseiq/backend/internal/server"
	"github.com/Exonical/licenseiq/backend/internal/telemetry"
	"github.com/Exonical/licenseiq/backend/internal/version"
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
		logger.Fatal("connect database", zap.Error(err))
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
		Checkers: []server.HealthChecker{
			database.Checker{DB: db},
			cache.Checker{Cache: valkeyClient},
		},
	})

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

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server exited", zap.Error(err))
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("shutdown server", zap.Error(err))
	}

	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal("server exited", zap.Error(err))
	}
	logger.Info("server stopped")
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
