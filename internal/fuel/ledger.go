package fuel

import (
	"fmt"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type LedgerEntry struct {
	ID         string
	LotID      string
	OrderID    string
	QuantityKG float64
	Direction  string
	CreatedAt  time.Time
	Reason     string
}
type Ledger struct{ Entries []LedgerEntry }

func (l Ledger) Balance(lotID string, opening float64) float64 {
	balance := opening
	for _, entry := range l.Entries {
		if entry.LotID != lotID {
			continue
		}
		if entry.Direction == "in" {
			balance += entry.QuantityKG
		} else {
			balance -= entry.QuantityKG
		}
	}
	return balance
}
func (l Ledger) Validate(lotID string, opening float64) error {
	entries := append([]LedgerEntry(nil), l.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })
	balance := opening
	for _, entry := range entries {
		if entry.LotID != lotID {
			continue
		}
		if entry.QuantityKG <= 0 {
			return domain.ErrInvalid
		}
		if entry.Direction == "in" {
			balance += entry.QuantityKG
		} else if entry.Direction == "out" {
			balance -= entry.QuantityKG
		} else {
			return fmt.Errorf("%w: ledger direction", domain.ErrInvalid)
		}
		if balance < 0 {
			return fmt.Errorf("%w: negative fuel balance", domain.ErrConflict)
		}
	}
	return nil
}
func (l Ledger) Since(lotID string, after time.Time) []LedgerEntry {
	result := make([]LedgerEntry, 0)
	for _, entry := range l.Entries {
		if entry.LotID == lotID && !entry.CreatedAt.Before(after) {
			result = append(result, entry)
		}
	}
	return result
}
func (l *Ledger) Append(entry LedgerEntry) error {
	if entry.ID == "" || entry.LotID == "" || entry.QuantityKG <= 0 {
		return domain.ErrInvalid
	}
	l.Entries = append(l.Entries, entry)
	return nil
}
