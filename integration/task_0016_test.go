package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func TestSampleReadHonorsCancelledCustodyLookup(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0016?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	if _, err = rt.Store.DB.Exec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',1000,'active',CURRENT_TIMESTAMP);
INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','UTC','00:00','23:59','active',CURRENT_TIMESTAMP);
INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2030-01-01','2030-01-02','claimed',1,CURRENT_TIMESTAMP);
INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT-16','green-methanol',100,'approved','2030-01-01');
INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',25,0,'sampled',1,CURRENT_TIMESTAMP);
INSERT INTO samples(id,order_id,chain_ref,receiver,quality_state,created_at) VALUES ('s','o','CHAIN-16','lab','received',CURRENT_TIMESTAMP);
INSERT INTO custody_events(id,sample_id,actor_id,action,created_at) VALUES ('c','s','user-quality','received',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	rt.Store.Hooks.CustodyReadStarted = started
	rt.Store.Hooks.CustodyReadRelease = release
	cancelled, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := rt.Quality.GetSample(cancelled, domain.Actor{ID: "user-quality", TenantID: "tenant-zj"}, "s")
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lookup error=%v", err)
		}
	case <-time.After(100 * time.Millisecond):
		close(release)
		<-done
		t.Fatal("cancelled lookup did not converge")
	}
}
