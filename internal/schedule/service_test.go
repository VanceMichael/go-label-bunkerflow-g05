package schedule

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func scheduleFixture(t *testing.T) (*Service, domain.Actor, func()) {
	t.Helper()
	store, err := storage.Open(context.Background(), "file:schedule-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}
	_, err = store.DB.Exec(`INSERT INTO terminals(id, tenant_id, name, timezone, open_from, open_until, status, created_at) VALUES ('terminal-1','tenant-zj','Ningbo','Asia/Shanghai','00:00','23:59','active',CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	return New(store, audit.New(store), outbox.New(store)), actor, func() { store.Close() }
}

func TestCreateWindowIsAtomicWhenAuditFails(t *testing.T) {
	svc, actor, closeFn := scheduleFixture(t)
	defer closeFn()
	svc.Store.Hooks.FailAudit = true
	_, err := svc.CreateWindow(context.Background(), actor, WindowInput{TerminalID: "terminal-1", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-1")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	var count int
	if err := svc.Store.DB.QueryRow(`SELECT COUNT(*) FROM bunker_windows`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("windows = %d", count)
	}
}

func TestConcurrentWindowClaimHasOneOwner(t *testing.T) {
	svc, actor, closeFn := scheduleFixture(t)
	defer closeFn()
	item, err := svc.CreateWindow(context.Background(), actor, WindowInput{TerminalID: "terminal-1", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(2 * time.Hour)}, "req-2")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, owner := range []string{"a", "b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			results <- svc.ClaimWindow(context.Background(), actor, item.ID, owner, "req-claim-"+owner)
		}(owner)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("claim error = %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful claims = %d", success)
	}
}
