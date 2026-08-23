package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestCancelledInvoiceGenerationDoesNotCommit(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0023?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	_, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','released',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-23','green-methanol',75,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,25,'completed',1,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-finance", TenantID: "tenant-zj", Role: "finance"}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if invoice, err := rt.Billing.Generate(cancelled, actor, "o", "req-23-cancel"); !errors.Is(err, domain.ErrCancelled) {
		t.Fatalf("cancelled generation invoice=%+v error=%v", invoice, err)
	}
	for table, want := range map[string]int{"invoices": 0, "outbox_messages": 0} {
		var count int
		if err := rt.Store.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s count=%d", table, count)
		}
	}
	invoice, err := rt.Billing.Generate(ctx, actor, "o", "req-23-success")
	if err != nil {
		t.Fatalf("normal generation: %v", err)
	}
	if invoice.State != "issued" || invoice.Amount != 3000 {
		t.Fatalf("normal invoice=%+v", invoice)
	}
}
