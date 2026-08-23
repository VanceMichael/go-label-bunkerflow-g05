package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func billingFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:billing-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-finance", TenantID: "tenant-zj", Role: "finance"}
	if _, err := store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',10000,'active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','open',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-1','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o-1','tenant-zj','v','w','l',100,100,'completed',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO invoices(id,order_id,amount_cents,currency,state,created_at) VALUES ('inv-1','o-1',12000,'USD','issued',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	svc := New(store, audit.New(store), outbox.New(store))
	svc.Gateway = &LocalGateway{}
	return svc, actor, func() { store.Close() }
}

func invoiceState(t *testing.T, svc *Service, id string) string {
	t.Helper()
	var state string
	if err := svc.Store.DB.QueryRow(`SELECT state FROM invoices WHERE id=?`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

func attemptState(t *testing.T, svc *Service, key string) (string, string) {
	t.Helper()
	var state, gatewayRef string
	if err := svc.Store.DB.QueryRow(`SELECT state,COALESCE(gateway_ref,'') FROM payment_attempts WHERE payment_key=?`, key).Scan(&state, &gatewayRef); err != nil {
		t.Fatal(err)
	}
	return state, gatewayRef
}

// TestPayRetryDoesNotDoubleChargeAfterAuditFailure reproduces the reported
// incident: the gateway charge succeeds but local audit fails (the transaction
// rolls back), so finance retries with the same payment key. The gateway must
// be charged exactly once and the invoice must end paid.
func TestPayRetryDoesNotDoubleChargeAfterAuditFailure(t *testing.T) {
	svc, actor, closeFn := billingFixture(t)
	defer closeFn()
	gateway := svc.Gateway.(*LocalGateway)

	// First attempt: gateway succeeds, but local audit fails.
	svc.Store.Hooks.FailAudit = true
	err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-1", "req-1")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("first Pay error = %v", err)
	}
	if gateway.Charges["pay-key-1"] != 1 {
		t.Fatalf("gateway charged %d times on first attempt, want 1", gateway.Charges["pay-key-1"])
	}
	if state := invoiceState(t, svc, "inv-1"); state != "issued" {
		t.Fatalf("invoice state after failed audit = %s, want issued", state)
	}
	if state, ref := attemptState(t, svc, "pay-key-1"); state != "charged" || ref == "" {
		t.Fatalf("attempt state=%s ref=%q, want charged with a ref", state, ref)
	}

	// Retry with the same payment key: audit succeeds now. The gateway must
	// NOT be charged again.
	svc.Store.Hooks.FailAudit = false
	if err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-1", "req-2"); err != nil {
		t.Fatalf("retry Pay error = %v", err)
	}
	if gateway.Charges["pay-key-1"] != 1 {
		t.Fatalf("gateway charged %d times after retry, want 1", gateway.Charges["pay-key-1"])
	}
	if state := invoiceState(t, svc, "inv-1"); state != "paid" {
		t.Fatalf("invoice state after retry = %s, want paid", state)
	}
}

// TestPayRetryReconcilesAfterGatewayFailureThenSuccess verifies that when the
// gateway itself fails, retrying after the outage succeeds with a single charge.
func TestPayRetryReconcilesAfterGatewayFailureThenSuccess(t *testing.T) {
	svc, actor, closeFn := billingFixture(t)
	defer closeFn()
	gateway := svc.Gateway.(*LocalGateway)

	gateway.Fail = true
	if err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-2", "req-1"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("first Pay error = %v", err)
	}
	if gateway.Charges["pay-key-2"] != 1 {
		t.Fatalf("gateway charged %d times on failed attempt, want 1", gateway.Charges["pay-key-2"])
	}
	if state, _ := attemptState(t, svc, "pay-key-2"); state != "failed" {
		t.Fatalf("attempt state after gateway failure = %s, want failed", state)
	}
	if state := invoiceState(t, svc, "inv-1"); state != "issued" {
		t.Fatalf("invoice state after gateway failure = %s, want issued", state)
	}

	gateway.Fail = false
	if err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-2", "req-2"); err != nil {
		t.Fatalf("retry Pay error = %v", err)
	}
	// One failed charge + one successful charge = 2 total gateway calls, but
	// only one *successful* debit. The successful attempt is recorded once.
	if gateway.Charges["pay-key-2"] != 2 {
		t.Fatalf("gateway charged %d times total, want 2", gateway.Charges["pay-key-2"])
	}
	if state, ref := attemptState(t, svc, "pay-key-2"); state != "charged" || ref == "" {
		t.Fatalf("attempt state=%s ref=%q, want charged with a ref", state, ref)
	}
	if state := invoiceState(t, svc, "inv-1"); state != "paid" {
		t.Fatalf("invoice state after retry = %s, want paid", state)
	}
}

// TestPayReusedKeyRejectsDifferentAmount ensures a payment key cannot be
// silently repurposed for a different invoice amount.
func TestPayReusedKeyRejectsDifferentAmount(t *testing.T) {
	svc, actor, closeFn := billingFixture(t)
	defer closeFn()
	// Seed a prior charged attempt for this key at a different amount.
	if _, err := svc.Store.DB.Exec(`INSERT INTO payment_attempts(id,tenant_id,invoice_id,payment_key,amount_cents,state,gateway_ref,created_at) VALUES ('pa-seed','tenant-zj','inv-1','pay-key-3',999,'charged','seeded',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	// The invoice's amount (12000) differs from the seeded attempt (999).
	if err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-3", "req-1"); !errors.Is(err, domain.ErrIdempotency) {
		t.Fatalf("reuse error = %v, want ErrIdempotency", err)
	}
	if state := invoiceState(t, svc, "inv-1"); state != "issued" {
		t.Fatalf("inv-1 state = %s, want issued", state)
	}
}

func TestPayAlreadyPaidIsNoop(t *testing.T) {
	svc, actor, closeFn := billingFixture(t)
	defer closeFn()
	if err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-4", "req-1"); err != nil {
		t.Fatal(err)
	}
	gateway := svc.Gateway.(*LocalGateway)
	charges := gateway.Charges["pay-key-4"]
	if err := svc.Pay(context.Background(), actor, "inv-1", "pay-key-4", "req-2"); err != nil {
		t.Fatalf("second Pay error = %v", err)
	}
	if gateway.Charges["pay-key-4"] != charges {
		t.Fatalf("gateway charged again on paid invoice: %d -> %d", charges, gateway.Charges["pay-key-4"])
	}
	if state := invoiceState(t, svc, "inv-1"); state != "paid" {
		t.Fatalf("invoice state = %s, want paid", state)
	}
}
