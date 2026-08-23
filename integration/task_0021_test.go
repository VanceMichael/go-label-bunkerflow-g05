package integration

import (
	"context"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/app"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/worker"
)

func TestOutboxWorkerShutdownDrainsCurrentMessage(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0021?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	now := time.Now().UTC()
	_, err = rt.Store.DB.Exec(`INSERT INTO outbox_messages(id,tenant_id,topic,payload,status,attempts,next_attempt,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		"m", "tenant-zj", "bunkering.completed", "o", "pending", 0, storage.StringTime(now.Add(-time.Minute)), storage.StringTime(now.Add(-time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	runner := worker.NewRunner(rt.Outbox, "worker-a", func(ctx context.Context, _ domain.OutboxMessage) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}, nil)
	runner.Interval = time.Millisecond
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- runner.Run(runCtx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start current message")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not converge after shutdown")
	}
	var status, owner string
	if err := rt.Store.DB.QueryRow(`SELECT status,COALESCE(lease_owner,'') FROM outbox_messages WHERE id='m'`).Scan(&status, &owner); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || owner != "worker-a" {
		t.Fatalf("message status=%s owner=%s", status, owner)
	}
}
