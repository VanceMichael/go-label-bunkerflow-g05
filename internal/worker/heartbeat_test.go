package worker

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"testing"
	"time"
)

func TestHeartbeatOwnsAndRenewsMessage(t *testing.T) {
	store, err := storage.Open(context.Background(), "file:heartbeat?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := outbox.New(store)
	if err := svc.Enqueue(context.Background(), store.DB, "tenant-zj", "test", "payload"); err != nil {
		t.Fatal(err)
	}
	item, err := svc.Claim(context.Background(), "worker-a", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	heartbeat := NewHeartbeat(store)
	if err := heartbeat.Renew(context.Background(), item.ID, "worker-a", time.Now(), time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := heartbeat.Release(context.Background(), item.ID, "worker-a"); err != nil {
		t.Fatal(err)
	}
	if err := heartbeat.Release(context.Background(), item.ID, "worker-a"); err == nil {
		t.Fatal("released lease twice")
	}
	_ = domain.ErrLeaseLost
}
