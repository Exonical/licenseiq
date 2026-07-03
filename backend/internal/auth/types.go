package auth

import (
	"context"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

type Principal struct {
	UserID           *uuid.UUID
	APIKeyID         *uuid.UUID
	Role             domain.Role
	Email            string
	ExternalSubject  string
	IsServiceAccount bool
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

func RequestContext(principal Principal, ip, session string) app.RequestContext {
	return app.RequestContext{
		ActorUserID:   principal.UserID,
		ActorAPIKeyID: principal.APIKeyID,
		IPAddress:     ip,
		SessionID:     session,
	}
}
