package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	Base
	OwnerType        AttachmentOwnerType
	OwnerID          uuid.UUID
	Filename         string
	ContentType      string
	SizeBytes        int64
	StorageKey       string
	UploadedByUserID *uuid.UUID
	UploadedAt       time.Time
}

func (a *Attachment) Validate() error {
	if a == nil {
		return fmt.Errorf("%w: attachment is nil", ErrValidation)
	}
	a.Filename = strings.TrimSpace(a.Filename)
	a.ContentType = strings.TrimSpace(a.ContentType)
	a.StorageKey = strings.TrimSpace(a.StorageKey)
	if err := a.OwnerType.Validate(); err != nil {
		return err
	}
	if a.OwnerID == uuid.Nil {
		return fmt.Errorf("%w: attachment owner id is required", ErrValidation)
	}
	if a.Filename == "" {
		return fmt.Errorf("%w: filename is required", ErrValidation)
	}
	if a.StorageKey == "" {
		return fmt.Errorf("%w: storage key is required", ErrValidation)
	}
	if a.UploadedAt.IsZero() {
		return fmt.Errorf("%w: uploaded at is required", ErrValidation)
	}
	if a.SizeBytes < 0 {
		return fmt.Errorf("%w: size bytes cannot be negative", ErrValidation)
	}
	return nil
}
