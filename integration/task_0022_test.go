package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func TestActiveMessageLeaseCannotBeReclaimed(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0022?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err = rt.Store.DB.Exec(`INSERT INTO outbox_messages(id,tenant_id,topic,payload,status,attempts,next_attempt,lease_owner,lease_until,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"m", "tenant-zj", "bunkering.completed", "o", "pending", 1, storage.StringTime(now.Add(-time.Minute)), "worker-a", storage.StringTime(now.Add(time.Minute)), storage.StringTime(now.Add(-time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	if item, err := rt.Outbox.Claim(ctx, "worker-b", now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("active lease reclaimed: item=%+v error=%v", item, err)
	}
	var owner string
	var attempts int
	if err := rt.Store.DB.QueryRow(`SELECT lease_owner,attempts FROM outbox_messages WHERE id='m'`).Scan(&owner, &attempts); err != nil {
		t.Fatal(err)
	}
	if owner != "worker-a" || attempts != 1 {
		t.Fatalf("owner=%s attempts=%d", owner, attempts)
	}
}
