package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

// TestRecoveryResumesFromUnconfirmedStep verifies that recovering a cancelled
// transfer preserves already-confirmed steps (and their checkpoint) and does
// not touch fuel: the reservation is re-acquired when the order re-enters the
// transferring state, so deducting here would double-count.
func TestRecoveryResumesFromUnconfirmedStep(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, "file:recovery-resume?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := store.DB.Exec(query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}
	mustExec(`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',200000,'active',CURRENT_TIMESTAMP)`)
	mustExec(`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`)
	mustExec(`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`)
	mustExec(`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`)
	mustExec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',100,0,'cancelled',5,CURRENT_TIMESTAMP)`)

	confirmedAt := storage.StringTime(time.Now().UTC())
	mustExec(`INSERT INTO transfer_steps(id,order_id,position,name,status,confirmed_at) VALUES ('s1','o',1,'connect','completed',?)`, confirmedAt)
	mustExec(`INSERT INTO transfer_steps(id,order_id,position,name,status,confirmed_at) VALUES ('s2','o',2,'precheck','completed',?)`, confirmedAt)
	mustExec(`INSERT INTO transfer_steps(id,order_id,position,name,status,confirmed_at) VALUES ('s3','o',3,'transfer','pending',NULL)`)
	mustExec(`INSERT INTO transfer_steps(id,order_id,position,name,status,confirmed_at) VALUES ('s4','o',4,'disconnect','pending',NULL)`)

	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	if err := NewRecovery(store).Replay(ctx, actor, "o"); err != nil {
		t.Fatalf("replay: %v", err)
	}

	// Order is back to planned so the operator can resume the workflow.
	var state string
	if err := store.DB.QueryRow(`SELECT state FROM transfer_orders WHERE id='o'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != string(domain.StatePlanned) {
		t.Fatalf("state=%s want planned", state)
	}

	// Confirmed steps and their checkpoint survive; only unconfirmed steps reset to pending.
	rows, err := store.DB.Query(`SELECT position,status,confirmed_at FROM transfer_steps WHERE order_id='o' ORDER BY position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type step struct {
		position    int
		status      string
		confirmedAt *string
	}
	var got []step
	for rows.Next() {
		var s step
		var c *string
		if err := rows.Scan(&s.position, &s.status, &c); err != nil {
			t.Fatal(err)
		}
		s.confirmedAt = c
		got = append(got, s)
	}
	if len(got) != 4 {
		t.Fatalf("steps=%d want 4", len(got))
	}
	for _, s := range got {
		switch s.position {
		case 1, 2:
			if s.status != "completed" {
				t.Errorf("step %d status=%s want completed", s.position, s.status)
			}
			if s.confirmedAt == nil || *s.confirmedAt != confirmedAt {
				t.Errorf("step %d checkpoint lost", s.position)
			}
		case 3, 4:
			if s.status != "pending" {
				t.Errorf("step %d status=%s want pending", s.position, s.status)
			}
			if s.confirmedAt != nil {
				t.Errorf("step %d checkpoint should be cleared", s.position)
			}
		}
	}

	// Fuel balance unchanged: the abort already released the reservation, and
	// StartTransfer re-reserves when the order re-enters transferring.
	var available float64
	if err := store.DB.QueryRow(`SELECT available_kg FROM fuel_lots WHERE id='l'`).Scan(&available); err != nil {
		t.Fatal(err)
	}
	if available != 1000 {
		t.Fatalf("available=%v want 1000 (no double-deduction on recovery)", available)
	}
}

func TestReplayRejectsNonCancelledOperation(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, "file:recovery-reject?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, q := range []string{
		`INSERT INTO vessels(id,tenant_id,imo,name,flag,deadweight_kg,status,created_at) VALUES ('v','tenant-zj','9384756','Atlas','CN',200000,'active',CURRENT_TIMESTAMP)`,
		`INSERT INTO terminals(id,tenant_id,name,timezone,open_from,open_until,status,created_at) VALUES ('t','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`,
		`INSERT INTO bunker_windows(id,tenant_id,terminal_id,starts_at,ends_at,status,version,created_at) VALUES ('w','tenant-zj','t','2026-08-24T00:00:00Z','2026-08-24T02:00:00Z','claimed',1,CURRENT_TIMESTAMP)`,
		`INSERT INTO fuel_lots(id,tenant_id,lot_number,product,available_kg,quality_state,received_at) VALUES ('l','tenant-zj','LOT','green-methanol',1000,'approved','2026-08-23T00:00:00Z')`,
	} {
		if _, err := store.DB.Exec(q); err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
	}
	if _, err := store.DB.Exec(`INSERT INTO transfer_orders(id,tenant_id,vessel_id,window_id,fuel_lot_id,target_kg,transferred_kg,state,version,created_at) VALUES ('o','tenant-zj','v','w','l',100,0,'transferring',1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	err = NewRecovery(store).Replay(ctx, actor, "o")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err=%v want ErrConflict", err)
	}
}
