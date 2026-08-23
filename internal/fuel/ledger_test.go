package fuel

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestLedgerMaintainsBalanceAndRejectsNegative(t *testing.T) {
	now := time.Now()
	ledger := Ledger{}
	if err := ledger.Append(LedgerEntry{ID: "1", LotID: "l", QuantityKG: 100, Direction: "in", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Append(LedgerEntry{ID: "2", LotID: "l", QuantityKG: 30, Direction: "out", CreatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if got := ledger.Balance("l", 0); got != 70 {
		t.Fatalf("balance=%v", got)
	}
	if err := ledger.Validate("l", 0); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Append(LedgerEntry{ID: "3", LotID: "l", QuantityKG: 100, Direction: "out", CreatedAt: now.Add(2 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Validate("l", 0); err == nil {
		t.Fatal("negative ledger accepted")
	}
	_ = domain.ErrConflict
}
