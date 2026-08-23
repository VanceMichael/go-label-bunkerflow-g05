package audit

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestFilterEventsOrdersStableAndScopesAction(t *testing.T) {
	now := time.Now()
	events := []domain.AuditEvent{{ID: "2", Action: "b", CreatedAt: now}, {ID: "1", Action: "a", CreatedAt: now}, {ID: "3", Action: "a", CreatedAt: now.Add(time.Hour)}}
	filtered := FilterEvents(events, Filter{Action: "a"})
	if len(filtered) != 2 || filtered[0].ID != "1" {
		t.Fatalf("filtered=%+v", filtered)
	}
	if Actions(events) != "a,b" {
		t.Fatalf("actions=%s", Actions(events))
	}
}
