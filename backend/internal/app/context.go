package app

import (
	"context"

	"github.com/google/uuid"
)

type RequestContext struct {
	ActorUserID   *uuid.UUID
	ActorAPIKeyID *uuid.UUID
	IPAddress     string
	SessionID     string
}

type requestContextKey struct{}

func WithRequestContext(ctx context.Context, value RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey{}, value)
}

func RequestContextFromContext(ctx context.Context) RequestContext {
	if ctx == nil {
		return RequestContext{}
	}
	value, _ := ctx.Value(requestContextKey{}).(RequestContext)
	return value
}
