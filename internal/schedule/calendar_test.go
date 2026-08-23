package schedule

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestCalendarFindsFutureOpenSlots(t *testing.T) {
	now := time.Now()
	terminal := domain.Terminal{ID: "t", Status: "active"}
	calendar := Calendar{Terminal: terminal, Now: func() time.Time { return now }, Windows: []domain.BunkerWindow{{ID: "w", TerminalID: "t", Status: "open", StartsAt: now.Add(time.Hour), EndsAt: now.Add(2 * time.Hour)}}}
	if err := calendar.Validate(); err != nil {
		t.Fatal(err)
	}
	slot, err := calendar.Find("w")
	if err != nil || !slot.Available {
		t.Fatalf("slot=%+v err=%v", slot, err)
	}
	if !AvailableAt(slot.Window, now.Add(90*time.Minute)) {
		t.Fatal("window unavailable inside range")
	}
}
