package operations

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func openStore(t *testing.T) *storage.Store {
	t.Helper()
	ctx := context.Background()
	// Use an on-disk temp file so each test gets an isolated database; the
	// in-memory shared-cache DSN would persist across tests in the package.
	store, err := storage.Open(ctx, "file:"+t.Name()+"?mode=memory")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	seed := func(query string, args ...any) {
		t.Helper()
		if _, err := store.DB.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("seed %q: %v", query, err)
		}
	}
	seed(`INSERT OR IGNORE INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`)
	seed(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','1','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`)
	seed(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`)
	seed(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`)
	seed(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('ready','tenant-zj','v','w','l',100,0,'sampled',1,CURRENT_TIMESTAMP)`)
	seed(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('unready','tenant-zj','v','w','l',100,0,'transferring',1,CURRENT_TIMESTAMP)`)
	return store
}

func orderState(t *testing.T, store *storage.Store, id string) string {
	t.Helper()
	var state string
	if err := store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id=?`, id).Scan(&state); err != nil {
		t.Fatalf("select state %s: %v", id, err)
	}
	return state
}

// If any order in the batch fails the sampled precondition, no order in the
// batch may be committed. The batch must be atomic.
func TestBatchCompleteRollsBackAllOrdersWhenPreconditionFails(t *testing.T) {
	store := openStore(t)
	svc := New(store)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}

	err := svc.BatchComplete(context.Background(), actor, []string{"ready", "unready"}, "req-batch")
	if !errors.Is(err, domain.ErrNoQuality) {
		t.Fatalf("err=%v want ErrNoQuality", err)
	}

	if got := orderState(t, store, "ready"); got != string(domain.StateSampled) {
		t.Fatalf("ready order committed despite rollback: state=%s", got)
	}
	if got := orderState(t, store, "unready"); got != "transferring" {
		t.Fatalf("unready order changed: state=%s", got)
	}
}

// A fully valid batch completes every order atomically.
func TestBatchCompleteCommitsAllWhenPreconditionsPass(t *testing.T) {
	store := openStore(t)
	svc := New(store)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}

	// promote the unready order so the whole batch is sampled
	if _, err := store.DB.Exec(`UPDATE transfer_orders SET state='sampled' WHERE id='unready'`); err != nil {
		t.Fatal(err)
	}

	if err := svc.BatchComplete(context.Background(), actor, []string{"ready", "unready"}, "req-batch"); err != nil {
		t.Fatalf("err=%v", err)
	}
	if got := orderState(t, store, "ready"); got != string(domain.StateCompleted) {
		t.Fatalf("ready state=%s", got)
	}
	if got := orderState(t, store, "unready"); got != string(domain.StateCompleted) {
		t.Fatalf("unready state=%s", got)
	}
}

func TestBatchCompleteRejectsEmptyBatch(t *testing.T) {
	store := openStore(t)
	svc := New(store)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}
	if err := svc.BatchComplete(context.Background(), actor, nil, "req"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("err=%v want ErrInvalid", err)
	}
}

func TestBatchCompleteRejectsUnknownOrder(t *testing.T) {
	store := openStore(t)
	svc := New(store)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}
	err := svc.BatchComplete(context.Background(), actor, []string{"ready", "missing"}, "req")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
	if got := orderState(t, store, "ready"); got != string(domain.StateSampled) {
		t.Fatalf("ready order committed despite rollback: state=%s", got)
	}
}
