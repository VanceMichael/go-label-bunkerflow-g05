package auth

import (
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
)

func TestRolePolicySeparatesOperationalResponsibilities(t *testing.T) {
	p := DefaultPolicy()
	planner := domain.Actor{Role: "planner"}
	quality := domain.Actor{Role: "quality"}
	if p.Allows(planner, "window.write") != nil {
		t.Fatal("planner cannot write windows")
	}
	if p.Allows(quality, "window.write") == nil {
		t.Fatal("quality wrote windows")
	}
	if p.Allows(quality, "quality.write") != nil {
		t.Fatal("quality cannot review")
	}
	if !p.RoleValid("finance") || p.RoleValid("unknown") {
		t.Fatal("role policy invalid")
	}
}
