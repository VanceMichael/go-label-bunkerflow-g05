package schedule

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func scheduleFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:schedule-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	_, err = store.DB.Exec(`INSERT INTO terminals(id, tenant_id, name, timezone, open_from, open_until, status, created_at) VALUES ('terminal-1','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	return New(store, audit.New(store), outbox.New(store)), actor, func() { store.Close() }
}

func TestCreateWindowIsAtomicWhenAuditFails(t *testing.T) {
	svc, actor, closeFn := scheduleFixture(t)
	defer closeFn()
	svc.Store.Hooks.FailAudit = true
	_, err := svc.CreateWindow(context.Background(), actor, WindowInput{TerminalID: "terminal-1", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-1")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	var count int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM bunker_windows`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("windows = %d", count)
	}
}

func TestCancelWindowIsAtomicWhenAuditFails(t *testing.T) {
	svc, actor, closeFn := scheduleFixture(t)
	defer closeFn()
	window, err := svc.CreateWindow(context.Background(), actor, WindowInput{TerminalID: "terminal-1", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-create")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ClaimWindow(context.Background(), actor, window.ID, "owner-a", "req-claim"); err != nil {
		t.Fatal(err)
	}
	svc.Store.Hooks.FailAudit = true
	if !errors.Is(svc.CancelWindow(context.Background(), actor, window.ID, "req-cancel"), domain.ErrUnavailable) {
		t.Fatal("expected audit failure")
	}
	stored, err := svc.Get(context.Background(), actor, window.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "claimed" {
		t.Fatalf("status=%s", stored.Status)
	}
	var owner sql.NullString
	if err := svc.Store.DB.QueryRow(`SELECT owner_id FROM bunker_windows WHERE id=?`, window.ID).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if !owner.Valid || owner.String != "owner-a" {
		t.Fatalf("owner=%+v", owner)
	}
}

func TestCancelWindowRejectsActiveTransfer(t *testing.T) {
	svc, actor, closeFn := scheduleFixture(t)
	defer closeFn()
	window, err := svc.CreateWindow(context.Background(), actor, WindowInput{TerminalID: "terminal-1", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-create")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Store.DB.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v',?, 'l',100,0,'planned',1,CURRENT_TIMESTAMP)`, window.ID); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(svc.CancelWindow(context.Background(), actor, window.ID, "req-cancel"), domain.ErrConflict) {
		t.Fatal("expected conflict")
	}
	var status string
	if err := svc.Store.DB.QueryRow(`SELECT status FROM bunker_windows WHERE id=?`, window.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "open" {
		t.Fatalf("status=%s", status)
	}
}

func TestConcurrentWindowClaimHasOneOwner(t *testing.T) {
	svc, actor, closeFn := scheduleFixture(t)
	defer closeFn()
	item, err := svc.CreateWindow(context.Background(), actor, WindowInput{TerminalID: "terminal-1", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-2")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, owner := range []string{"a", "b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			results <- svc.ClaimWindow(context.Background(), actor, item.ID, owner, "req-claim-"+owner)
		}(owner)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("claim error = %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful claims = %d", success)
	}
}
