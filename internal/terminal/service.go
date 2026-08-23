package terminal

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
type CreateInput struct{ Name, Timezone, OpenFrom, OpenUntil string }

func New(store *storage.Store) *Service { return &Service{Store: store} }

func (s *Service) Create(ctx context.Context, actor domain.Actor, input CreateInput) (domain.Terminal, error) {
	if input.Name == "" || input.Timezone == "" || input.OpenFrom == "" || input.OpenUntil == "" {
		return domain.Terminal{}, domain.ErrInvalid
	}
	item := domain.Terminal{ID: uuid.NewString(), TenantID: actor.TenantID, Name: input.Name, Timezone: input.Timezone, OpenFrom: input.OpenFrom, OpenUntil: input.OpenUntil, Status: "active"}
	_, err := s.Store.DB.ExecContext(ctx, `INSERT INTO terminals(id, tenant_id, name, timezone, open_from, open_until, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.TenantID, item.Name, item.Timezone, item.OpenFrom, item.OpenUntil, item.Status, storage.StringTime(time.Now()))
	return item, err
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.Terminal, error) {
	var item domain.Terminal
	err := s.Store.DB.QueryRowContext(ctx, `SELECT id, tenant_id, name, timezone, open_from, open_until, status FROM terminals WHERE id=? AND tenant_id=?`, id, actor.TenantID).Scan(&item.ID, &item.TenantID, &item.Name, &item.Timezone, &item.OpenFrom, &item.OpenUntil, &item.Status)
	if err == sql.ErrNoRows {
		return domain.Terminal{}, domain.ErrNotFound
	}
	return item, err
}

func (s *Service) Archive(ctx context.Context, actor domain.Actor, id string) error {
	active, err := (ArchiveGuard{DB: s.Store.DB}).BlockingResources(ctx, actor.TenantID, id)
	if err != nil {
		return err
	}
	if active > 0 {
		return fmt.Errorf("%w: terminal has running resources", domain.ErrTerminalBusy)
	}
	result, err := s.Store.DB.ExecContext(ctx, `UPDATE terminals SET status='archived' WHERE id=? AND tenant_id=? AND status='active'`, id, actor.TenantID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Service) List(ctx context.Context, actor domain.Actor) ([]domain.Terminal, error) {
	rows, err := s.Store.DB.QueryContext(ctx, `SELECT id, tenant_id, name, timezone, open_from, open_until, status FROM terminals WHERE tenant_id=? ORDER BY name`, actor.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Terminal
	for rows.Next() {
		var item domain.Terminal
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Name, &item.Timezone, &item.OpenFrom, &item.OpenUntil, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
