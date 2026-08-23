package terminal

import (
	"testing"
	"time"
)

func TestNextOpeningAdvancesToNextLocalDay(t *testing.T) {
	hours, err := ParseHours("08:00", "18:00", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	value := time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC)
	next := hours.NextOpening(value)
	if next.Hour() != 8 || !next.After(value) {
		t.Fatalf("next=%v", next)
	}
}
