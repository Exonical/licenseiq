package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	licmcp "github.com/Exonical/licenseiq/backend/internal/mcp"
	"github.com/Exonical/licenseiq/backend/internal/platform/database"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const mcpServerFlagKey = "mcp-server"

func runMCP(cfg config.Config, logger *zap.Logger) error {
	if !cfg.MCP.Enabled {
		return fmt.Errorf("mcp server is disabled")
	}
	db, err := openDatabaseWithRetry(cfg.Postgres, 5, 2*time.Second)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(db); err != nil {
			logger.Warn("close database", zap.Error(err))
		}
	}()
	runtime, err := buildAppRuntime(cfg, logger, db)
	if err != nil {
		return err
	}
	defer runtime.featureFlagManager.Close()
	actor, err := resolveMCPActor(context.Background(), cfg, runtime.identitySvc)
	if err != nil {
		return err
	}
	if !runtime.featureFlagManager.Evaluate(context.Background(), mcpServerFlagKey, false) {
		return fmt.Errorf("mcp server feature flag is disabled")
	}
	server := licmcp.NewServer(licmcp.Dependencies{
		Licenses:    runtime.services.Licenses,
		Vendors:     runtime.services.Vendors,
		Products:    runtime.services.Products,
		Assignments: runtime.services.Assignments,
		Reports:     runtime.services.Reports,
		Jira:        runtime.services.Jira,
	}, licmcp.Options{
		Principal:         actor,
		SessionID:         "mcp",
		DashboardResource: "licenseiq://dashboard/summary",
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Info("mcp server starting")
	if err := server.Run(ctx, &sdkmcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("mcp server stopped")
	return nil
}

func resolveMCPActor(ctx context.Context, cfg config.Config, identity appIdentityService) (auth.Principal, error) {
	email := strings.TrimSpace(cfg.MCP.ServiceAccountEmail)
	if email == "" {
		email = strings.TrimSpace(cfg.Auth.Bootstrap.AdminEmail)
	}
	if email == "" {
		return auth.Principal{}, fmt.Errorf("mcp service account email is required")
	}
	users, err := identity.ListServiceAccounts(ctx, domain.ListFilter{Limit: 500})
	if err != nil {
		return auth.Principal{}, err
	}
	for i := range users {
		user := users[i]
		if strings.EqualFold(strings.TrimSpace(user.Email), email) {
			return auth.Principal{
				UserID:           &user.ID,
				Role:             domain.RoleAdministrator,
				Email:            user.Email,
				IsServiceAccount: true,
			}, nil
		}
	}
	if cfg.MCP.ServiceAccountEmail == "" {
		return auth.Principal{}, fmt.Errorf("mcp service account email %q not found", email)
	}
	created, err := identity.CreateServiceAccount(ctx, domain.User{
		Email:            email,
		DisplayName:      strings.TrimSpace(cfg.MCP.ServiceAccountDisplayName),
		Role:             domain.RoleAdministrator,
		IsServiceAccount: true,
		Active:           true,
	})
	if err != nil {
		return auth.Principal{}, err
	}
	return auth.Principal{
		UserID:           &created.ID,
		Role:             domain.RoleAdministrator,
		Email:            created.Email,
		IsServiceAccount: true,
	}, nil
}

type appIdentityService interface {
	ListServiceAccounts(context.Context, domain.ListFilter) ([]domain.User, error)
	CreateServiceAccount(context.Context, domain.User) (*domain.User, error)
}
