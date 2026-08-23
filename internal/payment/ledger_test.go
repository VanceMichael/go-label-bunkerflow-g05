package payment

import (
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestPaymentLedgerReplaysSuccessfulAttempt(t *testing.T) {
	ledger := Ledger{}
	attempt, err := ledger.Start("invoice-1", "key-1", 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.Succeed(attempt.ID, "gateway-1"); err != nil {
		t.Fatal(err)
	}
	replayed, err := ledger.Start("invoice-1", "key-1", 100, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != attempt.ID {
		t.Fatalf("replayed=%+v", replayed)
	}
	if successful, ok := ledger.Successful("invoice-1"); !ok || successful.GatewayRef != "gateway-1" {
		t.Fatalf("successful=%+v ok=%v", successful, ok)
	}
}

func TestPaymentLedgerRejectsChangedIdempotentRequest(t *testing.T) {
	ledger := Ledger{}
	if _, err := ledger.Start("invoice-1", "key", 100, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Start("invoice-1", "key", 200, time.Now()); !errors.Is(err, domain.ErrIdempotency) {
		t.Fatalf("error=%v", err)
	}
}

func TestPaymentLedgerTracksRetryableFailuresAndCopies(t *testing.T) {
	ledger := Ledger{}
	attempt, err := ledger.Start("invoice-1", "key", 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.Fail(attempt.ID, domain.ErrUnavailable); err != nil {
		t.Fatal(err)
	}
	retryable := ledger.Retryable("invoice-1")
	if len(retryable) != 1 || retryable[0].Error == "" {
		t.Fatalf("retryable=%+v", retryable)
	}
	copyOf := ledger.Copy()
	copyOf.Attempts[0].State = "tampered"
	if ledger.Attempts[0].State == "tampered" {
		t.Fatal("ledger copy shared attempts")
	}
}
