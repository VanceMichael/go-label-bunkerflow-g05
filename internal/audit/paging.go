package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

type Cursor struct {
	CreatedAt time.Time
	ID        string
}
type Page struct {
	Items   []domain.AuditEvent
	Next    *Cursor
	HasMore bool
}

func (s *Service) Page(ctx context.Context, actor domain.Actor, after *Cursor, limit int) (Page, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query := `SELECT id,tenant_id,actor_id,action,object_id,request_id,created_at FROM audit_events WHERE tenant_id=?`
	args := []any{actor.TenantID}
	if after != nil {
		query += ` AND (created_at>? OR (created_at=? AND id>?))`
		stamp := storage.StringTime(after.CreatedAt)
		args = append(args, stamp, stamp, after.ID)
	}
	query += ` ORDER BY created_at,id LIMIT ?`
	args = append(args, limit+1)
	rows, err := openPageRows(ctx, s.Store.DB, query, args...)
	if err != nil {
		return Page{}, err
	}
	items := make([]domain.AuditEvent, 0, limit)
	for len(items) < limit && rows.Next() {
		var item domain.AuditEvent
		var actorID sql.NullString
		var created string
		if err := rows.Scan(&item.ID, &item.TenantID, &actorID, &item.Action, &item.ObjectID, &item.RequestID, &created); err != nil {
			return Page{}, err
		}
		item.ActorID = actorID.String
		item.CreatedAt, err = storage.ParseTime(created)
		if err != nil {
			return Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	page := Page{Items: items}
	if len(items) == limit {
		page.HasMore = true
		last := items[len(items)-1]
		page.Next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}
func EncodeCursor(cursor Cursor) string {
	return fmt.Sprintf("%s|%s", storage.StringTime(cursor.CreatedAt), cursor.ID)
}
