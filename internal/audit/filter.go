package audit

import (
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Filter struct {
	Action   string
	ObjectID string
	From     *time.Time
	Until    *time.Time
	ActorID  string
}

func (f Filter) Match(event domain.AuditEvent) bool {
	if f.Action != "" && event.Action != f.Action {
		return false
	}
	if f.ObjectID != "" && event.ObjectID != f.ObjectID {
		return false
	}
	if f.ActorID != "" && event.ActorID != f.ActorID {
		return false
	}
	if f.From != nil && event.CreatedAt.Before(*f.From) {
		return false
	}
	if f.Until != nil && event.CreatedAt.After(*f.Until) {
		return false
	}
	return true
}
func FilterEvents(events []domain.AuditEvent, filter Filter) []domain.AuditEvent {
	result := make([]domain.AuditEvent, 0, len(events))
	for _, event := range events {
		if filter.Match(event) {
			result = append(result, event)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}
func Actions(events []domain.AuditEvent) string {
	values := make([]string, 0, len(events))
	seen := map[string]bool{}
	for _, event := range events {
		if strings.TrimSpace(event.Action) != "" && !seen[event.Action] {
			seen[event.Action] = true
			values = append(values, event.Action)
		}
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}
