package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

// TestClaimRespectsActiveLease guards against the regression where a second
// worker re-claims a message that another worker still holds under a valid
// lease, which would cause duplicate delivery.
func TestClaimRespectsActiveLease(t *testing.T) {
	store, err := storage.Open(context.Background(), "file:outbox-lease-race?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store)
	if err := svc.Enqueue(context.Background(), store.DB, "tenant-zj", "topic", "payload"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	// First worker claims the only available message.
	first, err := svc.Claim(context.Background(), "worker-a", now)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	// The lease is still active, so a second worker must not re-claim it.
	if _, err := svc.Claim(context.Background(), "worker-b", now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second claim while lease active: err=%v want ErrNotFound", err)
	}
	// After the lease expires the message becomes claimable again.
	expired := first.LeaseUntil.Add(time.Second)
	if _, err := svc.Claim(context.Background(), "worker-b", expired); err != nil {
		t.Fatalf("claim after lease expiry: %v", err)
	}
}
