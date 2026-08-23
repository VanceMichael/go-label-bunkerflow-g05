package integration

import (
	"context"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/quality"
)

func TestSampleBatchFailureLeavesNoPartialCustody(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0015?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	if _, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-15','green-methanol',100,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,0,'transferring',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	rt.Store.Hooks.FailAudit = true
	rt.Store.Hooks.SamplesPrelude = make(chan struct{})
	_, err = rt.Quality.ReceiveSamples(ctx, domain.Actor{ID: "user-quality", TenantID: "tenant-zj"}, "o", []quality.SampleInput{{ChainRef: "C-1", Receiver: "alice"}, {ChainRef: "C-2", Receiver: "bob"}}, "req-15")
	if err == nil {
		t.Fatal("sample batch succeeded")
	}
	var samples, events int
	if err := rt.Store.DB.QueryRow(`SELECT COUNT(*) FROM samples`).Scan(&samples); err != nil {
		t.Fatal(err)
	}
	if err := rt.Store.DB.QueryRow(`SELECT COUNT(*) FROM custody_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if samples != 0 || events != 0 {
		t.Fatalf("samples=%d events=%d", samples, events)
	}
}
