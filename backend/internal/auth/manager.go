package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"
)

type Manager struct {
	identity app.IdentityService
	users    domain.UserRepository
	keys     domain.APIKeyRepository
	oidc     *OIDCAuthenticator
	logger   *zap.Logger
}

func NewManager(ctx context.Context, cfg config.AuthConfig, identity app.IdentityService, users domain.UserRepository, keys domain.APIKeyRepository, logger *zap.Logger) (*Manager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	oidc, err := NewOIDCAuthenticator(ctx, cfg.OIDC)
	if err != nil {
		return nil, err
	}
	return &Manager{identity: identity, users: users, keys: keys, oidc: oidc, logger: logger}, nil
}

func (m *Manager) Middleware() func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		principal, err := m.authenticate(ctx.Context(), ctx.Header("Authorization"), ctx.Header("X-API-Key"))
		if err != nil {
			writeAuthError(ctx, httpStatusForAuthError(err), authMessageForError(err))
			return
		}
		requiredRole, ok := requiredRoleForOperation(ctx.Operation())
		if !ok {
			m.logger.Error("operation missing auth metadata", zap.String("method", ctx.Method()), zap.String("path", ctx.URL().Path))
			writeAuthError(ctx, 500, "internal server error")
			return
		}
		if !Allows(principal.Role, requiredRole) {
			writeAuthError(ctx, 403, "forbidden")
			return
		}
		reqCtx := app.WithRequestContext(ctx.Context(), RequestContext(*principal, ctx.RemoteAddr(), ctx.Header("X-Request-ID")))
		reqCtx = WithPrincipal(reqCtx, *principal)
		next(huma.WithContext(ctx, reqCtx))
	}
}

func requiredRoleForOperation(op *huma.Operation) (domain.Role, bool) {
	if op == nil || op.Metadata == nil {
		return domain.RoleViewer, false
	}
	resource, _ := op.Metadata["resource"].(string)
	action, _ := op.Metadata["action"].(string)
	if resource == "" || action == "" {
		return domain.RoleViewer, false
	}
	return RequiredRole(resource, action)
}

func (m *Manager) authenticate(ctx context.Context, authorization, apiKeyHeader string) (*Principal, error) {
	if token := firstNonEmpty(apiKeyHeader, bearerAPIToken(authorization)); token != "" {
		return m.authenticateAPIKey(ctx, token)
	}
	if token := bearerToken(authorization); token != "" {
		return m.authenticateOIDC(ctx, token)
	}
	return nil, errUnauthorized
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func bearerAPIToken(header string) string {
	token := bearerToken(header)
	if token == "" || !strings.HasPrefix(token, "liq_") {
		return ""
	}
	return token
}

func (m *Manager) authenticateAPIKey(ctx context.Context, token string) (*Principal, error) {
	keyID, _, ok := app.APIKeyTokenParts(token)
	if !ok {
		return nil, errUnauthorized
	}
	stored, err := m.keys.GetByKeyID(ctx, keyID)
	if err != nil {
		return nil, errUnauthorized
	}
	if stored.ExpiresAt != nil && time.Now().UTC().After(*stored.ExpiresAt) {
		return nil, errUnauthorized
	}
	if !app.APIKeyHashMatches(stored.HashedKey, token) {
		return nil, errUnauthorized
	}
	user, err := m.identity.GetServiceAccount(ctx, stored.OwnerUserID)
	if err != nil || !user.Active {
		return nil, errUnauthorized
	}
	now := time.Now().UTC()
	stored.LastUsedAt = &now
	_ = m.keys.Update(ctx, stored)
	return &Principal{UserID: &user.ID, APIKeyID: &stored.ID, Role: user.Role, Email: user.Email, ExternalSubject: user.ExternalSubject, IsServiceAccount: true}, nil
}

func (m *Manager) authenticateOIDC(ctx context.Context, token string) (*Principal, error) {
	if m.oidc == nil {
		return nil, errUnauthorized
	}
	user, err := m.oidc.Authenticate(ctx, token)
	if err != nil {
		return nil, errUnauthorized
	}
	upserted, err := m.identity.UpsertAuthenticatedUser(ctx, *user)
	if err != nil {
		return nil, errUnauthorized
	}
	return &Principal{UserID: &upserted.ID, Role: upserted.Role, Email: upserted.Email, ExternalSubject: upserted.ExternalSubject}, nil
}

func writeAuthError(ctx huma.Context, status int, message string) {
	ctx.SetStatus(status)
	ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	_, _ = fmt.Fprintf(ctx.BodyWriter(), `{"message":%q}`, message)
}

var errUnauthorized = errors.New("unauthorized")

func httpStatusForAuthError(err error) int {
	if errors.Is(err, errUnauthorized) {
		return 401
	}
	return 500
}

func authMessageForError(err error) string {
	if errors.Is(err, errUnauthorized) {
		return "unauthorized"
	}
	return "internal server error"
}

func (m *Manager) Bootstrap(ctx context.Context, cfg config.BootstrapConfig) (string, error) {
	if strings.TrimSpace(cfg.AdminEmail) == "" {
		return "", nil
	}
	email := strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	user, err := m.users.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}
	if err == nil {
		user.IsServiceAccount = true
		user.Active = true
		user.Role = domain.RoleAdministrator
		if strings.TrimSpace(cfg.AdminDisplayName) != "" {
			user.DisplayName = cfg.AdminDisplayName
		}
		if err := m.users.Update(ctx, user); err != nil {
			return "", err
		}
	} else {
		created, err := m.identity.CreateServiceAccount(ctx, domain.User{Email: email, DisplayName: cfg.AdminDisplayName, Role: domain.RoleAdministrator, IsServiceAccount: true, Active: true})
		if err != nil {
			return "", err
		}
		user = created
	}
	keys, err := m.identity.ListAPIKeys(ctx, user.ID, domain.ListFilter{Limit: 500})
	if err != nil {
		return "", err
	}
	if len(keys) > 0 {
		return "", nil
	}
	keyName := strings.TrimSpace(cfg.AdminAPIKeyName)
	if keyName == "" {
		keyName = "bootstrap-admin"
	}
	if strings.TrimSpace(cfg.AdminAPIKey) != "" {
		_, err = m.identity.CreateAPIKeyWithToken(ctx, domain.APIKey{OwnerUserID: user.ID, Name: keyName, Scopes: []string{"*"}}, cfg.AdminAPIKey)
		return cfg.AdminAPIKey, err
	}
	_, plain, err := m.identity.CreateAPIKey(ctx, domain.APIKey{OwnerUserID: user.ID, Name: keyName, Scopes: []string{"*"}})
	return plain, err
}
