package domain

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type LicenseIssueLink struct {
	Base
	LicenseID   uuid.UUID
	IssueKey    string
	IssueURL    string
	Status      string
	RenewalDate *time.Time
}

func (l *LicenseIssueLink) Validate() error {
	if l == nil {
		return fmt.Errorf("%w: license issue link is nil", ErrValidation)
	}
	l.IssueKey = strings.TrimSpace(l.IssueKey)
	l.IssueURL = strings.TrimSpace(l.IssueURL)
	l.Status = strings.TrimSpace(l.Status)
	if l.LicenseID == uuid.Nil {
		return fmt.Errorf("%w: license issue link license id is required", ErrValidation)
	}
	if l.IssueKey == "" {
		return fmt.Errorf("%w: license issue link issue key is required", ErrValidation)
	}
	if l.IssueURL != "" {
		if _, err := url.ParseRequestURI(l.IssueURL); err != nil {
			return fmt.Errorf("%w: license issue link issue url: %v", ErrValidation, err)
		}
	}
	if l.RenewalDate != nil && l.RenewalDate.IsZero() {
		return fmt.Errorf("%w: license issue link renewal date is invalid", ErrValidation)
	}
	return nil
}
