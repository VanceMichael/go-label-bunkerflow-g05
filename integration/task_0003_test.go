package integration

import (
	"context"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/schedule"
)

func TestWindowCreationIsAtomicAcrossAuditFailure(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0003?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	if _, err = rt.Store.DB.Exec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	rt.Store.Hooks.FailAudit = true
	rt.Store.Hooks.WindowPrelude = make(chan struct{})
	_, err = rt.Schedule.CreateWindow(ctx, domain.Actor{ID: "u", TenantID: "tenant-zj"}, schedule.WindowInput{TerminalID: "t", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-3")
	if err == nil {
		t.Fatal("succeeded")
	}
	var count int
	if err := rt.Store.DB.QueryRow(`SELECT COUNT(*) FROM bunker_windows`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("windows=%d", count)
	}
}
