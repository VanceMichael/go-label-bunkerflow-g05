package worker

import (
	"context"
	"database/sql"
)

// resumeFromCheckpoint restores a cancelled transfer so that execution
// continues from the first unconfirmed step. Steps that were already
// confirmed (status='completed') are preserved together with their
// confirmed_at checkpoint; only the unconfirmed steps are reset to
// 'pending'.
//
// Fuel is intentionally not adjusted here. Reservation is owned by the
// transfer-start flow, which re-reserves when the order re-enters the
// transferring state. Deducting during recovery would double-count against
// that later reservation, so the available balance is left untouched.
func resumeFromCheckpoint(ctx context.Context, tx *sql.Tx, orderID string) error {
	_, err := tx.ExecContext(ctx, `UPDATE transfer_steps SET status='pending' WHERE order_id=? AND status!='completed'`, orderID)
	return err
}
