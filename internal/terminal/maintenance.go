package terminal

import (
	"fmt"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Maintenance struct {
	ID         string
	TerminalID string
	StartsAt   time.Time
	EndsAt     time.Time
	Reason     string
	Status     string
}

func (m Maintenance) Valid() error {
	if m.ID == "" || m.TerminalID == "" || m.Reason == "" || !m.EndsAt.After(m.StartsAt) {
		return domain.ErrInvalid
	}
	if m.Status != "planned" && m.Status != "active" && m.Status != "closed" {
		return fmt.Errorf("%w: maintenance status", domain.ErrInvalid)
	}
	return nil
}
func (m Maintenance) Conflicts(window domain.BunkerWindow) bool {
	return m.Status != "closed" && window.Status != "cancelled" && m.StartsAt.Before(window.EndsAt) && window.StartsAt.Before(m.EndsAt)
}
func SortMaintenance(items []Maintenance) []Maintenance {
	copyOf := append([]Maintenance(nil), items...)
	sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].StartsAt.Before(copyOf[j].StartsAt) })
	return copyOf
}
func Due(items []Maintenance, now time.Time) []Maintenance {
	result := make([]Maintenance, 0)
	for _, item := range items {
		if item.Status == "planned" && !item.StartsAt.After(now) {
			result = append(result, item)
		}
	}
	return SortMaintenance(result)
}
