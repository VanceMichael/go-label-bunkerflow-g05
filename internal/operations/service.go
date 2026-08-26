package operations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/storage"
)

type Service struct{ Store *storage.Store }

func New(store *storage.Store) *Service { return &Service{Store: store} }
func (s *Service) BatchComplete(ctx context.Context, actor domain.Actor, orderIDs []string, requestID string) error {
	if len(orderIDs) == 0 {
		return domain.ErrInvalid
	}
	return s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		// Validate every precondition before mutating any order. Doing both
		// inside one transaction makes the batch atomic: if any order is not
		// yet sampled the whole transaction rolls back and no order is
		// completed, so a single failing precondition cannot leave the rest
		// of the batch partially committed.
		for _, id := range orderIDs {
			var state string
			if err := tx.QueryRowContext(ctx, `SELECT state FROM transfer_orders WHERE id=? AND tenant_id=?`, id, actor.TenantID).Scan(&state); err == sql.ErrNoRows {
				return domain.ErrNotFound
			} else if err != nil {
				return err
			}
			if state != string(domain.StateSampled) {
				return fmt.Errorf("%w: order %s is %s", domain.ErrNoQuality, id, state)
			}
		}
		for _, id := range orderIDs {
			result, err := tx.ExecContext(ctx, `UPDATE transfer_orders SET state='completed', version=version+1 WHERE id=? AND tenant_id=? AND state='sampled'`, id, actor.TenantID)
			if err != nil {
				return err
			}
			n, _ := result.RowsAffected()
			if n != 1 {
				return fmt.Errorf("%w: order %s no longer sampled", domain.ErrConflict, id)
			}
		}
		_ = requestID
		return nil
	})
}
