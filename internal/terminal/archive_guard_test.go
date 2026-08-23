package terminal

import (
	"context"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

const tenantID = "tenant-zj"

func setupGuardDB(t *testing.T) (*ArchiveGuard, string) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:archive-guard?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	db := store.DB
	if _, err := db.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	return &ArchiveGuard{DB: db}, "t"
}

func TestBlockingResourcesEmptyTerminalIsZero(t *testing.T) {
	g, terminalID := setupGuardDB(t)
	active, err := g.BlockingResources(context.Background(), tenantID, terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if active != 0 {
		t.Fatalf("expected 0 blocking resources, got %d", active)
	}
}

func TestBlockingResourcesCountsInFlightTransferLease(t *testing.T) {
	g, terminalID := setupGuardDB(t)
	db := g.DB
	if _, err := db.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','released',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	leaseUntil := storage.StringTime(time.Now().Add(5 * time.Minute))
	if _, err := db.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES ('o','tenant-zj','v','w','l',100,0,'transferring','worker',?,1,CURRENT_TIMESTAMP)`, leaseUntil); err != nil {
		t.Fatal(err)
	}
	active, err := g.BlockingResources(context.Background(), tenantID, terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("expected 1 blocking resource (in-flight lease), got %d", active)
	}
}

func TestBlockingResourcesCountsPendingOutboxMessage(t *testing.T) {
	g, terminalID := setupGuardDB(t)
	db := g.DB
	if _, err := db.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','cancelled',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	now := storage.StringTime(time.Now())
	if _, err := db.Exec(`INSERT INTO outbox_messages(id,tenant_id,topic,payload,status,attempts,next_attempt,created_at) VALUES ('m','tenant-zj','window.created','w','pending',0,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	active, err := g.BlockingResources(context.Background(), tenantID, terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("expected 1 blocking resource (pending outbox), got %d", active)
	}
}

func TestBlockingResourcesIgnoresReleasedLease(t *testing.T) {
	g, terminalID := setupGuardDB(t)
	db := g.DB
	if _, err := db.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','released',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES ('o','tenant-zj','v','w','l',100,100,'completed',NULL,NULL,1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	active, err := g.BlockingResources(context.Background(), tenantID, terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if active != 0 {
		t.Fatalf("expected 0 blocking resources (released lease), got %d", active)
	}
}

func TestArchiveBlockedByInFlightLease(t *testing.T) {
	g, terminalID := setupGuardDB(t)
	db := g.DB
	if _, err := db.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','released',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	leaseUntil := storage.StringTime(time.Now().Add(5 * time.Minute))
	if _, err := db.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES ('o','tenant-zj','v','w','l',100,0,'transferring','worker',?,1,CURRENT_TIMESTAMP)`, leaseUntil); err != nil {
		t.Fatal(err)
	}
	active, err := g.BlockingResources(context.Background(), tenantID, terminalID)
	if err != nil {
		t.Fatal(err)
	}
	if active == 0 {
		t.Fatal("archive allowed despite in-flight transfer lease")
	}
}
