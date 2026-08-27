package integration

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/vessel"
	"testing"
	"time"
)

func TestRegisterRollsBackWhenAuditFails(t *testing.T) {
	c := context.Background()
	r, e := app.New(c, app.Config{DatabaseURL: "file:t1?mode=memory&cache=shared"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Shutdown(c)
	r.Store.Hooks.FailAudit = true
	_, e = r.Vessel.Register(c, domain.Actor{ID: "u", TenantID: "tenant-zj"}, vessel.RegisterInput{IMO: "9384756", Name: "A", Flag: "CN", CertificateNumber: "C", ExpiresAt: time.Now().Add(time.Hour), DeadweightKG: 1, Verified: true}, "q")
	if e == nil {
		t.Fatal("succeeded")
	}
	r.Store.Hooks.FailAudit = false
	var n int
	r.Store.DB.QueryRow("SELECT COUNT(*) FROM vessels WHERE tenant_id='tenant-zj'").Scan(&n)
	if n != 0 {
		t.Fatalf("vessels=%d", n)
	}
}
