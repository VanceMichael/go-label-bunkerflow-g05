package schedule

import (
	"fmt"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Slot struct {
	Window    domain.BunkerWindow
	Available bool
	Reason    string
}
type Calendar struct {
	Terminal domain.Terminal
	Windows  []domain.BunkerWindow
	Now      func() time.Time
}

func (c Calendar) Slots() []Slot {
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	items := make([]domain.BunkerWindow, len(c.Windows))
	copy(items, c.Windows)
	sort.Slice(items, func(i, j int) bool { return items[i].StartsAt.Before(items[j].StartsAt) })
	result := make([]Slot, 0, len(items))
	for _, window := range items {
		available := window.Status == "open" && window.StartsAt.After(now) && c.Terminal.Status == "active"
		reason := "available"
		if !available {
			reason = "not available"
		}
		result = append(result, Slot{Window: window, Available: available, Reason: reason})
	}
	return result
}
func (c Calendar) Find(id string) (Slot, error) {
	for _, slot := range c.Slots() {
		if slot.Window.ID == id {
			return slot, nil
		}
	}
	return Slot{}, fmt.Errorf("window %s: %w", id, domain.ErrNotFound)
}
func (c Calendar) Validate() error {
	if c.Terminal.Status != "active" {
		return domain.ErrConflict
	}
	for _, window := range c.Windows {
		if !window.EndsAt.After(window.StartsAt) {
			return domain.ErrInvalid
		}
		if window.TerminalID != c.Terminal.ID {
			return domain.ErrConflict
		}
	}
	return nil
}
func AvailableAt(window domain.BunkerWindow, at time.Time) bool {
	return window.Status == "open" && !at.Before(window.StartsAt) && at.Before(window.EndsAt)
}
