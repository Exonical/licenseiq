package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	Base
	ActorUserID    *uuid.UUID
	ActorAPIKeyID  *uuid.UUID
	Action         AuditAction
	EntityType     string
	EntityID       uuid.UUID
	PreviousValues json.RawMessage
	NewValues      json.RawMessage
	IPAddress      string
	SessionID      string
}

func (a *AuditLog) Validate() error {
	if a == nil {
		return fmt.Errorf("%w: audit log is nil", ErrValidation)
	}
	a.EntityType = strings.TrimSpace(a.EntityType)
	a.IPAddress = strings.TrimSpace(a.IPAddress)
	a.SessionID = strings.TrimSpace(a.SessionID)
	if err := a.Action.Validate(); err != nil {
		return err
	}
	if a.EntityType == "" {
		return fmt.Errorf("%w: entity type is required", ErrValidation)
	}
	if a.EntityID == uuid.Nil {
		return fmt.Errorf("%w: entity id is required", ErrValidation)
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return nil
}
