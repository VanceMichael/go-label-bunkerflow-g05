package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func TestTerminalArchiveWaitsForOperationalResources(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0029?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-29','green-methanol',75,'approved','2030-01-01');
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','released',1,CURRENT_TIMESTAMP);
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,0,'transferring','worker-a',?,1,CURRENT_TIMESTAMP);
INSERT INTO outbox_messages(id,tenant_id,topic,payload,status,attempts,next_attempt,lease_owner,lease_until,created_at) VALUES ('m','tenant-zj','bunkering.completed','o','pending',1,?,'worker-a',?,CURRENT_TIMESTAMP)`, storage.StringTime(time.Now().Add(time.Minute)), storage.StringTime(time.Now().Add(-time.Minute)), storage.StringTime(time.Now().Add(time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	if err := rt.Terminal.Archive(ctx, actor, "t"); !errors.Is(err, domain.ErrTerminalBusy) {
		t.Fatalf("archive error=%v", err)
	}
	var status string
	if err := rt.Store.DB.QueryRow(`SELECT status FROM terminals WHERE id='t'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("terminal status=%s", status)
	}
}
