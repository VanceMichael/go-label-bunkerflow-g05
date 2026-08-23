package integration

import (
	"context"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestAbortFailurePreservesTransferLease(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0014?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-14','green-methanol',90,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES ('o','tenant-zj','v','w','l',10,0,'transferring','worker-a','2030-01-01T01:00:00Z',2,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	rt.Store.Hooks.FailAudit = true
	err = rt.Bunkering.Abort(ctx, domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}, "o", "req-14")
	rt.Store.Hooks.FailAudit = false
	if err == nil {
		t.Fatal("abort unexpectedly succeeded")
	}
	var state, owner string
	var available float64
	if err := rt.Store.DB.QueryRow(`SELECT state,lease_owner FROM transfer_orders WHERE id='o'`).Scan(&state, &owner); err != nil {
		t.Fatal(err)
	}
	if err := rt.Store.DB.QueryRow(`SELECT available_kg FROM fuel_lots WHERE id='l'`).Scan(&available); err != nil {
		t.Fatal(err)
	}
	if state != "transferring" || owner != "worker-a" || available != 90 {
		t.Fatalf("state=%s owner=%s available=%v", state, owner, available)
	}
}
