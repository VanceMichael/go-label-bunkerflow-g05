package bunkering

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/quality"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func abortFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:abort-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	for _, ddl := range []string{
		`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`,
		`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`,
		`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`,
		`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`,
	} {
		if _, err := store.DB.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(store, audit.New(store), outbox.New(store), fuel.New(store, audit.New(store)), quality.New(store, audit.New(store)), nil)
	return svc, actor, func() { store.Close() }
}

// seedTransferring drives an order to the transferring state, where the lease is
// held and target_kg has been subtracted from available capacity.
func seedTransferring(t *testing.T, svc *Service, actor domain.Actor, target float64) string {
	t.Helper()
	order, err := svc.Create(context.Background(), actor, CreateInput{VesselID: "v", WindowID: "w", FuelLotID: "l", TargetKG: target, IdempotencyKey: "create"}, "req")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Approve(context.Background(), actor, order.ID, "req"); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkAlongside(context.Background(), actor, order.ID, "req"); err != nil {
		t.Fatal(err)
	}
	if err := svc.StartTransfer(context.Background(), actor, order.ID, "worker-a", "req"); err != nil {
		t.Fatal(err)
	}
	return order.ID
}

func TestAbortReleasesFuelAndClearsLeaseAtomically(t *testing.T) {
	svc, actor, closeFn := abortFixture(t)
	defer closeFn()
	orderID := seedTransferring(t, svc, actor, 100)

	if err := svc.Abort(context.Background(), actor, orderID, "req-abort"); err != nil {
		t.Fatalf("abort: %v", err)
	}
	stored, err := svc.Get(context.Background(), actor, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.StateCancelled {
		t.Fatalf("state=%s", stored.State)
	}
	if stored.LeaseOwner != "" || !stored.LeaseUntil.IsZero() {
		t.Fatalf("lease not cleared: owner=%q until=%v", stored.LeaseOwner, stored.LeaseUntil)
	}
	lot, err := svc.Fuel.Get(context.Background(), actor, "l")
	if err != nil {
		t.Fatal(err)
	}
	if lot.AvailableKG != 1000 {
		t.Fatalf("available=%v want 1000", lot.AvailableKG)
	}
}

func TestAbortAuditFailureLeavesFuelAndLeaseUntouched(t *testing.T) {
	svc, actor, closeFn := abortFixture(t)
	defer closeFn()
	orderID := seedTransferring(t, svc, actor, 100)

	// Simulate the external audit sink failing mid-abort: the compensation must
	// roll back with the rest of the transaction, leaving the order transferring
	// and the lease held so recovery does not reprocess it.
	svc.Store.Hooks.FailAudit = true
	err := svc.Abort(context.Background(), actor, orderID, "req-abort")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error=%v", err)
	}
	svc.Store.Hooks.FailAudit = false

	stored, err := svc.Get(context.Background(), actor, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.StateTransferring {
		t.Fatalf("state=%s want transferring", stored.State)
	}
	if stored.LeaseOwner != "worker-a" {
		t.Fatalf("lease owner=%q want worker-a", stored.LeaseOwner)
	}
	if !stored.LeaseUntil.After(time.Now()) {
		t.Fatalf("lease until=%v not in future", stored.LeaseUntil)
	}
	lot, err := svc.Fuel.Get(context.Background(), actor, "l")
	if err != nil {
		t.Fatal(err)
	}
	// Fuel was reserved (100kg) when the transfer started; a failed abort must
	// not double-restore it.
	if lot.AvailableKG != 900 {
		t.Fatalf("available=%v want 900 (reserved not restored on failure)", lot.AvailableKG)
	}
}

func TestAbortPreTransferDoesNotInflateAvailable(t *testing.T) {
	svc, actor, closeFn := abortFixture(t)
	defer closeFn()
	order, err := svc.Create(context.Background(), actor, CreateInput{VesselID: "v", WindowID: "w", FuelLotID: "l", TargetKG: 100, IdempotencyKey: "create"}, "req")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Approve(context.Background(), actor, order.ID, "req"); err != nil {
		t.Fatal(err)
	}
	// Aborting before StartTransfer never reserved fuel; the old code restored it anyway.
	if err := svc.Abort(context.Background(), actor, order.ID, "req-abort"); err != nil {
		t.Fatal(err)
	}
	lot, err := svc.Fuel.Get(context.Background(), actor, "l")
	if err != nil {
		t.Fatal(err)
	}
	if lot.AvailableKG != 1000 {
		t.Fatalf("available=%v want 1000 (no reserve before transfer)", lot.AvailableKG)
	}
}
