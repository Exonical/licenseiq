package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Assignment struct {
	Base
	LicenseID  uuid.UUID
	TargetType AssignmentTargetType
	TargetID   string
	TargetName string
	AssignedAt time.Time
	Notes      string
}

func (a *Assignment) Validate() error {
	if a == nil {
		return fmt.Errorf("%w: assignment is nil", ErrValidation)
	}
	a.TargetID = strings.TrimSpace(a.TargetID)
	a.TargetName = strings.TrimSpace(a.TargetName)
	a.Notes = strings.TrimSpace(a.Notes)
	if a.LicenseID == uuid.Nil {
		return fmt.Errorf("%w: assignment license id is required", ErrValidation)
	}
	if err := a.TargetType.Validate(); err != nil {
		return err
	}
	if a.TargetID == "" {
		return fmt.Errorf("%w: assignment target id is required", ErrValidation)
	}
	if a.TargetName == "" {
		return fmt.Errorf("%w: assignment target name is required", ErrValidation)
	}
	if a.AssignedAt.IsZero() {
		return fmt.Errorf("%w: assigned at is required", ErrValidation)
	}
	return nil
}
