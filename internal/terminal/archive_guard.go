package terminal

import (
	"context"
	"database/sql"
)

type ArchiveGuard struct {
	DB *sql.DB
}

// BlockingResources counts the runtime resources that must drain before a
// terminal can be archived: active bunker windows, in-flight transfer leases
// still held against the terminal's windows, and outbox messages pending
// publication for the terminal's windows or transfer orders.
func (g ArchiveGuard) BlockingResources(ctx context.Context, tenantID, terminalID string) (int, error) {
	var activeWindows int
	err := g.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bunker_windows WHERE tenant_id=? AND terminal_id=? AND status NOT IN ('cancelled','released')`, tenantID, terminalID).Scan(&activeWindows)
	if err != nil {
		return 0, err
	}

	var inFlightLeases int
	err = g.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM transfer_orders o JOIN bunker_windows w ON w.id=o.window_id WHERE w.tenant_id=? AND w.terminal_id=? AND o.lease_until IS NOT NULL`, tenantID, terminalID).Scan(&inFlightLeases)
	if err != nil {
		return 0, err
	}

	var pendingOutbox int
	err = g.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_messages WHERE tenant_id=? AND status='pending' AND (payload IN (SELECT id FROM bunker_windows WHERE terminal_id=?) OR payload IN (SELECT o.id FROM transfer_orders o JOIN bunker_windows w ON w.id=o.window_id WHERE w.terminal_id=?))`, tenantID, terminalID, terminalID).Scan(&pendingOutbox)
	if err != nil {
		return 0, err
	}

	return activeWindows + inFlightLeases + pendingOutbox, nil
}
