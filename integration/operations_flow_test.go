package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/bunkering"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/quality"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/schedule"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/terminal"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/vessel"
)

func TestCompleteBunkeringFlowPersistsEveryLifecycleBoundary(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:complete-flow?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	rt.Start(ctx)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	terminalItem, err := rt.Terminal.Create(ctx, actor, terminal.CreateInput{Name: "Ningbo Transfer Anchorage", Timezone: "Asia/Shanghai", OpenFrom: "00:00", OpenUntil: "23:59"})
	if err != nil {
		t.Fatal(err)
	}
	vesselItem, err := rt.Vessel.Register(ctx, actor, vessel.RegisterInput{IMO: "9384756", Name: "Green Atlas", Flag: "CN", CertificateNumber: "CERT-1", ExpiresAt: time.Now().Add(365 * 24 * time.Hour), DeadweightKG: 200000, Verified: true}, "req-vessel")
	if err != nil {
		t.Fatal(err)
	}
	lot, err := rt.Fuel.ReceiveLot(ctx, actor, fuel.ReceiveInput{LotNumber: "ZJ-MEOH-001", Product: "green-methanol", QuantityKG: 5000, ReceivedAt: time.Now(), Quality: domain.QualityApproved}, "req-lot")
	if err != nil {
		t.Fatal(err)
	}
	window, err := rt.Schedule.CreateWindow(ctx, actor, schedule.WindowInput{TerminalID: terminalItem.ID, StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(3 * time.Hour)}, "req-window")
	if err != nil {
		t.Fatal(err)
	}
	order, err := rt.Bunkering.Create(ctx, actor, bunkering.CreateInput{VesselID: vesselItem.ID, WindowID: window.ID, FuelLotID: lot.ID, TargetKG: 1000, IdempotencyKey: "create-order-1"}, "req-order")
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Bunkering.Approve(ctx, actor, order.ID, "req-approve"); err != nil {
		t.Fatal(err)
	}
	if err := rt.Bunkering.MarkAlongside(ctx, actor, order.ID, "req-alongside"); err != nil {
		t.Fatal(err)
	}
	if err := rt.Bunkering.StartTransfer(ctx, actor, order.ID, "worker-a", "req-start"); err != nil {
		t.Fatal(err)
	}
	for position := 1; position <= 4; position++ {
		if err := rt.Bunkering.CompleteStep(ctx, actor, order.ID, position, "req-step"); err != nil {
			t.Fatalf("step %d: %v", position, err)
		}
	}
	samples, err := rt.Quality.ReceiveSamples(ctx, actor, order.ID, []quality.SampleInput{{ChainRef: "CHAIN-1", Receiver: "quality-lab"}}, "req-sample")
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Quality.Review(ctx, actor, samples[0].ID, true, "req-review"); err != nil {
		t.Fatal(err)
	}
	invoice, err := rt.Bunkering.Complete(ctx, actor, order.ID, "req-complete")
	if err != nil {
		t.Fatal(err)
	}
	if invoice.State != "issued" || invoice.Amount <= 0 {
		t.Fatalf("invoice=%+v", invoice)
	}
	stored, err := rt.Bunkering.Get(ctx, actor, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.StateCompleted || stored.TransferredKG != 1000 {
		t.Fatalf("stored=%+v", stored)
	}
	lotAfter, err := rt.Fuel.Get(ctx, actor, lot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if lotAfter.AvailableKG != 4000 {
		t.Fatalf("available=%v", lotAfter.AvailableKG)
	}
	events, err := rt.Audit.List(ctx, actor, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 10 {
		t.Fatalf("audit events=%d", len(events))
	}
}

func TestCancelledStartTransferPreservesStateAndFuel(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:cancel-start?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	rt.Start(ctx)
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Store.DB.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.Store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	order, err := rt.Bunkering.Create(ctx, actor, bunkering.CreateInput{VesselID: "v", WindowID: "w", FuelLotID: "l", TargetKG: 100}, "req")
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Bunkering.Approve(ctx, actor, order.ID, "req"); err != nil {
		t.Fatal(err)
	}
	if err := rt.Bunkering.MarkAlongside(ctx, actor, order.ID, "req"); err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := rt.Bunkering.StartTransfer(cancelled, actor, order.ID, "worker", "req"); !errors.Is(err, domain.ErrCancelled) {
		t.Fatalf("error=%v", err)
	}
	stored, err := rt.Bunkering.Get(ctx, actor, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.StateAlongside {
		t.Fatalf("state=%s", stored.State)
	}
	lot, err := rt.Fuel.Get(ctx, actor, "l")
	if err != nil {
		t.Fatal(err)
	}
	if lot.AvailableKG != 1000 {
		t.Fatalf("available=%v", lot.AvailableKG)
	}
}

func TestTenantCannotReadAnotherTenantResources(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:tenant-isolation?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	rt.Start(ctx)
	zj := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	fj := domain.Actor{ID: "user-fj", TenantID: "tenant-fj", Role: "planner"}
	v, err := rt.Vessel.Register(ctx, zj, vessel.RegisterInput{IMO: "9384756", Name: "Atlas", Flag: "CN", CertificateNumber: "C", ExpiresAt: time.Now().Add(time.Hour), DeadweightKG: 1000, Verified: true}, "req")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Vessel.Get(ctx, fj, v.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross tenant error=%v", err)
	}
	items, err := rt.Vessel.List(ctx, fj)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("cross tenant list=%+v", items)
	}
}
