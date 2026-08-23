package integration

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/bunkering"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
)

func TestCreateBunkeringReplayReturnsOriginalResponse(t *testing.T) {
	c := context.Background()
	r, e := app.New(c, app.Config{DatabaseURL: "file:t8?mode=memory&cache=shared"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Shutdown(c)
	_, e = r.Store.DB.Exec("INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','A','CN',100,'active',CURRENT_TIMESTAMP);INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','T','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','open',1,CURRENT_TIMESTAMP);INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','L','green-methanol',100,'approved','2030-01-01')")
	if e != nil {
		t.Fatal(e)
	}
	a := domain.Actor{ID: "u", TenantID: "tenant-zj"}
	in := bunkering.CreateInput{VesselID: "v", WindowID: "w", FuelLotID: "l", TargetKG: 10, IdempotencyKey: "same"}
	x, e := r.Bunkering.Create(c, a, in, "1")
	if e != nil {
		t.Fatal(e)
	}
	y, e := r.Bunkering.Create(c, a, in, "2")
	if e != nil {
		t.Fatal(e)
	}
	if x.ID != y.ID {
		t.Fatalf("ids %s %s", x.ID, y.ID)
	}
}
