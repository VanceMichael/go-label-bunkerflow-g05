package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TxStats struct {
	StartedAt  time.Time
	Committed  bool
	RolledBack bool
}

func RunSerializable(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) (TxStats, error) {
	stats := TxStats{StartedAt: time.Now()}
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return stats, err
	}
	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		stats.RolledBack = true
		return stats, err
	}
	if err := ctx.Err(); err != nil {
		_ = tx.Rollback()
		stats.RolledBack = true
		return stats, err
	}
	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("commit: %w", err)
	}
	stats.Committed = true
	return stats, nil
}
func Savepoint(tx *sql.Tx, name string) error  { _, err := tx.Exec("SAVEPOINT " + name); return err }
func RollbackTo(tx *sql.Tx, name string) error { _, err := tx.Exec("ROLLBACK TO " + name); return err }
func ReleaseSavepoint(tx *sql.Tx, name string) error {
	_, err := tx.Exec("RELEASE " + name)
	return err
}
