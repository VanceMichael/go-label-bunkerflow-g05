package integration

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestCancelledCertificateReplacementPreservesSnapshot(t *testing.T) {
	c := context.Background()
	r, e := app.New(c, app.Config{DatabaseURL: "file:t2?mode=memory&cache=shared"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Shutdown(c)
	r.Store.DB.Exec("INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','A','CN',1,'active',CURRENT_TIMESTAMP)")
	r.Store.DB.Exec("INSERT INTO vessel_certificates(id,vessel_id,number,expires_at,verified,created_at) VALUES ('c','v','OLD','2030-01-01T00:00:00Z',1,CURRENT_TIMESTAMP)")
	x, cancel := context.WithCancel(c)
	cancel()
	e = r.Vessel.ReplaceCertificate(x, domain.Actor{TenantID: "tenant-zj"}, "v", "NEW", time.Now().Add(time.Hour), true, "q")
	if e == nil {
		t.Fatal("cancel succeeded")
	}
	var n string
	r.Store.DB.QueryRow("SELECT number FROM vessel_certificates WHERE id='c'").Scan(&n)
	if n != "OLD" {
		t.Fatalf("number=%s", n)
	}
}
