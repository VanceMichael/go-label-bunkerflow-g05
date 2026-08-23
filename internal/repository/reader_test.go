package repository

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"testing"
)

func TestScopePreventsCrossTenantAccess(t *testing.T) {
	scope := NewScope(domain.Actor{ID: "a", TenantID: "tenant-zj", Role: "planner"})
	if err := scope.Validate(); err != nil {
		t.Fatal(err)
	}
	predicate, args := scope.Predicate("v")
	if predicate != "v.tenant_id = ?" || len(args) != 1 || args[0] != "tenant-zj" {
		t.Fatalf("predicate=%s args=%v", predicate, args)
	}
	if err := scope.CanObject("tenant-fj"); err == nil {
		t.Fatal("cross tenant object allowed")
	}
	_ = storage.StringTime
}
