package fuel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func fuelFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:fuel-service-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	return New(store, audit.New(store)), actor, func() { store.Close() }
}

func TestReceiveLotIsAtomicWhenAuditFails(t *testing.T) {
	svc, actor, closeFn := fuelFixture(t)
	defer closeFn()
	svc.Store.Hooks.FailAudit = true
	_, err := svc.ReceiveLot(context.Background(), actor, ReceiveInput{LotNumber: "ZJ-MEOH-001", Product: "green-methanol", QuantityKG: 5000, ReceivedAt: time.Now(), Quality: domain.QualityApproved}, "req-lot")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	var count int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM fuel_lots`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("fuel lots persisted despite audit failure: %d", count)
	}
}

func TestReceiveLotPersistsWhenAuditSucceeds(t *testing.T) {
	svc, actor, closeFn := fuelFixture(t)
	defer closeFn()
	item, err := svc.ReceiveLot(context.Background(), actor, ReceiveInput{LotNumber: "ZJ-MEOH-002", Product: "green-methanol", QuantityKG: 5000, ReceivedAt: time.Now(), Quality: domain.QualityApproved}, "req-lot")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM fuel_lots WHERE id=?`, item.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("fuel lots = %d", count)
	}
}
