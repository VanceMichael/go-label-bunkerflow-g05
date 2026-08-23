package idempotency

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
)

func TestFingerprintBindsTenantKeyAndBody(t *testing.T) {
	actor := domain.Actor{TenantID: "tenant-zj"}
	f, err := NewFingerprint(actor, "key", map[string]any{"target": 100})
	if err != nil {
		t.Fatal(err)
	}
	if !f.Matches(actor, "key", map[string]any{"target": 100}) {
		t.Fatal("matching request rejected")
	}
	if f.Matches(actor, "key", map[string]any{"target": 200}) {
		t.Fatal("changed body matched")
	}
	if f.Matches(domain.Actor{TenantID: "tenant-fj"}, "key", map[string]any{"target": 100}) {
		t.Fatal("cross tenant fingerprint matched")
	}
}
