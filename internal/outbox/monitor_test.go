package outbox

import (
	"context"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"testing"
)

func TestOutboxHealthReportsPendingBacklog(t *testing.T) {
	store, err := storage.Open(context.Background(), "file:outbox-health?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store)
	if err := svc.Enqueue(context.Background(), store.DB, "tenant-zj", "topic", "payload"); err != nil {
		t.Fatal(err)
	}
	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.Pending != 1 || !health.Healthy() {
		t.Fatalf("health=%+v", health)
	}
}
