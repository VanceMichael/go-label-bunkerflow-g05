package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type Health struct{ DB *sql.DB }

func (h Health) Ping(ctx context.Context) error {
	if h.DB == nil {
		return fmt.Errorf("database is nil")
	}
	if err := h.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	return nil
}
func (h Health) Tables(ctx context.Context) ([]string, error) {
	rows, err := h.DB.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
func (h Health) HasTable(ctx context.Context, name string) (bool, error) {
	var count int
	err := h.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&count)
	return count == 1, err
}
