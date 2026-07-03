package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type License struct {
	Base
	ProductID             uuid.UUID
	VendorID              uuid.UUID
	Department            string
	LicenseKey            string
	SubscriptionID        string
	ContractNumber        string
	PurchaseOrder         string
	Invoice               string
	PurchaseDate          *time.Time
	RenewalDate           *time.Time
	ExpirationDate        *time.Time
	MaintenanceExpiration *time.Time
	SeatCount             int
	AssignedSeats         int
	Cost                  decimal.Decimal
	Currency              string
	Notes                 string
	Type                  LicenseType
}

func (l *License) Validate() error {
	if l == nil {
		return fmt.Errorf("%w: license is nil", ErrValidation)
	}
	l.LicenseKey = strings.TrimSpace(l.LicenseKey)
	l.Department = strings.TrimSpace(l.Department)
	l.SubscriptionID = strings.TrimSpace(l.SubscriptionID)
	l.ContractNumber = strings.TrimSpace(l.ContractNumber)
	l.PurchaseOrder = strings.TrimSpace(l.PurchaseOrder)
	l.Invoice = strings.TrimSpace(l.Invoice)
	l.Currency = strings.TrimSpace(strings.ToUpper(l.Currency))
	l.Notes = strings.TrimSpace(l.Notes)
	if l.ProductID == uuid.Nil {
		return fmt.Errorf("%w: license product id is required", ErrValidation)
	}
	if l.VendorID == uuid.Nil {
		return fmt.Errorf("%w: license vendor id is required", ErrValidation)
	}
	if err := l.Type.Validate(); err != nil {
		return err
	}
	if l.SeatCount < 0 {
		return fmt.Errorf("%w: seat count cannot be negative", ErrValidation)
	}
	if l.AssignedSeats < 0 {
		return fmt.Errorf("%w: assigned seats cannot be negative", ErrValidation)
	}
	if l.AssignedSeats > l.SeatCount {
		return fmt.Errorf("%w: assigned seats cannot exceed seat count", ErrValidation)
	}
	if l.Currency != "" && len(l.Currency) != 3 {
		return fmt.Errorf("%w: currency must be ISO-4217 code", ErrValidation)
	}
	return nil
}

func (l License) AvailableSeats() int {
	return l.SeatCount - l.AssignedSeats
}
