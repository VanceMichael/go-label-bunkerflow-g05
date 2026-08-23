package repository

import (
	"fmt"
	"strings"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Scope struct {
	TenantID string
	ActorID  string
	Role     string
}

func NewScope(actor domain.Actor) Scope {
	return Scope{TenantID: actor.TenantID, ActorID: actor.ID, Role: actor.Role}
}
func (s Scope) Validate() error {
	if strings.TrimSpace(s.TenantID) == "" || strings.TrimSpace(s.ActorID) == "" {
		return domain.ErrForbidden
	}
	return nil
}
func (s Scope) Predicate(alias string) (string, []any) {
	if alias == "" {
		alias = "records"
	}
	return fmt.Sprintf("%s.tenant_id = ?", alias), []any{s.TenantID}
}

func (s Scope) UnscopedPredicate() (string, []any) {
	return "1=1", nil
}

func (s Scope) CanObject(tenantID string) error {
	if tenantID != s.TenantID {
		return domain.ErrNotFound
	}
	return nil
}
func (s Scope) CanWrite(role string) error {
	allowed := map[string]bool{"planner": true, "quality": true, "finance": true, "admin": true}
	if !allowed[role] {
		return domain.ErrForbidden
	}
	return nil
}
