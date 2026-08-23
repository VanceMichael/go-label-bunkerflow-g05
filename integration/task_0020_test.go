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

func TestPublishFailureKeepsMessageRetryable(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0020?mode=memory&cache=shared"}, nil)
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
	rt.Store.Hooks.FailBroker = true
	item := domain.OutboxMessage{ID: "m", Topic: "bunkering.completed", Payload: "o", Status: "pending", Attempts: 1, LeaseOwner: "worker-a", LeaseUntil: now.Add(time.Minute)}
	if err := rt.Outbox.Publish(ctx, item, "worker-a", now); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("publish error=%v", err)
	}
	var status string
	var owner *string
	if err := rt.Store.DB.QueryRow(`SELECT status,lease_owner FROM outbox_messages WHERE id='m'`).Scan(&status, &owner); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || owner != nil {
		t.Fatalf("status=%s owner=%v", status, owner)
	}
	rt.Store.Hooks.FailBroker = false
	retry, err := rt.Outbox.Claim(ctx, "worker-b", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim retry: %v", err)
	}
	if retry.ID != "m" || retry.LeaseOwner != "worker-b" {
		t.Fatalf("retry=%+v", retry)
	}
}
