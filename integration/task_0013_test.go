package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestStaleLeaseRenewalCannotOverwriteNewOwner(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0013?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-13','green-methanol',100,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,0,'transferring','worker-b','2030-01-01T01:00:00Z',2,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	err = rt.Bunkering.RenewLease(ctx, domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}, "o", "worker-a", now)
	if !errors.Is(err, domain.ErrLeaseLost) {
		t.Fatalf("stale owner renewal error=%v", err)
	}
	var owner, until string
	if err := rt.Store.DB.QueryRow(`SELECT lease_owner,lease_until FROM transfer_orders WHERE id='o'`).Scan(&owner, &until); err != nil {
		t.Fatal(err)
	}
	if owner != "worker-b" || until != "2030-01-01T01:00:00Z" {
		t.Fatalf("owner=%s lease_until=%s", owner, until)
	}
}
