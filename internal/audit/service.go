package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct{ Store *storage.Store }

func New(store *storage.Store) *Service { return &Service{Store: store} }

func (s *Service) Record(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, actor domain.Actor, action, objectID, requestID string) error {
	if s.Store.Hooks.FailAudit {
		return fmt.Errorf("%w: audit sink unavailable", domain.ErrUnavailable)
	}
	_, err := q.ExecContext(ctx, `INSERT INTO audit_events(id, tenant_id, actor_id, action, object_id, request_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), actor.TenantID, actor.ID, action, objectID, requestID, storage.StringTime(time.Now()))
	if err != nil {
		return fmt.Errorf("record audit: %w", err)
	}
	return nil
}

func (s *Service) List(ctx context.Context, actor domain.Actor, limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.Store.DB.QueryContext(ctx, `SELECT id, tenant_id, actor_id, action, object_id, request_id, created_at FROM audit_events WHERE tenant_id=? ORDER BY created_at, id LIMIT ?`, actor.TenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	var result []domain.AuditEvent
	for rows.Next() {
		var item domain.AuditEvent
		var created string
		var actorID sql.NullString
		if err := rows.Scan(&item.ID, &item.TenantID, &actorID, &item.Action, &item.ObjectID, &item.RequestID, &created); err != nil {
			return nil, err
		}
		item.ActorID = actorID.String
		item.CreatedAt, err = storage.ParseTime(created)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
