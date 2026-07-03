package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	Base
	Email            string
	DisplayName      string
	ExternalSubject  string
	Role             Role
	IsServiceAccount bool
	Active           bool
}

type APIKey struct {
	Base
	OwnerUserID uuid.UUID
	Name        string
	HashedKey   string
	Scopes      []string
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
}

func (u *User) Validate() error {
	if u == nil {
		return fmt.Errorf("%w: user is nil", ErrValidation)
	}
	u.Email = strings.TrimSpace(strings.ToLower(u.Email))
	u.DisplayName = strings.TrimSpace(u.DisplayName)
	u.ExternalSubject = strings.TrimSpace(u.ExternalSubject)
	if u.Email == "" {
		return fmt.Errorf("%w: email is required", ErrValidation)
	}
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return fmt.Errorf("%w: invalid email: %v", ErrValidation, err)
	}
	if err := u.Role.Validate(); err != nil {
		return err
	}
	return nil
}

func (a *APIKey) Validate() error {
	if a == nil {
		return fmt.Errorf("%w: api key is nil", ErrValidation)
	}
	a.Name = strings.TrimSpace(a.Name)
	a.HashedKey = strings.TrimSpace(a.HashedKey)
	a.Scopes = trimStrings(a.Scopes)
	if a.OwnerUserID == uuid.Nil {
		return fmt.Errorf("%w: api key owner user id is required", ErrValidation)
	}
	if a.Name == "" {
		return fmt.Errorf("%w: api key name is required", ErrValidation)
	}
	if a.HashedKey == "" {
		return fmt.Errorf("%w: api key hash is required", ErrValidation)
	}
	return nil
}
