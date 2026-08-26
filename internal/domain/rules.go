package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type WindowRule struct {
	MinNotice       time.Duration
	MaxDuration     time.Duration
	AllowedStatuses []string
}

func (r WindowRule) Validate(start, end, now time.Time) error {
	if start.Before(now.Add(r.MinNotice)) {
		return fmt.Errorf("%w: insufficient notice", ErrInvalid)
	}
	if end.Before(start) || end.Sub(start) > r.MaxDuration {
		return fmt.Errorf("%w: window duration", ErrInvalid)
	}
	return nil
}
func (r WindowRule) Allows(status string) bool {
	for _, candidate := range r.AllowedStatuses {
		if candidate == status {
			return true
		}
	}
	return false
}

type TransferRule struct {
	MaxDraftKG         float64
	MinDraftKG         float64
	MaxRateKGPerMinute float64
	RequiredSteps      int
}

func (r TransferRule) Validate(target, vesselDraft float64) error {
	if target <= 0 || target > r.MaxDraftKG {
		return fmt.Errorf("%w: target quantity", ErrInvalid)
	}
	if vesselDraft < r.MinDraftKG {
		return fmt.Errorf("%w: vessel draft", ErrConflict)
	}
	return nil
}
func (r TransferRule) RateWithin(quantity float64, minutes int) bool {
	if minutes <= 0 {
		return false
	}
	return quantity/float64(minutes) <= r.MaxRateKGPerMinute
}

type ChargeRule struct {
	BaseCents       int64
	PerKG           int64
	HazardSurcharge int64
	NightSurcharge  int64
}

func (r ChargeRule) Amount(quantity float64, hazardous, night bool) int64 {
	amount := r.BaseCents + int64(math.Round(quantity))*r.PerKG
	if hazardous {
		amount += r.HazardSurcharge
	}
	if night {
		amount += r.NightSurcharge
	}
	return amount
}
func (r ChargeRule) ValidateCurrency(currency string) error {
	if strings.ToUpper(currency) != "USD" && strings.ToUpper(currency) != "CNY" {
		return fmt.Errorf("%w: currency", ErrInvalid)
	}
	return nil
}

type SafetyRule struct {
	MaxWindKnots  float64
	MaxWaveMeters float64
	RequiredCrew  int
}

func (r SafetyRule) Validate(wind, wave float64, crew int) error {
	if wind > r.MaxWindKnots || wave > r.MaxWaveMeters {
		return fmt.Errorf("%w: weather outside operating envelope", ErrConflict)
	}
	if crew < r.RequiredCrew {
		return fmt.Errorf("%w: insufficient crew", ErrConflict)
	}
	return nil
}

func PermitReady(status, certificate string, weatherOK bool) error {
	if status != "approved" {
		return fmt.Errorf("%w: permit status", ErrConflict)
	}
	if certificate != "active" {
		return fmt.Errorf("%w: vessel certificate", ErrConflict)
	}
	if !weatherOK {
		return fmt.Errorf("%w: weather", ErrConflict)
	}
	return nil
}
func SameTenant(values ...string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value == "" || value != values[0] {
			return false
		}
	}
	return true
}
func WithinInclusive(value, start, end time.Time) bool {
	return !value.Before(start) && !value.After(end)
}
func IsTerminalState(state OperationState) bool {
	return state == StateCompleted || state == StateCancelled
}
func StateLabel(state OperationState) string {
	switch state {
	case StatePlanned:
		return "awaiting approval"
	case StateApproved:
		return "approved for berth"
	case StateAlongside:
		return "vessel alongside"
	case StateTransferring:
		return "fuel transfer active"
	case StateSampled:
		return "quality review pending"
	case StateCompleted:
		return "completed and billed"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}
