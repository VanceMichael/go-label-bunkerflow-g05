package domain

import (
	"testing"
	"time"
)

func TestWindowAndTransferRules(t *testing.T) {
	now := time.Now()
	rule := WindowRule{MinNotice: time.Hour, MaxDuration: 4 * time.Hour}
	if err := rule.Validate(now.Add(2*time.Hour), now.Add(3*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if rule.Validate(now, now.Add(time.Hour), now) == nil {
		t.Fatal("short notice accepted")
	}
	transfer := TransferRule{MaxDraftKG: 1000, MinDraftKG: 10, MaxRateKGPerMinute: 100}
	if err := transfer.Validate(500, 20); err != nil {
		t.Fatal(err)
	}
	if !transfer.RateWithin(500, 10) || transfer.RateWithin(500, 2) {
		t.Fatal("rate rule wrong")
	}
}
func TestTenantAndTimeHelpers(t *testing.T) {
	if !SameTenant("a", "a") || SameTenant("a", "b") {
		t.Fatal("tenant helper wrong")
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	if !WithinInclusive(start, start, end) || !WithinInclusive(end, start, end) {
		t.Fatal("inclusive range wrong")
	}
	if StateLabel(StateCompleted) != "completed and billed" {
		t.Fatal("state label wrong")
	}
}
