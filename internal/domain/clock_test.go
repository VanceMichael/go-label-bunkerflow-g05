package domain

import (
	"testing"
	"time"
)

func TestClockDateBoundaries(t *testing.T) {
	value := time.Date(2026, 8, 23, 14, 30, 0, 0, time.FixedZone("CST", 8*3600))
	if DateKey(value) != "2026-08-23" || MonthKey(value) != "2026-08" {
		t.Fatal("date keys wrong")
	}
	if !SameBusinessDay(value, value.Add(8*time.Hour)) {
		t.Fatal("same day wrong")
	}
	if !StartOfDay(value).Before(value) || !EndOfDay(value).After(value) {
		t.Fatal("day bounds wrong")
	}
	fixed := FixedClock{Value: value}
	if !fixed.Now().Equal(value) {
		t.Fatal("fixed clock wrong")
	}
}
