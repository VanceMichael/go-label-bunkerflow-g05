package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/billing"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestPaymentRetryDoesNotChargeTwice(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0024?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','released',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-24','green-methanol',75,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,25,'completed',1,CURRENT_TIMESTAMP);
INSERT INTO invoices(id,order_id,amount_cents,currency,state,created_at) VALUES ('i','o',3000,'USD','issued',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-finance", TenantID: "tenant-zj", Role: "finance"}
	gateway := &billing.LocalGateway{}
	rt.Billing.Gateway = gateway
	rt.Store.Hooks.FailAudit = true
	if err := rt.Billing.Pay(ctx, actor, "i", "payment-24", "req-24-first"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("first payment error=%v", err)
	}
	rt.Store.Hooks.FailAudit = false
	if err := rt.Billing.Pay(ctx, actor, "i", "payment-24", "req-24-retry"); err != nil {
		t.Fatalf("payment retry: %v", err)
	}
	if gateway.Charges["payment-24"] != 1 {
		t.Fatalf("gateway charges=%d", gateway.Charges["payment-24"])
	}
	var state, key string
	if err := rt.Store.DB.QueryRow(`SELECT state,COALESCE(payment_key,'') FROM invoices WHERE id='i'`).Scan(&state, &key); err != nil {
		t.Fatal(err)
	}
	if state != "paid" || key != "payment-24" {
		t.Fatalf("invoice state=%s payment_key=%s", state, key)
	}
}
