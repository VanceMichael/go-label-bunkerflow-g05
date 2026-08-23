package worker

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

type Recovery struct{ Store *storage.Store }

func NewRecovery(store *storage.Store) *Recovery { return &Recovery{Store: store} }
func (r *Recovery) Replay(ctx context.Context, actor domain.Actor, orderID string) error {
	return r.Store.WithTx(ctx, func(tx *sql.Tx) error {
		var state string
		if err := tx.QueryRowContext(ctx, `SELECT state FROM transfer_orders WHERE id=? AND tenant_id=?`, orderID, actor.TenantID).Scan(&state); err != nil {
			return err
		}
		if state != "cancelled" {
			return fmt.Errorf("%w: only cancelled operations can be recovered", domain.ErrConflict)
		}
		if err := replayFromBeginning(ctx, tx, orderID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='planned',version=version+1 WHERE id=? AND tenant_id=?`, orderID, actor.TenantID); err != nil {
			return err
		}
		return nil
	})
}
