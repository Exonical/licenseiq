package domain

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type Product struct {
	Base
	Name        string
	VendorID    uuid.UUID
	Category    string
	Version     string
	Website     string
	Description string
	Tags        []string
}

func (p *Product) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: product is nil", ErrValidation)
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Category = strings.TrimSpace(p.Category)
	p.Version = strings.TrimSpace(p.Version)
	p.Website = strings.TrimSpace(p.Website)
	p.Description = strings.TrimSpace(p.Description)
	p.Tags = trimStrings(p.Tags)
	if p.Name == "" {
		return fmt.Errorf("%w: product name is required", ErrValidation)
	}
	if p.VendorID == uuid.Nil {
		return fmt.Errorf("%w: product vendor id is required", ErrValidation)
	}
	if p.Website != "" {
		if _, err := url.ParseRequestURI(p.Website); err != nil {
			return fmt.Errorf("%w: product website: %v", ErrValidation, err)
		}
	}
	return nil
}
