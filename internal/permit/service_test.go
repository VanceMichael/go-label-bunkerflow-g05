package permit

import (
	"testing"
	"time"
)

func TestPermitRequiresEverySafetyCheckBeforeIssue(t *testing.T) {
	permit, err := New("permit-1", "order-1")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	if permit.Ready() {
		t.Fatal("new permit is ready")
	}
	for _, check := range StandardChecks() {
		permit, err = permit.Complete(check.Code, "user-planner", now)
		if err != nil {
			t.Fatalf("complete %s: %v", check.Code, err)
		}
	}
	if !permit.Ready() || len(permit.Missing()) != 0 {
		t.Fatalf("permit=%+v missing=%v", permit, permit.Missing())
	}
	issued, err := permit.Issue("user-planner", now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !issued.Active(now.Add(30 * time.Minute)) {
		t.Fatal("issued permit is inactive")
	}
	if issued.Active(now.Add(2 * time.Hour)) {
		t.Fatal("expired permit is active")
	}
}

func TestPermitCopyDoesNotShareChecks(t *testing.T) {
	permit, err := New("permit-1", "order-1")
	if err != nil {
		t.Fatal(err)
	}
	copyOf := permit.Copy()
	copyOf.Checks[0].Label = "tampered"
	if permit.Checks[0].Label == "tampered" {
		t.Fatal("permit copy shared checks")
	}
}

func TestPermitCannotIssueWithMissingCheck(t *testing.T) {
	permit, err := New("permit-1", "order-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range StandardChecks()[1:] {
		permit, err = permit.Complete(check.Code, "actor", time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := permit.Issue("actor", time.Now(), time.Hour); err == nil {
		t.Fatal("incomplete permit issued")
	}
	if len(permit.Missing()) != 1 || permit.Missing()[0].Code != "VESSEL_CERTIFICATE" {
		t.Fatalf("missing=%+v", permit.Missing())
	}
}
