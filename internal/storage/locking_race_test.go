package storage

import (
	"context"
	"testing"
	"time"
)

// TestRenewOrderLeaseRequiresCurrentOwner guards against the ownership race where
// a previous lease holder arrives late to renew after its lease was taken over by
// another worker. The renew must match on lease_owner so the stale holder cannot
// overwrite the new holder's lease_until.
func TestRenewOrderLeaseRequiresCurrentOwner(t *testing.T) {
	store, err := Open(context.Background(), "file:renew-lease-race?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	orderID := "order-renew"
	if _, err := store.DB.ExecContext(ctx, `INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES (?,?,?,?,?,?,?,?)`, "v", "tenant-zj", "9384756", "Atlas", "CN", 1000, "active", StringTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.ExecContext(ctx, `INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES (?,?,?,?,?,?,?,?)`, "t", "tenant-zj", "Ningbo", "Asia/Shanghai", "00:00", "23:59", "active", StringTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.ExecContext(ctx, `INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES (?,?,?,?,?,?,?,?)`, "w", "tenant-zj", "t", StringTime(now.Add(time.Hour)), StringTime(now.Add(2*time.Hour)), "claimed", 1, StringTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.ExecContext(ctx, `INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES (?,?,?,?,?,?,?)`, "l", "tenant-zj", "LOT", "green-methanol", 1000, "approved", StringTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.ExecContext(ctx, `INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,lease_owner,lease_until,version,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, orderID, "tenant-zj", "v", "w", "l", 100, 0, "transferring", "worker-a", StringTime(now.Add(5*time.Minute)), 1, StringTime(now)); err != nil {
		t.Fatal(err)
	}

	// The new holder takes over the lease.
	if _, err := store.DB.ExecContext(ctx, `UPDATE transfer_orders SET lease_owner='worker-b', lease_until=?, version=version+1 WHERE id=?`, StringTime(now.Add(5*time.Minute)), orderID); err != nil {
		t.Fatal(err)
	}

	// The previous holder (worker-a) tries to renew late. It must be rejected.
	result, err := RenewOrderLease(ctx, store.DB, orderID, "tenant-zj", "worker-a", now, now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if n, _ := result.RowsAffected(); n != 0 {
		t.Fatalf("stale holder renewed a taken-over lease: rows=%d", n)
	}

	// The current holder (worker-b) can renew while the lease is active.
	result, err = RenewOrderLease(ctx, store.DB, orderID, "tenant-zj", "worker-b", now, now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("renew current: %v", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		t.Fatalf("current holder could not renew: rows=%d", n)
	}

	var owner, until string
	if err := store.DB.QueryRowContext(ctx, `SELECT lease_owner, lease_until FROM transfer_orders WHERE id=?`, orderID).Scan(&owner, &until); err != nil {
		t.Fatal(err)
	}
	if owner != "worker-b" {
		t.Fatalf("owner overwritten to %s", owner)
	}
	if parsed, perr := ParseTime(until); perr != nil || !parsed.After(now.Add(9*time.Minute)) {
		t.Fatalf("lease_until not extended: %v", until)
	}
}
