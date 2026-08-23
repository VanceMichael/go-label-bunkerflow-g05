package worker

import (
	"context"
	"database/sql"
)

func replayFromBeginning(ctx context.Context, tx *sql.Tx, orderID string) error {
	var lotID string
	var target float64
	if err := tx.QueryRowContext(ctx, `SELECT fuel_lot_id,target_kg FROM transfer_orders WHERE id=?`, orderID).Scan(&lotID, &target); err != nil {
		return err
	}
	var confirmedTransfer int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM transfer_steps WHERE order_id=? AND name='transfer' AND status='completed'`, orderID).Scan(&confirmedTransfer); err != nil {
		return err
	}
	if confirmedTransfer > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE fuel_lots SET available_kg=available_kg-? WHERE id=?`, target, lotID); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `UPDATE transfer_steps SET status='pending',confirmed_at=NULL WHERE order_id=?`, orderID)
	return err
}
