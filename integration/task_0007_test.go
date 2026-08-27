package integration

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
)

func TestFuelLotListingNeverCrossesTenantBoundary(t *testing.T) {
	c := context.Background()
	r, e := app.New(c, app.Config{DatabaseURL: "file:t7?mode=memory&cache=shared"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Shutdown(c)
	r.Store.DB.Exec("INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('a','tenant-zj','A','green-methanol',1,'approved','2030-01-01T00:00:00Z'),('b','tenant-fj','B','green-methanol',1,'approved','2030-01-02T00:00:00Z')")
	x, e := r.Fuel.ListLots(c, domain.Actor{TenantID: "tenant-zj"}, "", 50)
	if e != nil {
		t.Fatal(e)
	}
	if len(x) != 1 || x[0].TenantID != "tenant-zj" {
		t.Fatalf("lots=%+v", x)
	}
}
