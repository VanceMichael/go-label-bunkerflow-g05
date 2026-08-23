package permit

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Check struct {
	Code        string
	Label       string
	Required    bool
	Completed   bool
	CompletedBy string
	CompletedAt *time.Time
}

type Permit struct {
	ID        string
	OrderID   string
	Status    string
	Checks    []Check
	IssuedBy  string
	IssuedAt  *time.Time
	ExpiresAt *time.Time
	Version   int64
}

func StandardChecks() []Check {
	return []Check{
		{Code: "VESSEL_CERTIFICATE", Label: "Vessel certificate is active", Required: true},
		{Code: "HOSE_INSPECTION", Label: "Transfer hose passed inspection", Required: true},
		{Code: "EMERGENCY_LINK", Label: "Emergency stop link is connected", Required: true},
		{Code: "CREW_BRIEFING", Label: "Transfer crews completed briefing", Required: true},
		{Code: "WEATHER", Label: "Weather is inside operating envelope", Required: true},
		{Code: "PORT_NOTICE", Label: "Port control received notice", Required: true},
	}
}

func New(id, orderID string) (Permit, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(orderID) == "" {
		return Permit{}, domain.ErrInvalid
	}
	return Permit{ID: id, OrderID: orderID, Status: "pending", Checks: StandardChecks(), Version: 1}, nil
}

func (p Permit) Complete(code, actor string, at time.Time) (Permit, error) {
	if p.Status != "pending" {
		return p, fmt.Errorf("%w: permit is %s", domain.ErrConflict, p.Status)
	}
	if strings.TrimSpace(actor) == "" || at.IsZero() {
		return p, domain.ErrInvalid
	}
	copyOf := p.Copy()
	found := false
	for index := range copyOf.Checks {
		if copyOf.Checks[index].Code != code {
			continue
		}
		found = true
		copyOf.Checks[index].Completed = true
		copyOf.Checks[index].CompletedBy = actor
		completedAt := at.UTC()
		copyOf.Checks[index].CompletedAt = &completedAt
	}
	if !found {
		return p, domain.ErrNotFound
	}
	copyOf.Version++
	return copyOf, nil
}

func (p Permit) Ready() bool {
	if p.Status != "pending" {
		return false
	}
	for _, check := range p.Checks {
		if check.Required && !check.Completed {
			return false
		}
	}
	return true
}

func (p Permit) Issue(actor string, at time.Time, duration time.Duration) (Permit, error) {
	if !p.Ready() {
		return p, fmt.Errorf("%w: required safety checks are incomplete", domain.ErrConflict)
	}
	if actor == "" || duration <= 0 {
		return p, domain.ErrInvalid
	}
	copyOf := p.Copy()
	issued := at.UTC()
	expires := issued.Add(duration)
	copyOf.Status = "approved"
	copyOf.IssuedBy = actor
	copyOf.IssuedAt = &issued
	copyOf.ExpiresAt = &expires
	copyOf.Version++
	return copyOf, nil
}

func (p Permit) Active(at time.Time) bool {
	return p.Status == "approved" && p.IssuedAt != nil && p.ExpiresAt != nil &&
		!at.Before(*p.IssuedAt) && at.Before(*p.ExpiresAt)
}

func (p Permit) Revoke(actor string, at time.Time) (Permit, error) {
	if p.Status != "approved" || actor == "" || at.IsZero() {
		return p, domain.ErrConflict
	}
	copyOf := p.Copy()
	copyOf.Status = "revoked"
	copyOf.Version++
	return copyOf, nil
}

func (p Permit) Missing() []Check {
	missing := make([]Check, 0)
	for _, check := range p.Checks {
		if check.Required && !check.Completed {
			missing = append(missing, check)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].Code < missing[j].Code })
	return missing
}

func (p Permit) Copy() Permit {
	copyOf := p
	copyOf.Checks = append([]Check(nil), p.Checks...)
	return copyOf
}
