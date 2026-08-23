package domain

import (
	"errors"
	"testing"
	"time"
)

func TestOperationStateMachineAllowsOnlyOperationalTransitions(t *testing.T) {
	valid := [][2]OperationState{
		{StatePlanned, StateApproved}, {StateApproved, StateAlongside},
		{StateAlongside, StateTransferring}, {StateTransferring, StateSampled},
		{StateSampled, StateCompleted}, {StatePlanned, StateCancelled},
	}
	for _, pair := range valid {
		if err := Transition(pair[0], pair[1]); err != nil {
			t.Fatalf("%s -> %s: %v", pair[0], pair[1], err)
		}
	}
	invalid := [][2]OperationState{{StateCompleted, StatePlanned}, {StatePlanned, StateCompleted}, {StateApproved, StateCompleted}, {StateCancelled, StateApproved}}
	for _, pair := range invalid {
		if err := Transition(pair[0], pair[1]); !errors.Is(err, ErrConflict) {
			t.Fatalf("%s -> %s error = %v", pair[0], pair[1], err)
		}
	}
}

func TestCertificateRequiresVerificationAndFutureExpiry(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	if (Certificate{Verified: false, ExpiresAt: now.Add(time.Hour)}).Active(now) {
		t.Fatal("unverified certificate was accepted")
	}
	if (Certificate{Verified: true, ExpiresAt: now.Add(-time.Second)}).Active(now) {
		t.Fatal("expired certificate was accepted")
	}
	if !(Certificate{Verified: true, ExpiresAt: now.Add(time.Hour)}).Active(now) {
		t.Fatal("valid certificate was rejected")
	}
}

func TestNormalizeIdentifiersKeepsTenantSafeCanonicalValues(t *testing.T) {
	if got := NormalizeIMO("  9384756 "); got != "9384756" {
		t.Fatalf("IMO = %q", got)
	}
	if got := NormalizeLotNumber(" lot-zj-01 "); got != "LOT-ZJ-01" {
		t.Fatalf("lot = %q", got)
	}
	if ValidateIMO("9384756") == false {
		t.Fatal("valid IMO rejected")
	}
	if ValidateIMO("bad") {
		t.Fatal("invalid IMO accepted")
	}
}
