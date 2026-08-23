package terminal

import (
	"testing"
	"time"
)

func TestOperatingHoursHandlesDayAndOvernightWindows(t *testing.T) {
	hours, err := ParseHours("08:00", "18:00", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if !hours.OpenAt(time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("daytime closed")
	}
	if hours.OpenAt(time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC)) {
		t.Fatal("night open")
	}
	overnight, err := ParseHours("22:00", "06:00", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if !overnight.OpenAt(time.Date(2026, 8, 23, 23, 0, 0, 0, time.UTC)) {
		t.Fatal("overnight start closed")
	}
	if !overnight.OpenAt(time.Date(2026, 8, 24, 5, 0, 0, 0, time.UTC)) {
		t.Fatal("overnight end closed")
	}
}
