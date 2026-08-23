package incident

import (
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type TimelineEvent struct {
	ID         string
	IncidentID string
	Action     string
	ActorID    string
	At         time.Time
	Detail     string
}
type Timeline []TimelineEvent

func (t Timeline) Ordered() Timeline {
	copyOf := append(Timeline(nil), t...)
	sort.SliceStable(copyOf, func(i, j int) bool {
		if copyOf[i].At.Equal(copyOf[j].At) {
			return copyOf[i].ID < copyOf[j].ID
		}
		return copyOf[i].At.Before(copyOf[j].At)
	})
	return copyOf
}
func (t Timeline) HasAction(action string) bool {
	for _, event := range t {
		if event.Action == action {
			return true
		}
	}
	return false
}
func (t Timeline) Summary() string {
	ordered := t.Ordered()
	parts := make([]string, 0, len(ordered))
	for _, event := range ordered {
		if strings.TrimSpace(event.Detail) != "" {
			parts = append(parts, event.Action+":"+event.Detail)
		} else {
			parts = append(parts, event.Action)
		}
	}
	return strings.Join(parts, " -> ")
}
func (t *Timeline) Add(event TimelineEvent) error {
	if event.ID == "" || event.IncidentID == "" || event.Action == "" || event.At.IsZero() {
		return domain.ErrInvalid
	}
	*t = append(*t, event)
	return nil
}
