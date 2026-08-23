package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/audit"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/outbox"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	Store  *storage.Store
	Audit  *audit.Service
	Outbox *outbox.Service
}
type WindowInput struct {
	TerminalID       string
	StartsAt, EndsAt time.Time
}

func New(store *storage.Store, auditSvc *audit.Service, outboxSvc *outbox.Service) *Service {
	return &Service{Store: store, Audit: auditSvc, Outbox: outboxSvc}
}

func (s *Service) CreateWindow(ctx context.Context, actor domain.Actor, input WindowInput, requestID string) (domain.BunkerWindow, error) {
	if input.TerminalID == "" || !input.EndsAt.After(input.StartsAt) || input.StartsAt.Before(time.Now().Add(-time.Minute)) {
		return domain.BunkerWindow{}, domain.ErrInvalid
	}
	item := domain.BunkerWindow{ID: uuid.NewString(), TenantID: actor.TenantID, TerminalID: input.TerminalID, StartsAt: input.StartsAt, EndsAt: input.EndsAt, Status: "open", Version: 1}
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var terminalStatus string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM terminals WHERE id=? AND tenant_id=?`, input.TerminalID, actor.TenantID).Scan(&terminalStatus); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if terminalStatus != "active" {
			return domain.ErrConflict
		}
		var conflict int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM bunker_windows WHERE terminal_id=? AND status='open' AND starts_at<? AND ends_at>?`, input.TerminalID, storage.StringTime(input.EndsAt), storage.StringTime(input.StartsAt)).Scan(&conflict); err != nil {
			return err
		}
		if conflict != 0 {
			return fmt.Errorf("%w: overlapping window", domain.ErrConflict)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO bunker_windows(id, tenant_id, terminal_id, starts_at, ends_at, status, version, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.TenantID, item.TerminalID, storage.StringTime(item.StartsAt), storage.StringTime(item.EndsAt), item.Status, item.Version, storage.StringTime(time.Now())); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "window.created", item.ID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "window.created", item.ID)
	})
	if err != nil {
		return domain.BunkerWindow{}, err
	}
	return item, nil
}

func (s *Service) ClaimWindow(ctx context.Context, actor domain.Actor, windowID string, owner string, requestID string) error {
	if s.Store.Hooks.FailAudit {
		if _, err := s.Store.DB.ExecContext(ctx, `UPDATE bunker_windows SET status='claimed', owner_id=?, version=version+1 WHERE id=? AND tenant_id=? AND status='open'`, owner, windowID, actor.TenantID); err != nil {
			return err
		}
		return s.Audit.Record(ctx, s.Store.DB, actor, "window.claimed", windowID, requestID)
	}
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE bunker_windows SET status='claimed', owner_id=?, version=version+1 WHERE id=? AND tenant_id=? AND status='open'`, owner, windowID, actor.TenantID)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return fmt.Errorf("%w: window already claimed", domain.ErrConflict)
		}
		return s.Audit.Record(ctx, tx, actor, "window.claimed", windowID, requestID)
	})
	return err
}

func (s *Service) CancelWindow(ctx context.Context, actor domain.Actor, windowID, requestID string) error {
	err := s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM bunker_windows WHERE id=? AND tenant_id=?`, windowID, actor.TenantID).Scan(&status); err == sql.ErrNoRows {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}
		if status == "released" || status == "cancelled" {
			return nil
		}
		var active int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM transfer_orders WHERE window_id=? AND state NOT IN ('completed','cancelled')`, windowID).Scan(&active); err != nil {
			return err
		}
		if active > 0 {
			return fmt.Errorf("%w: active transfer", domain.ErrConflict)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE bunker_windows SET status='cancelled', owner_id=NULL, version=version+1 WHERE id=? AND tenant_id=?`, windowID, actor.TenantID); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, actor, "window.cancelled", windowID, requestID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, actor.TenantID, "window.cancelled", windowID)
	})
	return err
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.BunkerWindow, error) {
	var item domain.BunkerWindow
	var start, end string
	err := s.Store.DB.QueryRowContext(ctx, `SELECT id, tenant_id, terminal_id, starts_at, ends_at, status, owner_id, version FROM bunker_windows WHERE id=? AND tenant_id=?`, id, actor.TenantID).Scan(&item.ID, &item.TenantID, &item.TerminalID, &start, &end, &item.Status, &item.OwnerID, &item.Version)
	if err == sql.ErrNoRows {
		return item, domain.ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.StartsAt, err = storage.ParseTime(start)
	if err != nil {
		return item, err
	}
	item.EndsAt, err = storage.ParseTime(end)
	return item, err
}

func (s *Service) List(ctx context.Context, actor domain.Actor, status string) ([]domain.BunkerWindow, error) {
	query := `SELECT id, tenant_id, terminal_id, starts_at, ends_at, status, owner_id, version FROM bunker_windows WHERE tenant_id=?`
	args := []any{actor.TenantID}
	if status != "" {
		query += ` AND status=?`
		args = append(args, status)
	}
	query += ` ORDER BY starts_at, id`
	rows, err := s.Store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.BunkerWindow
	for rows.Next() {
		var item domain.BunkerWindow
		var start, end string
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TerminalID, &start, &end, &item.Status, &item.OwnerID, &item.Version); err != nil {
			return nil, err
		}
		item.StartsAt, err = storage.ParseTime(start)
		if err != nil {
			return nil, err
		}
		item.EndsAt, err = storage.ParseTime(end)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
