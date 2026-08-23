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

func TestReadinessAndShutdownConverge(t *testing.T) {
	ctx := context.Background()
	rt, err := app.New(ctx, app.Config{DatabaseURL: "file:task-0030?mode=memory&cache=shared"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	_, err = rt.Store.DB.Exec(`INSERT INTO outbox_messages(id,tenant_id,topic,payload,status,attempts,next_attempt,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		"m", "tenant-zj", "bunkering.completed", "o", "pending", 0, storage.StringTime(time.Now().Add(-time.Minute)), storage.StringTime(time.Now().Add(-time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	rt.Worker.Broker = func(context.Context, domain.OutboxMessage) error {
		close(started)
		<-release
		return nil
	}
	rt.Worker.Interval = time.Millisecond
	rt.Start(ctx)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not enter publish")
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	err = rt.Shutdown(shutdownCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error=%v", err)
	}
	if rt.Ready() {
		t.Fatal("runtime remained ready after shutdown timeout")
	}
	close(release)
}
