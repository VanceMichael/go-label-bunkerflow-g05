package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var embeddedMigrations embed.FS

type Hooks struct {
	FailAudit      bool
	FailOutbox     bool
	FailBroker     bool
	PublishStarted chan struct{}
	PublishRelease <-chan struct{}
}

type Store struct {
	DB    *sql.DB
	Hooks Hooks
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{DB: db}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	contents, err := embeddedMigrations.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read embedded migration: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, string(contents)); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrCancelled, err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ctx.Err(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("%w: %v", domain.ErrCancelled, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func ExecContext(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, query string, args ...any) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrCancelled, err)
	}
	_, err := q.ExecContext(ctx, query, args...)
	return err
}

func QueryRowContext(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, query string, args ...any) *sql.Row {
	return q.QueryRowContext(ctx, query, args...)
}

func StringTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func ParseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", value, err)
	}
	return parsed, nil
}

func IsMissing(err error) bool { return errors.Is(err, sql.ErrNoRows) }
