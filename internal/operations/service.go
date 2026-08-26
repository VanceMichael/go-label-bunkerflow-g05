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
	return s.Store.WithCommittedPrelude(ctx, func(ctx context.Context, db *sql.DB) error {
		for _, id := range orderIDs {
			if _, err := db.ExecContext(ctx, `UPDATE transfer_orders SET state='completed',version=version+1 WHERE id=? AND tenant_id=? AND state='sampled'`, id, actor.TenantID); err != nil {
				return err
			}
		}
		return nil
	}, func(tx *sql.Tx) error {
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
		_ = requestID
		return nil
	})
}
