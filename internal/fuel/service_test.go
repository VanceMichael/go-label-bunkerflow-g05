package fuel

import (
	"context"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func TestListLotsIsolatesByTenant(t *testing.T) {
	store, err := storage.Open(context.Background(), "file:fuel-tenant-isolation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	// Seed approved lots for two tenants. tenant-zj gets three so paging is exercised.
	seed := []struct{ id, tenant, lot string }{
		{"lot-zj-1", "tenant-zj", "ZJ-1"},
		{"lot-zj-2", "tenant-zj", "ZJ-2"},
		{"lot-zj-3", "tenant-zj", "ZJ-3"},
		{"lot-fj-1", "tenant-fj", "FJ-1"},
	}
	for i, s := range seed {
		if _, err := store.DB.Exec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES (?,?,?,'green-methanol',1000,'approved',?)`,
			s.id, s.tenant, s.lot, storage.StringTime(now.Add(time.Duration(i)*time.Second))); err != nil {
			t.Fatal(err)
		}
	}

	zj := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	svc := &Service{Store: store}

	// Full list: only tenant-zj rows.
	got, err := svc.ListLots(context.Background(), zj, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d want 3, got=%+v", len(got), got)
	}
	for _, l := range got {
		if l.TenantID != "tenant-zj" {
			t.Fatalf("cross-tenant lot returned: %+v", l)
		}
	}

	// Filtering keeps isolation.
	filtered, err := svc.ListLots(context.Background(), zj, "approved", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 3 {
		t.Fatalf("filtered len=%d want 3", len(filtered))
	}

	// Paging never leaks across tenants, even with a tiny limit.
	page, err := svc.ListLots(context.Background(), zj, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Fatalf("page len=%d want 2", len(page))
	}
	for _, l := range page {
		if l.TenantID != "tenant-zj" {
			t.Fatalf("cross-tenant lot in page: %+v", l)
		}
	}

	// Other tenant sees none of tenant-zj's lots.
	fj := domain.Actor{ID: "user-fj", TenantID: "tenant-fj", Role: "planner"}
	fjGot, err := svc.ListLots(context.Background(), fj, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(fjGot) != 1 || fjGot[0].TenantID != "tenant-fj" {
		t.Fatalf("fj lots=%+v", fjGot)
	}
}
