package integration

import (
	"context"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/worker"
)

func TestRecoveryResumesAfterConfirmedTransferStep(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0027?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','released',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-27','green-methanol',75,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,25,'cancelled',1,CURRENT_TIMESTAMP);
INSERT INTO transfer_steps(id,order_id,position,name,status,confirmed_at) VALUES ('step','o',3,'transfer','completed','2030-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	if err := worker.NewRecovery(rt.Store).Replay(ctx, actor, "o"); err != nil {
		t.Fatal(err)
	}
	var available float64
	if err := rt.Store.DB.QueryRow(`SELECT available_kg FROM fuel_lots WHERE id='l'`).Scan(&available); err != nil {
		t.Fatal(err)
	}
	if available != 75 {
		t.Fatalf("available fuel=%v", available)
	}
	var status string
	if err := rt.Store.DB.QueryRow(`SELECT status FROM transfer_steps WHERE id='step'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatalf("step status=%s", status)
	}
}
