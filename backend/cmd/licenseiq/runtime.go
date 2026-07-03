package main

import (
	"context"

	apilayer "github.com/Exonical/licenseiq/backend/internal/api"
	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/featureflags"
	"github.com/Exonical/licenseiq/backend/internal/jira"
	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/Exonical/licenseiq/backend/internal/platform/database/persistence"
	"github.com/Exonical/licenseiq/backend/internal/worker"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type appRuntime struct {
	identitySvc            app.IdentityService
	authManager            *auth.Manager
	featureFlagManager     *featureflags.Manager
	notificationDispatcher *notify.Dispatcher
	jiraSvc                app.JiraService
	services               apilayer.Services
	scheduler              *worker.Scheduler
}

func buildAppRuntime(cfg config.Config, logger *zap.Logger, db *gorm.DB) (*appRuntime, error) {
	auditRepo := persistence.NewAuditRepository(db)
	userRepo := persistence.NewUserRepository(db)
	apiKeyRepo := persistence.NewAPIKeyRepository(db)
	identitySvc := app.NewIdentityService(userRepo, apiKeyRepo, auditRepo)
	authManager, err := auth.NewManager(context.Background(), cfg.Auth, identitySvc, userRepo, apiKeyRepo, logger)
	if err != nil {
		return nil, err
	}
	if plain, err := authManager.Bootstrap(context.Background(), cfg.Auth.Bootstrap); err != nil {
		return nil, err
	} else if plain != "" {
		logger.Warn("bootstrap administrator api key", zap.String("plaintext", plain))
	}
	featureFlagRepo := persistence.NewFeatureFlagRepository(db)
	featureFlagManager, err := featureflags.NewManager(context.Background(), cfg.FeatureFlags, featureFlagRepo, logger)
	if err != nil {
		return nil, err
	}
	notificationDispatcher, err := notify.NewDispatcher(cfg.Notifications)
	if err != nil {
		return nil, err
	}
	jiraClient, err := jira.NewClient(cfg.Jira, logger)
	if err != nil {
		return nil, err
	}
	licenseRepo := persistence.NewLicenseRepository(db)
	productRepo := persistence.NewProductRepository(db)
	vendorRepo := persistence.NewVendorRepository(db)
	attachmentRepo := persistence.NewAttachmentRepository(db)
	linkRepo := persistence.NewLicenseIssueLinkRepository(db)
	assignmentRepo := persistence.NewAssignmentRepository(db)
	reportsSvc := app.NewReportingService(vendorRepo, productRepo, licenseRepo)
	jiraSvc := app.NewJiraService(jiraClient, cfg.Jira.ProjectKey, cfg.Jira.IssueType, licenseRepo, vendorRepo, productRepo, attachmentRepo, linkRepo, auditRepo)
	services := apilayer.Services{
		Vendors:       app.NewVendorService(vendorRepo, auditRepo),
		Products:      app.NewProductService(productRepo, auditRepo),
		Licenses:      app.NewLicenseService(licenseRepo, auditRepo),
		Assignments:   app.NewAssignmentService(assignmentRepo, auditRepo),
		Attachments:   app.NewAttachmentService(attachmentRepo, auditRepo),
		FeatureFlags:  app.NewFeatureFlagService(featureFlagRepo),
		Jira:          jiraSvc,
		Identity:      identitySvc,
		Notifications: notificationDispatcher,
		Reports:       reportsSvc,
	}
	return &appRuntime{
		identitySvc:            identitySvc,
		authManager:            authManager,
		featureFlagManager:     featureFlagManager,
		notificationDispatcher: notificationDispatcher,
		jiraSvc:                jiraSvc,
		services:               services,
		scheduler:              worker.NewScheduler(logger, cfg.Workers.Timeout),
	}, nil
}
