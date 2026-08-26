package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/operations"
)

func TestBatchCompleteRejectsWithoutPartialCommit(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0028?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','released',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-28','green-methanol',75,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o1','tenant-zj','v','w','l',25,25,'sampled',1,CURRENT_TIMESTAMP),('o2','tenant-zj','v','w','l',25,25,'transferring',1,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	if err := operations.New(rt.Store).BatchComplete(ctx, actor, []string{"o1", "o2"}, "req-28"); !errors.Is(err, domain.ErrNoQuality) {
		t.Fatalf("batch error=%v", err)
	}
	for id := range map[string]struct{}{"o1": {}, "o2": {}} {
		var state string
		if err := rt.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id=?`, id).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state != map[string]string{"o1": "sampled", "o2": "transferring"}[id] {
			t.Fatalf("%s state=%s", id, state)
		}
	}
}
