package integration

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/fuel"
	"testing"
	"time"
)

func TestReceiveLotRollsBackAfterAuditError(t *testing.T) {
	c := context.Background()
	r, e := app.New(c, app.Config{DatabaseURL: "file:t6?mode=memory&cache=shared"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Shutdown(c)
	r.Store.Hooks.FailAudit = true
	_, e = r.Fuel.ReceiveLot(c, domain.Actor{ID: "u", TenantID: "tenant-zj"}, fuel.ReceiveInput{LotNumber: "L", Product: "green-methanol", QuantityKG: 10, ReceivedAt: time.Now(), Quality: domain.QualityApproved}, "q")
	if e == nil {
		t.Fatal("succeeded")
	}
	r.Store.Hooks.FailAudit = false
	var n int
	r.Store.DB.QueryRow("SELECT COUNT(*) FROM fuel_lots WHERE tenant_id='tenant-zj'").Scan(&n)
	if n != 0 {
		t.Fatalf("lots=%d", n)
	}
}
