package integration

import (
	"context"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestApprovalAuditFailureLeavesPlanPending(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0009?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	if _, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-9','green-methanol',75,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,0,'planned',1,CURRENT_TIMESTAMP);
INSERT INTO safety_permits(id,order_id,status,created_at) VALUES ('p','o','pending',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	rt.Store.Hooks.FailAudit = true
	if err := rt.Bunkering.Approve(ctx, domain.Actor{ID: "user-planner", TenantID: "tenant-zj"}, "o", "req-9"); err == nil {
		t.Fatal("approval succeeded")
	}
	var state, permit string
	if err := rt.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id='o'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := rt.Store.DB.QueryRow(`SELECT status FROM safety_permits WHERE id='p'`).Scan(&permit); err != nil {
		t.Fatal(err)
	}
	if state != string(domain.StatePlanned) || permit != "pending" {
		t.Fatalf("state=%s permit=%s", state, permit)
	}
}
