package domain

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type Vendor struct {
	Base
	Name           string
	SupportURL     string
	AccountManager string
	Notes          string
	Contacts       []VendorContact
}

type VendorContact struct {
	Name  string
	Email string
	Phone string
	Role  string
}

func (v *Vendor) Validate() error {
	if v == nil {
		return fmt.Errorf("%w: vendor is nil", ErrValidation)
	}
	v.Name = strings.TrimSpace(v.Name)
	v.SupportURL = strings.TrimSpace(v.SupportURL)
	v.AccountManager = strings.TrimSpace(v.AccountManager)
	v.Notes = strings.TrimSpace(v.Notes)
	if v.Name == "" {
		return fmt.Errorf("%w: vendor name is required", ErrValidation)
	}
	if v.SupportURL != "" {
		if _, err := url.ParseRequestURI(v.SupportURL); err != nil {
			return fmt.Errorf("%w: vendor support url: %v", ErrValidation, err)
		}
	}
	for i := range v.Contacts {
		if err := v.Contacts[i].Validate(); err != nil {
			return fmt.Errorf("%w: contact %d: %v", ErrValidation, i, err)
		}
	}
	return nil
}

func (c *VendorContact) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	c.Email = strings.TrimSpace(c.Email)
	c.Phone = strings.TrimSpace(c.Phone)
	c.Role = strings.TrimSpace(c.Role)
	if c.Name == "" {
		return fmt.Errorf("%w: vendor contact name is required", ErrValidation)
	}
	if c.Email != "" {
		if _, err := mail.ParseAddress(c.Email); err != nil {
			return fmt.Errorf("%w: vendor contact email: %v", ErrValidation, err)
		}
	}
	return nil
}

func NewVendorID() uuid.UUID { return uuid.New() }
