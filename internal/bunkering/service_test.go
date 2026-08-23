package bunkering

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/idempotency"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/quality"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func completeFixture(t *testing.T) (*Service, *quality.Service, domain.Actor, string, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:bunkering-complete-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	for _, stmt := range []string{
		`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v-1','tenant-zj','9384756','Atlas','CN',10000,'active',CURRENT_TIMESTAMP)`,
		`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t-1','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`,
		`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w-1','tenant-zj','t-1','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','open',1,CURRENT_TIMESTAMP)`,
		`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l-1','tenant-zj','LOT-1','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`,
		`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o-1','tenant-zj','v-1','w-1','l-1',100,0,'sampled',1,CURRENT_TIMESTAMP)`,
		`INSERT INTO transfer_steps(id,order_id,position,name,status) VALUES ('s-1','o-1',1,'connect','completed'),('s-2','o-1',2,'precheck','completed'),('s-3','o-1',3,'transfer','completed'),('s-4','o-1',4,'disconnect','completed')`,
		`INSERT INTO samples(id,order_id,chain_ref,receiver,quality_state,created_at) VALUES ('sam-1','o-1','CHAIN-1','quality-lab','approved','2026-08-23T00:00:00Z')`,
	} {
		if _, err := store.DB.Exec(stmt); err != nil {
			t.Fatalf("seed %q: %v", stmt, err)
		}
	}
	auditSvc := audit.New(store)
	outboxSvc := outbox.New(store)
	fuelSvc := fuel.New(store, auditSvc)
	qualitySvc := quality.New(store, auditSvc)
	idempotencySvc := idempotency.New(store)
	svc := New(store, auditSvc, outboxSvc, fuelSvc, qualitySvc, idempotencySvc)
	return svc, qualitySvc, actor, "o-1", func() { store.Close() }
}

// TestCompleteRollsBackWhenOutboxFails reproduces the cross-entity rollback bug:
// when the outbox enqueue fails during Complete, the order state, invoice and
// audit event must all roll back atomically so a retry converges.
func TestCompleteRollsBackWhenOutboxFails(t *testing.T) {
	svc, _, actor, orderID, closeFn := completeFixture(t)
	defer closeFn()
	ctx := context.Background()

	svc.Store.Hooks.FailOutbox = true
	if _, err := svc.Complete(ctx, actor, orderID, "req-complete"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}

	var state string
	if err := svc.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id=?`, orderID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != string(domain.StateSampled) {
		t.Fatalf("order state = %s, want %s", state, domain.StateSampled)
	}

	var invoiceCount int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM invoices WHERE order_id=?`, orderID).Scan(&invoiceCount); err != nil {
		t.Fatal(err)
	}
	if invoiceCount != 0 {
		t.Fatalf("invoices = %d, want 0", invoiceCount)
	}

	var auditCount int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action='bunkering.completed'`).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 0 {
		t.Fatalf("audit events = %d, want 0", auditCount)
	}

	// Retry must converge once the outbox is available again.
	svc.Store.Hooks.FailOutbox = false
	invoice, err := svc.Complete(ctx, actor, orderID, "req-complete-retry")
	if err != nil {
		t.Fatalf("retry error = %v", err)
	}
	if invoice.State != "issued" || invoice.Amount <= 0 {
		t.Fatalf("invoice = %+v", invoice)
	}
	stored, err := svc.Get(ctx, actor, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.StateCompleted || stored.TransferredKG != 100 {
		t.Fatalf("stored = %+v", stored)
	}
}

// TestCompleteRollsBackWhenAuditFails mirrors the rollback guarantee for the
// audit sink within the same single transaction.
func TestCompleteRollsBackWhenAuditFails(t *testing.T) {
	svc, _, actor, orderID, closeFn := completeFixture(t)
	defer closeFn()
	ctx := context.Background()

	svc.Store.Hooks.FailAudit = true
	if _, err := svc.Complete(ctx, actor, orderID, "req-complete"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}

	var state string
	if err := svc.Store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id=?`, orderID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != string(domain.StateSampled) {
		t.Fatalf("order state = %s, want %s", state, domain.StateSampled)
	}

	var invoiceCount int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM invoices WHERE order_id=?`, orderID).Scan(&invoiceCount); err != nil {
		t.Fatal(err)
	}
	if invoiceCount != 0 {
		t.Fatalf("invoices = %d, want 0", invoiceCount)
	}

	// Retry converges once the audit sink recovers.
	svc.Store.Hooks.FailAudit = false
	_, err := svc.Complete(ctx, actor, orderID, "req-complete-retry")
	if err != nil {
		t.Fatalf("retry error = %v", err)
	}
	if _, err := svc.Get(ctx, actor, orderID); err != nil {
		t.Fatal(err)
	}
}
