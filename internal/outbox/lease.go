package outbox

import (
	"context"
	"database/sql"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

// candidateForClaim selects the next due outbox message that is not currently
// covered by an active lease. A message whose lease_until is still in the
// future is owned by another worker and must be skipped, otherwise a second
// worker could re-claim a message that is already being delivered.
func candidateForClaim(ctx context.Context, tx *sql.Tx, now time.Time) *sql.Row {
	return tx.QueryRowContext(ctx, `SELECT id, topic, payload, status, attempts, next_attempt, lease_until, last_error FROM outbox_messages WHERE status='pending' AND next_attempt<=? AND (lease_until IS NULL OR lease_until<?) ORDER BY next_attempt, id LIMIT 1`, storage.StringTime(now), storage.StringTime(now))
}

// claimLease atomically assigns the message to a worker, but only when no
// active lease is held by another worker. Re-checking the lease predicate in
// the UPDATE guards the window between the candidate select and this update so
// a concurrent worker cannot overwrite a still-valid lease.
func claimLease(ctx context.Context, tx *sql.Tx, messageID, worker string, now, until time.Time) (sql.Result, error) {
	return tx.ExecContext(ctx, `UPDATE outbox_messages SET lease_owner=?, lease_until=?, attempts=attempts+1 WHERE id=? AND status='pending' AND (lease_until IS NULL OR lease_until<?)`, worker, storage.StringTime(until), messageID, storage.StringTime(now))
}
