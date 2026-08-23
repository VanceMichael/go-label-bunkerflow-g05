package integration

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
)

func TestCancelWindowFailurePreservesAssignment(t *testing.T) {
	c := context.Background()
	r, e := app.New(c, app.Config{DatabaseURL: "file:t5?mode=memory&cache=shared"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Shutdown(c)
	r.Store.DB.Exec("INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','N','UTC','00:00','23:59','active',CURRENT_TIMESTAMP)")
	r.Store.DB.Exec("INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,owner_id,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed','owner',1,CURRENT_TIMESTAMP)")
	r.Store.Hooks.FailAudit = true
	if r.Schedule.CancelWindow(c, domain.Actor{TenantID: "tenant-zj"}, "w", "q") == nil {
		t.Fatal("succeeded")
	}
	r.Store.Hooks.FailAudit = false
	var s string
	r.Store.DB.QueryRow("SELECT status FROM bunker_windows WHERE id='w'").Scan(&s)
	if s == "cancelled" {
		t.Fatalf("status=%s", s)
	}
}
