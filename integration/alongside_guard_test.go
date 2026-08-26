package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/bunkering"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

// TestMarkAlongsideRejectsUnapprovedOrder guards the operation state machine:
// an order that has not been approved (still in the planned state) must not be
// advanced to alongside.
func TestMarkAlongsideRejectsUnapprovedOrder(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:alongside-guard?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	rt.Start(ctx)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	if _, err := rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Store.DB.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	order, err := rt.Bunkering.Create(ctx, actor, bunkering.CreateInput{VesselID: "v", WindowID: "w", FuelLotID: "l", TargetKG: 100, IdempotencyKey: "create-guard"}, "req")
	if err != nil {
		t.Fatal(err)
	}

	// Intentionally skip Approve so the order stays in the planned state.
	if err := rt.Bunkering.MarkAlongside(ctx, actor, order.ID, "req-alongside"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict advancing unapproved order to alongside, got %v", err)
	}

	stored, err := rt.Bunkering.Get(ctx, actor, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.StatePlanned {
		t.Fatalf("order state mutated to %s, expected planned", stored.State)
	}
}
