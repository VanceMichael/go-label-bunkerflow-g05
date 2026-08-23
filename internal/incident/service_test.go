package incident

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func incidentFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:incident-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	if _, err := store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',100,0,'transferring',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	return New(store, audit.New(store), outbox.New(store)), actor, func() { store.Close() }
}

func TestOpenIsAtomicWhenOutboxFails(t *testing.T) {
	svc, actor, closeFn := incidentFixture(t)
	defer closeFn()
	svc.Store.Hooks.FailOutbox = true
	_, err := svc.Open(context.Background(), actor, "o", "high", "hose leak", "req-1")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	var state string
	if err := svc.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id=?`, "o").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "transferring" {
		t.Fatalf("order state = %s, want transferring", state)
	}
	var incidents int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM incidents`).Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if incidents != 0 {
		t.Fatalf("orphan incidents = %d", incidents)
	}
	var outbox int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM outbox_messages`).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if outbox != 0 {
		t.Fatalf("orphan outbox = %d", outbox)
	}
}

func TestOpenPersistsIncidentAndCancelsOrder(t *testing.T) {
	svc, actor, closeFn := incidentFixture(t)
	defer closeFn()
	item, err := svc.Open(context.Background(), actor, "o", "high", "hose leak", "req-2")
	if err != nil {
		t.Fatal(err)
	}
	var state string
	if err := svc.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id=?`, "o").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "cancelled" {
		t.Fatalf("order state = %s, want cancelled", state)
	}
	var incidents int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM incidents WHERE id=?`, item.ID).Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 {
		t.Fatalf("incidents = %d", incidents)
	}
	var outbox int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM outbox_messages WHERE topic='incident.opened'`).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if outbox != 1 {
		t.Fatalf("outbox = %d", outbox)
	}
}

func TestOpenRejectsNonTransferringOrder(t *testing.T) {
	svc, actor, closeFn := incidentFixture(t)
	defer closeFn()
	if _, err := svc.Store.DB.Exec(`UPDATE transfer_orders SET state='alongside' WHERE id='o'`); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Open(context.Background(), actor, "o", "high", "hose leak", "req-3")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v", err)
	}
	var incidents int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM incidents`).Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if incidents != 0 {
		t.Fatalf("incidents = %d", incidents)
	}
}
