package auth

import (
	"fmt"
	"strings"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Policy struct{ Permissions map[string]map[string]bool }

func DefaultPolicy() Policy {
	return Policy{Permissions: map[string]map[string]bool{
		"planner": {"vessel.read": true, "vessel.write": true, "window.read": true, "window.write": true, "bunkering.read": true, "bunkering.write": true, "audit.read": true},
		"quality": {"vessel.read": true, "bunkering.read": true, "quality.read": true, "quality.write": true, "audit.read": true},
		"finance": {"bunkering.read": true, "invoice.read": true, "invoice.write": true, "audit.read": true},
		"admin":   {"vessel.read": true, "vessel.write": true, "window.read": true, "window.write": true, "bunkering.read": true, "bunkering.write": true, "quality.read": true, "quality.write": true, "invoice.read": true, "invoice.write": true, "audit.read": true},
	}}
}
func (p Policy) Allows(actor domain.Actor, permission string) error {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return domain.ErrInvalid
	}
	if p.Permissions[actor.Role][permission] {
		return nil
	}
	return fmt.Errorf("%w: %s cannot %s", domain.ErrForbidden, actor.Role, permission)
}
func (p Policy) RequireAny(actor domain.Actor, permissions ...string) error {
	for _, permission := range permissions {
		if p.Permissions[actor.Role][permission] {
			return nil
		}
	}
	return domain.ErrForbidden
}
func (p Policy) RoleValid(role string) bool { _, ok := p.Permissions[role]; return ok }
