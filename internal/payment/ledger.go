package payment

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Attempt struct {
	ID         string
	InvoiceID  string
	Key        string
	Amount     int64
	State      string
	GatewayRef string
	CreatedAt  time.Time
	Error      string
}

type Ledger struct{ Attempts []Attempt }

func (l *Ledger) Start(invoiceID, key string, amount int64, at time.Time) (Attempt, error) {
	if strings.TrimSpace(invoiceID) == "" || strings.TrimSpace(key) == "" || amount <= 0 || at.IsZero() {
		return Attempt{}, domain.ErrInvalid
	}
	for _, existing := range l.Attempts {
		if existing.Key != key {
			continue
		}
		if existing.InvoiceID != invoiceID || existing.Amount != amount {
			return Attempt{}, domain.ErrIdempotency
		}
		return existing, nil
	}
	attempt := Attempt{ID: fmt.Sprintf("pay-%d", len(l.Attempts)+1), InvoiceID: invoiceID, Key: key, Amount: amount, State: "started", CreatedAt: at.UTC()}
	l.Attempts = append(l.Attempts, attempt)
	return attempt, nil
}

func (l *Ledger) Succeed(id, gatewayRef string) error {
	if gatewayRef == "" {
		return domain.ErrInvalid
	}
	for index := range l.Attempts {
		if l.Attempts[index].ID != id {
			continue
		}
		if l.Attempts[index].State == "succeeded" {
			return nil
		}
		if l.Attempts[index].State != "started" {
			return domain.ErrConflict
		}
		l.Attempts[index].State = "succeeded"
		l.Attempts[index].GatewayRef = gatewayRef
		return nil
	}
	return domain.ErrNotFound
}

func (l *Ledger) Fail(id string, cause error) error {
	for index := range l.Attempts {
		if l.Attempts[index].ID != id {
			continue
		}
		if l.Attempts[index].State == "succeeded" {
			return domain.ErrConflict
		}
		l.Attempts[index].State = "failed"
		if cause != nil {
			l.Attempts[index].Error = cause.Error()
		}
		return nil
	}
	return domain.ErrNotFound
}

func (l Ledger) Successful(invoiceID string) (Attempt, bool) {
	for _, attempt := range l.Attempts {
		if attempt.InvoiceID == invoiceID && attempt.State == "succeeded" {
			return attempt, true
		}
	}
	return Attempt{}, false
}

func (l Ledger) Retryable(invoiceID string) []Attempt {
	result := make([]Attempt, 0)
	for _, attempt := range l.Attempts {
		if attempt.InvoiceID == invoiceID && attempt.State == "failed" {
			result = append(result, attempt)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result
}

func (l Ledger) Copy() Ledger { return Ledger{Attempts: append([]Attempt(nil), l.Attempts...)} }
