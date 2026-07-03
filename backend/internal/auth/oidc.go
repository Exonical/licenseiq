package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Token interface {
	Claims(v any) error
	Subject() string
	Issuer() string
	Audience() []string
	Expiry() time.Time
}

type TokenVerifier interface {
	Verify(context.Context, string) (Token, error)
}

type OIDCAuthenticator struct {
	cfg          config.OIDCConfig
	verifier     TokenVerifier
	oauth2Config oauth2.Config
}

func NewOIDCAuthenticator(ctx context.Context, cfg config.OIDCConfig) (*OIDCAuthenticator, error) {
	if cfg.IssuerURL == "" {
		return nil, nil
	}
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	return &OIDCAuthenticator{
		cfg:      cfg,
		verifier: &oidcVerifier{verifier: verifier},
		oauth2Config: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       cfg.ScopesOrDefault(),
		},
	}, nil
}

func (a *OIDCAuthenticator) OAuth2Config() oauth2.Config { return a.oauth2Config }

func (a *OIDCAuthenticator) Authenticate(ctx context.Context, raw string) (*domain.User, error) {
	if a == nil || a.verifier == nil {
		return nil, errors.New("oidc disabled")
	}
	token, err := a.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := token.Claims(&claims); err != nil {
		return nil, err
	}
	if a.cfg.Audience != "" && !claimContainsAudience(claims["aud"], a.cfg.Audience) {
		return nil, fmt.Errorf("audience mismatch")
	}
	role := a.cfg.DefaultRole
	if mapped := roleFromClaim(claims[a.cfg.RoleClaim], a.cfg.RoleMappings); mapped != "" {
		role = mapped
	}
	email := claimString(claims, "email")
	subject := strings.TrimSpace(token.Subject())
	if subject == "" {
		subject = email
	}
	if email == "" {
		email = subject
	}
	if email == "" {
		return nil, fmt.Errorf("missing identity claims")
	}
	return &domain.User{
		Email:           email,
		DisplayName:     firstNonEmpty(claimString(claims, "name"), claimString(claims, "preferred_username"), email),
		ExternalSubject: subject,
		Role:            role,
		Active:          true,
	}, nil
}

type oidcVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (v *oidcVerifier) Verify(ctx context.Context, raw string) (Token, error) {
	token, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	return &oidcToken{token: token}, nil
}

type oidcToken struct{ token *oidc.IDToken }

func (t *oidcToken) Claims(v any) error { return t.token.Claims(v) }
func (t *oidcToken) Subject() string    { return t.token.Subject }
func (t *oidcToken) Issuer() string     { return t.token.Issuer }
func (t *oidcToken) Audience() []string { return append([]string(nil), t.token.Audience...) }
func (t *oidcToken) Expiry() time.Time  { return t.token.Expiry }

func roleFromClaim(value any, mappings map[string]domain.Role) domain.Role {
	switch v := value.(type) {
	case string:
		if role, ok := mappings[v]; ok {
			return role
		}
	case []any:
		for _, item := range v {
			if role, ok := mappings[fmt.Sprint(item)]; ok {
				return role
			}
		}
	case []string:
		for _, item := range v {
			if role, ok := mappings[item]; ok {
				return role
			}
		}
	}
	return ""
}

func claimContainsAudience(value any, audience string) bool {
	switch v := value.(type) {
	case string:
		return v == audience
	case []any:
		for _, item := range v {
			if fmt.Sprint(item) == audience {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if item == audience {
				return true
			}
		}
	}
	return false
}

func claimString(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
