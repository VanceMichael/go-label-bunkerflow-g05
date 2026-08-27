package vessel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func newServiceWithHooks(t *testing.T, hooks storage.Hooks) (*Service, context.Context, func()) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.Open(ctx, "file:register-atomic?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	store.Hooks = hooks
	svc := New(store, audit.New(store))
	return svc, ctx, func() { _ = store.Close() }
}

func validInput() RegisterInput {
	return RegisterInput{
		IMO: "9384756", Name: "Green Atlas", Flag: "CN",
		CertificateNumber: "CERT-1",
		ExpiresAt:         time.Now().Add(365 * 24 * time.Hour),
		DeadweightKG:      200000,
		Verified:          true,
	}
}

// TestRegisterLeavesNothingOnAuditFailure reproduces the reported bug: when the
// compliance audit sink is unavailable the whole registration must fail and
// leave no vessel behind, so the caller can retry safely once the dependency
// recovers instead of colliding on the unique IMO of a ghost row.
func TestRegisterLeavesNothingOnAuditFailure(t *testing.T) {
	svc, ctx, done := newServiceWithHooks(t, storage.Hooks{FailAudit: true})
	defer done()
	actor := domain.Actor{ID: "user-planner", TenantID: "tenant-zj", Role: "planner"}

	if _, err := svc.Register(ctx, actor, validInput(), "req-1"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
	if items, err := svc.List(ctx, actor); err != nil || len(items) != 0 {
		t.Fatalf("expected zero vessels after failed register, got %d (err=%v)", len(items), err)
	}

	// Dependency recovers: the same request must now succeed because no partial
	// record was committed to block the unique IMO constraint.
	svc.Audit.Store.Hooks.FailAudit = false
	if _, err := svc.Register(ctx, actor, validInput(), "req-1-retry"); err != nil {
		t.Fatalf("safe retry after recovery failed: %v", err)
	}
	if items, _ := svc.List(ctx, actor); len(items) != 1 {
		t.Fatalf("expected exactly one vessel after retry, got %d", len(items))
	}
}
