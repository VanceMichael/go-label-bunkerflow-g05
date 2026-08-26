package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestCompletionRollsBackWhenOutboxEnqueueFails(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0019?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-19','green-methanol',100,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,0,'sampled',1,CURRENT_TIMESTAMP);
INSERT INTO samples(id,order_id,chain_ref,receiver,quality_state,created_at) VALUES ('s','o','CHAIN-19','lab','approved',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-quality", TenantID: "tenant-zj", Role: "quality"}
	rt.Store.Hooks.FailOutbox = true
	if _, err := rt.Bunkering.Complete(ctx, actor, "o", "req-19-fail"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("completion error=%v", err)
	}
	var state string
	if err := rt.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id='o'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != string(domain.StateSampled) {
		t.Fatalf("order state=%s", state)
	}
	var invoices int
	if err := rt.Store.DB.QueryRow(`SELECT COUNT(*) FROM invoices WHERE order_id='o'`).Scan(&invoices); err != nil {
		t.Fatal(err)
	}
	if invoices != 0 {
		t.Fatalf("invoices after failed completion=%d", invoices)
	}
	rt.Store.Hooks.FailOutbox = false
	invoice, err := rt.Bunkering.Complete(ctx, actor, "o", "req-19-retry")
	if err != nil {
		t.Fatalf("completion retry: %v", err)
	}
	if invoice.State != "issued" {
		t.Fatalf("retry invoice=%+v", invoice)
	}
}
