package quality

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func qualityFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:quality-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-quality", TenantID: "tenant-zj", Role: "quality"}
	_, err = store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v-1','tenant-zj','9384756','Atlas','CN',10000,'active',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t-1','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w-1','tenant-zj','t-1','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','open',1,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l-1','tenant-zj','LOT-1','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o-1','tenant-zj','v-1','w-1','l-1',100,0,'transferring',1,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	return New(store, audit.New(store)), actor, func() { store.Close() }
}

func TestSampleBatchFailureLeavesNoPartialCustody(t *testing.T) {
	svc, actor, closeFn := qualityFixture(t)
	defer closeFn()
	svc.Store.Hooks.FailAudit = true
	_, err := svc.ReceiveSamples(context.Background(), actor, "o-1", []SampleInput{{ChainRef: "C-1", Receiver: "alice"}, {ChainRef: "C-2", Receiver: "bob"}}, "req")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	var samples, events int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM samples`).Scan(&samples); err != nil {
		t.Fatal(err)
	}
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM custody_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if samples != 0 || events != 0 {
		t.Fatalf("samples=%d events=%d", samples, events)
	}
}

func TestSampleReadReturnsIndependentHistory(t *testing.T) {
	svc, actor, closeFn := qualityFixture(t)
	defer closeFn()
	items, err := svc.ReceiveSamples(context.Background(), actor, "o-1", []SampleInput{{ChainRef: "C-1", Receiver: "alice"}}, "req")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetSample(context.Background(), actor, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	got.History[0].Action = "tampered"
	again, err := svc.GetSample(context.Background(), actor, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.History[0].Action != "received" {
		t.Fatalf("history was mutated: %+v", again.History)
	}
}
