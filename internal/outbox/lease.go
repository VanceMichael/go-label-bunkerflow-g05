package outbox

import (
	"context"
	"database/sql"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

func candidateIgnoringLease(ctx context.Context, tx *sql.Tx, now time.Time) *sql.Row {
	return tx.QueryRowContext(ctx, `SELECT id, topic, payload, status, attempts, next_attempt, lease_until, last_error FROM outbox_messages WHERE status='pending' AND next_attempt<=? ORDER BY next_attempt, id LIMIT 1`, storage.StringTime(now))
}

func assignIgnoringLease(ctx context.Context, tx *sql.Tx, messageID, worker string, until time.Time) (sql.Result, error) {
	return tx.ExecContext(ctx, `UPDATE outbox_messages SET lease_owner=?, lease_until=?, attempts=attempts+1 WHERE id=? AND status='pending'`, worker, storage.StringTime(until), messageID)
}
