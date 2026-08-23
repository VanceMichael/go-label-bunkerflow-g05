package vessel

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestRegistrySearchAndExpiringCertificates(t *testing.T) {
	now := time.Now()
	registry := Registry{}
	item := domain.Vessel{ID: "1", TenantID: "tenant-zj", IMO: "9384756", Name: "Atlas", Certificate: domain.Certificate{Verified: true, ExpiresAt: now.Add(24 * time.Hour)}}
	if err := registry.Add(item); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.FindIMO(" 9384756 "); !ok {
		t.Fatal("IMO not found")
	}
	if len(registry.Search("atl")) != 1 {
		t.Fatal("search failed")
	}
	if len(registry.Expiring(now, 48*time.Hour)) != 1 {
		t.Fatal("expiring certificate missing")
	}
	if registry.Add(item) == nil {
		t.Fatal("duplicate vessel accepted")
	}
}
