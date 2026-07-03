package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RenewalReminderLog struct {
	Base
	LicenseID     uuid.UUID
	ThresholdDays int
	RenewalDate   time.Time
	SentAt        time.Time
}

func (r *RenewalReminderLog) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: renewal reminder log is nil", ErrValidation)
	}
	if r.LicenseID == uuid.Nil {
		return fmt.Errorf("%w: renewal reminder log license id is required", ErrValidation)
	}
	if r.ThresholdDays <= 0 {
		return fmt.Errorf("%w: renewal reminder log threshold days must be positive", ErrValidation)
	}
	if r.RenewalDate.IsZero() {
		return fmt.Errorf("%w: renewal reminder log renewal date is required", ErrValidation)
	}
	if r.SentAt.IsZero() {
		return fmt.Errorf("%w: renewal reminder log sent at is required", ErrValidation)
	}
	return nil
}
