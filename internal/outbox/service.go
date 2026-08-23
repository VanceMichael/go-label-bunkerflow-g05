package outbox

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

func (s *Service) Enqueue(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, tenantID, topic, payload string) error {
	if s.Store.Hooks.FailOutbox {
		return fmt.Errorf("%w: outbox unavailable", domain.ErrUnavailable)
	}
	now := time.Now().UTC()
	_, err := q.ExecContext(ctx, `INSERT INTO outbox_messages(id, tenant_id, topic, payload, status, attempts, next_attempt, created_at) VALUES (?, ?, ?, ?, 'pending', 0, ?, ?)`, uuid.NewString(), tenantID, topic, payload, storage.StringTime(now), storage.StringTime(now))
	if err != nil {
		return fmt.Errorf("enqueue outbox: %w", err)
	}
	return nil
}

func (s *Service) Claim(ctx context.Context, worker string, now time.Time) (domain.OutboxMessage, error) {
	tx, err := s.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.OutboxMessage{}, err
	}
	defer tx.Rollback()
	var item domain.OutboxMessage
	var due, until, last sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT id, topic, payload, status, attempts, next_attempt, lease_until, last_error FROM outbox_messages WHERE status='pending' AND next_attempt<=? AND (lease_until IS NULL OR lease_until<?) ORDER BY next_attempt, id LIMIT 1`, storage.StringTime(now), storage.StringTime(now)).Scan(&item.ID, &item.Topic, &item.Payload, &item.Status, &item.Attempts, &due, &until, &last)
	if err == sql.ErrNoRows {
		return domain.OutboxMessage{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.OutboxMessage{}, err
	}
	lease := now.Add(2 * time.Minute)
	result, err := tx.ExecContext(ctx, `UPDATE outbox_messages SET lease_owner=?, lease_until=?, attempts=attempts+1 WHERE id=? AND status='pending' AND (lease_until IS NULL OR lease_until<?)`, worker, storage.StringTime(lease), item.ID, storage.StringTime(now))
	if err != nil {
		return domain.OutboxMessage{}, err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return domain.OutboxMessage{}, domain.ErrConflict
	}
	item.Attempts++
	item.LeaseOwner, item.LeaseUntil = worker, lease
	if err := tx.Commit(); err != nil {
		return domain.OutboxMessage{}, err
	}
	return item, nil
}

func (s *Service) Publish(ctx context.Context, item domain.OutboxMessage, worker string, now time.Time) error {
	if s.Store.Hooks.PublishStarted != nil {
		close(s.Store.Hooks.PublishStarted)
	}
	if s.Store.Hooks.PublishRelease != nil {
		select {
		case <-s.Store.Hooks.PublishRelease:
		case <-ctx.Done():
			return fmt.Errorf("%w: publish cancelled", domain.ErrCancelled)
		}
	}
	if s.Store.Hooks.FailBroker {
		return s.Fail(ctx, item, worker, now, domain.ErrUnavailable)
	}
	result, err := s.Store.DB.ExecContext(ctx, `UPDATE outbox_messages SET status='sent', lease_owner=NULL, lease_until=NULL WHERE id=? AND lease_owner=?`, item.ID, worker)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return domain.ErrLeaseLost
	}
	return nil
}

func (s *Service) Fail(ctx context.Context, item domain.OutboxMessage, worker string, now time.Time, cause error) error {
	next := now.Add(time.Duration(item.Attempts+1) * time.Second)
	_, err := s.Store.DB.ExecContext(ctx, `UPDATE outbox_messages SET status=CASE WHEN attempts>=5 THEN 'dead' ELSE 'pending' END, next_attempt=?, lease_owner=NULL, lease_until=NULL, last_error=? WHERE id=? AND lease_owner=?`, storage.StringTime(next), cause.Error(), item.ID, worker)
	if err != nil {
		return err
	}
	return cause
}

func (s *Service) Count(ctx context.Context, status string) (int, error) {
	var count int
	err := s.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_messages WHERE status=?`, status).Scan(&count)
	return count, err
}
