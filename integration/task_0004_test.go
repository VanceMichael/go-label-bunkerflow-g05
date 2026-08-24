package integration

import (
	"context"
	"testing"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestWindowClaimAuditFailurePreservesOpenWindow(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0004?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	if _, err = rt.Store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','open',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	rt.Store.Hooks.FailAudit = true
	if err := rt.Schedule.ClaimWindow(ctx, domain.Actor{ID: "u", TenantID: "tenant-zj"}, "w", "owner-a", "req-4"); err == nil {
		t.Fatal("claim succeeded")
	}
	var status, owner string
	if err := rt.Store.DB.QueryRow(`SELECT status,COALESCE(owner_id,'') FROM bunker_windows WHERE id='w'`).Scan(&status, &owner); err != nil {
		t.Fatal(err)
	}
	if status != "open" || owner != "" {
		t.Fatalf("status=%s owner=%s", status, owner)
	}
}
