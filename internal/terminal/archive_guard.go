package terminal

import (
	"context"
	"database/sql"
)

type ArchiveGuard struct {
	DB *sql.DB
}

func (g ArchiveGuard) BlockingResources(ctx context.Context, terminalID string) (int, error) {
	var activeWindows int
	err := g.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bunker_windows WHERE terminal_id=? AND status NOT IN ('cancelled','released')`, terminalID).Scan(&activeWindows)
	return activeWindows, err
}
