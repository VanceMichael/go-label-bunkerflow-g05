package worker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

type Heartbeat struct{ Store *storage.Store }

func NewHeartbeat(store *storage.Store) *Heartbeat { return &Heartbeat{Store: store} }
func (h *Heartbeat) Acquire(ctx context.Context, id, owner string, now time.Time, ttl time.Duration) error {
	if owner == "" || ttl <= 0 {
		return domain.ErrInvalid
	}
	result, err := h.Store.DB.ExecContext(ctx, `UPDATE outbox_messages SET lease_owner=?,lease_until=? WHERE id=? AND status='pending' AND (lease_until IS NULL OR lease_until<?)`, owner, storage.StringTime(now.Add(ttl)), id, storage.StringTime(now))
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return domain.ErrConflict
	}
	return nil
}
func (h *Heartbeat) Renew(ctx context.Context, id, owner string, now time.Time, ttl time.Duration) error {
	result, err := h.Store.DB.ExecContext(ctx, `UPDATE outbox_messages SET lease_until=? WHERE id=? AND lease_owner=? AND lease_until>?`, storage.StringTime(now.Add(ttl)), id, owner, storage.StringTime(now))
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return domain.ErrLeaseLost
	}
	return nil
}
func (h *Heartbeat) Release(ctx context.Context, id, owner string) error {
	result, err := h.Store.DB.ExecContext(ctx, `UPDATE outbox_messages SET lease_owner=NULL,lease_until=NULL WHERE id=? AND lease_owner=?`, id, owner)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return fmt.Errorf("%w: release", domain.ErrLeaseLost)
	}
	return nil
}
func (h *Heartbeat) CountActive(ctx context.Context, owner string) (int, error) {
	var count int
	err := h.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_messages WHERE lease_owner=? AND lease_until>?`, owner, storage.StringTime(time.Now())).Scan(&count)
	return count, err
}

var _ = sql.ErrNoRows
